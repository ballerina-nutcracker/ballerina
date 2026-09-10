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

// mappingUndefCell is the widest cell holding no value. The predefined undef cell is immutable,
// which would exclude the mutable optional-never fields a record can declare.
func mappingUndefCell(env Env) SemType {
	return cellContainingWithEnvSemTypeCellMutability(env, Undef, CellMutabilityLimited)
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
