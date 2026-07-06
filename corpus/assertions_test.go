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

package corpus

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"ballerina-lang-go/test_util"
)

// resolvedDiag is a flattened, file/line-resolved view of one error
// diagnostic produced by the compiler. Lines are 1-based, inclusive on
// both ends.
type resolvedDiag struct {
	file      string // matches the key used when registering the file
	startLine int
	endLine   int
}

// baseLine is a (basename, line) pair used to compare a panic annotation
// against the file:line tokens in a runtime stack trace.
type baseLine struct {
	base string
	line int
}

func (b baseLine) String() string {
	return fmt.Sprintf("%s:%d", b.base, b.line)
}

// assertAnnotations dispatches to the correct annotation check based on the
// test name suffix (`-v.bal` / `-e.bal` / `-p.bal` for single-file, or
// `-v` / `-e` / `-p` for projects). It is invoked inline from the same
// `t.Run` that already validates the txtar snapshot, so failures from both
// surface together.
func assertAnnotations(t *testing.T, sources []annSourceFile, name, stdout, stderr string, diags []resolvedDiag) {
	t.Helper()
	anns, err := parseAnnotations(sources)
	if err != nil {
		t.Errorf("failed to parse annotations: %v", err)
		return
	}

	switch {
	case hasAnySuffix(name, "-fv.bal", "-fe.bal", "-fp.bal", "-fv", "-fe", "-fp"):
		// Future tests document a current `fatal[...]` bailout from the
		// front-end. The exact diagnostics are not annotation-checked because
		// the compiler hasn't reached the point where it can attribute the
		// failure to a specific source location.
	case hasAnySuffix(name, "-v.bal", "-v"):
		assertOutputAnnotations(t, anns, stdout, stderr)
	case hasAnySuffix(name, "-e.bal", "-e"):
		assertErrorAnnotations(t, anns, diags)
	case hasAnySuffix(name, "-p.bal", "-p"):
		assertPanicAnnotations(t, anns, stdout, stderr)
	}
}

func hasAnySuffix(name string, suffixes ...string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

func assertOutputAnnotations(t *testing.T, anns annotations, stdout, stderr string) {
	t.Helper()

	if strings.TrimRight(stderr, "\n") != "" {
		t.Errorf("expected empty stderr for -v test, got:\n%s", stderr)
	}

	outputs, ok := singleOutputFile(t, anns)
	if !ok {
		return
	}

	if len(outputs) == 0 && strings.TrimRight(stdout, "\n") != "" {
		t.Errorf("test produced stdout but has no @output annotations:\n%s", stdout)
		return
	}

	expected := buildExpectedStdout(outputs)
	if stdout != expected {
		t.Errorf("@output annotation mismatch\n%s",
			test_util.FormatExpectedGot(expected, stdout))
	}
}

// singleOutputFile returns the ordered output annotations for the one file
// that carries them. If `@output` annotations are spread across more than one
// file the test fails (we cannot define a deterministic source order across
// modules).
func singleOutputFile(t *testing.T, anns annotations) ([]outputAnn, bool) {
	t.Helper()
	var owners []string
	for file, fa := range anns {
		if len(fa.outputs) > 0 {
			owners = append(owners, file)
		}
	}
	switch len(owners) {
	case 0:
		return nil, true
	case 1:
		return anns[owners[0]].outputs, true
	default:
		sort.Strings(owners)
		t.Errorf("@output annotations must live in a single file; found in: %s",
			strings.Join(owners, ", "))
		return nil, false
	}
}

func buildExpectedStdout(outputs []outputAnn) string {
	if len(outputs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, o := range outputs {
		b.WriteString(o.value)
		b.WriteByte('\n')
	}
	return b.String()
}

func assertErrorAnnotations(t *testing.T, anns annotations, diags []resolvedDiag) {
	t.Helper()

	if !hasErrorAnnotation(anns) {
		t.Errorf("-e test must have at least one @error annotation")
		return
	}

	if len(diags) == 0 {
		t.Errorf("-e test produced no error diagnostics")
		return
	}

	covered := make(map[string]map[int]bool)
	for _, d := range diags {
		if !diagnosticCoversAnnotatedLine(d, anns, covered) {
			t.Errorf("diagnostic at %s:%d-%d not covered by any @error annotation",
				d.file, d.startLine, d.endLine)
		}
	}

	for file, fa := range anns {
		for _, e := range fa.errors {
			if !covered[file][e.line] {
				t.Errorf("@error annotation at %s:%d is not covered by any diagnostic", file, e.line)
			}
		}
	}
}

func hasErrorAnnotation(anns annotations) bool {
	for _, fa := range anns {
		if len(fa.errors) > 0 {
			return true
		}
	}
	return false
}

func diagnosticCoversAnnotatedLine(d resolvedDiag, anns annotations, covered map[string]map[int]bool) bool {
	fa, ok := anns[d.file]
	if !ok {
		return false
	}
	matched := false
	for _, e := range fa.errors {
		if e.line >= d.startLine && e.line <= d.endLine {
			if covered[d.file] == nil {
				covered[d.file] = make(map[int]bool)
			}
			covered[d.file][e.line] = true
			matched = true
		}
	}
	return matched
}

// stackFrameLineRe matches a frame like `funcName(file.bal:42)` or
// `(file.bal:42)`. Captures the file (basename) and 1-based line.
var stackFrameLineRe = regexp.MustCompile(`\(([^()\s:]+):(\d+)\)`)

func parseStackFrames(stderr string) []baseLine {
	matches := stackFrameLineRe.FindAllStringSubmatch(stderr, -1)
	frames := make([]baseLine, 0, len(matches))
	for _, m := range matches {
		line, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		frames = append(frames, baseLine{base: m[1], line: line})
	}
	return frames
}

func assertPanicAnnotations(t *testing.T, anns annotations, stdout, stderr string) {
	t.Helper()

	want, ok := singlePanicAnnotation(t, anns)
	if !ok {
		return
	}
	outputs, ok := singleOutputFile(t, anns)
	if !ok {
		return
	}
	expectedStdout := buildExpectedStdout(outputs)
	if stdout != expectedStdout {
		t.Errorf("@output annotation mismatch for -p test\n%s",
			test_util.FormatExpectedGot(expectedStdout, stdout))
	}

	frames := parseStackFrames(stderr)
	if len(frames) == 0 {
		// FIXME: https://github.com/ballerina-platform/ballerina-lang-go/issues/477
		if !strings.Contains(stderr, "error:") {
			t.Errorf("stderr does not contain a runtime error message for @panic at %s\nstderr:\n%s", want, stderr)
		}
		return
	}
	if frames[0] != want {
		t.Errorf("top stack frame mismatch for @panic: want %s, got %s\nstderr:\n%s", want, frames[0], stderr)
	}
}

// singlePanicAnnotation enforces that the test has exactly one `@panic`
// annotation across all of its source files and returns its baseLine view.
func singlePanicAnnotation(t *testing.T, anns annotations) (baseLine, bool) {
	t.Helper()
	var (
		count int
		owner string
		ann   panicAnn
	)
	for file, fa := range anns {
		count += len(fa.panics)
		if len(fa.panics) > 0 {
			owner = file
			ann = fa.panics[0]
		}
	}
	if count != 1 {
		t.Errorf("-p test must have exactly one @panic annotation, found %d", count)
		return baseLine{}, false
	}
	return baseLine{base: filepath.Base(owner), line: ann.line}, true
}
