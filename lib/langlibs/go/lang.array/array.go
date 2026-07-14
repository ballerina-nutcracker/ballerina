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

package array

import (
	"encoding/base64"
	"encoding/hex"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.array"
)

func arrayLength(args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	return int64(list.Len()), nil
}

func arrayToBase64(args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data := list.ToByteSlice()
	return base64.StdEncoding.EncodeToString(data), nil
}

func arrayToBase16(args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data := list.ToByteSlice()
	return hex.EncodeToString(data), nil
}

func arrayFromBase64(byteArrTy semtypes.SemType, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return values.NewErrorWithMessage("failed to decode base64 string"), nil
	}
	return values.ByteSliceToList(byteArrTy, ctx.TypeCtx, data), nil
}

func arrayFromBase16(byteArrTy semtypes.SemType, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	data, err := hex.DecodeString(s)
	if err != nil {
		return values.NewErrorWithMessage("failed to decode base16 string"), nil
	}
	return values.ByteSliceToList(byteArrTy, ctx.TypeCtx, data), nil
}

func arrayPush(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	list.Append(ctx.TypeCtx, args[1:]...)
	return nil, nil
}

func arrayMap(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source := args[0].(*values.List)
	callback := args[1].(*values.Function)
	memberTy := semtypes.ListProj(ctx.TypeCtx, source.Type, semtypes.INT)
	argListDef := semtypes.NewListDefinition()
	argListTy := argListDef.DefineListTypeWrapped(ctx.Env.TypeEnv, []semtypes.SemType{memberTy}, 1, semtypes.NEVER, semtypes.CellMutability_CELL_MUT_NONE)
	var resultMemberTy semtypes.SemType
	if semtypes.IsNever(memberTy) {
		resultMemberTy = semtypes.FunctionReturnType(ctx.TypeCtx, callback.Type, semtypes.FunctionParamListType(ctx.TypeCtx, callback.Type))
	} else {
		resultMemberTy = semtypes.FunctionReturnType(ctx.TypeCtx, callback.Type, argListTy)
	}

	items := make([]values.BalValue, source.Len())
	callbackArgs := make([]values.BalValue, 1)
	for i := range source.Len() {
		callbackArgs[0] = source.Get(i)
		result, err := ctx.InvokeFunctionValue(callback, callbackArgs)
		if err != nil {
			return nil, err
		}
		items[i] = result
	}

	resultDef := semtypes.NewListDefinition()
	resultTy := resultDef.DefineListTypeWrappedWithEnvSemType(ctx.Env.TypeEnv, resultMemberTy)
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, resultTy)
	filler, _ := values.FillerFactoryFor(ctx.TypeCtx, resultMemberTy)
	return values.NewList(resultTy, atomic, false, filler, 0, items), nil
}

func initArrayModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return arrayLength(args)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase64", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return arrayToBase64(args)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase16", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return arrayToBase16(args)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBase64", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return arrayFromBase64(byteArrTy, ctx, args)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBase16", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return arrayFromBase16(byteArrTy, ctx, args)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "push", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return arrayPush(ctx, args)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "map", arrayMap)
}

func init() {
	runtime.RegisterModuleInitializer(initArrayModule)
}
