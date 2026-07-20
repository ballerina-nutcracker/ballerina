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
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
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

	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromString", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be a string")
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return values.NewErrorWithMessage("'int' from string: invalid number format: " + s), nil
		}
		return n, nil
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromHexString", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be a string")
		}
		negative := false
		input := s
		if len(input) > 0 && input[0] == '-' {
			negative = true
			input = input[1:]
		}
		if len(input) == 0 {
			return values.NewErrorWithMessage("invalid hex string: \"" + s + "\""), nil
		}
		n, err := strconv.ParseUint(input, 16, 64)
		if err != nil {
			return values.NewErrorWithMessage("invalid hex string: \"" + s + "\""), nil
		}
		result := int64(n)
		if negative {
			result = -result
		}
		return result, nil
	})
}

func init() {
	runtime.RegisterModuleInitializer(initIntModule)
}
