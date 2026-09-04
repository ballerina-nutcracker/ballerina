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
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
)

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
