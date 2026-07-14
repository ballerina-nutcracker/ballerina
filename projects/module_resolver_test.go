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
	"context"
	"os"
	"testing"
)

// newTestModuleResolver builds a moduleResolver whose "local" custom repo
// points to testdata/repo/bala, and whose blended manifest routes
// (mockorg, localPkg) to "local" when localPkg is non-empty.
func newTestModuleResolver(t *testing.T, localPkg string) *moduleResolver {
	t.Helper()
	v100, err := NewPackageVersionFromString("1.0.0")
	if err != nil {
		t.Fatalf("NewPackageVersionFromString: %v", err)
	}

	rootDesc := NewPackageDescriptor(NewPackageOrg("myorg"), NewPackageName("mypkg"), v100)

	var deps []Dependency
	if localPkg != "" {
		deps = []Dependency{
			NewDependencyWithRepository(NewPackageOrg("mockorg"), NewPackageName(localPkg), v100, "local"),
		}
	}
	manifest := NewPackageManifestFromParams(PackageManifestParams{
		PackageDesc:  rootDesc,
		Dependencies: deps,
	})

	localRepo := NewFileSystemRepository(os.DirFS("testdata/repo/bala"), ".")
	resolver := newPackageResolverWithCustom(newPackageCache(), nil, map[string]Repository{"local": localRepo})
	bm := newBlendedManifest(withPackageManifest(manifest))

	return &moduleResolver{
		rootPkgDesc:       rootDesc,
		blendedManifest:   bm,
		responseMap:       make(map[moduleLoadRequestKey]*importModuleResponse),
		packageResolver:   resolver,
		resolutionOptions: ResolutionOptions{},
	}
}

// TestResolveFromUserSpecifiedRepo_Hit verifies the happy path: a module that
// exists in the user-specified "local" repo is returned as resolved.
func TestResolveFromUserSpecifiedRepo_Hit(t *testing.T) {
	t.Parallel()
	mr := newTestModuleResolver(t, "mockpkg")

	resp := mr.resolveFromUserSpecifiedRepo(context.Background(), "mockorg", "mockpkg", "mockpkg")
	if resp == nil {
		t.Fatal("expected non-nil response for module found in local repo")
		return
	}
	if resp.resolutionStatus != resolutionStatusResolved {
		t.Errorf("resolutionStatus = %v, want resolved", resp.resolutionStatus)
	}
	if resp.moduleDesc.Name().String() != "mockpkg" {
		t.Errorf("moduleDesc.Name() = %q, want %q", resp.moduleDesc.Name().String(), "mockpkg")
	}
	// Root package is different from the resolved package, so packageDescriptor must be non-nil.
	if resp.packageDescriptor == nil {
		t.Error("expected packageDescriptor to be non-nil for external dep")
	}
}

// TestResolveFromUserSpecifiedRepo_PackageNotResolved verifies that a nil is
// returned when the user-specified repo cannot resolve the package at all.
func TestResolveFromUserSpecifiedRepo_PackageNotResolved(t *testing.T) {
	t.Parallel()
	mr := newTestModuleResolver(t, "nosuchpkg")

	resp := mr.resolveFromUserSpecifiedRepo(context.Background(), "mockorg", "nosuchpkg", "nosuchpkg")
	if resp != nil {
		t.Errorf("expected nil when package not found in local repo, got %+v", resp)
	}
}

// TestResolveFromUserSpecifiedRepo_ModuleNotInPackage verifies that nil is
// returned when the package is resolved but does not contain the requested module.
func TestResolveFromUserSpecifiedRepo_ModuleNotInPackage(t *testing.T) {
	t.Parallel()
	mr := newTestModuleResolver(t, "mockpkg")

	// mockpkg only exports module "mockpkg", not "mockpkg.submod".
	resp := mr.resolveFromUserSpecifiedRepo(context.Background(), "mockorg", "mockpkg", "mockpkg.submod")
	if resp != nil {
		t.Errorf("expected nil when requested module not in package, got %+v", resp)
	}
}

// TestResolveRequest_UserSpecifiedRepoTakesPriority verifies the integration:
// resolveRequest returns early with the user-specified-repo result without
// falling through to the default chain.
func TestResolveRequest_UserSpecifiedRepoTakesPriority(t *testing.T) {
	t.Parallel()
	mr := newTestModuleResolver(t, "mockpkg")

	org := NewPackageOrg("mockorg")
	req := newModuleLoadRequest(&org, "mockpkg")

	resp := mr.resolveRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if resp.resolutionStatus != resolutionStatusResolved {
		t.Errorf("resolutionStatus = %v, want resolved", resp.resolutionStatus)
	}
}
