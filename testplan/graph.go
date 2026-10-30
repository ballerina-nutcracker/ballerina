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

package testplan

import (
	"fmt"
	"sort"
	"strings"
)

// Graph is the validated dependency structure built from a TestPlan's tests —
// a Go port of execute.bal's orderTests/restructureTest (register.bal's
// ExecutionManager bookkeeping), confirmed to match its error messages
// exactly against the corpus (testerina-disabled-dependency-v and a
// hand-verified cyclic-dependency case).
type Graph struct {
	// Order lists every test name in a valid topological order: tests with
	// no dependencies come first (sorted by name, matching register.bal's
	// sortTestFunctionsByName), followed by dependent tests in an order
	// where each one follows everything it depends on.
	Order []string

	// Dependents maps a test name to the names of tests that depend on it —
	// the reverse of TestFunction.DependsOn, needed later to cascade a skip
	// to dependents when a dependency fails or doesn't run.
	Dependents map[string][]string
}

// BuildGraph validates plan's dependsOn references and returns the resulting
// Graph. Returns an error for:
//   - a dependsOn target that isn't itself a discovered test (should already
//     be caught at compile time by the discovery plugin's own validation;
//     handled here too since this function has no other way to enforce it)
//   - a dependsOn target with Enable=false
//   - a cyclic dependency chain
func BuildGraph(plan *TestPlan) (*Graph, error) {
	byName := make(map[string]TestFunction, len(plan.Tests))
	for _, t := range plan.Tests {
		byName[t.Name] = t
	}

	g := &Graph{Dependents: make(map[string][]string)}

	var independent, dependent []TestFunction
	for _, t := range plan.Tests {
		if len(t.DependsOn) == 0 {
			independent = append(independent, t)
		} else {
			dependent = append(dependent, t)
		}
	}
	sort.Slice(independent, func(i, j int) bool { return independent[i].Name < independent[j].Name })

	b := &graphBuilder{byName: byName, graph: g, visited: make(map[string]bool)}
	for _, t := range independent {
		// Already has a fixed place in Order — mark visited so a dependent
		// test that also depends on it doesn't append it a second time.
		g.Order = append(g.Order, t.Name)
		b.visited[t.Name] = true
	}
	for _, t := range dependent {
		if b.visited[t.Name] {
			continue
		}
		if err := b.visit(t, nil); err != nil {
			return nil, err
		}
	}
	return g, nil
}

type graphBuilder struct {
	byName  map[string]TestFunction
	graph   *Graph
	visited map[string]bool
}

// visit mirrors restructureTest's recursion exactly: path is the chain of
// test names from the DFS root to testFunction (exclusive), threaded as a
// fresh extended slice at each call rather than a shared push/pop stack —
// the same approach execute.bal's own comment documents using for the same
// reason (no array pop in this interpreter yet).
func (b *graphBuilder) visit(testFunction TestFunction, path []string) error {
	currentPath := append(append([]string{}, path...), testFunction.Name)

	for _, depName := range testFunction.DependsOn {
		dep, ok := b.byName[depName]
		if !ok {
			return fmt.Errorf("the dependent test function is either disabled or not included")
		}

		if !dep.Enable {
			return fmt.Errorf("test %q depends on function %q, but it is either disabled or not included",
				testFunction.Name, dep.Name)
		}
		b.graph.Dependents[dep.Name] = append(b.graph.Dependents[dep.Name], testFunction.Name)

		if cycleStart := indexOf(currentPath, dep.Name); cycleStart != -1 {
			cycle := append(append([]string{}, currentPath[cycleStart:]...), dep.Name)
			return fmt.Errorf("cyclic test dependencies detected: %s", strings.Join(cycle, " -> "))
		}
		if !b.visited[dep.Name] {
			if err := b.visit(dep, currentPath); err != nil {
				return err
			}
		}
	}

	b.visited[testFunction.Name] = true
	b.graph.Order = append(b.graph.Order, testFunction.Name)
	return nil
}

func indexOf(s []string, target string) int {
	for i, v := range s {
		if v == target {
			return i
		}
	}
	return -1
}
