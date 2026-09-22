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
	if sameSemType(Val, constraint) {
		return Typedesc
	}

	mappingDef := NewMappingDefinition()
	mappingType := mappingDef.Define(env, nil, constraint, MappingMutability(CellMutabilityNone))
	bdd := subtypeDataAt(mappingType, btMapping).(bdd)
	return createBasicSemType(btTypeDesc, bdd)
}

// TypedescConstraint extracts the constraint T from a typedesc<T>.
// Returns Val when td is the unconstrained typedesc, nil if td is not a typedesc built via TypedescContaining.
func TypedescConstraint(ctx Context, td SemType) SemType {
	if !IsSubtypeSimple(td, Typedesc) {
		return SemType{}
	}
	if td.some() == 0 {
		return Val
	}
	mappingTy := convertTypeDescToMapping(ctx, td)
	return MappingMemberTypeInnerVal(ctx, mappingTy, String)
}

func convertTypeDescToMapping(ctx Context, ty SemType) SemType {
	td := Intersect(ty, Typedesc)
	if IsEmpty(ctx, td) {
		return SemType{}
	}
	bdd := subtypeDataAt(td, btTypeDesc)
	return createBasicSemType(btMapping, bdd)
}
