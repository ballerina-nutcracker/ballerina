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

func walkQueryExpr(cx *functionContext, expr *ast.BLangQueryExpr) desugaredNode[ast.BLangActionOrExpression] {
	fromClause := expr.QueryClauseList[0].(*ast.BLangFromClause)

	finalClauseIndex := len(expr.QueryClauseList) - 1
	var onConflictClause *ast.BLangOnConflictClause
	if clause, isOnConflict := expr.QueryClauseList[finalClauseIndex].(*ast.BLangOnConflictClause); isOnConflict {
		onConflictClause = clause
		finalClauseIndex--
	}

	var (
		selectClause  *ast.BLangSelectClause
		collectClause *ast.BLangCollectClause
	)
	if clause, ok := expr.QueryClauseList[finalClauseIndex].(*ast.BLangSelectClause); ok {
		selectClause = clause
	} else {
		collectClause = expr.QueryClauseList[finalClauseIndex].(*ast.BLangCollectClause)
	}
	fromCollectionTy := fromClause.Collection.GetDeterminedType()
	if collectClause != nil ||
		semtypes.IsSubtype(cx.typeCtx(), fromCollectionTy, semtypes.Union(semtypes.String, semtypes.XML)) ||
		queryExprNeedsRowPipeline(expr, 1, finalClauseIndex) {
		return walkQueryExprWithRows(cx, expr, fromClause, selectClause, collectClause, finalClauseIndex, onConflictClause)
	}
	orderByClauseIndices := queryOrderByClauseIndices(expr, 1, finalClauseIndex)

	queryTy := expr.GetDeterminedType()
	basePos := expr.GetPosition()

	var initStmts []ast.StatementNode
	collRef, keysRef, lenRef, ok := createQueryCollectionSource(cx, &initStmts, fromClause.Collection, basePos)
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}

	resultName, resultSymbol := cx.addDesugardSymbol(queryTy, model.SymbolKindVariable, basePos)
	resultVar := &ast.BLangVariable{
		Name: newIdentifier(resultName),
	}
	resultVar.Name.SetDeterminedType(semtypes.Never)
	resultVar.SetDeterminedType(semtypes.Never)
	switch expr.QueryConstructType {
	case ast.TypeKindMap:
		emptyMap := &ast.BLangMappingConstructorExpr{
			Fields: []ast.MappingField{},
		}
		emptyMap.SetDeterminedType(semtypes.Intersect(queryTy, semtypes.Mapping))
		resultVar.SetInitialExpression(emptyMap)
	default:
		emptyList := &ast.BLangListConstructorExpr{
			Exprs: []ast.BLangExpression{},
		}
		emptyList.SetDeterminedType(semtypes.List)
		emptyList.AtomicType = semtypes.ListAtomicInner
		resultVar.SetInitialExpression(emptyList)
	}
	resultVar.SetSymbol(resultSymbol)
	resultVarDef := &ast.BLangVariableDef{Var: resultVar}
	resultVarDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(resultVarDef, basePos)
	initStmts = append(initStmts, resultVarDef)

	resultRef := &ast.BLangVarRef{
		VariableName: resultVar.Name,
	}
	resultRef.SetSymbol(resultSymbol)
	resultRef.SetDeterminedType(queryTy)

	var seenKeysRef *ast.BLangVarRef
	if onConflictClause != nil && expr.QueryConstructType == ast.TypeKindMap {
		seenKeysRef = createQueryMapStore(cx, &initStmts, basePos)
	}

	loopBinding, ok := queryRowBindingFromVarDef(cx, fromClause.VariableDefinitionNode, "from")
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}
	initStmts = append(initStmts, createQueryBindingDeclaration(loopBinding, basePos))
	stageInput := queryOrderStageInput{
		rowCountRef: lenRef,
	}
	stageStart := 1
	for _, orderByClauseIndex := range orderByClauseIndices {
		stageInput, ok = appendQueryOrderByStageStmts(
			cx,
			expr,
			collRef,
			keysRef,
			loopBinding,
			stageStart,
			orderByClauseIndex,
			stageInput,
			&initStmts,
			basePos,
		)
		if !ok {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
		}
		stageStart = orderByClauseIndex + 1
	}
	ok = appendQueryFinalStageStmts(
		cx,
		expr,
		collRef,
		keysRef,
		loopBinding,
		stageStart,
		finalClauseIndex,
		stageInput,
		resultRef,
		selectClause,
		onConflictClause,
		seenKeysRef,
		&initStmts,
		basePos,
	)
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}

	setPositionIfMissing(resultRef, basePos)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: resultRef,
	}
}

func queryExprNeedsRowPipeline(queryExpr *ast.BLangQueryExpr, startClauseIndex int, endClauseIndex int) bool {
	for i := startClauseIndex; i < endClauseIndex; i++ {
		switch queryExpr.QueryClauseList[i].(type) {
		case *ast.BLangFromClause, *ast.BLangJoinClause, *ast.BLangGroupByClause:
			return true
		}
	}
	return false
}

func queryOrderByClauseIndices(queryExpr *ast.BLangQueryExpr, startClauseIndex int, endClauseIndex int) []int {
	indices := make([]int, 0)
	for i := startClauseIndex; i < endClauseIndex; i++ {
		if _, isOrderBy := queryExpr.QueryClauseList[i].(*ast.BLangOrderByClause); isOrderBy {
			indices = append(indices, i)
		}
	}
	return indices
}

type queryLetStore struct {
	binding  queryRowBinding
	storeRef *ast.BLangVarRef
}

type queryRowBinding struct {
	varName         ast.IdentifierNode
	symbol          model.SymbolRef
	valueTy         semtypes.SemType
	groupAggregated bool
}

type queryOrderStageInput struct {
	indexRowsRef  *ast.BLangVarRef
	rowCountRef   *ast.BLangVarRef
	payloadStores []queryLetStore
}

func createQueryCollectionSource(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	collectionExpr ast.BLangActionOrExpression,
	pos diagnostics.Location,
) (*ast.BLangVarRef, *ast.BLangVarRef, *ast.BLangVarRef, bool) {
	collResult := walkExpression(cx, collectionExpr)
	collExpr := collResult
	collTy := collExpr.GetDeterminedType()

	collVarDef, collRef := assignActionOrExpressionToLocal(cx, collExpr, pos)
	*initStmts = append(*initStmts, collVarDef)

	lengthSource := ast.BLangExpression(collRef)
	var keysRef *ast.BLangVarRef
	tyCtx := semtypes.ContextFrom(cx.typeEnv())
	switch {
	case semtypes.IsSubtype(tyCtx, collTy, semtypes.List):
	case semtypes.IsSubtype(tyCtx, collTy, semtypes.Mapping):
		keysInvocation := createKeysInvocation(cx, collRef)
		if keysInvocation == nil {
			return nil, nil, nil, false
		}
		keysVarDef, keysLocalRef := assignToLocal(cx, keysInvocation, pos)
		*initStmts = append(*initStmts, keysVarDef)
		keysRef = keysLocalRef
		lengthSource = keysRef
	default:
		cx.internalError("query collection type should have been validated during type resolution", collectionExpr.GetPosition())
		return nil, nil, nil, false
	}

	lenRef, ok := createQueryLengthRef(cx, initStmts, lengthSource, pos)
	if !ok {
		return nil, nil, nil, false
	}
	return collRef, keysRef, lenRef, true
}

// createQueryPipelineCollectionSource keeps list/map sources indexable and turns string, XML,
// stream, or object:Iterable sources into a receiver pulled by the generated next loop.
func createQueryPipelineCollectionSource(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	collectionExpr ast.BLangActionOrExpression,
	pos diagnostics.Location,
) (queryActionCollectionSource, bool) {
	collectionTy := collectionExpr.GetDeterminedType()
	tyCtx := cx.typeCtx()
	if semtypes.IsSubtype(tyCtx, collectionTy, semtypes.List) ||
		semtypes.IsSubtype(tyCtx, collectionTy, semtypes.Mapping) {
		collectionRef, keysRef, rowCountRef, ok := createQueryCollectionSource(
			cx, initStmts, collectionExpr, pos,
		)
		return queryActionCollectionSource{
			collectionRef: collectionRef,
			keysRef:       keysRef,
			rowCountRef:   rowCountRef,
		}, ok
	}

	collectionResult := walkExpression(cx, collectionExpr)
	collectionValue := collectionResult
	collectionTy = collectionValue.GetDeterminedType()
	collectionVarDef, collectionRef := assignActionOrExpressionToLocal(cx, collectionValue, pos)
	*initStmts = append(*initStmts, collectionVarDef)

	switch {
	case semtypes.IsSubtype(tyCtx, collectionTy, semtypes.Stream):
		return queryActionCollectionSource{
			nextReceiverRef: collectionRef,
			nextReceiverTy:  collectionTy,
		}, true
	case semtypes.IsSubtype(tyCtx, collectionTy, semtypes.String),
		semtypes.IsSubtype(tyCtx, collectionTy, semtypes.XML),
		semtypes.IsSubtype(tyCtx, collectionTy, semtypes.Object):
		iteratorInvocation := createIteratorInvocation(cx, collectionRef, collectionTy, pos)
		if iteratorInvocation == nil {
			return queryActionCollectionSource{}, false
		}
		iteratorVarDef, iteratorRef := assignToLocal(cx, iteratorInvocation, pos)
		*initStmts = append(*initStmts, iteratorVarDef)
		return queryActionCollectionSource{
			nextReceiverRef: iteratorRef,
			nextReceiverTy:  iteratorInvocation.GetDeterminedType(),
		}, true
	default:
		cx.internalError("query action collection type should have been validated during type resolution", collectionExpr.GetPosition())
		return queryActionCollectionSource{}, false
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

func walkQueryExprWithRows(
	cx *functionContext,
	expr *ast.BLangQueryExpr,
	fromClause *ast.BLangFromClause,
	selectClause *ast.BLangSelectClause,
	collectClause *ast.BLangCollectClause,
	finalClauseIndex int,
	onConflictClause *ast.BLangOnConflictClause,
) desugaredNode[ast.BLangActionOrExpression] {
	queryTy := expr.GetDeterminedType()
	basePos := expr.GetPosition()
	var initStmts []ast.StatementNode

	resultName, resultSymbol := cx.addDesugardSymbol(queryTy, model.SymbolKindVariable, basePos)
	resultVar := &ast.BLangVariable{
		Name: newIdentifier(resultName),
	}
	resultVar.Name.SetDeterminedType(semtypes.Never)
	resultVar.SetDeterminedType(semtypes.Never)
	if collectClause == nil {
		switch expr.QueryConstructType {
		case ast.TypeKindMap:
			emptyMap := &ast.BLangMappingConstructorExpr{
				Fields: []ast.MappingField{},
			}
			emptyMap.SetDeterminedType(semtypes.Intersect(queryTy, semtypes.Mapping))
			resultVar.SetInitialExpression(emptyMap)
		default:
			emptyList := &ast.BLangListConstructorExpr{
				Exprs: []ast.BLangExpression{},
			}
			emptyList.SetDeterminedType(semtypes.List)
			emptyList.AtomicType = semtypes.ListAtomicInner
			resultVar.SetInitialExpression(emptyList)
		}
	}
	resultVar.SetSymbol(resultSymbol)
	resultVarDef := &ast.BLangVariableDef{Var: resultVar}
	resultVarDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(resultVarDef, basePos)
	initStmts = append(initStmts, resultVarDef)

	resultRef := &ast.BLangVarRef{
		VariableName: resultVar.Name,
	}
	resultRef.SetSymbol(resultSymbol)
	resultRef.SetDeterminedType(queryTy)

	var seenKeysRef *ast.BLangVarRef
	if onConflictClause != nil && expr.QueryConstructType == ast.TypeKindMap {
		seenKeysRef = createQueryMapStore(cx, &initStmts, basePos)
	}

	rowsRef := createQueryListStore(cx, &initStmts, basePos)
	bindings, ok := appendInitialQueryRows(cx, rowsRef, fromClause, &initStmts, basePos)
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}

	for i := 1; i < finalClauseIndex; i++ {
		switch clause := expr.QueryClauseList[i].(type) {
		case *ast.BLangFromClause:
			bindings, rowsRef, ok = appendQueryFromClauseRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		case *ast.BLangJoinClause:
			bindings, rowsRef, ok = appendQueryJoinClauseRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		case *ast.BLangLetClause:
			bindings, ok = applyQueryLetClauseToRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		case *ast.BLangWhereClause:
			rowsRef, ok = applyQueryWhereClauseToRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		case *ast.BLangGroupByClause:
			bindings, rowsRef, ok = applyQueryGroupByClauseToRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		case *ast.BLangLimitClause:
			rowsRef, ok = applyQueryLimitClauseToRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		case *ast.BLangOrderByClause:
			ok = applyQueryOrderByClauseToRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		default:
			cx.internalError("query clause shape should have been validated during type resolution", clause.GetPosition())
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
		}
		if !ok {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
		}
	}

	if selectClause != nil {
		ok = appendQueryRowsSelectResultStmts(
			cx,
			rowsRef,
			bindings,
			expr,
			resultRef,
			selectClause,
			onConflictClause,
			seenKeysRef,
			basePos,
			&initStmts,
		)
	} else {
		ok = appendQueryRowsCollectResultStmts(
			cx,
			rowsRef,
			bindings,
			resultRef,
			collectClause,
			basePos,
			&initStmts,
		)
	}
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: expr}
	}

	setPositionIfMissing(resultRef, basePos)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: resultRef,
	}
}

// walkQueryAction lowers a query action into setup statements, streaming loops, and a nil-or-error result.
// Frame-local clauses are nested into each loop iteration and end in the do body; order-by and
// group-by clauses materialize rows and start a new streaming segment after the barrier.
func walkQueryAction(cx *functionContext, action *ast.BLangQueryAction) desugaredNode[ast.BLangActionOrExpression] {
	basePos := action.GetPosition()
	var initStmts []ast.StatementNode

	resultTy := action.GetDeterminedType()
	resultName, resultSymbol := cx.addDesugardSymbol(resultTy, model.SymbolKindVariable, basePos)
	resultVar := &ast.BLangVariable{
		Name: &ast.BLangIdentifier{Value: resultName},
	}
	resultVar.SetDeterminedType(resultTy)
	resultVar.SetInitialExpression(createQueryNilLiteral(basePos))
	resultVar.SetSymbol(resultSymbol)
	resultVarDef := &ast.BLangVariableDef{Var: resultVar}
	setPositionIfMissing(resultVarDef, basePos)
	initStmts = append(initStmts, resultVarDef)

	resultRef := &ast.BLangVarRef{VariableName: resultVar.Name}
	resultRef.SetSymbol(resultSymbol)
	resultRef.SetDeterminedType(resultTy)
	setPositionIfMissing(resultRef, basePos)

	if len(action.QueryClauseList) == 0 {
		cx.internalError("query action shape should have been validated during type resolution", action.GetPosition())
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
	}
	fromClause, ok := action.QueryClauseList[0].(*ast.BLangFromClause)
	if !ok {
		cx.internalError("query action must start with a from clause", action.GetPosition())
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
	}
	if action.DoClause == nil || action.DoClause.Body == nil {
		cx.internalError("query action requires a do clause", action.GetPosition())
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
	}

	fromBinding, ok := queryRowBindingFromVarDef(cx, fromClause.VariableDefinitionNode, "from")
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
	}
	initStmts = append(initStmts, createQueryBindingDeclaration(fromBinding, basePos))

	completionState := queryActionCompletionState{resultRef: resultRef}
	controlInfo := queryActionControlFlow(action.DoClause.Body)
	var controlState *queryActionControlFlowState
	if controlInfo.hasBreak || controlInfo.hasContinue {
		if len(cx.loopVarStack) == 0 {
			cx.internalError("query action break/continue must have an enclosing loop", action.GetPosition())
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
		}
		controlState = &queryActionControlFlowState{}
		stopVarDef, stopRef := assignToLocal(cx, createBoolLiteral(false, basePos), basePos)
		initStmts = append(initStmts, stopVarDef)
		controlState.stopRef = stopRef
		completionState.controlStopRef = stopRef
		if controlInfo.hasBreak {
			breakVarDef, breakRef := assignToLocal(cx, createBoolLiteral(false, basePos), basePos)
			initStmts = append(initStmts, breakVarDef)
			controlState.breakRef = breakRef
		}
		if controlInfo.hasContinue {
			continueVarDef, continueRef := assignToLocal(cx, createBoolLiteral(false, basePos), basePos)
			initStmts = append(initStmts, continueVarDef)
			controlState.continueRef = continueRef
		}
	}
	if !semtypes.IsEmpty(cx.typeCtx(), semtypes.Intersect(resultTy, semtypes.Error)) {
		stopVarDef, stopRef := assignToLocal(cx, createBoolLiteral(false, basePos), basePos)
		initStmts = append(initStmts, stopVarDef)
		completionState.stopRef = stopRef
	}

	collectionSource, ok := createQueryPipelineCollectionSource(cx, &initStmts, fromClause.Collection, basePos)
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
	}
	pipelineState, ok := prepareQueryActionPipeline(
		cx, action.QueryClauseList, completionState, &initStmts, basePos,
	)
	if !ok {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
	}

	segmentStart := 1
	bindings := []queryRowBinding{fromBinding}
	var rowsRef *ast.BLangVarRef
	firstSegment := true
	for {
		barrierIndex := nextQueryActionBarrier(action.QueryClauseList, segmentStart)
		segmentEnd := len(action.QueryClauseList)
		if barrierIndex >= 0 {
			segmentEnd = barrierIndex
		}

		var terminal queryActionSegmentTerminal
		var segmentRowsRef *ast.BLangVarRef
		if barrierIndex >= 0 {
			segmentRowsRef = createQueryListStore(cx, &initStmts, basePos)
			terminal = queryActionRowTerminal(cx, segmentRowsRef, basePos)
		} else {
			if controlState != nil {
				cx.pushLoopVar(nil)
				controlState.loopDepth = len(cx.loopVarStack)
				cx.pushQueryActionControl(*controlState)
			}
			doResult := walkBlockStmt(cx, action.DoClause.Body)
			if controlState != nil {
				cx.popQueryActionControl()
				cx.popLoopVar()
			}
			doBlock, isBlock := doResult.replacementNode.(*ast.BLangBlockStmt)
			if !isBlock {
				cx.internalError("query action do clause should desugar to a block statement", action.GetPosition())
				return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
			}
			doStmts := append([]ast.StatementNode{}, doResult.initStmts...)
			doStmts = append(doStmts, doBlock.Stmts...)
			terminal = func([]queryRowBinding) ([]ast.StatementNode, bool) {
				return doStmts, true
			}
		}

		segmentStmts, outputBindings, stopRefs, ok := buildQueryActionSegmentStmts(
			cx,
			action.QueryClauseList,
			segmentStart,
			segmentEnd,
			bindings,
			terminal,
			pipelineState,
			&initStmts,
			basePos,
		)
		if !ok {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
		}

		if firstSegment {
			appendQueryActionCollectionSegment(
				cx, collectionSource, fromBinding, segmentStmts, stopRefs, completionState, &initStmts, basePos,
			)
			firstSegment = false
		} else if !appendQueryActionRowsSegment(cx, rowsRef, bindings, segmentStmts, stopRefs, &initStmts, basePos) {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
		}
		bindings = outputBindings

		if barrierIndex < 0 {
			break
		}

		rowsRef = segmentRowsRef
		switch clause := action.QueryClauseList[barrierIndex].(type) {
		case *ast.BLangOrderByClause:
			ok = applyQueryOrderByClauseToRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		case *ast.BLangGroupByClause:
			bindings, rowsRef, ok = applyQueryGroupByClauseToRows(cx, rowsRef, bindings, clause, basePos, &initStmts)
		default:
			cx.internalError("query action barrier should be order by or group by", action.GetPosition())
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
		}
		if !ok {
			return desugaredNode[ast.BLangActionOrExpression]{replacementNode: action}
		}
		segmentStart = barrierIndex + 1
	}

	if controlState != nil {
		appendQueryActionControlDispatch(cx, controlState, &initStmts, basePos)
	}
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       initStmts,
		replacementNode: resultRef,
	}
}

type queryActionSegmentTerminal func([]queryRowBinding) ([]ast.StatementNode, bool)

type queryActionPipelineState struct {
	limits     map[*ast.BLangLimitClause]queryActionLimitState
	joins      map[*ast.BLangJoinClause]queryActionJoinState
	completion queryActionCompletionState
}

type queryActionCompletionState struct {
	resultRef      *ast.BLangVarRef
	stopRef        *ast.BLangVarRef
	controlStopRef *ast.BLangVarRef
}

// Type resolution rejects stream and object:Iterable sources for query expressions,
// so their iterator segments cannot produce completion errors.
func queryExpressionCompletionState() queryActionCompletionState {
	return queryActionCompletionState{}
}

type queryActionControlFlowState struct {
	loopDepth   int
	stopRef     *ast.BLangVarRef
	breakRef    *ast.BLangVarRef
	continueRef *ast.BLangVarRef
}

type queryActionCollectionSource struct {
	collectionRef   *ast.BLangVarRef
	keysRef         *ast.BLangVarRef
	rowCountRef     *ast.BLangVarRef
	nextReceiverRef *ast.BLangVarRef
	nextReceiverTy  semtypes.SemType
}

type queryActionLimitState struct {
	limitRef   *ast.BLangVarRef
	counterRef *ast.BLangVarRef
	stopRef    *ast.BLangVarRef
}

type queryActionJoinState struct {
	binding     queryRowBinding
	rowsRef     *ast.BLangVarRef
	rowCountRef *ast.BLangVarRef
	keyTy       semtypes.SemType
}

// prepareQueryActionPipeline emits the clause state that must exist before frame processing.
// It evaluates each limit once and caches each join's right-side [value, key] rows in clause order.
func prepareQueryActionPipeline(
	cx *functionContext,
	clauses []ast.BLangNode,
	completion queryActionCompletionState,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) (queryActionPipelineState, bool) {
	state := queryActionPipelineState{
		limits:     make(map[*ast.BLangLimitClause]queryActionLimitState),
		joins:      make(map[*ast.BLangJoinClause]queryActionJoinState),
		completion: completion,
	}
	for i := 1; i < len(clauses); i++ {
		switch clause := clauses[i].(type) {
		case *ast.BLangLimitClause:
			limitState, ok := prepareQueryActionLimit(cx, clause, initStmts)
			if !ok {
				return queryActionPipelineState{}, false
			}
			state.limits[clause] = limitState
		case *ast.BLangJoinClause:
			joinState, ok := prepareQueryActionJoin(cx, clause, completion, initStmts, pos)
			if !ok {
				return queryActionPipelineState{}, false
			}
			state.joins[clause] = joinState
		}
	}
	return state, true
}

// prepareQueryActionLimit generates state equivalent to:
//
//	limit = <expression>; panic if limit < 0; count = 0; stopped = limit == 0
//
// The returned references are used by the segment loop to count emitted frames and stop upstream pulls.
func prepareQueryActionLimit(
	cx *functionContext,
	clause *ast.BLangLimitClause,
	initStmts *[]ast.StatementNode,
) (queryActionLimitState, bool) {
	clausePos := clause.GetPosition()
	limitResult := walkExpression(cx, clause.Expression)
	limitExpr, ok := limitResult.(ast.BLangExpression)
	if !ok {
		cx.internalError("query action limit expression should have been validated during type resolution", clause.GetPosition())
		return queryActionLimitState{}, false
	}
	limitVarDef, limitRef := assignToLocal(cx, limitExpr, clausePos)
	*initStmts = append(*initStmts, limitVarDef, createNegativeLimitPanicIf(cx, limitRef, clausePos))
	counterRef := createQueryCounterRef(cx, initStmts, clausePos)
	stopVarDef, stopRef := assignToLocal(cx, createBoolLiteral(false, clausePos), clausePos)
	*initStmts = append(*initStmts, stopVarDef)

	zeroLimitCond := &ast.BLangBinaryExpr{
		LhsExpr: createQueryVarRefAt(limitRef, clausePos),
		RhsExpr: createIntLiteral(0),
		OpKind:  model.OperatorKind_EQUAL,
	}
	zeroLimitCond.SetDeterminedType(semtypes.Boolean)
	zeroLimitIf := &ast.BLangIf{
		Expr: zeroLimitCond,
		Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{
			createQueryTrueAssignment(stopRef, clausePos),
		}},
	}
	zeroLimitIf.SetScope(cx.currentScope())
	zeroLimitIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(zeroLimitIf, clausePos)
	*initStmts = append(*initStmts, zeroLimitIf)

	return queryActionLimitState{
		limitRef:   limitRef,
		counterRef: counterRef,
		stopRef:    stopRef,
	}, true
}

// prepareQueryActionJoin evaluates the right side once and generates a cache-building loop:
//
//	for each value in <join collection> { rows.push([value, <right key>]); }
//
// Streaming left-side frames later scan this cache instead of reevaluating the join collection or key.
func prepareQueryActionJoin(
	cx *functionContext,
	clause *ast.BLangJoinClause,
	completion queryActionCompletionState,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) (queryActionJoinState, bool) {
	binding, ok := queryRowBindingFromVarDef(cx, clause.VariableDefinitionNode, "join")
	if !ok {
		return queryActionJoinState{}, false
	}
	*initStmts = append(*initStmts, createQueryBindingDeclaration(binding, clause.GetPosition()))
	rowsRef := createQueryListStore(cx, initStmts, pos)
	collectionSource, ok := createQueryPipelineCollectionSource(cx, initStmts, clause.Collection, pos)
	if !ok {
		return queryActionJoinState{}, false
	}
	rhsResult := walkExpression(cx, clause.OnClause.EqualsExpr)
	rhsExpr, ok := rhsResult.(ast.BLangExpression)
	if !ok {
		cx.internalError("query action join key expression should have been validated during type resolution", clause.GetPosition())
		return queryActionJoinState{}, false
	}
	rowTuple := createQueryRowTupleExpr(
		nil,
		[]ast.BLangExpression{createQueryBindingVarRef(binding), rhsExpr},
		pos,
	)
	pushRow := createArrayPushInvocation(cx.pkgCtx, rowsRef, rowTuple)
	if pushRow == nil {
		return queryActionJoinState{}, false
	}
	pushStmt := &ast.BLangExpressionStmt{Expr: pushRow}
	setPositionIfMissing(pushStmt, pos)
	bodyStmts := []ast.StatementNode{pushStmt}
	// Join right sides are initialized independently; the shared completion state only
	// prevents failed pipelines from emitting frames after all join caches are prepared.
	appendQueryActionCollectionSegment(
		cx,
		collectionSource,
		binding,
		bodyStmts,
		nil,
		completion,
		initStmts,
		pos,
	)
	rowCountRef := collectionSource.rowCountRef
	if rowCountRef == nil {
		rowCountRef, ok = createQueryLengthRef(cx, initStmts, rowsRef, pos)
		if !ok {
			return queryActionJoinState{}, false
		}
	}

	keyTy := rhsExpr.GetDeterminedType()
	if semtypes.IsZero(keyTy) {
		keyTy = semtypes.Any
	}
	return queryActionJoinState{
		binding:     binding,
		rowsRef:     rowsRef,
		rowCountRef: rowCountRef,
		keyTy:       keyTy,
	}, true
}

// nextQueryActionBarrier finds the next clause that requires all input frames before it can emit output.
func nextQueryActionBarrier(clauses []ast.BLangNode, start int) int {
	for i := start; i < len(clauses); i++ {
		switch clauses[i].(type) {
		case *ast.BLangOrderByClause, *ast.BLangGroupByClause:
			return i
		}
	}
	return -1
}

// queryActionRowTerminal creates a segment terminal that materializes the current bindings as:
//
//	rows.push([binding1, binding2, ...]);
func queryActionRowTerminal(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	pos diagnostics.Location,
) queryActionSegmentTerminal {
	return func(bindings []queryRowBinding) ([]ast.StatementNode, bool) {
		rowTuple := createQueryRowTupleExpr(bindings, nil, pos)
		pushRow := createArrayPushInvocation(cx.pkgCtx, rowsRef, rowTuple)
		if pushRow == nil {
			return nil, false
		}
		pushStmt := &ast.BLangExpressionStmt{Expr: pushRow}
		setPositionIfMissing(pushStmt, pos)
		return []ast.StatementNode{pushStmt}, true
	}
}

// buildQueryActionSegmentStmts recursively nests frame-local clauses around a terminal operation.
// It generates nested loops for from/join, assignments for let, and conditional blocks for where/limit,
// producing a shape such as: let assignment; if where { for fromValue { <join>; <terminal> } }.
func buildQueryActionSegmentStmts(
	cx *functionContext,
	clauses []ast.BLangNode,
	clauseIndex int,
	endClauseIndex int,
	bindings []queryRowBinding,
	terminal queryActionSegmentTerminal,
	pipelineState queryActionPipelineState,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) ([]ast.StatementNode, []queryRowBinding, []*ast.BLangVarRef, bool) {
	if clauseIndex == endClauseIndex {
		stmts, ok := terminal(bindings)
		return stmts, bindings, queryActionCompletionStopRefs(pipelineState.completion), ok
	}

	switch clause := clauses[clauseIndex].(type) {
	case *ast.BLangFromClause:
		fromBinding, ok := queryRowBindingFromVarDef(cx, clause.VariableDefinitionNode, "from")
		if !ok {
			return nil, nil, nil, false
		}
		*initStmts = append(*initStmts, createQueryBindingDeclaration(fromBinding, clause.GetPosition()))
		newBindings := append(append([]queryRowBinding{}, bindings...), fromBinding)
		nextStmts, outputBindings, stopRefs, ok := buildQueryActionSegmentStmts(
			cx, clauses, clauseIndex+1, endClauseIndex, newBindings, terminal, pipelineState, initStmts, pos,
		)
		if !ok {
			return nil, nil, nil, false
		}
		var fromStmts []ast.StatementNode
		collectionSource, ok := createQueryPipelineCollectionSource(
			cx, &fromStmts, clause.Collection, clause.GetPosition(),
		)
		if !ok {
			return nil, nil, nil, false
		}
		appendQueryActionCollectionSegment(
			cx,
			collectionSource,
			fromBinding,
			nextStmts,
			stopRefs,
			pipelineState.completion,
			&fromStmts,
			clause.GetPosition(),
		)
		return fromStmts, outputBindings, stopRefs, true
	case *ast.BLangLetClause:
		letStmts := make([]ast.StatementNode, 0, len(clause.LetVarDeclarations))
		newBindings := append([]queryRowBinding{}, bindings...)
		for i := range clause.LetVarDeclarations {
			varDef := &clause.LetVarDeclarations[i]
			if varDef.Var == nil || varDef.Var.Expr == nil {
				cx.internalError("query let clause bindings should have been validated during type resolution", clause.GetPosition())
				return nil, nil, nil, false
			}
			binding, ok := queryRowBindingFromVarDef(cx, varDef, "let")
			if !ok {
				return nil, nil, nil, false
			}
			*initStmts = append(*initStmts, createQueryBindingDeclaration(binding, clause.GetPosition()))
			letResult := walkExpression(cx, varDef.Var.Expr)
			letStmts = append(letStmts, createQueryBindingAssignment(
				binding,
				letResult,
				clause.GetPosition(),
			))
			newBindings = append(newBindings, binding)
		}
		nextStmts, outputBindings, stopRefs, ok := buildQueryActionSegmentStmts(
			cx, clauses, clauseIndex+1, endClauseIndex, newBindings, terminal, pipelineState, initStmts, pos,
		)
		return append(letStmts, nextStmts...), outputBindings, stopRefs, ok
	case *ast.BLangWhereClause:
		whereResult := walkExpression(cx, clause.Expression)
		var whereStmts []ast.StatementNode
		whereExpr, isExpression := whereResult.(ast.BLangExpression)
		if !isExpression {
			whereVarDef, whereRef := assignActionOrExpressionToLocal(cx, whereResult, clause.GetPosition())
			whereStmts = append(whereStmts, whereVarDef)
			whereExpr = whereRef
		}
		nextStmts, outputBindings, stopRefs, ok := buildQueryActionSegmentStmts(
			cx, clauses, clauseIndex+1, endClauseIndex, bindings, terminal, pipelineState, initStmts, pos,
		)
		if !ok {
			return nil, nil, nil, false
		}
		whereIf := &ast.BLangIf{
			Expr: whereExpr,
			Body: ast.BLangBlockStmt{Stmts: nextStmts},
		}
		whereIf.SetScope(cx.currentScope())
		whereIf.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(whereIf, clause.GetPosition())
		return append(whereStmts, whereIf), outputBindings, stopRefs, true
	case *ast.BLangLimitClause:
		limitState, exists := pipelineState.limits[clause]
		if !exists {
			cx.internalError("query action limit state should have been prepared", clause.GetPosition())
			return nil, nil, nil, false
		}
		nextStmts, outputBindings, downstreamStops, ok := buildQueryActionSegmentStmts(
			cx, clauses, clauseIndex+1, endClauseIndex, bindings, terminal, pipelineState, initStmts, pos,
		)
		if !ok {
			return nil, nil, nil, false
		}
		withinLimit := &ast.BLangBinaryExpr{
			LhsExpr: createQueryVarRefAt(limitState.counterRef, clause.GetPosition()),
			RhsExpr: createQueryVarRefAt(limitState.limitRef, clause.GetPosition()),
			OpKind:  model.OperatorKind_LESS_THAN,
		}
		withinLimit.SetDeterminedType(semtypes.Boolean)
		reachedLimit := &ast.BLangBinaryExpr{
			LhsExpr: createQueryVarRefAt(limitState.counterRef, clause.GetPosition()),
			RhsExpr: createQueryVarRefAt(limitState.limitRef, clause.GetPosition()),
			OpKind:  model.OperatorKind_GREATER_EQUAL,
		}
		reachedLimit.SetDeterminedType(semtypes.Boolean)
		reachedLimitIf := &ast.BLangIf{
			Expr: reachedLimit,
			Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{
				createQueryTrueAssignment(limitState.stopRef, clause.GetPosition()),
			}},
		}
		reachedLimitIf.SetScope(cx.currentScope())
		reachedLimitIf.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(reachedLimitIf, clause.GetPosition())
		limitBody := []ast.StatementNode{createIncrementStmt(limitState.counterRef), reachedLimitIf}
		limitBody = append(limitBody, nextStmts...)
		limitIf := &ast.BLangIf{
			Expr: withinLimit,
			Body: ast.BLangBlockStmt{Stmts: limitBody},
		}
		limitIf.SetScope(cx.currentScope())
		limitIf.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(limitIf, clause.GetPosition())
		stopRefs := append([]*ast.BLangVarRef{limitState.stopRef}, downstreamStops...)
		return []ast.StatementNode{limitIf}, outputBindings, stopRefs, true
	case *ast.BLangJoinClause:
		joinState, exists := pipelineState.joins[clause]
		if !exists {
			cx.internalError("query action join state should have been prepared", clause.GetPosition())
			return nil, nil, nil, false
		}
		newBindings := append(append([]queryRowBinding{}, bindings...), joinState.binding)
		nextStmts, outputBindings, stopRefs, ok := buildQueryActionSegmentStmts(
			cx, clauses, clauseIndex+1, endClauseIndex, newBindings, terminal, pipelineState, initStmts, pos,
		)
		if !ok {
			return nil, nil, nil, false
		}
		joinStmts := buildStreamingQueryActionJoin(cx, clause, joinState, nextStmts, stopRefs, pos)
		return joinStmts, outputBindings, stopRefs, true
	default:
		cx.internalError("query clause shape should have been validated during type resolution", clause.GetPosition())
		return nil, nil, nil, false
	}
}

// appendQueryActionCollectionSegment generates an indexed loop for lists/maps or a lazy next loop
// for streams and object:Iterable values, stopping when a downstream limit or completion error fires.
func appendQueryActionCollectionSegment(
	cx *functionContext,
	source queryActionCollectionSource,
	binding queryRowBinding,
	segmentStmts []ast.StatementNode,
	stopRefs []*ast.BLangVarRef,
	completion queryActionCompletionState,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) {
	if source.rowCountRef == nil {
		appendQueryActionIteratorSegment(cx, source, binding, segmentStmts, stopRefs, completion, initStmts, pos)
		return
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	elementAccess := queryElementAccess(source.collectionRef, source.keysRef, loopCounterRef, binding.valueTy)
	bodyStmts := []ast.StatementNode{createQueryBindingAssignment(binding, elementAccess, pos)}
	bodyStmts = append(bodyStmts, segmentStmts...)
	bodyStmts = append(bodyStmts, createIncrementStmt(loopCounterRef))
	appendQueryActionWhile(cx, loopCounterRef, source.rowCountRef, bodyStmts, stopRefs, initStmts, pos)
}

// appendQueryActionIteratorSegment generates a lazy next loop. Nil marks normal completion,
// an error becomes the query-action result and stops the whole pipeline, and records emit frames.
func appendQueryActionIteratorSegment(
	cx *functionContext,
	source queryActionCollectionSource,
	binding queryRowBinding,
	segmentStmts []ast.StatementNode,
	stopRefs []*ast.BLangVarRef,
	completion queryActionCompletionState,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) {
	doneVarDef, doneRef := assignToLocal(cx, createBoolLiteral(false, pos), pos)
	*initStmts = append(*initStmts, doneVarDef)

	nextInvocation := createQueryActionNextInvocation(cx, source.nextReceiverRef, source.nextReceiverTy)
	nextReturnTy := nextInvocation.GetDeterminedType()
	nextVarDef, nextRef := assignToLocal(cx, nextInvocation, pos)
	bodyStmts := []ast.StatementNode{nextVarDef}

	nilIf := &ast.BLangIf{
		Expr: createQueryActionTypeTest(nextRef, semtypes.Nil, pos),
		Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{
			createQueryTrueAssignment(doneRef, pos),
		}},
	}
	nilIf.SetScope(cx.currentScope())
	nilIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(nilIf, pos)
	bodyStmts = append(bodyStmts, nilIf)

	errorTy := semtypes.Intersect(nextReturnTy, semtypes.Error)
	if !semtypes.IsEmpty(cx.typeCtx(), errorTy) {
		if completion.resultRef == nil || completion.stopRef == nil {
			cx.internalError("query action completion state is missing for an error-producing iterator", pos)
			return
		}
		errorValueRef := createQueryVarRefAt(nextRef, pos)
		errorValueRef.SetDeterminedType(errorTy)
		resultAssign := &ast.BLangAssignment{
			VarRef: createQueryVarRefAt(completion.resultRef, pos),
			Expr:   errorValueRef,
		}
		resultAssign.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(resultAssign, pos)
		firstErrorCond := &ast.BLangUnaryExpr{
			Expr:     createQueryVarRefAt(completion.stopRef, pos),
			Operator: model.OperatorKind_NOT,
		}
		firstErrorCond.SetDeterminedType(semtypes.Boolean)
		firstErrorIf := &ast.BLangIf{
			Expr: firstErrorCond,
			Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{
				resultAssign,
				createQueryTrueAssignment(completion.stopRef, pos),
			}},
		}
		firstErrorIf.SetScope(cx.currentScope())
		firstErrorIf.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(firstErrorIf, pos)
		errorIf := &ast.BLangIf{
			Expr: createQueryActionTypeTest(nextRef, semtypes.Error, pos),
			Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{
				firstErrorIf,
				createQueryTrueAssignment(doneRef, pos),
			}},
		}
		errorIf.SetScope(cx.currentScope())
		errorIf.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(errorIf, pos)
		bodyStmts = append(bodyStmts, errorIf)
	}

	nextValueRef := createQueryVarRefAt(nextRef, pos)
	nextValueRef.SetDeterminedType(semtypes.Mapping)
	valueAccess := &ast.BLangIndexBasedAccess{IndexExpr: createStringLiteral("value", pos)}
	valueAccess.Expr = nextValueRef
	valueAccess.SetDeterminedType(binding.valueTy)
	setPositionIfMissing(valueAccess, pos)
	valueBody := []ast.StatementNode{createQueryBindingAssignment(binding, valueAccess, pos)}
	valueBody = append(valueBody, segmentStmts...)
	valueIf := &ast.BLangIf{
		Expr: createQueryActionTypeTest(nextRef, semtypes.Mapping, pos),
		Body: ast.BLangBlockStmt{Stmts: valueBody},
	}
	valueIf.SetScope(cx.currentScope())
	valueIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(valueIf, pos)
	bodyStmts = append(bodyStmts, valueIf)

	condition := &ast.BLangUnaryExpr{
		Expr:     createQueryVarRefAt(doneRef, pos),
		Operator: model.OperatorKind_NOT,
	}
	condition.SetDeterminedType(semtypes.Boolean)
	appendQueryActionConditionalWhile(cx, condition, bodyStmts, stopRefs, initStmts, pos)
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

func queryActionCompletionStopRefs(completion queryActionCompletionState) []*ast.BLangVarRef {
	var stopRefs []*ast.BLangVarRef
	if completion.stopRef != nil {
		stopRefs = append(stopRefs, completion.stopRef)
	}
	if completion.controlStopRef != nil {
		stopRefs = append(stopRefs, completion.controlStopRef)
	}
	return stopRefs
}

// appendQueryActionRowsSegment generates a loop over rows materialized by a preceding barrier.
// Each iteration restores the row's bindings before executing the next streaming segment.
func appendQueryActionRowsSegment(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	segmentStmts []ast.StatementNode,
	stopRefs []*ast.BLangVarRef,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) bool {
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return false
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)
	bodyStmts := []ast.StatementNode{rowVarDef}
	bodyStmts = appendQueryRowRestoreStmts(bodyStmts, rowRef, bindings, pos)
	bodyStmts = append(bodyStmts, segmentStmts...)
	bodyStmts = append(bodyStmts, createIncrementStmt(loopCounterRef))
	appendQueryActionWhile(cx, loopCounterRef, rowCountRef, bodyStmts, stopRefs, initStmts, pos)
	return true
}

// appendQueryActionWhile appends a generated while loop with the shape:
//
//	while counter < rowCount && !stop1 && !stop2 { <bodyStmts> }
//
// The caller includes the counter increment in bodyStmts so specialized loops can control progression.
func appendQueryActionWhile(
	cx *functionContext,
	loopCounterRef *ast.BLangVarRef,
	rowCountRef *ast.BLangVarRef,
	bodyStmts []ast.StatementNode,
	stopRefs []*ast.BLangVarRef,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) {
	cond := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	appendQueryActionConditionalWhile(cx, cond, bodyStmts, stopRefs, initStmts, pos)
}

func appendQueryActionConditionalWhile(
	cx *functionContext,
	condition ast.BLangExpression,
	bodyStmts []ast.StatementNode,
	stopRefs []*ast.BLangVarRef,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) {
	loopCond := queryActionLoopCondition(condition, stopRefs, pos)
	whileStmt := &ast.BLangWhile{
		Expr: loopCond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*initStmts = append(*initStmts, whileStmt)
}

// buildStreamingQueryActionJoin generates a nested scan of the cached right-side join rows for one
// left frame. Matching rows execute nextStmts immediately; an outer join performs one extra sentinel
// iteration that binds nil and executes nextStmts when no cached row matched.
func buildStreamingQueryActionJoin(
	cx *functionContext,
	clause *ast.BLangJoinClause,
	state queryActionJoinState,
	nextStmts []ast.StatementNode,
	stopRefs []*ast.BLangVarRef,
	pos diagnostics.Location,
) []ast.StatementNode {
	lhsResult := walkExpression(cx, clause.OnClause.OnExpr)
	lhsVarDef, lhsRef := assignActionOrExpressionToLocal(cx, lhsResult, pos)
	stmts := []ast.StatementNode{lhsVarDef}

	var matchedRef *ast.BLangVarRef
	if clause.IsOuterJoinFlag {
		matchedVarDef, matchedLocalRef := assignToLocal(cx, createBoolLiteral(false, pos), pos)
		stmts = append(stmts, matchedVarDef)
		matchedRef = matchedLocalRef
	}

	var innerSetup []ast.StatementNode
	innerCounterRef := createQueryCounterRef(cx, &innerSetup, pos)
	stmts = append(stmts, innerSetup...)
	rowAccess := createQueryRowSlotAccess(state.rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = innerCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)
	joinValueAccess := createQueryRowSlotAccess(rowRef, 0, state.binding.valueTy, pos)
	joinKeyAccess := createQueryRowSlotAccess(rowRef, 1, state.keyTy, pos)

	if !clause.IsOuterJoinFlag {
		innerBody := []ast.StatementNode{
			rowVarDef,
			createQueryBindingAssignment(state.binding, joinValueAccess, pos),
		}
		matchCond := &ast.BLangBinaryExpr{
			LhsExpr: createQueryVarRefAt(lhsRef, pos),
			RhsExpr: joinKeyAccess,
			OpKind:  model.OperatorKind_EQUAL,
		}
		matchCond.SetDeterminedType(semtypes.Boolean)
		matchIf := &ast.BLangIf{
			Expr: matchCond,
			Body: ast.BLangBlockStmt{Stmts: nextStmts},
		}
		matchIf.SetScope(cx.currentScope())
		matchIf.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(matchIf, clause.GetPosition())
		innerBody = append(innerBody, matchIf, createIncrementStmt(innerCounterRef))
		appendQueryActionWhile(cx, innerCounterRef, state.rowCountRef, innerBody, stopRefs, &stmts, pos)
		return stmts
	}

	emitVarDef, emitRef := assignToLocal(cx, createBoolLiteral(false, pos), pos)
	realIterationBody := []ast.StatementNode{
		rowVarDef,
		createQueryBindingAssignment(state.binding, joinValueAccess, pos),
	}
	matchCond := &ast.BLangBinaryExpr{
		LhsExpr: createQueryVarRefAt(lhsRef, pos),
		RhsExpr: joinKeyAccess,
		OpKind:  model.OperatorKind_EQUAL,
	}
	matchCond.SetDeterminedType(semtypes.Boolean)
	markMatch := []ast.StatementNode{
		createQueryTrueAssignment(matchedRef, pos),
		createQueryTrueAssignment(emitRef, pos),
	}
	matchIf := &ast.BLangIf{Expr: matchCond, Body: ast.BLangBlockStmt{Stmts: markMatch}}
	matchIf.SetScope(cx.currentScope())
	matchIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(matchIf, pos)
	realIterationBody = append(realIterationBody, matchIf)

	notMatched := &ast.BLangUnaryExpr{
		Expr:     createQueryVarRefAt(matchedRef, pos),
		Operator: model.OperatorKind_NOT,
	}
	notMatched.SetDeterminedType(semtypes.Boolean)
	unmatchedBody := []ast.StatementNode{
		createQueryBindingAssignment(state.binding, createQueryNilLiteral(pos), pos),
		createQueryTrueAssignment(emitRef, pos),
	}
	unmatchedIf := &ast.BLangIf{Expr: notMatched, Body: ast.BLangBlockStmt{Stmts: unmatchedBody}}
	unmatchedIf.SetScope(cx.currentScope())
	unmatchedIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(unmatchedIf, pos)

	realIterationCond := &ast.BLangBinaryExpr{
		LhsExpr: createQueryVarRefAt(innerCounterRef, pos),
		RhsExpr: createQueryVarRefAt(state.rowCountRef, pos),
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	realIterationCond.SetDeterminedType(semtypes.Boolean)
	iterationIf := &ast.BLangIf{
		Expr:     realIterationCond,
		Body:     ast.BLangBlockStmt{Stmts: realIterationBody},
		ElseStmt: &ast.BLangBlockStmt{Stmts: []ast.StatementNode{unmatchedIf}},
	}
	iterationIf.SetScope(cx.currentScope())
	iterationIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(iterationIf, pos)

	emitIf := &ast.BLangIf{
		Expr: createQueryVarRefAt(emitRef, pos),
		Body: ast.BLangBlockStmt{Stmts: nextStmts},
	}
	emitIf.SetScope(cx.currentScope())
	emitIf.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(emitIf, pos)

	innerBody := []ast.StatementNode{emitVarDef, iterationIf, emitIf, createIncrementStmt(innerCounterRef)}
	innerCond := &ast.BLangBinaryExpr{
		LhsExpr: innerCounterRef,
		RhsExpr: state.rowCountRef,
		OpKind:  model.OperatorKind_LESS_EQUAL,
	}
	innerCond.SetDeterminedType(semtypes.Boolean)
	loopCond := queryActionLoopCondition(innerCond, stopRefs, pos)
	innerWhile := &ast.BLangWhile{Expr: loopCond, Body: ast.BLangBlockStmt{Stmts: innerBody}}
	innerWhile.SetScope(cx.currentScope())
	innerWhile.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(innerWhile, pos)
	return append(stmts, innerWhile)
}

// queryActionLoopCondition extends a loop condition with one !stopped operand per downstream limit.
func queryActionLoopCondition(
	baseCond ast.BLangExpression,
	stopRefs []*ast.BLangVarRef,
	pos diagnostics.Location,
) ast.BLangExpression {
	cond := baseCond
	for _, stopRef := range stopRefs {
		notStopped := &ast.BLangUnaryExpr{
			Expr:     createQueryVarRefAt(stopRef, pos),
			Operator: model.OperatorKind_NOT,
		}
		notStopped.SetDeterminedType(semtypes.Boolean)
		andExpr := &ast.BLangBinaryExpr{
			LhsExpr: cond,
			RhsExpr: notStopped,
			OpKind:  model.OperatorKind_AND,
		}
		andExpr.SetDeterminedType(semtypes.Boolean)
		cond = andExpr
	}
	return cond
}

// createQueryTrueAssignment creates a typed AST assignment equivalent to ref = true.
func createQueryTrueAssignment(
	ref *ast.BLangVarRef,
	pos diagnostics.Location,
) *ast.BLangAssignment {
	assign := &ast.BLangAssignment{
		VarRef: createQueryVarRefAt(ref, pos),
		Expr:   createBoolLiteral(true, pos),
	}
	assign.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assign, pos)
	return assign
}

func appendInitialQueryRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	fromClause *ast.BLangFromClause,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) ([]queryRowBinding, bool) {
	loopBinding, ok := queryRowBindingFromVarDef(cx, fromClause.VariableDefinitionNode, "from")
	if !ok {
		return nil, false
	}
	*initStmts = append(*initStmts, createQueryBindingDeclaration(loopBinding, pos))
	collectionSource, ok := createQueryPipelineCollectionSource(cx, initStmts, fromClause.Collection, pos)
	if !ok {
		return nil, false
	}

	rowTuple := createQueryRowTupleExpr(
		nil,
		[]ast.BLangExpression{createQueryBindingVarRef(loopBinding)},
		pos,
	)
	pushRow := createArrayPushInvocation(cx.pkgCtx, rowsRef, rowTuple)
	if pushRow == nil {
		return nil, false
	}
	pushStmt := &ast.BLangExpressionStmt{Expr: pushRow}
	setPositionIfMissing(pushStmt, pos)

	appendQueryActionCollectionSegment(
		cx, collectionSource, loopBinding, []ast.StatementNode{pushStmt}, nil,
		queryExpressionCompletionState(), initStmts, pos,
	)

	return []queryRowBinding{loopBinding}, true
}

// appendQueryFromClauseRows expands every input row with every value produced by an intermediate
// from clause. The collection setup and inner loop are generated inside the outer row loop because
// the collection expression can reference bindings restored from the current input row.
func appendQueryFromClauseRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	clause *ast.BLangFromClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) ([]queryRowBinding, *ast.BLangVarRef, bool) {
	fromBinding, ok := queryRowBindingFromVarDef(cx, clause.VariableDefinitionNode, "from")
	if !ok {
		return nil, nil, false
	}
	*initStmts = append(*initStmts, createQueryBindingDeclaration(fromBinding, clause.GetPosition()))
	newRowsRef := createQueryListStore(cx, initStmts, pos)
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return nil, nil, false
	}
	outerCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = outerCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	outerBody := []ast.StatementNode{rowVarDef}
	outerBody = appendQueryRowRestoreStmts(outerBody, rowRef, bindings, pos)
	collectionSource, ok := createQueryPipelineCollectionSource(
		cx, &outerBody, clause.Collection, clause.GetPosition(),
	)
	if !ok {
		return nil, nil, false
	}
	rowTuple := createQueryRowTupleExpr(
		bindings,
		[]ast.BLangExpression{createQueryBindingVarRef(fromBinding)},
		pos,
	)
	pushRow := createArrayPushInvocation(cx.pkgCtx, newRowsRef, rowTuple)
	if pushRow == nil {
		return nil, nil, false
	}
	pushStmt := &ast.BLangExpressionStmt{Expr: pushRow}
	setPositionIfMissing(pushStmt, clause.GetPosition())
	appendQueryActionCollectionSegment(
		cx, collectionSource, fromBinding, []ast.StatementNode{pushStmt}, nil,
		queryExpressionCompletionState(), &outerBody, clause.GetPosition(),
	)
	outerBody = append(outerBody, createIncrementStmt(outerCounterRef))

	outerCond := &ast.BLangBinaryExpr{
		LhsExpr: outerCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	outerCond.SetDeterminedType(semtypes.Boolean)
	outerWhile := &ast.BLangWhile{
		Expr: outerCond,
		Body: ast.BLangBlockStmt{Stmts: outerBody},
	}
	outerWhile.SetScope(cx.currentScope())
	outerWhile.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(outerWhile, clause.GetPosition())
	*initStmts = append(*initStmts, outerWhile)

	newBindings := append(append([]queryRowBinding{}, bindings...), fromBinding)
	return newBindings, newRowsRef, true
}

func queryRowBindingFromVarDef(
	cx *functionContext,
	variableDefinitionNode *ast.BLangVariableDef,
	clauseName string,
) (queryRowBinding, bool) {
	varDef := variableDefinitionNode
	if varDef.Var == nil || varDef.Var.Symbol().IsEmpty() {
		cx.internalError(fmt.Sprintf(
			"query %s clause binding should have been validated during type resolution",
			clauseName,
		), varDef.GetPosition())
		return queryRowBinding{}, false
	}
	valueTy := cx.symbolType(varDef.Var.Symbol())
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
	}, true
}

func createQueryBindingDeclaration(binding queryRowBinding, pos diagnostics.Location) *ast.BLangVariableDef {
	variable := &ast.BLangVariable{
		Name: binding.varName,
	}
	variable.SetSymbol(binding.symbol)
	variable.Name.SetDeterminedType(semtypes.Never)
	variable.SetDeterminedType(semtypes.Never)
	varDef := &ast.BLangVariableDef{Var: variable}
	varDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(varDef, pos)
	return varDef
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

// walkQueryActionLoopControl stops the generated query loops and records the transfer to perform
// after they unwind. The synthetic continue skips the remainder of the current generated loop body.
func walkQueryActionLoopControl(
	state *queryActionControlFlowState,
	isBreak bool,
	pos diagnostics.Location,
) desugaredNode[ast.StatementNode] {
	controlRef := state.continueRef
	if isBreak {
		controlRef = state.breakRef
	}
	controlStmt := &ast.BLangContinue{}
	controlStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(controlStmt, pos)
	return desugaredNode[ast.StatementNode]{
		initStmts: []ast.StatementNode{
			createQueryTrueAssignment(controlRef, pos),
			createQueryTrueAssignment(state.stopRef, pos),
		},
		replacementNode: controlStmt,
	}
}

// appendQueryActionControlDispatch emits the original transfer after all generated query loops.
// Continue is lowered here so a surrounding desugared foreach still advances its loop variable.
func appendQueryActionControlDispatch(
	cx *functionContext,
	state *queryActionControlFlowState,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) {
	if state.breakRef != nil {
		breakStmt := &ast.BLangBreak{}
		breakStmt.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(breakStmt, pos)
		breakResult := walkStatement(cx, breakStmt)
		bodyStmts := append([]ast.StatementNode{}, breakResult.initStmts...)
		bodyStmts = append(bodyStmts, breakResult.replacementNode)
		*initStmts = append(*initStmts, createQueryActionControlIf(cx, state.breakRef, bodyStmts, pos))
	}
	if state.continueRef != nil {
		continueStmt := &ast.BLangContinue{}
		continueStmt.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(continueStmt, pos)
		continueResult := walkStatement(cx, continueStmt)
		bodyStmts := append([]ast.StatementNode{}, continueResult.initStmts...)
		bodyStmts = append(bodyStmts, continueResult.replacementNode)
		*initStmts = append(*initStmts, createQueryActionControlIf(cx, state.continueRef, bodyStmts, pos))
	}
}

func createQueryActionControlIf(
	cx *functionContext,
	conditionRef *ast.BLangVarRef,
	bodyStmts []ast.StatementNode,
	pos diagnostics.Location,
) *ast.BLangIf {
	ifStmt := &ast.BLangIf{
		Expr: createQueryVarRefAt(conditionRef, pos),
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	ifStmt.SetScope(cx.currentScope())
	ifStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(ifStmt, pos)
	return ifStmt
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
	rowTuple := &ast.BLangListConstructorExpr{Exprs: exprs}
	rowTuple.SetDeterminedType(semtypes.List)
	rowTuple.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(rowTuple, pos)
	return rowTuple
}

func createQueryNilLiteral(pos diagnostics.Location) *ast.BLangLiteral {
	nilLit := &ast.BLangLiteral{Value: nil}
	nilLit.SetDeterminedType(semtypes.Nil)
	setPositionIfMissing(nilLit, pos)
	return nilLit
}

func appendModelStatements(bodyStmts []ast.StatementNode, stmts []ast.StatementNode) []ast.StatementNode {
	return append(bodyStmts, stmts...)
}

func applyQueryLetClauseToRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	clause *ast.BLangLetClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) ([]queryRowBinding, bool) {
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return nil, false
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	bodyStmts := []ast.StatementNode{rowVarDef}
	bodyStmts = appendQueryRowRestoreStmts(bodyStmts, rowRef, bindings, pos)

	newBindings := append([]queryRowBinding{}, bindings...)
	for i := range clause.LetVarDeclarations {
		varDef := &clause.LetVarDeclarations[i]
		if varDef.Var == nil || varDef.Var.Expr == nil {
			cx.internalError("query let clause bindings should have been validated during type resolution", varDef.GetPosition())
			return nil, false
		}
		binding, ok := queryRowBindingFromVarDef(cx, varDef, "let")
		if !ok {
			return nil, false
		}
		*initStmts = append(*initStmts, createQueryBindingDeclaration(binding, pos))
		letResult := walkExpression(cx, varDef.Var.Expr)
		bodyStmts = append(bodyStmts, createQueryBindingAssignment(
			binding,
			letResult,
			pos,
		))

		pushLetValue := createArrayPushInvocation(cx.pkgCtx, rowRef, createQueryBindingVarRef(binding))
		if pushLetValue == nil {
			return nil, false
		}
		bodyStmts = append(bodyStmts, &ast.BLangExpressionStmt{Expr: pushLetValue})
		newBindings = append(newBindings, binding)
	}
	bodyStmts = append(bodyStmts, createIncrementStmt(loopCounterRef))

	cond := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*initStmts = append(*initStmts, whileStmt)
	return newBindings, true
}

func applyQueryWhereClauseToRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	clause *ast.BLangWhereClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) (*ast.BLangVarRef, bool) {
	filteredRowsRef := createQueryListStore(cx, initStmts, pos)
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return nil, false
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	bodyStmts := []ast.StatementNode{rowVarDef}
	bodyStmts = appendQueryRowRestoreStmts(bodyStmts, rowRef, bindings, pos)

	whereResult := walkExpression(cx, clause.Expression)

	pushFiltered := createArrayPushInvocation(cx.pkgCtx, filteredRowsRef, rowRef)
	if pushFiltered == nil {
		return nil, false
	}
	pushStmt := &ast.BLangExpressionStmt{Expr: pushFiltered}
	setPositionIfMissing(pushStmt, pos)
	filterIf := &ast.BLangIf{
		Expr: whereResult.(ast.BLangExpression),
		Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{pushStmt}},
	}
	filterIf.SetScope(cx.currentScope())
	filterIf.SetDeterminedType(semtypes.Never)
	bodyStmts = append(bodyStmts, filterIf, createIncrementStmt(loopCounterRef))

	cond := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*initStmts = append(*initStmts, whileStmt)
	return filteredRowsRef, true
}

func applyQueryGroupByClauseToRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	clause *ast.BLangGroupByClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) ([]queryRowBinding, *ast.BLangVarRef, bool) {
	keyedRowsRef := createQueryListStore(cx, initStmts, pos)
	keyRowsRef := createQueryListStore(cx, initStmts, pos)
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return nil, nil, false
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	bodyStmts := []ast.StatementNode{rowVarDef}
	bodyStmts = appendQueryRowRestoreStmts(bodyStmts, rowRef, bindings, pos)

	groupingSymbols := make(map[model.SymbolRef]bool)
	newBindings := append([]queryRowBinding{}, bindings...)
	keyExprs := make([]ast.BLangExpression, 0, len(clause.GroupingKeyList))
	for i := range clause.GroupingKeyList {
		groupingKey := clause.GroupingKeyList[i]
		switch {
		case groupingKey.VariableRef != nil:
			symbol := cx.pkgCtx.compilerCtx.UnnarrowedSymbol(groupingKey.VariableRef.Symbol())
			groupingSymbols[symbol] = true
			keyResult := walkExpression(cx, groupingKey.VariableRef)
			keyExprs = append(keyExprs, keyResult.(ast.BLangExpression))
		case groupingKey.VariableDef != nil:
			varDef := groupingKey.VariableDef
			keyResult := walkExpression(cx, varDef.Var.Expr)
			keyExpr := keyResult.(ast.BLangExpression)
			if queryVarDefHasBindableSymbol(varDef) {
				binding, ok := queryRowBindingFromVarDef(cx, varDef, "group by")
				if !ok {
					return nil, nil, false
				}
				*initStmts = append(*initStmts, createQueryBindingDeclaration(binding, pos))
				bodyStmts = append(bodyStmts, createQueryBindingAssignment(binding, keyExpr, pos))
				pushGroupVar := createPushInvocation(cx, rowRef, createQueryBindingVarRef(binding))
				if pushGroupVar == nil {
					return nil, nil, false
				}
				bodyStmts = append(bodyStmts, &ast.BLangExpressionStmt{Expr: pushGroupVar})
				groupingSymbols[binding.symbol] = true
				newBindings = append(newBindings, binding)
				keyExpr = createQueryBindingVarRef(binding)
			}
			keyExprs = append(keyExprs, keyExpr)
		default:
			cx.internalError("query group by clause keys should have been validated during type resolution", clause.GetPosition())
			return nil, nil, false
		}
	}

	keyTuple := &ast.BLangListConstructorExpr{Exprs: keyExprs}
	keyTuple.SetDeterminedType(semtypes.List)
	keyTuple.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(keyTuple, clause.GetPosition())
	pushKey := createPushInvocation(cx, keyRowsRef, keyTuple)
	pushRow := createPushInvocation(cx, keyedRowsRef, rowRef)
	if pushKey == nil || pushRow == nil {
		return nil, nil, false
	}
	bodyStmts = append(bodyStmts,
		&ast.BLangExpressionStmt{Expr: pushKey},
		&ast.BLangExpressionStmt{Expr: pushRow},
		createIncrementStmt(loopCounterRef),
	)

	cond := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*initStmts = append(*initStmts, whileStmt)

	scalarFlags := buildQueryGroupScalarFlags(newBindings, groupingSymbols, clause.GetPosition())
	groupInvocation := createQueryGroupInvocation(cx, keyedRowsRef, keyRowsRef, scalarFlags)
	if groupInvocation == nil {
		return nil, nil, false
	}
	groupedRowsDef, groupedRowsRef := assignToLocal(cx, groupInvocation, clause.GetPosition())
	*initStmts = append(*initStmts, groupedRowsDef)
	return queryGroupOutputBindings(cx, newBindings, groupingSymbols), groupedRowsRef, true
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
	listExpr := &ast.BLangListConstructorExpr{Exprs: flags}
	listExpr.SetDeterminedType(semtypes.List)
	listExpr.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(listExpr, pos)
	return listExpr
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

func applyQueryLimitClauseToRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	clause *ast.BLangLimitClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) (*ast.BLangVarRef, bool) {
	limitResult := walkExpression(cx, clause.Expression)
	limitExpr := limitResult.(ast.BLangExpression)
	limitVarDef, limitRef := assignToLocal(cx, limitExpr, clause.GetPosition())
	*initStmts = append(*initStmts, limitVarDef)
	*initStmts = append(*initStmts, createNegativeLimitPanicIf(cx, limitRef, clause.GetPosition()))

	limitedRowsRef := createQueryListStore(cx, initStmts, pos)
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return nil, false
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	limitCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	bodyStmts := []ast.StatementNode{rowVarDef}
	bodyStmts = appendQueryRowRestoreStmts(bodyStmts, rowRef, bindings, pos)

	withinLimitCond := &ast.BLangBinaryExpr{
		LhsExpr: limitCounterRef,
		RhsExpr: createQueryVarRefAt(limitRef, pos),
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	withinLimitCond.SetDeterminedType(semtypes.Boolean)
	pushLimited := createArrayPushInvocation(cx.pkgCtx, limitedRowsRef, rowRef)
	if pushLimited == nil {
		return nil, false
	}
	pushStmt := &ast.BLangExpressionStmt{Expr: pushLimited}
	setPositionIfMissing(pushStmt, pos)
	limitBody := ast.BLangBlockStmt{
		Stmts: []ast.StatementNode{
			pushStmt,
			createIncrementStmt(limitCounterRef),
		},
	}
	limitIf := &ast.BLangIf{
		Expr: withinLimitCond,
		Body: limitBody,
	}
	limitIf.SetScope(cx.currentScope())
	limitIf.SetDeterminedType(semtypes.Never)
	bodyStmts = append(bodyStmts, limitIf, createIncrementStmt(loopCounterRef))

	cond := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*initStmts = append(*initStmts, whileStmt)
	return limitedRowsRef, true
}

func applyQueryOrderByClauseToRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	clause *ast.BLangOrderByClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) bool {
	keyRowsRef := createQueryListStore(cx, initStmts, pos)
	indexRowsRef := createQueryListStore(cx, initStmts, pos)
	payloadRef := createQueryListStore(cx, initStmts, pos)
	pushRowsPayload := createArrayPushInvocation(cx.pkgCtx, payloadRef, rowsRef)
	if pushRowsPayload == nil {
		return false
	}
	payloadStmt := &ast.BLangExpressionStmt{Expr: pushRowsPayload}
	setPositionIfMissing(payloadStmt, pos)
	*initStmts = append(*initStmts, payloadStmt)

	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return false
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	bodyStmts := []ast.StatementNode{rowVarDef}
	bodyStmts = appendQueryRowRestoreStmts(bodyStmts, rowRef, bindings, pos)

	keyTuple := buildOrderKeyTupleExpr(cx, clause, pos)
	pushKeys := createArrayPushInvocation(cx.pkgCtx, keyRowsRef, keyTuple)
	pushIndex := createArrayPushInvocation(cx.pkgCtx, indexRowsRef, loopCounterRef)
	if pushKeys == nil || pushIndex == nil {
		return false
	}
	bodyStmts = append(bodyStmts,
		&ast.BLangExpressionStmt{Expr: pushKeys},
		&ast.BLangExpressionStmt{Expr: pushIndex},
		createIncrementStmt(loopCounterRef),
	)

	cond := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*initStmts = append(*initStmts, whileStmt)

	directionsExpr := buildOrderDirectionExpr(clause, pos)
	sortInvocation := createQuerySortInvocation(cx, keyRowsRef, directionsExpr, indexRowsRef, payloadRef)
	if sortInvocation == nil {
		return false
	}
	sortStmt := &ast.BLangExpressionStmt{Expr: sortInvocation}
	setPositionIfMissing(sortStmt, pos)
	*initStmts = append(*initStmts, sortStmt)
	return true
}

func appendQueryJoinClauseRows(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	clause *ast.BLangJoinClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) ([]queryRowBinding, *ast.BLangVarRef, bool) {
	joinBinding, ok := queryRowBindingFromVarDef(cx, clause.VariableDefinitionNode, "join")
	if !ok {
		return nil, nil, false
	}
	*initStmts = append(*initStmts, createQueryBindingDeclaration(joinBinding, pos))

	newRowsRef := createQueryListStore(cx, initStmts, pos)
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return nil, nil, false
	}
	outerCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = outerCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	outerBody := []ast.StatementNode{rowVarDef}
	outerBody = appendQueryRowRestoreStmts(outerBody, rowRef, bindings, pos)

	lhsResult := walkExpression(cx, clause.OnClause.OnExpr)
	lhsVarDef, lhsRef := assignToLocal(cx, lhsResult.(ast.BLangExpression), pos)
	outerBody = append(outerBody, lhsVarDef)

	var matchedRef *ast.BLangVarRef
	if clause.IsOuterJoinFlag {
		matchedVarDef, matchedLocalRef := assignToLocal(cx, createBoolLiteral(false, pos), pos)
		outerBody = append(outerBody, matchedVarDef)
		matchedRef = matchedLocalRef
	}

	var joinSetup []ast.StatementNode
	joinSource, ok := createQueryPipelineCollectionSource(cx, &joinSetup, clause.Collection, pos)
	if !ok {
		return nil, nil, false
	}
	outerBody = appendModelStatements(outerBody, joinSetup)

	var innerBody []ast.StatementNode
	rhsResult := walkExpression(cx, clause.OnClause.EqualsExpr)

	matchCond := &ast.BLangBinaryExpr{
		LhsExpr: createQueryVarRefAt(lhsRef, pos),
		RhsExpr: rhsResult.(ast.BLangExpression),
		OpKind:  model.OperatorKind_EQUAL,
	}
	matchCond.SetDeterminedType(semtypes.Boolean)

	matchBodyStmts := make([]ast.StatementNode, 0, 3)
	if matchedRef != nil {
		markMatched := &ast.BLangAssignment{
			VarRef: createQueryVarRefAt(matchedRef, pos),
			Expr:   createBoolLiteral(true, pos),
		}
		markMatched.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(markMatched, pos)
		matchBodyStmts = append(matchBodyStmts, markMatched)
	}
	matchTuple := createQueryRowTupleExpr(bindings, []ast.BLangExpression{createQueryBindingVarRef(joinBinding)}, pos)
	pushMatch := createArrayPushInvocation(cx.pkgCtx, newRowsRef, matchTuple)
	if pushMatch == nil {
		return nil, nil, false
	}
	pushMatchStmt := &ast.BLangExpressionStmt{Expr: pushMatch}
	setPositionIfMissing(pushMatchStmt, pos)
	matchBodyStmts = append(matchBodyStmts, pushMatchStmt)

	matchIf := &ast.BLangIf{
		Expr: matchCond,
		Body: ast.BLangBlockStmt{Stmts: matchBodyStmts},
	}
	matchIf.SetScope(cx.currentScope())
	matchIf.SetDeterminedType(semtypes.Never)
	innerBody = append(innerBody, matchIf)
	appendQueryActionCollectionSegment(
		cx, joinSource, joinBinding, innerBody, nil,
		queryExpressionCompletionState(), &outerBody, pos,
	)

	if matchedRef != nil {
		notMatched := &ast.BLangUnaryExpr{
			Expr:     createQueryVarRefAt(matchedRef, pos),
			Operator: model.OperatorKind_NOT,
		}
		notMatched.SetDeterminedType(semtypes.Boolean)
		unmatchedTuple := createQueryRowTupleExpr(bindings, []ast.BLangExpression{createQueryNilLiteral(pos)}, pos)
		pushUnmatched := createArrayPushInvocation(cx.pkgCtx, newRowsRef, unmatchedTuple)
		if pushUnmatched == nil {
			return nil, nil, false
		}
		pushUnmatchedStmt := &ast.BLangExpressionStmt{Expr: pushUnmatched}
		setPositionIfMissing(pushUnmatchedStmt, pos)
		notMatchedIf := &ast.BLangIf{
			Expr: notMatched,
			Body: ast.BLangBlockStmt{Stmts: []ast.StatementNode{pushUnmatchedStmt}},
		}
		notMatchedIf.SetScope(cx.currentScope())
		notMatchedIf.SetDeterminedType(semtypes.Never)
		outerBody = append(outerBody, notMatchedIf)
	}

	outerBody = append(outerBody, createIncrementStmt(outerCounterRef))
	outerCond := &ast.BLangBinaryExpr{
		LhsExpr: outerCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	outerCond.SetDeterminedType(semtypes.Boolean)
	outerWhile := &ast.BLangWhile{
		Expr: outerCond,
		Body: ast.BLangBlockStmt{Stmts: outerBody},
	}
	outerWhile.SetScope(cx.currentScope())
	outerWhile.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(outerWhile, pos)
	*initStmts = append(*initStmts, outerWhile)

	newBindings := append(append([]queryRowBinding{}, bindings...), joinBinding)
	return newBindings, newRowsRef, true
}

func appendQueryRowsSelectResultStmts(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	queryExpr *ast.BLangQueryExpr,
	resultRef *ast.BLangVarRef,
	selectClause *ast.BLangSelectClause,
	onConflictClause *ast.BLangOnConflictClause,
	seenKeysRef *ast.BLangVarRef,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) bool {
	rowCountRef, ok := createQueryLengthRef(cx, initStmts, rowsRef, pos)
	if !ok {
		return false
	}
	loopCounterRef := createQueryCounterRef(cx, initStmts, pos)
	rowAccess := createQueryRowSlotAccess(rowsRef, 0, semtypes.List, pos)
	rowAccess.IndexExpr = loopCounterRef
	rowVarDef, rowRef := assignToLocal(cx, rowAccess, pos)

	bodyStmts := []ast.StatementNode{rowVarDef}
	bodyStmts = appendQueryRowRestoreStmts(bodyStmts, rowRef, bindings, pos)

	bodyStmts, ok = appendQuerySelectResultStmts(
		cx,
		queryExpr,
		resultRef,
		selectClause,
		onConflictClause,
		seenKeysRef,
		pos,
		bodyStmts,
	)
	if !ok {
		return false
	}
	bodyStmts = append(bodyStmts, createIncrementStmt(loopCounterRef))

	cond := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	cond.SetDeterminedType(semtypes.Boolean)
	whileStmt := &ast.BLangWhile{
		Expr: cond,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	whileStmt.SetScope(cx.currentScope())
	whileStmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(whileStmt, pos)
	*initStmts = append(*initStmts, whileStmt)
	return true
}

func appendQueryRowsCollectResultStmts(
	cx *functionContext,
	rowsRef *ast.BLangVarRef,
	bindings []queryRowBinding,
	resultRef *ast.BLangVarRef,
	collectClause *ast.BLangCollectClause,
	pos diagnostics.Location,
	initStmts *[]ast.StatementNode,
) bool {
	flattenFlags := buildQueryCollectFlattenFlags(bindings, collectClause.GetPosition())
	collectInvocation := createQueryCollectInvocation(cx, rowsRef, createIntLiteral(int64(len(bindings))), flattenFlags)
	if collectInvocation == nil {
		return false
	}
	collectRowDef, collectRowRef := assignToLocal(cx, collectInvocation, collectClause.GetPosition())
	*initStmts = append(*initStmts, collectRowDef)

	bodyStmts := make([]ast.StatementNode, 0, len(bindings)+2)
	for i, binding := range bindings {
		collectBinding := binding
		if !binding.groupAggregated {
			collectBinding.valueTy = queryListValueType(cx.typeEnv(), binding.valueTy, false)
		}
		bodyStmts = append(bodyStmts, createQueryBindingAssignment(
			collectBinding,
			createQueryRowSlotAccess(collectRowRef, i, collectBinding.valueTy, pos),
			pos,
		))
	}

	collectResult := walkExpression(cx, collectClause.Expression)
	assignResult := &ast.BLangAssignment{
		VarRef: resultRef,
		Expr:   collectResult.(ast.BLangExpression),
	}
	assignResult.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assignResult, collectClause.GetPosition())
	bodyStmts = append(bodyStmts, assignResult)
	*initStmts = append(*initStmts, bodyStmts...)
	return true
}

func buildQueryCollectFlattenFlags(bindings []queryRowBinding, pos diagnostics.Location) *ast.BLangListConstructorExpr {
	flags := make([]ast.BLangExpression, 0, len(bindings))
	for _, binding := range bindings {
		flags = append(flags, createBoolLiteral(binding.groupAggregated, pos))
	}
	listExpr := &ast.BLangListConstructorExpr{Exprs: flags}
	listExpr.SetDeterminedType(semtypes.List)
	listExpr.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(listExpr, pos)
	return listExpr
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

func queryStageBaseIndexExpr(loopCounterRef *ast.BLangVarRef, indexRowsRef *ast.BLangVarRef) ast.BLangExpression {
	if indexRowsRef == nil {
		return loopCounterRef
	}
	rowIndexAccess := &ast.BLangIndexBasedAccess{IndexExpr: loopCounterRef}
	rowIndexAccess.Expr = indexRowsRef
	rowIndexAccess.SetDeterminedType(semtypes.Int)
	return rowIndexAccess
}

func appendQueryOrderByStageStmts(
	cx *functionContext,
	queryExpr *ast.BLangQueryExpr,
	collRef ast.BLangExpression,
	keysRef *ast.BLangVarRef,
	loopBinding queryRowBinding,
	startClauseIndex int,
	orderByClauseIndex int,
	stageInput queryOrderStageInput,
	initStmts *[]ast.StatementNode,
	basePos diagnostics.Location,
) (queryOrderStageInput, bool) {
	orderByClause := queryExpr.QueryClauseList[orderByClauseIndex].(*ast.BLangOrderByClause)

	orderKeyRowsRef := createQueryListStore(cx, initStmts, basePos)
	sortedIndexRowsRef := createQueryListStore(cx, initStmts, basePos)
	newLetStores, ok := createPreOrderLetStores(cx, queryExpr, startClauseIndex, orderByClauseIndex, initStmts, basePos)
	if !ok {
		return queryOrderStageInput{}, false
	}
	stageStores := make([]queryLetStore, 0, len(stageInput.payloadStores)+len(newLetStores))
	for _, store := range stageInput.payloadStores {
		stageStores = append(stageStores, queryLetStore{
			binding:  store.binding,
			storeRef: createQueryListStore(cx, initStmts, basePos),
		})
	}
	stageStores = append(stageStores, newLetStores...)
	payloadRowsRef, ok := createQueryPayloadStore(cx, initStmts, basePos, stageStores)
	if !ok {
		return queryOrderStageInput{}, false
	}

	loopCounterRef := createQueryCounterRef(cx, initStmts, basePos)
	baseIndexExpr := queryStageBaseIndexExpr(loopCounterRef, stageInput.indexRowsRef)
	elementAccess := queryElementAccess(collRef, keysRef, baseIndexExpr, loopBinding.valueTy)

	var bodyStmts []ast.StatementNode
	bodyStmts = append(bodyStmts, createQueryBindingAssignment(loopBinding, elementAccess, basePos))
	for _, store := range stageInput.payloadStores {
		storeAccess := &ast.BLangIndexBasedAccess{IndexExpr: loopCounterRef}
		storeAccess.Expr = store.storeRef
		storeAccess.SetDeterminedType(store.binding.valueTy)
		bodyStmts = append(bodyStmts, createQueryBindingAssignment(store.binding, storeAccess, basePos))
	}
	declaredBindings := make(map[model.SymbolRef]bool, len(stageStores))
	for _, store := range stageStores {
		declaredBindings[store.binding.symbol] = true
	}
	bodyStmts, ok = appendQueryIntermediateClauseStmts(
		cx,
		queryExpr,
		loopCounterRef,
		initStmts,
		bodyStmts,
		startClauseIndex,
		orderByClauseIndex,
		declaredBindings,
	)
	if !ok {
		return queryOrderStageInput{}, false
	}

	keyTuple := buildOrderKeyTupleExpr(cx, orderByClause, basePos)
	if pushKeys := createArrayPushInvocation(cx.pkgCtx, orderKeyRowsRef, keyTuple); pushKeys != nil {
		pushStmt := &ast.BLangExpressionStmt{Expr: pushKeys}
		setPositionIfMissing(pushStmt, basePos)
		bodyStmts = append(bodyStmts, pushStmt)
	} else {
		return queryOrderStageInput{}, false
	}
	if pushIndex := createArrayPushInvocation(cx.pkgCtx, sortedIndexRowsRef, baseIndexExpr); pushIndex != nil {
		pushStmt := &ast.BLangExpressionStmt{Expr: pushIndex}
		setPositionIfMissing(pushStmt, basePos)
		bodyStmts = append(bodyStmts, pushStmt)
	} else {
		return queryOrderStageInput{}, false
	}
	for _, store := range stageStores {
		pushStore := createArrayPushInvocation(cx.pkgCtx, store.storeRef, createQueryBindingVarRef(store.binding))
		if pushStore == nil {
			return queryOrderStageInput{}, false
		}
		pushStmt := &ast.BLangExpressionStmt{Expr: pushStore}
		setPositionIfMissing(pushStmt, basePos)
		bodyStmts = append(bodyStmts, pushStmt)
	}
	bodyStmts = append(bodyStmts, createIncrementStmt(loopCounterRef))

	stageCondition := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: stageInput.rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	stageCondition.SetDeterminedType(semtypes.Boolean)
	stageWhile := &ast.BLangWhile{
		Expr: stageCondition,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	stageWhile.SetScope(cx.currentScope())
	stageWhile.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(stageWhile, basePos)
	*initStmts = append(*initStmts, stageWhile)

	directionsExpr := buildOrderDirectionExpr(orderByClause, basePos)
	sortInvocation := createQuerySortInvocation(cx, orderKeyRowsRef, directionsExpr, sortedIndexRowsRef, payloadRowsRef)
	if sortInvocation == nil {
		return queryOrderStageInput{}, false
	}
	sortStmt := &ast.BLangExpressionStmt{Expr: sortInvocation}
	setPositionIfMissing(sortStmt, basePos)
	*initStmts = append(*initStmts, sortStmt)

	sortedLenRef, ok := createQueryLengthRef(cx, initStmts, sortedIndexRowsRef, basePos)
	if !ok {
		return queryOrderStageInput{}, false
	}
	return queryOrderStageInput{
		indexRowsRef:  sortedIndexRowsRef,
		rowCountRef:   sortedLenRef,
		payloadStores: stageStores,
	}, true
}

func appendQueryFinalStageStmts(
	cx *functionContext,
	queryExpr *ast.BLangQueryExpr,
	collRef ast.BLangExpression,
	keysRef *ast.BLangVarRef,
	loopBinding queryRowBinding,
	startClauseIndex int,
	selectClauseIndex int,
	stageInput queryOrderStageInput,
	resultRef *ast.BLangVarRef,
	selectClause *ast.BLangSelectClause,
	onConflictClause *ast.BLangOnConflictClause,
	seenKeysRef *ast.BLangVarRef,
	initStmts *[]ast.StatementNode,
	basePos diagnostics.Location,
) bool {
	loopCounterRef := createQueryCounterRef(cx, initStmts, basePos)
	baseIndexExpr := queryStageBaseIndexExpr(loopCounterRef, stageInput.indexRowsRef)
	elementAccess := queryElementAccess(collRef, keysRef, baseIndexExpr, loopBinding.valueTy)

	var bodyStmts []ast.StatementNode
	bodyStmts = append(bodyStmts, createQueryBindingAssignment(loopBinding, elementAccess, basePos))
	for _, store := range stageInput.payloadStores {
		storeAccess := &ast.BLangIndexBasedAccess{IndexExpr: loopCounterRef}
		storeAccess.Expr = store.storeRef
		storeAccess.SetDeterminedType(store.binding.valueTy)
		bodyStmts = append(bodyStmts, createQueryBindingAssignment(store.binding, storeAccess, basePos))
	}
	declaredBindings := make(map[model.SymbolRef]bool, len(stageInput.payloadStores))
	for _, store := range stageInput.payloadStores {
		declaredBindings[store.binding.symbol] = true
	}
	var ok bool
	bodyStmts, ok = appendQueryIntermediateClauseStmts(
		cx,
		queryExpr,
		loopCounterRef,
		initStmts,
		bodyStmts,
		startClauseIndex,
		selectClauseIndex,
		declaredBindings,
	)
	if !ok {
		return false
	}
	bodyStmts, ok = appendQuerySelectResultStmts(
		cx,
		queryExpr,
		resultRef,
		selectClause,
		onConflictClause,
		seenKeysRef,
		basePos,
		bodyStmts,
	)
	if !ok {
		return false
	}
	bodyStmts = append(bodyStmts, createIncrementStmt(loopCounterRef))

	finalCondition := &ast.BLangBinaryExpr{
		LhsExpr: loopCounterRef,
		RhsExpr: stageInput.rowCountRef,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	finalCondition.SetDeterminedType(semtypes.Boolean)
	finalWhile := &ast.BLangWhile{
		Expr: finalCondition,
		Body: ast.BLangBlockStmt{Stmts: bodyStmts},
	}
	finalWhile.SetScope(cx.currentScope())
	finalWhile.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(finalWhile, basePos)
	*initStmts = append(*initStmts, finalWhile)
	return true
}

func createQueryListStore(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) *ast.BLangVarRef {
	listName, listSymbol := cx.addDesugardSymbol(semtypes.List, model.SymbolKindVariable, pos)
	emptyList := &ast.BLangListConstructorExpr{Exprs: []ast.BLangExpression{}}
	emptyList.SetDeterminedType(semtypes.List)
	emptyList.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(emptyList, pos)
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

func createPreOrderLetStores(
	cx *functionContext,
	queryExpr *ast.BLangQueryExpr,
	startClauseIndex int,
	endClauseIndex int,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
) ([]queryLetStore, bool) {
	var stores []queryLetStore
	for i := startClauseIndex; i < endClauseIndex; i++ {
		clause, isLet := queryExpr.QueryClauseList[i].(*ast.BLangLetClause)
		if !isLet {
			continue
		}
		for i := range clause.LetVarDeclarations {
			varDef := &clause.LetVarDeclarations[i]
			binding, ok := queryRowBindingFromVarDef(cx, varDef, "let")
			if !ok {
				return nil, false
			}
			*initStmts = append(*initStmts, createQueryBindingDeclaration(binding, pos))
			storeRef := createQueryListStore(cx, initStmts, pos)
			stores = append(stores, queryLetStore{
				binding:  binding,
				storeRef: storeRef,
			})
		}
	}
	return stores, true
}

func createQueryPayloadStore(
	cx *functionContext,
	initStmts *[]ast.StatementNode,
	pos diagnostics.Location,
	letStores []queryLetStore,
) (*ast.BLangVarRef, bool) {
	payloadRef := createQueryListStore(cx, initStmts, pos)
	for _, store := range letStores {
		pushPayload := createArrayPushInvocation(cx.pkgCtx, payloadRef, store.storeRef)
		if pushPayload == nil {
			return nil, false
		}
		pushStmt := &ast.BLangExpressionStmt{Expr: pushPayload}
		setPositionIfMissing(pushStmt, pos)
		*initStmts = append(*initStmts, pushStmt)
	}
	return payloadRef, true
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
	negativeCond := &ast.BLangBinaryExpr{
		LhsExpr: createQueryVarRefAt(limitRef, pos),
		RhsExpr: zero,
		OpKind:  model.OperatorKind_LESS_THAN,
	}
	negativeCond.SetDeterminedType(semtypes.Boolean)
	setPositionIfMissing(negativeCond, pos)

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

func buildOrderKeyTupleExpr(
	cx *functionContext,
	orderByClause *ast.BLangOrderByClause,
	pos diagnostics.Location,
) *ast.BLangListConstructorExpr {
	keyExprs := make([]ast.BLangExpression, 0, len(orderByClause.OrderByKeyList))
	for i := range orderByClause.OrderByKeyList {
		keyResult := walkExpression(cx, orderByClause.OrderByKeyList[i].Expression)
		keyExprs = append(keyExprs, keyResult.(ast.BLangExpression))
	}
	keyTuple := &ast.BLangListConstructorExpr{Exprs: keyExprs}
	keyTuple.SetDeterminedType(semtypes.List)
	keyTuple.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(keyTuple, pos)
	return keyTuple
}

func buildOrderDirectionExpr(orderByClause *ast.BLangOrderByClause, pos diagnostics.Location) *ast.BLangListConstructorExpr {
	directions := make([]ast.BLangExpression, 0, len(orderByClause.OrderByKeyList))
	for i := range orderByClause.OrderByKeyList {
		directions = append(directions, createBoolLiteral(!orderByClause.OrderByKeyList[i].IsDescending, pos))
	}
	listExpr := &ast.BLangListConstructorExpr{Exprs: directions}
	listExpr.SetDeterminedType(semtypes.List)
	listExpr.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(listExpr, pos)
	return listExpr
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

func appendQuerySelectResultStmts(
	cx *functionContext,
	queryExpr *ast.BLangQueryExpr,
	resultRef *ast.BLangVarRef,
	selectClause *ast.BLangSelectClause,
	onConflictClause *ast.BLangOnConflictClause,
	seenKeysRef *ast.BLangVarRef,
	basePos diagnostics.Location,
	bodyStmts []ast.StatementNode,
) ([]ast.StatementNode, bool) {
	selectResult := walkExpression(cx, selectClause.Expression)
	selectExpr, isExpression := selectResult.(ast.BLangExpression)
	if !isExpression {
		selectVarDef, selectRef := assignActionOrExpressionToLocal(cx, selectResult, basePos)
		bodyStmts = append(bodyStmts, selectVarDef)
		selectExpr = selectRef
	}

	switch queryExpr.QueryConstructType {
	case ast.TypeKindMap:
		selectTy := selectExpr.GetDeterminedType()
		pairName, pairSymbol := cx.addDesugardSymbol(selectTy, model.SymbolKindVariable, selectClause.GetPosition())
		pairVar := &ast.BLangVariable{
			Name: newIdentifier(pairName),
		}
		pairVar.Name.SetDeterminedType(semtypes.Never)
		pairVar.SetDeterminedType(semtypes.Never)
		pairVar.SetInitialExpression(selectExpr)
		pairVar.SetSymbol(pairSymbol)
		pairVarDef := &ast.BLangVariableDef{Var: pairVar}
		pairVarDef.SetDeterminedType(semtypes.Never)
		setPositionIfMissing(pairVarDef, basePos)
		bodyStmts = append(bodyStmts, pairVarDef)

		pairRef := &ast.BLangVarRef{
			VariableName: pairVar.Name,
		}
		pairRef.SetSymbol(pairSymbol)
		pairRef.SetDeterminedType(selectTy)

		keyAccess := &ast.BLangIndexBasedAccess{
			IndexExpr: createIntLiteral(0),
		}
		keyAccess.Expr = pairRef
		keyAccess.SetDeterminedType(semtypes.String)

		valueAccess := &ast.BLangIndexBasedAccess{
			IndexExpr: createIntLiteral(1),
		}
		valueAccess.Expr = pairRef
		valueAccess.SetDeterminedType(semtypes.Any)

		if onConflictClause != nil {
			if seenKeysRef == nil {
				cx.internalError("on conflict query lowering requires seen-key map", onConflictClause.GetPosition())
				return nil, false
			}
			seenLookup := &ast.BLangIndexBasedAccess{
				IndexExpr: keyAccess,
			}
			seenLookup.Expr = seenKeysRef
			seenLookup.SetDeterminedType(semtypes.Any)
			conflictCond := &ast.BLangBinaryExpr{
				LhsExpr: seenLookup,
				RhsExpr: createBoolLiteral(true, basePos),
				OpKind:  model.OperatorKind_EQUAL,
			}
			conflictCond.SetDeterminedType(semtypes.Boolean)

			conflictResult := walkExpression(cx, onConflictClause.Expression)
			conflictBody := make([]ast.StatementNode, 0, 2)

			conflictExpr := conflictResult.(ast.BLangExpression)
			conflictTy := conflictExpr.GetDeterminedType()
			conflictName, conflictSymbol := cx.addDesugardSymbol(conflictTy, model.SymbolKindVariable, onConflictClause.GetPosition())
			conflictVar := &ast.BLangVariable{
				Name: newIdentifier(conflictName),
			}
			conflictVar.Name.SetDeterminedType(semtypes.Never)
			conflictVar.SetDeterminedType(semtypes.Never)
			conflictVar.SetInitialExpression(conflictExpr)
			conflictVar.SetSymbol(conflictSymbol)
			conflictVarDef := &ast.BLangVariableDef{Var: conflictVar}
			conflictVarDef.SetDeterminedType(semtypes.Never)
			setPositionIfMissing(conflictVarDef, basePos)
			conflictBody = append(conflictBody, conflictVarDef)

			conflictRef := &ast.BLangVarRef{
				VariableName: conflictVar.Name,
			}
			conflictRef.SetSymbol(conflictSymbol)
			conflictRef.SetDeterminedType(conflictTy)

			isErrorExpr := ast.NewBLangTypeTestExpr(
				basePos,
				conflictRef,
				ast.TypeData{Type: semtypes.Error},
				false,
			)
			isErrorExpr.SetDeterminedType(semtypes.Boolean)

			assignResult := &ast.BLangAssignment{
				VarRef: resultRef,
				Expr:   conflictRef,
			}
			assignResult.SetDeterminedType(semtypes.Never)
			breakStmt := &ast.BLangBreak{}
			breakStmt.SetDeterminedType(semtypes.Never)
			errorBody := ast.BLangBlockStmt{
				Stmts: []ast.StatementNode{assignResult, breakStmt},
			}
			errorIf := &ast.BLangIf{
				Expr: isErrorExpr,
				Body: errorBody,
			}
			errorIf.SetScope(cx.currentScope())
			errorIf.SetDeterminedType(semtypes.Never)
			conflictBody = append(conflictBody, errorIf)

			onConflictIf := &ast.BLangIf{
				Expr: conflictCond,
				Body: ast.BLangBlockStmt{
					Stmts: conflictBody,
				},
			}
			onConflictIf.SetScope(cx.currentScope())
			onConflictIf.SetDeterminedType(semtypes.Never)
			bodyStmts = append(bodyStmts, onConflictIf)

			markSeen := createMapPutAssignment(seenKeysRef, keyAccess, createBoolLiteral(true, basePos))
			setPositionIfMissing(markSeen, basePos)
			bodyStmts = append(bodyStmts, markSeen)
		}

		mapPutStmt := createMapPutAssignment(resultRef, keyAccess, valueAccess)
		setPositionIfMissing(mapPutStmt, basePos)
		bodyStmts = append(bodyStmts, mapPutStmt)
	default:
		pushInvocation := createArrayPushInvocation(cx.pkgCtx, resultRef, selectExpr)
		if pushInvocation == nil {
			return nil, false
		}
		bodyStmts = append(bodyStmts, &ast.BLangExpressionStmt{Expr: pushInvocation})
	}
	return bodyStmts, true
}

func appendQueryIntermediateClauseStmts(
	cx *functionContext,
	queryExpr *ast.BLangQueryExpr,
	idxRef ast.LExpr,
	initStmts *[]ast.StatementNode,
	bodyStmts []ast.StatementNode,
	startClauseIndex int,
	endClauseIndex int,
	declaredBindings map[model.SymbolRef]bool,
) ([]ast.StatementNode, bool) {
	for i := startClauseIndex; i < endClauseIndex; i++ {
		switch clause := queryExpr.QueryClauseList[i].(type) {
		case *ast.BLangLetClause:
			for i := range clause.LetVarDeclarations {
				varDef := &clause.LetVarDeclarations[i]
				if varDef.Var == nil || varDef.Var.Expr == nil {
					cx.internalError("query let clause bindings should have been validated during type resolution", varDef.GetPosition())
					return nil, false
				}
				binding, ok := queryRowBindingFromVarDef(cx, varDef, "let")
				if !ok {
					return nil, false
				}
				if !declaredBindings[binding.symbol] {
					*initStmts = append(*initStmts, createQueryBindingDeclaration(binding, clause.GetPosition()))
					declaredBindings[binding.symbol] = true
				}
				letResult := walkExpression(cx, varDef.Var.Expr)
				bodyStmts = append(bodyStmts, createQueryBindingAssignment(
					binding,
					letResult,
					clause.GetPosition(),
				))
			}
		case *ast.BLangWhereClause:
			whereResult := walkExpression(cx, clause.Expression)
			whereCond := whereResult.(ast.BLangExpression)
			notWhereCond := &ast.BLangUnaryExpr{
				Expr:     whereCond,
				Operator: model.OperatorKind_NOT,
			}
			notWhereCond.SetDeterminedType(semtypes.Boolean)
			continueStmt := &ast.BLangContinue{}
			continueStmt.SetDeterminedType(semtypes.Never)
			skipBody := ast.BLangBlockStmt{
				Stmts: []ast.StatementNode{
					createIncrementStmt(idxRef),
					continueStmt,
				},
			}
			filterIf := &ast.BLangIf{
				Expr: notWhereCond,
				Body: skipBody,
			}
			filterIf.SetScope(cx.currentScope())
			filterIf.SetDeterminedType(semtypes.Never)
			bodyStmts = append(bodyStmts, filterIf)
		case *ast.BLangLimitClause:
			limitPos := clause.GetPosition()
			limitResult := walkExpression(cx, clause.Expression)
			limitExpr := limitResult.(ast.BLangExpression)
			limitVarDef, limitRef := assignToLocal(cx, limitExpr, limitPos)
			*initStmts = append(*initStmts, limitVarDef)
			*initStmts = append(*initStmts, createNegativeLimitPanicIf(cx, limitRef, limitPos))

			limitCounterName, limitCounterSymbol := cx.addDesugardSymbol(semtypes.Int, model.SymbolKindVariable, limitPos)
			limitCounterVar := &ast.BLangVariable{
				Name: newIdentifier(limitCounterName),
			}
			limitCounterVar.Name.SetDeterminedType(semtypes.Never)
			limitCounterVar.SetDeterminedType(semtypes.Never)
			limitCounterVar.SetInitialExpression(createIntLiteral(0))
			limitCounterVar.SetSymbol(limitCounterSymbol)
			limitCounterVarDef := &ast.BLangVariableDef{Var: limitCounterVar}
			limitCounterVarDef.SetDeterminedType(semtypes.Never)
			setPositionIfMissing(limitCounterVarDef, queryExpr.GetPosition())
			*initStmts = append(*initStmts, limitCounterVarDef)

			limitCounterRef := &ast.BLangVarRef{
				VariableName: limitCounterVar.Name,
			}
			limitCounterRef.SetSymbol(limitCounterSymbol)
			limitCounterRef.SetDeterminedType(semtypes.Int)

			reachedLimitCond := &ast.BLangBinaryExpr{
				LhsExpr: limitCounterRef,
				RhsExpr: createQueryVarRefAt(limitRef, limitPos),
				OpKind:  model.OperatorKind_GREATER_EQUAL,
			}
			reachedLimitCond.SetDeterminedType(semtypes.Boolean)

			continueStmt := &ast.BLangContinue{}
			continueStmt.SetDeterminedType(semtypes.Never)
			skipBody := ast.BLangBlockStmt{
				Stmts: []ast.StatementNode{
					createIncrementStmt(idxRef),
					continueStmt,
				},
			}
			limitIf := &ast.BLangIf{
				Expr: reachedLimitCond,
				Body: skipBody,
			}
			limitIf.SetScope(cx.currentScope())
			limitIf.SetDeterminedType(semtypes.Never)
			bodyStmts = append(bodyStmts, limitIf)

			bodyStmts = append(bodyStmts, createIncrementStmt(limitCounterRef))
		case *ast.BLangOrderByClause:
			cx.internalError("query order by clauses should have been split before generic intermediate lowering", clause.GetPosition())
			return nil, false
		default:
			cx.internalError("query clause shape should have been validated during type resolution", clause.GetPosition())
			return nil, false
		}
	}
	return bodyStmts, true
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

func createQuerySortInvocation(
	cx *functionContext,
	keysExpr ast.BLangExpression,
	directionsExpr ast.BLangExpression,
	indicesExpr ast.BLangExpression,
	payloadExpr ast.BLangExpression,
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
	inv.ArgExprs = []ast.BLangExpression{keysExpr, directionsExpr, indicesExpr, payloadExpr}
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
