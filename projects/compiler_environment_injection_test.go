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

package projects_test

import (
	"path/filepath"
	"testing"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util"
)

// TestInjectedCompilerEnvironmentIsShared asserts the environment supplied
// through ProjectLoadConfig is the one every compilation context references,
// so a single recording covers the whole compilation.
//
// The CLI trace tests already prove this end to end for the shapes 'bal run'
// reaches: a dependency's or workspace member's spans can only land in the
// root's trace if they share the environment. Duplicate() has no CLI path, so
// it is asserted here instead.
func TestInjectedCompilerEnvironmentIsShared(t *testing.T) {
	require := test_util.NewRequire(t)
	injected := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)

	absPath, err := filepath.Abs(filepath.Join("testdata", "workspace-simple"))
	require.NoError(err)

	result, err := loadProject(absPath, projects.ProjectLoadConfig{CompilerEnvironment: injected})
	require.NoError(err)

	workspace := result.Project().(*projects.WorkspaceProject)
	assertUsesEnvironment(t, "workspace", workspace, injected)
	for _, member := range workspace.Projects() {
		assertUsesEnvironment(t, "member "+member.SourceRoot(), member, injected)
	}
	assertUsesEnvironment(t, "duplicated workspace", workspace.Duplicate(), injected)
}

func assertUsesEnvironment(t *testing.T, what string, project projects.Project, injected *compilercontext.CompilerEnvironment) {
	t.Helper()
	if got := project.Environment().CompilerEnvironment(); got != injected {
		t.Fatalf("%s compiler environment = %p, want the injected %p", what, got, injected)
	}
}
