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

// Native-CLI implementation of the pal.HTTP.Listen contract: a net/http-backed
// HTTP listener that binds a real TCP socket. NewPlatform (in pal.go) wires
// Listen into pal.HTTP.Listen. A WASM/web-editor platform supplies its own
// Listen that delivers requests from the JS host without binding a socket.

package palnative

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"ballerina-lang-go/platform/pal"
)

// Listen is the pal.HTTP.Listen factory for the native-CLI platform. It binds a
// TCP socket (optionally TLS-wrapped) and serves handler on a background
// goroutine, returning a handle for lifecycle control.
func Listen(cfg pal.ServerConfig, handler http.Handler) (pal.ServerHandle, error) {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	if cfg.HTTPVersion == "2.0" {
		protocols.SetHTTP2(true)
		if cfg.TLS == nil {
			protocols.SetUnencryptedHTTP2(true)
		}
	}

	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 60 * time.Second
	}

	var tlsCfg *tls.Config
	if cfg.TLS != nil {
		var err error
		tlsCfg, err = buildServerTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
	}

	server := &http.Server{
		Addr:      addr,
		Handler:   handler,
		Protocols: protocols,
		// ReadHeaderTimeout guards against slow-loris without aborting request bodies
		// mid-stream (ReadTimeout would do that and breaks proxies/uploads).
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      writeTimeout,
		// Evict idle HTTP/1.1 keep-alive connections that haven't been used.
		IdleTimeout: 300 * time.Second,
		// 16 KB max header size; the default 1 MB is wasteful for typical REST APIs.
		MaxHeaderBytes: 1 << 14,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	serveLn := ln
	if tlsCfg != nil {
		serveLn = tls.NewListener(ln, tlsCfg)
	}
	go func() {
		_ = server.Serve(serveLn)
	}()
	return &httpServerHandle{server: server}, nil
}

// httpServerHandle adapts *http.Server to pal.ServerHandle.
type httpServerHandle struct {
	server *http.Server
}

func (h *httpServerHandle) Shutdown(ctx context.Context) error { return h.server.Shutdown(ctx) }
func (h *httpServerHandle) Close() error                       { return h.server.Close() }

// buildServerTLSConfig assembles a *tls.Config from the pre-read PEM material
// and settings carried by pal.ServerTLSConfig.
func buildServerTLSConfig(c *pal.ServerTLSConfig) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(c.CertPEM, c.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("X509KeyPair: %w", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}} //nolint:gosec

	// mTLS: client certificate verification.
	if len(c.ClientCACertPEM) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(c.ClientCACertPEM) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.ClientCAs = pool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	tlsCfg.MinVersion = c.MinVersion
	tlsCfg.MaxVersion = c.MaxVersion
	if len(c.CipherSuiteNames) > 0 {
		if resolved := resolveCipherSuites(c.CipherSuiteNames); len(resolved) > 0 {
			tlsCfg.CipherSuites = resolved
		}
	}
	tlsCfg.SessionTicketsDisabled = c.DisableSessionTickets
	return tlsCfg, nil
}
