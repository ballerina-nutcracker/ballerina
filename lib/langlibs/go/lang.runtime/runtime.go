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
	"math"
	"time"

	"ballerina/decimal"
	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.runtime"
)

func initRuntimeModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sleep",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			seconds, _ := args[0].(*decimal.Decimal)
			rt.Platform().Time.Sleep(secondsToSleepDuration(seconds.Float64()))
			return nil, nil
		})
}

// secondsToSleepDuration converts a duration given in seconds to a
// time.Duration, clamping non-finite and out-of-int64-range results
// (reachable from decimal's much wider value range) to the max
// representable duration instead of relying on the platform-specific
// float-to-int64 overflow behavior. Non-positive and NaN durations pass
// through as a no-op, same as time.Sleep's own documented behavior.
func secondsToSleepDuration(seconds float64) time.Duration {
	nanos := seconds * float64(time.Second)
	if math.IsNaN(nanos) || nanos <= 0 {
		return 0
	}
	if nanos >= math.MaxInt64 {
		return math.MaxInt64
	}
	return time.Duration(nanos)
}

func init() {
	runtime.RegisterModuleInitializer(initRuntimeModule)
}
