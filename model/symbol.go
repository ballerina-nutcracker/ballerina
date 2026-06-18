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

package model

import (
	"errors"
	"iter"
	"slices"
	"strings"
	"sync"

	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

type Scope interface {
	GetSymbol(name string) (SymbolRef, bool)
	GetPrefixedSymbol(prefix, name string) (SymbolRef, bool)
	AddSymbol(name string, symbol Symbol)
}

// XMLNSReservedPrefix is the predeclared prefix that cannot be redeclared.
const XMLNSReservedPrefix = "xmlns"

// XMLNSReservedURI is the URI bound to the predeclared `xmlns` prefix.
const XMLNSReservedURI = "http://www.w3.org/2000/xmlns/"

// DefaultXMLNSSymbolName is the prefix name used for the default XML namespace.
const DefaultXMLNSSymbolName = "$DEFAULT_XMLNS"

// SymbolSpaceProvider provides access to symbol spaces for block-level scopes
type SymbolSpaceProvider interface {
	MainSpace() *SymbolSpace
}

// BlockLevelScope combines Scope and SymbolSpaceProvider for block-level scopes
type BlockLevelScope interface {
	Scope
	SymbolSpaceProvider
}

// Symbol methods should never be called directly. Instead call them via the compiler context.
type Symbol interface {
	Name() string
	Type() semtypes.SemType
	Kind() SymbolKind
	SetType(semtypes.SemType)
	IsPublic() bool
	Copy() Symbol
}

// ValueSymbol is satisfied by both *VariableSymbol and *ConstantValueSymbol.
// Callers that only care about a symbol's value-symbol facets (whether it is a
// constant, parameter, final, etc.) should type-assert against this interface
// rather than the concrete *VariableSymbol, so that constants — which are stored as
// *ConstantValueSymbol — are matched too.
type ValueSymbol interface {
	Symbol
	IsConst() bool
	IsParameter() bool
	IsIsolated() bool
	IsFinal() bool
	IsConfigurable() bool
}

// symbolTypeSetter is a private interface for updating symbol types during type resolution.
// All concrete symbol types implement this through symbolBase.
type symbolTypeSetter interface {
	SetType(semtypes.SemType)
}

type FuncSymbolFlags uint8

type valueSymbolFlags uint8

const (
	FuncSymbolFlagIsolated FuncSymbolFlags = 1 << iota
	FuncSymbolFlagTransactional
)

const (
	valueSymbolFlagConst valueSymbolFlags = 1 << iota
	valueSymbolFlagParameter
	valueSymbolFlagIsolated
	valueSymbolFlagFinal
	valueSymbolFlagConfigurable
	valueSymbolFlagListener
)

type FunctionSymbol interface {
	Symbol
	Signature() FunctionSignature
	SetSignature(FunctionSignature)
	DefaultableParams() *DefaultableParamInfo
	SetDefaultableParams(DefaultableParamInfo)
	IncludedRecordParams() *IncludedRecordParamInfo
	SetIncludedRecordParams(*IncludedRecordParamInfo)
	ParamNames() []string
}

// DependentlyTypedFunctionSymbol represents a [dependently typed function]. Actual function signature
// is determined at each call site by calling Monomorphize.
// TODO: this is very similar to [GenericFunctionSymbol]; merge both. #389
type DependentlyTypedFunctionSymbol interface {
	FunctionSymbol
	Monomorphize(ctx semtypes.Context, name string, polymorphicRef SymbolRef, argTys []semtypes.SemType) FunctionSymbol
	ParamTypes() []semtypes.SemType
	ReturnType() TypeOp
	NRequiredArgs() int
	FuncFlags() FuncSymbolFlags
	SetParamTypes(types []semtypes.SemType)
	SetReturnType(op TypeOp)
}

// MonomorphicFunctionSymbol represent a polymorphic function after monomrophizisation.
// It carries a reference back to the underlying polymorphic function so BIR can dispatch to it,
// but front end can treat this as non polymorphic.
type MonomorphicFunctionSymbol interface {
	FunctionSymbol
	PolymorphicSymbol() SymbolRef
}

type BinaryTypeOpKind uint8

const (
	TypeOpUnion BinaryTypeOpKind = iota
	TypeOpIntersection
)

// TypeOp represents a partially applied type. This is necessary to represent things like
// return types of dependently typed functions in a serializable manner.
type TypeOp interface {
	Apply(ctx semtypes.Context, args []semtypes.SemType) semtypes.SemType
}

type BinaryTypeOp struct {
	Kind BinaryTypeOpKind
	Lhs  TypeOp
	Rhs  TypeOp
}

func (binary *BinaryTypeOp) Apply(ctx semtypes.Context, args []semtypes.SemType) semtypes.SemType {
	lhs := binary.Lhs.Apply(ctx, args)
	rhs := binary.Rhs.Apply(ctx, args)
	if binary.Kind == TypeOpUnion {
		return semtypes.Union(lhs, rhs)
	}
	return semtypes.Intersect(lhs, rhs)
}

type IdentityTypeOp struct {
	Type semtypes.SemType
}

func (identity *IdentityTypeOp) Apply(_ semtypes.Context, _ []semtypes.SemType) semtypes.SemType {
	return identity.Type
}

// RefTypeOp references a typedesc parameter by position. Apply resolves to the constraint T of args[Index] (typedesc<T>).
type RefTypeOp struct {
	Index int
}

func (ref *RefTypeOp) Apply(ctx semtypes.Context, args []semtypes.SemType) semtypes.SemType {
	return semtypes.TypedescConstraint(ctx, args[ref.Index])
}

type SymbolKind uint

const (
	SymbolKindType SymbolKind = iota
	SymbolKindConstant
	SymbolKindVariable
	SymbolKindParemeter
	SymbolKindFunction
	SymbolKindAnnotation
	SymbolKindXMLNS
)

const sourceAnnotationAttachPointPrefix = "source:"

type annotationAttachPointSet uint64

const sourceAnnotationAttachPointShift = 32

var annotationAttachPointKeys = [...]string{
	"type",
	"object",
	"function",
	"objectfunction",
	"serviceremotefunction",
	"parameter",
	"return",
	"service",
	"field",
	"objectfield",
	"recordfield",
	"listener",
	"annotation",
	"external",
	"var",
	"const",
	"worker",
	"class",
}

type (
	PackageIdentifier struct {
		Organization string
		Package      string
		Version      string
	}

	// We are using indeces here with the same rational as RefAtoms, instead of pointers
	SymbolRef struct {
		Index      int
		SpaceIndex int
	}

	ModuleScope struct {
		Main       *SymbolSpace
		Prefix     map[string]ExportedSymbolSpace
		Annotation *SymbolSpace
	}

	PackageScope struct {
		Virtual    *ModuleScope   // Virtual scope is used to store virtual symbols that don't have a compilation unit
		MainSpaces []*SymbolSpace // Virtual space + CU spaces
	}

	// ExportedSymbolSpace is a readonly representation of symbols exported by a Module.
	// A package can be backed by multiple compilation-unit symbol spaces.
	ExportedSymbolSpace struct {
		MainSpaces       []*SymbolSpace
		AnnotationSpaces []*SymbolSpace
	}

	BlockScopeBase struct {
		Parent Scope
		Main   *SymbolSpace
		Prefix map[string]ExportedSymbolSpace
	}

	// This is a delimiter to help detect if we need to capture a symbol as a closure
	// TODO: need to think how to implement closures correctly
	FunctionScope struct {
		BlockScopeBase
	}

	BlockScope struct {
		BlockScopeBase
	}

	SymbolSpace struct {
		mu          sync.RWMutex
		Pkg         PackageIdentifier
		lookupTable map[string]int
		symbols     []Symbol
		index       int
	}

	symbolBase struct {
		name     string
		ty       semtypes.SemType
		isPublic bool
	}

	TypeSymbol struct {
		symbolBase
	}

	AnnotationSymbol struct {
		symbolBase
		isConst      bool
		attachPoints annotationAttachPointSet
	}

	ErrorTypeSymbol struct {
		TypeSymbol
		distinctTypeBase
	}

	// memberHolderBase carries direct + type-inclusion-inherited members
	// (fields and optional rest-type for records; fields + methods for classes
	// and object type aliases).
	memberHolderBase struct {
		members []InclusionMember
	}

	distinctTypeBase struct {
		typeIDs []int
	}

	classSymbolBase struct {
		TypeSymbol
		memberHolderBase
		distinctTypeBase
		methods         map[string]SymbolRef
		resourceMethods []SymbolRef
	}

	classSymbol struct {
		classSymbolBase
	}

	NetworkClassSymbol struct {
		classSymbolBase
	}

	RecordSymbol struct {
		TypeSymbol
		memberHolderBase
	}

	ObjectTypeSymbol struct {
		TypeSymbol
		memberHolderBase
		distinctTypeBase
	}

	FieldDefault struct {
		FieldName string
		FnRef     SymbolRef
	}

	VariableSymbol struct {
		symbolBase
		flags valueSymbolFlags
	}

	// ConstantValueSymbol is a VariableSymbol for a module constant whose
	// expression has been folded at compile time. The value is stored on the
	// symbol because it cannot always be recovered from the constant's type.
	ConstantValueSymbol struct {
		VariableSymbol
		value values.BalValue
	}

	XMLNSSymbol struct {
		symbolBase
		uri string
	}

	functionSymbol struct {
		symbolBase
		signature            FunctionSignature
		defaultableParams    DefaultableParamInfo
		includedRecordParams *IncludedRecordParamInfo
	}

	monomorphicFunctionSymbol struct {
		functionSymbol
		polymorhpicFn SymbolRef
	}

	ResourceMethodSymbol struct {
		functionSymbol
		methodName   string
		pathListType semtypes.SemType
		pathParams   []SymbolRef
	}

	dependentlyTypedFunctionSymbol struct {
		symbolBase
		paramNames           []string
		nRequiredArgs        int
		Flags                FuncSymbolFlags
		defaultable          DefaultableParamInfo
		includedRecordParams *IncludedRecordParamInfo

		// Populated by type resolver at stage 4.
		paramTypes []semtypes.SemType
		retType    TypeOp
	}

	FunctionSignature struct {
		ParamTypes    []semtypes.SemType
		ParamNames    []string
		ReturnType    semtypes.SemType
		RestParamType semtypes.SemType
		Flags         FuncSymbolFlags
	}

	DefaultableParamKind uint8

	DefaultableParam struct {
		Symbol SymbolRef
		Kind   DefaultableParamKind
	}

	DefaultableParamInfo struct {
		params      []DefaultableParam
		defaultable []bool
	}

	IncludedRecordParamInfo struct {
		params     []bool
		fieldNames [][]string
	}
)

func (ref SymbolRef) IsEmpty() bool {
	return ref == SymbolRef{}
}

const (
	DefaultableParamKindExpr DefaultableParamKind = iota
	DefaultableParamKindInferredTypedesc
)

type InclusionMemberKind uint8

const (
	InclusionMemberKindField InclusionMemberKind = iota
	InclusionMemberKindMethod
	InclusionMemberKindRemoteMethod
	InclusionMemberKindResourceMethod
	InclusionMemberKindRestType
)

type InclusionMember interface {
	MemberName() string
	MemberKind() InclusionMemberKind
	MemberType() semtypes.SemType
	SetMemberType(semtypes.SemType)
}

type FieldDescriptorFlag uint8

const (
	FieldDescriptorReadonly FieldDescriptorFlag = 1 << iota
	FieldDescriptorOptional
	FieldDescriptorHasDefault
)

type FieldDescriptor struct {
	name         string
	ty           semtypes.SemType
	flags        FieldDescriptorFlag
	DefaultFnRef SymbolRef
	isPublic     bool
}

func NewFieldDescriptor(name string, flags FieldDescriptorFlag, isPublic bool) FieldDescriptor {
	return FieldDescriptor{name: name, flags: flags, isPublic: isPublic}
}

func (f *FieldDescriptor) MemberName() string                { return f.name }
func (f *FieldDescriptor) MemberKind() InclusionMemberKind   { return InclusionMemberKindField }
func (f *FieldDescriptor) MemberType() semtypes.SemType      { return f.ty }
func (f *FieldDescriptor) SetMemberType(ty semtypes.SemType) { f.ty = ty }
func (f *FieldDescriptor) IsPublic() bool                    { return f.isPublic }
func (f *FieldDescriptor) IsReadonly() bool                  { return f.flags&FieldDescriptorReadonly != 0 }
func (f *FieldDescriptor) IsOptional() bool                  { return f.flags&FieldDescriptorOptional != 0 }
func (f *FieldDescriptor) HasDefault() bool                  { return f.flags&FieldDescriptorHasDefault != 0 }

type MethodDescriptor struct {
	name      string
	kind      InclusionMemberKind
	ty        semtypes.SemType
	MethodRef SymbolRef
	isPublic  bool
}

func NewMethodDescriptor(name string, kind InclusionMemberKind, isPublic bool, methodRef SymbolRef) MethodDescriptor {
	return MethodDescriptor{name: name, kind: kind, isPublic: isPublic, MethodRef: methodRef}
}

func (m *MethodDescriptor) MemberName() string                { return m.name }
func (m *MethodDescriptor) MemberKind() InclusionMemberKind   { return m.kind }
func (m *MethodDescriptor) MemberType() semtypes.SemType      { return m.ty }
func (m *MethodDescriptor) SetMemberType(ty semtypes.SemType) { m.ty = ty }
func (m *MethodDescriptor) IsPublic() bool                    { return m.isPublic }

type RestTypeDescriptor struct {
	ty semtypes.SemType
}

func NewRestTypeDescriptor() RestTypeDescriptor {
	return RestTypeDescriptor{}
}

func (r *RestTypeDescriptor) MemberName() string                { panic("RestTypeDescriptor has no name") }
func (r *RestTypeDescriptor) MemberKind() InclusionMemberKind   { return InclusionMemberKindRestType }
func (r *RestTypeDescriptor) MemberType() semtypes.SemType      { return r.ty }
func (r *RestTypeDescriptor) SetMemberType(ty semtypes.SemType) { r.ty = ty }

var (
	_ InclusionMember = &FieldDescriptor{}
	_ InclusionMember = &MethodDescriptor{}
	_ InclusionMember = &RestTypeDescriptor{}
)

var (
	_ Scope                          = &ModuleScope{}
	_ Scope                          = &PackageScope{}
	_ Scope                          = &FunctionScope{}
	_ Scope                          = &BlockScope{}
	_ Symbol                         = &TypeSymbol{}
	_ Symbol                         = &AnnotationSymbol{}
	_ Symbol                         = &classSymbol{}
	_ Symbol                         = &NetworkClassSymbol{}
	_ ClassSymbol                    = &classSymbol{}
	_ ClassSymbol                    = &NetworkClassSymbol{}
	_ Symbol                         = &RecordSymbol{}
	_ Symbol                         = &ObjectTypeSymbol{}
	_ MemberCarrier                  = &classSymbol{}
	_ MemberCarrier                  = &NetworkClassSymbol{}
	_ MemberCarrier                  = &RecordSymbol{}
	_ MemberCarrier                  = &ObjectTypeSymbol{}
	_ Symbol                         = &VariableSymbol{}
	_ Symbol                         = &ConstantValueSymbol{}
	_ ValueSymbol                    = &VariableSymbol{}
	_ ValueSymbol                    = &ConstantValueSymbol{}
	_ Symbol                         = &XMLNSSymbol{}
	_ Symbol                         = &functionSymbol{}
	_ FunctionSymbol                 = &functionSymbol{}
	_ DependentlyTypedFunctionSymbol = &dependentlyTypedFunctionSymbol{}
	_ MonomorphicFunctionSymbol      = &monomorphicFunctionSymbol{}
	_ FunctionSymbol                 = &ResourceMethodSymbol{}
	_ Symbol                         = &SymbolRef{}
	_ SymbolSpaceProvider            = &ModuleScope{}
	_ SymbolSpaceProvider            = &PackageScope{}
)

func (space *SymbolSpace) AddSymbol(name string, symbol Symbol) {
	if _, ok := symbol.(*SymbolRef); ok {
		panic("SymbolRef cannot be added to a SymbolSpace")
	}
	space.mu.Lock()
	space.lookupTable[name] = len(space.symbols)
	space.symbols = append(space.symbols, symbol)
	space.mu.Unlock()
}

func (space *SymbolSpace) GetSymbol(name string) (SymbolRef, bool) {
	space.mu.RLock()
	index, ok := space.lookupTable[name]
	space.mu.RUnlock()
	if !ok {
		return SymbolRef{}, false
	}
	return SymbolRef{Index: index, SpaceIndex: space.SpaceIndex()}, true
}

// AppendSymbol appends a symbol to the space and returns its index. Thread-safe.
func (space *SymbolSpace) AppendSymbol(symbol Symbol) int {
	// We really need this lock only for module level symbols but we don't distinguish between module level space and other spaces
	space.mu.Lock()
	defer space.mu.Unlock()
	index := len(space.symbols)
	space.symbols = append(space.symbols, symbol)
	return index
}

// RefAt returns a SymbolRef for the symbol at the given index.
func (space *SymbolSpace) RefAt(index int) SymbolRef {
	return SymbolRef{Index: index, SpaceIndex: space.SpaceIndex()}
}

// SpaceIndex returns the non-zero symbol-space index used in SymbolRef.
func (space *SymbolSpace) SpaceIndex() int {
	return space.index + 1
}

func (space *SymbolSpace) SymbolAt(index int) Symbol {
	space.mu.RLock()
	defer space.mu.RUnlock()
	return space.symbols[index]
}

func (space *SymbolSpace) Len() int {
	space.mu.RLock()
	defer space.mu.RUnlock()
	return len(space.symbols)
}

// Symbols returns an iterator over a snapshot of the symbol references in the
// space. Symbols appended after the snapshot is taken are not included.
func (space *SymbolSpace) Symbols() iter.Seq[SymbolRef] {
	return func(yield func(SymbolRef) bool) {
		space.mu.RLock()
		refs := make([]SymbolRef, len(space.symbols))
		for i := range space.symbols {
			refs[i] = space.RefAt(i)
		}
		space.mu.RUnlock()
		for _, ref := range refs {
			if !yield(ref) {
				return
			}
		}
	}
}

func NewSymbolSpaceInner(packageID PackageID, index int) *SymbolSpace {
	return &SymbolSpace{index: index, Pkg: PackageIdentifierFromID(&packageID), lookupTable: make(map[string]int), symbols: make([]Symbol, 0)}
}

func PackageIdentifierFromID(id *PackageID) PackageIdentifier {
	return PackageIdentifier{
		Organization: id.OrgName.Value(),
		Package:      id.PkgName.Value(),
		Version:      id.Version.Value(),
	}
}

func (ms *ModuleScope) Exports() ExportedSymbolSpace {
	return NewExportedSymbolSpaces([]*SymbolSpace{ms.Main}, []*SymbolSpace{ms.Annotation})
}

func (ms *ModuleScope) GetSymbol(name string) (SymbolRef, bool) {
	return ms.Main.GetSymbol(name)
}

func (ms *ModuleScope) MainSpace() *SymbolSpace {
	return ms.Main
}

func mapToLangPrefixIfNeeded(prefix string) string {
	switch prefix {
	case "int":
		return "lang.int"
	case "boolean":
		return "lang.boolean"
	case "decimal":
		return "lang.decimal"
	case "array":
		return "lang.array"
	case "map":
		return "lang.map"
	case "string":
		return "lang.string"
	case "xml":
		return "lang.xml"
	case "object":
		return "lang.object"
	default:
		return prefix
	}
}

func (ms *ModuleScope) GetPrefixedSymbol(prefix, name string) (SymbolRef, bool) {
	if prefix == "" {
		return ms.Main.GetSymbol(name)
	}
	exported, ok := ms.Prefix[prefix]
	if !ok {
		exported, ok = ms.Prefix[mapToLangPrefixIfNeeded(prefix)]
		if !ok {
			return SymbolRef{}, false
		}
	}
	return exported.GetSymbol(name)
}

func (ms *ModuleScope) GetAnnotationSymbol(prefix, name string) (SymbolRef, bool) {
	if prefix == "" {
		return ms.Annotation.GetSymbol(name)
	}
	exported, ok := ms.Prefix[prefix]
	if !ok {
		exported, ok = ms.Prefix[mapToLangPrefixIfNeeded(prefix)]
		if !ok {
			return SymbolRef{}, false
		}
	}
	return exported.GetAnnotationSymbol(name)
}

func (ms *ModuleScope) AddSymbol(name string, symbol Symbol) {
	ms.Main.AddSymbol(name, symbol)
}

func (ms *ModuleScope) AddAnnotationSymbol(name string, symbol Symbol) {
	ms.Annotation.AddSymbol(name, symbol)
}

func (ps *PackageScope) GetSymbol(name string) (SymbolRef, bool) {
	for _, main := range ps.MainSpaces {
		if ref, ok := main.GetSymbol(name); ok {
			return ref, true
		}
	}
	return ps.Virtual.GetSymbol(name)
}

func (ps *PackageScope) GetPrefixedSymbol(prefix, name string) (SymbolRef, bool) {
	return ps.Virtual.GetPrefixedSymbol(prefix, name)
}

func (ps *PackageScope) AddSymbol(name string, symbol Symbol) {
	ps.Virtual.AddSymbol(name, symbol)
}

func (ps *PackageScope) MainSpace() *SymbolSpace {
	return ps.Virtual.Main
}

func NewExportedSymbolSpaces(mainSpaces, annotationSpaces []*SymbolSpace) ExportedSymbolSpace {
	return ExportedSymbolSpace{MainSpaces: mainSpaces, AnnotationSpaces: annotationSpaces}
}

func (space *ExportedSymbolSpace) PublicMainSymbols() iter.Seq[SymbolRef] {
	return func(yield func(SymbolRef) bool) {
		for _, main := range space.MainSpaces {
			for ref := range main.Symbols() {
				if !main.SymbolAt(ref.Index).IsPublic() {
					continue
				}
				if !yield(ref) {
					return
				}
			}
		}
	}
}

func (space *ExportedSymbolSpace) GetSymbol(name string) (SymbolRef, bool) {
	for _, main := range space.MainSpaces {
		ref, ok := main.GetSymbol(name)
		if !ok {
			continue
		}
		sym := main.SymbolAt(ref.Index)
		if !sym.IsPublic() {
			return SymbolRef{}, false
		}
		return ref, true
	}
	return SymbolRef{}, false
}

func (space *ExportedSymbolSpace) GetAnnotationSymbol(name string) (SymbolRef, bool) {
	for _, annotationSpace := range space.AnnotationSpaces {
		ref, ok := annotationSpace.GetSymbol(name)
		if !ok {
			continue
		}
		sym := annotationSpace.SymbolAt(ref.Index)
		if !sym.IsPublic() {
			return SymbolRef{}, false
		}
		return ref, true
	}
	return SymbolRef{}, false
}

func (bs *BlockScopeBase) GetSymbol(name string) (SymbolRef, bool) {
	ref, ok := bs.Main.GetSymbol(name)
	if ok {
		return ref, true
	}
	return bs.Parent.GetSymbol(name)
}

func (bs *BlockScopeBase) GetPrefixedSymbol(prefix, name string) (SymbolRef, bool) {
	if bs.Prefix != nil {
		if exported, ok := bs.Prefix[prefix]; ok {
			return exported.GetSymbol(name)
		}
	}
	return bs.Parent.GetPrefixedSymbol(prefix, name)
}

func (bs *BlockScopeBase) AddSymbol(name string, symbol Symbol) {
	bs.Main.AddSymbol(name, symbol)
}

func (bs *BlockScopeBase) MainSpace() *SymbolSpace {
	return bs.Main
}

func (ba *symbolBase) Name() string {
	return ba.name
}

func (ba *symbolBase) Type() semtypes.SemType {
	return ba.ty
}

func (ba *symbolBase) SetType(ty semtypes.SemType) {
	ba.ty = ty
}

func (ba *symbolBase) IsPublic() bool {
	return ba.isPublic
}

func (ref *SymbolRef) Name() string {
	panic("unexpected")
}

func (ref *SymbolRef) Type() semtypes.SemType {
	panic("unexpected")
}

func (ref *SymbolRef) SetType(ty semtypes.SemType) {
	panic("unexpected")
}

func (ref *SymbolRef) Kind() SymbolKind {
	panic("unexpected")
}

func (ref *SymbolRef) IsPublic() bool {
	panic("unexpected")
}

func (ref *SymbolRef) Copy() Symbol {
	panic("SymbolRef can't be copied")
}

func (ts *TypeSymbol) Kind() SymbolKind {
	return SymbolKindType
}

func (ts *TypeSymbol) Copy() Symbol {
	panic("TypeSymbol cannot be copied")
}

func (as *AnnotationSymbol) Kind() SymbolKind {
	return SymbolKindAnnotation
}

func (as *AnnotationSymbol) Copy() Symbol {
	panic("AnnotationSymbol cannot be copied")
}

func (as *AnnotationSymbol) IsConst() bool {
	return as.isConst
}

func (as *AnnotationSymbol) AttachPointKeys() []string {
	keys := make([]string, 0, len(annotationAttachPointKeys))
	for i, key := range annotationAttachPointKeys {
		bit := annotationAttachPointSet(1 << i)
		if as.attachPoints.has(bit) {
			keys = append(keys, key)
		}
		if as.attachPoints.has(bit << sourceAnnotationAttachPointShift) {
			keys = append(keys, SourceAnnotationAttachPointKey(key))
		}
	}
	return keys
}

func (as *AnnotationSymbol) AllowsAttachPoint(point string) bool {
	if as.attachPoints == 0 {
		return true
	}
	if as.hasAttachPoint(point) {
		return true
	}
	if point == "recordfield" || point == "objectfield" {
		if as.hasAttachPoint("field") || as.hasSourceAttachPoint("field") {
			return true
		}
	}
	return as.hasSourceAttachPoint(point)
}

func (as *AnnotationSymbol) IsRuntimeVisibleAt(point string) bool {
	if as.attachPoints == 0 {
		return true
	}
	if point == "recordfield" || point == "objectfield" {
		return as.hasAttachPoint(point) || as.hasAttachPoint("field")
	}
	return as.hasAttachPoint(point)
}

func AnnotationKey(pkg PackageIdentifier, name string) string {
	return pkg.Organization + "/" + pkg.Package + ":" + pkg.Version + ":" + name
}

func SourceAnnotationAttachPointKey(point string) string {
	return sourceAnnotationAttachPointPrefix + point
}

func (as *AnnotationSymbol) hasAttachPoint(point string) bool {
	bit, ok := annotationAttachPointBit(point)
	return ok && as.attachPoints.has(bit)
}

func (as *AnnotationSymbol) hasSourceAttachPoint(point string) bool {
	bit, ok := annotationAttachPointBit(point)
	return ok && as.attachPoints.has(bit<<sourceAnnotationAttachPointShift)
}

func (s annotationAttachPointSet) has(bit annotationAttachPointSet) bool {
	return s&bit != 0
}

func annotationAttachPointBit(point string) (annotationAttachPointSet, bool) {
	switch point {
	case "type":
		return 1 << 0, true
	case "object":
		return 1 << 1, true
	case "function":
		return 1 << 2, true
	case "objectfunction":
		return 1 << 3, true
	case "serviceremotefunction":
		return 1 << 4, true
	case "parameter":
		return 1 << 5, true
	case "return":
		return 1 << 6, true
	case "service":
		return 1 << 7, true
	case "field":
		return 1 << 8, true
	case "objectfield":
		return 1 << 9, true
	case "recordfield":
		return 1 << 10, true
	case "listener":
		return 1 << 11, true
	case "annotation":
		return 1 << 12, true
	case "external":
		return 1 << 13, true
	case "var":
		return 1 << 14, true
	case "const":
		return 1 << 15, true
	case "worker":
		return 1 << 16, true
	case "class":
		return 1 << 17, true
	default:
		return 0, false
	}
}

// MemberCarrier is implemented by symbols that carry direct + inclusion-inherited members.
// TypeSymbol does not implement this; only RecordSymbol, ClassSymbol, and ObjectTypeSymbol do.
type MemberCarrier interface {
	Members() []InclusionMember
	AddMember(InclusionMember)
	FieldDefaults() []FieldDefault
}

type ObjectType interface {
	DistinctTypeIDs() []int
	SetDistinctTypeIDs(typeIDs []int)
}

type ClassSymbol interface {
	Symbol
	MemberCarrier
	ObjectType
	SetMethods(map[string]SymbolRef)
	MethodSymbol(name string) (SymbolRef, bool)
}

func (m *memberHolderBase) Members() []InclusionMember { return m.members }
func (m *memberHolderBase) AddMember(im InclusionMember) {
	m.members = append(m.members, im)
}

func (m *memberHolderBase) FieldDefaults() []FieldDefault {
	var defaults []FieldDefault
	for _, im := range m.members {
		if fd, ok := im.(*FieldDescriptor); ok && !fd.DefaultFnRef.IsEmpty() {
			defaults = append(defaults, FieldDefault{FieldName: fd.name, FnRef: fd.DefaultFnRef})
		}
	}
	return defaults
}

func (d *distinctTypeBase) DistinctTypeIDs() []int {
	return d.typeIDs
}

func (d *distinctTypeBase) SetDistinctTypeIDs(typeIDs []int) {
	d.typeIDs = typeIDs
}

func (r *RecordSymbol) Fields() iter.Seq2[string, *FieldDescriptor] {
	return func(yield func(string, *FieldDescriptor) bool) {
		for _, m := range r.members {
			fd, ok := m.(*FieldDescriptor)
			if !ok {
				continue
			}
			if !yield(fd.name, fd) {
				return
			}
		}
	}
}

func (r *RecordSymbol) Field(name string) (*FieldDescriptor, bool) {
	for _, m := range r.members {
		if fd, ok := m.(*FieldDescriptor); ok && fd.name == name {
			return fd, true
		}
	}
	return nil, false
}

func (r *RecordSymbol) RestField() (*RestTypeDescriptor, bool) {
	for _, m := range r.members {
		if rd, ok := m.(*RestTypeDescriptor); ok {
			return rd, true
		}
	}
	return nil, false
}

func (vs *VariableSymbol) Kind() SymbolKind {
	if vs.hasFlag(valueSymbolFlagConst) {
		return SymbolKindConstant
	}
	if vs.hasFlag(valueSymbolFlagParameter) {
		return SymbolKindParemeter
	}
	return SymbolKindVariable
}

func (vs *VariableSymbol) IsConst() bool {
	return vs.hasFlag(valueSymbolFlagConst) || vs.hasFlag(valueSymbolFlagParameter)
}

func (vs *VariableSymbol) IsParameter() bool { return vs.hasFlag(valueSymbolFlagParameter) }

func (vs *VariableSymbol) IsIsolated() bool { return vs.hasFlag(valueSymbolFlagIsolated) }

func (vs *VariableSymbol) SetIsolated() { vs.setFlag(valueSymbolFlagIsolated) }

func (vs *VariableSymbol) IsFinal() bool { return vs.hasFlag(valueSymbolFlagFinal) }

func (vs *VariableSymbol) SetFinal() { vs.setFlag(valueSymbolFlagFinal) }

func (vs *VariableSymbol) IsConfigurable() bool { return vs.hasFlag(valueSymbolFlagConfigurable) }

func (vs *VariableSymbol) SetConfigurable() { vs.setFlag(valueSymbolFlagConfigurable) }

func (vs *VariableSymbol) IsListener() bool { return vs.hasFlag(valueSymbolFlagListener) }

func (vs *VariableSymbol) SetListener() { vs.setFlag(valueSymbolFlagListener) }

func (vs *VariableSymbol) hasFlag(flag valueSymbolFlags) bool { return vs.flags&flag != 0 }

func (vs *VariableSymbol) setFlag(flag valueSymbolFlags) { vs.flags |= flag }

func (vs *VariableSymbol) Copy() Symbol {
	cp := *vs
	return &cp
}

// SetConstantValue records the folded value for this constant.
func (cs *ConstantValueSymbol) SetConstantValue(value values.BalValue) {
	cs.value = value
}

// ConstantValue returns the folded value.
func (cs *ConstantValueSymbol) ConstantValue() values.BalValue {
	return cs.value
}

func (cs *ConstantValueSymbol) Copy() Symbol {
	cp := *cs
	return &cp
}

func (xs *XMLNSSymbol) Kind() SymbolKind {
	return SymbolKindXMLNS
}

func (xs *XMLNSSymbol) URI() string {
	return xs.uri
}

func (xs *XMLNSSymbol) Copy() Symbol {
	cp := *xs
	return &cp
}

func XMLNamespaceURI(symbol Symbol) (string, error) {
	xmlns, ok := symbol.(*XMLNSSymbol)
	if !ok {
		return "", errors.New("expected XML namespace symbol")
	}
	return xmlns.URI(), nil
}

func XMLNamespaceDeclKey(symbol Symbol) (string, error) {
	xmlns, ok := symbol.(*XMLNSSymbol)
	if !ok {
		return "", errors.New("expected XML namespace symbol")
	}
	name := xmlns.Name()
	if name == DefaultXMLNSSymbolName {
		return "xmlns", nil
	}
	return "xmlns:" + name, nil
}

func (fs *functionSymbol) Kind() SymbolKind {
	return SymbolKindFunction
}

func (fs *functionSymbol) Copy() Symbol {
	cp := *fs
	return &cp
}

func (fs *functionSymbol) Signature() FunctionSignature {
	return fs.signature
}

func (fs *functionSymbol) SetSignature(sig FunctionSignature) {
	fs.signature = sig
}

func (fs *functionSymbol) DefaultableParams() *DefaultableParamInfo {
	return &fs.defaultableParams
}

func (fs *functionSymbol) SetDefaultableParams(info DefaultableParamInfo) {
	fs.defaultableParams = info
}

func (fs *functionSymbol) IncludedRecordParams() *IncludedRecordParamInfo {
	return fs.includedRecordParams
}

func (fs *functionSymbol) SetIncludedRecordParams(info *IncludedRecordParamInfo) {
	fs.includedRecordParams = info
}

func (fs *functionSymbol) ParamNames() []string {
	return fs.Signature().ParamNames
}

func NewFunctionSymbol(name string, signature FunctionSignature, isPublic bool) FunctionSymbol {
	return &functionSymbol{
		symbolBase: symbolBase{name: name, isPublic: isPublic},
		signature:  signature,
	}
}

func NewDefaultableParamInfo(paramCount int) DefaultableParamInfo {
	return DefaultableParamInfo{
		params:      make([]DefaultableParam, paramCount),
		defaultable: make([]bool, paramCount),
	}
}

func (d *DefaultableParamInfo) Get(index int) (DefaultableParam, bool) {
	if index >= len(d.defaultable) || !d.defaultable[index] {
		return DefaultableParam{}, false
	}
	return d.params[index], true
}

func (d *DefaultableParamInfo) SetDefaultable(index int, symbol SymbolRef) {
	d.defaultable[index] = true
	d.params[index] = DefaultableParam{Symbol: symbol, Kind: DefaultableParamKindExpr}
}

func (d *DefaultableParamInfo) SetInferredTypedesc(index int) {
	d.defaultable[index] = true
	d.params[index] = DefaultableParam{Kind: DefaultableParamKindInferredTypedesc}
}

func NewIncludedRecordParamInfo(paramCount int) *IncludedRecordParamInfo {
	return &IncludedRecordParamInfo{
		params:     make([]bool, paramCount),
		fieldNames: make([][]string, paramCount),
	}
}

func (i *IncludedRecordParamInfo) Set(index int) {
	i.params[index] = true
}

func (i *IncludedRecordParamInfo) SetFields(index int, names []string) {
	i.fieldNames[index] = names
}

func (i *IncludedRecordParamInfo) Fields(index int) []string {
	if index >= len(i.fieldNames) {
		return nil
	}
	return i.fieldNames[index]
}

func (i *IncludedRecordParamInfo) LookupField(name string) (int, bool) {
	for idx, names := range i.fieldNames {
		if slices.Contains(names, name) {
			return idx, true
		}
	}
	return -1, false
}

func (i *IncludedRecordParamInfo) IsIncluded(index int) bool {
	if index >= len(i.params) {
		return false
	}
	return i.params[index]
}

func (i *IncludedRecordParamInfo) Len() int {
	return len(i.params)
}

func (fs *FunctionSignature) IsIsolated() bool {
	return fs.Flags&FuncSymbolFlagIsolated != 0
}

func (fs *FunctionSignature) IsTransactional() bool {
	return fs.Flags&FuncSymbolFlagTransactional != 0
}

func NewVariableSymbol(name string, isPublic bool, isConst bool, isParameter bool) VariableSymbol {
	var flags valueSymbolFlags
	if isConst {
		flags |= valueSymbolFlagConst
	}
	if isParameter {
		flags |= valueSymbolFlagParameter
	}
	return VariableSymbol{
		symbolBase: symbolBase{name: name, isPublic: isPublic},
		flags:      flags,
	}
}

func NewConstantValueSymbol(name string, isPublic bool) *ConstantValueSymbol {
	return &ConstantValueSymbol{
		VariableSymbol: NewVariableSymbol(name, isPublic, true, false),
	}
}

func NewTypeSymbol(name string, isPublic bool) TypeSymbol {
	return TypeSymbol{
		symbolBase: symbolBase{name: name, isPublic: isPublic},
	}
}

func NewAnnotationSymbol(name string, isPublic bool, isConst bool, attachPoints []string) AnnotationSymbol {
	var attachPointSet annotationAttachPointSet
	for _, point := range attachPoints {
		source := false
		if strings.HasPrefix(point, sourceAnnotationAttachPointPrefix) {
			source = true
			point = strings.TrimPrefix(point, sourceAnnotationAttachPointPrefix)
		}
		bit, ok := annotationAttachPointBit(point)
		if !ok {
			panic("unknown annotation attach point")
		}
		if source {
			bit <<= sourceAnnotationAttachPointShift
		}
		attachPointSet |= bit
	}
	return AnnotationSymbol{
		symbolBase:   symbolBase{name: name, isPublic: isPublic},
		isConst:      isConst,
		attachPoints: attachPointSet,
	}
}

func NewXMLNSSymbol(prefix, uri string) *XMLNSSymbol {
	return &XMLNSSymbol{
		symbolBase: symbolBase{name: prefix, isPublic: true},
		uri:        uri,
	}
}

func NewClassSymbol(name string, isPublic bool) ClassSymbol {
	return &classSymbol{
		classSymbolBase: newClassSymbolBase(name, isPublic),
	}
}

// NewNetworkClassSymbol creates a ClassSymbol for classes representing network
// interaction objects (e.g. clients and services).
func NewNetworkClassSymbol(name string, isPublic bool) ClassSymbol {
	return &NetworkClassSymbol{
		classSymbolBase: newClassSymbolBase(name, isPublic),
	}
}

func newClassSymbolBase(name string, isPublic bool) classSymbolBase {
	return classSymbolBase{
		TypeSymbol: TypeSymbol{
			symbolBase: symbolBase{name: name, isPublic: isPublic},
		},
		methods: map[string]SymbolRef{},
	}
}

func (c *classSymbolBase) ResourceMethods() []SymbolRef {
	return c.resourceMethods
}

func (c *classSymbolBase) AddResourceMethod(ref SymbolRef) {
	c.resourceMethods = append(c.resourceMethods, ref)
}

func NewResourceMethodSymbol(name, methodName string, isPublic bool) *ResourceMethodSymbol {
	return &ResourceMethodSymbol{
		functionSymbol: functionSymbol{
			symbolBase: symbolBase{name: name, isPublic: isPublic},
		},
		methodName: methodName,
	}
}

func (r *ResourceMethodSymbol) MethodName() string {
	return r.methodName
}

func (r *ResourceMethodSymbol) PathListType() semtypes.SemType {
	return r.pathListType
}

func (r *ResourceMethodSymbol) SetPathListType(ty semtypes.SemType) {
	r.pathListType = ty
}

func (r *ResourceMethodSymbol) PathParams() []SymbolRef {
	return r.pathParams
}

func (r *ResourceMethodSymbol) SetPathParams(params []SymbolRef) {
	r.pathParams = params
}

func (r *ResourceMethodSymbol) Copy() Symbol {
	cp := *r
	return &cp
}

func NewRecordSymbol(name string, isPublic bool) RecordSymbol {
	return RecordSymbol{
		TypeSymbol: TypeSymbol{
			symbolBase: symbolBase{name: name, isPublic: isPublic},
		},
	}
}

func NewObjectTypeSymbol(name string, isPublic bool) ObjectTypeSymbol {
	return ObjectTypeSymbol{
		TypeSymbol: TypeSymbol{
			symbolBase: symbolBase{name: name, isPublic: isPublic},
		},
	}
}

func NewErrorTypeSymbol(name string, isPublic bool) ErrorTypeSymbol {
	return ErrorTypeSymbol{
		TypeSymbol: TypeSymbol{
			symbolBase: symbolBase{name: name, isPublic: isPublic},
		},
	}
}

func (c *classSymbolBase) SetMethods(methods map[string]SymbolRef) {
	c.methods = methods
}

func (c *classSymbolBase) MethodSymbol(name string) (SymbolRef, bool) {
	ref, ok := c.methods[name]
	return ref, ok
}

func NewDependentlyTypedFunctionSymbol(name string, paramNames []string, nRequiredArgs int, flags FuncSymbolFlags, isPublic bool) DependentlyTypedFunctionSymbol {
	return &dependentlyTypedFunctionSymbol{
		symbolBase:    symbolBase{name: name, isPublic: isPublic},
		paramNames:    paramNames,
		nRequiredArgs: nRequiredArgs,
		Flags:         flags,
	}
}

func (s *dependentlyTypedFunctionSymbol) Kind() SymbolKind { return SymbolKindFunction }

func (s *dependentlyTypedFunctionSymbol) Type() semtypes.SemType {
	panic("DependentlyTypedFunctionSymbol must be Monomorphized")
}

func (s *dependentlyTypedFunctionSymbol) SetType(_ semtypes.SemType) {
	panic("DependentlyTypedFunctionSymbol must be Monomorphized")
}

func (s *dependentlyTypedFunctionSymbol) Signature() FunctionSignature {
	panic("DependentlyTypedFunctionSymbol must be Monomorphized")
}

func (s *dependentlyTypedFunctionSymbol) SetSignature(_ FunctionSignature) {
	panic("DependentlyTypedFunctionSymbol must be Monomorphized")
}

func (s *dependentlyTypedFunctionSymbol) DefaultableParams() *DefaultableParamInfo {
	return &s.defaultable
}

func (s *dependentlyTypedFunctionSymbol) SetDefaultableParams(info DefaultableParamInfo) {
	s.defaultable = info
}

func (s *dependentlyTypedFunctionSymbol) IncludedRecordParams() *IncludedRecordParamInfo {
	return s.includedRecordParams
}

func (s *dependentlyTypedFunctionSymbol) SetIncludedRecordParams(info *IncludedRecordParamInfo) {
	s.includedRecordParams = info
}

func (s *dependentlyTypedFunctionSymbol) ParamNames() []string { return s.paramNames }

func (s *dependentlyTypedFunctionSymbol) Copy() Symbol {
	cp := *s
	return &cp
}

func (s *dependentlyTypedFunctionSymbol) ParamTypes() []semtypes.SemType { return s.paramTypes }
func (s *dependentlyTypedFunctionSymbol) ReturnType() TypeOp             { return s.retType }
func (s *dependentlyTypedFunctionSymbol) NRequiredArgs() int             { return s.nRequiredArgs }
func (s *dependentlyTypedFunctionSymbol) FuncFlags() FuncSymbolFlags     { return s.Flags }

func (s *dependentlyTypedFunctionSymbol) SetParamTypes(types []semtypes.SemType) {
	s.paramTypes = types
}

func (s *dependentlyTypedFunctionSymbol) SetReturnType(op TypeOp) {
	s.retType = op
}

func (s *dependentlyTypedFunctionSymbol) Monomorphize(ctx semtypes.Context, name string, origRef SymbolRef, argTys []semtypes.SemType) FunctionSymbol {
	fixed := argTys
	rest := semtypes.NEVER
	if len(argTys) > s.nRequiredArgs {
		fixed = argTys[:s.nRequiredArgs]
		for _, each := range argTys[s.nRequiredArgs:] {
			rest = semtypes.Union(rest, each)
		}
	}
	returnType := s.retType.Apply(ctx, argTys)
	sig := FunctionSignature{
		ParamTypes:    fixed,
		ParamNames:    s.paramNames,
		RestParamType: rest,
		ReturnType:    returnType,
		Flags:         s.Flags,
	}
	return &monomorphicFunctionSymbol{
		functionSymbol: functionSymbol{
			symbolBase:           symbolBase{name: name, isPublic: s.isPublic},
			signature:            sig,
			defaultableParams:    s.defaultable,
			includedRecordParams: s.includedRecordParams,
		},
		polymorhpicFn: origRef,
	}
}

func (m *monomorphicFunctionSymbol) PolymorphicSymbol() SymbolRef { return m.polymorhpicFn }

func (m *monomorphicFunctionSymbol) Copy() Symbol {
	cp := *m
	return &cp
}
