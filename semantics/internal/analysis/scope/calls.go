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

package scope

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// nonIsolatedCall reports whether inv calls a non-isolated function or passes
// a non-isolated function to an isolated parameter; argIndex is that
// argument's index, or -1 when the callee itself is not isolated.
func nonIsolatedCall(cx Context, inv *ast.BLangInvocation) (argIndex int, nonIsolated bool) {
	ctx := cx.CompilerCtx()
	tyCtx := cx.TyCtx()
	isolatedFn := semtypes.CreateIsolatedFn(tyCtx)
	fnRef := inv.ResolvedSymbol()
	if !semtypes.IsSubtype(tyCtx, ctx.SymbolType(fnRef), isolatedFn) {
		return -1, true
	}
	switch ctx.GetSymbol(fnRef).(type) {
	case model.MonomorphicFunctionSymbol:
		sig, ok := ctx.GetFunctionSignature(fnRef)
		if !ok {
			cx.InternalErr("function signature not found", inv.GetPosition())
			return 0, false
		}
		for i, arg := range inv.CallArgs() {
			if sig.IsIsolatedParam(i) && !semtypes.IsSubtype(tyCtx, arg.Type(), isolatedFn) {
				return i, true
			}
		}
		return 0, false
	case model.FunctionSymbol, model.ValueSymbol:
		// Only monomorphised lang-lib functions have isolated parameters, so
		// only the callee's type (checked above) matters.
		return 0, false
	default:
		cx.InternalErr("unexpected symbol kind in invocation", inv.GetPosition())
		return 0, false
	}
}
