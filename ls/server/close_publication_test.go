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
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

type diagnosticBarrierTransport struct {
	output  bytes.Buffer
	entered chan struct{}
	release chan struct{}
	first   sync.Once
	unblock sync.Once
}

func (t *diagnosticBarrierTransport) Read([]byte) (int, error) { return 0, io.EOF }
func (t *diagnosticBarrierTransport) Write(p []byte) (int, error) {
	t.first.Do(func() { close(t.entered); <-t.release })
	return t.output.Write(p)
}
func (t *diagnosticBarrierTransport) releaseWrite() { t.unblock.Do(func() { close(t.release) }) }

func waitForDiagnosticContender(t *testing.T, function string) {
	t.Helper()
	deadline := time.After(receiveTimeout)
	stack := make([]byte, 1<<20)
	for {
		n := runtime.Stack(stack, true)
		for _, goroutine := range strings.Split(string(stack[:n]), "\n\n") {
			if strings.Contains(goroutine, function) && strings.Contains(goroutine, "[sync.Mutex.Lock]") {
				return
			}
		}
		select {
		case <-deadline:
			t.Fatalf("no blocked diagnostic contender in %s", function)
		default:
			runtime.Gosched()
		}
	}
}

func TestClosePublicationBoundary(t *testing.T) {
	t.Parallel()
	for _, closeFirst := range []bool{false, true} {
		name := "publication-admitted"
		if closeFirst {
			name = "close-admitted"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			projects, compiler, bus := newCoreServices()
			defer bus.Close()
			defer compiler.Shutdown()
			path := filepath.Join(t.TempDir(), "main.bal")
			text := "public function main() { missing(); }\n"
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			uri, _ := workspace.FileURIFromPath(path)
			if _, err := projects.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeOpen, URI: uri, Text: text, Version: 1}); err != nil {
				t.Fatal(err)
			}
			compiler.Flush()
			diags, gen, ok := compiler.DiagnosticsFor(path)
			if !ok || len(diags[uri]) == 0 {
				t.Fatal("missing error for controlled publication")
			}
			transport := &diagnosticBarrierTransport{entered: make(chan struct{}), release: make(chan struct{})}
			srv := New(transport, projects, compiler, nil)
			defer srv.Flush()
			defer transport.releaseWrite()
			closeParams, _ := json.Marshal(protocol.DidCloseTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri.String()}})
			closed := make(chan error, 1)
			published := make(chan struct{})
			publish := func() { srv.publishDocumentDiagnostics(path, gen, uri, diags[uri]); close(published) }
			closeDocument := func() { closed <- srv.handleDidClose(closeParams) }
			if closeFirst {
				go closeDocument()
			} else {
				go publish()
			}
			select {
			case <-transport.entered:
			case <-time.After(receiveTimeout):
				t.Fatal("diagnostic write did not reach barrier")
			}
			if srv.diagnosticMu.TryLock() {
				srv.diagnosticMu.Unlock()
				t.Error("diagnostic boundary was released before its transport write")
			}
			if closeFirst {
				go publish()
				waitForDiagnosticContender(t, "(*Server).publishDocumentDiagnostics")
			} else {
				go closeDocument()
				waitForDiagnosticContender(t, "(*Server).handleDidClose")
				if _, open := projects.Snapshot(uri); !open {
					t.Error("close removed buffer while admitted publication was blocked")
				}
			}
			transport.releaseWrite()
			if err := <-closed; err != nil {
				t.Fatal(err)
			}
			<-published
			srv.Flush()
			reader := bufio.NewReader(bytes.NewReader(transport.output.Bytes()))
			cleared := false
			count := 0
			for {
				message, err := protocol.ReadMessage(reader)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				params := decodePublishDiagnostics(t, message)
				count++
				if len(params.Diagnostics) == 0 {
					cleared = true
				} else if cleared {
					t.Fatal("nonempty diagnostics followed close clear")
				}
			}
			expectedCount := 2
			if closeFirst {
				expectedCount = 1
			}
			if !cleared || count != expectedCount {
				t.Fatalf("diagnostic frames=%d, cleared=%v", count, cleared)
			}
		})
	}
}
