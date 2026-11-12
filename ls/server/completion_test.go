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
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/completion"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestCompletionOnStaleTextIsIncomplete(t *testing.T) {
	t.Parallel()
	projects, compiler, bus := newCoreServices()
	defer bus.Close()
	defer compiler.Shutdown()
	uri, _ := workspace.FileURIFromPath(filepath.Join(t.TempDir(), "main.bal"))
	requestText := "public function main() {\n    \n}\n"
	sealedText := "public function main() {\n    int x = 1;\n    \n}\n"
	if _, err := projects.Apply(context.Background(), workspace.DocumentChange{
		Kind: workspace.ChangeOpen, URI: uri, ClientURI: uri.String(), Text: sealedText, Version: 2,
	}); err != nil {
		t.Fatal(err)
	}
	compiler.Flush()
	srv := New(&batchTransport{}, projects, compiler, nil)
	got := srv.handleCompletion(context.Background(), completion.Request{URI: uri, Text: requestText, Offset: 29})
	if got.err != nil {
		t.Fatalf("handleCompletion error = %v", got.err)
	}
	encoded, err := json.Marshal(got.result)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.result.(protocol.CompletionList); !ok {
		t.Fatalf("handleCompletion result = %T %s, want protocol.CompletionList", got.result, encoded)
	}
	if want := `{"isIncomplete":true,"items":[]}`; string(encoded) != want {
		t.Fatalf("handleCompletion result = %s, want %s", encoded, want)
	}
}
