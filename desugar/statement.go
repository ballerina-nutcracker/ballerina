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
	"ballerina-lang-go/ast"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
)

func walkStatement(cx *functionContext, node ast.StatementNode) desugaredNode[ast.StatementNode] {
	switch stmt := node.(type) {
	case *ast.BLangBlockStmt:
		return walkBlockStmt(cx, stmt)
	case *ast.BLangAssignment:
		return walkAssignment(cx, stmt)
	case *ast.BLangCompoundAssignment:
		return walkCompoundAssignment(cx, stmt)
	case *ast.BLangExpressionStmt:
		return walkExpressionStmt(cx, stmt)
	case *ast.BLangIf:
		return walkIf(cx, stmt)
	case *ast.BLangWhile:
		return walkWhile(cx, stmt)
	case *ast.BLangDo:
		return walkDo(cx, stmt)
	case *ast.BLangLock:
		return walkLock(cx, stmt)
	case *ast.BLangForeach:
		return visitForEach(cx, stmt)
	case *ast.BLangSimpleVariableDef:
		return walkSimpleVariableDef(cx, stmt)
	case *ast.BLangReturn:
		return walkReturn(cx, stmt)
	case *ast.BLangPanic:
		return walkPanic(cx, stmt)
	case *ast.BLangBreak:
		return desugaredNode[ast.StatementNode]{replacementNode: stmt}
	case *ast.BLangContinue:
		return walkContinue(cx, stmt)
	case *ast.BLangMatchStatement:
		return walkMatchStatement(cx, stmt)
	case *ast.BLangXMLNS:
		return desugaredNode[ast.StatementNode]{replacementNode: stmt}
	default:
		panic("unexpected statement type")
	}
}

func walkBlockStmt(cx *functionContext, stmt *ast.BLangBlockStmt) desugaredNode[ast.StatementNode] {
	var allStmts []ast.StatementNode

	for _, childStmt := range stmt.Stmts {
		result := walkStatement(cx, childStmt)
		allStmts = append(allStmts, result.initStmts...)
		allStmts = append(allStmts, result.replacementNode)
	}

	stmt.Stmts = allStmts
	return desugaredNode[ast.StatementNode]{replacementNode: stmt}
}

func walkBlockFunctionBody(cx *functionContext, body *ast.BLangBlockFunctionBody) {
	var allStmts []ast.StatementNode

	for _, stmt := range body.Stmts {
		result := walkStatement(cx, stmt)
		allStmts = append(allStmts, result.initStmts...)
		allStmts = append(allStmts, result.replacementNode)
	}

	body.Stmts = allStmts
}

func walkAssignment(cx *functionContext, stmt *ast.BLangAssignment) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.VarRef != nil {
		result := walkExpression(cx, stmt.VarRef)
		initStmts = append(initStmts, result.initStmts...)
		stmt.VarRef = result.replacementNode.(ast.LExpr)
	}

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func walkCompoundAssignment(cx *functionContext, stmt *ast.BLangCompoundAssignment) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.VarRef != nil {
		result := walkExpression(cx, stmt.VarRef)
		initStmts = append(initStmts, result.initStmts...)
		stmt.VarRef = result.replacementNode.(ast.LExpr)
	}

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func walkExpressionStmt(cx *functionContext, stmt *ast.BLangExpressionStmt) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func walkIf(cx *functionContext, stmt *ast.BLangIf) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode.(ast.BLangExpression)
	}

	// Push if scope before visiting body
	cx.pushScope(stmt.Scope())
	bodyResult := walkBlockStmt(cx, &stmt.Body)
	stmt.Body = *bodyResult.replacementNode.(*ast.BLangBlockStmt)
	cx.popScope()

	if stmt.ElseStmt != nil {
		elseResult := walkStatement(cx, stmt.ElseStmt)
		if len(elseResult.initStmts) > 0 {
			elseBlock := &ast.BLangBlockStmt{
				Stmts: append(elseResult.initStmts, elseResult.replacementNode),
			}
			elseBlock.SetPosition(stmt.GetPosition())
			stmt.ElseStmt = elseBlock
		} else {
			stmt.ElseStmt = elseResult.replacementNode
		}
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func walkWhile(cx *functionContext, stmt *ast.BLangWhile) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode.(ast.BLangExpression)
	}

	// Push nil to loopVarStack to indicate this is a native while (not desugared foreach)
	cx.pushLoopVar(nil)
	// Push while scope before visiting body
	cx.pushScope(stmt.Scope())
	bodyResult := walkBlockStmt(cx, &stmt.Body)
	stmt.Body = *bodyResult.replacementNode.(*ast.BLangBlockStmt)
	cx.popScope()
	cx.popLoopVar()

	// Only walk onFail clause if it has a body
	if stmt.OnFailClause.Body != nil {
		onFailResult := walkOnFailClause(cx, &stmt.OnFailClause)
		stmt.OnFailClause = *onFailResult.replacementNode.(*ast.BLangOnFailClause)
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func walkLock(cx *functionContext, stmt *ast.BLangLock) desugaredNode[ast.StatementNode] {
	bodyResult := walkBlockStmt(cx, &stmt.Body)
	stmt.Body = *bodyResult.replacementNode.(*ast.BLangBlockStmt)
	return desugaredNode[ast.StatementNode]{replacementNode: stmt}
}

func walkDo(cx *functionContext, stmt *ast.BLangDo) desugaredNode[ast.StatementNode] {
	bodyResult := walkBlockStmt(cx, &stmt.Body)
	stmt.Body = *bodyResult.replacementNode.(*ast.BLangBlockStmt)

	// Only walk onFail clause if it has a body
	if stmt.OnFailClause.Body != nil {
		onFailResult := walkOnFailClause(cx, &stmt.OnFailClause)
		stmt.OnFailClause = *onFailResult.replacementNode.(*ast.BLangOnFailClause)
	}

	return desugaredNode[ast.StatementNode]{
		replacementNode: stmt,
	}
}

func walkOnFailClause(cx *functionContext, clause *ast.BLangOnFailClause) desugaredNode[ast.StatementNode] {
	bodyResult := walkBlockStmt(cx, clause.Body)
	clause.Body = bodyResult.replacementNode.(*ast.BLangBlockStmt)

	return desugaredNode[ast.StatementNode]{
		replacementNode: clause,
	}
}

func walkSimpleVariableDef(cx *functionContext, stmt *ast.BLangSimpleVariableDef) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.Var != nil {
		if typeNode := stmt.Var.TypeNode(); typeNode != nil {
			result := desugarTypeDesc(cx, typeNode, cx.currentScope())
			for i := range result.functions {
				result.functions[i] = desugarFunction(cx.pkgCtx, result.functions[i])
			}
			cx.pkgCtx.addLocalTypeDescDefaultFunctions(result.functions)
			for _, field := range result.recordFields {
				initStmts = append(initStmts, desugarRecordFieldDefault(cx, field))
			}
		} else {
			cx.pkgCtx.addDefaultClosureOwner(stmt.Var.Expr)
		}
		if stmt.Var.Expr != nil {
			result := walkExpression(cx, stmt.Var.Expr)
			initStmts = append(initStmts, result.initStmts...)
			stmt.Var.Expr = result.replacementNode
		}
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func walkPanic(cx *functionContext, stmt *ast.BLangPanic) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode.(ast.BLangExpression)
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func walkReturn(cx *functionContext, stmt *ast.BLangReturn) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}

func createIncrementStmt(loopVar ast.LExpr) *ast.BLangAssignment {
	basePos := loopVar.GetPosition()

	oneLiteral := &ast.BLangNumericLiteral{
		BLangLiteral: ast.BLangLiteral{
			Value:         int64(1),
			OriginalValue: "1",
		},
		Kind: ast.NodeKind_NUMERIC_LITERAL,
	}
	oneLiteral.SetDeterminedType(semtypes.INT)
	addExpr := &ast.BLangBinaryExpr{
		LhsExpr: loopVar,
		RhsExpr: oneLiteral,
		OpKind:  model.OperatorKind_ADD,
	}
	addExpr.SetDeterminedType(semtypes.INT)
	incrementStmt := &ast.BLangAssignment{
		VarRef: loopVar,
		Expr:   addExpr,
	}
	incrementStmt.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(incrementStmt, basePos)
	return incrementStmt
}

func walkContinue(cx *functionContext, stmt *ast.BLangContinue) desugaredNode[ast.StatementNode] {
	// Check if we're in a desugared foreach (has a loop variable)
	loopVar := cx.currentLoopVar()
	if loopVar != nil {
		// For desugared foreach, we need to add increment before continue
		incrementStmt := createIncrementStmt(loopVar)

		// Return increment as initStmts and continue as replacement
		return desugaredNode[ast.StatementNode]{
			initStmts:       []ast.StatementNode{incrementStmt},
			replacementNode: stmt,
		}
	}

	// For native while loops, continue as-is
	return desugaredNode[ast.StatementNode]{
		initStmts:       []ast.StatementNode{},
		replacementNode: stmt,
	}
}

func visitForEach(cx *functionContext, stmt *ast.BLangForeach) desugaredNode[ast.StatementNode] {
	cx.pushScope(stmt.Scope())
	defer cx.popScope()
	if isRangeExpr(stmt.Collection) {
		rangeExpr := stmt.Collection.(*ast.BLangBinaryExpr)
		return desugarForEachOnRange(cx, rangeExpr, stmt.VariableDef, &stmt.Body, stmt.Scope())
	}
	tyCtx := semtypes.ContextFrom(cx.typeEnv())
	if semtypes.IsSubtype(tyCtx, stmt.Collection.GetDeterminedType(), semtypes.LIST) {
		return desugarForEachOnList(cx, stmt.Collection, stmt.VariableDef, &stmt.Body, stmt.Scope())
	}
	if semtypes.IsSubtype(tyCtx, stmt.Collection.GetDeterminedType(), semtypes.MAPPING) {
		return desugarForEachOnMap(cx, stmt.Collection, stmt.VariableDef, &stmt.Body, stmt.Scope())
	}
	return desugarForEachOnIterable(cx, stmt.Collection, stmt.VariableDef, &stmt.Body, stmt.Scope())
}

func desugarForEachOnList(cx *functionContext, collection ast.BLangActionOrExpression, loopVarDef *ast.BLangSimpleVariableDef, body *ast.BLangBlockStmt, foreachScope model.Scope) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	basePos := collection.GetPosition()

	// Step 1: evaluate collection once into a temp variable
	collResult := walkExpression(cx, collection)
	initStmts = append(initStmts, collResult.initStmts...)
	collExpr := collResult.replacementNode

	collType := collExpr.GetDeterminedType()
	collName, collVarSymbol := cx.addDesugardSymbol(collType, model.SymbolKindVariable, false, basePos)
	collVarName := &ast.BLangIdentifier{Value: collName}
	collVar := &ast.BLangSimpleVariable{Name: collVarName}
	collVar.SetDeterminedType(collType)
	collVar.SetInitialExpression(collExpr)
	collVar.SetSymbol(collVarSymbol)
	collVarDef := &ast.BLangSimpleVariableDef{Var: collVar}
	setPositionIfMissing(collVarDef, basePos)
	initStmts = append(initStmts, collVarDef)

	collVarRef := &ast.BLangSimpleVarRef{VariableName: collVarName}
	collVarRef.SetSymbol(collVarSymbol)
	collVarRef.SetDeterminedType(collType)

	// Step 2: index variable ($desugar$N = 0)
	zeroLiteral := &ast.BLangNumericLiteral{
		BLangLiteral: ast.BLangLiteral{
			Value:         int64(0),
			OriginalValue: "0",
		},
		Kind: ast.NodeKind_NUMERIC_LITERAL,
	}
	zeroLiteral.SetDeterminedType(semtypes.INT)

	idxName, idxVarSymbol := cx.addDesugardSymbol(semtypes.INT, model.SymbolKindVariable, false, basePos)
	idxVarName := &ast.BLangIdentifier{Value: idxName}
	idxVar := &ast.BLangSimpleVariable{Name: idxVarName}
	idxVar.SetDeterminedType(semtypes.INT)
	idxVar.SetInitialExpression(zeroLiteral)
	idxVar.SetSymbol(idxVarSymbol)
	idxVarDef := &ast.BLangSimpleVariableDef{Var: idxVar}
	setPositionIfMissing(idxVarDef, basePos)
	initStmts = append(initStmts, idxVarDef)

	idxVarRef := &ast.BLangSimpleVarRef{VariableName: idxVarName}
	idxVarRef.SetSymbol(idxVarSymbol)
	idxVarRef.SetDeterminedType(semtypes.INT)

	// Step 3: length variable ($desugar$M = length(collVar))
	lengthInvocation := createLengthInvocation(cx, collVarRef)

	lenName, lenVarSymbol := cx.addDesugardSymbol(semtypes.INT, model.SymbolKindVariable, false, basePos)
	lenVarName := &ast.BLangIdentifier{Value: lenName}
	lenVar := &ast.BLangSimpleVariable{Name: lenVarName}
	lenVar.SetDeterminedType(semtypes.INT)
	lenVar.SetInitialExpression(lengthInvocation)
	lenVar.SetSymbol(lenVarSymbol)
	lenVarDef := &ast.BLangSimpleVariableDef{Var: lenVar}
	setPositionIfMissing(lenVarDef, basePos)
	initStmts = append(initStmts, lenVarDef)

	lenVarRef := &ast.BLangSimpleVarRef{VariableName: lenVarName}
	lenVarRef.SetSymbol(lenVarSymbol)
	lenVarRef.SetDeterminedType(semtypes.INT)

	// Step 4: while condition ($idx < $len)
	whileCondition := &ast.BLangBinaryExpr{
		LhsExpr: idxVarRef,
		RhsExpr: lenVarRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	whileCondition.SetDeterminedType(semtypes.BOOLEAN)

	// Step 5: element access (collVar[$idx])
	elementAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: idxVarRef,
	}
	elementAccess.Expr = collVarRef
	elementAccess.SetDeterminedType(loopVarDef.Var.GetDeterminedType())

	// Step 6: patch loop var def initial expression
	loopVarDef.Var.SetInitialExpression(elementAccess)

	// Step 7: build body
	incrementStmt := createIncrementStmt(idxVarRef)
	cx.pushLoopVar(idxVarRef)

	newBodyStmts := make([]ast.StatementNode, 0, len(body.Stmts)+2)
	newBodyStmts = append(newBodyStmts, loopVarDef)
	newBodyStmts = append(newBodyStmts, body.Stmts...)
	if len(newBodyStmts) > 0 {
		if isAppendReachable(newBodyStmts[len(newBodyStmts)-1]) {
			newBodyStmts = append(newBodyStmts, incrementStmt)
		}
	}
	body.Stmts = newBodyStmts

	bodyResult := walkBlockStmt(cx, body)
	newBody := bodyResult.replacementNode.(*ast.BLangBlockStmt)

	cx.popLoopVar()

	// Step 8: create while loop
	whileStmt := &ast.BLangWhile{
		Expr: whileCondition,
		Body: *newBody,
	}
	whileStmt.SetScope(foreachScope)
	whileStmt.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(whileStmt, basePos)

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: whileStmt,
	}
}

func createLengthInvocation(cx *functionContext, collection ast.BLangExpression) *ast.BLangInvocation {
	pkgName := "lang.array"
	space, ok := cx.getImportedSymbolSpace(pkgName)
	if !ok {
		cx.internalError(pkgName + " symbol space not found")
		return nil
	}
	symbolRef, ok := space.GetSymbol("length")
	if !ok {
		cx.internalError(pkgName + ":length symbol not found")
		return nil
	}
	basePos := collection.GetPosition()

	orgIdent := &ast.BLangIdentifier{Value: "ballerina"}
	pkgLangIdent := ast.BLangIdentifier{Value: "lang"}
	pkgArrayIdent := ast.BLangIdentifier{Value: "array"}
	aliasIdent := &ast.BLangIdentifier{Value: pkgName}

	imp := ast.BLangImportPackage{
		OrgName:      orgIdent,
		PkgNameComps: []ast.BLangIdentifier{pkgLangIdent, pkgArrayIdent},
		Alias:        aliasIdent,
	}
	setPositionIfMissing(&imp, basePos)

	cx.addImplicitImport(pkgName, imp)

	nameIdent := &ast.BLangIdentifier{Value: "length"}
	pkgAliasIdent := &ast.BLangIdentifier{Value: pkgName}

	inv := &ast.BLangInvocation{PkgAlias: pkgAliasIdent}
	inv.Name = nameIdent
	inv.ArgExprs = []ast.BLangExpression{collection}
	inv.SetSymbol(symbolRef)
	inv.SetDeterminedType(semtypes.INT)
	setPositionIfMissing(inv, basePos)
	return inv
}

func desugarForEachOnMap(cx *functionContext, collection ast.BLangActionOrExpression, loopVarDef *ast.BLangSimpleVariableDef, body *ast.BLangBlockStmt, foreachScope model.Scope) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	basePos := collection.GetPosition()

	// Step 1: evaluate collection once into a temp variable
	collResult := walkExpression(cx, collection)
	initStmts = append(initStmts, collResult.initStmts...)
	collExpr := collResult.replacementNode

	collType := collExpr.GetDeterminedType()
	collName, collVarSymbol := cx.addDesugardSymbol(collType, model.SymbolKindVariable, false, basePos)
	collVarName := &ast.BLangIdentifier{Value: collName}
	collVar := &ast.BLangSimpleVariable{Name: collVarName}
	collVar.SetDeterminedType(collType)
	collVar.SetInitialExpression(collExpr)
	collVar.SetSymbol(collVarSymbol)
	collVarDef := &ast.BLangSimpleVariableDef{Var: collVar}
	setPositionIfMissing(collVarDef, basePos)
	initStmts = append(initStmts, collVarDef)

	collVarRef := &ast.BLangSimpleVarRef{VariableName: collVarName}
	collVarRef.SetSymbol(collVarSymbol)
	collVarRef.SetDeterminedType(collType)

	// Step 2: keys variable ($desugar$N = lang.map:keys(collVar))
	keysInvocation := createKeysInvocation(cx, collVarRef)
	keysType := keysInvocation.GetDeterminedType()

	keysName, keysVarSymbol := cx.addDesugardSymbol(keysType, model.SymbolKindVariable, false, basePos)
	keysVarName := &ast.BLangIdentifier{Value: keysName}
	keysVar := &ast.BLangSimpleVariable{Name: keysVarName}
	keysVar.SetDeterminedType(keysType)
	keysVar.SetInitialExpression(keysInvocation)
	keysVar.SetSymbol(keysVarSymbol)
	keysVarDef := &ast.BLangSimpleVariableDef{Var: keysVar}
	setPositionIfMissing(keysVarDef, basePos)
	initStmts = append(initStmts, keysVarDef)

	keysVarRef := &ast.BLangSimpleVarRef{VariableName: keysVarName}
	keysVarRef.SetSymbol(keysVarSymbol)
	keysVarRef.SetDeterminedType(keysType)

	// Step 3: index variable ($desugar$N = 0)
	zeroLiteral := &ast.BLangNumericLiteral{
		BLangLiteral: ast.BLangLiteral{
			Value:         int64(0),
			OriginalValue: "0",
		},
		Kind: ast.NodeKind_NUMERIC_LITERAL,
	}
	zeroLiteral.SetDeterminedType(semtypes.INT)

	idxName, idxVarSymbol := cx.addDesugardSymbol(semtypes.INT, model.SymbolKindVariable, false, basePos)
	idxVarName := &ast.BLangIdentifier{Value: idxName}
	idxVar := &ast.BLangSimpleVariable{Name: idxVarName}
	idxVar.SetDeterminedType(semtypes.INT)
	idxVar.SetInitialExpression(zeroLiteral)
	idxVar.SetSymbol(idxVarSymbol)
	idxVarDef := &ast.BLangSimpleVariableDef{Var: idxVar}
	setPositionIfMissing(idxVarDef, basePos)
	initStmts = append(initStmts, idxVarDef)

	idxVarRef := &ast.BLangSimpleVarRef{VariableName: idxVarName}
	idxVarRef.SetSymbol(idxVarSymbol)
	idxVarRef.SetDeterminedType(semtypes.INT)

	// Step 4: length variable ($desugar$N = lang.array:length(keysVar))
	lengthInvocation := createLengthInvocation(cx, keysVarRef)

	lenName, lenVarSymbol := cx.addDesugardSymbol(semtypes.INT, model.SymbolKindVariable, false, basePos)
	lenVarName := &ast.BLangIdentifier{Value: lenName}
	lenVar := &ast.BLangSimpleVariable{Name: lenVarName}
	lenVar.SetDeterminedType(semtypes.INT)
	lenVar.SetInitialExpression(lengthInvocation)
	lenVar.SetSymbol(lenVarSymbol)
	lenVarDef := &ast.BLangSimpleVariableDef{Var: lenVar}
	setPositionIfMissing(lenVarDef, basePos)
	initStmts = append(initStmts, lenVarDef)

	lenVarRef := &ast.BLangSimpleVarRef{VariableName: lenVarName}
	lenVarRef.SetSymbol(lenVarSymbol)
	lenVarRef.SetDeterminedType(semtypes.INT)

	// Step 5: while condition ($idx < $len)
	whileCondition := &ast.BLangBinaryExpr{
		LhsExpr: idxVarRef,
		RhsExpr: lenVarRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	whileCondition.SetDeterminedType(semtypes.BOOLEAN)

	// Step 6: key access (keysVar[$idx]) then map access (collVar[key])
	keyAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: idxVarRef,
	}
	keyAccess.Expr = keysVarRef
	keyAccess.SetDeterminedType(semtypes.STRING)

	mapAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: keyAccess,
	}
	mapAccess.Expr = collVarRef
	mapAccess.SetDeterminedType(loopVarDef.Var.GetDeterminedType())

	// Step 7: patch loop var def initial expression
	loopVarDef.Var.SetInitialExpression(mapAccess)

	// Step 8: build body
	incrementStmt := createIncrementStmt(idxVarRef)
	cx.pushLoopVar(idxVarRef)

	newBodyStmts := make([]ast.StatementNode, 0, len(body.Stmts)+2)
	newBodyStmts = append(newBodyStmts, loopVarDef)
	newBodyStmts = append(newBodyStmts, body.Stmts...)
	if len(newBodyStmts) > 0 {
		if isAppendReachable(newBodyStmts[len(newBodyStmts)-1]) {
			newBodyStmts = append(newBodyStmts, incrementStmt)
		}
	}
	body.Stmts = newBodyStmts

	bodyResult := walkBlockStmt(cx, body)
	newBody := bodyResult.replacementNode.(*ast.BLangBlockStmt)

	cx.popLoopVar()

	// Step 9: create while loop
	whileStmt := &ast.BLangWhile{
		Expr: whileCondition,
		Body: *newBody,
	}
	whileStmt.SetScope(foreachScope)
	whileStmt.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(whileStmt, basePos)

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: whileStmt,
	}
}

func createKeysInvocation(cx *functionContext, collection ast.BLangExpression) *ast.BLangInvocation {
	pkgName := "lang.map"
	space, ok := cx.getImportedSymbolSpace(pkgName)
	if !ok {
		cx.internalError(pkgName + " symbol space not found")
		return nil
	}
	symbolRef, ok := space.GetSymbol("keys")
	if !ok {
		cx.internalError(pkgName + ":keys symbol not found")
		return nil
	}
	fnSymbol := cx.getSymbol(symbolRef).(model.FunctionSymbol)
	returnType := fnSymbol.TypedSignature().ReturnType
	cx.addImplicitImport(pkgName, ast.BLangImportPackage{
		OrgName:      &ast.BLangIdentifier{Value: "ballerina"},
		PkgNameComps: []ast.BLangIdentifier{{Value: "lang"}, {Value: "map"}},
		Alias:        &ast.BLangIdentifier{Value: pkgName},
	})
	inv := &ast.BLangInvocation{PkgAlias: &ast.BLangIdentifier{Value: pkgName}}
	inv.Name = &ast.BLangIdentifier{Value: "keys"}
	inv.ArgExprs = []ast.BLangExpression{collection}
	inv.SetSymbol(symbolRef)
	inv.SetDeterminedType(returnType)
	return inv
}

func desugarForEachOnRange(cx *functionContext, rangeExpr *ast.BLangBinaryExpr, loopVarDef *ast.BLangSimpleVariableDef, body *ast.BLangBlockStmt, foreachScope model.Scope) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	basePos := rangeExpr.GetPosition()

	startResult := walkExpression(cx, rangeExpr.LhsExpr)
	initStmts = append(initStmts, startResult.initStmts...)
	startExpr := startResult.replacementNode

	endResult := walkExpression(cx, rangeExpr.RhsExpr)
	initStmts = append(initStmts, endResult.initStmts...)
	endExpr := endResult.replacementNode

	loopVarDef.Var.SetInitialExpression(startExpr)
	initStmts = append(initStmts, loopVarDef)

	loopVarRef := &ast.BLangSimpleVarRef{
		VariableName: loopVarDef.Var.Name,
	}
	loopVarRef.SetSymbol(loopVarDef.Var.Symbol())

	endName, endVarSymbol := cx.addDesugardSymbol(semtypes.INT, model.SymbolKindVariable, false, basePos)
	endVarName := &ast.BLangIdentifier{Value: endName}
	endVar := &ast.BLangSimpleVariable{Name: endVarName}
	endVar.SetDeterminedType(semtypes.INT)
	endVar.SetInitialExpression(endExpr)
	endVar.SetSymbol(endVarSymbol)

	endVarDef := &ast.BLangSimpleVariableDef{
		Var: endVar,
	}
	setPositionIfMissing(endVarDef, basePos)
	initStmts = append(initStmts, endVarDef)

	endVarRef := &ast.BLangSimpleVarRef{
		VariableName: endVarName,
	}
	endVarRef.SetSymbol(endVarSymbol)

	var compOp model.OperatorKind
	if rangeExpr.GetOperatorKind() == model.OperatorKind_CLOSED_RANGE {
		compOp = model.OperatorKind_LESS_EQUAL // <= for closed range
	} else {
		compOp = model.OperatorKind_LESS_THAN // < for half-open range
	}

	whileCondition := &ast.BLangBinaryExpr{
		LhsExpr: loopVarRef,
		RhsExpr: endVarRef,
		OpKind:  compOp,
	}
	whileCondition.SetDeterminedType(semtypes.BOOLEAN)

	incrementStmt := createIncrementStmt(loopVarRef)

	// Note: foreach scope is already pushed by visitForEach at the top level
	cx.pushLoopVar(loopVarRef)

	newBodyStmts := make([]ast.StatementNode, len(body.Stmts))
	copy(newBodyStmts, body.Stmts)
	if len(newBodyStmts) > 0 {
		if isAppendReachable(newBodyStmts[len(newBodyStmts)-1]) {
			newBodyStmts = append(newBodyStmts, incrementStmt)
		}
	} else {
		// just replace it with a no-op
		emptyBlock := &ast.BLangBlockStmt{}
		setPositionIfMissing(emptyBlock, basePos)
		return desugaredNode[ast.StatementNode]{
			replacementNode: emptyBlock,
		}
	}
	body.Stmts = newBodyStmts

	bodyResult := walkBlockStmt(cx, body)
	newBody := bodyResult.replacementNode.(*ast.BLangBlockStmt)

	cx.popLoopVar()

	// 10. Create the while loop using the foreach scope
	whileStmt := &ast.BLangWhile{
		Expr: whileCondition,
		Body: *newBody,
	}
	whileStmt.SetScope(foreachScope)
	whileStmt.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(whileStmt, basePos)

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: whileStmt,
	}
}

// TODO: do we need to think about if-else here as well?
// If the last statement in a block is something like panic, return, continue or break, then we shouldn't append
// nodes after that. I would make that node unreacheable. We need to make sure desugared AST is still valid.
func isAppendReachable(stmt ast.StatementNode) bool {
	switch stmt := stmt.(type) {
	case *ast.BLangReturn, *ast.BLangContinue, *ast.BLangBreak, *ast.BLangPanic:
		return false
	case *ast.BLangBlockStmt:
		if len(stmt.Stmts) == 0 {
			return true
		}
		lastChild := stmt.Stmts[len(stmt.Stmts)-1]
		return isAppendReachable(lastChild)
	default:
		return true
	}
}

func isRangeExpr(expr ast.BLangActionOrExpression) bool {
	if binaryExpr, ok := expr.(*ast.BLangBinaryExpr); ok {
		switch binaryExpr.GetOperatorKind() {
		case model.OperatorKind_CLOSED_RANGE, model.OperatorKind_HALF_OPEN_RANGE:
			return true
		default:
			return false
		}
	}
	return false
}

func createIteratorInvocation(cx *functionContext, receiver ast.BLangExpression, receiverType semtypes.SemType, pos diagnostics.Location) *ast.BLangInvocation {
	if semtypes.IsSubtype(cx.typeCtx(), receiverType, semtypes.XML) {
		return createXMLIteratorInvocation(cx, receiver, receiverType)
	}
	return createMethodInvocation(cx, receiver, "iterator", receiverType, []ast.BLangExpression{}, pos)
}

func createXMLIteratorInvocation(cx *functionContext, receiver ast.BLangExpression, receiverType semtypes.SemType) *ast.BLangInvocation {
	pkgName := "lang.xml"
	space, ok := cx.pkgCtx.getImportedSymbolSpace(pkgName)
	if !ok {
		cx.pkgCtx.internalError(pkgName + " symbol space not found")
		return nil
	}
	iteratorRef, ok := space.GetSymbol("iterator")
	if !ok {
		cx.pkgCtx.internalError(pkgName + ":iterator symbol not found")
		return nil
	}
	cx.pkgCtx.addImplicitImport(pkgName, ast.BLangImportPackage{
		OrgName:      &ast.BLangIdentifier{Value: "ballerina"},
		PkgNameComps: []ast.BLangIdentifier{{Value: "lang"}, {Value: "xml"}},
		Alias:        &ast.BLangIdentifier{Value: pkgName},
	})
	inv := &ast.BLangInvocation{PkgAlias: &ast.BLangIdentifier{Value: pkgName}}
	inv.Name = &ast.BLangIdentifier{Value: "iterator"}
	inv.ArgExprs = []ast.BLangExpression{receiver}
	inv.SetSymbol(iteratorRef)
	inv.SetDeterminedType(cx.pkgCtx.xmlIteratorType(semtypes.XMLItemType(receiverType)))
	inv.SetPosition(receiver.GetPosition())
	return inv
}

func (ctx *packageContext) xmlIteratorType(itemTy semtypes.SemType) semtypes.SemType {
	return ctx.xmlIteratorTypes.GetOrBuild(itemTy, func() semtypes.SemType {
		return buildXMLIteratorType(ctx.typeEnv(), itemTy)
	})
}

func buildXMLIteratorType(env semtypes.Env, itemTy semtypes.SemType) semtypes.SemType {
	recordDef := semtypes.NewMappingDefinition()
	recordTy := recordDef.DefineMappingTypeWrapped(env,
		[]semtypes.Field{semtypes.FieldFrom("value", itemTy, false, false)},
		semtypes.NEVER)
	nextReturnTy := semtypes.Union(recordTy, semtypes.NIL)
	ld := semtypes.NewListDefinition()
	emptyParams := ld.DefineListTypeWrapped(env, nil, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	fd := semtypes.NewFunctionDefinition()
	nextFnTy := fd.Define(env, emptyParams, nextReturnTy, semtypes.FunctionQualifiersFrom(env, true, false))
	iterOd := semtypes.NewObjectDefinition()
	return iterOd.Define(env, semtypes.ObjectQualifiersDEFAULT, []semtypes.Member{{
		Name:       "next",
		ValueTy:    nextFnTy,
		Kind:       semtypes.MemberKindMethod,
		Visibility: semtypes.VisibilityPublic,
		Immutable:  true,
	}})
}

func createMethodInvocation(cx *functionContext, receiver ast.BLangExpression, methodName string, receiverType semtypes.SemType, args []ast.BLangExpression, pos diagnostics.Location) *ast.BLangInvocation {
	tyCtx := cx.typeCtx()
	fnTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst(methodName), receiverType)

	argTys := make([]semtypes.SemType, len(args))
	for i, arg := range args {
		argTys[i] = arg.GetDeterminedType()
	}
	ld := semtypes.NewListDefinition()
	paramList := ld.DefineListTypeWrapped(cx.typeEnv(), argTys, len(argTys), semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	retTy := semtypes.FunctionReturnType(tyCtx, fnTy, paramList)

	_, fnSymRef := cx.addDesugardSymbol(fnTy, model.SymbolKindFunction, false, pos)

	inv := &ast.BLangInvocation{}
	inv.Name = &ast.BLangIdentifier{Value: methodName}
	inv.Expr = receiver
	inv.ArgExprs = args
	inv.SetSymbol(fnSymRef)
	inv.SetDeterminedType(retTy)
	return inv
}

func desugarForEachOnIterable(cx *functionContext, collection ast.BLangActionOrExpression, loopVarDef *ast.BLangSimpleVariableDef, body *ast.BLangBlockStmt, foreachScope model.Scope) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode
	basePos := collection.GetPosition()
	tyCtx := cx.typeCtx()

	// Step 1: Evaluate collection into temp var
	collResult := walkExpression(cx, collection)
	initStmts = append(initStmts, collResult.initStmts...)
	collExpr := collResult.replacementNode

	collType := collExpr.GetDeterminedType()
	collName, collSymbol := cx.addDesugardSymbol(collType, model.SymbolKindVariable, false, basePos)
	collVarName := &ast.BLangIdentifier{Value: collName}
	collVar := &ast.BLangSimpleVariable{Name: collVarName}
	collVar.SetDeterminedType(collType)
	collVar.SetInitialExpression(collExpr)
	collVar.SetSymbol(collSymbol)
	collVarDef := &ast.BLangSimpleVariableDef{Var: collVar}
	setPositionIfMissing(collVarDef, basePos)
	initStmts = append(initStmts, collVarDef)

	collVarRef := &ast.BLangSimpleVarRef{VariableName: collVarName}
	collVarRef.SetSymbol(collSymbol)
	collVarRef.SetDeterminedType(collType)

	// Step 2: Create iterator = collection.iterator()
	iteratorInv := createIteratorInvocation(cx, collVarRef, collType, basePos)
	iteratorType := iteratorInv.GetDeterminedType()

	iterName, iterSymbol := cx.addDesugardSymbol(iteratorType, model.SymbolKindVariable, false, basePos)
	iterVarName := &ast.BLangIdentifier{Value: iterName}
	iterVar := &ast.BLangSimpleVariable{Name: iterVarName}
	iterVar.SetDeterminedType(iteratorType)
	iterVar.SetInitialExpression(iteratorInv)
	iterVar.SetSymbol(iterSymbol)
	iterVarDef := &ast.BLangSimpleVariableDef{Var: iterVar}
	setPositionIfMissing(iterVarDef, basePos)
	initStmts = append(initStmts, iterVarDef)

	// Step 3: while(true) condition
	trueLiteral := &ast.BLangLiteral{Value: true, OriginalValue: "true"}
	trueLiteral.SetDeterminedType(semtypes.BOOLEAN)

	// Step 4: Build while body
	var whileBodyStmts []ast.StatementNode

	// 4a: $next = $iterator.next()
	iterVarRef := &ast.BLangSimpleVarRef{VariableName: iterVarName}
	iterVarRef.SetSymbol(iterSymbol)
	iterVarRef.SetDeterminedType(iteratorType)

	nextInv := createMethodInvocation(cx, iterVarRef, "next", iteratorType, []ast.BLangExpression{}, basePos)
	nextReturnType := nextInv.GetDeterminedType()

	nextName, nextSymbol := cx.addDesugardSymbol(nextReturnType, model.SymbolKindVariable, false, basePos)
	nextVarName := &ast.BLangIdentifier{Value: nextName}
	nextVar := &ast.BLangSimpleVariable{Name: nextVarName}
	nextVar.SetDeterminedType(nextReturnType)
	nextVar.SetInitialExpression(nextInv)
	nextVar.SetSymbol(nextSymbol)
	nextVarDef := &ast.BLangSimpleVariableDef{Var: nextVar}
	setPositionIfMissing(nextVarDef, basePos)
	whileBodyStmts = append(whileBodyStmts, nextVarDef)

	// 4b: if $next is () { break }
	nextRefForNilCheck := &ast.BLangSimpleVarRef{VariableName: nextVarName}
	nextRefForNilCheck.SetSymbol(nextSymbol)
	nextRefForNilCheck.SetDeterminedType(nextReturnType)

	nilCheck := &ast.BLangTypeTestExpr{}
	nilCheck.Expr = nextRefForNilCheck
	nilCheck.Type = ast.TypeData{Type: semtypes.NIL}
	nilCheck.SetDeterminedType(semtypes.BOOLEAN)

	breakStmt := &ast.BLangBreak{}
	setPositionIfMissing(breakStmt, basePos)
	nilCheckIf := &ast.BLangIf{
		Expr: nilCheck,
		Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{breakStmt}},
	}
	nilCheckIf.SetScope(foreachScope)
	setPositionIfMissing(nilCheckIf, basePos)
	whileBodyStmts = append(whileBodyStmts, nilCheckIf)

	// 4c: if $next is error { panic $next } (only if error is possible)
	hasError := !semtypes.IsEmpty(tyCtx, semtypes.Intersect(nextReturnType, semtypes.ERROR))
	if hasError {
		nextRefForErrCheck := &ast.BLangSimpleVarRef{VariableName: nextVarName}
		nextRefForErrCheck.SetSymbol(nextSymbol)
		nextRefForErrCheck.SetDeterminedType(nextReturnType)

		errCheck := &ast.BLangTypeTestExpr{}
		errCheck.Expr = nextRefForErrCheck
		errCheck.Type = ast.TypeData{Type: semtypes.ERROR}
		errCheck.SetDeterminedType(semtypes.BOOLEAN)

		nextRefForPanic := &ast.BLangSimpleVarRef{VariableName: nextVarName}
		nextRefForPanic.SetSymbol(nextSymbol)
		nextRefForPanic.SetDeterminedType(nextReturnType)

		panicStmt := &ast.BLangPanic{Expr: nextRefForPanic}
		setPositionIfMissing(panicStmt, basePos)
		errCheckIf := &ast.BLangIf{
			Expr: errCheck,
			Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{panicStmt}},
		}
		errCheckIf.SetScope(foreachScope)
		setPositionIfMissing(errCheckIf, basePos)
		whileBodyStmts = append(whileBodyStmts, errCheckIf)
	}

	// 4d: loopVar = $next.value (field access desugared to index access by walkBlockStmt)
	nextRefForValue := &ast.BLangSimpleVarRef{VariableName: nextVarName}
	nextRefForValue.SetSymbol(nextSymbol)
	nextRefForValue.SetDeterminedType(semtypes.MAPPING)

	valueAccess := &ast.BLangFieldBaseAccess{
		Field: &ast.BLangIdentifier{Value: "value"},
	}
	valueAccess.Expr = nextRefForValue
	valueAccess.SetDeterminedType(loopVarDef.Var.GetDeterminedType())
	setPositionIfMissing(valueAccess, basePos)

	loopVarDef.Var.SetInitialExpression(valueAccess)
	whileBodyStmts = append(whileBodyStmts, loopVarDef)

	// 4e: original body stmts
	whileBodyStmts = append(whileBodyStmts, body.Stmts...)

	body.Stmts = whileBodyStmts

	// No loop var tracking needed — no index increment for iterable foreach
	// Continue in iterable foreach just goes back to calling next()
	cx.pushLoopVar(nil)
	bodyResult := walkBlockStmt(cx, body)
	newBody := bodyResult.replacementNode.(*ast.BLangBlockStmt)
	cx.popLoopVar()

	// Step 5: Create while loop
	whileStmt := &ast.BLangWhile{
		Expr: trueLiteral,
		Body: *newBody,
	}
	whileStmt.SetScope(foreachScope)
	whileStmt.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(whileStmt, basePos)

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: whileStmt,
	}
}

func walkMatchStatement(cx *functionContext, stmt *ast.BLangMatchStatement) desugaredNode[ast.StatementNode] {
	var initStmts []ast.StatementNode

	if stmt.Expr != nil {
		result := walkExpression(cx, stmt.Expr)
		initStmts = append(initStmts, result.initStmts...)
		stmt.Expr = result.replacementNode
	}

	for i := range stmt.MatchClauses {
		clause := &stmt.MatchClauses[i]
		// Const-pattern expressions are lowered directly by BIR generation, so
		// they must be walked here too — otherwise a reference to a folded
		// constant would survive as a dangling global once the constant is
		// dropped from the BIR package.
		for _, pattern := range clause.Patterns {
			constPattern, ok := pattern.(*ast.BLangConstPattern)
			if !ok {
				continue
			}
			patternResult := walkExpression(cx, constPattern.Expr)
			initStmts = append(initStmts, patternResult.initStmts...)
			constPattern.Expr = patternResult.replacementNode.(ast.BLangExpression)
		}
		if clause.Guard != nil {
			guardResult := walkExpression(cx, clause.Guard)
			initStmts = append(initStmts, guardResult.initStmts...)
			clause.Guard = guardResult.replacementNode.(ast.BLangExpression)
		}
		bodyResult := walkBlockStmt(cx, &clause.Body)
		clause.Body = *bodyResult.replacementNode.(*ast.BLangBlockStmt)
	}

	return desugaredNode[ast.StatementNode]{
		initStmts:       initStmts,
		replacementNode: stmt,
	}
}
