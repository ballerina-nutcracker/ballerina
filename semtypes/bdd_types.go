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

// I am not very happy about the way this turned out. In an ideal world what we should be doing is serailize semtypes in to a form
// that is close as possible to ballerina syntax (with negation) then reparse it and re-resolve the type cleanly in a new type env.
// To do this not only we need to serilaize type but type operations more generally, which we currently do in a half-hearted manner for BDDs.
// When we have time we should think about a better way to do this that does not require these speical cases.

import (
	"bytes"
)

// BDD DNF representation: union of intersections

type atomEntry struct {
	isRec bool
	index int32
}

type atomConjunction struct {
	nPosAtoms uint32
	posAtoms  []atomEntry
	nNegAtoms uint32
	negAtoms  []atomEntry
}

type unionOfIntersections struct {
	nConjunctions uint32
	conjunctions  []atomConjunction
}

// Atomic type entries for each BDD kind

type listAtomicTypeEntry struct {
	fixedLength int32
	nInitial    int32
	initial     []TypePoolIndex
	rest        TypePoolIndex
	mut         uint8
}

type mappingAtomicTypeEntry struct {
	nFields int32
	names   []enumerableStringDataEntry
	types   []TypePoolIndex
	muts    []uint8
	rest    TypePoolIndex
	restMut uint8
}

type functionAtomicTypeEntry struct {
	paramType  TypePoolIndex
	retType    TypePoolIndex
	qualifiers TypePoolIndex
	isGeneric  bool
}

type xmlAtomicTypeEntry struct {
	index int32
}

type xmlSubtypeEntry struct {
	primitives int32
	sequence   unionOfIntersections
}

// Serialization: decompose BDD into DNF

type bddSerializationContext struct {
	pool *TypePool
	cx   Context

	listAtomMap     map[atom]int32
	mappingAtomMap  map[atom]int32
	functionAtomMap map[atom]int32
	xmlAtomMap      map[atom]int32

	bp *binaryPool
}

func newBddSerializationContext(pool *TypePool, cx Context, bp *binaryPool) *bddSerializationContext {
	sc := &bddSerializationContext{
		pool:            pool,
		cx:              cx,
		listAtomMap:     make(map[atom]int32),
		mappingAtomMap:  make(map[atom]int32),
		functionAtomMap: make(map[atom]int32),
		xmlAtomMap:      make(map[atom]int32),
		bp:              bp,
	}
	// Reserve index 0 in list and mapping atom tables for BDD_REC_ATOM_READONLY
	bp.listAtomicTypes = append(bp.listAtomicTypes, listAtomicTypeEntry{})
	bp.mappingAtomicTypes = append(bp.mappingAtomicTypes, mappingAtomicTypeEntry{})
	return sc
}

func (sc *bddSerializationContext) serializeListBdd(bdd Bdd) unionOfIntersections {
	return sc.serializeBdd(bdd, sc.listAtomMap, sc.serializeListAtom, kind_LIST_ATOM)
}

func (sc *bddSerializationContext) serializeMappingBdd(bdd Bdd) unionOfIntersections {
	return sc.serializeBdd(bdd, sc.mappingAtomMap, sc.serializeMappingAtom, kind_MAPPING_ATOM)
}

func (sc *bddSerializationContext) serializeFunctionBdd(bdd Bdd) unionOfIntersections {
	return sc.serializeBdd(bdd, sc.functionAtomMap, sc.serializeFunctionAtom, kind_FUNCTION_ATOM)
}

func (sc *bddSerializationContext) serializeXmlBdd(bdd Bdd) unionOfIntersections {
	return sc.serializeBdd(bdd, sc.xmlAtomMap, sc.serializeXMLAtom, kind_XML_ATOM)
}

func (sc *bddSerializationContext) serializeXmlSubtype(xs *xmlSubtype) xmlSubtypeEntry {
	return xmlSubtypeEntry{
		primitives: int32(xs.Primitives),
		sequence:   sc.serializeXmlBdd(xs.Sequence),
	}
}

func (sc *bddSerializationContext) serializeBdd(
	bdd Bdd,
	atomMap map[atom]int32,
	serializeAtom func(atom) int32,
	atomKind kind,
) unionOfIntersections {
	var conjs []atomConjunction
	bddEvery(sc.cx, bdd, conjunctionNil, conjunctionNil, func(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
		var posAtoms []atomEntry
		for c := pos; c != conjunctionNil; c = cx.conjunctionNext(c) {
			posAtoms = append(posAtoms, sc.resolveAtom(cx.conjunctionAtom(c), atomMap, serializeAtom, atomKind))
		}
		// Reverse since conjunction is built in reverse order by bddEvery
		for i, j := 0, len(posAtoms)-1; i < j; i, j = i+1, j-1 {
			posAtoms[i], posAtoms[j] = posAtoms[j], posAtoms[i]
		}
		var negAtoms []atomEntry
		for c := neg; c != conjunctionNil; c = cx.conjunctionNext(c) {
			negAtoms = append(negAtoms, sc.resolveAtom(cx.conjunctionAtom(c), atomMap, serializeAtom, atomKind))
		}
		for i, j := 0, len(negAtoms)-1; i < j; i, j = i+1, j-1 {
			negAtoms[i], negAtoms[j] = negAtoms[j], negAtoms[i]
		}
		conjs = append(conjs, atomConjunction{
			nPosAtoms: uint32(len(posAtoms)),
			posAtoms:  posAtoms,
			nNegAtoms: uint32(len(negAtoms)),
			negAtoms:  negAtoms,
		})
		return true
	})
	return unionOfIntersections{
		nConjunctions: uint32(len(conjs)),
		conjunctions:  conjs,
	}
}

func (sc *bddSerializationContext) resolveAtom(
	atom atom,
	atomMap map[atom]int32,
	serializeAtom func(atom) int32,
	atomKind kind,
) atomEntry {
	if recAtom, ok := atom.(*recAtom); ok {
		if recAtom.index() < 0 {
			return atomEntry{isRec: true, index: int32(recAtom.index())}
		}
		if recAtom.index() == BDD_REC_ATOM_READONLY {
			if atomKind == kind_LIST_ATOM || atomKind == kind_MAPPING_ATOM {
				return atomEntry{isRec: false, index: 0}
			}
		}
	}
	if idx, ok := atomMap[atom]; ok {
		return atomEntry{isRec: true, index: idx}
	}
	idx := serializeAtom(atom)
	return atomEntry{isRec: false, index: idx}
}

func (sc *bddSerializationContext) serializeListAtom(atom atom) int32 {
	idx := int32(len(sc.bp.listAtomicTypes))
	sc.listAtomMap[atom] = idx
	sc.bp.listAtomicTypes = append(sc.bp.listAtomicTypes, listAtomicTypeEntry{})

	at := sc.cx.ListAtomType(atom)
	initial := make([]TypePoolIndex, len(at.Members.initial))
	var mut uint8
	for i := range at.Members.initial {
		cell := at.Members.initial[i]
		initial[i] = sc.pool.Put(cellInner(cell))
		mut = uint8(cellMut(cell))
	}
	rest := sc.pool.Put(cellInner(at.rest))
	if len(at.Members.initial) == 0 {
		mut = uint8(cellMut(at.rest))
	}

	sc.bp.listAtomicTypes[idx] = listAtomicTypeEntry{
		fixedLength: int32(at.Members.FixedLength),
		nInitial:    int32(len(initial)),
		initial:     initial,
		rest:        rest,
		mut:         mut,
	}
	return idx
}

func (sc *bddSerializationContext) serializeMappingAtom(atom atom) int32 {
	idx := int32(len(sc.bp.mappingAtomicTypes))
	sc.mappingAtomMap[atom] = idx
	sc.bp.mappingAtomicTypes = append(sc.bp.mappingAtomicTypes, mappingAtomicTypeEntry{})

	at := sc.cx.MappingAtomType(atom)
	names := make([]enumerableStringDataEntry, len(at.Names))
	types := make([]TypePoolIndex, len(at.Types))
	muts := make([]uint8, len(at.Types))
	for i, name := range at.Names {
		b := []byte(name)
		names[i] = enumerableStringDataEntry{len: int32(len(b)), values: b}
		atomTy := at.Types[i]
		types[i] = sc.pool.Put(cellInner(atomTy))
		muts[i] = uint8(cellMut(atomTy))
	}
	rest := sc.pool.Put(cellInner(at.Rest))
	restMut := uint8(cellMut(at.Rest))

	sc.bp.mappingAtomicTypes[idx] = mappingAtomicTypeEntry{
		nFields: int32(len(names)),
		names:   names,
		types:   types,
		muts:    muts,
		rest:    rest,
		restMut: restMut,
	}
	return idx
}

func (sc *bddSerializationContext) serializeFunctionAtom(atom atom) int32 {
	idx := int32(len(sc.bp.functionAtomicTypes))
	sc.functionAtomMap[atom] = idx
	sc.bp.functionAtomicTypes = append(sc.bp.functionAtomicTypes, functionAtomicTypeEntry{})

	at := sc.cx.FunctionAtomType(atom)
	entry := functionAtomicTypeEntry{
		paramType:  sc.pool.Put(at.ParamType),
		retType:    sc.pool.Put(at.RetType),
		qualifiers: sc.pool.Put(at.Qualifiers),
		isGeneric:  at.IsGeneric,
	}

	sc.bp.functionAtomicTypes[idx] = entry
	return idx
}

func (sc *bddSerializationContext) serializeXMLAtom(atom atom) int32 {
	idx := int32(len(sc.bp.xmlAtomicTypes))
	sc.xmlAtomMap[atom] = idx
	sc.bp.xmlAtomicTypes = append(sc.bp.xmlAtomicTypes, xmlAtomicTypeEntry{
		index: int32(atom.index()),
	})
	return idx
}

func cellMut(cell SemType) CellMutability {
	bdd := cell.subtypeDataList()[0].(bddNode)
	cat := bdd.atom().(*typeAtom).AtomicType.(*cellAtomicType)
	return cat.Mut
}

// Deserialization: reconstruct BDDs from DNF

type bddDeserializationContext struct {
	pool *TypePool
	env  Env
	bp   *binaryPool

	listAtomDefs []*ListDefinition
	listAtoms    []atom

	mappingAtomDefs []*MappingDefinition
	mappingAtoms    []atom

	functionAtomDefs []*FunctionDefinition
	functionAtoms    []atom

	xmlAtoms []atom
}

func newBddDeserializationContext(pool *TypePool, env Env, bp *binaryPool) *bddDeserializationContext {
	return &bddDeserializationContext{
		pool:             pool,
		env:              env,
		bp:               bp,
		listAtomDefs:     make([]*ListDefinition, len(bp.listAtomicTypes)),
		listAtoms:        make([]atom, len(bp.listAtomicTypes)),
		mappingAtomDefs:  make([]*MappingDefinition, len(bp.mappingAtomicTypes)),
		mappingAtoms:     make([]atom, len(bp.mappingAtomicTypes)),
		functionAtomDefs: make([]*FunctionDefinition, len(bp.functionAtomicTypes)),
		functionAtoms:    make([]atom, len(bp.functionAtomicTypes)),
		xmlAtoms:         make([]atom, len(bp.xmlAtomicTypes)),
	}
}

func (dc *bddDeserializationContext) deserializeType(poolIndex int) SemType {
	if !IsZero(dc.pool.tys[poolIndex]) {
		return dc.pool.tys[poolIndex]
	}
	te := dc.bp.types[poolIndex]
	var subtypeDataList []ProperSubtypeData
	for _, sde := range dc.bp.subtypeData[te.subtypeDataStart:te.subtypeDataEnd] {
		var data ProperSubtypeData
		switch sde.kind {
		case intSubtypeData:
			data = toIntSubtype(dc.bp.intSubtypes[sde.index])
		case booleanSubtypeData:
			data = toBooleanSubtype(dc.bp.booleanSubtype[sde.index])
		case floatSubtypeData:
			data = toFloatSubtype(dc.bp.floatSubtype[sde.index])
		case decimalSubtypeData:
			data = toDecimalSubtype(dc.bp.decimalSubtype[sde.index])
		case stringSubtypeData:
			data = toStringSubtype(dc.bp.stringSubtype[sde.index])
		case listBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.listBdds[sde.index], dc.deserializeListAtom)
		case mappingBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.mappingBdds[sde.index], dc.deserializeMappingAtom)
		case functionBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.functionBdds[sde.index], dc.deserializeFunctionAtom)
		case typedescBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.typedescBdds[sde.index], dc.deserializeMappingAtom)
		case errorBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.errorBdds[sde.index], dc.deserializeMappingAtom)
		case tableBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.tableBdds[sde.index], dc.deserializeListAtom)
		case objectBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.objectBdds[sde.index], dc.deserializeMappingAtom)
		case streamBddSubtypeData:
			data = dc.deserializeBddFromDnf(dc.bp.streamBdds[sde.index], dc.deserializeListAtom)
		case xmlSubtypeData:
			entry := dc.bp.xmlSubtypes[sde.index]
			sequence := dc.deserializeBddFromDnf(entry.sequence, dc.deserializeXmlAtom)
			data = xmlSubtypeFrom(int(entry.primitives), sequence)
		}
		subtypeDataList = append(subtypeDataList, data)
	}
	ty := createComplexSemTypeWithAllBitSetSomeBitSetSubtypeDataList(
		basicTypeBitSet(te.all), basicTypeBitSet(te.some), subtypeDataList,
	)
	dc.pool.tys[poolIndex] = ty
	return ty
}

func (dc *bddDeserializationContext) deserializeBddFromDnf(
	dnf unionOfIntersections,
	deserializeAtom func(int32) atom,
) Bdd {
	atoms := make(map[int32]atom)
	for _, conj := range dnf.conjunctions {
		for _, a := range conj.posAtoms {
			if _, ok := atoms[a.index]; !ok {
				atoms[a.index] = deserializeAtom(a.index)
			}
		}
		for _, a := range conj.negAtoms {
			if _, ok := atoms[a.index]; !ok {
				atoms[a.index] = deserializeAtom(a.index)
			}
		}
	}
	return buildBddFromDnf(dnf, atoms)
}

func (dc *bddDeserializationContext) deserializeListAtom(atomIndex int32) atom {
	if atomIndex == 0 {
		ro := createRecAtom(BDD_REC_ATOM_READONLY, recAtomKindList)
		return &ro
	}
	if dc.listAtoms[atomIndex] != nil {
		return dc.listAtoms[atomIndex]
	}
	if def := dc.listAtomDefs[atomIndex]; def != nil {
		placeholder := def.GetSemType(dc.env)
		atom := extractAtom(placeholder)
		dc.listAtoms[atomIndex] = atom
		return atom
	}

	def := NewListDefinition()
	dc.listAtomDefs[atomIndex] = &def

	entry := dc.bp.listAtomicTypes[atomIndex]
	initial := make([]SemType, entry.nInitial)
	for j := range initial {
		initial[j] = dc.resolvePoolType(entry.initial[j])
	}
	rest := dc.resolvePoolType(entry.rest)
	mut := CellMutability(entry.mut)

	result := def.DefineListTypeWrapped(dc.env, initial, int(entry.fixedLength), rest, mut)
	atom := extractAtom(result)
	dc.listAtoms[atomIndex] = atom
	return atom
}

func (dc *bddDeserializationContext) deserializeMappingAtom(atomIndex int32) atom {
	if atomIndex < 0 {
		atom := createDistinctRecAtom(int(atomIndex))
		return &atom
	}
	if atomIndex == 0 {
		ro := createRecAtom(BDD_REC_ATOM_READONLY, recAtomKindMapping)
		return &ro
	}
	if dc.mappingAtoms[atomIndex] != nil {
		return dc.mappingAtoms[atomIndex]
	}
	if def := dc.mappingAtomDefs[atomIndex]; def != nil {
		placeholder := def.GetSemType(dc.env)
		atom := extractAtom(placeholder)
		dc.mappingAtoms[atomIndex] = atom
		return atom
	}

	def := NewMappingDefinition()
	dc.mappingAtomDefs[atomIndex] = &def

	entry := dc.bp.mappingAtomicTypes[atomIndex]
	cellFields := make([]CellField, entry.nFields)
	for j := range cellFields {
		inner := dc.resolvePoolType(entry.types[j])
		mut := CellMutability(entry.muts[j])
		cellFields[j] = cellFieldFrom(string(entry.names[j].values),
			cellContainingWithEnvSemTypeCellMutability(dc.env, inner, mut))
	}
	restInner := dc.resolvePoolType(entry.rest)
	restCell := cellContainingWithEnvSemTypeCellMutability(dc.env, restInner, CellMutability(entry.restMut))

	result := def.Define(dc.env, cellFields, restCell)
	atom := extractAtom(result)
	dc.mappingAtoms[atomIndex] = atom
	return atom
}

func (dc *bddDeserializationContext) deserializeFunctionAtom(atomIndex int32) atom {
	if dc.functionAtoms[atomIndex] != nil {
		return dc.functionAtoms[atomIndex]
	}
	if def := dc.functionAtomDefs[atomIndex]; def != nil {
		placeholder := def.GetSemType(dc.env)
		atom := extractAtom(placeholder)
		dc.functionAtoms[atomIndex] = atom
		return atom
	}
	def := NewFunctionDefinition()
	dc.functionAtomDefs[atomIndex] = &def

	entry := dc.bp.functionAtomicTypes[atomIndex]
	paramType := dc.resolvePoolType(entry.paramType)
	retType := dc.resolvePoolType(entry.retType)
	qualifiers := dc.resolvePoolType(entry.qualifiers)

	var result SemType
	if entry.isGeneric {
		result = def.DefineGeneric(dc.env, paramType, retType, newFunctionQualifiersFromSemType(qualifiers))
	} else {
		result = def.Define(dc.env, paramType, retType, newFunctionQualifiersFromSemType(qualifiers))
	}
	atom := extractAtom(result)
	dc.functionAtoms[atomIndex] = atom
	return atom
}

func (dc *bddDeserializationContext) deserializeXmlAtom(atomIndex int32) atom {
	if dc.xmlAtoms[atomIndex] != nil {
		return dc.xmlAtoms[atomIndex]
	}
	entry := dc.bp.xmlAtomicTypes[atomIndex]
	recAtom := createXMLRecAtom(int(entry.index))
	dc.xmlAtoms[atomIndex] = &recAtom
	return &recAtom
}

func (dc *bddDeserializationContext) resolvePoolType(idx TypePoolIndex) SemType {
	if idx <= 0 {
		bits := idx & indexMask
		return basicTypeBitSet(bits).semType()
	}
	return dc.deserializeType(int(idx - 1))
}

func extractAtom(ty SemType) atom {
	cst := ty
	return cst.subtypeDataList()[0].(bddNode).atom()
}

func buildBddFromDnf(dnf unionOfIntersections, atoms map[int32]atom) Bdd {
	var bdd Bdd = bddNothing()
	for _, conj := range dnf.conjunctions {
		var term Bdd = bddAll()
		for _, a := range conj.posAtoms {
			term = bddIntersect(term, bddAtom(atoms[a.index]))
		}
		for _, a := range conj.negAtoms {
			term = bddDiff(term, bddAtom(atoms[a.index]))
		}
		bdd = bddUnion(bdd, term)
	}
	return bdd
}

// Marshal/unmarshal for BDD types

func marshalAtomEntry(buf *bytes.Buffer, entry atomEntry) {
	write(buf, entry.isRec)
	write(buf, entry.index)
}

func unmarshalAtomEntry(r *bytes.Reader) atomEntry {
	var entry atomEntry
	read(r, &entry.isRec)
	read(r, &entry.index)
	return entry
}

func marshalBddDnf(buf *bytes.Buffer, dnf unionOfIntersections) {
	write(buf, dnf.nConjunctions)
	for _, conj := range dnf.conjunctions {
		write(buf, conj.nPosAtoms)
		for _, a := range conj.posAtoms {
			marshalAtomEntry(buf, a)
		}
		write(buf, conj.nNegAtoms)
		for _, a := range conj.negAtoms {
			marshalAtomEntry(buf, a)
		}
	}
}

func unmarshalBddDnf(r *bytes.Reader) unionOfIntersections {
	var dnf unionOfIntersections
	read(r, &dnf.nConjunctions)
	dnf.conjunctions = make([]atomConjunction, dnf.nConjunctions)
	for i := range dnf.conjunctions {
		read(r, &dnf.conjunctions[i].nPosAtoms)
		dnf.conjunctions[i].posAtoms = make([]atomEntry, dnf.conjunctions[i].nPosAtoms)
		for j := range dnf.conjunctions[i].posAtoms {
			dnf.conjunctions[i].posAtoms[j] = unmarshalAtomEntry(r)
		}
		read(r, &dnf.conjunctions[i].nNegAtoms)
		dnf.conjunctions[i].negAtoms = make([]atomEntry, dnf.conjunctions[i].nNegAtoms)
		for j := range dnf.conjunctions[i].negAtoms {
			dnf.conjunctions[i].negAtoms[j] = unmarshalAtomEntry(r)
		}
	}
	return dnf
}

func marshalListAtomicType(buf *bytes.Buffer, entry listAtomicTypeEntry) {
	write(buf, entry.fixedLength)
	write(buf, entry.nInitial)
	for _, idx := range entry.initial {
		write(buf, int32(idx))
	}
	write(buf, int32(entry.rest))
	write(buf, entry.mut)
}

func unmarshalListAtomicType(r *bytes.Reader) listAtomicTypeEntry {
	var entry listAtomicTypeEntry
	read(r, &entry.fixedLength)
	read(r, &entry.nInitial)
	entry.initial = make([]TypePoolIndex, entry.nInitial)
	for j := range entry.initial {
		var idx int32
		read(r, &idx)
		entry.initial[j] = TypePoolIndex(idx)
	}
	var rest int32
	read(r, &rest)
	entry.rest = TypePoolIndex(rest)
	read(r, &entry.mut)
	return entry
}

func marshalMappingAtomicType(buf *bytes.Buffer, entry mappingAtomicTypeEntry) {
	write(buf, entry.nFields)
	for _, name := range entry.names {
		write(buf, name.len)
		buf.Write(name.values)
	}
	for _, idx := range entry.types {
		write(buf, int32(idx))
	}
	for _, m := range entry.muts {
		write(buf, m)
	}
	rest := int32(entry.rest)
	write(buf, rest)
	write(buf, entry.restMut)
}

func unmarshalMappingAtomicType(r *bytes.Reader) mappingAtomicTypeEntry {
	var entry mappingAtomicTypeEntry
	read(r, &entry.nFields)
	entry.names = make([]enumerableStringDataEntry, entry.nFields)
	for j := range entry.names {
		read(r, &entry.names[j].len)
		entry.names[j].values = make([]byte, entry.names[j].len)
		if _, err := r.Read(entry.names[j].values); err != nil {
			panic(err)
		}
	}
	entry.types = make([]TypePoolIndex, entry.nFields)
	for j := range entry.types {
		var idx int32
		read(r, &idx)
		entry.types[j] = TypePoolIndex(idx)
	}
	entry.muts = make([]uint8, entry.nFields)
	for j := range entry.muts {
		read(r, &entry.muts[j])
	}
	var rest int32
	read(r, &rest)
	entry.rest = TypePoolIndex(rest)
	read(r, &entry.restMut)
	return entry
}

func marshalFunctionAtomicType(buf *bytes.Buffer, entry functionAtomicTypeEntry) {
	write(buf, int32(entry.paramType))
	write(buf, int32(entry.retType))
	write(buf, int32(entry.qualifiers))
	write(buf, entry.isGeneric)
}

func unmarshalFunctionAtomicType(r *bytes.Reader) functionAtomicTypeEntry {
	var entry functionAtomicTypeEntry
	var paramType, retType, qualifiers int32
	read(r, &paramType)
	read(r, &retType)
	read(r, &qualifiers)
	entry.paramType = TypePoolIndex(paramType)
	entry.retType = TypePoolIndex(retType)
	entry.qualifiers = TypePoolIndex(qualifiers)
	read(r, &entry.isGeneric)
	return entry
}
