// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package native

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/model"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

// listenerState is the Go-side state of an http:Listener object, stored on the
// object's "$state" field. The HTTP server is created lazily in Listener.start
// (driven by the module's $start lifecycle hook); attach only registers
// services. The program stays alive while the runtime is in its listening
// state — the runtime lifecycle owns signal handling and shutdown.
type listenerState struct {
	host           string
	port           int
	timeout        time.Duration
	httpVersion    string
	tlsCfg         *pal.ServerTLSConfig
	mu             sync.RWMutex
	services       []*serviceEntry
	server         pal.ServerHandle
	servingStrands map[uint64]struct{}
	shutdownOnce   sync.Once
	shutdownDone   chan struct{}
	shutdownErr    error // set once before shutdownDone is closed; safe to read after
}

type serviceEntry struct {
	basePath string
	svcObj   *values.Object
}

// registerListenerExterns registers the Listener class definition and its
// extern methods. Called from initHttpModule.
func registerListenerExterns(rt *runtime.Runtime) {
	listenerClassDef := &bir.BIRClassDef{
		Name:      model.Name("Listener"),
		LookupKey: "ballerina/http:Listener",
		Fields:    []bir.ObjectField{},
		VTable: map[string]*bir.BIRFunction{
			"initNative":    {FunctionLookupKey: "ballerina/http:Listener.initNative"},
			"attach":        {FunctionLookupKey: "ballerina/http:Listener.attach"},
			"detach":        {FunctionLookupKey: "ballerina/http:Listener.detach"},
			"start":         {FunctionLookupKey: "ballerina/http:Listener.start"},
			"gracefulStop":  {FunctionLookupKey: "ballerina/http:Listener.gracefulStop"},
			"immediateStop": {FunctionLookupKey: "ballerina/http:Listener.immediateStop"},
		},
	}
	runtime.RegisterExternClassDef(rt, listenerClassDef)

	// Default lambdas for the optional config/name parameters (both default to ()).
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Listener.init$default$1",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return nil, nil })
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Listener.attach$default$1",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return nil, nil })

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.initNative",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			port := int(args[1].(int64))
			state := &listenerState{
				host:           "0.0.0.0",
				port:           port,
				timeout:        60 * time.Second,
				httpVersion:    "2.0",
				servingStrands: make(map[uint64]struct{}),
				shutdownDone:   make(chan struct{}),
			}
			if len(args) > 2 {
				if cfg, ok := args[2].(*values.Map); ok {
					if v, ok := cfg.Get("host"); ok {
						if s, ok := v.(string); ok && s != "" {
							state.host = s
						}
					}
					if v, ok := cfg.Get("timeout"); ok {
						if d, ok := v.(*decimal.Decimal); ok {
							state.timeout = decimalToDuration(d)
						}
					}
					if v, ok := cfg.Get("httpVersion"); ok {
						if s, ok := v.(string); ok && s != "" {
							state.httpVersion = s
						}
					}
					if state.httpVersion == "1.0" {
						msg := "warning [ballerina/http]: HTTP/1.0 is not supported by the Go HTTP runtime; falling back to HTTP/1.1\n"
						_, _ = rt.Platform().IO.Stderr([]byte(msg))
						state.httpVersion = "1.1"
					}
					if v, ok := cfg.Get("secureSocket"); ok {
						if ssMap, ok := v.(*values.Map); ok {
							tlsCfg, err := buildListenerTLSConfig(ssMap, rt.Platform().FS)
							if err != nil {
								return values.NewErrorWithMessage("Listener.init secureSocket: " + err.Error()), nil
							}
							state.tlsCfg = tlsCfg
						}
					}
				}
			}
			self.Put("$state", state)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.attach",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			svcObj, ok := args[1].(*values.Object)
			if !ok {
				return values.NewErrorWithMessage("Listener.attach: expected service object"), nil
			}
			var attachArg values.BalValue
			if len(args) > 2 {
				attachArg = args[2]
			}
			basePath := extractAttachPath(attachArg)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)

			if msg := validateServiceForHTTP(svcObj); msg != "" {
				return values.NewErrorWithMessage("Listener.attach: " + msg), nil
			}

			state.mu.Lock()
			defer state.mu.Unlock()
			// Two services cannot share a base path: service-level dispatch could
			// not pick between them deterministically.
			for _, e := range state.services {
				if e.basePath == basePath {
					return values.NewErrorWithMessage("Listener.attach: a service is already attached to base path " + basePath), nil
				}
			}
			entry := &serviceEntry{basePath: basePath, svcObj: svcObj}
			state.services = append(state.services, entry)
			// Longest base path first so the most specific service wins routing.
			sort.Slice(state.services, func(i, j int) bool {
				return len(state.services[i].basePath) > len(state.services[j].basePath)
			})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.detach",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			svcObj := args[1].(*values.Object)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)
			state.mu.Lock()
			defer state.mu.Unlock()
			for i, e := range state.services {
				if e.svcObj == svcObj {
					state.services = append(state.services[:i], state.services[i+1:]...)
					break
				}
			}
			return nil, nil
		})

	// Listener.start creates and starts the HTTP server. It is invoked by the
	// module's $start lifecycle hook after all services have been attached.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.start",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)
			// Held for the whole operation (not just the check) so a concurrent
			// start cannot race past this check before the first sets state.server,
			// and gracefulStop/immediateStop (which take state.mu.RLock) cannot
			// observe a nil server while a start is still in flight.
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.server != nil {
				return nil, nil
			}
			server, err := startHTTPServer(rt, state)
			if err != nil {
				return values.NewErrorWithMessage("Listener.start: " + err.Error()), nil
			}
			state.server = server
			return nil, nil
		})

	// Listener.gracefulStop drains in-flight requests before closing the server.
	//
	// This extern has two callers that need opposite blocking behaviour:
	//   - A resource function invoking ep.gracefulStop() runs inline on the
	//     handler's own goroutine (same strand that is currently serving the
	//     HTTP request). Blocking here on server.Shutdown would deadlock: the
	//     connection can't go idle until the handler returns, but the handler
	//     can't return until Shutdown does.
	//   - The runtime's signal-triggered graceful stop (SIGINT/SIGTERM) runs on
	//     a separate strand that never served a request. It must block until
	//     the drain completes, because the process exits right after this call
	//     returns.
	//
	// We tell the two apart via the calling strand: dispatchRequest registers
	// its strand ID in state.servingStrands for the duration of the resource
	// invocation, so a hit on that set means we're on the handler path.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.gracefulStop",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)
			state.mu.RLock()
			server := state.server
			timeout := state.timeout
			_, isServingStrand := state.servingStrands[ctx.StrandID]
			state.mu.RUnlock()
			if server == nil {
				return nil, nil
			}

			state.shutdownOnce.Do(func() {
				go func() {
					shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()
					state.shutdownErr = server.Shutdown(shutdownCtx)
					close(state.shutdownDone)
				}()
			})

			if isServingStrand {
				select {
				case <-state.shutdownDone:
					if state.shutdownErr != nil {
						return values.NewErrorWithMessage("Listener.gracefulStop: " + state.shutdownErr.Error()), nil
					}
				default:
				}
				return nil, nil
			}
			<-state.shutdownDone
			if state.shutdownErr != nil {
				return values.NewErrorWithMessage("Listener.gracefulStop: " + state.shutdownErr.Error()), nil
			}
			return nil, nil
		})

	// Listener.immediateStop closes the server and all active connections at once.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.immediateStop",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)
			state.mu.RLock()
			server := state.server
			state.mu.RUnlock()
			if server != nil {
				_ = server.Close()
			}
			return nil, nil
		})
}

// extractAttachPath converts the Ballerina attach-point value to a base path string.
// () → "/", "foo" → "/foo", ["a","b"] → "/a/b"
func extractAttachPath(v values.BalValue) string {
	if v == nil {
		return "/"
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return "/"
		}
		if !strings.HasPrefix(val, "/") {
			val = "/" + val
		}
		if val != "/" {
			val = strings.TrimSuffix(val, "/")
		}
		return val
	case *values.List:
		parts := make([]string, val.Len())
		for i := range val.Len() {
			if s, ok := val.Get(i).(string); ok {
				parts[i] = s
			}
		}
		return "/" + strings.Join(parts, "/")
	}
	return "/"
}

// buildListenerTLSConfig builds a pal.ServerTLSConfig from a ListenerSecureSocket
// map, reading all PEM material via fs. The concrete *tls.Config is assembled by
// the platform (palnative), keeping this code free of crypto/tls so it stays
// portable to the WASM target.
func buildListenerTLSConfig(ssMap *values.Map, fs pal.FS) (*pal.ServerTLSConfig, error) {
	keyVal, ok := ssMap.Get("key")
	if !ok {
		return nil, fmt.Errorf("secureSocket.key is required")
	}
	keyMap, ok := keyVal.(*values.Map)
	if !ok {
		return nil, fmt.Errorf("secureSocket.key must be a CertKey record")
	}

	certFileVal, _ := keyMap.Get("certFile")
	keyFileVal, _ := keyMap.Get("keyFile")
	certFilePath, _ := certFileVal.(string)
	keyFilePath, _ := keyFileVal.(string)

	certPEM, err := fs.ReadFile(certFilePath)
	if err != nil {
		return nil, fmt.Errorf("key.certFile: %w", err)
	}
	keyPEM, err := fs.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("key.keyFile: %w", err)
	}

	cfg := &pal.ServerTLSConfig{CertPEM: certPEM, KeyPEM: keyPEM}

	// mTLS: client certificate verification.
	if v, ok := ssMap.Get("mutualSsl"); ok {
		if b, ok := v.(bool); ok && b {
			caCertPathVal, ok := ssMap.Get("cert")
			if !ok {
				return nil, fmt.Errorf("secureSocket.cert is required when mutualSsl is enabled")
			}
			caCertPath, ok := caCertPathVal.(string)
			if !ok || caCertPath == "" {
				return nil, fmt.Errorf("secureSocket.cert is required when mutualSsl is enabled")
			}
			caCertPEM, err := fs.ReadFile(caCertPath)
			if err != nil {
				return nil, fmt.Errorf("secureSocket.cert (CA): %w", err)
			}
			cfg.ClientCACertPEM = caCertPEM
		}
	}

	// TLS version bounds (raw IANA version codes; platform applies them).
	if v, ok := ssMap.Get("protocol"); ok {
		if list, ok := v.(*values.List); ok {
			tlsVersionMap := map[string]uint16{
				"TLSv1.0": 0x0301, "TLSv1.1": 0x0302,
				"TLSv1.2": 0x0303, "TLSv1.3": 0x0304,
			}
			for i := range list.Len() {
				if s, ok := list.Get(i).(string); ok {
					if ver, found := tlsVersionMap[s]; found {
						if cfg.MinVersion == 0 || ver < cfg.MinVersion {
							cfg.MinVersion = ver
						}
						if ver > cfg.MaxVersion {
							cfg.MaxVersion = ver
						}
					}
				}
			}
		}
	}

	// Cipher suites — carried as IANA names; the platform resolves them to IDs.
	if v, ok := ssMap.Get("ciphers"); ok {
		if list, ok := v.(*values.List); ok {
			for i := range list.Len() {
				if s, ok := list.Get(i).(string); ok {
					cfg.CipherSuiteNames = append(cfg.CipherSuiteNames, s)
				}
			}
		}
	}

	// Session tickets.
	if v, ok := ssMap.Get("shareSession"); ok {
		if b, ok := v.(bool); ok && !b {
			cfg.DisableSessionTickets = true
		}
	}

	return cfg, nil
}

// validateServiceForHTTP rejects service objects that contain remote methods,
// which are not supported for HTTP dispatch. Normal and resource methods are
// allowed. Returns a non-empty error message if validation fails.
func validateServiceForHTTP(svcObj *values.Object) string {
	if svcObj.HasRemoteMethods() {
		return "service object must not have remote methods"
	}
	return ""
}

// startHTTPServer builds the platform-neutral dispatch handler and hands it to
// the platform's HTTP.Listen, which owns the transport: the native platform
// binds a TCP socket (optionally TLS-wrapped) and serves on a background
// goroutine, while a WASM/web platform registers the handler with its JS host.
// All request routing and dispatch stays here in the shared handler.
func startHTTPServer(rt *runtime.Runtime, state *listenerState) (pal.ServerHandle, error) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				msg := panicMessage(rec)
				logMsg := fmt.Sprintf("error [ballerina/http]: panic while handling %s %s: %s\n", r.Method, r.URL.Path, msg)
				_, _ = rt.Platform().IO.Stderr([]byte(logMsg))
				writeErrorJSON(rt, w, r, http.StatusInternalServerError, msg)
			}
		}()
		dispatchRequest(rt, state, w, r)
	})
	cfg := pal.ServerConfig{
		Host:         state.host,
		Port:         state.port,
		HTTPVersion:  state.httpVersion,
		WriteTimeout: state.timeout,
		TLS:          state.tlsCfg,
	}
	return rt.Platform().HTTP.Listen(cfg, handler)
}

// panicMessage extracts a human-readable message from a recovered panic value,
// unwrapping Ballerina error values so a resource method's unrecovered runtime panic
// (e.g. a missing method or an out-of-range access) surfaces the same message a
// returned `error` would have (see the err.Error() branch in dispatchRequest), instead
// of a generic, undiagnosable "internal server error".
func panicMessage(r any) string {
	switch v := r.(type) {
	case *values.Error:
		return v.Message
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r)
	}
}

// dispatchRequest routes an incoming HTTP request to the matching service and
// resource method.
func dispatchRequest(rt *runtime.Runtime, state *listenerState, w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	var found *serviceEntry
	var subPath string
	for _, e := range state.services {
		if rest, ok := matchBasePath(r.URL.Path, e.basePath); ok {
			found = e
			subPath = rest
			break
		}
	}
	state.mu.RUnlock()

	if found == nil {
		writeErrorJSON(rt, w, r, http.StatusNotFound, "no matching resource found for path")
		return
	}

	segments := splitURLPath(subPath)
	ctx := rt.NewExternContext()
	state.mu.Lock()
	state.servingStrands[ctx.StrandID] = struct{}{}
	state.mu.Unlock()
	defer func() {
		state.mu.Lock()
		delete(state.servingStrands, ctx.StrandID)
		state.mu.Unlock()
	}()
	httpMethod := strings.ToLower(r.Method)

	// Resource-level dispatch is delegated to the language runtime: it coerces
	// the raw path segments to each resource's declared parameter types and
	// selects the unique matching resource (the same dispatch used by
	// client->/path access). HTTP only owns service-level (base-path) routing.
	for _, accessorKey := range []string{httpMethod, "default"} {
		handle, extraArgs, ok := ctx.LookupResourceMethodByPath(found.svcObj, accessorKey, segments)
		if !ok {
			continue
		}
		var invocationArgs []values.BalValue
		switch {
		case extraArgs == 1:
			// The resource declares a single parameter beyond its path params; inject the request.
			req, err := buildRequestFromHTTP(ctx.TypeCtx, r)
			if err != nil {
				writeErrorJSON(rt, w, r, http.StatusBadRequest, "failed to read request body: "+err.Error())
				return
			}
			invocationArgs = []values.BalValue{req}
		case extraArgs > 1:
			// More than one non-path parameter is not a supported resource signature.
			writeErrorJSON(rt, w, r, http.StatusInternalServerError, "resource method has an unsupported parameter signature")
			return
		default:
			// Resource takes no Request parameter; discard the body.
			if r.Body != nil {
				_ = r.Body.Close()
			}
		}
		result, err := ctx.InvokeMethod(handle, invocationArgs)
		if err != nil {
			writeErrorJSON(rt, w, r, http.StatusInternalServerError, err.Error())
			return
		}
		writeResult(rt, ctx.TypeCtx, w, r, result)
		return
	}
	// The path matched a service but no resource under the requested method. If
	// the same path resolves under a different method it is a 405; otherwise 404.
	for _, accessor := range found.svcObj.AllResourceMethodNames() {
		if accessor == httpMethod || accessor == "default" {
			continue
		}
		if _, _, ok := ctx.LookupResourceMethodByPath(found.svcObj, accessor, segments); ok {
			writeErrorJSON(rt, w, r, http.StatusMethodNotAllowed, "method not allowed for path")
			return
		}
	}
	writeErrorJSON(rt, w, r, http.StatusNotFound, "no matching resource found for path")
}

// matchBasePath reports whether path is under the attach point basePath,
// returning the remaining sub-path. A match requires a path-segment boundary:
// basePath "/foo" matches "/foo" and "/foo/bar" but not "/foobar".
func matchBasePath(path, basePath string) (string, bool) {
	if basePath == "/" {
		return path, true
	}
	if path == basePath {
		return "", true
	}
	if strings.HasPrefix(path, basePath) && path[len(basePath)] == '/' {
		return path[len(basePath):], true
	}
	return "", false
}

// splitURLPath splits a URL sub-path into segments, stripping leading/trailing slashes.
func splitURLPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// buildRequestFromHTTP builds an http:Request value from r, buffering small
// bodies eagerly and streaming large ones lazily for passthrough.
func buildRequestFromHTTP(tc semtypes.Context, r *http.Request) (*values.Object, error) {
	var bodyBuf []byte
	var bodyStream io.ReadCloser
	cl := r.ContentLength
	switch {
	case r.Body == nil || cl == 0:
		// no body or explicitly empty
	case cl >= 0 && cl <= eagerBufferThreshold:
		data, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			return nil, err
		}
		bodyBuf = data
		cl = int64(len(data))
	default:
		bodyStream = r.Body
	}
	return buildRequest(tc, r.Method, r.RequestURI, r.Proto, r.Header, bodyStream, cl, r.URL.RawQuery, bodyBuf), nil
}

// buildRequest constructs a Ballerina Request object from HTTP request data.
// bodyStream is the raw request body; it is stored lazily in a requestBodyHolder
// so the body is only read from the network when a getPayload method is called.
// bodyBuf, when non-nil, is an already-read body; bodyStream must be nil in that case.
func buildRequest(tc semtypes.Context, method, rawPath, httpVersion string, headers map[string][]string, bodyStream io.ReadCloser, contentLength int64, rawQuery string, bodyBuf []byte) *values.Object {
	headersMap := newMappingValue(tc)
	for k, vals := range headers {
		items := make([]values.BalValue, len(vals))
		for i, v := range vals {
			items[i] = v
		}
		headersMap.Put(tc, strings.ToLower(k), newListValue(tc, items))
	}
	var holder *requestBodyHolder
	switch {
	case bodyBuf != nil:
		holder = &requestBodyHolder{buf: bodyBuf, contentLength: int64(len(bodyBuf))}
	case bodyStream != nil:
		holder = &requestBodyHolder{stream: bodyStream, contentLength: contentLength}
	default:
		holder = &requestBodyHolder{buf: []byte{}, contentLength: 0}
	}
	return values.NewObject(
		semtypes.OBJECT,
		map[string]values.BalValue{
			"rawPath":     rawPath,
			"method":      method,
			"httpVersion": httpVersion,
			"$headers":    headersMap,
			"$body":       holder,
			"$queryStr":   rawQuery,
		},
		requestMethodKeys(),
		nil,
	)
}

// writeErrorJSON writes a JSON error response in the standard Ballerina HTTP error format.
func writeErrorJSON(rt *runtime.Runtime, w http.ResponseWriter, r *http.Request, status int, message string) {
	type errorPayload struct {
		Timestamp string `json:"timestamp"`
		Status    int    `json:"status"`
		Reason    string `json:"reason"`
		Message   string `json:"message"`
		Path      string `json:"path"`
		Method    string `json:"method"`
	}
	payload := errorPayload{
		Timestamp: rt.Platform().Time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		Status:    status,
		Reason:    http.StatusText(status),
		Message:   message,
		Path:      r.URL.Path,
		Method:    r.Method,
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeResult writes a Ballerina resource method return value as an HTTP response.
func writeResult(rt *runtime.Runtime, _ semtypes.Context, w http.ResponseWriter, r *http.Request, result values.BalValue) {
	switch v := result.(type) {
	case nil:
		w.WriteHeader(http.StatusAccepted)
	case *values.Error:
		writeErrorJSON(rt, w, r, http.StatusInternalServerError, v.Message)
	case *values.Object:
		statusCodeVal, _ := v.Get("statusCode")
		statusCode := http.StatusOK
		// Go's WriteHeader panics outside [100, 999]; fall back to 200 for an
		// out-of-range value rather than crashing the handler.
		if sc, ok := statusCodeVal.(int64); ok && sc >= 100 && sc <= 999 {
			statusCode = int(sc)
		}
		bodyVal, _ := v.Get("body")
		holder, _ := bodyVal.(*responseBodyHolder)

		// Emit headers from the response object, excluding hop-by-hop headers.
		// Forwarding hop-by-hop headers (e.g. Transfer-Encoding, Connection) from a
		// backend response to the downstream client violates RFC 7230 §6.1 and can
		// cause framing errors in HTTP/1.1 keep-alive connections.
		if hdrsVal, ok := v.Get("$headers"); ok {
			if hdrs, ok := hdrsVal.(*values.Map); ok {
				for _, k := range hdrs.Keys() {
					if _, skip := hopByHopHeaders[strings.ToLower(k)]; skip {
						continue
					}
					val, _ := hdrs.Get(k)
					list, ok := val.(*values.List)
					if !ok {
						continue
					}
					for i := range list.Len() {
						s, _ := list.Get(i).(string)
						if i == 0 {
							w.Header().Set(k, s)
						} else {
							w.Header().Add(k, s)
						}
					}
				}
			}
		}
		// WriteHeader must be called before writing the body; once body bytes
		// start flowing via writeStream, headers are already committed.
		w.WriteHeader(statusCode)
		if holder != nil {
			// Headers are already committed; a write error here can't be recovered
			// with a JSON error body, so just stop.
			if err := holder.writeStream(w); err != nil {
				return
			}
		}
	default:
		writeErrorJSON(rt, w, r, http.StatusInternalServerError, "unexpected return type from resource method")
	}
}

// writeStream writes the body to w via io.Copy (streaming) or w.Write (buffered),
// then closes the stream. After this call the holder is exhausted.
func (h *responseBodyHolder) writeStream(w io.Writer) error {
	var (
		s   io.ReadCloser
		buf []byte
	)
	h.once.Do(func() {
		if h.stream != nil {
			s = h.stream
			h.stream = nil
			h.buf = []byte{}
		} else if len(h.buf) > 0 {
			buf = h.buf
			h.buf = []byte{}
		}
	})
	if s != nil {
		_, err := io.Copy(w, s)
		_ = s.Close()
		return err
	}
	if len(buf) > 0 {
		_, err := w.Write(buf)
		return err
	}
	// once was already fired by a prior materialize(); h.buf holds the materialized bytes.
	if len(h.buf) > 0 {
		_, err := w.Write(h.buf)
		return err
	}
	return nil
}
