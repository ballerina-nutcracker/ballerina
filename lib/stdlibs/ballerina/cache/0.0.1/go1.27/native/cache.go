// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName      = "ballerina"
	moduleName   = "cache"
	cacheHandle  = "$cache"
	expTimeField = "expTime"
)

func initCacheModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externInit", initExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externPut", putExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externGet", getExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externRemove", removeExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externRemoveAll", removeAllExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externHasKey", hasKeyExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externKeys", keysExtern(rt))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externSize", sizeExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCleanUp", cleanUpExtern)
}

func init() {
	runtime.RegisterModuleInitializer(initCacheModule)
}

func initExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self := args[0].(*values.Object)
	capacity, _ := self.Get("maxCapacity")
	evictionFactor, _ := self.Get("evictionFactor")
	self.Put(cacheHandle, newLRUCache(capacity.(int64), evictionFactor.(float64)))
	return nil, nil
}

func cacheOf(self values.BalValue) *lruCache {
	h, _ := self.(*values.Object).Get(cacheHandle)
	return h.(*lruCache)
}

func putExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	entry := args[2].(*values.Map)
	expTime, _ := entry.Get(expTimeField)
	cacheOf(args[0]).put(args[1].(string), entry, truncate(expTime.(*decimal.Decimal)))
	return nil, nil
}

func getExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	entry := cacheOf(args[0]).get(args[1].(string), truncate(args[2].(*decimal.Decimal)))
	if entry == nil {
		return nil, nil
	}
	return entry, nil
}

func removeExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cacheOf(args[0]).remove(args[1].(string))
	return nil, nil
}

func removeAllExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cacheOf(args[0]).removeAll()
	return nil, nil
}

func hasKeyExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return cacheOf(args[0]).hasKey(args[1].(string)), nil
}

func keysExtern(rt *runtime.Runtime) extern.NativeFunc {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	strArrTy := ld.Define(env, nil, semtypes.ListRest(semtypes.String))
	atomic := semtypes.ToListAtomicType(env, strArrTy)
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return values.NewList(strArrTy, atomic, false, nil, 0, cacheOf(args[0]).keys()), nil
	}
}

func sizeExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return int64(cacheOf(args[0]).size()), nil
}

func cleanUpExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	cacheOf(args[0]).cleanUp(truncate(args[1].(*decimal.Decimal)))
	return nil, nil
}

// truncate drops the fractional seconds of an epoch time the way jBallerina's
// `BigDecimal.longValue()` does, so expiry is compared at whole-second granularity.
func truncate(d *decimal.Decimal) int64 {
	whole, _ := d.Truncate()
	n, _, _ := whole.Int64()
	return n
}
