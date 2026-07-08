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
)

type MappingAtomicType struct {
	Names []string
	Types []SemType
	Rest  SemType
}

var _ atomicType = &MappingAtomicType{}

func mappingAtomicTypeFrom(names []string, types []SemType, rest SemType) MappingAtomicType {
	return MappingAtomicType{
		Names: names,
		Types: types,
		Rest:  rest,
	}
}

func (m *MappingAtomicType) atomKind() kind {
	return kind_MAPPING_ATOM
}

func (m *MappingAtomicType) FieldInnerVal(name string) SemType {
	for i, n := range m.Names {
		if n == name {
			return cellInnerVal(m.Types[i])
		}
	}
	return cellInnerVal(m.Rest)
}

func (m *MappingAtomicType) IsOptional(cx Context, name string) bool {
	for i, n := range m.Names {
		if n == name {
			return IsSubtype(cx, UNDEF, cellInner(m.Types[i]))
		}
	}
	return true
}

type matchQuantifier int

const (
	matchAny matchQuantifier = iota
	matchAll
)

func AnyMappingAtomHasFieldByName(cx Context, ty SemType, key string) bool {
	return mappingAtomsMatch(cx, ty, matchAny, func(_ Context, atom *MappingAtomicType) bool {
		return mappingAtomHasFieldByName(atom, key)
	})
}

func AllMappingAtomHasFieldByName(cx Context, ty SemType, key string) bool {
	// I think this is fine, but may have problems with narrowing. Spec describes unions only assuming positive atoms
	return mappingAtomsMatch(cx, ty, matchAll, func(_ Context, atom *MappingAtomicType) bool {
		return mappingAtomHasFieldByName(atom, key)
	})
}

func AllMappingAtomsHaveOptionalFieldByName(cx Context, ty SemType, key string) bool {
	return mappingAtomsMatch(cx, ty, matchAll, func(cx Context, atom *MappingAtomicType) bool {
		return mappingAtomHasOptionalFieldByName(cx, atom, key)
	})
}

func mappingAtomsMatch(cx Context, ty SemType, quantifier matchQuantifier, predicate func(Context, *MappingAtomicType) bool) bool {
	if !IsSubtypeSimple(ty, MAPPING) {
		return false
	}
	if ty.some() == 0 {
		return false
	}
	bdd := getComplexSubtypeData(ty, BTMapping).(Bdd)
	if simple, ok := bdd.(*bddNodeSimple); ok {
		return predicate(cx, cx.MappingAtomType(simple.atom()))
	}

	return bddMappingAtomsMatch(cx, bdd, quantifier, predicate)
}

func bddMappingAtomsMatch(cx Context, bdd Bdd, quantifier matchQuantifier, predicate func(Context, *MappingAtomicType) bool) bool {
	switch quantifier {
	case matchAny:
		found := false
		bddEvery(cx, bdd, conjunctionNil, conjunctionNil, func(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
			_ = neg
			for h := pos; h != conjunctionNil; h = cx.conjunctionNext(h) {
				if predicate(cx, cx.MappingAtomType(cx.conjunctionAtom(h))) {
					found = true
					return false
				}
			}
			return true
		})
		return found
	case matchAll:
		return bddEvery(cx, bdd, conjunctionNil, conjunctionNil, func(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
			_ = neg
			for h := pos; h != conjunctionNil; h = cx.conjunctionNext(h) {
				if !predicate(cx, cx.MappingAtomType(cx.conjunctionAtom(h))) {
					return false
				}
			}
			return true
		})
	}
	panic("unreachable")
}

func mappingAtomHasFieldByName(atom *MappingAtomicType, key string) bool {
	return slices.Contains(atom.Names, key)
}

func mappingAtomHasOptionalFieldByName(_ Context, atom *MappingAtomicType, key string) bool {
	for i, n := range atom.Names {
		if n == key {
			return ContainsUndef(cellInner(atom.Types[i]))
		}
	}
	return false
}
