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

package test_util

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

// These are unit tests because they cover cases a corpus test can't express: the marker
// parser is test infrastructure that corpus tests can't target in isolation, and the
// assertion helper tests need multi-line warnings, warnings located in another file, and
// the helpers' own failure reporting.

func TestParseMarkers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		content string
		want    []Marker
	}{
		{
			name:    "single markers of each kind",
			content: "io:println(1); // @output 1\nint x = 1; // @warn\nint y = a / b; // @panic\n// @error\n",
			want: []Marker{
				{Line: 1, Kind: MarkerOutput, Value: "1"},
				{Line: 2, Kind: markerWarn},
				{Line: 3, Kind: MarkerPanic},
				{Line: 4, Kind: MarkerError},
			},
		},
		{
			name:    "single markers of each kind on comment-only lines",
			content: "// @output 1\n// @warn\n// @panic\nx; // @error\n",
			want: []Marker{
				{Line: 1, Kind: MarkerOutput, Value: "1"},
				{Line: 2, Kind: markerWarn},
				{Line: 3, Kind: MarkerPanic},
				{Line: 4, Kind: MarkerError},
			},
		},
		{
			name:    "output and warn",
			content: "x; // @output 3 // @warn",
			want:    []Marker{{Line: 1, Kind: MarkerOutput, Value: "3"}, {Line: 1, Kind: markerWarn}},
		},
		{
			name:    "panic and warn",
			content: "x; // @panic // @warn",
			want:    []Marker{{Line: 1, Kind: MarkerPanic}, {Line: 1, Kind: markerWarn}},
		},
		{
			name:    "error and warn",
			content: "x; // @error // @warn",
			want:    []Marker{{Line: 1, Kind: MarkerError}, {Line: 1, Kind: markerWarn}},
		},
		{
			name:    "two errors",
			content: "x; // @error LHS // @error RHS",
			want:    []Marker{{Line: 1, Kind: MarkerError}, {Line: 1, Kind: MarkerError}},
		},
		{
			name:    "two outputs in source order",
			content: "x; // @output a // @output b",
			want:    []Marker{{Line: 1, Kind: MarkerOutput, Value: "a"}, {Line: 1, Kind: MarkerOutput, Value: "b"}},
		},
		{
			name:    "output containing a url",
			content: "x; // @output http://www.w3.org/2000/xmlns/",
			want:    []Marker{{Line: 1, Kind: MarkerOutput, Value: "http://www.w3.org/2000/xmlns/"}},
		},
		{
			name:    "output containing a url followed by warn",
			content: "x; // @output {http://a}b // @warn",
			want:    []Marker{{Line: 1, Kind: MarkerOutput, Value: "{http://a}b"}, {Line: 1, Kind: markerWarn}},
		},
		{
			name:    "output trimming",
			content: "x; // @output   a b \t\ny; // @output\nz; //@output x\r",
			want: []Marker{
				{Line: 1, Kind: MarkerOutput, Value: "  a b"},
				{Line: 2, Kind: MarkerOutput, Value: ""},
				{Line: 3, Kind: MarkerOutput, Value: "x"},
			},
		},
		{
			name:    "lookalike keywords are not markers",
			content: "x; // @warning\ny; // @warns\nz; // @errors",
			want:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ParseMarkers(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseMarkers(%q) = %+v, want %+v", tc.content, got, tc.want)
			}
		})
	}
}

func TestWarnMarkerLines(t *testing.T) {
	t.Parallel()
	content := "a\nb; // @warn\nc; // @warn // @warn\nd; // @output 1\ne; // @warn"
	got := warnMarkerLines(content)
	want := []int{2, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("warnMarkerLines() = %v, want %v", got, want)
	}
}

type recordingReporter struct {
	failures []string
}

func (r *recordingReporter) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

const (
	markerTestFile  = "main.bal"
	markerOtherFile = "other.bal"
)

type warnFixture struct {
	de      *diagnostics.DiagnosticEnv
	content string
}

func newWarnFixture(content string) *warnFixture {
	de := diagnostics.NewDiagnosticEnv()
	de.RegisterFile(markerTestFile, text.NewStringTextDocument(content))
	de.RegisterFile(markerOtherFile, text.NewStringTextDocument(content))
	return &warnFixture{de: de, content: content}
}

// warningSpanning builds a warning in fileName spanning from the start of startLine to the end of endLine (1-based).
func (f *warnFixture) warningSpanning(fileName string, startLine, endLine int) diagnostics.Diagnostic {
	lines := strings.SplitAfter(f.content, "\n")
	start := 0
	for _, l := range lines[:startLine-1] {
		start += len(l)
	}
	end := start
	for _, l := range lines[startLine-1 : endLine] {
		end += len(l)
	}
	code := "WARNING"
	info := diagnostics.NewDiagnosticInfo(&code, "unused variable 'x'", diagnostics.Warning)
	return diagnostics.CreateDiagnostic(info, diagnostics.NewLocation(f.de, fileName, start, end-1))
}

func (f *warnFixture) warningAt(line int) diagnostics.Diagnostic {
	return f.warningSpanning(markerTestFile, line, line)
}

func assertFailures(t *testing.T, r *recordingReporter, want ...string) {
	t.Helper()
	if len(r.failures) != len(want) {
		t.Fatalf("failures = %q, want %d failures containing %q", r.failures, len(want), want)
	}
	for i, w := range want {
		if !strings.Contains(r.failures[i], w) {
			t.Errorf("failure %d = %q, want it to contain %q", i, r.failures[i], w)
		}
	}
}

func TestAssertWarnMarkers(t *testing.T) {
	t.Parallel()
	t.Run("warning on marked line", func(t *testing.T) {
		t.Parallel()
		f := newWarnFixture("a;\nint x = 1; // @warn\nb;\n")
		r := &recordingReporter{}
		AssertWarnMarkers(r, markerTestFile, f.content, []diagnostics.Diagnostic{f.warningAt(2)}, f.de)
		assertFailures(t, r)
	})
	t.Run("warning on unmarked line", func(t *testing.T) {
		t.Parallel()
		f := newWarnFixture("a;\nint x = 1;\nb;\n")
		r := &recordingReporter{}
		AssertWarnMarkers(r, markerTestFile, f.content, []diagnostics.Diagnostic{f.warningAt(2)}, f.de)
		assertFailures(t, r, "warning at main.bal:2 not covered by any @warn marker")
	})
	t.Run("marker without warning", func(t *testing.T) {
		t.Parallel()
		f := newWarnFixture("a;\nint x = 1; // @warn\nb;\n")
		r := &recordingReporter{}
		AssertWarnMarkers(r, markerTestFile, f.content, nil, f.de)
		assertFailures(t, r, "@warn marker at main.bal:2 not covered by any warning")
	})
	t.Run("marker on a later line of a multi-line warning", func(t *testing.T) {
		t.Parallel()
		f := newWarnFixture("a;\nint x =\n  1; // @warn\nb;\n")
		r := &recordingReporter{}
		AssertWarnMarkers(r, markerTestFile, f.content, []diagnostics.Diagnostic{f.warningSpanning(markerTestFile, 2, 3)}, f.de)
		assertFailures(t, r,
			"@warn marker at main.bal:3 not covered by any warning",
			"warning at main.bal:2 not covered by any @warn marker")
	})
	t.Run("warning in another file on the marked line", func(t *testing.T) {
		t.Parallel()
		f := newWarnFixture("a;\nint x = 1; // @warn\nb;\n")
		r := &recordingReporter{}
		AssertWarnMarkers(r, markerTestFile, f.content, []diagnostics.Diagnostic{f.warningSpanning(markerOtherFile, 2, 2)}, f.de)
		assertFailures(t, r, "@warn marker at main.bal:2 not covered by any warning")
	})
}

func TestAssertWarnMarkersCovered(t *testing.T) {
	t.Parallel()
	t.Run("warning on unmarked line", func(t *testing.T) {
		t.Parallel()
		f := newWarnFixture("a;\nint x = 1;\nb;\n")
		r := &recordingReporter{}
		AssertWarnMarkersCovered(r, markerTestFile, f.content, []diagnostics.Diagnostic{f.warningAt(2)}, f.de)
		assertFailures(t, r)
	})
	t.Run("marker without warning", func(t *testing.T) {
		t.Parallel()
		f := newWarnFixture("a;\nint x = 1; // @warn\nb;\n")
		r := &recordingReporter{}
		AssertWarnMarkersCovered(r, markerTestFile, f.content, nil, f.de)
		assertFailures(t, r, "@warn marker at main.bal:2 not covered by any warning")
	})
}
