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

package corpus

import (
	"encoding/json"
	"os"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const traceFileName = "traces.json"

// Fixtures reused from the existing corpus; only readTraceFixture is specific
// to tracing, because no other fixture reads its own trace back.
var (
	runAndPrintFixture  = filepath.Join("corpus", "cli", "testdata", "run", "projects", "run-and-print")
	multiModuleFixture  = filepath.Join("projects", "testdata", "multi-module-project")
	compileErrorFixture = filepath.Join("corpus", "cli", "testdata", "build", "compile-error", "project")
	syntaxErrorFixture  = filepath.Join("corpus", "cli", "testdata", "run", "projects", "syntax-error")
	singleFileFixture   = filepath.Join("corpus", "cli", "testdata", "run", "single-bal-files")
	workspaceFixture    = filepath.Join("corpus", "cli", "testdata", "run", "workspaces", "run-workspace-corpus")
	readTraceFixture    = filepath.Join("corpus", "cli", "testdata", "run", "trace", "read-trace")
	childSpanFixture    = filepath.Join("corpus", "cli", "testdata", "run", "trace", "children")
)

const runAndPrintOutput = "project says hi"

type traceEvent struct {
	Name string  `json:"name"`
	Cat  string  `json:"cat"`
	Ph   string  `json:"ph"`
	Pid  int     `json:"pid"`
	Tid  int     `json:"tid"`
	Ts   float64 `json:"ts"`
	Dur  float64 `json:"dur"`
	Args struct {
		SpanID   string `json:"span_id"`
		ParentID string `json:"parent_id"`
	} `json:"args"`
}

type traceDocument struct {
	TraceEvents []traceEvent `json:"traceEvents"`
}

// TestBalStatsFlagsRemoved covers the removal of --stats/--stats-oneline:
// run, build, and pack must reject them in both build modes, and no help
// text may advertise them.
func TestBalStatsFlagsRemoved(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)

	for _, debugBuild := range []bool{false, true} {
		t.Run(buildModeName(debugBuild), func(t *testing.T) {
			t.Parallel()
			balBin, repoRoot, coverDir := integrationTestBalCLI(t, debugBuild)
			projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

			for _, command := range []string{"run", "build", "pack"} {
				for _, flag := range []string{"--stats", "--stats-oneline"} {
					_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
						[]string{"BAL_ENV=" + cliIntegrationBalEnv}, command, projectDir, flag)
					if exitCode == 0 {
						t.Errorf("bal %s %s succeeded; expected the removed flag to be rejected", command, flag)
					}
					if !strings.Contains(stderr, "unknown flag") {
						t.Errorf("bal %s %s: expected an unknown-flag error, got stderr:\n%s", command, flag, stderr)
					}
				}

				helpOut, _, _ := runCLICommand(t, balBin, repoRoot, coverDir, command, "--help")
				if strings.Contains(helpOut, "--stats") {
					t.Errorf("bal %s --help still advertises --stats:\n%s", command, helpOut)
				}
			}
		})
	}
}

// TestBalTraceFlagIsDebugRunOnly covers flag visibility: --trace exists only on
// a debug build's run command.
func TestBalTraceFlagIsDebugRunOnly(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)

	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	t.Run("release run rejects --trace", func(t *testing.T) {
		t.Parallel()
		balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
		_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
			[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "run", projectDir, "--trace")
		if exitCode == 0 || !strings.Contains(stderr, "unknown flag") {
			t.Errorf("release bal run --trace: exit=%d stderr:\n%s", exitCode, stderr)
		}
	})

	for _, debugBuild := range []bool{false, true} {
		t.Run("build and pack reject --trace in "+buildModeName(debugBuild), func(t *testing.T) {
			t.Parallel()
			balBin, repoRoot, coverDir := integrationTestBalCLI(t, debugBuild)
			for _, command := range []string{"build", "pack"} {
				_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
					[]string{"BAL_ENV=" + cliIntegrationBalEnv}, command, projectDir, "--trace")
				if exitCode == 0 || !strings.Contains(stderr, "unknown flag") {
					t.Errorf("bal %s --trace: exit=%d stderr:\n%s", command, exitCode, stderr)
				}
			}
		})
	}

	t.Run("debug run advertises --trace", func(t *testing.T) {
		t.Parallel()
		balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
		helpOut, _, _ := runCLICommand(t, balBin, repoRoot, coverDir, "run", "--help")
		if !strings.Contains(helpOut, "--trace") {
			t.Errorf("debug bal run --help does not advertise --trace:\n%s", helpOut)
		}
	})
}

// TestBalRunWithoutTraceFlagWritesNoTrace covers a debug run that does not ask
// for tracing: nothing is collected and nothing is written.
func TestBalRunWithoutTraceFlagWritesNoTrace(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
	workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)

	stdout, stderr, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".")
	if exitCode != 0 {
		t.Fatalf("bal run failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, runAndPrintOutput) {
		t.Errorf("program output missing:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(workDir, traceFileName)); !os.IsNotExist(err) {
		t.Errorf("expected no trace file without --trace, stat err = %v", err)
	}
}

// TestBalRunTraceOutputPaths covers path handling: the bare flag, cwd-relative
// and absolute paths, silent replacement, created parents, and rejection of an
// explicitly empty value.
func TestBalRunTraceOutputPaths(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)

	t.Run("bare flag writes traces.json in the working directory", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		runTracedFixture(t, balBin, workDir, coverDir, "--trace")
		assertTraceHasSpans(t, filepath.Join(workDir, traceFileName))
	})

	t.Run("relative path creates missing parents", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		runTracedFixture(t, balBin, workDir, coverDir, "--trace=out/nested/run.json")
		assertTraceHasSpans(t, filepath.Join(workDir, "out", "nested", "run.json"))
	})

	t.Run("absolute path is independent of the project location", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		tracePath := filepath.Join(t.TempDir(), "elsewhere", "run.json")
		runTracedFixture(t, balBin, workDir, coverDir, "--trace="+tracePath)
		assertTraceHasSpans(t, tracePath)
	})

	t.Run("existing output is silently replaced", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		tracePath := filepath.Join(workDir, traceFileName)
		if err := os.WriteFile(tracePath, []byte("stale contents"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, stderr := runTracedFixture(t, balBin, workDir, coverDir, "--trace")
		if strings.Contains(stderr, "stale") {
			t.Errorf("unexpected warning about the replaced file:\n%s", stderr)
		}
		assertTraceHasSpans(t, tracePath)
	})

	t.Run("explicitly empty value is rejected", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		stdout, stderr, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace=")
		if exitCode == 0 {
			t.Fatalf("bal run --trace= succeeded; expected it to be rejected\nstdout:\n%s", stdout)
		}
		if !strings.Contains(stderr, "--trace requires a path") {
			t.Errorf("expected a --trace usage error, got stderr:\n%s", stderr)
		}
		if strings.Contains(stdout, runAndPrintOutput) {
			t.Errorf("program ran despite the rejected flag:\n%s", stdout)
		}
	})
}

// TestBalRunTraceWriteFailureFailsTheRun covers output failures: the command
// fails and the program never runs.
func TestBalRunTraceWriteFailureFailsTheRun(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)

	t.Run("destination is a directory", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		if err := os.Mkdir(filepath.Join(workDir, "trace-dir"), 0o755); err != nil {
			t.Fatal(err)
		}
		assertTraceWriteFailure(t, balBin, workDir, coverDir, "--trace=trace-dir")
	})

	t.Run("parent path is a file", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		if err := os.WriteFile(filepath.Join(workDir, "blocker"), []byte("not a directory"), 0o644); err != nil {
			t.Fatal(err)
		}
		assertTraceWriteFailure(t, balBin, workDir, coverDir, "--trace=blocker/run.json")
	})

	t.Run("destination directory is unwritable", func(t *testing.T) {
		t.Parallel()
		if goruntime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("directory permissions do not block writes here")
		}
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		readOnly := filepath.Join(workDir, "readonly")
		if err := os.Mkdir(readOnly, 0o555); err != nil {
			t.Fatal(err)
		}
		assertTraceWriteFailure(t, balBin, workDir, coverDir, "--trace=readonly/run.json")
	})
}

// TestBalRunTraceLabels covers the invocation labels and identities produced by
// valid compilations of each project shape.
func TestBalRunTraceLabels(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)

	for _, tc := range []struct {
		name     string
		fixture  string
		runDir   string
		runArg   string
		expected []string
	}{
		{
			name:    "single file",
			fixture: singleFileFixture,
			runArg:  "run-and-print.bal",
			expected: []string{
				"Parse run-and-print.bal",
				"AST Build run-and-print.bal",
				"Symbol Resolution $anon/run-and-print:0.0.0",
				"Package Assembly $anon/run-and-print:0.0.0",
				"Top-Level Type Resolution $anon/run-and-print:0.0.0",
				"Local Type Resolution $anon/run-and-print:0.0.0",
				"Semantic Analysis $anon/run-and-print:0.0.0",
				"CFG Creation $anon/run-and-print:0.0.0",
				"CFG Analysis $anon/run-and-print:0.0.0",
				"Desugaring $anon/run-and-print:0.0.0",
				"BIR Generation $anon/run-and-print:0.0.0",
				// A source dependency compiled in the same environment keeps
				// its diagnostic identity prefix.
				"Parse ballerina/io/0.0.1::io.bal",
				"BIR Generation ballerina/io:0.0.1",
			},
		},
		{
			name:    "multi file and multi module",
			fixture: multiModuleFixture,
			runArg:  ".",
			expected: []string{
				"Parse main.bal",
				"Parse utils.bal",
				"Parse modules/services/svc.bal",
				"Parse modules/storage/db.bal",
				"AST Build utils.bal",
				"AST Build modules/storage/db.bal",
				"Package Assembly testorg/multimoduleproject:0.1.0",
				"Symbol Resolution testorg/multimoduleproject.storage:0.1.0",
				"BIR Generation testorg/multimoduleproject.services:0.1.0",
				"BIR Generation testorg/multimoduleproject:0.1.0",
			},
		},
		{
			name:    "workspace member",
			fixture: workspaceFixture,
			runDir:  "pkgmain",
			runArg:  ".",
			expected: []string{
				"Parse pkgmain/main.bal",
				"AST Build pkgmain/main.bal",
				"Symbol Resolution testorg/cli_ws_pkgmain:0.1.0",
				"BIR Generation testorg/cli_ws_pkgmain:0.1.0",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			workDir := copyTraceFixture(t, repoRoot, tc.fixture)
			runDir := workDir
			if tc.runDir != "" {
				runDir = filepath.Join(workDir, tc.runDir)
			}

			stdout, stderr, exitCode := runBalInDir(t, balBin, runDir, coverDir, "run", tc.runArg, "--trace")
			if exitCode != 0 {
				t.Fatalf("bal run failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
			}

			events := readTrace(t, filepath.Join(runDir, traceFileName))
			names := map[string]int{}
			for _, event := range events {
				names[event.Name]++
			}
			for _, want := range tc.expected {
				if names[want] == 0 {
					t.Errorf("missing span %q; recorded spans:\n%s", want, strings.Join(sortedNames(names), "\n"))
				}
			}
		})
	}
}

// TestBalRunTraceOnCompilationFailure covers partial and empty recordings:
// diagnostic barriers still hold, so no span exists for a stage that never ran.
func TestBalRunTraceOnCompilationFailure(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)

	t.Run("syntax error yields a partial trace", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, syntaxErrorFixture)
		_, _, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace")
		if exitCode == 0 {
			t.Fatal("expected a non-zero exit code for a syntax error")
		}

		names := map[string]int{}
		for _, event := range readTrace(t, filepath.Join(workDir, traceFileName)) {
			names[event.Name]++
		}
		if names["Parse main.bal"] == 0 {
			t.Errorf("expected the failing package's parse span; got:\n%s", strings.Join(sortedNames(names), "\n"))
		}
		for name := range names {
			if strings.HasPrefix(name, "Desugaring testorg/") || strings.HasPrefix(name, "BIR Generation testorg/") {
				t.Errorf("span %q recorded for a stage that must not have run", name)
			}
		}
	})

	t.Run("symbol resolution failure stops at its barrier", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, compileErrorFixture)
		_, _, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace")
		if exitCode == 0 {
			t.Fatal("expected a non-zero exit code for an undefined symbol")
		}

		names := map[string]int{}
		for _, event := range readTrace(t, filepath.Join(workDir, traceFileName)) {
			names[event.Name]++
		}
		const failingPkg = "testorg/broken:0.1.0"
		for _, want := range []string{
			"Parse main.bal",
			"Parse util.bal",
			"Symbol Resolution " + failingPkg,
		} {
			if names[want] == 0 {
				t.Errorf("missing span %q; recorded spans:\n%s", want, strings.Join(sortedNames(names), "\n"))
			}
		}
		for _, unwanted := range []string{
			"Local Type Resolution " + failingPkg,
			"Semantic Analysis " + failingPkg,
			"CFG Creation " + failingPkg,
			"CFG Analysis " + failingPkg,
			"Desugaring " + failingPkg,
			"BIR Generation " + failingPkg,
		} {
			if names[unwanted] != 0 {
				t.Errorf("span %q recorded for a stage that must not have run", unwanted)
			}
		}
	})

	t.Run("recovery-mode AST construction is recorded", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, syntaxErrorFixture)
		_, _, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--dump-recovered-ast", "--trace")
		if exitCode == 0 {
			t.Fatal("expected a non-zero exit code for a syntax error")
		}

		found := false
		for _, event := range readTrace(t, filepath.Join(workDir, traceFileName)) {
			if event.Name == "AST Build (recovered) main.bal" {
				found = true
			}
		}
		if !found {
			t.Error("expected a recovered-mode AST build span")
		}
	})

	t.Run("compilation and write failures are both reported", func(t *testing.T) {
		t.Parallel()
		workDir := copyTraceFixture(t, repoRoot, syntaxErrorFixture)
		if err := os.Mkdir(filepath.Join(workDir, "trace-dir"), 0o755); err != nil {
			t.Fatal(err)
		}
		_, stderr, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace=trace-dir")
		if exitCode == 0 {
			t.Fatal("expected a non-zero exit code")
		}
		if !strings.Contains(stderr, "compilation contains errors") {
			t.Errorf("original compilation failure was not retained:\n%s", stderr)
		}
		if !strings.Contains(stderr, "write trace") {
			t.Errorf("trace write failure was not additionally reported:\n%s", stderr)
		}
	})

	t.Run("early load failure yields an empty trace", func(t *testing.T) {
		t.Parallel()
		workDir := t.TempDir()
		_, _, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", "no-such-file.bal", "--trace")
		if exitCode == 0 {
			t.Fatal("expected a non-zero exit code for a missing source file")
		}
		if events := readTrace(t, filepath.Join(workDir, traceFileName)); len(events) != 0 {
			t.Errorf("expected an empty trace, got %d events", len(events))
		}
	})
}

// TestBalRunTraceIsCompleteBeforeRuntime covers the ordering guarantee: the
// running program reads the finished recording during initialization, and the
// program's own failure afterwards neither rewrites nor extends it.
func TestBalRunTraceIsCompleteBeforeRuntime(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
	workDir := copyTraceFixture(t, repoRoot, readTraceFixture)

	// The fixture reports the trace size it read and then panics.
	stdout, stderr, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace")
	if exitCode == 0 {
		t.Fatalf("expected the program's runtime failure to fail the run\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	observedLen := observedTraceLength(t, stdout)
	tracePath := filepath.Join(workDir, traceFileName)
	onDisk, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if observedLen != len(onDisk) {
		t.Errorf("trace observed at runtime was %d bytes, on disk it is %d; the runtime changed it",
			observedLen, len(onDisk))
	}
	assertTraceHasSpans(t, tracePath)
}

func observedTraceLength(t *testing.T, stdout string) int {
	t.Helper()
	_, observed, found := strings.Cut(stdout, "trace bytes: ")
	if !found {
		t.Fatalf("program did not observe the trace:\n%s", stdout)
	}
	observed, _, _ = strings.Cut(observed, "\n")
	length, err := strconv.Atoi(strings.TrimSpace(observed))
	if err != nil {
		t.Fatalf("unexpected program output %q: %v", stdout, err)
	}
	return length
}

func skipTraceTestOnWasm(t *testing.T) {
	t.Helper()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
}

func buildModeName(debugBuild bool) string {
	if debugBuild {
		return "debug build"
	}
	return "release build"
}

// copyTraceFixture copies a repo-relative fixture into a temp directory so each
// run owns its working directory and trace output.
func copyTraceFixture(t *testing.T, repoRoot, fixture string) string {
	t.Helper()
	workDir := t.TempDir()
	copyDir(t, filepath.Join(repoRoot, fixture), workDir)
	return workDir
}

// runBalInDir runs the CLI with workDir as the process working directory, which
// is what cwd-relative trace paths resolve against.
func runBalInDir(t *testing.T, balBin, workDir, coverDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	env := append(os.Environ(), "BAL_ENV="+cliIntegrationBalEnv)
	if coverDir != "" {
		commandCoverDir := t.TempDir()
		env = append(env, "GOCOVERDIR="+commandCoverDir)
		defer mergeCLICoverageDir(t, commandCoverDir, coverDir)
	}
	return runNativeCLICommandWithEnv(t, balBin, workDir, args, env)
}

func runTracedFixture(t *testing.T, balBin, workDir, coverDir string, traceArg string) (stdout, stderr string) {
	t.Helper()
	stdout, stderr, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", traceArg)
	if exitCode != 0 {
		t.Fatalf("bal run %s failed: exit=%d\nstdout:\n%s\nstderr:\n%s", traceArg, exitCode, stdout, stderr)
	}
	return stdout, stderr
}

func assertTraceWriteFailure(t *testing.T, balBin, workDir, coverDir, traceArg string) {
	t.Helper()
	stdout, stderr, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", traceArg)
	if exitCode == 0 {
		t.Fatalf("bal run %s succeeded; expected the trace write to fail\nstdout:\n%s", traceArg, stdout)
	}
	if strings.Contains(stdout, runAndPrintOutput) {
		t.Errorf("program ran despite the trace write failure:\n%s", stdout)
	}
	if !strings.Contains(stderr, "trace") {
		t.Errorf("expected a trace write error, got stderr:\n%s", stderr)
	}
}

func readTrace(t *testing.T, path string) []traceEvent {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading trace: %v", err)
	}
	var document traceDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("trace at %s is not valid JSON: %v\n%s", path, err, data)
	}
	if !strings.Contains(string(data), `"traceEvents"`) {
		t.Fatalf("trace at %s has no traceEvents array:\n%s", path, data)
	}
	return document.TraceEvents
}

// assertTraceHasSpans checks the document a Chrome-compatible viewer would open:
// complete-duration events on one logical process with nonnegative durations.
func assertTraceHasSpans(t *testing.T, path string) {
	t.Helper()
	events := readTrace(t, path)
	if len(events) == 0 {
		t.Fatalf("trace at %s has no spans", path)
	}
	spanIDs := map[string]bool{}
	for _, event := range events {
		if event.Ph != "X" || event.Cat != "frontend" {
			t.Fatalf("unexpected event envelope: %+v", event)
		}
		if event.Dur < 0 {
			t.Fatalf("negative duration: %+v", event)
		}
		if event.Args.SpanID == "" || event.Args.SpanID == "0" {
			t.Fatalf("missing span id: %+v", event)
		}
		if spanIDs[event.Args.SpanID] {
			t.Fatalf("duplicate span id %s", event.Args.SpanID)
		}
		spanIDs[event.Args.SpanID] = true
		if event.Pid != events[0].Pid {
			t.Fatalf("expected one logical process id, got %d and %d", event.Pid, events[0].Pid)
		}
	}
}

func sortedNames(names map[string]int) []string {
	unique := make([]string, 0, len(names))
	for name := range names {
		unique = append(unique, name)
	}
	slices.Sort(unique)
	return unique
}

// TestBalRunTraceRejectsNativeDependencyHandoff covers the native-interpreter
// handoff guard: with tracing enabled the run fails before the native runner is
// prepared, and the parent process's own spans are still written. The
// non-tracing handoff path is unchanged and covered by
// TestNativeRunner_ColdBuildAndCacheHit.
func TestBalRunTraceRejectsNativeDependencyHandoff(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)

	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	copyDir(t, filepath.Join(repoRoot, "projects", "testdata", "repo", "bala"), centralCache)

	workDir := t.TempDir()
	copyDir(t, filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v"), workDir)

	env := append(envWithoutVars(os.Environ(), "BALLERINA_SRC"), "BAL_ENV="+tempHome)
	if coverDir != "" {
		commandCoverDir := t.TempDir()
		env = append(env, "GOCOVERDIR="+commandCoverDir)
		defer mergeCLICoverageDir(t, commandCoverDir, coverDir)
	}

	stdout, stderr, exitCode := runNativeCLICommandWithEnv(t, balBin, workDir, []string{"run", ".", "--trace"}, env)
	if exitCode == 0 {
		t.Fatalf("expected the traced native handoff to fail\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "--trace is not supported") {
		t.Errorf("expected the native-handoff guard message, got stderr:\n%s", stderr)
	}
	if strings.Contains(stderr, "building native interpreter") {
		t.Errorf("native runner was prepared before the guard fired:\n%s", stderr)
	}
	if strings.Contains(stderr, "panic:") {
		t.Errorf("the guard panicked instead of reporting an error:\n%s", stderr)
	}
	readTrace(t, filepath.Join(workDir, traceFileName))
}

// traceEpsilon absorbs the float rounding of recomputing an end offset as
// ts + dur; it is far below the one-nanosecond resolution of the recording.
const traceEpsilon = 1e-6

const childSpanPackage = "testorg/cli_trace_children:0.1.0"

// TestBalRunTraceChildSpans covers the span tree: every phase decomposes into
// children, the recorded parentage is well formed, and the lanes render the
// tree without false nesting.
func TestBalRunTraceChildSpans(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
	workDir := copyTraceFixture(t, repoRoot, childSpanFixture)

	// The fixture panics once it has exercised every definition, because its
	// listener would otherwise keep the run alive. The trace is written before
	// the program starts, so the recording is complete either way.
	stdout, stderr, _ := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace", "--nested")
	if !strings.Contains(stdout, "traced") {
		t.Fatalf("fixture did not compile and run:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	events := readTrace(t, filepath.Join(workDir, traceFileName))
	assertTraceHasSpans(t, filepath.Join(workDir, traceFileName))
	assertTraceTreeInvariants(t, events)

	t.Run("each phase decomposes into children", func(t *testing.T) {
		for phase, children := range map[string][]string{
			"Symbol Resolution": {"Import Binding", "Top-Level Symbols main.bal", "Symbol Resolution main.bal"},
			"Top-Level Type Resolution": {
				"Type Definition Entry", "Class Type Counter", "Function Signature main",
				"Global Variable total", "Package Constants", "Service $service$0",
			},
			"Local Type Resolution": {"Class Field Inits Counter", "Function Body accumulate"},
			"Semantic Analysis": {
				"Module Variable Metadata", "Module Isolation Validation",
				"Constant LIMIT", "Type Definition Amount", "Class Definition Counter", "Function main",
			},
			"CFG Creation": {"Function main", "Method add"},
			"CFG Analysis": {
				"Reachability Analysis", "Reachability add", "Explicit Return Analysis",
				"Uninitialized Variable Analysis", "Uninitialized Field Analysis",
				"Uninitialized Global Variable Analysis",
			},
			"Desugaring":     {"Class Counter", "Function add", "Service $service$0", "Resource Method get"},
			"BIR Generation": {"Global Variables", "Class Counter", "Method add", "Function main"},
		} {
			recorded := phaseSpanNames(t, events, phase, childSpanPackage)
			for _, child := range children {
				if !slices.Contains(recorded, child) {
					t.Errorf("%s is missing child span %q; recorded:\n%s",
						phase, child, strings.Join(recorded, "\n"))
				}
			}
		}
	})

	t.Run("a desugared method nests under its class", func(t *testing.T) {
		assertSpanParent(t, events, "Desugaring", "Function add", "Class Counter")
	})

	t.Run("a local type resolution body is a sibling of top-level functions", func(t *testing.T) {
		assertSpanParent(t, events, "Local Type Resolution", "Function Body add",
			"Local Type Resolution "+childSpanPackage)
	})

	t.Run("a per-function reachability span nests under its analysis", func(t *testing.T) {
		assertSpanParent(t, events, "CFG Analysis", "Reachability add", "Reachability Analysis")
	})

	t.Run("a generated BIR method nests under its class", func(t *testing.T) {
		assertSpanParent(t, events, "BIR Generation", "Method add", "Class Counter")
	})

	t.Run("the sequential BIR subtree occupies one lane", func(t *testing.T) {
		lanes := map[int]bool{}
		for _, event := range phaseSubtree(t, events, "BIR Generation", childSpanPackage) {
			lanes[event.Tid] = true
		}
		if len(lanes) != 1 {
			t.Errorf("BIR generation spans occupy %d lanes, want 1", len(lanes))
		}
	})
}

// TestBalRunTraceChildSpansOnCompilationFailure covers a partial recording:
// children recorded before the failure are present, every span that names a
// parent finds it, and the tree invariants still hold.
func TestBalRunTraceChildSpansOnCompilationFailure(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
	workDir := copyTraceFixture(t, repoRoot, compileErrorFixture)

	_, _, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace", "--nested")
	if exitCode == 0 {
		t.Fatal("expected a non-zero exit code for an undefined symbol")
	}

	events := readTrace(t, filepath.Join(workDir, traceFileName))
	assertTraceTreeInvariants(t, events)
	recorded := phaseSpanNames(t, events, "Symbol Resolution", "testorg/broken:0.1.0")
	if !slices.Contains(recorded, "Import Binding") {
		t.Errorf("no symbol resolution children survived the failure; recorded:\n%s",
			strings.Join(recorded, "\n"))
	}
}

func traceEventsByID(events []traceEvent) map[string]traceEvent {
	byID := make(map[string]traceEvent, len(events))
	for _, event := range events {
		byID[event.Args.SpanID] = event
	}
	return byID
}

// phaseRootName returns the name of the root span the event descends from.
func phaseRootName(t *testing.T, byID map[string]traceEvent, event traceEvent) string {
	t.Helper()
	for range len(byID) + 1 {
		if event.Args.ParentID == "" {
			return event.Name
		}
		parent, ok := byID[event.Args.ParentID]
		if !ok {
			t.Fatalf("span %q names parent %s which is not in the document", event.Name, event.Args.ParentID)
		}
		event = parent
	}
	t.Fatalf("parentage of span %q does not reach a root", event.Name)
	return ""
}

// phaseSubtree returns every span recorded under the named phase of one
// package, including the phase root itself.
func phaseSubtree(t *testing.T, events []traceEvent, phase, pkg string) []traceEvent {
	t.Helper()
	byID := traceEventsByID(events)
	root := phase + " " + pkg
	var subtree []traceEvent
	for _, event := range events {
		if phaseRootName(t, byID, event) == root {
			subtree = append(subtree, event)
		}
	}
	if len(subtree) == 0 {
		t.Fatalf("no spans recorded under %q", root)
	}
	return subtree
}

func phaseSpanNames(t *testing.T, events []traceEvent, phase, pkg string) []string {
	t.Helper()
	var names []string
	for _, event := range phaseSubtree(t, events, phase, pkg) {
		names = append(names, event.Name)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func assertSpanParent(t *testing.T, events []traceEvent, phase, child, wantParent string) {
	t.Helper()
	byID := traceEventsByID(events)
	found := false
	for _, event := range phaseSubtree(t, events, phase, childSpanPackage) {
		if event.Name != child {
			continue
		}
		found = true
		if got := byID[event.Args.ParentID].Name; got != wantParent {
			t.Errorf("%s: parent of %q = %q, want %q", phase, child, got, wantParent)
		}
	}
	if !found {
		t.Errorf("%s recorded no span named %q", phase, child)
	}
}

// assertTraceTreeInvariants checks the document-wide guarantees: parentage
// resolves and is acyclic, children are time-contained in their parents, and on
// every lane two slices are disjoint or nested inside a genuine ancestor.
func assertTraceTreeInvariants(t *testing.T, events []traceEvent) {
	t.Helper()
	byID := traceEventsByID(events)
	for _, event := range events {
		if event.Args.ParentID == "" {
			continue
		}
		parent, ok := byID[event.Args.ParentID]
		if !ok {
			t.Fatalf("span %q names parent %s which is not in the document", event.Name, event.Args.ParentID)
		}
		// Reaching a root proves this span's ancestry is finite, so the parent
		// relation contains no cycle through it.
		phaseRootName(t, byID, event)
		if event.Ts+traceEpsilon < parent.Ts ||
			event.Ts+event.Dur > parent.Ts+parent.Dur+traceEpsilon {
			t.Fatalf("span %q is not contained in parent %q", event.Name, parent.Name)
		}
	}
	assertTraceLaneInvariants(t, byID, events)
}

func assertTraceLaneInvariants(t *testing.T, byID map[string]traceEvent, events []traceEvent) {
	t.Helper()
	type lane struct {
		pid, tid int
	}
	byLane := map[lane][]traceEvent{}
	for _, event := range events {
		key := lane{pid: event.Pid, tid: event.Tid}
		byLane[key] = append(byLane[key], event)
	}
	for key, laneEvents := range byLane {
		for i, a := range laneEvents {
			for _, b := range laneEvents[i+1:] {
				aEnd, bEnd := a.Ts+a.Dur, b.Ts+b.Dur
				switch {
				case aEnd <= b.Ts+traceEpsilon || bEnd <= a.Ts+traceEpsilon:
				case a.Ts <= b.Ts+traceEpsilon && bEnd <= aEnd+traceEpsilon:
					assertTraceAncestor(t, byID, key.tid, a, b)
				case b.Ts <= a.Ts+traceEpsilon && aEnd <= bEnd+traceEpsilon:
					assertTraceAncestor(t, byID, key.tid, b, a)
				default:
					t.Fatalf("lane %d overlaps %q and %q partially", key.tid, a.Name, b.Name)
				}
			}
		}
	}
}

func assertTraceAncestor(t *testing.T, byID map[string]traceEvent, tid int, outer, inner traceEvent) {
	t.Helper()
	for id := inner.Args.ParentID; id != ""; id = byID[id].Args.ParentID {
		if id == outer.Args.SpanID {
			return
		}
	}
	t.Fatalf("lane %d draws %q inside unrelated %q", tid, inner.Name, outer.Name)
}

// TestBalRunTraceIsFlatWithoutNested covers the default recording: --trace
// alone records the phase spans and none of the children within them.
func TestBalRunTraceIsFlatWithoutNested(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
	workDir := copyTraceFixture(t, repoRoot, childSpanFixture)

	stdout, stderr, _ := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--trace")
	if !strings.Contains(stdout, "traced") {
		t.Fatalf("fixture did not compile and run:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	events := readTrace(t, filepath.Join(workDir, traceFileName))
	for _, event := range events {
		if event.Args.ParentID != "" {
			t.Fatalf("span %q carries parent_id %q without --nested",
				event.Name, event.Args.ParentID)
		}
	}
	// The phase roots are still there; only what they contain is gone.
	names := map[string]bool{}
	for _, event := range events {
		names[event.Name] = true
	}
	if !names["Desugaring "+childSpanPackage] {
		t.Errorf("phase spans missing from an unnested recording:\n%s",
			strings.Join(sortedNames(countNames(events)), "\n"))
	}
	for _, child := range []string{"Class Counter", "Import Binding", "Reachability Analysis"} {
		if names[child] {
			t.Errorf("child span %q recorded without --nested", child)
		}
	}
}

// TestBalNestedFlagIsDebugRunOnly covers --nested's visibility and its
// dependence on --trace.
func TestBalNestedFlagIsDebugRunOnly(t *testing.T) {
	t.Parallel()
	skipTraceTestOnWasm(t)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	t.Run("release run rejects --nested", func(t *testing.T) {
		t.Parallel()
		balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
		_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
			[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "run", projectDir, "--nested")
		if exitCode == 0 || !strings.Contains(stderr, "unknown flag") {
			t.Errorf("release bal run --nested: exit=%d stderr:\n%s", exitCode, stderr)
		}
	})

	t.Run("debug run advertises --nested", func(t *testing.T) {
		t.Parallel()
		balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
		helpOut, _, _ := runCLICommand(t, balBin, repoRoot, coverDir, "run", "--help")
		if !strings.Contains(helpOut, "--nested") {
			t.Errorf("debug bal run --help does not advertise --nested:\n%s", helpOut)
		}
	})

	t.Run("--nested without --trace is rejected", func(t *testing.T) {
		t.Parallel()
		balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)
		workDir := copyTraceFixture(t, repoRoot, runAndPrintFixture)
		stdout, stderr, exitCode := runBalInDir(t, balBin, workDir, coverDir, "run", ".", "--nested")
		if exitCode == 0 {
			t.Fatalf("bal run --nested succeeded without --trace\nstdout:\n%s", stdout)
		}
		if !strings.Contains(stderr, "--nested requires --trace") {
			t.Errorf("expected a --nested usage error, got stderr:\n%s", stderr)
		}
		if strings.Contains(stdout, runAndPrintOutput) {
			t.Errorf("program ran despite the rejected flag:\n%s", stdout)
		}
	})
}

func countNames(events []traceEvent) map[string]int {
	names := map[string]int{}
	for _, event := range events {
		names[event.Name]++
	}
	return names
}
