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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/test/0.0.1/go1.27/native"
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

// executeTestCommand runs the test command, returning cobra's own
// stdout/stderr buffers plus the real OS-level stdout the console report is
// written to directly. Not safe to run in parallel with anything else that
// reads/writes os.Stdout — none of these tests call t.Parallel().
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

// writeWorkspaceFixture creates a workspace with 2 members: "passing" (one
// passing test) and "failing" (one failing test).
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

// writeMultiModuleFixture creates a package with tests in both its default
// module and a named submodule under modules/.
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

// Skip-cascade and data-provider/rerun-failed edge cases that need an exact
// failure message (including its trailing-tabs-only line) live as corpus
// fixtures under corpus/cli/testdata/test/{projects,flags,steps}, not here.

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
// corrupting a multi-byte UTF-8 character while building module_status.json.
// See testrunner.BuildModuleStatus's doc comment for why the exact message
// has no double-escaped quotes or synthetic trailing newline.
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

// ---------------------------------------------------------------------------
// GroupingTest: most scenarios now live as corpus cases under
// corpus/cli/testdata/test/{projects,flags}/grouping*. Only the 2
// standalone-file cases below stay here (no file-based corpus driver yet).
// ---------------------------------------------------------------------------

// groupingBalSource is groups-test.bal's content, shared by the 2
// standalone-file GroupingTest listing tests below.
const groupingBalSource = `import ballerina/test;

@test:Config{
    groups:["g1","g2"]
}
function testFunc1 () {
    test:assertFalse(false, msg = "errorMessage");
}

@test:Config{
    groups:["g1","g2","g3"]
}
function testFunc2 () {
    test:assertFalse(false, msg = "errorMessage");
}

@test:Config{
    groups:["g1","g2","g3","g4"]
}
function testFunc3 () {
    test:assertFalse(false, msg = "errorMessage");
}

@test:Config{
    groups:["g5"]
}
function testFunc4 () {
    test:assertFalse(true, msg = "errorMessage");
}

@test:Config{
    groups:["g6"]
}
function testFunc5 () {
    test:assertFalse(true, msg = "errorMessage");
}

@test:Config{}
function testFunc6 () {
    test:assertFalse(false, msg = "errorMessage");
}
`

// TestTestCommand_Grouping_ListingOfSingleTestGroups matches jballerina's
// testListingOfSingleTestGroups: --list-groups lists every discovered
// group.
func TestTestCommand_Grouping_ListingOfSingleTestGroups(t *testing.T) {
	dir := t.TempDir()
	balFile := writeStandaloneFixture(t, dir, "groups-test.bal", groupingBalSource)
	stdout, _, stderr, err := executeTestCommand(t, balFile, "--list-groups")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "[g1, g2, g3, g4, g5, g6]") {
		t.Errorf("expected the full group list, got: %s", stdout)
	}
}

// TestTestCommand_Grouping_ListGroupsWithOtherFlags matches jballerina's
// testListGroupsWithOtherFlags: --groups g1 --list-groups still lists every
// group and prints the "other flags skipped" warning.
func TestTestCommand_Grouping_ListGroupsWithOtherFlags(t *testing.T) {
	dir := t.TempDir()
	balFile := writeStandaloneFixture(t, dir, "groups-test.bal", groupingBalSource)
	stdout, _, stderr, err := executeTestCommand(t, balFile, "--groups", "g1", "--list-groups")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Warning: Other flags are skipped when list-groups flag is provided.") {
		t.Errorf("expected the other-flags-skipped warning, got: %s", stdout)
	}
	if !strings.Contains(stdout, "[g1, g2, g3, g4, g5, g6]") {
		t.Errorf("expected the full, unfiltered group list despite --groups g1, got: %s", stdout)
	}
}

// ---------------------------------------------------------------------------
// RerunFailedTest: most scenarios now live as corpus cases under
// corpus/cli/testdata/test/{flags,steps}/rerun-failed-*. Only FullTest below
// stays here (parses rerun_test.json's structured content).
// ---------------------------------------------------------------------------

// writeRerunFailedFixture is a direct port of jballerina's
// rerun-failed-tests/tests/main_test.bal: 2 always-passing and 2
// always-failing test functions, independent of each other.
func writeRerunFailedFixture(t *testing.T) string {
	t.Helper()
	return writeTestFixture(t, "rerun_failed", `import ballerina/test;

@test:Config {}
function testFunctionPass1() {
    test:assertTrue(true, msg = "Failed!");
}

@test:Config {}
function testFunctionPass2() {
    test:assertTrue(true, msg = "Failed!");
}

@test:Config {}
function testFunctionFail1() {
    test:assertTrue(false, msg = "Failed!");
}

@test:Config {}
function testFunctionFail2() {
    test:assertTrue(false, msg = "Failed!");
}
`)
}

// TestTestCommand_RerunFailed_FullTest matches jballerina's testFullTest: a
// full run with 2 passing/2 failing writes rerun_test.json with the 2
// failures.
func TestTestCommand_RerunFailed_FullTest(t *testing.T) {
	dir := writeRerunFailedFixture(t)
	stdout, _, _, err := executeTestCommand(t, dir)
	if err == nil {
		t.Fatalf("expected a non-nil error for a failing suite. stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "2 passing") || !strings.Contains(stdout, "2 failing") || !strings.Contains(stdout, "0 skipped") {
		t.Errorf("expected 2 passing/2 failing/0 skipped, got: %s", stdout)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, "target", "rerun_test.json"))
	if readErr != nil {
		t.Fatalf("expected rerun_test.json to be written: %v", readErr)
	}
	var rerun map[string]struct {
		TestNames []string `json:"testNames"`
	}
	if err := json.Unmarshal(data, &rerun); err != nil {
		t.Fatalf("unmarshal rerun_test.json: %v", err)
	}
	entry, ok := rerun["rerun_failed"]
	if !ok {
		t.Fatalf("expected a rerun_failed entry, got: %+v", rerun)
	}
	gotNames := map[string]bool{}
	for _, n := range entry.TestNames {
		gotNames[n] = true
	}
	if len(gotNames) != 2 || !gotNames["testFunctionFail1"] || !gotNames["testFunctionFail2"] {
		t.Errorf("expected rerun_test.json to list exactly testFunctionFail1/testFunctionFail2, got: %+v", entry.TestNames)
	}
}

// ---------------------------------------------------------------------------
// DataProviderTest: most scenarios now live as corpus cases under
// corpus/cli/testdata/test/flags/data-providers. Only the table-driven
// CodeFragmentKeys[WithWildCard] cases below stay here.
// ---------------------------------------------------------------------------

// dataProviderMainTestSource is a direct, line-for-line port of
// data-providers/tests/new-data-provider-tests.bal.
const dataProviderMainTestSource = `// Copyright (c) 2021 WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
//
// WSO2 Inc. licenses this file to you under the Apache License,
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

import ballerina/test;

type CodeFragment [string, string];

@test:Config {
    dataProvider: dataGen
}
function intDataProviderTest(int value1, int value2, int result1) returns error? {
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
}

@test:Config {
    dataProvider: dataGen2
}
function fruitsDataProviderTest(int value1, int value2, string fruit) returns error? {
    test:assertEquals(value1, value2, msg = "The sum is not correct");
    test:assertEquals(fruit.length(), 6);
}

@test:Config{
    dataProvider: dataGen3
}
function jsonDataProviderTest (json json1, json json2, json json3) {
    json a = {"a": "a"};
    json b = {"b": "b"};
    json c = {"c": "c"};
    test:assertEquals(json1, a, msg = "json data provider failed");
    test:assertEquals(json2, b, msg = "json data provider failed");
    test:assertEquals(json3, c, msg = "json data provider failed");
}

@test:Config {
    dataProvider: dataGen4
}
function errorDataProviderTest(int value1, int value2, int result1) returns error? {
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
}

@test:Config {
    dataProvider: dataGen6
}
function testFunction1(int value1, int value2, int result1) returns error? {
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
}

@test:Config {
    dataProvider: dataGen7
}
function intArrayDataProviderTest(int value1, int value2, int result1) returns error? {
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
}

@test:Config {
    dataProvider: dataGen8
}
function testFunction2(int value1, int value2, int result1) returns error? {
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
}

@test:Config {
    dataProvider: dataGen9
}
function testFunction3(string value1, string value2) returns error? {
   string _ = value2;
   test:assertEquals("E", value1, msg = "The code fragment is not correct.");
}

string tally = "";  // used to track which function has been executed

function beforeFunction() {
    tally += "b";
}

function afterFunction() {
    tally += "a";
}

@test:Config {
    dataProvider:  dataGen10,
     before:  beforeFunction,
    after:  afterFunction
}
function testDividingValues(string fValue, string sValue, string result) returns error? {
    tally += "f";
    
    int value1 = check 'int:fromString(fValue);
    int value2 = check 'int:fromString(sValue);
    int result1 = check 'int:fromString(result);

    test:assertEquals(value1/value2, result1, msg = "Incorrect Division");
}

@test:Config {
    dependsOn: [testDividingValues]
}
function testExecutionOfBeforeAfter() {
    test:assertEquals(tally, "bfabfabfabfabfa");
}

function beforeFailsFunction() {
    tally += "b";

    // Condition for a failure
    if (tally == "bfabfab") {
        // Something happens during the 3rd iteration
        int _ = 9/0;
    }
}

@test:Config {
    dataProvider:  dataGen10,
    before:  beforeFailsFunction,
    after:  afterFunction
}
function testDividingValuesWithBeforeFailing(string fValue, string sValue, string result) returns error? {
    tally += "f";
    
    int value1 = check 'int:fromString(fValue);
    int value2 = check 'int:fromString(sValue);
    int result1 = check 'int:fromString(result);

    test:assertEquals(value1/value2, result1, msg = "Incorrect Division");
}

// Depends on testDividingValuesWithBeforeFailing
// However since it fails in one instance, dependsOn will not work
@test:Config {}
function testExecutionOfBeforeFailing() {
    test:assertEquals(tally, "bfabfabbfabfa");
}

function afterFailsFunction() {
    tally += "a";

    // Condition for a failure
    if (tally == "bfabfabfa") {
        // Something happens during the 4th iteration
        int _ = 9/0;
    }
}

@test:Config {
    dataProvider:  dataGen10,
    before:  beforeFunction,
    after:  afterFailsFunction
}
function testDividingValuesWithAfterFailing(string fValue, string sValue, string result) returns error? {
    tally += "f";
    
    int value1 = check 'int:fromString(fValue);
    int value2 = check 'int:fromString(sValue);
    int result1 = check 'int:fromString(result);

    test:assertEquals(value1/value2, result1, msg = "Incorrect Division");
}

// Depends on testDividingValuesWithAfterFailing
// However since it fails in one instance, dependsOn will not work
@test:Config {}
function testExecutionOfAfterFailing() {
    test:assertEquals(tally, "bfabfabfabfabfa");
}

@test:Config {
    dataProvider:  dataGen11,
    before:  beforeFunction,
    after:  afterFunction
}
function testDividingValuesNegative(string fValue, string sValue, string result) returns error? {
    tally += "f";
    
    int value1 = check 'int:fromString(fValue);
    int value2 = check 'int:fromString(sValue);
    int result1 = check 'int:fromString(result);

    test:assertEquals(value1/value2, result1, msg = "Incorrect Division");
}

@test:Config {
    dependsOn: [testDividingValuesNegative]
}
function testExecutionOfDataValueFailing() {
    test:assertEquals(tally, "bfabfabfabfabfa");
}

@test:Config {
    dataProvider:  dataGen12
}
function errorData(error input, string expected) {
    final string actual = getFunction(input);
    test:assertEquals(actual, expected);
}

@test:Config {
    dataProvider: dataGen13
}
function mapOfTupleOfFunctionTest(function (int x) returns boolean func, int value) {
    test:assertTrue(func(value));
}

@test:Config {
    dataProvider: dataGen14
}
function arrayOfArrayOfFunctionTest(function (int x) returns boolean func1, function (int x) returns boolean func2) {
    int value = 2;
    test:assertTrue(func1(value) || func2(value));
}

// Data Providers

function dataGen() returns map<[int, int, int]>|error {
    map<[int, int, int]> dataSet = {
        "Case1": [1, 2, 4],
        "Case2": [10, 20, 31],
        "Case3": [5, 6, 11]
    };
    return dataSet;
}

function dataGen2() returns map<[int, int, string]>|error {
    map<[int, int, string]> dataSet = {
        "banana": [10, 10, "banana"],
        "cherry": [5, 5, "cherry"]
    };
    return dataSet;
}

function dataGen3() returns map<[json,json, json]> {
    map<[json, json, json]> dataSet = {
        "json1": [{"a": "a"}, {"b": "b"}, {"c": "c"}],
        "json2": [{"a": "a"}, {"b": "b"}, {"c": "c"}]
    };
    return dataSet;
}

function dataGen4() returns map<[string]>|error{
    return error("Error occurred while generating data set.");
}

function dataGen6() returns map<[int, int, int]>|error {
    map<[int, int, int]> dataSet = {
        "CaseNew1": [1, 2, 3],
        "CaseNew2": [10, 20, 30],
        "Case3": [5, 6, 11]
    };
    return dataSet;
}

function dataGen7() returns int[][] {
    return [[1, 2, 4], [10, 20, 31], [5, 6, 11]];
}

function dataGen8() returns map<[int, int, int]>|error {
    map<[int, int, int]> dataSet = {
        "Case#1": [1, 2, 3],
        "Case#2": [10, 20, 30],
        "Case#3": [5, 6, 11]
    };
    return dataSet;
}

function dataGen9() returns map<CodeFragment>|error {
    CodeFragment[] sources = [
        ["E", "` +
	"`" +
	`'" +  string` +
	"`" +
	`"\a"` +
	"`" +
	` + string` +
	"`" +
	`"` +
	"`" +
	`],
        ["E", "\"\\" + "u{D7FF}\"" + "\"\t\""],
        ["E",  "a +\n\r b"],
        ["E", "(x * 1) != (y / 3) || (a ^ b) == (b & c) >> (1 % 2)"],
        ["E",  "(1"],
        ["E", "a:x(c,d)[]; ^(x|y).ok();"],
        ["E",  string` +
	"`" +
	`map<any> v = { "x": 1 };` +
	"`" +
	`]];

    map<CodeFragment> tests = {};
    foreach var s in sources {
        tests[s[1]] = s;
    }
    return tests;
}

function dataGen10() returns (string[][]) {
    return [["10", "2", "5"], ["10", "1", "10"], ["10", "2", "5"], ["10", "1", "10"], ["10", "2", "5"]];
}

function dataGen11() returns (string[][]) {
    return [["10", "2", "5"], ["10", "1", "10"], ["10", "0", "5"], ["10", "1", "10"], ["10", "2", "5"]];
}

function dataGen12() returns map<[error, string]> {
    error e = error("foo");

    return {
        "foo": [e, "foo"]
    };
}

function dataGen13() returns map<[function, int]> {
    map<[function, int]> dataSet = {
        "1": [isOdd, 1],
        "2": [isEven, 2],
        "3": [isOdd, 3],
        "4": [isEven, 4]
    };
    return dataSet;
}

function dataGen14() returns function[][] {
    function[][] dataSet = [[isOdd, isEven], [isEven, isOdd], [isEven, isEven]];
    return dataSet;
}

function isEven(int x) returns boolean {
    return x % 2 == 0;
}

function isOdd(int x) returns boolean {
    return x % 2 != 0;
}

public function getFunction(error e) returns (string) {
    return e.message();
}

type Feed record {
    int responseCode;
    string message;
};

@test:Config{ dataProvider:getStateResponseDataProvider }
function testGetState(Feed dataFeed) {
    test:assertEquals(200, dataFeed.responseCode);
    test:assertEquals("Hello World!!!", dataFeed.message);
}

function getStateResponseDataProvider() returns Feed[][] {
     return [
            [{responseCode:200, message:"Hello World!!!"}],
            [{responseCode:20, message:"Hello World!!!"}]
     ];
}
`

// dataProviderModule1TestSource is a direct port of
// data-providers/modules/module1/tests/module1_test.bal.
const dataProviderModule1TestSource = `// Copyright (c) 2021 WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
//
// WSO2 Inc. licenses this file to you under the Apache License,
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

import ballerina/test;

@test:Config{
    dataProvider: dataGenMod1
}
function stringDataProviderMod1Test (string fValue, string sValue, string result) returns error? {

    int|error val1 = fValue.cloneWithType(int);
    int value1 = val1 is int ? val1 : 0;
    int|error val2 = sValue.cloneWithType(int);
    var value2 = val2 is int ? val2 : 0;
    int|error res1 = result.cloneWithType(int);
    var result1 = res1 is int ? res1 : 0;
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
    return;
}

function dataGenMod1() returns string[][] | error{
    return [["1", "2", "3"], ["10", "20", "30"], ["5", "6", "11"]];
}
`

// writeDataProviderFixture ports jballerina's data-providers project: a
// default module plus a module1 submodule.
func writeDataProviderFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	toml := "[package]\norg = \"testorg\"\nname = \"dataproviders\"\nversion = \"0.1.0\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte(toml), 0o644); err != nil {
		t.Fatalf("write Ballerina.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.bal"), []byte("public function main() {\n}\n"), 0o644); err != nil {
		t.Fatalf("write main.bal: %v", err)
	}
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("mkdir tests: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "new-data-provider-tests.bal"), []byte(dataProviderMainTestSource), 0o644); err != nil {
		t.Fatalf("write tests/new-data-provider-tests.bal: %v", err)
	}
	mod1TestsDir := filepath.Join(dir, "modules", "module1", "tests")
	if err := os.MkdirAll(mod1TestsDir, 0o755); err != nil {
		t.Fatalf("mkdir modules/module1/tests: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mod1TestsDir, "module1_test.bal"), []byte(dataProviderModule1TestSource), 0o644); err != nil {
		t.Fatalf("write modules/module1/tests/module1_test.bal: %v", err)
	}
	return dir
}

// codeFragmentKeys are testFunction3/dataGen9's 7 map keys, a byte-for-byte
// port of dataGen9.
var codeFragmentKeys = []string{
	"`'\"\\a\"\"",
	"\"\\u{D7FF}\"\"\t\"",
	"a +\n\r b",
	`(x * 1) != (y / 3) || (a ^ b) == (b & c) >> (1 % 2)`,
	`(1`,
	`a:x(c,d)[]; ^(x|y).ok();`,
	`map<any> v = { "x": 1 };`,
}

// codeFragmentKeysWithWildCard are the same 7 keys with one character each
// replaced by a '*' wildcard.
var codeFragmentKeysWithWildCard = []string{
	"`'\"\\*\"\"",
	"\"\\u{D7FF}\"*\t\"",
	"a +\n* b",
	`(x * 1) *= (y / 3) || (a ^ b) == (b & c) >> (1 % 2)`,
	`*1`,
	`a:x(c*d)[]; ^(x|y).ok();`,
	`map<any> v = { *x": 1 };`,
}

// TestTestCommand_DataProvider_CodeFragmentKeys matches jballerina's
// testCodeFragmentKeys (7 sub-cases): special-character keys round-trip
// through a percent-encoded, single-quoted --tests filter.
func TestTestCommand_DataProvider_CodeFragmentKeys(t *testing.T) {
	for i, key := range codeFragmentKeys {
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			dir := writeDataProviderFixture(t)
			arg := "testFunction3#'" + native.EscapeSpecialCharacters(key) + "'"
			stdout, _, stderr, err := executeTestCommand(t, dir, "--tests", arg)
			if err != nil {
				t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
			}
			if !strings.Contains(stdout, "1 passing") || !strings.Contains(stdout, "0 failing") || !strings.Contains(stdout, "0 skipped") {
				t.Errorf("expected 1 passing/0 failing/0 skipped, got: %s", stdout)
			}
		})
	}
}

// TestTestCommand_DataProvider_CodeFragmentKeysWithWildCard matches
// jballerina's testCodeFragmentKeysWithWildCard (7 sub-cases): same, with a
// '*' wildcard in each key.
func TestTestCommand_DataProvider_CodeFragmentKeysWithWildCard(t *testing.T) {
	for i, key := range codeFragmentKeysWithWildCard {
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			dir := writeDataProviderFixture(t)
			arg := "testFunction3#'" + native.EscapeSpecialCharacters(key) + "'"
			stdout, _, stderr, err := executeTestCommand(t, dir, "--tests", arg)
			if err != nil {
				t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
			}
			if !strings.Contains(stdout, "1 passing") || !strings.Contains(stdout, "0 failing") || !strings.Contains(stdout, "0 skipped") {
				t.Errorf("expected 1 passing/0 failing/0 skipped, got: %s", stdout)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// negative/InvalidDataProviderTestCase (jballerina
// single-file-tests/invalid-data-providers)
//
// Only testEmptyDataProvider is ported here. The other 3 (argument-type
// mismatch between a data provider and its test function) were GAP-BLOCKED
// on a panic that crashed the process — now stale: testrunner.invoke's
// generic recover() already turns this into a clean [fail], confirmed by
// corpus/cli/testdata/test/projects/data-provider-arg-mismatch-does-not-crash.
// Not re-triaged for porting here (separate scope).
// ---------------------------------------------------------------------------

// TestTestCommand_InvalidDataProvider_EmptyDataProvider matches
// jballerina's testEmptyDataProvider: a dataProvider whose return type
// doesn't match the expected shape is a compile-time error.
func TestTestCommand_InvalidDataProvider_EmptyDataProvider(t *testing.T) {
	dir := t.TempDir()
	balFile := writeStandaloneFixture(t, dir, "empty-data-provider-test.bal", `import ballerina/test;

@test:Config {
    dataProvider: provider
}
function test(int i) returns error? {
    int _ = i;
}

function provider() returns map<[int]> {
    return {};
}

@test:Config {
    dataProvider: provider2
}
function test2(int i) returns error? {
    int _ = i;
}

function provider2() returns int[]{
    return [];
}
`)
	stdout, _, stderr, err := executeTestCommand(t, balFile)
	if err == nil {
		t.Fatalf("expected a compile-time error for provider2's mismatched return type. stdout: %s", stdout)
	}
	if !strings.Contains(stderr, "incompatible type") {
		t.Errorf("expected an incompatible-type compile error, got stderr: %s", stderr)
	}
}

// TestTestCommand_BeforeHookNamedInitNotFound pins a real divergence: a
// before:/after: hook named "init" isn't found by runtime.LookupFunction,
// even though jballerina treats it as an ordinary function reference. Root
// cause: this port compiles a user init() into the SAME bir.BIRFunction
// that also runs the module's global-var initializers, so registering it
// for by-name lookup (tried, then reverted) would silently re-run those
// initializers too, corrupting unrelated global state — confirmed via a
// live repro. Properly fixing this needs separating the two as distinct
// bir.BIRFunctions in desugar/birgen; out of scope here.
func TestTestCommand_BeforeHookNamedInitNotFound(t *testing.T) {
	dir := writeTestFixture(t, "inithookmod", `import ballerina/test;

int i = 0;

function init() {
    i = 1;
}

@test:Config {
    before: init
}
function testBefore() {
    test:assertTrue(i == 1, msg = "Expected i to be 1, but i = " + i.toString());
}
`)
	stdout, _, _, err := executeTestCommand(t, dir)
	if err == nil {
		t.Errorf("this port does not support a before:/after: hook named \"init\" — "+
			"if this now passes, the underlying divergence has been fixed (see this test's "+
			"doc comment) and this test (plus BasicCasesTest#testAnnotations) should be ported "+
			"for real. stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "function init not found") {
		t.Errorf(`expected the specific "function init not found" failure this divergence produces today, got: %s`, stdout)
	}
}
