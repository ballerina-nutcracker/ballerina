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

import "ballerina-lang-go/model"

type FailureBreakMode uint

const (
	FailureBreakMode_NOT_BREAKABLE FailureBreakMode = iota
	FailureBreakMode_BREAK_WITHIN_BLOCK
	FailureBreakMode_BREAK_TO_OUTER_BLOCK
)

func (*bLangStatementBase) isStatement() {}

func (*BLangXMLNS) isStatement()        {}
func (*BLangOnFailClause) isStatement() {}

type (
	bLangStatementBase struct {
		bLangNodeBase
	}
	BLangAssignment struct {
		bLangStatementBase
		VarRef LExpr
		Expr   BLangActionOrExpression
	}
	BLangBlockStmt struct {
		bLangStatementBase
		Stmts            []StatementNode
		FailureBreakMode FailureBreakMode
		IsLetExpr        bool
	}
	BLangBreak struct {
		bLangStatementBase
	}

	BLangCompoundAssignment struct {
		bLangStatementBase
		VarRef       LExpr
		Expr         BLangActionOrExpression
		OpKind       model.OperatorKind
		ModifiedExpr BLangExpression
	}
	BLangContinue struct {
		bLangStatementBase
	}
	BLangDo struct {
		bLangStatementBase
		Body         BLangBlockStmt
		OnFailClause BLangOnFailClause
	}

	BLangExpressionStmt struct {
		bLangStatementBase
		Expr BLangActionOrExpression
	}

	BLangIf struct {
		bLangStatementBase
		scope    model.Scope
		Expr     BLangExpression
		Body     BLangBlockStmt
		ElseStmt StatementNode
	}

	BLangWhile struct {
		bLangStatementBase
		scope        model.Scope
		Expr         BLangExpression
		Body         BLangBlockStmt
		OnFailClause BLangOnFailClause
	}

	BLangForeach struct {
		bLangStatementBase
		scope             model.Scope
		VariableDef       *BLangSimpleVariableDef
		Collection        BLangActionOrExpression
		Body              BLangBlockStmt
		OnFailClause      *BLangOnFailClause
		IsDeclaredWithVar bool
	}

	BLangSimpleVariableDef struct {
		bLangStatementBase
		Var *BLangSimpleVariable
	}

	BLangReturn struct {
		bLangStatementBase
		Expr BLangActionOrExpression
	}

	BLangPanic struct {
		bLangStatementBase
		Expr BLangExpression
	}

	BLangMatchStatement struct {
		bLangStatementBase
		Expr         BLangActionOrExpression
		MatchClauses []BLangMatchClause
		IsExhaustive bool
	}

	BLangLock struct {
		bLangStatementBase
		Body BLangBlockStmt
		// LockKey is the content-addressed identifier of the restricted
		// variable, set by the lock analyzer. For a module-level isolated
		// variable it has the form "org/pkg:varName"; for a non-immutable
		// field of an isolated class accessed via self it has the form
		// "org/pkg:ClassName.fieldName". Empty when the lock body has no
		// restricted variable (we treat this as a semantic error though
		// spec doesn't).
		LockKey string
		// RestrictedSymbol is the symbol of the restricted module-level
		// variable, when applicable. Zero (model.SymbolRef{}) for the
		// self-field case (the field has no directly-accessible SymbolRef
		// from the AST) or when the body has no restricted variable. Used
		// by the locality collector so the restricted module variable is
		// treated as a root of the enclosing function for isolation
		// analysis purposes.
		RestrictedSymbol model.SymbolRef
	}
)

var (
	_ AssignmentNode          = &BLangAssignment{}
	_ CompoundAssignmentNode  = &BLangCompoundAssignment{}
	_ StatementNode           = &BLangContinue{}
	_ DoNode                  = &BLangDo{}
	_ BlockStatementNode      = &BLangBlockStmt{}
	_ ExpressionStatementNode = &BLangExpressionStmt{}
	_ IfNode                  = &BLangIf{}
	_ WhileNode               = &BLangWhile{}
	_ ForeachNode             = &BLangForeach{}
	_ VariableDefinitionNode  = &BLangSimpleVariableDef{}
	_ ReturnNode              = &BLangReturn{}
	_ PanicNode               = &BLangPanic{}
)

var (
	_ NodeWithScope = &BLangIf{}
	_ NodeWithScope = &BLangWhile{}
	_ NodeWithScope = &BLangForeach{}
)

var (
	_ BLangNode = &BLangAssignment{}
	_ BLangNode = &BLangBlockStmt{}
	_ BLangNode = &BLangBreak{}
	_ BLangNode = &BLangCompoundAssignment{}
	_ BLangNode = &BLangContinue{}
	_ BLangNode = &BLangDo{}
	_ BLangNode = &BLangExpressionStmt{}
	_ BLangNode = &BLangIf{}
	_ BLangNode = &BLangWhile{}
	_ BLangNode = &BLangForeach{}
	_ BLangNode = &BLangSimpleVariableDef{}
	_ BLangNode = &BLangReturn{}
	_ BLangNode = &BLangPanic{}
	_ BLangNode = &BLangMatchStatement{}
	_ BLangNode = &BLangLock{}
)

func (b *BLangAssignment) GetVariable() LExpr {
	return b.VarRef
}

func (b *BLangAssignment) GetExpression() BLangActionOrExpression {
	return b.Expr
}

func (b *BLangAssignment) IsDeclaredWithVar() bool {
	return false
}

func (b *BLangAssignment) SetActionOrExpression(actionOrExpression BLangActionOrExpression) {
	b.Expr = actionOrExpression
}

func (b *BLangAssignment) SetDeclaredWithVar(isDeclaredWithVar bool) {
}

func (b *BLangAssignment) SetVariable(variableReferenceNode LExpr) {
	b.VarRef = variableReferenceNode
}

func (b *BLangBlockStmt) GetStatements() []StatementNode {
	return b.Stmts
}

func (b *BLangBlockStmt) AddStatement(statement StatementNode) {
	b.Stmts = append(b.Stmts, statement)
}

func (b *BLangCompoundAssignment) IsDeclaredWithVar() bool {
	return false
}

func (b *BLangCompoundAssignment) SetDeclaredWithVar(_ bool) {
	panic("compound assignemnt can't be declared with var")
}

func (b *BLangCompoundAssignment) GetOperatorKind() model.OperatorKind {
	return b.OpKind
}

func (b *BLangCompoundAssignment) GetVariable() LExpr {
	return b.VarRef
}

func (b *BLangCompoundAssignment) GetExpression() BLangActionOrExpression {
	return b.Expr
}

func (b *BLangCompoundAssignment) SetActionOrExpression(actionOrExpression BLangActionOrExpression) {
	b.Expr = actionOrExpression
}

func (b *BLangCompoundAssignment) SetVariable(variableReferenceNode LExpr) {
	b.VarRef = variableReferenceNode
}

func (b *BLangDo) GetBody() BlockStatementNode {
	return &b.Body
}

func (b *BLangDo) SetBody(body BlockStatementNode) {
	if blockStmt, ok := body.(*BLangBlockStmt); ok {
		b.Body = *blockStmt
		return
	}
	panic("body is not a BLangBlockStmt")
}

func (b *BLangDo) GetOnFailClause() OnFailClauseNode {
	return &b.OnFailClause
}

func (b *BLangDo) SetOnFailClause(onFailClause OnFailClauseNode) {
	if onFailClause, ok := onFailClause.(*BLangOnFailClause); ok {
		b.OnFailClause = *onFailClause
		return
	}
	panic("onFailClause is not a BLangOnFailClause")
}

func (b *BLangExpressionStmt) GetExpression() BLangActionOrExpression {
	return b.Expr
}

func (b *BLangIf) Scope() model.Scope {
	return b.scope
}

func (b *BLangIf) SetScope(scope model.Scope) {
	b.scope = scope
}

func (b *BLangIf) GetCondition() BLangExpression {
	return b.Expr
}

func (b *BLangIf) GetBody() BlockStatementNode {
	return &b.Body
}

func (b *BLangIf) GetElseStatement() StatementNode {
	return b.ElseStmt
}

func (b *BLangIf) SetCondition(condition BLangExpression) {
	b.Expr = condition
}

func (b *BLangIf) SetBody(body BlockStatementNode) {
	if blockStmt, ok := body.(*BLangBlockStmt); ok {
		b.Body = *blockStmt
		return
	}
	panic("body is not a BLangBlockStmt")
}

func (b *BLangIf) SetElseStatement(elseStatement StatementNode) {
	b.ElseStmt = elseStatement
}

func (b *BLangWhile) Scope() model.Scope {
	return b.scope
}

func (b *BLangWhile) SetScope(scope model.Scope) {
	b.scope = scope
}

func (b *BLangWhile) GetCondition() BLangExpression {
	return b.Expr
}

func (b *BLangWhile) SetCondition(condition BLangExpression) {
	b.Expr = condition
}

func (b *BLangWhile) GetBody() BlockStatementNode {
	return &b.Body
}

func (b *BLangWhile) SetBody(body BlockStatementNode) {
	if blockStmt, ok := body.(*BLangBlockStmt); ok {
		b.Body = *blockStmt
		return
	}
	panic("body is not a BLangBlockStmt")
}

func (b *BLangWhile) GetOnFailClause() OnFailClauseNode {
	return &b.OnFailClause
}

func (b *BLangWhile) SetOnFailClause(onFailClause OnFailClauseNode) {
	if onFailClause, ok := onFailClause.(*BLangOnFailClause); ok {
		b.OnFailClause = *onFailClause
		return
	}
	panic("onFailClause is not a BLangOnFailClause")
}

func (b *BLangForeach) Scope() model.Scope {
	return b.scope
}

func (b *BLangForeach) SetScope(scope model.Scope) {
	b.scope = scope
}

func (b *BLangForeach) GetVariableDefinitionNode() VariableDefinitionNode {
	return b.VariableDef
}

func (b *BLangForeach) SetVariableDefinitionNode(node VariableDefinitionNode) {
	if node == nil {
		b.VariableDef = nil
		return
	}
	if varDef, ok := node.(*BLangSimpleVariableDef); ok {
		b.VariableDef = varDef
		return
	}
	panic("node is not a *BLangSimpleVariableDef")
}

func (b *BLangForeach) GetCollection() BLangActionOrExpression {
	return b.Collection
}

func (b *BLangForeach) SetCollection(collection BLangActionOrExpression) {
	b.Collection = collection
}

func (b *BLangForeach) GetBody() BlockStatementNode {
	return &b.Body
}

func (b *BLangForeach) SetBody(body BlockStatementNode) {
	if blockStmt, ok := body.(*BLangBlockStmt); ok {
		b.Body = *blockStmt
		return
	}
	panic("body is not a BLangBlockStmt")
}

func (b *BLangForeach) GetIsDeclaredWithVar() bool {
	return b.IsDeclaredWithVar
}

func (b *BLangForeach) GetOnFailClause() OnFailClauseNode {
	if b.OnFailClause == nil {
		return nil
	}
	return b.OnFailClause
}

func (b *BLangForeach) SetOnFailClause(onFailClause OnFailClauseNode) {
	if onFailClause == nil {
		b.OnFailClause = nil
		return
	}
	if clause, ok := onFailClause.(*BLangOnFailClause); ok {
		b.OnFailClause = clause
		return
	}
	panic("onFailClause is not a *BLangOnFailClause")
}

func (b *BLangSimpleVariableDef) GetVariable() VariableNode {
	return b.Var
}

func (b *BLangSimpleVariableDef) SetVariable(variable VariableNode) {
	if v, ok := variable.(*BLangSimpleVariable); ok {
		b.Var = v
	} else {
		panic("variable is not a BLangSimpleVariable")
	}
}

func (b *BLangReturn) GetExpression() BLangActionOrExpression {
	return b.Expr
}

func (b *BLangReturn) SetActionOrExpression(actionOrExpression BLangActionOrExpression) {
	b.Expr = actionOrExpression
}

func (b *BLangPanic) GetExpression() BLangExpression {
	return b.Expr
}

func (b *BLangLock) GetBody() BlockStatementNode {
	return &b.Body
}
