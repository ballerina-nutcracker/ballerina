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
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type recoveryPointCase struct {
	name    string
	source  string
	recover func(t *testing.T, n *nodeBuilder, root st.Node) (ast.Node, st.Node)
	check   func(ast.Node) bool
}

// TestRecoveryPointsProduceBadNodes checks recovery-point positions with strict ranges, which recovery corpus
// goldens do not expose. Bad identifier positions are absent from those goldens, so both ranges are checked for them.
func TestRecoveryPointsProduceBadNodes(t *testing.T) {
	t.Parallel()

	cases := []recoveryPointCase{
		{
			name:   "top-level",
			source: "// leading comment\nfunction f() returns int $ { return 1; }",
			recover: func(t *testing.T, n *nodeBuilder, root st.Node) (ast.Node, st.Node) {
				node := findSTNode[*st.FunctionDefinition](t, root)
				return n.transformTopLevel(node), node
			},
			check: func(node ast.Node) bool { _, ok := node.(*ast.BLangBadTopLevelNode); return ok },
		},
		{
			name:   "statement",
			source: "function f() {\n    break $;\n}",
			recover: func(t *testing.T, n *nodeBuilder, root st.Node) (ast.Node, st.Node) {
				node := findSTNode[*st.BreakStatementNode](t, root)
				return n.transformStatement(node), node
			},
			check: func(node ast.Node) bool { _, ok := node.(*ast.BLangBadStmt); return ok },
		},
		{
			name:   "expression",
			source: "function f() {\n    int x =\n        (1 + 2 $);\n}",
			recover: func(t *testing.T, n *nodeBuilder, root st.Node) (ast.Node, st.Node) {
				node := findSTNode[*st.BracedExpressionNode](t, root)
				return n.transformExpr(node), node
			},
			check: func(node ast.Node) bool { _, ok := node.(*ast.BLangBadExprOrAction); return ok },
		},
		{
			name:   "action or expression",
			source: "function f() {\n    int x =\n        (1 + 2 $);\n}",
			recover: func(t *testing.T, n *nodeBuilder, root st.Node) (ast.Node, st.Node) {
				node := findSTNode[*st.BracedExpressionNode](t, root)
				return n.transformActionOrExpr(node), node
			},
			check: func(node ast.Node) bool { _, ok := node.(*ast.BLangBadExprOrAction); return ok },
		},
		{
			name:   "type",
			source: "function f((int $) a) { }",
			recover: func(t *testing.T, n *nodeBuilder, root st.Node) (ast.Node, st.Node) {
				node := findSTNode[*st.ParenthesisedTypeDescriptorNode](t, root)
				return n.transformType(node), node
			},
			check: func(node ast.Node) bool { _, ok := node.(*ast.BLangBadTypeNode); return ok },
		},
		{
			name:   "identifier",
			source: "function f() { int = 1; }",
			recover: func(t *testing.T, n *nodeBuilder, root st.Node) (ast.Node, st.Node) {
				token := findSTNode[*st.CaptureBindingPatternNode](t, root).VariableName()
				return n.transformIdentifier(token), token
			},
			check: func(node ast.Node) bool { _, ok := node.(*ast.BLangBadIdentifier); return ok },
		},
	}
	for _, testCase := range cases {
		ranges := map[string]rangeFn{"strict": st.Node.TextRange}
		if testCase.name == "identifier" {
			ranges["recovered"] = st.Node.TextRangeWithMinutiae
		}
		for rangeName, rangeOf := range ranges {
			t.Run(testCase.name+"/"+rangeName, func(t *testing.T) {
				t.Parallel()
				builder, syntaxTree := newTestNodeBuilder(t, testCase.source, rangeOf)
				bad, failing := testCase.recover(t, builder, syntaxTree.RootNode)
				if bad == nil || !testCase.check(bad) {
					t.Fatalf("recovery point returned %T", bad)
				}
				if len(builder.cx.Diagnostics()) == 0 {
					t.Fatal("recovery point did not report the failure")
				}
				expected := rangeOf(failing)
				assertLocationOffsets(t, bad.GetPosition(), expected.StartOffset, expected.EndOffset)
			})
		}
	}
}

func parseTestSourceWithContext(t *testing.T, source string) (*context.CompilerContext, *st.SyntaxTree) {
	t.Helper()
	cx := context.NewCompilerContext(context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false))
	syntaxTree, err := parser.GetSyntaxTree(cx, "test.bal", source)
	if err != nil {
		t.Fatal(err)
	}
	return cx, syntaxTree
}

func newTestNodeBuilder(t *testing.T, source string, rangeOf rangeFn) (*nodeBuilder, *st.SyntaxTree) {
	t.Helper()
	cx, syntaxTree := parseTestSourceWithContext(t, source)
	return newNodeBuilder(cx, rangeOf), syntaxTree
}

func findSTNode[T st.Node](t *testing.T, root st.Node) T {
	t.Helper()
	found, ok := searchSTNode[T](root)
	if !ok {
		var zero T
		t.Fatalf("no %T in the syntax tree", zero)
	}
	return found
}

func searchSTNode[T st.Node](node st.Node) (T, bool) {
	var zero T
	if typed, ok := node.(T); ok {
		return typed, true
	}
	nonTerminal, ok := node.(st.NonTerminalNode)
	if !ok {
		return zero, false
	}
	for child := range nonTerminal.ChildNodes() {
		if child == nil {
			continue
		}
		if found, ok := searchSTNode[T](child); ok {
			return found, true
		}
	}
	return zero, false
}

func assertLocationOffsets(t *testing.T, location diagnostics.Location, start, end int) {
	t.Helper()
	if gotStart, gotEnd := location.StartOffset(), location.EndOffset(); gotStart != start || gotEnd != end {
		t.Fatalf("location = %d:%d, want %d:%d", gotStart, gotEnd, start, end)
	}
}
