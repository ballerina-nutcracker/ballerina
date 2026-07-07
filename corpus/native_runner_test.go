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
	"os"
	"path/filepath"
	"testing"

	"ballerina-lang-go/lib/stdlibs"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/test_util"

	// Blank-import native package implementations so their init() functions
	// register extern functions with the runtime before the tests run.
	// These packages live in testdata and are not included in ./... builds,
	// but explicit imports compile and link them normally.
	_ "ballerina-lang-go/projects/testdata/repo/bala/acmeorg/calcpkg/1.0.0/go1.26/native"
	_ "ballerina-lang-go/projects/testdata/repo/bala/mockorg/nativepkg/1.0.0/go1.26/native"
)

const nativeTestDataDir = "extern/testdata"

// TestNativeMultiOrgPackages verifies that native (Go-implemented) Ballerina
// packages from multiple organisations resolve and execute correctly alongside
// pure-Ballerina packages with transitive dependencies.
//
// Package matrix:
//   - mockorg/nativepkg  — native, one org (hello() returns a string)
//   - acmeorg/calcpkg    — native, second org (multiply, abs)
//   - mockorg/greetpkg   — pure Ballerina, v4 bala format
//   - mockorg/middlepkg  — pure Ballerina, transitive deps (leafpkg, aaaleafpkg)
func TestNativeMultiOrgPackages(t *testing.T) {
	projectDir := filepath.Join(nativeTestDataDir, "native-multi-org-v")
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatal(err)
	}

	testRepoPath, err := filepath.Abs(filepath.Join("..", "projects", "testdata", "repo", "bala"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := projects.Load(
		os.DirFS(absProjectDir),
		".",
		projects.ProjectLoadConfig{
			Repositories: []projects.Repository{
				// Bundled stdlib must come first (same order as defaultRepositories).
				projects.NewFileSystemRepository(stdlibs.FS, "."),
				projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
			},
		},
	)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	pkg := result.Project().CurrentPackage()
	compilation := pkg.Compilation()
	if compilation.DiagnosticResult().HasErrors() {
		for _, d := range compilation.DiagnosticResult().Diagnostics() {
			t.Logf("diagnostic: %v", d)
		}
		t.Fatal("compilation had errors")
	}

	backend := projects.NewBallerinaBackend(compilation)
	birPkgs := backend.BIRPackages()

	var stdout, stderr bytes.Buffer
	rt := runtime.NewRuntime(test_util.TestPal(&stdout, &stderr), result.Project().Environment().TypeEnv())

	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			t.Fatalf("runtime error: %v", err)
		}
	}

	const txtarPath = "extern/output/native-multi-org-v.txtar"
	actualStdout := test_util.NormalizeNewlines(stdout.String())
	actualStderr := test_util.NormalizeNewlines(stderr.String())

	if *update {
		if test_util.UpdateTxtarArchiveIfNeeded(t, txtarPath, test_util.TxtarFilesStdoutStderr(actualStdout, actualStderr)) {
			t.Fatalf("Updated expected file: %s", txtarPath)
		}
		return
	}

	expectedStdout, expectedStderr, err := test_util.LoadTxtarStdoutStderr(txtarPath)
	if err != nil {
		t.Fatalf("failed to load golden file %s: %v", txtarPath, err)
	}
	expectedStdout = test_util.NormalizeNewlines(expectedStdout)
	expectedStderr = test_util.NormalizeNewlines(expectedStderr)

	if actualStdout != expectedStdout {
		t.Errorf("stdout mismatch\n%s", test_util.FormatExpectedGot(expectedStdout, actualStdout))
	}
	if actualStderr != expectedStderr {
		t.Errorf("stderr mismatch\n%s", test_util.FormatExpectedGot(expectedStderr, actualStderr))
	}
}
