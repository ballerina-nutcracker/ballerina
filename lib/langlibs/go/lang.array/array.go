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

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
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
		return values.ByteSliceToList(byteArrTy, ctx.TypeEnv(), data), nil
	}
}

func arrayFromBase16(byteArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s := args[0].(string)
		data, err := hex.DecodeString(s)
		if err != nil {
			return values.NewErrorWithMessage("failed to decode base16 string"), nil
		}
		return values.ByteSliceToList(byteArrTy, ctx.TypeEnv(), data), nil
	}
}

func arrayPush(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	list.Append(ctx.TypeCtx(), args[1:]...)
	return nil, nil
}

func arrayMap(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	source := args[0].(*values.List)
	callback := args[1].(*values.Function)
	memberTy := semtypes.ListProj(ctx.TypeCtx(), source.Type, semtypes.Int)
	argListDef := semtypes.NewListDefinition()
	argListTy := argListDef.Define(ctx.TypeEnv(), []semtypes.SemType{memberTy},
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	var resultMemberTy semtypes.SemType
	if semtypes.IsNever(memberTy) {
		resultMemberTy = semtypes.FunctionReturnType(ctx.TypeCtx(), callback.Type, semtypes.FunctionParamListType(ctx.TypeCtx(), callback.Type))
	} else {
		resultMemberTy = semtypes.FunctionReturnType(ctx.TypeCtx(), callback.Type, argListTy)
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
	resultTy := resultDef.Define(ctx.TypeEnv(), nil, semtypes.ListRest(resultMemberTy))
	atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), resultTy)
	filler, _ := values.FillerFactoryFor(ctx.TypeCtx(), resultMemberTy)
	return values.NewList(resultTy, atomic, false, filler, 0, items), nil
}

// arrayIndexOf returns nil (no match) once startIndex reaches the list's
// length, before the loop below ever indexes into it.
func arrayIndexOf(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	val := args[1]
	startIndex := int64(0)
	if len(args) > 2 && args[2] != nil {
		si, ok := args[2].(int64)
		if !ok {
			panic(fmt.Sprintf("internal error: unexpected startIndex type %T", args[2]))
		}
		startIndex = si
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

func arrayRemove(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	index := args[1].(int64)
	if index < 0 || index >= int64(list.Len()) {
		panic(values.NewErrorWithMessage(fmt.Sprintf("invalid array index: %d", index)))
	}
	return list.RemoveAt(ctx.TypeCtx(), int(index)), nil
}

func arrayRemoveAll(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	list.Clear()
	return nil, nil
}

func initArrayModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.Define(env, nil, semtypes.ListRest(semtypes.Byte))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", arrayLength)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase64", arrayToBase64)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBase16", arrayToBase16)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBase64", arrayFromBase64(byteArrTy))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBase16", arrayFromBase16(byteArrTy))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "push", arrayPush)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "map", arrayMap)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "indexOf", arrayIndexOf)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "remove", arrayRemove)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "removeAll", arrayRemoveAll)
}

func init() {
	runtime.RegisterModuleInitializer(initArrayModule)
}
