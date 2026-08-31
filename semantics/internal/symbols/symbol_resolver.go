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

package symbols

import (
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
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
	GetScope() model.BlockLevelScope
	GetCtx() *context.CompilerContext
	TypeContext() semtypes.Context
	GetTypeDefns() map[model.SymbolRef]*ast.BLangTypeDefinition
	GetClassDefns() map[model.SymbolRef]*ast.BLangClassDefinition
	nextDefaultSymbolName() string
}

type (
	compilationUnitImportsWithSymbols struct {
		compilationUnit *ast.BLangCompilationUnit
		imports         map[string]model.ExportedSymbolSpace
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
		resolver *compilationUnitSymbolResolver
	}

	moduleAstNodeHolder struct {
		typeDefns  map[string]moduleAstNode[*ast.BLangTypeDefinition]
		classDefns map[string]moduleAstNode[*ast.BLangClassDefinition]
	}

	moduleSymbolResolver struct {
		ctx            *context.CompilerContext
		tyCtx          semtypes.Context
		packageScope   *model.ModuleScope
		pkgID          model.PackageID
		typeDefns      map[model.SymbolRef]*ast.BLangTypeDefinition
		classDefns     map[model.SymbolRef]*ast.BLangClassDefinition
		packageSymbols map[string]model.SymbolRef
		prevPos        map[string]prevPos
		prevAnnotPos   map[string]prevPos
		defaultCounter int
		serviceCounter int
		moduleNodes    moduleAstNodeHolder
	}

	compilationUnitSymbolResolver struct {
		moduleResolver *moduleSymbolResolver
		scope          *model.ModuleScope
		usedPrefixes   map[string]bool
		varTracker     varTracker
	}

	blockSymbolResolver struct {
		parent     symbolResolver
		scope      model.BlockLevelScope
		node       ast.BLangNode
		varTracker *varTracker
	}
)

var (
	_ symbolResolver   = &compilationUnitSymbolResolver{}
	_ symbolResolver   = &blockSymbolResolver{}
	_ varStatusTracker = &varTracker{}
)

func markInit(resolver symbolResolver, name string, symbol model.SymbolRef, pos diagnostics.Location) {
	if isIgnoredDeclName(name) {
		return
	}
	resolver.markInit(symbol, pos)
}

func (r *compilationUnitSymbolResolver) markInit(sym model.SymbolRef, pos diagnostics.Location) {
	r.varTracker.markInit(sym, pos)
}

func (r *compilationUnitSymbolResolver) markUsed(sym model.SymbolRef) {
	if r.varTracker.isTracked(sym) {
		r.varTracker.markUsed(sym)
	}
}

func (r *compilationUnitSymbolResolver) getUnused() []varDeclInfo {
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

func newModuleSymbolResolver(ctx *context.CompilerContext, pkgID model.PackageID) *moduleSymbolResolver {
	packageScope := ctx.NewModuleScope(pkgID, nil)
	return &moduleSymbolResolver{
		ctx:            ctx,
		tyCtx:          semtypes.ContextFrom(ctx.GetTypeEnv()),
		packageScope:   packageScope,
		pkgID:          pkgID,
		typeDefns:      make(map[model.SymbolRef]*ast.BLangTypeDefinition),
		classDefns:     make(map[model.SymbolRef]*ast.BLangClassDefinition),
		packageSymbols: make(map[string]model.SymbolRef),
		prevPos:        make(map[string]prevPos),
		prevAnnotPos:   make(map[string]prevPos),
		moduleNodes: moduleAstNodeHolder{
			typeDefns:  make(map[string]moduleAstNode[*ast.BLangTypeDefinition]),
			classDefns: make(map[string]moduleAstNode[*ast.BLangClassDefinition]),
		},
	}
}

func (m *moduleAstNodeHolder) add(cu *ast.BLangCompilationUnit, resolver *compilationUnitSymbolResolver) {
	for _, node := range cu.TopLevelNodes {
		switch n := node.(type) {
		case *ast.BLangTypeDefinition:
			m.typeDefns[n.Name.GetValue()] = moduleAstNode[*ast.BLangTypeDefinition]{node: n, resolver: resolver}
		case *ast.BLangClassDefinition:
			m.classDefns[n.Name.GetValue()] = moduleAstNode[*ast.BLangClassDefinition]{node: n, resolver: resolver}
		}
	}
}

func newCompilationUnitSymbolResolver(moduleResolver *moduleSymbolResolver, scope *model.ModuleScope) *compilationUnitSymbolResolver {
	return &compilationUnitSymbolResolver{
		moduleResolver: moduleResolver,
		scope:          scope,
		usedPrefixes:   make(map[string]bool),
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

func (ms *compilationUnitSymbolResolver) GetSymbol(name string) (model.SymbolRef, scopeKind, bool) {
	if ref, ok := ms.moduleResolver.packageSymbols[name]; ok {
		return ref, moduleScopeKind, true
	}
	ref, ok := ms.moduleResolver.packageScope.Main.GetSymbol(name)
	return ref, moduleScopeKind, ok
}

func (ms *compilationUnitSymbolResolver) GetSymbolFromCurrentScope(name string) (model.SymbolRef, scopeKind, bool) {
	ref, ok := ms.scope.Main.GetSymbol(name)
	return ref, moduleScopeKind, ok
}

func (ms *compilationUnitSymbolResolver) GetPkgID() model.PackageID {
	return ms.moduleResolver.pkgID
}

func (ms *compilationUnitSymbolResolver) GetScope() model.BlockLevelScope {
	return ms.scope
}

func (ms *compilationUnitSymbolResolver) GetPrefixedSymbol(prefix, name string) (model.SymbolRef, bool) {
	if prefix != "" {
		ms.usedPrefixes[prefix] = true
	}
	return ms.scope.GetPrefixedSymbol(prefix, name)
}

func (ms *compilationUnitSymbolResolver) GetAnnotationSymbol(prefix, name string) (model.SymbolRef, bool) {
	if prefix != "" {
		ms.usedPrefixes[prefix] = true
	}
	return ms.scope.GetAnnotationSymbol(prefix, name)
}

func (ms *compilationUnitSymbolResolver) AddSymbol(name string, symbol model.Symbol) {
	ms.scope.AddSymbol(name, symbol)
}

func (ms *compilationUnitSymbolResolver) GetCtx() *context.CompilerContext {
	return ms.moduleResolver.ctx
}

func (ms *compilationUnitSymbolResolver) TypeContext() semtypes.Context {
	return ms.moduleResolver.tyCtx
}

func (ms *moduleSymbolResolver) nextDefaultSymbolName() string {
	name := fmt.Sprintf("$default$%d", ms.defaultCounter)
	ms.defaultCounter++
	return name
}

func (ms *moduleSymbolResolver) nextServiceSymbolName() string {
	name := fmt.Sprintf("$service$%d", ms.serviceCounter)
	ms.serviceCounter++
	return name
}

func (ms *compilationUnitSymbolResolver) nextDefaultSymbolName() string {
	return ms.moduleResolver.nextDefaultSymbolName()
}

func (ms *compilationUnitSymbolResolver) GetTypeDefns() map[model.SymbolRef]*ast.BLangTypeDefinition {
	return ms.moduleResolver.typeDefns
}

func (ms *compilationUnitSymbolResolver) GetClassDefns() map[model.SymbolRef]*ast.BLangClassDefinition {
	return ms.moduleResolver.classDefns
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

func (bs *blockSymbolResolver) GetScope() model.BlockLevelScope {
	return bs.scope
}

func (bs *blockSymbolResolver) GetCtx() *context.CompilerContext {
	return bs.parent.GetCtx()
}

func (bs *blockSymbolResolver) nextDefaultSymbolName() string {
	return bs.parent.nextDefaultSymbolName()
}

func (bs *blockSymbolResolver) TypeContext() semtypes.Context {
	return bs.parent.TypeContext()
}

func associateFunctionSignatureRef(ctx *context.CompilerContext, owner model.SymbolRef, ref model.FunctionSignatureRef, pos diagnostics.Location) {
	if !ctx.AssociateFunctionSignature(owner, ref) {
		ctx.InternalError("function signature already set", pos)
	}
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

func addTopLevelSymbol(resolver *compilationUnitSymbolResolver, name string, symbol model.Symbol, pos diagnostics.Location) bool {
	if prevRef, _, exists := resolver.GetSymbol(name); exists {
		resolver.markUsed(prevRef)
		msg := "redeclared symbol '" + name + "'"
		if prev, ok := resolver.moduleResolver.prevPos[name]; ok && !prev.reported {
			semanticError(resolver, msg, prev.pos)
			prev.reported = true
			resolver.moduleResolver.prevPos[name] = prev
		}
		semanticError(resolver, msg, pos)
		return false
	}
	resolver.AddSymbol(name, symbol)
	ref, _, _ := resolver.GetSymbolFromCurrentScope(name)
	resolver.moduleResolver.packageSymbols[name] = ref
	resolver.moduleResolver.prevPos[name] = prevPos{pos: pos}
	return true
}

func addTopLevelAnnotationSymbol(resolver *compilationUnitSymbolResolver, name string, symbol model.Symbol, pos diagnostics.Location) bool {
	if _, exists := resolver.scope.Annotation.GetSymbol(name); exists {
		msg := "redeclared annotation '" + name + "'"
		if prev, ok := resolver.moduleResolver.prevAnnotPos[name]; ok && !prev.reported {
			semanticError(resolver, msg, prev.pos)
			prev.reported = true
			resolver.moduleResolver.prevAnnotPos[name] = prev
		}
		semanticError(resolver, msg, pos)
		return false
	}
	resolver.scope.AddAnnotationSymbol(name, symbol)
	resolver.moduleResolver.prevAnnotPos[name] = prevPos{pos: pos}
	return true
}

func annotationAttachPointKey(attachPoint ast.AttachPoint) string {
	point := attachPoint.Point.String()
	if attachPoint.Source {
		return model.SourceAnnotationAttachPointKey(point)
	}
	return point
}

func (ms *compilationUnitSymbolResolver) isTypeRefToTypedesc(ref *ast.BLangUserDefinedType, visited map[model.SymbolRef]bool) bool {
	pkgAlias, typeName := ref.PkgAlias.GetValue(), ref.TypeName.Value
	if pkgAlias != "" {
		symRef, ok := ms.GetPrefixedSymbol(pkgAlias, typeName)
		if !ok {
			return false
		}
		ty := ms.moduleResolver.ctx.GetSymbol(symRef).Type()
		return !semtypes.IsZero(ty) && semtypes.IsSubtype(ms.moduleResolver.tyCtx, ty, semtypes.Typedesc)
	}
	symRef, _, ok := ms.GetSymbol(typeName)
	if !ok {
		return false
	}
	if visited[symRef] {
		return false
	}
	visited[symRef] = true
	td, ok := ms.moduleResolver.typeDefns[symRef]
	if !ok {
		return false
	}
	return ms.isDescriptorTypedesc(td.GetTypeData().TypeDescriptor, visited)
}

// isDescriptorTypedesc reports whether a type descriptor AST node is (directly or via a user-
// defined reference chain) a typedesc type.
func (ms *compilationUnitSymbolResolver) isDescriptorTypedesc(desc any, visited map[model.SymbolRef]bool) bool {
	switch tn := desc.(type) {
	case *ast.BLangValueType:
		return tn.TypeKind == ast.TypeKindTypeDesc
	case *ast.BLangBuiltInRefTypeNode:
		return tn.TypeKind == ast.TypeKindTypeDesc
	case *ast.BLangConstrainedType:
		return tn.ConstraintKind() == ast.TypeKindTypeDesc
	case *ast.BLangUserDefinedType:
		return ms.isTypeRefToTypedesc(tn, visited)
	}
	return false
}

// allocateFunctionSymbolInner creates the appropriate function symbol for a function declaration.
// If the return type references a typedesc parameter (dependently-typed), it creates a
// DependentlyTypedFunctionSymbol; otherwise a plain FunctionSymbol. The returned symbol has
// no type information yet — it is filled during type resolution.
func (ms *compilationUnitSymbolResolver) allocateFunctionSymbolInner(fn *ast.BLangFunction, name string, isPublic bool) model.Symbol {
	if ms.isDependentlyTyped(fn) {
		if fn.RestParam != nil {
			ms.moduleResolver.ctx.Unimplemented("rest parameters are not supported on dependently-typed functions", fn.GetPosition())
		}
		if _, isExtern := fn.Body.(*ast.BLangExternFunctionBody); !isExtern {
			ms.moduleResolver.ctx.SemanticError("dependently typed function must be external", fn.GetPosition())
		}
		return model.NewDependentlyTypedFunctionSymbol(name, fn.FuncSymbolFlags(), isPublic, symbolLocationForNode(fn))
	}
	return model.NewFunctionSymbol(name, model.TypedFunctionSignature{}, isPublic, symbolLocationForNode(fn))
}

// isDependentlyTyped reports whether a function's return type references one of its typedesc
// parameters by name.
func (ms *compilationUnitSymbolResolver) isDependentlyTyped(fn ast.InvokableNode) bool {
	retTd := fn.GetReturnTypeDescriptor()
	if retTd == nil {
		return false
	}
	typedescParams := make(map[string]struct{})
	for _, param := range fn.GetParameters() {
		if param.Name == nil {
			continue
		}
		if ms.isDescriptorTypedesc(param.TypeNode(), make(map[model.SymbolRef]bool)) {
			typedescParams[param.Name.GetValue()] = struct{}{}
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
		if n.PkgAlias.GetValue() != "" {
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

type namedDeclaration interface {
	GetName() ast.IdentifierNode
}

func symbolLocationForNode(node namedDeclaration) diagnostics.Location {
	return node.GetName().GetPosition()
}

func Resolve(
	cx *context.CompilerContext,
	pkgID model.PackageID,
	compilationUnits []*ast.BLangCompilationUnit,
	implicitImports map[string]model.ExportedSymbolSpace,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	defaultOrg string,
) (model.Scope, model.ExportedSymbolSpace, map[string]model.ExportedSymbolSpace) {
	cuImportsList := bindImports(cx, compilationUnits, implicitImports, publicSymbols, defaultOrg)
	moduleResolver := newModuleSymbolResolver(cx, pkgID)
	injectOpaqueSymbols(pkgID, moduleResolver)
	cuResolvers := make([]*compilationUnitSymbolResolver, len(cuImportsList))
	for i, cuImports := range cuImportsList {
		scope := cx.NewModuleScope(pkgID, cuImports.imports)
		cuImports.compilationUnit.Scope = scope
		cuResolvers[i] = newCompilationUnitSymbolResolver(moduleResolver, scope)
		moduleResolver.moduleNodes.add(cuImports.compilationUnit, cuResolvers[i])
	}
	for i, resolver := range cuResolvers {
		resolver.allocateTopLevelSymbols(cuImportsList[i].compilationUnit)
	}

	importedSymbols := make(map[string]model.ExportedSymbolSpace)
	for i, cuImports := range cuImportsList {
		cu := cuImports.compilationUnit
		resolver := cuResolvers[i]
		processCompilationUnitXMLNS(resolver, cu)
		ast.Walk(resolver, cu)
		reportUnusedImports(resolver, compilationUnitImports(cu))
		reportUnusedVariables(cx, resolver.getUnused())
		maps.Copy(importedSymbols, cuImports.imports)
	}

	mainSpaces := make([]*model.SymbolSpace, 0, len(cuResolvers)+1)
	annotationSpaces := make([]*model.SymbolSpace, 0, len(cuResolvers)+1)
	for _, resolver := range cuResolvers {
		mainSpaces = append(mainSpaces, resolver.scope.Main)
		annotationSpaces = append(annotationSpaces, resolver.scope.Annotation)
	}
	mainSpaces = append(mainSpaces, moduleResolver.packageScope.Main)
	annotationSpaces = append(annotationSpaces, moduleResolver.packageScope.Annotation)
	pkgScope := &model.PackageScope{Virtual: moduleResolver.packageScope, MainSpaces: mainSpaces}
	return pkgScope, model.NewExportedSymbolSpaces(mainSpaces, annotationSpaces), importedSymbols
}

func (ms *compilationUnitSymbolResolver) allocateTopLevelSymbols(cu *ast.BLangCompilationUnit) {
	for _, node := range cu.TopLevelNodes {
		switch n := node.(type) {
		case *ast.BLangTypeDefinition:
			ms.allocateTypeSymbol(n, make(map[string]struct{}))
		case *ast.BLangFunction:
			ms.allocateFunctionSymbol(n)
		case *ast.BLangVariable:
			if n.IsConstant() {
				ms.allocateConstantSymbol(n)
			} else {
				ms.allocateGlobalVarSymbol(n)
			}
		case *ast.BLangClassDefinition:
			ms.allocateClassSymbol(n)
		case *ast.BLangAnnotation:
			ms.allocateAnnotationSymbol(n)
		}
	}
}

func (ms *compilationUnitSymbolResolver) allocateTypeSymbol(typeDef *ast.BLangTypeDefinition, seen map[string]struct{}) {
	name := typeDef.Name.GetValue()
	if ref, ok := ms.moduleResolver.packageSymbols[name]; ok {
		if existing, ok := ms.moduleResolver.typeDefns[ref]; ok && existing == typeDef {
			return
		}
	}
	isPublic := typeDef.IsPublic()
	var symbol model.Symbol
	var signatureRef model.FunctionSignatureRef
	hasUntypeFunctionSignature := false
	switch ty := typeDef.GetTypeData().TypeDescriptor.(type) {
	case *ast.BLangRecordType:
		symbol = new(model.NewRecordSymbol(name, isPublic, typeDef.Name.GetPosition()))
	case *ast.BLangObjectType:
		symbol = new(model.NewObjectTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
	case *ast.BLangErrorTypeNode:
		symbol = new(model.NewErrorTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
	case *ast.BLangFunctionType:
		symbol = new(model.NewTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
		signatureRef, hasUntypeFunctionSignature = ensureFunctionTypeSignature(ms, ms.scope, ty)
	case *ast.BLangUserDefinedType:
		seen[name] = struct{}{}
		prefix := ty.PkgAlias.Value
		targetName := ty.TypeName.Value
		if prefix == "" {
			ms.ensureTypeAllocated(ty, seen)
		}
		var symRef model.SymbolRef
		var ok bool
		if prefix != "" {
			symRef, ok = ms.GetPrefixedSymbol(prefix, targetName)
		} else {
			symRef, _, ok = ms.GetSymbol(targetName)
		}
		if !ok {
			symbol = new(model.NewTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
			break
		}
		ty.SetSymbol(symRef)
		signatureRef, hasUntypeFunctionSignature = ms.moduleResolver.ctx.FunctionSignatureRef(symRef)
		switch ms.moduleResolver.ctx.GetSymbol(symRef).(type) {
		case *model.RecordSymbol:
			symbol = new(model.NewRecordSymbol(name, isPublic, typeDef.Name.GetPosition()))
		case *model.ErrorTypeSymbol:
			symbol = new(model.NewErrorTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
		case model.ObjectType:
			symbol = new(model.NewObjectTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
		default:
			symbol = new(model.NewTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
		}
	default:
		symbol = new(model.NewTypeSymbol(name, isPublic, typeDef.Name.GetPosition()))
	}
	if !addTopLevelSymbol(ms, name, symbol, typeDef.Name.GetPosition()) {
		return
	}
	symRef, _, _ := ms.GetSymbol(name)
	if typeDef.IsDistinct() {
		switch carrier := ms.moduleResolver.ctx.GetSymbol(symRef).(type) {
		case *model.ErrorTypeSymbol:
			carrier.SetDistinctTypeIDs([]int{ms.moduleResolver.ctx.DistinctTypeID(symRef)})
			registerLangLibDistinctTypeSymbol(ms, typeDef.Name.GetValue(), symRef, typeDef.GetPosition())
		case model.ObjectType:
			carrier.SetDistinctTypeIDs([]int{ms.moduleResolver.ctx.DistinctTypeID(symRef)})
			registerLangLibDistinctTypeSymbol(ms, typeDef.Name.GetValue(), symRef, typeDef.GetPosition())
		default:
			ms.moduleResolver.ctx.Unimplemented("distinct types are only supported for object and error types", typeDef.GetPosition())
		}
	}
	ms.moduleResolver.typeDefns[symRef] = typeDef
	if hasUntypeFunctionSignature {
		associateFunctionSignatureRef(ms.moduleResolver.ctx, symRef, signatureRef, typeDef.GetPosition())
	}
}

func (ms *compilationUnitSymbolResolver) ensureTypeAllocated(ref *ast.BLangUserDefinedType, seen map[string]struct{}) {
	if ref.PkgAlias.Value != "" {
		// Imported symbol should have been resolved already
		return
	}
	name := ref.TypeName.Value
	if _, ok := ms.moduleResolver.packageSymbols[name]; ok {
		return
	}

	if _, ok := seen[name]; ok {
		return
	}
	seen[name] = struct{}{}
	td, ok := ms.moduleResolver.moduleNodes.typeDefns[name]
	if ok {
		td.resolver.allocateTypeSymbol(td.node, seen)
		return
	}
	classDef, ok := ms.moduleResolver.moduleNodes.classDefns[name]
	if !ok {
		// no such symbol to allocate.
		return
	}
	classDef.resolver.allocateClassSymbol(classDef.node)
}

func (ms *compilationUnitSymbolResolver) allocateFunctionSymbol(fn *ast.BLangFunction) {
	name := fn.Name.GetValue()
	symbol := ms.allocateFunctionSymbolInner(fn, name, fn.IsPublic())
	if !addTopLevelSymbol(ms, name, symbol, fn.Name.GetPosition()) {
		return
	}
	ref, _, _ := ms.GetSymbolFromCurrentScope(name)
	fn.SetSymbol(ref)
}

func (ms *compilationUnitSymbolResolver) allocateAnnotationSymbol(annotation *ast.BLangAnnotation) {
	name := annotation.Name.GetValue()
	attachPoints := make([]string, 0, len(annotation.AttachPoints()))
	for _, attachPoint := range annotation.AttachPoints() {
		attachPoints = append(attachPoints, annotationAttachPointKey(attachPoint))
	}
	symbol := model.NewAnnotationSymbol(name, annotation.IsPublic(), annotation.IsConst(), attachPoints, annotation.Name.GetPosition())
	addTopLevelAnnotationSymbol(ms, name, &symbol, annotation.Name.GetPosition())
}

func (ms *compilationUnitSymbolResolver) allocateConstantSymbol(constDef *ast.BLangVariable) {
	name := constDef.Name.GetValue()
	isPublic := constDef.IsPublic()
	if !addTopLevelSymbol(ms, name, model.NewConstantValueSymbol(name, isPublic, constDef.Name.GetPosition()), constDef.Name.GetPosition()) {
		return
	}
	if !isPublic {
		symRef, _, _ := ms.GetSymbol(name)
		markInit(ms, name, symRef, constDef.GetPosition())
	}
}

func (ms *compilationUnitSymbolResolver) allocateGlobalVarSymbol(globalVar *ast.BLangVariable) {
	name := globalVar.Name.GetValue()
	isPublic := globalVar.IsPublic()
	{
		symbol := model.NewVariableSymbol(name, isPublic, false, false, globalVar.Name.GetPosition())
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
	symRef, _, _ := ms.GetSymbolFromCurrentScope(name)
	globalVar.SetSymbol(symRef)
	if !isPublic {
		markInit(ms, name, symRef, globalVar.GetPosition())
	}
}

func (ms *compilationUnitSymbolResolver) allocateClassSymbol(classDef *ast.BLangClassDefinition) {
	name := classDef.Name.GetValue()
	if ref, ok := ms.moduleResolver.packageSymbols[name]; ok {
		if existing, ok := ms.moduleResolver.classDefns[ref]; ok && existing == classDef {
			return
		}
	}
	symbol := newClassSymbolForDefn(classDef)
	if !addTopLevelSymbol(ms, name, symbol, classDef.Name.GetPosition()) {
		return
	}
	symRef, _, _ := ms.GetSymbol(name)
	if classDef.IsDistinct() {
		symbol.SetDistinctTypeIDs([]int{ms.moduleResolver.ctx.DistinctTypeID(symRef)})
		registerLangLibDistinctTypeSymbol(ms, classDef.Name.GetValue(), symRef, classDef.GetPosition())
	}
	ms.moduleResolver.classDefns[symRef] = classDef
}

func registerLangLibDistinctTypeSymbol(ms *compilationUnitSymbolResolver, typeName string, ref model.SymbolRef, pos diagnostics.Location) {
	if ms.moduleResolver.pkgID.OrgName == nil || ms.moduleResolver.pkgID.PkgName == nil ||
		ms.moduleResolver.pkgID.OrgName.Value() != "ballerina" || !strings.HasPrefix(ms.moduleResolver.pkgID.PkgName.Value(), "lang.") {
		return
	}
	if !ms.moduleResolver.ctx.RegisterLangLibDistinctTypeSymbol(ms.moduleResolver.pkgID.PkgName.Value(), typeName, ref) {
		ms.moduleResolver.ctx.InternalError("failed to register lang library distinct type symbol: "+ms.moduleResolver.pkgID.PkgName.Value()+":"+typeName, pos)
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

func reportUnusedImports(resolver *compilationUnitSymbolResolver, imports []ast.BLangImportPackage) {
	for i := range imports {
		imp := &imports[i]
		alias := imp.Alias.Value
		if alias == string(model.IGNORE) {
			continue
		}
		if !resolver.usedPrefixes[alias] {
			resolver.moduleResolver.ctx.SemanticError("unused import prefix '"+alias+"'", imp.GetPosition())
		}
	}
}

func newClassSymbolForDefn(classDef *ast.BLangClassDefinition) model.ClassSymbol {
	name := classDef.Name.GetValue()
	isPublic := classDef.IsPublic()
	location := classDef.Name.GetPosition()
	if classDef.IsClient() || classDef.IsService() {
		return model.NewNetworkClassSymbol(name, isPublic, location)
	}
	return model.NewClassSymbol(name, isPublic, location)
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
	space := r.packageScope.MainSpace()
	for _, sym := range model.OpaqueSymbols(pkg) {
		fillinOpaqueSymbol(sym, space)
		r.packageScope.AddSymbol(sym.Name(), sym)
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

func newMonomorphizationCache() (func(...semtypes.SemType) (model.SymbolRef, bool), func(model.SymbolRef, ...semtypes.SemType)) {
	type cacheNode struct {
		children map[semtypes.InternHandle]*cacheNode
		ref      model.SymbolRef
		stored   bool
	}

	var mu sync.Mutex
	interner := semtypes.NewSemtypeInterner()
	root := cacheNode{children: make(map[semtypes.InternHandle]*cacheNode)}
	nodeFor := func(keys []semtypes.SemType, create bool) *cacheNode {
		if len(keys) == 0 {
			panic("monomorphization cache requires at least one key type")
		}
		node := &root
		for _, key := range keys {
			handle := interner.Intern(key)
			next := node.children[handle]
			if next == nil {
				if !create {
					return nil
				}
				next = &cacheNode{children: make(map[semtypes.InternHandle]*cacheNode)}
				node.children[handle] = next
			}
			node = next
		}
		return node
	}
	lookup := func(keys ...semtypes.SemType) (model.SymbolRef, bool) {
		mu.Lock()
		defer mu.Unlock()
		node := nodeFor(keys, false)
		if node == nil || !node.stored {
			return model.SymbolRef{}, false
		}
		return node.ref, true
	}
	store := func(ref model.SymbolRef, keys ...semtypes.SemType) {
		mu.Lock()
		defer mu.Unlock()
		node := nodeFor(keys, true)
		node.ref = ref
		node.stored = true
	}
	return lookup, store
}

func resolveFunction(functionResolver *blockSymbolResolver, function *ast.BLangFunction) {
	resolveFunctionInner(functionResolver, function.GetParameters(), function.RestParam, function, function.Body)
}

func resolveFunctionInner(functionResolver *blockSymbolResolver, requiredParams []ast.BLangVariable, restParam *ast.BLangVariable, walkNode ast.BLangNode, body ast.FunctionBodyNode) {
	trackParams := !isExternalFunctionBody(body)
	scope := functionResolver.scope.MainSpace()
	for i := range requiredParams {
		param := &requiredParams[i]
		name := param.Name.GetValue()
		if _, exists := scope.GetSymbol(name); exists {
			semanticError(functionResolver, "redeclared symbol '"+name+"'", param.GetPosition())
			continue
		}
		symbol := model.NewVariableSymbol(name, false, false, true, symbolLocationForNode(param))
		addSymbolAndSetOnNode(functionResolver, name, &symbol, param)
		if trackParams {
			markInit(functionResolver, name, param.Symbol(), param.GetPosition())
		}
	}
	if restParam != nil {
		rest := restParam
		name := rest.Name.GetValue()
		if _, exists := scope.GetSymbol(name); exists {
			semanticError(functionResolver, "redeclared symbol '"+name+"'", rest.GetPosition())
		} else {
			symbol := model.NewVariableSymbol(name, false, false, true, symbolLocationForNode(rest))
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

func ensureFunctionTypeSignature(resolver symbolResolver, targetScope model.Scope, fnType *ast.BLangFunctionType) (model.FunctionSignatureRef, bool) {
	if fnType.IsAnyFunction() {
		return 0, false
	}
	if ref := fnType.SignatureRef(); ref != 0 {
		// Already set
		return ref, true
	}
	params := signatureParams(resolver, targetScope, fnType)
	ref := resolver.GetCtx().AllocateFunctionSignature(params, fnType.RestParameter() != nil)
	fnType.SetSignatureRef(ref)
	return ref, true
}

func associateFunctionSignatureFromTypeDescriptor[T symbolResolver](resolver T, owner model.SymbolRef, typeNode any, pos diagnostics.Location) {
	if owner.IsEmpty() {
		return
	}
	ref, ok := functionSignatureRefFromTypeDescriptor(resolver, typeNode, pos)
	if !ok {
		return
	}
	associateFunctionSignatureRef(resolver.GetCtx(), owner, ref, pos)
}

func functionSignatureRefFromTypeDescriptor[T symbolResolver](resolver T, typeNode any, pos diagnostics.Location) (model.FunctionSignatureRef, bool) {
	switch ty := typeNode.(type) {
	case *ast.BLangReturnTypeDescriptor:
		return functionSignatureRefFromTypeDescriptor(resolver, ty.TypeDescriptor, pos)
	case *ast.BLangFunctionType:
		return ensureFunctionTypeSignature(resolver, resolver.GetScope(), ty)
	case *ast.BLangUserDefinedType:
		return resolver.GetCtx().FunctionSignatureRef(ty.Symbol())
	default:
		return 0, false
	}
}

func associateReturnFunctionSignature[T symbolResolver](resolver T, source model.FunctionSignatureRef, returnType any, pos diagnostics.Location) {
	target, found := functionSignatureRefFromTypeDescriptor(resolver, returnType, pos)
	if !found {
		return
	}
	if !resolver.GetCtx().AssociateReturnFunctionSignature(source, target) {
		internalError(resolver, "function return signature already set", pos)
	}
}

type symbolFunctionSignature interface {
	ast.FunctionSignature
	Symbol() model.SymbolRef
}

func allocateSymbols(alloc symbolResolver, targetScope model.Scope, sig symbolFunctionSignature, pos diagnostics.Location) {
	cx := alloc.GetCtx()
	owner := sig.Symbol()
	if owner.IsEmpty() {
		return
	}
	if ref, ok := cx.FunctionSignatureRef(owner); ok {
		associateReturnFunctionSignature(alloc, ref, sig.ReturnType(), pos)
		return
	}
	params := signatureParams(alloc, targetScope, sig)
	ref := cx.AllocateFunctionSignature(params, sig.RestParameter() != nil)
	associateFunctionSignatureRef(cx, owner, ref, pos)
	associateReturnFunctionSignature(alloc, ref, sig.ReturnType(), pos)
}

func signatureParams(alloc symbolResolver, targetScope model.Scope, sig ast.FunctionSignature) []model.Param {
	requiredParams := sig.Parameters()
	params := make([]model.Param, 0, len(requiredParams)+1)
	for _, param := range requiredParams {
		var flag model.ParamFlag
		var defaultParam *model.DefaultableParam
		var includedRecord *model.IncludedRecordMetadata
		if param.IsIncludedRecordParam() {
			flag |= model.ParamFlagIncludedRecordParam
			includedRecord = &model.IncludedRecordMetadata{}
		}
		if param.IsDefaultable() {
			flag |= model.ParamFlagDefaultable
			if _, ok := param.DefaultExpr().(*ast.BLangInferredTypedescDefault); ok {
				defaultParam = &model.DefaultableParam{Kind: model.DefaultableParamKindInferredTypedesc}
			} else {
				name := alloc.nextDefaultSymbolName()
				// Until type resolution we don't know the type of the parameters to create this function signature.
				defaultFnSym := model.NewFunctionSymbol(name, model.TypedFunctionSignature{}, false, param.GetPosition())
				targetScope.AddSymbol(name, defaultFnSym)
				symRef, _ := targetScope.GetSymbol(name)
				defaultParam = &model.DefaultableParam{Symbol: symRef, Kind: model.DefaultableParamKindExpr}
			}
		}
		params = append(params, model.Param{Name: param.ParamName(), Flag: flag, Default: defaultParam, IncludedRecord: includedRecord})
	}
	if rest := sig.RestParameter(); rest != nil {
		params = append(params, model.Param{Name: rest.ParamName(), Flag: model.ParamFlagRestParam})
	}
	return params
}

func resolveLambdaFunction(functionResolver *blockSymbolResolver, parent *blockSymbolResolver, function *ast.BLangFunction) {
	// Check for shadowing on parameters against the enclosing function scope
	params := function.GetParameters()
	for i := range params {
		param := &params[i]
		name := param.Name.GetValue()
		if isShadowed(parent, name) {
			semanticError(functionResolver, "Variable already defined: "+name, param.GetPosition())
		}
		symbol := model.NewVariableSymbol(name, false, false, true, symbolLocationForNode(param))
		addSymbolAndSetOnNode(functionResolver, name, &symbol, param)
		markInit(functionResolver, name, param.Symbol(), param.GetPosition())
	}

	if function.RestParam != nil {
		restParam := function.RestParam
		name := restParam.Name.GetValue()
		if isShadowed(parent, name) {
			semanticError(functionResolver, "Variable already defined: "+name, restParam.GetPosition())
		}
		symbol := model.NewVariableSymbol(name, false, false, true, symbolLocationForNode(restParam))
		addSymbolAndSetOnNode(functionResolver, name, &symbol, restParam)
		markInit(functionResolver, name, restParam.Symbol(), restParam.GetPosition())
	}

	ast.Walk(functionResolver, function)
	reportUnusedVariables(functionResolver.GetCtx(), functionResolver.getUnused())
}

func bindImports(
	ctx *context.CompilerContext,
	compilationUnits []*ast.BLangCompilationUnit,
	implicitImports map[string]model.ExportedSymbolSpace,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	defaultOrg string,
) []compilationUnitImportsWithSymbols {
	result := make([]compilationUnitImportsWithSymbols, len(compilationUnits))
	for i, cu := range compilationUnits {
		imports := make(map[string]model.ExportedSymbolSpace)
		for _, imp := range compilationUnitImports(cu) {
			resolveExternalImport(ctx, &imp, defaultOrg, publicSymbols, imports)
		}
		maps.Copy(imports, implicitImports)
		result[i] = compilationUnitImportsWithSymbols{compilationUnit: cu, imports: imports}
	}
	return result
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

func (bs *blockSymbolResolver) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case ast.BLangBadNode:
		return nil
	case *ast.BLangXMLNS:
		processBlockXMLNS(bs, n)
		if uriExpr := n.GetNamespaceURI(); uriExpr != nil {
			ast.Walk(bs, uriExpr)
		}
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
	case *ast.BLangVariableDef:
		defineVariable(bs, n.GetVariable(), n.GetVariable().IsFinal())
	case *ast.BLangVariable:
		walkSimpleVariableChildren(bs, n, n.Symbol())
		return nil
	case *ast.BLangLambdaFunction:
		fn := n.Function
		name := fn.Name.GetValue()
		signature := model.TypedFunctionSignature{}
		symbol := model.NewFunctionSymbol(name, signature, false, symbolLocationForNode(fn))
		addSymbolAndSetOnNode(bs, name, symbol, fn)
		functionResolver := newFunctionResolver(bs, fn)
		fn.SetScope(functionResolver.scope)
		resolveLambdaFunction(functionResolver, bs, fn)
		allocateSymbols(bs, bs.scope, fn, fn.GetPosition())
		return nil
	default:
		return visitInnerSymbolResolver(bs, n)
	}
	return bs
}

func walkSimpleVariableChildren[T symbolResolver](resolver T, variable *ast.BLangVariable, owner model.SymbolRef) {
	if variable.Name != nil {
		ast.Walk(resolver, variable.Name)
	}
	for i := range variable.AnnAttachments {
		ast.Walk(resolver, &variable.AnnAttachments[i])
	}
	if typeNode := variable.TypeNode(); typeNode != nil {
		ast.Walk(resolver, typeNode.(ast.BLangNode))
		associateFunctionSignatureFromTypeDescriptor(resolver, owner, typeNode, variable.GetPosition())
	}
	if variable.Expr != nil {
		ast.Walk(resolver, variable.Expr.(ast.BLangNode))
	}
}

func resolveFunctionTypeSymbols[T symbolResolver](resolver T, fnType *ast.BLangFunctionType) {
	ensureFunctionTypeSignature(resolver, resolver.GetScope(), fnType)
	paramScope := resolver.GetCtx().NewBlockScope(resolver.GetScope(), resolver.GetPkgID())
	paramResolver := &blockSymbolResolver{parent: resolver, scope: paramScope, node: fnType}
	for i := range fnType.RequiredParams {
		param := fnType.RequiredParams[i]
		if param.TypeDesc != nil {
			ast.Walk(resolver, param.TypeDesc.(ast.BLangNode))
		}
		if param.Name != nil {
			name := param.Name.GetValue()
			symbol := model.NewVariableSymbol(name, false, false, true, param.Name.GetPosition())
			paramScope.AddSymbol(name, &symbol)
			ref, _ := paramScope.GetSymbol(name)
			param.SymbolRef = ref
			param.Name.SetDeterminedType(semtypes.Never)
			associateFunctionSignatureFromTypeDescriptor(resolver, param.SymbolRef, param.TypeDesc, param.GetPosition())
		}
		if param.InitExpr != nil {
			ast.Walk(paramResolver, param.InitExpr.(ast.BLangNode))
		}
	}
	if fnType.RestParam != nil {
		param := fnType.RestParam
		if param.TypeDesc != nil {
			ast.Walk(resolver, param.TypeDesc.(ast.BLangNode))
		}
		if param.Name != nil {
			name := param.Name.GetValue()
			symbol := model.NewVariableSymbol(name, false, false, true, param.Name.GetPosition())
			paramScope.AddSymbol(name, &symbol)
			ref, _ := paramScope.GetSymbol(name)
			param.SymbolRef = ref
			param.Name.SetDeterminedType(semtypes.Never)
			associateFunctionSignatureFromTypeDescriptor(resolver, param.SymbolRef, param.TypeDesc, param.GetPosition())
		}
	}
	if fnType.ReturnTypeDescriptor != nil {
		ast.Walk(resolver, fnType.ReturnTypeDescriptor.(ast.BLangNode))
	}
	if ref := fnType.SignatureRef(); ref != 0 {
		associateReturnFunctionSignature(resolver, ref, fnType.ReturnTypeDescriptor, fnType.GetPosition())
	}
}

func visitInnerSymbolResolver[T symbolResolver](resolver T, node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case ast.BLangBadNode:
		return nil
	case *ast.BLangFunctionType:
		resolveFunctionTypeSymbols(resolver, n)
		return nil
	case *ast.BMethodDecl:
		resolveFunctionTypeSymbols(resolver, &n.BLangFunctionType)
		if n.Symbol().IsEmpty() {
			space := resolver.GetScope().MainSpace()
			index := space.AppendSymbol(model.NewFunctionSymbol(n.Name(), model.TypedFunctionSignature{}, false, n.GetPosition()))
			n.SetSymbol(space.RefAt(index))
		}
		associateFunctionSignatureRef(resolver.GetCtx(), n.Symbol(), n.SignatureRef(), n.GetPosition())
		return nil
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
		if common.IsSelfFieldAccess(n) {
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
	case *ast.BLangVariable:
		referVariable(resolver, n)
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
type invocable interface {
	GetName() ast.IdentifierNode
	SetRawSymbol(model.Symbol)
}

func createDeferredMethodSymbol[T symbolResolver](resolver T, n invocable) {
	name := n.GetName().GetValue()
	scope := resolver.GetScope().(model.SymbolSpaceProvider)
	n.SetRawSymbol(common.NewDeferredMethodSymbol(name, scope.MainSpace()))
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

func resolveAnnotationReference[T symbolResolver](resolver T, pkgAlias, name ast.IdentifierNode, pos diagnostics.Location, target symbolRefNode) {
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
	GetName() ast.IdentifierNode
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

func defineVariable(resolver *blockSymbolResolver, variable *ast.BLangVariable, isFinal bool) {
	name := variable.Name.GetValue()
	if isShadowed(resolver, name) {
		semanticError(resolver, "Variable already defined: "+name, variable.GetPosition())
	}
	symbol := model.NewVariableSymbol(name, false, isFinal, false, symbolLocationForNode(variable))
	if isFinal {
		symbol.SetFinal()
	}
	addSymbolAndSetOnNode(resolver, name, &symbol, variable)
	markInit(resolver, name, variable.Symbol(), variable.GetPosition())
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
		case *ast.BLangFunctionType:
			return
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

func (ms *compilationUnitSymbolResolver) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangFunction:
		if n.Symbol().IsEmpty() {
			return nil
		}
		functionResolver := newFunctionResolver(ms, n)
		n.SetScope(functionResolver.scope)
		resolveFunction(functionResolver, n)
		allocateSymbols(ms, ms.scope, n, n.GetPosition())
		return nil
	case *ast.BLangVariable:
		if !n.IsConstant() && n.Symbol().IsEmpty() {
			return nil
		}
		name := n.Name.GetValue()
		symRef, _, ok := ms.GetSymbol(name)
		if !ok {
			kind := "variable"
			if n.IsConstant() {
				kind = "constant"
			}
			internalError(ms, "Module level "+kind+" symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		walkSimpleVariableChildren(ms, n, symRef)
		return nil
	case *ast.BLangTypeDefinition:
		name := n.Name.GetValue()
		symRef, _, ok := ms.GetSymbol(name)
		if !ok {
			internalError(ms, "Module level type symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		return &functionSignatureTypeDataResolver{
			resolver: ms,
			owner:    symRef,
			typeNode: n.GetTypeData().TypeDescriptor,
			pos:      n.GetPosition(),
		}
	case *ast.BLangAnnotation:
		name := n.Name.GetValue()
		symRef, ok := ms.GetAnnotationSymbol("", name)
		if !ok {
			internalError(ms, "Module level annotation symbol not found: "+name, n.Name.GetPosition())
		}
		n.SetSymbol(symRef)
		return ms
	case *ast.BLangClassDefinition:
		name := n.Name.GetValue()
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
		name := fn.Name.GetValue()
		signature := model.TypedFunctionSignature{}
		symbol := model.NewFunctionSymbol(name, signature, false, symbolLocationForNode(fn))
		ms.AddSymbol(name, symbol)
		symRef, _, _ := ms.GetSymbolFromCurrentScope(name)
		fn.SetSymbol(symRef)
		functionResolver := newFunctionResolver(ms, fn)
		fn.SetScope(functionResolver.scope)
		resolveLambdaFunction(functionResolver, functionResolver, fn)
		allocateSymbols(ms, ms.scope, fn, fn.GetPosition())
		return nil
	default:
		return visitInnerSymbolResolver(ms, n)
	}
}

func (ms *compilationUnitSymbolResolver) VisitTypeData(_ *ast.TypeData) ast.Visitor {
	return ms
}

type functionSignatureTypeDataResolver struct {
	resolver symbolResolver
	owner    model.SymbolRef
	typeNode ast.TypeDescriptor
	pos      diagnostics.Location
}

func (r *functionSignatureTypeDataResolver) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		associateFunctionSignatureFromTypeDescriptor(r.resolver, r.owner, r.typeNode, r.pos)
		return nil
	}
	visitor := r.resolver.Visit(node)
	if visitor == r.resolver {
		return r
	}
	return visitor
}

func (r *functionSignatureTypeDataResolver) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	visitor := r.resolver.VisitTypeData(typeData)
	if visitor == r.resolver {
		return r
	}
	return visitor
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
				if semtypes.IsZero(incTy) || !semtypes.IsSubtype(resolver.TypeContext(), incTy, semtypes.Object) {
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
			if semtypes.IsZero(incTy) || !semtypes.IsSubtype(resolver.TypeContext(), incTy, semtypes.Mapping) {
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
	for _, field := range defn.Fields {
		directFields = append(directFields, inclusionMemberForSymbolResolution{
			name:     field.Name.GetValue(),
			isPublic: field.IsPublic(),
		})
	}
	return collectTransitiveFields(ctx, defn.Inclusions, directFields, localTypeDefns, localClassDefns)
}

func resolveServiceDefinition(ms *compilationUnitSymbolResolver, svc *ast.BLangService) {
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

	allocateServiceResourceMethodSymbols(ms, svcResolver, svc.ResourceMethods)

	finishResolveClassDefinition(ms, svcResolver, svc.Fields, svc.Methods, svc.ResourceMethods, svc.InitFunction, nil, svcResolver.scope, serviceMethodSymbolName, resourceMethodsAreNetworkClass)

	serviceSymbolName := ms.moduleResolver.nextServiceSymbolName()
	serviceSymbol := model.NewTypeSymbol(serviceSymbolName, false, svc.GetPosition())
	svcResolver.AddSymbol(serviceSymbolName, &serviceSymbol)
	serviceSymbolRef, _, _ := svcResolver.GetSymbol(serviceSymbolName)
	svc.SetSymbol(serviceSymbolRef)
	for i := range svc.AnnAttachments {
		ast.Walk(svcResolver, &svc.AnnAttachments[i])
	}
}

func resolveClassDefinition(ms *compilationUnitSymbolResolver, classDef *ast.BLangClassDefinition) {
	className := classDef.Name.GetValue()
	classMethodSymbolName := func(methodName string) string {
		return className + "." + methodName
	}
	classSym := ms.moduleResolver.ctx.GetSymbol(classDef.Symbol()).(model.ClassSymbol)
	networkClassSym, isNetworkClass := ms.moduleResolver.ctx.GetSymbol(classDef.Symbol()).(*model.NetworkClassSymbol)

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

func finishResolveClassDefinition(ms *compilationUnitSymbolResolver, blockRes *blockSymbolResolver, fields []*ast.BLangVariable, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod, initFn *ast.BLangFunction, includedFields []inclusionMemberForSymbolResolution, methodTargetScope methodSymbolTargetScope, methodSymbolName func(string) string, resourceMethodsAreNetworkClass bool) {
	for _, field := range fields {
		name := field.GetName().GetValue()
		if _, sk, exists := blockRes.GetSymbol(name); exists && sk == blockScopeKind {
			semanticError(blockRes, "redeclared symbol '"+name+"'", field.GetPosition())
			continue
		}
		isPublic := field.IsPublic()
		symbol := model.NewVariableSymbol(name, isPublic, false, false, symbolLocationForNode(field))
		blockRes.AddSymbol(name, &symbol)
		symRef, _ := blockRes.scope.MainSpace().GetSymbol(name)
		field.SetSymbol(symRef)
	}

	orderedMethods := common.MethodsInResolutionOrder(methods)
	for _, m := range orderedMethods {
		if _, sk, exists := blockRes.GetSymbol(m.Name); exists && sk == blockScopeKind {
			semanticError(blockRes, "redeclared symbol '"+model.StripRemotePrefix(m.Name)+"'", m.Method.Name.GetPosition())
			continue
		}
		isPublic := m.Method.IsPublic()
		symbol := ms.allocateFunctionSymbolInner(m.Method, m.Name, isPublic)
		symbolName := methodSymbolName(m.Name)
		methodTargetScope.AddSymbol(symbolName, symbol)
		symRef, _ := methodTargetScope.MainSpace().GetSymbol(symbolName)
		m.Method.SetSymbol(symRef)
	}

	for _, m := range includedFields {
		if _, _, exists := blockRes.GetSymbol(m.name); exists {
			continue
		}
		symbol := model.NewVariableSymbol(m.name, m.isPublic, false, false, diagnostics.NewBuiltinLocation())
		blockRes.AddSymbol(m.name, &symbol)
	}

	if initFn != nil {
		symbol := ms.allocateFunctionSymbolInner(initFn, "init", initFn.IsPublic())
		symbolName := methodSymbolName("init")
		methodTargetScope.AddSymbol(symbolName, symbol)
		symRef, _ := methodTargetScope.MainSpace().GetSymbol(symbolName)
		initFn.SetSymbol(symRef)
	}

	selfSymbol := model.NewVariableSymbol("self", false, false, false, diagnostics.NewBuiltinLocation())
	blockRes.AddSymbol("self", &selfSymbol)

	for _, field := range fields {
		ast.Walk(blockRes, field)
	}

	if initFn != nil {
		initResolver := newFunctionResolver(blockRes, initFn)
		initFn.SetScope(initResolver.scope)
		resolveFunction(initResolver, initFn)
		allocateSymbols(ms, ms.scope, initFn, initFn.GetPosition())
	}

	for _, m := range orderedMethods {
		methodResolver := newFunctionResolver(blockRes, m.Method)
		m.Method.SetScope(methodResolver.scope)
		resolveFunction(methodResolver, m.Method)
		allocateSymbols(ms, ms.scope, m.Method, m.Method.GetPosition())
	}

	for _, rm := range resourceMethods {
		if !resourceMethodsAreNetworkClass {
			continue
		}
		methodResolver := newFunctionResolver(blockRes, rm)
		rm.SetScope(methodResolver.scope)
		resolveResourceMethod(methodResolver, rm)
		allocateSymbols(ms, ms.scope, rm, rm.GetPosition())
	}
}

type methodSymbolTargetScope interface {
	model.Scope
	MainSpace() *model.SymbolSpace
}

func allocateObjectResourceMethodSymbols(ms *compilationUnitSymbolResolver, blockRes *blockSymbolResolver, classDef *ast.BLangClassDefinition, networkClassSym *model.NetworkClassSymbol, isNetworkClass bool) {
	className := classDef.Name.GetValue()
	for idx, rm := range classDef.ResourceMethods {
		if !isNetworkClass {
			semanticError(blockRes, "resource methods are only allowed in client or service classes", rm.GetPosition())
			continue
		}
		mangledName := className + "." + mangledResourceMethodName(rm.Name.GetValue(), idx)
		symRef := ms.allocateResourceMethodSymbol(ms.scope, rm, mangledName, classDef.IsPublic() && rm.IsPublic())
		networkClassSym.AddResourceMethod(symRef)
	}
}

func allocateServiceResourceMethodSymbols(ms *compilationUnitSymbolResolver, blockRes *blockSymbolResolver, resourceMethods []*ast.BLangResourceMethod) {
	for idx, rm := range resourceMethods {
		key := mangledResourceMethodName(rm.Name.GetValue(), idx)
		ms.allocateResourceMethodSymbol(blockRes.scope, rm, key, rm.IsPublic())
	}
}

func (ms *compilationUnitSymbolResolver) allocateResourceMethodSymbol(targetScope methodSymbolTargetScope, rm *ast.BLangResourceMethod, symbolName string, isPublic bool) model.SymbolRef {
	var symbol model.ResourceMethodSymbol
	if ms.isDependentlyTyped(rm) {
		if rm.GetRestParam() != nil {
			ms.moduleResolver.ctx.Unimplemented("rest parameters are not supported on dependently-typed functions", rm.GetPosition())
		}
		if _, isExtern := rm.GetBody().(*ast.BLangExternFunctionBody); !isExtern {
			ms.moduleResolver.ctx.SemanticError("dependently typed function must be external", rm.GetPosition())
		}
		symbol = model.NewDependentlyTypedResourceMethodSymbol(symbolName, rm.Name.GetValue(), rm.FuncSymbolFlags(), isPublic, symbolLocationForNode(rm))
	} else {
		symbol = model.NewResourceMethodSymbol(symbolName, rm.Name.GetValue(), isPublic, symbolLocationForNode(rm))
	}
	targetScope.AddSymbol(symbolName, symbol)
	symRef, _ := targetScope.MainSpace().GetSymbol(symbolName)
	rm.SetSymbol(symRef)
	return symRef
}

func publishObjectMethodTable(classSym model.ClassSymbol, classDef *ast.BLangClassDefinition) {
	methodTable := make(map[string]model.SymbolRef, len(classDef.Methods))
	for _, m := range common.MethodsInResolutionOrder(classDef.Methods) {
		methodTable[m.Name] = m.Method.Symbol()
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
		symbol := model.NewVariableSymbol(name, false, false, true, seg.GetPosition())
		functionResolver.AddSymbol(name, &symbol)
	}
	resolveFunctionInner(functionResolver, rm.GetParameters(), rm.RestParam, rm, rm.Body)
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

func resolveSelfFieldAccess[T symbolResolver](resolver T, n *ast.BLangFieldBaseAccess, classScope model.BlockLevelScope) {
	varRef := n.Expr.(*ast.BLangVarRef)
	referSimpleVariableReference(resolver, varRef)
	fieldName := n.Field.GetValue()
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
