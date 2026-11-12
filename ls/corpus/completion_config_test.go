// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package corpus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/completion"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

type completionConfig struct {
	Position         protocol.Position `json:"position"`
	Source           string            `json:"source"`
	Description      string            `json:"description"`
	TriggerCharacter string            `json:"triggerCharacter"`
	Items            []json.RawMessage `json:"items"`
}

type completionConfigItem struct {
	InsertText          string          `json:"insertText"`
	Detail              string          `json:"detail"`
	Label               string          `json:"label"`
	SortText            string          `json:"sortText"`
	FilterText          string          `json:"filterText"`
	AdditionalTextEdits json.RawMessage `json:"additionalTextEdits"`
}

type completionConfigEnv struct {
	ConfigDir      string
	PackageOrg     string
	PackageName    string
	Repository     string // path under completion/testdata; empty means no repository is configured
	ProjectRoot    string // path under completion/testdata copied into the temporary workspace; empty uses PackageName or an overlay-only single file
	Packages       []completion.AvailablePackage
	SnippetSupport bool
	ExactItems     bool
}

func TestSingleRequestCompletionConfigs(t *testing.T) {
	t.Parallel()
	env := completionConfigEnv{
		ConfigDir:      "completion/testdata/config",
		PackageOrg:     "",
		PackageName:    "",
		Repository:     "",
		ProjectRoot:    "source/two-file-module",
		Packages:       nil,
		SnippetSupport: false,
		ExactItems:     true,
	}
	configs := discoverCompletionConfigs(t, env.ConfigDir)
	runCompletionConfigs(t, env, configs)
}

func TestEnumDeclarationCompletionConfigs(t *testing.T) {
	t.Parallel()
	env := completionConfigEnv{
		ConfigDir:      "completion/testdata/enum_decl_ctx/config",
		PackageOrg:     "completion",
		PackageName:    "enum_decl_ctx",
		Repository:     "repository",
		ProjectRoot:    "enum_decl_ctx/source/projectls",
		Packages:       enumMemberAvailablePackages,
		SnippetSupport: true,
	}
	configs := discoverCompletionConfigs(t, env.ConfigDir)
	runCompletionConfigs(t, env, configs)
}

func TestTypeDefinitionCompletionConfigs(t *testing.T) {
	t.Parallel()
	env := completionConfigEnv{
		ConfigDir:      "completion/testdata/type_def/config",
		PackageOrg:     "completion",
		PackageName:    "type_def",
		Repository:     "repository",
		Packages:       typeDefinitionAvailablePackages,
		SnippetSupport: true,
	}
	configs := discoverCompletionConfigs(t, env.ConfigDir)
	runCompletionConfigs(t, env, configs)
}

func discoverCompletionConfigs(t *testing.T, configDir string) []string {
	t.Helper()
	entries, err := fs.ReadDir(os.DirFS("."), configDir)
	if err != nil {
		t.Fatal(err)
	}
	var configs []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			configs = append(configs, path.Join(configDir, entry.Name()))
		}
	}
	sort.Strings(configs)
	return configs
}

func skipList() []string {
	// TODO: These configs are blocked by https://github.com/ballerina-nutcracker/ballerina/issues/1011.
	return []string{
		"completion/testdata/enum_decl_ctx/config/config1.json",
		"completion/testdata/enum_decl_ctx/config/config3.json",
		"completion/testdata/enum_decl_ctx/config/config4.json",
	}
}

func runCompletionConfigs(t *testing.T, env completionConfigEnv, configs []string) {
	t.Helper()
	client := newCompletionConfigClient(t, env, len(configs))
	skips := skipList()
	for _, configPath := range configs {
		t.Run(path.Base(configPath), func(t *testing.T) {
			t.Parallel()
			if slices.Contains(skips, path.Clean(configPath)) {
				t.Skip()
			}
			config, raw := readCompletionConfig(t, configPath)
			source, err := fs.ReadFile(os.DirFS("."), path.Join("completion/testdata", config.Source))
			if err != nil {
				t.Fatal(err)
			}
			actual := client.complete(t, env, config.Source, string(source), config.Position)
			if *update {
				writeCompletionConfigItems(t, configPath, raw, actual)
				return
			}
			if env.ExactItems {
				compareExactCompletionConfigItems(t, config.Items, actual)
			} else {
				compareCompletionConfigItems(t, config.Items, actual)
			}
		})
	}
}

func readCompletionConfig(t *testing.T, configPath string) (completionConfig, []byte) {
	t.Helper()
	content, err := fs.ReadFile(os.DirFS("."), configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config completionConfig
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	return config, content
}

// Splice the items so golden updates preserve unrelated fixture formatting.
func writeCompletionConfigItems(t *testing.T, configPath string, raw []byte, items []protocol.CompletionItem) {
	t.Helper()
	encodedItems, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := spliceTopLevelJSONField(raw, "items", encodedItems)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyCompletionProject(t *testing.T, from, to string) {
	t.Helper()
	entries, err := fs.ReadDir(os.DirFS(from), ".")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		source := filepath.Join(from, entry.Name())
		destination := filepath.Join(to, entry.Name())
		if entry.IsDir() {
			copyCompletionProject(t, source, destination)
			continue
		}
		content, err := fs.ReadFile(os.DirFS(from), entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

type completionConfigClient struct {
	mu        sync.Mutex
	transport *stepTransport
	server    *server.Server
	workspace *workspace.ProjectService
	nextID    int
}

func newCompletionConfigClient(t *testing.T, env completionConfigEnv, configs int) *completionConfigClient {
	t.Helper()
	transport := &stepTransport{}
	bus := event.New()
	projectService := workspace.New(os.DirFS, bus, workspace.WithMaxProjects(2*configs))
	compiler := compile.New(projectService, bus, compile.WithDebounce(0), compile.WithMaxStableSnapshots(2*configs))
	srv := server.New(transport, projectService, compiler, bus, server.WithAvailablePackages(env.Packages))
	client := &completionConfigClient{
		mu:        sync.Mutex{},
		transport: transport,
		server:    srv,
		workspace: projectService,
		nextID:    2,
	}
	t.Cleanup(func() {
		client.mu.Lock()
		defer client.mu.Unlock()
		defer bus.Close()
		defer compiler.Shutdown()
		client.exchange(t)
	})
	initializeParams := map[string]any{}
	if env.SnippetSupport {
		initializeParams["capabilities"] = map[string]any{"textDocument": map[string]any{"completion": map[string]any{"completionItem": map[string]any{"snippetSupport": true}}}}
	}
	responses := client.exchange(t,
		completionConfigMessage(1, "initialize", initializeParams),
		completionConfigMessage(nil, "initialized", map[string]any{}),
	)
	completionConfigResult(t, 1, responses)
	return client
}

// Callers serialize whole fixture transactions, including repository changes and all asynchronous output.
func (c *completionConfigClient) exchange(t *testing.T, messages ...protocol.Message) []json.RawMessage {
	t.Helper()
	messages = append(messages,
		completionConfigMessage(nil, "$test/flushRequests", map[string]any{}),
		completionConfigMessage(nil, "$test/flush", map[string]any{}),
	)
	var input bytes.Buffer
	for _, message := range messages {
		if err := protocol.WriteMessage(&input, message); err != nil {
			t.Fatal(err)
		}
	}
	c.transport.writer.Reset()
	c.transport.reader = bytes.NewReader(input.Bytes())
	if err := c.server.Serve(); err != nil {
		t.Fatal(err)
	}
	responses, err := readMessages(c.transport.writer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return responses
}

func completionConfigResult(t *testing.T, id int, responses []json.RawMessage) json.RawMessage {
	t.Helper()
	var result json.RawMessage
	for _, response := range responses {
		var message struct {
			ID     *int               `json:"id"`
			Result json.RawMessage    `json:"result"`
			Error  *protocol.RPCError `json:"error"`
		}
		if err := json.Unmarshal(response, &message); err != nil {
			t.Fatal(err)
		}
		if message.ID == nil {
			continue
		}
		if *message.ID != id || len(result) != 0 {
			t.Fatalf("unexpected or duplicate response ID: %d", *message.ID)
		}
		if message.Error != nil {
			t.Fatalf("request %d failed: %+v", id, message.Error)
		}
		if len(message.Result) == 0 {
			t.Fatalf("request %d response has no result", id)
		}
		result = message.Result
	}
	if len(result) == 0 {
		t.Fatalf("missing response ID: %d", id)
	}
	return result
}

func (c *completionConfigClient) complete(t *testing.T, env completionConfigEnv, sourcePath, source string, position protocol.Position) []protocol.CompletionItem {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	root := filepath.Join(t.TempDir(), "completion workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	documentPath := filepath.Join(root, "main.bal")
	if relative, ok := strings.CutPrefix(sourcePath, env.ProjectRoot+"/"); env.ProjectRoot != "" && ok {
		root = filepath.Join(root, path.Base(env.ProjectRoot))
		copyCompletionProject(t, filepath.Join("completion/testdata", env.ProjectRoot), root)
		documentPath = filepath.Join(root, filepath.FromSlash(relative))
	} else if env.PackageName != "" {
		toml := fmt.Sprintf("[package]\norg = %q\nname = %q\nversion = \"0.1.0\"\n", env.PackageOrg, env.PackageName)
		if err := os.WriteFile(filepath.Join(root, "Ballerina.toml"), []byte(toml), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(documentPath, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		documentPath = filepath.Join(root, path.Base(sourcePath))
	}
	documentURI, err := workspace.FileURIFromPath(documentPath)
	if err != nil {
		t.Fatal(err)
	}
	var repositories []projects.Repository
	if env.Repository != "" {
		repositories = append(repositories, projects.NewFileSystemRepository(os.DirFS(filepath.Join("completion/testdata", env.Repository)), "."))
	}
	workspace.WithRepositories(repositories)(c.workspace)
	id := c.nextID
	c.nextID++
	responses := c.exchange(t,
		completionConfigMessage(nil, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri": documentURI.String(), "languageId": "ballerina", "version": 1, "text": source,
			},
		}),
		completionConfigMessage(nil, "$test/flush", map[string]any{}),
		completionConfigMessage(id, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": documentURI.String()},
			"position":     position,
		}),
		completionConfigMessage(nil, "$test/flushRequests", map[string]any{}),
		completionConfigMessage(nil, "textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": documentURI.String()},
		}),
	)
	var items []protocol.CompletionItem
	if err := json.Unmarshal(completionConfigResult(t, id, responses), &items); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.workspace.Snapshot(documentURI); ok {
		t.Fatal("completion fixture document is still open")
	}
	return items
}

var enumMemberAvailablePackages = append(append([]completion.AvailablePackage(nil), typeDefinitionAvailablePackages...),
	completion.AvailablePackage{Organization: "ballerina", Name: "jballerina.java"},
	completion.AvailablePackage{Organization: "projectls", Name: "constants"},
)

var typeDefinitionAvailablePackages = []completion.AvailablePackage{
	{Organization: "ballerina", Name: "lang.array"},
	{Organization: "ballerina", Name: "lang.runtime"},
	{Organization: "ballerina", Name: "test"},
	{Organization: "ballerina", Name: "lang.value"},
	{Organization: "ballerina", Name: "module1"},
	{Organization: "ballerina", Name: "lang.regexp"},
	{Organization: "test", Name: "project1"},
	{Organization: "test", Name: "project2"},
	{Organization: "test", Name: "local_project1"},
	{Organization: "test", Name: "local_project2"},
}

func completionConfigMessage(id any, method string, params any) protocol.Message {
	encodedParams, _ := json.Marshal(params)
	var encodedID json.RawMessage
	if id != nil {
		encodedID, _ = json.Marshal(id)
	}
	return protocol.Message{JSONRPC: "2.0", ID: encodedID, Method: method, Params: encodedParams}
}

func compareExactCompletionConfigItems(t *testing.T, expected []json.RawMessage, actual []protocol.CompletionItem) {
	t.Helper()
	encoded, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(encoded, &items); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalizeMessages(expected), normalizeMessages(items)) {
		t.Fatalf("completion items mismatch\nactual: %s\nexpected: %s", formatMessages(items), formatMessages(expected))
	}
}

// Completion order is ignored, but duplicate counts must match.
func compareCompletionConfigItems(t *testing.T, expected []json.RawMessage, actual []protocol.CompletionItem) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("completion item count = %d, want %d", len(actual), len(expected))
	}
	actualSignatures := make([]string, len(actual))
	remaining := make(map[string]int, len(actual))
	for i, item := range actual {
		encoded, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		signature := completionConfigSignature(encoded)
		actualSignatures[i] = signature
		remaining[signature]++
	}
	for _, item := range expected {
		encoded, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		signature := completionConfigSignature(encoded)
		if remaining[signature] == 0 {
			t.Fatalf("completion item missing: %s\nactual: %s", signature, actualSignatures)
		}
		remaining[signature]--
	}
}

// Comparison excludes kind, documentation, and insertTextFormat.
func completionConfigSignature(encoded []byte) string {
	var item completionConfigItem
	_ = json.Unmarshal(encoded, &item)
	additional := ""
	if len(item.AdditionalTextEdits) > 0 && string(item.AdditionalTextEdits) != "null" {
		additional = string(item.AdditionalTextEdits)
	}
	return strings.ReplaceAll(fmt.Sprintf("{%s,%s,%s,%s,%s,%s}", item.InsertText, item.Detail, item.Label, item.SortText, item.FilterText, additional), "\r\n", "\n")
}
