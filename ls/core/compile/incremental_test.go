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
	"path/filepath"
	"testing"
)

// TestPackageDriver_UnchangedModule_CarriesForwardSameDriverInstance is
// design item 1 + item 4's combined regression check: after a body-only edit
// to greet (signature unchanged), a second generation seeded with the first
// generation's committed carry-forward state must:
//   - NOT reuse greet's own prior moduleDriver — greet itself is dirty (its
//     text changed) and must be rebuilt fresh.
//   - reuse consumer's prior moduleDriver unchanged, even though consumer is
//     topologically downstream of greet — this is only possible if design
//     item 4's fingerprint pruning fired (consumer would otherwise fall
//     inside topoTailScope's over-invalidated suffix and get rebuilt just for
//     following greet in topo order). This is the one assertion that only
//     passes if pruning specifically targets greet's true dependents, since
//     topoTailScope alone would rebuild it regardless.
//
// standalone (no dependency relationship to greet at all) is deliberately not
// asserted to keep the same driver instance: whether topoTailScope's
// over-invalidated suffix happens to include it depends on where the
// topological sort places an unrelated, dependency-free module — an
// accepted, unspecified detail of the fast path (see reset_scope.go). Its
// diagnostics must still surface either way, which is asserted below.
func TestPackageDriver_UnchangedModule_CarriesForwardSameDriverInstance(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg1, _ := openMultimoduleFixture(t, projSvc)

	gen1 := newPackageDriver(pkg1, projSvc.OpenText, nil, nil)
	gen1.advanceAll(stageCFGAnalyzed)
	if len(gen1.phase1Errored) != 0 {
		t.Fatalf("gen1 phase1Errored = %v, want empty", gen1.phase1Errored)
	}
	committed := gen1.snapshotForCommit()
	priorState := &packageGenState{modules: committed}

	greetID := moduleIDFor(t, pkg1, "multimoduleproject.greet")
	consumerID := moduleIDFor(t, pkg1, "multimoduleproject.consumer")

	dir := multimoduleFixtureDir(t)
	greetURI := fileURI(t, "file://"+filepath.Join(dir, "modules", "greet", "greet.bal"))
	// Body-only edit: greet's signature (returns string) is unchanged, so its
	// fingerprint must be unchanged too.
	applyOpen(t, projSvc, greetURI, "public function greeting() returns string {\n    int x = \"hello\";\n    _ = x;\n    return \"hello\";\n}\n")

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg2 := proj.CurrentPackage()

	gen2 := newPackageDriver(pkg2, projSvc.OpenText, priorState, nil)
	gen2.advanceAll(stageCFGAnalyzed)
	if len(gen2.phase1Errored) != 0 {
		t.Fatalf("gen2 phase1Errored = %v, want empty (signature unchanged)", gen2.phase1Errored)
	}

	if gen2.drivers[greetID] == committed[greetID].driver {
		t.Error("greet (directly dirty) must NOT carry forward its old driver")
	}
	if gen2.drivers[consumerID] != committed[consumerID].driver {
		t.Error("consumer (dependent of greet, but greet's fingerprint is unchanged) must carry forward the exact same driver instance — item 4 pruning did not fire")
	}

	// The carried-forward modules' diagnostics must still be reachable
	// through the new generation (the trap: silently dropping them from
	// pd.drivers would make their diagnostics vanish from allDiagnostics).
	if n, ok := moduleDiagCount(t, gen2, pkg2, "multimoduleproject.standalone"); !ok || n != 1 {
		t.Errorf("gen2 standalone diagnostics = %d ok=%v, want 1 (carried forward, not lost)", n, ok)
	}
}

// TestPackageDriver_AllDirty_NoPriorState_FullResetScope verifies the "first
// generation ever" (or extractForURI's nil-priorState) case: every module is
// in the reset scope, matching pre-ticket-28 full-rebuild-every-time
// behavior exactly.
func TestPackageDriver_AllDirty_NoPriorState_FullResetScope(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	pd := newPackageDriver(pkg, projSvc.OpenText, nil, nil)
	modules := pd.topoModules()
	pd.ensureResetScope(modules, pkg.Resolution().ModuleDependencyGraph())

	for _, m := range modules {
		if !pd.resetScope[m.ModuleID()] {
			t.Errorf("module %v missing from reset scope with no prior state", m.ModuleID())
		}
	}
}

// TestPackageDriver_Aborted_StopsBetweenModules verifies design item 5:
// a stale() that flips true after the first module processed stops the
// Phase 1 loop before any further module is reached.
func TestPackageDriver_Aborted_StopsBetweenModules(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	calls := 0
	stale := func() bool {
		calls++
		return calls > 1
	}
	pd := newPackageDriver(pkg, projSvc.OpenText, nil, stale)
	pd.advanceAll(stageCFGAnalyzed)

	if !pd.aborted() {
		t.Fatal("expected aborted() = true")
	}
	if len(pd.drivers) != 1 {
		t.Fatalf("drivers reached = %d, want exactly 1 (aborted before a second module)", len(pd.drivers))
	}
}

// TestPackageDriver_AbortedFromStart_NoModulesReached verifies a
// generation superseded before it starts touches nothing.
func TestPackageDriver_AbortedFromStart_NoModulesReached(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	pd := newPackageDriver(pkg, projSvc.OpenText, nil, func() bool { return true })
	pd.advanceAll(stageCFGAnalyzed)

	if len(pd.drivers) != 0 {
		t.Errorf("drivers reached = %d, want 0 (aborted from the start)", len(pd.drivers))
	}
	if len(pd.phase1Errored) != 0 {
		t.Errorf("phase1Errored = %v, want empty (aborted from the start)", pd.phase1Errored)
	}
}
