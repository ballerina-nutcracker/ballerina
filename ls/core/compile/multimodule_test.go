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

// moduleDriverFor returns this generation's moduleDriver for moduleName,
// reaching directly into pd.drivers (package-internal — this test file is
// package compile) rather than through moduleDiagCount's diagnostics-only
// view, so a test can inspect pkgNode/units directly (ticket 38).
func moduleDriverFor(t *testing.T, pd *packageDriver, pkg *projects.Package, moduleName string) (*moduleDriver, bool) {
	t.Helper()
	module := pkg.ModuleByName(mustModuleName(t, pkg, moduleName))
	d, ok := pd.drivers[module.ModuleID()]
	return d, ok
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

// TestModuleDriver_OwnSyntaxError_RetainsValidSiblingsAndDependents verifies
// that a module-local syntax error does not gate that module's own Phase 1
// (runPhase1Module's doc comment: "Parser diagnostics from recovered units
// do not prevent a stage from executing"): the malformed declaration
// recovers as a clean BLangBadTopLevelNode that symbol/top-level-type
// resolution skip without adding a new error, so the module's valid sibling
// declaration is still resolved, its exports are still published, and its
// dependents still proceed through Phase 1. Only the parser's own
// diagnostic is published (publishableDiagnostics), never mixed with
// resolver/type diagnostics from resolving the rest of the file.
func TestModuleDriver_OwnSyntaxError_RetainsValidSiblingsAndDependents(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)
	// A valid sibling declaration (greeting) plus a genuine top-level syntax
	// error (a malformed variable declarator: a declared type and name with
	// no initializer expression) in the same file.
	brokenGreet := "public function greeting() returns string {\n    return \"hello\";\n}\n\nint target = ;\n"
	applyOpen(t, projSvc, greetURI, brokenGreet)

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg = proj.CurrentPackage()

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
	// Ticket 58 narrowed ensureLocalTypeResolved's gate to
	// !d.topLevelTypeResolutionUsable, so a lone parser diagnostic no longer
	// blocks Phase 2's local-type-resolution stage the way it used to;
	// semantic analysis still never runs, since ensureSemanticAnalyzed keeps
	// its own untouched blanket d.ctx.HasDiagnostics() gate.
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

// TestPackageDriver_CrossModuleImport_PublishesExportedSymbols verifies
// Phase 1 -> Phase 2 barrier correctness across a real multi-module package:
// consumer (which imports greet) must see greet's exported symbols and
// compile with zero diagnostics, and main (which imports consumer) likewise.
// The unrelated standalone module keeps its own fixed diagnostic.
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

	// Unlike projects/package_compilation.go:139-143, greet's Phase 1 failure
	// does not skip Phase 2 package-wide: standalone does not depend on greet,
	// so it still runs Phase 2 and surfaces its own fixed semantic error.
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

	// gen1's own driver set must be untouched by gen2 running.
	if n, ok := moduleDiagCount(t, gen1, pkg1, "multimoduleproject.greet"); !ok || n != 0 {
		t.Errorf("gen1 greet diagnostics after gen2 ran = %d ok=%v, want unchanged 0", n, ok)
	}
}

// TestPackageDriver_DependencyPublicSurfaceBroken_DependentDriverNeverExists
// makes ticket 38 scenario 2's "no driver, so no pkgNode and no units either"
// conclusion explicit, rather than inferred the way
// TestPackageDriver_DependencyPhase1Error_CascadesSkipWithoutPanic's
// moduleDiagCount ok=false checks do. Same fixture mutation as that test
// (greet's return type changed to an UndefinedType, breaking greet's public
// signature): consumer's dependency (greet) fails Phase 1, so
// dependencyErrored (multimodule.go) causes consumer to be skipped before
// driverFor is ever called — pd.drivers must have no entry for consumer at
// all, meaning there is no moduleDriver instance whose pkgNode/units a
// completion fallback (ticket 37's Option D) could ever read for consumer in
// this generation.
func TestPackageDriver_DependencyPublicSurfaceBroken_DependentDriverNeverExists(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)
	brokenGreet := "public function greeting() returns UndefinedType {\n    return \"hello\";\n}\n"
	applyOpen(t, projSvc, greetURI, brokenGreet)

	proj, err := projSvc.Project(greetURI)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg = proj.CurrentPackage()

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

// TestPackageDriver_DependencyBodyOnlyError_DependentCarriesForwardPkgNode
// resolves ticket 37's open question for ticket 38 scenario 3: when a
// dependency's error is body-only (its exported public surface is
// unchanged), is the dependent module pulled into the reset scope and
// rebuilt, or does it adopt its own prior generation via
// adoptCarriedForward and keep a fully usable pkgNode untouched?
//
// Answer, established by direct observation below: the dependent (consumer)
// is NOT pulled into the reset scope (pd.resetScope[consumerID] is false)
// and adopts the exact same *moduleDriver instance* its previous generation
// finished with — its pkgNode is still populated and its diagnostics are
// still 0, exactly as if nothing had happened.
//
// This is not a coincidence of this one fixture: recordFingerprintAndPrune
// (multimodule.go) computes a module's fingerprint immediately after Phase 1
// (parse -> symbol resolution -> top-level/public-node type resolution),
// strictly before Phase 2 (local/body type resolution and semantic
// analysis) ever runs (advanceAll, multimodule.go). A "body-only" error, by
// construction, only ever surfaces during Phase 2
// (semantics.ResolvePrivateNodesTypes / AnalyzeSemantics — semantics.go's
// doc comment), so Phase 1 always completes clean for a genuinely body-only
// error, and the fingerprint computed from that clean context is always
// valid and (for an unchanged signature) unchanged. There is no reachable
// case in this architecture where a body-only error causes phase1Errored —
// that would require Phase 1 itself to inspect function-body content, which
// it structurally does not (ResolveSymbols/ResolvePublicNodeTypes only see
// package symbols and public node types, not private/body nodes). So
// scenario 2's cascade-skip path and scenario 3's carry-forward path are
// mutually exclusive by construction, not just by this fixture's luck.
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
	// Body-only edit: greet's signature (returns string) is unchanged, so
	// this only ever surfaces in AnalyzeSemantics (Phase 2), never in Phase
	// 1's symbol/top-level-type resolution.
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

	// greet itself, by contrast, is directly dirty (its own text changed) and
	// must be rebuilt fresh, with its new body error visible.
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

// TestModuleDriver_ZeroDocuments_NeverConstructedThroughPublicAPI documents,
// rather than tests, ticket 38 scenario 4 (d.stage < stageParsed because
// ensureParsed built zero units). This is not reasonably constructible
// through the public projects/workspace API:
//
//   - ensureParsed (stage.go:191-229) only ends up with zero units if either
//     module.DocumentIDs() is empty or every module.Document(docID) is nil,
//     or parser.GetSyntaxTree returns a nil *st.SyntaxTree.
//   - parser.GetSyntaxTree (parser.go:14682-14692) never returns a nil tree
//     or a non-nil error for any input, including an empty string — so that
//     branch is dead in practice, confirmed by reading its implementation.
//   - A named or default module is only discovered by projects' loader when
//     it has at least one .bal file backing it; there is no public
//     workspace/projects API to open a module with zero documents (a module
//     directory containing zero .bal files does not become a Module at all).
//
// So the only way to reach d.stage < stageParsed after advanceTo is called
// is a module that was never really loaded in the first place — not a real
// LS scenario, and not something this test file forces through unexported
// internals. A completion fallback (ticket 37's Option D) reaching this
// branch would only ever see the zero-value moduleDriver{} it started from
// (pkgNode nil, units nil, stage stageUnstarted), which is already
// vacuously true of every fresh moduleDriver before advanceTo runs at all.
