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
)

// NewWorker pushes the closure of a named worker body. A worker has its own
// return context and its own locals, but no signature: it inherits isolation,
// class and self from the enclosing function, and its return type is the
// eventual type of its future<T> symbol.
func NewWorker(parent Context, worker *ast.BLangNamedWorkerDeclaration) Context {
	enclosing, _ := EnclosingFunction(parent)
	fc := &functionContext{
		contextBase: contextBase{outer: parent},
		locals:      make(map[model.SymbolRef]VarDecl),
		retTy:       semtypes.FutureEventualType(parent.TyCtx(), parent.CompilerCtx().SymbolType(worker.Symbol())),
		id:          nextScopeID(parent),
		kind:        fnWorker,
	}
	fc.class, fc.self = enclosing.class, enclosing.self
	fc.isolated = enclosing.IsIsolated()
	fc.active = closureRules(fc.isolated)
	return fc
}

// NewRepeatedRegion pushes the scope of a region whose expressions may be
// evaluated more than once per call of the enclosing function: a loop, or a
// query clause after the first. It tells ReferenceWorker that a single
// syntactic reference is not a single runtime reference.
func NewRepeatedRegion(parent Context) Context {
	return &repeatedRegionContext{contextBase: contextBase{outer: parent}, ruleBase: ruleBase{active: parent.rules()}}
}

// NewWorkerRegion pushes the scope of a worker region, started at the point
// where it is created. It records the named workers of the region, none of
// which has been referenced yet.
func NewWorkerRegion(parent Context, workers []*ast.BLangNamedWorkerDeclaration) Context {
	refs := make(map[model.SymbolRef]*ast.BLangVarRef, len(workers))
	for _, worker := range workers {
		refs[worker.Symbol()] = nil
	}
	return &workerRegionContext{contextBase: contextBase{outer: parent}, ruleBase: ruleBase{active: parent.rules()}, refs: refs}
}

// ReferenceWorker reports whether a reference to a worker name is valid. A
// worker may be referenced as a value at most once per execution of the
// region that declares it, so both a second reference and a reference that may
// execute repeatedly are rejected. A reference to anything other than a worker
// is valid.
func ReferenceWorker(cx Context, ref *ast.BLangVarRef) bool {
	sym := ref.Symbol()
	if cx.CompilerCtx().GetSymbol(sym).Kind() != model.SymbolKindWorker {
		return true
	}
	// A reference repeats if it sits in a loop or a closure between it and the
	// region's start point. A worker body is transparent: it runs exactly once
	// per execution of the region.
	repeatable := false
	for cur := cx; cur != nil; cur = cur.parent() {
		switch c := cur.(type) {
		case *repeatedRegionContext:
			repeatable = true
		case *functionContext:
			if c.kind != fnWorker {
				repeatable = true
			}
		case *workerRegionContext:
			seen, declaresWorker := c.refs[sym]
			if !declaresWorker {
				continue
			}
			if seen != nil {
				return invalidWorkerReference(cx, sym, ref, "is referenced more than once")
			}
			if repeatable {
				return invalidWorkerReference(cx, sym, ref, "may be evaluated more than once")
			}
			c.refs[sym] = ref
			return true
		}
	}
	// Sanity check: the walk above returns at the region's owner, and symbol
	// resolution only makes a worker name visible inside the region.
	cx.InternalErr("worker referenced outside the region that declares it", ref.GetPosition())
	return false
}

func invalidWorkerReference(cx Context, sym model.SymbolRef, ref *ast.BLangVarRef, reason string) bool {
	cx.SemanticErr("worker '"+cx.CompilerCtx().SymbolName(sym)+"' "+reason, ref.GetPosition())
	return false
}

func innermostFunction(cx Context) (*functionContext, bool) {
	for cur := cx; cur != nil; cur = cur.parent() {
		if fc, ok := cur.(*functionContext); ok {
			return fc, true
		}
	}
	return nil, false
}
