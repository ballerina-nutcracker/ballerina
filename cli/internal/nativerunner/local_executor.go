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

// Package nativerunner provides the CLI-local implementation of NativeExecutor.
// It builds a custom interpreter binary using the local Go toolchain and executes it.
package nativerunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"ballerina-lang-go/cli/internal/nativeexec"
)

const (
	// MinGoVersion is the minimum Go toolchain version required to build a native interpreter.
	MinGoVersion = "1.26"
)

// defaultTargetPackage is the Go import path (relative to interpreterRoot)
// New builds — the full bal CLI, matching bal run's re-exec use case.
const defaultTargetPackage = "cli/cmd"

// LocalExecutor builds a custom interpreter binary using the local Go toolchain.
// It implements nativeexec.NativeExecutor.
type LocalExecutor struct {
	// interpreterRoot is the directory that contains the ballerina-lang-go go.mod.
	interpreterRoot string
	// outputBinary is the path where the compiled native binary is written.
	// Relative paths are resolved against the interpreter root.
	outputBinary string
	// targetPackage is the Go import path, relative to interpreterRoot, that
	// go build compiles — e.g. "cli/cmd" (the full CLI, for bal run's
	// re-exec use) or "cli/cmd/balrt" (the slim runtime-only stub, for bal
	// build). The generated blank-import file is injected into this same
	// package directory via the build overlay, so its init()s register
	// against whichever binary is actually being produced.
	targetPackage string
}

var _ nativeexec.NativeExecutor = (*LocalExecutor)(nil)

// New creates a LocalExecutor targeting the full bal CLI (cli/cmd) —
// bal run's use case, where the rebuilt binary re-executes and takes over
// the whole process. interpreterRoot is the directory containing the
// ballerina-lang-go go.mod; outputBinary is the destination path for the
// compiled native interpreter (typically <project>/target/bin/bal).
func New(interpreterRoot, outputBinary string) *LocalExecutor {
	return NewForTarget(interpreterRoot, outputBinary, defaultTargetPackage)
}

// NewForTarget creates a LocalExecutor targeting an arbitrary Go package
// (relative to interpreterRoot) instead of the full CLI — e.g. bal build
// uses "cli/cmd/balrt" to produce a slim, standalone stub with native code
// woven in, rather than rebuilding the full bal CLI.
func NewForTarget(interpreterRoot, outputBinary, targetPackage string) *LocalExecutor {
	return &LocalExecutor{
		interpreterRoot: interpreterRoot,
		outputBinary:    outputBinary,
		targetPackage:   targetPackage,
	}
}

// Available reports true when a sufficiently new Go toolchain is on PATH and
// the interpreter source root is reachable.
func (e *LocalExecutor) Available() bool {
	goExe, err := exec.LookPath("go")
	if err != nil {
		return false
	}
	if !goVersionAtLeast(goExe, MinGoVersion) {
		return false
	}
	_, err = os.Stat(filepath.Join(e.interpreterRoot, "go.mod"))
	return err == nil
}

// goVersionAtLeast reports whether the Go binary at goExe is at least minVersion.
// minVersion is a dot-separated string such as "1.26" or "1.26.0".
func goVersionAtLeast(goExe, minVersion string) bool {
	out, err := exec.Command(goExe, "version").Output()
	if err != nil {
		return false
	}
	// "go version go1.26.1 linux/amd64" → field[2] = "go1.26.1"
	fields := strings.Fields(string(out))
	if len(fields) < 3 {
		return false
	}
	installed := strings.TrimPrefix(fields[2], "go")
	return versionAtLeast(installed, minVersion)
}

// versionAtLeast reports whether dot-separated version a is >= b.
// Missing trailing components are treated as zero: "1.26" == "1.26.0".
// Non-numeric components (e.g. "rc1", "beta2") are treated as incompatible.
func versionAtLeast(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	n := max(len(aParts), len(bParts))
	for i := range n {
		av, bv := 0, 0
		if i < len(aParts) {
			var err error
			av, err = strconv.Atoi(aParts[i])
			if err != nil {
				return false
			}
		}
		if i < len(bParts) {
			var err error
			bv, err = strconv.Atoi(bParts[i])
			if err != nil {
				return false
			}
		}
		if av != bv {
			return av > bv
		}
	}
	return true
}

// Prepare builds a custom interpreter that embeds all req.Payloads' native Go
// sources and returns a Runner that re-executes the program via that binary.
// If a previously built binary with a matching fingerprint already exists at
// outputBinary, it is reused without rebuilding.
func (e *LocalExecutor) Prepare(ctx context.Context, req nativeexec.NativeRunnerRequest) (nativeexec.Runner, error) {
	outBin, tmpDir, err := e.buildOrReuse(ctx, req)
	if err != nil {
		return nil, err
	}
	return &localRunner{
		binaryPath: outBin,
		args:       req.Args,
		env:        nativeexec.AppendNativeMode(req.Env),
		stdout:     req.Stdout,
		stderr:     req.Stderr,
		tmpDir:     tmpDir, // "" on a cache hit — nothing to clean up
	}, nil
}

// Build compiles (or reuses a cached) binary that embeds all req.Payloads'
// native Go sources and returns its path. Unlike Prepare, it never wraps the
// result for re-exec: bal build needs a bare artifact path to hand to
// executable.Pack, not a Runner that takes over the process. Any temporary
// build directory is cleaned up before returning, since there's no re-exec
// step left to keep it alive for.
func (e *LocalExecutor) Build(ctx context.Context, req nativeexec.NativeRunnerRequest) (string, error) {
	outBin, tmpDir, err := e.buildOrReuse(ctx, req)
	if tmpDir != "" {
		defer func() { _ = os.RemoveAll(tmpDir) }()
	}
	if err != nil {
		return "", err
	}
	return outBin, nil
}

// buildOrReuse is the shared core of Prepare and Build: it resolves the
// output binary path, reuses it as-is on a fingerprint cache hit (tmpDir ==
// ""), or else assembles a temp bundle module, runs go build, and persists
// the new fingerprint. The caller owns cleaning up tmpDir when non-empty.
func (e *LocalExecutor) buildOrReuse(ctx context.Context, req nativeexec.NativeRunnerRequest) (outBin, tmpDir string, err error) {
	// Empty TargetOS/TargetArch means build for the host — the same
	// convention Go's own GOOS/GOARCH env vars use.
	targetOS := req.TargetOS
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	targetArch := req.TargetArch
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	// Fast path: reuse cached binary when native imports haven't changed.
	fingerprint, fpErr := localFingerprint(e.interpreterRoot, req.Payloads, targetOS, targetArch)
	if fpErr == nil {
		if cachedBin, ok := e.loadCachedBinary(fingerprint); ok {
			return cachedBin, "", nil
		}
	}

	// dir (not the named tmpDir return) holds the real temp path throughout,
	// so the cleanup defer below always targets the directory that was
	// actually created — even on early-error returns, which set the named
	// tmpDir return to "" before the defer runs.
	dir, mkErr := os.MkdirTemp("", "bal-bundle-*")
	if mkErr != nil {
		return "", "", fmt.Errorf("creating temp bundle dir: %w", mkErr)
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(dir)
		}
	}()

	// Write each native package into its own subdirectory with its own go.mod.
	for _, payload := range req.Payloads {
		pkgDir := filepath.Join(dir, moduleDirName(payload.GoModuleName()))
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			return "", "", fmt.Errorf("creating package dir: %w", err)
		}
		if err := writeNativeFiles(pkgDir, payload); err != nil {
			return "", "", err
		}
		modContent := fmt.Sprintf("module %s\n\ngo %s\n\nrequire ballerina-lang-go v0.0.0\nreplace ballerina-lang-go => %s\n",
			payload.GoModuleName(), MinGoVersion, e.interpreterRoot)
		if err := os.WriteFile(filepath.Join(pkgDir, "go.mod"), []byte(modContent), 0o600); err != nil {
			return "", "", fmt.Errorf("writing go.mod for %s: %w", payload.GoModuleName(), err)
		}
	}

	// Generate the init file that blank-imports every native package.
	var initContent strings.Builder
	initContent.WriteString("package main\n\n")
	for _, payload := range req.Payloads {
		fmt.Fprintf(&initContent, "import _ %q\n", payload.GoModuleName())
	}
	initFile := filepath.Join(dir, "native_init_gen.go")
	if err := os.WriteFile(initFile, []byte(initContent.String()), 0o600); err != nil {
		return "", "", fmt.Errorf("writing native_init_gen.go: %w", err)
	}

	// Write overlay.json that injects native_init_gen.go into the target package.
	overlayDst := filepath.Join(e.interpreterRoot, filepath.FromSlash(e.targetPackage), "native_init_gen.go")
	overlay := map[string]map[string]string{
		"Replace": {overlayDst: initFile},
	}
	overlayJSON, err := json.Marshal(overlay)
	if err != nil {
		return "", "", fmt.Errorf("marshalling overlay: %w", err)
	}
	overlayFile := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayFile, overlayJSON, 0o600); err != nil {
		return "", "", fmt.Errorf("writing overlay.json: %w", err)
	}

	// Write patched go.mod + go.sum into dir with all native packages.
	patchedModFile, err := writePatchedGoMod(dir, e.interpreterRoot, req.Payloads, dir)
	if err != nil {
		return "", "", err
	}

	// Ensure output directory exists. Resolve relative paths against interpreterRoot
	// so that build, fingerprint, and run all reference the same absolute path.
	outBin = e.outputBinary
	if !filepath.IsAbs(outBin) {
		outBin = filepath.Join(e.interpreterRoot, outBin)
	}
	if err := os.MkdirAll(filepath.Dir(outBin), 0o755); err != nil {
		return "", "", fmt.Errorf("creating output directory: %w", err)
	}

	stderr := req.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	if _, err := fmt.Fprintln(stderr, "info: building native interpreter using local Go toolchain"); err != nil {
		return "", "", fmt.Errorf("writing build status: %w", err)
	}
	buildCmd := exec.CommandContext(ctx, "go", "build",
		"-C", e.interpreterRoot,
		"-modfile", patchedModFile,
		"-overlay", overlayFile,
		"-tags", "native_interp",
		"-o", outBin,
		"./"+e.targetPackage,
	)
	buildCmd.Env = crossCompileEnv(targetOS, targetArch)
	buildCmd.Stdout = io.Discard
	buildCmd.Stderr = stderr
	if err := buildCmd.Run(); err != nil {
		return "", "", fmt.Errorf("building native interpreter: %w", err)
	}

	// Persist fingerprint so the next invocation can skip the build.
	if fpErr == nil {
		_ = nativeexec.WriteFingerprint(outBin, fingerprint)
	}

	ok = true
	return outBin, dir, nil
}

// loadCachedBinary returns the already-compiled binary's path when the stored
// fingerprint matches the current one. Returns false if a rebuild is needed.
func (e *LocalExecutor) loadCachedBinary(fingerprint string) (string, bool) {
	outBin := e.outputBinary
	if !filepath.IsAbs(outBin) {
		outBin = filepath.Join(e.interpreterRoot, outBin)
	}
	fpFile := nativeexec.FingerprintPath(outBin)
	existing, err := os.ReadFile(fpFile)
	if err != nil || strings.TrimSpace(string(existing)) != fingerprint {
		return "", false
	}
	if _, err := os.Stat(outBin); err != nil {
		return "", false
	}
	return outBin, true
}

// localFingerprint hashes the interpreter's go.mod + go.sum, the installed Go
// toolchain version, and the target GOOS/GOARCH (to catch toolchain upgrades,
// dependency changes, and cross-compilation target changes) plus the payload
// contents via FingerprintPayloads. targetOS/targetArch must already be
// resolved to concrete values (the host's own, if the caller didn't
// override) — this function doesn't itself default empty strings to host,
// so a fingerprint always reflects exactly what was actually built.
func localFingerprint(interpreterRoot string, payloads []nativeexec.NativePayload, targetOS, targetArch string) (string, error) {
	seeds := make([][]byte, 0, 4)
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(interpreterRoot, name))
		if err != nil {
			return "", fmt.Errorf("reading interpreter %s: %w", name, err)
		}
		seeds = append(seeds, data)
	}
	if ver, err := installedGoVersion(); err == nil {
		seeds = append(seeds, []byte(ver))
	}
	seeds = append(seeds, []byte(targetOS+"/"+targetArch))
	return nativeexec.FingerprintPayloads(payloads, seeds...)
}

// crossCompileEnv returns the environment for the go build subprocess,
// targeting targetOS/targetArch. CGO_ENABLED=0 matches the existing
// non-native cross-compilation path (build.go) and is a safety net, not
// strictly required: native dependencies are confirmed Go-sources-only (no
// cgo), so this should never actually change behavior — but if one ever
// snuck in a cgo import, this makes it fail clearly (no C cross-compiler
// available) rather than silently attempting something unsupported. Any
// GOOS/GOARCH/CGO_ENABLED already present in the parent environment are
// dropped first, so there's exactly one value for each — os/exec does not
// guarantee which of two duplicate keys wins.
func crossCompileEnv(targetOS, targetArch string) []string {
	base := os.Environ()
	env := make([]string, 0, len(base)+3)
	for _, e := range base {
		if strings.HasPrefix(e, "GOOS=") || strings.HasPrefix(e, "GOARCH=") || strings.HasPrefix(e, "CGO_ENABLED=") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "GOOS="+targetOS, "GOARCH="+targetArch, "CGO_ENABLED=0")
}

// installedGoVersion returns the full version string from `go version`.
func installedGoVersion() (string, error) {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// writeNativeFiles copies Go source files from payload.FS() into dir.
func writeNativeFiles(dir string, payload nativeexec.NativePayload) error {
	return fs.WalkDir(payload.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		data, err := fs.ReadFile(payload.FS(), p)
		if err != nil {
			return fmt.Errorf("reading native source %s: %w", p, err)
		}
		dst := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o600)
	})
}

// writePatchedGoMod reads the interpreter's go.mod, appends a require+replace
// pair for every native payload, and writes patched-go.mod + patched-go.sum
// into dstDir. Each payload's module root is tmpDir/<moduleDirName>.
// Returns the path to the patched go.mod file.
func writePatchedGoMod(dstDir, interpreterRoot string, payloads []nativeexec.NativePayload, tmpDir string) (string, error) {
	src := filepath.Join(interpreterRoot, "go.mod")
	original, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("reading interpreter go.mod: %w", err)
	}

	var patched bytes.Buffer
	patched.Write(bytes.TrimRight(original, "\n"))
	for _, payload := range payloads {
		pkgDir := filepath.Join(tmpDir, moduleDirName(payload.GoModuleName()))
		fmt.Fprintf(&patched, "\nrequire %s v0.0.0\nreplace %s => %s",
			payload.GoModuleName(), payload.GoModuleName(), pkgDir)
	}
	patched.WriteByte('\n')

	dst := filepath.Join(dstDir, "patched-go.mod")
	if err := os.WriteFile(dst, patched.Bytes(), 0o600); err != nil {
		return "", fmt.Errorf("writing patched go.mod: %w", err)
	}

	// -modfile expects a matching patched-go.sum alongside patched-go.mod.
	sumSrc := filepath.Join(interpreterRoot, "go.sum")
	sumData, err := os.ReadFile(sumSrc)
	if err != nil {
		return "", fmt.Errorf("reading interpreter go.sum: %w", err)
	}
	sumDst := filepath.Join(dstDir, "patched-go.sum")
	if err := os.WriteFile(sumDst, sumData, 0o600); err != nil {
		return "", fmt.Errorf("writing patched go.sum: %w", err)
	}

	return dst, nil
}

// moduleDirName converts a Go module path into a safe directory name by
// replacing forward slashes with underscores.
func moduleDirName(modulePath string) string {
	return strings.ReplaceAll(modulePath, "/", "_")
}

// localRunner executes the compiled native interpreter.
type localRunner struct {
	binaryPath string
	args       []string
	env        []string
	stdout     io.Writer
	stderr     io.Writer
	// tmpDir holds temporary build artifacts. Empty when using a cached binary.
	tmpDir string
}

var _ nativeexec.Runner = (*localRunner)(nil)

func (r *localRunner) Run(ctx context.Context) (nativeexec.ExitCode, error) {
	cmd := exec.CommandContext(ctx, r.binaryPath, r.args...)
	cmd.Env = r.env
	cmd.Stdin = os.Stdin
	cmd.Stdout = r.stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = r.stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nativeexec.ExitCode(exitErr.ExitCode()), nil
		}
		return 1, fmt.Errorf("executing native interpreter: %w", err)
	}
	return 0, nil
}

func (r *localRunner) Close() error {
	if r.tmpDir == "" {
		return nil
	}
	return os.RemoveAll(r.tmpDir)
}
