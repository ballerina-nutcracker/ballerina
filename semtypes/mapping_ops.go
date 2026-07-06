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

type mappingOps struct{}

var _ BasicTypeOps = &mappingOps{}

func mappingSubtypeIsEmpty(cx Context, t SubtypeData) bool {
	return memoSubtypeIsEmpty(cx, cx.mappingMemo(), func(cx Context, b Bdd) bool {
		return bddEvery(cx, b, conjunctionNil, conjunctionNil, mappingFormulaIsEmpty)
	}, t.(Bdd))
}

func mappingFormulaIsEmpty(cx Context, posList conjunctionHandle, negList conjunctionHandle) bool {
	var combined *MappingAtomicType
	if posList == conjunctionNil {
		combined = &MAPPING_ATOMIC_INNER
	} else {
		combined = cx.MappingAtomType(cx.conjunctionAtom(posList))
		p := cx.conjunctionNext(posList)
		for {
			if p == conjunctionNil {
				break
			} else {
				m := intersectMapping(cx.Env(), combined, cx.MappingAtomType(cx.conjunctionAtom(p)))
				if m == nil {
					return true
				} else {
					combined = m
				}
				p = cx.conjunctionNext(p)
			}
		}
		for i := range combined.Types {
			if IsEmpty(cx, combined.Types[i]) {
				return true
			}
		}
	}
	if !mappingInhabitedFast(cx, combined, negList) {
		return true
	}
	return (!mappingInhabited(cx, combined, negList))
}

func mappingInhabitedFast(cx Context, pos *MappingAtomicType, negList conjunctionHandle) bool {
	if negList == conjunctionNil {
		return true
	} else {
		neg := cx.MappingAtomType(cx.conjunctionAtom(negList))
		negNext := cx.conjunctionNext(negList)
		pairing := newFieldPairs(pos, neg)
		if !IsEmpty(cx, Diff(pos.Rest, neg.Rest)) {
			return mappingInhabitedFast(cx, pos, negNext)
		}
		for fieldPair := range pairing {
			intersect := Intersect(fieldPair.Type1, fieldPair.Type2)
			if IsEmpty(cx, intersect) {
				return mappingInhabitedFast(cx, pos, negNext)
			}
			d := Diff(fieldPair.Type1, fieldPair.Type2)
			if !IsEmpty(cx, d) {
				return mappingInhabitedFast(cx, pos, negNext)
			}
		}
		return false
	}
}

func mappingInhabited(cx Context, pos *MappingAtomicType, negList conjunctionHandle) bool {
	if negList == conjunctionNil {
		return true
	} else {
		neg := cx.MappingAtomType(cx.conjunctionAtom(negList))
		negNext := cx.conjunctionNext(negList)
		pairing := newFieldPairs(pos, neg)
		if !IsEmpty(cx, Diff(pos.Rest, neg.Rest)) {
			return mappingInhabited(cx, pos, negNext)
		}
		for fieldPair := range pairing {
			intersect := Intersect(fieldPair.Type1, fieldPair.Type2)
			if IsEmpty(cx, intersect) {
				return mappingInhabited(cx, pos, negNext)
			}
			d := Diff(fieldPair.Type1, fieldPair.Type2)
			if !IsEmpty(cx, d) {
				var mt MappingAtomicType
				if fieldPair.Index1 < 0 {
					mt = insertField(*pos, fieldPair.Name, d)
				} else {
					posTypes := append([]SemType(nil), pos.Types...)
					posTypes[fieldPair.Index1] = d
					mt = mappingAtomicTypeFrom(pos.Names, posTypes, pos.Rest)
				}
				if mappingInhabited(cx, &mt, negNext) {
					return true
				}
			}
		}
		return false
	}
}

func insertField(m MappingAtomicType, name string, t SemType) MappingAtomicType {
	names := append([]string(nil), m.Names...)
	names = append(names, "")
	types := append([]SemType(nil), m.Types...)
	types = append(types, SemType{})
	i := len(names) - 1
	for {
		if (i == 0) || codePointCompare(names[i-1], name) {
			names[i] = name
			types[i] = t
			break
		}
		names[i] = names[i-1]
		types[i] = types[i-1]
		i = (i - 1)
	}
	return mappingAtomicTypeFrom(names, types, m.Rest)
}

func intersectMapping(env Env, m1 *MappingAtomicType, m2 *MappingAtomicType) *MappingAtomicType {
	var names []string
	var types []SemType
	pairing := newFieldPairs(m1, m2)
	for fieldPair := range pairing {
		names = append(names, fieldPair.Name)
		t := intersectMemberSemTypes(env, fieldPair.Type1, fieldPair.Type2)
		if IsNever(cellInner(fieldPair.Type1)) {
			return nil
		}
		types = append(types, t)
	}
	rest := intersectMemberSemTypes(env, m1.Rest, m2.Rest)
	return new(mappingAtomicTypeFrom(names, types, rest))
}

func bddMappingMemberTypeInnerCore(cx Context, b Bdd, key SubtypeData, accum SemType) SemType {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return accum
		}
		return NEVER
	} else {
		bn := b.(bddNode)
		return Union(bddMappingMemberTypeInnerCore(cx, bn.left(), key, Intersect(mappingAtomicMemberTypeInner(*cx.MappingAtomType(bn.atom()), key), accum)), Union(bddMappingMemberTypeInnerCore(cx, bn.middle(), key, accum), bddMappingMemberTypeInnerCore(cx, bn.right(), key, accum)))
	}
}

func mappingAtomicMemberTypeInner(atomic MappingAtomicType, key SubtypeData) SemType {
	var memberType SemType
	for _, ty := range mappingAtomicApplicableMemberTypesInner(atomic, key) {
		if IsZero(memberType) {
			memberType = ty
		} else {
			memberType = Union(memberType, ty)
		}
	}
	if IsZero(memberType) {
		return UNDEF
	}
	return memberType
}

func mappingAtomicApplicableMemberTypesInner(atomic MappingAtomicType, key SubtypeData) []SemType {
	var types []SemType
	for i := range atomic.Types {
		types = append(types, cellInner(atomic.Types[i]))
	}
	var memberTypes []SemType
	rest := cellInner(atomic.Rest)
	if isAllSubtype(key) {
		memberTypes = append(memberTypes, types...)
		memberTypes = append(memberTypes, rest)
	} else {
		coverage := getStringSubtypeListCoverage(key.(stringSubtype), atomic.Names)
		for _, index := range coverage.Indices {
			memberTypes = append(memberTypes, types[index])
		}
		if !coverage.IsSubtype {
			memberTypes = append(memberTypes, rest)
		}
	}
	return memberTypes
}

func newMappingOps() mappingOps {
	return mappingOps{}
}

func (m *mappingOps) Union(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeUnion(d1, d2)
}

func (m *mappingOps) Intersect(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeIntersect(d1, d2)
}

func (m *mappingOps) Diff(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeDiff(d1, d2)
}

func (m *mappingOps) complement(d SubtypeData) SubtypeData {
	return bddSubtypeComplement(d)
}

func (m *mappingOps) IsEmpty(cx Context, d SubtypeData) bool {
	return mappingSubtypeIsEmpty(cx, d)
}
