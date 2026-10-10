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

import "github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

type BadTopLevelNodeKind uint8

const (
	BadTopLevelNodeUnknown BadTopLevelNodeKind = iota
	BadTopLevelNodeImport
	BadTopLevelNodeFunction
	BadTopLevelNodeTypeDefinition
	BadTopLevelNodeEnum
	BadTopLevelNodeConstant
	BadTopLevelNodeVariable
	BadTopLevelNodeListener
	BadTopLevelNodeClass
	BadTopLevelNodeService
	BadTopLevelNodeAnnotation
	BadTopLevelNodeXMLNamespace
	BadTopLevelNodeField
	BadTopLevelNodeTypeInclusion
)

func (kind BadTopLevelNodeKind) String() string {
	switch kind {
	case BadTopLevelNodeImport:
		return "import"
	case BadTopLevelNodeFunction:
		return "function"
	case BadTopLevelNodeTypeDefinition:
		return "type-definition"
	case BadTopLevelNodeEnum:
		return "enum"
	case BadTopLevelNodeConstant:
		return "constant"
	case BadTopLevelNodeVariable:
		return "variable"
	case BadTopLevelNodeListener:
		return "listener"
	case BadTopLevelNodeClass:
		return "class"
	case BadTopLevelNodeService:
		return "service"
	case BadTopLevelNodeAnnotation:
		return "annotation"
	case BadTopLevelNodeXMLNamespace:
		return "xml-namespace"
	case BadTopLevelNodeField:
		return "field"
	case BadTopLevelNodeTypeInclusion:
		return "type-inclusion"
	default:
		return "unknown"
	}
}

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
	recoveredKind BadTopLevelNodeKind
}

func NewBLangBadTopLevelNode(pos diagnostics.Location, kind BadTopLevelNodeKind) *BLangBadTopLevelNode {
	return &BLangBadTopLevelNode{
		bLangBadNodeBase: bLangBadNodeBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		recoveredKind:    kind,
	}
}

func (node *BLangBadTopLevelNode) GetRecoveredKind() BadTopLevelNodeKind {
	return node.recoveredKind
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

func (*BLangBadTopLevelNode) isTopLevel() {}

func (*BLangBadStmt) isStatement() {}

func (*BLangBadExprOrAction) actionOrExpression() {}
func (*BLangBadExprOrAction) expressionNode()     {}
func (*BLangBadExprOrAction) actionNode()         {}
func (*BLangBadExprOrAction) isLExpr()            {}

func (*BLangBadTypeNode) badNode() {}
