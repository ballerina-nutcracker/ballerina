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
	"maps"
	"slices"
	"sync"
)

// packageContext holds internal state for a Package.
// It manages module contexts and package-level metadata.
type packageContext struct {
	project              Project
	packageID            PackageID
	packageManifest      PackageManifest
	compilationOptions   CompilationOptions
	moduleContextMap     map[ModuleID]*moduleContext
	moduleIDs            []ModuleID
	defaultModuleContext *moduleContext       // cached default module
	ballerinaTomlContext *tomlDocumentContext // Ballerina.toml context (nil if not present)

	// Lazy-initialized fields (thread-safe via sync.Once, matching documentContext pattern).
	packageCompilation     *PackageCompilation
	compilationOnce        sync.Once
	packageResolution      *PackageResolution
	resolutionOnce         sync.Once
	modDependencyGraph     *DependencyGraph[ModuleDescriptor]
	modDependencyGraphOnce sync.Once
}

// newPackageContext creates a packageContext from PackageConfig.
func newPackageContext(project Project, packageConfig PackageConfig, compilationOptions CompilationOptions) *packageContext {
	// Determine if syntax tree should be disabled from compilation options
	disableSyntaxTree := compilationOptions.DisableSyntaxTree()

	// Build module context map from all modules
	moduleContextMap := make(map[ModuleID]*moduleContext)
	var moduleIDs []ModuleID

	// Add default module
	defaultModuleConfig := packageConfig.DefaultModule()
	defaultModuleCtx := newModuleContext(project, defaultModuleConfig, disableSyntaxTree)
	moduleContextMap[defaultModuleConfig.ModuleID()] = defaultModuleCtx
	moduleIDs = append(moduleIDs, defaultModuleConfig.ModuleID())

	// Add other modules
	for _, moduleConfig := range packageConfig.OtherModules() {
		moduleCtx := newModuleContext(project, moduleConfig, disableSyntaxTree)
		moduleContextMap[moduleConfig.ModuleID()] = moduleCtx
		moduleIDs = append(moduleIDs, moduleConfig.ModuleID())
	}

	// Create tomlDocumentContext for Ballerina.toml if present
	var ballerinaTomlCtx *tomlDocumentContext
	if packageConfig.HasBallerinaToml() {
		ballerinaTomlCtx = newTomlDocumentContext(packageConfig.BallerinaToml())
	}

	return &packageContext{
		project:              project,
		packageID:            packageConfig.PackageID(),
		packageManifest:      packageConfig.PackageManifest(),
		compilationOptions:   compilationOptions,
		moduleContextMap:     moduleContextMap,
		moduleIDs:            moduleIDs,
		defaultModuleContext: defaultModuleCtx,
		ballerinaTomlContext: ballerinaTomlCtx,
	}
}

// newPackageContextFromMaps creates a packageContext directly from module context maps.
// This is used for creating modified package contexts.
func newPackageContextFromMaps(
	project Project,
	packageID PackageID,
	packageManifest PackageManifest,
	compilationOptions CompilationOptions,
	moduleContextMap map[ModuleID]*moduleContext,
	ballerinaTomlContext *tomlDocumentContext,
) *packageContext {
	// Ensure moduleContextMap is initialized to prevent nil map panics
	if moduleContextMap == nil {
		moduleContextMap = make(map[ModuleID]*moduleContext)
	}

	// Build moduleIDs from map keys
	moduleIDs := make([]ModuleID, 0, len(moduleContextMap))
	var defaultModuleContext *moduleContext
	for id, ctx := range moduleContextMap {
		moduleIDs = append(moduleIDs, id)
		if ctx.isDefault() {
			defaultModuleContext = ctx
		}
	}

	return &packageContext{
		project:              project,
		packageID:            packageID,
		packageManifest:      packageManifest,
		compilationOptions:   compilationOptions,
		moduleContextMap:     moduleContextMap,
		moduleIDs:            moduleIDs,
		defaultModuleContext: defaultModuleContext,
		ballerinaTomlContext: ballerinaTomlContext,
	}
}

// getPackageID returns the package identifier.
func (p *packageContext) getPackageID() PackageID {
	return p.packageID
}

// getPackageName returns the package name.
func (p *packageContext) getPackageName() PackageName {
	return p.packageManifest.Name()
}

// getPackageOrg returns the package organization.
func (p *packageContext) getPackageOrg() PackageOrg {
	return p.packageManifest.Org()
}

// getPackageVersion returns the package version.
func (p *packageContext) getPackageVersion() PackageVersion {
	return p.packageManifest.Version()
}

// getDescriptor returns the package descriptor.
func (p *packageContext) getDescriptor() PackageDescriptor {
	return p.packageManifest.PackageDescriptor()
}

// getPackageManifest returns the manifest.
func (p *packageContext) getPackageManifest() PackageManifest {
	return p.packageManifest
}

// getCompilationOptions returns the compilation options.
func (p *packageContext) getCompilationOptions() CompilationOptions {
	return p.compilationOptions
}

// getModuleIDs returns a defensive copy of all module IDs.
func (p *packageContext) getModuleIDs() []ModuleID {
	return slices.Clone(p.moduleIDs)
}

// getModuleContext returns context for a module ID.
func (p *packageContext) getModuleContext(moduleID ModuleID) *moduleContext {
	return p.moduleContextMap[moduleID]
}

// getModuleContextByName returns context for a module name.
func (p *packageContext) getModuleContextByName(moduleName ModuleName) *moduleContext {
	for _, ctx := range p.moduleContextMap {
		if ctx.getModuleName().Equals(moduleName) {
			return ctx
		}
	}
	return nil
}

// getDefaultModuleContext returns the default module context.
// The default module context is always set during construction (newPackageContext
// or newPackageContextFromMaps), so this should never return nil for valid packages.
// Panics if no default module is found as a safety guard.
func (p *packageContext) getDefaultModuleContext() *moduleContext {
	if p.defaultModuleContext == nil {
		panic("Default module not found. This is a bug in the Project API")
	}
	return p.defaultModuleContext
}

// getProject returns the project reference.
func (p *packageContext) getProject() Project {
	return p.project
}

// getModuleContextMap returns a shallow copy of the module context map.
func (p *packageContext) getModuleContextMap() map[ModuleID]*moduleContext {
	return maps.Clone(p.moduleContextMap)
}

// getPackageCompilation returns the cached PackageCompilation, creating it on first call.
// Uses sync.Once for thread-safe lazy initialization.
func (p *packageContext) getPackageCompilation() *PackageCompilation {
	p.compilationOnce.Do(func() {
		p.packageCompilation = newPackageCompilation(p, p.compilationOptions)
	})
	return p.packageCompilation
}

// getResolution returns the cached PackageResolution, creating it on first call.
// Uses sync.Once for thread-safe lazy initialization.
func (p *packageContext) getResolution() *PackageResolution {
	p.resolutionOnce.Do(func() {
		env := p.project.Environment()
		p.packageResolution = newPackageResolution(p, env)
	})
	return p.packageResolution
}

// containsModule checks if the package contains a module with the given ID.
func (p *packageContext) containsModule(moduleID ModuleID) bool {
	_, ok := p.moduleContextMap[moduleID]
	return ok
}

// getBallerinaTomlContext returns the Ballerina.toml context, or nil if not present.
func (p *packageContext) getBallerinaTomlContext() *tomlDocumentContext {
	return p.ballerinaTomlContext
}

// moduleDependencyGraph returns the module dependency graph for this package.
// The graph contains only modules within this package (not external dependencies).
// For source packages, it analyzes imports. For bala packages, it returns a simple
// graph with just nodes (no import analysis needed since they're pre-compiled).
func (p *packageContext) moduleDependencyGraph() *DependencyGraph[ModuleDescriptor] {
	p.modDependencyGraphOnce.Do(func() {
		p.modDependencyGraph = p.buildModuleDependencyGraph()
	})
	return p.modDependencyGraph
}

// buildModuleDependencyGraph constructs a dependency graph for modules within this package.
func (p *packageContext) buildModuleDependencyGraph() *DependencyGraph[ModuleDescriptor] {
	builder := newDependencyGraphBuilder(moduleDescriptorCmp)

	// Add all modules as nodes
	for _, modID := range p.moduleIDs {
		modCtx := p.moduleContextMap[modID]
		if modCtx != nil {
			builder.addNode(modCtx.getDescriptor())
		}
	}

	// For source packages, analyze imports to find intra-package dependencies
	if p.project.Kind() != ProjectKindBala {
		env := p.project.Environment()
		// nil blendedManifest: intra-package edges never trip the
		// user-specified-repository branch in resolveRequest.
		moduleResolver := newModuleResolver(p.getDescriptor(), nil, env)

		for _, modID := range p.moduleIDs {
			modCtx := p.moduleContextMap[modID]
			if modCtx == nil {
				continue
			}

			fromDesc := modCtx.getDescriptor()

			// Get all module load requests for this module
			requests := modCtx.populateModuleLoadRequests()
			requests = append(requests, modCtx.populateTestModuleLoadRequests()...)

			// Resolve requests and add edges for same-package dependencies
			responses := moduleResolver.resolveModuleLoadRequests(context.Background(), requests)
			for _, resp := range responses {
				if resp.resolutionStatus == resolutionStatusResolved {
					toDesc := resp.moduleDesc
					// Only add edge if it's a different module within the same package
					// (packageDescriptor is nil for same-package modules)
					if !fromDesc.Equals(toDesc) && resp.packageDescriptor == nil {
						builder.addDependency(fromDesc, toDesc)
					}
				}
			}
		}
	}

	return builder.build()
}

// duplicate creates a copy of the context.
// The duplicated context has all module contexts duplicated as well.
func (p *packageContext) duplicate(project Project) *packageContext {
	// Duplicate module contexts
	moduleContextMap := make(map[ModuleID]*moduleContext, len(p.moduleIDs))
	for _, moduleID := range p.moduleIDs {
		if moduleCtx := p.moduleContextMap[moduleID]; moduleCtx != nil {
			moduleContextMap[moduleID] = moduleCtx.duplicate(project)
		}
	}

	return newPackageContextFromMaps(
		project,
		p.packageID,
		p.packageManifest,
		p.compilationOptions,
		moduleContextMap,
		p.ballerinaTomlContext, // Ballerina.toml is immutable, can share reference
	)
}
