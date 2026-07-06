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

// Package testharness provides the unified Run + Validate + Update integration
// test harness used by corpus and extern tests. It lives in a sub-package of
// test_util to avoid importing heavy compiler/runtime packages into test_util
// itself (which is imported by parser tests, creating a cycle).
package testharness

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/tools/diagnostics"
)

// TestCase / TestKind / TestSuffix and the suffix constants live in test_util
// (no heavy imports needed). The harness consumes them from there directly.

// ---------------------------------------------------------------------------
// TestSuffix: bitset describing the corpus naming convention.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Discovery
// ---------------------------------------------------------------------------

// GetSingleFileTestCases walks inputDir/**/*.bal and returns matching
// TestCases with IsProject=false. inputDir is the directory containing the
// .bal sources (typically "<corpus>/bal"). mask filters by suffix at the
// source. kind determines the corresponding expected-output directory and
// extension, resolved relative to filepath.Dir(inputDir).
func GetSingleFileTestCases(inputDir string, kind test_util.TestKind, mask test_util.TestSuffix) ([]test_util.TestCase, error) {
	outputDir, outputExt := outputDirAndExt(filepath.Dir(inputDir), kind)
	var cases []test_util.TestCase
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".bal") {
			return nil
		}
		rel, _ := filepath.Rel(inputDir, path)
		tc := test_util.TestCase{
			Name:         rel,
			InputPath:    path,
			ExpectedPath: filepath.Join(outputDir, strings.TrimSuffix(rel, ".bal")+outputExt),
		}
		if tc.Suffix()&mask == 0 {
			return nil
		}
		cases = append(cases, tc)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cases, nil
}

// GetProjectTestCases enumerates direct subdirectories of inputDir whose
// name carries a recognised suffix. Returns TestCases with IsProject=true.
// inputDir is the directory containing the project subtrees (typically
// "<corpus>/project" or "<corpus>/workspace"). kind is used purely to
// resolve the expected-output path, relative to filepath.Dir(inputDir).
func GetProjectTestCases(inputDir string, kind test_util.TestKind, mask test_util.TestSuffix) ([]test_util.TestCase, error) {
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return nil, nil
	}
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, err
	}
	subdir := filepath.Base(inputDir)
	outputDir, outputExt := outputDirAndExt(filepath.Dir(inputDir), kind)
	expectedSubDir := filepath.Join(outputDir, subdir)
	var cases []test_util.TestCase
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		tc := test_util.TestCase{
			Name:         subdir + "/" + dirName,
			InputPath:    filepath.Join(inputDir, dirName),
			ExpectedPath: filepath.Join(expectedSubDir, dirName+outputExt),
			IsProject:    true,
		}
		if tc.Suffix()&mask == 0 {
			continue
		}
		cases = append(cases, tc)
	}
	return cases, nil
}

func outputDirAndExt(basePath string, kind test_util.TestKind) (string, string) {
	switch kind {
	case test_util.AST:
		return filepath.Join(basePath, "ast"), ".txt"
	case test_util.Parser:
		return filepath.Join(basePath, "parser"), ".json"
	case test_util.BIR:
		return filepath.Join(basePath, "bir"), ".txt"
	case test_util.CFG:
		return filepath.Join(basePath, "cfg"), ".txt"
	case test_util.Desugar:
		return filepath.Join(basePath, "desugared"), ".txt"
	case test_util.Integration:
		return filepath.Join(basePath, "integration"), ".txtar"
	case test_util.Bench:
		return filepath.Join(basePath, "bench-integration"), ".txtar"
	}
	return filepath.Join(basePath, "integration"), ".txtar"
}

// ---------------------------------------------------------------------------
// TestPal: in-memory PAL with stdout/stderr capture and a diagnostics slot.
// ---------------------------------------------------------------------------

// ResolvedDiag is a flattened, file/line-resolved view of one error
// diagnostic. Lines are 1-based, inclusive on both ends.
type ResolvedDiag struct {
	File      string
	StartLine int
	EndLine   int
}

// TestPal is the in-memory PAL handed to Run. It captures stdout/stderr and
// stores structured compile diagnostics for Validate to read.
type TestPal interface {
	Platform() pal.Platform
	Stdout() string
	Stderr() string
	WriteStdout(s string)
	WriteStderr(s string)
	Diagnostics() []ResolvedDiag
	SetDiagnostics([]ResolvedDiag)
	// SetReporter attaches a failure reporter used by the in-memory
	// signal source watchdog (see test_util.NewTestSignalSource).
	SetReporter(r test_util.FailReporter)
	// SendGracefulStop pushes a graceful stop signal onto the PAL's signal
	// channel so the runtime can wind down deterministically from tests.
	SendGracefulStop()
	// Close releases test PAL resources such as signal watchdog timers.
	Close()
}

// stubHTTP returns a fixed canned response. Tests that need a different
// behaviour can swap in their own HTTPClient via a custom TestPal.
type stubHTTP struct{}

func (c *stubHTTP) Execute(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string, _ map[string][]string) (int, map[string][]string, io.ReadCloser, error) {
	return 200, map[string][]string{}, io.NopCloser(strings.NewReader("test body")), nil
}

type testPal struct {
	mu            sync.Mutex
	stdout        bytes.Buffer
	stderr        bytes.Buffer
	diags         []ResolvedDiag
	reporter      test_util.FailReporter
	signalSrc     pal.SignalSource
	signalCh      chan pal.Signal
	signalCleanup func()
	signalInit    bool
}

// NewTestPal returns a fresh in-memory TestPal. The optional reporter is
// notified if the signal-watchdog forces a graceful shutdown.
func NewTestPal() TestPal {
	return &testPal{}
}

func normalizePath(path string) string {
	if goruntime.GOOS == "windows" && strings.HasPrefix(path, "/tmp/") {
		return filepath.Join(os.TempDir(), path[5:])
	}
	return path
}

func (p *testPal) Platform() pal.Platform {
	p.ensureSignalSource()
	return pal.Platform{
		IO: pal.IO{
			Stdout: func(b []byte) (int, error) {
				p.mu.Lock()
				defer p.mu.Unlock()
				return p.stdout.Write(b)
			},
			Stderr: func(b []byte) (int, error) {
				p.mu.Lock()
				defer p.mu.Unlock()
				return p.stderr.Write(b)
			},
		},
		FS: pal.FS{
			ReadFile: func(path string) ([]byte, error) {
				return os.ReadFile(normalizePath(path))
			},
			WriteFile: func(path string, data []byte) error {
				return os.WriteFile(normalizePath(path), data, 0o644)
			},
			AppendFile: func(path string, data []byte) (err error) {
				f, err := os.OpenFile(normalizePath(path), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
				if err != nil {
					return err
				}
				defer func() {
					if cerr := f.Close(); cerr != nil && err == nil {
						err = cerr
					}
				}()
				_, err = f.Write(data)
				return err
			},
		},
		Time: pal.Time{
			Now:          time.Now,
			MonotonicNow: func() time.Duration { return time.Since(time.Time{}) },
		},
		HTTP: pal.HTTP{
			NewClient: func(_ pal.ClientConfig) pal.HTTPClient {
				return &stubHTTP{}
			},
		},
		Signals: p.signalSrc,
	}
}

func (p *testPal) ensureSignalSource() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.signalInit {
		return
	}
	p.signalSrc, p.signalCh, p.signalCleanup = test_util.NewTestSignalSource(p.reporter, test_util.TestSignalTimeout)
	p.signalInit = true
}

func (p *testPal) SendGracefulStop() {
	p.ensureSignalSource()
	defer func() { _ = recover() }() // channel may already be closed
	p.signalCh <- pal.GracefulStop
}

func (p *testPal) Close() {
	p.mu.Lock()
	cleanup := p.signalCleanup
	p.mu.Unlock()
	if cleanup != nil {
		cleanup()
	}
}

func (p *testPal) Stdout() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stdout.String()
}

func (p *testPal) Stderr() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stderr.String()
}

func (p *testPal) WriteStdout(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stdout.WriteString(s)
}

func (p *testPal) WriteStderr(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stderr.WriteString(s)
}

func (p *testPal) Diagnostics() []ResolvedDiag {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.diags
}

func (p *testPal) SetDiagnostics(d []ResolvedDiag) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.diags = d
}

func (p *testPal) SetReporter(r test_util.FailReporter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reporter = r
}

// ---------------------------------------------------------------------------
// Run: compile + interpret a TestCase against a TestPal.
// ---------------------------------------------------------------------------

const panicPrefix = "panic: "

// ExternRegistration registers one native function on the runtime before
// interpretation.
type ExternRegistration struct {
	Org      string
	Module   string
	FuncName string
	Impl     extern.NativeFunc
}

// Run compiles and (if compilation succeeded) interprets the test case,
// writing all output through pal. Diagnostics flow into both pal.WriteStderr
// (rendered text, for golden-file diff) and pal.SetDiagnostics (structured,
// for @error marker checking).
func Run(t testing.TB, tc test_util.TestCase, pal TestPal, externs []ExternRegistration) {
	t.Helper()
	pal.SetReporter(t)
	defer pal.Close()
	defer func() {
		if r := recover(); r != nil {
			msg := strings.TrimPrefix(fmt.Sprintf("%v", r), panicPrefix)
			pal.WriteStdout(panicPrefix + msg + "\n")
		}
	}()

	fsys, entry := runFS(tc)
	ballerinaEnvFs, err := ballerinaEnvFS()
	if err != nil {
		pal.WriteStdout(err.Error() + "\n")
		return
	}

	result, err := projects.Load(fsys, entry, projects.ProjectLoadConfig{BallerinaEnvFs: ballerinaEnvFs})
	if err != nil {
		pal.WriteStdout(err.Error() + "\n")
		return
	}
	tyEnv := result.Project().Environment().TypeEnv()
	currentPkg := result.Project().CurrentPackage()
	compilation := currentPkg.Compilation()

	var stderr bytes.Buffer
	if tc.IsProject {
		printDiagnostics(fsys, &stderr, result.Diagnostics(), compilation.DiagnosticEnv())
	}
	printDiagnostics(fsys, &stderr, compilation.DiagnosticResult(), compilation.DiagnosticEnv())
	pal.WriteStderr(stderr.String())
	pal.SetDiagnostics(resolveErrorDiagnostics(compilation.DiagnosticResult(), compilation.DiagnosticEnv()))

	loaderHasErrors := tc.IsProject && result.Diagnostics().HasErrors()
	if loaderHasErrors || compilation.DiagnosticResult().HasErrors() {
		return
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()
	if len(birPkgs) == 0 {
		t.Fatalf("compilation succeeded but produced no BIR packages for %s", tc.Name)
	}

	rt := runtime.NewRuntime(pal.Platform(), tyEnv)
	for _, e := range externs {
		runtime.RegisterExternFunction(rt, e.Org, e.Module, e.FuncName, e.Impl)
	}
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			pal.WriteStderr(err.Error() + "\n")
			return
		}
	}
	rt.Listen()
	invokeTestMain(t, rt, birPkgs, pal)
	hasListeners := hasListeners(birPkgs)
	if hasListeners {
		pal.SendGracefulStop()
	}
	code := <-rt.ExitStatus
	switch tc.Suffix() {
	case test_util.SuffixValid, test_util.SuffixFutureValid:
		expected := uint8(0)
		if hasListeners {
			expected = gracefulStopExitCode
		}
		if code != expected {
			t.Errorf("%s: expected exit code %d, got %d", tc.Name, expected, code)
		}
	case test_util.SuffixPanic, test_util.SuffixFuturePanic:
		if code == 0 {
			t.Errorf("%s: expected non-zero exit code, got 0", tc.Name)
		}
	}
}

// gracefulStopExitCode mirrors runtime/lifecycle.go's graceful stop code
// (128 + SIGINT). Kept here to avoid exporting the constant just for tests.
const gracefulStopExitCode uint8 = 130

func hasListeners(pkgs []*bir.BIRPackage) bool {
	// if the package have listeners we expect lifecycle hooks
	for _, p := range pkgs {
		if p != nil && p.StartFunction != nil {
			return true
		}
	}
	return false
}

// testMainFunctionName is the optional user-defined function the test harness
// invokes after Listen() to drive listeners from inside the listening state.
const testMainFunctionName = "testMain"

// invokeTestMain looks up `testMain` on each BIR package and invokes it on
// the live runtime. Used by listener tests to exercise the runtime while
// it's parked in Listening state, before the harness pushes a graceful stop.
func invokeTestMain(t testing.TB, rt *runtime.Runtime, pkgs []*bir.BIRPackage, pal TestPal) {
	t.Helper()
	for _, p := range pkgs {
		if p == nil || p.PackageID == nil || p.PackageID.OrgName == nil || p.PackageID.PkgName == nil {
			continue
		}
		org := p.PackageID.OrgName.Value()
		module := p.PackageID.PkgName.Value()
		fn, ok := runtime.LookupFunction(rt, org, module, testMainFunctionName)
		if !ok {
			continue
		}
		if _, err := runtime.InvokeFunction(rt, fn, nil); err != nil {
			pal.WriteStderr(err.Error() + "\n")
		}
	}
}

func runFS(tc test_util.TestCase) (fs.FS, string) {
	if tc.IsProject {
		return os.DirFS(tc.InputPath), "."
	}
	return os.DirFS(filepath.Dir(tc.InputPath)), filepath.Base(tc.InputPath)
}

func ballerinaEnvFS() (fs.FS, error) {
	if v := os.Getenv(projects.BallerinaEnvVar); v != "" {
		return os.DirFS(v), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return os.DirFS(filepath.Join(home, projects.UserHomeDirName)), nil
}

func resolveErrorDiagnostics(result projects.DiagnosticResult, de *diagnostics.DiagnosticEnv) []ResolvedDiag {
	errs := result.Errors()
	if len(errs) == 0 {
		return nil
	}
	out := make([]ResolvedDiag, 0, len(errs))
	for _, d := range errs {
		loc := d.Location()
		if !diagnostics.LocationHasSource(loc) {
			continue
		}
		out = append(out, ResolvedDiag{
			File:      de.FileName(loc),
			StartLine: de.StartLine(loc) + 1,
			EndLine:   de.EndLine(loc) + 1,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Validate / Update
// ---------------------------------------------------------------------------

// Validate diffs pal's captured output against the golden file at
// tc.ExpectedPath, runs suffix-based invariants on the expected content, and
// checks @output/@error/@panic markers in the source(s) against both.
func Validate(t *testing.T, tc test_util.TestCase, pal TestPal) {
	t.Helper()
	expectedStdout, expectedStderr, err := test_util.LoadTxtarStdoutStderr(tc.ExpectedPath)
	if err != nil {
		t.Fatalf("failed to load expected from %s: %v", tc.ExpectedPath, err)
	}
	checkExpectedOutputInvariants(t, tc, expectedStdout, expectedStderr)

	actualStdout := pal.Stdout()
	actualStderr := normalizeIntegrationStderr(pal.Stderr())

	stdoutMismatch := expectedStdout != actualStdout
	stderrMismatch := expectedStderr != actualStderr
	if !stdoutMismatch && !stderrMismatch {
		assertAnnotations(t, tc, actualStdout, pal.Stderr(), pal.Diagnostics())
		return
	}

	var msg strings.Builder
	if stdoutMismatch {
		fmt.Fprintf(&msg, "stdout mismatch\n%s", test_util.FormatExpectedGot(expectedStdout, actualStdout))
	}
	if stderrMismatch {
		if msg.Len() > 0 {
			msg.WriteString("\n\n")
		}
		fmt.Fprintf(&msg, "stderr mismatch\n%s", test_util.FormatExpectedGot(expectedStderr, actualStderr))
	}
	t.Errorf("%s", msg.String())
	assertAnnotations(t, tc, actualStdout, pal.Stderr(), pal.Diagnostics())
}

// Update writes pal's captured output to tc.ExpectedPath as a .txtar archive.
// Runs invariants against the new content; does NOT fail the test on change.
func Update(t *testing.T, tc test_util.TestCase, pal TestPal) {
	t.Helper()
	stdout := pal.Stdout()
	stderr := normalizeIntegrationStderr(pal.Stderr())
	checkExpectedOutputInvariants(t, tc, stdout, stderr)
	_ = test_util.UpdateTxtarArchiveIfNeeded(t, tc.ExpectedPath, test_util.TxtarFilesStdoutStderr(stdout, stderr))
}

func checkExpectedOutputInvariants(t *testing.T, tc test_util.TestCase, stdout, stderr string) {
	t.Helper()
	stderrNonEmpty := strings.TrimSpace(stderr) != ""
	switch tc.Suffix() {
	case test_util.SuffixValid:
		if stderrNonEmpty {
			t.Fatalf("-v test %q has non-empty expected stderr:\n%s", tc.Name, stderr)
		}
		if strings.Contains(stdout, "panic:") {
			t.Fatalf("-v test %q has a runtime panic in expected stdout:\n%s", tc.Name, stdout)
		}
	case test_util.SuffixError:
		if !stderrNonEmpty {
			t.Fatalf("-e test %q has empty expected stderr", tc.Name)
		}
		if !strings.HasPrefix(strings.TrimSpace(stderr), "error[") {
			t.Fatalf("-e test %q expected stderr is not a compile diagnostic (`error[...]: ...`):\n%s", tc.Name, stderr)
		}
		numErr := strings.Count(stderr, "\nerror[") + boolToInt(strings.HasPrefix(stderr, "error["))
		numLoc := strings.Count(stderr, "--> ")
		if numLoc < numErr {
			t.Fatalf("-e test %q expected stderr has a diagnostic with no source location (%d errors, %d `-->` lines):\n%s", tc.Name, numErr, numLoc, stderr)
		}
	case test_util.SuffixFutureValid, test_util.SuffixFutureError, test_util.SuffixFuturePanic:
		if !stderrNonEmpty {
			t.Fatalf("future test %q has empty expected stderr; must surface a `fatal[...]` bailout", tc.Name)
		}
		if !strings.HasPrefix(strings.TrimSpace(stderr), "fatal[") {
			t.Fatalf("future test %q expected stderr is not a `fatal[...]` bailout:\n%s", tc.Name, stderr)
		}
	case test_util.SuffixPanic:
		if !stderrNonEmpty {
			t.Fatalf("-p test %q has empty expected stderr", tc.Name)
		}
		if strings.HasPrefix(strings.TrimSpace(stderr), "error[") {
			t.Fatalf("-p test %q expected stderr begins with a compile diagnostic; -p tests must produce a runtime panic:\n%s", tc.Name, stderr)
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func splitStderrDiagnostics(stderr string) []string {
	var out []string
	for part := range strings.SplitSeq(stderr, "\n\n") {
		s := strings.TrimSpace(part)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func normalizeIntegrationStderr(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}
	diags := splitStderrDiagnostics(stderr)
	slices.Sort(diags)
	return strings.Join(diags, "\n\n") + "\n"
}

// ---------------------------------------------------------------------------
// Annotation parsing + assertion (moved from corpus/).
// ---------------------------------------------------------------------------

type outputAnn struct {
	line  int
	value string
}
type (
	errorAnn struct{ line int }
	panicAnn struct{ line int }
)

type fileAnns struct {
	outputs []outputAnn
	errors  []errorAnn
	panics  []panicAnn
}

type annotations map[string]*fileAnns

type annSourceFile struct {
	key     string
	absPath string
}

var (
	outputRe = regexp.MustCompile(`(?://)\s*@output\b[ \t]?(.*)$`)
	errorRe  = regexp.MustCompile(`(?://)\s*@error\b`)
	panicRe  = regexp.MustCompile(`(?://)\s*@panic\b`)
)

func parseAnnotations(sources []annSourceFile) (annotations, error) {
	anns := annotations{}
	for _, src := range sources {
		content, err := os.ReadFile(src.absPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", src.absPath, err)
		}
		if fa := parseAnnotationsInFile(string(content)); fa != nil {
			anns[src.key] = fa
		}
	}
	return anns, nil
}

func parseAnnotationsInFile(content string) *fileAnns {
	var fa fileAnns
	for i, raw := range strings.Split(content, "\n") {
		lineNo := i + 1
		line := strings.TrimSuffix(raw, "\r")
		commentIdx := strings.Index(line, "//")
		if commentIdx < 0 {
			continue
		}
		comment := line[commentIdx:]
		if m := outputRe.FindStringSubmatch(comment); m != nil {
			fa.outputs = append(fa.outputs, outputAnn{line: lineNo, value: strings.TrimRight(m[1], " \t")})
			continue
		}
		if panicRe.MatchString(comment) {
			fa.panics = append(fa.panics, panicAnn{line: lineNo})
			continue
		}
		if errorRe.MatchString(comment) {
			fa.errors = append(fa.errors, errorAnn{line: lineNo})
		}
	}
	if fa.outputs == nil && fa.errors == nil && fa.panics == nil {
		return nil
	}
	return &fa
}

func collectSources(tc test_util.TestCase) ([]annSourceFile, error) {
	if !tc.IsProject {
		return []annSourceFile{{key: filepath.Base(tc.InputPath), absPath: tc.InputPath}}, nil
	}
	var sources []annSourceFile
	err := filepath.Walk(tc.InputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".bal") {
			return nil
		}
		rel, err := filepath.Rel(tc.InputPath, path)
		if err != nil {
			return err
		}
		sources = append(sources, annSourceFile{key: filepath.ToSlash(rel), absPath: path})
		return nil
	})
	return sources, err
}

type baseLine struct {
	base string
	line int
}

func (b baseLine) String() string { return fmt.Sprintf("%s:%d", b.base, b.line) }

func assertAnnotations(t *testing.T, tc test_util.TestCase, stdout, stderr string, diags []ResolvedDiag) {
	t.Helper()
	sources, err := collectSources(tc)
	if err != nil {
		t.Errorf("failed to collect sources: %v", err)
		return
	}
	anns, err := parseAnnotations(sources)
	if err != nil {
		t.Errorf("failed to parse annotations: %v", err)
		return
	}
	switch tc.Suffix() {
	case test_util.SuffixFutureValid, test_util.SuffixFutureError, test_util.SuffixFuturePanic:
		// front-end bailout; no precise location to verify.
	case test_util.SuffixValid:
		assertOutputAnnotations(t, anns, stdout, stderr)
	case test_util.SuffixError:
		assertErrorAnnotations(t, anns, diags)
	case test_util.SuffixPanic:
		assertPanicAnnotations(t, anns, stdout, stderr)
	}
}

func assertOutputAnnotations(t *testing.T, anns annotations, stdout, stderr string) {
	t.Helper()
	if strings.TrimRight(stderr, "\n") != "" {
		t.Errorf("expected empty stderr for -v test, got:\n%s", stderr)
	}
	outputs, ok := singleOutputFile(t, anns)
	if !ok {
		return
	}
	if len(outputs) == 0 && strings.TrimRight(stdout, "\n") != "" {
		t.Errorf("test produced stdout but has no @output annotations:\n%s", stdout)
		return
	}
	expected := buildExpectedStdout(outputs)
	if stdout != expected {
		t.Errorf("@output annotation mismatch\n%s", test_util.FormatExpectedGot(expected, stdout))
	}
}

func singleOutputFile(t *testing.T, anns annotations) ([]outputAnn, bool) {
	t.Helper()
	var owners []string
	for file, fa := range anns {
		if len(fa.outputs) > 0 {
			owners = append(owners, file)
		}
	}
	switch len(owners) {
	case 0:
		return nil, true
	case 1:
		return anns[owners[0]].outputs, true
	default:
		sort.Strings(owners)
		t.Errorf("@output annotations must live in a single file; found in: %s", strings.Join(owners, ", "))
		return nil, false
	}
}

func buildExpectedStdout(outputs []outputAnn) string {
	if len(outputs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, o := range outputs {
		b.WriteString(o.value)
		b.WriteByte('\n')
	}
	return b.String()
}

func assertErrorAnnotations(t *testing.T, anns annotations, diags []ResolvedDiag) {
	t.Helper()
	if !hasErrorAnnotation(anns) {
		t.Errorf("-e test must have at least one @error annotation")
		return
	}
	if len(diags) == 0 {
		t.Errorf("-e test produced no error diagnostics")
		return
	}
	covered := make(map[string]map[int]bool)
	for _, d := range diags {
		if !diagnosticCoversAnnotatedLine(d, anns, covered) {
			t.Errorf("diagnostic at %s:%d-%d not covered by any @error annotation", d.File, d.StartLine, d.EndLine)
		}
	}
	for file, fa := range anns {
		for _, e := range fa.errors {
			if !covered[file][e.line] {
				t.Errorf("@error annotation at %s:%d is not covered by any diagnostic", file, e.line)
			}
		}
	}
}

func hasErrorAnnotation(anns annotations) bool {
	for _, fa := range anns {
		if len(fa.errors) > 0 {
			return true
		}
	}
	return false
}

func diagnosticCoversAnnotatedLine(d ResolvedDiag, anns annotations, covered map[string]map[int]bool) bool {
	fa, ok := anns[d.File]
	if !ok {
		return false
	}
	matched := false
	for _, e := range fa.errors {
		if e.line >= d.StartLine && e.line <= d.EndLine {
			if covered[d.File] == nil {
				covered[d.File] = make(map[int]bool)
			}
			covered[d.File][e.line] = true
			matched = true
		}
	}
	return matched
}

var stackFrameLineRe = regexp.MustCompile(`\(([^()\s:]+):(\d+)\)`)

func parseStackFrames(stderr string) []baseLine {
	matches := stackFrameLineRe.FindAllStringSubmatch(stderr, -1)
	frames := make([]baseLine, 0, len(matches))
	for _, m := range matches {
		line, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		frames = append(frames, baseLine{base: m[1], line: line})
	}
	return frames
}

func assertPanicAnnotations(t *testing.T, anns annotations, stdout, stderr string) {
	t.Helper()
	want, ok := singlePanicAnnotation(t, anns)
	if !ok {
		return
	}
	outputs, ok := singleOutputFile(t, anns)
	if !ok {
		return
	}
	expectedStdout := buildExpectedStdout(outputs)
	if stdout != expectedStdout {
		t.Errorf("@output annotation mismatch for -p test\n%s", test_util.FormatExpectedGot(expectedStdout, stdout))
	}
	frames := parseStackFrames(stderr)
	if len(frames) == 0 {
		if !strings.Contains(stderr, "error:") {
			t.Errorf("stderr does not contain a runtime error message for @panic at %s\nstderr:\n%s", want, stderr)
		}
		return
	}
	if frames[0] != want {
		t.Errorf("top stack frame mismatch for @panic: want %s, got %s\nstderr:\n%s", want, frames[0], stderr)
	}
}

func singlePanicAnnotation(t *testing.T, anns annotations) (baseLine, bool) {
	t.Helper()
	var (
		count int
		owner string
		ann   panicAnn
	)
	for file, fa := range anns {
		count += len(fa.panics)
		if len(fa.panics) > 0 {
			owner = file
			ann = fa.panics[0]
		}
	}
	if count != 1 {
		t.Errorf("-p test must have exactly one @panic annotation, found %d", count)
		return baseLine{}, false
	}
	return baseLine{base: filepath.Base(owner), line: ann.line}, true
}

// ---------------------------------------------------------------------------
// Diagnostic printer (moved from corpus/).
// ---------------------------------------------------------------------------

type diagnosticLocation struct {
	filePath            string
	startLine, startCol int
	endLine, endCol     int
	numWidth            int
}

func buildDiagnosticLocation(filePath string, startLine, startCol, endLine, endCol int) diagnosticLocation {
	startLineNumStr := fmt.Sprintf("%d", startLine+1)
	endLineNumStr := fmt.Sprintf("%d", endLine+1)
	numWidth := len(startLineNumStr)
	if w := len(endLineNumStr); w > numWidth {
		numWidth = w
	}
	return diagnosticLocation{
		filePath: filePath, startLine: startLine, startCol: startCol,
		endLine: endLine, endCol: endCol, numWidth: numWidth,
	}
}

func printDiagnostics(fsys fs.FS, w io.Writer, diagResult projects.DiagnosticResult, de *diagnostics.DiagnosticEnv) {
	for _, d := range diagResult.Diagnostics() {
		printDiagnostic(fsys, w, d, de)
	}
}

func printDiagnostic(fsys fs.FS, w io.Writer, d diagnostics.Diagnostic, de *diagnostics.DiagnosticEnv) {
	printDiagnosticHeader(w, d)
	location := d.Location()
	if diagnostics.IsLocationEmpty(location) {
		_, _ = fmt.Fprintln(w)
		return
	}
	if !diagnostics.LocationHasSource(location) {
		_, _ = fmt.Fprintf(w, "  --> %s\n\n", de.FileName(location))
		return
	}
	loc := buildDiagnosticLocation(
		de.FileName(location),
		de.StartLine(location), de.StartColumn(location),
		de.EndLine(location), de.EndColumn(location),
	)
	printDiagnosticLocation(w, loc)
	printSourceSnippet(w, loc, fsys)
	_, _ = fmt.Fprintln(w)
}

func printDiagnosticHeader(w io.Writer, d diagnostics.Diagnostic) {
	info := d.DiagnosticInfo()
	codeStr := ""
	if c := info.Code(); c != "" {
		codeStr = fmt.Sprintf("[%s]", c)
	}
	_, _ = fmt.Fprintf(w, "%s%s: %s\n",
		strings.ToLower(info.Severity().String()), codeStr, d.Message(),
	)
}

func printDiagnosticLocation(w io.Writer, loc diagnosticLocation) {
	_, _ = fmt.Fprintf(w, "%*s--> %s:%d:%d\n",
		loc.numWidth, "", loc.filePath, loc.startLine+1, loc.startCol+1,
	)
	if loc.filePath != "" {
		_, _ = fmt.Fprintf(w, "%*s |\n", loc.numWidth, "")
	}
}

func printSourceSnippet(w io.Writer, loc diagnosticLocation, fsys fs.FS) {
	content, err := fs.ReadFile(fsys, loc.filePath)
	if err != nil {
		return
	}
	lines := strings.Split(string(content), "\n")
	if loc.startLine >= len(lines) {
		return
	}
	for line := loc.startLine; line <= loc.endLine && line < len(lines); line++ {
		lineContent := strings.TrimSuffix(lines[line], "\r")
		lineNumStr := fmt.Sprintf("%d", line+1)
		startCol := 0
		var endCol int
		switch {
		case loc.startLine == loc.endLine:
			startCol = loc.startCol
			endCol = loc.endCol
		case line == loc.startLine:
			startCol = loc.startCol
			endCol = len(lineContent)
		case line == loc.endLine:
			startCol = 0
			endCol = loc.endCol
		default:
			startCol = 0
			endCol = len(lineContent)
		}
		var highlightLen int
		startCol, _, highlightLen = computeTrimmedCaretSpan(lineContent, startCol, endCol)
		_, _ = fmt.Fprintf(w, "%*s | %s\n", loc.numWidth, lineNumStr, lineContent)
		_, _ = fmt.Fprintf(w, "%*s | %s\n", loc.numWidth, "", buildPointer(lineContent, startCol, highlightLen))
	}
}

func computeTrimmedCaretSpan(lineContent string, startCol, endCol int) (trimStartCol, trimEndCol, highlightLen int) {
	firstNonWS := -1
	for i := 0; i < len(lineContent); i++ {
		if lineContent[i] != ' ' && lineContent[i] != '\t' {
			firstNonWS = i
			break
		}
	}
	lastNonWS := len(lineContent)
	hasNonWS := firstNonWS != -1
	if hasNonWS {
		for lastNonWS > firstNonWS && (lineContent[lastNonWS-1] == ' ' || lineContent[lastNonWS-1] == '\t') {
			lastNonWS--
		}
	}
	if !hasNonWS {
		return startCol, startCol, 0
	}
	if startCol < firstNonWS {
		startCol = firstNonWS
	}
	highlightLen = endCol - startCol
	return startCol, endCol, highlightLen
}

func buildPointer(lineContent string, startCol, highlightLen int) string {
	var b strings.Builder
	for i := 0; i < startCol && i < len(lineContent); i++ {
		if lineContent[i] == '\t' {
			b.WriteByte('\t')
		} else {
			b.WriteByte(' ')
		}
	}
	for range highlightLen {
		b.WriteByte('^')
	}
	return b.String()
}
