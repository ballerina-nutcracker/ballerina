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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/cli/templates"
	"github.com/ballerina-nutcracker/ballerina/projects"

	"github.com/spf13/cobra"
)

// addError returns an error formatted with the add-command USAGE block.
// Cobra prefixes it with "ballerina:" when RunE returns.
func addError(format string, args ...any) error {
	return usageError("add <module-name>", format, args...)
}

var addCmd = createAddCmd()

// createAddCmd creates a new instance of the 'add' command.
// This factory function enables parallel test execution.
func createAddCmd() *cobra.Command {
	var template string

	cmd := &cobra.Command{
		Use:   "add <module-name>",
		Short: "Add a new module to the current package",
		Long: `	Add a new module to the current package.

	Creates modules/<module-name>/<module-name>.bal from the given
	template. Must be run inside an existing Ballerina package; it does not
	create a package or modify Ballerina.toml.

	The resulting module's fully-qualified name is <package-name>.<module-name>.
	Use that qualified name to import it:
		import <org-name>/<package-name>.<module-name>;

	Module names can contain only alphanumerics, underscores, and periods,
	and the maximum length is 256 characters.`,
		Args: validateAddArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, args[0], template)
		},
	}

	cmd.Flags().StringVarP(&template, "template", "t", string(addTemplateLib),
		fmt.Sprintf("Acceptable values: %v default: %s", validAddTemplates, addTemplateLib))

	return cmd
}

// addTemplateName is a validated --template flag value for 'bal add'.
type addTemplateName string

const (
	addTemplateLib     addTemplateName = "lib"
	addTemplateService addTemplateName = "service"
)

// validAddTemplates is the closed set of templates 'bal add' accepts.
var validAddTemplates = []addTemplateName{addTemplateLib, addTemplateService}

// validateAddTemplate ensures the raw --template flag value is one 'bal add'
// accepts (case-insensitive) and returns the typed equivalent.
func validateAddTemplate(raw string) (addTemplateName, error) {
	lower := addTemplateName(strings.ToLower(raw))
	for _, t := range validAddTemplates {
		if t == lower {
			return t, nil
		}
	}
	return "", fmt.Errorf("unsupported template provided. run 'bal add --help' to see available templates")
}

// validateAddArgs validates the arguments for the 'add' command.
func validateAddArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return addError("module name is not provided")
	}
	if len(args) > 1 {
		return addError("too many arguments")
	}
	return nil
}

// runAdd executes the 'add' command: it scaffolds a new module inside the
// package rooted at the current directory.
// Java source: io.ballerina.cli.cmd.AddCommand
func runAdd(cmd *cobra.Command, moduleName, template string) error {
	if _, err := os.Stat(projects.BallerinaTomlFile); err != nil {
		return addError("not a Ballerina project\nYou should run this command inside a Ballerina project.")
	}

	if err := validateModuleName(moduleName); err != nil {
		return addError("%w", err)
	}

	modulePath := filepath.Join(projects.ModulesDir, moduleName)

	tmpl, err := validateAddTemplate(template)
	if err != nil {
		return addError("%w", err)
	}

	sourceContent, err := getAddTemplateSource(tmpl)
	if err != nil {
		return addError("failed to read template: %w", err)
	}

	if err := createModule(modulePath, moduleName, sourceContent); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return addError("a module already exists with the given name : '%s' :\nExisting module path %s", moduleName, modulePath)
		}
		return addError("error occurred while creating module : %w", err)
	}

	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Added new Ballerina module at "+modulePath); err != nil {
		return addError("failed to write success message: %w", err)
	}
	return nil
}

// getAddTemplateSource returns the module source file content for the given
// template.
func getAddTemplateSource(template addTemplateName) (sourceContent string, err error) {
	switch template {
	case addTemplateService:
		return templates.ReadTemplate(templates.ServiceBal)
	default: // addTemplateLib
		return templates.ReadTemplate(templates.LibBal)
	}
}

// createModule atomically claims modulePath — os.Mkdir (not MkdirAll) fails
// with fs.ErrExist if another process created it first, instead of silently
// succeeding on an existing directory — then writes
// modulePath/<moduleName>.bal. Since this call is the one that created
// modulePath, it's always safe to remove on a later write failure: it can
// never belong to a concurrent invocation. modulePath's parent (normally
// projects.ModulesDir) is derived from modulePath itself rather than
// hardcoded, so this stays correct for any modulePath a caller passes.
func createModule(modulePath, moduleName, sourceContent string) error {
	if err := os.MkdirAll(filepath.Dir(modulePath), 0755); err != nil {
		return fmt.Errorf("failed to create modules directory: %w", err)
	}
	if err := os.Mkdir(modulePath, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	sourcePath := filepath.Join(modulePath, moduleName+".bal")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		_ = os.RemoveAll(modulePath)
		return fmt.Errorf("failed to create %s: %w", sourcePath, err)
	}

	return nil
}

// validateModuleName reports the same per-rule diagnostics as Java's
// ProjectUtils.validateModuleName / getValidateUnderscoreError, applied in
// the same order: character set, length, then underscore placement, then
// leading digit. Module names follow the same identifier rules as package
// names, so this reuses the regexes already defined for that in
// name_utils.go. Wording otherwise mirrors Java, but drops its trailing
// periods to satisfy staticcheck's ST1005.
// Java source: io.ballerina.projects.util.ProjectUtils
func validateModuleName(name string) error {
	if !validPackageNamePattern.MatchString(name) || allDotsPattern.MatchString(name) {
		return fmt.Errorf("invalid module name : '%s' :\nModule name can only contain alphanumerics, underscores and periods", name)
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("invalid module name : '%s' :\nMaximum length of module name is 256 characters", name)
	}
	if strings.HasPrefix(name, "_") {
		return fmt.Errorf("invalid module name : '%s' :\nModule name cannot have initial underscore characters", name)
	}
	if strings.HasSuffix(name, "_") {
		return fmt.Errorf("invalid module name : '%s' :\nModule name cannot have trailing underscore characters", name)
	}
	if consecutiveUnderscoresPattern.MatchString(name) {
		return fmt.Errorf("invalid module name : '%s' :\nModule name cannot have consecutive underscore characters", name)
	}
	if startsWithDigitPattern.MatchString(name) {
		return fmt.Errorf("invalid module name : '%s' :\nModule name cannot have initial numeric characters", name)
	}
	return nil
}
