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

	d := newModuleDriver(env)
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

func TestModuleDriver_IdempotentOnAlreadyCompletedStage(t *testing.T) {
	projSvc, _ := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projSvc, u, oneSemanticErrorSource)

	proj, err := projSvc.Project(u)
	if err != nil || proj == nil {
		t.Fatalf("Project: %v", err)
	}
	module, env := defaultModuleFor(t, proj)

	d := newModuleDriver(env)
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

	gen1 := newModuleDriver(env1)
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
		gen2 = newModuleDriver(env2)
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
