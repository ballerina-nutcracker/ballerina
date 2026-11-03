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
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// validateBlock validates each statement; a statement that reports is
// abandoned and the next one runs.
func validateBlock(cx scope.Context, stmts []ast.StatementNode) {
	for _, stmt := range stmts {
		validateStmt(cx, stmt)
	}
}

func validateStmt(cx scope.Context, stmt ast.StatementNode) {
	switch stmt := stmt.(type) {
	case *ast.BLangBadStmt:
		// The AST builder already reported the failed statement.
		return
	case *ast.BLangVariableDef:
		if ast.SymbolIsSet(stmt.Var) {
			validateVariableDef(cx, scope.DefineLocal(cx, stmt))
		}
	case *ast.BLangAssignment:
		validateAssignment(cx, stmt)
	case *ast.BLangCompoundAssignment:
		validateAssignment(cx, stmt)
	case *ast.BLangExpressionStmt:
		validateExpressionStmt(cx, stmt)
	case *ast.BLangIf:
		validateIf(cx, stmt)
	case *ast.BLangWhile:
		validateWhile(cx, stmt)
	case *ast.BLangForeach:
		validateForeach(cx, stmt)
	case *ast.BLangDo:
		validateDo(cx, stmt)
	case *ast.BLangMatchStatement:
		validateMatch(cx, stmt)
	case *ast.BLangReturn:
		validateReturn(cx, stmt)
	case *ast.BLangPanic:
		validateActionOrExpr(cx, stmt.Expr, semtypes.Error)
	case *ast.BLangLock:
		validateLock(cx, stmt)
	case *ast.BLangBlockStmt:
		validateBlock(cx, stmt.Stmts)
	case *ast.BLangBreak, *ast.BLangContinue:
		// Nothing in this stage depends on loops.
	case *ast.BLangXMLNS:
		// An xmlns declaration was checked by symbol resolution.
	default:
		cx.InternalErr("unexpected statement", stmt.GetPosition())
	}
}

// validateVariableDef validates every variable definition: statement-level,
// foreach, on-fail and query clause variables.
func validateVariableDef(cx scope.Context, local scope.LocalDef) bool {
	def := local.Node()
	return validateVariable(cx, def.Var, cx.CompilerCtx().SymbolType(def.Var.Symbol()))
}

func validateExpressionStmt(cx scope.Context, stmt *ast.BLangExpressionStmt) {
	if _, ok := validateActionOrExpr(cx, stmt.Expr, semtypes.SemType{}); !ok {
		return
	}
	if !semtypes.IsSubtype(cx.TyCtx(), stmt.Expr.Type(), semtypes.Nil) {
		cx.SemanticErr("expression value must be assigned", stmt.Expr.GetPosition())
	}
}

// validateIf validates the condition, the body and the else branch; an error
// in the condition does not abandon the body or the else branch.
func validateIf(cx scope.Context, stmt *ast.BLangIf) {
	validateActionOrExpr(cx, stmt.Expr, semtypes.Boolean)
	validateBlock(cx, stmt.Body.Stmts)
	if stmt.ElseStmt != nil {
		validateStmt(cx, stmt.ElseStmt)
	}
}

// validateWhile validates the condition, the body and the on-fail clause; an
// error in the condition does not abandon the body.
func validateWhile(cx scope.Context, stmt *ast.BLangWhile) {
	validateActionOrExpr(cx, stmt.Expr, semtypes.Boolean)
	validateBlock(cx, stmt.Body.Stmts)
	if stmt.OnFailClause.Body != nil {
		validateOnFail(cx, &stmt.OnFailClause)
	}
}

// validateForeach validates the collection, the loop variable, the body and
// the on-fail clause; an error in the header does not abandon the body.
func validateForeach(cx scope.Context, stmt *ast.BLangForeach) {
	if ast.SymbolIsSet(stmt.VariableDef.Var) {
		validateForeachHeader(cx, stmt, scope.DefineBoundLocal(cx, stmt.VariableDef))
	}
	validateBlock(cx, stmt.Body.Stmts)
	if stmt.OnFailClause != nil {
		validateOnFail(cx, stmt.OnFailClause)
	}
}

// validateForeachHeader validates the collection and the loop variable.
func validateForeachHeader(cx scope.Context, stmt *ast.BLangForeach, loopVar scope.LocalDef) {
	if _, ok := validateActionOrExpr(cx, stmt.Collection, semtypes.SemType{}); !ok {
		return
	}
	if !validateForeachVariableType(cx, stmt) {
		return
	}
	validateVariableDef(cx, loopVar)
}

// validateForeachVariableType checks the loop variable's type against the
// values the collection yields.
func validateForeachVariableType(cx scope.Context, stmt *ast.BLangForeach) bool {
	tyCtx := cx.TyCtx()
	collection := stmt.Collection
	variableType := cx.CompilerCtx().SymbolType(stmt.VariableDef.Var.Symbol())
	if binExpr, isBinary := collection.(*ast.BLangBinaryExpr); isBinary && common.IsRangeExpr(binExpr) {
		if !semtypes.IsSubtype(tyCtx, variableType, semtypes.Int) {
			cx.SemanticErr("foreach variable must be a subtype of int for range expression", collection.GetPosition())
			return false
		}
		return true
	}
	return validateForeachValueType(cx, stmt, variableType)
}

// validateForeachValueType checks the loop variable's type against the values
// a list, mapping, XML or Iterable collection yields.
func validateForeachValueType(cx scope.Context, stmt *ast.BLangForeach, variableType semtypes.SemType) bool {
	tyCtx := cx.TyCtx()
	collection := stmt.Collection
	collectionType := collection.Type()
	var valueType semtypes.SemType
	switch {
	case semtypes.IsSubtype(tyCtx, collectionType, semtypes.List):
		valueType = semtypes.Never
		for _, each := range semtypes.ListAllMemberTypesInner(tyCtx, collectionType).Types {
			valueType = semtypes.Union(valueType, each)
		}
	case semtypes.IsSubtype(tyCtx, collectionType, semtypes.Mapping):
		valueType = semtypes.MappingMemberTypeInnerVal(tyCtx, collectionType, semtypes.String)
	case semtypes.IsSubtype(tyCtx, collectionType, semtypes.XML):
		valueType = semtypes.XMLItemType(collectionType)
	case semtypes.IsSubtype(tyCtx, collectionType, semtypes.String):
		valueType = semtypes.Char
	default:
		iterableTy, ok := common.IterableType(cx.CompilerCtx())
		if !ok || !semtypes.IsSubtype(tyCtx, collectionType, iterableTy) {
			cx.SemanticErr("foreach collection must be subtype of object:Iterable", collection.GetPosition())
			return false
		}
		nextReturnTy := iteratorNextReturnType(tyCtx, collectionType)
		if !semtypes.IsSubtype(tyCtx, semtypes.Diff(nextReturnTy, semtypes.Mapping), semtypes.Nil) {
			cx.SemanticErr("foreach collection must have an iteration completion type of nil", collection.GetPosition())
			return false
		}
		valueType = semtypes.MappingMemberTypeInnerVal(tyCtx, semtypes.Intersect(nextReturnTy, semtypes.Mapping), semtypes.StringConst("value"))
	}
	if !semtypes.IsSubtype(tyCtx, valueType, variableType) {
		cx.SemanticErr(strings.Join([]string{"invalid variable type ", semtypes.ToString(tyCtx, variableType)}, ""), stmt.VariableDef.Var.GetPosition())
		return false
	}
	return true
}

// iteratorNextReturnType is the return type of next() on the iterator() of an
// Iterable; for a well-formed Iterable it has the shape record {| T value; |}|C.
func iteratorNextReturnType(tyCtx semtypes.Context, iterableTy semtypes.SemType) semtypes.SemType {
	argsDefn := semtypes.NewListDefinition()
	emptyArgs := argsDefn.Define(tyCtx.Env(), nil, semtypes.ListMutability(semtypes.CellMutabilityNone))
	iteratorTy := semtypes.FunctionReturnType(tyCtx, semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("iterator"), iterableTy), emptyArgs)
	return semtypes.FunctionReturnType(tyCtx, semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("next"), iteratorTy), emptyArgs)
}

func validateDo(cx scope.Context, stmt *ast.BLangDo) {
	validateBlock(cx, stmt.Body.Stmts)
	if stmt.OnFailClause.Body != nil {
		validateOnFail(cx, &stmt.OnFailClause)
	}
}

// validateOnFail validates the error variable and the body; an error in the
// variable does not abandon the body.
func validateOnFail(cx scope.Context, clause *ast.BLangOnFailClause) {
	if clause.VariableDefinitionNode != nil {
		validateVariableDef(cx, scope.DefineBoundLocal(cx, clause.VariableDefinitionNode))
	}
	validateBlock(cx, clause.Body.Stmts)
}

// validateMatch validates the subject and each clause; an error in the subject
// does not abandon the clauses, and a clause that reports is abandoned and
// the next clause runs.
func validateMatch(cx scope.Context, stmt *ast.BLangMatchStatement) {
	validateActionOrExpr(cx, stmt.Expr, semtypes.SemType{})
	for i := range stmt.MatchClauses {
		validateMatchClause(cx, &stmt.MatchClauses[i])
	}
}

func validateMatchClause(cx scope.Context, clause *ast.BLangMatchClause) {
	for _, pattern := range clause.Patterns {
		if !dispatchMatchPattern(cx, pattern) {
			return
		}
	}
	if clause.Guard != nil {
		if _, ok := validateActionOrExpr(cx, clause.Guard, semtypes.Boolean); !ok {
			return
		}
	}
	validateBlock(cx, clause.Body.Stmts)
}

func dispatchMatchPattern(cx scope.Context, pattern ast.BLangMatchPattern) bool {
	switch pattern := pattern.(type) {
	case *ast.BLangConstPattern:
		_, ok := validateActionOrExpr(cx, pattern.Expr, semtypes.SemType{})
		return ok
	case *ast.BLangWildCardMatchPattern:
		return true
	default:
		cx.InternalErr("unexpected match pattern", pattern.GetPosition())
		return false
	}
}

func validateReturn(cx scope.Context, ret *ast.BLangReturn) {
	fn, inFunction := scope.EnclosingFunction(cx)
	if !inFunction || semtypes.IsZero(fn.RetTy) {
		cx.SemanticErr("return statement not allowed in this context", ret.GetPosition())
		return
	}
	f := scope.NoFacts
	if ret.Expr != nil {
		var ok bool
		f, ok = validateActionOrExpr(cx, ret.Expr, fn.RetTy)
		if !ok {
			return
		}
	} else if !semtypes.IsSubtype(cx.TyCtx(), fn.RetTy, semtypes.Nil) {
		cx.SemanticErr("expect a return value", ret.GetPosition())
		return
	}
	scope.ApplyReturnStmt(cx, ret, f, ret.Expr != nil)
}

// validateLock validates a lock body under its lock scope; the body must
// reference a restricted variable. A nested lock is reported and its body is
// validated under the enclosing lock.
func validateLock(cx scope.Context, lock *ast.BLangLock) {
	if _, nested := scope.EnclosingLock(cx); nested {
		cx.SemanticErr("lock statement cannot be nested inside another lock statement", lock.GetPosition())
		validateBlock(cx, lock.Body.Stmts)
		return
	}
	lockCx := scope.NewLock(cx, lock)
	validateBlock(lockCx, lock.Body.Stmts)
	if info, _ := scope.EnclosingLock(lockCx); info.Restricted.IsEmpty() {
		cx.SemanticErr("no restricted variable referenced in lock statement", info.Position)
	}
}

// validateAssignment validates an assignment or compound assignment: the
// target, the value, and the final-variable and target-kind rules. A final
// variable is assigned only by its deferred initialization: a module variable
// in the module init function, or a local of the current function; a final
// field only by a plain `=` in its class init when it has no initializer.
func validateAssignment(cx scope.Context, a ast.AssignmentNode) {
	_, compound := a.(*ast.BLangCompoundAssignment)
	if access, isField := a.GetVariable().(*ast.BLangFieldBaseAccess); isField {
		if name, reassigned := scope.FinalFieldReassigned(cx, access, compound); reassigned {
			cx.SemanticErr(strings.Join([]string{"cannot assign a value to final field '", name, "'"}, ""), access.GetPosition())
			return
		}
	}
	lhs, ok := dispatchTarget(cx, a.GetVariable())
	if !ok {
		return
	}
	rhs, ok := validateActionOrExpr(cx, a.GetExpression(), assignmentExpectedType(compound, a.GetVariable().Type()))
	if !ok {
		return
	}
	if ref, isVar := a.GetVariable().(*ast.BLangVarRef); isVar {
		sym := ref.Symbol()
		if sym.IsEmpty() {
			cx.InternalErr("unexpected nil symbol", ref.GetPosition())
			return
		}
		ctx := cx.CompilerCtx()
		if meta, isValue := ctx.ValueSymbolMetadata(sym); isValue && meta.Final {
			fn, inFunction := scope.EnclosingFunction(cx)
			md, moduleVar := cx.ModuleVar(sym)
			deferredModuleInit := !compound && inFunction && fn.IsModuleInit() && moduleVar && md.HasNoInitializer()
			local, owner, isLocal := scope.LookupLocal(cx, sym)
			deferredLocal := !compound && inFunction && isLocal && owner == fn.ID && local.HasNoInitializer()
			if !deferredModuleInit && !deferredLocal {
				cx.SemanticErr(strings.Join([]string{"cannot assign a value to final variable '", ctx.SymbolName(sym), "'"}, ""), ref.GetPosition())
				return
			}
			if deferredLocal && fn.IsLambda() {
				cx.UnimplementedErr("deferred initialization of a final variable in a lambda", ref.GetPosition())
				return
			}
		}
		if msg, notAssignable := common.AssignmentTargetError(ctx.SymbolKind(sym)); notAssignable {
			cx.SemanticErr(msg, ref.GetPosition())
			return
		}
	}
	scope.ApplyAssignment(cx, a, lhs, rhs)
}

// assignmentExpectedType is the target's type for `=`; a compound
// assignment's `target op value` was checked by type resolution.
func assignmentExpectedType(compound bool, targetType semtypes.SemType) semtypes.SemType {
	if compound {
		return semtypes.SemType{}
	}
	return targetType
}

// dispatchTarget validates an assignment target top-down and builds its
// LValue; a receiver that is not itself a target is an ordinary expression.
func dispatchTarget(cx scope.Context, e ast.BLangExpression) (scope.LValue, bool) {
	switch e := e.(type) {
	case *ast.BLangVarRef:
		return validateTargetVar(cx, e)
	case *ast.BLangWildCardBindingPattern:
		return validateTargetWildcard(cx, e)
	case *ast.BLangFieldBaseAccess:
		return validateTargetField(cx, e)
	case *ast.BLangIndexBasedAccess:
		return validateTargetIndex(cx, e)
	case *ast.BLangBadExprOrAction:
		cx.InternalErr("failed to find variable symbol", e.GetPosition())
		return scope.LValue{}, false
	default:
		_, ok := validateActionOrExpr(cx, e, semtypes.SemType{})
		return scope.LValue{}, ok
	}
}

func validateTargetVar(cx scope.Context, ref *ast.BLangVarRef) (scope.LValue, bool) {
	scope.ApplyVarRef(cx, ref)
	lv := scope.TargetVar(ref)
	if !validateResolvedType(cx, ref, semtypes.SemType{}) {
		return lv, false
	}
	return lv, true
}

func validateTargetWildcard(cx scope.Context, p *ast.BLangWildCardBindingPattern) (scope.LValue, bool) {
	lv := scope.TargetWildcard(p)
	if !validateResolvedType(cx, p, semtypes.SemType{}) {
		return lv, false
	}
	return lv, true
}

func validateTargetField(cx scope.Context, access *ast.BLangFieldBaseAccess) (scope.LValue, bool) {
	inner, ok := dispatchTarget(cx, access.Expr)
	if !ok {
		return inner, false
	}
	scope.ApplyFieldAccess(cx, access, scope.ReceiverSymbol(access))
	lv := scope.TargetField(cx, inner, access)
	if !validateResolvedType(cx, access, semtypes.SemType{}) {
		return lv, false
	}
	return lv, true
}

func validateTargetIndex(cx scope.Context, access *ast.BLangIndexBasedAccess) (scope.LValue, bool) {
	inner, ok := dispatchTarget(cx, access.Expr)
	if !ok {
		return inner, false
	}
	if !validateIndexContainer(cx, access) {
		return inner, false
	}
	if _, ok := validateActionOrExpr(cx, access.IndexExpr, indexKeyType(cx.TyCtx(), access.Expr.Type())); !ok {
		return inner, false
	}
	lv := scope.TargetIndex(inner, access)
	if !validateResolvedType(cx, access, semtypes.SemType{}) {
		return lv, false
	}
	return lv, true
}
