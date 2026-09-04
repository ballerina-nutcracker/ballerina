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

package driver

import (
	stdcontext "context"
	"sort"
	"sync"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type PackageDescriptor struct {
	Org     string
	Name    string
	Version string
}

type ModuleDescriptor struct {
	Package PackageDescriptor
	Name    string
}

type Env struct {
	mu               sync.Mutex
	compiler         *compilercontext.CompilerEnvironment
	bound            bool
	publishedSymbols map[model.PackageIdentifier]model.ExportedSymbolSpace
	implicitSymbols  map[string]model.ExportedSymbolSpace
}

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

func (c *Context) setRoot(descriptor PackageDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.root = descriptor
}

func (c *Context) isRoot(descriptor PackageDescriptor) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.root == descriptor
}

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

func (c *Context) Err() error { return c.ctx.Err() }

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

func (c *Context) Diagnostics() []diagnostics.Diagnostic {
	entries := c.sortedDiagnosticEntries()
	result := make([]diagnostics.Diagnostic, len(entries))
	for i, entry := range entries {
		result[i] = entry.diagnostic
	}
	return result
}

func (c *Context) HasDiagnostics() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.diagnostics) > 0
}

func (c *Context) HasErrors() bool {
	return hasErrors(c.Diagnostics())
}

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

func (c *Context) ModuleHasErrors(module ModuleDescriptor) bool {
	return hasErrors(c.ModuleDiagnostics(module))
}

func hasErrors(values []diagnostics.Diagnostic) bool {
	for _, diagnostic := range values {
		switch diagnostic.DiagnosticInfo().Severity() {
		case diagnostics.Error, diagnostics.Fatal:
			return true
		}
	}
	return false
}

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

func (c *Context) newCompilerContext(module ModuleDescriptor) *compilercontext.CompilerContext {
	cx := compilercontext.NewCompilerContext(c.env.compiler)
	cx.InitModuleStats(module.Name)
	if c.newContextHook != nil {
		c.newContextHook(cx)
	}
	return cx
}

func (c *Context) drainModule(cx *compilercontext.CompilerContext, module ModuleDescriptor,
	stage compilercontext.CompilationStage, test bool, document int,
) {
	c.drainModuleSince(cx, module, stage, test, document, 0, 0)
}

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

func (c *Context) setTopologicalModules(modules []ModuleDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, module := range modules {
		c.topologicalIndex[module] = i
	}
}

func (c *Context) moduleIndexLocked(module ModuleDescriptor) int {
	if index, ok := c.topologicalIndex[module]; ok {
		return index
	}
	if index, ok := c.discoveryIndex[module]; ok {
		return index
	}
	return int(^uint(0) >> 1)
}
