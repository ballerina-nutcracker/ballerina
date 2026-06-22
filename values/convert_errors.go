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
	"strings"

	"ballerina-lang-go/semtypes"
)

const conversionErrorMessage = "{ballerina/lang.value}ConversionError"

type conversionFailure struct {
	detailMessage string
}

func wrapConversionError(err error) *Error {
	detail := err.(*conversionFailure).detailMessage
	detailMap := NewMap(semtypes.MAPPING, &semtypes.MAPPING_ATOMIC_INNER, true, []MapEntry{
		{Key: "message", Value: detail},
	})
	return NewError(semtypes.ERROR, conversionErrorMessage, nil, "", detailMap)
}

func incompatibleConversion(tc semtypes.Context, value BalValue, targetType semtypes.SemType) *conversionFailure {
	sourceTy := SemTypeForValue(value)
	return newConversionFailure(fmt.Sprintf("'%s' value cannot be converted to '%s'",
		semtypes.ToString(tc, sourceTy), semtypes.ToString(tc, targetType)))
}

func cannotConvertNil(tc semtypes.Context, targetType semtypes.SemType) *conversionFailure {
	return newConversionFailure(fmt.Sprintf("'()' value cannot be converted to '%s'", semtypes.ToString(tc, targetType)))
}

func (e *conversionFailure) Error() string {
	return e.detailMessage
}

func newConversionFailure(message string) *conversionFailure {
	return &conversionFailure{detailMessage: message}
}

func unionErrorMessage(errors []string) string {
	var b strings.Builder
	tabs := 0
	for _, err := range errors {
		switch err {
		case "{":
			b.WriteString("\n\t\t")
			b.WriteString(strings.Repeat("  ", tabs))
			b.WriteByte('{')
			tabs++
		case "}":
			tabs--
			b.WriteString("\n\t\t")
			b.WriteString(strings.Repeat("  ", tabs))
			b.WriteByte('}')
		case "or":
			b.WriteString("\n\t\t")
			b.WriteString(strings.Repeat("  ", tabs))
			b.WriteString("or")
		default:
			b.WriteString("\n\t\t")
			b.WriteString(strings.Repeat("  ", tabs))
			b.WriteString(err)
		}
	}
	return b.String()
}
