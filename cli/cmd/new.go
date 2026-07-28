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
	"sort"
	"strings"

	"ballerina/cli/templates"
	"ballerina/common/tomlparser"
	"ballerina/projects"

	"github.com/spf13/cobra"
)

var newCmd = createNewCmd()

// createNewCmd creates a new instance of the 'new' command.
// This factory function enables parallel test execution.
func createNewCmd() *cobra.Command {
	// Local options for this command instance (avoids global state for parallel tests)
	var workspace bool
	var template string

	cmd := &cobra.Command{
		Use:   "new <package-path>",
		Short: "Create a new Ballerina package",
		Long: `	Create a new Ballerina package or workspace.

	Creates the given path if it does not exist and initializes a Ballerina
	package in it. It generates the Ballerina.toml, main.bal, and .gitignore
	files inside the package directory. However, for existing paths, the
	main.bal file is only created if there are no other Ballerina source
	files (.bal) in the directory.

	The package directory will have the structure below.
		.
		├── Ballerina.toml
		├── .gitignore
		└── main.bal

	Any directory becomes a Ballerina package if that directory has a
	'Ballerina.toml' file. It contains the organization name, package name,
	and the version. The package root directory is the default module
	directory.

	Use the --workspace flag to create a workspace project containing
	multiple packages. If the target directory already contains Ballerina
	packages, they will be discovered and added to the workspace.`,
		Args: validateNewArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl, err := validateTemplate(template)
			if err != nil {
				printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
				return err
			}
			return runNew(cmd, args, workspace, tmpl)
		},
	}

	cmd.Flags().BoolVar(&workspace, "workspace", false, "")
	cmd.Flags().StringVarP(&template, "template", "t", string(templateDefault),
		fmt.Sprintf("Acceptable values: %v default: %s", validTemplates, templateDefault))

	return cmd
}

// templateName is a validated --template flag value.
type templateName string

const (
	templateDefault templateName = "default"
	templateMain    templateName = "main"
	templateService templateName = "service"
	templateLib     templateName = "lib"
)

// validTemplates is the closed set of accepted --template values.
var validTemplates = []templateName{templateDefault, templateMain, templateService, templateLib}

// validateTemplate ensures the raw --template flag value is one of the
// accepted templates and returns the typed equivalent.
func validateTemplate(raw string) (templateName, error) {
	for _, t := range validTemplates {
		if string(t) == raw {
			return t, nil
		}
	}
	return "", fmt.Errorf("invalid template '%s'. Acceptable values: %v", raw, validTemplates)
}

// validateNewArgs validates the arguments for the 'new' command.
func validateNewArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := fmt.Errorf("project path is not provided")
		printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
		return err
	}
	if len(args) > 1 {
		err := fmt.Errorf("too many arguments")
		printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
		return err
	}
	return nil
}

// runNew executes the 'new' command.
func runNew(cmd *cobra.Command, args []string, workspace bool, template templateName) error {
	projectPath := args[0]

	// Convert to absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		printErrorTo(cmd.ErrOrStderr(), fmt.Errorf("invalid path: %w", err), "new <project-path>", false)
		return err
	}

	if workspace {
		return runNewWorkspace(cmd, absPath, projectPath, template)
	}
	return runNewPackage(cmd, absPath, projectPath, template)
}

// runNewPackage creates a new single Ballerina package.
func runNewPackage(cmd *cobra.Command, absPath, projectPath string, template templateName) error {
	// Derive package name from directory name
	packageName := filepath.Base(absPath)

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err == nil {
		// Directory exists - check for conflicts
		if !info.IsDir() {
			err := fmt.Errorf("path exists and is not a directory: %s", absPath)
			printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
			return err
		}

		if err := checkExistingDirectory(absPath); err != nil {
			printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
			return err
		}
	} else if !os.IsNotExist(err) {
		// Some other error (not "does not exist")
		err := fmt.Errorf("error checking path: %w", err)
		printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
		return err
	}
	// If path doesn't exist, it will be created by initPackage (including parent dirs)

	// Validate or guess package name
	var nameWarning string
	if !validatePackageName(packageName) {
		guessedName := guessPkgName(packageName)
		nameWarning = fmt.Sprintf("package name is derived as '%s'. Edit the Ballerina.toml to change it.", guessedName)
		packageName = guessedName
	}

	// Check if we're inside a workspace - use parent of package path
	workspaceRoot := findWorkspaceRoot(filepath.Dir(absPath))

	// Get organization name (from workspace if available, otherwise guess)
	orgName := guessOrgName()
	if workspaceRoot != "" {
		if wsOrgName := getOrgNameFromWorkspace(workspaceRoot); wsOrgName != "" {
			orgName = wsOrgName
		}
	}

	// Create the package
	if err := initPackage(absPath, packageName, orgName, template); err != nil {
		printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
		return err
	}

	// Print success message
	if nameWarning != "" {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), nameWarning)
	}

	// Use relative path in output if originally provided as relative
	displayPath := projectPath
	if filepath.IsAbs(projectPath) {
		displayPath = absPath
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created new package '%s' at %s.\n", packageName, displayPath)

	// If inside a workspace, add the package to the workspace
	if workspaceRoot != "" {
		relPath, err := filepath.Rel(workspaceRoot, absPath)
		if err != nil {
			printErrorTo(cmd.ErrOrStderr(), fmt.Errorf("failed to compute relative path: %w", err), "new <project-path>", false)
			return err
		}
		if err := addPackageToWorkspace(workspaceRoot, relPath); err != nil {
			printErrorTo(cmd.ErrOrStderr(), err, "new <project-path>", false)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added package to workspace at %s.\n", workspaceRoot)
	}

	return nil
}

// runNewWorkspace creates a new workspace project or converts an existing directory to workspace.
func runNewWorkspace(cmd *cobra.Command, absPath, projectPath string, template templateName) error {
	// Validate path
	if err := validateWorkspacePath(absPath); err != nil {
		printErrorTo(cmd.ErrOrStderr(), err, "new --workspace <path>", false)
		return err
	}

	// Create directory if needed
	if err := os.MkdirAll(absPath, 0755); err != nil {
		printErrorTo(cmd.ErrOrStderr(), err, "new --workspace <path>", false)
		return err
	}

	// Discover existing packages
	existingPkgs := discoverExistingPackages(absPath)

	var packages []string
	if len(existingPkgs) == 0 {
		// New workspace - create sample package
		pkgName := getWorkspacePackageName(template)
		pkgPath := filepath.Join(absPath, pkgName)
		orgName := guessOrgName()

		if err := initPackage(pkgPath, pkgName, orgName, template); err != nil {
			printErrorTo(cmd.ErrOrStderr(), err, "new --workspace <path>", false)
			return err
		}
		packages = []string{pkgName}

		// Use relative path in output if originally provided as relative
		displayPath := projectPath
		if filepath.IsAbs(projectPath) {
			displayPath = absPath
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created new workspace at %s.\n", displayPath)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created new package '%s' at %s.\n", pkgName, filepath.Join(displayPath, pkgName))
	} else {
		// Convert existing directory to workspace
		packages = existingPkgs

		// Use relative path in output if originally provided as relative
		displayPath := projectPath
		if filepath.IsAbs(projectPath) {
			displayPath = absPath
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Converting directory to workspace at %s.\n", displayPath)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Discovered %d package(s): %s\n",
			len(packages), strings.Join(packages, ", "))
	}

	// Write workspace Ballerina.toml
	if err := writeWorkspaceToml(absPath, packages); err != nil {
		printErrorTo(cmd.ErrOrStderr(), err, "new --workspace <path>", false)
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Workspace created successfully.")
	return nil
}

// discoverExistingPackages walks the workspace directory recursively and
// returns workspace-relative paths to every directory that contains a
// Ballerina.toml. Hidden directories (names starting with ".") are skipped.
// Paths are returned forward-slash-normalized (matching the manifest's
// stored format) and sorted for deterministic output.
//
// Workspace member paths can be nested (for example, "packages/pkg-a"), so
// this function must descend into subdirectories rather than only inspect
// immediate children.
func discoverExistingPackages(workspacePath string) []string {
	var packages []string

	_ = filepath.WalkDir(workspacePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != workspacePath && strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		// The workspace root itself is not a member; skip its Ballerina.toml.
		if path == workspacePath {
			return nil
		}
		tomlPath := filepath.Join(path, projects.BallerinaTomlFile)
		if _, err := os.Stat(tomlPath); err != nil {
			return nil
		}
		rel, err := filepath.Rel(workspacePath, path)
		if err != nil {
			return nil
		}
		packages = append(packages, filepath.ToSlash(rel))
		// A package directory does not contain another package; stop descending.
		return filepath.SkipDir
	})

	sort.Strings(packages)
	return packages
}

// writeWorkspaceToml creates the workspace Ballerina.toml file.
func writeWorkspaceToml(workspacePath string, packages []string) error {
	var quotedPkgs []string
	for _, pkg := range packages {
		quotedPkgs = append(quotedPkgs, fmt.Sprintf("%q", pkg))
	}

	content := fmt.Sprintf("[workspace]\npackages = [%s]\n",
		strings.Join(quotedPkgs, ", "))

	tomlPath := filepath.Join(workspacePath, projects.BallerinaTomlFile)
	return os.WriteFile(tomlPath, []byte(content), 0644)
}

// getWorkspacePackageName returns the sample package name based on the template.
func getWorkspacePackageName(template templateName) string {
	switch template {
	case templateService:
		return "hello-service"
	case templateLib:
		return "hello-lib"
	default:
		return "hello-app"
	}
}

// validateWorkspacePath validates that the path can be used for a new workspace.
func validateWorkspacePath(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil // New directory - OK
	}
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("path exists and is not a directory: %s", path)
	}

	// Check if Ballerina.toml exists at root
	tomlPath := filepath.Join(path, projects.BallerinaTomlFile)
	if _, err := os.Stat(tomlPath); err == nil {
		if isWorkspaceToml(tomlPath) {
			return fmt.Errorf("directory is already a workspace: %s", path)
		}
		return fmt.Errorf("directory is already a Ballerina package: %s\n"+
			"To create a workspace containing this package, run from the parent directory", path)
	}

	return nil
}

// isWorkspaceToml reports whether the file at tomlPath defines a top-level [workspace] table.
// IO errors and parse errors return false so malformed or missing files don't get classified
// as workspaces.
func isWorkspaceToml(tomlPath string) bool {
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		return false
	}
	t, err := tomlparser.ReadString(string(content))
	if err != nil {
		return false
	}
	_, ok := t.GetTable("workspace")
	return ok
}

// findWorkspaceRoot searches for a workspace root starting from the given path.
// Returns the workspace root path if found, or empty string if not inside a workspace.
func findWorkspaceRoot(startPath string) string {
	current := startPath
	for {
		tomlPath := filepath.Join(current, projects.BallerinaTomlFile)
		if _, err := os.Stat(tomlPath); err == nil {
			if isWorkspaceToml(tomlPath) {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return ""
		}
		current = parent
	}
}

// getOrgNameFromWorkspace gets the organization name from the first package
// declared in the workspace manifest. Reading the manifest (rather than scanning
// immediate child directories) ensures nested member paths like
// `packages = ["nested/pkg-a"]` are honored.
func getOrgNameFromWorkspace(workspaceRoot string) string {
	wsToml, err := os.ReadFile(filepath.Join(workspaceRoot, projects.BallerinaTomlFile))
	if err != nil {
		return ""
	}
	packages := parseWorkspacePackages(string(wsToml))
	if len(packages) == 0 {
		return ""
	}

	// packages stores forward-slash paths; convert to OS-native for filesystem access.
	tomlPath := filepath.Join(workspaceRoot, filepath.FromSlash(packages[0]), projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		return ""
	}

	// Simple parsing to extract org name
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "org") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				orgName := strings.TrimSpace(parts[1])
				orgName = strings.Trim(orgName, "\"")
				return orgName
			}
		}
	}
	return ""
}

// addPackageToWorkspace adds a package to the workspace's packages list.
func addPackageToWorkspace(workspaceRoot, packagePath string) error {
	tomlPath := filepath.Join(workspaceRoot, projects.BallerinaTomlFile)
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		return fmt.Errorf("failed to read workspace Ballerina.toml: %w", err)
	}

	// Always store forward-slash paths in the workspace TOML so the file is portable
	// across platforms.
	packagePath = filepath.ToSlash(packagePath)

	contentStr := string(content)

	// Parse existing packages from the TOML
	existingPackages := parseWorkspacePackages(contentStr)

	// Check if package is already in the list
	for _, pkg := range existingPackages {
		if pkg == packagePath {
			return nil // Already exists
		}
	}

	// Add the new package
	existingPackages = append(existingPackages, packagePath)
	sort.Strings(existingPackages)

	// Build the new packages array
	var quotedPkgs []string
	for _, pkg := range existingPackages {
		quotedPkgs = append(quotedPkgs, fmt.Sprintf("%q", pkg))
	}
	newPackagesLine := fmt.Sprintf("packages = [%s]", strings.Join(quotedPkgs, ", "))

	// Replace the packages line in the content
	lines := strings.Split(contentStr, "\n")
	var newLines []string
	packagesReplaced := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "packages") && strings.Contains(trimmed, "=") {
			newLines = append(newLines, newPackagesLine)
			packagesReplaced = true
		} else {
			newLines = append(newLines, line)
		}
	}

	// If packages line wasn't found, append it after [workspace]
	if !packagesReplaced {
		for i, line := range newLines {
			if strings.TrimSpace(line) == "[workspace]" {
				// Insert packages line after [workspace]
				newLines = append(newLines[:i+1], append([]string{newPackagesLine}, newLines[i+1:]...)...)
				break
			}
		}
	}

	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(tomlPath, []byte(newContent), 0644)
}

// parseWorkspacePackages extracts the workspace.packages array from a workspace
// Ballerina.toml. Uses the in-repo TOML parser so multi-line arrays, comments
// and quoting variants all decode correctly.
func parseWorkspacePackages(content string) []string {
	t, err := tomlparser.ReadString(content)
	if err != nil {
		return nil
	}
	ws, ok := t.GetTable("workspace")
	if !ok {
		return nil
	}
	raw, ok := ws.GetArray("packages")
	if !ok {
		return nil
	}
	var packages []string
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			packages = append(packages, s)
		}
	}
	return packages
}

// checkExistingDirectory validates that an existing directory can be used for a new package.
func checkExistingDirectory(path string) error {
	// Check for Ballerina.toml (already a project)
	ballerinaToml := filepath.Join(path, projects.BallerinaTomlFile)
	if _, err := os.Stat(ballerinaToml); err == nil {
		return fmt.Errorf("directory is already a Ballerina project: %s", path)
	}

	// Check for conflicting files
	conflictingFiles := []string{
		"Dependencies.toml",
		"BalTool.toml",
		"Package.md",
		"Module.md",
		projects.ModulesDir,
		projects.TestsDir,
	}

	var found []string
	for _, name := range conflictingFiles {
		if _, err := os.Stat(filepath.Join(path, name)); err == nil {
			found = append(found, name)
		}
	}

	if len(found) > 0 {
		return fmt.Errorf("existing %s file/directory(s) were found. Please use a different directory to create the package",
			strings.Join(found, ", "))
	}

	return nil
}

// hasExistingBalFiles checks if the directory contains any .bal files.
func hasExistingBalFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), projects.BalFileExtension) {
			return true
		}
	}
	return false
}

// initPackage creates a new Ballerina package at the specified path.
func initPackage(projectPath, packageName, orgName string, template templateName) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Track created files for cleanup on error
	var createdFiles []string
	cleanup := func() {
		for i := len(createdFiles) - 1; i >= 0; i-- {
			_ = os.Remove(createdFiles[i])
		}
	}

	// Create Ballerina.toml
	manifestContent, err := templates.ReadTemplate(templates.ManifestApp)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to read manifest template: %w", err)
	}
	manifestContent = strings.ReplaceAll(manifestContent, templates.OrgNamePlaceholder, orgName)
	manifestContent = strings.ReplaceAll(manifestContent, templates.PkgNamePlaceholder, packageName)

	ballerinaToml := filepath.Join(projectPath, projects.BallerinaTomlFile)
	if err := os.WriteFile(ballerinaToml, []byte(manifestContent), 0644); err != nil {
		cleanup()
		return fmt.Errorf("failed to create Ballerina.toml: %w", err)
	}
	createdFiles = append(createdFiles, ballerinaToml)

	// Create source file based on template (only if no existing .bal files)
	if !hasExistingBalFiles(projectPath) {
		sourceFile, sourceContent, err := getTemplateSource(template)
		if err != nil {
			cleanup()
			return fmt.Errorf("failed to read template: %w", err)
		}

		sourcePath := filepath.Join(projectPath, sourceFile)
		if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
			cleanup()
			return fmt.Errorf("failed to create %s: %w", sourceFile, err)
		}
		createdFiles = append(createdFiles, sourcePath)
	}

	// Create .gitignore
	gitignoreContent, err := templates.ReadTemplate(templates.Gitignore)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to read gitignore template: %w", err)
	}

	gitignore := filepath.Join(projectPath, ".gitignore")
	if err := os.WriteFile(gitignore, []byte(gitignoreContent), 0644); err != nil {
		cleanup()
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	createdFiles = append(createdFiles, gitignore)

	return nil
}

// getTemplateSource returns the source file name and content for the given template.
func getTemplateSource(template templateName) (fileName string, content string, err error) {
	switch template {
	case templateLib:
		content, err = templates.ReadTemplate(templates.LibBal)
		return "lib.bal", content, err
	case templateService:
		content, err = templates.ReadTemplate(templates.ServiceBal)
		return "service.bal", content, err
	default: // templateDefault, templateMain
		content, err = templates.ReadTemplate(templates.MainBal)
		return "main.bal", content, err
	}
}
