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
	"math"
	"regexp"
	"strconv"
	"strings"
)

var scientificNotation = regexp.MustCompile(`^(-?[0-9]+)\.([0-9]+)[Ee]([+-]?[0-9]+)$`)

type digitParts struct {
	integerDigits  int
	fractionDigits int
}

// digitPartsOf counts digits on Java's Double.toString rendering of value so that
// the counts agree with jBallerina, including for values rendered in scientific notation.
func digitPartsOf(value float64) digitParts {
	rendered := javaDoubleString(value)
	if m := scientificNotation.FindStringSubmatch(rendered); m != nil {
		return digitPartsFromScientific(value, m)
	}
	return digitPartsFromPlain(value, rendered)
}

func digitPartsFromPlain(value float64, rendered string) digitParts {
	parts := strings.Split(rendered, ".")
	integerDigits := len(parts[0])
	if value < 0 {
		integerDigits--
	}
	if len(parts) == 1 {
		return digitParts{integerDigits, 0}
	}
	return digitParts{integerDigits, len(parts[1])}
}

func digitPartsFromScientific(value float64, groups []string) digitParts {
	countA := len(groups[1])
	if value < 0 {
		countA--
	}
	countB := len(groups[2])
	if groups[2] == "0" {
		countB = 0
	}
	exponent, _ := strconv.Atoi(groups[3])
	if exponent > 0 {
		shifted := min(countB, exponent)
		return digitParts{countA + shifted, countB - shifted}
	}
	return digitParts{max(1, countA+exponent), countB - exponent}
}

// javaDoubleString reproduces Double.toString: plain notation for magnitudes in
// [1e-3, 1e7) and computerized scientific notation otherwise, with the shortest
// digits that round-trip.
func javaDoubleString(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	case value == 0:
		if math.Signbit(value) {
			return "-0.0"
		}
		return "0.0"
	}
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	mantissa, exponentText, _ := strings.Cut(strconv.FormatFloat(value, 'e', -1, 64), "e")
	digits := strings.Replace(mantissa, ".", "", 1)
	exponent, _ := strconv.Atoi(exponentText)
	if value >= 1e-3 && value < 1e7 {
		return sign + plainDigits(digits, exponent)
	}
	fraction := digits[1:]
	if fraction == "" {
		fraction = "0"
	}
	return sign + digits[:1] + "." + fraction + "E" + strconv.Itoa(exponent)
}

func plainDigits(digits string, exponent int) string {
	if exponent < 0 {
		return "0." + strings.Repeat("0", -exponent-1) + digits
	}
	if len(digits) <= exponent+1 {
		return digits + strings.Repeat("0", exponent+1-len(digits)) + ".0"
	}
	return digits[:exponent+1] + "." + digits[exponent+1:]
}
