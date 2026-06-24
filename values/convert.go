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
	"math"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/semtypes"
)

// CloneWithType implements the cloneWithType abstract operation defined in the Ballerina spec
// (https://ballerina.io/spec/lang/master/#section_16.6).
//
// It constructs a value of targetType by deep-cloning value, applying the following conversions:
//   - the inherent type of any structural value comes from targetType
//   - numeric values may be converted between int, float, and decimal via NumericConvert
//   - if targetType is a record with default values, missing fields are filled from those defaults
//
// Note: currently only covers the json subset required by fromJsonWithType; support for
// the full anydata domain (xml, table, cycles) will be added as those types come into scope.
//
// On failure it returns a ConversionError wrapped as *Error.
func CloneWithType(tc semtypes.Context, value BalValue, targetType semtypes.SemType) (BalValue, *Error) {
	var unionErrors []string
	convertibleType, err := getConvertibleType(tc, value, targetType, &unionErrors, true)
	if err != nil {
		return nil, wrapConversionError(err)
	}
	return convert(tc, value, convertibleType, targetType, &unionErrors), nil
}

func convert(tc semtypes.Context, value BalValue, convertibleType, targetType semtypes.SemType,
	unionErrors *[]string,
) BalValue {
	inherentType := convertibleType

	if value == nil {
		return nil
	}

	if isLikeType(tc, value, convertibleType, false) {
		return cloneValue(tc, value, inherentType)
	}

	switch value := value.(type) {
	case *Map:
		return convertMapping(tc, value, inherentType, unionErrors)
	case *List:
		return convertList(tc, value, inherentType, unionErrors)
	}

	converted, _ := convertNumeric(tc, value, convertibleType)
	return converted
}

func convertMapping(tc semtypes.Context, source *Map, inherentType semtypes.SemType,
	unionErrors *[]string,
) BalValue {
	atomic := semtypes.ToMappingAtomicType(tc, inherentType)
	entries := make([]MapEntry, 0, source.Len()+len(atomic.Names))
	seen := make(map[string]struct{}, source.Len())
	for _, key := range source.Keys() {
		seen[key] = struct{}{}
		fieldTy := mappingFieldType(tc, inherentType, atomic, key)
		val, _ := source.Get(key)
		convertibleFieldTy, _ := getConvertibleType(tc, val, fieldTy, unionErrors, true)
		converted := convert(tc, val, convertibleFieldTy, fieldTy, unionErrors)
		entries = append(entries, MapEntry{Key: key, Value: converted})
	}
	for _, name := range atomic.Names {
		if _, ok := seen[name]; ok {
			continue
		}
		if !fieldNeedsNilWhenMissing(tc, inherentType, name, atomic) {
			continue
		}
		entries = append(entries, MapEntry{Key: name, Value: nil})
	}
	readonly := semtypes.IsSubtype(tc, inherentType, semtypes.VAL_READONLY)
	return NewMap(inherentType, atomic, readonly, entries)
}

func convertList(tc semtypes.Context, source *List, inherentType semtypes.SemType,
	unionErrors *[]string,
) BalValue {
	atomic := semtypes.ToListAtomicType(tc, inherentType)
	items := make([]BalValue, source.Len())
	for i := 0; i < source.Len(); i++ {
		memberTy := atomic.MemberAtInnerVal(i)
		narrowedMemberTy, _ := getConvertibleType(tc, source.Get(i), memberTy, unionErrors, true)
		items[i] = convert(tc, source.Get(i), narrowedMemberTy, memberTy, unionErrors)
	}
	restFiller, _ := FillerFactoryFor(tc, atomic.Rest())
	readonly := semtypes.IsSubtype(tc, inherentType, semtypes.VAL_READONLY)
	return NewList(inherentType, atomic, readonly, restFiller, len(items), items)
}

func cloneValue(tc semtypes.Context, value BalValue, targetType semtypes.SemType) BalValue {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case *List:
		listType := targetType
		lat := semtypes.ToListAtomicType(tc, targetType)
		if lat == nil {
			lat = semtypes.ToListAtomicType(tc, v.Type)
			listType = v.Type
		}
		items := make([]BalValue, v.Len())
		for i := 0; i < v.Len(); i++ {
			items[i] = cloneValue(tc, v.Get(i), lat.MemberAtInnerVal(i))
		}
		restFiller, _ := FillerFactoryFor(tc, lat.Rest())
		readonly := semtypes.IsSubtype(tc, listType, semtypes.VAL_READONLY)
		return NewList(listType, lat, readonly, restFiller, v.Len(), items)
	case *Map:
		atomic := semtypes.ToMappingAtomicType(tc, targetType)
		mappingTarget := targetType
		if atomic == nil {
			atomic = semtypes.ToMappingAtomicType(tc, v.Type)
			mappingTarget = v.Type
		}
		entries := make([]MapEntry, 0, v.Len())
		for _, key := range v.Keys() {
			val, _ := v.Get(key)
			fieldType := mappingFieldType(tc, mappingTarget, atomic, key)
			entries = append(entries, MapEntry{Key: key, Value: cloneValue(tc, val, fieldType)})
		}
		readonly := semtypes.IsSubtype(tc, mappingTarget, semtypes.VAL_READONLY)
		return NewMap(mappingTarget, atomic, readonly, entries)
	default:
		return value
	}
}

func isNumericConvertible(tc semtypes.Context, value BalValue, target semtypes.SemType) bool {
	switch value.(type) {
	case int64, float64, *decimal.Decimal:
	default:
		return false
	}
	switch {
	case semtypes.IsSubtypeSimple(target, semtypes.INT),
		semtypes.IsSubtypeSimple(target, semtypes.FLOAT),
		semtypes.IsSubtypeSimple(target, semtypes.DECIMAL),
		semtypes.IsSubtype(tc, target, semtypes.BYTE):
		converted, err := convertNumeric(tc, value, target)
		return err == nil && semtypes.IsSubtype(tc, SemTypeForValue(converted), target)
	default:
		return false
	}
}

func convertNumeric(tc semtypes.Context, value BalValue, target semtypes.SemType) (BalValue, error) {
	switch {
	case semtypes.IsSubtype(tc, target, semtypes.BYTE):
		v, err := toInt(value)
		if err != nil {
			return nil, err
		}
		i := v.(int64)
		if i >= 0 && i <= 255 {
			return i, nil
		}
		return nil, incompatibleConversion(tc, value, target)
	case semtypes.IsSubtypeSimple(target, semtypes.INT):
		return toInt(value)
	case semtypes.IsSubtypeSimple(target, semtypes.FLOAT):
		return toFloat(value)
	default: // DECIMAL
		return toDecimal(value)
	}
}

func toInt(value BalValue) (BalValue, error) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, newConversionFailure("cannot convert non-finite float to int")
		}
		rounded := math.RoundToEven(v)
		if rounded < float64(math.MinInt64) || rounded >= float64(math.MaxInt64) {
			return nil, newConversionFailure("cannot convert out-of-range float to int")
		}
		return int64(rounded), nil
	case *decimal.Decimal:
		n, ok, _ := v.Int64()
		if !ok {
			return nil, newConversionFailure("cannot convert decimal to int64: value out of range")
		}
		return n, nil
	default: // int64
		return value.(int64), nil
	}
}

func toFloat(value BalValue) (BalValue, error) {
	switch v := value.(type) {
	case *decimal.Decimal:
		return v.Float64(), nil
	default: // int64
		return float64(value.(int64)), nil
	}
}

func toDecimal(value BalValue) (BalValue, error) {
	switch v := value.(type) {
	case float64:
		d, err := decimal.FromFloat64(v)
		if err != nil {
			return nil, newConversionFailure(err.Error())
		}
		return d, nil
	default: // int64
		return decimal.FromInt64(value.(int64)), nil
	}
}
