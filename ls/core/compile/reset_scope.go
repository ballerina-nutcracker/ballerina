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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

// reset_scope.go computes design item 1's reset scope: the set of modules a
// generation must rebuild from scratch, mirroring ls-ref's
// modulesFromTopoOrder (server.go:592-601, the fast topo-tail path) and
// dependentClosure (server.go:603-635, the precise reverse-BFS fallback used
// when topo order can't be trusted). Everything outside the reset scope
// carries forward the prior generation's moduleDriver by reference — see
// packageDriver.adoptCarriedForward in multimodule.go.
package compile

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/ballerina-nutcracker/ballerina/projects"
)

// moduleTextHash returns a deterministic hash of module's own document set
// (name + live content per document, sorted by name), used to seed the dirty
// set a reset scope is computed from: a module whose hash differs from its
// last-committed generation's hash has genuinely changed content.
func moduleTextHash(module *projects.Module, openText textProvider) string {
	type namedText struct {
		name string
		text string
	}
	docIDs := module.DocumentIDs()
	entries := make([]namedText, 0, len(docIDs))
	for _, docID := range docIDs {
		doc := module.Document(docID)
		if doc == nil {
			continue
		}
		entries = append(entries, namedText{name: doc.Name(), text: documentText(module, doc, openText)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e.name))
		h.Write([]byte{0})
		h.Write([]byte(e.text))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// topoTailScope returns every module from the earliest dirty module's
// topological position onward (ls-ref's modulesFromTopoOrder fast path).
// This deliberately over-invalidates: a module later in topo order with no
// real dependency edge to what changed is still included, the same
// simplification ls-ref accepts rather than requiring an exact reverse
// dependency graph up front. Returns an empty (non-nil) scope if dirty is
// empty.
func topoTailScope(modules []*projects.Module, dirty map[projects.ModuleID]bool) map[projects.ModuleID]bool {
	firstDirty := -1
	for i, m := range modules {
		if dirty[m.ModuleID()] {
			firstDirty = i
			break
		}
	}
	if firstDirty < 0 {
		return map[projects.ModuleID]bool{}
	}
	scope := make(map[projects.ModuleID]bool, len(modules)-firstDirty)
	for _, m := range modules[firstDirty:] {
		scope[m.ModuleID()] = true
	}
	return scope
}

// reverseDependents builds the reverse (dependent) edges of pkg's own-module
// dependency graph from projects.DependencyGraph's forward-only
// DirectDependencies: for every module, every direct dependency it names
// gains that module as one of its direct dependents. Used both by
// dependentClosure (the BFS fallback) and by packageDriver's fingerprint
// pruning (design item 4), which needs a fingerprint-unchanged module's
// direct dependents specifically, not a full closure.
func reverseDependents(pkg *projects.Package, depGraph *projects.DependencyGraph[projects.ModuleDescriptor]) map[projects.ModuleID][]projects.ModuleID {
	reverse := make(map[projects.ModuleID][]projects.ModuleID)
	for _, id := range pkg.ModuleIDs() {
		module := pkg.Module(id)
		if module == nil {
			continue
		}
		for _, depDesc := range depGraph.DirectDependencies(module.Descriptor()) {
			depModule := pkg.ModuleByName(depDesc.Name())
			if depModule == nil {
				continue // external dependency: not tracked by this driver
			}
			depID := depModule.ModuleID()
			reverse[depID] = append(reverse[depID], id)
		}
	}
	return reverse
}

// dependentClosure returns dirty's transitive closure over reverse (every
// module reachable by repeatedly following "is a direct dependent of"),
// mirroring ls-ref's dependentModuleClosure — the precise fallback used when
// topological order isn't available/trustworthy for the fast topoTailScope
// path.
func dependentClosure(dirty map[projects.ModuleID]bool, reverse map[projects.ModuleID][]projects.ModuleID) map[projects.ModuleID]bool {
	scope := make(map[projects.ModuleID]bool, len(dirty))
	queue := make([]projects.ModuleID, 0, len(dirty))
	for id := range dirty {
		scope[id] = true
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, dependent := range reverse[id] {
			if scope[dependent] {
				continue
			}
			scope[dependent] = true
			queue = append(queue, dependent)
		}
	}
	return scope
}

// allModuleIDs returns every module's ID as a fully-populated scope (the "no
// carry-forward data yet, reset everything" case).
func allModuleIDs(modules []*projects.Module) map[projects.ModuleID]bool {
	scope := make(map[projects.ModuleID]bool, len(modules))
	for _, m := range modules {
		scope[m.ModuleID()] = true
	}
	return scope
}
