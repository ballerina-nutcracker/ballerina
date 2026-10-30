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
	"strings"
	"testing"
	"time"
)

// TestWriteConsoleReport_MatchesDataProviderSingleFailureGolden reproduces
// cli/cmd/testdata/golden-output/data-provider-single-failure.txt byte for
// byte (confirmed directly against `od -c` of that file, not reconstructed
// from reading report.bal alone) — a data-driven test with one passing row
// and one failing row, the failing row's multi-line assertion message, and
// its "testSquare:bad has failed." live notice (note: ":" separator here,
// not the "#" used in the final [fail] line's name).
func TestWriteConsoleReport_MatchesDataProviderSingleFailureGolden(t *testing.T) {
	report := RunReport{
		Results: []Result{
			{Name: "testSquare#good", Status: Pass, Type: DataDrivenMapOfTuple},
			{Name: "testSquare#bad", Status: Fail, Type: DataDrivenMapOfTuple,
				Message: "Assertion Failed!\n \nexpected: '5'\nactual\t: '4'"},
		},
		Events: []ConsoleEvent{
			{TestFailure: &TestFailureNotice{Name: "testSquare", Suffix: "bad", HasSuffix: true}},
		},
	}

	var sb strings.Builder
	WriteConsoleReport(&sb, report, false, 0)

	want := "\n\ttestSquare:bad has failed.\n\n" +
		"\t\t[pass] testSquare#good\n" +
		"\n\t\t[fail] testSquare#bad:\n" +
		"\n\t\t    Assertion Failed!\n\t\t\t \n\t\t\texpected: '5'\n\t\t\tactual\t: '4'\n\t\t\t\n\t\t\t\n" +
		"\n\n" +
		"\t\t1 passing\n\t\t1 failing\n\t\t0 skipped\n" +
		"\n\t\tTest execution time : 0.000s\n"

	if got := sb.String(); got != want {
		t.Errorf("console report mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestWriteConsoleReport_MatchesSkipDependencyFailureGolden reproduces
// cli/cmd/testdata/golden-output/skip-dependency-failure.txt byte for byte —
// a plain (non-data-driven) test failure, whose notice uses an empty suffix
// ("testDependency: has failed." — colon present, nothing after it), plus
// one skipped dependent.
func TestWriteConsoleReport_MatchesSkipDependencyFailureGolden(t *testing.T) {
	report := RunReport{
		Results: []Result{
			{Name: "testDependency", Status: Fail, Message: "dependency setup failed"},
			{Name: "testDependent", Status: Skip},
		},
		Events: []ConsoleEvent{
			{TestFailure: &TestFailureNotice{Name: "testDependency", Suffix: "", HasSuffix: true}},
		},
	}

	var sb strings.Builder
	WriteConsoleReport(&sb, report, false, 0)

	want := "\n\ttestDependency: has failed.\n\n" +
		"\n\t\t[fail] testDependency:\n" +
		"\n\t\t    dependency setup failed\n\t\t\t\n\t\t\t\n" +
		"\n\n" +
		"\t\t0 passing\n\t\t1 failing\n\t\t1 skipped\n" +
		"\n\t\tTest execution time : 0.000s\n"

	if got := sb.String(); got != want {
		t.Errorf("console report mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestWriteConsoleReport_MatchesBeforeSuiteFailureGolden reproduces
// cli/cmd/testdata/golden-output/skip-beforesuite-failure.txt byte for byte
// — a hook failure (not a test failure at all: the one test was simply
// skipped), confirming the "\t[fail] name[label]:" format and that the
// skip itself produces no console line (only a count).
func TestWriteConsoleReport_MatchesBeforeSuiteFailureGolden(t *testing.T) {
	hf := HookFailure{Hook: "before test suite function", Message: "beforeSuiteFails\nbeforeSuite failed"}
	report := RunReport{
		Results:      []Result{{Name: "testA", Status: Skip}},
		HookFailures: []HookFailure{hf},
		Events:       []ConsoleEvent{{Hook: &hf}},
	}

	var sb strings.Builder
	WriteConsoleReport(&sb, report, false, 0)

	want := "\t[fail] beforeSuiteFails[before test suite function]:\n\t    beforeSuite failed\n\t\t\n\t\t\n" +
		"\n\n" +
		"\t\t0 passing\n\t\t0 failing\n\t\t1 skipped\n" +
		"\n\t\tTest execution time : 0.000s\n"

	if got := sb.String(); got != want {
		t.Errorf("console report mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteConsoleReport_NoTestsFound(t *testing.T) {
	var sb strings.Builder
	WriteConsoleReport(&sb, RunReport{}, false, 0)

	want := "\n\n\t\tNo tests found\n\n\t\tTest execution time : 0.000s\n"
	if got := sb.String(); got != want {
		t.Errorf("console report mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteConsoleReport_SystemConsoleSuppressesPassLines(t *testing.T) {
	report := RunReport{Results: []Result{{Name: "testA", Status: Pass}}}

	var sb strings.Builder
	WriteConsoleReport(&sb, report, true, 0)

	if strings.Contains(sb.String(), "[pass]") {
		t.Errorf("expected no [pass] line when isSystemConsole is true, got %q", sb.String())
	}
}

// TestWriteConsoleReport_InterleavesHookAndTestFailuresChronologically
// confirms Events — not two separately-batched lists — drives the live
// notice ordering: a test's own body failing, then its AfterGroups hook
// also failing (both can happen within one executeOne call), must print in
// that same order, not all-hooks-then-all-tests.
func TestWriteConsoleReport_InterleavesHookAndTestFailuresChronologically(t *testing.T) {
	hf := HookFailure{Hook: "after test group function for the test", Message: "afterG\nafterG failed"}
	report := RunReport{
		Results: []Result{{Name: "testA", Status: Fail, Message: "testA failed"}},
		Events: []ConsoleEvent{
			{TestFailure: &TestFailureNotice{Name: "testA", HasSuffix: true}},
			{Hook: &hf},
		},
	}

	var sb strings.Builder
	WriteConsoleReport(&sb, report, false, 0)

	testIdx := strings.Index(sb.String(), "testA: has failed")
	hookIdx := strings.Index(sb.String(), "[fail] afterG[")
	if testIdx < 0 || hookIdx < 0 || testIdx > hookIdx {
		t.Errorf("expected the test failure notice before the hook failure notice, got %q", sb.String())
	}
}

func TestSummarize(t *testing.T) {
	report := RunReport{
		Results: []Result{
			{Status: Pass}, {Status: Pass}, {Status: Fail}, {Status: Skip},
		},
	}
	s := Summarize(report)
	if s.Passed != 2 || s.Failed != 1 || s.Skipped != 1 {
		t.Errorf("got %+v, want Passed=2 Failed=1 Skipped=1", s)
	}
	if s.ExitCode != 1 {
		t.Errorf("expected ExitCode 1 (a test failed), got %d", s.ExitCode)
	}
}

func TestSummarize_ExitCodeFromHookFailureAlone(t *testing.T) {
	report := RunReport{
		Results:      []Result{{Status: Pass}},
		HookFailures: []HookFailure{{Hook: "before test suite function", Message: "f\nmsg"}},
	}
	s := Summarize(report)
	if s.ExitCode != 1 {
		t.Errorf("expected ExitCode 1 (a hook failed, even though no test did), got %d", s.ExitCode)
	}
}

func TestSummarize_CleanRunExitsZero(t *testing.T) {
	report := RunReport{Results: []Result{{Status: Pass}, {Status: Skip}}}
	if s := Summarize(report); s.ExitCode != 0 {
		t.Errorf("expected ExitCode 0, got %d", s.ExitCode)
	}
}

func TestFormatSeconds(t *testing.T) {
	if got := formatSeconds(1500 * time.Millisecond); got != "1.500" {
		t.Errorf("formatSeconds(1.5s) = %q, want %q", got, "1.500")
	}
}
