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

package native

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/values"
)

// assert.bal's own usage of sprintf is exclusively `sprintf("%s", v)` — the
// %d/%f/%b/%x specifiers are part of the native's contract but never
// exercised by any real ballerina/test code path, so corpus can't reach them
// (see CHECKPOINT.md's coverage entry). Covered directly here instead,
// matching this package's existing convention (see test_io_test.go) of
// filling gaps corpus genuinely can't reach.
func TestSprintf(t *testing.T) {
	cases := []struct {
		name   string
		format string
		args   []values.BalValue
		want   string
	}{
		{"string", "%s", []values.BalValue{"hello"}, "hello"},
		{"int", "%d", []values.BalValue{int64(42)}, "42"},
		{"float", "%f", []values.BalValue{3.5}, "3.5"},
		{"bool_lower", "%b", []values.BalValue{true}, "true"},
		{"bool_upper", "%B", []values.BalValue{false}, "FALSE"},
		{"hex_lower", "%x", []values.BalValue{int64(255)}, "ff"},
		{"hex_upper", "%X", []values.BalValue{int64(255)}, "FF"},
		{"literal_percent", "100%%", nil, "100%"},
		{"padded_string", "%10s", []values.BalValue{"hi"}, "        hi"},
		{"multiple_args", "%s=%d", []values.BalValue{"x", int64(5)}, "x=5"},
		{"no_format_specifiers", "plain text", nil, "plain text"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := append([]values.BalValue{c.format}, c.args...)
			got, err := sprintf(nil, args)
			if err != nil {
				t.Fatalf("sprintf(%q, %v): unexpected error: %v", c.format, c.args, err)
			}
			if got != c.want {
				t.Errorf("sprintf(%q, %v) = %q, want %q", c.format, c.args, got, c.want)
			}
		})
	}
}

func TestSprintf_Errors(t *testing.T) {
	cases := []struct {
		name   string
		format string
		args   []values.BalValue
	}{
		{"not_enough_args", "%s %s", []values.BalValue{"only one"}},
		{"wrong_type_for_d", "%d", []values.BalValue{"not an int"}},
		{"invalid_specifier", "%q", []values.BalValue{"x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := append([]values.BalValue{c.format}, c.args...)
			if _, err := sprintf(nil, args); err == nil {
				t.Errorf("sprintf(%q, %v): expected an error, got none", c.format, c.args)
			}
		})
	}
}

// getKeysDiff itself takes *values.List args (a concrete runtime type this
// package's tests otherwise avoid hand-constructing — see test_io_test.go's
// header comment); keysNotIn is the plain-Go helper underneath it and is
// tested directly instead. getKeysDiff's own BalValue-boundary behavior is
// covered end-to-end by the corpus tests (map-comparison paths reach it via
// assert.bal's getMapValueDiff).
func TestKeysNotIn(t *testing.T) {
	cases := []struct {
		name     string
		source   []string
		other    []string
		wantDiff []string
	}{
		{"some_missing", []string{"a", "b", "c"}, []string{"b"}, []string{"a", "c"}},
		{"none_missing", []string{"a", "b"}, []string{"a", "b", "c"}, nil},
		{"all_missing", []string{"a", "b"}, []string{"c"}, []string{"a", "b"}},
		{"empty_source", nil, []string{"a"}, nil},
		{"preserves_source_order", []string{"z", "a", "m"}, nil, []string{"z", "a", "m"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := keysNotIn(c.source, c.other)
			if len(got) != len(c.wantDiff) {
				t.Fatalf("keysNotIn(%v, %v) = %v, want %v", c.source, c.other, got, c.wantDiff)
			}
			for i := range got {
				if got[i] != c.wantDiff[i] {
					t.Errorf("keysNotIn(%v, %v) = %v, want %v", c.source, c.other, got, c.wantDiff)
					break
				}
			}
		})
	}
}

// TestChunkByLength guards against byte-based chunking splitting a multi-byte character in half.
func TestChunkByLength(t *testing.T) {
	s := strings.Repeat("あ", 90)
	chunks := chunkByLength(s, 80)
	if len(chunks) != 2 {
		t.Fatalf("chunkByLength returned %d chunks, want 2: %v", len(chunks), chunks)
	}
	for i, c := range chunks {
		if !utf8.ValidString(c) {
			t.Errorf("chunk %d is not valid UTF-8: %q", i, c)
		}
	}
	if got := strings.Join(chunks, ""); got != s {
		t.Errorf("chunks don't reconstruct the original string: got len %d, want len %d", len(got), len(s))
	}
	if got := utf8.RuneCountInString(chunks[0]); got != 80 {
		t.Errorf("first chunk has %d runes, want 80", got)
	}
}

// TestDiffLines checks ordinary diffs still get a proper LCS edit script, not just linear removals/additions.
func TestDiffLines(t *testing.T) {
	a := []string{"same1", "removed", "same2"}
	b := []string{"same1", "added", "same2"}
	got := diffLines(a, b)
	want := []diffOp{
		{' ', "same1"},
		{'-', "removed"},
		{'+', "added"},
		{' ', "same2"},
	}
	if len(got) != len(want) {
		t.Fatalf("diffLines(%v, %v) = %v, want %v", a, b, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("op %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestDiffLinesLargeInputFallsBackToLinear plants one shared line in large, otherwise-disjoint
// input: only a real LCS pass would find it, so its absence proves the table was skipped entirely.
func TestDiffLinesLargeInputFallsBackToLinear(t *testing.T) {
	const n = 2001 // n*n > maxDiffCells (4,000,000)
	mid := n / 2
	a := make([]string, n)
	b := make([]string, n)
	for i := range a {
		if i == mid {
			a[i], b[i] = "SHARED", "SHARED"
			continue
		}
		a[i] = fmt.Sprintf("a-line-%d", i)
		b[i] = fmt.Sprintf("b-line-%d", i)
	}

	done := make(chan []diffOp, 1)
	go func() { done <- diffLines(a, b) }()
	select {
	case got := <-done:
		if len(got) != 2*n {
			t.Fatalf("diffLines returned %d ops, want %d (linear fallback: every line removed+added)", len(got), 2*n)
		}
		for _, op := range got {
			if op.kind == ' ' {
				t.Fatalf("found a context match (%+v) in a large-input diff — the O(n*m) LCS table"+
					" should have been skipped entirely, not just computed slowly", op)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("diffLines did not return within 10s — likely still building the full O(n*m) table")
	}
}

func TestJoinWithLeadingSpace(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"empty", nil, ""},
		{"single", []string{"a"}, " a"},
		{"multiple", []string{"a", "b", "c"}, " a, b, c"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := joinWithLeadingSpace(c.keys); got != c.want {
				t.Errorf("joinWithLeadingSpace(%v) = %q, want %q", c.keys, got, c.want)
			}
		})
	}
}
