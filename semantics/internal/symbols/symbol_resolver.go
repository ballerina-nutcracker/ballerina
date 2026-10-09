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

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/opaque"
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
	// recordDefaultScope is the module level scope owning generated record field default functions. These
	// have to live in a module level scope since they are part of the module's exported symbols.
	recordDefaultScope() model.Scope
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

func (ms *compilationUnitSymbolResolver) recordDefaultScope() model.Scope {
	return ms.scope
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

func (bs *blockSymbolResolver) recordDefaultScope() model.Scope {
	return bs.parent.recordDefaultScope()
}

func (bs *blockSymbolResolver) TypeContext() semtypes.Context {
	return bs.parent.TypeContext()
}

func associateFunctionSignatureRef(ctx *context.CompilerContext, owner model.SymbolRef, ref model.FunctionSignatureRef, pos diagnostics.Location) bool {
	if !ctx.AssociateFunctionSignature(owner, ref) {
		ctx.InternalError("function signature already set", pos)
		return false
	}
	return true
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
			resolver.GetCtx().SemanticError(msg, prev.pos)
			prev.reported = true
			resolver.moduleResolver.prevPos[name] = prev
		}
		resolver.GetCtx().SemanticError(msg, pos)
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
			resolver.GetCtx().SemanticError(msg, prev.pos)
			prev.reported = true
			resolver.moduleResolver.prevAnnotPos[name] = prev
		}
		resolver.GetCtx().SemanticError(msg, pos)
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
func (ms *compilationUnitSymbolResolver) allocateFunctionSymbolInner(fn *ast.BLangFunction, name string, isPublic bool) (model.Symbol, bool) {
	if ms.isDependentlyTyped(fn) {
		ok := true
		if fn.RestParam != nil {
			ms.moduleResolver.ctx.Unimplemented("rest parameters are not supported on dependently-typed functions", fn.GetPosition())
			ok = false
		}
		if _, isExtern := fn.Body.(*ast.BLangExternFunctionBody); !isExtern {
			ms.moduleResolver.ctx.SemanticError("dependently typed function must be external", fn.GetPosition())
			ok = false
		}
		return model.NewDependentlyTypedFunctionSymbol(name, fn.FuncSymbolFlags(), isPublic, symbolLocationForNode(fn)), ok
	}
	return model.NewFunctionSymbol(name, model.TypedFunctionSignature{}, isPublic, symbolLocationForNode(fn)), true
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
	case *ast.BLangArrayType:
		return returnTypeReferencesTypedescParam(n.Elemtype.TypeDescriptor.(ast.BLangNode), typedescParams)
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

func addSymbolAndSetOnNode(resolver symbolResolver, name string, symbol model.Symbol, node ast.BNodeWithSymbol) {
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
	moduleVisibility map[PackageIdentifier]ModuleVisibility,
	defaultOrg, currentPackageName string,
) (model.Scope, model.ExportedSymbolSpace, map[string]model.ExportedSymbolSpace) {
	cuImportsList := bindImports(cx, compilationUnits, implicitImports, publicSymbols, moduleVisibility, defaultOrg, currentPackageName)
	moduleResolver := newModuleSymbolResolver(cx, pkgID)
	// Opaque symbols go into the module scope before source top-level symbols are
	// allocated, so a declaration marked @opaque finds its symbol already there and
	// binds to it instead of declaring a second one.
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
		resolveCompilationUnit(resolver, cu)
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
	typeDef.SetSymbol(symRef)
	ms.moduleResolver.typeDefns[symRef] = typeDef
	if typeDef.IsDistinct() {
		var ok bool
		switch carrier := ms.moduleResolver.ctx.GetSymbol(symRef).(type) {
		case *model.ErrorTypeSymbol:
			carrier.SetDistinctTypeIDs([]int{ms.moduleResolver.ctx.DistinctTypeID(symRef)})
			ok = registerLangLibDistinctTypeSymbol(ms, typeDef.Name.GetValue(), symRef, typeDef.GetPosition())
		case model.ObjectType:
			carrier.SetDistinctTypeIDs([]int{ms.moduleResolver.ctx.DistinctTypeID(symRef)})
			ok = registerLangLibDistinctTypeSymbol(ms, typeDef.Name.GetValue(), symRef, typeDef.GetPosition())
		default:
			ms.moduleResolver.ctx.Unimplemented("distinct types are only supported for object and error types", typeDef.GetPosition())
		}
		if !ok {
			return
		}
	}
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

func (ms *compilationUnitSymbolResolver) allocateFunctionSymbol(fn *ast.BLangFunction) bool {
	name := fn.Name.GetValue()
	if isLangLibPackage(ms) && isOpaqueDeclaration(fn) {
		return ms.bindOpaqueDeclaration(fn, name)
	}
	symbol, ok := ms.allocateFunctionSymbolInner(fn, name, fn.IsPublic())
	if !addTopLevelSymbol(ms, name, symbol, fn.Name.GetPosition()) {
		return false
	}
	ref, _, _ := ms.GetSymbolFromCurrentScope(name)
	fn.SetSymbol(ref)
	return ok
}

// opaqueMarker marks a lang library declaration as the source form of an opaque
// function: the compiler takes the parameter names, defaults and documentation from
// the declaration, but the types of a call are decided by the function's
// monomorphizer rather than by the declared signature. It is meaningful only in a
// ballerina/lang.* package; anywhere else it is an ordinary documentation line.
const opaqueMarker = "@opaque"

// isOpaqueDeclaration reports whether fn carries the opaque marker in its
// documentation.
func isOpaqueDeclaration(fn *ast.BLangFunction) bool {
	doc := fn.GetMarkdownDocumentationAttachment()
	if doc == nil {
		return false
	}
	for _, line := range doc.DocumentationLines {
		if strings.TrimSpace(line.Text) == opaqueMarker {
			return true
		}
	}
	return false
}

// bindOpaqueDeclaration points a declaration marked @opaque at the opaque symbol
// injected for it, rather than declaring a function symbol of its own. Everything
// after this treats the declaration as an ordinary function: its untyped signature,
// including a $default$N provider per defaulted parameter, is allocated from the AST
// by allocateSymbols, and its defaults are desugared like any other function's.
func (ms *compilationUnitSymbolResolver) bindOpaqueDeclaration(fn *ast.BLangFunction, name string) bool {
	ref, ok := ms.moduleResolver.packageScope.GetSymbol(name)
	if !ok {
		ms.GetCtx().SemanticError("no opaque function '"+name+"' in this module", fn.Name.GetPosition())
		return false
	}
	if _, ok := ms.GetCtx().GetSymbol(ref).(*model.OpaqueFunctionSymbol); !ok {
		ms.GetCtx().SemanticError("'"+name+"' is not an opaque function", fn.Name.GetPosition())
		return false
	}
	fn.SetSymbol(ref)
	return true
}

func (ms *compilationUnitSymbolResolver) allocateAnnotationSymbol(annotation *ast.BLangAnnotation) bool {
	name := annotation.Name.GetValue()
	attachPoints := make([]string, 0, len(annotation.AttachPoints()))
	for _, attachPoint := range annotation.AttachPoints() {
		attachPoints = append(attachPoints, annotationAttachPointKey(attachPoint))
	}
	symbol := model.NewAnnotationSymbol(name, annotation.IsPublic(), annotation.IsConst(), attachPoints, annotation.Name.GetPosition())
	if !addTopLevelAnnotationSymbol(ms, name, &symbol, annotation.Name.GetPosition()) {
		return false
	}
	symRef, _ := ms.scope.Annotation.GetSymbol(name)
	annotation.SetSymbol(symRef)
	return true
}

func (ms *compilationUnitSymbolResolver) allocateConstantSymbol(constDef *ast.BLangVariable) bool {
	name := constDef.Name.GetValue()
	isPublic := constDef.IsPublic()
	if !addTopLevelSymbol(ms, name, model.NewConstantValueSymbol(name, isPublic, constDef.Name.GetPosition()), constDef.Name.GetPosition()) {
		return false
	}
	symRef, _, _ := ms.GetSymbol(name)
	constDef.SetSymbol(symRef)
	if !isPublic {
		markInit(ms, name, symRef, constDef.GetPosition())
	}
	return true
}

func (ms *compilationUnitSymbolResolver) allocateGlobalVarSymbol(globalVar *ast.BLangVariable) bool {
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
			return false
		}
	}
	symRef, _, _ := ms.GetSymbolFromCurrentScope(name)
	globalVar.SetSymbol(symRef)
	if !isPublic {
		markInit(ms, name, symRef, globalVar.GetPosition())
	}
	return true
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
	classDef.SetSymbol(symRef)
	ms.moduleResolver.classDefns[symRef] = classDef
	if classDef.IsDistinct() {
		symbol.SetDistinctTypeIDs([]int{ms.moduleResolver.ctx.DistinctTypeID(symRef)})
		registerLangLibDistinctTypeSymbol(ms, classDef.Name.GetValue(), symRef, classDef.GetPosition())
	}
}

// isLangLibPackage reports whether the module being resolved belongs to a
// ballerina/lang.* package, the only place lang library specific declarations such
// as distinct type symbols and opaque function declarations are honoured.
func isLangLibPackage(ms *compilationUnitSymbolResolver) bool {
	pkgID := ms.moduleResolver.pkgID
	return pkgID.OrgName != nil && pkgID.PkgName != nil &&
		pkgID.OrgName.Value() == "ballerina" && strings.HasPrefix(pkgID.PkgName.Value(), "lang.")
}

func registerLangLibDistinctTypeSymbol(ms *compilationUnitSymbolResolver, typeName string, ref model.SymbolRef, pos diagnostics.Location) bool {
	if !isLangLibPackage(ms) {
		return true
	}
	if !ms.moduleResolver.ctx.RegisterLangLibDistinctTypeSymbol(ms.moduleResolver.pkgID.PkgName.Value(), typeName, ref) {
		ms.moduleResolver.ctx.InternalError("failed to register lang library distinct type symbol: "+ms.moduleResolver.pkgID.PkgName.Value()+":"+typeName, pos)
		return false
	}
	return true
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

func reportUnusedVariables(ctx *context.CompilerContext, unused []varDeclInfo) bool {
	for _, v := range unused {
		name := ctx.SymbolName(v.varSym)
		ctx.SemanticError("unused variable '"+name+"'", v.pos)
	}
	return len(unused) == 0
}

func reportUnusedImports(resolver *compilationUnitSymbolResolver, imports []ast.BLangImportPackage) bool {
	ok := true
	for i := range imports {
		imp := &imports[i]
		alias := imp.Alias.Value
		if imp.Alias.OriginalValue == string(model.IGNORE) {
			continue
		}
		if !resolver.usedPrefixes[alias] {
			resolver.moduleResolver.ctx.SemanticError("unused import prefix '"+alias+"'", imp.GetPosition())
			ok = false
		}
	}
	return ok
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
		r.packageScope.AddSymbol(sym.Name(), sym)
		ref, _ := r.packageScope.GetSymbol(sym.Name())
		fillinOpaqueSymbol(r.ctx, sym, space, pkg, ref)
	}
}

// fillinOpaqueSymbol fills in any information that needs to be stored in the opaque symbol
// that is used within semantic package. An opaque function symbol gets the symbol space its
// monomorphizations are added to and the single untyped signature every monomorphization of
// it shares.
//
// A definition declared in the lang library's own source gets no signature here: the
// declaration marked @opaque allocates it from the AST, like any other function.
func fillinOpaqueSymbol(ctx *context.CompilerContext, sym model.Symbol, space *model.SymbolSpace,
	pkg model.PackageIdentifier, ref model.SymbolRef) bool {
	fn, ok := sym.(*model.OpaqueFunctionSymbol)
	if !ok {
		return true
	}
	fn.SymbolSpace = space
	location := diagnostics.NewBuiltinLocation()
	definition, found := opaque.LookupFunction(pkg.Organization, pkg.Package, fn.OpaqueID())
	if !found {
		ctx.InternalError("no definition for opaque function", location)
		return false
	}
	if definition.Name() != fn.Name() {
		ctx.InternalError("opaque function definition name mismatch", location)
		return false
	}
	if definition.IsSourceDeclared() {
		return true
	}
	sigRef := ctx.AllocateFunctionSignature(definition.Params(), definition.HasRest())
	return associateFunctionSignatureRef(ctx, ref, sigRef, location)
}

func resolveFunction(functionResolver *blockSymbolResolver, function *ast.BLangFunction) bool {
	return resolveFunctionInner(functionResolver, function.GetParameters(), function.RestParam, function, function.Body)
}

func resolveFunctionInner(functionResolver *blockSymbolResolver, requiredParams []ast.BLangVariable, restParam *ast.BLangVariable, fn ast.InvokableNode, body ast.FunctionBodyNode) bool {
	trackParams := !isExternalFunctionBody(body)
	scope := functionResolver.scope.MainSpace()
	ok := true
	for i := range requiredParams {
		param := &requiredParams[i]
		name := param.Name.GetValue()
		if _, exists := scope.GetSymbol(name); exists {
			functionResolver.GetCtx().SemanticError("redeclared symbol '"+name+"'", param.GetPosition())
			ok = false
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
			functionResolver.GetCtx().SemanticError("redeclared symbol '"+name+"'", rest.GetPosition())
			ok = false
		} else {
			symbol := model.NewVariableSymbol(name, false, false, true, symbolLocationForNode(rest))
			addSymbolAndSetOnNode(functionResolver, name, &symbol, rest)
			if trackParams {
				markInit(functionResolver, name, rest.Symbol(), rest.GetPosition())
			}
		}
	}
	ok = resolveInvokableChildren(functionResolver, fn) && ok
	ok = reportUnusedVariables(functionResolver.GetCtx(), functionResolver.getUnused()) && ok
	return ok
}

func resolveInvokableChildren(functionResolver *blockSymbolResolver, fn ast.InvokableNode) bool {
	ok := true
	attachments := fn.GetAnnotationAttachments()
	for i := range attachments {
		ok = resolveAnnotationAttachment(functionResolver, &attachments[i]) && ok
	}
	if rm, isResourceMethod := fn.(*ast.BLangResourceMethod); isResourceMethod {
		for i := range rm.ResourcePath {
			if paramType := rm.ResourcePath[i].ParamType; paramType != nil {
				ok = resolveTypeDesc(functionResolver, paramType) && ok
			}
		}
	}
	params := fn.GetParameters()
	for i := range params {
		ok = resolveVariableChildren(functionResolver, &params[i], params[i].Symbol()) && ok
	}
	if rest := fn.GetRestParam(); rest != nil {
		ok = resolveVariableChildren(functionResolver, rest, rest.Symbol()) && ok
	}
	if returnType := fn.GetReturnTypeDescriptor(); returnType != nil {
		ok = resolveTypeDesc(functionResolver, returnType) && ok
	}
	if body := fn.GetBody(); body != nil {
		ok = resolveFunctionBody(functionResolver, body) && ok
	}
	return ok
}

func resolveFunctionBody(r *blockSymbolResolver, body ast.FunctionBodyNode) bool {
	switch b := body.(type) {
	case *ast.BLangBlockFunctionBody:
		return resolveStmts(r, b.Stmts)
	case *ast.BLangExprFunctionBody:
		if b.Expr == nil {
			return true
		}
		return resolveExpr(r, b.Expr)
	case *ast.BLangExternFunctionBody:
		return true
	default:
		return reportUnexpectedNode(r, body)
	}
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
	params := signatureParams(resolver, targetScope, targetScope, fnType)
	ref := resolver.GetCtx().AllocateFunctionSignature(params, fnType.RestParameter() != nil)
	fnType.SetSignatureRef(ref)
	return ref, true
}

func associateFunctionSignatureFromTypeDescriptor(resolver symbolResolver, owner model.SymbolRef, typeNode any, pos diagnostics.Location) bool {
	if owner.IsEmpty() {
		return true
	}
	ref, ok := functionSignatureRefFromTypeDescriptor(resolver, typeNode, pos)
	if !ok {
		return true
	}
	return associateFunctionSignatureRef(resolver.GetCtx(), owner, ref, pos)
}

func functionSignatureRefFromTypeDescriptor(resolver symbolResolver, typeNode any, _ diagnostics.Location) (model.FunctionSignatureRef, bool) {
	if returnType, ok := typeNode.(*ast.BLangReturnTypeDescriptor); ok {
		typeNode = returnType.TypeDescriptor
	}
	switch ty := typeNode.(type) {
	case *ast.BLangFunctionType:
		return ensureFunctionTypeSignature(resolver, resolver.GetScope(), ty)
	case *ast.BLangUserDefinedType:
		return resolver.GetCtx().FunctionSignatureRef(ty.Symbol())
	default:
		return 0, false
	}
}

func associateReturnFunctionSignature(resolver symbolResolver, source model.FunctionSignatureRef, returnType any, pos diagnostics.Location) bool {
	target, found := functionSignatureRefFromTypeDescriptor(resolver, returnType, pos)
	if !found {
		return true
	}
	if !resolver.GetCtx().AssociateReturnFunctionSignature(source, target) {
		resolver.GetCtx().InternalError("function return signature already set", pos)
		return false
	}
	return true
}

type symbolFunctionSignature interface {
	ast.FunctionSignature
	Symbol() model.SymbolRef
}

// allocateSymbols allocates the function signature of sig. targetScope owns the generated
// default value function symbols, while ownerScope is sig's own scope, which parents the
// closure scopes allocated for its default expressions.
func allocateSymbols(alloc symbolResolver, targetScope, ownerScope model.Scope, sig symbolFunctionSignature, pos diagnostics.Location) bool {
	cx := alloc.GetCtx()
	owner := sig.Symbol()
	if owner.IsEmpty() {
		return true
	}
	if ref, ok := cx.FunctionSignatureRef(owner); ok {
		return associateReturnFunctionSignature(alloc, ref, sig.ReturnType(), pos)
	}
	params := signatureParams(alloc, targetScope, ownerScope, sig)
	ref := cx.AllocateFunctionSignature(params, sig.RestParameter() != nil)
	ok := associateFunctionSignatureRef(cx, owner, ref, pos)
	ok = associateReturnFunctionSignature(alloc, ref, sig.ReturnType(), pos) && ok
	return ok
}

func signatureParams(alloc symbolResolver, targetScope, ownerScope model.Scope, sig ast.FunctionSignature) []model.Param {
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
				defaultParam = &model.DefaultableParam{
					Symbol: symRef,
					Kind:   model.DefaultableParamKindExpr,
					Scope:  alloc.GetCtx().NewFunctionScope(ownerScope, alloc.GetPkgID()),
				}
			}
		}
		params = append(params, model.Param{Name: param.ParamName(), Flag: flag, Default: defaultParam, IncludedRecord: includedRecord})
	}
	if rest := sig.RestParameter(); rest != nil {
		params = append(params, model.Param{Name: rest.ParamName(), Flag: model.ParamFlagRestParam})
	}
	return params
}

func resolveLambdaFunction(functionResolver *blockSymbolResolver, parent *blockSymbolResolver, function *ast.BLangFunction) bool {
	ok := true
	params := function.GetParameters()
	for i := range params {
		param := &params[i]
		name := param.Name.GetValue()
		if isShadowed(parent, name) {
			functionResolver.GetCtx().SemanticError("Variable already defined: "+name, param.GetPosition())
			ok = false
		}
		symbol := model.NewVariableSymbol(name, false, false, true, symbolLocationForNode(param))
		addSymbolAndSetOnNode(functionResolver, name, &symbol, param)
		markInit(functionResolver, name, param.Symbol(), param.GetPosition())
	}

	if function.RestParam != nil {
		restParam := function.RestParam
		name := restParam.Name.GetValue()
		if isShadowed(parent, name) {
			functionResolver.GetCtx().SemanticError("Variable already defined: "+name, restParam.GetPosition())
			ok = false
		}
		symbol := model.NewVariableSymbol(name, false, false, true, symbolLocationForNode(restParam))
		addSymbolAndSetOnNode(functionResolver, name, &symbol, restParam)
		markInit(functionResolver, name, restParam.Symbol(), restParam.GetPosition())
	}

	ok = resolveInvokableChildren(functionResolver, function) && ok
	ok = reportUnusedVariables(functionResolver.GetCtx(), functionResolver.getUnused()) && ok
	return ok
}

func bindImports(
	ctx *context.CompilerContext,
	compilationUnits []*ast.BLangCompilationUnit,
	implicitImports map[string]model.ExportedSymbolSpace,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	moduleVisibility map[PackageIdentifier]ModuleVisibility,
	defaultOrg, currentPackageName string,
) []compilationUnitImportsWithSymbols {
	result := make([]compilationUnitImportsWithSymbols, len(compilationUnits))
	for i, cu := range compilationUnits {
		imports := make(map[string]model.ExportedSymbolSpace)
		for _, imp := range compilationUnitImports(cu) {
			resolveExternalImport(ctx, &imp, defaultOrg, currentPackageName, publicSymbols, moduleVisibility, imports)
		}
		maps.Copy(imports, implicitImports)
		result[i] = compilationUnitImportsWithSymbols{compilationUnit: cu, imports: imports}
	}
	return result
}

// resolveExternalImport looks up the import's exported symbols in publicSymbols
// (populated as each dependency's module is compiled) and binds them to the
// import alias or the last name component. Reports an "Unknown import" error
// when the package was not resolved upstream, or a "not exported" error when
// the resolved module belongs to a different package and isn't in that
// package's exported-modules list — same-package imports are always exempt.
// Java source: org.wso2.ballerinalang.compiler.semantics.analyzer.SymbolEnter
// (the bPackageSymbol.exported check around import-declaration visiting)
func resolveExternalImport(
	ctx *context.CompilerContext,
	imp *ast.BLangImportPackage,
	defaultOrg, currentPackageName string,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	moduleVisibility map[PackageIdentifier]ModuleVisibility,
	result map[string]model.ExportedSymbolSpace,
) bool {
	id := resolveImportPackageIdentifier(imp, defaultOrg)
	symbols, ok := publicSymbols[id]
	if !ok {
		ctx.SemanticError("Unknown import: "+id.OrgName+"/"+id.ModuleName, imp.GetPosition())
		return false
	}
	if vis, ok := moduleVisibility[id]; ok {
		samePackage := vis.PackageOrg == defaultOrg && vis.PackageName == currentPackageName
		if !samePackage && !vis.Exported {
			ctx.SemanticError(
				fmt.Sprintf("%s/%s:%s is not exported", vis.PackageOrg, vis.PackageName, id.ModuleName),
				imp.GetPosition())
			return false
		}
	}
	var key string
	if imp.Alias != nil {
		if imp.Alias.OriginalValue == string(model.IGNORE) {
			return true
		}
		key = imp.Alias.Value
	} else {
		comps := imp.GetPackageName()
		key = comps[len(comps)-1].GetValue()
	}
	result[key] = symbols
	return true
}

type PackageIdentifier struct {
	OrgName    string
	ModuleName string
}

// ModuleVisibility records which package owns a module and whether that
// module is exported, so cross-package imports of non-exported modules can
// be rejected while same-package imports are exempt.
// Java source: io.ballerina.projects.internal.ManifestBuilder /
// org.wso2.ballerinalang.compiler.semantics.analyzer.SymbolEnter (the
// `bPackageSymbol.exported` check around import resolution)
type ModuleVisibility struct {
	PackageOrg  string
	PackageName string
	Exported    bool
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

func resolveFunctionTypeSymbols(r symbolResolver, fnType *ast.BLangFunctionType) bool {
	ensureFunctionTypeSignature(r, r.GetScope(), fnType)
	paramScope := r.GetCtx().NewBlockScope(r.GetScope(), r.GetPkgID())
	paramResolver := &blockSymbolResolver{parent: r, scope: paramScope, node: fnType}
	ok := true
	for _, param := range fnType.RequiredParams {
		if param.TypeDesc != nil {
			ok = resolveTypeDesc(r, param.TypeDesc) && ok
		}
		if param.Name != nil {
			defineFunctionTypeParam(paramScope, param)
			ok = associateFunctionSignatureFromTypeDescriptor(r, param.SymbolRef, param.TypeDesc, param.GetPosition()) && ok
		}
		if param.InitExpr != nil {
			ok = resolveExpr(paramResolver, param.InitExpr) && ok
		}
	}
	if fnType.RestParam != nil {
		param := fnType.RestParam
		if param.TypeDesc != nil {
			ok = resolveTypeDesc(r, param.TypeDesc) && ok
		}
		if param.Name != nil {
			defineFunctionTypeParam(paramScope, param)
			ok = associateFunctionSignatureFromTypeDescriptor(r, param.SymbolRef, param.TypeDesc, param.GetPosition()) && ok
		}
	}
	if fnType.ReturnTypeDescriptor != nil {
		ok = resolveTypeDesc(r, fnType.ReturnTypeDescriptor) && ok
	}
	if ref := fnType.SignatureRef(); ref != 0 {
		ok = associateReturnFunctionSignature(r, ref, fnType.ReturnTypeDescriptor, fnType.GetPosition()) && ok
	}
	return ok
}

func defineFunctionTypeParam(paramScope model.Scope, param *ast.BLangFunctionTypeParam) {
	name := param.Name.GetValue()
	symbol := model.NewVariableSymbol(name, false, false, true, param.Name.GetPosition())
	paramScope.AddSymbol(name, &symbol)
	ref, _ := paramScope.GetSymbol(name)
	param.SymbolRef = ref
	param.Name.SetDeterminedType(semtypes.Never)
}

// since we don't have type information we can't determine if this is an actual method call or need to be converted
// to a function call.
type invocable interface {
	GetName() ast.IdentifierNode
	SetRawSymbol(model.Symbol)
}

func createDeferredMethodSymbol(resolver symbolResolver, n invocable) {
	name := n.GetName().GetValue()
	scope := resolver.GetScope().(model.SymbolSpaceProvider)
	n.SetRawSymbol(common.NewDeferredMethodSymbol(name, scope.MainSpace()))
}

func referUserDefinedType(resolver symbolResolver, n *ast.BLangUserDefinedType) bool {
	name := n.GetTypeName().GetValue()
	var prefix string
	if n.GetPackageAlias() != nil {
		prefix = n.GetPackageAlias().GetValue()
	}
	if !resolveSymbolRef(resolver, name, prefix, n.GetPosition(), n, "Unknown type") {
		return false
	}
	markUnprefixedRefUsed(resolver, name, prefix)
	return true
}

func markUnprefixedRefUsed(resolver symbolResolver, name, prefix string) {
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

func resolveSymbolRef(
	resolver symbolResolver,
	name string,
	prefix string,
	pos diagnostics.Location,
	target symbolRefNode,
	unknownMessage string,
) bool {
	if prefix != "" {
		symRef, ok := resolver.GetPrefixedSymbol(prefix, name)
		if !ok {
			resolver.GetCtx().SemanticError("Unknown symbol: "+name, pos)
			return false
		}
		target.SetSymbol(symRef)
	} else {
		symRef, _, ok := resolver.GetSymbol(name)
		if !ok {
			resolver.GetCtx().SemanticError(unknownMessage+": "+name, pos)
			return false
		}
		target.SetSymbol(symRef)
	}
	return true
}

func resolveAnnotationReference(resolver symbolResolver, pkgAlias, name ast.IdentifierNode, pos diagnostics.Location, target symbolRefNode) bool {
	if name == nil {
		return true
	}
	prefix := ""
	if pkgAlias != nil {
		prefix = pkgAlias.GetValue()
	}
	symRef, ok := resolver.GetAnnotationSymbol(prefix, name.GetValue())
	if !ok {
		resolver.GetCtx().SemanticError("Unknown annotation: "+name.GetValue(), pos)
		return false
	}
	target.SetSymbol(symRef)
	return true
}

func referSimpleVariableReference(resolver symbolResolver, n ast.SimpleVariableReferenceNode) bool {
	name := n.GetVariableName().GetValue()
	var prefix string
	if n.GetPackageAlias() != nil {
		prefix = n.GetPackageAlias().GetValue()
	}
	ok := resolveSymbolRef(resolver, name, prefix, n.GetPosition(), n.(ast.BNodeWithSymbol), "Unknown symbol")
	markUnprefixedRefUsed(resolver, name, prefix)
	return ok
}

func resolveFunctionRef(resolver symbolResolver, invocation *ast.BLangInvocation) bool {
	name := invocation.GetName().GetValue()
	prefix := invocation.GetPackageAlias().GetValue()
	ok := resolveSymbolRef(resolver, name, prefix, invocation.GetPosition(), invocation, "Unknown symbol")
	markUnprefixedRefUsed(resolver, name, prefix)
	return ok
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

func defineVariable(resolver *blockSymbolResolver, variable *ast.BLangVariable, isFinal bool) bool {
	name := variable.Name.GetValue()
	ok := true
	if isShadowed(resolver, name) {
		resolver.GetCtx().SemanticError("Variable already defined: "+name, variable.GetPosition())
		ok = false
	}
	symbol := model.NewVariableSymbol(name, false, false, false, symbolLocationForNode(variable))
	if isFinal {
		symbol.SetFinal()
	}
	addSymbolAndSetOnNode(resolver, name, &symbol, variable)
	markInit(resolver, name, variable.Symbol(), variable.GetPosition())
	return ok
}

func resolveForeach(bs *blockSymbolResolver, n *ast.BLangForeach) bool {
	resolver := newBlockSymbolResolverWithBlockScope(bs, n)
	n.SetScope(resolver.scope)
	ok := true
	if n.Collection != nil {
		ok = resolveExpr(resolver, n.Collection) && ok
	}
	if n.VariableDef != nil {
		variable := n.VariableDef.GetVariable()
		ok = defineVariable(resolver, variable, true) && ok
		ok = resolveVariableChildren(resolver, variable, variable.Symbol()) && ok
	}
	ok = resolveBlockStmt(resolver, &n.Body) && ok
	if n.OnFailClause != nil {
		ok = resolveOnFailClause(resolver, n.OnFailClause) && ok
	}
	return ok
}

type inclusionMemberForSymbolResolution struct {
	name     string
	isPublic bool
}

// resolveObjectInclusions update the AST node references with correct symbol references. Will add semantic errors if the type
// reference is for something that can't be included. This means after this stage we have the gurantee symbol ref always refer
// to a valid AST node.
func resolveObjectInclusions(resolver symbolResolver, unresolvedInclusions []*ast.BLangUserDefinedType) ([]model.SymbolRef, []diagnostics.Location, []inclusionMemberForSymbolResolution, bool) {
	ctx := resolver.GetCtx()
	localTypeDefns := resolver.GetTypeDefns()
	localClassDefns := resolver.GetClassDefns()
	inclusions := make([]model.SymbolRef, 0, len(unresolvedInclusions))
	positions := make([]diagnostics.Location, 0, len(unresolvedInclusions))
	var includedFields []inclusionMemberForSymbolResolution
	ok := true
	for _, inc := range unresolvedInclusions {
		if !referUserDefinedType(resolver, inc) {
			ok = false
			continue
		}
		symRef := inc.Symbol()
		if tDefn, isLocalType := localTypeDefns[symRef]; isLocalType {
			if _, isObjectDescriptor := tDefn.GetTypeData().TypeDescriptor.(*ast.BLangObjectType); !isObjectDescriptor {
				if _, isObjectType := ctx.GetSymbol(symRef).(*model.ObjectTypeSymbol); !isObjectType {
					ctx.SemanticError("type inclusion must be an object type or class", inc.GetPosition())
					ok = false
					continue
				}
			}
			includedFields = append(includedFields, collectTransitiveFieldsFromTypeDefn(ctx, tDefn, localTypeDefns, localClassDefns)...)
		} else if classDefn, isLocalClass := localClassDefns[symRef]; isLocalClass {
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
					ok = false
					continue
				}
				carrier = s
			default:
				ctx.SemanticError("type inclusion must be an object type or class", inc.GetPosition())
				ok = false
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
	return inclusions, positions, includedFields, ok
}

// allocateRecordDefaultSymbols allocates a module unique function symbol and the scope owning the
// closure desugar generates for each record field with a default expression. Since the field type is
// not known at symbol resolution the typed signature is left empty and populated during type
// resolution.
func allocateRecordDefaultSymbols(resolver symbolResolver, recordType *ast.BLangRecordType) {
	scope := resolver.recordDefaultScope()
	for _, field := range recordType.FieldPtrs() {
		if field.Default == nil {
			continue
		}
		name := resolver.nextDefaultSymbolName()
		symbol := model.NewFunctionSymbol(name, model.TypedFunctionSignature{}, false, field.GetPosition())
		scope.AddSymbol(name, symbol)
		symRef, _ := scope.GetSymbol(name)
		field.Default.FnRef = symRef
		field.Default.FnScope = resolver.GetCtx().NewFunctionScope(scope, resolver.GetPkgID())
	}
}

func resolveRecordTypeInclusions(resolver symbolResolver, typeInclusions []ast.BType) ([]model.SymbolRef, []diagnostics.Location, bool) {
	ctx := resolver.GetCtx()
	localTypeDefns := resolver.GetTypeDefns()
	var inclusions []model.SymbolRef
	var positions []diagnostics.Location
	ok := true
	for _, inc := range typeInclusions {
		udt, isUserDefined := inc.(*ast.BLangUserDefinedType)
		if !isUserDefined {
			ctx.SemanticError("type inclusion must be a user-defined type", inc.(ast.BLangNode).GetPosition())
			ok = false
			continue
		}
		if !referUserDefinedType(resolver, udt) {
			ok = false
			continue
		}
		symRef := udt.Symbol()
		if tDefn, isLocal := localTypeDefns[symRef]; isLocal {
			if _, isRecord := tDefn.GetTypeData().TypeDescriptor.(*ast.BLangRecordType); !isRecord {
				ctx.SemanticError("included type is not a record type", udt.GetPosition())
				ok = false
				continue
			}
		} else {
			sym := ctx.GetSymbol(symRef)
			if _, isRecord := sym.(*model.RecordSymbol); !isRecord {
				ctx.SemanticError("included type is not a record type", udt.GetPosition())
				ok = false
				continue
			}
			incTy := ctx.SymbolType(symRef)
			if semtypes.IsZero(incTy) || !semtypes.IsSubtype(resolver.TypeContext(), incTy, semtypes.Mapping) {
				ctx.SemanticError("included type is not a record type", udt.GetPosition())
				ok = false
				continue
			}
		}
		inclusions = append(inclusions, symRef)
		positions = append(positions, udt.GetPosition())
	}
	return inclusions, positions, ok
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

func resolveServiceDefinition(ms *compilationUnitSymbolResolver, svc *ast.BLangService) bool {
	ok := true
	if typeDescriptor := svc.GetTypeData().TypeDescriptor; typeDescriptor != nil {
		ok = resolveTypeDesc(ms, typeDescriptor) && ok
	}

	for _, expr := range svc.AttachedExprs {
		ok = resolveExpr(ms, expr) && ok
	}

	if svc.InitFunction != nil && len(svc.InitFunction.RequiredParams) > 0 {
		ms.GetCtx().SemanticError("service 'init' must not declare required parameters", svc.InitFunction.RequiredParams[0].GetPosition())
		ok = false
	}

	svcResolver := newBlockSymbolResolverWithBlockScope(ms, svc)
	svc.SetScope(svcResolver.scope)

	ok = allocateServiceResourceMethodSymbols(ms, svcResolver, svc.ResourceMethods) && ok

	ok = finishResolveClassDefinition(ms, svcResolver, svc.Fields, svc.Methods, svc.ResourceMethods, svc.InitFunction, nil, svcResolver.scope, "") && ok

	serviceSymbolName := ms.moduleResolver.nextServiceSymbolName()
	serviceSymbol := model.NewTypeSymbol(serviceSymbolName, false, svc.GetPosition())
	svcResolver.AddSymbol(serviceSymbolName, &serviceSymbol)
	serviceSymbolRef, _, _ := svcResolver.GetSymbol(serviceSymbolName)
	svc.SetSymbol(serviceSymbolRef)
	for i := range svc.AnnAttachments {
		ok = resolveAnnotationAttachment(svcResolver, &svc.AnnAttachments[i]) && ok
	}
	return ok
}

func resolveClassDefinition(ms *compilationUnitSymbolResolver, classDef *ast.BLangClassDefinition) bool {
	className := classDef.Name.GetValue()
	ok := true
	methodSymbolPrefix := className + "."
	owner := classDef.Symbol()

	classResolver := newBlockSymbolResolverWithBlockScope(ms, classDef)
	classDef.SetScope(classResolver.scope)
	for i := range classDef.AnnAttachments {
		ok = resolveAnnotationAttachment(classResolver, &classDef.AnnAttachments[i]) && ok
	}

	inclusions, positions, includedFields, inclusionsOk := resolveObjectInclusions(ms, classDef.PopUnresolvedInclusions())
	ok = inclusionsOk && ok
	classDef.Inclusions, classDef.InclusionPositions = inclusions, positions
	var methodTargetScope methodSymbolTargetScope
	if !owner.IsEmpty() {
		networkClassSym, _ := ms.moduleResolver.ctx.GetSymbol(owner).(*model.NetworkClassSymbol)
		ok = allocateObjectResourceMethodSymbols(ms, classResolver, classDef, networkClassSym) && ok
		methodTargetScope = ms.scope
	} else if !classDef.IsClient() && !classDef.IsService() {
		ok = allocateObjectResourceMethodSymbols(ms, classResolver, classDef, nil) && ok
	}
	return finishResolveClassDefinition(ms, classResolver, classDef.Fields, classDef.Methods, classDef.ResourceMethods, classDef.InitFunction, includedFields, methodTargetScope, methodSymbolPrefix) && ok
}

func finishResolveClassDefinition(ms *compilationUnitSymbolResolver, blockRes *blockSymbolResolver, fields []*ast.BLangVariable, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod, initFn *ast.BLangFunction, includedFields []inclusionMemberForSymbolResolution, methodTargetScope methodSymbolTargetScope, methodSymbolPrefix string) bool {
	ok := true
	for _, field := range fields {
		name := field.GetName().GetValue()
		if _, sk, exists := blockRes.GetSymbol(name); exists && sk == blockScopeKind {
			blockRes.GetCtx().SemanticError("redeclared symbol '"+name+"'", field.GetPosition())
			ok = false
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
			blockRes.GetCtx().SemanticError("redeclared symbol '"+model.StripRemotePrefix(m.Name)+"'", m.Method.Name.GetPosition())
			ok = false
			continue
		}
		ok = allocateMethodSymbol(ms, m.Method, m.Name, m.Method.IsPublic(), methodTargetScope, methodSymbolPrefix) && ok
	}

	for _, m := range includedFields {
		if _, _, exists := blockRes.GetSymbol(m.name); exists {
			continue
		}
		symbol := model.NewVariableSymbol(m.name, m.isPublic, false, false, diagnostics.NewBuiltinLocation())
		blockRes.AddSymbol(m.name, &symbol)
	}

	if initFn != nil {
		ok = allocateMethodSymbol(ms, initFn, "init", initFn.IsPublic(), methodTargetScope, methodSymbolPrefix) && ok
	}

	selfSymbol := model.NewVariableSymbol("self", false, false, false, diagnostics.NewBuiltinLocation())
	blockRes.AddSymbol("self", &selfSymbol)

	for _, field := range fields {
		ok = resolveVariableChildren(blockRes, field, field.Symbol()) && ok
	}

	if initFn != nil {
		initResolver := newFunctionResolver(blockRes, initFn)
		initFn.SetScope(initResolver.scope)
		ok = resolveFunction(initResolver, initFn) && ok
		ok = allocateSymbols(ms, ms.scope, initResolver.scope, initFn, initFn.GetPosition()) && ok
	}

	for _, m := range orderedMethods {
		methodResolver := newFunctionResolver(blockRes, m.Method)
		m.Method.SetScope(methodResolver.scope)
		ok = resolveFunction(methodResolver, m.Method) && ok
		ok = allocateSymbols(ms, ms.scope, methodResolver.scope, m.Method, m.Method.GetPosition()) && ok
	}

	for _, rm := range resourceMethods {
		methodResolver := newFunctionResolver(blockRes, rm)
		rm.SetScope(methodResolver.scope)
		ok = resolveResourceMethod(methodResolver, rm) && ok
		ok = allocateSymbols(ms, ms.scope, methodResolver.scope, rm, rm.GetPosition()) && ok
	}
	return ok
}

// allocateMethodSymbol builds the method's symbol, which reports its declaration errors, and registers it in
// methodTargetScope. A nil methodTargetScope means the class has no owner symbol, so nothing is registered.
func allocateMethodSymbol(ms *compilationUnitSymbolResolver, method *ast.BLangFunction, name string, isPublic bool, methodTargetScope methodSymbolTargetScope, methodSymbolPrefix string) bool {
	symbol, ok := ms.allocateFunctionSymbolInner(method, name, isPublic)
	if methodTargetScope == nil {
		return ok
	}
	symbolName := methodSymbolPrefix + name
	methodTargetScope.AddSymbol(symbolName, symbol)
	symRef, _ := methodTargetScope.MainSpace().GetSymbol(symbolName)
	method.SetSymbol(symRef)
	return ok
}

type methodSymbolTargetScope interface {
	model.Scope
	MainSpace() *model.SymbolSpace
}

func allocateObjectResourceMethodSymbols(ms *compilationUnitSymbolResolver, blockRes *blockSymbolResolver, classDef *ast.BLangClassDefinition, networkClassSym *model.NetworkClassSymbol) bool {
	className := classDef.Name.GetValue()
	ok := true
	for idx, rm := range classDef.ResourceMethods {
		if networkClassSym == nil {
			blockRes.GetCtx().SemanticError("resource methods are only allowed in client or service classes", rm.GetPosition())
			ok = false
			continue
		}
		mangledName := className + "." + mangledResourceMethodName(rm.Name.GetValue(), idx)
		symRef, symbolOk := ms.allocateResourceMethodSymbol(ms.scope, rm, mangledName, classDef.IsPublic() && rm.IsPublic())
		ok = symbolOk && ok
		networkClassSym.AddResourceMethod(symRef)
	}
	return ok
}

func allocateServiceResourceMethodSymbols(ms *compilationUnitSymbolResolver, blockRes *blockSymbolResolver, resourceMethods []*ast.BLangResourceMethod) bool {
	ok := true
	for idx, rm := range resourceMethods {
		key := mangledResourceMethodName(rm.Name.GetValue(), idx)
		_, symbolOk := ms.allocateResourceMethodSymbol(blockRes.scope, rm, key, rm.IsPublic())
		ok = symbolOk && ok
	}
	return ok
}

func (ms *compilationUnitSymbolResolver) allocateResourceMethodSymbol(targetScope methodSymbolTargetScope, rm *ast.BLangResourceMethod, symbolName string, isPublic bool) (model.SymbolRef, bool) {
	var symbol model.ResourceMethodSymbol
	ok := true
	if ms.isDependentlyTyped(rm) {
		if rm.GetRestParam() != nil {
			ms.moduleResolver.ctx.Unimplemented("rest parameters are not supported on dependently-typed functions", rm.GetPosition())
			ok = false
		}
		if _, isExtern := rm.GetBody().(*ast.BLangExternFunctionBody); !isExtern {
			ms.moduleResolver.ctx.SemanticError("dependently typed function must be external", rm.GetPosition())
			ok = false
		}
		symbol = model.NewDependentlyTypedResourceMethodSymbol(symbolName, rm.Name.GetValue(), rm.FuncSymbolFlags(), isPublic, symbolLocationForNode(rm))
	} else {
		symbol = model.NewResourceMethodSymbol(symbolName, rm.Name.GetValue(), isPublic, symbolLocationForNode(rm))
	}
	targetScope.AddSymbol(symbolName, symbol)
	symRef, _ := targetScope.MainSpace().GetSymbol(symbolName)
	rm.SetSymbol(symRef)
	return symRef, ok
}

func mangledResourceMethodName(methodName string, idx int) string {
	return fmt.Sprintf("$resource$%s$%d", methodName, idx)
}

func resolveResourceMethod(functionResolver *blockSymbolResolver, rm *ast.BLangResourceMethod) bool {
	// Limit collision detection to the current function scope, matching
	// resolveFunctionInner. functionResolver.GetSymbol would otherwise delegate
	// into the enclosing class scope (also a blockSymbolResolver) and wrongly
	// reject path params that shadow a class field.
	scope := functionResolver.scope.MainSpace()
	ok := true
	for i := range rm.ResourcePath {
		seg := &rm.ResourcePath[i]
		if seg.Kind == ast.ResourcePathSegmentName || seg.Name == "" {
			continue
		}
		name := seg.Name
		if _, exists := scope.GetSymbol(name); exists {
			functionResolver.GetCtx().SemanticError("redeclared symbol '"+name+"'", seg.GetPosition())
			ok = false
			continue
		}
		symbol := model.NewVariableSymbol(name, false, false, true, seg.GetPosition())
		functionResolver.AddSymbol(name, &symbol)
	}
	return resolveFunctionInner(functionResolver, rm.GetParameters(), rm.RestParam, rm, rm.Body) && ok
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

func resolveSelfFieldAccess(resolver symbolResolver, n *ast.BLangFieldBaseAccess, classScope model.BlockLevelScope) bool {
	varRef := n.Expr.(*ast.BLangVarRef)
	ok := referSimpleVariableReference(resolver, varRef)
	fieldName := n.Field.GetValue()
	if _, found := classScope.MainSpace().GetSymbol(fieldName); !found {
		resolver.GetCtx().SemanticError("undefined member '"+fieldName+"'", n.Field.GetPosition())
		ok = false
	}
	return ok
}

func reportUnexpectedNode(r symbolResolver, node ast.Node) bool {
	r.GetCtx().InternalError(fmt.Sprintf("unexpected node kind in symbol resolution: %T", node), node.GetPosition())
	return false
}

func resolveCompilationUnit(r *compilationUnitSymbolResolver, cu *ast.BLangCompilationUnit) {
	for _, node := range cu.TopLevelNodes {
		resolveTopLevelNode(r, node)
	}
}

func resolveTopLevelNode(r *compilationUnitSymbolResolver, node ast.TopLevelNode) bool {
	switch n := node.(type) {
	case ast.BLangBadNode:
		return false
	case *ast.BLangImportPackage:
		return true
	case *ast.BLangXMLNS:
		uri := n.GetNamespaceURI()
		if uri == nil {
			return false
		}
		return resolveExpr(r, uri)
	case *ast.BLangFunction:
		return resolveModuleFunction(r, n)
	case *ast.BLangVariable:
		return resolveModuleVariable(r, n)
	case *ast.BLangTypeDefinition:
		return resolveTypeDefinition(r, n)
	case *ast.BLangAnnotation:
		return resolveAnnotation(r, n)
	case *ast.BLangClassDefinition:
		return resolveClassDefinition(r, n)
	case *ast.BLangService:
		return resolveServiceDefinition(r, n)
	default:
		return reportUnexpectedNode(r, node)
	}
}

func resolveModuleFunction(r *compilationUnitSymbolResolver, fn *ast.BLangFunction) bool {
	functionResolver := newFunctionResolver(r, fn)
	fn.SetScope(functionResolver.scope)
	ok := resolveFunction(functionResolver, fn)
	return allocateSymbols(r, r.scope, functionResolver.scope, fn, fn.GetPosition()) && ok
}

func resolveModuleVariable(r *compilationUnitSymbolResolver, n *ast.BLangVariable) bool {
	return resolveVariableChildren(r, n, n.Symbol())
}

func resolveTypeDefinition(r *compilationUnitSymbolResolver, n *ast.BLangTypeDefinition) bool {
	ok := true
	typeDescriptor := n.GetTypeData().TypeDescriptor
	if typeDescriptor != nil {
		ok = resolveTypeDesc(r, typeDescriptor) && ok
	}
	attachments := n.GetAnnotationAttachments()
	for i := range attachments {
		ok = resolveAnnotationAttachment(r, &attachments[i]) && ok
	}
	if typeDescriptor != nil {
		ok = associateFunctionSignatureFromTypeDescriptor(r, n.Symbol(), typeDescriptor, n.GetPosition()) && ok
	}
	return ok
}

func resolveAnnotation(r *compilationUnitSymbolResolver, n *ast.BLangAnnotation) bool {
	ok := true
	attachments := n.GetAnnotationAttachments()
	for i := range attachments {
		ok = resolveAnnotationAttachment(r, &attachments[i]) && ok
	}
	if typeDescriptor := n.GetTypeDescriptor(); typeDescriptor != nil {
		ok = resolveTypeDesc(r, typeDescriptor) && ok
	}
	return ok
}

func resolveVariableChildren(r symbolResolver, variable *ast.BLangVariable, owner model.SymbolRef) bool {
	ok := true
	for i := range variable.AnnAttachments {
		ok = resolveAnnotationAttachment(r, &variable.AnnAttachments[i]) && ok
	}
	if typeNode := variable.TypeNode(); typeNode != nil {
		ok = resolveTypeDesc(r, typeNode) && ok
		ok = associateFunctionSignatureFromTypeDescriptor(r, owner, typeNode, variable.GetPosition()) && ok
	}
	if variable.Expr != nil {
		ok = resolveExpr(r, variable.Expr) && ok
	}
	return ok
}

func resolveAnnotationAttachment(r symbolResolver, a *ast.BLangAnnotationAttachment) bool {
	ok := resolveAnnotationReference(r, a.GetPackageAlias(), a.GetAnnotationName(), a.GetPosition(), a)
	if a.Expr != nil {
		ok = resolveExpr(r, a.Expr) && ok
	}
	return ok
}

func resolveStmts(r *blockSymbolResolver, stmts []ast.StatementNode) bool {
	ok := true
	for _, stmt := range stmts {
		ok = resolveStmt(r, stmt) && ok
	}
	return ok
}

func resolveStmt(r *blockSymbolResolver, stmt ast.StatementNode) bool {
	switch n := stmt.(type) {
	case ast.BLangBadNode:
		return false
	case *ast.BLangXMLNS:
		return resolveLocalXMLNS(r, n)
	case *ast.BLangBlockStmt:
		return resolveBlockStmt(r, n)
	case *ast.BLangAssignment:
		return resolveAssignment(r, n)
	case *ast.BLangCompoundAssignment:
		return resolveCompoundAssignment(r, n)
	case *ast.BLangExpressionStmt:
		return resolveExpressionStmt(r, n)
	case *ast.BLangIf:
		return resolveIf(r, n)
	case *ast.BLangWhile:
		return resolveWhile(r, n)
	case *ast.BLangForeach:
		return resolveForeach(r, n)
	case *ast.BLangDo:
		return resolveDo(r, n)
	case *ast.BLangLock:
		return resolveLock(r, n)
	case *ast.BLangMatchStatement:
		return resolveMatchStatement(r, n)
	case *ast.BLangVariableDef:
		return resolveVariableDef(r, n)
	case *ast.BLangReturn:
		return resolveReturn(r, n)
	case *ast.BLangPanic:
		return resolvePanic(r, n)
	case *ast.BLangOnFailClause:
		return resolveOnFailClause(r, n)
	case *ast.BLangBreak, *ast.BLangContinue:
		return true
	default:
		return reportUnexpectedNode(r, stmt)
	}
}

func resolveLocalXMLNS(r *blockSymbolResolver, n *ast.BLangXMLNS) bool {
	ok := processBlockXMLNS(r, n)
	if uri := n.GetNamespaceURI(); uri != nil {
		ok = resolveExpr(r, uri) && ok
	}
	return ok
}

func resolveBlockStmt(r *blockSymbolResolver, n *ast.BLangBlockStmt) bool {
	return resolveStmts(newBlockSymbolResolverWithBlockScope(r, n), n.Stmts)
}

func resolveAssignment(r *blockSymbolResolver, n *ast.BLangAssignment) bool {
	ok := true
	if n.VarRef != nil {
		ok = resolveLExpr(r, n.VarRef) && ok
	}
	if n.Expr != nil {
		ok = resolveExpr(r, n.Expr) && ok
	}
	return ok
}

func resolveCompoundAssignment(r *blockSymbolResolver, n *ast.BLangCompoundAssignment) bool {
	ok := true
	if n.VarRef != nil {
		ok = resolveLExpr(r, n.VarRef) && ok
	}
	if n.Expr != nil {
		ok = resolveExpr(r, n.Expr) && ok
	}
	return ok
}

func resolveLExpr(r *blockSymbolResolver, lexpr ast.LExpr) bool {
	if bp, isBindingPattern := lexpr.(ast.BindingPatternNode); isBindingPattern {
		return resolveBindingPattern(r, bp)
	}
	return resolveExpr(r, lexpr)
}

func resolveBindingPattern(r *blockSymbolResolver, bp ast.BindingPatternNode) bool {
	switch n := bp.(type) {
	case *ast.BLangCaptureBindingPattern, *ast.BLangWildCardBindingPattern, *ast.BLangRestBindingPattern:
		return true
	case *ast.BLangSimpleBindingPattern:
		ok := true
		if n.CaptureBindingPattern != nil {
			ok = resolveBindingPattern(r, n.CaptureBindingPattern) && ok
		}
		if n.WildCardBindingPattern != nil {
			ok = resolveBindingPattern(r, n.WildCardBindingPattern) && ok
		}
		return ok
	case *ast.BLangErrorBindingPattern:
		ok := true
		if n.ErrorTypeReference != nil {
			ok = resolveUserDefinedType(r, n.ErrorTypeReference) && ok
		}
		if n.ErrorMessageBindingPattern != nil {
			ok = resolveBindingPattern(r, n.ErrorMessageBindingPattern) && ok
		}
		if n.ErrorCauseBindingPattern != nil {
			ok = resolveBindingPattern(r, n.ErrorCauseBindingPattern) && ok
		}
		if n.ErrorFieldBindingPatterns != nil {
			ok = resolveBindingPattern(r, n.ErrorFieldBindingPatterns) && ok
		}
		return ok
	case *ast.BLangErrorMessageBindingPattern:
		if n.SimpleBindingPattern == nil {
			return true
		}
		return resolveBindingPattern(r, n.SimpleBindingPattern)
	case *ast.BLangErrorCauseBindingPattern:
		ok := true
		if n.SimpleBindingPattern != nil {
			ok = resolveBindingPattern(r, n.SimpleBindingPattern) && ok
		}
		if n.ErrorBindingPattern != nil {
			ok = resolveBindingPattern(r, n.ErrorBindingPattern) && ok
		}
		return ok
	case *ast.BLangErrorFieldBindingPatterns:
		ok := true
		for i := range n.NamedArgBindingPatterns {
			ok = resolveBindingPattern(r, &n.NamedArgBindingPatterns[i]) && ok
		}
		if n.RestBindingPattern != nil {
			ok = resolveBindingPattern(r, n.RestBindingPattern) && ok
		}
		return ok
	case *ast.BLangNamedArgBindingPattern:
		if n.BindingPattern == nil {
			return true
		}
		return resolveBindingPattern(r, n.BindingPattern)
	default:
		return reportUnexpectedNode(r, bp)
	}
}

func resolveExpressionStmt(r *blockSymbolResolver, n *ast.BLangExpressionStmt) bool {
	return resolveOptionalExpr(r, n.Expr)
}

func resolveIf(r *blockSymbolResolver, n *ast.BLangIf) bool {
	resolver := newBlockSymbolResolverWithBlockScope(r, n)
	n.SetScope(resolver.scope)
	ok := true
	if n.Expr != nil {
		ok = resolveExpr(resolver, n.Expr) && ok
	}
	ok = resolveBlockStmt(resolver, &n.Body) && ok
	if n.ElseStmt != nil {
		ok = resolveStmt(resolver, n.ElseStmt) && ok
	}
	return ok
}

func resolveWhile(r *blockSymbolResolver, n *ast.BLangWhile) bool {
	resolver := newBlockSymbolResolverWithBlockScope(r, n)
	n.SetScope(resolver.scope)
	ok := true
	if n.Expr != nil {
		ok = resolveExpr(resolver, n.Expr) && ok
	}
	ok = resolveBlockStmt(resolver, &n.Body) && ok
	return resolveOnFailClause(resolver, &n.OnFailClause) && ok
}

func resolveDo(r *blockSymbolResolver, n *ast.BLangDo) bool {
	resolver := newBlockSymbolResolverWithBlockScope(r, n)
	ok := resolveBlockStmt(resolver, &n.Body)
	return resolveOnFailClause(resolver, &n.OnFailClause) && ok
}

func resolveLock(r *blockSymbolResolver, n *ast.BLangLock) bool {
	return resolveBlockStmt(newBlockSymbolResolverWithBlockScope(r, n), &n.Body)
}

func resolveOnFailClause(r *blockSymbolResolver, n *ast.BLangOnFailClause) bool {
	ok := true
	if n.Body != nil {
		ok = resolveBlockStmt(r, n.Body) && ok
	}
	if n.VariableDefinitionNode != nil {
		ok = resolveVariableDef(r, n.VariableDefinitionNode) && ok
	}
	return ok
}

func resolveMatchStatement(r *blockSymbolResolver, n *ast.BLangMatchStatement) bool {
	ok := true
	if n.Expr != nil {
		ok = resolveExpr(r, n.Expr) && ok
	}
	for i := range n.MatchClauses {
		ok = resolveMatchClause(r, &n.MatchClauses[i]) && ok
	}
	return ok
}

func resolveMatchClause(r *blockSymbolResolver, n *ast.BLangMatchClause) bool {
	ok := true
	for _, pattern := range n.Patterns {
		ok = resolveMatchPattern(r, pattern) && ok
	}
	if n.Guard != nil {
		ok = resolveExpr(r, n.Guard) && ok
	}
	return resolveBlockStmt(r, &n.Body) && ok
}

func resolveMatchPattern(r *blockSymbolResolver, mp ast.MatchPatternNode) bool {
	switch n := mp.(type) {
	case ast.BLangBadNode:
		return false
	case *ast.BLangConstPattern:
		if n.Expr == nil {
			return true
		}
		return resolveExpr(r, n.Expr)
	case *ast.BLangWildCardMatchPattern:
		return true
	default:
		return reportUnexpectedNode(r, mp)
	}
}

func resolveVariableDef(r *blockSymbolResolver, n *ast.BLangVariableDef) bool {
	variable := n.GetVariable()
	ok := defineVariable(r, variable, variable.IsFinal())
	return resolveVariableChildren(r, variable, variable.Symbol()) && ok
}

func resolveReturn(r *blockSymbolResolver, n *ast.BLangReturn) bool {
	return resolveOptionalExpr(r, n.Expr)
}

func resolvePanic(r *blockSymbolResolver, n *ast.BLangPanic) bool {
	return resolveOptionalExpr(r, n.Expr)
}

func resolveQueryClause(r *blockSymbolResolver, clause ast.BLangNode) bool {
	switch n := clause.(type) {
	case ast.BLangBadNode:
		return false
	case *ast.BLangFromClause:
		return resolveInputClause(r, &n.BLangInputClause)
	case *ast.BLangJoinClause:
		ok := true
		if n.Collection != nil {
			ok = resolveExpr(r, n.Collection) && ok
		}
		if n.OnClause.OnExpr != nil {
			ok = resolveExpr(r, n.OnClause.OnExpr) && ok
		}
		if n.VariableDefinitionNode != nil {
			ok = resolveVariableDef(r, n.VariableDefinitionNode) && ok
		}
		if n.OnClause.EqualsExpr != nil {
			ok = resolveExpr(r, n.OnClause.EqualsExpr) && ok
		}
		return ok
	case *ast.BLangLetClause:
		ok := true
		for i := range n.LetVarDeclarations {
			ok = resolveVariableDef(r, &n.LetVarDeclarations[i]) && ok
		}
		return ok
	case *ast.BLangGroupByClause:
		ok := true
		for _, groupingKey := range n.GetGroupingKeyList() {
			ok = resolveQueryClause(r, groupingKey) && ok
		}
		return ok
	case *ast.BLangGroupingKey:
		if n.VariableRef != nil {
			return resolveExpr(r, n.VariableRef)
		}
		if n.VariableDef != nil {
			return resolveVariableDef(r, n.VariableDef)
		}
		return true
	case *ast.BLangOnClause:
		ok := true
		if n.OnExpr != nil {
			ok = resolveExpr(r, n.OnExpr) && ok
		}
		if n.EqualsExpr != nil {
			ok = resolveExpr(r, n.EqualsExpr) && ok
		}
		return ok
	case *ast.BLangOrderByClause:
		ok := true
		for i := range n.OrderByKeyList {
			ok = resolveQueryClause(r, &n.OrderByKeyList[i]) && ok
		}
		return ok
	case *ast.BLangSelectClause:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangWhereClause:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangLimitClause:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangOrderKey:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangOnConflictClause:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangCollectClause:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangOnFailClause:
		return resolveOnFailClause(r, n)
	case *ast.BLangDoClause:
		if n.Body == nil {
			return true
		}
		return resolveBlockStmt(r, n.Body)
	default:
		return reportUnexpectedNode(r, clause)
	}
}

func resolveInputClause(r *blockSymbolResolver, n *ast.BLangInputClause) bool {
	ok := true
	if n.Collection != nil {
		ok = resolveExpr(r, n.Collection) && ok
	}
	if n.VariableDefinitionNode != nil {
		ok = resolveVariableDef(r, n.VariableDefinitionNode) && ok
	}
	return ok
}

func resolveOptionalExpr(r symbolResolver, expr ast.BLangActionOrExpression) bool {
	if expr == nil {
		return true
	}
	return resolveExpr(r, expr)
}

func resolveExprs(r symbolResolver, exprs []ast.BLangExpression) bool {
	ok := true
	for _, expr := range exprs {
		ok = resolveExpr(r, expr) && ok
	}
	return ok
}

func resolveExpr(r symbolResolver, expr ast.BLangActionOrExpression) bool {
	switch n := expr.(type) {
	case ast.BLangBadNode:
		return false
	case *ast.BLangBinaryExpr:
		ok := resolveOptionalExpr(r, n.LhsExpr)
		return resolveOptionalExpr(r, n.RhsExpr) && ok
	case *ast.BLangTernaryExpr:
		ok := resolveOptionalExpr(r, n.Condition)
		ok = resolveOptionalExpr(r, n.ThenExpr) && ok
		return resolveOptionalExpr(r, n.ElseExpr) && ok
	case *ast.BLangQueryExpr:
		return resolveQueryExpr(r, n)
	case *ast.BLangUnaryExpr:
		return resolveOptionalExpr(r, n.Expr)
	case *ast.BLangNilConditionalExpr:
		ok := resolveOptionalExpr(r, n.LhsExpr)
		return resolveOptionalExpr(r, n.RhsExpr) && ok
	case *ast.BLangCheckedExpr:
		return resolveOptionalExpr(r, n.Expr)
	case *ast.BLangCheckPanickedExpr:
		return resolveOptionalExpr(r, n.Expr)
	case *ast.BLangTrapExpr:
		return resolveOptionalExpr(r, n.Expr)
	case *ast.BLangGroupExpr:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangIndexBasedAccess:
		ok := resolveOptionalExpr(r, n.Expr)
		return resolveOptionalExpr(r, n.IndexExpr) && ok
	case *ast.BLangFieldBaseAccess:
		return resolveFieldBaseAccess(r, n)
	case *ast.BLangListConstructorExpr:
		return resolveExprs(r, n.Exprs)
	case *ast.BLangMappingConstructorExpr:
		return resolveMappingConstructorExpr(r, n)
	case *ast.BLangErrorConstructorExpr:
		return resolveErrorConstructorExpr(r, n)
	case *ast.BLangInvocation:
		return resolveInvocation(r, n)
	case *ast.BLangLambdaFunction:
		return resolveLambda(r, n)
	case *ast.BLangAnnotAccessExpr:
		ok := resolveAnnotationReference(r, n.PkgAlias, n.AnnotationName, n.GetPosition(), n)
		return resolveOptionalExpr(r, n.Expr) && ok
	case *ast.BLangTypedescExpr:
		if typeDescriptor := n.GetTypeDescriptor(); typeDescriptor != nil {
			return resolveTypeDesc(r, typeDescriptor)
		}
		return true
	case *ast.BLangInferredTypedescDefault, *ast.BLangDefaultArg:
		return true
	case *ast.BLangTypeConversionExpr:
		ok := resolveOptionalExpr(r, n.Expression)
		if n.TypeDescriptor != nil {
			ok = resolveTypeDesc(r, n.TypeDescriptor) && ok
		}
		return ok
	case *ast.BLangTypeTestExpr:
		ok := resolveOptionalExpr(r, n.Expr)
		return resolveTypeData(r, &n.Type) && ok
	case *ast.BLangVarRef:
		return referSimpleVariableReference(r, n)
	case *ast.BLangConstRef:
		return referSimpleVariableReference(r, n)
	case *ast.BLangLiteral, *ast.BLangNumericLiteral:
		return true
	case *ast.BLangXMLSequenceLiteral:
		return resolveExprs(r, n.Children)
	case *ast.BLangTemplateExpr:
		return resolveExprs(r, n.Insertions)
	case *ast.BLangXMLTemplateExpr:
		ok := resolveXMLTemplateNamespaces(r, r.GetScope(), n)
		return resolveExprs(r, n.Insertions) && ok
	case *ast.BLangXMLElementLiteral:
		rootNeeds := map[string]model.SymbolRef{}
		ok := resolveXMLElementLiteralNamespaces(r, r.GetScope(), n, rootNeeds)
		return mergeNamespaces(r, n, rootNeeds) && ok
	case *ast.BLangXMLAttribute:
		return resolveOptionalExpr(r, n.Value)
	case *ast.BLangXMLPILiteral, *ast.BLangXMLCommentLiteral, *ast.BLangXMLTextLiteral:
		return true
	case *ast.BLangXMLFilterExpression:
		ok := resolveNamePatterns(r, n.NamePattern)
		return resolveOptionalExpr(r, n.Expression) && ok
	case *ast.BLangXMLStepExpression:
		return resolveXMLStepExpression(r, n)
	case *ast.BLangNamedArgsExpression:
		return resolveOptionalExpr(r, n.Expr)
	case *ast.BLangNewExpression:
		ok := true
		if n.TypeDescriptor != nil {
			ok = resolveTypeDesc(r, n.TypeDescriptor)
		}
		return resolveExprs(r, n.ArgsExprs) && ok
	case *ast.BLangStartAction:
		return resolveOptionalExpr(r, n.Call)
	case *ast.BLangSingleWaitAction:
		return resolveOptionalExpr(r, n.FutureExpr)
	case *ast.BLangAlternateWaitAction:
		return resolveExprs(r, n.FutureExprs)
	case *ast.BLangMultipleWaitAction:
		return resolveExprs(r, n.FutureExprs)
	case *ast.BLangRemoteMethodCallAction:
		// We are creating a deferred symbol here since without determining the type of the reciever we can't determine the actual function symbol
		createDeferredMethodSymbol(r, n)
		ok := resolveOptionalExpr(r, n.Expr)
		return resolveExprs(r, n.ArgExprs) && ok
	case *ast.BLangClientResourceAccessAction:
		ok := resolveOptionalExpr(r, n.Expr)
		for i := range n.Path {
			ok = resolveOptionalExpr(r, n.Path[i].Expr) && ok
		}
		return resolveExprs(r, n.ArgExprs) && ok
	default:
		return reportUnexpectedNode(r, expr)
	}
}

func resolveQueryExpr(r symbolResolver, n *ast.BLangQueryExpr) bool {
	resolver := newBlockSymbolResolverWithBlockScope(r, n)
	ok := true
	for _, clause := range n.QueryClauseList {
		ok = resolveQueryClause(resolver, clause) && ok
	}
	return ok
}

func resolveMappingConstructorExpr(r symbolResolver, n *ast.BLangMappingConstructorExpr) bool {
	resolver := newBlockSymbolResolverWithBlockScope(r, n)
	ok := true
	for _, field := range n.Fields {
		kv, isKeyValue := field.(*ast.BLangMappingKeyValueField)
		if !isKeyValue {
			continue
		}
		if kv.Key != nil && kv.Key.Expr != nil {
			ok = resolveExpr(resolver, kv.Key.Expr) && ok
		}
		ok = resolveOptionalExpr(resolver, kv.ValueExpr) && ok
	}
	return ok
}

func resolveErrorConstructorExpr(r symbolResolver, n *ast.BLangErrorConstructorExpr) bool {
	ok := true
	if n.ErrorTypeRef != nil {
		ok = resolveUserDefinedType(r, n.ErrorTypeRef)
	}
	ok = resolveExprs(r, n.PositionalArgs) && ok
	for i := range n.NamedArgs {
		ok = resolveExpr(r, &n.NamedArgs[i]) && ok
	}
	return ok
}

func resolveFieldBaseAccess(r symbolResolver, n *ast.BLangFieldBaseAccess) bool {
	if common.IsSelfFieldAccess(n) {
		if classScope, ok := getEnclosingClassBodyScope(r); ok {
			return resolveSelfFieldAccess(r, n, classScope)
		}
	}
	return resolveOptionalExpr(r, n.Expr)
}

func resolveInvocation(r symbolResolver, n *ast.BLangInvocation) bool {
	ok := true
	if n.GetExpression() != nil {
		createDeferredMethodSymbol(r, n)
	} else {
		ok = resolveFunctionRef(r, n)
	}
	ok = resolveOptionalExpr(r, n.Expr) && ok
	return resolveExprs(r, n.ArgExprs) && ok
}

func resolveLambda(r symbolResolver, n *ast.BLangLambdaFunction) bool {
	fn := n.Function
	name := fn.Name.GetValue()
	symbol := model.NewFunctionSymbol(name, model.TypedFunctionSignature{}, false, symbolLocationForNode(fn))
	switch parent := r.(type) {
	case *blockSymbolResolver:
		addSymbolAndSetOnNode(parent, name, symbol, fn)
		functionResolver := newFunctionResolver(parent, fn)
		fn.SetScope(functionResolver.scope)
		ok := resolveLambdaFunction(functionResolver, parent, fn)
		return allocateSymbols(parent, parent.scope, functionResolver.scope, fn, fn.GetPosition()) && ok
	case *compilationUnitSymbolResolver:
		parent.AddSymbol(name, symbol)
		symRef, _, _ := parent.GetSymbolFromCurrentScope(name)
		fn.SetSymbol(symRef)
		functionResolver := newFunctionResolver(parent, fn)
		fn.SetScope(functionResolver.scope)
		ok := resolveLambdaFunction(functionResolver, functionResolver, fn)
		return allocateSymbols(parent, parent.scope, functionResolver.scope, fn, fn.GetPosition()) && ok
	default:
		return reportUnexpectedNode(r, n)
	}
}

func resolveNamePatterns(r symbolResolver, patterns []ast.BLangAtomicNamePattern) bool {
	ok := true
	for i, pattern := range patterns {
		resolved, patternOk := resolveAtomicNamePattern(r, r.GetScope(), pattern)
		patterns[i] = resolved
		ok = patternOk && ok
	}
	return ok
}

func resolveXMLStepExpression(r symbolResolver, n *ast.BLangXMLStepExpression) bool {
	ok := resolveOptionalExpr(r, n.Expression)
	if n.Start != nil {
		ok = resolveNamePatterns(r, n.Start.NamePattern) && ok
	}
	for _, extension := range n.Extensions {
		ok = resolveXMLStepExtend(r, extension) && ok
	}
	return ok
}

func resolveXMLStepExtend(r symbolResolver, extension ast.XMLStepExtend) bool {
	switch n := extension.(type) {
	case *ast.BLangXMLStepFilterExtend:
		return resolveNamePatterns(r, n.NamePattern)
	case *ast.BLangXMLStepIndexExtend:
		return resolveOptionalExpr(r, n.Expression)
	case *ast.BLangXMLStepMethodCallExtend:
		if n.Invocation == nil {
			return true
		}
		return resolveExprs(r, n.Invocation.ArgExprs)
	default:
		return reportUnexpectedNode(r, extension)
	}
}

func resolveTypeData(r symbolResolver, typeData *ast.TypeData) bool {
	if typeData.TypeDescriptor == nil {
		return true
	}
	return resolveTypeDesc(r, typeData.TypeDescriptor)
}

func resolveTypeDesc(r symbolResolver, td ast.TypeDescriptor) bool {
	switch n := td.(type) {
	case ast.BLangBadNode:
		return false
	case *ast.BLangArrayType:
		ok := resolveTypeData(r, &n.Elemtype)
		for _, size := range n.Sizes {
			if size != nil {
				ok = resolveExpr(r, size) && ok
			}
		}
		return ok
	case *ast.BLangUserDefinedType:
		return resolveUserDefinedType(r, n)
	case *ast.BLangValueType, *ast.BLangBuiltInRefTypeNode:
		return true
	case *ast.BLangFiniteTypeNode:
		return resolveExprs(r, n.ValueSpace)
	case *ast.BLangUnionTypeNode:
		ok := resolveTypeData(r, n.Lhs())
		return resolveTypeData(r, n.Rhs()) && ok
	case *ast.BLangIntersectionTypeNode:
		ok := resolveTypeData(r, n.Lhs())
		return resolveTypeData(r, n.Rhs()) && ok
	case *ast.BLangErrorTypeNode:
		return resolveTypeData(r, &n.DetailType)
	case *ast.BLangConstrainedType:
		ok := resolveTypeData(r, &n.Type)
		return resolveTypeData(r, &n.Constraint) && ok
	case *ast.BLangStreamType:
		ok := resolveTypeData(r, &n.ValueType)
		return resolveTypeData(r, &n.CompletionType) && ok
	case *ast.BLangTupleTypeNode:
		return resolveTupleType(r, n)
	case *ast.BLangRecordType:
		return resolveRecordType(r, n)
	case *ast.BLangObjectType:
		return resolveObjectType(r, n)
	case *ast.BLangFunctionType:
		return resolveFunctionTypeSymbols(r, n)
	case *ast.BLangReturnTypeDescriptor:
		ok := true
		for i := range n.AnnAttachments {
			ok = resolveAnnotationAttachment(r, &n.AnnAttachments[i]) && ok
		}
		if n.TypeDescriptor != nil {
			ok = resolveTypeDesc(r, n.TypeDescriptor) && ok
		}
		return ok
	default:
		return reportUnexpectedNode(r, td)
	}
}

func resolveUserDefinedType(r symbolResolver, n *ast.BLangUserDefinedType) bool {
	return referUserDefinedType(r, n)
}

func resolveTupleType(r symbolResolver, n *ast.BLangTupleTypeNode) bool {
	ok := true
	for i := range n.Members {
		member := &n.Members[i]
		for j := range member.AnnAttachments {
			ok = resolveAnnotationAttachment(r, &member.AnnAttachments[j]) && ok
		}
		if member.TypeDesc != nil {
			ok = resolveTypeDesc(r, member.TypeDesc) && ok
		}
	}
	if n.Rest != nil {
		ok = resolveTypeDesc(r, n.Rest) && ok
	}
	return ok
}

func resolveRecordType(r symbolResolver, n *ast.BLangRecordType) bool {
	inclusions, positions, ok := resolveRecordTypeInclusions(r, n.TypeInclusions)
	n.Inclusions, n.InclusionPositions = inclusions, positions
	allocateRecordDefaultSymbols(r, n)
	for _, field := range n.FieldPtrs() {
		attachments := field.GetAnnotationAttachments()
		for i := range attachments {
			ok = resolveAnnotationAttachment(r, &attachments[i]) && ok
		}
		if field.Type != nil {
			ok = resolveTypeDesc(r, field.Type) && ok
		}
		if field.Default != nil && field.Default.Expr != nil {
			ok = resolveExpr(r, field.Default.Expr) && ok
		}
	}
	if n.RestType != nil {
		ok = resolveTypeDesc(r, n.RestType) && ok
	}
	return ok
}

func resolveObjectType(r symbolResolver, n *ast.BLangObjectType) bool {
	inclusions, positions, _, ok := resolveObjectInclusions(r, n.PopUnresolvedInclusions())
	n.Inclusions, n.InclusionPositions = inclusions, positions
	for member := range n.Members() {
		ok = resolveObjectMember(r, member) && ok
	}
	return ok
}

func resolveObjectMember(r symbolResolver, member ast.ObjectMember) bool {
	switch n := member.(type) {
	case *ast.BObjectField:
		ok := true
		attachments := n.GetAnnotationAttachments()
		for i := range attachments {
			ok = resolveAnnotationAttachment(r, &attachments[i]) && ok
		}
		if n.Ty != nil {
			ok = resolveTypeDesc(r, n.Ty) && ok
		}
		return ok
	case *ast.BMethodDecl:
		return resolveMethodDecl(r, n)
	default:
		if node, isNode := member.(ast.Node); isNode {
			return reportUnexpectedNode(r, node)
		}
		r.GetCtx().InternalError(fmt.Sprintf("unexpected node kind in symbol resolution: %T", member), diagnostics.Location{})
		return false
	}
}

func resolveMethodDecl(r symbolResolver, n *ast.BMethodDecl) bool {
	ok := resolveFunctionTypeSymbols(r, &n.BLangFunctionType)
	if n.Symbol().IsEmpty() {
		space := r.GetScope().MainSpace()
		index := space.AppendSymbol(model.NewFunctionSymbol(n.Name(), model.TypedFunctionSignature{}, false, n.GetPosition()))
		n.SetSymbol(space.RefAt(index))
	}
	return associateFunctionSignatureRef(r.GetCtx(), n.Symbol(), n.SignatureRef(), n.GetPosition()) && ok
}
