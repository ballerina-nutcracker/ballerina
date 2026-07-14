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

package semtypes

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"ballerina-lang-go/decimal"
)

type TypePool struct {
	tys      []SemType
	memo     map[InternHandle]TypePoolIndex
	interner *SemtypeInterner
}

func NewTypePool() *TypePool {
	return &TypePool{
		memo:     make(map[InternHandle]TypePoolIndex),
		interner: NewSemtypeInterner(),
	}
}

// TypePoolIndex represent handle to the [TypePool]
// If first bit is set or value is 0 it is a inline type (used to represent simple basic types)
// else it is an TypePoolIndex + 1 to the typePool
type TypePoolIndex int32

const indexMask = (1 << 31) - 1

func (pool *TypePool) Get(i TypePoolIndex) SemType {
	if i <= 0 {
		bits := i & indexMask
		return basicTypeBitSet(bits).semType()
	}
	return pool.tys[i-1]
}

func (pool *TypePool) Put(ty SemType) TypePoolIndex {
	if ty.some() == 0 {
		return TypePoolIndex(ty.all() | 1<<31)
	}
	handle := pool.interner.Intern(ty)
	if cached, ok := pool.memo[handle]; ok {
		return cached
	}
	id := len(pool.tys) + 1
	pool.tys = append(pool.tys, ty)
	pool.memo[handle] = TypePoolIndex(id)
	return TypePoolIndex(id)
}

func (pool *TypePool) PutObjectDefinition(ty SemType) TypePoolIndex {
	return pool.Put(stripObjectDistinctAtoms(ty))
}

func (pool *TypePool) PutErrorDefinition(ty SemType) TypePoolIndex {
	return pool.Put(stripErrorDistinctAtoms(ty))
}

func fromTypePool(pool *TypePool, env Env) binaryPool {
	bp := binaryPool{}
	cx := ContextFrom(env)
	sc := newBddSerializationContext(pool, cx, &bp)
	for i := 0; i < len(pool.tys); i++ {
		ty := pool.tys[i]
		subtypes := unpack(ty)
		start := uint32(len(bp.subtypeData))
		for _, bs := range subtypes {
			var entry subtypeDataEntry
			switch data := bs.SubtypeData.(type) {
			case intSubtype:
				entry = subtypeDataEntry{kind: intSubtypeData, index: uint32(len(bp.intSubtypes))}
				bp.intSubtypes = append(bp.intSubtypes, fromIntSubtype(&data))
			case booleanSubtype:
				entry = subtypeDataEntry{kind: booleanSubtypeData, index: uint32(len(bp.booleanSubtype))}
				bp.booleanSubtype = append(bp.booleanSubtype, fromBooleanSubtype(&data))
			case floatSubtype:
				entry = subtypeDataEntry{kind: floatSubtypeData, index: uint32(len(bp.floatSubtype))}
				bp.floatSubtype = append(bp.floatSubtype, fromFloatSubtype(&data))
			case decimalSubtype:
				entry = subtypeDataEntry{kind: decimalSubtypeData, index: uint32(len(bp.decimalSubtype))}
				bp.decimalSubtype = append(bp.decimalSubtype, fromDecimalSubtype(&data))
			case stringSubtype:
				entry = subtypeDataEntry{kind: stringSubtypeData, index: uint32(len(bp.stringSubtype))}
				bp.stringSubtype = append(bp.stringSubtype, fromStringSubtype(&data))
			case *xmlSubtype:
				entry = subtypeDataEntry{kind: xmlSubtypeData, index: uint32(len(bp.xmlSubtypes))}
				bp.xmlSubtypes = append(bp.xmlSubtypes, sc.serializeXmlSubtype(data))
			case Bdd:
				switch bs.BasicTypeCode {
				case BTList:
					entry = subtypeDataEntry{kind: listBddSubtypeData, index: uint32(len(bp.listBdds))}
					bp.listBdds = append(bp.listBdds, sc.serializeListBdd(data))
				case BTMapping:
					entry = subtypeDataEntry{kind: mappingBddSubtypeData, index: uint32(len(bp.mappingBdds))}
					bp.mappingBdds = append(bp.mappingBdds, sc.serializeMappingBdd(data))
				case BTFunction:
					entry = subtypeDataEntry{kind: functionBddSubtypeData, index: uint32(len(bp.functionBdds))}
					bp.functionBdds = append(bp.functionBdds, sc.serializeFunctionBdd(data))
				case BTTypeDesc:
					entry = subtypeDataEntry{kind: typedescBddSubtypeData, index: uint32(len(bp.typedescBdds))}
					bp.typedescBdds = append(bp.typedescBdds, sc.serializeMappingBdd(data))
				case BTError:
					entry = subtypeDataEntry{kind: errorBddSubtypeData, index: uint32(len(bp.errorBdds))}
					bp.errorBdds = append(bp.errorBdds, sc.serializeMappingBdd(data))
				case BTTable:
					entry = subtypeDataEntry{kind: tableBddSubtypeData, index: uint32(len(bp.tableBdds))}
					bp.tableBdds = append(bp.tableBdds, sc.serializeListBdd(data))
				case BTObject:
					entry = subtypeDataEntry{kind: objectBddSubtypeData, index: uint32(len(bp.objectBdds))}
					bp.objectBdds = append(bp.objectBdds, sc.serializeMappingBdd(data))
				case BTStream:
					entry = subtypeDataEntry{kind: streamBddSubtypeData, index: uint32(len(bp.streamBdds))}
					bp.streamBdds = append(bp.streamBdds, sc.serializeListBdd(data))
				default:
					panic(fmt.Sprintf("unsupported BDD basic type code: %v", bs.BasicTypeCode))
				}
			default:
				panic(fmt.Sprintf("unexpected subtype data type: %T (basic type code: %v)", bs.SubtypeData, bs.BasicTypeCode))
			}
			bp.subtypeData = append(bp.subtypeData, entry)
		}
		end := uint32(len(bp.subtypeData))
		bp.types = append(bp.types, typeEntry{
			all:              uint32(ty.all()),
			some:             uint32(ty.some()),
			subtypeDataStart: start,
			subtypeDataEnd:   end,
		})
	}
	bp.nIntSubtypes = uint32(len(bp.intSubtypes))
	bp.nBooleanSubtypes = uint32(len(bp.booleanSubtype))
	bp.nFloatSubtypes = uint32(len(bp.floatSubtype))
	bp.nDecimalSubtypes = uint32(len(bp.decimalSubtype))
	bp.nStringSubtypes = uint32(len(bp.stringSubtype))
	bp.nListBdds = uint32(len(bp.listBdds))
	bp.nMappingBdds = uint32(len(bp.mappingBdds))
	bp.nFunctionBdds = uint32(len(bp.functionBdds))
	bp.nTypedescBdds = uint32(len(bp.typedescBdds))
	bp.nErrorBdds = uint32(len(bp.errorBdds))
	bp.nTableBdds = uint32(len(bp.tableBdds))
	bp.nObjectBdds = uint32(len(bp.objectBdds))
	bp.nStreamBdds = uint32(len(bp.streamBdds))
	bp.nXmlAtomicTypes = uint32(len(bp.xmlAtomicTypes))
	bp.nXmlSubtypes = uint32(len(bp.xmlSubtypes))
	bp.nListAtomicTypes = uint32(len(bp.listAtomicTypes))
	bp.nMappingAtomicTypes = uint32(len(bp.mappingAtomicTypes))
	bp.nFunctionAtomicTypes = uint32(len(bp.functionAtomicTypes))
	return bp
}

func toTypePool(bp binaryPool, env Env) *TypePool {
	pool := &TypePool{
		memo:     make(map[InternHandle]TypePoolIndex),
		interner: NewSemtypeInterner(),
		tys:      make([]SemType, len(bp.types)),
	}
	dc := newBddDeserializationContext(pool, env, &bp)
	for i := range bp.types {
		dc.deserializeType(i)
	}
	return pool
}

func MarshalTypePool(pool *TypePool, env Env) []byte {
	bp := fromTypePool(pool, env)
	buf := &bytes.Buffer{}

	write(buf, bp.nIntSubtypes)
	write(buf, bp.nBooleanSubtypes)
	write(buf, bp.nFloatSubtypes)
	write(buf, bp.nDecimalSubtypes)
	write(buf, bp.nStringSubtypes)

	for _, entry := range bp.intSubtypes {
		marshalIntSubtype(buf, entry)
	}
	for _, entry := range bp.booleanSubtype {
		marshalBooleanSubtype(buf, entry)
	}
	for _, entry := range bp.floatSubtype {
		marshalFloatSubtype(buf, entry)
	}
	for _, entry := range bp.decimalSubtype {
		marshalDecimalSubtype(buf, entry)
	}
	for _, entry := range bp.stringSubtype {
		marshalStringSubtype(buf, entry)
	}

	write(buf, bp.nListBdds)
	write(buf, bp.nMappingBdds)
	write(buf, bp.nFunctionBdds)
	write(buf, bp.nTypedescBdds)
	write(buf, bp.nErrorBdds)
	write(buf, bp.nTableBdds)
	write(buf, bp.nObjectBdds)
	write(buf, bp.nStreamBdds)
	for _, entry := range bp.listBdds {
		marshalBddDnf(buf, entry)
	}
	for _, entry := range bp.mappingBdds {
		marshalBddDnf(buf, entry)
	}
	for _, entry := range bp.functionBdds {
		marshalBddDnf(buf, entry)
	}
	for _, entry := range bp.typedescBdds {
		marshalBddDnf(buf, entry)
	}
	for _, entry := range bp.errorBdds {
		marshalBddDnf(buf, entry)
	}
	for _, entry := range bp.tableBdds {
		marshalBddDnf(buf, entry)
	}
	for _, entry := range bp.objectBdds {
		marshalBddDnf(buf, entry)
	}
	for _, entry := range bp.streamBdds {
		marshalBddDnf(buf, entry)
	}

	write(buf, bp.nListAtomicTypes)
	write(buf, bp.nMappingAtomicTypes)
	write(buf, bp.nFunctionAtomicTypes)
	write(buf, bp.nXmlAtomicTypes)
	for _, entry := range bp.listAtomicTypes {
		marshalListAtomicType(buf, entry)
	}
	for _, entry := range bp.mappingAtomicTypes {
		marshalMappingAtomicType(buf, entry)
	}
	for _, entry := range bp.functionAtomicTypes {
		marshalFunctionAtomicType(buf, entry)
	}
	for _, entry := range bp.xmlAtomicTypes {
		write(buf, entry.index)
	}

	write(buf, bp.nXmlSubtypes)
	for _, entry := range bp.xmlSubtypes {
		write(buf, entry.primitives)
		marshalBddDnf(buf, entry.sequence)
	}

	marshalSubtypeData(buf, bp.subtypeData)
	marshalTypes(buf, bp.types)

	return buf.Bytes()
}

func UnmarshalTypePool(data []byte, env Env) *TypePool {
	r := bytes.NewReader(data)
	bp := binaryPool{}

	read(r, &bp.nIntSubtypes)
	read(r, &bp.nBooleanSubtypes)
	read(r, &bp.nFloatSubtypes)
	read(r, &bp.nDecimalSubtypes)
	read(r, &bp.nStringSubtypes)

	bp.intSubtypes = make([]intSubTypeEntry, bp.nIntSubtypes)
	for i := range bp.intSubtypes {
		bp.intSubtypes[i] = unmarshalIntSubtype(r)
	}
	bp.booleanSubtype = make([]booleanSubtypeEntry, bp.nBooleanSubtypes)
	for i := range bp.booleanSubtype {
		bp.booleanSubtype[i] = unmarshalBooleanSubtype(r)
	}
	bp.floatSubtype = make([]floatSubtypeEntry, bp.nFloatSubtypes)
	for i := range bp.floatSubtype {
		bp.floatSubtype[i] = unmarshalFloatSubtype(r)
	}
	bp.decimalSubtype = make([]decimalSubtypeEntry, bp.nDecimalSubtypes)
	for i := range bp.decimalSubtype {
		bp.decimalSubtype[i] = unmarshalDecimalSubtype(r)
	}
	bp.stringSubtype = make([]stringSubtypeEntry, bp.nStringSubtypes)
	for i := range bp.stringSubtype {
		bp.stringSubtype[i] = unmarshalStringSubtype(r)
	}

	read(r, &bp.nListBdds)
	read(r, &bp.nMappingBdds)
	read(r, &bp.nFunctionBdds)
	read(r, &bp.nTypedescBdds)
	read(r, &bp.nErrorBdds)
	read(r, &bp.nTableBdds)
	read(r, &bp.nObjectBdds)
	read(r, &bp.nStreamBdds)
	bp.listBdds = make([]unionOfIntersections, bp.nListBdds)
	for i := range bp.listBdds {
		bp.listBdds[i] = unmarshalBddDnf(r)
	}
	bp.mappingBdds = make([]unionOfIntersections, bp.nMappingBdds)
	for i := range bp.mappingBdds {
		bp.mappingBdds[i] = unmarshalBddDnf(r)
	}
	bp.functionBdds = make([]unionOfIntersections, bp.nFunctionBdds)
	for i := range bp.functionBdds {
		bp.functionBdds[i] = unmarshalBddDnf(r)
	}
	bp.typedescBdds = make([]unionOfIntersections, bp.nTypedescBdds)
	for i := range bp.typedescBdds {
		bp.typedescBdds[i] = unmarshalBddDnf(r)
	}
	bp.errorBdds = make([]unionOfIntersections, bp.nErrorBdds)
	for i := range bp.errorBdds {
		bp.errorBdds[i] = unmarshalBddDnf(r)
	}
	bp.tableBdds = make([]unionOfIntersections, bp.nTableBdds)
	for i := range bp.tableBdds {
		bp.tableBdds[i] = unmarshalBddDnf(r)
	}
	bp.objectBdds = make([]unionOfIntersections, bp.nObjectBdds)
	for i := range bp.objectBdds {
		bp.objectBdds[i] = unmarshalBddDnf(r)
	}
	bp.streamBdds = make([]unionOfIntersections, bp.nStreamBdds)
	for i := range bp.streamBdds {
		bp.streamBdds[i] = unmarshalBddDnf(r)
	}

	read(r, &bp.nListAtomicTypes)
	read(r, &bp.nMappingAtomicTypes)
	read(r, &bp.nFunctionAtomicTypes)
	read(r, &bp.nXmlAtomicTypes)
	bp.listAtomicTypes = make([]listAtomicTypeEntry, bp.nListAtomicTypes)
	for i := range bp.listAtomicTypes {
		bp.listAtomicTypes[i] = unmarshalListAtomicType(r)
	}
	bp.mappingAtomicTypes = make([]mappingAtomicTypeEntry, bp.nMappingAtomicTypes)
	for i := range bp.mappingAtomicTypes {
		bp.mappingAtomicTypes[i] = unmarshalMappingAtomicType(r)
	}
	bp.functionAtomicTypes = make([]functionAtomicTypeEntry, bp.nFunctionAtomicTypes)
	for i := range bp.functionAtomicTypes {
		bp.functionAtomicTypes[i] = unmarshalFunctionAtomicType(r)
	}
	bp.xmlAtomicTypes = make([]xmlAtomicTypeEntry, bp.nXmlAtomicTypes)
	for i := range bp.xmlAtomicTypes {
		read(r, &bp.xmlAtomicTypes[i].index)
	}

	read(r, &bp.nXmlSubtypes)
	bp.xmlSubtypes = make([]xmlSubtypeEntry, bp.nXmlSubtypes)
	for i := range bp.xmlSubtypes {
		read(r, &bp.xmlSubtypes[i].primitives)
		bp.xmlSubtypes[i].sequence = unmarshalBddDnf(r)
	}

	bp.subtypeData = unmarshalSubtypeData(r)
	bp.types = unmarshalTypes(r)

	return toTypePool(bp, env)
}

// binaryPool represent how type pool is represented in memory
type binaryPool struct {
	nIntSubtypes     uint32
	nBooleanSubtypes uint32
	nFloatSubtypes   uint32
	nDecimalSubtypes uint32
	nStringSubtypes  uint32
	intSubtypes      []intSubTypeEntry
	booleanSubtype   []booleanSubtypeEntry
	floatSubtype     []floatSubtypeEntry
	decimalSubtype   []decimalSubtypeEntry
	stringSubtype    []stringSubtypeEntry

	nListBdds     uint32
	nMappingBdds  uint32
	nFunctionBdds uint32
	nTypedescBdds uint32
	nErrorBdds    uint32
	nTableBdds    uint32
	nObjectBdds   uint32
	nStreamBdds   uint32
	listBdds      []unionOfIntersections
	mappingBdds   []unionOfIntersections
	functionBdds  []unionOfIntersections
	typedescBdds  []unionOfIntersections
	errorBdds     []unionOfIntersections
	tableBdds     []unionOfIntersections
	objectBdds    []unionOfIntersections
	streamBdds    []unionOfIntersections

	nListAtomicTypes     uint32
	nMappingAtomicTypes  uint32
	nFunctionAtomicTypes uint32
	nXmlAtomicTypes      uint32
	nXmlSubtypes         uint32
	listAtomicTypes      []listAtomicTypeEntry
	mappingAtomicTypes   []mappingAtomicTypeEntry
	functionAtomicTypes  []functionAtomicTypeEntry
	xmlAtomicTypes       []xmlAtomicTypeEntry
	xmlSubtypes          []xmlSubtypeEntry

	subtypeData []subtypeDataEntry
	types       []typeEntry
}

type typeEntry struct {
	all              uint32
	some             uint32
	subtypeDataStart uint32
	subtypeDataEnd   uint32
}

func marshalTypes(buf *bytes.Buffer, types []typeEntry) {
	write(buf, uint32(len(types)))
	for _, entry := range types {
		write(buf, entry.all)
		write(buf, entry.some)
		write(buf, entry.subtypeDataStart)
		write(buf, entry.subtypeDataEnd)
	}
}

func unmarshalTypes(r *bytes.Reader) []typeEntry {
	var count uint32
	read(r, &count)
	types := make([]typeEntry, count)
	for i := range types {
		read(r, &types[i].all)
		read(r, &types[i].some)
		read(r, &types[i].subtypeDataStart)
		read(r, &types[i].subtypeDataEnd)
	}
	return types
}

type subtypeDataEntry struct {
	kind  subtypeDataKind
	index uint32
}

type subtypeDataKind uint8

const (
	intSubtypeData subtypeDataKind = iota
	booleanSubtypeData
	floatSubtypeData
	decimalSubtypeData
	stringSubtypeData
	listBddSubtypeData
	mappingBddSubtypeData
	functionBddSubtypeData
	errorBddSubtypeData
	tableBddSubtypeData
	xmlSubtypeData
	objectBddSubtypeData
	streamBddSubtypeData
	typedescBddSubtypeData
)

func marshalSubtypeData(buf *bytes.Buffer, entries []subtypeDataEntry) {
	write(buf, uint32(len(entries)))
	for _, entry := range entries {
		write(buf, uint8(entry.kind))
		write(buf, entry.index)
	}
}

func unmarshalSubtypeData(r *bytes.Reader) []subtypeDataEntry {
	var count uint32
	read(r, &count)
	entries := make([]subtypeDataEntry, count)
	for i := range entries {
		var kind uint8
		read(r, &kind)
		entries[i].kind = subtypeDataKind(kind)
		read(r, &entries[i].index)
	}
	return entries
}

// int subtype

type intSubTypeEntry struct {
	nRanges uint32
	ranges  []intRange
}

func fromIntSubtype(st *intSubtype) intSubTypeEntry {
	ranges := make([]intRange, len(st.Ranges))
	for i, r := range st.Ranges {
		ranges[i] = intRange(r)
	}
	return intSubTypeEntry{nRanges: uint32(len(ranges)), ranges: ranges}
}

func toIntSubtype(entry intSubTypeEntry) intSubtype {
	ranges := make([]intRange, len(entry.ranges))
	for i, r := range entry.ranges {
		ranges[i] = intRange(r)
	}
	return newIntSubtypeFromRanges(ranges)
}

func marshalIntSubtype(buf *bytes.Buffer, entry intSubTypeEntry) {
	write(buf, entry.nRanges)
	for _, r := range entry.ranges {
		write(buf, r.Min)
		write(buf, r.Max)
	}
}

func unmarshalIntSubtype(r *bytes.Reader) intSubTypeEntry {
	var entry intSubTypeEntry
	read(r, &entry.nRanges)
	entry.ranges = make([]intRange, entry.nRanges)
	for j := range entry.ranges {
		read(r, &entry.ranges[j].Min)
		read(r, &entry.ranges[j].Max)
	}
	return entry
}

// boolean subtype

type booleanSubtypeEntry bool

func fromBooleanSubtype(st *booleanSubtype) booleanSubtypeEntry {
	return booleanSubtypeEntry(st.Value)
}

func toBooleanSubtype(entry booleanSubtypeEntry) booleanSubtype {
	return booleanSubtypeFrom(bool(entry))
}

func marshalBooleanSubtype(buf *bytes.Buffer, entry booleanSubtypeEntry) {
	write(buf, bool(entry))
}

func unmarshalBooleanSubtype(r *bytes.Reader) booleanSubtypeEntry {
	var v bool
	read(r, &v)
	return booleanSubtypeEntry(v)
}

// float subtype

// These have fixed sizes so binary can read them
type sized interface {
	int8 | int16 | int32 | int64 |
		uint8 | uint16 | uint32 | uint64 |
		float32 | float64 | decimalEntry
}
type enumerableSubtypeEntry[T sized] struct {
	allowed bool
	nValues int32
	values  []T
}

type floatSubtypeEntry enumerableSubtypeEntry[float64]

func fromFloatSubtype(st *floatSubtype) floatSubtypeEntry {
	nValues := len(st.Values())
	values := make([]float64, nValues)
	for i, v := range st.Values() {
		values[i] = v.Value()
	}
	return floatSubtypeEntry{allowed: st.Allowed(), nValues: int32(nValues), values: values}
}

func toFloatSubtype(entry floatSubtypeEntry) ProperSubtypeData {
	values := make([]enumerableType[float64], entry.nValues)
	for i, v := range entry.values {
		f := newEnumerableFloatFromFloat64(v)
		values[i] = &f
	}
	return createFloatSubtype(entry.allowed, values)
}

func marshalFloatSubtype(buf *bytes.Buffer, entry floatSubtypeEntry) {
	write(buf, entry.allowed)
	write(buf, entry.nValues)
	for _, v := range entry.values {
		write(buf, v)
	}
}

func unmarshalFloatSubtype(r *bytes.Reader) floatSubtypeEntry {
	var entry floatSubtypeEntry
	read(r, &entry.allowed)
	read(r, &entry.nValues)
	entry.values = make([]float64, entry.nValues)
	for j := range entry.values {
		read(r, &entry.values[j])
	}
	return entry
}

// decimal subtype

type decimalSubtypeEntry enumerableSubtypeEntry[decimalEntry]

// Decimal singletons can hold arbitrary-precision values, so we round-trip them
// through their canonical decimal128 string form rather than packing into fixed
// width integers.
type decimalEntry struct {
	value string
}

func fromDecimalSubtype(st *decimalSubtype) decimalSubtypeEntry {
	values := make([]decimalEntry, len(st.Values()))
	for i, v := range st.Values() {
		r := v.Value()
		values[i] = decimalEntry{value: r.String()}
	}
	return decimalSubtypeEntry{allowed: st.Allowed(), nValues: int32(len(values)), values: values}
}

func toDecimalSubtype(entry decimalSubtypeEntry) ProperSubtypeData {
	values := make([]enumerableType[decimal.Decimal], len(entry.values))
	for i, v := range entry.values {
		r, err := decimal.FromString(v.value)
		if err != nil {
			panic(fmt.Sprintf("invalid serialized decimal value %q: %v", v.value, err))
		}
		d := enumerableDecimalFrom(*r)
		values[i] = &d
	}
	return createDecimalSubtype(entry.allowed, values)
}

func marshalDecimalSubtype(buf *bytes.Buffer, entry decimalSubtypeEntry) {
	write(buf, entry.allowed)
	write(buf, entry.nValues)
	for _, v := range entry.values {
		b := []byte(v.value)
		write(buf, int32(len(b)))
		if _, err := buf.Write(b); err != nil {
			panic(fmt.Sprintf("writing decimal value bytes: %v", err))
		}
	}
}

func unmarshalDecimalSubtype(r *bytes.Reader) decimalSubtypeEntry {
	var entry decimalSubtypeEntry
	read(r, &entry.allowed)
	read(r, &entry.nValues)
	entry.values = make([]decimalEntry, entry.nValues)
	for j := range entry.values {
		var n int32
		read(r, &n)
		b := make([]byte, n)
		if _, err := r.Read(b); err != nil {
			panic(fmt.Sprintf("reading decimal value bytes: %v", err))
		}
		entry.values[j].value = string(b)
	}
	return entry
}

// string subtype

type stringSubtypeEntry struct {
	charData    enumerableStringData
	nonCharData enumerableStringData
}

type enumerableStringData struct {
	allowed bool
	num     int64
	values  []enumerableStringDataEntry
}

type enumerableStringDataEntry struct {
	len    int32
	values []byte
}

func fromEnumerableStringSubtype(es enumerableSubtype[string]) enumerableStringData {
	entries := make([]enumerableStringDataEntry, len(es.Values()))
	for i, v := range es.Values() {
		b := []byte(v.Value())
		entries[i] = enumerableStringDataEntry{len: int32(len(b)), values: b}
	}
	return enumerableStringData{allowed: es.Allowed(), num: int64(len(entries)), values: entries}
}

func toEnumerableStrings(data enumerableStringData) (bool, []enumerableType[string]) {
	values := make([]enumerableType[string], len(data.values))
	for i, v := range data.values {
		values[i] = enumerableCharStringFrom(string(v.values))
	}
	return data.allowed, values
}

func fromStringSubtype(st *stringSubtype) stringSubtypeEntry {
	return stringSubtypeEntry{
		charData:    fromEnumerableStringSubtype(st.GetChar()),
		nonCharData: fromEnumerableStringSubtype(st.GetNonChar()),
	}
}

func toStringSubtype(entry stringSubtypeEntry) stringSubtype {
	charAllowed, charValues := toEnumerableStrings(entry.charData)
	nonCharAllowed, nonCharValues := toEnumerableStrings(entry.nonCharData)
	return stringSubtypeFrom(
		charStringSubtypeFrom(charAllowed, charValues),
		nonCharStringSubtypeFrom(nonCharAllowed, nonCharValues),
	)
}

func marshalStringSubtype(buf *bytes.Buffer, entry stringSubtypeEntry) {
	marshalEnumerableStringData(buf, entry.charData)
	marshalEnumerableStringData(buf, entry.nonCharData)
}

func unmarshalStringSubtype(r *bytes.Reader) stringSubtypeEntry {
	return stringSubtypeEntry{
		charData:    unmarshalEnumerableStringData(r),
		nonCharData: unmarshalEnumerableStringData(r),
	}
}

func marshalEnumerableStringData(buf *bytes.Buffer, data enumerableStringData) {
	write(buf, data.allowed)
	write(buf, data.num)
	for _, entry := range data.values {
		write(buf, entry.len)
		_, err := buf.Write(entry.values)
		if err != nil {
			panic(fmt.Sprintf("writing string entry bytes: %v", err))
		}
	}
}

func unmarshalEnumerableStringData(r *bytes.Reader) enumerableStringData {
	var data enumerableStringData
	read(r, &data.allowed)
	read(r, &data.num)
	data.values = make([]enumerableStringDataEntry, data.num)
	for i := range data.values {
		read(r, &data.values[i].len)
		data.values[i].values = make([]byte, data.values[i].len)
		_, err := r.Read(data.values[i].values)
		if err != nil {
			panic(fmt.Sprintf("reading string entry bytes: %v", err))
		}
	}
	return data
}

// binary helpers

func write(buf *bytes.Buffer, data any) {
	if err := binary.Write(buf, binary.BigEndian, data); err != nil {
		panic(fmt.Sprintf("writing binary data: %v", err))
	}
}

func read(r *bytes.Reader, v any) {
	if err := binary.Read(r, binary.BigEndian, v); err != nil {
		panic(fmt.Sprintf("reading binary data: %v", err))
	}
}
