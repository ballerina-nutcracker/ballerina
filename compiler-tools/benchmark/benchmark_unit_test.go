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
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestConfigValidateMemoryMode(t *testing.T) {
	target := writeTempBalFile(t)
	cases := []struct {
		name    string
		cfg     config
		wantErr string
	}{
		{
			name: "allows_warmup_and_runs",
			cfg:  config{baseRef: baseRef, headRef: headRef, target: target, mode: memoryMode, warmup: 4, runs: 10},
		},
		{
			name:    "rejects_negative_warmup",
			cfg:     config{baseRef: baseRef, headRef: headRef, target: target, mode: memoryMode, warmup: -1, runs: 1},
			wantErr: "warmup must be non-negative",
		},
		{
			name:    "rejects_zero_runs",
			cfg:     config{baseRef: baseRef, headRef: headRef, target: target, mode: memoryMode, warmup: 0, runs: 0},
			wantErr: "runs must be greater than zero",
		},
		{
			name:    "rejects_invalid_mode",
			cfg:     config{baseRef: baseRef, headRef: headRef, target: target, mode: benchmarkMode("cpu")},
			wantErr: "mode must be one of",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validate() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestReportModeSpecificRendering(t *testing.T) {
	cases := []struct {
		name       string
		mode       benchmarkMode
		wantTitle  string
		wantMean   string
		wantWinner string
		wantMetric string
	}{
		{name: "time", mode: timeMode, wantTitle: "Ballerina Benchmark", wantMean: "MEAN (ms)", wantWinner: "is faster", wantMetric: "2000.000"},
		{name: "memory", mode: memoryMode, wantTitle: "Ballerina Memory Benchmark", wantMean: "PEAK RSS (MiB)", wantWinner: "uses less memory", wantMetric: "2.000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := &report{
				BaseRef: baseRef,
				HeadRef: headRef,
				Mode:    tc.mode,
				results: []runResult{{
					label: "case.bal",
					export: benchExport{Results: []benchResult{
						{Command: "base", Mean: 2, Stddev: 0.25},
						{Command: "head", Mean: 1, Stddev: 0.10},
					}},
				}},
			}
			outPath := filepath.Join(t.TempDir(), "report.html")
			if err := rep.export(outPath); err != nil {
				t.Fatalf("export() returned error: %v", err)
			}
			html, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
			text := string(html)
			for _, want := range []string{tc.wantTitle, tc.wantMean, tc.wantWinner, tc.wantMetric} {
				if !strings.Contains(text, want) {
					t.Fatalf("report did not contain %q", want)
				}
			}
		})
	}
}

func TestParseMaxRSSMiBRejectsMissingMetric(t *testing.T) {
	if _, err := parseMaxRSSMiB("elapsed time: 1s"); err == nil {
		t.Fatal("expected parseMaxRSSMiB() to reject output without max RSS")
	}
}

func TestRequireMemoryTool(t *testing.T) {
	tool, err := currentMemoryTool()
	if err != nil {
		t.Skip(err)
	}
	if _, err := os.Stat(tool.path); err != nil {
		t.Skipf("%s is unavailable: %v", tool.path, err)
	}
	if err := requireMemoryTool(); err != nil {
		t.Fatalf("requireMemoryTool() returned error: %v", err)
	}
}

func TestRunRequiresHyperfineInTimeMode(t *testing.T) {
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	b := &benchmark{config: config{mode: timeMode, target: "missing.bal"}}
	if err := b.run(); err == nil || !strings.Contains(err.Error(), "hyperfine is required") {
		t.Fatalf("run() error = %v, want hyperfine lookup failure", err)
	}
}

func TestRunMemoryModeValidatesMemoryToolBeforeResolvingTarget(t *testing.T) {
	tool, err := currentMemoryTool()
	if err != nil {
		t.Skip(err)
	}
	if _, err := os.Stat(tool.path); err != nil {
		t.Skipf("%s is unavailable: %v", tool.path, err)
	}
	b := &benchmark{config: config{mode: memoryMode, target: "missing.bal"}}
	if err := b.run(); err == nil || !strings.Contains(err.Error(), "failed to resolve benchmark target") {
		t.Fatalf("run() error = %v, want target resolution failure", err)
	}
}

func TestRunMemoryBenchmarkUsesTimeOutput(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)

	restore := stubExecCommand(t, []memoryCommandStub{{stderr: memoryCommandOutputForMiB(2)}})
	defer restore()

	b := &benchmark{config: config{baseRef: baseRef, headRef: headRef, runs: 1}}
	export, err := b.runMemoryBenchmark("/base", "/head", "case.bal", "bal")
	if err != nil {
		t.Fatalf("runMemoryBenchmark() returned error: %v", err)
	}
	if len(export.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(export.Results))
	}
	for _, result := range export.Results {
		if result.Mean != 2 || result.Median != 2 || result.Stddev != 0 {
			t.Fatalf("unexpected memory result: %+v", result)
		}
		if !strings.Contains(result.Command, "case.bal") {
			t.Fatalf("command %q does not include target", result.Command)
		}
	}
}

func TestRunMemoryCommandUsesWarmupAndRuns(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)

	restore := stubExecCommand(t, []memoryCommandStub{
		{stderr: memoryCommandOutputForMiB(100)},
		{stderr: memoryCommandOutputForMiB(1)},
		{stderr: memoryCommandOutputForMiB(3)},
	})
	defer restore()

	b := &benchmark{config: config{warmup: 1, runs: 2}}
	result, err := b.runMemoryCommand("bal", "case.bal")
	if err != nil {
		t.Fatalf("runMemoryCommand() returned error: %v", err)
	}
	if result.Mean != 2 || result.Median != 2 || result.Stddev != math.Sqrt(2) {
		t.Fatalf("unexpected memory result: %+v", result)
	}
}

func TestRunMemoryBenchmarkReportsCommandFailure(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)

	restore := stubExecCommand(t, []memoryCommandStub{{stderr: "boom\n", exitCode: 1}})
	defer restore()

	b := &benchmark{config: config{baseRef: baseRef, headRef: headRef, runs: 1}}
	_, err := b.runMemoryBenchmark("/base", "/head", "case.bal", "bal")
	if err == nil || !strings.Contains(err.Error(), "failed to run memory benchmark for "+baseRef) {
		t.Fatalf("runMemoryBenchmark() error = %v", err)
	}
}

func TestRunMemoryBenchmarkReportsHeadFailure(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)

	restore := stubExecCommand(t, []memoryCommandStub{
		{stderr: memoryCommandOutputForMiB(2)},
		{stderr: "boom\n", exitCode: 1},
	})
	defer restore()

	b := &benchmark{config: config{baseRef: baseRef, headRef: headRef, runs: 1}}
	_, err := b.runMemoryBenchmark("/base", "/head", "case.bal", "bal")
	if err == nil || !strings.Contains(err.Error(), "failed to run memory benchmark for "+headRef) {
		t.Fatalf("runMemoryBenchmark() error = %v", err)
	}
}

func TestRunMemoryCommandReportsParseFailure(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)

	restore := stubExecCommand(t, []memoryCommandStub{{stderr: "no rss here\n"}})
	defer restore()

	b := &benchmark{config: config{runs: 1}}
	_, err := b.runMemoryCommand("bal", "case.bal")
	if err == nil || !strings.Contains(err.Error(), "failed to parse maximum resident set size") {
		t.Fatalf("runMemoryCommand() error = %v", err)
	}
}

func TestRunMemorySampleTimesOut(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)

	originalTimeout := memorySampleTimeout
	memorySampleTimeout = 10 * time.Millisecond
	defer func() { memorySampleTimeout = originalTimeout }()

	original := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmdArgs := []string{"-test.run=TestMemoryCommandHelper", "--", name}
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_MEMORY_COMMAND_HELPER=1",
			"GO_MEMORY_COMMAND_SLEEP=200ms",
		)
		return cmd
	}
	defer func() { execCommandContext = original }()

	_, err := runMemorySample("bal", "case.bal")
	if err == nil || !strings.Contains(err.Error(), "command timed out after") {
		t.Fatalf("runMemorySample() error = %v", err)
	}
}

func TestRunBenchmarksTimeModeUsesHyperfineExport(t *testing.T) {
	skipHyperfineStubUnsupported(t)

	tmp := t.TempDir()
	writeFakeHyperfine(t, tmp)
	prependPath(t, tmp)

	targetPath := filepath.Join("cases", "1-v.bal")
	target := &benchmarkTarget{
		mode:  multipleFilesMode,
		label: "cases",
		root:  "cases",
		paths: []string{targetPath},
	}
	b := &benchmark{config: config{baseRef: baseRef, headRef: headRef, mode: timeMode, runs: 1}}
	results, err := b.runBenchmarks("/base", "/head", target, "bal", t.TempDir())
	if err != nil {
		t.Fatalf("runBenchmarks() returned error: %v", err)
	}
	if len(results) != 1 || results[0].label != targetPath {
		t.Fatalf("unexpected results: %+v", results)
	}
	if got := results[0].export.Results[0].Mean; got != 0.5 {
		t.Fatalf("mean = %v, want 0.5", got)
	}
}

func TestRunBenchmarksTimeModeReturnsHyperfineError(t *testing.T) {
	skipHyperfineStubUnsupported(t)

	tmp := t.TempDir()
	path := filepath.Join(tmp, "hyperfine")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prependPath(t, tmp)

	target := &benchmarkTarget{mode: singleFileMode, label: "case.bal", paths: []string{"case.bal"}}
	b := &benchmark{config: config{baseRef: baseRef, headRef: headRef, mode: timeMode, runs: 1}}
	_, err := b.runBenchmarks("/base", "/head", target, "bal", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "failed to run hyperfine") {
		t.Fatalf("runBenchmarks() error = %v", err)
	}
}

func TestRunBenchmarksMemoryMode(t *testing.T) {
	skipMemoryBenchmarkUnsupported(t)

	restore := stubExecCommand(t, []memoryCommandStub{{stderr: memoryCommandOutputForMiB(1)}})
	defer restore()

	target := &benchmarkTarget{mode: singleFileMode, label: "case.bal", paths: []string{"case.bal"}}
	b := &benchmark{config: config{baseRef: baseRef, headRef: headRef, mode: memoryMode, runs: 1}}
	results, err := b.runBenchmarks("/base", "/head", target, "bal", t.TempDir())
	if err != nil {
		t.Fatalf("runBenchmarks() returned error: %v", err)
	}
	if len(results) != 1 || results[0].label != "case.bal" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if got := results[0].export.Results[0].Mean; got != 1 {
		t.Fatalf("mean = %v, want 1", got)
	}
}

func skipMemoryBenchmarkUnsupported(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("memory benchmark mode is not supported on Windows")
	}
}

func skipHyperfineStubUnsupported(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("hyperfine stub is not supported on Windows")
	}
}

func writeTempBalFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.bal")
	if err := os.WriteFile(path, []byte("public function main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeHyperfine(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "hyperfine")
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--export-json" ]; then
    shift
    printf '{"results":[{"command":"base","mean":0.5,"stddev":0.01,"median":0.5},{"command":"head","mean":0.4,"stddev":0.02,"median":0.4}]}' > "$1"
    exit 0
  fi
  shift
done
exit 1
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func prependPath(t *testing.T, dir string) {
	t.Helper()
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
}

func memoryCommandOutputForMiB(mib int) string {
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("%d  maximum resident set size\n", mib*1024*1024)
	}
	return fmt.Sprintf("Maximum resident set size (kbytes): %d\n", mib*1024)
}

type memoryCommandStub struct {
	stderr   string
	exitCode int
}

func stubExecCommand(t *testing.T, stubs []memoryCommandStub) func() {
	t.Helper()
	original := execCommandContext
	call := 0
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		stub := stubs[len(stubs)-1]
		if call < len(stubs) {
			stub = stubs[call]
		}
		call++
		cmdArgs := append([]string{"-test.run=TestMemoryCommandHelper", "--", name}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_MEMORY_COMMAND_HELPER=1",
			"GO_MEMORY_COMMAND_STDERR="+stub.stderr,
			fmt.Sprintf("GO_MEMORY_COMMAND_EXIT=%d", stub.exitCode),
		)
		return cmd
	}
	return func() { execCommandContext = original }
}

func TestMemoryCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_MEMORY_COMMAND_HELPER") != "1" {
		return
	}
	if sleep := os.Getenv("GO_MEMORY_COMMAND_SLEEP"); sleep != "" {
		duration, err := time.ParseDuration(sleep)
		if err != nil {
			panic(err)
		}
		time.Sleep(duration)
	}
	_, _ = fmt.Fprint(os.Stderr, os.Getenv("GO_MEMORY_COMMAND_STDERR"))
	if os.Getenv("GO_MEMORY_COMMAND_EXIT") != "0" {
		os.Exit(1)
	}
	os.Exit(0)
}
