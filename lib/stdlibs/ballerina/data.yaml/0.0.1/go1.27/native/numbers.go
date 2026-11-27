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

// Number parsing and printing that follows the Java methods jBallerina relies on: Long.parseLong,
// Double.parseDouble, new BigDecimal(String) and Double.toString.

package native

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func nan() float64 {
	return math.NaN()
}

func posInf() float64 {
	return math.Inf(1)
}

func negInf() float64 {
	return math.Inf(-1)
}

func isRangeError(err error) bool {
	numErr, ok := err.(*strconv.NumError)
	return ok && numErr.Err == strconv.ErrRange
}

// parseJavaLong accepts an optional sign followed by decimal digits within the int range.
func parseJavaLong(text string) (int64, bool) {
	digits := text
	if digits != "" && (digits[0] == '+' || digits[0] == '-') {
		digits = digits[1:]
	}
	if !isDigits(digits) {
		return 0, false
	}
	n, err := strconv.ParseInt(text, 10, 64)
	return n, err == nil
}

// parseJavaDouble accepts what Double.parseDouble accepts, except a trailing float or double suffix:
// surrounding whitespace, a sign, NaN, Infinity, decimal and hexadecimal floating-point literals.
func parseJavaDouble(text string) (float64, bool) {
	if text == "" || strings.ContainsAny(text[len(text)-1:], "fFdD") {
		return 0, false
	}
	trimmed := strings.TrimFunc(text, func(r rune) bool { return r <= ' ' })
	unsigned := strings.TrimLeft(trimmed, "+-")
	if len(trimmed)-len(unsigned) > 1 {
		return 0, false
	}
	negative := strings.HasPrefix(trimmed, "-")
	switch unsigned {
	case "NaN":
		return math.NaN(), true
	case "Infinity":
		if negative {
			return math.Inf(-1), true
		}
		return math.Inf(1), true
	}
	if !isJavaDecimalFloat(unsigned) && !isJavaHexFloat(unsigned) {
		return 0, false
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil && !isRangeError(err) {
		return 0, false
	}
	return f, true
}

func isJavaDecimalFloat(s string) bool {
	mantissa, exponent, hasExponent := strings.Cut(strings.Replace(s, "E", "e", 1), "e")
	whole, fraction, _ := strings.Cut(mantissa, ".")
	if whole == "" && fraction == "" {
		return false
	}
	if (whole != "" && !isDigits(whole)) || (fraction != "" && !isDigits(fraction)) {
		return false
	}
	if hasExponent {
		exponent = strings.TrimPrefix(strings.TrimPrefix(exponent, "+"), "-")
		return isDigits(exponent)
	}
	return true
}

func isJavaHexFloat(s string) bool {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	mantissa, exponent, hasExponent := strings.Cut(strings.ToLower(s[2:]), "p")
	whole, fraction, _ := strings.Cut(mantissa, ".")
	if !hasExponent || (whole == "" && fraction == "") {
		return false
	}
	if (whole != "" && !isHexDigits(whole)) || (fraction != "" && !isHexDigits(fraction)) {
		return false
	}
	return isDigits(strings.TrimPrefix(strings.TrimPrefix(exponent, "+"), "-"))
}

// parseJavaBigDecimal accepts what new BigDecimal(String) accepts and rounds it to 34 significant
// digits, keeping the scale of the text.
func parseJavaBigDecimal(text string) (*decimal.Decimal, bool) {
	unsigned := text
	if unsigned != "" && (unsigned[0] == '+' || unsigned[0] == '-') {
		unsigned = unsigned[1:]
	}
	if !isJavaDecimalFloat(unsigned) {
		return nil, false
	}
	d, err := decimal.FromLiteral(text)
	return d, err == nil
}

// javaDoubleString renders f as Java's Double.toString does.
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

// javaValueOf renders a value the way String.valueOf renders the jBallerina runtime value.
func javaValueOf(v values.BalValue) string {
	switch v := v.(type) {
	case nil:
		return "null"
	case string:
		return v
	case float64:
		return javaDoubleString(v)
	}
	return values.String(v, make(map[uintptr]bool))
}

// formatJavaInt renders an integer the way java.text.MessageFormat does in the en-US locale.
func formatJavaInt(n int64) string {
	digits := strconv.FormatInt(n, 10)
	negative := strings.HasPrefix(digits, "-")
	digits = strings.TrimPrefix(digits, "-")
	var sb strings.Builder
	if negative {
		sb.WriteByte('-')
	}
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
