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

// Package executable handles BIR embedding for bal build output and startup detection.
//
// A compiled Ballerina executable is an unmodified bal binary with a BIR payload
// and a 16-byte trailer appended to it:
//
//	[bal binary bytes] [BIR payload] [8-byte payload offset] [8-byte magic]
//
// At startup, the binary reads its own last 16 bytes. Finding the magic means it
// is running as a compiled program — it deserializes the payload and runs the BIR
// instead of entering the CLI.
package executable

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"ballerina-lang-go/bir"
	bircodec "ballerina-lang-go/bir/codec"
	balctx "ballerina-lang-go/context"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/semtypes"
)

const (
	magic       = "BALEXE\x00\x01"
	trailerSize = 16 // 8-byte payload offset + 8-byte magic
)

// runtimeStubDirName and balrtStubName locate the slim runner stub relative
// to the bal distribution's own directory (see DistributionDir):
// <distDir>/rt/<GOOS>-<GOARCH>/balrt[.exe]. See ResolveStub.
const runtimeStubDirName = "rt"
const balrtStubName = "balrt"

// Platform identifies a build target (GOOS/GOARCH).
type Platform struct {
	OS   string
	Arch string
}

// HostPlatform is the platform bal itself is currently running on.
func HostPlatform() Platform {
	return Platform{OS: goruntime.GOOS, Arch: goruntime.GOARCH}
}

// ResolveTargetPlatform builds a Platform from optional target OS/arch
// overrides (e.g. bal build's --target-os/--target-arch flags), defaulting
// whichever is empty to the host's own value — the same convention Go's own
// GOOS/GOARCH environment variables use, so setting only one dimension does
// what a user would expect (e.g. --target-arch arm64 alone on a
// darwin/amd64 host resolves to darwin/arm64, not an error).
func ResolveTargetPlatform(targetOS, targetArch string) Platform {
	p := HostPlatform()
	if targetOS != "" {
		p.OS = targetOS
	}
	if targetArch != "" {
		p.Arch = targetArch
	}
	return p
}

// supportedPlatforms is the fixed set of cross-compilation targets bal build
// supports — chosen by cross-referencing Rust's and Node.js's own officially
// supported platform tiers (see migration-docs/specs), not a technical
// ceiling: nothing here prevents another platform's stub from working if
// one were built and placed at the right path. It's a provisioning-cost
// decision, kept as a fixed list so an unsupported/mistyped target fails
// clearly instead of silently looking for a stub that will never exist.
var supportedPlatforms = []Platform{
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
}

func isSupportedPlatform(p Platform) bool {
	for _, sp := range supportedPlatforms {
		if sp == p {
			return true
		}
	}
	return false
}

func supportedPlatformsList() string {
	names := make([]string, len(supportedPlatforms))
	for i, p := range supportedPlatforms {
		names[i] = p.OS + "/" + p.Arch
	}
	return strings.Join(names, ", ")
}

// Key identifies which stub ResolveStub should produce for a build.
//
// Fingerprint == "" means "no native Go dependencies" — resolved by looking
// up a pre-built, installer-provided stub; no Go toolchain involved. A
// non-empty Fingerprint (a hash over the resolved native-dependency set)
// will route to a toolchain-based custom build with that native code woven
// in, once that support lands (see
// migration-docs/specs/build-command-architecture.md); ResolveStub rejects
// it for now rather than mishandling it.
type Key struct {
	Platform    Platform
	Fingerprint string
}

// DistributionDir returns the directory containing the currently running bal
// distribution — the root a release archive was extracted to, which
// ResolveStub expects to find a sibling rt/<GOOS>-<GOARCH>/balrt[.exe] under
// for every supported platform.
//
// It resolves through any symlink pointing at the real binary (e.g. one
// placed on PATH via "ln -s /opt/ballerina-1.2.3/bal /usr/local/bin/bal", a
// common install pattern) — os.Executable() alone does not guarantee this,
// and returning the symlink's own directory (/usr/local/bin) instead of the
// real distribution root would make the rt/ lookup fail for anyone who
// installs bal that way instead of adding the whole distribution directory
// to PATH.
func DistributionDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating bal's own executable: %w", err)
	}
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving bal's real executable path: %w", err)
	}
	return filepath.Dir(real), nil
}

// ResolveStub returns the path to the runner stub that Pack should embed the
// BIR payload into.
//
// For key.Fingerprint == "" (no native Go dependencies), bal build never
// invokes the Go toolchain: it looks up a slim, runtime-only stub (no
// compiler, no CLI) at <distDir>/rt/<GOOS>-<GOARCH>/balrt (".exe" on a
// Windows target), where distDir is typically the result of DistributionDir
// — so a machine with only a released bal install and no Go can still build
// pure-Ballerina packages, for the host platform or a cross-compiled target
// alike. Every release archive bundles rt/<platform>/balrt for all platforms
// bal build supports (see supportedPlatforms), not just the one it was built
// for, so cross-compiling works immediately after extracting a single
// archive — no separate per-target provisioning step. ResolveStub itself
// only ever reads that location; if the stub isn't there, this returns a
// clear, actionable error rather than silently falling back to anything else
// or compiling one on the spot.
//
// key.Platform must be one of the platforms bal build supports (see
// supportedPlatforms) — an unsupported or mistyped target fails clearly
// rather than silently looking for a stub that will never exist.
//
// overridePath, when non-empty, is used as-is instead of computing the
// predefined path — an explicit escape hatch so the default installation
// layout can change later without breaking a bal build already compiled
// against a specific stub location. It comes from cli/cmd.RuntimeStubPath,
// a variable set via -ldflags at bal's own build time (the same mechanism
// as Version) — not a bal build flag, so this stays entirely transparent to
// whoever just runs bal build. It bypasses both distDir and the key.Platform
// lookup entirely (a packager taking an explicit path is assumed to already
// match whatever they intend), but is still validated to exist before use,
// with the same clear-error behavior as the default path.
func ResolveStub(key Key, distDir, overridePath string) (string, error) {
	if key.Fingerprint != "" {
		return "", fmt.Errorf("native Go dependencies are not yet supported by bal build")
	}

	if overridePath != "" {
		if info, err := os.Stat(overridePath); err == nil && !info.IsDir() {
			return overridePath, nil
		}
		return "", fmt.Errorf("runner stub not found at %s (overridden via RuntimeStubPath at bal's build time)", overridePath)
	}

	if !isSupportedPlatform(key.Platform) {
		return "", fmt.Errorf("unsupported target platform %s/%s; supported: %s",
			key.Platform.OS, key.Platform.Arch, supportedPlatformsList())
	}

	name := balrtStubName
	if key.Platform.OS == "windows" {
		name += ".exe"
	}
	platformDir := key.Platform.OS + "-" + key.Platform.Arch
	stubPath := filepath.Join(distDir, runtimeStubDirName, platformDir, name)

	if info, err := os.Stat(stubPath); err == nil && !info.IsDir() {
		return stubPath, nil
	}
	return "", fmt.Errorf(
		"runner stub not found at %s; build and place it there before running bal build "+
			"(GOOS=%s GOARCH=%s go build -o %s ./cli/cmd/balrt, from the ballerina-lang-go module root)",
		stubPath, key.Platform.OS, key.Platform.Arch, stubPath)
}

// Pack writes a self-contained Ballerina executable to outPath.
//
// The output is: [stub bytes] [BIR payload] [16-byte trailer]
//
// stubPath is the path to the runner binary for the target platform — typically
// the currently running bal binary (os.Executable()) for same-platform builds.
// outPath is created (with parent directories) and made executable.
func Pack(stubPath string, birPkgs []*bir.BIRPackage, tyEnv semtypes.Env, outPath string) error {
	stub, err := os.Open(stubPath)
	if err != nil {
		return fmt.Errorf("opening runner stub: %w", err)
	}
	defer func() { _ = stub.Close() }()

	stubInfo, err := stub.Stat()
	if err != nil {
		return fmt.Errorf("stat runner stub: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	// Ensure execute bits are set even when the file already existed (O_TRUNC
	// reuses the inode and does not apply the mode argument).
	if err := os.Chmod(outPath, 0o755); err != nil {
		_ = out.Close()
		return fmt.Errorf("setting output file permissions: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, stub); err != nil {
		return fmt.Errorf("copying stub: %w", err)
	}
	payloadOffset := stubInfo.Size()

	payload, err := marshalPayload(birPkgs, tyEnv)
	if err != nil {
		return err
	}
	if _, err := out.Write(payload); err != nil {
		return fmt.Errorf("writing BIR payload: %w", err)
	}

	trailer := make([]byte, trailerSize)
	binary.LittleEndian.PutUint64(trailer[:8], uint64(payloadOffset))
	copy(trailer[8:], magic)
	if _, err := out.Write(trailer); err != nil {
		return fmt.Errorf("writing trailer: %w", err)
	}
	return nil
}

// TryLoad checks whether the running binary has embedded BIR.
//
// Returns (pkgs, tyEnv, nil) if an embedded program is found.
// Returns (nil, nil, nil) if this is a plain bal binary with no embedded BIR.
// Returns (nil, nil, err) if the magic is present but the payload is corrupt.
func TryLoad() ([]*bir.BIRPackage, semtypes.Env, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, nil
	}
	return tryLoadFrom(exe)
}

// tryLoadFrom implements TryLoad against an explicit file path, so tests can
// exercise it against a constructed file without needing to replace the test
// binary itself (which is what os.Executable() would otherwise resolve to).
func tryLoadFrom(exe string) ([]*bir.BIRPackage, semtypes.Env, error) {
	f, err := os.Open(exe)
	if err != nil {
		return nil, nil, nil
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil || info.Size() < int64(trailerSize) {
		return nil, nil, nil
	}

	trailer := make([]byte, trailerSize)
	if _, err := f.ReadAt(trailer, info.Size()-int64(trailerSize)); err != nil {
		return nil, nil, nil
	}
	if string(trailer[8:]) != magic {
		return nil, nil, nil
	}

	rawOffset := binary.LittleEndian.Uint64(trailer[:8])
	// Guard against corrupt offsets that wrap to negative when cast to int64,
	// which would make payloadSize a large positive value and cause OOM.
	if rawOffset > uint64(info.Size()-int64(trailerSize)) {
		return nil, nil, fmt.Errorf("invalid embedded payload offset %d", rawOffset)
	}
	payloadOffset := int64(rawOffset)
	payloadSize := info.Size() - payloadOffset - int64(trailerSize)
	if payloadSize <= 0 {
		return nil, nil, fmt.Errorf("invalid embedded payload size %d", payloadSize)
	}

	payload := make([]byte, payloadSize)
	if _, err := f.ReadAt(payload, payloadOffset); err != nil {
		return nil, nil, fmt.Errorf("reading embedded payload: %w", err)
	}

	pkgs, tyEnv, err := unmarshalPayload(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("corrupt embedded program: %w", err)
	}
	return pkgs, tyEnv, nil
}

// Run initializes and executes the given BIR packages against a fresh
// runtime, blocking until the program's listening phase ends. It returns the
// process exit code the caller should exit with (0 on success).
//
// Shared by bal's main() (embedded-program fast path) and balrt's main() (its
// only path) so the two never drift on how a compiled Ballerina program is
// actually run.
func Run(birPkgs []*bir.BIRPackage, tyEnv semtypes.Env) int {
	pal, cleanupSignals := palnative.NewPlatform()
	defer cleanupSignals()

	rt := runtime.NewRuntime(pal, tyEnv)
	var initErr error
	for _, pkg := range birPkgs {
		if err := rt.Init(*pkg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			initErr = err
			break
		}
	}
	rt.Listen()
	if initErr != nil {
		return 1
	}
	return int(<-rt.ExitStatus)
}

func marshalPayload(birPkgs []*bir.BIRPackage, tyEnv semtypes.Env) ([]byte, error) {
	// Format: [uint32 count] ([uint32 len] [BIR bytes])*
	if len(birPkgs) > math.MaxUint32 {
		return nil, fmt.Errorf("too many BIR packages: %d", len(birPkgs))
	}
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, uint32(len(birPkgs)))
	buf := append([]byte(nil), count...)

	for _, pkg := range birPkgs {
		data, err := bircodec.Marshal(tyEnv, pkg)
		if err != nil {
			return nil, fmt.Errorf("serializing %s: %w", pkg.PackageID.PkgName.Value(), err)
		}
		if len(data) > math.MaxUint32 {
			return nil, fmt.Errorf("serialized BIR package too large: %d bytes", len(data))
		}
		lenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBytes, uint32(len(data)))
		buf = append(buf, lenBytes...)
		buf = append(buf, data...)
	}
	return buf, nil
}

func unmarshalPayload(payload []byte) ([]*bir.BIRPackage, semtypes.Env, error) {
	if len(payload) < 4 {
		return nil, nil, fmt.Errorf("payload too short")
	}

	tyEnv := semtypes.CreateTypeEnv()
	env := balctx.NewCompilerEnvironment(tyEnv, false)
	ctx := balctx.NewCompilerContext(env)

	count := int(binary.BigEndian.Uint32(payload[:4]))
	if count > (len(payload)-4)/4 {
		return nil, nil, fmt.Errorf("invalid package count %d", count)
	}
	pos := 4
	pkgs := make([]*bir.BIRPackage, 0, count)

	for i := range count {
		if pos+4 > len(payload) {
			return nil, nil, fmt.Errorf("truncated at package %d", i)
		}
		pkgLen := int(binary.BigEndian.Uint32(payload[pos : pos+4]))
		pos += 4

		if pos+pkgLen > len(payload) {
			return nil, nil, fmt.Errorf("truncated BIR at package %d", i)
		}
		pkg, err := bircodec.Unmarshal(ctx, payload[pos:pos+pkgLen])
		if err != nil {
			return nil, nil, fmt.Errorf("deserializing package %d: %w", i, err)
		}
		pos += pkgLen
		pkgs = append(pkgs, pkg)
	}
	return pkgs, tyEnv, nil
}
