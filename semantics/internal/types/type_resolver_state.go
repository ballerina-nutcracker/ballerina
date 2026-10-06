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
)

// Every AST write performed during type resolution goes through these helpers, so that a candidate trial (see
// selectCandidate) can resolve expressions without committing anything.

type symbolNode interface {
	ast.BLangNode
	SetSymbol(model.SymbolRef)
}

type methodSymbolNode interface {
	ast.BLangNode
	SetMethodSymbol(model.SymbolRef)
}

func setNodeType(_ typeResolver, node ast.BLangNode, ty semtypes.SemType) {
	node.SetDeterminedType(ty)
}

type typeDataNode interface {
	SetTypeData(ast.TypeData)
}

func setTypeData(_ typeResolver, node typeDataNode, data ast.TypeData) {
	node.SetTypeData(data)
}

func setTypeDataType(_ typeResolver, data *ast.TypeData, ty semtypes.SemType) {
	data.Type = ty
}

func setLiteralValue(_ typeResolver, node *ast.BLangLiteral, value any) {
	node.SetValue(value)
}

func setLiteralSymbolType(t typeResolver, node *ast.BLangLiteral, ty semtypes.SemType) {
	updateSymbolType(t, node, ty)
}

func setNodeSymbol(_ typeResolver, node symbolNode, ref model.SymbolRef) {
	node.SetSymbol(ref)
}

func setMethodSymbol(_ typeResolver, node methodSymbolNode, ref model.SymbolRef) {
	node.SetMethodSymbol(ref)
}

func setInvocationResolvedSymbol(_ typeResolver, inv invocable, ref model.SymbolRef) {
	inv.SetResolvedSymbol(ref)
}

func setInvocationCallArgs(_ typeResolver, inv invocable, args []ast.BLangExpression) {
	inv.SetCallArgs(args)
}

func moveLangLibReceiver(_ typeResolver, expr *ast.BLangInvocation, args []ast.BLangExpression, pkgAlias ast.BLangIdentifier) {
	expr.ArgExprs = args
	expr.Expr = nil
	expr.PkgAlias = &pkgAlias
}

func clearStreamOperationSymbol(_ typeResolver, expr *ast.BLangInvocation) {
	expr.RawSymbol = nil
}

func setLaxAccess(_ typeResolver, expr *ast.BLangFieldBaseAccess) {
	expr.SetLax()
}

func setTypedescConstraint(_ typeResolver, node *ast.BLangTypedescExpr, constraint semtypes.SemType) {
	node.Constraint = constraint
}

func setNewExpressionResult(_ typeResolver, node *ast.BLangNewExpression, args []ast.BLangExpression, classRef model.SymbolRef) {
	node.ArgsExprs = args
	node.ClassSymbol = classRef
}

// setMappingKey commits the type of a key whose expression carries a symbol (a `{x}` shorthand key) and the types
// of the key nodes. A literal key is resolved by the caller.
func setMappingKey(t typeResolver, kv *ast.BLangMappingKeyValueField, valueTy semtypes.SemType) {
	switch keyExpr := kv.Key.Expr.(type) {
	case *ast.BLangLiteral:
	case ast.BNodeWithSymbol:
		t.setSymbolType(keyExpr.Symbol(), valueTy)
		if e, ok := keyExpr.(ast.BLangExpression); ok {
			e.SetDeterminedType(valueTy)
		}
		if ref, ok := keyExpr.(*ast.BLangVarRef); ok {
			setVarRefIdentifierTypes(t, ref)
		}
	}
	kv.Key.SetDeterminedType(semtypes.Never)
	kv.SetDeterminedType(semtypes.Never)
}

func setMappingConstructorInherentType(_ typeResolver, node *ast.BLangMappingConstructorExpr, atom semtypes.MappingAtomicType, defaults []model.FieldDefault) {
	node.AtomicType = atom
	node.FieldDefaults = defaults
}

func setListConstructorInherentType(_ typeResolver, node *ast.BLangListConstructorExpr, atom semtypes.ListAtomicType, spreadMembers []bool) {
	node.AtomicType = atom
	node.SpreadMembers = nil
	for _, isSpread := range spreadMembers {
		if isSpread {
			node.SpreadMembers = spreadMembers
			break
		}
	}
}

func setGroupByNonGroupingKeys(_ typeResolver, clause *ast.BLangGroupByClause, keys balCommon.Set[string]) {
	clause.NonGroupingKeys = keys
}

func setFunctionTypedSignature(_ typeResolver, sym model.FunctionSymbol, sig model.TypedFunctionSignature) {
	sym.SetTypedSignature(sig)
}

func setLambdaSymbolType(t typeResolver, fn *ast.BLangFunction, ty semtypes.SemType) {
	updateSymbolType(t, fn, ty)
}

func setXMLStepLowering(_ typeResolver, expr *ast.BLangXMLStepExpression, lowered ast.BLangExpression) {
	expr.LoweredExpression = lowered
}

type neverVisitor struct {
	t typeResolver
}

func (v neverVisitor) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if semtypes.IsZero(node.GetDeterminedType()) {
		setNodeType(v.t, node, semtypes.Never)
	}
	return v
}

func (v neverVisitor) VisitTypeData(_ *ast.TypeData) ast.Visitor {
	return v
}

// setOtherNodesAsNever sets the type of every AST node whose determined type is not set to NEVER.
func setOtherNodesAsNever(t typeResolver, node ast.BLangNode) {
	ast.Walk(neverVisitor{t: t}, node)
}
