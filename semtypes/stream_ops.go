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

type streamOps struct{}

var _ basicTypeOps = &streamOps{}

func streamSubtypeComplement(t subtypeData) subtypeData {
	return bddSubtypeDiff(listSubtypeTwoElement, t)
}

func streamSubtypeIsEmpty(cx Context, t subtypeData) bool {
	b := t.(bdd)
	if bddPosMaybeEmpty(b) {
		b = bddIntersect(b, listSubtypeTwoElement)
	}
	return listSubtypeIsEmpty(cx, b)
}

func newStreamOps() streamOps {
	this := streamOps{}
	return this
}

func (s *streamOps) complement(t subtypeData) subtypeData {
	return streamSubtypeComplement(t)
}

func (s *streamOps) IsEmpty(cx Context, t subtypeData) bool {
	return streamSubtypeIsEmpty(cx, t)
}

func (s *streamOps) Union(d1 subtypeData, d2 subtypeData) subtypeData {
	return bddSubtypeUnion(d1, d2)
}

func (s *streamOps) Intersect(d1 subtypeData, d2 subtypeData) subtypeData {
	return bddSubtypeIntersect(d1, d2)
}

func (s *streamOps) Diff(d1 subtypeData, d2 subtypeData) subtypeData {
	return bddSubtypeDiff(d1, d2)
}
