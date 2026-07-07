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

// ResolutionOptions controls package resolution behavior.
type ResolutionOptions struct {
	offline      bool // Skip remote repositories
	sticky       bool // Use Dependencies.toml versions strictly
	dumpGraph    bool // Debug: dump dependency graph
	disableCache bool // Force fresh resolution, skip cache
}

// NewResolutionOptions creates default resolution options.
func NewResolutionOptions() ResolutionOptions {
	return ResolutionOptions{}
}

// Offline returns whether to skip remote repositories.
func (o ResolutionOptions) Offline() bool { return o.offline }

// Sticky returns whether to use Dependencies.toml versions strictly.
func (o ResolutionOptions) Sticky() bool { return o.sticky }

// DumpGraph returns whether to dump the dependency graph for debugging.
func (o ResolutionOptions) DumpGraph() bool { return o.dumpGraph }

// DisableCache returns whether to skip cache and force fresh resolution.
func (o ResolutionOptions) DisableCache() bool { return o.disableCache }

// WithOffline returns a copy with offline set.
func (o ResolutionOptions) WithOffline(offline bool) ResolutionOptions {
	o.offline = offline
	return o
}

// WithSticky returns a copy with sticky set.
func (o ResolutionOptions) WithSticky(sticky bool) ResolutionOptions {
	o.sticky = sticky
	return o
}

// WithDisableCache returns a copy with disableCache set.
func (o ResolutionOptions) WithDisableCache(disableCache bool) ResolutionOptions {
	o.disableCache = disableCache
	return o
}

// ResolutionRequest represents a request to resolve a package.
type ResolutionRequest struct {
	descriptor PackageDescriptor
	scope      PackageDependencyScope
	repo       string // non-empty routes resolution to the user-specified custom repo (e.g., "local"); empty uses the default chain
}

// NewResolutionRequest creates a resolution request for a package descriptor.
func NewResolutionRequest(descriptor PackageDescriptor) ResolutionRequest {
	return ResolutionRequest{
		descriptor: descriptor,
		scope:      PackageDependencyScopeDefault,
	}
}

// NewResolutionRequestWithScope creates a resolution request with a specific scope.
func NewResolutionRequestWithScope(descriptor PackageDescriptor, scope PackageDependencyScope) ResolutionRequest {
	return ResolutionRequest{
		descriptor: descriptor,
		scope:      scope,
	}
}

// newResolutionRequestWithRepository builds a request routed to a user-specified
// custom repository. An empty repo string falls back to NewResolutionRequest.
func newResolutionRequestWithRepository(descriptor PackageDescriptor, repo string) ResolutionRequest {
	return ResolutionRequest{
		descriptor: descriptor,
		scope:      PackageDependencyScopeDefault,
		repo:       repo,
	}
}

// Descriptor returns the package descriptor being requested.
func (r ResolutionRequest) Descriptor() PackageDescriptor { return r.descriptor }

// Scope returns the dependency scope.
func (r ResolutionRequest) Scope() PackageDependencyScope { return r.scope }

// repository returns the user-specified custom repository, or empty if unset.
func (r ResolutionRequest) repository() string { return r.repo }

// Org returns the package organization.
func (r ResolutionRequest) Org() PackageOrg { return r.descriptor.Org() }

// Name returns the package name.
func (r ResolutionRequest) Name() PackageName { return r.descriptor.Name() }

// Version returns the package version.
func (r ResolutionRequest) Version() PackageVersion { return r.descriptor.Version() }

// PackageDependencyScope indicates the scope of a dependency.
type PackageDependencyScope int

const (
	// PackageDependencyScopeDefault is the default scope for runtime dependencies.
	PackageDependencyScopeDefault PackageDependencyScope = iota
	// PackageDependencyScopeTestOnly is for test-only dependencies.
	PackageDependencyScopeTestOnly
)

// ResolutionResponse represents the result of resolving a package.
type ResolutionResponse struct {
	resolvedPackage *Package
	request         ResolutionRequest
	status          resolutionStatus
}

// NewResolvedResponse creates a successful resolution response.
func NewResolvedResponse(pkg *Package, request ResolutionRequest) ResolutionResponse {
	return ResolutionResponse{
		resolvedPackage: pkg,
		request:         request,
		status:          resolutionStatusResolved,
	}
}

// NewUnresolvedResponse creates a failed resolution response.
func NewUnresolvedResponse(request ResolutionRequest) ResolutionResponse {
	return ResolutionResponse{
		request: request,
		status:  resolutionStatusUnresolved,
	}
}

// Package returns the resolved package, or nil if unresolved.
func (r ResolutionResponse) Package() *Package { return r.resolvedPackage }

// Request returns the original resolution request.
func (r ResolutionResponse) Request() ResolutionRequest { return r.request }

// IsResolved returns true if the package was successfully resolved.
func (r ResolutionResponse) IsResolved() bool { return r.status == resolutionStatusResolved }
