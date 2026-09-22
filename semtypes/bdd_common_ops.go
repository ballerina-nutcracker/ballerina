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

type bddOpMemoKey struct {
	B1 bdd
	B2 bdd
}

type bddOpMemo struct {
	UnionMemo        map[bddOpMemoKey]bdd
	IntersectionMemo map[bddOpMemoKey]bdd
	DiffMemo         map[bddOpMemoKey]bdd
}

type bddCommonOpsBase struct{}

type bddCommonOpsMethods struct{}

func bddAtom(atom atom) bddNode {
	return bddNodeCreate(atom, bddAll(), bddNothing(), bddNothing())
}

func bddUnion(b1 bdd, b2 bdd) bdd {
	return bddUnionWithMemo(createBddOpMemo(), b1, b2)
}

func bddUnionWithMemo(memoTable *bddOpMemo, b1 bdd, b2 bdd) bdd {
	key := bddOpMemoKey{B1: b1, B2: b2}
	memoized, ok := memoTable.UnionMemo[key]
	if ok {
		return memoized
	}
	memoized = bddUnionInner(memoTable, b1, b2)
	memoTable.UnionMemo[key] = memoized
	return memoized
}

func bddUnionInner(memo *bddOpMemo, b1 bdd, b2 bdd) bdd {
	if b1 == b2 {
		return b1
	}

	if allOrNothing1, ok := b1.(*bddAllOrNothing); ok {
		if allOrNothing1.IsAll() {
			return bddAll()
		}
		return b2
	}

	if allOrNothing2, ok := b2.(*bddAllOrNothing); ok {
		if allOrNothing2.IsAll() {
			return bddAll()
		}
		return b1
	}

	b1Bdd := b1.(bddNode)
	b2Bdd := b2.(bddNode)
	cmp := atomCmp(b1Bdd.atom(), b2Bdd.atom())
	if cmp < 0 {
		return bddCreate(b1Bdd.atom(), b1Bdd.left(), bddUnionWithMemo(memo, b1Bdd.middle(), b2), b1Bdd.right())
	} else if cmp > 0 {
		return bddCreate(b2Bdd.atom(), b2Bdd.left(), bddUnionWithMemo(memo, b1, b2Bdd.middle()), b2Bdd.right())
	} else {
		return bddCreate(b1Bdd.atom(), bddUnionWithMemo(memo, b1Bdd.left(), b2Bdd.left()), bddUnionWithMemo(memo, b1Bdd.middle(), b2Bdd.middle()), bddUnionWithMemo(memo, b1Bdd.right(), b2Bdd.right()))
	}
}

func bddIntersect(b1 bdd, b2 bdd) bdd {
	return bddIntersectWithMemo(createBddOpMemo(), b1, b2)
}

func bddIntersectWithMemo(memo *bddOpMemo, b1 bdd, b2 bdd) bdd {
	key := bddOpMemoKey{B1: b1, B2: b2}
	memoized, ok := memo.IntersectionMemo[key]
	if ok {
		return memoized
	}
	memoized = bddIntersectInner(memo, b1, b2)
	memo.IntersectionMemo[key] = memoized
	return memoized
}

func bddIntersectInner(memo *bddOpMemo, b1 bdd, b2 bdd) bdd {
	if b1 == b2 {
		return b1
	}

	if allOrNothing1, ok := b1.(*bddAllOrNothing); ok {
		if allOrNothing1.IsAll() {
			return b2
		}
		return bddNothing()
	}

	if allOrNothing2, ok := b2.(*bddAllOrNothing); ok {
		if allOrNothing2.IsAll() {
			return b1
		}
		return bddNothing()
	}

	b1Bdd := b1.(bddNode)
	b2Bdd := b2.(bddNode)
	cmp := atomCmp(b1Bdd.atom(), b2Bdd.atom())
	if cmp < 0 {
		return bddCreate(b1Bdd.atom(), bddIntersectWithMemo(memo, b1Bdd.left(), b2), bddIntersectWithMemo(memo, b1Bdd.middle(), b2), bddIntersectWithMemo(memo, b1Bdd.right(), b2))
	} else if cmp > 0 {
		return bddCreate(b2Bdd.atom(), bddIntersectWithMemo(memo, b1, b2Bdd.left()), bddIntersectWithMemo(memo, b1, b2Bdd.middle()), bddIntersectWithMemo(memo, b1, b2Bdd.right()))
	} else {
		return bddCreate(b1Bdd.atom(), bddIntersectWithMemo(memo, bddUnionWithMemo(memo, b1Bdd.left(), b1Bdd.middle()), bddUnionWithMemo(memo, b2Bdd.left(), b2Bdd.middle())), bddNothing(), bddIntersectWithMemo(memo, bddUnionWithMemo(memo, b1Bdd.right(), b1Bdd.middle()), bddUnionWithMemo(memo, b2Bdd.right(), b2Bdd.middle())))
	}
}

func bddDiff(b1 bdd, b2 bdd) bdd {
	return bddDiffWithMemo(createBddOpMemo(), b1, b2)
}

func bddDiffWithMemo(memo *bddOpMemo, b1 bdd, b2 bdd) bdd {
	key := bddOpMemoKey{B1: b1, B2: b2}
	memoized, ok := memo.DiffMemo[key]
	if ok {
		return memoized
	}
	memoized = bddDiffInner(memo, b1, b2)
	memo.DiffMemo[key] = memoized
	return memoized
}

func bddDiffInner(memo *bddOpMemo, b1 bdd, b2 bdd) bdd {
	if b1 == b2 {
		return bddNothing()
	}

	if allOrNothing2, ok := b2.(*bddAllOrNothing); ok {
		if allOrNothing2.IsAll() {
			return bddNothing()
		}
		return b1
	}

	if allOrNothing1, ok := b1.(*bddAllOrNothing); ok {
		if allOrNothing1.IsAll() {
			return bddComplement(b2)
		}
		return bddNothing()
	}

	b1Bdd := b1.(bddNode)
	b2Bdd := b2.(bddNode)
	cmp := atomCmp(b1Bdd.atom(), b2Bdd.atom())
	if cmp < 0 {
		return bddCreate(b1Bdd.atom(), bddDiffWithMemo(memo, bddUnionWithMemo(memo, b1Bdd.left(), b1Bdd.middle()), b2), bddNothing(), bddDiffWithMemo(memo, bddUnionWithMemo(memo, b1Bdd.right(), b1Bdd.middle()), b2))
	} else if cmp > 0 {
		return bddCreate(b2Bdd.atom(), bddDiffWithMemo(memo, b1, bddUnionWithMemo(memo, b2Bdd.left(), b2Bdd.middle())), bddNothing(), bddDiffWithMemo(memo, b1, bddUnionWithMemo(memo, b2Bdd.right(), b2Bdd.middle())))
	} else {
		return bddCreate(b1Bdd.atom(), bddDiffWithMemo(memo, bddUnionWithMemo(memo, b1Bdd.left(), b1Bdd.middle()), bddUnionWithMemo(memo, b2Bdd.left(), b2Bdd.middle())), bddNothing(), bddDiffWithMemo(memo, bddUnionWithMemo(memo, b1Bdd.right(), b1Bdd.middle()), bddUnionWithMemo(memo, b2Bdd.right(), b2Bdd.middle())))
	}
}

func bddComplement(b bdd) bdd {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		return allOrNothing.complement()
	}
	return bddNodeComplement(b.(bddNode))
}

func bddNodeComplement(b bddNode) bdd {
	bddNothing := bddNothing()
	if b.right() == bddNothing {
		return bddCreate(b.atom(), bddNothing, bddComplement(bddUnion(b.left(), b.middle())), bddComplement(b.middle()))
	} else if b.left() == bddNothing {
		return bddCreate(b.atom(), bddComplement(b.middle()), bddComplement(bddUnion(b.right(), b.middle())), bddNothing)
	} else if b.middle() == bddNothing {
		return bddCreate(b.atom(), bddComplement(b.left()), bddComplement(bddUnion(b.left(), b.right())), bddComplement(b.right()))
	} else {
		return bddCreate(b.atom(), bddComplement(bddUnion(b.left(), b.middle())), bddNothing, bddComplement(bddUnion(b.right(), b.middle())))
	}
}

func bddCreate(atom atom, left bdd, middle bdd, right bdd) bdd {
	if allOrNothing, ok := middle.(*bddAllOrNothing); ok && allOrNothing.IsAll() {
		return middle
	}
	if left == right {
		return bddUnion(left, right)
	}
	return bddNodeCreate(atom, left, middle, right)
}

func atomCmp(a1 atom, a2 atom) int {
	r1, ok1 := a1.(*recAtom)
	r2, ok2 := a2.(*recAtom)

	if ok1 {
		if ok2 {
			return r1.index() - r2.index()
		}
		return -1
	} else if ok2 {
		return 1
	}
	return a1.index() - a2.index()
}

func (c *bddCommonOpsMethods) BddToString(b bdd, inner bool) string {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return "1"
		}
		return "0"
	}

	var str string
	bdd := b.(bddNode)
	a := bdd.atom()
	if recAtom, ok := a.(*recAtom); ok {
		str = "r" + string(rune(recAtom.index()))
	} else {
		str = "a" + string(rune(a.index()))
	}
	str = str + "?" + c.BddToString(bdd.left(), true) + ":" + c.BddToString(bdd.middle(), true) + ":" + c.BddToString(bdd.right(), true)
	if inner {
		str = "(" + str + ")"
	}
	return str
}

func createBddOpMemo() *bddOpMemo {
	return &bddOpMemo{
		UnionMemo:        make(map[bddOpMemoKey]bdd),
		IntersectionMemo: make(map[bddOpMemoKey]bdd),
		DiffMemo:         make(map[bddOpMemoKey]bdd),
	}
}
