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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package compile

import (
	"path/filepath"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/nodebuilder"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// moduleStage records the last completed stage of a module's LS compilation.
type moduleStage int

const (
	stageUnstarted moduleStage = iota
	stageParsed
	stageSymbolResolved
	stageTopLevelTypeResolved
	stageLocalTypeResolved
	stageSemanticAnalyzed
	stageCFGBuilt
	stageCFGAnalyzed
)

// moduleResolutionInput carries cross-module symbol-resolution inputs.
type moduleResolutionInput struct {
	defaultOrg       string
	implicitImports  map[string]model.ExportedSymbolSpace
	publicSymbols    map[semantics.PackageIdentifier]model.ExportedSymbolSpace
	moduleVisibility map[semantics.PackageIdentifier]semantics.ModuleVisibility
}

// newModuleResolutionInput defaults nil maps to empty maps.
func newModuleResolutionInput(defaultOrg string, implicitImports map[string]model.ExportedSymbolSpace, publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace) moduleResolutionInput {
	if implicitImports == nil {
		implicitImports = make(map[string]model.ExportedSymbolSpace)
	}
	if publicSymbols == nil {
		publicSymbols = make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace)
	}
	return moduleResolutionInput{
		defaultOrg:       defaultOrg,
		implicitImports:  implicitImports,
		publicSymbols:    publicSymbols,
		moduleVisibility: make(map[semantics.PackageIdentifier]semantics.ModuleVisibility),
	}
}

// moduleDriver advances a module using a fresh compiler context over a shared
// environment. Completed stages are not rerun.
type moduleDriver struct {
	ctx       *context.CompilerContext
	stage     moduleStage
	openText  textProvider
	sources   map[projects.DocumentID]string
	sourceIDs map[string]projects.DocumentID
	stale     func() bool

	pkgID                        *model.PackageID
	units                        []*ast.BLangCompilationUnit
	symbolResolutionUsable       bool
	topLevelTypeResolutionUsable bool
	localTypeResolutionUsable    bool
	// typeResolvable permits type resolution after usable symbol resolution
	// or after all remaining symbol references have been bound.
	typeResolvable bool
	// hasBadNodes records use of the recovering node builder and blocks
	// semantic analysis and later stages.
	hasBadNodes bool
	pkgNode     *ast.BLangPackage
	imported    map[string]model.ExportedSymbolSpace
	exported    model.ExportedSymbolSpace
	cfg         *semantics.PackageCFG

	// parserDiagnosticCount captures diagnostics before symbol resolution.
	// When nonzero, publishableDiagnostics omits later-stage diagnostics.
	parserDiagnosticCount int
}

// publishableDiagnostics suppresses later-stage diagnostics when parsing
// already reported diagnostics.
func (d *moduleDriver) publishableDiagnostics() []diagnostics.Diagnostic {
	all := d.ctx.Diagnostics()
	if d.parserDiagnosticCount > 0 && d.parserDiagnosticCount < len(all) {
		return all[:d.parserDiagnosticCount]
	}
	return all
}

// newModuleDriver creates a fresh context over env. openText overrides loaded
// document text; stale is checked on entry to advanceTo. Both may be nil.
func newModuleDriver(env *context.CompilerEnvironment, openText textProvider, stale func() bool) *moduleDriver {
	return &moduleDriver{ctx: context.NewCompilerContext(env), openText: openText, stale: stale,
		sources: make(map[projects.DocumentID]string), sourceIDs: make(map[string]projects.DocumentID)}
}

// currentStage returns the stage this driver's module has reached.
func (d *moduleDriver) currentStage() moduleStage {
	return d.stage
}

// diagnosticContext returns the fresh per-generation CompilerContext this
// driver's diagnostics live on.
func (d *moduleDriver) diagnosticContext() *context.CompilerContext {
	return d.ctx
}

// advanceTo attempts target and its prerequisites unless superseded. Stage
// gates can stop progress early; completed stages are no-ops.
func (d *moduleDriver) advanceTo(target moduleStage, module *projects.Module, input moduleResolutionInput) {
	if d.stale != nil && d.stale() {
		return
	}
	switch target {
	case stageParsed:
		d.ensureParsed(module)
	case stageSymbolResolved:
		d.ensureSymbolResolved(module, input)
	case stageTopLevelTypeResolved:
		d.ensureTopLevelTypeResolved(module, input)
	case stageLocalTypeResolved:
		d.ensureLocalTypeResolved(module, input)
	case stageSemanticAnalyzed:
		d.ensureSemanticAnalyzed(module, input)
	case stageCFGBuilt:
		d.ensureCFGBuilt(module, input)
	case stageCFGAnalyzed:
		d.ensureCFGAnalyzed(module, input)
	}
}

// ensureParsed registers untracked source names and builds compilation units,
// capturing each document's parser input. Test documents are excluded.
func (d *moduleDriver) ensureParsed(module *projects.Module) {
	if d.stage >= stageParsed {
		return
	}

	env := d.ctx.DiagnosticEnv()
	docIDs := module.DocumentIDs()
	units := make([]*ast.BLangCompilationUnit, 0, len(docIDs))

	for _, docID := range docIDs {
		doc := module.Document(docID)
		if doc == nil {
			continue
		}
		name := moduleFileRegistrationKey(module, doc.Name())
		content := documentText(module, doc, d.openText)
		d.sources[docID] = content
		d.sourceIDs[name] = docID
		if !alreadyRegistered(env, name) {
			env.RegisterFile(name, doc.TextDocument())
			markRegistered(env, name)
		}
		syntaxTree, _ := parser.GetSyntaxTree(d.ctx, name, content)
		if syntaxTree == nil {
			continue
		}
		if syntaxTree.RootNode.HasDiagnostics() {
			units = append(units, nodebuilder.GetRecoveredCompilationUnit(d.ctx, syntaxTree))
			d.hasBadNodes = true
			continue
		}
		units = append(units, nodebuilder.GetCompilationUnit(d.ctx, syntaxTree))
	}

	if len(units) == 0 {
		return
	}

	pkgID := d.newPackageID(module.Descriptor())
	for _, cu := range units {
		cu.SetPackageID(pkgID)
	}
	d.pkgID = pkgID
	d.units = units
	d.stage = stageParsed
}

// ensureSymbolResolved binds imports and resolves the module's own symbols
// (compiler stage 3). Requires stageParsed; a no-op if already reached.
func (d *moduleDriver) ensureSymbolResolved(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageSymbolResolved {
		return
	}
	d.ensureParsed(module)
	if d.stage < stageParsed {
		return
	}
	diagnosticCount := len(d.ctx.Diagnostics())
	d.parserDiagnosticCount = diagnosticCount
	pkgScope, exported, imported := semantics.ResolveSymbols(
		d.ctx,
		*d.pkgID,
		d.units,
		input.implicitImports,
		input.publicSymbols,
		input.moduleVisibility,
		input.defaultOrg,
		module.Descriptor().PackageName().Value(),
	)
	pkgNode := nodebuilder.ToPackageFromCompilationUnits(d.ctx, d.units)
	pkgNode.Imports = nil
	pkgNode.PackageID = d.pkgID
	pkgNode.Scope = pkgScope
	d.imported = imported
	d.exported = exported
	d.pkgNode = pkgNode
	d.symbolResolutionUsable = !d.hasNewErrorsSince(diagnosticCount)
	if !d.symbolResolutionUsable {
		bindUnresolvedReferences(d.ctx, *d.pkgID, pkgNode)
	}
	d.typeResolvable = d.symbolResolutionUsable || !hasUnresolvedSymbols(pkgNode)
	d.stage = stageSymbolResolved
}

// ensureTopLevelTypeResolved resolves top-level types when typeResolvable is
// true, even with prior diagnostics. Usability records only newly added errors.
func (d *moduleDriver) ensureTopLevelTypeResolved(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageTopLevelTypeResolved {
		return
	}
	d.ensureSymbolResolved(module, input)
	if d.stage < stageSymbolResolved || !d.typeResolvable {
		return
	}
	diagnosticCount := len(d.ctx.Diagnostics())
	semantics.ResolvePublicNodeTypes(d.ctx, d.pkgNode, d.imported)
	d.topLevelTypeResolutionUsable = !d.hasNewErrorsSince(diagnosticCount)
	d.stage = stageTopLevelTypeResolved
}

// hasNewErrorsSince checks for error or fatal diagnostics after diagnosticCount.
func (d *moduleDriver) hasNewErrorsSince(diagnosticCount int) bool {
	for _, diagnostic := range d.ctx.Diagnostics()[diagnosticCount:] {
		switch diagnostic.DiagnosticInfo().Severity() {
		case diagnostics.Error, diagnostics.Fatal:
			return true
		}
	}
	return false
}

// ensureLocalTypeResolved resolves types of function bodies and other inner
// nodes (compiler stage 5). Requires stageTopLevelTypeResolved.
func (d *moduleDriver) ensureLocalTypeResolved(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageLocalTypeResolved {
		return
	}
	d.ensureTopLevelTypeResolved(module, input)
	if d.stage < stageTopLevelTypeResolved {
		return
	}
	diagnosticCount := len(d.ctx.Diagnostics())
	semantics.ResolvePrivateNodesTypes(d.ctx, d.pkgNode, d.imported)
	d.localTypeResolutionUsable = !d.hasNewErrorsSince(diagnosticCount)
	d.stage = stageLocalTypeResolved
}

// ensureSemanticAnalyzed runs semantic analysis (compiler stage 6). Requires
// stageLocalTypeResolved. Never runs for a module with bad AST nodes or with
// any diagnostic from an earlier stage.
func (d *moduleDriver) ensureSemanticAnalyzed(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageSemanticAnalyzed {
		return
	}
	d.ensureLocalTypeResolved(module, input)
	if d.stage < stageLocalTypeResolved {
		return
	}
	if d.hasBadNodes || d.ctx.HasDiagnostics() {
		return
	}

	semantics.AnalyzeSemantics(d.ctx, d.pkgNode, d.imported)
	d.stage = stageSemanticAnalyzed
}

// ensureCFGBuilt builds the module's control-flow graph (compiler stage 7).
// Requires stageSemanticAnalyzed.
func (d *moduleDriver) ensureCFGBuilt(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageCFGBuilt {
		return
	}
	d.ensureSemanticAnalyzed(module, input)
	if d.stage < stageSemanticAnalyzed {
		return
	}
	if d.ctx.HasDiagnostics() {
		return
	}

	d.cfg = semantics.CreateControlFlowGraph(d.ctx, d.pkgNode)
	d.stage = stageCFGBuilt
}

// ensureCFGAnalyzed runs reachability and explicit-return analysis (compiler
// stage 8), the final LS stage. Requires stageCFGBuilt.
func (d *moduleDriver) ensureCFGAnalyzed(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageCFGAnalyzed {
		return
	}
	d.ensureCFGBuilt(module, input)
	if d.stage < stageCFGBuilt {
		return
	}
	if d.ctx.HasDiagnostics() {
		return
	}

	semantics.AnalyzeCFG(d.ctx, d.pkgNode, d.cfg)
	d.stage = stageCFGAnalyzed
}

// newPackageID interns the module descriptor's identity in the shared environment.
func (d *moduleDriver) newPackageID(desc projects.ModuleDescriptor) *model.PackageID {
	orgName := model.Name(desc.Org().Value())
	parts := strings.Split(desc.Name().String(), ".")
	nameComps := make([]model.Name, 0, len(parts))
	for _, part := range parts {
		nameComps = append(nameComps, model.Name(part))
	}
	version := model.Name(desc.Version().String())
	if version == "" {
		version = model.DEFAULT_VERSION
	}
	return d.ctx.NewPackageID(orgName, nameComps, version)
}

// moduleFileRegistrationKey prefixes workspace document names with the loaded
// source root and named-module path. BALA names use org/module/version::docName.
func moduleFileRegistrationKey(module *projects.Module, docName string) string {
	desc := module.Descriptor()
	project := module.Project()
	if project.Kind() == projects.ProjectKindBala {
		return desc.Org().Value() + "/" + desc.Name().String() + "/" + desc.Version().String() + "::" + docName
	}
	prefix := ""
	if root := filepath.ToSlash(project.SourceRoot()); root != "" && root != "." {
		prefix = root + "/"
	}
	if !desc.Name().IsDefaultModuleName() {
		prefix += projects.ModulesDir + "/" + desc.Name().ModuleNamePart() + "/"
	}
	return prefix + docName
}
