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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/cli/internal/testfilter"
	debugcommon "github.com/ballerina-nutcracker/ballerina/common"
	_ "github.com/ballerina-nutcracker/ballerina/lib/rt"
	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/test/0.0.1/go1.27/native"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/testplan"
	"github.com/ballerina-nutcracker/ballerina/testrunner"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

	"github.com/spf13/cobra"
)

// invalidRerunDataMessage is the user-facing wording for an unusable
// --rerun-failed rerun_test.json, matching jballerina byte-for-byte. Lives
// here rather than in testfilter.ErrInvalidRerunData: a Go error string
// carrying this trailing period would violate staticcheck's ST1005.
const invalidRerunDataMessage = "error while running failed tests : Invalid failed test data. Please run `bal test` command."

type testCmdOptions struct {
	groups        string
	disableGroups string
	tests         string
	listGroups    bool
	testReport    bool
	rerunFailed   bool
	codeCoverage  bool
	graalvm       bool
	graalvmOpts   string
	cloud         string
	parallel      bool
	offline       bool
	sticky        bool
	targetDir     string

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
}

var testCmd = createTestCmd()

func createTestCmd() *cobra.Command {
	opts := &testCmdOptions{}
	cmd := &cobra.Command{
		Use:   "test [<source-file.bal> | <package-dir>]",
		Short: "Run package tests",
		Long: `	Run the tests of the current package, a standalone '.bal' file, or every
	member package of a workspace.

	Discovers '@test:*'-annotated functions in the package's test sources
	(or, for a standalone file, in the file itself), registers them with
	ballerina/test, and executes them in-process, printing a pass/fail/skip
	summary. When run at a workspace root, every member package's tests run
	in turn; when run inside one member's own directory, only that member's
	tests run.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(cmd, args, opts)
		},
	}
	cmd.Flags().StringVar(&opts.groups, "groups", "", "Test groups to be executed")
	cmd.Flags().StringVar(&opts.disableGroups, "disable-groups", "", "Test groups to be disabled")
	cmd.Flags().StringVar(&opts.tests, "tests", "", "Test functions to be executed")
	cmd.Flags().BoolVar(&opts.listGroups, "list-groups", false, "List the groups available in the tests")
	cmd.Flags().BoolVar(&opts.testReport, "test-report", false, "Enable test report generation")
	cmd.Flags().BoolVar(&opts.rerunFailed, "rerun-failed", false, "Rerun failed tests")
	cmd.Flags().BoolVar(&opts.codeCoverage, "code-coverage", false, "Enable code coverage (not supported)")
	cmd.Flags().BoolVar(&opts.graalvm, "graalvm", false, "Run test suite against native image (not supported)")
	cmd.Flags().StringVar(&opts.graalvmOpts, "graalvm-build-options", "", "Native image build options (not supported)")
	cmd.Flags().StringVar(&opts.cloud, "cloud", "", "Enable cloud artifact generation (not supported)")
	cmd.Flags().BoolVar(&opts.parallel, "parallel", false, "Enable parallel execution of tests (not supported, falls back to serial)")
	cmd.Flags().BoolVar(&opts.offline, "offline", false, "Resolve dependencies offline")
	cmd.Flags().BoolVar(&opts.sticky, "sticky", false, "Stick to exact versions locked (if exists)")
	cmd.Flags().StringVar(&opts.targetDir, "target-dir", "", "Target directory path")
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
	return cmd
}

func testError(format string, args ...any) error {
	return usageError("test [<source-file.bal> | <package-dir>]", format, args...)
}

func runTest(cmd *cobra.Command, args []string, opts *testCmdOptions) error {
	stderr := cmd.ErrOrStderr()

	// Unsupported-scope flags: fail clearly rather than silently misbehaving
	// or crashing (see TODO.md's "Deferred bal test features").
	if opts.graalvm || opts.graalvmOpts != "" {
		return testError("--graalvm is not supported: this interpreter has no separate native-image step")
	}
	if opts.cloud != "" {
		return testError("--cloud is not supported")
	}
	if opts.codeCoverage {
		_, _ = fmt.Fprintln(stderr, "warning: --code-coverage is not supported yet; running without coverage")
	}
	if opts.parallel {
		_, _ = fmt.Fprintln(stderr, "warning: --parallel is not supported yet; running tests serially")
	}

	buildOpts := projects.NewBuildOptionsBuilder().
		WithSkipTests(false).
		WithOffline(opts.offline).
		WithSticky(opts.sticky).
		WithTargetDir(opts.targetDir).
		WithTestReport(opts.testReport).
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
		return testError("invalid project path %q: %w", path, err)
	}

	// A single .bal file loads like bal run/bal build load one: fsys rooted
	// at the parent dir, loadPath is the filename within it. A standalone
	// file can't be a workspace member, so workspace detection only applies
	// to directories (matches build.go).
	baseDir := path
	loadPath := "."
	if !info.IsDir() {
		if filepath.Ext(path) != ".bal" {
			return testError("%q is not a package directory or a .bal file", path)
		}
		baseDir = filepath.Dir(path)
		loadPath = filepath.Base(path)
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return testError("resolve absolute path: %w", err)
	}

	// Detect whether absBaseDir sits inside a workspace without being its
	// root — e.g. cwd is a workspace member's own directory. If so, load
	// from the workspace root instead so sibling member-to-member
	// dependencies resolve, matching bal build's findWorkspaceRoot handling.
	// Only applies to directories; a standalone .bal file can't be a
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
		return testError("resolve ballerina env path: %w", err)
	}

	fsys := os.DirFS(effectiveBaseDir)
	result, err := projects.Load(fsys, effectiveLoadPath, projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		return testError("failed to load package: %w", err)
	}
	if diagResult := result.Diagnostics(); diagResult.HasErrors() || diagResult.HasWarnings() {
		printDiagnostics(fsys, stderr, diagResult, !isTerminal(), diagnostics.NewDiagnosticEnv())
		if diagResult.HasErrors() {
			return testError("package loading reported errors")
		}
	}

	project := result.Project()
	if project.Kind() == projects.ProjectKindWorkspace {
		workspace := project.(*projects.WorkspaceProject)

		if workspaceRoot != "" && workspaceRoot != absBaseDir {
			// absBaseDir names one specific member (we walked up to
			// workspaceRoot to load it) — test just that member, matching
			// bal run's disambiguation.
			memberProject := findBuildProjectByPath(workspace, workspaceRoot, absBaseDir)
			if memberProject == nil {
				return testError("no package found at path %s within workspace %s", absBaseDir, workspaceRoot)
			}
			exitCode, pkgResult, err := runTestsForProject(cmd, opts, stderr, fsys, memberProject, absBaseDir)
			return finishStandaloneTestRun(cmd, absBaseDir, exitCode, pkgResult, err)
		}

		return runTestsForWorkspace(cmd, opts, stderr, fsys, effectiveLoadPath, workspace, absBaseDir, ballerinaEnvPath, &buildOpts)
	}

	exitCode, pkgResult, err := runTestsForProject(cmd, opts, stderr, fsys, project, absBaseDir)
	return finishStandaloneTestRun(cmd, absBaseDir, exitCode, pkgResult, err)
}

// finishStandaloneTestRun writes the package's test_results.json if
// requested, then translates a non-zero suite exit code into the usual
// "there are test failures" command error.
func finishStandaloneTestRun(cmd *cobra.Command, projectDir string, exitCode int, pkgResult *packageTestResult, err error) error {
	if err != nil {
		return err
	}
	if pkgResult != nil {
		if reportErr := writeTestResultsReport(cmd, projectDir, *pkgResult); reportErr != nil {
			return reportErr
		}
	}
	if exitCode != 0 {
		// The test suite's own failure, not a test-usage mistake — no USAGE block.
		cmd.SilenceUsage = true
		return fmt.Errorf("there are test failures")
	}
	return nil
}

// runTestsForWorkspace runs every member's tests in turn (a failing member
// doesn't stop the rest), each via a freshly-reloaded workspace so members
// get independent CompilerEnvironments. Exit reflects whether any member failed.
func runTestsForWorkspace(cmd *cobra.Command, opts *testCmdOptions, stderr io.Writer, fsys fs.FS,
	loadPath string, workspace *projects.WorkspaceProject, workspaceAbsRoot, ballerinaEnvPath string,
	buildOpts *projects.BuildOptions) error {
	anyFailed := false
	var pkgResults []packageTestResult
	for _, bp := range workspace.Projects() {
		memberDir := filepath.Join(workspaceAbsRoot, bp.SourceRoot())

		memberResult, err := projects.Load(fsys, loadPath, projects.ProjectLoadConfig{
			BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
			BuildOptions:   buildOpts,
		})
		if err != nil {
			return testError("failed to load package: %w", err)
		}

		memberWorkspace := memberResult.Project().(*projects.WorkspaceProject)
		memberProject := findBuildProjectByPath(memberWorkspace, workspaceAbsRoot, memberDir)
		if memberProject == nil {
			return testError("no package found at path %s within workspace %s", memberDir, workspaceAbsRoot)
		}

		pkgName := memberProject.CurrentPackage().PackageName().Value()
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nRunning tests for package '%s'\n", pkgName)

		exitCode, pkgResult, err := runTestsForProject(cmd, opts, stderr, fsys, memberProject, memberDir)
		if err != nil {
			// A genuine load/compile/internal error, not just "some tests
			// failed" — matches jballerina's re-throw for non-test-failure
			// exceptions instead of continuing to the next member.
			return err
		}
		if exitCode != 0 {
			anyFailed = true
		}
		if pkgResult != nil {
			pkgResults = append(pkgResults, *pkgResult)
		}
	}

	if len(pkgResults) > 0 {
		report := workspaceTestReport{WorkspaceName: filepath.Base(workspaceAbsRoot), Packages: pkgResults}
		for _, pkg := range pkgResults {
			report.TotalTests += pkg.TotalTests
			report.Passed += pkg.Passed
			report.Failed += pkg.Failed
			report.Skipped += pkg.Skipped
		}
		if err := writeTestResultsReport(cmd, workspaceAbsRoot, report); err != nil {
			return err
		}
	}

	if anyFailed {
		cmd.SilenceUsage = true
		return fmt.Errorf("there are test failures")
	}
	return nil
}

// runTestsForProject compiles the package once, runs each module's tests
// with its own fresh runtime, and returns the combined exit code plus the
// package's aggregated report (nil unless --test-report and tests ran).
func runTestsForProject(cmd *cobra.Command, opts *testCmdOptions, stderr io.Writer, fsys fs.FS,
	project projects.Project, projectDir string) (int, *packageTestResult, error) {
	pkg := project.CurrentPackage()
	compilation := pkg.Compilation()
	if cd := compilation.DiagnosticResult(); cd.HasErrors() || cd.HasWarnings() {
		printDiagnostics(fsys, stderr, cd, !isTerminal(), compilation.DiagnosticEnv())
		if cd.HasErrors() {
			return 1, nil, testError("compilation contains errors")
		}
	}

	if opts.statsOneline {
		_, _ = fmt.Fprint(stderr, compilation.StatsReportOneline())
	} else if opts.stats {
		_, _ = fmt.Fprint(stderr, compilation.StatsReport())
	}

	modules := pkg.Modules()
	var exitCode int
	var err error
	if len(modules) == 1 {
		// Fast path, and identical behavior to before multi-module support:
		// always run the (only) module, even with zero discovered tests —
		// startSuite() itself reports "No tests found" in that case.
		exitCode, err = runTestsForModule(cmd, opts, stderr, fsys, project, compilation, modules[0], projectDir)
	} else {
		exitCode, err = runTestsForModules(cmd, opts, stderr, fsys, project, compilation, modules, projectDir)
	}
	if err != nil {
		return 1, nil, err
	}

	if !opts.testReport {
		return exitCode, nil, nil
	}
	result, hadAny := buildPackageTestResult(pkg, modules, projectDir)
	if !hadAny {
		return exitCode, nil, nil
	}
	return exitCode, &result, nil
}

// runTestsForModules runs every module that has discovered tests. A module
// with zero discovered tests is skipped entirely — no header, no report.
func runTestsForModules(cmd *cobra.Command, opts *testCmdOptions, stderr io.Writer, fsys fs.FS,
	project projects.Project, compilation *projects.PackageCompilation,
	modules []*projects.Module, projectDir string) (int, error) {
	anyHadTests := false
	anyFailed := false
	for _, module := range modules {
		if !discoverTests(stderr, module).HasAny() {
			continue
		}
		anyHadTests = true
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nRunning tests for module '%s'\n", module.ModuleName().String())
		exitCode, err := runTestsForModule(cmd, opts, stderr, fsys, project, compilation, module, projectDir)
		if err != nil {
			return 1, err
		}
		if exitCode != 0 {
			anyFailed = true
		}
	}
	if !anyHadTests {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\tNo tests found")
		return 0, nil
	}
	if anyFailed {
		return 1, nil
	}
	return 0, nil
}

// discoverTests reads the compiler plugin's stashed TestPlan for module. A
// missing stash is reported loudly, not silently treated as normal.
func discoverTests(stderr io.Writer, module *projects.Module) *testplan.TestPlan {
	key := testplan.ModuleKey{
		Org:    module.PackageInstance().PackageOrg().Value(),
		Module: module.ModuleName().String(),
	}
	if plan, ok := testplan.Load(key); ok {
		return plan
	}
	_, _ = fmt.Fprintf(stderr, "warning: no compiler-plugin TestPlan for %s/%s; treating as no discovered tests\n",
		key.Org, key.Module)
	return &testplan.TestPlan{}
}

// runTestsForModule runs one module's test suite in-process via its own
// fresh runtime and testrunner.Run — no generated glue, no registration.
// Returns the suite's own exit code (0 = all passed) separately from error.
func runTestsForModule(cmd *cobra.Command, opts *testCmdOptions, stderr io.Writer, fsys fs.FS,
	project projects.Project, compilation *projects.PackageCompilation,
	module *projects.Module, projectDir string) (int, error) {
	discovered := discoverTests(stderr, module)

	moduleName := module.ModuleName().String()
	pkgName := module.PackageInstance().PackageName().Value()
	rerunPath := filepath.Join(projectDir, projects.TargetDir, "rerun_test.json")
	included, err := testfilter.Select(discovered, testfilter.SelectOptions{
		Tests:         opts.tests,
		Groups:        opts.groups,
		DisableGroups: opts.disableGroups,
		RerunFailed:   opts.rerunFailed,
		RerunJSONPath: rerunPath,
		ModuleName:    moduleName,
		PackageName:   pkgName,
	})
	if err != nil {
		// Mirrors filter.bal#setTestOptions's own handling of an invalid
		// --rerun-failed rerun_test.json: print the message and report it as
		// a suite failure (exit 1), not a hard CLI usage error.
		message := err.Error()
		if errors.Is(err, testfilter.ErrInvalidRerunData) {
			message = invalidRerunDataMessage
		}
		_, _ = fmt.Fprintln(os.Stdout, "error: "+message)
		return 1, nil
	}

	// --list-groups lists every discovered group and short-circuits,
	// ignoring other selection flags (matches jballerina).
	if opts.listGroups {
		if listGroupsOtherFlagsWarning(opts) {
			_, _ = fmt.Fprint(os.Stdout, "\nWarning: Other flags are skipped when list-groups flag is provided.\n\n")
		}
		printGroupsList(os.Stdout, testGroupsList(discovered, nil))
		return 0, nil
	}

	// Mirrors startSuite's own "testRegistry.getFunctions().length() == 0 &&
	// getDependentFunctions().length() == 0" check — register.bal only ever
	// registers an *included* test, so an empty included set is exactly that
	// condition's Go-side equivalent.
	if len(included) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\tNo tests found")
		return 0, nil
	}

	// Build/run against the included subset only, so a --tests/--groups
	// selection can't fail over an unselected test's bad dependency.
	// Safe: included is already closed over dependsOn (testfilter.Select).
	selected := filterPlan(discovered, included)
	graph, err := testplan.BuildGraph(selected)
	if err != nil {
		// Mirrors startSuite's own `orderTests()` failure branch: print the
		// message and report it as a suite failure (exit 1), not a hard CLI
		// usage error — a cyclic or disabled dependency is a problem with the
		// test code, not with how bal test was invoked.
		_, _ = fmt.Fprintln(os.Stdout, err.Error())
		return 1, nil
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		return 1, testError("BIR generation failed: no BIR package produced")
	}

	rootOrg := module.PackageInstance().PackageOrg().Value()
	rootName := pkgName
	for _, birPkg := range birPkgs {
		if birPkg.PackageID.OrgName.Value() == rootOrg && birPkg.PackageID.PkgName.Value() == rootName {
			// bal test never runs the package's own main() — only the
			// directly-invoked test/hook functions drive execution.
			birPkg.MainFunction = nil
		}
	}

	tyEnv := project.Environment().TypeEnv()
	pal, cleanupSignals := palnative.NewPlatform()
	defer cleanupSignals()
	rt := runtime.NewRuntime(pal, tyEnv)
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
			cmd.SilenceErrors = true
			return 1, err
		}
	}
	rt.Listen()

	// --rerun-failed replaces --tests-based selection entirely (matching
	// filter.bal#setTestOptions's own mutually-exclusive branching), so a
	// data-driven test's previously-failed sub-keys come from rerun_test.json
	// instead of (never alongside) a --tests "name#key" value.
	subKeyFilters := testrunner.ParseSubKeyFilters(opts.tests)
	for name, keys := range testfilter.SubKeysFromRerun(testfilter.SelectOptions{
		RerunFailed: opts.rerunFailed, RerunJSONPath: rerunPath, ModuleName: moduleName,
	}) {
		subKeyFilters[name] = keys
	}
	start := time.Now()
	report := testrunner.Run(rt, rootOrg, moduleName, selected, graph, included, subKeyFilters)
	executionTime := time.Since(start)

	// Gracefully stop any module-level listeners the package under test
	// declared, now that the suite has finished with them — bal test starts
	// listeners the same way bal run does, but unlike bal run (meant to run
	// forever until a signal), it must tear them down itself once done, or
	// registered runtime:onGracefulStop handlers never run.
	rt.RequestGracefulStop()

	// ballerina/test's own println writes straight to the real process
	// stdout rather than through cobra's captured writer — this matches that
	// existing split (cmd.OutOrStdout() is only used for the "Running tests
	// for module/package" headers around this call) rather than introducing
	// a new one.
	testrunner.WriteConsoleReport(os.Stdout, report, native.IsSystemConsole(), executionTime)

	if opts.testReport || opts.codeCoverage {
		statusPath := filepath.Join(projectDir, projects.TargetDir, "cache", "tests_cache", moduleName, "module_status.json")
		if err := testrunner.WriteModuleStatusFile(statusPath, testrunner.BuildModuleStatus(report)); err != nil {
			return 1, err
		}
	}
	if err := testrunner.WriteRerunEntry(rerunPath, moduleName, testrunner.BuildRerunEntry(report, moduleName)); err != nil {
		return 1, err
	}

	return testrunner.Summarize(report).ExitCode, nil
}

// filterPlan returns a copy of plan containing only the tests named in
// included; hooks are carried over unchanged since they're global and never
// subject to --tests/--groups filtering.
func filterPlan(plan *testplan.TestPlan, included map[string]bool) *testplan.TestPlan {
	filtered := &testplan.TestPlan{
		BeforeSuite:  plan.BeforeSuite,
		AfterSuite:   plan.AfterSuite,
		BeforeEach:   plan.BeforeEach,
		AfterEach:    plan.AfterEach,
		BeforeGroups: plan.BeforeGroups,
		AfterGroups:  plan.AfterGroups,
	}
	for _, t := range plan.Tests {
		if included[t.Name] {
			filtered.Tests = append(filtered.Tests, t)
		}
	}
	return filtered
}

// testGroupsList lists every group named by an included test, deduplicated,
// in first-seen order. included == nil means no restriction.
func testGroupsList(plan *testplan.TestPlan, included map[string]bool) []string {
	seen := make(map[string]bool)
	var groups []string
	for _, t := range plan.Tests {
		if included != nil && !included[t.Name] {
			continue
		}
		for _, g := range t.Groups {
			if !seen[g] {
				seen[g] = true
				groups = append(groups, g)
			}
		}
	}
	return groups
}

// printGroupsList ports execute.bal#startSuite's --list-groups branch
// exactly: "\tFollowing groups are available :\n\t[g1, g2]" or
// "\tThere are no groups available!".
func printGroupsList(w io.Writer, groups []string) {
	if len(groups) == 0 {
		_, _ = fmt.Fprintln(w, "\tThere are no groups available!")
		return
	}
	_, _ = fmt.Fprintln(w, "\tFollowing groups are available :")
	_, _ = fmt.Fprintln(w, "\t["+strings.Join(groups, ", ")+"]")
}

// listGroupsOtherFlagsWarning reports whether any selection flag is set
// alongside --list-groups, which jballerina warns about and ignores.
func listGroupsOtherFlagsWarning(opts *testCmdOptions) bool {
	return opts.rerunFailed || opts.codeCoverage || opts.testReport ||
		opts.groups != "" || opts.disableGroups != "" || opts.tests != ""
}
