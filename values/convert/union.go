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

package convert

import (
	"ballerina-lang-go/semtypes"
)

func unionMemberTypes(tc semtypes.Context, ty semtypes.SemType) []semtypes.SemType {
	var members []semtypes.SemType
	basic := semtypes.WidenToBasicTypes(ty)

	if semtypes.ContainsBasicType(basic, semtypes.MAPPING) {
		mappingTy := semtypes.Intersect(ty, semtypes.MAPPING)
		if !semtypes.IsEmpty(tc, mappingTy) {
			members = append(members, mappingUnionMembers(tc, mappingTy)...)
		}
	}
	if semtypes.ContainsBasicType(basic, semtypes.LIST) {
		listTy := semtypes.Intersect(ty, semtypes.LIST)
		if !semtypes.IsEmpty(tc, listTy) {
			members = append(members, listUnionMembers(tc, listTy)...)
		}
	}

	simpleBasics := []semtypes.SemType{
		semtypes.NIL, semtypes.BOOLEAN, semtypes.INT, semtypes.FLOAT, semtypes.DECIMAL,
		semtypes.STRING, semtypes.XML, semtypes.ERROR,
	}
	for _, bt := range simpleBasics {
		if semtypes.ContainsBasicType(basic, bt) {
			member := semtypes.Intersect(ty, bt)
			if !semtypes.IsEmpty(tc, member) {
				members = append(members, member)
			}
		}
	}
	return members
}

func mappingUnionMembers(tc semtypes.Context, mappingTy semtypes.SemType) []semtypes.SemType {
	alts := semtypes.MappingAlternatives(tc, mappingTy)
	members := make([]semtypes.SemType, 0, len(alts))
	for _, alt := range alts {
		members = append(members, alt.SemType)
	}
	return members
}

func listUnionMembers(tc semtypes.Context, listTy semtypes.SemType) []semtypes.SemType {
	alts := semtypes.ListAlternatives(tc, listTy)
	members := make([]semtypes.SemType, 0, len(alts))
	for _, alt := range alts {
		members = append(members, alt.SemType)
	}
	return members
}

func isUnionType(tc semtypes.Context, ty semtypes.SemType) bool {
	members := unionMemberTypes(tc, ty)
	return len(members) > 1
}
