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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/cli/internal/nativeexec"
	"github.com/ballerina-nutcracker/ballerina/cli/internal/testerina"
	debugcommon "github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/compilerplugin"
	_ "github.com/ballerina-nutcracker/ballerina/lib/rt"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

	"github.com/spf13/cobra"
)

type testOptions struct {
	tests            string
	list             bool
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
}

var testCmd = createTestCmd()

func createTestCmd() *cobra.Command {
	opts := &testOptions{}
	cmd := &cobra.Command{
		Use:   "test [<package-dir> | .]",
		Short: "Compile the current package and run its tests",
		Long: `	Compile the current Ballerina package and run its tests.

	Every function annotated with '@test:Config' in the current package is
	executed against a live runtime, in module dependency order and in source
	order within each module.

	As in 'bal run', module initialization runs first: each module's 'init()'
	is invoked and the root module's 'main()' runs if it declares one.

	Use --tests to run a subset of the tests and --list to print the tests
	that would run without running them.

	Note: Testing individual '.bal' files of a package is not allowed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(cmd, args, opts)
		},
	}
	cmd.Flags().StringVar(&opts.tests, "tests", "", "Comma-separated test filters ([<module>:]<name>, '*' wildcard)")
	cmd.Flags().BoolVar(&opts.list, "list", false, "List the tests that would run and exit")
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
	return cmd
}

func testError(format string, args ...any) error {
	return usageError("test [<package-dir> | .]", format, args...)
}

func runTests(cmd *cobra.Command, args []string, opts *testOptions) error {
	stderr := cmd.ErrOrStderr()

	filter, err := testerina.ParseFilter(opts.tests)
	if err != nil {
		return testError("%w", err)
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
				return testError("error creating log file %s: %w", opts.logFile, err)
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
		return testError("%w", err)
	}
	if !info.IsDir() {
		return testError("cannot test a single source file; 'bal test' requires a package")
	}
	absBaseDir, err := filepath.Abs(path)
	if err != nil {
		return testError("%w", err)
	}

	workspaceRoot := findWorkspaceRoot(absBaseDir)
	fsys := os.DirFS(path)
	loadPath := "."
	if workspaceRoot != "" {
		fsys = os.DirFS(workspaceRoot)
	}

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return testError("%w", err)
	}

	collector := testerina.NewCollector()
	result, err := projects.Load(fsys, loadPath, projects.ProjectLoadConfig{
		BallerinaEnvFs:  os.DirFS(ballerinaEnvPath),
		BuildOptions:    &buildOpts,
		CompilerPlugins: []compilerplugin.InjectedPlugin{collector.Plugin()},
	})
	if err != nil {
		return testError("%w", err)
	}
	if result.Diagnostics().HasErrors() {
		printDiagnostics(fsys, stderr, result.Diagnostics(), !isTerminal(), diagnostics.NewDiagnosticEnv())
		return fmt.Errorf("project loading contains errors")
	}

	project, err := testProject(result.Project(), workspaceRoot, absBaseDir)
	if err != nil {
		return err
	}
	pkg := project.CurrentPackage()

	if !nativeexec.InNativeMode() {
		if err := execWithNativeRunner(pkg, project, absBaseDir); err != nil {
			return testError("%w", err)
		}
	}

	birPkgs, err := compilePackageForTests(cmd, fsys, pkg, project.Environment().TypeEnv(), opts)
	if err != nil {
		return err
	}

	schedule := testerina.NewSchedule(collector.Plan(rootModuleOrder(pkg)), filter)
	if opts.list {
		printTestList(cmd.OutOrStdout(), schedule)
		return nil
	}
	return executeSchedule(cmd, project.Environment(), birPkgs, schedule)
}

// testProject narrows a loaded workspace to the member package under test.
func testProject(project projects.Project, workspaceRoot, absBaseDir string) (projects.Project, error) {
	if project.Kind() != projects.ProjectKindWorkspace {
		return project, nil
	}
	workspace, ok := project.(*projects.WorkspaceProject)
	if !ok {
		return nil, testError("internal error: expected WorkspaceProject")
	}
	if workspaceRoot == "" || absBaseDir == workspaceRoot {
		return nil, testError(
			"cannot test a workspace project directly. Use 'bal test <package-path>' to test a specific package within the workspace")
	}
	buildProject := findBuildProjectByPath(workspace, workspaceRoot, absBaseDir)
	if buildProject == nil {
		return nil, testError("no package found at path %s within workspace %s", absBaseDir, workspaceRoot)
	}
	return buildProject, nil
}

func compilePackageForTests(
	cmd *cobra.Command, fsys fs.FS, pkg *projects.Package, tyEnv semtypes.Env, opts *testOptions,
) ([]*bir.BIRPackage, error) {
	stderr := cmd.ErrOrStderr()
	compilation := pkg.Compilation()
	compilationDiags := compilation.DiagnosticResult()
	if compilationDiags.DiagnosticCount() > 0 {
		printDiagnostics(fsys, stderr, compilationDiags, !isTerminal(), compilation.DiagnosticEnv())
	}
	if compilationDiags.HasErrors() {
		return nil, fmt.Errorf("compilation contains errors")
	}

	backend := projects.NewBallerinaBackend(compilation)
	if backend.DiagnosticResult().HasErrors() {
		printDiagnostics(fsys, stderr, backend.DiagnosticResult(), !isTerminal(), compilation.DiagnosticEnv())
		return nil, fmt.Errorf("BIR generation failed")
	}
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		return nil, fmt.Errorf("BIR generation failed: no BIR package produced")
	}

	if opts.statsOneline {
		_, _ = fmt.Fprint(stderr, compilation.StatsReportOneline())
	} else if opts.stats {
		_, _ = fmt.Fprint(stderr, compilation.StatsReport())
	}
	if opts.dumpBIR {
		dumpRootPackageBIR(stderr, pkg, tyEnv, birPkgs)
	}
	return birPkgs, nil
}

// dumpRootPackageBIR prints the BIR of the root package's own modules, as run
// does; dependency packages are left out.
func dumpRootPackageBIR(w io.Writer, pkg *projects.Package, tyEnv semtypes.Env, birPkgs []*bir.BIRPackage) {
	prettyPrinter := bir.PrettyPrinter{}
	tyCtx := semtypes.ContextFrom(tyEnv)
	rootOrgName := pkg.PackageOrg().Value()
	rootPkgName := pkg.PackageName().Value()
	for _, birPkg := range birPkgs {
		if birPkg.PackageID.OrgName.Value() != rootOrgName || birPkg.PackageID.PkgName.Value() != rootPkgName {
			continue
		}
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "==================BEGIN BIR==================")
		_, _ = fmt.Fprintln(w, strings.TrimSpace(prettyPrinter.Print(tyCtx, *birPkg)))
		_, _ = fmt.Fprintln(w, "===================END BIR===================")
	}
}

// rootModuleOrder returns the root package's modules in topological order.
func rootModuleOrder(pkg *projects.Package) []testerina.ModuleKey {
	rootDescriptor := pkg.Descriptor()
	var order []testerina.ModuleKey
	for _, module := range pkg.Compilation().Resolution().ModuleDependencyGraph().ToTopologicallySortedList() {
		if !module.PackageDescriptor().Equals(rootDescriptor) {
			continue
		}
		order = append(order, testerina.ModuleKey{
			Org: module.Org().Value(), Module: module.Name().String(),
		})
	}
	return order
}

func printTestList(w io.Writer, schedule testerina.Schedule) {
	for _, step := range schedule.Steps {
		_, _ = fmt.Fprint(w, step.Label())
		if !step.Test.Enabled {
			_, _ = fmt.Fprint(w, " (disabled)")
		}
		_, _ = fmt.Fprintln(w)
	}
}

// executeSchedule initializes the runtime with the test PAL overrides installed
// and runs the schedule against it.
func executeSchedule(
	cmd *cobra.Command, env *projects.Environment, birPkgs []*bir.BIRPackage, schedule testerina.Schedule,
) error {
	platform, cleanupSignals := palnative.NewPlatform()
	defer cleanupSignals()

	// The runner must exist before NewRuntime: extern.InitEnv copies
	// pal.Platform by value, so every override has to be in place first.
	runner := testerina.NewRunner(platform, platform.Signals.Signals, env.TypeEnv())
	rt := runtime.NewRuntime(runner.Platform(), env.TypeEnv())

	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			// Runtime errors carry their own stack-trace-like format; printing
			// them verbatim keeps them readable.
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
			cmd.SilenceErrors = true
			rt.Listen()
			runner.Shutdown(rt)
			return err
		}
	}
	rt.Listen()
	runner.Start(rt)

	reporter := testerina.NewPlainReporter(cmd.OutOrStdout())
	if isTerminal() {
		reporter = testerina.NewTTYReporter(cmd.OutOrStdout())
	}
	summary := runner.Run(schedule, reporter)
	// Graceful-stop handlers only run inside Shutdown, so a failure they report
	// lands after Run's own snapshot.
	runner.Shutdown(rt)
	summary.RunFailures = runner.RunFailures()
	reporter.End(summary)

	if summary.Failed() {
		cmd.SilenceErrors = true
		return fmt.Errorf("tests failed")
	}
	return nil
}
