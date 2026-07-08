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

// NetworkQualifier represents the network qualifier of an object (client, service, or none)
// Migrated from ObjectQualifiers.java:58
type NetworkQualifier uint8

const (
	NetworkQualifierClient NetworkQualifier = iota
	NetworkQualifierService
	NetworkQualifierNone
)

var (
	networkQualifierClientTag = StringConst("client")
	networkQualifierClient    = Field{Name: "network", Ty: networkQualifierClientTag, Ro: true, Opt: false}

	networkQualifierServiceTag = StringConst("service")
	networkQualifierService    = Field{Name: "network", Ty: networkQualifierServiceTag, Ro: true, Opt: false}

	// Object can't be both client and service, which is enforced by the enum. We are using a union here so that
	// if this is none it matches both
	networkQualifierNone = Field{Name: "network", Ty: Union(networkQualifierClientTag, networkQualifierServiceTag), Ro: true, Opt: false}
)

// field returns the Field representation for this NetworkQualifier
// Migrated from ObjectQualifiers.java:73
func (nq *NetworkQualifier) field() Field {
	switch *nq {
	case NetworkQualifierClient:
		return networkQualifierClient
	case NetworkQualifierService:
		return networkQualifierService
	case NetworkQualifierNone:
		return networkQualifierNone
	default:
		panic("invalid network qualifier")
	}
}

func IsIsolatedObject(ctx Context, ty SemType) bool {
	objectTy := convertObjectToMappingTy(ctx, ty)
	if IsZero(objectTy) {
		return false
	}
	qualifiersMap := mappingMemberTypeInner(ctx, objectTy, StringConst("$qualifiers"))
	if IsZero(qualifiersMap) {
		return false
	}
	isolatedTy := mappingMemberTypeInner(ctx, qualifiersMap, StringConst("isolated"))
	if IsZero(isolatedTy) {
		return false
	}
	return IsSubtype(ctx, isolatedTy, BooleanConst(true))
}

func IsNetworkInteractionObject(ctx Context, ty SemType) bool {
	return IsClientObject(ctx, ty) || IsServiceObject(ctx, ty)
}

func IsClientObject(ctx Context, ty SemType) bool {
	return objectHasNetworkQualifier(ctx, ty, networkQualifierClientTag)
}

func IsServiceObject(ctx Context, ty SemType) bool {
	return objectHasNetworkQualifier(ctx, ty, networkQualifierServiceTag)
}

func objectHasNetworkQualifier(ctx Context, ty SemType, qualifierTag SemType) bool {
	objectTy := convertObjectToMappingTy(ctx, ty)
	if IsZero(objectTy) {
		return false
	}
	qualifiersMap := mappingMemberTypeInner(ctx, objectTy, StringConst("$qualifiers"))
	if IsZero(qualifiersMap) {
		return false
	}
	networkTy := mappingMemberTypeInner(ctx, qualifiersMap, StringConst("network"))
	if IsZero(networkTy) {
		return false
	}
	return IsSubtype(ctx, networkTy, qualifierTag)
}

// ObjectQualifiers represents object-type-quals in the spec
// Migrated from ObjectQualifiers.java:43
type ObjectQualifiers struct {
	isolated         bool
	readonly         bool
	networkQualifier NetworkQualifier
}

// ObjectQualifiersDEFAULT is the default ObjectQualifiers instance
// Migrated from ObjectQualifiers.java:45
var ObjectQualifiersDEFAULT = ObjectQualifiers{isolated: false, readonly: false, networkQualifier: NetworkQualifierNone}

// defaultQualifiers returns the default ObjectQualifiers instance
// Migrated from ObjectQualifiers.java:47
func defaultQualifiers() ObjectQualifiers {
	return ObjectQualifiersDEFAULT
}

// ObjectQualifiersFrom creates an ObjectQualifiers instance with the given parameters
// Migrated from ObjectQualifiers.java:51
func ObjectQualifiersFrom(isolated bool, readonly bool, networkQualifier NetworkQualifier) ObjectQualifiers {
	if networkQualifier == NetworkQualifierNone && !isolated {
		return defaultQualifiers()
	}
	return ObjectQualifiers{isolated: isolated, readonly: readonly, networkQualifier: networkQualifier}
}

// Field creates a CellField representing these qualifiers
// Migrated from ObjectQualifiers.java:82
func (oq *ObjectQualifiers) Field(env Env) CellField {
	md := NewMappingDefinition()
	var isolatedField Field
	if oq.isolated {
		isolatedField = Field{Name: "isolated", Ty: BooleanConst(true), Ro: true, Opt: false}
	} else {
		isolatedField = Field{Name: "isolated", Ty: BOOLEAN, Ro: true, Opt: false}
	}
	networkField := oq.networkQualifier.field()
	ty := md.DefineMappingTypeWrappedWithEnvFieldsSemTypeCellMutability(
		env,
		[]Field{isolatedField, networkField},
		NEVER,
		CellMutability_CELL_MUT_NONE,
	)
	return cellFieldFrom("$qualifiers", cellContaining(env, ty))
}
