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
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// MarkerKind identifies a corpus test marker such as `// @output`.
type MarkerKind string

const (
	MarkerOutput MarkerKind = "output"
	MarkerError  MarkerKind = "error"
	MarkerPanic  MarkerKind = "panic"
	markerWarn   MarkerKind = "warn"
)

// Marker is a single corpus test marker. Line is 1-based and Value is only set for MarkerOutput.
type Marker struct {
	Line  int
	Kind  MarkerKind
	Value string
}

var markerRe = regexp.MustCompile(`//\s*@(output|error|panic|warn)\b`)

// ParseMarkers returns every marker in content in source order. A line may carry
// several markers; each runs until the next marker on the line or end of line.
func ParseMarkers(content string) []Marker {
	var out []Marker
	for i, raw := range strings.Split(content, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		commentIdx := strings.Index(line, "//")
		if commentIdx < 0 {
			continue
		}
		comment := line[commentIdx:]
		locs := markerRe.FindAllStringSubmatchIndex(comment, -1)
		for j, loc := range locs {
			end := len(comment)
			if j+1 < len(locs) {
				end = locs[j+1][0]
			}
			kind := MarkerKind(comment[loc[2]:loc[3]])
			value := ""
			if kind == MarkerOutput {
				value = strings.TrimRight(strings.TrimPrefix(comment[loc[1]:end], " "), " \t")
			}
			out = append(out, Marker{Line: i + 1, Kind: kind, Value: value})
		}
	}
	return out
}

func warnMarkerLines(content string) []int {
	var lines []int
	for _, m := range ParseMarkers(content) {
		if m.Kind == markerWarn {
			lines = append(lines, m.Line)
		}
	}
	slices.Sort(lines)
	return slices.Compact(lines)
}

func warningLines(inputPath string, warnings []diagnostics.Diagnostic, de *diagnostics.DiagnosticEnv) map[int]bool {
	lines := map[int]bool{}
	for _, w := range warnings {
		loc := w.Location()
		if !diagnostics.LocationHasSource(loc) || de.FileName(loc) != inputPath {
			continue
		}
		lines[de.StartLine(loc)+1] = true
	}
	return lines
}

// AssertWarnMarkersCovered fails for every `@warn` line in content that is not the
// start line of a warning located in inputPath.
func AssertWarnMarkersCovered(r FailReporter, inputPath, content string, warnings []diagnostics.Diagnostic, de *diagnostics.DiagnosticEnv) {
	assertWarnMarkersCovered(r, inputPath, warnMarkerLines(content), warningLines(inputPath, warnings, de))
}

// AssertWarnMarkers checks `@warn` markers against warnings located in inputPath in
// both directions: every marker needs a warning and every warning needs a marker.
func AssertWarnMarkers(r FailReporter, inputPath, content string, warnings []diagnostics.Diagnostic, de *diagnostics.DiagnosticEnv) {
	markerLines := warnMarkerLines(content)
	lines := warningLines(inputPath, warnings, de)
	assertWarnMarkersCovered(r, inputPath, markerLines, lines)
	markers := map[int]bool{}
	for _, m := range markerLines {
		markers[m] = true
	}
	for _, line := range slices.Sorted(maps.Keys(lines)) {
		if !markers[line] {
			r.Errorf("warning at %s:%d not covered by any @warn marker", inputPath, line)
		}
	}
}

func assertWarnMarkersCovered(r FailReporter, inputPath string, markerLines []int, lines map[int]bool) {
	for _, m := range markerLines {
		if !lines[m] {
			r.Errorf("@warn marker at %s:%d not covered by any warning", inputPath, m)
		}
	}
}
