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
	"slices"
	"sync"

	"ballerina/ast"
	"ballerina/context"
	"ballerina/model"
	"ballerina/semtypes"
)

type basicBlock struct {
	// id uniquely identify a basic block within function scope. Root bb will have id 0.
	id int
	// bbs from which you can reach this block. If this is empty and it is not a root then this block is not
	// reachable
	parents []int
	// subset of parents; each entry is also in parents. These are the back edges of the CFG (e.g. loop-back edges).
	backedgeParents []int
	// bbs to which you can transition from this block. For blocks with conditional jumps this will 2. For blocks
	// that don't terminate normally (panic) and for those that mark the end of function scope (return) this
	// will be empty
	children []int
	// Nodes inside this block. Almost always these are statements but ternary expression will have expressions
	// as children.
	nodes []ast.Node
}

type bbRef int

func (bb *basicBlock) ref() bbRef {
	return bbRef(bb.id)
}

func (bb *basicBlock) isReachable() bool {
	return bb.id == 0 || len(bb.parents) > 0
}

func (bb *basicBlock) isTerminal() bool {
	return len(bb.children) == 0
}

func (bb *basicBlock) isEmpty() bool {
	return bb.id != 0 && len(bb.parents) == 0 && len(bb.nodes) == 0
}

type functionCFG struct {
	bbs       []basicBlock
	topoOrder []int // block IDs in topological order (non-backedge DAG); computed by markBackedges
}

type PackageCFG struct {
	funcCfgs   map[model.SymbolRef]functionCFG
	methodCfgs map[model.SymbolRef]map[model.SymbolRef]functionCFG
}

func (cfg *PackageCFG) lookupFunctionCfg(ref model.SymbolRef) (functionCFG, bool) {
	if fcfg, ok := cfg.funcCfgs[ref]; ok {
		return fcfg, true
	}
	for _, classMethods := range cfg.methodCfgs {
		if fcfg, ok := classMethods[ref]; ok {
			return fcfg, true
		}
	}
	return functionCFG{}, false
}

func (cfg *PackageCFG) allFunctionCfgs(yield func(model.SymbolRef, *functionCFG) bool) {
	for ref, fcfg := range cfg.funcCfgs {
		if !yield(ref, &fcfg) {
			return
		}
	}
	for _, classMethods := range cfg.methodCfgs {
		for ref, fcfg := range classMethods {
			if !yield(ref, &fcfg) {
				return
			}
		}
	}
}

func CreateControlFlowGraph(ctx *context.CompilerContext, pkg *ast.BLangPackage) *PackageCFG {
	cfg := &PackageCFG{
		funcCfgs:   make(map[model.SymbolRef]functionCFG),
		methodCfgs: make(map[model.SymbolRef]map[model.SymbolRef]functionCFG),
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, fn := range pkg.Functions {
		wg.Add(1)
		fn := fn // Capture loop variable
		go func() {
			defer wg.Done()
			fnCfg := analyzeFunction(ctx, &fn)
			mu.Lock()
			cfg.funcCfgs[fn.Symbol()] = fnCfg
			mu.Unlock()
		}()
	}
	if pkg.InitFunction != nil {
		wg.Add(1)
		initFn := pkg.InitFunction
		go func() {
			defer wg.Done()
			fnCfg := analyzeFunction(ctx, initFn)
			mu.Lock()
			cfg.funcCfgs[initFn.Symbol()] = fnCfg
			mu.Unlock()
		}()
	}
	analyzeClassBody := func(dest map[model.SymbolRef]functionCFG, initFn *ast.BLangFunction, methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod) {
		analyzeMethod := func(sym model.SymbolRef, body ast.FunctionBodyNode) {
			wg.Go(func() {
				fnCfg := analyzeFunctionBody(ctx, body)
				mu.Lock()
				dest[sym] = fnCfg
				mu.Unlock()
			})
		}
		if initFn != nil {
			analyzeMethod(initFn.Symbol(), initFn.Body)
		}
		for _, method := range methods {
			analyzeMethod(method.Symbol(), method.Body)
		}
		for _, rm := range resourceMethods {
			analyzeMethod(rm.Symbol(), rm.Body)
		}
	}
	for i := range pkg.ClassDefinitions {
		c := &pkg.ClassDefinitions[i]
		methodCfgs := make(map[model.SymbolRef]functionCFG)
		cfg.methodCfgs[c.Symbol()] = methodCfgs
		analyzeClassBody(methodCfgs, c.InitFunction, c.Methods, c.ResourceMethods)
	}
	for i := range pkg.Services {
		s := &pkg.Services[i]
		analyzeClassBody(cfg.funcCfgs, s.InitFunction, s.Methods, s.ResourceMethods)
	}
	wg.Wait()
	return cfg
}

func analyzeFunction(ctx *context.CompilerContext, fn *ast.BLangFunction) functionCFG {
	return analyzeFunctionBody(ctx, fn.Body)
}

func analyzeFunctionBody(ctx *context.CompilerContext, body ast.FunctionBodyNode) functionCFG {
	tyCtx := semtypes.ContextFrom(ctx.GetTypeEnv())
	analyzer := functionControlFlowAnalyzer{
		ctx:   ctx,
		tyCtx: tyCtx,
	}
	return analyzer.analyzeBody(body)
}

type functionControlFlowAnalyzer struct {
	ctx   *context.CompilerContext
	tyCtx semtypes.Context
	bbs   []basicBlock
	loops []loopControlFlowData
}

type loopControlFlowData struct {
	// Block with the condition
	loopHead bbRef
	// Block with the loop body
	loopBody bbRef
	loopEnd  bbRef
}

// stmtEffect tracks how a statement affects control flow
type stmtEffect struct {
	// The basic block to use for the next statement
	// -1 means control flow terminated (return, panic, break, continue)
	nextBB bbRef
}

var terminalBB = bbRef(-1)

// terminatedEffect returns an effect indicating control flow has terminated
func terminatedEffect() stmtEffect {
	return stmtEffect{nextBB: terminalBB}
}

func (effect *stmtEffect) isTerminal() bool {
	return effect.nextBB == terminalBB
}

// continueEffect returns an effect indicating control flow continues to the given block
func continueEffect(bb bbRef) stmtEffect {
	return stmtEffect{nextBB: bb}
}

func (analyzer *functionControlFlowAnalyzer) analyzeBody(body ast.FunctionBodyNode) functionCFG {
	switch fnBody := body.(type) {
	case *ast.BLangBlockFunctionBody:
		analyzer.analyzeBlockFunctionBody(fnBody)
	case *ast.BLangExprFunctionBody:
		analyzer.analyzeExprFunctionBody(fnBody)
	case *ast.BLangExternFunctionBody:
		// No body to analyze
	}
	return analyzer.getCfg()
}

func (analyzer *functionControlFlowAnalyzer) analyzeExprFunctionBody(fnBody *ast.BLangExprFunctionBody) {
	// TODO: we need to deal with ternary expression which will have control flow so bbs
	panic("unimplemented")
}

func (analyzer *functionControlFlowAnalyzer) analyzeBlockFunctionBody(fnBody *ast.BLangBlockFunctionBody) {
	rootBB := basicBlock{}
	analyzer.bbs = append(analyzer.bbs, rootBB)
	_ = analyzer.analyzeStatements(rootBB.ref(), fnBody.Stmts)
}

func (analyzer *functionControlFlowAnalyzer) analyzeStatements(curBB bbRef, statements []ast.StatementNode) stmtEffect {
	currentBB := curBB

	for _, stmt := range statements {
		// If current block is terminated (nextBB == -1), remaining statements are unreachable
		// but we still analyze them for completeness
		if currentBB == bbRef(-1) {
			analyzer.ctx.SemanticError("Unreachable code", stmt.GetPosition())
			// Create an unreachable block for the remaining statements
			currentBB = analyzer.createNewBB()
		}

		effect := analyzer.analyzeStatement(currentBB, stmt)

		// Update current block for next statement
		currentBB = effect.nextBB
	}

	// Return the final block where control flow continues
	return continueEffect(currentBB)
}

func (analyzer *functionControlFlowAnalyzer) addEdge(from, to bbRef) {
	// Only add edge if it doesn't already exist
	toInt := int(to)
	fromInt := int(from)

	// Check if child edge already exists
	childExists := slices.Contains(analyzer.bbs[from].children, toInt)
	if !childExists {
		analyzer.bbs[from].children = append(analyzer.bbs[from].children, toInt)
	}

	// Check if parent edge already exists
	parentExists := slices.Contains(analyzer.bbs[to].parents, fromInt)
	if !parentExists {
		analyzer.bbs[to].parents = append(analyzer.bbs[to].parents, fromInt)
	}
}

// createNewBB creates a new basic block and returns its reference
func (analyzer *functionControlFlowAnalyzer) createNewBB() bbRef {
	id := len(analyzer.bbs)
	bb := basicBlock{id: id}
	analyzer.bbs = append(analyzer.bbs, bb)
	return bb.ref()
}

// addNode adds a node to a basic block
func (analyzer *functionControlFlowAnalyzer) addNode(bb bbRef, node ast.Node) {
	analyzer.bbs[bb].nodes = append(analyzer.bbs[bb].nodes, node)
}

// analyzeStatement dispatches to the appropriate handler based on statement type
func (analyzer *functionControlFlowAnalyzer) analyzeStatement(curBB bbRef, stmt ast.StatementNode) stmtEffect {
	ternaryChecker := &ternaryExpressionChecker{}
	ast.Walk(ternaryChecker, stmt.(ast.BLangNode))
	if ternaryChecker.hasTernaryExpression {
		return createTernaryExpressionEffect(analyzer, curBB, ternaryChecker.ifTrue, ternaryChecker.ifFalse)
	}
	switch s := stmt.(type) {
	case *ast.BLangReturn:
		return analyzer.analyzeReturn(curBB, s)
	case *ast.BLangPanic:
		return analyzer.analyzePanic(curBB, s)
	case *ast.BLangExpressionStmt:
		analyzer.addNode(curBB, stmt)
		if expr, ok := s.Expr.(ast.BLangExpression); ok && alwaysTerminatesViaCheck(analyzer.tyCtx, expr) {
			return terminatedEffect()
		}
		return continueEffect(curBB)
	case *ast.BLangIf:
		return analyzer.analyzeIf(curBB, s)
	case *ast.BLangBlockStmt:
		return analyzer.analyzeBlockStmt(curBB, s)
	case *ast.BLangLock:
		return analyzer.analyzeBlockStmt(curBB, &s.Body)
	case *ast.BLangWhile:
		return analyzer.analyzeWhile(curBB, s)
	case *ast.BLangForeach:
		return analyzer.analyzeForeach(curBB, s)
	// These should be handled while handling while statement
	case *ast.BLangBreak:
		analyzer.addNode(curBB, stmt)
		if len(analyzer.loops) == 0 {
			analyzer.ctx.SemanticError("break statement not allowed outside loop", stmt.GetPosition())
			return continueEffect(curBB)
		}
		loopData := analyzer.loops[len(analyzer.loops)-1]
		analyzer.addEdge(curBB, loopData.loopEnd)
		return terminatedEffect()
	case *ast.BLangContinue:
		analyzer.addNode(curBB, stmt)
		if len(analyzer.loops) == 0 {
			analyzer.ctx.SemanticError("continue statement not allowed outside loop", stmt.GetPosition())
			return continueEffect(curBB)
		}
		loopData := analyzer.loops[len(analyzer.loops)-1]
		analyzer.addEdge(curBB, loopData.loopHead)
		return terminatedEffect()
	case *ast.BLangMatchStatement:
		return analyzer.analyzeMatch(curBB, s)
	default:
		// For unimplemented statement types, just add to current block and continue
		analyzer.addNode(curBB, stmt)
		return continueEffect(curBB)
	}
}

func createTernaryExpressionEffect(analyzer *functionControlFlowAnalyzer, curBB bbRef, expressionNode1, expressionNode2 ast.BLangExpression) stmtEffect {
	panic("unimplemented")
}

type ternaryExpressionChecker struct {
	hasTernaryExpression bool
	ifTrue               ast.BLangExpression
	ifFalse              ast.BLangExpression
}

var _ ast.Visitor = &ternaryExpressionChecker{}

func (c *ternaryExpressionChecker) Visit(node ast.BLangNode) ast.Visitor {
	if c.hasTernaryExpression {
		return nil
	}
	if _, ok := node.(*ast.BLangElvisExpr); ok {
		panic("unimplemented")
	}
	return c
}

func (c *ternaryExpressionChecker) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return nil
}

// Terminal statement handlers - these terminate control flow

func (analyzer *functionControlFlowAnalyzer) analyzeReturn(curBB bbRef, stmt *ast.BLangReturn) stmtEffect {
	analyzer.addNode(curBB, stmt)
	// Return terminates execution - current block has no children
	return terminatedEffect()
}

func (analyzer *functionControlFlowAnalyzer) analyzePanic(curBB bbRef, stmt *ast.BLangPanic) stmtEffect {
	analyzer.addNode(curBB, stmt)
	return terminatedEffect()
}

// alwaysTerminatesViaCheck reports whether the expression is a `check` or
// `checkpanic` whose operand is statically a subtype of error. In that case the
// `check` always returns from the enclosing function and the `checkpanic`
// always panics, so following statements are unreachable.
func alwaysTerminatesViaCheck(tyCtx semtypes.Context, expr ast.BLangExpression) bool {
	var inner ast.BLangExpression
	switch e := expr.(type) {
	case *ast.BLangCheckPanickedExpr:
		if v, ok := e.Expr.(ast.BLangExpression); ok {
			inner = v
		}
	case *ast.BLangCheckedExpr:
		if v, ok := e.Expr.(ast.BLangExpression); ok {
			inner = v
		}
	}
	if inner == nil {
		return false
	}
	return semtypes.IsSubtype(tyCtx, inner.GetDeterminedType(), semtypes.ERROR)
}

// Branching statement handlers

func (analyzer *functionControlFlowAnalyzer) analyzeBlockStmt(curBB bbRef, stmt *ast.BLangBlockStmt) stmtEffect {
	// Block statement just recursively analyzes its statements
	return analyzer.analyzeStatements(curBB, stmt.Stmts)
}

var (
	trueTy  = semtypes.BooleanConst(true)
	falseTy = semtypes.BooleanConst(false)
)

func (analyzer *functionControlFlowAnalyzer) analyzeIf(curBB bbRef, stmt *ast.BLangIf) stmtEffect {
	initBB := curBB
	ifTrue := analyzer.createNewBB()
	ifTrueEffect := analyzer.analyzeBlockStmt(ifTrue, &stmt.Body)
	finally := analyzer.createNewBB()
	if !ifTrueEffect.isTerminal() {
		analyzer.addEdge(ifTrueEffect.nextBB, finally)
	}
	analyzer.addNode(initBB, stmt.Expr)
	if !analyzer.isFalse(stmt.Expr) {
		analyzer.addEdge(initBB, ifTrue)
	}
	if stmt.ElseStmt != nil {
		ifFalse := analyzer.createNewBB()
		ifFalseEffect := analyzer.analyzeStatement(ifFalse, stmt.ElseStmt)
		if !ifFalseEffect.isTerminal() {
			analyzer.addEdge(ifFalseEffect.nextBB, finally)
		}
		if !analyzer.isTrue(stmt.Expr) {
			analyzer.addEdge(initBB, ifFalse)
		}
	} else if !analyzer.isTrue(stmt.Expr) {
		analyzer.addEdge(initBB, finally)
	}
	return continueEffect(finally)
}

func (analyzer *functionControlFlowAnalyzer) isFalse(node ast.BLangExpression) bool {
	return semtypes.IsSameType(analyzer.tyCtx, node.GetDeterminedType(), falseTy)
}

func (analyzer *functionControlFlowAnalyzer) isTrue(node ast.BLangExpression) bool {
	return semtypes.IsSameType(analyzer.tyCtx, node.GetDeterminedType(), trueTy)
}

func (analyzer *functionControlFlowAnalyzer) analyzeWhile(curBB bbRef, stmt *ast.BLangWhile) stmtEffect {
	loopHead := analyzer.createNewBB()
	loopBody := analyzer.createNewBB()
	loopEnd := analyzer.createNewBB()
	loopData := loopControlFlowData{
		loopHead: loopHead,
		loopBody: loopBody,
		loopEnd:  loopEnd,
	}
	analyzer.loops = append(analyzer.loops, loopData)
	analyzer.addEdge(curBB, loopHead)
	expr := stmt.Expr
	analyzer.addNode(loopHead, expr)
	if !analyzer.isFalse(expr) {
		analyzer.addEdge(loopHead, loopBody)
	}
	if !analyzer.isTrue(expr) {
		analyzer.addEdge(loopHead, loopEnd)
	}
	bodyEffect := analyzer.analyzeBlockStmt(loopBody, &stmt.Body)
	bodyEnd := bodyEffect.nextBB
	// Can happend with return/panic, continue or break respectively at the end. In all these cases we don't need to add explicit
	//  edges.
	if !bodyEffect.isTerminal() {
		analyzer.addEdge(bodyEnd, loopHead)
	}
	// Pop the loop from the stack now that we're done analyzing it
	analyzer.loops = analyzer.loops[:len(analyzer.loops)-1]
	return continueEffect(loopEnd)
}

func (analyzer *functionControlFlowAnalyzer) analyzeMatch(curBB bbRef, stmt *ast.BLangMatchStatement) stmtEffect {
	analyzer.addNode(curBB, stmt.Expr)
	finally := analyzer.createNewBB()
	hasIncoming := false
	for i := range stmt.MatchClauses {
		clause := &stmt.MatchClauses[i]
		clauseBB := analyzer.createNewBB()
		analyzer.addEdge(curBB, clauseBB)
		clauseEffect := analyzer.analyzeBlockStmt(clauseBB, &clause.Body)
		if !clauseEffect.isTerminal() {
			analyzer.addEdge(clauseEffect.nextBB, finally)
			hasIncoming = true
		}
	}
	if !stmt.IsExhaustive {
		analyzer.addEdge(curBB, finally)
		hasIncoming = true
	}
	if !hasIncoming {
		return terminatedEffect()
	}
	return continueEffect(finally)
}

func (cfg *functionCFG) markBackedges() {
	if len(cfg.bbs) == 0 {
		return
	}
	type color uint8
	const white color = 0
	const gray color = 1
	const black color = 2
	colors := make([]color, len(cfg.bbs))
	var postOrder []int
	var dfs func(id int)
	dfs = func(id int) {
		colors[id] = 1
		for _, childID := range cfg.bbs[id].children {
			switch colors[childID] {
			case gray:
				cfg.bbs[childID].backedgeParents = append(cfg.bbs[childID].backedgeParents, id)
			case white:
				dfs(childID)
			}
		}
		colors[id] = black
		postOrder = append(postOrder, id)
	}
	dfs(0)
	// Reverse post-order gives a topological ordering of the non-backedge DAG.
	cfg.topoOrder = make([]int, 0, len(cfg.bbs))
	for i := len(postOrder) - 1; i >= 0; i-- {
		cfg.topoOrder = append(cfg.topoOrder, postOrder[i])
	}
	// Append any unreachable blocks (not visited by DFS from root) at the end.
	for i := range cfg.bbs {
		if colors[i] == 0 {
			cfg.topoOrder = append(cfg.topoOrder, i)
		}
	}
}

func (analyzer *functionControlFlowAnalyzer) analyzeForeach(curBB bbRef, stmt *ast.BLangForeach) stmtEffect {
	loopHead := analyzer.createNewBB()
	loopBody := analyzer.createNewBB()
	loopEnd := analyzer.createNewBB()
	loopData := loopControlFlowData{
		loopHead: loopHead,
		loopBody: loopBody,
		loopEnd:  loopEnd,
	}
	analyzer.loops = append(analyzer.loops, loopData)
	analyzer.addEdge(curBB, loopHead)
	analyzer.addNode(loopHead, stmt.Collection)
	if stmt.VariableDef != nil {
		analyzer.addNode(loopHead, stmt.VariableDef)
	}
	analyzer.addEdge(loopHead, loopBody)
	analyzer.addEdge(loopHead, loopEnd)
	bodyEffect := analyzer.analyzeBlockStmt(loopBody, &stmt.Body)
	if !bodyEffect.isTerminal() {
		analyzer.addEdge(bodyEffect.nextBB, loopHead)
	}
	analyzer.loops = analyzer.loops[:len(analyzer.loops)-1]
	return continueEffect(loopEnd)
}

func (analyzer *functionControlFlowAnalyzer) getCfg() functionCFG {
	analyzer.pruneEmptyBlocks()
	cfg := functionCFG{bbs: analyzer.bbs}
	cfg.markBackedges()
	return cfg
}

func (analyzer *functionControlFlowAnalyzer) pruneEmptyBlocks() {
	for {
		empty := analyzer.findEmptyBlocks()
		if len(empty) == 0 {
			break
		}
		for _, bbIdx := range empty {
			bb := &analyzer.bbs[bbIdx]
			for _, childIdx := range bb.children {
				analyzer.removeParent(childIdx, bbIdx)
			}
		}
		analyzer.removeBlocksAndReindex(empty)
	}
}

func (analyzer *functionControlFlowAnalyzer) findEmptyBlocks() []int {
	var empty []int
	for i := range analyzer.bbs {
		if analyzer.bbs[i].isEmpty() {
			empty = append(empty, i)
		}
	}
	return empty
}

func (analyzer *functionControlFlowAnalyzer) removeParent(bbIdx, parentIdx int) {
	parents := analyzer.bbs[bbIdx].parents
	newParents := parents[:0]
	for _, p := range parents {
		if p != parentIdx {
			newParents = append(newParents, p)
		}
	}
	analyzer.bbs[bbIdx].parents = newParents
}

func remapRefs(current []int, mapping map[int]int) []int {
	result := make([]int, 0, len(current))
	for _, v := range current {
		if newV, ok := mapping[v]; ok {
			result = append(result, newV)
		}
	}
	return result
}

func (analyzer *functionControlFlowAnalyzer) removeBlocksAndReindex(toRemove []int) {
	removeSet := make(map[int]bool)
	for _, i := range toRemove {
		removeSet[i] = true
	}
	var newBbs []basicBlock
	oldToNew := make(map[int]int)
	for oldIdx, bb := range analyzer.bbs {
		if removeSet[oldIdx] {
			continue
		}
		newIdx := len(newBbs)
		oldToNew[oldIdx] = newIdx
		bb.id = newIdx
		newBbs = append(newBbs, bb)
	}
	for i := range newBbs {
		bb := &newBbs[i]
		bb.parents = remapRefs(bb.parents, oldToNew)
		bb.children = remapRefs(bb.children, oldToNew)
	}
	analyzer.bbs = newBbs
}
