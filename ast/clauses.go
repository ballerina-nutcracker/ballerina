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
	"github.com/ballerina-nutcracker/ballerina/common"
)

type TypeParamEntry struct {
	TypeParam BType
	BoundType BType
}

type (
	bLangInputClauseBase[C BLangActionOrExpression] struct {
		bLangNodeBase
		Collection             C
		VariableDefinitionNode *BLangVariableDef
		IsDeclaredWithVarFlag  bool
	}
	BLangFromClause struct {
		// Match jBallerina, which accepts actions as from-clause collections.
		bLangInputClauseBase[BLangActionOrExpression]
	}
	BLangJoinClause struct {
		bLangInputClauseBase[BLangExpression]
		OnClause        BLangOnClause
		IsOuterJoinFlag bool
	}
	BLangLetClause struct {
		bLangNodeBase
		LetVarDeclarations []BLangVariableDef
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
		GroupingKeyList []*BLangGroupingKey
		NonGroupingKeys common.Set[string]
	}
	BLangGroupingKey struct {
		bLangNodeBase
		VariableDef *BLangVariableDef
		VariableRef *BLangVarRef
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
		// Match jBallerina, which also accepts actions in select clauses.
		Expression BLangActionOrExpression
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
		VariableDefinitionNode *BLangVariableDef
		VarType                BType
		BodyContainsFail       bool
		IsInternal             bool
		isDeclaredWithVar      bool
	}
)

var (
	_ FromClauseNode    = &BLangFromClause{}
	_ Node              = &BLangLetClause{}
	_ Node              = &BLangWhereClause{}
	_ Node              = &BLangLimitClause{}
	_ Node              = &BLangOrderByClause{}
	_ Node              = &BLangOrderKey{}
	_ SelectClauseNode  = &BLangSelectClause{}
	_ Node              = &BLangOnConflictClause{}
	_ CollectClauseNode = &BLangCollectClause{}
	_ DoClauseNode      = &BLangDoClause{}
)

func NewBLangFromClause(pos Location, collection BLangActionOrExpression, variableDefinition *BLangVariableDef, declaredWithVar bool) *BLangFromClause {
	return &BLangFromClause{bLangInputClauseBase: bLangInputClauseBase[BLangActionOrExpression]{
		bLangNodeBase:          bLangNodeBase{pos: pos},
		Collection:             collection,
		VariableDefinitionNode: variableDefinition,
		IsDeclaredWithVarFlag:  declaredWithVar,
	}}
}

func NewBLangJoinClause(pos Location, collection BLangExpression, variableDefinition *BLangVariableDef, declaredWithVar, outer bool, onClause *BLangOnClause) *BLangJoinClause {
	clause := &BLangJoinClause{
		bLangInputClauseBase: bLangInputClauseBase[BLangExpression]{
			bLangNodeBase:          bLangNodeBase{pos: pos},
			Collection:             collection,
			VariableDefinitionNode: variableDefinition,
			IsDeclaredWithVarFlag:  declaredWithVar,
		},
		IsOuterJoinFlag: outer,
	}
	if onClause != nil {
		clause.OnClause = *onClause
	}
	return clause
}

func NewBLangOnClause(pos Location, onExpr, equalsExpr BLangExpression) *BLangOnClause {
	return &BLangOnClause{bLangNodeBase: bLangNodeBase{pos: pos}, OnExpr: onExpr, EqualsExpr: equalsExpr}
}

func NewBLangLimitClause(pos Location, expr BLangExpression) *BLangLimitClause {
	return &BLangLimitClause{bLangNodeBase: bLangNodeBase{pos: pos}, Expression: expr}
}

func NewBLangSelectClause(pos Location, expr BLangActionOrExpression) *BLangSelectClause {
	return &BLangSelectClause{bLangNodeBase: bLangNodeBase{pos: pos}, Expression: expr}
}

func NewBLangOnConflictClause(pos Location, expr BLangExpression) *BLangOnConflictClause {
	return &BLangOnConflictClause{bLangNodeBase: bLangNodeBase{pos: pos}, Expression: expr}
}

func NewBLangCollectClause(pos Location, expr BLangExpression, nonGroupingKeys common.Set[string]) *BLangCollectClause {
	return &BLangCollectClause{bLangNodeBase: bLangNodeBase{pos: pos}, Expression: expr, NonGroupingKeys: nonGroupingKeys}
}

func NewBLangGroupingKeyWithVariableDef(pos Location, variableDef *BLangVariableDef) *BLangGroupingKey {
	return &BLangGroupingKey{bLangNodeBase: bLangNodeBase{pos: pos}, VariableDef: variableDef}
}

func NewBLangGroupingKeyWithVariableRef(pos Location, variableRef *BLangVarRef) *BLangGroupingKey {
	return &BLangGroupingKey{bLangNodeBase: bLangNodeBase{pos: pos}, VariableRef: variableRef}
}

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

func (b *bLangInputClauseBase[C]) GetCollection() C {
	return b.Collection
}

func (b *bLangInputClauseBase[C]) GetVariableDefinitionNode() *BLangVariableDef {
	if b.VariableDefinitionNode == nil {
		return nil
	}
	return b.VariableDefinitionNode
}

func (b *bLangInputClauseBase[C]) IsDeclaredWithVar() bool {
	return b.IsDeclaredWithVarFlag
}

func (b *BLangJoinClause) GetOnClause() *BLangOnClause {
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

func (b *BLangOnClause) GetEqualsExpression() BLangExpression {
	return b.EqualsExpr
}

func (b *BLangGroupByClause) AddGroupingKey(groupingKey *BLangGroupingKey) {
	b.GroupingKeyList = append(b.GroupingKeyList, groupingKey)
}

func (b *BLangGroupByClause) GetGroupingKeyList() []*BLangGroupingKey {
	return b.GroupingKeyList
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

func (b *BLangLimitClause) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangSelectClause) GetExpression() BLangActionOrExpression {
	return b.Expression
}

func (b *BLangOnConflictClause) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangCollectClause) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangDoClause) GetBody() *BLangBlockStmt {
	return b.Body
}

func (b *BLangOnFailClause) IsDeclaredWithVar() bool {
	return b.isDeclaredWithVar
}

func (b *BLangOnFailClause) GetVariableDefinitionNode() *BLangVariableDef {
	if b.VariableDefinitionNode == nil {
		return nil
	}
	return b.VariableDefinitionNode
}

func (b *BLangOnFailClause) GetBody() *BLangBlockStmt {
	return b.Body
}
