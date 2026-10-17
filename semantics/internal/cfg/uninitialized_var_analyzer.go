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
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/common/bitset"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
)

// uninitVarAnalyzer performs data flow analysis for uninitialized variables
type uninitVarAnalyzer struct {
	ctx               *context.CompilerContext
	fn                ast.BLangNode
	fcfg              *functionCFG
	vars              *bitset.Context[model.SymbolRef]
	states            *initStateStore
	implicitInitState *initState // vars initialized by language constructs, applied at every block entry
}

// newUninitVarAnalyzer creates a new analyzer for a function
func newUninitVarAnalyzer(ctx *context.CompilerContext, fn ast.BLangNode, fcfg *functionCFG) *uninitVarAnalyzer {
	vars := bitset.NewContext[model.SymbolRef]()
	implicitInitState := buildImplicitInitState(fn, vars)
	return &uninitVarAnalyzer{
		ctx:  ctx,
		fn:   fn,
		fcfg: fcfg,
		vars: vars,
		states: newInitStateStore(fcfg, func() *initState {
			return &initState{tracked: vars.New(), init: vars.New()}
		}),
		implicitInitState: implicitInitState,
	}
}

func buildImplicitInitState(fn ast.BLangNode, vars *bitset.Context[model.SymbolRef]) *initState {
	state := &initState{tracked: bitset.BitSet{}, init: bitset.BitSet{}}
	ast.Walk(&implicitVarMarker{vars: vars, state: state}, fn)
	return state
}

type implicitVarMarker struct {
	vars  *bitset.Context[model.SymbolRef]
	state *initState
}

func (m *implicitVarMarker) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if foreach, ok := node.(*ast.BLangForeach); ok && foreach.VariableDef != nil {
		m.state.markInitialized(m.vars.Intern(foreach.VariableDef.Var.Symbol()))
	}
	return m
}

func (m *implicitVarMarker) VisitTypeData(*ast.TypeData) ast.Visitor { return m }

func (a *uninitVarAnalyzer) analyze() {
	if len(a.fcfg.bbs) == 0 {
		return
	}
	for _, i := range a.fcfg.topoOrder {
		bb := &a.fcfg.bbs[i]
		state := a.states.entryState(bb)
		state.markAllInitialized(a.implicitInitState)
		for _, node := range bb.nodes {
			a.analyzeNode(node, state)
		}
		a.states.setExit(bb, state)
	}
}

// analyzeNode processes a single node in the CFG
func (a *uninitVarAnalyzer) analyzeNode(node ast.Node, state *initState) {
	switch n := node.(type) {
	case *ast.BLangVariableDef:
		p := a.vars.Intern(n.Var.Symbol())
		if n.Var.Expr != nil {
			a.checkExpression(n.Var.Expr, state)
			state.markInitialized(p)
		} else {
			state.markTracked(p)
		}
	case *ast.BLangAssignment:
		a.checkExpression(n.Expr, state)
		a.markAssignmentTarget(n.VarRef, state)
	case *ast.BLangCompoundAssignment:
		a.checkExpression(n.VarRef, state)
		a.checkExpression(n.Expr, state)
		a.markAssignmentTarget(n.VarRef, state)
	case ast.BLangExpression:
		// Expression nodes (like conditions in while loops) need to be checked
		a.checkExpression(n, state)
	case ast.BLangNode:
		checker := &varRefChecker{analyzer: a, state: state}
		ast.Walk(checker, n)
	default:
		a.ctx.InternalError("unexpected node", node.GetPosition())
	}
}

// markAssignmentTarget marks the target of an assignment as initialized
func (a *uninitVarAnalyzer) markAssignmentTarget(expr ast.BLangExpression, state *initState) {
	// Check for wildcard pattern - no tracking needed
	if _, ok := expr.(*ast.BLangWildCardBindingPattern); ok {
		return
	}

	// For simple variable references, mark as initialized
	if nodeWithSymbol, ok := expr.(ast.NodeWithSymbol); ok {
		if p, ok := a.vars.Lookup(nodeWithSymbol.Symbol()); ok && state.isTracked(p) {
			state.markInitialized(p)
		}
		return
	}

	// For complex expressions (index access, field access), check the expression
	// but don't mark anything as initialized - these modify existing data
	a.checkExpression(expr, state)
}

// checkExpression walks an expression and checks all variable references
func (a *uninitVarAnalyzer) checkExpression(expr ast.BLangActionOrExpression, state *initState) {
	checker := &varRefChecker{analyzer: a, state: state}
	ast.Walk(checker, expr)
}

// checkVariableReference checks if a variable is initialized before use
func (a *uninitVarAnalyzer) checkVariableReference(symRef model.SymbolRef, node ast.Node, state *initState) {
	p, ok := a.vars.Lookup(symRef)
	if !ok || !state.isTracked(p) {
		return
	}

	if !state.isInitialized(p) {
		// Variable is tracked but not initialized - report error
		a.ctx.SemanticError("variable may not be initialized", node.GetPosition())
	}
}

// varRefChecker is a visitor that checks variable references in expressions
type varRefChecker struct {
	analyzer *uninitVarAnalyzer
	state    *initState
}

var _ ast.Visitor = &varRefChecker{}

func (v *varRefChecker) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}

	// Check if this node is a variable reference
	if inv, ok := node.(*ast.BLangInvocation); ok && ast.IsStreamOperation(inv) {
		return nil
	}
	if nodeWithSymbol, ok := node.(ast.NodeWithSymbol); ok {
		v.analyzer.checkVariableReference(nodeWithSymbol.Symbol(), node, v.state)
	}

	return v
}

func (v *varRefChecker) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	// TypeDesc could have default values
	return v
}

// analyzeUninitializedVars is the public entry point for uninitialized variable analysis
func analyzeUninitializedVars(ctx *context.CompilerContext, pkg *ast.BLangPackage, cfg *PackageCFG) {
	var wg sync.WaitGroup
	for _, fn := range common.PackageFunctionDecls(pkg) {
		wg.Add(1)
		go func(fn common.FunctionDecl) {
			defer wg.Done()
			analyzeFunctionUninitializedVars(ctx, fn, cfg)
		}(fn)
	}
	for i := range cfg.lambdaCfgs {
		lambda := &cfg.lambdaCfgs[i]
		wg.Go(func() { newUninitVarAnalyzer(ctx, lambda.fn, &lambda.cfg).analyze() })
	}
	wg.Wait()
}

// analyzeFunctionUninitializedVars analyzes a single function for uninitialized variables
func analyzeFunctionUninitializedVars(ctx *context.CompilerContext, fn common.FunctionDecl, cfg *PackageCFG) {
	fnCfg, ok := cfg.lookupFunctionCfg(fn.Symbol())
	if !ok {
		return
	}

	analyzer := newUninitVarAnalyzer(ctx, fn, &fnCfg)
	analyzer.analyze()
}
