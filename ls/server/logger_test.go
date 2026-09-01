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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

// newLoggingBatchServer builds a batch-driven Server (see cancellation_test.go)
// wired to an observability.Logger backed by a captured stderr buffer, so
// tests can assert a request-scoped logger reached every dispatch path from
// handleMessage.
func newLoggingBatchServer(t *testing.T) (*Server, *batchTransport, *bytes.Buffer) {
	t.Helper()
	platform, _ := palnative.NewPlatform()
	bus := event.New()
	projects := workspace.New(platform, bus)
	compiler := compile.New(projects, bus, compile.WithDebounce(0))
	t.Cleanup(func() { compiler.Shutdown(); bus.Close() })
	var logBuf bytes.Buffer
	logger := observability.New(pal.IO{Stderr: func(p []byte) (int, error) { return logBuf.Write(p) }}, pal.OS{})
	bt := &batchTransport{}
	return New(bt, projects, compiler, bus, WithLogger(logger)), bt, &logBuf
}

// TestRequestScopedLoggerReachesEveryDispatchPath verifies a request-scoped
// logger carrying method (+id for requests) is attached at handleMessage for
// every category of inbound message: a lifecycle request (initialize), a
// notification, and a tracked request — not just the two methods
// dispatchTracked itself covers.
func TestRequestScopedLoggerReachesEveryDispatchPath(t *testing.T) {
	srv, bt, logBuf := newLoggingBatchServer(t)
	_, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"b1"`, "$pal/blockRequest", map[string]any{}),
		notifMessage("$pal/releaseRequest", map[string]any{"id": "b1"}),
		notifMessage("$pal/flushRequests", map[string]any{}),
		reqMessage(`"sh"`, "shutdown", nil),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	out := logBuf.String()

	// Lifecycle and tracked requests: method+id both present.
	for _, want := range []string{
		`method=initialize`, `id="\"1\""`,
		`method=$pal/blockRequest`, `id="\"b1\""`,
		`method=shutdown`, `id="\"sh\""`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q; got:\n%s", want, out)
		}
	}
	// Notification: method present (notifications carry no id).
	if !strings.Contains(out, "method=initialized") {
		t.Errorf("log output missing notification method=initialized; got:\n%s", out)
	}
}
