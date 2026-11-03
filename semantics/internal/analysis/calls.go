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
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func validateInvocation(cx scope.Context, inv *ast.BLangInvocation, expected semtypes.SemType) (scope.Facts, bool) {
	if ast.IsStreamOperation(inv) && len(inv.ArgExprs) != 0 {
		cx.SemanticErr(strings.Join([]string{"stream method '", inv.Name.GetValue(), "' takes no arguments"}, ""), inv.GetPosition())
		return scope.NoFacts, false
	}
	return validateInvocationInner(cx, inv, semtypes.SemType{}, scope.AllFacts, scope.ApplyInvocation, expected)
}

func validateRemoteMethodCall(cx scope.Context, call *ast.BLangRemoteMethodCallAction, expected semtypes.SemType) (scope.Facts, bool) {
	return validateInvocationInner(cx, call, semtypes.SemType{}, scope.AllFacts, scope.ApplyRemoteCall, expected)
}

// validateClientResourceAccess validates the computed path segments against
// the resource method's path type, then the receiver and arguments.
func validateClientResourceAccess(cx scope.Context, call *ast.BLangClientResourceAccessAction, expected semtypes.SemType) (scope.Facts, bool) {
	rmSym, isResource := cx.CompilerCtx().GetSymbol(call.MethodSymbol()).(model.ResourceMethodSymbol)
	if !isResource {
		cx.InternalErr("client resource access action is not bound to a resource method", call.GetPosition())
		return scope.NoFacts, false
	}
	tyCtx := cx.TyCtx()
	operands := scope.AllFacts
	for i := range call.Path {
		if call.Path[i].Kind == ast.ResourceAccessSegmentComputed {
			segment, ok := validateActionOrExpr(cx, call.Path[i].Expr, semtypes.ListMemberTypeInnerVal(tyCtx, rmSym.PathListType(), semtypes.IntConst(int64(i))))
			if !ok {
				return scope.NoFacts, false
			}
			operands = scope.Meet(operands, segment)
		}
	}
	return validateInvocationInner(cx, call, semtypes.CreateClientObject(tyCtx), operands, scope.ApplyResourceAccess, expected)
}

// validateInvocationInner validates the receiver and arguments of call,
// folding their facts into operands, then runs the active rules on the call's
// facts and checks its resolved type.
func validateInvocationInner[T ast.Invocable](cx scope.Context, call T, receiverTy semtypes.SemType, operands scope.Facts,
	apply func(scope.Context, T, scope.Facts), expected semtypes.SemType) (scope.Facts, bool) {
	if recv := call.Receiver(); recv != nil {
		receiver, ok := validateActionOrExpr(cx, recv, receiverTy)
		if !ok {
			return scope.NoFacts, false
		}
		operands = scope.Meet(operands, receiver)
	}
	args, ok := validateCallArgs(cx, call)
	if !ok {
		return scope.NoFacts, false
	}
	f := scope.CallFacts(cx, call, scope.Meet(operands, args))
	apply(cx, call, f)
	return f, validateResolvedType(cx, call, expected)
}

// validateCallArgs validates the (lowered) arguments of a call against its
// callee's parameter types and folds their facts.
func validateCallArgs(cx scope.Context, call ast.Invocable) (scope.Facts, bool) {
	operands := scope.AllFacts
	for i, arg := range call.CallArgs() {
		argFacts, ok := validateCallArg(cx, call, i, arg)
		if !ok {
			return scope.NoFacts, false
		}
		operands = scope.Meet(operands, argFacts)
	}
	return operands, true
}

// validateCallArg validates argument i of call against the callee's parameter
// type.
func validateCallArg(cx scope.Context, call ast.Invocable, i int, arg ast.BLangExpression) (scope.Facts, bool) {
	switch arg.(type) {
	case *ast.BLangNamedArgsExpression:
		cx.InternalErr("named argument after call-argument lowering", arg.GetPosition())
		return scope.NoFacts, false
	case ast.BLangExpression:
		tyCtx := cx.TyCtx()
		return validateActionOrExpr(cx, arg, semtypes.ListMemberTypeInnerVal(tyCtx,
			semtypes.FunctionParamListType(tyCtx, cx.CompilerCtx().SymbolType(call.ResolvedSymbol())), semtypes.IntConst(int64(i))))
	default:
		cx.InternalErr("missing call argument", call.GetPosition())
		return scope.NoFacts, false
	}
}

func validateStart(cx scope.Context, start *ast.BLangStartAction, expected semtypes.SemType) (scope.Facts, bool) {
	if _, ok := start.Call.(ast.Invocable); !ok {
		cx.InternalErr("start action operand is not invocable", start.GetPosition())
		return scope.NoFacts, false
	}
	callExpected, ok := startCallExpectedType(cx, start, expected)
	if !ok {
		return scope.NoFacts, false
	}
	call, ok := validateActionOrExpr(scope.NewStartCall(cx), start.Call, callExpected)
	if !ok {
		return scope.NoFacts, false
	}
	f := scope.StartFacts(call)
	start.IsIsolated = f.IsolatedTransferred()
	scope.ApplyStartAction(cx, start, f)
	return f, validateResolvedType(cx, start, expected)
}

// startCallExpectedType is the type the call of a start action must have;
// type resolution has already rejected an expected type with no future part.
func startCallExpectedType(cx scope.Context, start *ast.BLangStartAction, expected semtypes.SemType) (semtypes.SemType, bool) {
	if semtypes.IsZero(expected) {
		return semtypes.SemType{}, true
	}
	tyCtx := cx.TyCtx()
	future := semtypes.Intersect(expected, semtypes.Future)
	if semtypes.IsEmpty(tyCtx, future) {
		cx.InternalErr("start action without a future expected type", start.GetPosition())
		return semtypes.SemType{}, false
	}
	return semtypes.FutureEventualType(tyCtx, future), true
}
