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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/test_util"
	"golang.org/x/tools/txtar"
)

// normalizeBalTestOutput masks the wall-clock "Test execution time" trailer
// (via corpus/integration_test.go's shared testExecutionTimePattern) and
// repo-root paths, so fixtures stay deterministic.
func normalizeBalTestOutput(s, repoRoot string) string {
	return testExecutionTimePattern.ReplaceAllString(normalizePaths(test_util.NormalizeNewlines(s), repoRoot), "Test execution time : <DURATION>s")
}

// assertBalTestMatchesTxtar runs `bal <args...>` and compares stdout,
// stderr and exit code against expectedPath (-update regenerates it).
func assertBalTestMatchesTxtar(t *testing.T, cli *balCLI, args []string, expectedPath string) {
	t.Helper()
	stdout, stderr, exitCode := runCLICommand(t, cli.relBin, cli.repoRoot, cli.coverDir, args...)
	actualStdout := normalizeBalTestOutput(stdout, cli.repoRoot)
	actualStderr := normalizeBalTestOutput(stderr, cli.repoRoot)
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

// testBalTestFlagsCorpus runs each flag-based scenario registered in the
// table below, mirroring TestBalRunCompilerPluginManifestCorpus's shape for
// a scenario that can't be purely directory-auto-discovered.
func testBalTestFlagsCorpus(t *testing.T, cli *balCLI) {
	testDataRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "testdata", "test", "flags")
	outputsRoot := filepath.Join(cli.repoRoot, "corpus", "cli", "output", "test", "flags")

	for _, tc := range balTestFlagCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runDir := filepath.Join(testDataRoot, tc.fixture)
			if len(tc.seed) > 0 {
				sandbox := t.TempDir()
				copyDir(t, runDir, sandbox)
				for rel, content := range tc.seed {
					path := filepath.Join(sandbox, rel)
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatalf("mkdir for seed file %s: %v", rel, err)
					}
					if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
						t.Fatalf("write seed file %s: %v", rel, err)
					}
				}
				runDir = sandbox
			}
			args := append([]string{"test", runDir}, tc.args...)
			assertBalTestMatchesTxtar(t, cli, args, filepath.Join(outputsRoot, tc.name+".txtar"))
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
