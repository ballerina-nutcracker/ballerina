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
	"math"

	"ballerina-lang-go/decimal"
)

// NumericConvertToInt converts a numeric BalValue (int64, float64, or *decimal.Decimal)
// to int64 using Ballerina's NumericConvert rules: float/decimal are rounded to nearest even.
func NumericConvertToInt(value BalValue) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("cannot cast non-finite value %v to int", v)
		}
		rounded := math.RoundToEven(v)
		if rounded < float64(math.MinInt64) || rounded >= float64(math.MaxInt64) {
			return 0, fmt.Errorf("cannot cast out-of-range value %v to int", v)
		}
		return int64(rounded), nil
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
		return 0, fmt.Errorf("cannot cast %v to int", value)
	}
}

// NumericConvertToFloat converts a numeric BalValue (int64, float64, or *decimal.Decimal)
// to float64 using Ballerina's NumericConvert rules.
func NumericConvertToFloat(value BalValue) (float64, error) {
	switch v := value.(type) {
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case *decimal.Decimal:
		return v.Float64(), nil
	default:
		return 0, fmt.Errorf("cannot cast %v to float", value)
	}
}

// NumericConvertToDecimal converts a numeric BalValue (int64, float64, or *decimal.Decimal)
// to *decimal.Decimal using Ballerina's NumericConvert rules.
// Non-finite floats (NaN, ±Inf) are rejected.
func NumericConvertToDecimal(value BalValue) (*decimal.Decimal, error) {
	switch v := value.(type) {
	case int64:
		return decimal.FromInt64(v), nil
	case float64:
		d, err := decimal.FromFloat64(v)
		if err != nil {
			return nil, err
		}
		return d, nil
	case *decimal.Decimal:
		return v, nil
	default:
		return nil, fmt.Errorf("cannot cast %v to decimal", value)
	}
}
