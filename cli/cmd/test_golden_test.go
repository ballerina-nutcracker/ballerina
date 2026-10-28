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

//go:build !js && !wasm

package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// updateGolden regenerates every golden fixture under testdata/golden-output
// instead of comparing against it — mirrors corpus's own `-update` flag
// (corpus/integration_test.go) for consistency across the repo's test
// infrastructure.
var updateGolden = flag.Bool("update", false, "update golden bal-test output fixtures")

var executionTimePattern = regexp.MustCompile(`Test execution time : [0-9.]+s`)

// normalizeGoldenOutput masks the two known sources of non-determinism in a
// captured `bal test` run: the wall-clock execution time, and the fixture's
// own absolute path (which varies by checkout location/CI workdir, and here
// is additionally a fresh t.TempDir() per test run — see copyFixtureDir).
// Every other line is otherwise fully deterministic, so no other
// normalization is needed, unlike jballerina's own broader "drop any line
// starting with a letter" regex in AssertionUtils.assertOutput, which is
// coarser than we need here.
func normalizeGoldenOutput(output, fixtureAbsDir string) string {
	output = executionTimePattern.ReplaceAllString(output, "Test execution time : <DURATION>s")
	output = strings.ReplaceAll(output, filepath.ToSlash(fixtureAbsDir), "<FIXTURE>")
	output = strings.ReplaceAll(output, fixtureAbsDir, "<FIXTURE>")
	return output
}

// copyFixtureDir copies testdata/golden/<name> into a fresh t.TempDir(),
// returning the copy's absolute path. Every scenario runs against this copy,
// never the checked-in original — `bal test` writes real build artifacts
// (target/, cache/, rerun_test.json) into whatever directory it's pointed
// at, and running straight against testdata/golden/<name> would leave those
// artifacts stray-committed into the fixture tree (confirmed: an earlier
// version of this harness did exactly that before this fix). A multi-step
// scenario (runGoldenScenarioSteps) deliberately reuses the same copy across
// its steps so state like rerun_test.json carries over, exactly as it would
// across two real invocations of `bal test` in the same project directory.
func copyFixtureDir(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", "golden", name)
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("fixture %s not found: %v", src, err)
	}

	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture %s: %v", src, err)
	}
	return dst
}

// runGoldenCommand runs `bal test` against fixtureDir as a real subprocess
// (go run . test ...), capturing combined stdout+stderr exactly as a real
// terminal would see them — mirroring jballerina's own
// testerina-integration-test harness (BMainInstance.runMainAndReadStdOut),
// which is the only mechanism in either codebase that can verify a real
// test/hook failure's exact console output byte-for-byte: corpus's
// `@output` annotations can't (the annotation parser strips trailing
// whitespace per line, and ballerina/test's own
// report.bal#formatFailedError unconditionally emits trailing-tabs-only
// lines on any failure — see TODO.md). Subprocess execution also means a
// scenario that happens to panic can't take the whole test binary down
// with it, same rationale as TestTestCommand_DataProvider_ArgMismatchDoesNotCrash.
func runGoldenCommand(fixtureDir string, extraArgs ...string) string {
	args := append([]string{"run", ".", "test", fixtureDir}, extraArgs...)
	out, _ := exec.Command("go", args...).CombinedOutput()
	return normalizeGoldenOutput(string(out), fixtureDir)
}

// compareGolden checks actual against testdata/golden-output/<name>.txt, or
// (with -update) writes actual there instead of comparing.
func compareGolden(t *testing.T, name, actual string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "golden-output", name+".txt")
	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("write golden file %s: %v", goldenPath, err)
		}
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file %s (run with -update to create it): %v", goldenPath, err)
	}
	if actual != string(expected) {
		t.Errorf("output for %q doesn't match %s (run with -update to see/accept the new output).\n--- got ---\n%s\n--- want ---\n%s",
			name, goldenPath, actual, string(expected))
	}
}

// runGoldenScenario runs a single `bal test` invocation against a fresh copy
// of testdata/golden/<name> and compares its output to
// testdata/golden-output/<name>.txt.
func runGoldenScenario(t *testing.T, name string, extraArgs ...string) {
	t.Helper()
	fixtureDir := copyFixtureDir(t, name)
	compareGolden(t, name, runGoldenCommand(fixtureDir, extraArgs...))
}

// runGoldenScenarioSteps runs multiple `bal test` invocations in sequence
// against the SAME fresh copy of testdata/golden/<name>, so state that
// persists on disk between real invocations (like target/rerun_test.json)
// carries over from one step to the next exactly as it would for a real
// user running `bal test` twice in the same project. Each step's own
// command line and captured output are concatenated (with a header showing
// the args used) into one combined golden file, so both steps are verified
// together.
func runGoldenScenarioSteps(t *testing.T, name string, steps ...[]string) {
	t.Helper()
	fixtureDir := copyFixtureDir(t, name)

	var combined strings.Builder
	for i, args := range steps {
		if i > 0 {
			combined.WriteString("\n")
		}
		if len(args) > 0 {
			fmt.Fprintf(&combined, "$ bal test %s\n", strings.Join(args, " "))
		} else {
			combined.WriteString("$ bal test\n")
		}
		combined.WriteString(runGoldenCommand(fixtureDir, args...))
	}
	compareGolden(t, name, combined.String())
}

func TestTestCommand_Golden_SkipDependencyFailure(t *testing.T) {
	runGoldenScenario(t, "skip-dependency-failure")
}

func TestTestCommand_Golden_SkipBeforeEachFailure(t *testing.T) {
	runGoldenScenario(t, "skip-beforeeach-failure")
}

func TestTestCommand_Golden_SkipBeforeSuiteFailure(t *testing.T) {
	runGoldenScenario(t, "skip-beforesuite-failure")
}

func TestTestCommand_Golden_SkipBeforeGroupsFailure(t *testing.T) {
	runGoldenScenario(t, "skip-beforegroups-failure")
}

func TestTestCommand_Golden_DataProviderSingleFailureOthersStillRun(t *testing.T) {
	runGoldenScenario(t, "data-provider-single-failure")
}

func TestTestCommand_Golden_RerunFailedWithDataProvider(t *testing.T) {
	runGoldenScenarioSteps(t, "rerun-failed-with-data-provider",
		nil,
		[]string{"--rerun-failed"},
	)
}

func TestTestCommand_Golden_RerunFailedNoPriorRun(t *testing.T) {
	runGoldenScenario(t, "rerun-failed-no-prior-run", "--rerun-failed")
}

func TestTestCommand_Golden_RerunFailedInvalidJson(t *testing.T) {
	runGoldenScenario(t, "rerun-failed-invalid-json", "--rerun-failed")
}

func TestTestCommand_Golden_RerunFailedMissingModuleKey(t *testing.T) {
	runGoldenScenario(t, "rerun-failed-missing-module-key", "--rerun-failed")
}

func TestTestCommand_Golden_AfterHookFailure(t *testing.T) {
	runGoldenScenario(t, "after-hook-failure")
}

func TestTestCommand_Golden_AfterEachFailure(t *testing.T) {
	runGoldenScenario(t, "aftereach-failure")
}

func TestTestCommand_Golden_AfterGroupsFailure(t *testing.T) {
	runGoldenScenario(t, "aftergroups-failure")
}

func TestTestCommand_Golden_DataProviderItselfErrors(t *testing.T) {
	runGoldenScenario(t, "dataprovider-itself-errors")
}

// TestTestCommand_Golden_AssertMapKeyDiff closes a real 0%-coverage gap
// found while checking native/test.go's coverage (2026-09-17):
// getKeysDiff/keysNotIn/joinWithLeadingSpace (assert.bal#getMapValueDiff's
// key-mismatch branch, mirroring jballerina's
// AssertionDiffEvaluator#getKeysDiff) were completely unexercised
// end-to-end — assertEquals on two maps/records with different key sets,
// not just different values, was never actually tested. Golden (not
// corpus): the diff message goes through the same
// report.bal#formatFailedError trailing-tabs-only-line formatting as every
// other assertion failure.
func TestTestCommand_Golden_AssertMapKeyDiff(t *testing.T) {
	runGoldenScenario(t, "assert-map-key-diff")
}
