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

type objectOps struct {
}

var _ basicTypeOps = &objectOps{}

func (o *objectOps) Diff(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddSubtypeDiff(t1, t2)
}

func (o *objectOps) Intersect(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddSubtypeIntersect(t1, t2)
}

func (o *objectOps) Union(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddSubtypeUnion(t1, t2)
}

func objectSubTypeIsEmpty(cx Context, t subtypeData) bool {
	return memoSubtypeIsEmpty(cx, cx.mappingMemo(), objectBddIsEmpty, t.(bdd))
}

func objectBddIsEmpty(cx Context, b bdd) bool {
	return bddEveryPositive(cx, b, conjunctionNil, conjunctionNil, mappingFormulaIsEmpty)
}

func newObjectOps() objectOps {
	this := objectOps{}
	return this
}

func (o *objectOps) complement(t subtypeData) subtypeData {
	return o.objectSubTypeComplement(t)
}

func (o *objectOps) IsEmpty(cx Context, t subtypeData) bool {
	return objectSubTypeIsEmpty(cx, t)
}

func (o *objectOps) objectSubTypeComplement(t subtypeData) subtypeData {
	return bddSubtypeDiff(mappingSubtypeObject, t)
}
