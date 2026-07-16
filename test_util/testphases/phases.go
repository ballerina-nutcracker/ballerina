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

// Package testphases provide utilities to run frontend upto a certain point so that a given
// frontend phase can be validated after that point
package testphases

import (
	"fmt"
	"io/fs"
	"os"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/bir"
	"ballerina-lang-go/context"
	"ballerina-lang-go/desugar"
	"ballerina-lang-go/lib/stdlibs"
	"ballerina-lang-go/model"
	"ballerina-lang-go/parser"
	"ballerina-lang-go/semantics"
	"ballerina-lang-go/test_util/langlib"
	"ballerina-lang-go/tools/text"
)

// Phase represents a frontend compilation phase
type Phase int

const (
	// PhaseParse runs only parsing (syntax tree generation)
	PhaseParse Phase = iota
	// PhaseAST runs parsing + AST generation
	PhaseAST
	// PhaseSymbolResolution runs through symbol resolution
	PhaseSymbolResolution
	// PhaseTypeResolution runs through type resolution
	PhaseTypeResolution
	// PhaseTypeNarrowing runs through type narrowing
	PhaseTypeNarrowing
	// PhaseSemanticAnalysis runs through semantic analysis
	PhaseSemanticAnalysis
	// PhaseCFG runs through CFG generation
	PhaseCFG
	// PhaseCFGAnalysis runs through CFG analysis (reachability, explicit return)
	PhaseCFGAnalysis
	// PhaseDesugar runs through desugaring
	PhaseDesugar
	// PhaseBIR runs through BIR generation
	PhaseBIR
)

// PipelineResult holds the results from running the frontend pipeline
type PipelineResult struct {
	CompilationUnit *ast.BLangCompilationUnit
	Package         *ast.BLangPackage
	CFG             *semantics.PackageCFG
	BIRPackage      *bir.BIRPackage
}

// stdlibEntry describes one embedded standard-library package to pre-compile.
type stdlibEntry struct {
	org      string
	name     string
	version  string
	platform string
}

// builtinStdlibs is the ordered list of standard-library packages baked into the
// binary that are still seeded manually for hand-rolled compile drivers.
// Order matters: a package must appear after all packages it imports
// (e.g. io before os, time before crypto).
var builtinStdlibs = []stdlibEntry{
	{"ballerina", "io", "0.0.1", "go1.26"},
	{"ballerina", "log", "0.0.1", "go1.26"},
	{"ballerina", "math.vector", "0.0.1", "go1.26"},
	{"ballerina", "os", "0.0.1", "go1.26"},
	{"ballerina", "random", "0.0.1", "go1.26"},
	{"ballerina", "time", "0.0.1", "go1.26"},
	{"ballerina", "url", "0.0.1", "go1.26"},
	{"ballerina", "crypto", "0.0.1", "go1.26"},
	{"ballerina", "uuid", "0.0.1", "go1.2"},
	{"ballerina", "mime", "0.0.1", "go1.2"},
	{"ballerina", "http", "0.0.1", "go1.26"},
	{"ballerina", "file", "0.0.1", "go1.2"},
	{"ballerina", "ldap", "0.0.1", "go1.2"},
	{"ballerina", "tcp", "0.0.1", "go1.2"},
	{"ballerina", "udp", "0.0.1", "go1.2"},
}

// loadBuiltinPublicSymbols compiles the embedded standard-library packages into
// sibling CompilerContexts that share env (and thus the same type-env and
// symbol table). The returned map can be merged directly into the publicSymbols
// passed to semantics.ResolveImports.
func loadBuiltinPublicSymbols(env *context.CompilerEnvironment) map[semantics.PackageIdentifier]model.ExportedSymbolSpace {
	result := make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace)

	for _, entry := range builtinStdlibs {
		balPath := fmt.Sprintf("ballerina/%s/%s/%s/%s.bal", entry.name, entry.version, entry.platform, entry.name)
		contentBytes, err := fs.ReadFile(stdlibs.FS, balPath)
		if err != nil {
			continue
		}
		content := string(contentBytes)

		cx := context.NewCompilerContext(env)
		virtualPath := fmt.Sprintf("$stdlib/ballerina/%s.bal", entry.name)
		cx.DiagnosticEnv().RegisterFile(virtualPath, text.NewStringTextDocument(content))

		st, err := parser.GetSyntaxTree(cx, virtualPath, content)
		if err != nil || cx.HasDiagnostics() {
			continue
		}

		cu := ast.GetCompilationUnit(cx, st)
		if cu == nil || cx.HasDiagnostics() {
			continue
		}
		pkgID := cx.NewPackageID(
			model.Name(entry.org),
			[]model.Name{model.Name(entry.name)},
			model.DEFAULT_VERSION,
		)
		cu.SetPackageID(pkgID)
		compilationUnits := []*ast.BLangCompilationUnit{cu}

		// Pass accumulated stdlib symbols so packages that import other stdlibs (e.g. os→io, crypto→time) resolve correctly.
		importedByCU := semantics.ResolveCompilationUnitImports(cx, compilationUnits, semantics.GetImplicitImports(cx),
			result, entry.org)
		pkgScope, exported := semantics.ResolveSymbols(cx, *pkgID, importedByCU)
		if cx.HasErrors() {
			continue
		}
		pkg := ast.ToPackageFromCompilationUnits(compilationUnits)
		pkg.PackageID = pkgID
		pkg.Scope = pkgScope
		pkg.Imports = nil

		semantics.ResolveTopLevelNodes(cx, pkg, importedByCU[0].Imports)
		if cx.HasErrors() {
			continue
		}

		result[semantics.PackageIdentifier{OrgName: entry.org, ModuleName: entry.name}] = exported
	}

	return result
}

func LoadLanglibs(env *context.CompilerEnvironment, cx *context.CompilerContext) (*langlib.Symbols, error) {
	stdlibSymbols := loadBuiltinPublicSymbols(env)
	symbols, err := langlib.Build(cx, stdlibSymbols)
	if err != nil {
		return nil, fmt.Errorf("loading lang libraries failed: %w", err)
	}
	return symbols, nil
}

// RunPipeline runs the frontend compilation pipeline up to the specified phase.
// It returns a PipelineResult containing the outputs relevant to that phase.
func RunPipeline(env *context.CompilerEnvironment, cx *context.CompilerContext, langlibs *langlib.Symbols, phase Phase, inputPath string) (*PipelineResult, error) {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", inputPath, err)
	}
	return RunPipelineWithContent(env, cx, langlibs, phase, inputPath, string(content))
}

// RunPipelineWithContent runs the frontend compilation pipeline for preloaded content.
// It returns a PipelineResult containing the outputs relevant to that phase.
func RunPipelineWithContent(env *context.CompilerEnvironment, cx *context.CompilerContext, langlibs *langlib.Symbols, phase Phase, inputPath string, content string) (*PipelineResult, error) {
	result := &PipelineResult{}

	// Register source file with DiagnosticEnv
	cx.DiagnosticEnv().RegisterFile(inputPath, text.NewStringTextDocument(content))

	// Phase 1: Parse
	syntaxTree, err := parser.GetSyntaxTree(cx, inputPath, content)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}
	if phase == PhaseParse {
		return result, nil
	}

	// Phase 2: AST
	result.CompilationUnit = ast.GetCompilationUnit(cx, syntaxTree)
	if result.CompilationUnit == nil || cx.HasDiagnostics() {
		return nil, fmt.Errorf("AST generation failed: compilation unit is nil")
	}
	if phase == PhaseAST {
		result.Package = ast.ToPackageFromCompilationUnits([]*ast.BLangCompilationUnit{result.CompilationUnit})
		return result, nil
	}

	// Phase 3: Symbol Resolution
	if langlibs == nil {
		var err error
		langlibs, err = LoadLanglibs(env, cx)
		if err != nil {
			return nil, err
		}
	}
	pkgID := result.CompilationUnit.GetPackageID()
	result.CompilationUnit.SetPackageID(pkgID)
	compilationUnits := []*ast.BLangCompilationUnit{result.CompilationUnit}
	importedByCU := semantics.ResolveCompilationUnitImports(cx, compilationUnits, langlibs.ImplicitImports, langlibs.PublicSymbols, "")
	pkgScope, _ := semantics.ResolveSymbols(cx, *pkgID, importedByCU)
	result.Package = ast.ToPackageFromCompilationUnits(compilationUnits)
	result.Package.PackageID = pkgID
	result.Package.Scope = pkgScope
	importedSymbols := importedByCU[0].Imports
	if phase == PhaseSymbolResolution || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 4: Type Resolution (top level nodes)
	semantics.ResolveTopLevelNodes(cx, result.Package, importedSymbols)
	if phase == PhaseTypeResolution || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 5: Type Resolution (inner nodes)
	semantics.ResolveLocalNodes(cx, result.Package, importedSymbols)
	if phase == PhaseTypeNarrowing || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 6: Semantic Analysis
	semanticAnalyzer := semantics.NewSemanticAnalyzer(cx)
	semanticAnalyzer.Analyze(result.Package, importedSymbols)
	if phase == PhaseSemanticAnalysis || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 7: CFG Generation
	result.CFG = semantics.CreateControlFlowGraph(cx, result.Package)
	if phase == PhaseCFG || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 8: CFG Analysis
	semantics.AnalyzeCFG(cx, result.Package, result.CFG)
	if phase == PhaseCFGAnalysis || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 9: Desugar
	result.Package = desugar.DesugarPackage(cx, result.Package, importedSymbols)
	if phase == PhaseDesugar || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 10: BIR Generation
	result.BIRPackage = bir.GenBir(cx, result.Package)
	return result, nil
}
