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
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
)

// The keystore lives with the other PKCS#12 fixtures under testdata/.
func TestJwtKeyStore(t *testing.T) {
	t.Parallel()
	runExtern(t, fileCase("jwt-keystore-v"), testharness.NewTestPal(), nil)
}

func TestJwtTrustStore(t *testing.T) {
	t.Parallel()
	runExtern(t, fileCase("jwt-truststore-v"), testharness.NewTestPal(), nil)
}

// jwksServer serves a JWKS holding the public half of testdata/jwt/rsa.key as kid "k1", plus malformed variants.
func jwksServer(t *testing.T) *httptest.Server {
	t.Helper()
	keyPEM, err := os.ReadFile("testdata/jwt/rsa.key")
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(keyPEM)
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	pub := parsed.(*rsa.PrivateKey).PublicKey
	encode := base64.RawURLEncoding.EncodeToString
	keys, err := json.Marshal(map[string]any{"keys": []map[string]string{
		{"kty": "RSA", "kid": "other", "n": encode(big.NewInt(3).Bytes()), "e": "AQAB"},
		{"kty": "RSA", "kid": "k1", "n": encode(pub.N.Bytes()), "e": encode(big.NewInt(int64(pub.E)).Bytes())},
	}})
	if err != nil {
		t.Fatal(err)
	}
	responses := map[string]struct {
		status int
		body   string
	}{
		"/jwks":           {200, string(keys)},
		"/created":        {201, `{"keys":[]}`},
		"/not-array":      {200, `{"keys":{}}`},
		"/no-keys":        {200, `{"x":1}`},
		"/bad-json":       {200, "not json"},
		"/no-modulus":     {200, `{"keys":[{"kid":"k1","e":"AQAB"}]}`},
		"/no-exponent":    {200, `{"keys":[{"kid":"k1","n":"AQAB"}]}`},
		"/number-modulus": {200, `{"keys":[{"kid":"k1","n":1,"e":"AQAB"}]}`},
		"/bad-modulus":    {200, `{"keys":[{"kid":"k1","n":"$$","e":"AQAB"}]}`},
		"/no-kid":         {200, `{"keys":[{"n":"a","e":"b"}]}`},
		"/number-kid":     {200, `{"keys":[{"kid":1,"n":"a","e":"b"}]}`},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := responses[r.URL.Path]
		if !ok || r.Method != http.MethodGet {
			w.WriteHeader(404)
			_, _ = w.Write([]byte("missing"))
			return
		}
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
}

func TestJwtJwks(t *testing.T) {
	t.Parallel()
	server := jwksServer(t)
	defer server.Close()
	runExtern(t, fileCase("jwt-jwks-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

func TestJwtJwksTLS(t *testing.T) {
	t.Parallel()
	caCertPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM := generateTestCerts(t)
	serverTLSCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}
	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(caCertPEM)
	plain := jwksServer(t)
	defer plain.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mtls/jwks" && len(r.TLS.PeerCertificates) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("client certificate required"))
			return
		}
		r.URL.Path = "/jwks"
		plain.Config.Handler.ServeHTTP(w, r)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverTLSCert}, ClientCAs: clientCAs, ClientAuth: tls.VerifyClientCertIfGiven}
	server.StartTLS()
	defer server.Close()

	tmpDir := t.TempDir()
	files := map[string][]byte{"ca.pem": caCertPEM, "client.pem": clientCertPEM, "client-key.pem": clientKeyPEM}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), data, 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	path := func(name string) string { return filepath.ToSlash(filepath.Join(tmpDir, name)) }

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/jwt;

function show(jwt:Payload|jwt:Error result) {
    if result is jwt:Error {
        error? cause = result.cause();
        io:println(cause is error ? cause.message() : result.message());
    } else {
        io:println("valid ", result?.sub ?: "-");
    }
}

public function main() returns error? {
    string url = "%[1]s/jwks";
    string mtlsUrl = "%[1]s/mtls/jwks";
    string token = check jwt:issue({username: "bob", keyId: "k1", signatureConfig: {config: {keyFile: "testdata/jwt/rsa.key"}}});
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {secureSocket: {cert: %[2]q}}}}})); // @output valid bob
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {httpVersion: jwt:HTTP_2, secureSocket: {cert: %[2]q}}}}})); // @output valid bob
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {secureSocket: {disable: true}}}}})); // @output valid bob
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url: mtlsUrl, clientConfig: {secureSocket: {cert: %[2]q, key: {certFile: %[3]q, keyFile: %[4]q}}}}}})); // @output valid bob
    jwt:Payload|jwt:Error withoutClientCert = jwt:validate(token, {signatureConfig: {jwksConfig: {url: mtlsUrl, clientConfig: {secureSocket: {cert: %[2]q}}}}});
    io:println(withoutClientCert is jwt:Error); // @output true
    jwt:Payload|jwt:Error untrusted = jwt:validate(token, {signatureConfig: {jwksConfig: {url}}});
    io:println(untrusted is jwt:Error); // @output true
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {secureSocket: {}}}}})); // @output Failed to init SSL context. Need to configure 'crypto:TrustStore' or 'cert' with client SSL certificates file.
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {secureSocket: {cert: {path: "truststore.p12", password: "secret"}}}}}})); // @output Failed to init SSL context. crypto:TrustStore is not yet supported for secureSocket.cert; use a certificate file path instead
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {secureSocket: {cert: %[2]q, key: {path: "keystore.p12", password: "secret"}}}}}})); // @output Failed to init SSL context. crypto:KeyStore is not yet supported for secureSocket.key; use a jwt:CertKey instead
    show(jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {secureSocket: {cert: %[2]q, key: {certFile: %[3]q, keyFile: %[4]q, keyPassword: "p"}}}}}})); // @output Failed to init SSL context. an encrypted private key is not yet supported for secureSocket.key
    jwt:Payload|jwt:Error missingCa = jwt:validate(token, {signatureConfig: {jwksConfig: {url, clientConfig: {secureSocket: {cert: "missing-ca.pem"}}}}});
    io:println(missingCa is jwt:Error); // @output true
}
`, server.URL, path("ca.pem"), path("client.pem"), path("client-key.pem"))
	tmpBalFile := filepath.Join(tmpDir, "jwt-jwks-tls-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}
	tc := test_util.TestCase{
		Name:         "jwt-jwks-tls-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "jwt-jwks-tls-v.txtar"),
	}
	runExtern(t, tc, newHTTPPal(palnative.NewHTTPClient), nil)
}
