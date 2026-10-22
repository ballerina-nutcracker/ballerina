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
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// newProjectOnlyService creates a workspace without a background compiler.
func newProjectOnlyService(t *testing.T) *workspace.ProjectService {
	t.Helper()
	bus := event.New()
	t.Cleanup(func() { bus.Close() })
	return workspace.New(os.DirFS, bus)
}

// multimoduleFixtureDir locates the main -> consumer -> greet fixture and its
// independent standalone module with one semantic diagnostic.
func multimoduleFixtureDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "multimodule"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return dir
}

func mustReadFixture(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(b)
}

func openMultimoduleFixture(t *testing.T, projSvc *workspace.ProjectService) (*projects.Package, workspace.DocumentURI) {
	t.Helper()
	dir := multimoduleFixtureDir(t)
	mainPath := filepath.Join(dir, "main.bal")
	u := fileURI(t, "file://"+mainPath)
	applyOpen(t, projSvc, u, mustReadFixture(t, mainPath))
	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg := proj.CurrentPackage()
	if pkg == nil {
		t.Fatal("CurrentPackage() = nil")
	}
	return pkg, u
}

func moduleDiagCount(t *testing.T, pd *packageDriver, pkg *projects.Package, moduleName string) (int, bool) {
	t.Helper()
	module := pkg.ModuleByName(mustModuleName(t, pkg, moduleName))
	d, ok := pd.drivers[module.ModuleID()]
	if !ok {
		return 0, false
	}
	return len(d.diagnosticContext().Diagnostics()), true
}

func moduleDriverFor(t *testing.T, pd *packageDriver, pkg *projects.Package, moduleName string) (*moduleDriver, bool) {
	t.Helper()
	module := pkg.ModuleByName(mustModuleName(t, pkg, moduleName))
	d, ok := pd.drivers[module.ModuleID()]
	return d, ok
}

func mustModuleName(t *testing.T, pkg *projects.Package, name string) projects.ModuleName {
	t.Helper()
	for _, id := range pkg.ModuleIDs() {
		module := pkg.Module(id)
		if module == nil {
			continue
		}
		if module.Descriptor().Name().String() == name {
			return module.Descriptor().Name()
		}
	}
	t.Fatalf("no module named %q in package", name)
	return projects.ModuleName{}
}

func TestModuleDriver_OwnSyntaxError_RetainsValidSiblingsAndDependents(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	openMultimoduleFixture(t, projSvc)

	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)
	brokenGreet := "public function greeting() returns string {\n    return \"hello\";\n}\n\nfunction target(int {\n}\n"
	applyOpen(t, projSvc, greetURI, brokenGreet)

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg := proj.CurrentPackage()

	pd := newPackageDriver(pkg, projSvc.OpenText, nil, nil)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("advanceAll panicked on a module-local syntax error: %v", r)
			}
		}()
		pd.advanceAll(stageCFGAnalyzed)
	}()

	greetModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.greet"))
	if pd.phase1Errored[greetModule.ModuleID()] {
		t.Fatal("a lone parser diagnostic must not fail Phase 1 for the module")
	}

	d, ok := moduleDriverFor(t, pd, pkg, "multimoduleproject.greet")
	if !ok {
		t.Fatal("greet's driver must exist")
	}
	if !d.diagnosticContext().HasDiagnostics() {
		t.Fatal("greet's parser diagnostic must be retained")
	}
	// Parser diagnostics permit local type resolution, but not semantic analysis.
	if d.currentStage() != stageLocalTypeResolved {
		t.Fatalf("greet stage = %v, want stageLocalTypeResolved", d.currentStage())
	}
	if !d.localTypeResolutionUsable {
		t.Error("a bad top-level node alone must not make local type resolution unusable")
	}
	if d.pkgNode == nil {
		t.Fatal("pkgNode must be built despite the parser diagnostic")
	}
	if len(d.units) != 2 {
		t.Fatalf("units = %d, want 2 (greet.bal and greet_util.bal)", len(d.units))
	}
	if got := d.publishableDiagnostics(); len(got) != d.parserDiagnosticCount {
		t.Errorf("publishableDiagnostics() = %d diagnostics, want exactly the %d parser diagnostics", len(got), d.parserDiagnosticCount)
	}

	if _, ok := pd.publicSymbols[packageIdentifierFor(greetModule)]; !ok {
		t.Error("greet exports must still be published despite the parser diagnostic")
	}
	if _, ok := moduleDriverFor(t, pd, pkg, "multimoduleproject.consumer"); !ok {
		t.Error("consumer must still proceed through Phase 1 since greet did not fail it")
	}
}

func TestPackageDriver_CrossModuleImport_PublishesExportedSymbols(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	pd := newPackageDriver(pkg, projSvc.OpenText, nil, nil)
	pd.advanceAll(stageCFGAnalyzed)

	if len(pd.phase1Errored) != 0 {
		t.Fatalf("phase1Errored = %v, want empty", pd.phase1Errored)
	}

	for _, name := range []string{"multimoduleproject", "multimoduleproject.consumer", "multimoduleproject.greet"} {
		n, ok := moduleDiagCount(t, pd, pkg, name)
		if !ok {
			t.Fatalf("module %q was never driven", name)
		}
		if n != 0 {
			t.Errorf("module %q diagnostics = %d, want 0", name, n)
		}
	}

	n, ok := moduleDiagCount(t, pd, pkg, "multimoduleproject.standalone")
	if !ok {
		t.Fatal("module \"multimoduleproject.standalone\" was never driven")
	}
	if n != 1 {
		t.Errorf("standalone diagnostics = %d, want 1 (its own fixed semantic error)", n)
	}
}

func TestPackageDriver_DependencyPhase1Error_CascadesSkipWithoutPanic(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	openMultimoduleFixture(t, projSvc)

	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)
	brokenGreet := "public function greeting() returns UndefinedType {\n    return \"hello\";\n}\n"
	applyOpen(t, projSvc, greetURI, brokenGreet)

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg := proj.CurrentPackage()

	pd := newPackageDriver(pkg, projSvc.OpenText, nil, nil)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("advanceAll panicked on a Phase 1 dependency error: %v", r)
			}
		}()
		pd.advanceAll(stageCFGAnalyzed)
	}()

	greetModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.greet"))
	if !pd.phase1Errored[greetModule.ModuleID()] {
		t.Error("greet must be marked phase1Errored")
	}
	n, ok := moduleDiagCount(t, pd, pkg, "multimoduleproject.greet")
	if !ok || n == 0 {
		t.Errorf("greet diagnostics = %d ok=%v, want >=1", n, ok)
	}
	greetDriver, ok := moduleDriverFor(t, pd, pkg, "multimoduleproject.greet")
	if !ok {
		t.Fatal("greet driver must exist")
	}
	if greetDriver.currentStage() != stageSymbolResolved {
		t.Errorf("greet stage = %v, want stageSymbolResolved: its unresolved type reference keeps it out of type resolution", greetDriver.currentStage())
	}
	if greetDriver.symbolResolutionUsable {
		t.Error("greet symbol resolution must be unusable after resolver error")
	}

	for _, name := range []string{"multimoduleproject.consumer", "multimoduleproject"} {
		if _, ok := moduleDiagCount(t, pd, pkg, name); ok {
			t.Errorf("module %q was driven despite its errored dependency; expected it to be skipped entirely", name)
		}
	}

	n, ok = moduleDiagCount(t, pd, pkg, "multimoduleproject.standalone")
	if !ok {
		t.Fatal("module \"multimoduleproject.standalone\" was never driven")
	}
	if n != 1 {
		t.Errorf("standalone diagnostics = %d, want 1 (its own fixed semantic error)", n)
	}
	standaloneModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.standalone"))
	if pd.phase1Errored[standaloneModule.ModuleID()] {
		t.Error("standalone must not be marked phase1Errored — its own Phase 1 succeeded")
	}
}

func TestPackageDriver_EditOneModule_UnrelatedDiagnosticsSurviveAcrossGeneration(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg1, _ := openMultimoduleFixture(t, projSvc)

	gen1 := newPackageDriver(pkg1, projSvc.OpenText, nil, nil)
	gen1.advanceAll(stageCFGAnalyzed)
	if len(gen1.phase1Errored) != 0 {
		t.Fatalf("gen1 phase1Errored = %v, want empty", gen1.phase1Errored)
	}
	if n, ok := moduleDiagCount(t, gen1, pkg1, "multimoduleproject.greet"); !ok || n != 0 {
		t.Fatalf("gen1 greet diagnostics = %d ok=%v, want 0", n, ok)
	}
	if n, ok := moduleDiagCount(t, gen1, pkg1, "multimoduleproject.standalone"); !ok || n != 1 {
		t.Fatalf("gen1 standalone diagnostics = %d ok=%v, want 1", n, ok)
	}

	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)
	editedGreet := "public function greeting() returns string {\n    int x = \"hello\";\n    _ = x;\n    return \"hello\";\n}\n"
	applyOpen(t, projSvc, greetURI, editedGreet)

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg2 := proj.CurrentPackage()

	gen2 := newPackageDriver(pkg2, projSvc.OpenText, nil, nil)
	gen2.advanceAll(stageCFGAnalyzed)
	if len(gen2.phase1Errored) != 0 {
		t.Fatalf("gen2 phase1Errored = %v, want empty (signature unchanged)", gen2.phase1Errored)
	}

	if n, ok := moduleDiagCount(t, gen2, pkg2, "multimoduleproject.greet"); !ok || n != 1 {
		t.Errorf("gen2 greet diagnostics = %d ok=%v, want 1 (new body error, not accumulated)", n, ok)
	}
	if n, ok := moduleDiagCount(t, gen2, pkg2, "multimoduleproject.consumer"); !ok || n != 0 {
		t.Errorf("gen2 consumer diagnostics = %d ok=%v, want 0 (unaffected by greet's body-only edit)", n, ok)
	}
	if n, ok := moduleDiagCount(t, gen2, pkg2, "multimoduleproject.standalone"); !ok || n != 1 {
		t.Errorf("gen2 standalone diagnostics = %d ok=%v, want 1 (unchanged, not duplicated)", n, ok)
	}

	if n, ok := moduleDiagCount(t, gen1, pkg1, "multimoduleproject.greet"); !ok || n != 0 {
		t.Errorf("gen1 greet diagnostics after gen2 ran = %d ok=%v, want unchanged 0", n, ok)
	}
}

func TestPackageDriver_DependencyPublicSurfaceBroken_DependentDriverNeverExists(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	openMultimoduleFixture(t, projSvc)

	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)
	brokenGreet := "public function greeting() returns UndefinedType {\n    return \"hello\";\n}\n"
	applyOpen(t, projSvc, greetURI, brokenGreet)

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg := proj.CurrentPackage()

	pd := newPackageDriver(pkg, projSvc.OpenText, nil, nil)
	pd.advanceAll(stageCFGAnalyzed)

	greetModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.greet"))
	if !pd.phase1Errored[greetModule.ModuleID()] {
		t.Fatal("greet must be marked phase1Errored (fixture assumption broken)")
	}

	d, ok := moduleDriverFor(t, pd, pkg, "multimoduleproject.consumer")
	if ok {
		t.Fatalf("consumer's driver exists (%+v) — want no entry at all: dependencyErrored must skip consumer before driverFor is ever called", d)
	}
	if d != nil {
		t.Error("consumer's driver must be nil when ok is false")
	}
}

func TestPackageDriver_DependencyBodyOnlyError_DependentCarriesForwardPkgNode(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg1, _ := openMultimoduleFixture(t, projSvc)

	gen1 := newPackageDriver(pkg1, projSvc.OpenText, nil, nil)
	gen1.advanceAll(stageCFGAnalyzed)
	if len(gen1.phase1Errored) != 0 {
		t.Fatalf("gen1 phase1Errored = %v, want empty", gen1.phase1Errored)
	}
	committed := gen1.snapshotForCommit()
	priorState := &packageGenState{modules: committed}

	consumerGen1, ok := moduleDriverFor(t, gen1, pkg1, "multimoduleproject.consumer")
	if !ok {
		t.Fatal("gen1 consumer driver must exist")
	}
	if consumerGen1.pkgNode == nil {
		t.Fatal("gen1 consumer pkgNode must be populated (fixture assumption broken)")
	}
	greetGen1, ok := moduleDriverFor(t, gen1, pkg1, "multimoduleproject.greet")
	if !ok {
		t.Fatal("gen1 greet driver must exist")
	}

	dir := multimoduleFixtureDir(t)
	greetURI := fileURI(t, "file://"+filepath.Join(dir, "modules", "greet", "greet.bal"))
	// The type mismatch surfaces during semantic analysis.
	applyOpen(t, projSvc, greetURI, "public function greeting() returns string {\n    int x = \"hello\";\n    _ = x;\n    return \"hello\";\n}\n")

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg2 := proj.CurrentPackage()

	consumerID := moduleIDFor(t, pkg2, "multimoduleproject.consumer")

	gen2 := newPackageDriver(pkg2, projSvc.OpenText, priorState, nil)
	gen2.advanceAll(stageCFGAnalyzed)
	if len(gen2.phase1Errored) != 0 {
		t.Fatalf("gen2 phase1Errored = %v, want empty — a body-only error must never fail Phase 1", gen2.phase1Errored)
	}

	if gen2.resetScope[consumerID] {
		t.Error("consumer must NOT be in gen2's reset scope — greet's fingerprint (public surface) is unchanged, so item 4 pruning must have removed it")
	}

	consumerGen2, ok := moduleDriverFor(t, gen2, pkg2, "multimoduleproject.consumer")
	if !ok {
		t.Fatal("consumer's driver must still exist in gen2 (carried forward, not dropped)")
	}
	if consumerGen2 != consumerGen1 {
		t.Error("consumer must carry forward the exact same *moduleDriver instance from gen1 — adoptCarriedForward must have run, not a fresh rebuild")
	}
	if consumerGen2.pkgNode == nil {
		t.Error("carried-forward consumer's pkgNode must still be populated and usable, untouched by greet's body-only edit")
	}
	if n, ok := moduleDiagCount(t, gen2, pkg2, "multimoduleproject.consumer"); !ok || n != 0 {
		t.Errorf("gen2 consumer diagnostics = %d ok=%v, want 0 (unaffected)", n, ok)
	}

	greetGen2, ok := moduleDriverFor(t, gen2, pkg2, "multimoduleproject.greet")
	if !ok {
		t.Fatal("gen2 greet driver must exist")
	}
	if greetGen2 == greetGen1 {
		t.Error("greet (directly dirty) must NOT carry forward its old driver instance")
	}
	if n, ok := moduleDiagCount(t, gen2, pkg2, "multimoduleproject.greet"); !ok || n != 1 {
		t.Errorf("gen2 greet diagnostics = %d ok=%v, want 1 (its own new body error)", n, ok)
	}
}
