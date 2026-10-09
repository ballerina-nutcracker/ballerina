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

package types

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func resolveWorkerAsyncSendAction(t typeResolver, chain *binding, e *ast.BLangWorkerAsyncSendAction) (semtypes.SemType, expressionEffect, bool) {
	chain, ok := resolveWorkerMessageSend(t, chain, e.Expr, e.Message, e.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	e.SetDeterminedType(semtypes.Nil)
	return semtypes.Nil, defaultExpressionEffect(chain), true
}

func resolveWorkerSyncSendAction(t typeResolver, chain *binding, e *ast.BLangWorkerSyncSendAction) (semtypes.SemType, expressionEffect, bool) {
	chain, ok := resolveWorkerMessageSend(t, chain, e.Expr, e.Message, e.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Union(workerFailureType(t, e.Peer.Symbol), semtypes.Nil)
	e.SetDeterminedType(resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

// resolveWorkerMessageSend resolves the value sent and publishes its type for
// the receive. The send isn't typed with the receive's expected type.
func resolveWorkerMessageSend(
	t typeResolver,
	chain *binding,
	expr ast.BLangExpression,
	message model.WorkerMessageRef,
	pos diagnostics.Location,
) (*binding, bool) {
	result, ok := resolveActionOrExpression(t, chain, expr, semtypes.SemType{})
	if !ok {
		return chain, false
	}
	if err := resolverMessageTypes(t).Publish(message, result.ty); err != nil {
		t.internalError(err.Error(), pos)
		return chain, false
	}
	return sequentialChain(t, result.effect), true
}

func resolveWorkerReceiveAction(t typeResolver, chain *binding, e *ast.BLangWorkerReceiveAction) (semtypes.SemType, expressionEffect, bool) {
	messageTy, ok := workerMessageType(t, e.Message, e.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Union(messageTy, workerFailureType(t, e.Peer.Symbol))
	e.SetDeterminedType(resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

// resolveWorkerMultipleReceiveAction types a multiple receive as the closed
// record of the received values, built like a mapping constructor with no
// expected type, unioned with the failure type of every peer.
func resolveWorkerMultipleReceiveAction(t typeResolver, chain *binding, e *ast.BLangWorkerMultipleReceiveAction) (semtypes.SemType, expressionEffect, bool) {
	fields := make([]semtypes.Field, len(e.Fields))
	failureTy := semtypes.Never
	for i, field := range e.Fields {
		messageTy, ok := workerMessageType(t, field.Message, e.GetPosition())
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		if !semtypes.SingleShape(messageTy).IsEmpty() {
			messageTy = semtypes.WidenToBasicTypes(messageTy)
		}
		fields[i] = semtypes.FieldFrom(field.FieldName, messageTy, false, false)
		failureTy = semtypes.Union(failureTy, workerFailureType(t, field.Peer.Symbol))
	}
	md := semtypes.NewMappingDefinition()
	recordTy := md.Define(t.typeEnv(), fields, semtypes.Never)
	resultTy := semtypes.Union(recordTy, failureTy)
	e.SetDeterminedType(resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

// resolveWorkerFlushAction types a flush with the failure of every peer whose
// messages it covers.
func resolveWorkerFlushAction(t typeResolver, chain *binding, e *ast.BLangWorkerFlushAction) (semtypes.SemType, expressionEffect, bool) {
	resultTy := semtypes.Nil
	for _, coverage := range e.Covered {
		resultTy = semtypes.Union(resultTy, workerFailureType(t, coverage.Peer))
	}
	e.SetDeterminedType(resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

// workerMessageType waits for the send of message to publish its type. A
// receive is only paired with a send pairing proved can run before it, so the
// wait ends.
func workerMessageType(t typeResolver, message model.WorkerMessageRef, pos diagnostics.Location) (semtypes.SemType, bool) {
	store := resolverMessageTypes(t)
	ty, ok := store.Type(message)
	if !ok {
		// The send failed to resolve, which it reported unless it was in a
		// candidate trial.
		if !t.isEphemeral() && !t.compilerContext().HasErrors() {
			t.internalError("worker message type was never published", pos)
		}
		return semtypes.SemType{}, false
	}
	return ty, true
}

func workerFailureType(t typeResolver, peer model.SymbolRef) semtypes.SemType {
	return common.WorkerFailureType(t.typeContext(), t.symbolType(peer))
}
