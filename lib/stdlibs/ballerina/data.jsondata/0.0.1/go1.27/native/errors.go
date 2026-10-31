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

package native

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/values"
)

// jsonError is an error jBallerina throws: it aborts the whole conversion, except where a union member
// attempt catches it and moves on to the next member.
type jsonError struct {
	message string
}

func (e *jsonError) Error() string {
	return e.message
}

// returnedError is an error jBallerina returns as a value from a conversion helper; the caller decides how to
// report it, usually by wrapping it in a message that names the field or type.
type returnedError struct {
	message string
}

func (e *returnedError) Error() string {
	return e.message
}

func newJSONError(format string, args ...any) *jsonError {
	return &jsonError{message: fmt.Sprintf(format, args...)}
}

func unsupportedTypeError(typeName string) *jsonError {
	return newJSONError("unsupported type '%s'", typeName)
}

func parserExceptionError(message string, line, column int) *jsonError {
	return newJSONError("'%s' at line: '%s' column: '%s'", message, formatJavaInt(int64(line)), formatJavaInt(int64(column)))
}

func incompatibleTypeError(typeName, value string) *jsonError {
	return newJSONError("incompatible expected type '%s' for value '%s'", typeName, value)
}

func arraySizeMismatchError() *jsonError {
	return newJSONError("array size is not compatible with the expected size")
}

func invalidTypeError(typeName, expected string) *jsonError {
	return newJSONError("invalid type '%s' expected '%s'", typeName, expected)
}

func incompatibleValueForFieldError(value, typeName, fieldPath string) *jsonError {
	return newJSONError("incompatible value '%s' for type '%s' in field '%s'", value, typeName, fieldPath)
}

func requiredFieldNotPresentError(fieldName string) *jsonError {
	return newJSONError("required field '%s' not present in JSON", fieldName)
}

func invalidTypeForFieldError(fieldPath string) *jsonError {
	return newJSONError("invalid type for field '%s'", fieldPath)
}

func duplicateFieldError(fieldName string) *jsonError {
	return newJSONError("duplicate field '%s'", fieldName)
}

func undefinedFieldError(fieldName string) *jsonError {
	return newJSONError("undefined field '%s'", fieldName)
}

func cyclicReferenceError() *jsonError {
	return newJSONError("the value has a cyclic reference")
}

func cannotConvertError(value, typeName string) *returnedError {
	return &returnedError{message: fmt.Sprintf("'string' value '%s' cannot be converted to '%s'", value, typeName)}
}

// formatMessageArg renders a value the way java.text.MessageFormat renders an argument object: numbers go
// through the locale number format, everything else through toString.
func formatMessageArg(v values.BalValue) string {
	switch v := v.(type) {
	case int64:
		return formatJavaInt(v)
	case float64:
		return formatJavaDouble(v)
	default:
		return javaValueOf(v)
	}
}

// javaValueOf renders a value the way String.valueOf renders the jBallerina runtime value.
func javaValueOf(v values.BalValue) string {
	if v == nil {
		return "null"
	}
	return values.String(v, make(map[uintptr]bool))
}

func formatJavaInt(n int64) string {
	digits := strconv.FormatUint(absInt(n), 10)
	grouped := groupThousands(digits)
	if n < 0 {
		return "-" + grouped
	}
	return grouped
}

func absInt(n int64) uint64 {
	if n < 0 {
		return uint64(-(n + 1)) + 1
	}
	return uint64(n)
}

// formatJavaDouble mirrors the default en-US NumberFormat: grouped integer digits and at most three
// fraction digits, rounded half-even.
func formatJavaDouble(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "∞"
	case math.IsInf(f, -1):
		return "-∞"
	}
	negative := math.Signbit(f)
	intPart, fracPart := roundHalfEvenDigits(exactDecimal(math.Abs(f)), 3)
	result := groupThousands(intPart)
	if fracPart != "" {
		result += "." + fracPart
	}
	if negative {
		return "-" + result
	}
	return result
}

// exactDecimal is the exact decimal expansion of a finite non-negative double, which DecimalFormat
// rounds from.
func exactDecimal(f float64) string {
	return new(big.Float).SetFloat64(f).Text('f', 1100)
}

// roundHalfEvenDigits rounds a plain non-negative decimal string to at most scale fraction digits, dropping
// trailing fraction zeros.
func roundHalfEvenDigits(plain string, scale int) (string, string) {
	intPart, fracPart, _ := strings.Cut(plain, ".")
	if len(fracPart) > scale {
		kept := fracPart[:scale]
		dropped := fracPart[scale:]
		roundUp := dropped[0] > '5' || (dropped[0] == '5' && strings.TrimRight(dropped[1:], "0") != "")
		if dropped[0] == '5' && !roundUp {
			last := intPart[len(intPart)-1]
			if scale > 0 {
				last = kept[scale-1]
			}
			roundUp = (last-'0')%2 == 1
		}
		fracPart = kept
		if roundUp {
			intPart, fracPart = incrementDigits(intPart, fracPart)
		}
	}
	return intPart, strings.TrimRight(fracPart, "0")
}

func incrementDigits(intPart, fracPart string) (string, string) {
	digits := []byte(intPart + fracPart)
	i := len(digits) - 1
	for ; i >= 0; i-- {
		if digits[i] == '9' {
			digits[i] = '0'
			continue
		}
		digits[i]++
		break
	}
	if i < 0 {
		digits = append([]byte{'1'}, digits...)
	}
	split := len(digits) - len(fracPart)
	return string(digits[:split]), string(digits[split:])
}

func groupThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	var sb strings.Builder
	head := len(digits) % 3
	if head > 0 {
		sb.WriteString(digits[:head])
	}
	for i := head; i < len(digits); i += 3 {
		if sb.Len() > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(digits[i : i+3])
	}
	return sb.String()
}
