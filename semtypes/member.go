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

type Member struct {
	Name       string
	ValueType  SemType
	Kind       MemberKind
	Visibility string
	Immutable  bool
}

func newMember(name string, valueTy SemType, kind MemberKind, visibility string, immutable bool) *Member {
	return &Member{Name: name, ValueType: valueTy, Kind: kind, Visibility: visibility, Immutable: immutable}
}

type memberTag interface {
	field() Field
}

type MemberKind uint8

const (
	MemberKindField MemberKind = iota
	MemberKindMethod
	MemberKindRemoteMethod
	MemberKindResourceMethod
)

func (k *MemberKind) field() Field {
	switch *k {
	case MemberKindField:
		return Field{name: "kind", typeOf: StringConst("field"), readonly: true, optional: false}
	case MemberKindMethod:
		return Field{name: "kind", typeOf: StringConst("method"), readonly: true, optional: false}
	case MemberKindRemoteMethod:
		return Field{name: "kind", typeOf: StringConst("remote-method"), readonly: true, optional: false}
	case MemberKindResourceMethod:
		return Field{name: "kind", typeOf: StringConst("resource-method"), readonly: true, optional: false}
	default:
		panic("invalid member kind")
	}
}

// toplevel field which matches methods, remote-methods and resource methods.
func allMethodField() Field {
	tys := []string{
		"method",
		"remote-method",
		"resource-method",
	}
	var ty = Never
	for _, each := range tys {
		ty = Union(ty, StringConst(each))
	}
	return Field{name: "kind", typeOf: ty, readonly: true, optional: false}
}

// VisibilityPublic is the visibility region of public members.
const VisibilityPublic = "public"

var visibilityAll = Field{name: "visibility", typeOf: String, readonly: true, optional: false}

func visibilityField(region string) Field {
	return Field{name: "visibility", typeOf: StringConst(region), readonly: true, optional: false}
}

// ObjectMemberKind returns the kind of the member as a subtype of "field"|"method"|"remote-method"|"resource-method"
func ObjectMemberKind(ctx Context, name, ty SemType) SemType {
	objectTy := convertObjectToMappingTy(ctx, ty)
	if IsZero(objectTy) {
		return SemType{}
	}
	memberMap := mappingMemberTypeInner(ctx, objectTy, name)
	return mappingMemberTypeInner(ctx, memberMap, StringConst("kind"))
}

// ObjectMemberVisibility returns the type of the member's visibility region string
func ObjectMemberVisibility(ctx Context, name, ty SemType) SemType {
	objectTy := convertObjectToMappingTy(ctx, ty)
	if IsZero(objectTy) {
		return SemType{}
	}
	memberMap := mappingMemberTypeInner(ctx, objectTy, name)
	return mappingMemberTypeInner(ctx, memberMap, StringConst("visibility"))
}

// ObjectMemberType returns the type of the member
func ObjectMemberType(ctx Context, name, ty SemType) SemType {
	objectTy := convertObjectToMappingTy(ctx, ty)
	if IsZero(objectTy) {
		return SemType{}
	}
	memberMap := mappingMemberTypeInner(ctx, objectTy, name)
	return mappingMemberTypeInner(ctx, memberMap, StringConst("value"))
}

// convertObjectToMappingTy structurally reinterprets the object component as a mapping type.
func convertObjectToMappingTy(_ Context, ty SemType) SemType {
	objectTy := Intersect(ty, Object)
	bdd := subtypeDataAt(objectTy, btObject)
	if isNothingSubtype(bdd) {
		return SemType{}
	}
	return createBasicSemType(btMapping, bdd)
}

func convertMappingToObjectTy(_ Context, ty SemType) SemType {
	mappingTy := Intersect(ty, Mapping)
	bdd := subtypeDataAt(mappingTy, btMapping)
	if isNothingSubtype(bdd) {
		return SemType{}
	}
	return createBasicSemType(btObject, bdd)
}
