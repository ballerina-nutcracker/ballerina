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
	"path"
	"sort"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/birgen"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/desugar"
	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/nodebuilder"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/test_util/langlib"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
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

// stdlibEntry describes one embedded standard-library module to pre-compile.
// pkg is the Ballerina.toml package directory under lib/stdlibs/ballerina/
// (e.g. "protobuf"); module is the full dotted module name as used in import
// statements (e.g. "protobuf.types.any"); dir is the path, relative to the
// package's platform directory, holding that module's .bal files ("" for the
// package's root module, "modules/types.any" for a sub-module). For a
// single-module package, pkg == module and dir == "".
type stdlibEntry struct {
	org      string
	pkg      string
	module   string
	dir      string
	version  string
	platform string
}

func flatEntry(pkg, version, platform string) stdlibEntry {
	return stdlibEntry{org: "ballerina", pkg: pkg, module: pkg, version: version, platform: platform}
}

func subModuleEntry(pkg, subModuleDir, version, platform string) stdlibEntry {
	return stdlibEntry{
		org: "ballerina", pkg: pkg, module: pkg + "." + subModuleDir,
		dir: "modules/" + subModuleDir, version: version, platform: platform,
	}
}

// builtinStdlibs is the ordered list of standard-library modules baked into the
// binary that are still seeded manually for hand-rolled compile drivers.
// Order matters: a module must appear after all modules it imports
// (e.g. io before os, time before crypto; a package's root module before its
// sub-modules).
var builtinStdlibs = []stdlibEntry{
	flatEntry("io", "0.0.1", "go1.26"),
	flatEntry("http", "0.0.1", "go1.26"),
	flatEntry("log", "0.0.1", "go1.26"),
	flatEntry("math.vector", "0.0.1", "go1.26"),
	flatEntry("os", "0.0.1", "go1.26"),
	flatEntry("random", "0.0.1", "go1.26"),
	flatEntry("time", "0.0.1", "go1.26"),
	flatEntry("url", "0.0.1", "go1.26"),
	flatEntry("crypto", "0.0.1", "go1.26"),
	flatEntry("avro", "0.0.1", "go1.26"),
	flatEntry("protobuf", "0.0.1", "go1.26"),
	subModuleEntry("protobuf", "types.any", "0.0.1", "go1.26"),
	subModuleEntry("protobuf", "types.duration", "0.0.1", "go1.26"),
	subModuleEntry("protobuf", "types.empty", "0.0.1", "go1.26"),
	subModuleEntry("protobuf", "types.struct", "0.0.1", "go1.26"),
	subModuleEntry("protobuf", "types.timestamp", "0.0.1", "go1.26"),
	subModuleEntry("protobuf", "types.wrappers", "0.0.1", "go1.26"),
}

// moduleBalFiles returns the sorted list of .bal file paths (embed-FS relative)
// making up the given module's source, under the package's platform directory.
func moduleBalFiles(entry stdlibEntry) ([]string, error) {
	moduleDir := fmt.Sprintf("ballerina/%s/%s/%s", entry.pkg, entry.version, entry.platform)
	if entry.dir != "" {
		moduleDir = moduleDir + "/" + entry.dir
	}
	dirEntries, err := fs.ReadDir(stdlibs.FS, moduleDir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".bal") {
			continue
		}
		files = append(files, path.Join(moduleDir, de.Name()))
	}
	sort.Strings(files)
	return files, nil
}

// loadBuiltinPublicSymbols compiles the embedded standard-library modules into
// sibling CompilerContexts that share env (and thus the same type-env and
// symbol table). The returned map can be merged directly into the publicSymbols
// passed to semantics.ResolveImports.
func loadBuiltinPublicSymbols(env *context.CompilerEnvironment) map[semantics.PackageIdentifier]model.ExportedSymbolSpace {
	result := make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace)

	for _, entry := range builtinStdlibs {
		balFiles, err := moduleBalFiles(entry)
		if err != nil || len(balFiles) == 0 {
			continue
		}

		cx := context.NewCompilerContext(env)
		compilationUnits := make([]*ast.BLangCompilationUnit, 0, len(balFiles))
		ok := true
		for _, balPath := range balFiles {
			contentBytes, err := fs.ReadFile(stdlibs.FS, balPath)
			if err != nil {
				ok = false
				break
			}
			content := string(contentBytes)

			virtualPath := "$stdlib/" + balPath
			cx.DiagnosticEnv().RegisterFile(virtualPath, text.NewStringTextDocument(content))

			st, err := parser.GetSyntaxTree(cx, virtualPath, content)
			if err != nil || cx.HasDiagnostics() {
				ok = false
				break
			}

			cu := nodebuilder.GetCompilationUnit(cx, st)
			if cu == nil || cx.HasDiagnostics() {
				ok = false
				break
			}
			compilationUnits = append(compilationUnits, cu)
		}
		if !ok {
			continue
		}

		pkgID := cx.NewPackageID(
			model.Name(entry.org),
			model.CreateNameComps(model.Name(entry.module)),
			model.DEFAULT_VERSION,
		)
		for _, cu := range compilationUnits {
			cu.SetPackageID(pkgID)
		}

		// Pass accumulated stdlib symbols so modules that import other stdlib
		// modules (e.g. os→io, crypto→time, protobuf.types.any→protobuf) resolve correctly.
		pkgScope, exported, importedSymbols := semantics.ResolveSymbols(
			cx,
			*pkgID,
			compilationUnits,
			make(map[string]model.ExportedSymbolSpace),
			result,
			nil,
			entry.org,
			"",
		)
		if cx.HasErrors() {
			continue
		}
		pkg := nodebuilder.ToPackageFromCompilationUnits(cx, compilationUnits)
		if cx.HasErrors() {
			continue
		}
		pkg.PackageID = pkgID
		pkg.Scope = pkgScope
		pkg.Imports = nil

		semantics.ResolvePublicNodeTypes(cx, pkg, importedSymbols)
		if cx.HasErrors() {
			continue
		}

		result[semantics.PackageIdentifier{OrgName: entry.org, ModuleName: entry.module}] = exported
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
	if cx.HasDiagnostics() {
		return nil, fmt.Errorf("parsing failed with diagnostics")
	}
	if phase == PhaseParse {
		return result, nil
	}

	// Phase 2: AST
	result.CompilationUnit = nodebuilder.GetCompilationUnit(cx, syntaxTree)
	if result.CompilationUnit == nil || cx.HasDiagnostics() {
		return nil, fmt.Errorf("AST generation failed: compilation unit is nil")
	}
	if phase == PhaseAST {
		result.Package = nodebuilder.ToPackageFromCompilationUnits(cx, []*ast.BLangCompilationUnit{result.CompilationUnit})
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
	pkgScope, _, importedSymbols := semantics.ResolveSymbols(
		cx,
		*pkgID,
		compilationUnits,
		langlibs.ImplicitImports,
		langlibs.PublicSymbols,
		nil,
		"",
		"",
	)
	result.Package = nodebuilder.ToPackageFromCompilationUnits(cx, compilationUnits)
	result.Package.PackageID = pkgID
	result.Package.Scope = pkgScope
	if phase == PhaseSymbolResolution || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 4: Type Resolution (top level nodes)
	semantics.ResolvePublicNodeTypes(cx, result.Package, importedSymbols)
	if phase == PhaseTypeResolution || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 5: Type Resolution (inner nodes)
	semantics.ResolvePrivateNodesTypes(cx, result.Package, importedSymbols)
	if phase == PhaseTypeNarrowing || cx.HasDiagnostics() {
		return result, nil
	}

	// Phase 6: Semantic Analysis
	semantics.AnalyzeSemantics(cx, result.Package, importedSymbols)
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
	result.BIRPackage = birgen.GenBir(cx, result.Package)
	if result.BIRPackage == nil {
		return result, nil
	}
	return result, nil
}
