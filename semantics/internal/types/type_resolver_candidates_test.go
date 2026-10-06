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

package types_test

import (
	"reflect"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util/testphases"
)

const noSurvivorSource = `
class First {
    function init(any literal, any langLib, any streamNext, any lax, any test, any cast, any list, any mapping, int marker) {
    }
}

class Second {
    function init(any literal, any langLib, any streamNext, any lax, any test, any cast, any list, any mapping, string marker) {
    }
}

public function main() returns error? {
    int[] arr = [1, 2];
    stream<int> s = arr.toStream();
    json j = {a: 1};
    any x = 1;
    First|Second _ = new (1.5, arr.length(), s.next(), check j.a, x is int, <int>x, [2.5, ...arr], {k: 3.5}, true);
}
`

func TestEphemeralTrialLeavesASTUnchanged(t *testing.T) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Fatalf("loading lang libraries failed: %v", err)
	}
	result, err := testphases.RunPipelineWithContent(env, cx, langlibs, testphases.PhaseTypeNarrowing, "no-survivor.bal", noSurvivorSource)
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	if !cx.HasDiagnostics() {
		t.Fatalf("expected the new expression to have no suitable object type")
	}
	finder := &newExpressionFinder{}
	ast.Walk(finder, result.Package)
	if finder.expr == nil {
		t.Fatalf("new expression not found")
	}
	if len(finder.expr.ArgsExprs) != 9 {
		t.Fatalf("expected the arguments to stay unlowered, got %d", len(finder.expr.ArgsExprs))
	}
	checker := &unresolvedChecker{t: t}
	for _, arg := range finder.expr.ArgsExprs {
		ast.Walk(checker, arg)
	}
}

type newExpressionFinder struct {
	expr *ast.BLangNewExpression
}

func (f *newExpressionFinder) Visit(node ast.BLangNode) ast.Visitor {
	if expr, ok := node.(*ast.BLangNewExpression); ok {
		f.expr = expr
		return nil
	}
	if node == nil || f.expr != nil {
		return nil
	}
	return f
}

func (f *newExpressionFinder) VisitTypeData(_ *ast.TypeData) ast.Visitor { return f }

type unresolvedChecker struct {
	t *testing.T
}

func (c *unresolvedChecker) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if !semtypes.IsZero(node.GetDeterminedType()) {
		c.t.Errorf("%T at %v has a determined type", node, node.GetPosition())
	}
	switch n := node.(type) {
	case *ast.BLangNumericLiteral:
		c.checkLiteralValue(&n.BLangLiteral)
	case *ast.BLangLiteral:
		c.checkLiteralValue(n)
	case *ast.BLangInvocation:
		if n.Expr == nil {
			c.t.Errorf("method call at %v was rewritten as a lang-lib call", n.GetPosition())
		}
		if _, deferred := n.RawSymbol.(*common.DeferredMethodSymbol); !deferred {
			c.t.Errorf("method call at %v has a resolved symbol", n.GetPosition())
		}
	case *ast.BLangFieldBaseAccess:
		if n.IsLax() {
			c.t.Errorf("field access at %v is marked lax", n.GetPosition())
		}
	case *ast.BLangTypeTestExpr:
		if !semtypes.IsZero(n.Type.Type) {
			c.t.Errorf("type test at %v has a resolved type", n.GetPosition())
		}
	case *ast.BLangListConstructorExpr:
		if !reflect.DeepEqual(n.AtomicType, semtypes.ListAtomicType{}) || len(n.SpreadMembers) != len(n.Exprs) {
			c.t.Errorf("list constructor at %v has an inherent type", n.GetPosition())
		}
	case *ast.BLangMappingConstructorExpr:
		if !reflect.DeepEqual(n.AtomicType, semtypes.MappingAtomicType{}) {
			c.t.Errorf("mapping constructor at %v has an inherent type", n.GetPosition())
		}
	}
	return c
}

func (c *unresolvedChecker) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	if !semtypes.IsZero(typeData.Type) {
		c.t.Errorf("type data %+v has a resolved type", typeData)
	}
	return c
}

func (c *unresolvedChecker) checkLiteralValue(lit *ast.BLangLiteral) {
	kind := lit.GetLiteralKind()
	isFloatingPoint := kind == ast.LiteralKindFloat || kind == ast.LiteralKindDecimal
	if isFloatingPoint && lit.Value != lit.OriginalValue {
		c.t.Errorf("literal %s at %v has value %v", lit.OriginalValue, lit.GetPosition(), lit.Value)
	}
}
