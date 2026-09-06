// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

type DriverDependencyResolver struct {
	environment *Environment
	mu          sync.Mutex
	native      map[driver.PackageDescriptor]*BalaProject
}

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

func (r *DriverDependencyResolver) CompilerEnvironment() *compilercontext.CompilerEnvironment {
	return r.environment.CompilerEnvironment()
}

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

func driverRepositoryFallbackDiagnostic(request driver.DependencyRequest) diagnostics.Diagnostic {
	info := diagnostics.NewDiagnosticInfo(nil,
		"dependency %s/%s:%s cannot be found in the '%s' repository. falling back to default repositories",
		diagnostics.Warning)
	return diagnostics.NewDefaultDiagnostic(info, diagnostics.NewBallerinaTomlLocation(0, 0), nil,
		request.Descriptor.Org, request.Descriptor.Name, request.Descriptor.Version, request.Repository)
}

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

func driverPackageDescriptor(descriptor PackageDescriptor) driver.PackageDescriptor {
	return driver.PackageDescriptor{Org: descriptor.Org().Value(), Name: descriptor.Name().Value(), Version: descriptor.Version().String()}
}

func relativeProjectPath(root, value string) string {
	value = path.Clean(pathToSlash(value))
	if root == "." {
		return value
	}
	return strings.TrimPrefix(value, root+"/")
}

func pathToSlash(value string) string { return strings.ReplaceAll(value, "\\", "/") }

func containsDriverModule(sources *driver.PackageSources, name string) bool {
	for _, module := range sources.Modules {
		if module.ID.Name == name {
			return true
		}
	}
	return false
}

var _ driver.DependencyResolver = (*DriverDependencyResolver)(nil)
