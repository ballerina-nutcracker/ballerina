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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

type httpClient struct {
	client            *http.Client
	idleTimeout       time.Duration // 0 = none
	maxEntityBodySize int64         // -1 = no limit; 0 or positive = byte cap
}

// The messages jBallerina's client reports when its idle timeout fires.
var (
	errIdleBeforeResponse = errors.New("Idle timeout triggered before initiating inbound response")         //nolint:staticcheck // jBallerina's message verbatim
	errIdleReadingBody    = errors.New("Idle timeout triggered while reading inbound response entity body") //nolint:staticcheck // jBallerina's message verbatim
)

// idleTimer cancels a request once it has gone timeout without progress, with the
// cause set by the latest arm call.
type idleTimer struct {
	timeout time.Duration
	cancel  context.CancelCauseFunc
	mu      sync.Mutex
	timer   *time.Timer
	cause   error
}

func (t *idleTimer) arm(cause error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cause = cause
	if t.timer == nil {
		t.timer = time.AfterFunc(t.timeout, t.fire)
		return
	}
	t.timer.Reset(t.timeout)
}

func (t *idleTimer) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer != nil {
		t.timer.Stop()
	}
}

func (t *idleTimer) fire() {
	t.mu.Lock()
	cause := t.cause
	t.mu.Unlock()
	t.cancel(cause)
}

// progressReader restarts the idle timer whenever the transport reads more of a
// streamed request body, so a slow upload is only cut off when it stalls.
type progressReader struct {
	body  io.Reader
	timer *idleTimer
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.body.Read(b)
	if n > 0 {
		p.timer.arm(errIdleBeforeResponse)
	}
	return n, err
}

func (p *progressReader) Close() error {
	if c, ok := p.body.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// idleBody times each blocked Read of a response body, so the body fails once no data
// arrives for the idle timeout, however long the whole body takes.
type idleBody struct {
	rc     io.ReadCloser
	timer  *idleTimer
	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (b *idleBody) Read(p []byte) (int, error) {
	b.timer.arm(errIdleReadingBody)
	n, err := b.rc.Read(p)
	b.timer.stop()
	if err != nil && !errors.Is(err, io.EOF) && errors.Is(context.Cause(b.ctx), errIdleReadingBody) {
		err = errIdleReadingBody
	}
	return n, err
}

func (b *idleBody) Close() error {
	b.timer.stop()
	err := b.rc.Close()
	b.cancel(nil)
	return err
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
	// A bounded streamed body must return a clean EOF after contentLength bytes so
	// net/http's post-write drain read never touches the (possibly already-closed)
	// underlying stream; in-memory bodies are left untouched so GetBody still works.
	if body != nil && contentLength >= 0 {
		switch body.(type) {
		case *bytes.Reader, *bytes.Buffer, *strings.Reader:
			// replayable in-memory body — leave untouched
		default:
			body = newLimitedReadCloser(body, contentLength)
		}
	}
	var timer *idleTimer
	reqCtx := ctx
	cancel := context.CancelCauseFunc(func(error) {})
	if c.idleTimeout > 0 {
		reqCtx, cancel = context.WithCancelCause(ctx)
		timer = &idleTimer{timeout: c.idleTimeout, cancel: cancel}
		switch body.(type) {
		case nil, *bytes.Reader, *bytes.Buffer, *strings.Reader:
			// written at once, and must stay replayable for redirects
		default:
			body = &progressReader{body: body, timer: timer}
		}
		timer.arm(errIdleBeforeResponse)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, targetURL, body)
	if err != nil {
		cancel(nil)
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
	if timer != nil {
		timer.stop()
	}
	if err != nil {
		if errors.Is(context.Cause(reqCtx), errIdleBeforeResponse) {
			err = errIdleBeforeResponse
		}
		cancel(nil)
		return 0, nil, nil, err
	}
	respBody := io.ReadCloser(resp.Body)
	if timer != nil {
		respBody = &idleBody{rc: resp.Body, timer: timer, ctx: reqCtx, cancel: cancel}
	}
	if c.maxEntityBodySize != -1 {
		// Check Content-Length first for an early error without reading the body.
		if resp.ContentLength > c.maxEntityBodySize {
			_ = respBody.Close()
			return 0, nil, nil, fmt.Errorf("response entity body size exceeds: %d bytes", c.maxEntityBodySize)
		}
		respBody = &limitedBodyReadCloser{rc: respBody, n: c.maxEntityBodySize, limit: c.maxEntityBodySize}
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
func NewHTTPClient(cfg pal.ClientConfig) (pal.HTTPClient, error) {
	tlsConfig, err := buildTLSConfig(cfg.TLS)
	if err != nil {
		return nil, err
	}
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
	if tlsConfig.VerifyConnection != nil && !cfg.TLS.SkipHostnameVerification {
		// The shared callback can't see which host a pooled connection dials, so
		// non-proxied HTTPS connections rebind it to the dialed host here.
		// Cloning at dial time picks up the ALPN protocols the transport adds.
		transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, _ := net.SplitHostPort(addr)
			connTLSConfig := transport.TLSClientConfig.Clone()
			if connTLSConfig.ServerName == "" {
				connTLSConfig.ServerName = host
			}
			connTLSConfig.VerifyConnection = tlsVerifyConnectionWithCNFallback(connTLSConfig.RootCAs, connTLSConfig.ServerName)
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			return tlsClientHandshake(ctx, conn, connTLSConfig, cfg.TLS.HandshakeTimeout)
		}
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
	// cfg.Timeout is an idle timeout, as in jBallerina, enforced per request in Execute
	// rather than as http.Client.Timeout, which would also bound a long-lived body.
	c := &http.Client{Transport: transport}
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
	return &httpClient{client: c, idleTimeout: cfg.Timeout, maxEntityBodySize: cfg.ResponseLimits.MaxEntityBodySize}, nil
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

// tlsVerifyChain verifies the server's certificate chain against rootCAs (the
// system pool when nil) without checking the hostname.
func tlsVerifyChain(rootCAs *x509.CertPool, cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return fmt.Errorf("x509: server presented no certificates")
	}
	opts := x509.VerifyOptions{
		Roots:         rootCAs,
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range cs.PeerCertificates[1:] {
		opts.Intermediates.AddCert(cert)
	}
	_, err := cs.PeerCertificates[0].Verify(opts)
	return err
}

// tlsVerifyConnectionWithCNFallback returns a VerifyConnection callback that verifies the
// server's certificate chain against rootCAs and falls back to CN-based hostname matching
// when no SANs are present. Go 1.15+ disabled CN-only hostname verification (RFC 6125 §2.3),
// but many self-signed and Java-issued certificates still rely on it.
//
// The hostname is checked against expectedServerName when set, else cs.ServerName, which
// Go leaves empty for IP literals since it never sends SNI for them (RFC 6066). Callers
// that know the dialed host pass it as expectedServerName; the cs.ServerName fallback only
// serves proxied HTTPS, where http.Transport runs the handshake itself. An empty name is
// rejected: matching it would accept any trusted certificate with no SANs and an empty CN.
func tlsVerifyConnectionWithCNFallback(rootCAs *x509.CertPool, expectedServerName string) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		serverName := expectedServerName
		if serverName == "" {
			serverName = cs.ServerName
		}
		if serverName == "" {
			return fmt.Errorf("x509: cannot verify certificate hostname: no server name")
		}
		if err := tlsVerifyChain(rootCAs, cs); err != nil {
			return err
		}
		// Try SAN-based verification first. Only fall back to CN matching for
		// certs that genuinely have no SANs — when SANs are present but don't
		// match, that is a real mismatch and must not be bypassed.
		leaf := cs.PeerCertificates[0]
		if err := leaf.VerifyHostname(serverName); err != nil {
			if len(leaf.DNSNames) > 0 || len(leaf.IPAddresses) > 0 {
				return err
			}
			return tlsMatchCN(leaf.Subject.CommonName, serverName)
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
