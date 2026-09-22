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

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// sharedBalBinary is built once and reused by tests that just need a working
// `bal`, so the worktree checkout + build isn't repeated per test.
var (
	sharedBalOnce sync.Once
	sharedBalPath string
	sharedBalErr  error
	// Recorded for TestMain: the shared checkout outlives every individual
	// test, so t.Cleanup cannot tear it down. Guarded because the interrupt
	// handler reads them while ensureSharedBalBinary may still be writing.
	sharedBalMu       sync.Mutex
	sharedBalRoot     string
	sharedBalWorktree string
)

// interruptHelperEnv names the marker file the re-executed helper process
// writes from its cleanup; see TestInterruptRunsCleanupBeforeExit.
const interruptHelperEnv = "HTTPBENCH_INTERRUPT_HELPER_MARKER"

// TestMain tears down the shared checkout. Without it every invocation of this
// package's tests leaks a git worktree registration plus ~420MB of checkout.
func TestMain(m *testing.M) {
	// The helper process installs its own handler and must not race this one
	// to os.Exit.
	stop := func() {}
	if os.Getenv(interruptHelperEnv) == "" {
		stop = onInterrupt(cleanupSharedBal)
	}
	code := m.Run()
	stop()
	cleanupSharedBal() // not deferred: os.Exit below would skip it
	os.Exit(code)
}

func cleanupSharedBal() {
	sharedBalMu.Lock()
	worktree, root := sharedBalWorktree, sharedBalRoot
	sharedBalMu.Unlock()
	if worktree != "" {
		removeWorktree(worktree)
	}
	if root != "" {
		_ = os.RemoveAll(root)
	}
}

func ensureSharedBalBinary(t *testing.T) string {
	t.Helper()
	skipWorktreeIntegrationOnWindows(t)
	sharedBalOnce.Do(func() {
		root, err := os.MkdirTemp("", "httpbench-shared-bal-*")
		if err != nil {
			sharedBalErr = err
			return
		}
		sharedBalMu.Lock()
		sharedBalRoot = root
		sharedBalMu.Unlock()
		wt, err := checkoutWorktree(root, "shared", "HEAD")
		if err != nil {
			sharedBalErr = err
			return
		}
		sharedBalMu.Lock()
		sharedBalWorktree = wt
		sharedBalMu.Unlock()
		sharedBalPath, sharedBalErr = buildInterpreter(wt)
	})
	if sharedBalErr != nil {
		t.Fatalf("failed to build shared bal binary for tests: %v", sharedBalErr)
	}
	return sharedBalPath
}

// skipWorktreeIntegrationOnWindows skips real git-worktree checkout/build
// tests there: httpbench only ever runs on ubuntu-latest in CI, and Windows
// test runs don't feed Codecov coverage anyway.
func skipWorktreeIntegrationOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("worktree checkout/build integration is exercised on Linux/macOS only")
	}
}

// requireWrk fails the test if wrk is unavailable (mirrors requireHyperfine
// in the sibling compiler-tools/benchmark tool). wrk isn't installable on
// Windows, so skip there instead.
func requireWrk(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("wrk"); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("wrk is unavailable on windows: %v", err)
		}
		t.Fatalf("wrk is required but unavailable: %v", err)
	}
}

func findFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestSanitize(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"feature/foo": "feature-foo",
		"a:b\\c":      "a-b-c",
		"HEAD":        "HEAD",
		"v1.2.3":      "v1.2.3",
	}
	for in, want := range cases {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRunCmdSuccessAndFailure(t *testing.T) {
	t.Parallel()
	if err := runCmd(t.TempDir(), "go", "version"); err != nil {
		t.Errorf("runCmd(go version) = %v, want nil", err)
	}
	if err := runCmd(t.TempDir(), "definitely-not-a-real-binary-xyz"); err == nil {
		t.Error("expected an error for a nonexistent binary")
	}
}

func TestRunCmdSilentSuccessAndFailure(t *testing.T) {
	t.Parallel()
	if err := runCmdSilent(t.TempDir(), "go", "version"); err != nil {
		t.Errorf("runCmdSilent(go version) = %v, want nil", err)
	}
	if err := runCmdSilent(t.TempDir(), "definitely-not-a-real-binary-xyz"); err == nil {
		t.Error("expected an error for a nonexistent binary")
	}
}

// Not parallel: real git worktree checkout against this repo's own .git.
func TestCheckoutAndRemoveWorktree(t *testing.T) {
	skipWorktreeIntegrationOnWindows(t)
	root := t.TempDir()
	wt, err := checkoutWorktree(root, "role1", "HEAD")
	if err != nil {
		t.Fatalf("checkoutWorktree: %v", err)
	}
	if want := "worktree-role1-HEAD"; filepath.Base(wt) != want {
		t.Errorf("worktree path = %q, want basename %q", wt, want)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Fatalf("expected worktree directory to exist: %v", err)
	}

	removeWorktree(wt)
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Fatalf("expected worktree directory to be removed, stat err = %v", err)
	}
}

// newBareWorktree registers a worktree without materializing the checkout.
// The removal and prune paths act on the registration, not the files, so the
// tests that exercise them do not need a 17k-file copy of the repo.
func newBareWorktree(t *testing.T, name string) string {
	t.Helper()
	skipWorktreeIntegrationOnWindows(t)
	path := filepath.Join(t.TempDir(), name)
	if err := runCmdSilent(".", "git", "worktree", "add", "--no-checkout", "--detach", path, "HEAD"); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}
	return path
}

// TestRemoveWorktreeRemovesLockedWorktree covers the interrupted-`worktree
// add` case: git leaves a "locked" marker behind, which a single --force
// refuses to remove and `git worktree prune` skips entirely. Not parallel —
// see TestCheckoutAndRemoveWorktree.
func TestRemoveWorktreeRemovesLockedWorktree(t *testing.T) {
	wt := newBareWorktree(t, "locked")
	// Torn down with raw git so a regression in removeWorktree fails the test
	// without also leaking the locked registration it is meant to reclaim.
	defer func() {
		_ = runCmdSilent(".", "git", "worktree", "unlock", wt)
		_ = runCmdSilent(".", "git", "worktree", "remove", "--force", "--force", wt)
	}()
	if err := runCmdSilent(".", "git", "worktree", "lock", wt); err != nil {
		t.Fatalf("git worktree lock: %v", err)
	}

	removeWorktree(wt)
	if worktreeRegistered(t, wt) {
		t.Fatalf("expected %q to be unregistered after removeWorktree", wt)
	}
}

// TestPruneWorktreesReclaimsReapedCheckouts reproduces the $TMPDIR-reaping
// case: the directory disappears behind git's back, leaving a registration
// that `worktree remove` can no longer reach. Not parallel — see
// TestCheckoutAndRemoveWorktree.
func TestPruneWorktreesReclaimsReapedCheckouts(t *testing.T) {
	wt := newBareWorktree(t, "reaped")
	if err := os.RemoveAll(wt); err != nil {
		t.Fatalf("removing the checkout behind git's back: %v", err)
	}

	pruneWorktrees()
	if worktreeRegistered(t, wt) {
		removeWorktree(wt)
		t.Fatalf("expected pruneWorktrees to reclaim the registration for %q", wt)
	}
}

func worktreeRegistered(t *testing.T, path string) bool {
	t.Helper()
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		t.Fatalf("git worktree list: %v", err)
	}
	resolved := resolvedWorktreePath(t, path)
	for _, line := range strings.Split(string(out), "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok && (p == path || p == resolved) {
			return true
		}
	}
	return false
}

// resolvedWorktreePath renders path the way `git worktree list` does, with
// symlinks expanded. It resolves the parent rather than path itself, because
// callers ask about checkouts that have already been deleted.
func resolvedWorktreePath(t *testing.T, path string) string {
	t.Helper()
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.Join(parent, filepath.Base(path))
}

// TestInterruptRunsCleanupBeforeExit re-executes this test binary as a child,
// sends it SIGTERM, and checks that the handler ran cleanup before exiting —
// the behaviour the whole fix exists for, which no in-process test can reach.
func TestInterruptRunsCleanupBeforeExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sending SIGTERM to a child process is unsupported on windows")
	}
	marker := filepath.Join(t.TempDir(), "cleanup-ran")
	cmd := exec.Command(os.Args[0], "-test.run=TestInterruptHelperProcess")
	cmd.Env = append(os.Environ(), interruptHelperEnv+"="+marker)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting helper: %v", err)
	}
	// Reaped on every path: the early t.Fatalf returns below would otherwise
	// orphan a child sleeping out its timeout.
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	// The helper prints once its handler is installed; signalling earlier would
	// kill it with the default disposition.
	if _, err := bufio.NewReader(stdout).ReadString('\n'); err != nil {
		t.Fatalf("waiting for helper to install its handler: %v\nhelper stderr:\n%s", err, stderr.String())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signalling helper: %v", err)
	}

	var exitErr *exec.ExitError
	if err := cmd.Wait(); !errors.As(err, &exitErr) {
		t.Fatalf("helper exit = %v, want an ExitError\nhelper stderr:\n%s", err, stderr.String())
	} else if got := exitErr.ExitCode(); got != interruptExitCode {
		t.Errorf("helper exit code = %d, want %d", got, interruptExitCode)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("cleanup did not run before exit: %v", err)
	}
}

// TestInterruptHelperProcess is the child half of
// TestInterruptRunsCleanupBeforeExit; it is inert unless re-executed with the
// marker path in the environment.
func TestInterruptHelperProcess(t *testing.T) {
	marker := os.Getenv(interruptHelperEnv)
	if marker == "" {
		t.Skip("helper process for TestInterruptRunsCleanupBeforeExit")
	}
	onInterrupt(func() { _ = os.WriteFile(marker, []byte("ran"), 0o600) })
	fmt.Println("ready")
	// Bounded so a missed signal fails the parent's assertions promptly rather
	// than running out the test timeout.
	time.Sleep(10 * time.Second) // the signal handler exits the process
}

func TestOnInterruptStopIsIdempotentAndSkipsCleanup(t *testing.T) {
	t.Parallel()
	called := false
	stop := onInterrupt(func() { called = true })
	stop()
	stop()
	if called {
		t.Error("cleanup ran without a signal")
	}
}

func TestCheckoutWorktreeFailsForInvalidRef(t *testing.T) {
	t.Parallel()
	if _, err := checkoutWorktree(t.TempDir(), "role", "definitely-not-a-real-ref-xyz"); err == nil {
		t.Fatal("expected an error for an invalid ref")
	}
}

// TestCheckoutWorktreeIsolatesRolesForSameRef guards the fix that qualifies
// worktree paths by role: base and head must not collide even when both
// point at the same ref. Not parallel — see TestCheckoutAndRemoveWorktree.
func TestCheckoutWorktreeIsolatesRolesForSameRef(t *testing.T) {
	skipWorktreeIntegrationOnWindows(t)
	root := t.TempDir()
	base, err := checkoutWorktree(root, "base", "HEAD")
	if err != nil {
		t.Fatalf("checkoutWorktree(base): %v", err)
	}
	defer removeWorktree(base)

	head, err := checkoutWorktree(root, "head", "HEAD")
	if err != nil {
		t.Fatalf("checkoutWorktree(head): %v", err)
	}
	defer removeWorktree(head)

	if base == head {
		t.Fatalf("expected distinct worktree paths for different roles, got %q twice", base)
	}
}

func TestBuildInterpreterProducesRunnableBinary(t *testing.T) {
	bin := ensureSharedBalBinary(t)
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("stat built interpreter: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("built interpreter %q is not executable", bin)
	}
}

func TestBuildInterpreterFailsForNonModuleDir(t *testing.T) {
	t.Parallel()
	if _, err := buildInterpreter(t.TempDir()); err == nil {
		t.Fatal("expected an error building in a directory with no ./cli/cmd package")
	}
}

func TestPortOpenAndWaitForPort(t *testing.T) {
	t.Parallel()
	port := findFreePort(t)
	if portOpen(port) {
		t.Fatalf("expected port %d to be free", port)
	}
	if waitForPort(port, 100*time.Millisecond) {
		t.Fatalf("waitForPort should time out on a closed port")
	}

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("listen on %d: %v", port, err)
	}
	if !waitForPort(port, 2*time.Second) {
		_ = ln.Close()
		t.Fatalf("waitForPort should observe the now-open port")
	}
	_ = ln.Close()

	waitPortClose(port, 2*time.Second)
	if portOpen(port) {
		t.Fatalf("expected port %d to be closed after waitPortClose", port)
	}
}

func TestReadPeakRSSUnknownPidReturnsZero(t *testing.T) {
	t.Parallel()
	if got := readPeakRSS(-1); got != 0 {
		t.Errorf("readPeakRSS(-1) = %v, want 0", got)
	}
}

func TestReadPeakRSSOwnProcess(t *testing.T) {
	t.Parallel()
	got := readPeakRSS(os.Getpid())
	if runtime.GOOS == "linux" {
		if got <= 0 {
			t.Errorf("readPeakRSS(self) = %v, want > 0 on linux", got)
		}
		return
	}
	if got != 0 {
		t.Errorf("readPeakRSS(self) = %v, want 0 without /proc", got)
	}
}

func TestRunWrkInvalidDuration(t *testing.T) {
	t.Parallel()
	if _, err := runWrk(1, 1, "not-a-duration"); err == nil {
		t.Fatal("expected an error for an invalid wrk duration")
	}
}

// TestMeasureOnceFailsFastWhenPortBusy fails before wrk or bal are ever
// invoked. Not parallel: binds the hardcoded servicePort, which
// TestMeasureOnceProducesSample also uses for a real service.
func TestMeasureOnceFailsFastWhenPortBusy(t *testing.T) {
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(servicePort)))
	if err != nil {
		t.Skipf("port %d unavailable for occupancy test: %v", servicePort, err)
	}
	defer func() { _ = ln.Close() }()

	_, err = measureOnce("irrelevant-binary", "irrelevant.bal", config{warmup: "1s", duration: "1s", conns: 1}, &cleanups{})
	if err == nil {
		t.Fatal("expected an error when the service port is already in use")
	}
}

// TestMeasureOnceProducesSample is the full launch->load->teardown path
// against a real bal build; it requires wrk on PATH — see requireWrk. Not
// parallel — see TestMeasureOnceFailsFastWhenPortBusy.
func TestMeasureOnceProducesSample(t *testing.T) {
	requireWrk(t)

	bal := ensureSharedBalBinary(t)
	helloFile := filepath.Join(t.TempDir(), "hello.bal")
	if err := os.WriteFile(helloFile, helloSource, 0o644); err != nil {
		t.Fatalf("writing hello.bal: %v", err)
	}

	cfg := config{warmup: "1s", duration: "1s", conns: 4}
	s, err := measureOnce(bal, helloFile, cfg, &cleanups{})
	if err != nil {
		t.Fatalf("measureOnce: %v", err)
	}
	if s.throughput <= 0 {
		t.Errorf("expected positive throughput, got %v", s.throughput)
	}
	if s.startupSec <= 0 {
		t.Errorf("expected positive startup time, got %v", s.startupSec)
	}
}
