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
}

func init() {
	runtime.RegisterModuleInitializer(initArrayModule)
}
