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
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
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

var httpPackageID = model.NewPackageID(
	model.DefaultPackageIDInterner,
	model.Name("ballerina"),
	[]model.Name{model.Name("http")},
	model.Name("0.0.1"),
)

const (
	orgName    = "ballerina"
	moduleName = "http"
)

// hopByHopHeaders is the set of headers that must not be forwarded by a proxy per
// RFC 7230 §6.1 and RFC 2616 §13.5.1. Keys are lowercase canonical form.
var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"proxy-connection":    {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

// removeHopByHopHeaders deletes hop-by-hop entries from h in place.
// It also honours the Connection header's own token list per RFC 7230 §6.1.
func removeHopByHopHeaders(h map[string][]string) {
	if connVals, ok := h["connection"]; ok {
		for _, f := range connVals {
			for _, tok := range strings.Split(f, ",") {
				delete(h, strings.ToLower(strings.TrimSpace(tok)))
			}
		}
	}
	for k := range h {
		if _, skip := hopByHopHeaders[strings.ToLower(k)]; skip {
			delete(h, k)
		}
	}
}

func init() {
	runtime.RegisterModuleInitializer(initHttpModule)
}

// httpTypes holds the semtypes used by the http runtime. They are built once
// in initHttpModule (single-threaded) and captured by the handler closures.
type httpTypes struct {
	byteArrTy  semtypes.SemType
	strArrTy   semtypes.SemType
	jsonListTy semtypes.SemType
	jsonMapTy  semtypes.SemType
}

// 8 KB matches Netty's HttpObjectDecoder.maxChunkSize used by jBallerina's transport.
// Bodies that fit in one chunk are buffered eagerly to skip the holder cost.
const eagerBufferThreshold = 8192

// requestBodyHolder holds an outbound (or inbound) request body.
// lazy — stream (io.ReadCloser) has not been read yet;
// materialized — the body has been read into buf.
type requestBodyHolder struct {
	once          sync.Once
	stream        io.ReadCloser
	buf           []byte
	readErr       error
	contentLength int64 // -1 if unknown; >=0 is the known byte count
}

func (h *requestBodyHolder) materialize() []byte {
	h.once.Do(func() {
		if h.stream != nil {
			data, err := io.ReadAll(h.stream)
			_ = h.stream.Close()
			h.stream = nil
			h.readErr = err
			h.buf = data
		}
		if h.buf == nil {
			h.buf = []byte{}
		}
	})
	return h.buf
}

// takeStream atomically takes ownership of the stream for zero-copy passthrough.
// Returns the stream (and clears it) when the body has not yet been read.
// Returns nil if already materialized; callers should fall back to materialize().
func (h *requestBodyHolder) takeStream() io.ReadCloser {
	var taken io.ReadCloser
	h.once.Do(func() {
		if h.stream != nil {
			taken = h.stream
			h.stream = nil
			h.buf = []byte{}
		}
	})
	return taken
}

// responseBodyHolder holds an HTTP response body.
// streaming — stream (io.ReadCloser) is available;
// materialized — the body has been read into buf.
type responseBodyHolder struct {
	once    sync.Once
	stream  io.ReadCloser
	buf     []byte
	readErr error // set if the stream returned an error during materialization
}

func (h *responseBodyHolder) materialize() ([]byte, error) {
	h.once.Do(func() {
		if h.stream != nil {
			var err error
			h.buf, err = io.ReadAll(h.stream)
			_ = h.stream.Close()
			h.stream = nil
			h.readErr = err
		}
		if h.buf == nil {
			h.buf = []byte{}
		}
	})
	return h.buf, h.readErr
}

// eagerBufferResponse pre-buffers the response when Content-Length fits within
// eagerBufferThreshold; otherwise stores the stream lazily.
func eagerBufferResponse(respHeaders map[string][]string, bodyStream io.ReadCloser) *responseBodyHolder {
	if bodyStream == nil {
		return &responseBodyHolder{buf: []byte{}}
	}
	var cl int64 = -1
	for k, vals := range respHeaders {
		if strings.EqualFold(k, "content-length") && len(vals) > 0 {
			if n, err := strconv.ParseInt(strings.TrimSpace(vals[0]), 10, 64); err == nil {
				cl = n
			}
			break
		}
	}
	if cl >= 0 && cl <= eagerBufferThreshold {
		data, readErr := io.ReadAll(bodyStream)
		_ = bodyStream.Close()
		return &responseBodyHolder{buf: data, readErr: readErr}
	}
	return &responseBodyHolder{stream: bodyStream}
}

// newMappingValue builds a fresh open map<anydata|error> value.
func newMappingValue(tc semtypes.Context) *values.Map {
	return values.NewMap(semtypes.MAPPING, semtypes.ToMappingAtomicType(tc, semtypes.MAPPING), false, nil)
}

// newListValue builds a fresh open list seeded with items.
func newListValue(tc semtypes.Context, items []values.BalValue) *values.List {
	return values.NewList(semtypes.LIST, semtypes.ToListAtomicType(tc, semtypes.LIST), false, nil, 0, items)
}

// newTypedListValue builds a typed list seeded with items.
func newTypedListValue(tc semtypes.Context, ty semtypes.SemType, items []values.BalValue) *values.List {
	return values.NewList(ty, semtypes.ToListAtomicType(tc, ty), false, nil, 0, items)
}

// setRequestHeader sets a single-value header on an http:Request object's $headers map.
func setRequestHeader(self *values.Object, name, val string, tc semtypes.Context) {
	hdrsVal, ok := self.Get("$headers")
	var hdrs *values.Map
	if ok {
		hdrs, _ = hdrsVal.(*values.Map)
	}
	if hdrs == nil {
		hdrs = newMappingValue(tc)
		self.Put("$headers", hdrs)
	}
	hdrs.Put(tc, name, newListValue(tc, []values.BalValue{val}))
}

// goCtxOrBackground returns the Go context from an extern.Context if available,
// otherwise returns context.Background().
func goCtxOrBackground(_ *extern.Context) context.Context {
	return context.Background()
}

// compressionModeOf reads the "$compression" field from a Client object.
// Returns "AUTO" when the field is absent (safe default).
func compressionModeOf(self *values.Object) string {
	if v, ok := self.Get("$compression"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "AUTO"
}

func initHttpModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	jsonTy := semtypes.CreateJSON(semtypes.ContextFrom(env))
	byteArrLd := semtypes.NewListDefinition()
	strArrLd := semtypes.NewListDefinition()
	jsonMapMd := semtypes.NewMappingDefinition()
	jsonListLd := semtypes.NewListDefinition()
	types := httpTypes{
		byteArrTy:  byteArrLd.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE),
		strArrTy:   strArrLd.DefineListTypeWrappedWithEnvSemType(env, semtypes.STRING),
		jsonMapTy:  jsonMapMd.DefineMappingTypeWrapped(env, nil, jsonTy),
		jsonListTy: jsonListLd.DefineListTypeWrappedWithEnvSemType(env, jsonTy),
	}

	// msgToBody converts a Ballerina RequestMessage value to (io.Reader, contentLength, contentType).
	msgToBody := func(tc semtypes.Context, msg values.BalValue) (io.Reader, int64, string) {
		switch v := msg.(type) {
		case string:
			b := []byte(v)
			return bytes.NewReader(b), int64(len(b)), "text/plain"
		case *values.Object:
			// http:Request object — extract body and content-type
			ct := "application/octet-stream"
			if hdrsVal, ok := v.Get("$headers"); ok {
				if hdrs, ok := hdrsVal.(*values.Map); ok {
					if cv, ok := hdrs.Get("content-type"); ok {
						if list, ok := cv.(*values.List); ok && list.Len() > 0 {
							if s, ok := list.Get(0).(string); ok {
								ct = s
							}
						}
					}
				}
			}
			bodyVal, _ := v.Get("$body")
			if holder, ok := bodyVal.(*requestBodyHolder); ok {
				buf := holder.materialize()
				if holder.readErr != nil {
					return nil, 0, "json_error"
				}
				return bytes.NewReader(buf), int64(len(buf)), ct
			}
			return bytes.NewReader([]byte{}), 0, ct
		case *values.List:
			if !semtypes.IsZero(v.Type) && semtypes.IsSubtype(tc, v.Type, types.byteArrTy) {
				if b, ok := listToBytes(v); ok {
					return bytes.NewReader(b), int64(len(b)), "application/octet-stream"
				}
			}
			b, err := toJSONBytes(v)
			if err != nil {
				return nil, 0, "json_error"
			}
			return bytes.NewReader(b), int64(len(b)), "application/json"
		default:
			b, err := toJSONBytes(v)
			if err != nil {
				return nil, 0, "json_error"
			}
			return bytes.NewReader(b), int64(len(b)), "application/json"
		}
	}

	execBody := func(ctx *extern.Context, verb string, args []values.BalValue) (values.BalValue, error) {
		self := args[0].(*values.Object)
		path := args[1].(string)
		var bodyReader io.Reader
		var contentLength int64
		contentType := ""
		if len(args) > 2 && args[2] != nil {
			var ct string
			bodyReader, contentLength, ct = msgToBody(ctx.TypeCtx, args[2])
			if bodyReader == nil && ct == "json_error" {
				return values.NewErrorWithMessage("failed to serialize body to JSON"), nil
			}
			contentType = ct
		}
		var reqHeaders map[string][]string
		if len(args) > 3 {
			reqHeaders = extractHeaders(args[3])
			for hdrKey, hdrVals := range reqHeaders {
				if strings.EqualFold(hdrKey, "content-type") && len(hdrVals) > 0 {
					contentType = hdrVals[0]
					break
				}
			}
		}
		if len(args) > 4 {
			if mt, ok := args[4].(string); ok && mt != "" {
				contentType = mt
			}
		}
		reqHeaders = applyCompressionHeaders(compressionModeOf(self), reqHeaders)
		urlVal, _ := self.Get("url")
		clientHandle, _ := self.Get("$httpClient")
		statusCode, respHeaders, respBodyStream, err := clientHandle.(pal.HTTPClient).Execute(
			goCtxOrBackground(ctx), verb, urlVal.(string)+path, bodyReader, contentLength, contentType, reqHeaders)
		if err != nil {
			return values.NewErrorWithMessage(err.Error()), nil
		}
		return buildResponse(ctx.TypeCtx, statusCode, respHeaders, respBodyStream), nil
	}

	// Client class def.
	clientClassDef := &bir.BIRClassDef{
		Name:      model.Name("Client"),
		LookupKey: "ballerina/http:Client",
		Fields: []bir.ObjectField{
			{Name: "url", Ty: semtypes.STRING},
			{Name: "timeout", Ty: semtypes.DECIMAL},
			{Name: "followRedirects", Ty: semtypes.Union(semtypes.MAPPING, semtypes.NIL)},
			{Name: "httpVersion", Ty: semtypes.STRING},
		},
		VTable: map[string]*bir.BIRFunction{
			"init":            {FunctionLookupKey: "ballerina/http:Client.init"},
			"initNative":      {FunctionLookupKey: "ballerina/http:Client.initNative"},
			"$remote$get":     {FunctionLookupKey: "ballerina/http:Client.$remote$get"},
			"$remote$post":    {FunctionLookupKey: "ballerina/http:Client.$remote$post"},
			"$remote$head":    {FunctionLookupKey: "ballerina/http:Client.$remote$head"},
			"$remote$options": {FunctionLookupKey: "ballerina/http:Client.$remote$options"},
			"$remote$put":     {FunctionLookupKey: "ballerina/http:Client.$remote$put"},
			"$remote$patch":   {FunctionLookupKey: "ballerina/http:Client.$remote$patch"},
			"$remote$delete":  {FunctionLookupKey: "ballerina/http:Client.$remote$delete"},
			"$remote$execute": {FunctionLookupKey: "ballerina/http:Client.$remote$execute"},
			"$remote$forward": {FunctionLookupKey: "ballerina/http:Client.$remote$forward"},
		},
	}
	runtime.RegisterExternClassDef(rt, clientClassDef)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "parseHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("parseHeader: expected string argument")
			}
			result, err := parseHeader(ctx.TypeCtx, input)
			if err != nil {
				return values.NewErrorWithMessage(err.Error()), nil
			}
			return result, nil
		})

	// initNative is the extern called by the Ballerina Client.init wrapper.
	// args are always [self, url, config].
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.initNative",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			url := args[1].(string)

			timeout := decimal.FromInt64(30)
			var followRedirects pal.FollowRedirects
			httpVersion := "2.0"
			var tlsCfg pal.TLSConfig
			var poolCfg pal.PoolConfig
			// Defaults match jBallerina's ResponseLimitConfigs and CommonClientConfiguration.
			responseLimits := pal.ResponseLimitConfig{
				MaxStatusLineLength: 4096,
				MaxHeaderSize:       8192,
				MaxEntityBodySize:   -1,
			}
			var proxyCfg pal.ProxyConfig

			if cfg, ok := args[2].(*values.Map); ok {
				if v, ok := cfg.Get("timeout"); ok {
					if d, ok := v.(*decimal.Decimal); ok {
						timeout = d
					}
				}
				if v, ok := cfg.Get("followRedirects"); ok {
					if frMap, ok := v.(*values.Map); ok {
						if ev, ok := frMap.Get("enabled"); ok {
							if b, ok := ev.(bool); ok {
								followRedirects.Enabled = b
							}
						}
						followRedirects.MaxCount = 5
						if mv, ok := frMap.Get("maxCount"); ok {
							if n, ok := mv.(int64); ok {
								followRedirects.MaxCount = int(n)
							}
						}
						if av, ok := frMap.Get("allowAuthHeaders"); ok {
							if b, ok := av.(bool); ok {
								followRedirects.AllowAuthHeaders = b
							}
						}
					}
				}
				if v, ok := cfg.Get("httpVersion"); ok {
					if s, ok := v.(string); ok {
						httpVersion = s
					}
				}
				if httpVersion == "1.0" {
					msg := "warning [ballerina/http]: HTTP/1.0 is not supported by the Go HTTP runtime; falling back to HTTP/1.1\n"
					_, _ = rt.Platform().IO.Stderr([]byte(msg))
					httpVersion = "1.1"
				}
				if ss, ok := cfg.Get("secureSocket"); ok {
					if ssMap, ok := ss.(*values.Map); ok {
						if v, ok := ssMap.Get("enable"); ok {
							if b, ok := v.(bool); ok && !b {
								tlsCfg.InsecureSkipVerify = true
							}
						}
						if v, ok := ssMap.Get("verifyHostName"); ok {
							if b, ok := v.(bool); ok && !b {
								tlsCfg.InsecureSkipVerify = true
							}
						}
						if v, ok := ssMap.Get("cert"); ok {
							if certPath, ok := v.(string); ok && certPath != "" {
								data, err := rt.Platform().FS.ReadFile(certPath)
								if err != nil {
									return values.NewErrorWithMessage("secureSocket.cert: " + err.Error()), nil
								}
								tlsCfg.CACertPEM = data
							}
						}
						if v, ok := ssMap.Get("key"); ok {
							if keyMap, ok := v.(*values.Map); ok {
								if cv, ok := keyMap.Get("certFile"); ok {
									if p, ok := cv.(string); ok && p != "" {
										data, err := rt.Platform().FS.ReadFile(p)
										if err != nil {
											return values.NewErrorWithMessage("secureSocket.key.certFile: " + err.Error()), nil
										}
										tlsCfg.ClientCertPEM = data
									}
								}
								if kv, ok := keyMap.Get("keyFile"); ok {
									if p, ok := kv.(string); ok && p != "" {
										data, err := rt.Platform().FS.ReadFile(p)
										if err != nil {
											return values.NewErrorWithMessage("secureSocket.key.keyFile: " + err.Error()), nil
										}
										tlsCfg.ClientKeyPEM = data
									}
								}
								// keyPassword: accepted at compile time, ignored at runtime
							}
						}
						if v, ok := ssMap.Get("serverName"); ok {
							if s, ok := v.(string); ok && s != "" {
								tlsCfg.ServerName = s
							}
						}
						if v, ok := ssMap.Get("shareSession"); ok {
							if b, ok := v.(bool); ok && !b {
								tlsCfg.DisableSessionTickets = true
							}
						}
						if v, ok := ssMap.Get("handshakeTimeout"); ok {
							if d, ok := v.(*decimal.Decimal); ok {
								tlsCfg.HandshakeTimeout = decimalToDuration(d)
							}
						}
						if v, ok := ssMap.Get("ciphers"); ok {
							if list, ok := v.(*values.List); ok {
								for i := 0; i < list.Len(); i++ {
									if name, ok := list.Get(i).(string); ok {
										tlsCfg.CipherSuiteNames = append(tlsCfg.CipherSuiteNames, name)
									}
								}
							}
						}
						if v, ok := ssMap.Get("protocol"); ok {
							if protoMap, ok := v.(*values.Map); ok {
								if vv, ok := protoMap.Get("versions"); ok {
									if list, ok := vv.(*values.List); ok {
										tlsVersionMap := map[string]uint16{
											"TLSv1.0": 0x0301,
											"TLSv1.1": 0x0302,
											"TLSv1.2": 0x0303,
											"TLSv1.3": 0x0304,
										}
										for i := 0; i < list.Len(); i++ {
											if s, ok := list.Get(i).(string); ok {
												if ver, found := tlsVersionMap[s]; found {
													if tlsCfg.MinVersion == 0 || ver < tlsCfg.MinVersion {
														tlsCfg.MinVersion = ver
													}
													if ver > tlsCfg.MaxVersion {
														tlsCfg.MaxVersion = ver
													}
												}
											}
										}
									}
								}
							}
						}
						// certValidation/sessionTimeout: accepted at compile time, not supported at runtime
					}
				}
				if v, ok := cfg.Get("poolConfig"); ok {
					if pcMap, ok := v.(*values.Map); ok {
						if mv, ok := pcMap.Get("maxIdleConnections"); ok {
							if n, ok := mv.(int64); ok {
								poolCfg.MaxIdleConnsPerHost = int(n)
							}
						}
						if mv, ok := pcMap.Get("maxActiveConnections"); ok {
							if n, ok := mv.(int64); ok {
								if n < 0 {
									poolCfg.MaxConnsPerHost = 0 // -1 means unlimited in Ballerina → 0 in Go
								} else {
									poolCfg.MaxConnsPerHost = int(n)
								}
							}
						}
						if mv, ok := pcMap.Get("waitTime"); ok {
							if d, ok := mv.(*decimal.Decimal); ok {
								poolCfg.ResponseHeaderTimeout = decimalToDuration(d)
							}
						}
						// maxActiveStreamsPerConnection: HTTP/2 only; not directly mappable in Go transport
					}
				}
				// Always disable Go's automatic Accept-Encoding injection so we control
				// it precisely per the compression mode (jBallerina AbstractHTTPAction logic).
				poolCfg.DisableCompression = true
			}
			compressionMode := "AUTO"
			if cfg, ok := args[2].(*values.Map); ok {
				if v, ok := cfg.Get("compression"); ok {
					if s, ok := v.(string); ok && s != "" {
						compressionMode = s
					}
				}
				if v, ok := cfg.Get("responseLimits"); ok {
					if rlMap, ok := v.(*values.Map); ok {
						if mv, ok := rlMap.Get("maxStatusLineLength"); ok {
							if n, ok := mv.(int64); ok {
								if n < 0 {
									return values.NewErrorWithMessage("invalid value for responseLimits.maxStatusLineLength: must be >= 0"), nil
								}
								responseLimits.MaxStatusLineLength = int(n)
							}
						}
						if mv, ok := rlMap.Get("maxHeaderSize"); ok {
							if n, ok := mv.(int64); ok {
								if n < 0 {
									return values.NewErrorWithMessage("invalid value for responseLimits.maxHeaderSize: must be >= 0"), nil
								}
								responseLimits.MaxHeaderSize = n
							}
						}
						if mv, ok := rlMap.Get("maxEntityBodySize"); ok {
							if n, ok := mv.(int64); ok {
								if n < -1 {
									return values.NewErrorWithMessage("invalid value for responseLimits.maxEntityBodySize: must be >= -1"), nil
								}
								responseLimits.MaxEntityBodySize = n
							}
						}
					}
				}
				if v, ok := cfg.Get("proxy"); ok {
					if proxyMap, ok := v.(*values.Map); ok {
						if hv, ok := proxyMap.Get("host"); ok {
							if s, ok := hv.(string); ok {
								proxyCfg.Host = s
							}
						}
						if pv, ok := proxyMap.Get("port"); ok {
							if n, ok := pv.(int64); ok {
								proxyCfg.Port = int(n)
							}
						}
						if uv, ok := proxyMap.Get("userName"); ok {
							if s, ok := uv.(string); ok {
								proxyCfg.UserName = s
							}
						}
						if pwv, ok := proxyMap.Get("password"); ok {
							if s, ok := pwv.(string); ok {
								proxyCfg.Password = s
							}
						}
					}
				}
			}
			httpClient := rt.Platform().HTTP.NewClient(pal.ClientConfig{
				Timeout:         decimalToDuration(timeout),
				FollowRedirects: followRedirects,
				HTTPVersion:     httpVersion,
				TLS:             tlsCfg,
				Pool:            poolCfg,
				ResponseLimits:  responseLimits,
				Proxy:           proxyCfg,
			})
			self.Put("url", url)
			self.Put("timeout", timeout)
			self.Put("followRedirects", nil)
			self.Put("httpVersion", httpVersion)
			self.Put("$compression", compressionMode)
			self.Put("$httpClient", httpClient)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$get",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			path := args[1].(string)
			var reqHeaders map[string][]string
			if len(args) > 2 {
				reqHeaders = extractHeaders(args[2])
			}
			reqHeaders = applyCompressionHeaders(compressionModeOf(self), reqHeaders)
			urlVal, _ := self.Get("url")
			clientHandle, _ := self.Get("$httpClient")
			statusCode, respHeaders, respBodyStream, err := clientHandle.(pal.HTTPClient).Execute(
				goCtxOrBackground(ctx), "GET", urlVal.(string)+path, nil, 0, "", reqHeaders)
			if err != nil {
				return values.NewErrorWithMessage(err.Error()), nil
			}
			return buildResponse(ctx.TypeCtx, statusCode, respHeaders, respBodyStream), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$post",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return execBody(ctx, "POST", args)
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$head",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			path := args[1].(string)
			var reqHeaders map[string][]string
			if len(args) > 2 {
				reqHeaders = extractHeaders(args[2])
			}
			reqHeaders = applyCompressionHeaders(compressionModeOf(self), reqHeaders)
			urlVal, _ := self.Get("url")
			clientHandle, _ := self.Get("$httpClient")
			statusCode, respHeaders, respBodyStream, err := clientHandle.(pal.HTTPClient).Execute(
				goCtxOrBackground(ctx), "HEAD", urlVal.(string)+path, nil, 0, "", reqHeaders)
			if err != nil {
				return values.NewErrorWithMessage(err.Error()), nil
			}
			return buildResponse(ctx.TypeCtx, statusCode, respHeaders, respBodyStream), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$options",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			path := args[1].(string)
			var reqHeaders map[string][]string
			if len(args) > 2 {
				reqHeaders = extractHeaders(args[2])
			}
			reqHeaders = applyCompressionHeaders(compressionModeOf(self), reqHeaders)
			urlVal, _ := self.Get("url")
			clientHandle, _ := self.Get("$httpClient")
			statusCode, respHeaders, respBodyStream, err := clientHandle.(pal.HTTPClient).Execute(
				goCtxOrBackground(ctx), "OPTIONS", urlVal.(string)+path, nil, 0, "", reqHeaders)
			if err != nil {
				return values.NewErrorWithMessage(err.Error()), nil
			}
			return buildResponse(ctx.TypeCtx, statusCode, respHeaders, respBodyStream), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$put",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return execBody(ctx, "PUT", args)
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$patch",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return execBody(ctx, "PATCH", args)
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$delete",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return execBody(ctx, "DELETE", args)
		})

	// execute: args = [self, httpVerb, path, message, headers?, mediaType?]
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$execute",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			verb := args[1].(string)
			path := args[2].(string)

			var bodyReader io.Reader
			var contentLength int64
			contentType := ""
			if len(args) > 3 && args[3] != nil {
				var ct string
				bodyReader, contentLength, ct = msgToBody(ctx.TypeCtx, args[3])
				if bodyReader == nil && ct == "json_error" {
					return values.NewErrorWithMessage("failed to serialize body to JSON"), nil
				}
				contentType = ct
			}
			var reqHeaders map[string][]string
			if len(args) > 4 {
				reqHeaders = extractHeaders(args[4])
				for hdrKey, hdrVals := range reqHeaders {
					if strings.EqualFold(hdrKey, "content-type") && len(hdrVals) > 0 {
						contentType = hdrVals[0]
						break
					}
				}
			}
			if len(args) > 5 {
				if mt, ok := args[5].(string); ok && mt != "" {
					contentType = mt
				}
			}
			reqHeaders = applyCompressionHeaders(compressionModeOf(self), reqHeaders)
			urlVal, _ := self.Get("url")
			clientHandle, _ := self.Get("$httpClient")
			statusCode, respHeaders, respBodyStream, err := clientHandle.(pal.HTTPClient).Execute(
				goCtxOrBackground(ctx), verb, urlVal.(string)+path, bodyReader, contentLength, contentType, reqHeaders)
			if err != nil {
				return values.NewErrorWithMessage(err.Error()), nil
			}
			return buildResponse(ctx.TypeCtx, statusCode, respHeaders, respBodyStream), nil
		})

	// forward: args = [self, path, request]
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$forward",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			path := args[1].(string)
			reqObj := args[2].(*values.Object)

			methodVal, _ := reqObj.Get("method")
			method, _ := methodVal.(string)
			if method == "" {
				method = "GET"
			}

			// Extract headers from the request object.
			hdrsVal, _ := reqObj.Get("$headers")
			var reqHeaders map[string][]string
			if hdrs, ok := hdrsVal.(*values.Map); ok {
				reqHeaders = make(map[string][]string, hdrs.Len())
				for _, k := range hdrs.Keys() {
					v, _ := hdrs.Get(k)
					if list, ok := v.(*values.List); ok {
						strs := make([]string, list.Len())
						for i := range list.Len() {
							if s, ok := list.Get(i).(string); ok {
								strs[i] = s
							}
						}
						reqHeaders[k] = strs
					}
				}
			}
			// Strip hop-by-hop headers per RFC 7230 §6.1 before forwarding.
			removeHopByHopHeaders(reqHeaders)
			// Apply compression mode to Accept-Encoding after hop-by-hop removal.
			reqHeaders = applyCompressionHeaders(compressionModeOf(self), reqHeaders)

			// Obtain the request body as an io.Reader for streaming passthrough.
			bodyVal, _ := reqObj.Get("$body")
			var bodyReader io.Reader
			var forwardContentLength int64
			if holder, ok := bodyVal.(*requestBodyHolder); ok {
				if stream := holder.takeStream(); stream != nil {
					bodyReader = stream
					forwardContentLength = holder.contentLength
				} else {
					buf := holder.materialize()
					if holder.readErr != nil {
						return values.NewErrorWithMessage("failed to read request body: " + holder.readErr.Error()), nil
					}
					if len(buf) > 0 {
						bodyReader = bytes.NewReader(buf)
						forwardContentLength = int64(len(buf))
					}
				}
			}

			contentType := ""
			if cts, ok := reqHeaders["content-type"]; ok && len(cts) > 0 {
				contentType = cts[0]
			}

			urlVal, _ := self.Get("url")
			clientHandle, _ := self.Get("$httpClient")
			statusCode, respHeaders, respBodyStream, err := clientHandle.(pal.HTTPClient).Execute(
				goCtxOrBackground(ctx), method, urlVal.(string)+path, bodyReader, forwardContentLength, contentType, reqHeaders)
			if err != nil {
				return values.NewErrorWithMessage(err.Error()), nil
			}
			return buildResponse(ctx.TypeCtx, statusCode, respHeaders, respBodyStream), nil
		})

	// Default lambdas for Response header position params (return "LEADING").
	leading := values.BalValue("LEADING")
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Response.hasHeader$default$1",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return leading, nil })
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Response.getHeader$default$1",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return leading, nil })
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Response.getHeaders$default$1",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return leading, nil })
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Response.getHeaderNames$default$0",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return leading, nil })
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Response.addHeader$default$2",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return leading, nil })
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Response.removeHeader$default$1",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return leading, nil })
	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Response.removeAllHeaders$default$0",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) { return leading, nil })

	// Response class def — registers `new http:Response()` constructor support.
	responseClassDef := &bir.BIRClassDef{
		Name:      model.Name("Response"),
		LookupKey: "ballerina/http:Response",
		Fields: []bir.ObjectField{
			{Name: "statusCode", Ty: semtypes.INT},
		},
		VTable: map[string]*bir.BIRFunction{
			"initNative":       {FunctionLookupKey: "ballerina/http:Response.initNative"},
			"setTextPayload":   {FunctionLookupKey: "ballerina/http:Response.setTextPayload"},
			"setJsonPayload":   {FunctionLookupKey: "ballerina/http:Response.setJsonPayload"},
			"setBinaryPayload": {FunctionLookupKey: "ballerina/http:Response.setBinaryPayload"},
			"setHeader":        {FunctionLookupKey: "ballerina/http:Response.setHeader"},
			"addHeader":        {FunctionLookupKey: "ballerina/http:Response.addHeader"},
			"removeHeader":     {FunctionLookupKey: "ballerina/http:Response.removeHeader"},
			"removeAllHeaders": {FunctionLookupKey: "ballerina/http:Response.removeAllHeaders"},
			"setContentType":   {FunctionLookupKey: "ballerina/http:Response.setContentType"},
			"getContentType":   {FunctionLookupKey: "ballerina/http:Response.getContentType"},
			"getTextPayload":   {FunctionLookupKey: "ballerina/http:Response.getTextPayload"},
			"getJsonPayload":   {FunctionLookupKey: "ballerina/http:Response.getJsonPayload"},
			"getBinaryPayload": {FunctionLookupKey: "ballerina/http:Response.getBinaryPayload"},
			"hasHeader":        {FunctionLookupKey: "ballerina/http:Response.hasHeader"},
			"getHeader":        {FunctionLookupKey: "ballerina/http:Response.getHeader"},
			"getHeaders":       {FunctionLookupKey: "ballerina/http:Response.getHeaders"},
			"getHeaderNames":   {FunctionLookupKey: "ballerina/http:Response.getHeaderNames"},
		},
	}
	runtime.RegisterExternClassDef(rt, responseClassDef)

	// Response write methods.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.initNative",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			self.Put("statusCode", int64(200))
			self.Put("$headers", newMappingValue(ctx.TypeCtx))
			self.Put("body", &responseBodyHolder{buf: []byte{}})
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.setTextPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			self.Put("body", &responseBodyHolder{buf: []byte(args[1].(string))})
			ct := "text/plain"
			if len(args) > 2 {
				if s, ok := args[2].(string); ok && s != "" {
					ct = s
				}
			}
			responseHeaders(self).Put(ctx.TypeCtx, "content-type", newListValue(ctx.TypeCtx, []values.BalValue{ct}))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.setJsonPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			b, err := toJSONBytes(args[1])
			if err != nil {
				return values.NewErrorWithMessage("setJsonPayload: " + err.Error()), nil
			}
			self.Put("body", &responseBodyHolder{buf: b})
			ct := "application/json"
			if len(args) > 2 {
				if s, ok := args[2].(string); ok && s != "" {
					ct = s
				}
			}
			responseHeaders(self).Put(ctx.TypeCtx, "content-type", newListValue(ctx.TypeCtx, []values.BalValue{ct}))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.setBinaryPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			list, ok := args[1].(*values.List)
			if !ok {
				return values.NewErrorWithMessage("setBinaryPayload: expected byte[]"), nil
			}
			b, ok := listToBytes(list)
			if !ok {
				return values.NewErrorWithMessage("setBinaryPayload: invalid byte value"), nil
			}
			self.Put("body", &responseBodyHolder{buf: b})
			ct := "application/octet-stream"
			if len(args) > 2 {
				if s, ok := args[2].(string); ok && s != "" {
					ct = s
				}
			}
			responseHeaders(self).Put(ctx.TypeCtx, "content-type", newListValue(ctx.TypeCtx, []values.BalValue{ct}))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.setHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			val := args[2].(string)
			headers := responseHeaders(self)
			headers.Put(ctx.TypeCtx, name, newListValue(ctx.TypeCtx, []values.BalValue{val}))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.addHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			val := args[2].(string)
			// args[3] is position — always treat as LEADING
			headers := responseHeaders(self)
			existing, ok := headers.Get(name)
			if ok {
				existing.(*values.List).Append(ctx.TypeCtx, val)
			} else {
				headers.Put(ctx.TypeCtx, name, newListValue(ctx.TypeCtx, []values.BalValue{val}))
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.removeHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			// args[2] is position — always treat as LEADING
			responseHeaders(self).Delete(ctx.TypeCtx, name)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.removeAllHeaders",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			// args[1] is position — always treat as LEADING
			for _, k := range responseHeaders(self).Keys() {
				responseHeaders(self).Delete(ctx.TypeCtx, k)
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.setContentType",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			ct := args[1].(string)
			responseHeaders(self).Put(ctx.TypeCtx, "content-type", newListValue(ctx.TypeCtx, []values.BalValue{ct}))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.getContentType",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			v, ok := responseHeaders(self).Get("content-type")
			if !ok {
				return "", nil
			}
			list := v.(*values.List)
			if list.Len() == 0 {
				return "", nil
			}
			if s, ok := list.Get(0).(string); ok {
				return s, nil
			}
			return "", nil
		})

	// Response read methods.
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.getTextPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			bodyVal, _ := self.Get("body")
			if holder, ok := bodyVal.(*responseBodyHolder); ok {
				buf, err := holder.materialize()
				if err != nil {
					return values.NewErrorWithMessage(err.Error()), nil
				}
				return string(buf), nil
			}
			// fallback: plain string (should not occur in normal flow)
			if s, ok := bodyVal.(string); ok {
				return s, nil
			}
			return "", nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.getJsonPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			bodyVal, _ := self.Get("body")
			var body []byte
			if holder, ok := bodyVal.(*responseBodyHolder); ok {
				var err error
				body, err = holder.materialize()
				if err != nil {
					return values.NewErrorWithMessage(err.Error()), nil
				}
			} else if s, ok := bodyVal.(string); ok {
				body = []byte(s)
			}
			dec := json.NewDecoder(bytes.NewReader(body))
			dec.UseNumber()
			var v interface{}
			if err := dec.Decode(&v); err != nil {
				return values.NewErrorWithMessage("failed to parse JSON payload: " + err.Error()), nil
			}
			return values.GoToBalValue(ctx.TypeCtx, v, types.jsonListTy, types.jsonMapTy), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.getBinaryPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			bodyVal, _ := self.Get("body")
			var raw []byte
			if holder, ok := bodyVal.(*responseBodyHolder); ok {
				var err error
				raw, err = holder.materialize()
				if err != nil {
					return values.NewErrorWithMessage(err.Error()), nil
				}
			} else if s, ok := bodyVal.(string); ok {
				raw = []byte(s)
			}
			items := make([]values.BalValue, len(raw))
			for i, b := range raw {
				items[i] = int64(b)
			}
			return newTypedListValue(ctx.TypeCtx, types.byteArrTy, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.hasHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			// args[2] is position — ignored
			_, ok := responseHeaders(self).Get(name)
			return ok, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.getHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			v, ok := responseHeaders(self).Get(name)
			if !ok {
				return values.NewErrorWithMessage("header not found: " + name), nil
			}
			list := v.(*values.List)
			if list.Len() == 0 {
				return values.NewErrorWithMessage("header has no values: " + name), nil
			}
			return list.Get(0), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.getHeaders",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			v, ok := responseHeaders(self).Get(name)
			if !ok {
				return values.NewErrorWithMessage("header not found: " + name), nil
			}
			return v.(*values.List), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Response.getHeaderNames",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			keys := responseHeaders(self).Keys()
			items := make([]values.BalValue, len(keys))
			for i, k := range keys {
				items[i] = k
			}
			return newTypedListValue(ctx.TypeCtx, types.strArrTy, items), nil
		})

	// Request class def.
	requestClassDef := &bir.BIRClassDef{
		Name:      model.Name("Request"),
		LookupKey: "ballerina/http:Request",
		Fields: []bir.ObjectField{
			{Name: "rawPath", Ty: semtypes.STRING},
			{Name: "method", Ty: semtypes.STRING},
			{Name: "httpVersion", Ty: semtypes.STRING},
		},
		VTable: map[string]*bir.BIRFunction{
			"initNative":          {FunctionLookupKey: "ballerina/http:Request.initNative"},
			"setTextPayload":      {FunctionLookupKey: "ballerina/http:Request.setTextPayload"},
			"setJsonPayload":      {FunctionLookupKey: "ballerina/http:Request.setJsonPayload"},
			"setBinaryPayload":    {FunctionLookupKey: "ballerina/http:Request.setBinaryPayload"},
			"setHeader":           {FunctionLookupKey: "ballerina/http:Request.setHeader"},
			"addHeader":           {FunctionLookupKey: "ballerina/http:Request.addHeader"},
			"removeHeader":        {FunctionLookupKey: "ballerina/http:Request.removeHeader"},
			"removeAllHeaders":    {FunctionLookupKey: "ballerina/http:Request.removeAllHeaders"},
			"getHeaderNames":      {FunctionLookupKey: "ballerina/http:Request.getHeaderNames"},
			"setContentType":      {FunctionLookupKey: "ballerina/http:Request.setContentType"},
			"getContentType":      {FunctionLookupKey: "ballerina/http:Request.getContentType"},
			"getTextPayload":      {FunctionLookupKey: "ballerina/http:Request.getTextPayload"},
			"getJsonPayload":      {FunctionLookupKey: "ballerina/http:Request.getJsonPayload"},
			"getBinaryPayload":    {FunctionLookupKey: "ballerina/http:Request.getBinaryPayload"},
			"getHeader":           {FunctionLookupKey: "ballerina/http:Request.getHeader"},
			"getHeaders":          {FunctionLookupKey: "ballerina/http:Request.getHeaders"},
			"hasHeader":           {FunctionLookupKey: "ballerina/http:Request.hasHeader"},
			"getQueryParams":      {FunctionLookupKey: "ballerina/http:Request.getQueryParams"},
			"getQueryParamValue":  {FunctionLookupKey: "ballerina/http:Request.getQueryParamValue"},
			"getQueryParamValues": {FunctionLookupKey: "ballerina/http:Request.getQueryParamValues"},
		},
	}
	runtime.RegisterExternClassDef(rt, requestClassDef)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.initNative",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			self.Put("$body", &requestBodyHolder{buf: []byte{}})
			self.Put("$headers", newMappingValue(ctx.TypeCtx))
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.setTextPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			payload, _ := args[1].(string)
			self.Put("$body", &requestBodyHolder{buf: []byte(payload)})
			ct := "text/plain"
			if len(args) > 2 {
				if s, ok := args[2].(string); ok && s != "" {
					ct = s
				}
			}
			setRequestHeader(self, "content-type", ct, ctx.TypeCtx)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.setJsonPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			b, err := toJSONBytes(args[1])
			if err != nil {
				return values.NewErrorWithMessage("setJsonPayload: " + err.Error()), nil
			}
			self.Put("$body", &requestBodyHolder{buf: b})
			ct := "application/json"
			if len(args) > 2 {
				if s, ok := args[2].(string); ok && s != "" {
					ct = s
				}
			}
			setRequestHeader(self, "content-type", ct, ctx.TypeCtx)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.setBinaryPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			list, ok := args[1].(*values.List)
			if !ok {
				return nil, nil
			}
			raw := make([]byte, list.Len())
			for i := range list.Len() {
				if b, ok := list.Get(i).(int64); ok {
					raw[i] = byte(b)
				}
			}
			self.Put("$body", &requestBodyHolder{buf: raw})
			ct := "application/octet-stream"
			if len(args) > 2 {
				if s, ok := args[2].(string); ok && s != "" {
					ct = s
				}
			}
			setRequestHeader(self, "content-type", ct, ctx.TypeCtx)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.setHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			val, _ := args[2].(string)
			setRequestHeader(self, name, val, ctx.TypeCtx)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.addHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			val, _ := args[2].(string)
			hdrsVal, ok := self.Get("$headers")
			var hdrs *values.Map
			if ok {
				hdrs, _ = hdrsVal.(*values.Map)
			}
			if hdrs == nil {
				hdrs = newMappingValue(ctx.TypeCtx)
				self.Put("$headers", hdrs)
			}
			existing, ok := hdrs.Get(name)
			if ok {
				existing.(*values.List).Append(ctx.TypeCtx, val)
			} else {
				hdrs.Put(ctx.TypeCtx, name, newListValue(ctx.TypeCtx, []values.BalValue{val}))
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.removeHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			hdrsVal, ok := self.Get("$headers")
			if !ok {
				return nil, nil
			}
			if hdrs, ok := hdrsVal.(*values.Map); ok {
				hdrs.Delete(ctx.TypeCtx, name)
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.removeAllHeaders",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			hdrsVal, ok := self.Get("$headers")
			if !ok {
				return nil, nil
			}
			if hdrs, ok := hdrsVal.(*values.Map); ok {
				for _, k := range hdrs.Keys() {
					hdrs.Delete(ctx.TypeCtx, k)
				}
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getHeaderNames",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			self := args[0].(*values.Object)
			hdrsVal, ok := self.Get("$headers")
			if !ok {
				return newTypedListValue(ctx.TypeCtx, types.strArrTy, nil), nil
			}
			hdrs, ok := hdrsVal.(*values.Map)
			if !ok {
				return newTypedListValue(ctx.TypeCtx, types.strArrTy, nil), nil
			}
			keys := hdrs.Keys()
			items := make([]values.BalValue, len(keys))
			for i, k := range keys {
				items[i] = k
			}
			return newTypedListValue(ctx.TypeCtx, types.strArrTy, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.setContentType",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			ct := args[1].(string)
			setRequestHeader(self, "content-type", ct, ctx.TypeCtx)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getContentType",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			hdrsVal, ok := self.Get("$headers")
			if !ok {
				return "", nil
			}
			hdrs, ok := hdrsVal.(*values.Map)
			if !ok {
				return "", nil
			}
			v, ok := hdrs.Get("content-type")
			if !ok {
				return "", nil
			}
			list := v.(*values.List)
			if list.Len() == 0 {
				return "", nil
			}
			if s, ok := list.Get(0).(string); ok {
				return s, nil
			}
			return "", nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getTextPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			bodyVal, _ := self.Get("$body")
			holder, _ := bodyVal.(*requestBodyHolder)
			if holder == nil {
				return "", nil
			}
			buf := holder.materialize()
			if holder.readErr != nil {
				return values.NewErrorWithMessage("failed to read request body: " + holder.readErr.Error()), nil
			}
			return string(buf), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getJsonPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			self := args[0].(*values.Object)
			bodyVal, _ := self.Get("$body")
			holder, _ := bodyVal.(*requestBodyHolder)
			var body []byte
			if holder != nil {
				body = holder.materialize()
				if holder.readErr != nil {
					return values.NewErrorWithMessage("failed to read request body: " + holder.readErr.Error()), nil
				}
			}
			dec := json.NewDecoder(bytes.NewReader(body))
			dec.UseNumber()
			var v interface{}
			if err := dec.Decode(&v); err != nil {
				return values.NewErrorWithMessage("getJsonPayload: " + err.Error()), nil
			}
			return values.GoToBalValue(ctx.TypeCtx, v, types.jsonListTy, types.jsonMapTy), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getBinaryPayload",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			self := args[0].(*values.Object)
			bodyVal, _ := self.Get("$body")
			holder, _ := bodyVal.(*requestBodyHolder)
			var raw []byte
			if holder != nil {
				raw = holder.materialize()
				if holder.readErr != nil {
					return values.NewErrorWithMessage("failed to read request body: " + holder.readErr.Error()), nil
				}
			}
			items := make([]values.BalValue, len(raw))
			for i, b := range raw {
				items[i] = int64(b)
			}
			return newTypedListValue(ctx.TypeCtx, types.byteArrTy, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			hdrsVal, _ := self.Get("$headers")
			hdrs, ok := hdrsVal.(*values.Map)
			if !ok {
				return values.NewErrorWithMessage("header not found: " + name), nil
			}
			v, ok := hdrs.Get(name)
			if !ok {
				return values.NewErrorWithMessage("header not found: " + name), nil
			}
			list := v.(*values.List)
			if list.Len() == 0 {
				return values.NewErrorWithMessage("header has no values: " + name), nil
			}
			return list.Get(0), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getHeaders",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			hdrsVal, _ := self.Get("$headers")
			hdrs, ok := hdrsVal.(*values.Map)
			if !ok {
				return values.NewErrorWithMessage("header not found: " + name), nil
			}
			v, ok := hdrs.Get(name)
			if !ok {
				return values.NewErrorWithMessage("header not found: " + name), nil
			}
			return v.(*values.List), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.hasHeader",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			name := strings.ToLower(args[1].(string))
			hdrsVal, _ := self.Get("$headers")
			hdrs, ok := hdrsVal.(*values.Map)
			if !ok {
				return false, nil
			}
			_, ok = hdrs.Get(name)
			return ok, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getQueryParams",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			self := args[0].(*values.Object)
			queryStrVal, _ := self.Get("$queryStr")
			queryStr, _ := queryStrVal.(string)
			parsed, _ := url.ParseQuery(queryStr)
			m := newMappingValue(ctx.TypeCtx)
			for k, vals := range parsed {
				items := make([]values.BalValue, len(vals))
				for i, v := range vals {
					items[i] = v
				}
				m.Put(ctx.TypeCtx, k, newTypedListValue(ctx.TypeCtx, types.strArrTy, items))
			}
			return m, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getQueryParamValue",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			paramName := args[1].(string)
			queryStrVal, _ := self.Get("$queryStr")
			queryStr, _ := queryStrVal.(string)
			parsed, _ := url.ParseQuery(queryStr)
			vals := parsed[paramName]
			if len(vals) == 0 {
				return nil, nil
			}
			return vals[0], nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Request.getQueryParamValues",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			self := args[0].(*values.Object)
			key := args[1].(string)
			queryStrVal, _ := self.Get("$queryStr")
			queryStr, _ := queryStrVal.(string)
			parsed, _ := url.ParseQuery(queryStr)
			vals := parsed[key]
			if len(vals) == 0 {
				return nil, nil
			}
			items := make([]values.BalValue, len(vals))
			for i, v := range vals {
				items[i] = v
			}
			return newTypedListValue(ctx.TypeCtx, types.strArrTy, items), nil
		})
}

// splitOutsideQuotes splits s on every occurrence of sep that is not inside a
// double-quoted string (RFC 7230 §3.2.6 quoted-string), honouring backslash escapes.
func splitOutsideQuotes(s string, sep byte) []string {
	var out []string
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && inQuote && i+1 < len(s):
			i++ // skip the escaped character
		case c == '"':
			inQuote = !inQuote
		case c == sep && !inQuote:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

func parseHeader(tc semtypes.Context, input string) (*values.List, error) {
	segments := splitOutsideQuotes(input, ',')
	list := newListValue(tc, nil)
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return nil, fmt.Errorf("invalid header value: empty segment")
		}
		parts := splitOutsideQuotes(seg, ';')
		headerVal := strings.TrimSpace(parts[0])
		if headerVal == "" {
			return nil, fmt.Errorf("invalid header value: missing value before parameters")
		}
		params := newMappingValue(tc)
		for _, param := range parts[1:] {
			param = strings.TrimSpace(param)
			if param == "" {
				continue
			}
			eqIdx := strings.IndexByte(param, '=')
			if eqIdx < 0 {
				params.Put(tc, strings.ToLower(param), "")
				continue
			}
			key := strings.ToLower(strings.TrimSpace(param[:eqIdx]))
			val := strings.TrimSpace(param[eqIdx+1:])
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = val[1 : len(val)-1]
			}
			params.Put(tc, key, val)
		}
		entry := newMappingValue(tc)
		entry.Put(tc, "value", headerVal)
		entry.Put(tc, "params", params)
		list.Append(tc, entry)
	}
	return list, nil
}

func decimalToDuration(d *decimal.Decimal) time.Duration {
	return time.Duration(d.Float64() * float64(time.Second))
}

// applyCompressionHeaders mutates headers according to the Ballerina compression mode, mirroring
// jBallerina's AbstractHTTPAction compression logic:
//
//   - ALWAYS: add "Accept-Encoding: deflate, gzip" if not already present.
//   - NEVER:  remove any existing "Accept-Encoding" header.
//   - AUTO (default): leave headers untouched — the server decides.
func applyCompressionHeaders(mode string, headers map[string][]string) map[string][]string {
	switch mode {
	case "ALWAYS":
		for k := range headers {
			if strings.EqualFold(k, "accept-encoding") {
				return headers // caller already set Accept-Encoding; don't override
			}
		}
		if headers == nil {
			headers = make(map[string][]string)
		}
		headers["accept-encoding"] = []string{"deflate, gzip"}
	case "NEVER":
		for k := range headers {
			if strings.EqualFold(k, "accept-encoding") {
				delete(headers, k)
				break
			}
		}
	}
	return headers
}

// decompressResponseBody wraps body with a gzip or deflate reader when the server
// returns a Content-Encoding header. The Content-Encoding entry is deleted from
// headers so callers see the decoded body without an encoding marker, matching the
// transparent decompression behaviour of jBallerina's Netty pipeline.
func decompressResponseBody(headers map[string][]string, body io.ReadCloser) io.ReadCloser {
	for k, vals := range headers {
		if !strings.EqualFold(k, "content-encoding") || len(vals) == 0 {
			continue
		}
		enc := strings.ToLower(strings.TrimSpace(vals[0]))
		switch enc {
		case "gzip":
			delete(headers, k)
			return &gzipReadCloser{underlying: body}
		case "deflate":
			delete(headers, k)
			return &deflateReadCloser{reader: flate.NewReader(body), underlying: body}
		}
		break
	}
	return body
}

// gzipReadCloser lazily creates the gzip reader on first Read. Deferring
// gzip.NewReader (which parses the header eagerly) means a malformed gzip stream
// surfaces as a read error to the payload getters — via responseBodyHolder.readErr
// — instead of being silently passed through as raw compressed bytes. This mirrors
// the deflate path and jBallerina's lazy Netty decompression.
type gzipReadCloser struct {
	reader     *gzip.Reader
	underlying io.ReadCloser
	started    bool
	initErr    error
}

func (g *gzipReadCloser) Read(p []byte) (int, error) {
	if !g.started {
		g.started = true
		gr, err := gzip.NewReader(g.underlying)
		if err != nil {
			g.initErr = err
			return 0, err
		}
		g.reader = gr
	}
	if g.initErr != nil {
		return 0, g.initErr
	}
	return g.reader.Read(p)
}

func (g *gzipReadCloser) Close() error {
	if g.reader != nil {
		_ = g.reader.Close()
	}
	return g.underlying.Close()
}

type deflateReadCloser struct {
	reader     io.ReadCloser
	underlying io.ReadCloser
}

func (d *deflateReadCloser) Read(p []byte) (int, error) { return d.reader.Read(p) }
func (d *deflateReadCloser) Close() error {
	_ = d.reader.Close()
	return d.underlying.Close()
}

// extractHeaders converts a Ballerina map<string|string[]>? value to Go request headers.
func extractHeaders(arg values.BalValue) map[string][]string {
	if arg == nil {
		return nil
	}
	hdrMap, ok := arg.(*values.Map)
	if !ok {
		return nil
	}
	result := make(map[string][]string, hdrMap.Len())
	for _, key := range hdrMap.Keys() {
		val, _ := hdrMap.Get(key)
		switch v := val.(type) {
		case string:
			result[key] = []string{v}
		case *values.List:
			strs := make([]string, v.Len())
			for i := range v.Len() {
				if s, ok := v.Get(i).(string); ok {
					strs[i] = s
				}
			}
			result[key] = strs
		}
	}
	return result
}

// buildResponse constructs a Ballerina Response object from HTTP response data.
// All header values are stored as *values.List under the internal "$headers" key.
// Content-Encoding (gzip/deflate) is transparently decoded, mirroring jBallerina's
// Netty pipeline behaviour; the Content-Encoding header is removed from the response.
func buildResponse(tc semtypes.Context, statusCode int, respHeaders map[string][]string, bodyStream io.ReadCloser) *values.Object {
	bodyStream = decompressResponseBody(respHeaders, bodyStream)
	headersMap := newMappingValue(tc)
	for k, vals := range respHeaders {
		items := make([]values.BalValue, len(vals))
		for i, v := range vals {
			items[i] = v
		}
		headersMap.Put(tc, strings.ToLower(k), newListValue(tc, items))
	}
	holder := eagerBufferResponse(respHeaders, bodyStream)
	return values.NewObject(
		semtypes.OBJECT,
		map[string]values.BalValue{
			"statusCode": int64(statusCode),
			"$headers":   headersMap,
			"body":       holder,
		},
		map[string]string{
			"getTextPayload":   "ballerina/http:Response.getTextPayload",
			"getJsonPayload":   "ballerina/http:Response.getJsonPayload",
			"getBinaryPayload": "ballerina/http:Response.getBinaryPayload",
			"hasHeader":        "ballerina/http:Response.hasHeader",
			"getHeader":        "ballerina/http:Response.getHeader",
			"getHeaders":       "ballerina/http:Response.getHeaders",
			"getHeaderNames":   "ballerina/http:Response.getHeaderNames",
		},
		nil,
	)
}

// responseHeaders returns the internal header map stored on a Response object.
func responseHeaders(self *values.Object) *values.Map {
	h, _ := self.Get("$headers")
	return h.(*values.Map)
}

// listToBytes converts a Ballerina byte[] (List of int64 in 0–255) to []byte.
func listToBytes(list *values.List) ([]byte, bool) {
	b := make([]byte, list.Len())
	for i := range list.Len() {
		n, ok := list.Get(i).(int64)
		if !ok || n < 0 || n > 255 {
			return nil, false
		}
		b[i] = byte(n)
	}
	return b, true
}

// toJSONBytes serializes a Ballerina value to JSON bytes.
func toJSONBytes(v values.BalValue) ([]byte, error) {
	return values.ToJSONByteArray(v)
}
