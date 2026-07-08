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

package decimalruntime

import (
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.decimal"
)

func initDecimalModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sum", decimalSum)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "max", decimalMax)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "min", decimalMin)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "abs", decimalAbs)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "round", decimalRound)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "quantize", decimalQuantize)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "floor", decimalFloor)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ceiling", decimalCeiling)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromString", decimalFromString)
}

func decimalSum(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	out := decimal.FromInt64(0)
	for _, arg := range args {
		var err *decimal.Error
		out, err = out.Add(arg.(*decimal.Decimal))
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func decimalMax(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	out := args[0].(*decimal.Decimal)
	for _, arg := range args[1:] {
		n := arg.(*decimal.Decimal)
		if n.Cmp(out) > 0 {
			out = n
		}
	}
	return out, nil
}

func decimalMin(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	out := args[0].(*decimal.Decimal)
	for _, arg := range args[1:] {
		n := arg.(*decimal.Decimal)
		if n.Cmp(out) < 0 {
			out = n
		}
	}
	return out, nil
}

func decimalAbs(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return args[0].(*decimal.Decimal).Abs(), nil
}

func decimalRound(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return decimalResult(args[0].(*decimal.Decimal).Round(args[1].(int64)))
}

func decimalQuantize(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return decimalResult(args[0].(*decimal.Decimal).Quantize(args[1].(*decimal.Decimal)))
}

func decimalFloor(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return decimalResult(args[0].(*decimal.Decimal).Floor())
}

func decimalCeiling(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return decimalResult(args[0].(*decimal.Decimal).Ceiling())
}

func decimalFromString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	n, err := decimal.FromLiteral(args[0].(string))
	if err != nil {
		return values.NewErrorWithMessage(err.Error()), nil
	}
	return n, nil
}

func decimalResult(out *decimal.Decimal, err *decimal.Error) (values.BalValue, error) {
	if err != nil {
		return nil, err
	}
	return out, nil
}

func init() {
	runtime.RegisterModuleInitializer(initDecimalModule)
}
