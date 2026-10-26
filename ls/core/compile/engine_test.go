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
	"context"
	"os"
	"sync"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	proj "github.com/ballerina-nutcracker/ballerina/projects"
)

func makeStableSnap(root string, gen uint64, byFile, resByFile map[string][]CompilerDiagnostic) StableSnapshot {
	if byFile == nil {
		byFile = map[string][]CompilerDiagnostic{}
	}
	if resByFile == nil {
		resByFile = map[string][]CompilerDiagnostic{}
	}
	return StableSnapshot{
		key:        CompilationKey{SourceRoot: root, Generation: gen},
		byFile:     byFile,
		resByFile:  resByFile,
		descriptor: "pkg",
	}
}

// newEngineTestServices wires a real workspace + engine with debounce(0) and
// a deferred shutdown so async cycles drain before the test ends.
func newEngineTestServices(t *testing.T) (*workspace.ProjectService, *CompilationService, *event.Bus) {
	t.Helper()
	bus := event.New()
	projects := workspace.New(os.DirFS, bus)
	svc := New(projects, bus, WithDebounce(0), WithMaxWorkers(2))
	t.Cleanup(func() { svc.Shutdown(); bus.Close() })
	return projects, svc, bus
}

func openDoc(t *testing.T, p *workspace.ProjectService, raw, text string) workspace.DocumentURI {
	t.Helper()
	u, err := workspace.NewFileURI(raw)
	if err != nil {
		t.Fatalf("NewFileURI: %v", err)
	}
	if _, err := p.Apply(context.Background(), workspace.DocumentChange{
		Kind: workspace.ChangeOpen, URI: u, Text: text, Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("Apply open: %v", err)
	}
	return u
}

func updateDoc(t *testing.T, p *workspace.ProjectService, raw, text string, version int32) {
	t.Helper()
	u, err := workspace.NewFileURI(raw)
	if err != nil {
		t.Fatalf("NewFileURI: %v", err)
	}
	if _, err := p.Apply(context.Background(), workspace.DocumentChange{
		Kind: workspace.ChangeUpdate, URI: u, Text: text, Version: version,
	}); err != nil {
		t.Fatalf("Apply update: %v", err)
	}
}

func collectGen(t *testing.T, bus *event.Bus, kind event.Kind) *genCollector {
	t.Helper()
	c := &genCollector{}
	bus.Subscribe([]event.Kind{kind}, func(e event.Event) { c.add(e.Generation()) })
	return c
}

type genCollector struct {
	mu  sync.Mutex
	got []uint64
}

func (c *genCollector) add(g uint64) {
	c.mu.Lock()
	c.got = append(c.got, g)
	c.mu.Unlock()
}

func (c *genCollector) slice() []uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.got
}

func lastGen(c *genCollector) uint64 {
	s := c.slice()
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}

func TestSnapshotStoreStaleGate(t *testing.T) {
	t.Parallel()
	bus := event.New()
	defer bus.Close()

	var current uint64 = 1
	store := newSnapshotStore(16, func(root string) (uint64, bool) { return current, true })

	var succeeded []uint64
	var cancelled []uint64
	bus.Subscribe([]event.Kind{event.CompilationSucceeded}, func(e event.Event) {
		succeeded = append(succeeded, e.Generation())
	})
	bus.Subscribe([]event.Kind{event.CompilationCancelled}, func(e event.Event) {
		cancelled = append(cancelled, e.Generation())
	})

	store.publishStable(makeStableSnap("/r", 1, nil, nil), bus)
	if got := store.latestStableGen("/r"); got != 1 {
		t.Fatalf("after gen1: stable gen = %d, want 1", got)
	}
	if len(succeeded) != 1 || succeeded[0] != 1 {
		t.Fatalf("CE-E1 for gen1: %v, want [1]", succeeded)
	}

	current = 2
	store.publishStable(makeStableSnap("/r", 1, nil, nil), bus)
	if got := store.latestStableGen("/r"); got != 1 {
		t.Fatalf("stale gen1 must not overwrite stable; got %d", got)
	}
	if len(cancelled) != 1 || cancelled[0] != 1 {
		t.Fatalf("CE-E3 for stale gen1: %v, want [1]", cancelled)
	}
	if len(succeeded) != 1 {
		t.Fatalf("stale publish must not emit CE-E1; got %v", succeeded)
	}
}

func TestSnapshotStoreEvictRoot(t *testing.T) {
	t.Parallel()
	bus := event.New()
	defer bus.Close()
	store := newSnapshotStore(16, func(string) (uint64, bool) { return 1, true })

	store.publishStable(makeStableSnap("/r", 1, nil, nil), bus)
	if _, ok := store.Stable("/r"); !ok {
		t.Fatal("expected stable for /r")
	}
	store.evictRoot("/r")
	if _, ok := store.Stable("/r"); ok {
		t.Fatal("expected stable evicted for /r")
	}
}

func TestEngineSingleFlightAndStaleGate(t *testing.T) {
	t.Parallel()
	projects, svc, bus := newEngineTestServices(t)
	succeeded := collectGen(t, bus, event.CompilationSucceeded)
	cancelled := collectGen(t, bus, event.CompilationCancelled)

	u := openDoc(t, projects, tempFileURI(t, "sf.bal").String(), "public function main() {}\n")
	updateDoc(t, projects, u.String(), "public function main() {}\nfunction f(){}\n", 2)
	svc.Flush()

	if len(succeeded.slice()) == 0 || lastGen(succeeded) != 2 {
		t.Fatalf("CE-E1 gens = %v, want last == 2", succeeded.slice())
	}
	snap, ok := svc.store.Stable(u.Path())
	if !ok || snap.Key().Generation != 2 {
		t.Fatalf("stable gen = %v ok=%v, want 2", snap.Key().Generation, ok)
	}
	if len(cancelled.slice()) == 0 {
		t.Fatalf("expected at least one CE-E3 for the superseded generation, got %v", cancelled.slice())
	}
}

func TestEnginePanicRecovery(t *testing.T) {
	t.Parallel()
	projects, svc, bus := newEngineTestServices(t)
	failed := collectGen(t, bus, event.CompilationFailed)

	svc.compileFn = func(pkg *proj.Package, cyc compileCycle) cycleResult { panic("boom") }
	u := openDoc(t, projects, tempFileURI(t, "panic.bal").String(), "public function main() {}\n")
	svc.Flush()

	if len(failed.slice()) != 1 || failed.slice()[0] != 1 {
		t.Fatalf("CE-E2: %v, want [1]", failed.slice())
	}
	if _, ok := svc.store.Stable(u.Path()); ok {
		t.Fatal("panicking compile must not store a snapshot")
	}

	succeeded := collectGen(t, bus, event.CompilationSucceeded)
	svc.compileFn = realCompilePackage
	updateDoc(t, projects, u.String(), "public function main() {}\n", 2)
	svc.Flush()
	if len(succeeded.slice()) == 0 || lastGen(succeeded) != 2 {
		t.Fatalf("after recovery: CE-E1 = %v, want last 2", succeeded.slice())
	}
}

func TestEngineResolutionVsCompilationClassification(t *testing.T) {
	t.Parallel()
	projects, svc, bus := newEngineTestServices(t)
	e5a := collectGen(t, bus, event.ResolutionDiagnosticsReady)
	e5b := collectGen(t, bus, event.CompilationDiagnosticsReady)

	svc.compileFn = func(pkg *proj.Package, cyc compileCycle) cycleResult {
		return cycleResult{resolutionErrored: true, byFile: map[string][]CompilerDiagnostic{}, resByFile: map[string][]CompilerDiagnostic{}}
	}
	openDoc(t, projects, tempFileURI(t, "reserr.bal").String(), "public function main() {}\n")
	svc.Flush()
	if len(e5a.slice()) != 1 {
		t.Fatalf("resolution-error cycle: E5a=%d, want 1", len(e5a.slice()))
	}
	if len(e5b.slice()) != 0 {
		t.Fatalf("resolution-error cycle: E5b=%d, want 0", len(e5b.slice()))
	}

	svc.compileFn = func(pkg *proj.Package, cyc compileCycle) cycleResult {
		return cycleResult{byFile: map[string][]CompilerDiagnostic{}, resByFile: map[string][]CompilerDiagnostic{}}
	}
	openDoc(t, projects, tempFileURI(t, "clean.bal").String(), "public function main() {}\n")
	svc.Flush()
	if len(e5a.slice()) < 2 {
		t.Fatalf("clean cycle: E5a=%d, want >=2", len(e5a.slice()))
	}
	if len(e5b.slice()) == 0 {
		t.Fatalf("clean cycle: E5b=%d, want >=1", len(e5b.slice()))
	}
}

func TestEngineCleanCycleEmitsE5aAndE5b(t *testing.T) {
	t.Parallel()
	projects, svc, bus := newEngineTestServices(t)
	e5a := collectGen(t, bus, event.ResolutionDiagnosticsReady)
	e5b := collectGen(t, bus, event.CompilationDiagnosticsReady)

	// The unused variable is a compiler diagnostic, not a package-resolution error.
	openDoc(t, projects, tempFileURI(t, "clean.bal").String(), "public function main() {\n int x = 1;\n}\n")
	svc.Flush()

	if len(e5a.slice()) == 0 {
		t.Fatalf("clean cycle: CE-E5a = %d, want >=1", len(e5a.slice()))
	}
	if len(e5b.slice()) == 0 {
		t.Fatalf("clean cycle: CE-E5b = %d, want >=1", len(e5b.slice()))
	}
	if lastGen(e5a) != lastGen(e5b) {
		t.Fatalf("clean cycle generation mismatch: E5a=%d E5b=%d", lastGen(e5a), lastGen(e5b))
	}
}

func TestEngineDiagnosticsFor(t *testing.T) {
	t.Parallel()
	projects, svc, bus := newEngineTestServices(t)
	_ = bus
	u := openDoc(t, projects, tempFileURI(t, "df.bal").String(), "public function main() {\n int x = 1;\n}\n")
	svc.Flush()

	diags, gen, ok := svc.DiagnosticsFor(u.Path())
	if !ok {
		t.Fatalf("expected stable snapshot for %s", u.Path())
	}
	if gen != 1 {
		t.Fatalf("generation = %d, want 1", gen)
	}
	if len(diags[u]) == 0 {
		t.Fatalf("expected diagnostics for %s", u)
	}
}

func TestEngineCancelSupersedesInFlight(t *testing.T) {
	t.Parallel()
	projects, svc, bus := newEngineTestServices(t)
	succeeded := collectGen(t, bus, event.CompilationSucceeded)
	cancelled := collectGen(t, bus, event.CompilationCancelled)

	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseNow := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseNow()
	svc.compileFn = func(pkg *proj.Package, cyc compileCycle) cycleResult {
		close(started)
		<-release
		return cycleResult{byFile: map[string][]CompilerDiagnostic{}, resByFile: map[string][]CompilerDiagnostic{}}
	}

	u := openDoc(t, projects, tempFileURI(t, "cancel.bal").String(), "public function main() {}\n")
	<-started

	svc.Cancel()
	if g, ok := projects.Generation(u.Path()); !ok || g != 2 {
		t.Fatalf("after Cancel: generation = %d ok=%v, want 2", g, ok)
	}

	releaseNow()
	svc.Flush()

	if len(succeeded.slice()) != 0 {
		t.Fatalf("cancelled compile must not emit CE-E1; got %v", succeeded.slice())
	}
	if len(cancelled.slice()) != 1 || cancelled.slice()[0] != 1 {
		t.Fatalf("cancelled compile must emit CE-E3 for gen 1; got %v", cancelled.slice())
	}
	if _, ok := svc.store.Stable(u.Path()); ok {
		t.Fatal("cancelled compile must store no stable snapshot")
	}
}
