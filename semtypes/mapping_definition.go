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

type mappingDefinitionOptions struct {
	mutability CellMutability
}

type MappingDefinitionOption func(*mappingDefinitionOptions)

func MappingMutability(mutability CellMutability) MappingDefinitionOption {
	return func(options *mappingDefinitionOptions) {
		options.mutability = mutability
	}
}

var _ Definition = &MappingDefinition{}

func fieldName(f cellField) string {
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
	m.semType = Never
}

func (m *MappingDefinition) Define(env Env, fields []Field, rest SemType, options ...MappingDefinitionOption) SemType {
	opts := mappingDefinitionOptions{mutability: CellMutabilityLimited}
	for _, option := range options {
		option(&opts)
	}

	cellFields := make([]cellField, 0, len(fields))
	for _, field := range fields {
		ty := field.typeOf
		if field.optional {
			ty = Union(ty, Undef)
		}
		mutability := opts.mutability
		if field.readonly {
			mutability = CellMutabilityNone
		}
		cellFields = append(cellFields, cellFieldFrom(field.name,
			cellContainingWithEnvSemTypeCellMutability(env, ty, mutability)))
	}
	restMutability := opts.mutability
	if IsNever(rest) {
		restMutability = CellMutabilityNone
	}
	restCell := cellContainingWithEnvSemTypeCellMutability(env, Union(rest, Undef), restMutability)
	return m.defineFromCells(env, cellFields, restCell)
}

func (m *MappingDefinition) defineFromCells(env Env, fields []cellField, rest SemType) SemType {
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

func (m *MappingDefinition) createSemType(env Env, atom atom) SemType {
	bdd := bddAtom(atom)
	s := getBasicSubtype(btMapping, bdd)
	m.semType = s
	return s
}

func (m *MappingDefinition) splitFields(fields []cellField) splitField {
	sortedFields := make([]cellField, len(fields))
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
