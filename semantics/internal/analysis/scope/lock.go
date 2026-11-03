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

package scope

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// lockCandidateVarRef registers an isolated module-level variable as the
// lock's restricted variable.
func lockCandidateVarRef(cx Context, ref *ast.BLangVarRef) {
	sym, ok := restrictedModuleVar(cx, ref.Symbol())
	if !ok {
		return
	}
	ctx := cx.CompilerCtx()
	registerRestricted(cx, sym, moduleVarLockKey(ctx.SymbolPackage(sym), ctx.SymbolName(sym)), ref.GetPosition())
}

// lockCandidateFieldAccess registers a mutable field of the enclosing
// isolated object, accessed through self, as the lock's restricted variable.
func lockCandidateFieldAccess(cx Context, access *ast.BLangFieldBaseAccess, receiver model.SymbolRef) {
	fn, ok := EnclosingFunction(cx)
	if !ok || !fn.isSelf(receiver) || !fn.class.IsIsolated() {
		return
	}
	field, ok := fn.class.field(access.Field.GetValue())
	if !ok || field.immutable {
		return
	}
	registerRestricted(cx, field.sym, field.lockKey, access.GetPosition())
}

// registerRestricted records sym as the restricted variable of the innermost
// lock if it has none, and reports a reference at pos to a second restricted
// variable.
func registerRestricted(cx Context, sym model.SymbolRef, key string, pos diagnostics.Location) {
	lc, ok := innermostLock(cx)
	if !ok {
		cx.InternalErr("lock candidate rule outside a lock", pos)
		return
	}
	if lc.restricted.IsEmpty() {
		lc.setRestricted(sym, key)
		return
	}
	if lc.restricted != sym {
		cx.SemanticErr("more than one restricted variable referenced in lock statement", pos)
	}
}

// lockTransferVarRef: a reference to a variable declared outside the lock,
// other than self and a restricted candidate, must be an isolated expression
// by its own type; inside an isolated expression this rule is not active.
func lockTransferVarRef(cx Context, ref *ast.BLangVarRef) {
	sym := ref.Symbol()
	if fn, ok := EnclosingFunction(cx); ok && fn.isSelf(sym) {
		return
	}
	if inLockLocals(cx, sym) || isRestrictedCandidate(cx, sym) {
		return
	}
	tyCtx := cx.TyCtx()
	if !semtypes.IsSubtype(tyCtx, ref.Type(), semtypes.CreateIsolated(tyCtx)) {
		cx.SemanticErr("access of mutable variable", ref.GetPosition())
	}
}

// lockAssignment: an assignment to a target declared outside the lock, other
// than a restricted candidate, needs a plain variable name on the left-hand
// side and an isolated expression on the right-hand side.
func lockAssignment(cx Context, a ast.AssignmentNode, lhs LValue, rhs Facts) {
	if lhs.kind == lvWildcard || isRestrictedCandidate(cx, lhs.root) || inLockLocals(cx, lhs.root) {
		return
	}
	if lhs.kind != lvPlain {
		cx.SemanticErr("assignment in a lock statement to a target defined outside the lock must use a plain variable name on the left-hand side", a.GetPosition())
		return
	}
	if !rhs.IsolatedTransferred() {
		cx.SemanticErr("access of mutable variable", a.GetExpression().GetPosition())
	}
}

// lockReturn: the value returned from inside a lock must be an isolated
// expression.
func lockReturn(cx Context, ret *ast.BLangReturn, f Facts, hasExpr bool) {
	if hasExpr && !f.IsolatedTransferred() {
		cx.SemanticErr("not an isolated expression", ret.Expr.GetPosition())
	}
}

// lockInvocation: only isolated functions may be called inside a lock. Type
// resolution resolves a lock body in an isolated context, so an argument to an
// isolated parameter is already an isolated function.
func lockInvocation(cx Context, inv *ast.BLangInvocation, _ Facts) {
	if ast.IsStreamOperation(inv) {
		return
	}
	argIndex, nonIsolated := nonIsolatedCall(cx, inv)
	if !nonIsolated {
		return
	}
	if argIndex >= 0 {
		cx.InternalErr("non-isolated argument to an isolated parameter inside lock statement", inv.CallArgs()[argIndex].GetPosition())
		return
	}
	cx.SemanticErr("invocation of a non-isolated function inside lock statement", inv.GetPosition())
}

// lockRemoteCall: only isolated remote methods may be called inside a lock.
func lockRemoteCall(cx Context, call *ast.BLangRemoteMethodCallAction, _ Facts) {
	if !calleeIsolated(cx, call.ResolvedSymbol()) {
		cx.SemanticErr("invocation of a non-isolated function inside lock statement", call.GetPosition())
	}
}

// lockResourceAccess: only isolated resource methods may be called inside a
// lock.
func lockResourceAccess(cx Context, call *ast.BLangClientResourceAccessAction, _ Facts) {
	if !calleeIsolated(cx, call.ResolvedSymbol()) {
		cx.SemanticErr("invocation of a non-isolated function inside lock statement", call.GetPosition())
	}
}

// startOutsideLock: a start action may not occur lexically within a lock
// statement, including within a closure defined in a lock body.
func startOutsideLock(cx Context, start *ast.BLangStartAction) {
	if lexicallyInLock(cx) {
		cx.SemanticErr("cannot start strand inside a lock statement", start.GetPosition())
	}
}
