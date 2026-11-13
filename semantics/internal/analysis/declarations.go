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
func Analyze(
	ctx *context.CompilerContext,
	pkg *ast.BLangPackage,
	importedSymbols map[string]model.ExportedSymbolSpace,
	parent context.TraceSpan,
) {
	metadataSpan := parent.StartChild("Module Variable Metadata", "")
	cx := scope.NewPackage(ctx, semtypes.ContextFrom(ctx.GetTypeEnv()), pkg, importedSymbols)
	metadataSpan.End()
	validatePackage(cx, pkg, parent)
}

// validatePackage validates every top-level declaration, each under its own
// span; a declaration that reports is abandoned and the next one runs.
func validatePackage(cx scope.Context, pkg *ast.BLangPackage, parent context.TraceSpan) {
	aliases := make([]string, 0, len(pkg.Imports))
	for _, imp := range pkg.Imports {
		span := parent.StartChild("Import", imp.Alias.GetValue())
		validateImport(cx, imp, aliases)
		span.End()
		if imp.Alias.OriginalValue != string(model.IGNORE) {
			aliases = append(aliases, imp.Alias.GetValue())
		}
	}
	for _, c := range common.DeclaredNodes(pkg.Constants) {
		span := parent.StartChild("Constant", c.Name.GetValue())
		validateVariable(cx, c, c.GetAssociatedType())
		span.End()
	}
	for _, v := range common.DeclaredNodes(pkg.GlobalVars) {
		span := parent.StartChild("Global Variable", v.Name.GetValue())
		validateVariable(cx, v, moduleVarExpectedType(v.IsListener(), cx.CompilerCtx().SymbolType(v.Symbol())))
		span.End()
	}
	for _, svc := range pkg.Services {
		span := parent.StartChildFunc("Service", func() string { return cx.CompilerCtx().SymbolName(svc.Symbol()) })
		validateService(cx, svc)
		span.End()
	}
	for _, fn := range common.DeclaredNodes(pkg.Functions) {
		span := parent.StartChild("Function", fn.Name.GetValue())
		validateFunction(cx, fn)
		span.End()
	}
	for _, td := range pkg.TypeDefinitions {
		span := parent.StartChild("Type Definition", td.Name.GetValue())
		validateTypeDefinition(cx, td)
		span.End()
	}
	for _, a := range pkg.Annotations {
		span := parent.StartChild("Annotation", a.Name.GetValue())
		validateAnnotation(cx, a)
		span.End()
	}
	if pkg.InitFunction != nil {
		span := parent.StartChild("Init Function", pkg.InitFunction.Name.GetValue())
		validateFunction(cx, pkg.InitFunction)
		span.End()
	}
	for _, c := range common.DeclaredNodes(pkg.ClassDefinitions) {
		span := parent.StartChild("Class Definition", c.Name.GetValue())
		validateClass(cx, c)
		span.End()
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
		md, isModuleVar := cx.ModuleVar(v.Symbol())
		isolatedVar := isModuleVar && md.IsIsolated()
		exprCx := cx
		if isolatedVar {
			exprCx = scope.NewIsolatedVarInit(cx, v)
		}
		f, ok := validateActionOrExpr(exprCx, v.Expr, expected)
		if !ok {
			return false
		}
		if isolatedVar && !f.Isolated() {
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
