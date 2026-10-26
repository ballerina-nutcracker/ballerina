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

func TestPackageDriver_UnchangedModule_CarriesForwardSameDriverInstance(t *testing.T) {
	t.Parallel()
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
	greetURI := fileURI(t, filepath.Join(dir, "modules", "greet", "greet.bal"))
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

	if n, ok := moduleDiagCount(t, gen2, pkg2, "multimoduleproject.standalone"); !ok || n != 1 {
		t.Errorf("gen2 standalone diagnostics = %d ok=%v, want 1 (carried forward, not lost)", n, ok)
	}
}

func TestPackageDriver_AllDirty_NoPriorState_FullResetScope(t *testing.T) {
	t.Parallel()
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

func TestPackageDriver_Aborted_StopsBetweenModules(t *testing.T) {
	t.Parallel()
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	var pd *packageDriver
	stale := func() bool { return pd != nil && len(pd.drivers) >= 1 }
	pd = newPackageDriver(pkg, projSvc.OpenText, nil, stale)
	pd.advanceAll(stageCFGAnalyzed)

	if !pd.aborted() {
		t.Fatal("expected aborted() = true")
	}
	if len(pd.drivers) != 1 {
		t.Fatalf("drivers reached = %d, want exactly 1 (aborted before a second module)", len(pd.drivers))
	}
}

func TestPackageDriver_AbortedFromStart_NoModulesReached(t *testing.T) {
	t.Parallel()
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
