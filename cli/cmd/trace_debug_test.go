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

//go:build debug

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// TestFinalizeWritesExactlyOnce covers the driver-owned output guard: a second
// finalization returns the first result without rewriting.
func TestFinalizeWritesExactlyOnce(t *testing.T) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), context.TraceOptions{Enabled: true})
	tracePath := filepath.Join(t.TempDir(), "traces.json")
	output := newRunTraceOutput(tracePath)

	if err := output.finalize(env); err != nil {
		t.Fatalf("first finalize: %v", err)
	}
	written, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(tracePath); err != nil {
		t.Fatal(err)
	}
	if err := output.finalize(env); err != nil {
		t.Fatalf("second finalize: %v", err)
	}
	if _, err := os.Stat(tracePath); !os.IsNotExist(err) {
		t.Fatalf("second finalize rewrote the trace, stat err = %v", err)
	}
	if string(written) == "" {
		t.Fatal("first finalize wrote an empty file")
	}
}

// TestFinalizeReportsAFailureOnlyToTheAttemptingCaller covers repeated
// finalization after a failure: the attempt is not retried, and only the caller
// that made it is told it failed.
func TestFinalizeReportsAFailureOnlyToTheAttemptingCaller(t *testing.T) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), context.TraceOptions{Enabled: true})
	blocked := filepath.Join(t.TempDir(), "traces.json")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	output := newRunTraceOutput(blocked)

	if err := output.finalize(env); err == nil {
		t.Fatal("expected the write to a directory to fail")
	}

	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	if err := output.finalize(env); err != nil {
		t.Fatalf("second finalize err = %v, want nil", err)
	}
	if _, err := os.Stat(blocked); !os.IsNotExist(err) {
		t.Fatalf("second finalize retried the write, stat err = %v", err)
	}
}

// TestFinalizeWithoutTracePathDoesNothing covers a debug run with no --trace.
func TestFinalizeWithoutTracePathDoesNothing(t *testing.T) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), context.TraceOptions{})
	workDir := t.TempDir()
	t.Chdir(workDir)

	if err := newRunTraceOutput("").finalize(env); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("finalize wrote %d files without a trace path", len(entries))
	}
}

// TestReportFailureDuringPanicPreservesThePanicAndTheStack covers a panic
// unwinding through a traced run: the partial recording is written, the
// panicking frames are reported before they are lost, and the original panic
// continues.
func TestReportFailureDuringPanicPreservesThePanicAndTheStack(t *testing.T) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), context.TraceOptions{Enabled: true})
	tracePath := filepath.Join(t.TempDir(), "traces.json")
	output := newRunTraceOutput(tracePath)

	stderr := panicThroughTraceOutput(t, output, env)

	if data, err := os.ReadFile(tracePath); err != nil || len(data) == 0 {
		t.Fatalf("partial trace not written: data=%q err=%v", data, err)
	}
	if !strings.Contains(stderr, "panic: boom") {
		t.Errorf("stderr does not report the panic value:\n%s", stderr)
	}
	if !strings.Contains(stderr, "panicThroughTraceOutput") {
		t.Errorf("stderr does not report the panicking frames:\n%s", stderr)
	}
}

// TestReportFailureDuringPanicReportsAWriteFailure covers a write failure while
// a panic unwinds: it is reported without suppressing the panic.
func TestReportFailureDuringPanicReportsAWriteFailure(t *testing.T) {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), context.TraceOptions{Enabled: true})
	blocked := filepath.Join(t.TempDir(), "traces.json")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}

	stderr := panicThroughTraceOutput(t, newRunTraceOutput(blocked), env)

	if !strings.Contains(stderr, "write trace") {
		t.Errorf("stderr does not report the write failure:\n%s", stderr)
	}
	if !strings.Contains(stderr, "panic: boom") {
		t.Errorf("stderr does not report the panic value:\n%s", stderr)
	}
}

// panicThroughTraceOutput panics through output's deferred handler, asserts the
// original panic reaches the caller, and returns what the handler wrote to
// stderr.
func panicThroughTraceOutput(
	t *testing.T,
	output *runTraceOutput,
	env *context.CompilerEnvironment,
) string {
	t.Helper()
	captured := filepath.Join(t.TempDir(), "stderr.txt")
	file, err := os.Create(captured)
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stderr
	os.Stderr = file
	defer func() {
		os.Stderr = original
		_ = file.Close()
	}()

	func() {
		defer func() {
			if recovered := recover(); recovered != "boom" {
				t.Errorf("recovered = %v, want the original panic to continue", recovered)
			}
		}()
		defer output.reportFailureDuringPanic(env)
		panic("boom")
	}()

	data, err := os.ReadFile(captured)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
