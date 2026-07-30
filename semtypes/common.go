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
	bddIsEmptyPredicate func(cx Context, b bdd) bool
)

func bddEvery(cx Context, b bdd, pos conjunctionHandle, neg conjunctionHandle, predicate bddPredicate) bool {
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

func bddEveryPositive(cx Context, b bdd, pos conjunctionHandle, neg conjunctionHandle, predicate bddPredicate) bool {
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

func bddPosMaybeEmpty(b bdd) bool {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		return allOrNothing.IsAll()
	} else {
		bn := b.(bddNode)
		return bddPosMaybeEmpty(bn.middle()) || bddPosMaybeEmpty(bn.right())
	}
}

func bddSubtypeUnion(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddUnion(t1.(bdd), t2.(bdd))
}

func bddSubtypeIntersect(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddIntersect(t1.(bdd), t2.(bdd))
}

func bddSubtypeDiff(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddDiff(t1.(bdd), t2.(bdd))
}

func bddSubtypeComplement(t subtypeData) subtypeData {
	return bddComplement(t.(bdd))
}

func notIsEmpty(cx Context, t subtypeData) bool {
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

func isNothingSubtype(t subtypeData) bool {
	if allOrNothing, ok := t.(allOrNothingSubtype); ok {
		return allOrNothing.IsNothingSubtype()
	}
	return false
}

func memoSubtypeIsEmpty(cx Context, memoTable map[bddKey]*bddMemo, isEmptyPredicate bddIsEmptyPredicate, b bdd) bool {
	key := b.canonicalKey()
	mm := memoTable[key]
	var m *bddMemo
	if mm != nil {
		res := mm.isEmpty
		switch res {
		case memostatusCyclic:
			return true
		case memostatusTrue, memostatusFalse:
			return res == memostatusTrue
		case memostatusNull:
			m = mm
		case memostatusLoop, memostatusProvisional:
			mm.isEmpty = memostatusLoop
			return true
		default:
			panic("Unexpected memo status")
		}
	} else {
		tmp := newBddMemo()
		m = &tmp
		memoTable[key] = m
	}
	m.isEmpty = memostatusProvisional
	initStackDepth := cx.getMemoStackDepth()
	cx.pushToMemoStack(m)
	isEmpty := isEmptyPredicate(cx, b)
	isLoop := m.isEmpty == memostatusLoop
	if !isEmpty || initStackDepth == 0 {
		for i := initStackDepth + 1; i < cx.getMemoStackDepth(); i++ {
			m := cx.getMemoStack(i).isEmpty
			if m == memostatusProvisional || m == memostatusLoop || m == memostatusCyclic {
				if isEmpty {
					cx.getMemoStack(i).isEmpty = memostatusTrue
				} else {
					cx.getMemoStack(i).isEmpty = memostatusNull
				}
			}
		}
		for cx.getMemoStackDepth() > initStackDepth {
			cx.popFromMemoStack()
		}
		if isLoop && isEmpty {
			m.isEmpty = memostatusCyclic
		} else {
			if isEmpty {
				m.isEmpty = memostatusTrue
			} else {
				m.isEmpty = memostatusFalse
			}
		}
	}
	return isEmpty
}

func isAllSubtype(t subtypeData) bool {
	if allOrNothing, ok := t.(allOrNothingSubtype); ok {
		return allOrNothing.IsAllSubtype()
	}
	return false
}
