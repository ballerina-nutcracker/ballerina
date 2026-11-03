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

package analysis

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	semcommon "github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// validateQueryExpr validates a query's clauses in order; their shape was
// validated by type resolution. Clause variables are locals of the enclosing
// closure, defined where their clause binds them.
func validateQueryExpr(cx scope.Context, e *ast.BLangQueryExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if len(e.QueryClauseList) < 2 {
		cx.InternalErr("query expression shape should have been validated during type resolution", e.GetPosition())
		return scope.NoFacts, false
	}
	for _, clause := range e.QueryClauseList {
		if !dispatchQueryClause(cx, e, clause, expected) {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func dispatchQueryClause(cx scope.Context, query *ast.BLangQueryExpr, clause ast.BLangNode, expected semtypes.SemType) bool {
	switch clause := clause.(type) {
	case *ast.BLangFromClause:
		return validateFromClause(cx, clause)
	case *ast.BLangJoinClause:
		return validateJoinClause(cx, clause)
	case *ast.BLangLetClause:
		return validateLetClause(cx, clause)
	case *ast.BLangWhereClause:
		return validateWhereClause(cx, clause)
	case *ast.BLangLimitClause:
		return validateLimitClause(cx, clause)
	case *ast.BLangGroupByClause:
		return validateGroupByClause(cx, clause)
	case *ast.BLangOrderByClause:
		return validateOrderByClause(cx, clause)
	case *ast.BLangSelectClause:
		return validateSelectClause(cx, query, clause, expected)
	case *ast.BLangCollectClause:
		return validateCollectClause(cx, query, clause)
	case *ast.BLangOnConflictClause:
		return validateOnConflictClause(cx, query, clause)
	default:
		cx.InternalErr("query expression shape should have been validated during type resolution", clause.GetPosition())
		return false
	}
}

func validateFromClause(cx scope.Context, clause *ast.BLangFromClause) bool {
	if _, ok := validateActionOrExpr(cx, clause.Collection, semtypes.SemType{}); !ok {
		return false
	}
	return validateVariableDef(cx, scope.DefineQueryLocal(cx, clause.VariableDefinitionNode))
}

func validateJoinClause(cx scope.Context, clause *ast.BLangJoinClause) bool {
	if _, ok := validateActionOrExpr(cx, clause.Collection, semtypes.SemType{}); !ok {
		return false
	}
	if !validateVariableDef(cx, scope.DefineQueryLocal(cx, clause.VariableDefinitionNode)) {
		return false
	}
	if clause.OnClause.OnExpr == nil || clause.OnClause.EqualsExpr == nil {
		cx.InternalErr("join clause shape should have been validated during type resolution", clause.GetPosition())
		return false
	}
	if _, ok := validateActionOrExpr(cx, clause.OnClause.OnExpr, semtypes.SemType{}); !ok {
		return false
	}
	_, ok := validateActionOrExpr(cx, clause.OnClause.EqualsExpr, semtypes.SemType{})
	return ok
}

func validateLetClause(cx scope.Context, clause *ast.BLangLetClause) bool {
	for i := range clause.LetVarDeclarations {
		def := &clause.LetVarDeclarations[i]
		if def.Var == nil || def.Var.Expr == nil {
			cx.SemanticErr("let clause supports only initialized simple variable declarations", clause.GetPosition())
			return false
		}
		if !validateVariableDef(cx, scope.DefineQueryLocal(cx, def)) {
			return false
		}
	}
	return true
}

func validateWhereClause(cx scope.Context, clause *ast.BLangWhereClause) bool {
	_, ok := validateActionOrExpr(cx, clause.Expression, semtypes.Boolean)
	return ok
}

func validateLimitClause(cx scope.Context, clause *ast.BLangLimitClause) bool {
	_, ok := validateActionOrExpr(cx, clause.Expression, semtypes.Int)
	return ok
}

func validateGroupByClause(cx scope.Context, clause *ast.BLangGroupByClause) bool {
	for _, key := range clause.GroupingKeyList {
		if !validateGroupingKey(cx, clause, key) {
			return false
		}
	}
	return true
}

func validateGroupingKey(cx scope.Context, clause *ast.BLangGroupByClause, key *ast.BLangGroupingKey) bool {
	anydata := semtypes.CreateAnydata(cx.TyCtx())
	switch {
	case key.VariableRef != nil:
		_, ok := validateActionOrExpr(cx, key.VariableRef, anydata)
		return ok
	case key.VariableDef != nil:
		def := key.VariableDef
		if def.Var == nil || def.Var.Expr == nil {
			cx.SemanticErr("group by clause supports only initialized simple variable declarations", clause.GetPosition())
			return false
		}
		if !validateVariableDef(cx, scope.DefineQueryLocal(cx, def)) {
			return false
		}
		if !semtypes.IsSubtype(cx.TyCtx(), cx.CompilerCtx().SymbolType(def.Var.Symbol()), anydata) {
			cx.SemanticErr("grouping key expression must be a subtype of anydata", key.GetPosition())
			return false
		}
		return true
	default:
		cx.InternalErr("group by clause shape should have been validated during type resolution", key.GetPosition())
		return false
	}
}

func validateOrderByClause(cx scope.Context, clause *ast.BLangOrderByClause) bool {
	ordered := semtypes.CreateOrdered(cx.TyCtx())
	for i := range clause.OrderByKeyList {
		if _, ok := validateActionOrExpr(cx, clause.OrderByKeyList[i].Expression, ordered); !ok {
			return false
		}
	}
	return true
}

func validateSelectClause(cx scope.Context, query *ast.BLangQueryExpr, clause *ast.BLangSelectClause, expected semtypes.SemType) bool {
	tyCtx := cx.TyCtx()
	selectExpected := semcommon.QuerySelectExpectedType(tyCtx, tyCtx.Env(), query.QueryConstructType, expected)
	if semtypes.IsZero(selectExpected) && query.QueryConstructType == ast.TypeKindMap {
		selectExpected = semcommon.MapQuerySelectExpectedType(tyCtx.Env())
	}
	_, ok := validateActionOrExpr(cx, clause.Expression, selectExpected)
	return ok
}

func validateCollectClause(cx scope.Context, query *ast.BLangQueryExpr, clause *ast.BLangCollectClause) bool {
	if _, ok := validateActionOrExpr(cx, clause.Expression, semtypes.SemType{}); !ok {
		return false
	}
	if query.QueryConstructType != ast.TypeKindNone {
		cx.SemanticErr("query construct types cannot be used with collect clause", clause.GetPosition())
		return false
	}
	return true
}

func validateOnConflictClause(cx scope.Context, query *ast.BLangQueryExpr, clause *ast.BLangOnConflictClause) bool {
	if _, ok := validateActionOrExpr(cx, clause.Expression, semtypes.Union(semtypes.Error, semtypes.Nil)); !ok {
		return false
	}
	if query.QueryConstructType != ast.TypeKindMap {
		cx.SemanticErr("on conflict clause is supported only for map query construct type", clause.GetPosition())
		return false
	}
	return true
}
