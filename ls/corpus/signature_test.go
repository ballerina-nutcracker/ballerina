// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package corpus

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

const signatureTestdataDirectory = "signature/testdata"
const signatureConfigDirectory = signatureTestdataDirectory + "/config"

type signatureConfig struct {
	Source   string            `json:"source"`
	Position protocol.Position `json:"position"`
	Expected []json.RawMessage `json:"expected"`
}

type signatureConfigRequest struct {
	ID       int
	Position protocol.Position
	Target   bool
}

type signatureBatch struct {
	root  string
	input []byte
	done  chan error
}

type signatureClient struct {
	input        chan signatureBatch
	done         chan struct{}
	compiler     *compile.CompilationService
	access       sync.RWMutex
	mu           sync.Mutex
	nextID       int
	pending      map[int]chan json.RawMessage
	repositories map[string]*signatureGateRepository
}

func newSignatureClient(t *testing.T, configs int) *signatureClient {
	t.Helper()
	client := &signatureClient{
		input: make(chan signatureBatch), done: make(chan struct{}), nextID: 1,
		pending: make(map[int]chan json.RawMessage), repositories: make(map[string]*signatureGateRepository),
	}
	output, writer := io.Pipe()
	transport := &struct {
		io.Reader
		io.Writer
	}{Reader: bytes.NewReader(nil), Writer: writer}
	bus := event.New()
	service := workspace.New(os.DirFS, bus, workspace.WithMaxProjects(2*configs))
	client.compiler = compile.New(service, bus, compile.WithDebounce(0))
	srv := server.New(transport, service, client.compiler, bus)
	served := make(chan struct{})
	go func() {
		defer close(served)
		for batch := range client.input {
			client.mu.Lock()
			gate, hasRepository := client.repositories[batch.root]
			client.mu.Unlock()
			var repositories []projects.Repository
			if gate != nil {
				repositories = []projects.Repository{gate}
			} else if hasRepository {
				repositories = []projects.Repository{projects.NewFileSystemRepository(os.DirFS(filepath.Join(batch.root, "repository")), ".")}
			}
			workspace.WithRepositories(repositories)(service)
			transport.Reader = bytes.NewReader(batch.input)
			batch.done <- srv.Serve()
		}
	}()
	go client.read(t, output)
	t.Cleanup(func() {
		var input bytes.Buffer
		if err := protocol.WriteMessage(&input, signatureConfigMessage(nil, "$test/flushRequests", map[string]any{})); err != nil {
			t.Fatal(err)
		}
		batch := signatureBatch{root: "", input: input.Bytes(), done: make(chan error, 1)}
		client.input <- batch
		if err := <-batch.done; err != nil {
			t.Error(err)
		}
		close(client.input)
		<-served
		srv.Flush()
		client.compiler.Shutdown()
		bus.Close()
		_ = writer.Close()
		<-client.done
		client.mu.Lock()
		defer client.mu.Unlock()
		for id := range client.pending {
			t.Errorf("missing response ID: %d", id)
		}
	})
	id, initialize, reply := client.request("initialize", map[string]any{"capabilities": json.RawMessage(signatureClientCapabilities)})
	var input bytes.Buffer
	if err := protocol.WriteMessage(&input, initialize); err != nil {
		t.Fatal(err)
	}
	if err := client.write("", input.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err := collectSignatureResponses([]signatureConfigRequest{{ID: id}}, []json.RawMessage{client.await(t, id, reply)}); err != nil {
		t.Fatal(err)
	}
	return client
}

func (c *signatureClient) request(method string, params any) (int, json.RawMessage, <-chan json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextID
	c.nextID++
	reply := make(chan json.RawMessage, 1)
	c.pending[id] = reply
	return id, signatureConfigMessage(id, method, params), reply
}

func (c *signatureClient) write(root string, input []byte) error {
	batch := signatureBatch{root: root, input: input, done: make(chan error, 1)}
	select {
	case c.input <- batch:
		return <-batch.done
	case <-c.done:
		return fmt.Errorf("signature connection closed")
	}
}

func (c *signatureClient) await(t *testing.T, id int, reply <-chan json.RawMessage) json.RawMessage {
	t.Helper()
	select {
	case raw := <-reply:
		return raw
	case <-c.done:
		t.Fatalf("signature connection closed before response ID: %d", id)
		return nil
	case <-time.After(time.Minute):
		t.Fatalf("missing response ID: %d", id)
		return nil
	}
}

func (c *signatureClient) read(t *testing.T, output *io.PipeReader) {
	defer close(c.done)
	defer func() {
		if err := output.Close(); err != nil {
			t.Error(err)
		}
	}()
	reader := bufio.NewReader(output)
	for {
		raw, err := protocol.ReadFrame(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Error(err)
			return
		}
		var response struct {
			ID      *int   `json:"id"`
			Method  string `json:"method"`
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(raw, &response); err != nil {
			t.Error(err)
			return
		}
		if response.ID == nil && response.Method != "" && response.JSONRPC == "2.0" {
			continue
		}
		if response.ID == nil {
			t.Errorf("invalid response: %s", raw)
			return
		}
		c.mu.Lock()
		reply, ok := c.pending[*response.ID]
		delete(c.pending, *response.ID)
		c.mu.Unlock()
		if !ok {
			t.Errorf("unexpected or duplicate response ID: %d", *response.ID)
			return
		}
		reply <- raw
	}
}

func TestSignatureConfigs(t *testing.T) {
	t.Parallel()
	configs := discoverSignatureConfigs(t)
	client := newSignatureClient(t, len(configs))
	for _, name := range configs {
		t.Run(strings.TrimPrefix(name, signatureConfigDirectory+"/"), func(t *testing.T) {
			t.Parallel()
			config, content := readSignatureConfig(t, name)
			actual := executeSignatureConfig(t, client, config, name)
			if len(actual) != len(config.Expected) {
				t.Fatalf("expected %d target outcomes, got %d", len(config.Expected), len(actual))
			}
			if *update {
				expected, err := json.MarshalIndent(actual, "  ", "  ")
				if err != nil {
					t.Fatal(err)
				}
				updated, err := spliceTopLevelJSONField(content, "expected", expected)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, updated, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			if !signatureResponsesEqual(config.Expected, actual) {
				t.Fatalf("signature payload mismatch\nactual: %s\nexpected: %s", formatMessages(actual), formatMessages(config.Expected))
			}
		})
	}
}

func signatureConfigFamily(fixturePath string) string {
	name := strings.TrimSuffix(path.Base(fixturePath), ".signatureHelp.json")
	if index := strings.LastIndex(name, "-query"); index >= 0 {
		if number, err := strconv.Atoi(name[index+6:]); err == nil && number > 1 {
			name = name[:index]
		}
	}
	return name
}

func discoverSignatureConfigs(t *testing.T) []string {
	t.Helper()
	configs, err := signatureConfigPaths(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) == 0 {
		t.Fatal("no signature configurations discovered")
	}
	return configs
}

func signatureConfigPaths(files fs.FS) ([]string, error) {
	var configs []string
	err := fs.WalkDir(files, signatureConfigDirectory, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && path.Ext(name) == ".json" {
			configs = append(configs, name)
		}
		return nil
	})
	return configs, err
}

func TestSignatureRecursiveDiscovery(t *testing.T) {
	files := fstest.MapFS{
		"signature/testdata/config/function.signatureHelp.json":               {Data: []byte(`{}`)},
		"signature/testdata/config/nested/function-query2.signatureHelp.json": {Data: []byte(`{}`)},
		"signature/testdata/config/nested/deeper/recovery.signatureHelp.json": {Data: []byte(`{}`)},
		"signature/testdata/config/notes.txt":                                 {Data: []byte("not a config")},
		"signature/testdata/outside.json":                                     {Data: []byte(`{}`)},
		"unrelated/archive/ignored.json":                                      {Data: []byte(`{}`)},
	}
	actual, err := signatureConfigPaths(files)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"signature/testdata/config/function.signatureHelp.json",
		"signature/testdata/config/nested/deeper/recovery.signatureHelp.json",
		"signature/testdata/config/nested/function-query2.signatureHelp.json",
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("recursive discovery = %v, want %v", actual, want)
	}
	if got := signatureConfigFamily(want[2]); got != "function" {
		t.Fatalf("nested split family = %s, want function", got)
	}
}

func readSignatureConfig(t *testing.T, name string) (signatureConfig, []byte) {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	config, err := decodeSignatureConfig(content)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return config, content
}

func decodeSignatureConfig(content []byte) (signatureConfig, error) {
	var config signatureConfig
	var wire struct {
		Source   string            `json:"source"`
		Expected []json.RawMessage `json:"expected"`
		Position *struct {
			Line      *uint32 `json:"line"`
			Character *uint32 `json:"character"`
		} `json:"position"`
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return config, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return config, fmt.Errorf("configuration must contain one JSON object")
	}
	if wire.Position == nil || wire.Position.Line == nil || wire.Position.Character == nil {
		return config, fmt.Errorf("configuration requires an explicit root position")
	}
	if len(wire.Expected) == 0 {
		return config, fmt.Errorf("configuration requires a nonempty expected array, including explicit null outcomes")
	}
	if !fs.ValidPath(wire.Source) || wire.Source == "." || strings.ContainsAny(wire.Source, "\\$:") {
		return config, fmt.Errorf("source must be a nonempty testdata-relative path")
	}
	for _, payload := range wire.Expected {
		if err := validateSignaturePayload(payload); err != nil {
			return config, err
		}
	}
	return signatureConfig{Source: wire.Source, Position: protocol.Position{Line: *wire.Position.Line, Character: *wire.Position.Character}, Expected: wire.Expected}, nil
}

func validateSignaturePayload(raw json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return fmt.Errorf("expected a signature payload, explicit null, or error object: %s", raw)
	}
	if responseError, ok := fields["error"]; ok {
		if len(fields) != 1 {
			return fmt.Errorf("error payload must contain only error: %s", raw)
		}
		var rpcError struct {
			Code    *int    `json:"code"`
			Message *string `json:"message"`
		}
		if err := json.Unmarshal(responseError, &rpcError); err != nil || rpcError.Code == nil || rpcError.Message == nil {
			return fmt.Errorf("invalid signature error: %s", raw)
		}
		return nil
	}
	for key := range fields {
		if key != "signatures" && key != "activeSignature" && key != "activeParameter" {
			return fmt.Errorf("unexpected signature payload field %q", key)
		}
	}
	var help protocol.SignatureHelp
	if err := json.Unmarshal(raw, &help); err != nil || help.Signatures == nil {
		return fmt.Errorf("expected a bare signature-help payload: %s", raw)
	}
	return nil
}

func TestSignatureConfigValidation(t *testing.T) {
	valid := `{"source":"function.bal","position":{"line":18,"character":11},"expected":[null]}`
	if _, err := decodeSignatureConfig([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"legacy requests", `"source":`, `"requests":[],"source":`},
		{"messages", `"source":`, `"messages":[],"source":`},
		{"document override", `"source":`, `"document":"main.bal","source":`},
		{"capabilities", `"source":`, `"capabilities":{},"source":`},
		{"inline sources", `"source":`, `"sources":{"main.bal":"public function main() {}"},"source":`},
		{"document metadata", `"source":`, `"document":{"uri":"${root}/signature.bal","languageId":"ballerina","version":1,"text":"duplicated"},"source":`},
		{"exposed ID", `"source":`, `"id":2,"source":`},
		{"setup flag", `"source":`, `"holdSignatureDependency":true,"source":`},
		{"unordered flag", `"source":`, `"unordered":true,"source":`},
		{"missing position", `,"position":{"line":18,"character":11}`, ``},
		{"missing character", `,"character":11`, ``},
		{"negative position", `"line":18`, `"line":-1`},
		{"null position", `{"line":18,"character":11}`, `null`},
		{"null expected", `"expected":[null]`, `"expected":null`},
		{"omitted null", `"expected":[null]`, `"expected":[]`},
		{"envelope expected", `"expected":[null]`, `"expected":[{"jsonrpc":"2.0","id":2,"result":null}]`},
		{"result wrapper", `"expected":[null]`, `"expected":[{"result":null}]`},
		{"empty expected object", `"expected":[null]`, `"expected":[{}]`},
		{"malformed error", `"expected":[null]`, `"expected":[{"error":{}}]`},
		{"null source", `"source":"function.bal"`, `"source":null`},
		{"empty source", `"source":"function.bal"`, `"source":""`},
		{"absent source text", `"source":"function.bal",`, ``},
		{"invalid capabilities", `"source":`, `"capabilities":false,"source":`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeSignatureConfig([]byte(strings.Replace(valid, tc.old, tc.replacement, 1))); err == nil {
				t.Fatal("accepted malformed simplified fixture")
			}
		})
	}
}

func TestSignatureSourcePaths(t *testing.T) {
	for _, source := range []string{"function.bal", "nested/function.bal"} {
		content := signatureConfigJSON(map[string]any{"source": source, "position": map[string]int{"line": 18, "character": 11}, "expected": []any{nil}})
		if _, err := decodeSignatureConfig(content); err != nil {
			t.Fatalf("valid testdata-relative source %q rejected: %v", source, err)
		}
	}
	for _, source := range []string{"../function.bal", "nested/../function.bal", "/tmp/function.bal", "C:/function.bal", "..", ".", "./function.bal", "nested//function.bal"} {
		t.Run(source, func(t *testing.T) {
			content := signatureConfigJSON(map[string]any{"source": source, "position": map[string]int{"line": 18, "character": 11}, "expected": []any{nil}})
			if _, err := decodeSignatureConfig(content); err == nil {
				t.Fatalf("unsafe source path %q accepted", source)
			}
		})
	}
}

func TestSignatureMalformedTranscriptStillRejected(t *testing.T) {
	files := fstest.MapFS{"signature/testdata/bad.json": {Data: []byte(`{"source":"function.bal","expected":[]}`)}}
	if _, err := discoverTranscripts(files); err == nil {
		t.Fatal("generic discovery accepted malformed transcript")
	}
}

func collectSignatureResponses(requests []signatureConfigRequest, messages []json.RawMessage) ([]json.RawMessage, error) {
	indices := make(map[int]int, len(requests))
	for index, request := range requests {
		indices[request.ID] = index
	}
	replies := make([]json.RawMessage, len(requests))
	for _, raw := range messages {
		var response struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      *int            `json:"id"`
			Method  string          `json:"method"`
			Result  json.RawMessage `json:"result"`
			Error   json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(raw, &response); err != nil {
			return nil, err
		}
		if response.ID == nil && response.Method != "" && response.JSONRPC == "2.0" {
			continue
		}
		if response.ID == nil || response.Method != "" || response.JSONRPC != "2.0" {
			return nil, fmt.Errorf("invalid response: %s", raw)
		}
		id := *response.ID
		index, ok := indices[id]
		if !ok || replies[index] != nil {
			return nil, fmt.Errorf("unexpected or duplicate response ID: %d", id)
		}
		if (len(response.Result) > 0) == (len(response.Error) > 0) {
			return nil, fmt.Errorf("response requires a result or error: %s", raw)
		}
		replies[index] = response.Result
		if len(response.Error) > 0 {
			replies[index] = signatureConfigJSON(map[string]json.RawMessage{"error": response.Error})
		}
		if id == 1 {
			var initialize struct {
				Capabilities struct {
					SignatureHelpProvider json.RawMessage `json:"signatureHelpProvider"`
				} `json:"capabilities"`
			}
			if len(response.Error) > 0 {
				return nil, fmt.Errorf("initialize failed: %s", response.Error)
			}
			if err := json.Unmarshal(response.Result, &initialize); err != nil {
				return nil, err
			}
			wanted := []json.RawMessage{json.RawMessage(`{"triggerCharacters":["(",","]}`)}
			if !signatureResponsesEqual(wanted, []json.RawMessage{initialize.Capabilities.SignatureHelpProvider}) {
				return nil, fmt.Errorf("signature capability advertisement changed: %s", response.Result)
			}
			continue
		}
		if err := validateSignaturePayload(replies[index]); err != nil {
			return nil, err
		}
	}
	selected := make([]json.RawMessage, 0)
	for index, reply := range replies {
		if reply == nil {
			return nil, fmt.Errorf("missing response ID: %d", requests[index].ID)
		}
		if requests[index].Target {
			selected = append(selected, reply)
		}
	}
	return selected, nil
}

func signatureResponsesEqual(expected, actual []json.RawMessage) bool {
	return reflect.DeepEqual(normalizeMessages(expected), normalizeMessages(actual))
}

func signatureSkipList() []string {
	return nil
}

func signatureConfigQueries(fixturePath string, config signatureConfig) []signatureConfigRequest {
	requests := []signatureConfigRequest{{Position: config.Position, Target: true}}
	switch signatureConfigFamily(fixturePath) {
	case "cancellation", "docs-cancellation":
		requests = append(requests, requests[0])
	case "docs-source-edit":
		requests = []signatureConfigRequest{{Position: protocol.Position{Line: 22, Character: 11}}, {Position: protocol.Position{Line: 22, Character: 14}}}
		target := 0
		if strings.HasSuffix(path.Base(fixturePath), "-query2.signatureHelp.json") {
			target = 1
		}
		requests[target] = signatureConfigRequest{Position: config.Position, Target: true}
	}
	return requests
}

const signatureClientCapabilities = `{"textDocument":{"signatureHelp":{"signatureInformation":{"documentationFormat":["markdown"],"parameterInformation":{"labelOffsetSupport":true}}}}}`

func signatureConfigJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("marshal signature configuration message: " + err.Error())
	}
	return encoded
}

func signatureConfigMessage(id any, method string, params any) json.RawMessage {
	message := map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
	if id != nil {
		message["id"] = id
	}
	return signatureConfigJSON(message)
}

const signatureRetargetOpen = `{"textDocument":{"uri":"${root}/modules/lib/lib.bal","languageId":"ballerina","version":1,"text":"// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).\n//\n// WSO2 LLC. licenses this file to you under the Apache License,\n// Version 2.0 (the \"License\"); you may not use this file except\n// in compliance with the License.\n// You may obtain a copy of the License at\n//\n// http://www.apache.org/licenses/LICENSE-2.0\n//\n// Unless required by applicable law or agreed to in writing,\n// software distributed under the License is distributed on an\n// \"AS IS\" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY\n// KIND, either express or implied. See the License for the\n// specific language governing permissions and limitations\n// under the License.\n\npublic function foo(int newCount, string newTitle) { _ = newCount; _ = newTitle; }\n"}}`

const signatureTargetChange = `{"textDocument":{"uri":"${root}/main.bal","version":2},"contentChanges":[{"text":"// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).\n//\n// WSO2 LLC. licenses this file to you under the Apache License,\n// Version 2.0 (the \"License\"); you may not use this file except\n// in compliance with the License.\n// You may obtain a copy of the License at\n//\n// http://www.apache.org/licenses/LICENSE-2.0\n//\n// Unless required by applicable law or agreed to in writing,\n// software distributed under the License is distributed on an\n// \"AS IS\" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY\n// KIND, either express or implied. See the License for the\n// specific language governing permissions and limitations\n// under the License.\n\nimport acme/gate;\nfunction foo(int count, string title) {}\npublic function main() {\n    foo(2, \"edited\");\n}\n","range":{"start":{"line":0,"character":0},"end":{"line":21,"character":0}}}]}`

const signatureDocumentationChange = `{"textDocument":{"uri":"${root}/main.bal","version":2},"contentChanges":[{"text":"// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).\n//\n// WSO2 LLC. licenses this file to you under the Apache License,\n// Version 2.0 (the \"License\"); you may not use this file except\n// in compliance with the License.\n// You may obtain a copy of the License at\n//\n// http://www.apache.org/licenses/LICENSE-2.0\n//\n// Unless required by applicable law or agreed to in writing,\n// software distributed under the License is distributed on an\n// \"AS IS\" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY\n// KIND, either express or implied. See the License for the\n// specific language governing permissions and limitations\n// under the License.\n\nimport acme/gate;\n# Edited description.\n# + count - Edited count.\n# + title - Edited title.\nfunction foo(int count, string title) {}\npublic function main() {\n    foo(2, \"edited\");\n}\n","range":{"start":{"line":0,"character":0},"end":{"line":24,"character":0}}}]}`

func executeSignatureConfig(t *testing.T, client *signatureClient, config signatureConfig, fixturePath string) []json.RawMessage {
	t.Helper()
	document, source, files := signatureFixtureSource(t, config)
	rootPath := materializeSources(t, files)
	rootURI, err := workspace.FileURIFromPath(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	root := rootURI.String()
	flush := signatureConfigMessage(nil, "$test/flush", map[string]any{})
	flushRequests := signatureConfigMessage(nil, "$test/flushRequests", map[string]any{})
	requests := signatureConfigQueries(fixturePath, config)
	replies := make([]<-chan json.RawMessage, len(requests))
	query := func(index int) json.RawMessage {
		id, message, reply := client.request("textDocument/signatureHelp", map[string]any{
			"textDocument": map[string]any{"uri": "${root}/" + document},
			"position":     requests[index].Position,
		})
		requests[index].ID, replies[index] = id, reply
		return message
	}
	first := query(0)
	var during []json.RawMessage
	switch signatureConfigFamily(fixturePath) {
	case "cancellation", "docs-cancellation":
		cancel := signatureConfigMessage(nil, "$/cancelRequest", map[string]any{"id": requests[0].ID})
		during = []json.RawMessage{cancel, cancel, flushRequests}
	case "dependency-retarget", "ast-dependency-retarget":
		during = []json.RawMessage{signatureConfigMessage(nil, "textDocument/didOpen", json.RawMessage(signatureRetargetOpen))}
	case "target-edit":
		during = []json.RawMessage{signatureConfigMessage(nil, "textDocument/didChange", json.RawMessage(signatureTargetChange))}
	case "docs-source-edit":
		during = []json.RawMessage{signatureConfigMessage(nil, "textDocument/didChange", json.RawMessage(signatureDocumentationChange))}
	}
	if len(during) > 0 {
		client.access.Lock()
		defer client.access.Unlock()
	} else {
		client.access.RLock()
		defer client.access.RUnlock()
	}
	var gate *signatureGateRepository
	info, statErr := os.Stat(filepath.Join(rootPath, "repository"))
	if len(during) > 0 || statErr == nil && info.IsDir() {
		if len(during) > 0 {
			gate = newSignatureGateRepository(projects.NewFileSystemRepository(os.DirFS(filepath.Join(rootPath, "repository")), "."))
			defer gate.release()
		}
		client.mu.Lock()
		client.repositories[rootPath] = gate
		client.mu.Unlock()
	}
	send := func(raw ...json.RawMessage) {
		messages, err := fixtureMessages(raw, source, root)
		if err != nil {
			t.Fatal(err)
		}
		var input bytes.Buffer
		for _, message := range messages {
			if err := protocol.WriteMessage(&input, message); err != nil {
				t.Fatal(err)
			}
		}
		if err := client.write(rootPath, input.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	messages := []json.RawMessage{
		signatureConfigMessage(nil, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": "${root}/" + document, "languageId": "ballerina", "version": 1, "text": source,
		}}),
	}
	next := 0
	if gate != nil {
		send(messages...)
		select {
		case <-gate.entered:
		case <-client.done:
			t.Fatal("signature connection closed before dependency barrier")
		case <-time.After(time.Minute):
			t.Fatal("signature dependency barrier was not reached")
		}
		if _, _, sealed := client.compiler.DiagnosticsFor(rootPath); sealed {
			t.Fatal("signature dependency gate must precede a sealed generation")
		}
		send(append([]json.RawMessage{first}, during...)...)
		gate.release()
		messages = []json.RawMessage{flush, flushRequests}
		next = 1
	}
	for index := next; index < len(requests); index++ {
		message := first
		if index > 0 {
			message = query(index)
		}
		messages = append(messages, message)
	}
	send(messages...)
	actual := make([]json.RawMessage, len(requests))
	for index, reply := range replies {
		actual[index] = client.await(t, requests[index].ID, reply)
	}
	payloads, err := collectSignatureResponses(requests, normalizeRoot(actual, root))
	if err != nil {
		t.Fatal(err)
	}
	return payloads
}

const awaitSignatureDependency = "$test/awaitSignatureDependency"
const releaseSignatureDependency = "$test/releaseSignatureDependency"

// Block dependency resolution so fixtures can issue requests before a generation seals.
type signatureGateRepository struct {
	projects.Repository
	entered     chan struct{}
	released    chan struct{}
	enterOnce   sync.Once
	releaseOnce sync.Once
}

func newSignatureGateRepository(repository projects.Repository) *signatureGateRepository {
	return &signatureGateRepository{Repository: repository, entered: make(chan struct{}), released: make(chan struct{})}
}

func (r *signatureGateRepository) await(ctx context.Context) error {
	r.enterOnce.Do(func() { close(r.entered) })
	select {
	case <-r.released:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *signatureGateRepository) release() { r.releaseOnce.Do(func() { close(r.released) }) }

func (r *signatureGateRepository) GetPackageVersions(ctx context.Context, org, name string, options projects.ResolutionOptions) ([]projects.PackageVersion, error) {
	if err := r.await(ctx); err != nil {
		return nil, err
	}
	return r.Repository.GetPackageVersions(ctx, org, name, options)
}

func (r *signatureGateRepository) GetPackage(ctx context.Context, org, name, version string, options projects.ResolutionOptions) (*projects.Package, error) {
	if err := r.await(ctx); err != nil {
		return nil, err
	}
	return r.Repository.GetPackage(ctx, org, name, version, options)
}

func signatureFixtureSource(t *testing.T, config signatureConfig) (document string, text string, files map[string]string) {
	t.Helper()
	fixtureFS := os.DirFS(".")
	source := path.Join(signatureTestdataDirectory, config.Source)
	content, err := fs.ReadFile(fixtureFS, source)
	if err != nil {
		t.Fatal(err)
	}
	files = make(map[string]string)
	for root := path.Dir(source); root != signatureTestdataDirectory && root != "."; root = path.Dir(root) {
		if _, err := fs.Stat(fixtureFS, path.Join(root, "Ballerina.toml")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatal(err)
		}
		if err := fs.WalkDir(fixtureFS, root, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			data, err := fs.ReadFile(fixtureFS, name)
			if err != nil {
				return err
			}
			files[strings.TrimPrefix(name, root+"/")] = string(data)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return strings.TrimPrefix(source, root+"/"), string(content), files
	}
	return path.Base(config.Source), string(content), files
}

func TestSignatureProjectSources(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name     string
		fixture  string
		document string
		project  bool
		required []string
	}{
		{name: "standalone function", fixture: "function", document: "function.bal"},
		{name: "project siblings", fixture: "project", document: "main.bal", project: true, required: []string{"sibling.bal"}},
		{name: "project module", fixture: "project-query2", document: "main.bal", project: true, required: []string{"modules/lib/lib.bal"}},
		{name: "project method", fixture: "project-query5", document: "main.bal", project: true, required: []string{"modules/lib/lib.bal"}},
		{name: "imported documentation", fixture: "docs-imported", document: "main.bal", project: true, required: []string{"repository/test/docs68/0.1.0/go1.27/bala.json", "repository/test/docs68/0.1.0/go1.27/modules/docs68/docs.bal"}},
		{name: "cross file documentation", fixture: "docs-cross-file", document: "main.bal", project: true, required: []string{"other.bal"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			fixturePath := path.Join(signatureConfigDirectory, scenario.fixture+".signatureHelp.json")
			config, _ := readSignatureConfig(t, fixturePath)
			document, text, files := signatureFixtureSource(t, config)
			if document != scenario.document {
				t.Fatalf("document = %q, want %q", document, scenario.document)
			}
			source := path.Join(signatureTestdataDirectory, config.Source)
			original, err := fs.ReadFile(os.DirFS("."), source)
			if err != nil {
				t.Fatal(err)
			}
			if text != string(original) {
				t.Fatal("target text differs from the source fixture")
			}
			root := materializeSources(t, files)
			if !scenario.project {
				if files == nil || len(files) != 0 {
					t.Fatalf("standalone source acquired project files: %v", files)
				}
			} else {
				projectFS := os.DirFS(filepath.FromSlash(path.Dir(source)))
				count := 0
				if err := fs.WalkDir(projectFS, ".", func(name string, entry fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if entry.IsDir() {
						return nil
					}
					original, err := fs.ReadFile(projectFS, name)
					if err != nil {
						return err
					}
					copied, err := fs.ReadFile(os.DirFS(root), name)
					if err != nil {
						return err
					}
					if value, ok := files[name]; !ok || value != string(original) || string(copied) != string(original) {
						t.Fatalf("copied project file %s differs from fixture", name)
					}
					count++
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				if len(files) != count || files[document] != text {
					t.Fatal("project tree or target identity changed")
				}
				for _, name := range append([]string{"Ballerina.toml"}, scenario.required...) {
					if _, ok := files[name]; !ok {
						t.Fatalf("project is missing %s", name)
					}
				}
			}
		})
	}
}

func TestSignatureDocumentationExternalIdentity(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name  string
		count int
	}{{"docs-imported", 5}, {"docs-stdlib", 1}} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			fixturePath := "signature/testdata/config/" + scenario.name + ".signatureHelp.json"
			assertExternalDocumentationIdentity(t, fixturePath, scenario.count)
		})
	}
}

func assertExternalDocumentationIdentity(t *testing.T, fixturePath string, expectedCount int) {
	t.Helper()
	fixture, _ := readSignatureConfig(t, fixturePath)
	document, text, files := signatureFixtureSource(t, fixture)
	root := materializeSources(t, files)
	var options []workspace.Option
	if info, err := fs.Stat(os.DirFS(root), "repository"); err == nil && info.IsDir() {
		options = append(options, workspace.WithRepositories([]projects.Repository{projects.NewFileSystemRepository(os.DirFS(filepath.Join(root, "repository")), ".")}))
	}
	uri, err := workspace.FileURIFromPath(filepath.Join(root, filepath.FromSlash(document)))
	if err != nil {
		t.Fatal(err)
	}
	bus := event.New()
	defer bus.Close()
	service := workspace.New(os.DirFS, bus, options...)
	compiler := compile.New(service, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	if _, err := service.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeOpen, URI: uri, Text: text, Version: 1, LanguageID: "ballerina"}); err != nil {
		t.Fatal(err)
	}
	sm, ok := compiler.SealedModuleFor(context.Background(), uri)
	if !ok {
		t.Fatal("no sealed module for documentation identity fixture")
	}
	visitor := &documentationCalls{sm: sm, refs: make([]model.SymbolRef, 0)}
	ast.Walk(visitor, sm.PackageNode())
	if len(visitor.refs) != expectedCount {
		t.Fatalf("got %d callees, want %d", len(visitor.refs), expectedCount)
	}
	for _, ref := range visitor.refs {
		if ref.IsEmpty() {
			t.Fatal("empty callee ref")
		}
		id := sm.Context().SymbolPackage(ref)
		projection, ok := sm.ExternalModuleProjection(id.Organization, id.Package)
		if !ok || projection.PackageNode == nil {
			t.Fatalf("missing external projection: %v", id)
		}
		declarations := &documentationDeclarations{functions: make(map[model.SymbolRef]*ast.BLangFunction)}
		ast.Walk(declarations, projection.PackageNode)
		function := declarations.functions[ref]
		if function == nil || function.Symbol() != ref || function.GetMarkdownDocumentationAttachment() == nil {
			t.Fatalf("callee %v has no exactly-equal documented declaration", ref)
		}
		callLocation, declarationLocation := sm.Context().SymbolLocation(ref), function.GetPosition()
		if callLocation.FileIndex() != declarationLocation.FileIndex() || callLocation.StartOffset() < declarationLocation.StartOffset() || callLocation.EndOffset() > declarationLocation.EndOffset() {
			t.Fatalf("callee %v declaration location mismatch", ref)
		}
		t.Logf("%s/%s %s: callsite=%v declaration=%v equal=true location=%v", id.Organization, id.Package, sm.Context().SymbolName(ref), ref, function.Symbol(), callLocation)
	}
}

type documentationCalls struct {
	sm   compile.SealedModule
	refs []model.SymbolRef
}

func (v *documentationCalls) Visit(node ast.BLangNode) ast.Visitor {
	var raw model.Symbol
	switch node := node.(type) {
	case *ast.BLangInvocation:
		raw = node.RawSymbol
	case *ast.BLangRemoteMethodCallAction:
		raw = node.RawSymbol
	case *ast.BLangNewExpression:
		cx := v.sm.Context()
		ty := node.Type()
		if !node.ClassSymbol.IsEmpty() {
			ty = cx.SymbolType(node.ClassSymbol)
		}
		if !semtypes.IsZero(ty) {
			atom, unique := semtypes.ObjectPositiveAtom(semtypes.ContextFrom(cx.GetTypeEnv()), ty)
			if unique {
				if table, ok := cx.ObjectMethodTable(atom); ok {
					if ref, ok := table.Methods["init"]; ok {
						v.refs = append(v.refs, ref)
					}
				}
			}
		}
	}
	if ref, ok := raw.(*model.SymbolRef); ok && ref != nil {
		v.refs = append(v.refs, *ref)
	}
	return v
}
func (v *documentationCalls) VisitTypeData(*ast.TypeData) ast.Visitor { return nil }

type documentationDeclarations struct {
	functions map[model.SymbolRef]*ast.BLangFunction
}

func (v *documentationDeclarations) Visit(node ast.BLangNode) ast.Visitor {
	if function, ok := node.(*ast.BLangFunction); ok {
		v.functions[function.Symbol()] = function
		return nil
	}
	return v
}
func (v *documentationDeclarations) VisitTypeData(*ast.TypeData) ast.Visitor { return nil }
