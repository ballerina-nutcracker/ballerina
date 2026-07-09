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

	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
)

type symbolLookup interface {
	getSymbol(ref model.SymbolRef) model.Symbol
	compilerContext() *context.CompilerContext
	typeContext() semtypes.Context
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
	sig, ok := lookup.compilerContext().GetFunctionSignature(symbolRef)
	if !ok {
		return argTys
	}
	switch fnSym := sym.(type) {
	case model.FunctionSymbol:
		return padFunctionDefaults(fnSym.TypedSignature().ParamTypes, sig, argTys)
	case model.ValueSymbol:
		fnTy := lookup.compilerContext().SymbolType(symbolRef)
		return padFunctionDefaults(functionValueParamTypes(lookup, fnTy, sig.FixedParamCount()), sig, argTys)
	default:
		lookup.internalError(fmt.Sprintf("unexpected symbol type %T in padArgTypesForDefaults", sym), loc)
		return argTys
	}
}

func padFunctionDefaults(paramTypes []semtypes.SemType, sig model.UntypedFunctionSignature, argTys []semtypes.SemType) []semtypes.SemType {
	totalParams := len(paramTypes)
	if len(argTys) >= totalParams {
		return argTys
	}
	for i := len(argTys); i < totalParams; i++ {
		if _, ok := sig.DefaultableParam(i); !ok {
			// When caller do function application with this they'll get an error and at that point they can decide
			// how to handle the error
			return argTys
		}
	}
	padded := make([]semtypes.SemType, totalParams)
	copy(padded, argTys)
	for i := len(argTys); i < totalParams; i++ {
		padded[i] = paramTypes[i]
	}
	return padded
}

func functionValueParamTypes(lookup symbolLookup, fnTy semtypes.SemType, count int) []semtypes.SemType {
	paramListTy := semtypes.FunctionParamListType(lookup.typeContext(), fnTy)
	paramTypes := make([]semtypes.SemType, count)
	for i := range paramTypes {
		paramTypes[i] = semtypes.ListMemberTypeInnerVal(lookup.typeContext(), paramListTy, semtypes.IntConst(int64(i)))
	}
	return paramTypes
}
