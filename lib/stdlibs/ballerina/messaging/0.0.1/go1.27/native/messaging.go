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
	orgName         = "ballerina"
	moduleName      = "messaging"
	onMessageMethod = "onMessage"
)

func initMessagingModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "invokeOnMessage", invokeOnMessage)
}

func init() {
	runtime.RegisterModuleInitializer(initMessagingModule)
}

func invokeOnMessage(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	svc := args[0].(*values.Object)
	h, _ := ctx.LookupRemoteMethod(svc, onMessageMethod)
	return ctx.InvokeMethod(h, []values.BalValue{svc, args[1]})
}
