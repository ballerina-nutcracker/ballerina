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
	"sync"

	"ballerina/ast"
	"ballerina/context"
	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
	"ballerina/values"
)

type desugaredNode[E ast.Node] struct {
	initStmts       []ast.StatementNode
	replacementNode E
}

// packageContext holds shared state for desugaring a single package.
//
// IMPORTANT: typeContext on packageContext must only be used from the goroutine
// that owns the package-level desugar flow (the main goroutine in DesugarPackage).
// Worker goroutines (per-function/class/service) must use their own non-shared
// typeContext via functionContext.typeCtx().
type packageContext struct {
	compilerCtx          *context.CompilerContext
	pkg                  *ast.BLangPackage
	importedSymbols      map[string]model.ExportedSymbolSpace
	importMu             sync.Mutex
	addedImplicitImports map[string]bool
	desugarSymbolCounter int
	typeContext          semtypes.Context
	xmlIteratorTypes     *semtypes.SemTypeCache
}

var _ desugarContext = &packageContext{}

func newPackageContext(compilerCtx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) *packageContext {
	return &packageContext{
		compilerCtx:          compilerCtx,
		pkg:                  pkg,
		importedSymbols:      importedSymbols,
		addedImplicitImports: make(map[string]bool),
		typeContext:          semtypes.ContextFrom(compilerCtx.GetTypeEnv()),
		xmlIteratorTypes:     semtypes.NewSemTypeCache(),
	}
}

func (ctx *packageContext) typeCtx() semtypes.Context {
	return ctx.typeContext
}

func (ctx *packageContext) addImplicitImport(pkgName string, imp ast.BLangImportPackage) {
	ctx.importMu.Lock()
	defer ctx.importMu.Unlock()
	if !ctx.addedImplicitImports[pkgName] {
		ctx.addedImplicitImports[pkgName] = true
		ctx.pkg.Imports = append(ctx.pkg.Imports, imp)
	}
}

func (ctx *packageContext) getImportedSymbolSpace(pkgName string) (model.ExportedSymbolSpace, bool) {
	space, ok := ctx.importedSymbols[pkgName]
	return space, ok
}

func (ctx *packageContext) symbolType(symbol model.SymbolRef) semtypes.SemType {
	return ctx.compilerCtx.SymbolType(symbol)
}

func (ctx *packageContext) newFunctionScope(parent model.Scope) *model.FunctionScope {
	return ctx.compilerCtx.NewFunctionScope(parent, *ctx.pkg.PackageID)
}

func (ctx *packageContext) getSymbol(ref model.SymbolRef) model.Symbol {
	return ctx.compilerCtx.GetSymbol(ref)
}

func (ctx *packageContext) getSymbolType(ref model.SymbolRef) semtypes.SemType {
	return ctx.compilerCtx.SymbolType(ref)
}

func (ctx *packageContext) setSymbolType(ref model.SymbolRef, ty semtypes.SemType) {
	ctx.compilerCtx.SetSymbolType(ref, ty)
}

func (ctx *packageContext) typeEnv() semtypes.Env {
	return ctx.compilerCtx.GetTypeEnv()
}

func (ctx *packageContext) nextDesugarSymbolName() string {
	name := fmt.Sprintf("$desugar$%d", ctx.desugarSymbolCounter)
	ctx.desugarSymbolCounter++
	return name
}

func (ctx *packageContext) addSymbolToSameSpace(ref model.SymbolRef, name string, symbol model.Symbol) model.SymbolRef {
	return ctx.compilerCtx.AddSymbolToSameSpace(ref, name, symbol)
}

func (ctx *packageContext) addModuleSymbol(name string, symbol model.Symbol) model.SymbolRef {
	ctx.pkg.Scope.AddSymbol(name, symbol)
	ref, _ := ctx.pkg.Scope.GetSymbol(name)
	return ref
}

func (ctx *packageContext) internalError(msg string) {
	ctx.compilerCtx.InternalError(msg, diagnostics.Location{})
}

func (ctx *packageContext) unimplemented(msg string) {
	ctx.compilerCtx.Unimplemented(msg, diagnostics.Location{})
}

type functionContext struct {
	pkgCtx               *packageContext
	scopeStack           []model.Scope
	desugarSymbolCounter int
	loopVarStack         []ast.LExpr // Stack to track loop variables (nil for while, varRef for desugared foreach)
	// typeContext is the non-shared type context for this function. It is owned
	// by the goroutine desugaring this function and must not be shared.
	typeContext semtypes.Context
}

// typeCtx returns the function-local type context, lazily creating it on first
// use. Because functionContext is confined to a single goroutine, this needs no
// synchronization.
func (ctx *functionContext) typeCtx() semtypes.Context {
	if ctx.typeContext == nil {
		ctx.typeContext = semtypes.ContextFrom(ctx.pkgCtx.typeEnv())
	}
	return ctx.typeContext
}

var _ desugarContext = &functionContext{}

func (ctx *functionContext) internalError(msg string) {
	ctx.pkgCtx.internalError(msg)
}

func (ctx *functionContext) unimplemented(msg string) {
	ctx.pkgCtx.unimplemented(msg)
}

func (ctx *functionContext) getImportedSymbolSpace(pkgName string) (model.ExportedSymbolSpace, bool) {
	return ctx.pkgCtx.getImportedSymbolSpace(pkgName)
}

func (ctx *functionContext) addImplicitImport(pkgName string, imp ast.BLangImportPackage) {
	ctx.pkgCtx.addImplicitImport(pkgName, imp)
}

func (ctx *functionContext) symbolType(symbol model.SymbolRef) semtypes.SemType {
	return ctx.pkgCtx.symbolType(symbol)
}

func (ctx *functionContext) pushScope(scope model.Scope) {
	ctx.scopeStack = append(ctx.scopeStack, scope)
}

func (ctx *functionContext) popScope() {
	if len(ctx.scopeStack) == 0 {
		ctx.internalError("cannot pop from empty scope stack")
	}
	ctx.scopeStack = ctx.scopeStack[:len(ctx.scopeStack)-1]
}

func (ctx *functionContext) currentScope() model.Scope {
	if len(ctx.scopeStack) == 0 {
		ctx.internalError("scope stack is empty")
	}
	return ctx.scopeStack[len(ctx.scopeStack)-1]
}

func (ctx *functionContext) pushLoopVar(varRef ast.LExpr) {
	ctx.loopVarStack = append(ctx.loopVarStack, varRef)
}

func (ctx *functionContext) popLoopVar() {
	if len(ctx.loopVarStack) == 0 {
		ctx.internalError("cannot pop from empty loopVar stack")
	}
	ctx.loopVarStack = ctx.loopVarStack[:len(ctx.loopVarStack)-1]
}

func (ctx *functionContext) currentLoopVar() ast.LExpr {
	if len(ctx.loopVarStack) == 0 {
		return nil
	}
	return ctx.loopVarStack[len(ctx.loopVarStack)-1]
}

func (ctx *functionContext) nextDesugarSymbolName() string {
	name := fmt.Sprintf("$desugar$%d", ctx.desugarSymbolCounter)
	ctx.desugarSymbolCounter++
	return name
}

func (ctx *functionContext) addSymbolToSameSpace(ref model.SymbolRef, name string, symbol model.Symbol) model.SymbolRef {
	return ctx.pkgCtx.addSymbolToSameSpace(ref, name, symbol)
}

func (ctx *functionContext) newFunctionScope(parent model.Scope) *model.FunctionScope {
	return ctx.pkgCtx.newFunctionScope(parent)
}

func (ctx *functionContext) setSymbolType(ref model.SymbolRef, ty semtypes.SemType) {
	ctx.pkgCtx.setSymbolType(ref, ty)
}

func (ctx *functionContext) getSymbol(ref model.SymbolRef) model.Symbol {
	return ctx.pkgCtx.getSymbol(ref)
}

func (ctx *functionContext) typeEnv() semtypes.Env {
	return ctx.pkgCtx.typeEnv()
}

type desugarContext interface {
	nextDesugarSymbolName() string
	addSymbolToSameSpace(ref model.SymbolRef, name string, symbol model.Symbol) model.SymbolRef
	newFunctionScope(parent model.Scope) *model.FunctionScope
	setSymbolType(ref model.SymbolRef, ty semtypes.SemType)
	symbolType(ref model.SymbolRef) semtypes.SemType
	getSymbol(ref model.SymbolRef) model.Symbol
	typeEnv() semtypes.Env
	internalError(msg string)
}

type desugaredSymbol struct {
	name     string
	ty       semtypes.SemType
	kind     model.SymbolKind
	isPublic bool
}

var _ model.Symbol = &desugaredSymbol{}

func (s *desugaredSymbol) Name() string {
	return s.name
}

func (s *desugaredSymbol) Type() semtypes.SemType {
	return s.ty
}

func (s *desugaredSymbol) Kind() model.SymbolKind {
	return s.kind
}

func (s *desugaredSymbol) SetType(_ semtypes.SemType) {
	panic("SetType is not supported for desugared symbols")
}

func (s *desugaredSymbol) IsPublic() bool {
	return s.isPublic
}

func (s *desugaredSymbol) Copy() model.Symbol {
	cp := *s
	return &cp
}

func (ctx *functionContext) addDesugardSymbol(ty semtypes.SemType, kind model.SymbolKind, isPublic bool) (string, model.SymbolRef) {
	if len(ctx.scopeStack) == 0 {
		ctx.internalError("cannot add desugared symbol when scope stack is empty")
	}
	name := ctx.nextDesugarSymbolName()
	symbol := &desugaredSymbol{
		name:     name,
		ty:       ty,
		kind:     kind,
		isPublic: isPublic,
	}
	ctx.currentScope().AddSymbol(name, symbol)
	ref, _ := ctx.currentScope().GetSymbol(name)
	return name, ref
}

// moduleInitNode is a unified handle over either a module-level constant or a
// module-level variable for the purpose of building the synthetic init function
// in dependency order.
type moduleInitNode struct {
	sym  model.SymbolRef
	expr ast.BLangExpression // nil if the declaration has no initializer
	name *ast.BLangIdentifier
}

// collectModuleInitNodes gathers module-level global variables for the synthetic
// init function. Constants are not included: a foldable constant is inlined at
// its use sites and an unfoldable one is a compile-time error, so no constant
// needs runtime initialization.
func collectModuleInitNodes(pkg *ast.BLangPackage) []moduleInitNode {
	nodes := make([]moduleInitNode, 0, len(pkg.GlobalVars))
	for i := range pkg.GlobalVars {
		gv := &pkg.GlobalVars[i]
		var expr ast.BLangExpression
		if gv.Expr != nil {
			expr = gv.Expr.(ast.BLangExpression)
		}
		nodes = append(nodes, moduleInitNode{
			sym:  gv.Symbol(),
			expr: expr,
			name: gv.Name,
		})
	}
	return nodes
}

// We desugar module initializers into the init function, so they should no longer be there.
func clearModuleInitExprs(pkg *ast.BLangPackage) {
	for i := range pkg.GlobalVars {
		pkg.GlobalVars[i].Expr = nil
	}
}

// Accumulate all the nodes referred by a given node. Assume all references to be valid
// (semantic analysis should have cought any invalid cases) and is agnostic towards the exact expression
type dependencyVisitor struct {
	compilerCtx    *context.CompilerContext
	nodeSet        map[model.SymbolRef]int // symbol → index into nodes slice
	runtimeGlobals map[string]model.SymbolRef
	deps           map[int]struct{}
}

// mark current node depnds on on the given
func (v *dependencyVisitor) depends(ref model.SymbolRef) {
	unnarrowed := v.compilerCtx.UnnarrowedSymbol(ref)
	if idx, ok := v.nodeSet[unnarrowed]; ok {
		v.deps[idx] = struct{}{}
	}
}

func (v *dependencyVisitor) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangConstRef:
		v.depends(n.Symbol())
	case *ast.BLangSimpleVarRef:
		v.depends(n.Symbol())
	case *ast.BLangAnnotAccessExpr:
		v.dependsOnRuntimeAnnotation(n)
	}
	return v
}

func (v *dependencyVisitor) VisitTypeData(_ *ast.TypeData) ast.Visitor { return v }

func (v *dependencyVisitor) dependsOnRuntimeAnnotation(expr *ast.BLangAnnotAccessExpr) {
	receiver, ok := expr.Expr.(ast.BNodeWithSymbol)
	if !ok {
		v.compilerCtx.InternalError("annotation access receiver has no symbol", expr.GetPosition())
		return
	}
	annotationSymbol := v.compilerCtx.GetSymbol(expr.Symbol())
	key := model.AnnotationKey(v.compilerCtx.SymbolPackage(expr.Symbol()), annotationSymbol.Name())
	ref, ok := v.compilerCtx.SymbolAnnotationValues(receiver.Symbol())[key].(*values.RuntimeAnnotationValueRef)
	if !ok {
		// Compile-time annotation values and absent annotations do not add module-init dependencies.
		return
	}
	if global, ok := v.runtimeGlobals[ref.GlobalLookupKey()]; ok {
		v.depends(global)
	}
}

func toplogicallySortInits(compilerCtx *context.CompilerContext, nodes []moduleInitNode) ([]int, bool) {
	nodeSet := make(map[model.SymbolRef]int, len(nodes))
	runtimeGlobals := make(map[string]model.SymbolRef, len(nodes))
	for i, n := range nodes {
		nodeSet[n.sym] = i
		ref := n.sym
		pkg := compilerCtx.SymbolPackage(ref)
		runtimeGlobals[pkg.Organization+"/"+pkg.Package+":"+n.name.GetValue()] = ref
	}

	deps := make([][]int, len(nodes))
	for i := range nodes {
		if nodes[i].expr == nil {
			continue
		}
		v := &dependencyVisitor{
			compilerCtx:    compilerCtx,
			nodeSet:        nodeSet,
			runtimeGlobals: runtimeGlobals,
			deps:           make(map[int]struct{}),
		}
		ast.Walk(v, nodes[i].expr)
		for d := range v.deps {
			deps[i] = append(deps[i], d)
		}
	}

	// https://en.wikipedia.org/wiki/Topological_sorting#Depth-first_search
	const (
		unvisited = 0
		inStack   = 1
		done      = 2
	)
	state := make([]int, len(nodes))
	order := make([]int, 0, len(nodes))

	var visit func(i int) bool
	visit = func(i int) bool {
		switch state[i] {
		case inStack:
			compilerCtx.InternalError(
				fmt.Sprintf("invalid cycle detected for %s", nodes[i].name.GetValue()),
				nodes[i].name.GetPosition(),
			)
			return false
		case done:
			return true
		default:
			state[i] = inStack
			for _, d := range deps[i] {
				if !visit(d) {
					return false
				}
			}
			state[i] = done
			order = append(order, i)
			return true
		}
	}

	for i := range nodes {
		if !visit(i) {
			return nil, false
		}
	}
	return order, true
}

func buildInitAssignment(compilerCtx *context.CompilerContext, node moduleInitNode) ast.StatementNode {
	initExpr := node.expr
	basePos := initExpr.GetPosition()
	varRef := &ast.BLangSimpleVarRef{
		VariableName: node.name,
	}
	varRef.SetSymbol(node.sym)
	varRef.SetDeterminedType(compilerCtx.SymbolType(node.sym))
	assignment := &ast.BLangAssignment{
		VarRef: varRef,
		Expr:   initExpr,
	}
	assignment.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(assignment, basePos)
	return assignment
}

// wrapInCheck wraps an expression with a check (check <expr>).
func wrapInCheck(expr ast.BLangExpression) ast.BLangExpression {
	exprTy := expr.GetDeterminedType()
	if !semtypes.ContainsBasicType(exprTy, semtypes.ERROR) {
		return expr
	}
	narrowed := semtypes.Diff(exprTy, semtypes.ERROR)
	checked := &ast.BLangCheckedExpr{Expr: expr}
	checked.SetDeterminedType(narrowed)
	checked.SetPosition(expr.GetPosition())
	return checked
}

// createExpressionStmt wraps the given expression into a BLangExpressionStmt
func createExpressionStmt(expr ast.BLangExpression, pos diagnostics.Location) *ast.BLangExpressionStmt {
	stmt := &ast.BLangExpressionStmt{Expr: expr}
	stmt.SetDeterminedType(semtypes.NEVER)
	stmt.SetPosition(pos)
	return stmt
}

// serviceInitResultType returns the static result type of constructing a
// service. If the service type is T  and init function return () then this is T, else if error? then it is E|error
func serviceInitResultType(pkgCtx *packageContext, svc *ast.BLangService, svcTy semtypes.SemType) semtypes.SemType {
	if svc.InitFunction == nil {
		return svcTy
	}
	fnSym, ok := pkgCtx.getSymbol(svc.InitFunction.Symbol()).(model.FunctionSymbol)
	if !ok {
		pkgCtx.internalError("failed to find init function symbol")
		return semtypes.NEVER
	}
	retTy := fnSym.Signature().ReturnType
	errComponent := semtypes.Diff(retTy, semtypes.NIL)
	return semtypes.Union(errComponent, svcTy)
}

func desugarInitFn(pkgCtx *packageContext, compilerCtx *context.CompilerContext, pkg *ast.BLangPackage) {
	nodes := collectModuleInitNodes(pkg)
	order, ok := toplogicallySortInits(compilerCtx, nodes)
	if !ok {
		pkgCtx.internalError("module init dependency ordering failed")
		return
	}

	// we need init if the package has any module level constant/variable with init expressions or services
	needInit := pkg.InitFunction != nil || len(pkg.Services) > 0
	if !needInit {
		for _, n := range nodes {
			if n.expr != nil {
				needInit = true
				break
			}
		}
	}
	if !needInit {
		return
	}

	initFnCreated := pkg.InitFunction == nil
	initPos := pickInitFunctionPosition(nodes, pkg)
	if initFnCreated {
		createInitFunction(compilerCtx, pkg, initPos)
	}

	// We unconditionally treat init to be fallable if listeners need lifecycle handling, irrespective of whether
	// service init or listener registration can actually fail.
	hasListeners := len(pkg.Services) > 0 || hasModuleListenerVar(compilerCtx, nodes)

	widenInitReturnTypeToErrorOptional(compilerCtx, pkg.InitFunction)

	var initStmts []ast.StatementNode
	var moduleListenersRef *ast.BLangSimpleVarRef
	if hasListeners {
		mlRef, mlInitStmt := addModuleListenersGlobal(pkgCtx, pkg, initPos)
		moduleListenersRef = mlRef
		initStmts = append(initStmts, mlInitStmt)
	}

	for _, idx := range order {
		node := nodes[idx]
		if node.expr == nil {
			continue
		}
		if vs, ok := compilerCtx.GetSymbol(node.sym).(*model.VariableSymbol); ok && vs.IsListener() {
			initStmts = append(initStmts, buildListnerInit(pkgCtx, node, moduleListenersRef)...)
		} else {
			initStmts = append(initStmts, buildInitAssignment(compilerCtx, node))
		}
	}
	clearModuleInitExprs(pkg)

	for i := range pkg.Services {
		initStmts = append(initStmts, buildServiceInitStmts(pkgCtx, pkg, &pkg.Services[i])...)
	}
	body := pkg.InitFunction.Body.(*ast.BLangBlockFunctionBody)
	if initFnCreated {
		body.Stmts = initStmts
	} else {
		// We prepend desugard statements before users init statments.
		body.Stmts = append(initStmts, body.Stmts...)
	}

	*pkg.InitFunction = *desugarFunction(pkgCtx, pkg.InitFunction)

	if hasListeners {
		createLifeCycleHooks(pkgCtx, pkg, moduleListenersRef, initPos)
	}
}

// createLifeCycleHooks generates `$start`, `$gracefulStop` and `$immediateStop`
// for the module and appends them to pkg.Functions. Each function iterates
// the `$moduleListeners` array and invokes the listener method that
// ListenerMethodFor returns for the function name, propagating errors via
// `check`.
func createLifeCycleHooks(pkgCtx *packageContext, pkg *ast.BLangPackage, moduleListenersRef *ast.BLangSimpleVarRef, initPos diagnostics.Location) {
	compilerCtx := pkgCtx.compilerCtx
	errorOrNil := semtypes.Union(semtypes.NIL, semtypes.ERROR)
	pkgID := pkg.PackageID
	tyCtx := pkgCtx.typeCtx()
	elementTy := semtypes.ListMemberTypeInnerVal(tyCtx, moduleListenersRef.GetDeterminedType(), semtypes.INT)

	buildMethodCallStmt := func(scope model.Scope, listenerRef ast.BLangExpression, methodName string) ast.StatementNode {
		fnTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst(methodName), elementTy)
		if semtypes.IsZero(fnTy) {
			pkgCtx.internalError("listener element type does not expose method " + methodName)
			return nil
		}
		ld := semtypes.NewListDefinition()
		paramList := ld.DefineListTypeWrapped(pkgCtx.typeEnv(), nil, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
		retTy := semtypes.FunctionReturnType(tyCtx, fnTy, paramList)

		fnSymName := "$" + methodName + "Method"
		fnSym := &desugaredSymbol{name: fnSymName, ty: fnTy, kind: model.SymbolKindFunction}
		scope.AddSymbol(fnSymName, fnSym)
		fnSymRef, _ := scope.GetSymbol(fnSymName)

		inv := &ast.BLangInvocation{}
		inv.Name = &ast.BLangIdentifier{Value: methodName}
		inv.Expr = listenerRef
		inv.SetSymbol(fnSymRef)
		inv.SetDeterminedType(retTy)
		inv.SetPosition(initPos)
		return createExpressionStmt(wrapInCheck(inv), initPos)
	}

	buildForeach := func(fnScope model.Scope, methodName string) *ast.BLangForeach {
		foreachScope := compilerCtx.NewBlockScope(fnScope, *pkgID)
		loopVarName := "$listener"
		loopSym := &desugaredSymbol{name: loopVarName, ty: elementTy, kind: model.SymbolKindVariable}
		foreachScope.AddSymbol(loopVarName, loopSym)
		loopSymRef, _ := foreachScope.GetSymbol(loopVarName)

		loopVarIdent := &ast.BLangIdentifier{Value: loopVarName}
		loopVarIdent.SetDeterminedType(semtypes.NEVER)
		loopVarIdent.SetPosition(initPos)
		loopVar := &ast.BLangSimpleVariable{Name: loopVarIdent}
		loopVar.SetDeterminedType(elementTy)
		loopVar.SetSymbol(loopSymRef)
		loopVar.SetPosition(initPos)
		loopVarDef := &ast.BLangSimpleVariableDef{Var: loopVar}
		loopVarDef.SetDeterminedType(semtypes.NEVER)
		loopVarDef.SetPosition(initPos)

		loopVarRef := &ast.BLangSimpleVarRef{VariableName: loopVarIdent}
		loopVarRef.SetSymbol(loopSymRef)
		loopVarRef.SetDeterminedType(elementTy)
		loopVarRef.SetPosition(initPos)

		collectionRef := *moduleListenersRef
		variableName := *moduleListenersRef.VariableName
		collectionRef.VariableName = &variableName
		bodyStmt := buildMethodCallStmt(foreachScope, loopVarRef, methodName)

		foreach := &ast.BLangForeach{
			VariableDef: loopVarDef,
			Collection:  &collectionRef,
			Body:        ast.BLangBlockStmt{Stmts: []ast.StatementNode{bodyStmt}},
		}
		foreach.Body.SetDeterminedType(semtypes.NEVER)
		foreach.Body.SetPosition(initPos)
		foreach.SetScope(foreachScope)
		foreach.SetDeterminedType(semtypes.NEVER)
		foreach.SetPosition(initPos)
		return foreach
	}

	buildLifecycleFn := func(fnName string) *ast.BLangFunction {
		fn := &ast.BLangFunction{}
		fn.Name = ast.BLangIdentifier{Value: fnName}
		fn.Name.SetDeterminedType(semtypes.NEVER)
		fn.SetDeterminedType(semtypes.NEVER)
		fn.SetPosition(initPos)

		signature := model.FunctionSignature{ReturnType: errorOrNil}
		fnSymbol := model.NewFunctionSymbol(fnName, signature, false)
		symbolSpace := compilerCtx.NewSymbolSpace(*pkgID)
		symbolSpace.AddSymbol(fnName, fnSymbol)
		symRef, _ := symbolSpace.GetSymbol(fnName)
		fn.SetSymbol(symRef)
		fnScope := compilerCtx.NewFunctionScope(nil, *pkgID)
		fn.SetScope(fnScope)

		foreachStmt := buildForeach(fnScope, ListenerMethodFor(fnName))

		body := &ast.BLangBlockFunctionBody{Stmts: []ast.StatementNode{foreachStmt}}
		body.SetDeterminedType(semtypes.NEVER)
		body.SetPosition(initPos)
		fn.Body = body
		return fn
	}

	for _, fnName := range []string{StartFunctionName, GracefulStopFunctionName, ImmediateStopFunctionName} {
		pkg.Functions = append(pkg.Functions, *buildLifecycleFn(fnName))
	}
}

// Names of the per-module lifecycle dispatch functions emitted by desugar.
// Each one iterates the module's listener array and invokes the matching
// listener method (see ListenerMethodFor).
const (
	StartFunctionName         = "$start"
	GracefulStopFunctionName  = "$gracefulStop"
	ImmediateStopFunctionName = "$immediateStop"
)

// ListenerMethodFor returns the listener method name that the given lifecycle
// dispatch function calls. Returns "" if name is not a lifecycle function.
func ListenerMethodFor(name string) string {
	switch name {
	case StartFunctionName:
		return "start"
	case GracefulStopFunctionName:
		return "gracefulStop"
	case ImmediateStopFunctionName:
		return "immediateStop"
	}
	return ""
}

func hasModuleListenerVar(compilerCtx *context.CompilerContext, nodes []moduleInitNode) bool {
	for _, n := range nodes {
		if vs, ok := compilerCtx.GetSymbol(n.sym).(*model.VariableSymbol); ok && vs.IsListener() {
			return true
		}
	}
	return false
}

func buildModuleInitVarRef(compilerCtx *context.CompilerContext, node moduleInitNode) *ast.BLangSimpleVarRef {
	pos := diagnostics.Location{}
	if node.expr != nil {
		pos = node.expr.GetPosition()
	}
	listenerVarRef := &ast.BLangSimpleVarRef{VariableName: node.name}
	listenerVarRef.SetSymbol(node.sym)
	listenerVarRef.SetDeterminedType(compilerCtx.SymbolType(node.sym))
	listenerVarRef.SetPosition(pos)
	return listenerVarRef
}

func buildListnerInit(pkgCtx *packageContext, node moduleInitNode, moduleListenersRef *ast.BLangSimpleVarRef) []ast.StatementNode {
	compilerCtx := pkgCtx.compilerCtx
	pos := node.expr.GetPosition()
	listenerVarRef := buildModuleInitVarRef(compilerCtx, node)

	assign := &ast.BLangAssignment{VarRef: listenerVarRef, Expr: wrapInCheck(node.expr)}
	assign.SetDeterminedType(semtypes.NEVER)
	assign.SetPosition(pos)

	stmts := []ast.StatementNode{assign}

	mlRef := *moduleListenersRef
	pushSrc := *listenerVarRef
	inv := createArrayPushInvocation(pkgCtx, &mlRef, &pushSrc)
	if inv == nil {
		pkgCtx.internalError("failed to create array:push invocation for module listener")
		return stmts
	}
	return append(stmts, createExpressionStmt(inv, pos))
}

func pickInitFunctionPosition(nodes []moduleInitNode, pkg *ast.BLangPackage) diagnostics.Location {
	for _, n := range nodes {
		if n.expr != nil {
			return n.expr.GetPosition()
		}
	}
	if len(pkg.Services) > 0 {
		return pkg.Services[0].GetPosition()
	}
	return diagnostics.Location{}
}

// widenInitReturnTypeToErrorOptional mutates the module init function so its
// return type is `error?`
func widenInitReturnTypeToErrorOptional(compilerCtx *context.CompilerContext, initFn *ast.BLangFunction) {
	newRet := semtypes.Union(semtypes.NIL, semtypes.ERROR)
	fnSym, ok := compilerCtx.GetSymbol(initFn.Symbol()).(model.FunctionSymbol)
	if !ok {
		compilerCtx.InternalError("module init function symbol is not a FunctionSymbol", initFn.GetPosition())
		return
	}
	sig := fnSym.Signature()
	sig.ReturnType = newRet
	fnSym.SetSignature(sig)
}

func createInitFunction(compilerCtx *context.CompilerContext, pkg *ast.BLangPackage, initPos diagnostics.Location) {
	pkg.InitFunction = &ast.BLangFunction{}
	pkg.InitFunction.Name = ast.BLangIdentifier{Value: "init"}
	pkg.InitFunction.Name.SetDeterminedType(semtypes.NEVER)
	body := &ast.BLangBlockFunctionBody{}
	body.SetDeterminedType(semtypes.NEVER)
	body.SetPosition(initPos)
	pkg.InitFunction.Body = body
	pkg.InitFunction.SetDeterminedType(semtypes.NEVER)
	pkg.InitFunction.SetPosition(initPos)
	pkgID := pkg.PackageID
	signature := model.FunctionSignature{ReturnType: semtypes.NIL}
	initSymbol := model.NewFunctionSymbol("init", signature, false)
	symbolSpace := compilerCtx.NewSymbolSpace(*pkgID)
	symbolSpace.AddSymbol("init", initSymbol)
	symRef, _ := symbolSpace.GetSymbol("init")
	pkg.InitFunction.SetSymbol(symRef)
	fnScope := compilerCtx.NewFunctionScope(nil, *pkgID)
	pkg.InitFunction.SetScope(fnScope)
}

// moduleListenersGlobalName is the module-level variable that holds every
// listener value evaluated during module init (see design.md). Lifecycle
// methods (`$gracefulStop`, `$immediateStop`) are suppose to use this array.
// https://github.com/ballerina-nutcracker/ballerina/issues/475
const moduleListenersGlobalName = "$moduleListeners"

func addModuleListenersGlobal(pkgCtx *packageContext, pkg *ast.BLangPackage, pos diagnostics.Location) (*ast.BLangSimpleVarRef, ast.StatementNode) {
	tyCtx := pkgCtx.typeCtx()
	env := pkgCtx.typeEnv()
	var listnerTop semtypes.SemType
	{
		listDefn := semtypes.NewListDefinition()
		stringArr := listDefn.DefineListTypeWrapped(env, nil, 0, semtypes.STRING, semtypes.CellMutability_CELL_MUT_LIMITED)
		listnerTop = semtypes.Union(semtypes.ListenerTy(tyCtx, semtypes.NEVER, stringArr), semtypes.Union(semtypes.ListenerTy(tyCtx, semtypes.NEVER, semtypes.STRING), semtypes.ListenerTy(tyCtx, semtypes.NEVER, semtypes.NIL)))
	}
	var arrTy semtypes.SemType
	{
		listDefn := semtypes.NewListDefinition()
		arrTy = listDefn.DefineListTypeWrapped(env, nil, 0, listnerTop, semtypes.CellMutability_CELL_MUT_LIMITED)
	}

	sym := model.NewVariableSymbol(moduleListenersGlobalName, false, false, false)
	symRef := pkgCtx.addModuleSymbol(moduleListenersGlobalName, &sym)
	pkgCtx.setSymbolType(symRef, arrTy)

	global := &ast.BLangSimpleVariable{}
	global.SetName(&ast.BLangIdentifier{Value: moduleListenersGlobalName})
	global.Name.SetDeterminedType(semtypes.NEVER)
	global.SetSymbol(symRef)
	global.SetDeterminedType(arrTy)
	global.SetPosition(pos)
	pkg.AddGlobalVariable(global)

	ref := &ast.BLangSimpleVarRef{VariableName: &ast.BLangIdentifier{Value: moduleListenersGlobalName}}
	ref.SetSymbol(symRef)
	ref.SetDeterminedType(arrTy)
	ref.SetPosition(pos)

	emptyList := &ast.BLangListConstructorExpr{Exprs: []ast.BLangExpression{}}
	emptyList.SetDeterminedType(arrTy)
	emptyList.AtomicType = semtypes.LIST_ATOMIC_INNER
	emptyList.SetPosition(pos)

	assignRef := *ref
	assign := &ast.BLangAssignment{VarRef: &assignRef, Expr: emptyList}
	assign.SetDeterminedType(semtypes.NEVER)
	assign.SetPosition(pos)
	return ref, assign
}

// buildServiceInitStmts produces the statements that, for each service
// declaration in the module, construct the service instance into a synthetic
// local in the init function and call `attach` on each listener in the
// service's `on` clause. The statements run in the module init function
// after all module-level variable initializers.
func buildServiceInitStmts(pkgCtx *packageContext, pkg *ast.BLangPackage, svc *ast.BLangService) []ast.StatementNode {
	if svc.Definition == nil {
		pkgCtx.internalError("service has no object definition at desugar")
		return nil
	}
	svcTy := svc.Definition.GetSemType(pkgCtx.typeEnv())
	if semtypes.IsZero(svcTy) {
		pkgCtx.internalError("service object type unresolved at desugar")
		return nil
	}
	initExpr := &BLangServiceInit{Service: svc}
	initExpr.SetDeterminedType(serviceInitResultType(pkgCtx, svc, svcTy))
	initExpr.SetPosition(svc.GetPosition())

	varDef, svcRef := createDesugaredLocal(pkgCtx, pkg.InitFunction.Scope(), svcTy, wrapInCheck(initExpr), svc.GetPosition())
	stmts := []ast.StatementNode{varDef}

	for _, listenerExpr := range svc.AttachedExprs {
		refCopy := *svcRef
		attachInv := buildListenerAttachInvocation(pkgCtx, svc, listenerExpr, &refCopy)
		if attachInv == nil {
			continue
		}
		stmts = append(stmts, createExpressionStmt(wrapInCheck(attachInv), svc.GetPosition()))
	}
	return stmts
}

// hoistInlineServiceListeners replaces each inline listener expression in
// the `on` clause of a service with a reference to a synthetic module-level
// `listener` variable initialized to that expression.
func hoistInlineServiceListeners(pkgCtx *packageContext, pkg *ast.BLangPackage) {
	for i := range pkg.Services {
		svc := &pkg.Services[i]
		for j, listenerExpr := range svc.AttachedExprs {
			_, ok := listenerExpr.(*ast.BLangSimpleVarRef)
			if ok {
				continue
			}

			pos := listenerExpr.GetPosition()
			exprTy := listenerExpr.GetDeterminedType()
			if semtypes.IsZero(exprTy) {
				pkgCtx.internalError("inline listener expression has no determined type at desugar")
				return
			}
			ty := semtypes.Diff(exprTy, semtypes.ERROR)
			name := pkgCtx.nextDesugarSymbolName()
			sym := model.NewVariableSymbol(name, false, false, false)
			sym.SetListener()
			symRef := pkgCtx.addModuleSymbol(name, &sym)
			pkgCtx.setSymbolType(symRef, ty)

			ident := &ast.BLangIdentifier{Value: name}
			ident.SetDeterminedType(semtypes.NEVER)
			ident.SetPosition(pos)

			gv := &ast.BLangSimpleVariable{Name: ident}
			gv.SetDeterminedType(ty)
			gv.SetSymbol(symRef)
			gv.SetInitialExpression(listenerExpr)
			gv.SetPosition(pos)
			pkg.AddGlobalVariable(gv)

			ref := &ast.BLangSimpleVarRef{VariableName: ident}
			ref.SetSymbol(symRef)
			ref.SetDeterminedType(ty)
			ref.SetPosition(pos)
			svc.AttachedExprs[j] = ref
		}
	}
}

func createDesugaredLocal(pkgCtx *packageContext, scope model.Scope, ty semtypes.SemType, initExpr ast.BLangExpression, pos diagnostics.Location) (*ast.BLangSimpleVariableDef, *ast.BLangSimpleVarRef) {
	name := pkgCtx.nextDesugarSymbolName()
	sym := &desugaredSymbol{name: name, ty: ty, kind: model.SymbolKindVariable}
	scope.AddSymbol(name, sym)
	symRef, _ := scope.GetSymbol(name)

	ident := &ast.BLangIdentifier{Value: name}
	ident.SetDeterminedType(semtypes.NEVER)
	ident.SetPosition(pos)

	variable := &ast.BLangSimpleVariable{Name: ident}
	variable.SetDeterminedType(ty)
	variable.SetSymbol(symRef)
	variable.SetInitialExpression(initExpr)
	variable.SetPosition(pos)

	varDef := &ast.BLangSimpleVariableDef{Var: variable}
	varDef.SetDeterminedType(semtypes.NEVER)
	varDef.SetPosition(pos)

	ref := &ast.BLangSimpleVarRef{VariableName: ident}
	ref.SetSymbol(symRef)
	ref.SetDeterminedType(ty)
	ref.SetPosition(pos)
	return varDef, ref
}

// createArrayPushInvocation builds an `array:push(<listExpr>, <valueExpr>)`
func createArrayPushInvocation(pkgCtx *packageContext, listExpr, valueExpr ast.BLangExpression) *ast.BLangInvocation {
	pkgName := "lang.array"
	space, ok := pkgCtx.getImportedSymbolSpace(pkgName)
	if !ok {
		pkgCtx.internalError(pkgName + " symbol space not found")
		return nil
	}
	pushRef, ok := space.GetSymbol("push")
	if !ok {
		pkgCtx.internalError(pkgName + ":push symbol not found")
		return nil
	}
	pushSym, ok := pkgCtx.getSymbol(pushRef).(*model.OpaqueFunctionSymbol)
	if !ok {
		pkgCtx.internalError(pkgName + ":push is not an opaque function symbol")
		return nil
	}
	pkgCtx.addImplicitImport(pkgName, ast.BLangImportPackage{
		OrgName:      &ast.BLangIdentifier{Value: "ballerina"},
		PkgNameComps: []ast.BLangIdentifier{{Value: "lang"}, {Value: "array"}},
		Alias:        &ast.BLangIdentifier{Value: pkgName},
	})
	inv := &ast.BLangInvocation{PkgAlias: &ast.BLangIdentifier{Value: pkgName}}
	inv.Name = &ast.BLangIdentifier{Value: pushSym.Name()}
	inv.ArgExprs = []ast.BLangExpression{listExpr, valueExpr}
	inv.SetSymbol(pushRef)
	inv.SetDeterminedType(semtypes.NIL)
	inv.SetPosition(valueExpr.GetPosition())
	return inv
}

func buildListenerStartInvocation(pkgCtx *packageContext, listenerExpr ast.BLangExpression) *ast.BLangInvocation {
	listenerTy := listenerExpr.GetDeterminedType()
	if semtypes.IsZero(listenerTy) {
		pkgCtx.internalError("listener expression has no determined type at desugar")
		return nil
	}
	startFnTy := semtypes.ObjectMemberType(pkgCtx.typeCtx(), semtypes.StringConst("start"), listenerTy)
	if semtypes.IsZero(startFnTy) {
		pkgCtx.internalError("listener type has no start method type at desugar")
		return nil
	}
	inv := &ast.BLangInvocation{}
	inv.Name = &ast.BLangIdentifier{Value: "start"}
	inv.Expr = listenerExpr
	argListDefn := semtypes.NewListDefinition()
	argListTy := argListDefn.DefineListTypeWrapped(pkgCtx.typeEnv(), []semtypes.SemType{}, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	inv.SetDeterminedType(semtypes.FunctionReturnType(pkgCtx.typeCtx(), startFnTy, argListTy))
	inv.SetPosition(listenerExpr.GetPosition())
	return inv
}

// buildListenerAttachInvocation produces an invocation expression
// `<listenerExpr>.attach(<svcRef>, <attachPoint>)` corresponding to a single
// (listener, service) pair.
func buildListenerAttachInvocation(pkgCtx *packageContext, svc *ast.BLangService, listenerExpr ast.BLangExpression, svcRef ast.BLangExpression) *ast.BLangInvocation {
	listenerTy := listenerExpr.GetDeterminedType()
	if semtypes.IsZero(listenerTy) {
		pkgCtx.internalError("listener expression has no determined type at desugar")
		return nil
	}
	tyCtx := pkgCtx.typeCtx()
	attachFnTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("attach"), listenerTy)
	if semtypes.IsZero(attachFnTy) {
		pkgCtx.internalError("listener type has no attach method type at desugar")
		return nil
	}
	attachPointExpr := buildAttachPointExpression(pkgCtx, svc)
	inv := &ast.BLangInvocation{}
	inv.Name = &ast.BLangIdentifier{Value: "attach"}
	inv.Expr = listenerExpr
	inv.ArgExprs = []ast.BLangExpression{svcRef, attachPointExpr}
	argListDefn := semtypes.NewListDefinition()
	argListTy := argListDefn.DefineListTypeWrapped(pkgCtx.typeEnv(), []semtypes.SemType{svcRef.GetDeterminedType(), attachPointExpr.GetDeterminedType()}, 2, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	inv.SetDeterminedType(semtypes.FunctionReturnType(tyCtx, attachFnTy, argListTy))
	inv.SetPosition(svc.GetPosition())
	return inv
}

// buildAttachPointExpression returns an AST expression representing the
// service's attach-point value: () for the absent case, the original string
// literal, or an array literal of the resource path segments.
func buildAttachPointExpression(pkgCtx *packageContext, svc *ast.BLangService) ast.BLangExpression {
	if svc.AttachPointLiteral != nil {
		return svc.AttachPointLiteral
	}
	if len(svc.AbsoluteResourcePath) == 0 {
		lit := &ast.BLangLiteral{Value: nil}
		lit.SetDeterminedType(semtypes.NIL)
		lit.SetPosition(svc.GetPosition())
		return lit
	}
	elements := make([]ast.BLangExpression, len(svc.AbsoluteResourcePath))
	tupleMembers := make([]semtypes.SemType, len(svc.AbsoluteResourcePath))
	for i := range svc.AbsoluteResourcePath {
		lit := &ast.BLangLiteral{Value: svc.AbsoluteResourcePath[i].Value}
		litTy := semtypes.StringConst(svc.AbsoluteResourcePath[i].Value)
		lit.SetDeterminedType(litTy)
		lit.SetPosition(svc.AbsoluteResourcePath[i].GetPosition())
		elements[i] = lit
		tupleMembers[i] = litTy
	}
	ld := semtypes.NewListDefinition()
	listTy := ld.DefineListTypeWrapped(pkgCtx.typeEnv(), tupleMembers, len(tupleMembers), semtypes.NEVER, semtypes.CellMutability_CELL_MUT_LIMITED)
	lat := semtypes.ToListAtomicType(pkgCtx.typeCtx(), listTy)
	arr := &ast.BLangListConstructorExpr{Exprs: elements, AtomicType: *lat}
	arr.SetDeterminedType(listTy)
	arr.SetPosition(svc.GetPosition())
	return arr
}

func newSimpleVariable(name string, ty semtypes.SemType) *ast.BLangSimpleVariable {
	v := &ast.BLangSimpleVariable{}
	v.Name = &ast.BLangIdentifier{Value: name}
	v.Name.SetDeterminedType(semtypes.NEVER)
	v.SetDeterminedType(ty)
	return v
}

func createDefaultValueFunction(name string, defaultExpr ast.BLangExpression) *ast.BLangFunction {
	retStmt := &ast.BLangReturn{Expr: defaultExpr}
	retStmt.SetDeterminedType(semtypes.NEVER)
	body := &ast.BLangBlockFunctionBody{Stmts: []ast.StatementNode{retStmt}}
	body.SetDeterminedType(semtypes.NEVER)

	fn := &ast.BLangFunction{}
	fn.Name = ast.BLangIdentifier{Value: name}
	fn.Name.SetDeterminedType(semtypes.NEVER)
	fn.Body = body
	fn.SetDeterminedType(semtypes.NEVER)
	setPositionIfMissing(fn, defaultExpr.GetPosition())
	return fn
}

type desugaredRecordFieldResult struct {
	fn     *ast.BLangFunction
	symRef model.SymbolRef
}

type desugaredTypeDescResult struct {
	recordFields []desugaredRecordFieldResult
}

func desugarTypeDesc(ctx desugarContext, typeDesc ast.BType, parentScope model.Scope) desugaredTypeDescResult {
	switch td := typeDesc.(type) {
	case *ast.BLangRecordType:
		return desugarRecordTypeDesc(ctx, td, parentScope)
	}
	return desugaredTypeDescResult{}
}

func desugarRecordTypeDesc(ctx desugarContext, recType *ast.BLangRecordType, parentScope model.Scope) desugaredTypeDescResult {
	var fields []desugaredRecordFieldResult
	for _, field := range recType.FieldPtrs() {
		if field.DefaultExpr == nil {
			continue
		}
		symRef := field.DefaultFnRef
		fn := createDefaultValueFunction(ctx.getSymbol(symRef).Name(), field.DefaultExpr)
		fnScope := ctx.newFunctionScope(parentScope)
		fn.SetSymbol(symRef)
		fn.SetScope(fnScope)

		fields = append(fields, desugaredRecordFieldResult{fn: fn, symRef: symRef})

	}
	return desugaredTypeDescResult{recordFields: fields}
}

func desugarTopLevelTypeDescs(cx *packageContext, pkg *ast.BLangPackage) {
	for i := range pkg.TypeDefinitions {
		defn := &pkg.TypeDefinitions[i]
		typeDesc, ok := defn.GetTypeData().TypeDescriptor.(ast.BType)
		if !ok {
			cx.internalError("type definition has no BType type descriptor")
			return
		}
		result := desugarTypeDesc(cx, typeDesc, nil)
		for _, rf := range result.recordFields {
			pkg.Functions = append(pkg.Functions, *rf.fn)
		}
	}
}

func desugarFunctionParamDefaults(ctx desugarContext, fn *ast.BLangFunction) []*ast.BLangFunction {
	fnSym := ctx.getSymbol(fn.Symbol()).(model.FunctionSymbol)
	defaultableParams := fnSym.DefaultableParams()
	var results []*ast.BLangFunction
	for j := range fn.RequiredParams {
		param := &fn.RequiredParams[j]
		dp, ok := defaultableParams.Get(j)
		if !ok {
			if param.IsDefaultableParam() {
				ctx.internalError("defaultable param info missing for parameter marked as defaultable")
			}
			continue
		}
		if dp.Kind == model.DefaultableParamKindInferredTypedesc {
			continue
		}
		symRef := dp.Symbol
		fnName := ctx.getSymbol(symRef).Name()
		fnScope := ctx.newFunctionScope(fn.Scope())

		defaultFn := createDefaultValueFunction(fnName, param.Expr.(ast.BLangExpression))
		defaultFn.SetSymbol(symRef)
		defaultFn.SetScope(fnScope)

		symbolMapping := make(map[model.SymbolRef]model.SymbolRef)
		for k := range fn.RequiredParams[:j] {
			precedingParam := fn.RequiredParams[k]
			paramName := precedingParam.Name.Value
			paramTy := ctx.symbolType(precedingParam.Symbol())
			newParam := newSimpleVariable(paramName, paramTy)
			newParam.SetRequiredParam()
			fnScope.AddSymbol(paramName, new(model.NewVariableSymbol(paramName, false, false, true)))
			paramSymRef, _ := fnScope.GetSymbol(paramName)
			ctx.setSymbolType(paramSymRef, paramTy)
			newParam.SetSymbol(paramSymRef)
			defaultFn.AddParameter(newParam)
			symbolMapping[precedingParam.Symbol()] = paramSymRef
		}
		remapSymbolRefs(defaultFn.Body.(ast.BLangNode), symbolMapping)

		results = append(results, defaultFn)
	}
	return results
}

func desugarTopLevelFunctionDefaults(pkgCtx *packageContext, pkg *ast.BLangPackage) {
	fnCount := len(pkg.Functions)
	for i := range fnCount {
		for _, fn := range desugarFunctionParamDefaults(pkgCtx, &pkg.Functions[i]) {
			pkg.Functions = append(pkg.Functions, *fn)
		}
	}
}

func desugarClassMethodDefaults(pkgCtx *packageContext, pkg *ast.BLangPackage) {
	desugarObjectMethodDefaults := func(initFn *ast.BLangFunction, methods map[string]*ast.BLangFunction) {
		if initFn != nil {
			for _, fn := range desugarFunctionParamDefaults(pkgCtx, initFn) {
				pkg.Functions = append(pkg.Functions, *fn)
			}
		}
		for _, method := range methods {
			for _, fn := range desugarFunctionParamDefaults(pkgCtx, method) {
				pkg.Functions = append(pkg.Functions, *fn)
			}
		}
	}
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		desugarObjectMethodDefaults(classDef.InitFunction, classDef.Methods)
	}
	for i := range pkg.Services {
		svc := &pkg.Services[i]
		desugarObjectMethodDefaults(svc.InitFunction, svc.Methods)
	}
}

type symbolRemapper struct {
	mapping map[model.SymbolRef]model.SymbolRef
}

func (r symbolRemapper) Visit(node ast.BLangNode) ast.Visitor {
	if ref, ok := node.(ast.BNodeWithSymbol); ok {
		oldSym := ref.Symbol()
		if newSym, found := r.mapping[oldSym]; found {
			ref.SetSymbol(newSym)
		}
	}
	return r
}

func (r symbolRemapper) VisitTypeData(_ *ast.TypeData) ast.Visitor {
	return r
}

// remapSymbolRefs updates symbols based on the mapping given
func remapSymbolRefs(node ast.BLangNode, mapping map[model.SymbolRef]model.SymbolRef) {
	if len(mapping) == 0 {
		return
	}
	ast.Walk(symbolRemapper{mapping: mapping}, node)
}

// DesugarPackage returns a desugared package (may be new or same instance)
func DesugarPackage(compilerCtx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) *ast.BLangPackage {
	if importedSymbols == nil {
		importedSymbols = make(map[string]model.ExportedSymbolSpace)
	}
	pkgCtx := newPackageContext(compilerCtx, pkg, importedSymbols)

	var wg sync.WaitGroup
	var panicErr any

	desugarFn := func(fn *ast.BLangFunction) {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					panicErr = r
				}
			}()
			*fn = *desugarFunction(pkgCtx, fn)
		})
	}

	// Desugar type definition default expressions into standalone functions
	desugarTopLevelTypeDescs(pkgCtx, pkg)

	desugarTopLevelFunctionDefaults(pkgCtx, pkg)
	desugarClassMethodDefaults(pkgCtx, pkg)

	desugarObjectDefinitionConcurrently := func(class *ast.BLangClassDefinition) {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					panicErr = r
				}
			}()
			desugarClassDefinition(pkgCtx, class)
			for name, method := range class.Methods {
				class.Methods[name] = desugarFunction(pkgCtx, method)
			}
			for _, rm := range class.ResourceMethods {
				desugarResourceMethod(pkgCtx, rm)
			}
			*class.InitFunction = *desugarFunction(pkgCtx, class.InitFunction)
		})
	}
	desugarServiceConcurrently := func(svc *ast.BLangService) {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					panicErr = r
				}
			}()
			desugarServiceDefinition(pkgCtx, svc)
			for name, method := range svc.Methods {
				svc.Methods[name] = desugarFunction(pkgCtx, method)
			}
			for _, rm := range svc.ResourceMethods {
				desugarResourceMethod(pkgCtx, rm)
			}
			*svc.InitFunction = *desugarFunction(pkgCtx, svc.InitFunction)
		})
	}
	for i := range pkg.Services {
		ensureServiceDefaultInitFunction(pkgCtx, &pkg.Services[i])
	}

	hoistInlineServiceListeners(pkgCtx, pkg)
	desugarInitFn(pkgCtx, compilerCtx, pkg)

	// Desugar all functions after desugarInitFn has created any lifecycle hooks.
	for i := range pkg.Functions {
		desugarFn(&pkg.Functions[i])
	}

	// Desugar class definitions (each class concurrently, members sequentially)
	for i := range pkg.ClassDefinitions {
		desugarObjectDefinitionConcurrently(&pkg.ClassDefinitions[i])
	}
	for i := range pkg.Services {
		desugarServiceConcurrently(&pkg.Services[i])
	}

	wg.Wait()
	if panicErr != nil {
		panic(panicErr)
	}

	pkg.Constants = nil
	return pkg
}

func desugarClassDefinition(pkgCtx *packageContext, class *ast.BLangClassDefinition) {
	if class.InitFunction == nil {
		class.InitFunction = synthesizeDefaultInitFunction(pkgCtx, class.Scope(), class.GetPosition())
	}
	desugarClassBodyInit(pkgCtx, class.Scope(), class.Fields, class.InitFunction)
}

func desugarServiceDefinition(pkgCtx *packageContext, svc *ast.BLangService) {
	// svc.InitFunction is guaranteed non-nil by the ensureServiceDefaultInitFunction pre-pass.
	desugarClassBodyInit(pkgCtx, svc.Scope(), svc.Fields, svc.InitFunction)
}

func synthesizeDefaultInitFunction(pkgCtx *packageContext, classScope model.Scope, pos diagnostics.Location) *ast.BLangFunction {
	fn := ast.BLangFunction{}
	fn.SetAttached()
	fn.Name = ast.BLangIdentifier{Value: "init"}
	body := &ast.BLangBlockFunctionBody{}
	body.SetPosition(pos)
	fn.Body = body
	fn.SetDeterminedType(semtypes.NEVER)
	fn.SetScope(pkgCtx.newFunctionScope(classScope))
	fn.SetPosition(pos)
	initSymbol := model.NewFunctionSymbol("init", model.FunctionSignature{ReturnType: semtypes.NIL}, false)
	classScope.AddSymbol("init", initSymbol)
	symRef, _ := classScope.GetSymbol("init")
	fn.SetSymbol(symRef)
	return &fn
}

// We are doing this seperately unlike class to avoid race conditions, service init it needed for module init
func ensureServiceDefaultInitFunction(pkgCtx *packageContext, svc *ast.BLangService) {
	if svc.InitFunction != nil {
		return
	}
	svc.InitFunction = synthesizeDefaultInitFunction(pkgCtx, svc.Scope(), svc.GetPosition())
}

func desugarClassBodyInit(pkgCtx *packageContext, classScope model.Scope, fields []ast.SimpleVariableNode, initFn *ast.BLangFunction) {
	selfRef, ok := classScope.GetSymbol("self")
	if !ok {
		pkgCtx.internalError("self symbol not found in class scope")
		return
	}
	classType := pkgCtx.getSymbol(selfRef).Type()

	var initStmts []ast.StatementNode
	for _, field := range fields {
		initExpr := field.GetInitialExpression()
		if initExpr == nil {
			continue
		}
		initExprBal := initExpr.(ast.BLangExpression)
		basePos := initExprBal.GetPosition()

		selfVarRef := &ast.BLangSimpleVarRef{
			VariableName: &ast.BLangIdentifier{Value: "self"},
		}
		selfVarRef.SetSymbol(selfRef)
		selfVarRef.SetDeterminedType(classType)

		fieldAccess := &ast.BLangFieldBaseAccess{
			Field: ast.BLangIdentifier{Value: field.GetName().GetValue()},
		}
		fieldAccess.Field.SetDeterminedType(semtypes.NEVER)
		fieldAccess.Expr = selfVarRef
		fieldAccess.SetDeterminedType(pkgCtx.getSymbolType(field.Symbol()))

		assignment := &ast.BLangAssignment{
			VarRef: fieldAccess,
			Expr:   initExprBal,
		}
		assignment.SetDeterminedType(semtypes.NEVER)
		setPositionIfMissing(assignment, basePos)

		initStmts = append(initStmts, assignment)
		field.(*ast.BLangSimpleVariable).SetInitialExpression(nil)
	}

	if len(initStmts) > 0 {
		body := initFn.Body.(*ast.BLangBlockFunctionBody)
		body.Stmts = append(initStmts, body.Stmts...)
	}
}

func desugarResourceMethod(pkgCtx *packageContext, rm *ast.BLangResourceMethod) {
	if rm.Body == nil {
		return
	}
	cx := &functionContext{pkgCtx: pkgCtx}
	cx.pushScope(rm.Scope())
	defer cx.popScope()
	switch body := rm.Body.(type) {
	case *ast.BLangBlockFunctionBody:
		walkBlockFunctionBody(cx, body)
	case *ast.BLangExprFunctionBody:
		result := walkExpression(cx, body.Expr.(ast.BLangActionOrExpression))
		if len(result.initStmts) > 0 {
			rm.Body = convertExprBodyToBlockBody(body, result)
		} else {
			body.Expr = result.replacementNode.(ast.BLangExpression)
		}
	}
}

// desugarFunction returns a desugared function (may be same or new instance)
func desugarFunction(pkgCtx *packageContext, fn *ast.BLangFunction) *ast.BLangFunction {
	if fn.Body == nil {
		return fn
	}

	cx := &functionContext{
		pkgCtx: pkgCtx,
	}

	// Push function scope
	cx.pushScope(fn.Scope())
	defer cx.popScope()

	switch body := fn.Body.(type) {
	case *ast.BLangBlockFunctionBody:
		walkBlockFunctionBody(cx, body)
	case *ast.BLangExprFunctionBody:
		if body.Expr != nil {
			result := walkExpression(cx, body.Expr.(ast.BLangActionOrExpression))
			// For expression bodies, init statements need special handling
			// They should be converted to a block body with statements
			if len(result.initStmts) > 0 {
				fn.Body = convertExprBodyToBlockBody(body, result)
			} else {
				body.Expr = result.replacementNode.(ast.BLangExpression)
			}
		}
	case *ast.BLangExternFunctionBody:
		// Nothing to desugar
	}

	return fn
}

// convertExprBodyToBlockBody converts expression function body to block body
// when there are init statements from desugaring
func convertExprBodyToBlockBody(
	exprBody *ast.BLangExprFunctionBody,
	result desugaredNode[ast.BLangActionOrExpression],
) *ast.BLangBlockFunctionBody {
	// Create return statement with the desugared expression
	returnStmt := &ast.BLangReturn{
		Expr: result.replacementNode,
	}

	// Build block with init statements + return
	stmts := make([]ast.StatementNode, 0, len(result.initStmts)+1)
	stmts = append(stmts, result.initStmts...)
	stmts = append(stmts, returnStmt)

	return &ast.BLangBlockFunctionBody{
		Stmts: stmts,
	}
}

// BLangServiceInit is a desugar-only expression that constructs an
// instance of the (anonymous) class body of the referenced service.
// It is never produced by user source: services have no name and are
// not addressable via `new`. The desugarer emits this in place of
// the `new <class>()` it would emit for a named class.
type BLangServiceInit struct {
	ast.AbstractExpression
	Service *ast.BLangService
}

var (
	_ ast.BLangExpression = &BLangServiceInit{}
	_ ast.BLangNode       = &BLangServiceInit{}
)
