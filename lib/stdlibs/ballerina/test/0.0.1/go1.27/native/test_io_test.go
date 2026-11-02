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
	"testing"

	"github.com/ballerina-nutcracker/ballerina/values"
)

func TestMatchWildcard(t *testing.T) {
	cases := []struct {
		name, pattern string
		want          bool
	}{
		{"testFoo", "testFoo", true},
		{"testFoo", "test*", true},
		{"testFooBar", "test*Bar", true},
		{"testFoo", "other*", false},
		{"testFoo(int)", "testFoo*", true}, // regex metachars in the name must not be interpreted
		{"testXFoo", "test.Foo", false},    // "." in the pattern is literal, not "any character"
		{"test.Foo", "test.Foo", true},     // literal "." in the pattern still matches itself
		{"testFoo", `test\`, false},        // a trailing "\" in the pattern must not break compilation
	}
	for _, c := range cases {
		got, err := matchWildcard(nil, []values.BalValue{c.name, c.pattern})
		if err != nil {
			t.Fatalf("matchWildcard(%q, %q): %v", c.name, c.pattern, err)
		}
		if got != c.want {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v", c.name, c.pattern, got, c.want)
		}
	}
}

func TestEscapeSpecialCharacters(t *testing.T) {
	got, err := escapeSpecialCharacters(nil, []values.BalValue{`a,b"c`})
	if err != nil {
		t.Fatalf("escapeSpecialCharacters: %v", err)
	}
	s, _ := got.(string)
	if s == `a,b"c` {
		t.Errorf("expected special characters to be encoded, got unchanged %q", s)
	}
}

func TestIsBalanced(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"(a[b]{c})", true},
		{"(a[b)", false},
		{"no brackets", true},
		{")(", false},
	}
	for _, c := range cases {
		if got := isBalanced(c.s); got != c.want {
			t.Errorf("isBalanced(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}
