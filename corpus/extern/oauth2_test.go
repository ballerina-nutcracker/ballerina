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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/test_util"
)

// oauth2Server is a token and introspection endpoint for the oauth2 tests. Token values carry a per-server request
// counter so the tests can tell a cached token from a newly requested one.
func oauth2Server() *httptest.Server {
	var mu sync.Mutex
	count := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		mu.Lock()
		count++
		n := count
		mu.Unlock()
		reply := func(status int, body any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write(marshal(body))
		}
		grant := form.Get("grant_type")
		switch path := r.URL.Path; path {
		case "/echo":
			info := marshal(map[string]any{"ct": r.Header.Get("Content-Type"), "auth": r.Header.Get("Authorization"),
				"x": r.Header.Values("X-Custom"), "body": string(raw)})
			reply(200, map[string]any{"access_token": strings.TrimSpace(string(info)), "expires_in": 3600, "extra": []any{1, 2.5, []any{}}})
		case "/token", "/token-short", "/created":
			expiresIn, status := 3600, 200
			switch path {
			case "/token-short":
				expiresIn = 1
			case "/created":
				status = 201
			}
			switch {
			case grant == "client_credentials":
				reply(status, map[string]any{"access_token": fmt.Sprintf("cc-%d", n), "expires_in": expiresIn})
			case grant == "password":
				reply(status, map[string]any{"access_token": fmt.Sprintf("pw-%d", n), "refresh_token": fmt.Sprintf("rt-%d", n), "expires_in": expiresIn})
			case grant == "refresh_token":
				reply(status, map[string]any{"access_token": "refreshed-" + form.Get("refresh_token"), "refresh_token": fmt.Sprintf("rt-r%d", n), "expires_in": expiresIn})
			case strings.HasPrefix(grant, "urn:ietf:params:oauth:grant-type:jwt-bearer"):
				reply(status, map[string]any{"access_token": fmt.Sprintf("jb-%d", n), "refresh_token": fmt.Sprintf("rt-%d", n), "expires_in": expiresIn})
			default:
				reply(400, map[string]any{"error": "unsupported_grant_type"})
			}
		case "/no-expiry":
			reply(200, map[string]any{"access_token": fmt.Sprintf("ne-%d", n)})
		case "/401":
			w.WriteHeader(401)
			_, _ = w.Write([]byte("unauthorized"))
		case "/bad-json":
			_, _ = w.Write([]byte("not json"))
		case "/no-access-token":
			reply(200, map[string]any{"x": 1})
		case "/number-access-token":
			reply(200, map[string]any{"access_token": 1})
		case "/big-number":
			_, _ = w.Write([]byte(`{"access_token":"x","expires_in":12345678901234567890}`))
		case "/introspect":
			introspect(w, r, form, reply)
		case "/hang":
			<-r.Context().Done()
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte("missing"))
		}
	}))
}

// marshal encodes without HTML escaping, so form payloads echo back with their literal "&".
func marshal(v any) []byte {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return []byte(sb.String())
}

func introspect(_ http.ResponseWriter, r *http.Request, form url.Values, reply func(int, any)) {
	switch form.Get("token") {
	case "active":
		reply(200, map[string]any{"active": true, "scope": "read write", "client_id": "c1", "username": "alice",
			"token_type": "Bearer", "exp": time.Now().Unix() + 3600, "iat": 1, "nbf": 1, "sub": "s", "aud": "a", "iss": "i",
			"jti": "j", "ext": map[string]any{"auth": r.Header.Get("Authorization"), "hint": form.Get("token_type_hint"),
				"extra": form.Get("extra")}})
	case "short":
		reply(200, map[string]any{"active": true, "exp": time.Now().Unix() + 1})
	case "string-exp":
		reply(200, map[string]any{"active": true, "exp": "soon"})
	default:
		reply(200, map[string]any{"active": false})
	}
}

func TestOAuth2Client(t *testing.T) {
	t.Parallel()
	server := oauth2Server()
	defer server.Close()
	runExtern(t, fileCase("oauth2-client-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

func TestOAuth2ClientTLS(t *testing.T) {
	t.Parallel()
	caCertPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM := generateTestCerts(t)
	serverTLSCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}
	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(caCertPEM)
	plain := oauth2Server()
	defer plain.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/mtls/") {
			if len(r.TLS.PeerCertificates) == 0 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/mtls")
		}
		plain.Config.Handler.ServeHTTP(w, r)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverTLSCert}, ClientCAs: clientCAs, ClientAuth: tls.VerifyClientCertIfGiven}
	server.StartTLS()
	defer server.Close()

	tmpDir := t.TempDir()
	for name, data := range map[string][]byte{"ca.pem": caCertPEM, "client.pem": clientCertPEM, "client-key.pem": clientKeyPEM} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), data, 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	path := func(name string) string { return filepath.ToSlash(filepath.Join(tmpDir, name)) }

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/oauth2;

function token(string url, oauth2:SecureSocket? secureSocket = ()) {
    oauth2:ClientConfiguration clientConfig = {};
    if secureSocket is oauth2:SecureSocket {
        clientConfig.secureSocket = secureSocket;
    }
    oauth2:ClientOAuth2Provider|error provider = trap new ({tokenUrl: url, clientId: "c1", clientSecret: "s1", clientConfig});
    if provider is error {
        error? cause = provider.cause();
        io:println("failed: ", cause is error ? cause.message() : provider.message());
        return;
    }
    string|error generated = provider.generateToken();
    io:println(generated is string ? generated.substring(0, 3) : generated.message());
}

public function main() {
    string base = "%[1]s";
    token(base + "/token", {cert: %[2]q}); // @output cc-
    token(base + "/token", {disable: true}); // @output cc-
    token(base + "/mtls/token", {cert: %[2]q, key: {certFile: %[3]q, keyFile: %[4]q}}); // @output cc-
    token(base + "/mtls/token", {cert: %[2]q}); // @output failed: Failed to get a success response from the endpoint. Response code: '401'. Response body: ''
    oauth2:ClientOAuth2Provider|error untrusted = trap new ({tokenUrl: base + "/token", clientId: "c1", clientSecret: "s1"});
    io:println(untrusted is error); // @output true
    token(base + "/token", {}); // @output failed: Failed to init SSL context. Need to configure 'crypto:TrustStore' or 'cert' with client SSL certificates file.
    token(base + "/token", {cert: {path: "truststore.p12", password: "secret"}}); // @output failed: Failed to init SSL context. crypto:TrustStore is not yet supported for secureSocket.cert; use a certificate file path instead
    token(base + "/token", {cert: %[2]q, key: {path: "keystore.p12", password: "secret"}}); // @output failed: Failed to init SSL context. crypto:KeyStore is not yet supported for secureSocket.key; use an oauth2:CertKey instead
    token(base + "/token", {cert: %[2]q, key: {certFile: %[3]q, keyFile: %[4]q, keyPassword: "p"}}); // @output failed: Failed to init SSL context. an encrypted private key is not yet supported for secureSocket.key
    oauth2:ClientOAuth2Provider|error missingCa = trap new ({tokenUrl: base + "/token", clientId: "c1", clientSecret: "s1", clientConfig: {secureSocket: {cert: "missing-ca.pem"}}});
    io:println(missingCa is error); // @output true
}
`, server.URL, path("ca.pem"), path("client.pem"), path("client-key.pem"))
	tmpBalFile := filepath.Join(tmpDir, "oauth2-client-tls-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}
	tc := test_util.TestCase{
		Name:         "oauth2-client-tls-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "oauth2-client-tls-v.txtar"),
	}
	runExtern(t, tc, newHTTPPal(palnative.NewHTTPClient), nil)
}
