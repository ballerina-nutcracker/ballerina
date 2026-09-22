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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// overlapHelperEnv names the directory the overlap helper writes its two
// cleanup markers into.
const overlapHelperEnv = "HTTPBENCH_OVERLAP_MARKER_DIR"

// TestInterruptOverlappingRegistrationsBothComplete models the live layout of
// this test binary: TestMain keeps one registration alive while run installs a
// second, and a single signal must run both before the process exits.
func TestInterruptOverlappingRegistrationsBothComplete(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sending SIGTERM to a child process is unsupported on windows")
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestOverlapHelperProcess")
	cmd.Env = append(os.Environ(),
		overlapHelperEnv+"="+dir,
		// Keep TestMain's own handler out of the helper: the two registrations
		// under test are installed explicitly below.
		interruptHelperEnv+"="+filepath.Join(dir, "unused"),
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting helper: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	if _, err := bufio.NewReader(stdout).ReadString('\n'); err != nil {
		t.Fatalf("waiting for helper to install its handlers: %v\nhelper stderr:\n%s", err, stderr.String())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signalling helper: %v", err)
	}
	_ = cmd.Wait()

	for _, name := range []string{"outer", "inner"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s cleanup did not complete before the process exited: %v", name, err)
		}
	}
}

// TestOverlapHelperProcess is the child half of
// TestInterruptOverlappingRegistrationsBothComplete; it is inert unless
// re-executed.
func TestOverlapHelperProcess(t *testing.T) {
	dir := os.Getenv(overlapHelperEnv)
	if dir == "" {
		t.Skip("helper process for TestInterruptOverlappingRegistrationsBothComplete")
	}
	// Stands in for TestMain's shared-checkout teardown: a real git worktree
	// removal is slow enough to still be running when a second handler exits.
	onInterrupt(func() {
		time.Sleep(500 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(dir, "outer"), []byte("ran"), 0o600)
	})
	// Stands in for the registration run installs for per-run resources.
	onInterrupt(func() {
		_ = os.WriteFile(filepath.Join(dir, "inner"), []byte("ran"), 0o600)
	})
	fmt.Println("ready")
	time.Sleep(10 * time.Second) // the signal handler exits the process
}
