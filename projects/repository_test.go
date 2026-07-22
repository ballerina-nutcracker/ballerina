/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package projects_test

import (
	"context"
	"os"
	"slices"
	"testing"

	"ballerina-lang-go/projects"
)

func newTestRepository(path string) *projects.FileSystemRepository {
	return projects.NewFileSystemRepository(os.DirFS(path), ".")
}

func TestRepository_GetPackageVersions(t *testing.T) {
	repo := newTestRepository("testdata/repo/bala")

	tests := []struct {
		name     string
		org      string
		pkg      string
		expected []string
	}{
		{
			name:     "multiple versions sorted",
			org:      "ballerina",
			pkg:      "http",
			expected: []string{"2.10.0", "2.10.1"},
		},
		{
			name:     "single version",
			org:      "testorg",
			pkg:      "testpkg",
			expected: []string{"1.0.1"},
		},
		{
			name:     "non-existent package",
			org:      "ballerina",
			pkg:      "nonexistent",
			expected: nil,
		},
		{
			name:     "non-existent org",
			org:      "nonexistent",
			pkg:      "http",
			expected: nil,
		},
		{
			name:     "mockorg package",
			org:      "mockorg",
			pkg:      "mockpkg",
			expected: []string{"1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions, err := repo.GetPackageVersions(context.Background(), tt.org, tt.pkg, projects.ResolutionOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Convert to strings for comparison
			var got []string
			for _, v := range versions {
				got = append(got, v.String())
			}
			if !slices.Equal(got, tt.expected) {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRepository_Exists(t *testing.T) {
	repo := newTestRepository("testdata/repo/bala")

	tests := []struct {
		name     string
		org      string
		pkg      string
		version  string
		expected bool
	}{
		{"exists", "ballerina", "http", "2.10.0", true},
		{"exists latest", "ballerina", "http", "2.10.1", true},
		{"version not exists", "ballerina", "http", "1.0.0", false},
		{"package not exists", "ballerina", "nonexistent", "1.0.0", false},
		{"org not exists", "nonexistent", "http", "1.0.0", false},
		{"testorg exists", "testorg", "testpkg", "1.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.Exists(context.Background(), tt.org, tt.pkg, tt.version)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists != tt.expected {
				t.Errorf("got %v, want %v", exists, tt.expected)
			}
		})
	}
}

func TestRepository_ContextCancellation(t *testing.T) {
	repo := newTestRepository("testdata/repo/bala")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	t.Run("GetPackageVersions cancelled", func(t *testing.T) {
		_, err := repo.GetPackageVersions(ctx, "ballerina", "http", projects.ResolutionOptions{})
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("Exists cancelled", func(t *testing.T) {
		_, err := repo.Exists(ctx, "ballerina", "http", "2.10.0")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestRepository_ImplementsProjectsRepository(t *testing.T) {
	// Compile-time check - if this compiles, FileSystemRepository implements projects.Repository
	var _ projects.Repository = (*projects.FileSystemRepository)(nil)
}
