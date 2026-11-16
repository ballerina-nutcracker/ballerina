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

// testResultsReportDir/File are the on-disk "report"/"test_results.json"
// path components under the target directory.
const (
	testResultsReportDir  = "report"
	testResultsReportFile = "test_results.json"
)

// testStatusEntryJSON/moduleStatusFileJSON mirror module_status.json's
// on-disk shape, kept in sync by hand with testrunner.TestStatusEntry/
// ModuleStatus, the types that write it.
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

// moduleTestResult adds a "name" field to module_status.json's content; the
// per-module file itself doesn't carry one, since it's keyed by its own
// directory name on disk, so the name is supplied by the caller instead.
type moduleTestResult struct {
	Name       string                `json:"name"`
	TotalTests int                   `json:"totalTests"`
	Passed     int                   `json:"passed"`
	Failed     int                   `json:"failed"`
	Skipped    int                   `json:"skipped"`
	Tests      []testStatusEntryJSON `json:"tests"`
}

// packageTestResult is one package's test_results.json content. Coverage
// fields are omitted because coverage isn't measured.
type packageTestResult struct {
	ProjectName  string             `json:"projectName"`
	TotalTests   int                `json:"totalTests"`
	Passed       int                `json:"passed"`
	Failed       int                `json:"failed"`
	Skipped      int                `json:"skipped"`
	ModuleStatus []moduleTestResult `json:"moduleStatus"`
}

// workspaceTestReport is the workspace-level wrapper written only when
// testing a whole workspace; a single package's own report is its
// packageTestResult, unwrapped.
type workspaceTestReport struct {
	WorkspaceName string              `json:"workspaceName"`
	TotalTests    int                 `json:"totalTests"`
	Passed        int                 `json:"passed"`
	Failed        int                 `json:"failed"`
	Skipped       int                 `json:"skipped"`
	Packages      []packageTestResult `json:"packages"`
}

// buildPackageTestResult aggregates the modules' module_status.json files,
// skipping modules without one. It returns false if none had one.
func buildPackageTestResult(pkg *projects.Package, modules []*projects.Module, targetDir string) (packageTestResult, bool) {
	result := packageTestResult{ProjectName: pkg.PackageName().Value()}
	hadAny := false

	for _, module := range modules {
		statusPath := filepath.Join(targetDir, "cache", "tests_cache",
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
// report/test_results.json under targetDir, printing a "Generating Test
// Report" message with the file path. HTML report generation is out of
// scope.
func writeTestResultsReport(cmd *cobra.Command, targetDir string, data any) error {
	reportDir := filepath.Join(targetDir, testResultsReportDir)
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
