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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !native_interp

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDriverSource(t *testing.T) {
	dir, err := ExtractDriverSource(t.TempDir(), "test-v0.0.1")
	if err != nil {
		t.Fatalf("ExtractDriverSource: %v", err)
	}

	for _, name := range []string{
		filepath.Join("cli", "go.mod"),
		filepath.Join("cli", "cmd"),
		filepath.Join("cli", "internal", "balrt"),
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("driver source must contain %s: %v", name, err)
		}
	}
	for _, name := range []string{"ast", "runtime", "projects", "semtypes"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("driver source must not contain dependency directory %s", name)
		}
	}
}

func TestExtractDriverSourceReusesCompleteReleaseCache(t *testing.T) {
	cacheRoot := t.TempDir()
	dir, err := ExtractDriverSource(cacheRoot, "test-v0.0.1")
	if err != nil {
		t.Fatalf("first ExtractDriverSource: %v", err)
	}
	marker := filepath.Join(dir, "marker")
	if err := os.WriteFile(marker, []byte("preserved"), 0o600); err != nil {
		t.Fatalf("writing marker: %v", err)
	}

	dir2, err := ExtractDriverSource(cacheRoot, "test-v0.0.1")
	if err != nil {
		t.Fatalf("second ExtractDriverSource: %v", err)
	}
	if dir2 != dir {
		t.Fatalf("second extraction returned %q, want %q", dir2, dir)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("complete release cache was unexpectedly replaced: %v", err)
	}
}

func TestExtractDriverSourceRepairsIncompleteReleaseCache(t *testing.T) {
	cacheRoot := t.TempDir()
	dir, err := ExtractDriverSource(cacheRoot, "test-v0.0.1")
	if err != nil {
		t.Fatalf("first ExtractDriverSource: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "cli", "go.mod")); err != nil {
		t.Fatalf("removing cli/go.mod: %v", err)
	}

	if _, err := ExtractDriverSource(cacheRoot, "test-v0.0.1"); err != nil {
		t.Fatalf("repairing extraction: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cli", "go.mod")); err != nil {
		t.Errorf("cli/go.mod was not restored: %v", err)
	}
}
