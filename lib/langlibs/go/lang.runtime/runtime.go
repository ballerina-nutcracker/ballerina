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

	// sleepPollInterval bounds how long we wait between yields, so a sleeping
	// strand still checks back in periodically instead of spinning as fast as
	// the scheduler will let it.
	sleepPollInterval = 10 * time.Millisecond
)

func runtimeSleep(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	seconds, ok := args[0].(*decimal.Decimal)
	if !ok {
		panic(fmt.Sprintf("internal error: unexpected seconds type %T", args[0]))
	}
	dur := time.Duration(seconds.Float64() * float64(time.Second))
	deadline := ctx.Env.Platform.Time.MonotonicNow() + dur
	// Wait out each interval before yielding, and always yield with a single
	// immediate receive (`<-ctx.Yield()`) rather than holding the returned
	// channel across other work, so strands sharing this one's cooperative
	// thread get a turn roughly every sleepPollInterval while this strand
	// sleeps.
	for {
		remaining := deadline - ctx.Env.Platform.Time.MonotonicNow()
		if remaining <= 0 {
			return nil, nil
		}
		<-ctx.Env.Platform.Time.After(min(remaining, sleepPollInterval))
		<-ctx.Yield()
	}
}

func initRuntimeModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sleep", runtimeSleep)
}

func init() {
	runtime.RegisterModuleInitializer(initRuntimeModule)
}
