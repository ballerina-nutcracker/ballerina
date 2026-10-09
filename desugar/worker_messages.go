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

package desugar

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// declareWorkerMessages creates the WorkerMessage of every message the
// workers of a body send, one per send, and records them for the message
// actions lowered after it.
func declareWorkerMessages(cx *functionContext, body *ast.BLangBlockFunctionBody, pos diagnostics.Location) []ast.StatementNode {
	var stmts []ast.StatementNode
	declare := func(messages []model.WorkerMessageRef) {
		for _, message := range messages {
			def, ref := assignToLocal(cx, createLangInternalInvocation(cx, "createWorkerMessage", semtypes.Handle, nil, pos), pos)
			stmts = append(stmts, def)
			cx.workerMessages[message] = ref
		}
	}
	declare(body.DefaultWorkerSendMessages)
	for _, worker := range body.Workers {
		declare(worker.SendMessages)
	}
	return stmts
}

func walkWorkerAsyncSendAction(cx *functionContext, action *ast.BLangWorkerAsyncSendAction) desugaredNode[ast.BLangActionOrExpression] {
	value := walkExpressionInner(cx, action.Expr)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       value.initStmts,
		replacementNode: setWorkerMessageValueInvocation(cx, action.Message, value.replacementNode, action.GetPosition()),
	}
}

// walkWorkerSyncSendAction lowers `v ->> w` to setting the message and then
// waiting until w received it.
func walkWorkerSyncSendAction(cx *functionContext, action *ast.BLangWorkerSyncSendAction) desugaredNode[ast.BLangActionOrExpression] {
	pos := action.GetPosition()
	value := walkExpressionInner(cx, action.Expr)
	set := expressionStatement(setWorkerMessageValueInvocation(cx, action.Message, value.replacementNode, pos), pos)
	args := []ast.BLangExpression{workerMessageRef(cx, action.Message, pos), workerFutureRef(cx, action.Peer.Symbol, pos)}
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       append(value.initStmts, set),
		replacementNode: createLangInternalInvocation(cx, "waitWorkerMessageReceived", action.GetDeterminedType(), args, pos),
	}
}

func walkWorkerReceiveAction(cx *functionContext, action *ast.BLangWorkerReceiveAction) desugaredNode[ast.BLangActionOrExpression] {
	pos := action.GetPosition()
	args := []ast.BLangExpression{workerMessageRef(cx, action.Message, pos), workerFutureRef(cx, action.Peer.Symbol, pos)}
	return desugaredNode[ast.BLangActionOrExpression]{
		replacementNode: createLangInternalInvocation(cx, "getWorkerMessageValue", action.GetDeterminedType(), args, pos),
	}
}

// walkWorkerMultipleReceiveAction lowers `<- {a: w1, b: w2}` to
//
//	(any|error)[]|error $r = getWorkerMessageValues([$m1, $m2], [$w1, $w2]);
//	$r is error ? $r : {a: $r[0], b: $r[1]}
func walkWorkerMultipleReceiveAction(cx *functionContext, action *ast.BLangWorkerMultipleReceiveAction) desugaredNode[ast.BLangActionOrExpression] {
	pos := action.GetPosition()
	messages := make([]ast.BLangExpression, len(action.Fields))
	senders := make([]ast.BLangExpression, len(action.Fields))
	for i, field := range action.Fields {
		messages[i] = workerMessageRef(cx, field.Message, pos)
		senders[i] = workerFutureRef(cx, field.Peer.Symbol, pos)
	}
	valuesTy := semtypes.Union(semtypes.List, semtypes.Error)
	call := createLangInternalInvocation(cx, "getWorkerMessageValues", valuesTy,
		[]ast.BLangExpression{workerListConstructor(messages, pos), workerListConstructor(senders, pos)}, pos)
	valuesDef, valuesRef := assignToLocal(cx, call, pos)

	resultTy := action.GetDeterminedType()
	recordTy := semtypes.Intersect(resultTy, semtypes.Mapping)
	record := &ast.BLangMappingConstructorExpr{Fields: make([]ast.MappingField, len(action.Fields))}
	for i, field := range action.Fields {
		fieldTy := semtypes.MappingMemberTypeInnerVal(cx.typeCtx(), recordTy, semtypes.StringConst(field.FieldName))
		value := &ast.BLangIndexBasedAccess{IndexExpr: createIntLiteral(int64(i))}
		value.Expr = retypedVarRef(valuesRef, semtypes.List)
		value.SetDeterminedType(fieldTy)
		setPositionIfMissing(value, pos)
		key := &ast.BLangMappingKey{Expr: createStringLiteral(field.FieldName, pos), Kind: ast.MappingKeyStringLiteral}
		key.SetDeterminedType(semtypes.Never)
		key.SetPosition(pos)
		kv := &ast.BLangMappingKeyValueField{Key: key, ValueExpr: value}
		kv.SetDeterminedType(semtypes.Never)
		kv.SetPosition(pos)
		record.Fields[i] = kv
	}
	record.AtomicType = *semtypes.ToMappingAtomicType(cx.typeCtx(), recordTy)
	record.SetDeterminedType(recordTy)
	setPositionIfMissing(record, pos)

	isError := &ast.BLangTypeTestExpr{Expr: copyVarRef(valuesRef), Type: ast.TypeData{Type: semtypes.Error}}
	isError.SetDeterminedType(semtypes.Boolean)
	setPositionIfMissing(isError, pos)
	result := &ast.BLangTernaryExpr{
		Condition: isError,
		ThenExpr:  retypedVarRef(valuesRef, semtypes.Error),
		ElseExpr:  record,
	}
	result.SetDeterminedType(resultTy)
	setPositionIfMissing(result, pos)
	return desugaredNode[ast.BLangActionOrExpression]{
		initStmts:       []ast.StatementNode{valuesDef},
		replacementNode: result,
	}
}

// walkWorkerFlushAction lowers a flush to waiting until each peer it covers
// received every message sent to it so far.
func walkWorkerFlushAction(cx *functionContext, action *ast.BLangWorkerFlushAction) desugaredNode[ast.BLangActionOrExpression] {
	pos := action.GetPosition()
	if len(action.Covered) == 0 {
		return desugaredNode[ast.BLangActionOrExpression]{replacementNode: createQueryNilLiteral(pos)}
	}
	groups := make([]ast.BLangExpression, len(action.Covered))
	receivers := make([]ast.BLangExpression, len(action.Covered))
	for i, coverage := range action.Covered {
		messages := make([]ast.BLangExpression, len(coverage.Messages))
		for j, message := range coverage.Messages {
			messages[j] = workerMessageRef(cx, message, pos)
		}
		groups[i] = workerListConstructor(messages, pos)
		receivers[i] = workerFutureRef(cx, coverage.Peer, pos)
	}
	args := []ast.BLangExpression{workerListConstructor(groups, pos), workerListConstructor(receivers, pos)}
	return desugaredNode[ast.BLangActionOrExpression]{
		replacementNode: createLangInternalInvocation(cx, "flushWorkerMessages", action.GetDeterminedType(), args, pos),
	}
}

// asyncSendsOf returns the async sends among a worker's statements, in source
// order. Nested functions have their own workers, so it doesn't descend into
// them.
func asyncSendsOf(stmts []ast.StatementNode) []*ast.BLangWorkerAsyncSendAction {
	collector := &asyncSendCollector{}
	for _, stmt := range stmts {
		ast.Walk(collector, stmt.(ast.BLangNode))
	}
	return collector.sends
}

type asyncSendCollector struct {
	sends []*ast.BLangWorkerAsyncSendAction
}

func (c *asyncSendCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch node := node.(type) {
	case *ast.BLangLambdaFunction, *ast.BLangFunction, *ast.BLangClassDefinition:
		return nil
	case *ast.BLangWorkerAsyncSendAction:
		c.sends = append(c.sends, node)
	}
	return c
}

func (c *asyncSendCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

// awaitDeliveryStatement holds a terminating worker until every async message
// it sent is received.
func awaitDeliveryStatement(cx *functionContext, sends []*ast.BLangWorkerAsyncSendAction, pos diagnostics.Location) ast.StatementNode {
	messages := make([]ast.BLangExpression, len(sends))
	receivers := make([]ast.BLangExpression, len(sends))
	for i, send := range sends {
		messages[i] = workerMessageRef(cx, send.Message, pos)
		receivers[i] = workerFutureRef(cx, send.Peer.Symbol, pos)
	}
	args := []ast.BLangExpression{workerListConstructor(messages, pos), workerListConstructor(receivers, pos)}
	return expressionStatement(createLangInternalInvocation(cx, "awaitWorkerMessageDelivery", semtypes.Nil, args, pos), pos)
}

func workerMessageRef(cx *functionContext, message model.WorkerMessageRef, pos diagnostics.Location) ast.BLangExpression {
	ref, ok := cx.workerMessages[message]
	if !ok {
		cx.internalError("no worker message variable", pos)
		return createQueryNilLiteral(pos)
	}
	return copyVarRef(ref)
}

func workerFutureRef(cx *functionContext, worker model.SymbolRef, pos diagnostics.Location) ast.BLangExpression {
	slot, ok := cx.workerFutureSlots[worker]
	if !ok {
		cx.internalError("no future slot for worker", pos)
		return createQueryNilLiteral(pos)
	}
	return copyVarRef(slot)
}

func retypedVarRef(ref *ast.BLangVarRef, ty semtypes.SemType) *ast.BLangVarRef {
	retyped := copyVarRef(ref)
	retyped.SetDeterminedType(ty)
	return retyped
}

// workerListConstructor builds a list of handles or futures for a
// lang.__internal call.
func workerListConstructor(exprs []ast.BLangExpression, pos diagnostics.Location) *ast.BLangListConstructorExpr {
	list := &ast.BLangListConstructorExpr{Exprs: exprs}
	list.SetDeterminedType(semtypes.List)
	list.AtomicType = semtypes.ListAtomicInner
	setPositionIfMissing(list, pos)
	return list
}

func setWorkerMessageValueInvocation(
	cx *functionContext,
	message model.WorkerMessageRef,
	value ast.BLangActionOrExpression,
	pos diagnostics.Location,
) ast.BLangExpression {
	args := []ast.BLangExpression{workerMessageRef(cx, message, pos), value.(ast.BLangExpression)}
	return createLangInternalInvocation(cx, "setWorkerMessageValue", semtypes.Nil, args, pos)
}
