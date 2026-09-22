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
	"os"
	"path/filepath"

	debugcommon "github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

	"github.com/spf13/cobra"
)

// balaSubdir is the directory under <project>/target/ that holds emitted
// .bala archives.
const balaSubdir = "bala"

// packOptions holds CLI flag values for `bal pack`. Kept structurally identical
// to runOpts so the two commands share the same compile-observability surface.
type packOptions struct {
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
	targetDir        string
}

var packCmd = createPackCmd()

// createPackCmd creates a fresh pack command with its own packOptions
// captured in the RunE closure. The factory shape is what makes tests
// parallel-safe — each test allocates its own command + options pair
// instead of racing on package-level globals.
//
// The global packCmd built from this factory is what bal.go and the
// build-tagged prof_*.go files reference; profiler flags get attached
// to the global only, so test-instantiated commands won't carry them
// (the tests don't exercise profiling).
func createPackCmd() *cobra.Command {
	opts := &packOptions{}
	cmd := &cobra.Command{
		Use:   "pack [<package-dir>]",
		Short: "Create the distribution format (.bala) of the current package",
		Long: `	Create a .bala archive of the current Ballerina package.

	Creates a distributable .bala archive in the '<project>/target/bala/'
	directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPack(cmd, args, opts)
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
	cmd.Flags().StringVar(&opts.targetDir, "target-dir", "", "target directory path")
	// Profiler flags are registered onto the global packCmd from prof_*.go's init().
	// They are intentionally NOT registered inside createPackCmd, so test-instantiated
	// commands skip profiler flags (the tests don't exercise profiling).
	return cmd
}

func packError(format string, args ...any) error {
	return usageError("pack [<package-dir>]", format, args...)
}

func runPack(cmd *cobra.Command, args []string, opts *packOptions) error {
	stderr := cmd.ErrOrStderr()

	// --target-dir is resolved relative to the process cwd (matching
	// clean.go's own --target-dir), independent of the package-dir arg.
	var targetDirOverride string
	if opts.targetDir != "" {
		abs, err := filepath.Abs(opts.targetDir)
		if err != nil {
			return packError("resolve target directory: %w", err)
		}
		targetDirOverride = abs
	}

	// Build options from CLI flags. Constructed before debug setup so
	// buildOpts is the single source of truth for all flag reads.
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
	// builds it runs against the global packCmd. Test-instantiated cmds
	// never carry the flag, so they skip Start.
	if cmd.Flag("prof") != nil {
		if err := profiler.Start(); err != nil {
			return packError("failed to start profiler: %w", err)
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
				return packError("error creating log file %s: %w", opts.logFile, err)
			}
			defer func() { _ = logWriter.Close() }()
			debugcommon.InitDebug(debugFlags, logWriter)
		} else {
			debugcommon.InitDebug(debugFlags, stderr)
		}
	}

	// path here is the process cwd, not project-relative; tests passing no
	// positional arg should t.Chdir and not run in parallel.
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		return packError("invalid project path %q: %w", path, err)
	}
	if !info.IsDir() {
		if filepath.Ext(path) == ".bal" {
			return packError("pack does not support single-file projects; %q is a .bal file", path)
		}
		return packError("pack requires a package directory; got %q", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return packError("resolve absolute path: %w", err)
	}

	// Detect whether absPath sits inside a workspace without being its
	// root — e.g. cwd is a workspace member's own directory. If so, load
	// from the workspace root instead so sibling member-to-member
	// dependencies resolve, matching bal run's findWorkspaceRoot handling.
	workspaceRoot := findWorkspaceRoot(absPath)
	effectiveBaseDir := absPath
	if workspaceRoot != "" && workspaceRoot != absPath {
		effectiveBaseDir = workspaceRoot
	}

	fsys := os.DirFS(effectiveBaseDir)
	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return packError("resolve ballerina env path: %w", err)
	}

	result, err := projects.Load(fsys, ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		return packError("failed to load package: %w", err)
	}

	if diagResult := result.Diagnostics(); diagResult.HasErrors() {
		printDiagnostics(fsys, stderr, diagResult, !isTerminal(), diagnostics.NewDiagnosticEnv())
		// Diagnostics carry the full failure detail, and this isn't a
		// pack-usage mistake, so no USAGE block — but cobra should still
		// print "ballerina: project loading contains errors" as a summary.
		return fmt.Errorf("project loading contains errors")
	}

	project := result.Project()
	if project.Kind() == projects.ProjectKindWorkspace {
		workspace := project.(*projects.WorkspaceProject)
		if workspaceRoot == "" || workspaceRoot == absPath {
			return packError("%q is a workspace; run bal pack <package-path> to pack a specific package within it", path)
		}
		// absPath names one specific member (we walked up to workspaceRoot
		// to load it) — pack just that member.
		memberProject := findBuildProjectByPath(workspace, workspaceRoot, absPath)
		if memberProject == nil {
			return packError("no package found at path %s within workspace %s", absPath, workspaceRoot)
		}
		project = memberProject
	}

	pkg := project.CurrentPackage()
	compilation := pkg.Compilation()
	if cd := compilation.DiagnosticResult(); cd.HasErrors() {
		printDiagnostics(fsys, stderr, cd, !isTerminal(), compilation.DiagnosticEnv())
		// Not a pack-usage mistake, so no USAGE block, but cobra should still
		// print "ballerina: compilation contains errors" as a summary.
		return fmt.Errorf("compilation contains errors")
	}

	targetDir := targetDirOverride
	if targetDir == "" {
		targetDir = filepath.Join(absPath, projects.TargetDir)
	}
	balaDir := filepath.Join(targetDir, balaSubdir)
	backend := projects.NewBallerinaBackend(compilation)
	backendDiags := backend.DiagnosticResult()
	if backendDiags.HasErrors() {
		printDiagnostics(fsys, stderr, backendDiags, !isTerminal(), compilation.DiagnosticEnv())
		return packError("BIR generation failed; .bala not produced")
	}
	balaPath, err := backend.EmitBala(balaDir)
	if err != nil {
		return packError("write bala: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", balaPath)
	return nil
}
