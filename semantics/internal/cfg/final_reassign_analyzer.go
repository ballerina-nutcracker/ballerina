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

package cfg

import (
	"maps"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
)

// possiblyAssignedState is the set of tracked variables that are assigned on
// at least one path reaching a program point.
type possiblyAssignedState map[model.SymbolRef]bool

// possiblyAssignedAnalyzer finds assignments to final variables declared
// without an initializer that may already hold a value.
type possiblyAssignedAnalyzer struct {
	ctx     *context.CompilerContext
	fcfg    *functionCFG
	tracked map[model.SymbolRef]bool
	exits   []possiblyAssignedState
	// reachable marks blocks reachable from the entry block, so assignments
	// in unreachable code do not flow into reachable code.
	reachable []bool
}

func newPossiblyAssignedAnalyzer(ctx *context.CompilerContext, fcfg *functionCFG, tracked map[model.SymbolRef]bool) *possiblyAssignedAnalyzer {
	exits := make([]possiblyAssignedState, len(fcfg.bbs))
	for i := range exits {
		exits[i] = possiblyAssignedState{}
	}
	return &possiblyAssignedAnalyzer{
		ctx:       ctx,
		fcfg:      fcfg,
		tracked:   tracked,
		exits:     exits,
		reachable: reachableFromEntry(fcfg),
	}
}

func reachableFromEntry(fcfg *functionCFG) []bool {
	reachable := make([]bool, len(fcfg.bbs))
	if len(fcfg.bbs) == 0 {
		return reachable
	}
	worklist := []int{0}
	reachable[0] = true
	for len(worklist) > 0 {
		id := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		for _, child := range fcfg.bbs[id].children {
			if !reachable[child] {
				reachable[child] = true
				worklist = append(worklist, child)
			}
		}
	}
	return reachable
}

// analyzeFinalReassignments reports assignments to final variables declared
// without an initializer when the variable is possibly assigned already.
// Locals are checked in every function; final module variables are checked in
// the module init function.
func analyzeFinalReassignments(ctx *context.CompilerContext, pkg *ast.BLangPackage, cfg *PackageCFG) {
	moduleFinals := deferredFinalModuleVars(pkg)
	var wg sync.WaitGroup
	for _, fn := range common.PackageFunctionDecls(pkg) {
		wg.Go(func() {
			fnCfg, ok := cfg.lookupFunctionCfg(fn.Symbol())
			if !ok {
				return
			}
			tracked := deferredFinalLocals(ctx, fn)
			if pkg.InitFunction != nil && fn.Symbol() == pkg.InitFunction.Symbol() {
				maps.Copy(tracked, moduleFinals)
			}
			if len(tracked) == 0 {
				return
			}
			newPossiblyAssignedAnalyzer(ctx, &fnCfg, tracked).analyze()
		})
	}
	wg.Wait()
}

func deferredFinalModuleVars(pkg *ast.BLangPackage) map[model.SymbolRef]bool {
	result := make(map[model.SymbolRef]bool)
	for i := range pkg.GlobalVars {
		v := pkg.GlobalVars[i]
		if v.Expr == nil && v.IsFinal() {
			result[v.Symbol()] = true
		}
	}
	return result
}

func deferredFinalLocals(ctx *context.CompilerContext, fn ast.BLangNode) map[model.SymbolRef]bool {
	collector := &deferredFinalCollector{ctx: ctx, result: make(map[model.SymbolRef]bool)}
	ast.Walk(collector, fn)
	return collector.result
}

type deferredFinalCollector struct {
	ctx    *context.CompilerContext
	result map[model.SymbolRef]bool
}

func (c *deferredFinalCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case nil, *ast.BLangLambdaFunction:
		return nil
	case *ast.BLangVariableDef:
		if n.Var.Expr != nil {
			return c
		}
		if meta, ok := c.ctx.ValueSymbolMetadata(n.Var.Symbol()); ok && meta.Final {
			c.result[n.Var.Symbol()] = true
		}
	}
	return c
}

func (c *deferredFinalCollector) VisitTypeData(*ast.TypeData) ast.Visitor { return c }

func (a *possiblyAssignedAnalyzer) analyze() {
	for changed := true; changed; {
		changed = false
		for _, i := range a.fcfg.topoOrder {
			if !a.reachable[i] {
				continue
			}
			bb := &a.fcfg.bbs[i]
			state := a.entryState(bb)
			for _, node := range bb.nodes {
				a.transfer(node, state, false)
			}
			if !maps.Equal(state, a.exits[i]) {
				a.exits[i] = state
				changed = true
			}
		}
	}
	for _, i := range a.fcfg.topoOrder {
		if !a.reachable[i] {
			continue
		}
		bb := &a.fcfg.bbs[i]
		state := a.entryState(bb)
		for _, node := range bb.nodes {
			a.transfer(node, state, true)
		}
	}
}

// entryState merges the exit states of all parents, back-edges included, so
// an assignment made in a previous loop iteration is visible.
func (a *possiblyAssignedAnalyzer) entryState(bb *basicBlock) possiblyAssignedState {
	state := possiblyAssignedState{}
	for _, parentID := range bb.parents {
		maps.Copy(state, a.exits[parentID])
	}
	return state
}

func (a *possiblyAssignedAnalyzer) transfer(node ast.Node, state possiblyAssignedState, reportErrors bool) {
	switch n := node.(type) {
	case *ast.BLangVariableDef:
		delete(state, n.Var.Symbol())
	case *ast.BLangAssignment:
		varRef, ok := n.VarRef.(*ast.BLangVarRef)
		if !ok || !a.tracked[varRef.Symbol()] {
			return
		}
		if state[varRef.Symbol()] && reportErrors {
			a.ctx.SemanticError("cannot assign a value to potentially initialized final '"+varRef.VariableName.GetValue()+"'", varRef.GetPosition())
		}
		state[varRef.Symbol()] = true
	}
}
