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
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ballerina-nutcracker/ballerina/cli/internal/nativeexec"
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
		{"1.26", "1.26rc1", false},
	}
	for _, tc := range cases {
		got := versionAtLeast(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("versionAtLeast(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestGoVersionAtLeast_ExecFails covers goExe not being runnable at all
// (e.g. a stale cached path to a binary that's since been removed).
func TestGoVersionAtLeast_ExecFails(t *testing.T) {
	t.Parallel()
	if goVersionAtLeast(filepath.Join(t.TempDir(), "no-such-go-binary"), "1.26") {
		t.Error("expected false when the go binary can't be executed")
	}
}

// writeFakeGoStub writes a fake "go" executable into dir that prints output
// (unquoted) when run as `<stub> version`, on both POSIX shells and Windows
// cmd.exe — name matches what exec.LookPath("go") resolves on each OS
// (PATHEXT includes .bat on Windows), so it also works when found via PATH.
func writeFakeGoStub(t *testing.T, dir, output string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "go.bat")
		mustWriteFile(t, path, "@echo off\r\necho "+output+"\r\n")
		return path
	}
	path := filepath.Join(dir, "go")
	mustWriteFile(t, path, "#!/bin/sh\necho "+output+"\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod fake go stub: %v", err)
	}
	return path
}

// TestGoVersionAtLeast_MalformedOutput covers a `go version` invocation
// that runs but doesn't print the expected "go version goX.Y os/arch" shape.
func TestGoVersionAtLeast_MalformedOutput(t *testing.T) {
	t.Parallel()
	fakeGo := writeFakeGoStub(t, t.TempDir(), "garbage")
	if goVersionAtLeast(fakeGo, "1.26") {
		t.Error("expected false for malformed `go version` output")
	}
}

// TestAvailable_NoGoOnPath covers the environment having no usable go
// toolchain at all — Available must report false, not error.
func TestAvailable_NoGoOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	e := New("/interp/root", "/out/bal")
	if e.Available() {
		t.Error("expected Available() to be false when go is not on PATH")
	}
}

// TestAvailable_GoTooOld covers a go on PATH that's older than
// MinGoVersion — Available must report false, distinct from go being
// entirely absent.
func TestAvailable_GoTooOld(t *testing.T) {
	dir := t.TempDir()
	writeFakeGoStub(t, dir, "go version go1.20.0 linux/amd64")
	t.Setenv("PATH", dir)

	e := New("/interp/root", "/out/bal")
	if e.Available() {
		t.Error("expected Available() to be false when go is older than MinGoVersion")
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

func TestWriteNativeWorkspace(t *testing.T) {
	t.Parallel()
	interpRoot := t.TempDir()
	for _, dir := range interpreterModuleDirs {
		mustWriteFile(t, filepath.Join(interpRoot, dir, "go.mod"), "module example.com/interpreter\n\ngo 1.26\n")
	}

	tmpDir := t.TempDir()
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{Module: "example.com/pkgA"},
		&nativeexec.GoSourcePayload{Module: "example.com/pkgB"},
	}
	for _, payload := range payloads {
		mustWriteFile(t, filepath.Join(tmpDir, moduleDirName(payload.GoModuleName()), "go.mod"),
			"module "+payload.GoModuleName()+"\n\ngo 1.26\n")
	}

	workspaceFile, err := writeNativeWorkspace(tmpDir, interpRoot, payloads)
	if err != nil {
		t.Fatalf("writeNativeWorkspace: %v", err)
	}
	content := mustReadFile(t, workspaceFile)
	for _, dir := range interpreterModuleDirs {
		if !strings.Contains(content, strconv.Quote(filepath.Join(interpRoot, dir))) {
			t.Errorf("workspace missing interpreter module %q:\n%s", dir, content)
		}
	}
	for _, payload := range payloads {
		payloadDir := strconv.Quote(filepath.Join(tmpDir, moduleDirName(payload.GoModuleName())))
		if !strings.Contains(content, payloadDir) {
			t.Errorf("workspace missing payload %s:\n%s", payload.GoModuleName(), content)
		}
	}
	if !strings.Contains(content, "github.com/ballerina-nutcracker/ballerina/ast v0.6.0 =>") {
		t.Errorf("workspace missing versioned interpreter replacement:\n%s", content)
	}
}

func TestWriteNativeWorkspace_MissingModule(t *testing.T) {
	t.Parallel()
	_, err := writeNativeWorkspace(t.TempDir(), "/nonexistent/root", nil)
	if err == nil {
		t.Error("expected error when an interpreter module is missing")
	}
}

// TestNew_DefaultsToCliCmdTarget checks New targets the full CLI (cli/cmd),
// matching bal run's re-exec use case.
func TestNew_DefaultsToCliCmdTarget(t *testing.T) {
	t.Parallel()
	e := New("/interp/root", "/out/bal")
	if e.targetPackage != "cli/cmd" {
		t.Errorf("New's targetPackage = %q, want %q", e.targetPackage, "cli/cmd")
	}
}

// TestNewForTarget_SetsTargetPackage checks it targets the slim stub
// (cli/internal/balrt) rather than the full CLI, matching bal build's use.
func TestNewForTarget_SetsTargetPackage(t *testing.T) {
	t.Parallel()
	e := NewForTarget("/interp/root", "/out/balrt", "cli/internal/balrt")
	if e.targetPackage != "cli/internal/balrt" {
		t.Errorf("NewForTarget's targetPackage = %q, want %q", e.targetPackage, "cli/internal/balrt")
	}
}

// newFakeInterpreterRoot creates minimal workspace manifests, enough for
// localFingerprint without a real interpreter checkout.
func newFakeInterpreterRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.work"), "go 1.26\n")
	for _, dir := range interpreterModuleDirs {
		mustWriteFile(t, filepath.Join(root, dir, "go.mod"), "module example.com/interpreter\n\ngo 1.26\n")
	}
	mustWriteFile(t, filepath.Join(root, "go.sum"), "")
	return root
}

// TestLocalFingerprint_DiffersByTargetPackage covers two identical builds
// (same interpreter root, payloads, Go version, platform) that differ only
// in targetPackage — e.g. bal run's "cli/cmd" vs bal build's
// "cli/internal/balrt" — which must produce different fingerprints, so a cache
// hit can never reuse a binary built for the wrong target package.
func TestLocalFingerprint_DiffersByTargetPackage(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}

	fpA, err := localFingerprint(interpRoot, "cli/cmd", payloads, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("localFingerprint (cli/cmd): %v", err)
	}
	fpB, err := localFingerprint(interpRoot, "cli/internal/balrt", payloads, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("localFingerprint (cli/internal/balrt): %v", err)
	}
	if fpA == fpB {
		t.Fatal("expected different fingerprints for different target packages, got the same one")
	}
}

// TestLoadCachedBinary covers a cache hit (unchanged dependency) versus
// misses (changed dependency, corrupted fingerprint, or a binary deleted
// out from under a still-valid fingerprint) — misses must fall back to a
// rebuild, not serve stale/missing output.
func TestLoadCachedBinary(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	fingerprint, err := localFingerprint(interpRoot, "cli/internal/balrt", payloads, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("localFingerprint: %v", err)
	}

	t.Run("hit", func(t *testing.T) {
		outBin := filepath.Join(t.TempDir(), "out-bin")
		mustWriteFile(t, outBin, "fake-binary-bytes")
		mustWriteFile(t, nativeexec.FingerprintPath(outBin), fingerprint)

		e := NewForTarget(interpRoot, outBin, "cli/internal/balrt")
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

		e := NewForTarget(interpRoot, outBin, "cli/internal/balrt")
		if _, ok := e.loadCachedBinary(fingerprint); ok {
			t.Error("expected a cache miss for a mismatched fingerprint")
		}
	})

	t.Run("miss: no fingerprint file", func(t *testing.T) {
		outBin := filepath.Join(t.TempDir(), "out-bin")
		mustWriteFile(t, outBin, "fake-binary-bytes")

		e := NewForTarget(interpRoot, outBin, "cli/internal/balrt")
		if _, ok := e.loadCachedBinary(fingerprint); ok {
			t.Error("expected a cache miss when no fingerprint file exists")
		}
	})

	t.Run("miss: binary deleted out from under a valid fingerprint", func(t *testing.T) {
		outBin := filepath.Join(t.TempDir(), "out-bin")
		mustWriteFile(t, nativeexec.FingerprintPath(outBin), fingerprint)
		// outBin itself is deliberately not created.

		e := NewForTarget(interpRoot, outBin, "cli/internal/balrt")
		if _, ok := e.loadCachedBinary(fingerprint); ok {
			t.Error("expected a cache miss when the binary itself is missing")
		}
	})
}

// Fingerprint-differs-by-target-platform coverage (different targets don't
// collide, same target is stable/cached) moved to a corpus-level test
// against the real bal build CLI: TestBalBuildNativeDependencyCacheByTarget
// (corpus/cli_integration_test.go).

// TestCrossCompileEnv checks the build env has exactly one
// GOOS/GOARCH/CGO_ENABLED each, not duplicates on top of the parent env.
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
// TestPrepare_CacheHit_EmptyTmpDir exercise the cache-hit fast path without
// invoking the real toolchain; a cold build is covered at the corpus level.
func TestBuild_CacheHit_ReturnsCachedPathNoToolchain(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	fingerprint, err := localFingerprint(interpRoot, "cli/internal/balrt", payloads, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("localFingerprint: %v", err)
	}

	outBin := filepath.Join(t.TempDir(), "balrt-native")
	mustWriteFile(t, outBin, "fake-binary-bytes")
	mustWriteFile(t, nativeexec.FingerprintPath(outBin), fingerprint)

	e := NewForTarget(interpRoot, outBin, "cli/internal/balrt")
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
	fingerprint, err := localFingerprint(interpRoot, "cli/cmd", payloads, runtime.GOOS, runtime.GOARCH)
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

// TestBuildOrReuse_TempDirCreationFails covers os.MkdirTemp itself failing
// (e.g. TMPDIR pointing at a read-only or missing directory) — must surface
// a clear error rather than a nil-dir panic downstream.
func TestBuildOrReuse_TempDirCreationFails(t *testing.T) {
	interpRoot := newFakeInterpreterRoot(t)
	blockedTMPDIR := filepath.Join(t.TempDir(), "no-such-tmpdir")
	t.Setenv("TMPDIR", blockedTMPDIR) // consulted by os.TempDir on unix
	t.Setenv("TMP", blockedTMPDIR)    // consulted by os.TempDir on windows

	e := NewForTarget(interpRoot, filepath.Join(t.TempDir(), "out"), "cli/internal/balrt")
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	_, err := e.Build(context.Background(), nativeexec.NativeRunnerRequest{Payloads: payloads})
	if err == nil {
		t.Fatal("expected an error when the temp bundle dir can't be created")
	}
	if !strings.Contains(err.Error(), "creating temp bundle dir") {
		t.Errorf("expected a 'creating temp bundle dir' error, got: %v", err)
	}
}

// TestBuildOrReuse_OutputDirBlocked covers the output binary's parent
// directory being unable to be created (a regular file sits where a
// directory component is needed).
func TestBuildOrReuse_OutputDirBlocked(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	blockingFile := filepath.Join(t.TempDir(), "blocker")
	mustWriteFile(t, blockingFile, "not a directory")
	outBin := filepath.Join(blockingFile, "bin", "balrt-native")

	e := NewForTarget(interpRoot, outBin, "cli/internal/balrt")
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	_, err := e.Build(context.Background(), nativeexec.NativeRunnerRequest{Payloads: payloads})
	if err == nil {
		t.Fatal("expected an error when the output directory can't be created")
	}
	if !strings.Contains(err.Error(), "creating output directory") {
		t.Errorf("expected a 'creating output directory' error, got: %v", err)
	}
}

// errWriter always fails, simulating a broken Stderr sink.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// TestBuildOrReuse_WriteBuildStatusFails covers the "info: building native
// interpreter" status line itself failing to write — must surface as a
// clear error and bail out before ever invoking the go toolchain.
func TestBuildOrReuse_WriteBuildStatusFails(t *testing.T) {
	t.Parallel()
	interpRoot := newFakeInterpreterRoot(t)
	e := NewForTarget(interpRoot, filepath.Join(t.TempDir(), "out"), "cli/internal/balrt")
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{
			GoFiles: fstest.MapFS{"pkg.go": {Data: []byte("package pkg\n")}},
			Module:  "example.com/pkg",
		},
	}
	_, err := e.Build(context.Background(), nativeexec.NativeRunnerRequest{
		Payloads: payloads,
		Stderr:   errWriter{},
	})
	if err == nil {
		t.Fatal("expected an error when writing the build status line fails")
	}
	if !strings.Contains(err.Error(), "writing build status") {
		t.Errorf("expected a 'writing build status' error, got: %v", err)
	}
}

// TestLocalRunner_Run_InvalidBinaryPath covers the non-ExitError failure
// path: a binaryPath that can't be executed at all (not merely a nonzero
// exit) must surface a wrapped "executing native interpreter" error.
func TestLocalRunner_Run_InvalidBinaryPath(t *testing.T) {
	t.Parallel()
	r := &localRunner{binaryPath: filepath.Join(t.TempDir(), "does-not-exist")}
	_, err := r.Run(context.Background())
	if err == nil {
		t.Fatal("expected an error for a nonexistent binary path")
	}
	if !strings.Contains(err.Error(), "executing native interpreter") {
		t.Errorf("expected an 'executing native interpreter' error, got: %v", err)
	}
}

// TestLocalRunner_Close_RemoveFails covers cleanup itself failing (e.g. a
// file inside tmpDir left without write permission on its parent) — Close
// must propagate that error rather than silently swallowing it.
func TestLocalRunner_Close_RemoveFails(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits don't apply on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores POSIX permission bits, so removal wouldn't actually fail")
	}
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "leftover"), "data")
	if err := os.Chmod(tmpDir, 0o500); err != nil {
		t.Fatalf("chmod tmpDir: %v", err)
	}
	defer func() { _ = os.Chmod(tmpDir, 0o700) }() // let t.TempDir() clean up

	r := &localRunner{tmpDir: tmpDir}
	if err := r.Close(); err == nil {
		t.Error("expected Close to propagate the RemoveAll failure")
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating parent directory for %s: %v", path, err)
	}
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

// TestBuild_CompileErrorInNativeSource covers a native Go dependency that
// fails to compile — must surface as a clear "building native interpreter"
// error, not a panic or hang. Uses the real repo as interpreterRoot (a fake
// minimal go.mod can't compile cli/internal/balrt).
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
	executor := NewForTarget(repoRoot, outBin, "cli/internal/balrt")
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
