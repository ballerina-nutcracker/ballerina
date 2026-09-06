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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestFindSourcesSortsModulesAndDocuments(t *testing.T) {
	dir := packageFixture(t, map[string]string{
		"z.bal":                      "function z() {}",
		"a.bal":                      "function a() {}",
		"modules/zeta/z.bal":         "function z() {}",
		"modules/alpha/main.bal":     "function a() {}",
		"modules/alpha/nested/x.bal": "function ignored() {}",
	})
	cx := newDriverContext()
	sources, err := driver.FindSources(cx, os.DirFS(dir), ".", filepath.Base(dir))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(sources.Modules), 3; got != want {
		t.Fatalf("module count = %d, want %d", got, want)
	}
	if sources.Modules[0].ID.Name != sources.Descriptor.Name || sources.Modules[1].ID.Name != sources.Descriptor.Name+".alpha" {
		t.Fatalf("unexpected module order: %v, %v", sources.Modules[0].ID, sources.Modules[1].ID)
	}
	if got := sources.Modules[0].Documents; len(got) != 2 || got[0] != "a.bal" || got[1] != "z.bal" {
		t.Fatalf("unexpected document order: %v", got)
	}
}

func TestParseKeepsResultsAfterSyntaxErrors(t *testing.T) {
	dir := packageFixture(t, map[string]string{"main.bal": "public function main( {"})
	cx := newDriverContext()
	sources, err := driver.FindSources(cx, os.DirFS(dir), ".", filepath.Base(dir))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := driver.Parse(cx, os.DirFS(dir), ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed == nil || len(parsed.Modules) != 1 || len(parsed.Modules[0].Documents) != 1 {
		t.Fatalf("parse result unavailable after syntax error: %#v", parsed)
	}
	if !cx.HasErrors() {
		t.Fatal("expected syntax diagnostics")
	}
}

func TestCanceledContextStopsDiscovery(t *testing.T) {
	dir := packageFixture(t, map[string]string{"main.bal": "public function main() {}"})
	standard, cancel := context.WithCancel(context.Background())
	cancel()
	env := driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false))
	cx := driver.NewContext(standard, env)
	_, err := driver.FindSources(cx, os.DirFS(dir), ".", filepath.Base(dir))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestEnvironmentBindsToOneContext(t *testing.T) {
	env := driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false))
	driver.NewContext(context.Background(), env)
	defer func() {
		if recover() == nil {
			t.Fatal("expected a second context binding to panic")
		}
	}()
	driver.NewContext(context.Background(), env)
}

type cancelingReadDirFS struct {
	fs.FS
	cancel context.CancelFunc
	once   sync.Once
	mu     sync.Mutex
	calls  []string
}

func (c *cancelingReadDirFS) Open(name string) (fs.File, error) {
	c.record("open:" + name)
	return c.FS.Open(name)
}

func (c *cancelingReadDirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	c.record("readdir:" + name)
	entries, err := fs.ReadDir(c.FS, name)
	if name == "." {
		c.once.Do(c.cancel)
	}
	return entries, err
}

func (c *cancelingReadDirFS) record(call string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, call)
}

func (c *cancelingReadDirFS) recordedCalls() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.calls...)
}

func TestCancellationDuringDiscoveryReturnsContextError(t *testing.T) {
	dir := packageFixture(t, map[string]string{"main.bal": "public function main() {}"})
	standard, cancel := context.WithCancel(context.Background())
	cx := driver.NewContext(standard, driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)))
	fsys := &cancelingReadDirFS{FS: os.DirFS(dir), cancel: cancel}
	_, err := driver.FindSources(cx, fsys, ".", filepath.Base(dir))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	calls := fsys.recordedCalls()
	if len(calls) == 0 || calls[len(calls)-1] != "readdir:." {
		t.Fatalf("filesystem operations continued after cancellation: %v", calls)
	}
}

func TestPublicResolutionClearsBoundImportsAndScopesAliasesPerFile(t *testing.T) {
	dir := packageFixture(t, map[string]string{
		"main.bal": "import ballerina/io; public function main() { io:println(\"main\"); }",
		"util.bal": "import ballerina/io; function util() { io:println(\"util\"); }",
	})
	parsed, cx := resolveFixture(t, dir)
	for _, module := range parsed.Modules {
		astModule := driver.ToAST(cx, module)
		if astModule == nil {
			t.Fatalf("AST conversion failed: %v", cx.Diagnostics())
		}
		public := driver.ResolvePublicNodes(cx, astModule)
		if public == nil {
			t.Fatalf("public resolution failed for compilation-unit-local aliases: %v", cx.Diagnostics())
		}
		if public.PackageNode.Imports != nil {
			t.Fatalf("bound package imports were retained for %v", public.ID)
		}
	}
}

func TestDuplicateAliasInOneCompilationUnitIsRejected(t *testing.T) {
	dir := packageFixture(t, map[string]string{
		"main.bal": "import ballerina/io; import ballerina/io; public function main() { io:println(\"main\"); }",
	})
	parsed, cx := resolveFixture(t, dir)
	for _, module := range parsed.Modules {
		astModule := driver.ToAST(cx, module)
		if astModule == nil {
			t.Fatalf("AST conversion failed: %v", cx.Diagnostics())
		}
		public := driver.ResolvePublicNodes(cx, astModule)
		if module.ID.Package == parsed.Root && public != nil {
			t.Fatal("duplicate alias in one compilation unit was accepted")
		}
		if module.ID.Package == parsed.Root && moduleCompletedStage(cx, module.ID.Name, compilercontext.StageLocalNodeResolution) {
			t.Fatal("root entered private resolution after a public-resolution error")
		}
	}
	if !cx.HasErrors() {
		t.Fatal("expected duplicate alias diagnostic")
	}
}

func TestPrivateResolutionErrorStopsBeforeSemanticAnalysis(t *testing.T) {
	dir := packageFixture(t, map[string]string{
		"main.bal": "public function main() { break; }",
	})
	parsed, cx := resolveFixture(t, dir)
	root := resolvePublicRoot(t, parsed, cx)
	if resolved := driver.ResolvePrivateNodes(cx, root); resolved != nil {
		t.Fatal("private type error did not stop private resolution")
	}
	if moduleCompletedStage(cx, root.ID.Name, compilercontext.StageSemanticAnalysis) {
		t.Fatal("module entered semantic analysis after private-resolution error")
	}
}

func TestSemanticErrorStopsBeforeCFGCreation(t *testing.T) {
	dir := packageFixture(t, map[string]string{
		"main.bal": "public function main() { int value = \"bad\"; _ = value; }",
	})
	parsed, cx := resolveFixture(t, dir)
	root := resolvePublicRoot(t, parsed, cx)
	resolved := driver.ResolvePrivateNodes(cx, root)
	if resolved == nil {
		t.Fatalf("private resolution failed too early: %v", cx.Diagnostics())
	}
	if analyzed := driver.AnalyzeSemantics(cx, resolved); analyzed != nil {
		t.Fatal("semantic error did not stop semantic analysis")
	}
	if moduleCompletedStage(cx, root.ID.Name, compilercontext.StageCFGCreation) {
		t.Fatal("module entered CFG creation after semantic error")
	}
}

func TestCFGErrorStopsBeforeDesugaring(t *testing.T) {
	dir := packageFixture(t, map[string]string{
		"main.bal": "function value() returns int {} public function main() {}",
	})
	parsed, cx := resolveFixture(t, dir)
	root := resolvePublicRoot(t, parsed, cx)
	resolved := driver.ResolvePrivateNodes(cx, root)
	if resolved == nil {
		t.Fatalf("private resolution failed too early: %v", cx.Diagnostics())
	}
	analyzed := driver.AnalyzeSemantics(cx, resolved)
	if analyzed == nil {
		t.Fatalf("semantic analysis failed too early: %v", cx.Diagnostics())
	}
	flow := driver.CreateControlFlowGraph(cx, analyzed)
	if flow == nil {
		t.Fatalf("CFG creation failed too early: %v", cx.Diagnostics())
	}
	if cfgAnalyzed := driver.AnalyzeControlFlowGraph(cx, flow); cfgAnalyzed != nil {
		t.Fatal("missing explicit return did not stop CFG analysis")
	}
	if moduleCompletedStage(cx, root.ID.Name, compilercontext.StageDesugaring) {
		t.Fatal("module entered desugaring after CFG error")
	}
}

func resolvePublicRoot(t *testing.T, parsed *driver.ParsedPackage, cx *driver.Context) *driver.PartiallyResolvedModule {
	t.Helper()
	var root *driver.PartiallyResolvedModule
	for _, module := range parsed.Modules {
		astModule := driver.ToAST(cx, module)
		if astModule == nil {
			t.Fatalf("AST conversion failed: %v", cx.Diagnostics())
		}
		public := driver.ResolvePublicNodes(cx, astModule)
		if public == nil {
			t.Fatalf("public resolution failed: %v", cx.Diagnostics())
		}
		if module.ID.Package == parsed.Root {
			root = public
		}
	}
	if root == nil {
		t.Fatal("root public module not found")
	}
	return root
}

func resolveFixture(t *testing.T, dir string) (*driver.ParsedPackage, *driver.Context) {
	t.Helper()
	buildOptions := projects.NewBuildOptionsBuilder().WithStats(true).Build()
	resolver := projects.NewDriverDependencyResolver(os.DirFS(dir), projects.ProjectLoadConfig{BuildOptions: &buildOptions})
	cx := driver.NewContext(context.Background(), driver.NewEnv(resolver.CompilerEnvironment()))
	sources, err := driver.FindSources(cx, os.DirFS(dir), ".", filepath.Base(dir))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := driver.Parse(cx, os.DirFS(dir), ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err = driver.ResolveDependencies(cx, parsed, resolver)
	if err != nil {
		t.Fatal(err)
	}
	return parsed, cx
}

func TestFullPipelineProducesDependencyFirstBIR(t *testing.T) {
	dir := packageFixture(t, map[string]string{"main.bal": "public function main() {}"})
	resolver := projects.NewDriverDependencyResolver(os.DirFS(dir), projects.ProjectLoadConfig{})
	cx := driver.NewContext(context.Background(), driver.NewEnv(resolver.CompilerEnvironment()))
	sources, err := driver.FindSources(cx, os.DirFS(dir), ".", filepath.Base(dir))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := driver.Parse(cx, os.DirFS(dir), ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err = driver.ResolveDependencies(cx, parsed, resolver)
	if err != nil {
		t.Fatal(err)
	}
	asts := make([]*driver.ASTModule, len(parsed.Modules))
	var wg sync.WaitGroup
	for index, module := range parsed.Modules {
		wg.Add(1)
		go func() { defer wg.Done(); asts[index] = driver.ToAST(cx, module) }()
	}
	wg.Wait()
	packages := make([]string, len(asts))
	for index, module := range asts {
		public := driver.ResolvePublicNodes(cx, module)
		resolved := driver.ResolvePrivateNodes(cx, public)
		analyzed := driver.AnalyzeSemantics(cx, resolved)
		flow := driver.CreateControlFlowGraph(cx, analyzed)
		flowAnalyzed := driver.AnalyzeControlFlowGraph(cx, flow)
		desugared := driver.Desugar(cx, flowAnalyzed)
		pkg := driver.GenerateBIR(cx, desugared)
		if pkg == nil {
			t.Fatalf("nil BIR at index %d; diagnostics: %v", index, cx.Diagnostics())
		}
		packages[index] = pkg.PackageID.PkgName.Value()
	}
	if packages[len(packages)-1] != sources.Descriptor.Name {
		t.Fatalf("last package = %q, want root %q", packages[len(packages)-1], sources.Descriptor.Name)
	}
}

func TestConcurrentAnonymousASTIdentitiesAreStableAcrossRuns(t *testing.T) {
	dir := packageFixture(t, map[string]string{
		"main.bal":                "public function main() { function (int) returns int f = function (int x) returns int { return x; }; record {| int value; |} r = {value: 1}; }",
		"modules/other/other.bal": "public function other() { function (string) returns string f = function (string x) returns string { return x; }; record {| string value; |} r = {value: \"x\"}; }",
	})
	var baseline string
	for run := range 10 {
		cx := newDriverContext()
		sources, err := driver.FindSources(cx, os.DirFS(dir), ".", filepath.Base(dir))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := driver.Parse(cx, os.DirFS(dir), ".", sources, parser.DebugOptions{})
		if err != nil {
			t.Fatal(err)
		}
		asts := make([]*driver.ASTModule, len(parsed.Modules))
		var wg sync.WaitGroup
		for index, module := range parsed.Modules {
			wg.Add(1)
			go func() {
				defer wg.Done()
				asts[index] = driver.ToAST(cx, module)
			}()
		}
		wg.Wait()
		outputs := make([]string, len(asts))
		for index, module := range asts {
			if module == nil {
				t.Fatalf("run %d AST %d failed: %v", run, index, cx.Diagnostics())
			}
			for _, document := range module.Documents {
				packageID := document.CompilationUnit.GetPackageID()
				if packageID == nil || packageID.OrgName.Value() != module.ID.Package.Org || packageID.PkgName.Value() != module.ID.Name {
					t.Fatalf("run %d module %s has wrong interned package ID: %v", run, module.ID.Name, packageID)
				}
			}
			outputs[index] = (&ast.PrettyPrinter{}).Print(module.PackageNode)
		}
		got := strings.Join(outputs, "\n---module---\n")
		if run == 0 {
			baseline = got
		} else if got != baseline {
			t.Fatalf("anonymous AST identities changed on run %d", run)
		}
	}
}

func moduleCompletedStage(cx *driver.Context, moduleName string, stage compilercontext.CompilationStage) bool {
	for _, module := range cx.ModuleStats() {
		if module.ModuleName != moduleName {
			continue
		}
		for _, timing := range module.Stages {
			if timing.Name == stage {
				return true
			}
		}
	}
	return false
}

func newDriverContext() *driver.Context {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	return driver.NewContext(context.Background(), driver.NewEnv(env))
}

func packageFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["Ballerina.toml"] = "[package]\norg = \"testorg\"\nname = \"fixture\"\nversion = \"0.1.0\"\n"
	for name, content := range files {
		fullPath := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
