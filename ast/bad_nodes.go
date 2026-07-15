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

type BLangBadNode interface {
	BLangNode
	badNode()
}

type bLangBadNodeBase struct {
	bLangNodeBase
}

func (*bLangBadNodeBase) badNode() {}

type BLangBadTopLevelNode struct {
	bLangBadNodeBase
}

type BLangBadStmt struct {
	bLangBadNodeBase
}

type BLangBadExprOrAction struct {
	bLangBadNodeBase
}

type BLangBadTypeNode struct {
	bLangTypeBase
}

type BLangBadIdentifier struct {
	bLangBadNodeBase
	Value         string
	OriginalValue string
	isLiteral     bool
}

func (*BLangBadTopLevelNode) isTopLevel() {}

func (*BLangBadStmt) isStatement() {}

func (*BLangBadExprOrAction) actionOrExpression() {}
func (*BLangBadExprOrAction) expressionNode()     {}
func (*BLangBadExprOrAction) actionNode()         {}
func (*BLangBadExprOrAction) isLExpr()            {}

func (*BLangBadTypeNode) badNode() {}

func (b *BLangBadIdentifier) GetValue() string { return b.Value }
func (b *BLangBadIdentifier) IsLiteral() bool  { return b.isLiteral }
