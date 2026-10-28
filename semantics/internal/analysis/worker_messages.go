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

package analysis

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	semcommon "github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// validateWorkerAsyncSend checks the value sent, and that a failure of the
// receiver can be observed: an async send to a worker that can fail must be
// followed by a sync send or flush to it.
func validateWorkerAsyncSend(cx scope.Context, e *ast.BLangWorkerAsyncSendAction, expected semtypes.SemType) (scope.Facts, bool) {
	if !validateWorkerMessageValue(cx, e.Expr) {
		return scope.NoFacts, false
	}
	failureTy := semcommon.WorkerFailureType(cx.TyCtx(), cx.CompilerCtx().SymbolType(e.Peer.Symbol))
	if !e.Covered && !semtypes.IsEmpty(cx.TyCtx(), failureTy) {
		cx.SemanticErr("async send to a worker that can fail must be followed by a sync send or flush to it", e.GetPosition())
		return scope.NoFacts, false
	}
	return validateWorkerMessageResult(cx, e, expected)
}

func validateWorkerSyncSend(cx scope.Context, e *ast.BLangWorkerSyncSendAction, expected semtypes.SemType) (scope.Facts, bool) {
	if !validateWorkerMessageValue(cx, e.Expr) {
		return scope.NoFacts, false
	}
	return validateWorkerMessageResult(cx, e, expected)
}

func validateWorkerMessageValue(cx scope.Context, expr ast.BLangExpression) bool {
	if _, ok := validateActionOrExpr(cx, expr, semtypes.SemType{}); !ok {
		return false
	}
	if !semtypes.IsSubtype(cx.TyCtx(), expr.Type(), semtypes.CreateCloneable(cx.TyCtx())) {
		cx.SemanticErr("worker message value must be a subtype of value:Cloneable", expr.GetPosition())
		return false
	}
	return true
}

func validateWorkerMessageResult(cx scope.Context, e ast.BLangActionOrExpression, expected semtypes.SemType) (scope.Facts, bool) {
	f := scope.FactsOfType(cx, e)
	return f, validateResolvedType(cx, e, expected)
}
