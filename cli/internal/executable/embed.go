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

// Package executable packs/detects embedded BIR in bal build output: an
// unmodified bal binary with a payload and 16-byte trailer appended:
//
//	[bal binary bytes] [BIR payload] [8-byte payload offset] [8-byte magic]
//
// Finding the magic at startup means: deserialize and run instead of the CLI.
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

// runtimeStubDirName and balrtStubName locate the runner stub at
// <distDir>/rt/<GOOS>-<GOARCH>/balrt[.exe]; see DistributionDir, ResolveStub.
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

// ResolveTargetPlatform defaults empty --target-os/-arch to the host's
// value, like Go's own GOOS/GOARCH — so setting only one dimension works.
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

// supportedPlatforms is the fixed list of targets bal build provisions
// stubs for — not a technical ceiling, just so a typo fails clearly.
var supportedPlatforms = []Platform{
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
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

// DistributionDir returns bal's own distribution root (rt/<GOOS>-<GOARCH>/
// balrt[.exe] lives under it), resolving symlinks first (e.g. bal on PATH).
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

// ResolveStub locates the runner stub for packages with no native Go deps
// (bal build uses buildNativeStub for those). overridePath
// (cli/cmd.RuntimeStubPath), if set, must exist and is used as-is.
// Otherwise: flat <distDir>/balrt (local dev builds) for the host
// platform, else <distDir>/rt/<GOOS>-<GOARCH>/balrt[.exe].
func ResolveStub(platform Platform, distDir, overridePath string) (string, error) {
	if overridePath != "" {
		if info, err := os.Stat(overridePath); err == nil && !info.IsDir() {
			return overridePath, nil
		}
		return "", fmt.Errorf("ballerina runtime binary not found at %s", overridePath)
	}

	if !isSupportedPlatform(platform) {
		return "", fmt.Errorf("unsupported target platform %s/%s; supported: %s",
			platform.OS, platform.Arch, supportedPlatformsList())
	}

	name := balrtStubName
	if platform.OS == "windows" {
		name += ".exe"
	}

	if platform == HostPlatform() {
		flatStubPath := filepath.Join(distDir, name)
		if info, err := os.Stat(flatStubPath); err == nil && !info.IsDir() {
			return flatStubPath, nil
		}
	}

	platformDir := platform.OS + "-" + platform.Arch
	stubPath := filepath.Join(distDir, runtimeStubDirName, platformDir, name)

	if info, err := os.Stat(stubPath); err == nil && !info.IsDir() {
		return stubPath, nil
	}
	return "", fmt.Errorf(
		"ballerina runtime binary for %s/%s not found at %s; your bal installation may be incomplete — try reinstalling bal",
		platform.OS, platform.Arch, stubPath)
}

// Pack writes [stub bytes][BIR payload][16-byte trailer] to outPath,
// creating parent dirs and making it executable. Writes to a sibling temp
// file and renames on success, so a partial failure can't corrupt outPath.
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

	payload, err := marshalPayload(birPkgs, tyEnv)
	if err != nil {
		return err
	}

	outDir := filepath.Dir(outPath)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	tmp, err := os.CreateTemp(outDir, ".bal-pack-*")
	if err != nil {
		return fmt.Errorf("creating temp output file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op once renamed over outPath

	if err := writePackedFile(tmp, stub, stubInfo.Size(), payload); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp output file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("setting output file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return fmt.Errorf("renaming output file into place: %w", err)
	}
	return nil
}

// writePackedFile writes stub bytes, then payload, then the trailer to out.
func writePackedFile(out io.Writer, stub io.Reader, stubSize int64, payload []byte) error {
	if _, err := io.Copy(out, stub); err != nil {
		return fmt.Errorf("copying stub: %w", err)
	}
	if _, err := out.Write(payload); err != nil {
		return fmt.Errorf("writing BIR payload: %w", err)
	}

	trailer := make([]byte, trailerSize)
	binary.LittleEndian.PutUint64(trailer[:8], uint64(stubSize))
	copy(trailer[8:], magic)
	if _, err := out.Write(trailer); err != nil {
		return fmt.Errorf("writing trailer: %w", err)
	}
	return nil
}

// TryLoad checks whether the running binary has embedded BIR: (pkgs, tyEnv,
// nil) if found, (nil, nil, nil) if plain, (nil, nil, err) if corrupt.
func TryLoad() ([]*bir.BIRPackage, semtypes.Env, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, nil
	}
	return tryLoadFrom(exe)
}

// tryLoadFrom implements TryLoad against an explicit path, so tests can
// use a constructed file instead of the test binary itself.
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
	// Rejects an offset that would wrap negative as int64 and blow up payloadSize.
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

// Run initializes and executes birPkgs, blocking until listening ends, and
// returns the exit code — shared by bal's and balrt's main().
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
	if len(birPkgs) == 0 {
		return nil, fmt.Errorf("no BIR packages to embed")
	}
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
	if count == 0 {
		return nil, nil, fmt.Errorf("payload contains no BIR packages")
	}
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
	if pos != len(payload) {
		return nil, nil, fmt.Errorf("unexpected trailing payload data")
	}
	return pkgs, tyEnv, nil
}
