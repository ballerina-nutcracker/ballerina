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

package compile

import (
	"github.com/ballerina-nutcracker/ballerina/context"
	libcommon "github.com/ballerina-nutcracker/ballerina/lib/common"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

var PackageID = model.INTERNAL_PKG

const PackageName = "lang.__internal"

var templateInsertionAllowedTypes = semtypes.Diff(semtypes.SimpleOrString, semtypes.Nil)

func GetInternalSymbols(ctx *context.CompilerContext) model.ExportedSymbolSpace {
	space := ctx.NewSymbolSpace(*PackageID)
	addInternalFunction(ctx, space, "querySort", model.FunctionSignature{
		ParamTypes: []semtypes.SemType{semtypes.List, semtypes.List, semtypes.List, semtypes.List},
		ReturnType: semtypes.Nil,
	})
	addInternalFunction(ctx, space, "queryGroup", model.FunctionSignature{
		ParamTypes: []semtypes.SemType{semtypes.List, semtypes.List, semtypes.List},
		ReturnType: semtypes.List,
	})
	addInternalFunction(ctx, space, "queryCollect", model.FunctionSignature{
		ParamTypes: []semtypes.SemType{semtypes.List, semtypes.Int, semtypes.List},
		ReturnType: semtypes.List,
	})
	addInternalFunction(ctx, space, "escapeXMLContent", model.FunctionSignature{
		ParamTypes: []semtypes.SemType{templateInsertionAllowedTypes},
		ReturnType: semtypes.String,
	})
	addInternalFunction(ctx, space, "escapeXMLAttribute", model.FunctionSignature{
		ParamTypes: []semtypes.SemType{templateInsertionAllowedTypes},
		ReturnType: semtypes.String,
	})
	return model.NewExportedSymbolSpaces([]*model.SymbolSpace{space}, nil)
}

func addInternalFunction(ctx *context.CompilerContext, space *model.SymbolSpace, name string, sig model.FunctionSignature) {
	symbol := model.NewFunctionSymbol(name, sig, true, diagnostics.NewBuiltinLocation())
	space.AddSymbol(name, symbol)
	ref, _ := space.GetSymbol(name)
	ctx.SetSymbolType(ref, libcommon.FunctionSignatureToSemType(ctx.GetTypeEnv(), &sig))
}
