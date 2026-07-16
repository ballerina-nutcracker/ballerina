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

func TestRecoveringNodeBuilderPreservesQualifiedReferenceIdentifiers(t *testing.T) {
	testCases := []struct {
		name        string
		source      string
		aliasValue  string
		nameValue   string
		badOriginal string
		isLiteral   bool
		missingName bool
	}{
		{
			name:       "valid",
			source:     "function foo() { x = mod:name; }",
			aliasValue: "mod",
			nameValue:  "name",
		},
		{
			name:        "missing name",
			source:      "function foo() { x = mod:; }",
			aliasValue:  "mod",
			missingName: true,
		},
		{
			name:        "unsupported identifier",
			source:      "function foo() { x = mod:_ ; }",
			aliasValue:  "mod",
			nameValue:   "_",
			badOriginal: "_",
			missingName: true,
		},
		{
			name:        "quoted unsupported identifier",
			source:      "function foo() { x = mod:'_; }",
			aliasValue:  "mod",
			nameValue:   "_",
			badOriginal: "'_",
			isLiteral:   true,
			missingName: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			compilationUnit, _ := buildNodeBuilderCompilationUnit(t, testCase.source, true)
			function := compilationUnit.TopLevelNodes[0].(*BLangFunction)
			assignment := function.Body.(*BLangBlockFunctionBody).Stmts[0].(*BLangAssignment)
			reference := assignment.GetExpression().(*BLangSimpleVarRef)

			assertIdentifierValue(t, reference.PkgAlias, testCase.aliasValue)
			if testCase.missingName {
				bad, ok := reference.VariableName.(*BLangBadIdentifier)
				if !ok {
					t.Fatalf("variable name = %T, want *BLangBadIdentifier", reference.VariableName)
				}
				if bad.Value != testCase.nameValue || bad.OriginalValue != testCase.badOriginal {
					t.Fatalf("bad identifier values = %q, %q, want %q, %q", bad.Value, bad.OriginalValue, testCase.nameValue, testCase.badOriginal)
				}
				if bad.IsLiteral() != testCase.isLiteral {
					t.Fatalf("bad identifier IsLiteral() = %t, want %t", bad.IsLiteral(), testCase.isLiteral)
				}
				return
			}
			assertIdentifierValue(t, reference.VariableName, testCase.nameValue)
		})
	}
}

func TestRecoveringNodeBuilderPreservesBadAnnotationAttachmentIdentifier(t *testing.T) {
	source := "@mod:_{} function foo() {}"
	compilationUnit, _ := buildNodeBuilderCompilationUnit(t, source, true)
	function := compilationUnit.TopLevelNodes[0].(*BLangFunction)
	attachments := function.GetAnnotationAttachments()
	if len(attachments) != 1 {
		t.Fatalf("annotation attachment count = %d, want 1", len(attachments))
	}
	attachment := attachments[0]
	assertIdentifierValue(t, attachment.GetPackageAlias(), "mod")
	assertBadIdentifier(t, attachment.GetAnnotationName(), "_", "_", strings.Index(source, "_"), strings.Index(source, "_")+1)
}

func TestRecoveringNodeBuilderPreservesBadAnnotationAccessIdentifier(t *testing.T) {
	source := "function foo() { x = Target.@mod:_; }"
	compilationUnit, _ := buildNodeBuilderCompilationUnit(t, source, true)
	function := compilationUnit.TopLevelNodes[0].(*BLangFunction)
	assignment := function.Body.(*BLangBlockFunctionBody).Stmts[0].(*BLangAssignment)
	access := assignment.GetExpression().(*BLangAnnotAccessExpr)
	assertIdentifierValue(t, access.PkgAlias, "mod")
	assertBadIdentifier(t, access.AnnotationName, "_", "_", strings.Index(source, "_"), strings.Index(source, "_")+1)
}

func TestRecoveringNodeBuilderReportsNestedSyntaxDiagnosticOnce(t *testing.T) {
	source := "function foo() { int x = ; }"
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
	syntaxTree, err := parser.GetSyntaxTree(cx, "test.bal", source)
	if err != nil {
		t.Fatal(err)
	}

	diagnosticsBeforeBuild := len(cx.Diagnostics())
	builder := NewRecoveringNodeBuilder(cx)
	builder.TransformModulePart(syntaxTree.RootNode.(*tree.ModulePart))
	if got := len(cx.Diagnostics()) - diagnosticsBeforeBuild; got != 1 {
		t.Fatalf("node builder reported %d syntax diagnostics, want 1", got)
	}
}

func TestRecoveringNodeBuilderHandlesMissingIdentifiers(t *testing.T) {
	testCases := []struct {
		name   string
		source string
	}{
		{name: "function name", source: "function () {}"},
		{name: "parameter name", source: "function foo(int ) {}"},
		{name: "variable name", source: "function foo() { int = 1; }"},
		{name: "named argument name", source: "function foo() { foo(=1); }"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			compilationUnit, _ := buildNodeBuilderCompilationUnit(t, testCase.source, true)
			if len(compilationUnit.TopLevelNodes) == 0 {
				t.Fatal("expected recovered top-level node")
			}
		})
	}
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

func assertIdentifierValue(t *testing.T, identifier IdentifierNode, value string) {
	t.Helper()
	if _, ok := identifier.(*BLangIdentifier); !ok {
		t.Fatalf("identifier = %T, want *BLangIdentifier", identifier)
	}
	if got := identifier.GetValue(); got != value {
		t.Fatalf("identifier value = %q, want %q", got, value)
	}
}

func assertBadIdentifier(t *testing.T, identifier IdentifierNode, value, originalValue string, start, end int) {
	t.Helper()
	bad, ok := identifier.(*BLangBadIdentifier)
	if !ok {
		t.Fatalf("identifier = %T, want *BLangBadIdentifier", identifier)
	}
	if bad.Value != value || bad.OriginalValue != originalValue {
		t.Fatalf("bad identifier values = %q, %q, want %q, %q", bad.Value, bad.OriginalValue, value, originalValue)
	}
	assertLocationOffsets(t, bad.GetPosition(), start, end)
}

func assertLocationOffsets(t *testing.T, location diagnostics.Location, start, end int) {
	t.Helper()
	if gotStart, gotEnd := location.StartOffset(), location.EndOffset(); gotStart != start || gotEnd != end {
		t.Fatalf("location = %d:%d, want %d:%d", gotStart, gotEnd, start, end)
	}
}
