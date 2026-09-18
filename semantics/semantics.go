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

// ResolveSymbols binds imports, resolves package symbols, and returns its scope,
// exported symbols, and the merged imported symbol spaces.
func ResolveSymbols(
	ctx *context.CompilerContext,
	pkgID model.PackageID,
	compilationUnits []*ast.BLangCompilationUnit,
	implicitImports map[string]model.ExportedSymbolSpace,
	publicSymbols map[PackageIdentifier]model.ExportedSymbolSpace,
	defaultOrg string,
) (model.Scope, model.ExportedSymbolSpace, map[string]model.ExportedSymbolSpace) {
	span := ctx.StartPackageSpan("Symbol Resolution", &pkgID)
	defer span.End()
	internalPublicSymbols := make(map[symbols.PackageIdentifier]model.ExportedSymbolSpace, len(publicSymbols))
	for id, symbolSpace := range publicSymbols {
		internalPublicSymbols[symbols.PackageIdentifier{OrgName: id.OrgName, ModuleName: id.ModuleName}] = symbolSpace
	}
	return symbols.Resolve(ctx, pkgID, compilationUnits, implicitImports, internalPublicSymbols, defaultOrg, span)
}

// ResolvePublicNodeTypes resolves the types exposed by a package.
func ResolvePublicNodeTypes(
	ctx *context.CompilerContext,
	pkg *ast.BLangPackage,
	importedSymbols map[string]model.ExportedSymbolSpace,
) {
	span := ctx.StartPackageSpan("Top-Level Type Resolution", pkg.PackageID)
	defer span.End()
	semantictypes.ResolvePublicNodes(ctx, pkg, importedSymbols, span)
}

// ResolvePrivateNodesTypes resolves package-local nodes and function bodies.
func ResolvePrivateNodesTypes(
	ctx *context.CompilerContext,
	pkg *ast.BLangPackage,
	importedSymbols map[string]model.ExportedSymbolSpace,
) {
	span := ctx.StartPackageSpan("Local Type Resolution", pkg.PackageID)
	defer span.End()
	semantictypes.ResolvePrivateNodes(ctx, pkg, importedSymbols, span)
}

// AnalyzeSemantics performs semantic analysis on a resolved package.
func AnalyzeSemantics(
	ctx *context.CompilerContext,
	pkg *ast.BLangPackage,
	importedSymbols map[string]model.ExportedSymbolSpace,
) {
	span := ctx.StartPackageSpan("Semantic Analysis", pkg.PackageID)
	defer span.End()
	analysis.Analyze(ctx, pkg, importedSymbols, span)
}

// PackageCFG is the public handle for a package control-flow graph.
type PackageCFG struct {
	graph *cfg.PackageCFG
}

// CreateControlFlowGraph builds control-flow graphs for a package.
func CreateControlFlowGraph(ctx *context.CompilerContext, pkg *ast.BLangPackage) *PackageCFG {
	span := ctx.StartPackageSpan("CFG Creation", pkg.PackageID)
	defer span.End()
	return &PackageCFG{graph: cfg.Build(ctx, pkg, span)}
}

// AnalyzeCFG performs reachability, return, and initialization analyses.
func AnalyzeCFG(ctx *context.CompilerContext, pkg *ast.BLangPackage, graph *PackageCFG) {
	span := ctx.StartPackageSpan("CFG Analysis", pkg.PackageID)
	defer span.End()
	cfg.Analyze(ctx, pkg, graph.graph, span)
}

// PrintCFGDot renders graph in Graphviz DOT format.
func PrintCFGDot(ctx *context.CompilerContext, graph *PackageCFG) string {
	return cfg.NewCFGDotExporter(ctx).Export(graph.graph)
}

// PrettyPrintCFG renders graph as deterministic text.
func PrettyPrintCFG(ctx *context.CompilerContext, graph *PackageCFG) string {
	return cfg.NewCFGPrettyPrinter(ctx).Print(graph.graph)
}
