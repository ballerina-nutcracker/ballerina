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
	"net/http/httptest"
	"testing"

	"ballerina/semtypes"
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
	tc := semtypes.ContextFrom(semtypes.CreateTypeEnv())
	r := httptest.NewRequest("POST", "/", failingReadCloser{})
	r.ContentLength = 10 // within eagerBufferThreshold, triggers io.ReadAll

	req, err := buildRequestFromHTTP(tc, r)
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
