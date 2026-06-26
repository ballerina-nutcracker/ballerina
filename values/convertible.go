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

package values

import (
	"ballerina-lang-go/semtypes"
)

func isNilable(target semtypes.SemType) bool {
	return semtypes.ContainsBasicType(target, semtypes.NIL)
}

func mappingFieldType(tc semtypes.Context, target semtypes.SemType, atomic *semtypes.MappingAtomicType, key string) semtypes.SemType {
	if atomic != nil {
		for _, name := range atomic.Names {
			if name == key {
				return atomic.FieldInnerVal(key)
			}
		}
	}
	return semtypes.MappingMemberTypeInnerVal(tc, target, semtypes.StringConst(key))
}

func isClosedRecord(atomic *semtypes.MappingAtomicType) bool {
	restTy := atomic.FieldInnerVal("\x00")
	return semtypes.IsNever(restTy)
}

func fieldMayOmitKey(tc semtypes.Context, target semtypes.SemType, name string) bool {
	return semtypes.AllMappingAtomsHaveOptionalFieldByName(tc, target, name)
}

func isStructuredValue(value BalValue) bool {
	switch value.(type) {
	case *List, *Map:
		return true
	default:
		return false
	}
}

var simpleBasicTypes = []semtypes.SemType{
	semtypes.NIL, semtypes.BOOLEAN, semtypes.INT, semtypes.FLOAT, semtypes.DECIMAL,
	semtypes.STRING, semtypes.XML, semtypes.ERROR,
}

func unionMemberTypes(tc semtypes.Context, ty semtypes.SemType) []semtypes.SemType {
	var members []semtypes.SemType
	basic := semtypes.WidenToBasicTypes(ty)

	if semtypes.ContainsBasicType(basic, semtypes.MAPPING) {
		mappingTy := semtypes.Intersect(ty, semtypes.MAPPING)
		if !semtypes.IsEmpty(tc, mappingTy) {
			for _, alt := range semtypes.MappingAlternatives(tc, mappingTy) {
				members = append(members, alt.SemType)
			}
		}
	}
	if semtypes.ContainsBasicType(basic, semtypes.LIST) {
		listTy := semtypes.Intersect(ty, semtypes.LIST)
		if !semtypes.IsEmpty(tc, listTy) {
			for _, alt := range semtypes.ListAlternatives(tc, listTy) {
				members = append(members, alt.SemType)
			}
		}
	}

	for _, bt := range simpleBasicTypes {
		if semtypes.ContainsBasicType(basic, bt) {
			member := semtypes.Intersect(ty, bt)
			if !semtypes.IsEmpty(tc, member) {
				members = append(members, member)
			}
		}
	}
	return members
}
