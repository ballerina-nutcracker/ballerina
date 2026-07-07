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

// Package packageresolution contains corpus-based CLI integration tests for
// package-resolution scenarios. Each subdirectory under corpus/package-resolution/
// that contains a project/ subdirectory is treated as a scenario.
//
// Run with:
//
//	go test ./corpus/package-resolution -run TestPackageResolutionScenarios -count=1 -v
package packageresolution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
)

var (
	pkgResBinsOnce sync.Once
	pkgResBalBin   string
	pkgResRepoRoot string
	pkgResBinsErr  error
)

// TestPackageResolutionScenarios runs each subdirectory with a project/ as a
// scenario: `bal run <project>` with BAL_ENV pointed at the scenario's bal_env/.
func TestPackageResolutionScenarios(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}

	ensureBalBinary(t)

	// Resolve the directory that contains this test file.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	scenariosDir := filepath.Dir(thisFile)

	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		t.Fatalf("failed to read scenarios directory %s: %v", scenariosDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		scenarioDir := filepath.Join(scenariosDir, name)
		projectDir := filepath.Join(scenarioDir, "project")
		if _, err := os.Stat(projectDir); err != nil {
			if os.IsNotExist(err) {
				// No project/ subdirectory — skip (could be a docs or future-scenario stub).
				continue
			}
			t.Fatalf("failed to stat scenario project dir %s: %v", projectDir, err)
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runScenario(t, scenarioDir, projectDir)
		})
	}
}

// runScenario executes a single package-resolution scenario.
func runScenario(t *testing.T, scenarioDir, projectDir string) {
	t.Helper()

	balEnvDir := filepath.Join(scenarioDir, "bal_env")
	expectedTxtarPath := filepath.Join(scenarioDir, "expected.txtar")

	// Read and parse expected.txtar before running anything so a missing file
	// fails fast rather than after a potentially slow binary invocation.
	expectedStdout, expectedStderr, err := loadScenarioTxtar(expectedTxtarPath)
	if err != nil {
		t.Fatalf("failed to load expected.txtar at %s: %v", expectedTxtarPath, err)
	}

	// Run: bal run <absolute-project-path>
	stdout, stderr := runBalRun(t, projectDir, balEnvDir)

	// Log full captured output on failure so CI logs surface stderr context
	// when only stdout-expected is non-empty. t.Cleanup runs in LIFO so
	// the failure marker is recorded before this fires.
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("bal run\n  BAL_ENV=%s\n  project=%s\n  stdout=%q\n  stderr=%q",
				balEnvDir, projectDir, stdout, stderr)
		}
	})

	// Assert stdout contains expected lines (substring match, per scenario pattern).
	assertContains(t, "stdout", stdout, expectedStdout)

	// Assert stderr contains expected lines (substring match).
	assertContains(t, "stderr", stderr, expectedStderr)
}

// runBalRun invokes the bal binary with `run <projectDir>`, overriding BAL_ENV
// to the given balEnvDir. Returns captured stdout and stderr.
func runBalRun(t *testing.T, projectDir, balEnvDir string) (stdout, stderr string) {
	t.Helper()

	deadline, ok := t.Deadline()
	var ctx context.Context
	var cancel context.CancelFunc
	if ok {
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, pkgResBalBin, "run", projectDir)
	cmd.Dir = pkgResRepoRoot

	// Build subprocess environment: inherit current env, then override BAL_ENV.
	env := os.Environ()
	env = setEnvVar(env, "BAL_ENV", balEnvDir)
	cmd.Env = env

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			t.Fatalf("failed to execute bal run %s: %v\nstdout:\n%s\nstderr:\n%s",
				projectDir, runErr, stdoutBuf.String(), stderrBuf.String())
		}
		// Non-zero exit is allowed — the assertions below will catch unexpected failures.
	}

	return stdoutBuf.String(), stderrBuf.String()
}

// assertContains checks that each non-empty trimmed line from expected is a
// substring of actual. Empty expected strings pass unconditionally.
func assertContains(t *testing.T, label, actual, expected string) {
	t.Helper()
	if strings.TrimSpace(expected) == "" {
		return
	}
	for _, line := range strings.Split(expected, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(actual, line) {
			t.Errorf("%s: expected to contain %q\nactual %s:\n%s", label, line, label, actual)
		}
	}
}

// loadScenarioTxtar parses a txtar file with stdout and stderr sections.
// Both sections are optional; absent sections return an empty string.
func loadScenarioTxtar(path string) (stdout, stderr string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	archive := txtar.Parse(data)
	for _, f := range archive.Files {
		switch f.Name {
		case "stdout":
			stdout = string(f.Data)
		case "stderr":
			stderr = string(f.Data)
		default:
			return "", "", fmt.Errorf("unexpected section %q in %s (want stdout or stderr)", f.Name, path)
		}
	}
	return stdout, stderr, nil
}

// setEnvVar returns a copy of env with the given key set to value. If the key
// already exists in env it is replaced; otherwise the pair is appended.
func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			result = append(result, prefix+value)
			found = true
		} else {
			result = append(result, e)
		}
	}
	if !found {
		result = append(result, prefix+value)
	}
	return result
}

// ensureBalBinary builds the bal CLI binary once for the lifetime of the test
// process using a sync.Once guard.
func ensureBalBinary(t *testing.T) {
	t.Helper()
	pkgResBinsOnce.Do(func() {
		// Resolve repo root: this file is at <repo>/corpus/package-resolution/resolution_test.go
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			pkgResBinsErr = fmt.Errorf("runtime.Caller failed")
			return
		}
		// corpus/package-resolution/ -> corpus/ -> repo root
		pkgResRepoRoot = filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

		tmpDir, err := os.MkdirTemp("", "bal-pkg-res-test")
		if err != nil {
			pkgResBinsErr = fmt.Errorf("create temp dir: %w", err)
			return
		}

		binName := "bal"
		if runtime.GOOS == "windows" {
			binName = "bal.exe"
		}
		pkgResBalBin = filepath.Join(tmpDir, binName)

		buildCmd := exec.Command("go", "build", "-o", pkgResBalBin, "./cli/cmd")
		buildCmd.Dir = pkgResRepoRoot
		if out, err := buildCmd.CombinedOutput(); err != nil {
			pkgResBinsErr = fmt.Errorf("build bal binary: %w\n%s", err, string(out))
		}
	})
	if pkgResBinsErr != nil {
		t.Fatalf("bal binary setup: %v", pkgResBinsErr)
	}
}
