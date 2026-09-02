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
	"errors"
	"io/fs"
	"path"
	"sync"
	"sync/atomic"

	"github.com/ballerina-nutcracker/ballerina/bir"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// PipelineStage identifies the compilation stage associated with an error.
type PipelineStage string

const (
	// StageSourceDiscovery discovers the package or workspace member sources.
	StageSourceDiscovery PipelineStage = "source-discovery"
	// StageParse generates syntax trees.
	StageParse PipelineStage = "parse"
	// StageDependencyResolution resolves and loads imported packages.
	StageDependencyResolution PipelineStage = "dependency-resolution"
	// StageAST generates abstract syntax trees.
	StageAST PipelineStage = "ast"
	// StageSymbolResolution resolves module-level symbols.
	StageSymbolResolution PipelineStage = "symbol-resolution"
	// StageTopLevelResolution resolves top-level node types.
	StageTopLevelResolution PipelineStage = "top-level-resolution"
	// StageLocalResolution resolves function bodies and other local nodes.
	StageLocalResolution PipelineStage = "local-resolution"
	// StageSemanticAnalysis performs semantic analysis.
	StageSemanticAnalysis PipelineStage = "semantic-analysis"
	// StageCFGCreation generates control-flow graphs.
	StageCFGCreation PipelineStage = "cfg-creation"
	// StageCFGAnalysis analyzes control-flow graphs.
	StageCFGAnalysis PipelineStage = "cfg-analysis"
	// StageDesugaring desugars the resolved AST.
	StageDesugaring PipelineStage = "desugaring"
	// StageBIRGeneration generates BIR packages.
	StageBIRGeneration PipelineStage = "bir-generation"
	// StageRuntimeCreation constructs the runtime.
	StageRuntimeCreation PipelineStage = "runtime-creation"
)

// ParseEvent describes a parsed source document.
type ParseEvent struct {
	Root       PackageDescriptor
	Module     ModuleDescriptor
	SourcePath string
	Test       bool
	Document   *ParsedDocument
}

// ASTEvent describes an AST document generated from a source document.
type ASTEvent struct {
	Root       PackageDescriptor
	Module     ModuleDescriptor
	SourcePath string
	Recovered  bool
	Document   *ASTDocument
}

// CFGEvent describes the CFG generated for a module.
type CFGEvent struct {
	Root   PackageDescriptor
	Module ModuleDescriptor
	CFG    *semantics.PackageCFG
}

// BIREvent describes the BIR generated for a module.
type BIREvent struct {
	Root    PackageDescriptor
	Module  ModuleDescriptor
	Package *bir.BIRPackage
}

// ErrorEvent describes diagnostics or an operational error produced by a stage.
type ErrorEvent struct {
	Stage       PipelineStage
	Package     PackageDescriptor
	Module      ModuleDescriptor
	SourcePath  string
	Diagnostics []diagnostics.Diagnostic
	Err         error
}

// LifecycleHooks observes artifacts produced by the pipeline.
//
// Returning true from an artifact hook requests that the pipeline stop at its
// next safe checkpoint. Work already running in another module is allowed to
// reach a checkpoint. Module-level hooks and OnError may run concurrently, so
// implementations that share state must synchronize access to it. OnRuntime is
// called synchronously after compilation succeeds and before Run returns.
type LifecycleHooks struct {
	OnParse   func(ParseEvent) bool
	OnAST     func(ASTEvent) bool
	OnCFG     func(CFGEvent) bool
	OnBIR     func(BIREvent) bool
	OnError   func(ErrorEvent)
	OnRuntime func(*runtime.Runtime)
}

// Pipeline compiles a source file or project and constructs its runtime.
// A Pipeline is single-use.
type Pipeline struct {
	standardContext stdcontext.Context
	fsys            fs.FS
	inputPath       string
	packageDirName  string
	env             *Env
	resolver        DependencyResolver
	platform        pal.Platform
	parserDebug     parser.DebugOptions
	hooks           LifecycleHooks

	run          atomic.Bool
	stop         atomic.Bool
	diagnosticMu sync.Mutex
	reported     map[uint64]bool
	driverCtx    *Context
	root         PackageDescriptor
	sourceRoot   string
	parsed       *ParsedPackage
	birPackages  []*bir.BIRPackage
	err          error
}

// NewPipeline constructs a compilation pipeline for inputPath.
func NewPipeline(standardContext stdcontext.Context, fsys fs.FS, inputPath, packageDirName string,
	env *Env, resolver DependencyResolver, platform pal.Platform, parserDebug parser.DebugOptions,
	hooks LifecycleHooks,
) *Pipeline {
	return &Pipeline{
		standardContext: standardContext,
		fsys:            fsys,
		inputPath:       inputPath,
		packageDirName:  packageDirName,
		env:             env,
		resolver:        resolver,
		platform:        platform,
		parserDebug:     parserDebug,
		hooks:           hooks,
		birPackages:     make([]*bir.BIRPackage, 0),
		reported:        make(map[uint64]bool),
	}
}

// Run compiles the input and constructs an uninitialized runtime.
//
// The returned boolean is false when compilation fails or a hook requests a
// stop. Run does not register BIR packages with the runtime and does not call
// runtime lifecycle methods; callers own Init and Listen.
func (p *Pipeline) Run() (*runtime.Runtime, bool) {
	if !p.run.CompareAndSwap(false, true) {
		panic("driver: pipeline can only run once")
	}
	if p.standardContext == nil || p.fsys == nil || p.env == nil || p.resolver == nil {
		p.fail(StageSourceDiscovery, errors.New("driver: invalid nil pipeline argument"))
		return nil, false
	}
	p.driverCtx = NewContext(p.standardContext, p.env)

	diagnosticStart := p.diagnosticCount()
	sources, err := p.findSources()
	p.reportDiagnosticsSince(diagnosticStart, StageSourceDiscovery)
	if err != nil {
		p.fail(StageSourceDiscovery, err)
		return nil, false
	}
	if p.driverCtx.HasErrors() || p.stopped() {
		return nil, false
	}
	p.root = sources.Descriptor

	diagnosticStart = p.diagnosticCount()
	parsed, err := Parse(p.driverCtx, p.fsys, p.sourceRoot, sources, p.parserDebug)
	p.parsed = parsed
	p.reportDiagnosticsSince(diagnosticStart, StageParse)
	if err != nil {
		p.fail(StageParse, err)
		return nil, false
	}
	p.emitParse(parsed, nil)
	if p.driverCtx.HasErrors() {
		p.emitRecoveredAST(parsed)
		return nil, false
	}
	if p.stopped() {
		return nil, false
	}

	rootDocuments := parsedDocumentSet(parsed)
	diagnosticStart = p.diagnosticCount()
	parsed, err = ResolveDependencies(p.driverCtx, parsed, p.resolver)
	p.parsed = parsed
	p.reportDiagnosticsSince(diagnosticStart, StageDependencyResolution)
	if err != nil {
		p.fail(StageDependencyResolution, err)
		return nil, false
	}
	if parsed != nil {
		p.emitParse(parsed, rootDocuments)
	}
	if p.driverCtx.HasErrors() || p.driverCtx.Err() != nil || p.stopped() {
		if p.driverCtx.Err() != nil {
			p.fail(StageDependencyResolution, p.driverCtx.Err())
		}
		return nil, false
	}

	astModules := make([]*ASTModule, len(parsed.Modules))
	var astWG sync.WaitGroup
	for index, module := range parsed.Modules {
		if p.stopped() {
			break
		}
		astWG.Add(1)
		go func() {
			defer astWG.Done()
			start := p.diagnosticCount()
			astModules[index] = ToAST(p.driverCtx, module)
			p.reportDiagnosticsSince(start, StageAST)
			if astModules[index] != nil {
				p.emitAST(astModules[index], false)
			}
		}()
	}
	astWG.Wait()
	if p.driverCtx.HasErrors() || p.driverCtx.Err() != nil || p.stopped() {
		if p.driverCtx.Err() != nil {
			p.fail(StageAST, p.driverCtx.Err())
		}
		return nil, false
	}

	publicModules := make([]*PartiallyResolvedModule, len(astModules))
	for index, module := range astModules {
		if module == nil {
			p.fail(StageAST, errors.New("driver: AST conversion produced no module"))
			return nil, false
		}
		if p.stopped() {
			return nil, false
		}
		start := p.diagnosticCount()
		publicModules[index] = ResolvePublicNodes(p.driverCtx, module)
		p.reportDiagnosticsSince(start, StageSymbolResolution)
		if publicModules[index] == nil || p.driverCtx.HasErrors() || p.driverCtx.Err() != nil {
			if p.driverCtx.Err() != nil {
				p.fail(StageTopLevelResolution, p.driverCtx.Err())
			}
			return nil, false
		}
	}

	p.birPackages = make([]*bir.BIRPackage, len(publicModules))
	var stageWG sync.WaitGroup
	for index, module := range publicModules {
		if p.stopped() {
			break
		}
		stageWG.Add(1)
		go func() {
			defer stageWG.Done()
			if p.stopped() {
				return
			}
			start := p.diagnosticCount()
			resolved := ResolvePrivateNodes(p.driverCtx, module)
			p.reportDiagnosticsSince(start, StageLocalResolution)
			if resolved == nil || p.stopped() {
				return
			}
			start = p.diagnosticCount()
			analyzed := AnalyzeSemantics(p.driverCtx, resolved)
			p.reportDiagnosticsSince(start, StageSemanticAnalysis)
			if analyzed == nil || p.stopped() {
				return
			}
			start = p.diagnosticCount()
			controlFlow := CreateControlFlowGraph(p.driverCtx, analyzed)
			p.reportDiagnosticsSince(start, StageCFGCreation)
			if controlFlow == nil {
				return
			}
			if p.hooks.OnCFG != nil && p.hooks.OnCFG(CFGEvent{Root: p.root, Module: module.ID, CFG: controlFlow.CFG}) {
				p.stop.Store(true)
			}
			if p.stopped() {
				return
			}
			start = p.diagnosticCount()
			cfgAnalyzed := AnalyzeControlFlowGraph(p.driverCtx, controlFlow)
			p.reportDiagnosticsSince(start, StageCFGAnalysis)
			if cfgAnalyzed == nil || p.stopped() {
				return
			}
			start = p.diagnosticCount()
			desugared := Desugar(p.driverCtx, cfgAnalyzed)
			p.reportDiagnosticsSince(start, StageDesugaring)
			if desugared == nil || p.stopped() {
				return
			}
			start = p.diagnosticCount()
			p.birPackages[index] = GenerateBIR(p.driverCtx, desugared)
			p.reportDiagnosticsSince(start, StageBIRGeneration)
			if p.birPackages[index] != nil && p.hooks.OnBIR != nil && p.hooks.OnBIR(BIREvent{
				Root: p.root, Module: module.ID, Package: p.birPackages[index],
			}) {
				p.stop.Store(true)
			}
		}()
	}
	stageWG.Wait()
	if p.driverCtx.Err() != nil {
		p.fail(StageBIRGeneration, p.driverCtx.Err())
		return nil, false
	}
	if p.driverCtx.HasErrors() || p.stopped() {
		return nil, false
	}
	for _, pkg := range p.birPackages {
		if pkg == nil {
			p.fail(StageBIRGeneration, errors.New("driver: BIR generation produced no package"))
			return nil, false
		}
	}

	rt := runtime.NewRuntime(p.platform, p.env.compiler.GetTypeEnv())
	if p.hooks.OnRuntime != nil {
		p.hooks.OnRuntime(rt)
	}
	return rt, true
}

// Context returns the driver context after Run has started.
func (p *Pipeline) Context() *Context { return p.driverCtx }

// Root returns the descriptor of the package selected for compilation.
func (p *Pipeline) Root() PackageDescriptor { return p.root }

// ParsedPackage returns the resolved parsed package, when available.
func (p *Pipeline) ParsedPackage() *ParsedPackage { return p.parsed }

// BIRPackages returns generated packages in dependency order.
func (p *Pipeline) BIRPackages() []*bir.BIRPackage {
	return append([]*bir.BIRPackage(nil), p.birPackages...)
}

// Err returns the operational error that stopped the pipeline, if any.
// Compilation diagnostics are available through Context and OnError.
func (p *Pipeline) Err() error { return p.err }

func (p *Pipeline) packageRoot() string {
	if p.packageDirName != "" {
		return p.inputPath
	}
	root := path.Dir(p.inputPath)
	if root == "" {
		return "."
	}
	return root
}

func (p *Pipeline) findSources() (*PackageSources, error) {
	info, statErr := fs.Stat(p.fsys, p.inputPath)
	if statErr != nil {
		return nil, statErr
	}
	p.sourceRoot = p.packageRoot()
	if !info.IsDir() {
		p.sourceRoot = path.Dir(p.inputPath)
	}
	if !info.IsDir() || p.packageDirName == "" {
		return FindSources(p.driverCtx, p.fsys, p.inputPath, p.packageDirName)
	}
	workspace, err := FindWorkspaceSources(p.driverCtx, p.fsys, ".")
	if err != nil || workspace == nil || p.driverCtx.HasErrors() {
		if workspace == nil && err == nil && !p.driverCtx.HasErrors() {
			return FindSources(p.driverCtx, p.fsys, p.inputPath, p.packageDirName)
		}
		return nil, err
	}
	var member *WorkspaceMemberSources
	var ok bool
	if p.inputPath == "." && len(workspace.Members) > 0 {
		member, ok = workspace.Members[0], true
	} else {
		member, ok = workspace.Member(p.inputPath)
	}
	if !ok {
		return nil, errors.New("driver: input is not a workspace member")
	}
	p.sourceRoot = member.Root
	p.resolver = NewWorkspaceDependencyResolver(p.fsys, workspace, p.resolver, workspace.Resolution)
	return member.Sources, nil
}

func (p *Pipeline) stopped() bool { return p.stop.Load() }

func (p *Pipeline) fail(stage PipelineStage, err error) {
	if err == nil {
		return
	}
	if p.err == nil {
		p.err = err
	}
	if p.hooks.OnError != nil {
		p.hooks.OnError(ErrorEvent{Stage: stage, Package: p.root, Err: err})
	}
}

func (p *Pipeline) diagnosticCount() int {
	if p.driverCtx == nil {
		return 0
	}
	p.driverCtx.mu.Lock()
	defer p.driverCtx.mu.Unlock()
	return len(p.driverCtx.diagnostics)
}

func (p *Pipeline) reportDiagnosticsSince(start int, fallback PipelineStage) {
	if p.hooks.OnError == nil || p.driverCtx == nil {
		return
	}
	p.driverCtx.mu.Lock()
	entries := append([]diagnosticEntry(nil), p.driverCtx.diagnostics[start:]...)
	p.driverCtx.mu.Unlock()
	for _, entry := range entries {
		severity := entry.diagnostic.DiagnosticInfo().Severity()
		if severity != diagnostics.Error && severity != diagnostics.Fatal {
			continue
		}
		p.diagnosticMu.Lock()
		if p.reported[entry.ordinal] {
			p.diagnosticMu.Unlock()
			continue
		}
		p.reported[entry.ordinal] = true
		p.diagnosticMu.Unlock()
		stage := pipelineStage(entry.stage)
		if stage == "" {
			stage = fallback
		}
		pkg := entry.pkg
		if entry.scope == scopeModule {
			pkg = entry.module.Package
		}
		if pkg == (PackageDescriptor{}) {
			pkg = p.root
		}
		p.hooks.OnError(ErrorEvent{
			Stage: stage, Package: pkg, Module: entry.module,
			SourcePath: p.sourcePath(entry), Diagnostics: []diagnostics.Diagnostic{entry.diagnostic},
		})
	}
}

func (p *Pipeline) sourcePath(entry diagnosticEntry) string {
	if p.parsed == nil || entry.scope != scopeModule {
		return ""
	}
	for _, module := range p.parsed.Modules {
		if module.ID != entry.module {
			continue
		}
		documents := module.Documents
		if entry.test {
			documents = module.TestDocuments
		}
		if entry.document >= 0 && entry.document < len(documents) && documents[entry.document] != nil {
			return documents[entry.document].SourcePath
		}
	}
	return ""
}

func (p *Pipeline) emitParse(parsed *ParsedPackage, skip map[*ParsedDocument]struct{}) {
	if parsed == nil || p.hooks.OnParse == nil {
		return
	}
	for _, module := range parsed.Modules {
		for _, item := range []struct {
			documents []*ParsedDocument
			test      bool
		}{{module.Documents, false}, {module.TestDocuments, true}} {
			for _, document := range item.documents {
				if document == nil {
					continue
				}
				if _, found := skip[document]; found {
					continue
				}
				if p.hooks.OnParse(ParseEvent{Root: p.root, Module: module.ID, SourcePath: document.SourcePath,
					Test: item.test, Document: document}) {
					p.stop.Store(true)
				}
			}
		}
	}
}

func (p *Pipeline) emitRecoveredAST(parsed *ParsedPackage) {
	if parsed == nil || p.hooks.OnAST == nil {
		return
	}
	for _, module := range parsed.Modules {
		recovered := ToRecoveredAST(p.driverCtx, module)
		if recovered != nil {
			p.emitASTBase(recovered.moduleBase, true)
		}
	}
}

func (p *Pipeline) emitAST(module *ASTModule, recovered bool) {
	if module != nil {
		p.emitASTBase(module.moduleBase, recovered)
	}
}

func (p *Pipeline) emitASTBase(module *moduleBase, recovered bool) {
	if p.hooks.OnAST == nil {
		return
	}
	for _, document := range module.Documents {
		if document != nil && p.hooks.OnAST(ASTEvent{Root: p.root, Module: module.ID,
			SourcePath: document.SourcePath, Recovered: recovered, Document: document}) {
			p.stop.Store(true)
		}
	}
}

func parsedDocumentSet(parsed *ParsedPackage) map[*ParsedDocument]struct{} {
	result := make(map[*ParsedDocument]struct{})
	if parsed == nil {
		return result
	}
	for _, module := range parsed.Modules {
		for _, document := range module.Documents {
			result[document] = struct{}{}
		}
		for _, document := range module.TestDocuments {
			result[document] = struct{}{}
		}
	}
	return result
}

func pipelineStage(stage compilercontext.CompilationStage) PipelineStage {
	switch stage {
	case compilercontext.StageParse:
		return StageParse
	case compilercontext.StageASTBuild:
		return StageAST
	case compilercontext.StageImportResolution:
		return StageDependencyResolution
	case compilercontext.StageSymbolResolution:
		return StageSymbolResolution
	case compilercontext.StageTopLevelTypeResolution:
		return StageTopLevelResolution
	case compilercontext.StageLocalNodeResolution:
		return StageLocalResolution
	case compilercontext.StageSemanticAnalysis:
		return StageSemanticAnalysis
	case compilercontext.StageCFGCreation:
		return StageCFGCreation
	case compilercontext.StageCFGAnalysis:
		return StageCFGAnalysis
	case compilercontext.StageDesugaring:
		return StageDesugaring
	case compilercontext.StageBIRGeneration:
		return StageBIRGeneration
	default:
		return ""
	}
}
