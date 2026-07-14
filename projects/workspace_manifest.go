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
	"slices"

	"ballerina-lang-go/tools/diagnostics"
)

// WorkspaceManifest represents the parsed [workspace] section from Ballerina.toml.
type WorkspaceManifest struct {
	packages    []string
	diagnostics DiagnosticResult
}

// Packages returns the relative paths to packages in this workspace.
func (m WorkspaceManifest) Packages() []string {
	return slices.Clone(m.packages)
}

// Diagnostics returns any diagnostics encountered while parsing the workspace manifest.
func (m WorkspaceManifest) Diagnostics() DiagnosticResult {
	return m.diagnostics
}

// newWorkspaceManifest creates a new WorkspaceManifest.
func newWorkspaceManifest(packages []string, diags []diagnostics.Diagnostic) WorkspaceManifest {
	return WorkspaceManifest{
		packages:    slices.Clone(packages),
		diagnostics: NewDiagnosticResult(diags),
	}
}
