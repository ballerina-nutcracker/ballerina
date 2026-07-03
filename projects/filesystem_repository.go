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
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"ballerina-lang-go/lib/langlibs"
	"ballerina-lang-go/lib/stdlibs"
)

const (
	// centralCacheSubpath is the path under BALLERINA_HOME for cached packages.
	centralCacheSubpath = "repositories/central.ballerina.io/bala"
	// localRepoCacheSubpath is the path under BALLERINA_HOME for packages
	// the user routed via `repository = "local"` in Ballerina.toml. Layout:
	// <HOME>/repositories/local/bala/.
	localRepoCacheSubpath = "repositories/local/bala"
	// platformAny is the platform directory name for platform-independent packages.
	platformAny = "any"
	// platformGoPrefix marks platform directories that target a specific Go
	// toolchain version (e.g. "go1.26"). Used as a fallback when no "any"
	// directory is present.
	platformGoPrefix = "go"
)

// bindableRepository is an internal interface for repositories that support late binding.
// Late binding allows repositories to be created before the Environment exists,
// then bound to the Environment during project loading.
type bindableRepository interface {
	Repository
	bind(env *Environment)
}

// FileSystemRepository loads packages from a local bala directory structure using fs.FS.
// Directory structure: basePath/{org}/{name}/{version}/any/package.json
type FileSystemRepository struct {
	basePath string
	fsys     fs.FS
	env      *Environment
}

// NewFileSystemRepository creates a repository that uses fs.FS for file access.
// The repository must be bound to an Environment before use via ProjectEnvironmentBuilder.
func NewFileSystemRepository(fsys fs.FS, basePath string) *FileSystemRepository {
	return &FileSystemRepository{
		basePath: basePath,
		fsys:     fsys,
	}
}

// bind sets the environment for this repository. Called internally during project loading.
func (r *FileSystemRepository) bind(env *Environment) {
	r.env = env
}

// GetPackage loads a specific version. Returns (nil, nil) if not found.
// FileSystemRepository ignores ResolutionOptions — it only ever reads from
// disk, so flags like Offline have no effect here.
func (r *FileSystemRepository) GetPackage(ctx context.Context, org, name, version string, _ ResolutionOptions) (*Package, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	versionDir := path.Join(r.basePath, org, name, version)
	info, exists, err := statIfExists(r.fsys, versionDir)
	if err != nil || !exists || !info.IsDir() {
		return nil, err
	}

	platformDir, found, err := r.findPlatformDir(versionDir)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	project, err := loadBalaProjectInEnvironment(r.fsys, platformDir, r.env)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %s/%s:%s: %w", org, name, version, err)
	}

	return project.CurrentPackage(), nil
}

// GetPackageVersions returns all available versions for a package.
func (r *FileSystemRepository) GetPackageVersions(ctx context.Context, org, name string, _ ResolutionOptions) ([]PackageVersion, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	packageDir := path.Join(r.basePath, org, name)
	entries, err := fs.ReadDir(r.fsys, packageDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var versions []PackageVersion
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version, err := NewPackageVersionFromString(entry.Name())
		if err != nil {
			continue
		}

		versionDir := path.Join(packageDir, entry.Name())
		_, found, err := r.findPlatformDir(versionDir)
		if err != nil {
			return nil, err
		}
		if found {
			versions = append(versions, version)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Compare(versions[j]) < 0
	})

	return versions, nil
}

// Exists checks if a specific package version exists in this repository.
func (r *FileSystemRepository) Exists(ctx context.Context, org, name, version string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	versionDir := path.Join(r.basePath, org, name, version)
	_, found, err := r.findPlatformDir(versionDir)
	if err != nil {
		return false, err
	}
	return found, nil
}

// findPlatformDir resolves the platform-specific subdirectory of a versioned
// bala directory. Priority order:
//  1. "any" — platform-agnostic balas win when present.
//  2. First valid "go*" directory found — Go-targeted balas.
//
// A directory only qualifies when it contains either a Bala.toml (new format)
// or a package.json (legacy v3 format) — the marker the bala loader uses to
// dispatch.
func (r *FileSystemRepository) findPlatformDir(versionDir string) (string, bool, error) {
	if dir, ok, err := r.checkPlatformDir(path.Join(versionDir, platformAny)); err != nil {
		return "", false, err
	} else if ok {
		return dir, true, nil
	}

	entries, err := fs.ReadDir(r.fsys, versionDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), platformGoPrefix) {
			if dir, ok, err := r.checkPlatformDir(path.Join(versionDir, entry.Name())); err != nil {
				return "", false, err
			} else if ok {
				return dir, true, nil
			}
		}
	}
	return "", false, nil
}

// checkPlatformDir returns (path, true, nil) when platformPath is a directory
// holding a valid bala manifest (Bala.toml or legacy package.json).
func (r *FileSystemRepository) checkPlatformDir(platformPath string) (string, bool, error) {
	info, exists, err := statIfExists(r.fsys, platformPath)
	if err != nil || !exists || !info.IsDir() {
		return "", false, err
	}
	for _, marker := range []string{BalaTomlFile, "package.json"} {
		if _, found, err := statIfExists(r.fsys, path.Join(platformPath, marker)); err != nil {
			return "", false, err
		} else if found {
			return platformPath, true, nil
		}
	}
	return "", false, nil
}

// statIfExists returns (info, true, nil) if path exists, (nil, false, nil) if not found,
// or (nil, false, err) for other errors.
func statIfExists(fsys fs.FS, path string) (fs.FileInfo, bool, error) {
	info, err := fs.Stat(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return info, true, nil
}

var _ Repository = (*FileSystemRepository)(nil)
var _ bindableRepository = (*FileSystemRepository)(nil)

// defaultRepositories returns repositories for the standard repository locations
// using the given ballerinaEnvFs.
//
// The bundled repository is searched first so standard libraries baked into
// the binary resolve without touching the central cache. Falls through to the
// central cache when the bundle does not advertise the requested package.
//
// The central repository is exposed as a RemoteRepository whose on-disk cache
// is the central bala directory. The RemoteRepository currently has no remote
// source wired in, so it behaves as a cache-only read until that arrives.
func defaultRepositories(ballerinaEnvFs fs.FS) ([]Repository, map[string]Repository) {
	centralCache := NewFileSystemRepository(ballerinaEnvFs, centralCacheSubpath)
	localCache := NewFileSystemRepository(ballerinaEnvFs, localRepoCacheSubpath)
	chain := append(bundledRepositories(), NewRemoteRepository(centralCache))
	custom := map[string]Repository{"local": localCache}
	return chain, custom
}

// bundledRepositories returns the repositories baked into the binary: the lang
// libraries and the standard libraries. They are always searched first so
// these packages (e.g. ballerina/io) resolve even when callers supply their
// own repository list.
func bundledRepositories() []Repository {
	return []Repository{
		NewFileSystemRepository(langlibs.FS, "."),
		NewFileSystemRepository(stdlibs.FS, "."),
	}
}
