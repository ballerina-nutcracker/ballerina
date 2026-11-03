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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func newLoggingBatchServer(t *testing.T) (*Server, *batchTransport, *bytes.Buffer) {
	t.Helper()
	bus := event.New()
	projects := workspace.New(os.DirFS, bus)
	compiler := compile.New(projects, bus, compile.WithDebounce(0))
	t.Cleanup(func() { compiler.Shutdown(); bus.Close() })
	var logBuf bytes.Buffer
	logger := observability.New(&logBuf)
	bt := &batchTransport{}
	return New(bt, projects, compiler, bus, WithLogger(logger)), bt, &logBuf
}

func TestHostSelectedFileOutputReplacesStderr(t *testing.T) {
	t.Parallel()
	var stderr, fileOutput bytes.Buffer
	selected := io.Writer(&fileOutput)
	bus := event.New()
	defer bus.Close()
	service := workspace.New(os.DirFS, bus)
	compiler := compile.New(service, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	transport := &batchTransport{}
	srv := New(transport, service, compiler, bus, WithLogger(observability.New(selected)))
	responses, err := runBatch(t, srv, transport, []protocol.Message{
		reqMessage(`1`, "initialize", map[string]any{}),
		reqMessage(`2`, "shutdown", nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(responses) != 2 || string(responses[0].ID) != "1" || string(responses[1].ID) != "2" {
		t.Fatalf("invalid protocol output: %+v", responses)
	}
	if stderr.Len() != 0 || !strings.Contains(fileOutput.String(), "method=shutdown") {
		t.Fatalf("stderr=%q selected output=%q", stderr.String(), fileOutput.String())
	}
}

type discoveryErrorFS struct{}

func (discoveryErrorFS) Open(string) (fs.File, error) { return nil, fs.ErrPermission }

func TestDiscoveryErrorsAreLoggedWithoutClientNotifications(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	opener := func(dir string) fs.FS {
		if dir == root {
			return discoveryErrorFS{}
		}
		return os.DirFS(dir)
	}
	bus := event.New()
	defer bus.Close()
	service := workspace.New(opener, bus)
	compiler := compile.New(service, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	var logs bytes.Buffer
	transport := &batchTransport{}
	srv := New(transport, service, compiler, bus, WithLogger(observability.New(&logs)))
	uri, err := workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := workspace.FileURIFromPath(filepath.Join(root, "Ballerina.toml"))
	if err != nil {
		t.Fatal(err)
	}
	responses, err := runBatch(t, srv, transport, []protocol.Message{
		reqMessage(`1`, "initialize", map[string]any{}),
		notifMessage("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri.String(), "languageId": "ballerina", "version": 1, "text": "public function main() {}"}}),
		notifMessage("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri.String(), "version": 2}, "contentChanges": []any{map[string]any{"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 0, "character": 0}}, "text": " "}}}),
		notifMessage("workspace/didChangeWatchedFiles", map[string]any{"changes": []any{map[string]any{"uri": manifest.String(), "type": 1}}}),
		notifMessage("$test/flush", map[string]any{}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(responses) != 1 || string(responses[0].ID) != "1" {
		t.Fatalf("unexpected client output: %+v", responses)
	}
	for _, method := range []string{"textDocument/didOpen", "textDocument/didChange", "workspace/didChangeWatchedFiles"} {
		found := false
		for line := range strings.SplitSeq(logs.String(), "\n") {
			if strings.Contains(line, "level=ERROR") && strings.Contains(line, "method="+method) && strings.Contains(line, "permission denied") {
				found = true
			}
		}
		if !found {
			t.Errorf("missing correlated discovery error for %s: %s", method, logs.String())
		}
	}
}

func TestRequestScopedLoggerReachesEveryDispatchPath(t *testing.T) {
	t.Parallel()
	srv, bt, logBuf := newLoggingBatchServer(t)
	_, err := runBatch(t, srv, bt, []protocol.Message{
		reqMessage(`"1"`, "initialize", map[string]any{}),
		notifMessage("initialized", map[string]any{}),
		reqMessage(`"b1"`, "$test/blockRequest", map[string]any{}),
		notifMessage("$test/releaseRequest", map[string]any{"id": "b1"}),
		notifMessage("$test/flushRequests", map[string]any{}),
		reqMessage(`"sh"`, "shutdown", nil),
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	out := logBuf.String()

	for _, want := range []string{
		`method=initialize`, `id="\"1\""`,
		`method=$test/blockRequest`, `id="\"b1\""`,
		`method=shutdown`, `id="\"sh\""`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q; got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "method=initialized") {
		t.Errorf("log output missing notification method=initialized; got:\n%s", out)
	}
}
