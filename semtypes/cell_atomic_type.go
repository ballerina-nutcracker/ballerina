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

type CellMutability uint

type cellAtomicType struct {
	Ty  SemType
	Mut CellMutability
}

const (
	CellMutability_CELL_MUT_NONE CellMutability = iota
	CellMutability_CELL_MUT_LIMITED
	CellMutability_CELL_MUT_UNLIMITED
)

var _ atomicType = &cellAtomicType{}

func newCellAtomicTypeFromTyMut(ty SemType, mut CellMutability) cellAtomicType {
	this := cellAtomicType{}
	this.Ty = ty
	this.Mut = mut
	return this
}

func cellAtomicTypeFrom(ty SemType, mut CellMutability) cellAtomicType {
	return newCellAtomicTypeFromTyMut(ty, mut)
}

func (c *cellAtomicType) atomKind() kind {
	return kind_CELL_ATOM
}
