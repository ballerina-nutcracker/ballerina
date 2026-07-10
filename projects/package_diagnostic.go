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
	"ballerina/tools/diagnostics"
)

// Compile-time interface check.
var _ diagnostics.Diagnostic = (*packageDiagnostic)(nil)

// packageDiagnostic is a decorator for diagnostics exposed via the Project API.
// All diagnostics in a package are enriched with module and project info before
// being exposed via project API classes.
// Java source: io.ballerina.projects.internal.PackageDiagnostic
type packageDiagnostic struct {
	diagnostic       diagnostics.Diagnostic
	moduleDescriptor ModuleDescriptor
	project          Project
	isWorkspaceDep   bool
}

// newPackageDiagnostic creates a PackageDiagnostic wrapping the given diagnostic
// with module descriptor and project information.
// Java source: PackageDiagnostic(Diagnostic, ModuleDescriptor, Project, boolean)
func newPackageDiagnostic(
	diagnostic diagnostics.Diagnostic,
	moduleDescriptor ModuleDescriptor,
	project Project,
	isWorkspaceDep bool,
) *packageDiagnostic {
	return &packageDiagnostic{
		diagnostic:       diagnostic,
		moduleDescriptor: moduleDescriptor,
		project:          project,
		isWorkspaceDep:   isWorkspaceDep,
	}
}

// Location delegates to the inner diagnostic's location.
// Java source: PackageDiagnostic.location()
func (pd *packageDiagnostic) Location() diagnostics.Location {
	return pd.diagnostic.Location()
}

// DiagnosticInfo delegates to the inner diagnostic's info.
// Java source: PackageDiagnostic.diagnosticInfo()
func (pd *packageDiagnostic) DiagnosticInfo() diagnostics.DiagnosticInfo {
	return pd.diagnostic.DiagnosticInfo()
}

// Message delegates to the inner diagnostic's message.
// Java source: PackageDiagnostic.message()
func (pd *packageDiagnostic) Message() string {
	return pd.diagnostic.Message()
}

// Properties delegates to the inner diagnostic's properties.
// Java source: PackageDiagnostic.properties()
func (pd *packageDiagnostic) Properties() []diagnostics.DiagnosticProperty[any] {
	return pd.diagnostic.Properties()
}

// String returns the string representation of the diagnostic.
// Java source: PackageDiagnostic.toString()
func (pd *packageDiagnostic) String() string {
	return pd.diagnostic.String()
}
