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

package semantics

import (
	"maps"
	"sync"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
)

// varInitState tracks which variables are definitely initialized
type varInitState struct {
	initVars map[model.SymbolRef]bool // true = definitely initialized
}

// newVarInitState creates an empty initialization state
func newVarInitState() *varInitState {
	return &varInitState{
		initVars: make(map[model.SymbolRef]bool),
	}
}

// clone creates a deep copy of the state
func (s *varInitState) clone() *varInitState {
	newState := newVarInitState()
	maps.Copy(newState.initVars, s.initVars)
	return newState
}

func mergeStates(s1, s2 *varInitState) *varInitState {
	// If one state is empty (unvisited block), use the other state
	// This handles the fixed-point iteration where some blocks haven't been analyzed yet
	if len(s1.initVars) == 0 {
		return s2.clone()
	}
	if len(s2.initVars) == 0 {
		return s1.clone()
	}

	result := newVarInitState()

	// Merge variables from both states
	// Collect all variables from both states
	allVars := make(map[model.SymbolRef]bool)
	for symRef := range s1.initVars {
		allVars[symRef] = true
	}
	for symRef := range s2.initVars {
		allVars[symRef] = true
	}

	// For each variable, it's initialized only if initialized in ALL paths
	for symRef := range allVars {
		init1, exists1 := s1.initVars[symRef]
		init2, exists2 := s2.initVars[symRef]

		if exists1 && exists2 {
			// Variable in both states - initialized if both are initialized
			result.initVars[symRef] = init1 && init2
		} else {
			// Variable in one state but not the other - treat as uninitialized
			// This handles variables that may be in different scopes
			result.initVars[symRef] = false
		}
	}

	return result
}

// markInitialized marks a variable as definitely initialized
func (s *varInitState) markInitialized(symRef model.SymbolRef) {
	s.initVars[symRef] = true
}

// markUninitialized marks a variable as declared but not initialized
func (s *varInitState) markUninitialized(symRef model.SymbolRef) {
	s.initVars[symRef] = false
}

// isTracked returns true if we're tracking this variable
func (s *varInitState) isTracked(symRef model.SymbolRef) bool {
	_, exists := s.initVars[symRef]
	return exists
}

// isInitialized returns true if the variable is definitely initialized
func (s *varInitState) isInitialized(symRef model.SymbolRef) bool {
	return s.initVars[symRef]
}

// blockState tracks initialization state at entry and exit of a basic block
type blockState struct {
	entry *varInitState // State when entering block
	exit  *varInitState // State when exiting block
}

// uninitVarAnalyzer performs data flow analysis for uninitialized variables
type uninitVarAnalyzer struct {
	ctx               *context.CompilerContext
	fn                ast.BLangNode
	fcfg              *functionCFG
	states            map[int]*blockState
	implicitInitState *varInitState // vars initialized by language constructs, used as entry state baseline
}

// newUninitVarAnalyzer creates a new analyzer for a function
func newUninitVarAnalyzer(ctx *context.CompilerContext, fn ast.BLangNode, fcfg *functionCFG) *uninitVarAnalyzer {
	analyzer := &uninitVarAnalyzer{
		ctx:               ctx,
		fn:                fn,
		fcfg:              fcfg,
		states:            make(map[int]*blockState),
		implicitInitState: buildImplicitInitState(fn),
	}

	// Initialize block states
	for i := range fcfg.bbs {
		analyzer.states[i] = &blockState{
			entry: newVarInitState(),
			exit:  newVarInitState(),
		}
	}

	return analyzer
}

func buildImplicitInitState(fn ast.BLangNode) *varInitState {
	state := newVarInitState()
	ast.Walk(&implicitVarMarker{state: state}, fn)
	return state
}

type implicitVarMarker struct {
	state *varInitState
}

func (m *implicitVarMarker) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if foreach, ok := node.(*ast.BLangForeach); ok && foreach.VariableDef != nil {
		m.state.markInitialized(foreach.VariableDef.Var.Symbol())
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
		entry := a.mergePredecessors(bb)
		a.states[i].entry = entry
		exit := a.analyzeBlock(bb, entry.clone())
		a.states[i].exit = exit
	}
}

// mergePredecessors merges the exit states of all non-backedge predecessors.
func (a *uninitVarAnalyzer) mergePredecessors(bb *basicBlock) *varInitState {
	backedgeSet := make(map[int]bool, len(bb.backedgeParents))
	for _, p := range bb.backedgeParents {
		backedgeSet[p] = true
	}
	result := (*varInitState)(nil)
	for _, parentID := range bb.parents {
		if backedgeSet[parentID] {
			continue
		}
		if result == nil {
			result = a.states[parentID].exit.clone()
		} else {
			result = mergeStates(result, a.states[parentID].exit)
		}
	}
	if result == nil {
		result = newVarInitState()
	}
	for symRef := range a.implicitInitState.initVars {
		result.markInitialized(symRef)
	}
	return result
}

// analyzeBlock performs intra-block analysis
func (a *uninitVarAnalyzer) analyzeBlock(bb *basicBlock, state *varInitState) *varInitState {
	for _, node := range bb.nodes {
		a.analyzeNode(node, state)
	}
	return state
}

// analyzeNode processes a single node in the CFG
func (a *uninitVarAnalyzer) analyzeNode(node ast.Node, state *varInitState) {
	switch n := node.(type) {
	case *ast.BLangSimpleVariableDef:
		symRef := n.Var.Symbol()
		if n.Var.Expr != nil {
			a.checkExpression(n.Var.Expr, state)
			state.markInitialized(symRef)
		} else if !state.isInitialized(symRef) {
			state.markUninitialized(symRef)
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
func (a *uninitVarAnalyzer) markAssignmentTarget(expr ast.BLangExpression, state *varInitState) {
	// Check for wildcard pattern - no tracking needed
	if _, ok := expr.(*ast.BLangWildCardBindingPattern); ok {
		return
	}

	// For simple variable references, mark as initialized
	if nodeWithSymbol, ok := expr.(ast.NodeWithSymbol); ok {
		symRef := nodeWithSymbol.Symbol()
		if state.isTracked(symRef) {
			state.markInitialized(symRef)
		}
		return
	}

	// For complex expressions (index access, field access), check the expression
	// but don't mark anything as initialized - these modify existing data
	a.checkExpression(expr, state)
}

// checkExpression walks an expression and checks all variable references
func (a *uninitVarAnalyzer) checkExpression(expr ast.BLangActionOrExpression, state *varInitState) {
	checker := &varRefChecker{analyzer: a, state: state}
	ast.Walk(checker, expr)
}

// checkVariableReference checks if a variable is initialized before use
func (a *uninitVarAnalyzer) checkVariableReference(symRef model.SymbolRef, node ast.Node, state *varInitState) {
	if !state.isTracked(symRef) {
		return
	}

	if !state.isInitialized(symRef) {
		// Variable is tracked but not initialized - report error
		a.ctx.SemanticError("variable may not be initialized", node.GetPosition())
	}
}

// varRefChecker is a visitor that checks variable references in expressions
type varRefChecker struct {
	analyzer *uninitVarAnalyzer
	state    *varInitState
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
	for _, fn := range packageFunctionDecls(pkg) {
		wg.Add(1)
		go func(fn functionDecl) {
			defer wg.Done()
			analyzeFunctionUninitializedVars(ctx, fn, cfg)
		}(fn)
	}
	wg.Wait()
}

// analyzeFunctionUninitializedVars analyzes a single function for uninitialized variables
func analyzeFunctionUninitializedVars(ctx *context.CompilerContext, fn functionDecl, cfg *PackageCFG) {
	fnCfg, ok := cfg.lookupFunctionCfg(fn.Symbol())
	if !ok {
		return
	}

	analyzer := newUninitVarAnalyzer(ctx, fn, &fnCfg)
	analyzer.analyze()
}
