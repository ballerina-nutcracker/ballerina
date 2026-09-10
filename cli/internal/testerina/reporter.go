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

package testerina

import (
	"fmt"
	"io"
	"strings"
)

// Reporter renders test progress and the final summary.
type Reporter interface {
	Begin(total int)
	Report(Result)
	End(Summary)
}

func NewPlainReporter(w io.Writer) Reporter {
	return &plainReporter{reporterBase: reporterBase{w: w}}
}

func NewTTYReporter(w io.Writer) Reporter {
	return &ttyReporter{reporterBase: reporterBase{w: w}}
}

// reporterBase holds the state and summary rendering both reporters share.
type reporterBase struct {
	w     io.Writer
	total int
}

func (r *reporterBase) Begin(total int) {
	r.total = total
}

func (r *reporterBase) End(summary Summary) {
	var passed, failed, errored, skipped int
	for _, result := range summary.Results {
		switch result.Outcome {
		case Passed:
			passed++
		case Failed:
			failed++
		case Errored:
			errored++
		case Skipped:
			skipped++
		}
		if result.Outcome == Failed || result.Outcome == Errored {
			r.printFailure(result)
		}
	}
	if len(summary.RunFailures) > 0 {
		_, _ = fmt.Fprintf(r.w, "%s\n", stepLabel(Step{}))
		for _, message := range summary.RunFailures {
			r.printIndented(message)
		}
	}
	_, _ = fmt.Fprintf(r.w, "%d passing, %d failing, %d errored, %d skipped\n",
		passed, failed, errored, skipped)
}

func (r *reporterBase) printFailure(result Result) {
	_, _ = fmt.Fprintf(r.w, "%s\n", stepLabel(result.Step))
	for _, message := range result.Messages {
		r.printIndented(message)
	}
	r.printIndented(string(result.Stdout))
}

func (r *reporterBase) printIndented(text string) {
	if text == "" {
		return
	}
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if line == "" {
			_, _ = fmt.Fprintln(r.w)
			continue
		}
		_, _ = fmt.Fprintf(r.w, "    %s\n", line)
	}
}

func stepLabel(step Step) string {
	if step.Test.Name == "" {
		return "<run>"
	}
	return step.Label()
}

func outcomeLabel(outcome Outcome) string {
	switch outcome {
	case Passed:
		return "PASS"
	case Failed:
		return "FAIL"
	case Errored:
		return "ERROR"
	default:
		return "SKIP"
	}
}

// plainReporter prints one line per result; used when stdout is not a terminal.
type plainReporter struct {
	reporterBase
}

func (r *plainReporter) Report(result Result) {
	_, _ = fmt.Fprintf(r.w, "%s %s\n", outcomeLabel(result.Outcome), stepLabel(result.Step))
}

// ttyReporter rewrites a single progress line in place.
type ttyReporter struct {
	reporterBase
	done, passed, failed, errored, skipped int
}

func (r *ttyReporter) Report(result Result) {
	r.done++
	switch result.Outcome {
	case Passed:
		r.passed++
	case Failed:
		r.failed++
	case Errored:
		r.errored++
	case Skipped:
		r.skipped++
	}
	_, _ = fmt.Fprintf(r.w, "\rRunning %d/%d  passed %d  failed %d  errored %d  skipped %d",
		r.done, r.total, r.passed, r.failed, r.errored, r.skipped)
}

func (r *ttyReporter) End(summary Summary) {
	if r.done > 0 {
		_, _ = fmt.Fprintln(r.w)
	}
	r.reporterBase.End(summary)
}
