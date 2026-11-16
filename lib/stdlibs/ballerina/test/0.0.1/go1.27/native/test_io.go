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
	"regexp"
	"strings"
)

// bracketChars/specialChars are the characters EscapeSpecialCharacters encodes.
var bracketChars = []string{"{", "}", "[", "]", "(", ")"}
var specialChars = []string{",", `\n`, `\r`, `\t`, "\n", "\r", "\t", `"`, `\`, "!", "`"}

// urlEncodeChar matches Java's URLEncoder.encode behavior for the single
// characters this module ever encodes (ASCII punctuation/whitespace only).
func urlEncodeChar(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		fmt.Fprintf(&b, "%%%02X", s[i])
	}
	return b.String()
}

func encodeChars(s string, chars []string) string {
	for _, c := range chars {
		if strings.Contains(s, c) {
			s = strings.ReplaceAll(s, c, urlEncodeChar(c))
		}
	}
	return s
}

// isBalanced reports whether s's brackets are properly nested and matched.
func isBalanced(s string) bool {
	var stack []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			stack = append(stack, s[i])
		case ')', ']', '}':
			if len(stack) == 0 {
				return false
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			switch s[i] {
			case ')':
				if top == '{' || top == '[' {
					return false
				}
			case '}':
				if top == '(' || top == '[' {
					return false
				}
			case ']':
				if top == '(' || top == '{' {
					return false
				}
			}
		}
	}
	return len(stack) == 0
}

// EscapeSpecialCharacters percent-encodes delimiter characters (commas,
// brackets, quotes, ...) out of key, so it's safe to write into
// module_status.json or reconstruct into a comma-separated --tests value.
func EscapeSpecialCharacters(key string) string {
	if !isBalanced(key) {
		key = encodeChars(key, bracketChars)
	}
	return encodeChars(key, specialChars)
}

// MatchWildcard reports whether name matches pattern, where '*' matches any
// run of characters and everything else is literal.
func MatchWildcard(name, pattern string) bool {
	segments := strings.Split(pattern, "*")
	quoted := make([]string, len(segments))
	for i, seg := range segments {
		quoted[i] = regexp.QuoteMeta(seg)
	}
	goPattern := strings.Join(quoted, ".*")

	// Built entirely from regexp.QuoteMeta segments joined by ".*", so this
	// can never fail to compile.
	re := regexp.MustCompile("^(?:" + goPattern + ")$")
	return re.MatchString(name)
}
