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

package testrunner

import (
	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/test/0.0.1/go1.27/native"
	"reflect"
	"testing"
)

func TestParseSubKeyFilters(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want SubKeyFilters
	}{
		{"empty", "", SubKeyFilters{}},
		{"no sub-keys at all", "testSquare,testOther", SubKeyFilters{}},
		{"single key", "testSquare#one", SubKeyFilters{"testSquare": {"one"}}},
		{
			"accumulates multiple keys for the same test, in order",
			"testSquare#one,testSquare#three",
			SubKeyFilters{"testSquare": {"one", "three"}},
		},
		{
			"mixes plain and keyed entries",
			"testOther,testSquare#one",
			SubKeyFilters{"testSquare": {"one"}},
		},
		{
			"module-qualified entry still extracts the key",
			"mymod:testSquare#one",
			SubKeyFilters{"testSquare": {"one"}},
		},
		{"wildcard key kept as-is, resolved later by matches", "testSquare#t*", SubKeyFilters{"testSquare": {"t*"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseSubKeyFilters(tc.raw)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseSubKeyFilters(%q) = %+v, want %+v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestSubKeyFilters_Matches(t *testing.T) {
	f := SubKeyFilters{"testSquare": {"one", "t*"}}

	tests := []struct {
		name, key string
		want      bool
	}{
		{"testSquare", "one", true},
		{"testSquare", "two", true},   // matches wildcard "t*"
		{"testSquare", "three", true}, // matches wildcard "t*"
		{"testSquare", "four", false},
		{"testUnfiltered", "anything", true}, // no entry at all — vacuously true
	}
	for _, tc := range tests {
		if got := f.matches(tc.name, tc.key); got != tc.want {
			t.Errorf("matches(%q, %q) = %v, want %v", tc.name, tc.key, got, tc.want)
		}
	}
}

// TestSubKeyFilters_MatchesSpecialCharacterKey is a regression test for a
// real bug this port shipped with: a data-provider key containing a
// delimiter character (here a comma and a double-quote) can't be typed
// literally on a command line, so the user supplies it percent-encoded.
// skipDataDrivenTest escapes both the filter and the row key before
// comparing; this port originally compared them raw, so `--tests
// 'fn#a%2Cb%22c'` silently selected no rows at all.
//
// Ported from the deleted corpus fixture
// testerina-dataprovider-special-char-key-v, which is what caught it.
func TestSubKeyFilters_MatchesSpecialCharacterKey(t *testing.T) {
	const rawKey = `a,b"c`
	escaped := native.EscapeSpecialCharacters(rawKey) // "a%2Cb%22c"

	f := SubKeyFilters{"testFunction3": {escaped}}
	if !f.matches("testFunction3", rawKey) {
		t.Errorf("escaped CLI filter %q should match raw row key %q", escaped, rawKey)
	}
	if f.matches("testFunction3", "other") {
		t.Error(`"other" should not match`)
	}
}
