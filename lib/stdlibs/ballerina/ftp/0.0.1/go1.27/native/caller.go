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
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

var callerMethods = []string{"get", "getBytes", "getText", "getJson", "getXml", "getCsv", "getBytesAsStream",
	"append", "put", "putBytes", "putText", "putJson", "putXml", "putCsv", "putBytesAsStream", "putCsvAsStream",
	"mkdir", "rmdir", "rename", "move", "copy", "exists", "size", "list", "isDirectory", "delete"}

// The Caller's methods are natives that forward to the wrapped Client; in Ballerina each call
// would need a lock around the final field (ballerina-nutcracker/ballerina#1431).
func (m *ftpModule) registerCallerExterns() {
	for _, name := range callerMethods {
		runtime.RegisterExternFunction(m.rt, orgName, moduleName, "Caller.$remote$"+name, forwardToClient(name))
	}
	runtime.RegisterExternFunction(m.rt, orgName, moduleName, "Caller.closeClient", forwardToClient("close"))
}

func callerClient(caller *values.Object) *values.Object {
	v, _ := caller.Get("client")
	client, _ := v.(*values.Object)
	return client
}

func forwardToClient(name string) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		client := callerClient(args[0].(*values.Object))
		handle, ok := ctx.LookupRemoteMethod(client, name)
		if !ok {
			return values.NewErrorWithMessage("client method invocation failed: " + name + " not found"), nil
		}
		forwarded := append([]values.BalValue{client}, args[1:]...)
		return ctx.InvokeMethod(handle, forwarded)
	}
}
