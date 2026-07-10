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
	"fmt"
	"reflect"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
)

type analyzer interface {
	ast.Visitor
	ctx() *context.CompilerContext
	tyCtx() semtypes.Context
	getSymbol(ref model.SymbolRef) model.Symbol
	internalError(message string, loc diagnostics.Location)
	importedPackage(alias string) *ast.BLangImportPackage
	unimplementedErr(message string, loc diagnostics.Location)
	semanticErr(message string, loc diagnostics.Location)
	syntaxErr(message string, loc diagnostics.Location)
	internalErr(message string, loc diagnostics.Location)
	parentAnalyzer() analyzer
	loc() diagnostics.Location
	moduleVarMetadata(ref model.SymbolRef) (varDeclMetadata, bool)
}

type (
	analyzerBase struct {
		parent analyzer
	}
	SemanticAnalyzer struct {
		analyzerBase
		compilerCtx *context.CompilerContext
		typeCtx     semtypes.Context
		// TODO: move the constant resolution to type resolver as well so that we can run semantic analyzer in parallel as well
		pkg              *ast.BLangPackage
		importedPkgs     map[string]*ast.BLangImportPackage
		importedSymbols  map[string]model.ExportedSymbolSpace
		moduleVarMetaMap map[model.SymbolRef]varDeclMetadata
	}
	constantAnalyzer struct {
		analyzerBase
		constant     *ast.BLangConstant
		expectedType semtypes.SemType
	}

	functionAnalyzer struct {
		analyzerBase
		function ast.BLangNode
		retTy    semtypes.SemType
		// enclosingClass is set when the function is a method of a class or
		// service body. nil for free functions.
		enclosingClass *enclosingClassBody
		// locals tracks variable declarations visible inside this function's
		// body, populated as normal semantic analysis walks the body. Used to
		// hand an outer-function scope to the isolation check when validating
		// closure expressions (record-field defaults, default-param exprs,
		// nested isolated function bodies).
		locals *localScope
	}

	loopAnalyzer struct {
		analyzerBase
		loop ast.BLangNode
	}

	lockAnalyzer struct {
		analyzerBase
		lock *ast.BLangLock
	}
)

var (
	_ analyzer = &constantAnalyzer{}
	_ analyzer = &SemanticAnalyzer{}
	_ analyzer = &functionAnalyzer{}
	_ analyzer = &loopAnalyzer{}
	_ analyzer = &lockAnalyzer{}
)

// expectedReturnType walks up the analyzer chain and returns the enclosing function's return type.
// Returns nil if not inside a function.
func expectedReturnType(a analyzer) semtypes.SemType {
	current := a
	for current != nil {
		if fa, ok := current.(*functionAnalyzer); ok {
			return fa.retTy
		}
		current = current.parentAnalyzer()
	}
	return semtypes.SemType{}
}

// enclosingFunctionAnalyzer walks up the analyzer chain and returns the
// nearest enclosing functionAnalyzer, or nil if there is none.
func enclosingFunctionAnalyzer(a analyzer) *functionAnalyzer {
	for current := a; current != nil; current = current.parentAnalyzer() {
		if fa, ok := current.(*functionAnalyzer); ok {
			return fa
		}
	}
	return nil
}

// enclosingFunctionLocals returns the locals scope of the nearest enclosing
// functionAnalyzer, or nil if none exists.
func enclosingFunctionLocals(a analyzer) *localScope {
	if fa := enclosingFunctionAnalyzer(a); fa != nil {
		return fa.locals
	}
	return nil
}

func returnFound(a analyzer, returnStmt *ast.BLangReturn) bool {
	retTy := expectedReturnType(a)
	if semtypes.IsZero(retTy) {
		a.ctx().SemanticError("return statement not allowed in this context", a.loc())
		return false
	}
	if returnStmt.Expr == nil {
		if !semtypes.IsSubtype(a.tyCtx(), retTy, semtypes.NIL) {
			a.ctx().SemanticError("expect a return value", returnStmt.GetPosition())
			return false
		}
	} else if !analyzeActionOrExpression(a, returnStmt.Expr, retTy) {
		return false
	}
	return true
}

func (ab *analyzerBase) parentAnalyzer() analyzer {
	return ab.parent
}

func (ab *analyzerBase) importedPackage(alias string) *ast.BLangImportPackage {
	return ab.parentAnalyzer().importedPackage(alias)
}

func (ab *analyzerBase) ctx() *context.CompilerContext {
	return ab.parentAnalyzer().ctx()
}

func (ab *analyzerBase) getSymbol(ref model.SymbolRef) model.Symbol {
	return ab.ctx().GetSymbol(ref)
}

func (ab *analyzerBase) internalError(message string, loc diagnostics.Location) {
	ab.ctx().InternalError(message, loc)
}

func (ab *analyzerBase) tyCtx() semtypes.Context {
	return ab.parentAnalyzer().tyCtx()
}

func (ab *analyzerBase) moduleVarMetadata(ref model.SymbolRef) (varDeclMetadata, bool) {
	return ab.parentAnalyzer().moduleVarMetadata(ref)
}

func (sa *SemanticAnalyzer) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return nil
}

func (fa *functionAnalyzer) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return nil
}

func (la *loopAnalyzer) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return la
}

func (fa *functionAnalyzer) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BLangReturn:
		if !returnFound(fa, n) {
			return nil
		}
		return fa
	case *ast.BLangIdentifier:
		return nil
	case *ast.BLangSimpleVarRef:
		checkIsolatedModuleVarOutsideLock(fa, n)
		return visitInner(fa, n)
	case *ast.BLangFieldBaseAccess:
		checkIsolatedFieldOutsideLock(fa, n)
		return visitInner(fa, n)
	default:
		// Delegate loop creation and common nodes to visitInner
		return visitInner(fa, node)
	}
}

func (la *loopAnalyzer) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	switch node.(type) {
	case *ast.BLangBreak, *ast.BLangContinue:
		return nil
	default:
		// Delegate nested loops and common nodes to visitInner
		return visitInner(la, node)
	}
}

func (fa *functionAnalyzer) loc() diagnostics.Location {
	return fa.function.GetPosition()
}

func (la *loopAnalyzer) loc() diagnostics.Location {
	return la.loop.GetPosition()
}

func (sa *SemanticAnalyzer) loc() diagnostics.Location {
	return sa.pkg.GetPosition()
}

func (ca *constantAnalyzer) loc() diagnostics.Location {
	return ca.constant.GetPosition()
}

func (ca *constantAnalyzer) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return ca
}

func (sa *SemanticAnalyzer) ctx() *context.CompilerContext {
	return sa.compilerCtx
}

func (sa *SemanticAnalyzer) tyCtx() semtypes.Context {
	return sa.typeCtx
}

func (sa *SemanticAnalyzer) importedPackage(alias string) *ast.BLangImportPackage {
	return sa.importedPkgs[alias]
}

func (la *loopAnalyzer) ctx() *context.CompilerContext {
	return la.parent.ctx()
}

func (la *loopAnalyzer) tyCtx() semtypes.Context {
	return la.parent.tyCtx()
}

func (sa *SemanticAnalyzer) unimplementedErr(message string, loc diagnostics.Location) {
	sa.compilerCtx.Unimplemented(message, loc)
}

func (sa *SemanticAnalyzer) semanticErr(message string, loc diagnostics.Location) {
	sa.compilerCtx.SemanticError(message, loc)
}

func (sa *SemanticAnalyzer) syntaxErr(message string, loc diagnostics.Location) {
	sa.compilerCtx.SyntaxError(message, loc)
}

func (sa *SemanticAnalyzer) internalErr(message string, loc diagnostics.Location) {
	sa.compilerCtx.InternalError(message, loc)
}

func (sa *SemanticAnalyzer) internalError(message string, loc diagnostics.Location) {
	sa.compilerCtx.InternalError(message, loc)
}

func (ca *constantAnalyzer) unimplementedErr(message string, loc diagnostics.Location) {
	ca.parentAnalyzer().ctx().Unimplemented(message, loc)
}

func (ca *constantAnalyzer) semanticErr(message string, loc diagnostics.Location) {
	ca.parentAnalyzer().ctx().SemanticError(message, loc)
}

func (ca *constantAnalyzer) syntaxErr(message string, loc diagnostics.Location) {
	ca.parentAnalyzer().ctx().SyntaxError(message, loc)
}

func (ca *constantAnalyzer) internalErr(message string, loc diagnostics.Location) {
	ca.parentAnalyzer().ctx().InternalError(message, loc)
}

func (fa *functionAnalyzer) unimplementedErr(message string, loc diagnostics.Location) {
	fa.parent.ctx().Unimplemented(message, loc)
}

func (fa *functionAnalyzer) semanticErr(message string, loc diagnostics.Location) {
	fa.parent.ctx().SemanticError(message, loc)
}

func (fa *functionAnalyzer) syntaxErr(message string, loc diagnostics.Location) {
	fa.parent.ctx().SyntaxError(message, loc)
}

func (fa *functionAnalyzer) internalErr(message string, loc diagnostics.Location) {
	fa.parent.ctx().InternalError(message, loc)
}

func (la *loopAnalyzer) unimplementedErr(message string, loc diagnostics.Location) {
	la.parent.ctx().Unimplemented(message, loc)
}

func (la *loopAnalyzer) semanticErr(message string, loc diagnostics.Location) {
	la.parent.ctx().SemanticError(message, loc)
}

func (la *loopAnalyzer) syntaxErr(message string, loc diagnostics.Location) {
	la.parent.ctx().SyntaxError(message, loc)
}

func (la *loopAnalyzer) internalErr(message string, loc diagnostics.Location) {
	la.parent.ctx().InternalError(message, loc)
}

func NewSemanticAnalyzer(ctx *context.CompilerContext) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		compilerCtx:     ctx,
		typeCtx:         semtypes.ContextFrom(ctx.GetTypeEnv()),
		importedPkgs:    make(map[string]*ast.BLangImportPackage),
		importedSymbols: make(map[string]model.ExportedSymbolSpace),
	}
}

func (sa *SemanticAnalyzer) Analyze(pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) {
	sa.pkg = pkg
	sa.importedPkgs = make(map[string]*ast.BLangImportPackage)
	if importedSymbols == nil {
		importedSymbols = make(map[string]model.ExportedSymbolSpace)
	}
	sa.importedSymbols = importedSymbols
	sa.moduleVarMetaMap = sa.buildModuleVarMetadata()
	sa.validateModuleLevelIsolatedDecls(pkg)
	ast.Walk(sa, pkg)
	sa.pkg = nil
	sa.importedPkgs = nil
	sa.importedSymbols = nil
	sa.moduleVarMetaMap = nil
}

func (sa *SemanticAnalyzer) moduleVarMetadata(ref model.SymbolRef) (varDeclMetadata, bool) {
	md, ok := sa.moduleVarMetaMap[ref]
	return md, ok
}

func createConstantAnalyzer(parent analyzer, constant *ast.BLangConstant) *constantAnalyzer {
	expectedType := constant.GetAssociatedType()
	return &constantAnalyzer{analyzerBase: analyzerBase{parent: parent}, constant: constant, expectedType: expectedType}
}

func (sa *SemanticAnalyzer) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		// Done
		return nil
	}
	switch n := node.(type) {
	case *ast.BLangImportPackage:
		sa.processImport(n)
		return nil
	case *ast.BLangConstant:
		return createConstantAnalyzer(sa, n)
	case *ast.BLangSimpleVariable:
		return sa
	case *ast.BLangSimpleVarRef:
		checkIsolatedModuleVarOutsideLock(sa, n)
		return nil
	case *ast.BLangReturn:
		// Error: return only valid in functions
		sa.semanticErr("return statement outside function", n.GetPosition())
		return nil
	case *ast.BLangWhile:
		// Error: loop only valid in functions
		sa.semanticErr("loop statement outside function", n.GetPosition())
		return nil
	case *ast.BLangIf:
		sa.semanticErr("if statement outside function", n.GetPosition())
		return nil
	default:
		// Now delegates function creation to visitInner
		return visitInner(sa, node)
	}
}

func (sa *SemanticAnalyzer) processImport(importNode *ast.BLangImportPackage) {
	alias := importNode.Alias.GetValue()

	// Check for duplicate imports
	if _, exists := sa.importedPkgs[alias]; exists {
		sa.semanticErr(fmt.Sprintf("import alias '%s' already defined", alias), importNode.GetPosition())
		return
	}

	sa.importedPkgs[alias] = importNode
}

func isImplicitImport(importNode *ast.BLangImportPackage) bool {
	return isLangImport(importNode, "array") || isLangImport(importNode, "int") || isLangImport(importNode, "map") || isLangImport(importNode, "string")
}

func isLangImport(importNode *ast.BLangImportPackage, name string) bool {
	return len(importNode.PkgNameComps) == 2 && importNode.PkgNameComps[0].GetValue() == "lang" && importNode.PkgNameComps[1].GetValue() == name
}

func validateInitFunction(a analyzer, function *ast.BLangFunction, fnSymbol model.FunctionSymbol, pos diagnostics.Location) {
	if function.IsPublic() {
		a.semanticErr("'init' function cannot be declared as public", pos)
	}

	actualReturnType := fnSymbol.Signature().ReturnType
	if !semtypes.IsZero(actualReturnType) {
		if !semtypes.IsSameType(a.tyCtx(), actualReturnType, semtypes.NIL) && !semtypes.IsSameType(a.tyCtx(), actualReturnType, semtypes.Union(semtypes.NIL, semtypes.ERROR)) {
			a.semanticErr("'init' function must have return type '()' or  'error?'", pos)
		}
	}

	if len(function.RequiredParams) > 0 || function.RestParam != nil {
		a.semanticErr("'init' function cannot have parameters", pos)
	}
}

func validateMainFunction(a analyzer, fnSymbol model.FunctionSymbol, pos diagnostics.Location) {
	if !fnSymbol.IsPublic() {
		a.semanticErr("'main' function must be public", pos)
	}

	actualReturnType := fnSymbol.Signature().ReturnType

	if !semtypes.IsZero(actualReturnType) {
		if !semtypes.IsSameType(a.tyCtx(), actualReturnType, semtypes.NIL) && !semtypes.IsSameType(a.tyCtx(), actualReturnType, semtypes.Union(semtypes.NIL, semtypes.ERROR)) {
			a.semanticErr("'main' function must have return type '()' or  'error?'", pos)
		}
	}
}

func initializeFunctionAnalyzer(parent analyzer, function *ast.BLangFunction) *functionAnalyzer {
	fa := initializeFunctionAnalyzerInner(parent, function, nil)
	// Validate main function constraints
	if function.Name.Value == "main" {
		fnSymbol := parent.ctx().GetSymbol(function.Symbol()).(model.FunctionSymbol)
		validateMainFunction(parent, fnSymbol, function.GetPosition())
	}
	if function.Name.Value == "init" {
		// this is to seperate class init from module init
		if _, isTopLevel := parent.(*SemanticAnalyzer); isTopLevel {
			fnSymbol := parent.ctx().GetSymbol(function.Symbol()).(model.FunctionSymbol)
			validateInitFunction(parent, function, fnSymbol, function.GetPosition())
		}
	}

	return fa
}

func initializeFunctionAnalyzerInner(parent analyzer, function *ast.BLangFunction, enclosing *enclosingClassBody) *functionAnalyzer {
	return initializeInvokableAnalyzer(parent, function, enclosing, buildFunctionLocals(parent, function))
}

// invokableSignatureNode is implemented by AST nodes that share the invokable
// base (functions and resource methods). It exposes the parameter/body surface
// the shared function analysis and validation helpers operate on.
type invokableSignatureNode interface {
	ast.BLangNode
	Symbol() model.SymbolRef
	IsIsolated() bool
	IsNative() bool
	RequiredParameters() []ast.BLangSimpleVariable
	GetRestParam() ast.SimpleVariableNode
	GetBody() ast.FunctionBodyNode
}

// initializeInvokableAnalyzer builds the functionAnalyzer shared by plain
// functions/methods and resource methods, running the per-parameter and
// signature validations on the given invokable using the provided locals scope.
func initializeInvokableAnalyzer(parent analyzer, function invokableSignatureNode, enclosing *enclosingClassBody, locals *localScope) *functionAnalyzer {
	fa := &functionAnalyzer{
		analyzerBase:   analyzerBase{parent: parent},
		function:       function,
		enclosingClass: enclosing,
		locals:         locals,
	}
	fnSymbol := parent.ctx().GetSymbol(function.Symbol()).(model.FunctionSymbol)
	if depSym, ok := fnSymbol.(model.DependentlyTypedFunctionSymbol); ok {
		validateDependentFunction(parent, function, depSym)
		validateDefaultParamTypes(parent, function)
		return fa
	}
	rejectInferredTypedescOnNonDependent(parent, function)
	fa.retTy = fnSymbol.Signature().ReturnType
	validateDefaultParamTypes(parent, function)
	if function.IsIsolated() && !function.IsNative() {
		validateIsolatedFunction(fa, function)
		validateIsolatedDefaultParams(fa, function)
	}
	return fa
}

// buildFunctionLocals creates the per-function locals scope, seeded with
// the function's parameters (and `self` for methods). Body-local variables
// are added later as normal semantic analysis encounters their definitions.
func buildFunctionLocals(parent analyzer, fn invokableSignatureNode) *localScope {
	scope := newLocalScope(enclosingFunctionLocals(parent), true)
	finishBuildFunctionLocals(parent, scope, fn.RequiredParameters(), fn.GetRestParam())
	return scope
}

// finishBuildFunctionLocals seeds a function-locals scope with the function's
// required parameters and rest parameter (if any).
func finishBuildFunctionLocals(parent analyzer, scope *localScope, requiredParams []ast.BLangSimpleVariable, restParam ast.SimpleVariableNode) {
	for _, param := range requiredParams {
		sym := param.Symbol()
		scope.define(sym, varDeclMetadata{Type: parent.ctx().SymbolType(sym), Final: true})
	}
	if restParam != nil {
		sym := restParam.Symbol()
		scope.define(sym, varDeclMetadata{Type: parent.ctx().SymbolType(sym), Final: true})
	}
}

// validateIsolatedDefaultParams runs the isolated-closure analysis on
// each non-`<>` default parameter expression of an isolated function.
// Default expressions are themselves closures invoked at call time, so
// for an isolated function they must independently satisfy the
// isolated-closure rules. The function-body's normal walker handles the
// non-isolated case implicitly: default expressions are walked through
// the function-analyzer, and `enclosingLockAnalyzer` stops at that
// boundary, so any isolated module variable reference is reported by
// `checkIsolatedModuleVarOutsideLock` exactly as it would be for any
// other unprotected read.
func validateIsolatedDefaultParams[A analyzer](a A, function invokableSignatureNode) {
	fa := enclosingFunctionAnalyzer(a)
	requiredParams := function.RequiredParameters()
	for i := range requiredParams {
		param := &requiredParams[i]
		if !param.IsDefaultableParam() {
			continue
		}
		// `<>` (inferred typedesc default) is a marker, not a real expression.
		if _, ok := param.Expr.(*ast.BLangInferredTypedescDefault); ok {
			continue
		}
		// We turn defaultable parameters to closures and for isolated functions those closures need to be isolated.
		// The function's own params (already in fa.locals) are visible as captures to each default expression.
		var parent *localScope
		if fa != nil {
			parent = fa.locals
		}
		expr := param.Expr.(ast.BLangNode)
		validateIsolatedCapture(a, parent, expr)
		isIsolatedFunctionInner(a, expr, parent)
	}
}

// rejectInferredTypedescOnNonDependent emits an error for any `<>` default on a function
// whose return type does not depend on that parameter.
func rejectInferredTypedescOnNonDependent(a analyzer, fn invokableSignatureNode) {
	requiredParams := fn.RequiredParameters()
	for i := range requiredParams {
		param := &requiredParams[i]
		if !param.IsDefaultableParam() {
			continue
		}
		if _, ok := param.Expr.(*ast.BLangInferredTypedescDefault); !ok {
			continue
		}
		a.semanticErr("inferred typedesc default '<>' requires the return type to depend on this parameter", param.GetPosition())
	}
}

// validateDependentFunction enforces the rules around dependently-typed functions:
//  1. The function body must be external. Otherwise: "dependently-typed function must be external".
//  2. A parameter with inferred-typedesc default '<>' must be the parameter the return type
//     depends on. Otherwise: "inferred typedesc default '<>' requires the return type to depend on this parameter".
//  3. A union of dependent and defined return parts must be disjoint.
func validateDependentFunction(a analyzer, fn invokableSignatureNode, sym model.DependentlyTypedFunctionSymbol) {
	if _, ok := fn.GetBody().(*ast.BLangExternFunctionBody); !ok {
		a.semanticErr("dependently-typed function must be external", fn.GetPosition())
	}
	retType := sym.ReturnType()
	if _, disjoint := checkDependentReturnParts(a, a.tyCtx(), retType, sym.ParamTypes(), fn.GetPosition()); !disjoint {
		a.semanticErr("dependently-typed function return type dependent and defined parts must be disjoint", fn.GetPosition())
	}
	requiredParams := fn.RequiredParameters()
	for i := range requiredParams {
		param := &requiredParams[i]
		if !param.IsDefaultableParam() {
			continue
		}
		if _, ok := param.Expr.(*ast.BLangInferredTypedescDefault); !ok {
			continue
		}
		if !typeOpReferencesIndex(retType, i) {
			a.semanticErr("inferred typedesc default '<>' requires the return type to depend on this parameter", param.GetPosition())
		}
	}
}

func typeOpReferencesIndex(op model.TypeOp, i int) bool {
	switch o := op.(type) {
	case *model.RefTypeOp:
		return o.Index == i
	case *model.BinaryTypeOp:
		return typeOpReferencesIndex(o.Lhs, i) || typeOpReferencesIndex(o.Rhs, i)
	}
	return false
}

func checkDependentReturnParts(a analyzer, ctx semtypes.Context, op model.TypeOp, paramTypes []semtypes.SemType, loc diagnostics.Location) (bool, bool) {
	switch o := op.(type) {
	case *model.RefTypeOp:
		// RefTypeOp is the dependent part: it references a typedesc parameter.
		// A single part has no sibling to overlap with, so it is disjoint by itself.
		return true, true
	case *model.IdentityTypeOp:
		// IdentityTypeOp is a defined/concrete return part, e.g. error or int.
		// A single part has no sibling to overlap with, so it is disjoint by itself.
		return false, true
	case *model.BinaryTypeOp:
		lhsDepends, lhsDisjoint := checkDependentReturnParts(a, ctx, o.Lhs, paramTypes, loc)
		rhsDepends, rhsDisjoint := checkDependentReturnParts(a, ctx, o.Rhs, paramTypes, loc)
		depends := lhsDepends || rhsDepends
		if !lhsDisjoint || !rhsDisjoint {
			return depends, false
		}
		// By definition not disjoint but could be NEVER
		if o.Kind != model.TypeOpUnion || lhsDepends == rhsDepends {
			return depends, true
		}
		intersection := semtypes.Intersect(o.Lhs.Apply(ctx, paramTypes), o.Rhs.Apply(ctx, paramTypes))
		return depends, semtypes.IsEmpty(ctx, intersection)
	default:
		a.internalErr(fmt.Sprintf("unknown dependent return type op: %T", op), loc)
		return false, false
	}
}

func validateDefaultParamTypes(a analyzer, function invokableSignatureNode) {
	requiredParams := function.RequiredParameters()
	for i := range requiredParams {
		param := &requiredParams[i]
		if !param.IsDefaultableParam() {
			continue
		}
		if _, ok := param.Expr.(*ast.BLangInferredTypedescDefault); ok {
			continue
		}
		paramTy := param.GetDeterminedType()
		exprTy := param.Expr.(ast.BLangExpression).GetDeterminedType()
		if semtypes.IsZero(exprTy) {
			a.internalErr("default expression has no determined type", param.Expr.(ast.BLangNode).GetPosition())
			continue
		}
		if !semtypes.IsSubtype(a.tyCtx(), exprTy, paramTy) {
			a.semanticErr("incompatible default value for parameter '"+param.Name.Value+"'", param.Expr.(ast.BLangNode).GetPosition())
		}
	}
}

func initializeMethodAnalyzer(parent analyzer, function *ast.BLangFunction, enclosing *enclosingClassBody) *functionAnalyzer {
	return initializeFunctionAnalyzerInner(parent, function, enclosing)
}

func initializeResourceMethodAnalyzer(parent analyzer, rm *ast.BLangResourceMethod, enclosing *enclosingClassBody) *functionAnalyzer {
	return initializeInvokableAnalyzer(parent, rm, enclosing, buildResourceMethodLocals(parent, rm))
}

func buildResourceMethodLocals(parent analyzer, method *ast.BLangResourceMethod) *localScope {
	scope := newLocalScope(nil, true)
	for i := range method.ResourcePath {
		seg := &method.ResourcePath[i]
		if seg.Kind == ast.ResourcePathSegmentName || seg.Name == "" {
			continue
		}
		ref, ok := method.Scope().GetSymbol(seg.Name)
		if !ok {
			continue
		}
		scope.define(ref, varDeclMetadata{Type: parent.ctx().SymbolType(ref), Final: true})
	}
	finishBuildFunctionLocals(parent, scope, method.RequiredParams, method.RestParam)
	ref, ok := method.Scope().GetSymbol("self")
	if !ok {
		parent.internalErr("resource method missing 'self' symbol", method.GetPosition())
		return scope
	}
	scope.define(ref, varDeclMetadata{Type: parent.ctx().SymbolType(ref), Final: true})
	return scope
}

// walkMethodBody descends through a method's body using the provided
// functionAnalyzer. We avoid walking the BLangFunction node itself because
// functionAnalyzer.Visit on that node would re-initialize a fresh analyzer
// (losing context like enclosingClass).
func walkMethodBody(fa *functionAnalyzer, method invokableSignatureNode) {
	requiredParams := method.RequiredParameters()
	for i := range requiredParams {
		ast.Walk(fa, &requiredParams[i])
	}
	if restParam := method.GetRestParam(); restParam != nil {
		ast.Walk(fa, restParam.(ast.BLangNode))
	}
	if method.GetBody() == nil {
		return
	}
	ast.Walk(fa, method.GetBody().(ast.BLangNode))
}

func initializeLoopAnalyzer(parent analyzer, loop ast.BLangNode) *loopAnalyzer {
	return &loopAnalyzer{
		analyzerBase: analyzerBase{parent: parent},
		loop:         loop,
	}
}

func initializeLockAnalyzer(parent analyzer, lock *ast.BLangLock) *lockAnalyzer {
	return &lockAnalyzer{
		analyzerBase: analyzerBase{parent: parent},
		lock:         lock,
	}
}

func (la *lockAnalyzer) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	return visitInner(la, node)
}

func (la *lockAnalyzer) VisitTypeData(_ *ast.TypeData) ast.Visitor { return la }

func (la *lockAnalyzer) loc() diagnostics.Location { return la.lock.GetPosition() }

func (la *lockAnalyzer) ctx() *context.CompilerContext { return la.parent.ctx() }
func (la *lockAnalyzer) tyCtx() semtypes.Context       { return la.parent.tyCtx() }
func (la *lockAnalyzer) unimplementedErr(m string, l diagnostics.Location) {
	la.parent.ctx().Unimplemented(m, l)
}

func (la *lockAnalyzer) semanticErr(m string, l diagnostics.Location) {
	la.parent.ctx().SemanticError(m, l)
}

func (la *lockAnalyzer) syntaxErr(m string, l diagnostics.Location) {
	la.parent.ctx().SyntaxError(m, l)
}

func (la *lockAnalyzer) internalErr(m string, l diagnostics.Location) {
	la.parent.ctx().InternalError(m, l)
}

// enclosingLockAnalyzer walks the analyzer parent chain looking for a
// lockAnalyzer that is in the same closure as `a`. The search stops at
// the nearest functionAnalyzer because a function/lambda body is a
// fresh closure: locks visible above it belong to the surrounding
// closure and may not be held when this closure runs (a lambda value
// can escape its defining lock and be invoked later, and a default
// parameter expression is itself a closure invoked at each call).
//
// Two callers depend on this scoping:
//
//   - the nested-lock check in visitInner (a `lock` inside another
//     `lock` is rejected, but a `lock` inside a lambda inside a `lock`
//     is fine because the lambda is a separate closure);
//   - checkIsolatedModuleVarOutsideLock (an isolated module variable
//     read inside a lambda inside a `lock` is still "outside a lock"
//     because the surrounding lock does not protect the lambda body).
func enclosingLockAnalyzer(a analyzer) *lockAnalyzer {
	for cur := a; cur != nil; cur = cur.parentAnalyzer() {
		if lock, ok := cur.(*lockAnalyzer); ok {
			return lock
		}
		if _, isFn := cur.(*functionAnalyzer); isFn {
			return nil
		}
	}
	return nil
}

func (ca *constantAnalyzer) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		// Done
		return nil
	}
	switch n := node.(type) {
	case *ast.BLangIdentifier:
		return nil
	case *ast.BLangFunction:
		ca.semanticErr("function definition not allowed in constant expression", n.GetPosition())
		return nil
	case *ast.BLangWhile:
		ca.semanticErr("loop not allowed in constant expression", n.GetPosition())
		return nil
	case *ast.BLangIf:
		ca.semanticErr("if statement not allowed in constant expression", n.GetPosition())
		return nil
	case *ast.BLangReturn:
		ca.semanticErr("return statement not allowed in constant expression", n.GetPosition())
		return nil
	case *ast.BLangBreak:
		ca.semanticErr("break statement not allowed in constant expression", n.GetPosition())
		return nil
	case *ast.BLangContinue:
		ca.semanticErr("continue statement not allowed in constant expression", n.GetPosition())
		return nil
	case ast.TypeDescriptor:
	case *ast.BLangTypeDefinition:
		// We have set the type at constructor
		return nil
	case ast.BLangExpression:
		bLangExpr := n
		hasErrors := false
		validateConstantExpr(ca.ctx(), bLangExpr, func(e ast.BLangExpression) {
			ca.semanticErr("expression is not a constant expression", e.GetPosition())
			hasErrors = true
		})
		if hasErrors {
			return nil
		}
		analyzeActionOrExpression(ca, bLangExpr, ca.expectedType)
		return nil
	}
	return ca
}

func validateConstantExpr(ctx *context.CompilerContext, expr ast.BLangExpression, onNonConst func(ast.BLangExpression)) {
	switch e := expr.(type) {
	case *ast.BLangLiteral, *ast.BLangNumericLiteral:
		// always valid
	case *ast.BLangSimpleVarRef:
		sym := ctx.GetSymbol(e.Symbol())
		if vs, ok := sym.(model.ValueSymbol); ok && vs.IsConst() {
			return
		}
		onNonConst(expr)
	case *ast.BLangUnaryExpr:
		validateConstantExpr(ctx, e.Expr, onNonConst)
	case *ast.BLangTypeConversionExpr:
		validateConstantExpr(ctx, e.Expression, onNonConst)
	case *ast.BLangGroupExpr:
		validateConstantExpr(ctx, e.Expression, onNonConst)
	case *ast.BLangBinaryExpr:
		validateConstantExpr(ctx, e.LhsExpr, onNonConst)
		validateConstantExpr(ctx, e.RhsExpr, onNonConst)
	case *ast.BLangListConstructorExpr:
		for _, member := range e.Exprs {
			validateConstantExpr(ctx, member, onNonConst)
		}
	case *ast.BLangMappingConstructorExpr:
		for _, field := range e.Fields {
			if kv, ok := field.(*ast.BLangMappingKeyValueField); ok {
				validateConstantExpr(ctx, kv.ValueExpr, onNonConst)
			}
		}
	case *ast.BLangTemplateExpr:
		for _, ins := range e.Insertions {
			validateConstantExpr(ctx, ins, onNonConst)
		}
	case *ast.BLangAnnotAccessExpr:
		validateConstantExpr(ctx, e.Expr, onNonConst)
	case *ast.BLangXMLTemplateExpr:
		for _, ins := range e.Insertions {
			validateConstantExpr(ctx, ins, onNonConst)
		}
	default:
		onNonConst(expr)
	}
}

// validateResolvedType validates that a resolved expression type is compatible with the expected type
func validateResolvedType[A analyzer](a A, expr ast.BLangActionOrExpression, expectedType semtypes.SemType) bool {
	resolvedTy := expr.GetDeterminedType()
	if semtypes.IsZero(resolvedTy) {
		a.internalErr(fmt.Sprintf("expression type not resolved for %T", expr), expr.GetPosition())
		return false
	}

	if semtypes.IsZero(expectedType) {
		return true
	}

	ctx := a.tyCtx()
	if !semtypes.IsSubtype(ctx, resolvedTy, expectedType) {
		a.semanticErr(formatIncompatibleTypeMessage(ctx, expectedType, resolvedTy), expr.GetPosition())
		return false
	}
	if semtypes.IsNever(resolvedTy) {
		if !semtypes.IsNever(expectedType) {
			a.semanticErr(formatIncompatibleTypeMessage(ctx, expectedType, resolvedTy), expr.GetPosition())
			return false
		}
	}

	return true
}

func formatIncompatibleTypeMessage(ctx semtypes.Context, expectedType semtypes.SemType, actualType semtypes.SemType) string {
	return fmt.Sprintf("incompatible type: expected %s, got %s", semtypes.ToString(ctx, expectedType), semtypes.ToString(ctx, actualType))
}

func analyzeActionOrExpression[A analyzer](a A, expr ast.BLangActionOrExpression, expectedType semtypes.SemType) bool {
	switch expr := expr.(type) {
	case *ast.BLangLiteral:
		return validateResolvedType(a, expr, expectedType)

	case *ast.BLangNumericLiteral:
		return validateResolvedType(a, expr, expectedType)

	case *ast.BLangSimpleVarRef:
		return validateResolvedType(a, expr, expectedType)

	case *ast.BLangLocalVarRef, *ast.BLangConstRef:
		panic("not implemented")

	case *ast.BLangBinaryExpr:
		return analyzeBinaryExpr(a, expr, expectedType)

	case *ast.BLangUnaryExpr:
		return analyzeUnaryExpr(a, expr, expectedType)

	case *ast.BLangInvocation:
		return analyzeInvocation[A](a, expr, expectedType)

	case *ast.BLangIndexBasedAccess:
		return analyzeIndexBasedAccess(a, expr, expectedType)

	case *ast.BLangFieldBaseAccess:
		return analyzeFieldBasedAccess(a, expr, expectedType)
	// Collections and Groups - validate members and result
	case *ast.BLangListConstructorExpr:
		return analyzeListConstructorExpr(a, expr, expectedType)

	case *ast.BLangMappingConstructorExpr:
		return analyzeMappingConstructorExpr(a, expr, expectedType)

	case *ast.BLangErrorConstructorExpr:
		return analyzeErrorConstructorExpr(a, expr, expectedType)

	case *ast.BLangGroupExpr:
		return analyzeActionOrExpression(a, expr.Expression, expectedType)

	case *ast.BLangQueryExpr:
		return analyzeQueryExpr(a, expr, expectedType)

	case *ast.BLangWildCardBindingPattern:
		return validateResolvedType(a, expr, expectedType)

	case *ast.BLangTypeConversionExpr:
		return validateTypeConversionExpr(a, expr, expectedType)

	case *ast.BLangTypeTestExpr:
		return validateResolvedType(a, expr, expectedType)
	case *ast.BLangCheckedExpr:
		return analyzeCheckedExpr(a, expr, expectedType)
	case *ast.BLangCheckPanickedExpr:
		return analyzeCheckPanickedExpr(a, expr, expectedType)
	case *ast.BLangTrapExpr:
		return analyzeTrapExpr(a, expr, expectedType)
	case *ast.BLangNamedArgsExpression:
		return analyzeActionOrExpression(a, expr.Expr, expectedType)
	case *ast.BLangNewExpression:
		return analyzeNewExpression(a, expr, expectedType)
	case *ast.BLangLambdaFunction:
		return analyzeLambdaFunction(a, expr)
	case *ast.BLangRemoteMethodCallAction:
		return analyzeInvocation(a, expr, expectedType)
	case *ast.BLangClientResourceAccessAction:
		return analyzeClientResourceAccessAction(a, expr, expectedType)
	case *ast.BLangInferredTypedescDefault:
		return validateResolvedType(a, expr, expectedType)
	case *ast.BLangTypedescExpr:
		return validateResolvedType(a, expr, expectedType)
	case *ast.BLangAnnotAccessExpr:
		// Annotation access is only valid on a typedesc value, so the receiver
		// is analyzed with typedesc as its expected type.
		if !analyzeActionOrExpression(a, expr.Expr, semtypes.TYPEDESC) {
			return false
		}
		return validateResolvedType(a, expr, expectedType)
	case *ast.BLangXMLElementLiteral:
		for i := range expr.Attrs {
			attr := &expr.Attrs[i]
			if attr.Value != nil && !analyzeActionOrExpression(a, attr.Value, semtypes.STRING) {
				return false
			}
		}
		return validateResolvedType(a, expr, expectedType)
	case *ast.BLangXMLSequenceLiteral, *ast.BLangXMLPILiteral, *ast.BLangXMLCommentLiteral, *ast.BLangXMLTextLiteral:
		return validateResolvedType(a, expr, expectedType)
	case *ast.BLangTemplateExpr:
		return analyzeTemplateExpr(a, expr, expectedType)
	case *ast.BLangXMLTemplateExpr:
		return analyzeXMLTemplateExpr(a, expr, expectedType)
	case *ast.BLangXMLAttribute:
		// XML attributes are metadata on elements and should not be analyzed as standalone expressions
		// Their values are already analyzed as part of XMLElement processing
		return validateResolvedType(a, expr, expectedType)
	default:
		a.internalErr("unexpected expression type: "+reflect.TypeOf(expr).String(), expr.GetPosition())
		return false
	}
}

func analyzeCheckedExpr[A analyzer](a A, expr *ast.BLangCheckedExpr, expectedType semtypes.SemType) bool {
	if !analyzeActionOrExpression(a, expr.Expr, semtypes.SemType{}) {
		return false
	}
	retTy := expectedReturnType(a)
	if !semtypes.IsZero(retTy) {
		exprTy := expr.Expr.GetDeterminedType()
		errorPart := semtypes.Intersect(exprTy, semtypes.ERROR)
		if !semtypes.IsEmpty(a.tyCtx(), errorPart) {
			if !semtypes.IsSubtype(a.tyCtx(), errorPart, retTy) {
				a.ctx().SemanticError("error type of check expression is not a subtype of the enclosing function's return type", expr.GetPosition())
			}
		}
	}
	return validateResolvedType(a, expr, expectedType)
}

var templateInsertionAllowedTypes = semtypes.Diff(semtypes.SIMPLE_OR_STRING, semtypes.NIL)

func analyzeTemplateExpr[A analyzer](a A, expr *ast.BLangTemplateExpr, expectedType semtypes.SemType) bool {
	for _, ins := range expr.Insertions {
		if !analyzeActionOrExpression(a, ins, templateInsertionAllowedTypes) {
			return false
		}
	}
	return validateResolvedType(a, expr, expectedType)
}

func analyzeXMLTemplateExpr[A analyzer](a A, expr *ast.BLangXMLTemplateExpr, expectedType semtypes.SemType) bool {
	if len(expr.InsertionKinds) != len(expr.Insertions) {
		a.internalError(fmt.Sprintf("xml template insertion kind count mismatch: got %d kinds for %d insertions", len(expr.InsertionKinds), len(expr.Insertions)), expr.GetPosition())
		return false
	}
	for i, ins := range expr.Insertions {
		allowed := xmlTemplateInsertionAllowedTypes(expr.InsertionKinds[i])
		if !analyzeActionOrExpression(a, ins, allowed) {
			return false
		}
	}
	return validateResolvedType(a, expr, expectedType)
}

func xmlTemplateInsertionAllowedTypes(kind ast.XMLTemplateInsertionKind) semtypes.SemType {
	if kind == ast.XMLTemplateInsertionKindContent {
		return semtypes.Union(templateInsertionAllowedTypes, semtypes.XML)
	}
	return templateInsertionAllowedTypes
}

func analyzeTrapExpr[A analyzer](a A, expr *ast.BLangTrapExpr, expectedType semtypes.SemType) bool {
	if !analyzeActionOrExpression(a, expr.Expr, semtypes.SemType{}) {
		return false
	}
	return validateResolvedType(a, expr, expectedType)
}

func analyzeCheckPanickedExpr[A analyzer](a A, expr *ast.BLangCheckPanickedExpr, expectedType semtypes.SemType) bool {
	if !analyzeActionOrExpression(a, expr.Expr, semtypes.SemType{}) {
		return false
	}
	return validateResolvedType(a, expr, expectedType)
}

type queryExprAnalysisClauses struct {
	fromClause       *ast.BLangFromClause
	selectClause     *ast.BLangSelectClause
	collectClause    *ast.BLangCollectClause
	onConflictClause *ast.BLangOnConflictClause
	lastClauseIndex  int
}

func queryExprClausesForAnalysis[A analyzer](
	a A,
	queryExpr *ast.BLangQueryExpr,
) (queryExprAnalysisClauses, bool) {
	if len(queryExpr.QueryClauseList) < 2 {
		a.internalErr("query expression shape should have been validated during type resolution", queryExpr.GetPosition())
		return queryExprAnalysisClauses{}, false
	}

	fromClause := queryExpr.QueryClauseList[0].(*ast.BLangFromClause)
	lastClauseIndex := len(queryExpr.QueryClauseList) - 1
	var onConflictClause *ast.BLangOnConflictClause
	if clause, isOnConflict := queryExpr.QueryClauseList[lastClauseIndex].(*ast.BLangOnConflictClause); isOnConflict {
		onConflictClause = clause
		lastClauseIndex--
	}
	if lastClauseIndex < 1 {
		a.internalErr("query expression shape should have been validated during type resolution", queryExpr.GetPosition())
		return queryExprAnalysisClauses{}, false
	}

	var (
		selectClause  *ast.BLangSelectClause
		collectClause *ast.BLangCollectClause
		ok            bool
	)
	if selectClause, ok = queryExpr.QueryClauseList[lastClauseIndex].(*ast.BLangSelectClause); !ok {
		collectClause, ok = queryExpr.QueryClauseList[lastClauseIndex].(*ast.BLangCollectClause)
	}
	if !ok {
		a.internalErr("query expression shape should have been validated during type resolution", queryExpr.GetPosition())
		return queryExprAnalysisClauses{}, false
	}
	return queryExprAnalysisClauses{
		fromClause:       fromClause,
		selectClause:     selectClause,
		collectClause:    collectClause,
		onConflictClause: onConflictClause,
		lastClauseIndex:  lastClauseIndex,
	}, true
}

func analyzeQueryExpr[A analyzer](a A, queryExpr *ast.BLangQueryExpr, expectedType semtypes.SemType) bool {
	// Query clause ordering and shape are validated during type resolution.
	clauses, ok := queryExprClausesForAnalysis(a, queryExpr)
	if !ok {
		return false
	}
	if !analyzeActionOrExpression(a, clauses.fromClause.Collection, semtypes.SemType{}) {
		return false
	}
	orderedTy := semtypes.CreateOrdered(a.tyCtx())

	for i := 1; i < clauses.lastClauseIndex; i++ {
		switch clause := queryExpr.QueryClauseList[i].(type) {
		case *ast.BLangJoinClause:
			if !analyzeActionOrExpression(a, clause.Collection, semtypes.SemType{}) {
				return false
			}
			if clause.OnClause.OnExpr == nil || clause.OnClause.EqualsExpr == nil {
				a.internalErr("join clause shape should have been validated during type resolution", clause.GetPosition())
				return false
			}
			if !analyzeActionOrExpression(a, clause.OnClause.OnExpr, semtypes.SemType{}) {
				return false
			}
			if !analyzeActionOrExpression(a, clause.OnClause.EqualsExpr, semtypes.SemType{}) {
				return false
			}
		case *ast.BLangLetClause:
			for i := range clause.LetVarDeclarations {
				varDef := &clause.LetVarDeclarations[i]
				if varDef.Var == nil || varDef.Var.Expr == nil {
					a.semanticErr("let clause supports only initialized simple variable declarations", clause.GetPosition())
					return false
				}
				var expectedType semtypes.SemType
				if ast.SymbolIsSet(varDef.Var) {
					expectedType = a.ctx().SymbolType(varDef.Var.Symbol())
				}
				if !analyzeActionOrExpression(a, varDef.Var.Expr.(ast.BLangExpression), expectedType) {
					return false
				}
			}
		case *ast.BLangWhereClause, *ast.BLangLimitClause:
			// Query clause type and shape validation already happen in type resolution.
		case *ast.BLangGroupByClause:
			anyData := semtypes.CreateAnydata(a.tyCtx())
			for j := range clause.GroupingKeyList {
				groupingKey := &clause.GroupingKeyList[j]
				switch {
				case groupingKey.VariableRef != nil:
					if !analyzeActionOrExpression(a, groupingKey.VariableRef, anyData) {
						return false
					}
				case groupingKey.VariableDef != nil:
					varDef := groupingKey.VariableDef
					if varDef.Var == nil || varDef.Var.Expr == nil {
						a.semanticErr("group by clause supports only initialized simple variable declarations", clause.GetPosition())
						return false
					}
					var expectedType semtypes.SemType
					if ast.SymbolIsSet(varDef.Var) {
						expectedType = a.ctx().SymbolType(varDef.Var.Symbol())
					}
					if !analyzeActionOrExpression(a, varDef.Var.Expr.(ast.BLangExpression), expectedType) {
						return false
					}
					if !semtypes.IsZero(expectedType) && !semtypes.IsSubtype(a.tyCtx(), expectedType, anyData) {
						a.semanticErr("grouping key expression must be a subtype of anydata", groupingKey.GetPosition())
						return false
					}
				default:
					a.internalErr("group by clause shape should have been validated during type resolution", groupingKey.GetPosition())
					return false
				}
			}
		case *ast.BLangOrderByClause:
			for j := range clause.OrderByKeyList {
				orderKey := &clause.OrderByKeyList[j]
				if !analyzeActionOrExpression(a, orderKey.Expression, orderedTy) {
					return false
				}
			}
		}
	}

	if clauses.selectClause != nil {
		selectExpectedTy := querySelectExpectedType(
			a.tyCtx(),
			a.tyCtx().Env(),
			queryExpr.QueryConstructType,
			expectedType,
		)
		if semtypes.IsZero(selectExpectedTy) && queryExpr.QueryConstructType == ast.TypeKind_MAP {
			selectExpectedTy = mapQuerySelectExpectedType(a.tyCtx().Env())
		}
		if !analyzeActionOrExpression(a, clauses.selectClause.Expression, selectExpectedTy) {
			return false
		}
	} else {
		if queryExpr.QueryConstructType != ast.TypeKind_NONE {
			a.semanticErr("query construct types cannot be used with collect clause", clauses.collectClause.GetPosition())
			return false
		}
		if !analyzeActionOrExpression(a, clauses.collectClause.Expression, semtypes.SemType{}) {
			return false
		}
	}

	if clauses.onConflictClause != nil {
		if queryExpr.QueryConstructType != ast.TypeKind_MAP {
			a.semanticErr("on conflict clause is supported only for map query construct type", clauses.onConflictClause.GetPosition())
			return false
		}
		if !analyzeActionOrExpression(a, clauses.onConflictClause.Expression, semtypes.Union(semtypes.ERROR, semtypes.NIL)) {
			return false
		}
	}

	return validateResolvedType(a, queryExpr, expectedType)
}

func analyzeNewExpression[A analyzer](a A, expr *ast.BLangNewExpression, expectedType semtypes.SemType) bool {
	if ast.IsStreamNewExpression(expr) {
		return analyzeStreamNewExpression(a, expr, expectedType)
	}
	return validateResolvedType(a, expr, expectedType)
}

func analyzeStreamNewExpression[A analyzer](a A, expr *ast.BLangNewExpression, expectedType semtypes.SemType) bool {
	cx := a.tyCtx()
	streamTy := expr.GetDeterminedType()
	valueTy := semtypes.StreamValueType(cx, streamTy)
	completionTy := semtypes.StreamCompletionType(cx, streamTy)
	if semtypes.IsZero(valueTy) || semtypes.IsZero(completionTy) {
		a.internalErr("failed to extract stream type parameters", expr.GetPosition())
		return false
	}
	implTy := semtypes.CreateStreamImplementorType(cx, valueTy, completionTy)
	arg := expr.ArgsExprs[0]
	if !analyzeActionOrExpression(a, arg, implTy) {
		return false
	}
	if !validateStreamCloseMethod(a, arg, completionTy) {
		return false
	}
	return validateResolvedType(a, expr, expectedType)
}

func validateStreamCloseMethod[A analyzer](a A, impl ast.BLangExpression, completionTy semtypes.SemType) bool {
	cx := a.tyCtx()
	implTy := impl.GetDeterminedType()
	closeName := semtypes.StringConst("close")
	kindTy := semtypes.ObjectMemberKind(cx, closeName, implTy)
	if !semtypes.IsSubtype(cx, kindTy, semtypes.StringConst("method")) {
		return true
	}
	visibilityTy := semtypes.ObjectMemberVisibility(cx, closeName, implTy)
	if !semtypes.IsSubtype(cx, visibilityTy, semtypes.StringConst("public")) {
		a.semanticErr("stream implementor close method must be public", impl.GetPosition())
		return false
	}
	closeFnTy := semtypes.ObjectMemberType(cx, closeName, implTy)
	paramListDefn := semtypes.NewListDefinition()
	emptyParamList := paramListDefn.DefineListTypeWrapped(cx.Env(), nil, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	returnTy := semtypes.FunctionReturnType(cx, closeFnTy, emptyParamList)
	expectedReturnTy := semtypes.Union(completionTy, semtypes.NIL)
	if semtypes.IsZero(returnTy) || !semtypes.IsSubtype(cx, returnTy, expectedReturnTy) {
		a.semanticErr("stream implementor close method is incompatible", impl.GetPosition())
		return false
	}
	return true
}

func enclosingFunctionIsIsolated(a analyzer) bool {
	fa := enclosingFunctionAnalyzer(a)
	if fa == nil {
		return false
	}
	fn, ok := fa.function.(invokableSignatureNode)
	return ok && fn.IsIsolated()
}

func analyzeLambdaFunction[A analyzer](a A, expr *ast.BLangLambdaFunction) bool {
	fa := initializeFunctionAnalyzer(a, expr.Function)
	fn := expr.Function
	if fn.IsIsolated() && fn.Body != nil && !enclosingFunctionIsIsolated(a) {
		validateIsolatedCapture(a, enclosingFunctionLocals(a), fn.Body.(ast.BLangNode))
	}
	// Walk params + body directly rather than the BLangFunction node
	// itself; otherwise the walker's first visit on BLangFunction would
	// re-enter visitInner's BLangFunction case and re-initialize the
	// analyzer, double-firing all per-init checks.
	for i := range fn.RequiredParams {
		ast.Walk(fa, &fn.RequiredParams[i])
	}
	if fn.RestParam != nil {
		ast.Walk(fa, fn.RestParam.(ast.BLangNode))
	}
	if fn.Body != nil {
		ast.Walk(fa, fn.GetBody().(ast.BLangNode))
	}
	return true
}

func validateTypeConversionExpr[A analyzer](a A, expr *ast.BLangTypeConversionExpr, expectedType semtypes.SemType) bool {
	if !analyzeActionOrExpression(a, expr.Expression, semtypes.SemType{}) {
		return false
	}
	exprTy := expr.Expression.GetDeterminedType()
	targetType := expr.TypeDescriptor.GetDeterminedType()
	intersection := semtypes.Intersect(exprTy, targetType)
	if semtypes.IsEmpty(a.tyCtx(), intersection) && !hasPotentialNumericConversions(exprTy, targetType) {
		a.semanticErr("impossible type conversion, intersection is empty", expr.GetPosition())
		return false
	}
	if !semtypes.IsZero(expectedType) && !semtypes.IsSubtype(a.tyCtx(), targetType, expectedType) {
		a.semanticErr(formatIncompatibleTypeMessage(a.tyCtx(), expectedType, targetType), expr.GetPosition())
		return false
	}
	return validateResolvedType(a, expr, expectedType)
}

func hasPotentialNumericConversions(exprTy, targetType semtypes.SemType) bool {
	if !semtypes.SingleNumericType(targetType).IsPresent() {
		return false
	}
	return semtypes.ContainsBasicType(exprTy, semtypes.NUMBER)
}

func analyzeFieldBasedAccess[A analyzer](a A, expr *ast.BLangFieldBaseAccess, expectedType semtypes.SemType) bool {
	if !analyzeActionOrExpression(a, expr.Expr, semtypes.SemType{}) {
		return false
	}
	return validateResolvedType(a, expr, expectedType)
}

func analyzeIndexBasedAccess[A analyzer](a A, expr *ast.BLangIndexBasedAccess, expectedType semtypes.SemType) bool {
	// Validate container expression
	containerExpr := expr.Expr
	if !analyzeActionOrExpression(a, containerExpr, semtypes.SemType{}) {
		return false
	}
	containerExprTy := containerExpr.GetDeterminedType()

	var keyExprExpectedType semtypes.SemType
	ctx := a.tyCtx()
	if semtypes.IsSubtype(ctx, containerExprTy, semtypes.LIST) ||
		semtypes.IsSubtype(ctx, containerExprTy, semtypes.STRING) ||
		semtypes.IsSubtype(ctx, containerExprTy, semtypes.XML) {
		keyExprExpectedType = semtypes.INT
	} else if semtypes.IsSubtype(ctx, containerExprTy, semtypes.TABLE) {
		a.unimplementedErr("table not supported", expr.GetPosition())
		return false
	} else if semtypes.IsSubtype(ctx, containerExprTy, semtypes.Union(semtypes.NIL, semtypes.MAPPING)) {
		keyExprExpectedType = semtypes.STRING
	} else {
		a.semanticErr("incompatible type for index based access", expr.GetPosition())
		return false
	}

	keyExpr := expr.IndexExpr
	if !analyzeActionOrExpression(a, keyExpr, keyExprExpectedType) {
		return false
	}

	return validateResolvedType(a, expr, expectedType)
}

func analyzeListConstructorExpr[A analyzer](a A, expr *ast.BLangListConstructorExpr, expectedType semtypes.SemType) bool {
	// The type resolver has already selected the inherent type and re-resolved members
	// with per-member expected types. We only need to validate members here.
	lat := expr.AtomicType
	memberIndex := 0
	restMember := false
	for i, memberExpr := range expr.Exprs {
		memberExpectedType := lat.MemberAtInnerVal(memberIndex)
		if restMember || expr.IsSpreadMember(i) {
			memberExpectedType = lat.Rest()
		}
		if expr.IsSpreadMember(i) {
			memberExpectedType = listOfMemberType(a.ctx().GetTypeEnv(), memberExpectedType)
			if !analyzeActionOrExpression(a, memberExpr, memberExpectedType) {
				return false
			}
			restMember = true
			memberIndex = lat.Members.FixedLength
			continue
		}
		if !analyzeActionOrExpression(a, memberExpr, memberExpectedType) {
			return false
		}
		if !restMember {
			memberIndex++
		}
	}
	for i := memberIndex; i < lat.Members.FixedLength; i++ {
		memberTy := lat.MemberAtInnerVal(i)
		if _, ok := semtypes.FillerValue(a.tyCtx(), memberTy); !ok {
			a.semanticErr(fmt.Sprintf("missing required member at index %d: type '%s' has no filler value", i, semtypes.ToString(a.tyCtx(), memberTy)), expr.GetPosition())
			return false
		}
	}
	return validateResolvedType(a, expr, expectedType)
}

func listOfMemberType(env semtypes.Env, memberTy semtypes.SemType) semtypes.SemType {
	ld := semtypes.NewListDefinition()
	return ld.DefineListTypeWrappedWithEnvSemType(env, memberTy)
}

func analyzeMappingConstructorExpr[A analyzer](a A, expr *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) bool {
	// The type resolver has already selected the inherent type and re-resolved field values
	// with per-field expected types. We only need to validate fields here.
	mat := expr.AtomicType
	hasValue := make(map[string]bool, len(expr.Fields)+len(expr.FieldDefaults))
	for _, fd := range expr.FieldDefaults {
		hasValue[fd.FieldName] = true
	}
	seen := make(map[string]bool, len(expr.Fields))
	namedFields := make(map[string]bool, len(mat.Names))
	for _, n := range mat.Names {
		namedFields[n] = true
	}
	for _, f := range expr.Fields {
		kv := f.(*ast.BLangMappingKeyValueField)
		keyName := recordKeyName(kv.Key)
		if seen[keyName] {
			a.semanticErr(fmt.Sprintf("duplicate key '%s' in mapping constructor", keyName), kv.Key.GetPosition())
			return false
		}
		seen[keyName] = true
		// For record type desc (ie len(mat.Names) > 0) if the key is not a string literal it must be
		// nameed field
		if kv.Key.Kind == ast.MappingKeyIdentifier && len(mat.Names) > 0 && !namedFields[keyName] {
			a.semanticErr(fmt.Sprintf("identifier '%s' cannot be used as a key for a rest field; use a string literal instead", keyName), kv.Key.GetPosition())
			return false
		}
		hasValue[keyName] = true
		fieldExpectedType := mat.FieldInnerVal(keyName)
		if !analyzeActionOrExpression(a, kv.ValueExpr, fieldExpectedType) {
			return false
		}
	}
	for _, name := range mat.Names {
		if hasValue[name] {
			continue
		}
		if mat.IsOptional(a.tyCtx(), name) {
			continue
		}
		a.semanticErr(fmt.Sprintf("missing non-defaultable required record field '%s'", name), expr.GetPosition())
		return false
	}
	return validateResolvedType(a, expr, expectedType)
}

func analyzeErrorConstructorExpr[A analyzer](a A, expr *ast.BLangErrorConstructorExpr, expectedType semtypes.SemType) bool {
	argCount := len(expr.PositionalArgs)
	if argCount < 1 || argCount > 2 {
		a.semanticErr("error constructor must have at least 1 and at most 2 positional arguments", expr.GetPosition())
		return false
	}
	tyCtx := a.tyCtx()

	msgArg := expr.PositionalArgs[0]
	if !analyzeActionOrExpression(a, msgArg, semtypes.STRING) {
		return false
	}
	mat, ok := semtypes.ErrorDetailAtomicType(tyCtx, expr.DeterminedType)
	if !ok {
		a.unimplementedErr("non-atomic detail types not supported", expr.GetPosition())
		return false
	}
	seen := make(map[string]bool, len(expr.NamedArgs))
	clonableTy := semtypes.CreateCloneable(tyCtx)
	for _, namedArg := range expr.NamedArgs {
		name := namedArg.Name.GetValue()
		if seen[name] {
			a.semanticErr(fmt.Sprintf("duplicate named argument '%s' in error constructor", name), namedArg.GetPosition())
			return false
		}
		seen[name] = true
		fieldType := mat.FieldInnerVal(name)
		if !analyzeActionOrExpression(a, namedArg.Expr, fieldType) {
			return false
		}
		if !semtypes.IsSubtype(tyCtx, namedArg.Expr.GetDeterminedType(), clonableTy) {
			a.semanticErr("named arguments must be subtypes of cloneable", namedArg.GetPosition())
			return false
		}
	}

	// Every field in the atom must be provided
	for _, name := range mat.Names {
		if !seen[name] {
			a.semanticErr(fmt.Sprintf("missing required field '%s' in error constructor", name), expr.GetPosition())
			return false
		}
	}

	if argCount == 2 {
		causeArg := expr.PositionalArgs[1]
		if !analyzeActionOrExpression(a, causeArg, semtypes.Union(semtypes.ERROR, semtypes.NIL)) {
			return false
		}
	}

	return validateResolvedType(a, expr, expectedType)
}

func analyzeUnaryExpr[A analyzer](a A, unaryExpr *ast.BLangUnaryExpr, expectedType semtypes.SemType) bool {
	if !analyzeActionOrExpression(a, unaryExpr.Expr, semtypes.SemType{}) {
		return false
	}

	exprTy := unaryExpr.Expr.GetDeterminedType()
	// Strip nil for nil-lifted numeric/bitwise unary operations
	underlyingTy := exprTy
	if semtypes.ContainsBasicType(exprTy, semtypes.NIL) {
		underlyingTy = semtypes.Diff(exprTy, semtypes.NIL)
	}

	switch unaryExpr.GetOperatorKind() {
	case model.OperatorKind_ADD, model.OperatorKind_SUB, model.OperatorKind_BITWISE_COMPLEMENT:
		if !isNumericType(a.tyCtx(), underlyingTy) {
			a.semanticErr(fmt.Sprintf("expect numeric type for %s", string(unaryExpr.GetOperatorKind())), unaryExpr.GetPosition())
			return false
		}
	case model.OperatorKind_NOT:
		if !semtypes.IsSubtype(a.tyCtx(), exprTy, semtypes.BOOLEAN) {
			a.semanticErr(fmt.Sprintf("expect boolean type for %s", string(unaryExpr.GetOperatorKind())), unaryExpr.GetPosition())
			return false
		}
	default:
		a.semanticErr(fmt.Sprintf("unsupported unary operator: %s", string(unaryExpr.GetOperatorKind())), unaryExpr.GetPosition())
		return false
	}

	return validateResolvedType(a, unaryExpr, expectedType)
}

func analyzeBinaryExpr[A analyzer](a A, binaryExpr *ast.BLangBinaryExpr, expectedType semtypes.SemType) bool {
	// Validate both operand expressions
	if !analyzeActionOrExpression(a, binaryExpr.LhsExpr, semtypes.SemType{}) {
		return false
	}
	if !analyzeActionOrExpression(a, binaryExpr.RhsExpr, semtypes.SemType{}) {
		return false
	}

	// Get operand types
	lhsTy := binaryExpr.LhsExpr.GetDeterminedType()
	rhsTy := binaryExpr.RhsExpr.GetDeterminedType()

	ctx := a.tyCtx()
	// Perform semantic validation based on operator type
	if isEqualityExpr(binaryExpr) {
		// For equality operators, ensure types have non-empty intersection
		intersection := semtypes.Intersect(lhsTy, rhsTy)
		if semtypes.IsEmpty(ctx, intersection) {
			a.semanticErr(fmt.Sprintf("incompatible types for %s", string(binaryExpr.GetOperatorKind())), binaryExpr.GetPosition())
			return false
		}
		switch binaryExpr.GetOperatorKind() {
		case model.OperatorKind_EQUAL, model.OperatorKind_NOT_EQUAL:
			anyData := semtypes.CreateAnydata(ctx)
			if !semtypes.IsSubtype(ctx, lhsTy, anyData) && !semtypes.IsSubtype(ctx, rhsTy, anyData) {
				a.semanticErr(fmt.Sprintf("expect anydata types for %s", string(binaryExpr.GetOperatorKind())), binaryExpr.GetPosition())
				return false
			}
		}
	} else if isBitWiseExpr(binaryExpr) {
		if !analyzeBitWiseExpr(a, binaryExpr, lhsTy, rhsTy) {
			return false
		}
	} else if isRangeExpr(binaryExpr) {
		if !semtypes.IsSubtype(a.tyCtx(), lhsTy, semtypes.INT) || !semtypes.IsSubtype(a.tyCtx(), rhsTy, semtypes.INT) {
			a.semanticErr(fmt.Sprintf("expect int types for %s", string(binaryExpr.GetOperatorKind())), binaryExpr.GetPosition())
			return false
		}
	} else if isShiftExpr(binaryExpr) {
		if !analyzeShiftExpr(a, lhsTy, rhsTy) {
			return false
		}
	} else if isLogicalExpression(binaryExpr) {
		if !semtypes.IsSubtype(a.tyCtx(), lhsTy, semtypes.BOOLEAN) || !semtypes.IsSubtype(a.tyCtx(), rhsTy, semtypes.BOOLEAN) {
			a.semanticErr(fmt.Sprintf("expect boolean types for %s", string(binaryExpr.GetOperatorKind())), binaryExpr.GetPosition())
			return false
		}
	}
	// for nil lifting expression we do semantic analysis as part of type resolver
	// Validate the resolved result type against expected type
	return validateResolvedType(a, binaryExpr, expectedType)
}

func analyzeBitWiseExpr[A analyzer](a A, binaryExpr *ast.BLangBinaryExpr, lhsTy, rhsTy semtypes.SemType) bool {
	ctx := a.tyCtx()
	if semtypes.ContainsBasicType(lhsTy, semtypes.NIL) || semtypes.ContainsBasicType(rhsTy, semtypes.NIL) {
		lhsTy = semtypes.Diff(lhsTy, semtypes.NIL)
		rhsTy = semtypes.Diff(rhsTy, semtypes.NIL)
	}
	if !semtypes.IsSubtype(ctx, lhsTy, semtypes.INT) || !semtypes.IsSubtype(ctx, rhsTy, semtypes.INT) {
		a.semanticErr("expect integer types for bitwise operators", binaryExpr.GetPosition())
		return false
	}
	return true
}

func analyzeShiftExpr[A analyzer](a A, lhsTy, rhsTy semtypes.SemType) bool {
	ctx := a.tyCtx()
	if semtypes.ContainsBasicType(lhsTy, semtypes.NIL) || semtypes.ContainsBasicType(rhsTy, semtypes.NIL) {
		lhsTy = semtypes.Diff(lhsTy, semtypes.NIL)
		rhsTy = semtypes.Diff(rhsTy, semtypes.NIL) //nolint:staticcheck,ineffassign // rhsTy will be used when nil-lifted binary ops are fully implemented
	}
	if !semtypes.IsSubtype(ctx, lhsTy, semtypes.INT) || !semtypes.IsSubtype(ctx, rhsTy, semtypes.INT) {
		return false
	}
	return true
}

type invocable interface {
	ast.BLangActionOrExpression
	ResolvedSymbol() model.SymbolRef
	SetResolvedSymbol(model.SymbolRef)
	Receiver() ast.BLangExpression
	CallArgs() []ast.BLangExpression
	SetCallArgs([]ast.BLangExpression)
	GetName() ast.IdentifierNode
	SetRawSymbol(model.Symbol)
}

func analyzeInvocation[A analyzer](a A, inv invocable, expectedType semtypes.SemType) bool {
	if ast.IsStreamOperation(inv) {
		return analyzeStreamOperation(a, inv.(*ast.BLangInvocation), expectedType)
	}
	symbol := inv.ResolvedSymbol()
	// Skip invocations that failed type resolution — an unresolved dependently-typed
	// symbol still sits in the invocation, but has no usable semtype.
	if _, isDep := a.ctx().GetSymbol(symbol).(model.DependentlyTypedFunctionSymbol); isDep {
		return false
	}
	fnTy := a.ctx().SymbolType(symbol)
	paramListTy := semtypes.FunctionParamListType(a.tyCtx(), fnTy)

	fnSymbol, isDirectCall := a.ctx().GetSymbol(symbol).(model.FunctionSymbol)
	// TODO: ideally we need to unify these when we no longer has restrictions on lambdas
	if !isDirectCall {
		if invocation, ok := inv.(*ast.BLangInvocation); ok {
			return analyzeLambdaInvocation(a, invocation, paramListTy, expectedType)
		}
		a.internalErr("expected function symbol", inv.GetPosition())
		return false
	}
	return analyzeDirectInvocation(a, inv, fnSymbol, paramListTy, expectedType)
}

// Path computed segments are typed against rmSym.PathType() during type
// resolution, not against the function parameter list, so we walk them
// here independently of the call's argument analysis.
func analyzeClientResourceAccessAction[A analyzer](a A, expr *ast.BLangClientResourceAccessAction, expectedType semtypes.SemType) bool {
	if !analyzeActionOrExpression(a, expr.Expr, semtypes.CreateClientObject(a.tyCtx())) {
		return false
	}
	pathType := resolvedResourceMethodPathType(a, expr)
	for i := range expr.Path {
		seg := &expr.Path[i]
		if seg.Kind != ast.ResourceAccessSegmentComputed {
			continue
		}
		segExpectedTy := resourcePathSegmentExpectedType(a.tyCtx(), pathType, i)
		if !analyzeActionOrExpression(a, seg.Expr, segExpectedTy) {
			return false
		}
	}
	return analyzeInvocation(a, expr, expectedType)
}

func resolvedResourceMethodPathType[A analyzer](a A, expr *ast.BLangClientResourceAccessAction) semtypes.SemType {
	ref := expr.MethodSymbol()
	rmSym, ok := a.getSymbol(ref).(*model.ResourceMethodSymbol)
	if !ok {
		return semtypes.SemType{}
	}
	return rmSym.PathListType()
}

func resourcePathSegmentExpectedType(ctx semtypes.Context, pathType semtypes.SemType, index int) semtypes.SemType {
	return semtypes.ListMemberTypeInnerVal(ctx, pathType, semtypes.IntConst(int64(index)))
}

func analyzeStreamOperation[A analyzer](a A, invocation *ast.BLangInvocation, expectedType semtypes.SemType) bool {
	if len(invocation.ArgExprs) != 0 {
		a.semanticErr("stream method '"+invocation.Name.Value+"' takes no arguments", invocation.GetPosition())
		return false
	}
	if invocation.Expr != nil {
		if !analyzeActionOrExpression(a, invocation.Expr, semtypes.SemType{}) {
			return false
		}
	}
	return validateResolvedType(a, invocation, expectedType)
}

func analyzeDirectInvocation[A analyzer](a A, inv invocable, fnSymbol model.FunctionSymbol, paramListTy, expectedType semtypes.SemType) bool {
	signature := fnSymbol.Signature()
	tyCtx := a.tyCtx()
	for i, arg := range inv.CallArgs() {
		switch arg := arg.(type) {
		case *ast.BLangNamedArgsExpression:
			name := arg.Name.Value
			targetIndex := -1
			for j, each := range signature.ParamNames {
				if each == name {
					targetIndex = j
					break
				}
			}
			key := semtypes.IntConst(int64(targetIndex))
			if !analyzeActionOrExpression(a, arg, semtypes.ListMemberTypeInnerVal(tyCtx, paramListTy, key)) {
				return false
			}
		default:
			key := semtypes.IntConst(int64(i))
			if !analyzeActionOrExpression(a, arg, semtypes.ListMemberTypeInnerVal(tyCtx, paramListTy, key)) {
				return false
			}
		}
	}

	return validateResolvedType(a, inv, expectedType)
}

func analyzeLambdaInvocation[A analyzer](a A, invocation *ast.BLangInvocation, paramListTy, expectedType semtypes.SemType) bool {
	tyCtx := a.tyCtx()

	for i, arg := range invocation.ArgExprs {
		key := semtypes.IntConst(int64(i))
		if !analyzeActionOrExpression(a, arg, semtypes.ListMemberTypeInnerVal(tyCtx, paramListTy, key)) {
			return false
		}
	}

	return validateResolvedType(a, invocation, expectedType)
}

func analyzeSimpleVariableDef[A analyzer](a A, simpleVariableDef *ast.BLangSimpleVariableDef) bool {
	variable := simpleVariableDef.GetVariable().(*ast.BLangSimpleVariable)
	expectedType := variable.GetDeterminedType()
	if variable.GetName().GetValue() == string(model.IGNORE) {
		if !semtypes.IsSubtype(a.tyCtx(), expectedType, semtypes.ANY) {
			a.semanticErr("wildcard binding pattern type must be a subtype of 'any'", variable.GetPosition())
			return false
		}
	}
	if ast.SymbolIsSet(variable) {
		symbolType := a.ctx().SymbolType(variable.Symbol())
		if !semtypes.IsZero(symbolType) {
			expectedType = symbolType
		}
	}
	if variable.Expr != nil && !analyzeActionOrExpression(a, variable.Expr, expectedType) {
		return false
	}
	setExpectedType(simpleVariableDef, expectedType)
	return true
}

func visitInner[A analyzer](a A, node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangLambdaFunction:
		// Lambdas are analyzed exactly once via analyzeLambdaFunction
		// (called from analyzeActionOrExpression). Stop the walker here
		// to avoid re-initializing/re-walking the same lambda body.
		_ = n
		return nil
	case *ast.BLangFunction:
		if _, isDep := a.ctx().GetSymbol(n.Symbol()).(model.DependentlyTypedFunctionSymbol); isDep {
			initializeFunctionAnalyzer(a, n)
			return nil
		}
		return initializeFunctionAnalyzer(a, n)
	case *ast.BLangWhile:
		if !analyzeWhile(a, n) {
			return nil
		}
		return initializeLoopAnalyzer(a, n)
	case *ast.BLangForeach:
		if !validateForeach(a, n) {
			return nil
		}
		return initializeLoopAnalyzer(a, n)
	case *ast.BLangLock:
		if enclosingLockAnalyzer(a) != nil {
			a.semanticErr("lock statement cannot be nested inside another lock statement", n.GetPosition())
			return nil
		}
		validateLockStmt(a, n)
		return initializeLockAnalyzer(a, n)
	case *ast.BLangIf:
		if !analyzeIf(a, n) {
			return nil
		}
		return a
	case *ast.BLangBreak, *ast.BLangContinue:
		return nil
	case *ast.BLangXMLNS:
		expr := n.GetNamespaceURI()
		validateResolvedType(a, expr, semtypes.STRING)
		validateConstantExpr(a.ctx(), expr, func(e ast.BLangExpression) {
			a.semanticErr("expression is not a constant expression", e.GetPosition())
		})
		return nil
	case *ast.BLangMatchStatement:
		return a
	case *ast.BLangSimpleVariableDef:
		if !analyzeSimpleVariableDef(a, n) {
			return nil
		}
		if fa := enclosingFunctionAnalyzer(a); fa != nil && fa.locals != nil {
			v := n.Var
			final := v.IsFinal()
			if sym, ok := a.ctx().GetSymbol(v.Symbol()).(model.ValueSymbol); ok && sym.IsFinal() {
				final = true
			}
			fa.locals.define(v.Symbol(), varDeclMetadata{
				Type:  v.GetDeterminedType(),
				Final: final,
			})
		}
		return a
	case *ast.BLangAssignment:
		if !analyzeAssignment(a, n) {
			return nil
		}
		return a
	case *ast.BLangCompoundAssignment:
		if !analyzeCompoundAssignment(a, n) {
			return nil
		}
		return a
	case *ast.BLangExpressionStmt:
		if !analyzeActionOrExpression(a, n.Expr, semtypes.SemType{}) {
			return nil
		}
		exprType := n.Expr.GetDeterminedType()
		if !semtypes.IsSubtype(a.tyCtx(), exprType, semtypes.NIL) {
			a.semanticErr("expression value must be assigned", n.Expr.GetPosition())
			return nil
		}
		return a
	case ast.BLangExpression:
		if !analyzeActionOrExpression(a, n, semtypes.SemType{}) {
			return nil
		}
		return a
	case *ast.BLangReturn:
		if !returnFound(a, n) {
			return nil
		}
		return nil
	case *ast.BLangPanic:
		analyzeActionOrExpression(a, n.Expr, semtypes.ERROR)
		return nil
	case *ast.BLangRecordType:
		validateRecordFieldDefaults(a, n)
		return nil
	case *ast.BLangObjectType:
		if !n.Isolated {
			validateObjInclusions(a, n.Inclusions, n.InclusionPositions)
		}
		return nil
	case *ast.BLangClassDefinition:
		analyzeClassLikeDefn(a, n.Fields, n.InitFunction, n.Methods, n.ResourceMethods, n.Inclusions,
			n.InclusionPositions, n.IsIsolated(), n.GetPosition(), enclosingFromClass(n))
		return nil
	case *ast.BLangService:
		for _, expr := range n.AttachedExprs {
			ast.Walk(a, expr.(ast.BLangNode))
		}
		validateServiceDeclaredType(a, n)
		analyzeClassLikeDefn(a, n.Fields, n.InitFunction, n.Methods, n.ResourceMethods, n.Inclusions,
			n.InclusionPositions, n.IsIsolated(), n.GetPosition(), enclosingFromService(n))
		return nil
	default:
		return a
	}
}

func validateServiceDeclaredType[A analyzer](a A, svc *ast.BLangService) {
	typeData := svc.GetTypeData()
	if typeData.TypeDescriptor == nil {
		return
	}
	bodyTy := typeData.Type
	declaredTy := typeData.TypeDescriptor.(ast.BType).GetTypeData().Type
	if semtypes.IsZero(bodyTy) || semtypes.IsZero(declaredTy) {
		a.internalErr("service type not resolved", svc.GetPosition())
		return
	}
	if !semtypes.IsSubtype(a.tyCtx(), bodyTy, declaredTy) {
		a.semanticErr("service body is not a subtype of the declared service type", svc.GetPosition())
	}
}

func analyzeClassLikeDefn[A analyzer](a A, fields []ast.SimpleVariableNode, initFn *ast.BLangFunction, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod, inclusions []model.SymbolRef, inclusionPositions []diagnostics.Location, isolated bool, pos diagnostics.Location, enclosing *enclosingClassBody) {
	analyzeClassBodyMembers(a, fields, initFn, methods, resourceMethods, enclosing)
	validateClassDefn(a, inclusions, inclusionPositions, resourceMethods, isolated, fields, pos)
}

// analyzeClassBodyMembers performs semantic analysis over the fields,
// init function, methods, and resource methods of a class-like body
// (used by both classes and services).
func analyzeClassBodyMembers[A analyzer](a A, fields []ast.SimpleVariableNode, initFn *ast.BLangFunction, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod, enclosing *enclosingClassBody) {
	for _, fieldNode := range fields {
		field := fieldNode.(*ast.BLangSimpleVariable)
		if field.Expr != nil {
			expectedType := a.ctx().SymbolType(field.Symbol())
			analyzeActionOrExpression(a, field.Expr.(ast.BLangExpression), expectedType)
			// Drive the visitor through the initializer so per-node semantic
			// checks (e.g. isolated-module-var refs) fire uniformly with
			// every other walked initializer.
			ast.Walk(a, field.Expr.(ast.BLangNode))
		}
	}
	if initFn != nil {
		fa := initializeMethodAnalyzer(a, initFn, enclosing)
		walkMethodBody(fa, initFn)
	}
	for _, named := range methodsInResolutionOrder(methods) {
		method := named.method
		fa := initializeMethodAnalyzer(a, method, enclosing)
		walkMethodBody(fa, method)
	}
	for _, rm := range resourceMethods {
		fa := initializeResourceMethodAnalyzer(a, rm, enclosing)
		validateResourceMethodReturnType(a, fa.retTy, rm)
		walkMethodBody(fa, rm)
	}
}

type assignmentNode interface {
	GetVariable() ast.LExpr
	GetExpression() ast.BLangActionOrExpression
}

func analyzeAssignment[A analyzer](a A, assignment assignmentNode) bool {
	variable := assignment.GetVariable()
	if symbolNode, ok := variable.(ast.BNodeWithSymbol); ok {
		symbol := symbolNode.Symbol()
		if !ast.SymbolIsSet(symbolNode) {
			a.internalErr("unexpected nil symbol", variable.GetPosition())
			return false
		}
		ctx := a.ctx()
		switch ctx.SymbolKind(symbol) {
		case model.SymbolKindConstant:
			a.semanticErr("cannot assign to constant", variable.GetPosition())
			return false
		case model.SymbolKindParemeter:
			a.semanticErr("cannot assign to parameter", variable.GetPosition())
			return false
		case model.SymbolKindFunction:
			a.semanticErr("cannot assign to function", variable.GetPosition())
			return false
		case model.SymbolKindType:
			a.semanticErr("cannot assign to type", variable.GetPosition())
			return false
		case model.SymbolKindAnnotation:
			a.semanticErr("cannot assign to annotation", variable.GetPosition())
			return false
		}
	}
	if !analyzeActionOrExpression(a, variable, semtypes.SemType{}) {
		return false
	}
	expectedType := variable.GetDeterminedType()
	expression := assignment.GetExpression()
	return analyzeActionOrExpression(a, expression, expectedType)
}

func analyzeCompoundAssignment[A analyzer](a A, assignment *ast.BLangCompoundAssignment) bool {
	if !analyzeAssignment(a, assignment) {
		return false
	}
	lhsTy := assignment.GetVariable().GetDeterminedType()
	rhsTy := assignment.GetExpression().GetDeterminedType()
	if semtypes.ContainsBasicType(lhsTy, semtypes.NIL) || semtypes.ContainsBasicType(rhsTy, semtypes.NIL) {
		a.semanticErr("compound assignment operands cannot be nilable", assignment.GetPosition())
		return false
	}
	return true
}

func analyzeIf[A analyzer](a A, ifStmt *ast.BLangIf) bool {
	return analyzeActionOrExpression(a, ifStmt.Expr, semtypes.BOOLEAN)
}

func analyzeWhile[A analyzer](a A, whileStmt *ast.BLangWhile) bool {
	return analyzeActionOrExpression(a, whileStmt.Expr, semtypes.BOOLEAN)
}

func getIterableType(cx *context.CompilerContext, symbolType func(model.SymbolRef) semtypes.SemType) (semtypes.SemType, bool) {
	ref, ok := cx.LangLibDistinctTypeSymbol("lang.object", "Iterable")
	if !ok {
		return semtypes.SemType{}, false
	}
	return symbolType(ref), true
}

func validateForeach[A analyzer](a A, foreachStmt *ast.BLangForeach) bool {
	collection := foreachStmt.Collection
	if !analyzeActionOrExpression(a, collection, semtypes.SemType{}) {
		return false
	}
	variable := foreachStmt.VariableDef.GetVariable().(*ast.BLangSimpleVariable)
	variableType := a.ctx().SymbolType(variable.Symbol())
	if binExpr, ok := collection.(*ast.BLangBinaryExpr); ok && isRangeExpr(binExpr) {
		if !semtypes.IsSubtype(a.tyCtx(), variableType, semtypes.INT) {
			a.semanticErr("foreach variable must be a subtype of int for range expression", collection.GetPosition())
			return false
		}
	} else {
		collectionType := collection.GetDeterminedType()
		var expectedValueType semtypes.SemType
		switch {
		case semtypes.IsSubtype(a.tyCtx(), collectionType, semtypes.LIST):
			memberTypes := semtypes.ListAllMemberTypesInner(a.tyCtx(), collectionType)
			result := semtypes.NEVER
			for _, each := range memberTypes.SemTypes {
				result = semtypes.Union(result, each)
			}
			expectedValueType = result
		case semtypes.IsSubtype(a.tyCtx(), collectionType, semtypes.MAPPING):
			expectedValueType = semtypes.MappingMemberTypeInnerVal(a.tyCtx(), collectionType, semtypes.STRING)
		case semtypes.IsSubtype(a.tyCtx(), collectionType, semtypes.XML):
			expectedValueType = semtypes.XMLItemType(collectionType)
		default:
			tyCtx := a.tyCtx()
			iterableTy, ok := getIterableType(a.ctx(), a.ctx().SymbolType)
			if !ok {
				a.semanticErr("foreach collection must be subtype of object:Iterable", collection.GetPosition())
				return false
			}
			if !semtypes.IsSubtype(tyCtx, collectionType, iterableTy) {
				a.semanticErr("foreach collection must be subtype of object:Iterable", collection.GetPosition())
				return false
			}
			// Extract value type from the iterator's next() return type
			iteratorMethodTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("iterator"), collectionType)
			ld := semtypes.NewListDefinition()
			emptyArgs := ld.DefineListTypeWrapped(a.tyCtx().Env(), []semtypes.SemType{}, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
			iteratorTy := semtypes.FunctionReturnType(tyCtx, iteratorMethodTy, emptyArgs)
			nextMethodTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("next"), iteratorTy)
			nextReturnTy := semtypes.FunctionReturnType(tyCtx, nextMethodTy, emptyArgs)
			// next returns record{|T value|}|C where C is completion type (nil, error|nil, etc.)
			recordPart := semtypes.Diff(nextReturnTy, semtypes.Union(semtypes.NIL, semtypes.ERROR))
			expectedValueType = semtypes.MappingMemberTypeInnerVal(tyCtx, recordPart, semtypes.StringConst("value"))
		}
		if !semtypes.IsSubtype(a.tyCtx(), expectedValueType, variableType) {
			a.ctx().SemanticError(fmt.Sprintf("invalid variable type %s", semtypes.ToString(a.tyCtx(), variableType)), variable.GetPosition())
			return false
		}
	}
	return true
}

func recordKeyName(key *ast.BLangMappingKey) string {
	switch expr := key.Expr.(type) {
	case *ast.BLangLiteral:
		return expr.Value.(string)
	case *ast.BLangSimpleVarRef:
		return expr.VariableName.Value
	default:
		panic(fmt.Sprintf("unexpected record key expression type: %T", key.Expr))
	}
}

func setExpectedType[E ast.BLangNode](e E, expectedType semtypes.SemType) {
	e.SetDeterminedType(expectedType)
}

// validateRecordFieldDefaults checks that all record field default expressions
// satisfy the isolated-function rules. Field defaults are turned into closures
// at record construction time, so they must not call non-isolated functions or
// access mutable module state.
func validateRecordFieldDefaults[A analyzer](a A, node *ast.BLangRecordType) {
	parent := enclosingFunctionLocals(a)
	for _, field := range node.Fields() {
		if field.DefaultExpr == nil {
			continue
		}
		expr := field.DefaultExpr.(ast.BLangNode)
		validateIsolatedCapture(a, parent, expr)
		isIsolatedFunctionInner(a, expr, parent)
	}
}

func validateClassDefn[A analyzer](a A, inclusions []model.SymbolRef, inclusionPositions []diagnostics.Location, resourceMethods []*ast.BLangResourceMethod, isolated bool, fields []ast.SimpleVariableNode, pos diagnostics.Location) {
	if isolated {
		validateIsolatedClassFields(a, fields)
	} else {
		validateObjInclusions(a, inclusions, inclusionPositions)
	}
	validateDuplicateResourceMethods(a, resourceMethods)
}

func validateResourceMethodReturnType[A analyzer](a A, retTy semtypes.SemType, rm *ast.BLangResourceMethod) {
	if !semtypes.IsEmpty(a.tyCtx(), semtypes.Intersect(retTy, semtypes.FUNCTION)) {
		a.semanticErr("resource method return type must not include a function type", rm.GetPosition())
		return
	}
	if !semtypes.IsEmpty(a.tyCtx(), semtypes.Intersect(retTy, semtypes.CreateClientObject(a.tyCtx()))) {
		a.semanticErr("resource method return type must not include a client object type", rm.GetPosition())
	}
}

func validateDuplicateResourceMethods[A analyzer](a A, rms []*ast.BLangResourceMethod) {
	if len(rms) < 2 {
		return
	}
	tyCtx := a.tyCtx()
	ctx := a.ctx()
	for i := 1; i < len(rms); i++ {
		later, ok := ctx.GetSymbol(rms[i].Symbol()).(*model.ResourceMethodSymbol)
		if !ok {
			a.internalErr("expected resource method symbol", rms[i].GetPosition())
			continue
		}
		for j := 0; j < i; j++ {
			earlier, ok := ctx.GetSymbol(rms[j].Symbol()).(*model.ResourceMethodSymbol)
			if !ok {
				a.internalErr("expected resource method symbol", rms[j].GetPosition())
				continue
			}
			if later.MethodName() != earlier.MethodName() {
				continue
			}
			if semtypes.IsSameType(tyCtx, later.PathListType(), earlier.PathListType()) {
				a.semanticErr("duplicate resource method '"+later.MethodName()+"'", rms[i].GetPosition())
				break
			}
		}
	}
}

func isImmutableField(tyCtx semtypes.Context, field *ast.BLangSimpleVariable) bool {
	return field.IsFinal() && semtypes.IsSubtype(tyCtx, field.GetDeterminedType(), semtypes.VAL_READONLY)
}

func validateIsolatedClassFields[A analyzer](a A, fields []ast.SimpleVariableNode) {
	tyCtx := a.tyCtx()
	for _, f := range fields {
		field := f.(*ast.BLangSimpleVariable)
		if field.IsPublic() && !isImmutableField(tyCtx, field) {
			a.semanticErr("public field of an isolated object must be \"final\" and have a type that is a subtype of \"readonly\"", field.GetPosition())
		}
	}
}

// validateObjInclusions validate all the inclusions of non isolated objects are non-isolated as well.
// For isolated objects there are no restrictions
func validateObjInclusions[A analyzer](a A, inclusions []model.SymbolRef, positions []diagnostics.Location) {
	tyCtx := a.tyCtx()
	for i, ref := range inclusions {
		incTy := a.ctx().SymbolType(ref)
		if semtypes.IsIsolatedObject(tyCtx, incTy) {
			a.semanticErr("cannot include isolated object type in non-isolated object", positions[i])
		}
	}
}

func isIsolatedFnSymbol[A analyzer](a A, tyCtx semtypes.Context, symbol model.SymbolRef) bool {
	isolatedTop := semtypes.CreateIsolatedFn(tyCtx)
	fnTy := a.ctx().SymbolType(symbol)
	return semtypes.IsSubtype(tyCtx, fnTy, isolatedTop)
}

// isIsolatedFuncInner validates an isolated function body: every variable reference
// must resolve to a constant or to a variable declared within the body itself.
func isIsolatedFuncInner[A analyzer](a A, node ast.BLangNode) {
	locals := make(map[model.SymbolRef]struct{})
	tyCtx := a.tyCtx()
	ctx := a.ctx()
	everyNode(a, node, func(analyzer A, inner ast.BLangNode) bool {
		switch inner := inner.(type) {
		case *ast.BLangSimpleVariableDef:
			locals[ctx.UnnarrowedSymbol(inner.Var.Symbol())] = struct{}{}
		case *ast.BLangInvocation:
			if ast.IsStreamOperation(inner) {
				return true
			}
			if !isIsolatedFnSymbol(a, tyCtx, inner.Symbol()) {
				a.semanticErr("invocation of a non-isolated function", inner.GetPosition())
			}
		case *ast.BLangRemoteMethodCallAction:
			if !isIsolatedFnSymbol(a, tyCtx, inner.MethodSymbol()) {
				a.semanticErr("invocation of a non-isolated function", inner.GetPosition())
			}
		case *ast.BLangNewExpression:
			if ast.IsStreamNewExpression(inner) {
				return true
			}
			classTy := a.ctx().SymbolType(inner.ClassSymbol)
			initTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("init"), classTy)
			if !semtypes.IsZero(initTy) && !semtypes.IsSubtype(tyCtx, initTy, semtypes.CreateIsolatedFn(tyCtx)) {
				a.semanticErr("non isolated initialization", inner.GetPosition())
			}
		case *ast.BLangSimpleVarRef:
			sym := a.ctx().GetSymbol(inner.Symbol())
			varSym, ok := sym.(model.ValueSymbol)
			if !ok {
				analyzer.unimplementedErr("unsupported reference in isolated function body", inner.GetPosition())
				return true
			}
			if varSym.Name() == "self" {
				return true
			}
			if varSym.IsConst() {
				return true
			}
			if _, isLocal := locals[ctx.UnnarrowedSymbol(inner.Symbol())]; !isLocal {
				a.semanticErr("access of mutable variable", inner.GetPosition())
			}
		}
		return true
	})
}

type everyNodeVisitor[A analyzer] struct {
	analyzer  A
	predicate func(A, ast.BLangNode) bool
	result    bool
}

func (v *everyNodeVisitor[A]) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return v
	}
	if !v.predicate(v.analyzer, node) {
		v.result = false
		return nil
	}
	return v
}

func (v *everyNodeVisitor[A]) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return v
}

func everyNode[A analyzer](a A, node ast.BLangNode, predicate func(A, ast.BLangNode) bool) bool {
	visitor := &everyNodeVisitor[A]{analyzer: a, predicate: predicate, result: true}
	ast.Walk(visitor, node)
	return visitor.result
}
