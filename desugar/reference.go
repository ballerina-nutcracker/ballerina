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
)

// referenceKind is what a name in an expression position actually denotes.
// Every kind other than an ordinary variable is lowered to something else.
type referenceKind int

const (
	ordinaryVariableRef referenceKind = iota
	constantValueRef
	namedWorkerRef
)

// referenceKindOf classifies a reference and returns the resolved symbol, so a
// caller that needs the symbol does not look it up a second time.
func referenceKindOf(cx *functionContext, ref *ast.BLangVarRef) (referenceKind, model.Symbol) {
	symbol := cx.getSymbol(ref.Symbol())
	if _, isConstant := symbol.(*model.ConstantValueSymbol); isConstant {
		return constantValueRef, symbol
	}
	if symbol.Kind() == model.SymbolKindWorker {
		return namedWorkerRef, symbol
	}
	return ordinaryVariableRef, symbol
}

// rewriteReference lowers a reference according to what it denotes: a constant
// folds to its value and a named worker becomes a load of the worker's hidden
// future slot. An ordinary variable reference needs no rewriting, which it
// reports by returning nil.
func rewriteReference(cx *functionContext, ref *ast.BLangVarRef) ast.BLangExpression {
	switch kind, symbol := referenceKindOf(cx, ref); kind {
	case constantValueRef:
		return materializeConstantRef(cx, symbol.(*model.ConstantValueSymbol), ref)
	case namedWorkerRef:
		return workerFutureSlotRef(cx, ref)
	default:
		return nil
	}
}
