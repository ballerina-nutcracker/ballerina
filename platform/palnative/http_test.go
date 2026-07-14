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
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ballerina-lang-go/platform/pal"
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

	client := NewHTTPClient(pal.ClientConfig{
		HTTPVersion: "2.0",
		TLS:         pal.TLSConfig{InsecureSkipVerify: true},
	})
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

	client := NewHTTPClient(pal.ClientConfig{
		HTTPVersion: "1.1",
		TLS:         pal.TLSConfig{InsecureSkipVerify: true},
	})
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

	client := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{InsecureSkipVerify: true},
	})
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

	client := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{InsecureSkipVerify: false},
	})
	_, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err == nil {
		t.Fatal("expected TLS verification error for self-signed cert, got nil")
	}
}

// TestNewHTTPClient_Timeout verifies that the client respects the timeout.
func TestNewHTTPClient_Timeout(t *testing.T) {
	// Server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client := NewHTTPClient(pal.ClientConfig{
		Timeout: 100 * time.Millisecond,
	})
	_, _, body, err := client.Execute(context.Background(), "GET", server.URL+"/", nil, 0, "", nil)
	if body != nil {
		_ = body.Close()
	}
	if err == nil {
		t.Fatal("expected timeout error, got nil")
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

	client := NewHTTPClient(pal.ClientConfig{
		FollowRedirects: pal.FollowRedirects{Enabled: false},
		ResponseLimits:  pal.ResponseLimitConfig{MaxEntityBodySize: -1},
	})
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

	client := NewHTTPClient(pal.ClientConfig{
		FollowRedirects: pal.FollowRedirects{Enabled: true, MaxCount: 3},
	})
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

	client := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
			InsecureSkipVerify: true,
		},
	})
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
	client := NewHTTPClient(pal.ClientConfig{
		TLS: pal.TLSConfig{
			CipherSuiteNames:   names,
			InsecureSkipVerify: true,
		},
	})
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
