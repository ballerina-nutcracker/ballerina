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
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestReceiveFramedMessages(t *testing.T) {
	t.Parallel()
	session := newTestSession(t, true)
	defer session.close()
	for _, version := range []int32{1, 2} {
		session.sendNotification("textDocument/didOpen", protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: tempFileURI(t, "main.bal"), LanguageID: "ballerina", Version: version,
				Text: "public function main() {}\n",
			},
		})
		message := session.receive(t, "textDocument/publishDiagnostics", "")
		if message.Method != "textDocument/publishDiagnostics" || len(message.ID) != 0 {
			t.Fatalf("notification envelope changed: %+v", message)
		}
		var params protocol.PublishDiagnosticsParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			t.Fatal(err)
		}
		if got, ok := params.Version.Value(); !ok || got != version {
			t.Fatalf("version = %v, want %d", params.Version, version)
		}
	}
}

func TestReceiveFailures(t *testing.T) {
	for _, scenario := range []struct {
		name string
		want string
	}{
		{"notification", "timeout waiting for textDocument/publishDiagnostics"},
		{"response", "timeout waiting for shutdown (request ID \"2\")"},
		{"read-error", "read message for textDocument/publishDiagnostics: EOF"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReceiveFailureProcess$", "-test.timeout=12s", "-test.v")
			cmd.Env = append(os.Environ(), "LS_RECEIVE_FAILURE="+scenario.name)
			start := time.Now()
			output, err := cmd.CombinedOutput()
			elapsed := time.Since(start)
			if ctx.Err() != nil || err == nil {
				t.Fatalf("child result = %v (%v):\n%s", err, ctx.Err(), output)
			}
			for _, want := range []string{scenario.want, "receive cleanup complete"} {
				if !bytes.Contains(output, []byte(want)) {
					t.Errorf("child output missing %q:\n%s", want, output)
				}
			}
			if bytes.Contains(output, []byte("panic:")) || bytes.Contains(output, []byte("WARNING: DATA RACE")) {
				t.Errorf("child crashed or raced:\n%s", output)
			}
			if scenario.name == "read-error" {
				if elapsed >= 5*time.Second || bytes.Contains(output, []byte("timeout waiting")) {
					t.Errorf("read error was not immediate (%v):\n%s", elapsed, output)
				}
			} else if elapsed < 5*time.Second || elapsed >= 10*time.Second {
				t.Errorf("timeout took %v, want approximately five seconds:\n%s", elapsed, output)
			}
		})
	}
}

func TestReceiveFailureProcess(t *testing.T) {
	scenario := os.Getenv("LS_RECEIVE_FAILURE")
	if scenario == "" {
		t.Skip("isolated deliberate receive failure")
	}
	session := newTestSession(t, true)
	t.Cleanup(func() {
		session.close()
		stacks := make([]byte, 1<<20)
		n := runtime.Stack(stacks, true)
		if bytes.Contains(stacks[:n], []byte("protocol.ReadMessage(")) ||
			bytes.Contains(stacks[:n], []byte("(*testSession).receive.func")) {
			t.Fatalf("receive goroutine survived cleanup:\n%s", stacks[:n])
		}
		t.Log("receive cleanup complete")
	})
	if scenario == "read-error" {
		session.clientR = bufio.NewReader(strings.NewReader(""))
	}
	// No document is opened and no shutdown is sent: suppress the expected message.
	if scenario == "response" {
		_ = session.receive(t, "shutdown", `"2"`)
	} else {
		_ = session.receive(t, "textDocument/publishDiagnostics", "")
	}
	t.Fatal("receive unexpectedly returned")
}
