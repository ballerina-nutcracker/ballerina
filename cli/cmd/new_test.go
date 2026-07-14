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

	"ballerina-lang-go/projects"
)

// =============================================================================
// Success Cases
// =============================================================================

// TestNewCommandWithAbsolutePaths tests creating packages at various path depths.
func TestNewCommandWithAbsolutePaths(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		path        string
		packageName string
	}{
		{"single level", "projectA", "projectA"},
		{"two levels", "dir1/projectA", "projectA"},
		{"three levels", "dir2/dir1/projectA", "projectA"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			projectPath := filepath.Join(tmpDir, tc.path)

			stdout, stderr, err := executeNewCommand(t, projectPath)
			if err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
			}

			// Verify success message
			if !strings.Contains(stdout, "Created new package") {
				t.Errorf("expected success message, got stdout: %s", stdout)
			}

			// Verify package structure
			assertPackageStructure(t, projectPath)
		})
	}
}

// TestNewCommandInExistingDirectory tests creating a package in a pre-existing empty directory.
func TestNewCommandInExistingDirectory(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "existing_project")

	// Create directory first
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	stdout, stderr, err := executeNewCommand(t, projectPath)
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Created new package") {
		t.Errorf("expected success message, got stdout: %s", stdout)
	}

	assertPackageStructure(t, projectPath)
}

// TestNewCommandBallerinaTomlContent verifies the content of generated Ballerina.toml.
func TestNewCommandBallerinaTomlContent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "myproject")

	_, _, err := executeNewCommand(t, projectPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	tomlPath := filepath.Join(projectPath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read Ballerina.toml: %v", err)
	}

	contentStr := string(content)

	// Verify required sections and fields
	requiredPatterns := []string{
		"[package]",
		"org = ",
		`name = "myproject"`,
		`version = "0.1.0"`,
		"[build-options]",
		"observabilityIncluded = true",
	}

	for _, pattern := range requiredPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("Ballerina.toml missing '%s'\nContent:\n%s", pattern, contentStr)
		}
	}
}

// TestNewCommandGitignoreContent verifies the content of generated .gitignore.
func TestNewCommandGitignoreContent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "myproject")

	_, _, err := executeNewCommand(t, projectPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	gitignorePath := filepath.Join(projectPath, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	contentStr := string(content)

	expectedPatterns := []string{
		"target/",
		"generated/",
		"Config.toml",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf(".gitignore missing '%s'\nContent:\n%s", pattern, contentStr)
		}
	}
}

// TestNewCommandMainBalContent verifies the content of generated main.bal.
func TestNewCommandMainBalContent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "myproject")

	_, _, err := executeNewCommand(t, projectPath)
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	mainBalPath := filepath.Join(projectPath, "main.bal")
	content, err := os.ReadFile(mainBalPath)
	if err != nil {
		t.Fatalf("failed to read main.bal: %v", err)
	}

	contentStr := string(content)

	expectedPatterns := []string{
		"import ballerina/io;",
		"public function main()",
		"io:println",
		"Hello, World!",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("main.bal missing '%s'\nContent:\n%s", pattern, contentStr)
		}
	}
}

// TestNewCommandWithInvalidProjectName tests package name sanitization.
func TestNewCommandWithInvalidProjectName(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		dirName string
	}{
		{"hyphen", "hello-app"},
		{"dollar", "my$project"},
		{"at sign", "my@project"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			projectPath := filepath.Join(tmpDir, tc.dirName)

			stdout, stderr, err := executeNewCommand(t, projectPath)
			if err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
			}

			// Verify warning about derived name
			if !strings.Contains(stderr, "package name is derived as") {
				t.Errorf("expected name derivation warning in stderr, got: %s", stderr)
			}

			// Verify success
			if !strings.Contains(stdout, "Created new package") {
				t.Errorf("expected success message, got stdout: %s", stdout)
			}

			// Verify Ballerina.toml has derived name
			tomlPath := filepath.Join(projectPath, projects.BallerinaTomlFile)
			content, err := os.ReadFile(tomlPath)
			if err != nil {
				t.Fatalf("failed to read Ballerina.toml: %v", err)
			}

			derivedName := guessPkgName(tc.dirName)
			if !strings.Contains(string(content), `name = "`+derivedName+`"`) {
				t.Errorf("expected derived name '%s' in Ballerina.toml, got:\n%s", derivedName, content)
			}
		})
	}
}

// TestNewCommandWithDigitPrefix tests names starting with digit get "app" prefix.
func TestNewCommandWithDigitPrefix(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "9project")

	stdout, stderr, err := executeNewCommand(t, projectPath)
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}

	// Verify warning
	if !strings.Contains(stderr, "package name is derived as") {
		t.Errorf("expected name derivation warning, got stderr: %s", stderr)
	}

	// Verify success
	if !strings.Contains(stdout, "Created new package") {
		t.Errorf("expected success message, got stdout: %s", stdout)
	}

	// Verify Ballerina.toml has "app9project"
	tomlPath := filepath.Join(projectPath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read Ballerina.toml: %v", err)
	}

	if !strings.Contains(string(content), `name = "app9project"`) {
		t.Errorf("expected name 'app9project' in Ballerina.toml, got:\n%s", content)
	}
}

// TestNewCommandWithOnlyNonAlphanumeric tests pure symbols default to "my_package".
func TestNewCommandWithOnlyNonAlphanumeric(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		dirName string
	}{
		{"hash", "#"},
		{"underscore only", "_"},
	}
	if runtime.GOOS != "windows" {
		testCases = append(testCases, struct {
			name    string
			dirName string
		}{"dots only", "..."})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			projectPath := filepath.Join(tmpDir, tc.dirName)

			stdout, stderr, err := executeNewCommand(t, projectPath)
			if err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
			}

			// Verify warning
			if !strings.Contains(stderr, "package name is derived as") {
				t.Errorf("expected name derivation warning, got stderr: %s", stderr)
			}

			// Verify success
			if !strings.Contains(stdout, "Created new package") {
				t.Errorf("expected success message, got stdout: %s", stdout)
			}

			// Verify Ballerina.toml has "my_package"
			tomlPath := filepath.Join(projectPath, projects.BallerinaTomlFile)
			content, err := os.ReadFile(tomlPath)
			if err != nil {
				t.Fatalf("failed to read Ballerina.toml: %v", err)
			}

			if !strings.Contains(string(content), `name = "my_package"`) {
				t.Errorf("expected name 'my_package' in Ballerina.toml, got:\n%s", content)
			}
		})
	}
}

// TestNewCommandNoArgs tests error when no arguments provided.
func TestNewCommandNoArgs(t *testing.T) {
	t.Parallel()
	_, stderr, err := executeNewCommandWithArgs(t)
	if err == nil {
		t.Fatal("expected error, got success")
	}

	if !strings.Contains(stderr, "project path is not provided") {
		t.Errorf("expected 'project path is not provided' error, got: %s", stderr)
	}
}

// TestNewCommandMultipleArgs tests error when too many arguments provided.
func TestNewCommandMultipleArgs(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "project1")
	path2 := filepath.Join(tmpDir, "project2")

	_, stderr, err := executeNewCommandWithArgs(t, path1, path2)
	if err == nil {
		t.Fatal("expected error, got success")
	}

	if !strings.Contains(stderr, "too many arguments") {
		t.Errorf("expected 'too many arguments' error, got: %s", stderr)
	}
}

// TestNewCommandInExistingProject tests error when directory is already a Ballerina project.
func TestNewCommandInExistingProject(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "existing_project")

	// Create directory with Ballerina.toml
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	tomlPath := filepath.Join(projectPath, projects.BallerinaTomlFile)
	if err := os.WriteFile(tomlPath, []byte("[package]\nname = \"test\""), 0644); err != nil {
		t.Fatalf("failed to create Ballerina.toml: %v", err)
	}

	_, stderr, err := executeNewCommand(t, projectPath)
	if err == nil {
		t.Fatal("expected error, got success")
	}

	if !strings.Contains(stderr, "directory is already a Ballerina project") {
		t.Errorf("expected 'already a Ballerina project' error, got: %s", stderr)
	}
}

// TestNewCommandWithExistingBalFiles tests that command succeeds when .bal files exist,
// but main.bal is NOT created (preserving existing code).
func TestNewCommandWithExistingBalFiles(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "dir_with_bal")

	// Create directory with .bal file
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	balFile := filepath.Join(projectPath, "existing.bal")
	if err := os.WriteFile(balFile, []byte("// existing file"), 0644); err != nil {
		t.Fatalf("failed to create .bal file: %v", err)
	}

	stdout, stderr, err := executeNewCommand(t, projectPath)
	if err != nil {
		t.Fatalf("command should succeed: %v\nstderr: %s", err, stderr)
	}

	// Verify success message
	if !strings.Contains(stdout, "Created new package") {
		t.Errorf("expected success message, got stdout: %s", stdout)
	}

	// Verify Ballerina.toml was created
	tomlPath := filepath.Join(projectPath, projects.BallerinaTomlFile)
	if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
		t.Errorf("Ballerina.toml should be created")
	}

	// Verify .gitignore was created
	gitignorePath := filepath.Join(projectPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Errorf(".gitignore should be created")
	}

	// Verify main.bal was NOT created (since .bal files already exist)
	mainBalPath := filepath.Join(projectPath, "main.bal")
	if _, err := os.Stat(mainBalPath); err == nil {
		t.Errorf("main.bal should NOT be created when .bal files already exist")
	}

	// Verify existing .bal file was not modified
	content, err := os.ReadFile(balFile)
	if err != nil {
		t.Fatalf("failed to read existing .bal file: %v", err)
	}
	if string(content) != "// existing file" {
		t.Errorf("existing .bal file was modified")
	}
}

// TestNewCommandWithConflictingFiles tests error when conflicting files/directories exist.
func TestNewCommandWithConflictingFiles(t *testing.T) {
	t.Parallel()
	conflictingItems := []struct {
		name  string
		isDir bool
	}{
		{"Dependencies.toml", false},
		{"Package.md", false},
		{"Module.md", false},
		{"BalTool.toml", false},
		{projects.ModulesDir, true},
		{projects.TestsDir, true},
	}

	for _, item := range conflictingItems {
		t.Run(item.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			projectPath := filepath.Join(tmpDir, "project")

			// Create project directory
			if err := os.MkdirAll(projectPath, 0755); err != nil {
				t.Fatalf("failed to create directory: %v", err)
			}

			// Create conflicting item
			conflictPath := filepath.Join(projectPath, item.name)
			if item.isDir {
				if err := os.MkdirAll(conflictPath, 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
			} else {
				if err := os.WriteFile(conflictPath, []byte(""), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
			}

			_, stderr, err := executeNewCommand(t, projectPath)
			if err == nil {
				t.Fatal("expected error, got success")
			}

			if !strings.Contains(stderr, "file/directory(s) were found") {
				t.Errorf("expected conflict error for '%s', got: %s", item.name, stderr)
			}
		})
	}
}

// =============================================================================
// Workspace Tests
// =============================================================================

// TestNewWorkspace_EmptyDir tests creating a workspace in an empty directory.
func TestNewWorkspace_EmptyDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-workspace")

	stdout, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}

	// Verify success messages
	if !strings.Contains(stdout, "Created new workspace") {
		t.Errorf("expected 'Created new workspace' message, got stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Created new package 'hello-app'") {
		t.Errorf("expected 'Created new package hello-app' message, got stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Workspace created successfully") {
		t.Errorf("expected 'Workspace created successfully' message, got stdout: %s", stdout)
	}

	// Verify workspace Ballerina.toml
	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read workspace Ballerina.toml: %v", err)
	}
	if !strings.Contains(string(content), "[workspace]") {
		t.Errorf("workspace Ballerina.toml missing [workspace] section:\n%s", content)
	}
	if !strings.Contains(string(content), `"hello-app"`) {
		t.Errorf("workspace Ballerina.toml missing hello-app package:\n%s", content)
	}

	// Verify sample package was created
	pkgPath := filepath.Join(workspacePath, "hello-app")
	assertPackageStructure(t, pkgPath)
}

// TestNewWorkspace_WithTemplate tests creating workspace with different templates.
func TestNewWorkspace_WithTemplate(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		template       string
		packageName    string
		sourceFile     string
		sourceContains string
	}{
		{"default", "hello-app", "main.bal", "public function main()"},
		{"main", "hello-app", "main.bal", "public function main()"},
		{"service", "hello-service", "service.bal", "service / on new http:Listener"},
		{"lib", "hello-lib", "lib.bal", "public function hello(string? name) returns string"},
	}

	for _, tc := range testCases {
		t.Run(tc.template, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			workspacePath := filepath.Join(tmpDir, "my-workspace")

			stdout, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace", "-t", tc.template)
			if err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
			}

			// Verify success message includes expected package name
			expectedMsg := "Created new package '" + tc.packageName + "'"
			if !strings.Contains(stdout, expectedMsg) {
				t.Errorf("expected '%s' message, got stdout: %s", expectedMsg, stdout)
			}

			// Verify package directory exists
			pkgPath := filepath.Join(workspacePath, tc.packageName)
			if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
				t.Errorf("expected package directory %s to exist", pkgPath)
			}

			// Verify correct source file exists
			sourcePath := filepath.Join(pkgPath, tc.sourceFile)
			sourceContent, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("expected source file %s to exist: %v", sourcePath, err)
			}
			if !strings.Contains(string(sourceContent), tc.sourceContains) {
				t.Errorf("source file %s missing expected content '%s':\n%s",
					tc.sourceFile, tc.sourceContains, sourceContent)
			}

			// Verify other template source files do NOT exist
			otherSources := []string{"main.bal", "lib.bal", "service.bal"}
			for _, other := range otherSources {
				if other == tc.sourceFile {
					continue
				}
				if _, err := os.Stat(filepath.Join(pkgPath, other)); err == nil {
					t.Errorf("unexpected source file %s exists for template %s", other, tc.template)
				}
			}

			// Verify workspace toml includes the package
			tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
			content, err := os.ReadFile(tomlPath)
			if err != nil {
				t.Fatalf("failed to read workspace Ballerina.toml: %v", err)
			}
			if !strings.Contains(string(content), `"`+tc.packageName+`"`) {
				t.Errorf("workspace Ballerina.toml missing %s package:\n%s", tc.packageName, content)
			}
		})
	}
}

// TestNewPackage_WithTemplate tests creating a standalone package with different templates.
func TestNewPackage_WithTemplate(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		template       string
		sourceFile     string
		sourceContains string
	}{
		{"default", "main.bal", "public function main()"},
		{"main", "main.bal", "public function main()"},
		{"service", "service.bal", "service / on new http:Listener"},
		{"lib", "lib.bal", "public function hello(string? name) returns string"},
	}

	for _, tc := range testCases {
		t.Run(tc.template, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			pkgPath := filepath.Join(tmpDir, "mypackage")

			_, stderr, err := executeNewCommandWithArgs(t, pkgPath, "-t", tc.template)
			if err != nil {
				t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
			}

			// Verify correct source file exists
			sourcePath := filepath.Join(pkgPath, tc.sourceFile)
			sourceContent, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("expected source file %s to exist: %v", sourcePath, err)
			}
			if !strings.Contains(string(sourceContent), tc.sourceContains) {
				t.Errorf("source file %s missing expected content '%s':\n%s",
					tc.sourceFile, tc.sourceContains, sourceContent)
			}

			// Verify other template source files do NOT exist
			otherSources := []string{"main.bal", "lib.bal", "service.bal"}
			for _, other := range otherSources {
				if other == tc.sourceFile {
					continue
				}
				if _, err := os.Stat(filepath.Join(pkgPath, other)); err == nil {
					t.Errorf("unexpected source file %s exists for template %s", other, tc.template)
				}
			}
		})
	}
}

// TestNewPackage_InsideWorkspace_WithTemplate tests creating a package with template inside workspace.
func TestNewPackage_InsideWorkspace_WithTemplate(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-workspace")

	// Create a workspace first
	_, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err != nil {
		t.Fatalf("failed to create workspace: %v\nstderr: %s", err, stderr)
	}

	// Create a lib package inside the workspace
	libPkgPath := filepath.Join(workspacePath, "mylib")
	stdout, stderr, err := executeNewCommandWithArgs(t, libPkgPath, "-t", "lib")
	if err != nil {
		t.Fatalf("failed to create lib package: %v\nstderr: %s", err, stderr)
	}

	// Verify lib.bal was created (not main.bal)
	libBalPath := filepath.Join(libPkgPath, "lib.bal")
	if _, err := os.Stat(libBalPath); os.IsNotExist(err) {
		t.Errorf("expected lib.bal to be created, but not found")
	}
	mainBalPath := filepath.Join(libPkgPath, "main.bal")
	if _, err := os.Stat(mainBalPath); err == nil {
		t.Errorf("main.bal should NOT be created for lib template")
	}

	// Verify workspace toml was updated
	if !strings.Contains(stdout, "Added package to workspace") {
		t.Errorf("expected 'Added package to workspace' message, got stdout: %s", stdout)
	}

	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read workspace Ballerina.toml: %v", err)
	}
	if !strings.Contains(string(content), `"mylib"`) {
		t.Errorf("workspace Ballerina.toml missing mylib package:\n%s", content)
	}
}

// TestNewWorkspace_ConvertExisting tests converting a directory with existing packages to workspace.
func TestNewWorkspace_ConvertExisting(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-project")

	// Create existing packages
	pkgAPath := filepath.Join(workspacePath, "pkg-a")
	pkgBPath := filepath.Join(workspacePath, "pkg-b")

	if err := os.MkdirAll(pkgAPath, 0755); err != nil {
		t.Fatalf("failed to create pkg-a: %v", err)
	}
	if err := os.MkdirAll(pkgBPath, 0755); err != nil {
		t.Fatalf("failed to create pkg-b: %v", err)
	}

	// Create Ballerina.toml in each package
	pkgAToml := filepath.Join(pkgAPath, projects.BallerinaTomlFile)
	pkgBToml := filepath.Join(pkgBPath, projects.BallerinaTomlFile)
	if err := os.WriteFile(pkgAToml, []byte("[package]\norg = \"testorg\"\nname = \"pkga\"\nversion = \"1.0.0\"\n"), 0644); err != nil {
		t.Fatalf("failed to create pkg-a/Ballerina.toml: %v", err)
	}
	if err := os.WriteFile(pkgBToml, []byte("[package]\norg = \"testorg\"\nname = \"pkgb\"\nversion = \"1.0.0\"\n"), 0644); err != nil {
		t.Fatalf("failed to create pkg-b/Ballerina.toml: %v", err)
	}

	stdout, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}

	// Verify conversion message
	if !strings.Contains(stdout, "Converting directory to workspace") {
		t.Errorf("expected 'Converting directory to workspace' message, got stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Discovered 2 package(s)") {
		t.Errorf("expected 'Discovered 2 package(s)' message, got stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "pkg-a") || !strings.Contains(stdout, "pkg-b") {
		t.Errorf("expected discovered packages to be listed, got stdout: %s", stdout)
	}

	// Verify NO hello-app was created
	helloAppPath := filepath.Join(workspacePath, "hello-app")
	if _, err := os.Stat(helloAppPath); err == nil {
		t.Errorf("hello-app should NOT be created when existing packages are found")
	}

	// Verify workspace Ballerina.toml
	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read workspace Ballerina.toml: %v", err)
	}
	if !strings.Contains(string(content), "[workspace]") {
		t.Errorf("workspace Ballerina.toml missing [workspace] section:\n%s", content)
	}
	if !strings.Contains(string(content), `"pkg-a"`) {
		t.Errorf("workspace Ballerina.toml missing pkg-a:\n%s", content)
	}
	if !strings.Contains(string(content), `"pkg-b"`) {
		t.Errorf("workspace Ballerina.toml missing pkg-b:\n%s", content)
	}
}

// TestNewWorkspace_AlreadyWorkspace tests error when directory is already a workspace.
func TestNewWorkspace_AlreadyWorkspace(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "existing-workspace")

	// Create directory with workspace Ballerina.toml
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	if err := os.WriteFile(tomlPath, []byte("[workspace]\npackages = [\"pkg1\"]\n"), 0644); err != nil {
		t.Fatalf("failed to create Ballerina.toml: %v", err)
	}

	_, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err == nil {
		t.Fatal("expected error, got success")
	}

	if !strings.Contains(stderr, "directory is already a workspace") {
		t.Errorf("expected 'already a workspace' error, got: %s", stderr)
	}
}

// TestNewWorkspace_AlreadyPackage tests error when directory is already a single package.
func TestNewWorkspace_AlreadyPackage(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "existing-package")

	// Create directory with package Ballerina.toml (no [workspace] section)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	tomlPath := filepath.Join(pkgPath, projects.BallerinaTomlFile)
	if err := os.WriteFile(tomlPath, []byte("[package]\norg = \"testorg\"\nname = \"test\"\nversion = \"1.0.0\"\n"), 0644); err != nil {
		t.Fatalf("failed to create Ballerina.toml: %v", err)
	}

	_, stderr, err := executeNewCommandWithArgs(t, pkgPath, "--workspace")
	if err == nil {
		t.Fatal("expected error, got success")
	}

	if !strings.Contains(stderr, "directory is already a Ballerina package") {
		t.Errorf("expected 'already a Ballerina package' error, got: %s", stderr)
	}
}

// TestNewWorkspace_LoadsCorrectly tests that a created workspace can be loaded with projects.Load().
func TestNewWorkspace_LoadsCorrectly(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-workspace")

	// Create workspace
	_, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}

	// Load the workspace with projects.Load()
	// Use DirFS rooted at workspace path with relative path "."
	fsys := os.DirFS(workspacePath)
	userHome, _ := os.UserHomeDir()
	ballerinaEnv := filepath.Join(userHome, projects.UserHomeDirName)
	ballerinaEnvFs := os.DirFS(ballerinaEnv)

	result, err := projects.Load(fsys, ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFs,
	})
	if err != nil {
		t.Fatalf("failed to load workspace: %v", err)
	}

	// Verify it's a workspace project
	project := result.Project()
	if project.Kind() != projects.ProjectKindWorkspace {
		t.Errorf("expected ProjectKindWorkspace, got: %v", project.Kind())
	}

	// Cast to WorkspaceProject and verify it has packages
	workspace, ok := project.(*projects.WorkspaceProject)
	if !ok {
		t.Fatalf("expected *projects.WorkspaceProject, got: %T", project)
	}

	// Verify workspace has 1 package (hello-app)
	if len(workspace.Manifest().Packages()) != 1 {
		t.Errorf("expected 1 package in workspace, got: %d", len(workspace.Manifest().Packages()))
	}
	if len(workspace.Projects()) != 1 {
		t.Errorf("expected 1 project in workspace, got: %d", len(workspace.Projects()))
	}
}

// TestNewPackage_InsideWorkspace tests creating a package inside an existing workspace.
// The package should be automatically added to the workspace's packages list.
func TestNewPackage_InsideWorkspace(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-workspace")

	// First, create a workspace
	_, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err != nil {
		t.Fatalf("failed to create workspace: %v\nstderr: %s", err, stderr)
	}

	// Now create a new package inside the workspace
	newPkgPath := filepath.Join(workspacePath, "new-pkg")
	stdout, stderr, err := executeNewCommandWithArgs(t, newPkgPath)
	if err != nil {
		t.Fatalf("failed to create package inside workspace: %v\nstderr: %s", err, stderr)
	}

	// Verify success message
	if !strings.Contains(stdout, "Created new package") {
		t.Errorf("expected 'Created new package' message, got stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Added package to workspace") {
		t.Errorf("expected 'Added package to workspace' message, got stdout: %s", stdout)
	}

	// Verify the package was created
	assertPackageStructure(t, newPkgPath)

	// Verify the workspace Ballerina.toml now includes the new package
	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read workspace Ballerina.toml: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"hello-app"`) {
		t.Errorf("workspace Ballerina.toml should still contain hello-app:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `"new-pkg"`) {
		t.Errorf("workspace Ballerina.toml should contain new-pkg:\n%s", contentStr)
	}

	// Verify the new package uses the same org as the workspace
	newPkgTomlPath := filepath.Join(newPkgPath, projects.BallerinaTomlFile)
	newPkgContent, err := os.ReadFile(newPkgTomlPath)
	if err != nil {
		t.Fatalf("failed to read new package Ballerina.toml: %v", err)
	}

	// Check that org name is consistent (both should have the same org)
	helloAppTomlPath := filepath.Join(workspacePath, "hello-app", projects.BallerinaTomlFile)
	helloAppContent, err := os.ReadFile(helloAppTomlPath)
	if err != nil {
		t.Fatalf("failed to read hello-app Ballerina.toml: %v", err)
	}

	// Extract org from both
	getOrg := func(content string) string {
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "org") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.Trim(strings.TrimSpace(parts[1]), "\"")
				}
			}
		}
		return ""
	}

	helloAppOrg := getOrg(string(helloAppContent))
	newPkgOrg := getOrg(string(newPkgContent))

	if helloAppOrg != newPkgOrg {
		t.Errorf("org names should match: hello-app has '%s', new-pkg has '%s'", helloAppOrg, newPkgOrg)
	}
}

// TestNewPackage_InsideWorkspace_NestedDir tests creating a package in a nested directory inside a workspace.
func TestNewPackage_InsideWorkspace_NestedDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-workspace")

	// First, create a workspace
	_, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err != nil {
		t.Fatalf("failed to create workspace: %v\nstderr: %s", err, stderr)
	}

	// Create a nested directory structure for the new package
	nestedPkgPath := filepath.Join(workspacePath, "packages", "nested-pkg")
	stdout, stderr, err := executeNewCommandWithArgs(t, nestedPkgPath)
	if err != nil {
		t.Fatalf("failed to create nested package inside workspace: %v\nstderr: %s", err, stderr)
	}

	// Verify success message
	if !strings.Contains(stdout, "Created new package") {
		t.Errorf("expected 'Created new package' message, got stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "Added package to workspace") {
		t.Errorf("expected 'Added package to workspace' message, got stdout: %s", stdout)
	}

	// Verify the workspace Ballerina.toml includes the nested package path
	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read workspace Ballerina.toml: %v", err)
	}

	contentStr := string(content)
	// Workspace TOML always stores forward-slash paths regardless of platform.
	expectedPath := "packages/nested-pkg"
	if !strings.Contains(contentStr, expectedPath) {
		t.Errorf("workspace Ballerina.toml should contain '%s':\n%s", expectedPath, contentStr)
	}
}

// TestNewWorkspace_SkipsHiddenDirs tests that hidden directories are not discovered as packages.
func TestNewWorkspace_SkipsHiddenDirs(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "my-project")

	// Create a normal package
	pkgPath := filepath.Join(workspacePath, "pkg-a")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("failed to create pkg-a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgPath, projects.BallerinaTomlFile),
		[]byte("[package]\norg = \"testorg\"\nname = \"pkga\"\nversion = \"1.0.0\"\n"), 0644); err != nil {
		t.Fatalf("failed to create Ballerina.toml: %v", err)
	}

	// Create a hidden directory with Ballerina.toml (should be ignored)
	hiddenPath := filepath.Join(workspacePath, ".hidden-pkg")
	if err := os.MkdirAll(hiddenPath, 0755); err != nil {
		t.Fatalf("failed to create .hidden-pkg: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenPath, projects.BallerinaTomlFile),
		[]byte("[package]\norg = \"testorg\"\nname = \"hidden\"\nversion = \"1.0.0\"\n"), 0644); err != nil {
		t.Fatalf("failed to create Ballerina.toml: %v", err)
	}

	stdout, stderr, err := executeNewCommandWithArgs(t, workspacePath, "--workspace")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}

	// Verify only pkg-a was discovered
	if !strings.Contains(stdout, "Discovered 1 package(s)") {
		t.Errorf("expected 'Discovered 1 package(s)' message, got stdout: %s", stdout)
	}
	if strings.Contains(stdout, ".hidden-pkg") {
		t.Errorf("hidden directory should not be discovered, got stdout: %s", stdout)
	}

	// Verify workspace toml only has pkg-a
	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read workspace Ballerina.toml: %v", err)
	}
	if strings.Contains(string(content), ".hidden-pkg") {
		t.Errorf("workspace Ballerina.toml should not contain hidden directory:\n%s", content)
	}
}

// =============================================================================
// Help
// =============================================================================

// TestNewCommandWithHelp tests the help flag.
func TestNewCommandWithHelp(t *testing.T) {
	t.Parallel()
	stdout, _, _ := executeNewCommandWithArgs(t, "--help")

	if !strings.Contains(stdout, "Create a new Ballerina package") {
		t.Errorf("expected help text, got: %s", stdout)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// executeNewCommand executes the new command with a single path argument.
func executeNewCommand(t *testing.T, projectPath string) (stdout, stderr string, err error) {
	t.Helper()
	return executeNewCommandWithArgs(t, projectPath)
}

// executeNewCommandWithArgs executes the new command with the given arguments.
// Creates a fresh command instance to support parallel test execution.
func executeNewCommandWithArgs(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	// Create fresh command instance for parallel safety
	// Each command instance has its own local options
	cmd := createNewCmd()

	// Capture stdout and stderr using cobra's built-in support
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	// Set arguments and execute
	cmd.SetArgs(args)
	err = cmd.Execute()

	return outBuf.String(), errBuf.String(), err
}

// assertPackageStructure verifies the expected package structure exists.
func assertPackageStructure(t *testing.T, projectPath string) {
	t.Helper()

	// Verify directory exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("project directory does not exist: %s", projectPath)
		return
	}

	// Verify Ballerina.toml exists
	tomlPath := filepath.Join(projectPath, projects.BallerinaTomlFile)
	if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
		t.Errorf("Ballerina.toml does not exist: %s", tomlPath)
	}

	// Verify main.bal exists
	mainPath := filepath.Join(projectPath, "main.bal")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Errorf("main.bal does not exist: %s", mainPath)
	}

	// Verify .gitignore exists
	gitignorePath := filepath.Join(projectPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Errorf(".gitignore does not exist: %s", gitignorePath)
	}

	// Verify Package.md does NOT exist (default template)
	packageMdPath := filepath.Join(projectPath, "Package.md")
	if _, err := os.Stat(packageMdPath); err == nil {
		t.Errorf("Package.md should not exist for default template")
	}
}
