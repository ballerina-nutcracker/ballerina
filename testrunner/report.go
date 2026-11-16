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
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Summary is RunReport's aggregate pass/fail/skip counts and overall exit
// code. ExitCode is 1 if any test or hook failed, 0 otherwise.
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

// WriteConsoleReport prints the end-of-suite report: [pass] lines (only when
// isTerminal is false), [fail] lines with their messages, the counts and
// the execution time. Run has already printed the "has failed" notices.
func WriteConsoleReport(w io.Writer, report RunReport, isTerminal bool, executionTime time.Duration) {
	if !isTerminal {
		for _, r := range resultsByStatus(report.Results, Pass) {
			_, _ = fmt.Fprintln(w, "\t\t[pass] "+r.Name)
			// An evaluation test prints its pass-rate message even when it
			// passed — the only TestType that does.
			if r.Type == EvalTest {
				_, _ = fmt.Fprintln(w, "\n\t\t    "+formatFailedError(r.Message, 3))
			}
		}
	}
	for _, r := range resultsByStatus(report.Results, Fail) {
		_, _ = fmt.Fprintln(w, "\n\t\t[fail] "+r.Name+":")
		// RawMessage messages print as is; see Result.RawMessage's doc.
		message := r.Message
		if !r.RawMessage {
			message += "\n"
		}
		_, _ = fmt.Fprintln(w, "\n\t\t    "+formatFailedError(message, 3))
		if r.Type == EvalTest {
			writeEvalReport(w, r.Eval)
		}
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

// writeHookFailure prints a hook failure notice. A HookFailure's Message is
// always "functionName\nerrorMessage", so splitting on the first newline
// recovers both parts.
func writeHookFailure(w io.Writer, hf HookFailure) {
	name, message := hf.Message, ""
	if idx := strings.IndexByte(hf.Message, '\n'); idx >= 0 {
		name, message = hf.Message[:idx], hf.Message[idx+1:]
	}
	_, _ = fmt.Fprintln(w, "\t[fail] "+name+"["+hf.Hook+"]:\n\t    "+formatFailedError(message+"\n", 2))
}

// writeTestFailureNotice prints a "has failed" notice: with a colon and
// suffix (empty for a general test, the row key for a data-driven row) when
// HasSuffix is true, or without a colon at all when it's false.
func writeTestFailureNotice(w io.Writer, n TestFailureNotice) {
	// fmt.Fprint, not Fprintln: the trailing "\n\n" is two deliberate newlines.
	if n.HasSuffix {
		_, _ = fmt.Fprint(w, "\n\t"+n.Name+":"+n.Suffix+" has failed.\n\n")
	} else {
		_, _ = fmt.Fprint(w, "\n\t"+n.Name+" has failed.\n\n")
	}
}

// writeEvalReport prints the per-run breakdown after a failing evaluation
// test's own summary message — one line per run for a non-data-driven
// evaluation (run id + message), one run-header line plus one line per
// data-provider row for a data-driven one. Outcomes == nil distinguishes the
// two.
func writeEvalReport(w io.Writer, eval *EvalSummary) {
	if eval == nil {
		return
	}
	_, _ = fmt.Fprintln(w, "\t\t\t    evaluation runs:")
	for _, run := range eval.RunResults {
		if run.Outcomes == nil {
			_, _ = fmt.Fprintln(w, "\n\t\t\t\t    run: "+strconv.Itoa(run.ID)+
				"\n\t\t\t\t\t    message: "+getConsoleMessage(run.ErrorMessage, "\n\t\t\t\t\t\t\t"))
			continue
		}
		_, _ = fmt.Fprintln(w, "\n\t\t\t\t    run: "+strconv.Itoa(run.ID))
		for _, outcome := range run.Outcomes {
			_, _ = fmt.Fprintln(w, "\n\t\t\t\t\t    entry: "+outcome.ID+
				"\n\t\t\t\t\t    message: "+getConsoleMessage(outcome.ErrorMessage, "\n\t\t\t\t\t\t\t"))
		}
	}
}

// getConsoleMessage returns "passed" for an empty message (that run/entry
// passed); otherwise each embedded newline is reindented to indent.
func getConsoleMessage(message, indent string) string {
	if message == "" {
		return "passed"
	}
	return strings.ReplaceAll(message, "\n", indent)
}

// formatFailedError splits message on "\n", appends a trailing empty line,
// then rejoins indented by tabCount tabs.
func formatFailedError(message string, tabCount int) string {
	lines := append(strings.Split(message, "\n"), "")
	return strings.Join(lines, "\n"+strings.Repeat("\t", tabCount))
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}
