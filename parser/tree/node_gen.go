// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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
//

package tree

import (
	"ballerina/parser/common"
)

type ModulePart struct {
	NonTerminalNodeBase
}

func (n ModulePart) Kind() common.SyntaxKind {
	return common.MODULE_PART
}

func (n ModulePart) Imports() NodeList[*ImportDeclarationNode] {
	return nodeListFrom[*ImportDeclarationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n ModulePart) Members() NodeList[ModuleMemberDeclarationNode] {
	return nodeListFrom[ModuleMemberDeclarationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ModulePart) EofToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ModuleMemberDeclarationNode = NonTerminalNode

type FunctionDefinition struct {
	ModuleMemberDeclarationNode
}

func (n FunctionDefinition) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionDefinition) QualifierList() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n FunctionDefinition) FunctionKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionDefinition) FunctionName() *IdentifierToken {
	val, ok := n.ChildInBucket(3).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionDefinition) RelativeResourcePath() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(4)))
}

func (n FunctionDefinition) FunctionSignature() *FunctionSignatureNode {
	val, ok := n.ChildInBucket(5).(*FunctionSignatureNode)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionDefinition) FunctionBody() FunctionBodyNode {
	val, ok := n.ChildInBucket(6).(FunctionBodyNode)
	if !ok {
		return nil
	}
	return val
}

type ImportDeclarationNode struct {
	NonTerminalNodeBase
}

func (n ImportDeclarationNode) Kind() common.SyntaxKind {
	return common.IMPORT_DECLARATION
}

func (n ImportDeclarationNode) ImportKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ImportDeclarationNode) OrgName() *ImportOrgNameNode {
	val, ok := n.ChildInBucket(1).(*ImportOrgNameNode)
	if !ok {
		return nil
	}
	return val
}

func (n ImportDeclarationNode) ModuleName() NodeList[*IdentifierToken] {
	return nodeListFrom[*IdentifierToken](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n ImportDeclarationNode) Prefix() *ImportPrefixNode {
	val, ok := n.ChildInBucket(3).(*ImportPrefixNode)
	if !ok {
		return nil
	}
	return val
}

func (n ImportDeclarationNode) Semicolon() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type ListenerDeclarationNode struct {
	ModuleMemberDeclarationNode
}

func (n ListenerDeclarationNode) Kind() common.SyntaxKind {
	return common.LISTENER_DECLARATION
}

func (n ListenerDeclarationNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n ListenerDeclarationNode) VisibilityQualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ListenerDeclarationNode) ListenerKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ListenerDeclarationNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(3).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ListenerDeclarationNode) VariableName() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ListenerDeclarationNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ListenerDeclarationNode) Initializer() Node {
	return n.ChildInBucket(6)
}

func (n ListenerDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(7).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypeDefinitionNode struct {
	ModuleMemberDeclarationNode
}

func (n TypeDefinitionNode) Kind() common.SyntaxKind {
	return common.TYPE_DEFINITION
}

func (n TypeDefinitionNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n TypeDefinitionNode) VisibilityQualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeDefinitionNode) TypeKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeDefinitionNode) TypeName() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeDefinitionNode) TypeDescriptor() Node {
	return n.ChildInBucket(4)
}

func (n TypeDefinitionNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

type ServiceDeclarationNode struct {
	ModuleMemberDeclarationNode
}

func (n ServiceDeclarationNode) Kind() common.SyntaxKind {
	return common.SERVICE_DECLARATION
}

func (n ServiceDeclarationNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n ServiceDeclarationNode) Qualifiers() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ServiceDeclarationNode) ServiceKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ServiceDeclarationNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(3).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ServiceDeclarationNode) AbsoluteResourcePath() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(4)))
}

func (n ServiceDeclarationNode) OnKeyword() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ServiceDeclarationNode) Expressions() NodeList[ExpressionNode] {
	return nodeListFrom[ExpressionNode](into[NonTerminalNode](n.ChildInBucket(6)))
}

func (n ServiceDeclarationNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(7).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ServiceDeclarationNode) Members() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(8)))
}

func (n ServiceDeclarationNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(9).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ServiceDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(10).(Token)
	if !ok {
		return nil
	}
	return val
}

type StatementNode = NonTerminalNode

type AssignmentStatementNode struct {
	StatementNode
}

func (n AssignmentStatementNode) Kind() common.SyntaxKind {
	return common.ASSIGNMENT_STATEMENT
}

func (n AssignmentStatementNode) VarRef() Node {
	return n.ChildInBucket(0)
}

func (n AssignmentStatementNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AssignmentStatementNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n AssignmentStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type CompoundAssignmentStatementNode struct {
	StatementNode
}

func (n CompoundAssignmentStatementNode) Kind() common.SyntaxKind {
	return common.COMPOUND_ASSIGNMENT_STATEMENT
}

func (n CompoundAssignmentStatementNode) LhsExpression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n CompoundAssignmentStatementNode) BinaryOperator() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n CompoundAssignmentStatementNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n CompoundAssignmentStatementNode) RhsExpression() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n CompoundAssignmentStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type VariableDeclarationNode struct {
	StatementNode
}

func (n VariableDeclarationNode) Kind() common.SyntaxKind {
	return common.LOCAL_VAR_DECL
}

func (n VariableDeclarationNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n VariableDeclarationNode) FinalKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n VariableDeclarationNode) TypedBindingPattern() *TypedBindingPatternNode {
	val, ok := n.ChildInBucket(2).(*TypedBindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n VariableDeclarationNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n VariableDeclarationNode) Initializer() ExpressionNode {
	val, ok := n.ChildInBucket(4).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n VariableDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

type BlockStatementNode struct {
	StatementNode
}

func (n BlockStatementNode) Kind() common.SyntaxKind {
	return common.BLOCK_STATEMENT
}

func (n BlockStatementNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n BlockStatementNode) Statements() NodeList[StatementNode] {
	return nodeListFrom[StatementNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n BlockStatementNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type BreakStatementNode struct {
	StatementNode
}

func (n BreakStatementNode) Kind() common.SyntaxKind {
	return common.BREAK_STATEMENT
}

func (n BreakStatementNode) BreakToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n BreakStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type FailStatementNode struct {
	StatementNode
}

func (n FailStatementNode) Kind() common.SyntaxKind {
	return common.FAIL_STATEMENT
}

func (n FailStatementNode) FailKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FailStatementNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n FailStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ExpressionStatementNode struct {
	StatementNode
}

func (n ExpressionStatementNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ExpressionStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type ContinueStatementNode struct {
	StatementNode
}

func (n ContinueStatementNode) Kind() common.SyntaxKind {
	return common.CONTINUE_STATEMENT
}

func (n ContinueStatementNode) ContinueToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ContinueStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type ExternalFunctionBodyNode struct {
	FunctionBodyNode
}

func (n ExternalFunctionBodyNode) Kind() common.SyntaxKind {
	return common.EXTERNAL_FUNCTION_BODY
}

func (n ExternalFunctionBodyNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ExternalFunctionBodyNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ExternalFunctionBodyNode) ExternalKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ExternalFunctionBodyNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type IfElseStatementNode struct {
	StatementNode
}

func (n IfElseStatementNode) Kind() common.SyntaxKind {
	return common.IF_ELSE_STATEMENT
}

func (n IfElseStatementNode) IfKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n IfElseStatementNode) Condition() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n IfElseStatementNode) IfBody() *BlockStatementNode {
	val, ok := n.ChildInBucket(2).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n IfElseStatementNode) ElseBody() Node {
	return n.ChildInBucket(3)
}

type ElseBlockNode struct {
	NonTerminalNodeBase
}

func (n ElseBlockNode) Kind() common.SyntaxKind {
	return common.ELSE_BLOCK
}

func (n ElseBlockNode) ElseKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ElseBlockNode) ElseBody() StatementNode {
	val, ok := n.ChildInBucket(1).(StatementNode)
	if !ok {
		return nil
	}
	return val
}

type WhileStatementNode struct {
	StatementNode
}

func (n WhileStatementNode) Kind() common.SyntaxKind {
	return common.WHILE_STATEMENT
}

func (n WhileStatementNode) WhileKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n WhileStatementNode) Condition() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n WhileStatementNode) WhileBody() *BlockStatementNode {
	val, ok := n.ChildInBucket(2).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n WhileStatementNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(3).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type PanicStatementNode struct {
	StatementNode
}

func (n PanicStatementNode) Kind() common.SyntaxKind {
	return common.PANIC_STATEMENT
}

func (n PanicStatementNode) PanicKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n PanicStatementNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n PanicStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReturnStatementNode struct {
	StatementNode
}

func (n ReturnStatementNode) Kind() common.SyntaxKind {
	return common.RETURN_STATEMENT
}

func (n ReturnStatementNode) ReturnKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReturnStatementNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReturnStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type LocalTypeDefinitionStatementNode struct {
	StatementNode
}

func (n LocalTypeDefinitionStatementNode) Kind() common.SyntaxKind {
	return common.LOCAL_TYPE_DEFINITION_STATEMENT
}

func (n LocalTypeDefinitionStatementNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n LocalTypeDefinitionStatementNode) TypeKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n LocalTypeDefinitionStatementNode) TypeName() Node {
	return n.ChildInBucket(2)
}

func (n LocalTypeDefinitionStatementNode) TypeDescriptor() Node {
	return n.ChildInBucket(3)
}

func (n LocalTypeDefinitionStatementNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type LockStatementNode struct {
	StatementNode
}

func (n LockStatementNode) Kind() common.SyntaxKind {
	return common.LOCK_STATEMENT
}

func (n LockStatementNode) LockKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n LockStatementNode) BlockStatement() *BlockStatementNode {
	val, ok := n.ChildInBucket(1).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n LockStatementNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(2).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type ForkStatementNode struct {
	StatementNode
}

func (n ForkStatementNode) Kind() common.SyntaxKind {
	return common.FORK_STATEMENT
}

func (n ForkStatementNode) ForkKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ForkStatementNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ForkStatementNode) NamedWorkerDeclarations() NodeList[*NamedWorkerDeclarationNode] {
	return nodeListFrom[*NamedWorkerDeclarationNode](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n ForkStatementNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type ForEachStatementNode struct {
	StatementNode
}

func (n ForEachStatementNode) Kind() common.SyntaxKind {
	return common.FOREACH_STATEMENT
}

func (n ForEachStatementNode) ForEachKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ForEachStatementNode) TypedBindingPattern() *TypedBindingPatternNode {
	val, ok := n.ChildInBucket(1).(*TypedBindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n ForEachStatementNode) InKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ForEachStatementNode) ActionOrExpressionNode() Node {
	return n.ChildInBucket(3)
}

func (n ForEachStatementNode) BlockStatement() *BlockStatementNode {
	val, ok := n.ChildInBucket(4).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n ForEachStatementNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(5).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type ExpressionNode = NonTerminalNode

type BinaryExpressionNode struct {
	ExpressionNode
}

func (n BinaryExpressionNode) LhsExpr() Node {
	return n.ChildInBucket(0)
}

func (n BinaryExpressionNode) Operator() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n BinaryExpressionNode) RhsExpr() Node {
	return n.ChildInBucket(2)
}

type BracedExpressionNode struct {
	ExpressionNode
}

func (n BracedExpressionNode) OpenParen() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n BracedExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n BracedExpressionNode) CloseParen() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type CheckExpressionNode struct {
	ExpressionNode
}

func (n CheckExpressionNode) CheckKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n CheckExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type FieldAccessExpressionNode struct {
	ExpressionNode
}

func (n FieldAccessExpressionNode) Kind() common.SyntaxKind {
	return common.FIELD_ACCESS
}

func (n FieldAccessExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n FieldAccessExpressionNode) DotToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FieldAccessExpressionNode) FieldName() NameReferenceNode {
	val, ok := n.ChildInBucket(2).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type FunctionCallExpressionNode struct {
	ExpressionNode
}

func (n FunctionCallExpressionNode) Kind() common.SyntaxKind {
	return common.FUNCTION_CALL
}

func (n FunctionCallExpressionNode) FunctionName() NameReferenceNode {
	val, ok := n.ChildInBucket(0).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionCallExpressionNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionCallExpressionNode) Arguments() NodeList[FunctionArgumentNode] {
	return nodeListFrom[FunctionArgumentNode](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n FunctionCallExpressionNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type MethodCallExpressionNode struct {
	ExpressionNode
}

func (n MethodCallExpressionNode) Kind() common.SyntaxKind {
	return common.METHOD_CALL
}

func (n MethodCallExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n MethodCallExpressionNode) DotToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MethodCallExpressionNode) MethodName() NameReferenceNode {
	val, ok := n.ChildInBucket(2).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n MethodCallExpressionNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MethodCallExpressionNode) Arguments() NodeList[FunctionArgumentNode] {
	return nodeListFrom[FunctionArgumentNode](into[NonTerminalNode](n.ChildInBucket(4)))
}

func (n MethodCallExpressionNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

type MappingConstructorExpressionNode struct {
	ExpressionNode
}

func (n MappingConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.MAPPING_CONSTRUCTOR
}

func (n MappingConstructorExpressionNode) OpenBrace() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MappingConstructorExpressionNode) Fields() NodeList[MappingFieldNode] {
	return nodeListFrom[MappingFieldNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n MappingConstructorExpressionNode) FieldNodes() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n MappingConstructorExpressionNode) CloseBrace() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type IndexedExpressionNode struct {
	TypeDescriptorNode
}

func (n IndexedExpressionNode) Kind() common.SyntaxKind {
	return common.INDEXED_EXPRESSION
}

func (n IndexedExpressionNode) ContainerExpression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n IndexedExpressionNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n IndexedExpressionNode) KeyExpression() NodeList[ExpressionNode] {
	return nodeListFrom[ExpressionNode](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n IndexedExpressionNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypeofExpressionNode struct {
	ExpressionNode
}

func (n TypeofExpressionNode) Kind() common.SyntaxKind {
	return common.TYPEOF_EXPRESSION
}

func (n TypeofExpressionNode) TypeofKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeofExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type UnaryExpressionNode struct {
	ExpressionNode
}

func (n UnaryExpressionNode) Kind() common.SyntaxKind {
	return common.UNARY_EXPRESSION
}

func (n UnaryExpressionNode) UnaryOperator() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n UnaryExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type ComputedNameFieldNode struct {
	MappingFieldNode
}

func (n ComputedNameFieldNode) Kind() common.SyntaxKind {
	return common.COMPUTED_NAME_FIELD
}

func (n ComputedNameFieldNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ComputedNameFieldNode) FieldNameExpr() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ComputedNameFieldNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ComputedNameFieldNode) ColonToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ComputedNameFieldNode) ValueExpr() ExpressionNode {
	val, ok := n.ChildInBucket(4).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type ConstantDeclarationNode struct {
	ModuleMemberDeclarationNode
}

func (n ConstantDeclarationNode) Kind() common.SyntaxKind {
	return common.CONST_DECLARATION
}

func (n ConstantDeclarationNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n ConstantDeclarationNode) VisibilityQualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ConstantDeclarationNode) ConstKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ConstantDeclarationNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(3).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ConstantDeclarationNode) VariableName() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ConstantDeclarationNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ConstantDeclarationNode) Initializer() Node {
	return n.ChildInBucket(6)
}

func (n ConstantDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(7).(Token)
	if !ok {
		return nil
	}
	return val
}

type ParameterNode = NonTerminalNode

type DefaultableParameterNode struct {
	ParameterNode
}

func (n DefaultableParameterNode) Kind() common.SyntaxKind {
	return common.DEFAULTABLE_PARAM
}

func (n DefaultableParameterNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n DefaultableParameterNode) TypeName() Node {
	return n.ChildInBucket(1)
}

func (n DefaultableParameterNode) ParamName() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n DefaultableParameterNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n DefaultableParameterNode) Expression() Node {
	return n.ChildInBucket(4)
}

type RequiredParameterNode struct {
	ParameterNode
}

func (n RequiredParameterNode) Kind() common.SyntaxKind {
	return common.REQUIRED_PARAM
}

func (n RequiredParameterNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n RequiredParameterNode) TypeName() Node {
	return n.ChildInBucket(1)
}

func (n RequiredParameterNode) ParamName() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type IncludedRecordParameterNode struct {
	ParameterNode
}

func (n IncludedRecordParameterNode) Kind() common.SyntaxKind {
	return common.INCLUDED_RECORD_PARAM
}

func (n IncludedRecordParameterNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n IncludedRecordParameterNode) AsteriskToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n IncludedRecordParameterNode) TypeName() Node {
	return n.ChildInBucket(2)
}

func (n IncludedRecordParameterNode) ParamName() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type RestParameterNode struct {
	ParameterNode
}

func (n RestParameterNode) Kind() common.SyntaxKind {
	return common.REST_PARAM
}

func (n RestParameterNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n RestParameterNode) TypeName() Node {
	return n.ChildInBucket(1)
}

func (n RestParameterNode) EllipsisToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RestParameterNode) ParamName() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type ImportOrgNameNode struct {
	NonTerminalNodeBase
}

func (n ImportOrgNameNode) Kind() common.SyntaxKind {
	return common.IMPORT_ORG_NAME
}

func (n ImportOrgNameNode) OrgName() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ImportOrgNameNode) SlashToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type ImportPrefixNode struct {
	NonTerminalNodeBase
}

func (n ImportPrefixNode) Kind() common.SyntaxKind {
	return common.IMPORT_PREFIX
}

func (n ImportPrefixNode) AsKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ImportPrefixNode) Prefix() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type MappingFieldNode = NonTerminalNode

type SpecificFieldNode struct {
	MappingFieldNode
}

func (n SpecificFieldNode) Kind() common.SyntaxKind {
	return common.SPECIFIC_FIELD
}

func (n SpecificFieldNode) ReadonlyKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n SpecificFieldNode) FieldName() Node {
	return n.ChildInBucket(1)
}

func (n SpecificFieldNode) Colon() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n SpecificFieldNode) ValueExpr() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type SpreadFieldNode struct {
	MappingFieldNode
}

func (n SpreadFieldNode) Kind() common.SyntaxKind {
	return common.SPREAD_FIELD
}

func (n SpreadFieldNode) Ellipsis() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n SpreadFieldNode) ValueExpr() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type FunctionArgumentNode = NonTerminalNode

type NamedArgumentNode struct {
	FunctionArgumentNode
}

func (n NamedArgumentNode) Kind() common.SyntaxKind {
	return common.NAMED_ARG
}

func (n NamedArgumentNode) ArgumentName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(0).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n NamedArgumentNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NamedArgumentNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type PositionalArgumentNode struct {
	FunctionArgumentNode
}

func (n PositionalArgumentNode) Kind() common.SyntaxKind {
	return common.POSITIONAL_ARG
}

func (n PositionalArgumentNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type RestArgumentNode struct {
	FunctionArgumentNode
}

func (n RestArgumentNode) Kind() common.SyntaxKind {
	return common.REST_ARG
}

func (n RestArgumentNode) Ellipsis() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RestArgumentNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type InferredTypedescDefaultNode struct {
	ExpressionNode
}

func (n InferredTypedescDefaultNode) Kind() common.SyntaxKind {
	return common.INFERRED_TYPEDESC_DEFAULT
}

func (n InferredTypedescDefaultNode) LtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n InferredTypedescDefaultNode) GtToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type ObjectTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n ObjectTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.OBJECT_TYPE_DESC
}

func (n ObjectTypeDescriptorNode) ObjectTypeQualifiers() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n ObjectTypeDescriptorNode) ObjectKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectTypeDescriptorNode) OpenBrace() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectTypeDescriptorNode) Members() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n ObjectTypeDescriptorNode) CloseBrace() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type ObjectConstructorExpressionNode struct {
	ExpressionNode
}

func (n ObjectConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.OBJECT_CONSTRUCTOR
}

func (n ObjectConstructorExpressionNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n ObjectConstructorExpressionNode) ObjectTypeQualifiers() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ObjectConstructorExpressionNode) ObjectKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectConstructorExpressionNode) TypeReference() TypeDescriptorNode {
	val, ok := n.ChildInBucket(3).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectConstructorExpressionNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectConstructorExpressionNode) Members() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(5)))
}

func (n ObjectConstructorExpressionNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(6).(Token)
	if !ok {
		return nil
	}
	return val
}

type RecordTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n RecordTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.RECORD_TYPE_DESC
}

func (n RecordTypeDescriptorNode) RecordKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordTypeDescriptorNode) BodyStartDelimiter() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordTypeDescriptorNode) Fields() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n RecordTypeDescriptorNode) RecordRestDescriptor() *RecordRestDescriptorNode {
	val, ok := n.ChildInBucket(3).(*RecordRestDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n RecordTypeDescriptorNode) BodyEndDelimiter() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReturnTypeDescriptorNode struct {
	NonTerminalNodeBase
}

func (n ReturnTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.RETURN_TYPE_DESCRIPTOR
}

func (n ReturnTypeDescriptorNode) ReturnsKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReturnTypeDescriptorNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ReturnTypeDescriptorNode) Type() Node {
	return n.ChildInBucket(2)
}

type NilTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n NilTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.NIL_TYPE_DESC
}

func (n NilTypeDescriptorNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NilTypeDescriptorNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type OptionalTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n OptionalTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.OPTIONAL_TYPE_DESC
}

func (n OptionalTypeDescriptorNode) TypeDescriptor() Node {
	return n.ChildInBucket(0)
}

func (n OptionalTypeDescriptorNode) QuestionMarkToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type ObjectFieldNode struct {
	NonTerminalNodeBase
}

func (n ObjectFieldNode) Kind() common.SyntaxKind {
	return common.OBJECT_FIELD
}

func (n ObjectFieldNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectFieldNode) VisibilityQualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectFieldNode) QualifierList() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n ObjectFieldNode) TypeName() Node {
	return n.ChildInBucket(3)
}

func (n ObjectFieldNode) FieldName() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectFieldNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectFieldNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(6).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ObjectFieldNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(7).(Token)
	if !ok {
		return nil
	}
	return val
}

type RecordFieldNode struct {
	NonTerminalNodeBase
}

func (n RecordFieldNode) Kind() common.SyntaxKind {
	return common.RECORD_FIELD
}

func (n RecordFieldNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldNode) ReadonlyKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldNode) TypeName() Node {
	return n.ChildInBucket(2)
}

func (n RecordFieldNode) FieldName() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldNode) QuestionMarkToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

type RecordFieldWithDefaultValueNode struct {
	NonTerminalNodeBase
}

func (n RecordFieldWithDefaultValueNode) Kind() common.SyntaxKind {
	return common.RECORD_FIELD_WITH_DEFAULT_VALUE
}

func (n RecordFieldWithDefaultValueNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldWithDefaultValueNode) ReadonlyKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldWithDefaultValueNode) TypeName() Node {
	return n.ChildInBucket(2)
}

func (n RecordFieldWithDefaultValueNode) FieldName() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldWithDefaultValueNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldWithDefaultValueNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(5).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n RecordFieldWithDefaultValueNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(6).(Token)
	if !ok {
		return nil
	}
	return val
}

type RecordRestDescriptorNode struct {
	NonTerminalNodeBase
}

func (n RecordRestDescriptorNode) Kind() common.SyntaxKind {
	return common.RECORD_REST_TYPE
}

func (n RecordRestDescriptorNode) TypeName() Node {
	return n.ChildInBucket(0)
}

func (n RecordRestDescriptorNode) EllipsisToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RecordRestDescriptorNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypeReferenceNode struct {
	TypeDescriptorNode
}

func (n TypeReferenceNode) Kind() common.SyntaxKind {
	return common.TYPE_REFERENCE
}

func (n TypeReferenceNode) AsteriskToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeReferenceNode) TypeName() Node {
	return n.ChildInBucket(1)
}

func (n TypeReferenceNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type AnnotationNode struct {
	NonTerminalNodeBase
}

func (n AnnotationNode) Kind() common.SyntaxKind {
	return common.ANNOTATION
}

func (n AnnotationNode) AtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationNode) AnnotReference() Node {
	return n.ChildInBucket(1)
}

func (n AnnotationNode) AnnotValue() *MappingConstructorExpressionNode {
	val, ok := n.ChildInBucket(2).(*MappingConstructorExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type MetadataNode struct {
	NonTerminalNodeBase
}

func (n MetadataNode) Kind() common.SyntaxKind {
	return common.METADATA
}

func (n MetadataNode) DocumentationString() Node {
	return n.ChildInBucket(0)
}

func (n MetadataNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

type ModuleVariableDeclarationNode struct {
	ModuleMemberDeclarationNode
}

func (n ModuleVariableDeclarationNode) Kind() common.SyntaxKind {
	return common.MODULE_VAR_DECL
}

func (n ModuleVariableDeclarationNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleVariableDeclarationNode) VisibilityQualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleVariableDeclarationNode) Qualifiers() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n ModuleVariableDeclarationNode) TypedBindingPattern() *TypedBindingPatternNode {
	val, ok := n.ChildInBucket(3).(*TypedBindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleVariableDeclarationNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleVariableDeclarationNode) Initializer() ExpressionNode {
	val, ok := n.ChildInBucket(5).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleVariableDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(6).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypeTestExpressionNode struct {
	ExpressionNode
}

func (n TypeTestExpressionNode) Kind() common.SyntaxKind {
	return common.TYPE_TEST_EXPRESSION
}

func (n TypeTestExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n TypeTestExpressionNode) IsKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeTestExpressionNode) TypeDescriptor() Node {
	return n.ChildInBucket(2)
}

type ActionNode = ExpressionNode

type RemoteMethodCallActionNode struct {
	ActionNode
}

func (n RemoteMethodCallActionNode) Kind() common.SyntaxKind {
	return common.REMOTE_METHOD_CALL_ACTION
}

func (n RemoteMethodCallActionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n RemoteMethodCallActionNode) RightArrowToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RemoteMethodCallActionNode) MethodName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(2).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n RemoteMethodCallActionNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RemoteMethodCallActionNode) Arguments() NodeList[FunctionArgumentNode] {
	return nodeListFrom[FunctionArgumentNode](into[NonTerminalNode](n.ChildInBucket(4)))
}

func (n RemoteMethodCallActionNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

type MapTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n MapTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.MAP_TYPE_DESC
}

func (n MapTypeDescriptorNode) MapKeywordToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MapTypeDescriptorNode) MapTypeParamsNode() *TypeParameterNode {
	val, ok := n.ChildInBucket(1).(*TypeParameterNode)
	if !ok {
		return nil
	}
	return val
}

type NilLiteralNode struct {
	ExpressionNode
}

func (n NilLiteralNode) Kind() common.SyntaxKind {
	return common.NIL_LITERAL
}

func (n NilLiteralNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NilLiteralNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type AnnotationDeclarationNode struct {
	ModuleMemberDeclarationNode
}

func (n AnnotationDeclarationNode) Kind() common.SyntaxKind {
	return common.ANNOTATION_DECLARATION
}

func (n AnnotationDeclarationNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationDeclarationNode) VisibilityQualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationDeclarationNode) ConstKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationDeclarationNode) AnnotationKeyword() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationDeclarationNode) TypeDescriptor() Node {
	return n.ChildInBucket(4)
}

func (n AnnotationDeclarationNode) AnnotationTag() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationDeclarationNode) OnKeyword() Token {
	val, ok := n.ChildInBucket(6).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationDeclarationNode) AttachPoints() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(7)))
}

func (n AnnotationDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(8).(Token)
	if !ok {
		return nil
	}
	return val
}

type AnnotationAttachPointNode struct {
	NonTerminalNodeBase
}

func (n AnnotationAttachPointNode) Kind() common.SyntaxKind {
	return common.ANNOTATION_ATTACH_POINT
}

func (n AnnotationAttachPointNode) SourceKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotationAttachPointNode) Identifiers() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(1)))
}

type XMLNamespaceDeclarationNode struct {
	StatementNode
}

func (n XMLNamespaceDeclarationNode) Kind() common.SyntaxKind {
	return common.XML_NAMESPACE_DECLARATION
}

func (n XMLNamespaceDeclarationNode) XmlnsKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLNamespaceDeclarationNode) Namespaceuri() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLNamespaceDeclarationNode) AsKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLNamespaceDeclarationNode) NamespacePrefix() *IdentifierToken {
	val, ok := n.ChildInBucket(3).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n XMLNamespaceDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type ModuleXMLNamespaceDeclarationNode struct {
	ModuleMemberDeclarationNode
}

func (n ModuleXMLNamespaceDeclarationNode) Kind() common.SyntaxKind {
	return common.MODULE_XML_NAMESPACE_DECLARATION
}

func (n ModuleXMLNamespaceDeclarationNode) XmlnsKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleXMLNamespaceDeclarationNode) Namespaceuri() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleXMLNamespaceDeclarationNode) AsKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleXMLNamespaceDeclarationNode) NamespacePrefix() *IdentifierToken {
	val, ok := n.ChildInBucket(3).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n ModuleXMLNamespaceDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type FunctionBodyBlockNode struct {
	FunctionBodyNode
}

func (n FunctionBodyBlockNode) Kind() common.SyntaxKind {
	return common.FUNCTION_BODY_BLOCK
}

func (n FunctionBodyBlockNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionBodyBlockNode) NamedWorkerDeclarator() *NamedWorkerDeclarator {
	val, ok := n.ChildInBucket(1).(*NamedWorkerDeclarator)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionBodyBlockNode) Statements() NodeList[StatementNode] {
	return nodeListFrom[StatementNode](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n FunctionBodyBlockNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionBodyBlockNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type NamedWorkerDeclarationNode struct {
	NonTerminalNodeBase
}

func (n NamedWorkerDeclarationNode) Kind() common.SyntaxKind {
	return common.NAMED_WORKER_DECLARATION
}

func (n NamedWorkerDeclarationNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n NamedWorkerDeclarationNode) TransactionalKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NamedWorkerDeclarationNode) WorkerKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NamedWorkerDeclarationNode) WorkerName() *IdentifierToken {
	val, ok := n.ChildInBucket(3).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n NamedWorkerDeclarationNode) ReturnTypeDesc() Node {
	return n.ChildInBucket(4)
}

func (n NamedWorkerDeclarationNode) WorkerBody() *BlockStatementNode {
	val, ok := n.ChildInBucket(5).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n NamedWorkerDeclarationNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(6).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type NamedWorkerDeclarator struct {
	NonTerminalNodeBase
}

func (n NamedWorkerDeclarator) Kind() common.SyntaxKind {
	return common.NAMED_WORKER_DECLARATOR
}

func (n NamedWorkerDeclarator) WorkerInitStatements() NodeList[StatementNode] {
	return nodeListFrom[StatementNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n NamedWorkerDeclarator) NamedWorkerDeclarations() NodeList[*NamedWorkerDeclarationNode] {
	return nodeListFrom[*NamedWorkerDeclarationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

type BasicLiteralNode struct {
	ExpressionNode
}

func (n BasicLiteralNode) LiteralToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypeDescriptorNode = ExpressionNode

type NameReferenceNode = TypeDescriptorNode

type SimpleNameReferenceNode struct {
	NameReferenceNode
}

func (n SimpleNameReferenceNode) Kind() common.SyntaxKind {
	return common.SIMPLE_NAME_REFERENCE
}

func (n SimpleNameReferenceNode) Name() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type QualifiedNameReferenceNode struct {
	NameReferenceNode
}

func (n QualifiedNameReferenceNode) Kind() common.SyntaxKind {
	return common.QUALIFIED_NAME_REFERENCE
}

func (n QualifiedNameReferenceNode) ModulePrefix() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n QualifiedNameReferenceNode) Colon() Node {
	return n.ChildInBucket(1)
}

func (n QualifiedNameReferenceNode) Identifier() *IdentifierToken {
	val, ok := n.ChildInBucket(2).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

type BuiltinSimpleNameReferenceNode struct {
	NameReferenceNode
}

func (n BuiltinSimpleNameReferenceNode) Name() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type TrapExpressionNode struct {
	ExpressionNode
}

func (n TrapExpressionNode) TrapKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TrapExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type ListConstructorExpressionNode struct {
	ExpressionNode
}

func (n ListConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.LIST_CONSTRUCTOR
}

func (n ListConstructorExpressionNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ListConstructorExpressionNode) Expressions() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ListConstructorExpressionNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypeCastExpressionNode struct {
	ExpressionNode
}

func (n TypeCastExpressionNode) Kind() common.SyntaxKind {
	return common.TYPE_CAST_EXPRESSION
}

func (n TypeCastExpressionNode) LtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeCastExpressionNode) TypeCastParam() *TypeCastParamNode {
	val, ok := n.ChildInBucket(1).(*TypeCastParamNode)
	if !ok {
		return nil
	}
	return val
}

func (n TypeCastExpressionNode) GtToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeCastExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type TypeCastParamNode struct {
	NonTerminalNodeBase
}

func (n TypeCastParamNode) Kind() common.SyntaxKind {
	return common.TYPE_CAST_PARAM
}

func (n TypeCastParamNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n TypeCastParamNode) Type() Node {
	return n.ChildInBucket(1)
}

type UnionTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n UnionTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.UNION_TYPE_DESC
}

func (n UnionTypeDescriptorNode) LeftTypeDesc() TypeDescriptorNode {
	val, ok := n.ChildInBucket(0).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n UnionTypeDescriptorNode) PipeToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n UnionTypeDescriptorNode) RightTypeDesc() TypeDescriptorNode {
	val, ok := n.ChildInBucket(2).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

type TableConstructorExpressionNode struct {
	ExpressionNode
}

func (n TableConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.TABLE_CONSTRUCTOR
}

func (n TableConstructorExpressionNode) TableKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TableConstructorExpressionNode) KeySpecifier() *KeySpecifierNode {
	val, ok := n.ChildInBucket(1).(*KeySpecifierNode)
	if !ok {
		return nil
	}
	return val
}

func (n TableConstructorExpressionNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TableConstructorExpressionNode) Rows() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n TableConstructorExpressionNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type KeySpecifierNode struct {
	NonTerminalNodeBase
}

func (n KeySpecifierNode) Kind() common.SyntaxKind {
	return common.KEY_SPECIFIER
}

func (n KeySpecifierNode) KeyKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n KeySpecifierNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n KeySpecifierNode) FieldNames() NodeList[*IdentifierToken] {
	return nodeListFrom[*IdentifierToken](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n KeySpecifierNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type StreamTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n StreamTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.STREAM_TYPE_DESC
}

func (n StreamTypeDescriptorNode) StreamKeywordToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n StreamTypeDescriptorNode) StreamTypeParamsNode() Node {
	return n.ChildInBucket(1)
}

type StreamTypeParamsNode struct {
	NonTerminalNodeBase
}

func (n StreamTypeParamsNode) Kind() common.SyntaxKind {
	return common.STREAM_TYPE_PARAMS
}

func (n StreamTypeParamsNode) LtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n StreamTypeParamsNode) LeftTypeDescNode() Node {
	return n.ChildInBucket(1)
}

func (n StreamTypeParamsNode) CommaToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n StreamTypeParamsNode) RightTypeDescNode() Node {
	return n.ChildInBucket(3)
}

func (n StreamTypeParamsNode) GtToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type LetExpressionNode struct {
	ExpressionNode
}

func (n LetExpressionNode) Kind() common.SyntaxKind {
	return common.LET_EXPRESSION
}

func (n LetExpressionNode) LetKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n LetExpressionNode) LetVarDeclarations() NodeList[*LetVariableDeclarationNode] {
	return nodeListFrom[*LetVariableDeclarationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n LetExpressionNode) InKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n LetExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type LetVariableDeclarationNode struct {
	NonTerminalNodeBase
}

func (n LetVariableDeclarationNode) Kind() common.SyntaxKind {
	return common.LET_VAR_DECL
}

func (n LetVariableDeclarationNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n LetVariableDeclarationNode) TypedBindingPattern() *TypedBindingPatternNode {
	val, ok := n.ChildInBucket(1).(*TypedBindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n LetVariableDeclarationNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n LetVariableDeclarationNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type TemplateExpressionNode struct {
	ExpressionNode
}

func (n TemplateExpressionNode) Type() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TemplateExpressionNode) StartBacktick() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TemplateExpressionNode) Content() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n TemplateExpressionNode) EndBacktick() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLItemNode = NonTerminalNode

type XMLElementNode struct {
	XMLItemNode
}

func (n XMLElementNode) Kind() common.SyntaxKind {
	return common.XML_ELEMENT
}

func (n XMLElementNode) StartTag() *XMLStartTagNode {
	val, ok := n.ChildInBucket(0).(*XMLStartTagNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLElementNode) Content() NodeList[XMLItemNode] {
	return nodeListFrom[XMLItemNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n XMLElementNode) EndTag() *XMLEndTagNode {
	val, ok := n.ChildInBucket(2).(*XMLEndTagNode)
	if !ok {
		return nil
	}
	return val
}

type XMLElementTagNode = NonTerminalNode

type XMLStartTagNode struct {
	XMLElementTagNode
}

func (n XMLStartTagNode) Kind() common.SyntaxKind {
	return common.XML_ELEMENT_START_TAG
}

func (n XMLStartTagNode) LtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLStartTagNode) Name() XMLNameNode {
	val, ok := n.ChildInBucket(1).(XMLNameNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLStartTagNode) Attributes() NodeList[*XMLAttributeNode] {
	return nodeListFrom[*XMLAttributeNode](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n XMLStartTagNode) GtToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLEndTagNode struct {
	XMLElementTagNode
}

func (n XMLEndTagNode) Kind() common.SyntaxKind {
	return common.XML_ELEMENT_END_TAG
}

func (n XMLEndTagNode) LtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLEndTagNode) SlashToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLEndTagNode) Name() XMLNameNode {
	val, ok := n.ChildInBucket(2).(XMLNameNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLEndTagNode) GtToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLNameNode = NonTerminalNode

type XMLSimpleNameNode struct {
	XMLNameNode
}

func (n XMLSimpleNameNode) Kind() common.SyntaxKind {
	return common.XML_SIMPLE_NAME
}

func (n XMLSimpleNameNode) Name() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLQualifiedNameNode struct {
	XMLNameNode
}

func (n XMLQualifiedNameNode) Kind() common.SyntaxKind {
	return common.XML_QUALIFIED_NAME
}

func (n XMLQualifiedNameNode) Prefix() *XMLSimpleNameNode {
	val, ok := n.ChildInBucket(0).(*XMLSimpleNameNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLQualifiedNameNode) Colon() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLQualifiedNameNode) Name() *XMLSimpleNameNode {
	val, ok := n.ChildInBucket(2).(*XMLSimpleNameNode)
	if !ok {
		return nil
	}
	return val
}

type XMLEmptyElementNode struct {
	XMLItemNode
}

func (n XMLEmptyElementNode) Kind() common.SyntaxKind {
	return common.XML_EMPTY_ELEMENT
}

func (n XMLEmptyElementNode) LtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLEmptyElementNode) Name() XMLNameNode {
	val, ok := n.ChildInBucket(1).(XMLNameNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLEmptyElementNode) Attributes() NodeList[*XMLAttributeNode] {
	return nodeListFrom[*XMLAttributeNode](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n XMLEmptyElementNode) SlashToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLEmptyElementNode) GtToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type InterpolationNode struct {
	XMLItemNode
}

func (n InterpolationNode) Kind() common.SyntaxKind {
	return common.INTERPOLATION
}

func (n InterpolationNode) InterpolationStartToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n InterpolationNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n InterpolationNode) InterpolationEndToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLTextNode struct {
	XMLItemNode
}

func (n XMLTextNode) Kind() common.SyntaxKind {
	return common.XML_TEXT
}

func (n XMLTextNode) Content() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLAttributeNode struct {
	NonTerminalNodeBase
}

func (n XMLAttributeNode) Kind() common.SyntaxKind {
	return common.XML_ATTRIBUTE
}

func (n XMLAttributeNode) AttributeName() XMLNameNode {
	val, ok := n.ChildInBucket(0).(XMLNameNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLAttributeNode) EqualToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLAttributeNode) Value() *XMLAttributeValue {
	val, ok := n.ChildInBucket(2).(*XMLAttributeValue)
	if !ok {
		return nil
	}
	return val
}

type XMLAttributeValue struct {
	NonTerminalNodeBase
}

func (n XMLAttributeValue) Kind() common.SyntaxKind {
	return common.XML_ATTRIBUTE_VALUE
}

func (n XMLAttributeValue) StartQuote() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLAttributeValue) Value() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n XMLAttributeValue) EndQuote() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLComment struct {
	XMLItemNode
}

func (n XMLComment) Kind() common.SyntaxKind {
	return common.XML_COMMENT
}

func (n XMLComment) CommentStart() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLComment) Content() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n XMLComment) CommentEnd() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLCDATANode struct {
	XMLItemNode
}

func (n XMLCDATANode) Kind() common.SyntaxKind {
	return common.XML_CDATA
}

func (n XMLCDATANode) CdataStart() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLCDATANode) Content() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n XMLCDATANode) CdataEnd() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLProcessingInstruction struct {
	XMLItemNode
}

func (n XMLProcessingInstruction) Kind() common.SyntaxKind {
	return common.XML_PI
}

func (n XMLProcessingInstruction) PiStart() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLProcessingInstruction) Target() XMLNameNode {
	val, ok := n.ChildInBucket(1).(XMLNameNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLProcessingInstruction) Data() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n XMLProcessingInstruction) PiEnd() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type TableTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n TableTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.TABLE_TYPE_DESC
}

func (n TableTypeDescriptorNode) TableKeywordToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TableTypeDescriptorNode) RowTypeParameterNode() Node {
	return n.ChildInBucket(1)
}

func (n TableTypeDescriptorNode) KeyConstraintNode() Node {
	return n.ChildInBucket(2)
}

type TypeParameterNode struct {
	NonTerminalNodeBase
}

func (n TypeParameterNode) Kind() common.SyntaxKind {
	return common.TYPE_PARAMETER
}

func (n TypeParameterNode) LtToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TypeParameterNode) TypeNode() TypeDescriptorNode {
	val, ok := n.ChildInBucket(1).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n TypeParameterNode) GtToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type KeyTypeConstraintNode struct {
	NonTerminalNodeBase
}

func (n KeyTypeConstraintNode) Kind() common.SyntaxKind {
	return common.KEY_TYPE_CONSTRAINT
}

func (n KeyTypeConstraintNode) KeyKeywordToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n KeyTypeConstraintNode) TypeParameterNode() Node {
	return n.ChildInBucket(1)
}

type FunctionTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n FunctionTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.FUNCTION_TYPE_DESC
}

func (n FunctionTypeDescriptorNode) QualifierList() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n FunctionTypeDescriptorNode) FunctionKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionTypeDescriptorNode) FunctionSignature() *FunctionSignatureNode {
	val, ok := n.ChildInBucket(2).(*FunctionSignatureNode)
	if !ok {
		return nil
	}
	return val
}

type FunctionSignatureNode struct {
	NonTerminalNodeBase
}

func (n FunctionSignatureNode) Kind() common.SyntaxKind {
	return common.FUNCTION_SIGNATURE
}

func (n FunctionSignatureNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionSignatureNode) Parameters() NodeList[ParameterNode] {
	return nodeListFrom[ParameterNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n FunctionSignatureNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FunctionSignatureNode) ReturnTypeDesc() *ReturnTypeDescriptorNode {
	val, ok := n.ChildInBucket(3).(*ReturnTypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

type AnonymousFunctionExpressionNode = ExpressionNode

type ExplicitAnonymousFunctionExpressionNode struct {
	AnonymousFunctionExpressionNode
}

func (n ExplicitAnonymousFunctionExpressionNode) Kind() common.SyntaxKind {
	return common.EXPLICIT_ANONYMOUS_FUNCTION_EXPRESSION
}

func (n ExplicitAnonymousFunctionExpressionNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n ExplicitAnonymousFunctionExpressionNode) QualifierList() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ExplicitAnonymousFunctionExpressionNode) FunctionKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ExplicitAnonymousFunctionExpressionNode) FunctionSignature() *FunctionSignatureNode {
	val, ok := n.ChildInBucket(3).(*FunctionSignatureNode)
	if !ok {
		return nil
	}
	return val
}

func (n ExplicitAnonymousFunctionExpressionNode) FunctionBody() FunctionBodyNode {
	val, ok := n.ChildInBucket(4).(FunctionBodyNode)
	if !ok {
		return nil
	}
	return val
}

type FunctionBodyNode = NonTerminalNode

type ExpressionFunctionBodyNode struct {
	FunctionBodyNode
}

func (n ExpressionFunctionBodyNode) Kind() common.SyntaxKind {
	return common.EXPRESSION_FUNCTION_BODY
}

func (n ExpressionFunctionBodyNode) RightDoubleArrow() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ExpressionFunctionBodyNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ExpressionFunctionBodyNode) Semicolon() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type TupleTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n TupleTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.TUPLE_TYPE_DESC
}

func (n TupleTypeDescriptorNode) OpenBracketToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TupleTypeDescriptorNode) MemberTypeDesc() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n TupleTypeDescriptorNode) CloseBracketToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ParenthesisedTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n ParenthesisedTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.PARENTHESISED_TYPE_DESC
}

func (n ParenthesisedTypeDescriptorNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ParenthesisedTypeDescriptorNode) Typedesc() TypeDescriptorNode {
	val, ok := n.ChildInBucket(1).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ParenthesisedTypeDescriptorNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type NewExpressionNode = ExpressionNode

type ExplicitNewExpressionNode struct {
	NewExpressionNode
}

func (n ExplicitNewExpressionNode) Kind() common.SyntaxKind {
	return common.EXPLICIT_NEW_EXPRESSION
}

func (n ExplicitNewExpressionNode) NewKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ExplicitNewExpressionNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(1).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ExplicitNewExpressionNode) ParenthesizedArgList() *ParenthesizedArgList {
	val, ok := n.ChildInBucket(2).(*ParenthesizedArgList)
	if !ok {
		return nil
	}
	return val
}

type ImplicitNewExpressionNode struct {
	NewExpressionNode
}

func (n ImplicitNewExpressionNode) Kind() common.SyntaxKind {
	return common.IMPLICIT_NEW_EXPRESSION
}

func (n ImplicitNewExpressionNode) NewKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ImplicitNewExpressionNode) ParenthesizedArgList() *ParenthesizedArgList {
	val, ok := n.ChildInBucket(1).(*ParenthesizedArgList)
	if !ok {
		return nil
	}
	return val
}

type ParenthesizedArgList struct {
	NonTerminalNodeBase
}

func (n ParenthesizedArgList) Kind() common.SyntaxKind {
	return common.PARENTHESIZED_ARG_LIST
}

func (n ParenthesizedArgList) OpenParenToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ParenthesizedArgList) Arguments() NodeList[FunctionArgumentNode] {
	return nodeListFrom[FunctionArgumentNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ParenthesizedArgList) CloseParenToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ClauseNode = NonTerminalNode

type IntermediateClauseNode = ClauseNode

type QueryConstructTypeNode struct {
	NonTerminalNodeBase
}

func (n QueryConstructTypeNode) Kind() common.SyntaxKind {
	return common.QUERY_CONSTRUCT_TYPE
}

func (n QueryConstructTypeNode) Keyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n QueryConstructTypeNode) KeySpecifier() *KeySpecifierNode {
	val, ok := n.ChildInBucket(1).(*KeySpecifierNode)
	if !ok {
		return nil
	}
	return val
}

type FromClauseNode struct {
	IntermediateClauseNode
}

func (n FromClauseNode) Kind() common.SyntaxKind {
	return common.FROM_CLAUSE
}

func (n FromClauseNode) FromKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FromClauseNode) TypedBindingPattern() *TypedBindingPatternNode {
	val, ok := n.ChildInBucket(1).(*TypedBindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n FromClauseNode) InKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FromClauseNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type WhereClauseNode struct {
	IntermediateClauseNode
}

func (n WhereClauseNode) Kind() common.SyntaxKind {
	return common.WHERE_CLAUSE
}

func (n WhereClauseNode) WhereKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n WhereClauseNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type LetClauseNode struct {
	IntermediateClauseNode
}

func (n LetClauseNode) Kind() common.SyntaxKind {
	return common.LET_CLAUSE
}

func (n LetClauseNode) LetKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n LetClauseNode) LetVarDeclarations() NodeList[*LetVariableDeclarationNode] {
	return nodeListFrom[*LetVariableDeclarationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

type JoinClauseNode struct {
	IntermediateClauseNode
}

func (n JoinClauseNode) Kind() common.SyntaxKind {
	return common.JOIN_CLAUSE
}

func (n JoinClauseNode) OuterKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n JoinClauseNode) JoinKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n JoinClauseNode) TypedBindingPattern() *TypedBindingPatternNode {
	val, ok := n.ChildInBucket(2).(*TypedBindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n JoinClauseNode) InKeyword() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n JoinClauseNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(4).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n JoinClauseNode) JoinOnCondition() *OnClauseNode {
	val, ok := n.ChildInBucket(5).(*OnClauseNode)
	if !ok {
		return nil
	}
	return val
}

type OnClauseNode struct {
	ClauseNode
}

func (n OnClauseNode) Kind() common.SyntaxKind {
	return common.ON_CLAUSE
}

func (n OnClauseNode) OnKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OnClauseNode) OnExpression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n OnClauseNode) EqualsKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OnClauseNode) EqualsExpression() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type LimitClauseNode struct {
	IntermediateClauseNode
}

func (n LimitClauseNode) Kind() common.SyntaxKind {
	return common.LIMIT_CLAUSE
}

func (n LimitClauseNode) LimitKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n LimitClauseNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type OnConflictClauseNode struct {
	ClauseNode
}

func (n OnConflictClauseNode) Kind() common.SyntaxKind {
	return common.ON_CONFLICT_CLAUSE
}

func (n OnConflictClauseNode) OnKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OnConflictClauseNode) ConflictKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OnConflictClauseNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type QueryPipelineNode struct {
	NonTerminalNodeBase
}

func (n QueryPipelineNode) Kind() common.SyntaxKind {
	return common.QUERY_PIPELINE
}

func (n QueryPipelineNode) FromClause() *FromClauseNode {
	val, ok := n.ChildInBucket(0).(*FromClauseNode)
	if !ok {
		return nil
	}
	return val
}

func (n QueryPipelineNode) IntermediateClauses() NodeList[IntermediateClauseNode] {
	return nodeListFrom[IntermediateClauseNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

type SelectClauseNode struct {
	ClauseNode
}

func (n SelectClauseNode) Kind() common.SyntaxKind {
	return common.SELECT_CLAUSE
}

func (n SelectClauseNode) SelectKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n SelectClauseNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type CollectClauseNode struct {
	ClauseNode
}

func (n CollectClauseNode) Kind() common.SyntaxKind {
	return common.COLLECT_CLAUSE
}

func (n CollectClauseNode) CollectKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n CollectClauseNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type QueryExpressionNode struct {
	ExpressionNode
}

func (n QueryExpressionNode) Kind() common.SyntaxKind {
	return common.QUERY_EXPRESSION
}

func (n QueryExpressionNode) QueryConstructType() *QueryConstructTypeNode {
	val, ok := n.ChildInBucket(0).(*QueryConstructTypeNode)
	if !ok {
		return nil
	}
	return val
}

func (n QueryExpressionNode) QueryPipeline() *QueryPipelineNode {
	val, ok := n.ChildInBucket(1).(*QueryPipelineNode)
	if !ok {
		return nil
	}
	return val
}

func (n QueryExpressionNode) ResultClause() ClauseNode {
	val, ok := n.ChildInBucket(2).(ClauseNode)
	if !ok {
		return nil
	}
	return val
}

func (n QueryExpressionNode) OnConflictClause() *OnConflictClauseNode {
	val, ok := n.ChildInBucket(3).(*OnConflictClauseNode)
	if !ok {
		return nil
	}
	return val
}

type QueryActionNode struct {
	ActionNode
}

func (n QueryActionNode) Kind() common.SyntaxKind {
	return common.QUERY_ACTION
}

func (n QueryActionNode) QueryPipeline() *QueryPipelineNode {
	val, ok := n.ChildInBucket(0).(*QueryPipelineNode)
	if !ok {
		return nil
	}
	return val
}

func (n QueryActionNode) DoKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n QueryActionNode) BlockStatement() *BlockStatementNode {
	val, ok := n.ChildInBucket(2).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

type IntersectionTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n IntersectionTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.INTERSECTION_TYPE_DESC
}

func (n IntersectionTypeDescriptorNode) LeftTypeDesc() Node {
	return n.ChildInBucket(0)
}

func (n IntersectionTypeDescriptorNode) BitwiseAndToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n IntersectionTypeDescriptorNode) RightTypeDesc() Node {
	return n.ChildInBucket(2)
}

type ImplicitAnonymousFunctionParameters struct {
	NonTerminalNodeBase
}

func (n ImplicitAnonymousFunctionParameters) Kind() common.SyntaxKind {
	return common.INFER_PARAM_LIST
}

func (n ImplicitAnonymousFunctionParameters) OpenParenToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ImplicitAnonymousFunctionParameters) Parameters() NodeList[*SimpleNameReferenceNode] {
	return nodeListFrom[*SimpleNameReferenceNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ImplicitAnonymousFunctionParameters) CloseParenToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ImplicitAnonymousFunctionExpressionNode struct {
	AnonymousFunctionExpressionNode
}

func (n ImplicitAnonymousFunctionExpressionNode) Kind() common.SyntaxKind {
	return common.IMPLICIT_ANONYMOUS_FUNCTION_EXPRESSION
}

func (n ImplicitAnonymousFunctionExpressionNode) Params() Node {
	return n.ChildInBucket(0)
}

func (n ImplicitAnonymousFunctionExpressionNode) RightDoubleArrow() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ImplicitAnonymousFunctionExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type StartActionNode struct {
	ExpressionNode
}

func (n StartActionNode) Kind() common.SyntaxKind {
	return common.START_ACTION
}

func (n StartActionNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n StartActionNode) StartKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n StartActionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type FlushActionNode struct {
	ExpressionNode
}

func (n FlushActionNode) Kind() common.SyntaxKind {
	return common.FLUSH_ACTION
}

func (n FlushActionNode) FlushKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FlushActionNode) PeerWorker() NameReferenceNode {
	val, ok := n.ChildInBucket(1).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type SingletonTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n SingletonTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.SINGLETON_TYPE_DESC
}

func (n SingletonTypeDescriptorNode) SimpleContExprNode() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type MethodDeclarationNode struct {
	NonTerminalNodeBase
}

func (n MethodDeclarationNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n MethodDeclarationNode) QualifierList() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n MethodDeclarationNode) FunctionKeyword() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MethodDeclarationNode) MethodName() *IdentifierToken {
	val, ok := n.ChildInBucket(3).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n MethodDeclarationNode) RelativeResourcePath() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(4)))
}

func (n MethodDeclarationNode) MethodSignature() *FunctionSignatureNode {
	val, ok := n.ChildInBucket(5).(*FunctionSignatureNode)
	if !ok {
		return nil
	}
	return val
}

func (n MethodDeclarationNode) Semicolon() Token {
	val, ok := n.ChildInBucket(6).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypedBindingPatternNode struct {
	NonTerminalNodeBase
}

func (n TypedBindingPatternNode) Kind() common.SyntaxKind {
	return common.TYPED_BINDING_PATTERN
}

func (n TypedBindingPatternNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(0).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n TypedBindingPatternNode) BindingPattern() BindingPatternNode {
	val, ok := n.ChildInBucket(1).(BindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

type BindingPatternNode = NonTerminalNode

type CaptureBindingPatternNode struct {
	BindingPatternNode
}

func (n CaptureBindingPatternNode) Kind() common.SyntaxKind {
	return common.CAPTURE_BINDING_PATTERN
}

func (n CaptureBindingPatternNode) VariableName() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type WildcardBindingPatternNode struct {
	BindingPatternNode
}

func (n WildcardBindingPatternNode) Kind() common.SyntaxKind {
	return common.WILDCARD_BINDING_PATTERN
}

func (n WildcardBindingPatternNode) UnderscoreToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type ListBindingPatternNode struct {
	BindingPatternNode
}

func (n ListBindingPatternNode) Kind() common.SyntaxKind {
	return common.LIST_BINDING_PATTERN
}

func (n ListBindingPatternNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ListBindingPatternNode) BindingPatterns() NodeList[BindingPatternNode] {
	return nodeListFrom[BindingPatternNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ListBindingPatternNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type MappingBindingPatternNode struct {
	BindingPatternNode
}

func (n MappingBindingPatternNode) Kind() common.SyntaxKind {
	return common.MAPPING_BINDING_PATTERN
}

func (n MappingBindingPatternNode) OpenBrace() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MappingBindingPatternNode) FieldBindingPatterns() NodeList[BindingPatternNode] {
	return nodeListFrom[BindingPatternNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n MappingBindingPatternNode) CloseBrace() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type FieldBindingPatternNode = BindingPatternNode

type FieldBindingPatternFullNode struct {
	FieldBindingPatternNode
}

func (n FieldBindingPatternFullNode) Kind() common.SyntaxKind {
	return common.FIELD_BINDING_PATTERN
}

func (n FieldBindingPatternFullNode) VariableName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(0).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n FieldBindingPatternFullNode) Colon() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FieldBindingPatternFullNode) BindingPattern() BindingPatternNode {
	val, ok := n.ChildInBucket(2).(BindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

type FieldBindingPatternVarnameNode struct {
	FieldBindingPatternNode
}

func (n FieldBindingPatternVarnameNode) Kind() common.SyntaxKind {
	return common.FIELD_BINDING_PATTERN
}

func (n FieldBindingPatternVarnameNode) VariableName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(0).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type RestBindingPatternNode struct {
	BindingPatternNode
}

func (n RestBindingPatternNode) Kind() common.SyntaxKind {
	return common.REST_BINDING_PATTERN
}

func (n RestBindingPatternNode) EllipsisToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RestBindingPatternNode) VariableName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(1).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type ErrorBindingPatternNode struct {
	BindingPatternNode
}

func (n ErrorBindingPatternNode) Kind() common.SyntaxKind {
	return common.ERROR_BINDING_PATTERN
}

func (n ErrorBindingPatternNode) ErrorKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorBindingPatternNode) TypeReference() Node {
	return n.ChildInBucket(1)
}

func (n ErrorBindingPatternNode) OpenParenthesis() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorBindingPatternNode) ArgListBindingPatterns() NodeList[BindingPatternNode] {
	return nodeListFrom[BindingPatternNode](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n ErrorBindingPatternNode) CloseParenthesis() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type NamedArgBindingPatternNode struct {
	BindingPatternNode
}

func (n NamedArgBindingPatternNode) Kind() common.SyntaxKind {
	return common.NAMED_ARG_BINDING_PATTERN
}

func (n NamedArgBindingPatternNode) ArgName() *IdentifierToken {
	val, ok := n.ChildInBucket(0).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n NamedArgBindingPatternNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NamedArgBindingPatternNode) BindingPattern() BindingPatternNode {
	val, ok := n.ChildInBucket(2).(BindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

type AsyncSendActionNode struct {
	ActionNode
}

func (n AsyncSendActionNode) Kind() common.SyntaxKind {
	return common.ASYNC_SEND_ACTION
}

func (n AsyncSendActionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n AsyncSendActionNode) RightArrowToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AsyncSendActionNode) PeerWorker() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(2).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type SyncSendActionNode struct {
	ActionNode
}

func (n SyncSendActionNode) Kind() common.SyntaxKind {
	return common.SYNC_SEND_ACTION
}

func (n SyncSendActionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n SyncSendActionNode) SyncSendToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n SyncSendActionNode) PeerWorker() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(2).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type ReceiveActionNode struct {
	ActionNode
}

func (n ReceiveActionNode) Kind() common.SyntaxKind {
	return common.RECEIVE_ACTION
}

func (n ReceiveActionNode) LeftArrow() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReceiveActionNode) ReceiveWorkers() Node {
	return n.ChildInBucket(1)
}

type ReceiveFieldsNode struct {
	NonTerminalNodeBase
}

func (n ReceiveFieldsNode) Kind() common.SyntaxKind {
	return common.RECEIVE_FIELDS
}

func (n ReceiveFieldsNode) OpenBrace() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReceiveFieldsNode) ReceiveFields() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ReceiveFieldsNode) CloseBrace() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type AlternateReceiveNode struct {
	NonTerminalNodeBase
}

func (n AlternateReceiveNode) Kind() common.SyntaxKind {
	return common.ALTERNATE_RECEIVE
}

func (n AlternateReceiveNode) Workers() NodeList[*SimpleNameReferenceNode] {
	return nodeListFrom[*SimpleNameReferenceNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

type RestDescriptorNode struct {
	NonTerminalNodeBase
}

func (n RestDescriptorNode) Kind() common.SyntaxKind {
	return common.REST_TYPE
}

func (n RestDescriptorNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(0).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n RestDescriptorNode) EllipsisToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type DoubleGTTokenNode struct {
	NonTerminalNodeBase
}

func (n DoubleGTTokenNode) Kind() common.SyntaxKind {
	return common.DOUBLE_GT_TOKEN
}

func (n DoubleGTTokenNode) OpenGTToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n DoubleGTTokenNode) EndGTToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type TrippleGTTokenNode struct {
	NonTerminalNodeBase
}

func (n TrippleGTTokenNode) Kind() common.SyntaxKind {
	return common.TRIPPLE_GT_TOKEN
}

func (n TrippleGTTokenNode) OpenGTToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TrippleGTTokenNode) MiddleGTToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TrippleGTTokenNode) EndGTToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type WaitActionNode struct {
	ActionNode
}

func (n WaitActionNode) Kind() common.SyntaxKind {
	return common.WAIT_ACTION
}

func (n WaitActionNode) WaitKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n WaitActionNode) WaitFutureExpr() Node {
	return n.ChildInBucket(1)
}

type WaitFieldsListNode struct {
	NonTerminalNodeBase
}

func (n WaitFieldsListNode) Kind() common.SyntaxKind {
	return common.WAIT_FIELDS_LIST
}

func (n WaitFieldsListNode) OpenBrace() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n WaitFieldsListNode) WaitFields() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n WaitFieldsListNode) CloseBrace() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type WaitFieldNode struct {
	NonTerminalNodeBase
}

func (n WaitFieldNode) Kind() common.SyntaxKind {
	return common.WAIT_FIELD
}

func (n WaitFieldNode) FieldName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(0).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n WaitFieldNode) Colon() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n WaitFieldNode) WaitFutureExpr() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type AnnotAccessExpressionNode struct {
	ExpressionNode
}

func (n AnnotAccessExpressionNode) Kind() common.SyntaxKind {
	return common.ANNOT_ACCESS
}

func (n AnnotAccessExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotAccessExpressionNode) AnnotChainingToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n AnnotAccessExpressionNode) AnnotTagReference() NameReferenceNode {
	val, ok := n.ChildInBucket(2).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type OptionalFieldAccessExpressionNode struct {
	ExpressionNode
}

func (n OptionalFieldAccessExpressionNode) Kind() common.SyntaxKind {
	return common.OPTIONAL_FIELD_ACCESS
}

func (n OptionalFieldAccessExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n OptionalFieldAccessExpressionNode) OptionalChainingToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OptionalFieldAccessExpressionNode) FieldName() NameReferenceNode {
	val, ok := n.ChildInBucket(2).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type ConditionalExpressionNode struct {
	ExpressionNode
}

func (n ConditionalExpressionNode) Kind() common.SyntaxKind {
	return common.CONDITIONAL_EXPRESSION
}

func (n ConditionalExpressionNode) LhsExpression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ConditionalExpressionNode) QuestionMarkToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ConditionalExpressionNode) MiddleExpression() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ConditionalExpressionNode) ColonToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ConditionalExpressionNode) EndExpression() ExpressionNode {
	val, ok := n.ChildInBucket(4).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type EnumDeclarationNode struct {
	ModuleMemberDeclarationNode
}

func (n EnumDeclarationNode) Kind() common.SyntaxKind {
	return common.ENUM_DECLARATION
}

func (n EnumDeclarationNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n EnumDeclarationNode) Qualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n EnumDeclarationNode) EnumKeywordToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n EnumDeclarationNode) Identifier() *IdentifierToken {
	val, ok := n.ChildInBucket(3).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n EnumDeclarationNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n EnumDeclarationNode) EnumMemberList() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(5)))
}

func (n EnumDeclarationNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(6).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n EnumDeclarationNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(7).(Token)
	if !ok {
		return nil
	}
	return val
}

type EnumMemberNode struct {
	NonTerminalNodeBase
}

func (n EnumMemberNode) Kind() common.SyntaxKind {
	return common.ENUM_MEMBER
}

func (n EnumMemberNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n EnumMemberNode) Identifier() *IdentifierToken {
	val, ok := n.ChildInBucket(1).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n EnumMemberNode) EqualToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n EnumMemberNode) ConstExprNode() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type ArrayTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n ArrayTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.ARRAY_TYPE_DESC
}

func (n ArrayTypeDescriptorNode) MemberTypeDesc() TypeDescriptorNode {
	val, ok := n.ChildInBucket(0).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ArrayTypeDescriptorNode) Dimensions() NodeList[*ArrayDimensionNode] {
	return nodeListFrom[*ArrayDimensionNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

type ArrayDimensionNode struct {
	NonTerminalNodeBase
}

func (n ArrayDimensionNode) Kind() common.SyntaxKind {
	return common.ARRAY_DIMENSION
}

func (n ArrayDimensionNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ArrayDimensionNode) ArrayLength() Node {
	return n.ChildInBucket(1)
}

func (n ArrayDimensionNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type TransactionStatementNode struct {
	StatementNode
}

func (n TransactionStatementNode) Kind() common.SyntaxKind {
	return common.TRANSACTION_STATEMENT
}

func (n TransactionStatementNode) TransactionKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n TransactionStatementNode) BlockStatement() *BlockStatementNode {
	val, ok := n.ChildInBucket(1).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n TransactionStatementNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(2).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type RollbackStatementNode struct {
	StatementNode
}

func (n RollbackStatementNode) Kind() common.SyntaxKind {
	return common.ROLLBACK_STATEMENT
}

func (n RollbackStatementNode) RollbackKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RollbackStatementNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n RollbackStatementNode) Semicolon() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type RetryStatementNode struct {
	StatementNode
}

func (n RetryStatementNode) Kind() common.SyntaxKind {
	return common.RETRY_STATEMENT
}

func (n RetryStatementNode) RetryKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RetryStatementNode) TypeParameter() *TypeParameterNode {
	val, ok := n.ChildInBucket(1).(*TypeParameterNode)
	if !ok {
		return nil
	}
	return val
}

func (n RetryStatementNode) Arguments() *ParenthesizedArgList {
	val, ok := n.ChildInBucket(2).(*ParenthesizedArgList)
	if !ok {
		return nil
	}
	return val
}

func (n RetryStatementNode) RetryBody() StatementNode {
	val, ok := n.ChildInBucket(3).(StatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n RetryStatementNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(4).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type CommitActionNode struct {
	ActionNode
}

func (n CommitActionNode) Kind() common.SyntaxKind {
	return common.COMMIT_ACTION
}

func (n CommitActionNode) CommitKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type TransactionalExpressionNode struct {
	ExpressionNode
}

func (n TransactionalExpressionNode) Kind() common.SyntaxKind {
	return common.TRANSACTIONAL_EXPRESSION
}

func (n TransactionalExpressionNode) TransactionalKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type ByteArrayLiteralNode struct {
	ExpressionNode
}

func (n ByteArrayLiteralNode) Kind() common.SyntaxKind {
	return common.BYTE_ARRAY_LITERAL
}

func (n ByteArrayLiteralNode) Type() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ByteArrayLiteralNode) StartBacktick() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ByteArrayLiteralNode) Content() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ByteArrayLiteralNode) EndBacktick() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLNavigateExpressionNode = ExpressionNode

type XMLFilterExpressionNode struct {
	XMLNavigateExpressionNode
}

func (n XMLFilterExpressionNode) Kind() common.SyntaxKind {
	return common.XML_FILTER_EXPRESSION
}

func (n XMLFilterExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLFilterExpressionNode) XmlPatternChain() *XMLNamePatternChainingNode {
	val, ok := n.ChildInBucket(1).(*XMLNamePatternChainingNode)
	if !ok {
		return nil
	}
	return val
}

type XMLStepExpressionNode struct {
	XMLNavigateExpressionNode
}

func (n XMLStepExpressionNode) Kind() common.SyntaxKind {
	return common.XML_STEP_EXPRESSION
}

func (n XMLStepExpressionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLStepExpressionNode) XmlStepStart() Node {
	return n.ChildInBucket(1)
}

func (n XMLStepExpressionNode) XmlStepExtend() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(2)))
}

type XMLNamePatternChainingNode struct {
	NonTerminalNodeBase
}

func (n XMLNamePatternChainingNode) Kind() common.SyntaxKind {
	return common.XML_NAME_PATTERN_CHAIN
}

func (n XMLNamePatternChainingNode) StartToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLNamePatternChainingNode) XmlNamePattern() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n XMLNamePatternChainingNode) GtToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLStepIndexedExtendNode struct {
	NonTerminalNodeBase
}

func (n XMLStepIndexedExtendNode) Kind() common.SyntaxKind {
	return common.XML_STEP_INDEXED_EXTEND
}

func (n XMLStepIndexedExtendNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLStepIndexedExtendNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLStepIndexedExtendNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type XMLStepMethodCallExtendNode struct {
	NonTerminalNodeBase
}

func (n XMLStepMethodCallExtendNode) Kind() common.SyntaxKind {
	return common.XML_STEP_METHOD_CALL_EXTEND
}

func (n XMLStepMethodCallExtendNode) DotToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLStepMethodCallExtendNode) MethodName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(1).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n XMLStepMethodCallExtendNode) ParenthesizedArgList() *ParenthesizedArgList {
	val, ok := n.ChildInBucket(2).(*ParenthesizedArgList)
	if !ok {
		return nil
	}
	return val
}

type XMLAtomicNamePatternNode struct {
	NonTerminalNodeBase
}

func (n XMLAtomicNamePatternNode) Kind() common.SyntaxKind {
	return common.XML_ATOMIC_NAME_PATTERN
}

func (n XMLAtomicNamePatternNode) Prefix() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLAtomicNamePatternNode) Colon() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n XMLAtomicNamePatternNode) Name() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type TypeReferenceTypeDescNode struct {
	TypeDescriptorNode
}

func (n TypeReferenceTypeDescNode) Kind() common.SyntaxKind {
	return common.TYPE_REFERENCE_TYPE_DESC
}

func (n TypeReferenceTypeDescNode) TypeRef() NameReferenceNode {
	val, ok := n.ChildInBucket(0).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type MatchStatementNode struct {
	StatementNode
}

func (n MatchStatementNode) Kind() common.SyntaxKind {
	return common.MATCH_STATEMENT
}

func (n MatchStatementNode) MatchKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MatchStatementNode) Condition() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n MatchStatementNode) OpenBrace() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MatchStatementNode) MatchClauses() NodeList[*MatchClauseNode] {
	return nodeListFrom[*MatchClauseNode](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n MatchStatementNode) CloseBrace() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MatchStatementNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(5).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type MatchClauseNode struct {
	NonTerminalNodeBase
}

func (n MatchClauseNode) Kind() common.SyntaxKind {
	return common.MATCH_CLAUSE
}

func (n MatchClauseNode) MatchPatterns() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n MatchClauseNode) MatchGuard() *MatchGuardNode {
	val, ok := n.ChildInBucket(1).(*MatchGuardNode)
	if !ok {
		return nil
	}
	return val
}

func (n MatchClauseNode) RightDoubleArrow() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MatchClauseNode) BlockStatement() *BlockStatementNode {
	val, ok := n.ChildInBucket(3).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

type MatchGuardNode struct {
	NonTerminalNodeBase
}

func (n MatchGuardNode) Kind() common.SyntaxKind {
	return common.MATCH_GUARD
}

func (n MatchGuardNode) IfKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MatchGuardNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type DistinctTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n DistinctTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.DISTINCT_TYPE_DESC
}

func (n DistinctTypeDescriptorNode) DistinctKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n DistinctTypeDescriptorNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(1).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

type ListMatchPatternNode struct {
	NonTerminalNodeBase
}

func (n ListMatchPatternNode) Kind() common.SyntaxKind {
	return common.LIST_MATCH_PATTERN
}

func (n ListMatchPatternNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ListMatchPatternNode) MatchPatterns() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ListMatchPatternNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type RestMatchPatternNode struct {
	NonTerminalNodeBase
}

func (n RestMatchPatternNode) Kind() common.SyntaxKind {
	return common.REST_MATCH_PATTERN
}

func (n RestMatchPatternNode) EllipsisToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RestMatchPatternNode) VarKeywordToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n RestMatchPatternNode) VariableName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(2).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type MappingMatchPatternNode struct {
	NonTerminalNodeBase
}

func (n MappingMatchPatternNode) Kind() common.SyntaxKind {
	return common.MAPPING_MATCH_PATTERN
}

func (n MappingMatchPatternNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MappingMatchPatternNode) FieldMatchPatterns() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n MappingMatchPatternNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type FieldMatchPatternNode struct {
	NonTerminalNodeBase
}

func (n FieldMatchPatternNode) Kind() common.SyntaxKind {
	return common.FIELD_MATCH_PATTERN
}

func (n FieldMatchPatternNode) FieldNameNode() *IdentifierToken {
	val, ok := n.ChildInBucket(0).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n FieldMatchPatternNode) ColonToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n FieldMatchPatternNode) MatchPattern() Node {
	return n.ChildInBucket(2)
}

type ErrorMatchPatternNode struct {
	NonTerminalNodeBase
}

func (n ErrorMatchPatternNode) Kind() common.SyntaxKind {
	return common.ERROR_MATCH_PATTERN
}

func (n ErrorMatchPatternNode) ErrorKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorMatchPatternNode) TypeReference() NameReferenceNode {
	val, ok := n.ChildInBucket(1).(NameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorMatchPatternNode) OpenParenthesisToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorMatchPatternNode) ArgListMatchPatternNode() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n ErrorMatchPatternNode) CloseParenthesisToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type NamedArgMatchPatternNode struct {
	NonTerminalNodeBase
}

func (n NamedArgMatchPatternNode) Kind() common.SyntaxKind {
	return common.NAMED_ARG_MATCH_PATTERN
}

func (n NamedArgMatchPatternNode) Identifier() *IdentifierToken {
	val, ok := n.ChildInBucket(0).(*IdentifierToken)
	if !ok {
		return nil
	}
	return val
}

func (n NamedArgMatchPatternNode) EqualToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NamedArgMatchPatternNode) MatchPattern() Node {
	return n.ChildInBucket(2)
}

type DocumentationNode = NonTerminalNode

type MarkdownDocumentationNode struct {
	DocumentationNode
}

func (n MarkdownDocumentationNode) Kind() common.SyntaxKind {
	return common.MARKDOWN_DOCUMENTATION
}

func (n MarkdownDocumentationNode) DocumentationLines() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(0)))
}

type MarkdownDocumentationLineNode struct {
	DocumentationNode
}

func (n MarkdownDocumentationLineNode) HashToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownDocumentationLineNode) DocumentElements() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

type MarkdownParameterDocumentationLineNode struct {
	DocumentationNode
}

func (n MarkdownParameterDocumentationLineNode) HashToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownParameterDocumentationLineNode) PlusToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownParameterDocumentationLineNode) ParameterName() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownParameterDocumentationLineNode) MinusToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownParameterDocumentationLineNode) DocumentElements() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(4)))
}

type BallerinaNameReferenceNode struct {
	DocumentationNode
}

func (n BallerinaNameReferenceNode) Kind() common.SyntaxKind {
	return common.BALLERINA_NAME_REFERENCE
}

func (n BallerinaNameReferenceNode) ReferenceType() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n BallerinaNameReferenceNode) StartBacktick() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n BallerinaNameReferenceNode) NameReference() Node {
	return n.ChildInBucket(2)
}

func (n BallerinaNameReferenceNode) EndBacktick() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type InlineCodeReferenceNode struct {
	DocumentationNode
}

func (n InlineCodeReferenceNode) Kind() common.SyntaxKind {
	return common.INLINE_CODE_REFERENCE
}

func (n InlineCodeReferenceNode) StartBacktick() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n InlineCodeReferenceNode) CodeReference() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n InlineCodeReferenceNode) EndBacktick() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type MarkdownCodeBlockNode struct {
	DocumentationNode
}

func (n MarkdownCodeBlockNode) Kind() common.SyntaxKind {
	return common.MARKDOWN_CODE_BLOCK
}

func (n MarkdownCodeBlockNode) StartLineHashToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownCodeBlockNode) StartBacktick() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownCodeBlockNode) LangAttribute() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownCodeBlockNode) CodeLines() NodeList[*MarkdownCodeLineNode] {
	return nodeListFrom[*MarkdownCodeLineNode](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n MarkdownCodeBlockNode) EndLineHashToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownCodeBlockNode) EndBacktick() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

type MarkdownCodeLineNode struct {
	DocumentationNode
}

func (n MarkdownCodeLineNode) Kind() common.SyntaxKind {
	return common.MARKDOWN_CODE_LINE
}

func (n MarkdownCodeLineNode) HashToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n MarkdownCodeLineNode) CodeDescription() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type OrderByClauseNode struct {
	IntermediateClauseNode
}

func (n OrderByClauseNode) Kind() common.SyntaxKind {
	return common.ORDER_BY_CLAUSE
}

func (n OrderByClauseNode) OrderKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OrderByClauseNode) ByKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OrderByClauseNode) OrderKey() NodeList[*OrderKeyNode] {
	return nodeListFrom[*OrderKeyNode](into[NonTerminalNode](n.ChildInBucket(2)))
}

type OrderKeyNode struct {
	NonTerminalNodeBase
}

func (n OrderKeyNode) Kind() common.SyntaxKind {
	return common.ORDER_KEY
}

func (n OrderKeyNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n OrderKeyNode) OrderDirection() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type GroupByClauseNode struct {
	IntermediateClauseNode
}

func (n GroupByClauseNode) Kind() common.SyntaxKind {
	return common.GROUP_BY_CLAUSE
}

func (n GroupByClauseNode) GroupKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n GroupByClauseNode) ByKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n GroupByClauseNode) GroupingKey() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(2)))
}

type GroupingKeyVarDeclarationNode struct {
	NonTerminalNodeBase
}

func (n GroupingKeyVarDeclarationNode) Kind() common.SyntaxKind {
	return common.GROUPING_KEY_VAR_DECLARATION
}

func (n GroupingKeyVarDeclarationNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(0).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n GroupingKeyVarDeclarationNode) SimpleBindingPattern() BindingPatternNode {
	val, ok := n.ChildInBucket(1).(BindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n GroupingKeyVarDeclarationNode) EqualsToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n GroupingKeyVarDeclarationNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(3).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type OnFailClauseNode struct {
	ClauseNode
}

func (n OnFailClauseNode) Kind() common.SyntaxKind {
	return common.ON_FAIL_CLAUSE
}

func (n OnFailClauseNode) OnKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OnFailClauseNode) FailKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n OnFailClauseNode) TypedBindingPattern() *TypedBindingPatternNode {
	val, ok := n.ChildInBucket(2).(*TypedBindingPatternNode)
	if !ok {
		return nil
	}
	return val
}

func (n OnFailClauseNode) BlockStatement() *BlockStatementNode {
	val, ok := n.ChildInBucket(3).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

type DoStatementNode struct {
	StatementNode
}

func (n DoStatementNode) Kind() common.SyntaxKind {
	return common.DO_STATEMENT
}

func (n DoStatementNode) DoKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n DoStatementNode) BlockStatement() *BlockStatementNode {
	val, ok := n.ChildInBucket(1).(*BlockStatementNode)
	if !ok {
		return nil
	}
	return val
}

func (n DoStatementNode) OnFailClause() *OnFailClauseNode {
	val, ok := n.ChildInBucket(2).(*OnFailClauseNode)
	if !ok {
		return nil
	}
	return val
}

type ClassDefinitionNode struct {
	ModuleMemberDeclarationNode
}

func (n ClassDefinitionNode) Kind() common.SyntaxKind {
	return common.CLASS_DEFINITION
}

func (n ClassDefinitionNode) Metadata() *MetadataNode {
	val, ok := n.ChildInBucket(0).(*MetadataNode)
	if !ok {
		return nil
	}
	return val
}

func (n ClassDefinitionNode) VisibilityQualifier() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClassDefinitionNode) ClassTypeQualifiers() NodeList[Token] {
	return nodeListFrom[Token](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n ClassDefinitionNode) ClassKeyword() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClassDefinitionNode) ClassName() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClassDefinitionNode) OpenBrace() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClassDefinitionNode) Members() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(6)))
}

func (n ClassDefinitionNode) CloseBrace() Token {
	val, ok := n.ChildInBucket(7).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClassDefinitionNode) SemicolonToken() Token {
	val, ok := n.ChildInBucket(8).(Token)
	if !ok {
		return nil
	}
	return val
}

type ResourcePathParameterNode struct {
	NonTerminalNodeBase
}

func (n ResourcePathParameterNode) OpenBracketToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ResourcePathParameterNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ResourcePathParameterNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(2).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ResourcePathParameterNode) EllipsisToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ResourcePathParameterNode) ParamName() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ResourcePathParameterNode) CloseBracketToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}

type RequiredExpressionNode struct {
	ExpressionNode
}

func (n RequiredExpressionNode) Kind() common.SyntaxKind {
	return common.REQUIRED_EXPRESSION
}

func (n RequiredExpressionNode) QuestionMarkToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

type ErrorConstructorExpressionNode struct {
	ExpressionNode
}

func (n ErrorConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.ERROR_CONSTRUCTOR
}

func (n ErrorConstructorExpressionNode) ErrorKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorConstructorExpressionNode) TypeReference() TypeDescriptorNode {
	val, ok := n.ChildInBucket(1).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorConstructorExpressionNode) OpenParenToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ErrorConstructorExpressionNode) Arguments() NodeList[FunctionArgumentNode] {
	return nodeListFrom[FunctionArgumentNode](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n ErrorConstructorExpressionNode) CloseParenToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type ParameterizedTypeDescriptorNode struct {
	TypeDescriptorNode
}

func (n ParameterizedTypeDescriptorNode) KeywordToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ParameterizedTypeDescriptorNode) TypeParamNode() *TypeParameterNode {
	val, ok := n.ChildInBucket(1).(*TypeParameterNode)
	if !ok {
		return nil
	}
	return val
}

type SpreadMemberNode struct {
	NonTerminalNodeBase
}

func (n SpreadMemberNode) Kind() common.SyntaxKind {
	return common.SPREAD_MEMBER
}

func (n SpreadMemberNode) Ellipsis() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n SpreadMemberNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

type ClientResourceAccessActionNode struct {
	ActionNode
}

func (n ClientResourceAccessActionNode) Kind() common.SyntaxKind {
	return common.CLIENT_RESOURCE_ACCESS_ACTION
}

func (n ClientResourceAccessActionNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(0).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ClientResourceAccessActionNode) RightArrowToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClientResourceAccessActionNode) SlashToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClientResourceAccessActionNode) ResourceAccessPath() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n ClientResourceAccessActionNode) DotToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ClientResourceAccessActionNode) MethodName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(5).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n ClientResourceAccessActionNode) Arguments() *ParenthesizedArgList {
	val, ok := n.ChildInBucket(6).(*ParenthesizedArgList)
	if !ok {
		return nil
	}
	return val
}

type ComputedResourceAccessSegmentNode struct {
	NonTerminalNodeBase
}

func (n ComputedResourceAccessSegmentNode) Kind() common.SyntaxKind {
	return common.COMPUTED_RESOURCE_ACCESS_SEGMENT
}

func (n ComputedResourceAccessSegmentNode) OpenBracketToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ComputedResourceAccessSegmentNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(1).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ComputedResourceAccessSegmentNode) CloseBracketToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ResourceAccessRestSegmentNode struct {
	NonTerminalNodeBase
}

func (n ResourceAccessRestSegmentNode) Kind() common.SyntaxKind {
	return common.RESOURCE_ACCESS_REST_SEGMENT
}

func (n ResourceAccessRestSegmentNode) OpenBracketToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ResourceAccessRestSegmentNode) EllipsisToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ResourceAccessRestSegmentNode) Expression() ExpressionNode {
	val, ok := n.ChildInBucket(2).(ExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ResourceAccessRestSegmentNode) CloseBracketToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReSequenceNode struct {
	NonTerminalNodeBase
}

func (n ReSequenceNode) Kind() common.SyntaxKind {
	return common.RE_SEQUENCE
}

func (n ReSequenceNode) ReTerm() NodeList[ReTermNode] {
	return nodeListFrom[ReTermNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

type ReTermNode = NonTerminalNode

type ReAtomQuantifierNode struct {
	ReTermNode
}

func (n ReAtomQuantifierNode) Kind() common.SyntaxKind {
	return common.RE_ATOM_QUANTIFIER
}

func (n ReAtomQuantifierNode) ReAtom() Node {
	return n.ChildInBucket(0)
}

func (n ReAtomQuantifierNode) ReQuantifier() *ReQuantifierNode {
	val, ok := n.ChildInBucket(1).(*ReQuantifierNode)
	if !ok {
		return nil
	}
	return val
}

type ReAtomCharOrEscapeNode struct {
	NonTerminalNodeBase
}

func (n ReAtomCharOrEscapeNode) Kind() common.SyntaxKind {
	return common.RE_LITERAL_CHAR_DOT_OR_ESCAPE
}

func (n ReAtomCharOrEscapeNode) ReAtomCharOrEscape() Node {
	return n.ChildInBucket(0)
}

type ReQuoteEscapeNode struct {
	NonTerminalNodeBase
}

func (n ReQuoteEscapeNode) Kind() common.SyntaxKind {
	return common.RE_QUOTE_ESCAPE
}

func (n ReQuoteEscapeNode) SlashToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReQuoteEscapeNode) ReSyntaxChar() Node {
	return n.ChildInBucket(1)
}

type ReSimpleCharClassEscapeNode struct {
	NonTerminalNodeBase
}

func (n ReSimpleCharClassEscapeNode) Kind() common.SyntaxKind {
	return common.RE_SIMPLE_CHAR_CLASS_ESCAPE
}

func (n ReSimpleCharClassEscapeNode) SlashToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReSimpleCharClassEscapeNode) ReSimpleCharClassCode() Node {
	return n.ChildInBucket(1)
}

type ReUnicodePropertyEscapeNode struct {
	NonTerminalNodeBase
}

func (n ReUnicodePropertyEscapeNode) Kind() common.SyntaxKind {
	return common.RE_UNICODE_PROPERTY_ESCAPE
}

func (n ReUnicodePropertyEscapeNode) SlashToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReUnicodePropertyEscapeNode) Property() Node {
	return n.ChildInBucket(1)
}

func (n ReUnicodePropertyEscapeNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReUnicodePropertyEscapeNode) ReUnicodeProperty() ReUnicodePropertyNode {
	val, ok := n.ChildInBucket(3).(ReUnicodePropertyNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReUnicodePropertyEscapeNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReUnicodePropertyNode = NonTerminalNode

type ReUnicodeScriptNode struct {
	ReUnicodePropertyNode
}

func (n ReUnicodeScriptNode) Kind() common.SyntaxKind {
	return common.RE_UNICODE_SCRIPT
}

func (n ReUnicodeScriptNode) ScriptStart() Node {
	return n.ChildInBucket(0)
}

func (n ReUnicodeScriptNode) ReUnicodePropertyValue() Node {
	return n.ChildInBucket(1)
}

type ReUnicodeGeneralCategoryNode struct {
	ReUnicodePropertyNode
}

func (n ReUnicodeGeneralCategoryNode) Kind() common.SyntaxKind {
	return common.RE_UNICODE_GENERAL_CATEGORY
}

func (n ReUnicodeGeneralCategoryNode) CategoryStart() Node {
	return n.ChildInBucket(0)
}

func (n ReUnicodeGeneralCategoryNode) ReUnicodeGeneralCategoryName() Node {
	return n.ChildInBucket(1)
}

type ReCharacterClassNode struct {
	NonTerminalNodeBase
}

func (n ReCharacterClassNode) Kind() common.SyntaxKind {
	return common.RE_CHARACTER_CLASS
}

func (n ReCharacterClassNode) OpenBracket() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReCharacterClassNode) Negation() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReCharacterClassNode) ReCharSet() Node {
	return n.ChildInBucket(2)
}

func (n ReCharacterClassNode) CloseBracket() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReCharSetRangeWithReCharSetNode struct {
	NonTerminalNodeBase
}

func (n ReCharSetRangeWithReCharSetNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE_WITH_RE_CHAR_SET
}

func (n ReCharSetRangeWithReCharSetNode) ReCharSetRange() *ReCharSetRangeNode {
	val, ok := n.ChildInBucket(0).(*ReCharSetRangeNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReCharSetRangeWithReCharSetNode) ReCharSet() Node {
	return n.ChildInBucket(1)
}

type ReCharSetRangeNode struct {
	NonTerminalNodeBase
}

func (n ReCharSetRangeNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE
}

func (n ReCharSetRangeNode) LhsReCharSetAtom() Node {
	return n.ChildInBucket(0)
}

func (n ReCharSetRangeNode) MinusToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReCharSetRangeNode) RhsReCharSetAtom() Node {
	return n.ChildInBucket(2)
}

type ReCharSetAtomWithReCharSetNoDashNode struct {
	NonTerminalNodeBase
}

func (n ReCharSetAtomWithReCharSetNoDashNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_ATOM_WITH_RE_CHAR_SET_NO_DASH
}

func (n ReCharSetAtomWithReCharSetNoDashNode) ReCharSetAtom() Node {
	return n.ChildInBucket(0)
}

func (n ReCharSetAtomWithReCharSetNoDashNode) ReCharSetNoDash() Node {
	return n.ChildInBucket(1)
}

type ReCharSetRangeNoDashWithReCharSetNode struct {
	NonTerminalNodeBase
}

func (n ReCharSetRangeNoDashWithReCharSetNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE_NO_DASH_WITH_RE_CHAR_SET
}

func (n ReCharSetRangeNoDashWithReCharSetNode) ReCharSetRangeNoDash() *ReCharSetRangeNoDashNode {
	val, ok := n.ChildInBucket(0).(*ReCharSetRangeNoDashNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReCharSetRangeNoDashWithReCharSetNode) ReCharSet() Node {
	return n.ChildInBucket(1)
}

type ReCharSetRangeNoDashNode struct {
	NonTerminalNodeBase
}

func (n ReCharSetRangeNoDashNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE_NO_DASH
}

func (n ReCharSetRangeNoDashNode) ReCharSetAtomNoDash() Node {
	return n.ChildInBucket(0)
}

func (n ReCharSetRangeNoDashNode) MinusToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReCharSetRangeNoDashNode) ReCharSetAtom() Node {
	return n.ChildInBucket(2)
}

type ReCharSetAtomNoDashWithReCharSetNoDashNode struct {
	NonTerminalNodeBase
}

func (n ReCharSetAtomNoDashWithReCharSetNoDashNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_ATOM_NO_DASH_WITH_RE_CHAR_SET_NO_DASH
}

func (n ReCharSetAtomNoDashWithReCharSetNoDashNode) ReCharSetAtomNoDash() Node {
	return n.ChildInBucket(0)
}

func (n ReCharSetAtomNoDashWithReCharSetNoDashNode) ReCharSetNoDash() Node {
	return n.ChildInBucket(1)
}

type ReCapturingGroupsNode struct {
	NonTerminalNodeBase
}

func (n ReCapturingGroupsNode) Kind() common.SyntaxKind {
	return common.RE_CAPTURING_GROUP
}

func (n ReCapturingGroupsNode) OpenParenthesis() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReCapturingGroupsNode) ReFlagExpression() *ReFlagExpressionNode {
	val, ok := n.ChildInBucket(1).(*ReFlagExpressionNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReCapturingGroupsNode) ReSequences() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(2)))
}

func (n ReCapturingGroupsNode) CloseParenthesis() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReFlagExpressionNode struct {
	NonTerminalNodeBase
}

func (n ReFlagExpressionNode) Kind() common.SyntaxKind {
	return common.RE_FLAG_EXPR
}

func (n ReFlagExpressionNode) QuestionMark() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReFlagExpressionNode) ReFlagsOnOff() *ReFlagsOnOffNode {
	val, ok := n.ChildInBucket(1).(*ReFlagsOnOffNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReFlagExpressionNode) Colon() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReFlagsOnOffNode struct {
	NonTerminalNodeBase
}

func (n ReFlagsOnOffNode) Kind() common.SyntaxKind {
	return common.RE_FLAGS_ON_OFF
}

func (n ReFlagsOnOffNode) LhsReFlags() *ReFlagsNode {
	val, ok := n.ChildInBucket(0).(*ReFlagsNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReFlagsOnOffNode) MinusToken() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReFlagsOnOffNode) RhsReFlags() *ReFlagsNode {
	val, ok := n.ChildInBucket(2).(*ReFlagsNode)
	if !ok {
		return nil
	}
	return val
}

type ReFlagsNode struct {
	NonTerminalNodeBase
}

func (n ReFlagsNode) Kind() common.SyntaxKind {
	return common.RE_FLAGS
}

func (n ReFlagsNode) ReFlag() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(0)))
}

type ReAssertionNode struct {
	ReTermNode
}

func (n ReAssertionNode) Kind() common.SyntaxKind {
	return common.RE_ASSERTION
}

func (n ReAssertionNode) ReAssertion() Node {
	return n.ChildInBucket(0)
}

type ReQuantifierNode struct {
	NonTerminalNodeBase
}

func (n ReQuantifierNode) Kind() common.SyntaxKind {
	return common.RE_QUANTIFIER
}

func (n ReQuantifierNode) ReBaseQuantifier() Node {
	return n.ChildInBucket(0)
}

func (n ReQuantifierNode) NonGreedyChar() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

type ReBracedQuantifierNode struct {
	NonTerminalNodeBase
}

func (n ReBracedQuantifierNode) Kind() common.SyntaxKind {
	return common.RE_BRACED_QUANTIFIER
}

func (n ReBracedQuantifierNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReBracedQuantifierNode) LeastTimesMatchedDigit() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(1)))
}

func (n ReBracedQuantifierNode) CommaToken() Token {
	val, ok := n.ChildInBucket(2).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReBracedQuantifierNode) MostTimesMatchedDigit() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(3)))
}

func (n ReBracedQuantifierNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(4).(Token)
	if !ok {
		return nil
	}
	return val
}

type MemberTypeDescriptorNode struct {
	NonTerminalNodeBase
}

func (n MemberTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.MEMBER_TYPE_DESC
}

func (n MemberTypeDescriptorNode) Annotations() NodeList[*AnnotationNode] {
	return nodeListFrom[*AnnotationNode](into[NonTerminalNode](n.ChildInBucket(0)))
}

func (n MemberTypeDescriptorNode) TypeDescriptor() TypeDescriptorNode {
	val, ok := n.ChildInBucket(1).(TypeDescriptorNode)
	if !ok {
		return nil
	}
	return val
}

type ReceiveFieldNode struct {
	NonTerminalNodeBase
}

func (n ReceiveFieldNode) Kind() common.SyntaxKind {
	return common.RECEIVE_FIELD
}

func (n ReceiveFieldNode) FieldName() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(0).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

func (n ReceiveFieldNode) Colon() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n ReceiveFieldNode) PeerWorker() *SimpleNameReferenceNode {
	val, ok := n.ChildInBucket(2).(*SimpleNameReferenceNode)
	if !ok {
		return nil
	}
	return val
}

type NaturalExpressionNode struct {
	ExpressionNode
}

func (n NaturalExpressionNode) Kind() common.SyntaxKind {
	return common.NATURAL_EXPRESSION
}

func (n NaturalExpressionNode) ConstKeyword() Token {
	val, ok := n.ChildInBucket(0).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NaturalExpressionNode) NaturalKeyword() Token {
	val, ok := n.ChildInBucket(1).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NaturalExpressionNode) ParenthesizedArgList() *ParenthesizedArgList {
	val, ok := n.ChildInBucket(2).(*ParenthesizedArgList)
	if !ok {
		return nil
	}
	return val
}

func (n NaturalExpressionNode) OpenBraceToken() Token {
	val, ok := n.ChildInBucket(3).(Token)
	if !ok {
		return nil
	}
	return val
}

func (n NaturalExpressionNode) Prompt() NodeList[Node] {
	return nodeListFrom[Node](into[NonTerminalNode](n.ChildInBucket(4)))
}

func (n NaturalExpressionNode) CloseBraceToken() Token {
	val, ok := n.ChildInBucket(5).(Token)
	if !ok {
		return nil
	}
	return val
}
