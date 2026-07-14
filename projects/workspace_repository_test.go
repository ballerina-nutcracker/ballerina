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

// Internal tests for workspaceRepository. The type is unexported and the
// resolver short-circuits on cache hits — workspace members are pre-cached
// at load time, so the repo's GetPackage / GetPackageVersions are not reached
// through ResolvePackages / ResolveByName in normal flows. These tests
// invoke the repository directly to lock in the contract of its methods.

package projects

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// loadWorkspaceForRepoTest loads testdata/workspace-simple and returns the
// resulting *WorkspaceProject, ready for use by repository unit tests.
func loadWorkspaceForRepoTest(t *testing.T) *WorkspaceProject {
	t.Helper()

	absPath, err := filepath.Abs(filepath.Join("testdata", "workspace-simple"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	fsys := os.DirFS(absPath)

	result, err := Load(fsys, ".")
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	ws, ok := result.Project().(*WorkspaceProject)
	if !ok {
		t.Fatalf("expected *WorkspaceProject, got %T", result.Project())
	}
	return ws
}

// repoBoundTo builds a fresh workspaceRepository bound to ws so the tests
// invoke it directly, bypassing the resolver's cache short-circuit.
func repoBoundTo(ws *WorkspaceProject) *workspaceRepository {
	r := newWorkspaceRepository()
	r.setWorkspace(ws)
	return r
}

func TestWorkspaceRepository_GetPackage(t *testing.T) {
	ws := loadWorkspaceForRepoTest(t)
	repo := repoBoundTo(ws)
	ctx := context.Background()
	opts := ResolutionOptions{}

	t.Run("matches org+name+version", func(t *testing.T) {
		pkg, err := repo.GetPackage(ctx, "testorg", "pkgb", "1.0.0", opts)
		if err != nil {
			t.Fatalf("GetPackage: %v", err)
		}
		if pkg == nil {
			t.Fatal("expected pkgb 1.0.0 to be found")
		}
		if got := pkg.Descriptor().Name().String(); got != "pkgb" {
			t.Errorf("name = %q, want pkgb", got)
		}
	})

	t.Run("empty version matches by org+name", func(t *testing.T) {
		pkg, err := repo.GetPackage(ctx, "testorg", "pkga", "", opts)
		if err != nil {
			t.Fatalf("GetPackage: %v", err)
		}
		if pkg == nil {
			t.Fatal("expected pkga (any version) to be found")
		}
	})

	t.Run("version mismatch returns nil", func(t *testing.T) {
		pkg, err := repo.GetPackage(ctx, "testorg", "pkgb", "9.9.9", opts)
		if err != nil {
			t.Fatalf("GetPackage: %v", err)
		}
		if pkg != nil {
			t.Errorf("expected nil for non-matching version, got %v", pkg.Descriptor())
		}
	})

	t.Run("unknown org returns nil", func(t *testing.T) {
		pkg, err := repo.GetPackage(ctx, "nope", "pkgb", "1.0.0", opts)
		if err != nil {
			t.Fatalf("GetPackage: %v", err)
		}
		if pkg != nil {
			t.Errorf("expected nil for unknown org, got %v", pkg.Descriptor())
		}
	})

	t.Run("unknown name returns nil", func(t *testing.T) {
		pkg, err := repo.GetPackage(ctx, "testorg", "missing", "1.0.0", opts)
		if err != nil {
			t.Fatalf("GetPackage: %v", err)
		}
		if pkg != nil {
			t.Errorf("expected nil for unknown name, got %v", pkg.Descriptor())
		}
	})

	t.Run("cancelled context returns ctx.Err", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		pkg, err := repo.GetPackage(cancelled, "testorg", "pkgb", "1.0.0", opts)
		if err == nil {
			t.Fatal("expected error from cancelled context")
		}
		if pkg != nil {
			t.Errorf("expected nil pkg on cancellation, got %v", pkg)
		}
	})
}

func TestWorkspaceRepository_GetPackageVersions(t *testing.T) {
	ws := loadWorkspaceForRepoTest(t)
	repo := repoBoundTo(ws)
	ctx := context.Background()
	opts := ResolutionOptions{}

	t.Run("known package returns its version", func(t *testing.T) {
		versions, err := repo.GetPackageVersions(ctx, "testorg", "pkga", opts)
		if err != nil {
			t.Fatalf("GetPackageVersions: %v", err)
		}
		if len(versions) != 1 {
			t.Fatalf("expected exactly 1 version, got %d", len(versions))
		}
		if got := versions[0].String(); got != "1.0.0" {
			t.Errorf("version = %q, want 1.0.0", got)
		}
	})

	t.Run("unknown package returns empty", func(t *testing.T) {
		versions, err := repo.GetPackageVersions(ctx, "testorg", "missing", opts)
		if err != nil {
			t.Fatalf("GetPackageVersions: %v", err)
		}
		if len(versions) != 0 {
			t.Errorf("expected empty slice for missing package, got %v", versions)
		}
	})

	t.Run("cancelled context returns ctx.Err", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := repo.GetPackageVersions(cancelled, "testorg", "pkga", opts)
		if err == nil {
			t.Fatal("expected error from cancelled context")
		}
	})
}
