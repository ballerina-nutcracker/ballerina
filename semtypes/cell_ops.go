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

type cellOps struct {
	CommonOps
}

var _ BasicTypeOps = &cellOps{}

func cellFormulaIsEmpty(cx Context, t SubtypeData) bool {
	return bddEvery(cx, t.(Bdd), conjunctionNil, conjunctionNil, cellFormulaIsEmptyInner)
}

func cellFormulaIsEmptyInner(cx Context, posList conjunctionHandle, negList conjunctionHandle) bool {
	var combined cellAtomicType
	if posList == conjunctionNil {
		combined = cellAtomicTypeFrom(VAL, CellMutability_CELL_MUT_UNLIMITED)
	} else {
		combined = *cellAtomType(cx.conjunctionAtom(posList))
		p := cx.conjunctionNext(posList)
		for p != conjunctionNil {
			combined = intersectCellAtomicType(&combined, cellAtomType(cx.conjunctionAtom(p)))
			p = cx.conjunctionNext(p)
		}
	}
	return !cellInhabited(cx, combined, negList)
}

func cellInhabited(cx Context, posCell cellAtomicType, negList conjunctionHandle) bool {
	pos := posCell.Ty
	if IsEmpty(cx, pos) {
		return false
	}
	switch posCell.Mut {
	case CellMutability_CELL_MUT_NONE:
		return cellMutNoneInhabited(cx, pos, negList)
	case CellMutability_CELL_MUT_LIMITED:
		return cellMutLimitedInhabited(cx, pos, negList)
	default:
		return cellMutUnlimitedInhabited(cx, pos, negList)
	}
}

func cellMutNoneInhabited(cx Context, pos SemType, negList conjunctionHandle) bool {
	negListUnionResult := cellNegListUnion(cx, negList)
	return IsNever(negListUnionResult) || !IsEmpty(cx, Diff(pos, negListUnionResult))
}

func cellNegListUnion(cx Context, negList conjunctionHandle) SemType {
	var negUnion SemType
	negUnion = NEVER
	neg := negList
	for neg != conjunctionNil {
		negUnion = Union(negUnion, cellAtomType(cx.conjunctionAtom(neg)).Ty)
		neg = cx.conjunctionNext(neg)
	}
	return negUnion
}

func cellMutLimitedInhabited(cx Context, pos SemType, negList conjunctionHandle) bool {
	if negList == conjunctionNil {
		return true
	}
	negAtomicCell := cellAtomType(cx.conjunctionAtom(negList))
	if negAtomicCell.Mut >= CellMutability_CELL_MUT_LIMITED && IsEmpty(cx, Diff(pos, negAtomicCell.Ty)) {
		return false
	}
	return cellMutLimitedInhabited(cx, pos, cx.conjunctionNext(negList))
}

func cellMutUnlimitedInhabited(cx Context, pos SemType, negList conjunctionHandle) bool {
	neg := negList
	for neg != conjunctionNil {
		cellAtom := cellAtomType(cx.conjunctionAtom(neg))
		if cellAtom.Mut == CellMutability_CELL_MUT_LIMITED && IsSameType(cx, VAL, cellAtom.Ty) {
			return false
		}
		neg = cx.conjunctionNext(neg)
	}
	negListUnionResult := cellNegListUnlimitedUnion(cx, negList)
	return IsNever(negListUnionResult) || !IsEmpty(cx, Diff(pos, negListUnionResult))
}

func cellNegListUnlimitedUnion(cx Context, negList conjunctionHandle) SemType {
	var negUnion SemType
	negUnion = NEVER
	neg := negList
	for neg != conjunctionNil {
		cellAtom := cellAtomType(cx.conjunctionAtom(neg))
		if cellAtom.Mut == CellMutability_CELL_MUT_UNLIMITED {
			negUnion = Union(negUnion, cellAtom.Ty)
		}
		neg = cx.conjunctionNext(neg)
	}
	return negUnion
}

func intersectCellAtomicType(c1 *cellAtomicType, c2 *cellAtomicType) cellAtomicType {
	ty := Intersect(c1.Ty, c2.Ty)
	mut := cellMutabilityMin(c1.Mut, c2.Mut)
	return cellAtomicTypeFrom(ty, mut)
}

func cellSubtypeUnion(t1 SubtypeData, t2 SubtypeData) ProperSubtypeData {
	return cellSubtypeDataEnsureProper(bddSubtypeUnion(t1, t2))
}

func cellSubtypeIntersect(t1 SubtypeData, t2 SubtypeData) ProperSubtypeData {
	return cellSubtypeDataEnsureProper(bddSubtypeIntersect(t1, t2))
}

func cellSubtypeDiff(t1 SubtypeData, t2 SubtypeData) ProperSubtypeData {
	return cellSubtypeDataEnsureProper(bddSubtypeDiff(t1, t2))
}

func cellSubtypeComplement(t SubtypeData) ProperSubtypeData {
	return cellSubtypeDataEnsureProper(bddSubtypeComplement(t))
}

func cellSubtypeDataEnsureProper(subtypeData SubtypeData) ProperSubtypeData {
	if allOrNothingSubtype, ok := subtypeData.(allOrNothingSubtype); ok {
		var a atom
		if allOrNothingSubtype.IsAllSubtype() {
			a = ATOM_CELL_VAL
		} else {
			a = ATOM_CELL_NEVER
		}
		return bddAtom(a)
	} else {
		return subtypeData.(ProperSubtypeData)
	}
}

func newCellOps() cellOps {
	this := cellOps{}
	return this
}

func (c *cellOps) Union(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	return cellSubtypeUnion(t1, t2)
}

func (c *cellOps) Intersect(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	return cellSubtypeIntersect(t1, t2)
}

func (c *cellOps) Diff(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	return cellSubtypeDiff(t1, t2)
}

func (c *cellOps) complement(t SubtypeData) SubtypeData {
	return cellSubtypeComplement(t)
}

func (c *cellOps) IsEmpty(cx Context, t SubtypeData) bool {
	return cellFormulaIsEmpty(cx, t)
}

func cellMutabilityMin(m1 CellMutability, m2 CellMutability) CellMutability {
	if m1 <= m2 {
		return m1
	}
	return m2
}
