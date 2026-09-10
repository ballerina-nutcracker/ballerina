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
	"slices"
	"sort"
)

// MappingSpreadShape is the atomic construction shape of a mapping operand of a spread field.
// Fields are sorted by name and carry the possible member values of each distinguished key;
// Rest carries the possible member values of every other key. It summarises the operand for
// inference and target compatibility, and is not a replacement for the operand's semantic type
// in exact collision queries.
type MappingSpreadShape struct {
	Fields    []Field
	Rest      SemType
	Inhabited bool
}

func (f Field) Name() string {
	return f.name
}

func (f Field) Type() SemType {
	return f.typeOf
}

func (f Field) Readonly() bool {
	return f.readonly
}

func (f Field) Optional() bool {
	return f.optional
}

// MappingShapeForSpread summarises ty, which the caller has already established to be a subtype
// of mapping. An uninhabited operand reports Inhabited false with no fields and a never rest; an
// inhabited empty record reports Inhabited true.
func MappingShapeForSpread(cx Context, ty SemType) MappingSpreadShape {
	ty = Intersect(ty, Mapping)
	if IsEmpty(cx, ty) {
		return MappingSpreadShape{Rest: Never}
	}
	names := mappingSpreadDistinguishedNames(cx, ty)
	rest := mappingSpreadRestType(cx, ty, names)
	fields := make([]Field, 0, len(names))
	for _, name := range names {
		memberTy := mappingSpreadMemberType(cx, ty, name)
		if IsNever(memberTy) && IsNever(rest) {
			// The name is excluded and nothing else could have supplied it either, so
			// recording the exclusion would not constrain the constructed shape.
			continue
		}
		optional := !mappingSpreadGuaranteesKey(cx, ty, name)
		fields = append(fields, FieldFrom(name, memberTy, false, optional))
	}
	return MappingSpreadShape{Fields: fields, Rest: rest, Inhabited: true}
}

// MappingSpreadAllowsKey reports whether some value of ty contains name. It answers against the
// complete semantic type, so negative constraints introduced by narrowing and semantically empty
// fields both suppress the key.
func MappingSpreadAllowsKey(cx Context, ty SemType, name string) bool {
	ty = Intersect(ty, Mapping)
	return !IsEmpty(cx, Intersect(ty, mappingKeyPresent(cx.Env(), name)))
}

// MappingSpreadsOverlap reports whether two independently constructed mapping values can supply
// the same key. The returned name is a diagnostic witness when the overlap is on a distinguished
// key; the empty string is itself a valid key, so callers must rely on the boolean.
func MappingSpreadsOverlap(cx Context, left, right SemType) (string, bool) {
	left = Intersect(left, Mapping)
	right = Intersect(right, Mapping)
	if IsEmpty(cx, left) || IsEmpty(cx, right) {
		return "", false
	}
	names := mergeNames(mappingSpreadDistinguishedNames(cx, left), mappingSpreadDistinguishedNames(cx, right))
	for _, name := range names {
		if MappingSpreadAllowsKey(cx, left, name) && MappingSpreadAllowsKey(cx, right, name) {
			return name, true
		}
	}
	// Outside every name either type mentions, both types are governed only by their rest
	// descriptors, so permitting any such key at all means permitting all of them.
	if mappingSpreadAllowsKeyOutside(cx, left, names) && mappingSpreadAllowsKeyOutside(cx, right, names) {
		return "", true
	}
	return "", false
}

func mappingSpreadMemberType(cx Context, ty SemType, name string) SemType {
	if !MappingSpreadAllowsKey(cx, ty, name) {
		return Never
	}
	return mappingRefinedMemberType(cx, ty, name)
}

// mappingMemberRefinementLimit bounds how finely the projection is split. Beyond it the
// remaining parts are kept whole, which is still the projection's own over-approximation.
const mappingMemberRefinementLimit = 32

// mappingRefinedMemberType is the member type of name in ty. The projection it starts from
// ignores negated mapping atoms, so it can retain values narrowing has excluded. Splitting it
// along the member types the type's own atoms distinguish and dropping the parts no value of ty
// can hold at name removes those, since emptiness does account for negated atoms.
func mappingRefinedMemberType(cx Context, ty SemType, name string) SemType {
	key := StringConst(name)
	parts := []SemType{MappingMemberTypeInnerValProj(cx, ty, key)}
	for _, candidate := range mappingAtomMemberTypes(cx, ty, key) {
		if len(parts) >= mappingMemberRefinementLimit {
			break
		}
		parts = splitMappingMemberParts(cx, ty, name, parts, candidate)
	}
	result := Never
	for _, part := range parts {
		result = Union(result, part)
	}
	return result
}

func splitMappingMemberParts(cx Context, ty SemType, name string, parts []SemType, candidate SemType) []SemType {
	split := make([]SemType, 0, len(parts))
	for _, part := range parts {
		for _, piece := range []SemType{Intersect(part, candidate), Diff(part, candidate)} {
			if IsEmpty(cx, piece) || !mappingCanHoldMemberValue(cx, ty, name, piece) {
				continue
			}
			split = append(split, piece)
		}
	}
	return split
}

// mappingCanHoldMemberValue reports whether some value of ty holds a value of member at name.
func mappingCanHoldMemberValue(cx Context, ty SemType, name string, member SemType) bool {
	return !IsEmpty(cx, Intersect(ty, mappingKeyWithin(cx.Env(), name, member)))
}

// mappingAtomMemberTypes is the member type at key of every mapping atom ty mentions, whether
// the atom occurs positively or negatively. They are the boundaries along which the values of
// the key can be told apart.
func mappingAtomMemberTypes(cx Context, ty SemType, key SemType) []SemType {
	ty = Intersect(ty, Mapping)
	if ty.some() == 0 {
		return nil
	}
	keyData := getStringSubtype(key)
	if isNothingSubtype(keyData) {
		return nil
	}
	var types []SemType
	visited := map[bdd]struct{}{}
	collectMappingAtomMemberTypes(cx, getComplexSubtypeData(ty, btMapping).(bdd), keyData, visited, &types)
	return types
}

func collectMappingAtomMemberTypes(cx Context, b bdd, key subtypeData, visited map[bdd]struct{}, types *[]SemType) {
	if _, done := visited[b]; done {
		return
	}
	visited[b] = struct{}{}
	bn, ok := b.(bddNode)
	if !ok {
		return
	}
	memberTy := Diff(mappingAtomicMemberTypeInnerProj(cx.MappingAtomType(bn.atom()), key), Undef)
	if !slices.ContainsFunc(*types, func(ty SemType) bool { return IsSameType(cx, ty, memberTy) }) {
		*types = append(*types, memberTy)
	}
	collectMappingAtomMemberTypes(cx, bn.left(), key, visited, types)
	collectMappingAtomMemberTypes(cx, bn.middle(), key, visited, types)
	collectMappingAtomMemberTypes(cx, bn.right(), key, visited, types)
}

func mappingSpreadGuaranteesKey(cx Context, ty SemType, name string) bool {
	return IsEmpty(cx, Intersect(ty, mappingKeyAbsent(cx.Env(), name)))
}

// mappingSpreadRestType is the member type of the keys outside names. Outside every name the
// type mentions the mapping atoms are governed by their rest descriptors alone, so projecting a
// single name none of them declares answers for the whole remaining partition.
func mappingSpreadRestType(cx Context, ty SemType, names []string) SemType {
	if !mappingSpreadAllowsKeyOutside(cx, ty, names) {
		return Never
	}
	return mappingRefinedMemberType(cx, ty, undeclaredName(names))
}

func undeclaredName(names []string) string {
	name := "@rest"
	for slices.Contains(names, name) {
		name += "_"
	}
	return name
}

func mappingSpreadAllowsKeyOutside(cx Context, ty SemType, names []string) bool {
	return !IsSubtype(cx, ty, mappingKeysWithin(cx.Env(), names))
}

// MappingOfMemberType returns the mapping type admitting every mapping whose member values are
// all within memberTy. Unlike MappingDefinition.Define it keeps a mutable rest cell even for a
// never member type, so that a record declaring an impossible optional field still qualifies.
func MappingOfMemberType(env Env, memberTy SemType) SemType {
	md := NewMappingDefinition()
	return md.defineFromCells(env, nil, cellContainingWithEnvSemTypeCellMutability(env, Union(memberTy, Undef), CellMutabilityLimited))
}

// MappingOfFieldTypes is the closed mapping type whose fields are exactly fields. It
// contextually types a nested mapping constructor operand of a spread field without widening the
// operand's inferred shape to an open map.
func MappingOfFieldTypes(env Env, fields []MappingFieldInfo) SemType {
	cellFields := make([]cellField, 0, len(fields))
	for _, field := range fields {
		cell := cellContainingWithEnvSemTypeCellMutability(env, field.Type, CellMutabilityLimited)
		cellFields = append(cellFields, cellFieldFrom(field.Name, cell))
	}
	md := NewMappingDefinition()
	return md.defineFromCells(env, cellFields, mappingUndefCell(env))
}

// mappingKeyWithin is the type of every mapping whose name field is present and holds a value
// within member.
func mappingKeyWithin(env Env, name string, member SemType) SemType {
	md := NewMappingDefinition()
	cell := cellContainingWithEnvSemTypeCellMutability(env, member, cellMutabilityUnlimited)
	return md.defineFromCells(env, []cellField{cellFieldFrom(name, cell)}, cellSemtypeInner)
}

// mappingKeyPresent is the type of every mapping that contains name.
func mappingKeyPresent(env Env, name string) SemType {
	md := NewMappingDefinition()
	return md.defineFromCells(env, []cellField{cellFieldFrom(name, cellSemtypeVal)}, cellSemtypeInner)
}

// mappingKeyAbsent is the type of every mapping that does not contain name.
func mappingKeyAbsent(env Env, name string) SemType {
	md := NewMappingDefinition()
	return md.defineFromCells(env, []cellField{cellFieldFrom(name, mappingUndefCell(env))}, cellSemtypeInner)
}

// mappingKeysWithin is the type of every mapping whose keys all come from names.
func mappingKeysWithin(env Env, names []string) SemType {
	fields := make([]cellField, 0, len(names))
	for _, name := range names {
		fields = append(fields, cellFieldFrom(name, cellSemtypeInner))
	}
	md := NewMappingDefinition()
	return md.defineFromCells(env, fields, mappingUndefCell(env))
}

// mappingUndefCell is the widest cell holding no value. The predefined undef cell is immutable,
// which would exclude the mutable optional-never fields a record can declare.
func mappingUndefCell(env Env) SemType {
	return cellContainingWithEnvSemTypeCellMutability(env, Undef, CellMutabilityLimited)
}

// mappingSpreadDistinguishedNames returns every field name any mapping atom of ty mentions,
// sorted and deduplicated. Names from negated atoms are included so that the remaining-key
// partition is genuinely governed by rest descriptors alone.
func mappingSpreadDistinguishedNames(cx Context, ty SemType) []string {
	if ty.some() == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	visited := map[bdd]struct{}{}
	collectMappingAtomNames(cx, getComplexSubtypeData(ty, btMapping).(bdd), seen, visited)
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func collectMappingAtomNames(cx Context, b bdd, seen map[string]struct{}, visited map[bdd]struct{}) {
	if _, done := visited[b]; done {
		return
	}
	visited[b] = struct{}{}
	bn, ok := b.(bddNode)
	if !ok {
		return
	}
	if isPositiveAtom(bn.atom()) {
		for _, name := range cx.MappingAtomType(bn.atom()).names {
			seen[name] = struct{}{}
		}
	}
	collectMappingAtomNames(cx, bn.left(), seen, visited)
	collectMappingAtomNames(cx, bn.middle(), seen, visited)
	collectMappingAtomNames(cx, bn.right(), seen, visited)
}

func mergeNames(left, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	names := make([]string, 0, len(left)+len(right))
	for _, group := range [][]string{left, right} {
		for _, name := range group {
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

type MappingField struct {
	Name       string
	Type       SemType
	IsOptional bool
}

// AllPossibleMappingFields describes every name a value of ty can distinguish, in the order the
// type declares them, and the type of its members at every other name. A name the type excludes
// is present with a never member type: it can never hold a value, but the rest type does not
// apply to it either. An uninhabited ty distinguishes no names and has a never rest.
//
// It reports false for a mapping type that is not a single mapping atom, that is one with more
// than one alternative or with negated atoms, since the atom alone does not describe those
// exactly. Callers that need an exact set of possible names do not support them.
func AllPossibleMappingFields(cx Context, ty SemType) ([]MappingField, SemType, bool) {
	alts := MappingAlternatives(cx, Intersect(ty, Mapping))
	if len(alts) > 1 {
		return nil, SemType{}, false
	}
	if len(alts) == 0 {
		return nil, Never, true
	}
	if alts[0].HasNegativeAtoms() {
		return nil, SemType{}, false
	}
	atom := alts[0].Atomic()
	if atom == nil {
		// The mapping top distinguishes no names and holds any value at every name.
		return nil, Val, true
	}
	names := atom.FieldNames()
	fields := make([]MappingField, 0, len(names))
	for _, name := range names {
		fields = append(fields, MappingField{
			Name:       name,
			Type:       atom.FieldInnerVal(name),
			IsOptional: atom.IsOptional(cx, name),
		})
	}
	return fields, atom.RestInnerVal(), true
}
