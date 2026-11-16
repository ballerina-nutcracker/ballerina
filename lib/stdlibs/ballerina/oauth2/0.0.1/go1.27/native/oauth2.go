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

// Package native implements the extern functions of ballerina/oauth2.
package native

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "oauth2"
)

// restrictedHeaders are the headers Java's HTTP client refuses to set; jBallerina panics on them
// (ballerina-platform/ballerina-library#9715), so the port reports them as errors.
var restrictedHeaders = map[string]bool{"connection": true, "content-length": true, "expect": true, "host": true, "upgrade": true}

type oauth2Types struct {
	jsonListTy semtypes.SemType
	jsonMapTy  semtypes.SemType
}

func initOAuth2Module(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	jsonTy := semtypes.CreateJSON(semtypes.ContextFrom(env))
	jsonListDef := semtypes.NewListDefinition()
	jsonMapDef := semtypes.NewMappingDefinition()
	types := oauth2Types{
		jsonListTy: jsonListDef.Define(env, nil, semtypes.ListRest(jsonTy)),
		jsonMapTy:  jsonMapDef.Define(env, nil, jsonTy),
	}
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externDoHttpRequest", doHTTPRequest(rt))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromJsonString", fromJSONString(types))
}

func init() {
	runtime.RegisterModuleInitializer(initOAuth2Module)
}

// doHTTPRequest posts a form-encoded payload as jBallerina's OAuth2Client does and returns the body of a 2xx response.
func doHTTPRequest(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		rawURL := args[0].(string)
		clientConfig := args[1].(*values.Map)
		payload := args[3].(string)
		if customPayload, ok := clientConfig.Get("customPayload"); ok {
			payload += "&" + customPayload.(string)
		}
		headers := map[string][]string{}
		for _, m := range []*values.Map{args[2].(*values.Map), mapField(clientConfig, "customHeaders")} {
			if m == nil {
				continue
			}
			for _, name := range m.Keys() {
				if restrictedHeaders[strings.ToLower(name)] {
					return errorValue("Failed to send the request to the endpoint. restricted header name: \"" + name + "\""), nil
				}
				value, _ := m.Get(name)
				headers[name] = append(headers[name], value.(string))
			}
		}

		connectTimeout := args[4].(*decimal.Decimal).Float64()
		if connectTimeout <= 0 {
			return errorValue("OAuth2 connection timeout must be greater than zero"), nil
		}
		reqTimeout := args[5].(*decimal.Decimal).Float64()
		if reqTimeout <= 0 {
			return errorValue("OAuth2 request timeout must be greater than zero"), nil
		}
		secureSocket := mapField(clientConfig, "secureSocket")
		target, insecureScheme := endpointURL(rawURL, secureSocket != nil)
		if _, err := url.ParseRequestURI(target); err != nil || strings.ContainsAny(target, " \t\r\n\"<>\\^`{|}") {
			return errorValue("Failed to create URI for the provided value \"" + rawURL + "\"."), nil
		}
		if insecureScheme {
			_, _ = rt.Platform().IO.Stderr([]byte("warning: [ballerina/oauth2] HTTPS is recommended but using HTTP\n"))
		}

		// httpVersion is not applied: Java's client falls back to HTTP/1.1 when the server lacks HTTP/2, but the
		// platform's HTTP/2 mode has no fallback, so the single token request always uses HTTP/1.1.
		cfg := pal.ClientConfig{
			Timeout:        time.Duration(reqTimeout * float64(time.Second)),
			HTTPVersion:    "1.1",
			Pool:           pal.PoolConfig{DialTimeout: time.Duration(connectTimeout * float64(time.Second))},
			ResponseLimits: pal.ResponseLimitConfig{MaxEntityBodySize: -1},
		}
		if secureSocket != nil {
			tlsCfg, failure := tlsConfig(rt, secureSocket)
			if failure != "" {
				return errorValue("Failed to init SSL context. " + failure), nil
			}
			cfg.TLS = tlsCfg
		}
		client, err := rt.Platform().HTTP.NewClient(cfg)
		if err != nil {
			return errorValue("Failed to init SSL context. " + err.Error()), nil
		}
		status, _, body, err := client.Execute(context.Background(), "POST", target, strings.NewReader(payload),
			int64(len(payload)), "application/x-www-form-urlencoded", headers)
		if err != nil {
			return errorValue("Failed to send the request to the endpoint. " + err.Error()), nil
		}
		defer func() { _ = body.Close() }()
		response, err := io.ReadAll(body)
		if err != nil {
			return errorValue("Failed to send the request to the endpoint. " + err.Error()), nil
		}
		if status < 200 || status >= 300 {
			return errorValue("Failed to get a success response from the endpoint. Response code: '" +
				strconv.Itoa(status) + "'. Response body: '" + string(response) + "'"), nil
		}
		return string(response), nil
	}
}

func errorValue(message string) *values.Error {
	return values.NewErrorWithMessage(message)
}

func mapField(m *values.Map, name string) *values.Map {
	v, _ := m.Get(name)
	field, _ := v.(*values.Map)
	return field
}

// endpointURL adds the scheme a URL without one implies and collapses repeated slashes after it, as jBallerina does.
// It also reports whether an http URL is used with a secure socket.
func endpointURL(raw string, secure bool) (string, bool) {
	scheme, rest, found := strings.Cut(raw, "://")
	if !found {
		rest = raw
		scheme = "http"
		if secure {
			scheme = "https"
		}
	}
	for strings.Contains(rest, "//") {
		rest = strings.ReplaceAll(rest, "//", "/")
	}
	return scheme + "://" + rest, found && scheme == "http" && secure
}

// tlsConfig maps oauth2:SecureSocket to the platform TLS settings. Trust and key stores need PKCS#12 decoding
// shared across stdlibs (#1338), so only certificate and key files are supported. A non-empty failure describes
// why the configuration can't be used.
func tlsConfig(rt *runtime.Runtime, secureSocket *values.Map) (cfg pal.TLSConfig, failure string) {
	if disable, _ := secureSocket.Get("disable"); disable == true {
		return pal.TLSConfig{InsecureSkipVerify: true}, ""
	}
	cert, ok := secureSocket.Get("cert")
	if !ok {
		return pal.TLSConfig{}, "Need to configure 'crypto:TrustStore' or 'cert' with client SSL certificates file."
	}
	certFile, ok := cert.(string)
	if !ok {
		return pal.TLSConfig{}, "crypto:TrustStore is not yet supported for secureSocket.cert; use a certificate file path instead"
	}
	fs := rt.Platform().FS
	caPEM, err := readCertificate(fs, certFile)
	if err != nil {
		return pal.TLSConfig{}, "Failed to get the public key from Crypto API. " + err.Error()
	}
	tlsCfg := pal.TLSConfig{CACertPEM: caPEM}
	certKey := mapField(secureSocket, "key")
	if certKey == nil {
		return tlsCfg, ""
	}
	keyFile, isCertKey := certKey.Get("keyFile")
	if !isCertKey {
		return pal.TLSConfig{}, "crypto:KeyStore is not yet supported for secureSocket.key; use an oauth2:CertKey instead"
	}
	if _, encrypted := certKey.Get("keyPassword"); encrypted {
		return pal.TLSConfig{}, "an encrypted private key is not yet supported for secureSocket.key"
	}
	clientCertFile, _ := certKey.Get("certFile")
	if tlsCfg.ClientCertPEM, err = readCertificate(fs, clientCertFile.(string)); err != nil {
		return pal.TLSConfig{}, "Failed to get the public key from Crypto API. " + err.Error()
	}
	if tlsCfg.ClientKeyPEM, err = fs.ReadFile(keyFile.(string)); err != nil {
		return pal.TLSConfig{}, "Failed to get the private key from Crypto API. " + err.Error()
	}
	return tlsCfg, ""
}

// readCertificate reads a PEM or DER certificate file and returns it as PEM.
func readCertificate(fs pal.FS, path string) ([]byte, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if block, _ := pem.Decode(data); block == nil {
		data = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	}
	return data, nil
}

// fromJSONString parses JSON as lang.value:fromJsonString does: object members keep their order, integers
// become int and other numbers decimal.
func fromJSONString(types oauth2Types) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		dec := json.NewDecoder(strings.NewReader(args[0].(string)))
		dec.UseNumber()
		p := jsonParser{dec: dec, tc: ctx.TypeCtx(), types: types}
		v, err := p.value()
		if err == nil {
			if _, extra := dec.Token(); extra != io.EOF {
				err = errors.New("unexpected data after the JSON value")
			}
		}
		if err != nil {
			return errorValue(err.Error()), nil
		}
		return v, nil
	}
}

type jsonParser struct {
	dec   *json.Decoder
	tc    semtypes.Context
	types oauth2Types
}

func (p *jsonParser) value() (values.BalValue, error) {
	tok, err := p.dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			return p.object()
		}
		return p.array()
	case json.Number:
		return numberValue(t.String())
	default:
		return t, nil
	}
}

func (p *jsonParser) object() (values.BalValue, error) {
	m := values.NewMap(p.types.jsonMapTy, semtypes.ToMappingAtomicType(p.tc, p.types.jsonMapTy), false, nil)
	for p.dec.More() {
		keyTok, err := p.dec.Token()
		if err != nil {
			return nil, err
		}
		item, err := p.value()
		if err != nil {
			return nil, err
		}
		m.Put(p.tc, keyTok.(string), item)
	}
	if _, err := p.dec.Token(); err != nil {
		return nil, err
	}
	return m, nil
}

func (p *jsonParser) array() (values.BalValue, error) {
	var items []values.BalValue
	for p.dec.More() {
		item, err := p.value()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if _, err := p.dec.Token(); err != nil {
		return nil, err
	}
	return values.NewList(p.types.jsonListTy, semtypes.ToListAtomicType(p.tc.Env(), p.types.jsonListTy), false, nil, 0, items), nil
}

func numberValue(s string) (values.BalValue, error) {
	if !strings.ContainsAny(s, ".eE") {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, errors.New("number '" + s + "' is out of the int range")
		}
		return i, nil
	}
	d, derr := decimal.FromString(s)
	if derr != nil {
		return nil, errors.New("number '" + s + "' is out of the decimal range")
	}
	return d, nil
}
