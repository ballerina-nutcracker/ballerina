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
	"encoding/pem"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// jBallerina's JWKS client has no timeouts and can block forever (ballerina-platform/ballerina-library#9713);
// these match the defaults ballerina/oauth2 uses for its endpoint calls.
const (
	jwksConnectTimeout = 15 * time.Second
	jwksRequestTimeout = 30 * time.Second
)

func getJwksResponse(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		clientConfig := args[1].(*values.Map)
		var secureSocket *values.Map
		if v, ok := clientConfig.Get("secureSocket"); ok {
			secureSocket, _ = v.(*values.Map)
		}
		target, insecureScheme := jwksURL(args[0].(string), secureSocket != nil)
		if insecureScheme {
			_, _ = rt.Platform().IO.Stderr([]byte("warning: [ballerina/jwt] HTTPS is recommended but using HTTP\n"))
		}
		// jBallerina panics on a URL it can't parse (ballerina-platform/ballerina-library#9714).
		if _, err := url.ParseRequestURI(target); err != nil || strings.ContainsAny(target, " \t\r\n\"<>\\^`{|}") {
			return values.NewErrorWithMessage("Failed to create URI for the provided value \"" + args[0].(string) + "\"."), nil
		}

		// httpVersion is not applied: Java's client falls back to HTTP/1.1 when the server lacks HTTP/2, but the
		// platform's HTTP/2 mode has no fallback, so the single JWKS GET always uses HTTP/1.1.
		cfg := pal.ClientConfig{
			Timeout:        jwksRequestTimeout,
			HTTPVersion:    "1.1",
			Pool:           pal.PoolConfig{DialTimeout: jwksConnectTimeout},
			ResponseLimits: pal.ResponseLimitConfig{MaxEntityBodySize: -1},
		}
		if secureSocket != nil {
			tlsCfg, failure := jwksTLSConfig(rt, secureSocket)
			if failure != "" {
				return values.NewErrorWithMessage("Failed to init SSL context. " + failure), nil
			}
			cfg.TLS = tlsCfg
		}
		client, err := rt.Platform().HTTP.NewClient(cfg)
		if err != nil {
			return values.NewErrorWithMessage("Failed to init SSL context. " + err.Error()), nil
		}
		status, _, body, err := client.Execute(context.Background(), "GET", target, nil, 0, "", nil)
		if err != nil {
			return values.NewErrorWithMessage("Failed to send the request to the endpoint. " + err.Error()), nil
		}
		defer func() { _ = body.Close() }()
		payload, err := io.ReadAll(body)
		if err != nil {
			return values.NewErrorWithMessage("Failed to send the request to the endpoint. " + err.Error()), nil
		}
		if status != 200 {
			return values.NewErrorWithMessage("Failed to get a success response from the endpoint. Response code: '" +
				strconv.Itoa(status) + "'. Response body: '" + string(payload) + "'"), nil
		}
		return string(payload), nil
	}
}

// jwksURL adds the scheme a URL without one implies and collapses repeated slashes after it, as jBallerina does.
// It also reports whether an http URL is used with a secure socket.
func jwksURL(raw string, secure bool) (string, bool) {
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

// jwksTLSConfig maps jwt:SecureSocket to the platform TLS settings. Trust and key stores need PKCS#12 decoding
// shared across stdlibs (#1338), so only certificate and key files are supported. A non-empty failure describes
// why the configuration can't be used.
func jwksTLSConfig(rt *runtime.Runtime, secureSocket *values.Map) (cfg pal.TLSConfig, failure string) {
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
	key, ok := secureSocket.Get("key")
	if !ok {
		return tlsCfg, ""
	}
	certKey := key.(*values.Map)
	keyFile, isCertKey := certKey.Get("keyFile")
	if !isCertKey {
		return pal.TLSConfig{}, "crypto:KeyStore is not yet supported for secureSocket.key; use a jwt:CertKey instead"
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
