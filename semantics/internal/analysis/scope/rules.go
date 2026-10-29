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
	"github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type ruleKind uint8

const (
	outsideLock ruleKind = iota
	isolation
	lockCandidate
	lockTransfer
	lockCall
)

// ruleSet is a static set of cross-cutting rules, one hook per node kind; a
// nil hook means the set is not interested in that kind.
type ruleSet struct {
	kind           ruleKind
	varRef         func(cx Context, ref *ast.BLangVarRef)
	fieldAccess    func(cx Context, access *ast.BLangFieldBaseAccess, receiver model.SymbolRef)
	invocation     func(cx Context, inv *ast.BLangInvocation, f Facts)
	remoteCall     func(cx Context, call *ast.BLangRemoteMethodCallAction, f Facts)
	resourceAccess func(cx Context, call *ast.BLangClientResourceAccessAction, f Facts)
	startAction    func(cx Context, start *ast.BLangStartAction, f Facts)
	newExpr        func(cx Context, n *ast.BLangNewExpression)
	assignment     func(cx Context, a ast.AssignmentNode, lhs LValue, rhs Facts)
	returnStmt     func(cx Context, ret *ast.BLangReturn, f Facts, hasExpr bool)
}

func outsideLockRules() ruleSet {
	return ruleSet{kind: outsideLock, varRef: outsideLockVarRef, fieldAccess: outsideLockFieldAccess}
}

func isolationRules() ruleSet {
	return ruleSet{
		kind:           isolation,
		varRef:         isolationVarRef,
		invocation:     isolationInvocation,
		remoteCall:     isolationRemoteCall,
		resourceAccess: isolationResourceAccess,
		startAction:    isolationStartAction,
		newExpr:        isolationNewExpr,
	}
}

func lockCandidateRules() ruleSet {
	return ruleSet{kind: lockCandidate, varRef: lockCandidateVarRef, fieldAccess: lockCandidateFieldAccess}
}

func lockTransferRules() ruleSet {
	return ruleSet{kind: lockTransfer, varRef: lockTransferVarRef, assignment: lockAssignment, returnStmt: lockReturn}
}

func lockCallRules() ruleSet {
	return ruleSet{
		kind:           lockCall,
		invocation:     lockInvocation,
		remoteCall:     lockRemoteCall,
		resourceAccess: lockResourceAccess,
	}
}

func (r ruleSet) applyVarRef(cx Context, ref *ast.BLangVarRef) {
	if r.varRef != nil {
		r.varRef(cx, ref)
	}
}

func (r ruleSet) applyFieldAccess(cx Context, access *ast.BLangFieldBaseAccess, receiver model.SymbolRef) {
	if r.fieldAccess != nil {
		r.fieldAccess(cx, access, receiver)
	}
}

func (r ruleSet) applyInvocation(cx Context, inv *ast.BLangInvocation, f Facts) {
	if r.invocation != nil {
		r.invocation(cx, inv, f)
	}
}

func (r ruleSet) applyRemoteCall(cx Context, call *ast.BLangRemoteMethodCallAction, f Facts) {
	if r.remoteCall != nil {
		r.remoteCall(cx, call, f)
	}
}

func (r ruleSet) applyResourceAccess(cx Context, call *ast.BLangClientResourceAccessAction, f Facts) {
	if r.resourceAccess != nil {
		r.resourceAccess(cx, call, f)
	}
}

func (r ruleSet) applyStartAction(cx Context, start *ast.BLangStartAction, f Facts) {
	if r.startAction != nil {
		r.startAction(cx, start, f)
	}
}

func (r ruleSet) applyNewExpr(cx Context, n *ast.BLangNewExpression) {
	if r.newExpr != nil {
		r.newExpr(cx, n)
	}
}

func (r ruleSet) applyAssignment(cx Context, a ast.AssignmentNode, lhs LValue, rhs Facts) {
	if r.assignment != nil {
		r.assignment(cx, a, lhs, rhs)
	}
}

func (r ruleSet) applyReturnStmt(cx Context, ret *ast.BLangReturn, f Facts, hasExpr bool) {
	if r.returnStmt != nil {
		r.returnStmt(cx, ret, f, hasExpr)
	}
}

// ApplyVarRef runs the active rules for a variable reference, innermost set
// first.
func ApplyVarRef(cx Context, ref *ast.BLangVarRef) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyVarRef(cx, ref)
	}
}

// ApplyFieldAccess runs the active rules for a field access; receiver is
// ReceiverSymbol(access).
func ApplyFieldAccess(cx Context, access *ast.BLangFieldBaseAccess, receiver model.SymbolRef) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyFieldAccess(cx, access, receiver)
	}
}

// ApplyInvocation runs the active rules for a function or method call.
func ApplyInvocation(cx Context, inv *ast.BLangInvocation, f Facts) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyInvocation(cx, inv, f)
	}
}

// ApplyRemoteCall runs the active rules for a remote method call.
func ApplyRemoteCall(cx Context, call *ast.BLangRemoteMethodCallAction, f Facts) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyRemoteCall(cx, call, f)
	}
}

// ApplyResourceAccess runs the active rules for a client resource access.
func ApplyResourceAccess(cx Context, call *ast.BLangClientResourceAccessAction, f Facts) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyResourceAccess(cx, call, f)
	}
}

// ApplyStartAction checks that a start action is not lexically within a lock,
// then runs the active rules for it.
func ApplyStartAction(cx Context, start *ast.BLangStartAction, f Facts) {
	startOutsideLock(cx, start)
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyStartAction(cx, start, f)
	}
}

// ApplyNewExpr runs the active rules for a new expression.
func ApplyNewExpr(cx Context, n *ast.BLangNewExpression) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyNewExpr(cx, n)
	}
}

// ApplyAssignment runs the active rules for an assignment or compound
// assignment after both sides are validated.
func ApplyAssignment(cx Context, a ast.AssignmentNode, lhs LValue, rhs Facts) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyAssignment(cx, a, lhs, rhs)
	}
}

// ApplyReturnStmt runs the active rules for a return statement; f is
// NoFacts when the statement has no expression.
func ApplyReturnStmt(cx Context, ret *ast.BLangReturn, f Facts, hasExpr bool) {
	rules := cx.rules()
	common.Assert(func() bool { return activationConsistent(cx, rules) })
	for i := len(rules) - 1; i >= 0; i-- {
		rules[i].applyReturnStmt(cx, ret, f, hasExpr)
	}
}

// activationConsistent reports whether rules are ordered by kind without
// duplicates and match the enclosing lock, isolated expression and function.
func activationConsistent(cx Context, rules []ruleSet) bool {
	var present [lockCall + 1]bool
	for i, r := range rules {
		if present[r.kind] || (i > 0 && rules[i-1].kind >= r.kind) {
			return false
		}
		present[r.kind] = true
	}
	if _, inLock := EnclosingLock(cx); inLock {
		if !present[lockCandidate] || !present[lockCall] || present[outsideLock] {
			return false
		}
		if present[lockTransfer] == insideIsolatedExpr(cx) {
			return false
		}
	} else if !present[outsideLock] || present[lockCandidate] || present[lockTransfer] || present[lockCall] {
		return false
	}
	fn, inFunction := EnclosingFunction(cx)
	return (inFunction && fn.IsIsolated()) == present[isolation]
}

func insideIsolatedExpr(cx Context) bool {
	for cur := cx; cur != nil; cur = cur.parent() {
		switch cur.(type) {
		case *isolatedExprContext:
			return true
		case *lockContext, *functionContext, *packageContext:
			return false
		}
	}
	return false
}
