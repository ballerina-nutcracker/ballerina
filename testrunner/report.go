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

// Package-level reporter, a Go port of report.bal — console output
// (consoleReport, plus the live notices execute.bal prints as it runs) and
// the two JSON artifacts (statusreport.go: moduleStatusReport,
// failedTestsReport), reading directly from a RunReport instead of
// report.bal's own ReportData/Result classes. Byte-for-byte format verified
// directly against real golden fixtures (cli/cmd/testdata/golden-output),
// not reconstructed from reading report.bal alone — see report_test.go.
package testrunner

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Summary is RunReport's aggregate pass/fail/skip counts and overall exit
// code — execute.bal#startSuite's exitCode is 1 if any test or hook failed,
// 0 otherwise; every enableExit() call site in the original corresponds to
// either a Fail Result or a HookFailure here, so both are derivable instead
// of needing their own tracked flag.
type Summary struct {
	Passed, Failed, Skipped int
	ExitCode                int
}

// Summarize computes Summary from a RunReport without printing anything.
func Summarize(report RunReport) Summary {
	var s Summary
	for _, r := range report.Results {
		switch r.Status {
		case Pass:
			s.Passed++
		case Fail:
			s.Failed++
		case Skip:
			s.Skipped++
		}
	}
	if s.Failed > 0 || len(report.HookFailures) > 0 {
		s.ExitCode = 1
	}
	return s
}

// WriteConsoleReport mirrors execute.bal#startSuite's console output byte
// for byte:
//  1. report.Events, replayed in the exact chronological order Run recorded
//     them — the old engine prints a hook failure (printExecutionError) or a
//     test/row failure's "has failed" notice (handleTestFuncOutput/
//     handleDataDrivenTestOutput/isTestReadyToExecute) live, as execution
//     reaches it; Run stays pure (no I/O), so the reporter replays the
//     recorded stream instead. A single executeOne call can interleave both
//     kinds (e.g. a test's own body fails, then its AfterGroups hook also
//     fails), which is exactly why this needs one ordered stream rather than
//     printing all hook failures and then all test failures.
//  2. [pass] lines — only when !isSystemConsole, matching
//     report.bal#consoleReport's isSystemConsole() gate (ballerina/test
//     hides them on an interactive terminal, keeps them for piped/CI output).
//  3. [fail] lines with each one's formatted message.
//  4. The passing/failing/skipped counts, or "No tests found" if nothing ran
//     (this is consoleReport's own empty check; execute.bal#startSuite has a
//     separate, earlier "No tests found" — single-tab, no exec-time line
//     at all — for zero discovered tests, which a caller is expected to
//     have already handled before ever calling Run, the same way
//     cli/cmd/test.go already checks testplan.TestPlan.HasAny()).
//  5. The execution time line.
//
// Passes and fails are printed as two separate groups (every pass, in its
// own relative order, then every fail, in its own relative order) rather
// than interleaved in overall execution order — ported exactly from
// ReportData storing them in two separate arrays (passed/failed) that
// consoleReport iterates one after the other.
func WriteConsoleReport(w io.Writer, report RunReport, isSystemConsole bool, executionTime time.Duration) {
	for _, ev := range report.Events {
		writeConsoleEvent(w, ev)
	}

	if !isSystemConsole {
		for _, r := range resultsByStatus(report.Results, Pass) {
			_, _ = fmt.Fprintln(w, "\t\t[pass] "+r.Name)
		}
	}
	for _, r := range resultsByStatus(report.Results, Fail) {
		_, _ = fmt.Fprintln(w, "\n\t\t[fail] "+r.Name+":")
		// "+\n": see writeHookFailure's comment — reproduces getErrorMessage's
		// trailing newline without storing it in Result.Message itself.
		// Skipped for RawMessage — see Result.RawMessage's own doc comment.
		message := r.Message
		if !r.RawMessage {
			message += "\n"
		}
		_, _ = fmt.Fprintln(w, "\n\t\t    "+formatFailedError(message, 3))
	}

	summary := Summarize(report)
	_, _ = fmt.Fprint(w, "\n\n") // two real newlines, see writeTestFailureNotice's comment
	if summary.Passed+summary.Failed+summary.Skipped == 0 {
		_, _ = fmt.Fprintln(w, "\t\tNo tests found")
	} else {
		_, _ = fmt.Fprintf(w, "\t\t%d passing\n", summary.Passed)
		_, _ = fmt.Fprintf(w, "\t\t%d failing\n", summary.Failed)
		_, _ = fmt.Fprintf(w, "\t\t%d skipped\n", summary.Skipped)
	}
	_, _ = fmt.Fprintf(w, "\n\t\tTest execution time : %ss\n", formatSeconds(executionTime))
}

func writeConsoleEvent(w io.Writer, ev ConsoleEvent) {
	switch {
	case ev.Hook != nil:
		writeHookFailure(w, *ev.Hook)
	case ev.TestFailure != nil:
		writeTestFailureNotice(w, *ev.TestFailure)
	}
}

// writeHookFailure mirrors execute.bal#printExecutionError. A HookFailure's
// Message is always "functionName\nerrorMessage" (set that way by
// invokeHook, matching executeFunction's own ExecutionError construction),
// so splitting on the first newline recovers both parts. The "+\n" before
// formatFailedError reproduces getErrorMessage's trailing newline, which the
// original bakes into the stored error message itself; this port keeps
// HookFailure.Message free of that synthetic newline and adds it back only
// here, at format time, rather than polluting the stored value.
func writeHookFailure(w io.Writer, hf HookFailure) {
	name, message := hf.Message, ""
	if idx := strings.IndexByte(hf.Message, '\n'); idx >= 0 {
		name, message = hf.Message[:idx], hf.Message[idx+1:]
	}
	_, _ = fmt.Fprintln(w, "\t[fail] "+name+"["+hf.Hook+"]:\n\t    "+formatFailedError(message+"\n", 2))
}

// writeTestFailureNotice mirrors handleTestFuncOutput/
// handleDataDrivenTestOutput's inline println (HasSuffix true: a colon
// always present, empty for a general test, the row key for a data-driven
// row) and isTestReadyToExecute's diagnostics-failure println (HasSuffix
// false: no colon at all).
func writeTestFailureNotice(w io.Writer, n TestFailureNotice) {
	// fmt.Fprint, not Fprintln: the trailing "\n\n" is two real, distinct
	// newlines confirmed against a golden fixture (the notice's own
	// template newline, plus println's implicit one in the original) — not
	// a mistaken double newline for vet to flag.
	if n.HasSuffix {
		_, _ = fmt.Fprint(w, "\n\t"+n.Name+":"+n.Suffix+" has failed.\n\n")
	} else {
		_, _ = fmt.Fprint(w, "\n\t"+n.Name+" has failed.\n\n")
	}
}

// formatFailedError ports report.bal#formatFailedError: split the message on
// "\n", append a trailing empty line, then rejoin indented by tabCount tabs.
func formatFailedError(message string, tabCount int) string {
	lines := append(strings.Split(message, "\n"), "")
	return strings.Join(lines, "\n"+strings.Repeat("\t", tabCount))
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}
