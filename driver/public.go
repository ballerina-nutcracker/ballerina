// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics"
)

func ResolvePublicNodes(ctx *Context, module *ASTModule) *PartiallyResolvedModule {
	if err := ctx.Err(); err != nil {
		return nil
	}
	cx := ctx.newCompilerContext(module.ID)
	packageID := module.PackageNode.PackageID
	units := make([]*ast.BLangCompilationUnit, len(module.Documents))
	for index, document := range module.Documents {
		units[index] = document.CompilationUnit
	}

	ctx.env.mu.Lock()
	publicSymbols := make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace, len(ctx.env.publishedSymbols))
	for identifier, symbols := range ctx.env.publishedSymbols {
		publicSymbols[semantics.PackageIdentifier{OrgName: identifier.Organization, ModuleName: identifier.Package}] = symbols
	}
	implicitSymbols := make(map[string]model.ExportedSymbolSpace, len(ctx.env.implicitSymbols))
	for name, symbols := range ctx.env.implicitSymbols {
		implicitSymbols[name] = symbols
	}
	ctx.env.mu.Unlock()

	cx.StartStage(compilercontext.StageSymbolResolution)
	scope, exported, imported := semantics.ResolveSymbols(cx, *packageID, units, implicitSymbols, publicSymbols, module.ID.Package.Org)
	cx.EndStage()
	diagnosticOffset, statsOffset := ctx.drainModuleSince(cx, module.ID, compilercontext.StageSymbolResolution, false, 0, 0, 0)
	module.PackageNode.Scope = scope
	module.PackageNode.Imports = nil
	if cx.HasErrors() || ctx.Err() != nil {
		return nil
	}

	cx.StartStage(compilercontext.StageTopLevelTypeResolution)
	semantics.ResolvePublicNodeTypes(cx, module.PackageNode, imported)
	cx.EndStage()
	ctx.drainModuleSince(cx, module.ID, compilercontext.StageTopLevelTypeResolution, false, 0, diagnosticOffset, statsOffset)
	if cx.HasErrors() || ctx.Err() != nil {
		return nil
	}

	identifier := model.PackageIdentifier{Organization: module.ID.Package.Org, Package: module.ID.Name, Version: module.ID.Package.Version}
	ctx.env.mu.Lock()
	ctx.env.publishedSymbols[identifier] = exported
	if module.implicitName != "" && module.implicitName != "lang.runtime" {
		ctx.env.implicitSymbols[module.implicitName] = exported
	}
	ctx.env.mu.Unlock()
	base := *module.moduleBase
	base.imported = imported
	return &PartiallyResolvedModule{moduleBase: &base}
}
