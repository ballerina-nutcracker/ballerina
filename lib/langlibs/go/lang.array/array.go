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
	"fmt"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.array"
)

func arrayLength(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	return int64(list.Len()), nil
}

func arrayToBase64(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data := list.ToByteSlice()
	return base64.StdEncoding.EncodeToString(data), nil
}

func arrayToBase16(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data := list.ToByteSlice()
	return hex.EncodeToString(data), nil
}

func arrayFromBase64(byteArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s := args[0].(string)
		data, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return values.NewErrorWithMessage("failed to decode base64 string"), nil
		}
		return values.ByteSliceToList(byteArrTy, ctx.TypeCtx, data), nil
	}
}

func arrayFromBase16(byteArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s := args[0].(string)
		data, err := hex.DecodeString(s)
		if err != nil {
			return values.NewErrorWithMessage("failed to decode base16 string"), nil
		}
		return values.ByteSliceToList(byteArrTy, ctx.TypeCtx, data), nil
	}
}

func arrayPush(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	list.Append(ctx.TypeCtx, args[1:]...)
	return nil, nil
}

// arrayIndexOf bails out before the int64->int conversion below: on a
// 32-bit int platform (e.g. wasm) a large startIndex would truncate,
// possibly to a negative value, and list.Get is unchecked.
func arrayIndexOf(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	val := args[1]
	startIndex := int64(0)
	if len(args) > 2 && args[2] != nil {
		if si, ok := args[2].(int64); ok {
			startIndex = si
		}
	}
	if startIndex < 0 {
		panic(values.NewErrorWithMessage(fmt.Sprintf("invalid array index: %d", startIndex)))
	}
	if startIndex >= int64(list.Len()) {
		return nil, nil
	}
	for i := int(startIndex); i < list.Len(); i++ {
		if values.DeepEquals(list.Get(i), val) {
			return int64(i), nil
		}
	}
	return nil, nil
}

func arrayRemove(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	index := args[1].(int64)
	if index < 0 || index >= int64(list.Len()) {
		panic(values.NewErrorWithMessage(fmt.Sprintf("invalid array index: %d", index)))
	}
	return list.RemoveAt(int(index)), nil
}

func arrayRemoveAll(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	list.Clear()
	return nil, nil
}

func initArrayModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", arrayLength)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase64", arrayToBase64)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase16", arrayToBase16)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBase64", arrayFromBase64(byteArrTy))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBase16", arrayFromBase16(byteArrTy))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "push", arrayPush)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "indexOf", arrayIndexOf)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "remove", arrayRemove)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "removeAll", arrayRemoveAll)
}

func init() {
	runtime.RegisterModuleInitializer(initArrayModule)
}
