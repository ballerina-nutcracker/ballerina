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
	"io/fs"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/cli/internal/nativeexec"
	"github.com/ballerina-nutcracker/ballerina/cli/internal/nativerunner"
	_ "github.com/ballerina-nutcracker/ballerina/lib/rt"
	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/projects"

	"github.com/spf13/cobra"
)

var runOpts struct {
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
	format           string // Output format (dot, etc.)
}

var runCmd = &cobra.Command{
	Use:   "run [<source-file.bal> | <package-dir> | .]",
	Short: "Build and run the current package or a Ballerina source file",
	Long: `	Build the current package and run it.

	The 'run' command builds and executes the given Ballerina package or
	a source file.

	A Ballerina program consists of one or more modules; one of these modules
	is distinguished as the root module, which is the default module of
	current package.

	Ballerina program execution consists of two consecutive phases.
	The initialization phase initializes all modules of a program one after
	another. If a module defines a function named 'init()', it will be
	invoked during this phase. If the root module of the program defines a
	public function named 'main()', then it will be invoked.

	If the initialization phase of program execution completes successfully,
	then execution proceeds to the listening phase. If there are no module
	listeners, then the listening phase immediately terminates successfully.
	Otherwise, the listening phase initializes the module listeners.

	A service declaration is the syntactic sugar for creating a service object
	and attaching it to the module listener specified in the service
	declaration.

	Note: Running individual '.bal' files of a package is not allowed.`,
	Args: validateSourceFile,
	RunE: runBallerina,
}

func init() {
	runCmd.Flags().BoolVar(&runOpts.dumpTokens, "dump-tokens", false, "Dump lexer tokens")
	runCmd.Flags().BoolVar(&runOpts.dumpST, "dump-st", false, "Dump syntax tree")
	runCmd.Flags().BoolVar(&runOpts.dumpAST, "dump-ast", false, "Dump abstract syntax tree")
	runCmd.Flags().BoolVar(&runOpts.dumpRecoveredAST, "dump-recovered-ast", false, "Dump recovered abstract syntax tree")
	runCmd.Flags().BoolVar(&runOpts.dumpCFG, "dump-cfg", false, "Dump control flow graph")
	runCmd.Flags().BoolVar(&runOpts.dumpBIR, "dump-bir", false, "Dump Ballerina Intermediate Representation")
	runCmd.Flags().BoolVar(&runOpts.traceRecovery, "trace-recovery", false, "Enable error recovery tracing")
	runCmd.Flags().BoolVar(&runOpts.stats, "stats", false, "Print per-stage compilation timing statistics")
	runCmd.Flags().BoolVar(&runOpts.statsOneline, "stats-oneline", false, "Print per-stage compilation timing totals only")
	runCmd.Flags().StringVar(&runOpts.logFile, "log-file", "", "Write debug output to specified file")
	runCmd.Flags().StringVar(&runOpts.format, "format", "", "Output format for dump operations (dot)")
	profiler.RegisterFlags(runCmd)
}

func runBallerina(cmd *cobra.Command, args []string) error {
	// Build options from CLI flags. Constructed before debug setup so
	// buildOpts can be the single source of truth for all flag reads.
	buildOpts := projects.NewBuildOptionsBuilder().
		WithDumpAST(runOpts.dumpAST).
		WithDumpRecoveredAST(runOpts.dumpRecoveredAST).
		WithDumpBIR(runOpts.dumpBIR).
		WithDumpCFG(runOpts.dumpCFG).
		WithDumpCFGFormat(projects.ParseCFGFormat(runOpts.format)).
		WithDumpTokens(runOpts.dumpTokens).
		WithDumpST(runOpts.dumpST).
		WithTraceRecovery(runOpts.traceRecovery).
		WithStats(runOpts.stats || runOpts.statsOneline).
		Build()

	if err := profiler.Start(); err != nil {
		// Profiler setup, not a run-usage mistake, so no USAGE block.
		return fmt.Errorf("failed to start profiler: %w", err)
	}
	defer func() { _ = profiler.Stop() }()

	// Default to current directory if no path provided (bal run == bal run .)
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		return runError("%w", err)
	}

	baseDir := path
	if !info.IsDir() {
		baseDir = filepath.Dir(path)
		path = filepath.Base(path)
	} else {
		path = "."
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return runError("%w", err)
	}
	workspaceRoot := findWorkspaceRoot(absBaseDir)
	packageDirName := filepath.Base(absBaseDir)
	fsys := fs.FS(os.DirFS(baseDir))
	if workspaceRoot != "" {
		relative, relErr := filepath.Rel(workspaceRoot, absBaseDir)
		if relErr != nil {
			return runError("%w", relErr)
		}
		path = filepath.ToSlash(relative)
		fsys = os.DirFS(workspaceRoot)
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return runError("%w", err)
	}
	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)

	parserDebug := parser.DebugOptions{DumpTokens: buildOpts.DumpTokens(), DumpSyntaxTree: buildOpts.DumpST(), TraceRecovery: buildOpts.TraceRecovery()}
	parserWriter := os.Stderr
	var logWriter *os.File
	if runOpts.logFile != "" && (parserDebug.DumpTokens || parserDebug.DumpSyntaxTree || parserDebug.TraceRecovery) {
		logWriter, err = os.Create(runOpts.logFile)
		if err != nil {
			return fmt.Errorf("error creating log file %s: %w", runOpts.logFile, err)
		}
		defer func() { _ = logWriter.Close() }()
		parserWriter = logWriter
	}
	platform, cleanupSignals := palnative.NewPlatform()
	defer cleanupSignals()
	compilation, err := compileRunInput(cmd.Context(), fsys, path, packageDirName, ballerinaEnvFs,
		buildOpts, parserDebug, parserWriter, absBaseDir, workspaceRoot, platform)
	if err != nil {
		return runError("%w", err)
	}
	compilationDiags := projects.NewDiagnosticResult(compilation.context.Diagnostics())
	if compilationDiags.DiagnosticCount() > 0 {
		printDiagnostics(fsys, os.Stderr, compilationDiags, !isTerminal(), compilation.context.DiagnosticEnv())
	}
	if compilationDiags.HasErrors() {
		if compilation.loadFailed {
			return fmt.Errorf("project loading contains errors")
		}
		return fmt.Errorf("compilation contains errors")
	}
	birPkgs := compilation.birPackages
	if len(birPkgs) == 0 {
		return fmt.Errorf("BIR generation failed: no BIR package produced")
	}

	if runOpts.statsOneline {
		fmt.Fprint(os.Stderr, projects.FormatStatsReportOneline(compilation.context.ModuleStats()))
	} else if buildOpts.Stats() {
		fmt.Fprint(os.Stderr, projects.FormatStatsReport(compilation.context.ModuleStats()))
	}

	rt := compilation.runtime
	var initErr error
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			// Runtime errors carry their own multi-line stack-trace-like
			// format; cobra's "ballerina:" prefix and run's USAGE block
			// would look out of place against the rest of the trace. Print
			// verbatim and silence cobra's default error print.
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
			cmd.SilenceErrors = true
			initErr = err
			break
		}
	}
	rt.Listen()
	if initErr != nil {
		return initErr
	}
	exitCode := <-rt.ExitStatus
	if exitCode != 0 {
		// The executed program's own exit code, not a run-usage mistake, so
		// this must stay a bare error — no USAGE block.
		return fmt.Errorf("exit: %d", exitCode)
	}
	return nil
}

func runError(format string, args ...any) error {
	return usageError("run [<source-file.bal> | <package-dir> | .]", format, args...)
}

func getBallerinaEnvPath() (string, error) {
	if balEnv := os.Getenv(projects.BallerinaEnvVar); balEnv != "" {
		return balEnv, nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(userHome, projects.UserHomeDirName), nil
}

// execWithNativeRunner builds a custom interpreter embedding any native Go
// dependencies and re-execs into it. On success it never returns (os.Exit).
func execWithNativeProjects(candidates []*projects.BalaProject, absBaseDir string) error {
	nativeBalaProjects := make([]*projects.BalaProject, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate.Platform(), "go") && !isEmbeddedPackage(candidate) {
			nativeBalaProjects = append(nativeBalaProjects, candidate)
		}
	}
	if len(nativeBalaProjects) == 0 {
		return nil
	}

	outBin := filepath.Join(absBaseDir, "target", "bin", "bal")
	if goruntime.GOOS == "windows" {
		outBin += ".exe"
	}
	executor, err := chooseNativeExecutor(outBin, "cli/cmd")
	if err != nil {
		return err
	}

	payloads := make([]nativeexec.NativePayload, 0, len(nativeBalaProjects))
	for _, bp := range nativeBalaProjects {
		goFS, err := bp.NativeGoSourceFS()
		if err != nil {
			return fmt.Errorf("reading native Go sources for %s: %w", bp.CurrentPackage().Descriptor().Name().Value(), err)
		}
		desc := bp.CurrentPackage().Descriptor()
		moduleName := desc.Org().Value() + "/" + desc.Name().Value() + "-native"
		payloads = append(payloads, &nativeexec.GoSourcePayload{GoFiles: goFS, Module: moduleName})
	}

	req := nativeexec.NativeRunnerRequest{
		Payloads: payloads,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Args:     os.Args[1:],
		Env:      nativeexec.AppendNativeMode(os.Environ()),
	}

	runner, err := executor.Prepare(context.Background(), req)
	if err != nil {
		return fmt.Errorf("building native interpreter: %w", err)
	}
	defer func() { _ = runner.Close() }()

	code, err := runner.Run(context.Background())
	if err != nil {
		return fmt.Errorf("running native interpreter: %w", err)
	}
	os.Exit(int(code))
	return nil // unreachable
}

// findNativeGoBalaProjects returns all resolved bala packages in resolution that
// have a go-prefixed platform (e.g. go1.26) and are not bundled in the embedded stdlib.
func findNativeGoBalaProjects(resolution *projects.PackageResolution, env *projects.Environment) []*projects.BalaProject {
	var result []*projects.BalaProject
	cache := env.PackageCache()
	for _, pkgDesc := range resolution.DependencyGraph().ToTopologicallySortedList() {
		dep := cache.Get(pkgDesc.Org().Value(), pkgDesc.Name().Value(), pkgDesc.Version().String())
		if dep == nil {
			continue
		}
		bp, ok := dep.Project().(*projects.BalaProject)
		if ok && strings.HasPrefix(bp.Platform(), "go") && !isEmbeddedPackage(bp) {
			result = append(result, bp)
		}
	}
	return result
}

// isEmbeddedPackage reports whether bp is in the bundled stdlib FS — its
// native code is already compiled in via lib/rt, so no rebuild is needed.
func isEmbeddedPackage(bp *projects.BalaProject) bool {
	desc := bp.CurrentPackage().Descriptor()
	return stdlibs.Contains(desc.Org().Value(), desc.Name().Value(), desc.Version().String())
}

// chooseNativeExecutor returns a LocalExecutor targeting targetPackage
// (e.g. "cli/cmd" for run's re-exec, "cli/internal/balrt" for build's slim stub),
// erroring if Go isn't installed or the CLI driver source can't be found.
func chooseNativeExecutor(outBin, targetPackage string) (nativeexec.NativeExecutor, error) {
	root, err := findInterpreterRoot()
	if err != nil {
		return nil, fmt.Errorf("native Go packages require the CLI driver source: %w", err)
	}
	local := nativerunner.NewForTarget(root, outBin, targetPackage)
	if !local.Available() {
		return nil, fmt.Errorf("native Go packages require Go %s or later to be installed", nativerunner.MinGoVersion)
	}
	return local, nil
}

// findInterpreterRoot returns the absolute path to a native-build driver workspace.
// It checks BALLERINA_SRC first, then falls back to the CLI source embedded in
// the binary and extracted to a cache directory on first use.
func findInterpreterRoot() (string, error) {
	root, err := locateInterpreterRoot()
	if err != nil {
		return "", err
	}
	// Must be canonical: a symlinked root (e.g. macOS os.TempDir()) silently
	// breaks nativerunner's -overlay path matching otherwise.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolving interpreter root %q: %w", root, err)
	}
	return resolved, nil
}

// locateInterpreterRoot finds the ballerina source tree without
// resolving symlinks; see findInterpreterRoot for why that resolution matters.
func locateInterpreterRoot() (string, error) {
	if src := os.Getenv("BALLERINA_SRC"); src != "" {
		return filepath.Abs(src)
	}

	cacheRoot, err := getBallerinaEnvPath()
	if err != nil {
		return "", fmt.Errorf("CLI driver source not found; set BALLERINA_SRC to the ballerina directory")
	}
	return extractDriverSource(cacheRoot, Version)
}
