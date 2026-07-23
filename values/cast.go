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

	"ballerina-lang-go/semtypes"
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

// ConvertNumericValue performs the numeric conversion behind the `<Type>` cast
// operator, sharing the same NumericConvertTo* rules used by cloneWithType and
// fromJsonWithType. Any conversion failure is reported as a bad-type-cast error.
func ConvertNumericValue(value BalValue, targetType semtypes.SemType) (BalValue, error) {
	switch {
	case semtypes.IsSubtypeSimple(targetType, semtypes.INT):
		n, err := NumericConvertToInt(value)
		if err != nil {
			return nil, fmt.Errorf("bad type cast: %w", err)
		}
		return n, nil
	case semtypes.IsSubtypeSimple(targetType, semtypes.FLOAT):
		f, err := NumericConvertToFloat(value)
		if err != nil {
			return nil, fmt.Errorf("bad type cast: %w", err)
		}
		return f, nil
	case semtypes.IsSubtypeSimple(targetType, semtypes.DECIMAL):
		d, err := NumericConvertToDecimal(value)
		if err != nil {
			return nil, fmt.Errorf("bad type cast: %w", err)
		}
		return d, nil
	default:
		return nil, ErrBadTypeCast
	}
}
