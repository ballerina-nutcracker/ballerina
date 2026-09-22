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
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

// TestHttpClientDataBindLocal covers each payload builder and the unknown-media-type
// fallback, including targets narrower than the type their builder produces. The responses
// are served by a Ballerina service in the fixture itself, so no Go server is needed.
func TestHttpClientDataBindLocal(t *testing.T) {
	skipIfNoLoopback(t)
	t.Parallel()
	runExtern(t, fileCase("http-client-databind-local-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientDataBindErrorsLocal covers the failure paths: status mapping, binding
// mismatches, incompatible media types, and the empty-body rules.
func TestHttpClientDataBindErrorsLocal(t *testing.T) {
	skipIfNoLoopback(t)
	t.Parallel()
	runExtern(t, fileCase("http-client-databind-errors-local-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientDataBindXmlLocal covers the xml payload builder: media-type variants, the
// narrower xml:Element target, empty and malformed bodies, an incompatible target, and an
// xml request payload round-tripping through getXmlPayload on an inbound request.
func TestHttpClientDataBindXmlLocal(t *testing.T) {
	skipIfNoLoopback(t)
	t.Parallel()
	runExtern(t, fileCase("http-client-databind-xml-local-v"), newHTTPPal(palnative.NewHTTPClient), nil)
}

// TestHttpClientDataBindNoContentTypeLocal covers the target-type fallback for a response
// carrying no Content-Type at all. A Ballerina service cannot produce one — the server always
// emits a Content-Type, and removeHeader on it does not take — so this case keeps a Go server.
func TestHttpClientDataBindNoContentTypeLocal(t *testing.T) {
	server := noContentTypeServer()
	defer server.Close()
	runExtern(t, fileCase("http-client-databind-no-content-type-local-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}

// noContentTypeServer suppresses Go's content sniffing so the responses really carry no
// Content-Type header.
func noContentTypeServer() *httptest.Server {
	bodies := map[string]string{
		"/no-type":        "untyped body",
		"/no-type-colour": "red",
		"/no-type-empty":  "",
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header()["Content-Type"] = nil
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, body)
	}))
}

// truncatingServer declares a Content-Length it does not deliver, so the transport closes the
// connection early and reading the response body fails with an unexpected EOF.
//
// Every body is a complete, valid payload for the target it is read into, and shorter than the
// declared length. Were the read to succeed, each one would bind cleanly, so the errors the
// test asserts can only come from the read failure — not from a parser or conversion error
// standing in for it.
func truncatingServer() *httptest.Server {
	type canned struct {
		status      int
		contentType string
		body        string
	}
	routes := map[string]canned{
		"/trunc-json": {200, "application/json", `{"name": "Alice"}`},
		"/trunc-text": {200, "text/plain", "red"},
		"/trunc-xml":  {200, "application/xml", "<a>1</a>"},
		"/trunc-blob": {200, "application/octet-stream", "\x01\x02"},
		"/trunc-form": {200, "application/x-www-form-urlencoded", "a=1"},
		"/trunc-404":  {404, "application/json", `{"error": "gone"}`},
		// A () target discards the body without inspecting its Content-Type, so it must
		// still surface the read failure rather than silently returning ().
		"/trunc-nil": {200, "text/plain", "ignored"},
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", route.contentType)
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(route.status)
		_, _ = fmt.Fprint(w, route.body)
	}))
	// The short write is deliberate; keep the transport's complaint out of the test output.
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.Start()
	return server
}

// TestHttpClientDataBindReadFailureLocal covers the read-failure path through each builder,
// a narrow target (where the failure must survive the conversion step), and a status error.
func TestHttpClientDataBindReadFailureLocal(t *testing.T) {
	server := truncatingServer()
	defer server.Close()
	runExtern(t, fileCase("http-client-databind-read-failure-local-v"), newHTTPPal(rewriteClient(server.URL)), nil)
}
