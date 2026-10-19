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

package compile

import (
	stdcontext "context"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
)

// Stage identifies a module's reached compiler stage.
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

// ExternalModuleProjection exposes a dependency's package node and exports
// for read-only feature queries.
type ExternalModuleProjection struct {
	PackageNode *ast.BLangPackage
	Symbols     model.ExportedSymbolSpace
}

// SealedModule exposes committed module state for read-only queries, retaining
// its context, parser input, document identities, and path metadata.
type SealedModule struct {
	pkgNode             *ast.BLangPackage
	externalProjections map[semantics.PackageIdentifier]ExternalModuleProjection
	ctx                 *context.CompilerContext
	moduleID            projects.ModuleID
	stage               Stage
	sources             map[projects.DocumentID]string
	sourceIDs           map[string]projects.DocumentID
	paths               workspace.SourcePaths
}

// PackageNode returns the package node, non-nil for a successful SealedModuleFor read.
func (m SealedModule) PackageNode() *ast.BLangPackage { return m.pkgNode }

// Context returns the module driver's compiler context.
func (m SealedModule) Context() *context.CompilerContext { return m.ctx }

// ModuleID returns the sealed module's identity.
func (m SealedModule) ModuleID() projects.ModuleID { return m.moduleID }

// Stage returns the stage this module's sealed generation reached.
func (m SealedModule) Stage() Stage { return m.stage }

// SourceText returns the exact parser input pinned to this module's AST.
func (m SealedModule) SourceText(uri workspace.DocumentURI) (string, bool) {
	name, ok := m.NameOf(uri)
	if !ok {
		return "", false
	}
	id, ok := m.sourceIDs[name]
	if !ok {
		return "", false
	}
	text, ok := m.sources[id]
	return text, ok
}

// NameOf uses the root pinned to this sealed generation, never live discovery.
func (m SealedModule) NameOf(uri workspace.DocumentURI) (string, bool) { return m.paths.NameOf(uri) }

func (m SealedModule) ExternalModuleProjection(org, name string) (ExternalModuleProjection, bool) {
	projection, ok := m.externalProjections[semantics.PackageIdentifier{OrgName: org, ModuleName: name}]
	return projection, ok
}

// SealedModuleFor waits for a stable current generation, then reads committed
// state for u's module. The wait follows superseding generations and respects
// ctx cancellation and armed debounce timers. Missing projects, drivers, or
// package nodes return false; recovered modules with diagnostics may be returned.
func (s *CompilationService) SealedModuleFor(ctx stdcontext.Context, u workspace.DocumentURI) (SealedModule, bool) {
	project, err := s.projects.Project(u)
	if err != nil || project == nil {
		return SealedModule{}, false
	}
	root, err := s.projects.ProjectKey(u)
	if err != nil || !s.isKnown(root) {
		return SealedModule{}, false
	}
	snap, ok := s.awaitStableGeneration(ctx, root)
	if !ok {
		return SealedModule{}, false
	}
	paths := s.projects.Paths(snap.project)
	name, ok := paths.NameOf(u)
	if !ok {
		return SealedModule{}, false
	}
	docID, ok := snap.project.DocumentID(name)
	if !ok {
		return SealedModule{}, false
	}
	// Module state is captured separately from the awaited stable snapshot.
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
		pkgNode:             d.pkgNode,
		externalProjections: modSnap.externalProjections,
		ctx:                 d.ctx,
		moduleID:            docID.ModuleID(),
		stage:               d.stage,
		sources:             d.sources,
		sourceIDs:           d.sourceIDs,
		paths:               paths,
	}, true
}

// awaitStableGeneration waits for a current stable snapshot, scheduling a cycle
// when none is queued or running. Cancellation, an unknown root, or an explicit
// failed-generation marker returns false.
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

				s.ensureCycleQueued(root, current)
			}
			if !waitChannel(ctx, ip.Done()) {
				return StableSnapshot{}, false
			}
			continue
		}
		if s.cycleQueuedOrRunning(root) {

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

// ensureCycleQueued enqueues immediately unless closed or a cycle is already
// running or pending. It does not inspect debounce timers.
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

// cycleQueuedOrRunning checks running, pending, and debounce-armed cycles.
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

// waitStoreChange waits for a store notification, retry interval, or cancellation.
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

// markGenerationFailed sets the marker checked by awaitStableGeneration.
// Only the matching generation is rejected; lifecycle events clear the marker.
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
