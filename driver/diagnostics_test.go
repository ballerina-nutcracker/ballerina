// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func TestComparePackageUsesSemanticVersionOrder(t *testing.T) {
	versions := []struct{ lower, higher string }{
		{"1.9", "1.10"},
		{"1.9", "1.9.1"},
		{"1.9.0", "1.10.0"},
		{"1.0.0-alpha", "1.0.0-alpha.1"},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta"},
		{"1.0.0-rc.1", "1.0.0"},
	}
	for _, test := range versions {
		lower := PackageDescriptor{Org: "a", Name: "b", Version: test.lower}
		higher := PackageDescriptor{Org: "a", Name: "b", Version: test.higher}
		if comparePackage(lower, higher) >= 0 {
			t.Errorf("expected %s before %s", test.lower, test.higher)
		}
	}
}

func TestPublicParseUsesDistinctConcurrentDocumentContextsAndSumsDurations(t *testing.T) {
	descriptor := PackageDescriptor{Org: "test", Name: "parse", Version: "1.0.0"}
	module := ModuleDescriptor{Package: descriptor, Name: descriptor.Name}
	sources := &PackageSources{Descriptor: descriptor, Modules: []*ModuleSources{{
		ID: module, Documents: []string{"a.bal", "b.bal"}, TestDocuments: []string{"c.bal"},
	}}}
	cx := NewContext(context.Background(), NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)))
	var mu sync.Mutex
	var documentContexts []*compilercontext.CompilerContext
	cx.newContextHook = func(created *compilercontext.CompilerContext) {
		mu.Lock()
		documentContexts = append(documentContexts, created)
		mu.Unlock()
	}
	fsys := fstest.MapFS{
		"a.bal": &fstest.MapFile{Data: []byte("function a() {}")},
		"b.bal": &fstest.MapFile{Data: []byte("function b() {}")},
		"c.bal": &fstest.MapFile{Data: []byte("function c() {}")},
	}
	if _, err := Parse(cx, fsys, ".", sources, parser.DebugOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(documentContexts) != 3 {
		t.Fatalf("compiler contexts = %d, want one per source/test document", len(documentContexts))
	}
	seen := make(map[*compilercontext.CompilerContext]bool)
	want := time.Duration(0)
	for _, document := range documentContexts {
		if seen[document] {
			t.Fatal("two documents shared a compiler context")
		}
		seen[document] = true
		stats := document.GetModuleStats()
		if stats == nil || len(stats.Stages) != 1 || stats.Stages[0].Name != compilercontext.StageParse {
			t.Fatalf("document stats = %#v, want one parse timing", stats)
		}
		want += stats.Stages[0].Duration
	}
	stats := cx.ModuleStats()
	if len(stats) != 1 || len(stats[0].Stages) != 1 || stats[0].Stages[0].Duration != want {
		t.Fatalf("aggregated stats = %#v, want exact parse sum %v", stats, want)
	}
}

func TestDocumentParseDurationsSumAcrossDistinctContexts(t *testing.T) {
	cx := NewContext(context.Background(), NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)))
	module := ModuleDescriptor{Package: PackageDescriptor{Org: "test", Name: "stats", Version: "1.0.0"}, Name: "stats"}
	documents := []*compilercontext.CompilerContext{cx.newCompilerContext(module), cx.newCompilerContext(module)}
	if documents[0] == documents[1] {
		t.Fatal("documents shared a compiler context")
	}
	want := time.Duration(0)
	for index, document := range documents {
		document.StartStage(compilercontext.StageParse)
		time.Sleep(time.Millisecond)
		document.EndStage()
		want += document.GetModuleStats().Stages[0].Duration
		cx.drainModule(document, module, compilercontext.StageParse, false, index)
	}
	stats := cx.ModuleStats()
	if len(stats) != 1 || len(stats[0].Stages) != 1 {
		t.Fatalf("ModuleStats = %#v, want one parse timing", stats)
	}
	if got := stats[0].Stages[0].Duration; got != want {
		t.Fatalf("StageParse duration = %v, want exact document sum %v", got, want)
	}
}

func TestDiagnosticsUseThreeScopesAndModuleFiltering(t *testing.T) {
	cx := NewContext(context.Background(), NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)))
	rootPackage := PackageDescriptor{Org: "root", Name: "app", Version: "1.0.0"}
	dependencyPackage := PackageDescriptor{Org: "acme", Name: "dep", Version: "2.0.0"}
	rootModule := ModuleDescriptor{Package: rootPackage, Name: "app"}
	otherModule := ModuleDescriptor{Package: rootPackage, Name: "app.other"}
	cx.setDiscoveryModules([]ModuleDescriptor{rootModule, otherModule})

	root := testDiagnostic("ROOT", diagnostics.Warning)
	dependency := testDiagnostic("PACKAGE", diagnostics.Error)
	module := testDiagnostic("MODULE", diagnostics.Error)
	other := testDiagnostic("OTHER", diagnostics.Warning)
	cx.addModuleDiagnostics(otherModule, compilercontext.StageSemanticAnalysis, false, 0, []diagnostics.Diagnostic{other})
	cx.addPackageDiagnostics(dependencyPackage, []diagnostics.Diagnostic{dependency})
	cx.addRootDiagnostics([]diagnostics.Diagnostic{root})
	cx.addModuleDiagnostics(rootModule, compilercontext.StageParse, false, 0, []diagnostics.Diagnostic{module})

	got := cx.Diagnostics()
	want := []diagnostics.Diagnostic{root, dependency, module, other}
	if len(got) != len(want) {
		t.Fatalf("Diagnostics length = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index].DiagnosticInfo().Code() != want[index].DiagnosticInfo().Code() {
			t.Fatalf("Diagnostics[%d] = %s, want %s", index, got[index].DiagnosticInfo().Code(), want[index].DiagnosticInfo().Code())
		}
	}
	moduleDiagnostics := cx.ModuleDiagnostics(rootModule)
	if len(moduleDiagnostics) != 1 || moduleDiagnostics[0].DiagnosticInfo().Code() != "MODULE" {
		t.Fatalf("ModuleDiagnostics(root) = %v, want only MODULE", moduleDiagnostics)
	}
	if !cx.ModuleHasErrors(rootModule) {
		t.Fatal("ModuleHasErrors(root) = false, want true")
	}
	if cx.ModuleHasErrors(otherModule) {
		t.Fatal("ModuleHasErrors(other) included a warning or a package diagnostic")
	}
}

func testDiagnostic(code string, severity diagnostics.DiagnosticSeverity) diagnostics.Diagnostic {
	return diagnostics.NewDefaultDiagnostic(
		diagnostics.NewDiagnosticInfo(&code, code, severity), diagnostics.Location{}, nil,
	)
}
