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

// Every AST write performed during type resolution goes through these helpers. Each of them does nothing while the
// resolver is ephemeral, so that a candidate trial (see selectCandidate) resolves expressions without committing
// anything.

type symbolNode interface {
	ast.BLangNode
	SetSymbol(model.SymbolRef)
}

type methodSymbolNode interface {
	ast.BLangNode
	SetMethodSymbol(model.SymbolRef)
}

func setNodeType(t typeResolver, node ast.BLangNode, ty semtypes.SemType) {
	if t.isEphemeral() {
		return
	}
	node.SetDeterminedType(ty)
}

type typeDataNode interface {
	SetTypeData(ast.TypeData)
}

func setTypeData(t typeResolver, node typeDataNode, data ast.TypeData) {
	if t.isEphemeral() {
		return
	}
	node.SetTypeData(data)
}

func setTypeDataType(t typeResolver, data *ast.TypeData, ty semtypes.SemType) {
	if t.isEphemeral() {
		return
	}
	data.Type = ty
}

func setLiteralValue(t typeResolver, node *ast.BLangLiteral, value any) {
	if t.isEphemeral() {
		return
	}
	node.SetValue(value)
}

func setLiteralSymbolType(t typeResolver, node *ast.BLangLiteral, ty semtypes.SemType) {
	if t.isEphemeral() {
		return
	}
	updateSymbolType(t, node, ty)
}

func setNodeSymbol(t typeResolver, node symbolNode, ref model.SymbolRef) {
	if t.isEphemeral() {
		return
	}
	node.SetSymbol(ref)
}

func setMethodSymbol(t typeResolver, node methodSymbolNode, ref model.SymbolRef) {
	if t.isEphemeral() {
		return
	}
	node.SetMethodSymbol(ref)
}

func setInvocationResolvedSymbol(t typeResolver, inv invocable, ref model.SymbolRef) {
	if t.isEphemeral() {
		return
	}
	inv.SetResolvedSymbol(ref)
}

func setInvocationCallArgs(t typeResolver, inv invocable, args []ast.BLangExpression) {
	if t.isEphemeral() {
		return
	}
	inv.SetCallArgs(args)
}

func moveLangLibReceiver(t typeResolver, expr *ast.BLangInvocation, args []ast.BLangExpression, pkgAlias ast.BLangIdentifier) {
	if t.isEphemeral() {
		return
	}
	expr.ArgExprs = args
	expr.Expr = nil
	expr.PkgAlias = &pkgAlias
}

func clearStreamOperationSymbol(t typeResolver, expr *ast.BLangInvocation) {
	if t.isEphemeral() {
		return
	}
	expr.RawSymbol = nil
}

func setLaxAccess(t typeResolver, expr *ast.BLangFieldBaseAccess) {
	if t.isEphemeral() {
		return
	}
	expr.SetLax()
}

func setTypedescConstraint(t typeResolver, node *ast.BLangTypedescExpr, constraint semtypes.SemType) {
	if t.isEphemeral() {
		return
	}
	node.Constraint = constraint
}

func setNewExpressionResult(t typeResolver, node *ast.BLangNewExpression, args []ast.BLangExpression, classRef model.SymbolRef) {
	if t.isEphemeral() {
		return
	}
	node.ArgsExprs = args
	node.ClassSymbol = classRef
}

// setMappingKey commits the type of a key whose expression carries a symbol (a `{x}` shorthand key) and the types
// of the key nodes. A literal key is resolved by the caller.
func setMappingKey(t typeResolver, kv *ast.BLangMappingKeyValueField, valueTy semtypes.SemType) {
	if t.isEphemeral() {
		return
	}
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

func setMappingConstructorInherentType(t typeResolver, node *ast.BLangMappingConstructorExpr, atom semtypes.MappingAtomicType, defaults []model.FieldDefault) {
	if t.isEphemeral() {
		return
	}
	node.AtomicType = atom
	node.FieldDefaults = defaults
}

func setListConstructorInherentType(t typeResolver, node *ast.BLangListConstructorExpr, atom semtypes.ListAtomicType, spreadMembers []bool) {
	if t.isEphemeral() {
		return
	}
	node.AtomicType = atom
	node.SpreadMembers = nil
	for _, isSpread := range spreadMembers {
		if isSpread {
			node.SpreadMembers = spreadMembers
			break
		}
	}
}

func setGroupByNonGroupingKeys(t typeResolver, clause *ast.BLangGroupByClause, keys balCommon.Set[string]) {
	if t.isEphemeral() {
		return
	}
	clause.NonGroupingKeys = keys
}

func setFunctionTypedSignature(t typeResolver, sym model.FunctionSymbol, sig model.TypedFunctionSignature) {
	if t.isEphemeral() {
		return
	}
	sym.SetTypedSignature(sig)
}

func setLambdaSymbolType(t typeResolver, fn *ast.BLangFunction, ty semtypes.SemType) {
	if t.isEphemeral() {
		return
	}
	updateSymbolType(t, fn, ty)
}

func setXMLStepLowering(t typeResolver, expr *ast.BLangXMLStepExpression, lowered ast.BLangExpression) {
	if t.isEphemeral() {
		return
	}
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

func setTypeDefinition(t typeResolver, field *semtypes.Definition, defn semtypes.Definition) {
	if t.isEphemeral() {
		return
	}
	*field = defn
}

// setOtherNodesAsNever sets the type of every AST node whose determined type is not set to NEVER.
func setOtherNodesAsNever(t typeResolver, node ast.BLangNode) {
	if t.isEphemeral() {
		return
	}
	ast.Walk(neverVisitor{t: t}, node)
}
