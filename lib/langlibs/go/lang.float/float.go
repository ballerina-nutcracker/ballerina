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
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.float"
)

var (
	decimalFloatingPointStringPattern     = regexp.MustCompile(`^(?:NaN|[+-]?(?:Infinity|(?:(?:0|[1-9][0-9]*)|(?:(?:0|[1-9][0-9]*)\.[0-9]+|\.[0-9]+)(?:[eE][+-]?[0-9]+)?|(?:0|[1-9][0-9]*)[eE][+-]?[0-9]+)))$`)
	hexadecimalFloatingPointStringPattern = regexp.MustCompile(`^(?:NaN|[+-]?(?:Infinity|0[xX](?:(?:[0-9A-Fa-f]+\.[0-9A-Fa-f]+|\.[0-9A-Fa-f]+)(?:[pP][+-]?[0-9]+)?|[0-9A-Fa-f]+[pP][+-]?[0-9]+)))$`)
)

func initFloatModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "isFinite", floatIsFinite)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "isInfinite", floatIsInfinite)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "isNaN", floatIsNaN)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sum", floatSum)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "max", floatMax)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "min", floatMin)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "abs", unaryMath(math.Abs))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "round", floatRound)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "floor", unaryMath(math.Floor))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ceiling", unaryMath(math.Ceil))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sqrt", unaryMath(math.Sqrt))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "cbrt", unaryMath(math.Cbrt))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "pow", floatPow)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "log", unaryMath(math.Log))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "log10", unaryMath(math.Log10))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "exp", unaryMath(math.Exp))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sin", unaryMath(math.Sin))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "cos", unaryMath(math.Cos))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "tan", unaryMath(math.Tan))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "acos", unaryMath(math.Acos))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "atan", unaryMath(math.Atan))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "asin", unaryMath(math.Asin))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "atan2", floatAtan2)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sinh", unaryMath(math.Sinh))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "cosh", unaryMath(math.Cosh))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "tanh", unaryMath(math.Tanh))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromString", floatFromString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toHexString", floatToHexString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromHexString", floatFromHexString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBitsInt", floatToBitsInt)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBitsInt", floatFromBitsInt)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toFixedString", floatToFixedString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toExpString", floatToExpString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "avg", floatAvg)
}

func floatIsFinite(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	x := args[0].(float64)
	return !math.IsNaN(x) && !math.IsInf(x, 0), nil
}

func floatIsInfinite(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return math.IsInf(args[0].(float64), 0), nil
}

func floatIsNaN(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return math.IsNaN(args[0].(float64)), nil
}

func floatSum(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	out := 0.0
	for _, arg := range args {
		out += arg.(float64)
	}
	return out, nil
}

func floatMax(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	out := math.Inf(-1)
	for _, arg := range args {
		out = math.Max(out, arg.(float64))
	}
	return out, nil
}

func floatMin(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	out := math.Inf(1)
	for _, arg := range args {
		out = math.Min(out, arg.(float64))
	}
	return out, nil
}

func unaryMath(fn func(float64) float64) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return fn(args[0].(float64)), nil
	}
}

func floatRound(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	x := args[0].(float64)
	fractionDigits := args[1].(int64)
	if x == 0 || math.IsNaN(x) || math.IsInf(x, 0) {
		return x, nil
	}
	if fractionDigits > 308 {
		return x, nil
	}
	if fractionDigits < -323 {
		return 0.0, nil
	}
	scale := math.Pow10(int(fractionDigits))
	scaled := x * scale
	if math.IsInf(scaled, 0) {
		return x, nil
	}
	rounded := math.RoundToEven(scaled)
	if rounded == 0 {
		return 0.0, nil
	}
	return rounded / scale, nil
}

func floatPow(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return math.Pow(args[0].(float64), args[1].(float64)), nil
}

func floatAtan2(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return math.Atan2(args[0].(float64), args[1].(float64)), nil
}

func floatFromString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	if !decimalFloatingPointStringPattern.MatchString(s) {
		return values.NewErrorWithMessage("invalid decimal floating point string: " + s), nil
	}
	return parseFloat(s)
}

func floatFromHexString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	if !hexadecimalFloatingPointStringPattern.MatchString(s) {
		return values.NewErrorWithMessage("invalid hexadecimal floating point string: " + s), nil
	}
	return parseFloat(normalizeHexFloatString(s))
}

func normalizeHexFloatString(s string) string {
	unsigned := strings.TrimPrefix(strings.TrimPrefix(s, "+"), "-")
	if unsigned == "NaN" || unsigned == "Infinity" || strings.ContainsAny(unsigned, "pP") {
		return s
	}
	return s + "p0"
}

func parseFloat(s string) (values.BalValue, error) {
	out, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return values.NewErrorWithMessage(err.Error()), nil
	}
	return out, nil
}

func floatToHexString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	x := args[0].(float64)
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return values.FormatFloat(x), nil
	}
	return normalizeHexFloat(strconv.FormatFloat(x, 'x', -1, 64)), nil
}

func normalizeHexFloat(s string) string {
	idx := strings.LastIndexByte(s, 'p')
	exp := s[idx+1:]
	if exp[0] == '+' {
		exp = exp[1:]
	}
	sign := ""
	if strings.HasPrefix(exp, "-") {
		sign = "-"
		exp = exp[1:]
	}
	exp = strings.TrimLeft(exp, "0")
	if exp == "" {
		exp = "0"
	}
	return s[:idx+1] + sign + exp
}

func floatToBitsInt(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return int64(math.Float64bits(args[0].(float64))), nil
}

func floatFromBitsInt(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	out := math.Float64frombits(uint64(args[0].(int64)))
	if math.IsNaN(out) {
		return math.NaN(), nil
	}
	return out, nil
}

func floatToFixedString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	x := args[0].(float64)
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return values.FormatFloat(x), nil
	}
	if args[1] == nil {
		return formatMinimalFixed(x), nil
	}
	digits := args[1].(int64)
	if digits < 0 {
		panic(values.NewErrorWithMessage("fractionDigits must be non-negative"))
	}
	return fmt.Sprintf("%.*f", int(digits), x), nil
}

func floatToExpString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	x := args[0].(float64)
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return values.FormatFloat(x), nil
	}
	if args[1] == nil {
		return normalizeExp(formatMinimalExp(x)), nil
	}
	digits := args[1].(int64)
	if digits < 0 {
		panic(values.NewErrorWithMessage("fractionDigits must be non-negative"))
	}
	return normalizeExp(strconv.FormatFloat(x, 'e', int(digits), 64)), nil
}

func formatMinimalFixed(x float64) string {
	out := strconv.FormatFloat(x, 'f', -1, 64)
	if math.Abs(x) < 1e7 && math.Trunc(x) == x && !strings.ContainsRune(out, '.') {
		return out + ".0"
	}
	return out
}

func formatMinimalExp(x float64) string {
	out := strconv.FormatFloat(x, 'e', -1, 64)
	idx := strings.LastIndexByte(out, 'e')
	if idx < 0 || strings.ContainsRune(out[:idx], '.') {
		return out
	}
	exp, err := strconv.Atoi(out[idx+1:])
	if err != nil || math.Abs(x) >= 1e7 || math.Trunc(x) != x {
		return out
	}
	zeros := exp
	if zeros < 1 {
		zeros = 1
	}
	return out[:idx] + "." + strings.Repeat("0", zeros) + out[idx:]
}

func normalizeExp(s string) string {
	idx := strings.LastIndexByte(s, 'e')
	if idx < 0 {
		return s
	}
	exp, err := strconv.Atoi(s[idx+1:])
	if err != nil {
		return s
	}
	sign := "+"
	if exp < 0 {
		sign = "-"
		exp = -exp
	}
	return fmt.Sprintf("%se%s%02d", s[:idx], sign, exp)
}

func floatAvg(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	if len(args) == 0 {
		return math.NaN(), nil
	}
	sum, _ := floatSum(nil, args)
	return sum.(float64) / float64(len(args)), nil
}

func init() {
	runtime.RegisterModuleInitializer(initFloatModule)
}
