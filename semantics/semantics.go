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

// ModuleVisibility records whether a module is exported, so cross-package
// imports of non-exported modules can be rejected while same-package
// imports are exempt.
type ModuleVisibility struct {
	PackageOrg  string
	PackageName string
	Exported    bool
}

// ResolveSymbols binds imports, resolves package symbols, and returns its scope,
// exported symbols, and the merged imported symbol spaces.
func ResolveSymbols(
	ctx *context.CompilerContext,
	pkgID model.PackageID,
	compilationUnits []*ast.BLangCompilationUnit,
	implicitImports map[string]model.ExportedSymbolSpace,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	moduleVisibility map[PackageIdentifier]ModuleVisibility,
	defaultOrg, currentPackageName string,
) (model.Scope, model.ExportedSymbolSpace, map[string]model.ExportedSymbolSpace) {
	internalPublicSymbols := make(map[symbols.PackageIdentifier]model.ExportedSymbolSpace, len(publicSymbols))
	for id, symbolSpace := range publicSymbols {
		internalPublicSymbols[symbols.PackageIdentifier{OrgName: id.OrgName, ModuleName: id.ModuleName}] = symbolSpace
	}
	internalModuleVisibility := make(map[symbols.PackageIdentifier]symbols.ModuleVisibility, len(moduleVisibility))
	for id, visibility := range moduleVisibility {
		internalModuleVisibility[symbols.PackageIdentifier{OrgName: id.OrgName, ModuleName: id.ModuleName}] = symbols.ModuleVisibility{
			PackageOrg:  visibility.PackageOrg,
			PackageName: visibility.PackageName,
			Exported:    visibility.Exported,
		}
	}
	return symbols.Resolve(ctx, pkgID, compilationUnits, implicitImports, internalPublicSymbols, internalModuleVisibility, defaultOrg, currentPackageName)
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

// PrintCFGDot renders graph in Graphviz DOT format.
func PrintCFGDot(ctx *context.CompilerContext, graph *PackageCFG) string {
	return cfg.NewCFGDotExporter(ctx).Export(graph.graph)
}

// PrettyPrintCFG renders graph as deterministic text.
func PrettyPrintCFG(ctx *context.CompilerContext, graph *PackageCFG) string {
	return cfg.NewCFGPrettyPrinter(ctx).Print(graph.graph)
}
