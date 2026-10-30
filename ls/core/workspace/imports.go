// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package workspace

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/st"
)

type importKey struct {
	org    string
	module string
}

func parseImportSet(content string) (map[importKey]struct{}, bool) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	tree, err := parser.GetSyntaxTree(context.NewCompilerContext(env), "import-set", content)
	if err != nil || tree == nil {
		return nil, false
	}
	return importSet(tree), true
}

func importSet(tree *st.SyntaxTree) map[importKey]struct{} {
	set := make(map[importKey]struct{})
	modulePart, ok := tree.RootNode.(*st.ModulePart)
	if !ok {
		return set
	}
	imports := modulePart.Imports()
	for i := 0; i < imports.Size(); i++ {
		decl := imports.Get(i)
		if decl.HasDiagnostics() {
			continue
		}
		set[importKeyOf(decl)] = struct{}{}
	}
	return set
}

func importKeyOf(decl *st.ImportDeclarationNode) importKey {
	var key importKey
	if org := decl.OrgName(); org != nil && org.OrgName() != nil {
		key.org = unquoted(org.OrgName().Text())
	}
	var parts []string
	moduleName := decl.ModuleName()
	for ident := range moduleName.Iterator() {
		parts = append(parts, unquoted(ident.Text()))
	}
	key.module = strings.Join(parts, ".")
	return key
}

func unquoted(text string) string {
	return strings.TrimPrefix(text, "'")
}

func sameImportSet(a, b map[importKey]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if _, ok := b[key]; !ok {
			return false
		}
	}
	return true
}
