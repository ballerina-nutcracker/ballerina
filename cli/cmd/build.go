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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ballerina-nutcracker/ballerina/cli/internal/executable"
	"github.com/ballerina-nutcracker/ballerina/cli/internal/nativeexec"
	debugcommon "github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

	"github.com/spf13/cobra"
)

const binSubdir = "bin"

// nativeStubName is the intermediate, native-woven stub buildNativeStub
// produces — distinct from outPath and the installed "balrt" name.
const nativeStubName = "balrt-native"

// RuntimeStubPath overrides the default <dist>/rt/<os>-<arch> runner-stub
// lookup (executable.ResolveStub) when non-empty. Set via -ldflags at bal's
// own build time (like Version in version.go), not a bal build flag.
var RuntimeStubPath = ""

type buildOptions struct {
	dumpTokens       bool
	dumpST           bool
	dumpAST          bool
	dumpRecoveredAST bool
	dumpCFG          bool
	dumpBIR          bool
	traceRecovery    bool
	stats            bool
	statsOneline     bool
	logFile          string
	format           string
	output           string // -o: explicit output path
	targetOS         string // cross-compile target OS; "" defaults to the host OS
	targetArch       string // cross-compile target architecture; "" defaults to the host arch
	targetDir        string
}

var buildCmd = createBuildCmd()

func createBuildCmd() *cobra.Command {
	opts := &buildOptions{}
	cmd := &cobra.Command{
		Use:   "build [<package-dir>]",
		Short: "Compile the current package into a standalone executable",
		Long: `	Compile the current Ballerina package into a standalone executable.

	Compiles the current package or the provided standalone '.bal' file and 
	generates a standalone executable in the <project>/target/bin
	directory.
	Note: Building individual '.bal' files of a package is not allowed.

	Use -o to specify a different output path.

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
	cmd.Flags().BoolVar(&opts.dumpRecoveredAST, "dump-recovered-ast", false, "Dump recovered abstract syntax tree")
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
	cmd.Flags().StringVar(&opts.targetDir, "target-dir", "", "target directory path")
	// Profiler flags are registered onto the global buildCmd from prof_*.go's init().
	// They are intentionally NOT registered inside createBuildCmd, so test-instantiated
	// commands skip profiler flags (the tests don't exercise profiling).
	return cmd
}

func buildError(format string, args ...any) error {
	return usageError("build [<package-dir>]", format, args...)
}

func runBuild(cmd *cobra.Command, args []string, opts *buildOptions) error {
	stderr := cmd.ErrOrStderr()

	// --target-dir is resolved relative to the process cwd (matching
	// pack.go/run.go's own --target-dir), independent of the package-dir arg.
	var targetDirOverride string
	if opts.targetDir != "" {
		abs, err := filepath.Abs(opts.targetDir)
		if err != nil {
			return buildError("resolve target directory: %w", err)
		}
		targetDirOverride = abs
	}

	buildOpts := projects.NewBuildOptionsBuilder().
		WithDumpAST(opts.dumpAST).
		WithDumpRecoveredAST(opts.dumpRecoveredAST).
		WithDumpBIR(opts.dumpBIR).
		WithDumpCFG(opts.dumpCFG).
		WithDumpCFGFormat(projects.ParseCFGFormat(opts.format)).
		WithDumpTokens(opts.dumpTokens).
		WithDumpST(opts.dumpST).
		WithTraceRecovery(opts.traceRecovery).
		WithStats(opts.stats || opts.statsOneline).
		WithTargetDir(targetDirOverride).
		Build()

	// Profiler flags are bound only when prof_*.go's init() registers them
	// on this cmd. In release builds RegisterFlags is a no-op; in debug
	// builds it runs against the global buildCmd. Test-instantiated cmds
	// never carry the flag, so they skip Start.
	if cmd.Flag("prof") != nil {
		if err := profiler.Start(); err != nil {
			return buildError("failed to start profiler: %w", err)
		}
		defer func() { _ = profiler.Stop() }()
	}

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
				return buildError("error creating log file %s: %w", opts.logFile, err)
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
		return buildError("invalid project path %q: %w", path, err)
	}

	// A single .bal file loads like bal run loads one: fsys rooted at the
	// parent dir, loadPath is the filename within it.
	baseDir := path
	loadPath := "."
	if !info.IsDir() {
		if filepath.Ext(path) != ".bal" {
			return buildError("%q is not a package directory or a .bal file", path)
		}
		baseDir = filepath.Dir(path)
		loadPath = filepath.Base(path)
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return buildError("resolve absolute path: %w", err)
	}

	// Detect whether absBaseDir sits inside a workspace without being its
	// root — e.g. cwd is a workspace member's own directory. If so, load
	// from the workspace root instead so sibling member-to-member
	// dependencies resolve, matching bal run's findWorkspaceRoot handling.
	// Only applies to directory builds; a standalone .bal file can't be a
	// workspace member.
	workspaceRoot := ""
	if info.IsDir() {
		workspaceRoot = findWorkspaceRoot(absBaseDir)
	}
	effectiveBaseDir, effectiveLoadPath := absBaseDir, loadPath
	if workspaceRoot != "" && workspaceRoot != absBaseDir {
		effectiveBaseDir, effectiveLoadPath = workspaceRoot, "."
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return buildError("resolve ballerina env path: %w", err)
	}

	fsys := os.DirFS(effectiveBaseDir)
	result, err := projects.Load(fsys, effectiveLoadPath, projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		return buildError("failed to load package: %w", err)
	}

	if diagResult := result.Diagnostics(); diagResult.HasErrors() || diagResult.HasWarnings() {
		printDiagnostics(fsys, stderr, diagResult, !isTerminal(), diagnostics.NewDiagnosticEnv())
		if diagResult.HasErrors() {
			// Diagnostics carry the full failure detail, and this isn't a
			// build-usage mistake, so no USAGE block — but cobra should still
			// print "ballerina: project loading contains errors" as a summary.
			return fmt.Errorf("project loading contains errors")
		}
	}

	project := result.Project()
	if project.Kind() == projects.ProjectKindWorkspace {
		workspace := project.(*projects.WorkspaceProject)

		if workspaceRoot != "" && workspaceRoot != absBaseDir {
			// absBaseDir names one specific member (we walked up to
			// workspaceRoot to load it) — build just that member, matching
			// bal run's disambiguation. -o/--target-dir are fine here: exactly
			// one output.
			memberProject := findBuildProjectByPath(workspace, workspaceRoot, absBaseDir)
			if memberProject == nil {
				return buildError("no package found at path %s within workspace %s", absBaseDir, workspaceRoot)
			}
			return buildOneProject(cmd, opts, stderr, fsys, memberProject, absBaseDir, targetDirOverride)
		}

		// -o names a single explicit output path, and --target-dir a single
		// output directory — neither makes sense when building every member
		// of a workspace to its own executable (all members would collide
		// on the same path).
		if opts.output != "" {
			return buildError("-o cannot be used when building a workspace; run bal build <package-path> to build a single package with a custom output path")
		}
		if opts.targetDir != "" {
			return buildError("--target-dir cannot be used when building a workspace; run bal build <package-path> to build a single package with a custom target directory")
		}

		for _, bp := range workspace.Projects() {
			// SourceRoot() is relative to the workspace root, not the member's
			// own directory (same reconstruction bal run's findBuildProjectByPath uses).
			memberDir := filepath.Join(absBaseDir, bp.SourceRoot())

			// Re-load the whole workspace fresh for each member rather than
			// reusing bp or loading the member's own directory in isolation.
			// Reusing bp would share one CompilerEnvironment across every
			// member — compiling a lang lib twice in the same environment
			// trips the distinct-type-symbol registry's once-per-environment
			// assumption. Loading just the member's own directory avoids
			// that sharing but loses workspaceRepository, breaking
			// member-to-member dependencies. Reloading the whole workspace
			// gives this iteration its own fresh environment (still with
			// sibling resolution intact) while only ever compiling the one
			// member picked out below, so no environment is shared across
			// two actual compiles.
			memberResult, err := projects.Load(fsys, effectiveLoadPath, projects.ProjectLoadConfig{
				BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
				BuildOptions:   &buildOpts,
			})
			if err != nil {
				return buildError("failed to load package: %w", err)
			}

			memberWorkspace := memberResult.Project().(*projects.WorkspaceProject)
			memberProject := findBuildProjectByPath(memberWorkspace, absBaseDir, memberDir)
			if memberProject == nil {
				return buildError("no package found at path %s within workspace %s", memberDir, absBaseDir)
			}

			// Stop at the first member that fails, matching jballerina
			// (which aborts the whole process on a compile error) instead
			// of continuing to later members. targetDirOverride is always ""
			// here since it was rejected above for the all-members case.
			if err := buildOneProject(cmd, opts, stderr, fsys, memberProject, memberDir, targetDirOverride); err != nil {
				return err
			}
		}
		return nil
	}

	return buildOneProject(cmd, opts, stderr, fsys, project, absBaseDir, targetDirOverride)
}

// buildOneProject compiles a single package (a plain build, or one workspace
// member) and packs it into a standalone executable. For a workspace
// member, projectDir is that member's own directory, so output and any
// native-stub cache land under its own target/ unless targetDirOverride is
// set.
func buildOneProject(cmd *cobra.Command, opts *buildOptions, stderr io.Writer, fsys fs.FS, project projects.Project, projectDir, targetDirOverride string) error {
	pkg := project.CurrentPackage()
	compilation := pkg.Compilation()
	if cd := compilation.DiagnosticResult(); cd.HasErrors() || cd.HasWarnings() {
		printDiagnostics(fsys, stderr, cd, !isTerminal(), compilation.DiagnosticEnv())
		if cd.HasErrors() {
			// Not a build-usage mistake, so no USAGE block, but cobra should
			// still print "ballerina: compilation contains errors" as a
			// summary.
			return fmt.Errorf("compilation contains errors")
		}
	}

	if opts.statsOneline {
		_, _ = fmt.Fprint(stderr, compilation.StatsReportOneline())
	} else if opts.stats {
		_, _ = fmt.Fprint(stderr, compilation.StatsReport())
	}

	backend := projects.NewBallerinaBackend(compilation)
	backendDiags := backend.DiagnosticResult()
	if backendDiags.HasErrors() {
		printDiagnostics(fsys, stderr, backendDiags, !isTerminal(), compilation.DiagnosticEnv())
		return buildError("BIR generation failed; executable not produced")
	}
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		return buildError("BIR generation failed: no BIR package produced")
	}

	tyEnv := project.Environment().TypeEnv()

	// Unset --target-os/--target-arch default to the host, like Go's GOOS/GOARCH.
	targetPlatform := executable.ResolveTargetPlatform(opts.targetOS, opts.targetArch)

	targetDir := targetDirOverride
	if targetDir == "" {
		targetDir = filepath.Join(projectDir, projects.TargetDir)
	}

	// Suffix follows the target platform, not the host running bal build.
	outPath := opts.output
	if outPath == "" {
		pkgName := pkg.PackageName().Value()
		if targetPlatform.OS == "windows" {
			pkgName += ".exe"
		}
		outPath = filepath.Join(targetDir, binSubdir, pkgName)
	}

	// Packages with no native Go deps use the bundled stub for targetPlatform;
	// packages with a native Go bala build a custom stub (buildNativeStub).
	resolution := pkg.Resolution()
	nativeBalaProjects := findNativeGoBalaProjects(resolution, project.Environment())

	var stubPath string
	if len(nativeBalaProjects) == 0 {
		// RuntimeStubPath is a host-platform binary; reject it when cross-compiling.
		if RuntimeStubPath != "" {
			if host := executable.HostPlatform(); targetPlatform != host {
				return buildError(
					"runtime stub override cannot be used when cross-compiling to %s/%s",
					targetPlatform.OS, targetPlatform.Arch)
			}
		}
		distDir, dErr := executable.DistributionDir()
		if dErr != nil {
			return buildError("resolve bal distribution directory: %w", dErr)
		}
		sp, rErr := executable.ResolveStub(targetPlatform, distDir, RuntimeStubPath)
		if rErr != nil {
			return buildError("%w", rErr)
		}
		stubPath = sp
	} else {
		sp, bErr := buildNativeStub(stderr, targetDir, nativeBalaProjects, targetPlatform)
		if bErr != nil {
			return buildError("building native interpreter stub: %w", bErr)
		}
		stubPath = sp
	}

	if err := executable.Pack(stubPath, birPkgs, tyEnv, outPath); err != nil {
		return buildError("write executable: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", outPath)
	return nil
}

// buildNativeStub builds (or reuses a fingerprint-cached) balrt-shaped stub
// embedding nativeBalaProjects' native Go sources, for bal build to pack
// instead of the predefined stub. Like execWithNativeRunner (run.go), but
// targets the slim cli/internal/balrt stub and returns a bare binary path rather
// than an auto-executing Runner.
//
// Cross-compiling works the same way (LocalExecutor.Build sets GOOS/GOARCH);
// the cache path is segmented by target platform so two targets don't
// clobber each other's cached stub.
func buildNativeStub(stderr io.Writer, targetDir string, nativeBalaProjects []*projects.BalaProject, targetPlatform executable.Platform) (string, error) {
	if err := executable.ValidatePlatform(targetPlatform); err != nil {
		return "", err
	}

	stubName := nativeStubName
	if targetPlatform.OS == "windows" {
		stubName += ".exe"
	}
	platformDir := targetPlatform.OS + "-" + targetPlatform.Arch
	outBin := filepath.Join(targetDir, binSubdir, "native", platformDir, stubName)

	executor, err := chooseNativeExecutor(outBin, "cli/internal/balrt")
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
