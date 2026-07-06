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

package maprt

import (
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.map"
)

func mapLength(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	m := args[0].(*values.Map)
	return int64(m.Len()), nil
}

func mapKeys(env semtypes.Env) extern.NativeFunc {
	ld := semtypes.NewListDefinition()
	stringArrayTy := ld.DefineListTypeWrappedWithEnvSemType(env, semtypes.STRING)
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		m := args[0].(*values.Map)
		keys := m.Keys()
		items := make([]values.BalValue, len(keys))
		for i, k := range keys {
			items[i] = k
		}
		atomic := semtypes.ToListAtomicType(ctx.TypeCtx, stringArrayTy)
		list := values.NewList(stringArrayTy, atomic, false, nil, 0, items)
		return list, nil
	}
}

func mapRemove(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	m := args[0].(*values.Map)
	key := args[1].(string)
	val, _ := m.Get(key)
	m.Delete(ctx.TypeCtx, key)
	return val, nil
}

func mapHasKey(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	m := args[0].(*values.Map)
	key := args[1].(string)
	_, ok := m.Get(key)
	return ok, nil
}

func initMapModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", mapLength)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "keys", mapKeys(rt.GetTypeEnv()))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "remove", mapRemove)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "hasKey", mapHasKey)
}

func init() {
	runtime.RegisterModuleInitializer(initMapModule)
}
