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

package intruntime

import (
	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/values"
	"fmt"
	"math"
	"strconv"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.int"
)

func initIntModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toHexString", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		n, ok := args[0].(int64)
		if !ok {
			return nil, fmt.Errorf("first argument must be an int")
		}
		switch {
		case n == 0:
			return "0", nil
		case n == math.MinInt64:
			return "-8000000000000000", nil
		case n < 0:
			return "-" + strconv.FormatInt(-n, 16), nil
		default:
			return strconv.FormatInt(n, 16), nil
		}
	})
}

func init() {
	runtime.RegisterModuleInitializer(initIntModule)
}
