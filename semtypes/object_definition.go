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

import "github.com/ballerina-nutcracker/ballerina/common"

// Represent object type desc.
type ObjectDefinition struct {
	mappingDefinition MappingDefinition
}

var _ Definition = &ObjectDefinition{}

func NewObjectDefinition() ObjectDefinition {
	this := ObjectDefinition{}
	this.mappingDefinition = NewMappingDefinition()
	return this
}

func ObjectDefinitionDistinct(distinctId int) SemType {
	common.Assert(func() bool { return distinctId >= 0 })
	bdd := bddAtom(new(createDistinctRecAtom(-distinctId - 1)))
	return getBasicSubtype(btObject, bdd)
}

func stripObjectDistinctAtoms(ty SemType) SemType {
	return stripDistinctAtomsFromSemType(ty, btObject, stripDistinctAtomsFromBdd)
}

func stripDistinctAtomsFromSemType(ty SemType, typeCode basicTypeCode, stripBdd func(bdd) bdd) SemType {
	typeBit := basicTypeBitSet(1 << typeCode.Code())
	if ty.some()&typeBit == 0 {
		return ty
	}
	all := ty.all()
	var subtypes []basicSubtype
	dataIndex := 0
	for code := 0; code <= typeCodeUndef; code++ {
		bit := basicTypeBitSet(1 << code)
		if ty.some()&bit == 0 {
			continue
		}
		data := ty.subtypeDataList()[dataIndex]
		dataIndex++
		if code == typeCode.Code() {
			stripped := stripBdd(data.(bdd))
			if allOrNothing, ok := stripped.(*bddAllOrNothing); ok {
				if allOrNothing.IsAll() {
					all |= bit
				}
				continue
			}
			data = stripped.(properSubtypeData)
		}
		subtypes = append(subtypes, basicSubtypeFrom(basicTypeCodeFrom(code), data))
	}
	return createComplexSemType(all, subtypes...)
}

// Each object type is represented as mapping type (with its basic type set to object) as fallows
//
//	{
//	  "$qualifiers": {
//	    boolean isolated,
//	    "client"|"service" network
//	  },
//	   [field_name]: {
//	     "field"|"method"|"remote-method"|"resource-method" kind,
//	     "public"|"private" visibility,
//	      Val value;
//	   }
//	   ...{
//	     "field" kind,
//	     "public"|"private" visibility,
//	      Val value;
//	   } | {
//	      "method"|"remote-method"|"resource-method" kind,
//	      "public"|"private" visibility,
//	      Function value;
//	   }
//	}
func (o *ObjectDefinition) Define(env Env, qualifiers ObjectQualifiers, members []Member) SemType {
	common.Assert(func() bool { return objectDefinitionValidateMembers(members) })
	var mut CellMutability
	if qualifiers.readonly {
		mut = CellMutabilityNone
	} else {
		mut = CellMutabilityLimited
	}
	var memberStream []cellField
	for _, member := range members {
		memberStream = append(memberStream, memberField(env, &member, mut))
	}
	qualifierStream := []cellField{qualifiers.Field(env)}
	var cellFields []cellField
	cellFields = append(cellFields, memberStream...)
	cellFields = append(cellFields, qualifierStream...)
	mappingType := o.mappingDefinition.Define(env, cellFields, o.restMemberType(env, mut, qualifiers.readonly))
	return o.objectContaining(mappingType)
}

func objectDefinitionValidateMembers(members []Member) bool {
	// Check if there are two members with same name
	nameMap := make(map[string]bool)
	for _, member := range members {
		if nameMap[member.Name] {
			return false
		}
		nameMap[member.Name] = true
	}
	return len(nameMap) == len(members)
}

func (o *ObjectDefinition) objectContaining(mappingType SemType) SemType {
	bdd := subtypeDataAt(mappingType, btMapping)
	return createBasicSemType(btObject, bdd)
}

func (o *ObjectDefinition) restMemberType(env Env, mut CellMutability, immutable bool) SemType {
	fieldDefn := NewMappingDefinition()
	var fieldValueType SemType
	if immutable {
		fieldValueType = ValReadonly
	} else {
		fieldValueType = Val
	}
	fieldMemberType := fieldDefn.DefineMappingTypeWrapped(
		env,
		[]Field{
			FieldFrom("value", fieldValueType, immutable, false),
			new(MemberKindField).field(),
			visibilityAll,
		},
		Never)

	methodDefn := NewMappingDefinition()
	methodMemberType := methodDefn.DefineMappingTypeWrapped(
		env,
		[]Field{
			FieldFrom("value", Function, true, false),
			allMethodField(),
			visibilityAll,
		},
		Never)
	return cellContainingWithEnvSemTypeCellMutability(env, Union(fieldMemberType, methodMemberType), mut)
}

func memberField(env Env, member *Member, mut CellMutability) cellField {
	md := NewMappingDefinition()
	var fieldMut CellMutability
	if member.Immutable {
		fieldMut = CellMutabilityNone
	} else {
		fieldMut = mut
	}
	semtype := md.DefineMappingTypeWrapped(
		env,
		[]Field{
			FieldFrom("value", member.ValueType, member.Immutable, false),
			(&member.Kind).field(),
			(&member.Visibility).field(),
		},
		Never)
	return cellFieldFrom(member.Name, cellContainingWithEnvSemTypeCellMutability(env, semtype, fieldMut))
}

func (o *ObjectDefinition) GetSemType(env Env) SemType {
	return o.objectContaining(o.mappingDefinition.GetSemType(env))
}
