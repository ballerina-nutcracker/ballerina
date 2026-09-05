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
	"strings"
	"testing"
)

// TestManifest_ReadmeValidation exercises manifestBuilder's readme
// validation (existence, ".md" extension) for an explicitly-declared
// readme, mirroring Java's ManifestBuilder#validateAndGetReadmePath.
func TestManifest_ReadmeValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fixture     string
		wantMessage string
	}{
		{
			name:        "wrong extension",
			fixture:     "wrong-extension-project",
			wantMessage: "invalid 'readme' under [package]: 'readme' can only have '.md' files",
		},
		{
			name:        "missing file",
			fixture:     "missing-files-project",
			wantMessage: "could not locate the readme file 'MISSING.md'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source, err := filepath.Abs(filepath.Join("testdata", tc.fixture))
			if err != nil {
				t.Fatalf("abs source: %v", err)
			}
			result, err := loadProject(source)
			if err != nil {
				t.Fatalf("load source project: %v", err)
			}

			diags := result.Project().CurrentPackage().Manifest().Diagnostics()
			var messages []string
			found := false
			for _, d := range diags {
				messages = append(messages, d.Message())
				if strings.Contains(d.Message(), tc.wantMessage) {
					found = true
				}
			}
			if !found {
				t.Errorf("manifest diagnostics = %v, want one containing %q", messages, tc.wantMessage)
			}
		})
	}
}

// TestManifest_ModuleNameValidation exercises manifestBuilder's
// [[package.modules]] name validation, mirroring Java's
// ManifestBuilder#validateAndGetModuleNodes: a name equal to the package's
// own name is rejected, and a name with no corresponding modules/
// directory is rejected too (whether or not it's otherwise
// package-name-prefixed).
func TestManifest_ModuleNameValidation(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "module-name-invalid-project"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	diags := result.Project().CurrentPackage().Manifest().Diagnostics()
	var messages []string
	for _, d := range diags {
		messages = append(messages, d.Message())
	}

	wantSubstrings := []string{
		"module 'moduleinvalidproject' is not allowed",
		"module 'moduleinvalidproject.missing' not found",
		"module 'totallywrongname' not found",
	}
	for _, want := range wantSubstrings {
		found := false
		for _, msg := range messages {
			if strings.Contains(msg, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("manifest diagnostics = %v, want one containing %q", messages, want)
		}
	}
}
