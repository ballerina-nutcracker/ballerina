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

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ballerina-lang-go/cli/internal/executable"
	"ballerina-lang-go/cli/internal/nativeexec"
	debugcommon "ballerina-lang-go/common"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/tools/diagnostics"

	"github.com/spf13/cobra"
)

const binSubdir = "bin"

// RuntimeStubPath overrides the default <dist>/rt/<os>-<arch> runner-stub
// lookup (executable.ResolveStub) when non-empty. It is set via
// -ldflags at bal's own build time (e.g. -X main.RuntimeStubPath=/custom/path),
// the same mechanism as Version (see version.go) — not a bal build flag, so
// the stub's location stays transparent to whoever just runs bal build. Only
// whoever builds/packages bal itself would ever set this, e.g. to match a
// non-default installation layout.
var RuntimeStubPath = ""

type buildOptions struct {
	dumpTokens    bool
	dumpST        bool
	dumpAST       bool
	dumpCFG       bool
	dumpBIR       bool
	traceRecovery bool
	stats         bool
	statsOneline  bool
	logFile       string
	format        string
	output        string // -o: explicit output path
	targetOS      string // cross-compile target OS; "" defaults to the host OS
	targetArch    string // cross-compile target architecture; "" defaults to the host arch
}

var buildCmd = createBuildCmd()

func createBuildCmd() *cobra.Command {
	opts := &buildOptions{}
	cmd := &cobra.Command{
		Use:   "build [<package-dir>]",
		Short: "Compile the current package into a standalone executable",
		Long: `	Compile the current Ballerina package into a standalone executable.

	The output binary embeds the compiled program and the Ballerina runtime.
	It runs without a bal installation and without the source files present.

	The default output path is <project>/target/bin/<package-name>.
	Use -o to specify a different path.

	Use --target-os/--target-arch to cross-compile for a different platform.
	Either may be given alone; the other defaults to the host's own value.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(cmd, args, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.dumpTokens, "dump-tokens", false, "Dump lexer tokens")
	cmd.Flags().BoolVar(&opts.dumpST, "dump-st", false, "Dump syntax tree")
	cmd.Flags().BoolVar(&opts.dumpAST, "dump-ast", false, "Dump abstract syntax tree")
	cmd.Flags().BoolVar(&opts.dumpCFG, "dump-cfg", false, "Dump control flow graph")
	cmd.Flags().BoolVar(&opts.dumpBIR, "dump-bir", false, "Dump Ballerina Intermediate Representation")
	cmd.Flags().BoolVar(&opts.traceRecovery, "trace-recovery", false, "Enable error recovery tracing")
	cmd.Flags().BoolVar(&opts.stats, "stats", false, "Print per-stage compilation timing statistics")
	cmd.Flags().BoolVar(&opts.statsOneline, "stats-oneline", false, "Print per-stage compilation timing totals only")
	cmd.Flags().StringVar(&opts.logFile, "log-file", "", "Write debug output to specified file")
	cmd.Flags().StringVar(&opts.format, "format", "", "Output format for dump operations (dot)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output path (default: target/bin/<package-name>)")
	cmd.Flags().StringVar(&opts.targetOS, "target-os", "", "Cross-compile target OS (default: host OS)")
	cmd.Flags().StringVar(&opts.targetArch, "target-arch", "", "Cross-compile target architecture (default: host arch)")
	return cmd
}

func buildError(w io.Writer, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	printErrorTo(w, err, "build [<package-dir>]", false)
	return err
}

func runBuild(cmd *cobra.Command, args []string, opts *buildOptions) error {
	stderr := cmd.ErrOrStderr()

	buildOpts := projects.NewBuildOptionsBuilder().
		WithDumpAST(opts.dumpAST).
		WithDumpBIR(opts.dumpBIR).
		WithDumpCFG(opts.dumpCFG).
		WithDumpCFGFormat(projects.ParseCFGFormat(opts.format)).
		WithDumpTokens(opts.dumpTokens).
		WithDumpST(opts.dumpST).
		WithTraceRecovery(opts.traceRecovery).
		WithStats(opts.stats || opts.statsOneline).
		Build()

	debugFlags := uint16(0)
	if buildOpts.DumpTokens() {
		debugFlags |= debugcommon.DUMP_TOKENS
	}
	if buildOpts.DumpST() {
		debugFlags |= debugcommon.DUMP_ST
	}
	if buildOpts.TraceRecovery() {
		debugFlags |= debugcommon.DEBUG_ERROR_RECOVERY
	}
	if debugFlags != 0 {
		if opts.logFile != "" {
			logWriter, err := os.Create(opts.logFile)
			if err != nil {
				return buildError(stderr, "error creating log file %s: %w", opts.logFile, err)
			}
			defer func() { _ = logWriter.Close() }()
			debugcommon.InitDebug(debugFlags, logWriter)
		} else {
			debugcommon.InitDebug(debugFlags, stderr)
		}
	}

	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		return buildError(stderr, "invalid project path %q: %w", path, err)
	}

	// A single .bal file is loaded the same way bal run loads one: fsys is
	// rooted at the file's parent directory, and loadPath is just the
	// filename within it. baseDir (the fsys root) doubles as the "project
	// root" for default-output-path purposes below, same as a package
	// directory's own path serves that role.
	baseDir := path
	loadPath := "."
	if !info.IsDir() {
		if filepath.Ext(path) != ".bal" {
			return buildError(stderr, "build requires a package directory or a .bal file; got %q", path)
		}
		baseDir = filepath.Dir(path)
		loadPath = filepath.Base(path)
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return buildError(stderr, "resolve absolute path: %w", err)
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return buildError(stderr, "resolve ballerina env path: %w", err)
	}

	fsys := os.DirFS(absBaseDir)
	result, err := projects.Load(fsys, loadPath, projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		return buildError(stderr, "failed to load package: %w", err)
	}

	if diagResult := result.Diagnostics(); diagResult.HasErrors() || diagResult.HasWarnings() {
		printDiagnostics(fsys, stderr, diagResult, !isTerminal(), diagnostics.NewDiagnosticEnv())
		if diagResult.HasErrors() {
			return buildError(stderr, "package loading reported errors")
		}
	}

	project := result.Project()
	if project.Kind() == projects.ProjectKindWorkspace {
		return buildError(stderr, "provided path %q is a workspace; expected a package directory", path)
	}

	pkg := project.CurrentPackage()
	compilation := pkg.Compilation()
	if cd := compilation.DiagnosticResult(); cd.HasErrors() || cd.HasWarnings() {
		printDiagnostics(fsys, stderr, cd, !isTerminal(), compilation.DiagnosticEnv())
		if cd.HasErrors() {
			return buildError(stderr, "compilation failed; executable not produced")
		}
	}

	if opts.statsOneline {
		_, _ = fmt.Fprint(stderr, compilation.StatsReportOneline())
	} else if opts.stats {
		_, _ = fmt.Fprint(stderr, compilation.StatsReport())
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		return buildError(stderr, "BIR generation failed: no packages produced")
	}

	tyEnv := project.Environment().TypeEnv()

	// --target-os/--target-arch default to the host's own value when unset —
	// the same convention Go's GOOS/GOARCH env vars use, so setting only one
	// dimension does what a user would expect.
	targetPlatform := executable.ResolveTargetPlatform(opts.targetOS, opts.targetArch)

	// Determine output path. The executable suffix follows the target
	// platform, not the host running bal build — cross-compiling for
	// Windows from a non-Windows machine must still produce a ".exe".
	outPath := opts.output
	if outPath == "" {
		pkgName := pkg.PackageName().Value()
		if targetPlatform.OS == "windows" {
			pkgName += ".exe"
		}
		outPath = filepath.Join(absBaseDir, projects.TargetDir, binSubdir, pkgName)
	}

	// Resolve the runner stub to embed the payload into. Packages with no
	// native Go dependencies (the common case) look up the stub bundled
	// alongside bal's own distribution for targetPlatform; no Go toolchain
	// involved. Packages that depend on a native Go bala instead build a
	// custom stub with that native code woven in — see buildNativeStub.
	resolution := pkg.Resolution()
	nativeBalaProjects := findNativeGoBalaProjects(resolution, project.Environment())

	var stubPath string
	if len(nativeBalaProjects) == 0 {
		// RuntimeStubPath (set via -ldflags at bal's own build time, not a
		// bal build flag) overrides the default <dist>/rt/<os>-<arch>
		// lookup, so the predefined layout can change later without
		// breaking a packager who already pins an explicit path.
		distDir, dErr := executable.DistributionDir()
		if dErr != nil {
			return buildError(stderr, "resolve bal distribution directory: %w", dErr)
		}
		sp, rErr := executable.ResolveStub(executable.Key{Platform: targetPlatform}, distDir, RuntimeStubPath)
		if rErr != nil {
			return buildError(stderr, "cannot locate runner stub: %w", rErr)
		}
		stubPath = sp
	} else {
		sp, bErr := buildNativeStub(stderr, absBaseDir, nativeBalaProjects, targetPlatform)
		if bErr != nil {
			return buildError(stderr, "building native interpreter stub: %w", bErr)
		}
		stubPath = sp
	}

	if err := executable.Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		return buildError(stderr, "writing executable: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", outPath)
	return nil
}

// nativeStubName is the intermediate, native-woven stub buildNativeStub
// produces — distinct from both outPath (the final packed executable) and
// the installed "balrt" name, so nothing collides on disk.
const nativeStubName = "balrt-native"

// buildNativeStub builds (or reuses a fingerprint-cached) balrt-shaped stub
// that embeds nativeBalaProjects' native Go sources, for bal build to embed
// the BIR payload into instead of the predefined installed stub. It mirrors
// execWithNativeRunner's (run.go) native-dependency build, but targets
// cli/cmd/balrt — a slim, standalone stub — rather than the full cli/cmd
// CLI, and returns a bare binary path rather than an auto-executing Runner:
// bal build needs to hand that path to executable.Pack, not re-exec into it.
//
// Cross-compiling is supported: LocalExecutor.Build sets GOOS/GOARCH on the
// go build subprocess, the same as the no-native-dep cross-compile path —
// native dependencies package Go sources only (no cgo), so no C
// cross-compiler is ever needed. The cache path is segmented by target
// platform (not just kept flat) so building for two different targets from
// the same project doesn't clobber each other's cached stub/fingerprint.
func buildNativeStub(stderr io.Writer, absBaseDir string, nativeBalaProjects []*projects.BalaProject, targetPlatform executable.Platform) (string, error) {
	stubName := nativeStubName
	if targetPlatform.OS == "windows" {
		stubName += ".exe"
	}
	platformDir := targetPlatform.OS + "-" + targetPlatform.Arch
	outBin := filepath.Join(absBaseDir, projects.TargetDir, binSubdir, "native", platformDir, stubName)

	executor, err := chooseNativeExecutor(outBin, "cli/cmd/balrt")
	if err != nil {
		return "", err
	}

	payloads := make([]nativeexec.NativePayload, 0, len(nativeBalaProjects))
	for _, bp := range nativeBalaProjects {
		goFS, err := bp.NativeGoSourceFS()
		if err != nil {
			return "", fmt.Errorf("reading native Go sources for %s: %w", bp.CurrentPackage().Descriptor().Name().Value(), err)
		}
		desc := bp.CurrentPackage().Descriptor()
		moduleName := desc.Org().Value() + "/" + desc.Name().Value() + "-native"
		payloads = append(payloads, &nativeexec.GoSourcePayload{GoFiles: goFS, Module: moduleName})
	}

	return executor.Build(context.Background(), nativeexec.NativeRunnerRequest{
		Payloads:   payloads,
		Stderr:     stderr,
		TargetOS:   targetPlatform.OS,
		TargetArch: targetPlatform.Arch,
	})
}
