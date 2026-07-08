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
	"math"
	"sort"
)

type listOps struct{}

var _ BasicTypeOps = &listOps{}

func listSubtypeIsEmpty(cx Context, t SubtypeData) bool {
	return memoSubtypeIsEmpty(cx, cx.listMemo(), func(cx Context, b Bdd) bool {
		return bddEvery(cx, b, conjunctionNil, conjunctionNil, listFormulaIsEmpty)
	}, t.(Bdd))
}

func listFormulaIsEmpty(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
	var members fixedLengthArray
	var rest SemType
	if pos == conjunctionNil {
		atom := LIST_ATOMIC_INNER
		members = atom.Members
		rest = atom.rest
	} else {
		// combine all the positive tuples using intersection
		lt := cx.ListAtomType(cx.conjunctionAtom(pos))
		members = lt.Members
		rest = lt.rest
		p := cx.conjunctionNext(pos)
		// the neg case is in case we grow the array in listInhabited
		if p != conjunctionNil || neg != conjunctionNil {
			// Jbal note: we don't need this as we already created copies when converting from array to list.
			// Just keeping this for the sake of source similarity between Bal code and Java.
			members = fixedArrayShallowCopy(members)
		}
		for {
			if p == conjunctionNil {
				break
			} else {
				d := cx.conjunctionAtom(p)
				p = cx.conjunctionNext(p)
				lt = cx.ListAtomType(d)
				intersectedMembers, intersectedRest, ok := listIntersectWith(cx.Env(), members, rest, lt.Members, lt.rest)
				if !ok {
					return true
				}
				members = intersectedMembers
				rest = intersectedRest
			}
		}
		if fixedArrayAnyEmpty(cx, members) {
			return true
		}
	}
	indices := listSamples(cx, members, rest, neg)
	memberTypes, nRequired := listSampleTypes(cx, members, rest, indices)
	memberTypesArray := make([]SemType, len(memberTypes))
	copy(memberTypesArray, memberTypes)
	if !listInhabitedFast(cx, indices, memberTypesArray, nRequired, neg) {
		// assert !listInhabited(cx, indices, memberTypes, nRequired, neg)
		return true
	}
	return !listInhabited(cx, indices, memberTypesArray, nRequired, neg)
}

func listInhabitedFast(cx Context, indices []int, memberTypes []SemType, nRequired int, neg conjunctionHandle) bool {
	if neg == conjunctionNil {
		return true
	}
	nt := cx.ListAtomType(cx.conjunctionAtom(neg))
	negNext := cx.conjunctionNext(neg)
	if nRequired > 0 && IsNever(listMemberAtInnerVal(nt.Members, nt.rest, indices[nRequired-1])) {
		return listInhabitedFast(cx, indices, memberTypes, nRequired, negNext)
	}
	negLen := nt.Members.FixedLength
	if negLen > 0 {
		for i := range memberTypes {
			index := indices[i]
			if index >= negLen {
				break
			}
			negMemberType := listMemberAt(nt.Members, nt.rest, index)
			common := Intersect(memberTypes[i], negMemberType)
			if IsEmpty(cx, common) {
				return listInhabitedFast(cx, indices, memberTypes, nRequired, negNext)
			}
		}
		lenMemberTypes := len(memberTypes)
		if lenMemberTypes < len(indices) && indices[lenMemberTypes] < negLen {
			return listInhabitedFast(cx, indices, memberTypes, nRequired, negNext)
		}

		for i := nRequired; i < len(memberTypes); i++ {
			if indices[i] >= negLen {
				break
			}
			t := memberTypes[:i]
			if listInhabitedFast(cx, indices, t, nRequired, negNext) {
				return true
			}
		}
	}
	for i := range memberTypes {
		d := Diff(memberTypes[i], listMemberAt(nt.Members, nt.rest, indices[i]))
		if !IsEmpty(cx, d) {
			return listInhabitedFast(cx, indices, memberTypes, nRequired, negNext)
		}
	}
	return false
}

func listSampleTypes(cx Context, members fixedLengthArray, rest SemType, indices []int) ([]SemType, int) {
	var memberTypes []SemType
	nRequired := 0
	for i := range indices {
		index := indices[i]
		t := cellContainingInnerVal(cx.Env(), listMemberAt(members, rest, index))
		if IsEmpty(cx, t) {
			break
		}
		memberTypes = append(memberTypes, t)
		if index < members.FixedLength {
			nRequired = i + 1
		}
	}
	return memberTypes, nRequired
}

func listSamples(cx Context, members fixedLengthArray, rest SemType, neg conjunctionHandle) []int {
	maxInitialLength := len(members.initial)
	var fixedLengths []int
	fixedLengths = append(fixedLengths, members.FixedLength)
	tem := neg
	nNeg := 0
	for {
		if tem != conjunctionNil {
			lt := cx.ListAtomType(cx.conjunctionAtom(tem))
			m := lt.Members
			if len(m.initial) > maxInitialLength {
				maxInitialLength = len(m.initial)
			}
			if m.FixedLength > maxInitialLength {
				fixedLengths = append(fixedLengths, m.FixedLength)
			}
			nNeg = nNeg + 1
			tem = cx.conjunctionNext(tem)
		} else {
			break
		}
	}
	sort.Ints(fixedLengths)
	var boundaries []int
	for i := 1; i <= maxInitialLength; i++ {
		boundaries = append(boundaries, i)
	}
	for _, n := range fixedLengths {
		if len(boundaries) == 0 || n > boundaries[len(boundaries)-1] {
			boundaries = append(boundaries, n)
		}
	}
	var indices []int
	lastBoundary := 0
	if nNeg == 0 {
		nNeg = 1
	}
	for _, b := range boundaries {
		segmentLength := b - lastBoundary
		nSamples := min(nNeg, segmentLength)
		for i := b - nSamples; i < b; i++ {
			indices = append(indices, i)
		}
		lastBoundary = b
	}
	for i := 0; i < nNeg; i++ {
		if lastBoundary > math.MaxInt-i {
			break
		}
		indices = append(indices, lastBoundary+i)
	}
	return indices
}

func listIntersectWith(env Env, members1 fixedLengthArray, rest1 SemType,
	members2 fixedLengthArray, rest2 SemType,
) (fixedLengthArray, SemType, bool) {
	if listLengthsDisjoint(members1, rest1, members2, rest2) {
		return fixedLengthArray{}, SemType{}, false
	}
	// This is different from nBallerina, but I think assuming we have normalized the FixedLengthArrays we must
	// consider fixedLengths not the size of initial members. For example consider any[4] and
	// [int, string, float...]. If we don't consider the fixedLength in the initial part we'll consider only the
	// first two elements and rest will compare essentially 5th element, meaning we are ignoring 3 and 4 elements
	max1 := members1.FixedLength
	max2 := members2.FixedLength
	maxLen := max(max2, max1)
	var initial []SemType
	for i := range maxLen {
		intersected := intersectMemberSemTypes(env, listMemberAt(members1, rest1, i),
			listMemberAt(members2, rest2, i))
		initial = append(initial, intersected)
	}
	fixedLen := max(members2.FixedLength, members1.FixedLength)
	return fixedLengthArrayFrom(initial, fixedLen), intersectMemberSemTypes(env, rest1, rest2), true
}

func fixedArrayShallowCopy(array fixedLengthArray) fixedLengthArray {
	return fixedLengthArrayFrom(array.initial, array.FixedLength)
}

func listInhabited(cx Context, indices []int, memberTypes []SemType, nRequired int, neg conjunctionHandle) bool {
	if neg == conjunctionNil {
		return true
	} else {
		nt := cx.ListAtomType(cx.conjunctionAtom(neg))
		negNext := cx.conjunctionNext(neg)
		if nRequired > 0 && IsNever(listMemberAtInnerVal(nt.Members, nt.rest, indices[nRequired-1])) {
			return listInhabited(cx, indices, memberTypes, nRequired, negNext)
		}
		negLen := nt.Members.FixedLength
		if negLen > 0 {
			for i := range memberTypes {
				index := indices[i]
				if index >= negLen {
					break
				}
				negMemberType := listMemberAt(nt.Members, nt.rest, index)
				common := Intersect(memberTypes[i], negMemberType)
				if IsEmpty(cx, common) {
					return listInhabited(cx, indices, memberTypes, nRequired, negNext)
				}
			}
			lenMemberTypes := len(memberTypes)
			if lenMemberTypes < len(indices) && indices[lenMemberTypes] < negLen {
				return listInhabited(cx, indices, memberTypes, nRequired, negNext)
			}
			for i := nRequired; i < len(memberTypes); i++ {
				if indices[i] >= negLen {
					break
				}
				t := memberTypes[:i]
				if listInhabited(cx, indices, t, nRequired, negNext) {
					return true
				}
			}
		}
		for i := range memberTypes {
			d := Diff(memberTypes[i], listMemberAt(nt.Members, nt.rest, indices[i]))
			if !IsEmpty(cx, d) {
				// Clone the slice
				t := make([]SemType, len(memberTypes))
				copy(t, memberTypes)
				t[i] = d
				nReq := max(i+1, nRequired)
				if listInhabited(cx, indices, t, nReq, negNext) {
					return true
				}
			}
		}
		return false
	}
}

func listMemberAtInnerVal(fixedArray fixedLengthArray, rest SemType, index int) SemType {
	return cellInnerVal(listMemberAt(fixedArray, rest, index))
}

func listLengthsDisjoint(members1 fixedLengthArray, rest1 SemType, members2 fixedLengthArray, rest2 SemType) bool {
	len1 := members1.FixedLength
	len2 := members2.FixedLength
	if len1 < len2 {
		return IsNever(cellInnerVal(rest1))
	}
	if len2 < len1 {
		return IsNever(cellInnerVal(rest2))
	}
	return false
}

func listMemberAt(fixedArray fixedLengthArray, rest SemType, index int) SemType {
	if index < fixedArray.FixedLength {
		return fixedArrayGet(fixedArray, index)
	}
	return rest
}

func fixedArrayAnyEmpty(cx Context, array fixedLengthArray) bool {
	for i := range array.initial {
		if IsEmpty(cx, array.initial[i]) {
			return true
		}
	}
	return false
}

func fixedArrayGet(members fixedLengthArray, index int) SemType {
	memberLen := len(members.initial)
	i := min(memberLen-1, index)
	return members.initial[i]
}

func listAtomicMemberTypeInnerVal(atomic ListAtomicType, key SubtypeData) SemType {
	return Diff(listAtomicMemberTypeInner(atomic, key), UNDEF)
}

func listAtomicMemberTypeInner(atomic ListAtomicType, key SubtypeData) SemType {
	return listAtomicMemberTypeAtInner(atomic.Members, atomic.rest, key)
}

func listAtomicMemberTypeAtInner(fixedArray fixedLengthArray, rest SemType, key SubtypeData) SemType {
	if intSubtype, ok := key.(intSubtype); ok {
		var m SemType
		m = NEVER
		initLen := len(fixedArray.initial)
		fixedLen := fixedArray.FixedLength
		if fixedLen != 0 {
			for i := range initLen {
				if intSubtypeContains(key, int64(i)) {
					m = Union(m, cellInner(fixedArrayGet(fixedArray, i)))
				}
			}
			if intSubtypeOverlapRange(intSubtype, rangeFrom(int64(initLen), int64(fixedLen-1))) {
				m = Union(m, cellInner(fixedArrayGet(fixedArray, fixedLen-1)))
			}
		}
		if fixedLen == 0 || intSubtypeMax(intSubtype) > int64(fixedLen-1) {
			m = Union(m, cellInner(rest))
		}
		return m
	}
	m := cellInner(rest)
	if fixedArray.FixedLength > 0 {
		for i := range fixedArray.initial {
			m = Union(m, cellInner(fixedArray.initial[i]))
		}
	}
	return m
}

func bddListMemberTypeInnerVal(cx Context, b Bdd, key SubtypeData, accum SemType) SemType {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return accum
		}
		return NEVER
	} else {
		bn := b.(bddNode)
		return Union(bddListMemberTypeInnerVal(cx, bn.left(), key, Intersect(listAtomicMemberTypeInnerVal(*cx.ListAtomType(bn.atom()), key), accum)), Union(bddListMemberTypeInnerVal(cx, bn.middle(), key, accum), bddListMemberTypeInnerVal(cx, bn.right(), key, accum)))
	}
}

func newListOps() listOps {
	this := listOps{}
	return this
}

func (l *listOps) Union(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeUnion(d1, d2)
}

func (l *listOps) Intersect(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeIntersect(d1, d2)
}

func (l *listOps) Diff(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeDiff(d1, d2)
}

func (l *listOps) complement(d SubtypeData) SubtypeData {
	return bddSubtypeComplement(d)
}

func (l *listOps) IsEmpty(cx Context, d SubtypeData) bool {
	return listSubtypeIsEmpty(cx, d)
}
