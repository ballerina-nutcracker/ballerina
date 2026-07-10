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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"ballerina/ast"
	"ballerina/bir"
	"ballerina/context"
	"ballerina/desugar"
	"ballerina/model"
	"ballerina/parser/tree"
	"ballerina/semantics"
	"ballerina/tools/diagnostics"
)

// moduleContext holds internal state for a Module.
// It manages document contexts for source and test documents.
type moduleContext struct {
	project                Project
	moduleID               ModuleID
	moduleDescriptor       ModuleDescriptor
	isDefaultModule        bool
	srcDocContextMap       map[DocumentID]*documentContext
	srcDocIDs              []DocumentID
	testDocContextMap      map[DocumentID]*documentContext
	testSrcDocIDs          []DocumentID
	moduleDescDependencies []ModuleDescriptor

	// Compilation state tracking.
	compilationState  moduleCompilationState
	stateInitialized  bool
	moduleDiagnostics []diagnostics.Diagnostic

	// Compilation artifacts.
	bLangPkg        *ast.BLangPackage
	bPackageSymbol  any // TODO(S3): BPackageSymbol once compiler symbol types are migrated
	compilerCtx     *context.CompilerContext
	importedSymbols map[string]model.ExportedSymbolSpace
	birPkg          *bir.BIRPackage
}

// newModuleContext creates a moduleContext from ModuleConfig.
func newModuleContext(project Project, moduleConfig ModuleConfig, disableSyntaxTree bool) *moduleContext {
	diagKeyPrefix := buildDiagKeyPrefix(project, moduleConfig.ModuleDescriptor())

	// Build source document context map
	srcDocContextMap := make(map[DocumentID]*documentContext)
	srcDocIDs := make([]DocumentID, 0, len(moduleConfig.SourceDocs()))
	for _, srcDocConfig := range moduleConfig.SourceDocs() {
		docID := srcDocConfig.DocumentID()
		srcDocContextMap[docID] = newDocumentContext(srcDocConfig, disableSyntaxTree, diagKeyPrefix)
		srcDocIDs = append(srcDocIDs, docID)
	}

	// Build test document context map
	testDocContextMap := make(map[DocumentID]*documentContext)
	testSrcDocIDs := make([]DocumentID, 0, len(moduleConfig.TestSourceDocs()))
	for _, testDocConfig := range moduleConfig.TestSourceDocs() {
		docID := testDocConfig.DocumentID()
		testDocContextMap[docID] = newDocumentContext(testDocConfig, disableSyntaxTree, diagKeyPrefix)
		testSrcDocIDs = append(testSrcDocIDs, docID)
	}

	// Copy dependencies
	depsCopy := slices.Clone(moduleConfig.Dependencies())

	env := project.Environment().compilerEnvironment()
	return &moduleContext{
		project:                project,
		moduleID:               moduleConfig.ModuleID(),
		moduleDescriptor:       moduleConfig.ModuleDescriptor(),
		isDefaultModule:        moduleConfig.IsDefaultModule(),
		srcDocContextMap:       srcDocContextMap,
		srcDocIDs:              srcDocIDs,
		testDocContextMap:      testDocContextMap,
		testSrcDocIDs:          testSrcDocIDs,
		moduleDescDependencies: depsCopy,
		compilerCtx:            context.NewCompilerContext(env),
	}
}

// buildDiagKeyPrefix returns the prefix prepended to a document name to form
// a globally-unique DiagnosticEnv key.
//
//   - bala dependency:    "<org>/<moduleName>/<version>::"
//   - workspace member:   "<member-dir>/[modules/<moduleNamePart>/]"
//   - build named module: "modules/<moduleNamePart>/"
//   - default module:     ""
func buildDiagKeyPrefix(project Project, md ModuleDescriptor) string {
	if project.Kind() == ProjectKindBala {
		return md.Org().Value() + "/" + md.Name().String() + "/" + md.Version().String() + "::"
	}

	prefix := ""
	// Normalize the source root to forward-slash form so DiagnosticEnv keys
	// stay consistent across OSes (project.SourceRoot() may use backslashes
	// on Windows).
	if root := filepath.ToSlash(project.SourceRoot()); root != "" && root != "." {
		// Workspace member (or any project loaded with a non-trivial source root):
		// scope all its files under the source-root directory.
		prefix = root + "/"
	}
	if !md.Name().IsDefaultModuleName() {
		prefix += ModulesDir + "/" + md.Name().ModuleNamePart() + "/"
	}
	return prefix
}

// newModuleContextFromMaps creates a moduleContext directly from document context maps.
// This is used for creating modified module contexts.
// The srcDocIDs and testSrcDocIDs parameters provide deterministic ordering.
func newModuleContextFromMaps(
	project Project,
	moduleID ModuleID,
	moduleDescriptor ModuleDescriptor,
	isDefaultModule bool,
	srcDocIDs []DocumentID,
	testSrcDocIDs []DocumentID,
	srcDocContextMap map[DocumentID]*documentContext,
	testDocContextMap map[DocumentID]*documentContext,
	moduleDescDependencies []ModuleDescriptor,
) *moduleContext {
	if srcDocContextMap == nil {
		srcDocContextMap = make(map[DocumentID]*documentContext)
	}
	if testDocContextMap == nil {
		testDocContextMap = make(map[DocumentID]*documentContext)
	}

	env := project.Environment().compilerEnvironment()
	return &moduleContext{
		project:                project,
		moduleID:               moduleID,
		moduleDescriptor:       moduleDescriptor,
		isDefaultModule:        isDefaultModule,
		srcDocContextMap:       srcDocContextMap,
		srcDocIDs:              srcDocIDs,
		testDocContextMap:      testDocContextMap,
		testSrcDocIDs:          testSrcDocIDs,
		moduleDescDependencies: slices.Clone(moduleDescDependencies),
		compilerCtx:            context.NewCompilerContext(env),
	}
}

// getModuleID returns the module identifier.
func (m *moduleContext) getModuleID() ModuleID {
	return m.moduleID
}

// getDescriptor returns the module descriptor.
func (m *moduleContext) getDescriptor() ModuleDescriptor {
	return m.moduleDescriptor
}

// getModuleName returns the module name.
func (m *moduleContext) getModuleName() ModuleName {
	return m.moduleDescriptor.Name()
}

// getSrcDocumentIDs returns a defensive copy of source document IDs.
func (m *moduleContext) getSrcDocumentIDs() []DocumentID {
	return slices.Clone(m.srcDocIDs)
}

// getTestSrcDocumentIDs returns a defensive copy of test document IDs.
func (m *moduleContext) getTestSrcDocumentIDs() []DocumentID {
	return slices.Clone(m.testSrcDocIDs)
}

// getDocumentContext returns the context for a document.
// It searches both source and test document maps.
func (m *moduleContext) getDocumentContext(documentID DocumentID) *documentContext {
	if ctx, ok := m.srcDocContextMap[documentID]; ok {
		return ctx
	}
	return m.testDocContextMap[documentID]
}

// getSrcDocContextMap returns the source document context map.
// This returns a shallow copy of the map.
func (m *moduleContext) getSrcDocContextMap() map[DocumentID]*documentContext {
	return maps.Clone(m.srcDocContextMap)
}

// getTestDocContextMap returns the test document context map.
// This returns a shallow copy of the map.
func (m *moduleContext) getTestDocContextMap() map[DocumentID]*documentContext {
	return maps.Clone(m.testDocContextMap)
}

// isDefault returns true if this is the default module.
func (m *moduleContext) isDefault() bool {
	return m.isDefaultModule
}

// getProject returns the project reference.
func (m *moduleContext) getProject() Project {
	return m.project
}

// getModuleDescDependencies returns a defensive copy of module descriptor dependencies.
func (m *moduleContext) getModuleDescDependencies() []ModuleDescriptor {
	return slices.Clone(m.moduleDescDependencies)
}

// resolveTypesAndSymbols performs parsing, AST building, symbol resolution, and type resolution.
// This phase must run sequentially respecting module dependencies.
func resolveTypesAndSymbols(moduleCtx *moduleContext) {
	moduleCtx.moduleDiagnostics = nil

	compilerCtx := moduleCtx.compilerCtx

	// Parse all source and test documents in parallel.
	compilerCtx.StartStage(context.StageParse)
	syntaxTrees := parseDocumentsParallel(
		moduleCtx.srcDocIDs,
		moduleCtx.srcDocContextMap,
		moduleCtx.testSrcDocIDs,
		moduleCtx.testDocContextMap,
	)
	compilerCtx.EndStage()

	if len(syntaxTrees) == 0 {
		return
	}

	// Build compilation units from syntax trees.
	compilerCtx.StartStage(context.StageASTBuild)
	compilationOptions := moduleCtx.project.BuildOptions().CompilationOptions()
	compilationUnits := buildCompilationUnits(compilerCtx, syntaxTrees, compilationOptions)

	if compilerCtx.HasDiagnostics() {
		return
	}

	pkgID := createModelPackageID(compilerCtx, moduleCtx.moduleDescriptor)
	for _, cu := range compilationUnits {
		cu.SetPackageID(pkgID)
	}
	compilerCtx.EndStage()

	// Resolve symbols (imports) before type resolution
	compilerCtx.StartStage(context.StageImportResolution)
	publicSymbols := moduleCtx.getProject().Environment().publicSymbols
	// PR-TODO: remove this after migration all lang libraries
	implicitImports := semantics.GetImplicitImports(compilerCtx)
	seedMigratedLangLibs(implicitImports, publicSymbols)
	importedSymbolsByCU := semantics.ResolveCompilationUnitImports(compilerCtx, compilationUnits, implicitImports, publicSymbols, moduleCtx.moduleDescriptor.Org().value)
	moduleCtx.importedSymbols = mergeCompilationUnitImports(importedSymbolsByCU)
	compilerCtx.EndStage()

	compilerCtx.StartStage(context.StageSymbolResolution)
	pkgScope, exported := semantics.ResolveSymbols(compilerCtx, *pkgID, importedSymbolsByCU)
	pkgNode := ast.ToPackageFromCompilationUnits(compilationUnits)
	pkgNode.Imports = nil
	pkgNode.PackageID = pkgID
	pkgNode.Scope = pkgScope
	moduleCtx.bLangPkg = pkgNode
	compilerCtx.EndStage()

	if compilerCtx.HasErrors() {
		// Do not publish a half-built symbol space; dependents would otherwise
		// see symbols with unresolved types and panic during type resolution.
		return
	}

	publicSymbols[semantics.PackageIdentifier{
		OrgName:    moduleCtx.moduleDescriptor.Org().value,
		ModuleName: moduleCtx.moduleID.moduleName,
	}] = exported

	// Add type resolution step (this only resolve types of top level nodes)
	compilerCtx.StartStage(context.StageTopLevelTypeResolution)
	semantics.ResolveTopLevelNodes(compilerCtx, pkgNode, moduleCtx.importedSymbols)
	compilerCtx.EndStage()
}

// analyzeAndDesugar performs CFG creation, semantic analysis, CFG analysis, and desugaring.
// This phase can run in parallel across modules after all modules complete Phase 1.
func analyzeAndDesugar(moduleCtx *moduleContext) {
	if moduleCtx.bLangPkg == nil || moduleCtx.compilerCtx == nil {
		return
	}

	pkgNode := moduleCtx.bLangPkg
	compilerCtx := moduleCtx.compilerCtx
	compilationOptions := moduleCtx.project.BuildOptions().CompilationOptions()

	if compilerCtx.HasDiagnostics() {
		return
	}

	// Resolve types of function bodies and inner nodes
	compilerCtx.StartStage(context.StageLocalNodeResolution)
	semantics.ResolveLocalNodes(compilerCtx, pkgNode, moduleCtx.importedSymbols)
	compilerCtx.EndStage()
	if compilerCtx.HasDiagnostics() {
		return
	}

	compilerCtx.StartStage(context.StageSemanticAnalysis)
	semanticAnalyzer := semantics.NewSemanticAnalyzer(moduleCtx.compilerCtx)
	semanticAnalyzer.Analyze(pkgNode, moduleCtx.importedSymbols)
	compilerCtx.EndStage()
	if compilerCtx.HasDiagnostics() {
		return
	}

	// Create control flow graph after semantic analysis.
	compilerCtx.StartStage(context.StageCFGCreation)
	cfg := semantics.CreateControlFlowGraph(compilerCtx, pkgNode)
	compilerCtx.EndStage()
	if compilerCtx.HasDiagnostics() {
		return
	}

	// Dump CFG if requested
	if compilationOptions.DumpCFG() {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "==================BEGIN CFG==================")
		if compilationOptions.DumpCFGFormat() == CFGFormatDot {
			dotExporter := semantics.NewCFGDotExporter(compilerCtx)
			fmt.Fprintln(os.Stderr, strings.TrimSpace(dotExporter.Export(cfg)))
		} else {
			prettyPrinter := semantics.NewCFGPrettyPrinter(compilerCtx)
			fmt.Fprintln(os.Stderr, strings.TrimSpace(prettyPrinter.Print(cfg)))
		}
		fmt.Fprintln(os.Stderr, "===================END CFG===================")
	}

	if compilerCtx.HasDiagnostics() {
		return
	}

	// Run CFG analyses (reachability and explicit return) after semantic analysis.
	compilerCtx.StartStage(context.StageCFGAnalysis)
	semantics.AnalyzeCFG(moduleCtx.compilerCtx, pkgNode, cfg)
	compilerCtx.EndStage()
	if compilerCtx.HasDiagnostics() {
		return
	}

	// Desugar package "lowering" AST to an AST that BIR gen can handle.
	compilerCtx.StartStage(context.StageDesugaring)
	moduleCtx.bLangPkg = desugar.DesugarPackage(moduleCtx.compilerCtx, moduleCtx.bLangPkg, moduleCtx.importedSymbols)
	compilerCtx.EndStage()

	moduleCtx.compilationState = moduleCompilationStateCompiled
}

// parseDocumentsParallel parses source and test documents in parallel.
// Returns syntax trees from source documents only (test docs are parsed but not returned).
func parseDocumentsParallel(
	srcDocIDs []DocumentID,
	srcDocContextMap map[DocumentID]*documentContext,
	testDocIDs []DocumentID,
	testDocContextMap map[DocumentID]*documentContext,
) []*tree.SyntaxTree {
	var (
		mu          sync.Mutex
		wg          sync.WaitGroup
		syntaxTrees []*tree.SyntaxTree
	)

	// Parse source documents - collect syntax trees
	for _, docID := range srcDocIDs {
		docCtx := srcDocContextMap[docID]
		if docCtx == nil {
			continue
		}

		wg.Add(1)
		go func(dc *documentContext) {
			defer wg.Done()
			st := dc.parse()
			if st != nil {
				mu.Lock()
				syntaxTrees = append(syntaxTrees, st)
				mu.Unlock()
			}
		}(docCtx)
	}

	// Parse test documents - no syntax trees collected
	for _, docID := range testDocIDs {
		docCtx := testDocContextMap[docID]
		if docCtx == nil {
			continue
		}

		wg.Add(1)
		go func(dc *documentContext) {
			defer wg.Done()
			dc.parse()
		}(docCtx)
	}

	wg.Wait()
	return syntaxTrees
}

func buildCompilationUnits(cx *context.CompilerContext, syntaxTrees []*tree.SyntaxTree, compilationOptions CompilationOptions) []*ast.BLangCompilationUnit {
	dumpAST := compilationOptions.DumpAST()
	var prettyPrinter ast.PrettyPrinter
	if dumpAST {
		prettyPrinter = ast.PrettyPrinter{}
	}

	compilationUnits := make([]*ast.BLangCompilationUnit, 0, len(syntaxTrees))
	for _, st := range syntaxTrees {
		cu := ast.GetCompilationUnit(cx, st)
		if dumpAST {
			fmt.Fprintln(os.Stderr, prettyPrinter.Print(cu))
		}
		compilationUnits = append(compilationUnits, cu)
	}
	return compilationUnits
}

func mergeCompilationUnitImports(imports []semantics.CompilationUnitImports) map[string]model.ExportedSymbolSpace {
	result := make(map[string]model.ExportedSymbolSpace)
	for _, cuImports := range imports {
		maps.Copy(result, cuImports.Imports)
	}
	return result
}

// seedMigratedLangLibs adds the migrated lang libraries (compiled as real
// packages and published to publicSymbols) to the implicit-imports map under
// their langlib key, so they are usable without an import statement. No-op
// until the lib has been compiled (e.g. while compiling the lib itself).
func seedMigratedLangLibs(implicitImports map[string]model.ExportedSymbolSpace, publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace) {
	for _, name := range []string{"lang.int", "lang.boolean", "lang.decimal", "lang.error", "lang.string", "lang.value", "lang.xml", "lang.float", "lang.array", "lang.map", "lang.object"} {
		if space, ok := publicSymbols[semantics.PackageIdentifier{OrgName: "ballerina", ModuleName: name}]; ok {
			implicitImports[name] = space
		}
	}
}

// createModelPackageID builds a model.PackageID from the module descriptor so BIR gen
// produces module-qualified function lookup keys that match cross-module call sites.
func createModelPackageID(compilerCtx *context.CompilerContext, desc ModuleDescriptor) *model.PackageID {
	orgName := model.Name(desc.Org().Value())
	moduleName := desc.Name().String()
	nameComps := make([]model.Name, 0)
	for _, part := range strings.Split(moduleName, ".") {
		nameComps = append(nameComps, model.Name(part))
	}
	version := model.Name(desc.Version().String())
	if version == "" {
		version = model.DEFAULT_VERSION
	}
	return compilerCtx.NewPackageID(orgName, nameComps, version)
}

// generateCodeInternal generates BIR for this module from the compiled BLangPackage.
// -> CompilerPhaseRunner.performBirGenPhases(bLangPackage)
func generateCodeInternal(moduleCtx *moduleContext) {
	if moduleCtx.bLangPkg == nil || moduleCtx.compilerCtx == nil {
		return
	}
	moduleCtx.compilerCtx.StartStage(context.StageBIRGeneration)
	moduleCtx.birPkg = bir.GenBir(moduleCtx.compilerCtx, moduleCtx.bLangPkg)
	moduleCtx.compilerCtx.EndStage()
}

// getBLangPackage returns the compiled BLangPackage.
func (m *moduleContext) getBLangPackage() *ast.BLangPackage {
	return m.bLangPkg
}

// getBIRPackage returns the generated BIR package.
func (m *moduleContext) getBIRPackage() *bir.BIRPackage {
	return m.birPkg
}

// getCompilationState returns the current compilation state of the module.
func (m *moduleContext) getCompilationState() moduleCompilationState {
	if !m.stateInitialized {
		// TODO: Check compilationCache.getBir() for pre-compiled BIR
		m.compilationState = moduleCompilationStateLoadedFromSources
		m.stateInitialized = true
	}
	return m.compilationState
}

// setCompilationState sets the compilation state of the module.
func (m *moduleContext) setCompilationState(state moduleCompilationState) {
	m.compilationState = state
	m.stateInitialized = true
}

// getDiagnostics returns the diagnostics produced during module compilation.
func (m *moduleContext) getDiagnostics() []diagnostics.Diagnostic {
	return m.compilerCtx.Diagnostics()
}

// duplicate creates a copy of the context.
// The duplicated context has all document contexts duplicated as well.
func (m *moduleContext) duplicate(project Project) *moduleContext {
	// Duplicate source document contexts
	srcDocContextMap := make(map[DocumentID]*documentContext, len(m.srcDocIDs))
	for _, docID := range m.srcDocIDs {
		if docCtx := m.srcDocContextMap[docID]; docCtx != nil {
			srcDocContextMap[docID] = docCtx.duplicate()
		}
	}

	// Duplicate test document contexts
	testDocContextMap := make(map[DocumentID]*documentContext, len(m.testSrcDocIDs))
	for _, docID := range m.testSrcDocIDs {
		if docCtx := m.testDocContextMap[docID]; docCtx != nil {
			testDocContextMap[docID] = docCtx.duplicate()
		}
	}

	return newModuleContextFromMaps(
		project,
		m.moduleID,
		m.moduleDescriptor,
		m.isDefaultModule,
		slices.Clone(m.srcDocIDs),
		slices.Clone(m.testSrcDocIDs),
		srcDocContextMap,
		testDocContextMap,
		m.moduleDescDependencies,
	)
}

// containsDocument checks if the module contains the given document ID.
func (m *moduleContext) containsDocument(documentID DocumentID) bool {
	_, inSrc := m.srcDocContextMap[documentID]
	_, inTest := m.testDocContextMap[documentID]
	return inSrc || inTest
}

// isTestDocument returns true if the given document ID is a test document.
func (m *moduleContext) isTestDocument(documentID DocumentID) bool {
	_, ok := m.testDocContextMap[documentID]
	return ok
}

func (m *moduleContext) populateModuleLoadRequests() []*moduleLoadRequest {
	var requests []*moduleLoadRequest
	for _, docID := range m.srcDocIDs {
		docCtx := m.srcDocContextMap[docID]
		if docCtx != nil {
			requests = append(requests, docCtx.moduleLoadRequests()...)
		}
	}
	return requests
}

func (m *moduleContext) populateTestModuleLoadRequests() []*moduleLoadRequest {
	var requests []*moduleLoadRequest
	for _, docID := range m.testSrcDocIDs {
		docCtx := m.testDocContextMap[docID]
		if docCtx != nil {
			requests = append(requests, docCtx.moduleLoadRequests()...)
		}
	}
	return requests
}
