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

package projects

import (
	"cmp"
	"strings"

	"ballerina-lang-go/tools/diagnostics"
)

// WorkspaceResolution holds the result of workspace-level dependency resolution.
// It determines the compilation order of packages within a workspace based on
// their inter-dependencies.
type WorkspaceResolution struct {
	workspace        *WorkspaceProject
	dependencyGraph  *DependencyGraph[*BuildProject]
	diagnosticResult DiagnosticResult
}

// newWorkspaceResolution creates a WorkspaceResolution for the given workspace.
func newWorkspaceResolution(workspace *WorkspaceProject) *WorkspaceResolution {
	r := &WorkspaceResolution{
		workspace: workspace,
	}
	r.dependencyGraph = r.buildDependencyGraph()
	return r
}

// buildDependencyGraph builds the dependency graph between workspace packages.
// It leverages each package's resolved dependencies to find workspace-internal dependencies.
func (r *WorkspaceResolution) buildDependencyGraph() *DependencyGraph[*BuildProject] {
	builder := newDependencyGraphBuilder(func(a, b *BuildProject) int {
		return cmp.Compare(a.SourceRoot(), b.SourceRoot())
	})
	var diags []diagnostics.Diagnostic

	// Build a lookup map: org/name -> BuildProject
	projectMap := make(map[string]*BuildProject)
	for _, project := range r.workspace.Projects() {
		pkg := project.CurrentPackage()
		if pkg == nil {
			continue
		}
		desc := pkg.Descriptor()
		key := desc.Org().Value() + "/" + desc.Name().Value()
		projectMap[key] = project
		builder.addNode(project)
	}

	// For each project, check if its resolved dependencies are in the workspace
	for _, project := range r.workspace.Projects() {
		pkg := project.CurrentPackage()
		if pkg == nil {
			continue
		}

		// Get resolved dependencies from package resolution
		resolution := pkg.Resolution()
		if resolution == nil {
			continue
		}

		for key := range resolution.ResolvedDependencies() {
			if depProject, isWorkspacePkg := projectMap[key]; isWorkspacePkg {
				// This is a workspace-internal dependency
				if depProject != project {
					builder.addDependency(project, depProject)
				}
			}
		}
	}

	graph := builder.build()

	// Check for cycles
	cycles := graph.FindCycles()
	if len(cycles) > 0 {
		for _, cycle := range cycles {
			var cycleNames []string
			for _, proj := range cycle {
				if proj.CurrentPackage() != nil {
					desc := proj.CurrentPackage().Descriptor()
					cycleNames = append(cycleNames, desc.Org().Value()+"/"+desc.Name().Value())
				}
			}
			diags = append(diags, createSimpleDiagnostic(
				diagnostics.Error,
				"circular dependency detected in workspace: "+strings.Join(cycleNames, " -> "),
			))
		}
	}

	r.diagnosticResult = NewDiagnosticResult(diags)
	return graph
}

// DependencyGraph returns the workspace package dependency graph.
func (r *WorkspaceResolution) DependencyGraph() *DependencyGraph[*BuildProject] {
	return r.dependencyGraph
}

// DiagnosticResult returns any diagnostics from resolution (e.g., cycle errors).
func (r *WorkspaceResolution) DiagnosticResult() DiagnosticResult {
	return r.diagnosticResult
}
