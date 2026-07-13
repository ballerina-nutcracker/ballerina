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

package executable

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/semtypes"
)

// compileMinimalPackage compiles a tiny, import-free Ballerina package and
// returns its BIR packages and type environment — real compiler output, not
// hand-built structs, so these tests exercise Pack/TryLoad against the same
// shape of data bal build actually produces.
func compileMinimalPackage(t *testing.T) ([]*bir.BIRPackage, semtypes.Env) {
	t.Helper()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "Ballerina.toml"), []byte(
		"[package]\norg = \"testorg\"\nname = \"stubtest\"\nversion = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatalf("writing Ballerina.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "main.bal"), []byte(
		"public function main() {\n}\n"), 0o644); err != nil {
		t.Fatalf("writing main.bal: %v", err)
	}

	result, err := projects.Load(os.DirFS(projectDir), ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("loading project: %v", err)
	}
	if diag := result.Diagnostics(); diag.HasErrors() {
		t.Fatalf("project loading reported errors: %v", diag)
	}

	pkg := result.Project().CurrentPackage()
	compilation := pkg.Compilation()
	if diag := compilation.DiagnosticResult(); diag.HasErrors() {
		t.Fatalf("compilation reported errors: %v", diag)
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		t.Fatalf("expected at least one BIR package")
	}
	return birPkgs, result.Project().Environment().TypeEnv()
}

// writeStub writes arbitrary content to serve as a "stub" for Pack — the
// tests only care about the trailer/payload framing Pack adds, not about
// having a real bal/balrt binary underneath.
func writeStub(t *testing.T, content string) string {
	t.Helper()
	stubPath := filepath.Join(t.TempDir(), "stub")
	if err := os.WriteFile(stubPath, []byte(content), 0o755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}
	return stubPath
}

// TestBuildAndRun mirrors a user's first-ever build of a new project: the
// output directory doesn't exist yet, and running bal build should produce a
// working executable in one step. Covers both the round-trip guarantee
// (Pack then read back the same program) and parent-directory creation.
func TestBuildAndRun(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")

	outPath := filepath.Join(t.TempDir(), "nested", "target", "bin", "myprogram")
	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("expected output at %s (parent dirs should have been created): %v", outPath, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected output to be executable, got mode %v", info.Mode())
	}

	gotPkgs, _, err := tryLoadFrom(outPath)
	if err != nil {
		t.Fatalf("tryLoadFrom: %v", err)
	}
	if len(gotPkgs) != len(birPkgs) {
		t.Fatalf("expected %d BIR packages back, got %d", len(birPkgs), len(gotPkgs))
	}
	for i, pkg := range gotPkgs {
		if pkg.PackageID.PkgName.Value() != birPkgs[i].PackageID.PkgName.Value() {
			t.Fatalf("package %d: expected name %q, got %q", i, birPkgs[i].PackageID.PkgName.Value(), pkg.PackageID.PkgName.Value())
		}
	}
}

// TestEditRebuildRun mirrors the everyday edit-build-run loop: bal build is
// run twice against the same output path (a user editing code and
// rebuilding). Regression test for the O_TRUNC-reuses-inode case, where the
// second build could silently lose its executable bit.
func TestEditRebuildRun(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")
	outPath := filepath.Join(t.TempDir(), "target", "bin", "myprogram")

	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("first Pack: %v", err)
	}
	if err := os.Chmod(outPath, 0o644); err != nil { // simulate a non-executable pre-existing file at that inode
		t.Fatalf("chmod: %v", err)
	}
	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("second Pack (rebuild): %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat after rebuild: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected output to still be executable after rebuild, got mode %v", info.Mode())
	}
}

// TestCorruptedCopy mirrors a build artifact that got mangled in transit
// (e.g. a text-mode file transfer altering binary content) — the trailer's
// magic marker no longer matches. This must be treated as "not a compiled
// program" rather than an error, same as any other plain binary.
func TestCorruptedCopy(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")
	outPath := filepath.Join(t.TempDir(), "program")
	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading packed file: %v", err)
	}
	// Mangle the magic (last 8 bytes of the trailer) without touching the offset.
	for i := len(data) - 8; i < len(data); i++ {
		data[i] ^= 0xFF
	}
	if err := os.WriteFile(outPath, data, 0o755); err != nil {
		t.Fatalf("writing mangled file: %v", err)
	}

	pkgs, tyEnvOut, err := tryLoadFrom(outPath)
	if err != nil {
		t.Fatalf("expected no error for a mangled magic marker, got: %v", err)
	}
	if pkgs != nil || tyEnvOut != nil {
		t.Fatalf("expected (nil, nil) for a mangled magic marker, got pkgs=%v tyEnv=%v", pkgs, tyEnvOut)
	}
}

// TestCorruptedOffset mirrors a build artifact whose own bytes got corrupted
// (a flaky disk or storage error hitting the trailer specifically) so the
// payload offset it records is invalid. Must fail with a clear error, not
// crash or attempt a huge/negative allocation.
func TestCorruptedOffset(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")
	outPath := filepath.Join(t.TempDir(), "program")
	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading packed file: %v", err)
	}
	// Overwrite the offset (first 8 bytes of the trailer) with a value far
	// beyond the file's size, leaving the magic intact so TryLoad proceeds
	// past the "is this a compiled program" check into the offset guard.
	binary.LittleEndian.PutUint64(data[len(data)-trailerSize:], ^uint64(0))
	if err := os.WriteFile(outPath, data, 0o755); err != nil {
		t.Fatalf("writing corrupted file: %v", err)
	}

	pkgs, tyEnvOut, err := tryLoadFrom(outPath)
	if err == nil {
		t.Fatalf("expected an error for a corrupted offset, got pkgs=%v tyEnv=%v", pkgs, tyEnvOut)
	}
}

// TestInterruptedTransfer mirrors a file transfer that was cut off partway
// (e.g. a dropped scp connection uploading to a deployment target): the
// trailer is intact and points at a real offset, but the payload bytes
// before it are shorter than what the payload's own internal structure
// declares. Must fail with a clear "truncated" error, not misinterpret
// leftover/partial bytes as a different, corrupt program.
func TestInterruptedTransfer(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")
	outPath := filepath.Join(t.TempDir(), "program")
	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading packed file: %v", err)
	}

	trailer := data[len(data)-trailerSize:]
	payloadOffset := int64(binary.LittleEndian.Uint64(trailer[:8]))
	payloadEnd := len(data) - trailerSize
	if payloadEnd-int(payloadOffset) < 8 {
		t.Fatalf("payload too small to truncate meaningfully in this test")
	}
	// Drop the last few bytes of the payload (simulating a transfer cut off
	// partway) while keeping the offset and magic intact, so tryLoadFrom
	// gets past the outer checks and into unmarshalPayload's own truncation
	// detection.
	truncatedPayloadEnd := payloadEnd - 4
	var truncated []byte
	truncated = append(truncated, data[:truncatedPayloadEnd]...)
	newTrailer := make([]byte, trailerSize)
	binary.LittleEndian.PutUint64(newTrailer[:8], uint64(payloadOffset))
	copy(newTrailer[8:], magic)
	truncated = append(truncated, newTrailer...)

	if err := os.WriteFile(outPath, truncated, 0o755); err != nil {
		t.Fatalf("writing truncated file: %v", err)
	}

	pkgs, tyEnvOut, err := tryLoadFrom(outPath)
	if err == nil {
		t.Fatalf("expected a truncation error, got pkgs=%v tyEnv=%v", pkgs, tyEnvOut)
	}
}

// TestNativeDependencyRejected mirrors the current, real behavior for any
// package with native Go dependencies (once that detection is wired into
// bal build) — ResolveStub must reject a non-empty Fingerprint with a clear
// error rather than mishandling it, since the toolchain-based build path
// isn't implemented yet.
func TestNativeDependencyRejected(t *testing.T) {
	_, err := ResolveStub(Key{Fingerprint: "deadbeef"}, t.TempDir(), "dev", "")
	if err == nil {
		t.Fatalf("expected an error for a non-empty Fingerprint, got none")
	}
}

// TestMissingStub mirrors the stub file having been deleted between
// ResolveStub resolving its path and Pack reading it (a real race — e.g. a
// concurrent process cleaning up, or a caller simply passing a stale path).
func TestMissingStub(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	missingStub := filepath.Join(t.TempDir(), "does-not-exist")
	outPath := filepath.Join(t.TempDir(), "program")

	err := Pack(missingStub, birPkgs, tyEnv, outPath)
	if err == nil {
		t.Fatalf("expected an error for a missing stub, got none")
	}
}

// TestOutputParentIsAFile mirrors a naming collision: a regular file
// already sits where Pack needs to create a parent directory (e.g. stale
// state from an earlier, different build, or a typo in the output path).
func TestOutputParentIsAFile(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")

	blockingFile := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(blockingFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("writing blocking file: %v", err)
	}
	outPath := filepath.Join(blockingFile, "bin", "program")

	err := Pack(stubPath, birPkgs, tyEnv, outPath)
	if err == nil {
		t.Fatalf("expected an error when a parent path component is a file, got none")
	}
}

// TestOutputPathIsADirectory mirrors leftover state from a previous failed
// build (or a typo) leaving a directory at the exact path bal build needs
// to write its output file to.
func TestOutputPathIsADirectory(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")

	outPath := filepath.Join(t.TempDir(), "program")
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		t.Fatalf("creating directory at output path: %v", err)
	}

	err := Pack(stubPath, birPkgs, tyEnv, outPath)
	if err == nil {
		t.Fatalf("expected an error when the output path is already a directory, got none")
	}
}
