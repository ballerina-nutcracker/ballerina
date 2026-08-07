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

	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/semtypes"
	"ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.string"
)

type stringIteratorHandle struct {
	value  string
	offset int
}

func stringLength(args []values.BalValue) (values.BalValue, error) {
	return int64(utf8.RuneCountInString(args[0].(string))), nil
}

func stringToBytes(byteArrTy semtypes.SemType, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return values.ByteSliceToList(byteArrTy, ctx.TypeEnv(), []byte(args[0].(string))), nil
}

func stringFromBytes(args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data := list.ToByteSlice()
	if !utf8.Valid(data) {
		return values.NewErrorWithMessage("invalid UTF-8 byte array"), nil
	}
	return string(data), nil
}

func initStringModule(rt *runtime.Runtime) {
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.DefineListTypeWrappedWithEnvSemType(rt.GetTypeEnv(), semtypes.BYTE)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringLength(args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBytes", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringToBytes(byteArrTy, ctx, args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBytes", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringFromBytes(args)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "createIteratorHandle", createStringIteratorHandle)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "iteratorHasNext", stringIteratorHasNext)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "iteratorNext", stringIteratorNext)
}

func createStringIteratorHandle(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return &stringIteratorHandle{value: args[0].(string)}, nil
}

func stringIteratorHasNext(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	iterator := args[0].(*stringIteratorHandle)
	return iterator.offset < len(iterator.value), nil
}

func stringIteratorNext(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	iterator := args[0].(*stringIteratorHandle)
	char, size := utf8.DecodeRuneInString(iterator.value[iterator.offset:])
	iterator.offset += size
	return string(char), nil
}

func init() {
	runtime.RegisterModuleInitializer(initStringModule)
}
