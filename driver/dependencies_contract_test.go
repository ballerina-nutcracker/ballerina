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

package driver_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type resolverCall struct {
	request driver.DependencyRequest
}

type fakeResolver struct {
	mu      sync.Mutex
	calls   []resolverCall
	resolve func(driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error)
}

func (r *fakeResolver) Resolve(_ context.Context, request driver.DependencyRequest) (
	fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error,
) {
	r.mu.Lock()
	r.calls = append(r.calls, resolverCall{request: request})
	r.mu.Unlock()
	return r.resolve(request)
}

func (r *fakeResolver) requests() []driver.DependencyRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]driver.DependencyRequest, len(r.calls))
	for i, call := range r.calls {
		result[i] = call.request
	}
	return result
}

func TestImplicitPackageRequestsAreExactAndOrdered(t *testing.T) {
	resolution := driver.ResolutionOptions{Offline: true, Sticky: true}
	parsed, cx := parseRoot(t, "", driver.Manifest{Resolution: resolution})
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"lang.__internal", "lang.int", "lang.boolean", "lang.decimal", "lang.error", "lang.float",
		"lang.string", "lang.value", "lang.xml", "lang.array", "lang.map", "lang.object", "lang.runtime",
	}
	requests := resolver.requests()
	if len(requests) != len(want) {
		t.Fatalf("resolver calls = %d, want %d: %#v", len(requests), len(want), requests)
	}
	for i, request := range requests {
		if request.Descriptor != (driver.PackageDescriptor{Org: "ballerina", Name: want[i], Version: "0.0.1"}) {
			t.Fatalf("request %d = %+v, want ballerina/%s:0.0.1", i, request.Descriptor, want[i])
		}
		if request.Resolution != resolution {
			t.Fatalf("request %d resolution = %+v, want %+v", i, request.Resolution, resolution)
		}
	}
}

func TestImplicitRootExclusionAndExplicitDeduplication(t *testing.T) {
	root := driver.PackageDescriptor{Org: "ballerina", Name: "lang.int", Version: "0.0.1"}
	sources := packageSources(root, []string{"main.bal"})
	cx := newDriverContext()
	parsed, err := driver.Parse(cx, fstest.MapFS{"main.bal": &fstest.MapFile{Data: []byte("import ballerina/lang.'int;")}}, ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	requests := resolver.requests()
	if len(requests) != 12 {
		t.Fatalf("resolver calls = %d, want 12 after root exclusion: %#v", len(requests), requests)
	}
	for _, request := range requests {
		if request.Descriptor == root {
			t.Fatal("root implicit package was resolved as a dependency")
		}
	}
}

func TestImplicitVersionConflictIsOperationalError(t *testing.T) {
	manifest := driver.Manifest{Dependencies: []driver.Dependency{{Org: "ballerina", Name: "lang.int", Version: "2.0.0"}}}
	parsed, cx := parseRoot(t, "import ballerina/lang.'int;", manifest)
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err == nil {
		t.Fatal("expected conflicting implicit version to return an operational error")
	}
}

func TestUnusedImplicitManifestVersionConflictIsOperationalError(t *testing.T) {
	manifest := driver.Manifest{Dependencies: []driver.Dependency{{Org: "ballerina", Name: "lang.int", Version: "2.0.0"}}}
	parsed, cx := parseRoot(t, "", manifest)
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return fstest.MapFS{}, ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err == nil {
		t.Fatal("expected unused conflicting implicit manifest version to return an operational error")
	}
	if calls := resolver.requests(); len(calls) != 0 {
		t.Fatalf("resolver was called before root implicit conflict validation: %#v", calls)
	}
}

func TestCancellationPrecedesInvalidPublicInputs(t *testing.T) {
	standard, cancel := context.WithCancel(context.Background())
	cancel()
	env := driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false))
	cx := driver.NewContext(standard, env)
	if _, err := driver.Parse(cx, nil, "../invalid", nil, parser.DebugOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse error = %v, want context.Canceled", err)
	}

	standard, cancel = context.WithCancel(context.Background())
	cancel()
	env = driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false))
	cx = driver.NewContext(standard, env)
	if _, err := driver.ResolveDependencies(cx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveDependencies error = %v, want context.Canceled", err)
	}
}

func TestResolverResponseValidation(t *testing.T) {
	code := "DISCOVERY"
	discoveryDiagnostic := diagnostics.NewDefaultDiagnostic(
		diagnostics.NewDiagnosticInfo(&code, "discovery", diagnostics.Warning), diagnostics.Location{}, nil,
	)
	tests := []struct {
		name     string
		response func(driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error)
	}{
		{name: "partial filesystem", response: func(driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
			return fstest.MapFS{}, "", nil, nil, nil
		}},
		{name: "diagnostics without package", response: func(driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
			return nil, "", nil, []diagnostics.Diagnostic{discoveryDiagnostic}, nil
		}},
		{name: "descriptor mismatch", response: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
			descriptor := request.Descriptor
			descriptor.Org = "wrong"
			return fstest.MapFS{}, ".", packageSources(descriptor, nil), nil, nil
		}},
		{name: "version mismatch", response: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
			descriptor := request.Descriptor
			descriptor.Version = "9.0.0"
			return fstest.MapFS{}, ".", packageSources(descriptor, nil), nil, nil
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, cx := parseRoot(t, "", driver.Manifest{})
			resolver := &fakeResolver{resolve: test.response}
			if _, err := driver.ResolveDependencies(cx, parsed, resolver); err == nil {
				t.Fatal("expected resolver contract error")
			}
		})
	}
}

func TestImportDrivenResolverResponseMustContainRequestedModule(t *testing.T) {
	parsed, cx := parseRoot(t, "import acme/foo.bar;", driver.Manifest{})
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		if request.Descriptor.Org == "ballerina" {
			return fstest.MapFS{}, ".", packageSources(request.Descriptor, nil), nil, nil
		}
		if request.Descriptor.Name == "foo.bar" {
			return nil, "", nil, nil, nil
		}
		descriptor := driver.PackageDescriptor{Org: request.Descriptor.Org, Name: request.Descriptor.Name, Version: "1.0.0"}
		return fstest.MapFS{}, ".", packageSources(descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err == nil {
		t.Fatal("expected resolver response without requested module to be rejected")
	}
}

func TestTransitiveImportsUseRootRepositoryOverride(t *testing.T) {
	dependencyFS := os.DirFS("testdata/source-form-dependency")
	dependencySources, err := driver.FindSources(newDriverContext(), dependencyFS, ".", "dep")
	if err != nil {
		t.Fatal(err)
	}
	rootManifest := driver.Manifest{Dependencies: []driver.Dependency{
		{Org: "acme", Name: "dep", Version: "1.0.0", Repository: "root-dep"},
		{Org: "acme", Name: "util", Version: "9.0.0", Repository: "root-util"},
	}, Resolution: driver.ResolutionOptions{Offline: true, Sticky: true}}
	parsed, cx := parseRoot(t, "import acme/dep.foo;", rootManifest)
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		if request.Descriptor.Org == "ballerina" {
			return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
		}
		switch request.Descriptor.Name {
		case "dep":
			return dependencyFS, ".", dependencySources, nil, nil
		case "util", "testutil", "leaf":
			descriptor := request.Descriptor
			if descriptor.Version == "" {
				descriptor.Version = "1.0.0"
			}
			return emptyResolvedPackage(descriptor, driver.Manifest{}), ".", packageSources(descriptor, nil), nil, nil
		default:
			return nil, "", nil, nil, nil
		}
	}}
	resolved, err := driver.ResolveDependencies(cx, parsed, resolver)
	if err != nil {
		t.Fatal(err)
	}
	var util []driver.DependencyRequest
	for _, request := range resolver.requests() {
		if request.Descriptor.Org == "acme" && request.Descriptor.Name == "util" {
			util = append(util, request)
		}
	}
	if len(util) != 1 {
		t.Fatalf("util requests = %#v, want one", util)
	}
	if got := util[0]; got.Descriptor.Version != "2.0.0" || got.Repository != "root-util" || !got.Resolution.Offline || !got.Resolution.Sticky {
		t.Fatalf("transitive util request = %+v, want dependency version, root repository override, and root resolution policy", got)
	}
	var ordinaryNames []string
	for _, request := range resolver.requests() {
		if request.Descriptor.Org == "acme" {
			ordinaryNames = append(ordinaryNames, request.Descriptor.Name)
		}
	}
	wantNames := []string{"dep.foo", "dep", "util", "testutil", "leaf"}
	if !reflect.DeepEqual(ordinaryNames, wantNames) {
		t.Fatalf("source-form dependency request order = %v, want %v", ordinaryNames, wantNames)
	}
	rootIndex, depIndex := -1, -1
	for index, module := range resolved.Modules {
		if module.ID.Name == "fixture" {
			rootIndex = index
		}
		if module.ID.Name == "dep.foo" {
			depIndex = index
		}
	}
	if depIndex < 0 || rootIndex < 0 || depIndex >= rootIndex {
		t.Fatalf("dependency-first module order = %v", moduleNames(resolved.Modules))
	}
}

func TestUnusedRootManifestDependencyDoesNotResolve(t *testing.T) {
	manifest := driver.Manifest{Dependencies: []driver.Dependency{{Org: "acme", Name: "unused", Version: "1.0.0"}}}
	parsed, cx := parseRoot(t, "", manifest)
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	for _, request := range resolver.requests() {
		if request.Descriptor.Org == "acme" {
			t.Fatalf("unused root dependency was resolved: %+v", request)
		}
	}
}

func TestVersionlessImportUsesFullCandidateThenDefaultOrgFallback(t *testing.T) {
	parsed, cx := parseRoot(t, "import foo.bar;", driver.Manifest{})
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		if request.Descriptor.Org == "ballerina" {
			return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
		}
		if request.Descriptor.Name == "foo.bar" {
			return nil, "", nil, nil, nil
		}
		descriptor := driver.PackageDescriptor{Org: "testorg", Name: "foo", Version: "1.2.3"}
		return fstest.MapFS{}, ".", &driver.PackageSources{Descriptor: descriptor, Modules: []*driver.ModuleSources{
			{ID: driver.ModuleDescriptor{Package: descriptor, Name: "foo"}},
			{ID: driver.ModuleDescriptor{Package: descriptor, Name: "foo.bar"}},
		}}, nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	var ordinary []driver.DependencyRequest
	for _, request := range resolver.requests() {
		if request.Descriptor.Org != "ballerina" {
			ordinary = append(ordinary, request)
		}
	}
	if len(ordinary) != 2 || ordinary[0].Descriptor.Org != "testorg" || ordinary[0].Descriptor.Name != "foo.bar" ||
		ordinary[1].Descriptor.Org != "testorg" || ordinary[1].Descriptor.Name != "foo" {
		t.Fatalf("candidate requests = %+v, want testorg/foo.bar then testorg/foo", ordinary)
	}
}

func TestFirstSelectedOrdinaryVersionWinsLaterTransitiveConflict(t *testing.T) {
	manifest := driver.Manifest{Dependencies: []driver.Dependency{
		{Org: "acme", Name: "util", Version: "9.0.0"},
		{Org: "acme", Name: "dep", Version: "1.0.0"},
	}, Resolution: driver.ResolutionOptions{Offline: true, Sticky: true}}
	parsed, cx := parseRoot(t, "import acme/util;\nimport acme/dep;", manifest)
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		if request.Descriptor.Org == "ballerina" {
			return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
		}
		if request.Descriptor.Name == "dep" {
			depManifest := driver.Manifest{Dependencies: []driver.Dependency{{Org: "acme", Name: "util", Version: "2.0.0"}}}
			sources := packageSources(request.Descriptor, nil)
			sources.Manifest = depManifest
			return emptyResolvedPackage(request.Descriptor, depManifest), ".", sources, nil, nil
		}
		return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	var util []driver.DependencyRequest
	for _, request := range resolver.requests() {
		if request.Descriptor.Org == "acme" && request.Descriptor.Name == "util" {
			util = append(util, request)
		}
	}
	if len(util) != 1 || util[0].Descriptor.Version != "9.0.0" || !util[0].Resolution.Offline || !util[0].Resolution.Sticky {
		t.Fatalf("util requests = %+v, want one sticky/offline selection at 9.0.0", util)
	}
}

func TestNonBallerinaLangPackageDoesNotConflictWithImplicitPackage(t *testing.T) {
	manifest := driver.Manifest{Dependencies: []driver.Dependency{{Org: "acme", Name: "lang.int", Version: "2.0.0"}}}
	parsed, cx := parseRoot(t, "import acme/lang.'int;", manifest)
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatalf("ordinary acme/lang.int was treated as implicit: %v", err)
	}
	found := false
	for _, request := range resolver.requests() {
		if request.Descriptor == (driver.PackageDescriptor{Org: "acme", Name: "lang.int", Version: "2.0.0"}) {
			found = true
		}
	}
	if !found {
		t.Fatal("acme/lang.int:2.0.0 was not resolved")
	}
}

func moduleNames(modules []*driver.ParsedModule) []string {
	result := make([]string, len(modules))
	for index, module := range modules {
		result[index] = module.ID.Name
	}
	return result
}

func TestSameModuleImportProducesCycleDiagnostic(t *testing.T) {
	parsed, cx := parseRoot(t, "import testorg/fixture;", driver.Manifest{})
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return emptyResolvedPackage(request.Descriptor, driver.Manifest{}), ".", packageSources(request.Descriptor, nil), nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	if !cx.HasErrors() {
		t.Fatal("expected dependency cycle diagnostic")
	}
	for _, diagnostic := range cx.Diagnostics() {
		if diagnostic.DiagnosticInfo().Code() == "UNRESOLVED_IMPORT" {
			t.Fatalf("same-module import was reported as unresolved: %v", diagnostic)
		}
		if diagnostic.DiagnosticInfo().Code() == "DEPENDENCY_RESOLUTION" {
			return
		}
	}
	t.Fatal("missing dependency cycle diagnostic")
}

func TestResolverManifestErrorStopsBeforeDependencyParseAndStaysOutOfModuleScope(t *testing.T) {
	manifest := driver.Manifest{Dependencies: []driver.Dependency{{Org: "acme", Name: "dep", Version: "1.0.0"}}}
	parsed, cx := parseRoot(t, "import acme/dep;", manifest)
	code := "DEPENDENCY_MANIFEST"
	manifestError := diagnostics.NewDefaultDiagnostic(
		diagnostics.NewDiagnosticInfo(&code, "dependency manifest error", diagnostics.Error), diagnostics.Location{}, nil,
	)
	dependencyReads := 0
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		if request.Descriptor.Org == "ballerina" {
			return fstest.MapFS{}, ".", packageSources(request.Descriptor, nil), nil, nil
		}
		dependencyFS := &readCountingMapFS{MapFS: fstest.MapFS{"dep.bal": &fstest.MapFile{Data: []byte("public function dep() {}")}}, reads: &dependencyReads}
		return dependencyFS, ".", packageSources(request.Descriptor, []string{"dep.bal"}), []diagnostics.Diagnostic{manifestError}, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	if !cx.HasErrors() {
		t.Fatal("resolver manifest error was not aggregated")
	}
	if dependencyReads != 0 {
		t.Fatalf("dependency source reads = %d, want 0 after resolver manifest error", dependencyReads)
	}
	if got := cx.ModuleDiagnostics(parsed.Modules[0].ID); len(got) != 0 {
		t.Fatalf("ModuleDiagnostics included package-scoped resolver error: %v", got)
	}
	ordinaryCalls := 0
	for _, request := range resolver.requests() {
		if request.Descriptor.Org == "acme" {
			ordinaryCalls++
		}
	}
	if ordinaryCalls != 1 {
		t.Fatalf("ordinary resolver calls = %d, want immediate stop after first error", ordinaryCalls)
	}
}

type readCountingMapFS struct {
	fstest.MapFS
	reads *int
}

func (f *readCountingMapFS) ReadFile(name string) ([]byte, error) {
	*f.reads++
	return fs.ReadFile(f.MapFS, name)
}

func TestUnresolvedImportIsSourceLocatedModuleDiagnostic(t *testing.T) {
	parsed, cx := parseRoot(t, "import acme/missing.foo;", driver.Manifest{})
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		if request.Descriptor.Org == "ballerina" {
			return fstest.MapFS{}, ".", packageSources(request.Descriptor, nil), nil, nil
		}
		return nil, "", nil, nil, nil
	}}
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	moduleDiagnostics := cx.ModuleDiagnostics(parsed.Modules[0].ID)
	if len(moduleDiagnostics) != 1 || moduleDiagnostics[0].DiagnosticInfo().Code() != "UNRESOLVED_IMPORT" ||
		cx.DiagnosticEnv().FileName(moduleDiagnostics[0].Location()) != "main.bal" {
		t.Fatalf("module diagnostics = %v, want one source-located unresolved import", moduleDiagnostics)
	}
}

func TestConcurrentParseDiagnosticsRemainModuleScopedAndOrdered(t *testing.T) {
	descriptor := driver.PackageDescriptor{Org: "testorg", Name: "diagnostics", Version: "1.0.0"}
	first := driver.ModuleDescriptor{Package: descriptor, Name: descriptor.Name}
	second := driver.ModuleDescriptor{Package: descriptor, Name: descriptor.Name + ".second"}
	sources := &driver.PackageSources{Descriptor: descriptor, Modules: []*driver.ModuleSources{
		{ID: first, Documents: []string{"first.bal"}},
		{ID: second, Documents: []string{"second.bal"}},
	}}
	cx := newDriverContext()
	_, err := driver.Parse(cx, fstest.MapFS{
		"first.bal":  &fstest.MapFile{Data: []byte("function first( {}")},
		"second.bal": &fstest.MapFile{Data: []byte("function second( {}")},
	}, ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cx.ModuleDiagnostics(first)) == 0 || len(cx.ModuleDiagnostics(second)) == 0 {
		t.Fatalf("module diagnostics missing: first=%v second=%v", cx.ModuleDiagnostics(first), cx.ModuleDiagnostics(second))
	}
	global := cx.Diagnostics()
	if len(global) < 2 || cx.DiagnosticEnv().FileName(global[0].Location()) != "first.bal" {
		t.Fatalf("concurrent diagnostic order = %v, want first module before second", global)
	}
}

func TestParseInputValidationMatrix(t *testing.T) {
	validDescriptor := driver.PackageDescriptor{Org: "testorg", Name: "fixture", Version: "1.0.0"}
	validModule := &driver.ModuleSources{ID: driver.ModuleDescriptor{Package: validDescriptor, Name: "fixture"}}
	tests := []struct {
		name    string
		root    string
		sources *driver.PackageSources
	}{
		{name: "nil sources", root: ".", sources: nil},
		{name: "invalid root", root: "../root", sources: packageSources(validDescriptor, nil)},
		{name: "missing descriptor", root: ".", sources: &driver.PackageSources{}},
		{name: "invalid version", root: ".", sources: packageSources(driver.PackageDescriptor{Org: "o", Name: "n", Version: "x"}, nil)},
		{name: "nil module", root: ".", sources: &driver.PackageSources{Descriptor: validDescriptor, Modules: []*driver.ModuleSources{nil}}},
		{name: "missing default", root: ".", sources: &driver.PackageSources{Descriptor: validDescriptor}},
		{name: "mismatched package", root: ".", sources: &driver.PackageSources{
			Descriptor: validDescriptor,
			Modules:    []*driver.ModuleSources{{ID: driver.ModuleDescriptor{Name: "fixture"}}},
		}},
		{name: "duplicate module", root: ".", sources: &driver.PackageSources{
			Descriptor: validDescriptor,
			Modules:    []*driver.ModuleSources{validModule, validModule},
		}},
		{name: "duplicate document", root: ".", sources: &driver.PackageSources{
			Descriptor: validDescriptor,
			Modules: []*driver.ModuleSources{{
				ID: validModule.ID, Documents: []string{"a.bal"}, TestDocuments: []string{"a.bal"},
			}},
		}},
		{name: "absolute document", root: ".", sources: packageSources(validDescriptor, []string{"/a.bal"})},
		{name: "escaping document", root: ".", sources: packageSources(validDescriptor, []string{"../a.bal"})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := driver.Parse(newDriverContext(), fstest.MapFS{}, test.root, test.sources, parser.DebugOptions{})
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestReadableSourcePathsAndOpaqueSyntaxTreeIdentitiesDoNotCollide(t *testing.T) {
	root := driver.PackageDescriptor{Org: "testorg", Name: "paths", Version: "1.0.0"}
	rootSource := "src/a|b/same.bal"
	rootTest := "tests/a:b/same.bal"
	dependencySource := "nested/a|b/same.bal"
	sources := &driver.PackageSources{Descriptor: root, Modules: []*driver.ModuleSources{
		{ID: driver.ModuleDescriptor{Package: root, Name: root.Name}, Documents: []string{"main.bal"}},
		{ID: driver.ModuleDescriptor{Package: root, Name: root.Name + ".other"}, Documents: []string{rootSource}, TestDocuments: []string{rootTest}},
	}}
	fsys := fstest.MapFS{
		"main.bal": &fstest.MapFile{Data: []byte("import acme/dep;")},
		rootSource: &fstest.MapFile{Data: []byte("function rootSourceMarker( {}")},
		rootTest:   &fstest.MapFile{Data: []byte("function rootTestMarker( {}")},
	}
	cx := newDriverContext()
	parsed, err := driver.Parse(cx, fsys, ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		if request.Descriptor.Org == "ballerina" {
			return fstest.MapFS{}, ".", packageSources(request.Descriptor, nil), nil, nil
		}
		descriptor := driver.PackageDescriptor{Org: "acme", Name: "dep", Version: "1.0.0"}
		dependencySources := packageSources(descriptor, []string{dependencySource})
		return fstest.MapFS{dependencySource: &fstest.MapFile{Data: []byte("function dependencyMarker( {}")}}, ".", dependencySources, nil, nil
	}}
	resolved, err := driver.ResolveDependencies(cx, parsed, resolver)
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := map[string]bool{"main.bal": true, rootSource: true, rootTest: true, dependencySource: true}
	identities := make(map[string]bool)
	for _, module := range resolved.Modules {
		for _, document := range append(append([]*driver.ParsedDocument(nil), module.Documents...), module.TestDocuments...) {
			if !wantPaths[document.SourcePath] {
				continue
			}
			identity := document.SyntaxTree.FilePath()
			if !strings.HasPrefix(identity, "R") && !strings.HasPrefix(identity, "D") {
				t.Fatalf("SyntaxTree.FilePath() = %q, want opaque diagnostic identity", identity)
			}
			if identities[identity] {
				t.Fatalf("duplicate syntax-tree diagnostic identity %q", identity)
			}
			identities[identity] = true
			if strings.HasPrefix(document.SourcePath, "R") || strings.HasPrefix(document.SourcePath, "D") {
				t.Fatalf("ParsedDocument exposed opaque identity %q", document.SourcePath)
			}
			delete(wantPaths, document.SourcePath)
		}
		if module.ID.Name == root.Name+".other" || module.ID.Name == "dep" {
			recovered := driver.ToRecoveredAST(cx, module)
			if recovered == nil {
				t.Fatalf("recovered AST conversion failed for %s: %v", module.ID.Name, cx.ModuleDiagnostics(module.ID))
			}
			want := rootSource
			if module.ID.Name == "dep" {
				want = dependencySource
			}
			if len(recovered.Documents) != 1 || recovered.Documents[0].SourcePath != want {
				t.Fatalf("ASTDocument paths for %s = %v, want %q", module.ID.Name, recovered.Documents, want)
			}
		}
	}
	if len(wantPaths) != 0 {
		t.Fatalf("missing parsed readable paths: %v", wantPaths)
	}
	seen := make(map[string]string)
	for _, diagnostic := range cx.Diagnostics() {
		location := diagnostic.Location()
		name := cx.DiagnosticEnv().FileName(location)
		if name == rootSource || name == rootTest || name == dependencySource {
			seen[name] = cx.DiagnosticEnv().TextDocument(location).String()
		}
	}
	for path, marker := range map[string]string{rootSource: "rootSourceMarker", rootTest: "rootTestMarker", dependencySource: "dependencyMarker"} {
		if !strings.Contains(seen[path], marker) {
			t.Fatalf("diagnostic document for %q = %q, want marker %q", path, seen[path], marker)
		}
	}
}

func parseRoot(t *testing.T, source string, manifest driver.Manifest) (*driver.ParsedPackage, *driver.Context) {
	t.Helper()
	descriptor := driver.PackageDescriptor{Org: "testorg", Name: "fixture", Version: "1.0.0"}
	sources := packageSources(descriptor, []string{"main.bal"})
	sources.Manifest = manifest
	cx := newDriverContext()
	parsed, err := driver.Parse(cx, fstest.MapFS{"main.bal": &fstest.MapFile{Data: []byte(source)}}, ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return parsed, cx
}

func packageSources(descriptor driver.PackageDescriptor, documents []string) *driver.PackageSources {
	return &driver.PackageSources{Descriptor: descriptor, Modules: []*driver.ModuleSources{{
		ID: driver.ModuleDescriptor{Package: descriptor, Name: descriptor.Name}, Documents: documents,
	}}}
}

func emptyResolvedPackage(descriptor driver.PackageDescriptor, manifest driver.Manifest) fs.FS {
	if descriptor.Version == "" {
		panic(fmt.Sprintf("test resolver received versionless descriptor: %+v", descriptor))
	}
	return fstest.MapFS{}
}

func TestDriverResultsDoNotExposeFrontendOrSourceOwners(t *testing.T) {
	for _, value := range []any{driver.PackageSources{}, driver.ParsedPackage{}} {
		assertNoDriverFieldType(t, reflect.TypeOf(value), map[string]bool{"io/fs.FS": true})
	}
	for _, value := range []any{
		driver.Context{}, driver.ASTModule{}, driver.RecoveredASTModule{}, driver.PartiallyResolvedModule{}, driver.ResolvedModule{},
		driver.SemanticallyAnalyzedModule{}, driver.ControlFlowModule{}, driver.AnalyzedModule{}, driver.DesugaredModule{},
	} {
		assertNoDriverFieldType(t, reflect.TypeOf(value), map[string]bool{
			"github.com/ballerina-nutcracker/ballerina/st.SyntaxTree":           true,
			"github.com/ballerina-nutcracker/ballerina/driver.ParsedDocument":   true,
			"github.com/ballerina-nutcracker/ballerina/context.CompilerContext": true,
		})
	}
}

func assertNoDriverFieldType(t *testing.T, root reflect.Type, forbidden map[string]bool) {
	t.Helper()
	visited := make(map[reflect.Type]bool)
	var inspect func(reflect.Type)
	inspect = func(current reflect.Type) {
		for current.Kind() == reflect.Pointer || current.Kind() == reflect.Slice || current.Kind() == reflect.Array {
			current = current.Elem()
		}
		key := current.PkgPath() + "." + current.Name()
		if forbidden[key] {
			t.Fatalf("%s retains forbidden type %s", root, key)
		}
		if current.Kind() != reflect.Struct || visited[current] || current.PkgPath() != "github.com/ballerina-nutcracker/ballerina/driver" {
			return
		}
		visited[current] = true
		for i := 0; i < current.NumField(); i++ {
			inspect(current.Field(i).Type)
		}
	}
	inspect(root)
}
