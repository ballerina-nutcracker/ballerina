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
	"errors"
	"fmt"
	"math"

	"ballerina/decimal"
	"ballerina/semtypes"
)

var ErrBadTypeCast = errors.New("bad type cast")

func CastValue(typeCtx semtypes.Context, value BalValue, targetType semtypes.SemType) (BalValue, error) {
	if semtypes.IsSubtype(typeCtx, SemTypeForValue(value), targetType) {
		return value, nil
	}

	converted, err := ConvertNumericValue(value, targetType)
	if err != nil {
		return nil, err
	}
	if !semtypes.IsSubtype(typeCtx, SemTypeForValue(converted), targetType) {
		return nil, ErrBadTypeCast
	}
	return converted, nil
}

func ConvertNumericValue(value BalValue, targetType semtypes.SemType) (BalValue, error) {
	switch {
	case semtypes.IsSubtypeSimple(targetType, semtypes.INT):
		return ToInt(value)
	case semtypes.IsSubtypeSimple(targetType, semtypes.FLOAT):
		return ToFloat(value)
	case semtypes.IsSubtypeSimple(targetType, semtypes.DECIMAL):
		return ToDecimal(value)
	default:
		return nil, ErrBadTypeCast
	}
}

func ToInt(value BalValue) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("bad type cast: cannot cast non-finite value %v to int", v)
		}
		if v < float64(math.MinInt64) || v > float64(math.MaxInt64) {
			return 0, fmt.Errorf("bad type cast: cannot cast out-of-range value %v to int", v)
		}
		return int64(math.RoundToEven(v)), nil
	case *decimal.Decimal:
		n, ok, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("cannot convert %v to int: %v", v, err)
		}
		if !ok {
			return 0, fmt.Errorf("cannot convert %v to int64: value out of range", v)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("bad type cast: cannot cast %v to int", value)
	}
}

func ToFloat(value BalValue) (float64, error) {
	switch v := value.(type) {
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case *decimal.Decimal:
		return v.Float64(), nil
	default:
		return 0, fmt.Errorf("bad type cast: cannot cast %v to float", value)
	}
}

func ToDecimal(value BalValue) (*decimal.Decimal, error) {
	switch v := value.(type) {
	case int64:
		return decimal.FromInt64(v), nil
	case float64:
		if math.IsInf(v, 0) {
			return nil, &decimal.Error{Kind: decimal.ErrOverflow}
		}
		if math.IsNaN(v) {
			return nil, &decimal.Error{Kind: decimal.ErrInvalid}
		}
		d, err := decimal.FromFloat64(v)
		if err != nil {
			return nil, err
		}
		return d, nil
	case *decimal.Decimal:
		return v, nil
	default:
		return nil, fmt.Errorf("bad type cast: cannot cast %v to decimal", value)
	}
}
