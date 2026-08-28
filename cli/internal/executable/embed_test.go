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
	"runtime"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/cli/internal/splice"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
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

// TestMarshalPayload_RejectsEmptyPackages covers marshaling zero BIR
// packages — Run would initialize nothing, so this must fail up front.
func TestMarshalPayload_RejectsEmptyPackages(t *testing.T) {
	_, tyEnv := compileMinimalPackage(t)
	if _, err := MarshalPayload(nil, tyEnv); err == nil {
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
	payload, err := MarshalPayload(birPkgs, tyEnv)
	if err != nil {
		t.Fatalf("MarshalPayload: %v", err)
	}
	payload = append(payload, 0xFF, 0xFF, 0xFF)

	if _, _, err := unmarshalPayload(payload); err == nil {
		t.Fatal("expected an error for a payload with unconsumed trailing bytes")
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

// TestTryLoadFrom_LinuxELFSection covers embed_linux.go's tryLoadFrom
// finding a real splice.EmbedELF-embedded payload section. Gated to linux since
// tryLoadFrom is a build-tagged function — on any other host, this
// package doesn't even compile embed_linux.go's implementation in.
func TestTryLoadFrom_LinuxELFSection(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("exercises the linux-only ELF-section reader")
	}

	birPkgs, tyEnv := compileMinimalPackage(t)
	payload, err := MarshalPayload(birPkgs, tyEnv)
	if err != nil {
		t.Fatalf("MarshalPayload: %v", err)
	}

	// The running test binary is itself a real linux ELF64 executable —
	// reused as the splice target so this test needs no cross-compile step.
	selfPath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "packed")
	if err := splice.EmbedELF(selfPath, payload, outPath); err != nil {
		t.Fatalf("splice.EmbedELF: %v", err)
	}

	gotPkgs, _, err := tryLoadFrom(outPath)
	if err != nil {
		t.Fatalf("tryLoadFrom: %v", err)
	}
	if len(gotPkgs) != len(birPkgs) {
		t.Fatalf("expected %d BIR packages back, got %d", len(birPkgs), len(gotPkgs))
	}
}

// TestTryLoadFrom_WindowsPESection covers embed_windows.go's tryLoadFrom
// finding a real splice.EmbedPE-embedded payload section. Gated to windows for
// the same reason TestTryLoadFrom_LinuxELFSection is: embed_windows.go
// is a build-tagged file, so tryLoadFrom only resolves to the PE-section
// reader when actually compiling for windows — there's no way to invoke
// it from a non-windows test binary, regardless of pe.NewFile's own
// host-independent parsing.
func TestTryLoadFrom_WindowsPESection(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("exercises the windows-only PE-section reader")
	}

	birPkgs, tyEnv := compileMinimalPackage(t)
	payload, err := MarshalPayload(birPkgs, tyEnv)
	if err != nil {
		t.Fatalf("MarshalPayload: %v", err)
	}

	selfPath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "packed.exe")
	if err := splice.EmbedPE(selfPath, payload, outPath); err != nil {
		t.Fatalf("splice.EmbedPE: %v", err)
	}

	gotPkgs, _, err := tryLoadFrom(outPath)
	if err != nil {
		t.Fatalf("tryLoadFrom: %v", err)
	}
	if len(gotPkgs) != len(birPkgs) {
		t.Fatalf("expected %d BIR packages back, got %d", len(birPkgs), len(gotPkgs))
	}
}

// TestTryLoadFrom_DarwinMachOSection covers embed_darwin.go's tryLoadFrom
// finding a real splice.EmbedMachO-embedded payload section. Gated to darwin for
// the same build-tag reason the linux/windows equivalents are.
func TestTryLoadFrom_DarwinMachOSection(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("exercises the darwin-only Mach-O-section reader")
	}

	birPkgs, tyEnv := compileMinimalPackage(t)
	payload, err := MarshalPayload(birPkgs, tyEnv)
	if err != nil {
		t.Fatalf("MarshalPayload: %v", err)
	}

	// The running test binary is itself a real darwin Mach-O64 executable
	// (already ad-hoc signed by Go's own linker on arm64) — reused as the
	// splice target so this test needs no cross-compile step.
	selfPath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "packed")
	if err := splice.EmbedMachO(selfPath, payload, outPath); err != nil {
		t.Fatalf("splice.EmbedMachO: %v", err)
	}

	gotPkgs, _, err := tryLoadFrom(outPath)
	if err != nil {
		t.Fatalf("tryLoadFrom: %v", err)
	}
	if len(gotPkgs) != len(birPkgs) {
		t.Fatalf("expected %d BIR packages back, got %d", len(birPkgs), len(gotPkgs))
	}
}
