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

//go:build !js && !wasm

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
)

// TestValidateTemplate_CaseInsensitive verifies -t matches regardless of
// case, consistent with bal add's own --template flag.
func TestValidateTemplate_CaseInsensitive(t *testing.T) {
	tests := []struct {
		raw  string
		want templateName
	}{
		{"LIB", templateLib},
		{"Lib", templateLib},
		{"lib", templateLib},
		{"SERVICE", templateService},
		{"Default", templateDefault},
		{"MAIN", templateMain},
	}
	for _, tt := range tests {
		got, err := validateTemplate(tt.raw)
		if err != nil {
			t.Errorf("validateTemplate(%q) = %v, want no error", tt.raw, err)
		} else if got != tt.want {
			t.Errorf("validateTemplate(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
	if _, err := validateTemplate("bogus"); err == nil {
		t.Error("validateTemplate(\"bogus\") = nil error, want an error")
	}
}

// TestNewPackage_LibTemplateGeneratesReadme verifies `bal new -t lib` creates
// a README.md with the package name substituted into the import example,
// and that other templates don't generate a README.md at all.
func TestNewPackage_LibTemplateGeneratesReadme(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	libPkgPath := filepath.Join(tmpDir, "mylib")

	cmd := createNewCmd()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{libPkgPath, "-t", "lib"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, errBuf.String())
	}

	readmePath := filepath.Join(libPkgPath, projects.ReadmeMdFile)
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("expected README.md to exist for lib template: %v", err)
	}
	contentStr := string(content)

	if strings.Contains(contentStr, "PKG_NAME") || strings.Contains(contentStr, "ORG_NAME") {
		t.Errorf("README.md still has unsubstituted placeholders:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "/mylib") {
		t.Errorf("README.md import example missing package name 'mylib':\n%s", contentStr)
	}
	// The default import prefix for `import ORG_NAME/mylib;` is the package
	// name ("mylib"), not the org — the usage examples must qualify calls
	// with mylib:, not ORG_NAME's substituted value.
	if !strings.Contains(contentStr, "mylib:hello(") {
		t.Errorf("README.md usage examples must qualify hello() with the package name 'mylib:':\n%s", contentStr)
	}
	if org := guessOrgName(); org != "mylib" {
		if strings.Contains(contentStr, org+":hello(") {
			t.Errorf("README.md must not qualify hello() with the org name %q:\n%s", org, contentStr)
		}
	}

	for _, tc := range []string{"default", "main", "service"} {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			pkgPath := filepath.Join(t.TempDir(), "mypackage")

			cmd := createNewCmd()
			var outBuf, errBuf bytes.Buffer
			cmd.SetOut(&outBuf)
			cmd.SetErr(&errBuf)
			cmd.SetArgs([]string{pkgPath, "-t", tc})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", err, errBuf.String())
			}
			if _, err := os.Stat(filepath.Join(pkgPath, projects.ReadmeMdFile)); err == nil {
				t.Errorf("unexpected README.md generated for template %q", tc)
			}
		})
	}
}

// TestInitPackage_CleansUpOnReadmeWriteFailure covers the lib-template-only
// README.md write branch: pre-creating README.md as a directory lets
// Ballerina.toml and lib.bal write successfully first, so cleanup() has
// more than one entry to unwind.
func TestInitPackage_CleansUpOnReadmeWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based write-failure injection is unix-only")
	}
	projectPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectPath, projects.ReadmeMdFile), 0755); err != nil {
		t.Fatalf("failed to pre-create README.md as a directory: %v", err)
	}

	err := initPackage(projectPath, "mypkg", "myorg", templateLib)
	if err == nil {
		t.Fatal("expected an error writing README.md over an existing directory")
	}
	if !strings.Contains(err.Error(), "failed to create README.md") {
		t.Errorf("err = %q, want 'failed to create README.md' message", err)
	}

	if _, statErr := os.Stat(filepath.Join(projectPath, projects.BallerinaTomlFile)); !os.IsNotExist(statErr) {
		t.Errorf("expected Ballerina.toml to be cleaned up, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(projectPath, "lib.bal")); !os.IsNotExist(statErr) {
		t.Errorf("expected lib.bal to be cleaned up, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(projectPath, ".gitignore")); !os.IsNotExist(statErr) {
		t.Errorf("expected .gitignore to be cleaned up, stat err = %v", statErr)
	}
}

// TestNewWorkspace_LoadsCorrectly tests that a workspace created by `bal new
// --workspace` can be loaded back with projects.Load() into the expected
// typed *projects.WorkspaceProject shape. This is a library-level round-trip
// check (does bal new's output parse back into the right Go struct), not
// something observable through the compiled binary's stdout/stderr/exit-code
// surface, so it stays a white-box in-process test rather than moving to
// corpus's CLI-level integration suite.
func TestNewWorkspace_LoadsCorrectly(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-workspace")

	cmd := createNewCmd()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{workspacePath, "--workspace"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, errBuf.String())
	}

	fsys := os.DirFS(workspacePath)
	userHome, _ := os.UserHomeDir()
	ballerinaEnvFs := os.DirFS(filepath.Join(userHome, projects.UserHomeDirName))

	result, err := projects.Load(fsys, ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFs,
	})
	if err != nil {
		t.Fatalf("failed to load workspace: %v", err)
	}

	project := result.Project()
	if project.Kind() != projects.ProjectKindWorkspace {
		t.Errorf("expected ProjectKindWorkspace, got: %v", project.Kind())
	}

	workspace, ok := project.(*projects.WorkspaceProject)
	if !ok {
		t.Fatalf("expected *projects.WorkspaceProject, got: %T", project)
	}

	if len(workspace.Manifest().Packages()) != 1 {
		t.Errorf("expected 1 package in workspace, got: %d", len(workspace.Manifest().Packages()))
	}
	if len(workspace.Projects()) != 1 {
		t.Errorf("expected 1 project in workspace, got: %d", len(workspace.Projects()))
	}
}
