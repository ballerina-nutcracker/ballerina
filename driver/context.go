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

package driver

import (
	stdcontext "context"
	"sort"
	"sync"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// PackageDescriptor identifies a package by its organization, name and version.
type PackageDescriptor struct {
	Org     string
	Name    string
	Version string
}

// ModuleDescriptor identifies a module by its owning package and module name.
type ModuleDescriptor struct {
	Package PackageDescriptor
	Name    string
}

// Env holds the compiler environment and the symbol spaces published by
// already-compiled modules. It is shared by every module compiled through a
// single Context and is guarded by its own mutex, so it is safe for concurrent
// module compilation.
type Env struct {
	mu               sync.Mutex
	compiler         *compilercontext.CompilerEnvironment
	bound            bool
	publishedSymbols map[model.PackageIdentifier]model.ExportedSymbolSpace
	implicitSymbols  map[string]model.ExportedSymbolSpace
}

// NewEnv returns an Env wrapping compilerEnv with empty symbol tables.
// It panics if compilerEnv is nil.
func NewEnv(compilerEnv *compilercontext.CompilerEnvironment) *Env {
	if compilerEnv == nil {
		panic("driver: nil compiler environment")
	}
	return &Env{
		compiler:         compilerEnv,
		publishedSymbols: make(map[model.PackageIdentifier]model.ExportedSymbolSpace),
		implicitSymbols:  make(map[string]model.ExportedSymbolSpace),
	}
}

type diagnosticScope uint8

const (
	scopeRoot diagnosticScope = iota
	scopePackage
	scopeModule
)

type diagnosticEntry struct {
	diagnostic diagnostics.Diagnostic
	scope      diagnosticScope
	pkg        PackageDescriptor
	module     ModuleDescriptor
	stage      compilercontext.CompilationStage
	test       bool
	document   int
	ordinal    uint64
}

// Context is the per-compilation driver state: it carries the cancellation
// context, the shared Env, and the accumulated diagnostics and per-stage
// timings for every module. All of its mutable state is guarded by a mutex,
// so its methods may be called from concurrently compiling modules.
type Context struct {
	ctx stdcontext.Context
	env *Env

	mu               sync.Mutex
	diagnostics      []diagnosticEntry
	nextOrdinal      uint64
	discoveryIndex   map[ModuleDescriptor]int
	nextDiscovery    int
	topologicalIndex map[ModuleDescriptor]int
	stats            map[ModuleDescriptor]map[compilercontext.CompilationStage]compilercontext.StageTiming
	root             PackageDescriptor
	newContextHook   func(*compilercontext.CompilerContext)
}

// setRoot records descriptor as the root package of this compilation,
// acquiring the context lock.
func (c *Context) setRoot(descriptor PackageDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.root = descriptor
}

// isRoot reports whether descriptor is the root package recorded by setRoot,
// acquiring the context lock.
func (c *Context) isRoot(descriptor PackageDescriptor) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.root == descriptor
}

// NewContext binds env to ctx and returns the resulting compilation Context.
// An Env may be bound only once; NewContext panics if either argument is nil
// or if env is already bound to another Context.
func NewContext(ctx stdcontext.Context, env *Env) *Context {
	if ctx == nil || env == nil {
		panic("driver: nil context or environment")
	}
	env.mu.Lock()
	defer env.mu.Unlock()
	if env.bound {
		panic("driver: environment is already bound to a context")
	}
	env.bound = true
	return &Context{
		ctx:              ctx,
		env:              env,
		discoveryIndex:   make(map[ModuleDescriptor]int),
		topologicalIndex: make(map[ModuleDescriptor]int),
		stats:            make(map[ModuleDescriptor]map[compilercontext.CompilationStage]compilercontext.StageTiming),
	}
}

// Err reports the cancellation state of the underlying standard context;
// a non-nil result means the pipeline stages should stop early.
func (c *Context) Err() error { return c.ctx.Err() }

// DiagnosticEnv returns the diagnostic environment of the underlying compiler
// environment, used to render the diagnostics collected by this context.
func (c *Context) DiagnosticEnv() *diagnostics.DiagnosticEnv {
	return c.env.compiler.DiagnosticEnv()
}

// ExportedSymbols returns a snapshot of the symbol spaces published by modules
// that completed public-node resolution.
func (c *Context) ExportedSymbols() map[model.PackageIdentifier]model.ExportedSymbolSpace {
	c.env.mu.Lock()
	defer c.env.mu.Unlock()
	result := make(map[model.PackageIdentifier]model.ExportedSymbolSpace, len(c.env.publishedSymbols))
	for identifier, symbols := range c.env.publishedSymbols {
		result[identifier] = symbols
	}
	return result
}

// Diagnostics returns every diagnostic collected so far, ordered by module
// position in the compilation order and, within a module, by the order in
// which the stages reported them.
func (c *Context) Diagnostics() []diagnostics.Diagnostic {
	entries := c.sortedDiagnosticEntries()
	result := make([]diagnostics.Diagnostic, len(entries))
	for i, entry := range entries {
		result[i] = entry.diagnostic
	}
	return result
}

// HasDiagnostics reports whether any diagnostic of any severity has been
// collected, without sorting or copying them.
func (c *Context) HasDiagnostics() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.diagnostics) > 0
}

// HasErrors reports whether any collected diagnostic has error or fatal
// severity.
func (c *Context) HasErrors() bool {
	return hasErrors(c.Diagnostics())
}

// ModuleDiagnostics returns the diagnostics attributed to module, in stage
// order, excluding package- and root-scoped diagnostics.
func (c *Context) ModuleDiagnostics(module ModuleDescriptor) []diagnostics.Diagnostic {
	entries := c.sortedDiagnosticEntries()
	result := make([]diagnostics.Diagnostic, 0)
	for _, entry := range entries {
		if entry.scope == scopeModule && entry.module == module {
			result = append(result, entry.diagnostic)
		}
	}
	return result
}

// ModuleHasErrors reports whether module produced a diagnostic with error or
// fatal severity.
func (c *Context) ModuleHasErrors(module ModuleDescriptor) bool {
	return hasErrors(c.ModuleDiagnostics(module))
}

// hasErrors reports whether values contains a diagnostic with error or fatal
// severity.
func hasErrors(values []diagnostics.Diagnostic) bool {
	for _, diagnostic := range values {
		switch diagnostic.DiagnosticInfo().Severity() {
		case diagnostics.Error, diagnostics.Fatal:
			return true
		}
	}
	return false
}

// ModuleStats returns the accumulated per-stage timings of every module that
// reported any, ordered by compilation order and with each module's stages in
// canonical pipeline order. It acquires the context lock.
func (c *Context) ModuleStats() []*compilercontext.ModuleStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	modules := make([]ModuleDescriptor, 0, len(c.stats))
	for module := range c.stats {
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool { return c.moduleIndexLocked(modules[i]) < c.moduleIndexLocked(modules[j]) })
	result := make([]*compilercontext.ModuleStats, 0, len(modules))
	for _, module := range modules {
		byStage := c.stats[module]
		stages := make([]compilercontext.StageTiming, 0, len(byStage))
		for _, stage := range pipelineStages {
			if timing, ok := byStage[stage]; ok {
				stages = append(stages, timing)
			}
		}
		result = append(result, &compilercontext.ModuleStats{ModuleName: module.Name, Stages: stages})
	}
	return result
}

var pipelineStages = []compilercontext.CompilationStage{
	compilercontext.StageParse,
	compilercontext.StageASTBuild,
	compilercontext.StageImportResolution,
	compilercontext.StageSymbolResolution,
	compilercontext.StageTopLevelTypeResolution,
	compilercontext.StageLocalNodeResolution,
	compilercontext.StageSemanticAnalysis,
	compilercontext.StageCFGCreation,
	compilercontext.StageCFGAnalysis,
	compilercontext.StageDesugaring,
	compilercontext.StageBIRGeneration,
}

// newCompilerContext creates a fresh compiler context for one stage of module,
// initialising its stats collector and applying the test hook if one is set.
func (c *Context) newCompilerContext(module ModuleDescriptor) *compilercontext.CompilerContext {
	cx := compilercontext.NewCompilerContext(c.env.compiler)
	cx.InitModuleStats(module.Name)
	if c.newContextHook != nil {
		c.newContextHook(cx)
	}
	return cx
}

// drainModule moves all diagnostics and stage timings from cx into c,
// attributing them to module and stage. Use it when cx was created for this
// stage alone; for a compiler context shared by several stages, use
// drainModuleSince so already-drained entries are not recorded twice.
func (c *Context) drainModule(cx *compilercontext.CompilerContext, module ModuleDescriptor,
	stage compilercontext.CompilationStage, test bool, document int,
) {
	c.drainModuleSince(cx, module, stage, test, document, 0, 0)
}

// drainModuleSince moves the diagnostics and stage timings recorded in cx at or
// after the cursor positions diagnosticStart and statsStart into c, attributing
// them to module and stage, and accumulating durations per stage. It returns the
// new cursor positions, which the caller passes to the next drainModuleSince
// call on the same compiler context. It acquires the context lock.
func (c *Context) drainModuleSince(cx *compilercontext.CompilerContext, module ModuleDescriptor,
	stage compilercontext.CompilationStage, test bool, document, diagnosticStart, statsStart int,
) (int, int) {
	allDiagnostics := cx.Diagnostics()
	stats := cx.GetModuleStats()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, diagnostic := range allDiagnostics[diagnosticStart:] {
		c.nextOrdinal++
		c.diagnostics = append(c.diagnostics, diagnosticEntry{
			diagnostic: diagnostic, scope: scopeModule, module: module, stage: stage,
			test: test, document: document, ordinal: c.nextOrdinal,
		})
	}
	if stats == nil {
		return len(allDiagnostics), 0
	}
	if c.stats[module] == nil {
		c.stats[module] = make(map[compilercontext.CompilationStage]compilercontext.StageTiming)
	}
	for _, timing := range stats.Stages[statsStart:] {
		current := c.stats[module][timing.Name]
		current.Name = timing.Name
		current.Duration += timing.Duration
		c.stats[module][timing.Name] = current
	}
	return len(allDiagnostics), len(stats.Stages)
}

// setDiscoveryModules assigns discovery-order indices to any of modules not
// already seen, preserving the index of ones that were. It acquires the
// context lock.
func (c *Context) setDiscoveryModules(modules []ModuleDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, module := range modules {
		if _, exists := c.discoveryIndex[module]; exists {
			continue
		}
		c.discoveryIndex[module] = c.nextDiscovery
		c.nextDiscovery++
	}
}

// setTopologicalModules records modules in topological order, overwriting any
// previously recorded ordering. It acquires the context lock.
func (c *Context) setTopologicalModules(modules []ModuleDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, module := range modules {
		c.topologicalIndex[module] = i
	}
}

// moduleIndexLocked returns the sort index of module, preferring its
// topological position over its discovery position and falling back to maxint
// for an unknown module. The caller must hold the context lock.
func (c *Context) moduleIndexLocked(module ModuleDescriptor) int {
	if index, ok := c.topologicalIndex[module]; ok {
		return index
	}
	if index, ok := c.discoveryIndex[module]; ok {
		return index
	}
	return int(^uint(0) >> 1)
}
