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

//go:build windows

package palnative

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"ballerina-lang-go/platform/pal"
	"golang.org/x/sys/windows"
)

const windowsCtrlCChildEnv = "PALNATIVE_WINDOWS_CTRL_C_CHILD"

func TestSignalSourceMapsConsoleInterruptToGracefulStop(t *testing.T) {
	if os.Getenv(windowsCtrlCChildEnv) == "1" {
		runWindowsCtrlCChild()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestSignalSourceMapsConsoleInterruptToGracefulStop$")
	cmd.Env = append(os.Environ(), windowsCtrlCChildEnv+"=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("waiting for child readiness: %v", err)
	}
	if line != "READY\n" {
		_ = cmd.Process.Kill()
		t.Fatalf("unexpected child readiness line %q", line)
	}

	if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid)); err != nil {
		_ = cmd.Process.Kill()
		t.Skipf("cannot generate Ctrl+Break console event: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("timed out waiting for child to receive Ctrl+C")
	}
}

func runWindowsCtrlCChild() {
	source, cleanup := newSignalSource()
	defer cleanup()
	fmt.Println("READY")

	select {
	case got, ok := <-source.Signals:
		if !ok {
			fmt.Fprintln(os.Stderr, "signal channel closed before PAL signal")
			os.Exit(1)
		}
		if got != pal.GracefulStop {
			fmt.Fprintf(os.Stderr, "expected %v, got %v\n", pal.GracefulStop, got)
			os.Exit(1)
		}
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "timed out waiting for PAL signal")
		os.Exit(1)
	}
}
