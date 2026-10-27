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
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

type sealedWaitContext struct {
	context.Context
	waiting chan<- struct{}
	once    sync.Once
}

func (c *sealedWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { c.waiting <- struct{}{} })
	return c.Context.Done()
}

type pausedCycleReader struct {
	projectReader
	paused  atomic.Bool
	entered chan struct{}
	resume  chan struct{}
}

func (r *pausedCycleReader) CurrentProject(root string) (projects.Project, uint64, bool) {
	if r.paused.CompareAndSwap(false, true) {
		close(r.entered)
		<-r.resume
	}
	return r.projectReader.CurrentProject(root)
}

func TestRecoveredPanicRejectsReaderRequeueAfterRecovery(t *testing.T) {
	t.Parallel()
	projectService, svc, bus := newEngineTestServices(t)
	svc.compileFn = func(*projects.Package, compileCycle) cycleResult { return cycleResult{} }
	u := openDoc(t, projectService, tempFileURI(t, "requeue-panic.bal").String(), "public function main() {}\n")
	svc.Flush()
	projectService.Supersede(u.Path())
	failed := collectGen(t, bus, event.CompilationFailed)
	var attempts atomic.Int32
	svc.compileFn = func(*projects.Package, compileCycle) cycleResult {
		attempts.Add(1)
		panic("requeue panic")
	}
	reader := &pausedCycleReader{projectReader: svc.reader, entered: make(chan struct{}), resume: make(chan struct{})}
	svc.reader = reader
	var resumeOnce sync.Once
	resume := func() { resumeOnce.Do(func() { close(reader.resume) }) }
	defer resume()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	readDone := make(chan bool, 1)
	go func() {
		_, ok := svc.SealedModuleFor(ctx, u)
		readDone <- ok
	}()
	select {
	case <-reader.entered:
	case <-ctx.Done():
		t.Fatal("reader did not reach scheduling before expiry")
	}

	svc.enqueueImmediately(u.Path(), 2)
	svc.Flush()
	if !svc.generationFailed(u.Path(), 2) || svc.cycleQueuedOrRunning(u.Path()) {
		t.Fatal("recovery did not finish while reader was paused before enqueue")
	}
	for range cap(svc.workSem) {
		svc.workSem <- struct{}{}
	}
	var releaseWorkersOnce sync.Once
	releaseWorkers := func() {
		releaseWorkersOnce.Do(func() {
			for range cap(svc.workSem) {
				<-svc.workSem
			}
		})
	}
	defer releaseWorkers()
	resume()
	select {
	case ok := <-readDone:
		if ok || ctx.Err() != nil {
			t.Errorf("resumed sealed read: ok=%v context=%v", ok, ctx.Err())
		}
	case <-ctx.Done():
		releaseWorkers()
		<-readDone
		svc.Flush()
		t.Fatal("failed-generation read was rescheduled into an occupied worker pool")
	}
	releaseWorkers()
	svc.Flush()
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts after recovery overtook reader = %d, want 1", got)
	}
	if got := failed.slice(); len(got) != 1 || got[0] != 2 {
		t.Errorf("failure events = %v, want [2]", got)
	}

	svc.compileFn = realCompilePackage
	after := "public function main() { /* after requeue rejection */ }\n"
	updateDoc(t, projectService, u.String(), after, 2)
	svc.Flush()
	view, ok := svc.SealedModuleFor(context.Background(), u)
	if !ok {
		t.Fatal("failed-generation rejection blocked a later real compilation")
	}
	if text, ok := view.SourceText(u); !ok || text != after {
		t.Errorf("later generation source = %q, %v", text, ok)
	}
}

func TestRecoveredPanicLifecycleResetAllowsSameGeneration(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind event.Kind
	}{
		{"registration", event.ProjectRegistered},
		{"eviction", event.ProjectEvicted},
		{"kind transition", event.ProjectKindTransitioned},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			projectService, svc, bus := newEngineTestServices(t)
			svc.compileFn = func(*projects.Package, compileCycle) cycleResult { panic("lifecycle reset panic") }
			u := openDoc(t, projectService, tempFileURI(t, "reset-panic.bal").String(), "public function main() {}\n")
			svc.Flush()
			if !svc.generationFailed(u.Path(), 1) {
				t.Fatal("missing failure marker before lifecycle reset")
			}
			switch tc.kind {
			case event.ProjectRegistered:
				bus.Publish(event.NewProjectRegisteredEvent(u.Path()))
			case event.ProjectEvicted:
				bus.Publish(event.NewProjectEvictedEvent(u.Path(), event.EvictionLRU))
				bus.Publish(event.NewProjectRegisteredEvent(u.Path()))
			case event.ProjectKindTransitioned:
				bus.Publish(event.NewProjectKindTransitionedEvent(u.Path(), u.Path()))
			}
			svc.compileFn = realCompilePackage
			svc.enqueueImmediately(u.Path(), 1)
			svc.Flush()
			if svc.generationFailed(u.Path(), 1) {
				t.Fatal("lifecycle reset retained the failure marker")
			}
			if _, ok := svc.SealedModuleFor(context.Background(), u); !ok {
				t.Fatal("lifecycle reset did not allow real compilation of the same generation")
			}
		})
	}
}

func TestRecoveredPanicRejectsAlreadyPendingGeneration(t *testing.T) {
	t.Parallel()
	projectService, svc, bus := newEngineTestServices(t)
	failed := collectGen(t, bus, event.CompilationFailed)
	succeeded := collectGen(t, bus, event.CompilationSucceeded)
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseNow := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseNow()
	var attempts atomic.Int32
	svc.compileFn = func(*projects.Package, compileCycle) cycleResult {
		if attempts.Add(1) == 1 {
			close(started)
			<-release
		}
		panic("pending panic")
	}
	u := openDoc(t, projectService, tempFileURI(t, "pending-panic.bal").String(), "public function main() {}\n")
	<-started
	svc.enqueueImmediately(u.Path(), 1)
	releaseNow()
	svc.Flush()
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts including pre-failure pending request = %d, want 1", got)
	}
	if got := failed.slice(); len(got) != 1 || got[0] != 1 {
		t.Errorf("failure events = %v, want [1]", got)
	}
	if got := succeeded.slice(); len(got) != 0 {
		t.Errorf("success events for failed generation = %v", got)
	}
}

func TestRecoveredPanicSealedWaitersAndLaterGeneration(t *testing.T) {
	t.Parallel()
	projectService, svc, bus := newEngineTestServices(t)
	failed := collectGen(t, bus, event.CompilationFailed)
	succeeded := collectGen(t, bus, event.CompilationSucceeded)
	var attempts atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseNow := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseNow()
	svc.compileFn = func(*projects.Package, compileCycle) cycleResult {
		if attempts.Add(1) == 1 {
			close(started)
			<-release
		}
		panic("sealed panic")
	}
	bus.Subscribe([]event.Kind{event.CompilationFailed}, func(e event.Event) {
		if !svc.generationFailed(e.SourceRoot(), e.Generation()) {
			t.Error("failure marker absent during failure publication")
		}
		ip, ok := svc.store.InProgress(e.SourceRoot())
		if !ok {
			t.Error("in-progress snapshot absent during failure publication")
			return
		}
		select {
		case <-ip.Done():
			t.Error("waiters released before failure publication")
		default:
		}
	})
	u := openDoc(t, projectService, tempFileURI(t, "sealed-panic.bal").String(), "public function main() {}\n")
	<-started
	var waiters sync.WaitGroup
	ready := make(chan struct{}, 8)
	for range 8 {
		waiters.Go(func() {
			deadline, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			ctx := &sealedWaitContext{Context: deadline, waiting: ready}
			if _, ok := svc.SealedModuleFor(ctx, u); ok {
				t.Error("panicked generation supplied a sealed module")
			}
			if ctx.Err() != nil {
				t.Error("sealed read waited for context expiry instead of failure")
			}
			if !svc.generationFailed(u.Path(), 1) {
				t.Error("failure marker absent after waiter release")
			}
		})
	}
	for range 8 {
		<-ready
	}
	releaseNow()
	waiters.Wait()
	svc.Flush()
	for range 3 {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, ok := svc.SealedModuleFor(ctx, u)
		if ok || ctx.Err() != nil {
			t.Errorf("repeated read: ok=%v context=%v", ok, ctx.Err())
		}
		cancel()
	}
	svc.Flush()
	if got := attempts.Load(); got != 1 {
		t.Errorf("compilation attempts = %d, want 1", got)
	}
	if got := failed.slice(); len(got) != 1 || got[0] != 1 {
		t.Errorf("failure events = %v, want [1]", got)
	}
	if got := succeeded.slice(); len(got) != 0 {
		t.Errorf("successful events for panicked generation = %v", got)
	}
	if _, ok := svc.store.Stable(u.Path()); ok {
		t.Error("panicked generation stored a successful snapshot")
	}

	svc.compileFn = realCompilePackage
	after := "public function main() { /* next generation */ }\n"
	updateDoc(t, projectService, u.String(), after, 2)
	svc.Flush()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	view, ok := svc.SealedModuleFor(ctx, u)
	if !ok {
		t.Fatal("real compilation after failure did not supply a sealed module")
	}
	if text, ok := view.SourceText(u); !ok || text != after {
		t.Errorf("later generation source = %q, %v", text, ok)
	}
	svc.Flush()
	if got := succeeded.slice(); len(got) != 1 || got[0] != 2 {
		t.Errorf("later generation successes = %v, want [2]", got)
	}
}

type panicLogWriter struct {
	bytes.Buffer
	closes int
}

func (w *panicLogWriter) Close() error {
	w.closes++
	return nil
}

func TestRecoveredPanicHostWriter(t *testing.T) {
	t.Parallel()
	var logs panicLogWriter
	logger := observability.New(&logs)
	bus := event.New(event.WithLogf(func(string, ...any) { t.Error("compile panic used subscriber diagnostics hook") }))
	defer bus.Close()
	projectService := workspace.New(os.DirFS, bus)
	svc := New(projectService, bus, WithDebounce(0), WithLogger(logger), WithLogger(nil))
	defer svc.Shutdown()
	if svc.logger != logger {
		t.Fatal("nil logger option replaced the injected logger")
	}
	svc.compileFn = func(*projects.Package, compileCycle) cycleResult { panic("host-writer panic") }
	u := openDoc(t, projectService, tempFileURI(t, "log-panic.bal").String(), "public function main() {}\n")
	svc.Flush()
	svc.Shutdown()
	out := logs.String()
	for _, want := range []string{"level=ERROR", "recovered=\"host-writer panic\"", "root=", u.Path(), "generation=1", "stack=", "runtime/debug.Stack", "runCycle", "TestRecoveredPanicHostWriter"} {
		if !strings.Contains(out, want) {
			t.Errorf("host log missing %q: %s", want, out)
		}
	}
	if got := strings.Count(out, "level=ERROR"); got != 1 {
		t.Errorf("recovery records = %d, want 1", got)
	}
	if logs.closes != 0 {
		t.Errorf("compiler closed host writer %d times", logs.closes)
	}
}

func TestRecoveredPanicNilLogger(t *testing.T) {
	t.Parallel()
	projectService, svc, _ := newEngineTestServices(t)
	WithLogger(nil)(svc)
	if svc.logger == nil {
		t.Fatal("nil logger option removed the quiet default")
	}
	svc.compileFn = func(*projects.Package, compileCycle) cycleResult { panic("quiet panic") }
	u := openDoc(t, projectService, tempFileURI(t, "quiet-panic.bal").String(), "public function main() {}\n")
	svc.Flush()
	if !svc.generationFailed(u.Path(), 1) {
		t.Fatal("nil logger prevented panic recovery")
	}
}

func TestRecoveredStalePanicDoesNotBlockCurrentGeneration(t *testing.T) {
	t.Parallel()
	projectService, svc, bus := newEngineTestServices(t)
	failed := collectGen(t, bus, event.CompilationFailed)
	succeeded := collectGen(t, bus, event.CompilationSucceeded)
	var attempts atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseNow := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseNow()
	svc.compileFn = func(pkg *projects.Package, cyc compileCycle) cycleResult {
		if attempts.Add(1) == 1 {
			close(started)
			<-release
			panic("superseded panic")
		}
		return realCompilePackage(pkg, cyc)
	}
	u := openDoc(t, projectService, tempFileURI(t, "stale-panic.bal").String(), "public function main() {}\n")
	<-started
	after := "public function main() { /* superseding generation */ }\n"
	updateDoc(t, projectService, u.String(), after, 2)
	bus.Flush()
	releaseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	view, ok := svc.SealedModuleFor(ctx, u)
	if !ok {
		t.Fatal("stale panic blocked the current sealed read")
	}
	if text, ok := view.SourceText(u); !ok || text != after {
		t.Errorf("current source = %q, %v", text, ok)
	}
	svc.Flush()
	if !svc.generationFailed(u.Path(), 1) || svc.generationFailed(u.Path(), 2) {
		t.Error("failure marker is not generation-specific")
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("compilation attempts = %d, want 2", got)
	}
	if got := failed.slice(); len(got) != 1 || got[0] != 1 {
		t.Errorf("failure events = %v, want [1]", got)
	}
	if got := succeeded.slice(); len(got) != 1 || got[0] != 2 {
		t.Errorf("success events = %v, want [2]", got)
	}
}
