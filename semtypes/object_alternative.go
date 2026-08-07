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

type ObjectAlternative struct {
	ObjectType SemType
	InitFnType SemType
}

func IsAtomicObjectType(cx Context, t SemType) bool {
	return IsSubtype(cx, t, OBJECT) && len(ObjectAlternatives(cx, t)) == 1
}

func ObjectAlternatives(cx Context, t SemType) []ObjectAlternative {
	mappingTy := convertObjectToMappingTy(cx, t)
	mappingAlternatives := MappingAlternatives(cx, mappingTy)
	var alts []ObjectAlternative
	initKey := StringConst("init")
	for _, each := range mappingAlternatives {
		if len(each.neg) > 0 {
			continue
		}
		objectTy := convertMappingToObjectTy(cx, each.SemType)
		initTy := ObjectMemberType(cx, initKey, objectTy)
		alts = append(alts, ObjectAlternative{
			ObjectType: objectTy,
			InitFnType: initTy,
		})
	}

	return alts
}
