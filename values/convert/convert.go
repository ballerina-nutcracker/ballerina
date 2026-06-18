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
	"ballerina-lang-go/values/core"
)

// Convert converts a value to the given target type.
// On failure it returns a lang.value ConversionError as *core.Error.
func Convert(tc semtypes.Context, value core.BalValue, targetType semtypes.SemType) (core.BalValue, *core.Error) {
	var unionErrors []string
	convertibleType, err := getConvertibleType(tc, value, targetType, &unionErrors, true)
	if err != nil {
		return nil, wrapConversionError(err)
	}
	return convert(tc, value, convertibleType, targetType, &unionErrors), nil
}

func convert(tc semtypes.Context, value core.BalValue, convertibleType, targetType semtypes.SemType,
	unionErrors *[]string,
) core.BalValue {
	inherentType := convertibleType

	if value == nil {
		return nil
	}

	if isLikeType(tc, value, convertibleType, false) {
		return cloneValue(tc, value, inherentType)
	}

	switch value := value.(type) {
	case *core.Map:
		return convertMapping(tc, value, inherentType, unionErrors)
	case *core.List:
		return convertList(tc, value, inherentType, unionErrors)
	}

	converted, _ := convertNumeric(tc, value, convertibleType)
	return converted
}

func convertMapping(tc semtypes.Context, source *core.Map, inherentType semtypes.SemType,
	unionErrors *[]string,
) core.BalValue {
	atomic := semtypes.ToMappingAtomicType(tc, inherentType)
	entries := make([]core.MapEntry, 0, source.Len()+len(atomic.Names))
	seen := make(map[string]struct{}, source.Len())
	for _, key := range source.Keys() {
		seen[key] = struct{}{}
		fieldTy := mappingFieldType(tc, inherentType, atomic, key)
		val, _ := source.Get(key)
		convertibleFieldTy, _ := getConvertibleType(tc, val, fieldTy, unionErrors, true)
		converted := convert(tc, val, convertibleFieldTy, fieldTy, unionErrors)
		entries = append(entries, core.MapEntry{Key: key, Value: converted})
	}
	for _, name := range atomic.Names {
		if _, ok := seen[name]; ok {
			continue
		}
		if !fieldNeedsNilWhenMissing(tc, inherentType, name, atomic) {
			continue
		}
		entries = append(entries, core.MapEntry{Key: name, Value: nil})
	}
	readonly := semtypes.IsSubtype(tc, inherentType, semtypes.VAL_READONLY)
	return core.NewMap(inherentType, atomic, readonly, entries)
}

func convertList(tc semtypes.Context, source *core.List, inherentType semtypes.SemType,
	unionErrors *[]string,
) core.BalValue {
	atomic := semtypes.ToListAtomicType(tc, inherentType)
	items := make([]core.BalValue, source.Len())
	for i := 0; i < source.Len(); i++ {
		memberTy := atomic.MemberAtInnerVal(i)
		narrowedMemberTy, _ := getConvertibleType(tc, source.Get(i), memberTy, unionErrors, true)
		items[i] = convert(tc, source.Get(i), narrowedMemberTy, memberTy, unionErrors)
	}
	restFiller, _ := core.FillerFactoryFor(tc, atomic.Rest())
	readonly := semtypes.IsSubtype(tc, inherentType, semtypes.VAL_READONLY)
	return core.NewList(inherentType, atomic, readonly, restFiller, len(items), items)
}
