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

package common

import "github.com/ballerina-nutcracker/ballerina/semtypes"

// ListenerTypes structurally checks whether ty is a valid listener object type
// and, on success, returns its projected service-target and attach-point types.
func ListenerTypes(cx semtypes.Context, ty semtypes.SemType, attachPointBound semtypes.SemType) (semtypes.SemType, semtypes.SemType, bool) {
	attachFnTy := semtypes.ObjectMemberType(cx, semtypes.StringConst("attach"), ty)
	if semtypes.IsZero(attachFnTy) {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	paramList := semtypes.FunctionParamListType(cx, attachFnTy)
	if semtypes.IsZero(paramList) {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	t := semtypes.ListMemberTypeInnerVal(cx, paramList, semtypes.IntConst(0))
	a := semtypes.ListMemberTypeInnerVal(cx, paramList, semtypes.IntConst(1))
	if !semtypes.IsSubtype(cx, t, semtypes.CreateServiceObject(cx)) {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	if !semtypes.IsSubtype(cx, a, attachPointBound) {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	if !semtypes.IsSubtype(cx, ty, semtypes.ListenerTy(cx, t, a)) {
		return semtypes.SemType{}, semtypes.SemType{}, false
	}
	return t, a, true
}

// ListenerAttachPointBound returns `string[] | string | ()`.
func ListenerAttachPointBound(cx semtypes.Context) semtypes.SemType {
	listDefn := semtypes.NewListDefinition()
	stringArr := listDefn.Define(cx.Env(), nil, semtypes.ListRest(semtypes.String))
	return semtypes.Union(stringArr, semtypes.Union(semtypes.String, semtypes.Nil))
}
