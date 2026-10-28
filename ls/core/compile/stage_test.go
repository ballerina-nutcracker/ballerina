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

const validSource = "public function main() {}\n"

const duplicateTopLevelSource = "public function duplicate() {}\npublic function duplicate() {}\n"
const recursivePublicTypeSource = "public type Invalid Invalid;\n"
const unusedTopLevelSource = "const int UNUSED = 1;\n"
const malformedBodySource = "function broken() {\n    int value = ;\n}\n"
const malformedSignatureAndBodySource = "function broken(int {\n    int value = ;\n}\n"
const malformedTopLevelSource = "function broken(int {\n}\n"

// breakOutsideLoopSource reports an error during local type resolution.
const breakOutsideLoopSource = "function main() {\n    break;\n}\n"

// oneSemanticErrorSource reports one type mismatch during semantic analysis.
const oneSemanticErrorSource = "public function main() {\n    int x = \"hello\";\n    _ = x;\n}\n"

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
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_UnresolvedReferenceReachesLocalTypeResolved(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_UnusedSymbolDoesNotBlockTypeResolution(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_MalformedFunctionSignatureRunsSymbolResolutionButUnusable(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_MalformedTopLevelReachesTopLevelTypeResolved(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_MalformedTopLevelReachesLocalTypeResolved(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_LocalTypeErrorStopsBeforeSemanticAnalysis(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_MalformedFunctionBodyReachesLocalTypeResolved(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

func TestModuleDriver_MalformedFunctionBodyWithSiblingRetainsSibling(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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

	d.advanceTo(stageSemanticAnalyzed, module, input)
	second := len(d.diagnosticContext().Diagnostics())
	if second != first {
		t.Fatalf("diagnostics after re-requesting completed stage = %d, want unchanged %d (re-invoked a completed stage)", second, first)
	}
	if got := d.currentStage(); got != stageSemanticAnalyzed {
		t.Fatalf("currentStage() = %v, want stageSemanticAnalyzed", got)
	}
}

// Re-parsing edited text retains the file index and isolates each driver's diagnostics.
func TestModuleDriver_SecondGenerationOverSameModuleDoesNotPanic(t *testing.T) {
	t.Parallel()
	projSvc, _ := newTestServices(t)
	u := tempFileURI(t, "main.bal")
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
	name, _ := projSvc.Paths(proj1).NameOf(u)
	idx1 := env1.DiagnosticEnv().FileIndex(name)

	updateDoc(t, projSvc, u.String(), oneSemanticErrorSource, 2)

	proj2, err := projSvc.Project(u)
	if err != nil || proj2 == nil {
		t.Fatalf("Project (gen2): %v", err)
	}
	module2, env2 := defaultModuleFor(t, proj2)

	if env1 != env2 {
		t.Fatal("gen2's CompilerEnvironment is not the same pointer as gen1's — the modifier chain is expected to reuse it; this test would no longer exercise duplicate RegisterFile registration for the edited file")
	}

	var gen2 *moduleDriver
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("gen2.advanceTo panicked while recompiling a registered file: %v", r)
			}
		}()
		gen2 = newModuleDriver(env2, projSvc.OpenText, nil)
		gen2.advanceTo(stageSemanticAnalyzed, module2, input)
	}()

	if got := gen2.currentStage(); got != stageSemanticAnalyzed {
		t.Fatalf("gen2 currentStage() = %v, want stageSemanticAnalyzed (a no-op gen2 would leave this at stageUnstarted)", got)
	}
	gen2Diags := gen2.diagnosticContext().Diagnostics()
	if len(gen2Diags) != 1 {
		t.Fatalf("gen2 diagnostics = %d, want 1 (derived from the new, erroring content)", len(gen2Diags))
	}
	location := gen2Diags[0].Location()
	diagnosticEnv := env2.DiagnosticEnv()
	if line, column := diagnosticEnv.StartLine(location), diagnosticEnv.StartColumn(location); line != 1 || column != 12 {
		t.Errorf("gen2 diagnostic position = (%d, %d), want (1, 12)", line, column)
	}
	if got := len(gen1.diagnosticContext().Diagnostics()); got != 0 {
		t.Errorf("gen1 diagnostics after gen2 ran = %d, want unchanged 0 (no cross-generation leak)", got)
	}

	idx2 := env1.DiagnosticEnv().FileIndex(name)
	if idx2 != idx1 {
		t.Errorf("FileIndex(%q) changed across generations (%d -> %d) — re-registration must keep the index stable", u.Path(), idx1, idx2)
	}
}
