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
	ast.BLangActionOrExpression
	ResolvedSymbol() model.SymbolRef
	Receiver() ast.BLangExpression
	SetReceiver(ast.BLangExpression)
	CallArgs() []ast.BLangExpression
	SetCallArgs([]ast.BLangExpression)
}

func walkExpression(cx *functionContext, node ast.BLangActionOrExpression) ast.BLangActionOrExpression {
	lowered := walkExpressionInner(cx, node)
	if len(lowered.initStmts) == 0 {
		return lowered.replacementNode
	}
	// Enclose setup statements at the expression boundary so they retain the
	// expression's lazy evaluation, loop, and trap semantics.
	return createExpressionThunk(cx, lowered)
}

func walkExpressionInner(cx *functionContext, node ast.BLangActionOrExpression) desugaredNode[ast.BLangActionOrExpression] {
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
	case *BLangServiceInit, *BLangExpressionThunk:
		// Desugar-introduced nodes; nothing to rewrite further.
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangNamedArgsExpression:
		result := walkExpression(cx, expr.Expr)
		expr.Expr = result.(ast.BLangExpression)
		return desugaredNode[ast.BLangActionOrExpression]{
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
		for i, child := range expr.Children {
			r := walkExpression(cx, child)
			expr.Children[i] = r.(ast.BLangExpression)
		}
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	case *ast.BLangXMLElementLiteral:
		for i := range expr.Attrs {
			if expr.Attrs[i].Value != nil {
				r := walkExpression(cx, expr.Attrs[i].Value)
				expr.Attrs[i].Value = r.(ast.BLangExpression)
			}
		}
		if expr.Content != nil {
			r := walkExpression(cx, expr.Content)
			expr.Content = r.(ast.BLangExpression)
		}
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
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

// FIXME: Remove this guard when check lowering can preserve the enclosing
// function's return semantics across generated thunk boundaries.
type checkedExpressionVisitor struct {
	cx *functionContext
}

func (v *checkedExpressionVisitor) Visit(node ast.BLangNode) ast.Visitor {
	switch node.(type) {
	case *ast.BLangCheckedExpr:
		v.cx.internalError("check expression cannot be lowered inside a generated expression thunk")
		return nil
	case *ast.BLangFunction, *ast.BLangLambdaFunction, *ast.BLangArrowFunction, *BLangExpressionThunk:
		return nil
	default:
		return v
	}
}

func (v *checkedExpressionVisitor) VisitTypeData(_ *ast.TypeData) ast.Visitor {
	return v
}

func createExpressionThunk(cx *functionContext, lowered desugaredNode[ast.BLangActionOrExpression]) *BLangExpressionThunk {
	pos := lowered.replacementNode.GetPosition()
	visitor := &checkedExpressionVisitor{cx: cx}
	for _, stmt := range lowered.initStmts {
		ast.Walk(visitor, stmt.(ast.BLangNode))
	}
	ast.Walk(visitor, lowered.replacementNode)

	ownerName := cx.getSymbol(cx.owner).Name()
	name := fmt.Sprintf("%s$%d$%d$thunk$%d", ownerName, cx.owner.SpaceIndex, cx.owner.Index, cx.thunkCounter)
	cx.thunkCounter++

	returnTy := lowered.replacementNode.GetDeterminedType()
	fnTy := cx.thunkFunctionType(returnTy)
	fnSymbol := model.NewFunctionSymbol(name, model.TypedFunctionSignature{ReturnType: returnTy}, false, pos)
	fnSymbol.SetType(fnTy)
	fnRef := cx.addSymbolToSameSpace(cx.owner, name, fnSymbol)

	returnStmt := &ast.BLangReturn{Expr: lowered.replacementNode}
	returnStmt.SetDeterminedType(semtypes.Never)
	returnStmt.SetPosition(pos)
	body := &ast.BLangBlockFunctionBody{Stmts: append(lowered.initStmts, returnStmt)}
	body.SetDeterminedType(semtypes.Never)
	body.SetPosition(pos)
	fnName := newIdentifier(name)
	fnName.SetPosition(pos)
	fn := ast.NewBLangFunction(ast.InvokableData{Position: pos, Name: fnName, Body: body})
	fn.SetDeterminedType(semtypes.Never)
	fn.SetSymbol(fnRef)
	fn.SetScope(cx.newFunctionScope(cx.currentScope()))

	lambda := &ast.BLangLambdaFunction{Function: fn}
	lambda.SetDeterminedType(fnTy)
	lambda.SetPosition(pos)
	thunk := &BLangExpressionThunk{Lambda: lambda}
	thunk.SetDeterminedType(returnTy)
	thunk.SetPosition(pos)
	return thunk
}

func walkBinaryExpr(cx *functionContext, expr *ast.BLangBinaryExpr) desugaredNode[ast.BLangActionOrExpression] {
	if expr.LhsExpr != nil {
		result := walkExpression(cx, expr.LhsExpr)
		expr.LhsExpr = result.(ast.BLangExpression)
	}

	if expr.RhsExpr != nil {
		result := walkExpression(cx, expr.RhsExpr)
		expr.RhsExpr = result.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkUnaryExpr(cx *functionContext, expr *ast.BLangUnaryExpr) desugaredNode[ast.BLangActionOrExpression] {

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		expr.Expr = result.(ast.BLangExpression)
	}

	// Unary + is identity — desugar to just the operand (BIR gen doesn't handle unary +)
	if expr.Operator == model.OperatorKind_ADD {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr.Expr}
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkElvisExpr(cx *functionContext, expr *ast.BLangElvisExpr) desugaredNode[ast.BLangActionOrExpression] {

	if expr.LhsExpr != nil {
		result := walkExpression(cx, expr.LhsExpr)
		expr.LhsExpr = result.(ast.BLangExpression)
	}

	if expr.RhsExpr != nil {
		result := walkExpression(cx, expr.RhsExpr)
		expr.RhsExpr = result.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkGroupExpr(cx *functionContext, expr *ast.BLangGroupExpr) desugaredNode[ast.BLangActionOrExpression] {

	if expr.Expression != nil {
		result := walkExpression(cx, expr.Expression)
		expr.Expression = result.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkIndexBasedAccess(cx *functionContext, expr *ast.BLangIndexBasedAccess) desugaredNode[ast.BLangActionOrExpression] {

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		expr.Expr = result.(ast.BLangExpression)
	}

	if expr.IndexExpr != nil {
		result := walkExpression(cx, expr.IndexExpr)
		expr.IndexExpr = result.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkFieldBaseAccess(cx *functionContext, expr *ast.BLangFieldBaseAccess) desugaredNode[ast.BLangActionOrExpression] {

	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		expr.Expr = result.(ast.BLangExpression)
	}

	if expr.IsOptionalAccess() {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}

	indexAccess := createFieldIndexAccess(expr.Expr, expr.Field.GetValue(), expr.GetDeterminedType(), expr.GetPosition())

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: indexAccess}
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

func walkTemplateExpr(cx *functionContext, expr *ast.BLangTemplateExpr) desugaredNode[ast.BLangActionOrExpression] {
	if len(expr.Insertions) == 0 {
		lit := &ast.BLangLiteral{Value: expr.Strings[0], OriginalValue: expr.Strings[0]}
		lit.SetPosition(expr.GetPosition())
		lit.SetDeterminedType(semtypes.StringConst(expr.Strings[0]))
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: lit}
	}
	for i, ins := range expr.Insertions {
		r := walkExpression(cx, ins)
		expr.Insertions[i] = r.(ast.BLangExpression)
	}
	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkClientResourceAccessAction(cx *functionContext, expr *ast.BLangClientResourceAccessAction) desugaredNode[ast.BLangActionOrExpression] {
	var initStmts []ast.StatementNode
	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		expr.Expr = result.(ast.BLangExpression)
	}
	for i := range expr.Path {
		seg := &expr.Path[i]
		if seg.Kind != ast.ResourceAccessSegmentComputed {
			continue
		}
		result := walkExpression(cx, seg.Expr)
		seg.Expr = result.(ast.BLangExpression)
	}
	initStmts = append(initStmts, walkInvocationArgs(cx, expr)...)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: expr,
	}
}

func walkXMLTemplateExpr(cx *functionContext, expr *ast.BLangXMLTemplateExpr) desugaredNode[ast.BLangActionOrExpression] {
	for i, ins := range expr.Insertions {
		r := walkExpression(cx, ins)
		insert := r.(ast.BLangExpression)
		if shouldEscapeXMLTemplateInsertion(insert, expr.InsertionKinds[i], cx) {
			insert = escapeXMLTemplateInsertion(cx, insert, expr.InsertionKinds[i])
		}
		expr.Insertions[i] = insert
	}
	expr.Strings = spliceXMLTemplateNamespaces(cx, expr.Strings, expr.NamespaceInsertions)
	plain := &ast.BLangTemplateExpr{Kind: ast.TemplateExprKindXML, Strings: expr.Strings, Insertions: expr.Insertions}
	plain.SetPosition(expr.GetPosition())
	plain.SetDeterminedType(expr.GetDeterminedType())
	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: plain}
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
		expr.SetReceiver(result.(ast.BLangExpression))
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
			cx.internalError("expected function signature to default expressions")
		}
		return hoistAndAddDefaultInvocations(cx, args, sig, pos)
	}

	initStmts := make([]ast.StatementNode, 0, len(args))
	walkedArgs := make([]ast.BLangExpression, len(args))
	for i, arg := range args {
		result := walkExpression(cx, arg)
		walkedArgs[i] = result.(ast.BLangExpression)
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
			varDef, varRef := assignToLocal(cx, result.(ast.BLangExpression), pos)
			hoistInit = append(hoistInit, varDef)
			hoistedArgs[i] = varRef
		} else {
			result := walkExpression(cx, arg)
			varDef, varRef := assignToLocal(cx, result.(ast.BLangExpression), arg.GetPosition())
			hoistInit = append(hoistInit, varDef)
			hoistedArgs[i] = varRef
		}
	}
	for i := fixedCount; i < len(args); i++ {
		arg := args[i]
		result := walkExpression(cx, arg)
		hoistedArgs[i] = result.(ast.BLangExpression)
	}

	return hoistInit, hoistedArgs
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
	tempName, tempSymRef := cx.addDesugardSymbol(ty, model.SymbolKindVariable, false, pos)
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
	for i := range expr.Exprs {
		result := walkExpression(cx, expr.Exprs[i])
		expr.Exprs[i] = result.(ast.BLangExpression)
	}

	if expr.HasSpreadMembers() {
		return desugarListConstructorWithSpread(cx, expr, nil)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
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
	//nolint:staticcheck // TODO
	if expr.ErrorTypeRef != nil {
		// ErrorTypeRef is a type descriptor, not an expression, so we don't walk it
	}

	for i := range expr.PositionalArgs {
		result := walkExpression(cx, expr.PositionalArgs[i])
		expr.PositionalArgs[i] = result.(ast.BLangExpression)
	}

	for i := range expr.NamedArgs {
		result := walkExpression(cx, expr.NamedArgs[i].Expr)
		expr.NamedArgs[i].Expr = result.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkCheckedExpr(cx *functionContext, expr *ast.BLangCheckedExpr) desugaredNode[ast.BLangActionOrExpression] {
	result := walkExpression(cx, expr.Expr)
	expr.Expr = result
	return desugaredNode[ast.BLangActionOrExpression]{
		replacementNode: expr,
	}
}

func walkCheckPanickedExpr(cx *functionContext, expr *ast.BLangCheckPanickedExpr) desugaredNode[ast.BLangActionOrExpression] {
	result := walkExpression(cx, expr.Expr)
	expr.Expr = result
	return desugaredNode[ast.BLangActionOrExpression]{
		replacementNode: expr,
	}
}

func walkTrapExpr(cx *functionContext, expr *ast.BLangTrapExpr) desugaredNode[ast.BLangActionOrExpression] {
	result := walkExpression(cx, expr.Expr)
	expr.Expr = result.(ast.BLangExpression)
	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
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
		expr.Expression = result.(ast.BLangExpression)
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
		expr.Expr = result.(ast.BLangExpression)
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
	if expr.Expr != nil {
		result := walkExpression(cx, expr.Expr)
		expr.Expr = result.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func walkArrowFunction(cx *functionContext, expr *ast.BLangArrowFunction) desugaredNode[ast.BLangActionOrExpression] {
	if expr.Body != nil {
		result := walkExpression(cx, expr.Body.Expr.(ast.BLangActionOrExpression))
		expr.Body.Expr = result.(ast.BLangExpression)
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
		kv.ValueExpr = result.(ast.BLangExpression)
	}

	return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
}

func createVarRef(varName ast.IdentifierNode, symbol model.SymbolRef, ty semtypes.SemType) *ast.BLangVarRef {
	ref := &ast.BLangVarRef{VariableName: varName}
	ref.SetSymbol(symbol)
	ref.SetDeterminedType(ty)
	return ref
}
