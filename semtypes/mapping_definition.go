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
	"sort"
)

type MappingDefinition struct {
	rec     *recAtom
	semType SemType
}

var _ Definition = &MappingDefinition{}

func fieldName(f CellField) string {
	return f.Name
}

func NewMappingDefinition() MappingDefinition {
	this := MappingDefinition{}
	this.rec = nil
	// Default field initializations

	return this
}

func (m *MappingDefinition) GetSemType(env Env) SemType {
	s := m.semType
	if IsZero(s) {
		rec := env.recMappingAtom()
		m.rec = &rec
		return m.createSemType(env, &rec)
	} else {
		return s
	}
}

func (m *MappingDefinition) SetSemTypeToNever() {
	m.semType = NEVER
}

func (m *MappingDefinition) Define(env Env, fields []CellField, rest SemType) SemType {
	sfh := m.splitFields(fields)
	atomicType := mappingAtomicTypeFrom(sfh.Names, sfh.Types, rest)
	var a atom
	rec := m.rec
	if rec != nil {
		a = rec
		env.setRecMappingAtomType(*rec, &atomicType)
	} else {
		a = env.mappingAtom(&atomicType)
	}
	return m.createSemType(env, a)
}

func (m *MappingDefinition) DefineMappingTypeWrapped(env Env, fields []Field, rest SemType) SemType {
	return m.DefineMappingTypeWrappedWithEnvFieldsSemTypeCellMutability(env, fields, rest, CellMutability_CELL_MUT_LIMITED)
}

func (m *MappingDefinition) DefineMappingTypeWrappedWithEnvFieldsSemTypeCellMutability(env Env, fields []Field, rest SemType, mut CellMutability) SemType {
	var cellFields []CellField
	for _, field := range fields {
		ty := field.Ty
		var optTy SemType
		if field.Opt {
			optTy = Union(ty, UNDEF)
		} else {
			optTy = ty
		}
		var ro CellMutability
		if field.Ro {
			ro = CellMutability_CELL_MUT_NONE
		} else {
			ro = mut
		}
		cellFields = append(cellFields, cellFieldFrom(field.Name, cellContainingWithEnvSemTypeCellMutability(env, optTy, ro)))
	}
	var restMut CellMutability
	if IsNever(rest) {
		restMut = CellMutability_CELL_MUT_NONE
	} else {
		restMut = mut
	}
	restCell := cellContainingWithEnvSemTypeCellMutability(env, Union(rest, UNDEF), restMut)
	return m.Define(env, cellFields, restCell)
}

func (m *MappingDefinition) createSemType(env Env, atom atom) SemType {
	bdd := bddAtom(atom)
	s := getBasicSubtype(BTMapping, bdd)
	m.semType = s
	return s
}

func (m *MappingDefinition) splitFields(fields []CellField) splitField {
	sortedFields := make([]CellField, len(fields))
	copy(sortedFields, fields)
	// Arrays.sort(sortedFields, Comparator.comparing(MappingDefinition::fieldName))
	sort.Slice(sortedFields, func(i, j int) bool {
		return fieldName(sortedFields[i]) < fieldName(sortedFields[j])
	})
	names := make([]string, len(sortedFields))
	types := make([]SemType, len(sortedFields))
	for i, field := range sortedFields {
		names[i] = field.Name
		types[i] = field.Type
	}
	return splitFieldFrom(names, types)
}
