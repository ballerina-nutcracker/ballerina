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
	setNodeType(t, e, semtypes.Nil)
	return semtypes.Nil, defaultExpressionEffect(chain), true
}

func resolveWorkerSyncSendAction(t typeResolver, chain *binding, e *ast.BLangWorkerSyncSendAction) (semtypes.SemType, expressionEffect, bool) {
	chain, ok := resolveWorkerMessageSend(t, chain, e.Expr, e.Message, e.GetPosition())
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Union(workerFailureType(t, e.Peer.Symbol), semtypes.Nil)
	setNodeType(t, e, resultTy)
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
	if err := t.compilerContext().SetWorkerMessageType(message, result.ty); err != nil {
		t.internalError(err.Error(), pos)
		return chain, false
	}
	return sequentialChain(t, result.effect), true
}

func resolveWorkerReceiveAction(t typeResolver, chain *binding, e *ast.BLangWorkerReceiveAction) (semtypes.SemType, expressionEffect, bool) {
	messageTy, ok := workerMessageType(t, e.Message)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Union(messageTy, workerFailureType(t, e.Peer.Symbol))
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

// resolveWorkerMultipleReceiveAction types a multiple receive as the record
// type R of the received values unioned with the failure type of every peer.
func resolveWorkerMultipleReceiveAction(t typeResolver, chain *binding, e *ast.BLangWorkerMultipleReceiveAction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool) {
	fields := make([]semtypes.MappingFieldInfo, len(e.Fields))
	failureTy := semtypes.Never
	for i, field := range e.Fields {
		messageTy, ok := workerMessageType(t, field.Message)
		if !ok {
			return semtypes.SemType{}, expressionEffect{}, false
		}
		fields[i] = semtypes.MappingFieldInfo{Name: field.FieldName, Type: messageTy}
		failureTy = semtypes.Union(failureTy, workerFailureType(t, field.Peer.Symbol))
	}
	recordTy, ok := multipleReceiveRecordType(t, e, fields, expectedType)
	if !ok {
		return semtypes.SemType{}, expressionEffect{}, false
	}
	resultTy := semtypes.Union(recordTy, failureTy)
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

// multipleReceiveRecordType chooses R the way a mapping constructor chooses
// its inherent type: the only alternative of the mapping part of expectedType
// that fields fit, where every field of R is received. Without an expected
// type, or when no alternative fits, R is the closed record of fields, which
// semantic analysis then checks against the expected type.
func multipleReceiveRecordType(t typeResolver, e *ast.BLangWorkerMultipleReceiveAction, fields []semtypes.MappingFieldInfo, expectedType semtypes.SemType) (semtypes.SemType, bool) {
	if semtypes.IsZero(expectedType) {
		return closedReceiveRecordType(t, fields), true
	}
	candidates, ok := mappingConstructorCandidates(t, e.GetPosition(), expectedType)
	if !ok {
		return semtypes.SemType{}, false
	}
	var fitting []mappingCandidate
	for _, candidate := range candidates {
		if semtypes.MappingAlternativeAllowsFields(t.typeContext(), candidate.alt, fields, noDefaultableFields) {
			fitting = append(fitting, candidate)
		}
	}
	switch len(fitting) {
	case 0:
		return closedReceiveRecordType(t, fields), true
	case 1:
		return fitting[0].alt.Type(), true
	default:
		t.semanticError("ambiguous inherent type for multiple receive", e.GetPosition())
		return semtypes.SemType{}, false
	}
}

func noDefaultableFields(*semtypes.MappingAtomicType) []string {
	return nil
}

func closedReceiveRecordType(t typeResolver, fields []semtypes.MappingFieldInfo) semtypes.SemType {
	recordFields := make([]semtypes.Field, len(fields))
	for i, field := range fields {
		fieldTy := field.Type
		if !semtypes.SingleShape(fieldTy).IsEmpty() {
			fieldTy = semtypes.WidenToBasicTypes(fieldTy)
		}
		recordFields[i] = semtypes.FieldFrom(field.Name, fieldTy, false, false)
	}
	return createClosedRecordType(t.typeEnv(), recordFields, semtypes.Never)
}

// resolveWorkerFlushAction types a flush with the failure of every peer whose
// messages it covers.
func resolveWorkerFlushAction(t typeResolver, chain *binding, e *ast.BLangWorkerFlushAction) (semtypes.SemType, expressionEffect, bool) {
	resultTy := semtypes.Nil
	for _, coverage := range e.Covered {
		resultTy = semtypes.Union(resultTy, workerFailureType(t, coverage.Peer))
	}
	setNodeType(t, e, resultTy)
	return resultTy, defaultExpressionEffect(chain), true
}

// workerMessageType waits for the send's type or a resolution failure.
func workerMessageType(t typeResolver, message model.WorkerMessageRef) (semtypes.SemType, bool) {
	result := <-t.compilerContext().WorkerMessageType(message)
	return result.Type, !result.Failed
}

func workerFailureType(t typeResolver, peer model.SymbolRef) semtypes.SemType {
	return common.WorkerFailureType(t.typeContext(), t.symbolType(peer))
}
