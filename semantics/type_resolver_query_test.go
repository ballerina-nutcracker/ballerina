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

package semantics

import (
	"ballerina/ast"
	"ballerina/context"
	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
	"strings"
	"testing"
)

type unsupportedExpr struct {
	ast.BLangLiteral
}

type unsupportedBType struct {
	ast.BLangValueType
}

var queryTestPos = diagnostics.NewBuiltinLocation()

func TestResolveQueryExprErrorCases(t *testing.T) {
	testCases := []struct {
		name    string
		query   *ast.BLangQueryExpr
		diagSub string
	}{
		{
			name: "missing select clause list",
			query: newQueryExpr(
				newFromClause(newIntListLiteral(1), nil, true),
			),
			diagSub: "query expression requires from and select clauses",
		},
		{
			name: "must start with from clause",
			query: newQueryExpr(
				newSelectClause(newIntLiteral(1)),
				newSelectClause(newIntLiteral(2)),
			),
			diagSub: "query expression must start with a from clause",
		},
		{
			name: "requires select clause",
			query: newQueryExpr(
				newFromClause(newIntListLiteral(1), nil, true),
				newWhereClause(newIntLiteral(1)),
			),
			diagSub: "query expression requires a select or collect clause",
		},
		{
			name: "from collection resolution fails",
			query: newQueryExpr(
				newFromClause(newUnsupportedExprNode(), nil, true),
				newSelectClause(newIntLiteral(1)),
			),
			diagSub: "unsupported expression type",
		},
		{
			name: "from collection is not a list",
			query: newQueryExpr(
				newFromClause(newIntLiteral(42), nil, true),
				newSelectClause(newIntLiteral(1)),
			),
			diagSub: "query from clause currently supports only list or map collections",
		},
		{
			name: "from binding variable is nil",
			query: newQueryExpr(
				newFromClause(newIntListLiteral(1), newEmptySimpleVarDef(), true),
				newSelectClause(newIntLiteral(1)),
			),
			diagSub: "only simple variable bindings are supported in from clause",
		},
		{
			name: "from binding type resolution fails",
			query: newQueryExpr(
				newFromClause(newIntListLiteral(1), newSimpleVarDef("x", newUnsupportedTypeNode(), nil), false),
				newSelectClause(newIntLiteral(1)),
			),
			diagSub: "unsupported type",
		},
		{
			name: "from binding type incompatible",
			query: newQueryExpr(
				newFromClause(newIntListLiteral(1), newSimpleVarDef("x", newValueType(ast.TypeKind_STRING), nil), false),
				newSelectClause(newIntLiteral(1)),
			),
			diagSub: "from clause variable type is incompatible with collection member type",
		},
		{
			name: "select expression resolution fails",
			query: newQueryExpr(
				newFromClause(newIntListLiteral(1), nil, true),
				newSelectClause(newUnsupportedExprNode()),
			),
			diagSub: "unsupported expression type",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolver, cx := newTestQueryResolver()
			_, _, ok := resolveQueryExpr(resolver, nil, testCase.query, semtypes.SemType{})
			if ok {
				t.Fatalf("expected resolveQueryExpr to fail")
			}
			assertDiagnosticContains(t, cx, testCase.diagSub)
		})
	}
}

func TestResolveQueryIntermediateClauseErrorCases(t *testing.T) {
	testCases := []struct {
		name    string
		clause  ast.BLangNode
		diagSub string
	}{
		{
			name: "let var declaration is nil",
			clause: newLetClause(
				newEmptySimpleVarDef(),
			),
			diagSub: "only simple variable declarations are supported in let clause",
		},
		{
			name: "let var declaration has no initializer",
			clause: newLetClause(
				newSimpleVarDef("y", nil, nil),
			),
			diagSub: "let clause variable declaration requires an initializer",
		},
		{
			name: "let initializer resolution fails",
			clause: newLetClause(
				newSimpleVarDef("y", nil, newUnsupportedExprNode()),
			),
			diagSub: "unsupported expression type",
		},
		{
			name: "let declared type resolution fails",
			clause: newLetClause(
				newSimpleVarDef("y", newUnsupportedTypeNode(), newIntLiteral(1)),
			),
			diagSub: "unsupported type",
		},
		{
			name: "let declared type incompatible",
			clause: newLetClause(
				newSimpleVarDef("y", newValueType(ast.TypeKind_STRING), newIntLiteral(1)),
			),
			diagSub: "let clause variable type is incompatible with initializer expression",
		},
		{
			name:    "where expression resolution fails",
			clause:  newWhereClause(newUnsupportedExprNode()),
			diagSub: "unsupported expression type",
		},
		{
			name:    "where expression non-boolean",
			clause:  newWhereClause(newIntLiteral(1)),
			diagSub: "where clause expression must be boolean",
		},
		{
			name:    "limit expression resolution fails",
			clause:  newLimitClause(newUnsupportedExprNode()),
			diagSub: "unsupported expression type",
		},
		{
			name:    "limit expression non-int",
			clause:  newLimitClause(newStringLiteral("x")),
			diagSub: "limit clause expression must be int",
		},
		{
			name:    "limit expression resolution fails",
			clause:  newLimitClause(newUnsupportedExprNode()),
			diagSub: "unsupported expression type",
		},
		{
			name:    "limit expression non-int",
			clause:  newLimitClause(newStringLiteral("x")),
			diagSub: "limit clause expression must be int",
		},
		{
			name: "order by key resolution fails",
			clause: newOrderByClause(
				newOrderKey(newUnsupportedExprNode(), true),
			),
			diagSub: "unsupported expression type",
		},
		{
			name: "order by key non-ordered type",
			clause: newOrderByClause(
				newOrderKey(newEmptyMappingLiteral(), true),
			),
			diagSub: "order by key expression must have an ordered type",
		},
		{
			name: "group by variable declaration has no initializer",
			clause: newGroupByClause(
				newGroupingKeyVarDef(newSimpleVarDef("g", nil, nil)),
			),
			diagSub: "group by variable declaration requires an initializer",
		},
		{
			name: "group by initializer resolution fails",
			clause: newGroupByClause(
				newGroupingKeyVarDef(newSimpleVarDef("g", nil, newUnsupportedExprNode())),
			),
			diagSub: "unsupported expression type",
		},
		{
			name: "group by declared type resolution fails",
			clause: newGroupByClause(
				newGroupingKeyVarDef(newSimpleVarDef("g", newUnsupportedTypeNode(), newIntLiteral(1))),
			),
			diagSub: "unsupported type",
		},
		{
			name: "group by declared type incompatible",
			clause: newGroupByClause(
				newGroupingKeyVarDef(newSimpleVarDef("g", newValueType(ast.TypeKind_STRING), newIntLiteral(1))),
			),
			diagSub: "group by variable type is incompatible with initializer expression",
		},
		{
			name: "group by key non-anydata type",
			clause: newGroupByClause(
				newGroupingKeyVarDef(newSimpleVarDef("g", nil, newErrorConstructorExpr())),
			),
			diagSub: "grouping key expression must be a subtype of anydata",
		},
		{
			name: "join collection non-list",
			clause: newJoinClause(
				newIntLiteral(1),
				newSimpleVarDef("j", nil, nil),
				true,
				false,
				newOnClause(newIntLiteral(1), newIntLiteral(1)),
			),
			diagSub: "query from clause currently supports only list or map collections",
		},
		{
			name: "outer join without var",
			clause: newJoinClause(
				newIntListLiteral(1),
				newSimpleVarDef("j", newValueType(ast.TypeKind_INT), nil),
				false,
				true,
				newOnClause(newIntLiteral(1), newIntLiteral(1)),
			),
			diagSub: "outer join clause variable must be declared with var",
		},
		{
			name: "join without on clause",
			clause: newJoinClause(
				newIntListLiteral(1),
				newSimpleVarDef("j", nil, nil),
				true,
				false,
				nil,
			),
			diagSub: "join clause requires an on clause",
		},
		{
			name: "join incompatible on clause types",
			clause: newJoinClause(
				newIntListLiteral(1),
				newSimpleVarDef("j", nil, nil),
				true,
				false,
				newOnClause(newStringLiteral("x"), newIntLiteral(1)),
			),
			diagSub: "incompatible type",
		},
		{
			name:    "unsupported intermediate clause",
			clause:  newCollectClause(),
			diagSub: "only join + let + where + group by + order by + limit clauses are supported as intermediate query clauses",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			query := newQueryExpr(
				newFromClause(newIntListLiteral(1), nil, true),
				testCase.clause,
				newSelectClause(newIntLiteral(1)),
			)
			resolver, cx := newTestQueryResolver()
			_, ok := resolveQueryIntermediateClauses(resolver, nil, query, len(query.QueryClauseList)-1)
			if ok {
				t.Fatalf("expected resolveQueryIntermediateClauses to fail")
			}
			assertDiagnosticContains(t, cx, testCase.diagSub)
		})
	}
}

func TestResolveQueryExprMapCollection(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	space := cx.NewSymbolSpace(*cx.GetDefaultPackage())
	mapSymbol := model.NewVariableSymbol("m", false, false, false)
	space.AddSymbol("m", &mapSymbol)
	mapSymbolRef, _ := space.GetSymbol("m")
	cx.SetSymbolType(mapSymbolRef, semtypes.MAPPING)

	mapRef := &ast.BLangSimpleVarRef{
		VariableName: &ast.BLangIdentifier{
			Value: "m",
		},
	}
	mapRef.SetPosition(queryTestPos)
	mapRef.SetSymbol(mapSymbolRef)

	query := newQueryExpr(
		newFromClause(mapRef, nil, true),
		newSelectClause(newIntLiteral(1)),
	)
	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for map collection")
	}
	if !semtypes.IsSubtype(semtypes.ContextFrom(cx.GetTypeEnv()), queryTy, semtypes.LIST) {
		t.Fatalf("expected query result type to be a list, got %v", queryTy)
	}
}

func TestResolveQueryExprMapConstructType(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1), nil, true),
		newSelectClause(newListLiteral(newStringLiteral("k"), newIntLiteral(1))),
	)
	query.QueryConstructType = ast.TypeKind_MAP

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for map construct type")
	}
	if !semtypes.IsSubtype(semtypes.ContextFrom(cx.GetTypeEnv()), queryTy, semtypes.MAPPING) {
		t.Fatalf("expected query result type to be mapping, got %v", queryTy)
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprCollectClause(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1, 2, 3), nil, true),
		newCollectClauseExpr(newIntLiteral(1)),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for collect clause")
	}
	if !semtypes.IsSubtypeSimple(queryTy, semtypes.INT) {
		t.Fatalf("expected query result type to be int, got %v", queryTy)
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprCollectClauseRejectsConstructType(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1, 2, 3), nil, true),
		newCollectClauseExpr(newIntLiteral(1)),
	)
	query.QueryConstructType = ast.TypeKind_MAP

	_, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if ok {
		t.Fatalf("expected resolveQueryExpr to fail for collect clause with query construct type")
	}
	assertDiagnosticContains(t, cx, "query construct types cannot be used with collect clause")
}

func TestResolveQueryExprCollectAggregatesVariables(t *testing.T) {
	resolver, cx := newTestQueryResolver()
	space := cx.NewSymbolSpace(*cx.GetDefaultPackage())
	xSymbolRef := addTestValueSymbol(cx, space, "x", semtypes.SemType{})
	xDef := newSimpleVarDef("x", nil, nil)
	xDef.Var.SetSymbol(xSymbolRef)
	collectXRef := newSimpleVarRef("x", xSymbolRef)

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1, 2, 3), xDef, true),
		newCollectClauseExpr(collectXRef),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for collect clause with query variable")
	}
	if !semtypes.IsSubtypeSimple(queryTy, semtypes.LIST) {
		t.Fatalf("expected collect result type to be a list, got %v", queryTy)
	}
	if !semtypes.IsSubtypeSimple(collectXRef.GetDeterminedType(), semtypes.LIST) {
		t.Fatalf("expected collect variable reference to be aggregated as a list, got %v", collectXRef.GetDeterminedType())
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprGroupByClauseAggregatesNonGroupingVars(t *testing.T) {
	resolver, cx := newTestQueryResolver()
	space := cx.NewSymbolSpace(*cx.GetDefaultPackage())
	xSymbolRef := addTestValueSymbol(cx, space, "x", semtypes.SemType{})
	ySymbolRef := addTestValueSymbol(cx, space, "y", semtypes.SemType{})

	xDef := newSimpleVarDef("x", nil, nil)
	xDef.Var.SetSymbol(xSymbolRef)
	yDef := newSimpleVarDef("y", nil, newIntLiteral(1))
	yDef.Var.SetSymbol(ySymbolRef)
	groupXRef := newSimpleVarRef("x", xSymbolRef)
	selectYRef := newSimpleVarRef("y", ySymbolRef)

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1, 2, 3), xDef, true),
		newLetClause(yDef),
		newGroupByClause(newGroupingKeyRef(groupXRef)),
		newSelectClause(selectYRef),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for group by clause")
	}
	if !semtypes.IsSubtypeSimple(queryTy, semtypes.LIST) {
		t.Fatalf("expected query result type to be a list, got %v", queryTy)
	}
	if !semtypes.IsSubtypeSimple(groupXRef.GetDeterminedType(), semtypes.INT) {
		t.Fatalf("expected grouping key variable to remain int, got %v", groupXRef.GetDeterminedType())
	}
	if !semtypes.IsSubtypeSimple(selectYRef.GetDeterminedType(), semtypes.LIST) {
		t.Fatalf("expected non-grouping variable to be aggregated as a non-empty list, got %v", selectYRef.GetDeterminedType())
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprCollectDoesNotReaggregateGroupByVars(t *testing.T) {
	resolver, cx := newTestQueryResolver()
	space := cx.NewSymbolSpace(*cx.GetDefaultPackage())
	xSymbolRef := addTestValueSymbol(cx, space, "x", semtypes.SemType{})
	ySymbolRef := addTestValueSymbol(cx, space, "y", semtypes.SemType{})

	xDef := newSimpleVarDef("x", nil, nil)
	xDef.Var.SetSymbol(xSymbolRef)
	yDef := newSimpleVarDef("y", nil, newIntLiteral(1))
	yDef.Var.SetSymbol(ySymbolRef)
	groupXRef := newSimpleVarRef("x", xSymbolRef)
	collectYRef := newSimpleVarRef("y", ySymbolRef)

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1, 2, 3), xDef, true),
		newLetClause(yDef),
		newGroupByClause(newGroupingKeyRef(groupXRef)),
		newCollectClauseExpr(collectYRef),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for group by + collect")
	}
	if !semtypes.IsSubtypeSimple(queryTy, semtypes.LIST) {
		t.Fatalf("expected collect result type to be a list, got %v", queryTy)
	}
	tyCtx := semtypes.ContextFrom(cx.GetTypeEnv())
	collectYTy := collectYRef.GetDeterminedType()
	if !semtypes.IsSubtypeSimple(collectYTy, semtypes.LIST) {
		t.Fatalf("expected grouped non-key variable to remain a list, got %v", collectYTy)
	}
	memberTy := semtypes.ListMemberTypeInnerVal(tyCtx, collectYTy, semtypes.IntConst(0))
	if !semtypes.IsSubtypeSimple(memberTy, semtypes.INT) {
		t.Fatalf("expected grouped non-key variable member type to be int, got %v", memberTy)
	}
	if semtypes.IsSubtypeSimple(memberTy, semtypes.LIST) {
		t.Fatalf("expected grouped non-key variable not to be re-aggregated as a list of lists, got %v", collectYTy)
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprGroupByVarDeclaration(t *testing.T) {
	resolver, cx := newTestQueryResolver()
	space := cx.NewSymbolSpace(*cx.GetDefaultPackage())
	xSymbolRef := addTestValueSymbol(cx, space, "x", semtypes.SemType{})
	nSymbolRef := addTestValueSymbol(cx, space, "n", semtypes.SemType{})

	xDef := newSimpleVarDef("x", nil, nil)
	xDef.Var.SetSymbol(xSymbolRef)
	nDef := newSimpleVarDef("n", nil, newIntLiteral(1))
	nDef.Var.SetSymbol(nSymbolRef)
	selectNRef := newSimpleVarRef("n", nSymbolRef)

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1, 2, 3), xDef, true),
		newGroupByClause(newGroupingKeyVarDef(nDef)),
		newSelectClause(selectNRef),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for group by variable declaration")
	}
	if !semtypes.IsSubtypeSimple(queryTy, semtypes.LIST) {
		t.Fatalf("expected query result type to be a list, got %v", queryTy)
	}
	if !semtypes.IsSubtypeSimple(selectNRef.GetDeterminedType(), semtypes.INT) {
		t.Fatalf("expected grouping variable declaration to be int, got %v", selectNRef.GetDeterminedType())
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprMapConstructTypeInvalidSelect(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1), nil, true),
		newSelectClause(newIntLiteral(1)),
	)
	query.QueryConstructType = ast.TypeKind_MAP

	_, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if ok {
		t.Fatalf("expected resolveQueryExpr to fail for invalid map select expression")
	}
	assertDiagnosticContains(t, cx, "incompatible type")
}

func TestResolveQueryExprOrderByClause(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(3, 1, 2), nil, true),
		newOrderByClause(newOrderKey(newIntLiteral(1), true)),
		newSelectClause(newIntLiteral(1)),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for order by clause")
	}
	if !semtypes.IsSubtype(semtypes.ContextFrom(cx.GetTypeEnv()), queryTy, semtypes.LIST) {
		t.Fatalf("expected query result type to be a list, got %v", queryTy)
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprJoinClause(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(1, 2), nil, true),
		newJoinClause(
			newIntListLiteral(2, 3),
			newSimpleVarDef("j", nil, nil),
			true,
			false,
			newOnClause(newIntLiteral(1), newIntLiteral(1)),
		),
		newSelectClause(newIntLiteral(1)),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for join clause")
	}
	if !semtypes.IsSubtype(semtypes.ContextFrom(cx.GetTypeEnv()), queryTy, semtypes.LIST) {
		t.Fatalf("expected query result type to be a list, got %v", queryTy)
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprMultipleOrderByConsecutive(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(3, 1, 2), nil, true),
		newOrderByClause(newOrderKey(newIntLiteral(1), true)),
		newOrderByClause(newOrderKey(newIntLiteral(2), false)),
		newSelectClause(newIntLiteral(1)),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for consecutive order by clauses")
	}
	if !semtypes.IsSubtype(semtypes.ContextFrom(cx.GetTypeEnv()), queryTy, semtypes.LIST) {
		t.Fatalf("expected query result type to be a list, got %v", queryTy)
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprMultipleOrderByNonConsecutive(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	query := newQueryExpr(
		newFromClause(newIntListLiteral(3, 1, 2), nil, true),
		newOrderByClause(newOrderKey(newIntLiteral(1), true)),
		newWhereClause(newBoolLiteral(true)),
		newOrderByClause(newOrderKey(newIntLiteral(2), false)),
		newSelectClause(newIntLiteral(1)),
	)

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for non-consecutive order by clauses")
	}
	if !semtypes.IsSubtype(semtypes.ContextFrom(cx.GetTypeEnv()), queryTy, semtypes.LIST) {
		t.Fatalf("expected query result type to be a list, got %v", queryTy)
	}
	if len(cx.Diagnostics()) > 0 {
		t.Fatalf("expected no diagnostics, got %v", cx.Diagnostics())
	}
}

func TestResolveQueryExprOrderByRejectsMixedSimpleUnion(t *testing.T) {
	resolver, cx := newTestQueryResolver()

	space := cx.NewSymbolSpace(*cx.GetDefaultPackage())
	keySymbol := model.NewVariableSymbol("k", false, false, false)
	space.AddSymbol("k", &keySymbol)
	keySymbolRef, _ := space.GetSymbol("k")
	cx.SetSymbolType(keySymbolRef, semtypes.Union(semtypes.INT, semtypes.STRING))

	keyRef := &ast.BLangSimpleVarRef{
		VariableName: &ast.BLangIdentifier{
			Value: "k",
		},
	}
	keyRef.SetPosition(queryTestPos)
	keyRef.SetSymbol(keySymbolRef)

	query := newQueryExpr(
		newFromClause(newIntListLiteral(3, 1, 2), nil, true),
		newOrderByClause(newOrderKey(keyRef, true)),
		newSelectClause(newIntLiteral(1)),
	)

	_, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if ok {
		t.Fatalf("expected resolveQueryExpr to fail for non-ordered mixed simple union")
	}
	assertDiagnosticContains(t, cx, "order by key expression must have an ordered type")
}

func TestResolveQueryExprOnConflictClauseErrors(t *testing.T) {
	t.Run("on conflict with non-map construct type", func(t *testing.T) {
		resolver, cx := newTestQueryResolver()
		query := newQueryExpr(
			newFromClause(newIntListLiteral(1), nil, true),
			newSelectClause(newIntLiteral(1)),
			newOnConflictClause(newErrorConstructorExpr()),
		)
		_, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
		if ok {
			t.Fatalf("expected resolveQueryExpr to fail for non-map on conflict")
		}
		assertDiagnosticContains(t, cx, "on conflict clause is supported only for map query construct type")
	})

	t.Run("on conflict expression must be error?", func(t *testing.T) {
		resolver, cx := newTestQueryResolver()
		query := newQueryExpr(
			newFromClause(newIntListLiteral(1), nil, true),
			newSelectClause(newListLiteral(newStringLiteral("k"), newIntLiteral(1))),
			newOnConflictClause(newIntLiteral(1)),
		)
		query.QueryConstructType = ast.TypeKind_MAP
		_, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
		if ok {
			t.Fatalf("expected resolveQueryExpr to fail for invalid on conflict expression")
		}
		assertDiagnosticContains(t, cx, "on conflict clause expression must be error?")
	})
}

func TestResolveQueryExprMapConstructTypeOnConflictNil(t *testing.T) {
	resolver, cx := newTestQueryResolver()
	query := newQueryExpr(
		newFromClause(newIntListLiteral(1), nil, true),
		newSelectClause(newListLiteral(newStringLiteral("k"), newIntLiteral(1))),
		newOnConflictClause(newNilLiteral()),
	)
	query.QueryConstructType = ast.TypeKind_MAP

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for map on conflict nil")
	}
	tyCtx := semtypes.ContextFrom(cx.GetTypeEnv())
	if !semtypes.IsSubtype(tyCtx, queryTy, semtypes.MAPPING) {
		t.Fatalf("expected query result type to be mapping, got %v", queryTy)
	}
	if semtypes.IsSubtype(tyCtx, semtypes.ERROR, queryTy) {
		t.Fatalf("expected query result type not to include error, got %v", queryTy)
	}
}

func TestResolveQueryExprMapConstructTypeOnConflictError(t *testing.T) {
	resolver, cx := newTestQueryResolver()
	query := newQueryExpr(
		newFromClause(newIntListLiteral(1), nil, true),
		newSelectClause(newListLiteral(newStringLiteral("k"), newIntLiteral(1))),
		newOnConflictClause(newErrorConstructorExpr()),
	)
	query.QueryConstructType = ast.TypeKind_MAP

	queryTy, _, ok := resolveQueryExpr(resolver, nil, query, semtypes.SemType{})
	if !ok {
		t.Fatalf("expected resolveQueryExpr to succeed for map on conflict error")
	}
	tyCtx := semtypes.ContextFrom(cx.GetTypeEnv())
	if !semtypes.IsSubtype(tyCtx, semtypes.ERROR, queryTy) {
		t.Fatalf("expected query result type to include error, got %v", queryTy)
	}
}

func newTestQueryResolver() (*packageTypeResolver, *context.CompilerContext) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	return newPackageTypeResolver(cx, &ast.BLangPackage{}, nil, nil), cx
}

func assertDiagnosticContains(t *testing.T, cx *context.CompilerContext, substr string) {
	t.Helper()
	diagnosticsList := cx.Diagnostics()
	if len(diagnosticsList) == 0 {
		t.Fatalf("expected at least one diagnostic containing %q, but diagnostics are empty", substr)
	}
	for _, diag := range diagnosticsList {
		if strings.Contains(diag.Message(), substr) {
			return
		}
	}

	messages := make([]string, len(diagnosticsList))
	for i, diag := range diagnosticsList {
		messages[i] = diag.Message()
	}
	t.Fatalf("expected diagnostic containing %q, got: %v", substr, messages)
}

func newQueryExpr(clauses ...ast.BLangNode) *ast.BLangQueryExpr {
	query := &ast.BLangQueryExpr{
		QueryClauseList: clauses,
	}
	query.SetPosition(queryTestPos)
	return query
}

func newFromClause(collection ast.BLangExpression, varDef *ast.BLangSimpleVariableDef, declaredWithVar bool) *ast.BLangFromClause {
	fromClause := &ast.BLangFromClause{
		BLangInputClause: ast.BLangInputClause{
			VariableDefinitionNode: varDef,
			IsDeclaredWithVarFlag:  declaredWithVar,
		},
	}
	fromClause.SetPosition(queryTestPos)
	fromClause.SetCollection(collection)
	return fromClause
}

func newJoinClause(
	collection ast.BLangExpression,
	varDef *ast.BLangSimpleVariableDef,
	declaredWithVar bool,
	isOuterJoin bool,
	onClause *ast.BLangOnClause,
) *ast.BLangJoinClause {
	joinClause := &ast.BLangJoinClause{
		BLangInputClause: ast.BLangInputClause{
			VariableDefinitionNode: varDef,
			IsDeclaredWithVarFlag:  declaredWithVar,
		},
		IsOuterJoinFlag: isOuterJoin,
	}
	if onClause != nil {
		joinClause.OnClause = *onClause
	}
	joinClause.SetPosition(queryTestPos)
	joinClause.SetCollection(collection)
	return joinClause
}

func newSelectClause(expr ast.BLangExpression) *ast.BLangSelectClause {
	selectClause := &ast.BLangSelectClause{}
	selectClause.SetPosition(queryTestPos)
	selectClause.SetExpression(expr)
	return selectClause
}

func newLetClause(defs ...*ast.BLangSimpleVariableDef) *ast.BLangLetClause {
	decls := make([]ast.BLangSimpleVariableDef, len(defs))
	for i, def := range defs {
		decls[i] = *def
	}
	letClause := &ast.BLangLetClause{
		LetVarDeclarations: decls,
	}
	letClause.SetPosition(queryTestPos)
	return letClause
}

func newWhereClause(expr ast.BLangExpression) *ast.BLangWhereClause {
	whereClause := &ast.BLangWhereClause{
		Expression: expr,
	}
	whereClause.SetPosition(queryTestPos)
	return whereClause
}

func newOnClause(lhs ast.BLangExpression, rhs ast.BLangExpression) *ast.BLangOnClause {
	onClause := &ast.BLangOnClause{
		OnExpr:     lhs,
		EqualsExpr: rhs,
	}
	onClause.SetPosition(queryTestPos)
	return onClause
}

func newLimitClause(expr ast.BLangExpression) *ast.BLangLimitClause {
	limitClause := &ast.BLangLimitClause{
		Expression: expr,
	}
	limitClause.SetPosition(queryTestPos)
	return limitClause
}

func newCollectClause() *ast.BLangCollectClause {
	return newCollectClauseExpr(newIntLiteral(1))
}

func newCollectClauseExpr(expr ast.BLangExpression) *ast.BLangCollectClause {
	collectClause := &ast.BLangCollectClause{}
	collectClause.SetPosition(queryTestPos)
	collectClause.SetExpression(expr)
	return collectClause
}

func newGroupByClause(keys ...ast.BLangGroupingKey) *ast.BLangGroupByClause {
	groupByClause := &ast.BLangGroupByClause{
		GroupingKeyList: keys,
	}
	groupByClause.SetPosition(queryTestPos)
	return groupByClause
}

func newGroupingKeyRef(ref *ast.BLangSimpleVarRef) ast.BLangGroupingKey {
	key := ast.BLangGroupingKey{
		VariableRef: ref,
	}
	key.SetPosition(queryTestPos)
	return key
}

func newGroupingKeyVarDef(varDef *ast.BLangSimpleVariableDef) ast.BLangGroupingKey {
	key := ast.BLangGroupingKey{
		VariableDef: varDef,
	}
	key.SetPosition(queryTestPos)
	return key
}

func newOrderByClause(keys ...ast.BLangOrderKey) *ast.BLangOrderByClause {
	orderByClause := &ast.BLangOrderByClause{
		OrderByKeyList: keys,
	}
	orderByClause.SetPosition(queryTestPos)
	return orderByClause
}

func newOrderKey(expr ast.BLangExpression, isAscending bool) ast.BLangOrderKey {
	orderKey := ast.BLangOrderKey{
		Expression:   expr,
		IsDescending: !isAscending,
	}
	orderKey.SetPosition(queryTestPos)
	return orderKey
}

func newOnConflictClause(expr ast.BLangExpression) *ast.BLangOnConflictClause {
	onConflictClause := &ast.BLangOnConflictClause{
		Expression: expr,
	}
	onConflictClause.SetPosition(queryTestPos)
	return onConflictClause
}

func newEmptySimpleVarDef() *ast.BLangSimpleVariableDef {
	varDef := &ast.BLangSimpleVariableDef{}
	varDef.SetPosition(queryTestPos)
	return varDef
}

func newSimpleVarDef(name string, typeNode ast.BType, expr ast.BLangExpression) *ast.BLangSimpleVariableDef {
	ident := &ast.BLangIdentifier{
		Value:         name,
		OriginalValue: name,
	}
	ident.SetPosition(queryTestPos)

	variable := &ast.BLangSimpleVariable{
		Name: ident,
	}
	variable.SetPosition(queryTestPos)
	variable.SetTypeNode(typeNode)
	if expr != nil {
		variable.SetExpr(expr)
	}

	varDef := &ast.BLangSimpleVariableDef{
		Var: variable,
	}
	varDef.SetPosition(queryTestPos)
	return varDef
}

func addTestValueSymbol(cx *context.CompilerContext, space *model.SymbolSpace, name string, ty semtypes.SemType) model.SymbolRef {
	valueSymbol := model.NewVariableSymbol(name, false, false, false)
	space.AddSymbol(name, &valueSymbol)
	symbolRef, _ := space.GetSymbol(name)
	if !semtypes.IsZero(ty) {
		cx.SetSymbolType(symbolRef, ty)
	}
	return symbolRef
}

func newSimpleVarRef(name string, symbol model.SymbolRef) *ast.BLangSimpleVarRef {
	ident := &ast.BLangIdentifier{
		Value:         name,
		OriginalValue: name,
	}
	ident.SetPosition(queryTestPos)
	ref := &ast.BLangSimpleVarRef{
		VariableName: ident,
	}
	ref.SetPosition(queryTestPos)
	ref.SetSymbol(symbol)
	return ref
}

func newValueType(typeKind ast.TypeKind) ast.BType {
	ty := &ast.BLangValueType{
		TypeKind: typeKind,
	}
	ty.SetPosition(queryTestPos)
	return ty
}

func newUnsupportedTypeNode() ast.BType {
	ty := &unsupportedBType{}
	ty.SetPosition(queryTestPos)
	return ty
}

func newIntLiteral(value int64) *ast.BLangLiteral {
	literal := &ast.BLangLiteral{}
	literal.SetPosition(queryTestPos)
	literal.SetValue(value)
	literal.SetValueType(ast.NewBType(ast.TypeTags_INT, "", 0))
	return literal
}

func newIntListLiteral(values ...int64) *ast.BLangListConstructorExpr {
	exprs := make([]ast.BLangExpression, 0, len(values))
	for _, value := range values {
		exprs = append(exprs, newIntLiteral(value))
	}
	listExpr := &ast.BLangListConstructorExpr{
		Exprs: exprs,
	}
	listExpr.SetPosition(queryTestPos)
	return listExpr
}

func newStringLiteral(value string) *ast.BLangLiteral {
	literal := &ast.BLangLiteral{}
	literal.SetPosition(queryTestPos)
	literal.SetValue(value)
	literal.SetValueType(ast.NewBType(ast.TypeTags_STRING, "", 0))
	return literal
}

func newBoolLiteral(value bool) *ast.BLangLiteral {
	literal := &ast.BLangLiteral{}
	literal.SetPosition(queryTestPos)
	literal.SetValue(value)
	literal.SetValueType(ast.NewBType(ast.TypeTags_BOOLEAN, "", 0))
	return literal
}

func newNilLiteral() *ast.BLangLiteral {
	literal := &ast.BLangLiteral{}
	literal.SetPosition(queryTestPos)
	literal.SetValue(nil)
	literal.SetValueType(ast.NewBType(ast.TypeTags_NIL, "", 0))
	return literal
}

func newListLiteral(values ...ast.BLangExpression) *ast.BLangListConstructorExpr {
	listExpr := &ast.BLangListConstructorExpr{
		Exprs: values,
	}
	listExpr.SetPosition(queryTestPos)
	return listExpr
}

func newEmptyMappingLiteral() *ast.BLangMappingConstructorExpr {
	mapExpr := &ast.BLangMappingConstructorExpr{
		Fields: []ast.MappingField{},
	}
	mapExpr.SetPosition(queryTestPos)
	return mapExpr
}

func newErrorConstructorExpr() *ast.BLangErrorConstructorExpr {
	errExpr := &ast.BLangErrorConstructorExpr{}
	errExpr.SetPosition(queryTestPos)
	return errExpr
}

func newUnsupportedExprNode() ast.BLangExpression {
	expr := &unsupportedExpr{}
	expr.SetPosition(queryTestPos)
	return expr
}
