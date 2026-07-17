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
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime" // aliased: this file also imports ballerina-lang-go/runtime as runtime
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	interpsrc "ballerina-lang-go"
	"ballerina-lang-go/lib/stdlibs"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testharness"

	// Blank-import native packages so their init() registers extern
	// functions before tests run; testdata isn't in ./... builds otherwise.
	_ "ballerina-lang-go/projects/testdata/repo/bala/acmeorg/calcpkg/1.0.0/go1.26/native"
	_ "ballerina-lang-go/projects/testdata/repo/bala/mockorg/nativepkg/1.0.0/go1.26/native"
)

const nativeTestDataDir = "extern/testdata"

// TestNativeMultiOrgPackages verifies native Go packages from multiple orgs
// resolve and run alongside pure-Ballerina packages with transitive deps.
func TestNativeMultiOrgPackages(t *testing.T) {
	t.Parallel()
	projectDir := filepath.Join(nativeTestDataDir, "native-multi-org-v")
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatal(err)
	}

	testRepoPath, err := filepath.Abs(filepath.Join("..", "projects", "testdata", "repo", "bala"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := projects.Load(
		os.DirFS(absProjectDir),
		".",
		projects.ProjectLoadConfig{
			Repositories: []projects.Repository{
				// Bundled stdlib must come first (same order as defaultRepositories).
				projects.NewFileSystemRepository(stdlibs.FS, "."),
				projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
			},
		},
	)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	pkg := result.Project().CurrentPackage()
	compilation := pkg.Compilation()
	if compilation.DiagnosticResult().HasErrors() {
		for _, d := range compilation.DiagnosticResult().Diagnostics() {
			t.Logf("diagnostic: %v", d)
		}
		t.Fatal("compilation had errors")
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()

	pal := testharness.NewTestPal()
	rt := runtime.NewRuntime(pal.Platform(), result.Project().Environment().TypeEnv())

	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			t.Fatalf("runtime error: %v", err)
		}
	}

	const txtarPath = "extern/output/native-multi-org-v.txtar"
	actualStdout := test_util.NormalizeNewlines(pal.Stdout())
	actualStderr := test_util.NormalizeNewlines(pal.Stderr())

	if *update {
		if test_util.UpdateTxtarArchiveIfNeeded(t, txtarPath, test_util.TxtarFilesStdoutStderr(actualStdout, actualStderr)) {
			t.Fatalf("Updated expected file: %s", txtarPath)
		}
		return
	}

	expectedStdout, expectedStderr, err := test_util.LoadTxtarStdoutStderr(txtarPath)
	if err != nil {
		t.Fatalf("failed to load golden file %s: %v", txtarPath, err)
	}
	expectedStdout = test_util.NormalizeNewlines(expectedStdout)
	expectedStderr = test_util.NormalizeNewlines(expectedStderr)

	if actualStdout != expectedStdout {
		t.Errorf("stdout mismatch\n%s", test_util.FormatExpectedGot(expectedStdout, actualStdout))
	}
	if actualStderr != expectedStderr {
		t.Errorf("stderr mismatch\n%s", test_util.FormatExpectedGot(expectedStderr, actualStderr))
	}
}

// nativeTestRepoPath returns the absolute path to the bala test repository.
func nativeTestRepoPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "projects", "testdata", "repo", "bala"))
	if err != nil {
		t.Fatalf("resolving test repo path: %v", err)
	}
	return p
}

// TestNativeGoSourceFS_Pipeline loads a go-platform bala and verifies
// NativeGoSourceFS() exposes the expected Go source file end-to-end.
func TestNativeGoSourceFS_Pipeline(t *testing.T) {
	t.Parallel()
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	repo := projects.NewFileSystemRepository(os.DirFS(nativeTestRepoPath(t)), ".")
	pkg, err := repo.GetPackage(context.Background(), "mockorg", "nativepkg", "1.0.0", projects.ResolutionOptions{})
	require.NoError(err)
	require.NotNil(pkg)

	bp, ok := pkg.Project().(*projects.BalaProject)
	if !ok {
		t.Fatalf("expected BalaProject, got %T", pkg.Project())
	}
	assert.True(strings.HasPrefix(bp.Platform(), "go"), "expected go-platform bala")

	goFS, err := bp.NativeGoSourceFS()
	require.NoError(err)
	require.NotNil(goFS)

	f, err := goFS.Open("nativepkg.go")
	require.NoError(err)
	require.NoError(f.Close())

	// A non-go-platform bala has no native/ dir, so this must error or be empty.
	anyRepo := projects.NewFileSystemRepository(os.DirFS(nativeTestRepoPath(t)), ".")
	anyPkg, err := anyRepo.GetPackage(context.Background(), "mockorg", "greetpkg", "1.0.0", projects.ResolutionOptions{})
	require.NoError(err)
	require.NotNil(anyPkg)
	greetBP, ok := anyPkg.Project().(*projects.BalaProject)
	if !ok {
		t.Fatalf("expected BalaProject, got %T", anyPkg.Project())
	}
	assert.False(strings.HasPrefix(greetBP.Platform(), "go"), "greetpkg should be 'any' platform")

	// any-platform balas have no native/ dir; fs.Sub succeeds but is empty.
	anyNativeFS, err := greetBP.NativeGoSourceFS()
	if err == nil {
		_, statErr := fs.Stat(anyNativeFS, ".")
		assert.NotNil(statErr, "expected no-native bala's FS to be inaccessible or empty")
	}
}

// TestNativeResolution_EmbeddedStdlibFilter verifies embedded-stdlib
// go-platform packages are distinguishable from user-defined native ones,
// mirroring the isEmbeddedPackage check in cli/cmd/run.go.
func TestNativeResolution_EmbeddedStdlibFilter(t *testing.T) {
	t.Parallel()
	require := test_util.NewRequire(t)

	absProjectDir, err := filepath.Abs(filepath.Join(nativeTestDataDir, "native-multi-org-v"))
	require.NoError(err)

	result, err := projects.Load(
		os.DirFS(absProjectDir),
		".",
		projects.ProjectLoadConfig{
			Repositories: []projects.Repository{
				projects.NewFileSystemRepository(stdlibs.FS, "."),
				projects.NewFileSystemRepository(os.DirFS(nativeTestRepoPath(t)), "."),
			},
		},
	)
	require.NoError(err)

	pkg := result.Project().CurrentPackage()
	resolution := pkg.Resolution()
	require.NotNil(resolution)

	cache := result.Project().Environment().PackageCache()

	var embeddedNative, userNative []string
	for _, pkgDesc := range resolution.DependencyGraph().ToTopologicallySortedList() {
		dep := cache.Get(pkgDesc.Org().Value(), pkgDesc.Name().Value(), pkgDesc.Version().String())
		if dep == nil {
			continue
		}
		bp, ok := dep.Project().(*projects.BalaProject)
		if !ok || !strings.HasPrefix(bp.Platform(), "go") {
			continue
		}
		desc := bp.CurrentPackage().Descriptor()
		label := desc.Org().Value() + "/" + desc.Name().Value()
		if stdlibs.Contains(desc.Org().Value(), desc.Name().Value(), desc.Version().String()) {
			embeddedNative = append(embeddedNative, label)
		} else {
			userNative = append(userNative, label)
		}
	}

	// ballerina/io is embedded stdlib — must not need a rebuild.
	if !slices.Contains(embeddedNative, "ballerina/io") {
		t.Errorf("expected ballerina/io in embedded native list, got %v", embeddedNative)
	}

	// These are user-defined native packages — they do require a rebuild.
	if !slices.Contains(userNative, "mockorg/nativepkg") {
		t.Errorf("expected mockorg/nativepkg in user native list, got %v", userNative)
	}
	if !slices.Contains(userNative, "acmeorg/calcpkg") {
		t.Errorf("expected acmeorg/calcpkg in user native list, got %v", userNative)
	}

	for _, name := range userNative {
		if slices.Contains(embeddedNative, name) {
			t.Errorf("user-defined native package %q must not appear in embedded list", name)
		}
	}
}

// TestInterpsrc_ExtractAndCache checks interpsrc.ExtractTo extracts once,
// then skips re-extraction on a second call with the same version.
func TestInterpsrc_ExtractAndCache(t *testing.T) {
	t.Parallel()
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	cacheRoot := t.TempDir()
	const version = "test-v0.0.1"

	dir1, err := interpsrc.ExtractTo(cacheRoot, version)
	require.NoError(err)
	require.NotEmpty(dir1)

	_, err = os.Stat(filepath.Join(dir1, "go.mod"))
	require.NoError(err, "go.mod must exist after extraction")

	// go.mod already exists, so this call should skip extraction and return
	// the same path.
	dir2, err := interpsrc.ExtractTo(cacheRoot, version)
	require.NoError(err)
	assert.Equal(dir1, dir2, "second call must return the cached path without re-extracting")
}

// TestNativeRunner_EmbeddedOnlyProjectNoRebuild checks a project depending
// only on embedded ballerina/io never triggers a native interpreter rebuild.
func TestNativeRunner_EmbeddedOnlyProjectNoRebuild(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	balFile := filepath.Join(repoRoot, "corpus", "cli", "testdata", "run", "single-bal-files", "run-and-print.bal")
	_, stderr, exitCode := runCLICommand(t, balBin, repoRoot, coverDir, "run", balFile)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstderr: %s", exitCode, stderr)
	}
	if strings.Contains(stderr, "info: building native interpreter") {
		t.Errorf("unexpected native interpreter build triggered for an embedded-only project\nstderr: %s", stderr)
	}
}

// TestNativeRunner_ColdBuildAndCacheHit runs a native-dependency project
// twice: the first run must build the interpreter (cold), the second must
// hit loadCachedRunner's fingerprint cache and skip the rebuild.
func TestNativeRunner_ColdBuildAndCacheHit(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}

	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	// Temp Ballerina home with the testdata native packages in its central cache.
	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	// Fresh project copy so no leftover binary can mask a cache miss.
	srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
	tempProject := t.TempDir()
	copyDir(t, srcProject, tempProject)

	runNative := func() (stdout, stderr string, code int) {
		env := append(os.Environ(),
			"BAL_ENV="+tempHome,
			"BALLERINA_SRC="+repoRoot,
		)
		if coverDir != "" {
			env = append(env, "GOCOVERDIR="+coverDir)
		}
		return runNativeCLICommandWithEnv(t, balBin, repoRoot, []string{"run", tempProject}, env)
	}

	stdout1, stderr1, code1 := runNative()
	if code1 != 0 {
		t.Fatalf("first run failed (exit %d)\nstdout: %s\nstderr: %s", code1, stdout1, stderr1)
	}
	if !strings.Contains(stderr1, "info: building native interpreter") {
		t.Errorf("first run: expected 'info: building native interpreter' in stderr\nstderr: %s", stderr1)
	}

	stdout2, stderr2, code2 := runNative()
	if code2 != 0 {
		t.Fatalf("second run failed (exit %d)\nstdout: %s\nstderr: %s", code2, stdout2, stderr2)
	}
	if strings.Contains(stderr2, "info: building native interpreter") {
		t.Errorf("second run: unexpected native interpreter rebuild (cache miss)\nstderr: %s", stderr2)
	}
	if test_util.NormalizeNewlines(stdout1) != test_util.NormalizeNewlines(stdout2) {
		t.Errorf("output differs between cold and cached run\nfirst:  %s\nsecond: %s", stdout1, stdout2)
	}
}

// runNativeCLICommandWithEnv is like runCLICommand but takes a custom env
// slice directly, skipping runCLICommandWithEnv's GOCOVERDIR/sandboxing
// logic — callers here already isolate their project and manage their own env.
func runNativeCLICommandWithEnv(t *testing.T, balBin, workDir string, args, env []string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(balBin, args...)
	cmd.Dir = workDir
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
		"failed to execute command %q (workDir: %s): %v\nstdout:\n%s\nstderr:\n%s",
		strings.Join(args, " "),
		workDir,
		err,
		stdoutStr,
		stderrStr,
	)
	return "", "", 0
}

// TestNativeGoSourceFS_MissingNativeDirDespiteGoPlatform covers a
// go-platform bala with no native/ directory (e.g. malformed/hand-edited).
// Built in-memory, mirroring mockorg/nativepkg's layout minus native/.
func TestNativeGoSourceFS_MissingNativeDirDespiteGoPlatform(t *testing.T) {
	t.Parallel()
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	balaFS := fstest.MapFS{
		"mockorg/nonativepkg/1.0.0/go1.26/Bala.toml": &fstest.MapFile{Data: []byte(`[bala]
schema_version = "4"

[build]
ballerina_version      = ""
implementation_vendor  = "WSO2"
language_spec_version  = "2024R1"
platform               = "go1.26"
`)},
		"mockorg/nonativepkg/1.0.0/go1.26/Ballerina.toml": &fstest.MapFile{Data: []byte(`[package]
org     = "mockorg"
name    = "nonativepkg"
version = "1.0.0"
`)},
		"mockorg/nonativepkg/1.0.0/go1.26/Dependencies.toml": &fstest.MapFile{Data: []byte(`[ballerina]
dependencies-toml-version = "2"

[[package]]
org     = "mockorg"
name    = "nonativepkg"
version = "1.0.0"
`)},
		"mockorg/nonativepkg/1.0.0/go1.26/nonativepkg.bal": &fstest.MapFile{Data: []byte(
			"public function hello() returns string = external;\n")},
	}

	repo := projects.NewFileSystemRepository(balaFS, ".")
	pkg, err := repo.GetPackage(context.Background(), "mockorg", "nonativepkg", "1.0.0", projects.ResolutionOptions{})
	require.NoError(err)
	require.NotNil(pkg)

	bp, ok := pkg.Project().(*projects.BalaProject)
	if !ok {
		t.Fatalf("expected BalaProject, got %T", pkg.Project())
	}
	assert.True(strings.HasPrefix(bp.Platform(), "go"), "test fixture must declare a go-platform bala")

	goFS, err := bp.NativeGoSourceFS()
	if err == nil {
		_, statErr := fs.Stat(goFS, ".")
		assert.NotNil(statErr, "expected a go-platform bala with no native/ directory to error or expose an inaccessible FS")
	}
}

// setupNativeTestFixtures copies native-multi-org-v and its bala deps into
// fresh temp dirs, returning the BAL_ENV root and project dir, so tests can
// mutate them without touching the checked-in fixtures.
func setupNativeTestFixtures(t *testing.T, repoRoot string) (balEnv, projectDir string) {
	t.Helper()
	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	srcProject := filepath.Join(repoRoot, "corpus", "extern", "testdata", "native-multi-org-v")
	tempProject := t.TempDir()
	copyDir(t, srcProject, tempProject)

	return tempHome, tempProject
}

// envWithoutVars returns base with entries matching any of names removed,
// so the caller can safely append its own value for that key.
func envWithoutVars(base []string, names ...string) []string {
	filtered := make([]string, 0, len(base))
	for _, e := range base {
		blocked := false
		for _, name := range names {
			if strings.HasPrefix(e, name+"=") {
				blocked = true
				break
			}
		}
		if !blocked {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// TestNativeRunner_GoToolchainUnavailable covers a native-dependency
// project when Go isn't on PATH: must fail with a clear "require Go ...
// installed" error that actually reaches the user (cobra's SilenceErrors
// means every error path needs its own explicit print — this test caught a
// case that was missing one, fixed in run.go).
func TestNativeRunner_GoToolchainUnavailable(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	tempHome, tempProject := setupNativeTestFixtures(t, repoRoot)

	// Empty dir as PATH so exec.LookPath("go") fails in the child process only.
	emptyPathDir := t.TempDir()
	env := append(envWithoutVars(os.Environ(), "PATH", "Path"),
		"BAL_ENV="+tempHome,
		"BALLERINA_SRC="+repoRoot,
		"PATH="+emptyPathDir,
	)
	if coverDir != "" {
		env = append(env, "GOCOVERDIR="+coverDir)
	}

	_, stderr, code := runNativeCLICommandWithEnv(t, balBin, repoRoot, []string{"run", tempProject}, env)
	if code == 0 {
		t.Fatalf("expected a non-zero exit code when the Go toolchain is unavailable, got 0\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "require Go") || !strings.Contains(stderr, "installed") {
		t.Errorf("expected a clear 'require Go ... installed' error, got:\n%s", stderr)
	}
}

// TestBalBuildNativeDependencyGoToolchainUnavailable is
// TestNativeRunner_GoToolchainUnavailable's counterpart for `bal build`:
// buildNativeStub must fail the same way (chooseNativeExecutor rejecting an
// unavailable toolchain) rather than only being covered for bal run's
// execWithNativeRunner path.
func TestBalBuildNativeDependencyGoToolchainUnavailable(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	tempHome, tempProject := setupNativeTestFixtures(t, repoRoot)

	emptyPathDir := t.TempDir()
	env := append(envWithoutVars(os.Environ(), "PATH", "Path"),
		"BAL_ENV="+tempHome,
		"BALLERINA_SRC="+repoRoot,
		"PATH="+emptyPathDir,
	)
	if coverDir != "" {
		env = append(env, "GOCOVERDIR="+coverDir)
	}

	_, stderr, code := runNativeCLICommandWithEnv(t, balBin, repoRoot, []string{"build", tempProject}, env)
	if code == 0 {
		t.Fatalf("expected a non-zero exit code when the Go toolchain is unavailable, got 0\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "require Go") || !strings.Contains(stderr, "installed") {
		t.Errorf("expected a clear 'require Go ... installed' error, got:\n%s", stderr)
	}
}

// TestNativeRunner_InterpreterSourceUnresolvable covers a native-dependency
// project when the embedded-source cache extraction can't succeed. The
// dev-build extraction path ignores BAL_ENV and uses os.TempDir(), so this
// redirects TMPDIR/TMP/TEMP (in the child process only) to a path blocked by
// a regular file, without touching the real shared temp dir.
func TestNativeRunner_InterpreterSourceUnresolvable(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	tempHome, tempProject := setupNativeTestFixtures(t, repoRoot)

	blockingFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockingFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("writing blocking file: %v", err)
	}

	env := append(envWithoutVars(os.Environ(), "BALLERINA_SRC", "TMPDIR", "TMP", "TEMP"),
		"BAL_ENV="+tempHome,
		"TMPDIR="+blockingFile,
		"TMP="+blockingFile,
		"TEMP="+blockingFile,
	)
	if coverDir != "" {
		env = append(env, "GOCOVERDIR="+coverDir)
	}

	_, stderr, code := runNativeCLICommandWithEnv(t, balBin, repoRoot, []string{"run", tempProject}, env)
	if code == 0 {
		t.Fatalf("expected a non-zero exit code when the interpreter source cannot be resolved, got 0\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "interpreter source") {
		t.Errorf("expected a clear 'interpreter source' error, got:\n%s", stderr)
	}
}

// TestNativeRunner_OutputPathBlocked covers a regular file blocking the
// output directory LocalExecutor needs to create — must fail cleanly.
func TestNativeRunner_OutputPathBlocked(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	tempHome, tempProject := setupNativeTestFixtures(t, repoRoot)

	blockedPath := filepath.Join(tempProject, "target")
	if err := os.WriteFile(blockedPath, []byte("blocking file"), 0o644); err != nil {
		t.Fatalf("writing blocking file: %v", err)
	}

	env := append(os.Environ(), "BAL_ENV="+tempHome, "BALLERINA_SRC="+repoRoot)
	if coverDir != "" {
		env = append(env, "GOCOVERDIR="+coverDir)
	}

	_, stderr, code := runNativeCLICommandWithEnv(t, balBin, repoRoot, []string{"run", tempProject}, env)
	if code == 0 {
		t.Fatalf("expected a non-zero exit code when the output directory is blocked, got 0\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "creating output directory") {
		t.Errorf("expected a clear 'creating output directory' error, got:\n%s", stderr)
	}
}

// TestNativeRunner_FingerprintInvalidatesOnSourceChange checks that editing
// a native dependency's Go source invalidates the fingerprint and triggers
// a rebuild, instead of serving stale output.
func TestNativeRunner_FingerprintInvalidatesOnSourceChange(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "js" || goruntime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}
	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)
	tempHome, tempProject := setupNativeTestFixtures(t, repoRoot)
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")

	env := append(os.Environ(), "BAL_ENV="+tempHome, "BALLERINA_SRC="+repoRoot)
	if coverDir != "" {
		env = append(env, "GOCOVERDIR="+coverDir)
	}
	run := func() (stdout, stderr string, code int) {
		return runNativeCLICommandWithEnv(t, balBin, repoRoot, []string{"run", tempProject}, env)
	}

	// First run: cold build.
	_, stderr1, code1 := run()
	if code1 != 0 {
		t.Fatalf("first run failed (exit %d): %s", code1, stderr1)
	}
	if !strings.Contains(stderr1, "info: building native interpreter") {
		t.Fatalf("expected a cold build, got:\n%s", stderr1)
	}

	// Modify the cached native source — must invalidate the fingerprint.
	const wantOriginal, wantModified = "hello from native Go", "hello from MODIFIED native Go"
	nativeGoFile := filepath.Join(centralCache, "mockorg", "nativepkg", "1.0.0", "go1.26", "native", "nativepkg.go")
	original := mustReadFileBytes(t, nativeGoFile)
	modified := bytes.Replace(original, []byte(wantOriginal), []byte(wantModified), 1)
	if bytes.Equal(original, modified) {
		t.Fatalf("test fixture assumption broken: expected string %q not found in %s", wantOriginal, nativeGoFile)
	}
	if err := os.WriteFile(nativeGoFile, modified, 0o644); err != nil {
		t.Fatalf("writing modified native source: %v", err)
	}

	stdout2, stderr2, code2 := run()
	if code2 != 0 {
		t.Fatalf("second run failed (exit %d): %s", code2, stderr2)
	}
	if !strings.Contains(stderr2, "info: building native interpreter") {
		t.Fatalf("expected a rebuild after modifying native source, got a cache hit:\n%s", stderr2)
	}
	if !strings.Contains(stdout2, wantModified) {
		t.Fatalf("expected updated output reflecting the modified native source, got:\n%s", stdout2)
	}
}
