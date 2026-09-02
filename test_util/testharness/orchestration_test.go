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

package testharness

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/bir"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

var orchestratedStages = []compilercontext.CompilationStage{
	compilercontext.StageASTBuild,
	compilercontext.StageSymbolResolution,
	compilercontext.StageLocalNodeResolution,
	compilercontext.StageSemanticAnalysis,
	compilercontext.StageCFGCreation,
	compilercontext.StageCFGAnalysis,
	compilercontext.StageDesugaring,
	compilercontext.StageBIRGeneration,
}

func TestOrchestratorStopsAfterEveryFailedStageAndSkipsInitialization(t *testing.T) {
	dir := harnessProject(t, false)
	sentinel := errors.New("injected stage failure")
	for targetIndex, target := range orchestratedStages {
		t.Run(string(target), func(t *testing.T) {
			var mu sync.Mutex
			var calls []compilercontext.CompilationStage
			hook := func(stage compilercontext.CompilationStage, module driver.ModuleDescriptor) error {
				if module.Package.Org != "testorg" || module.Package.Name != "fixture" {
					return nil
				}
				mu.Lock()
				calls = append(calls, stage)
				mu.Unlock()
				if stage == target {
					return sentinel
				}
				return nil
			}
			compiled, err := compileWithDriver(context.Background(), os.DirFS(dir), ".", filepath.Base(dir), projects.ProjectLoadConfig{}, hook)
			if !errors.Is(err, sentinel) {
				t.Fatalf("error = %v, want injected failure", err)
			}
			if compiled == nil {
				t.Fatal("missing partial compilation result")
			}
			mu.Lock()
			got := append([]compilercontext.CompilationStage(nil), calls...)
			mu.Unlock()
			if !slices.Contains(got, target) {
				t.Fatalf("target stage %s was not invoked: %v", target, got)
			}
			for _, later := range orchestratedStages[targetIndex+1:] {
				if slices.Contains(got, later) {
					t.Fatalf("stage %s ran after %s failed: %v", later, target, got)
				}
			}
			initializations := 0
			if err == nil {
				_ = initializeDriverPackages(compiled.BIRPackages, func(_ *bir.BIRPackage) error {
					initializations++
					return nil
				})
			}
			if initializations != 0 {
				t.Fatalf("runtime initialization count = %d, want 0", initializations)
			}
		})
	}
}

func TestOrchestratorAllowsBIRWhileAnotherModuleIsHeldEarlier(t *testing.T) {
	dir := harnessProject(t, true)
	slowEntered := make(chan struct{})
	rootEnteredBIR := make(chan struct{})
	releaseSlow := make(chan struct{})
	var slowOnce sync.Once
	var rootOnce sync.Once
	hook := func(stage compilercontext.CompilationStage, module driver.ModuleDescriptor) error {
		if module.Package.Org != "testorg" || module.Package.Name != "fixture" {
			return nil
		}
		if module.Name == "fixture.slow" && stage == compilercontext.StageLocalNodeResolution {
			slowOnce.Do(func() { close(slowEntered) })
			<-releaseSlow
		}
		if module.Name == "fixture" && stage == compilercontext.StageBIRGeneration {
			rootOnce.Do(func() { close(rootEnteredBIR) })
		}
		return nil
	}
	type result struct {
		compiled *DriverCompilation
		err      error
	}
	resultCh := make(chan result, 1)
	go func() {
		compiled, err := compileWithDriver(context.Background(), os.DirFS(dir), ".", filepath.Base(dir), projects.ProjectLoadConfig{}, hook)
		resultCh <- result{compiled: compiled, err: err}
	}()
	waitForSignal(t, slowEntered, "slow module entering private resolution")
	waitForSignal(t, rootEnteredBIR, "root module entering BIR while slow module is held")
	close(releaseSlow)
	compiledResult := <-resultCh
	if compiledResult.err != nil {
		t.Fatal(compiledResult.err)
	}
	var initialized []string
	if err := initializeDriverPackages(compiledResult.compiled.BIRPackages, func(pkg *bir.BIRPackage) error {
		initialized = append(initialized, pkg.PackageID.PkgName.Value())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want := make([]string, len(compiledResult.compiled.Parsed.Modules))
	for index, module := range compiledResult.compiled.Parsed.Modules {
		want[index] = module.ID.Name
	}
	if !slices.Equal(initialized, want) {
		t.Fatalf("initialization order = %v, want topological order %v", initialized, want)
	}
}

func TestOrchestratorJoinsCanceledModuleWorkers(t *testing.T) {
	dir := harnessProject(t, true)
	standard, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{}, 2)
	var active atomic.Int64
	hook := func(stage compilercontext.CompilationStage, module driver.ModuleDescriptor) error {
		if module.Package.Org != "testorg" || module.Package.Name != "fixture" || stage != compilercontext.StageLocalNodeResolution {
			return nil
		}
		active.Add(1)
		entered <- struct{}{}
		<-standard.Done()
		active.Add(-1)
		return standard.Err()
	}
	resultCh := make(chan error, 1)
	go func() {
		_, err := compileWithDriver(standard, os.DirFS(dir), ".", filepath.Base(dir), projects.ProjectLoadConfig{}, hook)
		resultCh <- err
	}()
	waitForSignal(t, entered, "first module worker")
	waitForSignal(t, entered, "second module worker")
	cancel()
	if err := <-resultCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if got := active.Load(); got != 0 {
		t.Fatalf("active workers after return = %d, want 0", got)
	}
}

func TestOrchestratorPropagatesCancellationFromDriverWrapper(t *testing.T) {
	dir := harnessProject(t, false)
	standard, cancel := context.WithCancel(context.Background())
	hook := func(stage compilercontext.CompilationStage, module driver.ModuleDescriptor) error {
		if module.Package.Org == "testorg" && module.Package.Name == "fixture" &&
			stage == compilercontext.StageLocalNodeResolution {
			cancel()
		}
		return nil
	}
	compiled, err := compileWithDriver(standard, os.DirFS(dir), ".", filepath.Base(dir), projects.ProjectLoadConfig{}, hook)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if compiled == nil || compiled.Context == nil {
		t.Fatal("missing partial compilation context")
	}
	if len(compiled.BIRPackages) > 0 && compiled.BIRPackages[len(compiled.BIRPackages)-1] != nil {
		t.Fatal("BIR was produced after cancellation")
	}
}

type parseCountingFS struct {
	fs.FS
	balReads atomic.Int64
}

func (f *parseCountingFS) ReadFile(name string) ([]byte, error) {
	if filepath.Ext(name) == ".bal" {
		f.balReads.Add(1)
	}
	return fs.ReadFile(f.FS, name)
}

func TestCompileWithDriverStopsBeforeParseOnManifestError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte("[package]\nversion = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.bal"), []byte("public function main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys := &parseCountingFS{FS: os.DirFS(dir)}
	compiled, err := CompileWithDriver(fsys, ".", filepath.Base(dir), projects.ProjectLoadConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if compiled == nil || compiled.Context == nil || !compiled.Context.HasErrors() {
		t.Fatal("manifest diagnostic was not preserved")
	}
	if compiled.Parsed != nil {
		t.Fatal("Parse ran after discovery produced an error")
	}
	if reads := fsys.balReads.Load(); reads != 0 {
		t.Fatalf(".bal reads = %d, want 0 after manifest error", reads)
	}
}

func harnessProject(t *testing.T, namedModule bool) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"Ballerina.toml": "[package]\norg = \"testorg\"\nname = \"fixture\"\nversion = \"1.0.0\"\n",
		"main.bal":       "public function main() {}",
	}
	if namedModule {
		files["modules/slow/slow.bal"] = "function slow() {}"
	}
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

func waitForSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}
