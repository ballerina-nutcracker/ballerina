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

func cellContaining(env Env, ty SemType) SemType {
	return cellContainingWithEnvSemTypeCellMutability(env, ty, CellMutability_CELL_MUT_LIMITED)
}

func roCellContaining(env Env, ty SemType) SemType {
	return cellContainingWithEnvSemTypeCellMutability(env, ty, CellMutability_CELL_MUT_NONE)
}

func cellContainingWithEnvSemTypeCellMutability(env Env, ty SemType, mut CellMutability) SemType {
	if ty.some() == 0 {
		if cellTy, ok := env.preallocatedTypeVals.basicTypeCell(ty.all(), mut); ok {
			return cellTy
		}
	}
	atomicCell := cellAtomicTypeFrom(ty, mut)
	atom := env.cellAtom(&atomicCell)
	bdd := bddAtom(atom)
	return getBasicSubtype(BTCell, bdd)
}
