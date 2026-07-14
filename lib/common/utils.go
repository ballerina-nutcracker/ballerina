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

package common

import (
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
)

func FunctionSignatureToSemType(env semtypes.Env, fs *model.FunctionSignature) semtypes.SemType {
	var restTy semtypes.SemType
	if !semtypes.IsZero(fs.RestParamType) {
		restTy = fs.RestParamType
	} else {
		restTy = semtypes.NEVER
	}

	// Build the parameter list type
	paramListDefn := semtypes.NewListDefinition()
	paramListTy := paramListDefn.DefineListTypeWrapped(env, fs.ParamTypes, len(fs.ParamTypes), restTy, semtypes.CellMutability_CELL_MUT_NONE)

	// Build the function type
	functionDefn := semtypes.NewFunctionDefinition()
	return functionDefn.Define(env, paramListTy, fs.ReturnType,
		semtypes.FunctionQualifiersFrom(env, fs.IsIsolated(), fs.IsTransactional()))
}
