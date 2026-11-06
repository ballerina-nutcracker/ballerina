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

// bal test corpus driver: three shapes, mirroring testBalRunCorpus's own
// auto-discovery pattern but for `bal test`. Supports -update like the rest
// of this package.
package corpus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/test_util"
	"golang.org/x/tools/txtar"
)

// sandboxedTestdataPathPattern matches an absolute path (POSIX or Windows,
// with an optional drive letter) into the per-subtest sandbox copy of
// corpus/cli/testdata that runCLICommandWithEnv's sandboxCLICommandArgs
// (cli_integration_test.go) creates under a fresh t.TempDir() whenever a
// CLI arg points into testdata (e.g. a standalone .bal file passed directly
// as the test target). That temp dir's name is random per run, so unlike a
// plain repo-root path it can't be normalized via normalizePaths(s, repoRoot).
var sandboxedTestdataPathPattern = regexp.MustCompile(`(?:[A-Za-z]:)?[/\\]\S*?[/\\]corpus[/\\]cli[/\\]testdata`)

// rootPathSuffixPattern matches a normalized "<ROOT>..." path segment so its
// separators can be canonicalized to "/" — the CLI builds paths with
// filepath.Join, so a golden generated on macOS/Linux embeds "/" after
// "<ROOT>" while the same run on Windows embeds "\", even once the random
// root prefix itself is stripped.
var rootPathSuffixPattern = regexp.MustCompile(`<ROOT>[^\s"]*`)

// normalizeBalTestOutput masks the wall-clock "Test execution time" trailer
// (via corpus/integration_test.go's shared testExecutionTimePattern),
// repo-root paths, sandboxed-testdata paths, and any extraRoots (e.g. a
// case's own explicit t.TempDir() sandbox, whose random path a message like
// "Generating Test Report" can print verbatim), so fixtures stay
// deterministic across platforms.
func normalizeBalTestOutput(s, repoRoot string, extraRoots ...string) string {
	s = normalizePaths(test_util.NormalizeNewlines(s), repoRoot)
	for _, root := range extraRoots {
		s = normalizePaths(s, root)
	}
	s = sandboxedTestdataPathPattern.ReplaceAllString(s, "<ROOT>/corpus/cli/testdata")
	s = rootPathSuffixPattern.ReplaceAllStringFunc(s, filepath.ToSlash)
	return testExecutionTimePattern.ReplaceAllString(s, "Test execution time : <DURATION>s")
}

// assertBalTestMatchesTxtar runs `bal <args...>` and compares stdout,
// stderr and exit code against expectedPath (-update regenerates it).
// extraRoots are passed through to normalizeBalTestOutput.
func assertBalTestMatchesTxtar(t *testing.T, cli *balCLI, args []string, expectedPath string, extraRoots ...string) {
	t.Helper()
	stdout, stderr, exitCode := runCLICommand(t, cli.relBin, cli.repoRoot, cli.coverDir, args...)
	actualStdout := normalizeBalTestOutput(stdout, cli.repoRoot, extraRoots...)
	actualStderr := normalizeBalTestOutput(stderr, cli.repoRoot, extraRoots...)
	actualExit := strconv.Itoa(exitCode)

	if *update {
		files := append(test_util.TxtarFilesStdoutStderr(actualStdout, actualStderr),
			txtar.File{Name: "exitcode", Data: []byte(actualExit)})
		if test_util.UpdateTxtarArchiveIfNeeded(t, expectedPath, files) {
			t.Logf("updated %s", expectedPath)
		}
		return
	}

	expectedStdout, expectedStderr, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(expectedPath)
	if err != nil {
		t.Fatalf("failed to parse txtar file %s: %v", expectedPath, err)
	}
	if actualStdout != expectedStdout || actualStderr != expectedStderr || actualExit != expectedExit {
		t.Fatalf("mismatch for %q (expected file %s)\n%s\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), expectedPath,
			test_util.FormatExpectedGot(expectedExit, actualExit), actualStdout, actualStderr)
	}
}

// testBalTestCorpus auto-discovers plain (no-flag) `bal test` project
// fixtures under testdata/test/projects/<name>/ and compares against
// output/test/projects/<name>.txtar — no new Go code needed per case.
func testBalTestCorpus(t *testing.T, cli *balCLI) {
	testDataRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "testdata", "test", "projects")
	outputsRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "output", "test", "projects")

	entries, err := os.ReadDir(testDataRoot)
	if err != nil {
		t.Fatalf("failed to read %s: %v", testDataRoot, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertBalTestMatchesTxtar(t, cli, []string{"test", filepath.Join(testDataRoot, name)},
				filepath.Join(outputsRoot, name+".txtar"))
		})
	}
}

// balTestFileCheck asserts about one file a `bal test` run leaves on disk,
// read from the (possibly sandboxed) fixture copy after the stdout/stderr/
// exitcode txtar comparison has already run. path is relative to the
// fixture's own base directory: the fixture dir itself for a directory
// fixture, or the directory CONTAINING the .bal file for a standalone-file
// fixture (target/ lands next to the file, not "inside" it).
type balTestFileCheck struct {
	path string
	// wantAbsent asserts the file does NOT exist; mutually exclusive with
	// check.
	wantAbsent bool
	// check, when wantAbsent is false, receives the file's content for
	// assertions (e.g. unmarshal and verify specific fields).
	check func(t *testing.T, data []byte)
}

// assertBalTestFileCheck resolves fc.path against baseDir and asserts on it
// (existence/absence, then fc.check against its content).
func assertBalTestFileCheck(t *testing.T, baseDir string, fc balTestFileCheck) {
	t.Helper()
	path := filepath.Join(baseDir, fc.path)
	data, err := os.ReadFile(path)
	if fc.wantAbsent {
		if err == nil {
			t.Errorf("expected %s to not exist, but it does", fc.path)
		} else if !os.IsNotExist(err) {
			t.Errorf("unexpected error reading %s: %v", fc.path, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("expected %s to exist: %v", fc.path, err)
	}
	if fc.check != nil {
		fc.check(t, data)
	}
}

// copyFileIntoSandbox copies a single standalone fixture file (copyDir only
// handles directories) into dstDir under its own base name, returning the
// copy's path.
func copyFileIntoSandbox(t *testing.T, src, dstDir string) string {
	t.Helper()
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dstDir, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", src, err)
	}
	dst := filepath.Join(dstDir, filepath.Base(src))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", dst, err)
	}
	return dst
}

// corpusTestStatusEntry/corpusModuleTestResult/corpusPackageTestResult/
// corpusWorkspaceTestReport mirror cli/cmd/testreport.go's
// testStatusEntryJSON/moduleTestResult/packageTestResult/
// workspaceTestReport (unexported there, in a different package, so
// re-declared here for JSON assertions) — kept in sync with that file's
// field tags, not derived from it.
type corpusTestStatusEntry struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	FailureMessage string `json:"failureMessage,omitempty"`
}

type corpusModuleTestResult struct {
	Name       string                  `json:"name"`
	TotalTests int                     `json:"totalTests"`
	Passed     int                     `json:"passed"`
	Failed     int                     `json:"failed"`
	Skipped    int                     `json:"skipped"`
	Tests      []corpusTestStatusEntry `json:"tests"`
}

type corpusPackageTestResult struct {
	ProjectName  string                   `json:"projectName"`
	TotalTests   int                      `json:"totalTests"`
	Passed       int                      `json:"passed"`
	Failed       int                      `json:"failed"`
	Skipped      int                      `json:"skipped"`
	ModuleStatus []corpusModuleTestResult `json:"moduleStatus"`
}

type corpusWorkspaceTestReport struct {
	WorkspaceName string                    `json:"workspaceName"`
	TotalTests    int                       `json:"totalTests"`
	Passed        int                       `json:"passed"`
	Failed        int                       `json:"failed"`
	Skipped       int                       `json:"skipped"`
	Packages      []corpusPackageTestResult `json:"packages"`
}

// balTestFlagCase is one explicit `bal test` invocation needing its own
// flags, against a fixture under testdata/test/flags/<fixture>. seed writes
// extra files (relative path -> content) into a sandbox COPY of the fixture
// before running — for state a real `bal test` run would produce on disk
// (e.g. target/rerun_test.json) but that git can't track directly, since
// **/target/ is gitignored.
type balTestFlagCase struct {
	name    string
	fixture string
	args    []string // extra args after "test <fixtureDir>"
	seed    map[string]string
}

// balTestFlagCheckCase is like balTestFlagCase but additionally asserts on
// files the run leaves on disk — for scenarios whose correctness can't be
// captured by stdout/stderr/exitcode alone (rerun_test.json or a
// test_report JSON file's content, or a file's absence). Kept as its own
// table/type, rather than a field on balTestFlagCase, so the ~70 existing
// positional balTestFlagCase literals don't all need a trailing field added.
type balTestFlagCheckCase struct {
	name       string
	fixture    string
	args       []string
	seed       map[string]string
	checkFiles []balTestFileCheck
}

// runBalTestFlagCase sandboxes the fixture when needed (seed or checkFiles
// is set, so parallel subtests never race on a shared fixture directory),
// runs it, compares against the txtar golden, then runs any checkFiles
// assertions against the resulting on-disk state.
func runBalTestFlagCase(t *testing.T, cli *balCLI, testDataRoot, outputsRoot, name, fixture string, args []string, seed map[string]string, checkFiles []balTestFileCheck) {
	t.Helper()
	runDir := filepath.Join(testDataRoot, fixture)
	// checkBaseDir is where balTestFileCheck.path is resolved from: runDir
	// itself for a directory fixture, or its parent for a standalone .bal
	// file (target/ lands next to the file, not "inside" it).
	checkBaseDir := runDir
	// sandboxed tracks whether checkBaseDir is our own explicit t.TempDir()
	// sandbox, whose random path needs normalizing out of stdout/stderr too
	// — e.g. the "Generating Test Report" message prints the report file's
	// absolute path verbatim.
	sandboxed := false
	if len(seed) > 0 || len(checkFiles) > 0 {
		sandboxed = true
		sandboxRoot := t.TempDir()
		info, statErr := os.Stat(runDir)
		if statErr != nil {
			t.Fatalf("stat fixture %s: %v", runDir, statErr)
		}
		if info.IsDir() {
			copyDir(t, runDir, sandboxRoot)
			runDir = sandboxRoot
			checkBaseDir = runDir
		} else {
			runDir = copyFileIntoSandbox(t, runDir, sandboxRoot)
			checkBaseDir = filepath.Dir(runDir)
		}
		for rel, content := range seed {
			path := filepath.Join(runDir, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir for seed file %s: %v", rel, err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("write seed file %s: %v", rel, err)
			}
		}
	}
	fullArgs := append([]string{"test", runDir}, args...)
	var extraRoots []string
	if sandboxed {
		extraRoots = []string{checkBaseDir}
	}
	assertBalTestMatchesTxtar(t, cli, fullArgs, filepath.Join(outputsRoot, name+".txtar"), extraRoots...)
	for _, fc := range checkFiles {
		assertBalTestFileCheck(t, checkBaseDir, fc)
	}
}

// testBalTestFlagsCorpus runs each flag-based scenario registered in the
// tables below, mirroring TestBalRunCompilerPluginManifestCorpus's shape for
// a scenario that can't be purely directory-auto-discovered.
func testBalTestFlagsCorpus(t *testing.T, cli *balCLI) {
	testDataRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "testdata", "test", "flags")
	outputsRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "output", "test", "flags")

	for _, tc := range balTestFlagCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runBalTestFlagCase(t, cli, testDataRoot, outputsRoot, tc.name, tc.fixture, tc.args, tc.seed, nil)
		})
	}
	for _, tc := range balTestFlagCheckCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runBalTestFlagCase(t, cli, testDataRoot, outputsRoot, tc.name, tc.fixture, tc.args, tc.seed, tc.checkFiles)
		})
	}
}

// balTestStepsCase is a multi-invocation scenario (e.g. --rerun-failed)
// against the SAME fixture copy across steps, so on-disk state
// (target/rerun_test.json) carries over exactly as two real `bal test`
// invocations would. Mirrors cli/cmd/test_golden_test.go's
// runGoldenScenarioSteps, now through the shared pre-built binary.
type balTestStepsCase struct {
	name    string
	fixture string
	steps   [][]string // each entry: extra args after "test <sandboxDir>"
}

var balTestFlagCases = []balTestFlagCase{
	{"data-provider-valid", "data-providers", []string{"--tests", "jsonDataProviderTest"}, nil},
	{"data-provider-valid-with-fail", "data-providers", []string{"--tests", "intDataProviderTest"}, nil},
	{"data-provider-case-without-quotes", "data-providers", []string{"--tests", "dataproviders:jsonDataProviderTest#json1"}, nil},
	{"data-provider-case", "data-providers", []string{"--tests", "dataproviders:jsonDataProviderTest#'json1'"}, nil},
	{"data-provider-mixed-type-without-quotes", "data-providers", []string{"--tests", "testFunction1#CaseNew*"}, nil},
	{"data-provider-mixed-type", "data-providers", []string{"--tests", "testFunction1#'CaseNew*'"}, nil},
	{"data-provider-special-keys", "data-providers", []string{"--tests", "testFunction2"}, nil},
	{"data-provider-array-with-fail", "data-providers", []string{"--tests", "intArrayDataProviderTest"}, nil},
	{"data-provider-multi-module-single-test", "data-providers", []string{"--tests", "stringDataProviderMod1Test#1"}, nil},
	{"data-provider-map-value", "data-providers", []string{"--tests", "testGetState"}, nil},
	{"data-provider-before-after-functions", "data-providers", []string{"--tests", "testExecutionOfBeforeAfter"}, nil},
	{"data-provider-before-failing", "data-providers", []string{"--tests", "testDividingValuesWithBeforeFailing,testExecutionOfBeforeFailing"}, nil},
	{"data-provider-after-failing", "data-providers", []string{"--tests", "testDividingValuesWithAfterFailing,testExecutionOfAfterFailing"}, nil},
	{"data-provider-single-failure-cascade", "data-providers", []string{"--tests", "testExecutionOfDataValueFailing"}, nil},
	// testFunction3#'<key>' round-trips 7 special-character keys through a
	// percent-encoded, single-quoted --tests filter — a direct port of
	// jballerina's testCodeFragmentKeys, args precomputed via
	// native.EscapeSpecialCharacters since this table is a plain literal.
	{"data-provider-code-fragment-key-0", "data-providers", []string{"--tests", "testFunction3#'%60'%22%5Ca%22%22'"}, nil},
	{"data-provider-code-fragment-key-1", "data-providers", []string{"--tests", "testFunction3#'%22%5Cu{D7FF}%22%22%09%22'"}, nil},
	{"data-provider-code-fragment-key-2", "data-providers", []string{"--tests", "testFunction3#'a +%0A%0D b'"}, nil},
	{"data-provider-code-fragment-key-3", "data-providers", []string{"--tests", "testFunction3#'(x * 1) %21= (y / 3) || (a ^ b) == (b & c) >> (1 % 2)'"}, nil},
	{"data-provider-code-fragment-key-4", "data-providers", []string{"--tests", "testFunction3#'%281'"}, nil},
	{"data-provider-code-fragment-key-5", "data-providers", []string{"--tests", "testFunction3#'a:x(c%2Cd)[]; ^(x|y).ok();'"}, nil},
	{"data-provider-code-fragment-key-6", "data-providers", []string{"--tests", "testFunction3#'map<any> v = { %22x%22: 1 };'"}, nil},
	// Same 7 keys with one character each replaced by a '*' wildcard — a
	// direct port of jballerina's testCodeFragmentKeysWithWildCard.
	{"data-provider-code-fragment-key-wildcard-0", "data-providers", []string{"--tests", "testFunction3#'%60'%22%5C*%22%22'"}, nil},
	{"data-provider-code-fragment-key-wildcard-1", "data-providers", []string{"--tests", "testFunction3#'%22%5Cu{D7FF}%22*%09%22'"}, nil},
	{"data-provider-code-fragment-key-wildcard-2", "data-providers", []string{"--tests", "testFunction3#'a +%0A* b'"}, nil},
	{"data-provider-code-fragment-key-wildcard-3", "data-providers", []string{"--tests", "testFunction3#'(x * 1) *= (y / 3) || (a ^ b) == (b & c) >> (1 % 2)'"}, nil},
	{"data-provider-code-fragment-key-wildcard-4", "data-providers", []string{"--tests", "testFunction3#'*1'"}, nil},
	{"data-provider-code-fragment-key-wildcard-5", "data-providers", []string{"--tests", "testFunction3#'a:x(c*d)[]; ^(x|y).ok();'"}, nil},
	{"data-provider-code-fragment-key-wildcard-6", "data-providers", []string{"--tests", "testFunction3#'map<any> v = { *x%22: 1 };'"}, nil},
	{"grouping-single-group-execution", "grouping", []string{"--groups", "g1"}, nil},
	{"grouping-multiple-group-execution", "grouping", []string{"--groups", "g2,g4"}, nil},
	{"grouping-single-group-exclusion", "grouping", []string{"--disable-groups", "g5"}, nil},
	{"grouping-multiple-group-exclusion", "grouping", []string{"--disable-groups", "g1,g5,g6"}, nil},
	{"grouping-non-existing-group-inclusion", "grouping", []string{"--groups", "g10"}, nil},
	{"grouping-non-existing-group-exclusion", "grouping", []string{"--disable-groups", "g10"}, nil},
	{"grouping-listing-of-test-groups", "grouping-listing", []string{"--list-groups"}, nil},
	{"grouping-after-groups-with-disabled-test", "grouping-after-groups-disabled", []string{"--groups", "g1"}, nil},
	{"rejects-graalvm", "rejects-unsupported-flags", []string{"--graalvm"}, nil},
	{"rejects-cloud", "rejects-unsupported-flags", []string{"--cloud", "docker"}, nil},
	{"module-qualified-tests-filter-bare", "module-qualified-tests-filter", []string{"--tests", "testInSubmod"}, nil},
	{"module-qualified-tests-filter-qualified", "module-qualified-tests-filter", []string{"--tests", "modquantestmod.submod:testInSubmod"}, nil},
	{"selected-function-non-existing", "selected-function", []string{"--tests", "nonExistingFunc"}, nil},
	{"selected-function-dependent-disabled", "selected-function", []string{"--tests", "testDependentDisabledFunc"}, nil},
	{"selected-function-single", "selected-function", []string{"--tests", "testFunc"}, nil},
	{"selected-function-dependent", "selected-function", []string{"--tests", "testFunc2"}, nil},
	{"selected-function-multiple", "selected-function", []string{"--tests", "testFunc,testFunc2"}, nil},
	{"selected-function-disabled", "selected-function", []string{"--tests", "testDisabledFunc"}, nil},
	{"rerun-failed-invalid-json", "rerun-failed-invalid-json", []string{"--rerun-failed"}, nil},
	{"rerun-failed-missing-module-key", "rerun-failed-missing-module-key", []string{"--rerun-failed"}, nil},
	{"rerun-failed-no-prior-run", "rerun-failed-no-prior-run", []string{"--rerun-failed"}, nil},
	{"list-groups", "list-groups", []string{"--list-groups"}, nil},
	{"group-filtering", "group-filtering", []string{"--groups", "unit"}, nil},
	{"standalone-all-passing", "standalone-all-passing.bal", nil, nil},
	{"standalone-rejects-non-bal-file", "standalone-rejects-non-bal-file/notes.txt", nil, nil},
	// jballerina's testListingOfSingleTestGroups: --list-groups lists every
	// discovered group.
	{"standalone-grouping-list-groups", "standalone-groups-test.bal", []string{"--list-groups"}, nil},
	// jballerina's testListGroupsWithOtherFlags: --groups g1 --list-groups
	// still lists every group and prints the "other flags skipped" warning.
	{"standalone-grouping-list-groups-with-other-flags", "standalone-groups-test.bal", []string{"--groups", "g1", "--list-groups"}, nil},
	{"workspace-single-member-passing", "workspace-single-member/passing", nil, nil},
	{"workspace-single-member-failing", "workspace-single-member/failing", nil, nil},
	{
		"rerun-failed-with-invalid-run-test-json", "rerun-failed-with-invalid-run-test-json", []string{"--rerun-failed"},
		map[string]string{"target/rerun_test.json": "This is an invalid json file. Testerina will throw an error when it tries to parse this as a json file."},
	},
	{
		"rerun-failed-with-missing-module-name", "rerun-failed-with-missing-module-name", []string{"--rerun-failed"},
		map[string]string{"target/rerun_test.json": `{"non_existent_module":{"testNames":["testFunctionFail1","testFunctionFail2"],"testModuleNames":{"testFunctionFail1":"rerun_failed","testFunctionFail2":"rerun_failed"},"subTestNames":{}}}`},
	},
}

// balTestFlagCheckCases are cli/cmd/test_test.go ports whose correctness
// needs a generated file's content, not just stdout/stderr/exitcode.
var balTestFlagCheckCases = []balTestFlagCheckCase{
	// jballerina-equivalent: a failing suite writes rerun_test.json listing
	// only the failed test. Direct port of
	// TestTestCommand_FailureExitsNonZeroAndWritesRerunFile.
	{
		name: "failure-writes-rerun-file", fixture: "hasfailure",
		checkFiles: []balTestFileCheck{{
			path: filepath.Join("target", "rerun_test.json"),
			check: func(t *testing.T, data []byte) {
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
			},
		}},
	},
	// Direct port of TestTestCommand_RerunFailed_FullTest (jballerina's
	// testFullTest): 2 passing/2 failing writes rerun_test.json with the 2
	// failures.
	{
		name: "rerun-failed-full-test", fixture: "rerun-failed-full-test",
		checkFiles: []balTestFileCheck{{
			path: filepath.Join("target", "rerun_test.json"),
			check: func(t *testing.T, data []byte) {
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
				got := map[string]bool{}
				for _, n := range entry.TestNames {
					got[n] = true
				}
				if len(got) != 2 || !got["testFunctionFail1"] || !got["testFunctionFail2"] {
					t.Errorf("expected exactly testFunctionFail1/testFunctionFail2, got: %+v", entry.TestNames)
				}
			},
		}},
	},
	// Direct port of TestTestCommand_StandaloneFile_FailureExitsNonZero:
	// target/ lands next to the standalone file, not inside a package dir.
	{
		name: "standalone-failure-writes-rerun-file", fixture: "standalone-fail.bal",
		checkFiles: []balTestFileCheck{{
			path:  filepath.Join("target", "rerun_test.json"),
			check: func(t *testing.T, data []byte) {},
		}},
	},
	// Direct port of TestTestCommand_TestReport_NotGeneratedWithoutFlag.
	{
		name: "report-not-generated-without-flag", fixture: "report-not-generated",
		checkFiles: []balTestFileCheck{{
			path:       filepath.Join("target", "report", "test_results.json"),
			wantAbsent: true,
		}},
	},
	// Direct port of TestTestCommand_TestReport_SinglePackage. The
	// "Generating Test Report" cobra message is asserted via this case's
	// own stdout golden, not here.
	{
		name: "report-single-package", fixture: "report-single-package", args: []string{"--test-report"},
		checkFiles: []balTestFileCheck{{
			path: filepath.Join("target", "report", "test_results.json"),
			check: func(t *testing.T, data []byte) {
				var report corpusPackageTestResult
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
			},
		}},
	},
	// Direct port of TestTestCommand_TestReport_FailureMessagePreservesNonAscii:
	// guards against corrupting a multi-byte UTF-8 character while building
	// module_status.json. See testrunner.BuildModuleStatus's doc comment
	// for why the exact message has no double-escaped quotes or synthetic
	// trailing newline.
	{
		name: "report-nonascii-failure-message", fixture: "report-nonascii-failure", args: []string{"--test-report"},
		checkFiles: []balTestFileCheck{{
			path: filepath.Join("target", "report", "test_results.json"),
			check: func(t *testing.T, data []byte) {
				var report corpusPackageTestResult
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
			},
		}},
	},
	// Direct port of TestTestCommand_TestReport_MultiModulePackage.
	{
		name: "report-multi-module-package", fixture: "report-multi-module", args: []string{"--test-report"},
		checkFiles: []balTestFileCheck{{
			path: filepath.Join("target", "report", "test_results.json"),
			check: func(t *testing.T, data []byte) {
				var report corpusPackageTestResult
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
			},
		}},
	},
	// Direct port of TestTestCommand_TestReport_Workspace: a workspace-level
	// test_results.json aggregates both members, with no per-member report.
	{
		name: "report-workspace", fixture: "report-workspace", args: []string{"--test-report"},
		checkFiles: []balTestFileCheck{
			{
				path: filepath.Join("target", "report", "test_results.json"),
				check: func(t *testing.T, data []byte) {
					var report corpusWorkspaceTestReport
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
				},
			},
			{path: filepath.Join("passing", "target", "report", "test_results.json"), wantAbsent: true},
			{path: filepath.Join("failing", "target", "report", "test_results.json"), wantAbsent: true},
		},
	},
}

var balTestStepsCases = []balTestStepsCase{
	{"rerun-failed-with-data-provider", "rerun-failed-with-data-provider", [][]string{nil, {"--rerun-failed"}}},
	{"rerun-failed-with-array-data-provider", "rerun-failed-with-array-data-provider", [][]string{nil, {"--rerun-failed"}}},
	{"rerun-failed-only-runs-previous-failures", "rerun-failed-only-runs-previous-failures", [][]string{nil, {"--rerun-failed"}}},
	{"rerun-failed-rerun-failed-test", "rerun-failed-rerun-failed-test", [][]string{nil, {"--rerun-failed"}}},
}

func testBalTestStepsCorpus(t *testing.T, cli *balCLI) {
	testDataRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "testdata", "test", "steps")
	outputsRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "output", "test", "steps")

	for _, tc := range balTestStepsCases {
		t.Run(tc.name, func(t *testing.T) {
			sandbox := t.TempDir()
			copyDir(t, filepath.Join(testDataRoot, tc.fixture), sandbox)

			var combined strings.Builder
			var lastExit int
			for i, extraArgs := range tc.steps {
				args := append([]string{"test", sandbox}, extraArgs...)
				stdout, _, exitCode := runCLICommand(t, cli.relBin, cli.repoRoot, cli.coverDir, args...)
				if i > 0 {
					combined.WriteString("\n")
				}
				displayArgs := append([]string{"test", "<FIXTURE>"}, extraArgs...)
				fmt.Fprintf(&combined, "$ bal %s\n", strings.Join(displayArgs, " "))
				combined.WriteString(normalizeBalTestOutput(stdout, cli.repoRoot))
				lastExit = exitCode
			}
			actualExit := strconv.Itoa(lastExit)
			expectedPath := filepath.Join(outputsRoot, tc.name+".txtar")

			if *update {
				files := append(test_util.TxtarFilesStdoutStderr(combined.String(), ""),
					txtar.File{Name: "exitcode", Data: []byte(actualExit)})
				if test_util.UpdateTxtarArchiveIfNeeded(t, expectedPath, files) {
					t.Logf("updated %s", expectedPath)
				}
				return
			}

			expectedStdout, _, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(expectedPath)
			if err != nil {
				t.Fatalf("failed to parse txtar file %s: %v", expectedPath, err)
			}
			if combined.String() != expectedStdout || actualExit != expectedExit {
				t.Fatalf("mismatch for steps case %q (expected file %s)\n%s\ngot:\n%s",
					tc.name, expectedPath, test_util.FormatExpectedGot(expectedExit, actualExit), combined.String())
			}
		})
	}
}
