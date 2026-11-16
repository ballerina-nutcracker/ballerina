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

import "slices"

// UpdateGraph returns the Graph for newPlan, reusing prevGraph (built from
// prevPlan) for tests whose DependsOn/Enable chain is unchanged. The order is
// valid for newPlan but can differ from BuildGraph(newPlan) when unchanged
// tests were reordered. A nil prevPlan or prevGraph falls back to BuildGraph.
func UpdateGraph(prevPlan *TestPlan, prevGraph *Graph, newPlan *TestPlan) (*Graph, error) {
	if prevPlan == nil || prevGraph == nil {
		return BuildGraph(newPlan)
	}

	prevByName := make(map[string]TestFunction, len(prevPlan.Tests))
	for _, t := range prevPlan.Tests {
		prevByName[t.Name] = t
	}
	newByName := make(map[string]TestFunction, len(newPlan.Tests))
	for _, t := range newPlan.Tests {
		newByName[t.Name] = t
	}

	removed := map[string]bool{}
	for name := range prevByName {
		if _, ok := newByName[name]; !ok {
			removed[name] = true
		}
	}

	changed := map[string]bool{}
	for _, t := range newPlan.Tests {
		old, existed := prevByName[t.Name]
		if !existed || old.Enable != t.Enable || !slices.Equal(old.DependsOn, t.DependsOn) {
			changed[t.Name] = true
		}
	}

	// A test whose own DependsOn/Enable are unchanged can still need
	// re-validation: if something it (transitively) depends on changed or
	// was removed, its own dependency chain may now error or cycle.
	// prevGraph.Dependents already holds the reverse edges needed to find
	// everyone downstream of a changed or removed test.
	affected := map[string]bool{}
	queue := make([]string, 0, len(changed)+len(removed))
	for name := range changed {
		affected[name] = true
		queue = append(queue, name)
	}
	queue = append(queue, keys(removed)...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		for _, dependent := range prevGraph.Dependents[name] {
			if !affected[dependent] {
				affected[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}

	// Everything else from the previous graph is unaffected: carry its
	// relative order forward unchanged, reclassifying by the new plan's
	// DependsOn (identical to the old one, by definition of unaffected).
	var keptIndependent, keptDependent []string
	for _, name := range prevGraph.Order {
		if removed[name] || affected[name] {
			continue
		}
		if len(newByName[name].DependsOn) == 0 {
			keptIndependent = append(keptIndependent, name)
		} else {
			keptDependent = append(keptDependent, name)
		}
	}

	// Affected tests that turn out independent join the sorted-by-name
	// prefix directly — BuildGraph places every independent test there,
	// regardless of whether it's new or pre-existing.
	var newIndependentAffected []TestFunction
	var newDependentAffected []TestFunction
	for _, t := range newPlan.Tests {
		if !affected[t.Name] {
			continue
		}
		if len(t.DependsOn) == 0 {
			newIndependentAffected = append(newIndependentAffected, t)
		} else {
			newDependentAffected = append(newDependentAffected, t)
		}
	}
	slices.SortFunc(newIndependentAffected, func(a, b TestFunction) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		default:
			return 0
		}
	})

	g := &Graph{Dependents: make(map[string][]string, len(prevGraph.Dependents))}
	// Carry over edges whose dependency side is unaffected, dropping any
	// dependent that will be re-added fresh by the affected-set visit below
	// (avoids double-recording the same edge). Reuse an untouched dependents
	// slice as is; copy only when an edge is dropped.
	for depName, dependents := range prevGraph.Dependents {
		if removed[depName] || affected[depName] {
			continue
		}
		needsFilter := false
		for _, dn := range dependents {
			if removed[dn] || affected[dn] {
				needsFilter = true
				break
			}
		}
		if !needsFilter {
			g.Dependents[depName] = dependents
			continue
		}
		var kept []string
		for _, dn := range dependents {
			if !removed[dn] && !affected[dn] {
				kept = append(kept, dn)
			}
		}
		if len(kept) > 0 {
			g.Dependents[depName] = kept
		}
	}

	b := &graphBuilder{byName: newByName, graph: g, visited: make(map[string]bool, len(newByName))}
	for _, name := range mergeByName(keptIndependent, newIndependentAffected) {
		g.Order = append(g.Order, name)
		b.visited[name] = true
	}
	for _, name := range keptDependent {
		g.Order = append(g.Order, name)
		b.visited[name] = true
	}

	for _, t := range newDependentAffected {
		if b.visited[t.Name] {
			continue
		}
		if err := b.visit(t, nil); err != nil {
			return nil, err
		}
	}

	return g, nil
}

func keys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// mergeByName merges kept (already sorted by name) with added (sorted by
// name) into one combined, sorted-by-name slice.
func mergeByName(kept []string, added []TestFunction) []string {
	result := make([]string, 0, len(kept)+len(added))
	i, j := 0, 0
	for i < len(kept) && j < len(added) {
		if kept[i] <= added[j].Name {
			result = append(result, kept[i])
			i++
		} else {
			result = append(result, added[j].Name)
			j++
		}
	}
	result = append(result, kept[i:]...)
	for ; j < len(added); j++ {
		result = append(result, added[j].Name)
	}
	return result
}
