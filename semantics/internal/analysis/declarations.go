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

// Package analysis is the semantic analyzer (stage 6): a single recursive
// descent that validates every node once under a chain of scope contexts.
package analysis

import (
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// Analyze validates a type-resolved package.
func Analyze(ctx *context.CompilerContext, pkg *ast.BLangPackage, importedSymbols map[string]model.ExportedSymbolSpace) {
	cx := scope.NewPackage(ctx, semtypes.ContextFrom(ctx.GetTypeEnv()), pkg, importedSymbols)
	validatePackage(cx, pkg)
}

// validatePackage validates every top-level declaration; a declaration that
// reports is abandoned and the next one runs.
func validatePackage(cx scope.Context, pkg *ast.BLangPackage) {
	aliases := make([]string, 0, len(pkg.Imports))
	for _, imp := range pkg.Imports {
		validateImport(cx, imp, aliases)
		if imp.Alias.OriginalValue != string(model.IGNORE) {
			aliases = append(aliases, imp.Alias.GetValue())
		}
	}
	for _, c := range pkg.Constants {
		validateVariable(cx, c, c.GetAssociatedType())
	}
	for _, v := range pkg.GlobalVars {
		validateVariable(cx, v, moduleVarExpectedType(v.IsListener(), cx.CompilerCtx().SymbolType(v.Symbol())))
	}
	for _, svc := range pkg.Services {
		validateService(cx, svc)
	}
	for _, fn := range pkg.Functions {
		validateFunction(cx, fn)
	}
	for _, td := range pkg.TypeDefinitions {
		validateTypeDefinition(cx, td)
	}
	for _, a := range pkg.Annotations {
		validateAnnotation(cx, a)
	}
	if pkg.InitFunction != nil {
		validateFunction(cx, pkg.InitFunction)
	}
	for _, c := range pkg.ClassDefinitions {
		validateClass(cx, c)
	}
}

func validateImport(cx scope.Context, imp *ast.BLangImportPackage, earlierAliases []string) {
	alias := imp.Alias.GetValue()
	if imp.Alias.OriginalValue != string(model.IGNORE) && slices.Contains(earlierAliases, alias) {
		cx.SemanticErr(strings.Join([]string{"import alias '", alias, "' already defined"}, ""), imp.GetPosition())
	}
}

// moduleVarExpectedType is a module variable's type, adjusted for a listener's
// initializer.
func moduleVarExpectedType(listener bool, symbolType semtypes.SemType) semtypes.SemType {
	if listener {
		return common.ListenerInitExpectedType(symbolType)
	}
	return symbolType
}

// validateVariable validates a module-level variable, constant, local
// variable or class field: annotations, type descriptor and initializer.
func validateVariable(cx scope.Context, v *ast.BLangVariable, expected semtypes.SemType) bool {
	if !validateAnnotationAttachments(cx, v.AnnAttachments) {
		return false
	}
	if typeNode := v.TypeNode(); typeNode != nil {
		if !dispatchTypeDesc(cx, typeNode) {
			return false
		}
	}
	if v.Expr != nil {
		f, ok := validateActionOrExpr(cx, v.Expr, expected)
		if !ok {
			return false
		}
		if md, isModuleVar := cx.ModuleVar(v.Symbol()); isModuleVar && md.IsIsolated() && !f.Isolated() {
			cx.SemanticErr("initializer of an isolated variable must be an isolated expression", v.Expr.GetPosition())
			return false
		}
	}
	if v.GetName().GetValue() == string(model.IGNORE) &&
		!semtypes.IsSubtype(cx.TyCtx(), cx.CompilerCtx().SymbolType(v.Symbol()), semtypes.Any) {
		cx.SemanticErr("wildcard binding pattern type must be a subtype of 'any'", v.GetPosition())
		return false
	}
	return true
}

func validateTypeDefinition(cx scope.Context, td *ast.BLangTypeDefinition) {
	typeDesc := td.GetTypeData().TypeDescriptor
	if typeDesc == nil {
		cx.InternalErr("type definition without a type descriptor", td.GetPosition())
		return
	}
	if !dispatchTypeDesc(cx, typeDesc) {
		return
	}
	validateAnnotationAttachments(cx, td.GetAnnotationAttachments())
}

func validateAnnotation(cx scope.Context, a *ast.BLangAnnotation) {
	if !validateAnnotationAttachments(cx, a.AnnAttachments) {
		return
	}
	if typeDesc := a.GetTypeDescriptor(); typeDesc != nil {
		dispatchTypeDesc(cx, typeDesc)
	}
}

func validateAnnotationAttachments(cx scope.Context, atts []ast.BLangAnnotationAttachment) bool {
	for i := range atts {
		if atts[i].Expr == nil {
			continue
		}
		if _, ok := validateActionOrExpr(cx, atts[i].Expr, semtypes.SemType{}); !ok {
			return false
		}
	}
	return true
}
