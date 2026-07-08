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
	"os"
	"path/filepath"

	debugcommon "ballerina-lang-go/common"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/tools/diagnostics"

	"github.com/spf13/cobra"
)

// balaSubdir is the directory under <project>/target/ that holds emitted
// .bala archives.
const balaSubdir = "bala"

// packOptions holds CLI flag values for `bal pack`. Kept structurally identical
// to runOpts so the two commands share the same compile-observability surface.
type packOptions struct {
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
	cmd.Flags().BoolVar(&opts.dumpCFG, "dump-cfg", false, "Dump control flow graph")
	cmd.Flags().BoolVar(&opts.dumpBIR, "dump-bir", false, "Dump Ballerina Intermediate Representation")
	cmd.Flags().BoolVar(&opts.traceRecovery, "trace-recovery", false, "Enable error recovery tracing")
	cmd.Flags().BoolVar(&opts.stats, "stats", false, "Print per-stage compilation timing statistics")
	cmd.Flags().BoolVar(&opts.statsOneline, "stats-oneline", false, "Print per-stage compilation timing totals only")
	cmd.Flags().StringVar(&opts.logFile, "log-file", "", "Write debug output to specified file")
	cmd.Flags().StringVar(&opts.format, "format", "", "Output format for dump operations (dot)")
	// Profiler flags are registered onto the global packCmd from prof_*.go's init().
	// They are intentionally NOT registered inside createPackCmd, so test-instantiated
	// commands skip profiler flags (the tests don't exercise profiling).
	return cmd
}

// packError reports a pack-specific failure to stderr (writer w) and
// returns the same error so cobra exits non-zero.
func packError(w io.Writer, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	printErrorTo(w, err, "pack [<package-dir>]", false)
	return err
}

func runPack(cmd *cobra.Command, args []string, opts *packOptions) error {
	stderr := cmd.ErrOrStderr()

	// Build options from CLI flags. Constructed before debug setup so
	// buildOpts is the single source of truth for all flag reads.
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

	// Profiler flags are bound only when prof_*.go's init() registers them
	// on this cmd. In release builds RegisterFlags is a no-op; in debug
	// builds it runs against the global packCmd. Test-instantiated cmds
	// never carry the flag, so they skip Start.
	if cmd.Flag("prof") != nil {
		if err := profiler.Start(); err != nil {
			return packError(stderr, "failed to start profiler: %w", err)
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
				return packError(stderr, "error creating log file %s: %w", opts.logFile, err)
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
		return packError(stderr, "invalid project path %q: %w", path, err)
	}
	if !info.IsDir() {
		if filepath.Ext(path) == ".bal" {
			return packError(stderr, "pack does not support single-file projects; %q is a .bal file", path)
		}
		return packError(stderr, "pack requires a package directory; got %q", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return packError(stderr, "resolve absolute path: %w", err)
	}

	fsys := os.DirFS(absPath)
	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return packError(stderr, "resolve ballerina env path: %w", err)
	}

	result, err := projects.Load(fsys, ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		return packError(stderr, "failed to load package: %w", err)
	}

	if diagResult := result.Diagnostics(); diagResult.HasErrors() {
		printDiagnostics(fsys, stderr, diagResult, !isTerminal(), diagnostics.NewDiagnosticEnv())
		return packError(stderr, "package loading reported errors")
	}

	project := result.Project()
	if project.Kind() == projects.ProjectKindWorkspace {
		return packError(stderr, "provided path %q is a workspace; expected a package directory", path)
	}

	pkg := project.CurrentPackage()
	compilation := pkg.Compilation()
	if cd := compilation.DiagnosticResult(); cd.HasErrors() {
		printDiagnostics(fsys, stderr, cd, !isTerminal(), compilation.DiagnosticEnv())
		return packError(stderr, "compilation failed; .bala not produced")
	}

	balaDir := filepath.Join(absPath, projects.TargetDir, balaSubdir)
	backend := projects.NewBallerinaBackend(compilation)
	balaPath, err := backend.EmitBala(balaDir)
	if err != nil {
		return packError(stderr, "write bala: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", balaPath)
	return nil
}
