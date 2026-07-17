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
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/semtypes"
)

// compileMinimalPackage compiles a tiny package and returns its BIR
// packages and type env — real compiler output, not hand-built structs.
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

// writeStub writes arbitrary content as a "stub" for Pack — tests only
// care about the trailer/payload framing, not a real bal/balrt binary.
func writeStub(t *testing.T, content string) string {
	t.Helper()
	stubPath := filepath.Join(t.TempDir(), "stub")
	if err := os.WriteFile(stubPath, []byte(content), 0o755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}
	return stubPath
}

// TestBuildAndRun covers a first build into a nonexistent output directory:
// Pack must create parent dirs, and the packed program must read back intact.
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
	// Windows has no POSIX execute bit — os.FileMode there only ever reflects
	// the read-only attribute (0666/0444), regardless of what Chmod requested.
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
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

// TestEditRebuildRun packs the same output path twice (edit-rebuild).
// Regression test: O_TRUNC reuses the inode, so a rebuild could silently
// lose its executable bit.
func TestEditRebuildRun(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")
	outPath := filepath.Join(t.TempDir(), "target", "bin", "myprogram")

	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("first Pack: %v", err)
	}
	if err := os.Chmod(outPath, 0o644); err != nil { // simulate a stale non-executable file at that inode
		t.Fatalf("chmod: %v", err)
	}
	if err := Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("second Pack (rebuild): %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat after rebuild: %v", err)
	}
	// See TestBuildAndRun: Windows has no POSIX execute bit to check.
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("expected output to still be executable after rebuild, got mode %v", info.Mode())
	}
}

// TestCorruptedCopy covers a mangled trailer magic marker (e.g. a
// text-mode transfer) — must be treated as "not a compiled program", not an error.
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

// TestCorruptedOffset covers a trailer whose recorded payload offset is
// invalid — must fail cleanly, not crash or attempt a huge allocation.
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
	// Offset far beyond the file size, magic left intact, so tryLoadFrom
	// reaches the offset guard rather than bailing out early.
	binary.LittleEndian.PutUint64(data[len(data)-trailerSize:], ^uint64(0))
	if err := os.WriteFile(outPath, data, 0o755); err != nil {
		t.Fatalf("writing corrupted file: %v", err)
	}

	pkgs, tyEnvOut, err := tryLoadFrom(outPath)
	if err == nil {
		t.Fatalf("expected an error for a corrupted offset, got pkgs=%v tyEnv=%v", pkgs, tyEnvOut)
	}
}

// TestInterruptedTransfer covers a truncated payload with an intact
// trailer — must fail with a clear "truncated" error, not misread
// leftover bytes as a different program.
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
	// Drop the last few payload bytes; keep offset and magic intact so
	// tryLoadFrom reaches unmarshalPayload's own truncation check.
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

// TestMarshalPayload_RejectsEmptyPackages covers marshaling zero BIR
// packages — Run would initialize nothing, so this must fail up front.
func TestMarshalPayload_RejectsEmptyPackages(t *testing.T) {
	_, tyEnv := compileMinimalPackage(t)
	if _, err := marshalPayload(nil, tyEnv); err == nil {
		t.Fatal("expected an error when marshaling zero BIR packages")
	}
}

// TestUnmarshalPayload_RejectsZeroCount covers a payload whose header
// declares zero packages — must fail rather than reach Run with nothing
// initialized.
func TestUnmarshalPayload_RejectsZeroCount(t *testing.T) {
	payload := make([]byte, 4) // count = 0
	if _, _, err := unmarshalPayload(payload); err == nil {
		t.Fatal("expected an error for a payload declaring zero packages")
	}
}

// TestUnmarshalPayload_RejectsTrailingBytes covers a payload with extra
// bytes after the last declared package — must not be silently accepted
// as valid framing.
func TestUnmarshalPayload_RejectsTrailingBytes(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	payload, err := marshalPayload(birPkgs, tyEnv)
	if err != nil {
		t.Fatalf("marshalPayload: %v", err)
	}
	payload = append(payload, 0xFF, 0xFF, 0xFF)

	if _, _, err := unmarshalPayload(payload); err == nil {
		t.Fatal("expected an error for a payload with unconsumed trailing bytes")
	}
}

// TestMissingStub covers the stub file being gone by the time Pack reads
// it (e.g. a stale path, or a concurrent cleanup).
func TestMissingStub(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	missingStub := filepath.Join(t.TempDir(), "does-not-exist")
	outPath := filepath.Join(t.TempDir(), "program")

	err := Pack(missingStub, birPkgs, tyEnv, outPath)
	if err == nil {
		t.Fatalf("expected an error for a missing stub, got none")
	}
}

// TestOutputParentIsAFile covers a regular file sitting where Pack needs
// to create a parent directory.
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

// TestOutputPathIsADirectory covers a directory already existing at the
// exact path Pack needs to write its output file to.
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

// TestPackFailureLeavesExistingExecutableIntact covers Pack's atomic-write
// guarantee: a failure partway through (here, a directory passed as the
// stub path, which fails io.Copy after the temp file already exists) must
// leave a previously-packed outPath byte-for-byte untouched, with no
// leftover temp file — not a truncated or corrupted executable.
func TestPackFailureLeavesExistingExecutableIntact(t *testing.T) {
	birPkgs, tyEnv := compileMinimalPackage(t)
	validStub := writeStub(t, "valid-stub-bytes")
	outPath := filepath.Join(t.TempDir(), "program")

	if err := Pack(validStub, birPkgs, tyEnv, outPath); err != nil {
		t.Fatalf("first Pack: %v", err)
	}
	before, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading packed file: %v", err)
	}

	dirAsStub := t.TempDir()
	if err := Pack(dirAsStub, birPkgs, tyEnv, outPath); err == nil {
		t.Fatal("expected Pack to fail when stubPath is a directory")
	}

	after, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading packed file after failed re-pack: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("expected outPath to be untouched after a failed Pack, but its contents changed")
	}

	entries, err := os.ReadDir(filepath.Dir(outPath))
	if err != nil {
		t.Fatalf("reading output dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(outPath) {
			t.Errorf("expected no leftover temp file, found %q", e.Name())
		}
	}
}

// TestPack_RejectsEmptyPackages covers Pack forwarding marshalPayload's
// own "no BIR packages to embed" error rather than writing a useless stub.
func TestPack_RejectsEmptyPackages(t *testing.T) {
	t.Parallel()
	_, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")
	outPath := filepath.Join(t.TempDir(), "program")

	if err := Pack(stubPath, nil, tyEnv, outPath); err == nil {
		t.Fatal("expected an error when packing zero BIR packages")
	}
	if _, err := os.Stat(outPath); err == nil {
		t.Error("expected no output file when packing fails before any write")
	}
}

// TestPack_CreateTempFails covers the output directory existing but not
// being writable — os.CreateTemp must fail cleanly rather than partially
// writing over a previous executable.
func TestPack_CreateTempFails(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits don't apply on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores POSIX permission bits, so the directory wouldn't actually be unwritable")
	}
	birPkgs, tyEnv := compileMinimalPackage(t)
	stubPath := writeStub(t, "stub-bytes")

	outDir := t.TempDir()
	if err := os.Chmod(outDir, 0o500); err != nil {
		t.Fatalf("chmod outDir: %v", err)
	}
	defer func() { _ = os.Chmod(outDir, 0o700) }() // let t.TempDir() clean up
	outPath := filepath.Join(outDir, "program")

	err := Pack(stubPath, birPkgs, tyEnv, outPath)
	if err == nil {
		t.Fatal("expected an error when the output directory isn't writable")
	}
	if !strings.Contains(err.Error(), "creating temp output file") {
		t.Errorf("expected a 'creating temp output file' error, got: %v", err)
	}
}

// errWriter always fails, simulating a broken output sink partway through
// writePackedFile's stub/payload/trailer sequence.
type errWriter struct {
	failAfter int // number of successful Write calls before failing
	calls     int
}

func (w *errWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls > w.failAfter {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

// TestWritePackedFile_PayloadWriteFails and TestWritePackedFile_TrailerWriteFails
// cover the two writes after the stub copy, each requiring the prior
// write(s) to succeed first.
func TestWritePackedFile_PayloadWriteFails(t *testing.T) {
	t.Parallel()
	w := &errWriter{failAfter: 1} // stub copy succeeds, payload write fails
	err := writePackedFile(w, strings.NewReader("stub-bytes"), 10, []byte("payload"))
	if err == nil {
		t.Fatal("expected an error when the payload write fails")
	}
	if !strings.Contains(err.Error(), "writing BIR payload") {
		t.Errorf("expected a 'writing BIR payload' error, got: %v", err)
	}
}

func TestWritePackedFile_TrailerWriteFails(t *testing.T) {
	t.Parallel()
	w := &errWriter{failAfter: 2} // stub copy and payload write succeed, trailer fails
	err := writePackedFile(w, strings.NewReader("stub-bytes"), 10, []byte("payload"))
	if err == nil {
		t.Fatal("expected an error when the trailer write fails")
	}
	if !strings.Contains(err.Error(), "writing trailer") {
		t.Errorf("expected a 'writing trailer' error, got: %v", err)
	}
}

// TestTryLoadFrom_NonexistentPath covers the exe path itself being
// unreadable (e.g. deleted or permission-denied) — must be treated as "not
// a compiled program", not surfaced as an error.
func TestTryLoadFrom_NonexistentPath(t *testing.T) {
	t.Parallel()
	pkgs, tyEnv, err := tryLoadFrom(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected no error for an unreadable path, got: %v", err)
	}
	if pkgs != nil || tyEnv != nil {
		t.Fatalf("expected (nil, nil) for an unreadable path, got pkgs=%v tyEnv=%v", pkgs, tyEnv)
	}
}

// TestTryLoadFrom_FileTooSmall covers a file too small to even contain a
// trailer (e.g. a stub with no embedded program at all) — must be treated
// as "not a compiled program", not an error.
func TestTryLoadFrom_FileTooSmall(t *testing.T) {
	t.Parallel()
	path := writeStub(t, "tiny")
	pkgs, tyEnv, err := tryLoadFrom(path)
	if err != nil {
		t.Fatalf("expected no error for a too-small file, got: %v", err)
	}
	if pkgs != nil || tyEnv != nil {
		t.Fatalf("expected (nil, nil) for a too-small file, got pkgs=%v tyEnv=%v", pkgs, tyEnv)
	}
}

// TestCorruptedZeroLengthPayload covers a trailer offset that passes the
// out-of-range guard exactly (pointing right at the trailer itself), so the
// derived payload size is zero — distinct from TestCorruptedOffset, which
// covers an offset that fails that guard outright.
func TestCorruptedZeroLengthPayload(t *testing.T) {
	t.Parallel()
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
	// Offset == size-trailerSize passes the range guard but yields a
	// derived payload size of exactly zero.
	zeroOffset := uint64(len(data) - trailerSize)
	binary.LittleEndian.PutUint64(data[len(data)-trailerSize:], zeroOffset)
	if err := os.WriteFile(outPath, data, 0o755); err != nil {
		t.Fatalf("writing corrupted file: %v", err)
	}

	pkgs, tyEnvOut, err := tryLoadFrom(outPath)
	if err == nil {
		t.Fatalf("expected an error for a zero-length payload, got pkgs=%v tyEnv=%v", pkgs, tyEnvOut)
	}
}

// TestUnmarshalPayload_RejectsTooShortPayload covers a payload too short to
// even contain the 4-byte package count.
func TestUnmarshalPayload_RejectsTooShortPayload(t *testing.T) {
	t.Parallel()
	if _, _, err := unmarshalPayload([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected an error for a payload shorter than the count header")
	}
}

// TestUnmarshalPayload_RejectsCountExceedingSize covers a declared package
// count too large to possibly fit in the remaining bytes, distinct from
// TestUnmarshalPayload_RejectsTrailingBytes (which under-declares).
func TestUnmarshalPayload_RejectsCountExceedingSize(t *testing.T) {
	t.Parallel()
	payload := make([]byte, 8) // count header + 4 bytes, nowhere near enough for count=1000
	binary.BigEndian.PutUint32(payload[:4], 1000)
	if _, _, err := unmarshalPayload(payload); err == nil {
		t.Fatal("expected an error for a package count too large for the payload size")
	}
}

// TestResolveTargetPlatform covers --target-os/--target-arch defaulting:
// either flag alone defaults the other to the host's value, like GOOS/GOARCH.
func TestResolveTargetPlatform(t *testing.T) {
	host := HostPlatform()

	tests := []struct {
		name       string
		targetOS   string
		targetArch string
		want       Platform
	}{
		{name: "both empty defaults to host", want: host},
		{name: "only OS given defaults arch to host", targetOS: "linux", want: Platform{OS: "linux", Arch: host.Arch}},
		{name: "only arch given defaults OS to host", targetArch: "arm64", want: Platform{OS: host.OS, Arch: "arm64"}},
		{name: "both given, no defaulting", targetOS: "windows", targetArch: "amd64", want: Platform{OS: "windows", Arch: "amd64"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTargetPlatform(tt.targetOS, tt.targetArch)
			if got != tt.want {
				t.Fatalf("ResolveTargetPlatform(%q, %q) = %+v, want %+v", tt.targetOS, tt.targetArch, got, tt.want)
			}
		})
	}
}

// Non-native cross-compile ResolveStub coverage (unsupported-platform
// rejection, correct-platform-among-several selection, Windows .exe suffix)
// moved to corpus-level tests against the real bal build CLI:
// TestBalBuildUnsupportedTargetPlatform and TestBalBuildCrossCompile
// (corpus/cli_integration_test.go).
