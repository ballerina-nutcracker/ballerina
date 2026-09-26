// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package test_util

import (
	"os"
	"path/filepath"
	"testing"
)

// ReadExpectedFile reads the expected output file and returns its contents
func ReadExpectedFile(t *testing.T, expectedPath string) string {
	t.Helper()
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read expected file %s: %v", expectedPath, err)
	}
	return string(data)
}

// UpdateIfNeeded compares the actual content with the expected file content.
// Returns true if the content differs and an update was made.
// Returns false if the content matches (no update needed).
// Optionally accepts a normalization function applied to both the existing and
// the actual content before comparison. The actual content is written as is.
func UpdateIfNeeded(t *testing.T, expectedPath, actual string, normalize ...func(string) string) bool {
	t.Helper()
	existingContent, err := os.ReadFile(expectedPath)
	if err == nil && normalizeForComparison(string(existingContent), normalize) == normalizeForComparison(actual, normalize) {
		return false
	}
	dir := filepath.Dir(expectedPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(expectedPath, []byte(actual), 0o644); err != nil {
		t.Fatalf("Failed to write expected file %s: %v", expectedPath, err)
	}
	t.Logf("Updated expected file: %s", expectedPath)
	return true
}

func normalizeForComparison(content string, normalize []func(string) string) string {
	if len(normalize) > 0 && normalize[0] != nil {
		return normalize[0](content)
	}
	return content
}
