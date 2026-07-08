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

type ListAtomicType struct {
	Members fixedLengthArray
	rest    SemType
}

var _ atomicType = &ListAtomicType{}

func newListAtomicTypeFromMembersRest(members fixedLengthArray, rest SemType) ListAtomicType {
	this := ListAtomicType{}
	this.Members = members
	this.rest = rest
	return this
}

func listAtomicTypeFrom(members fixedLengthArray, rest SemType) ListAtomicType {
	return newListAtomicTypeFromMembersRest(members, rest)
}

func (l *ListAtomicType) atomKind() kind {
	return kind_LIST_ATOM
}

func (atomic *ListAtomicType) MemberAtInnerVal(index int) SemType {
	return cellInnerVal(atomic.MemberAt(index))
}

func (atomic *ListAtomicType) MemberAt(index int) SemType {
	return listMemberAt(atomic.Members, atomic.rest, index)
}

func (atomic *ListAtomicType) Rest() SemType {
	return cellInnerVal(atomic.rest)
}
