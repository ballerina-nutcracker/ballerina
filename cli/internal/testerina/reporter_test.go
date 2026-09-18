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
	"bytes"
	"testing"
)

// The corpus harness runs bal over pipes, so it only ever exercises the plain
// reporter; the TTY progress line has to be covered here.
func TestTTYReporterRewritesOneProgressLine(t *testing.T) {
	module := ModuleKey{Org: "acme", Module: "app"}
	results := []Result{
		{Step: Step{Module: module, Test: TestFunction{Name: "testOne"}}, Outcome: Passed},
		{
			Step:     Step{Module: module, Test: TestFunction{Name: "testTwo"}},
			Outcome:  Failed,
			Messages: []string{"boom"},
			Stdout:   []byte("captured\n\nafter a blank line\n"),
		},
		{Step: Step{Module: module, Test: TestFunction{Name: "testThree"}}, Outcome: Errored, Messages: []string{"x"}},
		{Step: Step{Module: module, Test: TestFunction{Name: "testFour"}}, Outcome: Skipped},
	}

	var buf bytes.Buffer
	reporter := NewTTYReporter(&buf)
	reporter.Begin(len(results))
	for _, result := range results {
		reporter.Report(result)
	}
	reporter.End(Summary{Results: results})

	want := "\rRunning 1/4  passed 1  failed 0  errored 0  skipped 0" +
		"\rRunning 2/4  passed 1  failed 1  errored 0  skipped 0" +
		"\rRunning 3/4  passed 1  failed 1  errored 1  skipped 0" +
		"\rRunning 4/4  passed 1  failed 1  errored 1  skipped 1" +
		"\n" +
		"app:testTwo\n    boom\n    captured\n\n    after a blank line\n" +
		"app:testThree\n    x\n" +
		"1 passing, 1 failing, 1 errored, 1 skipped\n"
	if got := buf.String(); got != want {
		t.Fatalf("tty reporter output:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestPlainReporterPrintsOneLinePerResult(t *testing.T) {
	module := ModuleKey{Org: "acme", Module: "app.sub"}
	results := []Result{
		{Step: Step{Module: module, Test: TestFunction{Name: "testOne"}}, Outcome: Passed},
		{Step: Step{Module: module, Test: TestFunction{Name: "testTwo"}}, Outcome: Skipped},
	}

	var buf bytes.Buffer
	reporter := NewPlainReporter(&buf)
	reporter.Begin(len(results))
	for _, result := range results {
		reporter.Report(result)
	}
	reporter.End(Summary{Results: results, RunFailures: []string{"first", "second"}})

	want := "PASS sub:testOne\nSKIP sub:testTwo\n" +
		"<run>\n    first\n    second\n" +
		"1 passing, 0 failing, 0 errored, 1 skipped\n"
	if got := buf.String(); got != want {
		t.Fatalf("plain reporter output:\ngot:  %q\nwant: %q", got, want)
	}
}
