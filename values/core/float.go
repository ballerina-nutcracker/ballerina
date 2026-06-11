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

package core

import (
	"math"
	"strconv"
	"strings"
)

// FloatDeepEqual is Ballerina `==` for plain float values: +0 and -0 compare
// equal; any two NaNs compare equal.
func FloatDeepEqual(a, b float64) bool {
	if a == b {
		return true
	}
	return math.IsNaN(a) && math.IsNaN(b)
}

// FloatExactEqual is Ballerina `===` for plain float values: +0 and -0 are
// distinct; any two NaNs compare equal (including differing quiet-NaN payloads).
func FloatExactEqual(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return math.Float64bits(a) == math.Float64bits(b)
}

// FormatFloat renders a Ballerina float using `lang.float:toString` semantics.
//
// The format matches jBallerina's `StringUtils.getStringValue(double)`:
//   - NaN, Infinity, -Infinity for the non-finite cases
//   - 0.0 / -0.0 for signed zero
//   - plain decimal notation (one fractional digit min) when |x| in [1e-3, 1e7)
//   - scientific notation otherwise, with a trailing `.0` stripped from a
//     bare integer mantissa (so `1.0E101` is rendered as `1e101`)
func FormatFloat(f float64) string {
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
	if abs >= 1e-3 && abs < 1e7 {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.ContainsRune(s, '.') {
			s += ".0"
		}
		return s
	}
	return formatFloatScientific(f)
}

func formatFloatScientific(f float64) string {
	s := strconv.FormatFloat(f, 'e', -1, 64)
	eIdx := strings.IndexByte(s, 'e')
	mantissa := s[:eIdx]
	exp := s[eIdx+1:]
	sign := ""
	switch exp[0] {
	case '+':
		exp = exp[1:]
	case '-':
		sign = "-"
		exp = exp[1:]
	}
	exp = strings.TrimLeft(exp, "0")
	if exp == "" {
		exp = "0"
	}
	mantissa = strings.TrimSuffix(mantissa, ".0")
	return mantissa + "e" + sign + exp
}
