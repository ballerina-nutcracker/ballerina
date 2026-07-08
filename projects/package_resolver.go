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

// PackageResolver resolves packages from available repositories.
type PackageResolver interface {
	// ResolvePackages loads the packages specified in the requests.
	ResolvePackages(ctx context.Context, requests []ResolutionRequest, options ResolutionOptions) []ResolutionResponse

	// ResolveByName resolves packages by org and name, without requiring a specific version.
	// Returns all matching packages from cache or repositories.
	ResolveByName(ctx context.Context, org, name string, options ResolutionOptions) []*Package

	// AddRepository adds a repository to the resolver.
	// Repositories are searched in the order they are added.
	AddRepository(repo Repository)

	// Repositories returns the list of repositories in this resolver.
	Repositories() []Repository
}

// defaultPackageResolver is the default implementation of PackageResolver.
type defaultPackageResolver struct {
	cache        *PackageCache
	repositories []Repository
	// customRepos holds repositories addressable by name (e.g., "local"),
	// consulted only when a ResolutionRequest carries a non-empty repo name.
	customRepos map[string]Repository
}

// AddRepository adds a repository to the resolver.
// Repositories are searched in the order they are added.
func (r *defaultPackageResolver) AddRepository(repo Repository) {
	r.repositories = append(r.repositories, repo)
}

// Repositories returns the list of repositories in this resolver.
func (r *defaultPackageResolver) Repositories() []Repository {
	return r.repositories
}

// NewPackageResolver creates a new PackageResolver with no custom repositories.
// Use newPackageResolverWithCustom to supply a custom-repo registry.
func NewPackageResolver(cache *PackageCache, repositories ...Repository) PackageResolver {
	return newPackageResolverWithCustom(cache, repositories, nil)
}

// newPackageResolverWithCustom creates a resolver with an ordered chain and a
// custom-repo registry. A nil custom map is coerced to an empty non-nil map.
func newPackageResolverWithCustom(cache *PackageCache, chain []Repository, custom map[string]Repository) PackageResolver {
	if custom == nil {
		custom = map[string]Repository{}
	}
	return &defaultPackageResolver{
		cache:        cache,
		repositories: chain,
		customRepos:  custom,
	}
}

// ResolvePackages resolves packages by checking cache first, then repositories.
func (r *defaultPackageResolver) ResolvePackages(
	ctx context.Context,
	requests []ResolutionRequest,
	options ResolutionOptions,
) []ResolutionResponse {
	responses := make([]ResolutionResponse, 0, len(requests))

	for _, request := range requests {
		response := r.resolvePackage(ctx, request, options)
		responses = append(responses, response)
	}

	return responses
}

func (r *defaultPackageResolver) resolvePackage(
	ctx context.Context,
	request ResolutionRequest,
	options ResolutionOptions,
) ResolutionResponse {
	desc := request.Descriptor()
	org := desc.Org().Value()
	name := desc.Name().Value()
	version := desc.Version().String()

	// A user-specified repository routes through customRepos only; misses are
	// terminal and do not fall through to the default chain.
	if repoName := request.repository(); repoName != "" {
		if !options.DisableCache() {
			pkg := r.cache.Get(org, name, version)
			if pkg != nil {
				return NewResolvedResponse(pkg, request)
			}
		}
		repo, ok := r.customRepos[repoName]
		if !ok {
			return NewUnresolvedResponse(request)
		}
		pkg, err := repo.GetPackage(ctx, org, name, version, options)
		if err != nil || pkg == nil {
			return NewUnresolvedResponse(request)
		}
		if !options.DisableCache() {
			r.cache.Cache(pkg)
		}
		return NewResolvedResponse(pkg, request)
	}

	// 1. Check cache first (unless disabled)
	if !options.DisableCache() {
		pkg := r.cache.Get(org, name, version)
		if pkg != nil {
			return NewResolvedResponse(pkg, request)
		}
	}

	// 2. Try repositories in order
	for _, repo := range r.repositories {
		select {
		case <-ctx.Done():
			return NewUnresolvedResponse(request)
		default:
		}

		pkg, err := repo.GetPackage(ctx, org, name, version, options)
		if err != nil {
			continue // Try next repository
		}
		if pkg == nil {
			continue
		}

		// 3. Cache the loaded package
		if !options.DisableCache() {
			r.cache.Cache(pkg)
		}

		return NewResolvedResponse(pkg, request)
	}

	// Not found in any repository
	return NewUnresolvedResponse(request)
}

// ResolveByName resolves a package by org+name to its latest version.
// Returns a single-element slice on success, or an empty slice if no
// repository knows the package.
//
// On a cache miss, the resolver iterates the configured Repository chain
// and asks each one for its known versions via Repository.GetPackageVersions.
// The highest of those (selected by pickLatestVersion) is loaded with
// Repository.GetPackage and cached for subsequent lookups. The first
// repository that lists at least one version wins; older versions from
// that repository are not loaded — callers are expected to import a
// package that exists at the repository's latest version.
func (r *defaultPackageResolver) ResolveByName(
	ctx context.Context,
	org, name string,
	options ResolutionOptions,
) []*Package {
	// 1. Check cache first; pick the highest cached version.
	if !options.DisableCache() {
		if pkg, ok := pickLatest(r.cache.GetPackages(org, name), packageVersion); ok {
			return []*Package{pkg}
		}
	}

	// 2. Try repositories in order; first repo that owns the package wins.
	for _, repo := range r.repositories {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		versions, err := repo.GetPackageVersions(ctx, org, name, options)
		if err != nil || len(versions) == 0 {
			continue
		}
		latest, ok := pickLatest(versions, identityVersion)
		if !ok {
			// Repository listed no versions for this org+name.
			continue
		}

		// Once a repo lists at least one version, it owns the package by
		// precedence. If the subsequent GetPackage fails, treat the lookup
		// as terminally unresolved — falling through to a lower-priority
		// repo would silently substitute a different source's package for a
		// broken higher-priority one.
		pkg, err := repo.GetPackage(ctx, org, name, latest.String(), options)
		if err != nil || pkg == nil {
			return nil
		}

		if !options.DisableCache() {
			r.cache.Cache(pkg)
		}
		return []*Package{pkg}
	}

	return nil
}

// pickLatest returns the element of items whose extracted PackageVersion is
// the highest, plus ok=true. For an empty slice it returns the zero value of
// T and ok=false, letting the caller handle absence explicitly.
//
// Used both for picking the latest already-loaded package out of the cache
// (T = *Package) and for picking the latest version a repository advertised
// before loading (T = PackageVersion).
func pickLatest[T any](items []T, version func(T) PackageVersion) (T, bool) {
	var zero T
	if len(items) == 0 {
		return zero, false
	}
	best := items[0]
	for _, item := range items[1:] {
		if version(item).Compare(version(best)) > 0 {
			best = item
		}
	}
	return best, true
}

// packageVersion is the version extractor used by pickLatest when picking
// among already-loaded *Package instances.
func packageVersion(p *Package) PackageVersion {
	return p.Manifest().PackageDescriptor().Version()
}

// identityVersion is the trivial extractor for slices of PackageVersion.
func identityVersion(v PackageVersion) PackageVersion { return v }
