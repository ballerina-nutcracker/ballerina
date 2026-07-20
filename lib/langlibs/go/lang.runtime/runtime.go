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
	"time"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.runtime"
)

func initRuntimeModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sleep",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			seconds, _ := args[0].(*decimal.Decimal)
			rt.Platform().Time.Sleep(time.Duration(seconds.Float64() * float64(time.Second)))
			return nil, nil
		})
}

func init() {
	runtime.RegisterModuleInitializer(initRuntimeModule)
}
