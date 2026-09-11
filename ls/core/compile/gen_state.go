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

// gen_state.go holds design item 1's cross-generation carry-forward state:
// packageGenState, one per source root, persisted on CompilationService
// (genStateFor/setGenState) across background compile cycles. It is new,
// ls/-only shared mutable state — unlike everything else a packageDriver
// touches (CompilerEnvironment's own field-by-field locking, or per-instance
// state scoped to a single run), so it needs its own explicit ownership rule:
// only CompilationService's background runCycle (realCompilePackage) ever
// reads or writes it. Compile's inline extractForURI fallback — which can run
// concurrently with a background cycle over the same package/environment
// (see multimodule.go's Concurrency doc comment) — never touches it: it
// always passes a nil *packageGenState, so a packageDriver it drives runs a
// full, uncommitted, from-scratch compile of just its target module's prefix,
// exactly as it did before this ticket.
package compile

import (
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// moduleGenSnapshot is one module's carried-forward state from a previously
// committed generation: the moduleDriver it finished that generation with
// (read-only from here on — a later generation that carries it forward never
// mutates it further), whether it errored in Phase 1, its published exported
// symbol space (nil if it errored), its last computed public-API fingerprint
// (design item 4), and a hash of its own document set's content (used to seed
// the next generation's dirty set).
type moduleGenSnapshot struct {
	driver      *moduleDriver
	errored     bool
	exported    model.ExportedSymbolSpace
	fingerprint fingerprint
	textHash    string
}

// packageGenState is the persisted carry-forward state for one source root.
// It is replaced wholesale on each committed generation (never mutated
// in-place), which keeps design item 5's "abort and discard cleanly, no
// cleanup needed" claim true: an aborted (superseded) run simply never calls
// setGenState, so the previous, still-valid packageGenState stays in place
// untouched.
type packageGenState struct {
	modules map[projects.ModuleID]moduleGenSnapshot
}

// genStateFor returns root's carry-forward state, creating an empty one (an
// "everything is dirty" state — every module lookup misses, so the next
// reset-scope computation naturally resets everything) on first use.
func (s *CompilationService) genStateFor(root string) *packageGenState {
	s.genStatesMu.Lock()
	defer s.genStatesMu.Unlock()
	st, ok := s.genStates[root]
	if !ok {
		st = &packageGenState{modules: make(map[projects.ModuleID]moduleGenSnapshot)}
		s.genStates[root] = st
	}
	return st
}

// setGenState replaces root's carry-forward state with modules, the result of
// a just-finished, non-aborted generation.
func (s *CompilationService) setGenState(root string, modules map[projects.ModuleID]moduleGenSnapshot) {
	s.genStatesMu.Lock()
	defer s.genStatesMu.Unlock()
	s.genStates[root] = &packageGenState{modules: modules}
}

// evictGenState drops root's carry-forward state (project evicted, kind
// transitioned, or reloaded fresh — any of which invalidate module identity
// carried in the old state's moduleDrivers/ModuleIDs).
func (s *CompilationService) evictGenState(root string) {
	s.genStatesMu.Lock()
	defer s.genStatesMu.Unlock()
	delete(s.genStates, root)
}
