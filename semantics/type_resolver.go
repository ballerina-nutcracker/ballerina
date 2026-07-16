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
	"errors"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"

	"ballerina-lang-go/ast"
	balCommon "ballerina-lang-go/common"
	"ballerina-lang-go/context"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/values"
)

type distinctTypeSymbol interface {
	DistinctTypeIDs() []int
	SetDistinctTypeIDs([]int)
}

type typeResolver interface {
	typeContext() semtypes.Context
	expectedReturnType() semtypes.SemType
	parent() typeResolver
	typeEnv() semtypes.Env

	// Error reporting (proxied from CompilerContext)
	semanticError(message string, loc diagnostics.Location)
	internalError(message string, loc diagnostics.Location)
	unimplemented(message string, loc diagnostics.Location)
	syntaxError(message string, loc diagnostics.Location)

	// Symbol management (proxied from CompilerContext)
	symbolType(ref model.SymbolRef) semtypes.SemType
	setSymbolType(ref model.SymbolRef, ty semtypes.SemType)
	getSymbol(ref model.SymbolRef) model.Symbol
	unnarrowedSymbol(ref model.SymbolRef) model.SymbolRef
	symbolName(ref model.SymbolRef) string
	createNarrowedSymbol(ref model.SymbolRef) model.SymbolRef
	createFunctionSymbol(space *model.SymbolSpace, name string, sig model.FunctionSignature, fnTy semtypes.SemType) model.SymbolRef
	compilerContext() *context.CompilerContext

	// Import management
	lookupImportedSymbols(pkgName string) (model.ExportedSymbolSpace, bool)
	addImplicitImport(pkgName string, imp ast.BLangImportPackage)
	hasImplicitImport(pkgName string) bool

	// Closure capture tracking
	trackCapturedVar(ref model.SymbolRef)
	getCapturedVars() map[model.SymbolRef]bool
	setCapturedVars(vars map[model.SymbolRef]bool)

	ensureResolved(ref model.SymbolRef, depth int) bool

	setMappingAtomBType(mat *semtypes.MappingAtomicType, bType ast.BType)
	getMappingAtomBType(mat *semtypes.MappingAtomicType) (ast.BType, bool)

	setMappingAtomSymRef(mat *semtypes.MappingAtomicType, ref model.SymbolRef)
	getMappingAtomSymRef(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool)
	setClassAtomSymbol(mat *semtypes.MappingAtomicType, symbol model.SymbolRef)
	getClassAtomSymbol(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool)
	currentScope() model.Scope
	setCurrentScope(scope model.Scope)
	nextDefaultFnName() string
	nextMonoFnName(origName string) string

	lookupClassMethodSymbol(receiverTy semtypes.SemType, methodName string) (model.SymbolRef, bool)

	ensureNotEmpty(ty semtypes.SemType, onEmpty func()) bool
	xmlIteratorTypeCache() *semtypes.SemTypeCache
}

// deferredEmptinessCheck is an emptiness check that was registered while the
// type env still had unset recursive atoms. It runs once the env is ready.
type deferredEmptinessCheck struct {
	ty      semtypes.SemType
	onEmpty func()
}

// resolutionStatus tracks lazy resolution progress for cycle detection.
type resolutionStatus int

const (
	resolutionPending resolutionStatus = iota
	resolutionInProgress
	resolutionDone
)

type packageTypeResolver struct {
	ctx             *context.CompilerContext
	tyCtx           semtypes.Context
	importedSymbols map[string]model.ExportedSymbolSpace
	pkg             *ast.BLangPackage
	implicitImports map[string]ast.BLangImportPackage
	// capturedNarrowedVars tracks base symbols of narrowed variables captured across
	// a function boundary during lambda body resolution. nil when not inside a lambda.
	capturedNarrowedVars map[model.SymbolRef]bool

	// packageConstants maps a constant's symbol ref to its AST node.
	packageConstants map[model.SymbolRef]*ast.BLangConstant
	// inferredGlobalVarNodes holds module-level vars **without** a type
	// annotation. Their type comes from their initializer expression, so they
	// must be resolved lazily (driven by ensureResolved) the same way
	// constants are.
	inferredGlobalVarNodes map[model.SymbolRef]*ast.BLangSimpleVariable
	// lazyResolutionStatus tracks per-symbol resolution progress (for both
	// constants and inferred-typed module-level vars) for cycle detection.
	// Absence means resolution has not started.
	lazyResolutionStatus  map[model.SymbolRef]resolutionStatus
	functionNodes         map[model.SymbolRef]*ast.BLangFunction
	mappingAtomToBType    map[*semtypes.MappingAtomicType]ast.BType
	typeDefnNodes         map[model.SymbolRef]*ast.BLangTypeDefinition
	classDefnNodes        map[model.SymbolRef]*ast.BLangClassDefinition
	defaultFnSymbolCount  int
	monoCounters          map[string]int
	annotationGlobalCount int
	scope                 model.Scope
	mappingAtomToSymRef   map[*semtypes.MappingAtomicType]model.SymbolRef
	classAtomSymbols      map[*semtypes.MappingAtomicType]model.SymbolRef
	classSymbolByType     map[semtypes.InternHandle]model.SymbolRef
	semtypeInterner       *semtypes.SemtypeInterner
	xmlIteratorTypes      *semtypes.SemTypeCache

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
func (t *packageTypeResolver) xmlIteratorTypeCache() *semtypes.SemTypeCache {
	return t.xmlIteratorTypes
}

func (t *packageTypeResolver) semanticError(msg string, loc diagnostics.Location) {
	t.ctx.SemanticError(msg, loc)
}

func (t *packageTypeResolver) internalError(msg string, loc diagnostics.Location) {
	t.ctx.InternalError(msg, loc)
}

func (t *packageTypeResolver) unimplemented(msg string, loc diagnostics.Location) {
	t.ctx.Unimplemented(msg, loc)
}

func (t *packageTypeResolver) syntaxError(msg string, loc diagnostics.Location) {
	t.ctx.SyntaxError(msg, loc)
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

func (t *packageTypeResolver) createFunctionSymbol(space *model.SymbolSpace, name string, sig model.FunctionSignature, fnTy semtypes.SemType) model.SymbolRef {
	return t.ctx.CreateFunctionSymbol(space, name, sig, fnTy)
}

func (t *packageTypeResolver) compilerContext() *context.CompilerContext {
	return t.ctx
}

func (t *packageTypeResolver) setMappingAtomBType(mat *semtypes.MappingAtomicType, bType ast.BType) {
	t.mappingAtomToBType[mat] = bType
}

func (t *packageTypeResolver) getMappingAtomBType(mat *semtypes.MappingAtomicType) (ast.BType, bool) {
	bType, ok := t.mappingAtomToBType[mat]
	return bType, ok
}

func (t *packageTypeResolver) setMappingAtomSymRef(mat *semtypes.MappingAtomicType, ref model.SymbolRef) {
	t.mappingAtomToSymRef[mat] = ref
}

func (t *packageTypeResolver) getMappingAtomSymRef(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool) {
	ref, ok := t.mappingAtomToSymRef[mat]
	return ref, ok
}

func (t *packageTypeResolver) setClassAtomSymbol(mat *semtypes.MappingAtomicType, symbol model.SymbolRef) {
	t.classAtomSymbols[mat] = symbol
}

func (t *packageTypeResolver) getClassAtomSymbol(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool) {
	sym, ok := t.classAtomSymbols[mat]
	return sym, ok
}

func (t *packageTypeResolver) currentScope() model.Scope     { return t.scope }
func (t *packageTypeResolver) setCurrentScope(s model.Scope) { t.scope = s }

func (t *packageTypeResolver) nextDefaultFnName() string {
	name := fmt.Sprintf("$desugar$%d", t.defaultFnSymbolCount)
	t.defaultFnSymbolCount++
	return name
}

func (t *packageTypeResolver) nextMonoFnName(origName string) string {
	idx := t.monoCounters[origName]
	t.monoCounters[origName] = idx + 1
	return fmt.Sprintf("$mono$%s$%d", origName, idx)
}

func (t *packageTypeResolver) lookupClassMethodSymbol(receiverTy semtypes.SemType, methodName string) (model.SymbolRef, bool) {
	handle, ok := t.semtypeInterner.Lookup(receiverTy)
	if !ok {
		return model.SymbolRef{}, false
	}
	classRef, ok := t.classSymbolByType[handle]
	if !ok {
		return model.SymbolRef{}, false
	}
	classSym, ok := t.getSymbol(classRef).(model.ClassSymbol)
	if !ok {
		return model.SymbolRef{}, false
	}
	return classSym.MethodSymbol(methodName)
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

func (t *packageTypeResolver) trackCapturedVar(ref model.SymbolRef) {
	if t.capturedNarrowedVars != nil {
		t.capturedNarrowedVars[ref] = true
	}
}

func (t *packageTypeResolver) getCapturedVars() map[model.SymbolRef]bool {
	return t.capturedNarrowedVars
}

func (t *packageTypeResolver) setCapturedVars(vars map[model.SymbolRef]bool) {
	t.capturedNarrowedVars = vars
}

type functionTypeResolver struct {
	parentResolver       typeResolver
	tyCtx                semtypes.Context
	retTy                semtypes.SemType
	implicitImports      map[string]ast.BLangImportPackage
	capturedNarrowedVars map[model.SymbolRef]bool
	mappingAtomToBType   map[*semtypes.MappingAtomicType]ast.BType
	monoCounters         map[string]int
	defaultFnSymbolCount int
	scope                model.Scope
	mappingAtomToSymRef  map[*semtypes.MappingAtomicType]model.SymbolRef
}

func (f *functionTypeResolver) typeContext() semtypes.Context        { return f.tyCtx }
func (f *functionTypeResolver) expectedReturnType() semtypes.SemType { return f.retTy }
func (f *functionTypeResolver) parent() typeResolver                 { return f.parentResolver }
func (f *functionTypeResolver) typeEnv() semtypes.Env                { return f.parentResolver.typeEnv() }
func (f *functionTypeResolver) xmlIteratorTypeCache() *semtypes.SemTypeCache {
	return f.parentResolver.xmlIteratorTypeCache()
}

func (f *functionTypeResolver) semanticError(msg string, loc diagnostics.Location) {
	f.parentResolver.semanticError(msg, loc)
}

func (f *functionTypeResolver) internalError(msg string, loc diagnostics.Location) {
	f.parentResolver.internalError(msg, loc)
}

func (f *functionTypeResolver) unimplemented(msg string, loc diagnostics.Location) {
	f.parentResolver.unimplemented(msg, loc)
}

func (f *functionTypeResolver) syntaxError(msg string, loc diagnostics.Location) {
	f.parentResolver.syntaxError(msg, loc)
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

func (f *functionTypeResolver) createFunctionSymbol(space *model.SymbolSpace, name string, sig model.FunctionSignature, fnTy semtypes.SemType) model.SymbolRef {
	return f.parentResolver.createFunctionSymbol(space, name, sig, fnTy)
}

func (f *functionTypeResolver) compilerContext() *context.CompilerContext {
	return f.parentResolver.compilerContext()
}

func (f *functionTypeResolver) lookupClassMethodSymbol(receiverTy semtypes.SemType, methodName string) (model.SymbolRef, bool) {
	return f.parentResolver.lookupClassMethodSymbol(receiverTy, methodName)
}

func (f *functionTypeResolver) ensureNotEmpty(ty semtypes.SemType, onEmpty func()) bool {
	return f.parentResolver.ensureNotEmpty(ty, onEmpty)
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

func (f *functionTypeResolver) trackCapturedVar(ref model.SymbolRef) {
	if f.capturedNarrowedVars != nil {
		f.capturedNarrowedVars[ref] = true
	}
}

func (f *functionTypeResolver) getCapturedVars() map[model.SymbolRef]bool {
	return f.capturedNarrowedVars
}

func (f *functionTypeResolver) setCapturedVars(vars map[model.SymbolRef]bool) {
	f.capturedNarrowedVars = vars
}

func (f *functionTypeResolver) ensureResolved(ref model.SymbolRef, depth int) bool {
	return f.parentResolver.ensureResolved(ref, depth)
}

func (f *functionTypeResolver) setMappingAtomBType(mat *semtypes.MappingAtomicType, bType ast.BType) {
	f.mappingAtomToBType[mat] = bType
}

func (f *functionTypeResolver) getMappingAtomBType(mat *semtypes.MappingAtomicType) (ast.BType, bool) {
	if bType, ok := f.mappingAtomToBType[mat]; ok {
		return bType, true
	}
	return f.parentResolver.getMappingAtomBType(mat)
}

func (f *functionTypeResolver) setMappingAtomSymRef(mat *semtypes.MappingAtomicType, ref model.SymbolRef) {
	f.mappingAtomToSymRef[mat] = ref
}

func (f *functionTypeResolver) getMappingAtomSymRef(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool) {
	if ref, ok := f.mappingAtomToSymRef[mat]; ok {
		return ref, ok
	}
	return f.parentResolver.getMappingAtomSymRef(mat)
}

func (f *functionTypeResolver) setClassAtomSymbol(mat *semtypes.MappingAtomicType, symbol model.SymbolRef) {
	f.parentResolver.setClassAtomSymbol(mat, symbol)
}

func (f *functionTypeResolver) getClassAtomSymbol(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool) {
	return f.parentResolver.getClassAtomSymbol(mat)
}

func (f *functionTypeResolver) currentScope() model.Scope     { return f.scope }
func (f *functionTypeResolver) setCurrentScope(s model.Scope) { f.scope = s }

func (f *functionTypeResolver) nextDefaultFnName() string {
	name := fmt.Sprintf("$desugar$%d", f.defaultFnSymbolCount)
	f.defaultFnSymbolCount++
	return name
}

func (f *functionTypeResolver) nextMonoFnName(origName string) string {
	idx := f.monoCounters[origName]
	f.monoCounters[origName] = idx + 1
	return fmt.Sprintf("$mono$%s$%d", origName, idx)
}

func newPackageTypeResolver(ctx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace, moduleScope model.Scope) *packageTypeResolver {
	return &packageTypeResolver{
		ctx:                    ctx,
		tyCtx:                  semtypes.ContextFrom(ctx.GetTypeEnv()),
		importedSymbols:        importedSymbols,
		pkg:                    pkg,
		implicitImports:        make(map[string]ast.BLangImportPackage),
		packageConstants:       make(map[model.SymbolRef]*ast.BLangConstant),
		inferredGlobalVarNodes: make(map[model.SymbolRef]*ast.BLangSimpleVariable),
		lazyResolutionStatus:   make(map[model.SymbolRef]resolutionStatus),
		functionNodes:          make(map[model.SymbolRef]*ast.BLangFunction),
		mappingAtomToBType:     make(map[*semtypes.MappingAtomicType]ast.BType),
		typeDefnNodes:          make(map[model.SymbolRef]*ast.BLangTypeDefinition),
		classDefnNodes:         make(map[model.SymbolRef]*ast.BLangClassDefinition),
		mappingAtomToSymRef:    make(map[*semtypes.MappingAtomicType]model.SymbolRef),
		classAtomSymbols:       make(map[*semtypes.MappingAtomicType]model.SymbolRef),
		classSymbolByType:      make(map[semtypes.InternHandle]model.SymbolRef),
		semtypeInterner:        semtypes.NewSemtypeInterner(),
		xmlIteratorTypes:       semtypes.NewSemTypeCache(),
		monoCounters:           make(map[string]int),
		scope:                  moduleScope,
	}
}

func populateClassSymbolByType(t *packageTypeResolver, pkg *ast.BLangPackage) {
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		if ty := t.symbolType(classDef.Symbol()); !semtypes.IsZero(ty) {
			t.classSymbolByType[t.semtypeInterner.Intern(ty)] = classDef.Symbol()
		}
	}

	for _, importedSpace := range t.importedSymbols {
		for ref := range importedSpace.PublicMainSymbols() {
			if t.ctx.SymbolIsClass(ref) {
				if ty := t.ctx.SymbolType(ref); !semtypes.IsZero(ty) {
					t.classSymbolByType[t.semtypeInterner.Intern(ty)] = ref
				}
			}
		}
	}
}

func (t *packageTypeResolver) ensureResolved(ref model.SymbolRef, depth int) bool {
	if !semtypes.IsZero(t.symbolType(ref)) {
		return true
	}
	if defn, ok := t.typeDefnNodes[ref]; ok {
		_, ok := resolveTypeDefinition(t, defn, depth)
		return ok
	}
	if classDef, ok := t.classDefnNodes[ref]; ok {
		_, ok := resolveClassTypeDefinition(t, classDef, depth)
		return ok
	}
	if c, inMap := t.packageConstants[ref]; inMap {
		switch t.lazyResolutionStatus[ref] {
		case resolutionDone:
			return true
		case resolutionInProgress:
			var pos diagnostics.Location
			if c.Name != nil {
				pos = c.Name.GetPosition()
			}
			t.semanticError(fmt.Sprintf("invalid cycle detected for %s", t.symbolName(ref)), pos)
			return false
		default:
			t.lazyResolutionStatus[ref] = resolutionInProgress
			ok := resolveConstant(t, c)
			t.lazyResolutionStatus[ref] = resolutionDone
			return ok
		}
	}
	if gv, inMap := t.inferredGlobalVarNodes[ref]; inMap {
		switch t.lazyResolutionStatus[ref] {
		case resolutionDone:
			return true
		case resolutionInProgress:
			var pos diagnostics.Location
			if gv.Name != nil {
				pos = gv.Name.GetPosition()
			}
			t.semanticError(fmt.Sprintf("invalid cycle detected for %s", t.symbolName(ref)), pos)
			return false
		default:
			t.lazyResolutionStatus[ref] = resolutionInProgress
			ok := resolveSimpleVariable(t, nil, gv)
			t.lazyResolutionStatus[ref] = resolutionDone
			return ok
		}
	}
	if fn, ok := t.functionNodes[ref]; ok {
		_, ok := resolveFunctionSignature(t, fn)
		return ok
	}
	return true
}

// ResolveTopLevelNodes resolves type definitions, function signatures, and constants.
// After this (for the given package) all the semtypes are known. This means after resolving types of all the packages
// it is safe to use the closed world assumption to optimize type checks.
func ResolveTopLevelNodes(ctx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) {
	t := newPackageTypeResolver(ctx, pkg, importedSymbols, pkg.Scope)
	t.resolveTopLevelTypes(pkg)
}

func populateMappingAtomMaps(t typeResolver, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) {
	for i := range pkg.TypeDefinitions {
		defn := &pkg.TypeDefinitions[i]
		semType := t.symbolType(defn.Symbol())
		if _, ok := defn.GetTypeData().TypeDescriptor.(*ast.BLangRecordType); ok {
			mat := semtypes.ToMappingAtomicType(t.typeContext(), semType)
			if mat == nil {
				t.internalError("failed to extract mapping atomic type for record type", defn.GetPosition())
			}
			t.setMappingAtomSymRef(mat, defn.Symbol())
		}
	}
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		semType := t.symbolType(classDef.Symbol())
		mat := semtypes.ToObjectAtomicType(t.typeContext(), semType)
		t.setClassAtomSymbol(mat, classDef.Symbol())
	}

	for _, symbolSpace := range importedSymbols {
		for ref := range symbolSpace.PublicMainSymbols() {
			if t.compilerContext().SymbolKind(ref) != model.SymbolKindType {
				continue
			}
			semType := t.symbolType(ref)
			if semtypes.IsZero(semType) {
				continue
			}
			mat := semtypes.ToMappingAtomicType(t.typeContext(), semType)
			if mat != nil {
				t.setMappingAtomSymRef(mat, ref)
			}
			// Only real class symbols may back a `new` expression. Mapping any
			// type alias whose semtype merely contains an object atom (e.g. a
			// union like `RequestMessage = json|Request`) would clobber the
			// genuine class's atom mapping, so restrict this to ClassSymbols.
			if t.compilerContext().SymbolIsClass(ref) {
				if oat := semtypes.ToObjectAtomicType(t.typeContext(), semType); oat != nil {
					t.setClassAtomSymbol(oat, ref)
				}
			}
		}
	}
}

// ResolveLocalNodes resolves the types of function bodies and remaining inner nodes.
func ResolveLocalNodes(ctx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) {
	p := newPackageTypeResolver(ctx, pkg, importedSymbols, pkg.Scope)
	populateClassSymbolByType(p, pkg)
	populateMappingAtomMaps(p, pkg, importedSymbols)
	fns := packageFunctionDecls(pkg)

	allImports := make(map[string]ast.BLangImportPackage)
	resolveFieldInitsInScope := func(scope model.Scope, fields []ast.SimpleVariableNode) {
		ft := &functionTypeResolver{
			parentResolver:      p,
			tyCtx:               semtypes.ContextFrom(p.typeEnv()),
			implicitImports:     make(map[string]ast.BLangImportPackage),
			mappingAtomToBType:  make(map[*semtypes.MappingAtomicType]ast.BType),
			monoCounters:        make(map[string]int),
			scope:               scope,
			mappingAtomToSymRef: make(map[*semtypes.MappingAtomicType]model.SymbolRef),
		}
		for _, fieldNode := range fields {
			field := fieldNode.(*ast.BLangSimpleVariable)
			if field.Expr != nil {
				resolveActionOrExpression(ft, nil, field.Expr.(ast.BLangExpression), field.GetDeterminedType())
			}
		}
		maps.Copy(allImports, ft.implicitImports)
	}
	for i := range pkg.ClassDefinitions {
		c := &pkg.ClassDefinitions[i]
		resolveFieldInitsInScope(c.Scope(), c.Fields)
	}
	for i := range pkg.Services {
		s := &pkg.Services[i]
		resolveFieldInitsInScope(s.Scope(), s.Fields)
	}

	resolvers := make([]*functionTypeResolver, len(fns))
	var wg sync.WaitGroup
	for i, fn := range fns {
		wg.Add(1)
		go func(idx int, f functionDecl) {
			defer wg.Done()
			resolvers[idx] = resolveFunctionBody(p, f)
		}(i, fn)
	}
	wg.Wait()

	for _, t := range resolvers {
		maps.Copy(allImports, t.implicitImports)
	}
	importNames := make([]string, 0, len(allImports))
	for name := range allImports {
		importNames = append(importNames, name)
	}
	sort.Strings(importNames)
	for _, name := range importNames {
		pkg.Imports = append(pkg.Imports, allImports[name])
	}
}

func isPolymorphicFnSymbol(sym model.FunctionSymbol) bool {
	switch sym.(type) {
	case model.DependentlyTypedFunctionSymbol:
		return true
	default:
		return false
	}
}

type functionDecl interface {
	ast.InvokableNode
	ast.BNodeWithSymbol
	Scope() model.Scope
	IsIsolated() bool
	IsTransactional() bool
	FuncSymbolFlags() model.FuncSymbolFlags
}

// packageFunctionDecls returns every function-like declaration in pkg whose
// body needs processing: top-level functions, the package init function,
// and per-class init functions, methods, and resource methods.
func packageFunctionDecls(pkg *ast.BLangPackage) []functionDecl {
	var fns []functionDecl
	for i := range pkg.Functions {
		fns = append(fns, &pkg.Functions[i])
	}
	if pkg.InitFunction != nil {
		fns = append(fns, pkg.InitFunction)
	}
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		fns = appendClassBodyMethodDecls(fns, classDef.InitFunction, classDef.Methods, classDef.ResourceMethods)
	}
	for i := range pkg.Services {
		s := &pkg.Services[i]
		fns = appendClassBodyMethodDecls(fns, s.InitFunction, s.Methods, s.ResourceMethods)
	}
	return fns
}

func appendClassBodyMethodDecls(fns []functionDecl, initFn *ast.BLangFunction, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod) []functionDecl {
	if initFn != nil {
		fns = append(fns, initFn)
	}
	for name := range methods {
		fns = append(fns, methods[name])
	}
	for _, rm := range resourceMethods {
		fns = append(fns, rm)
	}
	return fns
}

type constantDepCollector struct {
	t       typeResolver
	nodeSet map[model.SymbolRef]int
	deps    map[int]struct{}
}

func (c *constantDepCollector) depends(ref model.SymbolRef) {
	unnarrowed := c.t.unnarrowedSymbol(ref)
	if idx, ok := c.nodeSet[unnarrowed]; ok {
		c.deps[idx] = struct{}{}
	}
}

func (c *constantDepCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangSimpleVarRef:
		c.depends(n.Symbol())
	case *ast.BLangConstRef:
		c.depends(n.Symbol())
	}
	return c
}

func (c *constantDepCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

func resolvePackageConstants(t *packageTypeResolver, pkg *ast.BLangPackage) bool {
	order, ok := topologicallySortConstants(t, pkg.Constants)
	if !ok {
		return false
	}
	for _, idx := range order {
		constant := &pkg.Constants[idx]
		ref := constant.Symbol()
		if t.lazyResolutionStatus[ref] == resolutionDone {
			continue
		}
		t.lazyResolutionStatus[ref] = resolutionInProgress
		ok := resolveConstant(t, constant)
		t.lazyResolutionStatus[ref] = resolutionDone
		if !ok {
			return false
		}
	}
	return true
}

func topologicallySortConstants(t typeResolver, constants []ast.BLangConstant) ([]int, bool) {
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
		v := &constantDepCollector{
			t:       t,
			nodeSet: nodeSet,
			deps:    make(map[int]struct{}),
		}
		ast.Walk(v, expr)
		for d := range v.deps {
			deps[i] = append(deps[i], d)
		}
		sort.Ints(deps[i])
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

func resolveInvokableSignature(t typeResolver, fn functionDecl, fnSym model.FunctionSymbol, requiredParams []ast.BLangSimpleVariable) (semtypes.SemType, []semtypes.SemType, semtypes.SemType, semtypes.SemType, bool) {
	paramTypes := make([]semtypes.SemType, len(requiredParams))
	for i := range requiredParams {
		resolveSimpleVariable(t, nil, &requiredParams[i])
		setOtherNodesAsNever(&requiredParams[i])
		paramTypes[i] = requiredParams[i].GetDeterminedType()
	}
	restTy := semtypes.NEVER
	if rp := fn.GetRestParam(); rp != nil {
		restParam := rp.(*ast.BLangSimpleVariable)
		resolveSimpleVariable(t, nil, restParam)
		setOtherNodesAsNever(restParam)
		elementType := restParam.GetDeterminedType()
		restTy = elementType
		listDefn := semtypes.NewListDefinition()
		restParamListTy := listDefn.DefineListTypeWrapped(t.typeEnv(), []semtypes.SemType{}, 0, elementType, semtypes.CellMutability_CELL_MUT_NONE)
		restParam.SetDeterminedType(restParamListTy)
		updateSymbolType(t, restParam, restParamListTy)
	}
	paramListDefn := semtypes.NewListDefinition()
	paramListTy := paramListDefn.DefineListTypeWrapped(t.typeEnv(), paramTypes, len(paramTypes), restTy, semtypes.CellMutability_CELL_MUT_NONE)
	var returnTy semtypes.SemType
	if retTd := fn.GetReturnTypeDescriptor(); retTd != nil {
		var ok bool
		returnTy, ok = resolveBType(t, retTd, 0)
		if !ok {
			return semtypes.SemType{}, nil, semtypes.SemType{}, semtypes.SemType{}, false
		}
		setOtherNodesAsNever(retTd)
	} else {
		returnTy = semtypes.NIL
	}
	fnDefn := semtypes.NewFunctionDefinition()
	fnType := fnDefn.Define(t.typeEnv(), paramListTy, returnTy,
		semtypes.FunctionQualifiersFrom(t.typeEnv(), fn.IsIsolated(), fn.IsTransactional()))
	updateSymbolType(t, fn, fnType)
	sig := fnSym.Signature()
	sig.Flags |= fn.FuncSymbolFlags()
	sig.ParamTypes = paramTypes
	paramNames := make([]string, len(requiredParams))
	for i := range requiredParams {
		paramNames[i] = requiredParams[i].GetName().GetValue()
	}
	sig.ParamNames = paramNames
	sig.ReturnType = returnTy
	sig.RestParamType = restTy
	fnSym.SetSignature(sig)
	return fnType, paramTypes, restTy, returnTy, true
}

func resolveFunctionBody(p *packageTypeResolver, fn functionDecl) *functionTypeResolver {
	fnSymbol := p.getSymbol(fn.Symbol())
	fnSym, ok := fnSymbol.(model.FunctionSymbol)
	if !ok {
		p.internalError("expected function symbol", fn.GetPosition())
		return nil
	}
	ft := &functionTypeResolver{
		parentResolver:      p,
		tyCtx:               semtypes.ContextFrom(p.typeEnv()),
		implicitImports:     make(map[string]ast.BLangImportPackage),
		mappingAtomToBType:  make(map[*semtypes.MappingAtomicType]ast.BType),
		monoCounters:        make(map[string]int),
		scope:               fn.Scope(),
		mappingAtomToSymRef: make(map[*semtypes.MappingAtomicType]model.SymbolRef),
	}
	if !isPolymorphicFnSymbol(fnSym) {
		ft.retTy = fnSym.Signature().ReturnType
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
		body.SetDeterminedType(semtypes.NEVER)
	case *ast.BLangExprFunctionBody:
		resolveActionOrExpression(ft, nil, body.Expr, ft.retTy)
	default:
		p.internalError("unexpected function body kind", body.GetPosition())
	}
	return ft
}

func (t *packageTypeResolver) resolveTopLevelTypes(pkg *ast.BLangPackage) {
	for i := range pkg.TypeDefinitions {
		defn := &pkg.TypeDefinitions[i]
		t.typeDefnNodes[defn.Symbol()] = defn
	}
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		t.classDefnNodes[classDef.Symbol()] = classDef
	}

	for i := range pkg.Constants {
		t.packageConstants[pkg.Constants[i].Symbol()] = &pkg.Constants[i]
	}
	for i := range pkg.Functions {
		t.functionNodes[pkg.Functions[i].Symbol()] = &pkg.Functions[i]
	}

	for i := range pkg.TypeDefinitions {
		defn := &pkg.TypeDefinitions[i]
		if _, ok := resolveTypeDefinition(t, defn, 0); !ok {
			return
		}
	}
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		if _, ok := resolveClassTypeDefinition(t, classDef, 0); !ok {
			return
		}
	}
	for i := range pkg.Annotations {
		if !resolveAnnotationDeclaration(t, &pkg.Annotations[i]) {
			return
		}
	}
	for i := range pkg.Services {
		if !resolveServiceType(t, &pkg.Services[i], 0) {
			return
		}
	}
	populateClassSymbolByType(t, pkg)
	populateMappingAtomMaps(t, pkg, t.importedSymbols)
	for i := range pkg.Functions {
		fn := &pkg.Functions[i]
		if _, ok := resolveFunctionSignature(t, fn); !ok {
			return
		}
	}
	if pkg.InitFunction != nil {
		if _, ok := resolveFunctionSignature(t, pkg.InitFunction); !ok {
			return
		}
	}
	for i := range pkg.GlobalVars {
		resolveGlobalVarType(t, &pkg.GlobalVars[i])
	}
	if !resolvePackageConstants(t, pkg) {
		return
	}
	// Annotation values can depend on constants, so resolve them after constants
	// have been folded even though the annotated type/function nodes were
	// resolved earlier in the top-level pass.
	resolveTopLevelAnnotationAttachments(t, pkg)
	for i := range pkg.Imports {
		setOtherNodesAsNever(&pkg.Imports[i])
	}
	for i := range pkg.Functions {
		fn := &pkg.Functions[i]
		fn.SetDeterminedType(semtypes.NEVER)
		fn.Name.SetDeterminedType(semtypes.NEVER)
	}
	if pkg.InitFunction != nil {
		pkg.InitFunction.SetDeterminedType(semtypes.NEVER)
		pkg.InitFunction.Name.SetDeterminedType(semtypes.NEVER)
	}
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		classDef.SetDeterminedType(semtypes.NEVER)
		classDef.Name.SetDeterminedType(semtypes.NEVER)
	}
	for i := range pkg.Services {
		pkg.Services[i].SetDeterminedType(semtypes.NEVER)
	}
	pkg.SetDeterminedType(semtypes.NEVER)
	for i := range pkg.GlobalVars {
		resolveGlobalVarInit(t, &pkg.GlobalVars[i])
		setOtherNodesAsNever(&pkg.GlobalVars[i])
	}
	detectGlobalVarInitCycles(t, pkg)
	attachPointBound := listenerAttachPointBound(t.typeContext())
	validateListenerVars(t, pkg, attachPointBound)
	for i := range pkg.Services {
		resolveServiceAttachedExpressions(t, &pkg.Services[i])
		validateServiceDeclaration(t, &pkg.Services[i], attachPointBound)
	}
	for i := range pkg.XmlnsList {
		resolveXMLNS(t, nil, &pkg.XmlnsList[i])
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
// cached on the package resolver: the expensive part, CreateCloneable, is
// already memoized in the semtype context, and caching on the shared resolver
// would be a data race (local-node resolution runs many type resolvers
// concurrently off the same package resolver).
func annotationMapType(t typeResolver) semtypes.SemType {
	return semtypes.Intersect(semtypes.MAPPING, semtypes.CreateCloneable(t.typeContext()))
}

func annotationMapListType(t typeResolver) semtypes.SemType {
	ld := semtypes.NewListDefinition()
	return ld.DefineListTypeWrappedWithEnvSemType(t.typeEnv(), annotationMapType(t))
}

func resolveAnnotationDeclaration(t typeResolver, annotation *ast.BLangAnnotation) bool {
	if annotation.Name != nil {
		setOtherNodesAsNever(annotation.Name)
	}
	var ty semtypes.SemType
	var ok bool
	if typeDesc := annotation.GetTypeDescriptor(); typeDesc != nil {
		ty, ok = resolveBType(t, typeDesc.(ast.BType), 0)
		if !ok {
			return false
		}
		if !annotationTypeValid(t, ty) {
			t.semanticError("annotation type must be a subtype of true|map<Cloneable>|map<Cloneable>[]", typeDesc.GetPosition())
			return false
		}
		if annotation.IsConst() && !semtypes.IsSubtype(t.typeContext(), ty, semtypes.VAL_READONLY) {
			t.semanticError("const annotation type must be readonly", typeDesc.GetPosition())
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
	annotation.SetDeterminedType(semtypes.NEVER)
	return true
}

func resolveTopLevelAnnotationAttachments(t typeResolver, pkg *ast.BLangPackage) {
	initialGlobalCount := len(pkg.GlobalVars)
	for i := range pkg.Annotations {
		resolveAnnotationAttachments(t, &pkg.Annotations[i], ast.Point_ANNOTATION, model.SymbolRef{})
	}
	for i := range pkg.TypeDefinitions {
		defn := &pkg.TypeDefinitions[i]
		resolveAnnotationAttachments(t, defn, ast.Point_TYPE, defn.Symbol())
		switch typeDesc := defn.GetTypeData().TypeDescriptor.(type) {
		case *ast.BLangRecordType:
			for _, field := range typeDesc.FieldPtrs() {
				resolveAnnotationAttachments(t, field, ast.Point_RECORD_FIELD, model.SymbolRef{})
			}
		case *ast.BLangObjectType:
			for member := range typeDesc.Members() {
				if field, ok := member.(*ast.BObjectField); ok {
					resolveAnnotationAttachments(t, field, ast.Point_OBJECT_FIELD, model.SymbolRef{})
				}
			}
		}
	}
	for i := range pkg.ClassDefinitions {
		classDef := &pkg.ClassDefinitions[i]
		classPoint := ast.Point_CLASS
		if classDef.IsService() {
			classPoint = ast.Point_SERVICE
		}
		resolveAnnotationAttachments(t, classDef, classPoint, classDef.Symbol())
		for j := range classDef.Fields {
			resolveAnnotationAttachments(t, classDef.Fields[j], ast.Point_OBJECT_FIELD, model.SymbolRef{})
		}
		if classDef.InitFunction != nil {
			resolveFunctionAnnotationAttachments(t, classDef.InitFunction, true)
		}
		for _, method := range classDef.Methods {
			resolveFunctionAnnotationAttachments(t, method, true)
		}
		for _, method := range classDef.ResourceMethods {
			resolveInvokableAnnotationAttachments(t, method, ast.Point_OBJECT_METHOD)
		}
	}
	for i := range pkg.Functions {
		resolveFunctionAnnotationAttachments(t, &pkg.Functions[i], false)
	}
	if pkg.InitFunction != nil {
		resolveFunctionAnnotationAttachments(t, pkg.InitFunction, false)
	}
	for i := range pkg.Constants {
		resolveAnnotationAttachments(t, &pkg.Constants[i], ast.Point_CONST, model.SymbolRef{})
	}
	for i := range pkg.GlobalVars {
		resolveAnnotationAttachments(t, &pkg.GlobalVars[i], ast.Point_VAR, model.SymbolRef{})
	}
	if initialGlobalCount < len(pkg.GlobalVars) {
		globals := make([]ast.BLangSimpleVariable, 0, len(pkg.GlobalVars))
		globals = append(globals, pkg.GlobalVars[initialGlobalCount:]...)
		globals = append(globals, pkg.GlobalVars[:initialGlobalCount]...)
		pkg.GlobalVars = globals
	}
}

func resolveFunctionAnnotationAttachments(
	t typeResolver,
	fn *ast.BLangFunction,
	attached bool,
) {
	point := ast.Point_FUNCTION
	if attached {
		point = ast.Point_OBJECT_METHOD
	}
	resolveInvokableAnnotationAttachments(t, fn, point)
}

func resolveInvokableAnnotationAttachments(
	t typeResolver,
	fn ast.InvokableNode,
	point ast.Point,
) {
	resolveAnnotationAttachments(t, fn, point, model.SymbolRef{})
	for _, parameter := range fn.GetParameters() {
		resolveAnnotationAttachments(t, parameter, ast.Point_PARAMETER, model.SymbolRef{})
	}
	if restParam := fn.GetRestParam(); restParam != nil {
		resolveAnnotationAttachments(t, restParam, ast.Point_PARAMETER, model.SymbolRef{})
	}
	if ret := fn.GetReturnTypeDescriptor(); ret != nil {
		resolveAnnotationAttachments(t, ret, ast.Point_RETURN, model.SymbolRef{})
	}
}

func resolveAnnotationAttachments(
	t typeResolver,
	node ast.AnnotatableNode,
	point ast.Point,
	typeSymbol model.SymbolRef,
) {
	seen := make(map[string]bool)
	repeatedValues := make(map[string]*repeatedAnnotationValue)
	repeatedOrder := make([]string, 0)
	pointKey := point.String()
	for _, attachment := range node.GetAnnotationAttachments() {
		ann, ok := attachment.(*ast.BLangAnnotationAttachment)
		if !ok || !ast.SymbolIsSet(ann) {
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
			t.internalError("annotation type is not resolved", ann.GetPosition())
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
		if _, _, ok := resolveActionOrExpression(t, nil, ann.Expr, valueType); !ok {
			continue
		}
		ann.SetDeterminedType(semtypes.NEVER)
		if ann.PkgAlias != nil {
			setOtherNodesAsNever(ann.PkgAlias)
		}
		if ann.AnnotationName != nil {
			setOtherNodesAsNever(ann.AnnotationName)
		}
		value, err := evaluateAnnotationValue(t, ann.Expr)
		runtimeValue := false
		if err != nil {
			if errors.Is(err, errNotConstantExpression) {
				if sym.IsConst() {
					t.semanticError("const annotation value must be a constant expression", ann.Expr.GetPosition())
					continue
				}
				runtimeValue = true
			} else {
				t.semanticError("cannot evaluate annotation constant expression: "+err.Error(), ann.Expr.GetPosition())
				continue
			}
		}
		if !runtimeValue {
			ann.AnnotationValue = value
		}
		storedOnType := typeSymbol != (model.SymbolRef{}) && sym.IsRuntimeVisibleAt(pointKey)
		if repeated && storedOnType {
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
			if storedOnType {
				setTypeAnnotationValue(t, typeSymbol, key, createRuntimeAnnotationGlobal(t, ann.Expr))
			}
			continue
		}
		if storedOnType {
			setTypeAnnotationValue(t, typeSymbol, key, value)
		}
	}

	for _, key := range repeatedOrder {
		group := repeatedValues[key]
		atomic := semtypes.ToListAtomicType(t.typeContext(), group.listType)
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
			expr.SetDeterminedType(group.listType)
			setTypeAnnotationValue(t, typeSymbol, key, createRuntimeAnnotationGlobal(t, expr))
			continue
		}
		restFiller, _ := values.FillerFactoryFor(t.typeContext(), atomic.Rest())
		value := values.NewList(group.listType, atomic, true, restFiller, len(group.values), group.values)
		setTypeAnnotationValue(t, typeSymbol, key, value)
	}
}

func annotationAttachmentValueType(t typeResolver, annotationType semtypes.SemType) (semtypes.SemType, bool) {
	if semtypes.IsSubtypeSimple(annotationType, semtypes.LIST) {
		memberTy := semtypes.ListMemberTypeInnerVal(t.typeContext(), annotationType, semtypes.INT)
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
		ann.Expr = newImplicitBooleanLiteral(true, ann.GetPosition())
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

func newImplicitBooleanLiteral(value bool, pos diagnostics.Location) *ast.BLangLiteral {
	lit := &ast.BLangLiteral{}
	lit.SetValueType(ast.NewBType(ast.TypeTags_BOOLEAN, model.Name(""), uint64(model.FlagReadonly)))
	lit.SetDeterminedType(semtypes.BooleanConst(value))
	lit.SetValue(value)
	lit.SetOriginalValue(strconv.FormatBool(value))
	lit.SetIsConstant(true)
	lit.SetPosition(pos)
	return lit
}

type repeatedAnnotationValue struct {
	listType    semtypes.SemType
	values      []values.BalValue
	expressions []ast.BLangExpression
	runtime     bool
}

func evaluateAnnotationValue(t typeResolver, expr ast.BLangExpression) (value values.AnnotationValue, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("constant expression evaluation panicked: %v", recovered)
		}
	}()
	return evaluateConstantExpression(t, expr)
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
	symbol := model.NewVariableSymbol(name, false, false, false)
	symbol.SetType(semtypes.ANY)
	resolver.scope.AddSymbol(name, &symbol)
	ref, _ := resolver.scope.GetSymbol(name)

	identifier := &ast.BLangIdentifier{Value: name}
	identifier.SetPosition(expr.GetPosition())
	identifier.SetDeterminedType(semtypes.NEVER)
	global := ast.BLangSimpleVariable{Name: identifier}
	global.SetPosition(expr.GetPosition())
	global.SetSymbol(ref)
	global.SetDeterminedType(semtypes.ANY)
	global.SetInitialExpression(expr)
	resolver.pkg.GlobalVars = append(resolver.pkg.GlobalVars, global)

	return &values.RuntimeAnnotationValueRef{
		Organization: resolver.pkg.PackageID.OrgName.Value(),
		Module:       resolver.pkg.PackageID.PkgName.Value(),
		GlobalName:   name,
	}
}

func setTypeAnnotationValue(t typeResolver, symbol model.SymbolRef, key string, value values.AnnotationValue) {
	t.compilerContext().SetSymbolAnnotationValue(symbol, key, value)
}

func resolveBlockStatements(t typeResolver, chain *binding, stmts []ast.StatementNode) (statementEffect, bool) {
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
			return statementEffect{result, true}, true
		}
	}
	return statementEffect{result, false}, true
}

func resolveStatement(t typeResolver, chain *binding, stmt ast.StatementNode) (statementEffect, bool) {
	effect, ok := resolveStatementInner(t, chain, stmt)
	stmt.(ast.BLangNode).SetDeterminedType(semtypes.NEVER)
	return effect, ok
}

func resolveCompoundAssignment(t typeResolver, chain *binding, s *ast.BLangCompoundAssignment) (statementEffect, bool) {
	lhs := s.GetVariable()
	rhs := s.GetExpression()
	lhsTy, rhsChain, ok := resolveCompoundAssignmentLhs(t, chain, lhs)
	if !ok {
		return statementEffect{}, false
	}
	if _, _, ok := resolveCompoundAssignmentInner(t, rhsChain, lhsTy, rhs, s.OpKind, s.GetPosition()); !ok {
		return statementEffect{}, false
	}
	if expr, ok := s.GetVariable().(ast.NodeWithSymbol); ok {
		return unnarrowSymbolAt(t, rhsChain, expr.Symbol(), lhs.GetPosition()), true
	}
	return defaultStmtEffect(rhsChain), true
}

// resolveCompoundAssignmentLhs resolves the LHS of a compound assignment and returns the
// narrowed LHS type to use as the operand type along with the chain in which the RHS should be
// resolved. The LHS node's determined type is always set to its writable (unnarrowed) type so
// that later assignment validation checks the RHS against the declared target type.
func resolveCompoundAssignmentLhs(t typeResolver, chain *binding, lhs ast.BLangExpression) (semtypes.SemType, *binding, bool) {
	switch lhs.(type) {
	case *ast.BLangIndexBasedAccess, *ast.BLangFieldBaseAccess:
		lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, lhs, semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, nil, false
		}
		return lhsTy, lhsEffect.ifTrue, true
	default:
		_, _, ok := resolveActionOrExpression(t, nil, lhs, semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, nil, false
		}
		if ref, isVarRef := varRefExp(chain, lhs); isVarRef {
			return t.symbolType(ref), chain, true
		}
		return lhs.GetDeterminedType(), chain, true
	}
}

func resolveCompoundAssignmentInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	// Use the widened basic-type form of the LHS as the contextual expected type for the RHS
	// so that literals (e.g. `r["x"] += 1` where `x` is float) are typed against the LHS basic
	// type rather than a possibly-singleton narrowed type.
	rhsExpectedType := semtypes.WidenToBasicTypes(lhsTy)
	switch op {
	case model.OperatorKind_ADD, model.OperatorKind_SUB:
		return resolveAdditiveExprInner(t, chain, lhsTy, rhs, op, rhsExpectedType, pos)
	case model.OperatorKind_MUL, model.OperatorKind_DIV, model.OperatorKind_MOD:
		return resolveMultiplicativeExprInner(t, chain, lhsTy, rhs, op, rhsExpectedType, pos)
	case model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		return resolveBitWiseExprInner(t, chain, lhsTy, rhs, op, pos)
	case model.OperatorKind_BITWISE_LEFT_SHIFT, model.OperatorKind_BITWISE_RIGHT_SHIFT, model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return resolveShiftExprInner(t, chain, lhsTy, rhs, op, pos)
	case model.OperatorKind_AND:
		return resolveAndExprInner(t, chain, lhsTy, defaultExpressionEffect(chain), rhs, pos)
	case model.OperatorKind_OR:
		return resolveOrExprInner(t, chain, lhsTy, defaultExpressionEffect(chain), rhs, pos)
	}
	t.internalError(fmt.Sprintf("unexpected compound assignment operator %s", string(op)), pos)
	return semtypes.SemType{}, expressionEffect{}, false
}

func resolveAssignment(t typeResolver, chain *binding, s assignmentNode) (statementEffect, bool) {
	var lhsTy semtypes.SemType
	switch lhs := s.GetVariable().(type) {
	case *ast.BLangIndexBasedAccess, *ast.BLangFieldBaseAccess:
		// we don't assign to the actual container so shoud use the narrowed type for the container variable
		var lhsEffect expressionEffect
		var ok bool
		lhsTy, lhsEffect, ok = resolveActionOrExpression(t, chain, lhs, semtypes.SemType{})
		if !ok {
			return statementEffect{}, false
		}
		chain = lhsEffect.ifTrue
	default:
		var ok bool
		lhsTy, _, ok = resolveActionOrExpression(t, nil, lhs, semtypes.SemType{})
		if !ok {
			return statementEffect{}, false
		}
	}
	if _, _, ok := resolveActionOrExpression(t, chain, s.GetExpression(), lhsTy); !ok {
		return statementEffect{}, false
	}
	if expr, ok := s.GetVariable().(ast.NodeWithSymbol); ok {
		return unnarrowSymbolAt(t, chain, expr.Symbol(), s.GetVariable().GetPosition()), true
	}
	return defaultStmtEffect(chain), true
}

func resolveStatementInner(t typeResolver, chain *binding, stmt ast.StatementNode) (statementEffect, bool) {
	if scoped, ok := stmt.(ast.NodeWithScope); ok {
		if scope := scoped.Scope(); scope != nil {
			prev := t.currentScope()
			t.setCurrentScope(scope)
			defer t.setCurrentScope(prev)
		}
	}
	switch s := stmt.(type) {
	case *ast.BLangSimpleVariableDef:
		return resolveVariableDefStmt(t, chain, s)
	case *ast.BLangAssignment:
		return resolveAssignment(t, chain, s)
	case *ast.BLangCompoundAssignment:
		return resolveCompoundAssignment(t, chain, s)
	case *ast.BLangExpressionStmt:
		if _, _, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.SemType{}); !ok {
			return defaultStmtEffect(chain), false
		}
		return defaultStmtEffect(chain), true
	// PT-TODO: extract if while out
	case *ast.BLangIf:
		_, exprEffect, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.BOOLEAN)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		ifTrueEffect, ok := resolveBlockStatements(t, exprEffect.ifTrue, s.Body.Stmts)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		s.Body.SetDeterminedType(semtypes.NEVER)
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
		_, exprEffect, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.BOOLEAN)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		loopT := &loopTypeResolver{parentResolver: t}
		bodyEffect, ok := resolveBlockStatements(loopT, exprEffect.ifTrue, s.Body.Stmts)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		s.Body.SetDeterminedType(semtypes.NEVER)
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
			if _, _, ok := resolveActionOrExpression(t, chain, s.Expr, t.expectedReturnType()); !ok {
				return defaultStmtEffect(chain), false
			}
		}
		return statementEffect{nil, true}, true
	case *ast.BLangBlockStmt:
		return resolveBlockStatements(t, chain, s.Stmts)
	case *ast.BLangLock:
		effect, ok := resolveBlockStatements(t, chain, s.Body.Stmts)
		s.Body.SetDeterminedType(semtypes.NEVER)
		return effect, ok
	case *ast.BLangForeach:
		collectionTy, _, ok := resolveActionOrExpression(t, chain, s.Collection, semtypes.SemType{})
		if !ok {
			return defaultStmtEffect(chain), false
		}
		variable := s.VariableDef.GetVariable().(*ast.BLangSimpleVariable)
		if s.GetIsDeclaredWithVar() {
			variableTy, ok := resolveForeachVariableType(t, s.Collection, collectionTy)
			if !ok {
				return defaultStmtEffect(chain), false
			}
			variable.Name.SetDeterminedType(semtypes.NEVER)
			setExpectedType(variable, variableTy)
			updateSymbolType(t, variable, variableTy)
		} else if !resolveSimpleVariable(t, chain, variable) {
			return defaultStmtEffect(chain), false
		}
		s.VariableDef.SetDeterminedType(semtypes.NEVER)
		// foreach may run zero times, so the post-loop chain starts from the
		// loop-entry chain. Body completion and any break paths are merged in.
		loopT := &loopTypeResolver{parentResolver: t}
		bodyEffect, ok := resolveBlockStatements(loopT, chain, s.Body.Stmts)
		s.Body.SetDeterminedType(semtypes.NEVER)
		if s.OnFailClause != nil {
			resolveOnFailClause(t, chain, s.OnFailClause)
		}
		if !ok {
			return defaultStmtEffect(chain), false
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
		if _, _, ok := resolveActionOrExpression(t, chain, s.Expr, semtypes.ERROR); !ok {
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
		resolveXMLNS(t, chain, s)
		return defaultStmtEffect(chain), true
	default:
		t.internalError(fmt.Sprintf("unhandled statement type: %T", stmt), stmt.GetPosition())
		return defaultStmtEffect(chain), false
	}
}

func resolveXMLNS(t typeResolver, chain *binding, decl *ast.BLangXMLNS) {
	decl.SetDeterminedType(semtypes.NEVER)
	if uriExpr := decl.GetNamespaceURI(); uriExpr != nil {
		resolveActionOrExpression(t, chain, uriExpr, semtypes.STRING)
	}
	if prefix := decl.GetPrefix(); prefix != nil {
		prefix.SetDeterminedType(semtypes.NEVER)
	}
}

func resolveOnFailClause(t typeResolver, chain *binding, clause *ast.BLangOnFailClause) {
	clause.SetDeterminedType(semtypes.NEVER)
	if clause.VariableDefinitionNode != nil {
		varDef := clause.VariableDefinitionNode
		variable := varDef.GetVariable().(*ast.BLangSimpleVariable)
		resolveSimpleVariable(t, chain, variable)
		varDef.SetDeterminedType(semtypes.NEVER)
	}
	if clause.Body != nil {
		resolveBlockStatements(t, chain, clause.Body.Stmts)
		clause.Body.SetDeterminedType(semtypes.NEVER)
	}
}

func resolveFunctionSignature(t typeResolver, fn *ast.BLangFunction) (semtypes.SemType, bool) {
	fnSym := t.getSymbol(fn.Symbol())
	if depSym, ok := fnSym.(model.DependentlyTypedFunctionSymbol); ok {
		return resolveDependentlyTypedFunctionSignature(t, fn, depSym)
	}
	if ty := t.symbolType(fn.Symbol()); !semtypes.IsZero(ty) {
		return ty, true
	}
	fnSymbol := fnSym.(model.FunctionSymbol)
	fnType, paramTypes, _, _, ok := resolveInvokableSignature(t, fn, fnSymbol, fn.RequiredParams)
	if !ok {
		return semtypes.SemType{}, false
	}

	setDefaultableParamFnSignatures(t, fnSymbol.DefaultableParams(), paramTypes)

	if !validateIncludedRecordParams(t, fn, fnSymbol) {
		return semtypes.SemType{}, false
	}

	return fnType, true
}

func validateIncludedRecordParams(t typeResolver, fn *ast.BLangFunction, fnSymbol model.FunctionSymbol) bool {
	info := fnSymbol.IncludedRecordParams()
	if info == nil {
		return true
	}
	paramNames := fnSymbol.ParamNames()
	fieldOrigin := make(map[string]int)
	for i := range fn.RequiredParams {
		param := &fn.RequiredParams[i]
		if !info.IsIncluded(i) {
			continue
		}
		udt, ok := param.TypeNode().(*ast.BLangUserDefinedType)
		if !ok {
			t.semanticError("included record parameter must be a record type", param.GetPosition())
			return false
		}
		recRef := udt.Symbol()
		t.ensureResolved(recRef, 0)
		recSym, ok := t.getSymbol(recRef).(*model.RecordSymbol)
		if !ok {
			t.semanticError("included record parameter must be a record type", param.GetPosition())
			return false
		}
		var fieldNames []string
		for name, field := range recSym.Fields() {
			if semtypes.IsNever(field.MemberType()) {
				continue
			}
			for j, pname := range paramNames {
				if j == i {
					continue
				}
				if pname == name {
					t.semanticError(
						fmt.Sprintf("parameter '%s' conflicts with field of included record parameter '%s'", name, paramNames[i]),
						param.GetPosition())
					return false
				}
			}
			if fn.RestParam != nil {
				restParam := fn.RestParam.(*ast.BLangSimpleVariable)
				if restParam.GetName().GetValue() == name {
					t.semanticError(
						fmt.Sprintf("parameter '%s' conflicts with field of included record parameter '%s'", name, paramNames[i]),
						param.GetPosition())
					return false
				}
			}
			if prev, seen := fieldOrigin[name]; seen {
				t.semanticError(
					fmt.Sprintf("duplicate field '%s' in included record parameters '%s' and '%s'", name, paramNames[prev], paramNames[i]),
					param.GetPosition())
				return false
			}
			fieldOrigin[name] = i
			fieldNames = append(fieldNames, name)
		}
		info.SetFields(i, fieldNames)
	}
	return true
}

func resolveDependentlyTypedFunctionSignature(t typeResolver, fn *ast.BLangFunction, sym model.DependentlyTypedFunctionSymbol) (semtypes.SemType, bool) {
	paramTypes := make([]semtypes.SemType, len(fn.RequiredParams))
	paramsByName := make(map[string]param, len(fn.RequiredParams))
	for i := range fn.RequiredParams {
		p := &fn.RequiredParams[i]
		resolveSimpleVariable(t, nil, p)
		paramTypes[i] = p.GetDeterminedType()
		paramsByName[p.GetName().GetValue()] = param{index: i, ty: paramTypes[i]}
	}
	retTd := fn.GetReturnTypeDescriptor()
	if retTd == nil {
		t.internalError("dependently-typed function has no return type descriptor", fn.GetPosition())
		return semtypes.SemType{}, false
	}
	retOp, ok := buildReturnTypeOp(t, paramsByName, retTd)
	if !ok {
		t.internalError("failed to build return type op for dependently-typed function", fn.GetPosition())
		return semtypes.SemType{}, false
	}
	setOtherNodesAsNever(retTd)
	sym.SetParamTypes(paramTypes)
	sym.SetReturnType(retOp)
	setDefaultableParamFnSignatures(t, sym.DefaultableParams(), paramTypes)
	if !validateIncludedRecordParams(t, fn, sym) {
		return semtypes.SemType{}, false
	}
	setOtherNodesAsNever(fn)
	return semtypes.NEVER, true
}

// setDefaultableParamFnSignatures populates the signature of each non-typedesc
// default-provider function. The signature is (paramTypes[:i]) -> paramTypes[i].
func setDefaultableParamFnSignatures(t typeResolver, defaultable *model.DefaultableParamInfo, paramTypes []semtypes.SemType) {
	for i := range paramTypes {
		dp, ok := defaultable.Get(i)
		if !ok {
			continue
		}
		if dp.Kind == model.DefaultableParamKindInferredTypedesc {
			continue
		}
		defaultFnSym := t.getSymbol(dp.Symbol).(model.FunctionSymbol)
		sig := model.FunctionSignature{
			ParamTypes: paramTypes[:i],
			ReturnType: paramTypes[i],
		}
		defaultFnSym.SetSignature(sig)
	}
}

type param struct {
	index int
	ty    semtypes.SemType
}

// buildReturnTypeOp translates a return-type-descriptor AST node into a TypeOp tree.
// A user-defined-type node whose name matches a typedesc parameter becomes a RefTypeOp.
// Union and intersection nodes recurse. Everything else is resolved to a concrete semtype
// and wrapped in an IdentityTypeOp.
func buildReturnTypeOp(t typeResolver, params map[string]param, node ast.BLangNode) (model.TypeOp, bool) {
	switch n := node.(type) {
	case *ast.BLangReturnTypeDescriptor:
		return buildReturnTypeOp(t, params, n.TypeDescriptor)
	case *ast.BLangUnionTypeNode:
		lhs, ok := buildReturnTypeOp(t, params, n.Lhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, false
		}
		rhs, ok := buildReturnTypeOp(t, params, n.Rhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, false
		}
		return &model.BinaryTypeOp{Kind: model.TypeOpUnion, Lhs: lhs, Rhs: rhs}, true
	case *ast.BLangIntersectionTypeNode:
		lhs, ok := buildReturnTypeOp(t, params, n.Lhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, false
		}
		rhs, ok := buildReturnTypeOp(t, params, n.Rhs().TypeDescriptor.(ast.BLangNode))
		if !ok {
			return nil, false
		}
		return &model.BinaryTypeOp{Kind: model.TypeOpIntersection, Lhs: lhs, Rhs: rhs}, true
	case *ast.BLangUserDefinedType:
		if n.PkgAlias.Value == "" {
			if p, ok := params[n.TypeName.Value]; ok && semtypes.IsSubtype(t.typeContext(), p.ty, semtypes.TYPEDESC) {
				return &model.RefTypeOp{Index: p.index}, true
			}
		}
		ty, ok := resolveBType(t, n, 0)
		if !ok {
			return nil, false
		}
		return &model.IdentityTypeOp{Type: ty}, true
	default:
		ty, ok := resolveBType(t, node.(ast.BType), 0)
		if !ok {
			return nil, false
		}
		return &model.IdentityTypeOp{Type: ty}, true
	}
}

func resolveLambdaFunctionExpr(t typeResolver, chain *binding, e *ast.BLangLambdaFunction) (semtypes.SemType, expressionEffect, bool) {
	fnType, ok := resolveFunctionSignature(t, e.Function)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	// Create a function type resolver for the lambda so expectedReturnType() is correct
	fnSym := t.getSymbol(e.Function.Symbol()).(model.FunctionSymbol)
	ft := &functionTypeResolver{
		parentResolver:      t,
		tyCtx:               semtypes.ContextFrom(t.typeEnv()),
		retTy:               fnSym.Signature().ReturnType,
		implicitImports:     make(map[string]ast.BLangImportPackage),
		mappingAtomToBType:  make(map[*semtypes.MappingAtomicType]ast.BType),
		monoCounters:        make(map[string]int),
		scope:               e.Function.Scope(),
		mappingAtomToSymRef: make(map[*semtypes.MappingAtomicType]model.SymbolRef),
	}

	// Push function boundary marker onto the chain
	boundaryChain := &binding{flags: bindingFlagFunctionBoundary, prev: chain}

	// Save and reset capture tracker (supports nested lambdas)
	prevCaptured := t.getCapturedVars()
	ft.setCapturedVars(make(map[model.SymbolRef]bool))

	switch body := e.Function.Body.(type) {
	case *ast.BLangBlockFunctionBody:
		resolveBlockStatements(ft, boundaryChain, body.Stmts)
		body.SetDeterminedType(semtypes.NEVER)
	case *ast.BLangExprFunctionBody:
		if _, _, ok := resolveActionOrExpression(ft, boundaryChain, body.Expr, ft.retTy); !ok {
			t.setCapturedVars(prevCaptured)
			return semtypes.SemType{}, expressionEffect{}, false
		}
		body.SetDeterminedType(semtypes.NEVER)
	}

	// Unnarrow all captured variables
	outerChain := chain
	for ref := range ft.getCapturedVars() {
		outerChain = unnarrowSymbol(t, outerChain, ref).binding
	}

	// propagate captured variables to parent
	if prevCaptured != nil {
		for ref := range ft.getCapturedVars() {
			prevCaptured[ref] = true
		}
	}

	t.setCapturedVars(prevCaptured)

	e.Function.SetDeterminedType(semtypes.NEVER)
	e.Function.Name.SetDeterminedType(semtypes.NEVER)
	setExpectedType(e, fnType)
	return fnType, defaultExpressionEffect(outerChain), true
}

func resolveTypeData(t typeResolver, typeData *ast.TypeData) bool {
	if typeData.TypeDescriptor == nil {
		return true
	}
	ty, ok := resolveBType(t, typeData.TypeDescriptor.(ast.BType), 0)
	if !ok {
		return false
	}
	typeData.Type = ty
	return true
}

type neverVisitor struct{}

func (neverVisitor) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if semtypes.IsZero(node.GetDeterminedType()) {
		node.SetDeterminedType(semtypes.NEVER)
	}
	return neverVisitor{}
}

func (neverVisitor) VisitTypeData(_ *ast.TypeData) ast.Visitor {
	return neverVisitor{}
}

// setOtherNodesAsNever set type of every ast node who's determined type is not set as NEVER
func setOtherNodesAsNever(node ast.BLangNode) {
	ast.Walk(neverVisitor{}, node)
}

func allocateDefaultFnSymbol(t typeResolver, fieldTy semtypes.SemType) model.SymbolRef {
	fnName := t.nextDefaultFnName()
	sig := model.FunctionSignature{ReturnType: fieldTy}
	fnSymbol := model.NewFunctionSymbol(fnName, sig, false)
	scope := t.currentScope()
	scope.AddSymbol(fnName, fnSymbol)
	ref, _ := scope.GetSymbol(fnName)
	return ref
}

func resolveTypeDefinition(t typeResolver, defn *ast.BLangTypeDefinition, depth int) (semtypes.SemType, bool) {
	if ty := t.symbolType(defn.Symbol()); !semtypes.IsZero(ty) {
		return ty, true
	}
	if defn.GetName() != nil {
		setOtherNodesAsNever(defn.GetName())
	}
	if depth == defn.GetCycleDepth() {
		t.semanticError(fmt.Sprintf("invalid cycle detected for type definition %s", defn.GetName().GetValue()), defn.GetPosition())
		return semtypes.SemType{}, false
	}
	defn.SetCycleDepth(depth)
	semType, ok := resolveBType(t, defn.GetTypeData().TypeDescriptor.(ast.BType), depth)
	if ok {
		semType, ok = resolveDistinctTypeDefinition(t, defn, semType)
	}
	if !ok {
		return semtypes.SemType{}, false
	}
	if semtypes.IsZero(defn.GetDeterminedType()) {
		defn.SetDeterminedType(semType)
		t.setSymbolType(defn.Symbol(), semType)
		defn.SetCycleDepth(-1)
		typeData := defn.GetTypeData()
		typeData.Type = semType
		defn.SetTypeData(typeData)
		addInclusionsToTypeSymbol(t, defn)
		name := defn.GetName().GetValue()
		pos := defn.GetPosition()
		t.ensureNotEmpty(semType, func() {
			t.semanticError(fmt.Sprintf("type definition %s is empty", name), pos)
		})
		return semType, true
	}
	return defn.GetDeterminedType(), true
}

func resolveClassTypeDefinition(t typeResolver, classDef *ast.BLangClassDefinition, depth int) (semtypes.SemType, bool) {
	if ty := t.symbolType(classDef.Symbol()); !semtypes.IsZero(ty) {
		return ty, true
	}
	if classDef.GetName() != nil {
		setOtherNodesAsNever(classDef.GetName())
	}
	if depth == classDef.GetCycleDepth() {
		t.semanticError(fmt.Sprintf("invalid cycle detected for type definition %s", classDef.GetName().GetValue()), classDef.GetPosition())
		return semtypes.SemType{}, false
	}
	classDef.SetCycleDepth(depth)
	semType, ok := resolveClassDefinitionType(t, classDef, depth)
	if !ok {
		return semtypes.SemType{}, false
	}
	return semType, true
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
			return semType, true
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
	for name := range td.Fields() {
		directFields[name] = true
	}

	// Add direct fields
	for name, field := range td.FieldPtrs() {
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
		rd.SetMemberType(td.RestType.(ast.BLangNode).GetDeterminedType())
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
	// Add direct members
	for m := range td.Members() {
		switch member := m.(type) {
		case *ast.BObjectField:
			fd := objectFieldDescriptor(member)
			members = append(members, &fd)
		case *ast.BMethodDecl:
			md := methodDescriptor(member, model.SymbolRef{})
			members = append(members, &md)
		}
	}
	return members
}

// classMembers accumulate members both added by type inclusion and defined in the class decl itself
func classMembers(t typeResolver, classDef *ast.BLangClassDefinition) []model.InclusionMember {
	var members []model.InclusionMember
	// Collect transitive members from included types
	for _, symRef := range classDef.Inclusions {
		incSym := getMemberCarrier(t, symRef)
		if incSym == nil {
			t.internalError("failed to find included symbol", classDef.GetPosition())
			return nil
		}
		members = append(members, incSym.Members()...)
	}
	// Add direct members
	for _, fieldNode := range classDef.Fields {
		field := fieldNode.(*ast.BLangSimpleVariable)
		fd := classFieldDescriptor(t, field)
		members = append(members, &fd)
	}
	for name := range classDef.Methods {
		method := classDef.Methods[name]
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
	fd.SetMemberType(field.GetDeterminedType())
	return fd
}

func methodDescriptor(method *ast.BMethodDecl, fnRef model.SymbolRef) model.MethodDescriptor {
	kind := model.InclusionMemberKindMethod
	switch method.MemberKind() {
	case ast.ObjectMemberKindRemoteMethod:
		kind = model.InclusionMemberKindRemoteMethod
	case ast.ObjectMemberKindResourceMethod:
		kind = model.InclusionMemberKindResourceMethod
	}
	md := model.NewMethodDescriptor(method.Name(), kind, method.IsPublic(), fnRef)
	md.SetMemberType(method.GetDeterminedType())
	return md
}

func classFieldDescriptor(t typeResolver, field *ast.BLangSimpleVariable) model.FieldDescriptor {
	var flags model.FieldDescriptorFlag
	if field.IsReadonly() {
		flags |= model.FieldDescriptorReadonly
	}
	fd := model.NewFieldDescriptor(field.Name.Value, flags, field.IsPublic())
	fd.SetMemberType(t.symbolType(field.Symbol()))
	return fd
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
	if field.DefaultExpr != nil {
		flags |= model.FieldDescriptorHasDefault
	}
	fd := model.NewFieldDescriptor(name, flags, true)
	fd.SetMemberType(field.Type.(ast.BLangNode).GetDeterminedType())
	fd.DefaultFnRef = field.DefaultFnRef
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

	semType, ok := finishResolveObjectDefinitionType(t, &od, classDef.Fields, classDef.Methods, classDef.ResourceMethods, classDef.InitFunction,
		classDef.Inclusions, classDef.GetPosition(), depth, classDef.IsIsolated(), classDef.IsReadonly(), isClient, isService,
		classDef.Symbol())
	if !ok {
		return semtypes.SemType{}, false
	}

	classDef.SetDeterminedType(semType)
	t.setSymbolType(classDef.Symbol(), semType)
	classDef.SetCycleDepth(-1)
	typeData := classDef.GetTypeData()
	typeData.Type = semType
	classDef.SetTypeData(typeData)
	addInclusionsToClassSymbol(t, classDef)
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

func resolveServiceType(t typeResolver, svc *ast.BLangService, depth int) bool {
	if !semtypes.IsZero(svc.GetDeterminedType()) {
		return true
	}
	if svc.Definition != nil {
		return true
	}

	od := semtypes.NewObjectDefinition()
	svc.Definition = &od

	semType, ok := finishResolveObjectDefinitionType(t, &od, svc.Fields, svc.Methods, svc.ResourceMethods, svc.InitFunction,
		nil, svc.GetPosition(), depth, svc.IsIsolated(), false, false, true, model.SymbolRef{})
	if !ok {
		return false
	}

	svc.SetDeterminedType(semType)
	typeData := svc.GetTypeData()
	if typeData.TypeDescriptor != nil {
		if _, ok := resolveBType(t, typeData.TypeDescriptor.(ast.BType), depth+1); !ok {
			return false
		}
	}
	typeData.Type = semType
	svc.SetTypeData(typeData)
	if selfRef, ok := svc.Scope().GetSymbol("self"); ok {
		t.setSymbolType(selfRef, semType)
	}
	t.ensureNotEmpty(semType, func() {
		t.semanticError("service definition is empty", svc.GetPosition())
	})
	return true
}

func finishResolveObjectDefinitionType(t typeResolver, od *semtypes.ObjectDefinition, fields []ast.SimpleVariableNode,
	methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod, initFn *ast.BLangFunction, inclusions []model.SymbolRef,
	pos diagnostics.Location, depth int, isIsolated, isReadonly, isClient, isService bool, distinctSymbol model.SymbolRef,
) (semtypes.SemType, bool) {
	for _, fieldNode := range fields {
		field := fieldNode.(*ast.BLangSimpleVariable)
		fieldTy, ok := resolveBType(t, field.TypeNode(), depth+1)
		if !ok {
			return semtypes.SemType{}, false
		}
		setExpectedType(field, fieldTy)
		updateSymbolType(t, field, fieldTy)
		field.Name.SetDeterminedType(semtypes.NEVER)
	}

	if initFn != nil {
		if _, ok := resolveFunctionSignature(t, initFn); !ok {
			return semtypes.SemType{}, false
		}
		initFn.SetDeterminedType(semtypes.NEVER)
		initFn.Name.SetDeterminedType(semtypes.NEVER)
	}

	for name := range methods {
		method := methods[name]
		if _, ok := resolveFunctionSignature(t, method); !ok {
			return semtypes.SemType{}, false
		}
		method.SetDeterminedType(semtypes.NEVER)
		method.Name.SetDeterminedType(semtypes.NEVER)
	}

	for _, rm := range resourceMethods {
		if !resolveResourceMethodSignature(t, isClient, isService, rm) {
			return semtypes.SemType{}, false
		}
		rm.SetDeterminedType(semtypes.NEVER)
		rm.Name.SetDeterminedType(semtypes.NEVER)
	}

	includedMembers, ok := collectObjectIncludedMembers(t, inclusions, pos, depth)
	if !ok {
		return semtypes.SemType{}, false
	}

	directMembers, ok := buildObjectDirectMembers(t, fields, methods, initFn, isClient, isService)
	if !ok {
		return semtypes.SemType{}, false
	}

	members, ok := validateOverridesAndMerge(t, directMembers, includedMembers, pos, false)
	if !ok {
		return semtypes.SemType{}, false
	}

	return defineObjectSemType(t, od, isIsolated, isReadonly, isClient, isService, members, distinctSymbol, inclusions), true
}

func collectObjectIncludedMembers(t typeResolver, inclusions []model.SymbolRef, pos diagnostics.Location, depth int) (map[string][]semtypes.Member, bool) {
	includedMembers := make(map[string][]semtypes.Member)
	incMembers, err := collectIncludedMembers(t, inclusions, depth)
	if err {
		t.semanticError("error resolving type inclusion", pos)
		return nil, false
	}
	for _, m := range incMembers {
		if m.MemberKind() == model.InclusionMemberKindRestType {
			t.internalError("unexpected rest inclusion", pos)
		}
		member := inclusionMemberToSemtypeMember(m)
		includedMembers[member.Name] = append(includedMembers[member.Name], member)
	}
	return includedMembers, true
}

func buildObjectDirectMembers(t typeResolver, fields []ast.SimpleVariableNode, methods map[string]*ast.BLangFunction, initFn *ast.BLangFunction, isClient bool, isService bool) ([]directMember, bool) {
	var directMembers []directMember
	for _, fieldNode := range fields {
		field := fieldNode.(*ast.BLangSimpleVariable)
		fieldTy := field.GetDeterminedType()
		vis := semtypes.VisibilityPrivate
		if field.IsPublic() {
			vis = semtypes.VisibilityPublic
		}
		directMembers = append(directMembers, directMember{
			name:       field.Name.Value,
			valueTy:    fieldTy,
			kind:       semtypes.MemberKindField,
			visibility: vis,
			immutable:  false,
			pos:        field.GetPosition(),
		})
	}

	if initMember, ok := initDirectMember(t, initFn); ok {
		directMembers = append(directMembers, initMember)
	} else {
		return nil, false
	}

	for name := range methods {
		method := methods[name]
		methodTy := methodMemberType(t, method.Symbol())
		vis := semtypes.VisibilityPrivate
		if method.IsPublic() {
			vis = semtypes.VisibilityPublic
		}
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

// initDirectMember returns the init function member (explicit or implicit).
func initDirectMember(t typeResolver, initFn *ast.BLangFunction) (directMember, bool) {
	if initFn != nil {
		initFnSymbol := t.getSymbol(initFn.Symbol()).(model.FunctionSymbol)
		sig := initFnSymbol.Signature()
		tyCtx := t.typeContext()
		if !semtypes.IsSubtype(tyCtx, sig.ReturnType, semtypes.Union(semtypes.ERROR, semtypes.NIL)) {
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
	paramListTy := paramListDefn.DefineListTypeWrapped(t.typeEnv(), nil, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	functionDefn := semtypes.NewFunctionDefinition()
	initFnType := functionDefn.Define(t.typeEnv(), paramListTy, semtypes.NIL,
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

func resolveLiteral(t typeResolver, n *ast.BLangLiteral, expectedType semtypes.SemType) bool {
	bType := n.GetValueType()
	var ty semtypes.SemType

	switch bType.BTypeGetTag() {
	case ast.TypeTags_INT, ast.TypeTags_BYTE, ast.TypeTags_FLOAT, ast.TypeTags_DECIMAL:
		var ok bool
		ty, ok = resolveNumericLiteralValue(t, n, expectedType)
		if !ok {
			return false
		}
	case ast.TypeTags_BOOLEAN:
		value := n.GetValue().(bool)
		ty = semtypes.BooleanConst(value)
	case ast.TypeTags_STRING:
		value := n.GetValue().(string)
		ty = semtypes.StringConst(value)
	case ast.TypeTags_NIL:
		ty = semtypes.NIL
	default:
		t.unimplemented("unsupported literal type", n.GetPosition())
		return false
	}

	setExpectedType(n, ty)

	// Update symbol type if this literal has a symbol
	updateSymbolType(t, n, ty)
	return true
}

func hasFloatTypeSuffix(s string) bool {
	if len(s) == 0 {
		return false
	}
	last := s[len(s)-1]
	return last == 'f' || last == 'F'
}

func determineCandidatesFromLiteral(t typeResolver, n *ast.BLangLiteral) semtypes.SemType {
	switch n.GetValueType().BTypeGetTag() {
	case ast.TypeTags_INT, ast.TypeTags_BYTE:
		return semtypes.NUMBER
	case ast.TypeTags_FLOAT:
		if hasFloatTypeSuffix(n.OriginalValue) {
			return semtypes.FLOAT
		}
		if balCommon.HasHexIndicator(n.OriginalValue) {
			return semtypes.FLOAT
		}
		return semtypes.Union(semtypes.FLOAT, semtypes.DECIMAL)
	case ast.TypeTags_DECIMAL:
		return semtypes.DECIMAL
	default:
		t.internalError(fmt.Sprintf("unexpected type tag %v for numeric literal", n.GetValueType().BTypeGetTag()), n.GetPosition())
		return semtypes.NEVER
	}
}

func determineCandidatesFromNumericLiteral(t typeResolver, n *ast.BLangNumericLiteral) semtypes.SemType {
	switch n.Kind {
	case ast.NodeKind_INTEGER_LITERAL:
		return semtypes.NUMBER
	case ast.NodeKind_DECIMAL_FLOATING_POINT_LITERAL:
		if hasFloatTypeSuffix(n.OriginalValue) {
			return semtypes.FLOAT
		}
		if balCommon.IsDecimalDiscriminated(n.OriginalValue) {
			return semtypes.DECIMAL
		}
		return semtypes.Union(semtypes.FLOAT, semtypes.DECIMAL)
	case ast.NodeKind_HEX_FLOATING_POINT_LITERAL:
		return semtypes.FLOAT
	default:
		t.internalError(fmt.Sprintf("unexpected numeric literal kind: %v", n.Kind), n.GetPosition())
		return semtypes.NEVER
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
	case semtypes.ContainsBasicType(candidates, semtypes.INT):
		return resolveAsInt(t, n)
	case semtypes.ContainsBasicType(candidates, semtypes.FLOAT):
		return resolveAsFloat(t, n)
	case semtypes.ContainsBasicType(candidates, semtypes.DECIMAL):
		return resolveAsDecimal(t, n)
	default:
		t.semanticError("no valid candidate to resolve numeric literal", n.GetPosition())
		return semtypes.SemType{}, false
	}
}

func resolveAsInt(t typeResolver, n *ast.BLangLiteral) (semtypes.SemType, bool) {
	var intVal int64
	switch v := n.GetValue().(type) {
	case int64:
		intVal = v
	case float64:
		intVal = int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			t.syntaxError(fmt.Sprintf("invalid int literal: %s", v), n.GetPosition())
			return semtypes.SemType{}, false
		}
		intVal = parsed
	default:
		t.internalError(fmt.Sprintf("unexpected int literal value type: %T", n.GetValue()), n.GetPosition())
		return semtypes.SemType{}, false
	}
	n.SetValue(intVal)
	return semtypes.IntConst(intVal), true
}

func resolveAsFloat(t typeResolver, n *ast.BLangLiteral) (semtypes.SemType, bool) {
	var floatVal float64
	switch v := n.GetValue().(type) {
	case string:
		parsed, ok := parseFloatValue(t, v, n.GetPosition())
		if !ok {
			return semtypes.SemType{}, false
		}
		floatVal = parsed
	case float64:
		floatVal = v
	case int64:
		floatVal = float64(v)
	default:
		t.internalError(fmt.Sprintf("unexpected float literal value type: %T", v), n.GetPosition())
		return semtypes.SemType{}, false
	}
	n.SetValue(floatVal)
	return semtypes.FloatConst(floatVal), true
}

func resolveAsDecimal(t typeResolver, n *ast.BLangLiteral) (semtypes.SemType, bool) {
	var decVal *decimal.Decimal
	switch v := n.GetValue().(type) {
	case string:
		parsed, ok := parseDecimalValue(t, stripFloatingPointTypeSuffix(v), n.GetPosition())
		if !ok {
			return semtypes.SemType{}, false
		}
		decVal = parsed
	case *decimal.Decimal:
		decVal = v
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
		t.internalError(fmt.Sprintf("unexpected decimal literal value type: %T", v), n.GetPosition())
		return semtypes.SemType{}, false
	}
	n.SetValue(decVal)
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
		t.syntaxError(fmt.Sprintf("invalid float literal: %s", strValue), pos)
		return 0, false
	}
	return f, true
}

func parseDecimalValue(t typeResolver, strValue string, pos diagnostics.Location) (*decimal.Decimal, bool) {
	d, err := decimal.FromLiteral(strValue)
	if err != nil {
		t.syntaxError(fmt.Sprintf("invalid decimal literal: %s", strValue), pos)
		return decimal.FromInt64(0), false
	}
	return d, true
}

func resolveNumericLiteral(t typeResolver, n *ast.BLangNumericLiteral, expectedType semtypes.SemType) bool {
	candidates := determineCandidatesFromNumericLiteral(t, n)
	candidates = narrowCandidates(candidates, expectedType)

	ty, ok := pickNumericType(t, &n.BLangLiteral, candidates)
	if !ok {
		return false
	}

	setExpectedType(n, ty)
	updateSymbolType(t, n, ty)
	return true
}

// updateSymbolType updates the symbol's type if the node has an associated symbol.
func updateSymbolType(t typeResolver, node ast.BLangNode, ty semtypes.SemType) {
	if nodeWithSymbol, ok := node.(ast.BNodeWithSymbol); ok && ast.SymbolIsSet(nodeWithSymbol) {
		t.setSymbolType(nodeWithSymbol.Symbol(), ty)
	}
}

func lookupSymbol(chain *binding, ref model.SymbolRef) model.SymbolRef {
	if chain == nil {
		return ref
	}
	narrowedRef, isNarrowed, _ := lookupBinding(chain, ref)
	if isNarrowed {
		return narrowedRef
	}
	return ref
}

func resolveVariableDefStmt(t typeResolver, chain *binding, s *ast.BLangSimpleVariableDef) (statementEffect, bool) {
	variable := s.GetVariable().(*ast.BLangSimpleVariable)
	variable.Name.SetDeterminedType(semtypes.NEVER)
	typeNode := variable.TypeNode()
	if typeNode != nil {
		semType, ok := resolveBType(t, typeNode, 0)
		if !ok {
			setExpectedType(variable, semtypes.NEVER)
			updateSymbolType(t, variable, semtypes.NEVER)
			return defaultStmtEffect(chain), false
		}
		setExpectedType(variable, semType)
		updateSymbolType(t, variable, semType)
	}

	effectChain := chain
	if variable.Expr != nil {
		expectedType := variable.GetDeterminedType()
		exprTy, effect, ok := resolveActionOrExpression(t, chain, variable.Expr, expectedType)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		effectChain = mergeChains(t, effect.ifTrue, effect.ifFalse, semtypes.Union)
		if typeNode == nil {
			setExpectedType(variable, exprTy)
			updateSymbolType(t, variable, exprTy)
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
func detectGlobalVarInitCycles(t typeResolver, pkg *ast.BLangPackage) {
	if len(pkg.GlobalVars) == 0 {
		return
	}
	// Inferred-type globals are cycle-checked via ensureResolved during the
	// init-expression pass (the in-progress nil-marker pattern that constants
	// also use). Skip them here to avoid duplicate diagnostics.
	nodeSet := make(map[model.SymbolRef]int, len(pkg.GlobalVars))
	for i := range pkg.GlobalVars {
		if pkg.GlobalVars[i].TypeNode() == nil {
			continue
		}
		nodeSet[pkg.GlobalVars[i].Symbol()] = i
	}

	deps := make([][]int, len(pkg.GlobalVars))
	for i := range pkg.GlobalVars {
		gv := &pkg.GlobalVars[i]
		if gv.Expr == nil || gv.TypeNode() == nil {
			continue
		}
		v := &globalVarDepCollector{
			t:       t,
			nodeSet: nodeSet,
			deps:    make(map[int]struct{}),
		}
		ast.Walk(v, gv.Expr)
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
	state := make([]int, len(pkg.GlobalVars))

	var visit func(i int) bool
	visit = func(i int) bool {
		switch state[i] {
		case inStack:
			t.semanticError(
				fmt.Sprintf("invalid cycle detected for %s", pkg.GlobalVars[i].Name.GetValue()),
				pkg.GlobalVars[i].Name.GetPosition(),
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

	for i := range pkg.GlobalVars {
		if pkg.GlobalVars[i].TypeNode() == nil {
			continue
		}
		if !visit(i) {
			return
		}
	}
}

type globalVarDepCollector struct {
	t       typeResolver
	nodeSet map[model.SymbolRef]int // symbol → index into pkg.GlobalVars
	deps    map[int]struct{}
}

func (c *globalVarDepCollector) depends(ref model.SymbolRef) {
	unnarrowed := c.t.unnarrowedSymbol(ref)
	if idx, ok := c.nodeSet[unnarrowed]; ok {
		c.deps[idx] = struct{}{}
	}
}

func (c *globalVarDepCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangSimpleVarRef:
		c.depends(n.Symbol())
	case *ast.BLangConstRef:
		c.depends(n.Symbol())
	}
	return c
}

func (c *globalVarDepCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

func resolveGlobalVarType(t typeResolver, node *ast.BLangSimpleVariable) bool {
	node.Name.SetDeterminedType(semtypes.NEVER)
	typeNode := node.TypeNode()
	if typeNode == nil {
		if pt, ok := t.(*packageTypeResolver); ok {
			pt.inferredGlobalVarNodes[node.Symbol()] = node
		}
		return true
	}
	semType, ok := resolveBType(t, typeNode, 0)
	if !ok {
		setExpectedType(node, semtypes.NEVER)
		updateSymbolType(t, node, semtypes.NEVER)
		return false
	}
	setExpectedType(node, semType)
	updateSymbolType(t, node, semType)
	return true
}

func resolveGlobalVarInit(t typeResolver, node *ast.BLangSimpleVariable) bool {
	if node.Expr == nil {
		return true
	}
	if node.TypeNode() == nil {
		if pt, ok := t.(*packageTypeResolver); ok {
			return pt.ensureResolved(node.Symbol(), 0)
		}
		return resolveSimpleVariable(t, nil, node)
	}
	semType := node.GetDeterminedType()
	if semtypes.IsZero(semType) {
		return false
	}
	expectedType := semType
	if node.IsListener() {
		// A listener-decl is allowed to have an init expression whose type
		// includes error; module init performs the runtime `is error` check
		// and panics if the value is an error.
		expectedType = semtypes.Union(semType, semtypes.ERROR)
	}
	_, _, ok := resolveActionOrExpression(t, nil, node.Expr, expectedType)
	return ok
}

// resolveServiceAttachedExpressions type-checks the listener expressions in
// a service's `on` clause so subsequent validation can read each expression's
// determined type.
func resolveServiceAttachedExpressions(t typeResolver, svc *ast.BLangService) {
	for _, expr := range svc.AttachedExprs {
		resolveActionOrExpression(t, nil, expr, semtypes.SemType{})
	}
}

// validateListenerVars verifies each module-level listener variable's
// resolved type is a subtype of the global LISTENER top type. Reports a
// semantic error otherwise.
func validateListenerVars(t typeResolver, pkg *ast.BLangPackage, attachPointBound semtypes.SemType) {
	tyCtx := t.typeContext()
	for i := range pkg.GlobalVars {
		gv := &pkg.GlobalVars[i]
		if !gv.IsListener() {
			continue
		}
		ty := gv.GetDeterminedType()
		if semtypes.IsZero(ty) {
			t.internalError("listener variable has no determined type", gv.GetPosition())
			continue
		}
		if _, _, ok := validateListenerType(tyCtx, ty, attachPointBound); !ok {
			t.semanticError("listener initializer is not a listener", gv.GetPosition())
		}
	}
}

// validateServiceDeclaration implements the type-resolver rules from the
// service/listener design: the `on` expression list must consist of
// listener variables, the attach-point type must be a subtype of the
// listeners' attach-point union, and the service body must be a subtype
// of the listeners' target object union (or the user-supplied service
// type when present).
func validateServiceDeclaration(t typeResolver, svc *ast.BLangService, attachPointBound semtypes.SemType) {
	tyCtx := t.typeContext()

	var expectedT semtypes.SemType
	var expectedA semtypes.SemType
	for i, expr := range svc.AttachedExprs {
		targetTy, attachTy, ok := validateListenerOnExpression(t, expr, attachPointBound)
		if !ok {
			return
		}
		if i == 0 {
			expectedT = targetTy
			expectedA = attachTy
			continue
		}
		expectedT = semtypes.Intersect(expectedT, targetTy)
		expectedA = semtypes.Intersect(expectedA, attachTy)
	}

	attachPointTy := serviceAttachPointType(t, svc)
	if !semtypes.IsSubtype(tyCtx, attachPointTy, expectedA) {
		t.semanticError("attach point is not assignable to listener's attach-point type", svc.GetPosition())
	}

	bodyTy := svc.Definition.GetSemType(t.typeEnv())
	if !semtypes.IsSubtype(tyCtx, bodyTy, expectedT) {
		t.semanticError("service body is not a subtype of the listener's expected service type", svc.GetPosition())
	}
}

func validateListenerOnExpression(t typeResolver, expr ast.BLangExpression, attachPointBound semtypes.SemType) (semtypes.SemType, semtypes.SemType, bool) {
	exprTy := expr.GetDeterminedType()
	if semtypes.IsZero(exprTy) {
		t.internalError("listener expression has no determined type", expr.GetPosition())
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	checkedTy := semtypes.Diff(exprTy, semtypes.ERROR)
	targetTy, attachTy, ok := validateListenerType(t.typeContext(), checkedTy, attachPointBound)
	if !ok {
		t.semanticError("expression in 'on' clause is not a listener", expr.GetPosition())
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	return targetTy, attachTy, true
}

func serviceAttachPointType(t typeResolver, svc *ast.BLangService) semtypes.SemType {
	if svc.AttachPointLiteral != nil {
		if !resolveLiteral(t, svc.AttachPointLiteral, semtypes.STRING) {
			return semtypes.NEVER
		}
		return svc.AttachPointLiteral.GetDeterminedType()
	}
	if len(svc.AbsoluteResourcePath) == 0 {
		return semtypes.NIL
	}
	segmentTypes := make([]semtypes.SemType, len(svc.AbsoluteResourcePath))
	for i := range svc.AbsoluteResourcePath {
		segmentTypes[i] = semtypes.StringConst(svc.AbsoluteResourcePath[i].Value)
	}
	listDefn := semtypes.NewListDefinition()
	return listDefn.DefineListTypeWrapped(t.typeEnv(), segmentTypes, len(segmentTypes), semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
}

func resolveSimpleVariable(t typeResolver, chain *binding, node *ast.BLangSimpleVariable) bool {
	node.Name.SetDeterminedType(semtypes.NEVER)
	typeNode := node.TypeNode()
	if typeNode == nil {
		if node.Expr != nil {
			exprTy, _, ok := resolveActionOrExpression(t, chain, node.Expr, semtypes.SemType{})
			if !ok {
				return false
			}
			setExpectedType(node, exprTy)
			updateSymbolType(t, node, exprTy)
		}
		return true
	}

	semType, ok := resolveBType(t, typeNode, 0)
	if !ok {
		setExpectedType(node, semtypes.NEVER)
		updateSymbolType(t, node, semtypes.NEVER)
		return false
	}

	setExpectedType(node, semType)
	updateSymbolType(t, node, semType)

	if node.Expr != nil {
		if _, _, ok := resolveActionOrExpression(t, chain, node.Expr, semType); !ok {
			return false
		}
	}

	return true
}

func resolveActionOrExpression(t typeResolver, chain *binding, expr ast.BLangActionOrExpression, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	// Check if already resolved
	if ty := expr.GetDeterminedType(); !semtypes.IsZero(ty) {
		return ty, defaultExpressionEffect(chain), true
	}

	ty, effect, ok := resolveExpressionInner(t, chain, expr, expectedType)
	if !ok {
		// Mark failed expressions so ast.Walk won't re-process them
		setExpectedType(expr, semtypes.NEVER)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if singletonEffect, isSingleton := singletonExprEffect(chain, expr); isSingleton {
		return ty, singletonEffect, true
	}
	return ty, effect, ok
}

func resolveExpressionInner(t typeResolver, chain *binding, expr ast.BLangActionOrExpression, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	switch e := expr.(type) {
	case *ast.BLangLiteral:
		if ok := resolveLiteral(t, e, expectedType); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		return e.GetDeterminedType(), defaultExpressionEffect(chain), true
	case *ast.BLangNumericLiteral:
		if ok := resolveNumericLiteral(t, e, expectedType); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		return e.GetDeterminedType(), defaultExpressionEffect(chain), true
	case *ast.BLangSimpleVarRef:
		return resolveSimpleVarRef(t, chain, e)
	case *ast.BLangLocalVarRef:
		return resolveLocalVarRef(t, chain, e)
	case *ast.BLangConstRef:
		return resolveConstRef(t, chain, e)
	case *ast.BLangBinaryExpr:
		return resolveBinaryExpr(t, chain, e, expectedType)
	case *ast.BLangUnaryExpr:
		return resolveUnaryExpr(t, chain, e, expectedType)
	case *ast.BLangInvocation:
		return resolveInvocation(t, chain, e, expectedType)
	case *ast.BLangIndexBasedAccess:
		return resolveIndexBasedAccess(t, chain, e)
	case *ast.BLangFieldBaseAccess:
		return resolveFieldBaseAccess(t, chain, e)
	case *ast.BLangListConstructorExpr:
		return resolveListConstructorExpr(t, chain, e, expectedType)
	case *ast.BLangMappingConstructorExpr:
		return resolveMappingConstructorExpr(t, chain, e, expectedType)
	case *ast.BLangErrorConstructorExpr:
		return resolveErrorConstructorExpr(t, chain, e, expectedType)
	case *ast.BLangGroupExpr:
		return resolveGroupExpr(t, chain, e, expectedType)
	case *ast.BLangQueryExpr:
		return resolveQueryExpr(t, chain, e, expectedType)
	case *ast.BLangWildCardBindingPattern:
		ty := semtypes.ANY
		setExpectedType(e, ty)
		return ty, defaultExpressionEffect(chain), true
	case *ast.BLangTypeConversionExpr:
		return resolveTypeConversionExpr(t, chain, e)
	case *ast.BLangTypeTestExpr:
		return resolveTypeTestExpr(t, chain, e)
	case *ast.BLangTypedescExpr:
		return resolveTypedescExpr(t, chain, e)
	case *ast.BLangAnnotAccessExpr:
		return resolveAnnotAccessExpr(t, chain, e)
	case *ast.BLangCheckedExpr:
		return resolveCheckedExpr(t, chain, e, expectedType)
	case *ast.BLangCheckPanickedExpr:
		return resolveCheckedExpr(t, chain, &e.BLangCheckedExpr, expectedType)
	case *ast.BLangTrapExpr:
		return resolveTrapExpr(t, chain, e)
	case *ast.BLangNamedArgsExpression:
		ty, effect, ok := resolveActionOrExpression(t, chain, e.Expr, expectedType)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		setExpectedType(e, ty)
		e.Name.SetDeterminedType(semtypes.NEVER)
		return ty, effect, true
	case *ast.BLangNewExpression:
		return resolveNewExpr(t, chain, e, expectedType)
	case *ast.BLangLambdaFunction:
		return resolveLambdaFunctionExpr(t, chain, e)
	case *ast.BLangRemoteMethodCallAction:
		return resolveRemoteMethodCallAction(t, chain, e, expectedType)
	case *ast.BLangClientResourceAccessAction:
		return resolveClientResourceAccessAction(t, chain, e, expectedType)
	case *ast.BLangInferredTypedescDefault:
		return resolveInferredTypedescDefault(t, chain, e, expectedType)
	case *ast.BLangXMLSequenceLiteral:
		return resolveXMLSequenceLiteral(t, chain, e, expectedType)
	case *ast.BLangTemplateExpr:
		return resolveTemplateExpr(t, chain, e)
	case *ast.BLangXMLTemplateExpr:
		return resolveXMLTemplateExpr(t, chain, e)
	case *ast.BLangXMLElementLiteral:
		return resolveXMLElementLiteral(t, chain, e)
	case *ast.BLangXMLPILiteral:
		return resolveXMLPILiteral(t, chain, e)
	case *ast.BLangXMLCommentLiteral:
		return resolveXMLCommentLiteral(t, chain, e)
	case *ast.BLangXMLTextLiteral:
		return resolveXMLTextLiteral(t, chain, e)
	default:
		t.internalError(fmt.Sprintf("unsupported expression type: %T", expr), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
}

// resolveInferredTypedescDefault handles the "<>" default value that appears as
// the initializer of a dependently-typed function's typedesc parameter.
//
// When encountered as the parameter's own default initializer (expectedType is
// the parameter's declared typedesc type) it just adopts that type. When it is
// synthesized into a call-site argument list, expectedType is the inferred
// typedesc<T>. In either case the determined type becomes expectedType.
func resolveInferredTypedescDefault(t typeResolver, chain *binding, e *ast.BLangInferredTypedescDefault, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if semtypes.IsZero(expectedType) || !semtypes.IsSubtype(t.typeContext(), expectedType, semtypes.TYPEDESC) {
		t.semanticError("inferred typedesc default '<>' is only allowed as the default for a typedesc parameter", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(e, expectedType)
	return expectedType, defaultExpressionEffect(chain), true
}

func resolveTypedescExpr(t typeResolver, chain *binding, e *ast.BLangTypedescExpr) (semtypes.SemType, expressionEffect, bool) {
	typeDesc := e.GetTypeDescriptor()
	if typeDesc == nil {
		t.internalError("typedesc expression has no type descriptor", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	constraint, ok := resolveBType(t, typeDesc.(ast.BType), 0)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	e.Constraint = constraint
	e.AnnotationValues = annotationValuesForTypeDescriptor(t, typeDesc)
	ty := semtypes.TypedescContaining(t.typeEnv(), constraint)
	setExpectedType(e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveAnnotAccessExpr(t typeResolver, chain *binding, e *ast.BLangAnnotAccessExpr) (semtypes.SemType, expressionEffect, bool) {
	receiverTy, effect, ok := resolveActionOrExpression(t, chain, e.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if !semtypes.IsSubtype(t.typeContext(), receiverTy, semtypes.TYPEDESC) {
		t.semanticError("annotation access is only allowed on typedesc values", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	sym, ok := t.getSymbol(e.Symbol()).(*model.AnnotationSymbol)
	if !ok {
		t.internalError("annotation access does not resolve to an annotation symbol", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	annTy := sym.Type()
	if semtypes.IsZero(annTy) {
		t.internalError("annotation type is not resolved", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	ty := semtypes.Union(annTy, semtypes.NIL)
	setExpectedType(e, ty)
	if e.PkgAlias != nil {
		setOtherNodesAsNever(e.PkgAlias)
	}
	if e.AnnotationName != nil {
		setOtherNodesAsNever(e.AnnotationName)
	}
	return ty, effect, true
}

func annotationValuesForTypeDescriptor(t typeResolver, typeDesc ast.TypeDescriptor) values.AnnotationValues {
	udt, ok := typeDesc.(*ast.BLangUserDefinedType)
	if !ok || !ast.SymbolIsSet(udt) {
		return values.NewAnnotationValues()
	}
	return t.compilerContext().SymbolAnnotationValues(udt.Symbol())
}

func resolveXMLTextLiteral(_ typeResolver, chain *binding, e *ast.BLangXMLTextLiteral) (semtypes.SemType, expressionEffect, bool) {
	ty := semtypes.XML_TEXT
	setExpectedType(e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveXMLCommentLiteral(_ typeResolver, chain *binding, e *ast.BLangXMLCommentLiteral) (semtypes.SemType, expressionEffect, bool) {
	ty := semtypes.XML_COMMENT
	setExpectedType(e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveXMLPILiteral(_ typeResolver, chain *binding, e *ast.BLangXMLPILiteral) (semtypes.SemType, expressionEffect, bool) {
	ty := semtypes.XML_PI
	setExpectedType(e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveXMLElementLiteral(t typeResolver, chain *binding, e *ast.BLangXMLElementLiteral) (semtypes.SemType, expressionEffect, bool) {
	for i := range e.Attrs {
		attr := &e.Attrs[i]
		if attr.Value != nil {
			if _, _, ok := resolveActionOrExpression(t, chain, attr.Value, semtypes.STRING); !ok {
				return semtypes.SemType{}, expressionEffect{}, false
			}
		}
		attr.SetDeterminedType(semtypes.NEVER)
	}
	if e.Content != nil {
		if _, _, ok := resolveActionOrExpression(t, chain, e.Content, semtypes.XML); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}
	ty := semtypes.XML_ELEMENT
	setExpectedType(e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveTemplateExpr(t typeResolver, chain *binding, e *ast.BLangTemplateExpr) (semtypes.SemType, expressionEffect, bool) {
	var ty semtypes.SemType
	if len(e.Insertions) == 0 {
		ty = semtypes.StringConst(e.Strings[0])
	} else {
		var ok bool
		ty, ok = resolveStringTemplateType(t, chain, e)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}
	setExpectedType(e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveStringTemplateType(t typeResolver, chain *binding, e *ast.BLangTemplateExpr) (semtypes.SemType, bool) {
	allSingleton := true
	var sb strings.Builder
	sb.WriteString(e.Strings[0])
	for i, ins := range e.Insertions {
		insTy, _, ok := resolveActionOrExpression(t, chain, ins, templateInsertionAllowedTypes)
		if !ok {
			return semtypes.SemType{}, false
		}
		if allSingleton && semtypes.IsSubtypeSimple(insTy, semtypes.STRING) {
			if shape := semtypes.SingleShape(insTy); !shape.IsEmpty() {
				sb.WriteString(shape.Get().Value.(string))
				sb.WriteString(e.Strings[i+1])
				continue
			}
		}
		allSingleton = false
	}
	if allSingleton {
		return semtypes.StringConst(sb.String()), true
	}
	return semtypes.STRING, true
}

func resolveXMLTemplateExpr(t typeResolver, chain *binding, e *ast.BLangXMLTemplateExpr) (semtypes.SemType, expressionEffect, bool) {
	if len(e.InsertionKinds) != len(e.Insertions) {
		t.internalError(fmt.Sprintf("xml template insertion kind count mismatch: got %d kinds for %d insertions", len(e.InsertionKinds), len(e.Insertions)), e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	for i, ins := range e.Insertions {
		allowed := xmlTemplateInsertionAllowedTypes(e.InsertionKinds[i])
		if _, _, ok := resolveActionOrExpression(t, chain, ins, allowed); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}
	setExpectedType(e, semtypes.XML)
	return semtypes.XML, defaultExpressionEffect(chain), true
}

func resolveXMLSequenceLiteral(t typeResolver, chain *binding, e *ast.BLangXMLSequenceLiteral, _ semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	childUnion := semtypes.NEVER
	for _, child := range e.Children {
		childTy, _, ok := resolveActionOrExpression(t, chain, child, semtypes.XML)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if !semtypes.IsSubtype(t.typeContext(), childTy, semtypes.XML) {
			t.semanticError(fmt.Sprintf("expected xml value, got %s", semtypes.ToString(t.typeContext(), childTy)), child.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		childUnion = semtypes.Union(childUnion, childTy)
	}
	ty := semtypes.XMLSequence(childUnion)
	setExpectedType(e, ty)
	return ty, defaultExpressionEffect(chain), true
}

func resolveNewExpr(t typeResolver, chain *binding, e *ast.BLangNewExpression, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	cx := t.typeContext()
	var determinedTy semtypes.SemType
	if e.TypeDescriptor != nil {
		resolvedTy, ok := resolveBType(t, e.TypeDescriptor, 0)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		determinedTy = resolvedTy
	} else {
		if semtypes.IsZero(expectedType) {
			t.semanticError("cannot infer type for implicit new expression", e.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		intersection := semtypes.Intersect(expectedType, semtypes.Union(semtypes.OBJECT, semtypes.STREAM))
		if semtypes.IsEmpty(cx, intersection) {
			t.semanticError("expected type is not an object or stream type", e.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		determinedTy = intersection
	}
	setExpectedType(e, determinedTy)

	switch {
	case semtypes.IsSubtypeSimple(determinedTy, semtypes.OBJECT):
		return resolveObjectNewExpr(t, chain, e, determinedTy)
	case semtypes.IsSubtypeSimple(determinedTy, semtypes.STREAM):
		return resolveStreamNewExpr(t, chain, e, determinedTy)
	default:
		t.semanticError("new expression target must be either an object or stream type", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
}

func resolveObjectNewExpr(t typeResolver, chain *binding, e *ast.BLangNewExpression, determinedTy semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	cx := t.typeContext()
	initKey := semtypes.StringConst("init")
	initFnTy := semtypes.ObjectMemberType(cx, initKey, determinedTy)
	var paramListTy semtypes.SemType
	if !semtypes.IsZero(initFnTy) {
		paramListTy = semtypes.FunctionParamListType(cx, initFnTy)
	}
	for i, arg := range e.ArgsExprs {
		var paramTy semtypes.SemType
		if !semtypes.IsZero(paramListTy) {
			key := semtypes.IntConst(int64(i))
			paramTy = semtypes.ListMemberTypeInnerVal(cx, paramListTy, key)
		}
		if _, _, ok := resolveActionOrExpression(t, chain, arg, paramTy); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}

	argTys := make([]semtypes.SemType, len(e.ArgsExprs))
	for i, arg := range e.ArgsExprs {
		argTys[i] = arg.GetDeterminedType()
	}
	objTy, ok := determineObjectType(t, e, argTys, determinedTy)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	atomicType := semtypes.ToObjectAtomicType(cx, objTy)
	if atomicType == nil {
		t.semanticError("non atomic object types not supported", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	e.AtomicType = atomicType

	classSymbol, found := t.getClassAtomSymbol(atomicType)
	if !found {
		t.internalError("failed to find class definition for object type", e.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	e.ClassSymbol = classSymbol

	return e.GetDeterminedType(), defaultExpressionEffect(chain), true
}

// padNewExprArgTypesForDefaults pads argTys with the init method's default param types for
// any trailing params omitted by the caller. Returns the padded slice and true when the init
// was resolved; returns (argTys, false) unchanged when the class or init cannot be found.
func padNewExprArgTypesForDefaults(t typeResolver, objectTy semtypes.SemType, argTys []semtypes.SemType, loc diagnostics.Location) ([]semtypes.SemType, bool) {
	oat := semtypes.ToObjectAtomicType(t.typeContext(), objectTy)
	if oat == nil {
		return argTys, false
	}
	classRef, ok := t.getClassAtomSymbol(oat)
	if !ok {
		return argTys, false
	}
	classSym, ok := t.getSymbol(classRef).(model.ClassSymbol)
	if !ok {
		return argTys, false
	}
	initRef, ok := classSym.MethodSymbol("init")
	if !ok {
		return argTys, false
	}
	return padArgTypesForDefaults(t, initRef, argTys, loc), true
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
	if _, _, ok := resolveActionOrExpression(t, chain, e.ArgsExprs[0], implTy); !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	e.SetDeterminedType(streamTy)
	return streamTy, defaultExpressionEffect(chain), true
}

func determineObjectType(t typeResolver, expr *ast.BLangNewExpression, argTys []semtypes.SemType, objectTy semtypes.SemType) (semtypes.SemType, bool) {
	cx := t.typeContext()
	alts := semtypes.ObjectAlternatives(cx, objectTy)

	type candidate struct {
		objType        semtypes.SemType
		initReturnType semtypes.SemType
	}
	var candidates []candidate
	for _, alt := range alts {
		altArgTys, _ := padNewExprArgTypesForDefaults(t, alt.ObjectType, argTys, expr.GetPosition())
		argLd := semtypes.NewListDefinition()
		altArgListTy := argLd.DefineListTypeWrapped(cx.Env(), altArgTys, len(altArgTys), semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
		paramListTy := semtypes.FunctionParamListType(cx, alt.InitFnType)
		if semtypes.IsSubtype(cx, altArgListTy, paramListTy) {
			retTy := semtypes.FunctionReturnType(cx, alt.InitFnType, altArgListTy)
			candidates = append(candidates, candidate{objType: alt.ObjectType, initReturnType: retTy})
		}
	}
	if len(candidates) == 0 {
		t.semanticError("failed to determine object type with fitting init function", expr.GetPosition())
		return semtypes.SemType{}, false
	} else if len(candidates) > 1 {
		t.semanticError("ambiguous object type", expr.GetPosition())
		return semtypes.SemType{}, false
	}
	resultObjType := candidates[0].objType
	if semtypes.IsSubtype(cx, objectTy, candidates[0].objType) {
		resultObjType = objectTy
	}
	expr.SetDeterminedType(semtypes.Union(resultObjType, semtypes.Diff(candidates[0].initReturnType, semtypes.NIL)))
	return resultObjType, true
}

func resolveTypeTestExpr(t typeResolver, chain *binding, e *ast.BLangTypeTestExpr) (semtypes.SemType, expressionEffect, bool) {
	exprTy, _, ok := resolveActionOrExpression(t, chain, e.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resolveTypeData(t, &e.Type)
	if e.Type.TypeDescriptor != nil {
		if tdNode, ok := e.Type.TypeDescriptor.(ast.BLangNode); ok {
			setOtherNodesAsNever(tdNode)
		}
	}
	testedTy := e.Type.Type

	var resultTy semtypes.SemType
	if semtypes.IsSubtype(t.typeContext(), exprTy, testedTy) {
		resultTy = semtypes.BooleanConst(!e.IsNegation())
	} else if semtypes.IsEmpty(t.typeContext(), semtypes.Intersect(exprTy, testedTy)) {
		resultTy = semtypes.BooleanConst(e.IsNegation())
	} else {
		resultTy = semtypes.BOOLEAN
	}

	setExpectedType(e, resultTy)

	ref, isVarRef := varRefExp(chain, e.Expr)
	if !isVarRef {
		return resultTy, defaultExpressionEffect(chain), true
	}
	tx := t.symbolType(ref)
	ref = t.unnarrowedSymbol(ref)
	testTy := e.Type.Type
	trueTy := semtypes.Intersect(tx, testTy)
	trueSym := narrowSymbol(t, ref, trueTy)
	trueChain := &binding{ref: ref, narrowedSymbol: trueSym, prev: chain}
	falseTy := semtypes.Diff(tx, testTy)
	falseSym := narrowSymbol(t, ref, falseTy)
	falseChain := &binding{ref: ref, narrowedSymbol: falseSym, prev: chain}
	if e.IsNegation() {
		return resultTy, expressionEffect{ifTrue: falseChain, ifFalse: trueChain}, true
	}
	return resultTy, expressionEffect{ifTrue: trueChain, ifFalse: falseChain}, true
}

func resolveTrapExpr(t typeResolver, chain *binding, e *ast.BLangTrapExpr) (semtypes.SemType, expressionEffect, bool) {
	exprTy, _, ok := resolveActionOrExpression(t, chain, e.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Union(exprTy, semtypes.ERROR)
	setExpectedType(e, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveCheckedExpr(t typeResolver, chain *binding, e *ast.BLangCheckedExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	var innerExpected semtypes.SemType
	if !semtypes.IsZero(expectedType) {
		innerExpected = semtypes.Union(expectedType, semtypes.ERROR)
	}
	exprTy, _, ok := resolveActionOrExpression(t, chain, e.Expr, innerExpected)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Diff(exprTy, semtypes.ERROR)
	setExpectedType(e, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveMappingConstructorExpr(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if !semtypes.IsZero(expectedType) {
		return resolveMappingConstructorWithExpectedType(t, chain, e, expectedType)
	}
	return resolveMappingConstructorBottomUp(t, chain, e)
}

func resolveMappingConstructorBottomUp(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr) (semtypes.SemType, expressionEffect, bool) {
	fields := make([]semtypes.Field, len(e.Fields))
	for i, f := range e.Fields {
		kv := f.(*ast.BLangMappingKeyValueField)
		valueTy, _, ok := resolveActionOrExpression(t, chain, kv.ValueExpr, semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		var broadTy semtypes.SemType
		if semtypes.SingleShape(valueTy).IsEmpty() {
			broadTy = valueTy
		} else {
			broadTy = semtypes.WidenToBasicTypes(valueTy)
		}
		var keyName string
		switch keyExpr := kv.Key.Expr.(type) {
		case *ast.BLangLiteral:
			keyName = keyExpr.Value.(string)
			resolveLiteral(t, keyExpr, semtypes.SemType{})
		case ast.BNodeWithSymbol:
			t.setSymbolType(keyExpr.Symbol(), valueTy)
			keyName = t.symbolName(keyExpr.Symbol())
			if e, ok := keyExpr.(ast.BLangExpression); ok {
				setExpectedType(e, valueTy)
			}
			if ref, ok := keyExpr.(*ast.BLangSimpleVarRef); ok {
				setVarRefIdentifierTypes(ref)
			}
		}
		kv.Key.SetDeterminedType(semtypes.NEVER)
		kv.SetDeterminedType(semtypes.NEVER)
		fields[i] = semtypes.FieldFrom(keyName, broadTy, false, false)
	}
	md := semtypes.NewMappingDefinition()
	mapTy := md.DefineMappingTypeWrapped(t.typeEnv(), fields, semtypes.NEVER)
	setExpectedType(e, mapTy)
	mat := semtypes.ToMappingAtomicType(t.typeContext(), mapTy)
	e.AtomicType = *mat
	return mapTy, defaultExpressionEffect(chain), true
}

func resolveMappingConstructorWithExpectedType(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	for _, f := range e.Fields {
		kv := f.(*ast.BLangMappingKeyValueField)
		if _, _, ok := resolveActionOrExpression(t, chain, kv.ValueExpr, semtypes.SemType{}); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		resolveMappingKey(t, kv)
	}

	resultType, mat, ok := selectMappingInherentType(t, e, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	for _, f := range e.Fields {
		kv := f.(*ast.BLangMappingKeyValueField)
		keyName := recordKeyName(kv.Key)
		requiredType := mat.FieldInnerVal(keyName)
		kv.ValueExpr.SetDeterminedType(semtypes.SemType{})
		if _, _, ok := resolveActionOrExpression(t, chain, kv.ValueExpr, requiredType); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}

	e.AtomicType = *mat
	if ref, ok := t.getMappingAtomSymRef(mat); ok {
		if carrier, ok := t.getSymbol(ref).(model.MemberCarrier); ok {
			e.FieldDefaults = carrier.FieldDefaults()
		}
	} else if bType, ok := t.getMappingAtomBType(mat); ok {
		// This happens for inline record type definitions given they don't have type symbol. Need to think of a way
		// to properly handle this
		if recTy, ok := bType.(*ast.BLangRecordType); ok {
			for name, field := range recTy.Fields() {
				if field.DefaultExpr != nil {
					e.FieldDefaults = append(e.FieldDefaults, model.FieldDefault{FieldName: name, FnRef: field.DefaultFnRef})
				}
			}
		}
	}
	setExpectedType(e, resultType)
	return resultType, defaultExpressionEffect(chain), true
}

func resolveMappingKey(t typeResolver, kv *ast.BLangMappingKeyValueField) {
	switch keyExpr := kv.Key.Expr.(type) {
	case *ast.BLangLiteral:
		resolveLiteral(t, keyExpr, semtypes.SemType{})
	case ast.BNodeWithSymbol:
		valueTy := kv.ValueExpr.GetDeterminedType()
		t.setSymbolType(keyExpr.Symbol(), valueTy)
		if e, ok := keyExpr.(ast.BLangExpression); ok {
			setExpectedType(e, valueTy)
		}
		if ref, ok := keyExpr.(*ast.BLangSimpleVarRef); ok {
			setVarRefIdentifierTypes(ref)
		}
	}
	kv.Key.SetDeterminedType(semtypes.NEVER)
	kv.SetDeterminedType(semtypes.NEVER)
}

func selectMappingInherentType(t typeResolver, expr *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, *semtypes.MappingAtomicType, bool) {
	expectedMappingType := semtypes.Intersect(expectedType, semtypes.MAPPING)
	tc := t.typeContext()
	if semtypes.IsEmpty(tc, expectedMappingType) {
		t.semanticError("mapping type not found in expected type", expr.GetPosition())
		return semtypes.SemType{}, nil, false
	}
	mat := semtypes.ToMappingAtomicType(tc, expectedMappingType)
	if mat != nil {
		return expectedMappingType, mat, true
	}
	alts := semtypes.MappingAlternatives(tc, expectedType)
	var validAlts []semtypes.MappingAlternative

	fields := make([]semtypes.MappingFieldInfo, len(expr.Fields))
	for i, f := range expr.Fields {
		kv := f.(*ast.BLangMappingKeyValueField)
		fields[i] = semtypes.MappingFieldInfo{Name: recordKeyName(kv.Key), Ty: kv.ValueExpr.GetDeterminedType()}
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

	for _, alt := range alts {
		if semtypes.MappingAlternativeAllowsFields(tc, alt, fields) {
			validAlts = append(validAlts, alt)
		}
	}
	if len(validAlts) == 0 {
		t.semanticError("no applicable inherent type for mapping constructor", expr.GetPosition())
		return semtypes.SemType{}, nil, false
	}
	if len(validAlts) > 1 {
		t.semanticError("ambiguous inherent type for mapping constructor", expr.GetPosition())
		return semtypes.SemType{}, nil, false
	}

	selectedSemType := validAlts[0].SemType
	mat = semtypes.ToMappingAtomicType(tc, selectedSemType)
	if mat == nil {
		t.semanticError("applicable type for mapping constructor is not atomic", expr.GetPosition())
		return semtypes.SemType{}, nil, false
	}

	return selectedSemType, mat, true
}

func resolveTypeConversionExpr(t typeResolver, chain *binding, e *ast.BLangTypeConversionExpr) (semtypes.SemType, expressionEffect, bool) {
	expectedType, ok := resolveBType(t, e.TypeDescriptor, 0)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	_, _, ok = resolveActionOrExpression(t, chain, e.Expression, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	setExpectedType(e, expectedType)
	return expectedType, defaultExpressionEffect(chain), true
}

// Helper functions for expression type checking

func setVarRefIdentifierTypes(ref *ast.BLangSimpleVarRef) {
	if ref.PkgAlias != nil {
		ref.PkgAlias.SetDeterminedType(semtypes.NEVER)
	}
	if ref.VariableName != nil {
		ref.VariableName.SetDeterminedType(semtypes.NEVER)
	}
}

type opExpr interface {
	GetOperatorKind() model.OperatorKind
}

func isEqualityExpr(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_EQUAL, model.OperatorKind_EQUALS, model.OperatorKind_NOT_EQUAL, model.OperatorKind_REF_EQUAL, model.OperatorKind_REF_NOT_EQUAL:
		return true
	default:
		return false
	}
}

func isMultiplicativeExpr(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_MUL, model.OperatorKind_DIV, model.OperatorKind_MOD:
		return true
	default:
		return false
	}
}

func isRangeExpr(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_CLOSED_RANGE, model.OperatorKind_HALF_OPEN_RANGE:
		return true
	default:
		return false
	}
}

func isBitWiseExpr(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		return true
	default:
		return false
	}
}

func isShiftExpr(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_BITWISE_LEFT_SHIFT,
		model.OperatorKind_BITWISE_RIGHT_SHIFT,
		model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return true
	default:
		return false
	}
}

func isRelationalExpr(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_LESS_THAN, model.OperatorKind_LESS_EQUAL, model.OperatorKind_GREATER_THAN, model.OperatorKind_GREATER_EQUAL:
		return true
	default:
		return false
	}
}

func isAdditiveExpr(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_ADD, model.OperatorKind_SUB:
		return true
	default:
		return false
	}
}

func isLogicalExpression(opExpr opExpr) bool {
	switch opExpr.GetOperatorKind() {
	case model.OperatorKind_AND, model.OperatorKind_OR:
		return true
	default:
		return false
	}
}

func isNumericType(cx semtypes.Context, ty semtypes.SemType) bool {
	return semtypes.IsSubtype(cx, ty, semtypes.NUMBER)
}

func resolveGroupExpr(t typeResolver, chain *binding, expr *ast.BLangGroupExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	innerTy, effect, ok := resolveActionOrExpression(t, chain, expr.Expression, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(expr, innerTy)
	return innerTy, effect, true
}

func resolveQueryExpr(
	t typeResolver,
	chain *binding,
	expr *ast.BLangQueryExpr,
	expectedType semtypes.SemType,
) (semtypes.SemType, expressionEffect, bool) {
	if len(expr.QueryClauseList) < 2 {
		t.semanticError("query expression requires from and select clauses", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	fromClause, ok := expr.QueryClauseList[0].(*ast.BLangFromClause)
	if !ok {
		t.semanticError("query expression must start with a from clause", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	fromClause.SetDeterminedType(semtypes.NEVER)

	lastClauseIndex := len(expr.QueryClauseList) - 1
	var onConflictClause *ast.BLangOnConflictClause
	if clause, isOnConflict := expr.QueryClauseList[lastClauseIndex].(*ast.BLangOnConflictClause); isOnConflict {
		onConflictClause = clause
		onConflictClause.SetDeterminedType(semtypes.NEVER)
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
	if selectClause, finalOK = expr.QueryClauseList[lastClauseIndex].(*ast.BLangSelectClause); finalOK {
		selectClause.SetDeterminedType(semtypes.NEVER)
	} else if collectClause, finalOK = expr.QueryClauseList[lastClauseIndex].(*ast.BLangCollectClause); finalOK {
		collectClause.SetDeterminedType(semtypes.NEVER)
	} else {
		t.semanticError("query expression requires a select or collect clause", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	collectionTy, _, ok := resolveActionOrExpression(t, chain, fromClause.Collection, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
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
		varDef.SetDeterminedType(semtypes.NEVER)

		variableTy := elementTy
		if !fromClause.IsDeclaredWithVarFlag && varDef.Var.TypeNode() != nil {
			variableTy, ok = resolveBType(t, varDef.Var.TypeNode(), 0)
			if !ok {
				return semtypes.SemType{}, expressionEffect{}, false
			}
			if !semtypes.IsSubtype(t.typeContext(), elementTy, variableTy) {
				t.semanticError("from clause variable type is incompatible with collection member type",
					varDef.GetPosition())
				return semtypes.SemType{}, expressionEffect{}, false
			}
		}

		if varDef.Var.Name != nil {
			varDef.Var.Name.SetDeterminedType(semtypes.NEVER)
		}
		varDef.Var.SetDeterminedType(semtypes.NEVER)
		updateSymbolType(t, varDef.Var, variableTy)
	}

	queryChain, ok := resolveQueryIntermediateClauses(t, chain, expr, lastClauseIndex)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	var queryTy semtypes.SemType
	if selectClause != nil {
		selectExpectedTy := querySelectExpectedType(
			t.typeContext(),
			t.typeEnv(),
			expr.QueryConstructType,
			expectedType,
		)
		selectTy, _, ok := resolveActionOrExpression(t, queryChain, selectClause.Expression, selectExpectedTy)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		switch expr.QueryConstructType {
		case ast.TypeKind_NONE:
			ld := semtypes.NewListDefinition()
			queryTy = ld.DefineListTypeWrappedWithEnvSemType(t.typeEnv(), selectTy)
		case ast.TypeKind_MAP:
			expectedSelectTy := mapQuerySelectExpectedType(t.typeEnv())
			if !semtypes.IsSubtype(t.typeContext(), selectTy, expectedSelectTy) {
				t.semanticError(
					formatIncompatibleTypeMessage(t.typeContext(), expectedSelectTy, selectTy),
					selectClause.GetPosition(),
				)
				return semtypes.SemType{}, expressionEffect{}, false
			}
			valueTy := semtypes.ListMemberTypeInnerVal(t.typeContext(), selectTy, semtypes.IntConst(1))
			md := semtypes.NewMappingDefinition()
			queryTy = md.DefineMappingTypeWrapped(t.typeEnv(), nil, valueTy)
		default:
			t.unimplemented("query construct type is not supported yet", expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
	} else {
		if expr.QueryConstructType != ast.TypeKind_NONE {
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
		collectTy, _, ok := resolveActionOrExpression(
			t,
			collectChain,
			collectClause.Expression,
			expectedType,
		)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		queryTy = collectTy
	}

	if onConflictClause != nil {
		if expr.QueryConstructType != ast.TypeKind_MAP {
			t.semanticError("on conflict clause is supported only for map query construct type",
				onConflictClause.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		conflictTy, _, ok := resolveActionOrExpression(t, queryChain, onConflictClause.Expression, semtypes.Union(semtypes.ERROR, semtypes.NIL))
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if !semtypes.IsSubtype(t.typeContext(), conflictTy, semtypes.Union(semtypes.ERROR, semtypes.NIL)) {
			t.semanticError("on conflict clause expression must be error?", onConflictClause.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		errorTy := semtypes.Intersect(conflictTy, semtypes.ERROR)
		if !semtypes.IsEmpty(t.typeContext(), errorTy) {
			queryTy = semtypes.Union(queryTy, errorTy)
		}
	}
	setExpectedType(expr, queryTy)
	return queryTy, defaultExpressionEffect(chain), true
}

func resolveQueryCollectionElementType(
	t typeResolver,
	collectionTy semtypes.SemType,
	pos diagnostics.Location,
) (semtypes.SemType, bool) {
	switch {
	case semtypes.IsSubtype(t.typeContext(), collectionTy, semtypes.LIST):
		memberTypes := semtypes.ListAllMemberTypesInner(t.typeContext(), collectionTy)
		result := semtypes.NEVER
		for _, each := range memberTypes.SemTypes {
			result = semtypes.Union(result, each)
		}
		return result, true
	case semtypes.IsSubtype(t.typeContext(), collectionTy, semtypes.MAPPING):
		return semtypes.MappingMemberTypeInnerValProj(t.typeContext(), collectionTy, semtypes.STRING), true
	default:
		t.unimplemented("query from clause currently supports only list or map collections", pos)
		return semtypes.SemType{}, false
	}
}

func resolveForeachVariableType(t typeResolver, collection ast.BLangActionOrExpression, collectionTy semtypes.SemType) (semtypes.SemType, bool) {
	if binaryExpr, ok := collection.(*ast.BLangBinaryExpr); ok && isRangeExpr(binaryExpr) {
		return semtypes.INT, true
	}
	ctx := t.typeContext()
	switch {
	case semtypes.IsSubtype(ctx, collectionTy, semtypes.LIST):
		return semtypes.ListMemberTypeInnerVal(ctx, collectionTy, semtypes.INT), true
	case semtypes.IsSubtype(ctx, collectionTy, semtypes.MAPPING):
		return semtypes.MappingMemberTypeInnerVal(ctx, collectionTy, semtypes.STRING), true
	case semtypes.IsSubtype(ctx, collectionTy, semtypes.XML):
		return semtypes.XMLItemType(collectionTy), true
	default:
		iterableTy, ok := getIterableType(t.compilerContext(), t.symbolType)
		if !ok {
			t.semanticError("foreach collection must be subtype of object:Iterable", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		if !semtypes.IsSubtype(ctx, collectionTy, iterableTy) {
			t.semanticError("foreach collection must be subtype of object:Iterable", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		ld := semtypes.NewListDefinition()
		emptyListTy := ld.DefineListTypeWrapped(t.typeEnv(), nil, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
		iteratorFnTy := semtypes.ObjectMemberType(ctx, semtypes.StringConst("iterator"), collectionTy)
		if semtypes.IsZero(iteratorFnTy) || !semtypes.IsSubtype(ctx, iteratorFnTy, semtypes.FUNCTION) {
			t.semanticError("foreach collection is not iterable", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		iteratorTy := semtypes.FunctionReturnType(ctx, iteratorFnTy, emptyListTy)
		nextFnTy := semtypes.ObjectMemberType(ctx, semtypes.StringConst("next"), iteratorTy)
		if semtypes.IsZero(nextFnTy) || !semtypes.IsSubtype(ctx, nextFnTy, semtypes.FUNCTION) {
			t.semanticError("foreach iterator does not have a next method", collection.GetPosition())
			return semtypes.SemType{}, false
		}
		nextReturnTy := semtypes.FunctionReturnType(ctx, nextFnTy, emptyListTy)
		valueRecordTy := semtypes.Diff(semtypes.Diff(nextReturnTy, semtypes.NIL), semtypes.ERROR)
		return semtypes.MappingMemberTypeInnerVal(ctx, valueRecordTy, semtypes.StringConst("value")), true
	}
}

func mapQuerySelectExpectedType(env semtypes.Env) semtypes.SemType {
	return mapQuerySelectExpectedTypeWithValue(env, semtypes.Union(semtypes.ANY, semtypes.ERROR))
}

func querySelectExpectedType(
	ctx semtypes.Context,
	env semtypes.Env,
	queryConstructType ast.TypeKind,
	expectedType semtypes.SemType,
) semtypes.SemType {
	switch queryConstructType {
	case ast.TypeKind_NONE:
		return listQuerySelectExpectedType(ctx, expectedType)
	case ast.TypeKind_MAP:
		return mapQuerySelectExpectedTypeFromQueryExpectedType(ctx, env, expectedType)
	default:
		return semtypes.SemType{}
	}
}

func listQuerySelectExpectedType(ctx semtypes.Context, expectedType semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(expectedType) {
		return semtypes.SemType{}
	}
	listTy := semtypes.Intersect(expectedType, semtypes.LIST)
	if semtypes.IsEmpty(ctx, listTy) {
		return semtypes.SemType{}
	}
	memberTypes := semtypes.ListAllMemberTypesInner(ctx, listTy)
	result := semtypes.NEVER
	for _, memberTy := range memberTypes.SemTypes {
		result = semtypes.Union(result, memberTy)
	}
	if semtypes.IsEmpty(ctx, result) {
		return semtypes.SemType{}
	}
	return result
}

func mapQuerySelectExpectedTypeFromQueryExpectedType(
	ctx semtypes.Context,
	env semtypes.Env,
	expectedType semtypes.SemType,
) semtypes.SemType {
	if semtypes.IsZero(expectedType) {
		return semtypes.SemType{}
	}
	mappingTy := semtypes.Intersect(expectedType, semtypes.MAPPING)
	if semtypes.IsEmpty(ctx, mappingTy) {
		return semtypes.SemType{}
	}
	valueTy := semtypes.MappingMemberTypeInnerValProj(ctx, mappingTy, semtypes.STRING)
	if semtypes.IsSubtype(ctx, semtypes.CreateAnydata(ctx), valueTy) {
		return semtypes.SemType{}
	}
	return mapQuerySelectExpectedTypeWithValue(env, valueTy)
}

func mapQuerySelectExpectedTypeWithValue(env semtypes.Env, valueTy semtypes.SemType) semtypes.SemType {
	ld := semtypes.NewListDefinition()
	return ld.DefineListTypeWrapped(env, []semtypes.SemType{semtypes.STRING, valueTy}, 2, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_LIMITED)
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
	variableDef ast.VariableDefinitionNode,
) []queryVariableInfo {
	varDef, ok := variableDef.(*ast.BLangSimpleVariableDef)
	if !ok || varDef == nil || varDef.Var == nil || !ast.SymbolIsSet(varDef.Var) {
		return variables
	}
	symbol := varDef.Var.Symbol()
	if seen[symbol] {
		return variables
	}
	seen[symbol] = true
	name := ""
	if varDef.Var.Name != nil {
		name = varDef.Var.Name.Value
	}
	return append(variables, queryVariableInfo{
		name:   name,
		symbol: symbol,
	})
}

func queryAggregatedListType(env semtypes.Env, elemTy semtypes.SemType, nonEmpty bool) semtypes.SemType {
	if semtypes.IsZero(elemTy) {
		elemTy = semtypes.ANY
	}
	ld := semtypes.NewListDefinition()
	if nonEmpty {
		return ld.DefineListTypeWrapped(env, []semtypes.SemType{elemTy}, 1, elemTy, semtypes.CellMutability_CELL_MUT_LIMITED)
	}
	return ld.DefineListTypeWrappedWithEnvSemType(env, elemTy)
}

func aggregateQueryVariable(t typeResolver, chain *binding, variable queryVariableInfo, nonEmpty bool) *binding {
	effectiveSymbol := lookupSymbol(chain, variable.symbol)
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

func resolveQueryGroupingKeyVarDef(t typeResolver, chain *binding, varDef *ast.BLangSimpleVariableDef) (semtypes.SemType, bool) {
	if varDef.Var == nil {
		t.unimplemented("only simple variable declarations are supported in group by clause", varDef.GetPosition())
		return semtypes.SemType{}, false
	}
	varDef.SetDeterminedType(semtypes.NEVER)
	if varDef.Var.Expr == nil {
		t.semanticError("group by variable declaration requires an initializer", varDef.GetPosition())
		return semtypes.SemType{}, false
	}
	var variableTy semtypes.SemType
	if !varDef.Var.GetIsDeclaredWithVar() && varDef.Var.TypeNode() != nil {
		var ok bool
		variableTy, ok = resolveBType(t, varDef.Var.TypeNode(), 0)
		if !ok {
			return semtypes.SemType{}, false
		}
	}
	initTy, _, ok := resolveActionOrExpression(t, chain, varDef.Var.Expr.(ast.BLangExpression), variableTy)
	if !ok {
		return semtypes.SemType{}, false
	}
	if semtypes.IsZero(variableTy) {
		variableTy = initTy
	} else if !semtypes.IsSubtype(t.typeContext(), initTy, variableTy) {
		t.semanticError("group by variable type is incompatible with initializer expression", varDef.GetPosition())
		return semtypes.SemType{}, false
	}
	if varDef.Var.Name != nil {
		varDef.Var.Name.SetDeterminedType(semtypes.NEVER)
	}
	varDef.Var.SetDeterminedType(semtypes.NEVER)
	updateSymbolType(t, varDef.Var, variableTy)
	return variableTy, true
}

func resolveQueryGroupByClause(
	t typeResolver,
	chain *binding,
	queryExpr *ast.BLangQueryExpr,
	clause *ast.BLangGroupByClause,
	clauseIndex int,
) (*binding, bool) {
	clause.SetDeterminedType(semtypes.NEVER)
	queryVariables := queryVariablesBeforeClause(queryExpr, clauseIndex)
	nonGroupingKeys := &balCommon.OrderedSet[string]{}
	for _, variable := range queryVariables {
		if variable.name != "" && variable.name != "_" {
			nonGroupingKeys.Add(variable.name)
		}
	}

	for i := range clause.GroupingKeyList {
		groupingKey := &clause.GroupingKeyList[i]
		groupingKey.SetDeterminedType(semtypes.NEVER)
		switch {
		case groupingKey.VariableRef != nil:
			keyTy, _, ok := resolveActionOrExpression(t, chain, groupingKey.VariableRef, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			if !validateQueryGroupingKeyType(t, keyTy, groupingKey.GetPosition()) {
				return nil, false
			}
			if groupingKey.VariableRef.VariableName != nil {
				nonGroupingKeys.Remove(groupingKey.VariableRef.VariableName.Value)
			}
		case groupingKey.VariableDef != nil:
			keyTy, ok := resolveQueryGroupingKeyVarDef(t, chain, groupingKey.VariableDef)
			if !ok {
				return nil, false
			}
			if !validateQueryGroupingKeyType(t, keyTy, groupingKey.GetPosition()) {
				return nil, false
			}
			if groupingKey.VariableDef.Var.Name != nil {
				nonGroupingKeys.Remove(groupingKey.VariableDef.Var.Name.Value)
			}
		default:
			t.semanticError("group by clause requires a grouping key", groupingKey.GetPosition())
			return nil, false
		}
	}
	clause.NonGroupingKeys = nonGroupingKeys

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
			clause.SetDeterminedType(semtypes.NEVER)
			collectionTy, _, ok := resolveActionOrExpression(t, currentChain, clause.Collection, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			elementTy, ok := resolveQueryCollectionElementType(t, collectionTy, clause.GetPosition())
			if !ok {
				return nil, false
			}
			varDef := clause.VariableDefinitionNode
			if varDef == nil || varDef.Var == nil {
				t.unimplemented("only simple variable bindings are supported in join clause", clause.GetPosition())
				return nil, false
			}
			varDef.SetDeterminedType(semtypes.NEVER)
			if clause.IsOuterJoinFlag && !clause.IsDeclaredWithVarFlag {
				t.semanticError("outer join clause variable must be declared with var", clause.GetPosition())
				return nil, false
			}
			variableTy := elementTy
			if clause.IsOuterJoinFlag {
				variableTy = semtypes.Union(variableTy, semtypes.NIL)
			}
			if !clause.IsDeclaredWithVarFlag && varDef.Var.TypeNode() != nil {
				variableTy, ok = resolveBType(t, varDef.Var.TypeNode(), 0)
				if !ok {
					return nil, false
				}
				if !semtypes.IsSubtype(t.typeContext(), elementTy, variableTy) {
					t.semanticError("join clause variable type is incompatible with collection member type",
						varDef.GetPosition())
					return nil, false
				}
			}
			if varDef.Var.Name != nil {
				varDef.Var.Name.SetDeterminedType(semtypes.NEVER)
			}
			varDef.Var.SetDeterminedType(semtypes.NEVER)
			updateSymbolType(t, varDef.Var, variableTy)

			if clause.OnClause.OnExpr == nil || clause.OnClause.EqualsExpr == nil {
				t.semanticError("join clause requires an on clause", clause.GetPosition())
				return nil, false
			}
			clause.OnClause.SetDeterminedType(semtypes.NEVER)
			lhsTy, _, ok := resolveActionOrExpression(t, currentChain, clause.OnClause.OnExpr, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			rhsTy, _, ok := resolveActionOrExpression(t, currentChain, clause.OnClause.EqualsExpr, semtypes.SemType{})
			if !ok {
				return nil, false
			}
			if !semtypes.IsSubtype(t.typeContext(), lhsTy, rhsTy) {
				t.semanticError(formatIncompatibleTypeMessage(t.typeContext(), rhsTy, lhsTy), clause.OnClause.EqualsExpr.GetPosition())
				return nil, false
			}
		case *ast.BLangLetClause:
			clause.SetDeterminedType(semtypes.NEVER)
			for i := range clause.LetVarDeclarations {
				varDef := &clause.LetVarDeclarations[i]
				if varDef.Var == nil {
					t.unimplemented("only simple variable declarations are supported in let clause",
						clause.GetPosition())
					return nil, false
				}
				varDef.SetDeterminedType(semtypes.NEVER)
				if varDef.Var.Expr == nil {
					t.semanticError("let clause variable declaration requires an initializer",
						varDef.GetPosition())
					return nil, false
				}
				initTy, _, ok := resolveActionOrExpression(t, currentChain, varDef.Var.Expr.(ast.BLangExpression), semtypes.SemType{})
				if !ok {
					return nil, false
				}
				variableTy := initTy
				if !varDef.Var.GetIsDeclaredWithVar() && varDef.Var.TypeNode() != nil {
					variableTy, ok = resolveBType(t, varDef.Var.TypeNode(), 0)
					if !ok {
						return nil, false
					}
					if !semtypes.IsSubtype(t.typeContext(), initTy, variableTy) {
						t.semanticError("let clause variable type is incompatible with initializer expression",
							varDef.GetPosition())
						return nil, false
					}
				}
				if varDef.Var.Name != nil {
					varDef.Var.Name.SetDeterminedType(semtypes.NEVER)
				}
				varDef.Var.SetDeterminedType(semtypes.NEVER)
				updateSymbolType(t, varDef.Var, variableTy)
			}
		case *ast.BLangWhereClause:
			clause.SetDeterminedType(semtypes.NEVER)
			whereTy, effect, ok := resolveActionOrExpression(t, currentChain, clause.Expression, semtypes.BOOLEAN)
			if !ok {
				return nil, false
			}
			if !semtypes.IsSubtype(t.typeContext(), whereTy, semtypes.BOOLEAN) {
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
			clause.SetDeterminedType(semtypes.NEVER)
			limitTy, _, ok := resolveActionOrExpression(t, currentChain, clause.Expression, semtypes.INT)
			if !ok {
				return nil, false
			}
			if !semtypes.IsSubtype(t.typeContext(), limitTy, semtypes.INT) {
				t.semanticError("limit clause expression must be int", clause.GetPosition())
				return nil, false
			}
		case *ast.BLangOrderByClause:
			clause.SetDeterminedType(semtypes.NEVER)
			orderedTy := semtypes.CreateOrdered(t.typeContext())
			for j := range clause.OrderByKeyList {
				orderKey := &clause.OrderByKeyList[j]
				orderKey.SetDeterminedType(semtypes.NEVER)
				keyTy, _, ok := resolveActionOrExpression(t, currentChain, orderKey.Expression, semtypes.SemType{})
				if !ok {
					return nil, false
				}
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

func resolveSimpleVarRef(t typeResolver, chain *binding, expr *ast.BLangSimpleVarRef) (semtypes.SemType, expressionEffect, bool) {
	baseSymbol := expr.Symbol()
	sym, isNarrowed, isCaptured := lookupBinding(chain, baseSymbol)
	if isNarrowed {
		expr.SetSymbol(sym)
	}
	if isCaptured {
		t.trackCapturedVar(baseSymbol)
	}
	if !t.ensureResolved(sym, 0) {
		return semtypes.SemType{}, defaultExpressionEffect(chain), false
	}
	ty := t.symbolType(sym)
	if t.getSymbol(sym).Kind() == model.SymbolKindType {
		ty = semtypes.TypedescContaining(t.typeEnv(), ty)
	}
	setExpectedType(expr, ty)
	setVarRefIdentifierTypes(expr)
	return ty, defaultExpressionEffect(chain), true
}

func resolveLocalVarRef(t typeResolver, chain *binding, expr *ast.BLangLocalVarRef) (semtypes.SemType, expressionEffect, bool) {
	return resolveSimpleVarRef(t, chain, &expr.BLangSimpleVarRef)
}

func resolveConstRef(t typeResolver, chain *binding, expr *ast.BLangConstRef) (semtypes.SemType, expressionEffect, bool) {
	sym, isNarrowed, _ := lookupBinding(chain, expr.Symbol())
	if isNarrowed {
		expr.SetSymbol(sym)
	}
	if !t.ensureResolved(sym, 0) {
		return semtypes.SemType{}, defaultExpressionEffect(chain), false
	}
	ty := t.symbolType(sym)
	setExpectedType(expr, ty)
	setVarRefIdentifierTypes(&expr.BLangSimpleVarRef)
	return ty, defaultExpressionEffect(chain), true
}

func resolveListConstructorExpr(t typeResolver, chain *binding, expr *ast.BLangListConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	if !semtypes.IsZero(expectedType) {
		return resolveListConstructorWithExpectedType(t, chain, expr, expectedType)
	}
	return resolveListConstructorInner(t, chain, expr)
}

func resolveListConstructorInner(t typeResolver, chain *binding, expr *ast.BLangListConstructorExpr) (semtypes.SemType, expressionEffect, bool) {
	memberTypes := make([]semtypes.SemType, 0, len(expr.Exprs))
	restTy := semtypes.NEVER
	spreadMembers := make([]bool, len(expr.Exprs))
	hasSpread := false
	for i, memberExpr := range expr.Exprs {
		isSpread := expr.IsSpreadMember(i) || isQueryAggregatedVariableReference(chain, memberExpr)
		memberTy, _, ok := resolveActionOrExpression(t, chain, memberExpr, semtypes.SemType{})
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if isSpread {
			spreadMembers[i] = true
			spreadMemberTy := semtypes.ListProj(t.typeContext(), memberTy, semtypes.INT)
			restTy = semtypes.Union(restTy, widenedListMemberType(spreadMemberTy))
			hasSpread = true
			continue
		}
		broadTy := widenedListMemberType(memberTy)
		if hasSpread {
			restTy = semtypes.Union(restTy, broadTy)
			continue
		}
		memberTypes = append(memberTypes, broadTy)
	}
	setListConstructorSpreadMembers(expr, spreadMembers)

	ld := semtypes.NewListDefinition()
	listTy := ld.DefineListTypeWrapped(t.typeEnv(), memberTypes, len(memberTypes), restTy, semtypes.CellMutability_CELL_MUT_LIMITED)

	setExpectedType(expr, listTy)
	lat := semtypes.ToListAtomicType(t.typeContext(), listTy)
	expr.AtomicType = *lat

	return listTy, defaultExpressionEffect(chain), true
}

func resolveListConstructorWithExpectedType(t typeResolver, chain *binding, expr *ast.BLangListConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	spreadMembers := make([]bool, len(expr.Exprs))
	for i, memberExpr := range expr.Exprs {
		spreadMembers[i] = expr.IsSpreadMember(i) || isQueryAggregatedVariableReference(chain, memberExpr)
		if _, _, ok := resolveActionOrExpression(t, chain, memberExpr, semtypes.SemType{}); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}
	setListConstructorSpreadMembers(expr, spreadMembers)

	resultType, lat, ok := selectListInherentType(t, expr, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	memberIndex := 0
	restMember := false
	for i, memberExpr := range expr.Exprs {
		isSpread := expr.IsSpreadMember(i)
		requiredType := lat.MemberAtInnerVal(memberIndex)
		if restMember || isSpread {
			requiredType = lat.Rest()
		}
		if semtypes.IsNever(requiredType) {
			if isSpread {
				t.semanticError("aggregated variable reference cannot be used as a spread member for a fixed-length list constructor", memberExpr.GetPosition())
				return semtypes.SemType{}, expressionEffect{}, false
			}
			t.semanticError("too many members in list constructor", expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		memberExpr.SetDeterminedType(semtypes.SemType{})
		if isSpread {
			spreadExpectedType := queryAggregatedListType(t.typeEnv(), requiredType, false)
			if _, _, ok := resolveActionOrExpression(t, chain, memberExpr, spreadExpectedType); !ok {
				return semtypes.SemType{}, expressionEffect{}, false
			}
			restMember = true
			memberIndex = lat.Members.FixedLength
			continue
		}
		if _, _, ok := resolveActionOrExpression(t, chain, memberExpr, requiredType); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if !restMember {
			memberIndex++
		}
	}

	expr.AtomicType = lat
	setExpectedType(expr, resultType)
	return resultType, defaultExpressionEffect(chain), true
}

func setListConstructorSpreadMembers(expr *ast.BLangListConstructorExpr, spreadMembers []bool) {
	for _, isSpread := range spreadMembers {
		if isSpread {
			expr.SpreadMembers = spreadMembers
			return
		}
	}
	expr.SpreadMembers = nil
}

func isQueryAggregatedVariableReference(chain *binding, expr ast.BLangExpression) bool {
	switch ref := expr.(type) {
	case *ast.BLangSimpleVarRef:
		return lookupQueryAggregatedBinding(chain, ref.Symbol())
	case *ast.BLangLocalVarRef:
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

func selectListInherentType(t typeResolver, expr *ast.BLangListConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, semtypes.ListAtomicType, bool) {
	expectedListType := semtypes.Intersect(expectedType, semtypes.LIST)
	tc := t.typeContext()
	if semtypes.IsEmpty(tc, expectedListType) {
		t.semanticError("list type not found in expected type", expr.GetPosition())
		return semtypes.SemType{}, semtypes.ListAtomicType{}, false
	}
	lat := semtypes.ToListAtomicType(tc, expectedListType)
	if lat != nil {
		return expectedListType, *lat, true
	}

	alts := semtypes.ListAlternatives(tc, expectedListType)

	members := make([]semtypes.ListMemberInfo, len(expr.Exprs))
	for i, expr := range expr.Exprs {
		members[i] = semtypes.ListMemberInfo{Index: i, ValType: expr.GetDeterminedType()}
	}

	var validAlts []semtypes.ListAlternative
	for _, alt := range alts {
		if semtypes.ListAlternativeAllowsMembers(tc, alt, members) {
			validAlts = append(validAlts, alt)
		}
	}

	if len(validAlts) == 0 {
		t.semanticError("no applicable inherent type for list constructor", expr.GetPosition())
		return semtypes.SemType{}, semtypes.ListAtomicType{}, false
	}
	if len(validAlts) > 1 {
		t.semanticError("ambiguous inherent type for list constructor", expr.GetPosition())
		return semtypes.SemType{}, semtypes.ListAtomicType{}, false
	}

	selectedSemType := validAlts[0].SemType
	lat = semtypes.ToListAtomicType(tc, selectedSemType)
	if lat == nil {
		t.semanticError("applicable type for list constructor is not atomic", expr.GetPosition())
		return semtypes.SemType{}, semtypes.ListAtomicType{}, false
	}

	return selectedSemType, *lat, true
}

func resolveErrorConstructorExpr(t typeResolver, chain *binding, expr *ast.BLangErrorConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	var errorTy semtypes.SemType

	if expr.ErrorTypeRef != nil {
		refTy, ok := resolveBType(t, expr.ErrorTypeRef, 0)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if !semtypes.IsSubtype(t.typeContext(), refTy, semtypes.ERROR) {
			t.semanticError("error type parameter must be a subtype of error", expr.ErrorTypeRef.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		} else {
			errorTy = refTy
		}
	} else {
		errorTy = semtypes.ERROR
	}

	if !semtypes.IsZero(expectedType) && semtypes.IsSameType(t.typeContext(), errorTy, semtypes.ERROR) {
		errorPart := semtypes.Intersect(expectedType, semtypes.ERROR)
		if !semtypes.IsEmpty(t.typeContext(), errorPart) {
			errorTy = errorPart
		}
	}

	setExpectedType(expr, errorTy)

	for _, arg := range expr.PositionalArgs {
		if _, _, ok := resolveActionOrExpression(t, chain, arg, semtypes.SemType{}); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}
	for i := range expr.NamedArgs {
		if _, _, ok := resolveActionOrExpression(t, chain, &expr.NamedArgs[i], semtypes.SemType{}); !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
	}
	return errorTy, defaultExpressionEffect(chain), true
}

func resolveUnaryExpr(t typeResolver, chain *binding, expr *ast.BLangUnaryExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	exprTy, innerEffect, ok := resolveActionOrExpression(t, chain, expr.Expr, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	// Check for nil lifting on numeric/bitwise unary operators
	nilLifted := false
	underlyingTy := exprTy
	if expr.GetOperatorKind() != model.OperatorKind_NOT {
		if semtypes.ContainsBasicType(exprTy, semtypes.NIL) {
			nilLifted = true
			underlyingTy = semtypes.Diff(exprTy, semtypes.NIL)
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
		if !semtypes.IsSubtype(t.typeContext(), underlyingTy, semtypes.INT) {
			t.semanticError(fmt.Sprintf("expect int type for %s", string(expr.GetOperatorKind())), expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if semtypes.IsSameType(t.typeContext(), underlyingTy, semtypes.INT) {
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
		if semtypes.IsSubtype(t.typeContext(), exprTy, semtypes.BOOLEAN) {
			if semtypes.IsSameType(t.typeContext(), exprTy, semtypes.BOOLEAN) {
				resultTy = semtypes.BOOLEAN
			} else {
				resultTy = semtypes.Diff(semtypes.BOOLEAN, exprTy)
			}
		} else {
			t.semanticError(fmt.Sprintf("expect boolean type for %s", string(expr.GetOperatorKind())), expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		setExpectedType(expr, resultTy)
		return resultTy, expressionEffect{ifTrue: innerEffect.ifFalse, ifFalse: innerEffect.ifTrue}, true
	default:
		t.internalError(fmt.Sprintf("unsupported unary operator: %s", string(expr.GetOperatorKind())), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	if nilLifted {
		resultTy = semtypes.Union(semtypes.NIL, resultTy)
	}
	setExpectedType(expr, resultTy)
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
	case bothSameType(semtypes.STRING):
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
	case bothSameType(semtypes.INT):
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
	case bothSameType(semtypes.FLOAT):
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
	case bothSameType(semtypes.DECIMAL):
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
		supportedTypes = semtypes.NUMBER
	}
	operandExpectedType := semtypes.Union(supportedTypes, semtypes.XML)
	if !semtypes.IsZero(expectedType) {
		operandExpectedType = semtypes.Intersect(operandExpectedType, expectedType)
	}
	lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy, effect, ok := resolveAdditiveExprInner(t, lhsEffect.ifTrue, lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expectedType, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(expr, resultTy)
	return resultTy, effect, true
}

func resolveAdditiveExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, expectedType semtypes.SemType, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	supportedTypes := additiveSupportedTypes
	if op == model.OperatorKind_SUB {
		supportedTypes = semtypes.NUMBER
	}
	operandExpectedType := semtypes.Union(supportedTypes, semtypes.XML)
	if !semtypes.IsZero(expectedType) {
		operandExpectedType = semtypes.Intersect(operandExpectedType, expectedType)
	}
	rhsTy, _, ok := resolveActionOrExpression(t, chain, rhs, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	singletonTy, ok := additiveSingletonType(t, lhsTy, rhsTy, op, pos)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if !semtypes.IsZero(singletonTy) {
		return singletonTy, defaultExpressionEffect(chain), true
	}

	lhsTy, rhsTy, nilLifted := nilLiftedUnderlyingType(lhsTy, rhsTy)

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
		resultTy = semtypes.Union(semtypes.NIL, resultTy)
	}
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveRangeExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	_, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.INT)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	_, _, ok = resolveActionOrExpression(t, lhsEffect.ifTrue, expr.RhsExpr, semtypes.INT)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := createIteratorType(t.typeEnv(), semtypes.INT, semtypes.NIL)
	setExpectedType(expr, resultTy)
	effect := defaultExpressionEffect(lhsEffect.ifTrue)
	return resultTy, effect, true
}

func resolveShiftExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.INT)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy, effect, ok := resolveShiftExprInner(t, lhsEffect.ifTrue, lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(expr, resultTy)
	return resultTy, effect, true
}

func resolveShiftExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsTy, _, ok := resolveActionOrExpression(t, chain, rhs, semtypes.INT)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	lhsTy, rhsTy, nilLifted := nilLiftedUnderlyingType(lhsTy, rhsTy)
	ctx := t.typeContext()
	// TODO: handle singleton typing here

	if semtypes.IsEmpty(ctx, lhsTy) || semtypes.IsEmpty(ctx, rhsTy) || !semtypes.IsSubtype(ctx, lhsTy, semtypes.INT) || !semtypes.IsSubtype(ctx, rhsTy, semtypes.INT) {
		t.semanticError(fmt.Sprintf("expect integer types for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.INT
	switch op {
	case model.OperatorKind_BITWISE_RIGHT_SHIFT, model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		for _, ty := range bitWiseOpLookOrder {
			if semtypes.IsSubtype(ctx, lhsTy, ty) {
				resultTy = ty
				break
			}
		}
	}
	if nilLifted {
		resultTy = semtypes.Union(resultTy, semtypes.NIL)
	}
	return resultTy, defaultExpressionEffect(chain), true
}

func nilLiftedUnderlyingType(lhsTy, rhsTy semtypes.SemType) (semtypes.SemType, semtypes.SemType, bool) {
	nilLifted := false
	if semtypes.ContainsBasicType(lhsTy, semtypes.NIL) {
		nilLifted = true
		lhsTy = semtypes.Diff(lhsTy, semtypes.NIL)
	}
	if semtypes.ContainsBasicType(rhsTy, semtypes.NIL) {
		nilLifted = true
		rhsTy = semtypes.Diff(rhsTy, semtypes.NIL)
	}
	return lhsTy, rhsTy, nilLifted
}

func resolveRelationalExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	rhsTy, _, ok := resolveActionOrExpression(t, lhsEffect.ifTrue, expr.RhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	if !semtypes.Comparable(t.typeContext(), lhsTy, rhsTy) {
		t.semanticError("values are not comparable", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.BOOLEAN
	setExpectedType(expr, resultTy)
	effect := defaultExpressionEffect(lhsEffect.ifTrue)
	return resultTy, effect, true
}

func resolveMultiplicativeExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	operandExpectedType := semtypes.NUMBER
	if !semtypes.IsZero(expectedType) {
		operandExpectedType = semtypes.Intersect(expectedType, operandExpectedType)
	}
	lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy, effect, ok := resolveMultiplicativeExprInner(t, lhsEffect.ifTrue, lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expectedType, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(expr, resultTy)
	return resultTy, effect, true
}

func resolveMultiplicativeExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, expectedType semtypes.SemType, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	operandExpectedType := semtypes.NUMBER
	if !semtypes.IsZero(expectedType) {
		operandExpectedType = semtypes.Intersect(expectedType, operandExpectedType)
	}
	rhsTy, _, ok := resolveActionOrExpression(t, chain, rhs, operandExpectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
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
	if !isNumericType(t.typeContext(), lhsBasicTy) || !isNumericType(t.typeContext(), rhsBasicTy) {
		t.semanticError(fmt.Sprintf("expect numeric types for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}
	var resultTy semtypes.SemType
	if !semtypes.IsSameType(t.typeContext(), lhsBasicTy, rhsBasicTy) {
		ctx := t.typeContext()
		if semtypes.IsSubtype(ctx, rhsBasicTy, semtypes.INT) {
			resultTy = lhsBasicTy
		} else if op == model.OperatorKind_MUL && semtypes.IsSubtype(ctx, lhsBasicTy, semtypes.INT) {
			resultTy = rhsBasicTy
		} else {
			t.semanticError("both operands must belong to same basic type", pos)
			return semtypes.SemType{}, expressionEffect{}, false
		}
	} else {
		resultTy = lhsBasicTy
	}
	if nilLifted {
		resultTy = semtypes.Union(semtypes.NIL, resultTy)
	}
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveBitWiseExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.INT)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy, effect, ok := resolveBitWiseExprInner(t, lhsEffect.ifTrue, lhsTy, expr.RhsExpr, expr.GetOperatorKind(), expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(expr, resultTy)
	return resultTy, effect, true
}

func resolveBitWiseExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, rhs ast.BLangActionOrExpression, op model.OperatorKind, pos diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsTy, _, ok := resolveActionOrExpression(t, chain, rhs, semtypes.INT)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	// TODO: handle singleton

	lhsTy, rhsTy, nilLifted := nilLiftedUnderlyingType(lhsTy, rhsTy)

	numLhsBits := semtypes.NBasicTypes(lhsTy)
	numRhsBits := semtypes.NBasicTypes(rhsTy)

	if numLhsBits != 1 || numRhsBits != 1 {
		t.semanticError(fmt.Sprintf("union types not supported for %s", string(op)), pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}

	ctx := t.typeContext()
	if !semtypes.IsSubtype(ctx, lhsTy, semtypes.INT) || !semtypes.IsSubtype(ctx, rhsTy, semtypes.INT) {
		t.semanticError("expect integer types for bitwise operators", pos)
		return semtypes.SemType{}, expressionEffect{}, false
	}

	resultTy := semtypes.INT
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
		resultTy = semtypes.Union(resultTy, semtypes.NIL)
	}
	return resultTy, defaultExpressionEffect(chain), true
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
	_, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	_, _, ok = resolveActionOrExpression(t, lhsEffect.ifTrue, expr.RhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	var effect expressionEffect
	// TODO: pass in lhs and rhs types instead
	if expr.OpKind == model.OperatorKind_EQUAL || expr.OpKind == model.OperatorKind_NOT_EQUAL {
		effect = equalityNarrowingEffect(t, chain, expr)
	} else {
		effect = defaultExpressionEffect(chain)
	}
	resultTy := semtypes.BOOLEAN
	expr.SetDeterminedType(resultTy)
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
	lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy, effect, ok := resolveAndExprInner(t, chain, lhsTy, lhsEffect, expr.RhsExpr, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(expr, resultTy)
	return resultTy, effect, true
}

func resolveAndExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, lhsEffect expressionEffect, rhs ast.BLangActionOrExpression, _ diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsTy, rhsEffect, ok := resolveActionOrExpression(t, lhsEffect.ifTrue, rhs, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	resultTy := semtypes.BOOLEAN
	if isSingletonBool(lhsTy, false) || isSingletonBool(rhsTy, false) {
		resultTy = semtypes.BooleanConst(false)
	} else if isSingletonBool(lhsTy, true) && isSingletonBool(rhsTy, true) {
		resultTy = semtypes.BooleanConst(true)
	} else if isSingletonBool(lhsTy, true) {
		resultTy = rhsTy
	}

	if effect, isSingleton := singletonResultEffect(chain, resultTy); isSingleton {
		return resultTy, effect, true
	}

	rhsDiffTrue := diff(rhsEffect.ifTrue, lhsEffect.ifTrue)
	rhsDiffFalse := diff(rhsEffect.ifFalse, lhsEffect.ifTrue)
	ifTrue := mergeChains(t, lhsEffect.ifTrue, rhsDiffTrue, semtypes.Intersect)
	ifFalse := mergeChains(t, lhsEffect.ifFalse, mergeChains(t, lhsEffect.ifTrue, rhsDiffFalse, semtypes.Intersect), semtypes.Union)
	return resultTy, expressionEffect{ifTrue: ifTrue, ifFalse: ifFalse}, true
}

func resolveOrExpr(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) (semtypes.SemType, expressionEffect, bool) {
	lhsTy, lhsEffect, ok := resolveActionOrExpression(t, chain, expr.LhsExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy, effect, ok := resolveOrExprInner(t, chain, lhsTy, lhsEffect, expr.RhsExpr, expr.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	setExpectedType(expr, resultTy)
	return resultTy, effect, true
}

func resolveOrExprInner(t typeResolver, chain *binding, lhsTy semtypes.SemType, lhsEffect expressionEffect, rhs ast.BLangActionOrExpression, _ diagnostics.Location) (semtypes.SemType, expressionEffect, bool) {
	rhsTy, rhsEffect, ok := resolveActionOrExpression(t, lhsEffect.ifFalse, rhs, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	resultTy := semtypes.BOOLEAN
	if isSingletonBool(lhsTy, true) || isSingletonBool(rhsTy, true) {
		resultTy = semtypes.BooleanConst(true)
	} else if isSingletonBool(lhsTy, false) && isSingletonBool(rhsTy, false) {
		resultTy = semtypes.BooleanConst(false)
	} else if isSingletonBool(lhsTy, false) {
		resultTy = rhsTy
	}

	if effect, isSingleton := singletonResultEffect(chain, resultTy); isSingleton {
		return resultTy, effect, true
	}

	rhsDiffTrue := diff(rhsEffect.ifTrue, lhsEffect.ifFalse)
	rhsDiffFalse := diff(rhsEffect.ifFalse, lhsEffect.ifFalse)
	ifTrue := mergeChains(t, lhsEffect.ifTrue, mergeChains(t, lhsEffect.ifFalse, rhsDiffTrue, semtypes.Intersect), semtypes.Union)
	ifFalse := mergeChains(t, lhsEffect.ifFalse, rhsDiffFalse, semtypes.Intersect)
	return resultTy, expressionEffect{ifTrue: ifTrue, ifFalse: ifFalse}, true
}

func equalityNarrowingEffect(t typeResolver, chain *binding, expr *ast.BLangBinaryExpr) expressionEffect {
	lhsRef, lhsIsVarRef := varRefExp(chain, expr.LhsExpr)
	rhsTy := expr.RhsExpr.GetDeterminedType()
	rhsIsSingleton := semtypes.SingleShape(rhsTy).IsPresent()
	if lhsIsVarRef && rhsIsSingleton {
		effect := buildEqualityNarrowing(t, chain, lhsRef, rhsTy)
		if expr.OpKind == model.OperatorKind_NOT_EQUAL {
			return expressionEffect{ifTrue: effect.ifFalse, ifFalse: effect.ifTrue}
		}
		return effect
	}
	rhsRef, rhsIsVarRef := varRefExp(chain, expr.RhsExpr)
	lhsTy := expr.LhsExpr.GetDeterminedType()
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
	symbolTy := t.symbolType(ref)
	trueTy := semtypes.Intersect(symbolTy, singletonTy)
	trueSym := narrowSymbol(t, ref, trueTy)
	trueChain := &binding{ref: ref, narrowedSymbol: trueSym, prev: chain}
	falseTy := semtypes.Diff(symbolTy, singletonTy)
	falseSym := narrowSymbol(t, ref, falseTy)
	falseChain := &binding{ref: ref, narrowedSymbol: falseSym, prev: chain}
	return expressionEffect{ifTrue: trueChain, ifFalse: falseChain}
}

var additiveSupportedTypes = semtypes.Union(semtypes.Union(semtypes.NUMBER, semtypes.STRING), semtypes.XML)

var bitWiseOpLookOrder = []semtypes.SemType{semtypes.UINT8, semtypes.UINT16, semtypes.UINT32}

func createIteratorType(env semtypes.Env, t, c semtypes.SemType) semtypes.SemType {
	od := semtypes.NewObjectDefinition()

	fields := []semtypes.Field{
		semtypes.FieldFrom("value", t, false, false),
	}
	rest := semtypes.NEVER
	recordTy := createClosedRecordType(env, fields, rest)

	resultTy := semtypes.Union(recordTy, c)

	ld := semtypes.NewListDefinition()
	listTy := ld.DefineListTypeWrapped(env, []semtypes.SemType{}, 0, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	fd := semtypes.NewFunctionDefinition()
	fnTy := fd.Define(env, listTy, resultTy, semtypes.FunctionQualifiersFrom(env, false, false))

	members := []semtypes.Member{
		{
			Name:       "next",
			ValueTy:    fnTy,
			Kind:       semtypes.MemberKindMethod,
			Visibility: semtypes.VisibilityPublic,
			Immutable:  true,
		},
	}
	return od.Define(env, semtypes.ObjectQualifiersDEFAULT, members)
}

func createClosedRecordType(env semtypes.Env, fields []semtypes.Field, rest semtypes.SemType) semtypes.SemType {
	md := semtypes.NewMappingDefinition()
	return md.DefineMappingTypeWrapped(env, fields, rest)
}

func resolveIndexBasedAccess(t typeResolver, chain *binding, expr *ast.BLangIndexBasedAccess) (semtypes.SemType, expressionEffect, bool) {
	containerExpr := expr.Expr
	containerExprTy, _, ok := resolveActionOrExpression(t, chain, containerExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	keyExpr := expr.IndexExpr
	keyExprTy, _, ok := resolveActionOrExpression(t, chain, keyExpr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	var resultTy semtypes.SemType
	tyCtx := t.typeContext()

	if semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.LIST) {
		resultTy = semtypes.ListMemberTypeInnerVal(t.typeContext(), containerExprTy, keyExprTy)
	} else if semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.Union(semtypes.MAPPING, semtypes.NIL)) {
		containerNilable := !semtypes.IsSubtype(t.typeContext(), containerExprTy, semtypes.MAPPING)
		mappingTy := containerExprTy
		if containerNilable {
			mappingTy = semtypes.Diff(containerExprTy, semtypes.NIL)
		}
		memberTy := semtypes.MappingMemberTypeInner(t.typeContext(), mappingTy, keyExprTy)
		maybeMissing := semtypes.ContainsUndef(memberTy) || containerNilable
		if maybeMissing {
			memberTy = semtypes.Union(semtypes.Diff(memberTy, semtypes.UNDEF), semtypes.NIL)
		}
		resultTy = memberTy
	} else if semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.STRING) {
		resultTy = semtypes.STRING
	} else {
		t.semanticError("unsupported container type for index based access", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	setExpectedType(expr, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func resolveFieldBaseAccess(t typeResolver, chain *binding, expr *ast.BLangFieldBaseAccess) (semtypes.SemType, expressionEffect, bool) {
	containerExprTy, _, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	key := expr.Field.Value
	if expr.IsOptionalAccess() {
		return resolveOptionalFieldBaseAccess(t, chain, expr, containerExprTy, key)
	}
	tyCtx := t.typeContext()

	var memberTy semtypes.SemType
	switch {
	case semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.OBJECT):
		memberTy = semtypes.ObjectMemberType(tyCtx, semtypes.StringConst(key), containerExprTy)
		if semtypes.IsZero(memberTy) {
			t.semanticError("field not found in object type", expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
	case semtypes.IsSubtype(tyCtx, containerExprTy, semtypes.Union(semtypes.MAPPING, semtypes.NIL)):
		containerNilable := !semtypes.IsSubtype(t.typeContext(), containerExprTy, semtypes.MAPPING)
		mappingTy := containerExprTy
		if containerNilable {
			mappingTy = semtypes.Diff(containerExprTy, semtypes.NIL)
		}
		var ok bool
		memberTy, ok = fieldBaseAccessMappingType(tyCtx, mappingTy, key, expr.IsLexpr())
		if !ok {
			t.semanticError("field base access is only possible for declared fields", expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if expr.IsCompoundAssignmentLValue() {
			readTy, readOk := fieldBaseAccessMappingType(tyCtx, mappingTy, key, false)
			writeTy, writeOk := fieldBaseAccessMappingType(tyCtx, mappingTy, key, true)
			if readOk && writeOk && !semtypes.IsSubtype(tyCtx, readTy, writeTy) {
				t.semanticError(fmt.Sprintf("incompatible type: expected %s, got %s", semtypes.ToString(tyCtx, writeTy), semtypes.ToString(tyCtx, readTy)), expr.GetPosition())
				return semtypes.SemType{}, expressionEffect{}, false
			}
		}
		if containerNilable {
			memberTy = semtypes.Union(memberTy, semtypes.NIL)
		}
	default:
		t.semanticError("unsupported container type for field access", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	setExpectedType(expr, memberTy)
	expr.Field.SetDeterminedType(semtypes.NEVER)
	return memberTy, defaultExpressionEffect(chain), true
}

func resolveOptionalFieldBaseAccess(t typeResolver, chain *binding, expr *ast.BLangFieldBaseAccess, T semtypes.SemType, fieldname string) (semtypes.SemType, expressionEffect, bool) {
	if expr.IsLexpr() {
		t.semanticError("optional field access cannot be used as an lvalue", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	tyCtx := t.typeContext()
	switch {
	case semtypes.IsSubtype(tyCtx, T, semtypes.XML):
		t.unimplemented("XML optional attribute access not supported", expr.GetPosition()) // https://github.com/ballerina-platform/ballerina-lang-go/issues/560
		return semtypes.SemType{}, expressionEffect{}, false
	case semtypes.IsSubtype(tyCtx, T, semtypes.Union(semtypes.MAPPING, semtypes.NIL)):
		Tbar := semtypes.Intersect(T, semtypes.MAPPING)
		if !semtypes.AnyMappingAtomHasFieldByName(tyCtx, Tbar, fieldname) {
			t.semanticError(fmt.Sprintf("%s is not an individual field descriptor in %s", fieldname, semtypes.ToString(tyCtx, T)), expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		K := semtypes.StringConst(fieldname)
		M := semtypes.MappingMemberTypeInner(tyCtx, Tbar, K)
		MBar := semtypes.Diff(M, semtypes.UNDEF)
		N := semtypes.NEVER
		if semtypes.IsSubtype(tyCtx, semtypes.NIL, T) || semtypes.ContainsUndef(M) {
			N = semtypes.NIL
		}
		// TODO: update for lax case https://github.com/ballerina-platform/ballerina-lang-go/issues/558
		E := semtypes.NEVER
		resultTy := semtypes.Union(semtypes.Union(MBar, N), E)
		setExpectedType(expr, resultTy)
		expr.Field.SetDeterminedType(semtypes.NEVER)
		return resultTy, defaultExpressionEffect(chain), true
	default:
		t.semanticError("optional field access must be subtype of xml|map|()", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
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
		return semtypes.Union(semtypes.Diff(memberTy, semtypes.UNDEF), semtypes.NIL), true
	}
	if isLexpr && semtypes.AllMappingAtomHasFieldByName(tyCtx, containerExprTy, key) {
		result := semtypes.Diff(memberTy, semtypes.UNDEF)
		if semtypes.AllMappingAtomsHaveOptionalFieldByName(tyCtx, containerExprTy, key) {
			result = semtypes.Union(result, semtypes.NIL)
		}
		return result, true
	}
	return semtypes.SemType{}, false
}

func resolveInvocation(t typeResolver, chain *binding, expr *ast.BLangInvocation, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	symbol := expr.RawSymbol
	if symbol == nil {
		t.internalError("invocation has no symbol", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	var (
		ty       semtypes.SemType
		effect   expressionEffect
		resolved bool
	)
	switch s := symbol.(type) {
	case *deferredMethodSymbol:
		ty, effect, resolved = resolveMethodCall(t, chain, expr, s, expectedType)
	case *model.SymbolRef:
		ty, effect, resolved = resolveFunctionCall(t, chain, expr, *s, expectedType)
	default:
		t.internalError(fmt.Sprintf("expected *model.SymbolRef, got %T", symbol), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if !resolved {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if expr.PkgAlias != nil {
		expr.PkgAlias.SetDeterminedType(semtypes.NEVER)
	}
	if expr.Name != nil {
		expr.Name.SetDeterminedType(semtypes.NEVER)
	}
	return ty, effect, true
}

func resolveMethodCall(t typeResolver, chain *binding, expr *ast.BLangInvocation, methodSymbol *deferredMethodSymbol, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	recieverTy, _, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.OBJECT) {
		return resolveObjectMethodCall(t, chain, expr, methodSymbol, expectedType)
	}
	if semtypes.IsSubtypeSimple(recieverTy, semtypes.STREAM) {
		return resolveStreamOperation(t, chain, expr, methodSymbol, expectedType)
	}
	var symbolRef model.SymbolRef
	var pkgAlias ast.BLangIdentifier
	switch {
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.LIST):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.array", methodSymbol.name, expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.INT):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.int", methodSymbol.name, expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.DECIMAL):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.decimal", methodSymbol.name, expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.FLOAT):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.float", methodSymbol.name, expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.MAPPING):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.map", methodSymbol.name, expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.ERROR):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.error", methodSymbol.name, expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.STRING):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.string", methodSymbol.name, expr)
	case semtypes.IsSubtype(t.typeContext(), recieverTy, semtypes.XML):
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.xml", methodSymbol.name, expr)
	default:
		symbolRef, pkgAlias, ok = resolveLangLibImport(t, "lang.value", methodSymbol.name, expr)
	}
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	argExprs := make([]ast.BLangExpression, len(expr.ArgExprs)+1)
	argExprs[0] = expr.Expr
	for i, arg := range expr.ArgExprs {
		argExprs[i+1] = arg
	}
	expr.SetSymbol(symbolRef)
	expr.ArgExprs = argExprs
	expr.Expr = nil
	expr.PkgAlias = &pkgAlias
	return resolveFunctionCall(t, chain, expr, symbolRef, expectedType)
}

func isRemoteMethod(t typeResolver, objType semtypes.SemType, methodName string) bool {
	ctx := t.typeContext()
	kindTy := semtypes.ObjectMemberKind(ctx, semtypes.StringConst(methodName), objType)
	return !semtypes.IsZero(kindTy) && semtypes.IsSubtype(ctx, kindTy, semtypes.StringConst("remote-method"))
}

func resolveObjectMethodCall(t typeResolver, chain *binding, expr *ast.BLangInvocation, methodSymbol *deferredMethodSymbol, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	recieverTy := expr.Expr.GetDeterminedType()
	if methodRef, ok := t.lookupClassMethodSymbol(recieverTy, methodSymbol.name); ok {
		expr.SetSymbol(methodRef)
		return resolveFunctionCall(t, chain, expr, methodRef, expectedType)
	}
	symbolRef, retTy, effect, ok := finishResolveMethodCall(t, chain, recieverTy, methodSymbol.name, methodSymbol, expr.ArgExprs, expr)
	if ok {
		expr.SetSymbol(symbolRef)
	}
	return retTy, effect, ok
}

func resolveStreamOperation(t typeResolver, chain *binding, expr *ast.BLangInvocation, methodSymbol *deferredMethodSymbol, _ semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	cx := t.typeContext()
	recieverTy := expr.Expr.GetDeterminedType()
	valueTy := semtypes.StreamValueType(cx, recieverTy)
	completionTy := semtypes.StreamCompletionType(cx, recieverTy)
	if semtypes.IsZero(valueTy) || semtypes.IsZero(completionTy) {
		t.internalError("failed to extract stream type parameters", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	var resultTy semtypes.SemType
	switch methodSymbol.name {
	case "next":
		nextRecordDefn := semtypes.NewMappingDefinition()
		nextRecord := nextRecordDefn.DefineMappingTypeWrapped(t.typeEnv(),
			[]semtypes.Field{semtypes.FieldFrom("value", valueTy, false, false)},
			semtypes.NEVER)
		resultTy = semtypes.Union(nextRecord, completionTy)
	case "close":
		resultTy = semtypes.Union(completionTy, semtypes.NIL)
	default:
		t.semanticError("stream type has no operation '"+methodSymbol.name+"'", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	expr.RawSymbol = nil
	setExpectedType(expr, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

func finishResolveMethodCall(t typeResolver, chain *binding, receiverTy semtypes.SemType, methodName string,
	methodSymbol *deferredMethodSymbol, argExprs []ast.BLangExpression, node ast.BLangNode,
) (model.SymbolRef, semtypes.SemType, expressionEffect, bool) {
	fnTy := semtypes.ObjectMemberType(t.typeContext(), semtypes.StringConst(methodName), receiverTy)
	if semtypes.IsZero(fnTy) || !semtypes.IsSubtype(t.typeContext(), fnTy, semtypes.FUNCTION) {
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
	argTys := make([]semtypes.SemType, len(argExprs))
	for i, arg := range argExprs {
		if _, namedParam := arg.(*ast.BLangNamedArgsExpression); namedParam {
			t.unimplemented("named parameters not supported for non atomic method calls", arg.GetPosition())
			return model.SymbolRef{}, semtypes.SemType{}, expressionEffect{}, false
		}
		key := semtypes.IntConst(int64(i))
		paramTy := semtypes.ListMemberTypeInnerVal(t.typeContext(), paramListTy, key)
		argTy, _, ok := resolveActionOrExpression(t, chain, arg, paramTy)
		if !ok {
			return model.SymbolRef{}, semtypes.SemType{}, expressionEffect{}, false
		}
		argTys[i] = argTy
	}
	argLd := semtypes.NewListDefinition()
	argListTy := argLd.DefineListTypeWrapped(t.typeEnv(), argTys, len(argTys), semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	retTy := semtypes.FunctionReturnType(t.typeContext(), fnTy, argListTy)
	sig := model.FunctionSignature{ParamTypes: argTys, ReturnType: retTy}
	symbolRef := t.createFunctionSymbol(methodSymbol.space, methodName, sig, fnTy)
	setExpectedType(node, retTy)
	return symbolRef, retTy, defaultExpressionEffect(chain), true
}

func resolveResourceMethodSignature(t typeResolver, isClient bool, isService bool, method *ast.BLangResourceMethod) bool {
	if !isClient && !isService {
		t.semanticError("resource methods are only allowed in client or service classes", method.GetPosition())
		return false
	}
	sym, ok := t.getSymbol(method.Symbol()).(*model.ResourceMethodSymbol)
	if !ok {
		t.internalError("expected resource method symbol", method.GetPosition())
		return false
	}
	pathTy, pathParamRefs, ok := resolveResourcePathType(t, method)
	if !ok {
		return false
	}
	sym.SetPathListType(pathTy)
	sym.SetPathParams(pathParamRefs)

	_, _, _, _, ok = resolveInvokableSignature(t, method, sym, method.RequiredParams)
	return ok
}

func resolveResourcePathType(t typeResolver, method *ast.BLangResourceMethod) (semtypes.SemType, []model.SymbolRef, bool) {
	anydata := semtypes.CreateAnydata(t.typeContext())
	var members []semtypes.SemType
	restMember := semtypes.NEVER
	var paramRefs []model.SymbolRef
	for i := range method.ResourcePath {
		seg := &method.ResourcePath[i]
		switch seg.Kind {
		case ast.ResourcePathSegmentName:
			literalTy := semtypes.StringConst(seg.Name)
			seg.SetDeterminedType(literalTy)
			members = append(members, literalTy)
		case ast.ResourcePathSegmentParam, ast.ResourcePathSegmentParamRest:
			if seg.ParamType == nil {
				t.internalError("resource path parameter is missing type", seg.GetPosition())
				return semtypes.SemType{}, nil, false
			}
			paramTy, ok := resolveBType(t, seg.ParamType, 0)
			if !ok {
				return semtypes.SemType{}, nil, false
			}
			if !semtypes.IsSubtype(t.typeContext(), paramTy, anydata) {
				// Not sure if we should allow anydata here? spec says it can be anydata but jBallerina only allow simple basic types
				t.semanticError("resource path parameter type must be a subtype of anydata", seg.GetPosition())
				return semtypes.SemType{}, nil, false
			}
			seg.SetDeterminedType(paramTy)
			symbolTy := paramTy
			if seg.Kind == ast.ResourcePathSegmentParamRest {
				restListDefn := semtypes.NewListDefinition()
				symbolTy = restListDefn.DefineListTypeWrapped(t.typeEnv(), []semtypes.SemType{}, 0, paramTy, semtypes.CellMutability_CELL_MUT_NONE)
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
	pathTy := listDefn.DefineListTypeWrapped(t.typeEnv(), members, len(members), restMember, semtypes.CellMutability_CELL_MUT_NONE)
	return pathTy, paramRefs, true
}

func resolveClientResourceAccessAction(t typeResolver, chain *binding, expr *ast.BLangClientResourceAccessAction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	receiverTy, _, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if !semtypes.IsClientObject(t.typeContext(), receiverTy) {
		t.semanticError("resource access action is only allowed on client objects", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	atomicType := semtypes.ToObjectAtomicType(t.typeContext(), receiverTy)
	if atomicType == nil {
		t.unimplemented("non-atomic receiver for resource access action", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	classRef, ok := t.getClassAtomSymbol(atomicType)
	if !ok {
		t.internalError("failed to find class definition for receiver type", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	networkSym, ok := t.getSymbol(classRef).(*model.NetworkClassSymbol)
	if !ok {
		t.internalError("client reciever must have network class symbol", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	argPathTy, _, ok := resolveResourceAccessPathType(t, chain, expr)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	methodName := expr.MethodName
	var matches []model.SymbolRef
	for _, rmRef := range networkSym.ResourceMethods() {
		rmSym, ok := t.getSymbol(rmRef).(*model.ResourceMethodSymbol)
		if !ok {
			t.internalError("expected resource method symbol", expr.GetPosition())
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if rmSym.MethodName() != methodName || !semtypes.IsSubtype(t.typeContext(), argPathTy, rmSym.PathListType()) {
			continue
		}
		matches = append(matches, rmRef)
	}
	if len(matches) == 0 {
		t.semanticError(fmt.Sprintf("no matching resource method '%s'", methodName), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if len(matches) > 1 {
		t.unimplemented("ambiguous resource method dispatch", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	expr.SetMethodSymbol(matches[0])
	return resolveFunctionCall(t, chain, expr, matches[0], expectedType)
}

func resolveResourceAccessPathType(t typeResolver, chain *binding, expr *ast.BLangClientResourceAccessAction) (semtypes.SemType, int, bool) {
	var members []semtypes.SemType
	for i := range expr.Path {
		seg := &expr.Path[i]
		switch seg.Kind {
		case ast.ResourceAccessSegmentName:
			members = append(members, semtypes.StringConst(seg.Name))
		case ast.ResourceAccessSegmentComputed:
			segTy, _, ok := resolveActionOrExpression(t, chain, seg.Expr, semtypes.SemType{})
			if !ok {
				return semtypes.SemType{}, 0, false
			}
			members = append(members, segTy)
		}
	}
	listDefn := semtypes.NewListDefinition()
	pathTy := listDefn.DefineListTypeWrapped(t.typeEnv(), members, len(members), semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	return pathTy, len(members), true
}

func resolveRemoteMethodCallAction(t typeResolver, chain *binding, expr *ast.BLangRemoteMethodCallAction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	receiverTy, _, ok := resolveActionOrExpression(t, chain, expr.Expr, semtypes.SemType{})
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	if !semtypes.IsClientObject(t.typeContext(), receiverTy) {
		t.semanticError("remote method call is only allowed on client objects", expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	methodName := expr.Name.GetValue()
	remoteMethodName := model.RemoteMethodName(methodName)
	if !isRemoteMethod(t, receiverTy, remoteMethodName) {
		t.semanticError(fmt.Sprintf("%s is not a remote method", methodName), expr.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}
	expr.Name.SetDeterminedType(semtypes.NEVER)
	if methodRef, ok := t.lookupClassMethodSymbol(receiverTy, remoteMethodName); ok {
		expr.SetMethodSymbol(methodRef)
		return resolveFunctionCall(t, chain, expr, methodRef, expectedType)
	}
	symbolRef, retTy, effect, ok := finishResolveMethodCall(t, chain, receiverTy, remoteMethodName, expr.RawSymbol.(*deferredMethodSymbol), expr.ArgExprs, expr)
	if ok {
		expr.SetMethodSymbol(symbolRef)
	}
	return retTy, effect, ok
}

func resolveLangLibImport(t typeResolver, pkgName string, methodName string, expr *ast.BLangInvocation) (model.SymbolRef, ast.BLangIdentifier, bool) {
	symbolSpace, ok := t.lookupImportedSymbols(pkgName)
	if !ok {
		t.internalError(fmt.Sprintf("%s symbol space not found", pkgName), expr.GetPosition())
		return model.SymbolRef{}, ast.BLangIdentifier{}, false
	}
	basePos := expr.GetPosition()
	pkgAlias := ast.BLangIdentifier{Value: pkgName}
	pkgAlias.SetPosition(basePos)
	if !t.hasImplicitImport(pkgName) {
		moduleName := strings.TrimPrefix(pkgName, "lang.")
		orgIdent := &ast.BLangIdentifier{Value: "ballerina"}
		langIdent := ast.BLangIdentifier{Value: "lang"}
		moduleIdent := ast.BLangIdentifier{Value: moduleName}
		setPositions(basePos, orgIdent, &langIdent, &moduleIdent)
		importNode := ast.BLangImportPackage{
			OrgName:      orgIdent,
			PkgNameComps: []ast.BLangIdentifier{langIdent, moduleIdent},
			Alias:        &pkgAlias,
		}
		setOtherNodesAsNever(&importNode)
		t.addImplicitImport(pkgName, importNode)
	}
	symbolRef, ok := symbolSpace.GetSymbol(methodName)
	if !ok {
		t.semanticError("method not found: "+methodName, expr.GetPosition())
		return model.SymbolRef{}, ast.BLangIdentifier{}, false
	}
	return symbolRef, pkgAlias, true
}

func resolveFunctionCallArgs(t typeResolver, chain *binding, inv invocable, fnSymbol model.SymbolRef, expectedType semtypes.SemType) ([]semtypes.SemType, model.SymbolRef, *binding, bool) {
	baseSymbol := t.getSymbol(fnSymbol)
	switch sym := baseSymbol.(type) {
	case model.DependentlyTypedFunctionSymbol:
		argTys, chain, ok := argArray(t, sym, sym.ParamTypes(), semtypes.SemType{}, chain, inv, expectedType)
		if !ok {
			return nil, fnSymbol, chain, false
		}
		monoName := t.nextMonoFnName(sym.Name())
		monoSym := sym.Monomorphize(t.typeContext(), monoName, fnSymbol, argTys)
		scope := t.currentScope()
		scope.AddSymbol(monoName, monoSym)
		monoRef, ok := scope.GetSymbol(monoName)
		if !ok {
			t.internalError("monomorphized symbol missing from scope", inv.GetPosition())
			return nil, fnSymbol, chain, false
		}
		monoSym.SetType(typeFromFunctionSignature(t, monoSym.Signature()))
		inv.SetResolvedSymbol(monoRef)
		return argTys, monoRef, chain, true
	case *model.OpaqueFunctionSymbol:
		pkg := t.compilerContext().SymbolPackage(fnSymbol)
		mono, ok := opaqueFunctionMonomorphizerFor(
			pkg.Organization,
			pkg.Package,
			sym.OpaqueID(),
		)
		if !ok {
			t.internalError("no monomorphizer for opaque function", inv.GetPosition())
			return nil, fnSymbol, chain, false
		}
		symbolRef, ok := mono(t, sym, fnSymbol, chain, inv.CallArgs(), inv.GetPosition())
		if !ok {
			return nil, fnSymbol, chain, false
		}
		fnSym := t.getSymbol(symbolRef).(model.FunctionSymbol)
		sig := fnSym.Signature()
		argTys, chain, ok := argArray(t, fnSym, sig.ParamTypes, sig.RestParamType, chain, inv, expectedType)
		if !ok {
			return nil, fnSymbol, chain, false
		}
		inv.SetResolvedSymbol(symbolRef)
		return argTys, symbolRef, chain, true
	case model.FunctionSymbol:
		if !t.ensureResolved(fnSymbol, 0) {
			return nil, fnSymbol, chain, false
		}
		sig := sym.Signature()
		argTys, chain, ok := argArray(t, sym, sig.ParamTypes, sig.RestParamType, chain, inv, expectedType)
		return argTys, fnSymbol, chain, ok
	case model.ValueSymbol:
		narrowedSymbol := lookupSymbol(chain, fnSymbol)
		inv.SetResolvedSymbol(narrowedSymbol)
		fnTy := t.symbolType(narrowedSymbol)
		if semtypes.IsZero(fnTy) {
			t.internalError("function symbol has no type", inv.GetPosition())
			return nil, narrowedSymbol, chain, false
		}
		if !semtypes.IsSubtype(t.typeContext(), fnTy, semtypes.FUNCTION) {
			t.semanticError("not a function value", inv.GetPosition())
			return nil, narrowedSymbol, chain, false
		}

		paramListTy := semtypes.FunctionParamListType(t.typeContext(), fnTy)
		if semtypes.IsZero(paramListTy) {
			// I don't think this can happen given we have already checked fnTy to be subtype of function
			t.internalError("empty function param list ty", inv.GetPosition())
			return nil, narrowedSymbol, chain, false
		}
		var argTys []semtypes.SemType
		for i, arg := range inv.CallArgs() {
			if _, namedParam := arg.(*ast.BLangNamedArgsExpression); namedParam {
				t.unimplemented("named parameters not supported for lambdas", arg.GetPosition())
				return nil, narrowedSymbol, chain, false
			}
			key := semtypes.IntConst(int64(i))
			paramTy := semtypes.ListMemberTypeInnerVal(t.typeContext(), paramListTy, key)
			argTy, argEffect, ok := resolveActionOrExpression(t, chain, arg, paramTy)
			if !ok {
				return nil, narrowedSymbol, chain, false
			}
			chain = argEffect.ifTrue
			argTys = append(argTys, argTy)
		}
		return argTys, narrowedSymbol, chain, true
	default:
		t.semanticError("not a function value", inv.GetPosition())
		return nil, fnSymbol, chain, false
	}
}

type mappingField struct {
	name string
	expr ast.BLangExpression
}

type valueSlot struct {
	expr ast.BLangExpression
}

type mappingSlot struct {
	recordTy    semtypes.SemType
	fields      []mappingField
	synthesized *ast.BLangMappingConstructorExpr
}

// marker interface for slots
type argSlot interface{ isArgSlot() }

func (*valueSlot) isArgSlot()   {}
func (*mappingSlot) isArgSlot() {}

func argArray(t typeResolver, sym model.FunctionSymbol, paramTypes []semtypes.SemType, restParamTy semtypes.SemType, chain *binding, inv invocable, callExpectedType semtypes.SemType) ([]semtypes.SemType, *binding, bool) {
	args := inv.CallArgs()
	loc := inv.GetPosition()
	paramNames := sym.ParamNames()
	nRequired := len(paramNames)

	inclInfo := sym.IncludedRecordParams()

	slots := make([]argSlot, nRequired)
	namedArgsByIndex := make(map[int]*ast.BLangNamedArgsExpression)
	seen := make(map[string]bool)
	var restArgs []ast.BLangExpression

	for i, arg := range args {
		switch a := arg.(type) {
		case *ast.BLangNamedArgsExpression:
			name := a.Name.Value
			if seen[name] {
				t.semanticError(fmt.Sprintf("duplicate arguments for %s", name), a.GetPosition())
				return nil, chain, false
			}
			seen[name] = true

			if idx := paramIndexOf(paramNames, name); idx >= 0 {
				switch slots[idx].(type) {
				case nil:
					// ok
				case *valueSlot:
					t.semanticError(fmt.Sprintf("repeated values for parameter %s", name), a.GetPosition())
					return nil, chain, false
				case *mappingSlot:
					t.semanticError(
						fmt.Sprintf("record value and field-level arguments for the same included record parameter '%s'", paramNames[idx]),
						a.GetPosition())
					return nil, chain, false
				}
				slots[idx] = &valueSlot{expr: a.Expr}
				namedArgsByIndex[idx] = a
				a.Name.DeterminedType = semtypes.NEVER
				continue
			}

			if inclInfo != nil {
				argTy, _, ok := resolveActionOrExpression(t, chain, a.Expr, semtypes.SemType{})
				if !ok {
					return nil, chain, false
				}
				idx, ok := includedRecordArgIndex(t, inclInfo, paramTypes, name, argTy, a.GetPosition())
				if !ok {
					return nil, chain, false
				}
				switch s := slots[idx].(type) {
				case nil:
					slots[idx] = &mappingSlot{
						recordTy: paramTypes[idx],
						fields:   []mappingField{{name: name, expr: a.Expr}},
					}
				case *valueSlot:
					t.semanticError(
						fmt.Sprintf("record value and field-level arguments for the same included record parameter '%s'", paramNames[idx]),
						a.GetPosition())
					return nil, chain, false
				case *mappingSlot:
					s.fields = append(s.fields, mappingField{name: name, expr: a.Expr})
				}
				a.Name.DeterminedType = semtypes.NEVER
				continue
			}

			t.semanticError(fmt.Sprintf("no such parameter %s", name), a.GetPosition())
			return nil, chain, false

		default:
			if i >= nRequired {
				restArgs = append(restArgs, arg)
				continue
			}
			switch slots[i].(type) {
			case nil:
				slots[i] = &valueSlot{expr: arg}
			case *valueSlot:
				t.semanticError(fmt.Sprintf("repeated values for parameter %s", paramNames[i]), arg.GetPosition())
				return nil, chain, false
			case *mappingSlot:
				t.semanticError(
					fmt.Sprintf("record value and field-level arguments for the same included record parameter '%s'", paramNames[i]),
					arg.GetPosition())
				return nil, chain, false
			}
		}
	}

	if inclInfo != nil {
		for i := 0; i < inclInfo.Len(); i++ {
			if !inclInfo.IsIncluded(i) {
				continue
			}
			if slots[i] == nil {
				slots[i] = &mappingSlot{recordTy: paramTypes[i]}
			}
		}
	}

	tys := make([]semtypes.SemType, 0, nRequired+len(restArgs))
	for i := range nRequired {
		switch s := slots[i].(type) {
		case nil:
			dp, isDefaultable := sym.DefaultableParams().Get(i)
			if !isDefaultable {
				t.semanticError(fmt.Sprintf("missing required parameter '%s'", paramNames[i]), loc)
				return nil, chain, false
			}
			if dp.Kind == model.DefaultableParamKindInferredTypedesc {
				if semtypes.IsZero(callExpectedType) {
					t.semanticError(fmt.Sprintf("cannot infer typedesc argument for parameter '%s': no contextually expected type", paramNames[i]), loc)
					return nil, chain, false
				}
				ctx := t.typeContext()
				T := semtypes.TypedescConstraint(ctx, paramTypes[i])
				S := semtypes.Intersect(T, callExpectedType)
				if semtypes.IsEmpty(ctx, S) {
					t.semanticError(fmt.Sprintf("cannot infer maximal type such that it is a subtype of both %s and %s", semtypes.ToString(ctx, T), semtypes.ToString(ctx, callExpectedType)), loc)
					return nil, chain, false
				}
				tys = append(tys, semtypes.TypedescContaining(t.typeEnv(), S))
				continue
			}
			tys = append(tys, paramTypes[i])

		case *valueSlot:
			ty, effect, ok := resolveActionOrExpression(t, chain, s.expr, paramTypes[i])
			if !ok {
				return nil, chain, false
			}
			if n, has := namedArgsByIndex[i]; has {
				n.DeterminedType = ty
			}
			// parameters have narrowing effects like capture
			chain = effect.ifTrue
			tys = append(tys, ty)

		case *mappingSlot:
			mc, effect, ok := resolveIncludedRecordSlot(t, chain, s, loc)
			if !ok {
				return nil, chain, false
			}
			chain = effect.ifTrue
			s.synthesized = mc
			tys = append(tys, paramTypes[i])
		}
	}

	for _, arg := range restArgs {
		ty, effect, ok := resolveActionOrExpression(t, chain, arg, restParamTy)
		if !ok {
			return nil, chain, false
		}
		chain = effect.ifTrue
		tys = append(tys, ty)
	}

	if inclInfo != nil {
		rewriteCallArgsForIncludedRecords(inv, args, slots, paramNames, inclInfo)
	}
	return tys, chain, true
}

func includedRecordArgIndex(t typeResolver, inclInfo *model.IncludedRecordParamInfo, paramTypes []semtypes.SemType, name string, argTy semtypes.SemType, pos diagnostics.Location) (int, bool) {
	var explicitMatches []int
	var restMatches []int
	keyTy := semtypes.StringConst(name)
	for i := 0; i < inclInfo.Len(); i++ {
		if !inclInfo.IsIncluded(i) {
			continue
		}
		memberTy := semtypes.MappingMemberTypeInnerVal(t.typeContext(), paramTypes[i], keyTy)
		if semtypes.IsEmpty(t.typeContext(), memberTy) || !semtypes.IsSubtype(t.typeContext(), argTy, memberTy) {
			continue
		}
		if includedRecordParamHasField(inclInfo, i, name) {
			explicitMatches = append(explicitMatches, i)
		} else {
			restMatches = append(restMatches, i)
		}
	}
	matches := explicitMatches
	if len(matches) == 0 {
		matches = restMatches
	}
	switch len(matches) {
	case 0:
		t.semanticError(fmt.Sprintf("no included record parameter accepts named argument '%s'", name), pos)
		return -1, false
	case 1:
		return matches[0], true
	default:
		t.semanticError(fmt.Sprintf("named argument '%s' matches multiple included record parameters", name), pos)
		return -1, false
	}
}

func includedRecordParamHasField(inclInfo *model.IncludedRecordParamInfo, index int, name string) bool {
	for _, fieldName := range inclInfo.Fields(index) {
		if fieldName == name {
			return true
		}
	}
	return false
}

func paramIndexOf(paramNames []string, name string) int {
	for i, each := range paramNames {
		if each == name {
			return i
		}
	}
	return -1
}

func resolveIncludedRecordSlot(t typeResolver, chain *binding, s *mappingSlot, loc diagnostics.Location) (*ast.BLangMappingConstructorExpr, expressionEffect, bool) {
	mat := semtypes.ToMappingAtomicType(t.typeContext(), s.recordTy)
	if mat == nil {
		t.semanticError("included record parameter is not an atomic mapping type", loc)
		return nil, expressionEffect{}, false
	}
	mc := &ast.BLangMappingConstructorExpr{}
	mc.SetPosition(loc)
	fields := make([]ast.MappingField, 0, len(s.fields))
	effect := defaultExpressionEffect(chain)
	for _, f := range s.fields {
		fieldTy := mat.FieldInnerVal(f.name)
		_, fe, ok := resolveActionOrExpression(t, chain, f.expr, fieldTy)
		if !ok {
			return nil, expressionEffect{}, false
		}
		chain = fe.ifTrue
		effect = fe
		keyLit := &ast.BLangLiteral{Value: f.name, OriginalValue: f.name}
		keyLit.SetPosition(f.expr.GetPosition())
		keyLit.SetValueType(ast.NewBType(ast.TypeTags_STRING, model.Name(""), 0))
		keyLit.SetDeterminedType(semtypes.STRING)
		kv := &ast.BLangMappingKeyValueField{
			Key:       &ast.BLangMappingKey{Expr: keyLit},
			ValueExpr: f.expr,
		}
		kv.Key.SetPosition(f.expr.GetPosition())
		kv.SetPosition(f.expr.GetPosition())
		kv.Key.SetDeterminedType(semtypes.NEVER)
		kv.SetDeterminedType(semtypes.NEVER)
		fields = append(fields, kv)
	}
	mc.Fields = fields
	mc.AtomicType = *mat
	if ref, ok := t.getMappingAtomSymRef(mat); ok {
		if carrier, ok := t.getSymbol(ref).(model.MemberCarrier); ok {
			mc.FieldDefaults = carrier.FieldDefaults()
		}
	}
	mc.SetDeterminedType(s.recordTy)
	return mc, effect, true
}

// I don't like this. But we are doing this to avoid having to do this in both desugar and semantic analysis. Ideally this should be in the desugar
func rewriteCallArgsForIncludedRecords(inv invocable, origArgs []ast.BLangExpression, slots []argSlot, paramNames []string, inclInfo *model.IncludedRecordParamInfo) {
	consumedFields := make(map[*ast.BLangNamedArgsExpression]bool)
	for i := 0; i < inclInfo.Len(); i++ {
		ms, ok := slots[i].(*mappingSlot)
		if !ok {
			continue
		}
		for _, field := range ms.fields {
			for _, arg := range origArgs {
				if named, ok := arg.(*ast.BLangNamedArgsExpression); ok && named.Expr == field.expr && named.Name.Value == field.name {
					consumedFields[named] = true
					break
				}
			}
		}
	}

	newArgs := make([]ast.BLangExpression, 0, len(origArgs))
	for _, arg := range origArgs {
		if named, ok := arg.(*ast.BLangNamedArgsExpression); ok && consumedFields[named] {
			continue
		}
		newArgs = append(newArgs, arg)
	}
	for i := 0; i < inclInfo.Len(); i++ {
		ms, ok := slots[i].(*mappingSlot)
		if !ok || ms.synthesized == nil {
			continue
		}
		mc := ms.synthesized
		named := &ast.BLangNamedArgsExpression{
			Name: ast.BLangIdentifier{Value: paramNames[i]},
			Expr: mc,
		}
		named.SetPosition(mc.GetPosition())
		named.Name.SetDeterminedType(semtypes.NEVER)
		named.SetDeterminedType(mc.GetDeterminedType())
		newArgs = append(newArgs, named)
	}
	inv.SetCallArgs(newArgs)
}

func resolveFunctionCall(t typeResolver, chain *binding, inv invocable, symbolRef model.SymbolRef, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	argTys, symbolRef, chain, ok := resolveFunctionCallArgs(t, chain, inv, symbolRef, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}

	argLd := semtypes.NewListDefinition()
	argListTy := argLd.DefineListTypeWrapped(t.typeEnv(), argTys, len(argTys), semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)

	retTy := semtypes.FunctionReturnType(t.typeContext(), t.symbolType(symbolRef), argListTy)
	if semtypes.IsZero(retTy) {
		// This can only happen when function call is not well-typed and since we
		// ensure funcTy is a function subtype, this can only be caused by invalid args
		t.semanticError("incompatible arguments for function call", inv.GetPosition())
		return semtypes.SemType{}, expressionEffect{}, false
	}

	setExpectedType(inv, retTy)
	return retTy, defaultExpressionEffect(chain), true
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
	sig := model.FunctionSignature{
		ParamTypes:    paramTypes,
		ReturnType:    retTy,
		RestParamType: semtypes.NEVER,
		Flags:         depSym.FuncFlags(),
	}
	return typeFromFunctionSignature(t, sig)
}

func typeFromFunctionSignature(t typeResolver, sig model.FunctionSignature) semtypes.SemType {
	paramListDefn := semtypes.NewListDefinition()
	paramListTy := paramListDefn.DefineListTypeWrapped(t.typeEnv(), sig.ParamTypes, len(sig.ParamTypes), sig.RestParamType, semtypes.CellMutability_CELL_MUT_NONE)
	fnDefn := semtypes.NewFunctionDefinition()
	return fnDefn.Define(t.typeEnv(), paramListTy, sig.ReturnType,
		semtypes.FunctionQualifiersFrom(t.typeEnv(), sig.IsIsolated(), sig.IsTransactional()))
}

func resolveFixedArraySize(t typeResolver, lenExp ast.BLangExpression) (int, bool) {
	actionOrExpr, ok := lenExp.(ast.BLangActionOrExpression)
	if !ok {
		t.semanticError("fixed-length array size must be a singleton int", lenExp.GetPosition())
		return 0, false
	}
	if _, _, ok := resolveActionOrExpression(t, nil, actionOrExpr, semtypes.INT); !ok {
		return 0, false
	}
	sizeTy := lenExp.GetDeterminedType()
	if semtypes.IsZero(sizeTy) || !semtypes.IsSubtype(t.typeContext(), sizeTy, semtypes.INT) {
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

func resolveBType(t typeResolver, btype ast.BType, depth int) (semtypes.SemType, bool) {
	bLangNode := btype.(ast.BLangNode)
	if !semtypes.IsZero(bLangNode.GetDeterminedType()) {
		return bLangNode.GetDeterminedType(), true
	}
	res, ok := resolveBTypeInner(t, btype, depth)
	if !ok {
		return semtypes.SemType{}, false
	}
	bLangNode.SetDeterminedType(res)
	typeData := btype.GetTypeData()
	typeData.Type = res
	btype.SetTypeData(typeData)
	return res, true
}

func resolveTypeDataPair(t typeResolver, typeData *ast.TypeData, depth int) (semtypes.SemType, bool) {
	ty, ok := resolveBType(t, typeData.TypeDescriptor.(ast.BType), depth)
	if !ok {
		return semtypes.SemType{}, false
	}
	typeData.Type = ty
	return ty, true
}

func resolveBTypeInner(t typeResolver, btype ast.BType, depth int) (semtypes.SemType, bool) {
	switch ty := btype.(type) {
	case *ast.BLangReturnTypeDescriptor:
		return resolveBType(t, ty.TypeDescriptor, depth)
	case *ast.BLangValueType:
		switch ty.TypeKind {
		case ast.TypeKind_BOOLEAN:
			return semtypes.BOOLEAN, true
		case ast.TypeKind_INT:
			return semtypes.INT, true
		case ast.TypeKind_FLOAT:
			return semtypes.FLOAT, true
		case ast.TypeKind_STRING:
			return semtypes.STRING, true
		case ast.TypeKind_NIL:
			return semtypes.NIL, true
		case ast.TypeKind_ANY:
			return semtypes.ANY, true
		case ast.TypeKind_DECIMAL:
			return semtypes.DECIMAL, true
		case ast.TypeKind_BYTE:
			return semtypes.BYTE, true
		case ast.TypeKind_ANYDATA:
			return semtypes.CreateAnydata(t.typeContext()), true
		case ast.TypeKind_HANDLE:
			return semtypes.HANDLE, true
		case ast.TypeKind_TYPEDESC:
			return semtypes.TYPEDESC, true
		case ast.TypeKind_XML:
			return semtypes.XML, true
		case ast.TypeKind_READONLY:
			return semtypes.VAL_READONLY, true
		case ast.TypeKind_NEVER:
			return semtypes.NEVER, true
		default:
			t.internalError("unexpected type tag", diagnostics.Location{})
			return semtypes.SemType{}, false
		}
	case *ast.BLangArrayType:
		defn := ty.Definition
		var semTy semtypes.SemType
		if defn == nil {
			d := semtypes.NewListDefinition()
			ty.Definition = &d
			elemTy, ok := resolveTypeDataPair(t, &ty.Elemtype, depth+1)
			if !ok {
				return semtypes.SemType{}, false
			}
			for i := len(ty.Sizes); i > 0; i-- {
				lenExp := ty.Sizes[i-1]
				if lenExp == nil {
					elemTy = d.DefineListTypeWrappedWithEnvSemType(t.typeEnv(), elemTy)
				} else {
					length, ok := resolveFixedArraySize(t, lenExp)
					if !ok {
						return semtypes.SemType{}, false
					}
					elemTy = d.DefineListTypeWrappedWithEnvSemTypesInt(t.typeEnv(), []semtypes.SemType{elemTy}, length)
				}
			}
			semTy = elemTy
		} else {
			semTy = defn.GetSemType(t.typeEnv())
		}
		return semTy, true
	case *ast.BLangUnionTypeNode:
		lhs, ok := resolveTypeDataPair(t, ty.Lhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, false
		}
		rhs, ok := resolveTypeDataPair(t, ty.Rhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, false
		}
		return semtypes.Union(lhs, rhs), true
	case *ast.BLangIntersectionTypeNode:
		lhs, ok := resolveTypeDataPair(t, ty.Lhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, false
		}
		rhs, ok := resolveTypeDataPair(t, ty.Rhs(), depth+1)
		if !ok {
			return semtypes.SemType{}, false
		}
		result := semtypes.Intersect(lhs, rhs)
		pos := ty.GetPosition()
		if !t.ensureNotEmpty(result, func() {
			t.semanticError("intersection type is empty (equivalent to never)", pos)
		}) {
			return semtypes.SemType{}, false
		}
		return result, true
	case *ast.BLangErrorTypeNode:
		if ty.IsTop() {
			return semtypes.ERROR, true
		} else {
			detailTy, ok := resolveBType(t, ty.DetailType.TypeDescriptor.(ast.BType), depth+1)
			if !ok {
				return semtypes.SemType{}, false
			}
			ty.DetailType.Type = detailTy
			return semtypes.ErrorWithDetail(detailTy), true
		}
	case *ast.BLangUserDefinedType:
		setOtherNodesAsNever(&ty.TypeName)
		setOtherNodesAsNever(&ty.PkgAlias)
		symbol := ty.Symbol()
		if ty.PkgAlias.Value != "" {
			return t.symbolType(symbol), true
		}
		if !t.ensureResolved(symbol, depth) {
			return semtypes.SemType{}, false
		}
		return t.symbolType(symbol), true
	case *ast.BLangFiniteTypeNode:
		result := semtypes.NEVER
		for _, value := range ty.ValueSpace {
			valueTy, _, ok := resolveActionOrExpression(t, nil, value, semtypes.SemType{})
			if !ok {
				return semtypes.SemType{}, false
			}
			result = semtypes.Union(result, valueTy)
		}
		return result, true
	case *ast.BLangConstrainedType:
		if _, ok := resolveTypeDataPair(t, &ty.Type, depth+1); !ok {
			return semtypes.SemType{}, false
		}
		defn := ty.Definition
		if defn == nil {
			switch ty.ConstraintKind() {
			case ast.TypeKind_MAP:
				d := semtypes.NewMappingDefinition()
				ty.Definition = &d
				rest, ok := resolveTypeDataPair(t, &ty.Constraint, depth+1)
				if !ok {
					return semtypes.SemType{}, false
				}
				semType := d.DefineMappingTypeWrapped(t.typeEnv(), nil, rest)
				mat := semtypes.ToMappingAtomicType(t.typeContext(), semType)
				t.setMappingAtomBType(mat, ty)
				return semType, true
			case ast.TypeKind_TYPEDESC:
				constraint, ok := resolveTypeDataPair(t, &ty.Constraint, depth+1)
				if !ok {
					return semtypes.SemType{}, false
				}
				return semtypes.TypedescContaining(t.typeEnv(), constraint), true
			case ast.TypeKind_XML:
				constraint, ok := resolveTypeDataPair(t, &ty.Constraint, depth+1)
				if !ok {
					return semtypes.SemType{}, false
				}
				if !semtypes.IsSubtype(t.typeContext(), constraint, semtypes.XML) {
					t.semanticError(fmt.Sprintf("xml type constraint must be a subtype of xml, got %s", semtypes.ToString(t.typeContext(), constraint)), ty.GetPosition())
					return semtypes.SemType{}, false
				}
				return semtypes.XMLSequence(constraint), true
			default:
				t.unimplemented("unsupported base type kind", diagnostics.Location{})
				return semtypes.SemType{}, false
			}
		} else {
			return defn.GetSemType(t.typeEnv()), true
		}
	case *ast.BLangBuiltInRefTypeNode:
		switch ty.TypeKind {
		case ast.TypeKind_MAP:
			return semtypes.MAPPING, true
		case ast.TypeKind_JSON:
			return semtypes.CreateJSON(t.typeContext()), true
		case ast.TypeKind_ANYDATA:
			return semtypes.CreateAnydata(t.typeContext()), true
		case ast.TypeKind_ANY:
			return semtypes.ANY, true
		case ast.TypeKind_XML:
			return semtypes.XML, true
		case ast.TypeKind_STREAM:
			return semtypes.STREAM, true
		case ast.TypeKind_TABLE, ast.TypeKind_FUTURE:
			t.unimplemented("unsupported builtin type kind: "+string(ty.TypeKind), ty.GetPosition())
			return semtypes.SemType{}, false
		default:
			t.internalError("Unexpected builtin type kind", ty.GetPosition())
		}
		return semtypes.SemType{}, false
	case *ast.BLangStreamType:
		if defn := ty.Definition; defn != nil {
			return defn.GetSemType(t.typeEnv()), true
		}
		valueTy, ok := resolveTypeDataPair(t, &ty.ValueType, depth+1)
		if !ok {
			return semtypes.SemType{}, false
		}
		completionTy, ok := resolveTypeDataPair(t, &ty.CompletionType, depth+1)
		if !ok {
			return semtypes.SemType{}, false
		}
		if !semtypes.IsSubtype(t.typeContext(), completionTy, semtypes.Union(semtypes.ERROR, semtypes.NIL)) {
			t.semanticError(
				"stream completion type must be a subtype of error?",
				ty.CompletionType.TypeDescriptor.GetPosition(),
			)
			return semtypes.SemType{}, false
		}
		d := semtypes.NewStreamDefinition()
		ty.Definition = &d
		return d.Define(t.typeEnv(), valueTy, completionTy), true
	case *ast.BLangTupleTypeNode:
		defn := ty.Definition
		if defn == nil {
			d := semtypes.NewListDefinition()
			ty.Definition = &d
			members := make([]semtypes.SemType, len(ty.Members))
			for i, member := range ty.Members {
				memberTy, ok := resolveBType(t, member.TypeDesc.(ast.BType), depth+1)
				if !ok {
					return semtypes.SemType{}, false
				}
				members[i] = memberTy
			}
			rest, ok := semtypes.NEVER, true //nolint:ineffassign // ok default overwritten when ty.Rest is non-nil
			if ty.Rest != nil {
				rest, ok = resolveBType(t, ty.Rest, depth+1)
				if !ok {
					return semtypes.SemType{}, false
				}
			}
			return d.DefineListTypeWrappedWithEnvSemTypesSemType(t.typeEnv(), members, rest), true
		}
		return defn.GetSemType(t.typeEnv()), true
	case *ast.BLangRecordType:
		defn := ty.Definition
		if defn != nil {
			return defn.GetSemType(t.typeEnv()), true
		}
		d := semtypes.NewMappingDefinition()
		ty.Definition = &d

		// Resolve and collect included members from symbols
		result, ok := resolveRecordInclusions(t, ty, depth)
		if !ok {
			return semtypes.SemType{}, false
		}

		seen := make(map[string]bool)
		var fields []semtypes.Field
		// TODO: need to think of a way to unify this with objects
		for name, field := range ty.FieldPtrs() {
			if seen[name] {
				t.semanticError(fmt.Sprintf("duplicate field name '%s'", name), field.GetPosition())
				return semtypes.SemType{}, false
			}
			seen[name] = true
			fieldTy, ok := resolveBType(t, field.Type, depth+1)
			if !ok {
				return semtypes.SemType{}, false
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
			if field.DefaultExpr != nil {
				if _, _, ok := resolveActionOrExpression(t, nil, field.DefaultExpr, fieldTy); !ok {
					return semtypes.SemType{}, false
				}
				field.DefaultFnRef = allocateDefaultFnSymbol(t, fieldTy)
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
			rest, ok = resolveBType(t, ty.RestType, depth+1)
			if !ok {
				return semtypes.SemType{}, false
			}
		} else if ty.IsOpen {
			rest = semtypes.CreateAnydata(t.typeContext())
		} else if result.multpleRestTy {
			t.semanticError("included rest type declared in multiple type inclusions must be overridden", ty.GetPosition())
			rest = semtypes.NEVER
		} else if !semtypes.IsZero(result.includedRestTy) {
			rest = result.includedRestTy
		} else {
			rest = semtypes.NEVER
		}
		semType := d.DefineMappingTypeWrapped(t.typeEnv(), fields, rest)
		mat := semtypes.ToMappingAtomicType(t.typeContext(), semType)
		t.setMappingAtomBType(mat, ty)
		return semType, true
	case *ast.BLangFunctionType:
		if ty.IsAnyFunction() {
			return semtypes.FUNCTION, true
		}
		if ty.Definition != nil {
			return ty.Definition.GetSemType(t.typeEnv()), true
		}
		fd := semtypes.NewFunctionDefinition()
		ty.Definition = &fd
		paramTypes := make([]semtypes.SemType, len(ty.RequiredParams))
		for i := range ty.RequiredParams {
			paramTy, ok := resolveBType(t, ty.RequiredParams[i].TypeDesc, depth+1)
			if !ok {
				return semtypes.SemType{}, false
			}
			paramTypes[i] = paramTy
			ty.RequiredParams[i].SetDeterminedType(paramTy)
			if ty.RequiredParams[i].Name != nil {
				ty.RequiredParams[i].Name.SetDeterminedType(semtypes.NEVER)
			}
		}
		restTy := semtypes.NEVER
		if ty.RestParam != nil {
			restParamTy, ok := resolveBType(t, ty.RestParam.TypeDesc, depth+1)
			if !ok {
				return semtypes.SemType{}, false
			}
			restTy = restParamTy
			ty.RestParam.SetDeterminedType(restParamTy)
		}
		paramListDefn := semtypes.NewListDefinition()
		paramListTy := paramListDefn.DefineListTypeWrapped(t.typeEnv(), paramTypes, len(paramTypes), restTy, semtypes.CellMutability_CELL_MUT_NONE)
		var returnTy semtypes.SemType
		if ty.ReturnTypeDescriptor != nil {
			var ok bool
			returnTy, ok = resolveBType(t, ty.ReturnTypeDescriptor, depth+1)
			if !ok {
				return semtypes.SemType{}, false
			}
		} else {
			returnTy = semtypes.NIL
		}
		isolated := ty.IsIsolated()
		transactional := ty.IsTransactional()
		fnType := fd.Define(t.typeEnv(), paramListTy, returnTy,
			semtypes.FunctionQualifiersFrom(t.typeEnv(), isolated, transactional))
		return fnType, true
	case *ast.BLangObjectType:
		return resolveObjectType(t, ty, depth)
	default:
		t.unimplemented("unsupported type", diagnostics.Location{})
		return semtypes.SemType{}, false
	}
}

func resolveObjectType(t typeResolver, ty *ast.BLangObjectType, depth int) (semtypes.SemType, bool) {
	defn := ty.Definition
	if defn != nil {
		return defn.GetSemType(t.typeEnv()), true
	}
	od := semtypes.NewObjectDefinition()
	ty.Definition = &od
	// Step 1: Accumulate included members from symbols
	includedMembers := make(map[string][]semtypes.Member)
	incMembers, err := collectIncludedMembers(t, ty.Inclusions, depth)
	if err {
		t.semanticError("error resolving type inclusion", ty.GetPosition())
		return semtypes.SemType{}, false
	}
	for _, m := range incMembers {
		if m.MemberKind() == model.InclusionMemberKindRestType {
			t.internalError("unexpected rest inclusion", ty.GetPosition())
		}
		member := inclusionMemberToSemtypeMember(m)
		includedMembers[member.Name] = append(includedMembers[member.Name], member)
	}

	// Step 2: Build direct members and validate overrides
	var directMembers []directMember
	for m := range ty.Members() {
		if m.MemberKind() == ast.ObjectMemberKindRemoteMethod {
			if ty.NetworkQuals != ast.ObjectNetworkQualsClient && ty.NetworkQuals != ast.ObjectNetworkQualsService {
				t.semanticError("remote methods are only allowed in client or service object types", ty.GetPosition())
				return semtypes.SemType{}, false
			}
		}
		valueTy, ok := resolveObjectMemberType(t, m, depth)
		if !ok {
			return semtypes.SemType{}, false
		}
		directMembers = append(directMembers, directMember{
			name:       m.Name(),
			valueTy:    valueTy,
			kind:       semtypeMemberKind(m.MemberKind()),
			visibility: semtypeVisibility(m.IsPublic()),
			immutable:  m.MemberKind() != ast.ObjectMemberKindField,
			pos:        ty.GetPosition(),
		})
	}

	members, ok := validateOverridesAndMerge(t, directMembers, includedMembers, ty.GetPosition(), true)
	if !ok {
		return semtypes.SemType{}, false
	}

	// Step 3: Create semtype
	networkQual := semtypeNetworkQualifier(ty.NetworkQuals)
	qualifiers := semtypes.ObjectQualifiersFrom(ty.Isolated, false, networkQual)
	semType := od.Define(t.typeEnv(), qualifiers, members)
	return semType, true
}

// directMember represents a member declared directly on a type (not inherited via inclusion).
type directMember struct {
	name       string
	valueTy    semtypes.SemType
	kind       semtypes.MemberKind
	visibility semtypes.Visibility
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
				if !semtypes.IsSubtype(t.typeContext(), dm.valueTy, incMember.ValueTy) {
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
			ValueTy:    dm.valueTy,
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

type recordInclusionResolutionResult struct {
	includedFields map[string][]model.FieldDescriptor
	includedRestTy semtypes.SemType
	multpleRestTy  bool
}

func resolveRecordInclusions(t typeResolver, recordTy *ast.BLangRecordType, depth int) (recordInclusionResolutionResult, bool) {
	// Resolve UDT nodes to set their DeterminedType
	for _, inc := range recordTy.TypeInclusions {
		if _, ok := resolveBType(t, inc, 0); !ok {
			return recordInclusionResolutionResult{}, false
		}
	}

	incMembers, err := collectIncludedMembers(t, recordTy.Inclusions, depth)
	if err {
		return recordInclusionResolutionResult{}, false
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
	return recordInclusionResolutionResult{includedFields, includedRest, needsRestOverride}, true
}

func resolveConstant(t typeResolver, constant *ast.BLangConstant) bool {
	if !semtypes.IsZero(t.symbolType(constant.Symbol())) {
		return true
	}
	if constant.Expr == nil {
		t.internalError("constant expression is nil", constant.GetPosition())
		return false
	}
	if constant.Name != nil {
		setOtherNodesAsNever(constant.Name)
	}

	var annotationType semtypes.SemType
	if typeNode := constant.TypeNode(); typeNode != nil {
		var ok bool
		annotationType, ok = resolveBType(t, typeNode, 0)
		if !ok {
			return false
		}
	}

	expr, ok := constant.Expr.(ast.BLangExpression)
	if !ok {
		t.internalError("constant expression is not an expression", constant.GetPosition())
		return false
	}
	exprTy, _, ok := resolveActionOrExpression(t, nil, expr, annotationType)
	if !ok {
		return false
	}
	value, err := evaluateConstantExpression(t, expr)
	if err != nil {
		// A const-expr is evaluated at compile time (spec §6.4). A genuine
		// evaluation failure — e.g. a cast that cannot be performed such as
		// <int>(1.0/0.0) — is therefore a compile-time error, not a deferred
		// runtime panic. Structural non-constness surfaces as
		// errNotConstantExpression and is reported by validateConstantExpr.
		if !errors.Is(err, errNotConstantExpression) {
			t.semanticError("expression is not a constant expression", expr.GetPosition())
		}
	} else if sym, ok := t.getSymbol(constant.Symbol()).(*model.ConstantValueSymbol); ok {
		sym.SetConstantValue(value)
	}

	// TODO: I am not sure if this is strictly correct given expression type would have changed based on the contextually expected type in things like structure constructor expressions.
	expectedType := exprTy
	setExpectedType(constant, expectedType)
	symbol := constant.Symbol()
	t.setSymbolType(symbol, expectedType)

	return true
}

func resolveMatchStatement(t typeResolver, chain *binding, stmt *ast.BLangMatchStatement) (statementEffect, bool) {
	_, exprEffect, ok := resolveActionOrExpression(t, chain, stmt.Expr, semtypes.SemType{})
	if !ok {
		return defaultStmtEffect(chain), false
	}
	chain = exprEffect.ifTrue

	exprRef, isVarRef := varRefExp(chain, stmt.Expr)
	var remainingType semtypes.SemType
	if isVarRef {
		remainingType = t.symbolType(exprRef)
	} else {
		remainingType = stmt.Expr.GetDeterminedType()
	}
	allNonCompletion := true
	var bodyEffects []statementEffect

	tyCtx := semtypes.ContextFrom(t.typeEnv())

	for i := range stmt.MatchClauses {
		clause := &stmt.MatchClauses[i]

		if semtypes.IsEmpty(tyCtx, remainingType) {
			t.semanticError("unreachable match clause", clause.GetPosition())
		}

		var bodyChain *binding
		var ok bool
		clause.AcceptedType, bodyChain, ok = matchClauseAcceptedType(t, chain, clause, remainingType)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		clauseAcceptedType := semtypes.Intersect(remainingType, clause.AcceptedType)

		clauseIsEmpty := semtypes.IsEmpty(tyCtx, clauseAcceptedType)
		if clauseIsEmpty {
			t.semanticError("unmatchable match clause", clause.GetPosition())
		}

		clause.AcceptedType = clauseAcceptedType

		if clauseIsEmpty {
			_, ok := resolveMatchClause(t, bodyChain, clause)
			if !ok {
				return defaultStmtEffect(chain), false
			}
			continue
		}

		if isVarRef {
			baseRef := t.unnarrowedSymbol(exprRef)
			narrowedSym := narrowSymbol(t, baseRef, clauseAcceptedType)
			bodyChain = &binding{
				ref:            baseRef,
				narrowedSymbol: narrowedSym,
				prev:           bodyChain,
			}
		}

		bodyEffect, ok := resolveMatchClause(t, bodyChain, clause)
		if !ok {
			return defaultStmtEffect(chain), false
		}
		bodyEffects = append(bodyEffects, bodyEffect)
		if !bodyEffect.nonCompletion {
			allNonCompletion = false
		}

		remainingType = semtypes.Diff(remainingType, clause.AcceptedType)
	}

	stmt.IsExhaustive = semtypes.IsEmpty(tyCtx, remainingType)

	if stmt.IsExhaustive && allNonCompletion {
		return statementEffect{chain, true}, true
	}

	var result *binding
	first := true
	for _, effect := range bodyEffects {
		if effect.nonCompletion {
			continue
		}
		if first {
			result = effect.binding
			first = false
		} else {
			result = mergeChains(t, result, effect.binding, semtypes.Union)
		}
	}
	return statementEffect{result, false}, true
}

func matchClauseAcceptedType(t typeResolver, chain *binding, clause *ast.BLangMatchClause, remainingType semtypes.SemType) (semtypes.SemType, *binding, bool) {
	tyCtx := semtypes.ContextFrom(t.typeEnv())
	acceptedTy := semtypes.NEVER
	patternRemaining := remainingType
	for i, pattern := range clause.Patterns {
		patternTy, ok := resolveMatchPattern(t, chain, pattern, remainingType)
		if !ok {
			return semtypes.SemType{}, nil, false
		}
		if i > 0 && semtypes.IsEmpty(tyCtx, semtypes.Intersect(patternTy, patternRemaining)) {
			t.semanticError("unmatchable match pattern", pattern.GetPosition())
		}
		patternRemaining = semtypes.Diff(patternRemaining, patternTy)
		acceptedTy = semtypes.Union(acceptedTy, patternTy)
	}
	if clause.Guard != nil {
		_, guardEffect, ok := resolveActionOrExpression(t, chain, clause.Guard, remainingType)
		if !ok {
			return semtypes.SemType{}, nil, false
		}
		return acceptedTy, guardEffect.ifTrue, true
	}
	return acceptedTy, chain, true
}

func resolveObjectMemberType(t typeResolver, m ast.ObjectMember, depth int) (semtypes.SemType, bool) {
	switch m := m.(type) {
	case *ast.BObjectField:
		valueTy, ok := resolveBType(t, m.Ty, depth+1)
		if ok {
			m.SetDeterminedType(valueTy)
		}
		return valueTy, ok
	case *ast.BMethodDecl:
		valueTy, ok := resolveBType(t, &m.BLangFunctionType, depth+1)
		if ok {
			m.SetDeterminedType(valueTy)
		}
		return valueTy, ok
	default:
		return semtypes.SemType{}, false
	}
}

func resolveMatchClause(t typeResolver, chain *binding, clause *ast.BLangMatchClause) (statementEffect, bool) {
	bodyEffect, ok := resolveBlockStatements(t, chain, clause.Body.Stmts)
	if !ok {
		return defaultStmtEffect(chain), false
	}
	clause.Body.SetDeterminedType(semtypes.NEVER)
	clause.SetDeterminedType(semtypes.NEVER)
	return bodyEffect, true
}

func isValidConstPatternExpr(t typeResolver, expr ast.BLangExpression) bool {
	var ref model.SymbolRef
	switch e := expr.(type) {
	case *ast.BLangSimpleVarRef:
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
		ty, _, ok := resolveActionOrExpression(t, chain, p.Expr, expectedTy)
		if !ok {
			return semtypes.SemType{}, false
		}
		if !isValidConstPatternExpr(t, p.Expr) {
			t.semanticError("match pattern variable reference must refer to a constant", p.Expr.GetPosition())
			return semtypes.SemType{}, false
		}
		p.SetAcceptedType(ty)
		p.SetDeterminedType(semtypes.NEVER)
		return ty, true
	case *ast.BLangWildCardMatchPattern:
		ty := semtypes.ANY
		p.SetAcceptedType(ty)
		p.SetDeterminedType(semtypes.NEVER)
		return ty, true
	default:
		t.internalError(fmt.Sprintf("unexpected match pattern type: %T", pattern), pattern.GetPosition())
		return semtypes.NEVER, false
	}
}

func semtypeMemberKind(kind ast.ObjectMemberKind) semtypes.MemberKind {
	switch kind {
	case ast.ObjectMemberKindField:
		return semtypes.MemberKindField
	case ast.ObjectMemberKindMethod:
		return semtypes.MemberKindMethod
	case ast.ObjectMemberKindRemoteMethod:
		return semtypes.MemberKindRemoteMethod
	case ast.ObjectMemberKindResourceMethod:
		return semtypes.MemberKindResourceMethod
	default:
		panic("invalid member kind")
	}
}

func inclusionMemberKindToSemtype(kind model.InclusionMemberKind) semtypes.MemberKind {
	switch kind {
	case model.InclusionMemberKindField:
		return semtypes.MemberKindField
	case model.InclusionMemberKindMethod:
		return semtypes.MemberKindMethod
	case model.InclusionMemberKindRemoteMethod:
		return semtypes.MemberKindRemoteMethod
	case model.InclusionMemberKindResourceMethod:
		return semtypes.MemberKindResourceMethod
	default:
		panic("invalid inclusion member kind")
	}
}

func inclusionMemberToSemtypeMember(m model.InclusionMember) semtypes.Member {
	kind := m.MemberKind()
	vis := semtypes.VisibilityPrivate
	if fd, ok := m.(*model.FieldDescriptor); ok {
		vis = semtypeVisibility(fd.IsPublic())
	} else if md, ok := m.(*model.MethodDescriptor); ok {
		vis = semtypeVisibility(md.IsPublic())
	}
	return semtypes.Member{
		Name:       m.MemberName(),
		ValueTy:    m.MemberType(),
		Kind:       inclusionMemberKindToSemtype(kind),
		Visibility: vis,
		Immutable:  kind != model.InclusionMemberKindField,
	}
}

func semtypeVisibility(isPublic bool) semtypes.Visibility {
	if isPublic {
		return semtypes.VisibilityPublic
	}
	return semtypes.VisibilityPrivate
}

func semtypeNetworkQualifier(nq ast.ObjectNetworkQuals) semtypes.NetworkQualifier {
	switch nq {
	case ast.ObjectNetworkQualsNone:
		return semtypes.NetworkQualifierNone
	case ast.ObjectNetworkQualsClient:
		return semtypes.NetworkQualifierClient
	case ast.ObjectNetworkQualsService:
		return semtypes.NetworkQualifierService
	default:
		panic("invalid network qualifier")
	}
}

func setPositions(pos diagnostics.Location, nodes ...ast.BLangNode) {
	for _, node := range nodes {
		node.SetPosition(pos)
	}
}

// opaqueFnMonomorphizer monomorphizes a generic lang-lib function at a call
// site. It resolves only the first (container) argument, builds the concrete
// monomorphized symbol, adds it to the opaque symbol's own space, and returns
// its ref. Results are cached on the opaque symbol.
type opaqueFnMonomorphizer func(t typeResolver, sym *model.OpaqueFunctionSymbol, polymorphicRef model.SymbolRef, chain *binding, args []ast.BLangExpression, pos diagnostics.Location) (model.SymbolRef, bool)

// Per-package opaque-function monomorphizer tables, indexed by opaque id.
// Assigned in init (not via var initializers) to avoid an initialization cycle:
// the monomorphizers' bodies reach back into the resolver call graph, which
// references these tables.
var (
	arrayOpaqueMonomorphizers []opaqueFnMonomorphizer
	mapOpaqueMonomorphizers   []opaqueFnMonomorphizer
	xmlOpaqueMonomorphizers   []opaqueFnMonomorphizer
)

func init() {
	arrayOpaqueMonomorphizers = []opaqueFnMonomorphizer{
		model.OpaqueFnArrayPush:    monomorphizeArrayPush,
		model.OpaqueFnArrayIndexOf: monomorphizeArrayIndexOf,
		model.OpaqueFnArrayRemove:  monomorphizeArrayRemove,
	}
	mapOpaqueMonomorphizers = []opaqueFnMonomorphizer{
		model.OpaqueFnMapRemove: monomorphizeMapRemove,
	}
	xmlOpaqueMonomorphizers = []opaqueFnMonomorphizer{
		model.OpaqueFnXMLIterator: monomorphizeXMLIterator,
	}
}

// opaqueFunctionMonomorphizerFor selects the monomorphizer for a generic
// lang-lib function, indexed by its opaque id within the owning package.
func opaqueFunctionMonomorphizerFor(org, pkg string, id int) (opaqueFnMonomorphizer, bool) {
	if org != "ballerina" {
		return nil, false
	}
	var monomorphizers []opaqueFnMonomorphizer
	switch pkg {
	case "lang.array":
		monomorphizers = arrayOpaqueMonomorphizers
	case "lang.map":
		monomorphizers = mapOpaqueMonomorphizers
	case "lang.xml":
		monomorphizers = xmlOpaqueMonomorphizers
	default:
		return nil, false
	}
	if id < 0 || id >= len(monomorphizers) {
		return nil, false
	}
	return monomorphizers[id], true
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

// containerArgExpr returns the expression bound to the container (first)
// parameter of an opaque lang-lib function. The container is always the first
// positional argument (Ballerina forbids a named argument before a positional
// one); when the call uses only named arguments it is matched by paramName.
// This keeps out-of-order named calls (e.g. map:remove(k = "x", m = myMap))
// from being monomorphized against the wrong argument.
func containerArgExpr(args []ast.BLangExpression, paramName string) (ast.BLangExpression, bool) {
	for _, arg := range args {
		named, ok := arg.(*ast.BLangNamedArgsExpression)
		if !ok {
			return arg, true
		}
		if named.Name.Value == paramName {
			return named.Expr, true
		}
	}
	return nil, false
}

// storeMonomorphizedOpaqueFn builds the monomorphic symbol for sig, adds it to
// the opaque symbol's space, sets its type, and caches it under containerTy.
func storeMonomorphizedOpaqueFn(t typeResolver, sym *model.OpaqueFunctionSymbol, polymorphicRef model.SymbolRef, sig model.FunctionSignature, containerTy semtypes.SemType) model.SymbolRef {
	mono := &monomorphicOpaqueFn{FunctionSymbol: model.NewFunctionSymbol(sym.Name(), sig, true), poly: polymorphicRef}
	mono.SetType(typeFromFunctionSignature(t, sig))
	space := sym.SymbolSpace
	idx := space.AppendSymbol(mono)
	mono.name = fmt.Sprintf("%s$mono$%d", sym.Name(), idx)
	ref := space.RefAt(idx)
	if sym.Store != nil {
		sym.Store(ref, containerTy)
	}
	return ref
}

func monomorphizeArrayPush(t typeResolver, sym *model.OpaqueFunctionSymbol, polymorphicRef model.SymbolRef, chain *binding, args []ast.BLangExpression, pos diagnostics.Location) (model.SymbolRef, bool) {
	containerExpr, ok := containerArgExpr(args, "arr")
	if !ok {
		t.semanticError("missing container argument", pos)
		return model.SymbolRef{}, false
	}
	containerTy, _, ok := resolveActionOrExpression(t, chain, containerExpr, semtypes.SemType{})
	if !ok {
		return model.SymbolRef{}, false
	}
	if sym.Lookup != nil {
		if ref, ok := sym.Lookup(containerTy); ok {
			return ref, true
		}
	}
	cx := t.typeContext()
	if !semtypes.IsSubtype(cx, containerTy, semtypes.LIST) {
		t.semanticError("expect first argument to be a subtype of (any|error)[]", pos)
		return model.SymbolRef{}, false
	}
	valType := semtypes.ListProj(cx, containerTy, semtypes.INT)
	sig := model.FunctionSignature{
		ParamTypes:    []semtypes.SemType{containerTy},
		RestParamType: valType,
		ReturnType:    semtypes.NIL,
		Flags:         model.FuncSymbolFlagIsolated,
	}
	return storeMonomorphizedOpaqueFn(t, sym, polymorphicRef, sig, containerTy), true
}

func monomorphizeArrayIndexOf(t typeResolver, sym *model.OpaqueFunctionSymbol, polymorphicRef model.SymbolRef, chain *binding, args []ast.BLangExpression, pos diagnostics.Location) (model.SymbolRef, bool) {
	containerExpr, ok := containerArgExpr(args, "arr")
	if !ok {
		t.semanticError("missing container argument", pos)
		return model.SymbolRef{}, false
	}
	containerTy, _, ok := resolveActionOrExpression(t, chain, containerExpr, semtypes.SemType{})
	if !ok {
		return model.SymbolRef{}, false
	}
	if sym.Lookup != nil {
		if ref, ok := sym.Lookup(containerTy); ok {
			return ref, true
		}
	}
	cx := t.typeContext()
	if !semtypes.IsSubtype(cx, containerTy, semtypes.LIST) {
		t.semanticError("expect first argument to be a subtype of (any|error)[]", pos)
		return model.SymbolRef{}, false
	}
	valType := semtypes.ListProj(cx, containerTy, semtypes.INT)
	sig := model.FunctionSignature{
		ParamTypes: []semtypes.SemType{containerTy, valType, semtypes.INT},
		ReturnType: semtypes.Union(semtypes.INT, semtypes.NIL),
		Flags:      model.FuncSymbolFlagIsolated,
	}
	return storeMonomorphizedOpaqueFn(t, sym, polymorphicRef, sig, containerTy), true
}

func monomorphizeArrayRemove(t typeResolver, sym *model.OpaqueFunctionSymbol, polymorphicRef model.SymbolRef, chain *binding, args []ast.BLangExpression, pos diagnostics.Location) (model.SymbolRef, bool) {
	containerExpr, ok := containerArgExpr(args, "arr")
	if !ok {
		t.semanticError("missing container argument", pos)
		return model.SymbolRef{}, false
	}
	containerTy, _, ok := resolveActionOrExpression(t, chain, containerExpr, semtypes.SemType{})
	if !ok {
		return model.SymbolRef{}, false
	}
	if sym.Lookup != nil {
		if ref, ok := sym.Lookup(containerTy); ok {
			return ref, true
		}
	}
	cx := t.typeContext()
	if !semtypes.IsSubtype(cx, containerTy, semtypes.LIST) {
		t.semanticError("expect first argument to be a subtype of (any|error)[]", pos)
		return model.SymbolRef{}, false
	}
	elementType := semtypes.ListProj(cx, containerTy, semtypes.INT)
	sig := model.FunctionSignature{
		ParamTypes: []semtypes.SemType{containerTy, semtypes.INT},
		ReturnType: elementType,
		Flags:      model.FuncSymbolFlagIsolated,
	}
	return storeMonomorphizedOpaqueFn(t, sym, polymorphicRef, sig, containerTy), true
}

func monomorphizeXMLIterator(t typeResolver, sym *model.OpaqueFunctionSymbol, polymorphicRef model.SymbolRef, chain *binding, args []ast.BLangExpression, pos diagnostics.Location) (model.SymbolRef, bool) {
	containerExpr, ok := containerArgExpr(args, "x")
	if !ok {
		t.semanticError("missing container argument", pos)
		return model.SymbolRef{}, false
	}
	containerTy, _, ok := resolveActionOrExpression(t, chain, containerExpr, semtypes.SemType{})
	if !ok {
		return model.SymbolRef{}, false
	}
	if sym.Lookup != nil {
		if ref, ok := sym.Lookup(containerTy); ok {
			return ref, true
		}
	}
	cx := t.typeContext()
	if !semtypes.IsSubtype(cx, containerTy, semtypes.XML) {
		t.semanticError("expect first argument to be a subtype of xml", pos)
		return model.SymbolRef{}, false
	}
	itemTy := semtypes.XMLItemType(containerTy)
	sig := model.FunctionSignature{
		ParamTypes:    []semtypes.SemType{containerTy},
		RestParamType: semtypes.NEVER,
		ReturnType:    createXMLIteratorType(t, itemTy),
		Flags:         model.FuncSymbolFlagIsolated,
	}
	return storeMonomorphizedOpaqueFn(t, sym, polymorphicRef, sig, containerTy), true
}

func createXMLIteratorType(t typeResolver, itemTy semtypes.SemType) semtypes.SemType {
	env := t.typeEnv()
	return t.xmlIteratorTypeCache().GetOrBuild(itemTy, func() semtypes.SemType {
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
	})
}

func monomorphizeMapRemove(t typeResolver, sym *model.OpaqueFunctionSymbol, polymorphicRef model.SymbolRef, chain *binding, args []ast.BLangExpression, pos diagnostics.Location) (model.SymbolRef, bool) {
	containerExpr, ok := containerArgExpr(args, "m")
	if !ok {
		t.semanticError("missing container argument", pos)
		return model.SymbolRef{}, false
	}
	containerTy, _, ok := resolveActionOrExpression(t, chain, containerExpr, semtypes.SemType{})
	if !ok {
		return model.SymbolRef{}, false
	}
	if sym.Lookup != nil {
		if ref, ok := sym.Lookup(containerTy); ok {
			return ref, true
		}
	}
	cx := t.typeContext()
	if !semtypes.IsSubtype(cx, containerTy, semtypes.MAPPING) {
		t.semanticError("expect first argument to be a subtype of map<any|error>", pos)
		return model.SymbolRef{}, false
	}
	memberType := semtypes.MappingMemberTypeInnerValProj(cx, containerTy, semtypes.STRING)
	sig := model.FunctionSignature{
		ParamTypes:    []semtypes.SemType{containerTy, semtypes.STRING},
		RestParamType: semtypes.NEVER,
		ReturnType:    memberType,
		Flags:         model.FuncSymbolFlagIsolated,
	}
	return storeMonomorphizedOpaqueFn(t, sym, polymorphicRef, sig, containerTy), true
}
