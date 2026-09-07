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

package corpus

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/bir"
	bircodec "github.com/ballerina-nutcracker/ballerina/bir/codec"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"

	_ "github.com/ballerina-nutcracker/ballerina/lib/rt"
)

const (
	corpusLibBaseDir = "../corpus/lib"

	corpusProjectBaseDir            = "../corpus/project"
	corpusProjectIntegrationBaseDir = "../corpus/integration/project"

	corpusWorkspaceBaseDir            = "../corpus/workspace"
	corpusWorkspaceIntegrationBaseDir = "../corpus/integration/workspace"

	panicPrefix = "panic: "
)

var (
	update = flag.Bool("update", false, "update corpus integration test outputs")

	// skipIntegrationTests is the integration-level *additional* skip list,
	// layered on top of the shared test_util.UnsupportedTests baseline.
	//
	// The authoritative "pi does not support this end-to-end yet" list lives in
	// test_util.UnsupportedTests and is reused by every per-stage corpus test.
	// Only add an entry here when a test must be skipped at integration time but
	// is still useful at earlier stages; otherwise add it to
	// test_util.UnsupportedTests so all stages pick it up.
	skipIntegrationTests = []string{
		// Workspace tests whose errors are at the project-loading level
		// (Ballerina.toml issues — missing package, TOML parse error). These
		// diagnostics have no source location in any .bal file, so they're
		// filtered out by resolveErrorDiagnostics. The annotation validator
		// requires source-located diagnostics for -e tests, so these can't be
		// satisfied today. Skip until the validator handles loader-level errors
		// (or until the diagnostics are re-routed to Ballerina.toml's text doc
		// once that's registered in DiagnosticEnv).
		"project/missing-package-e",
		"project/parse-error-e",
		// Pre-existing future test that does not currently surface a runtime
		// panic or a compile-time `fatal[...]` bailout, so it does not satisfy
		// the future-test contract yet. Tracked separately.
		"subset8/08-future/fieldlvalue1-fp.bal",
		// https://github.com/ballerina-nutcracker/ballerina/issues/533
		"subset9/09-template-expr/template-query-xml-sequence-fv.bal",
		// https://github.com/ballerina-nutcracker/ballerina/issues/538
		"subset9/09-object/readonly-distinct-object-fe.bal",
	}

	// Skip project-level integration tests with non-deterministic output.
	skipProjectIntegrationTests = []string{
		// Migrated from nballerina testSuite/08-import/const4-e: cycle-detection picks a different
		// break point than the upstream compiler, so the reported error path is not stable.
		"import-const4-e",

		// Expected error:
		"import-const5-e",
		"import-type3-e",

		// Expected clean run:
		"import-main-v",
		"import-type6-v",
	}
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

type testResult struct {
	success        bool
	expectedStdout string
	actualStdout   string
	expectedStderr string
	actualStderr   string
}

func TestIntegration(t *testing.T) {
	cases, err := testharness.GetSingleFileTestCases("../corpus/bal", test_util.Integration, test_util.SuffixAny)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			runHarnessCase(t, tc)
		})
	}
}

// TestLibIntegration runs the standard-library corpus end-to-end. These tests
// carry no per-stage goldens, so this is the only place their output is pinned.
func TestLibIntegration(t *testing.T) {
	cases, err := testharness.GetNestedSingleFileTestCases(corpusLibBaseDir, test_util.Integration, test_util.SuffixAny)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			runHarnessCase(t, tc)
		})
	}
}

func TestProjectIntegration(t *testing.T) {
	cases, err := testharness.GetProjectTestCases("../corpus/project", test_util.Integration, test_util.SuffixAny)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	for _, tc := range cases {
		t.Run(filepath.Base(tc.InputPath), func(t *testing.T) {
			t.Parallel()
			runHarnessCase(t, tc)
		})
	}
}

// TestWorkspaceIntegration runs the same compile + interpret pipeline against
// each fixture under corpus/workspace/<name>/, comparing stdout/stderr to
// corpus/integration/workspace/<name>.txtar.
func TestWorkspaceIntegration(t *testing.T) {
	cases, err := testharness.GetProjectTestCases("../corpus/workspace", test_util.Integration, test_util.SuffixAny)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	for _, tc := range cases {
		t.Run(filepath.Base(tc.InputPath), func(t *testing.T) {
			t.Parallel()
			runHarnessCase(t, tc)
		})
	}
}

// runHarnessCase wires Run + Validate/Update for one TestCase, applying the
// integration-level skip lists before invoking the harness.
func runHarnessCase(t *testing.T, tc test_util.TestCase) {
	if harnessSkip(tc) {
		t.Skipf("Skipping integration test for %s", tc.Name)
	}
	pal := testharness.NewTestPal()
	testharness.Run(t, tc, pal, nil)
	if *update {
		testharness.Update(t, tc, pal)
		return
	}
	testharness.Validate(t, tc, pal)
}

func harnessSkip(tc test_util.TestCase) bool {
	if tc.IsProject {
		dir := filepath.Base(tc.InputPath)
		if isProjectTestSkipped(dir) {
			return true
		}
		return isSkipKey("project/" + dir)
	}
	return isTestSkipped(tc)
}

// suffixOf returns the trailing -v / -e / -p / -fv / -fe / -fp marker on a
// test name (file or project dir), or "" when no recognized marker is
// present.
func suffixOf(name string) string {
	base := strings.TrimSuffix(filepath.Base(name), ".bal")
	if i := strings.LastIndex(base, "-"); i >= 0 {
		s := base[i+1:]
		switch s {
		case "v", "e", "p", "fv", "fe", "fp":
			return s
		}
	}
	return ""
}

func splitStderrDiagnostics(stderr string) []string {
	var diagnostics []string
	for part := range strings.SplitSeq(stderr, "\n\n") {
		diagnostic := strings.TrimSpace(part)
		if diagnostic != "" {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics
}

// logTimestampPattern matches the leading "time=<RFC3339>" field of a
// ballerina/log LOGFMT record so it can be normalized to a stable token,
// keeping golden files deterministic across runs.
var logTimestampPattern = regexp.MustCompile(`time=\S+`)

func normalizeIntegrationStderr(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}

	stderr = logTimestampPattern.ReplaceAllString(stderr, "time=<TIME>")
	diagnostics := splitStderrDiagnostics(stderr)

	slices.Sort(diagnostics)
	return strings.Join(diagnostics, "\n\n") + "\n"
}

func isTestSkipped(tc test_util.TestCase) bool {
	return isSkipKey(filepath.ToSlash(tc.Name))
}

// isSkipKey reports whether the given corpus-relative key should be skipped at
// integration time. A test is skipped when it is on the shared
// test_util.UnsupportedTests baseline or on the integration-only
// skipIntegrationTests additions.
func isSkipKey(key string) bool {
	return test_util.IsUnsupported(key) || slices.Contains(skipIntegrationTests, key)
}

func isProjectTestSkipped(dirName string) bool {
	return slices.Contains(skipProjectIntegrationTests, dirName)
}

func evaluateTestResult(expectedStdout, expectedStderr, actualStdout, actualStderr string) testResult {
	stderrMatch := expectedStderr == normalizeIntegrationStderr(actualStderr)
	return testResult{
		success:        actualStdout == expectedStdout && stderrMatch,
		expectedStdout: expectedStdout,
		actualStdout:   actualStdout,
		expectedStderr: expectedStderr,
		actualStderr:   actualStderr,
	}
}

// runCompilePhase exists separately from testharness.Run because
// TestBIRSerializationRoundtrip needs the raw BIR packages to
// serialize/deserialize before interpreting them.
func runCompilePhase(balFile string, stdoutBuf, stderrBuf *bytes.Buffer) (pkgs []*bir.BIRPackage, tyEnv semtypes.Env, err error) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("%v", r)
			msg = strings.TrimPrefix(msg, panicPrefix)
			fmt.Fprintf(stdoutBuf, "%s%s\n", panicPrefix, msg)
			err = fmt.Errorf("compile panic")
		}
	}()

	fsys := os.DirFS(filepath.Dir(balFile))

	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		fmt.Fprintf(stdoutBuf, "%s\n", err.Error())
		return nil, nil, err
	}
	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)

	compiled, err := testharness.CompileWithDriver(fsys, filepath.Base(balFile), "", projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFs,
	})
	if err != nil {
		fmt.Fprintf(stdoutBuf, "%s\n", err.Error())
		return nil, nil, err
	}
	tyEnv = compiled.Resolver.CompilerEnvironment().GetTypeEnv()

	diagnosticResult := projects.NewDiagnosticResult(compiled.Context.Diagnostics())
	testharness.PrintDiagnostics(fsys, stderrBuf, diagnosticResult, compiled.Context.DiagnosticEnv())
	if diagnosticResult.HasErrors() {
		return nil, tyEnv, nil
	}
	return compiled.BIRPackages, tyEnv, nil
}

// runInterpretPhase takes already-compiled birPkgs (rather than compiling
// itself) so TestBIRSerializationRoundtrip can interpret packages that went
// through a serialize/deserialize roundtrip in between.
func runInterpretPhase(birPkgs []*bir.BIRPackage, tyEnv semtypes.Env, stdoutBuf, stderrBuf *bytes.Buffer) {
	if len(birPkgs) == 0 {
		return
	}

	pal := testharness.NewTestPal()
	defer pal.Close()
	defer func() {
		stdoutBuf.WriteString(pal.Stdout())
		stderrBuf.WriteString(pal.Stderr())
	}()
	rt := runtime.NewRuntime(pal.Platform(), tyEnv)
	hasLifecycle := false
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			// For now just write the error string to stderr to match corpus expectations
			fmt.Fprintln(stderrBuf, err.Error())
			return
		}
		if birPkg.StartFunction != nil {
			hasLifecycle = true
		}
	}
	rt.Listen()
	for _, birPkg := range birPkgs {
		invokeTestMainOnPkg(rt, birPkg, stderrBuf)
	}
	if hasLifecycle {
		pal.SendGracefulStop()
	}
	<-rt.ExitStatus
}

const testMainFunctionName = "testMain"

func invokeTestMainOnPkg(rt *runtime.Runtime, pkg *bir.BIRPackage, stderrBuf *bytes.Buffer) {
	if pkg == nil || pkg.PackageID == nil || pkg.PackageID.OrgName == nil || pkg.PackageID.PkgName == nil {
		return
	}
	org := pkg.PackageID.OrgName.Value()
	module := pkg.PackageID.PkgName.Value()
	fn, ok := runtime.LookupFunction(rt, org, module, testMainFunctionName)
	if !ok {
		return
	}
	if _, err := runtime.InvokeFunction(rt, fn, nil); err != nil {
		fmt.Fprintln(stderrBuf, err.Error())
	}
}

func findProjectDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if suffixOf(name) != "" {
			dirs = append(dirs, filepath.Join(dir, name))
		}
	}
	return dirs
}

func runProjectInterpretPhase(birPkgs []*bir.BIRPackage, tyEnv semtypes.Env, stdoutBuf, stderrBuf *bytes.Buffer) {
	if len(birPkgs) == 0 {
		return
	}

	pal := testharness.NewTestPal()
	defer pal.Close()
	defer func() {
		stdoutBuf.WriteString(pal.Stdout())
		stderrBuf.WriteString(pal.Stderr())
	}()
	rt := runtime.NewRuntime(pal.Platform(), tyEnv)
	hasLifecycle := false
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			fmt.Fprintln(stderrBuf, err.Error())
			return
		}
		if birPkg.StartFunction != nil {
			hasLifecycle = true
		}
	}
	rt.Listen()
	for _, birPkg := range birPkgs {
		invokeTestMainOnPkg(rt, birPkg, stderrBuf)
	}
	if hasLifecycle {
		pal.SendGracefulStop()
	}
	<-rt.ExitStatus
}

func TestProjectSerializationRoundtrip(t *testing.T) {
	flag.Parse()

	if _, err := os.Stat(corpusProjectBaseDir); os.IsNotExist(err) {
		return
	}

	projectDirs := findProjectDirs(corpusProjectBaseDir)

	for _, projDir := range projectDirs {
		dirName := filepath.Base(projDir)
		if !strings.HasSuffix(dirName, "-v") {
			continue
		}
		txtarPath := filepath.Join(corpusProjectIntegrationBaseDir, dirName+".txtar")

		t.Run(dirName, func(t *testing.T) {
			t.Parallel()
			// Roundtrip test reuses the integration project skip list because any project
			// skipped at the integration level has no usable expected fixture.
			if isProjectTestSkipped(dirName) {
				t.Skipf("Skipping project serialization roundtrip for %s", dirName)
			}
			testProjectSerializationRoundtrip(t, dirName, projDir, txtarPath)
		})
	}
}

func testProjectSerializationRoundtrip(t *testing.T, dirName, projDir, txtarPath string) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic while running %s: %v", dirName, r)
		}
	}()

	expectedStdout, expectedStderr, err := test_util.LoadTxtarStdoutStderr(txtarPath)
	if err != nil {
		t.Fatalf("failed to load expected from %s: %v", txtarPath, err)
	}

	stdout, stderr := runProjectSerializationRoundtrip(projDir)
	result := evaluateTestResult(expectedStdout, expectedStderr, stdout, stderr)
	if result.success {
		return
	}

	stdoutMismatch := result.expectedStdout != result.actualStdout
	stderrMismatch := result.expectedStderr != result.actualStderr

	var msg strings.Builder
	if stdoutMismatch {
		fmt.Fprintf(&msg, "stdout mismatch\n%s", test_util.FormatExpectedGot(result.expectedStdout, result.actualStdout))
	}
	if stderrMismatch {
		if msg.Len() > 0 {
			msg.WriteString("\n\n")
		}
		fmt.Fprintf(&msg, "stderr mismatch\n%s", test_util.FormatExpectedGot(result.expectedStderr, result.actualStderr))
	}
	t.Errorf("%s", msg.String())
}

func runProjectSerializationRoundtrip(projectDir string) (stdout, stderr string) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	fsys := os.DirFS(projectDir)
	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		fmt.Fprintf(&stdoutBuf, "%s\n", err.Error())
		return stdoutBuf.String(), stderrBuf.String()
	}
	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)
	compiled, err := testharness.CompileWithDriver(fsys, ".", filepath.Base(projectDir), projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFs,
	})
	if err != nil {
		fmt.Fprintf(&stdoutBuf, "%s\n", err.Error())
		return stdoutBuf.String(), stderrBuf.String()
	}
	tyEnv := compiled.Resolver.CompilerEnvironment().GetTypeEnv()

	diagnosticResult := projects.NewDiagnosticResult(compiled.Context.Diagnostics())
	testharness.PrintDiagnostics(fsys, &stderrBuf, diagnosticResult, compiled.Context.DiagnosticEnv())
	if diagnosticResult.HasErrors() {
		return stdoutBuf.String(), stderrBuf.String()
	}

	birPkgs := compiled.BIRPackages
	if len(birPkgs) == 0 {
		return stdoutBuf.String(), stderrBuf.String()
	}
	serialized := make([][]byte, 0, len(birPkgs))
	for _, pkg := range birPkgs {
		birBytes, err := bircodec.Marshal(tyEnv, pkg)
		if err != nil {
			fmt.Fprintf(&stdoutBuf, "BIR serialization failed: %v\n", err)
			return stdoutBuf.String(), stderrBuf.String()
		}
		serialized = append(serialized, birBytes)
	}

	freshEnv := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	deserialized := make([]*bir.BIRPackage, 0, len(birPkgs))
	for _, birBytes := range serialized {
		freshCtx := context.NewCompilerContext(freshEnv)
		deserializedPkg, err := bircodec.Unmarshal(freshCtx, birBytes)
		if err != nil {
			fmt.Fprintf(&stdoutBuf, "BIR deserialization failed: %v\n", err)
			return stdoutBuf.String(), stderrBuf.String()
		}
		deserialized = append(deserialized, deserializedPkg)
	}

	runProjectInterpretPhase(deserialized, freshEnv.GetTypeEnv(), &stdoutBuf, &stderrBuf)
	return stdoutBuf.String(), stderrBuf.String()
}

func BenchmarkIntegration(b *testing.B) {
	testPairs := test_util.GetTests(b, test_util.Bench, func(path string) bool {
		return true
	})
	for _, testPair := range testPairs {
		b.Run(testPair.Name, func(b *testing.B) {
			expectedStdout, expectedStderr, err := test_util.LoadTxtarStdoutStderr(testPair.ExpectedPath)
			if err != nil {
				b.Fatalf("failed to load expected from %s: %v", testPair.ExpectedPath, err)
			}

			var pal testharness.TestPal
			b.ResetTimer()
			for b.Loop() {
				pal = testharness.NewTestPal()
				testharness.Run(b, testPair, pal, nil)
			}
			b.StopTimer()

			result := evaluateTestResult(expectedStdout, expectedStderr, pal.Stdout(), pal.Stderr())
			if !result.success {
				b.Fatalf("output mismatch for %s:\nstdout:\n%s\nstderr:\n%s",
					testPair.InputPath,
					test_util.FormatExpectedGot(result.expectedStdout, result.actualStdout),
					test_util.FormatExpectedGot(
						normalizeIntegrationStderr(result.expectedStderr),
						normalizeIntegrationStderr(result.actualStderr),
					))
			}
		})
	}
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
