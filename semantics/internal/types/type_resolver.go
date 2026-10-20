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

package types

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ast"
	balCommon "github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/opaque"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type distinctTypeSymbol interface {
	DistinctTypeIDs() []int
	SetDistinctTypeIDs([]int)
}

type assignmentNode interface {
	GetVariable() ast.LExpr
	GetExpression() ast.BLangActionOrExpression
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

type typeResolver interface {
	typeContext() semtypes.Context
	expectedReturnType() semtypes.SemType
	parent() typeResolver
	typeEnv() semtypes.Env
	isEphemeral() bool

	// Error reporting (proxied from CompilerContext)
	semanticError(message string, loc diagnostics.Location)
	internalError(message string, loc diagnostics.Location)
	unimplemented(message string, loc diagnostics.Location)

	// Symbol management (proxied from CompilerContext)
	symbolType(ref model.SymbolRef) semtypes.SemType
	setSymbolType(ref model.SymbolRef, ty semtypes.SemType)
	getSymbol(ref model.SymbolRef) model.Symbol
	unnarrowedSymbol(ref model.SymbolRef) model.SymbolRef
	symbolName(ref model.SymbolRef) string
	createNarrowedSymbol(ref model.SymbolRef) model.SymbolRef
	createFunctionSymbol(space *model.SymbolSpace, name string, sig model.TypedFunctionSignature, fnTy semtypes.SemType) model.SymbolRef
	allocateFunctionSignature(params []model.Param, hasRest bool) model.FunctionSignatureRef
	associateFunctionSignature(owner model.SymbolRef, ref model.FunctionSignatureRef) bool
	functionSignatureRef(owner model.SymbolRef) (model.FunctionSignatureRef, bool)
	returnFunctionSignatureRef(source model.FunctionSignatureRef) (model.FunctionSignatureRef, bool)
	updateFunctionSignatureIncludedRecords(ref model.FunctionSignatureRef, includedRecords []*model.IncludedRecordMetadata)
	functionSignature(owner model.SymbolRef) (model.UntypedFunctionSignature, bool)
	functionSignatureByRef(ref model.FunctionSignatureRef) model.UntypedFunctionSignature
	compilerContext() *context.CompilerContext

	// Import management
	lookupImportedSymbols(pkgName string) (model.ExportedSymbolSpace, bool)
	addImplicitImport(pkgName string, imp ast.BLangImportPackage)
	hasImplicitImport(pkgName string) bool

	ensureResolved(ref model.SymbolRef, depth int) bool

	setMappingDefaults(atom *semtypes.MappingAtomicType, defaults []model.FieldDefault)
	mappingDefaults(atom *semtypes.MappingAtomicType) ([]model.FieldDefault, bool)
	setObjectMethodTable(atom *semtypes.MappingAtomicType, table model.MethodTable)
	objectMethodTable(atom *semtypes.MappingAtomicType) (model.MethodTable, bool)
	currentScope() model.Scope
	setCurrentScope(scope model.Scope)
	nextXMLStepFnName() string
	nextMonoFnName(origName string) string
	packageID() *model.PackageID

	ensureNotEmpty(ty semtypes.SemType, onEmpty func()) bool
	opaqueContext() *opaque.Context
}

type resolverContext uint8

const (
	resolverContextIsolated resolverContext = 1 << iota
	resolverContextIsolatedXMLStep
)

func resolverContextForFunction(isolated bool) resolverContext {
	if isolated {
		return resolverContextIsolated
	}
	return 0
}

// deferredEmptinessCheck is an emptiness check that was registered while the
// type env still had unset recursive atoms. It runs once the env is ready.
type deferredEmptinessCheck struct {
	ty      semtypes.SemType
	onEmpty func()
}

// resolutionStatus tracks lazy resolution progress for cycle detection.
// resolutionFailed marks a constant or a global variable whose resolution
// already reported a diagnostic, so a reference to it fails without reporting
// another.
type resolutionStatus int

const (
	resolutionPending resolutionStatus = iota
	resolutionInProgress
	resolutionDone
	resolutionFailed
)

type atomSideTableBase struct {
	atomicTypeInterner      *semtypes.AtomicTypeInterner
	mappingDefaultsByHandle map[semtypes.InternHandle][]model.FieldDefault
	methodTablesByHandle    map[semtypes.InternHandle]model.MethodTable
}

func newAtomSideTableBase() *atomSideTableBase {
	return &atomSideTableBase{
		atomicTypeInterner:      semtypes.NewAtomicTypeInterner(),
		mappingDefaultsByHandle: make(map[semtypes.InternHandle][]model.FieldDefault),
		methodTablesByHandle:    make(map[semtypes.InternHandle]model.MethodTable),
	}
}

func (r *atomSideTableBase) setObjectMethodTable(atom *semtypes.MappingAtomicType, table model.MethodTable) {
	handle := r.atomicTypeInterner.Intern(atom)
	r.methodTablesByHandle[handle] = table
}

func (r *atomSideTableBase) objectMethodTable(atom *semtypes.MappingAtomicType) (model.MethodTable, bool) {
	handle, ok := r.atomicTypeInterner.Lookup(atom)
	if !ok {
		return model.MethodTable{}, false
	}
	table, ok := r.methodTablesByHandle[handle]
	return table, ok
}

func (r *atomSideTableBase) setMappingDefaults(atom *semtypes.MappingAtomicType, defaults []model.FieldDefault) {
	handle := r.atomicTypeInterner.Intern(atom)
	r.mappingDefaultsByHandle[handle] = defaults
}

func (r *atomSideTableBase) mappingDefaults(atom *semtypes.MappingAtomicType) ([]model.FieldDefault, bool) {
	handle, ok := r.atomicTypeInterner.Lookup(atom)
	if !ok {
		return nil, false
	}
	defaults, ok := r.mappingDefaultsByHandle[handle]
	return defaults, ok
}

type packageTypeResolver struct {
	ctx             *context.CompilerContext
	tyCtx           semtypes.Context
	importedSymbols map[string]model.ExportedSymbolSpace
	pkg             *ast.BLangPackage
	implicitImports map[string]ast.BLangImportPackage

	// packageConstants maps a constant's symbol ref to its AST node.
	packageConstants map[model.SymbolRef]*ast.BLangVariable
	// globalVarNodes holds every module-level var. Default expressions are
	// type-checked while type definitions, classes and function signatures are
	// resolved, so a reference to a module-level var can be reached before the
	// global-var pass runs. Both declared and inferred types are therefore
	// resolved lazily (driven by ensureResolved) the same way constants are.
	globalVarNodes map[model.SymbolRef]*ast.BLangVariable
	// lazyResolutionStatus tracks per-symbol resolution progress (for both
	// constants and module-level vars) for cycle detection.
	// Absence means resolution has not started.
	lazyResolutionStatus map[model.SymbolRef]resolutionStatus
	functionNodes        map[model.SymbolRef]*ast.BLangFunction
	typeDefnNodes        map[model.SymbolRef]*ast.BLangTypeDefinition
	classDefnNodes       map[model.SymbolRef]*ast.BLangClassDefinition
	monoCounters         map[string]int
	// xmlStepCounters holds the next index per owner name; a lambda resolver shares its enclosing body's table.
	xmlStepCounters       map[string]int
	annotationGlobalCount int
	scope                 model.Scope
	context               resolverContext
	// opaqueCtx is created on first use: most resolvers never monomorphize an opaque call.
	opaqueCtx      *opaque.Context
	ephemeralState ephemeralState

	deferredEmptinessChecks []deferredEmptinessCheck
}

func (t *packageTypeResolver) ensureNotEmpty(ty semtypes.SemType, onEmpty func()) bool {
	if t.typeEnv().IsReady() {
		if semtypes.IsEmpty(t.typeContext(), ty) {
			onEmpty()
			return false
		}
		return true
	}
	if t.isEphemeral() {
		return true
	}
	t.deferredEmptinessChecks = append(t.deferredEmptinessChecks, deferredEmptinessCheck{ty: ty, onEmpty: onEmpty})
	return true
}

// drainDeferredEmptinessChecks runs every queued emptiness check. The type
// env must be ready at this point; if it is not, that signals that not all
// recursive atoms were resolved which is a compiler bug.
func (t *packageTypeResolver) drainDeferredEmptinessChecks() {
	if !t.typeEnv().IsReady() {
		t.internalError("type env not ready when draining deferred emptiness checks", diagnostics.Location{})
		return
	}
	cx := t.typeContext()
	for _, c := range t.deferredEmptinessChecks {
		if semtypes.IsEmpty(cx, c.ty) {
			c.onEmpty()
		}
	}
	t.deferredEmptinessChecks = nil
}

func (t *packageTypeResolver) typeContext() semtypes.Context        { return t.tyCtx }
func (t *packageTypeResolver) expectedReturnType() semtypes.SemType { return semtypes.SemType{} }
func (t *packageTypeResolver) parent() typeResolver                 { return nil }
func (t *packageTypeResolver) typeEnv() semtypes.Env                { return t.ctx.GetTypeEnv() }
func (t *packageTypeResolver) isEphemeral() bool                    { return t.ephemeralState.depth > 0 }
func (t *packageTypeResolver) opaqueContext() *opaque.Context {
	if t.opaqueCtx == nil {
		t.opaqueCtx = opaque.NewContext(t.tyCtx)
	}
	return t.opaqueCtx
}

func (t *packageTypeResolver) semanticError(msg string, loc diagnostics.Location) {
	if t.isEphemeral() {
		return
	}
	t.ctx.SemanticError(msg, loc)
}

func (t *packageTypeResolver) internalError(msg string, loc diagnostics.Location) {
	t.ctx.InternalError(msg, loc)
}

func (t *packageTypeResolver) unimplemented(msg string, loc diagnostics.Location) {
	noteEphemeralUnimplemented(t)
	t.ctx.Unimplemented(msg, loc)
}

func (t *packageTypeResolver) symbolType(ref model.SymbolRef) semtypes.SemType {
	return t.ctx.SymbolType(ref)
}

func (t *packageTypeResolver) setSymbolType(ref model.SymbolRef, ty semtypes.SemType) {
	t.ctx.SetSymbolType(ref, ty)
}

func (t *packageTypeResolver) getSymbol(ref model.SymbolRef) model.Symbol {
	return t.ctx.GetSymbol(ref)
}

func (t *packageTypeResolver) unnarrowedSymbol(ref model.SymbolRef) model.SymbolRef {
	return t.ctx.UnnarrowedSymbol(ref)
}

func (t *packageTypeResolver) symbolName(ref model.SymbolRef) string {
	return t.ctx.SymbolName(ref)
}

func (t *packageTypeResolver) createNarrowedSymbol(ref model.SymbolRef) model.SymbolRef {
	return t.ctx.CreateNarrowedSymbol(ref)
}

func (t *packageTypeResolver) createFunctionSymbol(space *model.SymbolSpace, name string, sig model.TypedFunctionSignature, fnTy semtypes.SemType) model.SymbolRef {
	return t.ctx.CreateFunctionSymbol(space, name, sig, fnTy)
}

func (t *packageTypeResolver) allocateFunctionSignature(params []model.Param, hasRest bool) model.FunctionSignatureRef {
	return t.ctx.AllocateFunctionSignature(params, hasRest)
}

func (t *packageTypeResolver) associateFunctionSignature(owner model.SymbolRef, ref model.FunctionSignatureRef) bool {
	return t.ctx.AssociateFunctionSignature(owner, ref)
}

func (t *packageTypeResolver) functionSignatureRef(owner model.SymbolRef) (model.FunctionSignatureRef, bool) {
	return t.ctx.FunctionSignatureRef(owner)
}

func (t *packageTypeResolver) returnFunctionSignatureRef(source model.FunctionSignatureRef) (model.FunctionSignatureRef, bool) {
	return t.ctx.ReturnFunctionSignatureRef(source)
}

func (t *packageTypeResolver) updateFunctionSignatureIncludedRecords(ref model.FunctionSignatureRef, includedRecords []*model.IncludedRecordMetadata) {
	t.ctx.UpdateFunctionSignatureIncludedRecords(ref, includedRecords)
}

func (t *packageTypeResolver) functionSignature(owner model.SymbolRef) (model.UntypedFunctionSignature, bool) {
	return t.ctx.GetFunctionSignature(owner)
}

func (t *packageTypeResolver) functionSignatureByRef(ref model.FunctionSignatureRef) model.UntypedFunctionSignature {
	return t.ctx.GetFunctionSignatureByRef(ref)
}

func (t *packageTypeResolver) compilerContext() *context.CompilerContext {
	return t.ctx
}

func (t *packageTypeResolver) setMappingDefaults(atom *semtypes.MappingAtomicType, defaults []model.FieldDefault) {
	t.ctx.SetMappingDefaults(atom, defaults)
}

func (t *packageTypeResolver) mappingDefaults(atom *semtypes.MappingAtomicType) ([]model.FieldDefault, bool) {
	return t.ctx.MappingDefaults(atom)
}

func (t *packageTypeResolver) setObjectMethodTable(atom *semtypes.MappingAtomicType, table model.MethodTable) {
	t.ctx.SetObjectMethodTable(atom, table)
}

func (t *packageTypeResolver) objectMethodTable(atom *semtypes.MappingAtomicType) (model.MethodTable, bool) {
	return t.ctx.ObjectMethodTable(atom)
}

func (t *packageTypeResolver) currentScope() model.Scope     { return t.scope }
func (t *packageTypeResolver) setCurrentScope(s model.Scope) { t.scope = s }

// moduleXMLStepOwner names module-level lowering contexts. It starts with `$`
// so it can never clash with a user-written or generated function name.
const moduleXMLStepOwner = "$module"

// xmlStepFnPrefix starts the name of every function an XML step lowers to.
const xmlStepFnPrefix = "$xmlStep$"

func xmlStepFnName(owner string, idx int) string {
	return fmt.Sprintf("%s%s$%d", xmlStepFnPrefix, owner, idx)
}

func (t *packageTypeResolver) nextXMLStepFnName() string {
	return xmlStepFnName(moduleXMLStepOwner, nextXMLStepIndex(t.xmlStepCounters, moduleXMLStepOwner))
}

func nextXMLStepIndex(counters map[string]int, owner string) int {
	idx := counters[owner]
	counters[owner] = idx + 1
	return idx
}

// resolverXMLStepCounters returns the counter table of the body enclosing t, for a lambda resolver to share.
func resolverXMLStepCounters(t typeResolver) map[string]int {
	switch resolver := t.(type) {
	case *packageTypeResolver:
		return resolver.xmlStepCounters
	case *functionTypeResolver:
		return resolver.xmlStepCounters
	case *loopTypeResolver:
		return resolverXMLStepCounters(resolver.parentResolver)
	default:
		// Unreachable: these are the only resolver kinds. The fresh table keeps the caller from writing a nil map.
		t.internalError(fmt.Sprintf("no xml step counters for resolver %T", t), diagnostics.Location{})
		return make(map[string]int)
	}
}

func (t *packageTypeResolver) packageID() *model.PackageID { return t.pkg.PackageID }

func (t *packageTypeResolver) nextMonoFnName(origName string) string {
	idx := t.monoCounters[origName]
	t.monoCounters[origName] = idx + 1
	return fmt.Sprintf("$mono$%s$%d", origName, idx)
}

func (t *packageTypeResolver) lookupImportedSymbols(name string) (model.ExportedSymbolSpace, bool) {
	s, ok := t.importedSymbols[name]
	return s, ok
}

func (t *packageTypeResolver) addImplicitImport(name string, imp ast.BLangImportPackage) {
	t.implicitImports[name] = imp
}

func (t *packageTypeResolver) hasImplicitImport(name string) bool {
	_, ok := t.implicitImports[name]
	return ok
}

type functionTypeResolver struct {
	*atomSideTableBase
	parentResolver  typeResolver
	tyCtx           semtypes.Context
	retTy           semtypes.SemType
	implicitImports map[string]ast.BLangImportPackage
	monoCounters    map[string]int
	xmlStepOwner    string
	// xmlStepCounters holds the next index per owner name; a lambda resolver shares its enclosing body's table.
	xmlStepCounters map[string]int
	scope           model.Scope
	context         resolverContext
	// opaqueCtx is created on first use: most resolvers never monomorphize an opaque call.
	opaqueCtx      *opaque.Context
	ephemeralState *ephemeralState
}

func (f *functionTypeResolver) typeContext() semtypes.Context        { return f.tyCtx }
func (f *functionTypeResolver) expectedReturnType() semtypes.SemType { return f.retTy }
func (f *functionTypeResolver) parent() typeResolver                 { return f.parentResolver }
func (f *functionTypeResolver) typeEnv() semtypes.Env                { return f.parentResolver.typeEnv() }
func (f *functionTypeResolver) isEphemeral() bool {
	state := resolverEphemeralState(f)
	return state != nil && state.depth > 0
}
func (f *functionTypeResolver) opaqueContext() *opaque.Context {
	if f.opaqueCtx == nil {
		f.opaqueCtx = opaque.NewContext(f.tyCtx)
	}
	return f.opaqueCtx
}

func (f *functionTypeResolver) semanticError(msg string, loc diagnostics.Location) {
	if f.isEphemeral() {
		return
	}
	f.parentResolver.semanticError(msg, loc)
}

func (f *functionTypeResolver) internalError(msg string, loc diagnostics.Location) {
	f.parentResolver.internalError(msg, loc)
}

func (f *functionTypeResolver) unimplemented(msg string, loc diagnostics.Location) {
	noteEphemeralUnimplemented(f)
	f.parentResolver.unimplemented(msg, loc)
}

func (f *functionTypeResolver) symbolType(ref model.SymbolRef) semtypes.SemType {
	return f.parentResolver.symbolType(ref)
}

func (f *functionTypeResolver) setSymbolType(ref model.SymbolRef, ty semtypes.SemType) {
	f.parentResolver.setSymbolType(ref, ty)
}

func (f *functionTypeResolver) getSymbol(ref model.SymbolRef) model.Symbol {
	return f.parentResolver.getSymbol(ref)
}

func (f *functionTypeResolver) unnarrowedSymbol(ref model.SymbolRef) model.SymbolRef {
	return f.parentResolver.unnarrowedSymbol(ref)
}

func (f *functionTypeResolver) symbolName(ref model.SymbolRef) string {
	return f.parentResolver.symbolName(ref)
}

func (f *functionTypeResolver) createNarrowedSymbol(ref model.SymbolRef) model.SymbolRef {
	return f.parentResolver.createNarrowedSymbol(ref)
}

func (f *functionTypeResolver) createFunctionSymbol(space *model.SymbolSpace, name string, sig model.TypedFunctionSignature, fnTy semtypes.SemType) model.SymbolRef {
	return f.parentResolver.createFunctionSymbol(space, name, sig, fnTy)
}

func (f *functionTypeResolver) allocateFunctionSignature(params []model.Param, hasRest bool) model.FunctionSignatureRef {
	return f.parentResolver.allocateFunctionSignature(params, hasRest)
}

func (f *functionTypeResolver) associateFunctionSignature(owner model.SymbolRef, ref model.FunctionSignatureRef) bool {
	return f.parentResolver.associateFunctionSignature(owner, ref)
}

func (f *functionTypeResolver) functionSignatureRef(owner model.SymbolRef) (model.FunctionSignatureRef, bool) {
	return f.parentResolver.functionSignatureRef(owner)
}

func (f *functionTypeResolver) returnFunctionSignatureRef(source model.FunctionSignatureRef) (model.FunctionSignatureRef, bool) {
	return f.parentResolver.returnFunctionSignatureRef(source)
}

func (f *functionTypeResolver) updateFunctionSignatureIncludedRecords(ref model.FunctionSignatureRef, includedRecords []*model.IncludedRecordMetadata) {
	f.parentResolver.updateFunctionSignatureIncludedRecords(ref, includedRecords)
}

func (f *functionTypeResolver) functionSignature(owner model.SymbolRef) (model.UntypedFunctionSignature, bool) {
	return f.parentResolver.functionSignature(owner)
}

func (f *functionTypeResolver) functionSignatureByRef(ref model.FunctionSignatureRef) model.UntypedFunctionSignature {
	return f.parentResolver.functionSignatureByRef(ref)
}

func (f *functionTypeResolver) compilerContext() *context.CompilerContext {
	return f.parentResolver.compilerContext()
}

// ensureNotEmpty runs the emptiness check on this resolver's own type
// context. Function bodies are resolved concurrently (see
// ResolvePrivateNodes), and a semtypes.Context carries unsynchronized memo
// tables, so the check must never run on the shared parent context. Only the
// deferral path, which is reachable only while the env is still being built
// (a sequential phase), goes to the parent.
func (f *functionTypeResolver) ensureNotEmpty(ty semtypes.SemType, onEmpty func()) bool {
	if !f.typeEnv().IsReady() {
		if f.isEphemeral() {
			return true
		}
		return f.parentResolver.ensureNotEmpty(ty, onEmpty)
	}
	if semtypes.IsEmpty(f.typeContext(), ty) {
		onEmpty()
		return false
	}
	return true
}

func (f *functionTypeResolver) lookupImportedSymbols(name string) (model.ExportedSymbolSpace, bool) {
	return f.parentResolver.lookupImportedSymbols(name)
}

func (f *functionTypeResolver) addImplicitImport(name string, imp ast.BLangImportPackage) {
	f.implicitImports[name] = imp
}

func (f *functionTypeResolver) hasImplicitImport(name string) bool {
	_, ok := f.implicitImports[name]
	return ok
}

func (f *functionTypeResolver) ensureResolved(ref model.SymbolRef, depth int) bool {
	if ref.IsEmpty() {
		return false
	}
	if f.isEphemeral() {
		resolved := !semtypes.IsZero(f.symbolType(ref))
		if !resolved {
			f.internalError("unresolved symbol reached ephemeral resolution", diagnostics.Location{})
		}
		return resolved
	}
	return f.parentResolver.ensureResolved(ref, depth)
}

func (f *functionTypeResolver) mappingDefaults(atom *semtypes.MappingAtomicType) ([]model.FieldDefault, bool) {
	if _, ok := f.atomicTypeInterner.Lookup(atom); ok {
		return f.atomSideTableBase.mappingDefaults(atom)
	}
	return f.parentResolver.mappingDefaults(atom)
}

func (f *functionTypeResolver) setObjectMethodTable(atom *semtypes.MappingAtomicType, table model.MethodTable) {
	f.atomSideTableBase.setObjectMethodTable(atom, table)
}

func (f *functionTypeResolver) objectMethodTable(atom *semtypes.MappingAtomicType) (model.MethodTable, bool) {
	if _, ok := f.atomicTypeInterner.Lookup(atom); ok {
		return f.atomSideTableBase.objectMethodTable(atom)
	}
	return f.parentResolver.objectMethodTable(atom)
}

func (f *functionTypeResolver) currentScope() model.Scope     { return f.scope }
func (f *functionTypeResolver) setCurrentScope(s model.Scope) { f.scope = s }

func isolatedContext(t typeResolver) bool {
	return hasResolverContext(t, resolverContextIsolated)
}

func setIsolatedContext(t typeResolver, isolated bool) func() {
	return setResolverContext(t, resolverContextIsolated, isolated)
}

func setIsolatedXMLStepContext(t typeResolver, isolated bool) func() {
	return setResolverContext(t, resolverContextIsolatedXMLStep, isolated)
}

func isolatedXMLStepContext(t typeResolver) bool {
	return hasResolverContext(t, resolverContextIsolatedXMLStep)
}

// opaqueCallIsolated reports whether an opaque call site must be monomorphized
// as isolated. A record field's default expression is not a function, so the
// surrounding context is not an isolated one, but the lambda an XML step lowers
// to there is generated code that captures nothing mutable.
func opaqueCallIsolated(t typeResolver, args []ast.BLangExpression) bool {
	if isolatedContext(t) {
		return true
	}
	return isolatedXMLStepContext(t) && slices.ContainsFunc(args, isGeneratedXMLStepLambda)
}

func isGeneratedXMLStepLambda(expr ast.BLangExpression) bool {
	lambda, ok := expr.(*ast.BLangLambdaFunction)
	return ok && lambda.Function.Name != nil && strings.HasPrefix(lambda.Function.Name.GetValue(), xmlStepFnPrefix)
}

func hasResolverContext(t typeResolver, flag resolverContext) bool {
	for current := t; current != nil; current = current.parent() {
		switch resolver := current.(type) {
		case *functionTypeResolver:
			return resolver.context&flag != 0
		case *packageTypeResolver:
			return resolver.context&flag != 0
		}
	}
	return false
}

func setResolverContext(t typeResolver, flag resolverContext, enabled bool) func() {
	for current := t; current != nil; current = current.parent() {
		var contextFlags *resolverContext
		switch resolver := current.(type) {
		case *functionTypeResolver:
			contextFlags = &resolver.context
		case *packageTypeResolver:
			contextFlags = &resolver.context
		}
		if contextFlags != nil {
			previous := *contextFlags
			if enabled {
				*contextFlags |= flag
			} else {
				*contextFlags &^= flag
			}
			return func() { *contextFlags = previous }
		}
	}
	return func() {}
}

func (f *functionTypeResolver) nextXMLStepFnName() string {
	return xmlStepFnName(f.xmlStepOwner, nextXMLStepIndex(f.xmlStepCounters, f.xmlStepOwner))
}

func (f *functionTypeResolver) packageID() *model.PackageID { return f.parentResolver.packageID() }

func (f *functionTypeResolver) nextMonoFnName(origName string) string {
	idx := f.monoCounters[origName]
	f.monoCounters[origName] = idx + 1
	return fmt.Sprintf("$mono$%s$%d", origName, idx)
}

func newPackageTypeResolver(ctx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace, moduleScope model.Scope) *packageTypeResolver {
	return &packageTypeResolver{
		ctx:                  ctx,
		tyCtx:                semtypes.ContextFrom(ctx.GetTypeEnv()),
		importedSymbols:      importedSymbols,
		pkg:                  pkg,
		implicitImports:      make(map[string]ast.BLangImportPackage),
		packageConstants:     make(map[model.SymbolRef]*ast.BLangVariable),
		globalVarNodes:       make(map[model.SymbolRef]*ast.BLangVariable),
		lazyResolutionStatus: make(map[model.SymbolRef]resolutionStatus),
		functionNodes:        make(map[model.SymbolRef]*ast.BLangFunction),
		typeDefnNodes:        make(map[model.SymbolRef]*ast.BLangTypeDefinition),
		classDefnNodes:       make(map[model.SymbolRef]*ast.BLangClassDefinition),
		monoCounters:         make(map[string]int),
		xmlStepCounters:      make(map[string]int),
		scope:                moduleScope,
	}
}

func (t *packageTypeResolver) ensureResolved(ref model.SymbolRef, depth int) bool {
	if ref.IsEmpty() {
		return false
	}
	if !semtypes.IsZero(t.symbolType(ref)) {
		return true
	}
	if t.isEphemeral() {
		t.internalError("unresolved symbol reached ephemeral resolution", diagnostics.Location{})
		return false
	}
	if defn, ok := t.typeDefnNodes[ref]; ok {
		return t.resolveTypeOnce(ref, func() bool { return resolveTypeDefinition(t, defn, depth) })
	}
	if classDef, ok := t.classDefnNodes[ref]; ok {
		return t.resolveTypeOnce(ref, func() bool { return resolveClassTypeDefinition(t, classDef, depth) })
	}
	if c, inMap := t.packageConstants[ref]; inMap {
		switch t.lazyResolutionStatus[ref] {
		case resolutionDone:
			return true
		case resolutionFailed:
			return false
		case resolutionInProgress:
			var pos diagnostics.Location
			if c.Name != nil {
				pos = c.Name.GetPosition()
			}
			t.semanticError(fmt.Sprintf("invalid cycle detected for %s", t.symbolName(ref)), pos)
			return false
		default:
			return t.resolveLazyConstant(c)
		}
	}
	if gv, inMap := t.globalVarNodes[ref]; inMap {
		switch t.lazyResolutionStatus[ref] {
		case resolutionDone:
			return true
		case resolutionFailed:
			return false
		case resolutionInProgress:
			var pos diagnostics.Location
			if gv.Name != nil {
				pos = gv.Name.GetPosition()
			}
			t.semanticError(fmt.Sprintf("invalid cycle detected for %s", t.symbolName(ref)), pos)
			return false
		default:
			t.lazyResolutionStatus[ref] = resolutionInProgress
			var ok bool
			if gv.TypeNode() != nil {
				ok = resolveGlobalVarType(t, gv)
			} else {
				_, ok = resolveSimpleVariable(t, nil, gv)
			}
			if ok {
				t.lazyResolutionStatus[ref] = resolutionDone
			} else {
				t.lazyResolutionStatus[ref] = resolutionFailed
			}
			return ok
		}
	}
	if fn, ok := t.functionNodes[ref]; ok {
		_, _, ok := resolveFunctionSignature(t, nil, fn, depth)
		return ok
	}
	return true
}

// resolveTypeOnce resolves the type defined at ref unless an earlier attempt failed. A failed type has no type, and
// every reference to it fails without resolving it again, which would report its diagnostics again.
func (t *packageTypeResolver) resolveTypeOnce(ref model.SymbolRef, resolve func() bool) bool {
	if t.lazyResolutionStatus[ref] == resolutionFailed {
		return false
	}
	if !resolve() {
		t.lazyResolutionStatus[ref] = resolutionFailed
		return false
	}
	return true
}

// ResolvePublicNodeTypes resolves types of public symbols. After this dependencies can use the ExportedSymbolSpace for this package.
func ResolvePublicNodes(ctx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) {
	t := newPackageTypeResolver(ctx, pkg, importedSymbols, pkg.Scope)
	t.resolveTopLevelTypes(pkg)
	mergeImplicitImports(pkg, t.implicitImports)
}

// mergeImplicitImports appends the langlib imports the resolver added
// implicitly. Iteration order is the map's, so the resulting slice order is not
// stable; the pretty printers normalize import order when they print a package.
func mergeImplicitImports(pkg *ast.BLangPackage, imports map[string]ast.BLangImportPackage) {
	for name := range imports {
		imp := imports[name]
		pkg.AddImportIfAbsent(&imp)
	}
}

// ResolvePrivateNodesTypes resolves the types private nodes within the package. Then can be executed concurrently
func ResolvePrivateNodes(ctx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) {
	p := newPackageTypeResolver(ctx, pkg, importedSymbols, pkg.Scope)
	fns := common.DeclaredNodes(common.PackageFunctionDecls(pkg))

	allImports := make(map[string]ast.BLangImportPackage)
	resolveFieldInitsInScope := func(owner model.SymbolRef, scope model.Scope, fields []*ast.BLangVariable) {
		ft := &functionTypeResolver{
			atomSideTableBase: newAtomSideTableBase(),
			parentResolver:    p,
			tyCtx:             semtypes.ContextFrom(p.typeEnv()),
			implicitImports:   make(map[string]ast.BLangImportPackage),
			monoCounters:      make(map[string]int),
			xmlStepOwner:      p.getSymbol(owner).Name(),
			xmlStepCounters:   make(map[string]int),
			scope:             scope,
			ephemeralState:    &ephemeralState{},
		}
		for _, fieldNode := range fields {
			field := fieldNode
			if field.Expr != nil {
				resolveActionOrExpression(ft, nil, field.Expr.(ast.BLangExpression), field.TypeNode().Type())
			}
		}
		maps.Copy(allImports, ft.implicitImports)
	}
	for _, c := range common.DeclaredNodes(pkg.ClassDefinitions) {
		resolveFieldInitsInScope(c.Symbol(), c.Scope(), c.Fields)
	}
	for i := range pkg.Services {
		s := pkg.Services[i]
		resolveFieldInitsInScope(s.Symbol(), s.Scope(), s.Fields)
	}

	resolvers := make([]*functionTypeResolver, len(fns))
	var wg sync.WaitGroup
	for i, fn := range fns {
		if isOpaqueFunctionDecl(p, fn) {
			// The declaration of an opaque function has no body of its own to resolve;
			// a call to it is monomorphized into a function of the call's own types.
			continue
		}
		wg.Add(1)
		go func(idx int, f common.FunctionDecl) {
			defer wg.Done()
			resolvers[idx] = resolveFunctionBody(p, f)
		}(i, fn)
	}
	wg.Wait()

	for _, t := range resolvers {
		if t == nil {
			continue
		}
		maps.Copy(allImports, t.implicitImports)
	}
	mergeImplicitImports(pkg, allImports)
}

// isOpaqueFunctionDecl reports whether fn is the lang library source declaration of an
// opaque function, whose symbol is monomorphized per call site instead of typed here.
func isOpaqueFunctionDecl(p *packageTypeResolver, fn common.FunctionDecl) bool {
	_, ok := p.getSymbol(fn.Symbol()).(*model.OpaqueFunctionSymbol)
	return ok
}

func isPolymorphicFnSymbol(sym model.FunctionSymbol) bool {
	switch sym.(type) {
	case model.DependentlyTypedFunctionSymbol:
		return true
	default:
		return false
	}
}

func resolvePackageConstants(t *packageTypeResolver, constants []*ast.BLangVariable) {
	order, ok := topologicallySortConstants(t, constants)
	if !ok {
		return
	}
	// Every constant is resolved even after one fails, so each failing constant
	// reports its own diagnostic; one that depends on a failed constant fails
	// through ensureResolved without reporting another.
	for _, idx := range order {
		constant := constants[idx]
		switch t.lazyResolutionStatus[constant.Symbol()] {
		case resolutionDone, resolutionFailed:
		case resolutionPending, resolutionInProgress:
			t.resolveLazyConstant(constant)
		}
	}
}

func (t *packageTypeResolver) resolveLazyConstant(constant *ast.BLangVariable) bool {
	ref := constant.Symbol()
	t.lazyResolutionStatus[ref] = resolutionInProgress
	ok := resolveConstant(t, constant)
	if ok {
		t.lazyResolutionStatus[ref] = resolutionDone
	} else {
		t.lazyResolutionStatus[ref] = resolutionFailed
	}
	return ok
}

func topologicallySortConstants(t typeResolver, constants []*ast.BLangVariable) ([]int, bool) {
	nodeSet := make(map[model.SymbolRef]int, len(constants))
	for i := range constants {
		nodeSet[constants[i].Symbol()] = i
	}

	deps := make([][]int, len(constants))
	for i := range constants {
		expr, ok := constants[i].Expr.(ast.BLangExpression)
		if !ok {
			continue
		}
		deps[i] = symbolDeps(t, nodeSet, expr)
	}

	const (
		unvisited = 0
		inStack   = 1
		done      = 2
	)
	state := make([]int, len(constants))
	order := make([]int, 0, len(constants))
	stack := make([]int, 0, len(constants))

	reportCycle := func(i int) {
		reportIdx := i
		if len(stack) > 1 {
			reportIdx = stack[1]
		}
		pos := constants[reportIdx].GetPosition()
		if constants[reportIdx].Name != nil {
			pos = constants[reportIdx].Name.GetPosition()
		}
		t.semanticError(fmt.Sprintf("invalid cycle detected for %s", t.symbolName(constants[reportIdx].Symbol())), pos)
	}

	var visit func(int) bool
	visit = func(i int) bool {
		switch state[i] {
		case inStack:
			reportCycle(i)
			return false
		case done:
			return true
		}
		state[i] = inStack
		stack = append(stack, i)
		defer func() {
			stack = stack[:len(stack)-1]
		}()
		for _, d := range deps[i] {
			if !visit(d) {
				return false
			}
		}
		state[i] = done
		order = append(order, i)
		return true
	}

	for i := range constants {
		if !visit(i) {
			return nil, false
		}
	}
	return order, true
}

func resolveInvokableSignature(t typeResolver, chain *binding, fn common.FunctionDecl, fnSym model.FunctionSymbol, requiredParams []ast.BLangVariable, depth int) (semtypes.SemType, *binding, bool) {
	restoreContext := setIsolatedContext(t, fn.IsIsolated())
	defer restoreContext()
	// Every parameter is resolved even after one fails, so the body sees each parameter that has a type.
	paramsOk := true
	paramTypes := make([]semtypes.SemType, len(requiredParams))
	for i := range requiredParams {
		param := &requiredParams[i]
		chain, _ = resolveSimpleVariableInner(t, chain, param, depth+1)
		if !isTypedVariable(param) {
			paramsOk = false
			continue
		}
		if fnType, ok := param.TypeNode().(*ast.BLangFunctionType); ok {
			if !finalizeResolvedFunctionSignature(t, fnType) {
				return semtypes.SemType{}, chain, false
			}
		}
		paramTypes[i] = t.symbolType(param.Symbol())
	}
	restTy := semtypes.Never
	if restParam := fn.GetRestParam(); restParam != nil {
		chain, _ = resolveSimpleVariableInner(t, chain, restParam, depth+1)
		if isTypedVariable(restParam) {
			restTy = t.symbolType(restParam.Symbol())
			listDefn := semtypes.NewListDefinition()
			restParamListTy := listDefn.Define(t.typeEnv(), nil, semtypes.ListRest(restTy),
				semtypes.ListMutability(semtypes.CellMutabilityNone))
			updateSymbolType(t, restParam, restParamListTy)
		} else {
			paramsOk = false
		}
	}
	if !paramsOk {
		return semtypes.SemType{}, chain, false
	}
	returnTy, fnType, chain, ok := invokableSignatureTypes(t, chain, fn, paramTypes, restTy, depth)
	if !ok {
		return semtypes.SemType{}, chain, false
	}
	updateSymbolType(t, fn, fnType)
	sig := fnSym.TypedSignature()
	sig.Flags |= fn.FuncSymbolFlags()
	sig.ParamTypes = paramTypes
	sig.ReturnType = returnTy
	sig.RestParamType = restTy
	fnSym.SetTypedSignature(sig)
	return fnType, chain, true
}

// isTypedVariable reports whether variable declares a symbol and its type descriptor resolved. A failed
// initializer does not affect the variable's type.
func isTypedVariable(variable *ast.BLangVariable) bool {
	return ast.SymbolIsSet(variable) && !semtypes.IsZero(variable.TypeNode().Type())
}

// functionSignatureType computes the type of fn from its parameter, rest parameter and return type descriptors
// without writing anything. A defaulted parameter contributes the capture of its default provider to chain.
func functionSignatureType(t typeResolver, chain *binding, fn *ast.BLangFunction) (semtypes.SemType, *binding, bool) {
	restoreContext := setIsolatedContext(t, fn.IsIsolated())
	defer restoreContext()
	params := fn.GetParameters()
	paramTypes := make([]semtypes.SemType, len(params))
	for i := range params {
		paramTy, paramChain, ok := resolveBType(t, chain, params[i].TypeNode(), 1)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		chain = paramChain
		if params[i].Expr != nil {
			chain = addCaptureGroup(chain, params[i].CaptureGroup())
		}
		paramTypes[i] = paramTy
	}
	restTy := semtypes.Never
	if restParam := fn.GetRestParam(); restParam != nil {
		var ok bool
		if restTy, chain, ok = resolveBType(t, chain, restParam.TypeNode(), 1); !ok {
			return semtypes.SemType{}, chain, false
		}
	}
	_, fnType, chain, ok := invokableSignatureTypes(t, chain, fn, paramTypes, restTy, 0)
	return fnType, chain, ok
}

func invokableSignatureTypes(t typeResolver, chain *binding, fn common.FunctionDecl, paramTypes []semtypes.SemType, restTy semtypes.SemType, depth int) (returnTy semtypes.SemType, fnType semtypes.SemType, _ *binding, ok bool) {
	paramListDefn := semtypes.NewListDefinition()
	paramListTy := paramListDefn.Define(t.typeEnv(), paramTypes, semtypes.ListRest(restTy),
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	returnTy = semtypes.Nil
	if retTd := fn.GetReturnTypeDescriptor(); retTd != nil {
		if returnTy, chain, ok = resolveBType(t, chain, retTd, depth+1); !ok {
			return semtypes.SemType{}, semtypes.SemType{}, chain, false
		}
	}
	fnDefn := semtypes.NewFunctionDefinition()
	fnType = fnDefn.Define(t.typeEnv(), paramListTy, returnTy,
		semtypes.FunctionQualifiersFrom(t.typeEnv(), fn.IsIsolated(), fn.IsTransactional()))
	return returnTy, fnType, chain, true
}

func resolveFunctionBody(p *packageTypeResolver, fn common.FunctionDecl) *functionTypeResolver {
	fnSymbol := p.getSymbol(fn.Symbol())
	fnSym, ok := fnSymbol.(model.FunctionSymbol)
	if !ok {
		p.internalError("expected function symbol", fn.GetPosition())
		return nil
	}
	ft := &functionTypeResolver{
		atomSideTableBase: newAtomSideTableBase(),
		parentResolver:    p,
		tyCtx:             semtypes.ContextFrom(p.typeEnv()),
		implicitImports:   make(map[string]ast.BLangImportPackage),
		monoCounters:      make(map[string]int),
		xmlStepOwner:      fnSym.Name(),
		xmlStepCounters:   make(map[string]int),
		scope:             fn.Scope(),
		context:           resolverContextForFunction(fn.IsIsolated()),
		ephemeralState:    &ephemeralState{},
	}
	if !isPolymorphicFnSymbol(fnSym) {
		ft.retTy = fnSym.TypedSignature().ReturnType
	}
	body := fn.GetBody()
	if body == nil {
		p.internalError("function body is nil at body-resolution stage", fn.GetPosition())
		return ft
	}
	switch body := body.(type) {
	case *ast.BLangExternFunctionBody:
		_ = body
	case *ast.BLangBlockFunctionBody:
		resolveBlockStatements(ft, nil, body.Stmts)
	case *ast.BLangExprFunctionBody:
		resolveActionOrExpression(ft, nil, body.Expr, ft.retTy)
	default:
		p.internalError("unexpected function body kind", body.GetPosition())
	}
	return ft
}

func (t *packageTypeResolver) resolveTopLevelTypes(pkg *ast.BLangPackage) {
	for _, defn := range common.DeclaredNodes(pkg.TypeDefinitions) {
		t.typeDefnNodes[defn.Symbol()] = defn
	}
	for _, classDef := range common.DeclaredNodes(pkg.ClassDefinitions) {
		t.classDefnNodes[classDef.Symbol()] = classDef
	}
	for _, constant := range common.DeclaredNodes(pkg.Constants) {
		t.packageConstants[constant.Symbol()] = constant
	}
	for _, fn := range common.DeclaredNodes(pkg.Functions) {
		t.functionNodes[fn.Symbol()] = fn
	}
	for _, gv := range common.DeclaredNodes(pkg.GlobalVars) {
		t.globalVarNodes[gv.Symbol()] = gv
	}

	// A top-level node that fails to resolve has no type; the rest are still resolved so each reports its own
	// diagnostics.
	for _, defn := range common.DeclaredNodes(pkg.TypeDefinitions) {
		t.ensureResolved(defn.Symbol(), 0)
	}
	for _, classDef := range common.DeclaredNodes(pkg.ClassDefinitions) {
		t.ensureResolved(classDef.Symbol(), 0)
	}
	for _, annotation := range common.DeclaredNodes(pkg.Annotations) {
		resolveAnnotationDeclaration(t, annotation)
	}
	for _, fn := range common.DeclaredNodes(pkg.Functions) {
		resolveFunctionSignature(t, nil, fn, 0)
	}
	if pkg.InitFunction != nil {
		resolveFunctionSignature(t, nil, pkg.InitFunction, 0)
	}
	for _, gv := range common.DeclaredNodes(pkg.GlobalVars) {
		if gv.TypeNode() == nil {
			// An inferred global's type comes from its initializer, resolved
			// lazily via ensureResolved; this only stamps the name node.
			resolveGlobalVarType(t, gv)
			continue
		}
		t.ensureResolved(gv.Symbol(), 0)
	}
	for _, xmlns := range common.DeclaredNodes(pkg.XmlnsList) {
		resolveXMLNS(t, nil, xmlns)
	}
	resolvePackageConstants(t, common.DeclaredNodes(pkg.Constants))
	// Annotation values can depend on constants, so resolve them after constants
	// have been folded even though the annotated type/function nodes were
	// resolved earlier in the top-level pass.
	resolveTopLevelAnnotationAttachments(t, pkg)
	for _, gv := range common.DeclaredNodes(pkg.GlobalVars) {
		resolveGlobalVarInit(t, gv)
	}
	detectGlobalVarInitCycles(t, common.DeclaredNodes(pkg.GlobalVars))
	attachPointBound := common.ListenerAttachPointBound(t.typeContext())
	validateListenerVars(t, pkg, attachPointBound)
	for i := range pkg.Services {
		svc := pkg.Services[i]
		if resolveServiceAttachedExpressions(t, svc) {
			resolveServiceType(t, svc, 0, attachPointBound)
		}
	}
	t.drainDeferredEmptinessChecks()
}

// annotationTypeValid reports whether ty is a valid annotation type, i.e. a
// subtype of exactly one of: true, map<Cloneable>, map<Cloneable>[].
// Using a combined union for this check would permit mixed union types like
// true|map<Cloneable>, which the spec disallows.
func annotationTypeValid(t typeResolver, ty semtypes.SemType) bool {
	cx := t.typeContext()
	cloneableMap := annotationMapType(t)
	cloneableMapList := annotationMapListType(t)
	return semtypes.IsSubtype(cx, ty, semtypes.BooleanConst(true)) ||
		semtypes.IsSubtype(cx, ty, cloneableMap) ||
		semtypes.IsSubtype(cx, ty, cloneableMapList)
}

// annotationMapType is map<Cloneable>. It is recomputed per call rather than
// cached on the package resolver: caching on the shared resolver would be a
// data race (local-node resolution runs many type resolvers concurrently off
// the same package resolver).
func annotationMapType(t typeResolver) semtypes.SemType {
	return semtypes.Intersect(semtypes.Mapping, semtypes.CreateCloneable(t.typeContext()))
}

func annotationMapListType(t typeResolver) semtypes.SemType {
	ld := semtypes.NewListDefinition()
	return ld.Define(t.typeEnv(), nil, semtypes.ListRest(annotationMapType(t)))
}

func resolveAnnotationDeclaration(t typeResolver, annotation *ast.BLangAnnotation) bool {
	var ty semtypes.SemType
	var ok bool
	if typeDesc := annotation.GetTypeDescriptor(); typeDesc != nil {
		ty, _, ok = resolveBType(t, nil, typeDesc.(ast.BType), 0)
		if !ok {
			return false
		}
		if !annotationTypeValid(t, ty) {
			t.semanticError("annotation type must be a subtype of true|map<Cloneable>|map<Cloneable>[]", typeDesc.GetPosition())
			return false
		}
	} else {
		ty = semtypes.BooleanConst(true)
	}
	if annotation.HasSourceAttachPoint() && !annotation.IsConst() {
		t.semanticError("annotation declaration with source attach point must be const", annotation.GetPosition())
		return false
	}
	t.setSymbolType(annotation.Symbol(), ty)
	return true
}

func resolveTopLevelAnnotationAttachments(t typeResolver, pkg *ast.BLangPackage) {
	initialGlobalCount := len(pkg.GlobalVars)
	for i := range pkg.Annotations {
		resolveAnnotationAttachments(t, pkg.Annotations[i], ast.PointAnnotation, nil)
	}
	for i := range pkg.TypeDefinitions {
		defn := pkg.TypeDefinitions[i]
		resolveAnnotationAttachments(t, defn, ast.PointType, symbolAnnotationSink(t, defn.Symbol()))
		switch typeDesc := defn.GetTypeData().TypeDescriptor.(type) {
		case *ast.BLangRecordType:
			for name, field := range typeDesc.FieldPtrs() {
				resolveAnnotationAttachments(t, field, ast.PointRecordField,
					recordFieldAnnotationSink(t, defn.Symbol(), name))
			}
		case *ast.BLangObjectType:
			for member := range typeDesc.Members() {
				if field, ok := member.(*ast.BObjectField); ok {
					resolveAnnotationAttachments(t, field, ast.PointObjectField, nil)
				}
			}
		}
	}
	propagateRecordFieldAnnotations(t, pkg)
	for i := range pkg.ClassDefinitions {
		classDef := pkg.ClassDefinitions[i]
		classPoint := ast.PointClass
		if classDef.IsService() {
			classPoint = ast.PointService
		}
		resolveAnnotationAttachments(t, classDef, classPoint, symbolAnnotationSink(t, classDef.Symbol()))
		resolveClassBodyAnnotationAttachments(t, classDef.Fields, classDef.InitFunction, classDef.Methods, classDef.ResourceMethods)
	}
	for i := range pkg.Services {
		svc := pkg.Services[i]
		resolveAnnotationAttachments(t, svc, ast.PointService, symbolAnnotationSink(t, svc.Symbol()))
		resolveClassBodyAnnotationAttachments(t, svc.Fields, svc.InitFunction, svc.Methods, svc.ResourceMethods)
	}
	for i := range pkg.Functions {
		resolveFunctionAnnotationAttachments(t, pkg.Functions[i], false)
	}
	if pkg.InitFunction != nil {
		resolveFunctionAnnotationAttachments(t, pkg.InitFunction, false)
	}
	for i := range pkg.Constants {
		resolveAnnotationAttachments(t, pkg.Constants[i], ast.PointConst, nil)
	}
	for i := range pkg.GlobalVars {
		resolveAnnotationAttachments(t, pkg.GlobalVars[i], ast.PointVar, nil)
	}
	if initialGlobalCount < len(pkg.GlobalVars) {
		globals := make([]*ast.BLangVariable, 0, len(pkg.GlobalVars))
		globals = append(globals, pkg.GlobalVars[initialGlobalCount:]...)
		globals = append(globals, pkg.GlobalVars[:initialGlobalCount]...)
		pkg.GlobalVars = globals
	}
}

func resolveClassBodyAnnotationAttachments(t typeResolver, fields []*ast.BLangVariable, initFn *ast.BLangFunction,
	methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod,
) {
	for _, field := range fields {
		resolveAnnotationAttachments(t, field, ast.PointObjectField, nil)
	}
	if initFn != nil {
		resolveFunctionAnnotationAttachments(t, initFn, true)
	}
	for _, method := range common.MethodsInResolutionOrder(methods) {
		resolveFunctionAnnotationAttachments(t, method.Method, true)
	}
	for _, method := range resourceMethods {
		resolveInvokableAnnotationAttachments(t, method, ast.PointObjectMethod)
	}
}

func resolveFunctionAnnotationAttachments(
	t typeResolver,
	fn *ast.BLangFunction,
	attached bool,
) {
	point := ast.PointFunction
	if attached {
		point = ast.PointObjectMethod
	}
	resolveInvokableAnnotationAttachments(t, fn, point)
}

func resolveInvokableAnnotationAttachments(
	t typeResolver,
	fn ast.InvokableNode,
	point ast.Point,
) {
	resolveAnnotationAttachments(t, fn, point, nil)
	parameters := fn.GetParameters()
	for i := range parameters {
		resolveAnnotationAttachments(t, &parameters[i], ast.PointParameter, symbolAnnotationSink(t, parameters[i].Symbol()))
	}
	if restParam := fn.GetRestParam(); restParam != nil {
		resolveAnnotationAttachments(t, restParam, ast.PointParameter, symbolAnnotationSink(t, restParam.Symbol()))
	}
	if ret := fn.GetReturnTypeDescriptor(); ret != nil {
		resolveAnnotationAttachments(t, ret, ast.PointReturn, nil)
	}
}

// recordFieldAnnotationSource describes where a type definition can pick up
// field annotations other than its own: the records it includes, or the type it
// aliases. declaredFields names the fields the definition declares itself, which
// shadow anything inherited for the same field.
type recordFieldAnnotationSource struct {
	sources        []model.SymbolRef
	declaredFields map[string]bool
}

// propagateRecordFieldAnnotations copies the annotations of included record
// fields onto the records that include them, so that
//
//	type Derived record {| *Base; |};
//
// sees the annotations declared on Base's fields. jBallerina drops them at the
// inclusion boundary; we keep them, because a field pulled in by an inclusion is
// the same field, and validation driven by field annotations would otherwise
// silently skip every included field. Aliases (type Alias Base) carry them for
// the same reason.
//
// A field the including record declares itself shadows the inherited one, even
// when it carries no annotations of its own: it is a new declaration, and its
// annotations are exactly the ones written on it.
//
// This runs as a second pass so that it does not depend on the order type
// definitions appear in the source. Sources in other modules need no recursion:
// their annotations were already merged when that module was compiled, and reach
// us through the symbol pool.
func propagateRecordFieldAnnotations(t typeResolver, pkg *ast.BLangPackage) {
	sources := make(map[model.SymbolRef]recordFieldAnnotationSource)
	for i := range pkg.TypeDefinitions {
		defn := pkg.TypeDefinitions[i]
		if !ast.SymbolIsSet(defn) || semtypes.IsZero(t.symbolType(defn.Symbol())) {
			continue
		}
		switch typeDesc := defn.GetTypeData().TypeDescriptor.(type) {
		case *ast.BLangRecordType:
			if len(typeDesc.Inclusions) == 0 {
				continue
			}
			declared := make(map[string]bool)
			for name := range typeDesc.FieldPtrs() {
				declared[name] = true
			}
			sources[defn.Symbol()] = recordFieldAnnotationSource{
				sources:        typeDesc.Inclusions,
				declaredFields: declared,
			}
		case *ast.BLangUserDefinedType:
			if !ast.SymbolIsSet(typeDesc) {
				continue
			}
			sources[defn.Symbol()] = recordFieldAnnotationSource{sources: []model.SymbolRef{typeDesc.Symbol()}}
		}
	}
	for symbol := range sources {
		inherited := propagatedRecordFieldAnnotations(t, symbol, sources, make(map[model.SymbolRef]bool))
		for field, annotations := range inherited {
			for key, value := range annotations {
				t.compilerContext().SetRecordFieldAnnotationValue(symbol, field, key, value)
			}
		}
	}
}

// propagatedRecordFieldAnnotations returns the field annotations visible on the
// type at symbol: those reachable through its inclusions or alias target, minus
// the fields it redeclares, overlaid by the ones declared on its own fields.
func propagatedRecordFieldAnnotations(
	t typeResolver,
	symbol model.SymbolRef,
	sources map[model.SymbolRef]recordFieldAnnotationSource,
	visiting map[model.SymbolRef]bool,
) values.FieldAnnotationValues {
	merged := values.NewFieldAnnotationValues()
	if visiting[symbol] {
		// Cyclic inclusions and alias chains are rejected before annotation
		// resolution, so reaching this means that invariant no longer holds and
		// the metadata we would emit is incomplete.
		t.internalError("cyclic type reference while propagating record field annotations", diagnostics.Location{})
		return merged
	}
	visiting[symbol] = true
	defer delete(visiting, symbol)
	source := sources[symbol]
	for _, from := range source.sources {
		for field, annotations := range propagatedRecordFieldAnnotations(t, from, sources, visiting) {
			if source.declaredFields[field] {
				continue
			}
			for key, value := range annotations {
				merged.Set(field, key, value)
			}
		}
	}
	for field, annotations := range t.compilerContext().RecordFieldAnnotationValues(symbol) {
		for key, value := range annotations {
			merged.Set(field, key, value)
		}
	}
	return merged
}

// annotationSink records the runtime-visible annotation values resolved at one
// attachment site. A nil sink means the site keeps no runtime-visible values.
type annotationSink func(key string, value values.AnnotationValue)

// symbolAnnotationSink stores annotation values against symbol. It returns nil
// for the zero symbol, which marks an attachment site whose values are only
// validated and never kept.
func symbolAnnotationSink(t typeResolver, symbol model.SymbolRef) annotationSink {
	if symbol == (model.SymbolRef{}) {
		return nil
	}
	return func(key string, value values.AnnotationValue) {
		t.compilerContext().SetSymbolAnnotationValue(symbol, key, value)
	}
}

// recordFieldAnnotationSink stores annotation values against the field named
// field of the record type defined by typeDefinition. Record types are
// structural, so the field has no symbol of its own to key on.
func recordFieldAnnotationSink(t typeResolver, typeDefinition model.SymbolRef, field string) annotationSink {
	if typeDefinition == (model.SymbolRef{}) {
		return nil
	}
	return func(key string, value values.AnnotationValue) {
		t.compilerContext().SetRecordFieldAnnotationValue(typeDefinition, field, key, value)
	}
}

func resolveAnnotationAttachments(
	t typeResolver,
	node ast.AnnotatableNode,
	point ast.Point,
	sink annotationSink,
) {
	seen := make(map[string]bool)
	repeatedValues := make(map[string]*repeatedAnnotationValue)
	repeatedOrder := make([]string, 0)
	pointKey := point.String()
	attachments := node.GetAnnotationAttachments()
	for i := range attachments {
		ann := &attachments[i]
		if !ast.SymbolIsSet(ann) {
			continue
		}
		sym, ok := t.getSymbol(ann.Symbol()).(*model.AnnotationSymbol)
		if !ok {
			t.internalError("annotation reference does not resolve to an annotation symbol", ann.GetPosition())
			continue
		}
		if !sym.AllowsAttachPoint(pointKey) {
			t.semanticError("annotation '"+sym.Name()+"' is not allowed on "+pointKey, ann.GetPosition())
			continue
		}
		expectedType := sym.Type()
		if semtypes.IsZero(expectedType) {
			// An annotation whose declaration failed to resolve has no type.
			continue
		}
		valueType, repeated := annotationAttachmentValueType(t, expectedType)
		if semtypes.IsZero(valueType) {
			t.internalError("annotation attachment type is not supported", ann.GetPosition())
			continue
		}
		key := model.AnnotationKey(t.compilerContext().SymbolPackage(ann.Symbol()), sym.Name())
		if seen[key] && !repeated {
			t.semanticError("duplicate annotation '"+sym.Name()+"' on "+pointKey, ann.GetPosition())
			continue
		}
		seen[key] = true
		if ann.HasValue && semtypes.IsSubtype(t.typeContext(), valueType, semtypes.BooleanConst(true)) {
			t.semanticError("annotation '"+sym.Name()+"' does not allow a value", ann.GetPosition())
			continue
		}
		if !ann.HasValue && !prepareImplicitAnnotationValue(t, ann, expectedType, valueType) {
			continue
		}
		if _, ok := resolveActionOrExpression(t, nil, ann.Expr, valueType); !ok {
			continue
		}
		value, runtimeValue, ok := annotationAttachmentValue(t, sym.IsConst(), ann.Expr)
		if !ok {
			continue
		}
		if !runtimeValue {
			ann.AnnotationValue = value
		}
		storeValue := sink != nil && sym.IsRuntimeVisibleAt(pointKey)
		if repeated && storeValue {
			group := repeatedValues[key]
			if group == nil {
				group = &repeatedAnnotationValue{listType: expectedType}
				repeatedValues[key] = group
				repeatedOrder = append(repeatedOrder, key)
			}
			group.values = append(group.values, value)
			group.expressions = append(group.expressions, ann.Expr)
			group.runtime = group.runtime || runtimeValue
			continue
		}
		if runtimeValue {
			if storeValue {
				sink(key, createRuntimeAnnotationGlobal(t, ann.Expr))
			}
			continue
		}
		if storeValue {
			sink(key, value)
		}
	}

	for _, key := range repeatedOrder {
		group := repeatedValues[key]
		atomic := semtypes.ToListAtomicType(t.typeEnv(), group.listType)
		if atomic == nil {
			t.internalError("repeated annotation type is not an atomic list", diagnostics.Location{})
			continue
		}
		if group.runtime {
			expr := &ast.BLangListConstructorExpr{
				Exprs:      group.expressions,
				AtomicType: *atomic,
			}
			expr.SetPosition(group.expressions[0].GetPosition())
			setNodeType(t, expr, group.listType)
			sink(key, createRuntimeAnnotationGlobal(t, expr))
			continue
		}
		restFiller, _ := values.FillerFactoryFor(t.typeContext(), atomic.Rest())
		value := values.NewList(group.listType, atomic, true, restFiller, len(group.values), group.values)
		sink(key, value)
	}
}

func annotationAttachmentValueType(t typeResolver, annotationType semtypes.SemType) (semtypes.SemType, bool) {
	if semtypes.IsSubtypeSimple(annotationType, semtypes.List) {
		memberTy := semtypes.ListMemberTypeInnerVal(t.typeContext(), annotationType, semtypes.Int)
		if semtypes.IsNever(memberTy) {
			return semtypes.SemType{}, true
		}
		return memberTy, true
	}
	return annotationType, false
}

func prepareImplicitAnnotationValue(
	t typeResolver,
	ann *ast.BLangAnnotationAttachment,
	annotationType semtypes.SemType,
	valueType semtypes.SemType,
) bool {
	if semtypes.IsSubtype(t.typeContext(), semtypes.BooleanConst(true), annotationType) {
		ann.Expr = newImplicitBooleanLiteral(t, true, ann.GetPosition())
		return true
	}
	if !semtypes.IsSubtype(t.typeContext(), valueType, annotationMapType(t)) {
		t.semanticError("annotation '"+t.symbolName(ann.Symbol())+"' requires a value", ann.GetPosition())
		return false
	}
	expr := &ast.BLangMappingConstructorExpr{
		Fields: make([]ast.MappingField, 0),
	}
	expr.SetPosition(ann.GetPosition())
	ann.Expr = expr
	return true
}

func newImplicitBooleanLiteral(t typeResolver, value bool, pos diagnostics.Location) *ast.BLangLiteral {
	lit := ast.NewBLangLiteral(pos, ast.LiteralKindBoolean, value, strconv.FormatBool(value), true)
	setNodeType(t, lit, semtypes.BooleanConst(value))
	return lit
}

type repeatedAnnotationValue struct {
	listType    semtypes.SemType
	values      []values.BalValue
	expressions []ast.BLangExpression
	runtime     bool
}

// annotationAttachmentValue folds the value of an annotation attachment. A
// non-const annotation need not have a constant value, so a value that is not a
// constant expression, or one we cannot fold yet, is evaluated at runtime
// instead (runtimeValue). A const annotation reports both.
func annotationAttachmentValue(t typeResolver, isConst bool, expr ast.BLangExpression) (value values.AnnotationValue,
	runtimeValue bool, ok bool,
) {
	if _, isConstant := isConstantExpression(t, expr); !isConstant {
		if isConst {
			t.semanticError("const annotation value must be a constant expression", expr.GetPosition())
			return nil, false, false
		}
		return nil, true, true
	}
	value, unsupported, folded := tryFoldConstant(t, expr)
	switch {
	case folded:
		return value, false, true
	case unsupported != nil && !isConst:
		return nil, true, true
	case unsupported != nil:
		unsupported.report(t)
	}
	return nil, false, false
}

func createRuntimeAnnotationGlobal(t typeResolver, expr ast.BLangExpression) *values.RuntimeAnnotationValueRef {
	resolver, ok := t.(*packageTypeResolver)
	if !ok {
		t.internalError("runtime annotation value is not in a package resolver", expr.GetPosition())
		return &values.RuntimeAnnotationValueRef{}
	}
	var name string
	for {
		name = fmt.Sprintf("$annotation$%d", resolver.annotationGlobalCount)
		resolver.annotationGlobalCount++
		if _, exists := resolver.scope.GetSymbol(name); !exists {
			break
		}
	}
	symbol := model.NewVariableSymbol(name, false, false, false, diagnostics.NewBuiltinLocation())
	symbol.SetType(semtypes.Any)
	resolver.scope.AddSymbol(name, &symbol)
	ref, _ := resolver.scope.GetSymbol(name)

	identifier := &ast.BLangIdentifier{Value: name}
	identifier.SetPosition(expr.GetPosition())
	global := &ast.BLangVariable{Name: identifier}
	global.SetPosition(expr.GetPosition())
	global.SetSymbol(ref)
	global.SetInitialExpression(expr)
	resolver.pkg.GlobalVars = append(resolver.pkg.GlobalVars, global)

	return &values.RuntimeAnnotationValueRef{
		Organization: resolver.pkg.PackageID.OrgName.Value(),
		Module:       resolver.pkg.PackageID.PkgName.Value(),
		GlobalName:   name,
	}
}

func resolveBlockStatements(t typeResolver, chain *binding, stmts []ast.StatementNode) statementEffect {
	result := chain
	for i, each := range stmts {
		eachResult, ok := resolveStatement(t, result, each)
		if !ok {
			continue
		}
		if !eachResult.nonCompletion {
			result = eachResult.binding
		} else {
			rest := stmts[i+1:]
			if len(rest) > 0 {
				// These are unreachable nodes will be caught later by reachability analysis
				// we are doing type resolution here anyway to give error message to these statements
				resolveBlockStatements(t, chain, rest)
			}
			return statementEffect{result, true}
		}
	}
	return statementEffect{result, false}
}

func resolveCompoundAssignment(t typeResolver, chain *binding, s *ast.BLangCompoundAssignment) (statementEffect, bool) {
	lhs := s.GetVariable()
	lhsTy, rhsChain, ok := resolveCompoundAssignmentLhs(t, chain, lhs)
	if !ok {
		return statementEffect{}, false
	}
	rhsEffect, ok := resolveCompoundAssignmentInner(t, rhsChain, lhsTy, s)
	if !ok {
		return statementEffect{}, false
	}
	// Assignment unnarrowing applies on the chain the RHS leaves behind.
	rhsChain = sequentialChain(t, rhsEffect)
	if expr, ok := s.GetVariable().(ast.NodeWithSymbol); ok {
		return unnarrowSymbolAt(t, rhsChain, expr.Symbol(), lhs.GetPosition()), true
	}
	return defaultStmtEffect(rhsChain), true
}

// resolveCompoundAssignmentLhs resolves the LHS of a compound assignment and returns the
// LHS type to use as the operand type along with the chain in which the RHS should be
// resolved. The LHS node's type is always set to its writable (unnarrowed) type, which
// is the target type the result of the operation is checked against. A variable reference
// target is not narrowed: its operand type is the declared type of the variable (spec §7.15.1).
func resolveCompoundAssignmentLhs(t typeResolver, chain *binding, lhs ast.BLangExpression) (semtypes.SemType, *binding, bool) {
	switch lhs.(type) {
	case *ast.BLangIndexBasedAccess, *ast.BLangFieldBaseAccess:
		lhsResult, ok := resolveActionOrExpression(t, chain, lhs, semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, nil, false
		}
		lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
		return lhsTy, sequentialChain(t, lhsEffect), true
	default:
		lhsResult, ok := resolveActionOrExpression(t, nil, lhs, semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, nil, false
		}
		return lhsResult.ty, chain, true
	}
}

// resolveCompoundAssignmentInner resolves `lvexpr op expr` and checks that its static type is a
// subtype of the target type of the compound assignment (spec §7.15.3).
func resolveCompoundAssignmentInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, s *ast.BLangCompoundAssignment) (expressionEffect, bool) {
	target, rhs, op, pos := s.GetVariable(), s.GetExpression(), s.OpKind, s.GetPosition()
	if symbolNode, ok := target.(ast.BNodeWithSymbol); ok && ast.SymbolIsSet(symbolNode) {
		if msg, ok := common.AssignmentTargetError(t.compilerContext().SymbolKind(symbolNode.Symbol())); ok {
			t.semanticError(msg, target.GetPosition())
			return expressionEffect{}, false
		}
	}
	// Use the widened basic-type form of the LHS as the contextual expected type for the RHS
	// so that literals (e.g. `r["x"] += 1` where `x` is float) are typed against the LHS basic
	// type rather than a possibly-singleton narrowed type.
	rhsExpectedType := semtypes.WidenToBasicTypes(lhsTy)
	var opTy semtypes.SemType
	var effect expressionEffect
	var ok bool
	switch op {
	case model.OperatorKind_ADD, model.OperatorKind_SUB:
		opTy, effect, ok = resolveAdditiveExprInner(t, chain, lhsTy, rhs, op, rhsExpectedType, pos)
	case model.OperatorKind_MUL, model.OperatorKind_DIV:
		opTy, effect, ok = resolveMultiplicativeExprInner(t, chain, lhsTy, rhs, op, rhsExpectedType, pos)
	case model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		opTy, effect, ok = resolveBitWiseExprInner(t, chain, lhsTy, rhs, op, pos)
	case model.OperatorKind_BITWISE_LEFT_SHIFT, model.OperatorKind_BITWISE_RIGHT_SHIFT, model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		opTy, effect, ok = resolveShiftExprInner(t, chain, lhsTy, rhs, op, pos)
	default:
		t.internalError(fmt.Sprintf("unexpected compound assignment operator %s", string(op)), pos)
		return expressionEffect{}, false
	}
	if !ok {
		return expressionEffect{}, false
	}
	if semtypes.ContainsBasicType(lhsTy, semtypes.Nil) || semtypes.ContainsBasicType(rhs.Type(), semtypes.Nil) {
		t.semanticError("compound assignment operands cannot be nilable", pos)
		return expressionEffect{}, false
	}
	tyCtx := t.typeContext()
	targetTy := target.Type()
	if !semtypes.IsSubtype(tyCtx, opTy, targetTy) {
		t.semanticError(common.FormatIncompatibleTypeMessage(tyCtx, targetTy, opTy), pos)
		return expressionEffect{}, false
	}
	return effect, true
}

func resolveAssignment(t typeResolver, chain *binding, s assignmentNode) (statementEffect, bool) {
	var lhsTy semtypes.SemType
	switch lhs := s.GetVariable().(type) {
	case *ast.BLangIndexBasedAccess, *ast.BLangFieldBaseAccess:
		// we don't assign to the actual container so shoud use the narrowed type for the container variable
		var lhsEffect expressionEffect
		lhsResult, ok := resolveActionOrExpression(t, chain, lhs, semtypes.SemType{})
		if !ok {
			return statementEffect{}, false
		}
		lhsTy, lhsEffect = lhsResult.ty, lhsResult.effect
		chain = sequentialChain(t, lhsEffect)
	default:
		lhsResult, ok := resolveActionOrExpression(t, nil, lhs, semtypes.SemType{})
		if !ok {
			return statementEffect{}, false
		}
		lhsTy = lhsResult.ty
	}
	rhsResult, ok := resolveActionOrExpression(t, chain, s.GetExpression(), lhsTy)
	if !ok {
		return statementEffect{}, false
	}
	// Assignment unnarrowing applies on the chain the RHS leaves behind.
	chain = sequentialChain(t, rhsResult.effect)
	if expr, ok := s.GetVariable().(ast.NodeWithSymbol); ok {
		return unnarrowSymbolAt(t, chain, expr.Symbol(), s.GetVariable().GetPosition()), true
	}
	return defaultStmtEffect(chain), true
}

func resolveStatement(t typeResolver, chain *binding, stmt ast.StatementNode) (statementEffect, bool) {
	if _, ok := stmt.(*ast.BLangBadStmt); ok {
		return defaultStmtEffect(chain), true
	}
	if scoped, ok := stmt.(ast.NodeWithScope); ok {
		if scope := scoped.Scope(); scope != nil {
			prev := t.currentScope()
			t.setCurrentScope(scope)
			defer t.setCurrentScope(prev)
		}
	}
	switch s := stmt.(type) {
	case *ast.BLangVariableDef:
		return resolveVariableDefStmt(t, chain, s)
	case *ast.BLangAssignment:
		return resolveAssignment(t, chain, s)
	case *ast.BLangCompoundAssignment:
		return resolveCompoundAssignment(t, chain, s)
	case *ast.BLangExpressionStmt:
		exprResult, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.SemType{})
		if !ok {
			return defaultStmtEffect(chain), false
		}
		return defaultStmtEffect(sequentialChain(t, exprResult.effect)), true
	// PT-TODO: extract if while out
	case *ast.BLangIf:
		exprResult, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.Boolean)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		exprEffect := exprResult.effect
		ifTrueEffect := resolveBlockStatements(t, exprEffect.ifTrue, s.Body.Stmts)
		var ifFalseEffect statementEffect
		if s.ElseStmt != nil {
			ifFalseEffect, ok = resolveStatement(t, exprEffect.ifFalse, s.ElseStmt)
			if !ok {
				return defaultStmtEffect(chain), false
			}
		} else {
			ifFalseEffect = statementEffect{exprEffect.ifFalse, false}
		}
		return mergeStatementEffects(t, ifTrueEffect, ifFalseEffect), true
	case *ast.BLangWhile:
		// A capture later in one iteration precedes an earlier use on the next, so the
		// whole loop's captures are in effect before the condition is checked.
		chain = addCaptureGroup(chain, s.CaptureGroup())
		exprResult, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.Boolean)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		exprEffect := exprResult.effect
		loopT := &loopTypeResolver{parentResolver: t}
		bodyEffect := resolveBlockStatements(loopT, exprEffect.ifTrue, s.Body.Stmts)
		resolveOnFailClause(t, chain, &s.OnFailClause)
		validateLoopAssignments(t, loopT, bodyEffect, chain)
		result := exprEffect.ifFalse
		for _, b := range loopT.breaks {
			result = mergeChains(t, result, b, semtypes.Union)
		}
		if !bodyEffect.nonCompletion {
			result = mergeChains(t, result, bodyEffect.binding, semtypes.Union)
		}
		return statementEffect{result, false}, true
	case *ast.BLangReturn:
		if s.Expr != nil {
			if _, ok := resolveActionOrExpression(t, chain, s.Expr, t.expectedReturnType()); !ok {
				return defaultStmtEffect(chain), false
			}
		}
		return statementEffect{nil, true}, true
	case *ast.BLangBlockStmt:
		return resolveBlockStatements(t, chain, s.Stmts), true
	case *ast.BLangLock:
		restoreContext := setIsolatedContext(t, true)
		effect := resolveBlockStatements(t, chain, s.Body.Stmts)
		restoreContext()
		return effect, true
	case *ast.BLangForeach:
		collectionResult, ok := resolveActionOrExpression(t, chain, s.Collection, semtypes.SemType{})
		if !ok {
			return defaultStmtEffect(chain), false
		}
		collectionTy := collectionResult.ty
		// The collection is evaluated once, before the repeated region; the loop's
		// captures take effect for everything that repeats.
		chain = addCaptureGroup(sequentialChain(t, collectionResult.effect), s.CaptureGroup())
		variable := s.VariableDef.GetVariable()
		if s.GetIsDeclaredWithVar() {
			variableTy, ok := resolveForeachVariableType(t, s.Collection, collectionTy)
			if !ok {
				return defaultStmtEffect(chain), false
			}
			updateSymbolType(t, variable, variableTy)
		} else {
			varChain, ok := resolveSimpleVariable(t, chain, variable)
			if !ok {
				return defaultStmtEffect(chain), false
			}
			chain = varChain
			if fnType, isFnType := variable.TypeNode().(*ast.BLangFunctionType); isFnType {
				if !finalizeResolvedFunctionSignature(t, fnType) {
					return defaultStmtEffect(chain), false
				}
			}
		}
		// foreach may run zero times, so the post-loop chain starts from the
		// loop-entry chain. Body completion and any break paths are merged in.
		loopT := &loopTypeResolver{parentResolver: t}
		bodyEffect := resolveBlockStatements(loopT, chain, s.Body.Stmts)
		if s.OnFailClause != nil {
			resolveOnFailClause(t, chain, s.OnFailClause)
		}
		validateLoopAssignments(t, loopT, bodyEffect, chain)
		result := chain
		for _, b := range loopT.breaks {
			result = mergeChains(t, result, b, semtypes.Union)
		}
		if !bodyEffect.nonCompletion {
			result = mergeChains(t, result, bodyEffect.binding, semtypes.Union)
		}
		return statementEffect{result, false}, true
	case *ast.BLangPanic:
		if _, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.Error); !ok {
			return defaultStmtEffect(chain), false
		}
		return statementEffect{nil, true}, true
	case *ast.BLangMatchStatement:
		return resolveMatchStatement(t, chain, s)
	case *ast.BLangBreak:
		if loopT, ok := t.(*loopTypeResolver); ok {
			loopT.recordBreak(chain)
		} else {
			t.semanticError("break statement not allowed outside loop", s.GetPosition())
		}
		return statementEffect{binding: nil, nonCompletion: true}, true
	case *ast.BLangContinue:
		if loopT, ok := t.(*loopTypeResolver); ok {
			loopT.recordContinue(chain)
		} else {
			t.semanticError("continue statement not allowed outside loop", s.GetPosition())
		}
		return statementEffect{binding: nil, nonCompletion: true}, true
	case *ast.BLangXMLNS:
		if !resolveXMLNS(t, chain, s) {
			return defaultStmtEffect(chain), false
		}
		return defaultStmtEffect(chain), true
	default:
		t.internalError(fmt.Sprintf("unhandled statement type: %T", stmt), stmt.GetPosition())
		return defaultStmtEffect(chain), false
	}
}

func resolveXMLNS(t typeResolver, chain *binding, decl *ast.BLangXMLNS) bool {
	uriExpr := decl.GetNamespaceURI()
	if uriExpr == nil {
		t.internalError("xmlns declaration missing URI", decl.GetPosition())
		return false
	}
	uriResult, ok := resolveActionOrExpression(t, chain, uriExpr, semtypes.String)
	if !ok {
		return false
	}
	if !semtypes.IsSubtype(t.typeContext(), uriResult.ty, semtypes.String) {
		t.semanticError("xmlns URI must be a string", uriExpr.GetPosition())
		return false
	}
	if offender, ok := isConstantExpression(t, uriExpr); !ok {
		t.semanticError(notConstantExpressionMessage, offender.GetPosition())
		return false
	}
	value, ok := foldConstant(t, uriExpr)
	if !ok {
		return false
	}
	uri, isString := value.(string)
	if !isString {
		t.semanticError("xmlns URI must be a string", uriExpr.GetPosition())
		return false
	}
	if uri == "" {
		t.semanticError("XML namespace URI cannot be empty", decl.GetPosition())
		return false
	}
	if err := t.compilerContext().SetXMLNamespaceURI(decl.Symbol(), uri); err != nil {
		t.internalError(err.Error(), decl.GetPosition())
		return false
	}
	t.setSymbolType(decl.Symbol(), semtypes.String)
	return true
}

func resolveOnFailClause(t typeResolver, chain *binding, clause *ast.BLangOnFailClause) {
	if clause.VariableDefinitionNode != nil {
		varDef := clause.VariableDefinitionNode
		variable := varDef.GetVariable()
		resolveSimpleVariable(t, chain, variable)
	}
	if clause.Body != nil {
		resolveBlockStatements(t, chain, clause.Body.Stmts)
	}
}

func resolveFunctionSignature(t typeResolver, chain *binding, fn *ast.BLangFunction, depth int) (semtypes.SemType, *binding, bool) {
	fnSym := t.getSymbol(fn.Symbol())
	if depSym, ok := fnSym.(model.DependentlyTypedFunctionSymbol); ok {
		return resolveDependentlyTypedFunctionSignature(t, chain, fn, depSym, depth)
	}
	if _, ok := fnSym.(*model.OpaqueFunctionSymbol); ok {
		return resolveOpaqueFunctionSignature(t, chain, fn, depth)
	}
	if ty := t.symbolType(fn.Symbol()); !semtypes.IsZero(ty) {
		return ty, chain, !semtypes.IsNever(ty)
	}
	fnSymbol := fnSym.(model.FunctionSymbol)
	fnType, chain, ok := resolveInvokableSignature(t, chain, fn, fnSymbol, fn.GetParameters(), depth)
	if !ok {
		// A function whose signature does not resolve has type never.
		t.setSymbolType(fn.Symbol(), semtypes.Never)
		return semtypes.SemType{}, chain, false
	}
	if !finalizeResolvedFunctionSignature(t, fn) {
		return semtypes.SemType{}, chain, false
	}
	return fnType, chain, true
}

func finalizeResolvedFunctionSignature(t typeResolver, fn ast.FunctionSignature) bool {
	if fnType, ok := fn.(*ast.BLangFunctionType); ok && fnType.IsAnyFunction() {
		return true
	}
	sig, ref, ok := functionSignatureForNode(t, fn)
	if !ok {
		return false
	}
	params := fn.Parameters()
	paramTypes := make([]semtypes.SemType, len(params))
	for i, param := range params {
		typeDesc := param.Type()
		if typeDesc == nil {
			t.internalError("parameter has no type descriptor", param.GetPosition())
			return false
		}
		paramTypes[i] = typeDesc.Type()
	}
	return finalizeFunctionSignature(t, fn, sig, ref, paramTypes)
}

func finalizeFunctionSignature(t typeResolver, fn ast.FunctionSignature, sig model.UntypedFunctionSignature, ref model.FunctionSignatureRef, paramTypes []semtypes.SemType) bool {
	setDefaultableParamFnSignatures(t, sig, paramTypes, fn.GetPosition())
	return validateIncludedRecordParams(t, fn, ref, sig)
}

type symbolFunctionSignature interface {
	ast.FunctionSignature
	Symbol() model.SymbolRef
}

func functionSignatureForNode(t typeResolver, fn ast.FunctionSignature) (model.UntypedFunctionSignature, model.FunctionSignatureRef, bool) {
	if fnType, ok := fn.(*ast.BLangFunctionType); ok {
		ref := fnType.SignatureRef()
		return t.functionSignatureByRef(ref), ref, true
	}
	owner := fn.(symbolFunctionSignature).Symbol()
	ref, ok := t.functionSignatureRef(owner)
	if !ok {
		t.internalError("function signature not found", fn.GetPosition())
		return model.UntypedFunctionSignature{}, 0, false
	}
	return t.functionSignatureByRef(ref), ref, true
}

func validateIncludedRecordParams(t typeResolver, fn ast.FunctionSignature, ref model.FunctionSignatureRef, sig model.UntypedFunctionSignature) bool {
	requiredParams := fn.Parameters()
	params := make([]includedRecordParamData, len(requiredParams))
	for i, param := range requiredParams {
		params[i] = includedRecordParamData{typeDesc: param.Type(), pos: param.GetPosition()}
	}
	restName := ""
	if restParam := fn.RestParameter(); restParam != nil {
		restName = restParam.ParamName()
	}
	return validateIncludedRecordParamMetadata(t, ref, sig, params, restName)
}

type includedRecordParamData struct {
	typeDesc ast.BType
	pos      diagnostics.Location
}

func validateIncludedRecordParamMetadata(t typeResolver, ref model.FunctionSignatureRef, sig model.UntypedFunctionSignature, params []includedRecordParamData, restName string) bool {
	paramNames := sig.ParamNames
	fieldOrigin := make(map[string]int)
	includedRecords := make([]*model.IncludedRecordMetadata, len(sig.ParamNames))
	updated := false
	for i, param := range params {
		if i >= len(sig.ParamFlags) || sig.ParamFlags[i]&model.ParamFlagIncludedRecordParam == 0 {
			continue
		}
		udt, ok := param.typeDesc.(*ast.BLangUserDefinedType)
		if !ok {
			t.semanticError("included record parameter must be a record type", param.pos)
			return false
		}
		recRef := udt.Symbol()
		t.ensureResolved(recRef, 0)
		recSym, ok := t.getSymbol(recRef).(*model.RecordSymbol)
		if !ok {
			t.semanticError("included record parameter must be a record type", param.pos)
			return false
		}
		metadata := &model.IncludedRecordMetadata{}
		if rest, ok := recSym.RestField(); ok && !semtypes.IsNever(rest.MemberType()) {
			metadata.IsOpen = true
		}
		for name, field := range recSym.Fields() {
			if semtypes.IsNever(field.MemberType()) {
				metadata.NeverFields = append(metadata.NeverFields, name)
				continue
			}
			metadata.RequiredFields = append(metadata.RequiredFields, name)
			for j, pname := range paramNames {
				if j == i {
					continue
				}
				if pname == name {
					t.semanticError(
						fmt.Sprintf("parameter '%s' conflicts with field of included record parameter '%s'", name, paramNames[i]),
						param.pos,
					)
					return false
				}
			}
			if restName == name {
				t.semanticError(
					fmt.Sprintf("parameter '%s' conflicts with field of included record parameter '%s'", name, paramNames[i]),
					param.pos,
				)
				return false
			}
			if prev, seen := fieldOrigin[name]; seen {
				t.semanticError(
					fmt.Sprintf("duplicate field '%s' in included record parameters '%s' and '%s'", name, paramNames[prev], paramNames[i]),
					param.pos,
				)
				return false
			}
			fieldOrigin[name] = i
		}
		includedRecords[i] = metadata
		updated = true
	}
	if updated {
		t.updateFunctionSignatureIncludedRecords(ref, includedRecords)
	}
	return true
}

// resolveOpaqueFunctionSignature types the lang library source declaration of an opaque
// function. The declaration fixes the parameter names, their declared types and their
// defaults; it does not fix the type of a call, which the function's monomorphizer
// decides per call site, so the symbol itself is never given a type. The parameters are
// still resolved: their declared types are what a default expression is checked against
// and what the $default$N provider generated for it takes and returns.
func resolveOpaqueFunctionSignature(t typeResolver, chain *binding, fn *ast.BLangFunction, depth int) (semtypes.SemType, *binding, bool) {
	restoreContext := setIsolatedContext(t, fn.IsIsolated())
	defer restoreContext()
	params := fn.GetParameters()
	for i := range params {
		resolveSimpleVariableInner(t, nil, &params[i], depth+1)
	}
	if restParam := fn.GetRestParam(); restParam != nil {
		resolveSimpleVariableInner(t, nil, restParam, depth+1)
	}
	if retTd := fn.GetReturnTypeDescriptor(); retTd != nil {
		if _, _, ok := resolveBType(t, chain, retTd, depth+1); !ok {
			return semtypes.SemType{}, chain, false
		}
	}
	if !finalizeResolvedFunctionSignature(t, fn) {
		return semtypes.SemType{}, chain, false
	}
	return semtypes.Never, chain, true
}

func resolveDependentlyTypedFunctionSignature(t typeResolver, chain *binding, fn common.FunctionDecl, sym model.DependentlyTypedFunctionSymbol, depth int) (semtypes.SemType, *binding, bool) {
	params := fn.GetParameters()
	paramTypes := make([]semtypes.SemType, len(params))
	paramsByName := make(map[string]param, len(params))
	for i := range params {
		p := &params[i]
		resolveSimpleVariableInner(t, nil, p, depth+1)
		if fnType, ok := p.TypeNode().(*ast.BLangFunctionType); ok {
			if !finalizeResolvedFunctionSignature(t, fnType) {
				return semtypes.SemType{}, chain, false
			}
		}
		paramTypes[i] = t.symbolType(p.Symbol())
		paramsByName[p.GetName().GetValue()] = param{index: i, ty: paramTypes[i]}
	}
	retTd := fn.GetReturnTypeDescriptor()
	if retTd == nil {
		t.internalError("dependently-typed function has no return type descriptor", fn.GetPosition())
		return semtypes.SemType{}, chain, false
	}
	retOp, chain, ok := buildReturnTypeOp(t, chain, paramsByName, retTd)
	if !ok {
		t.internalError("failed to build return type op for dependently-typed function", fn.GetPosition())
		return semtypes.SemType{}, chain, false
	}
	sym.SetParamTypes(paramTypes)
	sym.SetReturnType(retOp)
	if !finalizeResolvedFunctionSignature(t, fn) {
		return semtypes.SemType{}, chain, false
	}
	return semtypes.Never, chain, true
}

// setDefaultableParamFnSignatures populates the signature of each non-typedesc
// default-provider function. The signature is (paramTypes[:i]) -> paramTypes[i].
func setDefaultableParamFnSignatures(t typeResolver, sig model.UntypedFunctionSignature, paramTypes []semtypes.SemType, loc diagnostics.Location) {
	for i := range paramTypes {
		dp, ok := sig.DefaultableParam(i)
		if !ok {
			continue
		}
		if dp.Kind == model.DefaultableParamKindInferredTypedesc {
			continue
		}
		defaultSig := model.TypedFunctionSignature{
			ParamTypes: paramTypes[:i],
			ReturnType: paramTypes[i],
		}
		t.getSymbol(dp.Symbol).(model.FunctionSymbol).SetTypedSignature(defaultSig)
		t.setSymbolType(dp.Symbol, typeFromFunctionSignature(t, defaultSig))
		if _, ok := t.functionSignatureRef(dp.Symbol); ok {
			continue
		}
		params := make([]model.Param, i)
		for j := range params {
			params[j] = model.Param{Name: sig.ParamNames[j], Flag: sig.ParamFlags[j]}
		}
		ref := t.allocateFunctionSignature(params, false)
		if !t.associateFunctionSignature(dp.Symbol, ref) {
			t.internalError("function signature already set", loc)
		}
	}
}

type param struct {
	index int
	ty    semtypes.SemType
}

// buildReturnTypeOp translates a return-type-descriptor AST node into a TypeOp tree.
// A user-defined-type node whose name matches a typedesc parameter becomes a RefTypeOp.
// Array, union, and intersection nodes recurse. Everything else is resolved to a concrete semtype
// and wrapped in an IdentityTypeOp.
func buildReturnTypeOp(t typeResolver, chain *binding, params map[string]param, node ast.BLangNode) (model.TypeOp, *binding, bool) {
	switch n := node.(type) {
	case *ast.BLangReturnTypeDescriptor:
		return buildReturnTypeOp(t, chain, params, n.TypeDescriptor)
	case *ast.BLangUnionTypeNode:
		lhs, chain, ok := buildReturnTypeOp(t, chain, params, n.Lhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, chain, false
		}
		rhs, chain, ok := buildReturnTypeOp(t, chain, params, n.Rhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, chain, false
		}
		return &model.BinaryTypeOp{Kind: model.TypeOpUnion, Lhs: lhs, Rhs: rhs}, chain, true
	case *ast.BLangIntersectionTypeNode:
		lhs, chain, ok := buildReturnTypeOp(t, chain, params, n.Lhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, chain, false
		}
		rhs, chain, ok := buildReturnTypeOp(t, chain, params, n.Rhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, chain, false
		}
		return &model.BinaryTypeOp{Kind: model.TypeOpIntersection, Lhs: lhs, Rhs: rhs}, chain, true
	case *ast.BLangArrayType:
		element, chain, ok := buildReturnTypeOp(t, chain, params, n.Elemtype.TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, chain, false
		}
		for i := len(n.Sizes); i > 0; i-- {
			lengthExpr := n.Sizes[i-1]
			if lengthExpr == nil {
				element = &model.ArrayTypeOp{Element: element, IsOpen: true}
				continue
			}
			length, ok := resolveFixedArraySize(t, lengthExpr)
			if !ok {
				return nil, chain, false
			}
			element = &model.ArrayTypeOp{Element: element, Length: length}
		}
		return element, chain, true
	case *ast.BLangUserDefinedType:
		if n.PkgAlias.GetValue() == "" {
			if p, ok := params[n.TypeName.Value]; ok && semtypes.IsSubtype(t.typeContext(), p.ty, semtypes.Typedesc) {
				return &model.RefTypeOp{Index: p.index}, chain, true
			}
		}
		ty, chain, ok := resolveBType(t, chain, n, 0)
		if !ok {
			return nil, chain, false
		}
		return &model.IdentityTypeOp{Type: ty}, chain, true
	default:
		ty, chain, ok := resolveBType(t, chain, node.(ast.BType), 0)
		if !ok {
			return nil, chain, false
		}
		return &model.IdentityTypeOp{Type: ty}, chain, true
	}
}

func resolveLambdaFunctionExpr(t typeResolver, chain *binding, e *ast.BLangLambdaFunction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if e.HasInferredParams() {
		return resolveInferredLambdaFunctionExpr(t, chain, e, expectedType)
	}
	if t.isEphemeral() {
		return resolveEphemeralLambdaFunctionExpr(t, chain, e)
	}
	fnType, chain, ok := resolveFunctionSignature(t, chain, e.Function, 0)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	// Create a function type resolver for the lambda so expectedReturnType() is correct
	fnSym := t.getSymbol(e.Function.Symbol()).(model.FunctionSymbol)
	ft := &functionTypeResolver{
		atomSideTableBase: newAtomSideTableBase(),
		parentResolver:    t,
		tyCtx:             semtypes.ContextFrom(t.typeEnv()),
		retTy:             fnSym.TypedSignature().ReturnType,
		implicitImports:   make(map[string]ast.BLangImportPackage),
		monoCounters:      make(map[string]int),
		xmlStepOwner:      fnSym.Name(),
		xmlStepCounters:   resolverXMLStepCounters(t),
		scope:             e.Function.Scope(),
		context:           resolverContextForFunction(fnSym.TypedSignature().Flags&model.FuncSymbolFlagIsolated != 0),
		ephemeralState:    resolverEphemeralState(t),
	}

	// Constructing the lambda captures its free variables. The body is checked
	// behind a function boundary with those captures already in effect.
	outerChain := addCaptureGroup(chain, e.Function.CaptureGroup())
	boundaryChain := &binding{flags: bindingFlagFunctionBoundary, prev: outerChain}

	switch body := e.Function.Body.(type) {
	case *ast.BLangBlockFunctionBody:
		resolveBlockStatements(ft, boundaryChain, body.Stmts)
	case *ast.BLangExprFunctionBody:
		if _, ok := resolveActionOrExpression(ft, boundaryChain, body.Expr, ft.retTy); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}

	setNodeType(t, e, fnType)
	return fnType, defaultExpressionEffect(outerChain), true
}

// resolveEphemeralLambdaFunctionExpr types an explicit lambda from its signature alone. Its body and parameter
// defaults are resolved by the final resolution; a candidate trial only needs the effect of constructing the lambda,
// which is the capture of its defaults' and its body's capture groups.
func resolveEphemeralLambdaFunctionExpr(t typeResolver, chain *binding, e *ast.BLangLambdaFunction) (semtypes.SemType, expressionEffect, bool) {
	fnType, chain, ok := functionSignatureType(t, chain, e.Function)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	return fnType, defaultExpressionEffect(addCaptureGroup(chain, e.Function.CaptureGroup())), true
}

func resolveInferredLambdaFunctionExpr(t typeResolver, chain *binding, e *ast.BLangLambdaFunction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	cx := t.typeContext()
	functionContext := semtypes.Intersect(expectedType, semtypes.Function)
	if semtypes.IsZero(expectedType) || semtypes.IsEmpty(cx, functionContext) {
		t.semanticError("cannot infer anonymous function parameter types without an expected function type", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if semtypes.IsSameType(cx, functionContext, semtypes.Function) {
		t.semanticError("cannot infer types of the arrow expression with unknown invokable type", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	paramListTy := semtypes.FunctionParamListType(cx, functionContext)
	params := e.Function.GetParameters()
	arityTypes := make([]semtypes.SemType, len(params))
	for i := range arityTypes {
		arityTypes[i] = semtypes.Val
	}
	arityDef := semtypes.NewListDefinition()
	arityTy := arityDef.Define(t.typeEnv(), arityTypes, semtypes.ListMutability(semtypes.CellMutabilityNone))
	if semtypes.IsEmpty(cx, semtypes.Intersect(paramListTy, arityTy)) {
		t.semanticError("anonymous function parameters are incompatible with the expected function type", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	paramTypes := make([]semtypes.SemType, len(params))
	for i := range params {
		paramTy := semtypes.ListMemberTypeInnerVal(cx, paramListTy, semtypes.IntConst(int64(i)))
		if semtypes.IsZero(paramTy) {
			t.semanticError("cannot infer anonymous function parameter type", params[i].GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if !t.isEphemeral() {
			updateSymbolType(t, &params[i], paramTy)
		}
		paramTypes[i] = paramTy
	}

	argListDef := semtypes.NewListDefinition()
	argListTy := argListDef.Define(t.typeEnv(), paramTypes, semtypes.ListMutability(semtypes.CellMutabilityNone))
	expectedReturnTy := semtypes.FunctionReturnType(cx, functionContext, argListTy)
	if semtypes.IsZero(expectedReturnTy) {
		t.semanticError("anonymous function parameters are incompatible with the expected function type", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	flags := model.FuncSymbolFlags(0)
	if semtypes.IsSubtype(cx, functionContext, semtypes.CreateIsolatedFn(cx)) {
		flags = model.FuncSymbolFlagIsolated
	}
	fnSym := t.getSymbol(e.Function.Symbol()).(model.FunctionSymbol)
	sig := model.TypedFunctionSignature{
		ParamTypes:    paramTypes,
		ReturnType:    expectedReturnTy,
		RestParamType: semtypes.Never,
		Flags:         flags,
	}
	setFunctionTypedSignature(t, fnSym, sig)
	ft := &functionTypeResolver{
		atomSideTableBase: newAtomSideTableBase(),
		parentResolver:    t,
		tyCtx:             semtypes.ContextFrom(t.typeEnv()),
		retTy:             expectedReturnTy,
		implicitImports:   make(map[string]ast.BLangImportPackage),
		monoCounters:      make(map[string]int),
		xmlStepOwner:      fnSym.Name(),
		xmlStepCounters:   resolverXMLStepCounters(t),
		scope:             e.Function.Scope(),
		context:           resolverContextForFunction(flags&model.FuncSymbolFlagIsolated != 0),
		ephemeralState:    resolverEphemeralState(t),
	}
	outerChain := addCaptureGroup(chain, e.Function.CaptureGroup())
	boundaryChain := &binding{flags: bindingFlagFunctionBoundary, prev: outerChain}
	if t.isEphemeral() {
		boundaryChain = bindInferredParams(t, boundaryChain, params, paramTypes)
	}
	body := e.Function.Body.(*ast.BLangExprFunctionBody)
	returnResult, ok := resolveActionOrExpression(ft, boundaryChain, body.Expr, expectedReturnTy)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	returnTy := returnResult.ty

	sig.ReturnType = returnTy
	setFunctionTypedSignature(t, fnSym, sig)
	fnType := typeFromFunctionSignature(t, sig)
	setLambdaSymbolType(t, e.Function, fnType)
	if !semtypes.IsSubtype(cx, fnType, expectedType) {
		t.semanticError(common.FormatIncompatibleTypeMessage(cx, expectedType, fnType), e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, e, fnType)
	return fnType, defaultExpressionEffect(outerChain), true
}

// bindInferredParams gives each inferred parameter its type for the current candidate trial on the body's chain,
// leaving the parameter symbols themselves untouched.
func bindInferredParams(t typeResolver, chain *binding, params []ast.BLangVariable, paramTypes []semtypes.SemType) *binding {
	for i := range params {
		ref := params[i].Symbol()
		chain = &binding{ref: ref, narrowedSymbol: narrowSymbol(t, ref, paramTypes[i]), prev: chain, flags: bindingFlagInferredParam}
	}
	return chain
}

func resolveTypeData(t typeResolver, chain *binding, typeData *ast.TypeData) (semtypes.SemType, *binding, bool) {
	if typeData.TypeDescriptor == nil {
		return typeData.Type, chain, true
	}
	ty, chain, ok := resolveBType(t, chain, typeData.TypeDescriptor.(ast.BType), 0)
	if !ok {
		return semtypes.SemType{}, chain, false
	}
	setTypeDataType(t, typeData, ty)
	return ty, chain, true
}

func resolveTypeDefinition(t typeResolver, defn *ast.BLangTypeDefinition, depth int) bool {
	if ty := t.symbolType(defn.Symbol()); !semtypes.IsZero(ty) {
		return true
	}
	if depth == defn.GetCycleDepth() {
		t.semanticError(fmt.Sprintf("invalid cycle detected for type definition %s", defn.GetName().GetValue()), defn.GetPosition())
		return false
	}
	defn.SetCycleDepth(depth)
	typeDescriptor := defn.GetTypeData().TypeDescriptor.(ast.BType)
	var semType semtypes.SemType
	var ok bool
	if objectType, isObjectType := typeDescriptor.(*ast.BLangObjectType); isObjectType {
		semType, _, ok = resolveObjectType(t, nil, objectType, depth, defn.Symbol())
		if ok {
			setResolvedTypeData(t, objectType, semType)
		}
	} else {
		semType, _, ok = resolveBType(t, nil, typeDescriptor, depth)
	}
	if ok {
		semType, ok = resolveDistinctTypeDefinition(t, defn, semType)
	}
	if !ok {
		return false
	}
	if semtypes.IsZero(t.symbolType(defn.Symbol())) {
		t.setSymbolType(defn.Symbol(), semType)
		defn.SetCycleDepth(-1)
		typeData := defn.GetTypeData()
		typeData.Type = semType
		setTypeData(t, defn, typeData)
		addInclusionsToTypeSymbol(t, defn)
		name := defn.GetName().GetValue()
		pos := defn.GetPosition()
		t.ensureNotEmpty(semType, func() {
			t.semanticError(fmt.Sprintf("type definition %s is empty", name), pos)
		})
		return true
	}
	return true
}

func resolveClassTypeDefinition(t typeResolver, classDef *ast.BLangClassDefinition, depth int) bool {
	if ty := t.symbolType(classDef.Symbol()); !semtypes.IsZero(ty) {
		return true
	}
	if depth == classDef.GetCycleDepth() {
		t.semanticError(fmt.Sprintf("invalid cycle detected for type definition %s", classDef.GetName().GetValue()), classDef.GetPosition())
		return false
	}
	classDef.SetCycleDepth(depth)
	_, ok := resolveClassDefinitionType(t, classDef, depth)
	return ok
}

func resolveDistinctTypeDefinition(t typeResolver, typeDef *ast.BLangTypeDefinition, semType semtypes.SemType) (semtypes.SemType, bool) {
	switch typeDesc := typeDef.GetTypeData().TypeDescriptor.(type) {
	case *ast.BLangObjectType:
		return appendDistinctObjectAtoms(t, semType, typeDef.Symbol(), typeDesc.Inclusions), true
	case *ast.BLangErrorTypeNode:
		return appendDistinctErrorAtoms(t, semType, typeDef), true
	case *ast.BLangUserDefinedType:
		if !typeDef.IsDistinct() {
			return semType, true
		}
		parent := t.getSymbol(typeDesc.Symbol())
		switch parent.(type) {
		case *model.ErrorTypeSymbol:
			return appendDistinctAliasAtoms(t, semType, typeDef.Symbol(), typeDesc.Symbol(), semtypes.ErrorDistinct), true
		case model.ObjectType:
			return appendDistinctAliasAtoms(t, semType, typeDef.Symbol(), typeDesc.Symbol(), semtypes.ObjectDefinitionDistinct), true
		default:
			t.semanticError("only object and error types can be distinct", typeDef.GetPosition())
			return semtypes.SemType{}, false
		}
	default:
		return semType, true
	}
}

func appendDistinctAliasAtoms(t typeResolver, semType semtypes.SemType, childRef, parentRef model.SymbolRef, atom func(int) semtypes.SemType) semtypes.SemType {
	child := t.getSymbol(childRef).(distinctTypeSymbol)
	parent := t.getSymbol(parentRef).(distinctTypeSymbol)

	idsByID := make(map[int]bool)
	for _, id := range parent.DistinctTypeIDs() {
		idsByID[id] = true
	}
	for _, id := range child.DistinctTypeIDs() {
		idsByID[id] = true
	}

	ids := make([]int, 0, len(idsByID))
	for id := range idsByID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	child.SetDistinctTypeIDs(ids)
	return intersectDistinctAtoms(semType, ids, atom)
}

func intersectDistinctAtoms(semType semtypes.SemType, ids []int, atom func(int) semtypes.SemType) semtypes.SemType {
	for _, id := range ids {
		semType = semtypes.Intersect(semType, atom(id))
	}
	return semType
}

func appendDistinctObjectAtoms(t typeResolver, semType semtypes.SemType, symbol model.SymbolRef, inclusions []model.SymbolRef) semtypes.SemType {
	carrier := t.getSymbol(symbol).(model.ObjectType)

	seen := make(map[int]bool)
	var ids []int
	addIDs := func(obj model.ObjectType) {
		for _, id := range obj.DistinctTypeIDs() {
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}

	addIDs(carrier)
	for _, inc := range inclusions {
		addIDs(t.getSymbol(inc).(model.ObjectType))
	}

	carrier.SetDistinctTypeIDs(ids)
	return intersectDistinctAtoms(semType, ids, semtypes.ObjectDefinitionDistinct)
}

func appendDistinctErrorAtoms(t typeResolver, semType semtypes.SemType, typeDef *ast.BLangTypeDefinition) semtypes.SemType {
	if !typeDef.IsDistinct() {
		return semType
	}
	carrier := t.getSymbol(typeDef.Symbol()).(*model.ErrorTypeSymbol)
	return intersectDistinctAtoms(semType, carrier.DistinctTypeIDs(), semtypes.ErrorDistinct)
}

// addInclusionsToTypeSymbol addes all the inclusions (both transitive and direct) to the type symbol
// This should be called only after resolving the underlying type
func addInclusionsToTypeSymbol(t typeResolver, defn *ast.BLangTypeDefinition) {
	var members []model.InclusionMember
	typeDesc := defn.GetTypeData().TypeDescriptor
	switch td := typeDesc.(type) {
	case *ast.BLangRecordType:
		members = recordTypeMembers(t, td)
	case *ast.BLangObjectType:
		members = objectTypeMembers(t, td)
	case *ast.BLangUserDefinedType:
		copyAliasMembersToTypeSymbol(t, defn.Symbol(), td.Symbol())
		return
	default:
		return
	}
	carrier := getMemberCarrierFromDefn(t, defn.Symbol(), defn.GetPosition())
	if carrier == nil {
		return
	}
	for _, m := range members {
		carrier.AddMember(m)
	}
}

func copyAliasMembersToTypeSymbol(t typeResolver, aliasRef, targetRef model.SymbolRef) {
	aliasCarrier, ok := t.getSymbol(aliasRef).(model.MemberCarrier)
	if !ok {
		return
	}
	if !t.ensureResolved(targetRef, 0) {
		return
	}
	targetCarrier := t.getSymbol(targetRef).(model.MemberCarrier)
	for _, m := range targetCarrier.Members() {
		aliasCarrier.AddMember(m)
	}
}

func addInclusionsToClassSymbol(t typeResolver, classDef *ast.BLangClassDefinition) {
	carrier := getMemberCarrierFromDefn(t, classDef.Symbol(), classDef.GetPosition())
	if carrier == nil {
		return
	}
	for _, m := range classMembers(t, classDef) {
		carrier.AddMember(m)
	}
}

func getMemberCarrierFromDefn(t typeResolver, ref model.SymbolRef, pos diagnostics.Location) model.MemberCarrier {
	sym := t.getSymbol(ref)
	switch s := sym.(type) {
	case *model.RecordSymbol:
		return s
	case *model.ObjectTypeSymbol:
		return s
	case model.ClassSymbol:
		return s
	default:
		t.internalError("unexpected type definition", pos)
		return nil
	}
}

func getMemberCarrier(t typeResolver, ref model.SymbolRef) model.MemberCarrier {
	sym := t.getSymbol(ref)
	switch s := sym.(type) {
	case *model.RecordSymbol:
		return s
	case *model.ObjectTypeSymbol:
		return s
	case model.ClassSymbol:
		return s
	default:
		t.internalError("symbol is not a member carrier", diagnostics.NewBuiltinLocation())
		return nil
	}
}

// recordTypeMembers accumulates members both added by type inclusion and defined in the record type itself
func recordTypeMembers(t typeResolver, td *ast.BLangRecordType) []model.InclusionMember {
	var members []model.InclusionMember
	directFields := make(map[string]bool)

	// Add direct fields, leaving out those whose type failed to resolve
	for name, field := range td.FieldPtrs() {
		if semtypes.IsZero(field.Type.Type()) {
			continue
		}
		directFields[name] = true
		fd := createFieldDescriptor(name, *field)
		members = append(members, &fd)
	}

	// Collect transitive members from included types
	for _, symRef := range td.Inclusions {
		incSym := getMemberCarrier(t, symRef)
		if incSym == nil {
			t.internalError("failed to find included symbol", td.GetPosition())
			continue
		}
		for _, m := range incSym.Members() {
			switch member := m.(type) {
			case *model.FieldDescriptor:
				if directFields[member.MemberName()] {
					continue
				}
				members = append(members, member)
			case *model.RestTypeDescriptor:
				members = append(members, member)
			default:
				t.internalError("unexpected member kind", td.GetPosition())
			}
		}
	}

	// Add rest type from this record's own rest type
	if td.RestType != nil {
		rd := model.NewRestTypeDescriptor()
		rd.SetMemberType(td.RestType.Type())
		members = append(members, &rd)
	}
	return members
}

// objectTypeMembers accumulate members both added by type inclusion and defined in the type desc itself
func objectTypeMembers(t typeResolver, td *ast.BLangObjectType) []model.InclusionMember {
	var members []model.InclusionMember
	// Collect transitive members from included types
	for _, symRef := range td.Inclusions {
		incSym := getMemberCarrier(t, symRef)
		if incSym == nil {
			t.internalError("failed to find included symbol", td.GetPosition())
			return nil
		}
		members = append(members, incSym.Members()...)
	}
	// Add direct members, leaving out those that failed to resolve
	for m := range td.Members() {
		switch member := m.(type) {
		case *ast.BObjectField:
			if semtypes.IsZero(member.Ty.Type()) {
				continue
			}
			fd := objectFieldDescriptor(member)
			members = append(members, &fd)
		case *ast.BMethodDecl:
			if semtypes.IsZero(member.Type()) {
				continue
			}
			md := methodDescriptor(member, member.Symbol())
			members = append(members, &md)
		}
	}
	return members
}

// classMembers accumulate members both added by type inclusion and defined in the class decl itself.
// A field the class declares itself replaces an included field of the same name. An included field
// has no default in the class, since the including class must initialize it.
func classMembers(t typeResolver, classDef *ast.BLangClassDefinition) []model.InclusionMember {
	var members []model.InclusionMember
	fields := typedFields(classDef.Fields)
	directFields := make(map[string]bool)
	for _, field := range fields {
		directFields[field.Name.GetValue()] = true
	}
	// Collect transitive members from included types
	for _, symRef := range classDef.Inclusions {
		incSym := getMemberCarrier(t, symRef)
		if incSym == nil {
			t.internalError("failed to find included symbol", classDef.GetPosition())
			return nil
		}
		for _, m := range incSym.Members() {
			field, ok := m.(*model.FieldDescriptor)
			if !ok {
				members = append(members, m)
				continue
			}
			if directFields[field.MemberName()] {
				continue
			}
			members = append(members, field.WithoutDefault())
		}
	}
	// Add direct members
	for _, field := range fields {
		fd := common.ClassFieldDescriptor(field, t.symbolType(field.Symbol()))
		members = append(members, &fd)
	}
	for name, method := range resolvedMethods(t, classDef.Methods) {
		md := classMethodDescriptor(t, name, method)
		members = append(members, &md)
	}
	return members
}

func objectFieldDescriptor(field *ast.BObjectField) model.FieldDescriptor {
	var flags model.FieldDescriptorFlag
	if field.IsReadonly() {
		flags |= model.FieldDescriptorReadonly
	}
	fd := model.NewFieldDescriptor(field.Name(), flags, field.IsPublic())
	fd.SetMemberType(field.Ty.Type())
	return fd
}

func methodDescriptor(method *ast.BMethodDecl, fnRef model.SymbolRef) model.MethodDescriptor {
	kind := model.InclusionMemberKindMethod
	switch method.MemberKind() {
	case ast.ObjectMemberKindRemoteMethod:
		kind = model.InclusionMemberKindRemoteMethod
	case ast.ObjectMemberKindResourceMethod:
		kind = model.InclusionMemberKindResourceMethod
	case ast.ObjectMemberKindField, ast.ObjectMemberKindMethod:
		// Preserve the ordinary method descriptor kind.
	}
	md := model.NewMethodDescriptor(method.Name(), kind, method.IsPublic(), fnRef)
	md.SetMemberType(method.Type())
	return md
}

func classMethodDescriptor(t typeResolver, name string, method *ast.BLangFunction) model.MethodDescriptor {
	kind := model.InclusionMemberKindMethod
	if method.IsRemote() {
		kind = model.InclusionMemberKindRemoteMethod
	} else if method.IsResource() {
		kind = model.InclusionMemberKindResourceMethod
	}
	md := model.NewMethodDescriptor(name, kind, method.IsPublic(), method.Symbol())
	md.SetMemberType(methodMemberType(t, method.Symbol()))
	return md
}

func createFieldDescriptor(name string, field ast.BField) model.FieldDescriptor {
	var flags model.FieldDescriptorFlag
	if field.IsReadonly() {
		flags |= model.FieldDescriptorReadonly
	}
	if field.IsOptional() {
		flags |= model.FieldDescriptorOptional
	}
	if field.Default != nil {
		flags |= model.FieldDescriptorHasDefault
	}
	fd := model.NewFieldDescriptor(name, flags, true)
	fd.SetMemberType(field.Type.Type())
	if field.Default != nil {
		fd.DefaultFnRef = field.Default.FnRef
	}
	return fd
}

func resolveClassDefinitionType(t typeResolver, classDef *ast.BLangClassDefinition, depth int) (semtypes.SemType, bool) {
	if classDef.Definition != nil {
		// Recursive self-reference while the surrounding class is still being
		// resolved. Return the partial type so callers can refer to it.
		recTy := classDef.Definition.GetSemType(t.typeEnv())
		semType := appendDistinctObjectAtoms(t, recTy, classDef.Symbol(), classDef.Inclusions)
		t.setSymbolType(classDef.Symbol(), semType)
		return semType, true
	}

	isClient := classDef.IsClient()
	isService := classDef.IsService()
	od := semtypes.NewObjectDefinition()
	classDef.Definition = &od

	semType, ok := finishResolveObjectDefinitionType(t, &od, t.symbolName(classDef.Symbol()), classDef.Fields, classDef.Methods,
		classDef.ResourceMethods, classDef.InitFunction, classDef.Inclusions, classDef.InclusionPositions, classDef.GetPosition(),
		depth, classDef.IsIsolated(), classDef.IsReadonly(), isClient, isService, classDef.Symbol())
	if !ok {
		return semtypes.SemType{}, false
	}

	t.setSymbolType(classDef.Symbol(), semType)
	classDef.SetCycleDepth(-1)
	typeData := classDef.GetTypeData()
	typeData.Type = semType
	setTypeData(t, classDef, typeData)
	addInclusionsToClassSymbol(t, classDef)
	registerClassMethodTable(t, semType, classDef)
	if selfRef, ok := classDef.Scope().GetSymbol("self"); ok {
		t.setSymbolType(selfRef, semType)
	}
	name := classDef.GetName().GetValue()
	pos := classDef.GetPosition()
	t.ensureNotEmpty(semType, func() {
		t.semanticError(fmt.Sprintf("class definition %s is empty", name), pos)
	})
	return semType, true
}

func resolveServiceType(t typeResolver, svc *ast.BLangService, depth int, attachPointBound semtypes.SemType) {
	typeData := svc.GetTypeData()
	var serviceTy semtypes.SemType
	if typeData.TypeDescriptor != nil {
		var ok bool
		serviceTy, _, ok = resolveBType(t, nil, typeData.TypeDescriptor.(ast.BType), depth+1)
		if !ok {
			return
		}
	} else {
		listenerServiceTypes := make([]semtypes.SemType, 0, len(svc.AttachedExprs))
		for _, expr := range svc.AttachedExprs {
			listenerServiceTy, _, ok := listenerType(t, expr, attachPointBound)
			if !ok {
				return
			}
			listenerServiceTypes = append(listenerServiceTypes, listenerServiceTy)
		}

		serviceTy = inferServiceType(listenerServiceTypes)
		if semtypes.IsEmpty(t.typeContext(), serviceTy) {
			t.semanticError("cannot derive a service type satisfying all listeners", svc.AttachedExprsPosition)
			return
		}
	}
	if !semtypes.IsSubtype(t.typeContext(), serviceTy, semtypes.CreateServiceObject(t.typeContext())) {
		t.semanticError("service type must be a subtype of service object {}", svc.GetPosition())
		return
	}
	if !semtypes.IsAtomicObjectType(t.typeContext(), serviceTy) {
		t.semanticError("service type must be atomic", svc.GetPosition())
		return
	}

	svc.AttachPointType = serviceAttachPointType(t, svc)
	if semtypes.IsNever(svc.AttachPointType) {
		return
	}

	od := semtypes.NewObjectDefinition()
	svc.Definition = &od
	objectBodyTy, ok := finishResolveObjectDefinitionType(t, &od, t.symbolName(svc.Symbol()), svc.Fields, svc.Methods,
		svc.ResourceMethods, svc.InitFunction, nil, nil, svc.GetPosition(), depth, svc.IsIsolated(), false, false, true,
		model.SymbolRef{})
	if !ok {
		return
	}
	registerFunctionMethodTable(t, objectBodyTy, model.SymbolRef{}, svc.Methods, svc.InitFunction, svc.GetPosition())
	structuralServiceTy := semtypes.StripObjectDistinctAtoms(serviceTy)
	if !semtypes.IsSubtype(t.typeContext(), objectBodyTy, structuralServiceTy) {
		t.semanticError("service body does not implement the service type", svc.GetPosition())
		return
	}
	objectBodyTy = semtypes.Intersect(objectBodyTy, serviceTy)

	typeData.Type = serviceTy
	setTypeData(t, svc, typeData)
	svc.ObjectBodyType = objectBodyTy
	t.setSymbolType(svc.Symbol(), objectBodyTy)
	if selfRef, ok := svc.Scope().GetSymbol("self"); ok {
		t.setSymbolType(selfRef, objectBodyTy)
	}
	t.ensureNotEmpty(objectBodyTy, func() {
		t.semanticError("service definition is empty", svc.GetPosition())
	})
}

func inferServiceType(listenerServiceTypes []semtypes.SemType) semtypes.SemType {
	serviceTy := listenerServiceTypes[0]
	for _, listenerServiceTy := range listenerServiceTypes[1:] {
		serviceTy = semtypes.Intersect(serviceTy, listenerServiceTy)
	}
	return serviceTy
}

func finishResolveObjectDefinitionType(t typeResolver, od *semtypes.ObjectDefinition, owner string, fields []*ast.BLangVariable,
	methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod, initFn *ast.BLangFunction, inclusions []model.SymbolRef,
	inclusionPositions []diagnostics.Location, pos diagnostics.Location, depth int, isIsolated, isReadonly, isClient, isService bool, distinctSymbol model.SymbolRef,
) (_ semtypes.SemType, ok bool) {
	defer func() {
		if !ok {
			defineNeverObject(t, od)
		}
	}()
	// A member that fails to resolve is left out of the object type; the rest are still resolved.
	for _, field := range fields {
		if fieldTy, _, ok := resolveBType(t, nil, field.TypeNode(), depth+1); ok {
			updateSymbolType(t, field, fieldTy)
		}
	}

	if initFn != nil {
		resolveFunctionSignature(t, nil, initFn, depth+1)
	}

	for _, method := range declaredMethods(methods) {
		resolveFunctionSignature(t, nil, method, depth+1)
	}

	for _, rm := range common.DeclaredNodes(resourceMethods) {
		resolveResourceMethodSignature(t, isClient, isService, rm, depth+1)
	}

	includedMembers, ok := collectObjectIncludedMembers(t, inclusions, inclusionPositions, pos, depth)
	if !ok {
		return semtypes.SemType{}, false
	}

	directMembers, ok := buildObjectDirectMembers(t, owner, fields, methods, initFn, isClient, isService)
	if !ok {
		return semtypes.SemType{}, false
	}

	members, ok := validateOverridesAndMerge(t, directMembers, includedMembers, pos, false)
	if !ok {
		return semtypes.SemType{}, false
	}

	return defineObjectSemType(t, od, isIsolated, isReadonly, isClient, isService, members, distinctSymbol, inclusions), true
}

func collectObjectIncludedMembers(t typeResolver, inclusions []model.SymbolRef, positions []diagnostics.Location, pos diagnostics.Location, depth int) (map[string][]semtypes.Member, bool) {
	includedMembers := make(map[string][]semtypes.Member)
	incMembers, ok := collectIncludedMembers(t, inclusions, positions, pos, depth)
	if !ok {
		return nil, false
	}
	for _, m := range incMembers {
		if m.MemberKind() == model.InclusionMemberKindRestType {
			t.internalError("unexpected rest inclusion", pos)
			return nil, false
		}
		member, ok := inclusionMemberToSemtypeMember(t, m, pos)
		if !ok {
			return nil, false
		}
		includedMembers[member.Name] = append(includedMembers[member.Name], member)
	}
	return includedMembers, true
}

func buildObjectDirectMembers(t typeResolver, owner string, fields []*ast.BLangVariable, methods map[string]*ast.BLangFunction, initFn *ast.BLangFunction, isClient bool, isService bool) ([]directMember, bool) {
	var directMembers []directMember
	for _, field := range typedFields(fields) {
		fieldTy := t.symbolType(field.Symbol())
		vis := memberVisibility(t, field.IsPublic(), privateOwner(field.IsPrivate(), owner))
		directMembers = append(directMembers, directMember{
			name:       field.Name.GetValue(),
			valueTy:    fieldTy,
			kind:       semtypes.MemberKindField,
			visibility: vis,
			immutable:  false,
			pos:        field.GetPosition(),
		})
	}

	if initFn == nil || hasResolvedSignature(t, initFn) {
		initMember, ok := initDirectMember(t, initFn)
		if !ok {
			return nil, false
		}
		directMembers = append(directMembers, initMember)
	}

	for name, method := range resolvedMethods(t, methods) {
		methodTy := methodMemberType(t, method.Symbol())
		vis := memberVisibility(t, method.IsPublic() || method.IsRemote() || method.IsResource(),
			privateOwner(method.IsPrivate(), owner))
		memberKind := semtypes.MemberKindMethod
		if method.IsRemote() {
			if !isClient && !isService {
				t.semanticError("remote methods are only allowed in client or service classes", method.GetPosition())
				return nil, false
			}
			memberKind = semtypes.MemberKindRemoteMethod
		} else if method.IsResource() {
			memberKind = semtypes.MemberKindResourceMethod
		}
		directMembers = append(directMembers, directMember{
			name:       name,
			valueTy:    methodTy,
			kind:       memberKind,
			visibility: vis,
			immutable:  true,
			pos:        method.GetPosition(),
		})
	}
	return directMembers, true
}

// typedFields returns the fields that declare a symbol and whose type resolved.
func typedFields(fields []*ast.BLangVariable) []*ast.BLangVariable {
	return slices.DeleteFunc(slices.Clone(fields), func(field *ast.BLangVariable) bool { return !isTypedVariable(field) })
}

// declaredMethods returns the methods that declare a symbol.
func declaredMethods(methods map[string]*ast.BLangFunction) map[string]*ast.BLangFunction {
	declared := maps.Clone(methods)
	maps.DeleteFunc(declared, func(_ string, method *ast.BLangFunction) bool { return !ast.SymbolIsSet(method) })
	return declared
}

// resolvedMethods returns the methods whose signature resolved.
func resolvedMethods(t typeResolver, methods map[string]*ast.BLangFunction) map[string]*ast.BLangFunction {
	resolved := maps.Clone(methods)
	maps.DeleteFunc(resolved, func(_ string, method *ast.BLangFunction) bool { return !hasResolvedSignature(t, method) })
	return resolved
}

// hasResolvedSignature reports whether fn declares a symbol whose signature resolved. A function whose signature
// does not resolve has type never. A dependently-typed function has no type of its own.
func hasResolvedSignature(t typeResolver, fn *ast.BLangFunction) bool {
	if !ast.SymbolIsSet(fn) {
		return false
	}
	if _, ok := t.getSymbol(fn.Symbol()).(model.DependentlyTypedFunctionSymbol); ok {
		return true
	}
	ty := t.symbolType(fn.Symbol())
	return semtypes.IsZero(ty) || !semtypes.IsNever(ty)
}

// initDirectMember returns the init function member (explicit or implicit).
func initDirectMember(t typeResolver, initFn *ast.BLangFunction) (directMember, bool) {
	if initFn != nil {
		initFnSymbol := t.getSymbol(initFn.Symbol()).(model.FunctionSymbol)
		sig := initFnSymbol.TypedSignature()
		tyCtx := t.typeContext()
		if !semtypes.IsSubtype(tyCtx, sig.ReturnType, semtypes.Union(semtypes.Error, semtypes.Nil)) {
			t.semanticError("invalid return type for init function", initFn.GetPosition())
			return directMember{}, false
		}
		return directMember{
			name:       "init",
			valueTy:    t.symbolType(initFn.Symbol()),
			kind:       semtypes.MemberKindMethod,
			visibility: semtypes.VisibilityPublic,
			immutable:  true,
			pos:        initFn.GetPosition(),
		}, true
	}
	paramListDefn := semtypes.NewListDefinition()
	paramListTy := paramListDefn.Define(t.typeEnv(), nil,
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	functionDefn := semtypes.NewFunctionDefinition()
	initFnType := functionDefn.Define(t.typeEnv(), paramListTy, semtypes.Nil,
		semtypes.FunctionQualifiersFrom(t.typeEnv(), false, false))
	return directMember{
		name:       "init",
		valueTy:    initFnType,
		kind:       semtypes.MemberKindMethod,
		visibility: semtypes.VisibilityPublic,
		immutable:  true,
	}, true
}

// defineObjectSemType finalises the object semtype using the class/service
// qualifiers and resolved members.
func defineObjectSemType(t typeResolver, od *semtypes.ObjectDefinition, isolated bool, readonly bool, isClient bool, isService bool,
	members []semtypes.Member, distinctSymbol model.SymbolRef, inclusions []model.SymbolRef,
) semtypes.SemType {
	networkQual := semtypes.NetworkQualifierNone
	if isClient {
		networkQual = semtypes.NetworkQualifierClient
	} else if isService {
		networkQual = semtypes.NetworkQualifierService
	}
	qualifiers := semtypes.ObjectQualifiersFrom(isolated, readonly, networkQual)
	semType := od.Define(t.typeEnv(), qualifiers, members)
	if distinctSymbol.IsEmpty() {
		return semType
	}
	return appendDistinctObjectAtoms(t, semType, distinctSymbol, inclusions)
}

func resolveLiteral(t typeResolver, n *ast.BLangLiteral, expectedType semtypes.SemType) (semtypes.SemType, bool) {
	var ty semtypes.SemType

	switch n.GetLiteralKind() {
	case ast.LiteralKindInt, ast.LiteralKindByte, ast.LiteralKindFloat, ast.LiteralKindDecimal:
		var ok bool
		ty, ok = resolveNumericLiteralValue(t, n, expectedType)
		if !ok {
			return semtypes.SemType{}, false
		}
	case ast.LiteralKindBoolean:
		value := n.GetValue().(bool)
		ty = semtypes.BooleanConst(value)
	case ast.LiteralKindString:
		value := n.GetValue().(string)
		ty = semtypes.StringConst(value)
	case ast.LiteralKindNil:
		ty = semtypes.Nil
	default:
		t.unimplemented("unsupported literal type", n.GetPosition())
		return semtypes.SemType{}, false
	}

	setNodeType(t, n, ty)
	setLiteralSymbolType(t, n, ty)
	return ty, true
}

func hasFloatTypeSuffix(s string) bool {
	if len(s) == 0 {
		return false
	}
	last := s[len(s)-1]
	return last == 'f' || last == 'F'
}

func determineCandidatesFromLiteral(t typeResolver, n *ast.BLangLiteral) semtypes.SemType {
	switch n.GetLiteralKind() {
	case ast.LiteralKindInt, ast.LiteralKindByte:
		return semtypes.Number
	case ast.LiteralKindFloat:
		if hasFloatTypeSuffix(n.OriginalValue) {
			return semtypes.Float
		}
		if balCommon.HasHexIndicator(n.OriginalValue) {
			return semtypes.Float
		}
		return semtypes.Union(semtypes.Float, semtypes.Decimal)
	case ast.LiteralKindDecimal:
		return semtypes.Decimal
	default:
		t.internalError(fmt.Sprintf("unexpected literal kind %v for numeric literal", n.GetLiteralKind()), n.GetPosition())
		return semtypes.Never
	}
}

func narrowCandidates(candidates, expectedType semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(expectedType) {
		return candidates
	}
	narrowed := semtypes.Intersect(candidates, expectedType)
	if !semtypes.IsNever(narrowed) {
		return narrowed
	}
	return candidates
}

func pickNumericType(t typeResolver, n *ast.BLangLiteral, candidates semtypes.SemType) (semtypes.SemType, bool) {
	switch {
	case semtypes.ContainsBasicType(candidates, semtypes.Int):
		return resolveAsInt(t, n)
	case semtypes.ContainsBasicType(candidates, semtypes.Float):
		return resolveAsFloat(t, n)
	case semtypes.ContainsBasicType(candidates, semtypes.Decimal):
		return resolveAsDecimal(t, n)
	default:
		t.semanticError("no valid candidate to resolve numeric literal", n.GetPosition())
		return semtypes.SemType{}, false
	}
}

func resolveAsInt(t typeResolver, n *ast.BLangLiteral) (semtypes.SemType, bool) {
	text := n.OriginalValue
	var intVal int64
	value, _ := balCommon.ParseIntLiteral(text)
	switch v := value.(type) {
	case int64:
		intVal = v
	case float64:
		intVal = int64(v)
	default:
		t.internalError(fmt.Sprintf("invalid int literal: %s", text), n.GetPosition())
		return semtypes.SemType{}, false
	}
	setLiteralValue(t, n, intVal)
	return semtypes.IntConst(intVal), true
}

func resolveAsFloat(t typeResolver, n *ast.BLangLiteral) (semtypes.SemType, bool) {
	text := n.OriginalValue
	var floatVal float64
	value, _ := balCommon.ParseIntLiteral(text)
	switch v := value.(type) {
	case int64:
		floatVal = float64(v)
	case float64:
		floatVal = v
	default:
		if balCommon.HasHexIndicator(text) {
			text = balCommon.NormalizeHexFloatLiteral(text)
		}
		parsed, ok := parseFloatValue(t, text, n.GetPosition())
		if !ok {
			return semtypes.SemType{}, false
		}
		floatVal = parsed
	}
	setLiteralValue(t, n, floatVal)
	return semtypes.FloatConst(floatVal), true
}

func resolveAsDecimal(t typeResolver, n *ast.BLangLiteral) (semtypes.SemType, bool) {
	text := n.OriginalValue
	var decVal *decimal.Decimal
	value, _ := balCommon.ParseIntLiteral(text)
	switch v := value.(type) {
	case int64:
		decVal = decimal.FromInt64(v)
	case float64:
		d, err := decimal.FromString(strconv.FormatFloat(v, 'g', -1, 64))
		if err != nil {
			t.internalError(fmt.Sprintf("failed to convert float %v to decimal: %v", v, err), n.GetPosition())
			return semtypes.SemType{}, false
		}
		decVal = d
	default:
		parsed, ok := parseDecimalValue(t, stripFloatingPointTypeSuffix(text), n.GetPosition())
		if !ok {
			return semtypes.SemType{}, false
		}
		decVal = parsed
	}
	setLiteralValue(t, n, decVal)
	return semtypes.DecimalConst(*decVal), true
}

func resolveNumericLiteralValue(t typeResolver, n *ast.BLangLiteral, expectedType semtypes.SemType) (semtypes.SemType, bool) {
	candidates := determineCandidatesFromLiteral(t, n)
	candidates = narrowCandidates(candidates, expectedType)
	return pickNumericType(t, n, candidates)
}

// stripFloatingPointTypeSuffix removes the f/F/d/D type suffix from a floating point literal string
func stripFloatingPointTypeSuffix(s string) string {
	last := s[len(s)-1]
	if last == 'f' || last == 'F' || last == 'd' || last == 'D' {
		return s[:len(s)-1]
	}
	return s
}

func parseFloatValue(t typeResolver, strValue string, pos diagnostics.Location) (float64, bool) {
	strValue = strings.TrimRight(strValue, "fF")
	f, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		t.semanticError(fmt.Sprintf("invalid float literal: %s", strValue), pos)
		return 0, false
	}
	return f, true
}

func parseDecimalValue(t typeResolver, strValue string, pos diagnostics.Location) (*decimal.Decimal, bool) {
	d, err := decimal.FromLiteral(strValue)
	if err != nil {
		t.semanticError(fmt.Sprintf("invalid decimal literal: %s", strValue), pos)
		return decimal.FromInt64(0), false
	}
	return d, true
}

func resolveNumericLiteral(t typeResolver, n *ast.BLangNumericLiteral, expectedType semtypes.SemType) (semtypes.SemType, bool) {
	candidates := determineCandidatesFromLiteral(t, &n.BLangLiteral)
	candidates = narrowCandidates(candidates, expectedType)

	ty, ok := pickNumericType(t, &n.BLangLiteral, candidates)
	if !ok {
		return semtypes.SemType{}, false
	}

	setNodeType(t, n, ty)
	setLiteralSymbolType(t, &n.BLangLiteral, ty)
	return ty, true
}

// updateSymbolType updates the symbol's type if the node has an associated symbol.
func updateSymbolType(t typeResolver, node ast.BLangNode, ty semtypes.SemType) {
	if nodeWithSymbol, ok := node.(ast.BNodeWithSymbol); ok && ast.SymbolIsSet(nodeWithSymbol) {
		t.setSymbolType(nodeWithSymbol.Symbol(), ty)
	}
}

func lookupSymbol(t typeResolver, chain *binding, ref model.SymbolRef) model.SymbolRef {
	if chain == nil {
		return ref
	}
	narrowedRef, isNarrowed := lookupBinding(t, chain, ref)
	if isNarrowed {
		return narrowedRef
	}
	return ref
}

func resolveVariableDefStmt(t typeResolver, chain *binding, s *ast.BLangVariableDef) (statementEffect, bool) {
	variable := s.GetVariable()
	typeNode := variable.TypeNode()
	if typeNode != nil {
		semType, typeChain, ok := resolveBType(t, chain, typeNode, 0)
		if !ok {
			updateSymbolType(t, variable, semtypes.Never)
			return defaultStmtEffect(chain), false
		}
		chain = typeChain
		updateSymbolType(t, variable, semType)
		if fnType, ok := typeNode.(*ast.BLangFunctionType); ok {
			if !finalizeResolvedFunctionSignature(t, fnType) {
				return defaultStmtEffect(chain), false
			}
		}
	}

	effectChain := chain
	if variable.Expr != nil {
		var expectedType semtypes.SemType
		if ast.SymbolIsSet(variable) {
			expectedType = t.symbolType(variable.Symbol())
		} else if typeNode != nil {
			expectedType = typeNode.Type()
		}
		result, ok := resolveActionOrExpression(t, chain, variable.Expr, expectedType)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		exprTy := result.ty
		effectChain = mergeChains(t, result.effect.ifTrue, result.effect.ifFalse, semtypes.Union)
		if typeNode == nil {
			updateSymbolType(t, variable, exprTy)
			if !associateInferredFunctionSignature(t, variable, result.functionSignature) {
				return defaultStmtEffect(chain), false
			}
		}
	}

	return defaultStmtEffect(effectChain), true
}

// detectGlobalVarInitCycles flags cycles in the dependency graph induced by
// module-level variable initializer expressions. Constants get cycle detection
// for free via packageTypeResolver.ensureResolved while types are being
// resolved; module vars don't go through that path, so we do a dedicated pass here.
// Cross-module references are leaves — imported modules' inits are guaranteed
// to have run already by the time this module's init runs.
func detectGlobalVarInitCycles(t typeResolver, globalVars []*ast.BLangVariable) {
	if len(globalVars) == 0 {
		return
	}
	// Inferred-type globals are cycle-checked via ensureResolved during the
	// init-expression pass (the in-progress nil-marker pattern that constants
	// also use). Skip them here to avoid duplicate diagnostics.
	nodeSet := make(map[model.SymbolRef]int, len(globalVars))
	for i := range globalVars {
		if globalVars[i].TypeNode() == nil {
			continue
		}
		nodeSet[globalVars[i].Symbol()] = i
	}

	deps := make([][]int, len(globalVars))
	for i := range globalVars {
		gv := globalVars[i]
		if gv.Expr == nil || gv.TypeNode() == nil {
			continue
		}
		deps[i] = symbolDeps(t, nodeSet, gv.Expr)
	}

	// https://en.wikipedia.org/wiki/Topological_sorting#Depth-first_search
	const (
		unvisited = 0
		inStack   = 1
		done      = 2
	)
	state := make([]int, len(globalVars))

	var visit func(i int) bool
	visit = func(i int) bool {
		switch state[i] {
		case inStack:
			t.semanticError(
				fmt.Sprintf("invalid cycle detected for %s", globalVars[i].Name.GetValue()),
				globalVars[i].Name.GetPosition(),
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
			return true
		}
	}

	for i := range globalVars {
		if globalVars[i].TypeNode() == nil {
			continue
		}
		if !visit(i) {
			return
		}
	}
}

func resolveGlobalVarType(t typeResolver, node *ast.BLangVariable) bool {
	typeNode := node.TypeNode()
	if typeNode == nil {
		return true
	}
	semType, _, ok := resolveBType(t, nil, typeNode, 0)
	if !ok {
		updateSymbolType(t, node, semtypes.Never)
		return false
	}
	updateSymbolType(t, node, semType)
	if fnType, ok := typeNode.(*ast.BLangFunctionType); ok {
		return finalizeResolvedFunctionSignature(t, fnType)
	}
	return true
}

func resolveGlobalVarInit(t typeResolver, node *ast.BLangVariable) bool {
	if node.Expr == nil {
		return true
	}
	if node.TypeNode() == nil {
		if pt, ok := t.(*packageTypeResolver); ok {
			return pt.ensureResolved(node.Symbol(), 0)
		}
		_, ok := resolveSimpleVariable(t, nil, node)
		return ok
	}
	semType := t.symbolType(node.Symbol())
	if semtypes.IsZero(semType) {
		return false
	}
	expectedType := semType
	if node.IsListener() {
		expectedType = common.ListenerInitExpectedType(semType)
	}
	_, ok := resolveActionOrExpression(t, nil, node.Expr, expectedType)
	return ok
}

// resolveServiceAttachedExpressions type-checks the listener expressions in
// a service's `on` clause so service type resolution can inspect them.
func resolveServiceAttachedExpressions(t typeResolver, svc *ast.BLangService) bool {
	for _, expr := range svc.AttachedExprs {
		if _, ok := resolveActionOrExpression(t, nil, expr, semtypes.SemType{}); !ok {
			return false
		}
	}
	return true
}

// validateListenerVars verifies each module-level listener variable's
// resolved type is a subtype of the global LISTENER top type. Reports a
// semantic error otherwise.
func validateListenerVars(t typeResolver, pkg *ast.BLangPackage, attachPointBound semtypes.SemType) {
	tyCtx := t.typeContext()
	for _, gv := range common.DeclaredNodes(pkg.GlobalVars) {
		if !gv.IsListener() {
			continue
		}
		ty := t.symbolType(gv.Symbol())
		if semtypes.IsZero(ty) {
			continue
		}
		if _, _, ok := common.ListenerTypes(tyCtx, ty, attachPointBound); !ok {
			t.semanticError("listener initializer is not a listener", gv.GetPosition())
		}
	}
}

func listenerType(t typeResolver, expr ast.BLangExpression, attachPointBound semtypes.SemType) (semtypes.SemType, semtypes.SemType, bool) {
	exprTy := expr.Type()
	if semtypes.IsZero(exprTy) {
		t.internalError("listener expression has no type", expr.GetPosition())
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	checkedTy := semtypes.Diff(exprTy, semtypes.Error)
	targetTy, attachTy, ok := common.ListenerTypes(t.typeContext(), checkedTy, attachPointBound)
	if !ok {
		t.semanticError("expression in 'on' clause is not a listener", expr.GetPosition())
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	return targetTy, attachTy, true
}

func serviceAttachPointType(t typeResolver, svc *ast.BLangService) semtypes.SemType {
	if svc.AttachPointLiteral != nil {
		ty, ok := resolveLiteral(t, svc.AttachPointLiteral, semtypes.String)
		if !ok {
			return semtypes.Never
		}
		return ty
	}
	if svc.AbsoluteResourcePath == nil {
		return semtypes.Nil
	}
	segmentTypes := make([]semtypes.SemType, len(svc.AbsoluteResourcePath))
	for i := range svc.AbsoluteResourcePath {
		segmentTypes[i] = semtypes.StringConst(svc.AbsoluteResourcePath[i].Value)
	}
	listDefn := semtypes.NewListDefinition()
	return listDefn.Define(t.typeEnv(), segmentTypes,
		semtypes.ListMutability(semtypes.CellMutabilityNone))
}

func associateInferredFunctionSignature(t typeResolver, variable *ast.BLangVariable, ref model.FunctionSignatureRef) bool {
	if ref == 0 {
		return true
	}
	if !t.associateFunctionSignature(variable.Symbol(), ref) {
		t.internalError("function signature already set", variable.GetPosition())
		return false
	}
	return true
}

func resolveSimpleVariable(t typeResolver, chain *binding, node *ast.BLangVariable) (*binding, bool) {
	return resolveSimpleVariableInner(t, chain, node, 0)
}

func resolveSimpleVariableInner(t typeResolver, chain *binding, node *ast.BLangVariable, depth int) (*binding, bool) {
	typeNode := node.TypeNode()
	if typeNode == nil {
		if node.Expr != nil {
			result, ok := resolveActionOrExpression(t, chain, node.Expr, semtypes.SemType{})
			if !ok {
				return chain, false
			}
			chain = sequentialChain(t, result.effect)
			exprTy := result.ty
			updateSymbolType(t, node, exprTy)
			if !associateInferredFunctionSignature(t, node, result.functionSignature) {
				return chain, false
			}
		}
		return chain, true
	}

	semType, chain, ok := resolveBType(t, chain, typeNode, depth)
	if !ok {
		updateSymbolType(t, node, semtypes.Never)
		return chain, false
	}

	updateSymbolType(t, node, semType)
	if node.Expr != nil {
		if isParameterDeclaration(t, node) {
			// A parameter's initializer is a deferred default provider, not part of
			// the enclosing function's flow.
			defaultResult, ok := resolveDefaultExpression(t, chain, node, node.Expr, semType)
			if !ok {
				return chain, false
			}
			chain = sequentialChain(t, defaultResult.effect)
		} else {
			initResult, ok := resolveActionOrExpression(t, chain, node.Expr, semType)
			if !ok {
				return chain, false
			}
			chain = sequentialChain(t, initResult.effect)
		}
	}

	return chain, true
}

// isParameterDeclaration reports whether node declares a function parameter.
func isParameterDeclaration(t typeResolver, node *ast.BLangVariable) bool {
	metadata, ok := t.compilerContext().ValueSymbolMetadata(node.Symbol())
	return ok && metadata.Parameter
}

// resolveDefaultExpression checks a deferred default provider's body behind a
// function boundary and returns only the effects of constructing that provider.
// Checking the body is not executing it, so the body's own effects do not reach
// the enclosing chain.
func resolveDefaultExpression(
	t typeResolver,
	chain *binding,
	owner ast.CaptureGroupOwner,
	expr ast.BLangActionOrExpression,
	expectedType semtypes.SemType,
) (expressionResult, bool) {
	constructionChain := addCaptureGroup(chain, owner.CaptureGroup())
	bodyChain := &binding{flags: bindingFlagFunctionBoundary, prev: constructionChain}
	result, ok := resolveActionOrExpression(t, bodyChain, expr, expectedType)
	if !ok {
		return expressionResult{}, false
	}
	result.effect = defaultExpressionEffect(constructionChain)
	return result, true
}

// resolveActionOrExpression resolves expr against expectedType. An expression that fails to resolve has type never.
func resolveActionOrExpression(t typeResolver, chain *binding, expr ast.BLangActionOrExpression, expectedType semtypes.SemType) (expressionResult, bool) {
	result, ok := resolveExpressionInner(t, chain, expr, expectedType)
	if !ok {
		setNodeType(t, expr, semtypes.Never)
		return expressionResult{}, false
	}
	// A constant-valued condition marks its impossible outcome here. The possible
	// outcome keeps the effects the expression produced.
	return withSingletonEffect(chain, result), true
}

func resolveExpressionInner(t typeResolver, chain *binding, expr ast.BLangActionOrExpression, expectedType semtypes.SemType) (expressionResult, bool) {
	resolved := func(ty semtypes.SemType, effect expressionEffect, ok bool) (expressionResult, bool) {
		return expressionResult{ty: ty, effect: effect}, ok
	}
	withFunctionSignature := func(result expressionResult, ok bool, ref model.FunctionSignatureRef) (expressionResult, bool) {
		if ok {
			result.functionSignature = ref
		}
		return result, ok
	}
	returnedFunction := func(result expressionResult, ok bool) (expressionResult, bool) {
		if !ok {
			return result, false
		}
		if result.symbol.IsEmpty() {
			return result, true
		}
		calleeRef, ok := t.functionSignatureRef(result.symbol)
		if ok {
			result.functionSignature, _ = t.returnFunctionSignatureRef(calleeRef)
		}
		return result, true
	}
	convertedFunction := func(result expressionResult, ok bool, expr *ast.BLangTypeConversionExpr) (expressionResult, bool) {
		var ref model.FunctionSignatureRef
		switch ty := expr.TypeDescriptor.(type) {
		case *ast.BLangFunctionType:
			if !ty.IsAnyFunction() {
				ref = ty.SignatureRef()
			}
		case *ast.BLangUserDefinedType:
			ref, _ = t.functionSignatureRef(ty.Symbol())
		}
		return withFunctionSignature(result, ok, ref)
	}

	if ty := expr.Type(); !semtypes.IsZero(ty) {
		result := expressionResult{ty: ty, effect: defaultExpressionEffect(chain)}
		switch e := expr.(type) {
		case *ast.BLangVarRef:
			ref, _ := t.functionSignatureRef(e.Symbol())
			return withFunctionSignature(result, true, ref)
		case *ast.BLangInvocation:
			return returnedFunction(withRawSymbol(result, e.RawSymbol), true)
		case *ast.BLangGroupExpr:
			return resolveActionOrExpression(t, chain, e.Expression, expectedType)
		case *ast.BLangTypeConversionExpr:
			return convertedFunction(result, true, e)
		case *ast.BLangNamedArgsExpression:
			return resolveActionOrExpression(t, chain, e.Expr, expectedType)
		case *ast.BLangLambdaFunction:
			ref, _ := t.functionSignatureRef(e.Function.Symbol())
			return withFunctionSignature(result, true, ref)
		case *ast.BLangRemoteMethodCallAction:
			return returnedFunction(withRawSymbol(result, e.RawSymbol), true)
		case *ast.BLangFieldBaseAccess:
			result.lax = e.IsLax()
			return result, true
		default:
			return result, true
		}
	}

	switch e := expr.(type) {
	case *ast.BLangLiteral:
		ty, ok := resolveLiteral(t, e, expectedType)
		if !ok {
			return expressionResult{}, false
		}
		return expressionResult{ty: ty, effect: defaultExpressionEffect(chain)}, true
	case *ast.BLangNumericLiteral:
		ty, ok := resolveNumericLiteral(t, e, expectedType)
		if !ok {
			return expressionResult{}, false
		}
		return expressionResult{ty: ty, effect: defaultExpressionEffect(chain)}, true
	case *ast.BLangVarRef:
		result, ok := resolveSimpleVarRef(t, chain, e)
		if !ok {
			return expressionResult{}, false
		}
		ref, _ := t.functionSignatureRef(result.symbol)
		return withFunctionSignature(result, true, ref)
	case *ast.BLangConstRef:
		return resolveConstRef(t, chain, e)
	case *ast.BLangBinaryExpr:
		return resolved(resolveBinaryExpr(t, chain, e, expectedType))
	case *ast.BLangTernaryExpr:
		return resolved(resolveTernaryExpr(t, chain, e, expectedType))
	case *ast.BLangNilConditionalExpr:
		return resolved(resolveNilConditionalExpr(t, chain, e, expectedType))
	case *ast.BLangUnaryExpr:
		return resolved(resolveUnaryExpr(t, chain, e, expectedType))
	case *ast.BLangInvocation:
		result, ok := resolveInvocation(t, chain, e, expectedType)
		return returnedFunction(result, ok)
	case *ast.BLangIndexBasedAccess:
		return resolved(resolveIndexBasedAccess(t, chain, e))
	case *ast.BLangFieldBaseAccess:
		return resolveFieldBaseAccess(t, chain, e)
	case *ast.BLangListConstructorExpr:
		return resolveListConstructorExpr(t, chain, e, expectedType)
	case *ast.BLangMappingConstructorExpr:
		return resolveMappingConstructorExpr(t, chain, e, expectedType)
	case *ast.BLangErrorConstructorExpr:
		return resolved(resolveErrorConstructorExpr(t, chain, e, expectedType))
	case *ast.BLangGroupExpr:
		result, ok := resolveActionOrExpression(t, chain, e.Expression, expectedType)
		if ok {
			setNodeType(t, e, result.ty)
		}
		return result, ok
	case *ast.BLangQueryExpr:
		return resolved(resolveQueryExpr(t, chain, e, expectedType))
	case *ast.BLangWildCardBindingPattern:
		ty := semtypes.Any
		setNodeType(t, e, ty)
		return expressionResult{ty: ty, effect: defaultExpressionEffect(chain)}, true
	case *ast.BLangTypeConversionExpr:
		result, ok := resolved(resolveTypeConversionExpr(t, chain, e))
		return convertedFunction(result, ok, e)
	case *ast.BLangTypeTestExpr:
		return resolved(resolveTypeTestExpr(t, chain, e))
	case *ast.BLangTypedescExpr:
		return resolved(resolveTypedescExpr(t, chain, e))
	case *ast.BLangAnnotAccessExpr:
		return resolved(resolveAnnotAccessExpr(t, chain, e))
	case *ast.BLangCheckedExpr:
		return resolved(resolveCheckedExpr(t, chain, e, expectedType))
	case *ast.BLangCheckPanickedExpr:
		return resolved(resolveCheckedExpr(t, chain, &e.BLangCheckedExpr, expectedType))
	case *ast.BLangTrapExpr:
		return resolved(resolveTrapExpr(t, chain, e))
	case *ast.BLangNamedArgsExpression:
		result, ok := resolveActionOrExpression(t, chain, e.Expr, expectedType)
		if !ok {
			return expressionResult{}, false
		}
		setNodeType(t, e, result.ty)
		return result, true
	case *ast.BLangNewExpression:
		return resolved(resolveNewExpr(t, chain, e, expectedType))
	case *ast.BLangLambdaFunction:
		result, ok := resolved(resolveLambdaFunctionExpr(t, chain, e, expectedType))
		ref, _ := t.functionSignatureRef(e.Function.Symbol())
		return withFunctionSignature(result, ok, ref)
	case *ast.BLangRemoteMethodCallAction:
		result, ok := resolveRemoteMethodCallAction(t, chain, e, expectedType)
		return returnedFunction(result, ok)
	case *ast.BLangClientResourceAccessAction:
		return resolveClientResourceAccessAction(t, chain, e, expectedType)
	case *ast.BLangStartAction:
		return resolved(resolveStartAction(t, chain, e, expectedType))
	case *ast.BLangSingleWaitAction:
		return resolved(resolveSingleWaitAction(t, chain, e))
	case *ast.BLangAlternateWaitAction:
		return resolved(resolveAlternateWaitAction(t, chain, e))
	case *ast.BLangMultipleWaitAction:
		return resolved(resolveMultipleWaitAction(t, chain, e, expectedType))
	case *ast.BLangInferredTypedescDefault:
		return resolved(resolveInferredTypedescDefault(t, chain, e, expectedType))
	case *ast.BLangDefaultArg:
		defaultFn := t.getSymbol(e.DefaultClosure).(model.FunctionSymbol)
		returnType := defaultFn.TypedSignature().ReturnType
		setNodeType(t, e, returnType)
		return expressionResult{ty: returnType, effect: defaultExpressionEffect(chain)}, true
	case *ast.BLangXMLSequenceLiteral:
		return resolved(resolveXMLSequenceLiteral(t, chain, e, expectedType))
	case *ast.BLangTemplateExpr:
		return resolved(resolveTemplateExpr(t, chain, e))
	case *ast.BLangXMLTemplateExpr:
		return resolved(resolveXMLTemplateExpr(t, chain, e, expectedType))
	case *ast.BLangXMLElementLiteral:
		return resolved(resolveXMLElementLiteral(t, chain, e, expectedType))
	case *ast.BLangXMLPILiteral:
		return resolved(resolveXMLPILiteral(t, chain, e, expectedType))
	case *ast.BLangXMLCommentLiteral:
		return resolved(resolveXMLCommentLiteral(t, chain, e, expectedType))
	case *ast.BLangXMLTextLiteral:
		return resolved(resolveXMLTextLiteral(t, chain, e))
	case *ast.BLangXMLFilterExpression:
		return resolved(resolveXMLFilterExpr(t, chain, e, expectedType))
	case *ast.BLangXMLStepExpression:
		return resolved(resolveXMLStepExpr(t, chain, e, expectedType))
	default:
		t.internalError(fmt.Sprintf("unsupported expression type: %T", expr), expr.GetPosition())
		return expressionResult{}, false
	}
}

// resolveInferredTypedescDefault handles the "<>" default value that appears as
// the initializer of a dependently-typed function's typedesc parameter.
//
// When encountered as the parameter's own default initializer (expectedType is
// the parameter's declared typedesc type) it just adopts that type. When it is
// synthesized into a call-site argument list, expectedType is the inferred
// typedesc<T>. In either case the type becomes expectedType.
func resolveInferredTypedescDefault(t typeResolver, chain *binding, e *ast.BLangInferredTypedescDefault, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if semtypes.IsZero(expectedType) || !semtypes.IsSubtype(t.typeContext(), expectedType, semtypes.Typedesc) {
		t.semanticError("inferred typedesc default '<>' is only allowed as the default for a typedesc parameter", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, e, expectedType)
	return expectedType, defaultExpressionEffect(chain), true
}

func resolveTypedescExpr(t typeResolver, chain *binding, e *ast.BLangTypedescExpr) (semtypes.SemType, expressionEffect, bool) {
	constraint := e.Constraint
	if typeDesc := e.GetTypeDescriptor(); typeDesc != nil {
		var ok bool
		if constraint, chain, ok = resolveBType(t, chain, typeDesc.(ast.BType), 0); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		setTypedescConstraint(t, e, constraint)
	} else if semtypes.IsZero(constraint) {
		t.internalError("typedesc expression has no type descriptor", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	ty := semtypes.TypedescContaining(t.typeEnv(), constraint)
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveAnnotAccessExpr(t typeResolver, chain *binding, e *ast.BLangAnnotAccessExpr) (semtypes.SemType, expressionEffect, bool) {
	receiverResult, ok := resolveActionOrExpression(t, chain, e.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	receiverTy := receiverResult.ty
	effect := defaultExpressionEffect(sequentialChain(t, receiverResult.effect))
	if !semtypes.IsSubtype(t.typeContext(), receiverTy, semtypes.Typedesc) {
		t.semanticError("annotation access is only allowed on typedesc values", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if !ast.SymbolIsSet(e) {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	sym, ok := t.getSymbol(e.Symbol()).(*model.AnnotationSymbol)
	if !ok {
		t.internalError("annotation access does not resolve to an annotation symbol", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	annTy := sym.Type()
	if semtypes.IsZero(annTy) {
		// An annotation whose declaration failed to resolve has no type.
		return semtypes.SemType{}, expressionEffect{}, false
	}
	ty := semtypes.Union(annTy, semtypes.Nil)
	setNodeType(t, e, ty)
	return ty, effect, true
}

func resolveXMLTextLiteral(t typeResolver, chain *binding, e *ast.BLangXMLTextLiteral) (semtypes.SemType, expressionEffect, bool) {
	ty := semtypes.XMLText
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

// xmlLiteralType narrows shape to expectedType, so a readonly expectation makes the literal
// immutable. A disjoint expectation is left for the caller's assignability check.
func xmlLiteralType(t typeResolver, shape semtypes.SemType, expectedType semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(expectedType) {
		return shape
	}
	narrowed := semtypes.Intersect(shape, expectedType)
	if semtypes.IsEmpty(t.typeContext(), narrowed) {
		return shape
	}
	return narrowed
}

// xmlMutabilityExpectation reduces the applicable contextual expectation, the expectation narrowed
// to xml, to what it says about mutability alone. A template or sequence literal builds a shape
// that is not statically known, so narrowing it against the whole expectation would accept
// `xml<xml:Element> x = xml `text`;`; narrowing against this can only ever take mutability away.
func xmlMutabilityExpectation(t typeResolver, expectedType semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(expectedType) {
		return semtypes.SemType{}
	}
	applicableType := semtypes.Intersect(expectedType, semtypes.XML)
	tc := t.typeContext()
	if !semtypes.IsEmpty(tc, applicableType) && semtypes.IsSubtype(tc, applicableType, semtypes.ValReadonly) {
		return semtypes.ValReadonly
	}
	return semtypes.SemType{}
}

// xmlChildExpectedType is the expectation to pass to the children of an xml literal of type ty. A
// readonly xml value is deeply immutable, so its children must be constructed readonly as well.
func xmlChildExpectedType(t typeResolver, ty semtypes.SemType) semtypes.SemType {
	if semtypes.IsSubtype(t.typeContext(), ty, semtypes.ValReadonly) {
		return semtypes.Intersect(semtypes.XML, semtypes.ValReadonly)
	}
	return semtypes.XML
}

func resolveXMLCommentLiteral(t typeResolver, chain *binding, e *ast.BLangXMLCommentLiteral, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	ty := xmlLiteralType(t, semtypes.XMLComment, expectedType)
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveXMLPILiteral(t typeResolver, chain *binding, e *ast.BLangXMLPILiteral, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	ty := xmlLiteralType(t, semtypes.XMLProcessingInstruction, expectedType)
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveXMLElementLiteral(t typeResolver, chain *binding, e *ast.BLangXMLElementLiteral, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	ty := xmlLiteralType(t, semtypes.XMLElement, expectedType)
	for i := range e.Attrs {
		attr := &e.Attrs[i]
		if attr.Value != nil {
			attrResult, ok := resolveActionOrExpression(t, chain, attr.Value, semtypes.String)
			if !ok {
				return semtypes.SemType{}, expressionEffect{}, false
			}
			chain = sequentialChain(t, attrResult.effect)
		}
	}
	if e.Content != nil {
		contentResult, ok := resolveActionOrExpression(t, chain, e.Content, xmlChildExpectedType(t, ty))
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = sequentialChain(t, contentResult.effect)
	}
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveTemplateExpr(t typeResolver, chain *binding, e *ast.BLangTemplateExpr) (semtypes.SemType, expressionEffect, bool) {
	var ty semtypes.SemType
	if len(e.Insertions) == 0 {
		ty = semtypes.StringConst(e.Strings[0])
	} else {
		var ok bool
		ty, chain, ok = resolveStringTemplateType(t, chain, e)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveStringTemplateType(t typeResolver, chain *binding, e *ast.BLangTemplateExpr) (semtypes.SemType, *binding, bool) {
	allSingleton := true
	var sb strings.Builder
	sb.WriteString(e.Strings[0])
	for i, ins := range e.Insertions {
		insResult, ok := resolveActionOrExpression(t, chain, ins, common.TemplateInsertionAllowedTypes)
		if !ok {
			return semtypes.SemType{}, nil, false
		}
		chain = sequentialChain(t, insResult.effect)
		insTy := insResult.ty
		if allSingleton && semtypes.IsSubtypeSimple(insTy, semtypes.String) {
			if shape := semtypes.SingleShape(insTy); !shape.IsEmpty() {
				sb.WriteString(shape.Get().Value.(string))
				sb.WriteString(e.Strings[i+1])
				continue
			}
		}
		allSingleton = false
	}
	if allSingleton {
		return semtypes.StringConst(sb.String()), chain, true
	}
	return semtypes.String, chain, true
}

func resolveXMLTemplateExpr(t typeResolver, chain *binding, e *ast.BLangXMLTemplateExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if len(e.InsertionKinds) != len(e.Insertions) {
		t.internalError(fmt.Sprintf("xml template insertion kind count mismatch: got %d kinds for %d insertions", len(e.InsertionKinds), len(e.Insertions)), e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	for i, ins := range e.Insertions {
		allowed := common.XMLTemplateInsertionAllowedTypes(e.InsertionKinds[i])
		insResult, ok := resolveActionOrExpression(t, chain, ins, allowed)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = sequentialChain(t, insResult.effect)
	}
	ty := xmlLiteralType(t, semtypes.XML, xmlMutabilityExpectation(t, expectedType))
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveXMLSequenceLiteral(t typeResolver, chain *binding, e *ast.BLangXMLSequenceLiteral, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	ty := xmlLiteralType(t, semtypes.XML, xmlMutabilityExpectation(t, expectedType))
	childExpectedType := xmlChildExpectedType(t, ty)
	childUnion := semtypes.Never
	for _, child := range e.Children {
		childResult, ok := resolveActionOrExpression(t, chain, child, childExpectedType)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = sequentialChain(t, childResult.effect)
		childTy := childResult.ty
		if !semtypes.IsSubtype(t.typeContext(), childTy, semtypes.XML) {
			t.semanticError(fmt.Sprintf("expected xml value, got %s", semtypes.ToString(t.typeContext(), childTy)), child.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		childUnion = semtypes.Union(childUnion, childTy)
	}
	ty = semtypes.XMLSequence(childUnion)
	setNodeType(t, e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveNewExpr(t typeResolver, chain *binding, e *ast.BLangNewExpression, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	cx := t.typeContext()
	var determinedTy semtypes.SemType
	if e.TypeDescriptor != nil {
		resolvedTy, descChain, ok := resolveBType(t, chain, e.TypeDescriptor, 0)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = descChain
		determinedTy = resolvedTy
	} else {
		if semtypes.IsZero(expectedType) {
			t.semanticError("cannot infer type for implicit new expression", e.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		intersection := semtypes.Intersect(expectedType, semtypes.Union(semtypes.Object, semtypes.Stream))
		if semtypes.IsEmpty(cx, intersection) {
			t.semanticError("expected type is not an object or stream type", e.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		determinedTy = intersection
	}

	switch {
	case semtypes.IsSubtypeSimple(determinedTy, semtypes.Object):
		return resolveObjectNewExpr(t, chain, e, determinedTy)
	case semtypes.IsSubtypeSimple(determinedTy, semtypes.Stream):
		return resolveStreamNewExpr(t, chain, e, determinedTy)
	default:
		t.semanticError("new expression target must be either an object or stream type", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
}

func resolveObjectNewExpr(t typeResolver, chain *binding, e *ast.BLangNewExpression, determinedTy semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	// An expression inside a trial is part of the root of an outer trial, whose dependencies were resolved before that
	// trial started.
	if !t.isEphemeral() && !resolveTrialDependencies(t, e.ArgsExprs) {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	candidates, ok := constructibleAlternatives(t, e, determinedTy)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if len(candidates) == 1 {
		return finishObjectNew(t, chain, e, determinedTy, candidates[0])
	}
	selected, outcome := selectCandidate(t, candidates, func(candidate constructibleAlternative) bool {
		_, _, _, ok := resolveObjectInitArgs(t, chain, e, candidate.alternative, candidate.table)
		return ok
	})
	switch outcome {
	case candidateOne:
		return finishObjectNew(t, chain, e, determinedTy, selected)
	case candidateAmbiguous:
		t.semanticError("ambiguous object type", e.GetPosition())
	case candidateUnsupported:
	default:
		t.semanticError("failed to find a suitable object type", e.GetPosition())
	}
	return semtypes.SemType{}, expressionEffect{}, false
}

type constructibleAlternative struct {
	alternative semtypes.ObjectAlternative
	table       model.MethodTable
}

func constructibleAlternatives(t typeResolver, e *ast.BLangNewExpression, determinedTy semtypes.SemType) ([]constructibleAlternative, bool) {
	alternatives := semtypes.ObjectAlternatives(t.typeContext(), determinedTy)
	if len(alternatives) == 0 {
		t.semanticError("failed to find a suitable object type", e.GetPosition())
		return nil, false
	}
	var candidates []constructibleAlternative
	for _, alternative := range alternatives {
		table, ok := methodTableForAlternative(t, alternative)
		if ok && constructibleMethodTable(t, table) {
			candidates = append(candidates, constructibleAlternative{alternative: alternative, table: table})
		}
	}
	if len(candidates) == 0 {
		t.semanticError("object type cannot be instantiated without a class", e.GetPosition())
		return nil, false
	}
	return candidates, true
}

func finishObjectNew(t typeResolver, chain *binding, e *ast.BLangNewExpression, determinedTy semtypes.SemType, candidate constructibleAlternative) (semtypes.SemType, expressionEffect, bool) {
	args, returnType, argsChain, ok := resolveObjectInitArgs(t, chain, e, candidate.alternative, candidate.table)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultObjType := candidate.alternative.Type()
	if semtypes.IsSubtype(t.typeContext(), determinedTy, resultObjType) {
		resultObjType = determinedTy
	}
	resultType := semtypes.Union(resultObjType, semtypes.Diff(returnType, semtypes.Nil))
	setNewExpressionResult(t, e, args, candidate.table.Owner)
	setNodeType(t, e, resultType)
	return resultType, defaultExpressionEffect(argsChain), true
}

// constructibleMethodTable reports whether table's owner is a class, which is what
// `new` needs. A named object type definition also registers a method table, with
// its type symbol as the owner, so that method signatures resolve; such an owner is
// not instantiable. Anonymous inline object types leave the owner empty.
func constructibleMethodTable(t typeResolver, table model.MethodTable) bool {
	if table.Owner.IsEmpty() {
		return false
	}
	_, ok := t.getSymbol(table.Owner).(model.ClassSymbol)
	return ok
}

func methodTableForAlternative(t typeResolver, alternative semtypes.ObjectAlternative) (model.MethodTable, bool) {
	atom, ok := semtypes.ObjectPositiveAtom(t.typeContext(), alternative.Type())
	if !ok {
		return model.MethodTable{}, false
	}
	return t.objectMethodTable(atom)
}

func initSignature(t typeResolver, table model.MethodTable, pos diagnostics.Location) (model.UntypedFunctionSignature, model.SymbolRef, bool) {
	initRef, ok := table.Methods["init"]
	if !ok {
		return model.UntypedFunctionSignature{}, model.SymbolRef{}, true
	}
	signature, ok := t.functionSignature(initRef)
	if !ok {
		t.internalError("init method has no function signature", pos)
		return model.UntypedFunctionSignature{}, model.SymbolRef{}, false
	}
	return signature, initRef, true
}

func resolveObjectInitArgs(t typeResolver, chain *binding, e *ast.BLangNewExpression, alternative semtypes.ObjectAlternative,
	table model.MethodTable,
) ([]ast.BLangExpression, semtypes.SemType, *binding, bool) {
	signature, initRef, ok := initSignature(t, table, e.GetPosition())
	if !ok {
		return nil, semtypes.SemType{}, chain, false
	}
	args, fail := lowerInvocationArgsWithSignature(t, e.ArgsExprs, signature, initRef, semtypes.SemType{}, e.GetPosition())
	if fail != nil {
		fail(t)
		return nil, semtypes.SemType{}, chain, false
	}
	if initRef.IsEmpty() {
		return args, semtypes.Nil, chain, true
	}
	cx := t.typeContext()
	paramListTy := semtypes.FunctionParamListType(cx, alternative.InitFunctionType())
	argTys, chain, ok := resolveArgs(t, args, chain, func(i int) semtypes.SemType {
		return semtypes.ListMemberTypeInnerVal(cx, paramListTy, semtypes.IntConst(int64(i)))
	})
	if !ok {
		return nil, semtypes.SemType{}, chain, false
	}
	argListDefinition := semtypes.NewListDefinition()
	argList := argListDefinition.Define(t.typeEnv(), argTys,
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	if !semtypes.IsSubtype(cx, argList, paramListTy) {
		t.semanticError("incompatible arguments for function call", e.GetPosition())
		return nil, semtypes.SemType{}, chain, false
	}
	return args, semtypes.FunctionReturnType(cx, alternative.InitFunctionType(), argList), chain, true
}

func resolveTrialDependencies(t typeResolver, roots []ast.BLangExpression) bool {
	for _, ref := range collectTrialDependencyRefs(roots) {
		if !t.ensureResolved(ref, 0) {
			return false
		}
	}
	return true
}

func resolveStreamNewExpr(t typeResolver, chain *binding, e *ast.BLangNewExpression, streamTy semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if len(e.ArgsExprs) != 1 {
		t.semanticError("new stream expression requires exactly one argument", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	cx := t.typeContext()
	valueTy := semtypes.StreamValueType(cx, streamTy)
	completionTy := semtypes.StreamCompletionType(cx, streamTy)
	if semtypes.IsZero(valueTy) || semtypes.IsZero(completionTy) {
		t.internalError("failed to extract stream type parameters", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	implTy := semtypes.CreateStreamImplementorType(cx, valueTy, completionTy)
	argResult, ok := resolveActionOrExpression(t, chain, e.ArgsExprs[0], implTy)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, e, streamTy)
	return streamTy, defaultExpressionEffect(sequentialChain(t, argResult.effect)), true
}

func resolveTypeTestExpr(t typeResolver, chain *binding, e *ast.BLangTypeTestExpr) (semtypes.SemType, expressionEffect, bool) {
	exprResult, ok := resolveActionOrExpression(t, chain, e.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	exprTy := exprResult.ty
	chain = sequentialChain(t, exprResult.effect)
	testedTy, chain, ok := resolveTypeData(t, chain, &e.TestedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	var resultTy semtypes.SemType
	if semtypes.IsSubtype(t.typeContext(), exprTy, testedTy) {
		resultTy = semtypes.BooleanConst(!e.IsNegation())
	} else if semtypes.IsEmpty(t.typeContext(), semtypes.Intersect(exprTy, testedTy)) {
		resultTy = semtypes.BooleanConst(e.IsNegation())
	} else {
		resultTy = semtypes.Boolean
	}

	setNodeType(t, e, resultTy)

	ref, isVarRef := varRefExp(t, chain, e.Expr)
	if !isVarRef || !canNarrow(t, chain, ref) {
		return resultTy, defaultExpressionEffect(chain), true
	}
	effect := narrowByType(t, chain, ref, testedTy)
	functionSignatures, ok := typeTestFunctionSignatureRefs(t, e.TestedType.TypeDescriptor, e.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if e.IsNegation() {
		effect.ifTrue, effect.ifFalse = effect.ifFalse, effect.ifTrue
		functionSignatures.ifTrue, functionSignatures.ifFalse = functionSignatures.ifFalse, functionSignatures.ifTrue
	}
	if !associateTypeTestFunctionSignature(t, effect.ifTrue.narrowedSymbol, functionSignatures.ifTrue, e.GetPosition()) ||
		!associateTypeTestFunctionSignature(t, effect.ifFalse.narrowedSymbol, functionSignatures.ifFalse, e.GetPosition()) {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	return resultTy, effect, true
}

type typeTestFunctionSignatures struct {
	ifTrue  model.FunctionSignatureRef
	ifFalse model.FunctionSignatureRef
}

func typeTestFunctionSignatureRefs(t typeResolver, typeDescriptor ast.TypeDescriptor, pos diagnostics.Location) (typeTestFunctionSignatures, bool) {
	var ref model.FunctionSignatureRef
	switch ty := typeDescriptor.(type) {
	case *ast.BLangFunctionType:
		if ty.IsAnyFunction() {
			return typeTestFunctionSignatures{}, true
		}
		ref = ty.SignatureRef()
		if ref == 0 {
			t.internalError("function type signature not found", pos)
			return typeTestFunctionSignatures{}, false
		}
	case *ast.BLangUserDefinedType:
		ref, _ = t.functionSignatureRef(ty.Symbol())
	}
	return typeTestFunctionSignatures{ifTrue: ref}, true
}

func associateTypeTestFunctionSignature(t typeResolver, narrowed model.SymbolRef, ref model.FunctionSignatureRef, pos diagnostics.Location) bool {
	if ref == 0 {
		return true
	}
	if !t.associateFunctionSignature(narrowed, ref) {
		t.internalError("function signature already set", pos)
		return false
	}
	return true
}

func resolveStartAction(t typeResolver, chain *binding, e *ast.BLangStartAction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	var callExpectedType semtypes.SemType
	if !semtypes.IsZero(expectedType) {
		futureExpectedType := semtypes.Intersect(expectedType, semtypes.Future)
		if semtypes.IsEmpty(t.typeContext(), futureExpectedType) {
			t.semanticError("start action requires a future expected type", e.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		callExpectedType = semtypes.FutureEventualType(t.typeContext(), futureExpectedType)
	}
	callResult, ok := resolveActionOrExpression(t, chain, e.Call, callExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.FutureContaining(t.typeEnv(), callResult.ty)
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(sequentialChain(t, callResult.effect)), true
}

func resolveSingleWaitAction(t typeResolver, chain *binding, e *ast.BLangSingleWaitAction) (semtypes.SemType, expressionEffect, bool) {
	futureResult, ok := resolveActionOrExpression(t, chain, e.FutureExpr, semtypes.Future)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	futureTy := futureResult.ty
	if !semtypes.IsSubtype(t.typeContext(), futureTy, semtypes.Future) {
		t.semanticError("wait action requires a future expression", e.FutureExpr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Union(semtypes.FutureEventualType(t.typeContext(), futureTy), semtypes.Error)
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(sequentialChain(t, futureResult.effect)), true
}

func resolveAlternateWaitAction(t typeResolver, chain *binding, e *ast.BLangAlternateWaitAction) (semtypes.SemType, expressionEffect, bool) {
	resultTy := semtypes.Error
	for _, futureExpr := range e.FutureExprs {
		futureResult, ok := resolveActionOrExpression(t, chain, futureExpr, semtypes.Future)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = sequentialChain(t, futureResult.effect)
		futureTy := futureResult.ty
		if !semtypes.IsSubtype(t.typeContext(), futureTy, semtypes.Future) {
			t.semanticError("wait action requires a future expression", futureExpr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		resultTy = semtypes.Union(resultTy, semtypes.FutureEventualType(t.typeContext(), futureTy))
	}
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveMultipleWaitAction(t typeResolver, chain *binding, e *ast.BLangMultipleWaitAction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if len(e.FutureExprs) != len(e.FieldNames) {
		t.internalError("multiple wait field and expression count mismatch", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	fields := make([]semtypes.Field, len(e.FutureExprs))
	seen := make(map[string]struct{}, len(e.FieldNames))
	for i, futureExpr := range e.FutureExprs {
		fieldName := e.FieldNames[i]
		if _, exists := seen[fieldName]; exists {
			t.semanticError(fmt.Sprintf("duplicate field name '%s'", fieldName), futureExpr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		seen[fieldName] = struct{}{}
		futureResult, ok := resolveActionOrExpression(t, chain, futureExpr, semtypes.Future)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		futureTy := futureResult.ty
		chain = sequentialChain(t, futureResult.effect)
		if !semtypes.IsSubtype(t.typeContext(), futureTy, semtypes.Future) {
			t.semanticError("wait action requires a future expression", futureExpr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		fieldTy := semtypes.Union(semtypes.FutureEventualType(t.typeContext(), futureTy), semtypes.Error)
		fields[i] = semtypes.FieldFrom(fieldName, fieldTy, false, false)
	}

	resultTy := createClosedRecordType(t.typeEnv(), fields, semtypes.Never)
	if !semtypes.IsZero(expectedType) {
		expectedMappingTy := semtypes.Intersect(expectedType, semtypes.Mapping)
		if semtypes.IsEmpty(t.typeContext(), expectedMappingTy) || !semtypes.IsSubtype(t.typeContext(), resultTy, expectedMappingTy) {
			t.semanticError("multiple wait result is incompatible with expected type", e.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if semtypes.ToMappingAtomicType(t.typeContext(), expectedMappingTy) != nil {
			resultTy = expectedMappingTy
		}
	}
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveTrapExpr(t typeResolver, chain *binding, e *ast.BLangTrapExpr) (semtypes.SemType, expressionEffect, bool) {
	exprResult, ok := resolveActionOrExpression(t, chain, e.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	exprTy := exprResult.ty
	resultTy := semtypes.Union(exprTy, semtypes.Error)
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(sequentialChain(t, exprResult.effect)), true
}

func resolveCheckedExpr(t typeResolver, chain *binding, e *ast.BLangCheckedExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	var innerExpected semtypes.SemType
	if !semtypes.IsZero(expectedType) {
		innerExpected = semtypes.Union(expectedType, semtypes.Error)
	}
	exprResult, ok := resolveActionOrExpression(t, chain, e.Expr, innerExpected)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	exprTy := exprResult.ty
	resultTy := semtypes.Diff(exprTy, semtypes.Error)
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(sequentialChain(t, exprResult.effect)), true
}

func resolveMappingConstructorExpr(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) (expressionResult, bool) {
	if semtypes.IsZero(expectedType) {
		return resolveMappingConstructorBottomUp(t, chain, e, false)
	}
	candidates, ok := mappingConstructorCandidates(t, e, expectedType)
	if !ok {
		return expressionResult{}, false
	}
	if len(candidates) == 1 {
		return finishMappingConstructor(t, chain, e, candidates[0])
	}
	// A constructor inside a trial is part of an outer trial root, whose dependencies are already resolved.
	if !t.isEphemeral() && !resolveTrialDependencies(t, []ast.BLangExpression{e}) {
		return expressionResult{}, false
	}
	selected, outcome := selectCandidate(t, candidates, func(candidate mappingCandidate) bool {
		_, ok := finishMappingConstructor(t, chain, e, candidate)
		return ok
	})
	switch outcome {
	case candidateOne:
		return finishMappingConstructor(t, chain, e, selected)
	case candidateAmbiguous:
		t.semanticError("ambiguous inherent type for mapping constructor", e.GetPosition())
	case candidateUnsupported:
	default:
		t.semanticError("no applicable inherent type for mapping constructor", e.GetPosition())
	}
	return expressionResult{}, false
}

type mappingCandidate struct {
	alt  semtypes.MappingAlternative
	atom *semtypes.MappingAtomicType
}

func mappingConstructorCandidates(t typeResolver, e *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) ([]mappingCandidate, bool) {
	return constructorCandidates(t, "mapping", e.GetPosition(), semtypes.Intersect(expectedType, semtypes.Mapping), semtypes.MappingAlternatives,
		func(alt semtypes.MappingAlternative) (mappingCandidate, bool) {
			atom := semtypes.ToMappingAtomicType(t.typeContext(), alt.Type())
			return mappingCandidate{alt: alt, atom: atom}, atom != nil
		})
}

type constructorAlternative interface {
	Type() semtypes.SemType
	HasNegativeAtoms() bool
}

// constructorCandidates builds one candidate per alternative of part, the list or mapping part of a constructor's
// expected type. Every alternative is built from an intersected atom, so candidate fails only on a compiler bug.
func constructorCandidates[A constructorAlternative, C any](t typeResolver, kind string, pos diagnostics.Location, part semtypes.SemType,
	alternatives func(semtypes.Context, semtypes.SemType) []A, candidate func(A) (C, bool),
) ([]C, bool) {
	if !t.ensureNotEmpty(part, func() {
		t.semanticError(kind+" type not found in expected type", pos)
	}) {
		return nil, false
	}
	alts := alternatives(t.typeContext(), part)
	if len(alts) == 0 {
		// Only an empty part has no alternatives, which ensureNotEmpty reports once the type env is ready.
		if t.typeEnv().IsReady() {
			t.internalError(kind+" constructor expected type has no alternatives", pos)
		}
		return nil, false
	}
	candidates := make([]C, len(alts))
	for i, alt := range alts {
		if alt.HasNegativeAtoms() {
			t.internalError(kind+" constructor alternative has negative atoms", pos)
			return nil, false
		}
		c, ok := candidate(alt)
		if !ok {
			t.internalError(kind+" constructor alternative is not atomic", pos)
			return nil, false
		}
		candidates[i] = c
	}
	return candidates, true
}

func finishMappingConstructor(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, candidate mappingCandidate) (expressionResult, bool) {
	fields, chain, ok := resolveMappingConstructorFields(t, chain, e, candidate)
	if !ok {
		return expressionResult{}, false
	}
	// Inside a trial the constructor must fit the candidate, so that an outer trial can tell candidates apart by an
	// inner constructor. Outside a trial a misfit is reported by semantic analysis at the field.
	if t.isEphemeral() && !semtypes.MappingAlternativeAllowsFields(t.typeContext(), candidate.alt, fields, func(atom *semtypes.MappingAtomicType) []string {
		return defaultableMappingFields(t, atom)
	}) {
		return expressionResult{}, false
	}
	var fieldDefaults []model.FieldDefault
	if defaults, found := t.mappingDefaults(candidate.atom); found {
		fieldDefaults = append([]model.FieldDefault(nil), defaults...)
	}
	setMappingConstructorInherentType(t, e, *candidate.atom, fieldDefaults)
	inherentTy := candidate.alt.Type()
	if len(e.ReadonlyFields) != 0 {
		// The inherent type the value is constructed with narrows the explicitly readonly
		// fields to immutable cells; the shape validated against remains SelectedAtomicType.
		inherentTy = semtypes.MappingWithReadonlyFields(t.typeEnv(), candidate.atom, e.ReadonlyFields)
	}
	setNodeType(t, e, inherentTy)
	return expressionResult{ty: inherentTy, effect: defaultExpressionEffect(chain)}, true
}

// resolveMappingConstructorFields resolves each field value against the candidate's field type for its key. A
// computed key takes the member type for the key's type and is not part of the returned fields.
func resolveMappingConstructorFields(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, candidate mappingCandidate) ([]semtypes.MappingFieldInfo, *binding, bool) {
	fields := make([]semtypes.MappingFieldInfo, 0, len(e.Fields))
	for _, f := range e.Fields {
		kv := f.(*ast.BLangMappingKeyValueField)
		computed := kv.Key.Kind == ast.MappingKeyComputed
		var keyName string
		var required semtypes.SemType
		if computed {
			keyTy, keyChain, ok := resolveComputedMappingKey(t, chain, kv)
			if !ok {
				return nil, chain, false
			}
			chain = keyChain
			required = semtypes.MappingMemberTypeInnerVal(t.typeContext(), candidate.alt.Type(), semtypes.Intersect(keyTy, semtypes.String))
		} else {
			name, ok := common.MappingKeyName(t.compilerContext(), kv.Key)
			if !ok {
				return nil, chain, false
			}
			keyName = name
			readonlyRequired, ok := readonlyFieldRequiredType(t, e, kv, candidate.atom.FieldInnerVal(keyName))
			if !ok {
				return nil, chain, false
			}
			required = readonlyRequired
		}
		result, ok := resolveActionOrExpression(t, chain, kv.ValueExpr, required)
		if !ok {
			return nil, chain, false
		}
		chain = sequentialChain(t, result.effect)
		if computed {
			continue
		}
		resolveMappingKey(t, kv, result.ty)
		fields = append(fields, semtypes.MappingFieldInfo{Name: keyName, Type: result.ty})
	}
	return fields, chain, true
}

// resolveMemberValue resolves a constructor member that has no contextually expected type.
func resolveMemberValue(t typeResolver, chain *binding, expr ast.BLangExpression, readonly bool) (expressionResult, bool) {
	if readonly {
		return resolveReadonlyMember(t, chain, expr)
	}
	return resolveActionOrExpression(t, chain, expr, semtypes.SemType{})
}

// resolveReadonlyMember resolves the value of a readonly member of a constructor that has no
// contextually expected type. Such a value has no expectation either, but it must be constructed
// readonly: a constructor infers its shape and makes it readonly, and anything else is resolved
// against readonly. Resolving a constructor against readonly instead would give it the whole
// readonly mapping or list as its inherent type, losing the shape the member's type must keep.
func resolveReadonlyMember(t typeResolver, chain *binding, expr ast.BLangExpression) (expressionResult, bool) {
	switch e := expr.(type) {
	case *ast.BLangMappingConstructorExpr:
		return resolveMappingConstructorBottomUp(t, chain, e, true)
	case *ast.BLangListConstructorExpr:
		return resolveListConstructorInner(t, chain, e, true)
	case *ast.BLangGroupExpr:
		result, ok := resolveReadonlyMember(t, chain, e.Expression)
		if ok {
			setNodeType(t, e, result.ty)
		}
		return result, ok
	default:
		return resolveActionOrExpression(t, chain, expr, semtypes.ValReadonly)
	}
}

// inferredMemberType is the type a member of type ty gets in the inherent type a constructor
// infers without a contextual expectation. A readonly structured member keeps its shape, since
// widening its single shape would leave only its basic type.
func inferredMemberType(ty semtypes.SemType, readonly bool) semtypes.SemType {
	if readonly && semtypes.IsSubtypeSimple(ty, semtypes.Union(semtypes.List, semtypes.Mapping)) {
		return ty
	}
	return widenedListMemberType(ty)
}

func resolveMappingConstructorBottomUp(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, readonlyValue bool) (expressionResult, bool) {
	var fields []semtypes.Field
	restTy := semtypes.Never
	for _, f := range e.Fields {
		kv := f.(*ast.BLangMappingKeyValueField)
		if kv.Key.Kind == ast.MappingKeyComputed {
			var ok bool
			_, chain, ok = resolveComputedMappingKey(t, chain, kv)
			if !ok {
				return expressionResult{}, false
			}
			valueResult, ok := resolveMemberValue(t, chain, kv.ValueExpr, readonlyValue)
			if !ok {
				return expressionResult{}, false
			}
			chain = sequentialChain(t, valueResult.effect)
			restTy = semtypes.Union(restTy, inferredMemberType(valueResult.ty, readonlyValue))
			continue
		}
		keyName, ok := common.MappingKeyName(t.compilerContext(), kv.Key)
		if !ok {
			return expressionResult{}, false
		}
		readonly := readonlyValue || e.IsReadonly(keyName)
		valueResult, ok := resolveMemberValue(t, chain, kv.ValueExpr, readonly)
		if !ok {
			return expressionResult{}, false
		}
		chain = sequentialChain(t, valueResult.effect)
		fieldTy := valueResult.ty
		if !e.IsReadonly(keyName) {
			fieldTy = inferredMemberType(valueResult.ty, readonly)
		}
		resolveMappingKey(t, kv, valueResult.ty)
		fields = append(fields, semtypes.FieldFrom(keyName, fieldTy, readonly, false))
	}
	mutability := semtypes.CellMutabilityLimited
	if readonlyValue {
		mutability = semtypes.CellMutabilityNone
	}
	md := semtypes.NewMappingDefinition()
	mapTy := md.Define(t.typeEnv(), fields, restTy, semtypes.MappingMutability(mutability))
	setNodeType(t, e, mapTy)
	mat := semtypes.ToMappingAtomicType(t.typeContext(), mapTy)
	setMappingConstructorInherentType(t, e, *mat, nil)
	return expressionResult{ty: mapTy, effect: defaultExpressionEffect(chain)}, true
}

// resolveComputedMappingKey resolves a computed key as an ordinary expression,
// so a variable it refers to keeps its own type.
func resolveComputedMappingKey(t typeResolver, chain *binding, kv *ast.BLangMappingKeyValueField) (semtypes.SemType, *binding, bool) {
	keyResult, ok := resolveActionOrExpression(t, chain, kv.Key.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, nil, false
	}
	return keyResult.ty, sequentialChain(t, keyResult.effect), true
}

// readonlyFieldRequiredType narrows the type an explicitly readonly field's value must have to its
// readonly part. A computed key cannot be marked readonly.
func readonlyFieldRequiredType(t typeResolver, e *ast.BLangMappingConstructorExpr, kv *ast.BLangMappingKeyValueField, memberTy semtypes.SemType) (semtypes.SemType, bool) {
	if kv.Key.Kind == ast.MappingKeyComputed {
		return memberTy, true
	}
	keyName, ok := common.MappingKeyName(t.compilerContext(), kv.Key)
	if !ok {
		return semtypes.SemType{}, false
	}
	if !e.IsReadonly(keyName) {
		return memberTy, true
	}
	requiredType := semtypes.Intersect(memberTy, semtypes.ValReadonly)
	if semtypes.IsEmpty(t.typeContext(), requiredType) {
		t.semanticError(fmt.Sprintf("field '%s' cannot be readonly: its type in the inherent type has no readonly values", keyName),
			kv.GetPosition())
		return semtypes.SemType{}, false
	}
	return requiredType, true
}

func resolveMappingKey(t typeResolver, kv *ast.BLangMappingKeyValueField, valueTy semtypes.SemType) {
	if keyExpr, ok := kv.Key.Expr.(*ast.BLangLiteral); ok {
		resolveLiteral(t, keyExpr, semtypes.SemType{})
	}
	setMappingKey(t, kv, valueTy)
}

func defaultableMappingFields(t typeResolver, atom *semtypes.MappingAtomicType) []string {
	defaults, _ := t.mappingDefaults(atom)
	fields := make([]string, len(defaults))
	for i, field := range defaults {
		fields[i] = field.FieldName
	}
	return fields
}

func resolveTypeConversionExpr(t typeResolver, chain *binding, e *ast.BLangTypeConversionExpr) (semtypes.SemType, expressionEffect, bool) {
	expectedType, chain, ok := resolveBType(t, chain, e.TypeDescriptor, 0)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	exprResult, ok := resolveActionOrExpression(t, chain, e.Expression, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	setNodeType(t, e, expectedType)
	return expectedType, defaultExpressionEffect(sequentialChain(t, exprResult.effect)), true
}

// Helper functions for expression type checking

func resolveQueryExpr(
	t typeResolver,
	chain *binding,
	expr *ast.BLangQueryExpr,
	expectedType semtypes.SemType,
) (semtypes.SemType, expressionEffect, bool) {
	if t.isEphemeral() {
		t.unimplemented("query expressions are not supported while selecting among candidate types", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if len(expr.QueryClauseList) < 2 {
		t.semanticError("query expression requires from and select clauses", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	fromClause, ok := expr.QueryClauseList[0].(*ast.BLangFromClause)
	if !ok {
		t.semanticError("query expression must start with a from clause", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	lastClauseIndex := len(expr.QueryClauseList) - 1
	var onConflictClause *ast.BLangOnConflictClause
	if clause, isOnConflict := expr.QueryClauseList[lastClauseIndex].(*ast.BLangOnConflictClause); isOnConflict {
		onConflictClause = clause
		lastClauseIndex--
	}
	if lastClauseIndex < 1 {
		t.semanticError("query expression requires a select or collect clause", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	var (
		selectClause  *ast.BLangSelectClause
		collectClause *ast.BLangCollectClause
		finalOK       bool
	)
	selectClause, finalOK = expr.QueryClauseList[lastClauseIndex].(*ast.BLangSelectClause)
	if !finalOK {
		collectClause, finalOK = expr.QueryClauseList[lastClauseIndex].(*ast.BLangCollectClause)
	}
	if !finalOK {
		t.semanticError("query expression requires a select or collect clause", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	collectionResult, ok := resolveActionOrExpression(t, chain, fromClause.Collection, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	collectionTy := collectionResult.ty
	// The initial collection is evaluated once, before the query's repeated region;
	// every subsequent clause and terminal expression sees the query's captures.
	chain = addCaptureGroup(sequentialChain(t, collectionResult.effect), expr.CaptureGroup())
	elementTy, ok := resolveQueryCollectionElementType(t, collectionTy, fromClause.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	if fromClause.VariableDefinitionNode != nil {
		varDef := fromClause.VariableDefinitionNode
		if varDef.Var == nil {
			t.unimplemented("only simple variable bindings are supported in from clause", fromClause.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}

		variableTy := elementTy
		if !fromClause.IsDeclaredWithVarFlag && varDef.Var.TypeNode() != nil {
			variableTy, chain, ok = resolveBType(t, chain, varDef.Var.TypeNode(), 0)
			if !ok {
				return semtypes.SemType{}, expressionEffect{}, false
			}
			if !semtypes.IsSubtype(t.typeContext(), elementTy, variableTy) {
				t.semanticError("from clause variable type is incompatible with collection member type",
					varDef.GetPosition())
				return semtypes.SemType{}, expressionEffect{}, false
			}
		}

		updateSymbolType(t, varDef.Var, variableTy)
	}

	queryChain, ok := resolveQueryIntermediateClauses(t, chain, expr, lastClauseIndex)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	var queryTy semtypes.SemType
	if selectClause != nil {
		selectExpectedTy := common.QuerySelectExpectedType(
			t.typeContext(),
			t.typeEnv(),
			expr.QueryConstructType,
			expectedType,
		)
		selectResult, ok := resolveActionOrExpression(t, queryChain, selectClause.Expression, selectExpectedTy)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		selectTy := selectResult.ty
		switch expr.QueryConstructType {
		case ast.TypeKindNone:
			ld := semtypes.NewListDefinition()
			queryTy = ld.Define(t.typeEnv(), nil, semtypes.ListRest(selectTy))
		case ast.TypeKindMap:
			expectedSelectTy := common.MapQuerySelectExpectedType(t.typeEnv())
			if !semtypes.IsSubtype(t.typeContext(), selectTy, expectedSelectTy) {
				t.semanticError(
					common.FormatIncompatibleTypeMessage(t.typeContext(), expectedSelectTy, selectTy),
					selectClause.GetPosition(),
				)
				return semtypes.SemType{}, expressionEffect{}, false
			}
			valueTy := semtypes.ListMemberTypeInnerVal(t.typeContext(), selectTy, semtypes.IntConst(1))
			md := semtypes.NewMappingDefinition()
			queryTy = md.Define(t.typeEnv(), nil, valueTy)
		default:
			t.unimplemented("query construct type is not supported yet", expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
	} else {
		if expr.QueryConstructType != ast.TypeKindNone {
			t.semanticError("query construct types cannot be used with collect clause", collectClause.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		collectChain := queryChain
		groupAggregatedSymbols := queryGroupAggregatedSymbolsBeforeClause(expr, lastClauseIndex)
		for _, variable := range queryVariablesBeforeClause(expr, lastClauseIndex) {
			if groupAggregatedSymbols[variable.symbol] {
				continue
			}
			collectChain = aggregateQueryVariable(t, collectChain, variable, false)
		}
		collectResult, ok := resolveActionOrExpression(
			t,
			collectChain,
			collectClause.Expression,
			expectedType,
		)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		collectTy := collectResult.ty
		queryTy = collectTy
	}

	if onConflictClause != nil {
		if expr.QueryConstructType != ast.TypeKindMap {
			t.semanticError("on conflict clause is supported only for map query construct type",
				onConflictClause.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		conflictResult, ok := resolveActionOrExpression(t, queryChain, onConflictClause.Expression, semtypes.Union(semtypes.Error, semtypes.Nil))
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		conflictTy := conflictResult.ty
		if !semtypes.IsSubtype(t.typeContext(), conflictTy, semtypes.Union(semtypes.Error, semtypes.Nil)) {
			t.semanticError("on conflict clause expression must be error?", onConflictClause.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		errorTy := semtypes.Intersect(conflictTy, semtypes.Error)
		if !semtypes.IsEmpty(t.typeContext(), errorTy) {
			queryTy = semtypes.Union(queryTy, errorTy)
		}
	}
	setNodeType(t, expr, queryTy)
	return queryTy, defaultExpressionEffect(chain), true
}

func resolveQueryCollectionElementType(
	t typeResolver,
	collectionTy semtypes.SemType,
	pos diagnostics.Location,
) (semtypes.SemType, bool) {
	switch {
	case semtypes.IsSubtype(t.typeContext(), collectionTy, semtypes.List):
		memberTypes := semtypes.ListAllMemberTypesInner(t.typeContext(), collectionTy)
		result := semtypes.Never
		for _, each := range memberTypes.Types {
			result = semtypes.Union(result, each)
		}
		return result, true
	case semtypes.IsSubtype(t.typeContext(), collectionTy, semtypes.Mapping):
		return semtypes.MappingMemberTypeInnerValProj(t.typeContext(), collectionTy, semtypes.String), true
	default:
		t.unimplemented("query from clause currently supports only list or map collections", pos)
		return semtypes.SemType{}, false
	}
}

func resolveForeachVariableType(t typeResolver, collection ast.BLangActionOrExpression, collectionTy semtypes.SemType) (semtypes.SemType, bool) {
	if binaryExpr, ok := collection.(*ast.BLangBinaryExpr); ok && common.IsRangeExpr(binaryExpr) {
		return semtypes.Int, true
	}
	ctx := t.typeContext()
	switch {
	case semtypes.IsSubtype(ctx, collectionTy, semtypes.List):
		return semtypes.ListMemberTypeInnerVal(ctx, collectionTy, semtypes.Int), true
	case semtypes.IsSubtype(ctx, collectionTy, semtypes.Mapping):
		return semtypes.MappingMemberTypeInnerVal(ctx, collectionTy, semtypes.String), true
	case semtypes.IsSubtype(ctx, collectionTy, semtypes.XML):
		return semtypes.XMLItemType(collectionTy), true
	case semtypes.IsSubtype(ctx, collectionTy, semtypes.String):
		return semtypes.Char, true
	default:
		iterableTy, ok := common.IterableType(t.compilerContext())
		if !ok {
			t.semanticError("foreach collection must be subtype of object:Iterable", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		if !semtypes.IsSubtype(ctx, collectionTy, iterableTy) {
			t.semanticError("foreach collection must be subtype of object:Iterable", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		ld := semtypes.NewListDefinition()
		emptyListTy := ld.Define(t.typeEnv(), nil,
			semtypes.ListMutability(semtypes.CellMutabilityNone))
		iteratorFnTy := semtypes.ObjectMemberType(ctx, semtypes.StringConst("iterator"), collectionTy)
		if semtypes.IsZero(iteratorFnTy) || !semtypes.IsSubtype(ctx, iteratorFnTy, semtypes.Function) {
			t.semanticError("foreach collection is not iterable", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		iteratorTy := semtypes.FunctionReturnType(ctx, iteratorFnTy, emptyListTy)
		nextFnTy := semtypes.ObjectMemberType(ctx, semtypes.StringConst("next"), iteratorTy)
		if semtypes.IsZero(nextFnTy) || !semtypes.IsSubtype(ctx, nextFnTy, semtypes.Function) {
			t.semanticError("foreach iterator does not have a next method", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		nextReturnTy := semtypes.FunctionReturnType(ctx, nextFnTy, emptyListTy)
		valueRecordTy := semtypes.Diff(semtypes.Diff(nextReturnTy, semtypes.Nil), semtypes.Error)
		return semtypes.MappingMemberTypeInnerVal(ctx, valueRecordTy, semtypes.StringConst("value")), true
	}
}

type queryVariableInfo struct {
	name   string
	symbol model.SymbolRef
}

func queryVariablesBeforeClause(queryExpr *ast.BLangQueryExpr, endIndex int) []queryVariableInfo {
	var variables []queryVariableInfo
	seen := make(map[model.SymbolRef]bool)
	for i := 0; i < endIndex; i++ {
		switch clause := queryExpr.QueryClauseList[i].(type) {
		case *ast.BLangFromClause:
			variables = appendQueryVariableInfo(variables, seen, clause.VariableDefinitionNode)
		case *ast.BLangJoinClause:
			variables = appendQueryVariableInfo(variables, seen, clause.VariableDefinitionNode)
		case *ast.BLangLetClause:
			for i := range clause.LetVarDeclarations {
				variables = appendQueryVariableInfo(variables, seen, &clause.LetVarDeclarations[i])
			}
		case *ast.BLangGroupByClause:
			for i := range clause.GroupingKeyList {
				variables = appendQueryVariableInfo(variables, seen, clause.GroupingKeyList[i].VariableDef)
			}
		}
	}
	return variables
}

func queryGroupAggregatedSymbolsBeforeClause(queryExpr *ast.BLangQueryExpr, endIndex int) map[model.SymbolRef]bool {
	aggregated := make(map[model.SymbolRef]bool)
	for i := 0; i < endIndex; i++ {
		groupByClause, ok := queryExpr.QueryClauseList[i].(*ast.BLangGroupByClause)
		if !ok || groupByClause.NonGroupingKeys == nil {
			continue
		}
		for _, variable := range queryVariablesBeforeClause(queryExpr, i) {
			if variable.name != "" && groupByClause.NonGroupingKeys.Contains(variable.name) {
				aggregated[variable.symbol] = true
			}
		}
	}
	return aggregated
}

func appendQueryVariableInfo(
	variables []queryVariableInfo,
	seen map[model.SymbolRef]bool,
	variableDef *ast.BLangVariableDef,
) []queryVariableInfo {
	varDef := variableDef
	if varDef == nil || varDef.Var == nil || !ast.SymbolIsSet(varDef.Var) {
		return variables
	}
	symbol := varDef.Var.Symbol()
	if seen[symbol] {
		return variables
	}
	seen[symbol] = true
	name := ""
	if varDef.Var.Name != nil {
		name = varDef.Var.Name.GetValue()
	}
	return append(variables, queryVariableInfo{
		name:   name,
		symbol: symbol,
	})
}

func queryAggregatedListType(env semtypes.Env, elemTy semtypes.SemType, nonEmpty bool) semtypes.SemType {
	if semtypes.IsZero(elemTy) {
		elemTy = semtypes.Any
	}
	ld := semtypes.NewListDefinition()
	if nonEmpty {
		return ld.Define(env, []semtypes.SemType{elemTy}, semtypes.ListRest(elemTy))
	}
	return ld.Define(env, nil, semtypes.ListRest(elemTy))
}

func aggregateQueryVariable(t typeResolver, chain *binding, variable queryVariableInfo, nonEmpty bool) *binding {
	effectiveSymbol := lookupSymbol(t, chain, variable.symbol)
	elemTy := t.symbolType(effectiveSymbol)
	aggregatedTy := queryAggregatedListType(t.typeEnv(), elemTy, nonEmpty)
	aggregatedSymbol := narrowSymbol(t, variable.symbol, aggregatedTy)
	return &binding{
		ref:            variable.symbol,
		narrowedSymbol: aggregatedSymbol,
		prev:           chain,
		flags:          bindingFlagQueryAggregated,
	}
}

func validateQueryGroupingKeyType(t typeResolver, keyTy semtypes.SemType, pos diagnostics.Location) bool {
	if !semtypes.IsSubtype(t.typeContext(), keyTy, semtypes.CreateAnydata(t.typeContext())) {
		t.semanticError("grouping key expression must be a subtype of anydata", pos)
		return false
	}
	return true
}

func resolveQueryGroupingKeyVarDef(t typeResolver, chain *binding, varDef *ast.BLangVariableDef) (semtypes.SemType, *binding, bool) {
	if varDef.Var == nil {
		t.unimplemented("only simple variable declarations are supported in group by clause", varDef.GetPosition())
		return semtypes.SemType{}, chain, false
	}
	if varDef.Var.Expr == nil {
		t.semanticError("group by variable declaration requires an initializer", varDef.GetPosition())
		return semtypes.SemType{}, chain, false
	}
	var variableTy semtypes.SemType
	if !varDef.Var.GetIsDeclaredWithVar() && varDef.Var.TypeNode() != nil {
		var ok bool
		variableTy, chain, ok = resolveBType(t, chain, varDef.Var.TypeNode(), 0)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
	}
	initResult, ok := resolveActionOrExpression(t, chain, varDef.Var.Expr.(ast.BLangExpression), variableTy)
	if !ok {
		return semtypes.SemType{}, chain, false
	}
	initTy := initResult.ty
	if semtypes.IsZero(variableTy) {
		variableTy = initTy
	} else if !semtypes.IsSubtype(t.typeContext(), initTy, variableTy) {
		t.semanticError("group by variable type is incompatible with initializer expression", varDef.GetPosition())
		return semtypes.SemType{}, chain, false
	}
	updateSymbolType(t, varDef.Var, variableTy)
	return variableTy, chain, true
}

func resolveQueryGroupByClause(
	t typeResolver,
	chain *binding,
	queryExpr *ast.BLangQueryExpr,
	clause *ast.BLangGroupByClause,
	clauseIndex int,
) (*binding, bool) {
	queryVariables := queryVariablesBeforeClause(queryExpr, clauseIndex)
	nonGroupingKeys := &balCommon.OrderedSet[string]{}
	for _, variable := range queryVariables {
		if variable.name != "" && variable.name != "_" {
			nonGroupingKeys.Add(variable.name)
		}
	}

	for i := range clause.GroupingKeyList {
		groupingKey := clause.GroupingKeyList[i]
		switch {
		case groupingKey.VariableRef != nil:
			keyResult, ok := resolveActionOrExpression(t, chain, groupingKey.VariableRef, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			keyTy := keyResult.ty
			if !validateQueryGroupingKeyType(t, keyTy, groupingKey.GetPosition()) {
				return nil, false
			}
			if groupingKey.VariableRef.VariableName != nil {
				nonGroupingKeys.Remove(groupingKey.VariableRef.VariableName.GetValue())
			}
		case groupingKey.VariableDef != nil:
			keyTy, _, ok := resolveQueryGroupingKeyVarDef(t, chain, groupingKey.VariableDef)
			if !ok {
				return nil, false
			}
			if !validateQueryGroupingKeyType(t, keyTy, groupingKey.GetPosition()) {
				return nil, false
			}
			if groupingKey.VariableDef.Var.Name != nil {
				nonGroupingKeys.Remove(groupingKey.VariableDef.Var.Name.GetValue())
			}
		default:
			t.semanticError("group by clause requires a grouping key", groupingKey.GetPosition())
			return nil, false
		}
	}
	setGroupByNonGroupingKeys(t, clause, nonGroupingKeys)

	resultChain := chain
	for _, variable := range queryVariables {
		if variable.name != "" && nonGroupingKeys.Contains(variable.name) {
			resultChain = aggregateQueryVariable(t, resultChain, variable, true)
		}
	}
	return resultChain, true
}

func resolveQueryIntermediateClauses(t typeResolver, chain *binding, queryExpr *ast.BLangQueryExpr, selectClauseIndex int) (*binding, bool) {
	currentChain := chain
	for i := 1; i < selectClauseIndex; i++ {
		switch clause := queryExpr.QueryClauseList[i].(type) {
		case *ast.BLangJoinClause:
			collectionResult, ok := resolveActionOrExpression(t, currentChain, clause.Collection, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			collectionTy := collectionResult.ty
			elementTy, ok := resolveQueryCollectionElementType(t, collectionTy, clause.GetPosition())
			if !ok {
				return nil, false
			}
			varDef := clause.VariableDefinitionNode
			if varDef == nil || varDef.Var == nil {
				t.unimplemented("only simple variable bindings are supported in join clause", clause.GetPosition())
				return nil, false
			}
			if clause.IsOuterJoinFlag && !clause.IsDeclaredWithVarFlag {
				t.semanticError("outer join clause variable must be declared with var", clause.GetPosition())
				return nil, false
			}
			variableTy := elementTy
			if clause.IsOuterJoinFlag {
				variableTy = semtypes.Union(variableTy, semtypes.Nil)
			}
			if !clause.IsDeclaredWithVarFlag && varDef.Var.TypeNode() != nil {
				variableTy, currentChain, ok = resolveBType(t, currentChain, varDef.Var.TypeNode(), 0)
				if !ok {
					return nil, false
				}
				if !semtypes.IsSubtype(t.typeContext(), elementTy, variableTy) {
					t.semanticError("join clause variable type is incompatible with collection member type",
						varDef.GetPosition())
					return nil, false
				}
			}
			updateSymbolType(t, varDef.Var, variableTy)

			if clause.OnClause.OnExpr == nil || clause.OnClause.EqualsExpr == nil {
				t.semanticError("join clause requires an on clause", clause.GetPosition())
				return nil, false
			}
			lhsResult, ok := resolveActionOrExpression(t, currentChain, clause.OnClause.OnExpr, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			lhsTy := lhsResult.ty
			currentChain = sequentialChain(t, lhsResult.effect)
			rhsResult, ok := resolveActionOrExpression(t, currentChain, clause.OnClause.EqualsExpr, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			rhsTy := rhsResult.ty
			currentChain = sequentialChain(t, rhsResult.effect)
			if !semtypes.IsSubtype(t.typeContext(), lhsTy, rhsTy) {
				t.semanticError(common.FormatIncompatibleTypeMessage(t.typeContext(), rhsTy, lhsTy), clause.OnClause.EqualsExpr.GetPosition())
				return nil, false
			}
		case *ast.BLangLetClause:
			for i := range clause.LetVarDeclarations {
				varDef := &clause.LetVarDeclarations[i]
				if varDef.Var == nil {
					t.unimplemented("only simple variable declarations are supported in let clause",
						clause.GetPosition())
					return nil, false
				}
				if varDef.Var.Expr == nil {
					t.semanticError("let clause variable declaration requires an initializer",
						varDef.GetPosition())
					return nil, false
				}
				initResult, ok := resolveActionOrExpression(t, currentChain, varDef.Var.Expr.(ast.BLangExpression), semtypes.SemType{})
				if !ok {
					return nil, false
				}
				initTy := initResult.ty
				currentChain = sequentialChain(t, initResult.effect)
				variableTy := initTy
				if !varDef.Var.GetIsDeclaredWithVar() && varDef.Var.TypeNode() != nil {
					variableTy, currentChain, ok = resolveBType(t, currentChain, varDef.Var.TypeNode(), 0)
					if !ok {
						return nil, false
					}
					if !semtypes.IsSubtype(t.typeContext(), initTy, variableTy) {
						t.semanticError("let clause variable type is incompatible with initializer expression",
							varDef.GetPosition())
						return nil, false
					}
				}
				updateSymbolType(t, varDef.Var, variableTy)
			}
		case *ast.BLangWhereClause:
			whereResult, ok := resolveActionOrExpression(t, currentChain, clause.Expression, semtypes.Boolean)
			if !ok {
				return nil, false
			}
			whereTy, effect := whereResult.ty, whereResult.effect
			if !semtypes.IsSubtype(t.typeContext(), whereTy, semtypes.Boolean) {
				t.semanticError("where clause expression must be boolean", clause.GetPosition())
				return nil, false
			}
			currentChain = effect.ifTrue
		case *ast.BLangGroupByClause:
			var ok bool
			currentChain, ok = resolveQueryGroupByClause(t, currentChain, queryExpr, clause, i)
			if !ok {
				return nil, false
			}
		case *ast.BLangLimitClause:
			limitResult, ok := resolveActionOrExpression(t, currentChain, clause.Expression, semtypes.Int)
			if !ok {
				return nil, false
			}
			limitTy := limitResult.ty
			currentChain = sequentialChain(t, limitResult.effect)
			if !semtypes.IsSubtype(t.typeContext(), limitTy, semtypes.Int) {
				t.semanticError("limit clause expression must be int", clause.GetPosition())
				return nil, false
			}
		case *ast.BLangOrderByClause:
			orderedTy := semtypes.CreateOrdered(t.typeContext())
			for j := range clause.OrderByKeyList {
				orderKey := &clause.OrderByKeyList[j]
				keyResult, ok := resolveActionOrExpression(t, currentChain, orderKey.Expression, semtypes.SemType{})
				if !ok {
					return nil, false
				}
				keyTy := keyResult.ty
				currentChain = sequentialChain(t, keyResult.effect)
				if !semtypes.IsSubtype(t.typeContext(), keyTy, orderedTy) ||
					!semtypes.Comparable(t.typeContext(), keyTy, keyTy) {
					t.semanticError("order by key expression must have an ordered type", orderKey.GetPosition())
					return nil, false
				}
			}
		default:
			t.unimplemented("only join + let + where + group by + order by + limit clauses are supported as intermediate query clauses", clause.GetPosition())
			return nil, false
		}
	}
	return currentChain, true
}

func resolveSimpleVarRef(t typeResolver, chain *binding, expr *ast.BLangVarRef) (expressionResult, bool) {
	baseSymbol := expr.Symbol()
	sym, isNarrowed := lookupBinding(t, chain, baseSymbol)
	if isNarrowed {
		setNodeSymbol(t, expr, sym)
	}
	if !t.ensureResolved(sym, 0) {
		return expressionResult{}, false
	}
	ty := t.symbolType(sym)
	if semtypes.IsZero(ty) {
		// A variable whose type failed to resolve has no type.
		return expressionResult{}, false
	}
	if t.getSymbol(sym).Kind() == model.SymbolKindType {
		ty = semtypes.TypedescContaining(t.typeEnv(), ty)
	}
	setNodeType(t, expr, ty)
	return expressionResult{ty: ty, effect: defaultExpressionEffect(chain), symbol: sym}, true
}

func resolveConstRef(t typeResolver, chain *binding, expr *ast.BLangConstRef) (expressionResult, bool) {
	sym, isNarrowed := lookupBinding(t, chain, expr.Symbol())
	if isNarrowed {
		setNodeSymbol(t, expr, sym)
	}
	if !t.ensureResolved(sym, 0) {
		return expressionResult{}, false
	}
	ty := t.symbolType(sym)
	setNodeType(t, expr, ty)
	return expressionResult{ty: ty, effect: defaultExpressionEffect(chain), symbol: sym}, true
}

func resolveListConstructorExpr(t typeResolver, chain *binding, expr *ast.BLangListConstructorExpr, expectedType semtypes.SemType) (expressionResult, bool) {
	if semtypes.IsZero(expectedType) {
		return resolveListConstructorInner(t, chain, expr, false)
	}
	candidates, ok := listConstructorCandidates(t, expr, expectedType)
	if !ok {
		return expressionResult{}, false
	}
	spread := listSpreadMembers(chain, expr)
	if len(candidates) == 1 {
		return finishListConstructor(t, chain, expr, spread, candidates[0])
	}
	// A constructor inside a trial is part of an outer trial root, whose dependencies are already resolved.
	if !t.isEphemeral() && !resolveTrialDependencies(t, []ast.BLangExpression{expr}) {
		return expressionResult{}, false
	}
	selected, outcome := selectCandidate(t, candidates, func(candidate listCandidate) bool {
		_, ok := finishListConstructor(t, chain, expr, spread, candidate)
		return ok
	})
	switch outcome {
	case candidateOne:
		return finishListConstructor(t, chain, expr, spread, selected)
	case candidateAmbiguous:
		t.semanticError("ambiguous inherent type for list constructor", expr.GetPosition())
	case candidateUnsupported:
	default:
		t.semanticError("no applicable inherent type for list constructor", expr.GetPosition())
	}
	return expressionResult{}, false
}

type listCandidate struct {
	alt  semtypes.ListAlternative
	atom *semtypes.ListAtomicType
}

func listConstructorCandidates(t typeResolver, expr *ast.BLangListConstructorExpr, expectedType semtypes.SemType) ([]listCandidate, bool) {
	return constructorCandidates(t, "list", expr.GetPosition(), semtypes.Intersect(expectedType, semtypes.List), semtypes.ListAlternatives,
		func(alt semtypes.ListAlternative) (listCandidate, bool) {
			atom := semtypes.ToListAtomicType(t.typeEnv(), alt.Type())
			return listCandidate{alt: alt, atom: atom}, atom != nil
		})
}

// listSpreadMembers classifies the members of expr that spread into the list: explicit spreads and references to
// query-aggregated variables.
func listSpreadMembers(chain *binding, expr *ast.BLangListConstructorExpr) []bool {
	spread := make([]bool, len(expr.Exprs))
	for i, memberExpr := range expr.Exprs {
		spread[i] = expr.IsSpreadMember(i) || isQueryAggregatedVariableReference(chain, memberExpr)
	}
	return spread
}

func finishListConstructor(t typeResolver, chain *binding, expr *ast.BLangListConstructorExpr, spread []bool, candidate listCandidate) (expressionResult, bool) {
	members, chain, ok := resolveListConstructorMembers(t, chain, expr, spread, candidate)
	if !ok {
		return expressionResult{}, false
	}
	// Inside a trial the constructor must fit the candidate, so that an outer trial can tell candidates apart by an
	// inner constructor. Outside a trial a misfit is reported by semantic analysis at the member.
	if t.isEphemeral() && !semtypes.ListAlternativeAllowsMembers(t.typeContext(), candidate.alt, members) {
		return expressionResult{}, false
	}
	setListConstructorInherentType(t, expr, *candidate.atom, spread)
	setNodeType(t, expr, candidate.alt.Type())
	return expressionResult{ty: candidate.alt.Type(), effect: defaultExpressionEffect(chain)}, true
}

// resolveListConstructorMembers resolves each member against the candidate's member type at its index. From the
// first spread member on, the index is the candidate's fixed length, for which the candidate yields its rest type.
func resolveListConstructorMembers(t typeResolver, chain *binding, expr *ast.BLangListConstructorExpr, spread []bool, candidate listCandidate) ([]semtypes.ListMemberInfo, *binding, bool) {
	members := make([]semtypes.ListMemberInfo, len(expr.Exprs))
	index := 0
	for i, memberExpr := range expr.Exprs {
		if spread[i] {
			index = max(index, candidate.atom.FixedLength())
		}
		required := candidate.atom.MemberAtInnerVal(index)
		if semtypes.IsNever(required) {
			if spread[i] {
				t.semanticError("aggregated variable reference cannot be used as a spread member for a fixed-length list constructor", memberExpr.GetPosition())
			} else {
				t.semanticError("too many members in list constructor", expr.GetPosition())
			}
			return nil, chain, false
		}
		if spread[i] {
			required = queryAggregatedListType(t.typeEnv(), required, false)
		}
		result, ok := resolveActionOrExpression(t, chain, memberExpr, required)
		if !ok {
			return nil, chain, false
		}
		chain = sequentialChain(t, result.effect)
		memberTy := result.ty
		if spread[i] {
			// A spread operand that is not a list fits no candidate. Outside a trial semantic analysis reports it.
			if t.isEphemeral() && !semtypes.IsSubtypeSimple(result.ty, semtypes.List) {
				return nil, chain, false
			}
			memberTy = semtypes.ListProj(t.typeContext(), semtypes.Intersect(result.ty, semtypes.List), semtypes.Int)
		}
		members[i] = semtypes.ListMemberInfo{Index: index, ValueType: memberTy}
		if index < candidate.atom.FixedLength() {
			index++
		}
	}
	return members, chain, true
}

func resolveListConstructorInner(t typeResolver, chain *binding, expr *ast.BLangListConstructorExpr, readonlyValue bool) (expressionResult, bool) {
	memberTypes := make([]semtypes.SemType, 0, len(expr.Exprs))
	restTy := semtypes.Never
	spreadMembers := make([]bool, len(expr.Exprs))
	hasSpread := false
	mutability := semtypes.CellMutabilityLimited
	if readonlyValue {
		mutability = semtypes.CellMutabilityNone
	}
	for i, memberExpr := range expr.Exprs {
		isSpread := expr.IsSpreadMember(i) || isQueryAggregatedVariableReference(chain, memberExpr)
		memberResult, ok := resolveMemberValue(t, chain, memberExpr, readonlyValue && !isSpread)
		if !ok {
			return expressionResult{}, false
		}
		memberTy := memberResult.ty
		chain = sequentialChain(t, memberResult.effect)
		if isSpread {
			spreadMembers[i] = true
			spreadMemberTy := semtypes.ListProj(t.typeContext(), memberTy, semtypes.Int)
			restTy = semtypes.Union(restTy, inferredMemberType(spreadMemberTy, readonlyValue))
			hasSpread = true
			continue
		}
		inferredTy := inferredMemberType(memberTy, readonlyValue)
		if hasSpread {
			restTy = semtypes.Union(restTy, inferredTy)
			continue
		}
		memberTypes = append(memberTypes, inferredTy)
	}

	ld := semtypes.NewListDefinition()
	listTy := ld.Define(t.typeEnv(), memberTypes, semtypes.ListRest(restTy), semtypes.ListMutability(mutability))

	setNodeType(t, expr, listTy)
	lat := semtypes.ToListAtomicType(t.typeEnv(), listTy)
	setListConstructorInherentType(t, expr, *lat, spreadMembers)

	return expressionResult{ty: listTy, effect: defaultExpressionEffect(chain)}, true
}

// contextualErrorType selects the error root an error constructor takes from its
// contextually expected type. An explicit type reference that is exactly `error`
// is refined the same way an omitted one is.
func contextualErrorType(t typeResolver, baseType, expectedType semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(expectedType) || !semtypes.IsSameType(t.typeContext(), baseType, semtypes.Error) {
		return baseType
	}
	errorPart := semtypes.Intersect(expectedType, semtypes.Error)
	if semtypes.IsEmpty(t.typeContext(), errorPart) {
		return baseType
	}
	return errorPart
}

func isQueryAggregatedVariableReference(chain *binding, expr ast.BLangExpression) bool {
	switch ref := expr.(type) {
	case *ast.BLangVarRef:
		return lookupQueryAggregatedBinding(chain, ref.Symbol())
	default:
		return false
	}
}

func widenedListMemberType(ty semtypes.SemType) semtypes.SemType {
	if semtypes.SingleShape(ty).IsEmpty() {
		return ty
	}
	return semtypes.WidenToBasicTypes(ty)
}

func resolveErrorConstructorExpr(t typeResolver, chain *binding, expr *ast.BLangErrorConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	var errorTy semtypes.SemType

	if expr.ErrorTypeRef != nil {
		refTy, typeChain, ok := resolveBType(t, chain, expr.ErrorTypeRef, 0)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = typeChain
		if !semtypes.IsSubtype(t.typeContext(), refTy, semtypes.Error) {
			t.semanticError("error type parameter must be a subtype of error", expr.ErrorTypeRef.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		} else {
			errorTy = refTy
		}
	} else {
		errorTy = semtypes.Error
	}

	errorTy = contextualErrorType(t, errorTy, expectedType)

	setNodeType(t, expr, errorTy)

	for _, arg := range expr.PositionalArgs {
		argResult, ok := resolveActionOrExpression(t, chain, arg, semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = sequentialChain(t, argResult.effect)
	}
	for i := range expr.NamedArgs {
		argResult, ok := resolveActionOrExpression(t, chain, &expr.NamedArgs[i], semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		chain = sequentialChain(t, argResult.effect)
	}
	return errorTy, defaultExpressionEffect(chain), true
}

func resolveUnaryExpr(t typeResolver, chain *binding, expr *ast.BLangUnaryExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	exprResult, ok := resolveActionOrExpression(t, chain, expr.Expr, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	exprTy, innerEffect := exprResult.ty, exprResult.effect
	chain = sequentialChain(t, innerEffect)

	// Check for nil lifting on numeric/bitwise unary operators
	nilLifted := false
	underlyingTy := exprTy
	if expr.GetOperatorKind() != model.OperatorKind_NOT {
		if semtypes.ContainsBasicType(exprTy, semtypes.Nil) {
			nilLifted = true
			underlyingTy = semtypes.Diff(exprTy, semtypes.Nil)
			if semtypes.IsEmpty(t.typeContext(), underlyingTy) {
				t.semanticError(fmt.Sprintf("expect numeric type for %s", string(expr.GetOperatorKind())), expr.GetPosition())
				return semtypes.SemType{}, expressionEffect{}, false
			}
		}
	}

	var resultTy semtypes.SemType
	switch expr.GetOperatorKind() {
	case model.OperatorKind_SUB:
		resultTy = negateNumericType(underlyingTy)
	case model.OperatorKind_ADD:
		resultTy = underlyingTy

	case model.OperatorKind_BITWISE_COMPLEMENT:
		if !semtypes.IsSubtype(t.typeContext(), underlyingTy, semtypes.Int) {
			t.semanticError(fmt.Sprintf("expect int type for %s", string(expr.GetOperatorKind())), expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if semtypes.IsSameType(t.typeContext(), underlyingTy, semtypes.Int) {
			resultTy = underlyingTy
			break
		}
		shape := semtypes.SingleShape(underlyingTy)
		if !shape.IsEmpty() {
			value, ok := shape.Get().Value.(int64)
			if !ok {
				t.internalError(fmt.Sprintf("unexpected singleton type for %s: %T", string(expr.GetOperatorKind()), shape.Get().Value), expr.GetPosition())
				return semtypes.SemType{}, expressionEffect{}, false
			}
			resultTy = semtypes.IntConst(^value)
		} else {
			resultTy = underlyingTy
		}

	case model.OperatorKind_NOT:
		if semtypes.IsSubtype(t.typeContext(), exprTy, semtypes.Boolean) {
			if semtypes.IsSameType(t.typeContext(), exprTy, semtypes.Boolean) {
				resultTy = semtypes.Boolean
			} else {
				resultTy = semtypes.Diff(semtypes.Boolean, exprTy)
			}
		} else {
			t.semanticError(fmt.Sprintf("expect boolean type for %s", string(expr.GetOperatorKind())), expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		setNodeType(t, expr, resultTy)
		return resultTy, expressionEffect{ifTrue: innerEffect.ifFalse, ifFalse: innerEffect.ifTrue}, true
	default:
		t.internalError(fmt.Sprintf("unsupported unary operator: %s", string(expr.GetOperatorKind())), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	if nilLifted {
		resultTy = semtypes.Union(semtypes.Nil, resultTy)
	}
	setNodeType(t, expr, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func negateNumericType(exprTy semtypes.SemType) semtypes.SemType {
	shape := semtypes.SingleShape(exprTy)
	if shape.IsEmpty() {
		return exprTy
	}
	switch v := shape.Get().Value.(type) {
	case int64:
		return semtypes.IntConst(v * -1)
	case float64:
		return semtypes.FloatConst(v * -1)
	case *decimal.Decimal:
		result := v.Neg()
		return semtypes.DecimalConst(*result)
	default:
		return exprTy
	}
}

func additiveSingletonType(t typeResolver, lhsTy, rhsTy semtypes.SemType, op model.OperatorKind, loc diagnostics.Location) (semtypes.SemType, bool) {
	bothSameType := func(ty semtypes.SemType) bool {
		return semtypes.IsSubtype(t.typeContext(), lhsTy, ty) && semtypes.IsSubtype(t.typeContext(), rhsTy, ty)
	}
	switch {
	case bothSameType(semtypes.XML):
		if op != model.OperatorKind_ADD {
			t.semanticError(fmt.Sprintf("unsupported operation %s for xml (only addition is supported)", string(op)), loc)
			return semtypes.SemType{}, false
		}
		resultTy := semtypes.XMLSequence(semtypes.Union(lhsTy, rhsTy))
		return resultTy, true
	case bothSameType(semtypes.String):
		if op != model.OperatorKind_ADD {
			t.semanticError(fmt.Sprintf("unsupported operation %s for string (only addition is supported)", string(op)), loc)
			return semtypes.SemType{}, false
		}
		lhsValue := semtypes.SingleShape(lhsTy)
		rhsValue := semtypes.SingleShape(rhsTy)
		if lhsValue.IsPresent() && rhsValue.IsPresent() {
			resultValue := lhsValue.Get().Value.(string) + rhsValue.Get().Value.(string)
			return semtypes.StringConst(resultValue), true
		}
		return semtypes.SemType{}, true
	case bothSameType(semtypes.Int):
		lhsValue := semtypes.SingleShape(lhsTy)
		rhsValue := semtypes.SingleShape(rhsTy)
		if lhsValue.IsPresent() && rhsValue.IsPresent() {
			var resultValue int64
			switch op {
			case model.OperatorKind_ADD:
				resultValue = lhsValue.Get().Value.(int64) + rhsValue.Get().Value.(int64)
			case model.OperatorKind_SUB:
				resultValue = lhsValue.Get().Value.(int64) - rhsValue.Get().Value.(int64)
			default:
				t.internalError(fmt.Sprintf("unexpect additive operand %s", string(op)), loc)
			}
			return semtypes.IntConst(resultValue), true
		}
		return semtypes.SemType{}, true
	case bothSameType(semtypes.Float):
		lhsValue := semtypes.SingleShape(lhsTy)
		rhsValue := semtypes.SingleShape(rhsTy)
		if lhsValue.IsPresent() && rhsValue.IsPresent() {
			var resultValue float64
			switch op {
			case model.OperatorKind_ADD:
				resultValue = lhsValue.Get().Value.(float64) + rhsValue.Get().Value.(float64)
			case model.OperatorKind_SUB:
				resultValue = lhsValue.Get().Value.(float64) - rhsValue.Get().Value.(float64)
			default:
				t.internalError(fmt.Sprintf("unexpect additive operand %s", string(op)), loc)
			}
			return semtypes.FloatConst(resultValue), true
		}
		return semtypes.SemType{}, true
	case bothSameType(semtypes.Decimal):
		lhsValue := semtypes.SingleShape(lhsTy)
		rhsValue := semtypes.SingleShape(rhsTy)
		if lhsValue.IsPresent() && rhsValue.IsPresent() {
			lhsDec := lhsValue.Get().Value.(*decimal.Decimal)
			rhsDec := rhsValue.Get().Value.(*decimal.Decimal)
			var result *decimal.Decimal
			var err *decimal.Error
			switch op {
			case model.OperatorKind_ADD:
				result, err = lhsDec.Add(rhsDec)
			case model.OperatorKind_SUB:
				result, err = lhsDec.Sub(rhsDec)
			default:
				t.internalError(fmt.Sprintf("unexpect additive operand %s", string(op)), loc)
			}
			if err != nil {
				return semtypes.SemType{}, true
			}
			return semtypes.DecimalConst(*result), true
		}
		return semtypes.SemType{}, true
	default:
		return semtypes.SemType{}, true
	}
}

func resolveAdditiveExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	supportedTypes := additiveSupportedTypes
	if expr.GetOperatorKind() == model.OperatorKind_SUB {
		supportedTypes = semtypes.Number
	}
	operandExpectedType := additiveOperandExpectedType(supportedTypes, expr.GetOperatorKind(), expectedType)
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
	resultTy, effect, ok := resolveAdditiveExprInner(t, sequentialChain(t, lhsEffect), lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expectedType, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func isXMLStringPair(ctx semtypes.Context, lhsTy, rhsTy semtypes.SemType) bool {
	return !semtypes.IsEmpty(ctx, lhsTy) && !semtypes.IsEmpty(ctx, rhsTy) &&
		((semtypes.IsSubtype(ctx, lhsTy, semtypes.XML) && semtypes.IsSubtype(ctx, rhsTy, semtypes.String)) ||
			(semtypes.IsSubtype(ctx, lhsTy, semtypes.String) && semtypes.IsSubtype(ctx, rhsTy, semtypes.XML)))
}

func additiveOperandExpectedType(supportedTypes semtypes.SemType, op model.OperatorKind, expectedType semtypes.SemType) semtypes.SemType {
	operandExpectedType := semtypes.Union(supportedTypes, semtypes.XML)
	if semtypes.IsZero(expectedType) {
		return operandExpectedType
	}
	operandExpectedType = semtypes.Intersect(operandExpectedType, expectedType)
	if op == model.OperatorKind_ADD && semtypes.ContainsBasicType(expectedType, semtypes.XML) {
		operandExpectedType = semtypes.Union(operandExpectedType, semtypes.String)
	}
	return operandExpectedType
}

func resolveAdditiveExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, expectedType semtypes.SemType, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	supportedTypes := additiveSupportedTypes
	if op == model.OperatorKind_SUB {
		supportedTypes = semtypes.Number
	}
	operandExpectedType := additiveOperandExpectedType(supportedTypes, op, expectedType)
	rhsResult, ok := resolveActionOrExpression(t, chain, rhs, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy := rhsResult.ty
	chain = sequentialChain(t, rhsResult.effect)
	singletonTy, ok := additiveSingletonType(t, lhsTy, rhsTy, op, pos)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if !semtypes.IsZero(singletonTy) {
		return singletonTy, defaultExpressionEffect(chain), true
	}

	lhsTy, rhsTy, nilLifted := nilLiftedUnderlyingType(lhsTy, rhsTy)

	if op == model.OperatorKind_ADD && isXMLStringPair(t.typeContext(), lhsTy, rhsTy) {
		xmlTy := semtypes.Intersect(semtypes.Union(lhsTy, rhsTy), semtypes.XML)
		resultTy := semtypes.XMLSequence(semtypes.Union(xmlTy, semtypes.XMLText))
		if nilLifted {
			resultTy = semtypes.Union(semtypes.Nil, resultTy)
		}
		return resultTy, defaultExpressionEffect(chain), true
	}

	numLhsBits := semtypes.NBasicTypes(lhsTy)
	numRhsBits := semtypes.NBasicTypes(rhsTy)

	if numLhsBits != 1 || numRhsBits != 1 {
		t.semanticError(fmt.Sprintf("union types not supported for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}

	ctx := t.typeContext()

	lhsBasicTy := semtypes.WidenToBasicTypes(lhsTy)
	rhsBasicTy := semtypes.WidenToBasicTypes(rhsTy)
	if !semtypes.IsSubtype(ctx, lhsBasicTy, supportedTypes) || !semtypes.IsSubtype(ctx, rhsBasicTy, supportedTypes) {
		msg := "expect numeric, string, or xml types"
		if op == model.OperatorKind_SUB {
			msg = "expect numeric types"
		}
		t.semanticError(fmt.Sprintf("%s for %s", msg, string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	} else if !semtypes.IsSameType(t.typeContext(), lhsBasicTy, rhsBasicTy) {
		t.semanticError("both operands must belong to same basic type", pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := lhsBasicTy
	if nilLifted {
		resultTy = semtypes.Union(semtypes.Nil, resultTy)
	}
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveRangeExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.Int)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsResult, ok := resolveActionOrExpression(t, sequentialChain(t, lhsResult.effect), expr.RhsExpr, semtypes.Int)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := createIteratorType(t.typeEnv(), semtypes.Int, semtypes.Nil)
	setNodeType(t, expr, resultTy)
	effect := defaultExpressionEffect(sequentialChain(t, rhsResult.effect))
	return resultTy, effect, true
}

func resolveShiftExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.Int)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
	resultTy, effect, ok := resolveShiftExprInner(t, sequentialChain(t, lhsEffect), lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveShiftExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsResult, ok := resolveActionOrExpression(t, chain, rhs, semtypes.Int)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy := rhsResult.ty
	chain = sequentialChain(t, rhsResult.effect)
	lhsTy, rhsTy, nilLifted := nilLiftedUnderlyingType(lhsTy, rhsTy)
	ctx := t.typeContext()
	// TODO: handle singleton typing here

	if semtypes.IsEmpty(ctx, lhsTy) || semtypes.IsEmpty(ctx, rhsTy) || !semtypes.IsSubtype(ctx, lhsTy, semtypes.Int) || !semtypes.IsSubtype(ctx, rhsTy, semtypes.Int) {
		t.semanticError(fmt.Sprintf("expect integer types for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Int
	switch op {
	case model.OperatorKind_BITWISE_RIGHT_SHIFT, model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		for _, ty := range bitWiseOpLookOrder {
			if semtypes.IsSubtype(ctx, lhsTy, ty) {
				resultTy = ty
				break
			}
		}
	case model.OperatorKind_BITWISE_LEFT_SHIFT:
		// The default int result type is correct for left shift.
	default:
		t.internalError(fmt.Sprintf("unexpected shift operator %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if nilLifted {
		resultTy = semtypes.Union(resultTy, semtypes.Nil)
	}
	return resultTy, defaultExpressionEffect(chain), true
}

func nilLiftedUnderlyingType(lhsTy, rhsTy semtypes.SemType) (semtypes.SemType, semtypes.SemType, bool) {
	nilLifted := false
	if semtypes.ContainsBasicType(lhsTy, semtypes.Nil) {
		nilLifted = true
		lhsTy = semtypes.Diff(lhsTy, semtypes.Nil)
	}
	if semtypes.ContainsBasicType(rhsTy, semtypes.Nil) {
		nilLifted = true
		rhsTy = semtypes.Diff(rhsTy, semtypes.Nil)
	}
	return lhsTy, rhsTy, nilLifted
}

func resolveRelationalExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
	rhsResult, ok := resolveActionOrExpression(t, sequentialChain(t, lhsEffect), expr.RhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy := rhsResult.ty

	if !semtypes.Comparable(t.typeContext(), lhsTy, rhsTy) {
		t.semanticError("values are not comparable", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Boolean
	setNodeType(t, expr, resultTy)
	effect := defaultExpressionEffect(sequentialChain(t, rhsResult.effect))
	return resultTy, effect, true
}

func resolveMultiplicativeExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	operandExpectedType := semtypes.Number
	if !semtypes.IsZero(expectedType) {
		operandExpectedType = semtypes.Intersect(expectedType, operandExpectedType)
	}
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
	resultTy, effect, ok := resolveMultiplicativeExprInner(t, sequentialChain(t, lhsEffect), lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expectedType, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveMultiplicativeExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, expectedType semtypes.SemType, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	operandExpectedType := semtypes.Number
	if !semtypes.IsZero(expectedType) {
		operandExpectedType = semtypes.Intersect(expectedType, operandExpectedType)
	}
	rhsResult, ok := resolveActionOrExpression(t, chain, rhs, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy := rhsResult.ty
	chain = sequentialChain(t, rhsResult.effect)
	// TODO: handle singleton

	lhsTy, rhsTy, nilLifted := nilLiftedUnderlyingType(lhsTy, rhsTy)

	numLhsBits := semtypes.NBasicTypes(lhsTy)
	numRhsBits := semtypes.NBasicTypes(rhsTy)

	if numLhsBits != 1 || numRhsBits != 1 {
		t.semanticError(fmt.Sprintf("union types not supported for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}

	lhsBasicTy := semtypes.WidenToBasicTypes(lhsTy)
	rhsBasicTy := semtypes.WidenToBasicTypes(rhsTy)
	if !common.IsNumericType(t.typeContext(), lhsBasicTy) || !common.IsNumericType(t.typeContext(), rhsBasicTy) {
		t.semanticError(fmt.Sprintf("expect numeric types for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	var resultTy semtypes.SemType
	if !semtypes.IsSameType(t.typeContext(), lhsBasicTy, rhsBasicTy) {
		ctx := t.typeContext()
		if semtypes.IsSubtype(ctx, rhsBasicTy, semtypes.Int) {
			resultTy = lhsBasicTy
		} else if op == model.OperatorKind_MUL && semtypes.IsSubtype(ctx, lhsBasicTy, semtypes.Int) {
			resultTy = rhsBasicTy
		} else {
			t.semanticError("both operands must belong to same basic type", pos)
			return semtypes.SemType{}, expressionEffect{}, false
		}
	} else {
		resultTy = lhsBasicTy
	}
	if nilLifted {
		resultTy = semtypes.Union(semtypes.Nil, resultTy)
	}
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveBitWiseExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.Int)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
	resultTy, effect, ok := resolveBitWiseExprInner(t, sequentialChain(t, lhsEffect), lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveBitWiseExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsResult, ok := resolveActionOrExpression(t, chain, rhs, semtypes.Int)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy := rhsResult.ty
	chain = sequentialChain(t, rhsResult.effect)
	// TODO: handle singleton

	lhsTy, rhsTy, nilLifted := nilLiftedUnderlyingType(lhsTy, rhsTy)

	numLhsBits := semtypes.NBasicTypes(lhsTy)
	numRhsBits := semtypes.NBasicTypes(rhsTy)

	if numLhsBits != 1 || numRhsBits != 1 {
		t.semanticError(fmt.Sprintf("union types not supported for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}

	ctx := t.typeContext()
	if !semtypes.IsSubtype(ctx, lhsTy, semtypes.Int) || !semtypes.IsSubtype(ctx, rhsTy, semtypes.Int) {
		t.semanticError("expect integer types for bitwise operators", pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}

	resultTy := semtypes.Int
	switch op {
	case model.OperatorKind_BITWISE_AND:
		for _, ty := range bitWiseOpLookOrder {
			if semtypes.IsSubtype(ctx, lhsTy, ty) || semtypes.IsSubtype(ctx, rhsTy, ty) {
				resultTy = ty
				break
			}
		}
	case model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		for _, ty := range bitWiseOpLookOrder {
			if semtypes.IsSubtype(ctx, lhsTy, ty) && semtypes.IsSubtype(ctx, rhsTy, ty) {
				resultTy = ty
				break
			}
		}
	default:
		t.unimplemented(fmt.Sprintf("unsupported bitwise operator: %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if nilLifted {
		resultTy = semtypes.Union(resultTy, semtypes.Nil)
	}
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveTernaryExpr(t typeResolver, chain *binding, expr *ast.BLangTernaryExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	conditionResult, ok := resolveActionOrExpression(t, chain, expr.Condition, semtypes.Boolean)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	thenExpected := expectedType
	elseExpected := expectedType
	if isSingletonBool(conditionResult.ty, true) {
		elseExpected = semtypes.SemType{}
	} else if isSingletonBool(conditionResult.ty, false) {
		thenExpected = semtypes.SemType{}
	}

	thenResult, ok := resolveActionOrExpression(t, conditionResult.effect.ifTrue, expr.ThenExpr, thenExpected)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	elseResult, ok := resolveActionOrExpression(t, conditionResult.effect.ifFalse, expr.ElseExpr, elseExpected)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	resultTy := semtypes.Union(thenResult.ty, elseResult.ty)
	if isSingletonBool(conditionResult.ty, true) {
		resultTy = thenResult.ty
	} else if isSingletonBool(conditionResult.ty, false) {
		resultTy = elseResult.ty
	}

	var effect expressionEffect
	if isSingletonBool(conditionResult.ty, true) {
		effect = thenResult.effect
	} else if isSingletonBool(conditionResult.ty, false) {
		effect = elseResult.effect
	} else {
		effect = expressionEffect{
			ifTrue:  mergeChains(t, thenResult.effect.ifTrue, elseResult.effect.ifTrue, semtypes.Union),
			ifFalse: mergeChains(t, thenResult.effect.ifFalse, elseResult.effect.ifFalse, semtypes.Union),
		}
	}
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveNilConditionalExpr(t typeResolver, chain *binding, expr *ast.BLangNilConditionalExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	// The LHS is always evaluated, so the RHS is checked on its outgoing chain.
	rhsResult, ok := resolveActionOrExpression(t, sequentialChain(t, lhsResult.effect), expr.RhsExpr, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	ctx := t.typeContext()
	var resultTy semtypes.SemType
	var effect expressionEffect
	switch {
	case semtypes.IsSameType(ctx, lhsResult.ty, semtypes.Nil):
		resultTy = rhsResult.ty
		effect = rhsResult.effect
	case semtypes.IsEmpty(ctx, semtypes.Intersect(lhsResult.ty, semtypes.Nil)):
		resultTy = lhsResult.ty
		effect = lhsResult.effect
	default:
		resultTy = semtypes.Union(semtypes.Diff(lhsResult.ty, semtypes.Nil), rhsResult.ty)
		nilBranchEffect := rhsResult.effect
		nonNilBranchEffect := lhsResult.effect
		effect = expressionEffect{
			ifTrue:  mergeChains(t, nilBranchEffect.ifTrue, nonNilBranchEffect.ifTrue, semtypes.Union),
			ifFalse: mergeChains(t, nilBranchEffect.ifFalse, nonNilBranchEffect.ifFalse, semtypes.Union),
		}
	}

	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveBinaryExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_ADD, model.OperatorKind_SUB:
		return resolveAdditiveExpr(t, chain, expr, expectedType)
	case model.OperatorKind_MUL, model.OperatorKind_DIV, model.OperatorKind_MOD:
		return resolveMultiplicativeExpr(t, chain, expr, expectedType)
	case model.OperatorKind_AND, model.OperatorKind_OR:
		return resolveLogicalExpr(t, chain, expr)
	case model.OperatorKind_EQUAL, model.OperatorKind_EQUALS, model.OperatorKind_NOT_EQUAL, model.OperatorKind_REF_EQUAL, model.OperatorKind_REF_NOT_EQUAL:
		return resolveEqualityExpr(t, chain, expr)
	case model.OperatorKind_CLOSED_RANGE, model.OperatorKind_HALF_OPEN_RANGE:
		return resolveRangeExpr(t, chain, expr)
	case model.OperatorKind_BITWISE_LEFT_SHIFT, model.OperatorKind_BITWISE_RIGHT_SHIFT, model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return resolveShiftExpr(t, chain, expr)
	case model.OperatorKind_LESS_THAN, model.OperatorKind_LESS_EQUAL, model.OperatorKind_GREATER_THAN, model.OperatorKind_GREATER_EQUAL:
		return resolveRelationalExpr(t, chain, expr)
	case model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		return resolveBitWiseExpr(t, chain, expr)
	default:
		t.internalError(fmt.Sprintf("Unexpected binary expr %s", expr.OpKind), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
}

func isSingletonBool(ty semtypes.SemType, value bool) bool {
	singleShape := semtypes.SingleShape(ty)
	if singleShape.IsPresent() {
		return singleShape.Get().Value == value
	} else {
		return false
	}
}

func resolveEqualityExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsResult, ok := resolveActionOrExpression(t, sequentialChain(t, lhsResult.effect), expr.RhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	// The narrowing an equality test produces applies after both operands have been
	// evaluated, so it is built on their outgoing chain.
	chain = sequentialChain(t, rhsResult.effect)
	var effect expressionEffect
	if expr.OpKind == model.OperatorKind_EQUAL || expr.OpKind == model.OperatorKind_NOT_EQUAL {
		effect = equalityNarrowingEffect(t, chain, expr, lhsResult.ty, rhsResult.ty)
	} else {
		effect = defaultExpressionEffect(chain)
	}
	resultTy := semtypes.Boolean
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveLogicalExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	switch expr.OpKind {
	case model.OperatorKind_AND:
		return resolveAndExpr(t, chain, expr)
	case model.OperatorKind_OR:
		return resolveOrExpr(t, chain, expr)
	default:
		t.internalError(fmt.Sprintf("Unexpected logical expression op %s", string(expr.OpKind)), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
}

func resolveAndExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
	resultTy, effect, ok := resolveAndExprInner(t, chain, lhsTy, lhsEffect, expr.RhsExpr, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveAndExprInner(t typeResolver, _ *binding, lhsTy semtypes.SemType, lhsEffect expressionEffect, rhs ast.BLangActionOrExpression, _ diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsResult, ok := resolveActionOrExpression(t, lhsEffect.ifTrue, rhs, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy, rhsEffect := rhsResult.ty, rhsResult.effect

	resultTy := semtypes.Boolean
	if isSingletonBool(lhsTy, false) || isSingletonBool(rhsTy, false) {
		resultTy = semtypes.BooleanConst(false)
	} else if isSingletonBool(lhsTy, true) && isSingletonBool(rhsTy, true) {
		resultTy = semtypes.BooleanConst(true)
	} else if isSingletonBool(lhsTy, true) {
		resultTy = rhsTy
	}

	// The RHS was checked on lhsEffect.ifTrue, so both RHS outcomes still have it as
	// an ancestor and the merges compose the full chains directly. A capture the RHS
	// makes therefore also reaches the false outcome, where the RHS did not run; that
	// over-approximation is the same conservative direction as the loop policy.
	ifTrue := mergeChains(t, lhsEffect.ifTrue, rhsEffect.ifTrue, semtypes.Intersect)
	ifFalse := mergeChains(t, lhsEffect.ifFalse, mergeChains(t, lhsEffect.ifTrue, rhsEffect.ifFalse, semtypes.Intersect), semtypes.Union)
	return resultTy, expressionEffect{ifTrue: ifTrue, ifFalse: ifFalse}, true
}

func resolveOrExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsResult, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, lhsEffect := lhsResult.ty, lhsResult.effect
	resultTy, effect, ok := resolveOrExprInner(t, chain, lhsTy, lhsEffect, expr.RhsExpr, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, expr, resultTy)
	return resultTy, effect, true
}

func resolveOrExprInner(t typeResolver, _ *binding, lhsTy semtypes.SemType, lhsEffect expressionEffect, rhs ast.BLangActionOrExpression, _ diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsResult, ok := resolveActionOrExpression(t, lhsEffect.ifFalse, rhs, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy, rhsEffect := rhsResult.ty, rhsResult.effect

	resultTy := semtypes.Boolean
	if isSingletonBool(lhsTy, true) || isSingletonBool(rhsTy, true) {
		resultTy = semtypes.BooleanConst(true)
	} else if isSingletonBool(lhsTy, false) && isSingletonBool(rhsTy, false) {
		resultTy = semtypes.BooleanConst(false)
	} else if isSingletonBool(lhsTy, false) {
		resultTy = rhsTy
	}

	// The RHS was checked on lhsEffect.ifFalse, so both RHS outcomes still have it as
	// an ancestor and the merges compose the full chains directly.
	ifTrue := mergeChains(t, lhsEffect.ifTrue, mergeChains(t, lhsEffect.ifFalse, rhsEffect.ifTrue, semtypes.Intersect), semtypes.Union)
	ifFalse := mergeChains(t, lhsEffect.ifFalse, rhsEffect.ifFalse, semtypes.Intersect)
	return resultTy, expressionEffect{ifTrue: ifTrue, ifFalse: ifFalse}, true
}

func equalityNarrowingEffect(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr, lhsTy, rhsTy semtypes.SemType) expressionEffect {
	lhsRef, lhsIsVarRef := varRefExp(t, chain, expr.LhsExpr)
	rhsIsSingleton := semtypes.SingleShape(rhsTy).IsPresent()
	if lhsIsVarRef && rhsIsSingleton {
		effect := buildEqualityNarrowing(t, chain, lhsRef, rhsTy)
		if expr.OpKind == model.OperatorKind_NOT_EQUAL {
			return expressionEffect{ifTrue: effect.ifFalse, ifFalse: effect.ifTrue}
		}
		return effect
	}
	rhsRef, rhsIsVarRef := varRefExp(t, chain, expr.RhsExpr)
	lhsIsSingleton := semtypes.SingleShape(lhsTy).IsPresent()
	if rhsIsVarRef && lhsIsSingleton {
		effect := buildEqualityNarrowing(t, chain, rhsRef, lhsTy)
		if expr.OpKind == model.OperatorKind_NOT_EQUAL {
			return expressionEffect{ifTrue: effect.ifFalse, ifFalse: effect.ifTrue}
		}
		return effect
	}
	return defaultExpressionEffect(chain)
}

func buildEqualityNarrowing(t typeResolver, chain *binding, ref model.SymbolRef, singletonTy semtypes.SemType) expressionEffect {
	if !canNarrow(t, chain, ref) {
		return defaultExpressionEffect(chain)
	}
	return narrowByType(t, chain, ref, singletonTy)
}

var additiveSupportedTypes = semtypes.Union(semtypes.Union(semtypes.Number, semtypes.String), semtypes.XML)

var bitWiseOpLookOrder = []semtypes.SemType{semtypes.UnsignedInt8, semtypes.UnsignedInt16, semtypes.UnsignedInt32}

func createIteratorType(env semtypes.Env, valueTy, completionTy semtypes.SemType) semtypes.SemType {
	od := semtypes.NewObjectDefinition()

	fields := []semtypes.Field{
		semtypes.FieldFrom("value", valueTy, false, false),
	}
	rest := semtypes.Never
	recordTy := createClosedRecordType(env, fields, rest)

	resultTy := semtypes.Union(recordTy, completionTy)

	ld := semtypes.NewListDefinition()
	listTy := ld.Define(env, nil, semtypes.ListMutability(semtypes.CellMutabilityNone))
	fd := semtypes.NewFunctionDefinition()
	fnTy := fd.Define(env, listTy, resultTy, semtypes.FunctionQualifiersFrom(env, false, false))

	members := []semtypes.Member{
		{
			Name:       "next",
			ValueType:  fnTy,
			Kind:       semtypes.MemberKindMethod,
			Visibility: semtypes.VisibilityPublic,
			Immutable:  true,
		},
	}
	return od.Define(env, semtypes.ObjectQualifiersDefault, members)
}

func createClosedRecordType(env semtypes.Env, fields []semtypes.Field, rest semtypes.SemType) semtypes.SemType {
	md := semtypes.NewMappingDefinition()
	return md.Define(env, fields, rest)
}

func resolveIndexBasedAccess(t typeResolver, chain *binding, expr *ast.BLangIndexBasedAccess) (semtypes.SemType, expressionEffect, bool) {
	containerExpr := expr.Expr
	containerExprResult, ok := resolveActionOrExpression(t, chain, containerExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	containerExprTy := containerExprResult.ty
	chain = sequentialChain(t, containerExprResult.effect)

	keyExpr := expr.IndexExpr
	keyExprResult, ok := resolveActionOrExpression(t, chain, keyExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	keyExprTy := keyExprResult.ty
	chain = sequentialChain(t, keyExprResult.effect)

	var resultTy semtypes.SemType
	tyCtx := t.typeContext()

	if semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.List) {
		resultTy = semtypes.ListMemberTypeInnerVal(t.typeContext(), containerExprTy, keyExprTy)
	} else if semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.Union(semtypes.Mapping, semtypes.Nil)) {
		containerNilable := !semtypes.IsSubtype(t.typeContext(), containerExprTy, semtypes.Mapping)
		mappingTy := containerExprTy
		if containerNilable {
			mappingTy = semtypes.Diff(containerExprTy, semtypes.Nil)
		}
		memberTy := semtypes.MappingMemberTypeInner(t.typeContext(), mappingTy, keyExprTy)
		maybeMissing := semtypes.ContainsUndef(memberTy) || containerNilable
		if maybeMissing {
			memberTy = semtypes.Union(semtypes.Diff(memberTy, semtypes.Undef), semtypes.Nil)
		}
		if expr.IsLexpr() {
			singletonStringKey := func(keyTy semtypes.SemType) (string, bool) {
				shape := semtypes.SingleShape(keyTy)
				if shape.IsEmpty() {
					return "", false
				}
				key, ok := shape.Get().Value.(string)
				return key, ok
			}
			if key, ok := singletonStringKey(keyExprTy); ok && mappingFieldIsReadonly(tyCtx, mappingTy, key) {
				reportReadonlyMutation(t, tyCtx, mappingTy, key, expr.GetPosition())
				return semtypes.SemType{}, expressionEffect{}, false
			}
		}
		resultTy = memberTy
	} else if semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.String) {
		if expr.IsLexpr() {
			t.semanticError(fmt.Sprintf("invalid operation: type '%s' does not support member access for assignment", semtypes.ToString(tyCtx, containerExprTy)), expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		resultTy = semtypes.Char
	} else {
		t.semanticError("unsupported container type for index based access", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	setNodeType(t, expr, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func isLax(t typeResolver, ty semtypes.SemType) bool {
	return isLaxInner(t, ty, semtypes.NewSemtypeInterner(), make(map[semtypes.InternHandle]bool))
}

func isLaxInner(t typeResolver, ty semtypes.SemType, interner *semtypes.SemTypeInterner, seen map[semtypes.InternHandle]bool) bool {
	tyCtx := t.typeContext()
	if semtypes.IsSameType(tyCtx, ty, semtypes.CreateJSON(tyCtx)) {
		return true
	}
	if !semtypes.IsSubtype(tyCtx, ty, semtypes.Mapping) {
		return false
	}
	handle := interner.Intern(ty)
	if seen[handle] {
		return false
	}
	seen[handle] = true
	defer delete(seen, handle)
	return semtypes.AllMapConstraintTypesMatch(tyCtx, ty, func(memberTy semtypes.SemType) bool {
		return isLaxInner(t, memberTy, interner, seen)
	})
}

func effectiveLaxType(t typeResolver, container expressionResult) (semtypes.SemType, bool) {
	if container.lax || isLax(t, container.ty) {
		return container.ty, true
	}
	return semtypes.SemType{}, false
}

func laxFieldMemberType(t typeResolver, expr *ast.BLangFieldBaseAccess, container expressionResult) (rawMemberTy semtypes.SemType, memberTy semtypes.SemType, ok bool) {
	containerTy, ok := effectiveLaxType(t, container)
	if !ok {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	mappingTy := semtypes.Intersect(containerTy, semtypes.Mapping)
	rawMemberTy = semtypes.MappingMemberTypeInner(t.typeContext(), mappingTy, semtypes.StringConst(expr.Field.GetValue()))
	memberTy = semtypes.Diff(rawMemberTy, semtypes.Undef)
	return rawMemberTy, memberTy, true
}

func resolveFieldBaseAccess(t typeResolver, chain *binding, expr *ast.BLangFieldBaseAccess) (expressionResult, bool) {
	containerExprResult, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return expressionResult{}, false
	}
	containerExprTy := containerExprResult.ty
	chain = sequentialChain(t, containerExprResult.effect)
	key := expr.Field.GetValue()
	if expr.IsOptionalAccess() {
		return resolveOptionalFieldBaseAccess(t, chain, expr, containerExprResult, key)
	}
	tyCtx := t.typeContext()
	if rawMemberTy, memberTy, lax := laxFieldMemberType(t, expr, containerExprResult); lax {
		if expr.IsLexpr() {
			t.semanticError("lax field access cannot be used as an lvalue", expr.GetPosition())
			return expressionResult{}, false
		}
		if !t.ensureNotEmpty(memberTy, func() {
			t.semanticError("lax field access has an empty member type", expr.GetPosition())
		}) {
			return expressionResult{}, false
		}
		resultTy := memberTy
		if semtypes.ContainsUndef(rawMemberTy) {
			resultTy = semtypes.Union(resultTy, semtypes.Error)
		}
		setNodeType(t, expr, resultTy)
		setLaxAccess(t, expr)
		return expressionResult{ty: resultTy, effect: defaultExpressionEffect(chain), lax: true}, true
	}

	var memberTy semtypes.SemType
	switch {
	case semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.Object):
		memberTy = semtypes.ObjectMemberType(tyCtx, semtypes.StringConst(key), containerExprTy)
		if semtypes.IsZero(memberTy) {
			t.semanticError("field not found in object type", expr.GetPosition())
			return expressionResult{}, false
		}
	case semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.Union(semtypes.Mapping, semtypes.Nil)):
		containerNilable := !semtypes.IsSubtype(t.typeContext(), containerExprTy, semtypes.Mapping)
		mappingTy := containerExprTy
		if containerNilable {
			mappingTy = semtypes.Diff(containerExprTy, semtypes.Nil)
		}
		var ok bool
		memberTy, ok = fieldBaseAccessMappingType(tyCtx, mappingTy, key, expr.IsLexpr())
		if !ok {
			t.semanticError("field base access is only possible for declared fields", expr.GetPosition())
			return expressionResult{}, false
		}
		if expr.IsCompoundAssignmentLValue() {
			readTy, readOk := fieldBaseAccessMappingType(tyCtx, mappingTy, key, false)
			writeTy, writeOk := fieldBaseAccessMappingType(tyCtx, mappingTy, key, true)
			if readOk && writeOk && !semtypes.IsSubtype(tyCtx, readTy, writeTy) {
				t.semanticError(fmt.Sprintf("incompatible type: expected %s, got %s", semtypes.ToString(tyCtx, writeTy), semtypes.ToString(tyCtx, readTy)), expr.GetPosition())
				return expressionResult{}, false
			}
		}
		if expr.IsLexpr() && mappingFieldIsReadonly(tyCtx, mappingTy, key) {
			reportReadonlyMutation(t, tyCtx, mappingTy, key, expr.GetPosition())
			return expressionResult{}, false
		}
		if containerNilable {
			memberTy = semtypes.Union(memberTy, semtypes.Nil)
		}
	default:
		t.semanticError("unsupported container type for field access", expr.GetPosition())
		return expressionResult{}, false
	}

	setNodeType(t, expr, memberTy)
	return expressionResult{ty: memberTy, effect: defaultExpressionEffect(chain)}, true
}

func resolveOptionalFieldBaseAccess(t typeResolver, chain *binding, expr *ast.BLangFieldBaseAccess, container expressionResult, fieldname string) (expressionResult, bool) {
	if expr.IsLexpr() {
		t.semanticError("optional field access cannot be used as an lvalue", expr.GetPosition())
		return expressionResult{}, false
	}

	tyCtx := t.typeContext()
	T := container.ty
	laxTy, lax := effectiveLaxType(t, container)
	if lax {
		T = laxTy
	}
	switch {
	case semtypes.IsSubtype(tyCtx, T, semtypes.XML):
		t.unimplemented("XML optional attribute access not supported", expr.GetPosition()) // https://github.com/ballerina-nutcracker/ballerina/issues/560
		return expressionResult{}, false
	case lax || semtypes.IsSubtype(tyCtx, T, semtypes.Union(semtypes.Mapping, semtypes.Nil)):
		Tbar := semtypes.Intersect(T, semtypes.Mapping)
		K := semtypes.StringConst(fieldname)
		M := semtypes.MappingMemberTypeInner(tyCtx, Tbar, K)
		MBar := semtypes.Diff(M, semtypes.Undef)
		if lax {
			if !t.ensureNotEmpty(MBar, func() {
				t.semanticError("lax optional field access has an empty member type", expr.GetPosition())
			}) {
				return expressionResult{}, false
			}
		} else if !semtypes.AnyMappingAtomHasFieldByName(tyCtx, Tbar, fieldname) {
			t.semanticError(fmt.Sprintf("%s is not an individual field descriptor in %s", fieldname, semtypes.ToString(tyCtx, T)), expr.GetPosition())
			return expressionResult{}, false
		}
		N := semtypes.Never
		if semtypes.IsSubtype(tyCtx, semtypes.Nil, T) || semtypes.ContainsUndef(M) {
			N = semtypes.Nil
		}
		E := semtypes.Never
		if lax {
			if !semtypes.IsSubtype(tyCtx, T, semtypes.Union(Tbar, semtypes.Nil)) {
				E = semtypes.Error
			}
			setLaxAccess(t, expr)
		}
		resultTy := semtypes.Union(semtypes.Union(MBar, N), E)
		setNodeType(t, expr, resultTy)
		return expressionResult{ty: resultTy, effect: defaultExpressionEffect(chain), lax: lax}, true
	default:
		t.semanticError("optional field access must be subtype of xml|map|()", expr.GetPosition())
		return expressionResult{}, false
	}
}

// mappingFieldIsReadonly reports whether a store to key can never succeed, that is whether the
// cell selected by key is CellMutabilityNone in every mapping alternative. A key whose cell
// cannot hold a value is not a readonly field, it is a key the mapping does not have, so it
// keeps whatever diagnostic it has otherwise.
//
// MappingAlternative negatives are not consulted. MappingAlternatives emits an alternative
// whenever the positive atoms intersect, without testing whether the negatives leave it
// inhabited, so the alternative set may be an over approximation. That is the safe direction
// here: a spurious alternative can only make "every alternative is CellMutabilityNone" harder
// to satisfy, so ignoring negatives can only miss an error, never invent one.
func mappingFieldIsReadonly(tyCtx semtypes.Context, mappingTy semtypes.SemType, key string) bool {
	alts := semtypes.MappingAlternatives(tyCtx, mappingTy)
	if len(alts) == 0 {
		return false
	}
	for _, alt := range alts {
		atomic := alt.Atomic()
		if atomic == nil {
			return false
		}
		cell := atomic.FieldCell(key)
		if semtypes.IsNever(semtypes.CellInnerVal(cell)) {
			return false
		}
		if semtypes.CellMut(cell) != semtypes.CellMutabilityNone {
			return false
		}
	}
	return true
}

func reportReadonlyMutation(t typeResolver, tyCtx semtypes.Context, containerTy semtypes.SemType, key string, loc diagnostics.Location) {
	if semtypes.IsSubtype(tyCtx, containerTy, semtypes.ValReadonly) {
		t.semanticError(fmt.Sprintf("cannot mutate readonly value of type '%s'", semtypes.ToString(tyCtx, containerTy)), loc)
		return
	}
	t.semanticError(fmt.Sprintf("cannot mutate readonly field '%s'", key), loc)
}

func fieldBaseAccessMappingType(tyCtx semtypes.Context, containerExprTy semtypes.SemType, key string, isLexpr bool) (semtypes.SemType, bool) {
	keyTy := semtypes.StringConst(key)
	memberTy := semtypes.MappingMemberTypeInner(tyCtx, containerExprTy, keyTy)
	if !semtypes.ContainsUndef(memberTy) {
		return memberTy, true
	}
	// I think the correct thing to check is if any has an "optional" field by the name but spec if very specific in
	// same any declared feild (without optional qualifier)
	if !isLexpr && semtypes.AnyMappingAtomHasFieldByName(tyCtx, containerExprTy, key) {
		return semtypes.Union(semtypes.Diff(memberTy, semtypes.Undef), semtypes.Nil), true
	}
	if isLexpr && semtypes.AllMappingAtomHasFieldByName(tyCtx, containerExprTy, key) {
		result := semtypes.Diff(memberTy, semtypes.Undef)
		if semtypes.AllMappingAtomsHaveOptionalFieldByName(tyCtx, containerExprTy, key) {
			result = semtypes.Union(result, semtypes.Nil)
		}
		return result, true
	}
	return semtypes.SemType{}, false
}

func resolveInvocation(t typeResolver, chain *binding, expr *ast.BLangInvocation, expectedType semtypes.SemType) (expressionResult, bool) {
	symbol := expr.RawSymbol
	if symbol == nil {
		// An unknown callee has no symbol.
		return expressionResult{}, false
	}
	switch s := symbol.(type) {
	case *common.DeferredMethodSymbol:
		return resolveMethodCall(t, chain, expr, s, expectedType)
	case *model.SymbolRef:
		return resolveFunctionCall(t, chain, expr, expr.ArgExprs, *s, expectedType)
	default:
		t.internalError(fmt.Sprintf("expected *model.SymbolRef, got %T", symbol), expr.GetPosition())
		return expressionResult{}, false
	}
}

func resolveMethodCall(t typeResolver, chain *binding, expr *ast.BLangInvocation, methodSymbol *common.DeferredMethodSymbol, expectedType semtypes.SemType) (expressionResult, bool) {
	recieverResult, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return expressionResult{}, false
	}
	recieverTy := recieverResult.ty
	// The receiver has been evaluated; its effects belong to this call's chain.
	chain = sequentialChain(t, recieverResult.effect)
	if semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.Object) {
		return resolveObjectMethodCall(t, chain, expr, recieverTy, methodSymbol, expectedType)
	}
	if semtypes.IsSubtypeSimple(recieverTy, semtypes.Stream) {
		return resolveStreamOperation(t, chain, expr, recieverTy, methodSymbol)
	}
	var symbolRef model.SymbolRef
	var pkgAlias ast.BLangIdentifier
	switch {
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.List):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.array", methodSymbol.MethodName(), expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.Int):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.int", methodSymbol.MethodName(), expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.Decimal):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.decimal", methodSymbol.MethodName(), expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.Float):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.float", methodSymbol.MethodName(), expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.Mapping):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.map", methodSymbol.MethodName(), expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.Error):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.error", methodSymbol.MethodName(), expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.String):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.string", methodSymbol.MethodName(), expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.XML):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.xml", methodSymbol.MethodName(), expr)
	default:
		ok = false
	}
	// lang.value applies to any value, so it's the fallback module.
	if !ok {
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.value", methodSymbol.MethodName(), expr)
	}
	if !ok {
		t.semanticError("method not found: "+methodSymbol.MethodName(), expr.GetPosition())
		return expressionResult{}, false
	}
	argExprs := make([]ast.BLangExpression, len(expr.ArgExprs)+1)
	argExprs[0] = expr.Expr
	copy(argExprs[1:], expr.ArgExprs)
	call, retTy, ok := resolveCallReturnType(t, chain, expr, argExprs, symbolRef, expectedType)
	if !ok {
		return expressionResult{}, false
	}
	moveLangLibReceiver(t, expr, call.lowered, pkgAlias)
	setInvocationResolvedSymbol(t, expr, call.callee)
	setNodeType(t, expr, retTy)
	return expressionResult{ty: retTy, effect: defaultExpressionEffect(call.chain), symbol: call.callee}, true
}

func isRemoteMethod(t typeResolver, objType semtypes.SemType, methodName string) bool {
	ctx := t.typeContext()
	kindTy := semtypes.ObjectMemberKind(ctx, semtypes.StringConst(methodName), objType)
	return !semtypes.IsZero(kindTy) && semtypes.IsSubtype(ctx, kindTy, semtypes.StringConst("remote-method"))
}

func resolveObjectMethodCall(t typeResolver, chain *binding, expr *ast.BLangInvocation, recieverTy semtypes.SemType,
	methodSymbol *common.DeferredMethodSymbol, expectedType semtypes.SemType,
) (expressionResult, bool) {
	if methodRef, found := sourceObjectMethodSymbol(t, recieverTy, methodSymbol.MethodName()); found {
		return resolveFunctionCall(t, chain, expr, expr.ArgExprs, methodRef, expectedType)
	}
	symbolRef, retTy, effect, ok := finishResolveMethodCall(t, chain, recieverTy, methodSymbol.MethodName(), methodSymbol, expr.ArgExprs, expr)
	if !ok {
		return expressionResult{}, false
	}
	setNodeSymbol(t, expr, symbolRef)
	return expressionResult{ty: retTy, effect: effect, symbol: symbolRef}, true
}

func sourceObjectMethodSymbol(t typeResolver, receiverTy semtypes.SemType, methodName string) (model.SymbolRef, bool) {
	atom, ok := semtypes.ObjectPositiveAtom(t.typeContext(), receiverTy)
	if !ok {
		return model.SymbolRef{}, false
	}
	table, ok := t.objectMethodTable(atom)
	if !ok {
		return model.SymbolRef{}, false
	}
	method, found := table.Methods[methodName]
	return method, found
}

func resolveStreamOperation(t typeResolver, chain *binding, expr *ast.BLangInvocation, recieverTy semtypes.SemType, methodSymbol *common.DeferredMethodSymbol) (expressionResult, bool) {
	cx := t.typeContext()
	valueTy := semtypes.StreamValueType(cx, recieverTy)
	completionTy := semtypes.StreamCompletionType(cx, recieverTy)
	if semtypes.IsZero(valueTy) || semtypes.IsZero(completionTy) {
		t.internalError("failed to extract stream type parameters", expr.GetPosition())
		return expressionResult{}, false
	}
	var resultTy semtypes.SemType
	switch methodSymbol.MethodName() {
	case "next":
		nextRecordDefn := semtypes.NewMappingDefinition()
		nextRecord := nextRecordDefn.Define(t.typeEnv(),
			[]semtypes.Field{semtypes.FieldFrom("value", valueTy, false, false)},
			semtypes.Never)
		resultTy = semtypes.Union(nextRecord, completionTy)
	case "close":
		resultTy = semtypes.Union(completionTy, semtypes.Nil)
	default:
		t.semanticError("stream type has no operation '"+methodSymbol.MethodName()+"'", expr.GetPosition())
		return expressionResult{}, false
	}
	clearStreamOperationSymbol(t, expr)
	setNodeType(t, expr, resultTy)
	return expressionResult{ty: resultTy, effect: defaultExpressionEffect(chain)}, true
}

func finishResolveMethodCall(t typeResolver, chain *binding, receiverTy semtypes.SemType, methodName string,
	methodSymbol *common.DeferredMethodSymbol, argExprs []ast.BLangExpression, node ast.BLangActionOrExpression,
) (model.SymbolRef, semtypes.SemType, expressionEffect, bool) {
	fnTy := semtypes.ObjectMemberType(t.typeContext(), semtypes.StringConst(methodName), receiverTy)
	if semtypes.IsZero(fnTy) || !semtypes.IsSubtype(t.typeContext(), fnTy, semtypes.Function) {
		remoteMethodName := model.RemoteMethodName(methodName)
		if methodName != remoteMethodName && isRemoteMethod(t, receiverTy, remoteMethodName) {
			t.semanticError("remote methods must be invoked using '->' notation", node.GetPosition())
		} else {
			t.semanticError("method not found: "+model.StripRemotePrefix(methodName), node.GetPosition())
		}
		return model.SymbolRef{}, semtypes.SemType{}, expressionEffect{}, false
	}
	paramListTy := semtypes.FunctionParamListType(t.typeContext(), fnTy)
	if semtypes.IsZero(paramListTy) {
		t.internalError("empty function param list ty", node.GetPosition())
		return model.SymbolRef{}, semtypes.SemType{}, expressionEffect{}, false
	}
	argTys, chain, ok := resolveArgs(t, argExprs, chain, func(i int) semtypes.SemType {
		return semtypes.ListMemberTypeInnerVal(t.typeContext(), paramListTy, semtypes.IntConst(int64(i)))
	})
	if !ok {
		return model.SymbolRef{}, semtypes.SemType{}, expressionEffect{}, false
	}
	retTy := functionTypeReturnType(t, fnTy, argTys)
	if semtypes.IsZero(retTy) {
		t.semanticError("incompatible arguments for function call", node.GetPosition())
		return model.SymbolRef{}, semtypes.SemType{}, expressionEffect{}, false
	}
	if t.isEphemeral() {
		return model.SymbolRef{}, retTy, defaultExpressionEffect(chain), true
	}
	sig := model.TypedFunctionSignature{ParamTypes: argTys, ReturnType: retTy}
	symbolRef := t.createFunctionSymbol(methodSymbol.SymbolSpace(), methodName, sig, fnTy)
	signatureRef := t.allocateFunctionSignature(make([]model.Param, len(argTys)), false)
	if !t.associateFunctionSignature(symbolRef, signatureRef) {
		t.internalError("function signature already set", node.GetPosition())
		return model.SymbolRef{}, semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, node, retTy)
	return symbolRef, retTy, defaultExpressionEffect(chain), true
}

func packageResolver(t typeResolver) *packageTypeResolver {
	switch resolver := t.(type) {
	case *packageTypeResolver:
		return resolver
	case *functionTypeResolver:
		return packageResolver(resolver.parentResolver)
	case *loopTypeResolver:
		return packageResolver(resolver.parentResolver)
	default:
		return nil
	}
}

func resolveResourceMethodSignature(t typeResolver, isClient bool, isService bool, method *ast.BLangResourceMethod, depth int) bool {
	if !isClient && !isService {
		t.semanticError("resource methods are only allowed in client or service classes", method.GetPosition())
		return false
	}
	sym, ok := t.getSymbol(method.Symbol()).(model.ResourceMethodSymbol)
	if !ok {
		t.internalError("expected resource method symbol", method.GetPosition())
		return false
	}
	pathTy, pathParamRefs, ok := resolveResourcePathType(t, method, depth)
	if !ok {
		return false
	}
	sym.SetPathListType(pathTy)
	sym.SetPathParams(pathParamRefs)

	if depSym, dependent := sym.(model.DependentlyTypedFunctionSymbol); dependent {
		_, _, ok = resolveDependentlyTypedFunctionSignature(t, nil, method, depSym, depth)
		return ok
	}
	_, _, ok = resolveInvokableSignature(t, nil, method, sym, method.GetParameters(), depth)
	if !ok {
		return false
	}
	return finalizeResolvedFunctionSignature(t, method)
}

func resolveResourcePathType(t typeResolver, method *ast.BLangResourceMethod, depth int) (semtypes.SemType, []model.SymbolRef, bool) {
	anydata := semtypes.CreateAnydata(t.typeContext())
	var members []semtypes.SemType
	restMember := semtypes.Never
	var paramRefs []model.SymbolRef
	for i := range method.ResourcePath {
		seg := &method.ResourcePath[i]
		switch seg.Kind {
		case ast.ResourcePathSegmentName:
			literalTy := semtypes.StringConst(seg.Name)
			members = append(members, literalTy)
		case ast.ResourcePathSegmentParam, ast.ResourcePathSegmentParamRest:
			if seg.ParamType == nil {
				t.internalError("resource path parameter is missing type", seg.GetPosition())
				return semtypes.SemType{}, nil, false
			}
			paramTy, _, ok := resolveBType(t, nil, seg.ParamType, depth+1)
			if !ok {
				return semtypes.SemType{}, nil, false
			}
			if !semtypes.IsSubtype(t.typeContext(), paramTy, anydata) {
				// Not sure if we should allow anydata here? spec says it can be anydata but jBallerina only allow simple basic types
				t.semanticError("resource path parameter type must be a subtype of anydata", seg.GetPosition())
				return semtypes.SemType{}, nil, false
			}
			symbolTy := paramTy
			if seg.Kind == ast.ResourcePathSegmentParamRest {
				restListDefn := semtypes.NewListDefinition()
				symbolTy = restListDefn.Define(t.typeEnv(), nil, semtypes.ListRest(paramTy),
					semtypes.ListMutability(semtypes.CellMutabilityNone))
			}
			if seg.Name != "" {
				ref, ok := method.Scope().GetSymbol(seg.Name)
				if !ok {
					t.internalError("resource path parameter symbol not found in scope", seg.GetPosition())
					return semtypes.SemType{}, nil, false
				}
				t.setSymbolType(ref, symbolTy)
				paramRefs = append(paramRefs, ref)
			}
			if seg.Kind == ast.ResourcePathSegmentParamRest {
				restMember = paramTy
			} else {
				members = append(members, paramTy)
			}
		}
	}
	listDefn := semtypes.NewListDefinition()
	pathTy := listDefn.Define(t.typeEnv(), members, semtypes.ListRest(restMember),
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	return pathTy, paramRefs, true
}

func resolveClientResourceAccessAction(t typeResolver, chain *binding, expr *ast.BLangClientResourceAccessAction, expectedType semtypes.SemType) (expressionResult, bool) {
	receiverResult, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return expressionResult{}, false
	}
	receiverTy := receiverResult.ty
	chain = sequentialChain(t, receiverResult.effect)
	if !semtypes.IsClientObject(t.typeContext(), receiverTy) {
		t.semanticError("resource access action is only allowed on client objects", expr.GetPosition())
		return expressionResult{}, false
	}
	atomicType := semtypes.ToObjectAtomicType(t.typeContext(), receiverTy)
	if atomicType == nil {
		t.unimplemented("non-atomic receiver for resource access action", expr.GetPosition())
		return expressionResult{}, false
	}
	table, ok := t.objectMethodTable(atomicType)
	if !ok {
		t.internalError("failed to find method table for receiver type", expr.GetPosition())
		return expressionResult{}, false
	}
	if table.Owner.IsEmpty() {
		t.internalError("client receiver method table has no owner", expr.GetPosition())
		return expressionResult{}, false
	}
	networkSym, ok := t.getSymbol(table.Owner).(*model.NetworkClassSymbol)
	if !ok {
		t.internalError("client reciever must have network class symbol", expr.GetPosition())
		return expressionResult{}, false
	}
	argPathTy, _, chain, ok := resolveResourceAccessPathType(t, chain, expr)
	if !ok {
		return expressionResult{}, false
	}
	methodName := expr.MethodName
	var matches []model.SymbolRef
	for _, rmRef := range networkSym.ResourceMethods() {
		rmSym, ok := t.getSymbol(rmRef).(model.ResourceMethodSymbol)
		if !ok {
			t.internalError("expected resource method symbol", expr.GetPosition())
			return expressionResult{}, false
		}
		if rmSym.MethodName() != methodName || !semtypes.IsSubtype(t.typeContext(), argPathTy, rmSym.PathListType()) {
			continue
		}
		matches = append(matches, rmRef)
	}
	if len(matches) == 0 {
		t.semanticError(fmt.Sprintf("no matching resource method '%s'", methodName), expr.GetPosition())
		return expressionResult{}, false
	}
	if len(matches) > 1 {
		t.unimplemented("ambiguous resource method dispatch", expr.GetPosition())
		return expressionResult{}, false
	}
	setMethodSymbol(t, expr, matches[0])
	return resolveFunctionCall(t, chain, expr, expr.ArgExprs, matches[0], expectedType)
}

func resolveResourceAccessPathType(t typeResolver, chain *binding, expr *ast.BLangClientResourceAccessAction) (semtypes.SemType, int, *binding, bool) {
	var members []semtypes.SemType
	for i := range expr.Path {
		seg := &expr.Path[i]
		switch seg.Kind {
		case ast.ResourceAccessSegmentName:
			members = append(members, semtypes.StringConst(seg.Name))
		case ast.ResourceAccessSegmentComputed:
			segResult, ok := resolveActionOrExpression(t, chain, seg.Expr, semtypes.SemType{})
			if !ok {
				return semtypes.SemType{}, 0, nil, false
			}
			chain = sequentialChain(t, segResult.effect)
			segTy := segResult.ty
			members = append(members, segTy)
		}
	}
	listDefn := semtypes.NewListDefinition()
	pathTy := listDefn.Define(t.typeEnv(), members,
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	return pathTy, len(members), chain, true
}

func resolveRemoteMethodCallAction(t typeResolver, chain *binding, expr *ast.BLangRemoteMethodCallAction, expectedType semtypes.SemType) (expressionResult, bool) {
	receiverResult, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return expressionResult{}, false
	}
	receiverTy := receiverResult.ty
	chain = sequentialChain(t, receiverResult.effect)
	if !semtypes.IsClientObject(t.typeContext(), receiverTy) {
		t.semanticError("remote method call is only allowed on client objects", expr.GetPosition())
		return expressionResult{}, false
	}
	methodName := expr.Name.GetValue()
	remoteMethodName := model.RemoteMethodName(methodName)
	if !isRemoteMethod(t, receiverTy, remoteMethodName) {
		t.semanticError(fmt.Sprintf("%s is not a remote method", methodName), expr.GetPosition())
		return expressionResult{}, false
	}
	if methodRef, found := sourceObjectMethodSymbol(t, receiverTy, remoteMethodName); found {
		return resolveFunctionCall(t, chain, expr, expr.ArgExprs, methodRef, expectedType)
	}
	symbolRef, retTy, effect, ok := finishResolveMethodCall(t, chain, receiverTy, remoteMethodName, expr.RawSymbol.(*common.DeferredMethodSymbol), expr.ArgExprs, expr)
	if !ok {
		return expressionResult{}, false
	}
	setMethodSymbol(t, expr, symbolRef)
	return expressionResult{ty: retTy, effect: effect, symbol: symbolRef}, true
}

func resolveLangLibImport(t typeResolver, pkgName string, methodName string, expr *ast.BLangInvocation) (model.SymbolRef, ast.BLangIdentifier, bool) {
	symbolSpace, ok := t.lookupImportedSymbols(pkgName)
	if !ok {
		t.internalError(fmt.Sprintf("%s symbol space not found", pkgName), expr.GetPosition())
		return model.SymbolRef{}, ast.BLangIdentifier{}, false
	}
	symbolRef, ok := symbolSpace.GetSymbol(methodName)
	if !ok {
		// Callers probe multiple lang libs and fall back to lang.value, so a miss
		// here is not an error and must not register an implicit import.
		return model.SymbolRef{}, ast.BLangIdentifier{}, false
	}
	return symbolRef, addImplicitLangLibImport(t, pkgName, expr.GetPosition()), true
}

func ensureLangLibImport(t typeResolver, pkgName string, pos diagnostics.Location) (model.ExportedSymbolSpace, ast.BLangIdentifier, bool) {
	symbolSpace, ok := t.lookupImportedSymbols(pkgName)
	if !ok {
		t.internalError(fmt.Sprintf("%s symbol space not found", pkgName), pos)
		return model.ExportedSymbolSpace{}, ast.BLangIdentifier{}, false
	}
	return symbolSpace, addImplicitLangLibImport(t, pkgName, pos), true
}

func addImplicitLangLibImport(t typeResolver, pkgName string, pos diagnostics.Location) ast.BLangIdentifier {
	pkgAlias := ast.BLangIdentifier{Value: pkgName}
	pkgAlias.SetPosition(pos)
	if !t.hasImplicitImport(pkgName) {
		moduleName := strings.TrimPrefix(pkgName, "lang.")
		orgIdent := &ast.BLangIdentifier{Value: "ballerina"}
		langIdent := ast.BLangIdentifier{Value: "lang"}
		moduleIdent := ast.BLangIdentifier{Value: moduleName}
		setPositions(pos, orgIdent, &langIdent, &moduleIdent)
		importNode := ast.BLangImportPackage{
			OrgName:      orgIdent,
			PkgNameComps: []ast.BLangIdentifier{langIdent, moduleIdent},
			Alias:        &pkgAlias,
		}
		t.addImplicitImport(pkgName, importNode)
	}
	return pkgAlias
}

type callArgsResult struct {
	argTys   []semtypes.SemType
	callee   model.SymbolRef
	calleeTy semtypes.SemType
	// sig is the callee's signature when its type is built from that signature, so the call can be
	// checked against the signature directly; otherwise it is nil and the call is checked against calleeTy.
	sig     *model.TypedFunctionSignature
	lowered []ast.BLangExpression
	chain   *binding
}

func resolveFunctionCallArgs(t typeResolver, chain *binding, inv invocable, args []ast.BLangExpression, fnSymbol model.SymbolRef, expectedType semtypes.SemType) (callArgsResult, bool) {
	baseSymbol := t.getSymbol(fnSymbol)
	switch sym := baseSymbol.(type) {
	case model.DependentlyTypedFunctionSymbol:
		lowered, fail := lowerInvocationArgs(t, args, fnSymbol, expectedType, inv.GetPosition())
		if fail != nil {
			fail(t)
			return callArgsResult{}, false
		}
		paramTypes := sym.ParamTypes()
		argTys, chain, ok := resolveArgs(t, lowered, chain, func(i int) semtypes.SemType {
			if i < len(paramTypes) {
				return paramTypes[i]
			}
			return semtypes.Never
		})
		if !ok {
			return callArgsResult{}, false
		}
		if !validateCallArgs(t, paramTypes, semtypes.Never, argTys, lowered, inv) {
			return callArgsResult{}, false
		}
		monoRef, monoSig, monoTy, ok := monomorphizeDependentCall(t, inv, sym, fnSymbol, argTys)
		if !ok {
			return callArgsResult{}, false
		}
		return callArgsResult{argTys: argTys, callee: monoRef, calleeTy: monoTy, sig: &monoSig, lowered: lowered, chain: chain}, true
	case *model.OpaqueFunctionSymbol:
		return resolveOpaqueCallArgs(t, chain, inv, args, sym, fnSymbol, expectedType)
	case model.FunctionSymbol:
		if !t.ensureResolved(fnSymbol, 0) {
			return callArgsResult{}, false
		}
		sig := sym.TypedSignature()
		lowered, fail := lowerInvocationArgs(t, args, fnSymbol, semtypes.SemType{}, inv.GetPosition())
		if fail != nil {
			fail(t)
			return callArgsResult{}, false
		}
		argTys, chain, ok := resolveArgs(t, lowered, chain, func(i int) semtypes.SemType {
			if i < len(sig.ParamTypes) {
				return sig.ParamTypes[i]
			}
			return sig.RestParamType
		})
		if !ok {
			return callArgsResult{}, false
		}
		return callArgsResult{argTys: argTys, callee: fnSymbol, calleeTy: t.symbolType(fnSymbol), sig: &sig, lowered: lowered, chain: chain}, true
	case model.ValueSymbol:
		narrowedSymbol := lookupSymbol(t, chain, fnSymbol)
		fnTy := t.symbolType(narrowedSymbol)
		if semtypes.IsZero(fnTy) {
			// A variable whose type failed to resolve has no type.
			return callArgsResult{}, false
		}
		if !semtypes.IsSubtype(t.typeContext(), fnTy, semtypes.Function) {
			t.semanticError("not a function value", inv.GetPosition())
			return callArgsResult{}, false
		}

		lowered := args
		if _, hasSig := t.functionSignature(narrowedSymbol); hasSig {
			var fail lowerFailure
			lowered, fail = lowerInvocationArgs(t, args, narrowedSymbol, expectedType, inv.GetPosition())
			if fail != nil {
				fail(t)
				return callArgsResult{}, false
			}
		}

		paramListTy := semtypes.FunctionParamListType(t.typeContext(), fnTy)
		if semtypes.IsZero(paramListTy) {
			// I don't think this can happen given we have already checked fnTy to be subtype of function
			t.internalError("empty function param list ty", inv.GetPosition())
			return callArgsResult{}, false
		}
		argTys, chain, ok := resolveArgs(t, lowered, chain, func(i int) semtypes.SemType {
			return semtypes.ListMemberTypeInnerVal(t.typeContext(), paramListTy, semtypes.IntConst(int64(i)))
		})
		if !ok {
			return callArgsResult{}, false
		}
		return callArgsResult{argTys: argTys, callee: narrowedSymbol, calleeTy: fnTy, lowered: lowered, chain: chain}, true
	default:
		t.semanticError("not a function value", inv.GetPosition())
		return callArgsResult{}, false
	}
}

func monomorphizeDependentCall(t typeResolver, inv invocable, sym model.DependentlyTypedFunctionSymbol, fnSymbol model.SymbolRef, argTys []semtypes.SemType) (model.SymbolRef, model.TypedFunctionSignature, semtypes.SemType, bool) {
	if t.isEphemeral() {
		monoSig := sym.Monomorphize(t.typeContext(), sym.Name(), fnSymbol, argTys).TypedSignature()
		return model.SymbolRef{}, monoSig, typeFromFunctionSignature(t, monoSig), true
	}
	monoName := t.nextMonoFnName(sym.Name())
	monoSym := sym.Monomorphize(t.typeContext(), monoName, fnSymbol, argTys)
	scope := t.currentScope()
	scope.AddSymbol(monoName, monoSym)
	monoRef, ok := scope.GetSymbol(monoName)
	if !ok {
		t.internalError("monomorphized symbol missing from scope", inv.GetPosition())
		return model.SymbolRef{}, model.TypedFunctionSignature{}, semtypes.SemType{}, false
	}
	if ref, ok := t.functionSignatureRef(fnSymbol); ok {
		if !t.associateFunctionSignature(monoRef, ref) {
			t.internalError("function signature already set", inv.GetPosition())
			return model.SymbolRef{}, model.TypedFunctionSignature{}, semtypes.SemType{}, false
		}
	}
	monoSig := monoSym.TypedSignature()
	monoTy := typeFromFunctionSignature(t, monoSig)
	monoSym.SetType(monoTy)
	return monoRef, monoSig, monoTy, true
}

func resolveOpaqueCallArgs(t typeResolver, chain *binding, inv invocable, args []ast.BLangExpression, sym *model.OpaqueFunctionSymbol,
	fnSymbol model.SymbolRef, expectedType semtypes.SemType,
) (callArgsResult, bool) {
	pkg := t.compilerContext().SymbolPackage(fnSymbol)
	defn, found := opaque.LookupFunction(pkg.Organization, pkg.Package, sym.OpaqueID())
	if !found {
		t.internalError("no definition for opaque function", inv.GetPosition())
		return callArgsResult{}, false
	}

	lowered, fail := lowerInvocationArgs(t, args, fnSymbol, expectedType, inv.GetPosition())
	if fail != nil {
		fail(t)
		return callArgsResult{}, false
	}

	// The monomorphizer resolves the arguments it needs to pick the function. Each argument is resolved only
	// once: the remaining ones are resolved afterwards, continuing from the chain these resolutions produce.
	resolveChain := chain
	resolvedArgs := make(map[ast.BLangExpression]semtypes.SemType)
	resolve := func(expr ast.BLangExpression, expected semtypes.SemType) (semtypes.SemType, bool) {
		if ty, ok := resolvedArgs[expr]; ok {
			return ty, true
		}
		result, ok := resolveActionOrExpression(t, resolveChain, expr, expected)
		if !ok {
			return semtypes.SemType{}, false
		}
		resolveChain = sequentialChain(t, result.effect)
		resolvedArgs[expr] = result.ty
		return result.ty, true
	}
	materialize := func(sig model.TypedFunctionSignature) (model.SymbolRef, bool) {
		return materializeOpaqueFn(t, sym, fnSymbol, sig, inv.GetPosition())
	}

	monoRef, ok := defn.Monomorphize(t.opaqueContext(), resolve, materialize, t.semanticError,
		opaqueCallIsolated(t, lowered), lowered, expectedType, inv.GetPosition())
	if !ok {
		return callArgsResult{}, false
	}

	sig := t.getSymbol(monoRef).(model.FunctionSymbol).TypedSignature()
	argTys := make([]semtypes.SemType, 0, len(lowered))
	chain = resolveChain
	for i, arg := range lowered {
		if ty, ok := resolvedArgs[arg]; ok {
			result := withSingletonEffect(chain, expressionResult{ty: ty, effect: defaultExpressionEffect(chain)})
			chain = sequentialChain(t, result.effect)
			argTys = append(argTys, ty)
			continue
		}
		paramTy := sig.RestParamType
		if i < len(sig.ParamTypes) {
			paramTy = sig.ParamTypes[i]
		}
		result, ok := resolveActionOrExpression(t, chain, arg, paramTy)
		if !ok {
			return callArgsResult{}, false
		}
		chain = sequentialChain(t, result.effect)
		argTys = append(argTys, result.ty)
	}
	return callArgsResult{argTys: argTys, callee: monoRef, calleeTy: t.symbolType(monoRef), sig: &sig, lowered: lowered, chain: chain}, true
}

type mappingField struct {
	name string
	expr ast.BLangExpression
	pos  diagnostics.Location
}

type lowerArgSlot struct {
	expr             ast.BLangExpression
	includedFields   []mappingField
	includedFieldPos diagnostics.Location
}

type lowerFailure func(typeResolver)

func semanticLowerFailure(msg string, pos diagnostics.Location) lowerFailure {
	return func(t typeResolver) {
		t.semanticError(msg, pos)
	}
}

func internalLowerFailure(msg string, pos diagnostics.Location) lowerFailure {
	return func(t typeResolver) {
		t.internalError(msg, pos)
	}
}

// lowerInvocationArgs lower arguments for invocation "like" expression (function/method call, new expression, client remote method call action, etc), such that after lowering
// we only have positional arguments. This means,
//
//	   positional arguments -> positional arguments
//	   named arguments      -> if name is a parameter then positional argument else field of a mapping constructor in the position of included record parameter
//	   defaulted arguments  -> default-argument marker if default expression != `<>`, otherwise typedesc
//		    The marker is replaced with a default-expression closure invocation during desugaring, after explicit arguments are hoisted.
//	NOTE: lowering depends on there being a UntypedFunctionSignature, if not using non positional arguments in an unsupported error. If you need to handle any such case you need to
//	properly set the UntypedFunctionSignature.
//	NOTE: lowering also validate the arguments it lower to be valid except for their type (lowering is untyped)
func lowerInvocationArgs(t typeResolver, args []ast.BLangExpression, fnRef model.SymbolRef, expectedType semtypes.SemType, pos diagnostics.Location) ([]ast.BLangExpression, lowerFailure) {
	sig, ok := t.functionSignature(fnRef)
	if !ok {
		return args, nil
	}
	return lowerInvocationArgsWithSignature(t, args, sig, fnRef, expectedType, pos)
}

func functionParamTypes(t typeResolver, fnRef model.SymbolRef) []semtypes.SemType {
	sym := t.getSymbol(fnRef)
	switch fn := sym.(type) {
	case model.DependentlyTypedFunctionSymbol:
		return fn.ParamTypes()
	case model.FunctionSymbol:
		return fn.TypedSignature().ParamTypes
	default:
		return nil
	}
}

func lowerInvocationArgsWithSignature(t typeResolver, args []ast.BLangExpression, sig model.UntypedFunctionSignature, fn model.SymbolRef, expectedType semtypes.SemType, pos diagnostics.Location) ([]ast.BLangExpression, lowerFailure) {
	fixedCount := sig.FixedParamCount()
	slots := make([]lowerArgSlot, fixedCount)
	var restArgs []ast.BLangExpression
	seenNamed := false
	seenNames := make(map[string]diagnostics.Location)

	// Move named args to correct position by either turning them to positional arg or accumulating fields to build incl. record arg
	for i, arg := range args {
		if named, ok := arg.(*ast.BLangNamedArgsExpression); ok {
			seenNamed = true
			if fail := lowerNamedCallArg(sig, slots, seenNames, named); fail != nil {
				return nil, fail
			}
			continue
		}
		if seenNamed {
			return nil, semanticLowerFailure("positional argument not allowed after named argument", arg.GetPosition())
		}
		if i < fixedCount {
			if slots[i].expr != nil || len(slots[i].includedFields) > 0 {
				return nil, semanticLowerFailure(fmt.Sprintf("repeated values for parameter %s", sig.ParamNames[i]), arg.GetPosition())
			}
			slots[i].expr = arg
		} else if sig.HasRest {
			restArgs = append(restArgs, arg)
		} else {
			return nil, semanticLowerFailure("too many arguments", arg.GetPosition())
		}
	}

	// Turn included record args to mapping constructors
	for i := range fixedCount {
		if sig.ParamFlags[i]&model.ParamFlagIncludedRecordParam == 0 || slots[i].expr != nil {
			continue
		}
		slots[i].expr = buildIncludedRecordArg(slots[i].includedFields, pos)
	}

	// Validate we have defaultable params for any missing
	for i := range fixedCount {
		if slots[i].expr != nil {
			continue
		}
		dp, ok := sig.DefaultableParam(i)
		if !ok {
			return nil, semanticLowerFailure(fmt.Sprintf("missing required parameter '%s'", sig.ParamNames[i]), pos)
		}
		if dp.Kind == model.DefaultableParamKindInferredTypedesc {
			if semtypes.IsZero(expectedType) {
				return nil, semanticLowerFailure(fmt.Sprintf("cannot infer typedesc argument for parameter '%s': no contextually expected type", sig.ParamNames[i]), pos)
			}
			paramTypes := functionParamTypes(t, fn)
			if i >= len(paramTypes) {
				return nil, internalLowerFailure("function parameter type not found", pos)
			}
			depSym, ok := t.getSymbol(fn).(model.DependentlyTypedFunctionSymbol)
			if !ok {
				return nil, internalLowerFailure("inferred typedesc param on non-dependent function", pos)
			}
			constraint := semtypes.TypedescConstraint(t.typeContext(), paramTypes[i])
			defaultArg, fail := lowerInferredTypedescDefaultArg(t, constraint, expectedType, depSym.ReturnType(), i, pos)
			if fail != nil {
				return nil, fail
			}
			slots[i].expr = defaultArg
			continue
		}
		defaultArg := &ast.BLangDefaultArg{DefaultClosure: dp.Symbol}
		defaultArg.SetPosition(pos)
		slots[i].expr = defaultArg
	}

	newArgs := make([]ast.BLangExpression, fixedCount+len(restArgs))
	for i := range fixedCount {
		newArgs[i] = slots[i].expr
	}
	copy(newArgs[fixedCount:], restArgs)
	return newArgs, nil
}

func lowerInferredTypedescDefaultArg(t typeResolver, constraint, expectedType semtypes.SemType, returnOp model.TypeOp, paramIndex int, pos diagnostics.Location) (ast.BLangExpression, lowerFailure) {
	ctx := t.typeContext()
	inferred, referenced, fail := inferTypedescConstraint(t, returnOp, paramIndex, constraint, expectedType, pos)
	if fail != nil {
		return nil, fail
	}
	if !referenced {
		return nil, internalLowerFailure("inferred typedesc parameter is not referenced by return type", pos)
	}
	if semtypes.IsEmpty(ctx, inferred) {
		return nil, semanticLowerFailure(
			fmt.Sprintf("cannot infer maximal type such that it is a subtype of both %s and %s",
				semtypes.ToString(ctx, constraint), semtypes.ToString(ctx, expectedType)),
			pos,
		)
	}
	expr := &ast.BLangTypedescExpr{Constraint: inferred}
	expr.SetPosition(pos)
	return expr, nil
}

func inferTypedescConstraint(t typeResolver, op model.TypeOp, paramIndex int, constraint, candidate semtypes.SemType, pos diagnostics.Location) (semtypes.SemType, bool, lowerFailure) {
	ctx := t.typeContext()
	switch current := op.(type) {
	case *model.RefTypeOp:
		if current.Index != paramIndex {
			return semtypes.Never, false, nil
		}
		if !semtypes.IsSubtype(ctx, candidate, constraint) {
			candidate = semtypes.Intersect(constraint, candidate)
		}
		return candidate, true, nil
	case *model.IdentityTypeOp:
		return semtypes.Never, false, nil
	case *model.ArrayTypeOp:
		if !typeOpReferencesParam(current.Element, paramIndex) {
			return semtypes.Never, false, nil
		}
		shape := arraySemType(ctx, semtypes.Val, current.Length, current.IsOpen)
		compatibleCandidate := semtypes.Intersect(candidate, shape)
		if semtypes.IsEmpty(ctx, compatibleCandidate) {
			return semtypes.Never, true, semanticLowerFailure(
				fmt.Sprintf("cannot infer typedesc argument from incompatible expected array type %s", semtypes.ToString(ctx, candidate)),
				pos,
			)
		}
		member := semtypes.ListProj(ctx, compatibleCandidate, semtypes.Int)
		if semtypes.IsEmpty(ctx, member) {
			return constraint, true, nil
		}
		return inferTypedescConstraint(t, current.Element, paramIndex, constraint, member, pos)
	case *model.BinaryTypeOp:
		lhsDepends := typeOpReferencesParam(current.Lhs, paramIndex)
		rhsDepends := typeOpReferencesParam(current.Rhs, paramIndex)
		if !lhsDepends && !rhsDepends {
			return semtypes.Never, false, nil
		}
		if lhsDepends && !rhsDepends {
			return inferTypedescConstraint(t, current.Lhs, paramIndex, constraint,
				inferBinaryOperandCandidate(ctx, current.Kind, candidate, current.Rhs.FixedPart(ctx)), pos)
		}
		if rhsDepends && !lhsDepends {
			return inferTypedescConstraint(t, current.Rhs, paramIndex, constraint,
				inferBinaryOperandCandidate(ctx, current.Kind, candidate, current.Lhs.FixedPart(ctx)), pos)
		}
		lhs, _, fail := inferTypedescConstraint(t, current.Lhs, paramIndex, constraint, candidate, pos)
		if fail != nil {
			return semtypes.Never, true, fail
		}
		rhs, _, fail := inferTypedescConstraint(t, current.Rhs, paramIndex, constraint, candidate, pos)
		if fail != nil {
			return semtypes.Never, true, fail
		}
		if current.Kind == model.TypeOpUnion {
			return semtypes.Intersect(lhs, rhs), true, nil
		}
		return semtypes.Union(lhs, rhs), true, nil
	default:
		return semtypes.Never, false, internalLowerFailure(fmt.Sprintf("unknown dependent return type op: %T", op), pos)
	}
}

func inferBinaryOperandCandidate(ctx semtypes.Context, kind model.BinaryTypeOpKind, candidate, fixed semtypes.SemType) semtypes.SemType {
	if kind != model.TypeOpUnion {
		return candidate
	}
	inferred := semtypes.Diff(candidate, fixed)
	if semtypes.IsEmpty(ctx, inferred) {
		return candidate
	}
	return inferred
}

func typeOpReferencesParam(op model.TypeOp, paramIndex int) bool {
	switch current := op.(type) {
	case *model.RefTypeOp:
		return current.Index == paramIndex
	case *model.ArrayTypeOp:
		return typeOpReferencesParam(current.Element, paramIndex)
	case *model.BinaryTypeOp:
		return typeOpReferencesParam(current.Lhs, paramIndex) || typeOpReferencesParam(current.Rhs, paramIndex)
	case *model.IdentityTypeOp:
		return false
	default:
		return false
	}
}

func arraySemType(ctx semtypes.Context, element semtypes.SemType, length int, isOpen bool) semtypes.SemType {
	definition := semtypes.NewListDefinition()
	if isOpen {
		return definition.Define(ctx.Env(), nil, semtypes.ListRest(element))
	}
	return definition.Define(ctx.Env(), []semtypes.SemType{element}, semtypes.ListFixedLength(length))
}

func lowerNamedCallArg(sig model.UntypedFunctionSignature, slots []lowerArgSlot, seenNames map[string]diagnostics.Location, expr *ast.BLangNamedArgsExpression) lowerFailure {
	name := expr.Name.GetValue()
	if _, seen := seenNames[name]; seen {
		return semanticLowerFailure(fmt.Sprintf("duplicate arguments for %s", name), expr.GetPosition())
	}
	seenNames[name] = expr.GetPosition()

	idx, result := sig.Index(name)
	if result != model.ParamIndexFound {
		return reportParamIndexError(result, hasIncludedRecordParam(sig, sig.FixedParamCount()), name, expr.GetPosition())
	}
	if sig.ParamFlags[idx]&model.ParamFlagIncludedRecordParam != 0 && name != sig.ParamNames[idx] {
		// field for included record param
		if slots[idx].expr != nil {
			return semanticLowerFailure(
				fmt.Sprintf("record value and field-level arguments for the same included record parameter '%s'", sig.ParamNames[idx]),
				expr.GetPosition(),
			)
		}
		slots[idx].includedFields = append(slots[idx].includedFields, mappingField{name: name, expr: expr.Expr, pos: expr.GetPosition()})
		if slots[idx].includedFieldPos == (diagnostics.Location{}) {
			// This means all the errors related to this mapping constructor is going to be against the first field. Ideally need to do better
			slots[idx].includedFieldPos = expr.GetPosition()
		}
		return nil
	} else {
		// actual named parameter
		if slots[idx].expr != nil {
			return semanticLowerFailure(fmt.Sprintf("repeated values for parameter %s", name), expr.GetPosition())
		}
		if len(slots[idx].includedFields) > 0 {
			return semanticLowerFailure(
				fmt.Sprintf("record value and field-level arguments for the same included record parameter '%s'", sig.ParamNames[idx]),
				expr.GetPosition(),
			)
		}
		slots[idx].expr = expr.Expr
		return nil
	}
}

func buildIncludedRecordArg(fields []mappingField, pos diagnostics.Location) *ast.BLangMappingConstructorExpr {
	mc := &ast.BLangMappingConstructorExpr{}
	mc.SetPosition(pos)
	mc.Fields = make([]ast.MappingField, 0, len(fields))
	for _, field := range fields {
		fieldPos := field.pos
		if fieldPos == (diagnostics.Location{}) {
			fieldPos = field.expr.GetPosition()
		}
		keyLit := ast.NewBLangLiteral(fieldPos, ast.LiteralKindString, field.name, field.name, false)
		key := &ast.BLangMappingKey{Expr: keyLit}
		key.SetPosition(fieldPos)
		kv := &ast.BLangMappingKeyValueField{Key: key, ValueExpr: field.expr}
		kv.SetPosition(fieldPos)
		mc.Fields = append(mc.Fields, kv)
	}
	return mc
}

func resolveArgs(t typeResolver, args []ast.BLangExpression, chain *binding, paramType func(int) semtypes.SemType) ([]semtypes.SemType, *binding, bool) {
	tys := make([]semtypes.SemType, 0, len(args))
	for i, arg := range args {
		if _, namedParam := arg.(*ast.BLangNamedArgsExpression); namedParam {
			// See lowerInvocationArgs
			t.unimplemented("named arguments not supported in this context", arg.GetPosition())
			return nil, chain, false
		}
		tyResult, ok := resolveActionOrExpression(t, chain, arg, paramType(i))
		if !ok {
			return nil, chain, false
		}
		ty, effect := tyResult.ty, tyResult.effect
		chain = sequentialChain(t, effect)
		tys = append(tys, ty)
	}
	return tys, chain, true
}

func hasIncludedRecordParam(sig model.UntypedFunctionSignature, nRequired int) bool {
	for i := range nRequired {
		if sig.ParamFlags[i]&model.ParamFlagIncludedRecordParam != 0 {
			return true
		}
	}
	return false
}

func reportParamIndexError(result model.ParamIndexResult, hasIncludedRecord bool, name string, pos diagnostics.Location) lowerFailure {
	switch result {
	case model.ParamIndexNotFound:
		if hasIncludedRecord {
			return semanticLowerFailure(fmt.Sprintf("no included record parameter accepts named argument '%s'", name), pos)
		}
		return semanticLowerFailure(fmt.Sprintf("no such parameter %s", name), pos)
	case model.ParamIndexAmbiguous:
		return semanticLowerFailure(fmt.Sprintf("named argument '%s' matches multiple included record parameters", name), pos)
	default:
		return internalLowerFailure("invalid parameter index result", pos)
	}
}

func resolveFunctionCall(t typeResolver, chain *binding, inv invocable, args []ast.BLangExpression, symbolRef model.SymbolRef, expectedType semtypes.SemType) (expressionResult, bool) {
	call, retTy, ok := resolveCallReturnType(t, chain, inv, args, symbolRef, expectedType)
	if !ok {
		return expressionResult{}, false
	}
	setInvocationResolvedSymbol(t, inv, call.callee)
	setInvocationCallArgs(t, inv, call.lowered)
	setNodeType(t, inv, retTy)
	return expressionResult{ty: retTy, effect: defaultExpressionEffect(call.chain), symbol: call.callee}, true
}

func resolveCallReturnType(t typeResolver, chain *binding, inv invocable, args []ast.BLangExpression, symbolRef model.SymbolRef, expectedType semtypes.SemType) (callArgsResult, semtypes.SemType, bool) {
	call, ok := resolveFunctionCallArgs(t, chain, inv, args, symbolRef, expectedType)
	if !ok {
		return callArgsResult{}, semtypes.SemType{}, false
	}
	var retTy semtypes.SemType
	if call.sig != nil {
		if !validateCallArgs(t, call.sig.ParamTypes, call.sig.RestParamType, call.argTys, call.lowered, inv) {
			return callArgsResult{}, semtypes.SemType{}, false
		}
		retTy = call.sig.ReturnType
	} else {
		retTy, ok = functionTypeCallReturnType(t, call.calleeTy, call.argTys, inv)
	}
	if !ok {
		return callArgsResult{}, semtypes.SemType{}, false
	}
	return call, retTy, true
}

func validateCallArgs(t typeResolver, paramTypes []semtypes.SemType, restTy semtypes.SemType, argTys []semtypes.SemType, args []ast.BLangExpression, inv invocable) bool {
	cx := t.typeContext()
	nParams := len(paramTypes)
	if len(argTys) < nParams {
		t.internalError("argument count does not match signature after lowering", inv.GetPosition())
		return false
	}
	// RestParamType is zero for functions declared in object type descriptors without a rest param
	if semtypes.IsZero(restTy) {
		restTy = semtypes.Never
	}
	valid := true
	for i, argTy := range argTys {
		paramTy := restTy
		if i < nParams {
			paramTy = paramTypes[i]
		}
		if !semtypes.IsSubtype(cx, argTy, paramTy) {
			t.semanticError(fmt.Sprintf("incompatible type for argument: expected '%s', got '%s'",
				semtypes.ToString(cx, paramTy), semtypes.ToString(cx, argTy)), args[i].GetPosition())
			valid = false
		}
	}
	return valid
}

func functionTypeCallReturnType(t typeResolver, fnTy semtypes.SemType, argTys []semtypes.SemType, inv invocable) (semtypes.SemType, bool) {
	retTy := functionTypeReturnType(t, fnTy, argTys)
	if semtypes.IsZero(retTy) {
		// This can only happen when function call is not well-typed and since we
		// ensure funcTy is a function subtype, this can only be caused by invalid args
		t.semanticError("incompatible arguments for function call", inv.GetPosition())
		return semtypes.SemType{}, false
	}
	return retTy, true
}

func functionTypeReturnType(t typeResolver, fnTy semtypes.SemType, argTys []semtypes.SemType) semtypes.SemType {
	argLd := semtypes.NewListDefinition()
	argListTy := argLd.Define(t.typeEnv(), argTys,
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	return semtypes.FunctionReturnType(t.typeContext(), fnTy, argListTy)
}

// methodMemberType returns a function type describing a class method for inclusion in its
// object type. For a dependently-typed method the symbol has no stored type (monomorphization
// happens per call site); synthesize a function type from its param types and the return type
// that results from applying the return TypeOp against those param types.
func methodMemberType(t typeResolver, methodRef model.SymbolRef) semtypes.SemType {
	sym := t.getSymbol(methodRef)
	depSym, ok := sym.(model.DependentlyTypedFunctionSymbol)
	if !ok {
		return t.symbolType(methodRef)
	}
	paramTypes := depSym.ParamTypes()
	retTy := depSym.ReturnType().Apply(t.typeContext(), paramTypes)
	sig := model.TypedFunctionSignature{
		ParamTypes:    paramTypes,
		ReturnType:    retTy,
		RestParamType: semtypes.Never,
		Flags:         depSym.FuncFlags(),
	}
	return typeFromFunctionSignature(t, sig)
}

func typeFromFunctionSignature(t typeResolver, sig model.TypedFunctionSignature) semtypes.SemType {
	return opaque.FunctionSemType(t.typeEnv(), sig)
}

func resolveFixedArraySize(t typeResolver, lenExp ast.BLangExpression) (int, bool) {
	actionOrExpr, ok := lenExp.(ast.BLangActionOrExpression)
	if !ok {
		t.semanticError("fixed-length array size must be a singleton int", lenExp.GetPosition())
		return 0, false
	}
	sizeResult, ok := resolveActionOrExpression(t, nil, actionOrExpr, semtypes.Int)
	if !ok {
		return 0, false
	}
	sizeTy := sizeResult.ty
	if semtypes.IsZero(sizeTy) || !semtypes.IsSubtype(t.typeContext(), sizeTy, semtypes.Int) {
		t.semanticError("fixed-length array size must be a singleton int", lenExp.GetPosition())
		return 0, false
	}
	shape := semtypes.SingleShape(sizeTy)
	if shape.IsEmpty() {
		t.semanticError("fixed-length array size must be a singleton int", lenExp.GetPosition())
		return 0, false
	}
	val, ok := shape.Get().Value.(int64)
	if !ok {
		t.semanticError("fixed-length array size must be a singleton int", lenExp.GetPosition())
		return 0, false
	}
	if val < 0 {
		t.semanticError("fixed-length array size must be non-negative", lenExp.GetPosition())
		return 0, false
	}
	return int(val), true
}

func resolveBType(t typeResolver, chain *binding, btype ast.BType, depth int) (semtypes.SemType, *binding, bool) {
	if ty := btype.Type(); !semtypes.IsZero(ty) {
		return ty, chain, true
	}
	res, chain, ok := resolveBTypeInner(t, chain, btype, depth)
	if !ok {
		return semtypes.SemType{}, chain, false
	}
	setResolvedTypeData(t, btype, res)
	return res, chain, true
}

func setResolvedTypeData(t typeResolver, btype ast.BType, ty semtypes.SemType) {
	typeData := btype.GetTypeData()
	typeData.Type = ty
	setTypeData(t, btype, typeData)
}

// isValidTypeDescRef reports whether a type descriptor may refer to ref. Only a typedesc
// parameter denotes a type descriptor, and only within the return type of a dependently-typed
// function; every other parameter reference in a type position is an error. Reports the error
// and returns false in that case.
func isValidTypeDescRef(t typeResolver, ref model.SymbolRef, pos diagnostics.Location) bool {
	sym, ok := t.getSymbol(ref).(model.ValueSymbol)
	if !ok || !sym.IsParameter() {
		return true
	}
	paramTy := t.symbolType(ref)
	if semtypes.IsSubtype(t.typeContext(), paramTy, semtypes.Typedesc) {
		return true
	}
	t.semanticError(fmt.Sprintf("invalid parameter reference: expected 'typedesc', found '%s'",
		semtypes.ToString(t.typeContext(), paramTy)), pos)
	return false
}

func resolveTypeDataPair(t typeResolver, chain *binding, typeData *ast.TypeData, depth int) (semtypes.SemType, *binding, bool) {
	ty, chain, ok := resolveBType(t, chain, typeData.TypeDescriptor.(ast.BType), depth)
	if !ok {
		return semtypes.SemType{}, chain, false
	}
	setTypeDataType(t, typeData, ty)
	return ty, chain, true
}

func resolveStreamCompletionType(t typeResolver, chain *binding, typeData *ast.TypeData, depth int) (semtypes.SemType, *binding, bool) {
	if typeData.TypeDescriptor == nil {
		setTypeDataType(t, typeData, semtypes.Nil)
		return semtypes.Nil, chain, true
	}
	return resolveTypeDataPair(t, chain, typeData, depth)
}

// typeDescriptorDeclaresSymbols reports whether resolving btype writes symbol state: method symbols of an object
// type or named parameters of a function type.
func typeDescriptorDeclaresSymbols(btype ast.BType) bool {
	switch ty := btype.(type) {
	case *ast.BLangObjectType:
		for m := range ty.Members() {
			if _, isMethod := m.(*ast.BMethodDecl); isMethod {
				return true
			}
		}
	case *ast.BLangFunctionType:
		for i := range ty.RequiredParams {
			if !ty.RequiredParams[i].SymbolRef.IsEmpty() {
				return true
			}
		}
		return ty.RestParam != nil && !ty.RestParam.SymbolRef.IsEmpty()
	}
	return false
}

// isDefinitionSet reports whether resolving btype reuses an existing atom. When it returns false, resolving btype
// defines a new atom and registers metadata for it, mapping defaults for a record type or a method table for an
// object type, outside the ephemeral-gated state.
func isDefinitionSet(btype ast.BType) bool {
	switch ty := btype.(type) {
	case *ast.BLangRecordType:
		return ty.Definition != nil
	case *ast.BLangObjectType:
		return ty.Definition != nil
	}
	return true
}

func resolveBTypeInner(t typeResolver, chain *binding, btype ast.BType, depth int) (_ semtypes.SemType, _ *binding, ok bool) {
	// closeDefinition closes the definition a descriptor created if the descriptor fails.
	var closeDefinition func()
	defer func() {
		if !ok && closeDefinition != nil {
			closeDefinition()
		}
	}()
	if t.isEphemeral() && typeDescriptorDeclaresSymbols(btype) {
		t.unimplemented("type descriptors that declare symbols are not supported while selecting among candidate types", btype.(ast.BLangNode).GetPosition())
		return semtypes.SemType{}, chain, false
	}
	if t.isEphemeral() && !isDefinitionSet(btype) {
		t.unimplemented("inline record and object type descriptors are not supported while selecting among candidate types", btype.(ast.BLangNode).GetPosition())
		return semtypes.SemType{}, chain, false
	}
	switch ty := btype.(type) {
	case *ast.BLangReturnTypeDescriptor:
		return resolveBType(t, chain, ty.TypeDescriptor, depth)
	case *ast.BLangBadTypeNode:
		return semtypes.SemType{}, chain, false
	case *ast.BLangValueType:
		switch ty.TypeKind {
		case ast.TypeKindBoolean:
			return semtypes.Boolean, chain, true
		case ast.TypeKindInt:
			return semtypes.Int, chain, true
		case ast.TypeKindFloat:
			return semtypes.Float, chain, true
		case ast.TypeKindString:
			return semtypes.String, chain, true
		case ast.TypeKindNil:
			return semtypes.Nil, chain, true
		case ast.TypeKindAny:
			return semtypes.Any, chain, true
		case ast.TypeKindDecimal:
			return semtypes.Decimal, chain, true
		case ast.TypeKindByte:
			return semtypes.Byte, chain, true
		case ast.TypeKindAnyData:
			return semtypes.CreateAnydata(t.typeContext()), chain, true
		case ast.TypeKindHandle:
			return semtypes.Handle, chain, true
		case ast.TypeKindTypeDesc:
			return semtypes.Typedesc, chain, true
		case ast.TypeKindXML:
			return semtypes.XML, chain, true
		case ast.TypeKindReadOnly:
			return semtypes.ValReadonly, chain, true
		case ast.TypeKindNever:
			return semtypes.Never, chain, true
		default:
			t.internalError("unexpected type tag", diagnostics.Location{})
			return semtypes.SemType{}, chain, false
		}
	case *ast.BLangArrayType:
		defn := ty.Definition
		var semTy semtypes.SemType
		if defn == nil {
			d := semtypes.NewListDefinition()
			setTypeDefinition(t, &ty.Definition, &d)
			closeDefinition = func() { defineNeverList(t, &d) }
			var elemTy semtypes.SemType
			var ok bool
			elemTy, chain, ok = resolveTypeDataPair(t, chain, &ty.Elemtype, depth+1)
			if !ok {
				return semtypes.SemType{}, chain, false
			}
			for i := len(ty.Sizes); i > 0; i-- {
				lenExp := ty.Sizes[i-1]
				if lenExp == nil {
					elemTy = d.Define(t.typeEnv(), nil, semtypes.ListRest(elemTy))
				} else {
					length, ok := resolveFixedArraySize(t, lenExp)
					if !ok {
						return semtypes.SemType{}, chain, false
					}
					elemTy = d.Define(t.typeEnv(), []semtypes.SemType{elemTy}, semtypes.ListFixedLength(length))
				}
			}
			semTy = elemTy
		} else {
			semTy = defn.GetSemType(t.typeEnv())
		}
		return semTy, chain, true
	case *ast.BLangUnionTypeNode:
		lhs, chain, ok := resolveTypeDataPair(t, chain, ty.Lhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		rhs, chain, ok := resolveTypeDataPair(t, chain, ty.Rhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		return semtypes.Union(lhs, rhs), chain, true
	case *ast.BLangIntersectionTypeNode:
		lhs, chain, ok := resolveTypeDataPair(t, chain, ty.Lhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		rhs, chain, ok := resolveTypeDataPair(t, chain, ty.Rhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		result := semtypes.Intersect(lhs, rhs)
		pos := ty.GetPosition()
		if !t.ensureNotEmpty(result, func() {
			t.semanticError("intersection type is empty (equivalent to never)", pos)
		}) {
			return semtypes.SemType{}, chain, false
		}
		return result, chain, true
	case *ast.BLangErrorTypeNode:
		if ty.IsTop() {
			return semtypes.Error, chain, true
		} else {
			detailTy, chain, ok := resolveBType(t, chain, ty.DetailType.TypeDescriptor.(ast.BType), depth+1)
			if !ok {
				return semtypes.SemType{}, chain, false
			}
			setTypeDataType(t, &ty.DetailType, detailTy)
			return semtypes.ErrorWithDetail(detailTy), chain, true
		}
	case *ast.BLangUserDefinedType:
		symbol := ty.Symbol()
		if ty.PkgAlias.GetValue() != "" {
			return t.symbolType(symbol), chain, true
		}
		if t.isEphemeral() {
			symbol = lookupDeclaredSymbol(chain, symbol)
		}
		if !t.ensureResolved(symbol, depth) {
			return semtypes.SemType{}, chain, false
		}
		if !isValidTypeDescRef(t, symbol, ty.GetPosition()) {
			return semtypes.SemType{}, chain, false
		}
		return t.symbolType(symbol), chain, true
	case *ast.BLangFiniteTypeNode:
		result := semtypes.Never
		for _, value := range ty.ValueSpace {
			valueResult, ok := resolveActionOrExpression(t, nil, value, semtypes.SemType{})
			if !ok {
				return semtypes.SemType{}, chain, false
			}
			valueTy := valueResult.ty
			result = semtypes.Union(result, valueTy)
		}
		return result, chain, true
	case *ast.BLangConstrainedType:
		if _, _, ok := resolveTypeDataPair(t, chain, &ty.BaseType, depth+1); !ok {
			return semtypes.SemType{}, chain, false
		}
		defn := ty.Definition
		if defn == nil {
			switch ty.ConstraintKind() {
			case ast.TypeKindMap:
				d := semtypes.NewMappingDefinition()
				setTypeDefinition(t, &ty.Definition, &d)
				closeDefinition = func() { defineNeverMapping(t, &d) }
				rest, chain, ok := resolveTypeDataPair(t, chain, &ty.Constraint, depth+1)
				if !ok {
					return semtypes.SemType{}, chain, false
				}
				semType := d.Define(t.typeEnv(), nil, rest)
				return semType, chain, true
			case ast.TypeKindTypeDesc:
				constraint, chain, ok := resolveTypeDataPair(t, chain, &ty.Constraint, depth+1)
				if !ok {
					return semtypes.SemType{}, chain, false
				}
				return semtypes.TypedescContaining(t.typeEnv(), constraint), chain, true
			case ast.TypeKindFuture:
				d := semtypes.NewFutureDefinition()
				setTypeDefinition(t, &ty.Definition, &d)
				constraint, chain, ok := resolveTypeDataPair(t, chain, &ty.Constraint, depth+1)
				if !ok {
					setTypeDefinition(t, &ty.Definition, nil)
					d.Define(t.typeEnv(), semtypes.Never)
					return semtypes.SemType{}, chain, false
				}
				return d.Define(t.typeEnv(), constraint), chain, true
			case ast.TypeKindXML:
				constraint, chain, ok := resolveTypeDataPair(t, chain, &ty.Constraint, depth+1)
				if !ok {
					return semtypes.SemType{}, chain, false
				}
				if !semtypes.IsSubtype(t.typeContext(), constraint, semtypes.XML) {
					t.semanticError(fmt.Sprintf("xml type constraint must be a subtype of xml, got %s", semtypes.ToString(t.typeContext(), constraint)), ty.GetPosition())
					return semtypes.SemType{}, chain, false
				}
				return semtypes.XMLSequence(constraint), chain, true
			default:
				t.unimplemented("unsupported base type kind", diagnostics.Location{})
				return semtypes.SemType{}, chain, false
			}
		} else {
			return defn.GetSemType(t.typeEnv()), chain, true
		}
	case *ast.BLangBuiltInRefTypeNode:
		switch ty.TypeKind {
		case ast.TypeKindMap:
			return semtypes.Mapping, chain, true
		case ast.TypeKindJSON:
			return semtypes.CreateJSON(t.typeContext()), chain, true
		case ast.TypeKindAnyData:
			return semtypes.CreateAnydata(t.typeContext()), chain, true
		case ast.TypeKindAny:
			return semtypes.Any, chain, true
		case ast.TypeKindXML:
			return semtypes.XML, chain, true
		case ast.TypeKindStream:
			return semtypes.Stream, chain, true
		case ast.TypeKindFuture:
			return semtypes.Future, chain, true
		case ast.TypeKindTable:
			t.unimplemented("unsupported builtin type kind: "+ty.TypeKind.String(), ty.GetPosition())
			return semtypes.SemType{}, chain, false
		default:
			t.internalError("Unexpected builtin type kind", ty.GetPosition())
		}
		return semtypes.SemType{}, chain, false
	case *ast.BLangStreamType:
		if defn := ty.Definition; defn != nil {
			return defn.GetSemType(t.typeEnv()), chain, true
		}
		valueTy, chain, ok := resolveTypeDataPair(t, chain, &ty.ValueType, depth+1)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		completionTy, chain, ok := resolveStreamCompletionType(t, chain, &ty.CompletionType, depth+1)
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		if !semtypes.IsSubtype(t.typeContext(), completionTy, semtypes.Union(semtypes.Error, semtypes.Nil)) {
			t.semanticError(
				"stream completion type must be a subtype of error?",
				ty.CompletionType.TypeDescriptor.GetPosition(),
			)
			return semtypes.SemType{}, chain, false
		}
		d := semtypes.NewStreamDefinition()
		setTypeDefinition(t, &ty.Definition, &d)
		return d.Define(t.typeEnv(), valueTy, completionTy), chain, true
	case *ast.BLangTupleTypeNode:
		defn := ty.Definition
		if defn == nil {
			d := semtypes.NewListDefinition()
			setTypeDefinition(t, &ty.Definition, &d)
			closeDefinition = func() { defineNeverList(t, &d) }
			members := make([]semtypes.SemType, len(ty.Members))
			for i, member := range ty.Members {
				var ok bool
				members[i], chain, ok = resolveBType(t, chain, member.TypeDesc.(ast.BType), depth+1)
				if !ok {
					return semtypes.SemType{}, chain, false
				}
			}
			rest := semtypes.Never
			if ty.Rest != nil {
				var ok bool
				rest, chain, ok = resolveBType(t, chain, ty.Rest, depth+1)
				if !ok {
					return semtypes.SemType{}, chain, false
				}
			}
			return d.Define(t.typeEnv(), members, semtypes.ListRest(rest)), chain, true
		}
		return defn.GetSemType(t.typeEnv()), chain, true
	case *ast.BLangRecordType:
		defn := ty.Definition
		if defn != nil {
			return defn.GetSemType(t.typeEnv()), chain, true
		}
		d := semtypes.NewMappingDefinition()
		setTypeDefinition(t, &ty.Definition, &d)
		closeDefinition = func() { defineNeverMapping(t, &d) }

		// Resolve and collect included members from symbols
		result, chain, ok := resolveRecordInclusions(t, chain, ty, depth)
		if !ok {
			return semtypes.SemType{}, chain, false
		}

		seen := make(map[string]bool)
		var fields []semtypes.Field
		// TODO: need to think of a way to unify this with objects
		for name, field := range ty.FieldPtrs() {
			if seen[name] {
				t.semanticError(fmt.Sprintf("duplicate field name '%s'", name), field.GetPosition())
				return semtypes.SemType{}, chain, false
			}
			seen[name] = true
			fieldTy, fieldChain, ok := resolveBType(t, chain, field.Type, depth+1)
			chain = fieldChain
			if !ok {
				// A field whose type fails to resolve is left out of the record type.
				continue
			}
			if incMembers, exists := result.includedFields[name]; exists {
				for _, incMember := range incMembers {
					if !semtypes.IsSubtype(t.typeContext(), fieldTy, incMember.MemberType()) {
						t.semanticError(
							fmt.Sprintf("field '%s' of type that overrides included field is not a subtype of the included field type", name),
							field.GetPosition(),
						)
					}
				}
				delete(result.includedFields, name)
			}
			if field.Default != nil {
				restoreContext := setIsolatedXMLStepContext(t, true)
				defaultResult, ok := resolveDefaultExpression(t, chain, field, field.Default.Expr, fieldTy)
				restoreContext()
				// The field's type does not depend on its default, so the field is kept even if the default fails.
				if ok {
					chain = sequentialChain(t, defaultResult.effect)
				}
			}
			ro := field.IsReadonly()
			opt := field.IsOptional()
			fields = append(fields, semtypes.FieldFrom(name, fieldTy, ro, opt))
		}

		for name, incMembers := range result.includedFields {
			if len(incMembers) > 1 {
				t.semanticError(fmt.Sprintf("included field '%s' declared in multiple type inclusions must be overridden", name), ty.GetPosition())
			}
		}

		for name, incMembers := range result.includedFields {
			if len(incMembers) > 1 {
				continue
			}
			fd := incMembers[0]
			fields = append(fields, semtypes.FieldFrom(name, fd.MemberType(), fd.IsReadonly(), fd.IsOptional()))
		}

		var rest semtypes.SemType
		if ty.RestType != nil {
			var ok bool
			rest, chain, ok = resolveBType(t, chain, ty.RestType, depth+1)
			if !ok {
				return semtypes.SemType{}, chain, false
			}
		} else if ty.IsOpen {
			rest = semtypes.CreateAnydata(t.typeContext())
		} else if result.multpleRestTy {
			t.semanticError("included rest type declared in multiple type inclusions must be overridden", ty.GetPosition())
			rest = semtypes.Never
		} else if !semtypes.IsZero(result.includedRestTy) {
			rest = result.includedRestTy
		} else {
			rest = semtypes.Never
		}
		semType := d.Define(t.typeEnv(), fields, rest)
		mat := semtypes.ToMappingAtomicType(t.typeContext(), semType)
		t.setMappingDefaults(mat, recordFieldDefaults(t, ty))
		return semType, chain, true
	case *ast.BLangFunctionType:
		if ty.IsAnyFunction() {
			return semtypes.Function, chain, true
		}
		if ty.Definition != nil {
			return ty.Definition.GetSemType(t.typeEnv()), chain, true
		}
		fd := semtypes.NewFunctionDefinition()
		setTypeDefinition(t, &ty.Definition, &fd)
		closeDefinition = func() { defineNeverFunction(t, &fd) }
		paramTypes := make([]semtypes.SemType, len(ty.RequiredParams))
		for i := range ty.RequiredParams {
			var paramTy semtypes.SemType
			var ok bool
			paramTy, chain, ok = resolveBType(t, chain, ty.RequiredParams[i].TypeDesc, depth+1)
			if !ok {
				return semtypes.SemType{}, chain, false
			}
			paramTypes[i] = paramTy
			if !ty.RequiredParams[i].SymbolRef.IsEmpty() {
				t.setSymbolType(ty.RequiredParams[i].SymbolRef, paramTy)
			}
			if ty.RequiredParams[i].InitExpr != nil {
				defaultResult, ok := resolveDefaultExpression(t, chain, ty.RequiredParams[i], ty.RequiredParams[i].InitExpr, paramTy)
				if !ok {
					return semtypes.SemType{}, chain, false
				}
				chain = sequentialChain(t, defaultResult.effect)
			}
		}
		restTy := semtypes.Never
		if ty.RestParam != nil {
			var ok bool
			restTy, chain, ok = resolveBType(t, chain, ty.RestParam.TypeDesc, depth+1)
			if !ok {
				return semtypes.SemType{}, chain, false
			}
			if !ty.RestParam.SymbolRef.IsEmpty() {
				t.setSymbolType(ty.RestParam.SymbolRef, restTy)
			}
		}
		paramListDefn := semtypes.NewListDefinition()
		paramListTy := paramListDefn.Define(t.typeEnv(), paramTypes, semtypes.ListRest(restTy),
			semtypes.ListMutability(semtypes.CellMutabilityNone))
		var returnTy semtypes.SemType
		if ty.ReturnTypeDescriptor != nil {
			var ok bool
			returnTy, chain, ok = resolveBType(t, chain, ty.ReturnTypeDescriptor, depth+1)
			if !ok {
				return semtypes.SemType{}, chain, false
			}
		} else {
			returnTy = semtypes.Nil
		}
		isolated := ty.IsIsolated()
		transactional := ty.IsTransactional()
		fnType := fd.Define(t.typeEnv(), paramListTy, returnTy,
			semtypes.FunctionQualifiersFrom(t.typeEnv(), isolated, transactional))
		closeDefinition = nil
		if !finalizeFunctionSignature(t, ty, t.functionSignatureByRef(ty.SignatureRef()), ty.SignatureRef(), paramTypes) {
			return semtypes.SemType{}, chain, false
		}
		return fnType, chain, true
	case *ast.BLangObjectType:
		return resolveObjectType(t, chain, ty, depth, model.SymbolRef{})
	default:
		t.unimplemented("unsupported type", diagnostics.Location{})
		return semtypes.SemType{}, chain, false
	}
}

// A descriptor that fails after creating its definition defines it with never for every component, so a recursive
// reference taken while resolving it does not leave an undefined atom in the type env.

func defineNeverList(t typeResolver, d *semtypes.ListDefinition) {
	d.Define(t.typeEnv(), nil, semtypes.ListRest(semtypes.Never))
}

func defineNeverMapping(t typeResolver, d *semtypes.MappingDefinition) {
	d.Define(t.typeEnv(), nil, semtypes.Never)
}

func defineNeverFunction(t typeResolver, d *semtypes.FunctionDefinition) {
	d.Define(t.typeEnv(), semtypes.Never, semtypes.Never, semtypes.FunctionQualifiersFrom(t.typeEnv(), false, false))
}

func defineNeverObject(t typeResolver, d *semtypes.ObjectDefinition) {
	d.Define(t.typeEnv(), semtypes.ObjectQualifiersDefault, nil)
}

func resolveObjectType(t typeResolver, chain *binding, ty *ast.BLangObjectType, depth int, owner model.SymbolRef) (_ semtypes.SemType, _ *binding, ok bool) {
	defn := ty.Definition
	if defn != nil {
		return defn.GetSemType(t.typeEnv()), chain, true
	}
	od := semtypes.NewObjectDefinition()
	setTypeDefinition(t, &ty.Definition, &od)
	defer func() {
		if !ok {
			defineNeverObject(t, &od)
		}
	}()
	// Step 1: Accumulate included members from symbols
	includedMembers := make(map[string][]semtypes.Member)
	incMembers, ok := collectIncludedMembers(t, ty.Inclusions, ty.InclusionPositions, ty.GetPosition(), depth)
	if !ok {
		return semtypes.SemType{}, chain, false
	}
	for _, m := range incMembers {
		if m.MemberKind() == model.InclusionMemberKindRestType {
			t.internalError("unexpected rest inclusion", ty.GetPosition())
			return semtypes.SemType{}, chain, false
		}
		member, ok := inclusionMemberToSemtypeMember(t, m, ty.GetPosition())
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		includedMembers[member.Name] = append(includedMembers[member.Name], member)
	}

	// Step 2: Build direct members and validate overrides
	var directMembers []directMember
	for m := range ty.Members() {
		if m.MemberKind() == ast.ObjectMemberKindRemoteMethod {
			if ty.NetworkQuals != ast.ObjectNetworkQualsClient && ty.NetworkQuals != ast.ObjectNetworkQualsService {
				t.semanticError("remote methods are only allowed in client or service object types", ty.GetPosition())
				return semtypes.SemType{}, chain, false
			}
		}
		var valueTy semtypes.SemType
		var ok bool
		valueTy, chain, ok = resolveObjectMemberType(t, chain, m, depth)
		if !ok {
			// A member that fails to resolve is left out of the object type.
			continue
		}
		kind, ok := semtypeMemberKind(t, m.MemberKind(), ty.GetPosition())
		if !ok {
			return semtypes.SemType{}, chain, false
		}
		directMembers = append(directMembers, directMember{
			name:       m.Name(),
			valueTy:    valueTy,
			kind:       kind,
			visibility: memberVisibility(t, isPublicObjectTypeMember(m), ""),
			immutable:  m.MemberKind() != ast.ObjectMemberKindField,
			pos:        m.(ast.BLangNode).GetPosition(),
		})
	}

	members, ok := validateOverridesAndMerge(t, directMembers, includedMembers, ty.GetPosition(), true)
	if !ok {
		return semtypes.SemType{}, chain, false
	}

	// Step 3: Create semtype
	networkQual, ok := semtypeNetworkQualifier(t, ty.NetworkQuals, ty.GetPosition())
	if !ok {
		return semtypes.SemType{}, chain, false
	}
	qualifiers := semtypes.ObjectQualifiersFrom(ty.Isolated, false, networkQual)
	semType := od.Define(t.typeEnv(), qualifiers, members)
	methods := make(map[string]model.SymbolRef)
	for _, included := range incMembers {
		md, ok := included.(*model.MethodDescriptor)
		if !ok || md.MemberKind() == model.InclusionMemberKindResourceMethod {
			continue
		}
		methods[md.MemberName()] = md.MethodRef
	}
	for member := range ty.Members() {
		method, ok := member.(*ast.BMethodDecl)
		if !ok || method.MemberKind() == ast.ObjectMemberKindResourceMethod {
			continue
		}
		methods[method.Name()] = method.Symbol()
	}
	survivingMethods := make(map[string]model.SymbolRef)
	for _, member := range members {
		if member.Kind != semtypes.MemberKindMethod && member.Kind != semtypes.MemberKindRemoteMethod {
			continue
		}
		if ref, ok := methods[member.Name]; ok {
			survivingMethods[member.Name] = ref
		}
	}
	registerMethodTable(t, semType, owner, survivingMethods, ty.GetPosition())
	return semType, chain, true
}

func registerMethodTable(t typeResolver, objectTy semtypes.SemType, owner model.SymbolRef,
	methods map[string]model.SymbolRef, pos diagnostics.Location,
) {
	atom := semtypes.ToObjectAtomicType(t.typeContext(), objectTy)
	if atom == nil {
		t.internalError("failed to get atomic type for object method table", pos)
		return
	}
	t.setObjectMethodTable(atom, model.NewMethodTable(owner, methods))
}

// registerFunctionMethodTable records class and service methods against the object's mapping atom.
// Resource methods are excluded because they use separate path-based metadata.
func registerFunctionMethodTable(t typeResolver, objectTy semtypes.SemType, owner model.SymbolRef,
	methods map[string]*ast.BLangFunction, initFn *ast.BLangFunction, pos diagnostics.Location,
) {
	table := make(map[string]model.SymbolRef, len(methods)+1)
	for name, method := range resolvedMethods(t, methods) {
		if method.IsResource() {
			continue
		}
		table[name] = method.Symbol()
	}
	if initFn != nil && hasResolvedSignature(t, initFn) {
		table["init"] = initFn.Symbol()
	}
	registerMethodTable(t, objectTy, owner, table, pos)
}

func registerClassMethodTable(t typeResolver, objectTy semtypes.SemType, classDef *ast.BLangClassDefinition) {
	registerFunctionMethodTable(t, objectTy, classDef.Symbol(), classDef.Methods, classDef.InitFunction,
		classDef.GetPosition())
}

// directMember represents a member declared directly on a type (not inherited via inclusion).
type directMember struct {
	name       string
	valueTy    semtypes.SemType
	kind       semtypes.MemberKind
	visibility string
	immutable  bool
	pos        diagnostics.Location
}

func validateOverridesAndMerge(t typeResolver, directMembers []directMember, includedMembers map[string][]semtypes.Member, pos diagnostics.Location, isObject bool) ([]semtypes.Member, bool) {
	var members []semtypes.Member
	for _, dm := range directMembers {
		if incMembers, exists := includedMembers[dm.name]; exists {
			for _, incMember := range incMembers {
				if incMember.Kind != dm.kind {
					t.semanticError(
						fmt.Sprintf("member '%s' conflicts with included member of different kind", dm.name),
						dm.pos,
					)
					return nil, false
				}
				if incMember.Visibility != dm.visibility {
					memberKind := "method"
					if dm.kind == semtypes.MemberKindField {
						memberKind = "field"
					}
					t.semanticError(
						fmt.Sprintf("mismatched visibility qualifiers for %s '%s' with object type inclusion", memberKind, dm.name),
						dm.pos,
					)
					return nil, false
				}
				if !semtypes.IsSubtype(t.typeContext(), dm.valueTy, incMember.ValueType) {
					t.semanticError(
						fmt.Sprintf("member '%s' that overrides included member is not a subtype of the included member type", dm.name),
						dm.pos,
					)
					return nil, false
				}
			}
			delete(includedMembers, dm.name)
		}
		members = append(members, semtypes.Member{
			Name:       dm.name,
			ValueType:  dm.valueTy,
			Kind:       dm.kind,
			Visibility: dm.visibility,
			Immutable:  dm.immutable,
		})
	}

	for name, incMembers := range includedMembers {
		if len(incMembers) == 1 {
			if isObject || incMembers[0].Kind == semtypes.MemberKindField {
				members = append(members, incMembers[0])
				continue
			}
			t.semanticError(
				fmt.Sprintf("included method '%s' must be overridden in class definition", name),
				pos,
			)
			return nil, false
		}
		t.semanticError(
			fmt.Sprintf("included member '%s' declared in multiple type inclusions must be overridden", name),
			pos,
		)
		return nil, false
	}

	return members, true
}

func recordFieldDefaults(t typeResolver, recordTy *ast.BLangRecordType) []model.FieldDefault {
	directFields := make(map[string]bool)
	var defaults []model.FieldDefault
	for name, field := range recordTy.FieldPtrs() {
		if semtypes.IsZero(field.Type.Type()) {
			continue
		}
		directFields[name] = true
		if field.Default != nil {
			defaults = append(defaults, model.FieldDefault{FieldName: name, FnRef: field.Default.FnRef})
		}
	}

	inherited := make(map[string]bool)
	for _, ref := range recordTy.Inclusions {
		carrier, ok := t.getSymbol(ref).(model.MemberCarrier)
		if !ok {
			continue
		}
		for _, fieldDefault := range carrier.FieldDefaults() {
			if directFields[fieldDefault.FieldName] || inherited[fieldDefault.FieldName] {
				continue
			}
			inherited[fieldDefault.FieldName] = true
			defaults = append(defaults, fieldDefault)
		}
	}
	return defaults
}

type recordInclusionResolutionResult struct {
	includedFields map[string][]model.FieldDescriptor
	includedRestTy semtypes.SemType
	multpleRestTy  bool
}

func resolveRecordInclusions(t typeResolver, chain *binding, recordTy *ast.BLangRecordType, depth int) (recordInclusionResolutionResult, *binding, bool) {
	// Resolve UDT nodes to set their type. Symbol resolution leaves an inclusion that cannot be included out of
	// recordTy.Inclusions, so the record type is built from the rest.
	for _, inc := range recordTy.TypeInclusions {
		if udt, ok := inc.(*ast.BLangUserDefinedType); !ok || !slices.Contains(recordTy.Inclusions, udt.Symbol()) {
			continue
		}
		if _, _, ok := resolveBType(t, chain, inc, 0); !ok {
			return recordInclusionResolutionResult{}, chain, false
		}
	}

	incMembers, ok := collectIncludedMembers(t, recordTy.Inclusions, recordTy.InclusionPositions, recordTy.GetPosition(), depth)
	if !ok {
		return recordInclusionResolutionResult{}, chain, false
	}

	includedFields := make(map[string][]model.FieldDescriptor)
	var includedRest semtypes.SemType
	needsRestOverride := false
	for _, m := range incMembers {
		switch member := m.(type) {
		case *model.FieldDescriptor:
			includedFields[member.MemberName()] = append(includedFields[member.MemberName()], *member)
		case *model.RestTypeDescriptor:
			restTy := member.MemberType()
			if !semtypes.IsZero(includedRest) {
				needsRestOverride = true
			}
			includedRest = restTy
		}
	}
	return recordInclusionResolutionResult{includedFields, includedRest, needsRestOverride}, chain, true
}

func resolveConstant(t typeResolver, constant *ast.BLangVariable) bool {
	if !semtypes.IsZero(t.symbolType(constant.Symbol())) {
		return true
	}
	if constant.Expr == nil {
		t.internalError("constant expression is nil", constant.GetPosition())
		return false
	}

	var annotationType semtypes.SemType
	if typeNode := constant.TypeNode(); typeNode != nil {
		var ok bool
		annotationType, _, ok = resolveBType(t, nil, typeNode, 0)
		if !ok {
			return false
		}
		if !validConstantType(t, annotationType) {
			t.semanticError(fmt.Sprintf("cannot declare a constant with type '%s', expected a subtype of 'anydata' that is not 'never'",
				semtypes.ToString(t.typeContext(), annotationType)), constant.Name.GetPosition())
			return false
		}
	}

	expr, ok := constant.Expr.(ast.BLangExpression)
	if !ok {
		t.internalError("constant expression is not an expression", constant.GetPosition())
		return false
	}
	if _, ok := resolveActionOrExpression(t, nil, expr, annotationType); !ok {
		return false
	}
	// Type resolution has to stay ahead of folding: the folder reads determined
	// types for casts and list constructors.
	if offender, ok := isConstantExpression(t, expr); !ok {
		t.semanticError(notConstantExpressionMessage, offender.GetPosition())
		return false
	}
	value, ok := foldConstant(t, expr)
	if !ok {
		return false
	}
	if sym, ok := t.getSymbol(constant.Symbol()).(*model.ConstantValueSymbol); ok {
		sym.SetConstantValue(value)
	}

	// The type of a constant is the intersection of readonly and the singleton
	// type containing just the shape of its value (spec §8.8). The evaluated
	// value carries that type, so taking it from there keeps the symbol type and
	// the value in agreement.
	expectedType := values.SemTypeForValue(value)
	symbol := constant.Symbol()
	// A constant's value is deeply immutable, so a reference to it has the readonly
	// intersection of its initializer type. Without this a reference to a structured
	// constant looks mutable to contextual readonly checking even though mutating it
	// panics at runtime.
	t.setSymbolType(symbol, semtypes.Intersect(expectedType, semtypes.ValReadonly))

	return true
}

func validConstantType(t typeResolver, ty semtypes.SemType) bool {
	cx := t.typeContext()
	return !semtypes.IsEmpty(cx, ty) && semtypes.IsSubtype(cx, ty, semtypes.CreateAnydata(cx))
}

func resolveMatchStatement(t typeResolver, chain *binding, stmt *ast.BLangMatchStatement) (statementEffect, bool) {
	exprResult, ok := resolveActionOrExpression(t, chain, stmt.Expr, semtypes.SemType{})
	if !ok {
		return defaultStmtEffect(chain), false
	}
	// The match subject is evaluated for its value, not tested as a condition.
	chain = sequentialChain(t, exprResult.effect)

	exprRef, isVarRef := varRefExp(t, chain, stmt.Expr)
	var remainingType semtypes.SemType
	if isVarRef {
		remainingType = t.symbolType(exprRef)
	} else {
		remainingType = stmt.Expr.Type()
	}
	var bodyEffects []statementEffect

	tyCtx := semtypes.ContextFrom(t.typeEnv())

	for i := range stmt.MatchClauses {
		clause := &stmt.MatchClauses[i]

		if semtypes.IsEmpty(tyCtx, remainingType) {
			t.semanticError("unreachable match clause", clause.GetPosition())
		}

		patternTy, ok := matchClausePatternType(t, chain, clause, remainingType)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		clauseAcceptedType := semtypes.Intersect(remainingType, patternTy)

		clauseIsEmpty := semtypes.IsEmpty(tyCtx, clauseAcceptedType)
		if clauseIsEmpty {
			t.semanticError("unmatchable match clause", clause.GetPosition())
		}

		clause.AcceptedType = clauseAcceptedType

		clauseChain := chain
		if isVarRef && !clauseIsEmpty && canNarrow(t, chain, exprRef) {
			baseRef := t.unnarrowedSymbol(exprRef)
			narrowedSym := narrowSymbol(t, baseRef, clauseAcceptedType)
			clauseChain = &binding{
				ref:            baseRef,
				narrowedSymbol: narrowedSym,
				prev:           chain,
			}
		}

		bodyChain := clauseChain
		if clause.Guard != nil {
			bodyChain, ok = resolveMatchGuard(t, clauseChain, clause.Guard)
			if !ok {
				return defaultStmtEffect(chain), false
			}
		}

		bodyEffect := resolveBlockStatements(t, bodyChain, clause.Body.Stmts)
		if clauseIsEmpty {
			continue
		}
		bodyEffects = append(bodyEffects, bodyEffect)

		if clause.Guard == nil {
			remainingType = semtypes.Diff(remainingType, clause.AcceptedType)
		}
	}

	stmt.IsExhaustive = semtypes.IsEmpty(tyCtx, remainingType)

	// When the match is not exhaustive the incoming chain is the "no clause matched" path and
	// must take part in the merge, otherwise the target keeps a clause's narrowing. An
	// exhaustive match has no such path, so the seed is marked non-completing and discarded.
	result := statementEffect{binding: chain, nonCompletion: stmt.IsExhaustive}
	for _, effect := range bodyEffects {
		result = mergeStatementEffects(t, result, effect)
	}
	return result, true
}

func matchClausePatternType(t typeResolver, chain *binding, clause *ast.BLangMatchClause, remainingType semtypes.SemType) (semtypes.SemType, bool) {
	tyCtx := semtypes.ContextFrom(t.typeEnv())
	acceptedTy := semtypes.Never
	patternRemaining := remainingType
	for i, pattern := range clause.Patterns {
		patternTy, ok := resolveMatchPattern(t, chain, pattern, remainingType)
		if !ok {
			return semtypes.SemType{}, false
		}
		if i > 0 && semtypes.IsEmpty(tyCtx, semtypes.Intersect(patternTy, patternRemaining)) {
			t.semanticError("unmatchable match pattern", pattern.GetPosition())
		}
		patternRemaining = semtypes.Diff(patternRemaining, patternTy)
		acceptedTy = semtypes.Union(acceptedTy, patternTy)
	}
	return acceptedTy, true
}

func resolveMatchGuard(t typeResolver, chain *binding, guard ast.BLangExpression) (*binding, bool) {
	guardResult, ok := resolveActionOrExpression(t, chain, guard, semtypes.Boolean)
	if !ok {
		return nil, false
	}
	return guardResult.effect.ifTrue, true
}

func resolveObjectMemberType(t typeResolver, chain *binding, m ast.ObjectMember, depth int) (semtypes.SemType, *binding, bool) {
	switch m := m.(type) {
	case *ast.BObjectField:
		valueTy, chain, ok := resolveBType(t, chain, m.Ty, depth+1)
		return valueTy, chain, ok
	case *ast.BMethodDecl:
		valueTy, chain, ok := resolveBType(t, chain, &m.BLangFunctionType, depth+1)
		if ok {
			t.setSymbolType(m.Symbol(), valueTy)
			sig, sigOk := functionTypeTypedSignature(t, chain, &m.BLangFunctionType, depth)
			if !sigOk {
				return semtypes.SemType{}, chain, false
			}
			t.getSymbol(m.Symbol()).(model.FunctionSymbol).SetTypedSignature(sig)
		}
		return valueTy, chain, ok
	default:
		return semtypes.SemType{}, chain, false
	}
}

// functionTypeTypedSignature computes the signature of an already resolved function type descriptor. The
// descriptor's flow effects were applied when it was resolved, so the chains these lookups produce are dropped.
func functionTypeTypedSignature(t typeResolver, chain *binding, fnType *ast.BLangFunctionType, depth int) (model.TypedFunctionSignature, bool) {
	paramTypes := make([]semtypes.SemType, len(fnType.RequiredParams))
	for i := range fnType.RequiredParams {
		paramTy, _, ok := resolveBType(t, chain, fnType.RequiredParams[i].TypeDesc, depth+1)
		if !ok {
			return model.TypedFunctionSignature{}, false
		}
		paramTypes[i] = paramTy
	}
	var restType semtypes.SemType
	if fnType.RestParam != nil {
		var ok bool
		if restType, _, ok = resolveBType(t, chain, fnType.RestParam.TypeDesc, depth+1); !ok {
			return model.TypedFunctionSignature{}, false
		}
	}
	returnType := semtypes.Nil
	if fnType.ReturnTypeDescriptor != nil {
		var ok bool
		if returnType, _, ok = resolveBType(t, chain, fnType.ReturnTypeDescriptor, depth+1); !ok {
			return model.TypedFunctionSignature{}, false
		}
	}
	var flags model.FuncSymbolFlags
	if fnType.IsIsolated() {
		flags |= model.FuncSymbolFlagIsolated
	}
	if fnType.IsTransactional() {
		flags |= model.FuncSymbolFlagTransactional
	}
	return model.TypedFunctionSignature{ParamTypes: paramTypes, ReturnType: returnType, RestParamType: restType, Flags: flags}, true
}

func isValidConstPatternExpr(t typeResolver, expr ast.BLangExpression) bool {
	var ref model.SymbolRef
	switch e := expr.(type) {
	case *ast.BLangVarRef:
		ref = e.Symbol()
	case *ast.BLangConstRef:
		ref = e.Symbol()
	default:
		return true
	}
	sym := t.getSymbol(ref)
	return sym != nil && sym.Kind() == model.SymbolKindConstant
}

func resolveMatchPattern(t typeResolver, chain *binding, pattern ast.BLangMatchPattern, expectedTy semtypes.SemType) (semtypes.SemType, bool) {
	switch p := pattern.(type) {
	case *ast.BLangConstPattern:
		tyResult, ok := resolveActionOrExpression(t, chain, p.Expr, expectedTy)
		if !ok {
			return semtypes.SemType{}, false
		}
		ty := tyResult.ty
		if !isValidConstPatternExpr(t, p.Expr) {
			t.semanticError("match pattern variable reference must refer to a constant", p.Expr.GetPosition())
			return semtypes.SemType{}, false
		}
		p.SetAcceptedType(ty)
		return ty, true
	case *ast.BLangWildCardMatchPattern:
		ty := semtypes.Any
		p.SetAcceptedType(ty)
		return ty, true
	default:
		t.internalError(fmt.Sprintf("unexpected match pattern type: %T", pattern), pattern.GetPosition())
		return semtypes.Never, false
	}
}

func semtypeMemberKind(t typeResolver, kind ast.ObjectMemberKind, loc diagnostics.Location) (semtypes.MemberKind, bool) {
	switch kind {
	case ast.ObjectMemberKindField:
		return semtypes.MemberKindField, true
	case ast.ObjectMemberKindMethod:
		return semtypes.MemberKindMethod, true
	case ast.ObjectMemberKindRemoteMethod:
		return semtypes.MemberKindRemoteMethod, true
	case ast.ObjectMemberKindResourceMethod:
		return semtypes.MemberKindResourceMethod, true
	default:
		t.internalError("invalid member kind", loc)
		return 0, false
	}
}

func inclusionMemberKindToSemtype(t typeResolver, kind model.InclusionMemberKind, loc diagnostics.Location) (semtypes.MemberKind, bool) {
	switch kind {
	case model.InclusionMemberKindField:
		return semtypes.MemberKindField, true
	case model.InclusionMemberKindMethod:
		return semtypes.MemberKindMethod, true
	case model.InclusionMemberKindRemoteMethod:
		return semtypes.MemberKindRemoteMethod, true
	case model.InclusionMemberKindResourceMethod:
		return semtypes.MemberKindResourceMethod, true
	default:
		t.internalError("invalid inclusion member kind", loc)
		return 0, false
	}
}

func inclusionMemberToSemtypeMember(t typeResolver, m model.InclusionMember, loc diagnostics.Location) (semtypes.Member, bool) {
	kind := m.MemberKind()
	memberKind, ok := inclusionMemberKindToSemtype(t, kind, loc)
	if !ok {
		return semtypes.Member{}, false
	}
	vis := memberVisibility(t, isPublicInclusionMember(m), "")
	return semtypes.Member{
		Name:       m.MemberName(),
		ValueType:  m.MemberType(),
		Kind:       memberKind,
		Visibility: vis,
		Immutable:  kind != model.InclusionMemberKindField,
	}, true
}

// memberVisibility returns the visibility region of an object member. privateOwner is the name of the class or
// service declaring a private member, and empty for every other member.
func memberVisibility(t typeResolver, isPublic bool, privateOwner string) string {
	if isPublic {
		return semtypes.VisibilityPublic
	}
	pkgID := t.packageID()
	modulePath := pkgID.OrgName.Value() + "/" + pkgID.Name.Value()
	if privateOwner != "" {
		return modulePath + ":" + privateOwner
	}
	return modulePath
}

func privateOwner(isPrivate bool, owner string) string {
	if isPrivate {
		return owner
	}
	return ""
}

// isPublicObjectTypeMember reports whether an object type descriptor member is in the public region. Remote
// methods, resource methods and init are always public.
func isPublicObjectTypeMember(m ast.ObjectMember) bool {
	kind := m.MemberKind()
	return m.IsPublic() || kind == ast.ObjectMemberKindRemoteMethod || kind == ast.ObjectMemberKindResourceMethod ||
		(kind == ast.ObjectMemberKindMethod && m.Name() == "init")
}

// isPublicInclusionMember reports whether an included member is in the public region. Remote methods, resource
// methods and init are always public.
func isPublicInclusionMember(m model.InclusionMember) bool {
	switch member := m.(type) {
	case *model.FieldDescriptor:
		return member.IsPublic()
	case *model.MethodDescriptor:
		kind := member.MemberKind()
		return member.IsPublic() || kind == model.InclusionMemberKindRemoteMethod ||
			kind == model.InclusionMemberKindResourceMethod || member.MemberName() == "init"
	default:
		return true
	}
}

func semtypeNetworkQualifier(t typeResolver, nq ast.ObjectNetworkQuals, loc diagnostics.Location) (semtypes.NetworkQualifier, bool) {
	switch nq {
	case ast.ObjectNetworkQualsNone:
		return semtypes.NetworkQualifierNone, true
	case ast.ObjectNetworkQualsClient:
		return semtypes.NetworkQualifierClient, true
	case ast.ObjectNetworkQualsService:
		return semtypes.NetworkQualifierService, true
	default:
		t.internalError("invalid network qualifier", loc)
		return 0, false
	}
}

func setPositions(pos diagnostics.Location, nodes ...ast.BLangNode) {
	for _, node := range nodes {
		node.SetPosition(pos)
	}
}

// monomorphicOpaqueFn satisfies model.MonomorphicFunctionSymbol: a concrete
// function symbol that carries a backref to its polymorphic opaque origin so
// BIR dispatches to the lang-lib extern.
type monomorphicOpaqueFn struct {
	model.FunctionSymbol
	name string
	poly model.SymbolRef
}

func (m *monomorphicOpaqueFn) Name() string { return m.name }

func (m *monomorphicOpaqueFn) PolymorphicSymbol() model.SymbolRef { return m.poly }

var _ model.MonomorphicFunctionSymbol = &monomorphicOpaqueFn{}

// materializeOpaqueFn builds the monomorphic symbol for sig, adds it to the opaque
// symbol's space, sets its type, and gives it the opaque function's shared untyped
// signature. Caching the result is the opaque package's job, so the cache key stays
// private to the function that chose it.
func materializeOpaqueFn(t typeResolver, sym *model.OpaqueFunctionSymbol,
	polymorphicRef model.SymbolRef, sig model.TypedFunctionSignature,
	loc diagnostics.Location) (model.SymbolRef, bool) {
	mono := &monomorphicOpaqueFn{
		FunctionSymbol: model.NewFunctionSymbol(sym.Name(), sig, true, loc),
		poly:           polymorphicRef,
	}
	mono.SetType(typeFromFunctionSignature(t, sig))
	space := sym.SymbolSpace
	idx := space.AppendSymbol(mono)
	mono.name = fmt.Sprintf("%s$mono$%d", sym.Name(), idx)
	ref := space.RefAt(idx)
	sigRef, ok := t.functionSignatureRef(polymorphicRef)
	if !ok {
		t.internalError("opaque function has no untyped signature", loc)
		return model.SymbolRef{}, false
	}
	if !t.associateFunctionSignature(ref, sigRef) {
		t.internalError("function signature already set", loc)
		return model.SymbolRef{}, false
	}
	return ref, true
}

func resolveXMLFilterExpr(t typeResolver, chain *binding, expr *ast.BLangXMLFilterExpression, _ semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	receiver, ok := resolveActionOrExpression(t, chain, expr.Expression, semtypes.XML)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultType := semtypes.XMLSequence(semtypes.XMLElement)
	setNodeType(t, expr, resultType)
	return resultType, defaultExpressionEffect(sequentialChain(t, receiver.effect)), true
}

func resolveXMLStepExpr(t typeResolver, chain *binding, expr *ast.BLangXMLStepExpression, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if expr.LoweredExpression == nil {
		lowered, ok := lowerXMLStepExpr(t, expr)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		// This is the one AST write allowed while the resolver is ephemeral. The lowering does not depend on the
		// candidate being tried, so the first resolution to reach the step makes it for all of them, and it is
		// stored before resolving so a trial that fails on the step still leaves it for the next one.
		expr.LoweredExpression = lowered
	}
	result, ok := resolveActionOrExpression(t, chain, expr.LoweredExpression, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setNodeType(t, expr, result.ty)
	return result.ty, result.effect, true
}

func lowerXMLStepExpr(t typeResolver, expr *ast.BLangXMLStepExpression) (ast.BLangExpression, bool) {
	pos := expr.GetPosition()
	fnName := t.nextXMLStepFnName()
	fnIdent := xmlStepIdentifier(fnName, pos)
	paramName := "$xmlStepItem"
	paramIdent := xmlStepIdentifier(paramName, pos)
	param := ast.BLangVariable{}
	param.SetName(paramIdent)
	param.SetPosition(pos)
	param.SetRequiredParam()

	fnScope := t.compilerContext().NewFunctionScope(t.currentScope(), *t.packageID())
	paramSymbol := model.NewVariableSymbol(paramName, false, false, true, pos)
	fnScope.AddSymbol(paramName, &paramSymbol)
	paramRef, _ := fnScope.GetSymbol(paramName)
	param.SetSymbol(paramRef)
	itemRef := &ast.BLangVarRef{VariableName: xmlStepIdentifier(paramName, pos)}
	itemRef.SetSymbol(paramRef)
	itemRef.SetPosition(pos)

	var current ast.BLangExpression
	var ok bool
	switch expr.Start.Kind {
	case ast.XMLStepStartKindAllChildren:
		current, ok = newDirectLangXMLInvocation(t, "getChildren", []ast.BLangExpression{itemRef}, pos)
	case ast.XMLStepStartKindElementChildren:
		current, ok = newDirectLangXMLInvocation(t, "getChildren", []ast.BLangExpression{itemRef}, pos)
		if ok {
			current = newXMLStepFilter(current, expr.Start.NamePattern, expr.Start.GetPosition())
		}
	case ast.XMLStepStartKindElementDescendants:
		current, ok = newDirectLangXMLInvocation(t, "getDescendants", []ast.BLangExpression{itemRef}, pos)
		if ok {
			current = newXMLStepFilter(current, expr.Start.NamePattern, expr.Start.GetPosition())
		}
	default:
		t.internalError("unsupported XML step start kind", expr.Start.GetPosition())
		return nil, false
	}
	if !ok {
		return nil, false
	}

	for _, extension := range expr.Extensions {
		switch extension := extension.(type) {
		case *ast.BLangXMLStepFilterExtend:
			current = newXMLStepFilter(current, extension.NamePattern, extension.GetPosition())
		case *ast.BLangXMLStepIndexExtend:
			current, ok = newDirectLangXMLInternalInvocation(t, "$stepIndex", []ast.BLangExpression{current, extension.Expression}, extension.GetPosition())
		case *ast.BLangXMLStepMethodCallExtend:
			invocation := &ast.BLangInvocation{}
			invocation.Expr = current
			invocation.PkgAlias = cloneXMLStepIdentifier(extension.Invocation.PkgAlias)
			invocation.Name = cloneXMLStepIdentifier(extension.Invocation.Name)
			invocation.ArgExprs = append([]ast.BLangExpression(nil), extension.Invocation.ArgExprs...)
			invocation.SetPosition(extension.Invocation.GetPosition())
			invocation.SetRawSymbol(common.NewDeferredMethodSymbol(invocation.Name.GetValue(), fnScope.MainSpace()))
			current = invocation
		default:
			t.internalError(fmt.Sprintf("unsupported XML step extension type: %T", extension), extension.GetPosition())
			return nil, false
		}
		if !ok {
			return nil, false
		}
	}

	body := &ast.BLangExprFunctionBody{Expr: current}
	body.SetPosition(pos)
	fn := ast.NewBLangFunction(ast.InvokableData{
		Position:       pos,
		Name:           fnIdent,
		RequiredParams: []ast.BLangVariable{param},
		Body:           body,
		Flags:          model.FlagLambda | model.FlagAnonymous,
	})
	fn.SetScope(fnScope)
	ownerSpace := t.currentScope().(model.SymbolSpaceProvider).MainSpace()
	fnRef := t.createFunctionSymbol(ownerSpace, fnName, model.TypedFunctionSignature{}, semtypes.SemType{})
	fn.SetSymbol(fnRef)
	signatureRef := t.allocateFunctionSignature([]model.Param{{Name: paramName}}, false)
	if !t.associateFunctionSignature(fnRef, signatureRef) {
		t.internalError("function signature already set", pos)
		return nil, false
	}
	lambda := &ast.BLangLambdaFunction{Function: fn}
	lambda.SetInferredParams()
	lambda.SetPosition(pos)

	nilName := ast.NewBLangLiteral(pos, ast.LiteralKindNil, nil, "()", true)
	elements, ok := newDirectLangXMLInvocation(t, "elements", []ast.BLangExpression{expr.Expression, nilName}, pos)
	if !ok {
		return nil, false
	}
	return newDirectLangXMLInvocation(t, "map", []ast.BLangExpression{elements, lambda}, pos)
}

func xmlStepIdentifier(value string, pos diagnostics.Location) *ast.BLangIdentifier {
	ident := &ast.BLangIdentifier{Value: value, OriginalValue: value}
	ident.SetPosition(pos)
	return ident
}

func cloneXMLStepIdentifier(ident ast.IdentifierNode) ast.IdentifierNode {
	if ident == nil {
		return nil
	}
	return xmlStepIdentifier(ident.GetValue(), ident.GetPosition())
}

func newXMLStepFilter(expression ast.BLangExpression, patterns []ast.BLangAtomicNamePattern, pos diagnostics.Location) *ast.BLangXMLFilterExpression {
	filter := &ast.BLangXMLFilterExpression{Expression: expression, NamePattern: patterns}
	filter.SetPosition(pos)
	return filter
}

func newDirectLangXMLInvocation(t typeResolver, name string, args []ast.BLangExpression, pos diagnostics.Location) (*ast.BLangInvocation, bool) {
	space, pkgAlias, ok := ensureLangLibImport(t, "lang.xml", pos)
	if !ok {
		return nil, false
	}
	ref, ok := space.GetSymbol(name)
	if !ok {
		t.internalError("lang.xml symbol not found: "+name, pos)
		return nil, false
	}
	return newResolvedXMLInvocation(ref, &pkgAlias, name, args, pos), true
}

// newDirectLangXMLInternalInvocation builds a call to a lang.xml symbol that is
// not visible to Ballerina source, so it looks the symbol up without the
// visibility check GetSymbol applies.
func newDirectLangXMLInternalInvocation(t typeResolver, name string, args []ast.BLangExpression, pos diagnostics.Location) (*ast.BLangInvocation, bool) {
	space, pkgAlias, ok := ensureLangLibImport(t, "lang.xml", pos)
	if !ok {
		return nil, false
	}
	ref, ok := space.GetInternalSymbol(name)
	if !ok {
		t.internalError("lang.xml internal symbol not found: "+name, pos)
		return nil, false
	}
	return newResolvedXMLInvocation(ref, &pkgAlias, name, args, pos), true
}

func newResolvedXMLInvocation(ref model.SymbolRef, pkgAlias ast.IdentifierNode, name string, args []ast.BLangExpression, pos diagnostics.Location) *ast.BLangInvocation {
	invocation := &ast.BLangInvocation{}
	invocation.PkgAlias = pkgAlias
	invocation.Name = xmlStepIdentifier(name, pos)
	invocation.ArgExprs = args
	invocation.SetSymbol(ref)
	invocation.SetPosition(pos)
	return invocation
}
