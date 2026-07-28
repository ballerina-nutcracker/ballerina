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
	nextMethod = "$stringIterator.next"
)

func stringLength(args []values.BalValue) (values.BalValue, error) {
	return int64(utf8.RuneCountInString(args[0].(string))), nil
}

func stringToBytes(byteArrTy semtypes.SemType, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return values.ByteSliceToList(byteArrTy, ctx.TypeCtx, []byte(args[0].(string))), nil
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
	runtime.RegisterExternFunction(rt, orgName, moduleName, "iterator", stringIterator)
	runtime.RegisterExternFunction(rt, orgName, moduleName, nextMethod, stringIteratorNext)
}

func stringIterator(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	str, _ := args[0].(string)
	chars := make([]values.BalValue, 0, utf8.RuneCountInString(str))
	for _, char := range str {
		chars = append(chars, string(char))
	}
	return values.NewObject(semtypes.OBJECT, map[string]values.BalValue{
		"chars": chars,
		"idx":   int64(0),
	}, map[string]string{
		"next": orgName + "/" + moduleName + ":" + nextMethod,
	}, nil), nil
}

func stringIteratorNext(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	iterator := args[0].(*values.Object)
	charsValue, _ := iterator.Get("chars")
	idxValue, _ := iterator.Get("idx")
	chars := charsValue.([]values.BalValue)
	idx := idxValue.(int64)
	if idx >= int64(len(chars)) {
		return nil, nil
	}
	iterator.Put("idx", idx+1)
	recordTy := stringIteratorNextRecordType(ctx.Env.TypeEnv)
	return values.NewMap(recordTy, semtypes.ToMappingAtomicType(ctx.TypeCtx, recordTy), false, []values.MapEntry{{
		Key:   "value",
		Value: chars[idx],
	}}), nil
}

func stringIteratorNextRecordType(env semtypes.Env) semtypes.SemType {
	def := semtypes.NewMappingDefinition()
	return def.DefineMappingTypeWrapped(env,
		[]semtypes.Field{semtypes.FieldFrom("value", semtypes.CHAR, false, false)},
		semtypes.NEVER)
}

func init() {
	runtime.RegisterModuleInitializer(initStringModule)
}
