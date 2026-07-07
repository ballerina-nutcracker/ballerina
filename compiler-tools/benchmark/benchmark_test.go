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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
)

const (
	baseRef                = "HEAD~1"
	headRef                = "HEAD"
	integrationSinglePath  = "./testdata/single-file/1-v.bal"
	integrationDirPath     = "./testdata/directory"
	integrationProjectPath = "./testdata/project"
	benchmarkTool          = "bal-benchmark-tool"
	integrationCoverDirEnv = "CODECOV_INTEGRATION_COVERDIR"
)

// integrationTargets are paths relative to compiler-tools/benchmark (the test binary cwd).
var integrationTargets = []string{
	integrationSinglePath,
}

var (
	buildBinaryOnce sync.Once
	binaryPath      string
	binaryCoverDir  string
	binaryBuildErr  error
)

func TestBenchmarkBinaryRunExportsHTML(t *testing.T) {
	skipUnlessBenchmarkIntegration(t)
	t.Parallel()

	bin := ensureBenchmarkBinary(t)
	for _, targetPath := range integrationTargets {
		t.Run(filepath.Base(targetPath), func(t *testing.T) {
			t.Parallel()
			tmp := t.TempDir()
			htmlPath := filepath.Join(tmp, "output.html")
			htmlReport := assertBenchmarkBinarySuccessAndReadReport(t, bin, htmlPath,
				"--warmup", "0",
				"--runs", "1",
				"--export-html", htmlPath,
				baseRef, headRef, targetPath,
			)
			if len(htmlReport) == 0 {
				t.Fatalf("expected non-empty html report at %q", htmlPath)
			}
			text := string(htmlReport)
			if !strings.Contains(text, baseRef) || !strings.Contains(text, headRef) {
				t.Fatalf("expected report to compare refs %q and %q", baseRef, headRef)
			}
		})
	}
}

func TestBenchmarkBinaryRunExportsHTMLForDirectoryTarget(t *testing.T) {
	skipUnlessBenchmarkIntegration(t)
	t.Parallel()

	target, err := resolveTarget(integrationDirPath)
	if err != nil {
		t.Fatalf("failed to resolve directory target %q: %v", integrationDirPath, err)
	}
	if target.mode != multipleFilesMode {
		t.Fatalf("expected multipleFilesMode for %q, got %v", integrationDirPath, target.mode)
	}

	bin := ensureBenchmarkBinary(t)
	tmp := t.TempDir()
	htmlPath := filepath.Join(tmp, "output.html")
	htmlReport := assertBenchmarkBinarySuccessAndReadReport(t, bin, htmlPath,
		"--warmup", "0",
		"--runs", "1",
		"--export-html", htmlPath,
		baseRef, headRef, integrationDirPath,
	)
	text := string(htmlReport)
	for _, path := range target.paths {
		if !strings.Contains(text, getRelativeLabel(target.root, path)) {
			t.Fatalf("expected report to include case label %q", getRelativeLabel(target.root, path))
		}
	}
}

func TestResolveDirectoryTargetIncludesProjectRootWithoutProjectFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	directFile := filepath.Join(root, "1-v.bal")
	projectDir := filepath.Join(root, "project-v")
	projectFile := filepath.Join(projectDir, "main.bal")
	moduleFile := filepath.Join(projectDir, "modules", "mod", "mod.bal")
	for _, path := range []string{directFile, projectFile, moduleFile} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("public function main() {}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(projectDir, "Ballerina.toml"), []byte("[package]\norg = \"benchorg\"\nname = \"project\"\nversion = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	target, err := resolveTarget(root)
	if err != nil {
		t.Fatalf("resolveTarget(%q): %v", root, err)
	}
	want := []string{directFile, projectDir}
	if !slices.Equal(target.paths, want) {
		t.Fatalf("target paths = %v, want %v", target.paths, want)
	}
}

func TestBenchmarkBinaryRunExportsHTMLForPackageTarget(t *testing.T) {
	skipUnlessBenchmarkIntegration(t)
	t.Parallel()

	target, err := resolveTarget(integrationProjectPath)
	if err != nil {
		t.Fatalf("failed to resolve package target %q: %v", integrationProjectPath, err)
	}
	if target.mode != packageMode {
		t.Fatalf("expected packageMode for %q, got %v", integrationProjectPath, target.mode)
	}

	bin := ensureBenchmarkBinary(t)
	tmp := t.TempDir()
	htmlPath := filepath.Join(tmp, "output.html")
	htmlReport := assertBenchmarkBinarySuccessAndReadReport(t, bin, htmlPath,
		"--warmup", "0",
		"--runs", "1",
		"--export-html", htmlPath,
		baseRef, headRef, integrationProjectPath,
	)
	text := string(htmlReport)
	if !strings.Contains(text, target.label) {
		t.Fatalf("expected report to include package label %q", target.label)
	}
}

func TestConfigValidateAllowsEmptyExportPath(t *testing.T) {
	t.Parallel()
	target := filepath.Join(t.TempDir(), "test.bal")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config{
		baseRef:    baseRef,
		headRef:    headRef,
		target:     target,
		warmup:     0,
		runs:       1,
		exportPath: "",
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() returned error for empty export path: %v", err)
	}
}

func TestParseConfigDefaultsToEmptyExportPath(t *testing.T) {
	t.Parallel()
	target := filepath.Join(t.TempDir(), "test.bal")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{
		"bal-bench",
		baseRef,
		headRef,
		target,
	}
	cfg, err := parseConfig()
	if err != nil {
		t.Fatalf("parseConfig() returned error: %v", err)
	}
	if cfg.exportPath != "" {
		t.Fatalf("expected exportPath to default to empty, got %q", cfg.exportPath)
	}
	if cfg.mode != timeMode {
		t.Fatalf("expected mode to default to %q, got %q", timeMode, cfg.mode)
	}
}

func TestBenchmarkBinaryFailsForInvalidMode(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--mode", "unknown",
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef, integrationSinglePath,
		},
		"mode must be one of",
	)
}

func TestBenchmarkBinaryFailsForInvalidRunsInMemoryMode(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--mode", "memory",
			"--runs", "0",
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef, integrationSinglePath,
		},
		"runs must be greater than zero",
	)
}

func TestBenchmarkBinaryFailsForInvalidWarmupInMemoryMode(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--mode", "memory",
			"--warmup", "-1",
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef, integrationSinglePath,
		},
		"warmup must be non-negative",
	)
}

func TestParseGNUMaxRSSMiB(t *testing.T) {
	t.Parallel()
	got, err := parseGNUMaxRSSMiB("Maximum resident set size (kbytes): 1536\n")
	if err != nil {
		t.Fatalf("parseGNUMaxRSSMiB() returned error: %v", err)
	}
	if got != 1.5 {
		t.Fatalf("parseGNUMaxRSSMiB() = %v, want 1.5", got)
	}
}

func TestParseMacOSMaxRSSMiB(t *testing.T) {
	t.Parallel()
	got, err := parseMacOSMaxRSSMiB("1572864  maximum resident set size\n")
	if err != nil {
		t.Fatalf("parseMacOSMaxRSSMiB() returned error: %v", err)
	}
	if got != 1.5 {
		t.Fatalf("parseMacOSMaxRSSMiB() = %v, want 1.5", got)
	}
}

func TestHyperfineFlagsWithPositiveWarmup(t *testing.T) {
	t.Parallel()
	b := &benchmark{config: config{warmup: 3, runs: 7}}
	got := b.hyperfineFlags()
	want := []string{"--show-output", "--warmup", "3", "--runs", "7"}
	if !slices.Equal(got, want) {
		t.Fatalf("hyperfineFlags() = %v, want %v", got, want)
	}
}

func TestHyperfineFlagsOmitsWarmupWhenZero(t *testing.T) {
	t.Parallel()
	b := &benchmark{config: config{warmup: 0, runs: 2}}
	got := b.hyperfineFlags()
	want := []string{"--show-output", "--runs", "2"}
	if !slices.Equal(got, want) {
		t.Fatalf("hyperfineFlags() = %v, want %v", got, want)
	}
}

func TestGetRelativeLabel(t *testing.T) {
	t.Parallel()
	t.Run("empty_root_uses_base_name", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join("some", "dir", "file.bal")
		got := getRelativeLabel("", path)
		want := filepath.Base(path)
		if got != want {
			t.Fatalf("getRelativeLabel(%q, %q) = %q, want %q", "", path, got, want)
		}
	})
	t.Run("under_root_uses_relative_path", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		path := filepath.Join(root, "nested", "case.bal")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		got := getRelativeLabel(root, path)
		want := filepath.Join("nested", "case.bal")
		if got != want {
			t.Fatalf("getRelativeLabel(%q, %q) = %q, want %q", root, path, got, want)
		}
	})
}

func TestBenchmarkResultLabel(t *testing.T) {
	t.Parallel()
	t.Run("non_directory_mode_uses_target_label", func(t *testing.T) {
		t.Parallel()
		target := &benchmarkTarget{mode: singleFileMode, label: "main.bal"}
		got := benchmarkResultLabel(target, "/ignored/path.bal")
		if got != "main.bal" {
			t.Fatalf("got %q, want main.bal", got)
		}
	})
	t.Run("multiple_files_returns_path_when_no_dotdot_prefix", func(t *testing.T) {
		t.Parallel()
		target := &benchmarkTarget{mode: multipleFilesMode, label: "cases"}
		path := filepath.Join("cases", "sub", "1-v.bal")
		got := benchmarkResultLabel(target, path)
		if got != path {
			t.Fatalf("got %q, want %q", got, path)
		}
	})
	t.Run("multiple_files_strips_leading_dotdot_slash", func(t *testing.T) {
		t.Parallel()
		target := &benchmarkTarget{mode: multipleFilesMode, label: "cases"}
		got := benchmarkResultLabel(target, "../outer/cases/sub/1-v.bal")
		want := "outer/cases/sub/1-v.bal"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

func TestBenchmarkBinaryFailsForMissingTarget(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef, filepath.Join(t.TempDir(), "does-not-exist.bal"),
		},
		"target does not exist",
	)
}

func TestBenchmarkBinaryFailsForNonBalFileTarget(t *testing.T) {
	requireHyperfine(t)
	t.Parallel()

	bin := ensureBenchmarkBinary(t)
	tmp := t.TempDir()
	txtPath := filepath.Join(tmp, "sample.txt")
	if err := os.WriteFile(txtPath, []byte("not ballerina"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--export-html", filepath.Join(tmp, "output.html"),
			baseRef, headRef, txtPath,
		},
		"not a .bal file",
	)
}

func TestBenchmarkBinaryFailsForDirectoryWithNoBalFiles(t *testing.T) {
	requireHyperfine(t)
	t.Parallel()

	bin := ensureBenchmarkBinary(t)
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "notes.txt"), []byte("readme"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--export-html", filepath.Join(tmp, "output.html"),
			baseRef, headRef, tmp,
		},
		"no .bal files found in directory",
	)
}

func TestBenchmarkBinaryFailsForInvalidRuns(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--runs", "0",
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef, integrationSinglePath,
		},
		"runs must be greater than zero",
	)
}

func TestBenchmarkBinaryFailsForInvalidWarmup(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--warmup", "-1",
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef, integrationSinglePath,
		},
		"warmup must be non-negative",
	)
}

func TestBenchmarkBinaryFailsForMissingRequiredArguments(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef,
		},
		"missing required arguments",
	)
}

func TestBenchmarkBinaryFailsForEmptyBaseRef(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			"", headRef, integrationSinglePath,
		},
		"baseRef is required",
	)
}

func TestBenchmarkBinaryFailsForEmptyHeadRef(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, "", integrationSinglePath,
		},
		"headRef is required",
	)
}

func TestBenchmarkBinaryFailsForEmptyTarget(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	assertBenchmarkBinaryFailure(t, bin,
		[]string{
			"--export-html", filepath.Join(t.TempDir(), "output.html"),
			baseRef, headRef, "",
		},
		"target is required",
	)
}

func TestBenchmarkBinaryFailsWithoutHyperfine(t *testing.T) {
	t.Parallel()
	bin := ensureBenchmarkBinary(t)
	cmd := exec.Command(bin,
		"--warmup", "0",
		"--runs", "1",
		"--export-html", filepath.Join(t.TempDir(), "output.html"),
		baseRef, headRef, integrationSinglePath,
	)
	cmd.Dir = "."
	cmd.Env = []string{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected command to fail without hyperfine\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "hyperfine is required but was not found in PATH") {
		t.Fatalf("stderr mismatch, expected hyperfine lookup failure\nstderr:\n%s", stderr.String())
	}
}

type commandResult struct {
	stdout string
	stderr string
	err    error
}

func ensureBenchmarkBinary(t *testing.T) string {
	t.Helper()
	buildBinaryOnce.Do(func() {
		var err error
		binaryCoverDir, err = resolveIntegrationCoverDir()
		if err != nil {
			binaryBuildErr = err
			return
		}
		path := filepath.Join(os.TempDir(), benchmarkTool)
		if runtime.GOOS == "windows" {
			path += ".exe"
		}
		args := []string{"build", "-o", path}
		if binaryCoverDir != "" {
			args = append(args, "-cover", "-coverpkg=./...")
		}
		args = append(args, ".")
		cmd := exec.Command("go", args...)
		cmd.Dir = "."
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		binaryBuildErr = cmd.Run()
		if binaryBuildErr != nil {
			binaryBuildErr = &buildError{
				err:    binaryBuildErr,
				stderr: stderr.String(),
			}
			return
		}
		binaryPath = path
	})
	if binaryBuildErr != nil {
		t.Fatalf("failed to build benchmark tool: %v", binaryBuildErr)
	}
	return binaryPath
}

func runBenchmarkBinary(t *testing.T, bin string, args ...string) commandResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = "."
	if binaryCoverDir != "" {
		cmd.Env = append(os.Environ(), "GOCOVERDIR="+binaryCoverDir)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return commandResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
		err:    err,
	}
}

func assertBenchmarkBinaryFailure(t *testing.T, bin string, args []string, wantInStdErr string) {
	t.Helper()
	result := runBenchmarkBinary(t, bin, args...)
	if result.err == nil {
		t.Fatalf("expected command to fail\nstdout:\n%s\nstderr:\n%s", result.stdout, result.stderr)
	}
	if !strings.Contains(result.stderr, wantInStdErr) {
		t.Fatalf("stderr mismatch, want %q\nstderr:\n%s", wantInStdErr, result.stderr)
	}
}

func assertBenchmarkBinarySuccessAndReadReport(t *testing.T, bin, htmlPath string, args ...string) []byte {
	t.Helper()
	result := runBenchmarkBinary(t, bin, args...)
	if result.err != nil {
		t.Fatalf("benchmark run failed: %v\nstderr:\n%s", result.err, result.stderr)
	}
	htmlReport, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("expected html report at %q: %v", htmlPath, err)
	}
	return htmlReport
}

type buildError struct {
	err    error
	stderr string
}

func (e *buildError) Error() string {
	if e.stderr == "" {
		return e.err.Error()
	}
	return e.err.Error() + ": " + e.stderr
}

func resolveIntegrationCoverDir() (string, error) {
	d := os.Getenv(integrationCoverDirEnv)
	if d == "" {
		return "", nil
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s %q: %w", integrationCoverDirEnv, d, err)
	}
	return d, nil
}

func skipUnlessBenchmarkIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping benchmark integration test in short mode")
	}
	requireHyperfine(t)
	if err := exec.Command("git", "rev-parse", "--verify", baseRef).Run(); err != nil {
		t.Skipf("skipping benchmark integration test because base ref %q is unavailable: %v", baseRef, err)
	}
	if err := exec.Command("git", "rev-parse", "--verify", headRef).Run(); err != nil {
		t.Skipf("skipping benchmark integration test because head ref %q is unavailable: %v", headRef, err)
	}
	for _, p := range append(integrationTargets, integrationDirPath, integrationProjectPath) {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("benchmark target %q is unavailable: %v", p, err)
		}
	}
}

func requireHyperfine(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("hyperfine"); err != nil {
		t.Fatalf("hyperfine is required but unavailable: %v", err)
	}
}
