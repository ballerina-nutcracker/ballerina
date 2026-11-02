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
	"os"
	"regexp"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// bracketChars/specialChars mirror StringUtils#BRACKET_CHARACTERS/SPECIAL_CHARACTERS.
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

// isBalanced mirrors StringUtils#isBalanced.
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

// escapeSpecialCharacters mirrors jballerina's StringUtils#escapeSpecialCharacters.
func escapeSpecialCharacters(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	key, _ := args[0].(string)
	return EscapeSpecialCharacters(key), nil
}

// EscapeSpecialCharacters mirrors jballerina's StringUtils#escapeSpecialCharacters.
// Exported so testrunner can reuse the exact same encoding for
// module_status.json test names instead of duplicating it — the same reuse
// pattern as MatchWildcard below.
func EscapeSpecialCharacters(key string) string {
	if !isBalanced(key) {
		key = encodeChars(key, bracketChars)
	}
	return encodeChars(key, specialChars)
}

// matchWildcard mirrors jballerina's StringUtils#matchWildcard.
func matchWildcard(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	functionName, _ := args[0].(string)
	functionPattern, _ := args[1].(string)

	matched, err := MatchWildcard(functionName, functionPattern)
	if err != nil {
		return values.NewErrorWithMessage("Invalid wildcard pattern: " + err.Error()), nil
	}
	return matched, nil
}

// MatchWildcard mirrors jballerina's StringUtils#matchWildcard: name matches
// pattern where '*' matches any run of characters, everything else literal.
// Exported so cli/internal/testfilter can reuse the exact same algorithm for
// --tests wildcard matching, computed in Go ahead of registration, instead
// of duplicating it.
func MatchWildcard(name, pattern string) (bool, error) {
	segments := strings.Split(pattern, "*")
	quoted := make([]string, len(segments))
	for i, seg := range segments {
		quoted[i] = regexp.QuoteMeta(seg)
	}
	goPattern := strings.Join(quoted, ".*")

	re, err := regexp.Compile("^(?:" + goPattern + ")$")
	if err != nil {
		return false, err
	}
	return re.MatchString(name), nil
}

// isSystemConsole approximates jballerina's System.console() != null check via a TTY test.
func isSystemConsole(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return IsSystemConsole(), nil
}

// IsSystemConsole approximates jballerina's System.console() != null check
// via a TTY test on stdout. Exported so testrunner's console reporter can
// apply the same pass-line-suppression rule report.bal#consoleReport does,
// without duplicating the TTY check.
func IsSystemConsole() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
