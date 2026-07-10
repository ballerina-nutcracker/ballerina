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
	"maps"
	"slices"
	"strings"
	"sync"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"

	langinternal "ballerina-lang-go/lib/langinternal/compile"
)

type scopeKind int

const (
	moduleScopeKind scopeKind = iota
	blockScopeKind
)

type varStatus uint8

const (
	varDeclared varStatus = iota
	varUsed
)

type varStatusTracker interface {
	markInit(sym model.SymbolRef, pos diagnostics.Location)
	markUsed(sym model.SymbolRef)
	getUnused() []varDeclInfo
}

type varDeclInfo struct {
	varSym model.SymbolRef
	pos    diagnostics.Location
}

type symbolResolver interface {
	varStatusTracker
	GetSymbol(name string) (model.SymbolRef, scopeKind, bool)
	ast.Visitor
	GetPrefixedSymbol(prefix, name string) (model.SymbolRef, bool)
	GetAnnotationSymbol(prefix, name string) (model.SymbolRef, bool)
	AddSymbol(name string, symbol model.Symbol)
	GetPkgID() model.PackageID
	GetScope() model.Scope
	GetCtx() *context.CompilerContext
	TypeContext() semtypes.Context
	GetTypeDefns() map[model.SymbolRef]*ast.BLangTypeDefinition
	GetClassDefns() map[model.SymbolRef]*ast.BLangClassDefinition
}

type (
	CompilationUnitImports struct {
		CompilationUnit *ast.BLangCompilationUnit
		Imports         map[string]model.ExportedSymbolSpace
	}

	defaultSymbolAllocator interface {
		GetCtx() *context.CompilerContext
		nextDefaultSymbolName() string
	}

	prevPos struct {
		pos      diagnostics.Location
		reported bool
	}

	varTracker struct {
		varIndex  map[model.SymbolRef]int
		declPos   []diagnostics.Location
		varStatus []varStatus
		symbol    []model.SymbolRef
	}

	moduleAstNode[T ast.BLangNode] struct {
		node     T
		resolver *moduleSymbolResolver
	}

	moduleAstNodeHolder struct {
		typeDefns  map[string]moduleAstNode[*ast.BLangTypeDefinition]
		classDefns map[string]moduleAstNode[*ast.BLangClassDefinition]
	}

	moduleSymbolResolver struct {
		ctx            *context.CompilerContext
		tyCtx          semtypes.Context
		scope          *model.ModuleScope
		packageScope   *model.ModuleScope
		pkgID          model.PackageID
		typeDefns      map[model.SymbolRef]*ast.BLangTypeDefinition
		classDefns     map[model.SymbolRef]*ast.BLangClassDefinition
		packageSymbols map[string]model.SymbolRef
		prevPos        map[string]prevPos
		prevAnnotPos   map[string]prevPos
		usedPrefixes   map[string]bool
		defaultCounter int
		varTracker     varTracker
		moduleNodes    moduleAstNodeHolder
	}

	blockSymbolResolver struct {
		parent     symbolResolver
		scope      model.BlockLevelScope
		node       ast.BLangNode
		varTracker *varTracker
	}
)

var (
	_ symbolResolver   = &moduleSymbolResolver{}
	_ symbolResolver   = &blockSymbolResolver{}
	_ varStatusTracker = &varTracker{}
)

func markInit(resolver symbolResolver, name string, symbol model.SymbolRef, pos diagnostics.Location) {
	if isIgnoredDeclName(name) {
		return
	}
	resolver.markInit(symbol, pos)
}

func (r *moduleSymbolResolver) markInit(sym model.SymbolRef, pos diagnostics.Location) {
	r.varTracker.markInit(sym, pos)
}

func (r *moduleSymbolResolver) markUsed(sym model.SymbolRef) {
	if r.varTracker.isTracked(sym) {
		r.varTracker.markUsed(sym)
	}
}

func (r *moduleSymbolResolver) getUnused() []varDeclInfo {
	return r.varTracker.getUnused()
}

func (r *blockSymbolResolver) markInit(sym model.SymbolRef, pos diagnostics.Location) {
	if r.varTracker == nil {
		r.parent.markInit(sym, pos)
		return
	}
	r.varTracker.markInit(sym, pos)
}

func (r *blockSymbolResolver) markUsed(sym model.SymbolRef) {
	tracker := r.varTracker
	if tracker == nil {
		r.parent.markUsed(sym)
		return
	}
	if tracker.isTracked(sym) {
		tracker.markUsed(sym)
		return
	}
	r.parent.markUsed(sym)
}

func (r *blockSymbolResolver) getUnused() []varDeclInfo {
	return r.varTracker.getUnused()
}

func (t *varTracker) isTracked(sym model.SymbolRef) bool {
	if t.varIndex == nil {
		return false
	}
	_, ok := t.varIndex[sym]
	return ok
}

func (t *varTracker) markInit(sym model.SymbolRef, pos diagnostics.Location) {
	index := len(t.symbol)
	if t.varIndex == nil {
		t.varIndex = make(map[model.SymbolRef]int)
	}
	t.varIndex[sym] = index
	t.symbol = append(t.symbol, sym)
	t.declPos = append(t.declPos, pos)
	t.varStatus = append(t.varStatus, varDeclared)
}

func (t *varTracker) markUsed(sym model.SymbolRef) {
	index := t.varIndex[sym]
	t.varStatus[index] = varUsed
}

func (t *varTracker) getUnused() []varDeclInfo {
	var res []varDeclInfo
	for i := range len(t.symbol) {
		status := t.varStatus[i]
		if status == varUsed {
			continue
		}
		res = append(res, varDeclInfo{t.symbol[i], t.declPos[i]})
	}
	return res
}

func newCompilationUnitsSymbolResolver(ctx *context.CompilerContext, pkgID model.PackageID) *moduleSymbolResolver {
	packageScope := ctx.NewModuleScope(pkgID, nil)
	return &moduleSymbolResolver{
		ctx:            ctx,
		tyCtx:          semtypes.ContextFrom(ctx.GetTypeEnv()),
		scope:          packageScope,
		packageScope:   packageScope,
		pkgID:          pkgID,
		typeDefns:      make(map[model.SymbolRef]*ast.BLangTypeDefinition),
		classDefns:     make(map[model.SymbolRef]*ast.BLangClassDefinition),
		packageSymbols: make(map[string]model.SymbolRef),
		prevPos:        make(map[string]prevPos),
		prevAnnotPos:   make(map[string]prevPos),
		usedPrefixes:   make(map[string]bool),
		moduleNodes: moduleAstNodeHolder{
			typeDefns:  make(map[string]moduleAstNode[*ast.BLangTypeDefinition]),
			classDefns: make(map[string]moduleAstNode[*ast.BLangClassDefinition]),
		},
	}
}

func (m *moduleAstNodeHolder) add(cu *ast.BLangCompilationUnit, resolver *moduleSymbolResolver) {
	for _, node := range cu.TopLevelNodes {
		switch n := node.(type) {
		case *ast.BLangTypeDefinition:
			m.typeDefns[n.Name.Value] = moduleAstNode[*ast.BLangTypeDefinition]{node: n, resolver: resolver}
		case *ast.BLangClassDefinition:
			m.classDefns[n.Name.Value] = moduleAstNode[*ast.BLangClassDefinition]{node: n, resolver: resolver}
		}
	}
}

func (ms *moduleSymbolResolver) forCompilationUnit(scope *model.ModuleScope) *moduleSymbolResolver {
	return &moduleSymbolResolver{
		ctx:            ms.ctx,
		tyCtx:          ms.tyCtx,
		scope:          scope,
		packageScope:   ms.packageScope,
		pkgID:          ms.pkgID,
		typeDefns:      ms.typeDefns,
		classDefns:     ms.classDefns,
		packageSymbols: ms.packageSymbols,
		prevPos:        ms.prevPos,
		prevAnnotPos:   ms.prevAnnotPos,
		usedPrefixes:   make(map[string]bool),
		defaultCounter: ms.defaultCounter,
		varTracker:     ms.varTracker,
		moduleNodes:    ms.moduleNodes,
	}
}

func newFunctionResolver(parent symbolResolver, node ast.BLangNode) *blockSymbolResolver {
	pkgID := parent.GetPkgID()
	parentScope := parent.GetScope()
	scope := parent.GetCtx().NewFunctionScope(parentScope, pkgID)
	return &blockSymbolResolver{
		parent:     parent,
		scope:      scope,
		node:       node,
		varTracker: new(varTracker{}),
	}
}

func newBlockSymbolResolverWithBlockScope(parent symbolResolver, node ast.BLangNode) *blockSymbolResolver {
	pkgID := parent.GetPkgID()
	parentScope := parent.GetScope()
	scope := parent.GetCtx().NewBlockScope(parentScope, pkgID)
	return &blockSymbolResolver{
		parent: parent,
		scope:  scope,
		node:   node,
	}
}

func (ms *moduleSymbolResolver) GetSymbol(name string) (model.SymbolRef, scopeKind, bool) {
	if ref, ok := ms.packageSymbols[name]; ok {
		return ref, moduleScopeKind, true
	}
	ref, ok := ms.packageScope.Main.GetSymbol(name)
	return ref, moduleScopeKind, ok
}

func (ms *moduleSymbolResolver) GetSymbolFromCurrentScope(name string) (model.SymbolRef, scopeKind, bool) {
	ref, ok := ms.scope.Main.GetSymbol(name)
	return ref, moduleScopeKind, ok
}

func (ms *moduleSymbolResolver) GetPkgID() model.PackageID {
	return ms.pkgID
}

func (ms *moduleSymbolResolver) GetScope() model.Scope {
	return ms.scope
}

func (ms *moduleSymbolResolver) GetPrefixedSymbol(prefix, name string) (model.SymbolRef, bool) {
	if prefix != "" {
		ms.usedPrefixes[prefix] = true
	}
	return ms.scope.GetPrefixedSymbol(prefix, name)
}

func (ms *moduleSymbolResolver) GetAnnotationSymbol(prefix, name string) (model.SymbolRef, bool) {
	if prefix != "" {
		ms.usedPrefixes[prefix] = true
	}
	return ms.scope.GetAnnotationSymbol(prefix, name)
}

func (ms *moduleSymbolResolver) AddSymbol(name string, symbol model.Symbol) {
	ms.scope.AddSymbol(name, symbol)
}

func (ms *moduleSymbolResolver) GetCtx() *context.CompilerContext {
	return ms.ctx
}

func (ms *moduleSymbolResolver) TypeContext() semtypes.Context {
	return ms.tyCtx
}

func (ms *moduleSymbolResolver) nextDefaultSymbolName() string {
	name := fmt.Sprintf("$default$%d", ms.defaultCounter)
	ms.defaultCounter++
	return name
}

func (ms *moduleSymbolResolver) GetTypeDefns() map[model.SymbolRef]*ast.BLangTypeDefinition {
	return ms.typeDefns
}

func (ms *moduleSymbolResolver) GetClassDefns() map[model.SymbolRef]*ast.BLangClassDefinition {
	return ms.classDefns
}

func (bs *blockSymbolResolver) GetSymbol(name string) (model.SymbolRef, scopeKind, bool) {
	ref, ok := bs.scope.MainSpace().GetSymbol(name)
	if ok {
		return ref, blockScopeKind, true
	}
	return bs.parent.GetSymbol(name)
}

func (bs *blockSymbolResolver) GetPrefixedSymbol(prefix, name string) (model.SymbolRef, bool) {
	return bs.parent.GetPrefixedSymbol(prefix, name)
}

func (bs *blockSymbolResolver) GetAnnotationSymbol(prefix, name string) (model.SymbolRef, bool) {
	return bs.parent.GetAnnotationSymbol(prefix, name)
}

func (bs *blockSymbolResolver) AddSymbol(name string, symbol model.Symbol) {
	bs.scope.AddSymbol(name, symbol)
}

func (bs *blockSymbolResolver) GetPkgID() model.PackageID {
	return bs.parent.GetPkgID()
}

func (bs *blockSymbolResolver) GetScope() model.Scope {
	return bs.scope
}

func (bs *blockSymbolResolver) GetCtx() *context.CompilerContext {
	return bs.parent.GetCtx()
}

func (bs *blockSymbolResolver) TypeContext() semtypes.Context {
	return bs.parent.TypeContext()
}

func (bs *blockSymbolResolver) GetTypeDefns() map[model.SymbolRef]*ast.BLangTypeDefinition {
	return bs.parent.GetTypeDefns()
}

func (bs *blockSymbolResolver) GetClassDefns() map[model.SymbolRef]*ast.BLangClassDefinition {
	return bs.parent.GetClassDefns()
}

// isIgnoredDeclName reports whether a name should be excluded from unused-variable tracking.
// The IGNORE name (`_`) is the user-facing opt-out; names beginning with `$` are compiler
// generated (default-param synthetic functions, desugar temporaries, etc.) and never user-visible.
func isIgnoredDeclName(name string) bool {
	if name == string(model.IGNORE) {
		return true
	}
	if len(name) > 0 && name[0] == '$' {
		return true
	}
	return false
}

func addTopLevelSymbol(resolver *moduleSymbolResolver, name string, symbol model.Symbol, pos diagnostics.Location) bool {
	if prevRef, _, exists := resolver.GetSymbol(name); exists {
		resolver.markUsed(prevRef)
		msg := "redeclared symbol '" + name + "'"
		if prev, ok := resolver.prevPos[name]; ok && !prev.reported {
			semanticError(resolver, msg, prev.pos)
			prev.reported = true
			resolver.prevPos[name] = prev
		}
		semanticError(resolver, msg, pos)
		return false
	}
	resolver.AddSymbol(name, symbol)
	ref, _, _ := resolver.GetSymbolFromCurrentScope(name)
	resolver.packageSymbols[name] = ref
	resolver.prevPos[name] = prevPos{pos: pos}
	return true
}

func addTopLevelAnnotationSymbol(resolver *moduleSymbolResolver, name string, symbol model.Symbol, pos diagnostics.Location) bool {
	if _, exists := resolver.scope.Annotation.GetSymbol(name); exists {
		msg := "redeclared annotation '" + name + "'"
		if prev, ok := resolver.prevAnnotPos[name]; ok && !prev.reported {
			semanticError(resolver, msg, prev.pos)
			prev.reported = true
			resolver.prevAnnotPos[name] = prev
		}
		semanticError(resolver, msg, pos)
		return false
	}
	resolver.scope.AddAnnotationSymbol(name, symbol)
	resolver.prevAnnotPos[name] = prevPos{pos: pos}
	return true
}

func annotationAttachPointKey(attachPoint ast.AttachPoint) string {
	point := attachPoint.Point.String()
	if attachPoint.Source {
		return model.SourceAnnotationAttachPointKey(point)
	}
	return point
}

func (ms *moduleSymbolResolver) isTypeRefToTypedesc(ref *ast.BLangUserDefinedType, visited map[model.SymbolRef]bool) bool {
	pkgAlias, typeName := ref.PkgAlias.Value, ref.TypeName.Value
	if pkgAlias != "" {
		symRef, ok := ms.GetPrefixedSymbol(pkgAlias, typeName)
		if !ok {
			return false
		}
		ty := ms.ctx.GetSymbol(symRef).Type()
		return !semtypes.IsZero(ty) && semtypes.IsSubtype(ms.tyCtx, ty, semtypes.TYPEDESC)
	}
	symRef, _, ok := ms.GetSymbol(typeName)
	if !ok {
		return false
	}
	if visited[symRef] {
		return false
	}
	visited[symRef] = true
	td, ok := ms.typeDefns[symRef]
	if !ok {
		return false
	}
	return ms.isDescriptorTypedesc(td.GetTypeData().TypeDescriptor, visited)
}

// isDescriptorTypedesc reports whether a type descriptor AST node is (directly or via a user-
// defined reference chain) a typedesc type.
func (ms *moduleSymbolResolver) isDescriptorTypedesc(desc any, visited map[model.SymbolRef]bool) bool {
	switch tn := desc.(type) {
	case *ast.BLangValueType:
		return tn.TypeKind == ast.TypeKind_TYPEDESC
	case *ast.BLangBuiltInRefTypeNode:
		return tn.TypeKind == ast.TypeKind_TYPEDESC
	case *ast.BLangConstrainedType:
		return tn.ConstraintKind() == ast.TypeKind_TYPEDESC
	case *ast.BLangUserDefinedType:
		return ms.isTypeRefToTypedesc(tn, visited)
	}
	return false
}

// allocateFunctionSymbolInner creates the appropriate function symbol for a function declaration.
// If the return type references a typedesc parameter (dependently-typed), it creates a
// DependentlyTypedFunctionSymbol; otherwise a plain FunctionSymbol. The returned symbol has
// no type information yet — it is filled during type resolution.
func (ms *moduleSymbolResolver) allocateFunctionSymbolInner(fn *ast.BLangFunction, name string, isPublic bool) model.Symbol {
	paramNames := make([]string, len(fn.RequiredParams))
	for i := range fn.RequiredParams {
		paramNames[i] = fn.RequiredParams[i].GetName().GetValue()
	}
	if ms.isDependentlyTyped(fn) {
		if fn.RestParam != nil {
			ms.ctx.Unimplemented("rest parameters are not supported on dependently-typed functions", fn.GetPosition())
		}
		return model.NewDependentlyTypedFunctionSymbol(name, paramNames, len(fn.RequiredParams), fn.FuncSymbolFlags(), isPublic)
	}
	return model.NewFunctionSymbol(name, model.FunctionSignature{}, isPublic)
}

// isDependentlyTyped reports whether a function's return type references one of its typedesc
// parameters by name.
func (ms *moduleSymbolResolver) isDependentlyTyped(fn *ast.BLangFunction) bool {
	retTd := fn.GetReturnTypeDescriptor()
	if retTd == nil {
		return false
	}
	typedescParams := make(map[string]struct{})
	for i := range fn.RequiredParams {
		param := &fn.RequiredParams[i]
		if param.Name == nil {
			continue
		}
		if ms.isDescriptorTypedesc(param.TypeNode(), make(map[model.SymbolRef]bool)) {
			typedescParams[param.Name.Value] = struct{}{}
		}
	}
	if len(typedescParams) == 0 {
		return false
	}
	return returnTypeReferencesTypedescParam(retTd, typedescParams)
}

func returnTypeReferencesTypedescParam(node ast.BLangNode, typedescParams map[string]struct{}) bool {
	switch n := node.(type) {
	case *ast.BLangReturnTypeDescriptor:
		return returnTypeReferencesTypedescParam(n.TypeDescriptor, typedescParams)
	case *ast.BLangUserDefinedType:
		if n.PkgAlias.Value != "" {
			return false
		}
		_, ok := typedescParams[n.TypeName.Value]
		return ok
	case *ast.BLangUnionTypeNode:
		if lhs, ok := n.Lhs().TypeDescriptor.(ast.BLangNode); ok && returnTypeReferencesTypedescParam(lhs, typedescParams) {
			return true
		}
		if rhs, ok := n.Rhs().TypeDescriptor.(ast.BLangNode); ok && returnTypeReferencesTypedescParam(rhs, typedescParams) {
			return true
		}
	case *ast.BLangIntersectionTypeNode:
		if lhs, ok := n.Lhs().TypeDescriptor.(ast.BLangNode); ok && returnTypeReferencesTypedescParam(lhs, typedescParams) {
			return true
		}
		if rhs, ok := n.Rhs().TypeDescriptor.(ast.BLangNode); ok && returnTypeReferencesTypedescParam(rhs, typedescParams) {
			return true
		}
	}
	return false
}

func addSymbolAndSetOnNode[T symbolResolver](resolver T, name string, symbol model.Symbol, node ast.BNodeWithSymbol) {
	resolver.AddSymbol(name, symbol)
	symRef, _, _ := resolver.GetSymbol(name)
	node.SetSymbol(symRef)
}

func ResolveSymbols(cx *context.CompilerContext, pkgID model.PackageID, cuImportsList []CompilationUnitImports) (model.Scope, model.ExportedSymbolSpace) {
	packageResolver := newCompilationUnitsSymbolResolver(cx, pkgID)
	injectOpaqueSymbols(pkgID, packageResolver)
	cuResolvers := make([]*moduleSymbolResolver, len(cuImportsList))
	for i, cuImports := range cuImportsList {
		scope := cx.NewModuleScope(pkgID, cuImports.Imports)
		cuImports.CompilationUnit.Scope = scope
		cuResolvers[i] = packageResolver.forCompilationUnit(scope)
		packageResolver.moduleNodes.add(cuImports.CompilationUnit, cuResolvers[i])
	}
	for i, resolver := range cuResolvers {
		resolver.allocateTopLevelSymbols(cuImportsList[i].CompilationUnit)
	}

	for i, cuImports := range cuImportsList {
		cu := cuImports.CompilationUnit
		resolver := cuResolvers[i]
		processCompilationUnitXMLNS(resolver, cu)
		ast.Walk(resolver, cu)
		reportUnusedImports(resolver, compilationUnitImports(cu))
		reportUnusedVariables(cx, resolver.getUnused())
	}

	mainSpaces := make([]*model.SymbolSpace, 0, len(cuResolvers)+1)
	annotationSpaces := make([]*model.SymbolSpace, 0, len(cuResolvers)+1)
	for _, resolver := range cuResolvers {
		mainSpaces = append(mainSpaces, resolver.scope.Main)
		annotationSpaces = append(annotationSpaces, resolver.scope.Annotation)
	}
	mainSpaces = append(mainSpaces, packageResolver.packageScope.Main)
	annotationSpaces = append(annotationSpaces, packageResolver.packageScope.Annotation)
	pkgScope := &model.PackageScope{Virtual: packageResolver.packageScope, MainSpaces: mainSpaces}
	return pkgScope, model.NewExportedSymbolSpaces(mainSpaces, annotationSpaces)
}

func (ms *moduleSymbolResolver) allocateTopLevelSymbols(cu *ast.BLangCompilationUnit) {
	for _, node := range cu.TopLevelNodes {
		switch n := node.(type) {
		case *ast.BLangTypeDefinition:
			ms.allocateTypeSymbol(n, make(map[string]struct{}))
		case *ast.BLangFunction:
			ms.allocateFunctionSymbol(n)
		case *ast.BLangConstant:
			ms.allocateConstantSymbol(n)
		case *ast.BLangSimpleVariable:
			ms.allocateGlobalVarSymbol(n)
		case *ast.BLangClassDefinition:
			ms.allocateClassSymbol(n)
		case *ast.BLangAnnotation:
			ms.allocateAnnotationSymbol(n)
		}
	}
}

func (ms *moduleSymbolResolver) allocateTypeSymbol(typeDef *ast.BLangTypeDefinition, seen map[string]struct{}) {
	name := typeDef.Name.Value
	if ref, ok := ms.packageSymbols[name]; ok {
		if existing, ok := ms.typeDefns[ref]; ok && existing == typeDef {
			return
		}
	}
	isPublic := typeDef.IsPublic()
	var symbol model.Symbol
	switch ty := typeDef.GetTypeData().TypeDescriptor.(type) {
	case *ast.BLangRecordType:
		symbol = new(model.NewRecordSymbol(name, isPublic))
	case *ast.BLangObjectType:
		symbol = new(model.NewObjectTypeSymbol(name, isPublic))
	case *ast.BLangErrorTypeNode:
		symbol = new(model.NewErrorTypeSymbol(name, isPublic))
	case *ast.BLangUserDefinedType:
		seen[name] = struct{}{}
		if ty.ModulePrefix() == "" {
			ms.ensureTypeAllocated(ty, seen)
		}
		var symRef model.SymbolRef
		var ok bool
		if ty.ModulePrefix() != "" {
			symRef, ok = ms.GetPrefixedSymbol(ty.ModulePrefix(), ty.Name())
		} else {
			symRef, _, ok = ms.GetSymbol(ty.Name())
		}
		if !ok {
			symbol = new(model.NewTypeSymbol(name, isPublic))
			break
		}
		switch ms.ctx.GetSymbol(symRef).(type) {
		case *model.RecordSymbol:
			symbol = new(model.NewRecordSymbol(name, isPublic))
		case *model.ObjectTypeSymbol:
			symbol = new(model.NewObjectTypeSymbol(name, isPublic))
		case *model.ErrorTypeSymbol:
			symbol = new(model.NewErrorTypeSymbol(name, isPublic))
		default:
			symbol = new(model.NewTypeSymbol(name, isPublic))
		}
	default:
		symbol = new(model.NewTypeSymbol(name, isPublic))
	}
	if !addTopLevelSymbol(ms, name, symbol, typeDef.Name.GetPosition()) {
		return
	}
	symRef, _, _ := ms.GetSymbol(name)
	if typeDef.IsDistinct() {
		switch carrier := ms.ctx.GetSymbol(symRef).(type) {
		case *model.ErrorTypeSymbol:
			carrier.SetDistinctTypeIDs([]int{ms.ctx.DistinctTypeID(symRef)})
			registerLangLibDistinctTypeSymbol(ms, typeDef.Name.Value, symRef, typeDef.GetPosition())
		case model.ObjectType:
			carrier.SetDistinctTypeIDs([]int{ms.ctx.DistinctTypeID(symRef)})
			registerLangLibDistinctTypeSymbol(ms, typeDef.Name.Value, symRef, typeDef.GetPosition())
		default:
			ms.ctx.Unimplemented("distinct types are only supported for object and error types", typeDef.GetPosition())
		}
	}
	ms.typeDefns[symRef] = typeDef
}

func (ms *moduleSymbolResolver) ensureTypeAllocated(identifer ast.Reference, seen map[string]struct{}) {
	if identifer.ModulePrefix() != "" {
		// Imported symbol should have been resolved already
		return
	}
	name := identifer.Name()
	if _, ok := ms.packageSymbols[name]; ok {
		return
	}

	if _, ok := seen[name]; ok {
		return
	}
	seen[name] = struct{}{}
	td, ok := ms.moduleNodes.typeDefns[name]
	if ok {
		td.resolver.allocateTypeSymbol(td.node, seen)
		return
	}
	classDef, ok := ms.moduleNodes.classDefns[name]
	if !ok {
		// no such symbol to allocate.
		return
	}
	classDef.resolver.allocateClassSymbol(classDef.node)
}

func (ms *moduleSymbolResolver) allocateFunctionSymbol(fn *ast.BLangFunction) {
	name := fn.Name.Value
	symbol := ms.allocateFunctionSymbolInner(fn, name, fn.IsPublic())
	addTopLevelSymbol(ms, name, symbol, fn.Name.GetPosition())
}

func (ms *moduleSymbolResolver) allocateAnnotationSymbol(annotation *ast.BLangAnnotation) {
	name := annotation.Name.Value
	attachPoints := make([]string, 0, len(annotation.AttachPoints()))
	for _, attachPoint := range annotation.AttachPoints() {
		attachPoints = append(attachPoints, annotationAttachPointKey(attachPoint))
	}
	symbol := model.NewAnnotationSymbol(name, annotation.IsPublic(), annotation.IsConst(), attachPoints)
	addTopLevelAnnotationSymbol(ms, name, &symbol, annotation.Name.GetPosition())
}

func (ms *moduleSymbolResolver) allocateConstantSymbol(constDef *ast.BLangConstant) {
	name := constDef.Name.Value
	isPublic := constDef.IsPublic()
	if !addTopLevelSymbol(ms, name, model.NewConstantValueSymbol(name, isPublic), constDef.Name.GetPosition()) {
		return
	}
	if !isPublic {
		symRef, _, _ := ms.GetSymbol(name)
		markInit(ms, name, symRef, constDef.GetPosition())
	}
}

func (ms *moduleSymbolResolver) allocateGlobalVarSymbol(globalVar *ast.BLangSimpleVariable) {
	name := globalVar.Name.Value
	isPublic := globalVar.IsPublic()
	{
		symbol := model.NewVariableSymbol(name, isPublic, false, false)
		if globalVar.IsFinal() {
			symbol.SetFinal()
		}
		if globalVar.IsConfigurable() {
			symbol.SetConfigurable()
		}
		if globalVar.Flags().Has(model.FlagIsolated) {
			symbol.SetIsolated()
		}
		if globalVar.IsListener() {
			symbol.SetListener()
		}
		if !addTopLevelSymbol(ms, name, &symbol, globalVar.Name.GetPosition()) {
			return
		}
	}
	if !isPublic {
		symRef, _, _ := ms.GetSymbol(name)
		markInit(ms, name, symRef, globalVar.GetPosition())
	}
}

func (ms *moduleSymbolResolver) allocateClassSymbol(classDef *ast.BLangClassDefinition) {
	name := classDef.Name.Value
	if ref, ok := ms.packageSymbols[name]; ok {
		if existing, ok := ms.classDefns[ref]; ok && existing == classDef {
			return
		}
	}
	symbol := newClassSymbolForDefn(classDef)
	if !addTopLevelSymbol(ms, name, symbol, classDef.Name.GetPosition()) {
		return
	}
	symRef, _, _ := ms.GetSymbol(name)
	if classDef.IsDistinct() {
		symbol.SetDistinctTypeIDs([]int{ms.ctx.DistinctTypeID(symRef)})
		registerLangLibDistinctTypeSymbol(ms, classDef.Name.Value, symRef, classDef.GetPosition())
	}
	ms.classDefns[symRef] = classDef
}

func registerLangLibDistinctTypeSymbol(ms *moduleSymbolResolver, typeName string, ref model.SymbolRef, pos diagnostics.Location) {
	if ms.pkgID.OrgName == nil || ms.pkgID.PkgName == nil ||
		ms.pkgID.OrgName.Value() != "ballerina" || !strings.HasPrefix(ms.pkgID.PkgName.Value(), "lang.") {
		return
	}
	if !ms.ctx.RegisterLangLibDistinctTypeSymbol(ms.pkgID.PkgName.Value(), typeName, ref) {
		ms.ctx.InternalError("failed to register lang library distinct type symbol: "+ms.pkgID.PkgName.Value()+":"+typeName, pos)
	}
}

func compilationUnitImports(cu *ast.BLangCompilationUnit) []ast.BLangImportPackage {
	imports := make([]ast.BLangImportPackage, 0)
	for _, node := range cu.TopLevelNodes {
		imp, ok := node.(*ast.BLangImportPackage)
		if ok {
			imports = append(imports, *imp)
		}
	}
	return imports
}

func reportUnusedVariables(ctx *context.CompilerContext, unused []varDeclInfo) {
	for _, v := range unused {
		name := ctx.SymbolName(v.varSym)
		ctx.SemanticError("unused variable '"+name+"'", v.pos)
	}
}

func reportUnusedImports(resolver *moduleSymbolResolver, imports []ast.BLangImportPackage) {
	for i := range imports {
		imp := &imports[i]
		alias := imp.Alias.Value
		if alias == string(model.IGNORE) {
			continue
		}
		if !resolver.usedPrefixes[alias] {
			resolver.ctx.SemanticError("unused import prefix '"+alias+"'", imp.GetPosition())
		}
	}
}

func newClassSymbolForDefn(classDef *ast.BLangClassDefinition) model.ClassSymbol {
	name := classDef.Name.Value
	isPublic := classDef.IsPublic()
	if classDef.IsClient() || classDef.IsService() {
		return model.NewNetworkClassSymbol(name, isPublic)
	}
	return model.NewClassSymbol(name, isPublic)
}

// injectOpaqueSymbols adds the Go-defined symbols of a builtin lang library to
// the package's symbol table before its AST symbols are resolved. It is a no-op
// for non-builtin packages, where model.OpaqueSymbols returns nil.
func injectOpaqueSymbols(pkgID model.PackageID, r *moduleSymbolResolver) {
	pkg := model.PackageIdentifier{
		Organization: pkgID.OrgName.Value(),
		Package:      pkgID.PkgName.Value(),
		Version:      pkgID.Version.Value(),
	}
	space := r.scope.MainSpace()
	for _, sym := range model.OpaqueSymbols(pkg) {
		fillinOpaqueSymbol(sym, space)
		r.AddSymbol(sym.Name(), sym)
	}
}

// fillinOpaqueSymbol fills in any information that needs to be stored in the opaque symbol
// that is used within semantic package.
func fillinOpaqueSymbol(sym model.Symbol, space *model.SymbolSpace) {
	fn, ok := sym.(*model.OpaqueFunctionSymbol)
	if !ok {
		return
	}
	fn.SymbolSpace = space
	fn.Lookup, fn.Store = newMonomorphizationCache()
}

// TODO: when we have cases where we have more than one type param types we need to properly
// implement this
func newMonomorphizationCache() (func(...semtypes.SemType) (model.SymbolRef, bool), func(model.SymbolRef, ...semtypes.SemType)) {
	var mu sync.Mutex
	interner := semtypes.NewSemtypeInterner()
	cache := make(map[semtypes.InternHandle]model.SymbolRef)
	keyOf := func(keys []semtypes.SemType) semtypes.InternHandle {
		if len(keys) != 1 {
			panic("monomorphization cache supports a single key type")
		}
		return interner.Intern(keys[0])
	}
	lookup := func(keys ...semtypes.SemType) (model.SymbolRef, bool) {
		mu.Lock()
		defer mu.Unlock()
		ref, ok := cache[keyOf(keys)]
		return ref, ok
	}
	store := func(ref model.SymbolRef, keys ...semtypes.SemType) {
		mu.Lock()
		defer mu.Unlock()
		cache[keyOf(keys)] = ref
	}
	return lookup, store
}

func resolveFunction(functionResolver *blockSymbolResolver, function *ast.BLangFunction) {
	resolveFunctionInner(functionResolver, function.RequiredParams, function.RestParam, function, function.Body)
}

func resolveFunctionInner(functionResolver *blockSymbolResolver, requiredParams []ast.BLangSimpleVariable, restParam ast.SimpleVariableNode, walkNode ast.BLangNode, body ast.FunctionBodyNode) {
	trackParams := !isExternalFunctionBody(body)
	scope := functionResolver.scope.MainSpace()
	for i := range requiredParams {
		param := &requiredParams[i]
		name := param.Name.Value
		if _, exists := scope.GetSymbol(name); exists {
			semanticError(functionResolver, "redeclared symbol '"+name+"'", param.GetPosition())
			continue
		}
		symbol := model.NewVariableSymbol(name, false, false, true)
		addSymbolAndSetOnNode(functionResolver, name, &symbol, param)
		if trackParams {
			markInit(functionResolver, name, param.Symbol(), param.GetPosition())
		}
	}
	if restParam != nil {
		rest := restParam.(*ast.BLangSimpleVariable)
		name := rest.Name.Value
		if _, exists := scope.GetSymbol(name); exists {
			semanticError(functionResolver, "redeclared symbol '"+name+"'", rest.GetPosition())
		} else {
			symbol := model.NewVariableSymbol(name, false, false, true)
			addSymbolAndSetOnNode(functionResolver, name, &symbol, rest)
			if trackParams {
				markInit(functionResolver, name, rest.Symbol(), rest.GetPosition())
			}
		}
	}
	ast.Walk(functionResolver, walkNode)
	reportUnusedVariables(functionResolver.GetCtx(), functionResolver.getUnused())
}

func isExternalFunctionBody(body ast.FunctionBodyNode) bool {
	_, ok := body.(*ast.BLangExternFunctionBody)
	return ok
}

func allocateDefaultParamSymbols(alloc defaultSymbolAllocator, targetScope model.Scope, function *ast.BLangFunction) {
	if len(function.RequiredParams) == 0 {
		return
	}
	cx := alloc.GetCtx()
	fnSymRef := function.Symbol()
	fnSym := cx.GetSymbol(fnSymRef).(model.FunctionSymbol)
	info := model.NewDefaultableParamInfo(len(function.RequiredParams))
	var inclInfo *model.IncludedRecordParamInfo
	for i := range function.RequiredParams {
		param := &function.RequiredParams[i]
		if param.IsIncludedRecordParam() {
			if inclInfo == nil {
				inclInfo = model.NewIncludedRecordParamInfo(len(function.RequiredParams))
			}
			inclInfo.Set(i)
			continue
		}
		if !param.IsDefaultableParam() {
			continue
		}
		if _, ok := param.Expr.(*ast.BLangInferredTypedescDefault); ok {
			info.SetInferredTypedesc(i)
			continue
		}
		name := alloc.nextDefaultSymbolName()
		// Until type resolution we don't know the type of the parametes to create this function signature
		defaultFnSym := model.NewFunctionSymbol(name, model.FunctionSignature{}, false)
		targetScope.AddSymbol(name, defaultFnSym)
		symRef, _ := targetScope.GetSymbol(name)
		info.SetDefaultable(i, symRef)
	}
	fnSym.SetDefaultableParams(info)
	fnSym.SetIncludedRecordParams(inclInfo)
}

func resolveLambdaFunction(functionResolver *blockSymbolResolver, parent *blockSymbolResolver, function *ast.BLangFunction) {
	// Check for shadowing on parameters against the enclosing function scope
	for i := range function.RequiredParams {
		param := &function.RequiredParams[i]
		name := param.Name.Value
		if isShadowed(parent, name) {
			semanticError(functionResolver, "Variable already defined: "+name, param.GetPosition())
		}
		symbol := model.NewVariableSymbol(name, false, false, true)
		addSymbolAndSetOnNode(functionResolver, name, &symbol, param)
		markInit(functionResolver, name, param.Symbol(), param.GetPosition())
	}

	if function.RestParam != nil {
		restParam := function.RestParam.(*ast.BLangSimpleVariable)
		name := restParam.Name.Value
		if isShadowed(parent, name) {
			semanticError(functionResolver, "Variable already defined: "+name, restParam.GetPosition())
		}
		symbol := model.NewVariableSymbol(name, false, false, true)
		addSymbolAndSetOnNode(functionResolver, name, &symbol, restParam)
		markInit(functionResolver, name, restParam.Symbol(), restParam.GetPosition())
	}

	ast.Walk(functionResolver, function)
	reportUnusedVariables(functionResolver.GetCtx(), functionResolver.getUnused())
}

func ResolveCompilationUnitImports(ctx *context.CompilerContext, compilationUnits []*ast.BLangCompilationUnit,
	implicitImports map[string]model.ExportedSymbolSpace, publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace, defaultOrg string,
) []CompilationUnitImports {
	result := make([]CompilationUnitImports, len(compilationUnits))
	for i, cu := range compilationUnits {
		imports := make(map[string]model.ExportedSymbolSpace)
		for _, imp := range compilationUnitImports(cu) {
			resolveExternalImport(ctx, &imp, defaultOrg, publicSymbols, imports)
		}
		maps.Copy(imports, implicitImports)
		result[i] = CompilationUnitImports{CompilationUnit: cu, Imports: imports}
	}
	return result
}

// bindIntrinsicImport binds a compiler-intrinsic symbol space under either the
// import's alias or the given default name.
func bindIntrinsicImport(
	imp *ast.BLangImportPackage,
	defaultName string,
	symbols model.ExportedSymbolSpace,
	result map[string]model.ExportedSymbolSpace,
) {
	key := defaultName
	if imp.Alias != nil {
		key = imp.Alias.Value
	}
	result[key] = symbols
}

// resolveExternalImport looks up the import's exported symbols in publicSymbols
// (populated as each dependency's module is compiled) and binds them to the
// import alias or the last name component. Reports an "Unknown import" error
// when the package was not resolved upstream.
func resolveExternalImport(
	ctx *context.CompilerContext,
	imp *ast.BLangImportPackage,
	defaultOrg string,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	result map[string]model.ExportedSymbolSpace,
) {
	id := resolveImportPackageIdentifier(imp, defaultOrg)
	symbols, ok := publicSymbols[id]
	if !ok {
		ctx.SemanticError("Unknown import: "+id.OrgName+"/"+id.ModuleName, imp.GetPosition())
		return
	}
	var key string
	if imp.Alias != nil {
		key = imp.Alias.Value
	} else {
		comps := imp.GetPackageName()
		key = comps[len(comps)-1].GetValue()
	}
	result[key] = symbols
}

type PackageIdentifier struct {
	OrgName    string
	ModuleName string
}

func resolveImportPackageIdentifier(imp *ast.BLangImportPackage, defaultOrg string) PackageIdentifier {
	nameComps := imp.GetPackageName()
	nameParts := make([]string, len(nameComps))
	for i, name := range nameComps {
		nameParts[i] = name.GetValue()
	}
	moduleName := strings.Join(nameParts, ".")
	var orgName string
	if imp.OrgName == nil || imp.OrgName.GetValue() == "" {
		orgName = defaultOrg
	} else {
		orgName = imp.OrgName.GetValue()
	}
	return PackageIdentifier{orgName, moduleName}
}

func GetImplicitImports(ctx *context.CompilerContext) map[string]model.ExportedSymbolSpace {
	result := make(map[string]model.ExportedSymbolSpace)
	result[langinternal.PackageName] = langinternal.GetInternalSymbols(ctx)
	return result
}

func (bs *blockSymbolResolver) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangXMLNS:
		processBlockXMLNS(bs, n)
		return nil
	case *ast.BLangFunction:
		// This happens because we visit from the top in [resolveFunction]
		if n == bs.node {
			return bs
		}
		functionResolver := newFunctionResolver(bs, n)
		n.SetScope(functionResolver.scope)
		resolveFunction(functionResolver, n)
		return nil
	case *ast.BLangIf:
		resolver := newBlockSymbolResolverWithBlockScope(bs, n)
		n.SetScope(resolver.scope)
		return resolver
	case *ast.BLangWhile:
		resolver := newBlockSymbolResolverWithBlockScope(bs, n)
		n.SetScope(resolver.scope)
		return resolver
	case *ast.BLangForeach:
		resolveForeachSymbols(bs, n)
		return nil
	case *ast.BLangBlockStmt, *ast.BLangDo, *ast.BLangLock:
		return newBlockSymbolResolverWithBlockScope(bs, n)
	case *ast.BLangSimpleVariableDef:
		defineVariable(bs, n.GetVariable(), n.GetVariable().(*ast.BLangSimpleVariable).IsFinal())
	case *ast.BLangLambdaFunction:
		fn := n.Function
		name := fn.Name.Value
		signature := model.FunctionSignature{}
		symbol := model.NewFunctionSymbol(name, signature, false)
		addSymbolAndSetOnNode(bs, name, symbol, fn)
		functionResolver := newFunctionResolver(bs, fn)
		fn.SetScope(functionResolver.scope)
		resolveLambdaFunction(functionResolver, bs, fn)
		return nil
	default:
		return visitInnerSymbolResolver(bs, n)
	}
	return bs
}

func visitInnerSymbolResolver[T symbolResolver](resolver T, node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangXMLElementLiteral:
		rootNeeds := map[string]model.SymbolRef{}
		resolveXMLElementLiteralNamespaces(resolver, resolver.GetScope(), n, rootNeeds)
		mergeNamespaces(resolver, n, rootNeeds)
		return nil
	case *ast.BLangXMLTemplateExpr:
		resolveXMLTemplateNamespaces(resolver, resolver.GetScope(), n)
		for _, ins := range n.Insertions {
			ast.Walk(resolver, ins)
		}
		return nil
	case *ast.BLangXMLSequenceLiteral:
		for _, child := range n.Children {
			ast.Walk(resolver, child)
		}
		return nil
	case *ast.BLangFieldBaseAccess:
		if isSelfFieldAccess(n) {
			if classScope, ok := getEnclosingClassBodyScope(resolver); ok {
				resolveSelfFieldAccess(resolver, n, classScope)
				return nil
			}
		}
	case *ast.BLangMappingConstructorExpr:
		return resolveMappingConstructor(resolver, n)
	case *ast.BLangAnnotationAttachment:
		resolveAnnotationReference(resolver, n.GetPackageAlias(), n.GetAnnotationName(), n.GetPosition(), n)
	case *ast.BLangAnnotAccessExpr:
		resolveAnnotationReference(resolver, n.PkgAlias, n.AnnotationName, n.GetPosition(), n)
	case *ast.BLangQueryExpr:
		return newBlockSymbolResolverWithBlockScope(resolver, n)
	case *ast.BLangInvocation:
		if n.GetExpression() != nil {
			createDeferredMethodSymbol(resolver, n)
		} else {
			resolveFunctionRef(resolver, n)
		}
	case *ast.BLangRemoteMethodCallAction:
		// We are creating a deferred symbol here since without determining the type of the reciever we can't determine the actual function symbol
		createDeferredMethodSymbol(resolver, n)
	case ast.VariableNode:
		referVariable(resolver, n.(variableNode))
	case ast.SimpleVariableReferenceNode:
		referSimpleVariableReference(resolver, n)
	case *ast.BLangUserDefinedType:
		referUserDefinedType(resolver, n)
	case *ast.BLangObjectType:
		n.Inclusions, n.InclusionPositions, _ = resolveObjectInclusions(resolver, n.PopUnresolvedInclusions())
	case *ast.BLangRecordType:
		n.Inclusions = resolveRecordTypeInclusions(resolver, n.TypeInclusions)
	}
	return resolver
}

func resolveMappingConstructor[T symbolResolver](resolver T, n *ast.BLangMappingConstructorExpr) ast.Visitor {
	return newBlockSymbolResolverWithBlockScope(resolver, n)
}

// since we don't have type information we can't determine if this is an actual method call or need to be converted
// to a function call.
func createDeferredMethodSymbol[T symbolResolver](resolver T, n invocable) {
	name := n.GetName().GetValue()
	scope := resolver.GetScope().(model.SymbolSpaceProvider)
	n.SetRawSymbol(new(deferredMethodSymbol{name: name, space: scope.MainSpace()}))
}

type deferredMethodSymbol struct {
	name  string
	space *model.SymbolSpace
}

var _ model.Symbol = &deferredMethodSymbol{}

// IsDeferredMethodSymbol returns true if the symbol is a deferred method symbol
// (a placeholder used during symbol resolution that will be resolved later).
func IsDeferredMethodSymbol(symbol any) bool {
	_, ok := symbol.(*deferredMethodSymbol)
	return ok
}

func (d *deferredMethodSymbol) Name() string {
	panic("method symbol has not been resolved yet")
}

func (d *deferredMethodSymbol) Type() semtypes.SemType {
	panic("method symbol has not been resolved yet")
}

func (d *deferredMethodSymbol) Kind() model.SymbolKind {
	panic("method symbol has not been resolved yet")
}

func (d *deferredMethodSymbol) SetType(semtypes.SemType) {
	panic("method symbol has not been resolved yet")
}

func (d *deferredMethodSymbol) IsPublic() bool {
	panic("method symbol has not been resolved yet")
}

func (d *deferredMethodSymbol) Copy() model.Symbol {
	panic("method symbol has not been resolved yet")
}

func referUserDefinedType[T symbolResolver](resolver T, n *ast.BLangUserDefinedType) {
	name := n.GetTypeName().GetValue()
	var prefix string
	if n.GetPackageAlias() != nil {
		prefix = n.GetPackageAlias().GetValue()
	}
	resolveSymbolRef(resolver, name, prefix, n.GetPosition(), n, "Unknown type")
	markUnprefixedRefUsed(resolver, name, prefix)
}

func markUnprefixedRefUsed[T symbolResolver](resolver T, name, prefix string) {
	if prefix != "" {
		return
	}
	symRef, _, ok := resolver.GetSymbol(name)
	if !ok {
		return
	}
	resolver.markUsed(symRef)
}

type symbolRefNode interface {
	SetSymbol(symbolRef model.SymbolRef)
}

func resolveSymbolRef[T symbolResolver](
	resolver T,
	name string,
	prefix string,
	pos diagnostics.Location,
	target symbolRefNode,
	unknownMessage string,
) {
	if prefix != "" {
		symRef, ok := resolver.GetPrefixedSymbol(prefix, name)
		if !ok {
			semanticError(resolver, "Unknown symbol: "+name, pos)
		}
		target.SetSymbol(symRef)
	} else {
		symRef, _, ok := resolver.GetSymbol(name)
		if !ok {
			semanticError(resolver, unknownMessage+": "+name, pos)
		}
		target.SetSymbol(symRef)
	}
}

func resolveAnnotationReference[T symbolResolver](resolver T, pkgAlias, name *ast.BLangIdentifier, pos diagnostics.Location, target symbolRefNode) {
	if name == nil {
		return
	}
	prefix := ""
	if pkgAlias != nil {
		prefix = pkgAlias.GetValue()
	}
	symRef, ok := resolver.GetAnnotationSymbol(prefix, name.GetValue())
	if !ok {
		semanticError(resolver, "Unknown annotation: "+name.GetValue(), pos)
		return
	}
	target.SetSymbol(symRef)
}

func referSimpleVariableReference[T symbolResolver](resolver T, n ast.SimpleVariableReferenceNode) {
	name := n.GetVariableName().GetValue()
	var prefix string
	if n.GetPackageAlias() != nil {
		prefix = n.GetPackageAlias().GetValue()
	}
	resolveSymbolRef(resolver, name, prefix, n.GetPosition(), n.(ast.BNodeWithSymbol), "Unknown symbol")
	markUnprefixedRefUsed(resolver, name, prefix)
}

type functionRefNode interface {
	GetName() *ast.BLangIdentifier
	GetPosition() diagnostics.Location
	GetPackageAlias() *ast.BLangIdentifier
	SetSymbol(symbolRef model.SymbolRef)
}

func resolveFunctionRef[T symbolResolver](resolver T, invocation *ast.BLangInvocation) {
	name := invocation.GetName().GetValue()
	prefix := invocation.GetPackageAlias().GetValue()
	resolveSymbolRef(resolver, name, prefix, invocation.GetPosition(), invocation, "Unknown symbol")
	markUnprefixedRefUsed(resolver, name, prefix)
}

type variableNode interface {
	GetName() *ast.BLangIdentifier
	GetPosition() diagnostics.Location
	SetSymbol(symbolRef model.SymbolRef)
}

func referVariable[T symbolResolver](resolver T, variable variableNode) {
	name := variable.GetName().GetValue()
	resolveSymbolRef(resolver, name, "", variable.GetPosition(), variable, "Unknown symbol")
}

// isShadowed checks if a name is already defined in an enclosing block scope.
// Mapping constructor scopes contain record keys that are not real variable bindings, so they are skipped.
func isShadowed(resolver *blockSymbolResolver, name string) bool {
	if name == string(model.IGNORE) {
		return false
	}
	current := resolver
	for current != nil {
		// Issue here is mapping constructor treats some of it's keys as simple variable ref; which is wrong but since they are variable they have symbols
		// and we have to resolve them. But they are not real variables
		if _, isMappingScope := current.node.(*ast.BLangMappingConstructorExpr); !isMappingScope {
			if _, ok := current.scope.MainSpace().GetSymbol(name); ok {
				return true
			}
		}
		if next, ok := current.parent.(*blockSymbolResolver); ok {
			current = next
		} else {
			break
		}
	}
	return false
}

func defineVariable(resolver *blockSymbolResolver, variable ast.VariableNode, isFinal bool) {
	switch variable := variable.(type) {
	case *ast.BLangSimpleVariable:
		name := variable.Name.Value
		if isShadowed(resolver, name) {
			semanticError(resolver, "Variable already defined: "+name, variable.GetPosition())
		}
		symbol := model.NewVariableSymbol(name, false, isFinal, false)
		if isFinal {
			symbol.SetFinal()
		}
		addSymbolAndSetOnNode(resolver, name, &symbol, variable)
		markInit(resolver, name, variable.Symbol(), variable.GetPosition())
	default:
		internalError(resolver, "Unsupported variable", variable.GetPosition())
		return
	}
}

func resolveForeachSymbols(bs *blockSymbolResolver, n *ast.BLangForeach) {
	resolver := newBlockSymbolResolverWithBlockScope(bs, n)
	n.SetScope(resolver.scope)
	if n.Collection != nil {
		ast.Walk(resolver, n.Collection.(ast.BLangNode))
	}
	if n.VariableDef != nil {
		defineVariable(resolver, n.VariableDef.GetVariable(), true)
		ast.Walk(resolver, n.VariableDef.Var)
	}
	ast.Walk(resolver, &n.Body)
	if n.OnFailClause != nil {
		ast.Walk(resolver, n.OnFailClause)
	}
}

func (bs *blockSymbolResolver) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	if typeData.TypeDescriptor == nil {
		return nil
	}
	td := typeData.TypeDescriptor
	setTypeDescriptorSymbol(bs, td)
	return bs
}

func setTypeDescriptorSymbol[T symbolResolver](resolver T, td ast.TypeDescriptor) {
	if bNodeWithSymbol, ok := td.(ast.BNodeWithSymbol); ok {
		if ast.SymbolIsSet(bNodeWithSymbol) {
			return
		}
		switch td := td.(type) {
		case *ast.BLangUserDefinedType:
			pkg := td.GetPackageAlias().GetValue()
			tyName := td.GetTypeName().GetValue()
			var symRef model.SymbolRef
			if pkg != "" {
				symRef, ok = resolver.GetPrefixedSymbol(pkg, tyName)
				if !ok {
					semanticError(resolver, "Unknown type: "+tyName, td.GetPosition())
				}
			} else {
				symRef, _, ok = resolver.GetSymbol(tyName)
				if !ok {
					semanticError(resolver, "Unknown type: "+tyName, td.GetPosition())
				}
				markUnprefixedRefUsed(resolver, tyName, pkg)
			}
			bNodeWithSymbol.SetSymbol(symRef)
		default:
			internalError(resolver, "Unsupported type descriptor", td.GetPosition())
		}
	}
}

func (ms *moduleSymbolResolver) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangFunction:
		name := n.Name.Value
		symRef, _, ok := ms.GetSymbol(name)
		if !ok {
			internalError(ms, "Module level function symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		functionResolver := newFunctionResolver(ms, n)
		n.SetScope(functionResolver.scope)
		resolveFunction(functionResolver, n)
		allocateDefaultParamSymbols(ms, ms.scope, n)
		return nil
	case *ast.BLangConstant:
		name := n.Name.Value
		symRef, _, ok := ms.GetSymbol(name)
		if !ok {
			internalError(ms, "Module level constant symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		// TODO: create a local scope and resolve the body?
		return ms
	case *ast.BLangSimpleVariable:
		name := n.Name.Value
		symRef, _, ok := ms.GetSymbol(name)
		if !ok {
			internalError(ms, "Module level variable symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		return ms
	case *ast.BLangTypeDefinition:
		name := n.Name.Value
		symRef, _, ok := ms.GetSymbol(name)
		if !ok {
			internalError(ms, "Module level type symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		return ms
	case *ast.BLangAnnotation:
		name := n.Name.Value
		symRef, ok := ms.GetAnnotationSymbol("", name)
		if !ok {
			internalError(ms, "Module level annotation symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		return ms
	case *ast.BLangClassDefinition:
		name := n.Name.Value
		symRef, _, ok := ms.GetSymbol(name)
		if !ok {
			internalError(ms, "Module level class symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		resolveClassDefinition(ms, n)
		return nil
	case *ast.BLangService:
		resolveServiceDefinition(ms, n)
		return nil
	case *ast.BLangLambdaFunction:
		fn := n.Function
		name := fn.Name.Value
		signature := model.FunctionSignature{}
		symbol := model.NewFunctionSymbol(name, signature, false)
		ms.AddSymbol(name, symbol)
		symRef, _, _ := ms.GetSymbolFromCurrentScope(name)
		fn.SetSymbol(symRef)
		functionResolver := newFunctionResolver(ms, fn)
		fn.SetScope(functionResolver.scope)
		resolveLambdaFunction(functionResolver, functionResolver, fn)
		return nil
	default:
		return visitInnerSymbolResolver(ms, n)
	}
}

func (ms *moduleSymbolResolver) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return ms
}

type inclusionMemberForSymbolResolution struct {
	name     string
	isPublic bool
}

// resolveObjectInclusions update the AST node references with correct symbol references. Will add semantic errors if the type
// reference is for something that can't be included. This means after this stage we have the gurantee symbol ref always refer
// to a valid AST node.
func resolveObjectInclusions[T symbolResolver](resolver T, unresolvedInclusions []*ast.BLangUserDefinedType) ([]model.SymbolRef, []diagnostics.Location, []inclusionMemberForSymbolResolution) {
	ctx := resolver.GetCtx()
	localTypeDefns := resolver.GetTypeDefns()
	localClassDefns := resolver.GetClassDefns()
	inclusions := make([]model.SymbolRef, 0, len(unresolvedInclusions))
	positions := make([]diagnostics.Location, 0, len(unresolvedInclusions))
	var includedFields []inclusionMemberForSymbolResolution
	for _, inc := range unresolvedInclusions {
		ast.Walk(resolver, inc)
		symRef := inc.Symbol()
		if tDefn, ok := localTypeDefns[symRef]; ok {
			if _, ok := tDefn.GetTypeData().TypeDescriptor.(*ast.BLangObjectType); !ok {
				if _, ok := ctx.GetSymbol(symRef).(*model.ObjectTypeSymbol); !ok {
					ctx.SemanticError("type inclusion must be an object type or class", inc.GetPosition())
					continue
				}
			}
			includedFields = append(includedFields, collectTransitiveFieldsFromTypeDefn(ctx, tDefn, localTypeDefns, localClassDefns)...)
		} else if classDefn, ok := localClassDefns[symRef]; ok {
			includedFields = append(includedFields, collectTransitiveFieldsFromClassDefn(ctx, classDefn, localTypeDefns, localClassDefns)...)
		} else {
			sym := ctx.GetSymbol(symRef)
			var carrier model.MemberCarrier
			switch s := sym.(type) {
			case model.ClassSymbol:
				carrier = s
			case *model.ObjectTypeSymbol:
				incTy := ctx.SymbolType(symRef)
				if semtypes.IsZero(incTy) || !semtypes.IsSubtype(resolver.TypeContext(), incTy, semtypes.OBJECT) {
					ctx.SemanticError("type inclusion must be an object type or class", inc.GetPosition())
					continue
				}
				carrier = s
			default:
				ctx.SemanticError("type inclusion must be an object type or class", inc.GetPosition())
				continue
			}
			for _, m := range carrier.Members() {
				if m.MemberKind() != model.InclusionMemberKindField {
					continue
				}
				fd := m.(*model.FieldDescriptor)
				includedFields = append(includedFields, inclusionMemberForSymbolResolution{
					name:     fd.MemberName(),
					isPublic: fd.IsPublic(),
				})
			}
		}
		inclusions = append(inclusions, symRef)
		positions = append(positions, inc.GetPosition())
	}
	return inclusions, positions, includedFields
}

func resolveRecordTypeInclusions[T symbolResolver](resolver T, typeInclusions []ast.BType) []model.SymbolRef {
	ctx := resolver.GetCtx()
	localTypeDefns := resolver.GetTypeDefns()
	var inclusions []model.SymbolRef
	for _, inc := range typeInclusions {
		udt, ok := inc.(*ast.BLangUserDefinedType)
		if !ok {
			ctx.SemanticError("type inclusion must be a user-defined type", inc.(ast.BLangNode).GetPosition())
			continue
		}
		ast.Walk(resolver, udt)
		symRef := udt.Symbol()
		if tDefn, ok := localTypeDefns[symRef]; ok {
			if _, ok := tDefn.GetTypeData().TypeDescriptor.(*ast.BLangRecordType); !ok {
				ctx.SemanticError("included type is not a record type", udt.GetPosition())
				continue
			}
		} else {
			sym := ctx.GetSymbol(symRef)
			if _, ok := sym.(*model.RecordSymbol); !ok {
				ctx.SemanticError("included type is not a record type", udt.GetPosition())
				continue
			}
			incTy := ctx.SymbolType(symRef)
			if semtypes.IsZero(incTy) || !semtypes.IsSubtype(resolver.TypeContext(), incTy, semtypes.MAPPING) {
				ctx.SemanticError("included type is not a record type", udt.GetPosition())
				continue
			}
		}
		inclusions = append(inclusions, symRef)
	}
	return inclusions
}

func collectTransitiveFields(ctx *context.CompilerContext, inclusions []model.SymbolRef, directFields []inclusionMemberForSymbolResolution, localTypeDefns map[model.SymbolRef]*ast.BLangTypeDefinition, localClassDefns map[model.SymbolRef]*ast.BLangClassDefinition) []inclusionMemberForSymbolResolution {
	var result []inclusionMemberForSymbolResolution
	for _, symRef := range inclusions {
		if tDefn, ok := localTypeDefns[symRef]; ok {
			result = append(result, collectTransitiveFieldsFromTypeDefn(ctx, tDefn, localTypeDefns, localClassDefns)...)
		} else if classDefn, ok := localClassDefns[symRef]; ok {
			result = append(result, collectTransitiveFieldsFromClassDefn(ctx, classDefn, localTypeDefns, localClassDefns)...)
		} else {
			sym := ctx.GetSymbol(symRef)
			var carrier model.MemberCarrier
			switch s := sym.(type) {
			case *model.RecordSymbol:
				carrier = s
			case *model.ObjectTypeSymbol:
				carrier = s
			case model.ClassSymbol:
				carrier = s
			default:
				continue
			}
			for _, m := range carrier.Members() {
				if m.MemberKind() != model.InclusionMemberKindField {
					continue
				}
				fd := m.(*model.FieldDescriptor)
				result = append(result, inclusionMemberForSymbolResolution{
					name:     fd.MemberName(),
					isPublic: fd.IsPublic(),
				})
			}
		}
	}
	result = append(result, directFields...)
	return result
}

func collectTransitiveFieldsFromTypeDefn(ctx *context.CompilerContext, defn *ast.BLangTypeDefinition, localTypeDefns map[model.SymbolRef]*ast.BLangTypeDefinition, localClassDefns map[model.SymbolRef]*ast.BLangClassDefinition) []inclusionMemberForSymbolResolution {
	objTy, ok := defn.GetTypeData().TypeDescriptor.(*ast.BLangObjectType)
	if !ok {
		return nil
	}
	var directFields []inclusionMemberForSymbolResolution
	for m := range objTy.Members() {
		if m.MemberKind() != ast.ObjectMemberKindField {
			continue
		}
		directFields = append(directFields, inclusionMemberForSymbolResolution{
			name:     m.Name(),
			isPublic: m.IsPublic(),
		})
	}
	return collectTransitiveFields(ctx, objTy.Inclusions, directFields, localTypeDefns, localClassDefns)
}

func collectTransitiveFieldsFromClassDefn(ctx *context.CompilerContext, defn *ast.BLangClassDefinition, localTypeDefns map[model.SymbolRef]*ast.BLangTypeDefinition, localClassDefns map[model.SymbolRef]*ast.BLangClassDefinition) []inclusionMemberForSymbolResolution {
	var directFields []inclusionMemberForSymbolResolution
	for _, fieldNode := range defn.Fields {
		field := fieldNode.(*ast.BLangSimpleVariable)
		directFields = append(directFields, inclusionMemberForSymbolResolution{
			name:     field.Name.Value,
			isPublic: field.IsPublic(),
		})
	}
	return collectTransitiveFields(ctx, defn.Inclusions, directFields, localTypeDefns, localClassDefns)
}

type namedClassMethod struct {
	name   string
	method *ast.BLangFunction
}

func methodsInResolutionOrder(methods map[string]*ast.BLangFunction) []namedClassMethod {
	names := slices.Sorted(maps.Keys(methods))
	result := make([]namedClassMethod, len(names))
	for i, name := range names {
		result[i] = namedClassMethod{name: name, method: methods[name]}
	}
	return result
}

func resolveServiceDefinition(ms *moduleSymbolResolver, svc *ast.BLangService) {
	if typeDescriptor := svc.GetTypeData().TypeDescriptor; typeDescriptor != nil {
		ast.Walk(ms, typeDescriptor.(ast.BLangNode))
	}

	for _, expr := range svc.AttachedExprs {
		ast.Walk(ms, expr.(ast.BLangNode))
	}

	if svc.InitFunction != nil && len(svc.InitFunction.RequiredParams) > 0 {
		semanticError(ms, "service 'init' must not declare required parameters", svc.InitFunction.RequiredParams[0].GetPosition())
	}

	serviceMethodSymbolName := func(methodName string) string {
		return methodName
	}
	resourceMethodsAreNetworkClass := true

	svcResolver := newBlockSymbolResolverWithBlockScope(ms, svc)
	svc.SetScope(svcResolver.scope)

	allocateServiceResourceMethodSymbols(svcResolver, svc.ResourceMethods)

	finishResolveClassDefinition(ms, svcResolver, svc.Fields, svc.Methods, svc.ResourceMethods, svc.InitFunction, nil, svcResolver.scope, serviceMethodSymbolName, resourceMethodsAreNetworkClass)
}

func resolveClassDefinition(ms *moduleSymbolResolver, classDef *ast.BLangClassDefinition) {
	className := classDef.Name.Value
	classMethodSymbolName := func(methodName string) string {
		return className + "." + methodName
	}
	classSym := ms.ctx.GetSymbol(classDef.Symbol()).(model.ClassSymbol)
	networkClassSym, isNetworkClass := ms.ctx.GetSymbol(classDef.Symbol()).(*model.NetworkClassSymbol)

	classResolver := newBlockSymbolResolverWithBlockScope(ms, classDef)
	classDef.SetScope(classResolver.scope)
	for i := range classDef.AnnAttachments {
		ast.Walk(classResolver, &classDef.AnnAttachments[i])
	}

	var includedFields []inclusionMemberForSymbolResolution
	classDef.Inclusions, classDef.InclusionPositions, includedFields = resolveObjectInclusions(ms, classDef.PopUnresolvedInclusions())
	allocateObjectResourceMethodSymbols(ms, classResolver, classDef, networkClassSym, isNetworkClass)

	finishResolveClassDefinition(ms, classResolver, classDef.Fields, classDef.Methods, classDef.ResourceMethods, classDef.InitFunction, includedFields, ms.scope, classMethodSymbolName, isNetworkClass)

	publishObjectMethodTable(classSym, classDef)
}

func finishResolveClassDefinition(ms *moduleSymbolResolver, blockRes *blockSymbolResolver, fields []ast.SimpleVariableNode, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod, initFn *ast.BLangFunction, includedFields []inclusionMemberForSymbolResolution, methodTargetScope methodSymbolTargetScope, methodSymbolName func(string) string, resourceMethodsAreNetworkClass bool) {
	for _, field := range fields {
		name := field.GetName().GetValue()
		if _, sk, exists := blockRes.GetSymbol(name); exists && sk == blockScopeKind {
			semanticError(blockRes, "redeclared symbol '"+name+"'", field.GetPosition())
			continue
		}
		isPublic := field.IsPublic()
		symbol := model.NewVariableSymbol(name, isPublic, false, false)
		blockRes.AddSymbol(name, &symbol)
	}

	orderedMethods := methodsInResolutionOrder(methods)
	for _, m := range orderedMethods {
		if _, sk, exists := blockRes.GetSymbol(m.name); exists && sk == blockScopeKind {
			semanticError(blockRes, "redeclared symbol '"+model.StripRemotePrefix(m.name)+"'", m.method.Name.GetPosition())
			continue
		}
		isPublic := m.method.IsPublic()
		symbol := ms.allocateFunctionSymbolInner(m.method, m.name, isPublic)
		symbolName := methodSymbolName(m.name)
		methodTargetScope.AddSymbol(symbolName, symbol)
		symRef, _ := methodTargetScope.MainSpace().GetSymbol(symbolName)
		m.method.SetSymbol(symRef)
	}

	for _, m := range includedFields {
		if _, _, exists := blockRes.GetSymbol(m.name); exists {
			continue
		}
		symbol := model.NewVariableSymbol(m.name, m.isPublic, false, false)
		blockRes.AddSymbol(m.name, &symbol)
	}

	if initFn != nil {
		signature := model.FunctionSignature{}
		symbol := model.NewFunctionSymbol("init", signature, false)
		addSymbolAndSetOnNode(blockRes, "init", symbol, initFn)
	}

	selfSymbol := model.NewVariableSymbol("self", false, false, false)
	blockRes.AddSymbol("self", &selfSymbol)

	for _, field := range fields {
		ast.Walk(blockRes, field.(ast.BLangNode))
	}

	if initFn != nil {
		initResolver := newFunctionResolver(blockRes, initFn)
		initFn.SetScope(initResolver.scope)
		resolveFunction(initResolver, initFn)
		allocateDefaultParamSymbols(ms, ms.scope, initFn)
	}

	for _, m := range orderedMethods {
		methodResolver := newFunctionResolver(blockRes, m.method)
		m.method.SetScope(methodResolver.scope)
		resolveFunction(methodResolver, m.method)
		allocateDefaultParamSymbols(ms, ms.scope, m.method)
	}

	for _, rm := range resourceMethods {
		if !resourceMethodsAreNetworkClass {
			continue
		}
		methodResolver := newFunctionResolver(blockRes, rm)
		rm.SetScope(methodResolver.scope)
		resolveResourceMethod(methodResolver, rm)
	}
}

type methodSymbolTargetScope interface {
	model.Scope
	MainSpace() *model.SymbolSpace
}

func allocateObjectResourceMethodSymbols(ms *moduleSymbolResolver, blockRes *blockSymbolResolver, classDef *ast.BLangClassDefinition, networkClassSym *model.NetworkClassSymbol, isNetworkClass bool) {
	className := classDef.Name.Value
	for idx, rm := range classDef.ResourceMethods {
		if !isNetworkClass {
			semanticError(blockRes, "resource methods are only allowed in client or service classes", rm.GetPosition())
			continue
		}
		mangledName := className + "." + mangledResourceMethodName(rm.Name.Value, idx)
		symRef := allocateResourceMethodSymbol(ms.scope, rm, mangledName, classDef.IsPublic() && rm.IsPublic())
		networkClassSym.AddResourceMethod(symRef)
	}
}

func allocateServiceResourceMethodSymbols(blockRes *blockSymbolResolver, resourceMethods []*ast.BLangResourceMethod) {
	for idx, rm := range resourceMethods {
		key := mangledResourceMethodName(rm.Name.Value, idx)
		allocateResourceMethodSymbol(blockRes.scope, rm, key, rm.IsPublic())
	}
}

func allocateResourceMethodSymbol(targetScope methodSymbolTargetScope, rm *ast.BLangResourceMethod, symbolName string, isPublic bool) model.SymbolRef {
	symbol := model.NewResourceMethodSymbol(symbolName, rm.Name.Value, isPublic)
	targetScope.AddSymbol(symbolName, symbol)
	symRef, _ := targetScope.MainSpace().GetSymbol(symbolName)
	rm.SetSymbol(symRef)
	return symRef
}

func publishObjectMethodTable(classSym model.ClassSymbol, classDef *ast.BLangClassDefinition) {
	methodTable := make(map[string]model.SymbolRef, len(classDef.Methods))
	for _, m := range methodsInResolutionOrder(classDef.Methods) {
		methodTable[m.name] = m.method.Symbol()
	}
	if classDef.InitFunction != nil {
		methodTable["init"] = classDef.InitFunction.Symbol()
	}
	classSym.SetMethods(methodTable)
}

func mangledResourceMethodName(methodName string, idx int) string {
	return fmt.Sprintf("$resource$%s$%d", methodName, idx)
}

func resolveResourceMethod(functionResolver *blockSymbolResolver, rm *ast.BLangResourceMethod) {
	// Limit collision detection to the current function scope, matching
	// resolveFunctionInner. functionResolver.GetSymbol would otherwise delegate
	// into the enclosing class scope (also a blockSymbolResolver) and wrongly
	// reject path params that shadow a class field.
	scope := functionResolver.scope.MainSpace()
	for i := range rm.ResourcePath {
		seg := &rm.ResourcePath[i]
		if seg.Kind == ast.ResourcePathSegmentName || seg.Name == "" {
			continue
		}
		name := seg.Name
		if _, exists := scope.GetSymbol(name); exists {
			semanticError(functionResolver, "redeclared symbol '"+name+"'", seg.GetPosition())
			continue
		}
		symbol := model.NewVariableSymbol(name, false, false, true)
		functionResolver.AddSymbol(name, &symbol)
	}
	resolveFunctionInner(functionResolver, rm.RequiredParams, rm.RestParam, rm, rm.Body)
}

func getEnclosingClassDef(resolver symbolResolver) *ast.BLangClassDefinition {
	for {
		bs, ok := resolver.(*blockSymbolResolver)
		if !ok {
			return nil
		}
		if classDef, ok := bs.node.(*ast.BLangClassDefinition); ok {
			return classDef
		}
		resolver = bs.parent
	}
}

func getEnclosingClassBodyScope(resolver symbolResolver) (model.BlockLevelScope, bool) {
	for {
		bs, ok := resolver.(*blockSymbolResolver)
		if !ok {
			return nil, false
		}
		switch bs.node.(type) {
		case *ast.BLangClassDefinition, *ast.BLangService:
			return bs.scope, true
		}
		resolver = bs.parent
	}
}

func isSelfFieldAccess(n *ast.BLangFieldBaseAccess) bool {
	varRef, ok := n.Expr.(*ast.BLangSimpleVarRef)
	if !ok {
		return false
	}
	return varRef.VariableName.Value == "self"
}

func resolveSelfFieldAccess[T symbolResolver](resolver T, n *ast.BLangFieldBaseAccess, classScope model.BlockLevelScope) {
	varRef := n.Expr.(*ast.BLangSimpleVarRef)
	referSimpleVariableReference(resolver, varRef)
	fieldName := n.Field.Value
	if _, ok := classScope.MainSpace().GetSymbol(fieldName); !ok {
		semanticError(resolver, "undefined member '"+fieldName+"'", n.Field.GetPosition())
	}
}

func internalError[T symbolResolver](resolver T, message string, pos diagnostics.Location) {
	resolver.GetCtx().InternalError(message, pos)
}

func semanticError[T symbolResolver](resolver T, message string, pos diagnostics.Location) {
	resolver.GetCtx().SemanticError(message, pos)
}
