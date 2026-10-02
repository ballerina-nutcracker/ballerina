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
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

// Corpus error markers cannot assert diagnostic identity or recovered AST shape.
func TestRecoveringNodeBuilderRetainsLeadingZeroParserDiagnostic(t *testing.T) {
	source := "function foo() { x = 1 + 02; } function valid() {}"
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
	tree, err := parser.GetSyntaxTree(cx, "test.bal", source)
	if err != nil {
		t.Fatal(err)
	}
	parserNode := st.FindDeepestDiagnosticSTNode(tree.RootNode.InternalNode())
	if parserNode == nil || len(parserNode.Diagnostics()) != 1 {
		t.Fatal("want one parser token diagnostic")
	}
	parserMessage := diagnosticMessage(parserNode.Diagnostics()[0])
	unit := GetRecoveredCompilationUnit(cx, tree)
	if len(cx.Diagnostics()) != 1 || cx.Diagnostics()[0].Message() != parserMessage {
		t.Fatalf("diagnostics = %v, want original parser diagnostic only", cx.Diagnostics())
	}
	fn := unit.TopLevelNodes[0].(*ast.BLangFunction)
	assignment := fn.Body.(*ast.BLangBlockFunctionBody).Stmts[0].(*ast.BLangAssignment)
	binary := assignment.GetExpression().(*ast.BLangBinaryExpr)
	if _, ok := binary.RhsExpr.(*ast.BLangBadExprOrAction); !ok {
		t.Fatalf("right operand = %T, want bad expression", binary.RhsExpr)
	}
	if _, ok := unit.TopLevelNodes[1].(*ast.BLangFunction); !ok {
		t.Fatal("valid sibling was discarded")
	}
}
