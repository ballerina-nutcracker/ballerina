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

package testrunner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/ballerina-nutcracker/ballerina/compilerpluginregistry"
	_ "github.com/ballerina-nutcracker/ballerina/lib/rt"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/testplan"
	"github.com/ballerina-nutcracker/ballerina/testrunner"
)

// TestRun_ReportEndToEnd compiles and runs the real "failing" fixture, then
// checks the live pipeline produces the same report format report_test.go's
// golden tests verify against hand-built RunReports.
func TestRun_ReportEndToEnd(t *testing.T) {
	rt, plan := compileAndRun(t, "failing")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "failing", plan, graph, nil, nil)

	var sb strings.Builder
	testrunner.WriteConsoleReport(&sb, report, false, 0)
	out := sb.String()

	if !strings.Contains(out, "\n\ttestFails: has failed.\n\n") {
		t.Errorf("expected the live \"has failed\" notice for testFails, got %q", out)
	}
	if !strings.Contains(out, "\t\t[pass] testPasses\n") {
		t.Errorf("expected a [pass] line for testPasses, got %q", out)
	}
	if !strings.Contains(out, "\n\t\t[fail] testFails:\n") {
		t.Errorf("expected a [fail] line for testFails, got %q", out)
	}
	if !strings.Contains(out, "Assertion Failed!") {
		t.Errorf("expected the assertion message in the report, got %q", out)
	}
	if !strings.Contains(out, "\t\t1 passing\n\t\t1 failing\n\t\t0 skipped\n") {
		t.Errorf("expected the summary counts, got %q", out)
	}

	summary := testrunner.Summarize(report)
	if summary.ExitCode != 1 {
		t.Errorf("expected ExitCode 1, got %d", summary.ExitCode)
	}
}

// compileAndRun compiles testdata/name (letting the real compiler plugin
// populate testplan's stash) and returns a runtime ready for invocation.
func compileAndRun(t *testing.T, name string) (*runtime.Runtime, *testplan.TestPlan) {
	t.Helper()

	absPath, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve user home: %v", err)
	}
	ballerinaEnvPath := os.Getenv(projects.BallerinaEnvVar)
	if ballerinaEnvPath == "" {
		ballerinaEnvPath = filepath.Join(userHome, projects.UserHomeDirName)
	}
	buildOpts := projects.NewBuildOptionsBuilder().WithOffline(true).WithSkipTests(false).Build()

	result, err := projects.Load(os.DirFS(absPath), ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	if diag := result.Diagnostics(); diag.HasErrors() {
		t.Fatalf("project failed to load: %+v", diag.Diagnostics())
	}

	project := result.Project()
	pkg := project.CurrentPackage()
	compilation := pkg.Compilation()
	if cd := compilation.DiagnosticResult(); cd.HasErrors() {
		var messages []string
		for _, d := range cd.Diagnostics() {
			messages = append(messages, d.Message())
		}
		t.Fatalf("unexpected compilation errors: %v", messages)
	}

	plan, ok := testplan.Load(testplan.ModuleKey{Org: "testorg", Module: name})
	if !ok {
		t.Fatalf("plugin did not stash a TestPlan for %s — did it fire?", name)
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		t.Fatal("no BIR packages produced")
	}
	pal, cleanup := palnative.NewPlatform()
	t.Cleanup(cleanup)
	rt := runtime.NewRuntime(pal, project.Environment().TypeEnv())
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			t.Fatalf("rt.Init: %v", err)
		}
	}
	rt.Listen()

	return rt, plan
}

func resultsByName(results []testrunner.Result) map[string]testrunner.Result {
	m := make(map[string]testrunner.Result, len(results))
	for _, r := range results {
		m[r.Name] = r
	}
	return m
}

func TestRun_PassSkipAndOrder(t *testing.T) {
	rt, plan := compileAndRun(t, "sample")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "sample", plan, graph, nil, nil)
	results := report.Results
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	// This fixture's BeforeSuite/AfterSuite/BeforeEach/AfterEach/BeforeGroups/
	// AfterGroups hooks all succeed silently — confirm the lifecycle wiring
	// doesn't itself introduce a spurious hook failure.
	if len(report.HookFailures) != 0 {
		t.Errorf("expected no hook failures, got %+v", report.HookFailures)
	}

	byName := resultsByName(results)
	// testDisabled is excluded entirely, not reported as skipped.
	if _, ok := byName["testDisabled"]; ok {
		t.Errorf("expected testDisabled to be excluded entirely, got a result: %+v", byName["testDisabled"])
	}
	if byName["testSetup"].Status != testrunner.Pass {
		t.Errorf("testSetup: expected Pass, got %v (%s)", byName["testSetup"].Status, byName["testSetup"].Message)
	}
	if byName["testMain"].Status != testrunner.Pass {
		t.Errorf("testMain: expected Pass, got %v (%s)", byName["testMain"].Status, byName["testMain"].Message)
	}

	// testMain depends on testSetup — it must run after it.
	setupIdx, mainIdx := -1, -1
	for i, r := range results {
		switch r.Name {
		case "testSetup":
			setupIdx = i
		case "testMain":
			mainIdx = i
		}
	}
	if setupIdx == -1 || mainIdx == -1 || setupIdx > mainIdx {
		t.Errorf("expected testSetup to run before testMain, got order %+v", results)
	}
}

func TestRun_IncludedFiltersWhatRuns(t *testing.T) {
	rt, plan := compileAndRun(t, "sample")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	results := testrunner.Run(rt, "testorg", "sample", plan, graph, map[string]bool{"testSetup": true}, nil).Results
	if len(results) != 1 || results[0].Name != "testSetup" {
		t.Fatalf("expected only testSetup to run, got %+v", results)
	}
}

func TestRun_FailedAssertionReportsFailWithMessage(t *testing.T) {
	rt, plan := compileAndRun(t, "failing")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	results := testrunner.Run(rt, "testorg", "failing", plan, graph, nil, nil).Results
	byName := resultsByName(results)

	if byName["testPasses"].Status != testrunner.Pass {
		t.Errorf("testPasses: expected Pass, got %v", byName["testPasses"].Status)
	}
	failed := byName["testFails"]
	if failed.Status != testrunner.Fail {
		t.Fatalf("testFails: expected Fail, got %v", failed.Status)
	}
	if failed.Message == "" {
		t.Error("testFails: expected a non-empty failure message")
	}
	t.Logf("testFails message: %q", failed.Message)
}

// TestRun_BeforeSuiteFailureSkipsEverything confirms a failed BeforeSuite
// skips every test and gates AfterSuite hooks on their own alwaysRun flag.
func TestRun_BeforeSuiteFailureSkipsEverything(t *testing.T) {
	rt, plan := compileAndRun(t, "beforesuitefails")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "beforesuitefails", plan, graph, nil, nil)

	byName := resultsByName(report.Results)
	if byName["testA"].Status != testrunner.Skip {
		t.Errorf("testA: expected Skip (BeforeSuite failed), got %v", byName["testA"].Status)
	}

	var sawSuiteFailure, sawAlwaysRunFailure, sawNormalFailure bool
	for _, hf := range report.HookFailures {
		switch {
		case hf.Hook == "before test suite function":
			sawSuiteFailure = true
		case hf.Hook == "after test suite function" && strings.Contains(hf.Message, "teardownAlways"):
			sawAlwaysRunFailure = true
		case hf.Hook == "after test suite function" && strings.Contains(hf.Message, "teardownNormal"):
			sawNormalFailure = true
		}
	}
	if !sawSuiteFailure {
		t.Error("expected the BeforeSuite failure to be recorded")
	}
	if !sawAlwaysRunFailure {
		t.Error("expected teardownAlways (alwaysRun: true) to have actually run despite the BeforeSuite failure")
	}
	if sawNormalFailure {
		t.Error("expected teardownNormal (no alwaysRun) to be skipped, but it ran")
	}
}

// TestRun_BeforeEachFailureIsSticky confirms a failed BeforeEach sets a
// shouldSkip flag that's never cleared — every remaining test skips too.
func TestRun_BeforeEachFailureIsSticky(t *testing.T) {
	rt, plan := compileAndRun(t, "beforeeachfails")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "beforeeachfails", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if byName["testA"].Status != testrunner.Skip {
		t.Errorf("testA: expected Skip, got %v", byName["testA"].Status)
	}
	if byName["testB"].Status != testrunner.Skip {
		t.Errorf("testB: expected Skip (shouldSkip is sticky), got %v", byName["testB"].Status)
	}

	// BeforeEach must only be attempted once — for testA — not again for
	// testB, since shouldSkip short-circuits the second attempt entirely.
	count := 0
	for _, hf := range report.HookFailures {
		if hf.Hook == "before each function for the test" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 BeforeEach failure recorded, got %d: %+v", count, report.HookFailures)
	}
}

// TestRun_BeforeGroupFailureOnlySkipsTheTriggeringTest confirms BeforeGroups
// fires once per group (first test only), so its failure skips only that
// test — a second test in the same group runs normally.
func TestRun_BeforeGroupFailureOnlySkipsTheTriggeringTest(t *testing.T) {
	rt, plan := compileAndRun(t, "beforegroupfails")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "beforegroupfails", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if byName["testA"].Status != testrunner.Skip {
		t.Errorf("testA: expected Skip (triggered the BeforeGroups failure), got %v", byName["testA"].Status)
	}
	if byName["testB"].Status != testrunner.Pass {
		t.Errorf("testB: expected Pass (BeforeGroups only fires once, for testA), got %v (%s)",
			byName["testB"].Status, byName["testB"].Message)
	}

	// AfterGroups has no alwaysRun here, and the group was marked
	// skip-after by the BeforeGroups failure, so it must never run.
	for _, hf := range report.HookFailures {
		if hf.Hook == "after test group function for the test" {
			t.Errorf("expected AfterGroups to be skipped (no alwaysRun), but it ran: %+v", hf)
		}
	}
}

// TestRun_FailedDependencyCascadesSkip confirms a failing test's dependents
// are skipped rather than run.
func TestRun_FailedDependencyCascadesSkip(t *testing.T) {
	rt, plan := compileAndRun(t, "dependsonfails")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "dependsonfails", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if byName["testA"].Status != testrunner.Fail {
		t.Errorf("testA: expected Fail, got %v", byName["testA"].Status)
	}
	if byName["testB"].Status != testrunner.Skip {
		t.Errorf("testB: expected Skip (its dependency testA failed), got %v", byName["testB"].Status)
	}
}

// TestRun_MapDataProviderRowsAndNoDependentCascade confirms each map key
// becomes its own "name#key" Result, and a dependent of a data-driven test
// is never skip-cascaded even when one row fails.
func TestRun_MapDataProviderRowsAndNoDependentCascade(t *testing.T) {
	rt, plan := compileAndRun(t, "dataprovidermap")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "dataprovidermap", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if len(report.Results) != 4 {
		t.Fatalf("expected 4 results (3 data rows + testDependent), got %d: %+v", len(report.Results), report.Results)
	}
	if byName["testSquare#one"].Status != testrunner.Pass {
		t.Errorf("testSquare#one: expected Pass, got %v", byName["testSquare#one"].Status)
	}
	if byName["testSquare#two"].Status != testrunner.Pass {
		t.Errorf("testSquare#two: expected Pass, got %v", byName["testSquare#two"].Status)
	}
	failed := byName["testSquare#three"]
	if failed.Status != testrunner.Fail {
		t.Fatalf("testSquare#three: expected Fail, got %v", failed.Status)
	}
	if failed.Message == "" {
		t.Error("testSquare#three: expected a non-empty failure message")
	}
	if byName["testDependent"].Status != testrunner.Pass {
		t.Errorf("testDependent: expected Pass (data-driven tests never skip-cascade dependents), got %v (%s)",
			byName["testDependent"].Status, byName["testDependent"].Message)
	}
}

// TestRun_TupleDataProviderRowsKeyedByIndex mirrors the other data-provider
// return shape — [...][] — whose rows are keyed by stringified position.
func TestRun_TupleDataProviderRowsKeyedByIndex(t *testing.T) {
	rt, plan := compileAndRun(t, "dataprovidertuple")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "dataprovidertuple", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(report.Results), report.Results)
	}
	if byName["testAdd#0"].Status != testrunner.Pass {
		t.Errorf("testAdd#0: expected Pass, got %v (%s)", byName["testAdd#0"].Status, byName["testAdd#0"].Message)
	}
	if byName["testAdd#1"].Status != testrunner.Fail {
		t.Errorf("testAdd#1: expected Fail, got %v", byName["testAdd#1"].Status)
	}
}

// TestRun_DataProviderFailureShortCircuitsBeforeHooks confirms a panicking
// data provider fails the whole test under its bare name and skips its
// entire hook lifecycle, including BeforeEach.
func TestRun_DataProviderFailureShortCircuitsBeforeHooks(t *testing.T) {
	rt, plan := compileAndRun(t, "dataproviderpanics")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "dataproviderpanics", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(report.Results), report.Results)
	}
	providerFailure := byName["testProviderFails"]
	if providerFailure.Status != testrunner.Fail {
		t.Fatalf("testProviderFails: expected Fail (reported under its bare name, no #key), got %v", providerFailure.Status)
	}
	if providerFailure.Message == "" {
		t.Error("testProviderFails: expected a non-empty failure message")
	}
	if byName["testZNormal"].Status != testrunner.Pass {
		t.Errorf("testZNormal: expected Pass (beforeEachCount must be exactly 1), got %v (%s)",
			byName["testZNormal"].Status, byName["testZNormal"].Message)
	}
	if len(report.HookFailures) != 0 {
		t.Errorf("expected no hook failures, got %+v", report.HookFailures)
	}
}

// TestRun_BeforeTestFailureSkipsWithoutInvokingAndCascades confirms a
// failing before: hook Skips the test without invoking its body, and still
// cascades skip to its dependent.
func TestRun_BeforeTestFailureSkipsWithoutInvokingAndCascades(t *testing.T) {
	rt, plan := compileAndRun(t, "beforetestfails")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "beforetestfails", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if byName["testA"].Status != testrunner.Skip {
		t.Errorf("testA: expected Skip (before: hook failed), got %v", byName["testA"].Status)
	}
	if byName["testZCheckSideEffect"].Status != testrunner.Pass {
		t.Errorf("testZCheckSideEffect: expected Pass (testA's body must never have run), got %v (%s)",
			byName["testZCheckSideEffect"].Status, byName["testZCheckSideEffect"].Message)
	}
	if byName["testB"].Status != testrunner.Skip {
		t.Errorf("testB: expected Skip (its dependency testA was skipped), got %v", byName["testB"].Status)
	}

	count := 0
	for _, hf := range report.HookFailures {
		if hf.Hook == "before test function for the test" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 before-test-function failure recorded, got %d: %+v", count, report.HookFailures)
	}
}

// TestRun_AfterTestFailureCascadesSkipDespitePassing confirms a failing
// after: hook doesn't change the test's own Pass result but still cascades
// skip to dependents.
func TestRun_AfterTestFailureCascadesSkipDespitePassing(t *testing.T) {
	rt, plan := compileAndRun(t, "aftertestfails")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "aftertestfails", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if byName["testA"].Status != testrunner.Pass {
		t.Errorf("testA: expected Pass (its own body succeeded), got %v (%s)", byName["testA"].Status, byName["testA"].Message)
	}
	if byName["testB"].Status != testrunner.Skip {
		t.Errorf("testB: expected Skip (testA's after: hook failed), got %v", byName["testB"].Status)
	}

	count := 0
	for _, hf := range report.HookFailures {
		if hf.Hook == "after test function for the test" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 after-test-function failure recorded, got %d: %+v", count, report.HookFailures)
	}
}

// TestRun_DataDrivenBeforeTestFailureSkipsEachRowUnderBareName confirms a
// before: hook is attempted independently per row, and a failure reports
// that row Skip under the test's bare name, not "name#key".
func TestRun_DataDrivenBeforeTestFailureSkipsEachRowUnderBareName(t *testing.T) {
	rt, plan := compileAndRun(t, "dataproviderbeforefails")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "dataproviderbeforefails", plan, graph, nil, nil)

	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results (one per data row), got %d: %+v", len(report.Results), report.Results)
	}
	for _, r := range report.Results {
		if r.Name != "testRows" {
			t.Errorf("expected every skipped row reported under the bare name %q, got %q", "testRows", r.Name)
		}
		if r.Status != testrunner.Skip {
			t.Errorf("testRows row: expected Skip, got %v", r.Status)
		}
	}

	count := 0
	for _, hf := range report.HookFailures {
		if hf.Hook == "before test function for the test" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected the before: hook to be attempted independently for both rows (2 failures), got %d: %+v",
			count, report.HookFailures)
	}
}

// TestRun_SubKeyFilterExactMatch confirms --tests "testSquare#one" runs only
// that row — the others aren't reported at all, not even Skip.
func TestRun_SubKeyFilterExactMatch(t *testing.T) {
	rt, plan := compileAndRun(t, "dataprovidermap")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	included := map[string]bool{"testSquare": true}
	subKeys := testrunner.ParseSubKeyFilters("testSquare#one")
	report := testrunner.Run(rt, "testorg", "dataprovidermap", plan, graph, included, subKeys)

	if len(report.Results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d: %+v", len(report.Results), report.Results)
	}
	if report.Results[0].Name != "testSquare#one" || report.Results[0].Status != testrunner.Pass {
		t.Errorf("expected testSquare#one Pass, got %+v", report.Results[0])
	}
}

// TestRun_SubKeyFilterMultipleKeys confirms
// "testSquare#one,testSquare#three" accumulates both suffixes under one
// entry, rather than the second overwriting the first.
func TestRun_SubKeyFilterMultipleKeys(t *testing.T) {
	rt, plan := compileAndRun(t, "dataprovidermap")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	included := map[string]bool{"testSquare": true}
	subKeys := testrunner.ParseSubKeyFilters("testSquare#one,testSquare#three")
	report := testrunner.Run(rt, "testorg", "dataprovidermap", plan, graph, included, subKeys)
	byName := resultsByName(report.Results)

	if len(report.Results) != 2 {
		t.Fatalf("expected exactly 2 results, got %d: %+v", len(report.Results), report.Results)
	}
	if byName["testSquare#one"].Status != testrunner.Pass {
		t.Errorf("testSquare#one: expected Pass, got %v", byName["testSquare#one"].Status)
	}
	if byName["testSquare#three"].Status != testrunner.Fail {
		t.Errorf("testSquare#three: expected Fail, got %v", byName["testSquare#three"].Status)
	}
	if _, ok := byName["testSquare#two"]; ok {
		t.Error("testSquare#two: expected no result at all (not requested), got one")
	}
}

// TestRun_SubKeyFilterWildcard mirrors skipDataDrivenTest's wildcard-suffix
// branch: "testSquare#t*" matches both "two" and "three" (both start with
// "t"), not "one".
func TestRun_SubKeyFilterWildcard(t *testing.T) {
	rt, plan := compileAndRun(t, "dataprovidermap")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	included := map[string]bool{"testSquare": true}
	subKeys := testrunner.ParseSubKeyFilters("testSquare#t*")
	report := testrunner.Run(rt, "testorg", "dataprovidermap", plan, graph, included, subKeys)
	byName := resultsByName(report.Results)

	if len(report.Results) != 2 {
		t.Fatalf("expected exactly 2 results, got %d: %+v", len(report.Results), report.Results)
	}
	if _, ok := byName["testSquare#one"]; ok {
		t.Error("testSquare#one: expected no result at all (doesn't match t*), got one")
	}
	if byName["testSquare#two"].Status != testrunner.Pass {
		t.Errorf("testSquare#two: expected Pass, got %v", byName["testSquare#two"].Status)
	}
	if byName["testSquare#three"].Status != testrunner.Fail {
		t.Errorf("testSquare#three: expected Fail, got %v", byName["testSquare#three"].Status)
	}
}

// TestRun_SubKeyFilterBypassedByEnabledDependent confirms a sub-key
// restriction is ignored entirely (every row runs) when a test transitively
// depending on the data-driven test is itself enabled and included.
func TestRun_SubKeyFilterBypassedByEnabledDependent(t *testing.T) {
	rt, plan := compileAndRun(t, "subkeydependentbypass")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	included := map[string]bool{"testSquare": true, "testUsesAll": true}
	subKeys := testrunner.ParseSubKeyFilters("testSquare#one")
	report := testrunner.Run(rt, "testorg", "subkeydependentbypass", plan, graph, included, subKeys)
	byName := resultsByName(report.Results)

	if byName["testSquare#one"].Status != testrunner.Pass {
		t.Errorf("testSquare#one: expected Pass, got %v", byName["testSquare#one"].Status)
	}
	if byName["testSquare#two"].Status != testrunner.Pass {
		t.Errorf("testSquare#two: expected Pass (bypassed by enabled dependent testUsesAll), got %v",
			byName["testSquare#two"].Status)
	}
	if byName["testUsesAll"].Status != testrunner.Pass {
		t.Errorf("testUsesAll: expected Pass, got %v (%s)", byName["testUsesAll"].Status, byName["testUsesAll"].Message)
	}
}

// TestRun_EmptyDataProviderRunsNothing confirms a validly-shaped data
// provider returning zero rows produces no Result, falling through to a
// "No tests found" report rather than "0 passing/0 failing/0 skipped".
func TestRun_EmptyDataProviderRunsNothing(t *testing.T) {
	rt, plan := compileAndRun(t, "emptydataprovider")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "emptydataprovider", plan, graph, nil, nil)

	if len(report.Results) != 0 {
		t.Errorf("expected no results from a zero-row data provider, got %+v", report.Results)
	}
	if len(report.HookFailures) != 0 {
		t.Errorf("expected no hook failures, got %+v", report.HookFailures)
	}
	if s := testrunner.Summarize(report); s.ExitCode != 0 {
		t.Errorf("expected ExitCode 0, got %d", s.ExitCode)
	}

	var sb strings.Builder
	testrunner.WriteConsoleReport(&sb, report, false, 0)
	if !strings.Contains(sb.String(), "No tests found") {
		t.Errorf("expected \"No tests found\", got %q", sb.String())
	}
}

// TestRun_DataProviderErrorAndFunctionValues confirms data rows with
// non-scalar elements (an `error` value, a `function` value) pass straight
// through as test arguments, not just ints.
func TestRun_DataProviderErrorAndFunctionValues(t *testing.T) {
	rt, plan := compileAndRun(t, "dataprovidervaluetypes")
	graph, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	report := testrunner.Run(rt, "testorg", "dataprovidervaluetypes", plan, graph, nil, nil)
	byName := resultsByName(report.Results)

	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(report.Results), report.Results)
	}
	if got := byName["testErrorData#foo"]; got.Status != testrunner.Pass {
		t.Errorf("testErrorData#foo: expected Pass, got %v (%s)", got.Status, got.Message)
	}
	if got := byName["testFunctionData#even"]; got.Status != testrunner.Pass {
		t.Errorf("testFunctionData#even: expected Pass, got %v (%s)", got.Status, got.Message)
	}
}
