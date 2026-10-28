// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestClientDiagnosticURI(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		uri     string
		openDir func(string) fs.FS
	}{
		{"native", strings.ReplaceAll(tempFileURI(t, "a@b.bal"), "@", "%40"), os.DirFS},
		// The injected filesystem exercises drive-URI spelling, not native Windows I/O.
		{"encoded-drive-synthetic", "file:///c%3A/workspace/a%40b.bal", func(string) fs.FS { return fstest.MapFS{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			canonical, err := workspace.NewFileURI(tc.uri)
			if err != nil {
				t.Fatal(err)
			}
			if canonical.String() == tc.uri {
				t.Fatal("test URI must differ from canonical identity")
			}
			var input, output bytes.Buffer
			send := func(method string, params any) {
				t.Helper()
				raw, err := json.Marshal(params)
				if err != nil {
					t.Fatal(err)
				}
				message := protocol.Message{JSONRPC: "2.0", Method: method, Params: raw}
				if method == "initialize" {
					message.ID = json.RawMessage("1")
				}
				if err := protocol.WriteMessage(&input, message); err != nil {
					t.Fatal(err)
				}
			}
			send("initialize", map[string]any{})
			send("textDocument/didOpen", protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{
				URI: tc.uri, LanguageID: "ballerina", Version: 1, Text: "public function main() {\n    int unused = 1;\n}\n",
			}})
			send("$test/flush", nil)
			send("textDocument/didChange", protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{URI: tc.uri, Version: 2},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					protocol.NewTextDocumentContentChangeEventTextDocumentContentChangePartial(protocol.TextDocumentContentChangePartial{
						Range: protocol.Range{}, Text: "// shifted\n",
					}),
				},
			})
			send("$test/flush", nil)
			send("textDocument/didChange", protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{URI: canonical.String(), Version: 3},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					protocol.NewTextDocumentContentChangeEventTextDocumentContentChangeWholeDocument(protocol.TextDocumentContentChangeWholeDocument{Text: ""}),
				},
			})
			send("$test/flush", nil)
			send("textDocument/didClose", protocol.DidCloseTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: canonical.String()}})
			send("$test/flush", nil)

			bus := event.New()
			defer bus.Close()
			projects := workspace.New(tc.openDir, bus)
			compiler := compile.New(projects, bus, compile.WithDebounce(0))
			defer compiler.Shutdown()
			srv := New(pipeTransport{reader: &input, writer: &output}, projects, compiler, bus)
			defer srv.Flush()
			if err := srv.Serve(); err != nil {
				t.Fatal(err)
			}
			reader := bufio.NewReader(&output)
			count := 0
			for {
				message, err := protocol.ReadMessage(reader)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				if message.Method == "" {
					continue
				}
				params := decodePublishDiagnostics(t, message)
				if params.URI != tc.uri {
					t.Errorf("publication %d URI = %q, want exact client URI %q", count, params.URI, tc.uri)
				}
				wantCount := 1
				if count >= 2 {
					wantCount = 0
				}
				if len(params.Diagnostics) != wantCount {
					t.Errorf("publication %d diagnostics = %v, want %d", count, params.Diagnostics, wantCount)
				}
				count++
			}
			if count != 4 {
				t.Fatalf("got %d diagnostic publications, want 4", count)
			}
		})
	}
}
