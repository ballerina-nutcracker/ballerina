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

// ProjectKind represents the type of project.
type ProjectKind int

const (
	// ProjectKindBuild represents a standard build project with Ballerina.toml.
	ProjectKindBuild ProjectKind = iota
	// ProjectKindSingleFile represents a single .bal file project.
	ProjectKindSingleFile
	// ProjectKindBala represents a compiled BALA package project.
	ProjectKindBala
	// ProjectKindWorkspace represents a multi-package workspace project.
	ProjectKindWorkspace
)

// String returns the string representation of ProjectKind.
func (k ProjectKind) String() string {
	switch k {
	case ProjectKindBuild:
		return "BUILD_PROJECT"
	case ProjectKindSingleFile:
		return "SINGLE_FILE_PROJECT"
	case ProjectKindBala:
		return "BALA_PROJECT"
	case ProjectKindWorkspace:
		return "WORKSPACE_PROJECT"
	default:
		return "UNKNOWN"
	}
}
