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
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// Type describes a resolved type in the language model. It is the value-level
// view of a type (kind + type-data) that symbol machinery works against.
type Type interface {
	GetTypeData() TypeData
}

// TypeData pairs the AST type descriptor with the resolved semantic type.
type TypeData struct {
	// TypeDescriptor is the AST-level type representation. Available after AST
	// construction; may be nil if the node has no attached descriptor.
	TypeDescriptor TypeDescriptor
	// Type is the resolved semantic type, set by the semantic analyzer.
	Type semtypes.SemType
}

// Core node interfaces.

type Node interface {
	GetPosition() diagnostics.Location
	GetDeterminedType() semtypes.SemType
}

type NodeWithSymbol interface {
	Node
	Symbol() model.SymbolRef
}

type TopLevelNode interface {
	Node
	isTopLevel()
}

type FunctionBodyNode interface {
	Node
	isFunctionBody()
}

// Function/invokable interfaces.

type InvokableNode interface {
	AnnotatableNode
	DocumentableNode
	GetName() IdentifierNode
	GetParameters() []BLangVariable
	GetReturnTypeDescriptor() *BLangReturnTypeDescriptor
	HasExplicitReturnTypeDescriptor() bool
	GetBody() FunctionBodyNode
	HasBody() bool
	GetRestParam() *BLangVariable
}

type FunctionSignature interface {
	BLangNode
	Parameters() []Param
	RestParameter() Param
	ReturnType() TypeDescriptor
	IsIsolated() bool
	IsTransactional() bool
}

type Param interface {
	BLangNode
	ParamName() string
	Type() BType
	DefaultExpr() BLangExpression
	Symbol() model.SymbolRef
	IsDefaultable() bool
	IsIncludedRecordParam() bool
}

// Class / service.

type TypeDescriptor interface {
	Node
	IsGrouped() bool
}

type FunctionTypeNode interface {
	TypeDescriptor
	GetParams() []*BLangFunctionTypeParam
	GetRestParam() *BLangFunctionTypeParam
	GetReturnTypeNode() TypeDescriptor
}

type ObjectMember interface {
	MemberKind() ObjectMemberKind
	Name() string
	IsPublic() bool
}

// Expression interfaces.

type VariableReferenceNode interface {
	BLangExpression
	isVariableReference()
}

// LExpr marks expressions that can appear on the left-hand side of an
// assignment: simple variable references, field-based access, and
// index-based access.
type LExpr interface {
	BLangExpression
	isLExpr()
}

type ActionNode interface {
	Node
	isAction()
}

type CommitExpressionNode interface {
	BLangExpression
	ActionNode
}

type SimpleVariableReferenceNode interface {
	VariableReferenceNode
	GetPackageAlias() IdentifierNode
	GetVariableName() IdentifierNode
}

type LiteralNode interface {
	BLangExpression
	GetValue() any
	SetValue(value any)
	GetOriginalValue() string
	SetOriginalValue(originalValue string)
	GetIsConstant() bool
	SetIsConstant(isConstant bool)
}

type MappingField interface {
	Node
	IsKeyValueField() bool
}

// Invocable represents a function, method, remote method, or client resource call.
type Invocable interface {
	BLangActionOrExpression
	ResolvedSymbol() model.SymbolRef
	Receiver() BLangExpression
	CallArgs() []BLangExpression
}

//sumtype:decl
type XMLStepExtend interface {
	BLangNode
	xmlStepExtend()
}

type GroupExpressionNode interface {
	BLangExpression
	GetExpression() BLangExpression
}

type TypedescExpressionNode interface {
	BLangExpression
	GetTypeDescriptor() TypeDescriptor
}

type NamedArgNode interface {
	BLangExpression
	GetName() IdentifierNode
	GetExpression() BLangExpression
}

// Statement interfaces.

type StatementNode interface {
	Node
	isStatement()
}

type AssignmentNode interface {
	StatementNode
	GetVariable() LExpr
	GetExpression() BLangActionOrExpression
	IsDeclaredWithVar() bool
}

type ExpressionStatementNode interface {
	StatementNode
	GetExpression() BLangActionOrExpression
}

// Binding pattern interfaces.

type BindingPatternNode interface {
	Node
	isBindingPattern()
}

type RestBindingPatternNode interface {
	Node
	GetIdentifier() *BLangIdentifier
}

// Match pattern interfaces.

type MatchPatternNode interface {
	Node
	GetAcceptedType() semtypes.SemType
}

// Clause interfaces.

type InputClauseNode interface {
	Node
	GetCollection() BLangExpression
	GetVariableDefinitionNode() *BLangVariableDef
	IsDeclaredWithVar() bool
}

type FromClauseNode = InputClauseNode

type SelectClauseNode interface {
	Node
	GetExpression() BLangExpression
}

type CollectClauseNode = SelectClauseNode

type DoClauseNode interface {
	Node
	GetBody() *BLangBlockStmt
}

// Documentation interfaces.

type DocumentableNode interface {
	Node
	GetMarkdownDocumentationAttachment() *BLangMarkdownDocumentation
}

// Other interfaces.

type IdentifierNode interface {
	BLangNode
	GetValue() string
}

type AnnotatableNode interface {
	Node
	IsPublic() bool
	GetAnnotationAttachments() []BLangAnnotationAttachment
	AddAnnotationAttachment(annAttachment BLangAnnotationAttachment)
}

type AttachPoint struct {
	Point  Point
	Source bool
}

// Point identifies an annotation attach point. There are fewer than 20 mutually
// exclusive values, so it is represented as a byte for cheap equality and
// storage. Use String for the canonical attach-point key.
type Point byte

const (
	PointType Point = iota
	PointObject
	PointFunction
	PointObjectMethod
	PointServiceRemote
	PointParameter
	PointReturn
	PointService
	PointField
	PointObjectField
	PointRecordField
	PointListener
	PointAnnotation
	PointExternal
	PointVar
	PointConst
	PointWorker
	PointClass
)

// String returns the canonical attach-point key (no spaces) — the form used as
// the key for annotation attach-point matching and for pretty printing. This is
// distinct from the space-separated source spelling parsed in node_builder.go
// (e.g. "object function" parses to PointObjectMethod whose key is
// "objectfunction").
func (p Point) String() string {
	switch p {
	case PointType:
		return "type"
	case PointObject:
		return "object"
	case PointFunction:
		return "function"
	case PointObjectMethod:
		return "objectfunction"
	case PointServiceRemote:
		return "serviceremotefunction"
	case PointParameter:
		return "parameter"
	case PointReturn:
		return "return"
	case PointService:
		return "service"
	case PointField:
		return "field"
	case PointObjectField:
		return "objectfield"
	case PointRecordField:
		return "recordfield"
	case PointListener:
		return "listener"
	case PointAnnotation:
		return "annotation"
	case PointExternal:
		return "external"
	case PointVar:
		return "var"
	case PointConst:
		return "const"
	case PointWorker:
		return "worker"
	case PointClass:
		return "class"
	default:
		panic("unknown annotation attach point")
	}
}
