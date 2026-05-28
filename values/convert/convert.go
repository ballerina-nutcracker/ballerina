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
	"ballerina-lang-go/values"
)

// Convert converts a value to the given target type.
// On failure it returns a lang.value ConversionError as *values.Error.
func Convert(tc semtypes.Context, value values.BalValue, targetType semtypes.SemType) (values.BalValue, *values.Error) {
	state := newConvertState()
	convertibleType, err := getConvertibleType(tc, value, targetType, state, true)
	if err != nil {
		return nil, wrapConversionError(tc, value, targetType, err)
	}
	converted, err := convert(tc, value, convertibleType, targetType, state)
	if err != nil {
		return nil, wrapConversionError(tc, value, targetType, err)
	}
	return converted, nil
}

func convert(tc semtypes.Context, value values.BalValue, convertibleType, targetType semtypes.SemType,
	state *convertState,
) (values.BalValue, error) {
	targetType = effectiveTargetType(tc, targetType)
	convertibleType = effectiveTargetType(tc, convertibleType)

	if value == nil {
		return nil, nil
	}

	pair := state.pair(value, convertibleType, tc)
	if _, ok := state.unresolved[pair]; ok {
		sourceTy := values.SemTypeForValue(value)
		return nil, cyclicValueReference(tc, sourceTy)
	}
	state.unresolved[pair] = struct{}{}
	defer delete(state.unresolved, pair)

	if isLikeType(tc, value, convertibleType, false) {
		return cloneValue(tc, value), nil
	}

	switch value.(type) {
	case *values.Map:
		if semtypes.IsSubtypeSimple(convertibleType, semtypes.MAPPING) {
			return convertMapping(tc, value.(*values.Map), convertibleType, state)
		}
	case *values.List:
		if semtypes.IsSubtypeSimple(convertibleType, semtypes.LIST) {
			return convertList(tc, value.(*values.List), convertibleType, state)
		}
	}

	if isNumericConvertible(tc, value, convertibleType) {
		converted, err := convertNumeric(tc, value, convertibleType)
		if err != nil {
			return nil, err
		}
		if !semtypes.IsSubtype(tc, values.SemTypeForValue(converted), convertibleType) {
			return nil, incompatibleConversion(tc, value, targetType)
		}
		return converted, nil
	}

	return nil, incompatibleConversion(tc, value, targetType)
}

func convertList(tc semtypes.Context, source *values.List, target semtypes.SemType, state *convertState) (values.BalValue, error) {
	atomic := semtypes.ToListAtomicType(tc, target)
	if atomic == nil {
		return nil, incompatibleConversion(tc, source, target)
	}
	items := make([]values.BalValue, source.Len())
	for i := 0; i < source.Len(); i++ {
		memberTy := atomic.MemberAtInnerVal(i)
		if _, err := getConvertibleType(tc, source.Get(i), memberTy, state, true); err != nil {
			return nil, err
		}
		converted, err := convert(tc, source.Get(i), memberTy, memberTy, state)
		if err != nil {
			return nil, err
		}
		items[i] = converted
	}
	restFiller, _ := values.FillerFactoryFor(tc, atomic.Rest())
	readonly := semtypes.IsSubtype(tc, target, semtypes.VAL_READONLY)
	return values.NewList(target, atomic, readonly, restFiller, len(items), items), nil
}

func convertMapping(tc semtypes.Context, source *values.Map, target semtypes.SemType, state *convertState) (values.BalValue, error) {
	atomic := semtypes.ToMappingAtomicType(tc, target)
	if atomic == nil {
		return nil, incompatibleConversion(tc, source, target)
	}
	entries := make([]values.MapEntry, 0, source.Len()+len(atomic.Names))
	seen := make(map[string]struct{}, source.Len())
	for _, key := range source.Keys() {
		seen[key] = struct{}{}
		fieldTy := mappingFieldType(tc, target, atomic, key)
		val, _ := source.Get(key)
		convertibleFieldTy, err := getConvertibleType(tc, val, fieldTy, state, true)
		if err != nil {
			return nil, err
		}
		converted, err := convert(tc, val, convertibleFieldTy, convertibleFieldTy, state)
		if err != nil {
			return nil, err
		}
		entries = append(entries, values.MapEntry{Key: key, Value: converted})
	}
	for _, name := range atomic.Names {
		if _, ok := seen[name]; ok {
			continue
		}
		if !fieldNeedsNilWhenMissing(tc, target, name, atomic) {
			continue
		}
		entries = append(entries, values.MapEntry{Key: name, Value: nil})
	}
	readonly := semtypes.IsSubtype(tc, target, semtypes.VAL_READONLY)
	return values.NewMap(target, atomic, readonly, entries), nil
}
