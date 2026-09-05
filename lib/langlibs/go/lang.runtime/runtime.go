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

package langruntime

import (
	"fmt"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.runtime"
)

func runtimeSleep(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	seconds, ok := args[0].(*decimal.Decimal)
	if !ok {
		panic(fmt.Sprintf("internal error: unexpected seconds type %T", args[0]))
	}
	dur := time.Duration(seconds.Float64() * float64(time.Second))
	deadline := ctx.Env.Platform.Time.MonotonicNow() + dur
	// Hand the thread back on every pass, the same way the wait actions poll
	// (see waitAllFutures). Blocking on a timer between yields would hold this
	// strand's turn for the whole timer, stalling every other strand sharing
	// the thread.
	for ctx.Env.Platform.Time.MonotonicNow() < deadline {
		<-ctx.Yield()
	}
	return nil, nil
}

func initRuntimeModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sleep", runtimeSleep)
}

func init() {
	runtime.RegisterModuleInitializer(initRuntimeModule)
}
