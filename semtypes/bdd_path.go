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

type bddPath struct {
	bdd Bdd
	pos []atom
	neg []atom
}

func newBddPathFromBddPath(src bddPath) bddPath {
	this := bddPath{}
	this.bdd = src.bdd
	this.pos = slices.Clone(src.pos)
	this.neg = slices.Clone(src.neg)
	return this
}

func newBddPath() bddPath {
	this := bddPath{}
	this.bdd = bddAll()
	this.pos = nil
	this.neg = nil
	return this
}

func bddPaths(b Bdd, paths *[]bddPath, accum bddPath) {
	allOrNothing, ok := b.(*bddAllOrNothing)
	if ok {
		if allOrNothing.IsAll() {
			*paths = append(*paths, accum)
		}
	} else {
		left := bddPathClone(accum)
		right := bddPathClone(accum)
		bn, ok := b.(bddNode)
		if !ok {
			panic("b is not a bddNode")
		}
		left.pos = append(left.pos, bn.atom())
		left.bdd = bddIntersect(left.bdd, bddAtom(bn.atom()))
		bddPaths(bn.left(), paths, left)
		bddPaths(bn.middle(), paths, accum)
		right.neg = append(right.neg, bn.atom())
		right.bdd = bddDiff(right.bdd, bddAtom(bn.atom()))
		bddPaths(bn.right(), paths, right)
	}
}

func bddPathsPositive(b Bdd, paths *[]bddPath, accum bddPath) {
	allOrNothing, ok := b.(*bddAllOrNothing)
	if ok {
		if allOrNothing.IsAll() {
			*paths = append(*paths, accum)
		}
	} else {
		left := bddPathClone(accum)
		right := bddPathClone(accum)
		bn, ok := b.(bddNode)
		if !ok {
			panic("b is not a bddNode")
		}
		if isPositiveAtom(bn.atom()) {
			left.pos = append(left.pos, bn.atom())
			left.bdd = bddIntersect(left.bdd, bddAtom(bn.atom()))
			right.neg = append(right.neg, bn.atom())
			right.bdd = bddDiff(right.bdd, bddAtom(bn.atom()))
		}
		bddPathsPositive(bn.left(), paths, left)
		bddPathsPositive(bn.middle(), paths, accum)
		bddPathsPositive(bn.right(), paths, right)
	}
}

func isPositiveAtom(atom atom) bool {
	if recAtom, ok := atom.(*recAtom); ok && recAtom.index() < 0 {
		return false
	}
	return true
}

func bddPathClone(path bddPath) bddPath {

	return newBddPathFromBddPath(path)
}

func bddPathFrom() bddPath {
	return newBddPath()
}
