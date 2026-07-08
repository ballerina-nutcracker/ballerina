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
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "tcp"
)

// tcpTypes holds the SemTypes shared by every native function in this
// module. Following this codebase's established stdlib convention (see
// ldap/crypto/file), byte arrays and generic maps are tagged with a single
// reusable SemType rather than a hand-built per-call one.
type tcpTypes struct {
	byteArrTy semtypes.SemType
}

func tcpError(msg string) *values.Error {
	return values.NewErrorWithMessage(msg)
}

func initTcpModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	byteArrBld := semtypes.NewListDefinition()
	types := tcpTypes{
		byteArrTy: byteArrBld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE),
	}

	registerClientFunctions(rt, types)
	registerListenerFunctions(rt, types)
}

func init() {
	runtime.RegisterModuleInitializer(initTcpModule)
}
