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
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type checkpointCancelContext struct {
	checks    atomic.Int64
	threshold atomic.Int64
}

func newCheckpointCancelContext() *checkpointCancelContext {
	ctx := &checkpointCancelContext{}
	ctx.threshold.Store(math.MaxInt64)
	return ctx
}

func (*checkpointCancelContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*checkpointCancelContext) Done() <-chan struct{}       { return nil }
func (*checkpointCancelContext) Value(any) any               { return nil }

func (c *checkpointCancelContext) Err() error {
	if c.checks.Add(1) >= c.threshold.Load() {
		return context.Canceled
	}
	return nil
}

func (c *checkpointCancelContext) cancelAfterChecks(count int64) {
	c.threshold.Store(c.checks.Load() + count)
}

func TestCancellationAtEveryFrontendWrapperCheckpoint(t *testing.T) {
	for _, target := range []string{"ast", "public", "private", "semantic", "cfg-create", "cfg-analyze", "desugar", "bir"} {
		t.Run(target, func(t *testing.T) {
			testFrontendCancellationCheckpoint(t, target)
		})
	}
}

func testFrontendCancellationCheckpoint(t *testing.T, target string) {
	t.Helper()
	dir := packageFixture(t, map[string]string{"main.bal": "public function main() {}"})
	resolver := projects.NewDriverDependencyResolver(os.DirFS(dir), projects.ProjectLoadConfig{})
	standard := newCheckpointCancelContext()
	cx := driver.NewContext(standard, driver.NewEnv(resolver.CompilerEnvironment()))
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
	astModules := make([]*driver.ASTModule, len(parsed.Modules))
	if target == "ast" {
		standard.cancelAfterChecks(3)
		if driver.ToAST(cx, parsed.Modules[len(parsed.Modules)-1]) != nil {
			t.Fatal("AST conversion succeeded after cancellation between conversion and package assembly")
		}
		assertCanceled(t, cx)
		return
	}
	for index, module := range parsed.Modules {
		astModules[index] = driver.ToAST(cx, module)
		if astModules[index] == nil {
			t.Fatalf("AST conversion failed before target %s: %v", target, cx.Diagnostics())
		}
	}
	publicModules := make([]*driver.PartiallyResolvedModule, len(astModules))
	for index, module := range astModules {
		if target == "public" {
			standard.cancelAfterChecks(2)
			if driver.ResolvePublicNodes(cx, module) != nil {
				t.Fatal("public resolution succeeded after cancellation between wrapped operations")
			}
			assertCanceled(t, cx)
			return
		}
		publicModules[index] = driver.ResolvePublicNodes(cx, module)
		if publicModules[index] == nil {
			t.Fatalf("public resolution failed before target %s: %v", target, cx.Diagnostics())
		}
	}
	root := publicModules[len(publicModules)-1]
	if target == "private" {
		standard.cancelAfterChecks(2)
		if driver.ResolvePrivateNodes(cx, root) != nil {
			t.Fatal("private resolution succeeded after cancellation at its post-operation checkpoint")
		}
		assertCanceled(t, cx)
		return
	}
	resolved := driver.ResolvePrivateNodes(cx, root)
	if resolved == nil {
		t.Fatalf("private resolution failed before target %s: %v", target, cx.Diagnostics())
	}
	if target == "semantic" {
		standard.cancelAfterChecks(2)
		if driver.AnalyzeSemantics(cx, resolved) != nil {
			t.Fatal("semantic analysis succeeded after cancellation at its post-operation checkpoint")
		}
		assertCanceled(t, cx)
		return
	}
	analyzed := driver.AnalyzeSemantics(cx, resolved)
	if analyzed == nil {
		t.Fatalf("semantic analysis failed before target %s: %v", target, cx.Diagnostics())
	}
	if target == "cfg-create" {
		standard.cancelAfterChecks(2)
		if driver.CreateControlFlowGraph(cx, analyzed) != nil {
			t.Fatal("CFG creation succeeded after cancellation at its post-operation checkpoint")
		}
		assertCanceled(t, cx)
		return
	}
	flow := driver.CreateControlFlowGraph(cx, analyzed)
	if flow == nil {
		t.Fatalf("CFG creation failed before target %s: %v", target, cx.Diagnostics())
	}
	if target == "cfg-analyze" {
		standard.cancelAfterChecks(2)
		if driver.AnalyzeControlFlowGraph(cx, flow) != nil {
			t.Fatal("CFG analysis succeeded after cancellation at its post-operation checkpoint")
		}
		assertCanceled(t, cx)
		return
	}
	flowAnalyzed := driver.AnalyzeControlFlowGraph(cx, flow)
	if flowAnalyzed == nil {
		t.Fatalf("CFG analysis failed before target %s: %v", target, cx.Diagnostics())
	}
	if target == "desugar" {
		standard.cancelAfterChecks(2)
		if driver.Desugar(cx, flowAnalyzed) != nil {
			t.Fatal("desugaring succeeded after cancellation at its post-operation checkpoint")
		}
		assertCanceled(t, cx)
		return
	}
	desugared := driver.Desugar(cx, flowAnalyzed)
	if desugared == nil {
		t.Fatalf("desugaring failed before target %s: %v", target, cx.Diagnostics())
	}
	standard.cancelAfterChecks(2)
	if driver.GenerateBIR(cx, desugared) != nil {
		t.Fatal("BIR generation succeeded after cancellation at its post-operation checkpoint")
	}
	assertCanceled(t, cx)
}

func assertCanceled(t *testing.T, cx *driver.Context) {
	t.Helper()
	if !errors.Is(cx.Err(), context.Canceled) {
		t.Fatalf("Context.Err() = %v, want context.Canceled", cx.Err())
	}
}

type blockingReadFS struct {
	fstest.MapFS
	started chan struct{}
	release chan struct{}
	active  atomic.Int64
}

func (f *blockingReadFS) ReadFile(name string) ([]byte, error) {
	f.active.Add(1)
	f.started <- struct{}{}
	<-f.release
	f.active.Add(-1)
	return fs.ReadFile(f.MapFS, name)
}

func TestCancellationDuringConcurrentParsingJoinsReaders(t *testing.T) {
	standard, cancel := context.WithCancel(context.Background())
	fsys := &blockingReadFS{
		MapFS: fstest.MapFS{
			"a.bal": &fstest.MapFile{Data: []byte("function a() {}")},
			"b.bal": &fstest.MapFile{Data: []byte("function b() {}")},
		},
		started: make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	descriptor := driver.PackageDescriptor{Org: "testorg", Name: "root", Version: "1.0.0"}
	sources := &driver.PackageSources{Descriptor: descriptor, Modules: []*driver.ModuleSources{{
		ID: driver.ModuleDescriptor{Package: descriptor, Name: descriptor.Name}, Documents: []string{"a.bal", "b.bal"},
	}}}
	cx := driver.NewContext(standard, driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)))
	resultCh := make(chan error, 1)
	go func() {
		_, err := driver.Parse(cx, fsys, ".", sources, parser.DebugOptions{})
		resultCh <- err
	}()
	<-fsys.started
	<-fsys.started
	cancel()
	close(fsys.release)
	if err := <-resultCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse error = %v, want context.Canceled", err)
	}
	if got := fsys.active.Load(); got != 0 {
		t.Fatalf("active readers after Parse returned = %d, want 0", got)
	}
}

type blockingDependencyResolver struct {
	started  chan struct{}
	finished chan struct{}
	once     sync.Once
}

func (r *blockingDependencyResolver) Resolve(ctx context.Context, _ driver.DependencyRequest) (
	fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error,
) {
	r.once.Do(func() { close(r.started) })
	<-ctx.Done()
	close(r.finished)
	return nil, "", nil, nil, ctx.Err()
}

func TestCancellationDuringResolverExecutionReturnsAfterResolver(t *testing.T) {
	parsed, _ := parseRoot(t, "", driver.Manifest{})
	standard, cancel := context.WithCancel(context.Background())
	cx := driver.NewContext(standard, driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)))
	rootSources := packageSources(parsed.Root, []string{"main.bal"})
	parsed, err := driver.Parse(cx, fstest.MapFS{"main.bal": &fstest.MapFile{}}, ".", rootSources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &blockingDependencyResolver{started: make(chan struct{}), finished: make(chan struct{})}
	resultCh := make(chan error, 1)
	go func() {
		_, err := driver.ResolveDependencies(cx, parsed, resolver)
		resultCh <- err
	}()
	<-resolver.started
	cancel()
	if err := <-resultCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveDependencies error = %v, want context.Canceled", err)
	}
	select {
	case <-resolver.finished:
	default:
		t.Fatal("ResolveDependencies returned before the resolver call finished")
	}
}

func TestCancellationAtFinalDependencyCheckpoint(t *testing.T) {
	measure := newCheckpointCancelContext()
	parsed, cx := parsedRootWithContext(t, measure)
	resolver := emptyPackageResolver()
	if _, err := driver.ResolveDependencies(cx, parsed, resolver); err != nil {
		t.Fatal(err)
	}
	checks := measure.checks.Load()

	standard := newCheckpointCancelContext()
	parsed, cx = parsedRootWithContext(t, standard)
	standard.threshold.Store(checks)
	if _, err := driver.ResolveDependencies(cx, parsed, emptyPackageResolver()); !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveDependencies error = %v, want context.Canceled at final checkpoint", err)
	}
}

func parsedRootWithContext(t *testing.T, standard context.Context) (*driver.ParsedPackage, *driver.Context) {
	t.Helper()
	descriptor := driver.PackageDescriptor{Org: "testorg", Name: "root", Version: "1.0.0"}
	sources := packageSources(descriptor, []string{"main.bal"})
	cx := driver.NewContext(standard, driver.NewEnv(compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)))
	parsed, err := driver.Parse(cx, fstest.MapFS{"main.bal": &fstest.MapFile{}}, ".", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return parsed, cx
}

func emptyPackageResolver() *fakeResolver {
	return &fakeResolver{resolve: func(request driver.DependencyRequest) (fs.FS, string, *driver.PackageSources, []diagnostics.Diagnostic, error) {
		return fstest.MapFS{}, ".", packageSources(request.Descriptor, nil), nil, nil
	}}
}
