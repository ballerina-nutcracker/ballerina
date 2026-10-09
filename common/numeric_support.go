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

package common

import (
	"strconv"
	"strings"
)

// HasHexIndicator checks if literal has hex indicator
// migrated from NumericLiteralSupport.java:77:5
func HasHexIndicator(literalValue string) bool {
	length := len(literalValue)
	// There should be at least 3 characters to form hex literal.
	if length < 3 {
		return false
	}
	// Check whether hex prefix is with positive and negative inputs.
	firstChar := literalValue[1]
	secondChar := literalValue[2]
	return firstChar == 'x' || firstChar == 'X' || secondChar == 'x' || secondChar == 'X'
}

// IsDecimalDiscriminated checks if numeric literal has decimal discriminator (d/D suffix)
// migrated from NumericLiteralSupport.java:110:5
func IsDecimalDiscriminated(literalValue string) bool {
	length := len(literalValue)
	// There should be at least 2 characters to form discriminated decimal literal.
	if length < 2 {
		return false
	}
	lastChar := literalValue[length-1]
	hasDecimalSuffix := (lastChar == 'd' || lastChar == 'D')
	if !hasDecimalSuffix {
		return false
	}
	// Check if it's not a hex literal
	return !HasHexIndicator(literalValue)
}

// NormalizeHexFloatLiteral appends the "p0" exponent a hex float literal needs for strconv.ParseFloat when it has none.
func NormalizeHexFloatLiteral(text string) string {
	if !strings.ContainsAny(text, "pP") {
		return text + "p0"
	}
	return text
}

// ParseIntLiteral parses the text of an int literal. It returns an int64 when the value fits, otherwise the value as a
// float64. ok is false when text is not an int literal or its value does not fit a float64.
func ParseIntLiteral(text string) (value any, ok bool) {
	digits, radix := text, 10
	if HasHexIndicator(text) {
		digits, radix = strings.ReplaceAll(strings.ToLower(text), "0x", ""), 16
	}
	if !isIntLiteralDigits(digits, radix) {
		return nil, false
	}
	if v, err := strconv.ParseInt(digits, radix, 64); err == nil {
		return v, true
	}
	f, err := strconv.ParseFloat(digits, 64)
	if err != nil {
		return nil, false
	}
	return f, true
}

func isIntLiteralDigits(digits string, radix int) bool {
	digits = strings.TrimPrefix(digits, "-")
	if digits == "" {
		return false
	}
	for _, c := range digits {
		isDigit := c >= '0' && c <= '9'
		if !isDigit && (radix != 16 || c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
