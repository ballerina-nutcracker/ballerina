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
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// newProjectOnlyService builds a *workspace.ProjectService with no
// CompilationService wired to its bus — deliberately, not an oversight.
// newTestServices (compile_test.go) wires a real CompilationService with
// WithDebounce(0), which fires a background compile (realCompilePackage,
// itself a packageDriver run) on every published edit. These packageDriver
// tests drive their own foreground packageDriver over the same package/
// environment; using newTestServices here would run two independent
// packageDriver instances concurrently over the same shared environment,
// which production handles safely (CompilerEnvironment guards its own shared
// state field-by-field — see multimodule.go's Concurrency doc comment) but
// would make these tests' assertions racy against a concurrently-mutating
// background compile — so these tests avoid the background compiler
// entirely instead, to keep foreground driver behavior deterministic.
func newProjectOnlyService(t *testing.T) *workspace.ProjectService {
	t.Helper()
	platform, _ := palnative.NewPlatform()
	bus := event.New()
	t.Cleanup(func() { bus.Close() })
	return workspace.New(platform, bus)
}

// multimoduleFixtureDir returns the absolute path to the on-disk fixture at
// testdata/multimodule: a real package with a genuine cross-module import
// chain (main -> consumer -> greet) plus an unrelated sibling module
// (standalone) with its own fixed diagnostic, used to verify the Phase 1 ->
// Phase 2 barrier and per-generation diagnostic scoping without touching
// projects/testdata fixtures owned by another package.
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

// openMultimoduleFixture opens main.bal from the multimodule fixture,
// loading the whole package (consumer.bal, greet.bal and standalone.bal read
// from disk), and returns the resulting *projects.Package.
func openMultimoduleFixture(t *testing.T, projSvc *workspace.ProjectService) (*projects.Package, uri.DocumentURI) {
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

// mustModuleName finds the ModuleName among pkg's modules whose full String()
// equals name (e.g. "multimoduleproject.consumer"), so tests don't need to
// hand-construct a projects.ModuleName (unexported-field-only construction).
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

// TestPackageDriver_CrossModuleImport_PublishesExportedSymbols verifies
// Phase 1 -> Phase 2 barrier correctness across a real multi-module package:
// consumer (which imports greet) must see greet's exported symbols and
// compile with zero diagnostics, and main (which imports consumer) likewise.
// The unrelated standalone module keeps its own fixed diagnostic.
func TestPackageDriver_CrossModuleImport_PublishesExportedSymbols(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	pd := newPackageDriver(pkg)
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

// TestPackageDriver_DependencyPhase1Error_CascadesSkipWithoutPanic verifies
// that a Phase 1 (symbol/top-level-type resolution) error in a dependency
// (greet) causes its dependent (consumer) and consumer's own dependent
// (main) to be skipped entirely — never reaching driverFor/ensureParsed —
// without a nil-pointer panic, mirroring
// projects/package_compilation.go:100-111's dependencyErrored gate. The
// unrelated standalone module is unaffected.
func TestPackageDriver_DependencyPhase1Error_CascadesSkipWithoutPanic(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)
	brokenGreet := "public function greeting() returns UndefinedType {\n    return \"hello\";\n}\n"
	applyOpen(t, projSvc, greetURI, brokenGreet)

	// Re-resolve the package: applyOpen's modifier chain publishes a new
	// current package on the same project.
	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg = proj.CurrentPackage()

	pd := newPackageDriver(pkg)
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

	for _, name := range []string{"multimoduleproject.consumer", "multimoduleproject"} {
		if _, ok := moduleDiagCount(t, pd, pkg, name); ok {
			t.Errorf("module %q was driven despite its errored dependency; expected it to be skipped entirely", name)
		}
	}

	// Phase 2 (which is where standalone's own semantic error would surface)
	// is skipped across the whole package once any module fails Phase 1,
	// mirroring projects/package_compilation.go:139-143's package-wide gate
	// ("subsequent stages operate on assumptions that top-level types are
	// fully resolved across the whole package"). standalone's Phase 1 still
	// ran cleanly (it has no dependency on greet), so its driver exists with
	// zero diagnostics — Phase 1 alone does not surface its body-only error.
	n, ok = moduleDiagCount(t, pd, pkg, "multimoduleproject.standalone")
	if !ok {
		t.Fatal("module \"multimoduleproject.standalone\" was never driven")
	}
	if n != 0 {
		t.Errorf("standalone diagnostics = %d, want 0 (Phase 2 skipped package-wide; its error is a Phase 2/AnalyzeSemantics diagnostic)", n)
	}
	standaloneModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.standalone"))
	if pd.phase1Errored[standaloneModule.ModuleID()] {
		t.Error("standalone must not be marked phase1Errored — its own Phase 1 succeeded")
	}
}

// TestPackageDriver_EditOneModule_UnrelatedDiagnosticsSurviveAcrossGeneration
// verifies: an edit to greet's body only (its public signature — and
// therefore its exported symbol space — is unchanged) produces a new
// diagnostic on greet in the next generation, while consumer (a downstream
// dependent whose Phase 1 view of greet did not change) and the unrelated
// standalone module keep their diagnostics unchanged, with no
// cross-generation accumulation (each generation uses a fresh packageDriver
// with fresh per-module CompilerContexts).
func TestPackageDriver_EditOneModule_UnrelatedDiagnosticsSurviveAcrossGeneration(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg1, _ := openMultimoduleFixture(t, projSvc)

	gen1 := newPackageDriver(pkg1)
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
	// Body-only edit: the signature (returns string) is unchanged, so
	// greet's exported symbol space is unchanged too — only its own
	// AnalyzeSemantics-stage diagnostic count should change.
	editedGreet := "public function greeting() returns string {\n    int x = \"hello\";\n    _ = x;\n    return \"hello\";\n}\n"
	applyOpen(t, projSvc, greetURI, editedGreet)

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg2 := proj.CurrentPackage()

	gen2 := newPackageDriver(pkg2)
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

	// gen1's own driver set must be untouched by gen2 running.
	if n, ok := moduleDiagCount(t, gen1, pkg1, "multimoduleproject.greet"); !ok || n != 0 {
		t.Errorf("gen1 greet diagnostics after gen2 ran = %d ok=%v, want unchanged 0", n, ok)
	}
}
