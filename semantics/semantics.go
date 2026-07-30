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

// Package semantics provides the public API for semantic compilation stages.
//
// ResolveSymbols, ResolvePublicNodeTypes, ResolvePrivateNodesTypes,
// AnalyzeSemantics, CreateControlFlowGraph, and AnalyzeCFG run in that order.
// Stages report diagnostics through context.CompilerContext and mutate package ASTs.
// Once symbol and public-node type resolution complete for dependencies, the
// remaining per-package stages may run concurrently.
package semantics

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/cfg"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/symbols"
	semantictypes "github.com/ballerina-nutcracker/ballerina/semantics/internal/types"
)

// PackageIdentifier identifies a Ballerina module while resolving imports.
type PackageIdentifier struct {
	OrgName    string
	ModuleName string
}

// CompilationUnitImports contains the imports resolved for one compilation unit.
type CompilationUnitImports struct {
	CompilationUnit *ast.BLangCompilationUnit
	Imports         map[string]model.ExportedSymbolSpace
}

// GetImplicitImports returns symbols for compiler-provided language-library imports.
// Deprecated: should be removed with https://github.com/ballerina-nutcracker/ballerina/issues/688.
func GetImplicitImports(ctx *context.CompilerContext) map[string]model.ExportedSymbolSpace {
	return symbols.GetImplicitImports(ctx)
}

// ResolveCompilationUnitImports resolves imports for each compilation unit.
// Deprecated: should be removed with https://github.com/ballerina-nutcracker/ballerina/issues/688.
func ResolveCompilationUnitImports(
	ctx *context.CompilerContext,
	compilationUnits []*ast.BLangCompilationUnit,
	implicitImports map[string]model.ExportedSymbolSpace,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	defaultOrg string,
) []CompilationUnitImports {
	internalPublicSymbols := make(map[symbols.PackageIdentifier]model.ExportedSymbolSpace, len(publicSymbols))
	for id, symbolSpace := range publicSymbols {
		internalPublicSymbols[symbols.PackageIdentifier{OrgName: id.OrgName, ModuleName: id.ModuleName}] = symbolSpace
	}
	resolved := symbols.ResolveCompilationUnitImports(ctx, compilationUnits, implicitImports, internalPublicSymbols, defaultOrg)
	result := make([]CompilationUnitImports, len(resolved))
	for i, imports := range resolved {
		result[i] = CompilationUnitImports{CompilationUnit: imports.CompilationUnit, Imports: imports.Imports}
	}
	return result
}

// ResolveSymbols resolves package symbols and returns its scope and exported symbols.
func ResolveSymbols(
	ctx *context.CompilerContext,
	pkgID model.PackageID,
	imports []CompilationUnitImports,
) (model.Scope, model.ExportedSymbolSpace) {
	internalImports := make([]symbols.CompilationUnitImports, len(imports))
	for i, unitImports := range imports {
		internalImports[i] = symbols.CompilationUnitImports{
			CompilationUnit: unitImports.CompilationUnit,
			Imports:         unitImports.Imports,
		}
	}
	return symbols.Resolve(ctx, pkgID, internalImports)
}

// ResolvePublicNodeTypes resolves the types exposed by a package.
func ResolvePublicNodeTypes(
	ctx *context.CompilerContext,
	pkg *ast.BLangPackage,
	importedSymbols map[string]model.ExportedSymbolSpace,
) {
	semantictypes.ResolvePublicNodes(ctx, pkg, importedSymbols)
}

// ResolvePrivateNodesTypes resolves package-local nodes and function bodies.
func ResolvePrivateNodesTypes(
	ctx *context.CompilerContext,
	pkg *ast.BLangPackage,
	importedSymbols map[string]model.ExportedSymbolSpace,
) {
	semantictypes.ResolvePrivateNodes(ctx, pkg, importedSymbols)
}

// AnalyzeSemantics performs semantic analysis on a resolved package.
func AnalyzeSemantics(
	ctx *context.CompilerContext,
	pkg *ast.BLangPackage,
	importedSymbols map[string]model.ExportedSymbolSpace,
) {
	analysis.Analyze(ctx, pkg, importedSymbols)
}

// PackageCFG is the public handle for a package control-flow graph.
type PackageCFG struct {
	graph *cfg.PackageCFG
}

// CreateControlFlowGraph builds control-flow graphs for a package.
func CreateControlFlowGraph(ctx *context.CompilerContext, pkg *ast.BLangPackage) *PackageCFG {
	return &PackageCFG{graph: cfg.Build(ctx, pkg)}
}

// AnalyzeCFG performs reachability, return, and initialization analyses.
func AnalyzeCFG(ctx *context.CompilerContext, pkg *ast.BLangPackage, graph *PackageCFG) {
	cfg.Analyze(ctx, pkg, graph.graph)
}

// CFGDotExporter renders package control-flow graphs in Graphviz DOT format.
type CFGDotExporter struct {
	exporter *cfg.CFGDotExporter
}

// NewCFGDotExporter creates a DOT exporter.
func NewCFGDotExporter(ctx *context.CompilerContext) *CFGDotExporter {
	return &CFGDotExporter{exporter: cfg.NewCFGDotExporter(ctx)}
}

// Export renders graph in Graphviz DOT format.
func (e *CFGDotExporter) Export(graph *PackageCFG) string {
	return e.exporter.Export(graph.graph)
}

// CFGPrettyPrinter renders package control-flow graphs as deterministic text.
type CFGPrettyPrinter struct {
	printer *cfg.CFGPrettyPrinter
}

// NewCFGPrettyPrinter creates a CFG pretty printer.
func NewCFGPrettyPrinter(ctx *context.CompilerContext) *CFGPrettyPrinter {
	return &CFGPrettyPrinter{printer: cfg.NewCFGPrettyPrinter(ctx)}
}

// Print renders graph as deterministic text.
func (p *CFGPrettyPrinter) Print(graph *PackageCFG) string {
	return p.printer.Print(graph.graph)
}
