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
	"github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	semcommon "github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// validateActionOrExpr validates an expression or action against expected
// (zero for none). Inside a lock an expression whose type is a subtype of
// Isolated is an isolated expression: references to variables outside the
// lock are allowed within it.
func validateActionOrExpr(cx scope.Context, e ast.BLangActionOrExpression, expected semtypes.SemType) (scope.Facts, bool) {
	exprCx := cx
	if expr, isExpr := e.(ast.BLangExpression); isExpr {
		tyCtx := cx.TyCtx()
		if _, inLock := scope.EnclosingLock(cx); inLock && semtypes.IsSubtype(tyCtx, expr.Type(), semtypes.CreateIsolated(tyCtx)) {
			exprCx = scope.NewIsolatedExpr(cx, expr)
		}
	}
	f, ok := validateActionOrExprInner(exprCx, e, expected)
	if !ok {
		return f, false
	}
	common.Assert(func() bool { return !f.IsolatedTransferred() || f.Isolated() })
	return f, true
}

func validateActionOrExprInner(cx scope.Context, e ast.BLangActionOrExpression, expected semtypes.SemType) (scope.Facts, bool) {
	switch e := e.(type) {
	case *ast.BLangLiteral, *ast.BLangNumericLiteral, *ast.BLangConstRef, *ast.BLangWildCardBindingPattern,
		*ast.BLangInferredTypedescDefault, *ast.BLangDefaultArg, *ast.BLangXMLPILiteral, *ast.BLangXMLCommentLiteral,
		*ast.BLangXMLTextLiteral:
		return validateLeafExpr(cx, e, expected)
	case *ast.BLangVarRef:
		return validateVarRef(cx, e, expected)
	case *ast.BLangBinaryExpr:
		return validateBinaryExpr(cx, e, expected)
	case *ast.BLangTernaryExpr:
		return validateTernaryExpr(cx, e, expected)
	case *ast.BLangNilConditionalExpr:
		return validateNilConditionalExpr(cx, e, expected)
	case *ast.BLangUnaryExpr:
		return validateUnaryExpr(cx, e, expected)
	case *ast.BLangInvocation:
		return validateInvocation(cx, e, expected)
	case *ast.BLangIndexBasedAccess:
		return validateIndexBasedAccess(cx, e, expected)
	case *ast.BLangFieldBaseAccess:
		return validateFieldAccess(cx, e, expected)
	case *ast.BLangListConstructorExpr:
		return validateListConstructor(cx, e, expected)
	case *ast.BLangMappingConstructorExpr:
		return validateMappingConstructor(cx, e, expected)
	case *ast.BLangErrorConstructorExpr:
		return validateErrorConstructor(cx, e, expected)
	case *ast.BLangGroupExpr:
		return validateGroupExpr(cx, e, expected)
	case *ast.BLangQueryExpr:
		return validateQueryExpr(cx, e, expected)
	case *ast.BLangTypeConversionExpr:
		return validateTypeConversion(cx, e, expected)
	case *ast.BLangTypeTestExpr:
		return validateTypeTest(cx, e, expected)
	case *ast.BLangCheckedExpr:
		return validateCheckedExpr(cx, e, expected)
	case *ast.BLangCheckPanickedExpr:
		return validateCheckPanickedExpr(cx, e, expected)
	case *ast.BLangTrapExpr:
		return validateTrapExpr(cx, e, expected)
	case *ast.BLangNamedArgsExpression:
		return validateNamedArg(cx, e, expected)
	case *ast.BLangNewExpression:
		return validateNewExpr(cx, e, expected)
	case *ast.BLangLambdaFunction:
		return validateLambdaFunction(cx, e, expected)
	case *ast.BLangRemoteMethodCallAction:
		return validateRemoteMethodCall(cx, e, expected)
	case *ast.BLangClientResourceAccessAction:
		return validateClientResourceAccess(cx, e, expected)
	case *ast.BLangStartAction:
		return validateStart(cx, e, expected)
	case *ast.BLangSingleWaitAction:
		return validateSingleWait(cx, e, expected)
	case *ast.BLangAlternateWaitAction:
		return validateAlternateWait(cx, e, expected)
	case *ast.BLangMultipleWaitAction:
		return validateMultipleWait(cx, e, expected)
	case *ast.BLangTypedescExpr:
		return validateTypedescExpr(cx, e, expected)
	case *ast.BLangAnnotAccessExpr:
		return validateAnnotAccess(cx, e, expected)
	case *ast.BLangXMLElementLiteral:
		return validateXMLElement(cx, e, expected)
	case *ast.BLangXMLSequenceLiteral:
		return validateXMLSequence(cx, e, expected)
	case *ast.BLangTemplateExpr:
		return validateTemplate(cx, e, expected)
	case *ast.BLangXMLTemplateExpr:
		return validateXMLTemplate(cx, e, expected)
	case *ast.BLangXMLFilterExpression:
		return validateXMLFilter(cx, e, expected)
	case *ast.BLangXMLStepExpression:
		return validateXMLStep(cx, e, expected)
	default:
		cx.InternalErr("unexpected expression", e.GetPosition())
		return scope.NoFacts, false
	}
}

// validateResolvedType checks e's determined type against expected.
func validateResolvedType(cx scope.Context, e ast.BLangActionOrExpression, expected semtypes.SemType) bool {
	resolved := e.Type()
	if semtypes.IsZero(resolved) {
		cx.InternalErr("expression type not resolved", e.GetPosition())
		return false
	}
	tyCtx := cx.TyCtx()
	if !semtypes.IsZero(expected) &&
		(!semtypes.IsSubtype(tyCtx, resolved, expected) || (semtypes.IsNever(resolved) && !semtypes.IsNever(expected))) {
		cx.SemanticErr(semcommon.FormatIncompatibleTypeMessage(tyCtx, expected, resolved), e.GetPosition())
		return false
	}
	return true
}

func validateLeafExpr(cx scope.Context, e ast.BLangActionOrExpression, expected semtypes.SemType) (scope.Facts, bool) {
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateVarRef(cx scope.Context, ref *ast.BLangVarRef, expected semtypes.SemType) (scope.Facts, bool) {
	f := scope.FactsOfType(cx, ref)
	scope.ApplyVarRef(cx, ref)
	if !validateResolvedType(cx, ref, expected) {
		return f, false
	}
	return f, true
}

func validateFieldAccess(cx scope.Context, access *ast.BLangFieldBaseAccess, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, access.Expr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, access)
	scope.ApplyFieldAccess(cx, access, scope.ReceiverSymbol(access))
	if !validateResolvedType(cx, access, expected) {
		return f, false
	}
	return f, true
}

func validateIndexBasedAccess(cx scope.Context, access *ast.BLangIndexBasedAccess, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, access.Expr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	if !validateIndexContainer(cx, access) {
		return scope.NoFacts, false
	}
	if _, ok := validateActionOrExpr(cx, access.IndexExpr, indexKeyType(cx.TyCtx(), access.Expr.Type())); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, access)
	if !validateResolvedType(cx, access, expected) {
		return f, false
	}
	return f, true
}

// validateIndexContainer checks that the container can be indexed.
func validateIndexContainer(cx scope.Context, access *ast.BLangIndexBasedAccess) bool {
	tyCtx := cx.TyCtx()
	containerTy := access.Expr.Type()
	switch {
	case isIntIndexed(tyCtx, containerTy):
		return true
	case semtypes.IsSubtype(tyCtx, containerTy, semtypes.Table):
		cx.UnimplementedErr("table not supported", access.GetPosition())
		return false
	case semtypes.IsSubtype(tyCtx, containerTy, semtypes.Union(semtypes.Nil, semtypes.Mapping)):
		return true
	default:
		cx.SemanticErr("incompatible type for index based access", access.GetPosition())
		return false
	}
}

func isIntIndexed(tyCtx semtypes.Context, containerTy semtypes.SemType) bool {
	return semtypes.IsSubtype(tyCtx, containerTy, semtypes.List) || semtypes.IsSubtype(tyCtx, containerTy, semtypes.String) ||
		semtypes.IsSubtype(tyCtx, containerTy, semtypes.XML)
}

// indexKeyType is the type the key of a container validated by
// validateIndexContainer must have.
func indexKeyType(tyCtx semtypes.Context, containerTy semtypes.SemType) semtypes.SemType {
	if isIntIndexed(tyCtx, containerTy) {
		return semtypes.Int
	}
	return semtypes.String
}

// validateBinaryExpr validates each operand against the type its operator
// requires; arithmetic, shift and relational operands are checked by type
// resolution.
func validateBinaryExpr(cx scope.Context, e *ast.BLangBinaryExpr, expected semtypes.SemType) (scope.Facts, bool) {
	var ok bool
	switch e.GetOperatorKind() {
	case model.OperatorKind_EQUAL, model.OperatorKind_NOT_EQUAL:
		ok = validateEqualityExpr(cx, e)
	case model.OperatorKind_REF_EQUAL, model.OperatorKind_REF_NOT_EQUAL:
		ok = validateRefEqualityExpr(cx, e)
	case model.OperatorKind_EQUALS:
		cx.InternalErr(strings.Join([]string{"unexpected equality operator ", string(e.GetOperatorKind())}, ""), e.GetPosition())
		return scope.NoFacts, false
	case model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		ok = validateBinaryOperands(cx, e, semtypes.Union(semtypes.Int, semtypes.Nil))
	case model.OperatorKind_CLOSED_RANGE, model.OperatorKind_HALF_OPEN_RANGE:
		ok = validateBinaryOperands(cx, e, semtypes.Int)
	case model.OperatorKind_AND, model.OperatorKind_OR:
		ok = validateBinaryOperands(cx, e, semtypes.Boolean)
	default:
		ok = validateBinaryOperands(cx, e, semtypes.SemType{})
	}
	if !ok {
		return scope.NoFacts, false
	}
	return scope.FactsOfType(cx, e), validateResolvedType(cx, e, expected)
}

// validateEqualityExpr: == and != also need an anydata operand.
func validateEqualityExpr(cx scope.Context, e *ast.BLangBinaryExpr) bool {
	if !validateRefEqualityExpr(cx, e) {
		return false
	}
	tyCtx := cx.TyCtx()
	anydata := semtypes.CreateAnydata(tyCtx)
	if !semtypes.IsSubtype(tyCtx, e.LhsExpr.Type(), anydata) && !semtypes.IsSubtype(tyCtx, e.RhsExpr.Type(), anydata) {
		cx.SemanticErr(strings.Join([]string{"expect anydata types for ", string(e.GetOperatorKind())}, ""), e.GetPosition())
		return false
	}
	return true
}

// validateRefEqualityExpr: the operand types of an equality must intersect.
func validateRefEqualityExpr(cx scope.Context, e *ast.BLangBinaryExpr) bool {
	if !validateBinaryOperands(cx, e, semtypes.SemType{}) {
		return false
	}
	tyCtx := cx.TyCtx()
	if semtypes.IsEmpty(tyCtx, semtypes.Intersect(e.LhsExpr.Type(), e.RhsExpr.Type())) {
		cx.SemanticErr(strings.Join([]string{"incompatible types for ", string(e.GetOperatorKind())}, ""), e.GetPosition())
		return false
	}
	return true
}

func validateBinaryOperands(cx scope.Context, e *ast.BLangBinaryExpr, operandTy semtypes.SemType) bool {
	if _, ok := validateActionOrExpr(cx, e.LhsExpr, operandTy); !ok {
		return false
	}
	_, ok := validateActionOrExpr(cx, e.RhsExpr, operandTy)
	return ok
}

func validateTernaryExpr(cx scope.Context, e *ast.BLangTernaryExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.Condition, semtypes.Boolean); !ok {
		return scope.NoFacts, false
	}
	if _, ok := validateActionOrExpr(cx, e.ThenExpr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	if _, ok := validateActionOrExpr(cx, e.ElseExpr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateNilConditionalExpr(cx scope.Context, e *ast.BLangNilConditionalExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.LhsExpr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	if _, ok := validateActionOrExpr(cx, e.RhsExpr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateUnaryExpr(cx scope.Context, e *ast.BLangUnaryExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.Expr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	tyCtx := cx.TyCtx()
	exprTy := e.Expr.Type()
	op := string(e.GetOperatorKind())
	switch e.GetOperatorKind() {
	case model.OperatorKind_ADD, model.OperatorKind_SUB, model.OperatorKind_BITWISE_COMPLEMENT:
		if !semcommon.IsNumericType(tyCtx, semtypes.Diff(exprTy, semtypes.Nil)) {
			cx.SemanticErr(strings.Join([]string{"expect numeric type for ", op}, ""), e.GetPosition())
			return f, false
		}
	case model.OperatorKind_NOT:
		if !semtypes.IsSubtype(tyCtx, exprTy, semtypes.Boolean) {
			cx.SemanticErr(strings.Join([]string{"expect boolean type for ", op}, ""), e.GetPosition())
			return f, false
		}
	default:
		cx.SemanticErr(strings.Join([]string{"unsupported unary operator: ", op}, ""), e.GetPosition())
		return f, false
	}
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateGroupExpr(cx scope.Context, e *ast.BLangGroupExpr, expected semtypes.SemType) (scope.Facts, bool) {
	inner, ok := validateActionOrExpr(cx, e.Expression, expected)
	if !ok {
		return scope.NoFacts, false
	}
	f := scope.Either(scope.FactsOfType(cx, e), inner)
	return f, true
}

func validateTypeConversion(cx scope.Context, e *ast.BLangTypeConversionExpr, expected semtypes.SemType) (scope.Facts, bool) {
	inner, ok := validateActionOrExpr(cx, e.Expression, semtypes.SemType{})
	if !ok {
		return scope.NoFacts, false
	}
	if !dispatchTypeDesc(cx, e.TypeDescriptor) {
		return scope.NoFacts, false
	}
	f := scope.Either(scope.FactsOfType(cx, e), inner)
	tyCtx := cx.TyCtx()
	exprTy := e.Expression.Type()
	targetTy := e.TypeDescriptor.Type()
	if semtypes.IsEmpty(tyCtx, semtypes.Intersect(exprTy, targetTy)) && !hasPotentialNumericConversion(exprTy, targetTy) {
		cx.SemanticErr("impossible type conversion, intersection is empty", e.GetPosition())
		return f, false
	}
	if !semtypes.IsZero(expected) && !semtypes.IsSubtype(tyCtx, targetTy, expected) {
		cx.SemanticErr(semcommon.FormatIncompatibleTypeMessage(tyCtx, expected, targetTy), e.GetPosition())
		return f, false
	}
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func hasPotentialNumericConversion(exprTy, targetTy semtypes.SemType) bool {
	return semtypes.SingleNumericType(targetTy).IsPresent() && semtypes.ContainsBasicType(exprTy, semtypes.Number)
}

func validateTypeTest(cx scope.Context, e *ast.BLangTypeTestExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.Expr, semtypes.SemType{}); !ok {
		return scope.NoFacts, false
	}
	if !dispatchTypeData(cx, &e.TestedType) {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

// validateCheckedExpr checks that the error part of the operand is a subtype
// of the enclosing function's return type. A default expression is its own
// closure with no return type yet (#1219).
func validateCheckedExpr(cx scope.Context, e *ast.BLangCheckedExpr, expected semtypes.SemType) (scope.Facts, bool) {
	inner, ok := validateActionOrExpr(cx, e.Expr, semtypes.SemType{})
	if !ok {
		return scope.NoFacts, false
	}
	f := scope.Either(scope.FactsOfType(cx, e), inner)
	fn, inFunction := scope.EnclosingFunction(cx)
	if inFunction && fn.IsExpressionClosure() {
		cx.UnimplementedErr("'check' in a default expression is not supported", e.GetPosition())
		return f, false
	}
	if inFunction && !semtypes.IsZero(fn.RetTy) {
		tyCtx := cx.TyCtx()
		errorPart := semtypes.Intersect(e.Expr.Type(), semtypes.Error)
		if !semtypes.IsEmpty(tyCtx, errorPart) && !semtypes.IsSubtype(tyCtx, errorPart, fn.RetTy) {
			cx.SemanticErr("error type of check expression is not a subtype of the enclosing function's return type", e.GetPosition())
			return f, false
		}
	}
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateCheckPanickedExpr(cx scope.Context, e *ast.BLangCheckPanickedExpr, expected semtypes.SemType) (scope.Facts, bool) {
	inner, ok := validateActionOrExpr(cx, e.Expr, semtypes.SemType{})
	if !ok {
		return scope.NoFacts, false
	}
	f := scope.Either(scope.FactsOfType(cx, e), inner)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateTrapExpr(cx scope.Context, e *ast.BLangTrapExpr, expected semtypes.SemType) (scope.Facts, bool) {
	inner, ok := validateActionOrExpr(cx, e.Expr, semtypes.SemType{})
	if !ok {
		return scope.NoFacts, false
	}
	f := scope.Either(scope.FactsOfType(cx, e), inner)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateNamedArg(cx scope.Context, e *ast.BLangNamedArgsExpression, expected semtypes.SemType) (scope.Facts, bool) {
	inner, ok := validateActionOrExpr(cx, e.Expr, expected)
	if !ok {
		return scope.NoFacts, false
	}
	f := scope.Either(scope.FactsOfType(cx, e), inner)
	return f, true
}

func validateLambdaFunction(cx scope.Context, e *ast.BLangLambdaFunction, expected semtypes.SemType) (scope.Facts, bool) {
	if !validateInvokableNode(scope.NewLambda(cx, e.Function), e.Function) {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateTypedescExpr(cx scope.Context, e *ast.BLangTypedescExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if td := e.GetTypeDescriptor(); td != nil {
		if !dispatchTypeDesc(cx, td) {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

// validateAnnotAccess: annotation access is only valid on a typedesc value.
func validateAnnotAccess(cx scope.Context, e *ast.BLangAnnotAccessExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.Expr, semtypes.Typedesc); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateNewExpr(cx scope.Context, e *ast.BLangNewExpression, expected semtypes.SemType) (scope.Facts, bool) {
	if e.TypeDescriptor != nil {
		if !dispatchTypeDesc(cx, e.TypeDescriptor) {
			return scope.NoFacts, false
		}
	}
	if ast.IsStreamNewExpression(e) {
		if !validateStreamNewArgs(cx, e) {
			return scope.NoFacts, false
		}
	} else {
		for _, arg := range e.ArgsExprs {
			if _, ok := validateActionOrExpr(cx, arg, semtypes.SemType{}); !ok {
				return scope.NoFacts, false
			}
		}
	}
	f := scope.FactsOfType(cx, e)
	scope.ApplyNewExpr(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

// validateStreamNewArgs checks the stream implementor argument of
// `new stream<T, C>(impl)` and its optional close method.
func validateStreamNewArgs(cx scope.Context, e *ast.BLangNewExpression) bool {
	tyCtx := cx.TyCtx()
	streamTy := e.Type()
	valueTy := semtypes.StreamValueType(tyCtx, streamTy)
	completionTy := semtypes.StreamCompletionType(tyCtx, streamTy)
	if semtypes.IsZero(valueTy) || semtypes.IsZero(completionTy) {
		cx.InternalErr("failed to extract stream type parameters", e.GetPosition())
		return false
	}
	impl := e.ArgsExprs[0]
	if _, ok := validateActionOrExpr(cx, impl, semtypes.CreateStreamImplementorType(tyCtx, valueTy, completionTy)); !ok {
		return false
	}
	implTy := impl.Type()
	closeName := semtypes.StringConst("close")
	if semtypes.IsSubtype(tyCtx, semtypes.ObjectMemberKind(tyCtx, closeName, implTy), semtypes.StringConst("method")) {
		if !semtypes.IsSubtype(tyCtx, semtypes.ObjectMemberVisibility(tyCtx, closeName, implTy), semtypes.StringConst(semtypes.VisibilityPublic)) {
			cx.SemanticErr("stream implementor close method must be public", impl.GetPosition())
			return false
		}
		paramsDefn := semtypes.NewListDefinition()
		emptyParams := paramsDefn.Define(tyCtx.Env(), nil, semtypes.ListMutability(semtypes.CellMutabilityNone))
		returnTy := semtypes.FunctionReturnType(tyCtx, semtypes.ObjectMemberType(tyCtx, closeName, implTy), emptyParams)
		if semtypes.IsZero(returnTy) || !semtypes.IsSubtype(tyCtx, returnTy, semtypes.Union(completionTy, semtypes.Nil)) {
			cx.SemanticErr("stream implementor close method is incompatible", impl.GetPosition())
			return false
		}
	}
	return true
}

func validateSingleWait(cx scope.Context, e *ast.BLangSingleWaitAction, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := validateActionOrExpr(cx, e.FutureExpr, semtypes.Future); !ok {
		return scope.NoFacts, false
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateAlternateWait(cx scope.Context, e *ast.BLangAlternateWaitAction, expected semtypes.SemType) (scope.Facts, bool) {
	for _, future := range e.FutureExprs {
		if _, ok := validateActionOrExpr(cx, future, semtypes.Future); !ok {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

func validateMultipleWait(cx scope.Context, e *ast.BLangMultipleWaitAction, expected semtypes.SemType) (scope.Facts, bool) {
	for _, future := range e.FutureExprs {
		if _, ok := validateActionOrExpr(cx, future, semtypes.Future); !ok {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}
