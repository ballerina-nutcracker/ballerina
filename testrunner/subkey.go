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
	"strings"

	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/test/0.0.1/go1.27/native"
)

// SubKeyFilters maps a data-driven test's base name to the raw sub-key
// patterns requested via "name#key" --tests entries. A test absent from
// this map has no sub-key restriction: every data row runs.
type SubKeyFilters map[string][]string

// ParseSubKeyFilters extracts sub-key filters from a raw, comma-separated
// --tests value. A test named more than once accumulates every suffix under
// the one entry, not just the last.
func ParseSubKeyFilters(tests string) SubKeyFilters {
	filters := make(SubKeyFilters)
	for _, entry := range strings.Split(tests, ",") {
		if entry == "" {
			continue
		}
		name := entry
		if colon := strings.Index(name, ":"); colon >= 0 {
			if hash := strings.Index(name, "#"); hash < 0 || hash > colon {
				name = name[colon+1:]
			}
		}
		hash := strings.Index(name, "#")
		if hash < 0 {
			continue
		}
		base, key := name[:hash], name[hash+1:]
		// jballerina's "--tests name#'key'" convention wraps a sub-key in
		// single quotes when disambiguating from the comma-separated list;
		// strip only a matching outer pair before matching in matches().
		if len(key) >= 2 && key[0] == '\'' && key[len(key)-1] == '\'' {
			key = key[1 : len(key)-1]
		}
		filters[base] = append(filters[base], key)
	}
	return filters
}

// matches reports whether a row's key, for the data-driven test named name,
// should actually execute — ported from skipDataDrivenTest's suffixMatch:
// true outright if name has no recorded sub-key filter (the common case),
// otherwise an exact or wildcard match against any one of its filters.
func (f SubKeyFilters) matches(name, key string) bool {
	keys, ok := f[name]
	if !ok {
		return true
	}
	// Both sides must be escaped: a row key with a delimiter (comma, quote)
	// can't be typed literally on the CLI, so the user percent-encodes it
	// while the raw key stays unescaped. Escaping both normalizes them.
	escapedKey := native.EscapeSpecialCharacters(key)
	for _, k := range keys {
		escapedFilter := native.EscapeSpecialCharacters(k)
		if escapedFilter == escapedKey {
			return true
		}
		// Wildcard matching only when the filter actually carries one —
		// matching skipDataDrivenTest's own `includesStr(.., WILDCARD)`
		// guard, which keeps a literal key from being treated as a pattern.
		if strings.Contains(escapedFilter, "*") {
			if matched, err := native.MatchWildcard(escapedKey, escapedFilter); err == nil && matched {
				return true
			}
		}
	}
	return false
}

// hasEnabledDependent reports whether a sub-key filter should be ignored
// entirely because a test transitively depending on it (at any depth) will
// actually run — a dependent may need every row's side effects.
func (r *runner) hasEnabledDependent(name string) bool {
	seen := make(map[string]bool)
	queue := append([]string(nil), r.graph.Dependents[name]...)
	for len(queue) > 0 {
		dep := queue[0]
		queue = queue[1:]
		if seen[dep] {
			continue
		}
		seen[dep] = true
		if t, ok := r.byName[dep]; ok && t.Enable && (r.included == nil || r.included[dep]) {
			return true
		}
		queue = append(queue, r.graph.Dependents[dep]...)
	}
	return false
}

// subKeyMatches is executeDataDriven's entry point for the sub-key decision:
// no filter at all, or a dependent that needs every row, both mean "run
// every row" regardless of what SubKeyFilters itself says.
func (r *runner) subKeyMatches(testName, key string) bool {
	if len(r.subKeyFilters) == 0 {
		return true
	}
	if r.hasEnabledDependent(testName) {
		return true
	}
	return r.subKeyFilters.matches(testName, key)
}
