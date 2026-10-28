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

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

const hostRepositoryPanic = "host repository version lookup panic"

type hostPanicRepository struct {
	attempted chan struct{}
	attempts  atomic.Int32
}

func (*hostPanicRepository) GetPackage(context.Context, string, string, string, projects.ResolutionOptions) (*projects.Package, error) {
	return nil, nil
}

func (r *hostPanicRepository) GetPackageVersions(_ context.Context, org, name string, _ projects.ResolutionOptions) ([]projects.PackageVersion, error) {
	if org == "acme" && name == "dep" {
		r.attempts.Add(1)
		select {
		case r.attempted <- struct{}{}:
		default:
		}
		panic(hostRepositoryPanic)
	}
	return nil, nil
}

type hostCaptureWriter struct {
	bytes.Buffer
	closed bool
}

func (w *hostCaptureWriter) Close() error {
	w.closed = true
	return nil
}

type hostTestTransport struct {
	io.Reader
	io.Writer
}

func TestServeHostRecordsRecoveredCompilePanicWithoutProtocolOutput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := "import acme/dep;\npublic function main() {}\n"
	for name, content := range map[string]string{
		"Ballerina.toml": "[package]\norg = \"test\"\nname = \"main\"\nversion = \"0.1.0\"\n",
		"main.bal":       source,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	uri, err := workspace.FileURIFromPath(filepath.Join(root, "main.bal"))
	if err != nil {
		t.Fatal(err)
	}
	repository := &hostPanicRepository{attempted: make(chan struct{}, 1)}
	input, client := io.Pipe()
	defer func() { _ = client.Close() }()
	var logs, output hostCaptureWriter
	done := make(chan error, 1)
	go func() {
		defer func() { _ = input.Close() }()
		done <- serveHost(hostTestTransport{Reader: input, Writer: &output}, os.DirFS, &logs,
			workspace.WithRepositories([]projects.Repository{repository}))
	}()
	send := func(message any) {
		t.Helper()
		if err := protocol.WriteMessage(client, message); err != nil {
			t.Fatal(err)
		}
	}
	send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	send(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
		"textDocument": map[string]any{"uri": uri.String(), "languageId": "ballerina", "version": 1, "text": source},
	}})
	select {
	case <-repository.attempted:
	case err := <-done:
		t.Fatalf("host stopped before compilation: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("real compilation did not reach the injected repository")
	}
	send(map[string]any{"jsonrpc": "2.0", "method": "$test/flush"})
	send(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"})
	send(map[string]any{"jsonrpc": "2.0", "method": "exit"})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveHost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("host did not stop after exit")
	}

	if attempts := repository.attempts.Load(); attempts != 1 {
		t.Errorf("repository panic attempts = %d, want 1", attempts)
	}
	var records []string
	for line := range strings.SplitSeq(logs.String(), "\n") {
		if strings.Contains(line, "compile: recovered panic") {
			records = append(records, line)
		}
	}
	if len(records) != 1 {
		t.Fatalf("recovered compile records = %d, want 1; host output:\n%s", len(records), logs.String())
	}
	for _, field := range []string{
		"level=ERROR", `recovered="` + hostRepositoryPanic + `"`, "root=" + root, "generation=1",
		`stack="goroutine `, "runtime/debug.Stack", "hostPanicRepository).GetPackageVersions",
	} {
		if !strings.Contains(records[0], field) {
			t.Errorf("recovery log missing %q: %s", field, records[0])
		}
	}
	if strings.Contains(output.String(), hostRepositoryPanic) || strings.Contains(output.String(), "recovered panic") {
		t.Errorf("panic leaked into protocol output: %s", output.String())
	}
	reader := bufio.NewReader(bytes.NewReader(output.Bytes()))
	for _, id := range []string{"1", "2"} {
		payload, err := protocol.ReadFrame(reader)
		if err != nil {
			t.Fatalf("response %s framing: %v", id, err)
		}
		var response struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Error   json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(payload, &response); err != nil {
			t.Fatal(err)
		}
		if response.JSONRPC != "2.0" || string(response.ID) != id || response.Method != "" || len(response.Error) != 0 {
			t.Errorf("unexpected response: %s", payload)
		}
	}
	if payload, err := protocol.ReadFrame(reader); err != io.EOF {
		t.Errorf("extra protocol output or notification: %s (%v)", payload, err)
	}
	if logs.closed || output.closed {
		t.Errorf("host closed caller-owned writers: log=%v protocol=%v", logs.closed, output.closed)
	}
}
