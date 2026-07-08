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

// ListenerTy returns object type Listener<T,A>,
// where T is a a subtype of service object {} and A is a subtype of string[]|string|()
//
//	object {
//	   public function attach(T svc, A attachPoint) returns error?;
//	   public function detach(T svc) returns error?;
//	   public function start() returns error?;
//	   public function gracefulStop() returns error?;
//	   public function immediateStop() returns error?;
//	}
func ListenerTy(cx Context, t, a SemType) SemType {
	if ty, ok := cx.listenerMemo(t, a); ok {
		return ty
	}
	env := cx.Env()
	errorOrNil := Union(ERROR, NIL)

	attachFnTy := listenerMethodType(env, []SemType{t, a}, errorOrNil)
	detachFnTy := listenerMethodType(env, []SemType{t}, errorOrNil)
	startFnTy := listenerMethodType(env, nil, errorOrNil)
	gracefulStopFnTy := listenerMethodType(env, nil, errorOrNil)
	immediateStopFnTy := listenerMethodType(env, nil, errorOrNil)

	quals := ObjectQualifiersFrom(false, false, NetworkQualifierNone)
	od := NewObjectDefinition()
	listenerTy := od.Define(env, quals, []Member{
		listenerPublicMethod("attach", attachFnTy),
		listenerPublicMethod("detach", detachFnTy),
		listenerPublicMethod("start", startFnTy),
		listenerPublicMethod("gracefulStop", gracefulStopFnTy),
		listenerPublicMethod("immediateStop", immediateStopFnTy),
	})
	cx.setListenerMemo(t, a, listenerTy)
	return listenerTy
}

func listenerMethodType(env Env, paramTys []SemType, returnTy SemType) SemType {
	paramListDefn := NewListDefinition()
	paramList := paramListDefn.DefineListTypeWrapped(env, paramTys, len(paramTys), NEVER, CellMutability_CELL_MUT_NONE)
	fnDefn := NewFunctionDefinition()
	return fnDefn.Define(env, paramList, returnTy, FunctionQualifiersFrom(env, false, false))
}

func listenerPublicMethod(name string, fnTy SemType) Member {
	return Member{
		Name:       name,
		ValueTy:    fnTy,
		Kind:       MemberKindMethod,
		Visibility: VisibilityPublic,
		Immutable:  true,
	}
}
