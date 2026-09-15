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
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type FailureBreakMode uint

const (
	FailureBreakModeNotBreakable FailureBreakMode = iota
	FailureBreakModeBreakWithinBlock
	FailureBreakModeBreakToOuterBlock
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
		VarRef LExpr
		Expr   BLangActionOrExpression
		OpKind model.OperatorKind
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
		VariableDef       *BLangVariableDef
		Collection        BLangActionOrExpression
		Body              BLangBlockStmt
		OnFailClause      *BLangOnFailClause
		IsDeclaredWithVar bool
	}

	BLangVariableDef struct {
		bLangStatementBase
		Var *BLangVariable
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
		// field of an isolated class or service accessed via self it has the
		// form "org/pkg:ClassName.fieldName" or
		// "org/pkg:$service$N.fieldName". Empty when the lock body has no
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
	_ StatementNode           = &BLangContinue{}
	_ ExpressionStatementNode = &BLangExpressionStmt{}
)

var (
	_ NodeWithScope = &BLangIf{}
	_ NodeWithScope = &BLangWhile{}
	_ NodeWithScope = &BLangForeach{}
)

func NewBLangAssignment(pos Location, variable LExpr, expr BLangActionOrExpression) *BLangAssignment {
	return &BLangAssignment{
		bLangStatementBase: bLangStatementBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		VarRef:             variable,
		Expr:               expr,
	}
}

func NewBLangCompoundAssignment(pos Location, variable LExpr, expr BLangActionOrExpression, opKind model.OperatorKind) *BLangCompoundAssignment {
	return &BLangCompoundAssignment{
		bLangStatementBase: bLangStatementBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		VarRef:             variable,
		Expr:               expr,
		OpKind:             opKind,
	}
}

func NewBLangIf(pos Location, condition BLangExpression, body *BLangBlockStmt, elseStmt StatementNode) *BLangIf {
	return &BLangIf{
		bLangStatementBase: bLangStatementBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		Expr:               condition,
		Body:               *body,
		ElseStmt:           elseStmt,
	}
}

func NewBLangWhile(pos Location, condition BLangExpression, body *BLangBlockStmt, onFailClause *BLangOnFailClause) *BLangWhile {
	stmt := &BLangWhile{
		bLangStatementBase: bLangStatementBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		Expr:               condition,
		Body:               *body,
	}
	if onFailClause != nil {
		stmt.OnFailClause = *onFailClause
	} else {
		stmt.OnFailClause.pos = diagnostics.NewBuiltinLocation()
	}
	return stmt
}

func NewBLangForeach(pos Location, variableDef *BLangVariableDef, collection BLangActionOrExpression, body *BLangBlockStmt, onFailClause *BLangOnFailClause) *BLangForeach {
	return &BLangForeach{
		bLangStatementBase: bLangStatementBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		VariableDef:        variableDef,
		Collection:         collection,
		Body:               *body,
		OnFailClause:       onFailClause,
		IsDeclaredWithVar:  variableDef.Var.IsDeclaredWithVar,
	}
}

func NewBLangReturn(pos Location, expr BLangActionOrExpression) *BLangReturn {
	return &BLangReturn{
		bLangStatementBase: bLangStatementBase{bLangNodeBase: bLangNodeBase{pos: pos}},
		Expr:               expr,
	}
}

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
	_ BLangNode = &BLangVariableDef{}
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

func (b *BLangBlockStmt) GetStatements() []StatementNode {
	return b.Stmts
}

func (b *BLangBlockStmt) AddStatement(statement StatementNode) {
	b.Stmts = append(b.Stmts, statement)
}

func (b *BLangCompoundAssignment) IsDeclaredWithVar() bool {
	return false
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

func (b *BLangDo) GetBody() *BLangBlockStmt {
	return &b.Body
}

func (b *BLangDo) GetOnFailClause() *BLangOnFailClause {
	return &b.OnFailClause
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

func (b *BLangIf) GetBody() *BLangBlockStmt {
	return &b.Body
}

func (b *BLangIf) GetElseStatement() StatementNode {
	return b.ElseStmt
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

func (b *BLangWhile) GetBody() *BLangBlockStmt {
	return &b.Body
}

func (b *BLangWhile) GetOnFailClause() *BLangOnFailClause {
	return &b.OnFailClause
}

func (b *BLangForeach) Scope() model.Scope {
	return b.scope
}

func (b *BLangForeach) SetScope(scope model.Scope) {
	b.scope = scope
}

func (b *BLangForeach) GetVariableDefinitionNode() *BLangVariableDef {
	return b.VariableDef
}

func (b *BLangForeach) GetCollection() BLangActionOrExpression {
	return b.Collection
}

func (b *BLangForeach) GetBody() *BLangBlockStmt {
	return &b.Body
}

func (b *BLangForeach) GetIsDeclaredWithVar() bool {
	return b.IsDeclaredWithVar
}

func (b *BLangForeach) GetOnFailClause() *BLangOnFailClause {
	if b.OnFailClause == nil {
		return nil
	}
	return b.OnFailClause
}

func (b *BLangVariableDef) GetVariable() *BLangVariable {
	return b.Var
}

func (b *BLangVariableDef) SetVariable(variable *BLangVariable) {
	b.Var = variable
}

func (b *BLangReturn) GetExpression() BLangActionOrExpression {
	return b.Expr
}

func (b *BLangPanic) GetExpression() BLangExpression {
	return b.Expr
}

func (b *BLangLock) GetBody() *BLangBlockStmt {
	return &b.Body
}
