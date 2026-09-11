// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// sealed.go implements the SealedModule view and the CompilationService read
// that obtains it (ADR: 2026-09-09-sealed-generation-completion). SealedModule
// is the reusable, read-only semantic-read seam over the committed moduleDriver
// of a sealed generation: it exposes the module's package node, the fresh
// per-generation CompilerContext the module's diagnostics and symbol queries
// live on, the module's identity, and the stage the module reached. It is never
// a handle to an in-progress driver — a driver only becomes sealable once its
// generation's carry-forward state has been committed (realCompilePackage's
// cyc.commit) and its snapshot published stable.
package compile

import (
	stdcontext "context"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// Stage is the exported name of the LS stage-ladder rung (stage.go's
// moduleStage). SealedModule exposes the rung a sealed module reached so a
// feature can gate itself by the sealed module's reached compiler stage.
type Stage = moduleStage

const (
	StageUnstarted            Stage = stageUnstarted
	StageParsed               Stage = stageParsed
	StageSymbolResolved       Stage = stageSymbolResolved
	StageTopLevelTypeResolved Stage = stageTopLevelTypeResolved
	StageLocalTypeResolved    Stage = stageLocalTypeResolved
	StageSemanticAnalyzed     Stage = stageSemanticAnalyzed
	StageCFGBuilt             Stage = stageCFGBuilt
	StageCFGAnalyzed          Stage = stageCFGAnalyzed
)

// SealedModule is an immutable view of one module's committed moduleDriver
// from the sealed generation current at read time. The driver's fields are
// read-only from here on (gen_state.go's commit rule): a later generation
// never mutates a committed driver, and an eviction (ProjectRegistered/
// ProjectEvicted/ProjectKindTransitioned) only drops the genStates map entry —
// never a captured driver — so a caller holding this view stays safe even if
// the root reloads mid-read. Symbol identity (SymbolRef, FunctionSignatureRef)
// remains valid because the view also pins the same CompilerEnvironment the
// driver compiled against.
type SealedModule struct {
	pkgNode  *ast.BLangPackage
	ctx      *context.CompilerContext
	moduleID projects.ModuleID
	stage    Stage
}

// PackageNode returns the sealed module's package node (nil only for a module
// that never reached StageSymbolResolved, in which case SealedModuleFor
// reports ok=false).
func (m SealedModule) PackageNode() *ast.BLangPackage { return m.pkgNode }

// Context returns the fresh per-generation CompilerContext this module's
// diagnostics and symbol queries live on.
func (m SealedModule) Context() *context.CompilerContext { return m.ctx }

// ModuleID returns the sealed module's identity.
func (m SealedModule) ModuleID() projects.ModuleID { return m.moduleID }

// Stage returns the stage this module's sealed generation reached.
func (m SealedModule) Stage() Stage { return m.stage }

// SealedModuleFor returns the sealed module containing document u, waiting
// (context-aware) for the generation current at read time to complete the
// normal background CFGAnalyzed cycle. The read never substitutes a stale
// generation: if a newer generation supersedes the target mid-wait, the read
// re-targets to the newer one; if the root is evicted or the request's context
// is cancelled, it reports ok=false. If the root has no queued or in-flight
// cycle, the read schedules the current generation itself; a debounce-armed
// timer counts as a queued cycle and is awaited, never bypassed. If the
// current generation cannot produce a usable semantic module (syntax/Phase-1
// failure, missing driver, or no package node), the read reports ok=false —
// callers surface an empty result, never stale semantics.
func (s *CompilationService) SealedModuleFor(ctx stdcontext.Context, u uri.DocumentURI) (SealedModule, bool) {
	project, err := s.projects.Project(u)
	if err != nil || project == nil {
		return SealedModule{}, false
	}
	root := project.SourceRoot()
	if !s.isKnown(root) {
		return SealedModule{}, false
	}
	snap, ok := s.awaitStableGeneration(ctx, root)
	if !ok {
		return SealedModule{}, false
	}
	docID, ok := snap.project.DocumentID(u.Path())
	if !ok {
		return SealedModule{}, false
	}
	// ADR §2: capture the current packageGenState under genStatesMu, then read
	// its driver after releasing the lock. A stable snapshot for generation N
	// can only coexist with a genState committed by N or a later committed
	// generation (realCompilePackage commits before runCycle publishes), so
	// the driver read here is never older than the awaited snapshot.
	s.genStatesMu.Lock()
	state := s.genStates[root]
	s.genStatesMu.Unlock()
	if state == nil {
		return SealedModule{}, false
	}
	modSnap, ok := state.modules[docID.ModuleID()]
	if !ok || modSnap.driver == nil {
		return SealedModule{}, false
	}
	d := modSnap.driver
	if d.pkgNode == nil || d.ctx == nil {
		return SealedModule{}, false
	}
	return SealedModule{
		pkgNode:  d.pkgNode,
		ctx:      d.ctx,
		moduleID: docID.ModuleID(),
		stage:    d.stage,
	}, true
}

// awaitStableGeneration blocks until the root's current generation has a
// stable snapshot, scheduling the cycle itself when the root has no queued,
// debounce-armed, or in-flight cycle (ADR §1). It returns false when ctx is
// cancelled, the root is evicted, or the current generation's compile failed
// (panicked) — the last via the failed-generation marker, which prevents an
// endless reschedule loop over a deterministically re-panicking cycle.
func (s *CompilationService) awaitStableGeneration(ctx stdcontext.Context, root string) (StableSnapshot, bool) {
	const retryInterval = 5 * time.Millisecond
	for {
		if ctx.Err() != nil {
			return StableSnapshot{}, false
		}
		current, ok := s.reader.Generation(root)
		if !ok {
			return StableSnapshot{}, false
		}
		if snap, ok := s.store.Stable(root); ok && snap.key.Generation >= current {
			return snap, true
		}
		if s.generationFailed(root, current) {
			return StableSnapshot{}, false
		}
		if ip, ok := s.store.InProgress(root); ok {
			if ip.key.Generation < current {
				// An older cycle is finishing; make sure the current
				// generation is queued behind it (depth-1 latest-wins parks it
				// in pending).
				s.ensureCycleQueued(root, current)
			}
			if !waitChannel(ctx, ip.Done()) {
				return StableSnapshot{}, false
			}
			continue
		}
		if s.cycleQueuedOrRunning(root) {
			// In flight (the submit→setInProgress window), parked in pending,
			// or debounce-armed: wait for the store to move.
			if !s.waitStoreChange(ctx, retryInterval) {
				return StableSnapshot{}, false
			}
			continue
		}
		s.ensureCycleQueued(root, current)
		if !s.waitStoreChange(ctx, retryInterval) {
			return StableSnapshot{}, false
		}
	}
}

// ensureCycleQueued schedules a cycle for (root, gen) if the root has none
// queued or in flight. enqueueImmediately is idempotent against the in-flight
// and depth-1 pending slots, so racing the CRITICAL-tier ProjectUpdated
// subscriber is safe: whichever path schedules first wins and the other
// either no-ops into the same pending slot or observes the store move.
func (s *CompilationService) ensureCycleQueued(root string, gen uint64) {
	if s.isClosed() {
		return
	}
	s.cycleMu.Lock()
	inFlight := s.inFlight[root]
	pending := s.pending[root] != nil
	s.cycleMu.Unlock()
	if inFlight || pending {
		return
	}
	s.enqueueImmediately(root, gen)
}

// cycleQueuedOrRunning reports whether root has a cycle in flight, parked in
// the depth-1 pending slot, or armed on a debounce timer. A debounce-armed
// cycle counts as queued: the sealed-generation read awaits it rather than
// bypassing the debounce with its own immediate cycle.
func (s *CompilationService) cycleQueuedOrRunning(root string) bool {
	s.cycleMu.Lock()
	inFlight := s.inFlight[root]
	pending := s.pending[root] != nil
	s.cycleMu.Unlock()
	if inFlight || pending {
		return true
	}
	s.debounceMu.Lock()
	armed := s.debounceTimers[root] != nil
	s.debounceMu.Unlock()
	return armed
}

// waitStoreChange waits for the next store mutation (or the retry interval, a
// safety net against a lost scheduling race, e.g. enqueueImmediately dropping
// a cycle because the project was concurrently reloaded).
func (s *CompilationService) waitStoreChange(ctx stdcontext.Context, retryInterval time.Duration) bool {
	select {
	case <-s.store.changeCh():
		return true
	case <-time.After(retryInterval):
		return true
	case <-ctx.Done():
		return false
	}
}

func waitChannel(ctx stdcontext.Context, ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

// markGenerationFailed records that root's cycle for gen panicked, so
// awaitStableGeneration stops rescheduling a deterministically failing cycle
// instead of looping forever. Superseded generations never block the marker:
// only an exact gen match short-circuits the wait, and lifecycle events clear
// the marker (see handleLifecycle), so a reloaded root's restarted generation
// counter is never shadowed by a stale marker.
func (s *CompilationService) markGenerationFailed(root string, gen uint64) {
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()
	if s.failedGens == nil {
		s.failedGens = make(map[string]uint64)
	}
	s.failedGens[root] = gen
}

func (s *CompilationService) generationFailed(root string, gen uint64) bool {
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()
	return s.failedGens[root] == gen && gen != 0
}

func (s *CompilationService) clearFailedGen(root string) {
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()
	delete(s.failedGens, root)
}
