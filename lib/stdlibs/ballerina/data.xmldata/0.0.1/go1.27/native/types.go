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

package native

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// typeKind is the jBallerina type tag a type is dispatched on.
type typeKind int

const (
	kindUnsupported typeKind = iota
	kindRecord
	kindMap
	kindArray
	kindJSON
	kindAnydata
	kindUnion
	kindString
	kindInt
	kindByte
	kindIntSubtype
	kindFloat
	kindDecimal
	kindBoolean
	kindNil
	kindFinite
	kindXML
	kindXMLElement
	kindXMLText
	kindXMLOther
)

// typeInfo is a classified type. For a record or map, mapping holds its atomic type; for an array,
// element and size describe it, with size -1 for an open array.
type typeInfo struct {
	kind    typeKind
	ty      semtypes.SemType
	mapping *semtypes.MappingAtomicType
	list    *semtypes.ListAtomicType
	element semtypes.SemType
	size    int
	members []semtypes.SemType
}

func (ti typeInfo) isXML() bool {
	switch ti.kind {
	case kindXML, kindXMLElement, kindXMLText, kindXMLOther:
		return true
	}
	return false
}

func (ti typeInfo) isNonNilablePrimitive() bool {
	switch ti.kind {
	case kindString, kindInt, kindFloat, kindDecimal, kindBoolean, kindByte:
		return true
	}
	return false
}

func (ti typeInfo) isAnydataOrJSON() bool {
	return ti.kind == kindAnydata || ti.kind == kindJSON
}

func (ti typeInfo) isStringValueAssignable() bool {
	return ti.kind == kindString || ti.isAnydataOrJSON()
}

func (ti typeInfo) isArrayValueAssignable() bool {
	return ti.kind == kindArray || ti.isAnydataOrJSON()
}

// scalarBasicTypes lists the scalar basic types in FromString's priority order, which is also the order union
// members are tried in, since runtime types do not keep their declaration order.
var scalarBasicTypes = []struct {
	ty   semtypes.SemType
	kind typeKind
}{
	{semtypes.Int, kindInt},
	{semtypes.Float, kindFloat},
	{semtypes.Decimal, kindDecimal},
	{semtypes.Nil, kindNil},
	{semtypes.Boolean, kindBoolean},
	{semtypes.String, kindString},
}

// typer answers the type questions the jBallerina implementation answers through its runtime type API.
type typer struct {
	tc         semtypes.Context
	cache      *typeCache
	jsonTy     semtypes.SemType
	anydataTy  semtypes.SemType
	root       semtypes.SemType
	rootAnnots typeAnnotations
}

func newTyper(cache *typeCache, tc semtypes.Context, root semtypes.SemType, rootAnnots typeAnnotations) *typer {
	return &typer{
		tc:         tc,
		cache:      cache,
		jsonTy:     semtypes.CreateJSON(tc),
		anydataTy:  semtypes.CreateAnydata(tc),
		root:       root,
		rootAnnots: rootAnnots,
	}
}

// typeCache holds, per runtime, the classification of each structured type and the map and array types
// defined over each constraint. Classifying a mapping or list type and defining a type both mint runtime
// atoms that stay live, so doing either per parsed value makes every later parse slower.
type typeCache struct {
	mu       sync.Mutex
	interner *semtypes.SemTypeInterner
	infos    typeKeyed[typeInfo]
	maps     typeKeyed[semtypes.SemType]
	arrays   typeKeyed[semtypes.SemType]
}

// typeKeyed keys interner-recognisable types by handle; the rest, such as byte or a union with a singleton
// member, are matched by type equality once per representation and then found by instance.
type typeKeyed[V any] struct {
	byHandle   map[semtypes.InternHandle]V
	byInstance map[semtypes.InstanceKey]V
	others     []typeEntry[V]
}

type typeEntry[V any] struct {
	key   semtypes.SemType
	value V
}

func newTypeKeyed[V any]() typeKeyed[V] {
	return typeKeyed[V]{
		byHandle:   make(map[semtypes.InternHandle]V),
		byInstance: make(map[semtypes.InstanceKey]V),
	}
}

func newTypeCache() *typeCache {
	return &typeCache{
		interner: semtypes.NewSemtypeInterner(),
		infos:    newTypeKeyed[typeInfo](),
		maps:     newTypeKeyed[semtypes.SemType](),
		arrays:   newTypeKeyed[semtypes.SemType](),
	}
}

func getOrCompute[V any](c *typeCache, tc semtypes.Context, entries *typeKeyed[V], key semtypes.SemType,
	compute func(semtypes.SemType) V) V {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !internable(key) {
		instance := key.InstanceKey()
		if value, ok := entries.byInstance[instance]; ok {
			return value
		}
		value := lookupOrCompute(tc, entries, key, compute)
		entries.byInstance[instance] = value
		return value
	}
	handle := c.interner.Intern(key)
	if value, ok := entries.byHandle[handle]; ok {
		return value
	}
	value := compute(key)
	entries.byHandle[handle] = value
	return value
}

func lookupOrCompute[V any](tc semtypes.Context, entries *typeKeyed[V], key semtypes.SemType,
	compute func(semtypes.SemType) V) V {
	for _, e := range entries.others {
		if semtypes.IsSameType(tc, e.key, key) {
			return e.value
		}
	}
	value := compute(key)
	entries.others = append(entries.others, typeEntry[V]{key: key, value: value})
	return value
}

var valueBasicTypes = []semtypes.SemType{semtypes.Int, semtypes.Float, semtypes.Decimal, semtypes.String}

// internable reports whether the interner can recognise a copy of ty: it never matches a proper subtype of
// a value type.
func internable(ty semtypes.SemType) bool {
	basic := semtypes.WidenToBasicTypes(ty)
	for _, valueTy := range valueBasicTypes {
		if semtypes.ContainsBasicType(basic, valueTy) && !semtypes.IsSubtypeSimple(valueTy, ty) {
			return false
		}
	}
	return true
}

// classifyMints reports whether classifying ty mints atoms, which only its mapping and list parts do.
func classifyMints(ty semtypes.SemType) bool {
	basic := semtypes.WidenToBasicTypes(ty)
	return semtypes.ContainsBasicType(basic, semtypes.Mapping) || semtypes.ContainsBasicType(basic, semtypes.List)
}

func (t *typer) classify(ty semtypes.SemType) typeInfo {
	if !classifyMints(ty) {
		return t.classifyType(ty)
	}
	return getOrCompute(t.cache, t.tc, &t.cache.infos, ty, t.classifyType)
}

func (t *typer) classifyType(ty semtypes.SemType) typeInfo {
	if semtypes.IsSameType(t.tc, ty, t.jsonTy) {
		return typeInfo{kind: kindJSON, ty: ty}
	}
	if semtypes.IsSameType(t.tc, ty, t.anydataTy) {
		return typeInfo{kind: kindAnydata, ty: ty}
	}
	parts := t.decompose(ty)
	switch len(parts) {
	case 0:
		return typeInfo{kind: kindUnsupported, ty: ty}
	case 1:
		return t.classifyMember(parts[0])
	default:
		return typeInfo{kind: kindUnion, ty: ty, members: parts}
	}
}

func (t *typer) classifyMember(ty semtypes.SemType) typeInfo {
	switch {
	case semtypes.IsSubtypeSimple(ty, semtypes.Mapping):
		atomic := semtypes.ToMappingAtomicType(t.tc, ty)
		if atomic == nil {
			return typeInfo{kind: kindUnsupported, ty: ty}
		}
		// `record {}` and `map<anydata>` are the same type at runtime; it is taken as the record.
		kind := kindMap
		if len(atomic.FieldNames()) > 0 || semtypes.IsNever(atomic.Rest()) ||
			semtypes.IsSameType(t.tc, atomic.Rest(), t.anydataTy) {
			kind = kindRecord
		}
		return typeInfo{kind: kind, ty: ty, mapping: atomic}
	case semtypes.IsSubtypeSimple(ty, semtypes.List):
		return t.classifyList(ty)
	case semtypes.IsSubtypeSimple(ty, semtypes.XML):
		return t.classifyXML(ty)
	}
	for _, basic := range scalarBasicTypes {
		if !semtypes.IsSubtypeSimple(ty, basic.ty) {
			continue
		}
		switch {
		case semtypes.IsSameType(t.tc, ty, basic.ty):
			return typeInfo{kind: basic.kind, ty: ty}
		case basic.kind == kindString && semtypes.IsSameType(t.tc, ty, semtypes.Char):
			return typeInfo{kind: kindString, ty: ty}
		case basic.kind == kindInt && semtypes.IsSameType(t.tc, ty, semtypes.Byte):
			return typeInfo{kind: kindByte, ty: ty}
		case basic.kind == kindInt && strings.HasPrefix(semtypes.ToString(t.tc, ty), "int:"):
			return typeInfo{kind: kindIntSubtype, ty: ty}
		default:
			return typeInfo{kind: kindFinite, ty: ty}
		}
	}
	return typeInfo{kind: kindUnsupported, ty: ty}
}

// classifyList accepts only array types: jBallerina's XML conversions have no tuple support.
func (t *typer) classifyList(ty semtypes.SemType) typeInfo {
	atomic := semtypes.ToListAtomicType(t.tc.Env(), ty)
	if atomic == nil {
		return typeInfo{kind: kindUnsupported, ty: ty}
	}
	rest := atomic.Rest()
	fixed := atomic.FixedLength()
	if !semtypes.IsNever(rest) {
		for i := 0; i < fixed; i++ {
			if !semtypes.IsSameType(t.tc, atomic.MemberAtInnerVal(i), rest) {
				return typeInfo{kind: kindUnsupported, ty: ty}
			}
		}
		return typeInfo{kind: kindArray, ty: ty, list: atomic, element: rest, size: -1}
	}
	if fixed == 0 {
		return typeInfo{kind: kindUnsupported, ty: ty}
	}
	element := atomic.MemberAtInnerVal(0)
	for i := 1; i < fixed; i++ {
		if !semtypes.IsSameType(t.tc, atomic.MemberAtInnerVal(i), element) {
			return typeInfo{kind: kindUnsupported, ty: ty}
		}
	}
	return typeInfo{kind: kindArray, ty: ty, list: atomic, element: element, size: fixed}
}

func (t *typer) classifyXML(ty semtypes.SemType) typeInfo {
	switch {
	case semtypes.IsSameType(t.tc, ty, semtypes.XML):
		return typeInfo{kind: kindXML, ty: ty}
	case semtypes.IsSubtype(t.tc, ty, semtypes.XMLElement):
		return typeInfo{kind: kindXMLElement, ty: ty}
	case semtypes.IsSubtype(t.tc, ty, semtypes.XMLText):
		return typeInfo{kind: kindXMLText, ty: ty}
	case semtypes.IsSubtype(t.tc, ty, semtypes.Union(semtypes.XMLComment, semtypes.XMLProcessingInstruction)):
		return typeInfo{kind: kindXMLOther, ty: ty}
	default:
		return typeInfo{kind: kindXML, ty: ty}
	}
}

// decompose splits ty into the members a union type tag would list.
func (t *typer) decompose(ty semtypes.SemType) []semtypes.SemType {
	var parts []semtypes.SemType
	if mapping := semtypes.Intersect(ty, semtypes.Mapping); !semtypes.IsEmpty(t.tc, mapping) {
		for _, alt := range semtypes.MappingAlternatives(t.tc, mapping) {
			parts = append(parts, alt.Type())
		}
	}
	if list := semtypes.Intersect(ty, semtypes.List); !semtypes.IsEmpty(t.tc, list) {
		for _, alt := range semtypes.ListAlternatives(t.tc, list) {
			parts = append(parts, alt.Type())
		}
	}
	covered := semtypes.Union(semtypes.Union(semtypes.Mapping, semtypes.List), semtypes.XML)
	if xmlPart := semtypes.Intersect(ty, semtypes.XML); !semtypes.IsEmpty(t.tc, xmlPart) {
		parts = append(parts, xmlPart)
	}
	for _, basic := range scalarBasicTypes {
		covered = semtypes.Union(covered, basic.ty)
		if member := semtypes.Intersect(ty, basic.ty); !semtypes.IsEmpty(t.tc, member) {
			parts = append(parts, member)
		}
	}
	if rest := semtypes.Diff(ty, covered); !semtypes.IsEmpty(t.tc, rest) {
		parts = append(parts, rest)
	}
	return parts
}

// isSimpleType is DataUtils.isSimpleType: whether a value of the type is never a structure.
func (t *typer) isSimpleType(ty semtypes.SemType) bool {
	ti := t.classify(ty)
	switch ti.kind {
	case kindJSON, kindAnydata, kindMap, kindRecord, kindXML, kindXMLElement, kindXMLText, kindXMLOther,
		kindUnsupported:
		return false
	case kindArray:
		return t.isSimpleType(ti.element)
	case kindUnion:
		for _, member := range ti.members {
			if !t.isSimpleType(member) {
				return false
			}
		}
	}
	return true
}

// containsUnionType is DataUtils.isContainsUnionType: a union of structures anywhere in the type makes
// jBallerina parse the document into an XML value and traverse that instead of streaming it.
func (t *typer) containsUnionType(ty semtypes.SemType) bool {
	return t.containsUnion(ty, map[string]bool{})
}

func (t *typer) containsUnion(ty semtypes.SemType, visiting map[string]bool) bool {
	key := semtypes.ToString(t.tc, ty)
	if visiting[key] {
		return false
	}
	visiting[key] = true
	ti := t.classify(ty)
	switch ti.kind {
	case kindUnion:
		for _, member := range ti.members {
			if !t.isSimpleType(member) {
				return true
			}
		}
	case kindArray:
		return t.containsUnion(ti.element, visiting)
	case kindMap:
		return t.containsUnion(ti.mapping.Rest(), visiting)
	case kindRecord:
		for _, name := range ti.mapping.FieldNames() {
			if t.containsUnion(ti.mapping.FieldInnerVal(name), visiting) {
				return true
			}
		}
		if rest := ti.mapping.Rest(); !semtypes.IsNever(rest) {
			return t.containsUnion(rest, visiting)
		}
	}
	return false
}

func (t *typer) isSupportedType(ty semtypes.SemType) bool {
	ti := t.classify(ty)
	switch ti.kind {
	case kindNil, kindInt, kindByte, kindFloat, kindDecimal, kindBoolean, kindString, kindRecord, kindMap,
		kindJSON, kindAnydata, kindFinite, kindXML, kindXMLElement, kindXMLText:
		return true
	case kindArray:
		return t.isSupportedType(ti.element)
	case kindUnion:
		for _, member := range ti.members {
			if t.isSupportedType(member) {
				return true
			}
		}
	}
	return false
}

func (t *typer) isSupportedTypeForAttributes(ty semtypes.SemType) bool {
	switch t.classify(ty).kind {
	case kindInt, kindIntSubtype, kindByte, kindString, kindFloat, kindBoolean, kindNil, kindDecimal, kindUnion,
		kindAnydata, kindJSON:
		return true
	}
	return false
}

func (t *typer) isNilable(ty semtypes.SemType) bool {
	return !semtypes.IsEmpty(t.tc, semtypes.Intersect(ty, semtypes.Nil))
}

func (t *typer) isReadonly(ty semtypes.SemType) bool {
	return semtypes.IsSubtype(t.tc, ty, semtypes.ValReadonly)
}

func (t *typer) contains(ty semtypes.SemType, v values.BalValue) bool {
	return semtypes.IsSubtype(t.tc, values.SemTypeForValue(v), ty)
}

func (t *typer) mapType(constraint semtypes.SemType) semtypes.SemType {
	return getOrCompute(t.cache, t.tc, &t.cache.maps, constraint, func(constraint semtypes.SemType) semtypes.SemType {
		md := semtypes.NewMappingDefinition()
		return md.Define(t.tc.Env(), nil, constraint)
	})
}

func (t *typer) arrayType(element semtypes.SemType) semtypes.SemType {
	return getOrCompute(t.cache, t.tc, &t.cache.arrays, element, func(element semtypes.SemType) semtypes.SemType {
		ld := semtypes.NewListDefinition()
		return ld.Define(t.tc.Env(), nil, semtypes.ListRest(element))
	})
}

// mapTypeFromConstraint is DataUtils.getMapTypeFromConstraintType.
func (t *typer) mapTypeFromConstraint(constraint semtypes.SemType) semtypes.SemType {
	ti := t.classify(constraint)
	switch ti.kind {
	case kindMap:
		return constraint
	case kindInt, kindFloat, kindString, kindBoolean, kindByte, kindDecimal, kindJSON, kindRecord, kindXML,
		kindXMLElement, kindXMLText, kindXMLOther, kindNil:
		return t.mapType(constraint)
	case kindArray:
		return t.mapType(ti.element)
	default:
		return t.mapType(t.anydataTy)
	}
}

// typeName renders ty the way the jBallerina runtime prints a type, as far as a structural type allows:
// declared type names are not available at runtime.
func (t *typer) typeName(ty semtypes.SemType) string {
	switch {
	case semtypes.IsSameType(t.tc, ty, t.jsonTy):
		return "json"
	case semtypes.IsSameType(t.tc, ty, t.anydataTy):
		return "anydata"
	case semtypes.IsSameType(t.tc, ty, semtypes.Nil):
		return "()"
	case semtypes.IsSameType(t.tc, ty, semtypes.Byte):
		return "byte"
	case semtypes.IsSameType(t.tc, ty, semtypes.Char):
		return "lang.string:Char"
	}
	ti := t.classify(ty)
	switch {
	case ti.kind == kindUnion && len(ti.members) == 2 && t.isNilable(ty):
		return t.typeName(semtypes.Diff(ty, semtypes.Nil)) + "?"
	case ti.kind == kindArray && ti.size == -1:
		return t.typeName(ti.element) + "[]"
	case ti.kind == kindArray:
		return fmt.Sprintf("%s[%d]", t.typeName(ti.element), ti.size)
	case ti.kind == kindMap:
		return "map<" + t.typeName(ti.mapping.Rest()) + ">"
	}
	name := semtypes.ToString(t.tc, ty)
	switch {
	case strings.HasPrefix(name, "int:"):
		return "lang." + name
	case (ti.kind == kindFinite || ti.kind == kindUnion) && strings.Contains(name, "|"):
		return "(" + name + ")"
	}
	return name
}
