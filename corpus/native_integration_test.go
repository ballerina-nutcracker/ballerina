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
	"runtime"
	"slices"
	"strings"
	"testing"

	interpsrc "ballerina-lang-go"
	"ballerina-lang-go/lib/stdlibs"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/test_util"
)

// nativeTestRepoPath returns the absolute path to the bala test repository.
func nativeTestRepoPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "projects", "testdata", "repo", "bala"))
	if err != nil {
		t.Fatalf("resolving test repo path: %v", err)
	}
	return p
}

// TestNativeGoSourceFS_Pipeline loads a go-platform bala via the file-system
// repository, calls NativeGoSourceFS(), and verifies the returned FS exposes
// the expected Go source file. This exercises the NativeGoSourceFS path in
// bala_project.go end-to-end, including the fsys → fs.Sub plumbing.
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

	// The native Go source file must be accessible via the returned FS.
	f, err := goFS.Open("nativepkg.go")
	require.NoError(err)
	require.NoError(f.Close())

	// No other packages in this bala have a native/ dir, so NativeGoSourceFS
	// for a non-go-platform bala must return an error or an empty FS.
	anyRepo := projects.NewFileSystemRepository(os.DirFS(nativeTestRepoPath(t)), ".")
	anyPkg, err := anyRepo.GetPackage(context.Background(), "mockorg", "greetpkg", "1.0.0", projects.ResolutionOptions{})
	require.NoError(err)
	require.NotNil(anyPkg)
	greetBP, ok := anyPkg.Project().(*projects.BalaProject)
	if !ok {
		t.Fatalf("expected BalaProject, got %T", anyPkg.Project())
	}
	assert.False(strings.HasPrefix(greetBP.Platform(), "go"), "greetpkg should be 'any' platform")

	// any-platform balas have no native/ dir; fs.Sub still succeeds but the
	// directory is empty — opening a .go file must fail.
	anyNativeFS, err := greetBP.NativeGoSourceFS()
	if err == nil {
		_, statErr := fs.Stat(anyNativeFS, ".")
		// Either the Sub fails outright or the directory doesn't exist.
		assert.NotNil(statErr, "expected no-native bala's FS to be inaccessible or empty")
	}
}

// TestNativeResolution_EmbeddedStdlibFilter loads the multi-org native test
// project and walks its resolved dependency graph. It verifies that
// go-platform packages that are part of the embedded stdlib bundle are
// distinguishable from user-defined native packages, mirroring the
// isEmbeddedPackage check in cli/cmd/run.go.
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

	// ballerina/io is bundled in the embedded stdlib — it must not reach the
	// native interpreter rebuild path.
	if !slices.Contains(embeddedNative, "ballerina/io") {
		t.Errorf("expected ballerina/io in embedded native list, got %v", embeddedNative)
	}

	// mockorg/nativepkg and acmeorg/calcpkg are user-defined native packages
	// that do require a rebuild.
	if !slices.Contains(userNative, "mockorg/nativepkg") {
		t.Errorf("expected mockorg/nativepkg in user native list, got %v", userNative)
	}
	if !slices.Contains(userNative, "acmeorg/calcpkg") {
		t.Errorf("expected acmeorg/calcpkg in user native list, got %v", userNative)
	}

	// No user-defined native package must appear in the embedded list.
	for _, name := range userNative {
		if slices.Contains(embeddedNative, name) {
			t.Errorf("user-defined native package %q must not appear in embedded list", name)
		}
	}
}

// TestInterpsrc_ExtractAndCache exercises interpsrc.ExtractTo end-to-end:
// the first call extracts the embedded source tree to a temp directory, and
// a second call with the same version skips re-extraction (fast path).
func TestInterpsrc_ExtractAndCache(t *testing.T) {
	t.Parallel()
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	cacheRoot := t.TempDir()
	const version = "test-v0.0.1"

	// First call: must extract and return a valid path with go.mod.
	dir1, err := interpsrc.ExtractTo(cacheRoot, version)
	require.NoError(err)
	require.NotEmpty(dir1)

	_, err = os.Stat(filepath.Join(dir1, "go.mod"))
	require.NoError(err, "go.mod must exist after extraction")

	// Second call: go.mod already exists, so extraction is skipped.
	// The returned path must be identical.
	dir2, err := interpsrc.ExtractTo(cacheRoot, version)
	require.NoError(err)
	assert.Equal(dir1, dir2, "second call must return the cached path without re-extracting")
}

// TestNativeRunner_EmbeddedOnlyProjectNoRebuild runs `bal run` on a project
// whose only stdlib dependency is ballerina/io (an embedded native package).
// It asserts that no "info: building native interpreter" message appears in
// stderr, confirming that isEmbeddedPackage correctly suppresses spurious
// native interpreter rebuilds.
func TestNativeRunner_EmbeddedOnlyProjectNoRebuild(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
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

// TestNativeRunner_ColdBuildAndCacheHit runs `bal run` on a project with a
// user-defined native Go dependency twice, using a temp project directory.
//
//   - First run (cold): stderr must contain "info: building native interpreter"
//     and stdout must match expected output.
//   - Second run (cache hit): stderr must NOT contain "info: building", confirming
//     that loadCachedRunner's fingerprint match bypasses the rebuild.
func TestNativeRunner_ColdBuildAndCacheHit(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "js" || runtime.GOARCH == "wasm" {
		t.Skip("skipping CLI integration test on WASM")
	}

	balBin, repoRoot, coverDir := integrationTestBalCLI(t, false)

	// Build a temp Ballerina home whose central cache contains the testdata
	// native packages so the CLI can resolve them.
	tempHome := t.TempDir()
	centralCache := filepath.Join(tempHome, "repositories", "central.ballerina.io", "bala")
	srcRepo := filepath.Join(repoRoot, "projects", "testdata", "repo", "bala")
	copyDir(t, srcRepo, centralCache)

	// Copy the project to a temp dir so the output binary goes to a fresh location
	// (preventing leftover binaries from a previous run from masking cache misses).
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
		return runCLICommandWithEnv(t, balBin, repoRoot, []string{"run", tempProject}, env)
	}

	// First run: cold build.
	stdout1, stderr1, code1 := runNative()
	if code1 != 0 {
		t.Fatalf("first run failed (exit %d)\nstdout: %s\nstderr: %s", code1, stdout1, stderr1)
	}
	if !strings.Contains(stderr1, "info: building native interpreter") {
		t.Errorf("first run: expected 'info: building native interpreter' in stderr\nstderr: %s", stderr1)
	}

	// Second run: should hit the fingerprint cache and skip the build.
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

// runCLICommandWithEnv runs balBin with args under the given working directory
// and environment. It is like runCLICommand but accepts a custom env slice.
func runCLICommandWithEnv(t *testing.T, balBin, workDir string, args, env []string) (stdout, stderr string, exitCode int) {
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
