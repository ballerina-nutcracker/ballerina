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

// moduleLoadRequestKey is used as a map key for caching module load responses.
type moduleLoadRequestKey struct {
	orgName    string
	moduleName string
}

func newModuleLoadRequestKey(request *moduleLoadRequest) moduleLoadRequestKey {
	orgName := ""
	if request.orgName != nil {
		orgName = request.orgName.Value()
	}
	return moduleLoadRequestKey{
		orgName:    orgName,
		moduleName: request.moduleName,
	}
}

// resolutionStatus indicates whether a module/package was resolved.
type resolutionStatus int

const (
	// resolutionStatusResolved indicates the module/package was found.
	resolutionStatusResolved resolutionStatus = iota
	// resolutionStatusUnresolved indicates the module/package was not found.
	resolutionStatusUnresolved
)

// importModuleResponse represents the result of resolving a moduleLoadRequest.
type importModuleResponse struct {
	packageDescriptor *PackageDescriptor // Package containing the module (nil for same package)
	moduleDesc        ModuleDescriptor   // Module descriptor
	resolutionStatus  resolutionStatus
}

// moduleResolver resolves module dependencies using PackageResolver.
type moduleResolver struct {
	rootPkgDesc       PackageDescriptor
	blendedManifest   *blendedManifest
	responseMap       map[moduleLoadRequestKey]*importModuleResponse
	packageResolver   PackageResolver
	resolutionOptions ResolutionOptions
}

func newModuleResolver(rootPkgDesc PackageDescriptor, blendedManifest *blendedManifest, env *Environment) *moduleResolver {
	return &moduleResolver{
		rootPkgDesc:       rootPkgDesc,
		blendedManifest:   blendedManifest,
		responseMap:       make(map[moduleLoadRequestKey]*importModuleResponse),
		packageResolver:   env.PackageResolver(),
		resolutionOptions: env.ResolutionOptions(),
	}
}

func (r *moduleResolver) resolveModuleLoadRequests(ctx context.Context, requests []*moduleLoadRequest) []*importModuleResponse {
	responses := make([]*importModuleResponse, 0, len(requests))

	for _, request := range requests {
		key := newModuleLoadRequestKey(request)

		// Check cache first
		if cached, ok := r.responseMap[key]; ok {
			responses = append(responses, cached)
			continue
		}

		// Try to resolve the module
		response := r.resolveRequest(ctx, request)
		r.responseMap[key] = response
		responses = append(responses, response)
	}

	return responses
}

func (r *moduleResolver) resolveRequest(ctx context.Context, request *moduleLoadRequest) *importModuleResponse {
	// Determine org - use request org or default to root package org
	org := r.rootPkgDesc.Org().Value()
	if request.orgName != nil {
		org = request.orgName.Value()
	}

	// Extract package name from module name (e.g., "http.auth" -> "http")
	pkgName := extractPackageName(request.moduleName)

	// Route through the user-specified repository when the root manifest names
	// one for this (org, name). Validation in blendedManifest guarantees the
	// package exists in that registry; if it doesn't carry the requested module,
	// fall through silently to the default chain.
	if blended, ok := r.blendedManifest.dependency(org, pkgName); ok && blended.Repository() != "" {
		desc := NewPackageDescriptor(blended.Org(), blended.Name(), blended.Version())
		customReq := newResolutionRequestWithRepository(desc, blended.Repository())
		responses := r.packageResolver.ResolvePackages(ctx, []ResolutionRequest{customReq}, r.resolutionOptions)
		if len(responses) > 0 && responses[0].IsResolved() {
			pkg := responses[0].Package()
			if pkg != nil {
				for _, mod := range pkg.Modules() {
					if mod.ModuleName().String() == request.moduleName {
						pkgDesc := pkg.Manifest().PackageDescriptor()
						var pkgDescPtr *PackageDescriptor
						if !pkgDesc.Equals(r.rootPkgDesc) {
							pkgDescPtr = &pkgDesc
						}
						return &importModuleResponse{
							packageDescriptor: pkgDescPtr,
							moduleDesc:        mod.Descriptor(),
							resolutionStatus:  resolutionStatusResolved,
						}
					}
				}
			}
		}
	}

	// Default chain: packages are returned oldest-first, so iterate in reverse.
	packages := r.packageResolver.ResolveByName(ctx, org, pkgName, r.resolutionOptions)
	for i := len(packages) - 1; i >= 0; i-- {
		pkg := packages[i]
		// Check if module exists in this package
		for _, mod := range pkg.Modules() {
			if mod.ModuleName().String() == request.moduleName {
				pkgDesc := pkg.Manifest().PackageDescriptor()
				// Only set packageDescriptor for external packages (nil for same-package)
				var pkgDescPtr *PackageDescriptor
				if !pkgDesc.Equals(r.rootPkgDesc) {
					pkgDescPtr = &pkgDesc
				}
				return &importModuleResponse{
					packageDescriptor: pkgDescPtr,
					moduleDesc:        mod.Descriptor(),
					resolutionStatus:  resolutionStatusResolved,
				}
			}
		}
	}

	// Module not found
	return &importModuleResponse{
		resolutionStatus: resolutionStatusUnresolved,
	}
}

// packageNameCandidates returns the package name(s) to try when resolving a module.
// The full module name is the first candidate (handles packages whose name contains a dot,
// e.g. "math.vector"). If the name contains a dot, the prefix before the first dot is also
// appended as a fallback (handles sub-modules, e.g. "http.auth" lives in package "http").
func packageNameCandidates(moduleName string) []string {
	for i, c := range moduleName {
		if c == '.' {
			return []string{moduleName, moduleName[:i]}
		}
	}
	return []string{moduleName}
}
