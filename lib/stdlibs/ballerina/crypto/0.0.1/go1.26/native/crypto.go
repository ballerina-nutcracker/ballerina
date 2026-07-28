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
	"ballerina/runtime"
	"ballerina/semtypes"
)

func initCryptoModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	byteArrBld := semtypes.NewListDefinition()
	keyMapBld := semtypes.NewMappingDefinition()
	utcBld := semtypes.NewListDefinition()
	types := cryptoTypes{
		byteArrTy: byteArrBld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE),
		keyMapTy:  keyMapBld.DefineMappingTypeWrapped(env, nil, semtypes.STRING),
		utcTy:     utcBld.TupleTypeWrappedRo(env, semtypes.INT, semtypes.DECIMAL),
	}
	registerHashFunctions(rt, types)
	registerHmacFunctions(rt, types)
	registerPasswordFunctions(rt, types)
	registerAesFunctions(rt, types)
	registerKeyFunctions(rt, types)
	registerRsaFunctions(rt, types)
	registerKdfFunctions(rt, types)
	registerUtilFunctions(rt, types)
}

func init() {
	runtime.RegisterModuleInitializer(initCryptoModule)
}
