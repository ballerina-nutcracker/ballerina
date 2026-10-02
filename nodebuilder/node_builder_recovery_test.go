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

package nodebuilder

import (
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

func TestRecoveringNodeBuilderIncludesMinutiaeInNodeRanges(t *testing.T) {
	t.Parallel()

	source := "// doc\nfunction foo(int x) {\n\treturn;\n}\n"
	strict, _ := buildNodeBuilderCompilationUnit(t, source, false)
	recovering, _ := buildNodeBuilderCompilationUnit(t, source, true)

	strictFunction := strict.TopLevelNodes[0].(*ast.BLangFunction)
	assertLocationOffsets(t, strictFunction.GetPosition(), strings.Index(source, "function"), strings.LastIndex(source, "}")+1)
	strictReturn := strictFunction.Body.(*ast.BLangBlockFunctionBody).Stmts[0]
	assertLocationOffsets(t, strictReturn.GetPosition(), strings.Index(source, "return;"), strings.Index(source, "return;")+len("return;"))

	recoveringFunction := recovering.TopLevelNodes[0].(*ast.BLangFunction)
	assertLocationOffsets(t, recoveringFunction.GetPosition(), 0, len(source))
	recoveringReturn := recoveringFunction.Body.(*ast.BLangBlockFunctionBody).Stmts[0]
	assertLocationOffsets(t, recoveringReturn.GetPosition(), strings.Index(source, "\treturn;"), strings.Index(source, "\n}")+1)
}

func TestRecoveringNodeBuilderReplacesMalformedQualifiedReferences(t *testing.T) {
	testCases := []struct {
		name        string
		source      string
		aliasValue  string
		nameValue   string
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
			missingName: true,
		},
		{
			name:        "quoted unsupported identifier",
			source:      "function foo() { x = mod:'_; }",
			aliasValue:  "mod",
			nameValue:   "_",
			missingName: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			compilationUnit, _ := buildNodeBuilderCompilationUnit(t, testCase.source, true)
			function := compilationUnit.TopLevelNodes[0].(*ast.BLangFunction)
			assignment := function.Body.(*ast.BLangBlockFunctionBody).Stmts[0].(*ast.BLangAssignment)
			if testCase.missingName {
				if _, ok := assignment.GetExpression().(*ast.BLangBadExprOrAction); !ok {
					t.Fatalf("expression = %T, want bad expression", assignment.GetExpression())
				}
				return
			}
			reference := assignment.GetExpression().(*ast.BLangVarRef)
			assertIdentifierValue(t, reference.PkgAlias, testCase.aliasValue)
			assertIdentifierValue(t, reference.VariableName, testCase.nameValue)
		})
	}
}

func TestRecoveringNodeBuilderReplacesBadAnnotationAttachment(t *testing.T) {
	source := "@mod:_{} function foo() {}"
	compilationUnit, _ := buildNodeBuilderCompilationUnit(t, source, true)
	if _, ok := compilationUnit.TopLevelNodes[0].(*ast.BLangBadTopLevelNode); !ok {
		t.Fatalf("declaration = %T, want bad declaration", compilationUnit.TopLevelNodes[0])
	}
}

func TestRecoveringNodeBuilderReplacesBadAnnotationAccess(t *testing.T) {
	source := "function foo() { x = Target.@mod:_; }"
	compilationUnit, _ := buildNodeBuilderCompilationUnit(t, source, true)
	function := compilationUnit.TopLevelNodes[0].(*ast.BLangFunction)
	assignment := function.Body.(*ast.BLangBlockFunctionBody).Stmts[0].(*ast.BLangAssignment)
	if _, ok := assignment.GetExpression().(*ast.BLangBadExprOrAction); !ok {
		t.Fatalf("expression = %T, want bad expression", assignment.GetExpression())
	}
}

func TestRecoveringNodeBuilderRetainsMalformedNestedBlockBoundary(t *testing.T) {
	for _, source := range []string{
		"function foo() { if true { int x=1; else { int y=2; } int later=3; } function valid() {}",
		"function foo() { if true int x=1; } else { int y=2; } int later=3; } function valid() {}",
	} {
		t.Run(source, func(t *testing.T) {
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
			tree, err := parser.GetSyntaxTree(cx, "test.bal", source)
			if err != nil {
				t.Fatal(err)
			}
			unit := GetRecoveredCompilationUnit(cx, tree)
			fn := unit.TopLevelNodes[0].(*ast.BLangFunction)
			statements := fn.Body.(*ast.BLangBlockFunctionBody).Stmts
			branch := statements[0].(*ast.BLangIf)
			body := branch.GetBody().Stmts
			if len(body) != 2 {
				t.Fatalf("nested statements = %d, want valid statement and bad boundary", len(body))
			}
			if _, ok := body[0].(*ast.BLangVariableDef); !ok {
				t.Fatalf("inner sibling = %T, want variable declaration", body[0])
			}
			bad, ok := body[1].(*ast.BLangBadStmt)
			if !ok {
				t.Fatalf("block boundary = %T, want bad statement", body[1])
			}
			members := tree.RootNode.(*st.ModulePart).Members()
			syntaxFn := members.Get(0).(*st.FunctionDefinition)
			syntaxStatements := syntaxFn.FunctionBody().(*st.FunctionBodyBlockNode).Statements()
			syntaxIf := syntaxStatements.Get(0).(*st.IfElseStatementNode)
			rangeWithMinutiae := syntaxIf.IfBody().TextRangeWithMinutiae()
			assertLocationOffsets(t, bad.GetPosition(), rangeWithMinutiae.StartOffset, rangeWithMinutiae.EndOffset)
			if _, ok := branch.GetElseStatement().(*ast.BLangBlockStmt).Stmts[0].(*ast.BLangVariableDef); !ok {
				t.Fatal("else sibling was discarded")
			}
			if _, ok := statements[1].(*ast.BLangVariableDef); !ok {
				t.Fatal("outer sibling was discarded")
			}
			if _, ok := unit.TopLevelNodes[1].(*ast.BLangFunction); !ok {
				t.Fatal("valid sibling function was discarded")
			}
			if len(cx.Diagnostics()) != 1 {
				t.Fatalf("diagnostics = %v, want one syntax diagnostic", cx.Diagnostics())
			}
		})
	}
}

func TestRecoveringNodeBuilderPreservesStandaloneBlockSiblings(t *testing.T) {
	source := "function foo() { { int '_ = 1; int good=2; } int later=3; }"
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
	tree, err := parser.GetSyntaxTree(cx, "test.bal", source)
	if err != nil {
		t.Fatal(err)
	}
	unit := GetRecoveredCompilationUnit(cx, tree)
	function := unit.TopLevelNodes[0].(*ast.BLangFunction)
	statements := function.Body.(*ast.BLangBlockFunctionBody).Stmts
	if len(statements) != 2 {
		t.Fatalf("outer statements = %d, want 2", len(statements))
	}
	block, ok := statements[0].(*ast.BLangBlockStmt)
	if !ok {
		t.Fatalf("standalone block = %T, want block statement", statements[0])
	}
	if len(block.Stmts) != 2 {
		t.Fatalf("inner statements = %d, want 2", len(block.Stmts))
	}
	bad, ok := block.Stmts[0].(*ast.BLangBadStmt)
	if !ok {
		t.Fatalf("malformed declaration = %T, want bad statement", block.Stmts[0])
	}
	assertLocationOffsets(t, bad.GetPosition(), strings.Index(source, "int '_"), strings.Index(source, "int good"))
	good, ok := block.Stmts[1].(*ast.BLangVariableDef)
	if !ok {
		t.Fatalf("inner sibling = %T, want variable declaration", block.Stmts[1])
	}
	assertIdentifierValue(t, good.Var.Name, "good")
	later, ok := statements[1].(*ast.BLangVariableDef)
	if !ok {
		t.Fatalf("outer sibling = %T, want variable declaration", statements[1])
	}
	assertIdentifierValue(t, later.Var.Name, "later")
	if len(cx.Diagnostics()) != 1 {
		t.Fatalf("diagnostics = %v, want one syntax diagnostic", cx.Diagnostics())
	}
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
	if cx.HasDiagnostics() {
		t.Fatal("parser reported diagnostics")
	}

	diagnosticsBeforeBuild := len(cx.Diagnostics())
	builder := newRecoveringNodeBuilder(cx)
	builder.transformModulePart(syntaxTree.RootNode.(*st.ModulePart))
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

func TestRecoveringNodeBuilderSignatureBoundaries(t *testing.T) {
	for _, source := range []string{
		"function _() {}",
		"function foo(int _) {}",
		"function foo(int... _) {}",
		"function foo(int x = ) {}",
		"function foo() returns mod:_ {}",
		"@mod:_ function foo() {}",
	} {
		t.Run(source, func(t *testing.T) {
			unit, _ := buildNodeBuilderCompilationUnit(t, source+" function valid() {}", true)
			if _, ok := unit.TopLevelNodes[0].(*ast.BLangBadTopLevelNode); !ok {
				t.Fatalf("declaration = %T, want bad declaration", unit.TopLevelNodes[0])
			}
			if _, ok := unit.TopLevelNodes[len(unit.TopLevelNodes)-1].(*ast.BLangFunction); !ok {
				t.Fatal("valid sibling was discarded")
			}
		})
	}
}

func TestRecoveringNodeBuilderRetainsClassMembersAndPackageDeclarations(t *testing.T) {
	source := "class C { int _; int good; function _(int x) {} function valid() { int x = ; } function init(int _) {} *mod:_; } function _() {} function valid() {}"
	unit, _ := buildNodeBuilderCompilationUnit(t, source, true)
	class := unit.TopLevelNodes[0].(*ast.BLangClassDefinition)
	if len(class.BadTopLevelNodes) != 4 || len(class.Fields) != 1 || len(class.Methods) != 1 || class.InitFunction != nil {
		t.Fatalf("members: bad=%d fields=%d methods=%d init=%v", len(class.BadTopLevelNodes), len(class.Fields), len(class.Methods), class.InitFunction)
	}
	method := class.Methods["valid"]
	decl := method.Body.(*ast.BLangBlockFunctionBody).Stmts[0].(*ast.BLangVariableDef)
	if _, ok := decl.Var.Expr.(*ast.BLangBadExprOrAction); !ok {
		t.Fatalf("method initializer = %T, want bad expression", decl.Var.Expr)
	}
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	pkg := ToPackageFromCompilationUnits(cx, []*ast.BLangCompilationUnit{unit})
	if len(pkg.BadTopLevelNodes) != 1 || len(pkg.ClassDefinitions) != 1 || len(pkg.Functions) != 1 || cx.HasDiagnostics() {
		t.Fatal("package assembly did not retain bad declarations and valid siblings")
	}
}

func TestRecoveringNodeBuilderExpressionBoundaries(t *testing.T) {
	for _, expression := range []string{"mod:_", "mod:_(1)", "foo(_ = 1)", "value._", "function(int _) returns int => 1", "(_)=>1", "-", "1 +", "value.@mod:_"} {
		t.Run(expression, func(t *testing.T) {
			unit, _ := buildNodeBuilderCompilationUnit(t, "function foo() { x = "+expression+"; }", true)
			fn := unit.TopLevelNodes[0].(*ast.BLangFunction)
			assignment := fn.Body.(*ast.BLangBlockFunctionBody).Stmts[0].(*ast.BLangAssignment)
			if _, ok := assignment.GetExpression().(*ast.BLangBadExprOrAction); !ok {
				t.Fatalf("expression = %T, want bad expression", assignment.GetExpression())
			}
		})
	}
}

func TestRecoveringNodeBuilderOptionalAccessAndRemoteCallNames(t *testing.T) {
	for _, expression := range []string{
		"value?._", "value?.'_", "value?.mod:_", "value?.mod:'_",
		"ep->_()", "ep->'_()", "ep->method(_ = 1)",
	} {
		t.Run(expression, func(t *testing.T) {
			source := "function foo() { x = " + expression + "; x = 1; } function valid() {}"
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
			tree, err := parser.GetSyntaxTree(cx, "test.bal", source)
			if err != nil {
				t.Fatal(err)
			}
			unit := GetRecoveredCompilationUnit(cx, tree)
			fn := unit.TopLevelNodes[0].(*ast.BLangFunction)
			statements := fn.Body.(*ast.BLangBlockFunctionBody).Stmts
			assignment := statements[0].(*ast.BLangAssignment)
			if _, ok := assignment.GetExpression().(*ast.BLangBadExprOrAction); !ok {
				t.Fatalf("expression = %T, want bad expression", assignment.GetExpression())
			}
			if len(cx.Diagnostics()) != 1 {
				t.Fatalf("diagnostics = %v, want one syntax diagnostic", cx.Diagnostics())
			}
			if _, ok := statements[1].(*ast.BLangAssignment); !ok {
				t.Fatal("valid sibling statement was discarded")
			}
			if _, ok := unit.TopLevelNodes[1].(*ast.BLangFunction); !ok {
				t.Fatal("valid sibling function was discarded")
			}
		})
	}
	for _, expression := range []string{"value?.member", "ep->method()"} {
		t.Run(expression, func(t *testing.T) {
			unit, _ := buildNodeBuilderCompilationUnit(t, "function foo() { x = "+expression+"; }", true)
			fn := unit.TopLevelNodes[0].(*ast.BLangFunction)
			assignment := fn.Body.(*ast.BLangBlockFunctionBody).Stmts[0].(*ast.BLangAssignment)
			switch expr := assignment.GetExpression().(type) {
			case *ast.BLangFieldBaseAccess:
				assertIdentifierValue(t, expr.Field, "member")
			case *ast.BLangRemoteMethodCallAction:
				assertIdentifierValue(t, expr.Name, "method")
			default:
				t.Fatalf("valid expression = %T", expr)
			}
		})
	}
}

func TestRecoveringNodeBuilderConstructorNamedArguments(t *testing.T) {
	for _, expression := range []string{"new C(_ = 1)", "new(_ = 1)", "new C('_ = 1)", "new('_ = 1)", "new C(value = 1)", "new(value = 1)"} {
		t.Run(expression, func(t *testing.T) {
			source := "function foo() { C x = " + expression + "; int _ = 1; } function valid() {}"
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
			tree, err := parser.GetSyntaxTree(cx, "test.bal", source)
			if err != nil {
				t.Fatal(err)
			}
			unit := GetRecoveredCompilationUnit(cx, tree)
			fn := unit.TopLevelNodes[0].(*ast.BLangFunction)
			statements := fn.Body.(*ast.BLangBlockFunctionBody).Stmts
			decl := statements[0].(*ast.BLangVariableDef)
			if strings.Contains(expression, "_") {
				if _, ok := decl.Var.Expr.(*ast.BLangBadExprOrAction); !ok {
					t.Fatalf("initializer = %T, want bad expression", decl.Var.Expr)
				}
				if len(cx.Diagnostics()) != 1 {
					t.Fatalf("diagnostics = %v, want one syntax diagnostic", cx.Diagnostics())
				}
			} else {
				constructor, ok := decl.Var.Expr.(*ast.BLangNewExpression)
				if !ok {
					t.Fatalf("initializer = %T, want constructor", decl.Var.Expr)
				}
				argument := constructor.ArgsExprs[0].(*ast.BLangNamedArgsExpression)
				assertIdentifierValue(t, argument.Name, "value")
				if cx.HasDiagnostics() {
					t.Fatalf("valid constructor diagnostics = %v", cx.Diagnostics())
				}
			}
			ignore := statements[1].(*ast.BLangVariableDef)
			assertIdentifierValue(t, ignore.Var.Name, "_")
			if _, ok := unit.TopLevelNodes[1].(*ast.BLangFunction); !ok {
				t.Fatal("valid sibling function was discarded")
			}
		})
	}
}

func TestRecoveringNodeBuilderRetainsServiceMembers(t *testing.T) {
	unit, _ := buildNodeBuilderCompilationUnit(t, "service / on endpoint { int _; int valid; resource function get path(int _) {} function valid() { int x = ; } }", true)
	service := unit.TopLevelNodes[0].(*ast.BLangService)
	if len(service.BadTopLevelNodes) != 2 || len(service.Fields) != 1 || len(service.Methods) != 1 || len(service.ResourceMethods) != 0 {
		t.Fatalf("members: bad=%d fields=%d methods=%d resources=%d", len(service.BadTopLevelNodes), len(service.Fields), len(service.Methods), len(service.ResourceMethods))
	}
}

func TestRecoveringNodeBuilderReportsPreviouslySilentIdentifiers(t *testing.T) {
	for _, source := range []string{"function _() {}", "function foo() { x = mod:'_; }", "function foo() { int '_ = 1; }"} {
		t.Run(source, func(t *testing.T) {
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
			tree, err := parser.GetSyntaxTree(cx, "test.bal", source)
			if err != nil {
				t.Fatal(err)
			}
			GetRecoveredCompilationUnit(cx, tree)
			if len(cx.Diagnostics()) != 1 {
				t.Fatalf("diagnostics = %v, want one syntax diagnostic", cx.Diagnostics())
			}
		})
	}
}

func TestRecoveringNodeBuilderPreservesIgnoreIdentifiers(t *testing.T) {
	unit, _ := buildNodeBuilderCompilationUnit(t, "function foo() { int _ = 1; }", true)
	fn := unit.TopLevelNodes[0].(*ast.BLangFunction)
	decl := fn.Body.(*ast.BLangBlockFunctionBody).Stmts[0].(*ast.BLangVariableDef)
	assertIdentifierValue(t, decl.Var.Name, "_")
}

func TestRecoveringNodeBuilderBadNodesCoverMinutiae(t *testing.T) {
	source := "// doc\nfunction foo() {}"
	_, syntaxTree := buildNodeBuilderCompilationUnit(t, source, true)
	modulePart := syntaxTree.RootNode.(*st.ModulePart)
	members := modulePart.Members()
	member := members.Get(0)

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
	builder := newRecoveringNodeBuilder(cx)
	bad := builder.badTopLevel(member)
	expected := member.TextRangeWithMinutiae()
	assertLocationOffsets(t, bad.GetPosition(), expected.StartOffset, expected.EndOffset)
}

func buildNodeBuilderCompilationUnit(t *testing.T, source string, recovering bool) (*ast.BLangCompilationUnit, *st.SyntaxTree) {
	t.Helper()
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
	syntaxTree, err := parser.GetSyntaxTree(cx, "test.bal", source)
	if err != nil {
		t.Fatal(err)
	}
	if cx.HasDiagnostics() {
		t.Fatal("parser reported diagnostics")
	}

	if !recovering {
		return GetCompilationUnit(cx, syntaxTree), syntaxTree
	}
	builder := newRecoveringNodeBuilder(cx)
	return builder.transformModulePart(syntaxTree.RootNode.(*st.ModulePart)).(*ast.BLangCompilationUnit), syntaxTree
}

func assertIdentifierValue(t *testing.T, identifier ast.IdentifierNode, value string) {
	t.Helper()
	if _, ok := identifier.(*ast.BLangIdentifier); !ok {
		t.Fatalf("identifier = %T, want *BLangIdentifier", identifier)
	}
	if got := identifier.GetValue(); got != value {
		t.Fatalf("identifier value = %q, want %q", got, value)
	}
}

func assertLocationOffsets(t *testing.T, location diagnostics.Location, start, end int) {
	t.Helper()
	if gotStart, gotEnd := location.StartOffset(), location.EndOffset(); gotStart != start || gotEnd != end {
		t.Fatalf("location = %d:%d, want %d:%d", gotStart, gotEnd, start, end)
	}
}
