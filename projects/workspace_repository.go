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
)

// workspaceRepository resolves packages from within a workspace.
// It allows packages in a workspace to import each other by org+name lookup.
// This repository has the highest priority in resolution order.
type workspaceRepository struct {
	workspace *WorkspaceProject
}

// newWorkspaceRepository creates a repository for workspace package resolution.
// The workspace reference must be set via setWorkspace before use.
func newWorkspaceRepository() *workspaceRepository {
	return &workspaceRepository{}
}

// setWorkspace sets the workspace reference for this repository.
// This must be called after the workspace is created.
func (r *workspaceRepository) setWorkspace(workspace *WorkspaceProject) {
	r.workspace = workspace
}

// GetPackage returns a package from the workspace matching the given org, name, and version.
// If version is empty, it matches by org+name only.
// Returns (nil, nil) if not found.
//
// ResolutionOptions are ignored — workspace lookups never consult a remote
// source, so flags like Offline have no effect here.
//
// The repository is constructed before the workspace exists (the loader and
// WorkspaceProject.Duplicate install the repo first, then call setWorkspace).
// During that window any consult must be a no-op rather than a nil-deref panic.
func (r *workspaceRepository) GetPackage(ctx context.Context, org, name, version string, _ ResolutionOptions) (*Package, error) {
	if r.workspace == nil {
		return nil, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	for _, project := range r.workspace.Projects() {
		pkg := project.CurrentPackage()
		if pkg == nil {
			continue
		}

		desc := pkg.Descriptor()
		if desc.Org().String() != org || desc.Name().String() != name {
			continue
		}

		// If version is specified, it must match exactly
		if version != "" && desc.Version().String() != version {
			continue
		}

		return pkg, nil
	}

	return nil, nil
}

// GetPackageVersions returns available versions for a package in the workspace.
// Since workspace packages have a single version, this returns at most one version.
//
// Returns nil if the workspace reference has not been set yet — see the note
// on GetPackage about construction order.
func (r *workspaceRepository) GetPackageVersions(ctx context.Context, org, name string, _ ResolutionOptions) ([]PackageVersion, error) {
	if r.workspace == nil {
		return nil, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	for _, project := range r.workspace.Projects() {
		pkg := project.CurrentPackage()
		if pkg == nil {
			continue
		}

		desc := pkg.Descriptor()
		if desc.Org().String() == org && desc.Name().String() == name {
			return []PackageVersion{desc.Version()}, nil
		}
	}

	return nil, nil
}

// Compile-time verification that workspaceRepository implements Repository.
var _ Repository = (*workspaceRepository)(nil)
