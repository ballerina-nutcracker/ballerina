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

// Package testfilter decides which tests of a testplan.TestPlan run for a
// `bal test` invocation, from --tests, --groups, --disable-groups and
// --rerun-failed.
package testfilter

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/test/0.0.1/go1.27/native"
	"github.com/ballerina-nutcracker/ballerina/testplan"
	"github.com/ballerina-nutcracker/ballerina/testrunner"
)

// SelectOptions holds the raw CLI flag values used to compute Select's
// inclusion decision.
type SelectOptions struct {
	Tests, Groups, DisableGroups string // raw, comma-separated CLI flag values
	RerunFailed                  bool
	RerunJSONPath                string // <target>/rerun_test.json; only read if RerunFailed
	ModuleName                   string // fully-qualified (e.g. "pkg" or "pkg.submod")
}

// Select returns the set of plan.Tests' names that should actually run. A
// name present and true in the result should run; every other discovered
// test should not. Suite/each/group hooks are never filtered here —
// testrunner.Run always runs every one of them.
//
// The returned map is never nil: an empty map means "run nothing", distinct
// from testrunner.Run's own nil-means-no-filtering sentinel.
func Select(plan *testplan.TestPlan, opts SelectOptions) (map[string]bool, error) {
	var included map[string]bool
	if opts.RerunFailed {
		var err error
		included, err = selectRerunFailed(opts)
		if err != nil {
			return nil, err
		}
	} else {
		included = selectByFilters(plan, opts)
	}
	closeDependsOn(plan, included)
	return included, nil
}

// selectByFilters applies the groups / disable-groups / hasTest checks,
// ANDed together. If Tests/Groups/DisableGroups are all empty, every test
// matches.
func selectByFilters(plan *testplan.TestPlan, opts SelectOptions) map[string]bool {
	groups := splitList(opts.Groups)
	disableGroups := splitList(opts.DisableGroups)
	filters := parseTestFilters(opts.Tests)

	included := make(map[string]bool, len(plan.Tests))
	for _, t := range plan.Tests {
		if len(groups) > 0 && !hasGroup(t.Groups, groups) {
			continue
		}
		if len(disableGroups) > 0 && hasGroup(t.Groups, disableGroups) {
			continue
		}
		if len(filters) > 0 && !hasTest(t.Name, opts.ModuleName, filters) {
			continue
		}
		included[t.Name] = true
	}
	return included
}

// hasGroup reports whether any of testGroups appears in filterGroups.
func hasGroup(testGroups, filterGroups []string) bool {
	for _, g := range testGroups {
		if slicesContain(filterGroups, g) {
			return true
		}
	}
	return false
}

// testFilter is one parsed --tests entry: a base name or wildcard pattern,
// with any module qualifier and data-provider sub-key already stripped. The
// sub-key is discarded here — it names a data-driven sub-case that only
// exists once the provider function runs; sub-key filtering is applied
// separately via testrunner.ParseSubKeyFilters.
type testFilter struct {
	Name   string // base name or wildcard pattern
	Module string // "" if this entry wasn't module-qualified
}

func parseTestFilters(raw string) []testFilter {
	var filters []testFilter
	for _, entry := range splitList(raw) {
		module, name, _, _ := testrunner.SplitTestsEntry(entry)
		filters = append(filters, testFilter{Name: name, Module: module})
	}
	return filters
}

// hasTest reports whether name matches one of filters, by exact name or '*'
// wildcard, unqualified or qualified by moduleName. "--tests <package>:<fn>"
// is scoped to the default module alone, not any submodule's.
func hasTest(name, moduleName string, filters []testFilter) bool {
	moduleOK := func(f testFilter) bool {
		return f.Module == "" || f.Module == moduleName
	}
	for _, f := range filters {
		if f.Name == name && !strings.Contains(f.Name, "*") && moduleOK(f) {
			return true
		}
	}
	for _, f := range filters {
		if !strings.Contains(f.Name, "*") {
			continue
		}
		if native.MatchWildcard(name, f.Name) && moduleOK(f) {
			return true
		}
	}
	return false
}

// closeDependsOn adds every function transitively reachable via an included
// test's DependsOn, regardless of whether that dependency itself matches any
// filter, belongs to an included group, or is enabled, so "--tests
// dependentFn" auto-includes its dependency and a disabled dependency pulled
// in this way still registers (and still reports depends-on-a-disabled-
// function if applicable).
func closeDependsOn(plan *testplan.TestPlan, included map[string]bool) {
	byName := make(map[string]testplan.TestFunction, len(plan.Tests))
	for _, t := range plan.Tests {
		byName[t.Name] = t
	}
	queue := make([]string, 0, len(included))
	for name := range included {
		queue = append(queue, name)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		t, ok := byName[name]
		if !ok {
			continue
		}
		for _, dep := range t.DependsOn {
			if !included[dep] {
				included[dep] = true
				queue = append(queue, dep)
			}
		}
	}
}

// rerunEntryJSON is the part of testrunner.RerunEntry that Select reads back.
type rerunEntryJSON struct {
	TestNames    []string            `json:"testNames"`
	SubTestNames map[string][]string `json:"subTestNames"`
}

// ErrInvalidRerunData reports that --rerun-failed's rerun_test.json is
// missing, malformed, or has no entry for this module. The user-facing
// message is cli/cmd/test.go's invalidRerunDataMessage; callers match this
// sentinel with errors.Is.
var ErrInvalidRerunData = errors.New("invalid failed test data")

// readRerunEntries reads and parses opts.RerunJSONPath, shared by
// selectRerunFailed and SubKeysFromRerun.
func readRerunEntries(path string) (map[string]rerunEntryJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries map[string]rerunEntryJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// selectRerunFailed reads the module's rerun_test.json entry and uses its
// test names directly, with no group/wildcard/module matching, replacing any
// --tests-based selection entirely. Any read/parse problem (missing file,
// malformed JSON, no entry for this module) is reported as an error rather
// than silently yielding an empty set.
func selectRerunFailed(opts SelectOptions) (map[string]bool, error) {
	entries, err := readRerunEntries(opts.RerunJSONPath)
	if err != nil {
		return nil, ErrInvalidRerunData
	}
	entry, ok := entries[opts.ModuleName]
	if !ok {
		return nil, ErrInvalidRerunData
	}
	included := map[string]bool{}
	for _, name := range entry.TestNames {
		included[name] = true
	}
	return included, nil
}

// SubKeysFromRerun reads opts.RerunJSONPath's entry for opts.ModuleName and
// returns its recorded per-test sub-key filters, in the shape
// testrunner.SubKeyFilters expects — nil if opts.RerunFailed is false, or if
// the file/module entry can't be read.
//
// A map-shaped data provider's recorded key comes back single-quoted (see
// BuildRerunEntry's write-side quoting); this strips that wrapping back off.
// A bare, unquoted key (a tuple-array-shaped provider's) passes through
// unchanged.
func SubKeysFromRerun(opts SelectOptions) map[string][]string {
	if !opts.RerunFailed {
		return nil
	}
	entries, err := readRerunEntries(opts.RerunJSONPath)
	if err != nil {
		return nil
	}
	entry, ok := entries[opts.ModuleName]
	if !ok || len(entry.SubTestNames) == 0 {
		return nil
	}
	result := make(map[string][]string, len(entry.SubTestNames))
	for name, keys := range entry.SubTestNames {
		unquoted := make([]string, len(keys))
		for i, k := range keys {
			unquoted[i] = unquoteMapKey(k)
		}
		result[name] = unquoted
	}
	return result
}

func unquoteMapKey(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return s[1 : len(s)-1]
	}
	return s
}

func splitList(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func slicesContain(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
