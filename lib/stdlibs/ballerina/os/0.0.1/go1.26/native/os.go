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

package native

import (
	"ballerina-lang-go/bir"
	"ballerina-lang-go/model"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "os"
)

func newProcessObject(handle pal.ProcessHandle) *values.Object {
	return values.NewObject(
		semtypes.OBJECT,
		map[string]values.BalValue{"$handle": handle},
		map[string]string{
			"waitForExit": "ballerina/os:Process.waitForExit",
			"output":      "ballerina/os:Process.output",
			"exit":        "ballerina/os:Process.exit",
		},
		nil,
	)
}

func getHandle(self *values.Object) pal.ProcessHandle {
	v, _ := self.Get("$handle")
	return v.(pal.ProcessHandle)
}

func osError(msg string) values.BalValue {
	return values.NewErrorWithMessage(msg)
}

func initOSModule(rt *runtime.Runtime) {
	runtime.RegisterExternClassDef(rt, &bir.BIRClassDef{
		Name:      model.Name("Process"),
		LookupKey: "ballerina/os:Process",
		Fields:    []bir.ObjectField{},
		VTable: map[string]*bir.BIRFunction{
			"waitForExit": {FunctionLookupKey: "ballerina/os:Process.waitForExit"},
			"output":      {FunctionLookupKey: "ballerina/os:Process.output"},
			"exit":        {FunctionLookupKey: "ballerina/os:Process.exit"},
		},
	})

	env := rt.GetTypeEnv()
	bld := semtypes.NewListDefinition()
	byteArrTy := bld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE)
	smd := semtypes.NewMappingDefinition()
	strMapTy := smd.DefineMappingTypeWrapped(env, nil, semtypes.STRING)

	// Atomic types are a structural property of the (fixed) SemTypes above and
	// do not vary per strand, so compute them once instead of on every call.
	initCtx := semtypes.ContextFrom(env)
	strMapAtomic := semtypes.ToMappingAtomicType(initCtx, strMapTy)
	byteArrAtomic := semtypes.ToListAtomicType(initCtx, byteArrTy)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getEnv",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			name, _ := args[0].(string)
			return rt.Platform().OS.GetEnv(name), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getUsername",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			return rt.Platform().OS.GetUsername(), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getUserHome",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			return rt.Platform().OS.GetUserHome(), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "setEnvExtern",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			key, _ := args[0].(string)
			val, _ := args[1].(string)
			if err := rt.Platform().OS.SetEnv(key, val); err != nil {
				return osError(err.Error()), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "unsetEnvExtern",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			key, _ := args[0].(string)
			if err := rt.Platform().OS.UnsetEnv(key); err != nil {
				return osError(err.Error()), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "listEnv",
		func(ctx *extern.Context, _ []values.BalValue) (values.BalValue, error) {

			envMap := rt.Platform().OS.ListEnv()
			m := values.NewMap(strMapTy, strMapAtomic, false, nil)
			for k, v := range envMap {
				m.Put(ctx.TypeCtx, k, v)
			}
			return m, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "exec",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			cmdMap, _ := args[0].(*values.Map)
			cmdVal, _ := cmdMap.Get("value")
			command, _ := cmdVal.(string)

			var cmdArgs []string
			if argList, ok := cmdMap.Get("arguments"); ok {
				if list, ok := argList.(*values.List); ok {
					for i := 0; i < list.Len(); i++ {
						if s, ok := list.Get(i).(string); ok {
							cmdArgs = append(cmdArgs, s)
						}
					}
				}
			}

			envOverride := make(map[string]string)
			if len(args) > 1 {
				if envMap, ok := args[1].(*values.Map); ok {
					for _, k := range envMap.Keys() {
						if v, ok := envMap.Get(k); ok {
							if sv, ok := v.(string); ok {
								envOverride[k] = sv
							}
						}
					}
				}
			}

			handle, err := rt.Platform().OS.Exec(command, cmdArgs, envOverride)
			if err != nil {
				return osError("Failed to retrieve the process object: " + err.Error()), nil
			}
			return newProcessObject(handle), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Process.waitForExit",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			code, err := getHandle(self).WaitForExit()
			if err != nil {
				return osError("Failed to wait for process to exit: " + err.Error()), nil
			}
			return int64(code), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "$Process.output$default$0",
		func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			return int64(1), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Process.output",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {

			self, _ := args[0].(*values.Object)
			stream, _ := args[1].(int64)
			handle := getHandle(self)
			var (
				data []byte
				err  error
			)
			if stream == 1 {
				data, err = handle.ReadStdout()
			} else {
				data, err = handle.ReadStderr()
			}
			if err != nil {
				return osError("Failed to read the output of the process: " + err.Error()), nil
			}
			items := make([]values.BalValue, len(data))
			for i, b := range data {
				items[i] = int64(b)
			}
			return values.NewList(byteArrTy, byteArrAtomic, false, nil, 0, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Process.exit",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			getHandle(self).Kill()
			return nil, nil
		})
}

func init() {
	runtime.RegisterModuleInitializer(initOSModule)
}
