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
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "test"
)

// externFail reports the failure to the host through PAL when a host installed
// a hook, and always yields the error value assertFail panics.
func externFail(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	msg, _ := args[0].(string)
	if fail := ctx.Env.Platform.Testing.Fail; fail != nil {
		fail(msg)
	}
	return values.NewErrorWithMessage(msg), nil
}

func initTestModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "externFail", externFail)
}

func init() {
	runtime.RegisterModuleInitializer(initTestModule)
}
