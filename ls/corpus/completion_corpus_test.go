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

package corpus

import (
	"encoding/json"
	"io/fs"
	"os"
	"slices"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestNoResolveOrEditsCompletionCorpus(t *testing.T) {
	t.Parallel()
	fixturePath := "completion/testdata/no-resolve-or-edits.completion.json"
	runTranscript(t, fixturePath)
	content, err := fs.ReadFile(os.DirFS("."), fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture transcript
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result protocol.InitializeResult `json:"result"`
	}
	if err := json.Unmarshal(fixture.Expected[0], &response); err != nil {
		t.Fatal(err)
	}
	want := []string{":", "."}
	provider, ok := response.Result.Capabilities.CompletionProvider.Value()
	if !ok {
		t.Fatal("missing completion provider")
	}
	got, ok := provider.TriggerCharacters.Value()
	if !ok || !slices.Equal(got, want) {
		t.Fatalf("completion trigger characters=%q, want %q", got, want)
	}
}
