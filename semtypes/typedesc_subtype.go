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

func TypedescContaining(env Env, constraint SemType) SemType {
	if sameSemType(VAL, constraint) {
		return TYPEDESC
	}

	mappingDef := NewMappingDefinition()
	mappingType := mappingDef.DefineMappingTypeWrappedWithEnvFieldsSemTypeCellMutability(env, nil, constraint, CellMutability_CELL_MUT_NONE)
	bdd := subtypeData(mappingType, BTMapping).(Bdd)
	return createBasicSemType(BTTypeDesc, bdd)
}

// TypedescConstraint extracts the constraint T from a typedesc<T>.
// Returns VAL when td is the unconstrained typedesc, nil if td is not a typedesc built via TypedescContaining.
func TypedescConstraint(ctx Context, td SemType) SemType {
	if !IsSubtypeSimple(td, TYPEDESC) {
		return SemType{}
	}
	if td.some() == 0 {
		return VAL
	}
	mappingTy := convertTypeDescToMapping(ctx, td)
	return MappingMemberTypeInnerVal(ctx, mappingTy, STRING)
}

func convertTypeDescToMapping(ctx Context, ty SemType) SemType {
	td := Intersect(ty, TYPEDESC)
	if IsEmpty(ctx, td) {
		return SemType{}
	}
	bdd := subtypeData(td, BTTypeDesc)
	return createBasicSemType(BTMapping, bdd)
}
