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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package extern_test

import (
	"errors"
	"os"
	goruntime "runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// skipIfNoFileWatch skips on platforms without a native directory-watch
// backend (js/wasm): fsnotify has no js/wasm backend, so file:Listener's
// 'start() always fails there with "fsnotify not supported on the current
// platform".
func skipIfNoFileWatch(t *testing.T) {
	t.Helper()
	if goruntime.GOOS == "js" {
		t.Skip("skipping file-watch-dependent test on js/wasm")
	}
}

// TestFileListenerEvents exercises the create/modify/delete dispatch of a
// single file:Listener service against a real OS-level directory watch.
func TestFileListenerEvents(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-events-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerTouch exercises a timestamp-only change, which fsnotify
// reports as Chmod, being dispatched to onModify.
func TestFileListenerTouch(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "file-listener-touch-v", FuncName: "touchFile",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				later := time.Now().Add(time.Hour)
				return nil, os.Chtimes(args[0].(string), later, later)
			}},
	}
	runExtern(t, fileCase("file-listener/file-listener-touch-v"), testharness.NewTestPal(), externs)
}

// TestFileListenerRecursive exercises dynamic recursive registration: a
// subdirectory created after start is itself watched for further events.
func TestFileListenerRecursive(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-recursive-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerMultiService exercises dispatch to every service attached
// to the same listener.
func TestFileListenerMultiService(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-multi-service-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerDetach exercises attach/detach: a detached service must
// stop receiving events.
func TestFileListenerDetach(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-detach-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerAttachError exercises the attach-time validation requiring
// at least one of onCreate/onModify/onDelete.
func TestFileListenerAttachError(t *testing.T) {
	t.Parallel()
	runExtern(t, fileCase("file-listener/file-listener-attach-error-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerInitError exercises Listener init validation: empty path,
// non-existent directory, and a path that is not a directory.
func TestFileListenerInitError(t *testing.T) {
	t.Parallel()
	runExtern(t, fileCase("file-listener/file-listener-init-error-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerStartError exercises start() failing when the watched
// directory is removed between init and start, so the OS-level watch add
// fails.
func TestFileListenerStartError(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-start-error-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerRemoteMethodPanic exercises recovery from a panic inside a
// remote method: onCreate panics mid-lock, and the listener must log the
// panic to stderr, release the held lock, and keep dispatching later events
// rather than crashing the process or deadlocking on the stranded lock.
//
// This bypasses the runExtern/.txtar-golden helper the other listener tests
// use, since the recovery log line embeds a nondeterministic OS temp path
// that the golden normalization doesn't account for; asserting directly on
// pal.Stdout()/Stderr() avoids that.
func TestFileListenerRemoteMethodPanic(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	pal := testharness.NewTestPal()
	done := make(chan struct{})
	go func() {
		defer close(done)
		testharness.Run(t, fileCase("file-listener/file-listener-panic-v"), pal, nil)
	}()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("file listener deadlocked: held lock not released after remote-method panic")
	}

	if stdout := pal.Stdout(); stdout != "modified=true\ndeleted=true\n" {
		t.Errorf("stdout = %q, want %q", stdout, "modified=true\ndeleted=true\n")
	}
	stderr := pal.Stderr()
	if !strings.Contains(stderr, "error [ballerina/file]: panic while dispatching onCreate for ") {
		t.Errorf("stderr missing panic log line, got: %q", stderr)
	}
	if !strings.Contains(stderr, "divide by zero") {
		t.Errorf("stderr missing panic message, got: %q", stderr)
	}
}

// watchErrorPal reports a watch failure from its own goroutine, as the native
// watcher does, as soon as a directory watch starts,
// standing in for an fsnotify queue overflow, which can't be forced reliably.
type watchErrorPal struct {
	testharness.TestPal
}

func (p watchErrorPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	watch := base.FS.Watch
	base.FS.Watch = func(path string, recursive bool, handler pal.WatchHandler) (pal.WatchHandle, error) {
		handle, err := watch(path, recursive, handler)
		if err == nil {
			go handler(pal.WatchEvent{Err: errors.New("event queue overflow")})
		}
		return handle, err
	}
	return base
}

// TestFileListenerWatchError exercises a watch failure being logged to
// stderr while later events keep dispatching. Like the panic test, it asserts
// directly on stderr because the log line embeds a temp path.
func TestFileListenerWatchError(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	p := watchErrorPal{TestPal: testharness.NewTestPal()}
	testharness.Run(t, fileCase("file-listener/file-listener-events-v"), p, nil)

	want := "created=true\ncreatedPathMatches=true\nmodified=true\ndeleted=true\n"
	if stdout := p.Stdout(); stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if stderr := p.Stderr(); !strings.Contains(stderr, "failed: event queue overflow") {
		t.Errorf("stderr missing watch error log line, got: %q", stderr)
	}
}

// recordingWatchPal keeps every handler passed to FS.Watch so a test can
// replay an event through a watch that has since been stopped.
type recordingWatchPal struct {
	testharness.TestPal
	mu       *sync.Mutex
	handlers *[]pal.WatchHandler
}

func (p recordingWatchPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	watch := base.FS.Watch
	base.FS.Watch = func(path string, recursive bool, handler pal.WatchHandler) (pal.WatchHandle, error) {
		p.mu.Lock()
		*p.handlers = append(*p.handlers, handler)
		p.mu.Unlock()
		return watch(path, recursive, handler)
	}
	return base
}

// TestFileListenerStaleEvent exercises an event that a stopped watch had
// already received: it must not reach the services, whether the listener
// stays stopped or has been restarted with a new watch.
func TestFileListenerStaleEvent(t *testing.T) {
	t.Parallel()
	skipIfNoFileWatch(t)
	p := recordingWatchPal{TestPal: testharness.NewTestPal(), mu: &sync.Mutex{}, handlers: &[]pal.WatchHandler{}}
	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "file-listener-stale-event-v", FuncName: "replayOnFirstWatch",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				p.mu.Lock()
				first := (*p.handlers)[0]
				p.mu.Unlock()
				first(pal.WatchEvent{Path: args[0].(string), Op: pal.WatchCreate})
				return nil, nil
			}},
	}
	runExtern(t, fileCase("file-listener/file-listener-stale-event-v"), p, externs)
}
