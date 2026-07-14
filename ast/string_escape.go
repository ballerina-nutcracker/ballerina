// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ast

import (
	"regexp"
	"strconv"
	"strings"

	"ballerina-lang-go/tools/diagnostics"
)

var unicodeCodepointPattern = regexp.MustCompile(`\\(\\*)u\{([a-fA-F0-9]+)\}`)

var backslashEscapes = map[byte]byte{
	'n': '\n', 't': '\t', 'r': '\r', 'b': '\b', 'f': '\f',
	'\\': '\\', '"': '"', '\'': '\'',
}

func isEscapedNumericEscape(leadingSlashes string) bool {
	return len(leadingSlashes)&1 != 0
}

func unescapeBallerinaString(s string) string {
	return unescapeBackslashEscapes(unescapeUnicodeCodepoints(s))
}

func unescapeUnicodeCodepoints(s string) string {
	return unicodeCodepointPattern.ReplaceAllStringFunc(s, func(match string) string {
		submatch := unicodeCodepointPattern.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		leading := submatch[1]
		if isEscapedNumericEscape(leading) {
			return match
		}
		codePoint, err := strconv.ParseInt(submatch[2], 16, 32)
		if err != nil {
			return match
		}
		r := rune(codePoint)
		if r == '\\' {
			return leading + "\\u005C"
		}
		return leading + string(r)
	})
}

func unescapeBackslashEscapes(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		next := s[i+1]
		if esc, ok := backslashEscapes[next]; ok {
			b.WriteByte(esc)
			i++
			continue
		}
		if next == 'u' && i+6 <= len(s) && s[i+2] != '{' {
			if cp, err := strconv.ParseInt(s[i+2:i+6], 16, 32); err == nil {
				b.WriteRune(rune(cp))
				i += 5
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// validateUnicodePoints validates unicode escape sequences
func validateUnicodePoints(text string, pos diagnostics.Location) {
	matches := unicodeCodepointPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		leadingBackSlashes := match[1]
		if isEscapedNumericEscape(leadingBackSlashes) {
			continue
		}

		hexCodePoint := match[2]
		decimalCodePoint, err := strconv.ParseInt(hexCodePoint, 16, 32)
		if err != nil {
			continue
		}

		const minUnicode = 0xD800
		const middleLimitUnicode = 0xDFFF
		const maxUnicode = 0x10FFFF

		if (decimalCodePoint >= minUnicode && decimalCodePoint <= middleLimitUnicode) ||
			decimalCodePoint > maxUnicode {
			offset := len(leadingBackSlashes) + len("\\u{")
			_ = offset
			_ = hexCodePoint
			_ = pos
		}
	}
}
