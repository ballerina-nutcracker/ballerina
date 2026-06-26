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
	"fmt"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/semtypes"
)

// CloneWithType implements the cloneWithType abstract operation defined in the Ballerina spec
// (https://ballerina.io/spec/lang/master/#section_16.6).
//
// It constructs a value of targetType by deep-cloning value, applying the following conversions:
//   - the inherent type of any structural value comes from targetType
//   - numeric values may be converted between int, float, and decimal via NumericConvert
//   - missing required fields (with or without defaults) cause a ConversionError; default injection
//     is not yet implemented (tracked as a separate work item)
//
// Cyclic values return a ConversionError (matching jballerina behaviour).
//
// On failure it returns a ConversionError wrapped as *Error.
func CloneWithType(tc semtypes.Context, value BalValue, targetType semtypes.SemType) (BalValue, *Error) {
	var unionErrors []string
	result, err := tryConvert(tc, value, targetType, &unionErrors, true, nil)
	if err != nil {
		return nil, wrapConversionError(err)
	}
	return result, nil
}

func tryConvert(tc semtypes.Context, value BalValue, target semtypes.SemType,
	unionErrors *[]string, allowNumeric bool, visiting map[BalValue]struct{}) (BalValue, *conversionFailure) {
	if value == nil {
		if isNilable(target) {
			return nil, nil
		}
		return nil, cannotConvertNil(tc, target)
	}

	if members := unionMemberTypes(tc, target); len(members) > 1 {
		return tryConvertUnion(tc, value, target, members, unionErrors, allowNumeric, visiting)
	}

	switch v := value.(type) {
	case *Map:
		if semtypes.IsSubtypeSimple(target, semtypes.MAPPING) {
			return tryConvertMapping(tc, v, target, unionErrors, allowNumeric, visiting)
		}
	case *List:
		if semtypes.IsSubtypeSimple(target, semtypes.LIST) {
			return tryConvertList(tc, v, target, unionErrors, allowNumeric, visiting)
		}
	default:
		valueTy := SemTypeForValue(v)
		if semtypes.IsSubtype(tc, valueTy, target) {
			return v, nil
		}
		if allowNumeric {
			switch v.(type) {
			case int64, float64, *decimal.Decimal:
				converted, numErr := convertNumeric(tc, v, target)
				if numErr != nil {
					return nil, numErr
				}
				if semtypes.IsSubtype(tc, SemTypeForValue(converted), target) {
					return converted, nil
				}
			}
		}
	}
	return nil, incompatibleConversion(tc, value, target)
}

func tryConvertUnion(tc semtypes.Context, value BalValue, target semtypes.SemType,
	members []semtypes.SemType, unionErrors *[]string, allowNumeric bool, visiting map[BalValue]struct{},
) (BalValue, *conversionFailure) {
	if isStructuredValue(value) {
		initial := len(*unionErrors)
		*unionErrors = append(*unionErrors, "{")
		for i, member := range members {
			if i > 0 {
				*unionErrors = append(*unionErrors, "or")
			}
			before := len(*unionErrors)
			result, err := tryConvert(tc, value, member, unionErrors, allowNumeric, visiting)
			if err == nil {
				*unionErrors = (*unionErrors)[:initial]
				return result, nil
			}
			*unionErrors = (*unionErrors)[:before]
			*unionErrors = append(*unionErrors, err.Error())
		}
		*unionErrors = append(*unionErrors, "}")
		return nil, newConversionFailure(unionErrorMessage((*unionErrors)[initial:]))
	}

	// For simple values prefer exact type match before allowing numeric conversion.
	for _, member := range members {
		if semtypes.IsSubtype(tc, SemTypeForValue(value), member) {
			return tryConvert(tc, value, member, unionErrors, false, visiting)
		}
	}
	for _, member := range members {
		if result, err := tryConvert(tc, value, member, unionErrors, allowNumeric, visiting); err == nil {
			return result, nil
		}
	}
	return nil, incompatibleConversion(tc, value, target)
}

func tryConvertMapping(tc semtypes.Context, source *Map, target semtypes.SemType,
	unionErrors *[]string, allowNumeric bool, visiting map[BalValue]struct{},
) (BalValue, *conversionFailure) {
	var cycleErr *conversionFailure
	visiting, cycleErr = enterCycleCheck(tc, source.Type, source, visiting)
	if cycleErr != nil {
		return nil, cycleErr
	}
	defer delete(visiting, source)

	atomic := semtypes.ToMappingAtomicType(tc, target)
	if atomic == nil {
		return nil, incompatibleConversion(tc, source, target)
	}

	closed := isClosedRecord(atomic)

	var declared map[string]struct{}
	if closed {
		declared = make(map[string]struct{}, len(atomic.Names))
		for _, name := range atomic.Names {
			declared[name] = struct{}{}
		}
	}

	entries := make([]MapEntry, 0, source.Len())
	seen := make(map[string]struct{}, source.Len())

	for _, key := range source.Keys() {
		seen[key] = struct{}{}
		if closed {
			if _, ok := declared[key]; !ok {
				return nil, incompatibleConversion(tc, source, target)
			}
		}
		fieldTy := mappingFieldType(tc, target, atomic, key)
		val, _ := source.Get(key)
		converted, err := tryConvert(tc, val, fieldTy, unionErrors, allowNumeric, visiting)
		if err != nil {
			return nil, err
		}
		entries = append(entries, MapEntry{Key: key, Value: converted})
	}

	for _, name := range atomic.Names {
		if _, ok := seen[name]; ok {
			continue
		}
		if fieldMayOmitKey(tc, target, name) {
			continue
		}
		// Required field (nilable or not) absent in source — always an error.
		// A nil value must be explicitly present in the source; it is not injected.
		return nil, incompatibleConversion(tc, source, target)
	}

	readonly := semtypes.IsSubtype(tc, target, semtypes.VAL_READONLY)
	return NewMap(target, atomic, readonly, entries), nil
}

func tryConvertList(tc semtypes.Context, source *List, target semtypes.SemType,
	unionErrors *[]string, allowNumeric bool, visiting map[BalValue]struct{},
) (BalValue, *conversionFailure) {
	var cycleErr *conversionFailure
	visiting, cycleErr = enterCycleCheck(tc, source.Type, source, visiting)
	if cycleErr != nil {
		return nil, cycleErr
	}
	defer delete(visiting, source)

	atomic := semtypes.ToListAtomicType(tc, target)
	if atomic == nil {
		return nil, incompatibleConversion(tc, source, target)
	}

	fixedLen := atomic.Members.FixedLength
	if semtypes.IsNever(atomic.Rest()) {
		if source.Len() != fixedLen {
			return nil, incompatibleConversion(tc, source, target)
		}
	} else if source.Len() < fixedLen {
		return nil, incompatibleConversion(tc, source, target)
	}

	items := make([]BalValue, source.Len())
	for i := 0; i < source.Len(); i++ {
		memberTy := atomic.MemberAtInnerVal(i)
		converted, err := tryConvert(tc, source.Get(i), memberTy, unionErrors, allowNumeric, visiting)
		if err != nil {
			return nil, err
		}
		items[i] = converted
	}

	restFiller, _ := FillerFactoryFor(tc, atomic.Rest())
	readonly := semtypes.IsSubtype(tc, target, semtypes.VAL_READONLY)
	return NewList(target, atomic, readonly, restFiller, len(items), items), nil
}

// enterCycleCheck lazily initialises visiting and checks whether source is already being
// converted in the current recursion stack. The caller must defer delete(visiting, source)
// on success so DAG-shared nodes are not falsely reported as cycles on the second reference.
func enterCycleCheck(tc semtypes.Context, sourceType semtypes.SemType, source BalValue, visiting map[BalValue]struct{}) (map[BalValue]struct{}, *conversionFailure) {
	if visiting == nil {
		visiting = make(map[BalValue]struct{})
	}
	if _, cycle := visiting[source]; cycle {
		return visiting, newConversionFailure(fmt.Sprintf("'%s' value has cyclic reference", semtypes.ToString(tc, sourceType)))
	}
	visiting[source] = struct{}{}
	return visiting, nil
}

func convertNumeric(tc semtypes.Context, value BalValue, target semtypes.SemType) (BalValue, *conversionFailure) {
	switch {
	case semtypes.IsSubtype(tc, target, semtypes.BYTE):
		n, err := NumericConvertToInt(value)
		if err != nil {
			return nil, newConversionFailure(err.Error())
		}
		if n >= 0 && n <= 255 {
			return n, nil
		}
		return nil, incompatibleConversion(tc, value, target)
	case semtypes.IsSubtypeSimple(target, semtypes.INT):
		n, err := NumericConvertToInt(value)
		if err != nil {
			return nil, newConversionFailure(err.Error())
		}
		return n, nil
	case semtypes.IsSubtypeSimple(target, semtypes.FLOAT):
		f, err := NumericConvertToFloat(value)
		if err != nil {
			return nil, newConversionFailure(err.Error())
		}
		return f, nil
	default: // DECIMAL
		d, err := NumericConvertToDecimal(value)
		if err != nil {
			return nil, newConversionFailure(err.Error())
		}
		return d, nil
	}
}
