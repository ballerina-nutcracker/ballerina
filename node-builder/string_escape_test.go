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

package nodebuilder

import (
	"testing"
)

func TestUnescapeBallerinaString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"test\\nname", "test\nname"},
		{"test\\u{41}", "testA"},
		{"\\u{0048}ello", "Hello"},
		{"test\\u{1F600}", "test😀"},
		{"test\\u{5C}", "test\\"},
		{"\\\\u{61}", "\\u{61}"},
		{"test\\\\\\u{61}", "test\\a"},
		{"Line1\\nLine2\\u{0009}Tab", "Line1\nLine2\tTab"},
		{"", ""},
		{"\\u0041\\u0061", "Aa"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := unescapeBallerinaString(tt.input)
			if got != tt.expected {
				t.Errorf("unescapeBallerinaString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
