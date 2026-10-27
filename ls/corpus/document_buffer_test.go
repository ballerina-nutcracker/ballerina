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
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
)

func TestDocumentBufferSynchronization(t *testing.T) {
	for _, tc := range []struct {
		name    string
		text    string
		version int32
	}{
		{"full-text-replacement", "// accepted\npublic function main() {\n    int after = 3;\n}\n", 5},
		{"empty-text-replacement", "", 4},
		{"line-end-insertion", "public function main() {\n    int unused = 1; int added = 2;\n}\n", 2},
		{"crlf-line-end-insertion", "public function main() {\r\n    int unused = 1; int added = 2;\r\n}\r\n", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runTranscriptWithCheck(t, "sync/testdata/"+tc.name+".didChange.json", func(ws *workspace.ProjectService, _ *compile.CompilationService, root string) {
				uri, err := workspace.FileURIFromPath(filepath.Join(root, "buffer.bal"))
				if err != nil {
					t.Fatal(err)
				}
				snap, ok := ws.Snapshot(uri)
				if !ok || snap.Text != tc.text || snap.Version != tc.version {
					t.Fatalf("document buffer = %q at version %d (open=%v), want %q at version %d", snap.Text, snap.Version, ok, tc.text, tc.version)
				}
			})
		})
	}
}
