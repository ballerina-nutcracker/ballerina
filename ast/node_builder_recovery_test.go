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

package ast

import (
	"strings"
	"testing"

	"ballerina-lang-go/context"
	"ballerina-lang-go/parser"
	"ballerina-lang-go/parser/tree"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/tools/text"
)

func TestRecoveringNodeBuilderIncludesMinutiaeInNodeRanges(t *testing.T) {
	t.Parallel()

	source := "// doc\nfunction foo(int x) {\n\treturn;\n}\n"
	strict, _ := buildNodeBuilderCompilationUnit(t, source, false)
	recovering, _ := buildNodeBuilderCompilationUnit(t, source, true)

	strictFunction := strict.TopLevelNodes[0].(*BLangFunction)
	assertLocationOffsets(t, strictFunction.GetPosition(), strings.Index(source, "function"), strings.LastIndex(source, "}")+1)
	strictReturn := strictFunction.Body.(*BLangBlockFunctionBody).Stmts[0]
	assertLocationOffsets(t, strictReturn.GetPosition(), strings.Index(source, "return;"), strings.Index(source, "return;")+len("return;"))

	recoveringFunction := recovering.TopLevelNodes[0].(*BLangFunction)
	assertLocationOffsets(t, recoveringFunction.GetPosition(), 0, len(source))
	recoveringReturn := recoveringFunction.Body.(*BLangBlockFunctionBody).Stmts[0]
	assertLocationOffsets(t, recoveringReturn.GetPosition(), strings.Index(source, "\treturn;"), strings.Index(source, "\n}")+1)
}

func TestRecoveringNodeBuilderBadNodesCoverMinutiae(t *testing.T) {
	source := "// doc\nfunction foo() {}"
	_, syntaxTree := buildNodeBuilderCompilationUnit(t, source, true)
	modulePart := syntaxTree.RootNode.(*tree.ModulePart)
	members := modulePart.Members()
	member := members.Get(0)

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
	builder := NewRecoveringNodeBuilder(cx)
	bad := builder.badTopLevel(member)
	expected := member.TextRangeWithMinutiae()
	assertLocationOffsets(t, bad.GetPosition(), expected.StartOffset, expected.EndOffset)
}

func buildNodeBuilderCompilationUnit(t *testing.T, source string, recovering bool) (*BLangCompilationUnit, *tree.SyntaxTree) {
	t.Helper()
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
	syntaxTree, err := parser.GetSyntaxTree(cx, "test.bal", source)
	if err != nil {
		t.Fatal(err)
	}

	if !recovering {
		return GetCompilationUnit(cx, syntaxTree), syntaxTree
	}
	builder := NewRecoveringNodeBuilder(cx)
	return builder.TransformModulePart(syntaxTree.RootNode.(*tree.ModulePart)).(*BLangCompilationUnit), syntaxTree
}

func assertLocationOffsets(t *testing.T, location diagnostics.Location, start, end int) {
	t.Helper()
	if gotStart, gotEnd := location.StartOffset(), location.EndOffset(); gotStart != start || gotEnd != end {
		t.Fatalf("location = %d:%d, want %d:%d", gotStart, gotEnd, start, end)
	}
}
