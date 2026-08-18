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

package types

import (
	goAst "go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTypeResolverMutationsUseHelpers parses every production Go file in this
// package other than type_resolver_state.go and rejects tracked AST or symbol
// state access. A direct call to a tracked getter or setter, or an assignment to
// a tracked field, therefore fails at its source position. This makes
// type_resolver_state.go the single boundary for state that ephemeral resolution
// must be able to isolate.
//
// This is a syntactic boundary test. It covers the methods and fields listed
// below; adding another kind of mutable resolution state requires adding its
// selector to the relevant set.
func TestTypeResolverMutationsUseHelpers(t *testing.T) {
	trackedCalls := map[string]bool{
		"GetDeterminedType": true,
		"SetDeterminedType": true,
		"SetCallArgs":       true,
		"SetResolvedSymbol": true,
		"SetSymbol":         true,
		"SetMethodSymbol":   true,
		"SetValue":          true,
		"SetTypedSignature": true,
		"SetParamTypes":     true,
		"SetReturnType":     true,
	}
	trackedAssignments := map[string]bool{
		"ArgsExprs":        true,
		"ClassSymbol":      true,
		"Constraint":       true,
		"AnnotationValues": true,
		"AtomicType":       true,
		"FieldDefaults":    true,
		"NonGroupingKeys":  true,
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "type_resolver_state.go" {
			continue
		}
		path := filepath.Clean(name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, path, data, 0)
		if err != nil {
			t.Fatal(err)
		}
		goAst.Inspect(file, func(node goAst.Node) bool {
			switch node := node.(type) {
			case *goAst.CallExpr:
				selector, ok := node.Fun.(*goAst.SelectorExpr)
				if ok && trackedCalls[selector.Sel.Name] {
					t.Errorf("%s: direct %s call outside type_resolver_state.go", fset.Position(node.Pos()), selector.Sel.Name)
				}
			case *goAst.AssignStmt:
				for _, lhs := range node.Lhs {
					selector, ok := lhs.(*goAst.SelectorExpr)
					if ok && trackedAssignments[selector.Sel.Name] {
						t.Errorf("%s: direct assignment to %s outside type_resolver_state.go", fset.Position(lhs.Pos()), selector.Sel.Name)
					}
				}
			}
			return true
		})
	}
}
