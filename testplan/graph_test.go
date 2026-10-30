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

func mustOrder(t *testing.T, plan *testplan.TestPlan) *testplan.Graph {
	t.Helper()
	g, err := testplan.BuildGraph(plan)
	if err != nil {
		t.Fatalf("BuildGraph: unexpected error: %v", err)
	}
	return g
}

func TestBuildGraph_IndependentTestsSortedByName(t *testing.T) {
	plan := &testplan.TestPlan{
		Tests: []testplan.TestFunction{
			{Name: "zTest", Enable: true},
			{Name: "aTest", Enable: true},
			{Name: "mTest", Enable: true},
		},
	}
	g := mustOrder(t, plan)
	if !reflect.DeepEqual(g.Order, []string{"aTest", "mTest", "zTest"}) {
		t.Errorf("expected name-sorted order, got %v", g.Order)
	}
}

func TestBuildGraph_DependentFollowsDependency(t *testing.T) {
	plan := &testplan.TestPlan{
		Tests: []testplan.TestFunction{
			{Name: "testMain", Enable: true, DependsOn: []string{"testSetup"}},
			{Name: "testSetup", Enable: true},
		},
	}
	g := mustOrder(t, plan)

	setupIdx := indexOf(g.Order, "testSetup")
	mainIdx := indexOf(g.Order, "testMain")
	if setupIdx == -1 || mainIdx == -1 {
		t.Fatalf("expected both tests in order, got %v", g.Order)
	}
	if setupIdx > mainIdx {
		t.Errorf("expected testSetup before testMain, got order %v", g.Order)
	}
	if !reflect.DeepEqual(g.Dependents["testSetup"], []string{"testMain"}) {
		t.Errorf("expected testSetup's dependents=[testMain], got %v", g.Dependents["testSetup"])
	}
}

func TestBuildGraph_ChainedDependencies(t *testing.T) {
	// c depends on b depends on a — must come out in order a, b, c.
	plan := &testplan.TestPlan{
		Tests: []testplan.TestFunction{
			{Name: "c", Enable: true, DependsOn: []string{"b"}},
			{Name: "b", Enable: true, DependsOn: []string{"a"}},
			{Name: "a", Enable: true},
		},
	}
	g := mustOrder(t, plan)
	if !reflect.DeepEqual(g.Order, []string{"a", "b", "c"}) {
		t.Errorf("expected [a b c], got %v", g.Order)
	}
}

// TestBuildGraph_DisabledDependency mirrors the corpus fixture
// testerina-disabled-dependency-v.bal exactly: testB depends on testA, but
// testA is explicitly disabled — expect an error naming both tests.
func TestBuildGraph_DisabledDependency(t *testing.T) {
	plan := &testplan.TestPlan{
		Tests: []testplan.TestFunction{
			{Name: "testA", Enable: false},
			{Name: "testB", Enable: true, DependsOn: []string{"testA"}},
		},
	}
	_, err := testplan.BuildGraph(plan)
	if err == nil {
		t.Fatal("expected an error for a disabled dependency")
	}
	if !strings.Contains(err.Error(), "testB") || !strings.Contains(err.Error(), "testA") {
		t.Errorf("expected the error to name both tests, got: %v", err)
	}
}

func TestBuildGraph_CyclicDependency(t *testing.T) {
	plan := &testplan.TestPlan{
		Tests: []testplan.TestFunction{
			{Name: "a", Enable: true, DependsOn: []string{"b"}},
			{Name: "b", Enable: true, DependsOn: []string{"a"}},
		},
	}
	_, err := testplan.BuildGraph(plan)
	if err == nil {
		t.Fatal("expected a cyclic dependency error")
	}
	if !strings.Contains(err.Error(), "cyclic") {
		t.Errorf("expected a cyclic-dependency error, got: %v", err)
	}
}

// TestBuildGraph_UnresolvedDependency covers a dependsOn target that isn't a
// discovered test at all — defensive handling for a case the discovery
// plugin's own validation should already have caught at compile time.
func TestBuildGraph_UnresolvedDependency(t *testing.T) {
	plan := &testplan.TestPlan{
		Tests: []testplan.TestFunction{
			{Name: "testMain", Enable: true, DependsOn: []string{"doesNotExist"}},
		},
	}
	_, err := testplan.BuildGraph(plan)
	if err == nil {
		t.Fatal("expected an error for an unresolved dependsOn target")
	}
}

func TestBuildGraph_MultipleDependentsOfOneTest(t *testing.T) {
	plan := &testplan.TestPlan{
		Tests: []testplan.TestFunction{
			{Name: "shared", Enable: true},
			{Name: "first", Enable: true, DependsOn: []string{"shared"}},
			{Name: "second", Enable: true, DependsOn: []string{"shared"}},
		},
	}
	g := mustOrder(t, plan)
	deps := g.Dependents["shared"]
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependents of shared, got %v", deps)
	}
}

func indexOf(s []string, target string) int {
	for i, v := range s {
		if v == target {
			return i
		}
	}
	return -1
}
