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

package xml

import (
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName        = "ballerina"
	moduleName     = "lang.xml"
	nextMethodName = "$xmlIterator.next"
)

func initXMLModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "iterator", xmlIterator)
	runtime.RegisterExternFunction(rt, orgName, moduleName, nextMethodName, xmlIteratorNext)
}

func xmlIterator(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	x, _ := args[0].(values.XMLValue)
	return values.NewObject(semtypes.OBJECT, map[string]values.BalValue{
		"items": x.IterItems(),
		"idx":   int64(0),
	}, map[string]string{
		"next": orgName + "/" + moduleName + ":" + nextMethodName,
	}, nil), nil
}

func xmlIteratorNext(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	it := args[0].(*values.Object)
	itemsValue, _ := it.Get("items")
	idxValue, _ := it.Get("idx")
	items := itemsValue.([]values.XMLValue)
	idx := idxValue.(int64)
	if idx >= int64(len(items)) {
		return nil, nil
	}
	it.Put("idx", idx+1)
	recordTy := xmlIteratorNextRecordType(ctx.Env.TypeEnv)
	return values.NewMap(recordTy, semtypes.ToMappingAtomicType(ctx.TypeCtx, recordTy), false, []values.MapEntry{{
		Key:   "value",
		Value: items[idx],
	}}), nil
}

func xmlIteratorNextRecordType(env semtypes.Env) semtypes.SemType {
	def := semtypes.NewMappingDefinition()
	return def.DefineMappingTypeWrapped(env,
		[]semtypes.Field{semtypes.FieldFrom("value", semtypes.XML, false, false)},
		semtypes.NEVER)
}

func init() {
	runtime.RegisterModuleInitializer(initXMLModule)
}
