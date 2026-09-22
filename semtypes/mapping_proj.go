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

func MappingMemberTypeInnerValProj(cx Context, t SemType, k SemType) SemType {
	return Diff(mappingMemberTypeInner(cx, t, k), Undef)
}

// This computes the spec operation called "member type of K in T",
// for when T is a subtype of mapping, and K is either `string` or a singleton string.
// This is what Castagna calls projection.
func mappingMemberTypeInner(cx Context, t SemType, k SemType) SemType {
	if t.some() == 0 {
		if (t.all() & Mapping.all()) != 0 {
			return Val
		}
		return Undef
	}
	keyData := getStringSubtype(k)
	if isNothingSubtype(keyData) {
		return Undef
	}
	return bddMappingMemberTypeInner(cx, getComplexSubtypeData(t, btMapping).(bdd), keyData, Inner)
}

func bddMappingMemberTypeInner(cx Context, b bdd, key subtypeData, accum SemType) SemType {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return accum
		} else {
			return Never
		}
	} else {
		bn := b.(bddNode)
		if !isPositiveAtom(bn.atom()) {
			return Union(
				bddMappingMemberTypeInner(cx, bn.left(), key, accum),
				Union(bddMappingMemberTypeInner(cx, bn.middle(), key, accum),
					bddMappingMemberTypeInner(cx, bn.right(), key, accum)))
		}
		return Union(
			bddMappingMemberTypeInner(cx, bn.left(), key,
				Intersect(mappingAtomicMemberTypeInnerProj(cx.MappingAtomType(bn.atom()), key), accum)),
			Union(bddMappingMemberTypeInner(cx, bn.middle(), key, accum),
				bddMappingMemberTypeInner(cx, bn.right(), key, accum)))
	}
}

func mappingAtomicMemberTypeInnerProj(atomic *MappingAtomicType, key subtypeData) SemType {
	var memberType SemType
	for _, ty := range mappingAtomicApplicableMemberTypesInnerProj(atomic, key) {
		if IsZero(memberType) {
			memberType = ty
		} else {
			memberType = Union(memberType, ty)
		}
	}
	if IsZero(memberType) {
		return Undef
	} else {
		return memberType
	}
}

func mappingAtomicApplicableMemberTypesInnerProj(atomic *MappingAtomicType, key subtypeData) []SemType {
	types := make([]SemType, len(atomic.types))
	for i := range atomic.types {
		types[i] = cellInner(atomic.types[i])
	}

	var memberTypes []SemType
	rest := cellInner(atomic.rest)
	if isAllSubtype(key) {
		memberTypes = append(memberTypes, types...)
		memberTypes = append(memberTypes, rest)
	} else {
		coverage := getStringSubtypeListCoverage(key.(stringSubtype), atomic.names)
		for _, index := range coverage.Indices {
			memberTypes = append(memberTypes, types[index])
		}
		if !coverage.IsSubtype {
			memberTypes = append(memberTypes, rest)
		}
	}
	return memberTypes
}
