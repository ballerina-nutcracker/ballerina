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
	"fmt"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// PackageCompilation represents compilation at package level by resolving all the dependencies.
// Java source: io.ballerina.projects.PackageCompilation
type PackageCompilation struct {
	rootPackageContext    *packageContext
	packageResolution     *PackageResolution
	compilationOptions    CompilationOptions
	compilerBackends      map[TargetPlatform]CompilerBackend
	backendMu             sync.Mutex
	pluginDiagnostics     []diagnostics.Diagnostic
	diagnosticResult      DiagnosticResult
	compilerEnv           *context.CompilerEnvironment
	compileOnce           sync.Once
	compilerPluginManager *compilerPluginResolver
}

// newPackageCompilation creates a PackageCompilation and triggers compilation.
// Java source: PackageCompilation.from(PackageContext, CompilationOptions)
func newPackageCompilation(rootPkgCtx *packageContext, compilationOptions CompilationOptions) *PackageCompilation {
	compilation := &PackageCompilation{
		rootPackageContext: rootPkgCtx,
		packageResolution:  rootPkgCtx.getResolution(),
		compilationOptions: compilationOptions,
		compilerBackends:   make(map[TargetPlatform]CompilerBackend),
		compilerEnv:        rootPkgCtx.project.Environment().compilerEnvironment(),
	}

	compilation.compile()

	// TODO(P6): Run code analyzers if project has updated only

	return compilation
}

// compile triggers one-time module compilation using sync.Once.
// Java source: PackageCompilation.compileModules()
func (c *PackageCompilation) compile() {
	c.compileOnce.Do(c.compileModulesInternal)
}

// compileModulesInternal performs the actual compilation of all modules.
func (c *PackageCompilation) compileModulesInternal() {
	var allDiagnostics []diagnostics.Diagnostic

	// Add resolution diagnostics
	allDiagnostics = append(allDiagnostics, c.packageResolution.DiagnosticResult().Diagnostics()...)

	// Add package manifest diagnostics
	allDiagnostics = append(allDiagnostics, c.getPackageContext().getPackageManifest().Diagnostics()...)

	// TODO(P6): Add dependency manifest diagnostics once DependencyManifest is migrated
	// allDiagnostics = append(allDiagnostics, c.getPackageContext().dependencyManifest().Diagnostics()...)

	// Add compilation diagnostics if no resolution errors
	if !c.packageResolution.DiagnosticResult().HasErrors() {
		// Register all module source files with the shared DiagnosticEnv using
		// Register source files with the shared DiagnosticEnv. The key includes
		// a per-package prefix (see documentContext.diagKeyPrefix) so same-basename
		// files across packages don't collide.
		de := c.compilerEnv.DiagnosticEnv()
		for _, moduleCtx := range c.packageResolution.topologicallySortedModuleList {
			for _, docCtx := range moduleCtx.srcDocContextMap {
				de.RegisterFile(docCtx.registrationKey(), docCtx.getTextDocument())
			}
		}

		// Build descriptor -> moduleContext lookup so we can propagate errors
		// from a dependency to its dependents and skip downstream stages.
		descToModule := make(map[ModuleDescriptor]*moduleContext, len(c.packageResolution.topologicallySortedModuleList))
		for _, moduleCtx := range c.packageResolution.topologicallySortedModuleList {
			descToModule[moduleCtx.getDescriptor()] = moduleCtx
		}
		erroredModules := make(map[ModuleID]struct{})
		moduleDepGraph := c.packageResolution.ModuleDependencyGraph()
		dependencyErrored := func(m *moduleContext) bool {
			for _, dep := range moduleDepGraph.DirectDependencies(m.getDescriptor()) {
				depCtx, ok := descToModule[dep]
				if !ok {
					continue
				}
				if _, errored := erroredModules[depCtx.getModuleID()]; errored {
					return true
				}
			}
			return false
		}

		// Phase 1a: Parse, AST build. No cross-module dependency, so this runs
		// concurrently across the whole package regardless of topological order.
		//
		// Exception: --dump-tokens/--dump-st/--dump-ast/--dump-recovered-ast write straight
		// to stderr with no ordering between modules, so concurrent parsing would interleave
		// dumped output nondeterministically. Parse sequentially when any is set instead.
		opts := c.compilationOptions
		runPerModule(c.packageResolution.topologicallySortedModuleList,
			!opts.DumpTokens() && !opts.DumpST() && !opts.DumpAST() && !opts.DumpRecoveredAST(),
			func(m *moduleContext) {
				if m.getCompilationState() != moduleCompilationStateLoadedFromSources {
					// TODO: Handle LOADED_FROM_CACHE state - load symbols from BIR
					return
				}
				parseModule(m)
			})

		// Phase 1b: import, symbol, and top-level type resolution. Sequential, since a
		// module's resolution needs its dependencies' published symbols. Runs for every
		// module so all top-level diagnostics are collected in one shot, but a dependent
		// of an errored module is skipped to avoid cascading noise.
		for _, moduleCtx := range c.packageResolution.topologicallySortedModuleList {
			if moduleCtx.getCompilationState() != moduleCompilationStateLoadedFromSources {
				continue
			}
			if dependencyErrored(moduleCtx) {
				erroredModules[moduleCtx.getModuleID()] = struct{}{}
				continue
			}
			resolveSymbolsAndTypes(moduleCtx)
			if moduleCtx.compilerCtx.HasErrors() {
				erroredModules[moduleCtx.getModuleID()] = struct{}{}
			}
		}

		// If any module failed top-level resolution, stop the compilation pipeline
		// here. Subsequent stages (local node resolution, semantic analysis, CFG,
		// desugar, BIR) operate on assumptions that top-level types are fully
		// resolved across the whole package; running them now produces noisy
		// cascading diagnostics or panics on nil types.
		if len(erroredModules) > 0 {
			c.collectModuleDiagnostics(&allDiagnostics, false)
			c.diagnosticResult = NewDiagnosticResult(allDiagnostics)
			return
		}

		modules := c.packageResolution.topologicallySortedModuleList
		// Phase 2: local node resolution, semantic analysis, and CFG analysis (parallel -
		// no cross-module dependencies).
		runPerModule(modules, true, analyzeAndCreateCFG)
		if modulesHaveErrors(modules) {
			c.collectModuleDiagnostics(&allDiagnostics, false)
			c.diagnosticResult = NewDiagnosticResult(allDiagnostics)
			return
		}

		// Phase 3: compiler plugins. This is a package-wide barrier before desugaring.
		c.compilerPluginManager = newCompilerPluginResolver(modules)
		if modulesHaveErrors(modules) {
			c.collectModuleDiagnostics(&allDiagnostics, false)
			c.diagnosticResult = NewDiagnosticResult(allDiagnostics)
			return
		}
		runPerModule(modules, true, func(module *moduleContext) {
			c.runCompilerPlugins(module)
		})
		if modulesHaveErrors(modules) {
			c.collectModuleDiagnostics(&allDiagnostics, false)
			c.diagnosticResult = NewDiagnosticResult(allDiagnostics)
			return
		}

		// Phase 4: desugar (parallel - no cross-module dependencies, and every module
		// completed plugins without errors), pipelined with BIR generation when
		// GenerateCode is set.
		generateCode := c.compilationOptions.GenerateCode()
		runPerModule(modules, true, func(m *moduleContext) {
			desugarModule(m)
			if generateCode && m.getCompilationState() == moduleCompilationStateCompiled {
				generateCodeInternal(m)
			}
		})

		phase4Errored := false
		for _, moduleCtx := range modules {
			if moduleCtx.getCompilationState() != moduleCompilationStateCompiled {
				phase4Errored = true
				break
			}
		}

		// Collect diagnostics from all modules. dropBIRDiags drops BIR-gen diagnostics if a
		// sibling failed Phase 4, since they'd only be noise from an unreliable context.
		c.collectModuleDiagnostics(&allDiagnostics, phase4Errored)
	}

	c.diagnosticResult = NewDiagnosticResult(allDiagnostics)
}

// runPerModule runs fn once per module, in parallel when concurrent is true. A panic in
// one module doesn't stop the others; the first one is re-raised after all finish.
func runPerModule(modules []*moduleContext, concurrent bool, fn func(*moduleContext)) {
	var wg sync.WaitGroup
	var panicsMu sync.Mutex
	var panics []any
	run := func(m *moduleContext) {
		defer func() {
			if r := recover(); r != nil {
				panicsMu.Lock()
				panics = append(panics, r)
				panicsMu.Unlock()
			}
		}()
		fn(m)
	}
	for _, moduleCtx := range modules {
		if !concurrent {
			run(moduleCtx)
			continue
		}
		wg.Add(1)
		go func(m *moduleContext) {
			defer wg.Done()
			run(m)
		}(moduleCtx)
	}
	wg.Wait()
	if len(panics) > 0 {
		// TODO: report diagnostics for panics instead of crashing the process.
		panic(panics[0])
	}
}

func modulesHaveErrors(modules []*moduleContext) bool {
	for _, module := range modules {
		if module.compilerCtx != nil && module.compilerCtx.HasErrors() {
			return true
		}
	}
	return false
}

func (c *PackageCompilation) runCompilerPlugins(module *moduleContext) {
	if !runsCompilerPlugins(module) {
		return
	}
	plugins := c.compilerPluginManager.pluginsFor(module)
	if len(plugins) == 0 {
		return
	}
	span := module.compilerCtx.StartPackageSpan("Compiler Plugin", module.bLangPkg.PackageID)
	defer span.End()
	for _, resolved := range plugins {
		exported, ok := c.rootPackageContext.project.Environment().publicSymbols[resolved.provider.exported]
		if !ok {
			module.compilerCtx.InternalError(
				fmt.Sprintf("compiler plugin symbol space not found for %s/%s", resolved.provider.org, resolved.provider.pkg),
				resolved.position.position,
			)
			return
		}
		transformed, err := resolved.plugin.PackageTransformer(module.compilerCtx, exported, module.bLangPkg)
		if err != nil {
			module.compilerCtx.InternalError(
				fmt.Sprintf("compiler plugin %s/%s:%s failed: %v", resolved.provider.org, resolved.provider.pkg, resolved.declaration.function, err),
				resolved.position.position,
			)
			return
		}
		if transformed == nil {
			module.compilerCtx.InternalError(
				fmt.Sprintf("compiler plugin %s/%s:%s returned a nil package", resolved.provider.org, resolved.provider.pkg, resolved.declaration.function),
				resolved.position.position,
			)
			return
		}
		module.bLangPkg = transformed
		if module.compilerCtx.HasErrors() {
			return
		}
	}
}

// collectModuleDiagnostics appends per-module compilation diagnostics to dst, applying
// severity filtering for bala projects. dropBIRDiags, when true, truncates a module's
// diagnostics to those recorded before its (pipelined) BIR generation ran — see the call
// site in compileModulesInternal for why.
func (c *PackageCompilation) collectModuleDiagnostics(dst *[]diagnostics.Diagnostic, dropBIRDiags bool) {
	isBala := c.getPackageContext().getProject().Kind() == ProjectKindBala
	for _, moduleCtx := range c.packageResolution.topologicallySortedModuleList {
		diags := moduleCtx.getDiagnostics()
		if dropBIRDiags && moduleCtx.birGenAttempted {
			diags = diags[:moduleCtx.birGenDiagStart]
		}
		for _, diag := range diags {
			severity := diag.DiagnosticInfo().Severity()
			if isBala && severity != diagnostics.Error && severity != diagnostics.Fatal {
				continue
			}
			// TODO(P6): Determine isWorkspaceDep from dependency graph root comparison
			isWorkspaceDep := false
			*dst = append(*dst,
				newPackageDiagnostic(diag, moduleCtx.getDescriptor(), moduleCtx.getProject(), isWorkspaceDep))
		}
	}
}

// Resolution returns the package resolution.
// Java source: PackageCompilation.getResolution()
func (c *PackageCompilation) Resolution() *PackageResolution {
	return c.packageResolution
}

// DiagnosticResult returns the diagnostic result from compilation.
// Java source: PackageCompilation.diagnosticResult()
func (c *PackageCompilation) DiagnosticResult() DiagnosticResult {
	return c.diagnosticResult
}

// DiagnosticEnv returns the diagnostic env for resolving byte offsets to line/column.
func (c *PackageCompilation) DiagnosticEnv() *diagnostics.DiagnosticEnv {
	return c.compilerEnv.DiagnosticEnv()
}

// SemanticModel returns the semantic model for the specified module.
// TODO(P6): Implement when SemanticModel/BallerinaSemanticModel is migrated.
// Java source: PackageCompilation.getSemanticModel(ModuleId)
func (c *PackageCompilation) SemanticModel(moduleID ModuleID) any {
	// TODO(P6): Return *SemanticModel once the type is implemented.
	return nil
}

// CodeActionManager returns the code action manager.
// TODO(P6): Implement when CompilerPluginManager is migrated.
// Java source: PackageCompilation.getCodeActionManager()
func (c *PackageCompilation) CodeActionManager() any {
	// TODO(P6): Return CodeActionManager once the type is implemented.
	return nil
}

// CompletionManager returns the completion manager.
// TODO(P6): Implement when CompilerPluginManager is migrated.
// Java source: PackageCompilation.getCompletionManager()
func (c *PackageCompilation) CompletionManager() any {
	// TODO(P6): Return CompletionManager once the type is implemented.
	return nil
}

// getCompilationOptions returns the compilation options.
// Java source: PackageCompilation.compilationOptions()
func (c *PackageCompilation) getCompilationOptions() CompilationOptions {
	return c.compilationOptions
}

// getPackageContext returns the root package context.
// Java source: PackageCompilation.packageContext()
func (c *PackageCompilation) getPackageContext() *packageContext {
	return c.rootPackageContext
}

// getCompilerPluginManager returns the compiler plugin manager.
func (c *PackageCompilation) getCompilerPluginManager() *compilerPluginResolver {
	return c.compilerPluginManager
}

// getPluginDiagnostics returns the plugin diagnostics.
// Java source: PackageCompilation.pluginDiagnostics()
func (c *PackageCompilation) getPluginDiagnostics() []diagnostics.Diagnostic {
	return c.pluginDiagnostics
}

// notifyCompilationCompletion notifies compilation completion to lifecycle listeners.
// TODO(P6): Implement when CompilerLifecycleManager is migrated.
// Java source: PackageCompilation.notifyCompilationCompletion(Path, BalCommand)
func (c *PackageCompilation) notifyCompilationCompletion() []diagnostics.Diagnostic {
	// TODO(P6): Delegate to CompilerLifecycleManager.runCodeGeneratedTasks()
	return nil
}

// getCompilerBackend returns a compiler backend for the given target platform,
// creating one via the creator function if not already cached.
// Thread-safe: uses a mutex to match Java's ConcurrentHashMap.computeIfAbsent() semantics.
// TODO(P6): Implement when compiler backend integration is complete.
// Java source: PackageCompilation.getCompilerBackend(TargetPlatform, Function)
func (c *PackageCompilation) getCompilerBackend(platform TargetPlatform, creator func(TargetPlatform) CompilerBackend) CompilerBackend {
	c.backendMu.Lock()
	defer c.backendMu.Unlock()
	if backend, ok := c.compilerBackends[platform]; ok {
		return backend
	}
	backend := creator(platform)
	c.compilerBackends[platform] = backend
	return backend
}
