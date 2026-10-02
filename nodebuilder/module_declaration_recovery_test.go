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
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

func TestRecoveringNodeBuilderModuleDeclarationBoundaries(t *testing.T) {
	for _, declaration := range []string{
		"int x = 1 + ;", "int x = mod:_;", "mod:_ x = 1;",
		"const x = 1 + ;", "const mod:_ x = 1;",
		"type A mod:_;", "type A mod:;",
	} {
		t.Run(declaration, func(t *testing.T) {
			source := declaration + " function valid() {}"
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			cx.DiagnosticEnv().RegisterFile("test.bal", text.TextDocumentFromText(source))
			tree, err := parser.GetSyntaxTree(cx, "test.bal", source)
			if err != nil {
				t.Fatal(err)
			}
			unit := GetRecoveredCompilationUnit(cx, tree)
			switch node := unit.TopLevelNodes[0].(type) {
			case *ast.BLangVariable:
				_, badType := node.TypeNode().(*ast.BLangBadTypeNode)
				_, badExpr := node.Expr.(*ast.BLangBadExprOrAction)
				if !badType && !badExpr {
					t.Fatalf("variable has no bad child: %+v", node)
				}
			case *ast.BLangTypeDefinition:
				if _, ok := node.GetTypeData().TypeDescriptor.(*ast.BLangBadTypeNode); !ok {
					t.Fatalf("type descriptor = %T, want bad type", node.GetTypeData().TypeDescriptor)
				}
			default:
				t.Fatalf("declaration = %T, want ordinary declaration with bad child", node)
			}
			if len(cx.Diagnostics()) != 1 {
				t.Fatalf("diagnostics = %v, want one syntax diagnostic", cx.Diagnostics())
			}
			if _, ok := unit.TopLevelNodes[1].(*ast.BLangFunction); !ok {
				t.Fatal("valid sibling was discarded")
			}
		})
	}
}

func TestNodeBuilderValidModuleDeclarationBoundaries(t *testing.T) {
	for _, recovering := range []bool{false, true} {
		unit, _ := buildNodeBuilderCompilationUnit(t, "int x = 1 + 2; const int y = 3; type A mod:Name; function valid() {}", recovering)
		if len(unit.TopLevelNodes) != 4 {
			t.Fatalf("declarations = %d, want 4", len(unit.TopLevelNodes))
		}
		for _, index := range []int{0, 1} {
			variable := unit.TopLevelNodes[index].(*ast.BLangVariable)
			if _, bad := variable.TypeNode().(*ast.BLangBadTypeNode); bad {
				t.Fatal("valid type was discarded")
			}
			if _, bad := variable.Expr.(*ast.BLangBadExprOrAction); bad {
				t.Fatal("valid initializer was discarded")
			}
		}
		typeDef := unit.TopLevelNodes[2].(*ast.BLangTypeDefinition)
		if _, ok := typeDef.GetTypeData().TypeDescriptor.(*ast.BLangUserDefinedType); !ok {
			t.Fatalf("valid type descriptor = %T", typeDef.GetTypeData().TypeDescriptor)
		}
	}
}

func TestRecoveringNodeBuilderInvalidModuleNames(t *testing.T) {
	for _, declaration := range []string{"int '_ = 1;", "const _ = 1;", "type _ int;", "@mod:_ int x = 1;"} {
		t.Run(declaration, func(t *testing.T) {
			unit, _ := buildNodeBuilderCompilationUnit(t, declaration+" function valid() {}", true)
			if _, ok := unit.TopLevelNodes[0].(*ast.BLangBadTopLevelNode); !ok {
				t.Fatalf("declaration = %T, want bad declaration", unit.TopLevelNodes[0])
			}
		})
	}
}
