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

// RemoteRepository represents a remote package repository (e.g., Central).
// In Java this is RemotePackageRepository: a remote source paired with a
// FileSystemRepository acting as its on-disk cache. Lookups try the cache
// first; on a miss, the remote source pulls into the cache and the cache is
// re-read.
//
// This Go port currently includes only the cache half. The remote-fetch
// half will be wired in once the centralclient binding is introduced; the
// type already implements Repository so callers can adopt it now without
// further surface change.
type RemoteRepository struct {
	cache *FileSystemRepository
}

// NewRemoteRepository wires a FileSystemRepository to act as the on-disk
// cache for this repository. cache must be non-nil.
func NewRemoteRepository(cache *FileSystemRepository) *RemoteRepository {
	if cache == nil {
		panic("projects: NewRemoteRepository requires a non-nil FileSystemRepository cache")
	}
	return &RemoteRepository{cache: cache}
}

// GetPackage returns a package from the cache; on a cache miss it would
// normally pull from the remote source. Until that source is wired in, a
// miss returns (nil, nil) and the resolver moves on to the next repository.
//
// When ResolutionOptions.Offline is set, the remote step is skipped — only
// what is already in the cache is returned.
func (r *RemoteRepository) GetPackage(ctx context.Context, org, name, version string, options ResolutionOptions) (*Package, error) {
	pkg, err := r.cache.GetPackage(ctx, org, name, version, options)
	if err != nil || pkg != nil {
		return pkg, err
	}
	if options.Offline() {
		return nil, nil
	}
	// TODO: fetch from the remote source into the cache, then re-read.
	return nil, nil
}

// GetPackageVersions returns versions known to this repository. Today only
// the local cache is consulted; once a remote source is wired in, online
// mode will merge in remotely-advertised versions.
//
// When ResolutionOptions.Offline is set, the remote step is skipped — only
// versions already in the cache are returned.
func (r *RemoteRepository) GetPackageVersions(ctx context.Context, org, name string, options ResolutionOptions) ([]PackageVersion, error) {
	local, err := r.cache.GetPackageVersions(ctx, org, name, options)
	if err != nil {
		return nil, err
	}
	if options.Offline() {
		return local, nil
	}
	// TODO: union with remote-advertised versions once the remote source is wired in.
	return local, nil
}

// bind forwards the environment binding to the underlying cache so that
// FileSystemRepository can hand the env to loadBalaProjectInEnvironment.
// This keeps the late-binding contract intact when RemoteRepository is the
// top-level entry in the resolver chain.
func (r *RemoteRepository) bind(env *Environment) {
	if r.cache != nil {
		r.cache.bind(env)
	}
}

var _ Repository = (*RemoteRepository)(nil)
var _ bindableRepository = (*RemoteRepository)(nil)
