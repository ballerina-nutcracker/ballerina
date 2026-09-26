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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/values"
)

// Full end-to-end coverage of the ballerina/test module (including everything
// that needs a real runtime *extern.Context, e.g. readModuleRerunEntry's
// record construction) lives in .bal-level verification, not here — see
// CHECKPOINT.md's P8.6-P8.8 entry. These tests cover the pure-Go helpers that
// don't need a runtime context at all.

func TestMatchWildcard(t *testing.T) {
	cases := []struct {
		name, pattern string
		want          bool
	}{
		{"testFoo", "testFoo", true},
		{"testFoo", "test*", true},
		{"testFooBar", "test*Bar", true},
		{"testFoo", "other*", false},
		{"testFoo(int)", "testFoo*", true}, // regex metachars in the name must not be interpreted
		{"testXFoo", "test.Foo", false},    // "." in the pattern is literal, not "any character"
		{"test.Foo", "test.Foo", true},     // literal "." in the pattern still matches itself
		{"testFoo", `test\`, false},        // a trailing "\" in the pattern must not break compilation
	}
	for _, c := range cases {
		got, err := matchWildcard(nil, []values.BalValue{c.name, c.pattern})
		if err != nil {
			t.Fatalf("matchWildcard(%q, %q): %v", c.name, c.pattern, err)
		}
		if got != c.want {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v", c.name, c.pattern, got, c.want)
		}
	}
}

func TestEscapeSpecialCharacters(t *testing.T) {
	got, err := escapeSpecialCharacters(nil, []values.BalValue{`a,b"c`})
	if err != nil {
		t.Fatalf("escapeSpecialCharacters: %v", err)
	}
	s, _ := got.(string)
	if s == `a,b"c` {
		t.Errorf("expected special characters to be encoded, got unchanged %q", s)
	}
}

func TestIsBalanced(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"(a[b]{c})", true},
		{"(a[b)", false},
		{"no brackets", true},
		{")(", false},
	}
	for _, c := range cases {
		if got := isBalanced(c.s); got != c.want {
			t.Errorf("isBalanced(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

// writeModuleRerunEntry/readModuleRerunEntry's BalValue conversion (record/map
// construction against a real semtypes.Env) is exercised end-to-end via
// .bal-level verification (see CHECKPOINT.md's P8.6-P8.8 entry: setTestOptions
// with --rerun-failed round-tripped through a real rerun_test.json), matching
// this package's existing convention (see time_test.go) of not hand-building
// BalValue graphs in Go tests for behavior the type checker already guarantees
// shape for. These tests cover the plain-Go merge/read logic beneath that.
func TestReadRerunFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rerun_test.json")

	initial := map[string]rerunEntryJSON{
		"mod1": {TestNames: []string{"testA"}, TestModuleNames: map[string]*string{"testA": ptr("mod1")}},
	}
	data, err := json.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	all, err := readRerunFile(path)
	if err != nil {
		t.Fatalf("readRerunFile: %v", err)
	}
	entry, ok := all["mod1"]
	if !ok || len(entry.TestNames) != 1 || entry.TestNames[0] != "testA" {
		t.Fatalf("unexpected round-tripped entry: %+v", all)
	}
	if entry.TestModuleNames["testA"] == nil || *entry.TestModuleNames["testA"] != "mod1" {
		t.Errorf("unexpected testModuleNames: %+v", entry.TestModuleNames)
	}

	// Merging a second module (mirroring writeModuleRerunEntry's read-modify-
	// write) must not clobber the first.
	all["mod2"] = rerunEntryJSON{TestNames: []string{"testB"}}
	data, err = json.Marshal(all)
	if err != nil {
		t.Fatalf("marshal merged: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write merged: %v", err)
	}
	all, err = readRerunFile(path)
	if err != nil {
		t.Fatalf("readRerunFile after merge: %v", err)
	}
	if _, ok := all["mod1"]; !ok {
		t.Errorf("mod1 entry was lost after merging mod2: %+v", all)
	}
	if _, ok := all["mod2"]; !ok {
		t.Errorf("mod2 entry missing: %+v", all)
	}
}

func ptr(s string) *string { return &s }

func TestReadRerunFileMissingReturnsEmpty(t *testing.T) {
	all, err := readRerunFile(filepath.Join(t.TempDir(), "does_not_exist.json"))
	if err != nil {
		t.Fatalf("expected no error for a missing file, got %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected an empty map, got %+v", all)
	}
}
