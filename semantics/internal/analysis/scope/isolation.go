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
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// isolationVarRef: an isolated closure may read its own locals, capture
// final variables of an isolated type, and read module-level variables that
// are final or configurable, not isolated, and of an isolated type.
func isolationVarRef(cx Context, ref *ast.BLangVarRef) {
	sym := ref.Symbol()
	fn, ok := EnclosingFunction(cx)
	if !ok || fn.isSelf(sym) {
		return
	}
	tyCtx := cx.TyCtx()
	isolatedTy := semtypes.CreateIsolated(tyCtx)
	if md, owner, local := LookupLocal(cx, sym); local {
		if owner != fn.ID && (!md.IsFinal() || !semtypes.IsSubtype(tyCtx, md.Type(), isolatedTy)) {
			cx.SemanticErr("invalid capture of mutable variable in isolated lambda", ref.GetPosition())
		}
		return
	}
	if _, inLock := EnclosingLock(cx); inLock && isRestrictedCandidate(cx, sym) {
		return
	}
	md, ok := cx.ModuleVar(sym)
	if !ok {
		return
	}
	if (md.IsFinal() || md.isConfigurable()) && !md.IsIsolated() && semtypes.IsSubtype(tyCtx, md.Type(), isolatedTy) {
		return
	}
	cx.SemanticErr("access of mutable variable", ref.GetPosition())
}

// isolationInvocation: an isolated closure may only call isolated functions.
func isolationInvocation(cx Context, inv *ast.BLangInvocation, _ Facts) {
	if ast.IsStreamOperation(inv) {
		return
	}
	argIndex, nonIsolated := nonIsolatedCall(cx, inv)
	if !nonIsolated {
		return
	}
	if argIndex < 0 {
		cx.SemanticErr("invocation of a non-isolated function", inv.GetPosition())
		return
	}
	cx.SemanticErr("invocation of a non-isolated function", inv.CallArgs()[argIndex].GetPosition())
}

// isolationRemoteCall: an isolated closure may only call isolated remote
// methods.
func isolationRemoteCall(cx Context, call *ast.BLangRemoteMethodCallAction, _ Facts) {
	if !calleeIsolated(cx, call.ResolvedSymbol()) {
		cx.SemanticErr("invocation of a non-isolated function", call.GetPosition())
	}
}

// isolationResourceAccess: an isolated closure may only call isolated
// resource methods.
func isolationResourceAccess(cx Context, call *ast.BLangClientResourceAccessAction, _ Facts) {
	if !calleeIsolated(cx, call.ResolvedSymbol()) {
		cx.SemanticErr("invocation of a non-isolated function", call.GetPosition())
	}
}

// isolationStartAction: an isolated closure may only start an isolated call
// with isolated operands.
func isolationStartAction(cx Context, start *ast.BLangStartAction, f Facts) {
	if !f.IsolatedTransferred() {
		cx.SemanticErr("start action is not isolated", start.GetPosition())
	}
}

// isolationNewExpr: an isolated closure may only construct objects whose init
// method is isolated.
func isolationNewExpr(cx Context, n *ast.BLangNewExpression) {
	if ast.IsStreamNewExpression(n) {
		return
	}
	tyCtx := cx.TyCtx()
	classTy := cx.CompilerCtx().SymbolType(n.ClassSymbol)
	initTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("init"), classTy)
	if semtypes.IsZero(initTy) || !semtypes.IsSubtype(tyCtx, initTy, semtypes.CreateIsolatedFn(tyCtx)) {
		cx.SemanticErr("non isolated initialization", n.GetPosition())
	}
}
