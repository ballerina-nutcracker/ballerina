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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ballerina-nutcracker/ballerina/projects"

	"github.com/spf13/cobra"
)

// testResultsReportDir/File mirror jballerina's TesterinaConstants
// REPORT_DIR_NAME/RESULTS_JSON_FILE ("report"/"test_results.json").
const (
	testResultsReportDir  = "report"
	testResultsReportFile = "test_results.json"
)

// testStatusEntryJSON/moduleStatusFileJSON mirror the on-disk shape
// native/test_io.go's writeModuleStatusReport already writes to each
// module's module_status.json (see lib/stdlibs/ballerina/test — kept in
// sync with that file, not re-derived from it, since the two live in
// different Go modules).
type testStatusEntryJSON struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	FailureMessage string `json:"failureMessage,omitempty"`
}

type moduleStatusFileJSON struct {
	TotalTests int                   `json:"totalTests"`
	Passed     int                   `json:"passed"`
	Failed     int                   `json:"failed"`
	Skipped    int                   `json:"skipped"`
	Tests      []testStatusEntryJSON `json:"tests"`
}

// moduleTestResult mirrors jballerina's ModuleStatus entity, adding the
// "name" field the per-module file itself doesn't carry (module_status.json
// is keyed by its own directory name on disk, so the name is supplied by
// the caller instead of read from the file's contents).
type moduleTestResult struct {
	Name       string                `json:"name"`
	TotalTests int                   `json:"totalTests"`
	Passed     int                   `json:"passed"`
	Failed     int                   `json:"failed"`
	Skipped    int                   `json:"skipped"`
	Tests      []testStatusEntryJSON `json:"tests"`
}

// packageTestResult mirrors jballerina's PackageTestResult entity (JSON
// field names match its Gson-serialized shape). Coverage-related fields
// (coveredLines, missedLines, coveragePercentage, moduleCoverage) are
// omitted entirely — code coverage isn't implemented in this port (see
// TODO.md's "Deferred bal test features"), so emitting zero-valued coverage
// fields would misrepresent "not measured" as "measured at zero".
type packageTestResult struct {
	ProjectName  string             `json:"projectName"`
	TotalTests   int                `json:"totalTests"`
	Passed       int                `json:"passed"`
	Failed       int                `json:"failed"`
	Skipped      int                `json:"skipped"`
	ModuleStatus []moduleTestResult `json:"moduleStatus"`
}

// workspaceTestReport mirrors jballerina's TestReport entity — the
// workspace-level wrapper written only when testing a whole workspace
// (a single package's own report is its packageTestResult, unwrapped,
// matching TestCommand.java's `report.getPackages().get(0)` special case).
type workspaceTestReport struct {
	WorkspaceName string              `json:"workspaceName"`
	TotalTests    int                 `json:"totalTests"`
	Passed        int                 `json:"passed"`
	Failed        int                 `json:"failed"`
	Skipped       int                 `json:"skipped"`
	Packages      []packageTestResult `json:"packages"`
}

// buildPackageTestResult reads back every one of the package's modules'
// module_status.json cache files (written by ballerina/test's
// moduleStatusReport whenever --test-report is set — see filter.bal's
// setTestOptions) and aggregates them into one packageTestResult, mirroring
// jballerina's RunTestsTask#performPostTestsTasks. A module with no
// module_status.json (it had zero discovered tests, so startSuite()
// returned via its "No tests found" fast path without ever registering a
// report generator) is simply skipped, matching jballerina's own
// loadModuleStatusFromFile-returns-null-so-continue behavior. The second
// return value is false if no module produced a status file at all (the
// whole package had zero tests), signaling the caller to skip writing a
// report entirely — matching jballerina's own "no tests found" no-op.
func buildPackageTestResult(pkg *projects.Package, modules []*projects.Module, projectDir string) (packageTestResult, bool) {
	result := packageTestResult{ProjectName: pkg.PackageName().Value()}
	hadAny := false

	for _, module := range modules {
		statusPath := filepath.Join(projectDir, projects.TargetDir, "cache", "tests_cache",
			module.ModuleName().String(), "module_status.json")
		data, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}
		var file moduleStatusFileJSON
		if json.Unmarshal(data, &file) != nil {
			continue
		}
		hadAny = true
		result.ModuleStatus = append(result.ModuleStatus, moduleTestResult{
			Name:       module.ModuleName().String(),
			TotalTests: file.TotalTests,
			Passed:     file.Passed,
			Failed:     file.Failed,
			Skipped:    file.Skipped,
			Tests:      file.Tests,
		})
		result.TotalTests += file.TotalTests
		result.Passed += file.Passed
		result.Failed += file.Failed
		result.Skipped += file.Skipped
	}

	sort.Slice(result.ModuleStatus, func(i, j int) bool {
		return result.ModuleStatus[i].Name < result.ModuleStatus[j].Name
	})
	return result, hadAny
}

// writeTestResultsReport writes data (a packageTestResult for a single
// package, or a workspaceTestReport for a whole workspace) as
// target/report/test_results.json under targetRoot, printing the same
// "Generating Test Report" + file-path messages as jballerina's
// TestCommand#generateTestReport. HTML report generation is out of scope
// (no report.zip template ported — see TODO.md).
func writeTestResultsReport(cmd *cobra.Command, targetRoot string, data any) error {
	reportDir := filepath.Join(targetRoot, projects.TargetDir, testResultsReportDir)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return testError("error occurred while generating test report: %w", err)
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return testError("error occurred while generating test report: %w", err)
	}

	reportPath := filepath.Join(reportDir, testResultsReportFile)
	if err := os.WriteFile(reportPath, jsonBytes, 0o644); err != nil {
		return testError("error occurred while generating test report: %w", err)
	}

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Generating Test Report")
	_, _ = fmt.Fprintf(out, "\t%s\n\n", reportPath)
	return nil
}
