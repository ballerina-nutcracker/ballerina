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

type ListDefinition struct {
	rec     *recAtom
	semType SemType
}

var _ Definition = &ListDefinition{}

func NewListDefinition() ListDefinition {
	this := ListDefinition{}
	this.rec = nil
	this.semType = SemType{}
	// Default field initializations

	return this
}

func (l *ListDefinition) GetSemType(env Env) SemType {
	s := l.semType
	if IsZero(s) {
		rec := env.recListAtom()
		l.rec = &rec
		return l.createSemType(env, &rec)
	} else {
		return s
	}
}

func (l *ListDefinition) TupleTypeWrapped(env Env, members ...SemType) SemType {
	return l.DefineListTypeWrappedWithEnvSemTypesInt(env, members, len(members))
}

func (l *ListDefinition) TupleTypeWrappedRo(env Env, members ...SemType) SemType {
	return l.DefineListTypeWrapped(env, members, len(members), NEVER, CellMutability_CELL_MUT_NONE)
}

func (l *ListDefinition) DefineListTypeWrapped(env Env, initial []SemType, fixedLength int, rest SemType, mut CellMutability) SemType {
	var initialCells []SemType
	for _, member := range initial {
		initialCells = append(initialCells, cellContainingWithEnvSemTypeCellMutability(env, member, mut))
	}
	var restMut CellMutability
	if IsNever(rest) {
		restMut = CellMutability_CELL_MUT_NONE
	} else {
		restMut = mut
	}
	restCell := cellContainingWithEnvSemTypeCellMutability(env, Union(rest, UNDEF), restMut)
	return l.define(env, initialCells, fixedLength, restCell)
}

func (l *ListDefinition) DefineListTypeWrappedWithEnvSemTypesInt(env Env, initial []SemType, size int) SemType {
	return l.DefineListTypeWrapped(env, initial, size, NEVER, CellMutability_CELL_MUT_LIMITED)
}

func (l *ListDefinition) DefineListTypeWrappedWithEnvSemTypesIntSemType(env Env, initial []SemType, fixedLength int, rest SemType) SemType {
	return l.DefineListTypeWrapped(env, initial, fixedLength, rest, CellMutability_CELL_MUT_LIMITED)
}

func (l *ListDefinition) DefineListTypeWrappedWithEnvSemType(env Env, rest SemType) SemType {
	return l.DefineListTypeWrappedWithEnvSemTypesIntSemType(env, nil, 0, rest)
}

func (l *ListDefinition) DefineListTypeWrappedWithEnvSemTypeCellMutability(env Env, rest SemType, mut CellMutability) SemType {
	return l.DefineListTypeWrapped(env, nil, 0, rest, mut)
}

func (l *ListDefinition) DefineListTypeWrappedWithEnvSemTypesSemType(env Env, initial []SemType, rest SemType) SemType {
	return l.DefineListTypeWrapped(env, initial, len(initial), rest, CellMutability_CELL_MUT_LIMITED)
}

func (l *ListDefinition) define(env Env, initial []SemType, fixedLength int, rest SemType) SemType {
	members := l.fixedLengthNormalize(fixedLengthArrayFrom(initial, fixedLength))
	atomicType := listAtomicTypeFrom(members, rest)
	var atom atom
	rec := l.rec
	if rec != nil {
		atom = rec
		env.setRecListAtomType(*rec, &atomicType)
	} else {
		atom = env.listAtom(&atomicType)
	}
	return l.createSemType(env, atom)
}

func (l *ListDefinition) fixedLengthNormalize(array fixedLengthArray) fixedLengthArray {
	initial := array.initial
	i := (len(initial) - 1)
	if i <= 0 {
		return array
	}
	last := initial[i]
	i = (i - 1)
	for i >= 0 {
		if !sameComplexSemType(last, initial[i]) {
			break
		}
		i = (i - 1)
	}
	return fixedLengthArrayFrom(initial[:i+2], array.FixedLength)
}

func (l *ListDefinition) createSemType(env Env, atom atom) SemType {
	bdd := bddAtom(atom)
	SemType := getBasicSubtype(BTList, bdd)
	l.semType = SemType
	return SemType
}
