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

package palnative

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

// ---------------------------------------------------------------------------
// tlsMatchCN
// ---------------------------------------------------------------------------

func TestTlsMatchCN(t *testing.T) {
	tests := []struct {
		pattern string
		host    string
		match   bool
		desc    string
	}{
		{"example.com", "example.com", true, "exact match"},
		{"example.com", "other.com", false, "different host"},
		{"example.com", "sub.example.com", false, "no wildcard on exact pattern"},
		{"*.example.com", "sub.example.com", true, "single-label wildcard match"},
		{"*.example.com", "example.com", false, "wildcard does not match apex"},
		{"*.example.com", "deep.sub.example.com", false, "wildcard matches only one label"},
		{"", "example.com", false, "empty pattern"},
		{"example.com", "", false, "empty host"},
		{"EXAMPLE.COM", "example.com", true, "case-insensitive exact"},
		{"*.EXAMPLE.COM", "sub.example.com", true, "case-insensitive wildcard"},
		{"example.com.", "example.com", true, "trailing dot stripped in pattern"},
		{"example.com", "example.com.", true, "trailing dot stripped in host"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tlsMatchCN(tc.pattern, tc.host)
			if tc.match && err != nil {
				t.Errorf("tlsMatchCN(%q, %q): expected match, got error: %v", tc.pattern, tc.host, err)
			}
			if !tc.match && err == nil {
				t.Errorf("tlsMatchCN(%q, %q): expected mismatch error, got nil", tc.pattern, tc.host)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveCipherSuites
// ---------------------------------------------------------------------------

// TestNewHTTPClient_HTTP2_TLS verifies that HTTPVersion "2.0" negotiates HTTP/2
// over a TLS connection (via ALPN), not HTTP/1.1.
func TestNewHTTPClient_HTTP2_TLS(t *testing.T) {
	var gotProto string
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProto = r.Proto
		w.WriteHeader(200)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		HTTPVersion: "2.0",
		TLS:         pal.TLSConfig{InsecureSkipVerify: true},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	status, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err != nil {
		t.Fatalf("expected successful connection with HTTPVersion 2.0, got: %v", err)
	}
	if status != 200 {
		t.Errorf("expected status 200, got %d", status)
	}
	if !strings.HasPrefix(gotProto, "HTTP/2") {
		t.Errorf("expected HTTP/2 connection with HTTPVersion 2.0, got proto: %s", gotProto)
	}
}

// TestNewHTTPClient_HTTP1_TLS verifies that HTTPVersion "1.1" forces HTTP/1.1
// even when connecting to an HTTP/2-capable TLS server.
func TestNewHTTPClient_HTTP1_TLS(t *testing.T) {
	var gotProto string
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProto = r.Proto
		w.WriteHeader(200)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		HTTPVersion: "1.1",
		TLS:         pal.TLSConfig{InsecureSkipVerify: true},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	status, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err != nil {
		t.Fatalf("expected successful connection with HTTPVersion 1.1, got: %v", err)
	}
	if status != 200 {
		t.Errorf("expected status 200, got %d", status)
	}
	if gotProto != "HTTP/1.1" {
		t.Errorf("expected HTTP/1.1 connection with HTTPVersion 1.1, got proto: %s", gotProto)
	}
}

func TestResolveCipherSuites_Empty(t *testing.T) {
	result := resolveCipherSuites([]string{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %v", result)
	}
}

func TestResolveCipherSuites_UnknownName(t *testing.T) {
	result := resolveCipherSuites([]string{"NOT_A_REAL_CIPHER"})
	if len(result) != 0 {
		t.Errorf("expected empty result for unknown cipher, got %v", result)
	}
}

func TestResolveCipherSuites_KnownSecureName(t *testing.T) {
	suites := tls.CipherSuites()
	if len(suites) == 0 {
		t.Skip("no cipher suites available on this platform")
	}
	name := suites[0].Name
	result := resolveCipherSuites([]string{name})
	if len(result) != 1 {
		t.Errorf("expected 1 resolved cipher for %q, got %d", name, len(result))
	}
	if result[0] != suites[0].ID {
		t.Errorf("expected cipher ID %d for %q, got %d", suites[0].ID, name, result[0])
	}
}

func TestResolveCipherSuites_MixedKnownUnknown(t *testing.T) {
	suites := tls.CipherSuites()
	if len(suites) == 0 {
		t.Skip("no cipher suites available on this platform")
	}
	names := []string{suites[0].Name, "UNKNOWN_CIPHER", suites[len(suites)-1].Name}
	result := resolveCipherSuites(names)
	if len(result) != 2 {
		t.Errorf("expected 2 resolved ciphers for mixed input, got %d", len(result))
	}
}

func TestResolveCipherSuites_InsecureName(t *testing.T) {
	insecure := tls.InsecureCipherSuites()
	if len(insecure) == 0 {
		t.Skip("no insecure cipher suites available on this platform")
	}
	name := insecure[0].Name
	result := resolveCipherSuites([]string{name})
	if len(result) != 1 {
		t.Errorf("expected 1 resolved insecure cipher for %q, got %d", name, len(result))
	}
}

// ---------------------------------------------------------------------------
// NewHTTPClient — real TLS handshake assertions
// ---------------------------------------------------------------------------

// TestNewHTTPClient_InsecureSkipVerify verifies that a client with
// InsecureSkipVerify=true can connect to a self-signed TLS server.
func TestNewHTTPClient_InsecureSkipVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{InsecureSkipVerify: true},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	status, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if err != nil {
		t.Fatalf("expected successful connection with InsecureSkipVerify=true, got: %v", err)
	}
	if body != nil {
		_ = body.Close()
	}
	if status != 200 {
		t.Errorf("expected status 200, got %d", status)
	}
}

// TestNewHTTPClient_TLSVerificationFails verifies that without InsecureSkipVerify
// a connection to a self-signed TLS server fails.
func TestNewHTTPClient_TLSVerificationFails(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{InsecureSkipVerify: false},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	_, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err == nil {
		t.Fatal("expected TLS verification error for self-signed cert, got nil")
	}
}

// newSelfSignedCert returns a self-signed CA-capable server certificate and its PEM.
func newSelfSignedCert(t *testing.T, commonName string, dnsNames []string, ipAddresses ...net.IP) (tls.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: commonName},
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// TestNewHTTPClient_CustomCAWithoutServerName verifies that with a custom CA and no
// configured ServerName, the certificate is verified against the request URL's host,
// including IP-literal hosts for which no SNI is sent.
func TestNewHTTPClient_CustomCAWithoutServerName(t *testing.T) {
	t.Parallel()
	loopback := net.ParseIP("127.0.0.1")
	tests := []struct {
		desc        string
		host        string
		cn          string
		dnsNames    []string
		ips         []net.IP
		httpVersion string
		wantErr     bool
	}{
		{desc: "SAN certificate", host: "localhost", dnsNames: []string{"localhost"}},
		{desc: "CN-only certificate", host: "localhost", cn: "localhost"},
		{desc: "SAN certificate over HTTP/2", host: "localhost", dnsNames: []string{"localhost"}, httpVersion: "2.0"},
		{desc: "hostname mismatch", host: "localhost", dnsNames: []string{"other.example"}, wantErr: true},
		{desc: "IP literal with IP SAN", host: "127.0.0.1", ips: []net.IP{loopback}},
		{desc: "IP literal with no SANs and empty CN", host: "127.0.0.1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			cert, caPEM := newSelfSignedCert(t, tc.cn, tc.dnsNames, tc.ips...)
			var gotProto string
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotProto = r.Proto
				w.WriteHeader(200)
			}))
			server.EnableHTTP2 = true
			server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
			server.StartTLS()
			defer server.Close()
			_, port, _ := net.SplitHostPort(server.Listener.Addr().String())

			client, err := NewHTTPClient(pal.ClientConfig{
				HTTPVersion: tc.httpVersion,
				TLS:         pal.TLSConfig{CACertPEM: caPEM},
			})
			if err != nil {
				t.Fatalf("NewHTTPClient: %v", err)
			}
			status, _, body, err := client.Execute(context.Background(), "GET", "https://"+net.JoinHostPort(tc.host, port)+"/", nil, 0, "", nil)
			if body != nil {
				_ = body.Close()
			}
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected certificate verification error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected verification against the URL host to succeed, got: %v", err)
			}
			if status != 200 {
				t.Errorf("expected status 200, got %d", status)
			}
			if tc.httpVersion == "2.0" && !strings.HasPrefix(gotProto, "HTTP/2") {
				t.Errorf("expected HTTP/2 over the custom-CA dialer, got proto: %s", gotProto)
			}
		})
	}
}

// TestTLSVerifyConnection_EmptyServerName verifies that the callback refuses to match
// when neither a configured nor an SNI server name is available.
func TestTLSVerifyConnection_EmptyServerName(t *testing.T) {
	t.Parallel()
	cert, caPEM := newSelfSignedCert(t, "", nil)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	verify := tlsVerifyConnectionWithCNFallback(pool, "")
	if err := verify(tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}}); err == nil {
		t.Fatal("expected an error for an empty server name, got nil")
	}
}

// TestNewHTTPClient_Timeout verifies that the client respects the timeout.
func TestNewHTTPClient_Timeout(t *testing.T) {
	t.Parallel()
	// Server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	_, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if !errors.Is(err, errIdleBeforeResponse) {
		t.Fatalf("expected %q, got %v", errIdleBeforeResponse, err)
	}
}

// unlimitedBody lifts the response size cap, which a zero ResponseLimitConfig sets to 0 bytes.
var unlimitedBody = pal.ResponseLimitConfig{MaxEntityBodySize: -1}

// TestNewHTTPClient_IdleTimeoutSlowConsumer verifies that only time spent waiting for data
// counts: a body that has already arrived can be read later than the timeout.
func TestNewHTTPClient_IdleTimeoutSlowConsumer(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ready"))
	}))
	defer server.Close()
	client, err := NewHTTPClient(pal.ClientConfig{Timeout: 100 * time.Millisecond, ResponseLimits: unlimitedBody})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	_, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	defer func() { _ = body.Close() }()
	time.Sleep(300 * time.Millisecond)
	data, err := io.ReadAll(body)
	if err != nil || string(data) != "ready" {
		t.Fatalf("got %q, %v; want \"ready\"", data, err)
	}
}

// TestNewHTTPClient_IdleTimeoutSlowUpload verifies that a streamed request body that keeps
// flowing keeps the request alive past the timeout.
func TestNewHTTPClient_IdleTimeoutSlowUpload(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		_, _ = w.Write(data)
	}))
	defer server.Close()
	client, err := NewHTTPClient(pal.ClientConfig{Timeout: 200 * time.Millisecond, ResponseLimits: unlimitedBody})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	pr, pw := io.Pipe()
	go func() {
		for range 8 {
			_, _ = pw.Write([]byte("part"))
			time.Sleep(50 * time.Millisecond)
		}
		_ = pw.Close()
	}()
	_, _, body, err := client.Execute(context.Background(), "POST", server.URL+"/", pr, -1, "text/plain", nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(body)
	if err != nil || len(data) != 8*len("part") {
		t.Fatalf("got %d bytes, %v; want %d bytes", len(data), err, 8*len("part"))
	}
}

// TestNewHTTPClient_IdleTimeoutSlowInMemoryUpload verifies that an in-memory body the server
// reads slowly keeps the request alive past the timeout while it keeps flowing. The server
// reads the rest at full speed, so the bytes buffered in the kernel drain within the timeout.
func TestNewHTTPClient_IdleTimeoutSlowInMemoryUpload(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 32*1024)
		for range 250 {
			if _, err := io.ReadFull(r.Body, buf); err != nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		rest, _ := io.Copy(io.Discard, r.Body)
		_, _ = fmt.Fprintf(w, "%d", 250*len(buf)+int(rest))
	}))
	defer server.Close()
	client, err := NewHTTPClient(pal.ClientConfig{Timeout: time.Second, ResponseLimits: unlimitedBody})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	payload := bytes.Repeat([]byte("x"), 16<<20)
	_, _, body, err := client.Execute(context.Background(), "POST", server.URL+"/", bytes.NewReader(payload),
		int64(len(payload)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(body)
	if err != nil || string(data) != strconv.Itoa(len(payload)) {
		t.Fatalf("got %q, %v; want %d", data, err, len(payload))
	}
}

// TestNewHTTPClient_IdleTimeoutUploadAfterResponse verifies that request-body reads after the
// response arrives do not re-arm the timer, so a consumer slower than the timeout still succeeds.
func TestNewHTTPClient_IdleTimeoutUploadAfterResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = http.NewResponseController(w).EnableFullDuplex()
		w.WriteHeader(http.StatusOK)
		_ = http.NewResponseController(w).Flush()
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte("done"))
	}))
	defer server.Close()
	client, err := NewHTTPClient(pal.ClientConfig{Timeout: 100 * time.Millisecond, ResponseLimits: unlimitedBody})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	pr, pw := io.Pipe()
	responded := make(chan struct{})
	go func() {
		<-responded
		_, _ = pw.Write([]byte("late"))
		_ = pw.Close()
	}()
	_, _, body, err := client.Execute(context.Background(), "POST", server.URL+"/", pr, -1, "text/plain", nil)
	close(responded)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	defer func() { _ = body.Close() }()
	time.Sleep(300 * time.Millisecond)
	data, err := io.ReadAll(body)
	if err != nil || string(data) != "done" {
		t.Fatalf("got %q, %v; want \"done\"", data, err)
	}
}

// TestNewHTTPClient_IdleTimeoutPerRedirectHop verifies that each followed redirect gets the
// whole timeout, so a chain longer than the timeout succeeds while every hop answers in time.
func TestNewHTTPClient_IdleTimeoutPerRedirectHop(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		hop, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
		if hop > 0 {
			http.Redirect(w, r, "/"+strconv.Itoa(hop-1), http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("done"))
	}))
	defer server.Close()
	client, err := NewHTTPClient(pal.ClientConfig{
		Timeout:         250 * time.Millisecond,
		ResponseLimits:  unlimitedBody,
		FollowRedirects: pal.FollowRedirects{Enabled: true},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	_, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/4", nil, 0, "", nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(body)
	if err != nil || string(data) != "done" {
		t.Fatalf("got %q, %v; want \"done\"", data, err)
	}
}

// TestNewHTTPClient_SizeLimitKeepsReadError verifies that a body that stalls once it reaches
// maxEntityBodySize fails with the idle timeout instead of ending cleanly at the limit.
func TestNewHTTPClient_SizeLimitKeepsReadError(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("four"))
		w.(http.Flusher).Flush()
		<-release
	}))
	defer server.Close()
	defer close(release)
	client, err := NewHTTPClient(pal.ClientConfig{
		Timeout:        100 * time.Millisecond,
		ResponseLimits: pal.ResponseLimitConfig{MaxEntityBodySize: 4},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	_, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(body)
	if !errors.Is(err, errIdleReadingBody) {
		t.Fatalf("got %q, %v; want %q", data, err, errIdleReadingBody)
	}
}

// TestNewHTTPClient_RedirectsDisabled verifies that a client with redirects
// disabled does NOT follow 3xx responses.
func TestNewHTTPClient_RedirectsDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/destination", http.StatusFound)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		FollowRedirects: pal.FollowRedirects{Enabled: false},
		ResponseLimits:  pal.ResponseLimitConfig{MaxEntityBodySize: -1},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	status, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/redirect", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusFound {
		t.Errorf("expected 302 (redirect not followed), got %d", status)
	}
}

// TestNewHTTPClient_RedirectsEnabled verifies that a client with redirects
// enabled follows 3xx responses up to the limit.
func TestNewHTTPClient_RedirectsEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/destination", http.StatusFound)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		FollowRedirects: pal.FollowRedirects{Enabled: true, MaxCount: 3},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	status, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/redirect", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("expected 200 (redirect followed), got %d", status)
	}
}

// TestNewHTTPClient_TLSVersionRange verifies that the client can be configured
// with specific TLS min/max versions and successfully connects.
func TestNewHTTPClient_TLSVersionRange(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	client, err := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	status, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 200 {
		t.Errorf("expected 200, got %d", status)
	}
}

// TestNewHTTPClient_ValidCipherSuites verifies that configuring known cipher
// suite names resolves and allows the TLS handshake to complete.
func TestNewHTTPClient_ValidCipherSuites(t *testing.T) {
	suites := tls.CipherSuites()
	if len(suites) == 0 {
		t.Skip("no cipher suites available")
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	// Build the cipher suite name list from the first available suite.
	names := make([]string, len(suites))
	for i, s := range suites {
		names[i] = s.Name
	}
	client, err := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{
			CipherSuiteNames:   names,
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	status, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err != nil {
		t.Fatalf("unexpected error with valid cipher suites: %v", err)
	}
	if status != 200 {
		t.Errorf("expected 200, got %d", status)
	}
}
