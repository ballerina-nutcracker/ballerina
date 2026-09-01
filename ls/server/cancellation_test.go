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
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

// batchTransport is a buffered transport: the driver frames a batch of
// messages into the reader, runs Serve (which returns on EOF), and reads the
// server's framed responses from the writer. Unlike io.Pipe, writes never
// block, so async request goroutines can reply while Serve is still draining
// (via $pal/flushRequests) without deadlocking the driver.
type batchTransport struct {
	reader *bytes.Reader
	writer bytes.Buffer
}

func (t *batchTransport) Read(p []byte) (int, error)  { return t.reader.Read(p) }
func (t *batchTransport) Write(p []byte) (int, error) { return t.writer.Write(p) }

func newBatchServer(t *testing.T) (*Server, *batchTransport) {
	t.Helper()
	platform, _ := palnative.NewPlatform()
	bus := event.New()
	projects := workspace.New(platform, bus)
	compiler := compile.New(projects, bus, compile.WithDebounce(0))
	t.Cleanup(func() { compiler.Shutdown(); bus.Close() })
	bt := &batchTransport{}
	return New(bt, projects, compiler, bus), bt
}

// runBatch frames messages into the transport, runs Serve to EOF, and returns
// the decoded responses plus the Serve error (errServerExited is mapped to nil
// by Serve itself; other lifecycle errors are returned for the test to check).
func runBatch(t *testing.T, srv *Server, bt *batchTransport, messages []protocol.Message) ([]protocol.Message, error) {
	t.Helper()
	var input bytes.Buffer
	for _, m := range messages {
		if err := protocol.WriteMessage(&input, m); err != nil {
			t.Fatalf("frame message: %v", err)
		}
	}
	bt.reader = bytes.NewReader(input.Bytes())
	bt.writer.Reset()
	err := srv.Serve()
	reader := bufio.NewReader(bytes.NewReader(bt.writer.Bytes()))
	var responses []protocol.Message
	for {
		m, rerr := protocol.ReadMessage(reader)
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			t.Fatalf("read response: %v", rerr)
		}
		responses = append(responses, m)
	}
	return responses, err
}

func reqMessage(id, method string, params any) protocol.Message {
	encoded, _ := json.Marshal(params)
	return protocol.Message{JSONRPC: "2.0", ID: json.RawMessage(id), Method: method, Params: encoded}
}

func notifMessage(method string, params any) protocol.Message {
	encoded, _ := json.Marshal(params)
	return protocol.Message{JSONRPC: "2.0", Method: method, Params: encoded}
}

func mustFindResponse(t *testing.T, responses []protocol.Message, id string) protocol.Message {
	t.Helper()
	for _, m := range responses {
		if string(m.ID) == id {
			return m
		}
	}
	t.Fatalf("no response with id %s among %d responses", id, len(responses))
	return protocol.Message{}
}

// TestRegistryEntryRemovedOnCompletion verifies a tracked request's registry
// entry is removed once its goroutine completes, so the id is free to reuse
// and a later $/cancelRequest for it is a lookup miss.
func TestRegistryEntryRemovedOnCompletion(t *testing.T) {
	srv, bt := newBatchServer(t)
	runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"r1"`, "$pal/blockRequest", map[string]any{}),
		notifMessage("$pal/releaseRequest", map[string]any{"id": "r1"}),
		notifMessage("$pal/flushRequests", map[string]any{}),
	})
	if entry := srv.registry.lookup(`"r1"`); entry != nil {
		t.Fatal("registry still has entry for completed request \"r1\"")
	}
}

// TestCancelAfterCompletionIsNoop verifies a $/cancelRequest for an
// already-completed request produces no response: the entry has been removed,
// so the cancel is a lookup miss.
func TestCancelAfterCompletionIsNoop(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"c1"`, "$pal/blockRequest", map[string]any{}),
		notifMessage("$pal/releaseRequest", map[string]any{"id": "c1"}),
		notifMessage("$pal/flushRequests", map[string]any{}),
		notifMessage("$/cancelRequest", map[string]any{"id": "c1"}),
		notifMessage("$pal/flushRequests", map[string]any{}),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2 (initialize + result; stale cancel must be a no-op): %+v", len(responses), responses)
	}
	r := mustFindResponse(t, responses, `"c1"`)
	if r.Error != nil {
		t.Fatalf("completion response = %+v, want result for \"c1\"", r)
	}
}

// TestDuplicateRequestIDRejectedKeepsOriginal verifies a new request reusing
// an in-flight id is rejected with Invalid Request (-32600) and does not
// replace the original registry entry — the original still completes normally.
func TestDuplicateRequestIDRejectedKeepsOriginal(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"d1"`, "$pal/blockRequest", map[string]any{}),
		reqMessage(`"d1"`, "$pal/blockRequest", map[string]any{}), // duplicate
		notifMessage("$pal/releaseRequest", map[string]any{"id": "d1"}),
		notifMessage("$pal/flushRequests", map[string]any{}),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	rejection := mustFindResponse(t, responses, `"d1"`)
	// Two responses share id "d1": the -32600 rejection and the original's
	// result. findResponse returns the first; verify it is the rejection.
	if rejection.Error == nil || rejection.Error.Code != rpcInvalidRequest {
		t.Fatalf("duplicate response = %+v, want Invalid Request -32600", rejection)
	}
	// Registry must be empty after the original completed.
	if entry := srv.registry.lookup(`"d1"`); entry != nil {
		t.Fatal("registry still has entry for \"d1\" after completion")
	}
}

// TestCancelProducesExactlyOneReply verifies a duplicate $/cancelRequest for
// the same in-flight id yields exactly one -32800 reply (the reply guard).
func TestCancelProducesExactlyOneReply(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"e1"`, "$pal/blockRequest", map[string]any{}),
		notifMessage("$/cancelRequest", map[string]any{"id": "e1"}),
		notifMessage("$/cancelRequest", map[string]any{"id": "e1"}),
		notifMessage("$pal/flushRequests", map[string]any{}),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2 (initialize + single -32800): %+v", len(responses), responses)
	}
	r := mustFindResponse(t, responses, `"e1"`)
	if r.Error == nil || r.Error.Code != rpcRequestCancelled {
		t.Fatalf("response = %+v, want RequestCancelled -32800", r)
	}
}

// TestShutdownDrainsTrackedRequests verifies shutdown cancels and drains
// in-flight tracked requests before replying: the blocked request's -32800
// reply precedes the shutdown response.
func TestShutdownDrainsTrackedRequests(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"sd"`, "$pal/blockRequest", map[string]any{}),
		reqMessage(`"sh"`, "shutdown", nil),
		notifMessage("exit", map[string]any{}),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	cancelled := mustFindResponse(t, responses, `"sd"`)
	if cancelled.Error == nil || cancelled.Error.Code != rpcRequestCancelled {
		t.Fatalf("in-flight response = %+v, want RequestCancelled -32800", cancelled)
	}
	sh := mustFindResponse(t, responses, `"sh"`)
	if string(sh.Result) != "null" {
		t.Fatalf("shutdown response = %+v, want null result", sh)
	}
}

// TestExitWithoutShutdownDrains verifies exit without shutdown still drains
// in-flight tracked requests (writing their -32800 replies) before returning
// the exit-without-shutdown error.
func TestExitWithoutShutdownDrains(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"xd"`, "$pal/blockRequest", map[string]any{}),
		notifMessage("exit", map[string]any{}),
	})
	if !errors.Is(err, errExitedWithoutShutdown) {
		t.Fatalf("Serve = %v, want %v", err, errExitedWithoutShutdown)
	}
	cancelled := mustFindResponse(t, responses, `"xd"`)
	if cancelled.Error == nil || cancelled.Error.Code != rpcRequestCancelled {
		t.Fatalf("in-flight response = %+v, want RequestCancelled -32800", cancelled)
	}
}

// TestCancelIDKeyDistinguishesStringAndInteger verifies the canonical registry
// key is the raw JSON of the id, so a string "5" and an integer 5 are distinct
// ids: cancelling one does not cancel the other.
func TestCancelIDKeyDistinguishesStringAndInteger(t *testing.T) {
	srv, bt := newBatchServer(t)
	responses, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"5"`, "$pal/blockRequest", map[string]any{}), // string id "5"
		reqMessage(`5`, "$pal/blockRequest", map[string]any{}),   // integer id 5
		notifMessage("$/cancelRequest", map[string]any{"id": "5"}),
		notifMessage("$pal/releaseRequest", map[string]any{"id": 5}),
		notifMessage("$pal/flushRequests", map[string]any{}),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	str := mustFindResponse(t, responses, `"5"`)
	if str.Error == nil || str.Error.Code != rpcRequestCancelled {
		t.Fatalf("string id \"5\" response = %+v, want -32800", str)
	}
	num := mustFindResponse(t, responses, `5`)
	if num.Error != nil {
		t.Fatalf("integer id 5 response = %+v, want result (must not be cancelled by string-id cancel)", num)
	}
}
