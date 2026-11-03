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
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// EnclosingFunction returns the innermost closure.
func EnclosingFunction(cx Context) (FunctionInfo, bool) {
	for cur := cx; cur != nil; cur = cur.parent() {
		switch c := cur.(type) {
		case *packageContext:
			return FunctionInfo{}, false
		case *functionContext:
			return FunctionInfo{class: c.class, RetTy: c.retTy, symbol: c.symbol, self: c.self, ID: c.id, kind: c.kind, isolated: c.isolated}, true
		case *lockContext, *isolatedExprContext, *factsContext:
			continue
		default:
			cx.InternalErr("unknown context", diagnostics.NewBuiltinLocation())
			return FunctionInfo{}, false
		}
	}
	return FunctionInfo{}, false
}

// EnclosingLock returns the innermost lock of the current closure. A closure
// body is never inside a lock of the enclosing code.
func EnclosingLock(cx Context) (LockInfo, bool) {
	lc, ok := innermostLock(cx)
	if !ok {
		return LockInfo{}, false
	}
	return LockInfo{Position: lc.lock.GetPosition(), Restricted: lc.restricted}, true
}

// LookupLocal finds a local of the current closure or of an enclosing one and
// the closure that owns it.
func LookupLocal(cx Context, sym model.SymbolRef) (VarDecl, ScopeID, bool) {
	sym = cx.CompilerCtx().UnnarrowedSymbol(sym)
	for cur := cx; cur != nil; cur = cur.parent() {
		switch c := cur.(type) {
		case *packageContext:
			return VarDecl{}, 0, false
		case *functionContext:
			if md, ok := c.locals[sym]; ok {
				return md, c.id, true
			}
		case *lockContext, *isolatedExprContext, *factsContext:
			continue
		default:
			cx.InternalErr("unknown context", diagnostics.NewBuiltinLocation())
			return VarDecl{}, 0, false
		}
	}
	return VarDecl{}, 0, false
}

// inLockLocals reports whether sym was declared inside the innermost lock's
// body.
func inLockLocals(cx Context, sym model.SymbolRef) bool {
	sym = cx.CompilerCtx().UnnarrowedSymbol(sym)
	for cur := cx; cur != nil; cur = cur.parent() {
		switch c := cur.(type) {
		case *packageContext, *functionContext:
			return false
		case *lockContext:
			_, ok := c.locals[sym]
			return ok
		case *isolatedExprContext, *factsContext:
			continue
		default:
			cx.InternalErr("unknown context", diagnostics.NewBuiltinLocation())
			return false
		}
	}
	return false
}

// LocalDef is a variable definition recorded as a local by one of the Define*
// functions.
type LocalDef struct {
	def *ast.BLangVariableDef
}

func (d LocalDef) Node() *ast.BLangVariableDef { return d.def }

// DefineLocal records a variable definition as a local of the innermost
// closure and of every lock between it and the closure.
func DefineLocal(cx Context, def *ast.BLangVariableDef) LocalDef {
	v := def.Var
	defineLocal(cx, v.Symbol(), localVarDecl(cx, v, v.Expr == nil))
	return LocalDef{def: def}
}

// DefineBoundLocal is DefineLocal for a variable initialized by its binding
// construct (foreach, on-fail).
func DefineBoundLocal(cx Context, def *ast.BLangVariableDef) LocalDef {
	v := def.Var
	defineLocal(cx, v.Symbol(), localVarDecl(cx, v, false))
	return LocalDef{def: def}
}

// DefineQueryLocal is DefineBoundLocal for a variable bound by a query clause
// (from, join, let, group-by key), which is implicitly final. Its type is the
// symbol's, since type resolution does not set the variable node's type.
func DefineQueryLocal(cx Context, def *ast.BLangVariableDef) LocalDef {
	sym := def.Var.Symbol()
	defineLocal(cx, sym, newVarDecl(cx.CompilerCtx().SymbolType(sym), true, false, false, false))
	return LocalDef{def: def}
}

func localVarDecl(cx Context, v *ast.BLangVariable, noInitializer bool) VarDecl {
	final := v.IsFinal()
	if sym, ok := cx.CompilerCtx().GetSymbol(v.Symbol()).(model.ValueSymbol); ok && sym.IsFinal() {
		final = true
	}
	return newVarDecl(cx.CompilerCtx().SymbolType(v.Symbol()), final, false, false, noInitializer)
}

func defineLocal(cx Context, sym model.SymbolRef, md VarDecl) {
	for cur := cx; cur != nil; cur = cur.parent() {
		switch c := cur.(type) {
		case *packageContext:
			return
		case *functionContext:
			common.Assert(func() bool { _, defined := c.locals[sym]; return !defined })
			c.locals[sym] = md
			return
		case *lockContext:
			c.locals[sym] = struct{}{}
		case *isolatedExprContext, *factsContext:
			continue
		default:
			cx.InternalErr("unknown context", diagnostics.NewBuiltinLocation())
			return
		}
	}
}

// ReceiverSymbol returns the symbol of a field access's receiver when the
// receiver is a variable reference, else empty.
func ReceiverSymbol(access *ast.BLangFieldBaseAccess) model.SymbolRef {
	receiver, ok := access.Expr.(*ast.BLangVarRef)
	if !ok {
		return model.SymbolRef{}
	}
	return receiver.Symbol()
}

// setRestricted records the restricted variable of lc and writes its lock key
// and symbol on the lock node.
func (lc *lockContext) setRestricted(sym model.SymbolRef, key string) {
	common.Assert(func() bool { return lc.restricted.IsEmpty() })
	lc.restricted = sym
	lc.lock.LockKey = key
	lc.lock.RestrictedSymbol = sym
}

func innermostLock(cx Context) (*lockContext, bool) {
	for cur := cx; cur != nil; cur = cur.parent() {
		switch c := cur.(type) {
		case *packageContext, *functionContext:
			return nil, false
		case *lockContext:
			return c, true
		case *isolatedExprContext, *factsContext:
			continue
		default:
			cx.InternalErr("unknown context", diagnostics.NewBuiltinLocation())
			return nil, false
		}
	}
	return nil, false
}

// lexicallyInLock reports whether cx is within a lock body, either of the
// current closure or of any enclosing one.
func lexicallyInLock(cx Context) bool {
	for cur := cx; cur != nil; cur = cur.parent() {
		switch cur.(type) {
		case *packageContext:
			return false
		case *lockContext:
			return true
		case *functionContext, *isolatedExprContext, *factsContext:
			continue
		default:
			cx.InternalErr("unknown context", diagnostics.NewBuiltinLocation())
			return false
		}
	}
	return false
}

func nextScopeID(cx Context) ScopeID {
	id := ScopeID(1)
	for cur := cx; cur != nil; cur = cur.parent() {
		if _, ok := cur.(*functionContext); ok {
			id++
		}
	}
	return id
}

// isRestrictedCandidate reports whether sym can be a lock's restricted
// variable: an isolated module-level value, or a non-immutable field of the
// enclosing isolated class.
func isRestrictedCandidate(cx Context, sym model.SymbolRef) bool {
	sym = cx.CompilerCtx().UnnarrowedSymbol(sym)
	if _, ok := restrictedModuleVar(cx, sym); ok {
		return true
	}
	fn, ok := EnclosingFunction(cx)
	if !ok || !fn.class.IsIsolated() {
		return false
	}
	field, ok := fn.class.fieldBySymbol(sym)
	return ok && !field.immutable
}

// restrictedModuleVar returns the unnarrowed symbol of sym when it is an
// isolated module-level value, the module-level kind of restricted candidate.
func restrictedModuleVar(cx Context, sym model.SymbolRef) (model.SymbolRef, bool) {
	sym = cx.CompilerCtx().UnnarrowedSymbol(sym)
	md, ok := cx.ModuleVar(sym)
	return sym, ok && md.IsIsolated()
}
