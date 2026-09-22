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

//go:build !native_interp

package cli

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDriverSource(t *testing.T) {
	dir, err := ExtractDriverSource(t.TempDir(), "test-v0.0.1")
	if err != nil {
		t.Fatalf("ExtractDriverSource: %v", err)
	}

	for _, name := range []string{
		filepath.Join("cli", "go.mod"),
		filepath.Join("cli", "cmd"),
		filepath.Join("cli", "internal", "balrt"),
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("driver source must contain %s: %v", name, err)
		}
	}
	for _, name := range []string{"ast", "runtime", "projects", "semtypes"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("driver source must not contain dependency directory %s", name)
		}
	}
}

func TestExtractDriverSourceReusesCompleteReleaseCache(t *testing.T) {
	cacheRoot := t.TempDir()
	dir, err := ExtractDriverSource(cacheRoot, "test-v0.0.1")
	if err != nil {
		t.Fatalf("first ExtractDriverSource: %v", err)
	}
	marker := filepath.Join(dir, "marker")
	if err := os.WriteFile(marker, []byte("preserved"), 0o600); err != nil {
		t.Fatalf("writing marker: %v", err)
	}

	dir2, err := ExtractDriverSource(cacheRoot, "test-v0.0.1")
	if err != nil {
		t.Fatalf("second ExtractDriverSource: %v", err)
	}
	if dir2 != dir {
		t.Fatalf("second extraction returned %q, want %q", dir2, dir)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("complete release cache was unexpectedly replaced: %v", err)
	}
}

// TestExtractDriverSourceDevUsesCacheRoot verifies the "dev" version's cache
// lives under the given (private) cacheRoot, not the shared os.TempDir() —
// a predictable path in a world-writable shared temp dir could be
// pre-created by another local user/process with arbitrary content.
func TestExtractDriverSourceDevUsesCacheRoot(t *testing.T) {
	cacheRoot := t.TempDir()
	dir, err := ExtractDriverSource(cacheRoot, "dev")
	if err != nil {
		t.Fatalf("ExtractDriverSource: %v", err)
	}
	rel, err := filepath.Rel(cacheRoot, dir)
	if err != nil || rel == ".." || (len(rel) >= 2 && rel[:2] == "..") {
		t.Errorf("dev driver source dir %q is not under cacheRoot %q", dir, cacheRoot)
	}
	if _, err := os.Stat(filepath.Join(dir, "cli", "go.mod")); err != nil {
		t.Errorf("driver source must contain cli/go.mod: %v", err)
	}
}

func TestExtractDriverSourceRepairsIncompleteReleaseCache(t *testing.T) {
	cacheRoot := t.TempDir()
	dir, err := ExtractDriverSource(cacheRoot, "test-v0.0.1")
	if err != nil {
		t.Fatalf("first ExtractDriverSource: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "cli", "go.mod")); err != nil {
		t.Fatalf("removing cli/go.mod: %v", err)
	}

	if _, err := ExtractDriverSource(cacheRoot, "test-v0.0.1"); err != nil {
		t.Fatalf("repairing extraction: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cli", "go.mod")); err != nil {
		t.Errorf("cli/go.mod was not restored: %v", err)
	}
}

// TestInstallDriverSourceNeverUninstallsCompletedCache stresses concurrent
// installDriverSource calls against an initially-incomplete dir, with a
// watcher goroutine polling completeness throughout. Before the recheck/
// remove/rename replacement sequence was lock-protected, a caller that
// observed dir as incomplete could still delete it after a different
// concurrent caller had already installed a complete copy in the meantime,
// so the watcher could observe a spurious complete -> incomplete
// transition. This asserts that never happens.
func TestInstallDriverSourceNeverUninstallsCompletedCache(t *testing.T) {
	source := DriverSource()
	if source == nil {
		t.Skip("CLI driver source is not embedded in this build")
	}

	// Run many rounds of many simultaneously-released installers: the
	// vulnerable window (between a losing installer's failed-rename
	// recheck and its subsequent remove+rename) is narrow, so a single
	// round with only a handful of installers rarely lands another
	// installer's whole extraction inside it. Repeating with a shared
	// start gate (all installers released at once via closing startGate)
	// maximizes contention per round and across rounds.
	const rounds = 10
	const installers = 32
	for round := range rounds {
		stagingParent := t.TempDir()
		dir := filepath.Join(t.TempDir(), "driver-src")

		// Pre-populate dir as incomplete: only go.work exists, so every
		// installer below starts out needing to replace it.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("round %d: pre-creating incomplete dir: %v", round, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "go.work"), []byte(driverWorkspace), 0o644); err != nil {
			t.Fatalf("round %d: writing partial go.work: %v", round, err)
		}

		stopWatching := make(chan struct{})
		var sawUninstall atomic.Bool
		var watchWG sync.WaitGroup
		watchWG.Add(1)
		go func() {
			defer watchWG.Done()
			wasComplete := false
			for {
				select {
				case <-stopWatching:
					return
				default:
				}
				complete := extractedDriverSourceComplete(dir)
				if wasComplete && !complete {
					sawUninstall.Store(true)
				}
				wasComplete = complete
			}
		}()

		startGate := make(chan struct{})
		var installWG sync.WaitGroup
		errs := make([]error, installers)
		installWG.Add(installers)
		for i := range installers {
			go func(i int) {
				defer installWG.Done()
				<-startGate
				_, errs[i] = installDriverSource(dir, stagingParent, source)
			}(i)
		}
		close(startGate)
		installWG.Wait()
		close(stopWatching)
		watchWG.Wait()

		for i, err := range errs {
			if err != nil {
				t.Errorf("round %d installer %d: %v", round, i, err)
			}
		}
		if !extractedDriverSourceComplete(dir) {
			t.Errorf("round %d: dir must be complete after all installers finish", round)
		}
		if sawUninstall.Load() {
			t.Fatalf("round %d: watcher observed dir go from complete to incomplete — a concurrent installer deleted another's completed install", round)
		}
	}
}
