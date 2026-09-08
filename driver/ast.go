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

package driver

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/nodebuilder"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

// ASTDocument is one source document of a module together with its text and
// the compilation unit built from it.
type ASTDocument struct {
	SourcePath      string
	TextDocument    text.TextDocument
	CompilationUnit *ast.BLangCompilationUnit
}

type moduleBase struct {
	ID           ModuleDescriptor
	Documents    []*ASTDocument
	PackageNode  *ast.BLangPackage
	imported     model.ImportedSymbolSpaces
	implicitName string
}

// ASTModule is a module whose documents have been lowered to a package AST
// without syntax-error recovery.
type ASTModule struct{ *moduleBase }

// RecoveredASTModule is a module lowered to a package AST with syntax-error
// recovery applied, for tooling that must operate on incomplete sources.
type RecoveredASTModule struct{ *moduleBase }

// PartiallyResolvedModule is a module whose symbols and public-node types have
// been resolved and published, but whose private nodes have not.
type PartiallyResolvedModule struct{ *moduleBase }

// ToAST builds the package AST for a parsed module. It returns nil if ctx is
// cancelled or the AST-build stage reports errors; the diagnostics are recorded
// on ctx.
func ToAST(ctx *Context, module *ParsedModule) *ASTModule {
	base := toAST(ctx, module, false)
	if base == nil {
		return nil
	}
	return &ASTModule{moduleBase: base}
}

// ToRecoveredAST builds the package AST for a parsed module with syntax-error
// recovery. It returns nil if ctx is cancelled or the AST-build stage reports
// errors; the diagnostics are recorded on ctx.
func ToRecoveredAST(ctx *Context, module *ParsedModule) *RecoveredASTModule {
	base := toAST(ctx, module, true)
	if base == nil {
		return nil
	}
	return &RecoveredASTModule{moduleBase: base}
}

// toAST runs the AST-build stage over every document of module, combining the
// resulting compilation units into one package node. When recovered is true the
// recovering node builder is used. It checks ctx for cancellation between
// documents and returns nil, having drained the stage's diagnostics into ctx,
// on cancellation or error.
func toAST(ctx *Context, module *ParsedModule, recovered bool) *moduleBase {
	if err := ctx.Err(); err != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	cx.StartStage(compilercontext.StageASTBuild)
	packageID := modelPackageID(cx, module.ID)
	documents := make([]*ASTDocument, len(module.Documents))
	compilationUnits := make([]*ast.BLangCompilationUnit, len(module.Documents))
	for index, document := range module.Documents {
		if err := ctx.Err(); err != nil {
			cx.EndStage()
			ctx.drainModule(cx, module.ID, compilercontext.StageASTBuild, false, 0)
			return nil
		}
		var compilationUnit *ast.BLangCompilationUnit
		if recovered {
			compilationUnit = nodebuilder.GetRecoveredCompilationUnitForSource(cx, document.SyntaxTree, packageID)
		} else {
			compilationUnit = nodebuilder.GetCompilationUnitForSource(cx, document.SyntaxTree, packageID)
		}
		if cx.HasErrors() {
			cx.EndStage()
			ctx.drainModule(cx, module.ID, compilercontext.StageASTBuild, false, 0)
			return nil
		}
		documents[index] = &ASTDocument{SourcePath: document.SourcePath, TextDocument: document.TextDocument, CompilationUnit: compilationUnit}
		compilationUnits[index] = compilationUnit
	}
	if err := ctx.Err(); err != nil {
		cx.EndStage()
		ctx.drainModule(cx, module.ID, compilercontext.StageASTBuild, false, 0)
		return nil
	}
	packageNode := nodebuilder.ToPackageFromCompilationUnits(cx, compilationUnits)
	packageNode.PackageID = packageID
	cx.EndStage()
	ctx.drainModule(cx, module.ID, compilercontext.StageASTBuild, false, 0)
	if cx.HasErrors() || ctx.Err() != nil {
		return nil
	}
	return &moduleBase{ID: module.ID, Documents: documents, PackageNode: packageNode,
		imported: model.NewImportedSymbolSpaces(), implicitName: module.implicitName}
}

// modelPackageID interns a model package ID for descriptor, splitting the
// module name on dots into name components and substituting the default version
// when the descriptor carries none.
func modelPackageID(cx *compilercontext.CompilerContext, descriptor ModuleDescriptor) *model.PackageID {
	parts := strings.Split(descriptor.Name, ".")
	components := make([]model.Name, len(parts))
	for i, part := range parts {
		components[i] = model.Name(part)
	}
	version := model.Name(descriptor.Package.Version)
	if version == "" {
		version = model.DEFAULT_VERSION
	}
	return cx.NewPackageID(model.Name(descriptor.Package.Org), components, version)
}
