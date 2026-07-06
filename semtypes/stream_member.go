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

// StreamValueType returns T from stream<T, C>, or nil if streamTy is
// not a subtype of STREAM.
func StreamValueType(cx Context, streamTy SemType) SemType {
	return streamMemberAt(cx, streamTy, 0)
}

// StreamCompletionType returns C from stream<T, C>, or nil if streamTy is
// not a subtype of STREAM.
func StreamCompletionType(cx Context, streamTy SemType) SemType {
	return streamMemberAt(cx, streamTy, 1)
}

func streamMemberAt(cx Context, streamTy SemType, index int64) SemType {
	if !IsSubtypeSimple(streamTy, STREAM) {
		return SemType{}
	}
	if streamTy.some() == 0 {
		return bareStreamMember(index)
	}
	return ListMemberTypeInnerVal(cx, convertStreamToListTy(cx, streamTy), IntConst(index))
}

func bareStreamMember(index int64) SemType {
	switch index {
	case 0:
		return VAL
	case 1:
		return Union(ERROR, NIL)
	default:
		panic("invalid stream member index")
	}
}

// CreateStreamImplementorType returns the object-type that an implementor for
// `new stream<valueTy, completionTy>(impl)` must conform to.
func CreateStreamImplementorType(cx Context, valueTy, completionTy SemType) SemType {
	if t, ok := cx.streamImplementorMemo(valueTy, completionTy); ok {
		return t
	}
	env := cx.Env()
	nextRecordDefn := NewMappingDefinition()
	nextRecord := nextRecordDefn.DefineMappingTypeWrapped(env,
		[]Field{FieldFrom("value", valueTy, false, false)}, NEVER)
	nextReturn := Union(nextRecord, completionTy)
	closeReturn := Union(completionTy, NIL)

	nextFnTy := streamMethodFunctionType(env, nextReturn)
	closeFnTy := streamMethodFunctionType(env, closeReturn)

	quals := ObjectQualifiersFrom(false, false, NetworkQualifierNone)
	nextOnlyDefn := NewObjectDefinition()
	nextOnly := nextOnlyDefn.Define(env, quals, []Member{
		streamPublicIsolatedMethod("next", nextFnTy),
	})
	withCloseDefn := NewObjectDefinition()
	withClose := withCloseDefn.Define(env, quals, []Member{
		streamPublicIsolatedMethod("next", nextFnTy),
		streamPublicIsolatedMethod("close", closeFnTy),
	})
	result := Union(nextOnly, withClose)
	cx.setStreamImplementorMemo(valueTy, completionTy, result)
	return result
}

func streamMethodFunctionType(env Env, returnTy SemType) SemType {
	paramListDefn := NewListDefinition()
	paramList := paramListDefn.DefineListTypeWrapped(env, nil, 0, NEVER, CellMutability_CELL_MUT_NONE)
	fnDefn := NewFunctionDefinition()
	return fnDefn.Define(env, paramList, returnTy, FunctionQualifiersFrom(env, true, false))
}

func streamPublicIsolatedMethod(name string, fnTy SemType) Member {
	return Member{
		Name:       name,
		ValueTy:    fnTy,
		Kind:       MemberKindMethod,
		Visibility: VisibilityPublic,
		Immutable:  false,
	}
}

func convertStreamToListTy(ctx Context, ty SemType) SemType {
	streamTy := Intersect(ty, STREAM)
	if IsEmpty(ctx, streamTy) {
		return SemType{}
	}
	bdd := subtypeData(streamTy, BTStream)
	return createBasicSemType(BTList, bdd)
}
