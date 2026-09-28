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

// Package desugar represents AST-> AST transforms
package desugar

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// Query expressions and query actions are lowered through one clause pipeline. The clauses
// become nested loops, assignments and conditionals that run once per frame and end in a
// terminal: pushing the select value, running the do body, or materializing the frame as a
// row. order by and group by need every input frame before they emit, so the pipeline is
// split at those clauses; the segment before a barrier materializes rows and the segment
// after it iterates the sorted or grouped rows.
//
// One generated int holds the pipeline state. Every loop runs while the state is running,
// so a limit, a completion error or a control transfer in the do body unwinds all loops
// through their conditions.
const (
	queryStateRunning int64 = iota
	// A limit was reached; no more input frames are needed.
	queryStateStopped
	// A clause completed early with an error, which is stored in the result variable.
	queryStateFailed
	// The do body ran break, which is re-issued after the loops.
	queryStateBreak
	// The do body ran continue, which is re-issued after the loops.
	queryStateContinue
)

type queryLowering struct {
	cx          *functionContext
	initStmts   []ast.StatementNode
	resultRef   *ast.BLangVarRef
	stateVarDef ast.StatementNode
	stateRef    *ast.BLangVarRef
	pos         diagnostics.Location
}

type queryTerminalKind int

const (
	queryTerminalDo queryTerminalKind = iota
	queryTerminalSelect
	queryTerminalRows
)

// queryTerminal is what a segment does with each frame that reaches its end.
type queryTerminal struct {
	kind queryTerminalKind
	// queryTerminalDo: the desugared do body.
	doStmts []ast.StatementNode
	// queryTerminalSelect: the select clause and how its values are collected.
	selectClause     *ast.BLangSelectClause
	constructType    ast.TypeKind
	onConflictClause *ast.BLangOnConflictClause
	seenKeysRef      *ast.BLangVarRef
	// queryTerminalRows: the frame is pushed to rowsRef. A barrier terminal also evaluates the
	// barrier's key expressions for the frame and pushes them to keyRowsRef.
	rowsRef    *ast.BLangVarRef
	keyRowsRef *ast.BLangVarRef
	orderBy    *ast.BLangOrderByClause
	groupBy    *ast.BLangGroupByClause
}

type queryRowBinding struct {
	varName         ast.IdentifierNode
	symbol          model.SymbolRef
	valueTy         semtypes.SemType
	groupAggregated bool
}

// queryCollectionSource is an indexable list or map source, or a receiver whose next method
// is pulled by a generated loop for string, XML, stream and object:Iterable sources.
type queryCollectionSource struct {
	collectionRef   *ast.BLangVarRef
	keysRef         *ast.BLangVarRef
	rowCountRef     *ast.BLangVarRef
	nextReceiverRef *ast.BLangVarRef
	nextReceiverTy  semtypes.SemType
}

type queryLimit struct {
	limitRef   *ast.BLangVarRef
	counterRef *ast.BLangVarRef
}

// queryJoin caches the join clause's right side as [value, key] rows before the pipeline runs.
type queryJoin struct {
	binding     queryRowBinding
	rowsRef     *ast.BLangVarRef
	rowCountRef *ast.BLangVarRef
	keyTy       semtypes.SemType
}

func walkQueryExpr(cx *functionContext, expr *ast.BLangQueryExpr) desugaredNode[ast.BLangActionOrExpression] {
	clauses := expr.QueryClauseList
	finalClauseIndex := len(clauses) - 1
	var onConflictClause *ast.BLangOnConflictClause
	if clause, isOnConflict := clauses[finalClauseIndex].(*ast.BLangOnConflictClause); isOnConflict {
		onConflictClause = clause
		finalClauseIndex--
	}
	queryTy := expr.GetDeterminedType()
	basePos := expr.GetPosition()
	q := &queryLowering{cx: cx, pos: basePos}
	pipeline := clauses[:finalClauseIndex]

	if selectClause, ok := clauses[finalClauseIndex].(*ast.BLangSelectClause); ok {
		q.resultRef = q.declareResult(queryTy, createQueryResultInitializer(expr.QueryConstructType, queryTy), basePos)
		terminal := queryTerminal{
			kind:             queryTerminalSelect,
			selectClause:     selectClause,
			constructType:    expr.QueryConstructType,
			onConflictClause: onConflictClause,
		}
		if onConflictClause != nil && expr.QueryConstructType == ast.TypeKindMap {
			terminal.seenKeysRef = createQueryMapStore(cx, &q.initStmts, basePos)
		}
		if _, ok := q.lowerPipeline(pipeline, terminal); !ok {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
		}
		return q.finish()
	}

	collectClause := clauses[finalClauseIndex].(*ast.BLangCollectClause)
	q.resultRef = q.declareResult(queryTy, createQueryNilLiteral(basePos), basePos)
	rowsRef := createQueryListStore(cx, &q.initStmts, basePos)
	bindings, ok := q.lowerPipeline(pipeline, queryTerminal{kind: queryTerminalRows, rowsRef: rowsRef})
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}
	q.appendCollectResult(rowsRef, bindings, collectClause)
	return q.finish()
}

// walkQueryAction lowers a query action into setup statements, the pipeline loops and a
// nil-or-error result.
func walkQueryAction(cx *functionContext, action *ast.BLangQueryAction) desugaredNode[ast.BLangActionOrExpression] {
	basePos := action.GetPosition()
	q := &queryLowering{cx: cx, pos: basePos}
	q.resultRef = q.declareResult(action.GetDeterminedType(), createQueryNilLiteral(basePos), basePos)
	doStmts, controlInfo := q.lowerDoBody(action.DoClause.Body)
	if _, ok := q.lowerPipeline(action.QueryClauseList, queryTerminal{kind: queryTerminalDo, doStmts: doStmts}); !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
	}
	q.appendControlDispatch(controlInfo, basePos)
	return q.finish()
}

func (q *queryLowering) finish() desugaredNode[ast.BLangActionOrExpression] {
	initStmts := q.initStmts
	if q.stateVarDef != nil {
		initStmts = append([]ast.StatementNode{q.stateVarDef}, initStmts...)
	}
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: q.resultRef,
	}
}

func createQueryResultInitializer(constructType ast.TypeKind, queryTy semtypes.SemType) ast.BLangExpression {
	if constructType == ast.TypeKindMap {
		emptyMap := &ast.BLangMappingConstructorExpr{Fields: []ast.MappingField{}}
		emptyMap.SetDeterminedType(semtypes.Intersect(queryTy, semtypes.Mapping))
		return emptyMap
	}
	emptyList := &ast.BLangListConstructorExpr{Exprs: []ast.BLangExpression{}}
	emptyList.SetDeterminedType(semtypes.List)
	emptyList.AtomicType = semtypes.ListAtomicInner
	return emptyList
}

func (q *queryLowering) declareResult(
	resultTy semtypes.SemType,
	initExpr ast.BLangExpression,
	pos diagnostics.Location,
) *ast.BLangVarRef {
	resultName, resultSymbol := q.cx.addDesugardSymbol(resultTy, model.SymbolKindVariable, pos)
	resultVar := &ast.BLangVariable{Name: newIdentifier(resultName)}
	resultVar.Name.SetDeterminedType(semtypes.Never)
	resultVar.SetDeterminedType(semtypes.Never)
	resultVar.SetInitialExpression(initExpr)
	resultVar.SetSymbol(resultSymbol)
	resultVarDef := &ast.BLangVariableDef{Var: resultVar}
	resultVarDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(resultVarDef, pos)
	q.initStmts = append(q.initStmts, resultVarDef)

	resultRef := &ast.BLangVarRef{VariableName: resultVar.Name}
	resultRef.SetSymbol(resultSymbol)
	resultRef.SetDeterminedType(resultTy)
	setPositionIfMissing(resultRef, pos)
	return resultRef
}

// state returns the pipeline state variable, declaring it on first use. Loops built before the
// first use are all inside the point that first needs to stop, so they need no state check.
func (q *queryLowering) state() *ast.BLangVarRef {
	if q.stateRef == nil {
		q.stateVarDef, q.stateRef = assignToLocal(q.cx, createIntLiteral(queryStateRunning), q.pos)
	}
	return q.stateRef
}

func (q *queryLowering) stateIs(state int64, pos diagnostics.Location) ast.BLangExpression {
	return createQueryIntComparison(q.state(), model.OperatorKind_EQUAL, createIntLiteral(state), pos)
}

func (q *queryLowering) setState(state int64, pos diagnostics.Location) ast.StatementNode {
	assign := &ast.BLangAssignment{
		VarRef: createQueryVarRefAt(q.state(), pos),
		Expr:   createIntLiteral(state),
	}
	assign.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assign, pos)
	return assign
}

// lowerDoBody desugars the do body once. A break or continue in the body targets the loop
// enclosing the query action, so it records the transfer in the pipeline state and leaves the
// innermost generated loop; appendControlDispatch re-issues the transfer after the loops.
func (q *queryLowering) lowerDoBody(body *ast.BLangBlockStmt) ([]ast.StatementNode, queryActionControlFlowInfo) {
	cx := q.cx
	controlInfo := queryActionControlFlow(body)
	controlled := controlInfo.hasBreak || controlInfo.hasContinue
	if controlled {
		cx.pushLoopVar(nil)
		cx.pushQueryActionControl(queryActionControlFlowState{
			loopDepth: len(cx.loopVarStack),
			stateRef:  q.state(),
		})
	}
	doResult := walkBlockStmt(cx, body)
	if controlled {
		cx.popQueryActionControl()
		cx.popLoopVar()
	}
	doStmts := append([]ast.StatementNode{}, doResult.initStmts...)
	return append(doStmts, doResult.replacementNode.(*ast.BLangBlockStmt).Stmts...), controlInfo
}

func (q *queryLowering) appendControlDispatch(controlInfo queryActionControlFlowInfo, pos diagnostics.Location) {
	if controlInfo.hasBreak {
		breakStmt := &ast.BLangBreak{}
		breakStmt.SetDeterminedType(semtypes.Never)
		q.appendControlTransfer(queryStateBreak, breakStmt, pos)
	}
	if controlInfo.hasContinue {
		continueStmt := &ast.BLangContinue{}
		continueStmt.SetDeterminedType(semtypes.Never)
		q.appendControlTransfer(queryStateContinue, continueStmt, pos)
	}
}

// appendControlTransfer re-issues a transfer recorded by the do body. The transfer is walked so
// that a surrounding desugared foreach still advances its loop variable and a surrounding query
// action records it in turn.
func (q *queryLowering) appendControlTransfer(state int64, transfer ast.StatementNode, pos diagnostics.Location) {
	setPositionIfMissing(transfer.(ast.BLangNode), pos)
	result := walkStatement(q.cx, transfer)
	bodyStmts := append([]ast.StatementNode{}, result.initStmts...)
	bodyStmts = append(bodyStmts, result.replacementNode)
	q.initStmts = append(q.initStmts, q.ifStmt(q.stateIs(state, pos), bodyStmts, pos))
}

// lowerPipeline generates the loops for every clause. The first from clause's collection is
// evaluated before anything else; segments after a barrier iterate the barrier's rows.
func (q *queryLowering) lowerPipeline(clauses []ast.BLangNode, terminal queryTerminal) ([]queryRowBinding, bool) {
	fromClause := clauses[0].(*ast.BLangFromClause)
	fromBinding := q.bindingFor(fromClause.VariableDefinitionNode)
	q.declareBinding(fromBinding, q.pos)
	source, ok := q.collectionSource(&q.initStmts, fromClause.Collection, q.pos)
	if !ok {
		return nil, false
	}

	bindings := []queryRowBinding{fromBinding}
	var rowsRef *ast.BLangVarRef
	segmentStart := 1
	for {
		barrierIndex := nextQueryBarrier(clauses, segmentStart)
		segmentEnd := len(clauses)
		segmentTerminal := terminal
		if barrierIndex >= 0 {
			segmentEnd = barrierIndex
			segmentTerminal = q.barrierTerminal(clauses[barrierIndex])
		}
		segmentStmts, outputBindings, ok := q.buildSegment(clauses, segmentStart, segmentEnd, bindings, segmentTerminal)
		if !ok {
			return nil, false
		}
		if segmentStart == 1 {
			q.appendCollectionLoop(&q.initStmts, source, fromBinding, segmentStmts, q.pos, true)
		} else if !q.appendRowsLoop(rowsRef, bindings, segmentStmts, q.pos) {
			return nil, false
		}
		bindings = outputBindings
		if barrierIndex < 0 {
			return bindings, true
		}
		rowsRef, bindings, ok = q.applyBarrier(segmentTerminal, bindings, clauses[barrierIndex].GetPosition())
		if !ok {
			return nil, false
		}
		q.appendBarrierRestart(clauses[barrierIndex].GetPosition())
		segmentStart = barrierIndex + 1
	}
}

// appendBarrierRestart resumes the pipeline after a barrier when a limit stopped the barrier's
// input: the barrier has every frame it needs, and the next segment consumes them all.
func (q *queryLowering) appendBarrierRestart(pos diagnostics.Location) {
	if q.stateRef == nil {
		return
	}
	restart := []ast.StatementNode{q.setState(queryStateRunning, pos)}
	q.initStmts = append(q.initStmts, q.ifStmt(q.stateIs(queryStateStopped, pos), restart, pos))
}

// nextQueryBarrier finds the next clause that needs all of its input frames before it emits.
func nextQueryBarrier(clauses []ast.BLangNode, start int) int {
	for i := start; i < len(clauses); i++ {
		switch clauses[i].(type) {
		case *ast.BLangOrderByClause, *ast.BLangGroupByClause:
			return i
		}
	}
	return -1
}

func (q *queryLowering) barrierTerminal(barrier ast.BLangNode) queryTerminal {
	terminal := queryTerminal{
		kind:       queryTerminalRows,
		rowsRef:    createQueryListStore(q.cx, &q.initStmts, q.pos),
		keyRowsRef: createQueryListStore(q.cx, &q.initStmts, q.pos),
	}
	switch clause := barrier.(type) {
	case *ast.BLangOrderByClause:
		terminal.orderBy = clause
	case *ast.BLangGroupByClause:
		terminal.groupBy = clause
	}
	return terminal
}

// applyBarrier sorts or groups the rows a barrier segment materialized and returns the rows
// and bindings the next segment iterates.
func (q *queryLowering) applyBarrier(
	terminal queryTerminal,
	bindings []queryRowBinding,
	pos diagnostics.Location,
) (*ast.BLangVarRef, []queryRowBinding, bool) {
	if terminal.orderBy != nil {
		directionsExpr := buildOrderDirectionExpr(terminal.orderBy, pos)
		sortInvocation := createQuerySortInvocation(q.cx, terminal.keyRowsRef, directionsExpr, terminal.rowsRef)
		if sortInvocation == nil {
			return nil, nil, false
		}
		sortStmt := &ast.BLangExpressionStmt{Expr: sortInvocation}
		setPositionIfMissing(sortStmt, pos)
		q.initStmts = append(q.initStmts, sortStmt)
		return terminal.rowsRef, bindings, true
	}
	groupingSymbols := queryGroupingSymbols(q.cx, terminal.groupBy)
	scalarFlags := buildQueryGroupScalarFlags(bindings, groupingSymbols, pos)
	groupInvocation := createQueryGroupInvocation(q.cx, terminal.rowsRef, terminal.keyRowsRef, scalarFlags)
	groupedRowsDef, groupedRowsRef := assignToLocal(q.cx, groupInvocation, pos)
	q.initStmts = append(q.initStmts, groupedRowsDef)
	return groupedRowsRef, queryGroupOutputBindings(q.cx, bindings, groupingSymbols), true
}

// buildSegment nests the clauses in [clauseIndex, endClauseIndex) around the terminal,
// producing a shape such as: let assignment; if where { for fromValue { <join>; <terminal> } }.
// The returned statements run once per input frame with the input bindings live.
func (q *queryLowering) buildSegment(
	clauses []ast.BLangNode,
	clauseIndex int,
	endClauseIndex int,
	bindings []queryRowBinding,
	terminal queryTerminal,
) ([]ast.StatementNode, []queryRowBinding, bool) {
	if clauseIndex == endClauseIndex {
		return q.buildTerminal(terminal, bindings)
	}
	switch clause := clauses[clauseIndex].(type) {
	case *ast.BLangFromClause:
		fromBinding := q.bindingFor(clause.VariableDefinitionNode)
		q.declareBinding(fromBinding, clause.GetPosition())
		nextStmts, outputBindings, ok := q.buildSegment(
			clauses, clauseIndex+1, endClauseIndex, appendQueryBinding(bindings, fromBinding), terminal,
		)
		if !ok {
			return nil, nil, false
		}
		// The collection expression can reference earlier bindings, so it is evaluated once per
		// input frame inside the enclosing loop.
		var fromStmts []ast.StatementNode
		source, ok := q.collectionSource(&fromStmts, clause.Collection, clause.GetPosition())
		if !ok {
			return nil, nil, false
		}
		q.appendCollectionLoop(&fromStmts, source, fromBinding, nextStmts, clause.GetPosition(), true)
		return fromStmts, outputBindings, true
	case *ast.BLangLetClause:
		letStmts := make([]ast.StatementNode, 0, len(clause.LetVarDeclarations))
		newBindings := append([]queryRowBinding{}, bindings...)
		for i := range clause.LetVarDeclarations {
			varDef := &clause.LetVarDeclarations[i]
			binding := q.bindingFor(varDef)
			q.declareBinding(binding, clause.GetPosition())
			letStmts = append(letStmts, createQueryBindingAssignment(
				binding, walkExpression(q.cx, varDef.Var.Expr), clause.GetPosition(),
			))
			newBindings = append(newBindings, binding)
		}
		nextStmts, outputBindings, ok := q.buildSegment(clauses, clauseIndex+1, endClauseIndex, newBindings, terminal)
		return append(letStmts, nextStmts...), outputBindings, ok
	case *ast.BLangWhereClause:
		whereResult := walkExpression(q.cx, clause.Expression)
		var whereStmts []ast.StatementNode
		whereExpr, isExpression := whereResult.(ast.BLangExpression)
		if !isExpression {
			whereVarDef, whereRef := assignToLocal(q.cx, whereResult, clause.GetPosition())
			whereStmts = append(whereStmts, whereVarDef)
			whereExpr = whereRef
		}
		nextStmts, outputBindings, ok := q.buildSegment(clauses, clauseIndex+1, endClauseIndex, bindings, terminal)
		if !ok {
			return nil, nil, false
		}
		return append(whereStmts, q.ifStmt(whereExpr, nextStmts, clause.GetPosition())), outputBindings, true
	case *ast.BLangLimitClause:
		limit := q.prepareLimit(clause)
		nextStmts, outputBindings, ok := q.buildSegment(clauses, clauseIndex+1, endClauseIndex, bindings, terminal)
		if !ok {
			return nil, nil, false
		}
		return []ast.StatementNode{q.limitStmt(limit, nextStmts, clause.GetPosition())}, outputBindings, true
	case *ast.BLangJoinClause:
		join, ok := q.prepareJoin(clause)
		if !ok {
			return nil, nil, false
		}
		nextStmts, outputBindings, ok := q.buildSegment(
			clauses, clauseIndex+1, endClauseIndex, appendQueryBinding(bindings, join.binding), terminal,
		)
		if !ok {
			return nil, nil, false
		}
		return q.buildJoinScan(clause, join, nextStmts, clause.GetPosition()), outputBindings, true
	default:
		q.cx.internalError("unsupported query clause", clause.GetPosition())
		return nil, nil, false
	}
}

// buildTerminal generates what happens to a frame at the end of a segment and returns the
// bindings the frame carries afterwards.
func (q *queryLowering) buildTerminal(terminal queryTerminal, bindings []queryRowBinding) ([]ast.StatementNode, []queryRowBinding, bool) {
	switch terminal.kind {
	case queryTerminalDo:
		return terminal.doStmts, bindings, true
	case queryTerminalSelect:
		stmts, ok := q.selectStmts(terminal)
		return stmts, bindings, ok
	default:
		return q.rowStmts(terminal, bindings)
	}
}

// rowStmts pushes the frame as a row. A barrier terminal evaluates the barrier's keys for the
// frame right after the preceding clauses, so key expressions interleave with them per frame.
func (q *queryLowering) rowStmts(terminal queryTerminal, bindings []queryRowBinding) ([]ast.StatementNode, []queryRowBinding, bool) {
	pos := q.pos
	var stmts []ast.StatementNode
	var keyExprs []ast.BLangExpression
	rowBindings := bindings
	switch {
	case terminal.orderBy != nil:
		pos = terminal.orderBy.GetPosition()
		keyExprs = buildOrderKeyExprs(q.cx, terminal.orderBy)
	case terminal.groupBy != nil:
		pos = terminal.groupBy.GetPosition()
		stmts, keyExprs, rowBindings = q.groupingKeyStmts(terminal.groupBy, bindings)
	}
	pushRow, ok := q.pushStmt(terminal.rowsRef, createQueryRowTupleExpr(rowBindings, nil, pos), pos)
	if !ok {
		return nil, nil, false
	}
	stmts = append(stmts, pushRow)
	if terminal.keyRowsRef != nil {
		pushKeys, ok := q.pushStmt(terminal.keyRowsRef, createQueryListExpr(keyExprs, pos), pos)
		if !ok {
			return nil, nil, false
		}
		stmts = append(stmts, pushKeys)
	}
	return stmts, rowBindings, true
}

// groupingKeyStmts evaluates a group by clause's keys for the current frame. A key declared with
// a variable becomes a new binding that is carried in the row.
func (q *queryLowering) groupingKeyStmts(
	clause *ast.BLangGroupByClause,
	bindings []queryRowBinding,
) ([]ast.StatementNode, []ast.BLangExpression, []queryRowBinding) {
	pos := clause.GetPosition()
	var stmts []ast.StatementNode
	keyExprs := make([]ast.BLangExpression, 0, len(clause.GroupingKeyList))
	rowBindings := append([]queryRowBinding{}, bindings...)
	for _, groupingKey := range clause.GroupingKeyList {
		if groupingKey.VariableRef != nil {
			keyExprs = append(keyExprs, walkExpression(q.cx, groupingKey.VariableRef).(ast.BLangExpression))
			continue
		}
		varDef := groupingKey.VariableDef
		keyExpr := walkExpression(q.cx, varDef.Var.Expr).(ast.BLangExpression)
		if queryVarDefHasBindableSymbol(varDef) {
			binding := q.bindingFor(varDef)
			q.declareBinding(binding, pos)
			stmts = append(stmts, createQueryBindingAssignment(binding, keyExpr, pos))
			rowBindings = append(rowBindings, binding)
			keyExpr = createQueryBindingVarRef(binding)
		}
		keyExprs = append(keyExprs, keyExpr)
	}
	return stmts, keyExprs, rowBindings
}

func queryGroupingSymbols(cx *functionContext, clause *ast.BLangGroupByClause) map[model.SymbolRef]bool {
	symbols := make(map[model.SymbolRef]bool)
	for _, groupingKey := range clause.GroupingKeyList {
		switch {
		case groupingKey.VariableRef != nil:
			symbols[cx.pkgCtx.compilerCtx.UnnarrowedSymbol(groupingKey.VariableRef.Symbol())] = true
		case queryVarDefHasBindableSymbol(groupingKey.VariableDef):
			symbols[groupingKey.VariableDef.Var.Symbol()] = true
		}
	}
	return symbols
}

// selectStmts appends the select value to the result: a push for lists, a keyed put for maps,
// where an on conflict clause decides whether a repeated key fails the query.
func (q *queryLowering) selectStmts(terminal queryTerminal) ([]ast.StatementNode, bool) {
	cx := q.cx
	selectClause := terminal.selectClause
	basePos := selectClause.GetPosition()
	var stmts []ast.StatementNode
	selectResult := walkExpression(cx, selectClause.Expression)
	selectExpr, isExpression := selectResult.(ast.BLangExpression)
	if !isExpression {
		selectVarDef, selectRef := assignToLocal(cx, selectResult, basePos)
		stmts = append(stmts, selectVarDef)
		selectExpr = selectRef
	}
	if terminal.constructType != ast.TypeKindMap {
		pushStmt, ok := q.pushStmt(q.resultRef, selectExpr, basePos)
		if !ok {
			return nil, false
		}
		return append(stmts, pushStmt), true
	}

	pairVarDef, pairRef := assignToLocal(cx, selectExpr, basePos)
	stmts = append(stmts, pairVarDef)
	keyAccess := createQueryRowSlotAccess(pairRef, 0, semtypes.String, basePos)
	valueAccess := createQueryRowSlotAccess(pairRef, 1, semtypes.Any, basePos)
	mapPutStmt := createMapPutAssignment(q.resultRef, keyAccess, valueAccess)
	setPositionIfMissing(mapPutStmt, basePos)
	if terminal.onConflictClause == nil {
		return append(stmts, mapPutStmt), true
	}
	stmts = append(stmts, q.onConflictStmts(terminal, keyAccess)...)
	markSeen := createMapPutAssignment(terminal.seenKeysRef, keyAccess, createBoolLiteral(true, basePos))
	setPositionIfMissing(markSeen, basePos)
	putStmts := []ast.StatementNode{markSeen, mapPutStmt}
	return append(stmts, q.ifStmt(q.stateIs(queryStateRunning, basePos), putStmts, basePos)), true
}

// onConflictStmts evaluates the on conflict expression when the key was already put; an error
// becomes the query result and fails the pipeline, which skips the put.
func (q *queryLowering) onConflictStmts(terminal queryTerminal, keyAccess ast.BLangExpression) []ast.StatementNode {
	cx := q.cx
	onConflictClause := terminal.onConflictClause
	pos := onConflictClause.GetPosition()
	seenLookup := &ast.BLangIndexBasedAccess{IndexExpr: keyAccess}
	seenLookup.Expr = terminal.seenKeysRef
	seenLookup.SetDeterminedType(semtypes.Any)
	conflictCond := &ast.BLangBinaryExpr{
		LhsExpr: seenLookup,
		RhsExpr: createBoolLiteral(true, pos),
		OpKind:  model.OperatorKind_EQUAL,
	}
	conflictCond.SetDeterminedType(semtypes.Boolean)

	conflictVarDef, conflictRef := assignToLocal(cx, walkExpression(cx, onConflictClause.Expression), pos)
	isErrorExpr := createQueryActionTypeTest(conflictRef, semtypes.Error, pos)
	failStmts := []ast.StatementNode{
		createQueryResultAssignment(q.resultRef, conflictRef, pos),
		q.setState(queryStateFailed, pos),
	}
	conflictBody := []ast.StatementNode{conflictVarDef, q.ifStmt(isErrorExpr, failStmts, pos)}
	return []ast.StatementNode{q.ifStmt(conflictCond, conflictBody, pos)}
}

// appendCollectResult evaluates the collect expression once over all rows, with every binding
// aggregated into a list, unless a clause completed early with an error.
func (q *queryLowering) appendCollectResult(
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	collectClause *ast.BLangCollectClause,
) {
	cx := q.cx
	pos := collectClause.GetPosition()
	flattenFlags := buildQueryCollectFlattenFlags(bindings, pos)
	collectInvocation := createQueryCollectInvocation(cx, rowsRef, createIntLiteral(int64(len(bindings))), flattenFlags)
	collectRowDef, collectRowRef := assignToLocal(cx, collectInvocation, pos)

	stmts := []ast.StatementNode{collectRowDef}
	for i, binding := range bindings {
		collectBinding := binding
		if !binding.groupAggregated {
			collectBinding.valueTy = queryListValueType(cx.typeEnv(), binding.valueTy, false)
		}
		stmts = append(stmts, createQueryBindingAssignment(
			collectBinding,
			createQueryRowSlotAccess(collectRowRef, i, collectBinding.valueTy, pos),
			pos,
		))
	}
	stmts = append(stmts, createQueryResultAssignment(q.resultRef, walkExpression(cx, collectClause.Expression), pos))
	if q.stateRef == nil {
		q.initStmts = append(q.initStmts, stmts...)
		return
	}
	notFailed := createQueryIntComparison(q.stateRef, model.OperatorKind_NOT_EQUAL, createIntLiteral(queryStateFailed), pos)
	q.initStmts = append(q.initStmts, q.ifStmt(notFailed, stmts, pos))
}

func buildQueryCollectFlattenFlags(bindings []queryRowBinding, pos diagnostics.Location) *ast.BLangListConstructorExpr {
	flags := make([]ast.BLangExpression, 0, len(bindings))
	for _, binding := range bindings {
		flags = append(flags, createBoolLiteral(binding.groupAggregated, pos))
	}
	return createQueryListExpr(flags, pos)
}

// prepareLimit evaluates the limit once before the loops:
//
//	limit = <expression>; panic if limit < 0; count = 0; if limit == 0 { state = stopped }
func (q *queryLowering) prepareLimit(clause *ast.BLangLimitClause) queryLimit {
	cx := q.cx
	pos := clause.GetPosition()
	limitVarDef, limitRef := assignToLocal(cx, walkExpression(cx, clause.Expression), pos)
	q.initStmts = append(q.initStmts, limitVarDef, createNegativeLimitPanicIf(cx, limitRef, pos))
	counterRef := createQueryCounterRef(cx, &q.initStmts, pos)
	zeroLimit := createQueryIntComparison(limitRef, model.OperatorKind_EQUAL, createIntLiteral(0), pos)
	noInputNeeded := createQueryAnd(q.stateIs(queryStateRunning, pos), zeroLimit, pos)
	q.initStmts = append(q.initStmts, q.ifStmt(noInputNeeded, []ast.StatementNode{q.setState(queryStateStopped, pos)}, pos))
	return queryLimit{limitRef: limitRef, counterRef: counterRef}
}

// limitStmt lets a frame through while the count is below the limit. Once the limit is reached
// the state stops the upstream loops after the frame's downstream work is done.
func (q *queryLowering) limitStmt(limit queryLimit, nextStmts []ast.StatementNode, pos diagnostics.Location) ast.StatementNode {
	withinLimit := createQueryIntComparison(limit.counterRef, model.OperatorKind_LESS_THAN, createQueryVarRefAt(limit.limitRef, pos), pos)
	reachedLimit := createQueryIntComparison(limit.counterRef, model.OperatorKind_GREATER_EQUAL, createQueryVarRefAt(limit.limitRef, pos), pos)
	stopUpstream := createQueryAnd(q.stateIs(queryStateRunning, pos), reachedLimit, pos)
	body := []ast.StatementNode{createIncrementStmt(limit.counterRef)}
	body = append(body, nextStmts...)
	body = append(body, q.ifStmt(stopUpstream, []ast.StatementNode{q.setState(queryStateStopped, pos)}, pos))
	return q.ifStmt(withinLimit, body, pos)
}

// prepareJoin evaluates the join's right side once and caches it as [value, key] rows:
//
//	for each value in <join collection> { rows.push([value, <right key>]); }
//
// The cache is built independently of the pipeline state, but a completion error in it fails
// the pipeline before any frame is emitted.
func (q *queryLowering) prepareJoin(clause *ast.BLangJoinClause) (queryJoin, bool) {
	cx := q.cx
	pos := clause.GetPosition()
	binding := q.bindingFor(clause.VariableDefinitionNode)
	q.declareBinding(binding, pos)
	rowsRef := createQueryListStore(cx, &q.initStmts, pos)
	source, ok := q.collectionSource(&q.initStmts, clause.Collection, pos)
	if !ok {
		return queryJoin{}, false
	}
	rhsExpr := walkExpression(cx, clause.OnClause.EqualsExpr).(ast.BLangExpression)
	rowTuple := createQueryRowTupleExpr(nil, []ast.BLangExpression{createQueryBindingVarRef(binding), rhsExpr}, pos)
	pushStmt, ok := q.pushStmt(rowsRef, rowTuple, pos)
	if !ok {
		return queryJoin{}, false
	}
	q.appendCollectionLoop(&q.initStmts, source, binding, []ast.StatementNode{pushStmt}, pos, false)
	rowCountRef := source.rowCountRef
	if rowCountRef == nil {
		rowCountRef, ok = createQueryLengthRef(cx, &q.initStmts, rowsRef, pos)
		if !ok {
			return queryJoin{}, false
		}
	}
	keyTy := rhsExpr.GetDeterminedType()
	if semtypes.IsZero(keyTy) {
		keyTy = semtypes.Any
	}
	return queryJoin{binding: binding, rowsRef: rowsRef, rowCountRef: rowCountRef, keyTy: keyTy}, true
}

// buildJoinScan scans the cached right-side rows for one left frame. Matching rows run nextStmts
// immediately; an outer join performs one extra sentinel iteration that binds nil and runs
// nextStmts when no cached row matched.
func (q *queryLowering) buildJoinScan(
	clause *ast.BLangJoinClause,
	join queryJoin,
	nextStmts []ast.StatementNode,
	pos diagnostics.Location,
) []ast.StatementNode {
	cx := q.cx
	lhsVarDef, lhsRef := assignToLocal(cx, walkExpression(cx, clause.OnClause.OnExpr), pos)
	stmts := []ast.StatementNode{lhsVarDef}
	innerCounterRef := createQueryCounterRef(cx, &stmts, pos)
	rowAccess := createQueryRowSlotAccess(join.rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = innerCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)
	bindValue := createQueryBindingAssignment(join.binding, createQueryRowSlotAccess(rowRef, 0, join.binding.valueTy, pos), pos)
	matchCond := &ast.BLangBinaryExpr{
		LhsExpr: createQueryVarRefAt(lhsRef, pos),
		RhsExpr: createQueryRowSlotAccess(rowRef, 1, join.keyTy, pos),
		OpKind:  model.OperatorKind_EQUAL,
	}
	matchCond.SetDeterminedType(semtypes.Boolean)

	if !clause.IsOuterJoinFlag {
		innerBody := []ast.StatementNode{rowVarDef, bindValue, q.ifStmt(matchCond, nextStmts, clause.GetPosition()), createIncrementStmt(innerCounterRef)}
		scanCond := createQueryIntComparison(innerCounterRef, model.OperatorKind_LESS_THAN, join.rowCountRef, pos)
		q.appendWhile(&stmts, scanCond, innerBody, pos, true)
		return stmts
	}

	matchedVarDef, matchedRef := assignToLocal(cx, createBoolLiteral(false, pos), pos)
	stmts = append(stmts, matchedVarDef)
	emitVarDef, emitRef := assignToLocal(cx, createBoolLiteral(false, pos), pos)
	markMatch := []ast.StatementNode{
		createQueryBoolAssignment(matchedRef, pos),
		createQueryBoolAssignment(emitRef, pos),
	}
	realIterationBody := []ast.StatementNode{rowVarDef, bindValue, q.ifStmt(matchCond, markMatch, pos)}
	notMatched := &ast.BLangUnaryExpr{Expr: createQueryVarRefAt(matchedRef, pos), Operator: model.OperatorKind_NOT}
	notMatched.SetDeterminedType(semtypes.Boolean)
	unmatchedBody := []ast.StatementNode{
		createQueryBindingAssignment(join.binding, createQueryNilLiteral(pos), pos),
		createQueryBoolAssignment(emitRef, pos),
	}
	realIterationCond := createQueryIntComparison(createQueryVarRefAt(innerCounterRef, pos), model.OperatorKind_LESS_THAN, createQueryVarRefAt(join.rowCountRef, pos), pos)
	iterationIf := q.ifStmt(realIterationCond, realIterationBody, pos)
	iterationIf.ElseStmt = &ast.BLangBlockStmt{Stmts: []ast.StatementNode{q.ifStmt(notMatched, unmatchedBody, pos)}}
	innerBody := []ast.StatementNode{emitVarDef, iterationIf, q.ifStmt(createQueryVarRefAt(emitRef, pos), nextStmts, pos), createIncrementStmt(innerCounterRef)}
	scanCond := createQueryIntComparison(innerCounterRef, model.OperatorKind_LESS_EQUAL, join.rowCountRef, pos)
	q.appendWhile(&stmts, scanCond, innerBody, pos, true)
	return stmts
}

// collectionSource keeps list and map sources indexable and turns string, XML, stream and
// object:Iterable sources into a receiver pulled by a generated next loop.
func (q *queryLowering) collectionSource(
	out *[]ast.StatementNode,
	collectionExpr ast.BLangActionOrExpression,
	pos diagnostics.Location,
) (queryCollectionSource, bool) {
	cx := q.cx
	tyCtx := cx.typeCtx()
	collectionValue := walkExpression(cx, collectionExpr)
	collectionTy := collectionValue.GetDeterminedType()
	collectionVarDef, collectionRef := assignToLocal(cx, collectionValue, pos)
	*out = append(*out, collectionVarDef)

	switch {
	case semtypes.IsSubtype(tyCtx, collectionTy, semtypes.List):
		rowCountRef, ok := createQueryLengthRef(cx, out, collectionRef, pos)
		return queryCollectionSource{collectionRef: collectionRef, rowCountRef: rowCountRef}, ok
	case semtypes.IsSubtype(tyCtx, collectionTy, semtypes.Mapping):
		keysInvocation := createKeysInvocation(cx, collectionRef)
		if keysInvocation == nil {
			return queryCollectionSource{}, false
		}
		keysVarDef, keysRef := assignToLocal(cx, keysInvocation, pos)
		*out = append(*out, keysVarDef)
		rowCountRef, ok := createQueryLengthRef(cx, out, keysRef, pos)
		return queryCollectionSource{collectionRef: collectionRef, keysRef: keysRef, rowCountRef: rowCountRef}, ok
	case semtypes.IsSubtype(tyCtx, collectionTy, semtypes.Stream):
		return queryCollectionSource{nextReceiverRef: collectionRef, nextReceiverTy: collectionTy}, true
	case semtypes.IsSubtype(tyCtx, collectionTy, semtypes.String),
		semtypes.IsSubtype(tyCtx, collectionTy, semtypes.XML),
		semtypes.IsSubtype(tyCtx, collectionTy, semtypes.Object):
		iteratorInvocation := createIteratorInvocation(cx, collectionRef, collectionTy, pos)
		if iteratorInvocation == nil {
			return queryCollectionSource{}, false
		}
		iteratorVarDef, iteratorRef := assignToLocal(cx, iteratorInvocation, pos)
		*out = append(*out, iteratorVarDef)
		return queryCollectionSource{nextReceiverRef: iteratorRef, nextReceiverTy: iteratorInvocation.GetDeterminedType()}, true
	default:
		cx.internalError("query collection type should have been validated during type resolution", collectionExpr.GetPosition())
		return queryCollectionSource{}, false
	}
}

// createQueryActionNextInvocation gives generated stream next calls the implementor method type;
// BIR still recognizes the actual stream receiver and emits a stream-next instruction.
func createQueryActionNextInvocation(
	cx *functionContext,
	receiver *ast.BLangVarRef,
	receiverTy semtypes.SemType,
) *ast.BLangInvocation {
	methodReceiverTy := receiverTy
	if semtypes.IsSubtype(cx.typeCtx(), receiverTy, semtypes.Stream) {
		valueTy := semtypes.StreamValueType(cx.typeCtx(), receiverTy)
		completionTy := semtypes.StreamCompletionType(cx.typeCtx(), receiverTy)
		methodReceiverTy = semtypes.CreateStreamImplementorType(cx.typeCtx(), valueTy, completionTy)
	}
	return createMethodInvocation(cx, receiver, "next", methodReceiverTy, nil, receiver.GetPosition())
}

// appendCollectionLoop generates an indexed loop for lists and maps or a next loop for the other
// sources. checkState is false for loops that run independently of the pipeline state.
func (q *queryLowering) appendCollectionLoop(
	out *[]ast.StatementNode,
	source queryCollectionSource,
	binding queryRowBinding,
	bodyStmts []ast.StatementNode,
	pos diagnostics.Location,
	checkState bool,
) {
	if source.rowCountRef == nil {
		q.appendIteratorLoop(out, source, binding, bodyStmts, pos, checkState)
		return
	}
	loopCounterRef := createQueryCounterRef(q.cx, out, pos)
	elementAccess := queryElementAccess(source.collectionRef, source.keysRef, loopCounterRef, binding.valueTy)
	loopBody := []ast.StatementNode{createQueryBindingAssignment(binding, elementAccess, pos)}
	loopBody = append(loopBody, bodyStmts...)
	loopBody = append(loopBody, createIncrementStmt(loopCounterRef))
	cond := createQueryIntComparison(loopCounterRef, model.OperatorKind_LESS_THAN, source.rowCountRef, pos)
	q.appendWhile(out, cond, loopBody, pos, checkState)
}

// appendIteratorLoop generates a next loop. Nil marks normal completion, an error becomes the
// query result and fails the pipeline, and records emit frames.
func (q *queryLowering) appendIteratorLoop(
	out *[]ast.StatementNode,
	source queryCollectionSource,
	binding queryRowBinding,
	bodyStmts []ast.StatementNode,
	pos diagnostics.Location,
	checkState bool,
) {
	cx := q.cx
	doneVarDef, doneRef := assignToLocal(cx, createBoolLiteral(false, pos), pos)
	*out = append(*out, doneVarDef)

	nextInvocation := createQueryActionNextInvocation(cx, source.nextReceiverRef, source.nextReceiverTy)
	nextReturnTy := nextInvocation.GetDeterminedType()
	nextVarDef, nextRef := assignToLocal(cx, nextInvocation, pos)
	loopBody := []ast.StatementNode{nextVarDef}
	markDone := createQueryBoolAssignment(doneRef, pos)
	loopBody = append(loopBody, q.ifStmt(createQueryActionTypeTest(nextRef, semtypes.Nil, pos), []ast.StatementNode{markDone}, pos))

	errorTy := semtypes.Intersect(nextReturnTy, semtypes.Error)
	if !semtypes.IsEmpty(cx.typeCtx(), errorTy) {
		errorValueRef := createQueryVarRefAt(nextRef, pos)
		errorValueRef.SetDeterminedType(errorTy)
		firstErrorBody := []ast.StatementNode{
			createQueryResultAssignment(q.resultRef, errorValueRef, pos),
			q.setState(queryStateFailed, pos),
		}
		errorBody := []ast.StatementNode{q.ifStmt(q.stateIs(queryStateRunning, pos), firstErrorBody, pos), markDone}
		loopBody = append(loopBody, q.ifStmt(createQueryActionTypeTest(nextRef, semtypes.Error, pos), errorBody, pos))
	}

	nextValueRef := createQueryVarRefAt(nextRef, pos)
	nextValueRef.SetDeterminedType(semtypes.Mapping)
	valueAccess := &ast.BLangIndexBasedAccess{IndexExpr: createStringLiteral("value", pos)}
	valueAccess.Expr = nextValueRef
	valueAccess.SetDeterminedType(binding.valueTy)
	setPositionIfMissing(valueAccess, pos)
	valueBody := []ast.StatementNode{createQueryBindingAssignment(binding, valueAccess, pos)}
	valueBody = append(valueBody, bodyStmts...)
	loopBody = append(loopBody, q.ifStmt(createQueryActionTypeTest(nextRef, semtypes.Mapping, pos), valueBody, pos))

	notDone := &ast.BLangUnaryExpr{Expr: createQueryVarRefAt(doneRef, pos), Operator: model.OperatorKind_NOT}
	notDone.SetDeterminedType(semtypes.Boolean)
	q.appendWhile(out, notDone, loopBody, pos, checkState)
}

// appendRowsLoop generates a loop over rows materialized by a barrier, restoring each row's
// bindings before running the next segment.
func (q *queryLowering) appendRowsLoop(
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	segmentStmts []ast.StatementNode,
	pos diagnostics.Location,
) bool {
	cx := q.cx
	rowCountRef, ok := createQueryLengthRef(cx, &q.initStmts, rowsRef, pos)
	if !ok {
		return false
	}
	loopCounterRef := createQueryCounterRef(cx, &q.initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)
	loopBody := []ast.StatementNode{rowVarDef}
	loopBody = appendQueryRowRestoreStmts(loopBody, rowRef, bindings, pos)
	loopBody = append(loopBody, segmentStmts...)
	loopBody = append(loopBody, createIncrementStmt(loopCounterRef))
	cond := createQueryIntComparison(loopCounterRef, model.OperatorKind_LESS_THAN, rowCountRef, pos)
	q.appendWhile(&q.initStmts, cond, loopBody, pos, true)
	return true
}

// appendWhile appends `while <cond> && state == running { <body> }`; the state operand is
// omitted for loops that run independently of the pipeline state or when nothing can stop it.
func (q *queryLowering) appendWhile(
	out *[]ast.StatementNode,
	cond ast.BLangExpression,
	bodyStmts []ast.StatementNode,
	pos diagnostics.Location,
	checkState bool,
) {
	if checkState && q.stateRef != nil {
		cond = createQueryAnd(cond, q.stateIs(queryStateRunning, pos), pos)
	}
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(q.cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*out = append(*out, whileStmt)
}

func (q *queryLowering) ifStmt(cond ast.BLangExpression, bodyStmts []ast.StatementNode, pos diagnostics.Location) *ast.BLangIf {
	ifStmt := &ast.BLangIf{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	ifStmt.SetScope(q.cx.currentScope())
	ifStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(ifStmt, pos)
	return ifStmt
}

func (q *queryLowering) pushStmt(listRef *ast.BLangVarRef, valueExpr ast.BLangExpression, pos diagnostics.Location) (ast.StatementNode, bool) {
	pushInvocation := createArrayPushInvocation(q.cx.pkgCtx, listRef, valueExpr)
	if pushInvocation == nil {
		return nil, false
	}
	pushStmt := &ast.BLangExpressionStmt{Expr: pushInvocation}
	setPositionIfMissing(pushStmt, pos)
	return pushStmt, true
}

func (q *queryLowering) bindingFor(varDef *ast.BLangVariableDef) queryRowBinding {
	valueTy := q.cx.symbolType(varDef.Var.Symbol())
	if semtypes.IsZero(valueTy) {
		valueTy = varDef.Var.GetDeterminedType()
	}
	if semtypes.IsZero(valueTy) {
		valueTy = semtypes.Any
	}
	return queryRowBinding{
		varName: varDef.Var.Name,
		symbol:  varDef.Var.Symbol(),
		valueTy: valueTy,
	}
}

// declareBinding declares a query variable once, before the loops that assign it per frame.
func (q *queryLowering) declareBinding(binding queryRowBinding, pos diagnostics.Location) {
	variable := &ast.BLangVariable{Name: binding.varName}
	variable.SetSymbol(binding.symbol)
	variable.Name.SetDeterminedType(semtypes.Never)
	variable.SetDeterminedType(semtypes.Never)
	varDef := &ast.BLangVariableDef{Var: variable}
	varDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(varDef, pos)
	q.initStmts = append(q.initStmts, varDef)
}

func appendQueryBinding(bindings []queryRowBinding, binding queryRowBinding) []queryRowBinding {
	return append(append([]queryRowBinding{}, bindings...), binding)
}

func createQueryActionTypeTest(
	expr *ast.BLangVarRef,
	ty semtypes.SemType,
	pos diagnostics.Location,
) *ast.BLangTypeTestExpr {
	test := &ast.BLangTypeTestExpr{
		Expr: createQueryVarRefAt(expr, pos),
		Type: ast.TypeData{Type: ty},
	}
	test.SetDeterminedType(semtypes.Boolean)
	setPositionIfMissing(test, pos)
	return test
}

func queryVarDefHasBindableSymbol(varDef *ast.BLangVariableDef) bool {
	return varDef != nil &&
		varDef.Var != nil &&
		varDef.Var.Name != nil &&
		varDef.Var.Name.GetValue() != "_" &&
		ast.SymbolIsSet(varDef.Var)
}

func queryGroupOutputBindings(
	cx *functionContext,
	bindings []queryRowBinding,
	groupingSymbols map[model.SymbolRef]bool,
) []queryRowBinding {
	result := make([]queryRowBinding, 0, len(bindings))
	for _, binding := range bindings {
		if groupingSymbols[binding.symbol] {
			result = append(result, binding)
			continue
		}
		binding.valueTy = queryListValueType(cx.typeEnv(), binding.valueTy, true)
		binding.groupAggregated = true
		result = append(result, binding)
	}
	return result
}

func buildQueryGroupScalarFlags(
	bindings []queryRowBinding,
	groupingSymbols map[model.SymbolRef]bool,
	pos diagnostics.Location,
) *ast.BLangListConstructorExpr {
	flags := make([]ast.BLangExpression, 0, len(bindings))
	for _, binding := range bindings {
		flags = append(flags, createBoolLiteral(groupingSymbols[binding.symbol], pos))
	}
	return createQueryListExpr(flags, pos)
}

func queryListValueType(env semtypes.Env, elemTy semtypes.SemType, nonEmpty bool) semtypes.SemType {
	if semtypes.IsZero(elemTy) {
		elemTy = semtypes.Any
	}
	ld := semtypes.NewListDefinition()
	if nonEmpty {
		return ld.Define(env, []semtypes.SemType{elemTy}, semtypes.ListRest(elemTy))
	}
	return ld.Define(env, nil, semtypes.ListRest(elemTy))
}

func createQueryBindingVarRef(binding queryRowBinding) *ast.BLangVarRef {
	return createVarRef(binding.varName, binding.symbol, binding.valueTy)
}

func createQueryBindingAssignment(
	binding queryRowBinding,
	expr ast.BLangActionOrExpression,
	pos diagnostics.Location,
) *ast.BLangAssignment {
	assign := &ast.BLangAssignment{
		VarRef: createQueryBindingVarRef(binding),
		Expr:   expr,
	}
	assign.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assign, pos)
	return assign
}

func createQueryResultAssignment(
	resultRef *ast.BLangVarRef,
	expr ast.BLangActionOrExpression,
	pos diagnostics.Location,
) *ast.BLangAssignment {
	assign := &ast.BLangAssignment{
		VarRef: createQueryVarRefAt(resultRef, pos),
		Expr:   expr,
	}
	assign.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assign, pos)
	return assign
}

// createQueryBoolAssignment creates a typed AST assignment equivalent to ref = true.
func createQueryBoolAssignment(ref *ast.BLangVarRef, pos diagnostics.Location) *ast.BLangAssignment {
	assign := &ast.BLangAssignment{
		VarRef: createQueryVarRefAt(ref, pos),
		Expr:   createBoolLiteral(true, pos),
	}
	assign.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assign, pos)
	return assign
}

func createQueryRowSlotAccess(
	rowExpr ast.BLangExpression,
	slot int,
	valueTy semtypes.SemType,
	pos diagnostics.Location,
) *ast.BLangIndexBasedAccess {
	access := &ast.BLangIndexBasedAccess{
		IndexExpr: createIntLiteral(int64(slot)),
	}
	access.Expr = rowExpr
	access.SetDeterminedType(valueTy)
	setPositionIfMissing(access, pos)
	return access
}

func appendQueryRowRestoreStmts(
	bodyStmts []ast.StatementNode,
	rowRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	pos diagnostics.Location,
) []ast.StatementNode {
	for i, binding := range bindings {
		bodyStmts = append(bodyStmts, createQueryBindingAssignment(
			binding,
			createQueryRowSlotAccess(rowRef, i, binding.valueTy, pos),
			pos,
		))
	}
	return bodyStmts
}

// queryActionControlFlow finds transfers that cross this query action's generated loops.
// Source loops and lambdas own their transfers, while nested query actions propagate through this one.
func queryActionControlFlow(body *ast.BLangBlockStmt) queryActionControlFlowInfo {
	info := queryActionControlFlowInfo{}
	visitor := &queryActionControlFlowVisitor{info: &info}
	ast.Walk(visitor, body)
	return info
}

type queryActionControlFlowInfo struct {
	hasBreak    bool
	hasContinue bool
}

type queryActionControlFlowVisitor struct {
	info *queryActionControlFlowInfo
}

var _ ast.Visitor = &queryActionControlFlowVisitor{}

func (v *queryActionControlFlowVisitor) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	switch node.(type) {
	case *ast.BLangWhile, *ast.BLangForeach, *ast.BLangLambdaFunction:
		return nil
	case *ast.BLangBreak:
		v.info.hasBreak = true
		return nil
	case *ast.BLangContinue:
		v.info.hasContinue = true
		return nil
	}
	return v
}

func (v *queryActionControlFlowVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return nil
}

// queryActionControlFlowState is pushed while a do body with break or continue is desugared.
// loopDepth tells nested loops in the body apart from the generated loops around it.
type queryActionControlFlowState struct {
	loopDepth int
	stateRef  *ast.BLangVarRef
}

// walkQueryActionLoopControl records the transfer in the query state and leaves the innermost
// generated loop with a continue; the loops then unwind through their conditions.
func walkQueryActionLoopControl(
	state *queryActionControlFlowState,
	isBreak bool,
	pos diagnostics.Location,
) desugaredNode[ast.StatementNode] {
	transfer := queryStateContinue
	if isBreak {
		transfer = queryStateBreak
	}
	recordTransfer := &ast.BLangAssignment{
		VarRef: createQueryVarRefAt(state.stateRef, pos),
		Expr:   createIntLiteral(transfer),
	}
	recordTransfer.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(recordTransfer, pos)
	controlStmt := &ast.BLangContinue{}
	controlStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(controlStmt, pos)
	return desugaredNode[ast.StatementNode]{
		initStmts:       []ast.StatementNode{recordTransfer},
		replacementNode: controlStmt,
	}
}

func createQueryRowTupleExpr(
	bindings []queryRowBinding,
	extraExprs []ast.BLangExpression,
	pos diagnostics.Location,
) *ast.BLangListConstructorExpr {
	exprs := make([]ast.BLangExpression, 0, len(bindings)+len(extraExprs))
	for _, binding := range bindings {
		exprs = append(exprs, createQueryBindingVarRef(binding))
	}
	exprs = append(exprs, extraExprs...)
	return createQueryListExpr(exprs, pos)
}

func createQueryListExpr(exprs []ast.BLangExpression, pos diagnostics.Location) *ast.BLangListConstructorExpr {
	listExpr := &ast.BLangListConstructorExpr{Exprs: exprs}
	listExpr.SetDeterminedType(semtypes.List)
	listExpr.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(listExpr, pos)
	return listExpr
}

func createQueryNilLiteral(pos diagnostics.Location) *ast.BLangLiteral {
	nilLit := &ast.BLangLiteral{Value: nil}
	nilLit.SetDeterminedType(semtypes.Nil)
	setPositionIfMissing(nilLit, pos)
	return nilLit
}

func createQueryIntComparison(
	lhs ast.BLangExpression,
	op model.OperatorKind,
	rhs ast.BLangExpression,
	pos diagnostics.Location,
) *ast.BLangBinaryExpr {
	cond := &ast.BLangBinaryExpr{LhsExpr: lhs, RhsExpr: rhs, OpKind: op}
	cond.SetDeterminedType(semtypes.Boolean)
	setPositionIfMissing(cond, pos)
	return cond
}

func createQueryAnd(lhs ast.BLangExpression, rhs ast.BLangExpression, pos diagnostics.Location) *ast.BLangBinaryExpr {
	andExpr := &ast.BLangBinaryExpr{LhsExpr: lhs, RhsExpr: rhs, OpKind: model.OperatorKind_AND}
	andExpr.SetDeterminedType(semtypes.Boolean)
	setPositionIfMissing(andExpr, pos)
	return andExpr
}

func createQueryCounterRef(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) *ast.BLangVarRef {
	counterName, counterSymbol := cx.addDesugardSymbol(semtypes.Int, model.SymbolKindVariable, pos)
	counterVar := &ast.BLangVariable{
		Name: newIdentifier(counterName),
	}
	counterVar.Name.SetDeterminedType(semtypes.Never)
	counterVar.SetDeterminedType(semtypes.Never)
	counterVar.SetInitialExpression(createIntLiteral(0))
	counterVar.SetSymbol(counterSymbol)
	counterVarDef := &ast.BLangVariableDef{Var: counterVar}
	counterVarDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(counterVarDef, pos)
	*initStmts = append(*initStmts, counterVarDef)

	counterRef := &ast.BLangVarRef{VariableName: counterVar.Name}
	counterRef.SetSymbol(counterSymbol)
	counterRef.SetDeterminedType(semtypes.Int)
	setPositionIfMissing(counterRef, pos)
	return counterRef
}

func createQueryLengthRef(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	source ast.BLangExpression,
	pos diagnostics.Location,
) (*ast.BLangVarRef, bool) {
	lengthInvocation := createLengthInvocation(cx, source)
	if lengthInvocation == nil {
		return nil, false
	}
	lengthName, lengthSymbol := cx.addDesugardSymbol(semtypes.Int, model.SymbolKindVariable, pos)
	lengthVar := &ast.BLangVariable{Name: newIdentifier(lengthName)}
	lengthVar.Name.SetDeterminedType(semtypes.Never)
	lengthVar.SetDeterminedType(semtypes.Never)
	lengthVar.SetInitialExpression(lengthInvocation)
	lengthVar.SetSymbol(lengthSymbol)
	lengthVarDef := &ast.BLangVariableDef{Var: lengthVar}
	lengthVarDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(lengthVarDef, pos)
	*initStmts = append(*initStmts, lengthVarDef)
	lengthRef := &ast.BLangVarRef{VariableName: lengthVar.Name}
	lengthRef.SetSymbol(lengthSymbol)
	lengthRef.SetDeterminedType(semtypes.Int)
	return lengthRef, true
}

func createQueryListStore(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) *ast.BLangVarRef {
	listName, listSymbol := cx.addDesugardSymbol(semtypes.List, model.SymbolKindVariable, pos)
	emptyList := createQueryListExpr([]ast.BLangExpression{}, pos)
	listVar := &ast.BLangVariable{Name: newIdentifier(listName)}
	listVar.Name.SetDeterminedType(semtypes.Never)
	listVar.SetDeterminedType(semtypes.Never)
	listVar.SetInitialExpression(emptyList)
	listVar.SetSymbol(listSymbol)
	setPositionIfMissing(listVar, pos)
	listVarDef := &ast.BLangVariableDef{Var: listVar}
	listVarDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(listVarDef, pos)
	*initStmts = append(*initStmts, listVarDef)
	listRef := &ast.BLangVarRef{VariableName: listVar.Name}
	listRef.SetSymbol(listSymbol)
	listRef.SetDeterminedType(semtypes.List)
	setPositionIfMissing(listRef, pos)
	return listRef
}

func createQueryMapStore(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) *ast.BLangVarRef {
	mapName, mapSymbol := cx.addDesugardSymbol(semtypes.Mapping, model.SymbolKindVariable, pos)
	emptyMap := &ast.BLangMappingConstructorExpr{Fields: []ast.MappingField{}}
	emptyMap.SetDeterminedType(semtypes.Mapping)
	setPositionIfMissing(emptyMap, pos)
	mapVar := &ast.BLangVariable{Name: newIdentifier(mapName)}
	mapVar.Name.SetDeterminedType(semtypes.Never)
	mapVar.SetDeterminedType(semtypes.Never)
	mapVar.SetInitialExpression(emptyMap)
	mapVar.SetSymbol(mapSymbol)
	setPositionIfMissing(mapVar, pos)
	mapVarDef := &ast.BLangVariableDef{Var: mapVar}
	mapVarDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(mapVarDef, pos)
	*initStmts = append(*initStmts, mapVarDef)
	mapRef := &ast.BLangVarRef{VariableName: mapVar.Name}
	mapRef.SetSymbol(mapSymbol)
	mapRef.SetDeterminedType(semtypes.Mapping)
	setPositionIfMissing(mapRef, pos)
	return mapRef
}

func createQueryVarRefAt(ref *ast.BLangVarRef, pos diagnostics.Location) *ast.BLangVarRef {
	varRef := createVarRef(ref.VariableName, ref.Symbol(), ref.GetDeterminedType())
	setPositionIfMissing(varRef, pos)
	return varRef
}

func createNegativeLimitPanicIf(
	cx *functionContext,
	limitRef *ast.BLangVarRef,
	pos diagnostics.Location,
) *ast.BLangIf {
	zero := createIntLiteral(0)
	setPositionIfMissing(zero, pos)
	negativeCond := createQueryIntComparison(createQueryVarRefAt(limitRef, pos), model.OperatorKind_LESS_THAN, zero, pos)

	panicStmt := &ast.BLangPanic{
		Expr: createErrorWithMessage("limit cannot be negative", pos),
	}
	panicStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(panicStmt, pos)

	negativeLimitIf := &ast.BLangIf{
		Expr: negativeCond,
		Body: ast.BLangBlockStmt{
			Stmts: []ast.StatementNode{panicStmt},
		},
	}
	negativeLimitIf.SetScope(cx.currentScope())
	negativeLimitIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(negativeLimitIf, pos)
	return negativeLimitIf
}

func buildOrderKeyExprs(cx *functionContext, orderByClause *ast.BLangOrderByClause) []ast.BLangExpression {
	keyExprs := make([]ast.BLangExpression, 0, len(orderByClause.OrderByKeyList))
	for i := range orderByClause.OrderByKeyList {
		keyResult := walkExpression(cx, orderByClause.OrderByKeyList[i].Expression)
		keyExprs = append(keyExprs, keyResult.(ast.BLangExpression))
	}
	return keyExprs
}

func buildOrderDirectionExpr(orderByClause *ast.BLangOrderByClause, pos diagnostics.Location) *ast.BLangListConstructorExpr {
	directions := make([]ast.BLangExpression, 0, len(orderByClause.OrderByKeyList))
	for i := range orderByClause.OrderByKeyList {
		directions = append(directions, createBoolLiteral(!orderByClause.OrderByKeyList[i].IsDescending, pos))
	}
	return createQueryListExpr(directions, pos)
}

func queryElementAccess(
	collRef ast.BLangExpression,
	keysRef *ast.BLangVarRef,
	indexExpr ast.BLangExpression,
	elementTy semtypes.SemType,
) ast.BLangExpression {
	if keysRef == nil {
		listAccess := &ast.BLangIndexBasedAccess{
			IndexExpr: indexExpr,
		}
		listAccess.Expr = collRef
		listAccess.SetDeterminedType(elementTy)
		return listAccess
	}
	keyAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: indexExpr,
	}
	keyAccess.Expr = keysRef
	keyAccess.SetDeterminedType(semtypes.String)
	mapAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: keyAccess,
	}
	mapAccess.Expr = collRef
	mapAccess.SetDeterminedType(elementTy)
	return mapAccess
}

func createIntLiteral(value int64) *ast.BLangNumericLiteral {
	lit := &ast.BLangNumericLiteral{
		BLangLiteral: ast.BLangLiteral{
			Value:         value,
			OriginalValue: fmt.Sprintf("%d", value),
		},
	}
	lit.SetDeterminedType(semtypes.Int)
	return lit
}

func createBoolLiteral(value bool, pos diagnostics.Location) *ast.BLangLiteral {
	originalValue := "false"
	if value {
		originalValue = "true"
	}
	lit := &ast.BLangLiteral{
		Value:         value,
		OriginalValue: originalValue,
	}
	lit.SetDeterminedType(semtypes.Boolean)
	setPositionIfMissing(lit, pos)
	return lit
}

func createStringLiteral(value string, pos diagnostics.Location) *ast.BLangLiteral {
	lit := &ast.BLangLiteral{
		Value:         value,
		OriginalValue: value,
	}
	lit.SetDeterminedType(semtypes.String)
	setPositionIfMissing(lit, pos)
	return lit
}

func createErrorWithMessage(message string, pos diagnostics.Location) *ast.BLangErrorConstructorExpr {
	errorExpr := &ast.BLangErrorConstructorExpr{
		PositionalArgs: []ast.BLangExpression{
			createStringLiteral(message, pos),
		},
	}
	errorExpr.SetDeterminedType(semtypes.Error)
	setPositionIfMissing(errorExpr, pos)
	return errorExpr
}

func createMapPutAssignment(mapExpr ast.BLangExpression, keyExpr ast.BLangExpression, valueExpr ast.BLangExpression) *ast.BLangAssignment {
	mapAccess := &ast.BLangIndexBasedAccess{
		IndexExpr: keyExpr,
	}
	mapAccess.Expr = mapExpr
	mapAccess.SetDeterminedType(semtypes.Any)
	assign := &ast.BLangAssignment{
		VarRef: mapAccess,
		Expr:   valueExpr,
	}
	assign.SetDeterminedType(semtypes.Never)
	return assign
}

// createQuerySortInvocation sorts the rows in place by their key rows.
func createQuerySortInvocation(
	cx *functionContext,
	keysExpr ast.BLangExpression,
	directionsExpr ast.BLangExpression,
	rowsExpr ast.BLangExpression,
) *ast.BLangInvocation {
	pkgName := langInternalPackageName
	space, ok := cx.getImportedSymbolSpace(pkgName)
	if !ok {
		cx.internalError(pkgName+" symbol space not found", keysExpr.GetPosition())
		return nil
	}
	symbolRef, ok := space.GetSymbol("querySort")
	if !ok {
		cx.internalError(pkgName+":querySort symbol not found", keysExpr.GetPosition())
		return nil
	}
	cx.addImplicitImport(pkgName, ast.BLangImportPackage{
		OrgName:      newIdentifier("ballerina"),
		PkgNameComps: []ast.BLangIdentifier{{Value: "lang"}, {Value: "__internal"}},
		Alias:        newIdentifier(pkgName),
	})
	inv := &ast.BLangInvocation{PkgAlias: newIdentifier(pkgName)}
	inv.Name = newIdentifier("querySort")
	inv.ArgExprs = []ast.BLangExpression{keysExpr, directionsExpr, rowsExpr}
	inv.SetSymbol(symbolRef)
	inv.SetDeterminedType(semtypes.Nil)
	setPositionIfMissing(inv, keysExpr.GetPosition())
	return inv
}

func createQueryGroupInvocation(
	cx *functionContext,
	rowsExpr ast.BLangExpression,
	keysExpr ast.BLangExpression,
	scalarFlagsExpr ast.BLangExpression,
) *ast.BLangInvocation {
	return createLangInternalInvocation(cx, "queryGroup", semtypes.List,
		[]ast.BLangExpression{rowsExpr, keysExpr, scalarFlagsExpr}, rowsExpr.GetPosition())
}

func createQueryCollectInvocation(
	cx *functionContext,
	rowsExpr ast.BLangExpression,
	slotCountExpr ast.BLangExpression,
	flattenFlagsExpr ast.BLangExpression,
) *ast.BLangInvocation {
	return createLangInternalInvocation(cx, "queryCollect", semtypes.List,
		[]ast.BLangExpression{rowsExpr, slotCountExpr, flattenFlagsExpr}, rowsExpr.GetPosition())
}

func createLangInternalInvocation(
	cx *functionContext,
	name string,
	returnTy semtypes.SemType,
	args []ast.BLangExpression,
	pos diagnostics.Location,
) *ast.BLangInvocation {
	pkgName := langInternalPackageName
	space, _ := cx.getImportedSymbolSpace(pkgName)
	symbolRef, _ := space.GetSymbol(name)
	cx.addImplicitImport(pkgName, ast.BLangImportPackage{
		OrgName:      newIdentifier("ballerina"),
		PkgNameComps: []ast.BLangIdentifier{{Value: "lang"}, {Value: "__internal"}},
		Alias:        newIdentifier(pkgName),
	})
	inv := &ast.BLangInvocation{PkgAlias: newIdentifier(pkgName)}
	inv.Name = newIdentifier(name)
	inv.ArgExprs = args
	inv.SetSymbol(symbolRef)
	inv.SetDeterminedType(returnTy)
	setPositionIfMissing(inv, pos)
	return inv
}

func createPushInvocation(cx *functionContext, listExpr ast.BLangExpression, valueExpr ast.BLangExpression) *ast.BLangInvocation {
	pkgName := "lang.array"
	space, ok := cx.getImportedSymbolSpace(pkgName)
	if !ok {
		cx.internalError(pkgName+" symbol space not found", listExpr.GetPosition())
		return nil
	}
	symbolRef, ok := space.GetSymbol("push")
	if !ok {
		cx.internalError(pkgName+":push symbol not found", listExpr.GetPosition())
		return nil
	}
	cx.addImplicitImport(pkgName, ast.BLangImportPackage{
		OrgName:      newIdentifier("ballerina"),
		PkgNameComps: []ast.BLangIdentifier{{Value: "lang"}, {Value: "array"}},
		Alias:        newIdentifier(pkgName),
	})
	inv := &ast.BLangInvocation{PkgAlias: newIdentifier(pkgName)}
	inv.Name = newIdentifier("push")
	inv.ArgExprs = []ast.BLangExpression{listExpr, valueExpr}
	inv.SetSymbol(symbolRef)
	inv.SetDeterminedType(semtypes.Nil)
	setPositionIfMissing(inv, listExpr.GetPosition())
	return inv
}
