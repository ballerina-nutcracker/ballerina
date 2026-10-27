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
	"slices"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// typeKind is the jBallerina type tag a target type is dispatched on.
type typeKind int

const (
	kindUnsupported typeKind = iota
	kindJSON
	kindAnydata
	kindRecord
	kindMap
	kindList
	kindUnion
	kindNil
	kindBoolean
	kindInt
	kindFloat
	kindDecimal
	kindString
	kindIntSubtype
	kindChar
	kindFinite
)

type typeInfo struct {
	kind    typeKind
	ty      semtypes.SemType
	mapping *semtypes.MappingAtomicType
	list    *semtypes.ListAtomicType
	members []semtypes.SemType
}

func (ti typeInfo) isExactBasic() bool {
	switch ti.kind {
	case kindNil, kindBoolean, kindInt, kindFloat, kindDecimal, kindString:
		return true
	}
	return false
}

func (ti typeInfo) isScalar() bool {
	return ti.isExactBasic() || ti.kind == kindIntSubtype || ti.kind == kindChar || ti.kind == kindFinite
}

var scalarBasicTypes = []struct {
	ty   semtypes.SemType
	kind typeKind
}{
	{semtypes.Nil, kindNil},
	{semtypes.Boolean, kindBoolean},
	{semtypes.Int, kindInt},
	{semtypes.Float, kindFloat},
	{semtypes.Decimal, kindDecimal},
	{semtypes.String, kindString},
}

type recordField struct {
	name     string
	ty       semtypes.SemType
	optional bool
}

// fieldSet holds the record fields that have not been seen in the JSON yet, keyed by their JSON key.
type fieldSet struct {
	byKey map[string]*recordField
	order []string
}

func (fs *fieldSet) remove(key string) *recordField {
	f, ok := fs.byKey[key]
	if !ok {
		return nil
	}
	delete(fs.byKey, key)
	return f
}

func (fs *fieldSet) remaining() []*recordField {
	result := make([]*recordField, 0, len(fs.byKey))
	for _, key := range fs.order {
		if f, ok := fs.byKey[key]; ok {
			result = append(result, f)
		}
	}
	return result
}

func emptyFieldSet() *fieldSet {
	return &fieldSet{byKey: make(map[string]*recordField)}
}

// typer answers the type questions the jBallerina implementation answers through its runtime type API.
type typer struct {
	tc            semtypes.Context
	cache         *typeCache
	jsonTy        semtypes.SemType
	anydataTy     semtypes.SemType
	jsonMapTy     semtypes.SemType
	jsonMap       *semtypes.MappingAtomicType
	jsonListTy    semtypes.SemType
	jsonList      *semtypes.ListAtomicType
	anydataMapTy  semtypes.SemType
	anydataMap    *semtypes.MappingAtomicType
	anydataListTy semtypes.SemType
	anydataList   *semtypes.ListAtomicType
	root          semtypes.SemType
	rootFieldKeys map[string]string
}

func newTyper(cache *typeCache, tc semtypes.Context, root semtypes.SemType, rootFieldKeys map[string]string) *typer {
	jsonTy := semtypes.CreateJSON(tc)
	jsonMapTy := semtypes.Intersect(jsonTy, semtypes.Mapping)
	jsonListTy := semtypes.Intersect(jsonTy, semtypes.List)
	anydataTy := semtypes.CreateAnydata(tc)
	anydataMapTy := semtypes.Intersect(anydataTy, semtypes.Mapping)
	anydataListTy := semtypes.Intersect(anydataTy, semtypes.List)
	return &typer{
		tc:            tc,
		cache:         cache,
		jsonTy:        jsonTy,
		anydataTy:     anydataTy,
		jsonMapTy:     jsonMapTy,
		jsonMap:       semtypes.ToMappingAtomicType(tc, jsonMapTy),
		jsonListTy:    jsonListTy,
		jsonList:      semtypes.ToListAtomicType(tc.Env(), jsonListTy),
		anydataMapTy:  anydataMapTy,
		anydataMap:    semtypes.ToMappingAtomicType(tc, anydataMapTy),
		anydataListTy: anydataListTy,
		anydataList:   semtypes.ToListAtomicType(tc.Env(), anydataListTy),
		root:          root,
		rootFieldKeys: rootFieldKeys,
	}
}

// typeCache holds the classification of each structured type a runtime has parsed into. Classifying a
// mapping or list type intersects its atoms, which mints runtime atoms, so it must happen once per type
// rather than once per value.
type typeCache struct {
	mu       sync.Mutex
	interner *semtypes.SemTypeInterner
	infos    map[semtypes.InternHandle]typeInfo
}

func newTypeCache() *typeCache {
	return &typeCache{interner: semtypes.NewSemtypeInterner(), infos: make(map[semtypes.InternHandle]typeInfo)}
}

func (c *typeCache) getOrClassify(ty semtypes.SemType, classify func(semtypes.SemType) typeInfo) typeInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	handle := c.interner.Intern(ty)
	if ti, ok := c.infos[handle]; ok {
		return ti
	}
	ti := classify(ty)
	c.infos[handle] = ti
	return ti
}

var valueBasicTypes = []semtypes.SemType{semtypes.Int, semtypes.Float, semtypes.Decimal, semtypes.String}

// cacheable reports whether ty is worth caching and the interner can recognise a copy of it: only mapping
// and list types are costly to classify, and the interner never matches a proper subtype of a value type.
func cacheable(ty semtypes.SemType) bool {
	basic := semtypes.WidenToBasicTypes(ty)
	if !semtypes.ContainsBasicType(basic, semtypes.Mapping) && !semtypes.ContainsBasicType(basic, semtypes.List) {
		return false
	}
	for _, valueTy := range valueBasicTypes {
		if semtypes.ContainsBasicType(basic, valueTy) && !semtypes.IsSubtypeSimple(valueTy, ty) {
			return false
		}
	}
	return true
}

func (t *typer) classify(ty semtypes.SemType) typeInfo {
	if !cacheable(ty) {
		return t.classifyType(ty)
	}
	return t.cache.getOrClassify(ty, t.classifyType)
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
	if semtypes.IsSubtypeSimple(ty, semtypes.Mapping) {
		atomic := semtypes.ToMappingAtomicType(t.tc, ty)
		if atomic == nil {
			return typeInfo{kind: kindUnsupported, ty: ty}
		}
		kind := kindMap
		if len(atomic.FieldNames()) > 0 || semtypes.IsNever(atomic.Rest()) {
			kind = kindRecord
		}
		return typeInfo{kind: kind, ty: ty, mapping: atomic}
	}
	if semtypes.IsSubtypeSimple(ty, semtypes.List) {
		atomic := semtypes.ToListAtomicType(t.tc.Env(), ty)
		if atomic == nil {
			return typeInfo{kind: kindUnsupported, ty: ty}
		}
		return typeInfo{kind: kindList, ty: ty, list: atomic}
	}
	for _, basic := range scalarBasicTypes {
		if !semtypes.IsSubtypeSimple(ty, basic.ty) {
			continue
		}
		switch {
		case semtypes.IsSameType(t.tc, ty, basic.ty):
			return typeInfo{kind: basic.kind, ty: ty}
		case basic.kind == kindInt && t.isIntWidthType(ty):
			return typeInfo{kind: kindIntSubtype, ty: ty}
		case basic.kind == kindString && semtypes.IsSameType(t.tc, ty, semtypes.Char):
			return typeInfo{kind: kindChar, ty: ty}
		default:
			return typeInfo{kind: kindFinite, ty: ty}
		}
	}
	return typeInfo{kind: kindUnsupported, ty: ty}
}

var intWidthTypes = []semtypes.SemType{
	semtypes.SignedInt8, semtypes.SignedInt16, semtypes.SignedInt32,
	semtypes.UnsignedInt8, semtypes.UnsignedInt16, semtypes.UnsignedInt32,
}

func (t *typer) isIntWidthType(ty semtypes.SemType) bool {
	return slices.ContainsFunc(intWidthTypes, func(width semtypes.SemType) bool {
		return semtypes.IsSameType(t.tc, ty, width)
	})
}

// decompose splits ty into the members a union type tag would list: one per mapping or list
// alternative, one per scalar basic type, and one for whatever is left (xml, table, ...).
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
	covered := semtypes.Union(semtypes.Mapping, semtypes.List)
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

func (t *typer) isNilable(ty semtypes.SemType) bool {
	return !semtypes.IsEmpty(t.tc, semtypes.Intersect(ty, semtypes.Nil))
}

func (t *typer) isReadonly(ty semtypes.SemType) bool {
	return semtypes.IsSubtype(t.tc, ty, semtypes.ValReadonly)
}

func (t *typer) contains(ty semtypes.SemType, v values.BalValue) bool {
	return semtypes.IsSubtype(t.tc, values.SemTypeForValue(v), ty)
}

// typeName renders ty the way the jBallerina runtime prints a type in an error message, as far as a
// structural type allows: declared type names are not available at runtime.
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
	if ti.kind == kindUnion && len(ti.members) == 2 && t.isNilable(ty) {
		return t.typeName(semtypes.Diff(ty, semtypes.Nil)) + "?"
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

// recordFields lists the fields of a record keyed by their JSON key. Only the target type's own fields can
// carry a `@jsondata:Name` override, since annotations of nested types do not reach native code.
func (t *typer) recordFields(ti typeInfo) (*fieldSet, error) {
	fs := emptyFieldSet()
	renames := t.rootFieldKeys
	if !semtypes.IsSameType(t.tc, ti.ty, t.root) {
		renames = nil
	}
	for _, name := range ti.mapping.FieldNames() {
		key := name
		if renamed, ok := renames[name]; ok {
			key = renamed
		}
		if _, exists := fs.byKey[key]; exists {
			return nil, duplicateFieldError(key)
		}
		fs.byKey[key] = &recordField{name: name, ty: ti.mapping.FieldInnerVal(name), optional: ti.mapping.IsOptional(t.tc, name)}
		fs.order = append(fs.order, key)
	}
	return fs, nil
}

// restType returns the type of the rest fields of a mapping, or false for a closed record.
func restType(ti typeInfo) (semtypes.SemType, bool) {
	rest := ti.mapping.Rest()
	return rest, !semtypes.IsNever(rest)
}

type mappingBuilder struct {
	ty      semtypes.SemType
	atomic  *semtypes.MappingAtomicType
	entries []values.MapEntry
}

func (b *mappingBuilder) put(key string, v values.BalValue) {
	b.entries = append(b.entries, values.MapEntry{Key: key, Value: v})
}

func (b *mappingBuilder) build(t *typer) *values.Map {
	return values.NewMap(b.ty, b.atomic, t.isReadonly(b.ty), b.entries)
}

type listBuilder struct {
	ty     semtypes.SemType
	atomic *semtypes.ListAtomicType
	items  []values.BalValue
}

func (b *listBuilder) set(index int, v values.BalValue) {
	for len(b.items) <= index {
		b.items = append(b.items, nil)
	}
	b.items[index] = v
}

// accepts reports whether index is within the list type, so a value at that index is kept rather than
// projected away.
func (b *listBuilder) accepts(index int) bool {
	return index < b.atomic.FixedLength() || !semtypes.IsNever(b.atomic.Rest())
}

// build pads a list shorter than its fixed length with filler values, as creating a jBallerina array or
// tuple of that type does.
func (b *listBuilder) build(t *typer) (*values.List, error) {
	for i := len(b.items); i < b.atomic.FixedLength(); i++ {
		filler, ok := values.FillerValue(t.tc, b.atomic.MemberAtInnerVal(i))
		if !ok {
			return nil, arraySizeMismatchError()
		}
		b.items = append(b.items, filler)
	}
	restFiller, _ := values.FillerFactoryFor(t.tc, b.atomic.Rest())
	return values.NewList(b.ty, b.atomic, t.isReadonly(b.ty), restFiller, len(b.items), b.items), nil
}

func (t *typer) newMappingBuilder(ti typeInfo) *mappingBuilder {
	return &mappingBuilder{ty: ti.ty, atomic: ti.mapping}
}

func (t *typer) newJSONMappingBuilder() *mappingBuilder {
	return &mappingBuilder{ty: t.jsonMapTy, atomic: t.jsonMap}
}

func (t *typer) newListBuilder(ti typeInfo) *listBuilder {
	return &listBuilder{ty: ti.ty, atomic: ti.list}
}

func (t *typer) newJSONListBuilder() *listBuilder {
	return &listBuilder{ty: t.jsonListTy, atomic: t.jsonList}
}

func (t *typer) newAnydataMappingBuilder() *mappingBuilder {
	return &mappingBuilder{ty: t.anydataMapTy, atomic: t.anydataMap}
}

func (t *typer) newAnydataListBuilder() *listBuilder {
	return &listBuilder{ty: t.anydataListTy, atomic: t.anydataList}
}
