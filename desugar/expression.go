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

// Package desugar represents AST-> AST transforms
package desugar

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type invocable interface {
	ast.Invocable
	SetReceiver(ast.BLangExpression)
	SetCallArgs([]ast.BLangExpression)
}

func walkExpression(cx *functionContext, node ast.BLangActionOrExpression) desugaredNode[ast.BLangActionOrExpression] {
	switch expr := node.(type) {
	case *ast.BLangBinaryExpr:
		return walkBinaryExpr(cx, expr)
	case *ast.BLangTernaryExpr:
		return walkTernaryExpr(cx, expr)
	case *ast.BLangUnaryExpr:
		return walkUnaryExpr(cx, expr)
	case *ast.BLangNilConditionalExpr:
		return walkNilConditionalExpr(cx, expr)
	case *ast.BLangGroupExpr:
		return walkGroupExpr(cx, expr)
	case *ast.BLangIndexBasedAccess:
		return walkIndexBasedAccess(cx, expr)
	case *ast.BLangFieldBaseAccess:
		return walkFieldBaseAccess(cx, expr)
	case *ast.BLangInvocation:
		return walkInvocation(cx, expr)
	case *ast.BLangListConstructorExpr:
		return walkListConstructorExpr(cx, expr)
	case *ast.BLangMappingConstructorExpr:
		return walkMappingConstructorExpr(cx, expr)
	case *ast.BLangErrorConstructorExpr:
		return walkErrorConstructorExpr(cx, expr)
	case *ast.BLangCheckedExpr:
		return walkCheckedExpr(cx, expr)
	case *ast.BLangCheckPanickedExpr:
		return walkCheckPanickedExpr(cx, expr)
	case *ast.BLangTrapExpr:
		return walkTrapExpr(cx, expr)
	case *ast.BLangLambdaFunction:
		return walkLambdaFunction(cx, expr)
	case *ast.BLangTypeConversionExpr:
		return walkTypeConversionExpr(cx, expr)
	case *ast.BLangTypeTestExpr:
		return walkTypeTestExpr(cx, expr)
	case *ast.BLangAnnotAccessExpr:
		return walkAnnotAccessExpr(cx, expr)
	case *ast.BLangArrowFunction:
		return walkArrowFunction(cx, expr)
	case *ast.BLangQueryExpr:
		return walkQueryExpr(cx, expr)
	case *ast.BLangTypedescExpr:
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangLiteral:
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangNumericLiteral:
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangVarRef:
		if replacement := materializeConstantRef(cx, expr); replacement != nil {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: replacement}
		}
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangConstRef:
		if replacement := materializeConstantRef(cx, &expr.BLangVarRef); replacement != nil {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: replacement}
		}
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangNewExpression:
		return walkNewExpression(cx, expr)
	case *BLangServiceInit:
		// Desugar-introduced node; nothing to rewrite further.
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangNamedArgsExpression:
		result := walkExpression(cx, expr.Expr)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       result.initStmts,
			replacementNode: expr,
		}
	case *ast.BLangRemoteMethodCallAction:
		return walkInvocation(cx, expr)
	case *ast.BLangClientResourceAccessAction:
		return walkClientResourceAccessAction(cx, expr)
	case *ast.BLangStartAction:
		return walkStartAction(cx, expr)
	case *ast.BLangSingleWaitAction:
		result := walkExpression(cx, expr.FutureExpr)
		expr.FutureExpr = result.replacementNode.(ast.BLangExpression)
		return desugaredNode[ast.BLangActionOrExpression]{initStmts: result.initStmts, replacementNode: expr}
	case *ast.BLangAlternateWaitAction:
		var initStmts []ast.StatementNode
		for i, futureExpr := range expr.FutureExprs {
			result := walkExpression(cx, futureExpr)
			initStmts = append(initStmts, result.initStmts...)
			expr.FutureExprs[i] = result.replacementNode.(ast.BLangExpression)
		}
		return desugaredNode[ast.BLangActionOrExpression]{initStmts: initStmts, replacementNode: expr}
	case *ast.BLangMultipleWaitAction:
		var initStmts []ast.StatementNode
		for i, futureExpr := range expr.FutureExprs {
			result := walkExpression(cx, futureExpr)
			initStmts = append(initStmts, result.initStmts...)
			expr.FutureExprs[i] = result.replacementNode.(ast.BLangExpression)
		}
		return desugaredNode[ast.BLangActionOrExpression]{initStmts: initStmts, replacementNode: expr}
	case *ast.BLangWildCardBindingPattern:
		// Wildcard binding pattern can appear in variable references (e.g., _ = expr)
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangXMLSequenceLiteral:
		var initStmts []ast.StatementNode
		for i, child := range expr.Children {
			r := walkExpression(cx, child)
			initStmts = append(initStmts, r.initStmts...)
			expr.Children[i] = r.replacementNode.(ast.BLangExpression)
		}
		return desugaredNode[ast.BLangActionOrExpression]{initStmts: initStmts, replacementNode: expr}
	case *ast.BLangXMLElementLiteral:
		var initStmts []ast.StatementNode
		for i := range expr.Attrs {
			if expr.Attrs[i].Value != nil {
				r := walkExpression(cx, expr.Attrs[i].Value)
				initStmts = append(initStmts, r.initStmts...)
				expr.Attrs[i].Value = r.replacementNode.(ast.BLangExpression)
			}
		}
		if expr.Content != nil {
			r := walkExpression(cx, expr.Content)
			initStmts = append(initStmts, r.initStmts...)
			expr.Content = r.replacementNode.(ast.BLangExpression)
		}
		return desugaredNode[ast.BLangActionOrExpression]{initStmts: initStmts, replacementNode: expr}
	case *ast.BLangXMLPILiteral, *ast.BLangXMLCommentLiteral, *ast.BLangXMLTextLiteral:
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangTemplateExpr:
		return walkTemplateExpr(cx, expr)
	case *ast.BLangXMLTemplateExpr:
		return walkXMLTemplateExpr(cx, expr)
	default:
		cx.internalError(fmt.Sprintf("unexpected expression type: %T", node), node.GetPosition())
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: node}
	}
}

func walkTernaryExpr(cx *functionContext, expr *ast.BLangTernaryExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	condition := walkExpression(cx, expr.Condition)
	initStmts = append(initStmts, condition.initStmts...)
	expr.Condition = condition.replacementNode.(ast.BLangExpression)

	thenExpr := walkExpression(cx, expr.ThenExpr)
	initStmts = append(initStmts, thenExpr.initStmts...)
	expr.ThenExpr = thenExpr.replacementNode.(ast.BLangExpression)

	elseExpr := walkExpression(cx, expr.ElseExpr)
	initStmts = append(initStmts, elseExpr.initStmts...)
	expr.ElseExpr = elseExpr.replacementNode.(ast.BLangExpression)

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkBinaryExpr(cx *functionContext, expr *ast.BLangBinaryExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.LhsExpr != nil {
		result := walkExpression(cx, expr.LhsExpr)
		initStmts = append(initStmts, result.initStmts...)
		expr.LhsExpr = result.replacementNode.(ast.BLangExpression)
	}

	if expr.RhsExpr != nil {
		result := walkExpression(cx, expr.RhsExpr)
		initStmts = append(initStmts, result.initStmts...)
		expr.RhsExpr = result.replacementNode.(ast.BLangExpression)
	}

	if !isNilLiftableBinaryOp(expr.OpKind) {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}

	lhsTy := expr.LhsExpr.GetDeterminedType()
	rhsTy := expr.RhsExpr.GetDeterminedType()
	if semtypes.IsZero(lhsTy) || semtypes.IsZero(rhsTy) {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}
	lhsHasNil := semtypes.ContainsBasicType(lhsTy, semtypes.Nil)
	rhsHasNil := semtypes.ContainsBasicType(rhsTy, semtypes.Nil)

	if !lhsHasNil && !rhsHasNil {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}

	basePos := expr.GetPosition()
	resultTy := expr.GetDeterminedType()

	// Create temp vars for nullable operands
	var lhsVarName *ast.BLangIdentifier
	var lhsSymbol model.SymbolRef
	if lhsHasNil {
		lhsVarName, lhsSymbol, initStmts = createOperandTempVar(cx, lhsTy, expr.LhsExpr, basePos, initStmts)
	}

	var rhsVarName *ast.BLangIdentifier
	var rhsSymbol model.SymbolRef
	if rhsHasNil {
		rhsVarName, rhsSymbol, initStmts = createOperandTempVar(cx, rhsTy, expr.RhsExpr, basePos, initStmts)
	}

	// Create result temp var initialized to nil
	resultVarName, resultSymbol, initStmts := createNilResultVar(cx, resultTy, basePos, initStmts)

	// Build the nil check condition
	var nilCheckCond ast.BLangExpression
	if lhsHasNil {
		nilCheckCond = createNilTypeTest(lhsVarName, lhsSymbol, lhsTy, basePos)
	}
	if rhsHasNil {
		rhsNilCheck := createNilTypeTest(rhsVarName, rhsSymbol, rhsTy, basePos)
		if nilCheckCond == nil {
			nilCheckCond = rhsNilCheck
		} else {
			orExpr := &ast.BLangBinaryExpr{
				LhsExpr: nilCheckCond,
				RhsExpr: rhsNilCheck,
				OpKind:  model.OperatorKind_OR,
			}
			orExpr.SetDeterminedType(semtypes.Boolean)
			orExpr.SetPosition(basePos)
			nilCheckCond = orExpr
		}
	}

	// Build the operation in the else branch
	var lhsRef ast.BLangExpression
	if lhsHasNil {
		lhsRef = createVarRef(lhsVarName, lhsSymbol, semtypes.Diff(lhsTy, semtypes.Nil))
	} else {
		lhsRef = expr.LhsExpr
	}

	var rhsRef ast.BLangExpression
	if rhsHasNil {
		rhsRef = createVarRef(rhsVarName, rhsSymbol, semtypes.Diff(rhsTy, semtypes.Nil))
	} else {
		rhsRef = expr.RhsExpr
	}

	newBinaryExpr := &ast.BLangBinaryExpr{
		LhsExpr: lhsRef,
		RhsExpr: rhsRef,
		OpKind:  expr.OpKind,
	}
	newBinaryExpr.SetDeterminedType(semtypes.Diff(resultTy, semtypes.Nil))
	newBinaryExpr.SetPosition(basePos)

	resultAssign := createResultAssignment(resultVarName, resultSymbol, resultTy, newBinaryExpr, basePos)

	elseBody := &ast.BLangBlockStmt{
		Stmts: []ast.StatementNode{resultAssign},
	}
	elseBody.SetDeterminedType(semtypes.Never)
	ifStmt := &ast.BLangIf{
		Expr:     nilCheckCond,
		Body:     ast.BLangBlockStmt{},
		ElseStmt: elseBody,
	}
	ifStmt.Body.SetDeterminedType(semtypes.Never)
	ifStmt.SetDeterminedType(semtypes.Never)
	ifStmt.SetScope(cx.currentScope())
	setPositionIfMissing(ifStmt, basePos)
	initStmts = append(initStmts, ifStmt)

	replacementRef := createVarRef(resultVarName, resultSymbol, resultTy)
	setPositionIfMissing(replacementRef, basePos)

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: replacementRef,
	}
}

func walkUnaryExpr(cx *functionContext, expr *ast.BLangUnaryExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
	}

	// Unary + is identity — desugar to just the operand (BIR gen doesn't handle unary +)
	if expr.Operator == model.OperatorKind_ADD {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr.Expr,
		}
	}

	if !isNilLiftableUnaryOp(expr.Operator) {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}

	operandTy := expr.Expr.GetDeterminedType()
	if !semtypes.ContainsBasicType(operandTy, semtypes.Nil) {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}

	basePos := expr.GetPosition()
	resultTy := expr.GetDeterminedType()

	// Create operand temp var
	operandVarName, operandSymbol, initStmts := createOperandTempVar(cx, operandTy, expr.Expr, basePos, initStmts)

	// Create result temp var initialized to nil
	resultVarName, resultSymbol, initStmts := createNilResultVar(cx, resultTy, basePos, initStmts)

	// Build nil check: if ($operand is ()) { } else { ... }
	nilCheck := createNilTypeTest(operandVarName, operandSymbol, operandTy, basePos)

	// Build the operation for the if-body (operand is not nil)
	nonNilTy := semtypes.Diff(operandTy, semtypes.Nil)
	operandRef := createVarRef(operandVarName, operandSymbol, nonNilTy)

	newUnary := &ast.BLangUnaryExpr{
		Expr:     operandRef,
		Operator: expr.Operator,
	}
	newUnary.SetDeterminedType(semtypes.Diff(resultTy, semtypes.Nil))
	newUnary.SetPosition(basePos)
	var opExpr ast.BLangExpression = newUnary

	resultAssign := createResultAssignment(resultVarName, resultSymbol, resultTy, opExpr, basePos)

	// if ($operand is ()) { } else { $result = op $operand }
	elseBody := &ast.BLangBlockStmt{
		Stmts: []ast.StatementNode{resultAssign},
	}
	elseBody.SetDeterminedType(semtypes.Never)
	ifStmt := &ast.BLangIf{
		Expr:     nilCheck,
		Body:     ast.BLangBlockStmt{},
		ElseStmt: elseBody,
	}
	ifStmt.Body.SetDeterminedType(semtypes.Never)
	ifStmt.SetDeterminedType(semtypes.Never)
	ifStmt.SetScope(cx.currentScope())
	setPositionIfMissing(ifStmt, basePos)
	initStmts = append(initStmts, ifStmt)

	replacementRef := createVarRef(resultVarName, resultSymbol, resultTy)
	setPositionIfMissing(replacementRef, basePos)

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: replacementRef,
	}
}

func walkNilConditionalExpr(cx *functionContext, expr *ast.BLangNilConditionalExpr) desugaredNode[ast.BLangActionOrExpression] {
	if cx.trapDepth > 0 {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}

	var initStmts []ast.StatementNode

	lhsResult := walkExpression(cx, expr.LhsExpr)
	initStmts = append(initStmts, lhsResult.initStmts...)
	expr.LhsExpr = lhsResult.replacementNode.(ast.BLangExpression)

	rhsResult := walkExpression(cx, expr.RhsExpr)
	expr.RhsExpr = rhsResult.replacementNode.(ast.BLangExpression)
	if len(rhsResult.initStmts) == 0 {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}

	basePos := expr.GetPosition()
	lhsTy := expr.LhsExpr.GetDeterminedType()
	resultTy := expr.GetDeterminedType()
	lhsName, lhsSymbol, initStmts := createOperandTempVar(cx, lhsTy, expr.LhsExpr, basePos, initStmts)
	resultName, resultSymbol, initStmts := createUninitializedResultVar(cx, resultTy, basePos, initStmts)

	rhsAssignment := createResultAssignment(resultName, resultSymbol, resultTy, expr.RhsExpr, basePos)
	rhsBody := &ast.BLangBlockStmt{Stmts: append(rhsResult.initStmts, rhsAssignment)}
	rhsBody.SetDeterminedType(semtypes.Never)

	lhsRef := createVarRef(lhsName, lhsSymbol, semtypes.Diff(lhsTy, semtypes.Nil))
	setPositionIfMissing(lhsRef, basePos)
	lhsAssignment := createResultAssignment(resultName, resultSymbol, resultTy, lhsRef, basePos)
	lhsBody := &ast.BLangBlockStmt{Stmts: []ast.StatementNode{lhsAssignment}}
	lhsBody.SetDeterminedType(semtypes.Never)

	ifStmt := &ast.BLangIf{
		Expr:     createNilTypeTest(lhsName, lhsSymbol, lhsTy, basePos),
		Body:     *rhsBody,
		ElseStmt: lhsBody,
	}
	ifStmt.SetDeterminedType(semtypes.Never)
	ifStmt.SetScope(cx.currentScope())
	setPositionIfMissing(ifStmt, basePos)
	initStmts = append(initStmts, ifStmt)

	resultRef := createVarRef(resultName, resultSymbol, resultTy)
	setPositionIfMissing(resultRef, basePos)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: resultRef,
	}
}

func walkGroupExpr(cx *functionContext, expr *ast.BLangGroupExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.Expression != nil {
		result := walkExpression(cx, expr.Expression)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expression = result.replacementNode.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkIndexBasedAccess(cx *functionContext, expr *ast.BLangIndexBasedAccess) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
	}

	if expr.IndexExpr != nil {
		result := walkExpression(cx, expr.IndexExpr)
		initStmts = append(initStmts, result.initStmts...)
		expr.IndexExpr = result.replacementNode.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkFieldBaseAccess(cx *functionContext, expr *ast.BLangFieldBaseAccess) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode
	if expr.IsLax() {
		return walkLaxFieldBaseAccess(cx, expr, initStmts)
	}

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
	}

	if expr.IsOptionalAccess() {
		return walkOptionalFieldBaseAccess(cx, expr, initStmts)
	}

	indexAccess := createFieldIndexAccess(expr.Expr, expr.Field.GetValue(), expr.GetDeterminedType(), expr.GetPosition())

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: indexAccess,
	}
}

func walkLaxFieldBaseAccess(cx *functionContext, expr *ast.BLangFieldBaseAccess, initStmts []ast.StatementNode) desugaredNode[ast.BLangActionOrExpression] {
	pos := expr.GetPosition()
	resultTy := expr.GetDeterminedType()
	memberTy := semtypes.Diff(resultTy, semtypes.Error)
	baseTy := expr.Expr.GetDeterminedType()

	receiverResult := walkExpression(cx, expr.Expr)
	initStmts = append(initStmts, receiverResult.initStmts...)
	receiverName, receiverSymbol, initStmts := createOperandTempVar(cx, baseTy, receiverResult.replacementNode.(ast.BLangExpression), pos, initStmts)

	var resultInit ast.BLangExpression
	if expr.IsOptionalAccess() {
		resultInit = createNilLiteral(pos)
	} else {
		resultInit = createErrorWithMessage("lax field access failed", pos)
	}
	resultName, resultSymbol, initStmts := createOperandTempVar(cx, resultTy, resultInit, pos, initStmts)

	receiverTest := func(ty semtypes.SemType, negated bool) *ast.BLangTypeTestExpr {
		ref := createVarRef(receiverName, receiverSymbol, baseTy)
		setPositionIfMissing(ref, pos)
		test := ast.NewBLangTypeTestExpr(pos, ref, ast.TypeData{Type: ty}, negated)
		test.SetDeterminedType(semtypes.Boolean)
		setPositionIfMissing(test, pos)
		return test
	}
	assignReceiver := func(ty semtypes.SemType) *ast.BLangAssignment {
		ref := createVarRef(receiverName, receiverSymbol, ty)
		setPositionIfMissing(ref, pos)
		return createResultAssignment(resultName, resultSymbol, resultTy, ref, pos)
	}
	assignError := func(message string) *ast.BLangAssignment {
		return createResultAssignment(resultName, resultSymbol, resultTy, createErrorWithMessage(message, pos), pos)
	}
	block := func(stmts ...ast.StatementNode) ast.BLangBlockStmt {
		body := ast.BLangBlockStmt{Stmts: stmts}
		body.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(&body, pos)
		return body
	}
	newIf := func(test ast.BLangExpression, body ast.BLangBlockStmt, elseStmt ast.StatementNode) *ast.BLangIf {
		stmt := &ast.BLangIf{Expr: test, Body: body, ElseStmt: elseStmt}
		stmt.SetDeterminedType(semtypes.Never)
		stmt.SetScope(cx.currentScope())
		setPositionIfMissing(stmt, pos)
		return stmt
	}

	mappingTy := semtypes.Intersect(baseTy, semtypes.Mapping)
	mapRef := createVarRef(receiverName, receiverSymbol, mappingTy)
	setPositionIfMissing(mapRef, pos)
	keyExpr := createStringLiteral(expr.Field.GetValue(), pos)
	getInvocation := createLangMapGetInvocation(cx, mapRef, keyExpr, memberTy, pos)
	lookupTy := semtypes.Union(memberTy, semtypes.Error)
	trapExpr := &ast.BLangTrapExpr{Expr: getInvocation}
	trapExpr.SetDeterminedType(lookupTy)
	setPositionIfMissing(trapExpr, pos)
	lookupName, lookupSymbol, lookupStmts := createOperandTempVar(cx, lookupTy, trapExpr, pos, nil)

	lookupErrorRef := createVarRef(lookupName, lookupSymbol, semtypes.Error)
	setPositionIfMissing(lookupErrorRef, pos)
	lookupErrorAssign := createResultAssignment(resultName, resultSymbol, resultTy, lookupErrorRef, pos)
	lookupValueRef := createVarRef(lookupName, lookupSymbol, memberTy)
	setPositionIfMissing(lookupValueRef, pos)
	lookupValueAssign := createResultAssignment(resultName, resultSymbol, resultTy, lookupValueRef, pos)
	lookupTestRef := createVarRef(lookupName, lookupSymbol, lookupTy)
	setPositionIfMissing(lookupTestRef, pos)
	lookupMemberTest := ast.NewBLangTypeTestExpr(pos, lookupTestRef, ast.TypeData{Type: memberTy}, false)
	lookupMemberTest.SetDeterminedType(semtypes.Boolean)
	setPositionIfMissing(lookupMemberTest, pos)
	invalidMemberBody := block(assignError("invalid lax field member"))
	memberIf := newIf(lookupMemberTest, block(lookupValueAssign), &invalidMemberBody)

	lookupErrorTest := createErrorTypeTest(lookupName, lookupSymbol, lookupTy, pos)
	var lookupErrorBody ast.BLangBlockStmt
	if expr.IsOptionalAccess() {
		lookupErrorBody = block()
	} else {
		lookupErrorBody = block(lookupErrorAssign)
	}
	lookupIf := newIf(lookupErrorTest, lookupErrorBody, memberIf)
	mappingBody := block(append(lookupStmts, lookupIf)...)

	notMappingIf := newIf(receiverTest(semtypes.Mapping, true), block(assignError("lax field access receiver is not a mapping")), &mappingBody)
	errorIf := newIf(receiverTest(semtypes.Error, false), block(assignReceiver(semtypes.Error)), notMappingIf)
	var outer ast.StatementNode = errorIf
	if expr.IsOptionalAccess() {
		outer = newIf(receiverTest(semtypes.Nil, false), block(), errorIf)
	}
	initStmts = append(initStmts, outer)

	resultRef := createVarRef(resultName, resultSymbol, resultTy)
	setPositionIfMissing(resultRef, pos)
	return desugaredNode[ast.BLangActionOrExpression]{initStmts: initStmts, replacementNode: resultRef}
}

func createLangMapGetInvocation(cx *functionContext, mapExpr ast.BLangExpression, keyExpr ast.BLangExpression, returnTy semtypes.SemType, pos diagnostics.Location) *ast.BLangInvocation {
	const pkgName = "lang.map"
	space, ok := cx.getImportedSymbolSpace(pkgName)
	if !ok {
		cx.internalError(pkgName+" symbol space not found", pos)
		return nil
	}
	symbolRef, ok := space.GetSymbol("get")
	if !ok {
		cx.internalError(pkgName+":get symbol not found", pos)
		return nil
	}
	cx.addImplicitImport(pkgName, ast.BLangImportPackage{
		OrgName:      &ast.BLangIdentifier{Value: "ballerina"},
		PkgNameComps: []ast.BLangIdentifier{{Value: "lang"}, {Value: "map"}},
		Alias:        &ast.BLangIdentifier{Value: pkgName},
	})
	pkgAlias := &ast.BLangIdentifier{Value: pkgName}
	pkgAlias.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(pkgAlias, pos)
	name := &ast.BLangIdentifier{Value: "get"}
	name.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(name, pos)
	inv := &ast.BLangInvocation{PkgAlias: pkgAlias}
	inv.Name = name
	inv.ArgExprs = []ast.BLangExpression{mapExpr, keyExpr}
	inv.SetSymbol(symbolRef)
	inv.SetDeterminedType(returnTy)
	setPositionIfMissing(inv, pos)
	return inv
}

func createNilLiteral(pos diagnostics.Location) *ast.BLangLiteral {
	lit := &ast.BLangLiteral{Value: nil}
	lit.SetDeterminedType(semtypes.Nil)
	setPositionIfMissing(lit, pos)
	return lit
}

func walkOptionalFieldBaseAccess(cx *functionContext, expr *ast.BLangFieldBaseAccess, initStmts []ast.StatementNode) desugaredNode[ast.BLangActionOrExpression] {
	basePos := expr.GetPosition()
	VName, VSymbol, initStmts := createOperandTempVar(cx, expr.Expr.GetDeterminedType(), expr.Expr, basePos, initStmts)
	resultTy := expr.GetDeterminedType()
	resultName, resultSymbol, initStmts := createNilResultVar(cx, resultTy, basePos, initStmts)

	VForError := createVarRef(VName, VSymbol, semtypes.Error)
	setPositionIfMissing(VForError, basePos)
	errorAssign := createResultAssignment(resultName, resultSymbol, resultTy, VForError, basePos)
	errorBody := &ast.BLangBlockStmt{Stmts: []ast.StatementNode{errorAssign}}
	errorBody.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(errorBody, basePos)

	baseTy := expr.Expr.GetDeterminedType()
	VForIndex := createVarRef(VName, VSymbol, semtypes.Diff(baseTy, semtypes.Error))
	setPositionIfMissing(VForIndex, basePos)
	fieldName := expr.Field.GetValue()
	indexAccess := createFieldIndexAccess(VForIndex, fieldName, optionalFieldIndexResultType(cx, baseTy, fieldName), basePos)
	indexAssign := createResultAssignment(resultName, resultSymbol, resultTy, indexAccess, basePos)
	elseBody := &ast.BLangBlockStmt{Stmts: []ast.StatementNode{indexAssign}}
	elseBody.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(elseBody, basePos)

	// TODO: update when handling lax case https://github.com/ballerina-nutcracker/ballerina/issues/558
	ifStmt := &ast.BLangIf{
		Expr:     createErrorTypeTest(VName, VSymbol, baseTy, basePos),
		Body:     *errorBody,
		ElseStmt: elseBody,
	}
	ifStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(ifStmt, basePos)
	initStmts = append(initStmts, ifStmt)

	replacementRef := createVarRef(resultName, resultSymbol, resultTy)
	setPositionIfMissing(replacementRef, basePos)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: replacementRef,
	}
}

func optionalFieldIndexResultType(cx *functionContext, baseTy semtypes.SemType, fieldName string) semtypes.SemType {
	tyCtx := cx.typeCtx()
	mappingTy := semtypes.Intersect(semtypes.Diff(semtypes.Diff(baseTy, semtypes.Error), semtypes.Nil), semtypes.Mapping)
	memberTy := semtypes.MappingMemberTypeInner(tyCtx, mappingTy, semtypes.StringConst(fieldName))
	if semtypes.ContainsUndef(memberTy) || semtypes.IsSubtype(tyCtx, semtypes.Nil, baseTy) {
		return semtypes.Union(semtypes.Diff(memberTy, semtypes.Undef), semtypes.Nil)
	}
	return memberTy
}

func createFieldIndexAccess(expr ast.BLangExpression, fieldName string, ty semtypes.SemType, pos diagnostics.Location) *ast.BLangIndexBasedAccess {
	lit := &ast.BLangLiteral{
		Value:         fieldName,
		OriginalValue: fieldName,
	}
	lit.SetPosition(pos)
	lit.SetDeterminedType(semtypes.String)

	indexAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: lit,
	}
	indexAccess.Expr = expr
	indexAccess.SetPosition(pos)
	indexAccess.SetDeterminedType(ty)
	return indexAccess
}

func createErrorTypeTest(varName *ast.BLangIdentifier, symbol model.SymbolRef, ty semtypes.SemType, pos diagnostics.Location) *ast.BLangTypeTestExpr {
	ref := createVarRef(varName, symbol, ty)
	setPositionIfMissing(ref, pos)
	typeTest := ast.NewBLangTypeTestExpr(pos, ref, ast.TypeData{Type: semtypes.Error}, false)
	typeTest.SetDeterminedType(semtypes.Boolean)
	return typeTest
}

func walkTemplateExpr(cx *functionContext, expr *ast.BLangTemplateExpr) desugaredNode[ast.BLangActionOrExpression] {
	if len(expr.Insertions) == 0 {
		lit := &ast.BLangLiteral{Value: expr.Strings[0], OriginalValue: expr.Strings[0]}
		lit.SetPosition(expr.GetPosition())
		lit.SetDeterminedType(semtypes.StringConst(expr.Strings[0]))
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: lit}
	}
	var initStmts []ast.StatementNode
	for i, ins := range expr.Insertions {
		r := walkExpression(cx, ins)
		initStmts = append(initStmts, r.initStmts...)
		expr.Insertions[i] = r.replacementNode.(ast.BLangExpression)
	}
	return desugaredNode[ast.BLangActionOrExpression]{initStmts: initStmts, replacementNode: expr}
}

func walkStartAction(cx *functionContext, action *ast.BLangStartAction) desugaredNode[ast.BLangActionOrExpression] {
	callResult := walkExpression(cx, action.Call)
	action.Call = callResult.replacementNode.(ast.Invocable)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       callResult.initStmts,
		replacementNode: action,
	}
}

func walkClientResourceAccessAction(cx *functionContext, expr *ast.BLangClientResourceAccessAction) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode
	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
	}
	for i := range expr.Path {
		seg := &expr.Path[i]
		if seg.Kind != ast.ResourceAccessSegmentComputed {
			continue
		}
		result := walkExpression(cx, seg.Expr)
		initStmts = append(initStmts, result.initStmts...)
		seg.Expr = result.replacementNode.(ast.BLangExpression)
	}
	initStmts = append(initStmts, walkInvocationArgs(cx, expr)...)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkXMLTemplateExpr(cx *functionContext, expr *ast.BLangXMLTemplateExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode
	for i, ins := range expr.Insertions {
		r := walkExpression(cx, ins)
		initStmts = append(initStmts, r.initStmts...)
		insert := r.replacementNode.(ast.BLangExpression)
		if shouldEscapeXMLTemplateInsertion(insert, expr.InsertionKinds[i], cx) {
			insert = escapeXMLTemplateInsertion(cx, insert, expr.InsertionKinds[i])
		}
		expr.Insertions[i] = insert
	}
	expr.Strings = spliceXMLTemplateNamespaces(cx, expr.Strings, expr.NamespaceInsertions)
	plain := &ast.BLangTemplateExpr{Kind: ast.TemplateExprKindXML, Strings: expr.Strings, Insertions: expr.Insertions}
	plain.SetPosition(expr.GetPosition())
	plain.SetDeterminedType(expr.GetDeterminedType())
	return desugaredNode[ast.BLangActionOrExpression]{initStmts: initStmts, replacementNode: plain}
}

func shouldEscapeXMLTemplateInsertion(insert ast.BLangExpression, kind ast.XMLTemplateInsertionKind, cx *functionContext) bool {
	if kind == ast.XMLTemplateInsertionKindAttribute {
		return true
	}
	// content needs to be escaped if they are not xml
	return !semtypes.IsSubtype(semtypes.ContextFrom(cx.typeEnv()), insert.GetDeterminedType(), semtypes.XML)
}

func escapeXMLTemplateInsertion(cx *functionContext, insert ast.BLangExpression, kind ast.XMLTemplateInsertionKind) ast.BLangExpression {
	switch kind {
	case ast.XMLTemplateInsertionKindAttribute:
		return createLangInternalInvocation(cx, "escapeXMLAttribute", semtypes.String, []ast.BLangExpression{insert}, insert.GetPosition())
	case ast.XMLTemplateInsertionKindContent:
		return createLangInternalInvocation(cx, "escapeXMLContent", semtypes.String, []ast.BLangExpression{insert}, insert.GetPosition())
	default:
		cx.internalError("unexpected xml template insert kind", insert.GetPosition())
		return insert
	}
}

type xmlNamespaceDecl struct {
	key string
	uri string
}

func xmlTemplateNamespaceDecls(cx *functionContext, refs []model.SymbolRef) []xmlNamespaceDecl {
	decls := make([]xmlNamespaceDecl, 0, len(refs))
	for _, ref := range refs {
		symbol := cx.getSymbol(ref)
		key, err := model.XMLNamespaceDeclKey(symbol)
		if err != nil {
			cx.internalError(err.Error(), symbol.Location())
			continue
		}
		uri, err := model.XMLNamespaceURI(symbol)
		if err != nil {
			cx.internalError(err.Error(), symbol.Location())
			continue
		}
		decls = append(decls, xmlNamespaceDecl{key: key, uri: uri})
	}
	return decls
}

func spliceXMLTemplateNamespaces(cx *functionContext, parts []string, insertions [][]ast.XMLTemplateNamespaceInsertion) []string {
	if len(insertions) == 0 || len(parts) == 0 {
		return parts
	}
	out := append([]string(nil), parts...)
	for stringIndex, stringInsertions := range insertions {
		if stringIndex >= len(out) || len(stringInsertions) == 0 {
			continue
		}
		ordered := append([]ast.XMLTemplateNamespaceInsertion(nil), stringInsertions...)
		sort.SliceStable(ordered, func(i, j int) bool {
			return ordered[i].Offset > ordered[j].Offset
		})
		for _, insn := range ordered {
			if len(insn.Namespaces) == 0 {
				continue
			}
			part := out[stringIndex]
			if insn.Offset < 0 || insn.Offset > len(part) {
				continue
			}

			namespaces := xmlTemplateNamespaceDecls(cx, insn.Namespaces)
			sort.SliceStable(namespaces, func(i, j int) bool {
				return namespaces[i].key < namespaces[j].key
			})

			var b strings.Builder
			b.WriteString(part[:insn.Offset])
			for _, ns := range namespaces {
				b.WriteByte(' ')
				b.WriteString(ns.key)
				b.WriteString("=\"")
				b.WriteString(values.EscapeXMLAttribute(ns.uri))
				b.WriteByte('"')
			}
			b.WriteString(part[insn.Offset:])
			out[stringIndex] = b.String()
		}
	}
	return out
}

func walkInvocation(cx *functionContext, expr invocable) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.Receiver() != nil {
		result := walkExpression(cx, expr.Receiver())
		initStmts = append(initStmts, result.initStmts...)
		expr.SetReceiver(result.replacementNode.(ast.BLangExpression))
	}

	initStmts = append(initStmts, walkInvocationArgs(cx, expr)...)

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkInvocationArgs(cx *functionContext, expr invocable) []ast.StatementNode {
	initStmts, args := walkCallArgs(cx, expr.CallArgs(), expr.GetPosition(), func() (model.UntypedFunctionSignature, bool) {
		return cx.functionSignature(expr.ResolvedSymbol())
	})
	expr.SetCallArgs(args)
	return initStmts
}

func walkCallArgs(cx *functionContext, args []ast.BLangExpression, pos diagnostics.Location, fnSig func() (model.UntypedFunctionSignature, bool)) ([]ast.StatementNode, []ast.BLangExpression) {
	if shouldHoistArgs(args) {
		sig, ok := fnSig()
		if !ok {
			cx.internalError("expected function signature to default expressions", pos)
			return nil, args
		}
		return hoistAndAddDefaultInvocations(cx, args, sig, pos)
	}

	initStmts := make([]ast.StatementNode, 0, len(args))
	walkedArgs := make([]ast.BLangExpression, len(args))
	for i, arg := range args {
		result := walkExpression(cx, arg)
		initStmts = append(initStmts, result.initStmts...)
		walkedArgs[i] = result.replacementNode.(ast.BLangExpression)
	}
	return initStmts, walkedArgs
}

// If you have default invocations you should hoist the other args to local declarations such that we don't
// invoke operations (such as function calls) twice
func shouldHoistArgs(args []ast.BLangExpression) bool {
	for _, arg := range args {
		if _, ok := arg.(*ast.BLangDefaultArg); ok {
			return true
		}
	}
	return false
}

func hoistAndAddDefaultInvocations(cx *functionContext, args []ast.BLangExpression, fnSig model.UntypedFunctionSignature, pos diagnostics.Location) ([]ast.StatementNode, []ast.BLangExpression) {
	fixedCount := fnSig.FixedParamCount()
	hoistInit := make([]ast.StatementNode, 0, len(args))
	hoistedArgs := make([]ast.BLangExpression, len(args))
	for i := range fixedCount {
		arg := args[i]
		if defaultArg, ok := arg.(*ast.BLangDefaultArg); ok {
			defaultClosureSym := defaultArg.DefaultClosure
			defaultCallSym := defaultClosureSym
			if localSym, ok := cx.defaultClosureVars[defaultClosureSym]; ok {
				defaultCallSym = localSym
			}
			defaultCall := &ast.BLangInvocation{}
			defaultCall.Name = newIdentifier(cx.pkgCtx.compilerCtx.SymbolName(defaultCallSym))
			defaultCall.ArgExprs = append([]ast.BLangExpression(nil), hoistedArgs[:i]...)
			defaultCall.SetSymbol(defaultCallSym)
			defaultCall.SetDeterminedType(cx.getSymbol(defaultClosureSym).(model.FunctionSymbol).TypedSignature().ReturnType)
			setPositionIfMissing(defaultCall, pos)
			result := walkExpression(cx, defaultCall)
			hoistInit = append(hoistInit, result.initStmts...)
			varDef, varRef := assignToLocal(cx, result.replacementNode.(ast.BLangExpression), pos)
			hoistInit = append(hoistInit, varDef)
			hoistedArgs[i] = varRef
		} else {
			result := walkExpression(cx, arg)
			hoistInit = append(hoistInit, result.initStmts...)
			varDef, varRef := assignToLocal(cx, result.replacementNode.(ast.BLangExpression), arg.GetPosition())
			hoistInit = append(hoistInit, varDef)
			hoistedArgs[i] = varRef
		}
	}
	for i := fixedCount; i < len(args); i++ {
		arg := args[i]
		result := walkExpression(cx, arg)
		hoistInit = append(hoistInit, result.initStmts...)
		hoistedArgs[i] = result.replacementNode.(ast.BLangExpression)
	}

	return hoistInit, hoistedArgs
}

func invocationSymbol(cx *functionContext, expr invocable) (model.SymbolRef, bool) {
	switch e := expr.(type) {
	case *ast.BLangInvocation:
		if e.RawSymbol == nil {
			return model.SymbolRef{}, false
		}
		return e.ResolvedSymbol(), true
	case *ast.BLangRemoteMethodCallAction:
		if e.RawSymbol == nil {
			return model.SymbolRef{}, false
		}
		return e.ResolvedSymbol(), true
	default:
		cx.internalError(fmt.Sprintf("unexpected invocation type: %T", expr), expr.GetPosition())
		return model.SymbolRef{}, false
	}
}

// synthesizeInferredTypedescArg builds the typedesc expression that fills a
// `typedesc param = <>` slot. The monomorphized signature's param type is
// typedesc<T>; we unwrap it to recover T as the constraint.
func synthesizeInferredTypedescArg(cx *functionContext, tdTy semtypes.SemType, pos diagnostics.Location) *ast.BLangTypedescExpr {
	tyCtx := cx.typeCtx()
	tdExpr := &ast.BLangTypedescExpr{Constraint: semtypes.TypedescConstraint(tyCtx, tdTy)}
	tdExpr.SetPosition(pos)
	tdExpr.SetDeterminedType(tdTy)
	return tdExpr
}

func assignToLocal(cx *functionContext, initExpr ast.BLangExpression, pos diagnostics.Location) (ast.StatementNode, *ast.BLangVarRef) {
	ty := initExpr.GetDeterminedType()
	tempName, tempSymRef := cx.addDesugardSymbol(ty, model.SymbolKindVariable, pos)
	tempVar := &ast.BLangVariable{Name: newIdentifier(tempName)}
	tempVar.Name.SetDeterminedType(semtypes.Never)
	tempVar.SetDeterminedType(semtypes.Never)
	tempVar.SetInitialExpression(initExpr)
	tempVar.SetSymbol(tempSymRef)
	varDef := &ast.BLangVariableDef{}
	varDef.SetVariable(tempVar)
	varDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(varDef, pos)

	varRef := &ast.BLangVarRef{VariableName: tempVar.Name}
	varRef.SetSymbol(tempSymRef)
	varRef.SetDeterminedType(ty)
	setPositionIfMissing(varRef, pos)
	return varDef, varRef
}

func walkListConstructorExpr(cx *functionContext, expr *ast.BLangListConstructorExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	for i := range expr.Exprs {
		result := walkExpression(cx, expr.Exprs[i])
		initStmts = append(initStmts, result.initStmts...)
		expr.Exprs[i] = result.replacementNode.(ast.BLangExpression)
	}

	if expr.HasSpreadMembers() {
		return desugarListConstructorWithSpread(cx, expr, initStmts)
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func desugarListConstructorWithSpread(
	cx *functionContext,
	expr *ast.BLangListConstructorExpr,
	initStmts []ast.StatementNode,
) desugaredNode[ast.BLangActionOrExpression] {
	pos := expr.GetPosition()
	emptyList := &ast.BLangListConstructorExpr{Exprs: []ast.BLangExpression{}}
	emptyList.SetDeterminedType(expr.GetDeterminedType())
	emptyList.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(emptyList, pos)

	resultDef, resultRef := assignToLocal(cx, emptyList, pos)
	initStmts = append(initStmts, resultDef)

	for i, memberExpr := range expr.Exprs {
		if !expr.IsSpreadMember(i) {
			pushMember := createPushInvocation(cx, resultRef, memberExpr)
			if pushMember == nil {
				return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
			}
			pushStmt := &ast.BLangExpressionStmt{Expr: pushMember}
			setPositionIfMissing(pushStmt, pos)
			initStmts = append(initStmts, pushStmt)
			continue
		}
		initStmts = appendSpreadListPushStmts(cx, initStmts, resultRef, memberExpr, pos)
	}

	resultRef.SetDeterminedType(expr.GetDeterminedType())
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: resultRef,
	}
}

func appendSpreadListPushStmts(
	cx *functionContext,
	initStmts []ast.StatementNode,
	resultRef *ast.BLangVarRef,
	spreadExpr ast.BLangExpression,
	pos diagnostics.Location,
) []ast.StatementNode {
	spreadDef, spreadRef := assignToLocal(cx, spreadExpr, pos)
	initStmts = append(initStmts, spreadDef)

	lengthRef, ok := createQueryLengthRef(cx, &initStmts, spreadRef, pos)
	if !ok {
		return initStmts
	}
	counterRef := createQueryCounterRef(cx, &initStmts, pos)
	tyCtx := semtypes.ContextFrom(cx.typeEnv())
	elemTy := semtypes.ListProj(tyCtx, spreadExpr.GetDeterminedType(), semtypes.Int)
	spreadAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: counterRef,
	}
	spreadAccess.Expr = spreadRef
	spreadAccess.SetDeterminedType(elemTy)
	setPositionIfMissing(spreadAccess, pos)

	pushMember := createPushInvocation(cx, resultRef, spreadAccess)
	if pushMember == nil {
		return initStmts
	}
	pushStmt := &ast.BLangExpressionStmt{Expr: pushMember}
	setPositionIfMissing(pushStmt, pos)
	bodyStmts := []ast.StatementNode{
		pushStmt,
		createIncrementStmt(counterRef),
	}
	cond := &ast.BLangBinaryExpr{
		LhsExpr: counterRef,
		RhsExpr: lengthRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	initStmts = append(initStmts, whileStmt)
	return initStmts
}

func walkErrorConstructorExpr(cx *functionContext, expr *ast.BLangErrorConstructorExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	//nolint:staticcheck // TODO
	if expr.ErrorTypeRef != nil {
		// ErrorTypeRef is a type descriptor, not an expression, so we don't walk it
	}

	for i := range expr.PositionalArgs {
		result := walkExpression(cx, expr.PositionalArgs[i])
		initStmts = append(initStmts, result.initStmts...)
		expr.PositionalArgs[i] = result.replacementNode.(ast.BLangExpression)
	}

	for i := range expr.NamedArgs {
		result := walkExpression(cx, expr.NamedArgs[i].Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.NamedArgs[i].Expr = result.replacementNode.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkCheckedExpr(cx *functionContext, expr *ast.BLangCheckedExpr) desugaredNode[ast.BLangActionOrExpression] {
	result := walkExpression(cx, expr.Expr)
	expr.Expr = result.replacementNode
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       result.initStmts,
		replacementNode: expr,
	}
}

func walkCheckPanickedExpr(cx *functionContext, expr *ast.BLangCheckPanickedExpr) desugaredNode[ast.BLangActionOrExpression] {
	result := walkExpression(cx, expr.Expr)
	expr.Expr = result.replacementNode
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       result.initStmts,
		replacementNode: expr,
	}
}

func walkTrapExpr(cx *functionContext, expr *ast.BLangTrapExpr) desugaredNode[ast.BLangActionOrExpression] {
	cx.trapDepth++
	result := walkExpression(cx, expr.Expr)
	cx.trapDepth--
	if len(result.initStmts) > 0 {
		// I don't think this can ever happen but if it does we need to think about how to add these statements in to the
		// trap region in BIR gen
		cx.internalError("Init statements will be hoisted outside of trap region", expr.GetPosition())
	}
	expr.Expr = result.replacementNode
	return desugaredNode[ast.BLangActionOrExpression]{initStmts: nil, replacementNode: expr}
}

func walkLambdaFunction(cx *functionContext, expr *ast.BLangLambdaFunction) desugaredNode[ast.BLangActionOrExpression] {
	if expr.Function != nil {
		if cx.pkgCtx.needsDefaultClosures(expr.Function.Symbol()) {
			defaults := desugarFunctionParamDefaults(cx, expr.Function, expr.Function.Symbol(), expr.Function.Scope())
			for i := range defaults {
				defaults[i] = desugarNestedFunction(cx, defaults[i])
			}
			cx.generatedFunctions = append(cx.generatedFunctions, defaults...)
		}
		expr.Function = desugarNestedFunction(cx, expr.Function)
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		replacementNode: expr,
	}
}

func walkTypeConversionExpr(cx *functionContext, expr *ast.BLangTypeConversionExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.Expression != nil {
		result := walkExpression(cx, expr.Expression)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expression = result.replacementNode.(ast.BLangExpression)
	}
	if fnType, ok := expr.TypeDescriptor.(*ast.BLangFunctionType); ok {
		result := desugarFunctionTypeDesc(cx, fnType, cx.currentScope())
		initStmts = append(initStmts, desugarLocalTypeDescDefaults(cx, result.functions)...)
		for _, field := range result.recordFields {
			initStmts = append(initStmts, desugarRecordFieldDefault(cx, field))
		}
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkTypeTestExpr(cx *functionContext, expr *ast.BLangTypeTestExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
	}
	if fnType, ok := expr.Type.TypeDescriptor.(*ast.BLangFunctionType); ok {
		result := desugarFunctionTypeDesc(cx, fnType, cx.currentScope())
		initStmts = append(initStmts, desugarLocalTypeDescDefaults(cx, result.functions)...)
		for _, field := range result.recordFields {
			initStmts = append(initStmts, desugarRecordFieldDefault(cx, field))
		}
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkAnnotAccessExpr(cx *functionContext, expr *ast.BLangAnnotAccessExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkArrowFunction(cx *functionContext, expr *ast.BLangArrowFunction) desugaredNode[ast.BLangActionOrExpression] {
	// Arrow functions have a body that may need desugaring
	if expr.Body != nil {
		result := walkExpression(cx, expr.Body.Expr.(ast.BLangActionOrExpression))
		expr.Body.Expr = result.replacementNode.(ast.BLangExpression)
		// Handle initStmts if needed - arrow functions may need special handling
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		replacementNode: expr,
	}
}

func walkNewExpression(cx *functionContext, expr *ast.BLangNewExpression) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode
	argsInit, args := walkCallArgs(cx, expr.ArgsExprs, expr.GetPosition(), func() (model.UntypedFunctionSignature, bool) {
		return initFunctionSymbol(cx, expr)
	})
	initStmts = append(initStmts, argsInit...)

	expr.ArgsExprs = args
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func initFunctionSymbol(cx *functionContext, expr *ast.BLangNewExpression) (model.UntypedFunctionSignature, bool) {
	if expr.ClassSymbol.IsEmpty() {
		return model.UntypedFunctionSignature{}, false
	}
	classSym, ok := cx.getSymbol(expr.ClassSymbol).(model.ClassSymbol)
	if !ok {
		return model.UntypedFunctionSignature{}, false
	}
	initRef, ok := classSym.MethodSymbol("init")
	if !ok {
		return model.UntypedFunctionSignature{}, false
	}
	untypedSig, ok := cx.pkgCtx.compilerCtx.GetFunctionSignature(initRef)
	return untypedSig, ok
}

func walkMappingConstructorExpr(cx *functionContext, expr *ast.BLangMappingConstructorExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	for _, field := range expr.Fields {
		kv := field.(*ast.BLangMappingKeyValueField)

		if kv.Key.Kind != ast.MappingKeyComputed {
			if varRef, ok := kv.Key.Expr.(*ast.BLangVarRef); ok {
				name := varRef.VariableName.GetValue()
				lit := &ast.BLangLiteral{
					Value:         name,
					OriginalValue: name,
				}
				lit.SetPosition(varRef.GetPosition())
				lit.SetDeterminedType(semtypes.String)
				kv.Key.Expr = lit
			}
		}

		result := walkExpression(cx, kv.ValueExpr)
		initStmts = append(initStmts, result.initStmts...)
		kv.ValueExpr = result.replacementNode.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func isNilLiftableBinaryOp(op model.OperatorKind) bool {
	switch op {
	case model.OperatorKind_ADD, model.OperatorKind_SUB,
		model.OperatorKind_MUL, model.OperatorKind_DIV, model.OperatorKind_MOD,
		model.OperatorKind_BITWISE_LEFT_SHIFT, model.OperatorKind_BITWISE_RIGHT_SHIFT,
		model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT,
		model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		return true
	default:
		return false
	}
}

func isNilLiftableUnaryOp(op model.OperatorKind) bool {
	switch op {
	case model.OperatorKind_ADD, model.OperatorKind_SUB, model.OperatorKind_BITWISE_COMPLEMENT:
		return true
	default:
		return false
	}
}

func createOperandTempVar(cx *functionContext, ty semtypes.SemType, initExpr ast.BLangExpression, pos diagnostics.Location, initStmts []ast.StatementNode) (*ast.BLangIdentifier, model.SymbolRef, []ast.StatementNode) {
	name, symbol := cx.addDesugardSymbol(ty, model.SymbolKindVariable, pos)
	varName := newIdentifier(name)
	tempVar := &ast.BLangVariable{Name: varName}
	tempVar.Name.SetDeterminedType(semtypes.Never)
	tempVar.SetDeterminedType(semtypes.Never)
	tempVar.SetInitialExpression(initExpr)
	tempVar.SetSymbol(symbol)
	varDef := &ast.BLangVariableDef{Var: tempVar}
	varDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(varDef, pos)
	return varName, symbol, append(initStmts, varDef)
}

func createNilResultVar(cx *functionContext, ty semtypes.SemType, pos diagnostics.Location, initStmts []ast.StatementNode) (*ast.BLangIdentifier, model.SymbolRef, []ast.StatementNode) {
	nilLit := &ast.BLangLiteral{Value: nil}
	nilLit.SetDeterminedType(semtypes.Nil)
	setPositionIfMissing(nilLit, pos)

	name, symbol := cx.addDesugardSymbol(ty, model.SymbolKindVariable, pos)
	varName := newIdentifier(name)
	tempVar := &ast.BLangVariable{Name: varName}
	tempVar.Name.SetDeterminedType(semtypes.Never)
	tempVar.SetDeterminedType(semtypes.Never)
	tempVar.SetInitialExpression(nilLit)
	tempVar.SetSymbol(symbol)
	varDef := &ast.BLangVariableDef{Var: tempVar}
	varDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(varDef, pos)
	return varName, symbol, append(initStmts, varDef)
}

func createUninitializedResultVar(cx *functionContext, ty semtypes.SemType, pos diagnostics.Location, initStmts []ast.StatementNode) (*ast.BLangIdentifier, model.SymbolRef, []ast.StatementNode) {
	name, symbol := cx.addDesugardSymbol(ty, model.SymbolKindVariable, pos)
	varName := newIdentifier(name)
	tempVar := &ast.BLangVariable{Name: varName}
	tempVar.Name.SetDeterminedType(semtypes.Never)
	tempVar.SetDeterminedType(semtypes.Never)
	tempVar.SetSymbol(symbol)
	varDef := &ast.BLangVariableDef{Var: tempVar}
	varDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(varDef, pos)
	return varName, symbol, append(initStmts, varDef)
}

func createNilTypeTest(varName *ast.BLangIdentifier, symbol model.SymbolRef, ty semtypes.SemType, pos diagnostics.Location) *ast.BLangTypeTestExpr {
	ref := createVarRef(varName, symbol, ty)
	typeTest := ast.NewBLangTypeTestExpr(pos, ref, ast.TypeData{Type: semtypes.Nil}, false)
	typeTest.SetDeterminedType(semtypes.Boolean)
	return typeTest
}

func createVarRef(varName ast.IdentifierNode, symbol model.SymbolRef, ty semtypes.SemType) *ast.BLangVarRef {
	ref := &ast.BLangVarRef{VariableName: varName}
	ref.SetSymbol(symbol)
	ref.SetDeterminedType(ty)
	return ref
}

func createResultAssignment(resultVarName ast.IdentifierNode, resultSymbol model.SymbolRef, resultTy semtypes.SemType, valueExpr ast.BLangActionOrExpression, pos diagnostics.Location) *ast.BLangAssignment {
	varRef := createVarRef(resultVarName, resultSymbol, resultTy)
	assign := &ast.BLangAssignment{
		VarRef: varRef,
		Expr:   valueExpr,
	}
	assign.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assign, pos)
	return assign
}
