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
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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
		wantStddev string
		wantWinner string
		wantMetric string
	}{
		{name: "time", mode: timeMode, wantTitle: "Ballerina Benchmark", wantMean: "MEAN (ms)", wantStddev: "STDDEV (ms)", wantWinner: "is faster", wantMetric: "2000.000"},
		{name: "memory", mode: memoryMode, wantTitle: "Ballerina Memory Benchmark", wantMean: "PEAK RSS (MiB)", wantStddev: "STDDEV (MiB)", wantWinner: "uses less memory", wantMetric: "2.000"},
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
			for _, want := range []string{tc.wantTitle, tc.wantMean, tc.wantStddev, tc.wantWinner, tc.wantMetric} {
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

func summaryReport(mode benchmarkMode, results ...runResult) *report {
	return &report{BaseRef: baseRef, HeadRef: headRef, Mode: mode, results: results}
}

func pairRun(label string, baseMean, baseStddev, headMean, headStddev float64) runResult {
	return runResult{label: label, export: benchExport{Results: []benchResult{
		{Command: "base", Mean: baseMean, Stddev: baseStddev},
		{Command: "head", Mean: headMean, Stddev: headStddev},
	}}}
}

// sectionLabels returns the backticked case labels listed under heading, in
// rendered order, so ordering assertions do not depend on bullet wording.
func sectionLabels(t *testing.T, summary, heading string) []string {
	t.Helper()
	var labels []string
	inSection := false
	for _, line := range strings.Split(summary, "\n") {
		if strings.HasPrefix(line, "### ") {
			inSection = line == "### "+heading
			continue
		}
		if !inSection || !strings.HasPrefix(line, "- `") {
			continue
		}
		rest := line[len("- `"):]
		end := strings.Index(rest, "`")
		if end < 0 {
			t.Fatalf("bullet %q has no closing backtick", line)
		}
		labels = append(labels, rest[:end])
	}
	return labels
}

// bulletFor returns the single rendered bullet naming label, so assertions can
// target the bullet text rather than the whole summary, whose closing caveat
// repeats much of the same vocabulary.
func bulletFor(t *testing.T, summary, label string) string {
	t.Helper()
	prefix := "- `" + label + "`"
	var found string
	for _, line := range strings.Split(summary, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		if found != "" {
			t.Fatalf("label %q appears in more than one bullet", label)
		}
		found = line
	}
	if found == "" {
		t.Fatalf("no bullet names %q:\n%s", label, summary)
	}
	return found
}

func TestCombinedSigma(t *testing.T) {
	cases := []struct {
		name         string
		base, head   benchResult
		wantCombined float64
	}{
		{name: "propagates_both_stddevs", base: benchResult{Stddev: 3}, head: benchResult{Stddev: 4}, wantCombined: 5},
		{name: "zero_when_both_zero", base: benchResult{Stddev: 0}, head: benchResult{Stddev: 0}, wantCombined: 0},
		{name: "uses_single_nonzero_side", base: benchResult{Stddev: 0}, head: benchResult{Stddev: 4}, wantCombined: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := combinedSigma(&tc.base, &tc.head); got != tc.wantCombined {
				t.Fatalf("combinedSigma() = %v, want %v", got, tc.wantCombined)
			}
		})
	}
}

func TestCoefficientOfVariation(t *testing.T) {
	cases := []struct {
		name string
		res  *benchResult
		want float64
	}{
		{name: "nil_result", res: nil, want: 0},
		{name: "zero_mean", res: &benchResult{Mean: 0, Stddev: 1}, want: 0},
		{name: "negative_mean", res: &benchResult{Mean: -4, Stddev: 1}, want: 0},
		{name: "ratio_of_stddev_to_mean", res: &benchResult{Mean: 4, Stddev: 1}, want: 0.25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := coefficientOfVariation(tc.res); got != tc.want {
				t.Fatalf("coefficientOfVariation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSanitizeLabel(t *testing.T) {
	cases := []struct {
		name  string
		label string
		want  string
	}{
		{name: "replaces_angle_brackets", label: "a<b>c.bal", want: "a?b?c.bal"},
		{name: "leaves_other_markdown_characters", label: "a&b_c*d.bal", want: "a&b_c*d.bal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeLabel(tc.label); got != tc.want {
				t.Fatalf("sanitizeLabel(%q) = %q, want %q", tc.label, got, tc.want)
			}
		})
	}
}

func TestInfoForMode(t *testing.T) {
	cases := []struct {
		name               string
		mode               benchmarkMode
		wantScale          float64
		wantUnit           string
		wantNoiseThreshold float64
	}{
		{name: "time", mode: timeMode, wantScale: 1000.0, wantUnit: "ms", wantNoiseThreshold: 0.05},
		{name: "memory", mode: memoryMode, wantScale: 1.0, wantUnit: "MiB", wantNoiseThreshold: 0.03},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := infoForMode(tc.mode)
			if info.scale != tc.wantScale {
				t.Fatalf("scale = %v, want %v", info.scale, tc.wantScale)
			}
			if info.unit != tc.wantUnit {
				t.Fatalf("unit = %q, want %q", info.unit, tc.wantUnit)
			}
			if info.noiseThreshold != tc.wantNoiseThreshold {
				t.Fatalf("noiseThreshold = %v, want %v", info.noiseThreshold, tc.wantNoiseThreshold)
			}
		})
	}
}

func TestResultPairRequiresBothSides(t *testing.T) {
	cases := []struct {
		name string
		run  runResult
	}{
		{name: "no_results", run: runResult{label: "case.bal"}},
		{name: "one_result", run: runResult{label: "case.bal", export: benchExport{Results: []benchResult{{Mean: 1}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base, head := resultPair(&tc.run)
			if base != nil || head != nil {
				t.Fatalf("resultPair() = (%v, %v), want (nil, nil)", base, head)
			}
		})
	}
}

func TestResultPairAliasesCallerSlice(t *testing.T) {
	run := pairRun("case.bal", 1, 0, 2, 0)
	base, head := resultPair(&run)
	base.Mean = 10
	head.Mean = 20
	if run.export.Results[0].Mean != 10 || run.export.Results[1].Mean != 20 {
		t.Fatalf("resultPair() returned copies: %+v", run.export.Results)
	}
}

func TestClassifySignificanceBounds(t *testing.T) {
	cases := []struct {
		name     string
		run      runResult
		wantKind entryKind
	}{
		{name: "above_both_bounds", run: pairRun("case.bal", 100, 1, 105, 1), wantKind: kindRegression},
		{name: "exactly_at_both_bounds", run: pairRun("case.bal", 100, 1, 101, 0), wantKind: kindRegression},
		{name: "below_sigma_bound", run: pairRun("case.bal", 100, 2, 101, 0), wantKind: kindNeutral},
		{name: "large_sigma_multiple_sub_one_percent", run: pairRun("case.bal", 100, 0.01, 100.5, 0), wantKind: kindNeutral},
		{name: "large_percentage_under_one_sigma", run: pairRun("case.bal", 100, 100, 150, 0), wantKind: kindNeutral},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := summaryReport(timeMode)
			entry := rep.classify(&tc.run)
			if entry.kind != tc.wantKind {
				t.Fatalf("kind = %v, want %v (entry %+v)", entry.kind, tc.wantKind, entry)
			}
		})
	}
}

func TestClassifyDirectionInBothModes(t *testing.T) {
	cases := []struct {
		name     string
		run      runResult
		wantKind entryKind
	}{
		{name: "slower_head_is_regression", run: pairRun("case.bal", 100, 1, 110, 1), wantKind: kindRegression},
		{name: "faster_head_is_improvement", run: pairRun("case.bal", 100, 1, 90, 1), wantKind: kindImprovement},
	}
	for _, mode := range []benchmarkMode{timeMode, memoryMode} {
		for _, tc := range cases {
			t.Run(string(mode)+"_"+tc.name, func(t *testing.T) {
				entry := summaryReport(mode).classify(&tc.run)
				if entry.kind != tc.wantKind {
					t.Fatalf("kind = %v, want %v", entry.kind, tc.wantKind)
				}
			})
		}
	}
}

func TestClassifyZeroCombinedSigma(t *testing.T) {
	cases := []struct {
		name       string
		mode       benchmarkMode
		run        runResult
		wantInf    bool
		wantSigmas float64
		wantKind   entryKind
	}{
		{name: "zero_delta_scores_zero", mode: timeMode, run: pairRun("case.bal", 100, 0, 100, 0), wantSigmas: 0, wantKind: kindNeutral},
		{name: "significant_delta_scores_infinity", mode: timeMode, run: pairRun("case.bal", 100, 0, 101, 0), wantInf: true, wantKind: kindRegression},
		{name: "sub_one_percent_delta_scores_infinity", mode: timeMode, run: pairRun("case.bal", 100, 0, 100.5, 0), wantInf: true, wantKind: kindNeutral},
		{name: "identical_memory_samples", mode: memoryMode, run: pairRun("case.bal", 2, 0, 2, 0), wantSigmas: 0, wantKind: kindNeutral},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := summaryReport(tc.mode).classify(&tc.run)
			if tc.wantInf {
				if !math.IsInf(entry.sigmas, 1) {
					t.Fatalf("sigmas = %v, want +Inf", entry.sigmas)
				}
			} else if entry.sigmas != tc.wantSigmas {
				t.Fatalf("sigmas = %v, want %v", entry.sigmas, tc.wantSigmas)
			}
			if entry.kind != tc.wantKind {
				t.Fatalf("kind = %v, want %v", entry.kind, tc.wantKind)
			}
		})
	}
}

func TestSummarizeNeverProducesNaNSigmas(t *testing.T) {
	var results []runResult
	stddevs := []float64{0, 1}
	heads := []float64{100, 101, 99}
	for _, baseStddev := range stddevs {
		for _, headStddev := range stddevs {
			for _, head := range heads {
				label := fmt.Sprintf("b%v-h%v-m%v.bal", baseStddev, headStddev, head)
				results = append(results, pairRun(label, 100, baseStddev, head, headStddev))
			}
		}
	}
	for _, mode := range []benchmarkMode{timeMode, memoryMode} {
		for _, entry := range summaryReport(mode, results...).summarize() {
			if math.IsNaN(entry.sigmas) {
				t.Fatalf("%s mode produced NaN sigmas for %q", mode, entry.label)
			}
		}
	}
}

func TestClassifyNoiseDetection(t *testing.T) {
	cases := []struct {
		name      string
		mode      benchmarkMode
		run       runResult
		wantNoisy bool
	}{
		{name: "time_noisy_base_only", mode: timeMode, run: pairRun("case.bal", 100, 6, 100, 0), wantNoisy: true},
		{name: "time_noisy_head_only", mode: timeMode, run: pairRun("case.bal", 100, 0, 100, 6), wantNoisy: true},
		{name: "time_noisy_both_sides", mode: timeMode, run: pairRun("case.bal", 100, 6, 100, 6), wantNoisy: true},
		{name: "time_exactly_at_threshold", mode: timeMode, run: pairRun("case.bal", 100, 5, 100, 0), wantNoisy: true},
		{name: "time_below_threshold", mode: timeMode, run: pairRun("case.bal", 100, 1, 100, 1), wantNoisy: false},
		{name: "memory_noisy_base_only", mode: memoryMode, run: pairRun("case.bal", 100, 4, 100, 0), wantNoisy: true},
		{name: "memory_noisy_head_only", mode: memoryMode, run: pairRun("case.bal", 100, 0, 100, 4), wantNoisy: true},
		{name: "memory_noisy_both_sides", mode: memoryMode, run: pairRun("case.bal", 100, 4, 100, 4), wantNoisy: true},
		{name: "memory_exactly_at_threshold", mode: memoryMode, run: pairRun("case.bal", 100, 3, 100, 0), wantNoisy: true},
		{name: "memory_below_threshold", mode: memoryMode, run: pairRun("case.bal", 100, 2, 100, 2), wantNoisy: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := summaryReport(tc.mode).classify(&tc.run)
			if entry.noisy != tc.wantNoisy {
				t.Fatalf("noisy = %v, want %v (maxCV %v)", entry.noisy, tc.wantNoisy, entry.maxCV)
			}
		})
	}
}

func TestClassifyUnavailableCases(t *testing.T) {
	cases := []struct {
		name string
		run  runResult
	}{
		{name: "no_results", run: runResult{label: "case.bal"}},
		{name: "single_result", run: runResult{label: "case.bal", export: benchExport{Results: []benchResult{{Mean: 1, Stddev: 1}}}}},
		{name: "zero_base_mean", run: pairRun("case.bal", 0, 1, 100, 100)},
		{name: "zero_head_mean", run: pairRun("case.bal", 100, 100, 0, 1)},
		{name: "negative_base_mean", run: pairRun("case.bal", -1, 1, 100, 100)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := summaryReport(timeMode).classify(&tc.run)
			if entry.kind != kindUnavailable {
				t.Fatalf("kind = %v, want kindUnavailable", entry.kind)
			}
			if entry.noisy {
				t.Fatal("unavailable entry must not be noisy")
			}
		})
	}
}

func TestRenderSummaryKeepsNoisySignificantCaseOutOfUnreliableSection(t *testing.T) {
	rep := summaryReport(timeMode, pairRun("case.bal", 100, 10, 130, 10))
	summary := rep.renderSummary()
	if got := sectionLabels(t, summary, "Regressions"); len(got) != 1 || got[0] != "case.bal" {
		t.Fatalf("Regressions section = %v, want [case.bal]", got)
	}
	bullet := bulletFor(t, summary, "case.bal")
	if !strings.Contains(bullet, "— unreliable: worst CV 10.0%") {
		t.Fatalf("regression bullet is missing the inline noise caveat: %q", bullet)
	}
	if strings.Contains(summary, "### Unreliable measurements") {
		t.Fatalf("noisy significant case was repeated under Unreliable measurements:\n%s", summary)
	}
}

func TestRenderSummaryNamesWorstCoefficientOfVariationForUnreliableCases(t *testing.T) {
	rep := summaryReport(timeMode, pairRun("case.bal", 100, 8, 100, 12))
	bullet := bulletFor(t, rep.renderSummary(), "case.bal")
	if !strings.Contains(bullet, "worst CV 12.0%") {
		t.Fatalf("unreliable bullet does not name the worst coefficient of variation: %q", bullet)
	}
}

func TestRenderSummaryOrdersRegressionsBySigmaThenPercentageThenLabel(t *testing.T) {
	rep := summaryReport(timeMode,
		pairRun("e-tie2.bal", 100, 5, 110, 5),
		pairRun("c-bigpct.bal", 50, 5, 60, 5),
		pairRun("a-inf.bal", 100, 0, 110, 0),
		pairRun("d-tie1.bal", 100, 5, 110, 5),
		pairRun("b-high.bal", 100, 1, 120, 1),
	)
	want := []string{"a-inf.bal", "b-high.bal", "c-bigpct.bal", "d-tie1.bal", "e-tie2.bal"}
	if got := sectionLabels(t, rep.renderSummary(), "Regressions"); !slices.Equal(got, want) {
		t.Fatalf("Regressions order = %v, want %v", got, want)
	}
}

func TestRenderSummaryOrdersImprovementsBySigmaThenPercentageThenLabel(t *testing.T) {
	rep := summaryReport(timeMode,
		pairRun("e-tie2.bal", 100, 5, 90, 5),
		pairRun("c-bigpct.bal", 50, 5, 40, 5),
		pairRun("a-inf.bal", 100, 0, 90, 0),
		pairRun("d-tie1.bal", 100, 5, 90, 5),
		pairRun("b-high.bal", 100, 1, 80, 1),
	)
	want := []string{"a-inf.bal", "b-high.bal", "c-bigpct.bal", "d-tie1.bal", "e-tie2.bal"}
	if got := sectionLabels(t, rep.renderSummary(), "Improvements"); !slices.Equal(got, want) {
		t.Fatalf("Improvements order = %v, want %v", got, want)
	}
}

func TestRenderSummaryOrdersUnreliableByCoefficientOfVariationAndCaps(t *testing.T) {
	results := []runResult{
		pairRun("tie-b.bal", 100, 20, 100, 0),
		pairRun("tie-a.bal", 100, 20, 100, 0),
	}
	for i := 0; i < 12; i++ {
		results = append(results, pairRun(fmt.Sprintf("case%02d.bal", i), 100, 20-float64(i), 100, 0))
	}
	summary := summaryReport(timeMode, results...).renderSummary()
	got := sectionLabels(t, summary, "Unreliable measurements")
	want := []string{"case00.bal", "tie-a.bal", "tie-b.bal", "case01.bal", "case02.bal",
		"case03.bal", "case04.bal", "case05.bal", "case06.bal", "case07.bal"}
	if !slices.Equal(got, want) {
		t.Fatalf("Unreliable measurements order = %v, want %v", got, want)
	}
	if !strings.Contains(summary, "…and 4 more (see the table below)") {
		t.Fatalf("summary is missing the truncation line:\n%s", summary)
	}
	if strings.Contains(summary, "case11.bal") {
		t.Fatalf("lowest-variance case survived the cap:\n%s", summary)
	}
}

func TestRenderSummaryOrdersNotMeasuredByLabel(t *testing.T) {
	rep := summaryReport(timeMode,
		runResult{label: "c.bal"},
		runResult{label: "a.bal"},
		runResult{label: "b.bal"},
	)
	want := []string{"a.bal", "b.bal", "c.bal"}
	if got := sectionLabels(t, rep.renderSummary(), "Not measured"); !slices.Equal(got, want) {
		t.Fatalf("Not measured order = %v, want %v", got, want)
	}
}

func TestRenderSummaryIsIndependentOfInputOrder(t *testing.T) {
	results := []runResult{
		pairRun("v.bal", 100, 10, 100, 10),
		pairRun("w.bal", 100, 10, 100, 10),
		pairRun("x.bal", 100, 10, 100, 10),
		pairRun("y.bal", 100, 10, 100, 10),
		pairRun("z.bal", 100, 10, 100, 10),
		pairRun("u.bal", 100, 0, 130, 0),
		{label: "t.bal"},
		{label: "s.bal"},
	}
	reversed := make([]runResult, 0, len(results))
	for i := len(results) - 1; i >= 0; i-- {
		reversed = append(reversed, results[i])
	}
	forward := summaryReport(timeMode, results...).renderSummary()
	backward := summaryReport(timeMode, reversed...).renderSummary()
	if forward != backward {
		t.Fatalf("renderSummary() depends on input order:\n%s\n---\n%s", forward, backward)
	}
}

func TestRenderSummaryOnCleanReportStatesNoCaseMoved(t *testing.T) {
	rep := summaryReport(memoryMode, pairRun("case.bal", 2, 0, 2, 0))
	summary := rep.renderSummary()
	if !strings.Contains(summary, "No case beyond 1 sigma and 1%.") {
		t.Fatalf("summary is missing the all-clean line:\n%s", summary)
	}
	if strings.Contains(summary, "###") {
		t.Fatalf("summary emitted a section heading for an all-clean report:\n%s", summary)
	}
}

func TestRenderSummaryFormatsMagnitudesThroughFormatMetric(t *testing.T) {
	cases := []struct {
		name     string
		mode     benchmarkMode
		run      runResult
		wantBase string
		wantHead string
	}{
		{name: "time_scales_seconds_to_milliseconds", mode: timeMode, run: pairRun("case.bal", 0.002, 0, 0.0025, 0),
			wantBase: "2.000 ms", wantHead: "2.500 ms"},
		{name: "memory_keeps_mebibytes", mode: memoryMode, run: pairRun("case.bal", 2.0, 0, 2.5, 0),
			wantBase: "2.000 MiB", wantHead: "2.500 MiB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary := summaryReport(tc.mode, tc.run).renderSummary()
			for _, want := range []string{tc.wantBase, tc.wantHead} {
				if !strings.Contains(summary, want) {
					t.Fatalf("summary does not contain %q:\n%s", want, summary)
				}
			}
		})
	}
}

func TestRenderSummaryRendersInfiniteSigmaAsZeroStddev(t *testing.T) {
	summary := summaryReport(timeMode, pairRun("case.bal", 100, 0, 130, 0)).renderSummary()
	if !strings.Contains(summary, "stddev 0") {
		t.Fatalf("summary does not report a zero standard deviation:\n%s", summary)
	}
	if strings.Contains(summary, "Inf") {
		t.Fatalf("summary leaked an infinite sigma multiple:\n%s", summary)
	}
}

func TestRenderSummaryWrapsLabelsInBackticks(t *testing.T) {
	summary := summaryReport(timeMode, pairRun("my_case*x.bal", 100, 0, 130, 0)).renderSummary()
	if !strings.Contains(summary, "`my_case*x.bal`") {
		t.Fatalf("summary does not wrap the label in backticks:\n%s", summary)
	}
}

func TestRenderSummaryEmitsNoAngleBrackets(t *testing.T) {
	summary := summaryReport(timeMode, pairRun("a<b>c.bal", 100, 0, 130, 0)).renderSummary()
	if strings.ContainsAny(summary, "<>") {
		t.Fatalf("summary emitted an angle bracket:\n%s", summary)
	}
}

func TestRenderSummaryCapsSectionsAtTenEntries(t *testing.T) {
	cases := []struct {
		name         string
		count        int
		wantTruncate string
	}{
		{name: "exactly_ten_entries_are_not_truncated", count: 10},
		{name: "eleven_entries_report_one_more", count: 11, wantTruncate: "…and 1 more (see the table below)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := make([]runResult, 0, tc.count)
			for i := 0; i < tc.count; i++ {
				results = append(results, pairRun(fmt.Sprintf("case%02d.bal", i), 100, 0, 110+float64(i), 0))
			}
			summary := summaryReport(timeMode, results...).renderSummary()
			if got := len(sectionLabels(t, summary, "Regressions")); got != maxSummaryEntriesPerSection {
				t.Fatalf("Regressions listed %d entries, want %d", got, maxSummaryEntriesPerSection)
			}
			if tc.wantTruncate == "" {
				if strings.Contains(summary, "…and") {
					t.Fatalf("summary truncated a full but uncapped section:\n%s", summary)
				}
				return
			}
			if !strings.Contains(summary, tc.wantTruncate) {
				t.Fatalf("summary does not contain %q:\n%s", tc.wantTruncate, summary)
			}
		})
	}
}

func TestSummaryDirectionAgreesWithComputeDelta(t *testing.T) {
	rep := summaryReport(timeMode,
		pairRun("slower.bal", 100, 0, 130, 0),
		pairRun("faster.bal", 100, 0, 70, 0),
		pairRun("flat.bal", 100, 0, 100, 0),
		runResult{label: "missing.bal"},
	)
	for i, entry := range rep.summarize() {
		base, head := resultPair(&rep.results[i])
		_, _, _, winnerRef := computeDelta(base, head, rep.BaseRef, rep.HeadRef)
		switch entry.kind {
		case kindRegression:
			if winnerRef != rep.BaseRef {
				t.Fatalf("%q is a regression but computeDelta named %q the winner", entry.label, winnerRef)
			}
		case kindImprovement:
			if winnerRef != rep.HeadRef {
				t.Fatalf("%q is an improvement but computeDelta named %q the winner", entry.label, winnerRef)
			}
		}
	}
}

func TestExportSummaryWritesRenderedMarkdown(t *testing.T) {
	rep := summaryReport(timeMode, pairRun("case.bal", 100, 0, 130, 0))
	outPath := filepath.Join(t.TempDir(), "summary.md")
	if err := rep.exportSummary(outPath); err != nil {
		t.Fatalf("exportSummary() returned error: %v", err)
	}
	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != rep.renderSummary() {
		t.Fatalf("exportSummary() wrote:\n%s\nwant:\n%s", written, rep.renderSummary())
	}
}

func TestExportSummaryReportsUnwritableDestination(t *testing.T) {
	rep := summaryReport(timeMode, pairRun("case.bal", 100, 0, 130, 0))
	outPath := filepath.Join(t.TempDir(), "missing-dir", "summary.md")
	err := rep.exportSummary(outPath)
	if err == nil || !strings.Contains(err.Error(), outPath) {
		t.Fatalf("exportSummary() error = %v, want an error naming %q", err, outPath)
	}
}

func TestWriteReportsHonoursEachExportIndependently(t *testing.T) {
	rep := summaryReport(timeMode, pairRun("case.bal", 100, 0, 130, 0))
	cases := []struct {
		name        string
		withHTML    bool
		withSummary bool
	}{
		{name: "html_only", withHTML: true},
		{name: "summary_only", withSummary: true},
		{name: "both", withHTML: true, withSummary: true},
		{name: "neither"},
	}
	var htmlOnly, summaryOnly []byte
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			htmlPath := filepath.Join(dir, "report.html")
			summaryPath := filepath.Join(dir, "summary.md")
			var htmlArg, summaryArg string
			if tc.withHTML {
				htmlArg = htmlPath
			}
			if tc.withSummary {
				summaryArg = summaryPath
			}
			if err := writeReports(rep, htmlArg, summaryArg); err != nil {
				t.Fatalf("writeReports() returned error: %v", err)
			}
			assertFileExists(t, htmlPath, tc.withHTML)
			assertFileExists(t, summaryPath, tc.withSummary)
			if tc.name == "html_only" {
				htmlOnly = readFileOrFail(t, htmlPath)
			}
			if tc.name == "summary_only" {
				summaryOnly = readFileOrFail(t, summaryPath)
			}
			if tc.name == "both" {
				if !bytes.Equal(readFileOrFail(t, summaryPath), summaryOnly) {
					t.Fatal("summary written with both flags differs from the summary-only result")
				}
				if !bytes.Equal(readFileOrFail(t, htmlPath), htmlOnly) {
					t.Fatal("html written with both flags differs from the html-only result")
				}
			}
		})
	}
}

func assertFileExists(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	if want && err != nil {
		t.Fatalf("expected %q to exist: %v", path, err)
	}
	if !want && err == nil {
		t.Fatalf("expected %q not to be created", path)
	}
}

func readFileOrFail(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
