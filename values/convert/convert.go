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
	var unionErrors []string
	convertibleType, err := getConvertibleType(tc, value, targetType, &unionErrors, true)
	if err != nil {
		return nil, wrapConversionError(tc, value, targetType, err)
	}
	return convert(tc, value, convertibleType, targetType, &unionErrors), nil
}

func convert(tc semtypes.Context, value values.BalValue, convertibleType, targetType semtypes.SemType,
	unionErrors *[]string,
) values.BalValue {
	inherentType := convertibleType

	if value == nil {
		return nil
	}

	if isLikeType(tc, value, convertibleType, false) {
		return cloneValue(tc, value, inherentType)
	}

	switch value := value.(type) {
	case *values.Map:
		if semtypes.IsSubtypeSimple(convertibleType, semtypes.MAPPING) {
			return convertMapping(tc, value, inherentType, targetType, unionErrors)
		}
	case *values.List:
		if semtypes.IsSubtypeSimple(convertibleType, semtypes.LIST) {
			return convertList(tc, value, inherentType, targetType, unionErrors)
		}
	}

	if isNumericConvertible(tc, value, convertibleType) {
		converted, err := convertNumeric(tc, value, convertibleType)
		if err != nil {
			panic(err)
		}
		return converted
	}

	panic("convert: value is not convertible after getConvertibleType")
}

func convertMapping(tc semtypes.Context, source *values.Map, inherentType, requestedTarget semtypes.SemType,
	unionErrors *[]string,
) values.BalValue {
	atomic := semtypes.ToMappingAtomicType(tc, inherentType)
	if atomic == nil {
		panic("convert: mapping target has no atomic representation")
	}
	entries := make([]values.MapEntry, 0, source.Len()+len(atomic.Names))
	seen := make(map[string]struct{}, source.Len())
	for _, key := range source.Keys() {
		seen[key] = struct{}{}
		fieldTy := mappingFieldType(tc, inherentType, atomic, key)
		val, _ := source.Get(key)
		convertibleFieldTy, err := getConvertibleType(tc, val, fieldTy, unionErrors, true)
		if err != nil {
			panic(err)
		}
		converted := convert(tc, val, convertibleFieldTy, fieldTy, unionErrors)
		entries = append(entries, values.MapEntry{Key: key, Value: converted})
	}
	for _, name := range atomic.Names {
		if _, ok := seen[name]; ok {
			continue
		}
		if !fieldNeedsNilWhenMissing(tc, inherentType, name, atomic) {
			continue
		}
		entries = append(entries, values.MapEntry{Key: name, Value: nil})
	}
	readonly := semtypes.IsSubtype(tc, requestedTarget, semtypes.VAL_READONLY)
	return values.NewMap(inherentType, atomic, readonly, entries)
}

func convertList(tc semtypes.Context, source *values.List, inherentType, requestedTarget semtypes.SemType,
	unionErrors *[]string,
) values.BalValue {
	atomic := semtypes.ToListAtomicType(tc, inherentType)
	if atomic == nil {
		panic("convert: list target has no atomic representation")
	}
	items := make([]values.BalValue, source.Len())
	for i := 0; i < source.Len(); i++ {
		memberTy := atomic.MemberAtInnerVal(i)
		narrowedMemberTy, err := getConvertibleType(tc, source.Get(i), memberTy, unionErrors, true)
		if err != nil {
			panic(err)
		}
		items[i] = convert(tc, source.Get(i), narrowedMemberTy, memberTy, unionErrors)
	}
	restFiller, _ := values.FillerFactoryFor(tc, atomic.Rest())
	readonly := semtypes.IsSubtype(tc, requestedTarget, semtypes.VAL_READONLY)
	return values.NewList(inherentType, atomic, readonly, restFiller, len(items), items)
}
