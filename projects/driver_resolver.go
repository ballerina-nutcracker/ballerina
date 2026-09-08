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
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// DriverDependencyResolver adapts the projects package resolution machinery to
// the driver.DependencyResolver interface, translating resolved packages into
// driver source descriptions and remembering the platform-specific bala
// projects that were pulled in along the way.
type DriverDependencyResolver struct {
	environment *Environment
	mu          sync.Mutex
	native      map[driver.PackageDescriptor]*BalaProject
}

// NewDriverDependencyResolver creates a resolver over projectFS, building the
// package environment (and its repositories) from cfg; when cfg carries no
// build options the defaults are used.
func NewDriverDependencyResolver(projectFS fs.FS, cfg ProjectLoadConfig) *DriverDependencyResolver {
	loader := newProjectLoader(projectFS, cfg.BallerinaEnvFs)
	buildOptions := NewBuildOptions()
	if cfg.BuildOptions != nil {
		buildOptions = *cfg.BuildOptions
	}
	environment := loader.createEnvironmentWithRepositories(cfg, buildOptions)
	return &DriverDependencyResolver{
		environment: environment,
		native:      make(map[driver.PackageDescriptor]*BalaProject),
	}
}

// CompilerEnvironment returns the shared compiler environment of the
// underlying package environment, so callers can build compiler contexts that
// agree with the resolved dependencies.
func (r *DriverDependencyResolver) CompilerEnvironment() *compilercontext.CompilerEnvironment {
	return r.environment.CompilerEnvironment()
}

// Resolve satisfies one driver dependency request: it resolves the requested
// package (by exact version, or by latest when the version is empty), falling
// back to the default repositories with a warning diagnostic when a custom
// repository does not have it. On success it returns the package's file
// system, its slash-cleaned source root and the driver-facing sources. A
// package that cannot be found, or that lacks the requested module, yields a
// nil result with no error so the driver can report the failure itself.
func (r *DriverDependencyResolver) Resolve(ctx context.Context, request driver.DependencyRequest) (
	fsys fs.FS, packageRoot string, sources *driver.PackageSources,
	discoveryDiagnostics []diagnostics.Diagnostic, err error,
) {
	options := NewResolutionOptions().WithOffline(request.Resolution.Offline).WithSticky(request.Resolution.Sticky)
	var pkg *Package
	var resolverDiagnostics []diagnostics.Diagnostic
	if request.Descriptor.Version == "" {
		pkg = r.resolveVersionless(ctx, request, options)
	} else {
		version, parseErr := NewPackageVersionFromString(request.Descriptor.Version)
		if parseErr != nil {
			return nil, "", nil, nil, parseErr
		}
		descriptor := NewPackageDescriptor(NewPackageOrg(request.Descriptor.Org), NewPackageName(request.Descriptor.Name), version)
		resolutionRequest := NewResolutionRequest(descriptor)
		if request.Repository != "" {
			resolutionRequest = newResolutionRequestWithRepository(descriptor, request.Repository)
		}
		responses := r.environment.PackageResolver().ResolvePackages(ctx, []ResolutionRequest{resolutionRequest}, options)
		if len(responses) > 0 && responses[0].IsResolved() {
			pkg = responses[0].Package()
		}
		if pkg == nil && request.Repository != "" {
			responses = r.environment.PackageResolver().ResolvePackages(ctx,
				[]ResolutionRequest{NewResolutionRequest(descriptor)}, options)
			if len(responses) > 0 && responses[0].IsResolved() {
				pkg = responses[0].Package()
				resolverDiagnostics = append(resolverDiagnostics, driverRepositoryFallbackDiagnostic(request))
			}
		}
	}
	if pkg == nil {
		return nil, "", nil, nil, nil
	}
	converted := driverSourcesFromPackage(pkg)
	if request.ModuleName != "" && !containsDriverModule(converted, request.ModuleName) {
		return nil, "", nil, nil, nil
	}
	project := pkg.Project()
	projectFS := project.Environment().fs()
	if bala, ok := project.(*BalaProject); ok {
		projectFS = bala.fsys
		if bala.Platform() != BalaPlatformAny {
			r.mu.Lock()
			r.native[converted.Descriptor] = bala
			r.mu.Unlock()
		}
	}
	discoveryDiagnostics = append(resolverDiagnostics, pkg.Manifest().Diagnostics()...)
	return projectFS, path.Clean(pathToSlash(project.SourceRoot())), converted, discoveryDiagnostics, nil
}

// driverRepositoryFallbackDiagnostic builds the Ballerina.toml warning emitted
// when a dependency is missing from its declared custom repository and was
// instead resolved from the default repositories.
func driverRepositoryFallbackDiagnostic(request driver.DependencyRequest) diagnostics.Diagnostic {
	info := diagnostics.NewDiagnosticInfo(nil,
		"dependency %s/%s:%s cannot be found in the '%s' repository. falling back to default repositories",
		diagnostics.Warning)
	return diagnostics.NewDefaultDiagnostic(info, diagnostics.NewBallerinaTomlLocation(0, 0), nil,
		request.Descriptor.Org, request.Descriptor.Name, request.Descriptor.Version, request.Repository)
}

// resolveVersionless resolves a request that named no version by picking the
// latest available one: through the package resolver's by-name lookup for the
// default repositories, or by listing the versions of the named custom
// repository and taking the highest. It returns nil when nothing matches.
func (r *DriverDependencyResolver) resolveVersionless(ctx context.Context, request driver.DependencyRequest,
	options ResolutionOptions,
) *Package {
	if request.Repository == "" {
		packages := r.environment.PackageResolver().ResolveByName(ctx, request.Descriptor.Org, request.Descriptor.Name, options)
		if len(packages) > 0 {
			return packages[0]
		}
		return nil
	}
	resolver, ok := r.environment.PackageResolver().(*defaultPackageResolver)
	if !ok {
		return nil
	}
	repository, ok := resolver.customRepos[request.Repository]
	if !ok {
		return nil
	}
	versions, err := repository.GetPackageVersions(ctx, request.Descriptor.Org, request.Descriptor.Name, options)
	if err != nil {
		return nil
	}
	version, ok := pickLatest(versions, identityVersion)
	if !ok {
		return nil
	}
	pkg, err := repository.GetPackage(ctx, request.Descriptor.Org, request.Descriptor.Name, version.String(), options)
	if err != nil {
		return nil
	}
	return pkg
}

// NativeProjects returns the distinct platform-specific bala projects recorded
// during resolution that back the given modules, in module order; it is how the
// CLI learns which native artifacts an execution needs.
func (r *DriverDependencyResolver) NativeProjects(modules []driver.ModuleDescriptor) []*BalaProject {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]*BalaProject, 0)
	seen := make(map[driver.PackageDescriptor]bool)
	for _, module := range modules {
		if seen[module.Package] {
			continue
		}
		seen[module.Package] = true
		if project := r.native[module.Package]; project != nil {
			result = append(result, project)
		}
	}
	return result
}

// driverSourcesFromPackage converts a resolved package into the driver's
// PackageSources view: its descriptor, its manifest dependencies, and its
// modules with default-module-first ordering and sorted, source-root-relative
// document paths.
func driverSourcesFromPackage(pkg *Package) *driver.PackageSources {
	descriptor := driverPackageDescriptor(pkg.Descriptor())
	manifest := driver.Manifest{}
	for _, dependency := range pkg.Manifest().Dependencies() {
		manifest.Dependencies = append(manifest.Dependencies, driver.Dependency{
			Org: dependency.Org().Value(), Name: dependency.Name().Value(), Version: dependency.Version().String(), Repository: dependency.Repository(),
		})
	}
	modules := pkg.Modules()
	sort.SliceStable(modules, func(i, j int) bool {
		if modules[i].IsDefaultModule() != modules[j].IsDefaultModule() {
			return modules[i].IsDefaultModule()
		}
		return modules[i].ModuleName().String() < modules[j].ModuleName().String()
	})
	result := &driver.PackageSources{Descriptor: descriptor, Manifest: manifest, Modules: make([]*driver.ModuleSources, len(modules))}
	root := path.Clean(pathToSlash(pkg.Project().SourceRoot()))
	for index, module := range modules {
		sources := &driver.ModuleSources{ID: driver.ModuleDescriptor{Package: descriptor, Name: module.ModuleName().String()}}
		for _, id := range module.DocumentIDs() {
			sources.Documents = append(sources.Documents, relativeProjectPath(root, pkg.Project().DocumentPath(id)))
		}
		for _, id := range module.TestDocumentIDs() {
			sources.TestDocuments = append(sources.TestDocuments, relativeProjectPath(root, pkg.Project().DocumentPath(id)))
		}
		sort.Strings(sources.Documents)
		sort.Strings(sources.TestDocuments)
		result.Modules[index] = sources
	}
	return result
}

// driverPackageDescriptor converts a projects package descriptor into the
// driver's plain string-valued form.
func driverPackageDescriptor(descriptor PackageDescriptor) driver.PackageDescriptor {
	return driver.PackageDescriptor{Org: descriptor.Org().Value(), Name: descriptor.Name().Value(), Version: descriptor.Version().String()}
}

// relativeProjectPath rewrites a project document path into the slash-separated
// path relative to root that the driver addresses documents by; a root of "."
// leaves the cleaned path as is.
func relativeProjectPath(root, value string) string {
	value = path.Clean(pathToSlash(value))
	if root == "." {
		return value
	}
	return strings.TrimPrefix(value, root+"/")
}

// pathToSlash rewrites Windows separators as forward slashes, so paths coming
// from the host file system can be used as io/fs and driver paths.
func pathToSlash(value string) string { return strings.ReplaceAll(value, "\\", "/") }

// containsDriverModule reports whether sources declares a module with the given
// name.
func containsDriverModule(sources *driver.PackageSources, name string) bool {
	for _, module := range sources.Modules {
		if module.ID.Name == name {
			return true
		}
	}
	return false
}

var _ driver.DependencyResolver = (*DriverDependencyResolver)(nil)
