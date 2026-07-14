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

	"ballerina-lang-go/ast"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/values"
)

type invocable interface {
	ast.BLangActionOrExpression
	ResolvedSymbol() model.SymbolRef
	Receiver() ast.BLangExpression
	SetReceiver(ast.BLangExpression)
	CallArgs() []ast.BLangExpression
	SetCallArgs([]ast.BLangExpression)
}

func walkExpression(cx *functionContext, node ast.BLangActionOrExpression) desugaredNode[ast.BLangActionOrExpression] {
	switch expr := node.(type) {
	case *ast.BLangBinaryExpr:
		return walkBinaryExpr(cx, expr)
	case *ast.BLangUnaryExpr:
		return walkUnaryExpr(cx, expr)
	case *ast.BLangElvisExpr:
		return walkElvisExpr(cx, expr)
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
	case *ast.BLangSimpleVarRef:
		if replacement := materializeConstantRef(cx, expr); replacement != nil {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: replacement}
		}
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangLocalVarRef:
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangConstRef:
		if replacement := materializeConstantRef(cx, &expr.BLangSimpleVarRef); replacement != nil {
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
		panic(fmt.Sprintf("unexpected expression type: %T", node))
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
	lhsHasNil := semtypes.ContainsBasicType(lhsTy, semtypes.NIL)
	rhsHasNil := semtypes.ContainsBasicType(rhsTy, semtypes.NIL)

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
			orExpr.SetDeterminedType(semtypes.BOOLEAN)
			orExpr.SetPosition(basePos)
			nilCheckCond = orExpr
		}
	}

	// Build the operation in the else branch
	var lhsRef ast.BLangExpression
	if lhsHasNil {
		lhsRef = createVarRef(lhsVarName, lhsSymbol, semtypes.Diff(lhsTy, semtypes.NIL))
	} else {
		lhsRef = expr.LhsExpr
	}

	var rhsRef ast.BLangExpression
	if rhsHasNil {
		rhsRef = createVarRef(rhsVarName, rhsSymbol, semtypes.Diff(rhsTy, semtypes.NIL))
	} else {
		rhsRef = expr.RhsExpr
	}

	newBinaryExpr := &ast.BLangBinaryExpr{
		LhsExpr: lhsRef,
		RhsExpr: rhsRef,
		OpKind:  expr.OpKind,
	}
	newBinaryExpr.SetDeterminedType(semtypes.Diff(resultTy, semtypes.NIL))
	newBinaryExpr.SetPosition(basePos)

	resultAssign := createResultAssignment(resultVarName, resultSymbol, resultTy, newBinaryExpr, basePos)

	elseBody := &ast.BLangBlockStmt{
		Stmts: []ast.StatementNode{resultAssign},
	}
	elseBody.SetDeterminedType(semtypes.NEVER)
	ifStmt := &ast.BLangIf{
		Expr:     nilCheckCond,
		Body:     ast.BLangBlockStmt{},
		ElseStmt: elseBody,
	}
	ifStmt.Body.SetDeterminedType(semtypes.NEVER)
	ifStmt.SetDeterminedType(semtypes.NEVER)
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
	if !semtypes.ContainsBasicType(operandTy, semtypes.NIL) {
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
	nonNilTy := semtypes.Diff(operandTy, semtypes.NIL)
	operandRef := createVarRef(operandVarName, operandSymbol, nonNilTy)

	newUnary := &ast.BLangUnaryExpr{
		Expr:     operandRef,
		Operator: expr.Operator,
	}
	newUnary.SetDeterminedType(semtypes.Diff(resultTy, semtypes.NIL))
	newUnary.SetPosition(basePos)
	var opExpr ast.BLangExpression = newUnary

	resultAssign := createResultAssignment(resultVarName, resultSymbol, resultTy, opExpr, basePos)

	// if ($operand is ()) { } else { $result = op $operand }
	elseBody := &ast.BLangBlockStmt{
		Stmts: []ast.StatementNode{resultAssign},
	}
	elseBody.SetDeterminedType(semtypes.NEVER)
	ifStmt := &ast.BLangIf{
		Expr:     nilCheck,
		Body:     ast.BLangBlockStmt{},
		ElseStmt: elseBody,
	}
	ifStmt.Body.SetDeterminedType(semtypes.NEVER)
	ifStmt.SetDeterminedType(semtypes.NEVER)
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

func walkElvisExpr(cx *functionContext, expr *ast.BLangElvisExpr) desugaredNode[ast.BLangActionOrExpression] {
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

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
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

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode.(ast.BLangExpression)
	}

	if expr.IsOptionalAccess() {
		return walkOptionalFieldBaseAccess(cx, expr, initStmts)
	}

	indexAccess := createFieldIndexAccess(expr.Expr, expr.Field.Value, expr.GetDeterminedType(), expr.GetPosition())

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: indexAccess,
	}
}

func walkOptionalFieldBaseAccess(cx *functionContext, expr *ast.BLangFieldBaseAccess, initStmts []ast.StatementNode) desugaredNode[ast.BLangActionOrExpression] {
	basePos := expr.GetPosition()
	VName, VSymbol, initStmts := createOperandTempVar(cx, expr.Expr.GetDeterminedType(), expr.Expr, basePos, initStmts)
	resultTy := expr.GetDeterminedType()
	resultName, resultSymbol, initStmts := createNilResultVar(cx, resultTy, basePos, initStmts)

	VForError := createVarRef(VName, VSymbol, semtypes.ERROR)
	setPositionIfMissing(VForError, basePos)
	errorAssign := createResultAssignment(resultName, resultSymbol, resultTy, VForError, basePos)
	errorBody := &ast.BLangBlockStmt{Stmts: []ast.StatementNode{errorAssign}}
	errorBody.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(errorBody, basePos)

	baseTy := expr.Expr.GetDeterminedType()
	VForIndex := createVarRef(VName, VSymbol, semtypes.Diff(baseTy, semtypes.ERROR))
	setPositionIfMissing(VForIndex, basePos)
	indexAccess := createFieldIndexAccess(VForIndex, expr.Field.Value, optionalFieldIndexResultType(cx, baseTy, expr.Field.Value), basePos)
	indexAssign := createResultAssignment(resultName, resultSymbol, resultTy, indexAccess, basePos)
	elseBody := &ast.BLangBlockStmt{Stmts: []ast.StatementNode{indexAssign}}
	elseBody.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(elseBody, basePos)

	// TODO: update when handling lax case https://github.com/ballerina-platform/ballerina-lang-go/issues/558
	ifStmt := &ast.BLangIf{
		Expr:     createErrorTypeTest(VName, VSymbol, baseTy, basePos),
		Body:     *errorBody,
		ElseStmt: elseBody,
	}
	ifStmt.SetDeterminedType(semtypes.NEVER)
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
	mappingTy := semtypes.Intersect(semtypes.Diff(semtypes.Diff(baseTy, semtypes.ERROR), semtypes.NIL), semtypes.MAPPING)
	memberTy := semtypes.MappingMemberTypeInner(tyCtx, mappingTy, semtypes.StringConst(fieldName))
	if semtypes.ContainsUndef(memberTy) || semtypes.IsSubtype(tyCtx, semtypes.NIL, baseTy) {
		return semtypes.Union(semtypes.Diff(memberTy, semtypes.UNDEF), semtypes.NIL)
	}
	return memberTy
}

func createFieldIndexAccess(expr ast.BLangExpression, fieldName string, ty semtypes.SemType, pos diagnostics.Location) *ast.BLangIndexBasedAccess {
	lit := &ast.BLangLiteral{
		Value:         fieldName,
		OriginalValue: fieldName,
	}
	lit.SetPosition(pos)
	lit.SetDeterminedType(semtypes.STRING)

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
	typeTest := &ast.BLangTypeTestExpr{
		Expr: ref,
		Type: ast.TypeData{Type: semtypes.ERROR},
	}
	typeTest.SetDeterminedType(semtypes.BOOLEAN)
	setPositionIfMissing(typeTest, pos)
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
	for i := range expr.ArgExprs {
		result := walkExpression(cx, expr.ArgExprs[i])
		initStmts = append(initStmts, result.initStmts...)
		expr.ArgExprs[i] = result.replacementNode.(ast.BLangExpression)
	}
	// Synthesize omitted defaultable arguments, mirroring walkInvocation's
	// direct-call handling for the resolved resource method.
	if fnSym, ok := cx.getSymbol(expr.MethodSymbol()).(model.FunctionSymbol); ok {
		initStmts = append(initStmts, walkDirectCallArgs(cx, expr, fnSym)...)
	}
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
		return createLangInternalInvocation(cx, "escapeXMLAttribute", semtypes.STRING, []ast.BLangExpression{insert}, insert.GetPosition())
	case ast.XMLTemplateInsertionKindContent:
		return createLangInternalInvocation(cx, "escapeXMLContent", semtypes.STRING, []ast.BLangExpression{insert}, insert.GetPosition())
	default:
		cx.internalError("unexpected xml template insert kind")
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
			cx.internalError(err.Error())
			continue
		}
		uri, err := model.XMLNamespaceURI(symbol)
		if err != nil {
			cx.internalError(err.Error())
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

	args := expr.CallArgs()
	for i := range args {
		result := walkExpression(cx, args[i])
		initStmts = append(initStmts, result.initStmts...)
		args[i] = result.replacementNode.(ast.BLangExpression)
	}
	expr.SetCallArgs(args)

	if ast.IsStreamOperation(expr) {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}
	symbolRef, hasSymbol := invocationSymbol(expr)
	if !hasSymbol {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}
	fnSym, isDirectCall := cx.getSymbol(symbolRef).(model.FunctionSymbol)
	if !isDirectCall {
		return desugaredNode[ast.BLangActionOrExpression]{
			initStmts:       initStmts,
			replacementNode: expr,
		}
	}

	directStmts := walkDirectCallArgs(cx, expr, fnSym)
	initStmts = append(initStmts, directStmts...)

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func invocationSymbol(expr invocable) (model.SymbolRef, bool) {
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
		panic("unexpected")
	}
}

func walkDirectCallArgs(cx *functionContext, expr invocable, fnSym model.FunctionSymbol) []ast.StatementNode {
	sig := fnSym.Signature()
	totalParams := len(sig.ParamTypes)
	if totalParams == 0 {
		return nil
	}

	reordered := make([]ast.BLangExpression, totalParams)
	for i, arg := range expr.CallArgs() {
		switch arg := arg.(type) {
		case *ast.BLangNamedArgsExpression:
			for j, name := range sig.ParamNames {
				if name == arg.Name.Value {
					reordered[j] = arg.Expr
					break
				}
			}
		default:
			if i < totalParams {
				reordered[i] = arg
			} else {
				reordered = append(reordered, arg)
			}
		}
	}

	pos := expr.GetPosition()
	defaultableParams := fnSym.DefaultableParams()
	var initStmts []ast.StatementNode

	var transformed []ast.BLangExpression
	for i := range totalParams {
		if reordered[i] != nil {
			continue
		}
		dp, isDefaultable := defaultableParams.Get(i)
		if isDefaultable && dp.Kind == model.DefaultableParamKindInferredTypedesc {
			reordered[i] = synthesizeInferredTypedescArg(cx, sig.ParamTypes[i], pos)
			continue
		}
		for j := len(transformed); j < i; j++ {
			varDef, varRef := assignToLocal(cx, reordered[j], pos)
			initStmts = append(initStmts, varDef)
			reordered[j] = varRef
			transformed = append(transformed, varRef)
		}
		defaultInv := &ast.BLangInvocation{}
		defaultInv.Name = &ast.BLangIdentifier{Value: cx.pkgCtx.compilerCtx.GetSymbol(dp.Symbol).Name()}
		defaultInv.ArgExprs = reordered[:i]
		defaultInv.SetSymbol(dp.Symbol)
		defaultInv.SetDeterminedType(sig.ParamTypes[i])
		setPositionIfMissing(defaultInv, pos)

		varDef, varRef := assignToLocal(cx, defaultInv, pos)
		initStmts = append(initStmts, varDef)
		reordered[i] = varRef
		transformed = append(transformed, varRef)
	}

	expr.SetCallArgs(reordered)
	return initStmts
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

func assignToLocal(cx *functionContext, initExpr ast.BLangExpression, pos diagnostics.Location) (ast.StatementNode, *ast.BLangSimpleVarRef) {
	ty := initExpr.GetDeterminedType()
	tempName, tempSymRef := cx.addDesugardSymbol(ty, model.SymbolKindVariable, false)
	tempVar := &ast.BLangSimpleVariable{Name: &ast.BLangIdentifier{Value: tempName}}
	tempVar.SetDeterminedType(ty)
	tempVar.SetInitialExpression(initExpr)
	tempVar.SetSymbol(tempSymRef)
	varDef := &ast.BLangSimpleVariableDef{}
	varDef.SetVariable(tempVar)
	varDef.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(varDef, pos)

	varRef := &ast.BLangSimpleVarRef{VariableName: tempVar.Name}
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
	emptyList.AtomicType = semtypes.LIST_ATOMIC_INNER
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
	resultRef *ast.BLangSimpleVarRef,
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
	elemTy := semtypes.ListProj(tyCtx, spreadExpr.GetDeterminedType(), semtypes.INT)
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
	cond.SetDeterminedType(semtypes.BOOLEAN)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.NEVER)
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
	return desugarCheckedExpr(cx, expr, false)
}

func walkCheckPanickedExpr(cx *functionContext, expr *ast.BLangCheckPanickedExpr) desugaredNode[ast.BLangActionOrExpression] {
	return desugarCheckedExpr(cx, &expr.BLangCheckedExpr, true)
}

func walkTrapExpr(cx *functionContext, expr *ast.BLangTrapExpr) desugaredNode[ast.BLangActionOrExpression] {
	result := walkExpression(cx, expr.Expr)
	if len(result.initStmts) > 0 {
		// I don't think this can ever happen but if it does we need to think about how to add these statements in to the
		// trap region in BIR gen
		cx.internalError("Init statements will be hoisted outside of trap region")
	}
	expr.Expr = result.replacementNode.(ast.BLangExpression)
	return desugaredNode[ast.BLangActionOrExpression]{initStmts: nil, replacementNode: expr}
}

func desugarCheckedExpr(cx *functionContext, expr *ast.BLangCheckedExpr, isPanic bool) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	// Walk the inner expression first
	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		initStmts = append(initStmts, result.initStmts...)
		expr.Expr = result.replacementNode
	}

	innerTy := expr.Expr.GetDeterminedType()
	resultTy := expr.GetDeterminedType()

	basePos := expr.Expr.GetPosition()

	// TODO: extract util to add definition and get reference
	// Create temp var: $desugar$N = <inner expr>
	tempName, tempSymbol := cx.addDesugardSymbol(innerTy, model.SymbolKindVariable, false)
	tempVarName := &ast.BLangIdentifier{Value: tempName}
	tempVarName.SetPosition(basePos)
	tempVar := &ast.BLangSimpleVariable{Name: tempVarName}
	tempVar.SetDeterminedType(innerTy)
	tempVar.SetInitialExpression(expr.Expr)
	tempVar.SetSymbol(tempSymbol)
	tempVar.SetPosition(basePos)
	tempVarDef := &ast.BLangSimpleVariableDef{Var: tempVar}
	tempVarDef.SetPosition(basePos)
	initStmts = append(initStmts, tempVarDef)

	// Type test: $desugar$N is error
	tempVarRefForTest := &ast.BLangSimpleVarRef{VariableName: tempVarName}
	tempVarRefForTest.SetSymbol(tempSymbol)
	tempVarRefForTest.SetDeterminedType(innerTy)
	tempVarRefForTest.SetPosition(basePos)

	typeTestExpr := &ast.BLangTypeTestExpr{}
	typeTestExpr.Expr = tempVarRefForTest
	typeTestExpr.Type = ast.TypeData{Type: semtypes.ERROR}
	typeTestExpr.SetDeterminedType(semtypes.BOOLEAN)
	typeTestExpr.SetPosition(basePos)

	// If body: return or panic
	tempVarRefForBody := &ast.BLangSimpleVarRef{VariableName: tempVarName}
	tempVarRefForBody.SetSymbol(tempSymbol)
	tempVarRefForBody.SetDeterminedType(innerTy)
	tempVarRefForBody.SetPosition(basePos)

	var bodyStmt ast.StatementNode
	if isPanic {
		panicStmt := &ast.BLangPanic{Expr: tempVarRefForBody}
		panicStmt.SetPosition(expr.GetPosition())
		bodyStmt = panicStmt
	} else {
		returnStmt := &ast.BLangReturn{Expr: tempVarRefForBody}
		returnStmt.SetPosition(basePos)
		bodyStmt = returnStmt
	}

	ifBody := ast.BLangBlockStmt{
		Stmts: []ast.StatementNode{bodyStmt},
	}
	ifBody.SetPosition(basePos)
	ifStmt := &ast.BLangIf{
		Expr: typeTestExpr,
		Body: ifBody,
	}
	ifStmt.SetPosition(basePos)
	initStmts = append(initStmts, ifStmt)

	// Replacement: var ref typed as non-error type
	replacementVarRef := &ast.BLangSimpleVarRef{VariableName: tempVarName}
	replacementVarRef.SetSymbol(tempSymbol)
	replacementVarRef.SetDeterminedType(resultTy)
	replacementVarRef.SetPosition(basePos)

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: replacementVarRef,
	}
}

func walkLambdaFunction(cx *functionContext, expr *ast.BLangLambdaFunction) desugaredNode[ast.BLangActionOrExpression] {
	// Desugar the function body
	if expr.Function != nil {
		expr.Function = desugarFunction(cx.pkgCtx, expr.Function)
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

	for i := range expr.ArgsExprs {
		result := walkExpression(cx, expr.ArgsExprs[i])
		initStmts = append(initStmts, result.initStmts...)
		expr.ArgsExprs[i] = result.replacementNode.(ast.BLangExpression)
	}

	// Fill in any defaultable init params the caller omitted. This mirrors what
	// walkDirectCallArgs does for regular function calls via walkInvocation.
	initStmts = append(initStmts, fillNewExprInitDefaults(cx, expr)...)

	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func fillNewExprInitDefaults(cx *functionContext, expr *ast.BLangNewExpression) []ast.StatementNode {
	if expr.ClassSymbol.IsEmpty() {
		return nil
	}
	classSym, ok := cx.getSymbol(expr.ClassSymbol).(model.ClassSymbol)
	if !ok {
		return nil
	}
	initRef, ok := classSym.MethodSymbol("init")
	if !ok {
		return nil
	}
	initFnSym, ok := cx.getSymbol(initRef).(model.FunctionSymbol)
	if !ok {
		return nil
	}
	sig := initFnSym.Signature()
	defaultableParams := initFnSym.DefaultableParams()
	if defaultableParams == nil {
		return nil
	}
	totalParams := len(sig.ParamTypes)
	if totalParams <= len(expr.ArgsExprs) {
		return nil
	}

	pos := expr.GetPosition()
	var initStmts []ast.StatementNode

	// Materialize original args into locals so the same node is not aliased into
	// multiple default-lambda invocations. Mirrors walkDirectCallArgs behaviour.
	originalLen := len(expr.ArgsExprs)
	for j := 0; j < originalLen; j++ {
		if _, ok := expr.ArgsExprs[j].(*ast.BLangSimpleVarRef); !ok {
			varDef, varRef := assignToLocal(cx, expr.ArgsExprs[j], pos)
			initStmts = append(initStmts, varDef)
			expr.ArgsExprs[j] = varRef
		}
	}

	for i := originalLen; i < totalParams; i++ {
		dp, ok := defaultableParams.Get(i)
		if !ok {
			break
		}
		if dp.Kind == model.DefaultableParamKindInferredTypedesc {
			tdExpr := synthesizeInferredTypedescArg(cx, sig.ParamTypes[i], pos)
			setPositionIfMissing(tdExpr, pos)
			varDef, varRef := assignToLocal(cx, tdExpr, pos)
			initStmts = append(initStmts, varDef)
			expr.ArgsExprs = append(expr.ArgsExprs, varRef)
			continue
		}
		defaultInv := &ast.BLangInvocation{}
		defaultInv.Name = &ast.BLangIdentifier{Value: cx.getSymbol(dp.Symbol).Name()}
		defaultInv.ArgExprs = append([]ast.BLangExpression(nil), expr.ArgsExprs[:i]...)
		defaultInv.SetSymbol(dp.Symbol)
		defaultInv.SetDeterminedType(sig.ParamTypes[i])
		setPositionIfMissing(defaultInv, pos)

		varDef, varRef := assignToLocal(cx, defaultInv, pos)
		initStmts = append(initStmts, varDef)
		expr.ArgsExprs = append(expr.ArgsExprs, varRef)
	}
	return initStmts
}

func walkMappingConstructorExpr(cx *functionContext, expr *ast.BLangMappingConstructorExpr) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode

	for _, field := range expr.Fields {
		kv := field.(*ast.BLangMappingKeyValueField)

		if kv.Key.Kind != ast.MappingKeyComputed {
			if varRef, ok := kv.Key.Expr.(*ast.BLangSimpleVarRef); ok {
				name := varRef.VariableName.Value
				lit := &ast.BLangLiteral{
					Value:         name,
					OriginalValue: name,
				}
				lit.SetPosition(varRef.GetPosition())
				lit.SetDeterminedType(semtypes.STRING)
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
	name, symbol := cx.addDesugardSymbol(ty, model.SymbolKindVariable, false)
	varName := &ast.BLangIdentifier{Value: name}
	tempVar := &ast.BLangSimpleVariable{Name: varName}
	tempVar.SetDeterminedType(ty)
	tempVar.SetInitialExpression(initExpr)
	tempVar.SetSymbol(symbol)
	varDef := &ast.BLangSimpleVariableDef{Var: tempVar}
	varDef.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(varDef, pos)
	return varName, symbol, append(initStmts, varDef)
}

func createNilResultVar(cx *functionContext, ty semtypes.SemType, pos diagnostics.Location, initStmts []ast.StatementNode) (*ast.BLangIdentifier, model.SymbolRef, []ast.StatementNode) {
	nilLit := &ast.BLangLiteral{Value: nil}
	nilLit.SetDeterminedType(semtypes.NIL)
	setPositionIfMissing(nilLit, pos)

	name, symbol := cx.addDesugardSymbol(ty, model.SymbolKindVariable, false)
	varName := &ast.BLangIdentifier{Value: name}
	tempVar := &ast.BLangSimpleVariable{Name: varName}
	tempVar.SetDeterminedType(ty)
	tempVar.SetInitialExpression(nilLit)
	tempVar.SetSymbol(symbol)
	varDef := &ast.BLangSimpleVariableDef{Var: tempVar}
	varDef.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(varDef, pos)
	return varName, symbol, append(initStmts, varDef)
}

func createNilTypeTest(varName *ast.BLangIdentifier, symbol model.SymbolRef, ty semtypes.SemType, pos diagnostics.Location) *ast.BLangTypeTestExpr {
	ref := createVarRef(varName, symbol, ty)
	typeTest := &ast.BLangTypeTestExpr{
		Expr: ref,
		Type: ast.TypeData{Type: semtypes.NIL},
	}
	typeTest.SetDeterminedType(semtypes.BOOLEAN)
	setPositionIfMissing(typeTest, pos)
	return typeTest
}

func createVarRef(varName *ast.BLangIdentifier, symbol model.SymbolRef, ty semtypes.SemType) *ast.BLangSimpleVarRef {
	ref := &ast.BLangSimpleVarRef{VariableName: varName}
	ref.SetSymbol(symbol)
	ref.SetDeterminedType(ty)
	return ref
}

func createResultAssignment(resultVarName *ast.BLangIdentifier, resultSymbol model.SymbolRef, resultTy semtypes.SemType, valueExpr ast.BLangExpression, pos diagnostics.Location) *ast.BLangAssignment {
	varRef := createVarRef(resultVarName, resultSymbol, resultTy)
	assign := &ast.BLangAssignment{
		VarRef: varRef,
		Expr:   valueExpr,
	}
	assign.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(assign, pos)
	return assign
}
