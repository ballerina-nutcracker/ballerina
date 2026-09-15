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

// TestManifest_IconValidation exercises manifestBuilder's icon validation
// (extension, existence, PNG-magic-header content check), mirroring Java's
// ManifestBuilder#validateIconPathForPng. Each case is reported as an error
// diagnostic on the manifest rather than a hard load failure — matching
// Java's diagnostic-based approach, which surfaces to bal build/run/pack
// once loaded (see cli/cmd/pack.go's Diagnostics().HasErrors() check).
func TestManifest_IconValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fixture     string
		wantMessage string
	}{
		{
			name:        "wrong extension",
			fixture:     "wrong-extension-project",
			wantMessage: "invalid 'icon' under [package]: 'icon' can only have 'png' images",
		},
		{
			name:        "missing file",
			fixture:     "missing-files-project",
			wantMessage: "could not locate icon path 'missing.png'",
		},
		{
			name:        "invalid content",
			fixture:     "icon-invalid-content-project",
			wantMessage: "invalid 'icon' under [package]: 'icon' can only have 'png' images",
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

// TestManifest_IconValidPngPasses confirms a real, valid PNG (as used by
// testdata/manifest-fields-project) produces no icon-related diagnostics.
func TestManifest_IconValidPngPasses(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "manifest-fields-project"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	for _, d := range result.Project().CurrentPackage().Manifest().Diagnostics() {
		if strings.Contains(d.Message(), "icon") {
			t.Errorf("unexpected icon diagnostic for a valid PNG: %s", d.Message())
		}
	}
}
