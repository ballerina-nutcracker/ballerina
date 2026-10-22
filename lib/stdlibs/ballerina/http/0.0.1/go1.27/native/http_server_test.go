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

package native

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// failingReadCloser errors out of Read before delivering the number of bytes
// its Content-Length promised, simulating a client that disconnects mid-body.
type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
func (failingReadCloser) Close() error             { return nil }

// TestBuildRequestFromHTTPBodyReadError covers the error return of
// buildRequestFromHTTP's eager-buffer path (dispatchRequest's caller maps this
// to an HTTP 400). This is unreachable through the module's own http:Client,
// which never sends a body shorter than its declared Content-Length, so it
// isn't exercisable from a corpus/.bal test and is covered here directly.
func TestBuildRequestFromHTTPBodyReadError(t *testing.T) {
	r := httptest.NewRequest("POST", "/", failingReadCloser{})
	r.ContentLength = 10 // within eagerBufferThreshold, triggers io.ReadAll

	req, err := buildRequestFromHTTP(r)
	if err == nil {
		t.Fatal("buildRequestFromHTTP: expected an error for a body that fails mid-read, got nil")
	}
	if req != nil {
		t.Fatalf("buildRequestFromHTTP: expected a nil request alongside the error, got %v", req)
	}
	if err.Error() != "connection reset" {
		t.Fatalf("buildRequestFromHTTP: expected the underlying read error to propagate, got %q", err)
	}
}

// TestBufferedRelayBodyReadError covers a relayed body short enough to be buffered whose backend
// fails mid-body: the read error fails the response, before or after its headers are sent. In
// jBallerina no module service backend makes such a relay fail, so it is not a corpus test.
func TestBufferedRelayBodyReadError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		received string
		sent     bool
	}{{"", false}, {"abcdefg", true}} {
		body := io.NopCloser(io.MultiReader(strings.NewReader(tc.received), failingReadCloser{}))
		holder := eagerBufferResponse(map[string][]string{"Content-Length": {"100"}}, body)
		rec := httptest.NewRecorder()
		dw := &deferredHeaderWriter{ResponseWriter: rec, status: http.StatusOK}
		if err := holder.writeStream(dw, false, 0); err == nil {
			t.Fatalf("received %q: expected the read error, got nil", tc.received)
		}
		if dw.sent != tc.sent {
			t.Fatalf("received %q: headers sent = %v, want %v", tc.received, dw.sent, tc.sent)
		}
		if rec.Body.String() != tc.received {
			t.Fatalf("received %q: wrote %q", tc.received, rec.Body.String())
		}
	}
}

// TestWriteBodyFailure checks the bare 500 drops the headers the service set.
func TestWriteBodyFailure(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Svc", "set")
	rec.Header().Set("Content-Length", "100")
	writeBodyFailure(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := rec.Header(); len(got) != 2 || got.Get("Content-Length") != "0" || got.Get("Connection") != "close" {
		t.Fatalf("headers = %v, want only Content-Length: 0 and Connection: close", got)
	}
}
