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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/test/0.0.1/go1.27/native"
)

// TestStatusEntry mirrors ballerina/test's TestStatusEntry record — one row
// of module_status.json (native test_io.go's testStatusEntryJSON).
type TestStatusEntry struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	FailureMessage string `json:"failureMessage,omitempty"`
}

// ModuleStatus mirrors module_status.json's top-level shape (native
// test_io.go's moduleStatusReportJSON) — the file cli/cmd/testreport.go
// already reads, unchanged, to build the final test_results.json.
type ModuleStatus struct {
	TotalTests int               `json:"totalTests"`
	Passed     int               `json:"passed"`
	Failed     int               `json:"failed"`
	Skipped    int               `json:"skipped"`
	Tests      []TestStatusEntry `json:"tests"`
}

// BuildModuleStatus mirrors report.bal#moduleStatusReport: one entry per
// Result, grouped passed-then-failed-then-skipped (ReportData's three
// separate arrays, each iterated in full before the next), names escaped
// the same way the original keeps delimiter characters (commas, brackets,
// quotes, ...) out of a test name before it can corrupt a comma-separated
// --tests value reconstructed from it later.
//
// Deliberately NOT ported: moduleStatusReport's replaceDoubleQuotes
// pre-escape on failureMessage. That exists to protect a hand-built JSON
// string from an unescaped quote; encoding/json.Marshal already escapes
// quotes correctly on its own, so applying it here would double-escape
// instead of reproducing the original's actual intent.
func BuildModuleStatus(report RunReport) ModuleStatus {
	status := ModuleStatus{}
	for _, r := range resultsByStatus(report.Results, Pass) {
		status.Tests = append(status.Tests, TestStatusEntry{Name: native.EscapeSpecialCharacters(r.Name), Status: "PASSED"})
	}
	for _, r := range resultsByStatus(report.Results, Fail) {
		status.Tests = append(status.Tests, TestStatusEntry{
			Name:           native.EscapeSpecialCharacters(r.Name),
			Status:         "FAILURE",
			FailureMessage: r.Message,
		})
	}
	for _, r := range resultsByStatus(report.Results, Skip) {
		status.Tests = append(status.Tests, TestStatusEntry{Name: native.EscapeSpecialCharacters(r.Name), Status: "SKIPPED"})
	}
	s := Summarize(report)
	status.Passed, status.Failed, status.Skipped = s.Passed, s.Failed, s.Skipped
	status.TotalTests = s.Passed + s.Failed + s.Skipped
	return status
}

// WriteModuleStatusFile writes status to path as JSON, mirroring native
// writeModuleStatusReport's mkdir-then-overwrite side effect.
func WriteModuleStatusFile(path string, status ModuleStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return writeFileCreatingDir(path, data)
}

// RerunEntry mirrors one module's entry in rerun_test.json (native
// test_io.go's rerunEntryJSON / writeModuleRerunEntry's arguments).
type RerunEntry struct {
	TestNames       []string            `json:"testNames"`
	TestModuleNames map[string]*string  `json:"testModuleNames"`
	SubTestNames    map[string][]string `json:"subTestNames"`
}

// BuildRerunEntry mirrors report.bal#failedTestsReport: one TestNames entry
// per failed Result (the "#key" suffix split off into SubTestNames, keyed by
// the base name) — ported including the original's own duplication, not
// deduplicated: two failing rows of the same data-driven test ("testSquare
// #one" and "testSquare#three" both failing) push "testSquare" into
// TestNames twice, matching failedTestsReport's unconditional
// testNames.push(testPrefix) inside its per-failure loop. A data-driven sub
// key is single-quoted when its provider was map-shaped
// (DataDrivenMapOfTuple) — ported exactly from failedTestsReport's own
// TestType check — and left bare for the tuple-array shape.
func BuildRerunEntry(report RunReport, moduleName string) RerunEntry {
	entry := RerunEntry{TestModuleNames: map[string]*string{}, SubTestNames: map[string][]string{}}
	for _, r := range resultsByStatus(report.Results, Fail) {
		prefix, suffix := splitResultName(r.Name)
		entry.TestNames = append(entry.TestNames, prefix)
		name := moduleName
		entry.TestModuleNames[prefix] = &name
		if suffix != "" {
			if r.Type == DataDrivenMapOfTuple {
				suffix = "'" + suffix + "'"
			}
			entry.SubTestNames[prefix] = append(entry.SubTestNames[prefix], suffix)
		}
	}
	return entry
}

// WriteRerunEntry merges entry into path's existing per-module map (if any)
// under moduleName, then writes the whole file back — mirroring native
// writeModuleRerunEntry's read-merge-write behavior, which must preserve
// other modules' already-recorded entries from a multi-module test run. A
// missing or unparseable existing file is treated as an empty map, matching
// native's own readRerunFile-fails-so-start-fresh fallback.
func WriteRerunEntry(path, moduleName string, entry RerunEntry) error {
	all := map[string]RerunEntry{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &all)
	}
	all[moduleName] = entry
	data, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return writeFileCreatingDir(path, data)
}

func writeFileCreatingDir(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func splitResultName(name string) (prefix, suffix string) {
	if idx := strings.IndexByte(name, '#'); idx >= 0 {
		return name[:idx], name[idx+1:]
	}
	return name, ""
}

func resultsByStatus(results []Result, status Status) []Result {
	var matched []Result
	for _, r := range results {
		if r.Status == status {
			matched = append(matched, r)
		}
	}
	return matched
}
