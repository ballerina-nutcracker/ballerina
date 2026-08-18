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

package types

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	balCommon "github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// Type resolution can run ephemerally while selecting among contextually
// expected types. An ephemeral resolver must be able to observe its own state
// changes and discard them when a candidate fails, without changing the shared
// AST or symbol metadata. These functions form the indirection through which
// that resolver can record and retrieve such state.
//
// All AST state reads and mutations performed during type resolution must go
// through these functions. Direct access would bypass an ephemeral resolver and
// could either read stale committed state or leak a failed candidate's state.

func nodeType(t typeResolver, node ast.BLangNode) semtypes.SemType {
	return t.nodeType(node)
}

func setNodeType(t typeResolver, node ast.BLangNode, ty semtypes.SemType) {
	t.setNodeType(node, ty)
}

func (t *packageTypeResolver) nodeType(node ast.BLangNode) semtypes.SemType {
	return node.GetDeterminedType()
}

func (t *packageTypeResolver) setNodeType(node ast.BLangNode, ty semtypes.SemType) {
	node.SetDeterminedType(ty)
}

func (f *functionTypeResolver) nodeType(node ast.BLangNode) semtypes.SemType {
	return node.GetDeterminedType()
}

func (f *functionTypeResolver) setNodeType(node ast.BLangNode, ty semtypes.SemType) {
	node.SetDeterminedType(ty)
}

type symbolNode interface {
	ast.BLangNode
	SetSymbol(model.SymbolRef)
}

type methodSymbolNode interface {
	ast.BLangNode
	SetMethodSymbol(model.SymbolRef)
}

func setNodeSymbol(_ typeResolver, node symbolNode, ref model.SymbolRef) {
	node.SetSymbol(ref)
}

func setInvocationResolvedSymbol(_ typeResolver, inv invocable, ref model.SymbolRef) {
	inv.SetResolvedSymbol(ref)
}

func setInvocationCallArgs(_ typeResolver, inv invocable, args []ast.BLangExpression) {
	inv.SetCallArgs(args)
}

func setMethodSymbol(_ typeResolver, node methodSymbolNode, ref model.SymbolRef) {
	node.SetMethodSymbol(ref)
}

func setLiteralValue(_ typeResolver, node *ast.BLangLiteral, value any) {
	node.SetValue(value)
}

func setTypedescMetadata(_ typeResolver, node *ast.BLangTypedescExpr, constraint semtypes.SemType, annotationValues values.AnnotationValues) {
	node.Constraint = constraint
	node.AnnotationValues = annotationValues
}

func setNewExpressionMetadata(_ typeResolver, node *ast.BLangNewExpression, atom *semtypes.MappingAtomicType, classRef model.SymbolRef) {
	node.AtomicType = atom
	node.ClassSymbol = classRef
}

func setNewExpressionArgs(_ typeResolver, node *ast.BLangNewExpression, args []ast.BLangExpression) {
	node.ArgsExprs = args
}

func setMappingConstructorAtomicType(_ typeResolver, node *ast.BLangMappingConstructorExpr, atom semtypes.MappingAtomicType) {
	node.AtomicType = atom
}

func setMappingConstructorFieldDefaults(_ typeResolver, node *ast.BLangMappingConstructorExpr, defaults []model.FieldDefault) {
	node.FieldDefaults = defaults
}

func setListConstructorAtomicType(_ typeResolver, node *ast.BLangListConstructorExpr, atom semtypes.ListAtomicType) {
	node.AtomicType = atom
}

func setGroupByNonGroupingKeys(_ typeResolver, clause *ast.BLangGroupByClause, keys balCommon.Set[string]) {
	clause.NonGroupingKeys = keys
}

func moveLangLibReceiver(_ typeResolver, expr *ast.BLangInvocation, args []ast.BLangExpression, pkgAlias ast.BLangIdentifier) {
	expr.ArgExprs = args
	expr.Expr = nil
	expr.PkgAlias = &pkgAlias
}

func setFunctionTypedSignature(_ typeResolver, sym model.FunctionSymbol, sig model.TypedFunctionSignature) {
	sym.SetTypedSignature(sig)
}

func setDependentFunctionSignature(
	_ typeResolver,
	sym model.DependentlyTypedFunctionSymbol,
	paramTypes []semtypes.SemType,
	returnType model.TypeOp,
) {
	sym.SetParamTypes(paramTypes)
	sym.SetReturnType(returnType)
}
