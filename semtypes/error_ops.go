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

type errorOps struct{}

var _ basicTypeOps = &errorOps{}

func errorSubtypeComplement(t subtypeData) subtypeData {
	return bddSubtypeDiff(bddSubtypeRo, t)
}

func errorSubtypeIsEmpty(cx Context, t subtypeData) bool {
	b := t.(bdd)
	if bddPosMaybeEmpty(b) {
		b = bddIntersect(b, bddSubtypeRo)
	}
	return memoSubtypeIsEmpty(cx, cx.mappingMemo(), errorBddIsEmpty, b)
}

func errorBddIsEmpty(cx Context, b bdd) bool {
	return bddEveryPositive(cx, b, conjunctionNil, conjunctionNil, mappingFormulaIsEmpty)
}

func (e *errorOps) complement(d subtypeData) subtypeData {
	return errorSubtypeComplement(d)
}

func (e *errorOps) IsEmpty(cx Context, t subtypeData) bool {
	return errorSubtypeIsEmpty(cx, t)
}

func (e *errorOps) Union(d1 subtypeData, d2 subtypeData) subtypeData {
	return bddSubtypeUnion(d1, d2)
}

func (e *errorOps) Intersect(d1 subtypeData, d2 subtypeData) subtypeData {
	return bddSubtypeIntersect(d1, d2)
}

func (e *errorOps) Diff(d1 subtypeData, d2 subtypeData) subtypeData {
	return bddSubtypeDiff(d1, d2)
}
