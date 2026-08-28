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

package compile

import (
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// CompilationKey is the supersession group. SourceRoot is the ADR-053
// core-internal key; Generation is the per-source-root monotonic counter.
type CompilationKey struct {
	SourceRoot string
	Generation uint64
}

// StableSnapshot is the frozen, fully-materialised result of the last
// successful accepted cycle for a (sourceRoot, generation). Synchronous, never
// cancelled. Its Package and diagnostics live on the persistent per-source-root
// CompilerEnvironment, so its symbols and file indices stay valid for the
// snapshot's life (until the source root is evicted).
type StableSnapshot struct {
	key               CompilationKey
	project           projects.Project
	pkg               *projects.Package
	descriptor        string
	byFile            map[string][]CompilerDiagnostic // fileName -> all diagnostics
	resByFile         map[string][]CompilerDiagnostic // fileName -> resolution subset
	resolutionErrored bool
}

// Key returns the snapshot's supersession key.
func (s StableSnapshot) Key() CompilationKey { return s.key }

// Project returns the published project.
func (s StableSnapshot) Project() projects.Project { return s.project }

// Package returns the modifier-chain package whose compile fired.
func (s StableSnapshot) Package() *projects.Package { return s.pkg }

// Diagnostics returns all diagnostics across files (flattened), for
// observability. The server publishes per-document via DiagnosticsFor.
func (s StableSnapshot) Diagnostics() []CompilerDiagnostic {
	var all []CompilerDiagnostic
	for _, ds := range s.byFile {
		all = append(all, ds...)
	}
	return all
}

// InProgressSnapshot is the current generation's pending/running compilation.
type InProgressSnapshot struct {
	key    CompilationKey
	done   <-chan struct{}
	result func() (StableSnapshot, error)
}

// Key returns the in-progress cycle's supersession key.
func (s InProgressSnapshot) Key() CompilationKey { return s.key }

// Done returns a channel closed when the cycle finishes.
func (s InProgressSnapshot) Done() <-chan struct{} { return s.done }

// SnapshotStore is the bounded dual-snapshot repository (ADR-058). Stable
// snapshots are retained by count (latest per source root, plus a global LRU
// bound); one in-progress slot per source root. The stale-publication gate
// runs in publishStable: a snapshot whose generation is no longer current is
// discarded (CE-E3) and never stored.
type SnapshotStore struct {
	mu         sync.Mutex
	stable     map[string]StableSnapshot
	inProgress map[string]InProgressSnapshot
	order      []string // LRU order of stable roots (front = oldest)
	maxStable  int
	genFn      func(string) (uint64, bool)
}

func newSnapshotStore(maxStable int, genFn func(string) (uint64, bool)) *SnapshotStore {
	if maxStable <= 0 {
		maxStable = 16
	}
	return &SnapshotStore{
		stable:     make(map[string]StableSnapshot),
		inProgress: make(map[string]InProgressSnapshot),
		maxStable:  maxStable,
		genFn:      genFn,
	}
}

// Stable returns the latest stable snapshot for root.
func (s *SnapshotStore) Stable(root string) (StableSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.stable[root]
	return snap, ok
}

// InProgress returns the current in-progress cycle for root.
func (s *SnapshotStore) InProgress(root string) (InProgressSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ip, ok := s.inProgress[root]
	return ip, ok
}

// latestStableGen returns the generation of the latest stable for root, or 0.
func (s *SnapshotStore) latestStableGen(root string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if snap, ok := s.stable[root]; ok {
		return snap.key.Generation
	}
	return 0
}

// currentGeneration reads the live generation for root via the workspace.
func (s *SnapshotStore) currentGeneration(root string) (uint64, bool) {
	if s.genFn == nil {
		return 0, false
	}
	return s.genFn(root)
}

// setInProgress records the in-progress cycle for root (overwriting any
// prior slot, which is by construction already finished).
func (s *SnapshotStore) setInProgress(root string, ip InProgressSnapshot) {
	s.mu.Lock()
	s.inProgress[root] = ip
	s.mu.Unlock()
}

// clearInProgress removes the in-progress slot for root.
func (s *SnapshotStore) clearInProgress(root string) {
	s.mu.Lock()
	delete(s.inProgress, root)
	s.mu.Unlock()
}

// publishStable runs on the compile goroutine: (1) stale gate, (2) store the
// snapshot synchronously, (3) publish CE-E1 (CRITICAL), CE-E5a (BEST_EFFORT),
// and — if resolution was clean — CE-E5b (BEST_EFFORT). A stale snapshot
// (its generation no longer current) is discarded with CE-E3 (COALESCEABLE)
// and not stored. The store-write precedes all Publish calls so any CE-E5a/E5b
// handler reads a populated store.
func (s *SnapshotStore) publishStable(snap StableSnapshot, bus *event.Bus) {
	if bus == nil {
		return
	}
	current, ok := s.currentGeneration(snap.key.SourceRoot)
	if !ok || snap.key.Generation != current {
		bus.Publish(event.NewCompilationCancelledEvent(snap.key.SourceRoot, snap.key.Generation))
		return
	}
	s.storeStable(snap)
	bus.Publish(event.NewCompilationSucceededEvent(snap.key.SourceRoot, snap.descriptor, snap.key.Generation))
	bus.Publish(event.NewResolutionDiagnosticsReadyEvent(snap.key.SourceRoot, snap.descriptor, snap.key.Generation))
	if !snap.resolutionErrored {
		bus.Publish(event.NewCompilationDiagnosticsReadyEvent(snap.key.SourceRoot, snap.descriptor, snap.key.Generation))
	}
}

// storeStable records the snapshot, evicting an LRU stable root when the bound
// is exceeded (never evicting the just-stored root).
func (s *SnapshotStore) storeStable(snap StableSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	root := snap.key.SourceRoot
	if _, exists := s.stable[root]; !exists {
		s.order = append(s.order, root)
		if len(s.order) > s.maxStable {
			victim := s.order[0]
			s.order = s.order[1:]
			delete(s.stable, victim)
		}
	}
	s.stable[root] = snap
}

// evictRoot drops the stable and in-progress state for root.
func (s *SnapshotStore) evictRoot(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.stable, root)
	delete(s.inProgress, root)
	for i, r := range s.order {
		if r == root {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
}

// evictAll drops every stable and in-progress snapshot (shutdown).
func (s *SnapshotStore) evictAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stable = make(map[string]StableSnapshot)
	s.inProgress = make(map[string]InProgressSnapshot)
	s.order = nil
}

// diagsForFile returns the all-diagnostics slice for a fileName within the
// snapshot, or nil. Used by CompilationService.DiagnosticsFor.
func (s StableSnapshot) diagsForFile(fileName string) []CompilerDiagnostic {
	if s.byFile == nil {
		return nil
	}
	return s.byFile[fileName]
}
