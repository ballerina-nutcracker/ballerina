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
	"slices"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/common/bitset"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
)

// uninitVarAnalyzer performs definite-assignment analysis on one function body. It tracks every variable the body can
// reference: the body's own locals, and the variables in scope when the body starts (module variables, `self` and,
// for an anonymous function or a named worker, the variables of the body that defines it).
type uninitVarAnalyzer struct {
	ctx      *context.CompilerContext
	body     *body
	vars     *bitset.Context[model.SymbolRef] // bit position of each tracked variable in every state
	states   *initStateStore
	loopVars map[*ast.BLangVariableDef]bool // a foreach binds its loop variable, which has no initializer
}

// body is a function, anonymous function or named worker body, with the anonymous function and named worker bodies
// it defines. A child is analyzed where its parent defines it, so only a parent can reach it.
type body struct {
	cfg      *functionCFG
	children map[ast.BLangNode]*body // keyed by the child's *ast.BLangFunction or *ast.BLangNamedWorkerDeclaration
}

// bodyGraphs holds the graphs of every anonymous function and named worker body in a package.
type bodyGraphs struct {
	lambdas map[*ast.BLangFunction]*functionCFG
	workers map[model.SymbolRef]workerCFG
}

func newBody(fcfg *functionCFG, graphs *bodyGraphs) *body {
	b := &body{cfg: fcfg, children: make(map[ast.BLangNode]*body)}
	collector := &childCollector{graphs: graphs, parent: b}
	for _, bb := range fcfg.bbs {
		for _, node := range bb.nodes {
			ast.Walk(collector, node.(ast.BLangNode))
		}
	}
	return b
}

// childCollector finds the anonymous functions and named workers a body defines. The signature of an anonymous
// function and the return type of a named worker are evaluated in the body that defines it, so an anonymous function
// within them is also a child.
type childCollector struct {
	graphs *bodyGraphs
	parent *body
}

func (c *childCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case nil:
		return nil
	case *ast.BLangLambdaFunction:
		c.parent.children[n.Function] = newBody(c.graphs.lambdas[n.Function], c.graphs)
		walkSignature(c, n.Function)
		return nil
	case *ast.BLangNamedWorkerDeclaration:
		worker := c.graphs.workers[n.Symbol()]
		c.parent.children[n] = newBody(&worker.cfg, c.graphs)
		if n.ReturnType != nil {
			ast.Walk(c, n.ReturnType)
		}
		return nil
	}
	return c
}

func (c *childCollector) VisitTypeData(*ast.TypeData) ast.Visitor { return c }

// moduleLevelLambda is an anonymous function outside every function body, with the variables bound by the queries
// that enclose it.
type moduleLevelLambda struct {
	fn        *ast.BLangFunction
	queryVars []model.SymbolRef
}

// moduleLevelLambdaCollector finds the anonymous functions outside every function body, such as in a module
// variable's initializer or a parameter default.
type moduleLevelLambdaCollector struct {
	lambdas   *[]moduleLevelLambda
	queryVars []model.SymbolRef // variables bound by the queries enclosing the current node
}

func (c *moduleLevelLambdaCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case nil, *ast.BLangBlockFunctionBody, *ast.BLangExprFunctionBody:
		return nil
	case *ast.BLangLambdaFunction:
		*c.lambdas = append(*c.lambdas, moduleLevelLambda{fn: n.Function, queryVars: c.queryVars})
	case *ast.BLangQueryExpr:
		// A query clause binds its variable before any expression of a later clause, including an anonymous
		// function, is evaluated, so the function can read every variable of the query.
		vars := &queryVarCollector{vars: slices.Clone(c.queryVars)}
		ast.Walk(vars, n)
		return &moduleLevelLambdaCollector{lambdas: c.lambdas, queryVars: vars.vars}
	}
	return c
}

func (c *moduleLevelLambdaCollector) VisitTypeData(*ast.TypeData) ast.Visitor { return c }

// queryVarCollector finds the variables a query binds, outside the bodies of the anonymous functions within it.
type queryVarCollector struct {
	vars []model.SymbolRef
}

func (c *queryVarCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case nil, *ast.BLangBlockFunctionBody, *ast.BLangExprFunctionBody:
		return nil
	case *ast.BLangVariableDef:
		c.vars = append(c.vars, n.Var.Symbol())
	}
	return c
}

func (c *queryVarCollector) VisitTypeData(*ast.TypeData) ast.Visitor { return c }

// walkSignature walks the parameter types, parameter defaults and return type of an anonymous function, which are
// evaluated in the body that defines it.
func walkSignature(v ast.Visitor, fn *ast.BLangFunction) {
	for i := range fn.RequiredParams {
		walkParam(v, &fn.RequiredParams[i])
	}
	if fn.RestParam != nil {
		walkParam(v, fn.RestParam)
	}
	if rt := fn.ReturnType(); rt != nil {
		ast.Walk(v, rt.(ast.BLangNode))
	}
}

func walkParam(v ast.Visitor, param *ast.BLangVariable) {
	if tn := param.TypeNode(); tn != nil {
		ast.Walk(v, tn.(ast.BLangNode))
	}
	if param.Expr != nil {
		ast.Walk(v, param.Expr)
	}
}

type loopVarCollector struct {
	loopVars map[*ast.BLangVariableDef]bool
}

func (c *loopVarCollector) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if foreach, ok := node.(*ast.BLangForeach); ok && foreach.VariableDef != nil {
		c.loopVars[foreach.VariableDef] = true
	}
	return c
}

func (c *loopVarCollector) VisitTypeData(*ast.TypeData) ast.Visitor { return c }

// newUninitVarAnalyzer returns an analyzer for b, which starts with the variables in entry.
func newUninitVarAnalyzer(
	ctx *context.CompilerContext, b *body, vars *bitset.Context[model.SymbolRef], entry *initState,
	loopVars map[*ast.BLangVariableDef]bool) *uninitVarAnalyzer {
	return &uninitVarAnalyzer{
		ctx:  ctx,
		body: b,
		vars: vars,
		states: newInitStateStore(b.cfg, entry, func() *initState {
			return &initState{tracked: vars.New(), init: vars.New()}
		}),
		loopVars: loopVars,
	}
}

// analyze checks the blocks execution can reach: the entry block and the blocks with a reachable forward predecessor.
// Unreachable code is already an error, so its blocks are not checked and leave an empty state, which merges ignore.
func (a *uninitVarAnalyzer) analyze() {
	fcfg := a.body.cfg
	if len(fcfg.bbs) == 0 {
		return
	}
	live := make([]bool, len(fcfg.bbs))
	for _, i := range fcfg.topoOrder {
		bb := &fcfg.bbs[i]
		state := a.states.entryState(bb)
		live[i] = i == 0 || slices.ContainsFunc(bb.parents, func(p int) bool {
			return live[p] && !slices.Contains(bb.backedgeParents, p)
		})
		if live[i] {
			for _, node := range bb.nodes {
				a.analyzeNode(node, state)
			}
		}
		a.states.setExit(bb, state)
	}
}

// analyzeNode processes a single node in the CFG
func (a *uninitVarAnalyzer) analyzeNode(node ast.Node, state *initState) {
	switch n := node.(type) {
	case *ast.BLangVariableDef:
		// A declared type can hold defaults, which are evaluated here
		if tn := n.Var.TypeNode(); tn != nil {
			ast.Walk(&varRefChecker{analyzer: a, state: state}, tn.(ast.BLangNode))
		}
		p := a.vars.Intern(n.Var.Symbol())
		switch {
		case n.Var.Expr != nil:
			a.checkExpression(n.Var.Expr, state)
			state.markInitialized(p)
		case a.loopVars[n]:
			state.markInitialized(p)
		default:
			state.markTracked(p)
		}
	case *ast.BLangAssignment:
		a.checkExpression(n.Expr, state)
		a.markAssignmentTarget(n.VarRef, state)
	case *ast.BLangCompoundAssignment:
		a.checkExpression(n.VarRef, state)
		a.checkExpression(n.Expr, state)
		a.markAssignmentTarget(n.VarRef, state)
	case *ast.BLangNamedWorkerDeclaration:
		// The return type is evaluated in this body; the worker's own body is analyzed as a child
		if n.ReturnType != nil {
			ast.Walk(&varRefChecker{analyzer: a, state: state}, n.ReturnType)
		}
		a.analyzeChild(n, state)
	case ast.BLangExpression:
		// Expression nodes (like conditions in while loops) need to be checked
		a.checkExpression(n, state)
	case ast.BLangNode:
		ast.Walk(&varRefChecker{analyzer: a, state: state}, n)
	default:
		a.ctx.InternalError("unexpected node", node.GetPosition())
	}
}

// analyzeChild analyzes the body of an anonymous function or named worker defined at this point, starting from a
// copy of state: a variable the body captures is initialized in the body exactly when it is initialized here.
func (a *uninitVarAnalyzer) analyzeChild(key ast.BLangNode, state *initState) {
	entry := &initState{tracked: a.vars.New(), init: a.vars.New()}
	entry.copyFrom(state)
	newUninitVarAnalyzer(a.ctx, a.body.children[key], a.vars, entry, a.loopVars).analyze()
}

// markAssignmentTarget marks the target of an assignment as initialized
func (a *uninitVarAnalyzer) markAssignmentTarget(expr ast.BLangExpression, state *initState) {
	// Check for wildcard pattern - no tracking needed
	if _, ok := expr.(*ast.BLangWildCardBindingPattern); ok {
		return
	}

	// For simple variable references, mark as initialized
	if nodeWithSymbol, ok := expr.(ast.NodeWithSymbol); ok {
		ref := a.ctx.UnnarrowedSymbol(nodeWithSymbol.Symbol())
		if p, ok := a.vars.Lookup(ref); ok && state.isTracked(p) {
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
	ast.Walk(&varRefChecker{analyzer: a, state: state}, expr)
}

// checkVariableReference reports a reference to a variable that may not be initialized. A narrowed symbol is checked
// as the variable it narrows. Parameters are initialized when the body starts, a reference with a module prefix is
// to a variable its module initialized before this one, and other symbols are not variables, so none of these is
// checked. Every other reference must be to a tracked variable.
func (a *uninitVarAnalyzer) checkVariableReference(symRef model.SymbolRef, node ast.Node, state *initState) {
	var alias ast.IdentifierNode
	switch node := node.(type) {
	case *ast.BLangVarRef:
		alias = node.PkgAlias
	case *ast.BLangInvocation:
		alias = node.PkgAlias
	}
	if alias != nil && alias.GetValue() != "" {
		return
	}
	ref := a.ctx.UnnarrowedSymbol(symRef)
	if a.ctx.SymbolKind(ref) != model.SymbolKindVariable {
		return
	}
	p, ok := a.vars.Lookup(ref)
	if !ok || !state.isTracked(p) {
		a.ctx.InternalError("reference to untracked variable '"+a.ctx.SymbolName(ref)+"'", node.GetPosition())
		return
	}
	if !state.isInitialized(p) {
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
	switch n := node.(type) {
	case *ast.BLangInvocation:
		// A stream operation, such as s.next(), has no function symbol. Its receiver and arguments are not checked
		// either.
		if ast.IsStreamOperation(n) {
			return nil
		}
	case *ast.BLangLambdaFunction:
		// The signature is evaluated in this body; the function's own body is analyzed as a child
		walkSignature(v, n.Function)
		v.analyzer.analyzeChild(n.Function, v.state)
		return nil
	case *ast.BLangFunctionTypeParam:
		// A parameter of a function type descriptor declares no variable, but its default is checked here
		return v
	case *ast.BLangVariableDef:
		// The only declarations within an expression are query variables, each bound by its clause: a from or join
		// variable, which has no initializer, to each member of the collection, and a let or group by key variable
		// to its initializer. They are initialized here; binding patterns are not yet handled separately. A declared
		// type can hold defaults, which are evaluated here too.
		if tn := n.Var.TypeNode(); tn != nil {
			ast.Walk(v, tn.(ast.BLangNode))
		}
		if n.Var.Expr != nil {
			ast.Walk(v, n.Var.Expr)
		}
		v.state.markInitialized(v.analyzer.vars.Intern(n.Var.Symbol()))
		return nil
	case *ast.BLangVariable:
		v.analyzer.ctx.InternalError("unexpected variable declaration in an expression", n.GetPosition())
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
func analyzeUninitializedVars(
	ctx *context.CompilerContext, pkg *ast.BLangPackage, cfg *PackageCFG, parent context.TraceSpan) {
	span := parent.StartChild("Uninitialized Variable Analysis", "")
	defer span.End()
	graphs := &bodyGraphs{lambdas: make(map[*ast.BLangFunction]*functionCFG), workers: cfg.workers}
	for i := range cfg.lambdaCfgs {
		graphs.lambdas[cfg.lambdaCfgs[i].fn] = &cfg.lambdaCfgs[i].cfg
	}
	loopVars := make(map[*ast.BLangVariableDef]bool)
	ast.Walk(&loopVarCollector{loopVars: loopVars}, pkg)
	var wg sync.WaitGroup
	for _, fn := range common.PackageFunctionDecls(pkg) {
		wg.Go(func() {
			analyzeFunctionUninitializedVars(ctx, pkg, fn, cfg, graphs, loopVars, span)
		})
	}
	// An anonymous function outside every function body starts with the module variables in scope
	var moduleLambdas []moduleLevelLambda
	ast.Walk(&moduleLevelLambdaCollector{lambdas: &moduleLambdas}, pkg)
	for _, lambda := range moduleLambdas {
		wg.Go(func() {
			vars := bitset.NewContext[model.SymbolRef]()
			entry := moduleScopeState(pkg, vars)
			for _, sym := range lambda.queryVars {
				entry.markInitialized(vars.Intern(sym))
			}
			b := newBody(graphs.lambdas[lambda.fn], graphs)
			newUninitVarAnalyzer(ctx, b, vars, entry, loopVars).analyze()
		})
	}
	wg.Wait()
}

// moduleScopeState returns the state in which every module variable is initialized.
func moduleScopeState(pkg *ast.BLangPackage, vars *bitset.Context[model.SymbolRef]) *initState {
	state := &initState{tracked: vars.New(), init: vars.New()}
	for _, v := range pkg.GlobalVars {
		state.markInitialized(vars.Intern(v.Symbol()))
	}
	return state
}

// moduleInitState returns the state in which the module's init function starts: a module variable with an
// initializer is initialized, and one without is declared but not yet initialized.
func moduleInitState(pkg *ast.BLangPackage, vars *bitset.Context[model.SymbolRef]) *initState {
	state := &initState{tracked: vars.New(), init: vars.New()}
	for _, v := range pkg.GlobalVars {
		if v.Expr == nil {
			state.markTracked(vars.Intern(v.Symbol()))
		} else {
			state.markInitialized(vars.Intern(v.Symbol()))
		}
	}
	return state
}

// analyzeFunctionUninitializedVars analyzes a single function for uninitialized variables
func analyzeFunctionUninitializedVars(
	ctx *context.CompilerContext, pkg *ast.BLangPackage, fn common.FunctionDecl, cfg *PackageCFG,
	graphs *bodyGraphs, loopVars map[*ast.BLangVariableDef]bool, parent context.TraceSpan) {
	span := parent.StartChild("Uninitialized Variables", fn.GetName().GetValue())
	defer span.End()
	fnCfg, ok := cfg.lookupFunctionCfg(fn.Symbol())
	if !ok {
		return
	}
	vars := bitset.NewContext[model.SymbolRef]()
	var entry *initState
	if pkg.InitFunction != nil && fn.Symbol() == pkg.InitFunction.Symbol() {
		entry = moduleInitState(pkg, vars)
	} else {
		entry = moduleScopeState(pkg, vars)
	}
	if self, ok := fn.Scope().GetSymbol("self"); ok {
		entry.markInitialized(vars.Intern(self))
	}
	newUninitVarAnalyzer(ctx, newBody(&fnCfg, graphs), vars, entry, loopVars).analyze()
}
