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
	"sync"

	"ballerina-lang-go/context"
	"ballerina-lang-go/tools/diagnostics"
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
	compilerPluginManager any // TODO(P6): CompilerPluginManager once plugin system is migrated
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

	// TODO(P6): CompilerPluginManager.from(compilation)
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

		// Phase 1: Parse, AST, symbol resolution, top-level type resolution.
		// Sequential because symbol/type resolution of a module needs its dependencies
		// to have published their public symbol spaces. We still run Phase 1 for every
		// module (even after some errored) so we collect all top-level diagnostics in
		// one shot, but a dependent of an errored module is skipped to avoid cascading
		// noise (its imports would not resolve).
		for _, moduleCtx := range c.packageResolution.topologicallySortedModuleList {
			moduleCtx.compilerCtx.InitModuleStats(moduleCtx.getModuleName().String())
			if moduleCtx.getCompilationState() != moduleCompilationStateLoadedFromSources {
				// TODO: Handle LOADED_FROM_CACHE state - load symbols from BIR
				continue
			}
			if dependencyErrored(moduleCtx) {
				erroredModules[moduleCtx.getModuleID()] = struct{}{}
				continue
			}
			resolveTypesAndSymbols(moduleCtx)
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
			c.collectModuleDiagnostics(&allDiagnostics)
			c.diagnosticResult = NewDiagnosticResult(allDiagnostics)
			return
		}

		// Phase 2: local node resolution, semantic analysis, CFG, desugar, BIR
		// (parallel - no cross-module dependencies).
		// Each goroutine has panic recovery to convert panics to diagnostics.
		var wg sync.WaitGroup
		var panicsMu sync.Mutex
		var panics []any
		for _, moduleCtx := range c.packageResolution.topologicallySortedModuleList {
			if moduleCtx.getCompilationState() != moduleCompilationStateLoadedFromSources {
				continue
			}
			wg.Add(1)
			go func(m *moduleContext) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicsMu.Lock()
						panics = append(panics, r)
						panicsMu.Unlock()
					}
				}()
				analyzeAndDesugar(m)
			}(moduleCtx)
		}
		wg.Wait()

		// Re-panic if any Phase 2 goroutine panicked.
		// This preserves the original behavior where semantic errors cause panics.
		if len(panics) > 0 {
			// TODO: report diagnostics for panics instead of crashing the process.
			panic(panics[0])
		}

		// Collect diagnostics from all modules
		c.collectModuleDiagnostics(&allDiagnostics)
	}

	// TODO(P6): Run plugin code analysis (runPluginCodeAnalysis)

	c.diagnosticResult = NewDiagnosticResult(allDiagnostics)
}

// collectModuleDiagnostics appends per-module compilation diagnostics to dst,
// applying severity filtering for bala projects.
func (c *PackageCompilation) collectModuleDiagnostics(dst *[]diagnostics.Diagnostic) {
	isBala := c.getPackageContext().getProject().Kind() == ProjectKindBala
	for _, moduleCtx := range c.packageResolution.topologicallySortedModuleList {
		for _, diag := range moduleCtx.getDiagnostics() {
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

// StatsReport returns a formatted compilation stats report with per-module breakdown.
// Returns empty string if stats were not enabled.
func (c *PackageCompilation) StatsReport() string {
	return formatStatsReport(collectModuleStats(c.packageResolution.topologicallySortedModuleList))
}

// StatsReportOneline returns a compact stats report showing only per-stage totals.
// Returns empty string if stats were not enabled.
func (c *PackageCompilation) StatsReportOneline() string {
	return formatStatsReportOneline(collectModuleStats(c.packageResolution.topologicallySortedModuleList))
}

func collectModuleStats(moduleList []*moduleContext) []*context.ModuleStats {
	var allStats []*context.ModuleStats
	for _, m := range moduleList {
		if s := m.compilerCtx.GetModuleStats(); s != nil {
			allStats = append(allStats, s)
		}
	}
	return allStats
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
// TODO(P6): Return CompilerPluginManager once the type is implemented.
// Java source: PackageCompilation.compilerPluginManager()
func (c *PackageCompilation) getCompilerPluginManager() any {
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
