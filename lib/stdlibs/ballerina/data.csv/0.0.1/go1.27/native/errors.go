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
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// csvError is a csv:Error. A tokenError is jBallerina's CsvParserException: the parser reports it with
// the line and column it was raised at, and failSafe always tolerates it.
type csvError struct {
	message    string
	tokenError bool
}

func (e *csvError) Error() string {
	return e.message
}

// conversionError is a runtime error that jBallerina returns as it is rather than as a csv:Error.
type conversionError struct {
	value *values.Error
}

// Error is the message jBallerina logs for the error, which is its type name.
func (e *conversionError) Error() string {
	if e.value.TypeName != "" {
		return e.value.TypeName
	}
	return e.value.Message
}

// nilRowError is the error jBallerina raises when input ends inside a quoted or escaped header: the
// last header value becomes a row that was never created, and appending it fails.
func nilRowError(rowTypeName string) *conversionError {
	message := fmt.Sprintf("incompatible types: expected '%s', found '()'", rowTypeName)
	detail := values.NewMap(semtypes.Mapping, &semtypes.MappingAtomicInner, true, []values.MapEntry{{Key: "message", Value: message}})
	return &conversionError{value: values.NewError(semtypes.Error, "{ballerina/lang.array}InherentTypeViolation", nil, "", detail)}
}

func newCsvError(format string, args ...any) *csvError {
	return &csvError{message: fmt.Sprintf(format, args...)}
}

func newTokenError(message string) *csvError {
	return &csvError{message: message, tokenError: true}
}

func expectedTokenError(what string) *csvError {
	return newTokenError("expected '" + what + "'")
}

func invalidTypeError(typeName string) *csvError {
	return newCsvError("invalid expected type '%s', expected a subtype of (map<anydata>|anydata[])[]", typeName)
}

func missingFieldError(field string) *csvError {
	return newCsvError("no matching header value is found for the required field '%s'", field)
}

func invalidCastError(value, typeName string) *csvError {
	return newCsvError("value '%s' cannot be cast into '%s'", value, typeName)
}

func invalidExpectedTypeError(typeName string) *csvError {
	return newCsvError("expected a array type, found '%s' ", typeName)
}

func invalidTokenError(message string, line, column int) *csvError {
	return newCsvError("error while reading the csv data '%s' in line: '%s', column: '%s'",
		message, formatJavaInt(int64(line)), formatJavaInt(int64(column)))
}

func arraySizeError(size int) *csvError {
	return newCsvError("invalid array size for expected array type, cannot be greater than '%s'", formatJavaInt(int64(size)))
}

func tupleSizeError(size int) *csvError {
	return newCsvError("invalid array size for expected tuple type, cannot be greater than '%s'", formatJavaInt(int64(size)))
}

func invalidTypeForFieldError(value, key string) *csvError {
	return newCsvError("no mapping type found for value '%s' in key '%s'", value, key)
}

func invalidTypeForArrayError(value string, index int, typeName string) *csvError {
	return newCsvError("value '%s' in index '%s' is not compatible with array type '%s'",
		value, formatJavaInt(int64(index)), typeName)
}

func onlyBasicTypesError(typeName string) *csvError {
	return newCsvError("expected type cannot contains types other than basic types, '%s' not allowed", typeName)
}

func arrayToMapError(value, typeName string) *csvError {
	return newCsvError("value '%s' cannot be cast into '%s', because fields in '%s' or the provided expected headers "+
		"are not matching with the '%s'", value, typeName, typeName, value)
}

func skipLinesRangeError() *csvError {
	return newCsvError("invalid range for the skipLines field. Start value must be less than or equal to end value.")
}

func skipLinesValueError() *csvError {
	return newCsvError("invalid input for the skipLines field. Both start and end values must be integers.")
}

func invalidHeaderCountError() *csvError {
	return newCsvError("invalid number of headers")
}

func emptyHeaderError() *csvError {
	return newCsvError("provided header row is empty")
}

func noFieldForHeaderError(header string) *csvError {
	return newCsvError("no mapping field in the expected type for header '%s'", header)
}

func sourceCannotConvertError(typeName string) *csvError {
	return newCsvError("source value cannot converted in to the '%s'", typeName)
}

func noCustomHeadersError() *csvError {
	return newCsvError("custom headers should be provided")
}

func varyingHeadersError() *csvError {
	return newCsvError("CSV data rows with varying headers are not yet supported")
}

func duplicateHeaderError(header string) *csvError {
	return newCsvError("duplicate header found: '%s'", header)
}

func fileIOError(detail string) *csvError {
	return newCsvError("file I/O operation failed: '%s'", detail)
}

// isFatal reports the errors failSafe does not tolerate. jBallerina compares each message against the
// message templates with their placeholders removed; only the two templates without a quoted
// placeholder can ever match.
func isFatal(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "invalid csv data format") || strings.Contains(msg, "provided header row is empty")
}

// javaValueOf renders a value the way String.valueOf renders the jBallerina runtime value.
func javaValueOf(v values.BalValue) string {
	switch v := v.(type) {
	case nil:
		return "null"
	case string:
		return v
	case float64:
		return javaDoubleString(v)
	default:
		return values.String(v, make(map[uintptr]bool))
	}
}

// formatMessageArg renders a value the way java.text.MessageFormat renders an argument object: Java
// numbers go through the en-US number format, everything else, decimal included, through toString.
func formatMessageArg(v values.BalValue) string {
	switch v := v.(type) {
	case int64:
		return formatJavaInt(v)
	case float64:
		return formatJavaDouble(v)
	}
	return javaValueOf(v)
}

// formatJavaDouble applies the default en-US NumberFormat to a double: the shortest round-trip digits,
// grouped, with at most three fraction digits rounded half-even on the exact binary value.
func formatJavaDouble(v float64) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "∞"
	case math.IsInf(v, -1):
		return "-∞"
	}
	abs := math.Abs(v)
	intPart, fracPart, _ := strings.Cut(strconv.FormatFloat(abs, 'f', -1, 64), ".")
	if len(fracPart) > 3 {
		roundUp := roundsUpHalfEven(abs, fracPart[2])
		fracPart = fracPart[:3]
		if roundUp {
			intPart, fracPart = incrementDigits(intPart, fracPart)
		}
	}
	fracPart = strings.TrimRight(fracPart, "0")
	result := groupDigits(intPart)
	if fracPart != "" {
		result += "." + fracPart
	}
	if v < 0 && strings.Trim(result, "0.,") != "" {
		return "-" + result
	}
	return result
}

// roundsUpHalfEven reports whether a non-negative double rounds up at the third fraction digit, judged
// on its exact binary value as java.text.DecimalFormat does.
func roundsUpHalfEven(abs float64, lastKept byte) bool {
	_, exactFrac, _ := strings.Cut(strconv.FormatFloat(abs, 'f', 1074, 64), ".")
	dropped := exactFrac[3:]
	switch {
	case dropped[0] != '5':
		return dropped[0] > '5'
	case strings.TrimRight(dropped[1:], "0") != "":
		return true
	}
	return (lastKept-'0')%2 == 1
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

// formatJavaInt renders an integer the way java.text.MessageFormat does in the en-US locale.
func formatJavaInt(n int64) string {
	digits := strconv.FormatInt(n, 10)
	if strings.HasPrefix(digits, "-") {
		return "-" + groupDigits(digits[1:])
	}
	return groupDigits(digits)
}

// groupDigits inserts en-US grouping separators into a string of decimal digits.
func groupDigits(digits string) string {
	var sb strings.Builder
	head := len(digits) % 3
	if head == 0 {
		head = 3
	}
	sb.WriteString(digits[:head])
	for i := head; i < len(digits); i += 3 {
		sb.WriteByte(',')
		sb.WriteString(digits[i : i+3])
	}
	return sb.String()
}
