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
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/test_util"
)

// skipIfNoLoopback skips on platforms without loopback TCP (js/wasm). Unlike
// skipIfNoNetwork these service tests only need localhost, not the internet.
func skipIfNoLoopback(t *testing.T) {
	t.Helper()
	if goruntime.GOOS == "js" {
		t.Skip("skipping loopback-dependent test on js/wasm")
	}
}

// TestHttpServiceBasic starts a Ballerina http service on a real port and
// drives it from testMain via a real http:Client, exercising the full
// listener → dispatch → resource → response path.
func TestHttpServiceBasic(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-basic-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServicePathParam exercises a typed (int) path parameter.
func TestHttpServicePathParam(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-path-param-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServicePathParamUnion exercises a union-typed (int|float) path
// parameter, and confirms a path parameter type outside the basic-type set
// never matches a raw URL segment.
func TestHttpServicePathParamUnion(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-path-param-union-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceRequest exercises Request injection and JSON body round-trip
// through a POST resource.
func TestHttpServiceRequest(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-request-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceRouting exercises 200 / 404 (unknown path) / 405 (wrong
// method) dispatch outcomes.
func TestHttpServiceRouting(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-routing-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServicePathBoundary verifies that base-path matching stops at a
// path-segment boundary: a service attached at /foo must not match /foobar
// or /foobaz.
func TestHttpServicePathBoundary(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-path-boundary-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceMultiService attaches two services at distinct base paths to a
// single listener and confirms each routes to the correct service.
func TestHttpServiceMultiService(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-multi-service-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceAttachTrailingSlash verifies that a base path supplied to
// Listener.attach with a trailing slash (e.g. "/foo/") still matches
// sub-paths at a segment boundary, the same as "/foo" would.
func TestHttpServiceAttachTrailingSlash(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-attach-trailing-slash-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceRequestAccessors exercises inbound Request.rawPath (which
// must retain the raw request-target, including the query string) and the
// getContentType/getHeaderNames/getQueryParamValues accessors.
func TestHttpServiceRequestAccessors(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-request-accessors-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceHTTP10Fallback verifies that a listener configured with
// HTTP_1_0 (accepted by the enum but unsupported by the Go HTTP runtime)
// falls back to HTTP/1.1 with a warning instead of forwarding "1.0" to the
// platform layer.
func TestHttpServiceHTTP10Fallback(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-http10-fallback-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceTypedParams exercises the runtime path dispatcher's coercion of
// boolean, decimal, and string path parameters.
func TestHttpServiceTypedParams(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-typed-params-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceRestPathParam exercises a resource whose path ends in a rest
// path parameter and declares no other parameters (regression: the dispatcher
// must not miscount the rest segment as an extra caller-supplied argument).
func TestHttpServiceRestPathParam(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-rest-path-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceDuplicateBasePath verifies that attaching two services at the
// same base path fails at listener-init time with a runtime error.
func TestHttpServiceDuplicateBasePath(t *testing.T) {
	runExtern(t, fileCase("http-service/http-svc-dup-path-p"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceStatusCode verifies that direct field assignment to
// resp.statusCode is honoured by the server (201, 404) and that the
// field default of 200 is used when no assignment is made.
func TestHttpServiceStatusCode(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-status-code-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceTLS exercises a TLS listener: the server loads its cert/key
// from disk (realFS) and the client connects over https with verification
// disabled for the self-signed cert.
func TestHttpServiceTLS(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-tls-v"), newHTTPPal(palnative.NewHTTPClient).withRealFS(), nil)
}

// TestHttpServiceClientPayloads drives a local service from testMain to exercise
// the client-side payload getters (getJsonPayload / getTextPayload /
// getBinaryPayload) and a default-200 response over the real palnative client.
func TestHttpServiceClientPayloads(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-client-payloads-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceClientMethods exercises the POST, PUT, DELETE, and PATCH client
// verbs against dedicated local resources that each return 200.
func TestHttpServiceClientMethods(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-client-methods-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceClientRedirect verifies the client follows a 302 redirect
// (Location header) emitted by a local service and lands on the 200 target.
func TestHttpServiceClientRedirect(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-client-redirect-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceResponseVariants covers the writeResult return-value cases: a
// () return mapped to 202, an error value mapped to 500, a multi-value response
// header, and a hop-by-hop header dropped before reaching the client.
func TestHttpServiceResponseVariants(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-response-variants-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceNoMatch covers dispatch 404s: a request to an unmatched base
// path and a request to a bare base path that resolves no resource.
func TestHttpServiceNoMatch(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-no-match-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceConfig covers the listener host and timeout configuration
// fields in initNative.
func TestHttpServiceConfig(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-config-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceLargeBody covers the streaming (non-eager) request body path
// in buildRequestFromHTTP by posting a body larger than eagerBufferThreshold.
func TestHttpServiceLargeBody(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-large-body-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceTLSOptions covers the optional TLS listener settings in
// buildListenerTLSConfig: protocol version bounds, cipher suites, and disabled
// session tickets.
func TestHttpServiceTLSOptions(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-tls-options-v"), newHTTPPal(palnative.NewHTTPClient).withRealFS(), nil)
}

// TestHttpServiceGracefulStop verifies that a resource calling
// ep.gracefulStop() on the listener it is attached to does not self-deadlock:
// the extern runs inline on the handler's own goroutine, so it must return
// immediately (letting the drain happen in the background) rather than
// blocking on server.Shutdown until this very connection goes idle. Before
// the fix this test hangs forever.
func TestHttpServiceGracefulStop(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-graceful-stop-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceImmediateStop covers immediate stop and requests-after-stop.
// immediateStop calls Close(), which severs active connections and shuts the
// listener down synchronously (unlike graceful stop, whose listener-close is
// backgrounded and would be racy to assert on). So the stop call itself errors
// (its connection is cut before the response arrives) and any request after
// stop is connection-refused.
func TestHttpServiceImmediateStop(t *testing.T) {
	skipIfNoLoopback(t)
	runExtern(t, fileCase("http-service/http-svc-immediate-stop-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpServiceMTLS covers the mutual-TLS branch of buildListenerTLSConfig: a
// listener configured with mutualSsl + a CA cert requires the client to present
// a cert signed by that CA. Certs are generated at runtime (reusing
// generateTestCerts), so the .bal source embedding the temp cert paths is
// materialised per run, mirroring TestHttpClientMTLS.
func TestHttpServiceMTLS(t *testing.T) {
	skipIfNoLoopback(t)
	caCertPEM, serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM := generateTestCerts(t)

	tmpDir := t.TempDir()
	files := map[string][]byte{
		"server.crt": serverCertPEM,
		"server.key": serverKeyPEM,
		"ca.crt":     caCertPEM,
		"client.crt": clientCertPEM,
		"client.key": clientKeyPEM,
	}
	paths := make(map[string]string, len(files))
	for name, data := range files {
		p := filepath.Join(tmpDir, name)
		if err := os.WriteFile(p, data, 0600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
		paths[name] = filepath.ToSlash(p)
	}

	balContent := fmt.Sprintf(`
import ballerina/http;
import ballerina/io;

http:ListenerConfiguration mtlsConfig = {
    httpVersion: http:HTTP_1_1,
    secureSocket: {
        key: {certFile: "%s", keyFile: "%s"},
        mutualSsl: true,
        cert: "%s"
    }
};

service /secure on new http:Listener(19208, mtlsConfig) {
    resource function get hello() returns http:Response {
        http:Response resp = new;
        resp.setTextPayload("mtls hello");
        return resp;
    }
}

public function testMain() returns error? {
    http:Client c = check new ("https://localhost:19208", {
        httpVersion: http:HTTP_1_1,
        secureSocket: {
            verifyHostName: false,
            key: {certFile: "%s", keyFile: "%s"}
        }
    });
    http:Response r = check c->get("/secure/hello");
    io:println(r.statusCode); // @output 200
    io:println(r.getTextPayload()); // @output mtls hello

    http:Client cNoCert = check new ("https://localhost:19208", {
        httpVersion: http:HTTP_1_1,
        secureSocket: {
            verifyHostName: false
        }
    });
    http:Response|error r2 = cNoCert->get("/secure/hello");
    io:println(r2 is error); // @output true
}
`, paths["server.crt"], paths["server.key"], paths["ca.crt"], paths["client.crt"], paths["client.key"])

	tmpBalFile := filepath.Join(tmpDir, "http-svc-mtls-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "http-svc-mtls-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "http-service", "http-svc-mtls-v.txtar"),
	}
	runExtern(t, tc, newHTTPPal(palnative.NewHTTPClient).withRealFS(), nil)
}
