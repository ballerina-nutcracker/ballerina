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
	"os"
	"path/filepath"
	"testing"

	_ "ballerina/lib/rt" // register stdlib runtime functions (io.println etc.)
	"ballerina/lib/stdlibs"
	"ballerina/projects"
)

// TestMixedBalaFormat verifies that a user project depending on both a v4 bala
// (Bala.toml, schema_version="4") and a legacy v3 bala (package.json) compiles
// and runs correctly. main() calls greet() from each dependency and prints the
// results. Expected stdout: "v4\nlegacy"
func TestMixedBalaFormat(t *testing.T) {
	repoPath := filepath.Join(packageResolutionTestDataDir, "mixed-bala-format", "repo")
	userProjectPath := filepath.Join(packageResolutionTestDataDir, "mixed-bala-format", "userProject")

	repo := projects.NewFileSystemRepository(os.DirFS(repoPath), ".")

	result, err := projects.Load(
		os.DirFS(userProjectPath),
		".",
		projects.ProjectLoadConfig{
			Repositories: []projects.Repository{
				repo,
				projects.NewFileSystemRepository(stdlibs.FS, "."),
			},
		},
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Diagnostics().HasErrors() {
		t.Fatalf("load diagnostics: %v", result.Diagnostics().Diagnostics())
	}

	compilation := result.Project().CurrentPackage().Compilation()
	if compilation.DiagnosticResult().HasErrors() {
		t.Fatalf("compilation diagnostics: %v", compilation.DiagnosticResult().Diagnostics())
	}

	birPkgs := projects.NewBallerinaBackend(compilation).BIRPackages()
	if len(birPkgs) == 0 {
		t.Fatal("backend produced no BIR packages")
	}

	got := interpretBIRPackagesStdout(t, result.Project().Environment().TypeEnv(), birPkgs)
	want := "v4\nlegacy"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}
