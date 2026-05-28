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
	"math"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

func isLikeType(tc semtypes.Context, value values.BalValue, target semtypes.SemType, allowNumeric bool) bool {
	valueTy := values.SemTypeForValue(value)
	if semtypes.IsSubtype(tc, valueTy, target) {
		return true
	}
	if !allowNumeric {
		return false
	}
	return isNumericConvertible(tc, value, target)
}

func isNumericConvertible(tc semtypes.Context, value values.BalValue, target semtypes.SemType) bool {
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
		return err == nil && semtypes.IsSubtype(tc, values.SemTypeForValue(converted), target)
	default:
		return false
	}
}

func convertNumeric(tc semtypes.Context, value values.BalValue, target semtypes.SemType) (values.BalValue, error) {
	switch {
	case semtypes.IsSubtypeSimple(target, semtypes.INT):
		return toInt(value)
	case semtypes.IsSubtypeSimple(target, semtypes.FLOAT):
		return toFloat(value)
	case semtypes.IsSubtypeSimple(target, semtypes.DECIMAL):
		return toDecimal(value)
	case semtypes.IsSubtype(tc, target, semtypes.BYTE):
		v, err := toInt(value)
		if err != nil {
			return nil, err
		}
		if i, ok := v.(int64); ok && i >= 0 && i <= 255 {
			return i, nil
		}
		return nil, incompatibleConversion(tc, value, target)
	default:
		return nil, incompatibleConversion(tc, value, target)
	}
}

func isNilable(tc semtypes.Context, target semtypes.SemType) bool {
	return semtypes.ContainsBasicType(target, semtypes.NIL)
}

func toInt(value values.BalValue) (values.BalValue, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, newConversionFailure("cannot convert non-finite float to int")
		}
		if v < float64(math.MinInt64) || v > float64(math.MaxInt64) {
			return nil, newConversionFailure("cannot convert out-of-range float to int")
		}
		return int64(math.RoundToEven(v)), nil
	case *decimal.Decimal:
		n, ok, err := v.Int64()
		if err != nil {
			return nil, newConversionFailure(err.Error())
		}
		if !ok {
			return nil, newConversionFailure("cannot convert decimal to int64: value out of range")
		}
		return n, nil
	default:
		return nil, newConversionFailure("cannot convert value to int")
	}
}

func toFloat(value values.BalValue) (values.BalValue, error) {
	switch v := value.(type) {
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case *decimal.Decimal:
		return v.Float64(), nil
	default:
		return nil, newConversionFailure("cannot convert value to float")
	}
}

func toDecimal(value values.BalValue) (values.BalValue, error) {
	switch v := value.(type) {
	case int64:
		return decimal.FromInt64(v), nil
	case float64:
		d, err := decimal.FromFloat64(v)
		if err != nil {
			return nil, newConversionFailure(err.Error())
		}
		return d, nil
	case *decimal.Decimal:
		return v, nil
	default:
		return nil, newConversionFailure("cannot convert value to decimal")
	}
}
