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
	"archive/zip"
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/test_util"
)

const (
	cliCoverDirEnv   = "BAL_GOCOVERDIR"
	cliCoverPackages = "github.com/ballerina-nutcracker/ballerina/..."
)

var (
	cliIntegrationBinsOnce    sync.Once
	cliIntegrationRelBalBin   string
	cliIntegrationDebugBalBin string
	cliIntegrationRepoRoot    string
	cliIntegrationCoverDir    string
	cliIntegrationBalEnv      string
	cliIntegrationBinsErr     error
	cliIntegrationCoverMerge  sync.Mutex

	cliIntegrationNoRuntimeBalBinOnce sync.Once
	cliIntegrationNoRuntimeBalBin     string
	cliIntegrationNoRuntimeBalBinErr  error
)

func TestBalHelp(t *testing.T) {
	t.Parallel()
	assertBalCommandMatchesTxtarFragments(t, []string{"--help"}, "help", "help.txtar")
}

func TestBalVersion(t *testing.T) {
	t.Parallel()
	assertBalCommandMatchesTxtarFragments(t, []string{"version"}, "version", "version.txtar")
}

func TestBalRunDumpFlags(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}

	singleBal := filepath.Join("corpus", "cli", "testdata", "run", "single-bal-files", "run-and-print.bal")
	recoveryBal := filepath.Join("corpus", "cli", "testdata", "run", "dump-flags", "recovered-ast.bal")

	tests := []struct {
		name   string
		flag   string
		file   string
		source string
	}{
		{"dump-bir", "--dump-bir", "dump-bir.txtar", singleBal},
		{"dump-st", "--dump-st", "dump-st.txtar", singleBal},
		{"dump-tokens", "--dump-tokens", "dump-tokens.txtar", singleBal},
		{"dump-ast", "--dump-ast", "dump-ast.txtar", singleBal},
		{"dump-recovered-ast", "--dump-recovered-ast", "dump-recovered-ast.txtar", recoveryBal},
		{"dump-cfg", "--dump-cfg", "dump-cfg.txtar", singleBal},
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertBalCommandMatchesTxtarFragmentsForBinary(t, balBin, repoRoot, coverDir,
				[]string{"run", tt.flag, tt.source},
				"run-dump-flags", tt.file)
		})
	}
}

func TestBalRunCorpus(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	testDataRoot := filepath.Join(repoRoot, "corpus", "cli", "testdata", "run")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "run")

	singleBalFiles := listBalRunCorpusPaths(t, filepath.Join(testDataRoot, "single-bal-files"), true)
	projects := listBalRunCorpusPaths(t, filepath.Join(testDataRoot, "projects"), false)

	for _, singleBalFile := range singleBalFiles {
		rel := filepath.Join("single-bal-files", strings.TrimSuffix(filepath.Base(singleBalFile), ".bal"))
		runBalRunCorpusCase(t, balBin, repoRoot, coverDir, outputsRoot, singleBalFile, rel)
	}

	for _, projectDir := range projects {
		rel := filepath.Join("projects", filepath.Base(projectDir))
		runBalRunCorpusCase(t, balBin, repoRoot, coverDir, outputsRoot, projectDir, rel)
	}
}

// TestBalRunWorkspaceCorpus tests the workspace branch in runBallerina (cli/cmd/run.go:206-229).
// It covers three behaviours:
//  1. workspace_root_rejected  — running the workspace root directly is rejected.
//  2. member_resolves          — running a member package path succeeds.
//  3. missing_member           — running a non-existent sub-path is rejected.
//
// Java equivalent: N/A — this is CLI-level integration coverage for the Go workspace branch.
func TestBalRunWorkspaceCorpus(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	wsRoot := filepath.Join(repoRoot, "corpus", "cli", "testdata", "run", "workspaces", "run-workspace-corpus")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "run", "workspaces")

	t.Run("workspace_root_rejected", func(t *testing.T) {
		assertBalCommandMatchesTxtarFragmentsForBinary(t, balBin, repoRoot, coverDir,
			[]string{"run", wsRoot},
			"run", "workspaces", "workspace-root-rejected.txtar")
	})

	t.Run("member_resolves", func(t *testing.T) {
		assertBalCommandMatchesTxtarFragmentsForBinary(t, balBin, repoRoot, coverDir,
			[]string{"run", filepath.Join(wsRoot, "pkgmain")},
			"run", "workspaces", "member-resolves.txtar")
	})

	t.Run("missing_member", func(t *testing.T) {
		// "notamember" is a real directory inside the workspace root but is NOT listed
		// in the workspace Ballerina.toml packages array, so the CLI must reject it.
		assertBalCommandMatchesTxtarFragmentsLoose(t, balBin, repoRoot, coverDir,
			[]string{"run", filepath.Join(wsRoot, "notamember")},
			filepath.Join(outputsRoot, "missing-member.txtar"))
	})
}

// TestBalPackCorpus exercises `bal pack` end-to-end through the coverage-aware
// CLI harness. Each subtest invokes the binary with the scenario's args and
// substring-matches the captured stdout/stderr/exitcode against the txtar at
// corpus/cli/output/pack/<scenario>.txtar. This is the corpus replacement for
// the in-process cli/cmd/pack_test.go suite — running via the real binary
// keeps subprocess coverage flowing into the cli/cmd profile.
//
// Java equivalent: N/A — pack is Go-only CLI integration coverage.
func TestBalPackCorpus(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	testdataRoot := filepath.Join("corpus", "cli", "testdata", "pack")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "pack")

	// Use a guaranteed-missing path under the testdata root for the
	// nonexistent-path scenario. The directory's parent exists (testdata/pack)
	// so the stat error is "no such file or directory" rather than something
	// else like "permission denied".
	missingPath := filepath.Join(testdataRoot, "this-path-does-not-exist")

	basicProject := filepath.Join(testdataRoot, "basic", "project")

	// useDebugBinary, when true, dispatches through the debug-tagged bal binary
	// instead of the release binary. Scenarios that exercise debug-only flags
	// (e.g. --prof, which is registered by prof_debug.go) must set this. If the
	// debug binary build failed (e.g. missing -tags debug support), the scenario
	// is t.Skip()ed rather than failed.
	tests := []struct {
		name           string
		args           []string
		txtar          string
		useDebugBinary bool
	}{
		{
			name:  "basic",
			args:  []string{"pack", basicProject},
			txtar: "basic.txtar",
		},
		{
			name:  "rejects-single-file",
			args:  []string{"pack", filepath.Join(testdataRoot, "rejects-single-file", "main.bal")},
			txtar: "rejects-single-file.txtar",
		},
		{
			name:  "nonexistent-path",
			args:  []string{"pack", missingPath},
			txtar: "nonexistent-path.txtar",
		},
		{
			name:  "not-ballerina-project",
			args:  []string{"pack", filepath.Join(testdataRoot, "not-ballerina-project", "empty")},
			txtar: "not-ballerina-project.txtar",
		},
		{
			name:  "too-many-args",
			args:  []string{"pack", "a", "b"},
			txtar: "too-many-args.txtar",
		},
		{
			name:  "compile-error",
			args:  []string{"pack", filepath.Join(testdataRoot, "compile-error", "project")},
			txtar: "compile-error.txtar",
		},
		{
			name:  "help",
			args:  []string{"pack", "--help"},
			txtar: "help.txtar",
		},
		{
			// These all just check that a debug flag doesn't break an
			// otherwise-successful pack, so they share basic.txtar.
			name:  "pack-with-dump-tokens",
			args:  []string{"pack", basicProject, "--dump-tokens"},
			txtar: "basic.txtar",
		},
		{
			name:  "pack-with-dump-st",
			args:  []string{"pack", basicProject, "--dump-st"},
			txtar: "basic.txtar",
		},
		{
			name:  "pack-with-trace-recovery",
			args:  []string{"pack", basicProject, "--trace-recovery"},
			txtar: "basic.txtar",
		},
		{
			name:  "pack-with-log-file",
			args:  []string{"pack", basicProject, "--dump-tokens", "--log-file={{TMPDIR}}/bal.log"},
			txtar: "basic.txtar",
		},
		{
			name:           "pack-with-prof",
			args:           []string{"pack", basicProject, "--prof"},
			txtar:          "basic.txtar",
			useDebugBinary: true,
		},
		{
			name:  "pack-malformed-manifest",
			args:  []string{"pack", filepath.Join(testdataRoot, "malformed-manifest", "project")},
			txtar: "pack-malformed-manifest.txtar",
		},
		{
			name:  "rejects-workspace",
			args:  []string{"pack", filepath.Join(testdataRoot, "rejects-workspace", "project")},
			txtar: "rejects-workspace.txtar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			binToUse := balBin
			if tt.useDebugBinary {
				if cliIntegrationDebugBalBin == "" {
					t.Skip("debug-tagged bal binary unavailable; skipping debug-only scenario")
				}
				binToUse = cliIntegrationDebugBalBin
			}

			// Substitute the per-scenario {{TMPDIR}} placeholder in each arg
			// with t.TempDir(). The TempDir is created lazily on first use
			// per subtest and cleaned up by the testing package. Only the
			// literal token "{{TMPDIR}}" is recognised — any other "{{...}}"
			// placeholder is rejected to prevent silent typos.
			args := substituteScenarioPlaceholders(t, tt.args)

			assertBalCommandMatchesTxtarFragmentsLoose(t, binToUse, repoRoot, coverDir,
				args, filepath.Join(outputsRoot, tt.txtar))
		})
	}
}

// TestBalPackCorpus_IncludeField exercises the package manifest's `include`
// glob patterns end-to-end through the real `bal pack` binary: it packs
// corpus/cli/testdata/pack/includes/project (include = ["docs/**",
// "assets/*.txt", "!assets/secret.txt"]) and asserts the produced .bala
// contains the matched files and excludes the negated one.
// Java equivalent: TestBalaWriter#testBuildProjectWithIncludes.
func TestBalPackCorpus_IncludeField(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	t.Parallel()
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	workDir := t.TempDir()
	copyDir(t, filepath.Join(repoRoot, "corpus", "cli", "testdata", "pack", "includes", "project"), workDir)

	stdout, stderr, exitCode := runCLICommand(t, balBin, repoRoot, coverDir, "pack", workDir)
	if exitCode != 0 {
		t.Fatalf("bal pack failed: exit=%d\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	balaPath := filepath.Join(workDir, "target", "bala", "testorg-includespkg-any-0.1.0.bala")
	zr, err := zip.OpenReader(balaPath)
	if err != nil {
		t.Fatalf("open bala %s: %v", balaPath, err)
	}
	defer func() { _ = zr.Close() }()

	entries := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		entries[f.Name] = true
	}

	for _, want := range []string{"docs/guide.md", "assets/data.txt"} {
		if !entries[want] {
			t.Errorf("bala missing included entry %q, got entries: %v", want, entryNames(entries))
		}
	}
	if entries["assets/secret.txt"] {
		t.Error("bala contains assets/secret.txt, which the !assets/secret.txt pattern should have excluded")
	}
}

// entryNames returns the keys of a zip-entry presence map, for error messages.
func entryNames(entries map[string]bool) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestBalBuildCorpus exercises `bal build` end-to-end: it runs `bal build` on
// a fixture project, then executes the *produced binary* directly (not the
// bal CLI) and checks its stdout/stderr/exitcode. This is the Phase 1
// "end-to-end working sample" milestone from
// migration-docs/specs/build-command-architecture.md — one fixture uses only
// pure Ballerina, the other calls a native (lib/rt) stdlib function
// (time:utcToString) to prove the embedded-binary output already dispatches
// to native Go code with no extra wiring.
func TestBalBuildCorpus(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	testdataRoot := filepath.Join("corpus", "cli", "testdata", "build")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "build")

	tests := []struct {
		name       string
		projectDir string // relative to repoRoot; a package directory, or (if isFile) a single .bal file
		isFile     bool   // true for a single-file build — target/ lives beside the file, not "inside" it
		pkgName    string
		runTxtar   string // expected stdout/stderr/exitcode of the produced binary
	}{
		{
			name:       "pure-ballerina",
			projectDir: filepath.Join(testdataRoot, "pure-ballerina", "project"),
			pkgName:    "build_pure_sample",
			runTxtar:   "pure-ballerina.txtar",
		},
		{
			name:       "native-stdlib",
			projectDir: filepath.Join(testdataRoot, "native-stdlib", "project"),
			pkgName:    "build_native_sample",
			runTxtar:   "native-stdlib.txtar",
		},
		{
			// Single .bal files are supported, the same as bal run — the
			// default output name is derived from the file's base name
			// (hello.bal -> hello), and target/ lands beside the file.
			name:       "single-file",
			projectDir: filepath.Join(testdataRoot, "single-file", "hello.bal"),
			isFile:     true,
			pkgName:    "hello",
			runTxtar:   "single-file.txtar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			absProjectDir := filepath.Join(repoRoot, tt.projectDir)
			targetDirBase := absProjectDir
			if tt.isFile {
				targetDirBase = filepath.Dir(absProjectDir)
			}
			t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(targetDirBase, "target")) })

			stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
				[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", tt.projectDir)
			if exitCode != 0 {
				t.Fatalf("bal build failed for %s: exit=%d\nstdout:\n%s\nstderr:\n%s", tt.projectDir, exitCode, stdout, stderr)
			}

			// Assert the default output path's shape (<project>/target/bin/<package-name>),
			// not just that some "Created " message was printed — this is the one
			// piece of runBuild's own behavior (as opposed to Pack/TryLoad's framing,
			// covered separately in cli/internal/executable) that this end-to-end
			// test is relied on to verify, since there's no separate in-process test
			// for it. The reported path's prefix isn't predictable up front:
			// runCLICommandWithEnv transparently rewrites testdata-rooted args into
			// a per-test sandbox copy (sandboxCLICommandArgs), so parse the actual
			// path out of stdout instead of assuming repoRoot/targetDirBase.
			binName := tt.pkgName
			if runtime.GOOS == "windows" {
				binName += ".exe"
			}
			wantSuffix := filepath.Join("target", "bin", binName)
			const createdPrefix = "Created "
			idx := strings.Index(stdout, createdPrefix)
			if idx == -1 {
				t.Fatalf("expected bal build stdout to report a %q line, got:\n%s", createdPrefix, stdout)
			}
			binPath := strings.TrimSpace(strings.SplitN(stdout[idx+len(createdPrefix):], "\n", 2)[0])
			if !strings.HasSuffix(binPath, wantSuffix) {
				t.Fatalf("expected bal build to report a path ending in %q, got %q", wantSuffix, binPath)
			}

			builtInfo, err := os.Stat(binPath)
			if err != nil {
				t.Fatalf("expected built binary at %s: %v", binPath, err)
			}

			// The produced binary must be built on the slim balrt stub
			// looked up alongside the bal binary's own distribution
			// (executable.ResolveStub, executable.DistributionDir), not the
			// full bal CLI binary — assert it's meaningfully smaller than
			// bal itself so this fails loudly if that lookup ever silently
			// used the wrong stub instead of erroring.
			balInfo, err := os.Stat(balBin)
			if err != nil {
				t.Fatalf("failed to stat bal binary %s: %v", balBin, err)
			}
			if builtInfo.Size() >= balInfo.Size()/2 {
				t.Fatalf("built binary %s (%d bytes) is not meaningfully smaller than bal (%d bytes); expected the slim balrt stub to be used",
					binPath, builtInfo.Size(), balInfo.Size())
			}

			runOut, runErr, runExit := runCLICommand(t, binPath, repoRoot, coverDir)
			runOut = test_util.NormalizeNewlines(runOut)
			runErr = test_util.NormalizeNewlines(runErr)

			expectedOut, expectedErr, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(filepath.Join(outputsRoot, tt.runTxtar))
			if err != nil {
				t.Fatalf("failed to parse txtar file for %s: %v", tt.name, err)
			}
			if runOut != expectedOut {
				t.Fatalf("unexpected stdout from built binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedOut, runOut))
			}
			if runErr != expectedErr {
				t.Fatalf("unexpected stderr from built binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedErr, runErr))
			}
			if strconv.Itoa(runExit) != expectedExit {
				t.Fatalf("unexpected exit code from built binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedExit, strconv.Itoa(runExit)))
			}
		})
	}
}

// substituteScenarioPlaceholders replaces the token "{{TMPDIR}}" in each arg
// with a fresh t.TempDir() (one TempDir per scenario, reused across args).
// Any other "{{...}}" token is treated as an unknown placeholder and fails
// the test — only TMPDIR is supported today.
func substituteScenarioPlaceholders(t *testing.T, args []string) []string {
	t.Helper()
	const tmpdirToken = "{{TMPDIR}}"
	var tmpDir string
	out := make([]string, len(args))
	for i, a := range args {
		if strings.Contains(a, tmpdirToken) {
			if tmpDir == "" {
				tmpDir = t.TempDir()
			}
			a = strings.ReplaceAll(a, tmpdirToken, tmpDir)
		}
		// Detect any leftover placeholder of the form "{{NAME}}" — these are
		// unsupported and must fail loudly rather than be passed verbatim.
		if openIdx := strings.Index(a, "{{"); openIdx != -1 {
			closeIdx := strings.Index(a[openIdx:], "}}")
			if closeIdx != -1 {
				t.Fatalf("unsupported scenario placeholder %q in arg %q (only {{TMPDIR}} is supported)",
					a[openIdx:openIdx+closeIdx+2], a)
			}
		}
		out[i] = a
	}
	return out
}

// normalizePaths replaces the absolute repo root in CLI output with the
// portable placeholder <ROOT> so txtar fixtures are machine-independent.
// It is applied to both the --update write path and the comparison path so
// the txtar content and the actual output are compared on equal footing.
func normalizePaths(s, repoRoot string) string {
	if repoRoot == "" {
		return s
	}
	return strings.ReplaceAll(s, repoRoot, "<ROOT>")
}

// assertBalCommandMatchesTxtarFragmentsLoose is like assertBalCommandMatchesTxtarFragmentsForBinary
// but uses fragment (substring) matching for both stdout and stderr. This is needed when stderr
// contains machine-specific absolute paths that cannot be captured exactly in a txtar fixture.
func assertBalCommandMatchesTxtarFragmentsLoose(t *testing.T, balBin, repoRoot, coverDir string, args []string, txtarPath string) {
	t.Helper()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}

	stdout, stderr, exitCode := runCLICommand(t, balBin, repoRoot, coverDir, args...)
	stdout = normalizePaths(test_util.NormalizeNewlines(stdout), repoRoot)
	stderr = normalizePaths(test_util.NormalizeNewlines(stderr), repoRoot)

	expectedStdoutFragments, expectedStderrFragments, expectedExitCode, err := test_util.LoadTxtarStdoutStderrExitcode(txtarPath)
	if err != nil {
		t.Fatalf("failed to parse txtar file %s: %v", txtarPath, err)
	}

	if strconv.Itoa(exitCode) != expectedExitCode {
		t.Fatalf("unexpected exit code for command %q with expected file %s\n%s\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), txtarPath,
			test_util.FormatExpectedGot(expectedExitCode, strconv.Itoa(exitCode)), stdout, stderr)
	}

	combinedOut := stdout + "\n" + stderr
	for _, fragment := range strings.Split(expectedStdoutFragments, "\n") {
		if strings.TrimSpace(fragment) == "" {
			continue
		}
		if !strings.Contains(combinedOut, fragment) {
			t.Fatalf("output missing expected stdout fragment %q for command %q with expected file %s\nstdout:\n%s\nstderr:\n%s",
				fragment, strings.Join(args, " "), txtarPath, stdout, stderr)
		}
	}
	for _, fragment := range strings.Split(expectedStderrFragments, "\n") {
		if strings.TrimSpace(fragment) == "" {
			continue
		}
		if !strings.Contains(stderr, fragment) {
			t.Fatalf("stderr missing expected fragment %q for command %q with expected file %s\nstdout:\n%s\nstderr:\n%s",
				fragment, strings.Join(args, " "), txtarPath, stdout, stderr)
		}
	}
}

// assertBalCommandMatchesTxtarFragmentsLooseWithEnv is like
// assertBalCommandMatchesTxtarFragmentsLoose but appends extraEnv on top of
// the process environment (via runCLICommandWithEnv) — needed for build
// scenarios that require a specific BAL_ENV (e.g. one with no runner stub
// installed, to exercise the missing-stub error path).
func assertBalCommandMatchesTxtarFragmentsLooseWithEnv(t *testing.T, balBin, repoRoot, coverDir string, extraEnv, args []string, txtarPath string) {
	t.Helper()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir, extraEnv, args...)
	stdout = test_util.NormalizeNewlines(stdout)
	stderr = test_util.NormalizeNewlines(stderr)

	expectedStdoutFragments, expectedStderrFragments, expectedExitCode, err := test_util.LoadTxtarStdoutStderrExitcode(txtarPath)
	if err != nil {
		t.Fatalf("failed to parse txtar file %s: %v", txtarPath, err)
	}

	if strconv.Itoa(exitCode) != expectedExitCode {
		t.Fatalf("unexpected exit code for command %q with expected file %s\n%s",
			strings.Join(args, " "), txtarPath,
			test_util.FormatExpectedGot(expectedExitCode, strconv.Itoa(exitCode)))
	}

	combinedOut := stdout + "\n" + stderr
	for _, fragment := range strings.Split(expectedStdoutFragments, "\n") {
		if strings.TrimSpace(fragment) == "" {
			continue
		}
		if !strings.Contains(combinedOut, fragment) {
			t.Fatalf("output missing expected stdout fragment %q for command %q with expected file %s\nstdout:\n%s\nstderr:\n%s",
				fragment, strings.Join(args, " "), txtarPath, stdout, stderr)
		}
	}
	for _, fragment := range strings.Split(expectedStderrFragments, "\n") {
		if strings.TrimSpace(fragment) == "" {
			continue
		}
		if !strings.Contains(stderr, fragment) {
			t.Fatalf("stderr missing expected fragment %q for command %q with expected file %s\nstdout:\n%s\nstderr:\n%s",
				fragment, strings.Join(args, " "), txtarPath, stdout, stderr)
		}
	}
}

// TestBalBuildScenarios covers bal build's error/edge-case paths — as
// opposed to TestBalBuildCorpus, which covers the successful build+run
// paths — mirroring TestBalPackCorpus's scenario-table pattern rather than
// an in-process cli/cmd/build_test.go (see corpus/cli_integration_test.go's
// TestBalPackCorpus doc comment for why: keeps subprocess coverage flowing
// into the cli/cmd profile, consistent with how pack's equivalent in-process
// suite was replaced).
func TestBalBuildScenarios(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	testdataRoot := filepath.Join("corpus", "cli", "testdata", "build")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "build")

	missingPath := filepath.Join(testdataRoot, "this-path-does-not-exist")

	tests := []struct {
		name                 string
		args                 []string
		txtar                string
		useBalWithoutRuntime bool // dispatch through a bal binary with no rt/ sibling
		projectTargetDir     string
	}{
		{
			name:  "nonexistent-path",
			args:  []string{"build", missingPath},
			txtar: "nonexistent-path.txtar",
		},
		{
			name:  "not-a-bal-file",
			args:  []string{"build", filepath.Join(testdataRoot, "not-a-bal-file", "notes.txt")},
			txtar: "not-a-bal-file.txtar",
		},
		{
			name:             "compile-error",
			args:             []string{"build", filepath.Join(testdataRoot, "compile-error", "project")},
			txtar:            "compile-error.txtar",
			projectTargetDir: filepath.Join(repoRoot, testdataRoot, "compile-error", "project", "target"),
		},
		{
			name:  "workspace-with-custom-output",
			args:  []string{"build", filepath.Join(testdataRoot, "workspace", "project"), "-o", "out"},
			txtar: "workspace-with-custom-output.txtar",
		},
		{
			// Invalid version in Ballerina.toml is a load-phase (manifest)
			// error, not a compile-phase one: it's reported via
			// result.Diagnostics(), distinct from compile-error's semantic
			// error via compilation.DiagnosticResult().
			name:  "invalid-package-version",
			args:  []string{"build", filepath.Join(testdataRoot, "invalid-package-version", "project")},
			txtar: "invalid-package-version.txtar",
		},
		{
			name:                 "missing-runtime",
			args:                 []string{"build", filepath.Join(testdataRoot, "pure-ballerina", "project")},
			txtar:                "missing-runtime.txtar",
			useBalWithoutRuntime: true,
		},
		{
			// Genuine TOML syntax error: projects.Load itself returns an
			// error, distinct from invalid-package-version's diagnostic
			// (Load succeeds, but its DiagnosticResult has errors).
			name:  "malformed-manifest",
			args:  []string{"build", filepath.Join(testdataRoot, "malformed-manifest", "project")},
			txtar: "malformed-manifest.txtar",
		},
		{
			// These four scenarios all just check that a debug flag doesn't
			// break an otherwise-successful build, so they share one txtar.
			name:  "build-with-dump-tokens",
			args:  []string{"build", filepath.Join(testdataRoot, "pure-ballerina", "project"), "--dump-tokens"},
			txtar: "build-debug-flag-success.txtar",
		},
		{
			name:  "build-with-dump-st",
			args:  []string{"build", filepath.Join(testdataRoot, "pure-ballerina", "project"), "--dump-st"},
			txtar: "build-debug-flag-success.txtar",
		},
		{
			name:  "build-with-trace-recovery",
			args:  []string{"build", filepath.Join(testdataRoot, "pure-ballerina", "project"), "--trace-recovery"},
			txtar: "build-debug-flag-success.txtar",
		},
		{
			name:  "build-with-log-file",
			args:  []string{"build", filepath.Join(testdataRoot, "pure-ballerina", "project"), "--dump-tokens", "--log-file={{TMPDIR}}/bal.log"},
			txtar: "build-debug-flag-success.txtar",
		},
		{
			// os.Create doesn't create parent directories, so a --log-file
			// under a nonexistent subdirectory of the fresh TMPDIR fails.
			name:  "build-log-file-blocked",
			args:  []string{"build", filepath.Join(testdataRoot, "pure-ballerina", "project"), "--dump-tokens", "--log-file={{TMPDIR}}/nonexistent-subdir/bal.log"},
			txtar: "build-log-file-blocked.txtar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.projectTargetDir != "" {
				t.Cleanup(func() { _ = os.RemoveAll(tt.projectTargetDir) })
			}
			binToUse := balBin
			if tt.useBalWithoutRuntime {
				binToUse = balBinaryWithoutRuntimeStub(t, repoRoot, coverDir)
			}
			args := substituteScenarioPlaceholders(t, tt.args)
			assertBalCommandMatchesTxtarFragmentsLooseWithEnv(t, binToUse, repoRoot, coverDir,
				[]string{"BAL_ENV=" + cliIntegrationBalEnv}, args, filepath.Join(outputsRoot, tt.txtar))
		})
	}
}

// TestBalBuildWorkspace covers `bal build <workspace-root>`: every member
// must be built to its own target/bin/<name>. Each member also depends on
// mockorg/leafpkg via repository = "local" (missing locally, falls back to
// central with a warning), so this also checks per-member diagnostics: the
// warning must appear once per member and the build must still succeed.
func TestBalBuildWorkspace(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	testdataRoot := filepath.Join("corpus", "cli", "testdata", "build")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "build")
	workspaceDir := filepath.Join(testdataRoot, "workspace", "project")

	// leafpkg only exists in this mock central cache, not cliIntegrationBalEnv.
	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	copyDir(t, filepath.Join(repoRoot, "projects", "testdata", "repo", "bala"), centralCache)

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + tempHome}, "build", workspaceDir)
	if exitCode != 0 {
		t.Fatalf("bal build on a workspace failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}

	const wantWarning = "dependency mockorg/leafpkg:1.0.0 cannot be found in the 'local' repository. falling back to default repositories"
	if got := strings.Count(stderr, wantWarning); got != 2 {
		t.Errorf("expected the local-repo-miss warning once per workspace member (2 total), got %d\nstderr:\n%s", got, stderr)
	}

	members := []struct {
		name     string
		runTxtar string
	}{
		{"pkga", "workspace-pkga.txtar"},
		{"pkgb", "workspace-pkgb.txtar"},
	}

	// Members build in manifest order, so parse each "Created " line in
	// turn rather than assuming a fixed path (sandboxCLICommandArgs rewrites
	// testdata-rooted args to a per-test sandbox copy).
	remaining := stdout
	const createdPrefix = "Created "
	for _, m := range members {
		idx := strings.Index(remaining, createdPrefix)
		if idx == -1 {
			t.Fatalf("expected a %q line for member %s, remaining stdout:\n%s", createdPrefix, m.name, remaining)
		}
		rest := remaining[idx+len(createdPrefix):]
		line, after, _ := strings.Cut(rest, "\n")
		binPath := strings.TrimSpace(line)
		wantSuffix := filepath.Join(m.name, "target", "bin", hostExeSuffix(m.name))
		if !strings.HasSuffix(binPath, wantSuffix) {
			t.Fatalf("expected bal build to report a path ending in %q for member %s, got %q", wantSuffix, m.name, binPath)
		}
		remaining = after

		if _, err := os.Stat(binPath); err != nil {
			t.Fatalf("expected built binary for member %s at %s: %v", m.name, binPath, err)
		}

		runOut, runErr, runExit := runCLICommand(t, binPath, repoRoot, coverDir)
		runOut = test_util.NormalizeNewlines(runOut)
		runErr = test_util.NormalizeNewlines(runErr)

		expectedOut, expectedErr, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(filepath.Join(outputsRoot, m.runTxtar))
		if err != nil {
			t.Fatalf("failed to parse txtar file for member %s: %v", m.name, err)
		}
		if runOut != expectedOut {
			t.Fatalf("unexpected stdout from member %s binary %s\n%s", m.name, binPath, test_util.FormatExpectedGot(expectedOut, runOut))
		}
		if runErr != expectedErr {
			t.Fatalf("unexpected stderr from member %s binary %s\n%s", m.name, binPath, test_util.FormatExpectedGot(expectedErr, runErr))
		}
		if strconv.Itoa(runExit) != expectedExit {
			t.Fatalf("unexpected exit code from member %s binary %s\n%s", m.name, binPath, test_util.FormatExpectedGot(expectedExit, strconv.Itoa(runExit)))
		}
	}
}

// TestBalBuildWorkspaceCrossDependency covers a workspace member (pkga)
// depending on a sibling member (pkgb) by org/name, resolved via
// workspaceRepository. Regression test: runBuild's workspace loop reloads
// each member into its own fresh Environment (to avoid sharing one
// CompilerEnvironment across members — see the comment in runBuild), and
// an earlier version of that reload loaded just the member's own directory
// in isolation, which lost workspaceRepository entirely and broke this
// exact scenario ("Unknown import"). The fix reloads the whole workspace
// per member instead, so sibling resolution stays intact.
func TestBalBuildWorkspaceCrossDependency(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	testdataRoot := filepath.Join("corpus", "cli", "testdata", "build")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "build")
	workspaceDir := filepath.Join(testdataRoot, "workspace-cross-dependency", "project")

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", workspaceDir)
	if exitCode != 0 {
		t.Fatalf("bal build on a workspace with a cross-member dependency failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}

	idx := strings.Index(stdout, "Created ")
	if idx == -1 {
		t.Fatalf("expected a 'Created' line for pkga, got stdout:\n%s", stdout)
	}
	rest := stdout[idx+len("Created "):]
	line, _, _ := strings.Cut(rest, "\n")
	binPath := strings.TrimSpace(line)
	wantSuffix := filepath.Join("pkga", "target", "bin", hostExeSuffix("pkga"))
	if !strings.HasSuffix(binPath, wantSuffix) {
		t.Fatalf("expected the first built member to be pkga (path ending in %q), got %q", wantSuffix, binPath)
	}

	runOut, runErr, runExit := runCLICommand(t, binPath, repoRoot, coverDir)
	runOut = test_util.NormalizeNewlines(runOut)
	runErr = test_util.NormalizeNewlines(runErr)
	expectedOut, expectedErr, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(filepath.Join(outputsRoot, "workspace-cross-dependency-pkga.txtar"))
	if err != nil {
		t.Fatalf("failed to parse golden txtar: %v", err)
	}
	if runOut != expectedOut {
		t.Fatalf("unexpected stdout from pkga binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedOut, runOut))
	}
	if runErr != expectedErr {
		t.Fatalf("unexpected stderr from pkga binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedErr, runErr))
	}
	if strconv.Itoa(runExit) != expectedExit {
		t.Fatalf("unexpected exit code from pkga binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedExit, strconv.Itoa(runExit)))
	}
}

// TestBalBuildFromWorkspaceMemberDirectory covers running bal build with the
// process cwd set to a workspace member's own directory (args empty, so
// path defaults to "."), rather than passing the member's path from outside
// the workspace. Regression test: build.go used to load straight from cwd
// with no workspace-root detection, so workspaceRepository-based sibling
// resolution (pkga importing pkgb by org/name) was invisible and failed
// with "Unknown import" — bal run already handled this correctly via
// findWorkspaceRoot.
func TestBalBuildFromWorkspaceMemberDirectory(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "build")

	srcWorkspace := filepath.Join(repoRoot, "corpus", "cli", "testdata", "build", "workspace-cross-dependency", "project")
	workspaceDir := t.TempDir()
	copyDir(t, srcWorkspace, workspaceDir)
	memberDir := filepath.Join(workspaceDir, "pkga")

	env := append(os.Environ(), "BAL_ENV="+cliIntegrationBalEnv)
	if coverDir != "" {
		commandCoverDir := t.TempDir()
		env = append(env, "GOCOVERDIR="+commandCoverDir)
		defer mergeCLICoverageDir(t, commandCoverDir, coverDir)
	}
	stdout, stderr, exitCode := runNativeCLICommandWithEnv(t, balBin, memberDir, []string{"build"}, env)
	if exitCode != 0 {
		t.Fatalf("bal build from inside a workspace member directory failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}

	idx := strings.Index(stdout, "Created ")
	if idx == -1 {
		t.Fatalf("expected a 'Created' line, got stdout:\n%s", stdout)
	}
	rest := stdout[idx+len("Created "):]
	line, _, _ := strings.Cut(rest, "\n")
	binPath := strings.TrimSpace(line)
	wantSuffix := filepath.Join("pkga", "target", "bin", hostExeSuffix("pkga"))
	if !strings.HasSuffix(binPath, wantSuffix) {
		t.Fatalf("expected the built member to be pkga (path ending in %q), got %q", wantSuffix, binPath)
	}

	runOut, runErr, runExit := runCLICommand(t, binPath, repoRoot, coverDir)
	runOut = test_util.NormalizeNewlines(runOut)
	runErr = test_util.NormalizeNewlines(runErr)
	expectedOut, expectedErr, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(filepath.Join(outputsRoot, "workspace-cross-dependency-pkga.txtar"))
	if err != nil {
		t.Fatalf("failed to parse golden txtar: %v", err)
	}
	if runOut != expectedOut {
		t.Fatalf("unexpected stdout from pkga binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedOut, runOut))
	}
	if runErr != expectedErr {
		t.Fatalf("unexpected stderr from pkga binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedErr, runErr))
	}
	if strconv.Itoa(runExit) != expectedExit {
		t.Fatalf("unexpected exit code from pkga binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedExit, strconv.Itoa(runExit)))
	}
}

// TestBalPackFromWorkspaceMemberDirectory is
// TestBalBuildFromWorkspaceMemberDirectory's counterpart for bal pack: pack
// already rejects being pointed at a workspace root directly, but a member's
// own directory (cwd inside it, no positional arg) must still resolve
// sibling dependencies via the workspace root rather than failing with
// "Unknown import".
func TestBalPackFromWorkspaceMemberDirectory(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	srcWorkspace := filepath.Join(repoRoot, "corpus", "cli", "testdata", "build", "workspace-cross-dependency", "project")
	workspaceDir := t.TempDir()
	copyDir(t, srcWorkspace, workspaceDir)
	memberDir := filepath.Join(workspaceDir, "pkga")

	env := append(os.Environ(), "BAL_ENV="+cliIntegrationBalEnv)
	if coverDir != "" {
		commandCoverDir := t.TempDir()
		env = append(env, "GOCOVERDIR="+commandCoverDir)
		defer mergeCLICoverageDir(t, commandCoverDir, coverDir)
	}
	stdout, stderr, exitCode := runNativeCLICommandWithEnv(t, balBin, memberDir, []string{"pack"}, env)
	if exitCode != 0 {
		t.Fatalf("bal pack from inside a workspace member directory failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "Created ") || !strings.Contains(stdout, ".bala") {
		t.Fatalf("expected a 'Created ... .bala' line, got stdout:\n%s", stdout)
	}
}

// =============================================================================
// bal push corpus tests
// =============================================================================
//
// TestBalPush* exercises `bal push` end-to-end through the coverage-aware CLI
// harness, mirroring (and replacing) the in-process cli/cmd/push_test.go
// suite so subprocess coverage flows into the cli/cmd profile the same way
// TestBalPackCorpus/TestBalBuildCorpus already do for pack/build. Each test
// gets its own isolated BAL_ENV (a fresh t.TempDir()) since push writes into
// <BAL_ENV>/repositories/local/bala/..., unlike the shared cliIntegrationBalEnv
// build/pack use read-only.

func TestBalPushExplicitPath(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml":           []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml":      []byte("[package]\norg = \"testorg\"\nname = \"myproject\"\nversion = \"0.1.0\"\n"),
		"Dependencies.toml":   []byte("[ballerina]\ndependencies-toml-version = \"2\"\n"),
		"main.bal":            []byte("public function main() {}\n"),
		"modules/sub/sub.bal": []byte("public function sub() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "testorg-myproject-any-0.1.0.bala", entries)

	stdout, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err != nil {
		t.Fatalf("push failed: %v\nstderr: %s", err, stderr)
	}

	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"testorg", "myproject", "0.1.0", "any")

	if !strings.Contains(stdout, "Pushed ") ||
		!strings.Contains(stdout, balaPath) ||
		!strings.Contains(stdout, destDir) {
		t.Errorf("expected stdout to announce push of %s to %s, got: %s",
			balaPath, destDir, stdout)
	}

	for name, want := range entries {
		got, err := os.ReadFile(filepath.Join(destDir, filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("expected extracted entry %s: %v", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("entry %s: contents differ\nwant: %q\ngot:  %q",
				name, want, got)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join(destDir, projects.BallerinaTomlFile))
	if err != nil {
		t.Fatalf("expected %s extracted: %v", projects.BallerinaTomlFile, err)
	}
	for _, want := range []string{
		`org = "testorg"`,
		`name = "myproject"`,
		`version = "0.1.0"`,
	} {
		if !strings.Contains(string(manifestBytes), want) {
			t.Errorf("expected manifest to contain %q, got: %s", want, manifestBytes)
		}
	}
}

// Filename intentionally disagrees with the manifest; destination must
// follow the manifest, not the filename.
func TestBalPushArbitraryFilename(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"testpkg\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "foo-bar-any-9.9.9.zip", entries)

	stdout, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err != nil {
		t.Fatalf("push with non-.bala extension failed: %v\nstderr: %s", err, stderr)
	}

	manifestDest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"mockorg", "testpkg", "1.0.0", "any")
	if _, err := os.Stat(manifestDest); err != nil {
		t.Fatalf("expected destination at manifest path %s, stat err: %v",
			manifestDest, err)
	}
	if !strings.Contains(stdout, manifestDest) {
		t.Errorf("expected stdout to mention manifest-derived destination %s, got: %s",
			manifestDest, stdout)
	}

	filenameDest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"foo", "bar", "9.9.9", "any")
	if _, err := os.Stat(filenameDest); !os.IsNotExist(err) {
		t.Errorf("expected filename-derived destination %s absent, stat err: %v",
			filenameDest, err)
	}

	for name := range entries {
		if _, err := os.Stat(filepath.Join(manifestDest, name)); err != nil {
			t.Errorf("expected entry %s extracted: %v", name, err)
		}
	}
}

func TestBalPushAutoDiscovery(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	projectDir := t.TempDir()
	balaDir := filepath.Join(projectDir, projects.TargetDir, "bala")
	if err := os.MkdirAll(balaDir, 0o755); err != nil {
		t.Fatalf("mkdir target/bala: %v", err)
	}

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"acme\"\nname = \"widgets\"\nversion = \"1.2.3\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := filepath.Join(balaDir, "acme-widgets-any-1.2.3.bala")
	writePushBalaFile(t, balaPath, entries)

	stdout, stderr, err := executePushCLI(t, projectDir, balaEnv, "--repository", "local")
	if err != nil {
		t.Fatalf("push failed: %v\nstderr: %s", err, stderr)
	}

	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"acme", "widgets", "1.2.3", "any")
	if !strings.Contains(stdout, destDir) {
		t.Errorf("expected stdout to mention destination %s, got: %s",
			destDir, stdout)
	}

	for name := range entries {
		if _, err := os.Stat(filepath.Join(destDir, name)); err != nil {
			t.Errorf("expected entry %s extracted: %v", name, err)
		}
	}
}

func TestBalPushOverwrite(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"acme", "widgets", "0.1.0", "any")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	junkPath := filepath.Join(destDir, "stale.txt")
	if err := os.WriteFile(junkPath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"acme\"\nname = \"widgets\"\nversion = \"0.1.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "acme-widgets-any-0.1.0.bala", entries)

	if _, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local"); err != nil {
		t.Fatalf("push failed: %v\nstderr: %s", err, stderr)
	}

	if _, err := os.Stat(junkPath); !os.IsNotExist(err) {
		t.Errorf("expected stale file removed, stat err: %v", err)
	}
	for name := range entries {
		if _, err := os.Stat(filepath.Join(destDir, name)); err != nil {
			t.Errorf("expected entry %s extracted: %v", name, err)
		}
	}
}

func TestBalPushMultipleBalas(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	projectDir := t.TempDir()
	balaDir := filepath.Join(projectDir, projects.TargetDir, "bala")
	if err := os.MkdirAll(balaDir, 0o755); err != nil {
		t.Fatalf("mkdir target/bala: %v", err)
	}
	for _, name := range []string{
		"acme-widgets-any-0.1.0.bala",
		"acme-widgets-any-0.2.0.bala",
	} {
		writePushBalaFile(t, filepath.Join(balaDir, name), map[string][]byte{
			"Bala.toml": []byte("[build]\nplatform = \"any\"\n"),
		})
	}

	_, stderr, err := executePushCLI(t, projectDir, balaEnv, "--repository", "local")
	if err == nil {
		t.Fatal("expected ambiguity error for multiple bala files, got success")
	}
	if !strings.Contains(stderr, "multiple") {
		t.Errorf("expected 'multiple' in stderr, got: %s", stderr)
	}
}

func TestBalPushNoBalaInTarget(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()
	projectDir := t.TempDir()

	_, stderr, err := executePushCLI(t, projectDir, balaEnv, "--repository", "local")
	if err == nil {
		t.Fatal("expected error when no bala is present, got success")
	}
	if !strings.Contains(stderr, "no .bala") &&
		!strings.Contains(stderr, ".bala file found") {
		t.Errorf("expected 'no .bala' error, got stderr: %s", stderr)
	}
}

// target/bala exists but is empty, unlike TestBalPushNoBalaInTarget where
// target/bala itself is missing.
func TestBalPushEmptyTargetBalaDir(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()
	projectDir := t.TempDir()
	balaDir := filepath.Join(projectDir, projects.TargetDir, "bala")
	if err := os.MkdirAll(balaDir, 0o755); err != nil {
		t.Fatalf("mkdir target/bala: %v", err)
	}

	_, stderr, err := executePushCLI(t, projectDir, balaEnv, "--repository", "local")
	if err == nil {
		t.Fatal("expected error when target/bala is empty, got success")
	}
	if !strings.Contains(stderr, "no .bala file found under") {
		t.Errorf("expected 'no .bala file found under' error, got stderr: %s", stderr)
	}
}

func TestBalPushDirectoryAsBalaPath(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	dir := t.TempDir()
	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, dir, "--repository", "local")
	if err == nil {
		t.Fatal("expected directory-rejection error, got success")
	}
	if !strings.Contains(stderr, "push requires a bala file; got directory") {
		t.Errorf("expected directory-rejection error, got: %s", stderr)
	}
}

func TestBalPushExplicitPathDoesNotExist(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	missing := filepath.Join(t.TempDir(), "does-not-exist.bala")
	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, missing, "--repository", "local")
	if err == nil {
		t.Fatal("expected error for a nonexistent explicit path, got success")
	}
	if !strings.Contains(stderr, "invalid bala path") {
		t.Errorf("expected 'invalid bala path' error, got: %s", stderr)
	}
}

func TestBalPushAutoDiscoverySkipsDirectory(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	projectDir := t.TempDir()
	balaDir := filepath.Join(projectDir, projects.TargetDir, "bala")
	if err := os.MkdirAll(filepath.Join(balaDir, "staging"), 0o755); err != nil {
		t.Fatalf("mkdir target/bala/staging: %v", err)
	}
	writePushBalaFile(t, filepath.Join(balaDir, "acme-widgets-any-0.1.0.bala"),
		validPushBalaEntries())

	_, stderr, err := executePushCLI(t, projectDir, balaEnv, "--repository", "local")
	if err != nil {
		t.Fatalf("expected push to succeed, got error: %v\nstderr: %s", err, stderr)
	}
	dest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"acme", "widgets", "1.2.3", "any")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("destination not created at %s: %v", dest, err)
	}
}

// bal pack never emits explicit directory entries, but a third-party tool
// might; guards the IsDir branch in extractZipEntry.
func TestBalPushZipWithDirectoryEntry(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	tmp := t.TempDir()
	balaPath := filepath.Join(tmp, "mockorg-foo-any-1.0.0.bala")
	writePushBalaFileWithDirEntries(t, balaPath, []pushBalaEntry{
		{"Bala.toml", []byte("[build]\nplatform = \"any\"\n")},
		{"Ballerina.toml", []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n")},
		{"modules/", nil},
		{"modules/sub/lib.bal", []byte("public function libfn() {}\n")},
	})

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err != nil {
		t.Fatalf("expected push to succeed, got error: %v\nstderr: %s", err, stderr)
	}

	dest := filepath.Join(balaEnv, "repositories", "local", "bala",
		"mockorg", "foo", "1.0.0", "any")
	if info, err := os.Stat(filepath.Join(dest, "modules")); err != nil || !info.IsDir() {
		t.Errorf("expected modules/ directory at destination, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "modules", "sub", "lib.bal")); err != nil {
		t.Errorf("expected modules/sub/lib.bal at destination, stat err: %v", err)
	}
}

// Entries are written in a fixed order so the malicious one is encountered first.
func TestBalPushZipSlip(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	tmp := t.TempDir()
	balaPath := filepath.Join(tmp, "evil-pkg-any-0.1.0.bala")
	writePushOrderedBalaFile(t, balaPath, []pushBalaEntry{
		{"../evil", []byte("pwned")},
		{"Bala.toml", []byte("[build]\nplatform = \"any\"\n")},
		{"Ballerina.toml", []byte("[package]\norg = \"evil\"\nname = \"pkg\"\nversion = \"0.1.0\"\n")},
	})

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected zip-slip rejection, got success")
	}
	if !strings.Contains(stderr, "zip-slip") &&
		!strings.Contains(stderr, "escapes destination") {
		t.Errorf("expected zip-slip error in stderr, got: %s", stderr)
	}

	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"evil", "pkg", "0.1.0", "any")
	escaped := filepath.Join(filepath.Dir(destDir), "evil")
	if _, err := os.Stat(escaped); !os.IsNotExist(err) {
		t.Errorf("expected zip-slip target absent, stat err: %v", err)
	}
}

// A literal backslash is a separate zip-slip guard from the "../" check
// TestBalPushZipSlip exercises.
func TestBalPushZipSlipBackslash(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	tmp := t.TempDir()
	balaPath := filepath.Join(tmp, "evil-pkg-any-0.1.0.bala")
	writePushOrderedBalaFile(t, balaPath, []pushBalaEntry{
		{`..\evil`, []byte("pwned")},
		{"Bala.toml", []byte("[build]\nplatform = \"any\"\n")},
		{"Ballerina.toml", []byte("[package]\norg = \"evil\"\nname = \"pkg\"\nversion = \"0.1.0\"\n")},
	})

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected zip-slip rejection, got success")
	}
	if !strings.Contains(stderr, "zip-slip") {
		t.Errorf("expected zip-slip error in stderr, got: %s", stderr)
	}

	// destParent (.../evil/pkg/0.1.0) is created before extraction starts;
	// only destDir itself (the "any" platform leaf) must stay absent.
	destDir := filepath.Join(balaEnv, "repositories", "local", "bala",
		"evil", "pkg", "0.1.0", "any")
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Errorf("expected destination %s absent, stat err: %v", destDir, err)
	}
}

func TestBalPushDestinationParentIsFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	// destDir's parent (.../acme/widgets/1.2.3) collides with a regular file,
	// so os.MkdirAll can't create it.
	destParent := filepath.Join(balaEnv, "repositories", "local", "bala", "acme", "widgets", "1.2.3")
	if err := os.MkdirAll(filepath.Dir(destParent), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(destParent), err)
	}
	if err := os.WriteFile(destParent, []byte("blocking file"), 0o644); err != nil {
		t.Fatalf("write %s: %v", destParent, err)
	}

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"acme\"\nname = \"widgets\"\nversion = \"1.2.3\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "acme-widgets-any-1.2.3.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected error when destination parent collides with a file, got success")
	}
	if !strings.Contains(stderr, "create destination parent") {
		t.Errorf("expected 'create destination parent' error, got: %s", stderr)
	}
}

func TestBalPushNotAZipArchive(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	dir := t.TempDir()
	balaPath := filepath.Join(dir, "mockorg-foo-any-1.0.0.bala")
	if err := os.WriteFile(balaPath, []byte("not a zip archive"), 0o644); err != nil {
		t.Fatalf("write %s: %v", balaPath, err)
	}

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected error for a non-zip bala file, got success")
	}
	if !strings.Contains(stderr, "open bala archive") {
		t.Errorf("expected 'open bala archive' error, got: %s", stderr)
	}
}

func TestBalPushMissingBallerinaToml(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml": []byte("[build]\nplatform = \"any\"\n"),
		"main.bal":  []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-manifest error, got success")
	}
	if !strings.Contains(stderr, "missing Ballerina.toml") {
		t.Errorf("expected missing-manifest error, got: %s", stderr)
	}

	// Identity must be read before any destination is created.
	repoRoot := filepath.Join(balaEnv, "repositories", "local", "bala")
	if entries, err := os.ReadDir(repoRoot); err == nil && len(entries) != 0 {
		t.Errorf("expected repo root untouched, found entries: %v", entries)
	}
}

func TestBalPushMissingBalaToml(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-Bala.toml error, got success")
	}
	if !strings.Contains(stderr, "missing Bala.toml") {
		t.Errorf("expected missing-Bala.toml error, got: %s", stderr)
	}

	repoRoot := filepath.Join(balaEnv, "repositories", "local", "bala")
	if entries, err := os.ReadDir(repoRoot); err == nil && len(entries) != 0 {
		t.Errorf("expected repo root untouched, found entries: %v", entries)
	}
}

func TestBalPushMalformedBallerinaToml(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("this is = = not = valid toml ["),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected parse error, got success")
	}
	if !strings.Contains(stderr, "parse") && !strings.Contains(stderr, "Ballerina.toml") {
		t.Errorf("expected parse error referencing Ballerina.toml, got: %s", stderr)
	}
}

func TestBalPushMissingOrgField(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field org") {
		t.Errorf("expected missing-org-field error, got: %s", stderr)
	}
}

func TestBalPushMissingNameField(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-name-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field name") {
		t.Errorf("expected missing-name-field error, got: %s", stderr)
	}
}

func TestBalPushMissingVersionField(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-version-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field version") {
		t.Errorf("expected missing-version-field error, got: %s", stderr)
	}
}

func TestBalPushMalformedBalaToml(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		// Unterminated table header → parser fails.
		"Bala.toml":      []byte("[build\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected parse error, got success")
	}
	if !strings.Contains(stderr, "parse") || !strings.Contains(stderr, "Bala.toml") {
		t.Errorf("expected parse error referencing Bala.toml, got: %s", stderr)
	}
}

func TestBalPushMissingPlatformField(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaEnv := t.TempDir()

	entries := map[string][]byte{
		"Bala.toml":      []byte("[build]\nschema_version = \"4\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"mockorg\"\nname = \"foo\"\nversion = \"1.0.0\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
	balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

	_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
	if err == nil {
		t.Fatal("expected missing-platform-field error, got success")
	}
	if !strings.Contains(stderr, "missing required field platform") {
		t.Errorf("expected missing-platform-field error, got: %s", stderr)
	}
}

// Manifest fields flow straight into filepath.Join(...) for the destination
// and into os.RemoveAll before extraction even starts, so "." / ".." /
// embedded separators must be rejected rather than trusted.
func TestBalPushMaliciousManifestFields(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	tests := []struct {
		name             string
		org, pkg, ver    string
		platform         string
		wantErrSubstring string
	}{
		{"org traversal", "../../../../tmp", "foo", "1.0.0", "any", `invalid org "../../../../tmp"`},
		{"name traversal", "mockorg", "..", "1.0.0", "any", `invalid name ".."`},
		{"version separator", "mockorg", "foo", "1.0/0", "any", `invalid version "1.0/0"`},
		{"platform separator", "mockorg", "foo", "1.0.0", "any/../evil", `invalid platform "any/../evil"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			balaEnv := t.TempDir()

			entries := map[string][]byte{
				"Bala.toml": []byte("[build]\nplatform = \"" + tt.platform + "\"\n"),
				"Ballerina.toml": []byte("[package]\norg = \"" + tt.org + "\"\nname = \"" +
					tt.pkg + "\"\nversion = \"" + tt.ver + "\"\n"),
				"main.bal": []byte("public function main() {}\n"),
			}
			balaPath := writePushBalaFixture(t, "mockorg-foo-any-1.0.0.bala", entries)

			_, stderr, err := executePushCLI(t, t.TempDir(), balaEnv, balaPath, "--repository", "local")
			if err == nil {
				t.Fatal("expected rejection, got success")
			}
			if !strings.Contains(stderr, tt.wantErrSubstring) {
				t.Errorf("expected %q in stderr, got: %s", tt.wantErrSubstring, stderr)
			}

			repoRoot := filepath.Join(balaEnv, "repositories", "local", "bala")
			if entries, err := os.ReadDir(repoRoot); err == nil && len(entries) != 0 {
				t.Errorf("expected repo root untouched, found entries: %v", entries)
			}
		})
	}
}

func TestBalPushMissingRepositoryFlag(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaPath := writePushBalaFixture(t, "acme-widgets-any-1.2.3.bala", validPushBalaEntries())

	_, stderr, err := executePushCLI(t, t.TempDir(), "", balaPath)
	if err == nil {
		t.Fatal("expected error when --repository is omitted, got success")
	}
	if !strings.Contains(stderr, `required flag(s) "repository" not set`) {
		t.Errorf("expected required-flag error, got stderr: %s", stderr)
	}
}

func TestBalPushUnsupportedRepositoryValue(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balaPath := writePushBalaFixture(t, "acme-widgets-any-1.2.3.bala", validPushBalaEntries())

	_, stderr, err := executePushCLI(t, t.TempDir(), "", balaPath, "--repository", "central")
	if err == nil {
		t.Fatal("expected error for --repository=central, got success")
	}
	if !strings.Contains(stderr, `unsupported --repository value "central"`) {
		t.Errorf("expected unsupported-value error, got stderr: %s", stderr)
	}
}

func TestBalPushHelp(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	stdout, stderr, err := executePushCLI(t, t.TempDir(), "", "--help")
	if err != nil {
		t.Fatalf("--help returned error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Push a Ballerina archive") {
		t.Errorf("expected push help text, got: %s", stdout)
	}
	if !strings.Contains(stdout, "only 'local' is supported in this") {
		t.Errorf("expected current-scope note in help, got: %s", stdout)
	}
}

// runPushCLI runs `bal push <args...>` as a subprocess in dir, with BAL_ENV
// set to balaEnv (skipped if balaEnv is ""). Each push test gets its own
// balaEnv, unlike build/pack's shared cliIntegrationBalEnv, since push writes
// into <BAL_ENV>/repositories/local/bala/...
func runPushCLI(t *testing.T, dir, balaEnv string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	balBin, _, coverDir := integrationTestBalCLI(t, false)
	env := os.Environ()
	if balaEnv != "" {
		env = append(env, "BAL_ENV="+balaEnv)
	}
	if coverDir != "" {
		commandCoverDir := t.TempDir()
		env = append(env, "GOCOVERDIR="+commandCoverDir)
		defer mergeCLICoverageDir(t, commandCoverDir, coverDir)
	}
	return runNativeCLICommandWithEnv(t, balBin, dir, append([]string{"push"}, args...), env)
}

func executePushCLI(t *testing.T, dir, balaEnv string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	stdout, stderr, exitCode := runPushCLI(t, dir, balaEnv, args...)
	if exitCode != 0 {
		err = fmt.Errorf("push exited with code %d", exitCode)
	}
	return stdout, stderr, err
}

func validPushBalaEntries() map[string][]byte {
	return map[string][]byte{
		"Bala.toml":      []byte("[build]\nplatform = \"any\"\n"),
		"Ballerina.toml": []byte("[package]\norg = \"acme\"\nname = \"widgets\"\nversion = \"1.2.3\"\n"),
		"main.bal":       []byte("public function main() {}\n"),
	}
}

func writePushBalaFixture(t *testing.T, filename string, entries map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	balaPath := filepath.Join(dir, filename)
	writePushBalaFile(t, balaPath, entries)
	return balaPath
}

// pushBalaEntry preserves order, unlike a map, for tests where entry order matters.
type pushBalaEntry struct {
	name    string
	content []byte
}

func writePushOrderedBalaFile(t *testing.T, balaPath string, entries []pushBalaEntry) {
	t.Helper()
	out, err := os.Create(balaPath)
	if err != nil {
		t.Fatalf("create %s: %v", balaPath, err)
	}
	zw := zip.NewWriter(out)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("create zip entry %s: %v", e.name, err)
		}
		if _, err := io.Copy(w, bytes.NewReader(e.content)); err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("write zip entry %s: %v", e.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", balaPath, err)
	}
}

// Entries whose name ends with "/" are written as explicit directory entries.
func writePushBalaFileWithDirEntries(t *testing.T, balaPath string, entries []pushBalaEntry) {
	t.Helper()
	out, err := os.Create(balaPath)
	if err != nil {
		t.Fatalf("create %s: %v", balaPath, err)
	}
	zw := zip.NewWriter(out)
	for _, e := range entries {
		isDir := strings.HasSuffix(e.name, "/")
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if isDir {
			h.Method = zip.Store
			h.SetMode(0o755 | os.ModeDir)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("create zip entry %s: %v", e.name, err)
		}
		if !isDir {
			if _, err := io.Copy(w, bytes.NewReader(e.content)); err != nil {
				_ = zw.Close()
				_ = out.Close()
				t.Fatalf("write zip entry %s: %v", e.name, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", balaPath, err)
	}
}

func writePushBalaFile(t *testing.T, balaPath string, entries map[string][]byte) {
	t.Helper()
	out, err := os.Create(balaPath)
	if err != nil {
		t.Fatalf("create %s: %v", balaPath, err)
	}
	zw := zip.NewWriter(out)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := io.Copy(w, bytes.NewReader(content)); err != nil {
			_ = zw.Close()
			_ = out.Close()
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		t.Fatalf("close zip writer: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", balaPath, err)
	}
}

// TestBalBuildLoadWarning covers a plain, non-workspace package with a
// repository = "local" dependency that's missing locally and falls back to
// central with a warning — the same mechanism TestBalBuildWorkspace checks
// per-member, but here at the top-level result.Diagnostics() check in
// runBuild (before any workspace/member branching), which a workspace
// build never exercises since it always takes the member-loop path.
func TestBalBuildLoadWarning(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "local-repo-miss-warning", "project")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "build")

	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	copyDir(t, filepath.Join(repoRoot, "projects", "testdata", "repo", "bala"), centralCache)

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + tempHome}, "build", projectDir)
	if exitCode != 0 {
		t.Fatalf("bal build failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	const wantWarning = "dependency mockorg/leafpkg:1.0.0 cannot be found in the 'local' repository. falling back to default repositories"
	if !strings.Contains(stderr, wantWarning) {
		t.Errorf("expected the local-repo-miss warning, got stderr:\n%s", stderr)
	}

	idx := strings.Index(stdout, "Created ")
	if idx == -1 {
		t.Fatalf("expected a 'Created' line, got stdout:\n%s", stdout)
	}
	rest := stdout[idx+len("Created "):]
	line, _, _ := strings.Cut(rest, "\n")
	binPath := strings.TrimSpace(line)

	runOut, runErr, runExit := runCLICommand(t, binPath, repoRoot, coverDir)
	runOut = test_util.NormalizeNewlines(runOut)
	runErr = test_util.NormalizeNewlines(runErr)
	expectedOut, expectedErr, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(filepath.Join(outputsRoot, "local-repo-miss-warning.txtar"))
	if err != nil {
		t.Fatalf("failed to parse golden txtar: %v", err)
	}
	if runOut != expectedOut {
		t.Fatalf("unexpected stdout from built binary\n%s", test_util.FormatExpectedGot(expectedOut, runOut))
	}
	if runErr != expectedErr {
		t.Fatalf("unexpected stderr from built binary\n%s", test_util.FormatExpectedGot(expectedErr, runErr))
	}
	if strconv.Itoa(runExit) != expectedExit {
		t.Fatalf("unexpected exit code from built binary\n%s", test_util.FormatExpectedGot(expectedExit, strconv.Itoa(runExit)))
	}
}

// TestBalBuildWorkspacePartialFailure covers a workspace where one member
// fails to compile: bal build must stop there, not attempt later members
// (matching jballerina), while still exiting non-zero.
func TestBalBuildWorkspacePartialFailure(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	workspaceDir := filepath.Join("corpus", "cli", "testdata", "build", "workspace-partial-failure", "project")

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", workspaceDir)
	if exitCode == 0 {
		t.Fatalf("expected a non-zero exit code when one workspace member fails to compile, got 0\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "compilation failed; executable not produced") {
		t.Errorf("expected pkga's compile failure to be reported, got stderr:\n%s", stderr)
	}
	if strings.Contains(stdout, "Created ") {
		t.Fatalf("expected no member to be built once pkga fails (bal build should stop, not continue to pkgb); stdout:\n%s", stdout)
	}
}

// TestBalBuildWorkspaceMemberManifestError covers a workspace where one
// member's manifest fails to load (as opposed to
// TestBalBuildWorkspacePartialFailure's compile error): the workspace-wide
// load itself already surfaces the member's error in result.Diagnostics()
// before runBuild ever reaches the per-member build loop, so the whole
// build must abort with none of the members — not even a healthy sibling —
// ever attempted.
func TestBalBuildWorkspaceMemberManifestError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	workspaceDir := filepath.Join("corpus", "cli", "testdata", "build", "workspace-member-manifest-error", "project")

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", workspaceDir)
	if exitCode == 0 {
		t.Fatalf("expected a non-zero exit code when a workspace member's manifest fails to load, got 0\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "invalid version") {
		t.Errorf("expected pkga's manifest error to be reported, got stderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "package loading reported errors") {
		t.Errorf("expected the load-phase error message, got stderr:\n%s", stderr)
	}
	if strings.Contains(stdout, "Created ") {
		t.Fatalf("expected no member to be built when a sibling's manifest fails to load; stdout:\n%s", stdout)
	}
}

// TestBalBuildCustomOutputPath covers the -o flag, including creating
// parent directories for a path that doesn't exist yet — common in
// CI/deployment workflows targeting a specific output location rather than
// the default target/bin/<package-name>.
func TestBalBuildCustomOutputPath(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	outPath := filepath.Join(t.TempDir(), "nested", "dir", "custom-name")

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir, "-o", outPath)
	if exitCode != 0 {
		t.Fatalf("bal build -o failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "Created "+outPath) {
		t.Fatalf("expected bal build stdout to report the custom output path %q, got:\n%s", outPath, stdout)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected built binary at custom output path %s: %v", outPath, err)
	}
}

// TestBalBuildOutputPathBlocked covers a regular file sitting where -o
// needs to create a parent directory: executable.Pack must fail, and
// buildOneProject must surface that as a clear "write executable" error
// rather than a bare/unwrapped one.
func TestBalBuildOutputPathBlocked(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	blockingFile := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blockingFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("writing blocking file: %v", err)
	}
	outPath := filepath.Join(blockingFile, "bin", "program")

	_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir, "-o", outPath)
	if exitCode == 0 {
		t.Fatal("expected a non-zero exit code when the output path is blocked, got 0")
	}
	if !strings.Contains(stderr, "write executable") {
		t.Errorf("expected a 'write executable' error, got stderr:\n%s", stderr)
	}
}

// TestBalBuildNoHomeNoBalEnv covers getBallerinaEnvPath's fallback when
// BAL_ENV is unset: it resolves the user's home directory itself, which
// fails if neither is available — e.g. a minimal container environment.
// Uses runNativeCLICommandWithEnv (a full, explicit env rather than
// extraEnv appended on top of the inherited one) since this needs to
// remove variables, not just add one; a copied project dir keeps the
// build output out of the checked-in fixture.
func TestBalBuildNoHomeNoBalEnv(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	srcProject := filepath.Join(repoRoot, "corpus", "cli", "testdata", "build", "pure-ballerina", "project")
	projectDir := t.TempDir()
	copyDir(t, srcProject, projectDir)

	env := envWithoutVars(os.Environ(), "BAL_ENV", "HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH")
	if coverDir != "" {
		env = append(env, "GOCOVERDIR="+coverDir)
	}

	_, stderr, exitCode := runNativeCLICommandWithEnv(t, balBin, repoRoot, []string{"build", projectDir}, env)
	if exitCode == 0 {
		t.Fatal("expected a non-zero exit code with neither BAL_ENV nor a resolvable home directory, got 0")
	}
	if !strings.Contains(stderr, "resolve ballerina env path") {
		t.Errorf("expected a 'resolve ballerina env path' error, got stderr:\n%s", stderr)
	}
}

// TestBalBuildStatsFlags covers --stats and --stats-oneline: both must
// print a per-stage compilation timing report to stderr without affecting
// the build's own success or output path.
func TestBalBuildStatsFlags(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	for _, flag := range []string{"--stats", "--stats-oneline"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
				[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir, flag)
			if exitCode != 0 {
				t.Fatalf("bal build %s failed: exit=%d\nstdout:\n%s\nstderr:\n%s", flag, exitCode, stdout, stderr)
			}
			if !strings.Contains(stderr, "Compilation Stats:") {
				t.Errorf("expected a compilation stats report in stderr for %s, got:\n%s", flag, stderr)
			}
			if !strings.Contains(stdout, "Created ") {
				t.Errorf("expected %s to still report a successful build, got stdout:\n%s", flag, stdout)
			}
		})
	}
}

// TestBalBuildRuntimeStubPathOverride covers the RuntimeStubPath link-time
// override (set via -ldflags -X main.RuntimeStubPath=..., not a bal build
// flag): when it points at an existing file, that file is used as the
// runner stub as-is, skipping the default <dist>/rt/<os>-<arch> lookup
// entirely; when it's missing, bal build must fail with a clear error
// naming the override path.
func TestBalBuildRuntimeStubPathOverride(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	_, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	t.Run("valid override is used as the stub", func(t *testing.T) {
		t.Parallel()
		overridePath := filepath.Join(t.TempDir(), "custom-stub")
		const stubContent = "custom-stub-bytes"
		if err := os.WriteFile(overridePath, []byte(stubContent), 0o755); err != nil {
			t.Fatalf("writing override stub: %v", err)
		}
		balBin := buildBalBinaryWithRuntimeStubOverride(t, repoRoot, coverDir, overridePath)

		stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
			[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir)
		if exitCode != 0 {
			t.Fatalf("bal build with a valid override failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
		}
		idx := strings.Index(stdout, "Created ")
		if idx == -1 {
			t.Fatalf("expected a 'Created' line, got stdout:\n%s", stdout)
		}
		rest := stdout[idx+len("Created "):]
		line, _, _ := strings.Cut(rest, "\n")
		binPath := strings.TrimSpace(line)

		data, err := os.ReadFile(binPath)
		if err != nil {
			t.Fatalf("reading produced binary: %v", err)
		}
		if !bytes.HasPrefix(data, []byte(stubContent)) {
			t.Error("expected the produced binary to start with the override stub's own bytes; ResolveStub must have used RuntimeStubPath rather than the default rt/ lookup")
		}
	})

	t.Run("missing override reports a clear error", func(t *testing.T) {
		t.Parallel()
		overridePath := filepath.Join(t.TempDir(), "does-not-exist")
		balBin := buildBalBinaryWithRuntimeStubOverride(t, repoRoot, coverDir, overridePath)

		_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
			[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir)
		if exitCode == 0 {
			t.Fatal("expected a non-zero exit code for a missing override stub")
		}
		wantMsg := "ballerina runtime binary not found at " + overridePath
		if !strings.Contains(stderr, wantMsg) {
			t.Errorf("expected stderr to contain %q, got:\n%s", wantMsg, stderr)
		}
	})

	t.Run("rejected when cross-compiling", func(t *testing.T) {
		t.Parallel()
		overridePath := filepath.Join(t.TempDir(), "custom-stub")
		if err := os.WriteFile(overridePath, []byte("custom-stub-bytes"), 0o755); err != nil {
			t.Fatalf("writing override stub: %v", err)
		}
		balBin := buildBalBinaryWithRuntimeStubOverride(t, repoRoot, coverDir, overridePath)

		nonHostTarget := "linux"
		if runtime.GOOS == "linux" {
			nonHostTarget = "darwin"
		}
		_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
			[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir,
			"--target-os", nonHostTarget, "--target-arch", "amd64")
		if exitCode == 0 {
			t.Fatal("expected a non-zero exit code when overriding the stub while cross-compiling")
		}
		if !strings.Contains(stderr, "runtime stub override cannot be used when cross-compiling") {
			t.Errorf("expected a clear cross-compile rejection, got stderr:\n%s", stderr)
		}
	})
}

// packedTrailerSize mirrors cli/internal/executable.Pack's own trailer
// layout (8-byte little-endian payload offset + 8-byte magic marker),
// duplicated here rather than imported since that package is cli/internal
// and unreachable from corpus — this is a black-box test of the packed
// executable format bal build/pack produce, not a peek at their internals.
const packedTrailerSize = 16

// corruptPackedTrailerOffset overwrites the offset half of a packed
// executable's trailer with an out-of-range value. The magic bytes are left
// intact, so TryLoad still recognizes the file as claiming to be a compiled
// program but fails to load it — exercising the err != nil branch in bal's
// and balrt's own main(), distinct from the not-a-compiled-program branch a
// plain (non-packed) file would hit.
func corruptPackedTrailerOffset(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(data) < packedTrailerSize {
		t.Fatalf("%s is too small to have a packed trailer", path)
	}
	trailer := data[len(data)-packedTrailerSize:]
	for i := range 8 {
		trailer[i] = 0xFF // offset = max uint64, guaranteed out of range
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatalf("writing corrupted %s: %v", path, err)
	}
}

// buildBalAsStubOutput builds a "packer" bal binary using the real bal
// binary itself as its own runner stub (via the RuntimeStubPath link-time
// override), then bal builds projectDir with it, returning the path to the
// resulting executable — bal's own machine code with a BIR payload appended.
// Running that output directly exercises bal.go's own main(), not balrt's.
func buildBalAsStubOutput(t *testing.T, balBin, repoRoot, coverDir, projectDir string) string {
	t.Helper()
	packerBal := buildBalBinaryWithRuntimeStubOverride(t, repoRoot, coverDir, balBin)
	stdout, stderr, exitCode := runCLICommandWithEnv(t, packerBal, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir)
	if exitCode != 0 {
		t.Fatalf("bal build with bal itself as the stub failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	idx := strings.Index(stdout, "Created ")
	if idx == -1 {
		t.Fatalf("expected a 'Created' line, got stdout:\n%s", stdout)
	}
	rest := stdout[idx+len("Created "):]
	line, _, _ := strings.Cut(rest, "\n")
	return strings.TrimSpace(line)
}

// TestBalrtRejectsPlainExecution covers balrt's main() run against a plain,
// non-packed binary. balrt has no purpose other than running a bal
// build/pack output, so this is its expected failure mode, not merely a
// theoretical error path — a plain balrt build (no BIR ever embedded) must
// already reproduce it.
func TestBalrtRejectsPlainExecution(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	_, repoRoot, coverDir := integrationTestBalCLI(t, false)

	balrtName := "balrt"
	if runtime.GOOS == "windows" {
		balrtName += ".exe"
	}
	balrtBin := filepath.Join(t.TempDir(), balrtName)
	if err := buildBalrtBinaryTo(repoRoot, coverDir, balrtBin); err != nil {
		t.Fatalf("building balrt binary: %v", err)
	}

	_, stderr, exitCode := runCLICommand(t, balrtBin, repoRoot, coverDir)
	if exitCode == 0 {
		t.Fatal("expected balrt to fail when run without an embedded program")
	}
	wantMsg := "balrt only runs compiled Ballerina executables produced by bal build"
	if !strings.Contains(stderr, wantMsg) {
		t.Errorf("expected stderr to contain %q, got:\n%s", wantMsg, stderr)
	}
}

// TestBalBuildCorruptedOutputReportsError covers running a bal build output
// (the common case: balrt as the runner stub) whose trailer has been
// corrupted after the fact — the magic marker still says "compiled
// program", but the payload is bad, so balrt's own main() must report a
// clear error rather than crash or silently do nothing.
func TestBalBuildCorruptedOutputReportsError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")
	outBin := filepath.Join(t.TempDir(), hostExeSuffix("myprogram"))

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir, "-o", outBin)
	if exitCode != 0 {
		t.Fatalf("bal build failed: exit=%d\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}

	corruptPackedTrailerOffset(t, outBin)

	_, runErr, runExit := runCLICommand(t, outBin, repoRoot, coverDir)
	if runExit == 0 {
		t.Fatal("expected a corrupted build output to fail rather than run")
	}
	if !strings.Contains(runErr, "ballerina:") || !strings.Contains(runErr, "invalid embedded payload offset") {
		t.Errorf("expected a clear 'ballerina: invalid embedded payload offset ...' error, got:\n%s", runErr)
	}
}

// TestBalBinaryAsRuntimeStub covers bal's own main() detecting and running
// an embedded program. RuntimeStubPath lets bal build use the full bal
// binary as its own runner stub; running the produced executable directly
// must execute the embedded program instead of behaving as the bal CLI.
func TestBalBinaryAsRuntimeStub(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")
	outputsRoot := filepath.Join(repoRoot, "corpus", "cli", "output", "build")

	producedBin := buildBalAsStubOutput(t, balBin, repoRoot, coverDir, projectDir)

	runOut, runErr, runExit := runCLICommand(t, producedBin, repoRoot, coverDir)
	runOut = test_util.NormalizeNewlines(runOut)
	runErr = test_util.NormalizeNewlines(runErr)
	expectedOut, expectedErr, expectedExit, err := test_util.LoadTxtarStdoutStderrExitcode(filepath.Join(outputsRoot, "pure-ballerina.txtar"))
	if err != nil {
		t.Fatalf("failed to parse golden txtar: %v", err)
	}
	if runOut != expectedOut {
		t.Fatalf("unexpected stdout running bal-as-stub output %s\n%s", producedBin, test_util.FormatExpectedGot(expectedOut, runOut))
	}
	if runErr != expectedErr {
		t.Fatalf("unexpected stderr running bal-as-stub output %s\n%s", producedBin, test_util.FormatExpectedGot(expectedErr, runErr))
	}
	if strconv.Itoa(runExit) != expectedExit {
		t.Fatalf("unexpected exit code running bal-as-stub output %s\n%s", producedBin, test_util.FormatExpectedGot(expectedExit, strconv.Itoa(runExit)))
	}
}

// TestBalBinaryAsRuntimeStub_CorruptedOutputReportsError is
// TestBalBuildCorruptedOutputReportsError's counterpart for bal.go's own
// main(): the produced executable here is bal's own machine code (not
// balrt's), so corrupting its trailer exercises bal.go's err != nil branch
// specifically.
func TestBalBinaryAsRuntimeStub_CorruptedOutputReportsError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	producedBin := buildBalAsStubOutput(t, balBin, repoRoot, coverDir, projectDir)
	corruptPackedTrailerOffset(t, producedBin)

	_, runErr, runExit := runCLICommand(t, producedBin, repoRoot, coverDir)
	if runExit == 0 {
		t.Fatal("expected a corrupted bal-as-stub output to fail rather than run")
	}
	if !strings.Contains(runErr, "ballerina:") || !strings.Contains(runErr, "invalid embedded payload offset") {
		t.Errorf("expected a clear 'ballerina: invalid embedded payload offset ...' error, got:\n%s", runErr)
	}
}

// TestBalBuildFlatRuntimeStubDefault covers ResolveStub's flat-sibling
// fallback: a bal binary built alongside a plain "balrt" (no rt/<os>-<arch>
// subtree, no RuntimeStubPath override) must still find and use it as the
// runner stub — the layout a local dev build produces with no extra flags.
func TestBalBuildFlatRuntimeStubDefault(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	_, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	dir := t.TempDir()
	balBin := filepath.Join(dir, cliIntegrationBalExecutableName(false))
	if err := buildBalBinaryTo(repoRoot, coverDir, balBin, false); err != nil {
		t.Fatalf("building bal binary: %v", err)
	}
	balrtName := "balrt"
	if runtime.GOOS == "windows" {
		balrtName += ".exe"
	}
	if err := buildBalrtBinaryTo(repoRoot, coverDir, filepath.Join(dir, balrtName)); err != nil {
		t.Fatalf("building balrt binary: %v", err)
	}

	_, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir)
	if exitCode != 0 {
		t.Fatalf("bal build with a flat sibling balrt (no rt/ layout, no override) failed: exit=%d\nstderr:\n%s", exitCode, stderr)
	}
}

// buildCrossBalrtStub cross-builds a balrt stub for goos/goarch (not
// necessarily the host's own) into rtDir/<goos>-<goarch>/balrt[.exe] — used
// to provision a second platform's stub in the shared test rt/ directory,
// alongside the host's own (already built by ensureCLIIntegrationBalBinaries),
// so a cross-compile test can verify bal build picks the correct one. No
// -cover: this binary is never executed (a different-arch binary can't run
// on the host) — only its packed output's header is inspected.
func buildCrossBalrtStub(t *testing.T, repoRoot, rtDir, goos, goarch string) string {
	t.Helper()
	name := "balrt"
	if goos == "windows" {
		name += ".exe"
	}
	outputPath := filepath.Join(rtDir, goos+"-"+goarch, name)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("creating rt dir for %s/%s: %v", goos, goarch, err)
	}

	base := os.Environ()
	env := make([]string, 0, len(base)+3)
	for _, e := range base {
		if strings.HasPrefix(e, "GOOS=") || strings.HasPrefix(e, "GOARCH=") || strings.HasPrefix(e, "CGO_ENABLED=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")

	cmd := exec.Command("go", "build", "-o", outputPath, "./cli/internal/balrt")
	cmd.Dir = repoRoot
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cross-building balrt for %s/%s: %v\n%s", goos, goarch, err, out)
	}
	return outputPath
}

// TestBalBuildCrossCompile covers `bal build --target-os/--target-arch` for
// a package with no native Go dependencies — the executable.ResolveStub
// path. The shared test bal binary only has its own host platform's balrt
// provisioned by default, so this provisions a second platform's stub
// itself and verifies bal build picks the correct one — not the host's
// own — by inspecting the produced binary's actual format (Pack copies the
// stub's bytes verbatim, so the packed output's header is exactly the
// stub's). Targets windows/amd64 specifically (unless the host itself is
// windows/amd64) so the target-platform-follows-.exe-suffix behavior is
// exercised deterministically, not left to chance.
func TestBalBuildCrossCompile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	targetOS, targetArch := "windows", "amd64"
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		targetOS, targetArch = "linux", "amd64"
	}
	buildCrossBalrtStub(t, repoRoot, filepath.Join(filepath.Dir(balBin), "rt"), targetOS, targetArch)

	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")
	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir,
		"--target-os", targetOS, "--target-arch", targetArch)
	if exitCode != 0 {
		t.Fatalf("cross-compiled bal build failed (exit %d)\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	const createdPrefix = "Created "
	idx := strings.Index(stdout, createdPrefix)
	if idx == -1 {
		t.Fatalf("expected bal build stdout to report a %q line, got:\n%s", createdPrefix, stdout)
	}
	binPath := strings.TrimSpace(strings.SplitN(stdout[idx+len(createdPrefix):], "\n", 2)[0])
	if targetOS == "windows" && !strings.HasSuffix(binPath, ".exe") {
		t.Fatalf("expected a .exe suffix for a windows cross-compile target, got %q", binPath)
	}

	gotFormat, err := binaryFormat(binPath)
	if err != nil {
		t.Fatalf("reading binary format of %s: %v", binPath, err)
	}
	if wantFormat := expectedBinaryFormat(targetOS); gotFormat != wantFormat {
		t.Fatalf("cross-compiled binary %s has format %q, want %q for target OS %q — the wrong platform's stub was likely used",
			binPath, gotFormat, wantFormat, targetOS)
	}
}

// TestBalBuildUnsupportedTargetPlatform covers a --target-os/--target-arch
// combination bal build doesn't support for a package with no native Go
// dependencies (executable.ResolveStub's platform-support check) — must
// fail clearly, naming the supported list, rather than silently looking for
// a stub that will never exist.
func TestBalBuildUnsupportedTargetPlatform(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	projectDir := filepath.Join("corpus", "cli", "testdata", "build", "pure-ballerina", "project")

	stdout, stderr, exitCode := runCLICommandWithEnv(t, balBin, repoRoot, coverDir,
		[]string{"BAL_ENV=" + cliIntegrationBalEnv}, "build", projectDir,
		"--target-os", "plan9", "--target-arch", "amd64")
	if exitCode == 0 {
		t.Fatalf("expected a non-zero exit code for an unsupported target platform, got 0\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "unsupported target platform plan9/amd64") {
		t.Errorf("expected stderr to name the unsupported platform, got:\n%s", stderr)
	}
}

// TestBalBuildNativeDependency exercises `bal build` end-to-end on a package
// with a native Go dependency: it must build a native-woven standalone stub
// (targeting cli/internal/balrt, not the full CLI) instead of looking up the
// predefined installed stub, and the produced binary must be genuinely
// standalone — runnable with no bal/Go toolchain involved. Mirrors
// TestNativeRunner_ColdBuildAndCacheHit's cold/cache-hit pattern for bal run
// (native_runner_test.go) and this file's own stub-size/run-and-compare
// pattern for bal build (TestBalBuildCorpus); reuses the same
// native-multi-org-v fixture + golden output as TestNativeMultiOrgPackages
// (native_runner_test.go).
func TestBalBuildNativeDependency(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}

	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	// Build a temp Ballerina home whose central cache contains the testdata
	// native packages so bal build can resolve them.
	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	// Copy the project to a temp dir so the output binary and native-build
	// cache go to a fresh location each run.
	srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
	projectDir := t.TempDir()
	copyDir(t, srcProject, projectDir)

	proxyURL := localDistributionProxy(t, repoRoot)
	moduleCache := isolatedGoModuleCache(t, tempHome)
	extraEnv := []string{
		"BAL_ENV=" + tempHome,
		"BALLERINA_SRC=",
		"GOPROXY=" + proxyURL + ",https://proxy.golang.org,direct",
		"GONOSUMDB=github.com/ballerina-nutcracker/ballerina*",
		"GOMODCACHE=" + moduleCache,
	}
	runBuild := func() (stdout, stderr string, code int) {
		return runCLICommandWithEnv(t, balBin, repoRoot, coverDir, extraEnv, "build", projectDir)
	}

	// First build: cold — must build a native-woven stub.
	stdout1, stderr1, code1 := runBuild()
	if code1 != 0 {
		t.Fatalf("first bal build failed (exit %d)\nstdout: %s\nstderr: %s", code1, stdout1, stderr1)
	}
	if !strings.Contains(stderr1, "info: building native interpreter") {
		t.Errorf("first build: expected 'info: building native interpreter' in stderr\nstderr: %s", stderr1)
	}

	binPath := filepath.Join(projectDir, "target", "bin", hostExeSuffix("nativemultiorg"))
	wantMessage := "Created " + binPath
	if !strings.Contains(stdout1, wantMessage) {
		t.Fatalf("expected bal build stdout to report %q, got:\n%s", wantMessage, stdout1)
	}

	builtInfo, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("expected built binary at %s: %v", binPath, err)
	}
	balInfo, err := os.Stat(balBin)
	if err != nil {
		t.Fatalf("failed to stat bal binary %s: %v", balBin, err)
	}
	if builtInfo.Size() >= balInfo.Size()/2 {
		t.Fatalf("built binary %s (%d bytes) is not meaningfully smaller than bal (%d bytes); expected the slim cli/internal/balrt-targeted stub, not the full CLI",
			binPath, builtInfo.Size(), balInfo.Size())
	}

	// Second build: must hit the fingerprint cache and skip rebuilding.
	stdout2, stderr2, code2 := runBuild()
	if code2 != 0 {
		t.Fatalf("second bal build failed (exit %d)\nstdout: %s\nstderr: %s", code2, stdout2, stderr2)
	}
	if strings.Contains(stderr2, "info: building native interpreter") {
		t.Errorf("second build: unexpected native interpreter rebuild (cache miss)\nstderr: %s", stderr2)
	}

	// The produced binary must be genuinely standalone — run it directly,
	// with no bal/Go toolchain involved, and compare against the same golden
	// output TestNativeMultiOrgPackages already verifies for this fixture.
	runOut, runErr, runExit := runCLICommand(t, binPath, repoRoot, coverDir)
	runOut = test_util.NormalizeNewlines(runOut)
	runErr = test_util.NormalizeNewlines(runErr)

	expectedOut, expectedErr, err := test_util.LoadTxtarStdoutStderr(filepath.Join(repoRoot, "corpus", "extern", "output", "native-multi-org-v.txtar"))
	if err != nil {
		t.Fatalf("failed to parse golden txtar: %v", err)
	}
	expectedOut = test_util.NormalizeNewlines(expectedOut)
	expectedErr = test_util.NormalizeNewlines(expectedErr)
	if runOut != expectedOut {
		t.Fatalf("unexpected stdout from built binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedOut, runOut))
	}
	if runErr != expectedErr {
		t.Fatalf("unexpected stderr from built binary %s\n%s", binPath, test_util.FormatExpectedGot(expectedErr, runErr))
	}
	if runExit != 0 {
		t.Fatalf("expected exit code 0 from built binary, got %d", runExit)
	}
}

// TestBalBuildNativeDependencyCrossCompile exercises `bal build
// --target-os/--target-arch` on a package with a native Go dependency:
// cross-compilation must actually cross-compile (LocalExecutor sets
// GOOS/GOARCH on the go build subprocess) rather than silently producing a
// host binary mislabeled as the requested target. Each target is chosen to
// never equal the host's own platform, so a regression back to the old
// host-only behavior would be caught by a format mismatch, not just a
// missing file. The windows/amd64 case also covers buildNativeStub's own
// ".exe" suffix on the intermediate native-woven stub it builds — distinct
// from buildOneProject's suffix on the final packed output, which the
// non-windows case alone wouldn't reach.
func TestBalBuildNativeDependencyCrossCompile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}

	nonHostTarget := "linux"
	if runtime.GOOS == "linux" {
		nonHostTarget = "darwin"
	}

	targets := []struct{ os, arch string }{
		{nonHostTarget, "amd64"},
		{"windows", "amd64"},
	}

	for _, target := range targets {
		targetOS, targetArch := target.os, target.arch
		t.Run(targetOS+"-"+targetArch, func(t *testing.T) {
			t.Parallel()
			balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

			tempHome := t.TempDir()
			centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
			srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
			copyDir(t, srcRepo, centralCache)

			srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
			projectDir := t.TempDir()
			copyDir(t, srcProject, projectDir)

			extraEnv := []string{"BAL_ENV=" + tempHome, "BALLERINA_SRC=" + repoRoot}
			stdout, stderr, code := runCLICommandWithEnv(t, balBin, repoRoot, coverDir, extraEnv,
				"build", projectDir, "--target-os", targetOS, "--target-arch", targetArch)
			if code != 0 {
				t.Fatalf("cross-compiled bal build failed (exit %d)\nstdout: %s\nstderr: %s", code, stdout, stderr)
			}
			if !strings.Contains(stderr, "info: building native interpreter") {
				t.Errorf("expected 'info: building native interpreter' in stderr\nstderr: %s", stderr)
			}

			binName := "nativemultiorg"
			if targetOS == "windows" {
				binName += ".exe"
			}
			binPath := filepath.Join(projectDir, "target", "bin", binName)
			if _, err := os.Stat(binPath); err != nil {
				t.Fatalf("expected cross-compiled binary at %s: %v", binPath, err)
			}

			gotFormat, err := binaryFormat(binPath)
			if err != nil {
				t.Fatalf("reading binary format of %s: %v", binPath, err)
			}
			wantFormat := expectedBinaryFormat(targetOS)
			if gotFormat != wantFormat {
				t.Fatalf("cross-compiled binary %s has format %q, want %q for target OS %q — cross-compilation likely silently produced a host binary instead",
					binPath, gotFormat, wantFormat, targetOS)
			}
		})
	}
}

// binaryFormat identifies an executable's format by its magic bytes,
// without depending on an external "file" command being present in CI.
func binaryFormat(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return "", err
	}
	switch {
	case magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F':
		return "elf", nil
	case magic[0] == 0xcf && magic[1] == 0xfa && magic[2] == 0xed && magic[3] == 0xfe:
		return "macho", nil
	case magic[0] == 'M' && magic[1] == 'Z':
		return "pe", nil
	default:
		return "unknown", nil
	}
}

// expectedBinaryFormat maps a GOOS value to the executable format Go
// produces for it.
func expectedBinaryFormat(goos string) string {
	switch goos {
	case "linux":
		return "elf"
	case "darwin":
		return "macho"
	case "windows":
		return "pe"
	default:
		return "unknown"
	}
}

// TestBalBuildNativeDependencyInvalidTarget covers a bogus --target-os/
// --target-arch combined with a native Go dependency. The native path
// validates against the same curated supportedPlatforms list as the
// no-native-dep path (executable.ResolveStub) via executable.ValidatePlatform,
// so this is rejected before ever touching the Go toolchain, not left to
// go build's own (less clear) validation.
func TestBalBuildNativeDependencyInvalidTarget(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}

	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
	projectDir := t.TempDir()
	copyDir(t, srcProject, projectDir)

	extraEnv := []string{"BAL_ENV=" + tempHome, "BALLERINA_SRC=" + repoRoot}
	stdout, stderr, code := runCLICommandWithEnv(t, balBin, repoRoot, coverDir, extraEnv,
		"build", projectDir, "--target-os", "bogus", "--target-arch", "amd64")

	if code == 0 {
		t.Fatalf("expected a non-zero exit code for an invalid target platform, got 0\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "unsupported target platform bogus/amd64") {
		t.Errorf("expected bal's own curated-platform-list error, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "building native interpreter stub") {
		t.Errorf("expected stderr to show bal's own wrapping context, got:\n%s", stderr)
	}

	binPath := filepath.Join(projectDir, "target", "bin", hostExeSuffix("nativemultiorg"))
	if _, err := os.Stat(binPath); err == nil {
		t.Errorf("expected no output binary to be produced for a failed build, found one at %s", binPath)
	}
}

// TestBalBuildNativeDependencyUnsupportedButBuildable covers a target that
// Go itself can genuinely cross-compile to (linux/386 is a real, universally
// supported GOOS/GOARCH pair) but that isn't in bal's curated
// supportedPlatforms list. The native-dependency path must still reject it —
// compiling the stub from source rather than looking up a prebuilt one isn't
// license to support extra platforms bal doesn't otherwise ship for.
func TestBalBuildNativeDependencyUnsupportedButBuildable(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}

	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
	projectDir := t.TempDir()
	copyDir(t, srcProject, projectDir)

	extraEnv := []string{"BAL_ENV=" + tempHome, "BALLERINA_SRC=" + repoRoot}
	stdout, stderr, code := runCLICommandWithEnv(t, balBin, repoRoot, coverDir, extraEnv,
		"build", projectDir, "--target-os", "linux", "--target-arch", "386")

	if code == 0 {
		t.Fatalf("expected a non-zero exit code for linux/386 (buildable but not curated), got 0\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "unsupported target platform linux/386") {
		t.Errorf("expected bal's own curated-platform-list error, got:\n%s", stderr)
	}

	binPath := filepath.Join(projectDir, "target", "bin", "native", "linux-386", "balrt-native")
	if _, err := os.Stat(binPath); err == nil {
		t.Errorf("expected no native stub to be produced for an unsupported target, found one at %s", binPath)
	}
}

// twoNonHostNativeTargets returns two GOOS/GOARCH pairs, both different from
// the host platform and from each other, so cache/cross-compile tests
// aren't sensitive to which platform they happen to run on.
func twoNonHostNativeTargets(t *testing.T) (osA, archA, osB, archB string) {
	t.Helper()
	candidates := [][2]string{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}
	var picked [][2]string
	for _, c := range candidates {
		if c[0] == runtime.GOOS && c[1] == runtime.GOARCH {
			continue
		}
		picked = append(picked, c)
		if len(picked) == 2 {
			break
		}
	}
	if len(picked) < 2 {
		t.Fatalf("could not find two non-host target platforms (host: %s/%s)", runtime.GOOS, runtime.GOARCH)
	}
	return picked[0][0], picked[0][1], picked[1][0], picked[1][1]
}

func mustReadFileBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return data
}

// TestBalBuildNativeDependencyCacheByTarget covers cache correctness across
// alternating cross-compile targets: building for target A, then target B,
// then target A again must cache-hit the third time (not needlessly
// rebuild) and must reuse target A's own cached stub — not target B's, nor
// a stale/incorrect one. Directly exercises buildNativeStub's
// platform-segmented cache path (target/bin/native/<os>-<arch>/).
func TestBalBuildNativeDependencyCacheByTarget(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}

	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
	projectDir := t.TempDir()
	copyDir(t, srcProject, projectDir)

	targetAOS, targetAArch, targetBOS, targetBArch := twoNonHostNativeTargets(t)

	extraEnv := []string{"BAL_ENV=" + tempHome, "BALLERINA_SRC=" + repoRoot}
	build := func(targetOS, targetArch string) (stdout, stderr string, code int) {
		return runCLICommandWithEnv(t, balBin, repoRoot, coverDir, extraEnv,
			"build", projectDir, "--target-os", targetOS, "--target-arch", targetArch)
	}
	stubPathFor := func(targetOS, targetArch string) string {
		name := "balrt-native"
		if targetOS == "windows" {
			name += ".exe"
		}
		return filepath.Join(projectDir, "target", "bin", "native", targetOS+"-"+targetArch, name)
	}

	// Build A (cold).
	_, stderrA1, codeA1 := build(targetAOS, targetAArch)
	if codeA1 != 0 {
		t.Fatalf("build A (cold) failed: %s", stderrA1)
	}
	if !strings.Contains(stderrA1, "info: building native interpreter") {
		t.Fatalf("expected a cold build for target A, got:\n%s", stderrA1)
	}
	stubABytes1 := mustReadFileBytes(t, stubPathFor(targetAOS, targetAArch))

	// Build B (cold — a different target must not reuse A's cache).
	_, stderrB, codeB := build(targetBOS, targetBArch)
	if codeB != 0 {
		t.Fatalf("build B (cold) failed: %s", stderrB)
	}
	if !strings.Contains(stderrB, "info: building native interpreter") {
		t.Fatalf("expected a cold build for target B (different from A), got:\n%s", stderrB)
	}

	// Build A again: must cache-hit (no rebuild message) and reuse the exact
	// same stub bytes as the first A build, not B's.
	_, stderrA2, codeA2 := build(targetAOS, targetAArch)
	if codeA2 != 0 {
		t.Fatalf("build A (second) failed: %s", stderrA2)
	}
	if strings.Contains(stderrA2, "info: building native interpreter") {
		t.Errorf("expected a cache hit rebuilding target A after building B in between, got a rebuild:\n%s", stderrA2)
	}
	stubABytes2 := mustReadFileBytes(t, stubPathFor(targetAOS, targetAArch))
	if !bytes.Equal(stubABytes1, stubABytes2) {
		t.Error("expected target A's cached stub to be byte-identical across builds; got different bytes")
	}

	// Both targets' stubs must still coexist on disk — a shared cache path
	// would have one silently overwrite the other.
	if _, err := os.Stat(stubPathFor(targetBOS, targetBArch)); err != nil {
		t.Errorf("expected target B's stub to still exist at %s: %v", stubPathFor(targetBOS, targetBArch), err)
	}
}

// TestBalBuildNativeDependencyPartialTargetOverride covers --target-arch
// given alone (no --target-os) combined with a native dependency: the
// unset OS dimension must default to the host's own, matching Go's own
// GOOS/GOARCH convention. That defaulting is already unit-tested generically
// (TestResolveTargetPlatform), but this exercises it actually threading
// through build.go's native path into buildNativeStub's cache path and
// LocalExecutor's cross-compile env — not just the flag-parsing layer.
func TestBalBuildNativeDependencyPartialTargetOverride(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}
	if runtime.GOOS == "windows" {
		// windows only has one supported arch (amd64) in the curated
		// supportedPlatforms list, so there's no valid arch-only override
		// target to test with when the host OS defaults to windows.
		t.Skip("no alternate supported architecture for windows in the curated platform list")
	}

	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
	projectDir := t.TempDir()
	copyDir(t, srcProject, projectDir)

	overrideArch := "amd64"
	if runtime.GOARCH == "amd64" {
		overrideArch = "arm64"
	}

	extraEnv := []string{"BAL_ENV=" + tempHome, "BALLERINA_SRC=" + repoRoot}
	stdout, stderr, code := runCLICommandWithEnv(t, balBin, repoRoot, coverDir, extraEnv,
		"build", projectDir, "--target-arch", overrideArch)
	if code != 0 {
		t.Fatalf("bal build --target-arch %s failed (exit %d)\nstdout: %s\nstderr: %s", overrideArch, code, stdout, stderr)
	}
	if !strings.Contains(stderr, "info: building native interpreter") {
		t.Fatalf("expected a cold build, got:\n%s", stderr)
	}

	binName := "nativemultiorg"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(projectDir, "target", "bin", binName)
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected built binary at %s: %v", binPath, err)
	}

	// The stub cache path must reflect host OS + overridden arch, not the
	// host's own arch — confirms the host-defaulted targetPlatform was
	// actually threaded through to buildNativeStub's cache path, not
	// silently ignored.
	stubName := "balrt-native"
	if runtime.GOOS == "windows" {
		stubName += ".exe"
	}
	expectedStubPath := filepath.Join(projectDir, "target", "bin", "native", runtime.GOOS+"-"+overrideArch, stubName)
	if _, err := os.Stat(expectedStubPath); err != nil {
		t.Fatalf("expected native stub cached at %s (host OS %q + overridden arch %q): %v", expectedStubPath, runtime.GOOS, overrideArch, err)
	}

	gotArch, err := binaryArch(binPath)
	if err != nil {
		t.Fatalf("reading architecture of %s: %v", binPath, err)
	}
	if gotArch != overrideArch {
		t.Fatalf("built binary %s has architecture %q, want %q — --target-arch alone was not correctly threaded through the native build path",
			binPath, gotArch, overrideArch)
	}
}

// binaryArch identifies an executable's target architecture by parsing its
// format-specific header (ELF/Mach-O/PE), returning a GOARCH-style string
// ("amd64", "arm64"). More precise than binaryFormat: it verifies the
// architecture actually threaded through, not just the OS-level format
// (which alone can't distinguish amd64 from arm64 on the same OS).
func binaryArch(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	if ef, err := elf.NewFile(f); err == nil {
		switch ef.Machine {
		case elf.EM_X86_64:
			return "amd64", nil
		case elf.EM_AARCH64:
			return "arm64", nil
		default:
			return "", fmt.Errorf("unrecognized ELF machine %v", ef.Machine)
		}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if mf, err := macho.NewFile(f); err == nil {
		switch mf.Cpu {
		case macho.CpuAmd64:
			return "amd64", nil
		case macho.CpuArm64:
			return "arm64", nil
		default:
			return "", fmt.Errorf("unrecognized Mach-O cpu %v", mf.Cpu)
		}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if pf, err := pe.NewFile(f); err == nil {
		switch pf.Machine {
		case pe.IMAGE_FILE_MACHINE_AMD64:
			return "amd64", nil
		case pe.IMAGE_FILE_MACHINE_ARM64:
			return "arm64", nil
		default:
			return "", fmt.Errorf("unrecognized PE machine %v", pf.Machine)
		}
	}
	return "", fmt.Errorf("unrecognized binary format at %s", path)
}

func assertBalCommandMatchesTxtarFragments(t *testing.T, args []string, txtarPathParts ...string) {
	t.Helper()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	assertBalCommandMatchesTxtarFragmentsForBinary(t, balBin, repoRoot, coverDir, args, txtarPathParts...)
}

func integrationTestBalCLI(t *testing.T, debugBuild bool) (balBin, repoRoot, coverDir string) {
	t.Helper()
	ensureCLIIntegrationBalBinaries(t)
	if debugBuild {
		return cliIntegrationDebugBalBin, cliIntegrationRepoRoot, cliIntegrationCoverDir
	}
	return cliIntegrationRelBalBin, cliIntegrationRepoRoot, cliIntegrationCoverDir
}

func ensureCLIIntegrationBalBinaries(t *testing.T) {
	t.Helper()
	cliIntegrationBinsOnce.Do(func() {
		cliIntegrationRepoRoot, cliIntegrationBinsErr = filepath.Abs("..")
		if cliIntegrationBinsErr != nil {
			return
		}
		cliIntegrationCoverDir, cliIntegrationBinsErr = resolveCLICoverageDir()
		if cliIntegrationBinsErr != nil {
			return
		}
		tmpDir, err := os.MkdirTemp("", "bal-cli-test")
		if err != nil {
			cliIntegrationBinsErr = err
			return
		}

		// bal build looks up its runner stub at
		// <dist>/rt/<GOOS>-<GOARCH>/balrt (executable.ResolveStub,
		// executable.DistributionDir), relative to wherever the running bal
		// binary itself lives — tmpDir plays that role here, so rt/ is a
		// sibling of the bal/bal-debug binaries built into it below.
		//
		// cliIntegrationBalEnv is a separate, unrelated isolated BAL_ENV used
		// only for BallerinaEnvFs (stdlib bala cache) resolution, keeping
		// tests from touching the real ~/.ballerina. Confirmed empty (no
		// pre-populated stdlib bala cache) works fine for these fixtures'
		// ballerina/time and ballerina/io imports, since core stdlib modules
		// resolve from lib/stdlibs source, not a bala cache lookup.
		cliIntegrationBalEnv = filepath.Join(tmpDir, "bal-env")
		if cliIntegrationBinsErr = os.MkdirAll(cliIntegrationBalEnv, 0o755); cliIntegrationBinsErr != nil {
			return
		}
		balrtName := "balrt"
		if runtime.GOOS == "windows" {
			balrtName += ".exe"
		}
		platformDir := runtime.GOOS + "-" + runtime.GOARCH
		balrtPath := filepath.Join(tmpDir, "rt", platformDir, balrtName)
		if cliIntegrationBinsErr = os.MkdirAll(filepath.Dir(balrtPath), 0o755); cliIntegrationBinsErr != nil {
			return
		}
		if cliIntegrationBinsErr = buildBalrtBinaryTo(cliIntegrationRepoRoot, cliIntegrationCoverDir, balrtPath); cliIntegrationBinsErr != nil {
			return
		}
		for _, spec := range []struct {
			debug   bool
			destPtr *string
		}{
			{false, &cliIntegrationRelBalBin},
			{true, &cliIntegrationDebugBalBin},
		} {
			name := cliIntegrationBalExecutableName(spec.debug)
			*spec.destPtr = filepath.Join(tmpDir, name)
			if cliIntegrationBinsErr = buildBalBinaryTo(cliIntegrationRepoRoot, cliIntegrationCoverDir, *spec.destPtr, spec.debug); cliIntegrationBinsErr != nil {
				return
			}
		}
	})
	if cliIntegrationBinsErr != nil {
		t.Fatalf("cli integration test binaries: %v", cliIntegrationBinsErr)
	}
}

func resolveCLICoverageDir() (string, error) {
	coverDir := os.Getenv(cliCoverDirEnv)
	if coverDir == "" {
		return "", nil
	}
	if err := os.MkdirAll(coverDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s %q: %w", cliCoverDirEnv, coverDir, err)
	}
	return coverDir, nil
}

// hostExeSuffix appends ".exe" to name on windows, matching bal build's own
// default output-path suffixing (see build.go) for a host (non-cross-compile)
// build.
func hostExeSuffix(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func cliIntegrationBalExecutableName(debugBuild bool) string {
	base := "bal"
	if debugBuild {
		base = "bal-debug"
	}
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func buildBalBinaryTo(repoRoot, coverDir, outputPath string, debugBuild bool) error {
	args := []string{"build"}
	if debugBuild {
		args = append(args, "-tags", "debug")
	}
	args = append(args, "-o", outputPath)
	if coverDir != "" {
		args = append(args, "-cover", "-coverpkg="+cliCoverPackages)
	}
	args = append(args, "./cli/cmd")

	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build bal binary: %w\n%s", err, string(out))
	}
	return nil
}

// balBinaryWithoutRuntimeStub builds a standalone bal binary into its own
// temp directory with no rt/ sibling, so executable.ResolveStub can't find a
// runner stub next to it — exercising the "stub not found" error path now
// that resolution is relative to the running bal binary's own directory
// (executable.DistributionDir) rather than $BAL_ENV.
func balBinaryWithoutRuntimeStub(t *testing.T, repoRoot, coverDir string) string {
	t.Helper()
	cliIntegrationNoRuntimeBalBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "bal-cli-test-no-runtime")
		if err != nil {
			cliIntegrationNoRuntimeBalBinErr = err
			return
		}
		cliIntegrationNoRuntimeBalBin = filepath.Join(dir, cliIntegrationBalExecutableName(false))
		cliIntegrationNoRuntimeBalBinErr = buildBalBinaryTo(repoRoot, coverDir, cliIntegrationNoRuntimeBalBin, false)
	})
	if cliIntegrationNoRuntimeBalBinErr != nil {
		t.Fatalf("building bal binary without runtime stub: %v", cliIntegrationNoRuntimeBalBinErr)
	}
	return cliIntegrationNoRuntimeBalBin
}

// buildBalBinaryWithRuntimeStubOverride builds a fresh bal binary with
// RuntimeStubPath baked in via -ldflags — the same link-time mechanism
// production packaging uses (see main.RuntimeStubPath) — so
// executable.ResolveStub's override branch can be exercised without a real
// rt/ distribution layout. Not cached like balBinaryWithoutRuntimeStub:
// each override path needs its own build.
func buildBalBinaryWithRuntimeStubOverride(t *testing.T, repoRoot, coverDir, overridePath string) string {
	t.Helper()
	dir := t.TempDir()
	balBin := filepath.Join(dir, cliIntegrationBalExecutableName(false))

	args := []string{"build", "-ldflags", "-X main.RuntimeStubPath=" + overridePath, "-o", balBin}
	if coverDir != "" {
		args = append(args, "-cover", "-coverpkg=./...")
	}
	args = append(args, "./cli/cmd")

	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building bal binary with RuntimeStubPath override: %v\n%s", err, string(out))
	}
	return balBin
}

// buildBalrtBinaryTo builds the slim runtime-only balrt stub that
// executable.ResolveStub looks up alongside bal's own distribution
// (<dist>/rt/<GOOS>-<GOARCH>/balrt). Built with the same -cover/-coverpkg
// instrumentation as bal/bal-debug when coverDir is set — balrt IS CLI code
// under test (executable.Run executes inside it, for every program bal
// build produces), and without this, that coverage is invisible even though
// TestBalBuildCorpus genuinely exercises it on every run: it only shows up
// once the produced binary is instrumented and BAL_GOCOVERDIR is set when
// running it (already threaded through in TestBalBuildCorpus's own
// runCLICommand call for the produced binary).
func buildBalrtBinaryTo(repoRoot, coverDir, outputPath string) error {
	args := []string{"build", "-o", outputPath}
	if coverDir != "" {
		args = append(args, "-cover", "-coverpkg=./...")
	}
	args = append(args, "./cli/internal/balrt")

	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build balrt binary: %w\n%s", err, string(out))
	}
	return nil
}

func assertBalCommandMatchesTxtarFragmentsForBinary(t *testing.T, balBin, repoRoot, coverDir string, args []string, txtarPathParts ...string) {
	t.Helper()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM (js/wasm)")
	}

	stdout, stderr, exitCode := runCLICommand(t, balBin, repoRoot, coverDir, args...)

	stdout = normalizePaths(test_util.NormalizeNewlines(stdout), repoRoot)
	stderr = normalizePaths(test_util.NormalizeNewlines(stderr), repoRoot)
	expectedPath := filepath.Join(append([]string{repoRoot, "corpus", "cli", "output"}, txtarPathParts...)...)

	expectedStdoutFragments, expectedStderr, expectedExitCode, err := test_util.LoadTxtarStdoutStderrExitcode(expectedPath)
	if err != nil {
		t.Fatalf("failed to parse txtar file %s: %v", expectedPath, err)
	}

	if stderr != expectedStderr {
		t.Fatalf("unexpected stderr for command %q with expected file %s\n%s", strings.Join(args, " "), expectedPath, test_util.FormatExpectedGot(expectedStderr, stderr))
	}
	if strconv.Itoa(exitCode) != expectedExitCode {
		t.Fatalf("unexpected exit code for command %q with expected file %s\n%s\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), expectedPath,
			test_util.FormatExpectedGot(expectedExitCode, strconv.Itoa(exitCode)), stdout, stderr)
	}
	combinedOut := stdout + "\n" + stderr
	for _, fragment := range strings.Split(expectedStdoutFragments, "\n") {
		if strings.TrimSpace(fragment) == "" {
			continue
		}
		if !strings.Contains(combinedOut, fragment) {
			t.Fatalf("output missing expected fragment %q for command %q with expected file %s\nstdout:\n%s\nstderr:\n%s", fragment, strings.Join(args, " "), expectedPath, stdout, stderr)
		}
	}
}

func runCLICommand(t *testing.T, balBin, repoRoot, coverDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	return runCLICommandWithEnv(t, balBin, repoRoot, coverDir, nil, args...)
}

// runCLICommandWithEnv is like runCLICommand but appends extraEnv on top of
// the process environment — needed for build scenarios that require a
// specific BAL_ENV (e.g. one with no runner stub installed, to exercise the
// missing-stub error path). Per os/exec, a later duplicate key wins, so
// extraEnv entries override any existing variable of the same name.
func runCLICommandWithEnv(t *testing.T, balBin, repoRoot, coverDir string, extraEnv []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	args = sandboxCLICommandArgs(t, repoRoot, args)

	cmd := exec.Command(balBin, args...)
	cmd.Dir = repoRoot
	env := os.Environ()
	if coverDir != "" {
		commandCoverDir := t.TempDir()
		env = append(env, "GOCOVERDIR="+commandCoverDir)
		defer mergeCLICoverageDir(t, commandCoverDir, coverDir)
	}
	env = append(env, extraEnv...)
	cmd.Env = env

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()
	if err == nil {
		return stdoutStr, stderrStr, 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdoutStr, stderrStr, exitErr.ExitCode()
	}
	t.Fatalf(
		"failed to execute command %q (repo: %s): %v\nstdout:\n%s\nstderr:\n%s",
		strings.Join(args, " "),
		repoRoot,
		err,
		stdoutStr,
		stderrStr,
	)
	return "", "", 0
}

func mergeCLICoverageDir(t *testing.T, fromDir, toDir string) {
	t.Helper()
	entries, err := os.ReadDir(fromDir)
	if err != nil {
		t.Fatalf("failed to read CLI coverage dir %q: %v", fromDir, err)
	}
	cliIntegrationCoverMerge.Lock()
	defer cliIntegrationCoverMerge.Unlock()
	if err := os.MkdirAll(toDir, 0o755); err != nil {
		t.Fatalf("failed to create CLI coverage dir %q: %v", toDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(fromDir, entry.Name())
		dst := filepath.Join(toDir, entry.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed to stat CLI coverage file %q: %v", dst, err)
		}
		if err := os.Rename(src, dst); err != nil {
			t.Fatalf("failed to move CLI coverage file %q to %q: %v", src, dst, err)
		}
	}
}

func sandboxCLICommandArgs(t *testing.T, repoRoot string, args []string) []string {
	t.Helper()

	const relTestdataRoot = "corpus/cli/testdata"
	absTestdataRoot := filepath.Join(repoRoot, filepath.FromSlash(relTestdataRoot))
	var sandboxTestdataRoot string
	rewritten := make([]string, len(args))
	copy(rewritten, args)

	for i, arg := range rewritten {
		if newArg, ok := rewriteCLIArgToSandbox(t, arg, relTestdataRoot, absTestdataRoot, &sandboxTestdataRoot); ok {
			rewritten[i] = newArg
		}
	}
	return rewritten
}

func rewriteCLIArgToSandbox(t *testing.T, arg, relTestdataRoot, absTestdataRoot string, sandboxTestdataRoot *string) (string, bool) {
	t.Helper()

	if strings.HasPrefix(arg, "--") {
		name, value, ok := strings.Cut(arg, "=")
		if !ok {
			return arg, false
		}
		if rewritten, changed := rewriteCLIPathToSandbox(t, value, relTestdataRoot, absTestdataRoot, sandboxTestdataRoot); changed {
			return name + "=" + rewritten, true
		}
		return arg, false
	}
	return rewriteCLIPathToSandbox(t, arg, relTestdataRoot, absTestdataRoot, sandboxTestdataRoot)
}

func rewriteCLIPathToSandbox(t *testing.T, value, relTestdataRoot, absTestdataRoot string, sandboxTestdataRoot *string) (string, bool) {
	t.Helper()

	if value == "" {
		return value, false
	}
	if filepath.IsAbs(value) {
		if rel, err := filepath.Rel(absTestdataRoot, value); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.Join(ensureCLITestdataSandbox(t, absTestdataRoot, sandboxTestdataRoot), rel), true
		}
		return value, false
	}

	nativeRelRoot := filepath.FromSlash(relTestdataRoot)
	if value == nativeRelRoot {
		return ensureCLITestdataSandbox(t, absTestdataRoot, sandboxTestdataRoot), true
	}
	if strings.HasPrefix(value, nativeRelRoot+string(filepath.Separator)) {
		rel := strings.TrimPrefix(value, nativeRelRoot+string(filepath.Separator))
		return filepath.Join(ensureCLITestdataSandbox(t, absTestdataRoot, sandboxTestdataRoot), rel), true
	}
	if strings.HasPrefix(value, relTestdataRoot+"/") {
		rel := strings.TrimPrefix(value, relTestdataRoot+"/")
		return filepath.Join(ensureCLITestdataSandbox(t, absTestdataRoot, sandboxTestdataRoot), filepath.FromSlash(rel)), true
	}
	return value, false
}

func ensureCLITestdataSandbox(t *testing.T, absTestdataRoot string, sandboxTestdataRoot *string) string {
	t.Helper()
	if *sandboxTestdataRoot != "" {
		return *sandboxTestdataRoot
	}
	root := filepath.Join(t.TempDir(), "corpus", "cli", "testdata")
	copyDir(t, absTestdataRoot, root)
	*sandboxTestdataRoot = root
	return root
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to read directory %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dst, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("failed to stat %s: %v", srcPath, err)
		}
		if info.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		copyFile(t, srcPath, dstPath, info.Mode())
	}
}

func copyFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("failed to open %s: %v", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		t.Fatalf("failed to create %s: %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		t.Fatalf("failed to copy %s to %s: %v", src, dst, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("failed to close %s: %v", dst, err)
	}
}

func listBalRunCorpusPaths(t *testing.T, dir string, balFilesOnly bool) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory %s: %v", dir, err)
	}
	paths := make([]string, 0)
	for _, entry := range entries {
		if balFilesOnly {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bal") {
				continue
			}
		} else if !entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(paths)
	return paths
}

func runBalRunCorpusCase(t *testing.T, balBin, repoRoot, coverDir, outputsRoot, runPath, outputKey string) {
	t.Helper()
	t.Run(strings.ReplaceAll(outputKey, string(filepath.Separator), "_"), func(t *testing.T) {
		t.Parallel()
		stdout, stderr, exitCode := runCLICommand(t, balBin, repoRoot, coverDir, "run", runPath)
		expectedPath := filepath.Join(outputsRoot, outputKey+".txtar")
		actualOutput := normalizePaths(test_util.NormalizeNewlines(stdout), repoRoot)
		actualError := normalizePaths(test_util.NormalizeNewlines(stderr), repoRoot)
		actualExitCode := strconv.Itoa(exitCode)

		expectedOutput, expectedError, expectedExitCode, err := test_util.LoadTxtarStdoutStderrExitcode(expectedPath)
		if err != nil {
			t.Fatalf("failed to parse txtar file %s: %v", expectedPath, err)
		}
		if expectedOutput != actualOutput || expectedError != actualError || expectedExitCode != actualExitCode {
			t.Fatalf(
				"unexpected output for %s\nexpected stdout:\n%s\nactual stdout:\n%s\nexpected stderr:\n%s\nactual stderr:\n%s\nexpected exitcode: %s\nactual exitcode: %s",
				runPath,
				expectedOutput,
				actualOutput,
				expectedError,
				actualError,
				expectedExitCode,
				actualExitCode,
			)
		}
	})
}
