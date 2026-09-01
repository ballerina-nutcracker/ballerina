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
	"sort"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// oldPathExtract replicates the pre-ticket-37 extraction logic (pkg.Compilation()
// + Document.SyntaxTree().FilePath()-keyed resolution) verbatim, as a
// regression oracle: realCompilePackage/extractForURI must produce
// diagnostics equivalent in shape/content to what this produced.
func oldPathExtract(pkg *projects.Package) (byFile, resByFile map[string][]CompilerDiagnostic, resErr bool) {
	comp := pkg.Compilation()
	env := comp.DiagnosticEnv()
	byFile = make(map[string][]CompilerDiagnostic)
	resByFile = make(map[string][]CompilerDiagnostic)
	cache := make(map[string]docInfo)
	resolve := func(fname string) (docInfo, bool) {
		if d, ok := cache[fname]; ok {
			return d, true
		}
		for _, moduleID := range pkg.ModuleIDs() {
			module := pkg.Module(moduleID)
			if module == nil {
				continue
			}
			for _, docID := range module.DocumentIDs() {
				doc := module.Document(docID)
				if doc == nil {
					continue
				}
				if doc.SyntaxTree().FilePath() == fname {
					text := doc.TextDocument().String()
					info := docInfo{text: text, lineStarts: computeLineStarts(text)}
					cache[fname] = info
					return info, true
				}
			}
		}
		cache[fname] = docInfo{}
		return docInfo{}, false
	}
	extract := func(diags []diagnostics.Diagnostic, target map[string][]CompilerDiagnostic) {
		for _, diag := range diags {
			location := diag.Location()
			if !diagnostics.LocationHasSource(location) {
				continue
			}
			fname := env.FileName(location)
			info, ok := resolve(fname)
			if !ok {
				continue
			}
			target[fname] = append(target[fname], convertDiag(info.text, info.lineStarts, diag))
		}
	}
	extract(comp.DiagnosticResult().Diagnostics(), byFile)
	extract(comp.Resolution().DiagnosticResult().Diagnostics(), resByFile)
	resErr = comp.Resolution().DiagnosticResult().HasErrors()
	return byFile, resByFile, resErr
}

// diagShape is the (Code, Message) pair a shape comparison checks, ignoring
// ordering within a file's diagnostic list.
type diagShape struct {
	Code    string
	Message string
}

func shapesFor(diags []CompilerDiagnostic) []diagShape {
	shapes := make([]diagShape, 0, len(diags))
	for _, d := range diags {
		shapes = append(shapes, diagShape{Code: d.Code, Message: d.Message})
	}
	sort.Slice(shapes, func(i, j int) bool {
		if shapes[i].Code != shapes[j].Code {
			return shapes[i].Code < shapes[j].Code
		}
		return shapes[i].Message < shapes[j].Message
	})
	return shapes
}

// assertSameShape asserts got and want have the same file-key set and, per
// file, the same (Code, Message) multiset — a shape/content equivalence
// check, not byte-for-byte, since the two paths reach the same diagnostics
// through different orchestration layers.
func assertSameShape(t *testing.T, label string, got, want map[string][]CompilerDiagnostic) {
	t.Helper()
	for fname := range want {
		if _, ok := got[fname]; !ok {
			t.Errorf("%s: missing file key %q present in old-path result (got keys: %v)", label, fname, keysOf(got))
		}
	}
	for fname := range got {
		if _, ok := want[fname]; !ok {
			t.Errorf("%s: unexpected file key %q not present in old-path result (want keys: %v)", label, fname, keysOf(want))
		}
	}
	for fname, wantDiags := range want {
		gotDiags, ok := got[fname]
		if !ok {
			continue // already reported above
		}
		gotShapes, wantShapes := shapesFor(gotDiags), shapesFor(wantDiags)
		if len(gotShapes) != len(wantShapes) {
			t.Errorf("%s[%s]: %d diagnostics, want %d (got=%v want=%v)", label, fname, len(gotShapes), len(wantShapes), gotShapes, wantShapes)
			continue
		}
		for i := range gotShapes {
			if gotShapes[i] != wantShapes[i] {
				t.Errorf("%s[%s][%d]: %+v, want %+v", label, fname, i, gotShapes[i], wantShapes[i])
			}
		}
	}
}

func keysOf(m map[string][]CompilerDiagnostic) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestRealCompilePackage_MatchesOldPackageCompilationPath is the regression
// check the ADR's Verification section requires: realCompilePackage
// (packageDriver) must produce diagnostics equivalent in shape/content to
// the old pkg.Compilation()-based path, for a multi-module fixture with a
// real diagnostic in a *named* module (greet, an edge target — the case
// where the registration-key scheme change matters, and where a
// topoModules() double-count would show up as a doubled diagnostic list).
// The old-path and new-path packages are two independent loads of the same
// on-disk fixture+edit (separate ProjectServices/environments), so neither
// side's registration observes the other's.
func TestRealCompilePackage_MatchesOldPackageCompilationPath(t *testing.T) {
	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	// Body-only edit: an AnalyzeSemantics-stage error inside greet.bal (an
	// edge target — consumer imports it), the case the ADR's Verification
	// section calls out ("a real diagnostic in a named module").
	editedGreet := "public function greeting() returns string {\n    int x = \"hello\";\n    _ = x;\n    return \"hello\";\n}\n"

	newSvc := newProjectOnlyService(t)
	_, mainURI := openMultimoduleFixture(t, newSvc)
	newGreetURI := fileURI(t, "file://"+greetPath)
	applyOpen(t, newSvc, newGreetURI, editedGreet)
	newProj, err := newSvc.Project(mainURI)
	if err != nil || newProj == nil {
		t.Fatalf("new-path Project: %v", err)
	}
	newPkg := newProj.CurrentPackage()

	oldSvc := newProjectOnlyService(t)
	_, oldMainURI := openMultimoduleFixture(t, oldSvc)
	oldGreetURI := fileURI(t, "file://"+greetPath)
	applyOpen(t, oldSvc, oldGreetURI, editedGreet)
	oldProj, err := oldSvc.Project(oldMainURI)
	if err != nil || oldProj == nil {
		t.Fatalf("old-path Project: %v", err)
	}
	oldPkg := oldProj.CurrentPackage()

	newResult := realCompilePackage(newPkg)
	wantByFile, wantResByFile, wantResErr := oldPathExtract(oldPkg)

	assertSameShape(t, "byFile", newResult.byFile, wantByFile)
	assertSameShape(t, "resByFile", newResult.resByFile, wantResByFile)
	if newResult.resolutionErrored != wantResErr {
		t.Errorf("resolutionErrored = %v, want %v", newResult.resolutionErrored, wantResErr)
	}

	// The regression check must not pass vacuously: greet's own file must
	// carry exactly one diagnostic on both sides, and that count must not be
	// doubled by a topoModules() duplicate (see the packageDriver test
	// suite's duplicate-node investigation).
	if got := len(newResult.byFile[greetPath]); got != 1 {
		t.Fatalf("byFile[%s] = %d diagnostics, want exactly 1 (not doubled, not vacuously empty)", greetPath, got)
	}
	if got := len(wantByFile[greetPath]); got != 1 {
		t.Fatalf("old-path byFile[%s] = %d diagnostics, want exactly 1 (test fixture assumption broken)", greetPath, got)
	}
}

// TestExtractForURI_MatchesOldPackageCompilationPath is the extractForURI
// half of the same regression check: Compile()'s inline fallback must
// produce the same diagnostics for greet.bal as the old
// pkg.Compilation()-based extractForURI did.
func TestExtractForURI_MatchesOldPackageCompilationPath(t *testing.T) {
	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	editedGreet := "public function greeting() returns string {\n    int x = \"hello\";\n    _ = x;\n    return \"hello\";\n}\n"

	newSvc := newProjectOnlyService(t)
	_, mainURI := openMultimoduleFixture(t, newSvc)
	newGreetURI := fileURI(t, "file://"+greetPath)
	applyOpen(t, newSvc, newGreetURI, editedGreet)
	newProj, err := newSvc.Project(mainURI)
	if err != nil || newProj == nil {
		t.Fatalf("new-path Project: %v", err)
	}
	newPkg := newProj.CurrentPackage()
	newDiags := extractForURI(newProj, newPkg, greetPath)

	oldSvc := newProjectOnlyService(t)
	_, oldMainURI := openMultimoduleFixture(t, oldSvc)
	oldGreetURI := fileURI(t, "file://"+greetPath)
	applyOpen(t, oldSvc, oldGreetURI, editedGreet)
	oldProj, err := oldSvc.Project(oldMainURI)
	if err != nil || oldProj == nil {
		t.Fatalf("old-path Project: %v", err)
	}
	oldPkg := oldProj.CurrentPackage()
	wantDiags := oldExtractForURI(oldProj, oldPkg, greetPath)

	if len(newDiags) != 1 {
		t.Fatalf("extractForURI diagnostics = %d, want exactly 1", len(newDiags))
	}
	if len(wantDiags) != 1 {
		t.Fatalf("old-path extractForURI diagnostics = %d, want exactly 1 (test fixture assumption broken)", len(wantDiags))
	}
	if newDiags[0].Code != wantDiags[0].Code || newDiags[0].Message != wantDiags[0].Message {
		t.Errorf("diagnostic = %+v, want %+v", newDiags[0], wantDiags[0])
	}
}

// oldExtractForURI replicates the pre-ticket-37 extractForURI (pkg.Compilation()
// based) verbatim, as a regression oracle.
func oldExtractForURI(project projects.Project, pkg *projects.Package, fileName string) []CompilerDiagnostic {
	docID, ok := project.DocumentID(fileName)
	if !ok {
		return nil
	}
	module := pkg.Module(docID.ModuleID())
	if module == nil {
		return nil
	}
	doc := module.Document(docID)
	if doc == nil {
		return nil
	}
	text := doc.TextDocument().String()
	compilation := pkg.Compilation()
	env := compilation.DiagnosticEnv()
	lineStarts := computeLineStarts(text)
	var diags []CompilerDiagnostic
	for _, diag := range compilation.DiagnosticResult().Diagnostics() {
		location := diag.Location()
		if !diagnostics.LocationHasSource(location) {
			continue
		}
		if env.FileName(location) != fileName {
			continue
		}
		diags = append(diags, convertDiag(text, lineStarts, diag))
	}
	return diags
}
