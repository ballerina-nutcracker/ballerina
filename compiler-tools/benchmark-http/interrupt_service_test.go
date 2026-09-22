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

//go:build unix

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// serviceLeakHelperEnv names the file the helper writes the service pid to.
const serviceLeakHelperEnv = "HTTPBENCH_SERVICE_LEAK_PID_FILE"

// TestInterruptStopsService re-executes this binary as a helper that mirrors
// measureOnce: a service started in its own process group with its teardown
// registered on the cleanup stack, plus the interrupt handler installed by run.
// Signalling the helper must not strand the service.
func TestInterruptStopsService(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "service.pid")
	cmd := exec.Command(os.Args[0], "-test.run=TestServiceLeakHelperProcess")
	cmd.Env = append(os.Environ(),
		serviceLeakHelperEnv+"="+pidFile,
		interruptHelperEnv+"="+filepath.Join(t.TempDir(), "unused"),
	)
	// Own process group: the signal below must reach only the helper, not this
	// test binary.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting helper: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	if _, err := bufio.NewReader(stdout).ReadString('\n'); err != nil {
		t.Fatalf("waiting for helper to start its service: %v", err)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signalling helper: %v", err)
	}
	_ = cmd.Wait()

	pid := readServicePid(t, pidFile)
	defer func() { _ = syscall.Kill(-pid, syscall.SIGKILL) }()
	// Give a correct teardown time to land before concluding the service leaked.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("service pid %d still running after the benchmark was interrupted", pid)
}

func readServicePid(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading service pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parsing service pid %q: %v", data, err)
	}
	return pid
}

// TestServiceLeakHelperProcess is the child half of TestInterruptStopsService;
// it is inert unless re-executed.
func TestServiceLeakHelperProcess(t *testing.T) {
	pidFile := os.Getenv(serviceLeakHelperEnv)
	if pidFile == "" {
		t.Skip("helper process for TestInterruptStopsService")
	}
	var c cleanups
	defer c.run()
	defer onInterrupt(c.run)()

	service := exec.Command("sleep", "120")
	setProcAttr(service)
	if err := service.Start(); err != nil {
		t.Fatalf("starting service: %v", err)
	}
	defer c.addScoped(func() { killGroup(service) })()

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(service.Process.Pid)), 0o600); err != nil {
		t.Fatalf("writing service pid: %v", err)
	}
	fmt.Println("ready")
	time.Sleep(30 * time.Second)
}
