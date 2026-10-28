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

package testplan_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/testplan"
)

func TestUpdateGraph_NilPrevDelegatesToBuildGraph(t *testing.T) {
	plan := &testplan.TestPlan{Tests: []testplan.TestFunction{{Name: "a", Enable: true}}}

	got, err := testplan.UpdateGraph(nil, nil, plan)
	if err != nil {
		t.Fatalf("UpdateGraph: unexpected error: %v", err)
	}
	want := mustOrder(t, plan)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected UpdateGraph(nil, nil, plan) to equal BuildGraph(plan); got %+v, want %+v", got, want)
	}
}

// TestUpdateGraph_MatchesFreshRebuild runs a series of realistic edits
// (add/remove/change a test, without reordering unrelated declarations) and
// checks UpdateGraph's result against a from-scratch BuildGraph call on the
// same resulting plan. This is the equivalence UpdateGraph's doc comment
// promises for ordinary edits.
func TestUpdateGraph_MatchesFreshRebuild(t *testing.T) {
	tests := []struct {
		name     string
		prevPlan *testplan.TestPlan
		newPlan  *testplan.TestPlan
	}{
		{
			name: "add an independent test",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "aTest", Enable: true},
				{Name: "zTest", Enable: true},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "aTest", Enable: true},
				{Name: "mTest", Enable: true},
				{Name: "zTest", Enable: true},
			}},
		},
		{
			name: "add a dependent test on an existing test",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "setup", Enable: true},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "setup", Enable: true},
				{Name: "main", Enable: true, DependsOn: []string{"setup"}},
			}},
		},
		{
			name: "remove an untouched independent test",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
			}},
		},
		{
			name: "change a dependent test's dependsOn to a different existing test",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true},
				{Name: "c", Enable: true, DependsOn: []string{"a"}},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true},
				{Name: "c", Enable: true, DependsOn: []string{"b"}},
			}},
		},
		{
			name: "disable a test that another depends on",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: false},
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
			}},
		},
		{
			name: "chain: add a new link that depends on the old tail",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
				{Name: "c", Enable: true, DependsOn: []string{"b"}},
			}},
		},
		{
			name: "introduce a cycle via an edit to an existing test",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true, DependsOn: []string{"b"}},
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
			}},
		},
		{
			name: "remove a test that another still depends on",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "a", Enable: true},
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "b", Enable: true, DependsOn: []string{"a"}},
			}},
		},
		{
			name: "diamond dependency, add a new top-level consumer",
			prevPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "base", Enable: true},
				{Name: "left", Enable: true, DependsOn: []string{"base"}},
				{Name: "right", Enable: true, DependsOn: []string{"base"}},
			}},
			newPlan: &testplan.TestPlan{Tests: []testplan.TestFunction{
				{Name: "base", Enable: true},
				{Name: "left", Enable: true, DependsOn: []string{"base"}},
				{Name: "right", Enable: true, DependsOn: []string{"base"}},
				{Name: "top", Enable: true, DependsOn: []string{"left", "right"}},
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prevGraph, err := testplan.BuildGraph(tc.prevPlan)
			if err != nil {
				t.Fatalf("BuildGraph(prevPlan): unexpected error: %v", err)
			}

			got, gotErr := testplan.UpdateGraph(tc.prevPlan, prevGraph, tc.newPlan)
			want, wantErr := testplan.BuildGraph(tc.newPlan)

			if (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("error mismatch: UpdateGraph err=%v, BuildGraph err=%v", gotErr, wantErr)
			}
			if gotErr != nil {
				if gotErr.Error() != wantErr.Error() {
					t.Errorf("error text mismatch:\nUpdateGraph: %v\nBuildGraph:  %v", gotErr, wantErr)
				}
				return
			}
			if !reflect.DeepEqual(got.Order, want.Order) {
				t.Errorf("Order mismatch:\nUpdateGraph: %v\nBuildGraph:  %v", got.Order, want.Order)
			}
			if !reflect.DeepEqual(got.Dependents, want.Dependents) {
				t.Errorf("Dependents mismatch:\nUpdateGraph: %v\nBuildGraph:  %v", got.Dependents, want.Dependents)
			}
		})
	}
}

// TestUpdateGraph_ReusesUnaffectedDependentsSlice proves the unaffected
// portion of the graph is carried over, not recomputed: an untouched test's
// Dependents entry must be the exact same slice BuildGraph(prevPlan)
// produced, not a freshly rebuilt one, after an edit that doesn't touch it.
func TestUpdateGraph_ReusesUnaffectedDependentsSlice(t *testing.T) {
	prevPlan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "shared", Enable: true},
		{Name: "first", Enable: true, DependsOn: []string{"shared"}},
		{Name: "second", Enable: true, DependsOn: []string{"shared"}},
		{Name: "untouched", Enable: true},
	}}
	prevGraph, err := testplan.BuildGraph(prevPlan)
	if err != nil {
		t.Fatalf("BuildGraph: unexpected error: %v", err)
	}

	newPlan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "shared", Enable: true},
		{Name: "first", Enable: true, DependsOn: []string{"shared"}},
		{Name: "second", Enable: true, DependsOn: []string{"shared"}},
		{Name: "untouched", Enable: true},
		{Name: "new", Enable: true},
	}}
	got, err := testplan.UpdateGraph(prevPlan, prevGraph, newPlan)
	if err != nil {
		t.Fatalf("UpdateGraph: unexpected error: %v", err)
	}

	oldDeps := prevGraph.Dependents["shared"]
	newDeps := got.Dependents["shared"]
	if len(oldDeps) == 0 || len(newDeps) == 0 {
		t.Fatalf("expected non-empty dependents for 'shared' in both graphs, got old=%v new=%v", oldDeps, newDeps)
	}
	if &oldDeps[0] != &newDeps[0] {
		t.Error("expected 'shared''s dependents slice to be reused verbatim for an edit that doesn't touch it or its dependents")
	}
}

// TestUpdateGraph_ReorderedUnaffectedTestsKeepPreviousRelativeOrder documents
// the one known gap from byte-identical BuildGraph output: if declarations
// for tests that otherwise didn't change are reordered relative to each
// other in the new plan (as opposed to a test merely being added or
// removed), UpdateGraph keeps their previous relative order rather than
// matching what a fresh rebuild would produce from the new plan order.
func TestUpdateGraph_ReorderedUnaffectedTestsKeepPreviousRelativeOrder(t *testing.T) {
	prevPlan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "first", Enable: true},
		{Name: "second", Enable: true, DependsOn: []string{"first"}},
		{Name: "third", Enable: true, DependsOn: []string{"second"}},
	}}
	prevGraph, err := testplan.BuildGraph(prevPlan)
	if err != nil {
		t.Fatalf("BuildGraph: unexpected error: %v", err)
	}

	// Same three tests, unchanged, but declared in a different order.
	newPlan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "third", Enable: true, DependsOn: []string{"second"}},
		{Name: "second", Enable: true, DependsOn: []string{"first"}},
		{Name: "first", Enable: true},
	}}

	got, err := testplan.UpdateGraph(prevPlan, prevGraph, newPlan)
	if err != nil {
		t.Fatalf("UpdateGraph: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got.Order, prevGraph.Order) {
		t.Errorf("expected UpdateGraph to keep the previous Order %v for an edit that only reorders unaffected declarations, got %v",
			prevGraph.Order, got.Order)
	}
}

func TestUpdateGraph_RemovedDependencyErrorsLikeFreshBuild(t *testing.T) {
	prevPlan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "a", Enable: true},
		{Name: "b", Enable: true, DependsOn: []string{"a"}},
	}}
	prevGraph, err := testplan.BuildGraph(prevPlan)
	if err != nil {
		t.Fatalf("BuildGraph: unexpected error: %v", err)
	}

	newPlan := &testplan.TestPlan{Tests: []testplan.TestFunction{
		{Name: "b", Enable: true, DependsOn: []string{"a"}},
	}}

	_, err = testplan.UpdateGraph(prevPlan, prevGraph, newPlan)
	if err == nil {
		t.Fatal("expected an error for a dependsOn target removed from the plan")
	}
	if !strings.Contains(err.Error(), "either disabled or not included") {
		t.Errorf("expected the unresolved-dependency error, got: %v", err)
	}
}
