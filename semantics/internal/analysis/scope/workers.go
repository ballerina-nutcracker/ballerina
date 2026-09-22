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

// DeclareWorkers records the named workers declared in the body of the
// innermost function, none of which has been referenced yet.
func DeclareWorkers(cx Context, workers []*ast.BLangNamedWorkerDeclaration) {
	fc, ok := innermostFunction(cx)
	if !ok || fc.kind == fnWorker {
		// Sanity check: named workers are only declared while validating a
		// function body, so a non-worker function context always encloses them.
		cx.InternalErr("named workers outside a function body", workers[0].GetPosition())
		return
	}
	fc.workerRefs = make(map[model.SymbolRef]*ast.BLangVarRef, len(workers))
	for _, worker := range workers {
		fc.workerRefs[worker.Symbol()] = nil
	}
}

// ReferenceWorker reports whether a reference to a worker name is valid. A
// worker may be referenced as a value at most once per execution of the
// function body that declares it, so both a second reference and a reference
// that may execute repeatedly are rejected. A reference to anything other
// than a worker is valid.
func ReferenceWorker(cx Context, ref *ast.BLangVarRef) bool {
	sym := ref.Symbol()
	if cx.CompilerCtx().GetSymbol(sym).Kind() != model.SymbolKindWorker {
		return true
	}
	// A reference repeats if it sits in a region that runs many times per call,
	// or in a closure over a worker declared further out. A worker body is
	// transparent: it runs exactly once per call of the declaring function.
	repeatable := false
	for cur := cx; cur != nil; cur = cur.parent() {
		switch c := cur.(type) {
		case *repeatedRegionContext:
			repeatable = true
		case *functionContext:
			if c.kind == fnWorker {
				continue
			}
			seen, declaresWorker := c.workerRefs[sym]
			if !declaresWorker {
				repeatable = true
				continue
			}
			if seen != nil {
				return invalidWorkerReference(cx, sym, ref, "is referenced more than once")
			}
			if repeatable {
				return invalidWorkerReference(cx, sym, ref, "may be evaluated more than once")
			}
			c.workerRefs[sym] = ref
			return true
		}
	}
	// Sanity check: the walk above returns at the function that declares the
	// worker, and symbol resolution only makes a worker name visible there.
	cx.InternalErr("worker referenced outside the function that declares it", ref.GetPosition())
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
