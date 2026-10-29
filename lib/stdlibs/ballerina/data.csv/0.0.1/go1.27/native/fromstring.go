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

// Port of jBallerina's FromString: it turns the text of a CSV value into a value of the expected type.

package native

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// jsonConversionOrder is the order FromString tries the members of json and anydata targets in.
var jsonConversionOrder = []typeInfo{
	{kind: kindNil, ty: semtypes.Nil}, {kind: kindInt, ty: semtypes.Int}, {kind: kindFloat, ty: semtypes.Float},
	{kind: kindDecimal, ty: semtypes.Decimal}, {kind: kindBoolean, ty: semtypes.Boolean},
	{kind: kindString, ty: semtypes.String},
}

// localeSymbols are the java.text.DecimalFormatSymbols that FromString's number parsing depends on.
type localeSymbols struct {
	decimalSeparator rune
	minus            rune
}

// commaDecimalLanguages and unicodeMinusLanguages approximate the CLDR data the JDK uses for number
// formatting; only the decimal separator and the minus sign affect which strings parse as numbers.
var (
	commaDecimalLanguages = map[string]bool{
		"bg": true, "cs": true, "da": true, "de": true, "el": true, "es": true, "fi": true, "fr": true,
		"hr": true, "hu": true, "id": true, "it": true, "lt": true, "lv": true, "nb": true, "nl": true,
		"no": true, "pl": true, "pt": true, "ro": true, "ru": true, "sk": true, "sl": true, "sr": true,
		"sv": true, "tr": true, "uk": true, "vi": true, "et": true,
	}
	unicodeMinusLanguages = map[string]bool{
		"et": true, "fi": true, "lt": true, "nb": true, "no": true, "sl": true, "sv": true,
	}
)

// localeFor resolves a locale string the way CsvUtils.createLocaleFromString does: the part before the
// first underscore is the language, and an unknown language falls back to the root symbols.
func localeFor(locale string) localeSymbols {
	language, _, _ := strings.Cut(locale, "_")
	language = strings.ToLower(language)
	symbols := localeSymbols{decimalSeparator: '.', minus: '-'}
	if commaDecimalLanguages[language] {
		symbols.decimalSeparator = ','
	}
	if unicodeMinusLanguages[language] {
		symbols.minus = '−'
	}
	return symbols
}

// converter holds the options FromString reads: the nil representation and the locale.
type converter struct {
	t        *typer
	nilValue *string
	locale   localeSymbols
}

func (c *converter) isNullValue(value string) bool {
	if c.nilValue == nil {
		return strings.EqualFold(value, "null") || strings.EqualFold(value, "()")
	}
	return value == *c.nilValue
}

// fromString is FromString.fromStringWithType; anydata is converted as json, as CsvCreator does.
func (c *converter) fromString(value string, ti typeInfo) (values.BalValue, bool) {
	switch ti.kind {
	case kindInt:
		return c.parseInt(value)
	case kindIntSubtype:
		n, ok := c.parseInt(value)
		if !ok || !c.t.contains(ti.ty, n) {
			return nil, false
		}
		return n, true
	case kindFloat:
		return c.parseFloat(value)
	case kindDecimal:
		return c.parseDecimal(value)
	case kindChar:
		return value, utf8.RuneCountInString(value) == 1
	case kindString:
		return value, true
	case kindBoolean:
		switch {
		case strings.EqualFold(value, "true"):
			return true, true
		case strings.EqualFold(value, "false"):
			return false, true
		}
		return nil, false
	case kindNil:
		return nil, c.isNullValue(value)
	case kindFinite:
		return c.toSingleton(value, ti)
	case kindUnion:
		return c.toUnion(value, ti.conversionOrder)
	case kindJSON, kindAnydata:
		return c.toUnion(value, jsonConversionOrder)
	}
	return nil, false
}

func (c *converter) toUnion(value string, conversionOrder []typeInfo) (values.BalValue, bool) {
	for _, member := range conversionOrder {
		if converted, ok := c.fromString(value, member); ok {
			return converted, true
		}
	}
	return nil, false
}

// toSingleton matches value against the singletons of a finite type: the text must equal the
// singleton's string form and convert to the singleton's own type.
func (c *converter) toSingleton(value string, ti typeInfo) (values.BalValue, bool) {
	for _, basic := range scalarBasicTypes {
		if semtypes.IsEmpty(c.t.tc, semtypes.Intersect(ti.ty, basic.ty)) {
			continue
		}
		converted, ok := c.fromString(value, typeInfo{kind: basic.kind, ty: basic.ty})
		if ok && c.t.contains(ti.ty, converted) && javaValueOf(converted) == value {
			return converted, true
		}
	}
	return nil, false
}

// parseInt accepts what NumberFormat.parse returns as a Long and the integer pattern matches: optional
// locale minus sign, then ASCII digits.
func (c *converter) parseInt(value string) (int64, bool) {
	digits, negative := c.stripMinus(value)
	if digits == "" || strings.Trim(digits, "0123456789") != "" {
		return 0, false
	}
	if negative && strings.Trim(digits, "0") == "" {
		// NumberFormat returns a negative zero as the Double -0.0, which the int conversion rejects.
		return 0, false
	}
	sign := ""
	if negative {
		sign = "-"
	}
	n, err := strconv.ParseInt(sign+digits, 10, 64)
	return n, err == nil
}

// parseFloat accepts an optional locale minus sign, digits, an optional locale decimal separator with
// fraction digits, and an optional exponent with an upper-case E.
func (c *converter) parseFloat(value string) (float64, bool) {
	digits, negative := c.stripMinus(value)
	mantissa, exponent, hasExponent := strings.Cut(digits, "E")
	whole, fraction, _ := strings.Cut(mantissa, string(c.locale.decimalSeparator))
	if whole == "" || strings.Trim(whole, "0123456789") != "" || strings.Trim(fraction, "0123456789") != "" {
		return 0, false
	}
	literal := whole + "." + fraction
	if hasExponent {
		expDigits := strings.TrimLeft(exponent, "+-")
		if len(exponent)-len(expDigits) > 1 || expDigits == "" || strings.Trim(expDigits, "0123456789") != "" {
			return 0, false
		}
		literal += "e" + exponent
	}
	if negative {
		literal = "-" + literal
	}
	f, err := strconv.ParseFloat(literal, 64)
	if err != nil && !isRangeError(err) {
		return 0, false
	}
	return f, true
}

// parseDecimal parses the value as a float and converts it as BigDecimal.valueOf(double) does, so the
// result carries the digits and scale of Java's Double.toString.
func (c *converter) parseDecimal(value string) (*decimal.Decimal, bool) {
	f, ok := c.parseFloat(value)
	if !ok || math.IsInf(f, 0) {
		return nil, false
	}
	d, err := decimal.FromString(javaDoubleString(f))
	return d, err == nil
}

func (c *converter) stripMinus(value string) (string, bool) {
	r, size := utf8.DecodeRuneInString(value)
	if size > 0 && r == c.locale.minus {
		return value[size:], true
	}
	return value, false
}

func isRangeError(err error) bool {
	numErr, ok := err.(*strconv.NumError)
	return ok && numErr.Err == strconv.ErrRange
}

// javaDoubleString renders f as Java's Double.toString does: the shortest digits that round-trip, plain
// notation with at least one fraction digit for magnitudes in [1e-3, 1e7), computerized scientific
// notation otherwise.
func javaDoubleString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	abs := math.Abs(f)
	sign := ""
	if f < 0 {
		sign = "-"
	}
	if abs >= 1e-3 && abs < 1e7 {
		plain := strconv.FormatFloat(abs, 'f', -1, 64)
		if !strings.Contains(plain, ".") {
			plain += ".0"
		}
		return sign + plain
	}
	mantissa, exponent, _ := strings.Cut(strconv.FormatFloat(abs, 'e', -1, 64), "e")
	if !strings.Contains(mantissa, ".") {
		mantissa += ".0"
	}
	exp, _ := strconv.Atoi(exponent)
	return sign + mantissa + "E" + strconv.Itoa(exp)
}
