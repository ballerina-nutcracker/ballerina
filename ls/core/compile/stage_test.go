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
	"testing"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// validSource has no diagnostics through any stage.
const validSource = "public function main() {}\n"

const duplicateTopLevelSource = "public function duplicate() {}\npublic function duplicate() {}\n"
const recursivePublicTypeSource = "public type Invalid Invalid;\n"
const unusedTopLevelSource = "const int UNUSED = 1;\n"
const malformedBodySource = "function broken() {\n    int value = ;\n}\n"
const malformedSignatureAndBodySource = "function broken(int {\n    int value = ;\n}\n"
const malformedTopLevelSource = "int value = ;\n"

// breakOutsideLoopSource triggers a diagnostic from inside
// ResolvePrivateNodesTypes itself (Phase 2 / stage 5's block-statement
// resolution reports "break statement not allowed outside loop") rather than
// from symbol resolution (stage 3) or semantic analysis (stage 6) — the
// signature and every earlier stage resolve this module diagnostic-free.
const breakOutsideLoopSource = "function main() {\n    break;\n}\n"

// oneSemanticErrorSource produces exactly one SEMANTIC_ERROR diagnostic
// ("incompatible type"), surfacing only during AnalyzeSemantics (stage 6) —
// confirmed empirically: parsing, symbol resolution, top-level and local type
// resolution all complete diagnostic-free for this fixture; only
// AnalyzeSemantics reports the type mismatch.
const oneSemanticErrorSource = "public function main() {\n    int x = \"hello\";\n    _ = x;\n}\n"

// defaultModuleFor returns proj's default module plus the project's shared
// compiler environment.
func defaultModuleFor(t *testing.T, proj projects.Project) (*projects.Module, *context.CompilerEnvironment) {
	t.Helper()
	pkg := proj.CurrentPackage()
	if pkg == nil {
		t.Fatal("CurrentPackage() = nil")
	}
	module := pkg.DefaultModule()
	if module == nil {
		t.Fatal("DefaultModule() = nil")
	}
	return module, proj.Environment().CompilerEnvironment()
}

func TestModuleDriver_AdvancesOnlyToRequestedStage(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, validSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageTopLevelTypeResolved, module, newModuleResolutionInput("", nil, nil))

	if got := d.currentStage(); got != stageTopLevelTypeResolved {
		t.Fatalf("currentStage() = %v, want stageTopLevelTypeResolved", got)
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want a package built during symbol resolution")
	}
	if d.cfg != nil {
		t.Error("cfg is set, but CFGBuilt/CFGAnalyzed were never requested")
	}
	if d.diagnosticContext().HasDiagnostics() {
		t.Errorf("unexpected diagnostics: %v", d.diagnosticContext().Diagnostics())
	}
}

func TestModuleDriver_ResolverErrorDoesNotAdvancePhase1(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, duplicateTopLevelSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageTopLevelTypeResolved, module, newModuleResolutionInput("", nil, nil))

	if d.currentStage() != stageSymbolResolved {
		t.Fatalf("currentStage() = %v, want stageSymbolResolved after resolver error", d.currentStage())
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want an executed partial package")
	}
	if d.symbolResolutionUsable {
		t.Error("resolver error must make symbol resolution unusable")
	}
	if !d.diagnosticContext().HasErrors() {
		t.Error("duplicate top-level declaration must report a resolver error")
	}
}

// TestModuleDriver_UnresolvedReferenceReachesLocalTypeResolved covers a
// cleanly parsed module referencing an unknown name: the reference is bound
// to a stand-in symbol (bindUnresolvedReferences), so type resolution runs
// without panicking, while semantic analysis still never runs.
func TestModuleDriver_UnresolvedReferenceReachesLocalTypeResolved(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, "public function main() {\n    int value = missing;\n    int other = value;\n}\n")

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageCFGAnalyzed, module, newModuleResolutionInput("", nil, nil))

	if d.currentStage() != stageLocalTypeResolved {
		t.Fatalf("currentStage() = %v, want stageLocalTypeResolved for an unresolved reference", d.currentStage())
	}
	if d.symbolResolutionUsable {
		t.Error("the unknown symbol must make symbol resolution unusable")
	}
	if d.cfg != nil {
		t.Error("cfg is set, but semantic analysis/CFG must never run on a module with errors")
	}
}

// TestModuleDriver_UnusedSymbolDoesNotBlockTypeResolution covers a module
// whose only error is an unused symbol: symbol resolution is unusable for
// dependents, but no symbol is left unresolved, so type resolution still runs.
func TestModuleDriver_UnusedSymbolDoesNotBlockTypeResolution(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, unusedTopLevelSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageTopLevelTypeResolved, module, newModuleResolutionInput("", nil, nil))

	if d.currentStage() != stageTopLevelTypeResolved {
		t.Fatalf("currentStage() = %v, want stageTopLevelTypeResolved despite unused-symbol diagnostic", d.currentStage())
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want the symbol-resolved package")
	}
	if d.symbolResolutionUsable {
		t.Error("unused-symbol diagnostic must make symbol resolution unusable for Phase 1")
	}
	if !d.diagnosticContext().HasErrors() {
		t.Error("unused top-level symbol must retain its diagnostic")
	}
}

func TestModuleDriver_PublicTypeErrorDoesNotAdvancePhase1(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, recursivePublicTypeSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageTopLevelTypeResolved, module, newModuleResolutionInput("", nil, nil))

	if d.currentStage() != stageTopLevelTypeResolved {
		t.Fatalf("currentStage() = %v, want stageTopLevelTypeResolved after public-type error", d.currentStage())
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want the successfully symbol-resolved package")
	}
	if !d.symbolResolutionUsable {
		t.Error("public-type error must not retroactively make symbol resolution unusable")
	}
	if d.topLevelTypeResolutionUsable {
		t.Error("public-type error must make top-level type resolution unusable")
	}
	if !d.diagnosticContext().HasErrors() {
		t.Error("recursive public type must report a public-type error")
	}
}

// TestModuleDriver_MalformedFunctionSignatureRunsSymbolResolutionButUnusable
// covers a function whose own signature is unparseable, alongside a valid
// sibling. Parser diagnostics no longer gate ensureSymbolResolved (removed
// along with the bad-node filtering this driver used to do): symbol
// resolution still runs so a module's scope (and thus e.g. completion) stays
// available. Here it also surfaces a genuine new resolver diagnostic (an
// unused-variable check inside the malformed body), which makes this stage's
// own result unusable for dependents -- same mechanism
// TestModuleDriver_ResolverErrorDoesNotAdvancePhase1 covers for a clean
// duplicate-declaration error, except that a module with bad AST nodes still
// continues to local type resolution. publishableDiagnostics
// hides that resolver diagnostic because the module already has a parser
// diagnostic (see parserDiagnosticCount's doc comment on moduleDriver):
// mixing "missing close paren token" with "unused variable" would be noise
// from resolving a subtree the parser already gave up on.
func TestModuleDriver_MalformedFunctionSignatureRunsSymbolResolutionButUnusable(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, malformedSignatureAndBodySource+"\npublic function sibling() {}\n")

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageSymbolResolved, module, newModuleResolutionInput("", nil, nil))

	if d.currentStage() != stageSymbolResolved {
		t.Fatalf("currentStage() = %v, want stageSymbolResolved despite malformed function signature", d.currentStage())
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want a package built despite the parser diagnostic")
	}
	if d.symbolResolutionUsable {
		t.Error("the malformed body's resolver diagnostic must make symbol resolution unusable")
	}
	if got := d.publishableDiagnostics(); len(got) != d.parserDiagnosticCount {
		t.Errorf("publishableDiagnostics() = %d diagnostics, want exactly the %d parser diagnostics", len(got), d.parserDiagnosticCount)
	}
}

// TestModuleDriver_MalformedTopLevelReachesTopLevelTypeResolved covers a
// fully unparseable top-level declaration (int value = ;): the parser
// recovers it as a BLangBadTopLevelNode, which symbol resolution and package
// assembly both skip cleanly (no symbol allocated, no new diagnostic), so
// this module's Phase 1 proceeds all the way through top-level type
// resolution using only the parser's own diagnostic.
func TestModuleDriver_MalformedTopLevelReachesTopLevelTypeResolved(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, malformedTopLevelSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	input := newModuleResolutionInput("", nil, nil)
	d.advanceTo(stageTopLevelTypeResolved, module, input)

	if d.currentStage() != stageTopLevelTypeResolved {
		t.Fatalf("currentStage() = %v, want stageTopLevelTypeResolved for a cleanly-recovered bad top-level node", d.currentStage())
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want a package built despite the parser diagnostic")
	}
	if !d.symbolResolutionUsable || !d.topLevelTypeResolutionUsable {
		t.Error("a bad top-level node alone must not add a new resolver/type error")
	}
	if !d.diagnosticContext().HasErrors() {
		t.Error("parser diagnostic must be retained")
	}
}

// TestModuleDriver_MalformedTopLevelReachesLocalTypeResolved covers ticket
// 58: a module whose only diagnostic is the parser's own (from the recovered
// bad top-level node) must still reach stageLocalTypeResolved, since Phase 2
// itself adds no new error for this recovery-mode node (ticket 40). Semantic
// analysis must not run even so -- it stays gated on the untouched blanket
// d.ctx.HasDiagnostics() check, which still sees the pre-existing parser
// diagnostic.
func TestModuleDriver_MalformedTopLevelReachesLocalTypeResolved(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, malformedTopLevelSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	input := newModuleResolutionInput("", nil, nil)
	d.advanceTo(stageSemanticAnalyzed, module, input)

	if d.currentStage() != stageLocalTypeResolved {
		t.Fatalf("currentStage() = %v, want stageLocalTypeResolved for a cleanly-recovered bad top-level node", d.currentStage())
	}
	if !d.localTypeResolutionUsable {
		t.Error("a bad top-level node alone must not make local type resolution unusable")
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want a package built despite the parser diagnostic")
	}
	if !d.diagnosticContext().HasErrors() {
		t.Error("parser diagnostic must be retained")
	}
}

// TestModuleDriver_LocalTypeErrorStopsBeforeSemanticAnalysis covers the other
// half of ticket 58: a module where Phase 2 (ResolvePrivateNodesTypes) itself
// reports a new error ("break statement not allowed outside loop") still
// reaches stageLocalTypeResolved -- mirroring how ensureTopLevelTypeResolved
// advances d.stage even when its own result is unusable -- but with
// localTypeResolutionUsable left false, which keeps semantic analysis's own
// blanket HasDiagnostics() gate from ever letting it run.
func TestModuleDriver_LocalTypeErrorStopsBeforeSemanticAnalysis(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, breakOutsideLoopSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	input := newModuleResolutionInput("", nil, nil)
	d.advanceTo(stageSemanticAnalyzed, module, input)

	if d.currentStage() != stageLocalTypeResolved {
		t.Fatalf("currentStage() = %v, want stageLocalTypeResolved (semantic analysis must not run)", d.currentStage())
	}
	if d.localTypeResolutionUsable {
		t.Error("break-outside-loop must make local type resolution unusable")
	}
	if d.cfg != nil {
		t.Error("cfg is set, but semantic analysis/CFG must never have run")
	}
	if !d.diagnosticContext().HasErrors() {
		t.Error("break-outside-loop must report a semantic error")
	}
}

// TestModuleDriver_MalformedFunctionBodyReachesLocalTypeResolved covers a
// function with a valid signature but a malformed body (missing initializer
// expression). Walking the body surfaces new resolver diagnostics (unknown
// symbol / unused variable), so symbol resolution is unusable for dependents,
// but a module with bad AST nodes still runs through local type resolution
// (moduleDriver.hasBadNodes) and never reaches semantic analysis.
func TestModuleDriver_MalformedFunctionBodyReachesLocalTypeResolved(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, malformedBodySource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageCFGAnalyzed, module, newModuleResolutionInput("", nil, nil))

	if d.currentStage() != stageLocalTypeResolved {
		t.Fatalf("currentStage() = %v, want stageLocalTypeResolved for malformed function body", d.currentStage())
	}
	if d.cfg != nil {
		t.Error("cfg is set, but semantic analysis/CFG must never run on a module with bad AST nodes")
	}
	if d.pkgNode == nil {
		t.Error("pkgNode = nil, want a package built despite the parser diagnostic")
	}
	if d.symbolResolutionUsable {
		t.Error("the malformed body's resolver diagnostics must make symbol resolution unusable")
	}
	if got := d.publishableDiagnostics(); len(got) != d.parserDiagnosticCount {
		t.Errorf("publishableDiagnostics() = %d diagnostics, want exactly the %d parser diagnostics", len(got), d.parserDiagnosticCount)
	}
}

// TestModuleDriver_MalformedFunctionBodyWithSiblingRetainsSibling is the
// sibling-bearing variant of the test above: symbol resolution is unusable
// for the same reason, but the valid sibling function
// must still appear in pkgNode -- nodebuilder no longer truncates package
// assembly at the first bad node it can't classify (see
// nodebuilder/mod.go's BLangBadTopLevelNode case), and this driver no longer
// drops the sibling's compilation unit outright the way the removed
// filtering used to.
func TestModuleDriver_MalformedFunctionBodyWithSiblingRetainsSibling(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, malformedBodySource+"\npublic function sibling() {}\n")

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	d.advanceTo(stageSemanticAnalyzed, module, newModuleResolutionInput("", nil, nil))

	if d.currentStage() != stageLocalTypeResolved {
		t.Fatalf("currentStage() = %v, want stageLocalTypeResolved for malformed body with sibling", d.currentStage())
	}
	if d.pkgNode == nil {
		t.Fatal("pkgNode = nil, want a package built despite the parser diagnostic")
	}
	if d.symbolResolutionUsable {
		t.Error("the malformed body's resolver diagnostics must make symbol resolution unusable")
	}
	var names []string
	for _, fn := range d.pkgNode.Functions {
		names = append(names, fn.Name.GetValue())
	}
	found := false
	for _, name := range names {
		if name == "sibling" {
			found = true
		}
	}
	if !found {
		t.Errorf("pkgNode.Functions = %v, want it to include the valid sibling function", names)
	}
}

func TestModuleDriver_IdempotentOnAlreadyCompletedStage(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, oneSemanticErrorSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env, projSvc.OpenText, nil)
	input := newModuleResolutionInput("", nil, nil)
	d.advanceTo(stageSemanticAnalyzed, module, input)

	first := len(d.diagnosticContext().Diagnostics())
	if first != 1 {
		t.Fatalf("diagnostics after first advance = %d, want 1", first)
	}

	// Re-requesting a stage already reached must not re-run it.
	d.advanceTo(stageSemanticAnalyzed, module, input)
	second := len(d.diagnosticContext().Diagnostics())
	if second != first {
		t.Fatalf("diagnostics after re-requesting completed stage = %d, want unchanged %d (re-invoked a completed stage)", second, first)
	}
	if got := d.currentStage(); got != stageSemanticAnalyzed {
		t.Fatalf("currentStage() = %v, want stageSemanticAnalyzed", got)
	}
}

// TestModuleDriver_SecondGenerationOverSameModuleDoesNotPanic simulates an
// edit to an already-open file (didChange on main.bal, valid -> erroring)
// and drives two moduleDriver generations over the same shared
// CompilerEnvironment/DiagnosticEnv. The second generation re-parses the
// edited content without panicking: ensureParsed skips RegisterFile for
// already-registered names (alreadyRegistered/markRegistered, multimodule.go)
// rather than re-registering under the same name, since RegisterFile panics on
// a same-name/different-content collision by design and nothing downstream
// reads DiagnosticEnv's stored content back. The edit direction is
// deliberately valid -> erroring (not the reverse) so the assertions cannot
// pass vacuously: a gen2 that silently no-ops (e.g. because it read a stale,
// empty module handle) would report zero diagnostics and an unreached stage,
// both of which are asserted against here.
func TestModuleDriver_SecondGenerationOverSameModuleDoesNotPanic(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, validSource)

	proj1, err := projSvc.Project(u)
	if err != nil || proj1 == nil {
		t.Fatalf("Project (gen1): %v", err)
	}
	module1, env1 := defaultModuleFor(t, proj1)

	gen1 := newModuleDriver(env1, projSvc.OpenText, nil)
	input := newModuleResolutionInput("", nil, nil)
	gen1.advanceTo(stageSemanticAnalyzed, module1, input)
	if got := gen1.currentStage(); got != stageSemanticAnalyzed {
		t.Fatalf("gen1 currentStage() = %v, want stageSemanticAnalyzed", got)
	}
	if got := len(gen1.diagnosticContext().Diagnostics()); got != 0 {
		t.Fatalf("gen1 diagnostics = %d, want 0 (valid source)", got)
	}
	idx1 := env1.DiagnosticEnv().FileIndex(u.Path())

	updateDoc(t, projSvc, "file:///workspace/main.bal", oneSemanticErrorSource, 2)

	proj2, err := projSvc.Project(u)
	if err != nil || proj2 == nil {
		t.Fatalf("Project (gen2): %v", err)
	}
	module2, env2 := defaultModuleFor(t, proj2)

	if env1 != env2 {
		t.Fatal("gen2's CompilerEnvironment is not the same pointer as gen1's — the modifier chain is expected to reuse it; this test would no longer exercise ticket 39's RegisterFile collision")
	}

	var gen2 *moduleDriver
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("gen2.advanceTo panicked (ticket 39 regression): %v", r)
			}
		}()
		gen2 = newModuleDriver(env2, projSvc.OpenText, nil)
		gen2.advanceTo(stageSemanticAnalyzed, module2, input)
	}()

	if got := gen2.currentStage(); got != stageSemanticAnalyzed {
		t.Fatalf("gen2 currentStage() = %v, want stageSemanticAnalyzed (a no-op gen2 would leave this at stageUnstarted)", got)
	}
	if got := len(gen2.diagnosticContext().Diagnostics()); got != 1 {
		t.Errorf("gen2 diagnostics = %d, want 1 (derived from the new, erroring content)", got)
	}
	if got := len(gen1.diagnosticContext().Diagnostics()); got != 0 {
		t.Errorf("gen1 diagnostics after gen2 ran = %d, want unchanged 0 (no cross-generation leak)", got)
	}

	idx2 := env1.DiagnosticEnv().FileIndex(u.Path())
	if idx2 != idx1 {
		t.Errorf("FileIndex(%q) changed across generations (%d -> %d) — ensureParsed skips re-registration for already-registered names, so the index must stay stable", u.Path(), idx1, idx2)
	}
}
