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
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// listenerState is the Go-side state of an http:Listener object, stored on the
// object's "$state" field. The HTTP server is created lazily in Listener.start
// (driven by the module's $start lifecycle hook); attach only registers
// services. The program stays alive while the runtime is in its listening
// state — the runtime lifecycle owns signal handling and shutdown.
type listenerState struct {
	types            *httpTypes
	requestTy        semtypes.SemType
	headersTy        semtypes.SemType
	createHeaders    extern.FunctionHandle
	callerTy         semtypes.SemType
	createCaller     extern.FunctionHandle
	interceptorTypes interceptorTypes
	host             string
	port             int
	timeout          time.Duration
	httpVersion      string
	tlsCfg           *pal.ServerTLSConfig
	mu               sync.RWMutex
	// lifecycle serialises attach and start, which call into Ballerina without holding mu.
	lifecycle sync.Mutex
	services  []*serviceEntry
	server    pal.ServerHandle
	// starting is set while start creates interceptors; a stop meanwhile sets stopRequested, so
	// start does not then serve.
	starting      bool
	stopRequested bool
}

type serviceEntry struct {
	basePath string
	svcObj   *values.Object
	config   serviceConfig
	// interceptors are the interceptor pipeline of the service, created when the listener
	// starts, or on attach once it has started.
	interceptors []interceptor
	created      bool
}

// registerListenerExterns registers the Listener class definition and its
// extern methods. Called from initHttpModule.
func registerListenerExterns(rt *runtime.Runtime, types *httpTypes) {
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
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			port := int(args[1].(int64))
			createHeaders, signature, err := resolveSignatureTypes(ctx)
			if err != nil {
				return nil, err
			}
			createCaller, callerSignature, err := resolveFactory(ctx, "createCaller")
			if err != nil {
				return nil, err
			}
			interceptorTypes, err := resolveInterceptorTypes(ctx)
			if err != nil {
				return nil, err
			}
			state := &listenerState{
				types:            types,
				requestTy:        signature.Params[0].Type,
				headersTy:        signature.ReturnType,
				createHeaders:    createHeaders,
				callerTy:         callerSignature.ReturnType,
				createCaller:     createCaller,
				interceptorTypes: interceptorTypes,
				host:             "0.0.0.0",
				port:             port,
				timeout:          60 * time.Second,
				httpVersion:      "2.0",
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
							var pemErr *listenerPEMError
							if errors.Is(err, errKeyFilesNotFound) || errors.As(err, &pemErr) {
								return httpError(ctx, genericListenerError, err.Error(), nil), nil
							}
							if err != nil {
								return httpError(ctx, genericListenerError, "Listener.init secureSocket: "+err.Error(), nil), nil
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
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			svcObj, ok := args[1].(*values.Object)
			if !ok {
				return httpError(ctx, genericListenerError, "Listener.attach: expected service object", nil), nil
			}
			var attachArg values.BalValue
			if len(args) > 2 {
				attachArg = args[2]
			}
			basePath := extractAttachPath(attachArg)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)

			if msg := validateServiceForHTTP(svcObj); msg != "" {
				return httpError(ctx, genericListenerError, "Listener.attach: "+msg, nil), nil
			}

			config := readServiceConfig(ctx, svcObj)
			// Attach and start are serialised, so a service is attached either before the
			// listener creates the interceptors of its services or after it has started.
			state.lifecycle.Lock()
			defer state.lifecycle.Unlock()
			state.mu.RLock()
			started := state.server != nil
			// Two services on one host cannot share a base path: service-level dispatch could
			// not pick between them deterministically.
			duplicate := slices.ContainsFunc(state.services, func(e *serviceEntry) bool {
				return e.basePath == basePath && e.config.host == config.host
			})
			state.mu.RUnlock()
			if duplicate {
				message := "Listener.attach: a service is already attached to base path " + basePath
				if config.host != "" {
					message += " under host name " + config.host
				}
				return httpError(ctx, genericListenerError, message, nil), nil
			}
			// Unlike jBallerina, a service attached to a started listener gets its interceptors (#9585).
			var interceptors []interceptor
			if started {
				var err error
				if interceptors, err = createInterceptors(ctx, state.interceptorTypes, svcObj); err != nil {
					return httpError(ctx, genericListenerError, registrationError+err.Error(), nil), nil
				}
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			entry := &serviceEntry{basePath: basePath, svcObj: svcObj, config: config, interceptors: interceptors,
				created: started}
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
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)
			state.lifecycle.Lock()
			defer state.lifecycle.Unlock()
			state.mu.Lock()
			state.starting, state.stopRequested = true, false
			state.mu.Unlock()
			err := createServiceInterceptors(ctx, state)
			// Held for the rest of the operation so a stop cannot observe a nil server
			// while the server is being published.
			state.mu.Lock()
			defer state.mu.Unlock()
			stopped := state.stopRequested
			state.starting, state.stopRequested = false, false
			if err != nil {
				return httpError(ctx, genericListenerError, registrationError+err.Error(), nil), nil
			}
			if state.server != nil || stopped {
				return nil, nil
			}
			server, err := startHTTPServer(rt, state)
			if err != nil {
				return httpError(ctx, genericListenerError, "Listener.start: "+err.Error(), nil), nil
			}
			state.server = server
			return nil, nil
		})

	// Listener.gracefulStop blocks until in-flight requests drain and the server
	// closes, per the http:Listener contract, returning () on a clean drain or an
	// error if the shutdown fails (e.g. the timeout elapses).
	//
	// The block is unconditional by design: a resource that calls ep.gracefulStop()
	// on the listener serving it self-deadlocks (the connection can't go idle until
	// the handler returns, but the handler is blocked on the drain) — caller error,
	// not a case to special-case.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.gracefulStop",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)
			server, timeout := state.takeForStop()
			if server == nil {
				return nil, nil
			}

			shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				return httpError(ctx, genericListenerError, "Listener.gracefulStop: "+err.Error(), nil), nil
			}
			return nil, nil
		})

	// Listener.immediateStop closes the server and all active connections at once.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Listener.immediateStop",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			stateVal, _ := self.Get("$state")
			state := stateVal.(*listenerState)
			if server, _ := state.takeForStop(); server != nil {
				_ = server.Close()
			}
			return nil, nil
		})
}

// takeForStop returns the server a stop acts on and its timeout. Before the server is published,
// it makes a start in progress end without serving.
func (state *listenerState) takeForStop() (pal.ServerHandle, time.Duration) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.server == nil && state.starting {
		state.stopRequested = true
	}
	return state.server, state.timeout
}

// createServiceInterceptors creates the interceptors of the services attached before the
// listener starts. It calls into Ballerina, so it holds no lock while doing so.
func createServiceInterceptors(ctx *extern.Context, state *listenerState) error {
	state.mu.RLock()
	var pending []*serviceEntry
	for _, e := range state.services {
		if !e.created {
			pending = append(pending, e)
		}
	}
	state.mu.RUnlock()
	for _, e := range pending {
		interceptors, err := createInterceptors(ctx, state.interceptorTypes, e.svcObj)
		if err != nil {
			return err
		}
		state.mu.Lock()
		e.interceptors, e.created = interceptors, true
		state.mu.Unlock()
	}
	return nil
}

// resolveSignatureTypes looks up createHeaders, whose signature gives the http:Request and
// http:Headers types, since stdlib natives cannot name a Ballerina type directly.
func resolveSignatureTypes(ctx *extern.Context) (extern.FunctionHandle, extern.FunctionSignature, error) {
	fn, signature, err := resolveFactory(ctx, "createHeaders")
	if err != nil || len(signature.Params) != 1 {
		return fn, signature, fmt.Errorf("http: cannot resolve the Request and Headers types from 'createHeaders'")
	}
	return fn, signature, nil
}

// resolveFactory looks up the private factory name, whose return type is the type it builds.
func resolveFactory(ctx *extern.Context, name string) (extern.FunctionHandle, extern.FunctionSignature, error) {
	fn, found := ctx.LookupFunction(orgName, moduleName, name)
	signature, ok := ctx.FunctionSignature(fn)
	if !found || !ok {
		return fn, signature, fmt.Errorf("http: cannot resolve the type built by '%s'", name)
	}
	return fn, signature, nil
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

// errKeyFilesNotFound carries jBallerina's message for a missing server certificate or key file.
var errKeyFilesNotFound = errors.New("Key file or server certificates file not found") //nolint:staticcheck // jBallerina's message verbatim

// listenerPEMError reports a listener certificate or key file that cannot be read or holds no usable PEM data.
type listenerPEMError struct {
	what, path string
}

func (e *listenerPEMError) Error() string {
	return "File does not contain valid " + e.what + ": " + e.path
}

// hasCertificates reports whether data holds at least one certificate and every certificate block parses.
func hasCertificates(data []byte) bool {
	found := false
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		if block.Type != "CERTIFICATE" {
			continue
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return false
		}
		found = true
	}
	return found
}

// hasPrivateKey reports whether data holds a PKCS #8, PKCS #1 or EC private key.
func hasPrivateKey(data []byte) bool {
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		switch block.Type {
		case "PRIVATE KEY":
			if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
				return true
			}
		case "RSA PRIVATE KEY":
			if _, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
				return true
			}
		case "EC PRIVATE KEY":
			if _, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
				return true
			}
		}
	}
	return false
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

	certPEM, certErr := fs.ReadFile(certFilePath)
	keyPEM, keyErr := fs.ReadFile(keyFilePath)
	if errors.Is(certErr, iofs.ErrNotExist) || errors.Is(keyErr, iofs.ErrNotExist) {
		return nil, errKeyFilesNotFound
	}
	if certErr != nil || !hasCertificates(certPEM) {
		return nil, &listenerPEMError{"certificates", certFilePath}
	}
	if keyErr != nil || !hasPrivateKey(keyPEM) {
		return nil, &listenerPEMError{"private key", keyFilePath}
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
			if err != nil || !hasCertificates(caCertPEM) {
				return nil, &listenerPEMError{"certificates", caCertPath}
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
		encoder := newResponseEncoder(w, r)
		defer func() {
			encoder.finish()
			// Drops the connection, so the client sees the body as incomplete.
			if encoder.aborted {
				panic(http.ErrAbortHandler)
			}
		}()
		defer func() {
			if rec := recover(); rec != nil {
				message := logPanic(rt, r, rec)
				// A response already under way cannot become a 500.
				if !encoder.started() {
					writePanicResponse(encoder, message)
				}
			}
		}()
		dispatchRequest(rt, state, encoder, r)
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

// logPanic reports a panic raised while handling r on stderr and returns its message.
func logPanic(rt *runtime.Runtime, r *http.Request, rec any) string {
	msg := panicMessage(rec)
	logMsg := fmt.Sprintf("error [ballerina/http]: panic while handling %s %s: %s\n", r.Method, r.URL.Path, msg)
	_, _ = rt.Platform().IO.Stderr([]byte(logMsg))
	return msg
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

// dispatchRequest routes an incoming HTTP request to the matching service, through the
// interceptors of the service, and on to the matching resource method.
func dispatchRequest(rt *runtime.Runtime, state *listenerState, w *responseEncoder, r *http.Request) {
	state.mu.RLock()
	services := hostServices(state.services, r.Host)
	var found *serviceEntry
	var subPath string
	var interceptors []interceptor
	for _, e := range services {
		if rest, ok := matchBasePath(r.URL.Path, e.basePath); ok {
			found = e
			subPath = rest
			interceptors = e.interceptors
			break
		}
	}
	state.mu.RUnlock()

	if services == nil {
		writeErrorJSON(rt, w, r, http.StatusNotFound, "no service has registered for listener : "+javaSocketAddress(r))
		return
	}
	if found == nil {
		writeErrorJSON(rt, w, r, http.StatusNotFound, "no matching service found for path: "+r.URL.EscapedPath())
		return
	}

	var body []byte
	if interceptors != nil {
		// Every stage reads the body afresh, and stages after a response read a copy of the
		// request, since net/http ends the request with the handler.
		read, err := io.ReadAll(r.Body)
		closeBody(r)
		if err != nil {
			writeErrorJSON(rt, w, r, http.StatusBadRequest, "failed to read request body: "+err.Error())
			return
		}
		body, r = read, r.Clone(r.Context())
		// Unlike jBallerina, responses from the interceptors of a service follow its configuration (#9887).
		w.useService(found.config)
	}
	conn := newCallerConn(rt, state.types, w, r, found.config)
	conn.pipeline = &requestPipeline{stages: interceptors, target: found.svcObj}
	x := &exchange{state: state, w: w, r: r, service: found, segments: splitURLPath(subPath),
		method: strings.ToLower(r.Method), conn: conn, body: body}
	conn.exchange = x
	x.runFrom(1)
}

// exchange is a request on its way through the interceptors of its service to the service
// itself, and its response on the way back. Each request-side stage runs once the one before it
// has returned.
type exchange struct {
	state    *listenerState
	w        *responseEncoder
	r        *http.Request
	service  *serviceEntry
	segments []string
	method   string
	conn     *callerConn
	// body is the request body each stage reads when the service has interceptors.
	body []byte
	// err is the error the request carries to the next request error interceptor.
	err *values.Error
}

// intercepted reports whether the service of the exchange has interceptors.
func (x *exchange) intercepted() bool {
	return len(x.conn.pipeline.stages) > 0
}

// lastStage is the position the response walk of a response from the service starts from.
func (x *exchange) lastStage() int {
	return len(x.conn.pipeline.stages) - 1
}

// runFrom runs the first request interceptor from position from whose resource matches the
// request, or else the service. While the request carries an error, it runs the first such
// request error interceptor instead, or else answers with the error. An interceptor that does
// not serve the request is skipped.
func (x *exchange) runFrom(from int) {
	rt := x.conn.rt
	stages := x.conn.pipeline.stages
	kind := requestKind
	if x.err != nil {
		kind = requestErrorKind
	}
	for i := from; i < len(stages); i++ {
		if stages[i].kind != kind {
			continue
		}
		ctx := rt.AcquirePooledContext()
		for _, accessorKey := range []string{x.method, "default"} {
			handle, extraArgs, ok := ctx.LookupResourceMethodByPath(stages[i].obj, accessorKey, x.segments)
			if ok {
				x.conn.accessor = accessorKey
				x.conn.pipeline.enter(i)
				x.conn.handledError, x.err = x.err, nil
				x.rewind()
				x.runInterceptor(ctx, i, handle, extraArgs)
				return
			}
		}
		rt.ReleasePooledContext(ctx)
	}
	x.rewind()
	x.conn.handledError = nil
	if err := x.err; err != nil {
		x.err = nil
		x.deliver(nil, err, x.lastStage())
		return
	}
	x.dispatchService()
}

// rewind gives the next stage the body from its start.
func (x *exchange) rewind() {
	if x.body != nil {
		x.r.Body = io.NopCloser(bytes.NewReader(x.body))
		x.r.ContentLength = int64(len(x.body))
	}
}

// runInterceptor runs the interceptor at position index, which owns ctx, on its own goroutine when
// it has an http:Caller parameter, as a resource does.
func (x *exchange) runInterceptor(ctx *extern.Context, index int, handle extern.MethodHandle, extraArgs int) {
	args, hasCaller, req, bindErr := bindResource(ctx, x.state, x.conn, handle, extraArgs)
	if bindErr != nil {
		x.conn.pipeline.leave(index)
		x.err = bindErr.toError(ctx)
		x.conn.rt.ReleasePooledContext(ctx)
		x.runFrom(index + 1)
		return
	}
	if !hasCaller {
		x.completeInterceptor(ctx, index, handle, args, req, false)
		return
	}
	go func() {
		defer x.conn.markSent()
		defer func() {
			if rec := recover(); rec != nil {
				x.conn.failPanic(logPanic(x.conn.rt, x.r, rec))
			}
		}()
		x.completeInterceptor(ctx, index, handle, args, req, true)
	}()
	<-x.conn.sent
}

func (x *exchange) completeInterceptor(ctx *extern.Context, index int, handle extern.MethodHandle,
	args []values.BalValue, req *values.Object, hasCaller bool) {
	result, failure, failed := invokeStage(ctx, x.conn.rt, x.r, handle, args)
	from, proceed := 0, false
	if failed {
		x.conn.pipeline.leave(index)
		x.conn.failPanic(failure)
	} else {
		from, proceed = x.afterInterceptor(ctx, index, result, hasCaller)
	}
	if proceed {
		if err := x.syncRequest(ctx, req); err != nil {
			x.conn.fail(http.StatusInternalServerError, err.Error())
			proceed = false
		}
	}
	x.conn.rt.ReleasePooledContext(ctx)
	if proceed {
		x.runFrom(from)
	}
}

// invokeStage calls the resource behind handle and reports a panic as the message of the 500
// response it gets.
func invokeStage(ctx *extern.Context, rt *runtime.Runtime, r *http.Request, handle extern.MethodHandle,
	args []values.BalValue) (result values.BalValue, failure string, failed bool) {
	defer func() {
		if rec := recover(); rec != nil {
			ctx.ReleaseAllHeldLocks()
			result, failure, failed = nil, logPanic(rt, r, rec), true
		}
	}()
	result, err := ctx.InvokeMethod(handle, args)
	if err != nil {
		return nil, err.Error(), true
	}
	return result, "", false
}

// afterInterceptor acts on what the request-side interceptor at position index returned, as
// jBallerina does, and reports where the exchange continues, if it does.
func (x *exchange) afterInterceptor(ctx *extern.Context, index int, result values.BalValue,
	hasCaller bool) (int, bool) {
	pipeline := x.conn.pipeline
	from, nextCalled := pipeline.leave(index)
	// Once a response was sent, the exchange continues only if the interceptor called ctx.next().
	stopped := x.conn.responded() && !nextCalled
	resultErr, isError := result.(*values.Error)
	obj, isObject := result.(*values.Object)
	nextService := isObject && isOfType(ctx.TypeCtx(), obj, x.state.interceptorTypes.nextService)
	switch {
	case isError && stopped:
		x.conn.rejectReturn(result)
		return 0, false
	case isError:
		x.err = resultErr
		return index + 1, true
	case nextService && stopped:
		return 0, false
	case nextService && obj == pipeline.stageAt(from):
		return from, true
	case nextService:
		message := nextMismatch
		if from >= len(pipeline.stages) {
			message = targetMismatch
		}
		x.err = httpError(ctx, internalInterceptorReturnError, message, nil)
		return from, true
	default:
		// Unlike jBallerina, a () return ends the exchange even after ctx.next() (#9884). The
		// response starts below the stage ctx.next() reached, as jBallerina's does.
		x.answer(ctx, result, hasCaller, from-1)
		return 0, false
	}
}

// syncRequest carries the headers and payload an interceptor set on its http:Request over to
// the stages after it.
func (x *exchange) syncRequest(ctx *extern.Context, req *values.Object) error {
	if req == nil {
		return nil
	}
	header := http.Header{}
	for name, values := range rawRequestHeaders(req) {
		header[http.CanonicalHeaderKey(name)] = values
	}
	x.r.Header = header
	holder, _ := req.Get("$body")
	h, ok := holder.(*requestBodyHolder)
	switch {
	case !ok:
	case h.parts != nil:
		data, wireContentType, err := encodeMultipartBody(ctx, h.parts, header.Get("Content-Type"))
		if err != nil {
			return err
		}
		header.Set("Content-Type", wireContentType)
		x.replaceBody(data)
	case h.payloadSet:
		x.replaceBody(h.buf)
		x.conn.payloadSet = true
	}
	return nil
}

func (x *exchange) replaceBody(body []byte) {
	x.body = body
	x.r.Header.Set("Content-Length", strconv.Itoa(len(body)))
	x.r.Header.Del("Transfer-Encoding")
}

// dispatchService dispatches the exchange to the resource of the service that matches it.
func (x *exchange) dispatchService() {
	rt, r, w, found := x.conn.rt, x.r, x.w, x.service
	ctx := rt.AcquirePooledContext()
	httpMethod := x.method
	// Resource-level dispatch is delegated to the language runtime: it coerces
	// the raw path segments to each resource's declared parameter types and
	// selects the unique matching resource (the same dispatch used by
	// client->/path access). HTTP only owns service-level (base-path) routing.
	for _, accessorKey := range []string{httpMethod, "default"} {
		handle, extraArgs, ok := ctx.LookupResourceMethodByPath(found.svcObj, accessorKey, x.segments)
		if !ok {
			continue
		}
		x.conn.accessor = accessorKey
		if !x.intercepted() {
			w.useService(found.config)
		}
		x.invokeResource(ctx, handle, extraArgs)
		return
	}
	defer rt.ReleasePooledContext(ctx)
	w.useService(defaultServiceConfig())
	// The path matched a service but no resource under the requested method. If
	// the same path resolves under a different method it is a 405; otherwise 404.
	var served []string
	for _, accessor := range found.svcObj.AllResourceMethodNames() {
		if accessor == httpMethod || accessor == "default" {
			continue
		}
		if _, _, ok := ctx.LookupResourceMethodByPath(found.svcObj, accessor, x.segments); ok {
			served = append(served, accessor)
			if httpMethod != "options" {
				break
			}
		}
	}
	notFound := "no matching resource found for path : " + r.RequestURI + " , method : " + r.Method
	if httpMethod != "options" && x.intercepted() {
		// A dispatching error goes to the response error interceptors.
		errType, message := internalResourceNotFoundError, notFound
		if served != nil {
			errType, message = internalResourceMethodNotAllowedError, "Method not allowed"
		}
		x.deliver(nil, httpError(ctx, errType, message, nil), x.lastStage())
		return
	}
	if !x.conn.claim() {
		return
	}
	defer x.conn.markSent()
	switch {
	case httpMethod == "options":
		// As jBallerina, only the base path itself, ignoring trailing slashes and without a query, lists the whole service.
		basePath := (&url.URL{Path: found.basePath}).EscapedPath()
		requestPath := r.URL.EscapedPath()
		if trimmed := strings.TrimRight(requestPath, "/"); trimmed != "" {
			requestPath = trimmed
		}
		answerOptions(rt, w, r, found, served, requestPath == basePath && r.URL.RawQuery == "")
	case served != nil:
		writeErrorJSON(rt, w, r, http.StatusMethodNotAllowed, "Method not allowed")
	default:
		writeErrorJSON(rt, w, r, http.StatusNotFound, notFound)
	}
}

// answerOptions answers an OPTIONS request no resource serves: a CORS preflight, or 204 with an
// Allow header listing the methods the path serves, or those of the whole service at its base path.
func answerOptions(rt *runtime.Runtime, w *responseEncoder, r *http.Request, found *serviceEntry, served []string,
	basePath bool) {
	methods := allowedMethods(served)
	if methods != nil {
		if preflight := found.config.cors.preflightHeaders(r, methods[:len(methods)-1]); preflight != nil {
			maps.Copy(w.Header(), preflight)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	} else if basePath {
		methods = allowedMethods(found.svcObj.AllResourceMethodNames())
	}
	if methods == nil {
		writeErrorJSON(rt, w, r, http.StatusNotFound, "no matching resource found for path : "+r.RequestURI+" , method : OPTIONS")
		return
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	w.WriteHeader(http.StatusNoContent)
}

// bindResource binds the arguments of the resource behind handle and returns them with the
// http:Request it binds, if any, or the binding failure.
func bindResource(ctx *extern.Context, state *listenerState, conn *callerConn, handle extern.MethodHandle,
	extraArgs int) ([]values.BalValue, bool, *values.Object, *bindingError) {
	if extraArgs == 0 {
		closeBody(conn.r)
		return nil, false, nil, nil
	}
	return resourceArgs(ctx, state, handle, conn)
}

// invokeResource binds the arguments of the resource behind handle, calls it and sends its
// response. It owns ctx and returns it to the pool once the resource has finished. A binding
// failure is answered with its error response, through the response error interceptors.
func (x *exchange) invokeResource(ctx *extern.Context, handle extern.MethodHandle, extraArgs int) {
	conn := x.conn
	args, hasCaller, _, bindErr := bindResource(ctx, x.state, conn, handle, extraArgs)
	if bindErr != nil {
		defer conn.rt.ReleasePooledContext(ctx)
		if x.intercepted() {
			x.deliver(nil, bindErr.toError(ctx), x.lastStage())
			return
		}
		conn.fail(bindErr.status, bindErr.message)
		return
	}
	if hasCaller {
		// The handler returns once a response is sent, which ends the exchange (an HTTP/2
		// stream ends only then), while the resource carries on.
		go runCallerResource(ctx, conn, handle, args)
		<-conn.sent
		return
	}
	defer conn.rt.ReleasePooledContext(ctx)
	result, failure, failed := invokeStage(ctx, conn.rt, conn.r, handle, args)
	switch {
	case failed:
		conn.failPanic(failure)
	case x.intercepted():
		x.answer(ctx, result, false, x.lastStage())
	default:
		conn.complete(ctx, result, false)
	}
}

// runCallerResource runs a resource that has an http:Caller parameter.
func runCallerResource(ctx *extern.Context, conn *callerConn, handle extern.MethodHandle, args []values.BalValue) {
	rt := conn.rt
	defer rt.ReleasePooledContext(ctx)
	defer conn.markSent()
	defer func() {
		if rec := recover(); rec != nil {
			ctx.ReleaseAllHeldLocks()
			message := logPanic(rt, conn.r, rec)
			if !conn.close() {
				writePanicResponse(conn.w, message)
			}
		}
	}()
	result, err := ctx.InvokeMethod(handle, args)
	if err != nil {
		if !conn.close() {
			writePanicResponse(conn.w, panicMessage(err))
		}
		return
	}
	if x := conn.exchange; x.intercepted() {
		x.answer(ctx, result, true, x.lastStage())
		return
	}
	conn.finish(ctx, result)
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
func buildRequestFromHTTP(r *http.Request) (*values.Object, error) {
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
	return buildRequest(r.Method, r.RequestURI, r.Proto, r.Header, bodyStream, cl, r.URL.RawQuery, bodyBuf), nil
}

// buildRequest constructs a Ballerina Request object from HTTP request data.
// bodyStream is the raw request body; it is stored lazily in a requestBodyHolder
// so the body is only read from the network when a getPayload method is called.
// bodyBuf, when non-nil, is an already-read body; bodyStream must be nil in that case.
func buildRequest(method, rawPath, httpVersion string, headers map[string][]string, bodyStream io.ReadCloser, contentLength int64, rawQuery string, bodyBuf []byte) *values.Object {
	// Headers are stored lazily: the raw map is only converted to the Ballerina
	// header Map on first access (see requestHeadersMap). A request that never
	// reads a header skips building that value graph entirely.
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
		semtypes.Object,
		map[string]values.BalValue{
			"rawPath":     rawPath,
			"method":      method,
			"httpVersion": httpVersion,
			"$headers":    &lazyRequestHeaders{raw: headers},
			"$body":       holder,
			"$queryStr":   rawQuery,
		},
		requestMethodKeys(),
		nil,
		nil,
	)
}

// writePanicResponse answers a request whose resource panicked: a 500 with the panic message as text.
func writePanicResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = io.WriteString(w, message)
}

// writeErrorJSON writes an ErrorPayload error response for the request.
func writeErrorJSON(rt *runtime.Runtime, w http.ResponseWriter, r *http.Request, status int, message string) {
	body := errorPayload(rt.Platform().Time.Now(), status, message, r.RequestURI, r.Method)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeResult writes a Ballerina resource method return value as an HTTP response,
// following jBallerina's Caller.returnResponse (http_connection.bal:141-196).
func writeResult(rt *runtime.Runtime, ctx *extern.Context, types *httpTypes,
	w http.ResponseWriter, r *http.Request, accessor string, result values.BalValue) {
	switch v := result.(type) {
	case nil:
		w.WriteHeader(http.StatusAccepted)
	case *values.Error:
		resp, err := errorResponse(ctx, v, r)
		if err != nil {
			writeErrorJSON(rt, w, r, http.StatusInternalServerError, panicMessage(err))
			return
		}
		writeResult(rt, ctx, types, w, r, accessor, resp)
	case *values.Object:
		_ = writeResponse(rt, ctx, w, r, v)
	case *values.Map:
		var resp *values.Object
		var err error
		// An anydata mapping cannot hold the status object every status code record carries.
		if status, _ := v.Get("status"); status != nil {
			if _, isObject := status.(*values.Object); isObject {
				resp, err = statusCodeResponse(ctx, v)
			}
		}
		switch {
		case err != nil:
			writeErrorJSON(rt, w, r, http.StatusInternalServerError, panicMessage(err))
		case resp != nil:
			writeResult(rt, ctx, types, w, r, accessor, resp)
		default:
			writeAnydataResult(rt, ctx, types, w, r, accessor, v)
		}
	default:
		writeAnydataResult(rt, ctx, types, w, r, accessor, v)
	}
}

// writeResponse writes resp. When resp cannot be sent as given it writes a 500 instead and
// returns the reason.
func writeResponse(rt *runtime.Runtime, ctx *extern.Context, w http.ResponseWriter, r *http.Request, resp *values.Object) error {
	holder, ok := responseBodyOf(resp)
	if !ok {
		return writeErrorResponse(rt, w, r, "unexpected return type from resource method")
	}
	statusCodeVal, _ := resp.Get("statusCode")
	statusCode := statusCodeVal.(int64)
	if !isFinalStatusCode(statusCode) {
		holder.discard()
		return writeErrorResponse(rt, w, r, fmt.Sprintf("invalid status code for a response: %d", statusCode))
	}
	wireContentType := ""
	if holder.parts != nil {
		data, ct, err := encodeMultipartBody(ctx, holder.parts, responseContentType(resp))
		if err != nil {
			return writeErrorResponse(rt, w, r, err.Error())
		}
		holder = &responseBodyHolder{buf: data}
		wireContentType = ct
	}
	writeResponseObject(w, resp, int(statusCode), holder, wireContentType)
	return nil
}

// writeErrorResponse writes a 500 ErrorPayload with message and returns message as an error.
func writeErrorResponse(rt *runtime.Runtime, w http.ResponseWriter, r *http.Request, message string) error {
	writeErrorJSON(rt, w, r, http.StatusInternalServerError, message)
	return errors.New(message)
}

// errorResponse converts an error a resource returned into the http:Response it is sent as.
func errorResponse(ctx *extern.Context, err *values.Error, r *http.Request) (*values.Object, error) {
	result, invokeErr := invokeModuleFunction(ctx, "toErrorResponse", err, r.RequestURI, r.Method)
	if invokeErr != nil {
		return nil, invokeErr
	}
	return result.(*values.Object), nil
}

// isFinalStatusCode reports whether code can end a response: a three-digit code other than an
// informational 1xx, which net/http would send as an interim response followed by a 200.
// 101 Switching Protocols is final.
func isFinalStatusCode(code int64) bool {
	return code == http.StatusSwitchingProtocols || (code >= 200 && code <= 999)
}

// statusCodeResponse converts a returned status code record into the http:Response it is sent
// as; the result is nil for any other mapping value.
func statusCodeResponse(ctx *extern.Context, v *values.Map) (*values.Object, error) {
	result, err := invokeModuleFunction(ctx, "toStatusCodeResponse", v)
	if err != nil {
		return nil, err
	}
	resp, _ := result.(*values.Object)
	return resp, nil
}

// An http:Response is identified by its internal body slot rather than by its type, which the
// server side has no handle on. The holder type is private to this package, so no
// Ballerina-defined object can forge it.
func responseBodyOf(obj *values.Object) (*responseBodyHolder, bool) {
	bodyVal, ok := obj.Get("body")
	if !ok {
		return nil, false
	}
	holder, ok := bodyVal.(*responseBodyHolder)
	return holder, ok
}

// The 201 rule keys on the declared accessor, not the request method, so a `default`
// resource reached by a POST stays 200 — matching jBallerina.
func writeAnydataResult(rt *runtime.Runtime, ctx *extern.Context, types *httpTypes,
	w http.ResponseWriter, r *http.Request, accessor string, v values.BalValue) {
	body, contentType, err := outboundPayload(ctx, types, v, "")
	if err != nil {
		writeErrorJSON(rt, w, r, http.StatusInternalServerError,
			"failed to serialize resource return value: "+err.Error())
		return
	}
	status := http.StatusOK
	if accessor == "post" {
		status = http.StatusCreated
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// wireContentType, when non-empty, replaces the Content-Type header, carrying the boundary a
// multipart body was encoded with.
func writeResponseObject(w http.ResponseWriter, resp *values.Object, statusCode int, holder *responseBodyHolder, wireContentType string) {
	// Forwarding hop-by-hop headers (e.g. Transfer-Encoding, Connection) from a
	// backend response to the downstream client violates RFC 7230 §6.1 and can
	// cause framing errors in HTTP/1.1 keep-alive connections. Headers the
	// response's own Connection value nominates are hop-by-hop too.
	if hdrsVal, ok := resp.Get("$headers"); ok {
		if hdrs, ok := hdrsVal.(*values.Map); ok {
			skip := hopByHopSkipSet(headerListValues(hdrs, "connection"))
			for _, k := range hdrs.Keys() {
				if _, ok := skip[strings.ToLower(k)]; ok {
					continue
				}
				val, _ := hdrs.Get(k)
				list := val.(*values.List)
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
	if wireContentType != "" {
		// The body parts were re-encoded, so a Content-Length copied from a backend no longer holds.
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Type", wireContentType)
	}
	// WriteHeader must be called before writing the body; once body bytes
	// start flowing via writeStream, headers are already committed, so a write error here
	// can't be recovered with a JSON error body.
	w.WriteHeader(statusCode)
	if err := holder.writeStream(w); err != nil {
		if encoder, ok := w.(*responseEncoder); ok {
			encoder.abort()
		}
	}
}

// copyBufPool reuses 32 KB buffers for streaming a backend response body to the
// downstream client. Without it, io.Copy → net/http's response.ReadFrom
// allocates a fresh 32 KB buffer on every request (kernel splice is unavailable
// when the source is a userspace stream), which dominates allocations at large
// payloads.
var copyBufPool = sync.Pool{New: func() any { b := make([]byte, 32*1024); return &b }}

// onlyWriter exposes only Write, hiding any ReadFrom the underlying writer has,
// so io.CopyBuffer uses our pooled buffer instead of response.ReadFrom's
// per-call allocation. Output is identical — the ResponseWriter's Write already
// handles chunked/content-length framing.
type onlyWriter struct{ io.Writer }

// discard closes a body stream that will not be written, releasing its connection.
func (h *responseBodyHolder) discard() {
	h.once.Do(func() {
		if h.stream != nil {
			_ = h.stream.Close()
			h.stream = nil
			h.buf = []byte{}
		}
	})
}

// writeStream writes the body to w via io.CopyBuffer (streaming, pooled buffer)
// or w.Write (buffered), then closes the stream. After this call the holder is
// exhausted.
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
		bufp := copyBufPool.Get().(*[]byte)
		_, err := io.CopyBuffer(onlyWriter{w}, s, *bufp)
		copyBufPool.Put(bufp)
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
