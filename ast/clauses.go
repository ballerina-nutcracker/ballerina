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

package ast

import (
	"ballerina/common"
)

type TypeParamEntry struct {
	TypeParam BType
	BoundType BType
}

type (
	BLangInputClause struct {
		bLangNodeBase
		Collection BLangExpression
		// PR-TODO: can this be nil?
		VariableDefinitionNode *BLangSimpleVariableDef
		IsDeclaredWithVarFlag  bool
	}
	BLangFromClause struct {
		BLangInputClause
	}
	BLangJoinClause struct {
		BLangInputClause
		OnClause        BLangOnClause
		IsOuterJoinFlag bool
	}
	BLangLetClause struct {
		bLangNodeBase
		LetVarDeclarations []BLangSimpleVariableDef
	}
	BLangOnClause struct {
		bLangNodeBase
		OnExpr     BLangExpression
		EqualsExpr BLangExpression
	}
	BLangWhereClause struct {
		bLangNodeBase
		Expression BLangExpression
	}
	BLangGroupByClause struct {
		bLangNodeBase
		GroupingKeyList []BLangGroupingKey
		NonGroupingKeys common.Set[string]
	}
	BLangGroupingKey struct {
		bLangNodeBase
		VariableDef *BLangSimpleVariableDef
		VariableRef *BLangSimpleVarRef
	}
	BLangLimitClause struct {
		bLangNodeBase
		Expression BLangExpression
	}
	BLangOrderByClause struct {
		bLangNodeBase
		OrderByKeyList []BLangOrderKey
	}
	BLangOrderKey struct {
		bLangNodeBase
		Expression   BLangExpression
		IsDescending bool
	}
	BLangSelectClause struct {
		bLangNodeBase
		Expression BLangExpression
	}
	BLangOnConflictClause struct {
		bLangNodeBase
		Expression BLangExpression
	}
	BLangCollectClause struct {
		bLangNodeBase
		Expression      BLangExpression
		NonGroupingKeys common.Set[string]
	}
	BLangDoClause struct {
		bLangNodeBase
		Body *BLangBlockStmt
	}
	BLangOnFailClause struct {
		bLangNodeBase
		Body                   *BLangBlockStmt
		VariableDefinitionNode *BLangSimpleVariableDef
		VarType                BType
		BodyContainsFail       bool
		IsInternal             bool
		isDeclaredWithVar      bool
	}
)

var (
	_ FromClauseNode    = &BLangFromClause{}
	_ JoinClauseNode    = &BLangJoinClause{}
	_ Node              = &BLangLetClause{}
	_ OnClauseNode      = &BLangOnClause{}
	_ Node              = &BLangWhereClause{}
	_ GroupByClauseNode = &BLangGroupByClause{}
	_ GroupingKeyNode   = &BLangGroupingKey{}
	_ Node              = &BLangLimitClause{}
	_ Node              = &BLangOrderByClause{}
	_ Node              = &BLangOrderKey{}
	_ SelectClauseNode  = &BLangSelectClause{}
	_ Node              = &BLangOnConflictClause{}
	_ CollectClauseNode = &BLangCollectClause{}
	_ DoClauseNode      = &BLangDoClause{}
	_ OnFailClauseNode  = &BLangOnFailClause{}
)

var (
	_ BLangNode = &BLangFromClause{}
	_ BLangNode = &BLangJoinClause{}
	_ BLangNode = &BLangLetClause{}
	_ BLangNode = &BLangOnClause{}
	_ BLangNode = &BLangWhereClause{}
	_ BLangNode = &BLangGroupByClause{}
	_ BLangNode = &BLangGroupingKey{}
	_ BLangNode = &BLangLimitClause{}
	_ BLangNode = &BLangOrderByClause{}
	_ BLangNode = &BLangOrderKey{}
	_ BLangNode = &BLangSelectClause{}
	_ BLangNode = &BLangOnConflictClause{}
	_ BLangNode = &BLangCollectClause{}
	_ BLangNode = &BLangDoClause{}
	_ BLangNode = &BLangOnFailClause{}
)

func (b *BLangJoinClause) GetCollection() BLangExpression {
	return b.Collection
}

func (b *BLangJoinClause) SetCollection(collection BLangExpression) {
	b.Collection = collection
}

func (b *BLangJoinClause) GetVariableDefinitionNode() VariableDefinitionNode {
	if b.VariableDefinitionNode == nil {
		return nil
	}
	return b.VariableDefinitionNode
}

func (b *BLangJoinClause) SetVariableDefinitionNode(variableDefinitionNode VariableDefinitionNode) {
	if variableDefinitionNode == nil {
		b.VariableDefinitionNode = nil
		return
	}
	b.VariableDefinitionNode = variableDefinitionNode.(*BLangSimpleVariableDef)
}

func (b *BLangJoinClause) IsDeclaredWithVar() bool {
	return b.IsDeclaredWithVarFlag
}

func (b *BLangJoinClause) GetOnClause() OnClauseNode {
	if b.OnClause.OnExpr == nil && b.OnClause.EqualsExpr == nil {
		return nil
	}
	return &b.OnClause
}

func (b *BLangJoinClause) IsOuterJoin() bool {
	return b.IsOuterJoinFlag
}

func (b *BLangOnClause) GetOnExpression() BLangExpression {
	return b.OnExpr
}

func (b *BLangOnClause) SetOnExpression(expression BLangExpression) {
	b.OnExpr = expression
}

func (b *BLangOnClause) GetEqualsExpression() BLangExpression {
	return b.EqualsExpr
}

func (b *BLangOnClause) SetEqualsExpression(expression BLangExpression) {
	b.EqualsExpr = expression
}

func (b *BLangFromClause) GetCollection() BLangExpression {
	return b.Collection
}

func (b *BLangFromClause) SetCollection(collection BLangExpression) {
	b.Collection = collection
}

func (b *BLangFromClause) GetVariableDefinitionNode() VariableDefinitionNode {
	if b.VariableDefinitionNode == nil {
		return nil
	}
	return b.VariableDefinitionNode
}

func (b *BLangFromClause) SetVariableDefinitionNode(variableDefinitionNode VariableDefinitionNode) {
	if variableDefinitionNode == nil {
		b.VariableDefinitionNode = nil
		return
	}
	b.VariableDefinitionNode = variableDefinitionNode.(*BLangSimpleVariableDef)
}

func (b *BLangFromClause) IsDeclaredWithVar() bool {
	return b.IsDeclaredWithVarFlag
}

func (b *BLangLetClause) GetKind() NodeKind {
	return NodeKind_LET_CLAUSE
}

func (b *BLangWhereClause) GetKind() NodeKind {
	return NodeKind_WHERE
}

func (b *BLangGroupByClause) GetKind() NodeKind {
	return NodeKind_GROUP_BY
}

func (b *BLangGroupByClause) AddGroupingKey(groupingKey GroupingKeyNode) {
	if key, ok := groupingKey.(*BLangGroupingKey); ok {
		b.GroupingKeyList = append(b.GroupingKeyList, *key)
		return
	}
	panic("groupingKey is not a BLangGroupingKey")
}

func (b *BLangGroupByClause) GetGroupingKeyList() []GroupingKeyNode {
	result := make([]GroupingKeyNode, len(b.GroupingKeyList))
	for i := range b.GroupingKeyList {
		result[i] = &b.GroupingKeyList[i]
	}
	return result
}

func (b *BLangGroupingKey) GetKind() NodeKind {
	return NodeKind_GROUPING_KEY
}

func (b *BLangGroupingKey) SetGroupingKey(groupingKey Node) {
	if key, ok := groupingKey.(*BLangSimpleVariableDef); ok {
		b.VariableDef = key
		b.VariableRef = nil
		return
	}
	key := groupingKey.(*BLangSimpleVarRef)
	b.VariableRef = key
	b.VariableDef = nil
}

func (b *BLangGroupingKey) GetGroupingKey() Node {
	if b.VariableRef != nil {
		return b.VariableRef
	}
	if b.VariableDef != nil {
		return b.VariableDef
	}
	return nil
}

func (b *BLangLimitClause) GetKind() NodeKind {
	return NodeKind_LIMIT
}

func (b *BLangLimitClause) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangLimitClause) SetExpression(expression BLangExpression) {
	b.Expression = expression
}

func (b *BLangSelectClause) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangSelectClause) SetExpression(expression BLangExpression) {
	b.Expression = expression
}

func (b *BLangOnConflictClause) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangOnConflictClause) SetExpression(expression BLangExpression) {
	b.Expression = expression
}

func (b *BLangCollectClause) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangCollectClause) SetExpression(expression BLangExpression) {
	b.Expression = expression
}

func (b *BLangDoClause) GetBody() BlockStatementNode {
	return b.Body
}

func (b *BLangDoClause) SetBody(body BlockStatementNode) {
	if body, ok := body.(*BLangBlockStmt); ok {
		b.Body = body
		return
	}
	panic("body is not a BLangBlockStmt")
}

func (b *BLangOnFailClause) SetDeclaredWithVar() {
	b.isDeclaredWithVar = true
}

func (b *BLangOnFailClause) IsDeclaredWithVar() bool {
	return b.isDeclaredWithVar
}

func (b *BLangOnFailClause) GetVariableDefinitionNode() VariableDefinitionNode {
	if b.VariableDefinitionNode == nil {
		return nil
	}
	return b.VariableDefinitionNode
}

func (b *BLangOnFailClause) SetVariableDefinitionNode(variableDefinitionNode VariableDefinitionNode) {
	if variableDefinitionNode == nil {
		b.VariableDefinitionNode = nil
		return
	}
	b.VariableDefinitionNode = variableDefinitionNode.(*BLangSimpleVariableDef)
}

func (b *BLangOnFailClause) GetBody() BlockStatementNode {
	return b.Body
}

func (b *BLangOnFailClause) SetBody(body BlockStatementNode) {
	if body, ok := body.(*BLangBlockStmt); ok {
		b.Body = body
		return
	}
	panic("body is not a BLangBlockStmt")
}
