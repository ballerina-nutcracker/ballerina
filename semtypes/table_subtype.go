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

type tableSubtype struct{}

func newTableSubtype() tableSubtype {
	this := tableSubtype{}
	return this
}

func tableContainingKeyConstraint(cx Context, tableConstraint SemType, keyConstraint SemType) SemType {
	var normalizedKc SemType
	lat := ToListAtomicType(cx, keyConstraint)
	if (lat != nil) && (CELL_ATOMIC_UNDEF == getCellAtomicType(lat.rest)) {
		members := lat.Members
		switch members.FixedLength {
		case 0:
			normalizedKc = VAL
		case 1:
			normalizedKc = getCellAtomicType(members.initial[0]).Ty
		default:
			normalizedKc = keyConstraint
		}
	} else {
		normalizedKc = keyConstraint
	}
	return tableContainingWithEnvSemTypeSemTypeSemType(cx.Env(), tableConstraint, normalizedKc, VAL)
}

func tableContainingKeySpecifier(cx Context, tableConstraint SemType, fieldNames []string) SemType {
	fieldNameSingletons := make([]SemType, len(fieldNames))
	fieldTypes := make([]SemType, len(fieldNames))
	for i := range fieldNames {
		key := StringConst(fieldNames[i])
		fieldNameSingletons[i] = key
		fieldTypes[i] = MappingMemberTypeInnerVal(cx, tableConstraint, key)
	}
	listDef1 := NewListDefinition()
	normalizedKs := listDef1.TupleTypeWrapped(cx.Env(), fieldNameSingletons...)
	var normalizedKc SemType
	if len(fieldTypes) > 1 {
		ld := NewListDefinition()
		normalizedKc = ld.TupleTypeWrapped(cx.Env(), fieldTypes...)
	} else {
		normalizedKc = fieldTypes[0]
	}
	return tableContainingWithEnvSemTypeSemTypeSemType(cx.Env(), tableConstraint, normalizedKc, normalizedKs)
}

func tableContainingDefault(env Env, tableConstraint SemType) SemType {
	return tableContainingWithEnvSemTypeCellMutability(env, tableConstraint, CellMutability_CELL_MUT_LIMITED)
}

func tableContainingWithEnvSemTypeCellMutability(env Env, tableConstraint SemType, mut CellMutability) SemType {
	var normalizedKc = VAL
	var normalizedKs = VAL
	return tableContaining(env, tableConstraint, normalizedKc, normalizedKs, mut)
}

func tableContaining(env Env, tableConstraint SemType, normalizedKc SemType, normalizedKs SemType, mut CellMutability) SemType {
	if !IsSubtypeSimple(tableConstraint, MAPPING) {
		panic("assertion failed")
	}
	typeParamArrDef := NewListDefinition()
	typeParamArray := typeParamArrDef.DefineListTypeWrappedWithEnvSemTypeCellMutability(env, tableConstraint, mut)
	listDef := NewListDefinition()
	tupleType := listDef.TupleTypeWrapped(env, typeParamArray, normalizedKc, normalizedKs)
	bdd := subtypeData(tupleType, BTList).(Bdd)
	return createBasicSemType(BTTable, bdd)
}

func tableContainingWithEnvSemTypeSemTypeSemType(env Env, tableConstraint SemType, normalizedKc SemType, normalizedKs SemType) SemType {
	return tableContaining(env, tableConstraint, normalizedKc, normalizedKs, CellMutability_CELL_MUT_LIMITED)
}
