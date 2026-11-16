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

// TestStatusEntry is one row of module_status.json.
type TestStatusEntry struct {
	Name              string                 `json:"name"`
	Status            string                 `json:"status"`
	FailureMessage    string                 `json:"failureMessage,omitempty"`
	EvaluationSummary *EvaluationSummaryJSON `json:"evaluationSummary,omitempty"`
}

// EvaluationSummaryJSON is module_status.json's "evaluationSummary" field,
// set only for a Pass/Fail entry whose Result.Type is EvalTest.
type EvaluationSummaryJSON struct {
	TargetPassRate   float64             `json:"targetPassRate"`
	ObservedPassRate float64             `json:"observedPassRate"`
	EvaluationRuns   []EvaluationRunJSON `json:"evaluationRuns"`
}

// EvaluationRunJSON mirrors one EvalRun.
type EvaluationRunJSON struct {
	ID           int                     `json:"id"`
	ErrorMessage string                  `json:"errorMessage,omitempty"` // non-data-driven only
	Outcomes     []EvaluationOutcomeJSON `json:"outcomes,omitempty"`     // data-driven only
	PassRate     float64                 `json:"passRate,omitempty"`     // data-driven only
}

// EvaluationOutcomeJSON mirrors one EvalOutcome.
type EvaluationOutcomeJSON struct {
	ID           string `json:"id"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// buildEvaluationSummaryJSON returns nil for a non-evaluation Result (eval
// is nil), so TestStatusEntry's omitempty drops the field entirely.
func buildEvaluationSummaryJSON(eval *EvalSummary) *EvaluationSummaryJSON {
	if eval == nil {
		return nil
	}
	runs := make([]EvaluationRunJSON, len(eval.RunResults))
	for i, run := range eval.RunResults {
		outcomes := make([]EvaluationOutcomeJSON, len(run.Outcomes))
		for j, o := range run.Outcomes {
			outcomes[j] = EvaluationOutcomeJSON(o)
		}
		runs[i] = EvaluationRunJSON{ID: run.ID, ErrorMessage: run.ErrorMessage, Outcomes: outcomes, PassRate: run.PassRate}
	}
	return &EvaluationSummaryJSON{
		TargetPassRate:   eval.TargetPassRate,
		ObservedPassRate: eval.ObservedPassRate,
		EvaluationRuns:   runs,
	}
}

// ModuleStatus is module_status.json's top-level shape, read by
// cli/cmd/testreport.go to build the final test_results.json.
type ModuleStatus struct {
	TotalTests int               `json:"totalTests"`
	Passed     int               `json:"passed"`
	Failed     int               `json:"failed"`
	Skipped    int               `json:"skipped"`
	Tests      []TestStatusEntry `json:"tests"`
}

// BuildModuleStatus lists passed, then failed, then skipped results with
// escaped names. failureMessage isn't pre-escaped: json.Marshal escapes it.
func BuildModuleStatus(report RunReport) ModuleStatus {
	status := ModuleStatus{Tests: []TestStatusEntry{}}
	for _, r := range resultsByStatus(report.Results, Pass) {
		status.Tests = append(status.Tests, TestStatusEntry{
			Name:              native.EscapeSpecialCharacters(r.Name),
			Status:            "PASSED",
			EvaluationSummary: buildEvaluationSummaryJSON(r.Eval),
		})
	}
	for _, r := range resultsByStatus(report.Results, Fail) {
		status.Tests = append(status.Tests, TestStatusEntry{
			Name:              native.EscapeSpecialCharacters(r.Name),
			Status:            "FAILURE",
			FailureMessage:    r.Message,
			EvaluationSummary: buildEvaluationSummaryJSON(r.Eval),
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

// WriteModuleStatusFile writes status to path as JSON, creating path's
// parent directory if needed.
func WriteModuleStatusFile(path string, status ModuleStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return writeFileCreatingDir(path, data)
}

// RerunEntry is one module's entry in rerun_test.json.
type RerunEntry struct {
	TestNames       []string            `json:"testNames"`
	TestModuleNames map[string]*string  `json:"testModuleNames"`
	SubTestNames    map[string][]string `json:"subTestNames"`
}

// BuildRerunEntry adds one TestNames entry per failed Result (the "#key"
// suffix split off into SubTestNames, keyed by the base name), not
// deduplicated: two failing rows of the same data-driven test ("testSquare
// #one" and "testSquare#three" both failing) push "testSquare" into
// TestNames twice. A data-driven sub key is single-quoted when its provider
// was map-shaped (DataDrivenMapOfTuple), bare for the tuple-array shape.
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
// under moduleName, then writes the whole file back, preserving other
// modules' already-recorded entries from a multi-module test run. A missing
// or unparseable existing file is treated as an empty map.
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
