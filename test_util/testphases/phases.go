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

package testphases

import (
	stdcontext "context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing/fstest"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/bir"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/test_util/langlib"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type Phase int

const (
	PhaseParse Phase = iota
	PhaseAST
	PhaseSymbolResolution
	PhaseTypeResolution
	PhaseTypeNarrowing
	PhaseSemanticAnalysis
	PhaseCFG
	PhaseCFGAnalysis
	PhaseDesugar
	PhaseBIR
)

type PipelineResult struct {
	CompilationUnit *ast.BLangCompilationUnit
	Package         *ast.BLangPackage
	CFG             *semantics.PackageCFG
	BIRPackage      *bir.BIRPackage
	DriverContext   *driver.Context
}

func LoadLanglibs(_ *compilercontext.CompilerEnvironment, _ *compilercontext.CompilerContext) (*langlib.Symbols, error) {
	return &langlib.Symbols{}, nil
}

func RunPipeline(env *compilercontext.CompilerEnvironment, cx *compilercontext.CompilerContext, _ *langlib.Symbols,
	phase Phase, inputPath string,
) (*PipelineResult, error) {
	directory := filepath.Dir(inputPath)
	return runPipeline(env, cx, phase, os.DirFS(directory), filepath.Base(inputPath))
}

func RunPipelineWithContent(env *compilercontext.CompilerEnvironment, cx *compilercontext.CompilerContext, _ *langlib.Symbols,
	phase Phase, inputPath string, content string,
) (*PipelineResult, error) {
	name := filepath.Base(inputPath)
	fsys := fstest.MapFS{name: &fstest.MapFile{Data: []byte(content)}}
	return runPipeline(env, cx, phase, fsys, name)
}

func runPipeline(env *compilercontext.CompilerEnvironment, legacy *compilercontext.CompilerContext, phase Phase,
	fsys fs.FS, input string,
) (*PipelineResult, error) {
	resolver := projects.NewDriverDependencyResolver(fsys, projects.ProjectLoadConfig{})
	driverContext := driver.NewContext(stdcontext.Background(), driver.NewEnv(env))
	result := &PipelineResult{DriverContext: driverContext}
	sources, err := driver.FindSources(driverContext, fsys, input, "")
	if err != nil {
		return nil, err
	}
	// Stage corpus goldens historically use the compiler's anonymous default
	// package identity for standalone files. Preserve that harness identity
	// while the driver-facing CLI uses the synthesized file-stem descriptor.
	sources.Descriptor.Name = "."
	sources.Modules[0].ID.Package = sources.Descriptor
	sources.Modules[0].ID.Name = "."
	parsed, err := driver.Parse(driverContext, fsys, ".", sources, parser.DebugOptions{})
	if err != nil {
		return nil, err
	}
	if phase == PhaseParse {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	if driverContext.HasErrors() {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	if phase == PhaseAST {
		module := driver.ToAST(driverContext, parsed.Modules[0])
		if module != nil {
			populateASTResult(result, module)
		}
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	parsed, err = driver.ResolveDependencies(driverContext, parsed, resolver)
	if err != nil {
		return nil, err
	}
	if driverContext.HasErrors() {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}

	astModules := make([]*driver.ASTModule, len(parsed.Modules))
	var astWG sync.WaitGroup
	for index, module := range parsed.Modules {
		astWG.Add(1)
		go func() {
			defer astWG.Done()
			astModules[index] = driver.ToAST(driverContext, module)
		}()
	}
	astWG.Wait()
	for _, module := range astModules {
		if module == nil {
			mirrorDiagnostics(legacy, driverContext.Diagnostics())
			return result, nil
		}
	}
	publicModules := make([]*driver.PartiallyResolvedModule, len(astModules))
	for index, module := range astModules {
		publicModules[index] = driver.ResolvePublicNodes(driverContext, module)
		if publicModules[index] == nil {
			mirrorDiagnostics(legacy, driverContext.Diagnostics())
			return result, nil
		}
	}
	rootIndex := rootModuleIndex(parsed)
	populatePublicResult(result, publicModules[rootIndex])
	if phase <= PhaseTypeResolution {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}

	resolved := driver.ResolvePrivateNodes(driverContext, publicModules[rootIndex])
	if resolved == nil {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	result.Package = resolved.PackageNode
	if phase == PhaseTypeNarrowing {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	analyzed := driver.AnalyzeSemantics(driverContext, resolved)
	if analyzed == nil {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	result.Package = analyzed.PackageNode
	if phase == PhaseSemanticAnalysis {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	controlFlow := driver.CreateControlFlowGraph(driverContext, analyzed)
	if controlFlow == nil {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	result.CFG = controlFlow.CFG
	if phase == PhaseCFG {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	cfgAnalyzed := driver.AnalyzeControlFlowGraph(driverContext, controlFlow)
	if cfgAnalyzed == nil {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	result.Package = cfgAnalyzed.PackageNode
	if phase == PhaseCFGAnalysis {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	desugared := driver.Desugar(driverContext, cfgAnalyzed)
	if desugared == nil {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	restoreExplicitImports(desugared.PackageNode, publicModules[rootIndex].Documents)
	result.Package = desugared.PackageNode
	if phase == PhaseDesugar {
		mirrorDiagnostics(legacy, driverContext.Diagnostics())
		return result, nil
	}
	result.BIRPackage = driver.GenerateBIR(driverContext, desugared)
	mirrorDiagnostics(legacy, driverContext.Diagnostics())
	return result, nil
}

func restoreExplicitImports(pkg *ast.BLangPackage, documents []*driver.ASTDocument) {
	explicit := make([]*ast.BLangImportPackage, 0)
	for _, document := range documents {
		for _, node := range document.CompilationUnit.GetTopLevelNodes() {
			if imported, ok := node.(*ast.BLangImportPackage); ok {
				explicit = append(explicit, imported)
			}
		}
	}
	pkg.Imports = append(explicit, pkg.Imports...)
}

func rootModuleIndex(parsed *driver.ParsedPackage) int {
	for index, module := range parsed.Modules {
		if module.ID.Package == parsed.Root && module.ID.Name == parsed.Root.Name {
			return index
		}
	}
	return len(parsed.Modules) - 1
}

func populateASTResult(result *PipelineResult, module *driver.ASTModule) {
	result.Package = module.PackageNode
	if len(module.Documents) > 0 {
		result.CompilationUnit = module.Documents[0].CompilationUnit
	}
}

func populatePublicResult(result *PipelineResult, module *driver.PartiallyResolvedModule) {
	result.Package = module.PackageNode
	if len(module.Documents) > 0 {
		result.CompilationUnit = module.Documents[0].CompilationUnit
	}
}

func mirrorDiagnostics(cx *compilercontext.CompilerContext, values []diagnostics.Diagnostic) {
	for _, diagnostic := range values {
		location := diagnostic.Location()
		switch diagnostic.DiagnosticInfo().Severity() {
		case diagnostics.Fatal:
			cx.InternalError(diagnostic.Message(), location)
		case diagnostics.Error:
			cx.SemanticError(diagnostic.Message(), location)
		}
	}
}
