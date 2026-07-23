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
	"math"
	"strconv"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.int"
)

func intToHexString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	n := args[0].(int64)
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
}

func intFromString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return values.NewErrorWithMessage("'int' from string: invalid number format: " + s), nil
	}
	return n, nil
}

// intFromHexString treats n as the unsigned magnitude; the largest
// representable magnitude is 2^63 (int64 min). It computes -n via
// -(n-1)-1 so the n == 2^63 case doesn't overflow an intermediate int64.
func intFromHexString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	negative := false
	input := s
	if len(input) > 0 && (input[0] == '-' || input[0] == '+') {
		negative = input[0] == '-'
		input = input[1:]
	}
	if len(input) == 0 {
		return values.NewErrorWithMessage("invalid hex string: \"" + s + "\""), nil
	}
	n, err := strconv.ParseUint(input, 16, 64)
	if err != nil {
		return values.NewErrorWithMessage("invalid hex string: \"" + s + "\""), nil
	}
	const maxMagnitude = uint64(math.MaxInt64) + 1
	if negative {
		if n > maxMagnitude {
			return values.NewErrorWithMessage("invalid hex string: \"" + s + "\""), nil
		}
		return -int64(n-1) - 1, nil
	}
	if n > math.MaxInt64 {
		return values.NewErrorWithMessage("invalid hex string: \"" + s + "\""), nil
	}
	return int64(n), nil
}

func initIntModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toHexString", intToHexString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromString", intFromString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromHexString", intFromHexString)
}

func init() {
	runtime.RegisterModuleInitializer(initIntModule)
}
