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
	"fmt"

	"ballerina/tools/diagnostics"
)

// PackageResolution holds the result of package dependency resolution.
// It builds a topologically sorted list of modules within the root package,
// respecting inter-module dependencies discovered from import statements.
type PackageResolution struct {
	rootPackageContext            *packageContext
	moduleResolver                *moduleResolver
	moduleDependencyGraph         *DependencyGraph[ModuleDescriptor]
	packageDependencyGraph        *DependencyGraph[PackageDescriptor]
	topologicallySortedModuleList []*moduleContext
	resolvedDependencies          map[string]*PackageDescriptor // org/name -> PackageDescriptor
	diagnostics                   []diagnostics.Diagnostic
	diagnosticResult              DiagnosticResult
	environment                   *Environment
	// blendedManifest is the per-resolution view of consumer dependency policy.
	blendedManifest *blendedManifest
}

// addDiagnostic accumulates a diagnostic raised during dependency resolution.
// Wired into blendedManifest so its validation warnings surface through
// PackageCompilation diagnostics.
func (r *PackageResolution) addDiagnostic(d diagnostics.Diagnostic) {
	r.diagnostics = append(r.diagnostics, d)
}

func newPackageResolution(pkgCtx *packageContext, env *Environment) *PackageResolution {
	r := &PackageResolution{
		rootPackageContext:   pkgCtx,
		resolvedDependencies: make(map[string]*PackageDescriptor),
		environment:          env,
	}

	// Must run before any BFS work so dequeues can consult it.
	r.blendedManifest = newBlendedManifest(
		withPackageManifest(pkgCtx.getPackageManifest()),
		withPackageResolver(env.PackageResolver(), env.ResolutionOptions()),
		withDiagnosticReporter(r.addDiagnostic),
	)

	// moduleResolver consults blendedManifest for direct-import routing; the
	// consumer's resolver runs before BFS cache pre-population.
	r.moduleResolver = newModuleResolver(pkgCtx.getDescriptor(), r.blendedManifest, env)

	// Build dependency graph from imports
	r.buildModuleDependencyGraph()

	// Build package dependency graph
	r.buildPackageDependencyGraph()

	// Resolve dependencies (topological sort)
	r.resolveDependencies()
	return r
}

func (r *PackageResolution) collectModuleDescriptors() []ModuleDescriptor {
	pkgCtx := r.rootPackageContext
	moduleDescs := make([]ModuleDescriptor, 0, len(pkgCtx.moduleIDs))
	for _, modID := range pkgCtx.moduleIDs {
		modCtx := pkgCtx.moduleContextMap[modID]
		if modCtx != nil {
			moduleDescs = append(moduleDescs, modCtx.getDescriptor())
		}
	}
	return moduleDescs
}

func (r *PackageResolution) buildModuleDependencyGraph() {
	pkgCtx := r.rootPackageContext
	builder := newDependencyGraphBuilder(moduleDescriptorCmp)

	// Add all modules as nodes first
	for _, modID := range pkgCtx.moduleIDs {
		modCtx := pkgCtx.moduleContextMap[modID]
		if modCtx != nil {
			builder.addNode(modCtx.getDescriptor())
		}
	}

	// Process each module's imports and add edges
	for _, modID := range pkgCtx.moduleIDs {
		modCtx := pkgCtx.moduleContextMap[modID]
		if modCtx == nil {
			continue
		}

		fromDesc := modCtx.getDescriptor()

		// Get all module load requests for this module
		requests := modCtx.populateModuleLoadRequests()
		requests = append(requests, modCtx.populateTestModuleLoadRequests()...)

		// Resolve requests and add edges
		responses := r.moduleResolver.resolveModuleLoadRequests(context.Background(), requests)
		for _, resp := range responses {
			if resp.resolutionStatus == resolutionStatusResolved {
				toDesc := resp.moduleDesc
				// Only add edge if the dependency is a different module
				if !fromDesc.Equals(toDesc) {
					builder.addDependency(fromDesc, toDesc)
				}
			}
		}
	}

	r.moduleDependencyGraph = builder.build()
}

func (r *PackageResolution) buildPackageDependencyGraph() {
	builder := newDependencyGraphBuilder(packageDescriptorCmp)
	ctx := context.Background()

	// Add root package as a node
	rootDesc := r.rootPackageContext.getDescriptor()
	builder.addNode(rootDesc)

	// Visited set keyed by org/name (first-seen wins for version conflicts)
	visited := make(map[string]bool)
	rootKey := rootDesc.Org().Value() + "/" + rootDesc.Name().Value()
	visited[rootKey] = true

	// Collect direct dependencies from root package's module imports
	var directDeps []*PackageDescriptor
	for _, modID := range r.rootPackageContext.moduleIDs {
		modCtx := r.rootPackageContext.moduleContextMap[modID]
		if modCtx == nil {
			continue
		}

		requests := modCtx.populateModuleLoadRequests()
		requests = append(requests, modCtx.populateTestModuleLoadRequests()...)

		responses := r.moduleResolver.resolveModuleLoadRequests(ctx, requests)
		for _, resp := range responses {
			if resp.resolutionStatus == resolutionStatusResolved && resp.packageDescriptor != nil {
				pkgDesc := resp.packageDescriptor
				key := pkgDesc.Org().Value() + "/" + pkgDesc.Name().Value()

				if !visited[key] {
					visited[key] = true
					r.resolvedDependencies[key] = pkgDesc
					builder.addNode(*pkgDesc)
					builder.addDependency(rootDesc, *pkgDesc)
					directDeps = append(directDeps, pkgDesc)
				}
			}
		}
	}

	// BFS for transitive dependencies
	r.resolveTransitiveDependencies(ctx, builder, directDeps, visited)

	r.packageDependencyGraph = builder.build()
}

// resolveTransitiveDependencies uses BFS to resolve transitive dependencies.
// Uses first-seen wins for version conflict resolution.
func (r *PackageResolution) resolveTransitiveDependencies(
	ctx context.Context,
	builder *dependencyGraphBuilder[PackageDescriptor],
	directDeps []*PackageDescriptor,
	visited map[string]bool,
) {
	// BFS queue of package descriptors to process
	queue := make([]PackageDescriptor, 0, len(directDeps))
	for _, dep := range directDeps {
		queue = append(queue, *dep)
	}

	resolver := r.environment.PackageResolver()
	options := r.environment.ResolutionOptions()

	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]

		// Overlay the user-specified repository (if any) for this (org, name).
		// Applies to both direct deps and transitive children since every
		// request is built here at dequeue time.
		var request ResolutionRequest
		if blended, ok := r.blendedManifest.dependency(current.Org().Value(), current.Name().Value()); ok && blended.Repository() != "" {
			request = newResolutionRequestWithRepository(current, blended.Repository())
		} else {
			request = NewResolutionRequest(current)
		}
		responses := resolver.ResolvePackages(ctx, []ResolutionRequest{request}, options)
		if len(responses) == 0 || !responses[0].IsResolved() {
			continue
		}

		pkg := responses[0].Package()
		if pkg == nil {
			continue
		}

		// Get transitive dependencies from the package's manifest
		for _, dep := range pkg.Manifest().Dependencies() {
			key := dep.Org().Value() + "/" + dep.Name().Value()

			if visited[key] {
				// Node was already discovered via another path (e.g. as a direct
				// dep of root). We must still record this edge so that topological
				// sort preserves the correct compilation order.
				if existing := r.resolvedDependencies[key]; existing != nil {
					builder.addDependency(current, *existing)
				}
				continue
			}
			visited[key] = true

			depDesc := NewPackageDescriptor(dep.Org(), dep.Name(), dep.Version())

			// Track resolved dependency
			r.resolvedDependencies[key] = &depDesc

			// Add to graph
			builder.addNode(depDesc)
			builder.addDependency(current, depDesc)

			// Enqueue for further traversal
			queue = append(queue, depDesc)
		}
	}
}

// bundledLangLibs lists the lang libraries that are migrated to bala bundles
// and are used implicitly (without an import statement), so they must be
// compiled ahead of the root package's modules and seeded into the
// implicit-imports map. Explicitly imported bundles (e.g. ballerina/io) are
// not listed here; they resolve through normal dependency resolution from the
// bundled langlibs repository.
var bundledLangLibs = []struct{ org, name, version string }{
	{"ballerina", "lang.int", "0.0.1"},
	{"ballerina", "lang.boolean", "0.0.1"},
	{"ballerina", "lang.decimal", "0.0.1"},
	{"ballerina", "lang.error", "0.0.1"},
	{"ballerina", "lang.float", "0.0.1"},
	{"ballerina", "lang.string", "0.0.1"},
	{"ballerina", "lang.value", "0.0.1"},
	{"ballerina", "lang.xml", "0.0.1"},
	{"ballerina", "lang.float", "0.0.1"},
	{"ballerina", "lang.array", "0.0.1"},
	{"ballerina", "lang.map", "0.0.1"},
	{"ballerina", "lang.object", "0.0.1"},
	{"ballerina", "lang.runtime", "0.0.1"},
}

// bundledLangLibModules resolves the migrated lang libraries and returns their
// module contexts (compiled ahead of the root package's modules). They are not
// added to the package dependency graph. A lib is skipped when the root package
// is that lib itself.
func (r *PackageResolution) bundledLangLibModules() []*moduleContext {
	rootDesc := r.rootPackageContext.getDescriptor()
	resolver := r.environment.PackageResolver()
	options := r.environment.ResolutionOptions()
	var modules []*moduleContext
	for _, lib := range bundledLangLibs {
		if rootDesc.Org().Value() == lib.org && rootDesc.Name().Value() == lib.name {
			continue
		}
		// Bundled lang libs ship with the compiler, so a resolution failure is
		// an internal bug rather than a user error: fail fast instead of
		// letting it surface as downstream undefined-symbol/type errors.
		desc, err := NewPackageDescriptorFromStrings(lib.org, lib.name, lib.version)
		if err != nil {
			panic(fmt.Sprintf("failed to build descriptor for bundled lang library %q: %v", bundledLangLibID(lib.org, lib.name, lib.version), err))
		}
		responses := resolver.ResolvePackages(context.Background(), []ResolutionRequest{NewResolutionRequest(desc)}, options)
		if len(responses) == 0 || !responses[0].IsResolved() {
			panic(fmt.Sprintf("failed to resolve bundled lang library %q", bundledLangLibID(lib.org, lib.name, lib.version)))
		}
		pkgCtx := responses[0].Package().packageCtx
		for _, modDesc := range pkgCtx.moduleDependencyGraph().ToTopologicallySortedList() {
			if modCtx := pkgCtx.getModuleContextByName(modDesc.Name()); modCtx != nil {
				modules = append(modules, modCtx)
			}
		}
	}
	return modules
}

func bundledLangLibID(org, name, version string) string {
	return fmt.Sprintf("%s/%s:%s", org, name, version)
}

// prependImplicitLangLibs returns the bundled lang-lib modules that are not
// already present in resolved, so an explicitly imported lang lib (resolved via
// normal dependency resolution) is not compiled and published a second time.
func prependImplicitLangLibs(bundled, resolved []*moduleContext) []*moduleContext {
	existing := make(map[ModuleID]bool, len(resolved))
	for _, mod := range resolved {
		existing[mod.getModuleID()] = true
	}
	var implicit []*moduleContext
	for _, mod := range bundled {
		if existing[mod.getModuleID()] {
			continue
		}
		existing[mod.getModuleID()] = true
		implicit = append(implicit, mod)
	}
	return implicit
}

// ResolvedDependencies returns the map of resolved external package dependencies.
func (r *PackageResolution) ResolvedDependencies() map[string]*PackageDescriptor {
	return r.resolvedDependencies
}

// DependencyGraph returns the package-level dependency graph.
func (r *PackageResolution) DependencyGraph() *DependencyGraph[PackageDescriptor] {
	return r.packageDependencyGraph
}

// ModuleDependencyGraph returns the module-level dependency graph.
func (r *PackageResolution) ModuleDependencyGraph() *DependencyGraph[ModuleDescriptor] {
	return r.moduleDependencyGraph
}

func (r *PackageResolution) resolveDependencies() {
	var sortedModuleList []*moduleContext

	// Sort packages topologically (dependencies before dependents)
	sortedPackages := r.packageDependencyGraph.ToTopologicallySortedList()

	packageCache := r.environment.PackageCache()

	// For each package in topological order, add its modules (sorted within the package)
	for _, pkgDesc := range sortedPackages {
		var pkgCtx *packageContext

		// Check if this is the root package
		if pkgDesc.Equals(r.rootPackageContext.getDescriptor()) {
			pkgCtx = r.rootPackageContext
		} else {
			// Get external package from cache
			cachedPkg := packageCache.Get(pkgDesc.Org().Value(), pkgDesc.Name().Value(), pkgDesc.Version().String())
			if cachedPkg == nil {
				continue
			}
			pkgCtx = cachedPkg.packageCtx
		}

		// Get module dependency graph for this package
		moduleDependencyGraph := pkgCtx.moduleDependencyGraph()

		// Topologically sort modules within this package
		sortedModuleDescs := moduleDependencyGraph.ToTopologicallySortedList()

		// Add module contexts in sorted order
		for _, modDesc := range sortedModuleDescs {
			modCtx := pkgCtx.getModuleContextByName(modDesc.Name())
			if modCtx != nil {
				sortedModuleList = append(sortedModuleList, modCtx)
			}
		}
	}

	// Check for cycles in root package's module graph
	cycles := r.moduleDependencyGraph.FindCycles()
	// TODO(P7): Create proper cycle diagnostics with DiagnosticCode
	_ = cycles

	// TODO: avoid always adding implicit lang libs here. Instead we need to think of away from front end to signal
	// to driver which implicit imports were added.
	bundledModules := r.bundledLangLibModules()
	sortedModuleList = append(prependImplicitLangLibs(bundledModules, sortedModuleList), sortedModuleList...)

	r.topologicallySortedModuleList = sortedModuleList
	r.diagnosticResult = NewDiagnosticResult(r.diagnostics)
}

// DiagnosticResult returns the diagnostics from resolution.
func (r *PackageResolution) DiagnosticResult() DiagnosticResult {
	return r.diagnosticResult
}
