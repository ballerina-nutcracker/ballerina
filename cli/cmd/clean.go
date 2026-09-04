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

	"github.com/ballerina-nutcracker/ballerina/projects"

	"github.com/spf13/cobra"
)

// cleanError returns an error formatted with the clean-command USAGE block.
// Cobra prefixes it with "ballerina:" when RunE returns.
func cleanError(format string, args ...any) error {
	return usageError("clean", format, args...)
}

var cleanCmd = createCleanCmd()

// createCleanCmd creates a new instance of the 'clean' command.
// This factory function enables parallel test execution.
func createCleanCmd() *cobra.Command {
	var targetDir string

	cmd := &cobra.Command{
		Use:   "clean [<package-dir>]",
		Short: "Clean the artifacts generated during the build",
		Long: `	Clean the artifacts generated during the build.

	Deletes the 'target' directory produced by 'bal pack'/'bal run', for the
	current directory or the given package directory. If the project is a
	workspace, every member package's target directory is deleted, along
	with the workspace's own. It's not an error for the target directory to
	already be absent.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(cmd, args, targetDir)
		},
	}

	cmd.Flags().StringVar(&targetDir, "target-dir", "", "target directory path")

	return cmd
}

// runClean executes the 'clean' command.
// Java source: io.ballerina.cli.cmd.CleanCommand
func runClean(cmd *cobra.Command, args []string, targetDir string) error {
	if targetDir != "" {
		return cleanCustomTargetDir(cmd, targetDir)
	}

	// path here mirrors pack.go's own handling: defaults to the process cwd
	// when no positional arg is given.
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		return cleanError("invalid project path %q: %w", path, err)
	}

	// A non-directory path is loaded the same way run.go handles single .bal
	// files: root the fs.FS at the parent directory and pass the file's own
	// name as the load path, so projects.Load can classify it (ProjectKindSingleFile
	// for a .bal file, or a load error for anything else) instead of us
	// guessing from the path alone.
	baseDir := path
	loadPath := "."
	if !info.IsDir() {
		baseDir = filepath.Dir(path)
		loadPath = filepath.Base(path)
	}

	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return cleanError("resolve absolute path: %w", err)
	}

	fsys := os.DirFS(absPath)
	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		return cleanError("resolve ballerina env path: %w", err)
	}

	buildOpts := projects.NewBuildOptionsBuilder().Build()
	result, err := projects.Load(fsys, loadPath, projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		return cleanError("invalid project directory: %w", err)
	}

	project := result.Project()
	switch project.Kind() {
	case projects.ProjectKindSingleFile:
		return cleanError("clean command is not supported for single file projects")
	case projects.ProjectKindWorkspace:
		ws, ok := project.(*projects.WorkspaceProject)
		if !ok {
			return cleanError("unexpected workspace project type %T", project)
		}
		for _, member := range ws.Projects() {
			if err := cleanDir(cmd, filepath.Join(absPath, member.TargetDir())); err != nil {
				return err
			}
		}
		return cleanDir(cmd, filepath.Join(absPath, ws.TargetDir()))
	default:
		return cleanDir(cmd, filepath.Join(absPath, project.TargetDir()))
	}
}

// cleanDir deletes dir if it exists. A missing dir is a silent no-op
// success, matching Java's CleanCommand#cleanProject.
func cleanDir(cmd *cobra.Command, dir string) error {
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return cleanError("failed to delete %s: %w", dir, err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Successfully deleted "+dir); err != nil {
		return cleanError("failed to write success message: %w", err)
	}
	return nil
}

// cleanCustomTargetDir handles `bal clean --target-dir <path>`: it operates
// directly on the given path without loading a project at all, after
// validating it looks like a real target directory.
func cleanCustomTargetDir(cmd *cobra.Command, targetDir string) error {
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return cleanError("resolve absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return cleanError("provided target directory '%s' does not exist.", absPath)
	}
	if err != nil {
		return cleanError("failed to stat %s: %w", absPath, err)
	}
	if !info.IsDir() {
		return cleanError("provided target path '%s' is not a directory.", absPath)
	}
	if !isValidTargetDir(absPath) {
		return cleanError("provided target directory '%s' is not a valid target directory.", absPath)
	}

	if err := os.RemoveAll(absPath); err != nil {
		return cleanError("failed to delete %s: %w", absPath, err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Successfully deleted '"+absPath+"'"); err != nil {
		return cleanError("failed to write success message: %w", err)
	}
	return nil
}

// isValidTargetDir recognizes a real target directory by checking for at
// least one of its expected contents, to avoid deleting an arbitrary
// directory by mistake. Unlike Java's CleanCommand (which requires a
// cache/ subdirectory unconditionally, since its build pipeline always
// creates one), this port's bal pack currently only ever produces
// target/bala/ — no cache/ directory — so cache/ is treated as one
// recognized marker among several rather than a mandatory gate.
func isValidTargetDir(dir string) bool {
	for _, sub := range []string{projects.CacheDir, "bala", "bin", "apidocs", filepath.Join(projects.CacheDir, "tests_cache")} {
		if info, err := os.Stat(filepath.Join(dir, sub)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}
