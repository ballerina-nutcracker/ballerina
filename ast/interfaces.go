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
	"iter"

	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
)

// Field represents an annotated type member with a name and type.
type Field interface {
	AnnotatableNode
	GetName() model.Name
	GetType() Type
}

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

type CompilationUnitNode interface {
	Node
	AddTopLevelNode(node TopLevelNode)
	GetTopLevelNodes() []TopLevelNode
	SetName(name string)
	GetName() string
}

type PackageNode interface {
	Node
	GetImports() []ImportPackageNode
	AddImport(importPkg ImportPackageNode)
	GetNamespaceDeclarations() []XMLNSDeclarationNode
	AddNamespaceDeclaration(xmlnsDecl XMLNSDeclarationNode)
	GetConstants() []ConstantNode
	GetGlobalVariables() []VariableNode
	AddGlobalVariable(globalVar SimpleVariableNode)
	GetServices() []ServiceNode
	AddService(service ServiceNode)
	GetFunctions() []FunctionNode
	AddFunction(function FunctionNode)
	GetTypeDefinitions() []*BLangTypeDefinition
	AddTypeDefinition(typeDefinition *BLangTypeDefinition)
	GetAnnotations() []AnnotationNode
	AddAnnotation(annotation AnnotationNode)
	GetClassDefinitions() []ClassDefinition
}

type ImportPackageNode interface {
	Node
	TopLevelNode
	GetOrgName() *BLangIdentifier
	GetPackageName() []*BLangIdentifier
	SetPackageName([]*BLangIdentifier)
	GetPackageVersion() *BLangIdentifier
	SetPackageVersion(*BLangIdentifier)
	GetAlias() *BLangIdentifier
	SetAlias(*BLangIdentifier)
}

type XMLNSDeclarationNode interface {
	TopLevelNode
	GetNamespaceURI() BLangExpression
	SetNamespaceURI(namespaceURI BLangExpression)
	GetPrefix() *BLangIdentifier
	SetPrefix(prefix *BLangIdentifier)
}

type AnnotationNode interface {
	AnnotatableNode
	DocumentableNode
	TopLevelNode
	GetName() IdentifierNode
	SetName(name IdentifierNode)
	GetTypeDescriptor() TypeDescriptor
	SetTypeDescriptor(typeDescriptor TypeDescriptor)
}

type FunctionBodyNode interface {
	Node
	isFunctionBody()
}

type ExprFunctionBodyNode interface {
	FunctionBodyNode
	GetExpr() BLangExpression
}

// Variable/constant interfaces.

type VariableNode interface {
	NodeWithSymbol
	AnnotatableNode
	DocumentableNode
	TopLevelNode
	GetInitialExpression() BLangActionOrExpression
	GetIsDeclaredWithVar() bool
	SetIsDeclaredWithVar(isDeclaredWithVar bool)
}

type SimpleVariableNode interface {
	VariableNode
	AnnotatableNode
	DocumentableNode
	TopLevelNode
	GetName() IdentifierNode
	SetName(name IdentifierNode)
}

type ConstantNode interface {
	GetAssociatedType() semtypes.SemType
}

// Function/invokable interfaces.

type InvokableNode interface {
	AnnotatableNode
	DocumentableNode
	GetName() IdentifierNode
	SetName(name IdentifierNode)
	GetParameters() []SimpleVariableNode
	AddParameter(param SimpleVariableNode)
	GetReturnTypeDescriptor() ReturnTypeDescriptor
	SetReturnTypeDescriptor(typeDescriptor TypeDescriptor)
	HasExplicitReturnTypeDescriptor() bool
	GetBody() FunctionBodyNode
	SetBody(body FunctionBodyNode)
	HasBody() bool
	GetRestParam() SimpleVariableNode
	SetRestParameter(restParam SimpleVariableNode)
}

type FunctionNode interface {
	InvokableNode
	AnnotatableNode
	TopLevelNode
}

// Class / service.

type ClassDefinition interface {
	AnnotatableNode
	DocumentableNode
	TopLevelNode
	GetName() IdentifierNode
	GetMethods() iter.Seq2[string, FunctionNode]
	GetMethod(name string) FunctionNode
	GetInitFunction() FunctionNode
}

type ServiceNode interface {
	AnnotatableNode
	DocumentableNode
	TopLevelNode
	GetAttachedExprs() []BLangExpression
	GetAbsolutePath() []*BLangIdentifier
	GetAttachPointLiteral() LiteralNode
}

type TypeDescriptor interface {
	Node
	IsGrouped() bool
}

type ReturnTypeDescriptor interface {
	BType
	AnnotatableNode
}

type ArrayTypeNode interface {
	TypeDescriptor
	GetElementType() TypeData
	GetDimensions() int
	GetSizes() []BLangExpression
}

type RecordTypeNode interface {
	TypeDescriptor
	GetRestFieldType() TypeData
	GetFields() iter.Seq2[string, Field]
}

type FunctionTypeParam interface {
	Node
	GetName() *string
	GetTypeDesc() Type
}

type FunctionTypeNode interface {
	TypeDescriptor
	GetParams() []FunctionTypeParam
	GetRestParam() FunctionTypeParam
	GetReturnTypeNode() TypeDescriptor
}

type TupleTypeNode interface {
	TypeDescriptor
	GetMembers() []MemberTypeDesc
	GetRest() TypeDescriptor
}

type MemberTypeDesc interface {
	Node
	AnnotatableNode
	GetTypeDesc() TypeDescriptor
}

type FiniteTypeNode interface {
	TypeDescriptor
	GetValueSet() []BLangExpression
	AddValue(value BLangExpression)
}

type ObjectMember interface {
	MemberKind() ObjectMemberKind
	Name() string
	IsPublic() bool
}

type ObjectType interface {
	TypeDescriptor
	Members() iter.Seq[ObjectMember]
	Member(name string) (ObjectMember, bool)
}

type UnionTypeNode interface {
	TypeDescriptor
	Lhs() *TypeData
	Rhs() *TypeData
}

type IntersectionTypeNode interface {
	TypeDescriptor
	Lhs() *TypeData
	Rhs() *TypeData
}

type ErrorTypeNode interface {
	TypeDescriptor
	GetDetailType() TypeData
}

type ConstrainedTypeNode interface {
	TypeDescriptor
	GetType() TypeData
	GetConstraint() TypeData
}

type UserDefinedTypeNode interface {
	TypeDescriptor
	GetPackageAlias() *BLangIdentifier
	GetTypeName() *BLangIdentifier
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

type BinaryExpressionNode interface {
	GetLeftExpression() BLangExpression
	GetRightExpression() BLangExpression
	GetOperatorKind() model.OperatorKind
}

type UnaryExpressionNode interface {
	GetExpression() BLangExpression
	GetOperatorKind() model.OperatorKind
}

type IndexBasedAccessNode interface {
	BLangExpression
	GetExpression() BLangExpression
	GetIndex() BLangExpression
}

type FieldBasedAccessNode interface {
	BLangExpression
	GetExpression() BLangExpression
	GetFieldName() IdentifierNode
}

type ListConstructorExprNode interface {
	BLangExpression
	GetExpressions() []BLangExpression
}

type TypeTestExpressionNode interface {
	BLangExpression
	GetExpression() BLangExpression
	GetType() TypeData
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

type ElvisExpressionNode interface {
	GetLeftExpression() BLangExpression
	GetRightExpression() BLangExpression
}

type MappingField interface {
	Node
	IsKeyValueField() bool
}

type MappingVarNameFieldNode interface {
	MappingField
	SimpleVariableReferenceNode
}

type MappingConstructor interface {
	BLangExpression
	GetFields() []MappingField
}

type MappingKeyValueFieldNode interface {
	MappingField
	GetKey() BLangExpression
	GetValue() BLangExpression
}

type MarkdownDocumentationTextAttributeNode interface {
	BLangExpression
	GetText() string
	SetText(text string)
}

type MarkdownDocumentationParameterAttributeNode interface {
	BLangExpression
	GetParameterName() *BLangIdentifier
	SetParameterName(parameterName *BLangIdentifier)
	GetParameterDocumentationLines() []string
	AddParameterDocumentationLine(text string)
	GetParameterDocumentation() string
}

type MarkdownDocumentationReturnParameterAttributeNode interface {
	BLangExpression
	GetReturnParameterDocumentationLines() []string
	AddReturnParameterDocumentationLine(text string)
	GetReturnParameterDocumentation() string
	GetReturnType() *BLangValueType
	SetReturnType(typ *BLangValueType)
}

type MarkDownDocumentationDeprecationAttributeNode interface {
	BLangExpression
	AddDeprecationDocumentationLine(text string)
	AddDeprecationLine(text string)
	GetDocumentation() string
}

type MarkDownDocumentationDeprecatedParametersAttributeNode interface {
	BLangExpression
	AddParameter(parameter MarkdownDocumentationParameterAttributeNode)
	GetParameters() []MarkdownDocumentationParameterAttributeNode
}

type WorkerReceiveNode interface {
	BLangExpression
	ActionNode
	GetWorkerName() *BLangIdentifier
	SetWorkerName(identifierNode *BLangIdentifier)
}

// WorkerSendExpressionNode currently has no concrete implementations; it
// exists as a placeholder for BLangWorkerReceive.Send until worker-send
// expressions are modeled.
type WorkerSendExpressionNode interface {
	BLangExpression
	ActionNode
	GetExpression() BLangExpression
	GetWorkerName() *BLangIdentifier
	SetWorkerName(identifierNode *BLangIdentifier)
}

type MarkdownDocumentationReferenceAttributeNode interface {
	Node
	GetType() DocumentationReferenceType
}

type LambdaFunctionNode interface {
	BLangExpression
	GetFunctionNode() FunctionNode
	SetFunctionNode(functionNode FunctionNode)
}

type InvocationNode interface {
	BLangExpression
	GetPackageAlias() IdentifierNode
	GetName() IdentifierNode
	GetArgumentExpressions() []BLangExpression
	GetRequiredArgs() []BLangExpression
	GetExpression() BLangExpression
}

type GroupExpressionNode interface {
	BLangExpression
	GetExpression() BLangExpression
}

type TypedescExpressionNode interface {
	BLangExpression
	GetTypeDescriptor() TypeDescriptor
	SetTypeDescriptor(typeDescriptor TypeDescriptor)
}

type NamedArgNode interface {
	BLangExpression
	SetName(name IdentifierNode)
	GetName() IdentifierNode
	GetExpression() BLangExpression
	SetExpression(expr BLangExpression)
}

type ErrorConstructorExpressionNode interface {
	BLangExpression
	GetPositionalArgs() []BLangExpression
	GetNamedArgs() []NamedArgNode
}

type TypeConversionNode interface {
	BLangExpression
	AnnotatableNode
	GetExpression() BLangExpression
	SetExpression(expression BLangExpression)
	GetTypeDescriptor() TypeDescriptor
	SetTypeDescriptor(typeDescriptor TypeDescriptor)
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
	SetDeclaredWithVar(IsDeclaredWithVar bool)
	SetVariable(lhs LExpr)
}

type CompoundAssignmentNode interface {
	AssignmentNode
	GetOperatorKind() model.OperatorKind
}

type BlockNode interface {
	Node
	GetStatements() []StatementNode
	AddStatement(statement StatementNode)
}

type BlockStatementNode interface {
	BlockNode
	StatementNode
}

type ExpressionStatementNode interface {
	StatementNode
	GetExpression() BLangActionOrExpression
}

type IfNode interface {
	StatementNode
	GetCondition() BLangExpression
	SetCondition(condition BLangExpression)
	GetBody() BlockStatementNode
	SetBody(body BlockStatementNode)
	GetElseStatement() StatementNode
	SetElseStatement(elseStatement StatementNode)
}

type VariableDefinitionNode interface {
	StatementNode
	GetVariable() VariableNode
	SetVariable(variable VariableNode)
}

type ReturnNode interface {
	StatementNode
	GetExpression() BLangActionOrExpression
}

type PanicNode interface {
	StatementNode
	GetExpression() BLangExpression
}

type TrapNode interface {
	BLangExpression
	GetExpression() BLangExpression
}

type DoNode interface {
	StatementNode
	GetBody() BlockStatementNode
	SetBody(body BlockStatementNode)
	GetOnFailClause() OnFailClauseNode
	SetOnFailClause(onFailClause OnFailClauseNode)
}

type WhileNode interface {
	StatementNode
	GetCondition() BLangExpression
	SetCondition(condition BLangExpression)
	GetBody() BlockStatementNode
	SetBody(body BlockStatementNode)
	GetOnFailClause() OnFailClauseNode
	SetOnFailClause(onFailClause OnFailClauseNode)
}

type ForeachNode interface {
	StatementNode
	GetVariableDefinitionNode() VariableDefinitionNode
	SetVariableDefinitionNode(node VariableDefinitionNode)
	GetCollection() BLangActionOrExpression
	GetBody() BlockStatementNode
	SetBody(body BlockStatementNode)
	GetIsDeclaredWithVar() bool
	GetOnFailClause() OnFailClauseNode
	SetOnFailClause(onFailClause OnFailClauseNode)
}

// Binding pattern interfaces.

type BindingPatternNode interface {
	Node
	isBindingPattern()
}

type WildCardBindingPatternNode interface {
	BindingPatternNode
	isWildCardBindingPattern()
}

type CaptureBindingPatternNode interface {
	BindingPatternNode
	GetIdentifier() *BLangIdentifier
	SetIdentifier(identifier *BLangIdentifier)
}

type SimpleBindingPatternNode interface {
	BindingPatternNode
	GetCaptureBindingPattern() CaptureBindingPatternNode
	SetCaptureBindingPattern(captureBindingPatternNode CaptureBindingPatternNode)
	GetWildCardBindingPattern() WildCardBindingPatternNode
	SetWildCardBindingPattern(wildCardBindingPatternNode WildCardBindingPatternNode)
}

type ErrorMessageBindingPatternNode interface {
	BindingPatternNode
	GetSimpleBindingPattern() SimpleBindingPatternNode
	SetSimpleBindingPattern(simpleBindingPatternNode SimpleBindingPatternNode)
}

type ErrorBindingPatternNode interface {
	BindingPatternNode
	GetErrorTypeReference() UserDefinedTypeNode
	SetErrorTypeReference(userDefinedTypeNode UserDefinedTypeNode)
	GetErrorMessageBindingPatternNode() ErrorMessageBindingPatternNode
	SetErrorMessageBindingPatternNode(errorMessageBindingPatternNode ErrorMessageBindingPatternNode)
	GetErrorCauseBindingPatternNode() ErrorCauseBindingPatternNode
	SetErrorCauseBindingPatternNode(errorCauseBindingPatternNode ErrorCauseBindingPatternNode)
	GetErrorFieldBindingPatternsNode() ErrorFieldBindingPatternsNode
	SetErrorFieldBindingPatternsNode(errorFieldBindingPatternsNode ErrorFieldBindingPatternsNode)
}

type ErrorCauseBindingPatternNode interface {
	BindingPatternNode
	GetSimpleBindingPattern() SimpleBindingPatternNode
	SetSimpleBindingPattern(simpleBindingPatternNode SimpleBindingPatternNode)
	GetErrorBindingPatternNode() ErrorBindingPatternNode
	SetErrorBindingPatternNode(errorBindingPatternNode ErrorBindingPatternNode)
}

type ErrorFieldBindingPatternsNode interface {
	BindingPatternNode
	GetNamedArgMatchPatterns() []NamedArgBindingPatternNode
	AddNamedArgBindingPattern(namedArgBindingPatternNode NamedArgBindingPatternNode)
	GetRestBindingPattern() RestBindingPatternNode
	SetRestBindingPattern(restBindingPattern RestBindingPatternNode)
}

type NamedArgBindingPatternNode interface {
	Node
	GetIdentifier() *BLangIdentifier
	SetIdentifier(identifier *BLangIdentifier)
	GetBindingPattern() BindingPatternNode
	SetBindingPattern(bindingPattern BindingPatternNode)
}

type RestBindingPatternNode interface {
	Node
	GetIdentifier() *BLangIdentifier
	SetIdentifier(identifier *BLangIdentifier)
}

// Match pattern interfaces.

type MatchStatement interface {
	StatementNode
	GetExpression() BLangExpression
	GetClauses() []MatchClause
}

type MatchClause interface {
	Node
	GetMatchGuard() BLangExpression
	GetBlockStatementNode() BlockStatementNode
	GetMatchPatterns() []MatchPatternNode
	GetAcceptedType() semtypes.SemType
}

type MatchPatternNode interface {
	Node
	GetAcceptedType() semtypes.SemType
}

type ConstPatternNode interface {
	MatchPatternNode
	GetExpression() BLangExpression
}

// Clause interfaces.

type InputClauseNode interface {
	Node
	GetCollection() BLangExpression
	SetCollection(collection BLangExpression)
	GetVariableDefinitionNode() VariableDefinitionNode
	SetVariableDefinitionNode(variableDefinitionNode VariableDefinitionNode)
	IsDeclaredWithVar() bool
}

type FromClauseNode interface {
	InputClauseNode
}

type JoinClauseNode interface {
	InputClauseNode
	GetOnClause() OnClauseNode
	IsOuterJoin() bool
}

type OnClauseNode interface {
	Node
	GetOnExpression() BLangExpression
	SetOnExpression(expression BLangExpression)
	GetEqualsExpression() BLangExpression
	SetEqualsExpression(expression BLangExpression)
}

type GroupByClauseNode interface {
	Node
	AddGroupingKey(groupingKey GroupingKeyNode)
	GetGroupingKeyList() []GroupingKeyNode
}

type GroupingKeyNode interface {
	Node
	GetGroupingKey() Node
	SetGroupingKey(groupingKey Node)
}

type SelectClauseNode interface {
	Node
	GetExpression() BLangExpression
	SetExpression(expression BLangExpression)
}

type QueryExpressionNode interface {
	BLangExpression
	GetQueryClauses() []Node
	AddQueryClause(queryClause Node)
}

type CollectClauseNode interface {
	Node
	GetExpression() BLangExpression
	SetExpression(expression BLangExpression)
}

type DoClauseNode interface {
	Node
	GetBody() BlockStatementNode
	SetBody(body BlockStatementNode)
}

type OnFailClauseNode interface {
	Node
	SetDeclaredWithVar()
	IsDeclaredWithVar() bool
	GetVariableDefinitionNode() VariableDefinitionNode
	SetVariableDefinitionNode(variableDefinitionNode VariableDefinitionNode)
	GetBody() BlockStatementNode
	SetBody(body BlockStatementNode)
}

// Documentation interfaces.

type DocumentableNode interface {
	Node
	GetMarkdownDocumentationAttachment() MarkdownDocumentationNode
	SetMarkdownDocumentationAttachment(documentationNode MarkdownDocumentationNode)
}

type MarkdownDocumentationNode interface {
	Node
	GetDocumentationLines() []MarkdownDocumentationTextAttributeNode
	AddDocumentationLine(documentationText MarkdownDocumentationTextAttributeNode)
	GetParameters() []MarkdownDocumentationParameterAttributeNode
	AddParameter(parameter MarkdownDocumentationParameterAttributeNode)
	GetReturnParameter() MarkdownDocumentationReturnParameterAttributeNode
	GetDeprecationDocumentation() MarkDownDocumentationDeprecationAttributeNode
	SetReturnParameter(returnParameter MarkdownDocumentationReturnParameterAttributeNode)
	SetDeprecationDocumentation(deprecationDocumentation MarkDownDocumentationDeprecationAttributeNode)
	SetDeprecatedParametersDocumentation(deprecatedParametersDocumentation MarkDownDocumentationDeprecatedParametersAttributeNode)
	GetDeprecatedParametersDocumentation() MarkDownDocumentationDeprecatedParametersAttributeNode
	GetDocumentation() string
	GetParameterDocumentations() map[string]MarkdownDocumentationParameterAttributeNode
	GetReturnParameterDocumentation() *string
	GetReferences() []MarkdownDocumentationReferenceAttributeNode
	AddReference(reference MarkdownDocumentationReferenceAttributeNode)
}

// Other interfaces.

type IdentifierNode interface {
	BLangNode
	GetValue() string
	IsLiteral() bool
}

type AnnotationAttachmentNode interface {
	GetPackageAlias() *BLangIdentifier
	SetPackageAlias(pkgAlias *BLangIdentifier)
	GetAnnotationName() *BLangIdentifier
	SetAnnotationName(name *BLangIdentifier)
	GetExpressionNode() BLangExpression
	SetExpressionNode(expr BLangExpression)
}

type AnnotatableNode interface {
	Node
	IsPublic() bool
	GetAnnotationAttachments() []AnnotationAttachmentNode
	AddAnnotationAttachment(annAttachment AnnotationAttachmentNode)
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
	Point_TYPE Point = iota
	Point_OBJECT
	Point_FUNCTION
	Point_OBJECT_METHOD
	Point_SERVICE_REMOTE
	Point_PARAMETER
	Point_RETURN
	Point_SERVICE
	Point_FIELD
	Point_OBJECT_FIELD
	Point_RECORD_FIELD
	Point_LISTENER
	Point_ANNOTATION
	Point_EXTERNAL
	Point_VAR
	Point_CONST
	Point_WORKER
	Point_CLASS
)

// String returns the canonical attach-point key (no spaces) — the form used as
// the key for annotation attach-point matching and for pretty printing. This is
// distinct from the space-separated source spelling parsed in node_builder.go
// (e.g. "object function" parses to Point_OBJECT_METHOD whose key is
// "objectfunction").
func (p Point) String() string {
	switch p {
	case Point_TYPE:
		return "type"
	case Point_OBJECT:
		return "object"
	case Point_FUNCTION:
		return "function"
	case Point_OBJECT_METHOD:
		return "objectfunction"
	case Point_SERVICE_REMOTE:
		return "serviceremotefunction"
	case Point_PARAMETER:
		return "parameter"
	case Point_RETURN:
		return "return"
	case Point_SERVICE:
		return "service"
	case Point_FIELD:
		return "field"
	case Point_OBJECT_FIELD:
		return "objectfield"
	case Point_RECORD_FIELD:
		return "recordfield"
	case Point_LISTENER:
		return "listener"
	case Point_ANNOTATION:
		return "annotation"
	case Point_EXTERNAL:
		return "external"
	case Point_VAR:
		return "var"
	case Point_CONST:
		return "const"
	case Point_WORKER:
		return "worker"
	case Point_CLASS:
		return "class"
	default:
		panic("unknown annotation attach point")
	}
}
