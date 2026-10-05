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
	"bytes"
	"encoding/json"
	"path"
	"reflect"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func TestSignatureJavaConfigs(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	const root = "signature/java/testdata"
	for _, directory := range []string{"statements/config", "expressions/config"} {
		entries, err := platform.FS.ReadDir(path.Join(root, directory))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			t.Run(entry.Name(), func(t *testing.T) {
				configPath := path.Join(root, directory, entry.Name())
				raw, err := platform.FS.ReadFile(configPath)
				if err != nil {
					t.Fatal(err)
				}
				var config struct {
					Position protocol.Position `json:"position"`
					Source   string            `json:"source"`
					Expected json.RawMessage   `json:"expected"`
				}
				if err := json.Unmarshal(raw, &config); err != nil {
					t.Fatal(err)
				}
				source, err := platform.FS.ReadFile(path.Join(root, config.Source))
				if err != nil {
					t.Fatal(err)
				}
				actual := signatureConfigRequest(t, platform, string(source), config.Position)
				if *update {
					updated, err := spliceTopLevelJSONField(raw, "expected", actual)
					if err != nil {
						t.Fatal(err)
					}
					if err := platform.FS.WriteFile(configPath, updated); err != nil {
						t.Fatal(err)
					}
				} else if !reflect.DeepEqual(normalizeMessages([]json.RawMessage{actual}), normalizeMessages([]json.RawMessage{config.Expected})) {
					t.Fatalf("signature mismatch\nactual: %s\nexpected: %s", actual, config.Expected)
				}
			})
		}
	}
}

func signatureConfigRequest(t *testing.T, platform pal.Platform, source string, position protocol.Position) json.RawMessage {
	t.Helper()
	documentPath := path.Join(t.TempDir(), "main.bal")
	if err := platform.FS.WriteFile(documentPath, []byte(source)); err != nil {
		t.Fatal(err)
	}
	transport := &stepTransport{}
	bus := event.New()
	defer bus.Close()
	projects := workspace.New(platform, bus)
	compiler := compile.New(projects, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	srv := server.New(transport, projects, compiler, bus)
	defer srv.Flush()
	messages := []protocol.Message{
		completionConfigMessage(1, "initialize", map[string]any{"capabilities": map[string]any{"textDocument": map[string]any{"signatureHelp": map[string]any{"signatureInformation": map[string]any{"parameterInformation": map[string]any{"labelOffsetSupport": true}}}}}}),
		completionConfigMessage(nil, "textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": "file://" + documentPath, "languageId": "ballerina", "version": 1, "text": source}}),
		completionConfigMessage(nil, "$pal/flush", map[string]any{}),
		completionConfigMessage(2, "textDocument/signatureHelp", map[string]any{"textDocument": map[string]any{"uri": "file://" + documentPath}, "position": position}),
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
		var message map[string]json.RawMessage
		if err := json.Unmarshal(response, &message); err != nil {
			t.Fatal(err)
		}
		if string(message["id"]) != "2" {
			continue
		}
		delete(message, "id")
		actual, err := json.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}
		return actual
	}
	t.Fatal("signature response missing")
	return nil
}
