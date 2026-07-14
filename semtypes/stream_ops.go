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

var _ BasicTypeOps = &streamOps{}

func streamSubtypeComplement(t SubtypeData) SubtypeData {
	return bddSubtypeDiff(LIST_SUBTYPE_TWO_ELEMENT, t)
}

func streamSubtypeIsEmpty(cx Context, t SubtypeData) bool {
	b := t.(Bdd)
	if bddPosMaybeEmpty(b) {
		b = bddIntersect(b, LIST_SUBTYPE_TWO_ELEMENT)
	}
	return listSubtypeIsEmpty(cx, b)
}

func newStreamOps() streamOps {
	this := streamOps{}
	return this
}

func (s *streamOps) complement(t SubtypeData) SubtypeData {
	return streamSubtypeComplement(t)
}

func (s *streamOps) IsEmpty(cx Context, t SubtypeData) bool {
	return streamSubtypeIsEmpty(cx, t)
}

func (s *streamOps) Union(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeUnion(d1, d2)
}

func (s *streamOps) Intersect(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeIntersect(d1, d2)
}

func (s *streamOps) Diff(d1 SubtypeData, d2 SubtypeData) SubtypeData {
	return bddSubtypeDiff(d1, d2)
}
