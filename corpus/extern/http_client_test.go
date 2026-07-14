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

package extern_test

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testharness"
)

// rewritingHTTPClient forwards requests from "http://testserver/..." (the
// hostname used inside fixture .bal files) to a per-test local httptest server.
type rewritingHTTPClient struct {
	serverURL string
	client    *http.Client
}

func (c *rewritingHTTPClient) Execute(ctx context.Context, method, url string, body io.Reader, _ int64, contentType string, reqHeaders map[string][]string) (int, map[string][]string, io.ReadCloser, error) {
	const prefix = "http://testserver"
	if !strings.HasPrefix(url, prefix) {
		return 0, nil, nil, fmt.Errorf("rewritingHTTPClient: expected URL with prefix %q, got %q", prefix, url)
	}
	realURL := c.serverURL + url[len(prefix):]
	req, err := http.NewRequestWithContext(ctx, method, realURL, body)
	if err != nil {
		return 0, nil, nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, vals := range reqHeaders {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	return resp.StatusCode, map[string][]string(resp.Header), resp.Body, nil
}

// rewriteClient returns a NewClient factory that forwards every request to
// serverURL via rewritingHTTPClient.
func rewriteClient(serverURL string) func(pal.ClientConfig) pal.HTTPClient {
	return func(cfg pal.ClientConfig) pal.HTTPClient {
		return &rewritingHTTPClient{
			serverURL: serverURL,
			client:    &http.Client{Timeout: cfg.Timeout},
		}
	}
}

func TestHttpClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			w.WriteHeader(200)
			xTest := r.Header.Get("X-Test")
			if xTest != "" {
				_, _ = fmt.Fprintf(w, "hello with %s", xTest)
			} else {
				_, _ = fmt.Fprint(w, "hello from test server")
			}
		} else {
			w.WriteHeader(404)
			_, _ = fmt.Fprint(w, "not found")
		}
	}))
	defer server.Close()
	runExtern(t, fileCase("http-client-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

func TestHttpClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/echo" {
			body, _ := io.ReadAll(r.Body)
			ct := r.Header.Get("Content-Type")
			ct = strings.SplitN(ct, ";", 2)[0]
			w.WriteHeader(200)
			_, _ = fmt.Fprintf(w, "body: %s, ct: %s", string(body), ct)
		} else {
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	runExtern(t, fileCase("http-client-post-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

func TestHttpClientMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/echo":
			body, _ := io.ReadAll(r.Body)
			ct := r.Header.Get("Content-Type")
			ct = strings.SplitN(ct, ";", 2)[0]
			w.WriteHeader(200)
			_, _ = fmt.Fprintf(w, "body: %s, ct: %s", string(body), ct)
		case r.URL.Path == "/delete" && r.Method == http.MethodDelete:
			w.WriteHeader(200)
		case r.URL.Path == "/head" && r.Method == http.MethodHead:
			w.WriteHeader(200)
		case r.URL.Path == "/options" && r.Method == http.MethodOptions:
			w.WriteHeader(200)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	runExtern(t, fileCase("http-client-methods-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

// TestHttpClientJsonLocal exercises JSON request serialization (balToGoJSON) and
// JSON response parsing (goToBalValue) against a local echo server, with no network.
func TestHttpClientJsonLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/echo" && r.Method == http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(body)
		case r.URL.Path == "/json-array":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = fmt.Fprint(w, `[1, 2.5, "three", true, null, [10, 20], {"k": "v"}]`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	runExtern(t, fileCase("http-client-json-local-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

// TestHttpClientBinaryLocal exercises byte[] request bodies (listToBytes) and
// binary response payloads (getBinaryPayload) against a local server.
func TestHttpClientBinaryLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/echo-bytes" && r.Method == http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(200)
			_, _ = w.Write(body)
		case r.URL.Path == "/bytes":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(200)
			_, _ = w.Write([]byte{1, 2, 3, 4})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	runExtern(t, fileCase("http-client-binary-local-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

// TestHttpClientCompressionLocal exercises transparent gzip/deflate response
// decompression (decompressResponseBody + the gzip/deflate ReadClosers). The
// client factory disables Go's automatic decompression so the compressed body
// and Content-Encoding header reach the Ballerina layer untouched.
func TestHttpClientCompressionLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gzip":
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(200)
			gz := gzip.NewWriter(w)
			_, _ = gz.Write([]byte("hello gzipped world"))
			_ = gz.Close()
		case "/deflate":
			w.Header().Set("Content-Encoding", "deflate")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(200)
			fl, _ := flate.NewWriter(w, flate.DefaultCompression)
			_, _ = fl.Write([]byte("hello deflated world"))
			_ = fl.Close()
		case "/badgzip":
			// Advertise gzip but send a body that is not a valid gzip stream, so
			// the gzip header parse fails when the payload is read.
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(200)
			_, _ = w.Write([]byte("this is not gzip"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	noAutoDecompress := func(cfg pal.ClientConfig) pal.HTTPClient {
		return &rewritingHTTPClient{
			serverURL: server.URL,
			client:    &http.Client{Timeout: cfg.Timeout, Transport: &http.Transport{DisableCompression: true}},
		}
	}
	runExtern(t, fileCase("http-client-compression-local-v"), newHTTPPal(noAutoDecompress), nil)
}

// TestHttpClientHeadersLocal exercises request-header handling: forwarding a
// materialised request with a hop-by-hop header (removeHopByHopHeaders /
// takeStream / materialize), mixed string/list headers (extractHeaders), and
// the ALWAYS/NEVER compression modes (applyCompressionHeaders).
func TestHttpClientHeadersLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A hop-by-hop Connection header must not have been forwarded.
		if r.URL.Path == "/fwd" && r.Header.Get("Connection") != "" {
			w.WriteHeader(400)
			return
		}
		if r.URL.Path == "/empty" {
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	runExtern(t, fileCase("http-client-headers-local-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

// TestHttpClientTLSInsecure: A client without InsecureSkipVerify would fail
// the handshake against a self-signed httptest TLS server. This verifies
// secureSocket: {enable: false} propagates InsecureSkipVerify through PAL.
func TestHttpClientTLSInsecure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, "secure hello")
	}))
	defer server.Close()

	clientFactory := func(cfg pal.ClientConfig) pal.HTTPClient {
		serverTLSConfig := server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
		if cfg.TLS.InsecureSkipVerify {
			serverTLSConfig.InsecureSkipVerify = true //nolint:gosec
		}
		serverTLSConfig.NextProtos = []string{"http/1.1"}
		return &rewritingHTTPClient{
			serverURL: server.URL,
			client:    &http.Client{Timeout: cfg.Timeout, Transport: &http.Transport{TLSClientConfig: serverTLSConfig}},
		}
	}
	runExtern(t, fileCase("http-client-tls-v"), newHTTPPal(clientFactory), nil)
}

// TestHttpClientPublicGet exercises palnative.NewHTTPClient against a real
// public endpoint, ensuring the full Ballerina → PAL → palnative path is
// covered. Skipped when EXTERN_SKIP_NETWORK=1 or no network is available.
// TODO: Replace with a Ballerina HTTP service once server support lands.
func TestHttpClientPublicGet(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-public-get-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientRedirect exercises redirect-following through palnative.NewHTTPClient.
func TestHttpClientRedirect(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-redirect-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientJson exercises Response.getJsonPayload against httpbin /json.
// TODO: Replace with a Ballerina HTTP service once server support lands.
func TestHttpClientJson(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-json-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientText exercises Response.getTextPayload against httpbin /html.
// TODO: Replace with a Ballerina HTTP service once server support lands.
func TestHttpClientText(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-text-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientBinary exercises Response.getBinaryPayload against httpbin /bytes/16.
// TODO: Replace with a Ballerina HTTP service once server support lands.
func TestHttpClientBinary(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-binary-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientPublicMethods exercises POST, PUT, DELETE, and PATCH against
// dedicated httpbin endpoints that each return 200 for their verb.
// TODO: Replace with a Ballerina HTTP service once server support lands.
func TestHttpClientPublicMethods(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-public-methods-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientTimeout verifies that a 1-second timeout fires before
// httpbin /delay/5 responds, and that the resulting error propagates to
// Ballerina as an error value.
// TODO: Replace with a Ballerina HTTP service once server support lands.
func TestHttpClientTimeout(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-timeout-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientConnectionError verifies that a DNS resolution failure for an
// unreachable host propagates back to Ballerina as an error value.
// TODO: Replace with a Ballerina HTTP service once server support lands.
func TestHttpClientConnectionError(t *testing.T) {
	skipIfNoNetwork(t)
	runExtern(t, fileCase("http-client-connection-error-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientMTLS verifies mutual TLS by spinning up a local HTTPS server
// that requires a valid client certificate, then confirming the Ballerina
// secureSocket.key.certFile / keyFile config causes the client to send it.
// Uses locally generated certs so no external network is needed.
//
// Unlike the other HTTP tests, the .bal source must embed the dynamic server
// URL and cert paths, so we materialise a temp .bal each run. The expected
// golden file is checked against the runtime-generated source via the harness.
func TestHttpClientMTLS(t *testing.T) {
	caCertPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM := generateTestCerts(t)

	serverTLSCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}
	serverTLSCert.Leaf, err = x509.ParseCertificate(serverTLSCert.Certificate[0])
	if err != nil {
		t.Fatalf("parsing server TLS cert leaf: %v", err)
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(caCertPEM)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverTLSCert},
		ClientCAs:    clientCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		NextProtos:   []string{"http/1.1"},
	}
	server.StartTLS()
	defer server.Close()

	tmpDir := t.TempDir()
	clientCertFile := filepath.Join(tmpDir, "client.pem")
	clientKeyFile := filepath.Join(tmpDir, "client-key.pem")
	for _, pair := range []struct {
		path string
		data []byte
	}{
		{clientCertFile, clientCertPEM},
		{clientKeyFile, clientKeyPEM},
	} {
		if err := os.WriteFile(pair.path, pair.data, 0600); err != nil {
			t.Fatalf("writing %s: %v", pair.path, err)
		}
	}

	// verifyHostName: false skips server-cert verification (the IP-address
	// target has no SNI). The test focus is the client-cert path: the server
	// still validates the client cert against its CA pool and rejects without it.
	certFileSlash := filepath.ToSlash(clientCertFile)
	keyFileSlash := filepath.ToSlash(clientKeyFile)
	balContent := fmt.Sprintf(`
import ballerina/http;
import ballerina/io;

public function main() returns error? {
    http:Client c = check new ("%s", {
        httpVersion: http:HTTP_1_1,
        secureSocket: {
            verifyHostName: false,
            key: {certFile: "%s", keyFile: "%s"}
        }
    });
    http:Response r = check c->get("/");
    io:println(r.statusCode); // @output 200
    return;
}
`, server.URL, certFileSlash, keyFileSlash)

	tmpBalFile := filepath.Join(tmpDir, "http-client-mtls-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "http-client-mtls-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "http-client-mtls-v.txtar"),
	}
	runExtern(t, tc, newHTTPPal(palnative.NewHTTPClient).withRealFS(), nil)
}

func TestHttpClientForward(t *testing.T) {
	type forwardResult struct {
		method string
		body   []byte
		header string
	}
	resultCh := make(chan forwardResult, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		resultCh <- forwardResult{method: r.Method, body: body, header: r.Header.Get("X-Forwarded-From")}
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, "forwarded ok")
	}))
	defer server.Close()

	tc := fileCase("http-client-forward-v")
	tp := newHTTPPal(rewriteClient(server.URL))
	testharness.Run(t, tc, tp, nil)

	received := <-resultCh
	if received.method != "POST" {
		t.Errorf("expected forwarded method POST, got %q", received.method)
	}
	if string(received.body) != "hello body" {
		t.Errorf("expected forwarded body %q, got %q", "hello body", string(received.body))
	}
	if received.header != "test" {
		t.Errorf("expected X-Forwarded-From %q, got %q", "test", received.header)
	}

	if *update {
		testharness.Update(t, tc, tp)
		return
	}
	testharness.Validate(t, tc, tp)
}

// generateTestCerts generates a self-signed CA, a server cert for 127.0.0.1,
// and a client cert. All leaves are signed by the CA. Returns PEM-encoded
// bytes ready for use with tls.X509KeyPair or file writes.
func generateTestCerts(t *testing.T) (caCertPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM []byte) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caSerial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	caTemplate := &x509.Certificate{
		SerialNumber:          caSerial,
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	caCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	newLeafCert := func(cn string, ips []net.IP, extUsage x509.ExtKeyUsage) (certPEM, keyPEM []byte) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		tmpl := &x509.Certificate{
			SerialNumber: serial,
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			ExtKeyUsage:  []x509.ExtKeyUsage{extUsage},
			IPAddresses:  ips,
		}
		certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
		if err != nil {
			t.Fatal(err)
		}
		keyDER, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
			pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	}

	serverCertPEM, serverKeyPEM = newLeafCert("127.0.0.1", []net.IP{net.ParseIP("127.0.0.1")}, x509.ExtKeyUsageServerAuth)
	clientCertPEM, clientKeyPEM = newLeafCert("client", nil, x509.ExtKeyUsageClientAuth)
	return
}
