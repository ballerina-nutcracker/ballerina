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

type (
	bddPredicate        func(cx Context, posList conjunctionHandle, negList conjunctionHandle) bool
	bddIsEmptyPredicate func(cx Context, b Bdd) bool
)

func bddEvery(cx Context, b Bdd, pos conjunctionHandle, neg conjunctionHandle, predicate bddPredicate) bool {
	saved := cx.conjunctionStackDepth()
	defer cx.resetConjunctionStack(saved)
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		return !allOrNothing.IsAll() || predicate(cx, pos, neg)
	} else {
		bn := b.(bddNode)
		result := bddEvery(cx, bn.left(), cx.pushConjunction(bn.atom(), pos), neg, predicate) &&
			bddEvery(cx, bn.middle(), pos, neg, predicate) &&
			bddEvery(cx, bn.right(), pos, cx.pushConjunction(bn.atom(), neg), predicate)
		return result
	}
}

func bddEveryPositive(cx Context, b Bdd, pos conjunctionHandle, neg conjunctionHandle, predicate bddPredicate) bool {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		return !allOrNothing.IsAll() || predicate(cx, pos, neg)
	} else {
		bn := b.(bddNode)
		saved := cx.conjunctionStackDepth()
		result := bddEveryPositive(cx, bn.left(), andIfPositive(cx, bn.atom(), pos), neg, predicate) &&
			bddEveryPositive(cx, bn.middle(), pos, neg, predicate) &&
			bddEveryPositive(cx, bn.right(), pos, andIfPositive(cx, bn.atom(), neg), predicate)
		cx.resetConjunctionStack(saved)
		return result
	}
}

func andIfPositive(cx Context, atom atom, next conjunctionHandle) conjunctionHandle {
	if !isPositiveAtom(atom) {
		return next
	}
	return cx.pushConjunction(atom, next)
}

func bddPosMaybeEmpty(b Bdd) bool {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		return allOrNothing.IsAll()
	} else {
		bn := b.(bddNode)
		return bddPosMaybeEmpty(bn.middle()) || bddPosMaybeEmpty(bn.right())
	}
}

func bddSubtypeUnion(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	return bddUnion(t1.(Bdd), t2.(Bdd))
}

func bddSubtypeIntersect(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	return bddIntersect(t1.(Bdd), t2.(Bdd))
}

func bddSubtypeDiff(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	return bddDiff(t1.(Bdd), t2.(Bdd))
}

func bddSubtypeComplement(t SubtypeData) SubtypeData {
	return bddComplement(t.(Bdd))
}

func notIsEmpty(cx Context, t SubtypeData) bool {
	return false
}

func codePointCompare(s1 string, s2 string) bool {
	if s1 == s2 {
		return false
	}
	len1 := len(s1)
	len2 := len(s2)
	if len1 < len2 && s2[:len1] == s1 {
		return true
	}
	r1 := []rune(s1)
	r2 := []rune(s2)
	for cp := 0; cp < len(r1) && cp < len(r2); {
		if r1[cp] == r2[cp] {
			cp += 1
			continue
		}
		return r1[cp] < r2[cp]
	}
	return false
}

func isNothingSubtype(t SubtypeData) bool {
	if allOrNothing, ok := t.(allOrNothingSubtype); ok {
		return allOrNothing.IsNothingSubtype()
	}
	return false
}

func memoSubtypeIsEmpty(cx Context, memoTable map[bddKey]*bddMemo, isEmptyPredicate bddIsEmptyPredicate, b Bdd) bool {
	key := b.canonicalKey()
	mm := memoTable[key]
	var m *bddMemo
	if mm != nil {
		res := mm.isEmpty
		switch res {
		case MemoStatus_CYCLIC:
			return true
		case MemoStatus_TRUE, MemoStatus_FALSE:
			return res == MemoStatus_TRUE
		case MemoStatus_NULL:
			m = mm
		case MemoStatus_LOOP, MemoStatus_PROVISIONAL:
			mm.isEmpty = MemoStatus_LOOP
			return true
		default:
			panic("Unexpected memo status")
		}
	} else {
		tmp := newBddMemo()
		m = &tmp
		memoTable[key] = m
	}
	m.isEmpty = MemoStatus_PROVISIONAL
	initStackDepth := cx.getMemoStackDepth()
	cx.pushToMemoStack(m)
	isEmpty := isEmptyPredicate(cx, b)
	isLoop := m.isEmpty == MemoStatus_LOOP
	if !isEmpty || initStackDepth == 0 {
		for i := initStackDepth + 1; i < cx.getMemoStackDepth(); i++ {
			m := cx.getMemoStack(i).isEmpty
			if m == MemoStatus_PROVISIONAL || m == MemoStatus_LOOP || m == MemoStatus_CYCLIC {
				if isEmpty {
					cx.getMemoStack(i).isEmpty = MemoStatus_TRUE
				} else {
					cx.getMemoStack(i).isEmpty = MemoStatus_NULL
				}
			}
		}
		for cx.getMemoStackDepth() > initStackDepth {
			cx.popFromMemoStack()
		}
		if isLoop && isEmpty {
			m.isEmpty = MemoStatus_CYCLIC
		} else {
			if isEmpty {
				m.isEmpty = MemoStatus_TRUE
			} else {
				m.isEmpty = MemoStatus_FALSE
			}
		}
	}
	return isEmpty
}

func isAllSubtype(t SubtypeData) bool {
	if allOrNothing, ok := t.(allOrNothingSubtype); ok {
		return allOrNothing.IsAllSubtype()
	}
	return false
}
