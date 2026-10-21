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
	"strconv"
	"strings"

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
	kindArray
	kindTuple
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

// isBasic is CsvUtils.isBasicType: the member types transform accepts inside a union.
func (ti typeInfo) isBasic() bool {
	switch ti.kind {
	case kindRecord, kindMap, kindArray, kindTuple, kindFinite, kindUnsupported:
		return false
	}
	return true
}

func (ti typeInfo) isList() bool {
	return ti.kind == kindArray || ti.kind == kindTuple
}

func (ti typeInfo) isMapping() bool {
	return ti.kind == kindRecord || ti.kind == kindMap
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

// typer answers the type questions the jBallerina implementation answers through its runtime type API.
type typer struct {
	tc            semtypes.Context
	jsonTy        semtypes.SemType
	anydataTy     semtypes.SemType
	stringMapTy   semtypes.SemType
	stringMap     *semtypes.MappingAtomicType
	stringListTy  semtypes.SemType
	stringList    *semtypes.ListAtomicType
	anydataListTy semtypes.SemType
	anydataList   *semtypes.ListAtomicType
}

func newTyper(tc semtypes.Context) *typer {
	env := tc.Env()
	stringMapTy := defineMap(env, nil, semtypes.String)
	stringListTy := defineArray(env, semtypes.String)
	anydataTy := semtypes.CreateAnydata(tc)
	anydataListTy := defineArray(env, anydataTy)
	return &typer{
		tc:            tc,
		jsonTy:        semtypes.CreateJSON(tc),
		anydataTy:     anydataTy,
		stringMapTy:   stringMapTy,
		stringMap:     semtypes.ToMappingAtomicType(tc, stringMapTy),
		stringListTy:  stringListTy,
		stringList:    semtypes.ToListAtomicType(env, stringListTy),
		anydataListTy: anydataListTy,
		anydataList:   semtypes.ToListAtomicType(env, anydataListTy),
	}
}

func (t *typer) classify(ty semtypes.SemType) typeInfo {
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
		if len(atomic.FieldNames()) > 0 || semtypes.IsNever(mappingRest(atomic)) {
			kind = kindRecord
		}
		return typeInfo{kind: kind, ty: ty, mapping: atomic}
	}
	if semtypes.IsSubtypeSimple(ty, semtypes.List) {
		atomic := semtypes.ToListAtomicType(t.tc.Env(), ty)
		if atomic == nil {
			return typeInfo{kind: kindUnsupported, ty: ty}
		}
		return typeInfo{kind: t.listKind(atomic), ty: ty, list: atomic}
	}
	for _, basic := range scalarBasicTypes {
		if !semtypes.IsSubtypeSimple(ty, basic.ty) {
			continue
		}
		switch {
		case semtypes.IsSameType(t.tc, ty, basic.ty):
			return typeInfo{kind: basic.kind, ty: ty}
		case basic.kind == kindInt && strings.HasPrefix(semtypes.ToString(t.tc, ty), "int:"):
			return typeInfo{kind: kindIntSubtype, ty: ty}
		case basic.kind == kindString && semtypes.IsSameType(t.tc, ty, semtypes.Char):
			return typeInfo{kind: kindChar, ty: ty}
		default:
			return typeInfo{kind: kindFinite, ty: ty}
		}
	}
	return typeInfo{kind: kindUnsupported, ty: ty}
}

// listKind tells an array from a tuple. Types are structural here, so a tuple whose members all share
// one type, such as [int, int], is indistinguishable from the closed array int[2] and is treated as one.
func (t *typer) listKind(atomic *semtypes.ListAtomicType) typeKind {
	n := atomic.FixedLength()
	if n == 0 {
		return kindArray
	}
	if !semtypes.IsNever(atomic.Rest()) {
		return kindTuple
	}
	first := atomic.MemberAtInnerVal(0)
	for i := 1; i < n; i++ {
		if !semtypes.IsSameType(t.tc, first, atomic.MemberAtInnerVal(i)) {
			return kindTuple
		}
	}
	return kindArray
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
	if nonNil := semtypes.Diff(ty, semtypes.Nil); t.isNilable(ty) {
		name := t.typeName(nonNil)
		if strings.Contains(name, "|") && !strings.HasPrefix(name, "(") {
			name = "(" + name + ")"
		}
		return name + "?"
	}
	if ti := t.classify(ty); ti.kind == kindArray {
		elemTy := t.rowShapeOf(ti).rest
		elem := t.typeName(elemTy)
		if t.classify(elemTy).kind == kindUnion && !strings.HasSuffix(elem, "?") {
			elem = "(" + elem + ")"
		}
		if size := ti.list.FixedLength(); size > 0 && semtypes.IsNever(ti.list.Rest()) {
			return elem + "[" + strconv.Itoa(size) + "]"
		}
		return elem + "[]"
	}
	name := semtypes.ToString(t.tc, ty)
	switch {
	case strings.HasPrefix(name, "int:"):
		return "lang." + name
	case t.classify(ty).kind == kindFinite && strings.Contains(name, "|"):
		return "(" + name + ")"
	}
	return name
}

// rowShape describes the type of one row of the result: a record, map, array or tuple.
type rowShape struct {
	typeInfo
	readonly bool
	// fields holds a record's declared fields by name, and fieldOrder their names in a fixed order.
	fields     map[string]*recordField
	fieldOrder []string
	// rest is the record rest field type, the tuple rest type, or the array element type; hasRest
	// reports whether it exists.
	rest    semtypes.SemType
	hasRest bool
	// size is the length of a closed array or the member count of a tuple; -1 for an open array.
	size int
}

type recordField struct {
	name     string
	ty       semtypes.SemType
	optional bool
	nilable  bool
}

func (t *typer) rowShapeOf(ti typeInfo) rowShape {
	shape := rowShape{typeInfo: ti, readonly: t.isReadonly(ti.ty), size: -1}
	switch ti.kind {
	case kindRecord, kindMap:
		shape.rest = mappingRest(ti.mapping)
		shape.hasRest = !semtypes.IsNever(shape.rest)
		shape.fields = make(map[string]*recordField)
		for _, name := range ti.mapping.FieldNames() {
			ty := ti.mapping.FieldInnerVal(name)
			shape.fields[name] = &recordField{
				name:     name,
				ty:       ty,
				optional: ti.mapping.IsOptional(t.tc, name),
				nilable:  t.isNilable(ty),
			}
			shape.fieldOrder = append(shape.fieldOrder, name)
		}
	case kindArray:
		if n := ti.list.FixedLength(); n > 0 && semtypes.IsNever(ti.list.Rest()) {
			shape.size = n
			shape.rest = ti.list.MemberAtInnerVal(0)
		} else {
			shape.rest = ti.list.Rest()
		}
		shape.hasRest = true
	case kindTuple:
		shape.size = ti.list.FixedLength()
		shape.rest = ti.list.Rest()
		shape.hasRest = !semtypes.IsNever(shape.rest)
	}
	return shape
}

// memberType is the declared type at index of a tuple, or the element type of an array.
func (s *rowShape) memberType(index int) (semtypes.SemType, bool) {
	if s.kind == kindArray {
		if s.size != -1 && index >= s.size {
			return semtypes.Never, false
		}
		return s.rest, true
	}
	if index < s.size {
		return s.list.MemberAtInnerVal(index), true
	}
	return s.rest, s.hasRest
}

// remainingFields copies the declared fields of a record, keyed by field name: the fields not yet
// matched with a header or key.
func (s *rowShape) remainingFields() map[string]*recordField {
	remaining := make(map[string]*recordField, len(s.fields))
	for name, f := range s.fields {
		remaining[name] = f
	}
	return remaining
}

// rowBuilder accumulates one row value.
type rowBuilder struct {
	shape    *rowShape
	declared []values.MapEntry
	rest     []values.MapEntry
	items    []values.BalValue
}

func newRowBuilder(shape *rowShape) *rowBuilder {
	return &rowBuilder{shape: shape}
}

func (b *rowBuilder) put(key string, v values.BalValue) {
	if _, declared := b.shape.fields[key]; declared {
		b.declared = upsert(b.declared, key, v)
		return
	}
	b.rest = upsert(b.rest, key, v)
}

func upsert(entries []values.MapEntry, key string, v values.BalValue) []values.MapEntry {
	for i := range entries {
		if entries[i].Key == key {
			entries[i].Value = v
			return entries
		}
	}
	return append(entries, values.MapEntry{Key: key, Value: v})
}

func (b *rowBuilder) set(index int, v values.BalValue) {
	for len(b.items) <= index {
		b.items = append(b.items, nil)
	}
	b.items[index] = v
}

func (b *rowBuilder) has(key string) bool {
	for _, e := range b.declared {
		if e.Key == key {
			return true
		}
	}
	return false
}

// build creates the row value. As jBallerina's record creation does, a required field that never
// received a value is present with value `()`.
func (b *rowBuilder) build(t *typer) (values.BalValue, error) {
	s := b.shape
	if s.isMapping() {
		entries := b.declared
		for _, name := range s.fieldOrder {
			if f := s.fields[name]; !f.optional && !b.has(name) {
				entries = append(entries, values.MapEntry{Key: name, Value: nil})
			}
		}
		entries = append(entries, b.rest...)
		return values.NewMap(s.ty, s.mapping, s.readonly, entries), nil
	}
	return buildList(t, s.ty, s.list, s.readonly, b.items)
}

// buildList pads a list shorter than its fixed length with filler values, as creating a jBallerina
// array or tuple of that type does.
func buildList(t *typer, ty semtypes.SemType, atomic *semtypes.ListAtomicType, readonly bool,
	items []values.BalValue) (*values.List, error) {
	for i := len(items); i < atomic.FixedLength(); i++ {
		filler, ok := values.FillerValue(t.tc, atomic.MemberAtInnerVal(i))
		if !ok {
			return nil, newCsvError("array size is not compatible with the expected size")
		}
		items = append(items, filler)
	}
	restFiller, _ := values.FillerFactoryFor(t.tc, atomic.Rest())
	return values.NewList(ty, atomic, readonly, restFiller, len(items), items), nil
}

func defineArray(env semtypes.Env, elem semtypes.SemType) semtypes.SemType {
	ld := semtypes.NewListDefinition()
	return ld.Define(env, nil, semtypes.ListRest(elem))
}

func defineMap(env semtypes.Env, fields []semtypes.Field, rest semtypes.SemType) semtypes.SemType {
	md := semtypes.NewMappingDefinition()
	return md.Define(env, fields, rest)
}

// mappingRest is the rest field type of a mapping: FieldInnerVal falls back to it for a name that is not
// a declared field.
func mappingRest(atomic *semtypes.MappingAtomicType) semtypes.SemType {
	names := atomic.FieldNames()
	undeclared := "\x00"
	for slices.Contains(names, undeclared) {
		undeclared += "\x00"
	}
	return atomic.FieldInnerVal(undeclared)
}
