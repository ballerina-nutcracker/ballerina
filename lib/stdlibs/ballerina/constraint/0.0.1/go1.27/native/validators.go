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
	"time"
	"unicode/utf16"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	minValue          = "minValue"
	maxValue          = "maxValue"
	minValueExclusive = "minValueExclusive"
	maxValueExclusive = "maxValueExclusive"
	maxDigits         = "maxDigits"
	maxIntegerDigits  = "maxIntegerDigits"
	maxFractionDigits = "maxFractionDigits"
	lengthKey         = "length"
	minLength         = "minLength"
	maxLength         = "maxLength"
	dateOption        = "option"
	messageKey        = "message"
	valueKey          = "value"

	validDate = "validDate"

	// Bounds of java.time.LocalDate.
	maxYear = 999999999
)

// errValueMismatch is raised when an annotation is applied to a value of a kind it cannot constrain.
var errValueMismatch = fmt.Errorf("annotation does not apply to the value")

// invalidConstraintError is raised when a constraint is configured with an unusable value.
type invalidConstraintError struct{ message string }

func (e *invalidConstraintError) Error() string { return e.message }

type failure struct {
	path       string
	constraint string
	message    string
	hasMessage bool
}

func (f failure) pathWithConstraint() string { return f.path + ":" + f.constraint }

// constraintEntry is one constraint of an annotation, with the optional custom message unwrapped.
type constraintEntry struct {
	name       string
	value      values.BalValue
	message    string
	hasMessage bool
}

func entriesOf(constraints *values.Map) []constraintEntry {
	keys := constraints.Keys()
	entries := make([]constraintEntry, 0, len(keys))
	for _, key := range keys {
		raw, _ := constraints.Get(key)
		entry := constraintEntry{name: key, value: raw}
		if wrapped, ok := raw.(*values.Map); ok {
			entry.value, _ = wrapped.Get(valueKey)
			if message, ok := wrapped.Get(messageKey); ok {
				entry.message, entry.hasMessage = message.(string)
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

type validation struct {
	failures []failure
	// now supplies today's date for the Date annotation.
	now func() time.Time
}

func (v *validation) fail(path string, entry constraintEntry, constraint string) {
	v.failures = append(v.failures, failure{path, constraint, entry.message, entry.hasMessage})
}

func (v *validation) apply(tag string, constraints *values.Map, value values.BalValue, path string) error {
	switch tag {
	case "Int":
		return v.validateInt(constraints, value, path)
	case "Float":
		return v.validateFloat(constraints, value, path)
	case "Number":
		return v.validateNumber(constraints, value, path)
	case "String":
		return v.validateString(constraints, value, path)
	case "Array":
		return v.validateArray(constraints, value, path)
	case "Date":
		return v.validateDate(constraints, value, path)
	}
	return nil
}

func positiveConstraint(entry constraintEntry, path, kind string) (int64, error) {
	limit, _ := entry.value.(int64)
	if limit <= 0 {
		return 0, &invalidConstraintError{
			"invalid value found for " + path + ":" + entry.name + " constraint. " + kind + " constraints should be positive"}
	}
	return limit, nil
}

func (v *validation) validateInt(constraints *values.Map, value values.BalValue, path string) error {
	number, err := toLong(value)
	if err != nil {
		return err
	}
	for _, entry := range entriesOf(constraints) {
		limit, _ := entry.value.(int64)
		switch entry.name {
		case maxDigits:
			if _, err := positiveConstraint(entry, path, "Digits"); err != nil {
				return err
			}
			digits := strings.TrimPrefix(strconv.FormatInt(number, 10), "-")
			if int64(len(digits)) > limit {
				v.fail(path, entry, maxDigits)
			}
		case maxIntegerDigits, maxFractionDigits:
			if _, err := positiveConstraint(entry, path, "Digits"); err != nil {
				return err
			}
		default:
			v.checkBounds(path, entry, number >= limit, number <= limit, number > limit, number < limit)
		}
	}
	return nil
}

func (v *validation) validateFloat(constraints *values.Map, value values.BalValue, path string) error {
	return v.validateFloating(constraints, value, path, func(entry constraintEntry) float64 {
		limit, _ := entry.value.(float64)
		return limit
	})
}

func (v *validation) validateNumber(constraints *values.Map, value values.BalValue, path string) error {
	return v.validateFloating(constraints, value, path, func(entry constraintEntry) float64 {
		limit, _ := entry.value.(*decimal.Decimal)
		if limit == nil {
			return 0
		}
		return limit.Float64()
	})
}

// validateFloating compares through float64 and counts digits on the double rendering, as jBallerina does for
// both the float and the number annotations.
func (v *validation) validateFloating(constraints *values.Map, value values.BalValue, path string,
	limitOf func(constraintEntry) float64,
) error {
	number, err := toDouble(value)
	if err != nil {
		return err
	}
	for _, entry := range entriesOf(constraints) {
		switch entry.name {
		case maxDigits, maxIntegerDigits, maxFractionDigits:
			limit, err := positiveConstraint(entry, path, "Digits")
			if err != nil {
				return err
			}
			if entry.name == maxDigits {
				continue
			}
			parts := digitPartsOf(number)
			digits := parts.integerDigits
			if entry.name == maxFractionDigits {
				digits = parts.fractionDigits
			}
			if int64(digits) > limit {
				v.fail(path, entry, entry.name)
			}
		default:
			limit := limitOf(entry)
			v.checkBounds(path, entry, number >= limit, number <= limit, number > limit, number < limit)
		}
	}
	return nil
}

func (v *validation) checkBounds(path string, entry constraintEntry, atLeast, atMost, above, below bool) {
	switch entry.name {
	case minValue:
		if !atLeast {
			v.fail(path, entry, minValue)
		}
	case maxValue:
		if !atMost {
			v.fail(path, entry, maxValue)
		}
	case minValueExclusive:
		if !above {
			v.fail(path, entry, minValueExclusive)
		}
	case maxValueExclusive:
		if !below {
			v.fail(path, entry, maxValueExclusive)
		}
	}
}

func (v *validation) validateString(constraints *values.Map, value values.BalValue, path string) error {
	text, ok := value.(string)
	if !ok {
		return errValueMismatch
	}
	return v.validateLength(constraints, int64(utf16Length(text)), path)
}

func (v *validation) validateArray(constraints *values.Map, value values.BalValue, path string) error {
	list, ok := value.(*values.List)
	if !ok {
		return errValueMismatch
	}
	return v.validateLength(constraints, int64(list.Len()), path)
}

func (v *validation) validateLength(constraints *values.Map, length int64, path string) error {
	for _, entry := range entriesOf(constraints) {
		switch entry.name {
		case lengthKey, minLength, maxLength:
		default:
			continue
		}
		limit, err := positiveConstraint(entry, path, "Length")
		if err != nil {
			return err
		}
		satisfied := map[string]bool{lengthKey: length == limit, minLength: length >= limit, maxLength: length <= limit}
		if !satisfied[entry.name] {
			v.fail(path, entry, entry.name)
		}
	}
	return nil
}

// utf16Length counts UTF-16 code units because jBallerina measures string length with java.lang.String.length.
func utf16Length(text string) int {
	length := 0
	for _, r := range text {
		length += len(utf16.Encode([]rune{r}))
	}
	return length
}

type date struct{ year, month, day int64 }

func (d date) compare(other date) int {
	for _, pair := range [][2]int64{{d.year, other.year}, {d.month, other.month}, {d.day, other.day}} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (v *validation) validateDate(constraints *values.Map, value values.BalValue, path string) error {
	record, ok := value.(*values.Map)
	if !ok {
		return &invalidConstraintError{"Invalid field type found for constraint:Date"}
	}
	var message constraintEntry
	if text, ok := constraints.Get(messageKey); ok {
		message.message, message.hasMessage = text.(string)
	}
	day, valid, err := v.validDate(record, path, message)
	if err != nil || !valid {
		return err
	}
	today := v.now()
	now := date{int64(today.Year()), int64(today.Month()), int64(today.Day())}
	for _, entry := range entriesOf(constraints) {
		if entry.name != dateOption {
			continue
		}
		order := day.compare(now)
		switch option, _ := entry.value.(string); option {
		case "PAST":
			if order >= 0 {
				v.fail(path, entry, "pastDate")
			}
		case "PAST_OR_PRESENT":
			if order > 0 {
				v.fail(path, entry, "pastOrPresentDate")
			}
		case "FUTURE":
			if order <= 0 {
				v.fail(path, entry, "futureDate")
			}
		case "FUTURE_OR_PRESENT":
			if order < 0 {
				v.fail(path, entry, "futureOrPresentDate")
			}
		}
	}
	return nil
}

// validDate checks the fields against the Gregorian calendar, reporting every out-of-range field.
func (v *validation) validDate(record *values.Map, path string, message constraintEntry) (date, bool, error) {
	var fields [3]int64
	for i, name := range []string{"year", "month", "day"} {
		raw, _ := record.Get(name)
		number, ok := raw.(int64)
		if !ok {
			return date{}, false, errValueMismatch
		}
		fields[i] = number
	}
	valid := true
	for i, bounds := range [][2]int64{{-maxYear, maxYear}, {1, 12}, {1, 31}} {
		if fields[i] < bounds[0] || fields[i] > bounds[1] {
			v.fail(path+"."+[]string{"year", "month", "day"}[i], message, validDate)
			valid = false
		}
	}
	if !valid {
		return date{}, false, nil
	}
	d := date{fields[0], fields[1], fields[2]}
	if d.day > daysInMonth(d.year, d.month) {
		v.fail(path+".day", message, validDate)
		return date{}, false, nil
	}
	return d, true, nil
}

func daysInMonth(year, month int64) int64 {
	switch month {
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

func toLong(value values.BalValue) (int64, error) {
	switch number := value.(type) {
	case int64:
		return number, nil
	case float64:
		return saturatingLong(number), nil
	case *decimal.Decimal:
		return saturatingLong(number.Float64()), nil
	}
	return 0, errValueMismatch
}

// saturatingLong follows Java's double to long conversion.
func saturatingLong(number float64) int64 {
	switch {
	case math.IsNaN(number):
		return 0
	case number >= math.MaxInt64:
		return math.MaxInt64
	case number <= math.MinInt64:
		return math.MinInt64
	}
	return int64(number)
}

func toDouble(value values.BalValue) (float64, error) {
	switch number := value.(type) {
	case int64:
		return float64(number), nil
	case float64:
		return number, nil
	case *decimal.Decimal:
		return number.Float64(), nil
	}
	return 0, errValueMismatch
}
