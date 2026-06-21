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

package context

import (
	"strconv"
	"sync"

	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/values"
)

type distinctTypeTracker struct {
	mu     sync.Mutex
	ids    map[model.SymbolRef]int
	refs   map[int]model.SymbolRef
	nextID int
}

type langLibDistinctTypeKey struct {
	packageName string
	typeName    string
}

type langLibDistinctTypeRegistry struct {
	mu      sync.RWMutex
	symbols map[langLibDistinctTypeKey]model.SymbolRef
}

func newLangLibDistinctTypeRegistry() langLibDistinctTypeRegistry {
	return langLibDistinctTypeRegistry{symbols: make(map[langLibDistinctTypeKey]model.SymbolRef)}
}

func (r *langLibDistinctTypeRegistry) register(packageName, typeName string, ref model.SymbolRef) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := langLibDistinctTypeKey{packageName: packageName, typeName: typeName}
	if existing, ok := r.symbols[key]; ok {
		return existing == ref
	}
	r.symbols[key] = ref
	return true
}

func (r *langLibDistinctTypeRegistry) lookup(packageName, typeName string) (model.SymbolRef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ref, ok := r.symbols[langLibDistinctTypeKey{packageName: packageName, typeName: typeName}]
	return ref, ok
}

func newDistinctTypeTracker() distinctTypeTracker {
	return distinctTypeTracker{
		ids:  make(map[model.SymbolRef]int),
		refs: make(map[int]model.SymbolRef),
	}
}

func (t *distinctTypeTracker) id(symbol model.SymbolRef) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	if id, ok := t.ids[symbol]; ok {
		return id
	}
	id := t.nextID
	t.nextID++
	t.ids[symbol] = id
	t.refs[id] = symbol
	return id
}

func (t *distinctTypeTracker) symbolRef(id int) (model.SymbolRef, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ref, ok := t.refs[id]
	return ref, ok
}

// CompilerEnvironment maintain the shared state of the frontend.
type CompilerEnvironment struct {
	anonTypeCount              map[*model.PackageID]int
	anonFuncCount              map[*model.PackageID]int
	packageInterner            *model.PackageIDInterner
	symbolSpaces               []*model.SymbolSpace
	symbolSpacesMu             sync.RWMutex // we need this because desugaring add new init functions concurrently we shouldn't need this if the spaces are scoped to the module, may be we should do that?
	typeEnv                    semtypes.Env
	underlyingSymbol           sync.Map
	distinctTypes              distinctTypeTracker
	langLibDistinctTypeSymbols langLibDistinctTypeRegistry
	// symbolAnnotations holds annotation values keyed by symbol ref, instead of
	// a field on each symbol — most symbols carry no annotations, so this avoids
	// a per-symbol word and keeps lookup keyed by reference. Values are written
	// single-threaded during top-level resolution and read concurrently later.
	symbolAnnotations sync.Map // model.SymbolRef -> values.AnnotationValues
	statsEnabled      bool
	diagnosticContext *diagnostics.DiagnosticEnv
}

// SetSymbolAnnotationValue records an annotation value for the given symbol.
func (c *CompilerEnvironment) SetSymbolAnnotationValue(symbol model.SymbolRef, key string, value values.AnnotationValue) {
	actual, _ := c.symbolAnnotations.LoadOrStore(symbol, values.NewAnnotationValues())
	actual.(values.AnnotationValues)[key] = value
}

// SymbolAnnotationValues returns the annotation values for the given symbol, or
// an empty set if it has none. Callers should treat the returned map as
// read-only compiler metadata.
func (c *CompilerEnvironment) SymbolAnnotationValues(symbol model.SymbolRef) values.AnnotationValues {
	if av, ok := c.symbolAnnotations.Load(symbol); ok {
		return av.(values.AnnotationValues)
	}
	return values.NewAnnotationValues()
}

func (c *CompilerEnvironment) DiagnosticEnv() *diagnostics.DiagnosticEnv {
	return c.diagnosticContext
}

func (c *CompilerEnvironment) NewSymbolSpace(packageID model.PackageID) *model.SymbolSpace {
	c.symbolSpacesMu.Lock()
	space := model.NewSymbolSpaceInner(packageID, len(c.symbolSpaces))
	c.symbolSpaces = append(c.symbolSpaces, space)
	c.symbolSpacesMu.Unlock()
	return space
}

func (c *CompilerEnvironment) NewModuleScope(pkg model.PackageID, prefixes map[string]model.ExportedSymbolSpace) *model.ModuleScope {
	if prefixes == nil {
		prefixes = make(map[string]model.ExportedSymbolSpace)
	}
	return &model.ModuleScope{
		Main:       c.NewSymbolSpace(pkg),
		Prefix:     prefixes,
		Annotation: c.NewSymbolSpace(pkg),
	}
}

func (c *CompilerEnvironment) NewFunctionScope(parent model.Scope, pkg model.PackageID) *model.FunctionScope {
	return &model.FunctionScope{
		BlockScopeBase: model.BlockScopeBase{
			Parent: parent,
			Main:   c.NewSymbolSpace(pkg),
		},
	}
}

func (c *CompilerEnvironment) NewBlockScope(parent model.Scope, pkg model.PackageID) *model.BlockScope {
	return &model.BlockScope{
		BlockScopeBase: model.BlockScopeBase{
			Parent: parent,
			Main:   c.NewSymbolSpace(pkg),
		},
	}
}

func (c *CompilerEnvironment) GetSymbol(symbol model.SymbolRef) model.Symbol {
	return c.symbolSpace(symbol.SpaceIndex).SymbolAt(symbol.Index)
}

func (c *CompilerEnvironment) symbolSpace(index int) *model.SymbolSpace {
	c.symbolSpacesMu.RLock()
	defer c.symbolSpacesMu.RUnlock()
	// We treat 0,0 as empty space
	return c.symbolSpaces[index-1]
}

func (c *CompilerEnvironment) SymbolPackage(symbol model.SymbolRef) model.PackageIdentifier {
	if symbol.IsEmpty() {
		panic("cannot get package for empty symbol ref")
	}
	return c.symbolSpace(symbol.SpaceIndex).Pkg
}

// FindSymbol is used for symbol serialization, and is not meant to be used by other parts. This lookup is
// potentially very slow and you should have a symbolRef in the AST for anywhere that you may need a symbol
func (c *CompilerEnvironment) FindSymbol(pkg model.PackageIdentifier, name string) (model.SymbolRef, bool) {
	c.symbolSpacesMu.RLock()
	spaces := append([]*model.SymbolSpace(nil), c.symbolSpaces...)
	c.symbolSpacesMu.RUnlock()
	for _, space := range spaces {
		if space.Pkg != pkg {
			continue
		}
		if ref, ok := space.GetSymbol(name); ok {
			return ref, true
		}
	}
	return model.SymbolRef{}, false
}

func (c *CompilerEnvironment) AddSymbolToSameSpace(ref model.SymbolRef, name string, symbol model.Symbol) model.SymbolRef {
	space := c.symbolSpace(ref.SpaceIndex)
	space.AddSymbol(name, symbol)
	newRef, _ := space.GetSymbol(name)
	return newRef
}

// CreateNarrowedSymbol create a narrowed symbol for the given baseRef symbol. IMPORTANT: baseRef must be the actual symbol
// not a narrowed symbol.
func (c *CompilerEnvironment) CreateNarrowedSymbol(baseRef model.SymbolRef) model.SymbolRef {
	symbolSpace := c.symbolSpace(baseRef.SpaceIndex)
	underlyingSymbolCopy := c.GetSymbol(baseRef).Copy()
	symbolIndex := symbolSpace.AppendSymbol(underlyingSymbolCopy)
	narrowedSymbol := model.SymbolRef{
		SpaceIndex: baseRef.SpaceIndex,
		Index:      symbolIndex,
	}
	c.underlyingSymbol.Store(narrowedSymbol, baseRef)
	return narrowedSymbol
}

func (c *CompilerEnvironment) CreateFunctionSymbol(space *model.SymbolSpace, name string, signature model.FunctionSignature, fnTy semtypes.SemType) model.SymbolRef {
	sym := model.NewFunctionSymbol(name, signature, false)
	sym.SetType(fnTy)
	symbolIndex := space.AppendSymbol(sym)
	return space.RefAt(symbolIndex)
}

func (c *CompilerEnvironment) UnnarrowedSymbol(symbol model.SymbolRef) model.SymbolRef {
	if underlying, ok := c.underlyingSymbol.Load(symbol); ok {
		return underlying.(model.SymbolRef)
	}
	return symbol
}

func (c *CompilerEnvironment) SymbolName(symbol model.SymbolRef) string {
	return c.GetSymbol(symbol).Name()
}

func (c *CompilerEnvironment) SymbolType(symbol model.SymbolRef) semtypes.SemType {
	return c.GetSymbol(symbol).Type()
}

func (c *CompilerEnvironment) SymbolLocation(symbol model.SymbolRef) diagnostics.Location {
	return c.GetSymbol(symbol).Location()
}

func (c *CompilerEnvironment) SetSymbolLocation(symbol model.SymbolRef, location diagnostics.Location) {
	c.GetSymbol(symbol).SetLocation(location)
}

func (c *CompilerEnvironment) SymbolKind(symbol model.SymbolRef) model.SymbolKind {
	return c.GetSymbol(symbol).Kind()
}

func (c *CompilerEnvironment) SymbolIsPublic(symbol model.SymbolRef) bool {
	return c.GetSymbol(symbol).IsPublic()
}

func (c *CompilerEnvironment) SymbolIsClass(symbol model.SymbolRef) bool {
	_, ok := c.GetSymbol(symbol).(model.ClassSymbol)
	return ok
}

type ValueSymbolMetadata struct {
	Parameter    bool
	Final        bool
	Const        bool
	Configurable bool
	Isolated     bool
}

func (c *CompilerEnvironment) ValueSymbolMetadata(symbol model.SymbolRef) (ValueSymbolMetadata, bool) {
	valueSymbol, ok := c.GetSymbol(symbol).(model.ValueSymbol)
	if !ok {
		return ValueSymbolMetadata{}, false
	}
	return ValueSymbolMetadata{
		Parameter:    valueSymbol.IsParameter(),
		Final:        valueSymbol.IsFinal(),
		Const:        valueSymbol.IsConst(),
		Configurable: valueSymbol.IsConfigurable(),
		Isolated:     valueSymbol.IsIsolated(),
	}, true
}

func (c *CompilerEnvironment) SetSymbolType(symbol model.SymbolRef, ty semtypes.SemType) {
	c.GetSymbol(symbol).SetType(ty)
}

func (c *CompilerEnvironment) DistinctTypeID(symbol model.SymbolRef) int {
	return c.distinctTypes.id(symbol)
}

func (c *CompilerEnvironment) DistinctTypeSymbolRef(id int) (model.SymbolRef, bool) {
	return c.distinctTypes.symbolRef(id)
}

func (c *CompilerEnvironment) RegisterLangLibDistinctTypeSymbol(packageName, typeName string, ref model.SymbolRef) bool {
	return c.langLibDistinctTypeSymbols.register(packageName, typeName, ref)
}

func (c *CompilerEnvironment) LangLibDistinctTypeSymbol(packageName, typeName string) (model.SymbolRef, bool) {
	return c.langLibDistinctTypeSymbols.lookup(packageName, typeName)
}

func (c *CompilerEnvironment) GetDefaultPackage() *model.PackageID {
	return c.packageInterner.GetDefaultPackage()
}

func (c *CompilerEnvironment) NewPackageID(orgName model.Name, nameComps []model.Name, version model.Name) *model.PackageID {
	return model.NewPackageID(c.packageInterner, orgName, nameComps, version)
}

func NewCompilerEnvironment(typeEnv semtypes.Env, statsEnabled bool) *CompilerEnvironment {
	return &CompilerEnvironment{
		anonTypeCount:              make(map[*model.PackageID]int),
		anonFuncCount:              make(map[*model.PackageID]int),
		packageInterner:            model.DefaultPackageIDInterner,
		distinctTypes:              newDistinctTypeTracker(),
		langLibDistinctTypeSymbols: newLangLibDistinctTypeRegistry(),
		typeEnv:                    typeEnv,
		statsEnabled:               statsEnabled,
		diagnosticContext:          diagnostics.NewDiagnosticEnv(),
	}
}

// GetTypeEnv returns the type environment for this context
func (c *CompilerEnvironment) GetTypeEnv() semtypes.Env {
	return c.typeEnv
}

const (
	anonPrefix      = "$anon"
	builtinAnonType = anonPrefix + "Type$builtin$"
	anonType        = anonPrefix + "Type$"
)

func (c *CompilerEnvironment) GetNextAnonymousFunctionKey(packageID *model.PackageID) string {
	nextValue := c.anonFuncCount[packageID]
	c.anonFuncCount[packageID] = nextValue + 1
	return anonPrefix + "Func$_" + strconv.Itoa(nextValue)
}

func (c *CompilerEnvironment) GetNextAnonymousTypeKey(packageID *model.PackageID) string {
	nextValue := c.anonTypeCount[packageID]
	c.anonTypeCount[packageID] = nextValue + 1
	if packageID != nil && model.ANNOTATIONS_PKG != packageID {
		return builtinAnonType + "_" + strconv.Itoa(nextValue)
	}
	return anonType + "_" + strconv.Itoa(nextValue)
}
