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

	"ballerina/semtypes"
)

const conversionErrorTypeName = "{ballerina/lang.value}ConversionError"

// conversionFailure describes a single conversion failure. isCyclic marks one from a cyclic
// reference, which convert short-circuits on instead of collecting alongside other mismatches.
type conversionFailure struct {
	detailMessage string
	isCyclic      bool
}

func wrapConversionError(err *conversionFailure) *Error {
	message := err.Error()
	detailMap := NewMap(semtypes.MAPPING, &semtypes.MAPPING_ATOMIC_INNER, true, []MapEntry{
		{Key: "message", Value: message},
	})
	return NewError(semtypes.ERROR, message, nil, conversionErrorTypeName, detailMap)
}

func (e *conversionFailure) Error() string {
	return e.detailMessage
}

func newConversionFailure(message string) *conversionFailure {
	return &conversionFailure{detailMessage: message}
}

func newCyclicConversionFailure(message string) *conversionFailure {
	return &conversionFailure{detailMessage: message, isCyclic: true}
}

func incompatibleConversion(tc semtypes.Context, value BalValue, targetType semtypes.SemType) *conversionFailure {
	sourceTy := SemTypeForValue(value)
	return newConversionFailure(fmt.Sprintf("'%s' value cannot be converted to '%s'",
		semtypes.ToString(tc, sourceTy), semtypes.ToString(tc, targetType)))
}

// missingRequiredField reports a required field absent from source. Fires regardless of a
// declared default in targetType, since default-value injection isn't implemented yet.
func missingRequiredField(tc semtypes.Context, value BalValue, targetType semtypes.SemType, fieldName string) *conversionFailure {
	sourceTy := SemTypeForValue(value)
	return newConversionFailure(fmt.Sprintf("'%s' value cannot be converted to '%s': field '%s' not present in value",
		semtypes.ToString(tc, sourceTy), semtypes.ToString(tc, targetType), fieldName))
}
