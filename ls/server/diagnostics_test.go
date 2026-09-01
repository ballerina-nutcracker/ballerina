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
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func newCoreServices() (*workspace.ProjectService, *compile.CompilationService, *event.Bus) {
	platform, _ := palnative.NewPlatform()
	bus := event.New()
	projects := workspace.New(platform, bus)
	return projects, compile.New(projects, bus, compile.WithDebounce(0)), bus
}

type pipeTransport struct {
	reader io.Reader
	writer io.Writer
}

func (t pipeTransport) Read(p []byte) (int, error)  { return t.reader.Read(p) }
func (t pipeTransport) Write(p []byte) (int, error) { return t.writer.Write(p) }

type testSession struct {
	t       *testing.T
	server  *Server
	clientW io.Writer
	clientR *bufio.Reader
	done    chan error
}

func newTestSession(t *testing.T, versionSupport bool) *testSession {
	t.Helper()
	serverRead, clientWrite := io.Pipe()
	clientRead, serverWrite := io.Pipe()
	transport := pipeTransport{reader: serverRead, writer: serverWrite}
	projects, compiler, bus := newCoreServices()
	session := &testSession{
		t:       t,
		server:  New(transport, projects, compiler, bus),
		clientW: clientWrite,
		clientR: bufio.NewReader(clientRead),
		done:    make(chan error, 1),
	}
	t.Cleanup(func() { compiler.Shutdown(); bus.Close() })
	go func() { session.done <- session.server.Serve() }()
	session.initialize(versionSupport)
	return session
}

func (s *testSession) initialize(versionSupport bool) {
	s.t.Helper()
	caps := protocol.ClientCapabilities{}
	textDoc := protocol.TextDocumentClientCapabilities{}
	pubDiag := protocol.PublishDiagnosticsClientCapabilities{}
	if versionSupport {
		pubDiag.VersionSupport = protocol.NewOptional(true)
	}
	textDoc.PublishDiagnostics = protocol.NewOptional(pubDiag)
	caps.TextDocument = protocol.NewOptional(textDoc)
	s.sendRequest("1", "initialize", protocol.InitializeParams{
		Capabilities: caps,
		ProcessID:    protocol.NullNullable[protocol.OrInitializeParamsProcessId](),
		RootURI:      protocol.NullNullable[protocol.OrInitializeParamsRootUri](),
	})
	response := s.receive(s.t)
	if response.Method != "" || len(response.ID) == 0 {
		s.t.Fatalf("expected initialize response, got %+v", response)
	}
}

func (s *testSession) sendRequest(id, method string, params any) {
	s.t.Helper()
	encoded, err := json.Marshal(params)
	if err != nil {
		s.t.Fatalf("marshal params: %v", err)
	}
	if err := protocol.WriteMessage(s.clientW, protocol.Message{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"` + id + `"`),
		Method:  method,
		Params:  encoded,
	}); err != nil {
		s.t.Fatalf("write request: %v", err)
	}
}

func (s *testSession) sendNotification(method string, params any) {
	s.t.Helper()
	encoded, err := json.Marshal(params)
	if err != nil {
		s.t.Fatalf("marshal params: %v", err)
	}
	if err := protocol.WriteMessage(s.clientW, protocol.Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  encoded,
	}); err != nil {
		s.t.Fatalf("write notification: %v", err)
	}
}

func (s *testSession) receive(t *testing.T) protocol.Message {
	t.Helper()
	message, err := protocol.ReadMessage(s.clientR)
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	return message
}

func (s *testSession) close() {
	if c, ok := s.clientW.(io.Closer); ok {
		_ = c.Close()
	}
	<-s.done
}

func decodePublishDiagnostics(t *testing.T, message protocol.Message) protocol.PublishDiagnosticsParams {
	t.Helper()
	if message.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("method = %q, want textDocument/publishDiagnostics", message.Method)
	}
	var params protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	return params
}

func TestPublishDiagnosticsAfterOpenChangeClose(t *testing.T) {
	session := newTestSession(t, true)
	defer session.close()

	const uri = "file:///workspace/main.bal"
	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "ballerina",
			Version:    1,
			Text:       "public function main() {\n    int x = 1;\n}\n",
		},
	})

	published := decodePublishDiagnostics(t, session.receive(t))
	if published.URI != uri {
		t.Fatalf("URI = %q, want %q", published.URI, uri)
	}
	if version, ok := published.Version.Value(); !ok || version != 1 {
		t.Fatalf("Version = %v, want 1", published.Version)
	}
	if len(published.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(published.Diagnostics))
	}
	diag := published.Diagnostics[0]
	severity, _ := diag.Severity.Value()
	if severity != protocol.DiagnosticSeverityError {
		t.Errorf("severity = %v, want SeverityError", severity)
	}
	source, _ := diag.Source.Value()
	if source != "ballerina" {
		t.Errorf("source = %q, want ballerina", source)
	}
	code, _ := diag.Code.Value()
	codeStr, _ := code.String()
	if codeStr != "SEMANTIC_ERROR" {
		t.Errorf("code = %q, want SEMANTIC_ERROR", codeStr)
	}
	message, _ := diag.Message.String()
	if message == "" {
		t.Error("message is empty")
	}
	wantStart := protocol.Position{Line: 1, Character: 4}
	wantEnd := protocol.Position{Line: 1, Character: 14}
	if diag.Range.Start != wantStart {
		t.Errorf("range start = %+v, want %+v", diag.Range.Start, wantStart)
	}
	if diag.Range.End != wantEnd {
		t.Errorf("range end = %+v, want %+v", diag.Range.End, wantEnd)
	}

	session.sendNotification("textDocument/didChange", protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: 2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			protocol.NewTextDocumentContentChangeEventTextDocumentContentChangePartial(protocol.TextDocumentContentChangePartial{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 3, Character: 0},
				},
				Text: "public function main() {}\n",
			}),
		},
	})

	cleared := decodePublishDiagnostics(t, session.receive(t))
	if cleared.URI != uri {
		t.Fatalf("cleared URI = %q, want %q", cleared.URI, uri)
	}
	if clearedVersion, ok := cleared.Version.Value(); !ok || clearedVersion != 2 {
		t.Fatalf("cleared version = %v, want 2", cleared.Version)
	}
	if len(cleared.Diagnostics) != 0 {
		t.Fatalf("cleared diagnostics = %d, want 0", len(cleared.Diagnostics))
	}

	session.sendNotification("textDocument/didClose", protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	onCloseMessage := session.receive(t)
	onClose := decodePublishDiagnostics(t, onCloseMessage)
	var onCloseParams struct {
		Diagnostics json.RawMessage `json:"diagnostics"`
	}
	if err := json.Unmarshal(onCloseMessage.Params, &onCloseParams); err != nil {
		t.Fatalf("unmarshal close params: %v", err)
	}
	if string(onCloseParams.Diagnostics) != "[]" {
		t.Fatalf("close diagnostics JSON = %s, want []", onCloseParams.Diagnostics)
	}
	if onClose.URI != uri {
		t.Fatalf("close URI = %q, want %q", onClose.URI, uri)
	}
	if onClose.Version.IsSet() {
		t.Fatalf("close version = %v, want not set", onClose.Version)
	}
	if len(onClose.Diagnostics) != 0 {
		t.Fatalf("close diagnostics = %d, want 0", len(onClose.Diagnostics))
	}
}

func TestPublishDiagnosticsOmitsVersionWithoutSupport(t *testing.T) {
	session := newTestSession(t, false)
	defer session.close()

	const uri = "file:///workspace/main.bal"
	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "ballerina",
			Version:    7,
			Text:       "public function main() {}\n",
		},
	})
	published := decodePublishDiagnostics(t, session.receive(t))
	if published.Version.IsSet() {
		t.Fatalf("Version = %v, want not set without versionSupport", published.Version)
	}
	if len(published.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %d, want 0", len(published.Diagnostics))
	}
}

func TestPublishDiagnosticsIgnoresNonFileURI(t *testing.T) {
	session := newTestSession(t, true)
	defer session.close()

	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "untitled:Untitled-1",
			LanguageID: "ballerina",
			Version:    1,
			Text:       "public function main() {\n    int x = 1;\n}\n",
		},
	})
	const uri = "file:///workspace/main.bal"
	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "ballerina",
			Version:    1,
			Text:       "public function main() {}\n",
		},
	})
	published := decodePublishDiagnostics(t, session.receive(t))
	if published.URI != uri {
		t.Fatalf("URI = %q, want the file: document only", published.URI)
	}
}

func TestPublishDiagnosticsTransportErrorReachesServe(t *testing.T) {
	serverRead, clientWrite := io.Pipe()
	clientRead, serverWrite := io.Pipe()
	transport := pipeTransport{reader: serverRead, writer: serverWrite}
	projects, compiler, bus := newCoreServices()
	server := New(transport, projects, compiler, bus)
	done := make(chan error, 1)
	go func() { done <- server.Serve() }()
	t.Cleanup(func() { compiler.Shutdown(); bus.Close() })

	session := &testSession{t: t, server: server, clientW: clientWrite, clientR: bufio.NewReader(clientRead), done: done}
	session.initialize(true)

	if err := serverWrite.Close(); err != nil {
		t.Fatalf("close server write: %v", err)
	}
	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///workspace/main.bal",
			LanguageID: "ballerina",
			Version:    1,
			Text:       "public function main() {}\n",
		},
	})
	// Under ticket 09 didOpen no longer writes synchronously; publishDiagnostics
	// arrives out-of-band via the CE subscriber. Closing the client's write end
	// (EOF to the server) lets Serve return. The async CE publish, running on a
	// drainer goroutine against the already-closed server write end, must not
	// crash the server; its write error is contained by writeMu + the engine's
	// shutdown drain.
	_ = clientWrite.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v, want nil (clean EOF; async write error contained)", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return on EOF")
	}
	_ = clientRead.Close()
}

// TestCancelRequestRoutesAsNotification verifies the $/cancelRequest mapping
// (design branch 3) at the protocol/server routing boundary: it is handled as
// a notification (no response is written), it does not crash the dispatch
// loop, and the server continues processing subsequent notifications. The
// semantic effect (CE-E3 gating) is covered by the compile-engine test.
func TestCancelRequestRoutesAsNotification(t *testing.T) {
	session := newTestSession(t, true)
	defer session.close()

	// $/cancelRequest before any document: Cancel applies to zero active roots
	// (a no-op) and must not produce a response or break the dispatch loop.
	session.sendNotification("$/cancelRequest", protocol.CancelParams{
		ID: protocol.NewOrCancelParamsIdString("1"),
	})

	const uri = "file:///workspace/cancel.bal"
	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "ballerina",
			Version:    1,
			Text:       "public function main() {\n    int x = 1;\n}\n",
		},
	})

	// The first message the client receives must be the publishDiagnostics for
	// the didOpen — proving $/cancelRequest wrote no response and dispatch
	// continued normally.
	published := decodePublishDiagnostics(t, session.receive(t))
	if published.URI != uri {
		t.Fatalf("first message URI = %q, want %q (a $/cancelRequest response must not precede it)", published.URI, uri)
	}
	if published.Version.IsSet() {
		ver, _ := published.Version.Value()
		if ver != 1 {
			t.Fatalf("version = %v, want 1", ver)
		}
	}
}

// TestDedupNotStaleAfterEviction verifies the CE publish dedup does not
// suppress a reloaded source root's diagnostics behind a stale high-water
// mark left by an evicted previous incarnation. With WithMaxProjects(1):
// open /a (gen 1, publish), close /a (background), open /b (evicts /a,
// clearing lastPublished["/a"]), then reopen /a — its gen-1 reload must
// still publish. Without the ProjectEvicted clear (or with a <= guard) the
// reload's first publish would be dropped.
func TestDedupNotStaleAfterEviction(t *testing.T) {
	serverRead, clientWrite := io.Pipe()
	clientRead, serverWrite := io.Pipe()
	transport := pipeTransport{reader: serverRead, writer: serverWrite}
	platform, _ := palnative.NewPlatform()
	bus := event.New()
	projects := workspace.New(platform, bus, workspace.WithMaxProjects(1))
	compiler := compile.New(projects, bus, compile.WithDebounce(0))
	server := New(transport, projects, compiler, bus)
	done := make(chan error, 1)
	go func() { done <- server.Serve() }()
	t.Cleanup(func() { compiler.Shutdown(); bus.Close(); _ = clientWrite.Close(); _ = clientRead.Close(); <-done })

	session := &testSession{t: t, server: server, clientW: clientWrite, clientR: bufio.NewReader(clientRead), done: done}
	session.initialize(true)

	const (
		uriA = "file:///a/dedup-a.bal"
		uriB = "file:///b/dedup-b.bal"
	)
	text := "public function main() {\n int x = 1;\n}\n"

	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uriA, LanguageID: "ballerina", Version: 1, Text: text},
	})
	_ = decodePublishDiagnostics(t, session.receive(t)) // gen 1 for /a

	session.sendNotification("textDocument/didClose", protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uriA},
	})
	_ = decodePublishDiagnostics(t, session.receive(t)) // clear for /a

	// Open /b: with WithMaxProjects(1) this evicts /a (the only, now-background,
	// root), publishing ProjectEvicted and clearing the dedup mark for /a.
	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uriB, LanguageID: "ballerina", Version: 1, Text: text},
	})
	_ = decodePublishDiagnostics(t, session.receive(t)) // gen 1 for /b

	// Reopen /a: the root reloads at gen 1. The dedup must NOT suppress this
	// behind the stale lastPublished["/a"] == 1 from the first incarnation.
	session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uriA, LanguageID: "ballerina", Version: 1, Text: text},
	})

	select {
	case msg := <-session.recv():
		published := decodePublishDiagnostics(t, msg)
		if published.URI != uriA {
			t.Fatalf("reopen publish URI = %q, want %q", published.URI, uriA)
		}
		if len(published.Diagnostics) == 0 {
			t.Fatal("reopen publish dropped the reload's diagnostics (dedup stale after eviction)")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reopen of evicted root produced no publishDiagnostics (dedup suppressed the reload)")
	}
}

// recv returns a channel that yields the next framed message, so a test can
// select on it with a timeout instead of blocking forever when a publish is
// wrongly suppressed.
func (s *testSession) recv() <-chan protocol.Message {
	ch := make(chan protocol.Message, 1)
	go func() { ch <- s.receive(s.t) }()
	return ch
}
