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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package projects

import (
	"fmt"
	"strings"
	"testing"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// TestParseDocumentsParallelPreservesDocumentOrder guards against
// parseDocumentsParallel returning syntax trees in goroutine-completion
// order instead of document order (#1364): AST-building and anonymous-name
// assignment walk the returned slice in order, so a reordering here makes
// anonymous names nondeterministic across runs of identical source.
func TestParseDocumentsParallelPreservesDocumentOrder(t *testing.T) {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)

	const numDocs = 30
	names := make([]string, numDocs)
	for i := range names {
		names[i] = fmt.Sprintf("doc%02d.bal", i)
	}

	for iter := 0; iter < 20; iter++ {
		cx := compilercontext.NewCompilerContext(env)

		srcDocIDs := make([]DocumentID, numDocs)
		srcDocContextMap := make(map[DocumentID]*documentContext, numDocs)
		for i, name := range names {
			// Vary content size so documents don't all take the same time to
			// parse: earlier documents are bigger (slower) than later ones,
			// so a naive append-on-completion would tend to reorder them.
			lines := numDocs - i
			var body strings.Builder
			fmt.Fprintf(&body, "public function f%d() {\n", i)
			for l := 0; l < lines; l++ {
				fmt.Fprintf(&body, "    int v%d = %d;\n", l, l)
			}
			body.WriteString("}\n")

			docID := newDocumentIDFromString(fmt.Sprintf("doc-%d", i), name, ModuleID{})
			docConfig := NewDocumentConfig(docID, name, body.String())
			srcDocIDs[i] = docID
			srcDocContextMap[docID] = newDocumentContext(docConfig, false, "")
		}

		syntaxTrees := parseDocumentsParallel(cx, srcDocIDs, srcDocContextMap, nil, nil)
		if len(syntaxTrees) != numDocs {
			t.Fatalf("iteration %d: got %d syntax trees, want %d", iter, len(syntaxTrees), numDocs)
		}
		for i, tree := range syntaxTrees {
			if got := tree.FilePath(); got != names[i] {
				t.Fatalf("iteration %d: syntaxTrees[%d].FilePath() = %q, want %q (order not preserved)", iter, i, got, names[i])
			}
		}
	}
}

func TestNewModuleCompilerContextInitializesStats(t *testing.T) {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)
	name := NewDefaultModuleName(NewPackageName("module"))

	compilerCtx := newModuleCompilerContext(env, name)
	stats := compilerCtx.GetModuleStats()
	if stats == nil {
		t.Fatal("expected module stats")
	}
	if stats.ModuleName != name.String() {
		t.Fatalf("module name = %q, want %q", stats.ModuleName, name.String())
	}
}
