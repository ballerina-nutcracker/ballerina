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
	ValueTy    SemType
	Kind       MemberKind
	Visibility Visibility
	Immutable  bool
}

func newMember(name string, valueTy SemType, kind MemberKind, visibility Visibility, immutable bool) *Member {
	return &Member{Name: name, ValueTy: valueTy, Kind: kind, Visibility: visibility, Immutable: immutable}
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
		return Field{Name: "kind", Ty: StringConst("field"), Ro: true, Opt: false}
	case MemberKindMethod:
		return Field{Name: "kind", Ty: StringConst("method"), Ro: true, Opt: false}
	case MemberKindRemoteMethod:
		return Field{Name: "kind", Ty: StringConst("remote-method"), Ro: true, Opt: false}
	case MemberKindResourceMethod:
		return Field{Name: "kind", Ty: StringConst("resource-method"), Ro: true, Opt: false}
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
	var ty = NEVER
	for _, each := range tys {
		ty = Union(ty, StringConst(each))
	}
	return Field{Name: "kind", Ty: ty, Ro: true, Opt: false}
}

type Visibility uint8

const (
	VisibilityPublic Visibility = iota
	VisibilityPrivate
)

var (
	visibilityPublicTag  = StringConst("public")
	visibilityPrivateTag = StringConst("private")
	visibilityAll        = Field{Name: "visibility", Ty: Union(visibilityPublicTag, visibilityPrivateTag), Ro: true, Opt: false}
)

func (v *Visibility) field() Field {
	switch *v {
	case VisibilityPublic:
		return Field{Name: "visibility", Ty: visibilityPublicTag, Ro: true, Opt: false}
	case VisibilityPrivate:
		return Field{Name: "visibility", Ty: visibilityPrivateTag, Ro: true, Opt: false}
	default:
		panic("invalid visibility")
	}
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

// ObjectMemberVisibility returns the visibility of the member as a subtype of "public"|"private"
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

func convertObjectToMappingTy(ctx Context, ty SemType) SemType {
	objectTy := Intersect(ty, OBJECT)
	if IsEmpty(ctx, objectTy) {
		return SemType{}
	}
	bdd := subtypeData(objectTy, BTObject)
	return createBasicSemType(BTMapping, bdd)
}

func convertMappingToObjectTy(ctx Context, ty SemType) SemType {
	mappingTy := Intersect(ty, MAPPING)
	if IsEmpty(ctx, mappingTy) {
		return SemType{}
	}
	bdd := subtypeData(mappingTy, BTMapping)
	return createBasicSemType(BTObject, bdd)
}
