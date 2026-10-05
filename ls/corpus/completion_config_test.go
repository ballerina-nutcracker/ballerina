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
	"path"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/completion"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

type completionConfig struct {
	Position         protocol.Position      `json:"position"`
	Source           string                 `json:"source"`
	Description      string                 `json:"description"`
	TriggerCharacter string                 `json:"triggerCharacter"`
	Items            []completionConfigItem `json:"items"`
}

type completionConfigItem struct {
	InsertText          string          `json:"insertText"`
	Detail              string          `json:"detail"`
	Label               string          `json:"label"`
	SortText            string          `json:"sortText"`
	FilterText          string          `json:"filterText"`
	AdditionalTextEdits json.RawMessage `json:"additionalTextEdits"`
}

// completionConfigEnv is the deterministic fixture environment a raw
// completion-config directory runs under: the config directory itself, the
// package identity its temp project is opened as, the test-only repository
// backing its external imports (if any), and the available-package catalog
// handed to the server. A later raw completion directory supplies its own
// values instead of the generic runner assuming type_def's.
type completionConfigEnv struct {
	ConfigDir   string
	PackageOrg  string
	PackageName string
	Repository  string // path under completion/testdata; empty means no repository is configured
	ProjectRoot string // path under completion/testdata copied into the temporary workspace; empty creates a single-file project
	Packages    []completion.AvailablePackage
}

func TestEnumDeclarationCompletionConfigs(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	env := completionConfigEnv{
		ConfigDir:   "completion/testdata/enum_decl_ctx/config",
		PackageOrg:  "completion",
		PackageName: "enum_decl_ctx",
		Repository:  "repository",
		ProjectRoot: "enum_decl_ctx/source/projectls",
		Packages:    enumMemberAvailablePackages,
	}
	configs := discoverCompletionConfigs(t, platform, env.ConfigDir)
	if len(configs) != 5 {
		t.Fatalf("completion config count = %d, want 5", len(configs))
	}
	runCompletionConfigs(t, platform, env, configs)
}

func TestTypeDefinitionCompletionConfigs(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	env := completionConfigEnv{
		ConfigDir:   "completion/testdata/type_def/config",
		PackageOrg:  "completion",
		PackageName: "type_def",
		Repository:  "repository",
		Packages:    typeDefinitionAvailablePackages,
	}
	configs := discoverCompletionConfigs(t, platform, env.ConfigDir)
	if len(configs) != 7 {
		t.Fatalf("completion config count = %d, want 7", len(configs))
	}
	runCompletionConfigs(t, platform, env, configs)
}

// discoverCompletionConfigs lists configDir's config JSON files in a
// deterministic order. Any expected count is the caller's concern, not
// this generic directory scan's.
func discoverCompletionConfigs(t *testing.T, platform pal.Platform, configDir string) []string {
	t.Helper()
	entries, err := platform.FS.ReadDir(configDir)
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

func runCompletionConfigs(t *testing.T, platform pal.Platform, env completionConfigEnv, configs []string) {
	t.Helper()
	skips := skipList()
	for _, configPath := range configs {
		configPath := configPath
		t.Run(path.Base(configPath), func(t *testing.T) {
			if slices.Contains(skips, path.Clean(configPath)) {
				t.Skip()
			}
			config, raw := readCompletionConfig(t, platform, configPath)
			source, err := platform.FS.ReadFile(path.Join("completion/testdata", config.Source))
			if err != nil {
				t.Fatal(err)
			}
			actual := completionConfigRequest(t, platform, env, config.Source, string(source), config.Position)
			if *update {
				writeCompletionConfigItems(t, platform, configPath, raw, actual)
				return
			}
			compareCompletionConfigItems(t, config.Items, actual)
		})
	}
}

func readCompletionConfig(t *testing.T, platform pal.Platform, configPath string) (completionConfig, []byte) {
	t.Helper()
	content, err := platform.FS.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config completionConfig
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	return config, content
}

// writeCompletionConfigItems replaces only raw's top-level "items" value
// with items' encoding, leaving every other byte of the config file --
// including key order, spacing, and every other field -- untouched. Unlike
// a decode/re-encode round trip, this can never drift the upstream config's
// byte-for-byte fixture bytes for anything other than the field being
// updated.
func writeCompletionConfigItems(t *testing.T, platform pal.Platform, configPath string, raw []byte, items []protocol.CompletionItem) {
	t.Helper()
	encodedItems, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := spliceTopLevelJSONField(raw, "items", encodedItems)
	if err != nil {
		t.Fatal(err)
	}
	if err := platform.FS.WriteFile(configPath, updated); err != nil {
		t.Fatal(err)
	}
}

func copyCompletionProject(t *testing.T, platform pal.Platform, from, to string) {
	t.Helper()
	entries, err := platform.FS.ReadDir(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := platform.FS.MkdirAll(to, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		source := path.Join(from, entry.Name())
		destination := path.Join(to, entry.Name())
		if entry.IsDir() {
			copyCompletionProject(t, platform, source, destination)
			continue
		}
		content, err := platform.FS.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := platform.FS.WriteFile(destination, content); err != nil {
			t.Fatal(err)
		}
	}
}

func completionConfigRequest(t *testing.T, platform pal.Platform, env completionConfigEnv, sourcePath, source string, position protocol.Position) []protocol.CompletionItem {
	t.Helper()
	root := t.TempDir()
	documentPath := path.Join(root, "main.bal")
	if relative, ok := strings.CutPrefix(sourcePath, env.ProjectRoot+"/"); env.ProjectRoot != "" && ok {
		root = path.Join(root, path.Base(env.ProjectRoot))
		copyCompletionProject(t, platform, path.Join("completion/testdata", env.ProjectRoot), root)
		documentPath = path.Join(root, relative)
	} else {
		toml := fmt.Sprintf("[package]\norg = %q\nname = %q\nversion = \"0.1.0\"\n", env.PackageOrg, env.PackageName)
		if err := platform.FS.WriteFile(path.Join(root, "Ballerina.toml"), []byte(toml)); err != nil {
			t.Fatal(err)
		}
		if err := platform.FS.WriteFile(documentPath, []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	transport := &stepTransport{}
	bus := event.New()
	defer bus.Close()
	var repositories []projects.Repository
	if env.Repository != "" {
		repositories = append(repositories, workspace.NewFileSystemRepository(platform, path.Join("completion/testdata", env.Repository)))
	}
	projectService := workspace.New(platform, bus, workspace.WithRepositories(repositories))
	compiler := compile.New(projectService, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	srv := server.New(transport, projectService, compiler, bus, server.WithAvailablePackages(env.Packages))
	defer srv.Flush()

	messages := []protocol.Message{
		completionConfigMessage(1, "initialize", map[string]any{}),
		completionConfigMessage(nil, "initialized", map[string]any{}),
		completionConfigMessage(nil, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri": "file://" + documentPath, "languageId": "ballerina", "version": 1, "text": source,
			},
		}),
		completionConfigMessage(nil, "$pal/flush", map[string]any{}),
		completionConfigMessage(2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": "file://" + documentPath},
			"position":     position,
		}),
		completionConfigMessage(nil, "$pal/flushRequests", map[string]any{}),
	}
	var input bytes.Buffer
	for _, message := range messages {
		if err := protocol.WriteMessage(&input, message); err != nil {
			t.Fatal(err)
		}
	}
	transport.reader = bytes.NewReader(input.Bytes())
	if err := srv.Serve(); err != nil {
		t.Fatal(err)
	}
	responses, err := readMessages(transport.writer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, response := range responses {
		var message struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(response, &message); err != nil || message.ID != 2 {
			continue
		}
		var items []protocol.CompletionItem
		if err := json.Unmarshal(message.Result, &items); err != nil {
			t.Fatal(err)
		}
		return items
	}
	t.Fatal("completion response missing")
	return nil
}

// typeDefinitionAvailablePackages mirrors the package identities mocked by
// Java LS AbstractLSTest; production completion has no configured catalog.
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

// compareCompletionConfigItems requires actual and expected to hold the same
// multiset of signatures -- an exact, order-agnostic count comparison, not
// mere subset containment: each expected signature consumes one matching
// occurrence from actual's remaining counts, so a duplicate in expected must
// be matched by an equal duplicate in actual, and the upfront length check
// then guarantees actual carries nothing beyond what expected accounts for.
func compareCompletionConfigItems(t *testing.T, expected []completionConfigItem, actual []protocol.CompletionItem) {
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

// completionConfigSignature intentionally ignores Java-comparator fields that
// are not represented consistently here: kind, documentation, and insertTextFormat.
func completionConfigSignature(encoded []byte) string {
	var item completionConfigItem
	_ = json.Unmarshal(encoded, &item)
	additional := ""
	if len(item.AdditionalTextEdits) > 0 && string(item.AdditionalTextEdits) != "null" {
		additional = string(item.AdditionalTextEdits)
	}
	return strings.ReplaceAll(fmt.Sprintf("{%s,%s,%s,%s,%s,%s}", item.InsertText, item.Detail, item.Label, item.SortText, item.FilterText, additional), "\r\n", "\n")
}
