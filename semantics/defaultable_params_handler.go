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
	"fmt"

	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
)

type symbolLookup interface {
	getSymbol(ref model.SymbolRef) model.Symbol
	internalError(message string, loc diagnostics.Location)
	unimplemented(message string, loc diagnostics.Location)
}

func padArgTypesForDefaults(lookup symbolLookup, symbolRef model.SymbolRef, argTys []semtypes.SemType, loc diagnostics.Location) []semtypes.SemType {
	sym := lookup.getSymbol(symbolRef)
	if _, ok := sym.(*model.OpaqueFunctionSymbol); ok {
		// TODO: there are type param functions with default params need to think how to implement this
		lookup.unimplemented("generic functions with default params not implemented", loc)
		return argTys
	}
	switch fnSym := sym.(type) {
	case model.FunctionSymbol:
		return padFunctionDefaults(fnSym, argTys)
	case model.ValueSymbol:
		// When we support lambdas we need to have a way to get a function symbol from the declaration (this means it have to be atomic) and then use the
		// same logic
		return argTys
	default:
		lookup.internalError(fmt.Sprintf("unexpected symbol type %T in padArgTypesForDefaults", sym), loc)
		return argTys
	}
}

func padFunctionDefaults(fnSym model.FunctionSymbol, argTys []semtypes.SemType) []semtypes.SemType {
	defaultableParams := fnSym.DefaultableParams()
	totalParams := len(fnSym.Signature().ParamTypes)
	if len(argTys) >= totalParams {
		return argTys
	}
	for i := len(argTys); i < totalParams; i++ {
		if _, ok := defaultableParams.Get(i); !ok {
			// When caller do function application with this they'll get an error and at that point they can decide
			// how to handle the error
			return argTys
		}
	}
	paramTypes := fnSym.Signature().ParamTypes
	padded := make([]semtypes.SemType, totalParams)
	copy(padded, argTys)
	for i := len(argTys); i < totalParams; i++ {
		padded[i] = paramTypes[i]
	}
	return padded
}
