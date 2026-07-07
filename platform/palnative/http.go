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

// Native-CLI implementation of the pal.HTTP contract: a net/http-backed HTTP
// client and the TLS plumbing it needs. NewPlatform (in pal.go) wires
// NewHTTPClient into pal.HTTP.NewClient. Other PAL implementations
// (e.g. WASM/web-editor) would supply their own version of this file.

package palnative

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ballerina-lang-go/platform/pal"
)

type httpClient struct {
	client            *http.Client
	maxEntityBodySize int64 // -1 = no limit; 0 or positive = byte cap
}

// limitedReadCloser bounds reads to n bytes (like io.LimitReader) while still
// forwarding Close to the underlying stream when it is an io.Closer. The bound
// gives net/http's post-write drain read a clean EOF without touching the
// underlying stream; forwarding Close ensures an owned stream is not leaked.
type limitedReadCloser struct {
	lr   *io.LimitedReader
	body io.Reader
}

func newLimitedReadCloser(body io.Reader, n int64) *limitedReadCloser {
	return &limitedReadCloser{lr: &io.LimitedReader{R: body, N: n}, body: body}
}

func (l *limitedReadCloser) Read(p []byte) (int, error) { return l.lr.Read(p) }

func (l *limitedReadCloser) Close() error {
	if c, ok := l.body.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (c *httpClient) Execute(ctx context.Context, method, targetURL string, body io.Reader, contentLength int64, contentType string, reqHeaders map[string][]string) (int, map[string][]string, io.ReadCloser, error) {
	// When a streamed body has a declared content length, bound it with
	// io.LimitReader. After writing the declared bytes, net/http's
	// transferWriter.writeBody does a second drain read (into io.Discard) to verify
	// there are no extra bytes. For a passthrough that forwards the inbound request's
	// r.Body, that stream may already be closed by the time the drain read runs —
	// surfacing as "http: invalid Read on closed Body", which aborts the upstream
	// connection mid-response and drops the request (~0.05% under load, streamed
	// bodies only). LimitReader returns EOF after contentLength bytes without
	// touching the underlying stream, so the drain read always sees a clean EOF.
	//
	// In-memory bodies (*bytes.Reader/*bytes.Buffer/*strings.Reader) are left as-is:
	// they are not the affected case, and net/http recognises them to populate
	// Request.GetBody so the request stays replayable on a stale pooled connection.
	//
	// The wrapper preserves the underlying stream's Close: net/http's transport
	// closes the request body when it is an io.ReadCloser, so an owned stream is
	// released rather than leaked. For the current streamed caller (forward,
	// forwarding the server-owned inbound r.Body) this Close is redundant with
	// the server's own — http request bodies tolerate a repeat Close — so it is
	// safe either way.
	if body != nil && contentLength >= 0 {
		switch body.(type) {
		case *bytes.Reader, *bytes.Buffer, *strings.Reader:
			// replayable in-memory body — leave untouched
		default:
			body = newLimitedReadCloser(body, contentLength)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return 0, nil, nil, err
	}
	// -1 means unknown length (chunked); >=0 tells Go to use Content-Length framing.
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}
	// Set default User-Agent before caller headers so caller can override it if needed.
	req.Header.Set("User-Agent", "ballerina")
	for k, vals := range reqHeaders {
		if len(vals) == 0 {
			continue
		}
		req.Header.Set(k, vals[0])
		for _, v := range vals[1:] {
			req.Header.Add(k, v)
		}
	}
	// Apply contentType (derived from mediaType) after caller headers so it
	// always takes priority over any Content-Type supplied in reqHeaders.
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	respBody := io.ReadCloser(resp.Body)
	if c.maxEntityBodySize != -1 {
		// Check Content-Length first for an early error without reading the body.
		if resp.ContentLength > c.maxEntityBodySize {
			_ = resp.Body.Close()
			return 0, nil, nil, fmt.Errorf("response entity body size exceeds: %d bytes", c.maxEntityBodySize)
		}
		respBody = &limitedBodyReadCloser{rc: resp.Body, n: c.maxEntityBodySize, limit: c.maxEntityBodySize}
	}
	return resp.StatusCode, map[string][]string(resp.Header), respBody, nil
}

// limitedBodyReadCloser enforces a maximum response body size.
// Reads are capped to limit bytes; on the next Read after the limit is consumed
// it peeks one byte from the underlying stream. If more data exists the limit was
// exceeded and an error is returned; if the stream is at EOF the body was within
// the limit and io.EOF is returned normally.
type limitedBodyReadCloser struct {
	rc       io.ReadCloser
	n        int64 // bytes remaining before limit
	limit    int64 // original limit (for error message)
	exceeded bool
}

func (l *limitedBodyReadCloser) Read(p []byte) (int, error) {
	if l.exceeded {
		return 0, fmt.Errorf("response entity body size exceeds: %d bytes", l.limit)
	}
	if l.n <= 0 {
		var b [1]byte
		nn, _ := l.rc.Read(b[:])
		if nn > 0 {
			l.exceeded = true
			return 0, fmt.Errorf("response entity body size exceeds: %d bytes", l.limit)
		}
		return 0, io.EOF
	}
	if int64(len(p)) > l.n {
		p = p[:l.n]
	}
	n, err := l.rc.Read(p)
	l.n -= int64(n)
	return n, err
}

func (l *limitedBodyReadCloser) Close() error { return l.rc.Close() }

// NewHTTPClient is the pal.HTTP.NewClient factory for the native-CLI
// platform. It builds a *http.Client configured from cfg and wraps it so the
// runtime sees only the pal.HTTPClient interface.
func NewHTTPClient(cfg pal.ClientConfig) pal.HTTPClient {
	tlsConfig := buildTLSConfig(cfg.TLS)
	// Build a net.Dialer with a configurable connect timeout.
	// TCP keep-alive is disabled (KeepAlive:-1) to match jBallerina's default
	// socketConfig.keepAlive=false; HTTP-level connection reuse is handled by the Transport pool.
	dialer := &net.Dialer{
		Timeout:   poolDefault(cfg.Pool.DialTimeout, 15*time.Second),
		KeepAlive: -1,
	}
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		TLSClientConfig:     tlsConfig,
		TLSHandshakeTimeout: cfg.TLS.HandshakeTimeout,
		// Pool sizing — defaults mirror jBallerina's PoolConfiguration:
		//   maxIdleConnections=100, maxActiveConnections=-1 (unlimited),
		//   minEvictableIdleTime=300s.
		MaxIdleConns:          poolDefaultInt(cfg.Pool.MaxIdleConns, 512),
		MaxIdleConnsPerHost:   poolDefaultInt(cfg.Pool.MaxIdleConnsPerHost, 100),
		MaxConnsPerHost:       cfg.Pool.MaxConnsPerHost, // 0 = unlimited; matches jBallerina -1
		IdleConnTimeout:       poolDefault(cfg.Pool.IdleConnTimeout, 300*time.Second),
		ResponseHeaderTimeout: cfg.Pool.ResponseHeaderTimeout,
		WriteBufferSize:       poolDefaultInt(cfg.Pool.WriteBufferSize, 32*1024),
		ReadBufferSize:        poolDefaultInt(cfg.Pool.ReadBufferSize, 32*1024),
		DisableCompression:    cfg.Pool.DisableCompression,
		// Response header size limit (jBallerina default 8192, always set explicitly).
		MaxResponseHeaderBytes: cfg.ResponseLimits.MaxHeaderSize,
	}
	if cfg.Proxy.Host != "" {
		proxyURL := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", cfg.Proxy.Host, cfg.Proxy.Port),
		}
		if cfg.Proxy.UserName != "" {
			proxyURL.User = url.UserPassword(cfg.Proxy.UserName, cfg.Proxy.Password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	protocols := new(http.Protocols)
	if cfg.HTTPVersion == "2.0" {
		// HTTP/2 mode: enable h2 (TLS/ALPN) and h2c (cleartext prior-knowledge, RFC 7540 §3.4).
		// SetHTTP1 is intentionally omitted so Go uses h2c prior-knowledge for http:// connections
		// rather than falling back to HTTP/1.1. SetHTTP1 only governs unencrypted traffic,
		// so https:// connections retain their normal ALPN h2→http/1.1 fallback.
		protocols.SetHTTP2(true)
		protocols.SetUnencryptedHTTP2(true)
	} else {
		// HTTP/1.x mode: cleartext HTTP/1 only; no h2 ALPN for https:// either.
		protocols.SetHTTP1(true)
	}
	transport.Protocols = protocols
	c := &http.Client{Timeout: cfg.Timeout, Transport: transport}
	if !cfg.FollowRedirects.Enabled {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		maxCount := cfg.FollowRedirects.MaxCount
		if maxCount <= 0 {
			maxCount = 5 // Ballerina default
		}
		allowAuth := cfg.FollowRedirects.AllowAuthHeaders
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) > maxCount {
				return http.ErrUseLastResponse
			}
			if allowAuth && len(via) > 0 {
				if auth := via[0].Header.Get("Authorization"); auth != "" {
					req.Header.Set("Authorization", auth)
				}
				if proxy := via[0].Header.Get("Proxy-Authorization"); proxy != "" {
					req.Header.Set("Proxy-Authorization", proxy)
				}
			}
			return nil
		}
	}
	return &httpClient{client: c, maxEntityBodySize: cfg.ResponseLimits.MaxEntityBodySize}
}

// poolDefault returns d if non-zero, otherwise def.
func poolDefault(d, def time.Duration) time.Duration {
	if d != 0 {
		return d
	}
	return def
}

// poolDefaultInt returns n if non-zero, otherwise def.
func poolDefaultInt(n, def int) int {
	if n != 0 {
		return n
	}
	return def
}

// resolveCipherSuites maps IANA TLS 1.2 cipher suite names to Go uint16 IDs.
// Unknown names are silently skipped; TLS 1.3 ciphers are unaffected regardless.
func resolveCipherSuites(names []string) []uint16 {
	m := make(map[string]uint16, len(tls.CipherSuites())+len(tls.InsecureCipherSuites()))
	for _, c := range tls.CipherSuites() {
		m[c.Name] = c.ID
	}
	for _, c := range tls.InsecureCipherSuites() {
		m[c.Name] = c.ID
	}
	ids := make([]uint16, 0, len(names))
	for _, name := range names {
		if id, ok := m[name]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// tlsVerifyConnectionWithCNFallback returns a VerifyConnection callback that verifies the
// server's certificate chain against rootCAs and falls back to CN-based hostname matching
// when no SANs are present. Go 1.15+ disabled CN-only hostname verification (RFC 6125 §2.3),
// but many self-signed and Java-issued certificates still rely on it.
func tlsVerifyConnectionWithCNFallback(rootCAs *x509.CertPool) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		opts := x509.VerifyOptions{
			Roots:         rootCAs,
			Intermediates: x509.NewCertPool(),
		}
		for _, cert := range cs.PeerCertificates[1:] {
			opts.Intermediates.AddCert(cert)
		}
		if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
			return err
		}
		// cs.ServerName is the SNI hostname (no port). Try SAN-based verification first.
		// Only fall back to CN matching for certs that genuinely have no SANs — when SANs
		// are present but don't match, that is a real mismatch and must not be bypassed.
		leaf := cs.PeerCertificates[0]
		if err := leaf.VerifyHostname(cs.ServerName); err != nil {
			if len(leaf.DNSNames) > 0 || len(leaf.IPAddresses) > 0 {
				return err
			}
			return tlsMatchCN(leaf.Subject.CommonName, cs.ServerName)
		}
		return nil
	}
}

// tlsMatchCN checks whether pattern (a certificate CN) matches host.
// Supports simple wildcard patterns of the form "*.example.com".
func tlsMatchCN(pattern, host string) error {
	pattern = strings.ToLower(strings.TrimSuffix(pattern, "."))
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if pattern == host {
		return nil
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		if strings.HasSuffix(host, suffix) && strings.Count(host, ".") == strings.Count(suffix, ".") {
			return nil
		}
	}
	return fmt.Errorf("x509: certificate CN %q does not match host %q", pattern, host)
}
