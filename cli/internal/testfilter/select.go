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

// Package testfilter computes, in Go, which of a testplan.TestPlan's tests
// should actually run for a given `bal test` invocation — the
// --tests/--groups/--disable-groups/--rerun-failed inclusion decision that
// lib/stdlibs/ballerina/test/0.0.1/go1.27/filter.bal historically made
// entirely in Ballerina, at registration time.
//
// This is the Go-native equivalent of upstream's compiler-plugin-based test
// framework design's Schedule (github.com/ballerina-nutcracker/ballerina PRs
// #921/#922): cli/cmd/test.go's testrunner-based wiring calls this package's
// Select directly and never calls into ballerina/test's own
// setTestOptions/filter.bal at all — this package is the sole owner of the
// inclusion decision (and, for --rerun-failed, of detecting and reporting an
// invalid rerun_test.json), not a Go-side pre-filter sitting in front of a
// Ballerina backstop.
package testfilter

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/test/0.0.1/go1.27/native"
	"github.com/ballerina-nutcracker/ballerina/testplan"
)

// SelectOptions mirrors the raw CLI flags/context filter.bal's setTestOptions
// already receives — see cli/cmd/test.go's existing call.
type SelectOptions struct {
	Tests, Groups, DisableGroups string // raw, comma-separated CLI flag values
	RerunFailed                  bool
	RerunJSONPath                string // <target>/rerun_test.json; only read if RerunFailed
	ModuleName, PackageName      string // ModuleName is already fully-qualified (e.g. "pkg" or "pkg.submod")
}

// Select returns the set of plan.Tests' names that should actually run. A
// name present and true in the result should run; any other discovered test
// name should not. Suite/each/group hooks are never filtered by this
// decision — testrunner.Run always runs every one of them.
//
// The returned map is never nil, distinguishing "restrict to this set"
// (including an empty set, meaning "run nothing") from the nil sentinel
// testrunner.Run itself treats as "no filtering at all" when included is
// never passed (e.g. a direct, non-CLI caller).
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

// selectByFilters ports filter.bal#registerTestConfig's three independent,
// ANDed checks (groups / disable-groups / hasTest), applied here before
// registration instead of inside it. If Tests/Groups/DisableGroups are all
// empty, every test matches — the same as filter.bal's own "no filter set"
// default.
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
		if len(filters) > 0 && !hasTest(t.Name, opts.ModuleName, opts.PackageName, filters) {
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
// sub-key is deliberately discarded — it names a data-driven sub-case that
// only exists once the provider function runs, a runtime concern this
// package doesn't resolve (see the package doc comment). The original,
// unmodified --tests string still reaches ballerina/test's own filter.bal,
// which continues to narrow data-driven sub-cases by key exactly as today.
type testFilter struct {
	Name   string // base name or wildcard pattern
	Module string // "" if this entry wasn't module-qualified
}

func parseTestFilters(raw string) []testFilter {
	var filters []testFilter
	for _, entry := range splitList(raw) {
		module := ""
		name := entry
		if colon := strings.Index(name, ":"); colon >= 0 {
			// A '#' (data-key separator) before the colon means the colon
			// belongs to something after the data key, not a module
			// qualifier — matches filter.bal#containsAPrefix.
			if hash := strings.Index(name, "#"); hash < 0 || hash > colon {
				module, name = name[:colon], name[colon+1:]
			}
		}
		if hash := strings.Index(name, "#"); hash >= 0 {
			name = name[:hash]
		}
		filters = append(filters, testFilter{Name: name, Module: module})
	}
	return filters
}

// hasTest ports filter.bal#hasTest/matchModuleName. Module qualification
// matches either the test's exact fully-qualified module name or the bare
// package name (meaning "any module in this package") — a simpler, exact
// comparison than filter.bal's own substring-containment check, safe to
// differ from since filter.bal remains the authoritative backstop: any test
// this function over-includes is still subject to filter.bal's own check
// once registered.
func hasTest(name, moduleName, packageName string, filters []testFilter) bool {
	moduleOK := func(f testFilter) bool {
		return f.Module == "" || f.Module == moduleName || f.Module == packageName
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
		if matched, err := native.MatchWildcard(name, f.Name); err == nil && matched && moduleOK(f) {
			return true
		}
	}
	return false
}

// closeDependsOn adds every function transitively reachable via an included
// test's DependsOn, regardless of whether that dependency itself matches any
// filter, belongs to an included group, or is enabled — reproducing both
// "--tests dependentFn auto-includes its dependency" and "a disabled
// dependency pulled in this way still gets registered, so the
// depends-on-a-disabled-function error still fires" (see the package doc
// comment and TODO.md).
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

// rerunEntryJSON mirrors rerun_test.json's per-module entry shape, matching
// native/test_io.go's rerunEntryJSON and cli/cmd/testreport.go's reader.
type rerunEntryJSON struct {
	TestNames    []string            `json:"testNames"`
	SubTestNames map[string][]string `json:"subTestNames"`
}

// ErrInvalidRerunData reports that --rerun-failed's rerun_test.json is
// missing, malformed, or has no entry for this module — the three
// conditions native/test_io.go's readModuleRerunEntryFactory collapses into
// one message. This package is the only place that detects them, now that
// nothing calls through to ballerina/test's own setTestOptions.
//
// The sentinel's own text is kept lint-clean (lowercase, no trailing
// period); the exact user-facing wording lives with the CLI that prints it
// (cli/cmd/test.go's invalidRerunDataMessage), since that string must match
// jballerina byte-for-byte and is pinned by golden fixtures. Callers
// distinguish it with errors.Is.
var ErrInvalidRerunData = errors.New("invalid failed test data")

// selectRerunFailed reads the module's rerun_test.json entry and uses its
// test names directly, with no group/wildcard/module matching (matching
// filter.bal#parseRerunJson's own behavior of replacing --tests-based
// selection entirely). Any read/parse problem (missing file, malformed
// JSON, no entry for this module) is reported as an error rather than
// silently yielding an empty set.
func selectRerunFailed(opts SelectOptions) (map[string]bool, error) {
	data, err := os.ReadFile(opts.RerunJSONPath)
	if err != nil {
		return nil, ErrInvalidRerunData
	}
	var entries map[string]rerunEntryJSON
	if err := json.Unmarshal(data, &entries); err != nil {
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
// the file/module entry can't be read (Select's own error from
// selectRerunFailed already covers reporting that condition for the same
// file; this just returns a zero value rather than re-detecting it).
//
// A map-shaped data provider's recorded key comes back single-quoted (see
// BuildRerunEntry's write-side quoting, mirroring
// report.bal#failedTestsReport); this strips that wrapping back off,
// mirroring execute.bal#skipDataDrivenTest's own read-side unquoting. A
// bare, unquoted key (a tuple-array-shaped provider's) passes through
// unchanged.
func SubKeysFromRerun(opts SelectOptions) map[string][]string {
	if !opts.RerunFailed {
		return nil
	}
	data, err := os.ReadFile(opts.RerunJSONPath)
	if err != nil {
		return nil
	}
	var entries map[string]rerunEntryJSON
	if err := json.Unmarshal(data, &entries); err != nil {
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
