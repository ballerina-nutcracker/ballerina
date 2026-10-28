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
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFixture creates a minimal package with the given Ballerina.toml
// package name and test source content, returning its directory.
func writeTestFixture(t *testing.T, pkgName, testSource string) string {
	t.Helper()
	dir := t.TempDir()
	toml := "[package]\norg = \"testorg\"\nname = \"" + pkgName + "\"\nversion = \"0.1.0\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte(toml), 0o644); err != nil {
		t.Fatalf("write Ballerina.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.bal"), []byte("public function main() {\n}\n"), 0o644); err != nil {
		t.Fatalf("write main.bal: %v", err)
	}
	if testSource != "" {
		testsDir := filepath.Join(dir, "tests")
		if err := os.MkdirAll(testsDir, 0o755); err != nil {
			t.Fatalf("mkdir tests: %v", err)
		}
		if err := os.WriteFile(filepath.Join(testsDir, "main_test.bal"), []byte(testSource), 0o644); err != nil {
			t.Fatalf("write test file: %v", err)
		}
	}
	return dir
}

// executeTestCommand runs the test command and returns cobra's own
// stdout/stderr buffers plus the real OS-level stdout the test suite's
// console report is written to. ballerina/test's println (native/test_io.go)
// writes straight to the process's real os.Stdout — the same execution-path
// separation bal run relies on for the programs it executes — so it never
// lands in cobra's SetOut/SetErr buffers; capturing real stdout is the only
// way to observe pass/fail/skip counts and messages from these tests.
//
// Not safe to run in parallel with anything else that reads/writes os.Stdout
// in this process — none of these tests call t.Parallel().
func executeTestCommand(t *testing.T, args ...string) (stdout, cobraStdout, cobraStderr string, err error) {
	t.Helper()

	realStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("create stdout pipe: %v", pipeErr)
	}
	os.Stdout = w

	cmd := createTestCmd()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()

	os.Stdout = realStdout
	_ = w.Close()
	captured, _ := io.ReadAll(r)

	return string(captured), outBuf.String(), errBuf.String(), err
}

func TestTestCommand_AllPassing(t *testing.T) {
	dir := writeTestFixture(t, "allpassing", `import ballerina/test;

@test:Config {}
function testOne() {
    test:assertTrue(true);
}
`)
	stdout, _, stderr, err := executeTestCommand(t, dir)
	if err != nil {
		t.Fatalf("expected success, got error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "1 passing") {
		t.Errorf("expected '1 passing' in output, got: %s", stdout)
	}
	// The package's own main() must never run during `bal test`.
	if strings.Contains(stdout, "MAIN_RAN") {
		t.Errorf("package main() ran unexpectedly during bal test: %s", stdout)
	}
}

func TestTestCommand_FailureExitsNonZeroAndWritesRerunFile(t *testing.T) {
	dir := writeTestFixture(t, "hasfailure", `import ballerina/test;

@test:Config {}
function testGood() {
    test:assertTrue(true);
}

@test:Config {}
function testBad() {
    test:assertTrue(false, "boom");
}
`)
	stdout, _, stderr, err := executeTestCommand(t, dir)
	if err == nil {
		t.Fatalf("expected a non-nil error for a failing suite, got success. stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "1 passing") || !strings.Contains(stdout, "1 failing") {
		t.Errorf("expected one pass and one failure in output, got: %s\nstderr: %s", stdout, stderr)
	}

	rerunPath := filepath.Join(dir, "target", "rerun_test.json")
	data, readErr := os.ReadFile(rerunPath)
	if readErr != nil {
		t.Fatalf("expected rerun_test.json to be written: %v", readErr)
	}
	var rerun map[string]struct {
		TestNames []string `json:"testNames"`
	}
	if err := json.Unmarshal(data, &rerun); err != nil {
		t.Fatalf("unmarshal rerun_test.json: %v", err)
	}
	entry, ok := rerun["hasfailure"]
	if !ok || len(entry.TestNames) != 1 || entry.TestNames[0] != "testBad" {
		t.Errorf("expected rerun_test.json to list only testBad, got: %+v", rerun)
	}
}

func TestTestCommand_RerunFailedOnlyRunsPreviousFailures(t *testing.T) {
	dir := writeTestFixture(t, "rerunpkg", `import ballerina/test;

@test:Config {}
function testGood() {
    test:assertTrue(true);
}

@test:Config {}
function testBad() {
    test:assertTrue(false, "boom");
}
`)
	if _, _, _, err := executeTestCommand(t, dir); err == nil {
		t.Fatal("expected the first run to fail")
	}

	stdout, _, _, err := executeTestCommand(t, dir, "--rerun-failed")
	if err == nil {
		t.Fatalf("expected the rerun to still report the persistent failure. stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "0 passing") || !strings.Contains(stdout, "1 failing") {
		t.Errorf("expected only the previously-failed test to run, got: %s", stdout)
	}
}

func TestTestCommand_NoTestsFound(t *testing.T) {
	dir := writeTestFixture(t, "notests", "")
	stdout, _, stderr, err := executeTestCommand(t, dir)
	if err != nil {
		t.Fatalf("expected success for a package with no tests, got: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "No tests found") {
		t.Errorf("expected 'No tests found', got: %s", stdout)
	}
}

func TestTestCommand_ListGroups(t *testing.T) {
	dir := writeTestFixture(t, "listgroups", `import ballerina/test;

@test:Config {
    groups: ["unit", "fast"]
}
function testOne() {
}
`)
	stdout, _, stderr, err := executeTestCommand(t, dir, "--list-groups")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "unit") || !strings.Contains(stdout, "fast") {
		t.Errorf("expected both groups listed, got: %s", stdout)
	}
}

func TestTestCommand_GroupFiltering(t *testing.T) {
	dir := writeTestFixture(t, "groupfilter", `import ballerina/test;

@test:Config {
    groups: ["unit"]
}
function testUnit() {
    test:assertTrue(true);
}

@test:Config {
    groups: ["integration"]
}
function testIntegration() {
    test:assertTrue(true);
}
`)
	stdout, _, stderr, err := executeTestCommand(t, dir, "--groups", "unit")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "1 passing") {
		t.Errorf("expected exactly one test (testUnit) to run, got: %s", stdout)
	}
}

func TestTestCommand_RejectsUnsupportedFlags(t *testing.T) {
	dir := writeTestFixture(t, "rejectflags", "")
	for _, args := range [][]string{
		{dir, "--graalvm"},
		{dir, "--cloud", "docker"},
	} {
		_, _, _, err := executeTestCommand(t, args...)
		if err == nil {
			t.Errorf("expected %v to be rejected with an error", args)
		}
	}
}

// writeStandaloneFixture creates a single standalone .bal file (no
// Ballerina.toml) at dir/name containing source, returning its full path.
func writeStandaloneFixture(t *testing.T, dir, name, source string) string {
	t.Helper()
	balFile := filepath.Join(dir, name)
	if err := os.WriteFile(balFile, []byte(source), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return balFile
}

func TestTestCommand_StandaloneFile_AllPassing(t *testing.T) {
	dir := t.TempDir()
	balFile := writeStandaloneFixture(t, dir, "standalone.bal", `import ballerina/io;
import ballerina/test;

public function main() {
    io:println("MAIN_RAN_UNEXPECTEDLY");
}

@test:Config {}
function testOne() {
    test:assertTrue(true);
}
`)
	stdout, _, stderr, err := executeTestCommand(t, balFile)
	if err != nil {
		t.Fatalf("expected success, got error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "1 passing") {
		t.Errorf("expected '1 passing' in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "MAIN_RAN") {
		t.Errorf("standalone file's main() ran unexpectedly during bal test: %s", stdout)
	}
}

func TestTestCommand_StandaloneFile_FailureExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	balFile := writeStandaloneFixture(t, dir, "standalone_fail.bal", `import ballerina/test;

@test:Config {}
function testBad() {
    test:assertTrue(false, "boom");
}
`)
	stdout, _, stderr, err := executeTestCommand(t, balFile)
	if err == nil {
		t.Fatalf("expected a non-nil error for a failing suite, got success. stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "1 failing") {
		t.Errorf("expected one failure in output, got: %s\nstderr: %s", stdout, stderr)
	}

	// target/ lands next to the standalone file, not inside a package dir.
	rerunPath := filepath.Join(dir, "target", "rerun_test.json")
	if _, statErr := os.Stat(rerunPath); statErr != nil {
		t.Errorf("expected rerun_test.json at %s: %v", rerunPath, statErr)
	}
}

func TestTestCommand_StandaloneFile_RejectsNonBalFile(t *testing.T) {
	dir := t.TempDir()
	notBal := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notBal, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}
	_, _, _, err := executeTestCommand(t, notBal)
	if err == nil {
		t.Error("expected an error for a non-.bal, non-directory path")
	}
}

// writeWorkspaceFixture creates a workspace at a fresh temp dir containing
// two member packages ("passing", which has one passing test, and "failing",
// which has one failing test) and returns the workspace root plus each
// member's own directory.
func writeWorkspaceFixture(t *testing.T) (workspaceRoot, passingDir, failingDir string) {
	t.Helper()
	root := t.TempDir()

	writeMember := func(name, testSource string) string {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(dir, "tests"), 0o755); err != nil {
			t.Fatalf("mkdir %s/tests: %v", name, err)
		}
		toml := "[package]\norg = \"testorg\"\nname = \"" + name + "\"\nversion = \"0.1.0\"\n"
		if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte(toml), 0o644); err != nil {
			t.Fatalf("write %s/Ballerina.toml: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.bal"), []byte("public function main() {\n}\n"), 0o644); err != nil {
			t.Fatalf("write %s/main.bal: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tests", "main_test.bal"), []byte(testSource), 0o644); err != nil {
			t.Fatalf("write %s/tests/main_test.bal: %v", name, err)
		}
		return dir
	}

	passingDir = writeMember("passing", `import ballerina/test;

@test:Config {}
function testGood() {
    test:assertTrue(true);
}
`)
	failingDir = writeMember("failing", `import ballerina/test;

@test:Config {}
function testBad() {
    test:assertTrue(false, "boom");
}
`)

	wsToml := "[workspace]\npackages = [\"passing\", \"failing\"]\n"
	if err := os.WriteFile(filepath.Join(root, "Ballerina.toml"), []byte(wsToml), 0o644); err != nil {
		t.Fatalf("write workspace Ballerina.toml: %v", err)
	}
	return root, passingDir, failingDir
}

func TestTestCommand_Workspace_RunsAllMembersAndAggregatesFailure(t *testing.T) {
	root, _, _ := writeWorkspaceFixture(t)

	// Package-name headers go through cobra's own OutOrStdout (cobraStdout);
	// each member's actual pass/fail report goes through ballerina/test's
	// println, which writes straight to the real process stdout (see
	// executeTestCommand's doc comment) — both land on the same real stdout
	// fd outside of tests, but the pipe swap here only captures one of them.
	stdout, cobraStdout, _, err := executeTestCommand(t, root)
	if err == nil {
		t.Fatalf("expected an error since one member fails. stdout: %s", stdout)
	}
	if !strings.Contains(cobraStdout, "'passing'") || !strings.Contains(cobraStdout, "'failing'") {
		t.Errorf("expected both member headers, got: %s", cobraStdout)
	}
	// Both members must run — a failing member must not stop the rest.
	if strings.Count(cobraStdout, "Running tests for package") != 2 {
		t.Errorf("expected both members to run, got: %s", cobraStdout)
	}
	if !strings.Contains(stdout, "1 passing") {
		t.Errorf("expected the 'passing' member's result in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "1 failing") {
		t.Errorf("expected the 'failing' member's failure in output, got: %s", stdout)
	}
}

func TestTestCommand_Workspace_SingleMemberOnly(t *testing.T) {
	_, passingDir, failingDir := writeWorkspaceFixture(t)

	stdout, _, stderr, err := executeTestCommand(t, passingDir)
	if err != nil {
		t.Fatalf("expected the passing member alone to succeed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "1 passing") {
		t.Errorf("expected 1 passing, got: %s", stdout)
	}
	if strings.Contains(stdout, "failing") && strings.Contains(stdout, "boom") {
		t.Errorf("the 'failing' member's test must not have run: %s", stdout)
	}

	stdout, _, _, err = executeTestCommand(t, failingDir)
	if err == nil {
		t.Fatalf("expected the failing member alone to fail. stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "1 failing") {
		t.Errorf("expected 1 failing, got: %s", stdout)
	}
}

// writeMultiModuleFixture creates a package (distinct from a workspace) with
// tests in both its default module and a named submodule under modules/ —
// the setup that exposed a bug where only the default module's tests ever
// ran (see TODO.md / CHECKPOINT.md, 2026-09-09).
func writeMultiModuleFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	toml := "[package]\norg = \"testorg\"\nname = \"multimodpkg\"\nversion = \"0.1.0\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte(toml), 0o644); err != nil {
		t.Fatalf("write Ballerina.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.bal"), []byte("public function main() {\n}\n"), 0o644); err != nil {
		t.Fatalf("write main.bal: %v", err)
	}

	defaultTestsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(defaultTestsDir, 0o755); err != nil {
		t.Fatalf("mkdir tests: %v", err)
	}
	defaultTestSource := `import ballerina/test;

@test:Config {}
function testDefaultModule() {
    test:assertTrue(true);
}
`
	if err := os.WriteFile(filepath.Join(defaultTestsDir, "main_test.bal"), []byte(defaultTestSource), 0o644); err != nil {
		t.Fatalf("write default module test file: %v", err)
	}

	submodDir := filepath.Join(dir, "modules", "submod")
	submodTestsDir := filepath.Join(submodDir, "tests")
	if err := os.MkdirAll(submodTestsDir, 0o755); err != nil {
		t.Fatalf("mkdir modules/submod/tests: %v", err)
	}
	if err := os.WriteFile(filepath.Join(submodDir, "lib.bal"),
		[]byte("public function double(int x) returns int {\n    return x * 2;\n}\n"), 0o644); err != nil {
		t.Fatalf("write modules/submod/lib.bal: %v", err)
	}
	submodTestSource := `import ballerina/test;

@test:Config {}
function testSubmodule() {
    test:assertEquals(double(2), 4);
}
`
	if err := os.WriteFile(filepath.Join(submodTestsDir, "lib_test.bal"), []byte(submodTestSource), 0o644); err != nil {
		t.Fatalf("write modules/submod/tests/lib_test.bal: %v", err)
	}

	return dir
}

func TestTestCommand_MultiModulePackage_RunsAllModules(t *testing.T) {
	dir := writeMultiModuleFixture(t)

	stdout, cobraStdout, stderr, err := executeTestCommand(t, dir)
	if err != nil {
		t.Fatalf("expected success, got error: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(cobraStdout, "multimodpkg.submod") {
		t.Errorf("expected a header naming the submodule, got: %s", cobraStdout)
	}
	if !strings.Contains(stdout, "[pass] testDefaultModule") {
		t.Errorf("expected the default module's test to run, got: %s", stdout)
	}
	if !strings.Contains(stdout, "[pass] testSubmodule") {
		t.Errorf("expected the submodule's test to run — this is the bug this test guards against, got: %s", stdout)
	}
}

// The tests below cover skip-cascading, data-provider edge cases, and
// rerun-failed edge cases — all scenarios that genuinely require a test/hook
// failure to appear in the report. corpus/bal/library/test-framework can't
// verify these: ballerina/test's own formatFailedError unconditionally
// appends a trailing tabs-only line to any failure message, which the
// corpus harness's @output annotations can never represent (they strip
// trailing whitespace when parsing) — confirmed via a standalone repro, not
// just for diff-formatted assertEquals failures. See TODO.md's "Corpus
// `@output` annotations can't represent trailing-whitespace-only lines"
// entry. Plain Go string/count comparisons here have no such limitation.

// The 4 skip-cascade scenarios (dependsOn, beforeEach, beforeSuite,
// beforeGroups failures) previously lived here as substring/count
// assertions. They now live in test_golden_test.go
// (TestTestCommand_Golden_Skip*), verified against full-fidelity golden
// output files instead — see that file's runGoldenScenario doc comment for
// why a golden-file harness can assert on the exact failure text
// (including report.bal's trailing-tabs-only lines) where corpus's
// @output annotations structurally cannot.

// TestTestCommand_DataProvider_ArgMismatchDoesNotCrash documents a confirmed,
// real bug (see TODO.md's "invokeFunction/InvokeFunctionValue don't validate
// argument types before binding" entry): a data provider whose values don't
// match its test function's actual parameter types crashes the *process*
// with a raw Go panic (`interface conversion: values.BalValue is int64, not
// string`) that escapes even `trap` — confirmed via
// runtime/internal/exec/executor.go#panicValueToErrorValue, which by design
// only converts `*values.Error` panics and deliberately re-panics anything
// else as "an unrecoverable interpreter issue". This is a real crash a user
// could hit with an ordinary data-provider authoring mistake, not just a
// missing nice-to-have error message.
//
// Run via subprocess (like testglue's TestGenerateSource_CompilesAndRuns)
// rather than executeTestCommand's in-process call: the panic is real and
// unrecovered, and letting it propagate in-process would crash this whole
// test binary and take every other test in the package down with it.
func TestTestCommand_DataProvider_ArgMismatchDoesNotCrash(t *testing.T) {
	dir := writeTestFixture(t, "ddmismatchmod", `import ballerina/test;

// Returns string tuples, but the test function expects ints — a genuine
// type mismatch between the data provider and its test, only detectable at
// runtime (both sides compile fine on their own).
function mismatchedDataSet() returns map<[string, string]> {
    return {"one": ["not", "numbers"]};
}

@test:Config {
    dataProvider: mismatchedDataSet
}
function testMismatched(int a, int b) {
    int sum = a + b;
    test:assertTrue(sum >= 0);
}
`)

	cmd := exec.Command("go", "run", ".", "test", dir)
	out, err := cmd.CombinedOutput()

	// TODO.md tracks this as a confirmed bug to fix upstream (runtime/exec,
	// out of this team's scope) — once InvokeFunctionValue validates
	// argument types and raises a proper *values.Error instead of a raw Go
	// panic, this process will exit normally (non-zero, a reported test
	// failure) instead of crashing, and this assertion should be flipped.
	if err == nil {
		t.Errorf("expected the process to exit non-zero, got success. output:\n%s", out)
	}
	if !strings.Contains(string(out), "panic:") {
		t.Logf("bug may already be fixed — process exited non-zero without the known panic signature. output:\n%s", out)
	}
}

// The data-provider single-failure and rerun-failed edge-case scenarios
// (single sub-case failure, rerun-failed with a data provider, no prior
// run, invalid JSON, wrong module key) previously lived here as
// substring/count assertions. They now live in test_golden_test.go
// (TestTestCommand_Golden_DataProviderSingleFailureOthersStillRun,
// _RerunFailed*), verified against full-fidelity golden output files
// instead — see runGoldenScenario's doc comment for why.

func TestTestCommand_TestReport_NotGeneratedWithoutFlag(t *testing.T) {
	dir := writeTestFixture(t, "noreportmod", `import ballerina/test;

@test:Config {}
function testOne() {
    test:assertTrue(true);
}
`)
	if _, _, _, err := executeTestCommand(t, dir); err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "target", "report", "test_results.json")); !os.IsNotExist(err) {
		t.Errorf("expected no test_results.json without --test-report, stat error: %v", err)
	}
}

func TestTestCommand_TestReport_SinglePackage(t *testing.T) {
	dir := writeTestFixture(t, "reportsinglemod", `import ballerina/test;

@test:Config {}
function testPass() {
    test:assertTrue(true);
}

@test:Config {}
function testFail() {
    test:assertTrue(false, "boom");
}
`)
	stdout, cobraStdout, _, err := executeTestCommand(t, dir, "--test-report")
	if err == nil {
		t.Fatalf("expected failure since one test fails. stdout: %s", stdout)
	}
	if !strings.Contains(cobraStdout, "Generating Test Report") {
		t.Errorf("expected the 'Generating Test Report' message, got: %s", cobraStdout)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, "target", "report", "test_results.json"))
	if readErr != nil {
		t.Fatalf("expected test_results.json to exist: %v", readErr)
	}
	var report packageTestResult
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("test_results.json is not valid JSON: %v\ncontent: %s", err, data)
	}
	if report.ProjectName != "reportsinglemod" {
		t.Errorf("expected projectName 'reportsinglemod', got %q", report.ProjectName)
	}
	if report.TotalTests != 2 || report.Passed != 1 || report.Failed != 1 || report.Skipped != 0 {
		t.Errorf("expected 2 total/1 passed/1 failed/0 skipped, got %+v", report)
	}
	if len(report.ModuleStatus) != 1 || report.ModuleStatus[0].Name != "reportsinglemod" {
		t.Errorf("expected exactly one module status entry named 'reportsinglemod', got %+v", report.ModuleStatus)
	}
}

// TestTestCommand_TestReport_FailureMessagePreservesNonAscii guards against
// corrupting a multi-byte UTF-8 character while building module_status.json
// (a lone byte from the middle of one isn't valid UTF-8 by itself, so a
// byte-at-a-time encoder can drop it outright — a real bug the old
// report.bal#replaceDoubleQuotes implementation once had).
//
// The exact expected string differs from that old implementation on
// purpose, in two ways testrunner.BuildModuleStatus's own doc comment
// documents as deliberate, not a regression: no double-escaped quotes
// (encoding/json.Marshal already escapes them correctly once; re-escaping
// first just corrupts the output) and no synthetic trailing "\n" (an
// artifact of getErrorMessage's own convention, kept out of the stored
// Result.Message testrunner builds this from).
func TestTestCommand_TestReport_FailureMessagePreservesNonAscii(t *testing.T) {
	dir := writeTestFixture(t, "reportnonasciimod", `import ballerina/test;

@test:Config {}
function testFail() {
    test:assertFail("café says \"hi\"");
}
`)
	if _, _, _, err := executeTestCommand(t, dir, "--test-report"); err == nil {
		t.Fatal("expected failure since the test fails")
	}

	data, readErr := os.ReadFile(filepath.Join(dir, "target", "report", "test_results.json"))
	if readErr != nil {
		t.Fatalf("expected test_results.json to exist: %v", readErr)
	}
	var report packageTestResult
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("test_results.json is not valid JSON: %v\ncontent: %s", err, data)
	}
	if len(report.ModuleStatus) != 1 || len(report.ModuleStatus[0].Tests) != 1 {
		t.Fatalf("expected exactly one module with one test entry, got %+v", report.ModuleStatus)
	}
	want := `café says "hi"`
	if got := report.ModuleStatus[0].Tests[0].FailureMessage; got != want {
		t.Errorf("failureMessage = %q, want %q", got, want)
	}
}

func TestTestCommand_TestReport_MultiModulePackage(t *testing.T) {
	dir := writeMultiModuleFixture(t)

	if _, cobraStdout, stderr, err := executeTestCommand(t, dir, "--test-report"); err != nil {
		t.Fatalf("expected success: %v\ncobraStdout: %s\nstderr: %s", err, cobraStdout, stderr)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, "target", "report", "test_results.json"))
	if readErr != nil {
		t.Fatalf("expected test_results.json to exist: %v", readErr)
	}
	var report packageTestResult
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("test_results.json is not valid JSON: %v\ncontent: %s", err, data)
	}
	if report.TotalTests != 2 || report.Passed != 2 {
		t.Errorf("expected 2 total/2 passed across both modules, got %+v", report)
	}
	if len(report.ModuleStatus) != 2 {
		t.Fatalf("expected one module status entry per module, got %+v", report.ModuleStatus)
	}
	if report.ModuleStatus[0].Name != "multimodpkg" || report.ModuleStatus[1].Name != "multimodpkg.submod" {
		t.Errorf("expected module status entries for both the default module and the submodule, got %+v", report.ModuleStatus)
	}
}

func TestTestCommand_TestReport_Workspace(t *testing.T) {
	root, _, _ := writeWorkspaceFixture(t)

	if _, cobraStdout, _, err := executeTestCommand(t, root, "--test-report"); err == nil {
		t.Fatalf("expected failure since one member fails. cobraStdout: %s", cobraStdout)
	}

	data, readErr := os.ReadFile(filepath.Join(root, "target", "report", "test_results.json"))
	if readErr != nil {
		t.Fatalf("expected a workspace-level test_results.json to exist: %v", readErr)
	}
	var report workspaceTestReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("test_results.json is not valid JSON: %v\ncontent: %s", err, data)
	}
	if report.WorkspaceName == "" {
		t.Errorf("expected a non-empty workspaceName, got %+v", report)
	}
	if report.TotalTests != 2 || report.Passed != 1 || report.Failed != 1 {
		t.Errorf("expected 2 total/1 passed/1 failed across both members, got %+v", report)
	}
	if len(report.Packages) != 2 {
		t.Fatalf("expected both members in the aggregated report, got %+v", report.Packages)
	}

	for _, member := range []string{"passing", "failing"} {
		if _, err := os.Stat(filepath.Join(root, member, "target", "report", "test_results.json")); !os.IsNotExist(err) {
			t.Errorf("expected no per-member test_results.json for %q (only the workspace-level one), stat error: %v",
				member, err)
		}
	}
}

// TestTestCommand_ModuleInitFailure locks in a real behavioral difference
// from jballerina found during the 2026-09-16 audit (TODO.md item #31):
// jballerina evaluates a module-level variable's initializer lazily, so
// `int a = 1/0;` only panics the first time something actually reads `a` —
// if that's inside a test function, it surfaces as an ordinary per-test
// failure (`[fail] testFunc:`, with the DivisionByZero error's own stack
// trace, "0 passing/1 failing"). This Go port evaluates module-level
// variables eagerly as part of runtime.Init (called once per module before
// any test runs), so the same source fails the whole run at load time —
// before any test framework code (registration, startSuite, the console
// report) ever runs. Confirmed this is a clean, caught error (not a raw Go
// panic) via runTestsForModule's existing `if err := rt.Init(...); err !=
// nil` handling — this test exists to prove that stays true, not to make
// the two ports match (that would require lazy module-variable
// initialization semantics, a much larger, unrelated change).
func TestTestCommand_ModuleInitFailure(t *testing.T) {
	dir := writeTestFixture(t, "initfailmod", `import ballerina/test;

int a = 1 / 0;

@test:Config {}
function testFunc() {
    test:assertEquals(a, 0);
}
`)
	stdout, _, cobraStderr, err := executeTestCommand(t, dir)
	if err == nil {
		t.Fatalf("expected a module-init failure to fail the command. stdout: %s", stdout)
	}
	if !strings.Contains(cobraStderr, "divide by zero") {
		t.Errorf("expected a clear divide-by-zero error, got stdout: %s\ncobraStderr: %s", stdout, cobraStderr)
	}
	if strings.Contains(stdout, "passing") || strings.Contains(stdout, "failing") {
		t.Errorf("no test report should print at all — init failure aborts before any test runs, got: %s", stdout)
	}
}

// TestTestCommand_SourcelessDefaultModule_RunsSubmoduleTests closes TODO.md
// item #32 (2026-09-16 audit): a package whose default module has NO source
// file at all — not even main.bal, just Ballerina.toml — and whose only
// content lives in two submodules that themselves have only test files
// (mirrors jballerina's SourcelessTestExecutionTests). Confirmed via a real
// repro before writing this that this already compiles and runs correctly
// (no load error from the missing main.bal, no spurious "No tests found"
// for the sourceless default module — it's simply absent from
// pkg.Modules() entirely, so the multi-module loop never sees it).
func TestTestCommand_SourcelessDefaultModule_RunsSubmoduleTests(t *testing.T) {
	dir := t.TempDir()
	toml := "[package]\norg = \"testorg\"\nname = \"sourcelessmods\"\nversion = \"0.1.0\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte(toml), 0o644); err != nil {
		t.Fatalf("write Ballerina.toml: %v", err)
	}

	for _, mod := range []struct{ name, testFn string }{
		{"module1", "test1"},
		{"module2", "test2"},
	} {
		testsDir := filepath.Join(dir, "modules", mod.name, "tests")
		if err := os.MkdirAll(testsDir, 0o755); err != nil {
			t.Fatalf("mkdir modules/%s/tests: %v", mod.name, err)
		}
		source := "import ballerina/test;\n\n@test:Config {}\nfunction " + mod.testFn + "() {\n" +
			"    test:assertTrue(true);\n}\n"
		if err := os.WriteFile(filepath.Join(testsDir, mod.name+"_test.bal"), []byte(source), 0o644); err != nil {
			t.Fatalf("write modules/%s test file: %v", mod.name, err)
		}
	}

	stdout, cobraStdout, stderr, err := executeTestCommand(t, dir)
	if err != nil {
		t.Fatalf("expected success, got error: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(cobraStdout, "sourcelessmods.module1") || !strings.Contains(cobraStdout, "sourcelessmods.module2") {
		t.Errorf("expected headers for both submodules, got: %s", cobraStdout)
	}
	if !strings.Contains(stdout, "[pass] test1") || !strings.Contains(stdout, "[pass] test2") {
		t.Errorf("expected both submodules' tests to run, got: %s", stdout)
	}
}

// TestTestCommand_ModuleQualifiedTestsFilter locks in a real bug fix found
// while closing out the lower-priority items from the 2026-09-16 audit:
// `--tests <package>.<module>:<name>` (the fully-qualified form) never
// matched anything, because filter.bal#getFullModuleName re-concatenated
// the package name onto testOptions.getModuleName() — which is already
// fully-qualified in this port's design (cli/cmd/test.go always passes
// module.ModuleName().String(), e.g. "pkg.submod", not a bare module-name
// part like jballerina's own TestOptions stores) — producing a doubled
// "pkg.pkg.submod" that never matched the filter the user actually typed.
// Confirmed via a real repro before fixing; this test covers both the
// bare-name (unqualified) and fully-qualified forms so a regression in
// either direction would be caught.
func TestTestCommand_ModuleQualifiedTestsFilter(t *testing.T) {
	dir := t.TempDir()
	toml := "[package]\norg = \"testorg\"\nname = \"modquantestmod\"\nversion = \"0.1.0\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte(toml), 0o644); err != nil {
		t.Fatalf("write Ballerina.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.bal"), []byte("public function main() {\n}\n"), 0o644); err != nil {
		t.Fatalf("write main.bal: %v", err)
	}
	submodTestsDir := filepath.Join(dir, "modules", "submod", "tests")
	if err := os.MkdirAll(submodTestsDir, 0o755); err != nil {
		t.Fatalf("mkdir modules/submod/tests: %v", err)
	}
	submodSource := `import ballerina/test;

@test:Config {}
function testInSubmod() {
    test:assertTrue(true);
}
`
	if err := os.WriteFile(filepath.Join(submodTestsDir, "submod_test.bal"), []byte(submodSource), 0o644); err != nil {
		t.Fatalf("write modules/submod/tests/submod_test.bal: %v", err)
	}

	for _, filter := range []string{"testInSubmod", "modquantestmod.submod:testInSubmod"} {
		stdout, _, stderr, err := executeTestCommand(t, dir, "--tests", filter)
		if err != nil {
			t.Fatalf("--tests %q: expected success, got error: %v\nstdout: %s\nstderr: %s", filter, err, stdout, stderr)
		}
		if !strings.Contains(stdout, "[pass] testInSubmod") {
			t.Errorf("--tests %q: expected testInSubmod to run, got: %s", filter, stdout)
		}
	}
}

// TestTestCommand_GracefullyStopsListeners closes TODO.md bug #23: a package
// that declares a module-level listener previously had it started (via
// rt.Listen()) but never gracefully stopped once the suite finished, so a
// runtime:onGracefulStop handler's side effect was never observed. Confirmed
// via a real repro before this fix that the handler's print never appeared
// (the process still exited cleanly — no hang — it just skipped teardown
// entirely). Fixed via runtime.Runtime.RequestGracefulStop, called from
// runTestsForModule right after startSuite() returns.
func TestTestCommand_GracefullyStopsListeners(t *testing.T) {
	dir := writeTestFixture(t, "listenerstopmod", `import ballerina/test;

@test:Config {}
function testSomething() {
    test:assertTrue(true);
}
`)
	mainSource := `import ballerina/http;
import ballerina/lang.runtime;
import ballerina/io;

function onStop() returns error? {
    io:println("GRACEFUL_STOP_HANDLER_RAN");
}

function init() {
    runtime:onGracefulStop(onStop);
}

service /echo on new http:Listener(20292) {
    resource function get hello() returns string {
        return "hi";
    }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.bal"), []byte(mainSource), 0o644); err != nil {
		t.Fatalf("overwrite main.bal: %v", err)
	}

	stdout, _, stderr, err := executeTestCommand(t, dir)
	if err != nil {
		t.Fatalf("expected success, got error: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "[pass] testSomething") {
		t.Errorf("expected the test to run, got: %s", stdout)
	}
	if !strings.Contains(stdout, "GRACEFUL_STOP_HANDLER_RAN") {
		t.Errorf("expected the onGracefulStop handler to run after the suite finished, got: %s", stdout)
	}
}
