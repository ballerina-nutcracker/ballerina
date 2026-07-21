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

package tree

import "ballerina/parser/common"

type STModulePart struct {
	STNode

	Imports STNode

	Members STNode

	EofToken STNode
}

var _ STNode = &STModulePart{}

func (n *STModulePart) Kind() common.SyntaxKind {
	return common.MODULE_PART
}

func (n *STModulePart) BucketCount() int {
	return 3
}

func (n *STModulePart) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Imports

	case 1:
		return n.Members

	case 2:
		return n.EofToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STModulePart) ChildBuckets() []STNode {
	return []STNode{

		n.Imports,

		n.Members,

		n.EofToken,
	}
}

func (n *STModulePart) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ModulePart{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STModuleMemberDeclarationNode = STNode

type STFunctionDefinition struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	QualifierList STNode

	FunctionKeyword STNode

	FunctionName STNode

	RelativeResourcePath STNode

	FunctionSignature STNode

	FunctionBody STNode
}

var _ STNode = &STFunctionDefinition{}

func (n *STFunctionDefinition) Kind() common.SyntaxKind {
	return n.STModuleMemberDeclarationNode.Kind()
}

func (n *STFunctionDefinition) BucketCount() int {
	return 7
}

func (n *STFunctionDefinition) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.QualifierList

	case 2:
		return n.FunctionKeyword

	case 3:
		return n.FunctionName

	case 4:
		return n.RelativeResourcePath

	case 5:
		return n.FunctionSignature

	case 6:
		return n.FunctionBody

	default:
		panic("invalid bucket index")
	}
}

func (n *STFunctionDefinition) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.QualifierList,

		n.FunctionKeyword,

		n.FunctionName,

		n.RelativeResourcePath,

		n.FunctionSignature,

		n.FunctionBody,
	}
}

func (n *STFunctionDefinition) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FunctionDefinition{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STImportDeclarationNode struct {
	STNode

	ImportKeyword STNode

	OrgName STNode

	ModuleName STNode

	Prefix STNode

	Semicolon STNode
}

var _ STNode = &STImportDeclarationNode{}

func (n *STImportDeclarationNode) Kind() common.SyntaxKind {
	return common.IMPORT_DECLARATION
}

func (n *STImportDeclarationNode) BucketCount() int {
	return 5
}

func (n *STImportDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ImportKeyword

	case 1:
		return n.OrgName

	case 2:
		return n.ModuleName

	case 3:
		return n.Prefix

	case 4:
		return n.Semicolon

	default:
		panic("invalid bucket index")
	}
}

func (n *STImportDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.ImportKeyword,

		n.OrgName,

		n.ModuleName,

		n.Prefix,

		n.Semicolon,
	}
}

func (n *STImportDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ImportDeclarationNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STListenerDeclarationNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	VisibilityQualifier STNode

	ListenerKeyword STNode

	TypeDescriptor STNode

	VariableName STNode

	EqualsToken STNode

	Initializer STNode

	SemicolonToken STNode
}

var _ STNode = &STListenerDeclarationNode{}

func (n *STListenerDeclarationNode) Kind() common.SyntaxKind {
	return common.LISTENER_DECLARATION
}

func (n *STListenerDeclarationNode) BucketCount() int {
	return 8
}

func (n *STListenerDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.VisibilityQualifier

	case 2:
		return n.ListenerKeyword

	case 3:
		return n.TypeDescriptor

	case 4:
		return n.VariableName

	case 5:
		return n.EqualsToken

	case 6:
		return n.Initializer

	case 7:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STListenerDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.VisibilityQualifier,

		n.ListenerKeyword,

		n.TypeDescriptor,

		n.VariableName,

		n.EqualsToken,

		n.Initializer,

		n.SemicolonToken,
	}
}

func (n *STListenerDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ListenerDeclarationNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeDefinitionNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	VisibilityQualifier STNode

	TypeKeyword STNode

	TypeName STNode

	TypeDescriptor STNode

	SemicolonToken STNode
}

var _ STNode = &STTypeDefinitionNode{}

func (n *STTypeDefinitionNode) Kind() common.SyntaxKind {
	return common.TYPE_DEFINITION
}

func (n *STTypeDefinitionNode) BucketCount() int {
	return 6
}

func (n *STTypeDefinitionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.VisibilityQualifier

	case 2:
		return n.TypeKeyword

	case 3:
		return n.TypeName

	case 4:
		return n.TypeDescriptor

	case 5:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeDefinitionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.VisibilityQualifier,

		n.TypeKeyword,

		n.TypeName,

		n.TypeDescriptor,

		n.SemicolonToken,
	}
}

func (n *STTypeDefinitionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeDefinitionNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STServiceDeclarationNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	Qualifiers STNode

	ServiceKeyword STNode

	TypeDescriptor STNode

	AbsoluteResourcePath STNode

	OnKeyword STNode

	Expressions STNode

	OpenBraceToken STNode

	Members STNode

	CloseBraceToken STNode

	SemicolonToken STNode
}

var _ STNode = &STServiceDeclarationNode{}

func (n *STServiceDeclarationNode) Kind() common.SyntaxKind {
	return common.SERVICE_DECLARATION
}

func (n *STServiceDeclarationNode) BucketCount() int {
	return 11
}

func (n *STServiceDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.Qualifiers

	case 2:
		return n.ServiceKeyword

	case 3:
		return n.TypeDescriptor

	case 4:
		return n.AbsoluteResourcePath

	case 5:
		return n.OnKeyword

	case 6:
		return n.Expressions

	case 7:
		return n.OpenBraceToken

	case 8:
		return n.Members

	case 9:
		return n.CloseBraceToken

	case 10:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STServiceDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.Qualifiers,

		n.ServiceKeyword,

		n.TypeDescriptor,

		n.AbsoluteResourcePath,

		n.OnKeyword,

		n.Expressions,

		n.OpenBraceToken,

		n.Members,

		n.CloseBraceToken,

		n.SemicolonToken,
	}
}

func (n *STServiceDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ServiceDeclarationNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STStatementNode = STNode

type STAssignmentStatementNode struct {
	STStatementNode

	VarRef STNode

	EqualsToken STNode

	Expression STNode

	SemicolonToken STNode
}

var _ STNode = &STAssignmentStatementNode{}

func (n *STAssignmentStatementNode) Kind() common.SyntaxKind {
	return common.ASSIGNMENT_STATEMENT
}

func (n *STAssignmentStatementNode) BucketCount() int {
	return 4
}

func (n *STAssignmentStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.VarRef

	case 1:
		return n.EqualsToken

	case 2:
		return n.Expression

	case 3:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STAssignmentStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.VarRef,

		n.EqualsToken,

		n.Expression,

		n.SemicolonToken,
	}
}

func (n *STAssignmentStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &AssignmentStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STCompoundAssignmentStatementNode struct {
	STStatementNode

	LhsExpression STNode

	BinaryOperator STNode

	EqualsToken STNode

	RhsExpression STNode

	SemicolonToken STNode
}

var _ STNode = &STCompoundAssignmentStatementNode{}

func (n *STCompoundAssignmentStatementNode) Kind() common.SyntaxKind {
	return common.COMPOUND_ASSIGNMENT_STATEMENT
}

func (n *STCompoundAssignmentStatementNode) BucketCount() int {
	return 5
}

func (n *STCompoundAssignmentStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LhsExpression

	case 1:
		return n.BinaryOperator

	case 2:
		return n.EqualsToken

	case 3:
		return n.RhsExpression

	case 4:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STCompoundAssignmentStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.LhsExpression,

		n.BinaryOperator,

		n.EqualsToken,

		n.RhsExpression,

		n.SemicolonToken,
	}
}

func (n *STCompoundAssignmentStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &CompoundAssignmentStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STVariableDeclarationNode struct {
	STStatementNode

	Annotations STNode

	FinalKeyword STNode

	TypedBindingPattern STNode

	EqualsToken STNode

	Initializer STNode

	SemicolonToken STNode
}

var _ STNode = &STVariableDeclarationNode{}

func (n *STVariableDeclarationNode) Kind() common.SyntaxKind {
	return common.LOCAL_VAR_DECL
}

func (n *STVariableDeclarationNode) BucketCount() int {
	return 6
}

func (n *STVariableDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.FinalKeyword

	case 2:
		return n.TypedBindingPattern

	case 3:
		return n.EqualsToken

	case 4:
		return n.Initializer

	case 5:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STVariableDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.FinalKeyword,

		n.TypedBindingPattern,

		n.EqualsToken,

		n.Initializer,

		n.SemicolonToken,
	}
}

func (n *STVariableDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &VariableDeclarationNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STBlockStatementNode struct {
	STStatementNode

	OpenBraceToken STNode

	Statements STNode

	CloseBraceToken STNode
}

var _ STNode = &STBlockStatementNode{}

func (n *STBlockStatementNode) Kind() common.SyntaxKind {
	return common.BLOCK_STATEMENT
}

func (n *STBlockStatementNode) BucketCount() int {
	return 3
}

func (n *STBlockStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBraceToken

	case 1:
		return n.Statements

	case 2:
		return n.CloseBraceToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STBlockStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBraceToken,

		n.Statements,

		n.CloseBraceToken,
	}
}

func (n *STBlockStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &BlockStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STBreakStatementNode struct {
	STStatementNode

	BreakToken STNode

	SemicolonToken STNode
}

var _ STNode = &STBreakStatementNode{}

func (n *STBreakStatementNode) Kind() common.SyntaxKind {
	return common.BREAK_STATEMENT
}

func (n *STBreakStatementNode) BucketCount() int {
	return 2
}

func (n *STBreakStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.BreakToken

	case 1:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STBreakStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.BreakToken,

		n.SemicolonToken,
	}
}

func (n *STBreakStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &BreakStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFailStatementNode struct {
	STStatementNode

	FailKeyword STNode

	Expression STNode

	SemicolonToken STNode
}

var _ STNode = &STFailStatementNode{}

func (n *STFailStatementNode) Kind() common.SyntaxKind {
	return common.FAIL_STATEMENT
}

func (n *STFailStatementNode) BucketCount() int {
	return 3
}

func (n *STFailStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FailKeyword

	case 1:
		return n.Expression

	case 2:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STFailStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.FailKeyword,

		n.Expression,

		n.SemicolonToken,
	}
}

func (n *STFailStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FailStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STExpressionStatementNode struct {
	STStatementNode

	Expression STNode

	SemicolonToken STNode
}

var _ STNode = &STExpressionStatementNode{}

func (n *STExpressionStatementNode) Kind() common.SyntaxKind {
	return n.STStatementNode.Kind()
}

func (n *STExpressionStatementNode) BucketCount() int {
	return 2
}

func (n *STExpressionStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STExpressionStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.SemicolonToken,
	}
}

func (n *STExpressionStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ExpressionStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STContinueStatementNode struct {
	STStatementNode

	ContinueToken STNode

	SemicolonToken STNode
}

var _ STNode = &STContinueStatementNode{}

func (n *STContinueStatementNode) Kind() common.SyntaxKind {
	return common.CONTINUE_STATEMENT
}

func (n *STContinueStatementNode) BucketCount() int {
	return 2
}

func (n *STContinueStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ContinueToken

	case 1:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STContinueStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.ContinueToken,

		n.SemicolonToken,
	}
}

func (n *STContinueStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ContinueStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STExternalFunctionBodyNode struct {
	STFunctionBodyNode

	EqualsToken STNode

	Annotations STNode

	ExternalKeyword STNode

	SemicolonToken STNode
}

var _ STNode = &STExternalFunctionBodyNode{}

func (n *STExternalFunctionBodyNode) Kind() common.SyntaxKind {
	return common.EXTERNAL_FUNCTION_BODY
}

func (n *STExternalFunctionBodyNode) BucketCount() int {
	return 4
}

func (n *STExternalFunctionBodyNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.EqualsToken

	case 1:
		return n.Annotations

	case 2:
		return n.ExternalKeyword

	case 3:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STExternalFunctionBodyNode) ChildBuckets() []STNode {
	return []STNode{

		n.EqualsToken,

		n.Annotations,

		n.ExternalKeyword,

		n.SemicolonToken,
	}
}

func (n *STExternalFunctionBodyNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ExternalFunctionBodyNode{
		FunctionBodyNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STIfElseStatementNode struct {
	STStatementNode

	IfKeyword STNode

	Condition STNode

	IfBody STNode

	ElseBody STNode
}

var _ STNode = &STIfElseStatementNode{}

func (n *STIfElseStatementNode) Kind() common.SyntaxKind {
	return common.IF_ELSE_STATEMENT
}

func (n *STIfElseStatementNode) BucketCount() int {
	return 4
}

func (n *STIfElseStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.IfKeyword

	case 1:
		return n.Condition

	case 2:
		return n.IfBody

	case 3:
		return n.ElseBody

	default:
		panic("invalid bucket index")
	}
}

func (n *STIfElseStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.IfKeyword,

		n.Condition,

		n.IfBody,

		n.ElseBody,
	}
}

func (n *STIfElseStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &IfElseStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STElseBlockNode struct {
	STNode

	ElseKeyword STNode

	ElseBody STNode
}

var _ STNode = &STElseBlockNode{}

func (n *STElseBlockNode) Kind() common.SyntaxKind {
	return common.ELSE_BLOCK
}

func (n *STElseBlockNode) BucketCount() int {
	return 2
}

func (n *STElseBlockNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ElseKeyword

	case 1:
		return n.ElseBody

	default:
		panic("invalid bucket index")
	}
}

func (n *STElseBlockNode) ChildBuckets() []STNode {
	return []STNode{

		n.ElseKeyword,

		n.ElseBody,
	}
}

func (n *STElseBlockNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ElseBlockNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STWhileStatementNode struct {
	STStatementNode

	WhileKeyword STNode

	Condition STNode

	WhileBody STNode

	OnFailClause STNode
}

var _ STNode = &STWhileStatementNode{}

func (n *STWhileStatementNode) Kind() common.SyntaxKind {
	return common.WHILE_STATEMENT
}

func (n *STWhileStatementNode) BucketCount() int {
	return 4
}

func (n *STWhileStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.WhileKeyword

	case 1:
		return n.Condition

	case 2:
		return n.WhileBody

	case 3:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STWhileStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.WhileKeyword,

		n.Condition,

		n.WhileBody,

		n.OnFailClause,
	}
}

func (n *STWhileStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &WhileStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STPanicStatementNode struct {
	STStatementNode

	PanicKeyword STNode

	Expression STNode

	SemicolonToken STNode
}

var _ STNode = &STPanicStatementNode{}

func (n *STPanicStatementNode) Kind() common.SyntaxKind {
	return common.PANIC_STATEMENT
}

func (n *STPanicStatementNode) BucketCount() int {
	return 3
}

func (n *STPanicStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.PanicKeyword

	case 1:
		return n.Expression

	case 2:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STPanicStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.PanicKeyword,

		n.Expression,

		n.SemicolonToken,
	}
}

func (n *STPanicStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &PanicStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReturnStatementNode struct {
	STStatementNode

	ReturnKeyword STNode

	Expression STNode

	SemicolonToken STNode
}

var _ STNode = &STReturnStatementNode{}

func (n *STReturnStatementNode) Kind() common.SyntaxKind {
	return common.RETURN_STATEMENT
}

func (n *STReturnStatementNode) BucketCount() int {
	return 3
}

func (n *STReturnStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReturnKeyword

	case 1:
		return n.Expression

	case 2:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STReturnStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReturnKeyword,

		n.Expression,

		n.SemicolonToken,
	}
}

func (n *STReturnStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReturnStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STLocalTypeDefinitionStatementNode struct {
	STStatementNode

	Annotations STNode

	TypeKeyword STNode

	TypeName STNode

	TypeDescriptor STNode

	SemicolonToken STNode
}

var _ STNode = &STLocalTypeDefinitionStatementNode{}

func (n *STLocalTypeDefinitionStatementNode) Kind() common.SyntaxKind {
	return common.LOCAL_TYPE_DEFINITION_STATEMENT
}

func (n *STLocalTypeDefinitionStatementNode) BucketCount() int {
	return 5
}

func (n *STLocalTypeDefinitionStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.TypeKeyword

	case 2:
		return n.TypeName

	case 3:
		return n.TypeDescriptor

	case 4:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STLocalTypeDefinitionStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.TypeKeyword,

		n.TypeName,

		n.TypeDescriptor,

		n.SemicolonToken,
	}
}

func (n *STLocalTypeDefinitionStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &LocalTypeDefinitionStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STLockStatementNode struct {
	STStatementNode

	LockKeyword STNode

	BlockStatement STNode

	OnFailClause STNode
}

var _ STNode = &STLockStatementNode{}

func (n *STLockStatementNode) Kind() common.SyntaxKind {
	return common.LOCK_STATEMENT
}

func (n *STLockStatementNode) BucketCount() int {
	return 3
}

func (n *STLockStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LockKeyword

	case 1:
		return n.BlockStatement

	case 2:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STLockStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.LockKeyword,

		n.BlockStatement,

		n.OnFailClause,
	}
}

func (n *STLockStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &LockStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STForkStatementNode struct {
	STStatementNode

	ForkKeyword STNode

	OpenBraceToken STNode

	NamedWorkerDeclarations STNode

	CloseBraceToken STNode
}

var _ STNode = &STForkStatementNode{}

func (n *STForkStatementNode) Kind() common.SyntaxKind {
	return common.FORK_STATEMENT
}

func (n *STForkStatementNode) BucketCount() int {
	return 4
}

func (n *STForkStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ForkKeyword

	case 1:
		return n.OpenBraceToken

	case 2:
		return n.NamedWorkerDeclarations

	case 3:
		return n.CloseBraceToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STForkStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.ForkKeyword,

		n.OpenBraceToken,

		n.NamedWorkerDeclarations,

		n.CloseBraceToken,
	}
}

func (n *STForkStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ForkStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STForEachStatementNode struct {
	STStatementNode

	ForEachKeyword STNode

	TypedBindingPattern STNode

	InKeyword STNode

	ActionOrExpressionNode STNode

	BlockStatement STNode

	OnFailClause STNode
}

var _ STNode = &STForEachStatementNode{}

func (n *STForEachStatementNode) Kind() common.SyntaxKind {
	return common.FOREACH_STATEMENT
}

func (n *STForEachStatementNode) BucketCount() int {
	return 6
}

func (n *STForEachStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ForEachKeyword

	case 1:
		return n.TypedBindingPattern

	case 2:
		return n.InKeyword

	case 3:
		return n.ActionOrExpressionNode

	case 4:
		return n.BlockStatement

	case 5:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STForEachStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.ForEachKeyword,

		n.TypedBindingPattern,

		n.InKeyword,

		n.ActionOrExpressionNode,

		n.BlockStatement,

		n.OnFailClause,
	}
}

func (n *STForEachStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ForEachStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STExpressionNode = STNode

type STBinaryExpressionNode struct {
	STExpressionNode

	LhsExpr STNode

	Operator STNode

	RhsExpr STNode
}

var _ STNode = &STBinaryExpressionNode{}

func (n *STBinaryExpressionNode) Kind() common.SyntaxKind {
	return n.STExpressionNode.Kind()
}

func (n *STBinaryExpressionNode) BucketCount() int {
	return 3
}

func (n *STBinaryExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LhsExpr

	case 1:
		return n.Operator

	case 2:
		return n.RhsExpr

	default:
		panic("invalid bucket index")
	}
}

func (n *STBinaryExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.LhsExpr,

		n.Operator,

		n.RhsExpr,
	}
}

func (n *STBinaryExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &BinaryExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STBracedExpressionNode struct {
	STExpressionNode

	OpenParen STNode

	Expression STNode

	CloseParen STNode
}

var _ STNode = &STBracedExpressionNode{}

func (n *STBracedExpressionNode) Kind() common.SyntaxKind {
	return n.STExpressionNode.Kind()
}

func (n *STBracedExpressionNode) BucketCount() int {
	return 3
}

func (n *STBracedExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParen

	case 1:
		return n.Expression

	case 2:
		return n.CloseParen

	default:
		panic("invalid bucket index")
	}
}

func (n *STBracedExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParen,

		n.Expression,

		n.CloseParen,
	}
}

func (n *STBracedExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &BracedExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STCheckExpressionNode struct {
	STExpressionNode

	CheckKeyword STNode

	Expression STNode
}

var _ STNode = &STCheckExpressionNode{}

func (n *STCheckExpressionNode) Kind() common.SyntaxKind {
	return n.STExpressionNode.Kind()
}

func (n *STCheckExpressionNode) BucketCount() int {
	return 2
}

func (n *STCheckExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.CheckKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STCheckExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.CheckKeyword,

		n.Expression,
	}
}

func (n *STCheckExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &CheckExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFieldAccessExpressionNode struct {
	STExpressionNode

	Expression STNode

	DotToken STNode

	FieldName STNode
}

var _ STNode = &STFieldAccessExpressionNode{}

func (n *STFieldAccessExpressionNode) Kind() common.SyntaxKind {
	return common.FIELD_ACCESS
}

func (n *STFieldAccessExpressionNode) BucketCount() int {
	return 3
}

func (n *STFieldAccessExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.DotToken

	case 2:
		return n.FieldName

	default:
		panic("invalid bucket index")
	}
}

func (n *STFieldAccessExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.DotToken,

		n.FieldName,
	}
}

func (n *STFieldAccessExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FieldAccessExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFunctionCallExpressionNode struct {
	STExpressionNode

	FunctionName STNode

	OpenParenToken STNode

	Arguments STNode

	CloseParenToken STNode
}

var _ STNode = &STFunctionCallExpressionNode{}

func (n *STFunctionCallExpressionNode) Kind() common.SyntaxKind {
	return common.FUNCTION_CALL
}

func (n *STFunctionCallExpressionNode) BucketCount() int {
	return 4
}

func (n *STFunctionCallExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FunctionName

	case 1:
		return n.OpenParenToken

	case 2:
		return n.Arguments

	case 3:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STFunctionCallExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.FunctionName,

		n.OpenParenToken,

		n.Arguments,

		n.CloseParenToken,
	}
}

func (n *STFunctionCallExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FunctionCallExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMethodCallExpressionNode struct {
	STExpressionNode

	Expression STNode

	DotToken STNode

	MethodName STNode

	OpenParenToken STNode

	Arguments STNode

	CloseParenToken STNode
}

var _ STNode = &STMethodCallExpressionNode{}

func (n *STMethodCallExpressionNode) Kind() common.SyntaxKind {
	return common.METHOD_CALL
}

func (n *STMethodCallExpressionNode) BucketCount() int {
	return 6
}

func (n *STMethodCallExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.DotToken

	case 2:
		return n.MethodName

	case 3:
		return n.OpenParenToken

	case 4:
		return n.Arguments

	case 5:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STMethodCallExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.DotToken,

		n.MethodName,

		n.OpenParenToken,

		n.Arguments,

		n.CloseParenToken,
	}
}

func (n *STMethodCallExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MethodCallExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMappingConstructorExpressionNode struct {
	STExpressionNode

	OpenBrace STNode

	Fields STNode

	CloseBrace STNode
}

var _ STNode = &STMappingConstructorExpressionNode{}

func (n *STMappingConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.MAPPING_CONSTRUCTOR
}

func (n *STMappingConstructorExpressionNode) BucketCount() int {
	return 3
}

func (n *STMappingConstructorExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBrace

	case 1:
		return n.Fields

	case 2:
		return n.CloseBrace

	default:
		panic("invalid bucket index")
	}
}

func (n *STMappingConstructorExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBrace,

		n.Fields,

		n.CloseBrace,
	}
}

func (n *STMappingConstructorExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MappingConstructorExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STIndexedExpressionNode struct {
	STTypeDescriptorNode

	ContainerExpression STNode

	OpenBracket STNode

	KeyExpression STNode

	CloseBracket STNode
}

var _ STNode = &STIndexedExpressionNode{}

func (n *STIndexedExpressionNode) Kind() common.SyntaxKind {
	return common.INDEXED_EXPRESSION
}

func (n *STIndexedExpressionNode) BucketCount() int {
	return 4
}

func (n *STIndexedExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ContainerExpression

	case 1:
		return n.OpenBracket

	case 2:
		return n.KeyExpression

	case 3:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STIndexedExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.ContainerExpression,

		n.OpenBracket,

		n.KeyExpression,

		n.CloseBracket,
	}
}

func (n *STIndexedExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &IndexedExpressionNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeofExpressionNode struct {
	STExpressionNode

	TypeofKeyword STNode

	Expression STNode
}

var _ STNode = &STTypeofExpressionNode{}

func (n *STTypeofExpressionNode) Kind() common.SyntaxKind {
	return common.TYPEOF_EXPRESSION
}

func (n *STTypeofExpressionNode) BucketCount() int {
	return 2
}

func (n *STTypeofExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TypeofKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeofExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.TypeofKeyword,

		n.Expression,
	}
}

func (n *STTypeofExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeofExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STUnaryExpressionNode struct {
	STExpressionNode

	UnaryOperator STNode

	Expression STNode
}

var _ STNode = &STUnaryExpressionNode{}

func (n *STUnaryExpressionNode) Kind() common.SyntaxKind {
	return common.UNARY_EXPRESSION
}

func (n *STUnaryExpressionNode) BucketCount() int {
	return 2
}

func (n *STUnaryExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.UnaryOperator

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STUnaryExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.UnaryOperator,

		n.Expression,
	}
}

func (n *STUnaryExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &UnaryExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STComputedNameFieldNode struct {
	STMappingFieldNode

	OpenBracket STNode

	FieldNameExpr STNode

	CloseBracket STNode

	ColonToken STNode

	ValueExpr STNode
}

var _ STNode = &STComputedNameFieldNode{}

func (n *STComputedNameFieldNode) Kind() common.SyntaxKind {
	return common.COMPUTED_NAME_FIELD
}

func (n *STComputedNameFieldNode) BucketCount() int {
	return 5
}

func (n *STComputedNameFieldNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracket

	case 1:
		return n.FieldNameExpr

	case 2:
		return n.CloseBracket

	case 3:
		return n.ColonToken

	case 4:
		return n.ValueExpr

	default:
		panic("invalid bucket index")
	}
}

func (n *STComputedNameFieldNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracket,

		n.FieldNameExpr,

		n.CloseBracket,

		n.ColonToken,

		n.ValueExpr,
	}
}

func (n *STComputedNameFieldNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ComputedNameFieldNode{
		MappingFieldNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STConstantDeclarationNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	VisibilityQualifier STNode

	ConstKeyword STNode

	TypeDescriptor STNode

	VariableName STNode

	EqualsToken STNode

	Initializer STNode

	SemicolonToken STNode
}

var _ STNode = &STConstantDeclarationNode{}

func (n *STConstantDeclarationNode) Kind() common.SyntaxKind {
	return common.CONST_DECLARATION
}

func (n *STConstantDeclarationNode) BucketCount() int {
	return 8
}

func (n *STConstantDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.VisibilityQualifier

	case 2:
		return n.ConstKeyword

	case 3:
		return n.TypeDescriptor

	case 4:
		return n.VariableName

	case 5:
		return n.EqualsToken

	case 6:
		return n.Initializer

	case 7:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STConstantDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.VisibilityQualifier,

		n.ConstKeyword,

		n.TypeDescriptor,

		n.VariableName,

		n.EqualsToken,

		n.Initializer,

		n.SemicolonToken,
	}
}

func (n *STConstantDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ConstantDeclarationNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STParameterNode = STNode

type STDefaultableParameterNode struct {
	STParameterNode

	Annotations STNode

	TypeName STNode

	ParamName STNode

	EqualsToken STNode

	Expression STNode
}

var _ STNode = &STDefaultableParameterNode{}

func (n *STDefaultableParameterNode) Kind() common.SyntaxKind {
	return common.DEFAULTABLE_PARAM
}

func (n *STDefaultableParameterNode) BucketCount() int {
	return 5
}

func (n *STDefaultableParameterNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.TypeName

	case 2:
		return n.ParamName

	case 3:
		return n.EqualsToken

	case 4:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STDefaultableParameterNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.TypeName,

		n.ParamName,

		n.EqualsToken,

		n.Expression,
	}
}

func (n *STDefaultableParameterNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &DefaultableParameterNode{
		ParameterNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRequiredParameterNode struct {
	STParameterNode

	Annotations STNode

	TypeName STNode

	ParamName STNode
}

var _ STNode = &STRequiredParameterNode{}

func (n *STRequiredParameterNode) Kind() common.SyntaxKind {
	return common.REQUIRED_PARAM
}

func (n *STRequiredParameterNode) BucketCount() int {
	return 3
}

func (n *STRequiredParameterNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.TypeName

	case 2:
		return n.ParamName

	default:
		panic("invalid bucket index")
	}
}

func (n *STRequiredParameterNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.TypeName,

		n.ParamName,
	}
}

func (n *STRequiredParameterNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RequiredParameterNode{
		ParameterNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STIncludedRecordParameterNode struct {
	STParameterNode

	Annotations STNode

	AsteriskToken STNode

	TypeName STNode

	ParamName STNode
}

var _ STNode = &STIncludedRecordParameterNode{}

func (n *STIncludedRecordParameterNode) Kind() common.SyntaxKind {
	return common.INCLUDED_RECORD_PARAM
}

func (n *STIncludedRecordParameterNode) BucketCount() int {
	return 4
}

func (n *STIncludedRecordParameterNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.AsteriskToken

	case 2:
		return n.TypeName

	case 3:
		return n.ParamName

	default:
		panic("invalid bucket index")
	}
}

func (n *STIncludedRecordParameterNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.AsteriskToken,

		n.TypeName,

		n.ParamName,
	}
}

func (n *STIncludedRecordParameterNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &IncludedRecordParameterNode{
		ParameterNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRestParameterNode struct {
	STParameterNode

	Annotations STNode

	TypeName STNode

	EllipsisToken STNode

	ParamName STNode
}

var _ STNode = &STRestParameterNode{}

func (n *STRestParameterNode) Kind() common.SyntaxKind {
	return common.REST_PARAM
}

func (n *STRestParameterNode) BucketCount() int {
	return 4
}

func (n *STRestParameterNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.TypeName

	case 2:
		return n.EllipsisToken

	case 3:
		return n.ParamName

	default:
		panic("invalid bucket index")
	}
}

func (n *STRestParameterNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.TypeName,

		n.EllipsisToken,

		n.ParamName,
	}
}

func (n *STRestParameterNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RestParameterNode{
		ParameterNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STImportOrgNameNode struct {
	STNode

	OrgName STNode

	SlashToken STNode
}

var _ STNode = &STImportOrgNameNode{}

func (n *STImportOrgNameNode) Kind() common.SyntaxKind {
	return common.IMPORT_ORG_NAME
}

func (n *STImportOrgNameNode) BucketCount() int {
	return 2
}

func (n *STImportOrgNameNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OrgName

	case 1:
		return n.SlashToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STImportOrgNameNode) ChildBuckets() []STNode {
	return []STNode{

		n.OrgName,

		n.SlashToken,
	}
}

func (n *STImportOrgNameNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ImportOrgNameNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STImportPrefixNode struct {
	STNode

	AsKeyword STNode

	Prefix STNode
}

var _ STNode = &STImportPrefixNode{}

func (n *STImportPrefixNode) Kind() common.SyntaxKind {
	return common.IMPORT_PREFIX
}

func (n *STImportPrefixNode) BucketCount() int {
	return 2
}

func (n *STImportPrefixNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.AsKeyword

	case 1:
		return n.Prefix

	default:
		panic("invalid bucket index")
	}
}

func (n *STImportPrefixNode) ChildBuckets() []STNode {
	return []STNode{

		n.AsKeyword,

		n.Prefix,
	}
}

func (n *STImportPrefixNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ImportPrefixNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMappingFieldNode = STNode

type STSpecificFieldNode struct {
	STMappingFieldNode

	ReadonlyKeyword STNode

	FieldName STNode

	Colon STNode

	ValueExpr STNode
}

var _ STNode = &STSpecificFieldNode{}

func (n *STSpecificFieldNode) Kind() common.SyntaxKind {
	return common.SPECIFIC_FIELD
}

func (n *STSpecificFieldNode) BucketCount() int {
	return 4
}

func (n *STSpecificFieldNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReadonlyKeyword

	case 1:
		return n.FieldName

	case 2:
		return n.Colon

	case 3:
		return n.ValueExpr

	default:
		panic("invalid bucket index")
	}
}

func (n *STSpecificFieldNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReadonlyKeyword,

		n.FieldName,

		n.Colon,

		n.ValueExpr,
	}
}

func (n *STSpecificFieldNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &SpecificFieldNode{
		MappingFieldNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STSpreadFieldNode struct {
	STMappingFieldNode

	Ellipsis STNode

	ValueExpr STNode
}

var _ STNode = &STSpreadFieldNode{}

func (n *STSpreadFieldNode) Kind() common.SyntaxKind {
	return common.SPREAD_FIELD
}

func (n *STSpreadFieldNode) BucketCount() int {
	return 2
}

func (n *STSpreadFieldNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Ellipsis

	case 1:
		return n.ValueExpr

	default:
		panic("invalid bucket index")
	}
}

func (n *STSpreadFieldNode) ChildBuckets() []STNode {
	return []STNode{

		n.Ellipsis,

		n.ValueExpr,
	}
}

func (n *STSpreadFieldNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &SpreadFieldNode{
		MappingFieldNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFunctionArgumentNode = STNode

type STNamedArgumentNode struct {
	STFunctionArgumentNode

	ArgumentName STNode

	EqualsToken STNode

	Expression STNode
}

var _ STNode = &STNamedArgumentNode{}

func (n *STNamedArgumentNode) Kind() common.SyntaxKind {
	return common.NAMED_ARG
}

func (n *STNamedArgumentNode) BucketCount() int {
	return 3
}

func (n *STNamedArgumentNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ArgumentName

	case 1:
		return n.EqualsToken

	case 2:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STNamedArgumentNode) ChildBuckets() []STNode {
	return []STNode{

		n.ArgumentName,

		n.EqualsToken,

		n.Expression,
	}
}

func (n *STNamedArgumentNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NamedArgumentNode{
		FunctionArgumentNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STPositionalArgumentNode struct {
	STFunctionArgumentNode

	Expression STNode
}

var _ STNode = &STPositionalArgumentNode{}

func (n *STPositionalArgumentNode) Kind() common.SyntaxKind {
	return common.POSITIONAL_ARG
}

func (n *STPositionalArgumentNode) BucketCount() int {
	return 1
}

func (n *STPositionalArgumentNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STPositionalArgumentNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,
	}
}

func (n *STPositionalArgumentNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &PositionalArgumentNode{
		FunctionArgumentNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRestArgumentNode struct {
	STFunctionArgumentNode

	Ellipsis STNode

	Expression STNode
}

var _ STNode = &STRestArgumentNode{}

func (n *STRestArgumentNode) Kind() common.SyntaxKind {
	return common.REST_ARG
}

func (n *STRestArgumentNode) BucketCount() int {
	return 2
}

func (n *STRestArgumentNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Ellipsis

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STRestArgumentNode) ChildBuckets() []STNode {
	return []STNode{

		n.Ellipsis,

		n.Expression,
	}
}

func (n *STRestArgumentNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RestArgumentNode{
		FunctionArgumentNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STInferredTypedescDefaultNode struct {
	STExpressionNode

	LtToken STNode

	GtToken STNode
}

var _ STNode = &STInferredTypedescDefaultNode{}

func (n *STInferredTypedescDefaultNode) Kind() common.SyntaxKind {
	return common.INFERRED_TYPEDESC_DEFAULT
}

func (n *STInferredTypedescDefaultNode) BucketCount() int {
	return 2
}

func (n *STInferredTypedescDefaultNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LtToken

	case 1:
		return n.GtToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STInferredTypedescDefaultNode) ChildBuckets() []STNode {
	return []STNode{

		n.LtToken,

		n.GtToken,
	}
}

func (n *STInferredTypedescDefaultNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &InferredTypedescDefaultNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STObjectTypeDescriptorNode struct {
	STTypeDescriptorNode

	ObjectTypeQualifiers STNode

	ObjectKeyword STNode

	OpenBrace STNode

	Members STNode

	CloseBrace STNode
}

var _ STNode = &STObjectTypeDescriptorNode{}

func (n *STObjectTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.OBJECT_TYPE_DESC
}

func (n *STObjectTypeDescriptorNode) BucketCount() int {
	return 5
}

func (n *STObjectTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ObjectTypeQualifiers

	case 1:
		return n.ObjectKeyword

	case 2:
		return n.OpenBrace

	case 3:
		return n.Members

	case 4:
		return n.CloseBrace

	default:
		panic("invalid bucket index")
	}
}

func (n *STObjectTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.ObjectTypeQualifiers,

		n.ObjectKeyword,

		n.OpenBrace,

		n.Members,

		n.CloseBrace,
	}
}

func (n *STObjectTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ObjectTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STObjectConstructorExpressionNode struct {
	STExpressionNode

	Annotations STNode

	ObjectTypeQualifiers STNode

	ObjectKeyword STNode

	TypeReference STNode

	OpenBraceToken STNode

	Members STNode

	CloseBraceToken STNode
}

var _ STNode = &STObjectConstructorExpressionNode{}

func (n *STObjectConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.OBJECT_CONSTRUCTOR
}

func (n *STObjectConstructorExpressionNode) BucketCount() int {
	return 7
}

func (n *STObjectConstructorExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.ObjectTypeQualifiers

	case 2:
		return n.ObjectKeyword

	case 3:
		return n.TypeReference

	case 4:
		return n.OpenBraceToken

	case 5:
		return n.Members

	case 6:
		return n.CloseBraceToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STObjectConstructorExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.ObjectTypeQualifiers,

		n.ObjectKeyword,

		n.TypeReference,

		n.OpenBraceToken,

		n.Members,

		n.CloseBraceToken,
	}
}

func (n *STObjectConstructorExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ObjectConstructorExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRecordTypeDescriptorNode struct {
	STTypeDescriptorNode

	RecordKeyword STNode

	BodyStartDelimiter STNode

	Fields STNode

	RecordRestDescriptor STNode

	BodyEndDelimiter STNode
}

var _ STNode = &STRecordTypeDescriptorNode{}

func (n *STRecordTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.RECORD_TYPE_DESC
}

func (n *STRecordTypeDescriptorNode) BucketCount() int {
	return 5
}

func (n *STRecordTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.RecordKeyword

	case 1:
		return n.BodyStartDelimiter

	case 2:
		return n.Fields

	case 3:
		return n.RecordRestDescriptor

	case 4:
		return n.BodyEndDelimiter

	default:
		panic("invalid bucket index")
	}
}

func (n *STRecordTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.RecordKeyword,

		n.BodyStartDelimiter,

		n.Fields,

		n.RecordRestDescriptor,

		n.BodyEndDelimiter,
	}
}

func (n *STRecordTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RecordTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReturnTypeDescriptorNode struct {
	STNode

	ReturnsKeyword STNode

	Annotations STNode

	Type STNode
}

var _ STNode = &STReturnTypeDescriptorNode{}

func (n *STReturnTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.RETURN_TYPE_DESCRIPTOR
}

func (n *STReturnTypeDescriptorNode) BucketCount() int {
	return 3
}

func (n *STReturnTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReturnsKeyword

	case 1:
		return n.Annotations

	case 2:
		return n.Type

	default:
		panic("invalid bucket index")
	}
}

func (n *STReturnTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReturnsKeyword,

		n.Annotations,

		n.Type,
	}
}

func (n *STReturnTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReturnTypeDescriptorNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNilTypeDescriptorNode struct {
	STTypeDescriptorNode

	OpenParenToken STNode

	CloseParenToken STNode
}

var _ STNode = &STNilTypeDescriptorNode{}

func (n *STNilTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.NIL_TYPE_DESC
}

func (n *STNilTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STNilTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParenToken

	case 1:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STNilTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParenToken,

		n.CloseParenToken,
	}
}

func (n *STNilTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NilTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STOptionalTypeDescriptorNode struct {
	STTypeDescriptorNode

	TypeDescriptor STNode

	QuestionMarkToken STNode
}

var _ STNode = &STOptionalTypeDescriptorNode{}

func (n *STOptionalTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.OPTIONAL_TYPE_DESC
}

func (n *STOptionalTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STOptionalTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TypeDescriptor

	case 1:
		return n.QuestionMarkToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STOptionalTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.TypeDescriptor,

		n.QuestionMarkToken,
	}
}

func (n *STOptionalTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &OptionalTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STObjectFieldNode struct {
	STNode

	Metadata STNode

	VisibilityQualifier STNode

	QualifierList STNode

	TypeName STNode

	FieldName STNode

	EqualsToken STNode

	Expression STNode

	SemicolonToken STNode
}

var _ STNode = &STObjectFieldNode{}

func (n *STObjectFieldNode) Kind() common.SyntaxKind {
	return common.OBJECT_FIELD
}

func (n *STObjectFieldNode) BucketCount() int {
	return 8
}

func (n *STObjectFieldNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.VisibilityQualifier

	case 2:
		return n.QualifierList

	case 3:
		return n.TypeName

	case 4:
		return n.FieldName

	case 5:
		return n.EqualsToken

	case 6:
		return n.Expression

	case 7:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STObjectFieldNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.VisibilityQualifier,

		n.QualifierList,

		n.TypeName,

		n.FieldName,

		n.EqualsToken,

		n.Expression,

		n.SemicolonToken,
	}
}

func (n *STObjectFieldNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ObjectFieldNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRecordFieldNode struct {
	STNode

	Metadata STNode

	ReadonlyKeyword STNode

	TypeName STNode

	FieldName STNode

	QuestionMarkToken STNode

	SemicolonToken STNode
}

var _ STNode = &STRecordFieldNode{}

func (n *STRecordFieldNode) Kind() common.SyntaxKind {
	return common.RECORD_FIELD
}

func (n *STRecordFieldNode) BucketCount() int {
	return 6
}

func (n *STRecordFieldNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.ReadonlyKeyword

	case 2:
		return n.TypeName

	case 3:
		return n.FieldName

	case 4:
		return n.QuestionMarkToken

	case 5:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STRecordFieldNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.ReadonlyKeyword,

		n.TypeName,

		n.FieldName,

		n.QuestionMarkToken,

		n.SemicolonToken,
	}
}

func (n *STRecordFieldNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RecordFieldNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRecordFieldWithDefaultValueNode struct {
	STNode

	Metadata STNode

	ReadonlyKeyword STNode

	TypeName STNode

	FieldName STNode

	EqualsToken STNode

	Expression STNode

	SemicolonToken STNode
}

var _ STNode = &STRecordFieldWithDefaultValueNode{}

func (n *STRecordFieldWithDefaultValueNode) Kind() common.SyntaxKind {
	return common.RECORD_FIELD_WITH_DEFAULT_VALUE
}

func (n *STRecordFieldWithDefaultValueNode) BucketCount() int {
	return 7
}

func (n *STRecordFieldWithDefaultValueNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.ReadonlyKeyword

	case 2:
		return n.TypeName

	case 3:
		return n.FieldName

	case 4:
		return n.EqualsToken

	case 5:
		return n.Expression

	case 6:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STRecordFieldWithDefaultValueNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.ReadonlyKeyword,

		n.TypeName,

		n.FieldName,

		n.EqualsToken,

		n.Expression,

		n.SemicolonToken,
	}
}

func (n *STRecordFieldWithDefaultValueNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RecordFieldWithDefaultValueNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRecordRestDescriptorNode struct {
	STNode

	TypeName STNode

	EllipsisToken STNode

	SemicolonToken STNode
}

var _ STNode = &STRecordRestDescriptorNode{}

func (n *STRecordRestDescriptorNode) Kind() common.SyntaxKind {
	return common.RECORD_REST_TYPE
}

func (n *STRecordRestDescriptorNode) BucketCount() int {
	return 3
}

func (n *STRecordRestDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TypeName

	case 1:
		return n.EllipsisToken

	case 2:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STRecordRestDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.TypeName,

		n.EllipsisToken,

		n.SemicolonToken,
	}
}

func (n *STRecordRestDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RecordRestDescriptorNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeReferenceNode struct {
	STTypeDescriptorNode

	AsteriskToken STNode

	TypeName STNode

	SemicolonToken STNode
}

var _ STNode = &STTypeReferenceNode{}

func (n *STTypeReferenceNode) Kind() common.SyntaxKind {
	return common.TYPE_REFERENCE
}

func (n *STTypeReferenceNode) BucketCount() int {
	return 3
}

func (n *STTypeReferenceNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.AsteriskToken

	case 1:
		return n.TypeName

	case 2:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeReferenceNode) ChildBuckets() []STNode {
	return []STNode{

		n.AsteriskToken,

		n.TypeName,

		n.SemicolonToken,
	}
}

func (n *STTypeReferenceNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeReferenceNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STAnnotationNode struct {
	STNode

	AtToken STNode

	AnnotReference STNode

	AnnotValue STNode
}

var _ STNode = &STAnnotationNode{}

func (n *STAnnotationNode) Kind() common.SyntaxKind {
	return common.ANNOTATION
}

func (n *STAnnotationNode) BucketCount() int {
	return 3
}

func (n *STAnnotationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.AtToken

	case 1:
		return n.AnnotReference

	case 2:
		return n.AnnotValue

	default:
		panic("invalid bucket index")
	}
}

func (n *STAnnotationNode) ChildBuckets() []STNode {
	return []STNode{

		n.AtToken,

		n.AnnotReference,

		n.AnnotValue,
	}
}

func (n *STAnnotationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &AnnotationNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMetadataNode struct {
	STNode

	DocumentationString STNode

	Annotations STNode
}

var _ STNode = &STMetadataNode{}

func (n *STMetadataNode) Kind() common.SyntaxKind {
	return common.METADATA
}

func (n *STMetadataNode) BucketCount() int {
	return 2
}

func (n *STMetadataNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.DocumentationString

	case 1:
		return n.Annotations

	default:
		panic("invalid bucket index")
	}
}

func (n *STMetadataNode) ChildBuckets() []STNode {
	return []STNode{

		n.DocumentationString,

		n.Annotations,
	}
}

func (n *STMetadataNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MetadataNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STModuleVariableDeclarationNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	VisibilityQualifier STNode

	Qualifiers STNode

	TypedBindingPattern STNode

	EqualsToken STNode

	Initializer STNode

	SemicolonToken STNode
}

var _ STNode = &STModuleVariableDeclarationNode{}

func (n *STModuleVariableDeclarationNode) Kind() common.SyntaxKind {
	return common.MODULE_VAR_DECL
}

func (n *STModuleVariableDeclarationNode) BucketCount() int {
	return 7
}

func (n *STModuleVariableDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.VisibilityQualifier

	case 2:
		return n.Qualifiers

	case 3:
		return n.TypedBindingPattern

	case 4:
		return n.EqualsToken

	case 5:
		return n.Initializer

	case 6:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STModuleVariableDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.VisibilityQualifier,

		n.Qualifiers,

		n.TypedBindingPattern,

		n.EqualsToken,

		n.Initializer,

		n.SemicolonToken,
	}
}

func (n *STModuleVariableDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ModuleVariableDeclarationNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeTestExpressionNode struct {
	STExpressionNode

	Expression STNode

	IsKeyword STNode

	TypeDescriptor STNode
}

var _ STNode = &STTypeTestExpressionNode{}

func (n *STTypeTestExpressionNode) Kind() common.SyntaxKind {
	return common.TYPE_TEST_EXPRESSION
}

func (n *STTypeTestExpressionNode) BucketCount() int {
	return 3
}

func (n *STTypeTestExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.IsKeyword

	case 2:
		return n.TypeDescriptor

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeTestExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.IsKeyword,

		n.TypeDescriptor,
	}
}

func (n *STTypeTestExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeTestExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STActionNode = STExpressionNode

type STRemoteMethodCallActionNode struct {
	STActionNode

	Expression STNode

	RightArrowToken STNode

	MethodName STNode

	OpenParenToken STNode

	Arguments STNode

	CloseParenToken STNode
}

var _ STNode = &STRemoteMethodCallActionNode{}

func (n *STRemoteMethodCallActionNode) Kind() common.SyntaxKind {
	return common.REMOTE_METHOD_CALL_ACTION
}

func (n *STRemoteMethodCallActionNode) BucketCount() int {
	return 6
}

func (n *STRemoteMethodCallActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.RightArrowToken

	case 2:
		return n.MethodName

	case 3:
		return n.OpenParenToken

	case 4:
		return n.Arguments

	case 5:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STRemoteMethodCallActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.RightArrowToken,

		n.MethodName,

		n.OpenParenToken,

		n.Arguments,

		n.CloseParenToken,
	}
}

func (n *STRemoteMethodCallActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RemoteMethodCallActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMapTypeDescriptorNode struct {
	STTypeDescriptorNode

	MapKeywordToken STNode

	MapTypeParamsNode STNode
}

var _ STNode = &STMapTypeDescriptorNode{}

func (n *STMapTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.MAP_TYPE_DESC
}

func (n *STMapTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STMapTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.MapKeywordToken

	case 1:
		return n.MapTypeParamsNode

	default:
		panic("invalid bucket index")
	}
}

func (n *STMapTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.MapKeywordToken,

		n.MapTypeParamsNode,
	}
}

func (n *STMapTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MapTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNilLiteralNode struct {
	STExpressionNode

	OpenParenToken STNode

	CloseParenToken STNode
}

var _ STNode = &STNilLiteralNode{}

func (n *STNilLiteralNode) Kind() common.SyntaxKind {
	return common.NIL_LITERAL
}

func (n *STNilLiteralNode) BucketCount() int {
	return 2
}

func (n *STNilLiteralNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParenToken

	case 1:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STNilLiteralNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParenToken,

		n.CloseParenToken,
	}
}

func (n *STNilLiteralNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NilLiteralNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STAnnotationDeclarationNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	VisibilityQualifier STNode

	ConstKeyword STNode

	AnnotationKeyword STNode

	TypeDescriptor STNode

	AnnotationTag STNode

	OnKeyword STNode

	AttachPoints STNode

	SemicolonToken STNode
}

var _ STNode = &STAnnotationDeclarationNode{}

func (n *STAnnotationDeclarationNode) Kind() common.SyntaxKind {
	return common.ANNOTATION_DECLARATION
}

func (n *STAnnotationDeclarationNode) BucketCount() int {
	return 9
}

func (n *STAnnotationDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.VisibilityQualifier

	case 2:
		return n.ConstKeyword

	case 3:
		return n.AnnotationKeyword

	case 4:
		return n.TypeDescriptor

	case 5:
		return n.AnnotationTag

	case 6:
		return n.OnKeyword

	case 7:
		return n.AttachPoints

	case 8:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STAnnotationDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.VisibilityQualifier,

		n.ConstKeyword,

		n.AnnotationKeyword,

		n.TypeDescriptor,

		n.AnnotationTag,

		n.OnKeyword,

		n.AttachPoints,

		n.SemicolonToken,
	}
}

func (n *STAnnotationDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &AnnotationDeclarationNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STAnnotationAttachPointNode struct {
	STNode

	SourceKeyword STNode

	Identifiers STNode
}

var _ STNode = &STAnnotationAttachPointNode{}

func (n *STAnnotationAttachPointNode) Kind() common.SyntaxKind {
	return common.ANNOTATION_ATTACH_POINT
}

func (n *STAnnotationAttachPointNode) BucketCount() int {
	return 2
}

func (n *STAnnotationAttachPointNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.SourceKeyword

	case 1:
		return n.Identifiers

	default:
		panic("invalid bucket index")
	}
}

func (n *STAnnotationAttachPointNode) ChildBuckets() []STNode {
	return []STNode{

		n.SourceKeyword,

		n.Identifiers,
	}
}

func (n *STAnnotationAttachPointNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &AnnotationAttachPointNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLNamespaceDeclarationNode struct {
	STStatementNode

	XmlnsKeyword STNode

	Namespaceuri STNode

	AsKeyword STNode

	NamespacePrefix STNode

	SemicolonToken STNode
}

var _ STNode = &STXMLNamespaceDeclarationNode{}

func (n *STXMLNamespaceDeclarationNode) Kind() common.SyntaxKind {
	return common.XML_NAMESPACE_DECLARATION
}

func (n *STXMLNamespaceDeclarationNode) BucketCount() int {
	return 5
}

func (n *STXMLNamespaceDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.XmlnsKeyword

	case 1:
		return n.Namespaceuri

	case 2:
		return n.AsKeyword

	case 3:
		return n.NamespacePrefix

	case 4:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLNamespaceDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.XmlnsKeyword,

		n.Namespaceuri,

		n.AsKeyword,

		n.NamespacePrefix,

		n.SemicolonToken,
	}
}

func (n *STXMLNamespaceDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLNamespaceDeclarationNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STModuleXMLNamespaceDeclarationNode struct {
	STModuleMemberDeclarationNode

	XmlnsKeyword STNode

	Namespaceuri STNode

	AsKeyword STNode

	NamespacePrefix STNode

	SemicolonToken STNode
}

var _ STNode = &STModuleXMLNamespaceDeclarationNode{}

func (n *STModuleXMLNamespaceDeclarationNode) Kind() common.SyntaxKind {
	return common.MODULE_XML_NAMESPACE_DECLARATION
}

func (n *STModuleXMLNamespaceDeclarationNode) BucketCount() int {
	return 5
}

func (n *STModuleXMLNamespaceDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.XmlnsKeyword

	case 1:
		return n.Namespaceuri

	case 2:
		return n.AsKeyword

	case 3:
		return n.NamespacePrefix

	case 4:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STModuleXMLNamespaceDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.XmlnsKeyword,

		n.Namespaceuri,

		n.AsKeyword,

		n.NamespacePrefix,

		n.SemicolonToken,
	}
}

func (n *STModuleXMLNamespaceDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ModuleXMLNamespaceDeclarationNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFunctionBodyBlockNode struct {
	STFunctionBodyNode

	OpenBraceToken STNode

	NamedWorkerDeclarator STNode

	Statements STNode

	CloseBraceToken STNode

	SemicolonToken STNode
}

var _ STNode = &STFunctionBodyBlockNode{}

func (n *STFunctionBodyBlockNode) Kind() common.SyntaxKind {
	return common.FUNCTION_BODY_BLOCK
}

func (n *STFunctionBodyBlockNode) BucketCount() int {
	return 5
}

func (n *STFunctionBodyBlockNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBraceToken

	case 1:
		return n.NamedWorkerDeclarator

	case 2:
		return n.Statements

	case 3:
		return n.CloseBraceToken

	case 4:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STFunctionBodyBlockNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBraceToken,

		n.NamedWorkerDeclarator,

		n.Statements,

		n.CloseBraceToken,

		n.SemicolonToken,
	}
}

func (n *STFunctionBodyBlockNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FunctionBodyBlockNode{
		FunctionBodyNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNamedWorkerDeclarationNode struct {
	STNode

	Annotations STNode

	TransactionalKeyword STNode

	WorkerKeyword STNode

	WorkerName STNode

	ReturnTypeDesc STNode

	WorkerBody STNode

	OnFailClause STNode
}

var _ STNode = &STNamedWorkerDeclarationNode{}

func (n *STNamedWorkerDeclarationNode) Kind() common.SyntaxKind {
	return common.NAMED_WORKER_DECLARATION
}

func (n *STNamedWorkerDeclarationNode) BucketCount() int {
	return 7
}

func (n *STNamedWorkerDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.TransactionalKeyword

	case 2:
		return n.WorkerKeyword

	case 3:
		return n.WorkerName

	case 4:
		return n.ReturnTypeDesc

	case 5:
		return n.WorkerBody

	case 6:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STNamedWorkerDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.TransactionalKeyword,

		n.WorkerKeyword,

		n.WorkerName,

		n.ReturnTypeDesc,

		n.WorkerBody,

		n.OnFailClause,
	}
}

func (n *STNamedWorkerDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NamedWorkerDeclarationNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNamedWorkerDeclarator struct {
	STNode

	WorkerInitStatements STNode

	NamedWorkerDeclarations STNode
}

var _ STNode = &STNamedWorkerDeclarator{}

func (n *STNamedWorkerDeclarator) Kind() common.SyntaxKind {
	return common.NAMED_WORKER_DECLARATOR
}

func (n *STNamedWorkerDeclarator) BucketCount() int {
	return 2
}

func (n *STNamedWorkerDeclarator) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.WorkerInitStatements

	case 1:
		return n.NamedWorkerDeclarations

	default:
		panic("invalid bucket index")
	}
}

func (n *STNamedWorkerDeclarator) ChildBuckets() []STNode {
	return []STNode{

		n.WorkerInitStatements,

		n.NamedWorkerDeclarations,
	}
}

func (n *STNamedWorkerDeclarator) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NamedWorkerDeclarator{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STBasicLiteralNode struct {
	STExpressionNode

	LiteralToken STNode
}

var _ STNode = &STBasicLiteralNode{}

func (n *STBasicLiteralNode) Kind() common.SyntaxKind {
	return n.STExpressionNode.Kind()
}

func (n *STBasicLiteralNode) BucketCount() int {
	return 1
}

func (n *STBasicLiteralNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LiteralToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STBasicLiteralNode) ChildBuckets() []STNode {
	return []STNode{

		n.LiteralToken,
	}
}

func (n *STBasicLiteralNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &BasicLiteralNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeDescriptorNode = STExpressionNode

type STNameReferenceNode = STTypeDescriptorNode

type STSimpleNameReferenceNode struct {
	STNameReferenceNode

	Name STNode
}

var _ STNode = &STSimpleNameReferenceNode{}

func (n *STSimpleNameReferenceNode) Kind() common.SyntaxKind {
	return common.SIMPLE_NAME_REFERENCE
}

func (n *STSimpleNameReferenceNode) BucketCount() int {
	return 1
}

func (n *STSimpleNameReferenceNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Name

	default:
		panic("invalid bucket index")
	}
}

func (n *STSimpleNameReferenceNode) ChildBuckets() []STNode {
	return []STNode{

		n.Name,
	}
}

func (n *STSimpleNameReferenceNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &SimpleNameReferenceNode{
		NameReferenceNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STQualifiedNameReferenceNode struct {
	STNameReferenceNode

	ModulePrefix STNode

	Colon STNode

	Identifier STNode
}

var _ STNode = &STQualifiedNameReferenceNode{}

func (n *STQualifiedNameReferenceNode) Kind() common.SyntaxKind {
	return common.QUALIFIED_NAME_REFERENCE
}

func (n *STQualifiedNameReferenceNode) BucketCount() int {
	return 3
}

func (n *STQualifiedNameReferenceNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ModulePrefix

	case 1:
		return n.Colon

	case 2:
		return n.Identifier

	default:
		panic("invalid bucket index")
	}
}

func (n *STQualifiedNameReferenceNode) ChildBuckets() []STNode {
	return []STNode{

		n.ModulePrefix,

		n.Colon,

		n.Identifier,
	}
}

func (n *STQualifiedNameReferenceNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &QualifiedNameReferenceNode{
		NameReferenceNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STBuiltinSimpleNameReferenceNode struct {
	STNameReferenceNode

	Name STNode
}

var _ STNode = &STBuiltinSimpleNameReferenceNode{}

func (n *STBuiltinSimpleNameReferenceNode) Kind() common.SyntaxKind {
	return n.STNameReferenceNode.Kind()
}

func (n *STBuiltinSimpleNameReferenceNode) BucketCount() int {
	return 1
}

func (n *STBuiltinSimpleNameReferenceNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Name

	default:
		panic("invalid bucket index")
	}
}

func (n *STBuiltinSimpleNameReferenceNode) ChildBuckets() []STNode {
	return []STNode{

		n.Name,
	}
}

func (n *STBuiltinSimpleNameReferenceNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &BuiltinSimpleNameReferenceNode{
		NameReferenceNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTrapExpressionNode struct {
	STExpressionNode

	TrapKeyword STNode

	Expression STNode
}

var _ STNode = &STTrapExpressionNode{}

func (n *STTrapExpressionNode) Kind() common.SyntaxKind {
	return n.STExpressionNode.Kind()
}

func (n *STTrapExpressionNode) BucketCount() int {
	return 2
}

func (n *STTrapExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TrapKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STTrapExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.TrapKeyword,

		n.Expression,
	}
}

func (n *STTrapExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TrapExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STListConstructorExpressionNode struct {
	STExpressionNode

	OpenBracket STNode

	Expressions STNode

	CloseBracket STNode
}

var _ STNode = &STListConstructorExpressionNode{}

func (n *STListConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.LIST_CONSTRUCTOR
}

func (n *STListConstructorExpressionNode) BucketCount() int {
	return 3
}

func (n *STListConstructorExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracket

	case 1:
		return n.Expressions

	case 2:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STListConstructorExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracket,

		n.Expressions,

		n.CloseBracket,
	}
}

func (n *STListConstructorExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ListConstructorExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeCastExpressionNode struct {
	STExpressionNode

	LtToken STNode

	TypeCastParam STNode

	GtToken STNode

	Expression STNode
}

var _ STNode = &STTypeCastExpressionNode{}

func (n *STTypeCastExpressionNode) Kind() common.SyntaxKind {
	return common.TYPE_CAST_EXPRESSION
}

func (n *STTypeCastExpressionNode) BucketCount() int {
	return 4
}

func (n *STTypeCastExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LtToken

	case 1:
		return n.TypeCastParam

	case 2:
		return n.GtToken

	case 3:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeCastExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.LtToken,

		n.TypeCastParam,

		n.GtToken,

		n.Expression,
	}
}

func (n *STTypeCastExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeCastExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeCastParamNode struct {
	STNode

	Annotations STNode

	Type STNode
}

var _ STNode = &STTypeCastParamNode{}

func (n *STTypeCastParamNode) Kind() common.SyntaxKind {
	return common.TYPE_CAST_PARAM
}

func (n *STTypeCastParamNode) BucketCount() int {
	return 2
}

func (n *STTypeCastParamNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.Type

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeCastParamNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.Type,
	}
}

func (n *STTypeCastParamNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeCastParamNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STUnionTypeDescriptorNode struct {
	STTypeDescriptorNode

	LeftTypeDesc STNode

	PipeToken STNode

	RightTypeDesc STNode
}

var _ STNode = &STUnionTypeDescriptorNode{}

func (n *STUnionTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.UNION_TYPE_DESC
}

func (n *STUnionTypeDescriptorNode) BucketCount() int {
	return 3
}

func (n *STUnionTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LeftTypeDesc

	case 1:
		return n.PipeToken

	case 2:
		return n.RightTypeDesc

	default:
		panic("invalid bucket index")
	}
}

func (n *STUnionTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.LeftTypeDesc,

		n.PipeToken,

		n.RightTypeDesc,
	}
}

func (n *STUnionTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &UnionTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTableConstructorExpressionNode struct {
	STExpressionNode

	TableKeyword STNode

	KeySpecifier STNode

	OpenBracket STNode

	Rows STNode

	CloseBracket STNode
}

var _ STNode = &STTableConstructorExpressionNode{}

func (n *STTableConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.TABLE_CONSTRUCTOR
}

func (n *STTableConstructorExpressionNode) BucketCount() int {
	return 5
}

func (n *STTableConstructorExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TableKeyword

	case 1:
		return n.KeySpecifier

	case 2:
		return n.OpenBracket

	case 3:
		return n.Rows

	case 4:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STTableConstructorExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.TableKeyword,

		n.KeySpecifier,

		n.OpenBracket,

		n.Rows,

		n.CloseBracket,
	}
}

func (n *STTableConstructorExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TableConstructorExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STKeySpecifierNode struct {
	STNode

	KeyKeyword STNode

	OpenParenToken STNode

	FieldNames STNode

	CloseParenToken STNode
}

var _ STNode = &STKeySpecifierNode{}

func (n *STKeySpecifierNode) Kind() common.SyntaxKind {
	return common.KEY_SPECIFIER
}

func (n *STKeySpecifierNode) BucketCount() int {
	return 4
}

func (n *STKeySpecifierNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.KeyKeyword

	case 1:
		return n.OpenParenToken

	case 2:
		return n.FieldNames

	case 3:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STKeySpecifierNode) ChildBuckets() []STNode {
	return []STNode{

		n.KeyKeyword,

		n.OpenParenToken,

		n.FieldNames,

		n.CloseParenToken,
	}
}

func (n *STKeySpecifierNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &KeySpecifierNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STStreamTypeDescriptorNode struct {
	STTypeDescriptorNode

	StreamKeywordToken STNode

	StreamTypeParamsNode STNode
}

var _ STNode = &STStreamTypeDescriptorNode{}

func (n *STStreamTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.STREAM_TYPE_DESC
}

func (n *STStreamTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STStreamTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.StreamKeywordToken

	case 1:
		return n.StreamTypeParamsNode

	default:
		panic("invalid bucket index")
	}
}

func (n *STStreamTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.StreamKeywordToken,

		n.StreamTypeParamsNode,
	}
}

func (n *STStreamTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &StreamTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STStreamTypeParamsNode struct {
	STNode

	LtToken STNode

	LeftTypeDescNode STNode

	CommaToken STNode

	RightTypeDescNode STNode

	GtToken STNode
}

var _ STNode = &STStreamTypeParamsNode{}

func (n *STStreamTypeParamsNode) Kind() common.SyntaxKind {
	return common.STREAM_TYPE_PARAMS
}

func (n *STStreamTypeParamsNode) BucketCount() int {
	return 5
}

func (n *STStreamTypeParamsNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LtToken

	case 1:
		return n.LeftTypeDescNode

	case 2:
		return n.CommaToken

	case 3:
		return n.RightTypeDescNode

	case 4:
		return n.GtToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STStreamTypeParamsNode) ChildBuckets() []STNode {
	return []STNode{

		n.LtToken,

		n.LeftTypeDescNode,

		n.CommaToken,

		n.RightTypeDescNode,

		n.GtToken,
	}
}

func (n *STStreamTypeParamsNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &StreamTypeParamsNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STLetExpressionNode struct {
	STExpressionNode

	LetKeyword STNode

	LetVarDeclarations STNode

	InKeyword STNode

	Expression STNode
}

var _ STNode = &STLetExpressionNode{}

func (n *STLetExpressionNode) Kind() common.SyntaxKind {
	return common.LET_EXPRESSION
}

func (n *STLetExpressionNode) BucketCount() int {
	return 4
}

func (n *STLetExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LetKeyword

	case 1:
		return n.LetVarDeclarations

	case 2:
		return n.InKeyword

	case 3:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STLetExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.LetKeyword,

		n.LetVarDeclarations,

		n.InKeyword,

		n.Expression,
	}
}

func (n *STLetExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &LetExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STLetVariableDeclarationNode struct {
	STNode

	Annotations STNode

	TypedBindingPattern STNode

	EqualsToken STNode

	Expression STNode
}

var _ STNode = &STLetVariableDeclarationNode{}

func (n *STLetVariableDeclarationNode) Kind() common.SyntaxKind {
	return common.LET_VAR_DECL
}

func (n *STLetVariableDeclarationNode) BucketCount() int {
	return 4
}

func (n *STLetVariableDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.TypedBindingPattern

	case 2:
		return n.EqualsToken

	case 3:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STLetVariableDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.TypedBindingPattern,

		n.EqualsToken,

		n.Expression,
	}
}

func (n *STLetVariableDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &LetVariableDeclarationNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTemplateExpressionNode struct {
	STExpressionNode

	Type STNode

	StartBacktick STNode

	Content STNode

	EndBacktick STNode
}

var _ STNode = &STTemplateExpressionNode{}

func (n *STTemplateExpressionNode) Kind() common.SyntaxKind {
	return n.STExpressionNode.Kind()
}

func (n *STTemplateExpressionNode) BucketCount() int {
	return 4
}

func (n *STTemplateExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Type

	case 1:
		return n.StartBacktick

	case 2:
		return n.Content

	case 3:
		return n.EndBacktick

	default:
		panic("invalid bucket index")
	}
}

func (n *STTemplateExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Type,

		n.StartBacktick,

		n.Content,

		n.EndBacktick,
	}
}

func (n *STTemplateExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TemplateExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLItemNode = STNode

type STXMLElementNode struct {
	STXMLItemNode

	StartTag STNode

	Content STNode

	EndTag STNode
}

var _ STNode = &STXMLElementNode{}

func (n *STXMLElementNode) Kind() common.SyntaxKind {
	return common.XML_ELEMENT
}

func (n *STXMLElementNode) BucketCount() int {
	return 3
}

func (n *STXMLElementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.StartTag

	case 1:
		return n.Content

	case 2:
		return n.EndTag

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLElementNode) ChildBuckets() []STNode {
	return []STNode{

		n.StartTag,

		n.Content,

		n.EndTag,
	}
}

func (n *STXMLElementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLElementNode{
		XMLItemNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLElementTagNode = STNode

type STXMLStartTagNode struct {
	STXMLElementTagNode

	LtToken STNode

	Name STNode

	Attributes STNode

	GtToken STNode
}

var _ STNode = &STXMLStartTagNode{}

func (n *STXMLStartTagNode) Kind() common.SyntaxKind {
	return common.XML_ELEMENT_START_TAG
}

func (n *STXMLStartTagNode) BucketCount() int {
	return 4
}

func (n *STXMLStartTagNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LtToken

	case 1:
		return n.Name

	case 2:
		return n.Attributes

	case 3:
		return n.GtToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLStartTagNode) ChildBuckets() []STNode {
	return []STNode{

		n.LtToken,

		n.Name,

		n.Attributes,

		n.GtToken,
	}
}

func (n *STXMLStartTagNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLStartTagNode{
		XMLElementTagNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLEndTagNode struct {
	STXMLElementTagNode

	LtToken STNode

	SlashToken STNode

	Name STNode

	GtToken STNode
}

var _ STNode = &STXMLEndTagNode{}

func (n *STXMLEndTagNode) Kind() common.SyntaxKind {
	return common.XML_ELEMENT_END_TAG
}

func (n *STXMLEndTagNode) BucketCount() int {
	return 4
}

func (n *STXMLEndTagNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LtToken

	case 1:
		return n.SlashToken

	case 2:
		return n.Name

	case 3:
		return n.GtToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLEndTagNode) ChildBuckets() []STNode {
	return []STNode{

		n.LtToken,

		n.SlashToken,

		n.Name,

		n.GtToken,
	}
}

func (n *STXMLEndTagNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLEndTagNode{
		XMLElementTagNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLNameNode = STNode

type STXMLSimpleNameNode struct {
	STXMLNameNode

	Name STNode
}

var _ STNode = &STXMLSimpleNameNode{}

func (n *STXMLSimpleNameNode) Kind() common.SyntaxKind {
	return common.XML_SIMPLE_NAME
}

func (n *STXMLSimpleNameNode) BucketCount() int {
	return 1
}

func (n *STXMLSimpleNameNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Name

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLSimpleNameNode) ChildBuckets() []STNode {
	return []STNode{

		n.Name,
	}
}

func (n *STXMLSimpleNameNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLSimpleNameNode{
		XMLNameNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLQualifiedNameNode struct {
	STXMLNameNode

	Prefix STNode

	Colon STNode

	Name STNode
}

var _ STNode = &STXMLQualifiedNameNode{}

func (n *STXMLQualifiedNameNode) Kind() common.SyntaxKind {
	return common.XML_QUALIFIED_NAME
}

func (n *STXMLQualifiedNameNode) BucketCount() int {
	return 3
}

func (n *STXMLQualifiedNameNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Prefix

	case 1:
		return n.Colon

	case 2:
		return n.Name

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLQualifiedNameNode) ChildBuckets() []STNode {
	return []STNode{

		n.Prefix,

		n.Colon,

		n.Name,
	}
}

func (n *STXMLQualifiedNameNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLQualifiedNameNode{
		XMLNameNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLEmptyElementNode struct {
	STXMLItemNode

	LtToken STNode

	Name STNode

	Attributes STNode

	SlashToken STNode

	GtToken STNode
}

var _ STNode = &STXMLEmptyElementNode{}

func (n *STXMLEmptyElementNode) Kind() common.SyntaxKind {
	return common.XML_EMPTY_ELEMENT
}

func (n *STXMLEmptyElementNode) BucketCount() int {
	return 5
}

func (n *STXMLEmptyElementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LtToken

	case 1:
		return n.Name

	case 2:
		return n.Attributes

	case 3:
		return n.SlashToken

	case 4:
		return n.GtToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLEmptyElementNode) ChildBuckets() []STNode {
	return []STNode{

		n.LtToken,

		n.Name,

		n.Attributes,

		n.SlashToken,

		n.GtToken,
	}
}

func (n *STXMLEmptyElementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLEmptyElementNode{
		XMLItemNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STInterpolationNode struct {
	STXMLItemNode

	InterpolationStartToken STNode

	Expression STNode

	InterpolationEndToken STNode
}

var _ STNode = &STInterpolationNode{}

func (n *STInterpolationNode) Kind() common.SyntaxKind {
	return common.INTERPOLATION
}

func (n *STInterpolationNode) BucketCount() int {
	return 3
}

func (n *STInterpolationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.InterpolationStartToken

	case 1:
		return n.Expression

	case 2:
		return n.InterpolationEndToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STInterpolationNode) ChildBuckets() []STNode {
	return []STNode{

		n.InterpolationStartToken,

		n.Expression,

		n.InterpolationEndToken,
	}
}

func (n *STInterpolationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &InterpolationNode{
		XMLItemNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLTextNode struct {
	STXMLItemNode

	Content STNode
}

var _ STNode = &STXMLTextNode{}

func (n *STXMLTextNode) Kind() common.SyntaxKind {
	return common.XML_TEXT
}

func (n *STXMLTextNode) BucketCount() int {
	return 1
}

func (n *STXMLTextNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Content

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLTextNode) ChildBuckets() []STNode {
	return []STNode{

		n.Content,
	}
}

func (n *STXMLTextNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLTextNode{
		XMLItemNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLAttributeNode struct {
	STNode

	AttributeName STNode

	EqualToken STNode

	Value STNode
}

var _ STNode = &STXMLAttributeNode{}

func (n *STXMLAttributeNode) Kind() common.SyntaxKind {
	return common.XML_ATTRIBUTE
}

func (n *STXMLAttributeNode) BucketCount() int {
	return 3
}

func (n *STXMLAttributeNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.AttributeName

	case 1:
		return n.EqualToken

	case 2:
		return n.Value

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLAttributeNode) ChildBuckets() []STNode {
	return []STNode{

		n.AttributeName,

		n.EqualToken,

		n.Value,
	}
}

func (n *STXMLAttributeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLAttributeNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLAttributeValue struct {
	STNode

	StartQuote STNode

	Value STNode

	EndQuote STNode
}

var _ STNode = &STXMLAttributeValue{}

func (n *STXMLAttributeValue) Kind() common.SyntaxKind {
	return common.XML_ATTRIBUTE_VALUE
}

func (n *STXMLAttributeValue) BucketCount() int {
	return 3
}

func (n *STXMLAttributeValue) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.StartQuote

	case 1:
		return n.Value

	case 2:
		return n.EndQuote

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLAttributeValue) ChildBuckets() []STNode {
	return []STNode{

		n.StartQuote,

		n.Value,

		n.EndQuote,
	}
}

func (n *STXMLAttributeValue) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLAttributeValue{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLComment struct {
	STXMLItemNode

	CommentStart STNode

	Content STNode

	CommentEnd STNode
}

var _ STNode = &STXMLComment{}

func (n *STXMLComment) Kind() common.SyntaxKind {
	return common.XML_COMMENT
}

func (n *STXMLComment) BucketCount() int {
	return 3
}

func (n *STXMLComment) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.CommentStart

	case 1:
		return n.Content

	case 2:
		return n.CommentEnd

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLComment) ChildBuckets() []STNode {
	return []STNode{

		n.CommentStart,

		n.Content,

		n.CommentEnd,
	}
}

func (n *STXMLComment) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLComment{
		XMLItemNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLCDATANode struct {
	STXMLItemNode

	CdataStart STNode

	Content STNode

	CdataEnd STNode
}

var _ STNode = &STXMLCDATANode{}

func (n *STXMLCDATANode) Kind() common.SyntaxKind {
	return common.XML_CDATA
}

func (n *STXMLCDATANode) BucketCount() int {
	return 3
}

func (n *STXMLCDATANode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.CdataStart

	case 1:
		return n.Content

	case 2:
		return n.CdataEnd

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLCDATANode) ChildBuckets() []STNode {
	return []STNode{

		n.CdataStart,

		n.Content,

		n.CdataEnd,
	}
}

func (n *STXMLCDATANode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLCDATANode{
		XMLItemNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLProcessingInstruction struct {
	STXMLItemNode

	PiStart STNode

	Target STNode

	Data STNode

	PiEnd STNode
}

var _ STNode = &STXMLProcessingInstruction{}

func (n *STXMLProcessingInstruction) Kind() common.SyntaxKind {
	return common.XML_PI
}

func (n *STXMLProcessingInstruction) BucketCount() int {
	return 4
}

func (n *STXMLProcessingInstruction) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.PiStart

	case 1:
		return n.Target

	case 2:
		return n.Data

	case 3:
		return n.PiEnd

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLProcessingInstruction) ChildBuckets() []STNode {
	return []STNode{

		n.PiStart,

		n.Target,

		n.Data,

		n.PiEnd,
	}
}

func (n *STXMLProcessingInstruction) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLProcessingInstruction{
		XMLItemNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTableTypeDescriptorNode struct {
	STTypeDescriptorNode

	TableKeywordToken STNode

	RowTypeParameterNode STNode

	KeyConstraintNode STNode
}

var _ STNode = &STTableTypeDescriptorNode{}

func (n *STTableTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.TABLE_TYPE_DESC
}

func (n *STTableTypeDescriptorNode) BucketCount() int {
	return 3
}

func (n *STTableTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TableKeywordToken

	case 1:
		return n.RowTypeParameterNode

	case 2:
		return n.KeyConstraintNode

	default:
		panic("invalid bucket index")
	}
}

func (n *STTableTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.TableKeywordToken,

		n.RowTypeParameterNode,

		n.KeyConstraintNode,
	}
}

func (n *STTableTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TableTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeParameterNode struct {
	STNode

	LtToken STNode

	TypeNode STNode

	GtToken STNode
}

var _ STNode = &STTypeParameterNode{}

func (n *STTypeParameterNode) Kind() common.SyntaxKind {
	return common.TYPE_PARAMETER
}

func (n *STTypeParameterNode) BucketCount() int {
	return 3
}

func (n *STTypeParameterNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LtToken

	case 1:
		return n.TypeNode

	case 2:
		return n.GtToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeParameterNode) ChildBuckets() []STNode {
	return []STNode{

		n.LtToken,

		n.TypeNode,

		n.GtToken,
	}
}

func (n *STTypeParameterNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeParameterNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STKeyTypeConstraintNode struct {
	STNode

	KeyKeywordToken STNode

	TypeParameterNode STNode
}

var _ STNode = &STKeyTypeConstraintNode{}

func (n *STKeyTypeConstraintNode) Kind() common.SyntaxKind {
	return common.KEY_TYPE_CONSTRAINT
}

func (n *STKeyTypeConstraintNode) BucketCount() int {
	return 2
}

func (n *STKeyTypeConstraintNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.KeyKeywordToken

	case 1:
		return n.TypeParameterNode

	default:
		panic("invalid bucket index")
	}
}

func (n *STKeyTypeConstraintNode) ChildBuckets() []STNode {
	return []STNode{

		n.KeyKeywordToken,

		n.TypeParameterNode,
	}
}

func (n *STKeyTypeConstraintNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &KeyTypeConstraintNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFunctionTypeDescriptorNode struct {
	STTypeDescriptorNode

	QualifierList STNode

	FunctionKeyword STNode

	FunctionSignature STNode
}

var _ STNode = &STFunctionTypeDescriptorNode{}

func (n *STFunctionTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.FUNCTION_TYPE_DESC
}

func (n *STFunctionTypeDescriptorNode) BucketCount() int {
	return 3
}

func (n *STFunctionTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.QualifierList

	case 1:
		return n.FunctionKeyword

	case 2:
		return n.FunctionSignature

	default:
		panic("invalid bucket index")
	}
}

func (n *STFunctionTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.QualifierList,

		n.FunctionKeyword,

		n.FunctionSignature,
	}
}

func (n *STFunctionTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FunctionTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFunctionSignatureNode struct {
	STNode

	OpenParenToken STNode

	Parameters STNode

	CloseParenToken STNode

	ReturnTypeDesc STNode
}

var _ STNode = &STFunctionSignatureNode{}

func (n *STFunctionSignatureNode) Kind() common.SyntaxKind {
	return common.FUNCTION_SIGNATURE
}

func (n *STFunctionSignatureNode) BucketCount() int {
	return 4
}

func (n *STFunctionSignatureNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParenToken

	case 1:
		return n.Parameters

	case 2:
		return n.CloseParenToken

	case 3:
		return n.ReturnTypeDesc

	default:
		panic("invalid bucket index")
	}
}

func (n *STFunctionSignatureNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParenToken,

		n.Parameters,

		n.CloseParenToken,

		n.ReturnTypeDesc,
	}
}

func (n *STFunctionSignatureNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FunctionSignatureNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STAnonymousFunctionExpressionNode = STExpressionNode

type STExplicitAnonymousFunctionExpressionNode struct {
	STAnonymousFunctionExpressionNode

	Annotations STNode

	QualifierList STNode

	FunctionKeyword STNode

	FunctionSignature STNode

	FunctionBody STNode
}

var _ STNode = &STExplicitAnonymousFunctionExpressionNode{}

func (n *STExplicitAnonymousFunctionExpressionNode) Kind() common.SyntaxKind {
	return common.EXPLICIT_ANONYMOUS_FUNCTION_EXPRESSION
}

func (n *STExplicitAnonymousFunctionExpressionNode) BucketCount() int {
	return 5
}

func (n *STExplicitAnonymousFunctionExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.QualifierList

	case 2:
		return n.FunctionKeyword

	case 3:
		return n.FunctionSignature

	case 4:
		return n.FunctionBody

	default:
		panic("invalid bucket index")
	}
}

func (n *STExplicitAnonymousFunctionExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.QualifierList,

		n.FunctionKeyword,

		n.FunctionSignature,

		n.FunctionBody,
	}
}

func (n *STExplicitAnonymousFunctionExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ExplicitAnonymousFunctionExpressionNode{
		AnonymousFunctionExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFunctionBodyNode = STNode

type STExpressionFunctionBodyNode struct {
	STFunctionBodyNode

	RightDoubleArrow STNode

	Expression STNode

	Semicolon STNode
}

var _ STNode = &STExpressionFunctionBodyNode{}

func (n *STExpressionFunctionBodyNode) Kind() common.SyntaxKind {
	return common.EXPRESSION_FUNCTION_BODY
}

func (n *STExpressionFunctionBodyNode) BucketCount() int {
	return 3
}

func (n *STExpressionFunctionBodyNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.RightDoubleArrow

	case 1:
		return n.Expression

	case 2:
		return n.Semicolon

	default:
		panic("invalid bucket index")
	}
}

func (n *STExpressionFunctionBodyNode) ChildBuckets() []STNode {
	return []STNode{

		n.RightDoubleArrow,

		n.Expression,

		n.Semicolon,
	}
}

func (n *STExpressionFunctionBodyNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ExpressionFunctionBodyNode{
		FunctionBodyNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTupleTypeDescriptorNode struct {
	STTypeDescriptorNode

	OpenBracketToken STNode

	MemberTypeDesc STNode

	CloseBracketToken STNode
}

var _ STNode = &STTupleTypeDescriptorNode{}

func (n *STTupleTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.TUPLE_TYPE_DESC
}

func (n *STTupleTypeDescriptorNode) BucketCount() int {
	return 3
}

func (n *STTupleTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracketToken

	case 1:
		return n.MemberTypeDesc

	case 2:
		return n.CloseBracketToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STTupleTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracketToken,

		n.MemberTypeDesc,

		n.CloseBracketToken,
	}
}

func (n *STTupleTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TupleTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STParenthesisedTypeDescriptorNode struct {
	STTypeDescriptorNode

	OpenParenToken STNode

	Typedesc STNode

	CloseParenToken STNode
}

var _ STNode = &STParenthesisedTypeDescriptorNode{}

func (n *STParenthesisedTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.PARENTHESISED_TYPE_DESC
}

func (n *STParenthesisedTypeDescriptorNode) BucketCount() int {
	return 3
}

func (n *STParenthesisedTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParenToken

	case 1:
		return n.Typedesc

	case 2:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STParenthesisedTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParenToken,

		n.Typedesc,

		n.CloseParenToken,
	}
}

func (n *STParenthesisedTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ParenthesisedTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNewExpressionNode = STExpressionNode

type STExplicitNewExpressionNode struct {
	STNewExpressionNode

	NewKeyword STNode

	TypeDescriptor STNode

	ParenthesizedArgList STNode
}

var _ STNode = &STExplicitNewExpressionNode{}

func (n *STExplicitNewExpressionNode) Kind() common.SyntaxKind {
	return common.EXPLICIT_NEW_EXPRESSION
}

func (n *STExplicitNewExpressionNode) BucketCount() int {
	return 3
}

func (n *STExplicitNewExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.NewKeyword

	case 1:
		return n.TypeDescriptor

	case 2:
		return n.ParenthesizedArgList

	default:
		panic("invalid bucket index")
	}
}

func (n *STExplicitNewExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.NewKeyword,

		n.TypeDescriptor,

		n.ParenthesizedArgList,
	}
}

func (n *STExplicitNewExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ExplicitNewExpressionNode{
		NewExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STImplicitNewExpressionNode struct {
	STNewExpressionNode

	NewKeyword STNode

	ParenthesizedArgList STNode
}

var _ STNode = &STImplicitNewExpressionNode{}

func (n *STImplicitNewExpressionNode) Kind() common.SyntaxKind {
	return common.IMPLICIT_NEW_EXPRESSION
}

func (n *STImplicitNewExpressionNode) BucketCount() int {
	return 2
}

func (n *STImplicitNewExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.NewKeyword

	case 1:
		return n.ParenthesizedArgList

	default:
		panic("invalid bucket index")
	}
}

func (n *STImplicitNewExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.NewKeyword,

		n.ParenthesizedArgList,
	}
}

func (n *STImplicitNewExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ImplicitNewExpressionNode{
		NewExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STParenthesizedArgList struct {
	STNode

	OpenParenToken STNode

	Arguments STNode

	CloseParenToken STNode
}

var _ STNode = &STParenthesizedArgList{}

func (n *STParenthesizedArgList) Kind() common.SyntaxKind {
	return common.PARENTHESIZED_ARG_LIST
}

func (n *STParenthesizedArgList) BucketCount() int {
	return 3
}

func (n *STParenthesizedArgList) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParenToken

	case 1:
		return n.Arguments

	case 2:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STParenthesizedArgList) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParenToken,

		n.Arguments,

		n.CloseParenToken,
	}
}

func (n *STParenthesizedArgList) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ParenthesizedArgList{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STClauseNode = STNode

type STIntermediateClauseNode = STClauseNode

type STQueryConstructTypeNode struct {
	STNode

	Keyword STNode

	KeySpecifier STNode
}

var _ STNode = &STQueryConstructTypeNode{}

func (n *STQueryConstructTypeNode) Kind() common.SyntaxKind {
	return common.QUERY_CONSTRUCT_TYPE
}

func (n *STQueryConstructTypeNode) BucketCount() int {
	return 2
}

func (n *STQueryConstructTypeNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Keyword

	case 1:
		return n.KeySpecifier

	default:
		panic("invalid bucket index")
	}
}

func (n *STQueryConstructTypeNode) ChildBuckets() []STNode {
	return []STNode{

		n.Keyword,

		n.KeySpecifier,
	}
}

func (n *STQueryConstructTypeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &QueryConstructTypeNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFromClauseNode struct {
	STIntermediateClauseNode

	FromKeyword STNode

	TypedBindingPattern STNode

	InKeyword STNode

	Expression STNode
}

var _ STNode = &STFromClauseNode{}

func (n *STFromClauseNode) Kind() common.SyntaxKind {
	return common.FROM_CLAUSE
}

func (n *STFromClauseNode) BucketCount() int {
	return 4
}

func (n *STFromClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FromKeyword

	case 1:
		return n.TypedBindingPattern

	case 2:
		return n.InKeyword

	case 3:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STFromClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.FromKeyword,

		n.TypedBindingPattern,

		n.InKeyword,

		n.Expression,
	}
}

func (n *STFromClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FromClauseNode{
		IntermediateClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STWhereClauseNode struct {
	STIntermediateClauseNode

	WhereKeyword STNode

	Expression STNode
}

var _ STNode = &STWhereClauseNode{}

func (n *STWhereClauseNode) Kind() common.SyntaxKind {
	return common.WHERE_CLAUSE
}

func (n *STWhereClauseNode) BucketCount() int {
	return 2
}

func (n *STWhereClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.WhereKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STWhereClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.WhereKeyword,

		n.Expression,
	}
}

func (n *STWhereClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &WhereClauseNode{
		IntermediateClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STLetClauseNode struct {
	STIntermediateClauseNode

	LetKeyword STNode

	LetVarDeclarations STNode
}

var _ STNode = &STLetClauseNode{}

func (n *STLetClauseNode) Kind() common.SyntaxKind {
	return common.LET_CLAUSE
}

func (n *STLetClauseNode) BucketCount() int {
	return 2
}

func (n *STLetClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LetKeyword

	case 1:
		return n.LetVarDeclarations

	default:
		panic("invalid bucket index")
	}
}

func (n *STLetClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.LetKeyword,

		n.LetVarDeclarations,
	}
}

func (n *STLetClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &LetClauseNode{
		IntermediateClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STJoinClauseNode struct {
	STIntermediateClauseNode

	OuterKeyword STNode

	JoinKeyword STNode

	TypedBindingPattern STNode

	InKeyword STNode

	Expression STNode

	JoinOnCondition STNode
}

var _ STNode = &STJoinClauseNode{}

func (n *STJoinClauseNode) Kind() common.SyntaxKind {
	return common.JOIN_CLAUSE
}

func (n *STJoinClauseNode) BucketCount() int {
	return 6
}

func (n *STJoinClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OuterKeyword

	case 1:
		return n.JoinKeyword

	case 2:
		return n.TypedBindingPattern

	case 3:
		return n.InKeyword

	case 4:
		return n.Expression

	case 5:
		return n.JoinOnCondition

	default:
		panic("invalid bucket index")
	}
}

func (n *STJoinClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.OuterKeyword,

		n.JoinKeyword,

		n.TypedBindingPattern,

		n.InKeyword,

		n.Expression,

		n.JoinOnCondition,
	}
}

func (n *STJoinClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &JoinClauseNode{
		IntermediateClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STOnClauseNode struct {
	STClauseNode

	OnKeyword STNode

	OnExpression STNode

	EqualsKeyword STNode

	EqualsExpression STNode
}

var _ STNode = &STOnClauseNode{}

func (n *STOnClauseNode) Kind() common.SyntaxKind {
	return common.ON_CLAUSE
}

func (n *STOnClauseNode) BucketCount() int {
	return 4
}

func (n *STOnClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OnKeyword

	case 1:
		return n.OnExpression

	case 2:
		return n.EqualsKeyword

	case 3:
		return n.EqualsExpression

	default:
		panic("invalid bucket index")
	}
}

func (n *STOnClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.OnKeyword,

		n.OnExpression,

		n.EqualsKeyword,

		n.EqualsExpression,
	}
}

func (n *STOnClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &OnClauseNode{
		ClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STLimitClauseNode struct {
	STIntermediateClauseNode

	LimitKeyword STNode

	Expression STNode
}

var _ STNode = &STLimitClauseNode{}

func (n *STLimitClauseNode) Kind() common.SyntaxKind {
	return common.LIMIT_CLAUSE
}

func (n *STLimitClauseNode) BucketCount() int {
	return 2
}

func (n *STLimitClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LimitKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STLimitClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.LimitKeyword,

		n.Expression,
	}
}

func (n *STLimitClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &LimitClauseNode{
		IntermediateClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STOnConflictClauseNode struct {
	STClauseNode

	OnKeyword STNode

	ConflictKeyword STNode

	Expression STNode
}

var _ STNode = &STOnConflictClauseNode{}

func (n *STOnConflictClauseNode) Kind() common.SyntaxKind {
	return common.ON_CONFLICT_CLAUSE
}

func (n *STOnConflictClauseNode) BucketCount() int {
	return 3
}

func (n *STOnConflictClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OnKeyword

	case 1:
		return n.ConflictKeyword

	case 2:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STOnConflictClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.OnKeyword,

		n.ConflictKeyword,

		n.Expression,
	}
}

func (n *STOnConflictClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &OnConflictClauseNode{
		ClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STQueryPipelineNode struct {
	STNode

	FromClause STNode

	IntermediateClauses STNode
}

var _ STNode = &STQueryPipelineNode{}

func (n *STQueryPipelineNode) Kind() common.SyntaxKind {
	return common.QUERY_PIPELINE
}

func (n *STQueryPipelineNode) BucketCount() int {
	return 2
}

func (n *STQueryPipelineNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FromClause

	case 1:
		return n.IntermediateClauses

	default:
		panic("invalid bucket index")
	}
}

func (n *STQueryPipelineNode) ChildBuckets() []STNode {
	return []STNode{

		n.FromClause,

		n.IntermediateClauses,
	}
}

func (n *STQueryPipelineNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &QueryPipelineNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STSelectClauseNode struct {
	STClauseNode

	SelectKeyword STNode

	Expression STNode
}

var _ STNode = &STSelectClauseNode{}

func (n *STSelectClauseNode) Kind() common.SyntaxKind {
	return common.SELECT_CLAUSE
}

func (n *STSelectClauseNode) BucketCount() int {
	return 2
}

func (n *STSelectClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.SelectKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STSelectClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.SelectKeyword,

		n.Expression,
	}
}

func (n *STSelectClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &SelectClauseNode{
		ClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STCollectClauseNode struct {
	STClauseNode

	CollectKeyword STNode

	Expression STNode
}

var _ STNode = &STCollectClauseNode{}

func (n *STCollectClauseNode) Kind() common.SyntaxKind {
	return common.COLLECT_CLAUSE
}

func (n *STCollectClauseNode) BucketCount() int {
	return 2
}

func (n *STCollectClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.CollectKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STCollectClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.CollectKeyword,

		n.Expression,
	}
}

func (n *STCollectClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &CollectClauseNode{
		ClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STQueryExpressionNode struct {
	STExpressionNode

	QueryConstructType STNode

	QueryPipeline STNode

	ResultClause STNode

	OnConflictClause STNode
}

var _ STNode = &STQueryExpressionNode{}

func (n *STQueryExpressionNode) Kind() common.SyntaxKind {
	return common.QUERY_EXPRESSION
}

func (n *STQueryExpressionNode) BucketCount() int {
	return 4
}

func (n *STQueryExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.QueryConstructType

	case 1:
		return n.QueryPipeline

	case 2:
		return n.ResultClause

	case 3:
		return n.OnConflictClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STQueryExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.QueryConstructType,

		n.QueryPipeline,

		n.ResultClause,

		n.OnConflictClause,
	}
}

func (n *STQueryExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &QueryExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STQueryActionNode struct {
	STActionNode

	QueryPipeline STNode

	DoKeyword STNode

	BlockStatement STNode
}

var _ STNode = &STQueryActionNode{}

func (n *STQueryActionNode) Kind() common.SyntaxKind {
	return common.QUERY_ACTION
}

func (n *STQueryActionNode) BucketCount() int {
	return 3
}

func (n *STQueryActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.QueryPipeline

	case 1:
		return n.DoKeyword

	case 2:
		return n.BlockStatement

	default:
		panic("invalid bucket index")
	}
}

func (n *STQueryActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.QueryPipeline,

		n.DoKeyword,

		n.BlockStatement,
	}
}

func (n *STQueryActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &QueryActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STIntersectionTypeDescriptorNode struct {
	STTypeDescriptorNode

	LeftTypeDesc STNode

	BitwiseAndToken STNode

	RightTypeDesc STNode
}

var _ STNode = &STIntersectionTypeDescriptorNode{}

func (n *STIntersectionTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.INTERSECTION_TYPE_DESC
}

func (n *STIntersectionTypeDescriptorNode) BucketCount() int {
	return 3
}

func (n *STIntersectionTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LeftTypeDesc

	case 1:
		return n.BitwiseAndToken

	case 2:
		return n.RightTypeDesc

	default:
		panic("invalid bucket index")
	}
}

func (n *STIntersectionTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.LeftTypeDesc,

		n.BitwiseAndToken,

		n.RightTypeDesc,
	}
}

func (n *STIntersectionTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &IntersectionTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STImplicitAnonymousFunctionParameters struct {
	STNode

	OpenParenToken STNode

	Parameters STNode

	CloseParenToken STNode
}

var _ STNode = &STImplicitAnonymousFunctionParameters{}

func (n *STImplicitAnonymousFunctionParameters) Kind() common.SyntaxKind {
	return common.INFER_PARAM_LIST
}

func (n *STImplicitAnonymousFunctionParameters) BucketCount() int {
	return 3
}

func (n *STImplicitAnonymousFunctionParameters) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParenToken

	case 1:
		return n.Parameters

	case 2:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STImplicitAnonymousFunctionParameters) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParenToken,

		n.Parameters,

		n.CloseParenToken,
	}
}

func (n *STImplicitAnonymousFunctionParameters) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ImplicitAnonymousFunctionParameters{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STImplicitAnonymousFunctionExpressionNode struct {
	STAnonymousFunctionExpressionNode

	Params STNode

	RightDoubleArrow STNode

	Expression STNode
}

var _ STNode = &STImplicitAnonymousFunctionExpressionNode{}

func (n *STImplicitAnonymousFunctionExpressionNode) Kind() common.SyntaxKind {
	return common.IMPLICIT_ANONYMOUS_FUNCTION_EXPRESSION
}

func (n *STImplicitAnonymousFunctionExpressionNode) BucketCount() int {
	return 3
}

func (n *STImplicitAnonymousFunctionExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Params

	case 1:
		return n.RightDoubleArrow

	case 2:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STImplicitAnonymousFunctionExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Params,

		n.RightDoubleArrow,

		n.Expression,
	}
}

func (n *STImplicitAnonymousFunctionExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ImplicitAnonymousFunctionExpressionNode{
		AnonymousFunctionExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STStartActionNode struct {
	STExpressionNode

	Annotations STNode

	StartKeyword STNode

	Expression STNode
}

var _ STNode = &STStartActionNode{}

func (n *STStartActionNode) Kind() common.SyntaxKind {
	return common.START_ACTION
}

func (n *STStartActionNode) BucketCount() int {
	return 3
}

func (n *STStartActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.StartKeyword

	case 2:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STStartActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.StartKeyword,

		n.Expression,
	}
}

func (n *STStartActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &StartActionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFlushActionNode struct {
	STExpressionNode

	FlushKeyword STNode

	PeerWorker STNode
}

var _ STNode = &STFlushActionNode{}

func (n *STFlushActionNode) Kind() common.SyntaxKind {
	return common.FLUSH_ACTION
}

func (n *STFlushActionNode) BucketCount() int {
	return 2
}

func (n *STFlushActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FlushKeyword

	case 1:
		return n.PeerWorker

	default:
		panic("invalid bucket index")
	}
}

func (n *STFlushActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.FlushKeyword,

		n.PeerWorker,
	}
}

func (n *STFlushActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FlushActionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STSingletonTypeDescriptorNode struct {
	STTypeDescriptorNode

	SimpleContExprNode STNode
}

var _ STNode = &STSingletonTypeDescriptorNode{}

func (n *STSingletonTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.SINGLETON_TYPE_DESC
}

func (n *STSingletonTypeDescriptorNode) BucketCount() int {
	return 1
}

func (n *STSingletonTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.SimpleContExprNode

	default:
		panic("invalid bucket index")
	}
}

func (n *STSingletonTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.SimpleContExprNode,
	}
}

func (n *STSingletonTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &SingletonTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMethodDeclarationNode struct {
	STNode

	Metadata STNode

	QualifierList STNode

	FunctionKeyword STNode

	MethodName STNode

	RelativeResourcePath STNode

	MethodSignature STNode

	Semicolon STNode
}

var _ STNode = &STMethodDeclarationNode{}

func (n *STMethodDeclarationNode) Kind() common.SyntaxKind {
	return n.STNode.Kind()
}

func (n *STMethodDeclarationNode) BucketCount() int {
	return 7
}

func (n *STMethodDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.QualifierList

	case 2:
		return n.FunctionKeyword

	case 3:
		return n.MethodName

	case 4:
		return n.RelativeResourcePath

	case 5:
		return n.MethodSignature

	case 6:
		return n.Semicolon

	default:
		panic("invalid bucket index")
	}
}

func (n *STMethodDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.QualifierList,

		n.FunctionKeyword,

		n.MethodName,

		n.RelativeResourcePath,

		n.MethodSignature,

		n.Semicolon,
	}
}

func (n *STMethodDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MethodDeclarationNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypedBindingPatternNode struct {
	STNode

	TypeDescriptor STNode

	BindingPattern STNode
}

var _ STNode = &STTypedBindingPatternNode{}

func (n *STTypedBindingPatternNode) Kind() common.SyntaxKind {
	return common.TYPED_BINDING_PATTERN
}

func (n *STTypedBindingPatternNode) BucketCount() int {
	return 2
}

func (n *STTypedBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TypeDescriptor

	case 1:
		return n.BindingPattern

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypedBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.TypeDescriptor,

		n.BindingPattern,
	}
}

func (n *STTypedBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypedBindingPatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STBindingPatternNode = STNode

type STCaptureBindingPatternNode struct {
	STBindingPatternNode

	VariableName STNode
}

var _ STNode = &STCaptureBindingPatternNode{}

func (n *STCaptureBindingPatternNode) Kind() common.SyntaxKind {
	return common.CAPTURE_BINDING_PATTERN
}

func (n *STCaptureBindingPatternNode) BucketCount() int {
	return 1
}

func (n *STCaptureBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.VariableName

	default:
		panic("invalid bucket index")
	}
}

func (n *STCaptureBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.VariableName,
	}
}

func (n *STCaptureBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &CaptureBindingPatternNode{
		BindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STWildcardBindingPatternNode struct {
	STBindingPatternNode

	UnderscoreToken STNode
}

var _ STNode = &STWildcardBindingPatternNode{}

func (n *STWildcardBindingPatternNode) Kind() common.SyntaxKind {
	return common.WILDCARD_BINDING_PATTERN
}

func (n *STWildcardBindingPatternNode) BucketCount() int {
	return 1
}

func (n *STWildcardBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.UnderscoreToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STWildcardBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.UnderscoreToken,
	}
}

func (n *STWildcardBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &WildcardBindingPatternNode{
		BindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STListBindingPatternNode struct {
	STBindingPatternNode

	OpenBracket STNode

	BindingPatterns STNode

	CloseBracket STNode
}

var _ STNode = &STListBindingPatternNode{}

func (n *STListBindingPatternNode) Kind() common.SyntaxKind {
	return common.LIST_BINDING_PATTERN
}

func (n *STListBindingPatternNode) BucketCount() int {
	return 3
}

func (n *STListBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracket

	case 1:
		return n.BindingPatterns

	case 2:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STListBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracket,

		n.BindingPatterns,

		n.CloseBracket,
	}
}

func (n *STListBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ListBindingPatternNode{
		BindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMappingBindingPatternNode struct {
	STBindingPatternNode

	OpenBrace STNode

	FieldBindingPatterns STNode

	CloseBrace STNode
}

var _ STNode = &STMappingBindingPatternNode{}

func (n *STMappingBindingPatternNode) Kind() common.SyntaxKind {
	return common.MAPPING_BINDING_PATTERN
}

func (n *STMappingBindingPatternNode) BucketCount() int {
	return 3
}

func (n *STMappingBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBrace

	case 1:
		return n.FieldBindingPatterns

	case 2:
		return n.CloseBrace

	default:
		panic("invalid bucket index")
	}
}

func (n *STMappingBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBrace,

		n.FieldBindingPatterns,

		n.CloseBrace,
	}
}

func (n *STMappingBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MappingBindingPatternNode{
		BindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFieldBindingPatternNode = STBindingPatternNode

type STFieldBindingPatternFullNode struct {
	STFieldBindingPatternNode

	VariableName STNode

	Colon STNode

	BindingPattern STNode
}

var _ STNode = &STFieldBindingPatternFullNode{}

func (n *STFieldBindingPatternFullNode) Kind() common.SyntaxKind {
	return common.FIELD_BINDING_PATTERN
}

func (n *STFieldBindingPatternFullNode) BucketCount() int {
	return 3
}

func (n *STFieldBindingPatternFullNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.VariableName

	case 1:
		return n.Colon

	case 2:
		return n.BindingPattern

	default:
		panic("invalid bucket index")
	}
}

func (n *STFieldBindingPatternFullNode) ChildBuckets() []STNode {
	return []STNode{

		n.VariableName,

		n.Colon,

		n.BindingPattern,
	}
}

func (n *STFieldBindingPatternFullNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FieldBindingPatternFullNode{
		FieldBindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFieldBindingPatternVarnameNode struct {
	STFieldBindingPatternNode

	VariableName STNode
}

var _ STNode = &STFieldBindingPatternVarnameNode{}

func (n *STFieldBindingPatternVarnameNode) Kind() common.SyntaxKind {
	return common.FIELD_BINDING_PATTERN
}

func (n *STFieldBindingPatternVarnameNode) BucketCount() int {
	return 1
}

func (n *STFieldBindingPatternVarnameNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.VariableName

	default:
		panic("invalid bucket index")
	}
}

func (n *STFieldBindingPatternVarnameNode) ChildBuckets() []STNode {
	return []STNode{

		n.VariableName,
	}
}

func (n *STFieldBindingPatternVarnameNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FieldBindingPatternVarnameNode{
		FieldBindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRestBindingPatternNode struct {
	STBindingPatternNode

	EllipsisToken STNode

	VariableName STNode
}

var _ STNode = &STRestBindingPatternNode{}

func (n *STRestBindingPatternNode) Kind() common.SyntaxKind {
	return common.REST_BINDING_PATTERN
}

func (n *STRestBindingPatternNode) BucketCount() int {
	return 2
}

func (n *STRestBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.EllipsisToken

	case 1:
		return n.VariableName

	default:
		panic("invalid bucket index")
	}
}

func (n *STRestBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.EllipsisToken,

		n.VariableName,
	}
}

func (n *STRestBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RestBindingPatternNode{
		BindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STErrorBindingPatternNode struct {
	STBindingPatternNode

	ErrorKeyword STNode

	TypeReference STNode

	OpenParenthesis STNode

	ArgListBindingPatterns STNode

	CloseParenthesis STNode
}

var _ STNode = &STErrorBindingPatternNode{}

func (n *STErrorBindingPatternNode) Kind() common.SyntaxKind {
	return common.ERROR_BINDING_PATTERN
}

func (n *STErrorBindingPatternNode) BucketCount() int {
	return 5
}

func (n *STErrorBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ErrorKeyword

	case 1:
		return n.TypeReference

	case 2:
		return n.OpenParenthesis

	case 3:
		return n.ArgListBindingPatterns

	case 4:
		return n.CloseParenthesis

	default:
		panic("invalid bucket index")
	}
}

func (n *STErrorBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.ErrorKeyword,

		n.TypeReference,

		n.OpenParenthesis,

		n.ArgListBindingPatterns,

		n.CloseParenthesis,
	}
}

func (n *STErrorBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ErrorBindingPatternNode{
		BindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNamedArgBindingPatternNode struct {
	STBindingPatternNode

	ArgName STNode

	EqualsToken STNode

	BindingPattern STNode
}

var _ STNode = &STNamedArgBindingPatternNode{}

func (n *STNamedArgBindingPatternNode) Kind() common.SyntaxKind {
	return common.NAMED_ARG_BINDING_PATTERN
}

func (n *STNamedArgBindingPatternNode) BucketCount() int {
	return 3
}

func (n *STNamedArgBindingPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ArgName

	case 1:
		return n.EqualsToken

	case 2:
		return n.BindingPattern

	default:
		panic("invalid bucket index")
	}
}

func (n *STNamedArgBindingPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.ArgName,

		n.EqualsToken,

		n.BindingPattern,
	}
}

func (n *STNamedArgBindingPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NamedArgBindingPatternNode{
		BindingPatternNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STAsyncSendActionNode struct {
	STActionNode

	Expression STNode

	RightArrowToken STNode

	PeerWorker STNode
}

var _ STNode = &STAsyncSendActionNode{}

func (n *STAsyncSendActionNode) Kind() common.SyntaxKind {
	return common.ASYNC_SEND_ACTION
}

func (n *STAsyncSendActionNode) BucketCount() int {
	return 3
}

func (n *STAsyncSendActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.RightArrowToken

	case 2:
		return n.PeerWorker

	default:
		panic("invalid bucket index")
	}
}

func (n *STAsyncSendActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.RightArrowToken,

		n.PeerWorker,
	}
}

func (n *STAsyncSendActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &AsyncSendActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STSyncSendActionNode struct {
	STActionNode

	Expression STNode

	SyncSendToken STNode

	PeerWorker STNode
}

var _ STNode = &STSyncSendActionNode{}

func (n *STSyncSendActionNode) Kind() common.SyntaxKind {
	return common.SYNC_SEND_ACTION
}

func (n *STSyncSendActionNode) BucketCount() int {
	return 3
}

func (n *STSyncSendActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.SyncSendToken

	case 2:
		return n.PeerWorker

	default:
		panic("invalid bucket index")
	}
}

func (n *STSyncSendActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.SyncSendToken,

		n.PeerWorker,
	}
}

func (n *STSyncSendActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &SyncSendActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReceiveActionNode struct {
	STActionNode

	LeftArrow STNode

	ReceiveWorkers STNode
}

var _ STNode = &STReceiveActionNode{}

func (n *STReceiveActionNode) Kind() common.SyntaxKind {
	return common.RECEIVE_ACTION
}

func (n *STReceiveActionNode) BucketCount() int {
	return 2
}

func (n *STReceiveActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LeftArrow

	case 1:
		return n.ReceiveWorkers

	default:
		panic("invalid bucket index")
	}
}

func (n *STReceiveActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.LeftArrow,

		n.ReceiveWorkers,
	}
}

func (n *STReceiveActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReceiveActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReceiveFieldsNode struct {
	STNode

	OpenBrace STNode

	ReceiveFields STNode

	CloseBrace STNode
}

var _ STNode = &STReceiveFieldsNode{}

func (n *STReceiveFieldsNode) Kind() common.SyntaxKind {
	return common.RECEIVE_FIELDS
}

func (n *STReceiveFieldsNode) BucketCount() int {
	return 3
}

func (n *STReceiveFieldsNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBrace

	case 1:
		return n.ReceiveFields

	case 2:
		return n.CloseBrace

	default:
		panic("invalid bucket index")
	}
}

func (n *STReceiveFieldsNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBrace,

		n.ReceiveFields,

		n.CloseBrace,
	}
}

func (n *STReceiveFieldsNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReceiveFieldsNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STAlternateReceiveNode struct {
	STNode

	Workers STNode
}

var _ STNode = &STAlternateReceiveNode{}

func (n *STAlternateReceiveNode) Kind() common.SyntaxKind {
	return common.ALTERNATE_RECEIVE
}

func (n *STAlternateReceiveNode) BucketCount() int {
	return 1
}

func (n *STAlternateReceiveNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Workers

	default:
		panic("invalid bucket index")
	}
}

func (n *STAlternateReceiveNode) ChildBuckets() []STNode {
	return []STNode{

		n.Workers,
	}
}

func (n *STAlternateReceiveNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &AlternateReceiveNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRestDescriptorNode struct {
	STNode

	TypeDescriptor STNode

	EllipsisToken STNode
}

var _ STNode = &STRestDescriptorNode{}

func (n *STRestDescriptorNode) Kind() common.SyntaxKind {
	return common.REST_TYPE
}

func (n *STRestDescriptorNode) BucketCount() int {
	return 2
}

func (n *STRestDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TypeDescriptor

	case 1:
		return n.EllipsisToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STRestDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.TypeDescriptor,

		n.EllipsisToken,
	}
}

func (n *STRestDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RestDescriptorNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STDoubleGTTokenNode struct {
	STNode

	OpenGTToken STNode

	EndGTToken STNode
}

var _ STNode = &STDoubleGTTokenNode{}

func (n *STDoubleGTTokenNode) Kind() common.SyntaxKind {
	return common.DOUBLE_GT_TOKEN
}

func (n *STDoubleGTTokenNode) BucketCount() int {
	return 2
}

func (n *STDoubleGTTokenNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenGTToken

	case 1:
		return n.EndGTToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STDoubleGTTokenNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenGTToken,

		n.EndGTToken,
	}
}

func (n *STDoubleGTTokenNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &DoubleGTTokenNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTrippleGTTokenNode struct {
	STNode

	OpenGTToken STNode

	MiddleGTToken STNode

	EndGTToken STNode
}

var _ STNode = &STTrippleGTTokenNode{}

func (n *STTrippleGTTokenNode) Kind() common.SyntaxKind {
	return common.TRIPPLE_GT_TOKEN
}

func (n *STTrippleGTTokenNode) BucketCount() int {
	return 3
}

func (n *STTrippleGTTokenNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenGTToken

	case 1:
		return n.MiddleGTToken

	case 2:
		return n.EndGTToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STTrippleGTTokenNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenGTToken,

		n.MiddleGTToken,

		n.EndGTToken,
	}
}

func (n *STTrippleGTTokenNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TrippleGTTokenNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STWaitActionNode struct {
	STActionNode

	WaitKeyword STNode

	WaitFutureExpr STNode
}

var _ STNode = &STWaitActionNode{}

func (n *STWaitActionNode) Kind() common.SyntaxKind {
	return common.WAIT_ACTION
}

func (n *STWaitActionNode) BucketCount() int {
	return 2
}

func (n *STWaitActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.WaitKeyword

	case 1:
		return n.WaitFutureExpr

	default:
		panic("invalid bucket index")
	}
}

func (n *STWaitActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.WaitKeyword,

		n.WaitFutureExpr,
	}
}

func (n *STWaitActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &WaitActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STWaitFieldsListNode struct {
	STNode

	OpenBrace STNode

	WaitFields STNode

	CloseBrace STNode
}

var _ STNode = &STWaitFieldsListNode{}

func (n *STWaitFieldsListNode) Kind() common.SyntaxKind {
	return common.WAIT_FIELDS_LIST
}

func (n *STWaitFieldsListNode) BucketCount() int {
	return 3
}

func (n *STWaitFieldsListNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBrace

	case 1:
		return n.WaitFields

	case 2:
		return n.CloseBrace

	default:
		panic("invalid bucket index")
	}
}

func (n *STWaitFieldsListNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBrace,

		n.WaitFields,

		n.CloseBrace,
	}
}

func (n *STWaitFieldsListNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &WaitFieldsListNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STWaitFieldNode struct {
	STNode

	FieldName STNode

	Colon STNode

	WaitFutureExpr STNode
}

var _ STNode = &STWaitFieldNode{}

func (n *STWaitFieldNode) Kind() common.SyntaxKind {
	return common.WAIT_FIELD
}

func (n *STWaitFieldNode) BucketCount() int {
	return 3
}

func (n *STWaitFieldNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FieldName

	case 1:
		return n.Colon

	case 2:
		return n.WaitFutureExpr

	default:
		panic("invalid bucket index")
	}
}

func (n *STWaitFieldNode) ChildBuckets() []STNode {
	return []STNode{

		n.FieldName,

		n.Colon,

		n.WaitFutureExpr,
	}
}

func (n *STWaitFieldNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &WaitFieldNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STAnnotAccessExpressionNode struct {
	STExpressionNode

	Expression STNode

	AnnotChainingToken STNode

	AnnotTagReference STNode
}

var _ STNode = &STAnnotAccessExpressionNode{}

func (n *STAnnotAccessExpressionNode) Kind() common.SyntaxKind {
	return common.ANNOT_ACCESS
}

func (n *STAnnotAccessExpressionNode) BucketCount() int {
	return 3
}

func (n *STAnnotAccessExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.AnnotChainingToken

	case 2:
		return n.AnnotTagReference

	default:
		panic("invalid bucket index")
	}
}

func (n *STAnnotAccessExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.AnnotChainingToken,

		n.AnnotTagReference,
	}
}

func (n *STAnnotAccessExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &AnnotAccessExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STOptionalFieldAccessExpressionNode struct {
	STExpressionNode

	Expression STNode

	OptionalChainingToken STNode

	FieldName STNode
}

var _ STNode = &STOptionalFieldAccessExpressionNode{}

func (n *STOptionalFieldAccessExpressionNode) Kind() common.SyntaxKind {
	return common.OPTIONAL_FIELD_ACCESS
}

func (n *STOptionalFieldAccessExpressionNode) BucketCount() int {
	return 3
}

func (n *STOptionalFieldAccessExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.OptionalChainingToken

	case 2:
		return n.FieldName

	default:
		panic("invalid bucket index")
	}
}

func (n *STOptionalFieldAccessExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.OptionalChainingToken,

		n.FieldName,
	}
}

func (n *STOptionalFieldAccessExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &OptionalFieldAccessExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STConditionalExpressionNode struct {
	STExpressionNode

	LhsExpression STNode

	QuestionMarkToken STNode

	MiddleExpression STNode

	ColonToken STNode

	EndExpression STNode
}

var _ STNode = &STConditionalExpressionNode{}

func (n *STConditionalExpressionNode) Kind() common.SyntaxKind {
	return common.CONDITIONAL_EXPRESSION
}

func (n *STConditionalExpressionNode) BucketCount() int {
	return 5
}

func (n *STConditionalExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LhsExpression

	case 1:
		return n.QuestionMarkToken

	case 2:
		return n.MiddleExpression

	case 3:
		return n.ColonToken

	case 4:
		return n.EndExpression

	default:
		panic("invalid bucket index")
	}
}

func (n *STConditionalExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.LhsExpression,

		n.QuestionMarkToken,

		n.MiddleExpression,

		n.ColonToken,

		n.EndExpression,
	}
}

func (n *STConditionalExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ConditionalExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STEnumDeclarationNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	Qualifier STNode

	EnumKeywordToken STNode

	Identifier STNode

	OpenBraceToken STNode

	EnumMemberList STNode

	CloseBraceToken STNode

	SemicolonToken STNode
}

var _ STNode = &STEnumDeclarationNode{}

func (n *STEnumDeclarationNode) Kind() common.SyntaxKind {
	return common.ENUM_DECLARATION
}

func (n *STEnumDeclarationNode) BucketCount() int {
	return 8
}

func (n *STEnumDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.Qualifier

	case 2:
		return n.EnumKeywordToken

	case 3:
		return n.Identifier

	case 4:
		return n.OpenBraceToken

	case 5:
		return n.EnumMemberList

	case 6:
		return n.CloseBraceToken

	case 7:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STEnumDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.Qualifier,

		n.EnumKeywordToken,

		n.Identifier,

		n.OpenBraceToken,

		n.EnumMemberList,

		n.CloseBraceToken,

		n.SemicolonToken,
	}
}

func (n *STEnumDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &EnumDeclarationNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STEnumMemberNode struct {
	STNode

	Metadata STNode

	Identifier STNode

	EqualToken STNode

	ConstExprNode STNode
}

var _ STNode = &STEnumMemberNode{}

func (n *STEnumMemberNode) Kind() common.SyntaxKind {
	return common.ENUM_MEMBER
}

func (n *STEnumMemberNode) BucketCount() int {
	return 4
}

func (n *STEnumMemberNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.Identifier

	case 2:
		return n.EqualToken

	case 3:
		return n.ConstExprNode

	default:
		panic("invalid bucket index")
	}
}

func (n *STEnumMemberNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.Identifier,

		n.EqualToken,

		n.ConstExprNode,
	}
}

func (n *STEnumMemberNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &EnumMemberNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STArrayTypeDescriptorNode struct {
	STTypeDescriptorNode

	MemberTypeDesc STNode

	Dimensions STNode
}

var _ STNode = &STArrayTypeDescriptorNode{}

func (n *STArrayTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.ARRAY_TYPE_DESC
}

func (n *STArrayTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STArrayTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.MemberTypeDesc

	case 1:
		return n.Dimensions

	default:
		panic("invalid bucket index")
	}
}

func (n *STArrayTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.MemberTypeDesc,

		n.Dimensions,
	}
}

func (n *STArrayTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ArrayTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STArrayDimensionNode struct {
	STNode

	OpenBracket STNode

	ArrayLength STNode

	CloseBracket STNode
}

var _ STNode = &STArrayDimensionNode{}

func (n *STArrayDimensionNode) Kind() common.SyntaxKind {
	return common.ARRAY_DIMENSION
}

func (n *STArrayDimensionNode) BucketCount() int {
	return 3
}

func (n *STArrayDimensionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracket

	case 1:
		return n.ArrayLength

	case 2:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STArrayDimensionNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracket,

		n.ArrayLength,

		n.CloseBracket,
	}
}

func (n *STArrayDimensionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ArrayDimensionNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTransactionStatementNode struct {
	STStatementNode

	TransactionKeyword STNode

	BlockStatement STNode

	OnFailClause STNode
}

var _ STNode = &STTransactionStatementNode{}

func (n *STTransactionStatementNode) Kind() common.SyntaxKind {
	return common.TRANSACTION_STATEMENT
}

func (n *STTransactionStatementNode) BucketCount() int {
	return 3
}

func (n *STTransactionStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TransactionKeyword

	case 1:
		return n.BlockStatement

	case 2:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STTransactionStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.TransactionKeyword,

		n.BlockStatement,

		n.OnFailClause,
	}
}

func (n *STTransactionStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TransactionStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRollbackStatementNode struct {
	STStatementNode

	RollbackKeyword STNode

	Expression STNode

	Semicolon STNode
}

var _ STNode = &STRollbackStatementNode{}

func (n *STRollbackStatementNode) Kind() common.SyntaxKind {
	return common.ROLLBACK_STATEMENT
}

func (n *STRollbackStatementNode) BucketCount() int {
	return 3
}

func (n *STRollbackStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.RollbackKeyword

	case 1:
		return n.Expression

	case 2:
		return n.Semicolon

	default:
		panic("invalid bucket index")
	}
}

func (n *STRollbackStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.RollbackKeyword,

		n.Expression,

		n.Semicolon,
	}
}

func (n *STRollbackStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RollbackStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRetryStatementNode struct {
	STStatementNode

	RetryKeyword STNode

	TypeParameter STNode

	Arguments STNode

	RetryBody STNode

	OnFailClause STNode
}

var _ STNode = &STRetryStatementNode{}

func (n *STRetryStatementNode) Kind() common.SyntaxKind {
	return common.RETRY_STATEMENT
}

func (n *STRetryStatementNode) BucketCount() int {
	return 5
}

func (n *STRetryStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.RetryKeyword

	case 1:
		return n.TypeParameter

	case 2:
		return n.Arguments

	case 3:
		return n.RetryBody

	case 4:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STRetryStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.RetryKeyword,

		n.TypeParameter,

		n.Arguments,

		n.RetryBody,

		n.OnFailClause,
	}
}

func (n *STRetryStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RetryStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STCommitActionNode struct {
	STActionNode

	CommitKeyword STNode
}

var _ STNode = &STCommitActionNode{}

func (n *STCommitActionNode) Kind() common.SyntaxKind {
	return common.COMMIT_ACTION
}

func (n *STCommitActionNode) BucketCount() int {
	return 1
}

func (n *STCommitActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.CommitKeyword

	default:
		panic("invalid bucket index")
	}
}

func (n *STCommitActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.CommitKeyword,
	}
}

func (n *STCommitActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &CommitActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTransactionalExpressionNode struct {
	STExpressionNode

	TransactionalKeyword STNode
}

var _ STNode = &STTransactionalExpressionNode{}

func (n *STTransactionalExpressionNode) Kind() common.SyntaxKind {
	return common.TRANSACTIONAL_EXPRESSION
}

func (n *STTransactionalExpressionNode) BucketCount() int {
	return 1
}

func (n *STTransactionalExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TransactionalKeyword

	default:
		panic("invalid bucket index")
	}
}

func (n *STTransactionalExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.TransactionalKeyword,
	}
}

func (n *STTransactionalExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TransactionalExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STByteArrayLiteralNode struct {
	STExpressionNode

	Type STNode

	StartBacktick STNode

	Content STNode

	EndBacktick STNode
}

var _ STNode = &STByteArrayLiteralNode{}

func (n *STByteArrayLiteralNode) Kind() common.SyntaxKind {
	return common.BYTE_ARRAY_LITERAL
}

func (n *STByteArrayLiteralNode) BucketCount() int {
	return 4
}

func (n *STByteArrayLiteralNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Type

	case 1:
		return n.StartBacktick

	case 2:
		return n.Content

	case 3:
		return n.EndBacktick

	default:
		panic("invalid bucket index")
	}
}

func (n *STByteArrayLiteralNode) ChildBuckets() []STNode {
	return []STNode{

		n.Type,

		n.StartBacktick,

		n.Content,

		n.EndBacktick,
	}
}

func (n *STByteArrayLiteralNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ByteArrayLiteralNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLNavigateExpressionNode = STExpressionNode

type STXMLFilterExpressionNode struct {
	STXMLNavigateExpressionNode

	Expression STNode

	XmlPatternChain STNode
}

var _ STNode = &STXMLFilterExpressionNode{}

func (n *STXMLFilterExpressionNode) Kind() common.SyntaxKind {
	return common.XML_FILTER_EXPRESSION
}

func (n *STXMLFilterExpressionNode) BucketCount() int {
	return 2
}

func (n *STXMLFilterExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.XmlPatternChain

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLFilterExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.XmlPatternChain,
	}
}

func (n *STXMLFilterExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLFilterExpressionNode{
		XMLNavigateExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLStepExpressionNode struct {
	STXMLNavigateExpressionNode

	Expression STNode

	XmlStepStart STNode

	XmlStepExtend STNode
}

var _ STNode = &STXMLStepExpressionNode{}

func (n *STXMLStepExpressionNode) Kind() common.SyntaxKind {
	return common.XML_STEP_EXPRESSION
}

func (n *STXMLStepExpressionNode) BucketCount() int {
	return 3
}

func (n *STXMLStepExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.XmlStepStart

	case 2:
		return n.XmlStepExtend

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLStepExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.XmlStepStart,

		n.XmlStepExtend,
	}
}

func (n *STXMLStepExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLStepExpressionNode{
		XMLNavigateExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLNamePatternChainingNode struct {
	STNode

	StartToken STNode

	XmlNamePattern STNode

	GtToken STNode
}

var _ STNode = &STXMLNamePatternChainingNode{}

func (n *STXMLNamePatternChainingNode) Kind() common.SyntaxKind {
	return common.XML_NAME_PATTERN_CHAIN
}

func (n *STXMLNamePatternChainingNode) BucketCount() int {
	return 3
}

func (n *STXMLNamePatternChainingNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.StartToken

	case 1:
		return n.XmlNamePattern

	case 2:
		return n.GtToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLNamePatternChainingNode) ChildBuckets() []STNode {
	return []STNode{

		n.StartToken,

		n.XmlNamePattern,

		n.GtToken,
	}
}

func (n *STXMLNamePatternChainingNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLNamePatternChainingNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLStepIndexedExtendNode struct {
	STNode

	OpenBracket STNode

	Expression STNode

	CloseBracket STNode
}

var _ STNode = &STXMLStepIndexedExtendNode{}

func (n *STXMLStepIndexedExtendNode) Kind() common.SyntaxKind {
	return common.XML_STEP_INDEXED_EXTEND
}

func (n *STXMLStepIndexedExtendNode) BucketCount() int {
	return 3
}

func (n *STXMLStepIndexedExtendNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracket

	case 1:
		return n.Expression

	case 2:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLStepIndexedExtendNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracket,

		n.Expression,

		n.CloseBracket,
	}
}

func (n *STXMLStepIndexedExtendNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLStepIndexedExtendNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLStepMethodCallExtendNode struct {
	STNode

	DotToken STNode

	MethodName STNode

	ParenthesizedArgList STNode
}

var _ STNode = &STXMLStepMethodCallExtendNode{}

func (n *STXMLStepMethodCallExtendNode) Kind() common.SyntaxKind {
	return common.XML_STEP_METHOD_CALL_EXTEND
}

func (n *STXMLStepMethodCallExtendNode) BucketCount() int {
	return 3
}

func (n *STXMLStepMethodCallExtendNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.DotToken

	case 1:
		return n.MethodName

	case 2:
		return n.ParenthesizedArgList

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLStepMethodCallExtendNode) ChildBuckets() []STNode {
	return []STNode{

		n.DotToken,

		n.MethodName,

		n.ParenthesizedArgList,
	}
}

func (n *STXMLStepMethodCallExtendNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLStepMethodCallExtendNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STXMLAtomicNamePatternNode struct {
	STNode

	Prefix STNode

	Colon STNode

	Name STNode
}

var _ STNode = &STXMLAtomicNamePatternNode{}

func (n *STXMLAtomicNamePatternNode) Kind() common.SyntaxKind {
	return common.XML_ATOMIC_NAME_PATTERN
}

func (n *STXMLAtomicNamePatternNode) BucketCount() int {
	return 3
}

func (n *STXMLAtomicNamePatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Prefix

	case 1:
		return n.Colon

	case 2:
		return n.Name

	default:
		panic("invalid bucket index")
	}
}

func (n *STXMLAtomicNamePatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.Prefix,

		n.Colon,

		n.Name,
	}
}

func (n *STXMLAtomicNamePatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &XMLAtomicNamePatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STTypeReferenceTypeDescNode struct {
	STTypeDescriptorNode

	TypeRef STNode
}

var _ STNode = &STTypeReferenceTypeDescNode{}

func (n *STTypeReferenceTypeDescNode) Kind() common.SyntaxKind {
	return common.TYPE_REFERENCE_TYPE_DESC
}

func (n *STTypeReferenceTypeDescNode) BucketCount() int {
	return 1
}

func (n *STTypeReferenceTypeDescNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TypeRef

	default:
		panic("invalid bucket index")
	}
}

func (n *STTypeReferenceTypeDescNode) ChildBuckets() []STNode {
	return []STNode{

		n.TypeRef,
	}
}

func (n *STTypeReferenceTypeDescNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TypeReferenceTypeDescNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMatchStatementNode struct {
	STStatementNode

	MatchKeyword STNode

	Condition STNode

	OpenBrace STNode

	MatchClauses STNode

	CloseBrace STNode

	OnFailClause STNode
}

var _ STNode = &STMatchStatementNode{}

func (n *STMatchStatementNode) Kind() common.SyntaxKind {
	return common.MATCH_STATEMENT
}

func (n *STMatchStatementNode) BucketCount() int {
	return 6
}

func (n *STMatchStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.MatchKeyword

	case 1:
		return n.Condition

	case 2:
		return n.OpenBrace

	case 3:
		return n.MatchClauses

	case 4:
		return n.CloseBrace

	case 5:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STMatchStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.MatchKeyword,

		n.Condition,

		n.OpenBrace,

		n.MatchClauses,

		n.CloseBrace,

		n.OnFailClause,
	}
}

func (n *STMatchStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MatchStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMatchClauseNode struct {
	STNode

	MatchPatterns STNode

	MatchGuard STNode

	RightDoubleArrow STNode

	BlockStatement STNode
}

var _ STNode = &STMatchClauseNode{}

func (n *STMatchClauseNode) Kind() common.SyntaxKind {
	return common.MATCH_CLAUSE
}

func (n *STMatchClauseNode) BucketCount() int {
	return 4
}

func (n *STMatchClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.MatchPatterns

	case 1:
		return n.MatchGuard

	case 2:
		return n.RightDoubleArrow

	case 3:
		return n.BlockStatement

	default:
		panic("invalid bucket index")
	}
}

func (n *STMatchClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.MatchPatterns,

		n.MatchGuard,

		n.RightDoubleArrow,

		n.BlockStatement,
	}
}

func (n *STMatchClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MatchClauseNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMatchGuardNode struct {
	STNode

	IfKeyword STNode

	Expression STNode
}

var _ STNode = &STMatchGuardNode{}

func (n *STMatchGuardNode) Kind() common.SyntaxKind {
	return common.MATCH_GUARD
}

func (n *STMatchGuardNode) BucketCount() int {
	return 2
}

func (n *STMatchGuardNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.IfKeyword

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STMatchGuardNode) ChildBuckets() []STNode {
	return []STNode{

		n.IfKeyword,

		n.Expression,
	}
}

func (n *STMatchGuardNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MatchGuardNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STDistinctTypeDescriptorNode struct {
	STTypeDescriptorNode

	DistinctKeyword STNode

	TypeDescriptor STNode
}

var _ STNode = &STDistinctTypeDescriptorNode{}

func (n *STDistinctTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.DISTINCT_TYPE_DESC
}

func (n *STDistinctTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STDistinctTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.DistinctKeyword

	case 1:
		return n.TypeDescriptor

	default:
		panic("invalid bucket index")
	}
}

func (n *STDistinctTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.DistinctKeyword,

		n.TypeDescriptor,
	}
}

func (n *STDistinctTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &DistinctTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STListMatchPatternNode struct {
	STNode

	OpenBracket STNode

	MatchPatterns STNode

	CloseBracket STNode
}

var _ STNode = &STListMatchPatternNode{}

func (n *STListMatchPatternNode) Kind() common.SyntaxKind {
	return common.LIST_MATCH_PATTERN
}

func (n *STListMatchPatternNode) BucketCount() int {
	return 3
}

func (n *STListMatchPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracket

	case 1:
		return n.MatchPatterns

	case 2:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STListMatchPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracket,

		n.MatchPatterns,

		n.CloseBracket,
	}
}

func (n *STListMatchPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ListMatchPatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRestMatchPatternNode struct {
	STNode

	EllipsisToken STNode

	VarKeywordToken STNode

	VariableName STNode
}

var _ STNode = &STRestMatchPatternNode{}

func (n *STRestMatchPatternNode) Kind() common.SyntaxKind {
	return common.REST_MATCH_PATTERN
}

func (n *STRestMatchPatternNode) BucketCount() int {
	return 3
}

func (n *STRestMatchPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.EllipsisToken

	case 1:
		return n.VarKeywordToken

	case 2:
		return n.VariableName

	default:
		panic("invalid bucket index")
	}
}

func (n *STRestMatchPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.EllipsisToken,

		n.VarKeywordToken,

		n.VariableName,
	}
}

func (n *STRestMatchPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RestMatchPatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMappingMatchPatternNode struct {
	STNode

	OpenBraceToken STNode

	FieldMatchPatterns STNode

	CloseBraceToken STNode
}

var _ STNode = &STMappingMatchPatternNode{}

func (n *STMappingMatchPatternNode) Kind() common.SyntaxKind {
	return common.MAPPING_MATCH_PATTERN
}

func (n *STMappingMatchPatternNode) BucketCount() int {
	return 3
}

func (n *STMappingMatchPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBraceToken

	case 1:
		return n.FieldMatchPatterns

	case 2:
		return n.CloseBraceToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STMappingMatchPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBraceToken,

		n.FieldMatchPatterns,

		n.CloseBraceToken,
	}
}

func (n *STMappingMatchPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MappingMatchPatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STFieldMatchPatternNode struct {
	STNode

	FieldNameNode STNode

	ColonToken STNode

	MatchPattern STNode
}

var _ STNode = &STFieldMatchPatternNode{}

func (n *STFieldMatchPatternNode) Kind() common.SyntaxKind {
	return common.FIELD_MATCH_PATTERN
}

func (n *STFieldMatchPatternNode) BucketCount() int {
	return 3
}

func (n *STFieldMatchPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FieldNameNode

	case 1:
		return n.ColonToken

	case 2:
		return n.MatchPattern

	default:
		panic("invalid bucket index")
	}
}

func (n *STFieldMatchPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.FieldNameNode,

		n.ColonToken,

		n.MatchPattern,
	}
}

func (n *STFieldMatchPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &FieldMatchPatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STErrorMatchPatternNode struct {
	STNode

	ErrorKeyword STNode

	TypeReference STNode

	OpenParenthesisToken STNode

	ArgListMatchPatternNode STNode

	CloseParenthesisToken STNode
}

var _ STNode = &STErrorMatchPatternNode{}

func (n *STErrorMatchPatternNode) Kind() common.SyntaxKind {
	return common.ERROR_MATCH_PATTERN
}

func (n *STErrorMatchPatternNode) BucketCount() int {
	return 5
}

func (n *STErrorMatchPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ErrorKeyword

	case 1:
		return n.TypeReference

	case 2:
		return n.OpenParenthesisToken

	case 3:
		return n.ArgListMatchPatternNode

	case 4:
		return n.CloseParenthesisToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STErrorMatchPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.ErrorKeyword,

		n.TypeReference,

		n.OpenParenthesisToken,

		n.ArgListMatchPatternNode,

		n.CloseParenthesisToken,
	}
}

func (n *STErrorMatchPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ErrorMatchPatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNamedArgMatchPatternNode struct {
	STNode

	Identifier STNode

	EqualToken STNode

	MatchPattern STNode
}

var _ STNode = &STNamedArgMatchPatternNode{}

func (n *STNamedArgMatchPatternNode) Kind() common.SyntaxKind {
	return common.NAMED_ARG_MATCH_PATTERN
}

func (n *STNamedArgMatchPatternNode) BucketCount() int {
	return 3
}

func (n *STNamedArgMatchPatternNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Identifier

	case 1:
		return n.EqualToken

	case 2:
		return n.MatchPattern

	default:
		panic("invalid bucket index")
	}
}

func (n *STNamedArgMatchPatternNode) ChildBuckets() []STNode {
	return []STNode{

		n.Identifier,

		n.EqualToken,

		n.MatchPattern,
	}
}

func (n *STNamedArgMatchPatternNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NamedArgMatchPatternNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STDocumentationNode = STNode

type STMarkdownDocumentationNode struct {
	STDocumentationNode

	DocumentationLines STNode
}

var _ STNode = &STMarkdownDocumentationNode{}

func (n *STMarkdownDocumentationNode) Kind() common.SyntaxKind {
	return common.MARKDOWN_DOCUMENTATION
}

func (n *STMarkdownDocumentationNode) BucketCount() int {
	return 1
}

func (n *STMarkdownDocumentationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.DocumentationLines

	default:
		panic("invalid bucket index")
	}
}

func (n *STMarkdownDocumentationNode) ChildBuckets() []STNode {
	return []STNode{

		n.DocumentationLines,
	}
}

func (n *STMarkdownDocumentationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MarkdownDocumentationNode{
		DocumentationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMarkdownDocumentationLineNode struct {
	STDocumentationNode

	HashToken STNode

	DocumentElements STNode
}

var _ STNode = &STMarkdownDocumentationLineNode{}

func (n *STMarkdownDocumentationLineNode) Kind() common.SyntaxKind {
	return n.STDocumentationNode.Kind()
}

func (n *STMarkdownDocumentationLineNode) BucketCount() int {
	return 2
}

func (n *STMarkdownDocumentationLineNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.HashToken

	case 1:
		return n.DocumentElements

	default:
		panic("invalid bucket index")
	}
}

func (n *STMarkdownDocumentationLineNode) ChildBuckets() []STNode {
	return []STNode{

		n.HashToken,

		n.DocumentElements,
	}
}

func (n *STMarkdownDocumentationLineNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MarkdownDocumentationLineNode{
		DocumentationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMarkdownParameterDocumentationLineNode struct {
	STDocumentationNode

	HashToken STNode

	PlusToken STNode

	ParameterName STNode

	MinusToken STNode

	DocumentElements STNode
}

var _ STNode = &STMarkdownParameterDocumentationLineNode{}

func (n *STMarkdownParameterDocumentationLineNode) Kind() common.SyntaxKind {
	return n.STDocumentationNode.Kind()
}

func (n *STMarkdownParameterDocumentationLineNode) BucketCount() int {
	return 5
}

func (n *STMarkdownParameterDocumentationLineNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.HashToken

	case 1:
		return n.PlusToken

	case 2:
		return n.ParameterName

	case 3:
		return n.MinusToken

	case 4:
		return n.DocumentElements

	default:
		panic("invalid bucket index")
	}
}

func (n *STMarkdownParameterDocumentationLineNode) ChildBuckets() []STNode {
	return []STNode{

		n.HashToken,

		n.PlusToken,

		n.ParameterName,

		n.MinusToken,

		n.DocumentElements,
	}
}

func (n *STMarkdownParameterDocumentationLineNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MarkdownParameterDocumentationLineNode{
		DocumentationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STBallerinaNameReferenceNode struct {
	STDocumentationNode

	ReferenceType STNode

	StartBacktick STNode

	NameReference STNode

	EndBacktick STNode
}

var _ STNode = &STBallerinaNameReferenceNode{}

func (n *STBallerinaNameReferenceNode) Kind() common.SyntaxKind {
	return common.BALLERINA_NAME_REFERENCE
}

func (n *STBallerinaNameReferenceNode) BucketCount() int {
	return 4
}

func (n *STBallerinaNameReferenceNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReferenceType

	case 1:
		return n.StartBacktick

	case 2:
		return n.NameReference

	case 3:
		return n.EndBacktick

	default:
		panic("invalid bucket index")
	}
}

func (n *STBallerinaNameReferenceNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReferenceType,

		n.StartBacktick,

		n.NameReference,

		n.EndBacktick,
	}
}

func (n *STBallerinaNameReferenceNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &BallerinaNameReferenceNode{
		DocumentationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STInlineCodeReferenceNode struct {
	STDocumentationNode

	StartBacktick STNode

	CodeReference STNode

	EndBacktick STNode
}

var _ STNode = &STInlineCodeReferenceNode{}

func (n *STInlineCodeReferenceNode) Kind() common.SyntaxKind {
	return common.INLINE_CODE_REFERENCE
}

func (n *STInlineCodeReferenceNode) BucketCount() int {
	return 3
}

func (n *STInlineCodeReferenceNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.StartBacktick

	case 1:
		return n.CodeReference

	case 2:
		return n.EndBacktick

	default:
		panic("invalid bucket index")
	}
}

func (n *STInlineCodeReferenceNode) ChildBuckets() []STNode {
	return []STNode{

		n.StartBacktick,

		n.CodeReference,

		n.EndBacktick,
	}
}

func (n *STInlineCodeReferenceNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &InlineCodeReferenceNode{
		DocumentationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMarkdownCodeBlockNode struct {
	STDocumentationNode

	StartLineHashToken STNode

	StartBacktick STNode

	LangAttribute STNode

	CodeLines STNode

	EndLineHashToken STNode

	EndBacktick STNode
}

var _ STNode = &STMarkdownCodeBlockNode{}

func (n *STMarkdownCodeBlockNode) Kind() common.SyntaxKind {
	return common.MARKDOWN_CODE_BLOCK
}

func (n *STMarkdownCodeBlockNode) BucketCount() int {
	return 6
}

func (n *STMarkdownCodeBlockNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.StartLineHashToken

	case 1:
		return n.StartBacktick

	case 2:
		return n.LangAttribute

	case 3:
		return n.CodeLines

	case 4:
		return n.EndLineHashToken

	case 5:
		return n.EndBacktick

	default:
		panic("invalid bucket index")
	}
}

func (n *STMarkdownCodeBlockNode) ChildBuckets() []STNode {
	return []STNode{

		n.StartLineHashToken,

		n.StartBacktick,

		n.LangAttribute,

		n.CodeLines,

		n.EndLineHashToken,

		n.EndBacktick,
	}
}

func (n *STMarkdownCodeBlockNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MarkdownCodeBlockNode{
		DocumentationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMarkdownCodeLineNode struct {
	STDocumentationNode

	HashToken STNode

	CodeDescription STNode
}

var _ STNode = &STMarkdownCodeLineNode{}

func (n *STMarkdownCodeLineNode) Kind() common.SyntaxKind {
	return common.MARKDOWN_CODE_LINE
}

func (n *STMarkdownCodeLineNode) BucketCount() int {
	return 2
}

func (n *STMarkdownCodeLineNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.HashToken

	case 1:
		return n.CodeDescription

	default:
		panic("invalid bucket index")
	}
}

func (n *STMarkdownCodeLineNode) ChildBuckets() []STNode {
	return []STNode{

		n.HashToken,

		n.CodeDescription,
	}
}

func (n *STMarkdownCodeLineNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MarkdownCodeLineNode{
		DocumentationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STOrderByClauseNode struct {
	STIntermediateClauseNode

	OrderKeyword STNode

	ByKeyword STNode

	OrderKey STNode
}

var _ STNode = &STOrderByClauseNode{}

func (n *STOrderByClauseNode) Kind() common.SyntaxKind {
	return common.ORDER_BY_CLAUSE
}

func (n *STOrderByClauseNode) BucketCount() int {
	return 3
}

func (n *STOrderByClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OrderKeyword

	case 1:
		return n.ByKeyword

	case 2:
		return n.OrderKey

	default:
		panic("invalid bucket index")
	}
}

func (n *STOrderByClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.OrderKeyword,

		n.ByKeyword,

		n.OrderKey,
	}
}

func (n *STOrderByClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &OrderByClauseNode{
		IntermediateClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STOrderKeyNode struct {
	STNode

	Expression STNode

	OrderDirection STNode
}

var _ STNode = &STOrderKeyNode{}

func (n *STOrderKeyNode) Kind() common.SyntaxKind {
	return common.ORDER_KEY
}

func (n *STOrderKeyNode) BucketCount() int {
	return 2
}

func (n *STOrderKeyNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.OrderDirection

	default:
		panic("invalid bucket index")
	}
}

func (n *STOrderKeyNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.OrderDirection,
	}
}

func (n *STOrderKeyNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &OrderKeyNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STGroupByClauseNode struct {
	STIntermediateClauseNode

	GroupKeyword STNode

	ByKeyword STNode

	GroupingKey STNode
}

var _ STNode = &STGroupByClauseNode{}

func (n *STGroupByClauseNode) Kind() common.SyntaxKind {
	return common.GROUP_BY_CLAUSE
}

func (n *STGroupByClauseNode) BucketCount() int {
	return 3
}

func (n *STGroupByClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.GroupKeyword

	case 1:
		return n.ByKeyword

	case 2:
		return n.GroupingKey

	default:
		panic("invalid bucket index")
	}
}

func (n *STGroupByClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.GroupKeyword,

		n.ByKeyword,

		n.GroupingKey,
	}
}

func (n *STGroupByClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &GroupByClauseNode{
		IntermediateClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STGroupingKeyVarDeclarationNode struct {
	STNode

	TypeDescriptor STNode

	SimpleBindingPattern STNode

	EqualsToken STNode

	Expression STNode
}

var _ STNode = &STGroupingKeyVarDeclarationNode{}

func (n *STGroupingKeyVarDeclarationNode) Kind() common.SyntaxKind {
	return common.GROUPING_KEY_VAR_DECLARATION
}

func (n *STGroupingKeyVarDeclarationNode) BucketCount() int {
	return 4
}

func (n *STGroupingKeyVarDeclarationNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.TypeDescriptor

	case 1:
		return n.SimpleBindingPattern

	case 2:
		return n.EqualsToken

	case 3:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STGroupingKeyVarDeclarationNode) ChildBuckets() []STNode {
	return []STNode{

		n.TypeDescriptor,

		n.SimpleBindingPattern,

		n.EqualsToken,

		n.Expression,
	}
}

func (n *STGroupingKeyVarDeclarationNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &GroupingKeyVarDeclarationNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STOnFailClauseNode struct {
	STClauseNode

	OnKeyword STNode

	FailKeyword STNode

	TypedBindingPattern STNode

	BlockStatement STNode
}

var _ STNode = &STOnFailClauseNode{}

func (n *STOnFailClauseNode) Kind() common.SyntaxKind {
	return common.ON_FAIL_CLAUSE
}

func (n *STOnFailClauseNode) BucketCount() int {
	return 4
}

func (n *STOnFailClauseNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OnKeyword

	case 1:
		return n.FailKeyword

	case 2:
		return n.TypedBindingPattern

	case 3:
		return n.BlockStatement

	default:
		panic("invalid bucket index")
	}
}

func (n *STOnFailClauseNode) ChildBuckets() []STNode {
	return []STNode{

		n.OnKeyword,

		n.FailKeyword,

		n.TypedBindingPattern,

		n.BlockStatement,
	}
}

func (n *STOnFailClauseNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &OnFailClauseNode{
		ClauseNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STDoStatementNode struct {
	STStatementNode

	DoKeyword STNode

	BlockStatement STNode

	OnFailClause STNode
}

var _ STNode = &STDoStatementNode{}

func (n *STDoStatementNode) Kind() common.SyntaxKind {
	return common.DO_STATEMENT
}

func (n *STDoStatementNode) BucketCount() int {
	return 3
}

func (n *STDoStatementNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.DoKeyword

	case 1:
		return n.BlockStatement

	case 2:
		return n.OnFailClause

	default:
		panic("invalid bucket index")
	}
}

func (n *STDoStatementNode) ChildBuckets() []STNode {
	return []STNode{

		n.DoKeyword,

		n.BlockStatement,

		n.OnFailClause,
	}
}

func (n *STDoStatementNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &DoStatementNode{
		StatementNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STClassDefinitionNode struct {
	STModuleMemberDeclarationNode

	Metadata STNode

	VisibilityQualifier STNode

	ClassTypeQualifiers STNode

	ClassKeyword STNode

	ClassName STNode

	OpenBrace STNode

	Members STNode

	CloseBrace STNode

	SemicolonToken STNode
}

var _ STNode = &STClassDefinitionNode{}

func (n *STClassDefinitionNode) Kind() common.SyntaxKind {
	return common.CLASS_DEFINITION
}

func (n *STClassDefinitionNode) BucketCount() int {
	return 9
}

func (n *STClassDefinitionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Metadata

	case 1:
		return n.VisibilityQualifier

	case 2:
		return n.ClassTypeQualifiers

	case 3:
		return n.ClassKeyword

	case 4:
		return n.ClassName

	case 5:
		return n.OpenBrace

	case 6:
		return n.Members

	case 7:
		return n.CloseBrace

	case 8:
		return n.SemicolonToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STClassDefinitionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Metadata,

		n.VisibilityQualifier,

		n.ClassTypeQualifiers,

		n.ClassKeyword,

		n.ClassName,

		n.OpenBrace,

		n.Members,

		n.CloseBrace,

		n.SemicolonToken,
	}
}

func (n *STClassDefinitionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ClassDefinitionNode{
		ModuleMemberDeclarationNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STResourcePathParameterNode struct {
	STNode

	OpenBracketToken STNode

	Annotations STNode

	TypeDescriptor STNode

	EllipsisToken STNode

	ParamName STNode

	CloseBracketToken STNode
}

var _ STNode = &STResourcePathParameterNode{}

func (n *STResourcePathParameterNode) Kind() common.SyntaxKind {
	return n.STNode.Kind()
}

func (n *STResourcePathParameterNode) BucketCount() int {
	return 6
}

func (n *STResourcePathParameterNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracketToken

	case 1:
		return n.Annotations

	case 2:
		return n.TypeDescriptor

	case 3:
		return n.EllipsisToken

	case 4:
		return n.ParamName

	case 5:
		return n.CloseBracketToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STResourcePathParameterNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracketToken,

		n.Annotations,

		n.TypeDescriptor,

		n.EllipsisToken,

		n.ParamName,

		n.CloseBracketToken,
	}
}

func (n *STResourcePathParameterNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ResourcePathParameterNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STRequiredExpressionNode struct {
	STExpressionNode

	QuestionMarkToken STNode
}

var _ STNode = &STRequiredExpressionNode{}

func (n *STRequiredExpressionNode) Kind() common.SyntaxKind {
	return common.REQUIRED_EXPRESSION
}

func (n *STRequiredExpressionNode) BucketCount() int {
	return 1
}

func (n *STRequiredExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.QuestionMarkToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STRequiredExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.QuestionMarkToken,
	}
}

func (n *STRequiredExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &RequiredExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STErrorConstructorExpressionNode struct {
	STExpressionNode

	ErrorKeyword STNode

	TypeReference STNode

	OpenParenToken STNode

	Arguments STNode

	CloseParenToken STNode
}

var _ STNode = &STErrorConstructorExpressionNode{}

func (n *STErrorConstructorExpressionNode) Kind() common.SyntaxKind {
	return common.ERROR_CONSTRUCTOR
}

func (n *STErrorConstructorExpressionNode) BucketCount() int {
	return 5
}

func (n *STErrorConstructorExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ErrorKeyword

	case 1:
		return n.TypeReference

	case 2:
		return n.OpenParenToken

	case 3:
		return n.Arguments

	case 4:
		return n.CloseParenToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STErrorConstructorExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.ErrorKeyword,

		n.TypeReference,

		n.OpenParenToken,

		n.Arguments,

		n.CloseParenToken,
	}
}

func (n *STErrorConstructorExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ErrorConstructorExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STParameterizedTypeDescriptorNode struct {
	STTypeDescriptorNode

	KeywordToken STNode

	TypeParamNode STNode
}

var _ STNode = &STParameterizedTypeDescriptorNode{}

func (n *STParameterizedTypeDescriptorNode) Kind() common.SyntaxKind {
	return n.STTypeDescriptorNode.Kind()
}

func (n *STParameterizedTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STParameterizedTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.KeywordToken

	case 1:
		return n.TypeParamNode

	default:
		panic("invalid bucket index")
	}
}

func (n *STParameterizedTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.KeywordToken,

		n.TypeParamNode,
	}
}

func (n *STParameterizedTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ParameterizedTypeDescriptorNode{
		TypeDescriptorNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STSpreadMemberNode struct {
	STNode

	Ellipsis STNode

	Expression STNode
}

var _ STNode = &STSpreadMemberNode{}

func (n *STSpreadMemberNode) Kind() common.SyntaxKind {
	return common.SPREAD_MEMBER
}

func (n *STSpreadMemberNode) BucketCount() int {
	return 2
}

func (n *STSpreadMemberNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Ellipsis

	case 1:
		return n.Expression

	default:
		panic("invalid bucket index")
	}
}

func (n *STSpreadMemberNode) ChildBuckets() []STNode {
	return []STNode{

		n.Ellipsis,

		n.Expression,
	}
}

func (n *STSpreadMemberNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &SpreadMemberNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STClientResourceAccessActionNode struct {
	STActionNode

	Expression STNode

	RightArrowToken STNode

	SlashToken STNode

	ResourceAccessPath STNode

	DotToken STNode

	MethodName STNode

	Arguments STNode
}

var _ STNode = &STClientResourceAccessActionNode{}

func (n *STClientResourceAccessActionNode) Kind() common.SyntaxKind {
	return common.CLIENT_RESOURCE_ACCESS_ACTION
}

func (n *STClientResourceAccessActionNode) BucketCount() int {
	return 7
}

func (n *STClientResourceAccessActionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Expression

	case 1:
		return n.RightArrowToken

	case 2:
		return n.SlashToken

	case 3:
		return n.ResourceAccessPath

	case 4:
		return n.DotToken

	case 5:
		return n.MethodName

	case 6:
		return n.Arguments

	default:
		panic("invalid bucket index")
	}
}

func (n *STClientResourceAccessActionNode) ChildBuckets() []STNode {
	return []STNode{

		n.Expression,

		n.RightArrowToken,

		n.SlashToken,

		n.ResourceAccessPath,

		n.DotToken,

		n.MethodName,

		n.Arguments,
	}
}

func (n *STClientResourceAccessActionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ClientResourceAccessActionNode{
		ActionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STComputedResourceAccessSegmentNode struct {
	STNode

	OpenBracketToken STNode

	Expression STNode

	CloseBracketToken STNode
}

var _ STNode = &STComputedResourceAccessSegmentNode{}

func (n *STComputedResourceAccessSegmentNode) Kind() common.SyntaxKind {
	return common.COMPUTED_RESOURCE_ACCESS_SEGMENT
}

func (n *STComputedResourceAccessSegmentNode) BucketCount() int {
	return 3
}

func (n *STComputedResourceAccessSegmentNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracketToken

	case 1:
		return n.Expression

	case 2:
		return n.CloseBracketToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STComputedResourceAccessSegmentNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracketToken,

		n.Expression,

		n.CloseBracketToken,
	}
}

func (n *STComputedResourceAccessSegmentNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ComputedResourceAccessSegmentNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STResourceAccessRestSegmentNode struct {
	STNode

	OpenBracketToken STNode

	EllipsisToken STNode

	Expression STNode

	CloseBracketToken STNode
}

var _ STNode = &STResourceAccessRestSegmentNode{}

func (n *STResourceAccessRestSegmentNode) Kind() common.SyntaxKind {
	return common.RESOURCE_ACCESS_REST_SEGMENT
}

func (n *STResourceAccessRestSegmentNode) BucketCount() int {
	return 4
}

func (n *STResourceAccessRestSegmentNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracketToken

	case 1:
		return n.EllipsisToken

	case 2:
		return n.Expression

	case 3:
		return n.CloseBracketToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STResourceAccessRestSegmentNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracketToken,

		n.EllipsisToken,

		n.Expression,

		n.CloseBracketToken,
	}
}

func (n *STResourceAccessRestSegmentNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ResourceAccessRestSegmentNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReSequenceNode struct {
	STNode

	ReTerm STNode
}

var _ STNode = &STReSequenceNode{}

func (n *STReSequenceNode) Kind() common.SyntaxKind {
	return common.RE_SEQUENCE
}

func (n *STReSequenceNode) BucketCount() int {
	return 1
}

func (n *STReSequenceNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReTerm

	default:
		panic("invalid bucket index")
	}
}

func (n *STReSequenceNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReTerm,
	}
}

func (n *STReSequenceNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReSequenceNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReTermNode = STNode

type STReAtomQuantifierNode struct {
	STReTermNode

	ReAtom STNode

	ReQuantifier STNode
}

var _ STNode = &STReAtomQuantifierNode{}

func (n *STReAtomQuantifierNode) Kind() common.SyntaxKind {
	return common.RE_ATOM_QUANTIFIER
}

func (n *STReAtomQuantifierNode) BucketCount() int {
	return 2
}

func (n *STReAtomQuantifierNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReAtom

	case 1:
		return n.ReQuantifier

	default:
		panic("invalid bucket index")
	}
}

func (n *STReAtomQuantifierNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReAtom,

		n.ReQuantifier,
	}
}

func (n *STReAtomQuantifierNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReAtomQuantifierNode{
		ReTermNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReAtomCharOrEscapeNode struct {
	STNode

	ReAtomCharOrEscape STNode
}

var _ STNode = &STReAtomCharOrEscapeNode{}

func (n *STReAtomCharOrEscapeNode) Kind() common.SyntaxKind {
	return common.RE_LITERAL_CHAR_DOT_OR_ESCAPE
}

func (n *STReAtomCharOrEscapeNode) BucketCount() int {
	return 1
}

func (n *STReAtomCharOrEscapeNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReAtomCharOrEscape

	default:
		panic("invalid bucket index")
	}
}

func (n *STReAtomCharOrEscapeNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReAtomCharOrEscape,
	}
}

func (n *STReAtomCharOrEscapeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReAtomCharOrEscapeNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReQuoteEscapeNode struct {
	STNode

	SlashToken STNode

	ReSyntaxChar STNode
}

var _ STNode = &STReQuoteEscapeNode{}

func (n *STReQuoteEscapeNode) Kind() common.SyntaxKind {
	return common.RE_QUOTE_ESCAPE
}

func (n *STReQuoteEscapeNode) BucketCount() int {
	return 2
}

func (n *STReQuoteEscapeNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.SlashToken

	case 1:
		return n.ReSyntaxChar

	default:
		panic("invalid bucket index")
	}
}

func (n *STReQuoteEscapeNode) ChildBuckets() []STNode {
	return []STNode{

		n.SlashToken,

		n.ReSyntaxChar,
	}
}

func (n *STReQuoteEscapeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReQuoteEscapeNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReSimpleCharClassEscapeNode struct {
	STNode

	SlashToken STNode

	ReSimpleCharClassCode STNode
}

var _ STNode = &STReSimpleCharClassEscapeNode{}

func (n *STReSimpleCharClassEscapeNode) Kind() common.SyntaxKind {
	return common.RE_SIMPLE_CHAR_CLASS_ESCAPE
}

func (n *STReSimpleCharClassEscapeNode) BucketCount() int {
	return 2
}

func (n *STReSimpleCharClassEscapeNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.SlashToken

	case 1:
		return n.ReSimpleCharClassCode

	default:
		panic("invalid bucket index")
	}
}

func (n *STReSimpleCharClassEscapeNode) ChildBuckets() []STNode {
	return []STNode{

		n.SlashToken,

		n.ReSimpleCharClassCode,
	}
}

func (n *STReSimpleCharClassEscapeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReSimpleCharClassEscapeNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReUnicodePropertyEscapeNode struct {
	STNode

	SlashToken STNode

	Property STNode

	OpenBraceToken STNode

	ReUnicodeProperty STNode

	CloseBraceToken STNode
}

var _ STNode = &STReUnicodePropertyEscapeNode{}

func (n *STReUnicodePropertyEscapeNode) Kind() common.SyntaxKind {
	return common.RE_UNICODE_PROPERTY_ESCAPE
}

func (n *STReUnicodePropertyEscapeNode) BucketCount() int {
	return 5
}

func (n *STReUnicodePropertyEscapeNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.SlashToken

	case 1:
		return n.Property

	case 2:
		return n.OpenBraceToken

	case 3:
		return n.ReUnicodeProperty

	case 4:
		return n.CloseBraceToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STReUnicodePropertyEscapeNode) ChildBuckets() []STNode {
	return []STNode{

		n.SlashToken,

		n.Property,

		n.OpenBraceToken,

		n.ReUnicodeProperty,

		n.CloseBraceToken,
	}
}

func (n *STReUnicodePropertyEscapeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReUnicodePropertyEscapeNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReUnicodePropertyNode = STNode

type STReUnicodeScriptNode struct {
	STReUnicodePropertyNode

	ScriptStart STNode

	ReUnicodePropertyValue STNode
}

var _ STNode = &STReUnicodeScriptNode{}

func (n *STReUnicodeScriptNode) Kind() common.SyntaxKind {
	return common.RE_UNICODE_SCRIPT
}

func (n *STReUnicodeScriptNode) BucketCount() int {
	return 2
}

func (n *STReUnicodeScriptNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ScriptStart

	case 1:
		return n.ReUnicodePropertyValue

	default:
		panic("invalid bucket index")
	}
}

func (n *STReUnicodeScriptNode) ChildBuckets() []STNode {
	return []STNode{

		n.ScriptStart,

		n.ReUnicodePropertyValue,
	}
}

func (n *STReUnicodeScriptNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReUnicodeScriptNode{
		ReUnicodePropertyNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReUnicodeGeneralCategoryNode struct {
	STReUnicodePropertyNode

	CategoryStart STNode

	ReUnicodeGeneralCategoryName STNode
}

var _ STNode = &STReUnicodeGeneralCategoryNode{}

func (n *STReUnicodeGeneralCategoryNode) Kind() common.SyntaxKind {
	return common.RE_UNICODE_GENERAL_CATEGORY
}

func (n *STReUnicodeGeneralCategoryNode) BucketCount() int {
	return 2
}

func (n *STReUnicodeGeneralCategoryNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.CategoryStart

	case 1:
		return n.ReUnicodeGeneralCategoryName

	default:
		panic("invalid bucket index")
	}
}

func (n *STReUnicodeGeneralCategoryNode) ChildBuckets() []STNode {
	return []STNode{

		n.CategoryStart,

		n.ReUnicodeGeneralCategoryName,
	}
}

func (n *STReUnicodeGeneralCategoryNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReUnicodeGeneralCategoryNode{
		ReUnicodePropertyNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCharacterClassNode struct {
	STNode

	OpenBracket STNode

	Negation STNode

	ReCharSet STNode

	CloseBracket STNode
}

var _ STNode = &STReCharacterClassNode{}

func (n *STReCharacterClassNode) Kind() common.SyntaxKind {
	return common.RE_CHARACTER_CLASS
}

func (n *STReCharacterClassNode) BucketCount() int {
	return 4
}

func (n *STReCharacterClassNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBracket

	case 1:
		return n.Negation

	case 2:
		return n.ReCharSet

	case 3:
		return n.CloseBracket

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCharacterClassNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBracket,

		n.Negation,

		n.ReCharSet,

		n.CloseBracket,
	}
}

func (n *STReCharacterClassNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCharacterClassNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCharSetRangeWithReCharSetNode struct {
	STNode

	ReCharSetRange STNode

	ReCharSet STNode
}

var _ STNode = &STReCharSetRangeWithReCharSetNode{}

func (n *STReCharSetRangeWithReCharSetNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE_WITH_RE_CHAR_SET
}

func (n *STReCharSetRangeWithReCharSetNode) BucketCount() int {
	return 2
}

func (n *STReCharSetRangeWithReCharSetNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReCharSetRange

	case 1:
		return n.ReCharSet

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCharSetRangeWithReCharSetNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReCharSetRange,

		n.ReCharSet,
	}
}

func (n *STReCharSetRangeWithReCharSetNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCharSetRangeWithReCharSetNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCharSetRangeNode struct {
	STNode

	LhsReCharSetAtom STNode

	MinusToken STNode

	RhsReCharSetAtom STNode
}

var _ STNode = &STReCharSetRangeNode{}

func (n *STReCharSetRangeNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE
}

func (n *STReCharSetRangeNode) BucketCount() int {
	return 3
}

func (n *STReCharSetRangeNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LhsReCharSetAtom

	case 1:
		return n.MinusToken

	case 2:
		return n.RhsReCharSetAtom

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCharSetRangeNode) ChildBuckets() []STNode {
	return []STNode{

		n.LhsReCharSetAtom,

		n.MinusToken,

		n.RhsReCharSetAtom,
	}
}

func (n *STReCharSetRangeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCharSetRangeNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCharSetAtomWithReCharSetNoDashNode struct {
	STNode

	ReCharSetAtom STNode

	ReCharSetNoDash STNode
}

var _ STNode = &STReCharSetAtomWithReCharSetNoDashNode{}

func (n *STReCharSetAtomWithReCharSetNoDashNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_ATOM_WITH_RE_CHAR_SET_NO_DASH
}

func (n *STReCharSetAtomWithReCharSetNoDashNode) BucketCount() int {
	return 2
}

func (n *STReCharSetAtomWithReCharSetNoDashNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReCharSetAtom

	case 1:
		return n.ReCharSetNoDash

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCharSetAtomWithReCharSetNoDashNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReCharSetAtom,

		n.ReCharSetNoDash,
	}
}

func (n *STReCharSetAtomWithReCharSetNoDashNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCharSetAtomWithReCharSetNoDashNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCharSetRangeNoDashWithReCharSetNode struct {
	STNode

	ReCharSetRangeNoDash STNode

	ReCharSet STNode
}

var _ STNode = &STReCharSetRangeNoDashWithReCharSetNode{}

func (n *STReCharSetRangeNoDashWithReCharSetNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE_NO_DASH_WITH_RE_CHAR_SET
}

func (n *STReCharSetRangeNoDashWithReCharSetNode) BucketCount() int {
	return 2
}

func (n *STReCharSetRangeNoDashWithReCharSetNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReCharSetRangeNoDash

	case 1:
		return n.ReCharSet

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCharSetRangeNoDashWithReCharSetNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReCharSetRangeNoDash,

		n.ReCharSet,
	}
}

func (n *STReCharSetRangeNoDashWithReCharSetNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCharSetRangeNoDashWithReCharSetNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCharSetRangeNoDashNode struct {
	STNode

	ReCharSetAtomNoDash STNode

	MinusToken STNode

	ReCharSetAtom STNode
}

var _ STNode = &STReCharSetRangeNoDashNode{}

func (n *STReCharSetRangeNoDashNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_RANGE_NO_DASH
}

func (n *STReCharSetRangeNoDashNode) BucketCount() int {
	return 3
}

func (n *STReCharSetRangeNoDashNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReCharSetAtomNoDash

	case 1:
		return n.MinusToken

	case 2:
		return n.ReCharSetAtom

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCharSetRangeNoDashNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReCharSetAtomNoDash,

		n.MinusToken,

		n.ReCharSetAtom,
	}
}

func (n *STReCharSetRangeNoDashNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCharSetRangeNoDashNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCharSetAtomNoDashWithReCharSetNoDashNode struct {
	STNode

	ReCharSetAtomNoDash STNode

	ReCharSetNoDash STNode
}

var _ STNode = &STReCharSetAtomNoDashWithReCharSetNoDashNode{}

func (n *STReCharSetAtomNoDashWithReCharSetNoDashNode) Kind() common.SyntaxKind {
	return common.RE_CHAR_SET_ATOM_NO_DASH_WITH_RE_CHAR_SET_NO_DASH
}

func (n *STReCharSetAtomNoDashWithReCharSetNoDashNode) BucketCount() int {
	return 2
}

func (n *STReCharSetAtomNoDashWithReCharSetNoDashNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReCharSetAtomNoDash

	case 1:
		return n.ReCharSetNoDash

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCharSetAtomNoDashWithReCharSetNoDashNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReCharSetAtomNoDash,

		n.ReCharSetNoDash,
	}
}

func (n *STReCharSetAtomNoDashWithReCharSetNoDashNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCharSetAtomNoDashWithReCharSetNoDashNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReCapturingGroupsNode struct {
	STNode

	OpenParenthesis STNode

	ReFlagExpression STNode

	ReSequences STNode

	CloseParenthesis STNode
}

var _ STNode = &STReCapturingGroupsNode{}

func (n *STReCapturingGroupsNode) Kind() common.SyntaxKind {
	return common.RE_CAPTURING_GROUP
}

func (n *STReCapturingGroupsNode) BucketCount() int {
	return 4
}

func (n *STReCapturingGroupsNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenParenthesis

	case 1:
		return n.ReFlagExpression

	case 2:
		return n.ReSequences

	case 3:
		return n.CloseParenthesis

	default:
		panic("invalid bucket index")
	}
}

func (n *STReCapturingGroupsNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenParenthesis,

		n.ReFlagExpression,

		n.ReSequences,

		n.CloseParenthesis,
	}
}

func (n *STReCapturingGroupsNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReCapturingGroupsNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReFlagExpressionNode struct {
	STNode

	QuestionMark STNode

	ReFlagsOnOff STNode

	Colon STNode
}

var _ STNode = &STReFlagExpressionNode{}

func (n *STReFlagExpressionNode) Kind() common.SyntaxKind {
	return common.RE_FLAG_EXPR
}

func (n *STReFlagExpressionNode) BucketCount() int {
	return 3
}

func (n *STReFlagExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.QuestionMark

	case 1:
		return n.ReFlagsOnOff

	case 2:
		return n.Colon

	default:
		panic("invalid bucket index")
	}
}

func (n *STReFlagExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.QuestionMark,

		n.ReFlagsOnOff,

		n.Colon,
	}
}

func (n *STReFlagExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReFlagExpressionNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReFlagsOnOffNode struct {
	STNode

	LhsReFlags STNode

	MinusToken STNode

	RhsReFlags STNode
}

var _ STNode = &STReFlagsOnOffNode{}

func (n *STReFlagsOnOffNode) Kind() common.SyntaxKind {
	return common.RE_FLAGS_ON_OFF
}

func (n *STReFlagsOnOffNode) BucketCount() int {
	return 3
}

func (n *STReFlagsOnOffNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.LhsReFlags

	case 1:
		return n.MinusToken

	case 2:
		return n.RhsReFlags

	default:
		panic("invalid bucket index")
	}
}

func (n *STReFlagsOnOffNode) ChildBuckets() []STNode {
	return []STNode{

		n.LhsReFlags,

		n.MinusToken,

		n.RhsReFlags,
	}
}

func (n *STReFlagsOnOffNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReFlagsOnOffNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReFlagsNode struct {
	STNode

	ReFlag STNode
}

var _ STNode = &STReFlagsNode{}

func (n *STReFlagsNode) Kind() common.SyntaxKind {
	return common.RE_FLAGS
}

func (n *STReFlagsNode) BucketCount() int {
	return 1
}

func (n *STReFlagsNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReFlag

	default:
		panic("invalid bucket index")
	}
}

func (n *STReFlagsNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReFlag,
	}
}

func (n *STReFlagsNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReFlagsNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReAssertionNode struct {
	STReTermNode

	ReAssertion STNode
}

var _ STNode = &STReAssertionNode{}

func (n *STReAssertionNode) Kind() common.SyntaxKind {
	return common.RE_ASSERTION
}

func (n *STReAssertionNode) BucketCount() int {
	return 1
}

func (n *STReAssertionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReAssertion

	default:
		panic("invalid bucket index")
	}
}

func (n *STReAssertionNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReAssertion,
	}
}

func (n *STReAssertionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReAssertionNode{
		ReTermNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReQuantifierNode struct {
	STNode

	ReBaseQuantifier STNode

	NonGreedyChar STNode
}

var _ STNode = &STReQuantifierNode{}

func (n *STReQuantifierNode) Kind() common.SyntaxKind {
	return common.RE_QUANTIFIER
}

func (n *STReQuantifierNode) BucketCount() int {
	return 2
}

func (n *STReQuantifierNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ReBaseQuantifier

	case 1:
		return n.NonGreedyChar

	default:
		panic("invalid bucket index")
	}
}

func (n *STReQuantifierNode) ChildBuckets() []STNode {
	return []STNode{

		n.ReBaseQuantifier,

		n.NonGreedyChar,
	}
}

func (n *STReQuantifierNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReQuantifierNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReBracedQuantifierNode struct {
	STNode

	OpenBraceToken STNode

	LeastTimesMatchedDigit STNode

	CommaToken STNode

	MostTimesMatchedDigit STNode

	CloseBraceToken STNode
}

var _ STNode = &STReBracedQuantifierNode{}

func (n *STReBracedQuantifierNode) Kind() common.SyntaxKind {
	return common.RE_BRACED_QUANTIFIER
}

func (n *STReBracedQuantifierNode) BucketCount() int {
	return 5
}

func (n *STReBracedQuantifierNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.OpenBraceToken

	case 1:
		return n.LeastTimesMatchedDigit

	case 2:
		return n.CommaToken

	case 3:
		return n.MostTimesMatchedDigit

	case 4:
		return n.CloseBraceToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STReBracedQuantifierNode) ChildBuckets() []STNode {
	return []STNode{

		n.OpenBraceToken,

		n.LeastTimesMatchedDigit,

		n.CommaToken,

		n.MostTimesMatchedDigit,

		n.CloseBraceToken,
	}
}

func (n *STReBracedQuantifierNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReBracedQuantifierNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STMemberTypeDescriptorNode struct {
	STNode

	Annotations STNode

	TypeDescriptor STNode
}

var _ STNode = &STMemberTypeDescriptorNode{}

func (n *STMemberTypeDescriptorNode) Kind() common.SyntaxKind {
	return common.MEMBER_TYPE_DESC
}

func (n *STMemberTypeDescriptorNode) BucketCount() int {
	return 2
}

func (n *STMemberTypeDescriptorNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.Annotations

	case 1:
		return n.TypeDescriptor

	default:
		panic("invalid bucket index")
	}
}

func (n *STMemberTypeDescriptorNode) ChildBuckets() []STNode {
	return []STNode{

		n.Annotations,

		n.TypeDescriptor,
	}
}

func (n *STMemberTypeDescriptorNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &MemberTypeDescriptorNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STReceiveFieldNode struct {
	STNode

	FieldName STNode

	Colon STNode

	PeerWorker STNode
}

var _ STNode = &STReceiveFieldNode{}

func (n *STReceiveFieldNode) Kind() common.SyntaxKind {
	return common.RECEIVE_FIELD
}

func (n *STReceiveFieldNode) BucketCount() int {
	return 3
}

func (n *STReceiveFieldNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.FieldName

	case 1:
		return n.Colon

	case 2:
		return n.PeerWorker

	default:
		panic("invalid bucket index")
	}
}

func (n *STReceiveFieldNode) ChildBuckets() []STNode {
	return []STNode{

		n.FieldName,

		n.Colon,

		n.PeerWorker,
	}
}

func (n *STReceiveFieldNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ReceiveFieldNode{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}

type STNaturalExpressionNode struct {
	STExpressionNode

	ConstKeyword STNode

	NaturalKeyword STNode

	ParenthesizedArgList STNode

	OpenBraceToken STNode

	Prompt STNode

	CloseBraceToken STNode
}

var _ STNode = &STNaturalExpressionNode{}

func (n *STNaturalExpressionNode) Kind() common.SyntaxKind {
	return common.NATURAL_EXPRESSION
}

func (n *STNaturalExpressionNode) BucketCount() int {
	return 6
}

func (n *STNaturalExpressionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.ConstKeyword

	case 1:
		return n.NaturalKeyword

	case 2:
		return n.ParenthesizedArgList

	case 3:
		return n.OpenBraceToken

	case 4:
		return n.Prompt

	case 5:
		return n.CloseBraceToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STNaturalExpressionNode) ChildBuckets() []STNode {
	return []STNode{

		n.ConstKeyword,

		n.NaturalKeyword,

		n.ParenthesizedArgList,

		n.OpenBraceToken,

		n.Prompt,

		n.CloseBraceToken,
	}
}

func (n *STNaturalExpressionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	return &NaturalExpressionNode{
		ExpressionNode: &NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, n.BucketCount()),
		},
	}
}
