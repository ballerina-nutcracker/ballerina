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

// Package context represents the front end state
package context

import (
	"sync"
	"time"

	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
)

type CompilationStage string

const (
	StageParse                  CompilationStage = "Parse"
	StageASTBuild               CompilationStage = "AST Build"
	StageImportResolution       CompilationStage = "Import Resolution"
	StageSymbolResolution       CompilationStage = "Symbol Resolution"
	StageTopLevelTypeResolution CompilationStage = "Top-Level Type Resolution"
	StageLocalNodeResolution    CompilationStage = "Local Node Resolution"
	StageSemanticAnalysis       CompilationStage = "Semantic Analysis"
	StageCFGCreation            CompilationStage = "CFG Creation"
	StageCFGAnalysis            CompilationStage = "CFG Analysis"
	StageDesugaring             CompilationStage = "Desugaring"
	StageBIRGeneration          CompilationStage = "BIR Generation"
)

type StageTiming struct {
	Name     CompilationStage
	Duration time.Duration
}

type ModuleStats struct {
	ModuleName string
	Stages     []StageTiming
}

type activeStage struct {
	name  CompilationStage
	start time.Time
}

// CompilerContext maintains frontend stage state for a package.
type CompilerContext struct {
	env         *CompilerEnvironment
	mu          sync.Mutex
	diagnostics []diagnostics.Diagnostic
	moduleStats *ModuleStats
	stage       activeStage
}

func (c *CompilerContext) DiagnosticEnv() *diagnostics.DiagnosticEnv {
	return c.env.DiagnosticEnv()
}

func (c *CompilerContext) NewSymbolSpace(packageID model.PackageID) *model.SymbolSpace {
	return c.env.NewSymbolSpace(packageID)
}

func (c *CompilerContext) NewModuleScope(pkg model.PackageID, prefixes map[string]model.ExportedSymbolSpace) *model.ModuleScope {
	return c.env.NewModuleScope(pkg, prefixes)
}

func (c *CompilerContext) NewFunctionScope(parent model.Scope, pkg model.PackageID) *model.FunctionScope {
	return c.env.NewFunctionScope(parent, pkg)
}

func (c *CompilerContext) NewBlockScope(parent model.Scope, pkg model.PackageID) *model.BlockScope {
	return c.env.NewBlockScope(parent, pkg)
}

func (c *CompilerContext) AddSymbolToSameSpace(ref model.SymbolRef, name string, symbol model.Symbol) model.SymbolRef {
	return c.env.AddSymbolToSameSpace(ref, name, symbol)
}

func (c *CompilerContext) GetSymbol(symbol model.SymbolRef) model.Symbol {
	return c.env.GetSymbol(symbol)
}

func (c *CompilerContext) SymbolPackage(symbol model.SymbolRef) model.PackageIdentifier {
	return c.env.SymbolPackage(symbol)
}

// CreateNarrowedSymbol create a narrowed symbol for the given baseRef symbol. IMPORTANT: baseRef must be the actual symbol
// not a narrowed symbol.
func (c *CompilerContext) CreateNarrowedSymbol(baseRef model.SymbolRef) model.SymbolRef {
	return c.env.CreateNarrowedSymbol(baseRef)
}

func (c *CompilerContext) CreateFunctionSymbol(space *model.SymbolSpace, name string, signature model.FunctionSignature, fnTy semtypes.SemType) model.SymbolRef {
	return c.env.CreateFunctionSymbol(space, name, signature, fnTy)
}

func (c *CompilerContext) UnnarrowedSymbol(symbol model.SymbolRef) model.SymbolRef {
	return c.env.UnnarrowedSymbol(symbol)
}

func (c *CompilerContext) SymbolName(symbol model.SymbolRef) string {
	return c.env.GetSymbol(symbol).Name()
}

func (c *CompilerContext) SymbolType(symbol model.SymbolRef) semtypes.SemType {
	return c.env.GetSymbol(symbol).Type()
}

func (c *CompilerContext) SymbolKind(symbol model.SymbolRef) model.SymbolKind {
	return c.env.GetSymbol(symbol).Kind()
}

func (c *CompilerContext) SymbolIsPublic(symbol model.SymbolRef) bool {
	return c.GetSymbol(symbol).IsPublic()
}

func (c *CompilerContext) SetSymbolType(symbol model.SymbolRef, ty semtypes.SemType) {
	c.GetSymbol(symbol).SetType(ty)
}

func (c *CompilerContext) DistinctTypeID(symbol model.SymbolRef) int {
	return c.env.DistinctTypeID(symbol)
}

func (c *CompilerContext) DistinctTypeSymbolRef(id int) (model.SymbolRef, bool) {
	return c.env.DistinctTypeSymbolRef(id)
}

func (c *CompilerContext) GetDefaultPackage() *model.PackageID {
	return c.env.GetDefaultPackage()
}

func (c *CompilerContext) NewPackageID(orgName model.Name, nameComps []model.Name, version model.Name) *model.PackageID {
	return c.env.NewPackageID(orgName, nameComps, version)
}

func (c *CompilerContext) Unimplemented(message string, pos diagnostics.Location) {
	c.addDiagnostic("UNIMPLEMENTED_ERROR", diagnostics.Fatal, message, pos)
}

func (c *CompilerContext) InternalError(message string, pos diagnostics.Location) {
	c.addDiagnostic("INTERNAL_ERROR", diagnostics.Fatal, message, pos)
}

func (c *CompilerContext) SyntaxError(message string, pos diagnostics.Location) {
	c.addDiagnostic("SYNTAX_ERROR", diagnostics.Error, message, pos)
}

func (c *CompilerContext) SemanticError(message string, pos diagnostics.Location) {
	c.addDiagnostic("SEMANTIC_ERROR", diagnostics.Error, message, pos)
}

func (c *CompilerContext) addDiagnostic(code string, severity diagnostics.DiagnosticSeverity, message string, pos diagnostics.Location) {
	diagnostic := diagnostics.CreateDiagnostic(diagnostics.NewDiagnosticInfo(&code, message, severity), pos)
	c.mu.Lock()
	c.diagnostics = append(c.diagnostics, diagnostic)
	c.mu.Unlock()
}

func (c *CompilerContext) HasDiagnostics() bool {
	return len(c.diagnostics) > 0
}

func (c *CompilerContext) HasErrors() bool {
	for _, diag := range c.diagnostics {
		switch diag.DiagnosticInfo().Severity() {
		case diagnostics.Error, diagnostics.Fatal:
			return true
		}
	}
	return false
}

func (c *CompilerContext) Diagnostics() []diagnostics.Diagnostic {
	return c.diagnostics
}

func NewCompilerContext(env *CompilerEnvironment) *CompilerContext {
	return &CompilerContext{
		env: env,
	}
}

// GetTypeEnv returns the type environment for this context
func (c *CompilerContext) GetTypeEnv() semtypes.Env {
	return c.env.GetTypeEnv()
}

func (c *CompilerContext) GetNextAnonymousFunctionKey(packageID *model.PackageID) string {
	return c.env.GetNextAnonymousFunctionKey(packageID)
}

func (c *CompilerContext) GetNextAnonymousTypeKey(packageID *model.PackageID) string {
	return c.env.GetNextAnonymousTypeKey(packageID)
}

func (c *CompilerContext) InitModuleStats(moduleName string) {
	if !c.env.statsEnabled {
		return
	}
	c.moduleStats = &ModuleStats{ModuleName: moduleName}
}

func (c *CompilerContext) StartStage(name CompilationStage) {
	if !c.env.statsEnabled {
		return
	}
	c.stage = activeStage{name: name, start: time.Now()}
}

func (c *CompilerContext) EndStage() {
	if !c.env.statsEnabled {
		return
	}
	c.moduleStats.Stages = append(c.moduleStats.Stages, StageTiming{
		Name:     c.stage.name,
		Duration: time.Since(c.stage.start),
	})
}

func (c *CompilerContext) GetModuleStats() *ModuleStats {
	return c.moduleStats
}
