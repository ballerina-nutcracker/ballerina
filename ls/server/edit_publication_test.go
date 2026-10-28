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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestEditPublicationBoundary(t *testing.T) {
	for _, versionSupport := range []bool{false, true} {
		for _, reopen := range []bool{false, true} {
			t.Run(fmt.Sprintf("versions=%v/reopen=%v", versionSupport, reopen), func(t *testing.T) {
				projects, compiler, bus := newCoreServices()
				defer bus.Close()
				defer compiler.Shutdown()
				path := filepath.Join(t.TempDir(), "main.bal")
				uri, _ := workspace.FileURIFromPath(path)
				clientURI := strings.Replace(uri.String(), "main.bal", "%6dain.bal", 1)
				oldText := "public function main() { string face = \"😀\"; missing(face); }\n"
				newText := "public function main() {\n    other();\n}\n"
				if _, err := projects.Apply(context.Background(), workspace.DocumentChange{
					Kind: workspace.ChangeOpen, URI: uri, ClientURI: clientURI, Text: oldText, Version: 2,
				}); err != nil {
					t.Fatal(err)
				}
				compiler.Flush()
				diags, gen, ok := compiler.DiagnosticsFor(path)
				if !ok || len(diags[uri]) != 1 {
					t.Fatalf("expected one controlled error, got %v", diags[uri])
				}
				transport := &diagnosticBarrierTransport{entered: make(chan struct{}), release: make(chan struct{})}
				srv := New(transport, projects, compiler, nil)
				srv.versionSupport = versionSupport
				defer transport.releaseWrite()
				published := make(chan struct{})
				go func() { srv.publishDocumentDiagnostics(path, gen, uri, diags[uri]); close(published) }()
				select {
				case <-transport.entered:
				case <-time.After(receiveTimeout):
					t.Fatal("publication did not reach transport barrier")
				}
				mutate := func(version int32, text string) error {
					if reopen {
						params, _ := json.Marshal(protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{
							URI: clientURI, Text: text, Version: version, LanguageID: "ballerina",
						}})
						return srv.handleDidOpen(context.Background(), params)
					}
					params := json.RawMessage(fmt.Sprintf(`{"textDocument":{"uri":%q,"version":%d},"contentChanges":[{"text":%q}]}`, clientURI, version, text))
					return srv.handleDidChange(context.Background(), params)
				}
				edited := make(chan error, 1)
				go func() {
					for i, text := range []string{newText, oldText, newText} {
						if err := mutate(int32(i+3), text); err != nil {
							edited <- err
							return
						}
					}
					edited <- nil
				}()
				contender := "(*Server).handleDidChange"
				if reopen {
					contender = "(*Server).handleDidOpen"
				}
				waitForDiagnosticContender(t, contender)
				if snap, _ := projects.Snapshot(uri); snap.Version != 2 || snap.Text != oldText {
					t.Fatal("document mutated before admitted publication finished")
				}
				transport.releaseWrite()
				select {
				case <-published:
				case <-time.After(receiveTimeout):
					t.Fatal("publication did not finish")
				}
				select {
				case err := <-edited:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(receiveTimeout):
					t.Fatal("edit did not finish")
				}
				compiler.Flush()
				srv.publishDocumentDiagnostics(path, gen, uri, diags[uri])
				newDiags, newGen, ok := compiler.DiagnosticsFor(path)
				if !ok || newGen == gen || len(newDiags[uri]) != 1 {
					t.Fatalf("missing current diagnostics: %v", newDiags)
				}
				srv.publishDocumentDiagnostics(path, newGen, uri, newDiags[uri])
				reader := bufio.NewReader(bytes.NewReader(transport.output.Bytes()))
				for i, expected := range []struct {
					version    int32
					start, end protocol.Position
				}{
					{2, protocol.Position{Line: 0, Character: 45}, protocol.Position{Line: 0, Character: 58}},
					{5, protocol.Position{Line: 1, Character: 4}, protocol.Position{Line: 1, Character: 11}},
				} {
					message, err := protocol.ReadMessage(reader)
					if err != nil {
						t.Fatal(err)
					}
					got := decodePublishDiagnostics(t, message)
					version, hasVersion := got.Version.Value()
					if got.URI != clientURI || hasVersion != versionSupport || (hasVersion && version != expected.version) {
						t.Fatalf("publication %d metadata: %+v", i, got)
					}
					if len(got.Diagnostics) != 1 || got.Diagnostics[0].Range.Start != expected.start || got.Diagnostics[0].Range.End != expected.end {
						t.Fatalf("publication %d range: %+v", i, got.Diagnostics)
					}
				}
				if _, err := protocol.ReadMessage(reader); err != io.EOF {
					t.Fatalf("obsolete diagnostics were not discarded: %v", err)
				}
			})
		}
	}
}
