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

func listProjInnerVal(cx Context, t SemType, k SemType) SemType {
	if t.some() == 0 {
		if (t.all() & LIST.all()) != 0 {
			return VAL
		}
		return NEVER
	}
	keyData := getIntSubtype(k)
	if isNothingSubtype(keyData) {
		return NEVER
	}
	return listProjBddInnerVal(cx, keyData, getComplexSubtypeData(t, BTList).(Bdd), conjunctionNil, conjunctionNil)
}

func listProjBddInnerVal(cx Context, k SubtypeData, b Bdd, pos conjunctionHandle, neg conjunctionHandle) SemType {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return listProjPathInnerVal(cx, k, pos, neg)
		} else {
			return NEVER
		}
	} else {
		bn := b.(bddNode)
		saved := cx.conjunctionStackDepth()
		result := Union(listProjBddInnerVal(cx, k, bn.left(), cx.pushConjunction(bn.atom(), pos), neg),
			Union(listProjBddInnerVal(cx, k, bn.middle(), pos, cx.pushConjunction(bn.atom(), neg)),
				listProjBddInnerVal(cx, k, bn.right(), pos, cx.pushConjunction(bn.atom(), neg))))
		cx.resetConjunctionStack(saved)
		return result
	}
}

func listProjPathInnerVal(cx Context, k SubtypeData, pos conjunctionHandle, neg conjunctionHandle) SemType {
	var members fixedLengthArray
	var rest SemType
	if pos == conjunctionNil {
		members = fixedLengthArrayEmpty()
		rest = cellContaining(cx.Env(), Union(VAL, UNDEF))
	} else {
		// combine all the positive tuples using intersection
		lt := cx.ListAtomType(cx.conjunctionAtom(pos))
		members = lt.Members
		rest = lt.rest
		p := cx.conjunctionNext(pos)
		// the neg case is in case we grow the array in listInhabited
		if p != conjunctionNil || neg != conjunctionNil {
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
					return NEVER
				}
				members = intersectedMembers
				rest = intersectedRest
			}
		}
		if fixedArrayAnyEmpty(cx, members) {
			return NEVER
		}
		// Ensure that we can use isNever on rest in listInhabited
		if !IsNever(cellInnerVal(rest)) && IsEmpty(cx, rest) {
			rest = roCellContaining(cx.Env(), NEVER)
		}
	}
	// return listProjExclude(cx, k, members, rest, listConjunction(cx, neg));
	indices := listSamples(cx, members, rest, neg)
	projIndices, keyIndices := listProjSamples(indices, k)
	sampleTypes, nRequired := listSampleTypes(cx, members, rest, projIndices)
	return listProjExcludeInnerVal(cx, projIndices, keyIndices, sampleTypes, nRequired, neg)
}

func listProjSamples(indices []int, k SubtypeData) ([]int, []int) {
	type indexBoolPair struct {
		index   int
		isInKey bool
	}
	var v []indexBoolPair
	for _, i := range indices {
		v = append(v, indexBoolPair{i, intSubtypeContains(k, int64(i))})
	}
	if intSubtype, ok := k.(*intSubtype); ok {
		for _, rng := range intSubtype.Ranges {
			max := rng.Max
			if rng.Max >= 0 {
				v = append(v, indexBoolPair{int(max), true})
				var min int
				if 0 > int(rng.Min) {
					min = 0
				} else {
					min = int(rng.Min)
				}
				if min < int(max) {
					v = append(v, indexBoolPair{min, true})
				}
			}
		}
	}
	sort.Slice(v, func(i, j int) bool {
		return v[i].index < v[j].index
	})
	var indices1 []int
	var keyIndices []int
	for _, ib := range v {
		if len(indices1) == 0 || ib.index != indices1[len(indices1)-1] {
			if ib.isInKey {
				keyIndices = append(keyIndices, len(indices1))
			}
			indices1 = append(indices1, ib.index)
		}
	}
	return indices1, keyIndices
}

func listProjExcludeInnerVal(cx Context, indices []int, keyIndices []int, memberTypes []SemType, nRequired int, neg conjunctionHandle) SemType {
	var p = NEVER
	if neg == conjunctionNil {
		length := len(memberTypes)
		for _, k := range keyIndices {
			if k < length {
				p = Union(p, cellInnerVal(memberTypes[k]))
			}
		}
	} else {
		nt := cx.ListAtomType(cx.conjunctionAtom(neg))
		negNext := cx.conjunctionNext(neg)
		if nRequired > 0 && IsNever(listMemberAtInnerVal(nt.Members, nt.rest, indices[nRequired-1])) {
			return listProjExcludeInnerVal(cx, indices, keyIndices, memberTypes, nRequired, negNext)
		}
		negLen := nt.Members.FixedLength
		if negLen > 0 {
			length := len(memberTypes)
			if length < len(indices) && indices[length] < negLen {
				return listProjExcludeInnerVal(cx, indices, keyIndices, memberTypes, nRequired, negNext)
			}
			for i := nRequired; i < len(memberTypes); i++ {
				if indices[i] >= negLen {
					break
				}
				t := append([]SemType(nil), memberTypes[0:i]...)
				p = Union(p, listProjExcludeInnerVal(cx, indices, keyIndices, t, nRequired, negNext))
			}
		}
		for i := range memberTypes {
			d := Diff(cellInnerVal(memberTypes[i]), listMemberAtInnerVal(nt.Members, nt.rest, indices[i]))
			if !IsEmpty(cx, d) {
				t := append([]SemType(nil), memberTypes...)
				t[i] = cellContaining(cx.Env(), d)
				var maxVal int
				if nRequired > (i + 1) {
					maxVal = nRequired
				} else {
					maxVal = i + 1
				}
				p = Union(p, listProjExcludeInnerVal(cx, indices, keyIndices, t, maxVal, negNext))
			}
		}
	}
	return p
}
