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
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
)

func TestContractInvalidation(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"return-type", "parameter-name", "parameter-default", "record-default", "diamond"} {
		for _, reverse := range []bool{false, true} {
			scenario := name
			if reverse {
				scenario += "-reverse"
			}
			t.Run(scenario, func(t *testing.T) {
				t.Parallel()
				runTranscriptWithCheck(t, "invalidation/testdata/"+scenario+".json", func(ws *workspace.ProjectService, compiler *compile.CompilationService, root string) {
					uri, _ := workspace.FileURIFromPath(filepath.Join(root, "modules/c/c.bal"))
					diags, gen, ok := compiler.DiagnosticsFor(root)
					if !ok || gen < 3 {
						t.Fatalf("missing edited generation: %d, available=%v", gen, ok)
					}
					if reverse {
						if len(diags[uri]) != 0 {
							t.Fatalf("reverse edit must clear consumer diagnostics: %v", diags[uri])
						}
					} else {
						want := "incompatible type"
						switch name {
						case "parameter-name":
							want = "count"
						case "parameter-default":
							want = "count"
						case "record-default":
							want = "count"
						}
						found := false
						for _, diagnostic := range diags[uri] {
							found = found || strings.Contains(strings.ToLower(diagnostic.Message), want)
						}
						if !found {
							t.Fatalf("missing required consumer diagnostic containing %q: %v", want, diags[uri])
						}
					}
					assertFreshEventDiagnostics(t, ws, compiler, root, root)
				})
			})
		}
	}
}
