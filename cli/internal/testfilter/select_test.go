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

package testfilter_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/cli/internal/testfilter"
	"github.com/ballerina-nutcracker/ballerina/testplan"
)

func includedNames(t *testing.T, included map[string]bool) []string {
	t.Helper()
	var names []string
	for name, in := range included {
		if in {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func TestSelect_NoFilters_IncludesEverything(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "a"}, {Name: "b"}}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("got %v, want [a b]", got)
	}
}

func TestSelect_PlainNameMatch(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "testA"}, {Name: "testB"}}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{Tests: "testB"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"testB"}) {
		t.Errorf("got %v, want [testB]", got)
	}
}

func TestSelect_Wildcard(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "testFoo"}, {Name: "testBar"}, {Name: "other"}}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{Tests: "test*"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"testBar", "testFoo"}) {
		t.Errorf("got %v, want [testBar testFoo]", got)
	}
}

func TestSelect_ModuleQualified(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "testA"}}}

	// Matches: qualifier equals this module.
	included, err := testfilter.Select(plan, testfilter.SelectOptions{
		Tests: "pkg.submod:testA", ModuleName: "pkg.submod", PackageName: "pkg",
	})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"testA"}) {
		t.Errorf("matching module: got %v, want [testA]", got)
	}

	// Doesn't match: qualifier names a different module.
	included, err = testfilter.Select(plan, testfilter.SelectOptions{
		Tests: "pkg.othermod:testA", ModuleName: "pkg.submod", PackageName: "pkg",
	})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); len(got) != 0 {
		t.Errorf("non-matching module: got %v, want none", got)
	}
}

func TestSelect_Groups(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "unit", Groups: []string{"unit"}},
		{Name: "slow", Groups: []string{"slow"}},
	}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{Groups: "unit"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"unit"}) {
		t.Errorf("got %v, want [unit]", got)
	}
}

func TestSelect_DisableGroups(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "unit", Groups: []string{"unit"}},
		{Name: "slow", Groups: []string{"slow"}},
	}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{DisableGroups: "slow"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"unit"}) {
		t.Errorf("got %v, want [unit]", got)
	}
}

func TestSelect_DependsOnTransitiveClosure(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "testFunc"},
		{Name: "testFunc2", DependsOn: []string{"testFunc"}},
		{Name: "unrelated"},
	}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{Tests: "testFunc2"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"testFunc", "testFunc2"}) {
		t.Errorf("got %v, want [testFunc testFunc2] (dependency auto-included)", got)
	}
}

// TestSelect_DisabledDependencyStillIncluded confirms a dependency pulled in
// only via transitive closure is registered even when it's disabled and
// wouldn't itself match any filter — required so execute.bal's "depends on a
// disabled function" error keeps firing (see the package doc comment).
func TestSelect_DisabledDependencyStillIncluded(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "testDisabled", Enable: false},
		{Name: "testDependent", DependsOn: []string{"testDisabled"}},
	}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{Tests: "testDependent"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"testDependent", "testDisabled"}) {
		t.Errorf("got %v, want [testDependent testDisabled]", got)
	}
}

// TestSelect_ExplicitlyNamedDisabledTestStillIncluded confirms Select never
// inspects Enable itself — filtering is purely name/group/wildcard/module
// based. registerTestConfig's own effectiveEnable computation (unchanged)
// is what makes an explicitly-named disabled test invisible to "No tests
// found" at runtime, not an exclusion here.
func TestSelect_ExplicitlyNamedDisabledTestStillIncluded(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "testDisabled", Enable: false}}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{Tests: "testDisabled"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"testDisabled"}) {
		t.Errorf("got %v, want [testDisabled]", got)
	}
}

func TestSelect_RerunFailed(t *testing.T) {
	dir := t.TempDir()
	rerunPath := filepath.Join(dir, "rerun_test.json")
	data, err := json.Marshal(map[string]any{
		"mymod": map[string]any{
			"testNames":       []string{"testBad"},
			"testModuleNames": map[string]any{},
			"subTestNames":    map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(rerunPath, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "testGood"}, {Name: "testBad"}}}
	included, err := testfilter.Select(plan, testfilter.SelectOptions{
		RerunFailed: true, RerunJSONPath: rerunPath, ModuleName: "mymod",
	})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got := includedNames(t, included); !slices.Equal(got, []string{"testBad"}) {
		t.Errorf("got %v, want [testBad]", got)
	}
}

// TestSelect_RerunFailed_MissingOrInvalidJSON confirms a missing file,
// malformed JSON, or a JSON file with no entry for this module all report
// ErrInvalidRerunData — the same three conditions native/test_io.go's
// readModuleRerunEntry collapses into one error. This package is the only
// place that detects them, since nothing calls through to ballerina/test's
// setTestOptions anymore. The user-facing wording is deliberately NOT
// asserted here: it lives with the CLI that prints it (cli/cmd/test.go's
// invalidRerunDataMessage) and is pinned by the rerun-failed golden
// fixtures, so this package only owns *which* error is reported.
func TestSelect_RerunFailed_MissingOrInvalidJSON(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "testGood"}}}

	t.Run("missing file", func(t *testing.T) {
		_, err := testfilter.Select(plan, testfilter.SelectOptions{
			RerunFailed: true, RerunJSONPath: filepath.Join(t.TempDir(), "does-not-exist.json"), ModuleName: "mymod",
		})
		if !errors.Is(err, testfilter.ErrInvalidRerunData) {
			t.Errorf("Select: got %v, want ErrInvalidRerunData", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "rerun_test.json")
		if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		_, err := testfilter.Select(plan, testfilter.SelectOptions{
			RerunFailed: true, RerunJSONPath: path, ModuleName: "mymod",
		})
		if !errors.Is(err, testfilter.ErrInvalidRerunData) {
			t.Errorf("Select: got %v, want ErrInvalidRerunData", err)
		}
	})

	t.Run("missing module key", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "rerun_test.json")
		if err := os.WriteFile(path, []byte(`{"someOtherModule":{"testNames":["x"]}}`), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		_, err := testfilter.Select(plan, testfilter.SelectOptions{
			RerunFailed: true, RerunJSONPath: path, ModuleName: "mymod",
		})
		if !errors.Is(err, testfilter.ErrInvalidRerunData) {
			t.Errorf("Select: got %v, want ErrInvalidRerunData", err)
		}
	})
}
