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

package stringruntime

import (
	"unicode/utf8"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.string"
)

func stringLength(args []values.BalValue) (values.BalValue, error) {
	return int64(utf8.RuneCountInString(args[0].(string))), nil
}

func stringToBytes(byteArrTy semtypes.SemType, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return byteSliceToList(byteArrTy, ctx, []byte(args[0].(string))), nil
}

func stringFromBytes(args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data, _ := listToByteSlice(list)
	if !utf8.Valid(data) {
		return values.NewErrorWithMessage("invalid UTF-8 byte array"), nil
	}
	return string(data), nil
}

func listToByteSlice(list *values.List) ([]byte, bool) {
	b := make([]byte, list.Len())
	for i := 0; i < list.Len(); i++ {
		n, ok := list.Get(i).(int64)
		if !ok || n < 0 || n > 255 {
			return nil, false
		}
		b[i] = byte(n)
	}
	return b, true
}

func byteSliceToList(byteArrTy semtypes.SemType, ctx *extern.Context, data []byte) *values.List {
	items := make([]values.BalValue, len(data))
	for i, b := range data {
		items[i] = int64(b)
	}
	return values.NewList(byteArrTy, semtypes.ToListAtomicType(ctx.TypeCtx, byteArrTy), false, nil, 0, items)
}

func initStringModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringLength(args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBytes", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringToBytes(byteArrTy, ctx, args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBytes", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringFromBytes(args)
	})
}

func init() {
	runtime.RegisterModuleInitializer(initStringModule)
}
