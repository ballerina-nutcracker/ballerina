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

func isLikeType(tc semtypes.Context, value BalValue, target semtypes.SemType, allowNumeric bool) bool {
	valueTy := SemTypeForValue(value)
	if semtypes.IsSubtype(tc, valueTy, target) {
		return true
	}
	if !allowNumeric {
		return false
	}
	return isNumericConvertible(tc, value, target)
}

func isNilable(target semtypes.SemType) bool {
	return semtypes.ContainsBasicType(target, semtypes.NIL)
}

func getConvertibleType(tc semtypes.Context, value BalValue, target semtypes.SemType,
	unionErrors *[]string, allowNumeric bool,
) (semtypes.SemType, error) {
	if value == nil {
		if isNilable(target) {
			return target, nil
		}
		return semtypes.SemType{}, cannotConvertNil(tc, target)
	}

	if semtypes.IsSameType(tc, target, semtypes.CreateJSON(tc)) {
		if isLikeType(tc, value, target, allowNumeric) {
			return target, nil
		}
	}

	if semtypes.IsSameType(tc, target, semtypes.CreateAnydata(tc)) {
		valueTy := SemTypeForValue(value)
		if semtypes.IsSubtype(tc, valueTy, target) {
			return target, nil
		}
	}

	if members := unionMemberTypes(tc, target); len(members) > 1 {
		return getConvertibleUnionMember(tc, value, target, members, unionErrors, allowNumeric)
	}

	switch value := value.(type) {
	case *Map:
		if semtypes.IsSubtypeSimple(target, semtypes.MAPPING) {
			if isConvertibleMapping(tc, value, target, unionErrors, allowNumeric) {
				return target, nil
			}
		}
	case *List:
		if semtypes.IsSubtypeSimple(target, semtypes.LIST) {
			if isConvertibleList(tc, value, target, unionErrors, allowNumeric) {
				return target, nil
			}
		}
	default:
		if isLikeType(tc, value, target, allowNumeric) {
			return target, nil
		}
	}

	return semtypes.SemType{}, incompatibleConversion(tc, value, target)
}

func getConvertibleUnionMember(tc semtypes.Context, value BalValue, target semtypes.SemType,
	members []semtypes.SemType, unionErrors *[]string, allowNumeric bool,
) (semtypes.SemType, error) {
	if isStructuredValue(value) {
		initial := len(*unionErrors)
		*unionErrors = append(*unionErrors, "{")
		for i, member := range members {
			if i > 0 {
				*unionErrors = append(*unionErrors, "or")
			}
			before := len(*unionErrors)
			convertible, err := getConvertibleType(tc, value, member, unionErrors, allowNumeric)
			if err == nil {
				*unionErrors = (*unionErrors)[:initial]
				return convertible, nil
			}
			if len(*unionErrors) == before {
				*unionErrors = append(*unionErrors, err.Error())
			}
		}
		*unionErrors = append(*unionErrors, "}")
		return semtypes.SemType{}, newConversionFailure(unionErrorMessage((*unionErrors)[initial:]))
	}

	for _, member := range members {
		if isLikeType(tc, value, member, false) {
			return getConvertibleType(tc, value, member, unionErrors, false)
		}
	}
	for _, member := range members {
		if convertible, err := getConvertibleType(tc, value, member, unionErrors, allowNumeric); err == nil {
			return convertible, nil
		}
	}
	return semtypes.SemType{}, incompatibleConversion(tc, value, target)
}

func isConvertibleList(tc semtypes.Context, source *List, target semtypes.SemType,
	unionErrors *[]string, allowNumeric bool,
) bool {
	atomic := semtypes.ToListAtomicType(tc, target)
	if atomic == nil {
		return false
	}
	fixedLen := atomic.Members.FixedLength
	if semtypes.IsNever(atomic.Rest()) {
		if source.Len() != fixedLen {
			return false
		}
	} else if source.Len() < fixedLen {
		return false
	}
	for i := 0; i < source.Len(); i++ {
		memberTy := atomic.MemberAtInnerVal(i)
		if _, err := getConvertibleType(tc, source.Get(i), memberTy, unionErrors, allowNumeric); err != nil {
			return false
		}
	}
	return true
}

func isConvertibleMapping(tc semtypes.Context, source *Map, target semtypes.SemType,
	unionErrors *[]string, allowNumeric bool,
) bool {
	atomic := semtypes.ToMappingAtomicType(tc, target)
	if atomic == nil {
		return false
	}

	declared := make(map[string]struct{}, len(atomic.Names))
	for _, name := range atomic.Names {
		declared[name] = struct{}{}
		if _, ok := source.Get(name); ok {
			continue
		}
		if fieldMayOmitKey(tc, target, name) {
			continue
		}
		if !fieldNeedsNilWhenMissing(tc, target, name, atomic) {
			return false
		}
		fieldTy := atomic.FieldInnerVal(name)
		if _, err := getConvertibleType(tc, nil, fieldTy, unionErrors, allowNumeric); err != nil {
			return false
		}
	}

	closed := isClosedRecord(atomic)
	for _, key := range source.Keys() {
		if closed {
			if _, ok := declared[key]; !ok {
				return false
			}
		}
		fieldTy := mappingFieldType(tc, target, atomic, key)
		val, _ := source.Get(key)
		if _, err := getConvertibleType(tc, val, fieldTy, unionErrors, allowNumeric); err != nil {
			return false
		}
	}
	return true
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

func fieldNeedsNilWhenMissing(tc semtypes.Context, target semtypes.SemType, name string, atomic *semtypes.MappingAtomicType) bool {
	if semtypes.AllMappingAtomsHaveOptionalFieldByName(tc, target, name) {
		return false
	}
	return isNilable(atomic.FieldInnerVal(name))
}

func isStructuredValue(value BalValue) bool {
	switch value.(type) {
	case *List, *Map:
		return true
	default:
		return false
	}
}

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
