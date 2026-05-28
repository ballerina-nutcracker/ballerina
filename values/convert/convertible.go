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
	"fmt"

	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

type typeValuePair struct {
	valueKey  string
	targetKey string
}

type convertState struct {
	unresolved map[typeValuePair]struct{}
	errors     []string
}

func getConvertibleType(tc semtypes.Context, value values.BalValue, target semtypes.SemType,
	state *convertState, allowNumeric bool,
) (semtypes.SemType, error) {
	target = effectiveTargetType(tc, target)

	if value == nil {
		if isNilable(tc, target) {
			return target, nil
		}
		return semtypes.SemType{}, cannotConvertNil(tc, target)
	}

	if semtypes.IsSubtypeSimple(target, semtypes.TYPEDESC) {
		if constraint := semtypes.TypedescConstraint(tc, target); !semtypes.IsZero(constraint) {
			return getConvertibleType(tc, value, constraint, state, allowNumeric)
		}
	}

	if semtypes.IsSameType(tc, target, semtypes.CreateJSON(tc)) {
		if isLikeType(tc, value, target, allowNumeric) {
			return target, nil
		}
		return semtypes.SemType{}, incompatibleConversion(tc, value, target)
	}

	if semtypes.IsSameType(tc, target, semtypes.CreateAnydata(tc)) {
		valueTy := values.SemTypeForValue(value)
		if semtypes.IsSubtype(tc, valueTy, target) {
			return target, nil
		}
	}

	if isUnionType(tc, target) {
		return getConvertibleUnionMember(tc, value, target, state, allowNumeric)
	}

	pair := state.pair(value, target, tc)
	if _, ok := state.unresolved[pair]; ok {
		sourceTy := values.SemTypeForValue(value)
		return semtypes.SemType{}, cyclicValueReference(tc, sourceTy)
	}
	state.unresolved[pair] = struct{}{}
	defer delete(state.unresolved, pair)

	switch value.(type) {
	case *values.Map:
		if semtypes.IsSubtypeSimple(target, semtypes.MAPPING) {
			if isConvertibleMapping(tc, value.(*values.Map), target, state, allowNumeric) {
				return target, nil
			}
		}
	case *values.List:
		if semtypes.IsSubtypeSimple(target, semtypes.LIST) {
			if isConvertibleList(tc, value.(*values.List), target, state, allowNumeric) {
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

func getConvertibleUnionMember(tc semtypes.Context, value values.BalValue, target semtypes.SemType,
	state *convertState, allowNumeric bool,
) (semtypes.SemType, error) {
	members := unionMemberTypes(tc, target)
	if isStructuredValue(value) {
		initialErrors := len(state.errors)
		state.errors = append(state.errors, "{")
		for i, member := range members {
			if i > 0 {
				state.errors = append(state.errors, "or")
			}
			before := len(state.errors)
			convertible, err := getConvertibleType(tc, value, member, state, allowNumeric)
			if err == nil {
				state.errors = state.errors[:initialErrors]
				return convertible, nil
			}
			if len(state.errors) == before {
				state.errors = append(state.errors, err.Error())
			}
		}
		state.errors = append(state.errors, "}")
		return semtypes.SemType{}, newConversionFailure(unionErrorMessage(state.errors[initialErrors:]))
	}

	for _, member := range members {
		if isLikeType(tc, value, member, false) {
			return getConvertibleType(tc, value, member, state, false)
		}
	}
	for _, member := range members {
		if convertible, err := getConvertibleType(tc, value, member, state, allowNumeric); err == nil {
			return convertible, nil
		}
	}
	return semtypes.SemType{}, incompatibleConversion(tc, value, target)
}

func isConvertibleList(tc semtypes.Context, source *values.List, target semtypes.SemType,
	state *convertState, allowNumeric bool,
) bool {
	atomic := semtypes.ToListAtomicType(tc, target)
	if atomic == nil {
		return false
	}
	for i := 0; i < source.Len(); i++ {
		memberTy := atomic.MemberAtInnerVal(i)
		if _, err := getConvertibleType(tc, source.Get(i), memberTy, state, allowNumeric); err != nil {
			return false
		}
	}
	return true
}

func isConvertibleMapping(tc semtypes.Context, source *values.Map, target semtypes.SemType,
	state *convertState, allowNumeric bool,
) bool {
	atomic := semtypes.ToMappingAtomicType(tc, target)
	if atomic == nil {
		return false
	}

	for _, name := range atomic.Names {
		if _, ok := source.Get(name); ok {
			continue
		}
		if !fieldMayOmitKey(tc, target, name, atomic) {
			return false
		}
	}

	closed := isClosedRecord(tc, atomic)
	declared := make(map[string]struct{}, len(atomic.Names))
	for _, name := range atomic.Names {
		declared[name] = struct{}{}
	}

	for _, key := range source.Keys() {
		if closed {
			if _, ok := declared[key]; !ok {
				return false
			}
		}
		fieldTy := mappingFieldType(tc, target, atomic, key)
		val, _ := source.Get(key)
		if _, err := getConvertibleType(tc, val, fieldTy, state, allowNumeric); err != nil {
			return false
		}
	}
	return true
}

func mappingFieldType(tc semtypes.Context, target semtypes.SemType, atomic *semtypes.MappingAtomicType, key string) semtypes.SemType {
	if atomic != nil {
		for _, name := range atomic.Names {
			if name == key {
				return effectiveTargetType(tc, atomic.FieldInnerVal(key))
			}
		}
	}
	return effectiveTargetType(tc, semtypes.MappingMemberTypeInnerVal(tc, target, semtypes.StringConst(key)))
}

func effectiveTargetType(tc semtypes.Context, target semtypes.SemType) semtypes.SemType {
	target = targetFromTypedesc(tc, target)
	if semtypes.IsSubtype(tc, target, semtypes.VAL_READONLY) && !semtypes.IsSameType(tc, target, semtypes.VAL_READONLY) {
		effective := semtypes.Diff(target, semtypes.VAL_READONLY)
		if !semtypes.IsEmpty(tc, effective) {
			return effectiveTargetType(tc, effective)
		}
	}
	return target
}

func isClosedRecord(_ semtypes.Context, atomic *semtypes.MappingAtomicType) bool {
	if atomic == nil {
		return false
	}
	restTy := atomic.FieldInnerVal("\x00")
	return semtypes.IsNever(restTy)
}

func fieldMayOmitKey(tc semtypes.Context, target semtypes.SemType, name string, atomic *semtypes.MappingAtomicType) bool {
	if semtypes.AllMappingAtomsHaveOptionalFieldByName(tc, target, name) {
		return true
	}
	return isNilable(tc, atomic.FieldInnerVal(name))
}

func fieldNeedsNilWhenMissing(tc semtypes.Context, target semtypes.SemType, name string, atomic *semtypes.MappingAtomicType) bool {
	if semtypes.AllMappingAtomsHaveOptionalFieldByName(tc, target, name) {
		return false
	}
	return isNilable(tc, atomic.FieldInnerVal(name))
}

func newConvertState() *convertState {
	return &convertState{unresolved: make(map[typeValuePair]struct{})}
}

func valueKey(value values.BalValue) string {
	if value == nil {
		return "nil"
	}
	return fmt.Sprintf("%T:%p", value, value)
}

func (s *convertState) pair(value values.BalValue, target semtypes.SemType, tc semtypes.Context) typeValuePair {
	return typeValuePair{
		valueKey:  valueKey(value),
		targetKey: semtypes.ToString(tc, target),
	}
}

func targetFromTypedesc(tc semtypes.Context, target semtypes.SemType) semtypes.SemType {
	if semtypes.IsSubtypeSimple(target, semtypes.TYPEDESC) {
		if constraint := semtypes.TypedescConstraint(tc, target); !semtypes.IsZero(constraint) {
			return constraint
		}
	}
	return target
}

func isStructuredValue(value values.BalValue) bool {
	switch value.(type) {
	case *values.List, *values.Map:
		return true
	default:
		return false
	}
}

func unionMemberTypes(tc semtypes.Context, ty semtypes.SemType) []semtypes.SemType {
	if semtypes.IsEmpty(tc, ty) {
		return nil
	}

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

	if len(members) == 0 {
		return []semtypes.SemType{ty}
	}
	return members
}

func mappingUnionMembers(tc semtypes.Context, mappingTy semtypes.SemType) []semtypes.SemType {
	if semtypes.IsSameType(tc, mappingTy, semtypes.MAPPING) {
		return []semtypes.SemType{mappingTy}
	}
	alts := semtypes.MappingAlternatives(tc, mappingTy)
	if len(alts) == 0 {
		return []semtypes.SemType{mappingTy}
	}
	members := make([]semtypes.SemType, 0, len(alts))
	for _, alt := range alts {
		members = append(members, alt.SemType)
	}
	return members
}

func listUnionMembers(tc semtypes.Context, listTy semtypes.SemType) []semtypes.SemType {
	if semtypes.IsSameType(tc, listTy, semtypes.LIST) {
		return []semtypes.SemType{listTy}
	}
	alts := semtypes.ListAlternatives(tc, listTy)
	if len(alts) == 0 {
		return []semtypes.SemType{listTy}
	}
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
