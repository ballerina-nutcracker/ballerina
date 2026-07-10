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

package floatruntime

import (
	"math"

	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.float"
)

func initFloatModule(rt *runtime.Runtime) {
	reg := func(name string, f extern.NativeFunc) {
		runtime.RegisterExternFunction(rt, orgName, moduleName, name, func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return f(nil, args)
		})
	}

	reg("abs", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return math.Abs(args[0].(float64)), nil
	})
	reg("ceiling", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return math.Ceil(args[0].(float64)), nil
	})
	reg("floor", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return math.Floor(args[0].(float64)), nil
	})
	reg("round", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		// Ballerina float:round is IEEE roundToIntegralTiesToEven, not half-away-from-zero.
		return math.RoundToEven(args[0].(float64)), nil
	})
	reg("sqrt", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return math.Sqrt(args[0].(float64)), nil
	})
	reg("pow", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return math.Pow(args[0].(float64), args[1].(float64)), nil
	})
}

func init() {
	runtime.RegisterModuleInitializer(initFloatModule)
}
