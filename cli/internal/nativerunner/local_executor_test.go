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

package nativerunner

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"ballerina-lang-go/cli/internal/nativeexec"
)

func TestVersionAtLeast(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.26", "1.26", true},
		{"1.26.1", "1.26", true},
		{"1.25", "1.26", false},
		{"2.0", "1.26", true},
		{"1.26", "1.26.0", true},
		{"1.26.0", "1.26", true},
		{"1.0", "2.0", false},
		{"1.26rc1", "1.26", false},
		{"1.26beta2", "1.26", false},
	}
	for _, tc := range cases {
		got := versionAtLeast(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("versionAtLeast(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestModuleDirName(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{"ballerinax/redis-native", "ballerinax_redis-native"},
		{"a/b/c", "a_b_c"},
		{"noSlash", "noSlash"},
		{"example.com/org/pkg", "example.com_org_pkg"},
	}
	for _, tc := range cases {
		got := moduleDirName(tc.in)
		if got != tc.want {
			t.Errorf("moduleDirName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWriteNativeFiles_CopiesGoFiles(t *testing.T) {
	t.Parallel()
	srcFS := fstest.MapFS{
		"main.go":       {Data: []byte("package main\n")},
		"sub/helper.go": {Data: []byte("package sub\n")},
		"README.md":     {Data: []byte("# readme\n")},
		"config.yaml":   {Data: []byte("key: value\n")},
	}
	payload := &nativeexec.GoSourcePayload{GoFiles: srcFS, Module: "example.com/pkg"}

	dir := t.TempDir()
	if err := writeNativeFiles(dir, payload); err != nil {
		t.Fatalf("writeNativeFiles: %v", err)
	}

	checkFileContent(t, filepath.Join(dir, "main.go"), "package main\n")
	checkFileContent(t, filepath.Join(dir, "sub", "helper.go"), "package sub\n")

	for _, name := range []string{"README.md", "config.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("non-.go file %q must not be copied", name)
		}
	}
}

func TestWritePatchedGoMod_AppendsRequireReplace(t *testing.T) {
	t.Parallel()
	interpRoot := t.TempDir()
	origMod := "module ballerina-lang-go\n\ngo 1.26\n"
	mustWriteFile(t, filepath.Join(interpRoot, "go.mod"), origMod)
	mustWriteFile(t, filepath.Join(interpRoot, "go.sum"), "")

	tmpDir := t.TempDir()
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{Module: "example.com/mypkg"},
	}

	patchedModFile, err := writePatchedGoMod(tmpDir, interpRoot, payloads, tmpDir)
	if err != nil {
		t.Fatalf("writePatchedGoMod: %v", err)
	}

	content := mustReadFile(t, patchedModFile)
	if !strings.Contains(content, "require example.com/mypkg v0.0.0") {
		t.Errorf("patched go.mod missing require directive:\n%s", content)
	}
	if !strings.Contains(content, "replace example.com/mypkg =>") {
		t.Errorf("patched go.mod missing replace directive:\n%s", content)
	}
	// Original module declaration must be preserved.
	if !strings.Contains(content, "module ballerina-lang-go") {
		t.Errorf("patched go.mod missing original module declaration:\n%s", content)
	}
}

func TestWritePatchedGoMod_MultiplePayloads(t *testing.T) {
	t.Parallel()
	interpRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(interpRoot, "go.mod"), "module ballerina-lang-go\n\ngo 1.26\n")
	mustWriteFile(t, filepath.Join(interpRoot, "go.sum"), "")

	tmpDir := t.TempDir()
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{Module: "example.com/pkgA"},
		&nativeexec.GoSourcePayload{Module: "example.com/pkgB"},
	}

	patchedModFile, err := writePatchedGoMod(tmpDir, interpRoot, payloads, tmpDir)
	if err != nil {
		t.Fatalf("writePatchedGoMod: %v", err)
	}

	content := mustReadFile(t, patchedModFile)
	for _, mod := range []string{"example.com/pkgA", "example.com/pkgB"} {
		if !strings.Contains(content, "require "+mod) {
			t.Errorf("patched go.mod missing require for %s:\n%s", mod, content)
		}
		if !strings.Contains(content, "replace "+mod) {
			t.Errorf("patched go.mod missing replace for %s:\n%s", mod, content)
		}
	}
}

func TestWritePatchedGoMod_WritesPatchedGoSum(t *testing.T) {
	t.Parallel()
	interpRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(interpRoot, "go.mod"), "module ballerina-lang-go\n\ngo 1.26\n")
	mustWriteFile(t, filepath.Join(interpRoot, "go.sum"), "github.com/foo/bar v1.0.0 h1:xxx\n")

	tmpDir := t.TempDir()
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{Module: "example.com/pkg"},
	}

	_, err := writePatchedGoMod(tmpDir, interpRoot, payloads, tmpDir)
	if err != nil {
		t.Fatalf("writePatchedGoMod: %v", err)
	}

	sumContent := mustReadFile(t, filepath.Join(tmpDir, "patched-go.sum"))
	if !strings.Contains(sumContent, "github.com/foo/bar") {
		t.Errorf("patched-go.sum must copy interpreter go.sum content:\n%s", sumContent)
	}
}

func TestWritePatchedGoMod_MissingGoMod(t *testing.T) {
	t.Parallel()
	_, err := writePatchedGoMod(t.TempDir(), "/nonexistent/root", nil, t.TempDir())
	if err == nil {
		t.Error("expected error when interpreter go.mod is missing")
	}
}

// TestNew_DefaultsToCliCmdTarget mirrors bal run's use of New: it must target
// the full CLI (cli/cmd), matching the pre-existing hardcoded behavior New
// replaced, so bal run's re-exec-into-the-full-binary flow is unaffected by
// the targetPackage parameterization.
func TestNew_DefaultsToCliCmdTarget(t *testing.T) {
	t.Parallel()
	e := New("/interp/root", "/out/bal")
	if e.targetPackage != "cli/cmd" {
		t.Errorf("New's targetPackage = %q, want %q", e.targetPackage, "cli/cmd")
	}
}

// TestNewForTarget_SetsTargetPackage mirrors bal build's use: it must target
// cli/cmd/balrt (the slim stub) rather than the full CLI.
func TestNewForTarget_SetsTargetPackage(t *testing.T) {
	t.Parallel()
	e := NewForTarget("/interp/root", "/out/balrt", "cli/cmd/balrt")
	if e.targetPackage != "cli/cmd/balrt" {
		t.Errorf("NewForTarget's targetPackage = %q, want %q", e.targetPackage, "cli/cmd/balrt")
	}
}

// newFakeInterpreterRoot creates a temp directory with a minimal go.mod/go.sum,
// enough for localFingerprint to hash without a real interpreter checkout.
func newFakeInterpreterRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module ballerina-lang-go\n\ngo 1.26\n")
	mustWriteFile(t, filepath.Join(root, "go.sum"), "")
	return root
}

// TestLoadCachedBinary mirrors the real-world case of a repeated bal
// build/run on an unchanged native dependency (cache hit — skip the
// toolchain build entirely) versus a changed dependency, a corrupted
// fingerprint file, or a binary deleted out from under a still-valid
// fingerprint (all cache misses — must fall back to a real rebuild rather
// than serve a stale or missing binary).
func TestLoadCachedBinary(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	fingerprint, err := localFingerprint(interpRoot, payloads, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("localFingerprint: %v", err)
	}

	t.Run("hit", func(t *testing.T) {
		outBin := filepath.Join(t.TempDir(), "out-bin")
		mustWriteFile(t, outBin, "fake-binary-bytes")
		mustWriteFile(t, nativeexec.FingerprintPath(outBin), fingerprint)

		e := NewForTarget(interpRoot, outBin, "cli/cmd/balrt")
		got, ok := e.loadCachedBinary(fingerprint)
		if !ok {
			t.Fatalf("expected a cache hit")
		}
		if got != outBin {
			t.Errorf("loadCachedBinary path = %q, want %q", got, outBin)
		}
	})

	t.Run("miss: fingerprint mismatch", func(t *testing.T) {
		outBin := filepath.Join(t.TempDir(), "out-bin")
		mustWriteFile(t, outBin, "fake-binary-bytes")
		mustWriteFile(t, nativeexec.FingerprintPath(outBin), "stale-fingerprint")

		e := NewForTarget(interpRoot, outBin, "cli/cmd/balrt")
		if _, ok := e.loadCachedBinary(fingerprint); ok {
			t.Error("expected a cache miss for a mismatched fingerprint")
		}
	})

	t.Run("miss: no fingerprint file", func(t *testing.T) {
		outBin := filepath.Join(t.TempDir(), "out-bin")
		mustWriteFile(t, outBin, "fake-binary-bytes")

		e := NewForTarget(interpRoot, outBin, "cli/cmd/balrt")
		if _, ok := e.loadCachedBinary(fingerprint); ok {
			t.Error("expected a cache miss when no fingerprint file exists")
		}
	})

	t.Run("miss: binary deleted out from under a valid fingerprint", func(t *testing.T) {
		outBin := filepath.Join(t.TempDir(), "out-bin")
		mustWriteFile(t, nativeexec.FingerprintPath(outBin), fingerprint)
		// outBin itself is deliberately not created.

		e := NewForTarget(interpRoot, outBin, "cli/cmd/balrt")
		if _, ok := e.loadCachedBinary(fingerprint); ok {
			t.Error("expected a cache miss when the binary itself is missing")
		}
	})
}

// TestLocalFingerprint_DiffersByTargetPlatform mirrors cross-compiling the
// same native dependency for two different targets from the same project:
// each target must get its own fingerprint, so a build for one platform is
// never mistaken for a cache hit on another (which would serve a binary of
// the wrong architecture).
func TestLocalFingerprint_DiffersByTargetPlatform(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}

	linuxAmd64, err := localFingerprint(interpRoot, payloads, "linux", "amd64")
	if err != nil {
		t.Fatalf("localFingerprint(linux/amd64): %v", err)
	}
	darwinArm64, err := localFingerprint(interpRoot, payloads, "darwin", "arm64")
	if err != nil {
		t.Fatalf("localFingerprint(darwin/arm64): %v", err)
	}
	if linuxAmd64 == darwinArm64 {
		t.Error("expected different fingerprints for different target platforms, got the same one")
	}

	// Re-requesting the same target must be stable (deterministic), or the
	// cache would never hit even for an unchanged target.
	linuxAmd64Again, err := localFingerprint(interpRoot, payloads, "linux", "amd64")
	if err != nil {
		t.Fatalf("localFingerprint(linux/amd64) again: %v", err)
	}
	if linuxAmd64 != linuxAmd64Again {
		t.Error("expected the same target platform to produce a stable fingerprint")
	}
}

// TestCrossCompileEnv verifies the go build subprocess environment
// buildOrReuse constructs for a target platform: it must set exactly one
// GOOS/GOARCH/CGO_ENABLED each (not append duplicates on top of whatever the
// parent process's environment already had), so os/exec's undefined
// behavior for duplicate keys never comes into play.
func TestCrossCompileEnv(t *testing.T) {
	t.Parallel()
	env := crossCompileEnv("linux", "amd64")

	counts := map[string]int{}
	var goos, goarch, cgo string
	for _, e := range env {
		switch {
		case strings.HasPrefix(e, "GOOS="):
			counts["GOOS"]++
			goos = strings.TrimPrefix(e, "GOOS=")
		case strings.HasPrefix(e, "GOARCH="):
			counts["GOARCH"]++
			goarch = strings.TrimPrefix(e, "GOARCH=")
		case strings.HasPrefix(e, "CGO_ENABLED="):
			counts["CGO_ENABLED"]++
			cgo = strings.TrimPrefix(e, "CGO_ENABLED=")
		}
	}

	for _, key := range []string{"GOOS", "GOARCH", "CGO_ENABLED"} {
		if counts[key] != 1 {
			t.Errorf("expected exactly one %s entry in the build env, got %d", key, counts[key])
		}
	}
	if goos != "linux" {
		t.Errorf("GOOS = %q, want %q", goos, "linux")
	}
	if goarch != "amd64" {
		t.Errorf("GOARCH = %q, want %q", goarch, "amd64")
	}
	if cgo != "0" {
		t.Errorf("CGO_ENABLED = %q, want %q", cgo, "0")
	}
}

// TestBuild_CacheHit_ReturnsCachedPathNoToolchain and
// TestPrepare_CacheHit_EmptyTmpDir exercise Build/Prepare's fast path (a
// pre-seeded, matching fingerprint) without invoking the real Go toolchain —
// a genuine cold build is covered at the corpus level (too slow/
// integration-shaped for a unit test, matching this file's existing
// precedent for writeNativeFiles/writePatchedGoMod).
func TestBuild_CacheHit_ReturnsCachedPathNoToolchain(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	fingerprint, err := localFingerprint(interpRoot, payloads, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("localFingerprint: %v", err)
	}

	outBin := filepath.Join(t.TempDir(), "balrt-native")
	mustWriteFile(t, outBin, "fake-binary-bytes")
	mustWriteFile(t, nativeexec.FingerprintPath(outBin), fingerprint)

	e := NewForTarget(interpRoot, outBin, "cli/cmd/balrt")
	got, err := e.Build(context.Background(), nativeexec.NativeRunnerRequest{Payloads: payloads})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got != outBin {
		t.Errorf("Build path = %q, want %q", got, outBin)
	}
}

func TestPrepare_CacheHit_EmptyTmpDir(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	fingerprint, err := localFingerprint(interpRoot, payloads, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("localFingerprint: %v", err)
	}

	outBin := filepath.Join(t.TempDir(), "bal")
	mustWriteFile(t, outBin, "fake-binary-bytes")
	mustWriteFile(t, nativeexec.FingerprintPath(outBin), fingerprint)

	e := New(interpRoot, outBin)
	runner, err := e.Prepare(context.Background(), nativeexec.NativeRunnerRequest{Payloads: payloads})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	local, ok := runner.(*localRunner)
	if !ok {
		t.Fatalf("expected *localRunner, got %T", runner)
	}
	if local.binaryPath != outBin {
		t.Errorf("runner.binaryPath = %q, want %q", local.binaryPath, outBin)
	}
	// A cache hit built nothing, so there's no tmpDir to clean up on Close().
	if local.tmpDir != "" {
		t.Errorf("expected empty tmpDir on a cache hit, got %q", local.tmpDir)
	}
	if err := local.Close(); err != nil {
		t.Errorf("Close on a cache-hit runner should be a no-op, got: %v", err)
	}
}

// helpers

func checkFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("%s: got %q, want %q", path, string(data), want)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// TestBuild_CompileErrorInNativeSource covers a native Go dependency whose
// source fails to compile. Unlike this file's other tests, it uses the real
// repo as interpreterRoot (not newFakeInterpreterRoot) — a fake minimal
// go.mod can't compile cli/cmd/balrt at all, so exercising a genuine build
// failure needs a genuine buildable module. The failure must surface as a
// clear "building native interpreter" error, not a panic or a hang,
// regardless of whether it's triggered via bal run or bal build (both
// funnel through this same Build method).
func TestBuild_CompileErrorInNativeSource(t *testing.T) {
	t.Parallel()
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}

	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{
				"broken.go": {Data: []byte("package brokennative\n\nfunc broken( {\n")}, // syntax error
			},
			Module: "example.com/broken-native",
		},
	}

	outBin := filepath.Join(t.TempDir(), "balrt-native")
	executor := NewForTarget(repoRoot, outBin, "cli/cmd/balrt")
	if !executor.Available() {
		t.Skip("Go toolchain or interpreter source unavailable in this environment")
	}

	_, buildErr := executor.Build(context.Background(), nativeexec.NativeRunnerRequest{
		Payloads: payloads,
		Stderr:   io.Discard,
	})
	if buildErr == nil {
		t.Fatal("expected an error building a native package with invalid Go source, got none")
	}
	if !strings.Contains(buildErr.Error(), "building native interpreter") {
		t.Errorf("expected the error to mention 'building native interpreter', got: %v", buildErr)
	}
	if _, statErr := os.Stat(outBin); statErr == nil {
		t.Errorf("expected no output binary for a failed native build, found one at %s", outBin)
	}
}
