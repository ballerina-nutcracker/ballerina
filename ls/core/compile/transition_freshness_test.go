// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package compile

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

type transitionDiscoveryErrorFS struct{}

func (transitionDiscoveryErrorFS) Open(string) (fs.File, error) { return nil, fs.ErrPermission }

func TestProjectTransitionRejectsObsoleteCompilation(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"manifest-round-trip", "discovery-recovery"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			const obsolete = "public function main() { int obsolete = 1; }\n"
			const latest = "public function main() { int current = 1; }\n"
			file := filepath.Join(root, "main.bal")
			if err := os.WriteFile(file, []byte(obsolete), 0o644); err != nil {
				t.Fatal(err)
			}
			uri, err := workspace.FileURIFromPath(file)
			if err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(root, "Ballerina.toml")
			manifest, err := workspace.FileURIFromPath(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			var failDiscovery atomic.Bool
			opener := func(dir string) fs.FS {
				if dir == root && failDiscovery.Load() {
					return transitionDiscoveryErrorFS{}
				}
				return os.DirFS(dir)
			}
			bus := event.New()
			service := workspace.New(opener, bus)
			compiler := New(service, bus, WithDebounce(0), WithMaxWorkers(2))
			oldStarted := make(chan projects.Project, 1)
			currentStarted := make(chan *packageGenState, 1)
			releaseOld, releaseCurrent := make(chan struct{}), make(chan struct{})
			var oldOnce, currentOnce sync.Once
			unblockOld := func() { oldOnce.Do(func() { close(releaseOld) }) }
			unblockCurrent := func() { currentOnce.Do(func() { close(releaseCurrent) }) }
			defer func() { unblockOld(); unblockCurrent(); compiler.Shutdown(); bus.Close() }()
			var first atomic.Bool
			compiler.compileFn = func(pkg *projects.Package, cyc compileCycle) cycleResult {
				if first.CompareAndSwap(false, true) {
					oldStarted <- pkg.Project()
					<-releaseOld
				} else if pkg.Project().Kind() == projects.ProjectKindSingleFile {
					currentStarted <- cyc.priorState
					<-releaseCurrent
				}
				return realCompilePackage(pkg, cyc)
			}
			succeeded, cancelled := &genCollector{}, &genCollector{}
			bus.Subscribe([]event.Kind{event.CompilationSucceeded, event.CompilationCancelled}, func(e event.Event) {
				if e.SourceRoot() != file {
					return
				}
				if e.Kind() == event.CompilationSucceeded {
					succeeded.add(e.Generation())
				} else {
					cancelled.add(e.Generation())
				}
			})
			if _, err := service.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeOpen, URI: uri, Text: obsolete, Version: 1}); err != nil {
				t.Fatal(err)
			}
			oldProject := awaitTransitionCompile(t, oldStarted)
			oldGen, ok := service.Generation(file)
			if !ok {
				t.Fatal("old project has no generation")
			}
			if scenario == "manifest-round-trip" {
				manifestText := "[package]\norg=\"test\"\nname=\"transition\"\nversion=\"0.1.0\"\n"
				if err := os.WriteFile(manifestPath, []byte(manifestText), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := service.ApplyWatchedFile(context.Background(), workspace.WatchedFileChange{URI: manifest, Kind: workspace.WatchedFileCreate}); err != nil {
					t.Fatal(err)
				}
				if _, err := service.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeUpdate, URI: uri, Text: latest, Version: 2}); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(manifestPath); err != nil {
					t.Fatal(err)
				}
				if err := service.ApplyWatchedFile(context.Background(), workspace.WatchedFileChange{URI: manifest, Kind: workspace.WatchedFileDelete}); err != nil {
					t.Fatal(err)
				}
			} else {
				failDiscovery.Store(true)
				if err := service.ApplyWatchedFile(context.Background(), workspace.WatchedFileChange{URI: manifest, Kind: workspace.WatchedFileModified}); !errors.Is(err, fs.ErrPermission) {
					t.Fatalf("discovery error=%v", err)
				}
				if _, err := service.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeUpdate, URI: uri, Text: latest, Version: 2}); !errors.Is(err, fs.ErrPermission) {
					t.Fatalf("update error=%v", err)
				}
				failDiscovery.Store(false)
				if _, err := service.Apply(context.Background(), workspace.DocumentChange{Kind: workspace.ChangeUpdate, URI: uri, Text: latest, Version: 3}); err != nil {
					t.Fatal(err)
				}
			}
			currentProject, currentGen, ok := service.CurrentProject(file)
			if !ok || currentProject == oldProject {
				t.Fatal("expected a restored, distinct single-file project")
			}
			if currentGen <= oldGen {
				t.Errorf("restored generation %d reuses obsolete generation %d", currentGen, oldGen)
			}
			unblockOld()
			prior := awaitTransitionCompile(t, currentStarted)
			bus.Flush()
			if stable, ok := compiler.store.Stable(file); ok {
				t.Errorf("obsolete compile published stable generation %d for project %p", stable.Key().Generation, stable.Project())
			}
			if prior != nil && len(prior.modules) != 0 {
				t.Errorf("obsolete compile committed %d carried-forward modules into the restored project", len(prior.modules))
			}
			for _, gen := range succeeded.slice() {
				if gen == oldGen {
					t.Errorf("obsolete generation %d emitted compilation success", gen)
				}
			}
			sawCancelled := false
			for _, gen := range cancelled.slice() {
				sawCancelled = sawCancelled || gen == oldGen
			}
			if !sawCancelled {
				t.Error("obsolete compile did not emit cancellation")
			}
			unblockCurrent()
			compiler.Flush()
			stable, ok := compiler.store.Stable(file)
			if !ok || stable.Project() != currentProject || stable.Key().Generation != currentGen {
				t.Fatalf("wrong restored snapshot: %+v, %v", stable.Key(), ok)
			}
			for _, diag := range stable.Diagnostics() {
				if strings.Contains(diag.Message, "obsolete") {
					t.Errorf("obsolete diagnostic survived: %s", diag.Message)
				}
			}
			diags, gen, ok := compiler.DiagnosticsFor(file)
			if !ok || gen != currentGen || len(diags[uri]) != 1 || !strings.Contains(diags[uri][0].Message, "current") {
				t.Fatalf("current diagnostics=%+v, generation=%d, ok=%v", diags, gen, ok)
			}
			sm, ok := compiler.SealedModuleFor(context.Background(), uri)
			if !ok {
				t.Fatal("restored sealed module unavailable")
			}
			if text, ok := sm.SourceText(uri); !ok || text != latest {
				t.Fatalf("restored semantic source=%q, %v", text, ok)
			}
		})
	}
}

func awaitTransitionCompile[T any](t *testing.T, ready <-chan T) T {
	t.Helper()
	select {
	case value := <-ready:
		return value
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for blocked compilation")
		var zero T
		return zero
	}
}
