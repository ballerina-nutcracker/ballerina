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

// Package executable packs/detects embedded BIR in bal build output.
// Each target platform's payload is embedded as a genuinely declared
// section in that platform's own binary format — see
// cli/internal/splice (the packer, called from bal build) and
// embed_darwin.go/embed_linux.go/embed_windows.go (the readers, one per
// platform, build-tag-gated so each balrt binary only links the parser
// its own target format needs).
package executable

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/bir"
	bircodec "github.com/ballerina-nutcracker/ballerina/bir/codec"
	balctx "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
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
	{OS: "windows", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
}

// ValidatePlatform returns an error unless platform is in supportedPlatforms.
// Shared by ResolveStub and buildNativeStub, so a target unsupported for
// the pre-built stub is rejected the same way regardless of path.
func ValidatePlatform(platform Platform) error {
	for _, sp := range supportedPlatforms {
		if sp == platform {
			return nil
		}
	}
	names := make([]string, len(supportedPlatforms))
	for i, p := range supportedPlatforms {
		names[i] = p.OS + "/" + p.Arch
	}
	return fmt.Errorf("unsupported target platform %s/%s; supported: %s",
		platform.OS, platform.Arch, strings.Join(names, ", "))
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

// ResolveStub locates the runner stub for packages with no native Go
// deps. overridePath, if set, must exist and is used as-is. Otherwise:
// flat <distDir>/balrt for the host platform, else
// <distDir>/rt/<GOOS>-<GOARCH>/balrt[.exe].
func ResolveStub(platform Platform, distDir, overridePath string) (string, error) {
	if overridePath != "" {
		if info, err := os.Stat(overridePath); err == nil && !info.IsDir() {
			return overridePath, nil
		}
		return "", fmt.Errorf("ballerina runtime binary not found at %s", overridePath)
	}

	if err := ValidatePlatform(platform); err != nil {
		return "", err
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

// TryLoad checks whether the running binary has embedded BIR: (pkgs,
// tyEnv, nil) if found, (nil, nil, nil) if plain, (nil, nil, err) if
// corrupt. tryLoadFrom is platform-specific — see embed_{darwin,linux,windows}.go.
func TryLoad() ([]*bir.BIRPackage, semtypes.Env, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, nil, nil
	}
	return tryLoadFrom(exe)
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

// MarshalPayload serializes birPkgs into the payload format
// cli/internal/splice's EmbedELF/EmbedPE/EmbedMachO all embed as-is:
// [uint32 BE count]([uint32 BE len][BIR bytes])*.
func MarshalPayload(birPkgs []*bir.BIRPackage, tyEnv semtypes.Env) ([]byte, error) {
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
