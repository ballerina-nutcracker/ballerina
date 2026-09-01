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

// stage.go implements the LS-owned per-module staged compilation driver
// (ticket 37). It bypasses projects.PackageCompilation and calls the
// compiler's public per-stage API directly: parser.GetSyntaxTree,
// nodebuilder.GetCompilationUnit, then semantics.ResolveSymbols,
// ResolvePublicNodeTypes, ResolvePrivateNodesTypes, AnalyzeSemantics,
// CreateControlFlowGraph, AnalyzeCFG in that order (semantics/semantics.go's
// doc comment documents this order). moduleDriver drives one module through
// this ladder; packageDriver (multimodule.go) orchestrates it across a
// package's modules, respecting the Phase 1 -> Phase 2 barrier. See
// docs/adr/2026-08-28-ls-owned-staged-compilation-pipeline.md.
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
)

// moduleStage is the LS-owned stage ladder for a single module's staged
// compilation. It is a new type owned by this package, not
// context.CompilationStage (a stats-only marker with no gating) or
// projects.moduleCompilationState (near-dead: only two of its ten states are
// ever set) — both dead/near-dead in packages the LS no longer routes
// compilation through, per the ADR.
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

// moduleResolutionInput carries the cross-module inputs a single module's
// symbol resolution needs. In this phase (single-module case) these are
// always empty: no dependency modules are compiled through this driver yet,
// and projects.Environment's publicSymbols map has no public accessor —
// deliberately not added here; wiring a module's real dependencies through
// the Phase 1 -> Phase 2 barrier is phase 2 (multi-module) scope.
type moduleResolutionInput struct {
	defaultOrg      string
	implicitImports map[string]model.ExportedSymbolSpace
	publicSymbols   map[semantics.PackageIdentifier]model.ExportedSymbolSpace
}

// newModuleResolutionInput builds a moduleResolutionInput, defaulting nil
// maps to empty ones (map fields are always initialized per repo convention;
// semantics.ResolveSymbols also writes into implicitImports).
func newModuleResolutionInput(defaultOrg string, implicitImports map[string]model.ExportedSymbolSpace, publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace) moduleResolutionInput {
	if implicitImports == nil {
		implicitImports = make(map[string]model.ExportedSymbolSpace)
	}
	if publicSymbols == nil {
		publicSymbols = make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace)
	}
	return moduleResolutionInput{defaultOrg: defaultOrg, implicitImports: implicitImports, publicSymbols: publicSymbols}
}

// moduleDriver advances a single module through the LS stage ladder,
// bypassing projects.PackageCompilation. It calls the compiler's public
// per-stage API directly over a *context.CompilerContext constructed fresh at
// driver-construction time — one moduleDriver instance corresponds to one
// generation-advance, per the ADR's re-entrancy rule. The
// *context.CompilerEnvironment behind that context is supplied by the caller
// and is expected to be the same shared instance reused across generations;
// diagnostics live on this driver's own fresh context and never accumulate
// on the shared environment or leak into another driver instance.
//
// Every ensureX method re-checks its own prerequisite via d.stage (never a
// proxy condition, e.g. a nil-check standing in for the stage) before doing
// any work, and is a no-op once d.stage has already reached or passed its
// target. This is the idempotency guard the ADR requires, and specifically
// avoids ls-ref's runModuleLocalTypeResolution asymmetry bug
// (ls-ref/lsp/diagnostics.go:262-279, which checks module.Package == nil
// instead of the stage field like its sibling stage functions).
type moduleDriver struct {
	ctx   *context.CompilerContext
	stage moduleStage

	pkgID    *model.PackageID
	units    []*ast.BLangCompilationUnit
	pkgNode  *ast.BLangPackage
	imported map[string]model.ExportedSymbolSpace
	exported model.ExportedSymbolSpace
	cfg      *semantics.PackageCFG
}

// newModuleDriver constructs a driver for one generation-advance of a single
// module, over the shared compiler environment env.
func newModuleDriver(env *context.CompilerEnvironment) *moduleDriver {
	return &moduleDriver{ctx: context.NewCompilerContext(env)}
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

// advanceTo drives module forward through the ladder up to (and including)
// target only, stopping early if an earlier stage produced diagnostics that
// gate further progress (mirroring projects/module_context.go's per-stage
// gates). Requesting a stage the module has already reached, or passed, is a
// no-op.
func (d *moduleDriver) advanceTo(target moduleStage, module *projects.Module, input moduleResolutionInput) {
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

// ensureParsed registers the module's source documents with the shared
// DiagnosticEnv and drives them through parsing and AST build (compiler
// stages 1-2, folded into the LS's single Parsed rung — the stage ladder has
// no separate ASTBuilt rung). Test documents are out of scope for this
// phase. A name is only ever passed to DiagnosticEnv.RegisterFile once per
// env, tracked via alreadyRegistered/markRegistered (multimodule.go) — a
// later generation over an edited module skips the call entirely rather
// than re-registering under the same name, since RegisterFile panics on a
// same-name/different-content collision by design and nothing downstream
// reads DiagnosticEnv's stored content back (see registeredNames' doc
// comment in multimodule.go for why that's safe).
//
// The registration key is moduleFileRegistrationKey(module, doc.Name()), not
// Document.Name() alone (a bare filename). A bare filename collides across
// modules with same-basename files (e.g. two named modules each with a
// "types.bal") — a real risk once this driver spans more than one module.
// moduleFileRegistrationKey rebuilds the same globally-unique composition
// documentContext.registrationKey() uses internally (source root + module
// path segment + name), from public API only: this package cannot see
// documentContext or its diagKeyPrefix. projects.Project.DocumentPath was
// considered instead (it is public) but rejected — it is unreliable for a
// SingleFileProject, whose DocumentPath returns the document's bare name
// rather than a source-root-joined path (projects/single_file_project.go's
// documentPath field is set to just the file's base name at load time), so
// it cannot be used as a collision-free, URI-comparable key uniformly across
// project kinds.
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
		content := doc.TextDocument().String()
		if !alreadyRegistered(env, name) {
			env.RegisterFile(name, doc.TextDocument())
			markRegistered(env, name)
		}
		syntaxTree, _ := parser.GetSyntaxTree(d.ctx, name, content)
		if syntaxTree == nil {
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
	if d.ctx.HasDiagnostics() {
		return
	}

	pkgScope, exported, imported := semantics.ResolveSymbols(
		d.ctx,
		*d.pkgID,
		d.units,
		input.implicitImports,
		input.publicSymbols,
		input.defaultOrg,
	)
	d.imported = imported
	d.exported = exported

	pkgNode := nodebuilder.ToPackageFromCompilationUnits(d.units)
	pkgNode.Imports = nil
	pkgNode.PackageID = d.pkgID
	pkgNode.Scope = pkgScope
	d.pkgNode = pkgNode
	d.stage = stageSymbolResolved
}

// ensureTopLevelTypeResolved resolves the types of the module's top-level
// (public) nodes (compiler stage 4). Requires stageSymbolResolved.
func (d *moduleDriver) ensureTopLevelTypeResolved(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageTopLevelTypeResolved {
		return
	}
	d.ensureSymbolResolved(module, input)
	if d.stage < stageSymbolResolved {
		return
	}
	if d.ctx.HasErrors() {
		return
	}

	semantics.ResolvePublicNodeTypes(d.ctx, d.pkgNode, d.imported)
	d.stage = stageTopLevelTypeResolved
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
	if d.ctx.HasDiagnostics() {
		return
	}

	semantics.ResolvePrivateNodesTypes(d.ctx, d.pkgNode, d.imported)
	d.stage = stageLocalTypeResolved
}

// ensureSemanticAnalyzed runs semantic analysis (compiler stage 6). Requires
// stageLocalTypeResolved.
func (d *moduleDriver) ensureSemanticAnalyzed(module *projects.Module, input moduleResolutionInput) {
	if d.stage >= stageSemanticAnalyzed {
		return
	}
	d.ensureLocalTypeResolved(module, input)
	if d.stage < stageLocalTypeResolved {
		return
	}
	if d.ctx.HasDiagnostics() {
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

// ensureCFGAnalyzed runs reachability and explicit-return analysis over the
// control-flow graph (compiler stage 8). Requires stageCFGBuilt. No
// desugar/BIR stage exists in this ladder — the LS stops here, matching
// ls-ref.
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

// newPackageID builds a model.PackageID from the module descriptor, mirroring
// projects.createModelPackageID (unexported, package-internal there) using
// only projects' public ModuleDescriptor accessors.
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

// moduleFileRegistrationKey composes a globally-unique DiagnosticEnv
// registration key for a document, mirroring the shape of
// projects/module_context.go's private buildDiagKeyPrefix (bala dependency:
// "<org>/<name>/<version>::<docName>"; everything else: "<sourceRoot>/[
// modules/<modulePart>/]<docName>", with the modules/ segment only for a
// named module). For a build or single-file project this composition equals
// the document's absolute file path, so it doubles as the key Compile's
// callers already use (a request URI path) with no separate translation.
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
