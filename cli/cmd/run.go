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
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	interpsrc "ballerina-lang-go"
	"ballerina-lang-go/bir"
	"ballerina-lang-go/cli/internal/nativeexec"
	"ballerina-lang-go/cli/internal/nativerunner"
	debugcommon "ballerina-lang-go/common"
	_ "ballerina-lang-go/lib/rt"
	"ballerina-lang-go/lib/stdlibs"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"

	"github.com/spf13/cobra"
)

var runOpts struct {
	dumpTokens    bool
	dumpST        bool
	dumpAST       bool
	dumpCFG       bool
	dumpBIR       bool
	traceRecovery bool
	stats         bool
	statsOneline  bool
	logFile       string
	format        string // Output format (dot, etc.)
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
		WithDumpBIR(runOpts.dumpBIR).
		WithDumpCFG(runOpts.dumpCFG).
		WithDumpCFGFormat(projects.ParseCFGFormat(runOpts.format)).
		WithDumpTokens(runOpts.dumpTokens).
		WithDumpST(runOpts.dumpST).
		WithTraceRecovery(runOpts.traceRecovery).
		WithStats(runOpts.stats || runOpts.statsOneline).
		Build()

	if err := profiler.Start(); err != nil {
		profErr := fmt.Errorf("failed to start profiler: %w", err)
		printError(profErr, "", false)
		return profErr
	}
	defer func() { _ = profiler.Stop() }()

	flags := uint16(0)

	if buildOpts.DumpTokens() {
		flags |= debugcommon.DUMP_TOKENS
	}
	if buildOpts.DumpST() {
		flags |= debugcommon.DUMP_ST
	}
	if buildOpts.TraceRecovery() {
		flags |= debugcommon.DEBUG_ERROR_RECOVERY
	}

	if flags != 0 {
		var logWriter *os.File
		var err error
		if runOpts.logFile != "" {
			logWriter, err = os.Create(runOpts.logFile)
			if err != nil {
				cmdErr := fmt.Errorf("error creating log file %s: %w", runOpts.logFile, err)
				printError(cmdErr, "", false)
				return cmdErr
			}
			defer func() { _ = logWriter.Close() }()
			debugcommon.InitDebug(flags, logWriter)
		} else {
			debugcommon.InitDebug(flags, os.Stderr)
		}
	}

	// Default to current directory if no path provided (bal run == bal run .)
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		printRunError(err)
		return err
	}

	baseDir := path
	if !info.IsDir() {
		baseDir = filepath.Dir(path)
		path = filepath.Base(path)
	} else {
		path = "."
	}

	// Detect if path is inside a workspace - if so, load the workspace instead
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		printRunError(err)
		return err
	}
	workspaceRoot := findWorkspaceRootForRun(absBaseDir)

	fsys := os.DirFS(baseDir)
	loadPath := path
	if workspaceRoot != "" {
		// When inside a workspace, load from the workspace root
		fsys = os.DirFS(workspaceRoot)
		loadPath = "."
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		printRunError(err)
		return err
	}
	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)

	result, err := projects.Load(fsys, loadPath, projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFs,
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		printRunError(err)
		return err
	}

	// Check for loading errors
	diagResult := result.Diagnostics()
	if diagResult.HasErrors() {
		// Given we don't have sources at this point it is okay to pass an empty diagnostic env
		printDiagnostics(fsys, os.Stderr, diagResult, !isTerminal(), diagnostics.NewDiagnosticEnv())
		return fmt.Errorf("project loading contains errors")
	}

	project := result.Project()

	// If it's a workspace project, resolve to the specific sub-package
	if project.Kind() == projects.ProjectKindWorkspace {
		workspace, ok := project.(*projects.WorkspaceProject)
		if !ok {
			err := fmt.Errorf("internal error: expected WorkspaceProject")
			printRunError(err)
			return err
		}

		// If user specified the workspace root itself, they can't run the workspace directly
		if workspaceRoot == "" || absBaseDir == workspaceRoot {
			err := fmt.Errorf("cannot run a workspace project directly. Use 'bal run <package-path>' to run a specific package within the workspace")
			printRunError(err)
			return err
		}

		// Find the BuildProject matching the user's path
		buildProject := findBuildProjectByPath(workspace, workspaceRoot, absBaseDir)
		if buildProject == nil {
			err := fmt.Errorf("no package found at path %s within workspace %s", absBaseDir, workspaceRoot)
			printRunError(err)
			return err
		}
		project = buildProject
	}

	pkg := project.CurrentPackage()

	// Detect native Go packages and re-execute via a custom interpreter if needed.
	// Skipped when already running as a native interpreter (BAL_NATIVE=1).
	if !nativeexec.InNativeMode() {
		if err := execWithNativeRunner(pkg, project, absBaseDir); err != nil {
			printRunError(err)
			return err
		}
	}

	// Get package compilation (triggers parsing, type checking, semantic analysis, CFG analysis)
	compilation := pkg.Compilation()

	// Print all diagnostics; only errors abort the run.
	compilationDiags := compilation.DiagnosticResult()
	if compilationDiags.DiagnosticCount() > 0 {
		printDiagnostics(fsys, os.Stderr, compilationDiags, !isTerminal(), compilation.DiagnosticEnv())
	}
	if compilationDiags.HasErrors() {
		return fmt.Errorf("compilation contains errors")
	}

	// Create backend and generate BIR
	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()

	if len(birPkgs) == 0 {
		err := fmt.Errorf("BIR generation failed: no BIR package produced")
		printError(err, "", false)
		return err
	}

	if runOpts.statsOneline {
		fmt.Fprint(os.Stderr, compilation.StatsReportOneline())
	} else if buildOpts.Stats() {
		fmt.Fprint(os.Stderr, compilation.StatsReport())
	}

	// Dump BIR if requested — only include packages belonging to the root package
	// (same org + package name), so all sub-modules are covered while external
	// imports are excluded.
	tyEnv := project.Environment().TypeEnv()
	if buildOpts.DumpBIR() {
		prettyPrinter := bir.PrettyPrinter{}
		tyCtx := semtypes.ContextFrom(tyEnv)
		rootOrgName := pkg.PackageOrg().Value()
		rootPkgName := pkg.PackageName().Value()
		for _, birPkg := range birPkgs {
			if birPkg.PackageID.OrgName.Value() != rootOrgName || birPkg.PackageID.PkgName.Value() != rootPkgName {
				continue
			}
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "==================BEGIN BIR==================")
			fmt.Fprintln(os.Stderr, strings.TrimSpace(prettyPrinter.Print(tyCtx, *birPkg)))
			fmt.Fprintln(os.Stderr, "===================END BIR===================")
		}
	}

	pal, cleanupSignals := palnative.NewPlatform()
	defer cleanupSignals()
	rt := runtime.NewRuntime(pal, tyEnv)
	var initErr error
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			printRuntimeError(err)
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
		return fmt.Errorf("exit: %d", exitCode)
	}
	return nil
}

func printRunError(err error) {
	printError(err, "run [<source-file.bal> | <package-dir> | .]", false)
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

// findWorkspaceRootForRun walks up the directory tree from the given absolute path
// to find a workspace root (a directory with Ballerina.toml containing [workspace]).
// Returns empty string if not inside a workspace.
func findWorkspaceRootForRun(startPath string) string {
	current := startPath
	for {
		tomlPath := filepath.Join(current, projects.BallerinaTomlFile)
		if info, err := os.Stat(tomlPath); err == nil && !info.IsDir() {
			if isWorkspaceToml(tomlPath) {
				return current
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// findBuildProjectByPath finds the BuildProject in a workspace whose source root
// matches the given absolute path. The workspaceAbsRoot is the absolute path to
// the workspace root on the local filesystem.
func findBuildProjectByPath(workspace *projects.WorkspaceProject, workspaceAbsRoot, absPath string) *projects.BuildProject {
	for _, bp := range workspace.Projects() {
		// BuildProject.SourceRoot() is relative to the workspace fs.FS root.
		// Join with the absolute workspace root to get the absolute path.
		bpAbs := filepath.Join(workspaceAbsRoot, bp.SourceRoot())
		if bpAbs == absPath {
			return bp
		}
	}
	return nil
}

// execWithNativeRunner checks whether any resolved dependency has Go-native
// sources. If so, it builds a custom interpreter that embeds those sources and
// re-executes the current command via that binary. On success this function never
// returns — it calls os.Exit after the child process finishes.
func execWithNativeRunner(pkg *projects.Package, project projects.Project, absBaseDir string) error {
	resolution := pkg.Resolution()
	nativeBalaProjects := findNativeGoBalaProjects(resolution, project.Environment())
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

// isEmbeddedPackage reports whether the bala project is present in the
// interpreter's bundled stdlib FS. Embedded packages have their Go native
// code already compiled into the binary via lib/rt and do not require a
// native interpreter rebuild.
func isEmbeddedPackage(bp *projects.BalaProject) bool {
	desc := bp.CurrentPackage().Descriptor()
	return stdlibs.Contains(desc.Org().Value(), desc.Name().Value(), desc.Version().String())
}

// chooseNativeExecutor returns a LocalExecutor targeting targetPackage
// (relative to the interpreter root — e.g. "cli/cmd" for bal run's re-exec
// case, "cli/cmd/balrt" for bal build's slim-stub case) when the Go
// toolchain and interpreter source are available. Returns an error if Go is
// not installed or the interpreter source cannot be located — native
// packages are not supported without a local Go toolchain. Remote build
// support is reserved for WASM.
func chooseNativeExecutor(outBin, targetPackage string) (nativeexec.NativeExecutor, error) {
	root, err := findInterpreterRoot()
	if err != nil {
		return nil, fmt.Errorf("native Go packages require the interpreter source: %w", err)
	}
	local := nativerunner.NewForTarget(root, outBin, targetPackage)
	if !local.Available() {
		return nil, fmt.Errorf("native Go packages require Go %s or later to be installed", nativerunner.MinGoVersion)
	}
	return local, nil
}

// findInterpreterRoot returns the absolute path to the ballerina-lang-go source tree.
// It checks BALLERINA_SRC first, then falls back to the source tree embedded
// in the binary (extracted to a cache directory on first use).
func findInterpreterRoot() (string, error) {
	root, err := locateInterpreterRoot()
	if err != nil {
		return "", err
	}
	// Must be canonical: nativerunner's -overlay matches paths against go
	// build -C's resolved form, so a symlinked root (e.g. os.TempDir() on
	// macOS) silently breaks native package injection otherwise.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolving interpreter root %q: %w", root, err)
	}
	return resolved, nil
}

// locateInterpreterRoot finds the ballerina-lang-go source tree without
// resolving symlinks; see findInterpreterRoot for why that resolution matters.
func locateInterpreterRoot() (string, error) {
	if src := os.Getenv("BALLERINA_SRC"); src != "" {
		return filepath.Abs(src)
	}

	cacheRoot, err := getBallerinaEnvPath()
	if err != nil {
		return "", fmt.Errorf("interpreter source not found; set BALLERINA_SRC to the ballerina-lang-go directory")
	}
	return interpsrc.ExtractTo(cacheRoot, Version)
}
