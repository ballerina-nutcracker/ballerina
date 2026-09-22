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

type functionOps struct{}

var _ basicTypeOps = &functionOps{}

func (f *functionOps) IsEmpty(cx Context, t subtypeData) bool {
	return memoSubtypeIsEmpty(cx, cx.functionMemo(), func(cx Context, b bdd) bool {
		return bddEvery(cx, b, conjunctionNil, conjunctionNil, functionFormulaIsEmpty)
	}, t.(bdd))
}

func (f *functionOps) complement(t subtypeData) subtypeData {
	return bddComplement(t.(bdd))
}

func (f *functionOps) Diff(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddDiff(t1.(bdd), t2.(bdd))
}

func (f *functionOps) Intersect(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddIntersect(t1.(bdd), t2.(bdd))
}

func (f *functionOps) Union(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddUnion(t1.(bdd), t2.(bdd))
}

func functionFormulaIsEmpty(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
	return functionPathIsEmpty(cx, functionIntersectRet(cx, pos), functionUnionParams(cx, pos), functionUnionQualifiers(cx, pos), pos, neg)
}

func functionPathIsEmpty(cx Context, rets SemType, params SemType, qualifiers SemType, pos conjunctionHandle, neg conjunctionHandle) bool {
	if neg == conjunctionNil {
		return false
	} else {
		t := cx.FunctionAtomType(cx.conjunctionAtom(neg))
		negNext := cx.conjunctionNext(neg)
		t0 := t.ParamType
		t1 := t.RetType
		t2 := t.Qualifiers
		if t.IsGeneric {
			return (((IsSubtype(cx, qualifiers, t2) && IsSubtype(cx, params, t0)) && IsSubtype(cx, rets, t1)) || functionPathIsEmpty(cx, rets, params, qualifiers, pos, negNext))
		}
		return (((IsSubtype(cx, qualifiers, t2) && IsSubtype(cx, t0, params)) && functionPhi(cx, t0, complement(t1), pos)) || functionPathIsEmpty(cx, rets, params, qualifiers, pos, negNext))
	}
}

func functionPhi(cx Context, t0 SemType, t1 SemType, pos conjunctionHandle) bool {
	if pos == conjunctionNil {
		return ((!IsNever(t0)) && (IsEmpty(cx, t0) || IsEmpty(cx, t1)))
	}
	return functionPhiInner(cx, t0, t1, pos)
}

func functionPhiInner(cx Context, t0 SemType, t1 SemType, pos conjunctionHandle) bool {
	if pos == conjunctionNil {
		return (IsEmpty(cx, t0) || IsEmpty(cx, t1))
	} else {
		s := cx.FunctionAtomType(cx.conjunctionAtom(pos))
		posNext := cx.conjunctionNext(pos)
		s0 := s.ParamType
		s1 := s.RetType
		return (((IsSubtype(cx, t0, s0) || IsSubtype(cx, functionIntersectRet(cx, posNext), complement(t1))) && functionPhiInner(cx, t0, Intersect(t1, s1), posNext)) && functionPhiInner(cx, Diff(t0, s0), t1, posNext))
	}
}

func functionUnionParams(cx Context, pos conjunctionHandle) SemType {
	if pos == conjunctionNil {
		return Never
	}
	return Union(cx.FunctionAtomType(cx.conjunctionAtom(pos)).ParamType, functionUnionParams(cx, cx.conjunctionNext(pos)))
}

func functionUnionQualifiers(cx Context, pos conjunctionHandle) SemType {
	if pos == conjunctionNil {
		return Never
	}
	return Union(cx.FunctionAtomType(cx.conjunctionAtom(pos)).Qualifiers, functionUnionQualifiers(cx, cx.conjunctionNext(pos)))
}

func functionIntersectRet(cx Context, pos conjunctionHandle) SemType {
	if pos == conjunctionNil {
		return Val
	}
	return Intersect(cx.FunctionAtomType(cx.conjunctionAtom(pos)).RetType, functionIntersectRet(cx, cx.conjunctionNext(pos)))
}

func NewFunctionOps() functionOps {
	this := functionOps{}
	return this
}

func (f *functionOps) functionTheta(cx Context, t0 SemType, t1 SemType, pos conjunctionHandle) bool {
	if pos == conjunctionNil {
		return (IsEmpty(cx, t0) || IsEmpty(cx, t1))
	} else {
		s := cx.FunctionAtomType(cx.conjunctionAtom(pos))
		posNext := cx.conjunctionNext(pos)
		s0 := s.ParamType
		s1 := s.RetType
		return ((IsSubtype(cx, t0, s0) || f.functionTheta(cx, Diff(s0, t0), s1, posNext)) && (IsSubtype(cx, t1, complement(s1)) || f.functionTheta(cx, s0, Intersect(s1, t1), posNext)))
	}
}

// Corresponds to dom^? in AMK tutorial.
func FunctionParamListType(cx Context, fnTy SemType) SemType {
	if !IsSubtypeSimple(fnTy, Function) {
		return SemType{}
	}
	if fnTy.some() == 0 {
		return Never
	}
	bdd := getComplexSubtypeData(fnTy, btFunction).(bdd)
	return functionParamListTypeInner(cx, Never, bdd)
}

func functionParamListTypeInner(cx Context, accumTy SemType, bdd bdd) SemType {
	if allOrNothing, ok := bdd.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return accumTy
		}
		return Any
	}
	bn := bdd.(bddNode)
	atomArgListTy := cx.FunctionAtomType(bn.atom()).ParamType
	return Intersect(functionParamListTypeInner(cx, Union(accumTy, atomArgListTy), bn.left()),
		Intersect(functionParamListTypeInner(cx, accumTy, bn.middle()),
			functionParamListTypeInner(cx, accumTy, bn.right())))
}

// Corresponds to apply^? in AMK tutorial.
func FunctionReturnType(cx Context, fnTy SemType, argList SemType) SemType {
	domain := FunctionParamListType(cx, fnTy)
	if IsZero(domain) || !IsSubtype(cx, argList, domain) {
		return SemType{}
	}
	if fnTy.some() == 0 {
		return Val
	}
	bdd := getComplexSubtypeData(fnTy, btFunction).(bdd)
	return functionReturnTypeInner(cx, argList, Val, bdd)
}

func functionReturnTypeInner(cx Context, accumArgList SemType, accumReturn SemType, bdd bdd) SemType {
	if IsEmpty(cx, accumArgList) || IsEmpty(cx, accumReturn) {
		return Never
	}
	switch b := bdd.(type) {
	case *bddAllOrNothing:
		if b.IsAll() {
			return accumReturn
		}
		return Never
	case bddNode:
		fnAtom := cx.FunctionAtomType(b.atom())
		atomArgListTy := fnAtom.ParamType
		atomReturnTy := fnAtom.RetType
		return Union(functionReturnTypeInner(cx, accumArgList, Intersect(accumReturn, atomReturnTy), b.left()),
			Union(functionReturnTypeInner(cx, Diff(accumArgList, atomArgListTy), accumReturn, b.left()),
				Union(functionReturnTypeInner(cx, accumArgList, accumReturn, b.middle()),
					functionReturnTypeInner(cx, accumArgList, accumReturn, b.right()))))

	default:
		panic("impossible")
	}
}

// CreateIsolatedFn returns the top type of isolated functions:
// the type of every `isolated function` value.
func CreateIsolatedFn(cx Context) SemType {
	if IsZero(cx._isolatedFnMemo) {
		fd := NewFunctionDefinition()
		env := cx.Env()
		cx._isolatedFnMemo = fd.Define(env, Never, Val, FunctionQualifiersFrom(env, true, false))
	}

	return cx._isolatedFnMemo
}
