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

import "github.com/ballerina-nutcracker/ballerina/semtypes"

type BadTopLevelNodeKind uint8

const (
	BadTopLevelNodeMissing BadTopLevelNodeKind = iota
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
	BadTopLevelNodeTrailingTokens
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
	case BadTopLevelNodeTrailingTokens:
		return "trailing-tokens"
	default:
		return "missing"
	}
}

type BadStmtKind uint8

const (
	BadStmtMissing BadStmtKind = iota
	BadStmtAssignment
	BadStmtCompoundAssignment
	BadStmtVariableDeclaration
	BadStmtBlock
	BadStmtBreak
	BadStmtFail
	BadStmtExpression
	BadStmtContinue
	BadStmtIfElse
	BadStmtWhile
	BadStmtPanic
	BadStmtReturn
	BadStmtLocalTypeDefinition
	BadStmtLock
	BadStmtFork
	BadStmtForEach
	BadStmtXMLNamespace
	BadStmtTransaction
	BadStmtRollback
	BadStmtRetry
	BadStmtMatch
	BadStmtDo
)

func (kind BadStmtKind) String() string {
	switch kind {
	case BadStmtAssignment:
		return "assignment"
	case BadStmtCompoundAssignment:
		return "compound-assignment"
	case BadStmtVariableDeclaration:
		return "variable-declaration"
	case BadStmtBlock:
		return "block"
	case BadStmtBreak:
		return "break"
	case BadStmtFail:
		return "fail"
	case BadStmtExpression:
		return "expression"
	case BadStmtContinue:
		return "continue"
	case BadStmtIfElse:
		return "if-else"
	case BadStmtWhile:
		return "while"
	case BadStmtPanic:
		return "panic"
	case BadStmtReturn:
		return "return"
	case BadStmtLocalTypeDefinition:
		return "local-type-definition"
	case BadStmtLock:
		return "lock"
	case BadStmtFork:
		return "fork"
	case BadStmtForEach:
		return "foreach"
	case BadStmtXMLNamespace:
		return "xml-namespace"
	case BadStmtTransaction:
		return "transaction"
	case BadStmtRollback:
		return "rollback"
	case BadStmtRetry:
		return "retry"
	case BadStmtMatch:
		return "match"
	case BadStmtDo:
		return "do"
	default:
		return "missing"
	}
}

type BadExprKind uint8

const (
	BadExprMissing BadExprKind = iota
	BadExprRemoteMethodCallAction
	BadExprClientResourceAccessAction
	BadExprStartAction
	BadExprWaitAction
	BadExprQueryAction
	BadExprAsyncSendAction
	BadExprSyncSendAction
	BadExprReceiveAction
	BadExprCommitAction
	BadExprFlushAction
	BadExprBracedAction
	BadExprSimpleLiteral
	BadExprVarRef
	BadExprTypedesc
	BadExprBraced
	BadExprBinary
	BadExprCheck
	BadExprFieldAccess
	BadExprFunctionCall
	BadExprMethodCall
	BadExprMappingConstructor
	BadExprIndexed
	BadExprTypeof
	BadExprUnary
	BadExprInferredTypedescDefault
	BadExprObjectConstructor
	BadExprTypeTest
	BadExprTrap
	BadExprListConstructor
	BadExprTypeCast
	BadExprTableConstructor
	BadExprLet
	BadExprTemplate
	BadExprExplicitAnonymousFunction
	BadExprImplicitAnonymousFunction
	BadExprExplicitNew
	BadExprImplicitNew
	BadExprQuery
	BadExprAnnotAccess
	BadExprOptionalFieldAccess
	BadExprConditional
	BadExprTransactional
	BadExprByteArrayLiteral
	BadExprXMLFilter
	BadExprXMLStep
	BadExprRequired
	BadExprErrorConstructor
	BadExprNatural
	BadExprPositionalArgument
	BadExprNamedArgument
	BadExprRestArgument
	BadExprWildcardBindingPattern
	BadExprBasicLiteral
)

func (kind BadExprKind) String() string {
	switch kind {
	case BadExprRemoteMethodCallAction:
		return "remote-method-call-action"
	case BadExprClientResourceAccessAction:
		return "client-resource-access-action"
	case BadExprStartAction:
		return "start-action"
	case BadExprWaitAction:
		return "wait-action"
	case BadExprQueryAction:
		return "query-action"
	case BadExprAsyncSendAction:
		return "async-send-action"
	case BadExprSyncSendAction:
		return "sync-send-action"
	case BadExprReceiveAction:
		return "receive-action"
	case BadExprCommitAction:
		return "commit-action"
	case BadExprFlushAction:
		return "flush-action"
	case BadExprBracedAction:
		return "braced-action"
	case BadExprSimpleLiteral:
		return "simple-literal"
	case BadExprVarRef:
		return "var-ref"
	case BadExprTypedesc:
		return "typedesc"
	case BadExprBraced:
		return "braced"
	case BadExprBinary:
		return "binary"
	case BadExprCheck:
		return "check"
	case BadExprFieldAccess:
		return "field-access"
	case BadExprFunctionCall:
		return "function-call"
	case BadExprMethodCall:
		return "method-call"
	case BadExprMappingConstructor:
		return "mapping-constructor"
	case BadExprIndexed:
		return "indexed"
	case BadExprTypeof:
		return "typeof"
	case BadExprUnary:
		return "unary"
	case BadExprInferredTypedescDefault:
		return "inferred-typedesc-default"
	case BadExprObjectConstructor:
		return "object-constructor"
	case BadExprTypeTest:
		return "type-test"
	case BadExprTrap:
		return "trap"
	case BadExprListConstructor:
		return "list-constructor"
	case BadExprTypeCast:
		return "type-cast"
	case BadExprTableConstructor:
		return "table-constructor"
	case BadExprLet:
		return "let"
	case BadExprTemplate:
		return "template"
	case BadExprExplicitAnonymousFunction:
		return "explicit-anonymous-function"
	case BadExprImplicitAnonymousFunction:
		return "implicit-anonymous-function"
	case BadExprExplicitNew:
		return "explicit-new"
	case BadExprImplicitNew:
		return "implicit-new"
	case BadExprQuery:
		return "query"
	case BadExprAnnotAccess:
		return "annot-access"
	case BadExprOptionalFieldAccess:
		return "optional-field-access"
	case BadExprConditional:
		return "conditional"
	case BadExprTransactional:
		return "transactional"
	case BadExprByteArrayLiteral:
		return "byte-array-literal"
	case BadExprXMLFilter:
		return "xml-filter"
	case BadExprXMLStep:
		return "xml-step"
	case BadExprRequired:
		return "required"
	case BadExprErrorConstructor:
		return "error-constructor"
	case BadExprNatural:
		return "natural"
	case BadExprPositionalArgument:
		return "positional-argument"
	case BadExprNamedArgument:
		return "named-argument"
	case BadExprRestArgument:
		return "rest-argument"
	case BadExprWildcardBindingPattern:
		return "wildcard-binding-pattern"
	case BadExprBasicLiteral:
		return "basic-literal"
	default:
		return "missing"
	}
}

type BadTypeKind uint8

const (
	BadTypeMissing BadTypeKind = iota
	BadTypeBuiltin
	BadTypeNil
	BadTypeQualifiedReference
	BadTypeSimpleReference
	BadTypeObject
	BadTypeRecord
	BadTypeOptional
	BadTypeMap
	BadTypeUnion
	BadTypeStream
	BadTypeTable
	BadTypeFunction
	BadTypeTuple
	BadTypeParenthesised
	BadTypeIntersection
	BadTypeSingleton
	BadTypeArray
	BadTypeTypeReference
	BadTypeDistinct
	BadTypeParameterized
)

func (kind BadTypeKind) String() string {
	switch kind {
	case BadTypeBuiltin:
		return "builtin"
	case BadTypeNil:
		return "nil"
	case BadTypeQualifiedReference:
		return "qualified-reference"
	case BadTypeSimpleReference:
		return "simple-reference"
	case BadTypeObject:
		return "object"
	case BadTypeRecord:
		return "record"
	case BadTypeOptional:
		return "optional"
	case BadTypeMap:
		return "map"
	case BadTypeUnion:
		return "union"
	case BadTypeStream:
		return "stream"
	case BadTypeTable:
		return "table"
	case BadTypeFunction:
		return "function"
	case BadTypeTuple:
		return "tuple"
	case BadTypeParenthesised:
		return "parenthesised"
	case BadTypeIntersection:
		return "intersection"
	case BadTypeSingleton:
		return "singleton"
	case BadTypeArray:
		return "array"
	case BadTypeTypeReference:
		return "type-reference"
	case BadTypeDistinct:
		return "distinct"
	case BadTypeParameterized:
		return "parameterized"
	default:
		return "missing"
	}
}

// RecoveredToken is a keyword or punctuation token found in the source span of a bad node.
type RecoveredToken struct {
	pos  Location
	text string
}

func NewRecoveredToken(pos Location, text string) RecoveredToken {
	return RecoveredToken{pos: pos, text: text}
}

func (t RecoveredToken) GetPosition() Location { return t.pos }
func (t RecoveredToken) Text() string          { return t.text }

type badNodeTokens struct {
	recoveredTokens []RecoveredToken
}

func (b *badNodeTokens) RecoveredTokens() []RecoveredToken { return b.recoveredTokens }

type BLangBadNode interface {
	BLangNode
	badNode()
}

type bLangBadNodeBase struct {
	bLangNodeBase
}

func (*bLangBadNodeBase) badNode() {}

// bLangRecoveredNodeBase is the base of bad nodes that stand in for a whole construct the node builder could not
// build.
type bLangRecoveredNodeBase struct {
	bLangBadNodeBase
	badNodeTokens
}

func newBLangRecoveredNodeBase(pos Location, tokens []RecoveredToken) bLangRecoveredNodeBase {
	return bLangRecoveredNodeBase{
		bLangBadNodeBase: bLangBadNodeBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		badNodeTokens:    badNodeTokens{recoveredTokens: tokens},
	}
}

type BLangBadTopLevelNode struct {
	bLangRecoveredNodeBase
	recoveredKind BadTopLevelNodeKind
}

func NewBLangBadTopLevelNode(pos Location, kind BadTopLevelNodeKind, tokens []RecoveredToken) *BLangBadTopLevelNode {
	return &BLangBadTopLevelNode{bLangRecoveredNodeBase: newBLangRecoveredNodeBase(pos, tokens), recoveredKind: kind}
}

func (b *BLangBadTopLevelNode) RecoveredKind() BadTopLevelNodeKind { return b.recoveredKind }

type BLangBadStmt struct {
	bLangRecoveredNodeBase
	recoveredKind BadStmtKind
}

func NewBLangBadStmt(pos Location, kind BadStmtKind, tokens []RecoveredToken) *BLangBadStmt {
	return &BLangBadStmt{bLangRecoveredNodeBase: newBLangRecoveredNodeBase(pos, tokens), recoveredKind: kind}
}

func (b *BLangBadStmt) RecoveredKind() BadStmtKind { return b.recoveredKind }

type BLangBadExprOrAction struct {
	bLangRecoveredNodeBase
	recoveredKind BadExprKind
}

func NewBLangBadExprOrAction(pos Location, kind BadExprKind, tokens []RecoveredToken) *BLangBadExprOrAction {
	return &BLangBadExprOrAction{bLangRecoveredNodeBase: newBLangRecoveredNodeBase(pos, tokens), recoveredKind: kind}
}

func (b *BLangBadExprOrAction) RecoveredKind() BadExprKind { return b.recoveredKind }

type BLangBadTypeNode struct {
	bLangTypeBase
	badNodeTokens
	recoveredKind BadTypeKind
}

func NewBLangBadTypeNode(pos Location, kind BadTypeKind, tokens []RecoveredToken) *BLangBadTypeNode {
	return &BLangBadTypeNode{
		bLangTypeBase: bLangTypeBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		badNodeTokens: badNodeTokens{recoveredTokens: tokens},
		recoveredKind: kind,
	}
}

func (b *BLangBadTypeNode) RecoveredKind() BadTypeKind { return b.recoveredKind }

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

func (*BLangBadExprOrAction) Type() semtypes.SemType { return semtypes.Never }

func (*BLangBadTypeNode) badNode() {}

func NewBLangBadIdentifier(pos Location, value, originalValue string, isLiteral bool) *BLangBadIdentifier {
	return &BLangBadIdentifier{
		bLangBadNodeBase: bLangBadNodeBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		Value:            value,
		OriginalValue:    originalValue,
		isLiteral:        isLiteral,
	}
}

func (b *BLangBadIdentifier) GetValue() string { return b.Value }
func (b *BLangBadIdentifier) IsLiteral() bool  { return b.isLiteral }
