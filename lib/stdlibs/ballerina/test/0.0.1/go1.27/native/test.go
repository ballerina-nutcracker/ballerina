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

package native

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "test"
)

// getBallerinaType mirrors jballerina's BallerinaTypeCheck#getBallerinaType.
func getBallerinaType(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	// Widen past the per-value singleton SemTypeForValue returns (e.g. the exact
	// string "hello") to the general basic type ("string"), matching jballerina.
	ty := semtypes.WidenToBasicTypes(values.SemTypeForValue(args[0]))
	return semtypes.ToString(ctx.TypeCtx(), ty), nil
}

// getStringDiff mirrors jballerina's AssertionDiffEvaluator#getStringDiff.
func getStringDiff(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	actual, _ := args[0].(string)
	expected, _ := args[1].(string)
	return unifiedStringDiff(actual, expected), nil
}

// getKeysDiff mirrors jballerina's AssertionDiffEvaluator#getKeysDiff.
func getKeysDiff(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	actualKeys := stringListValues(args[0])
	expectedKeys := stringListValues(args[1])

	expectedOnly := keysNotIn(expectedKeys, actualKeys)
	actualOnly := keysNotIn(actualKeys, expectedKeys)

	var b strings.Builder
	if len(expectedOnly) > 0 {
		b.WriteString("\nexpected keys\t:")
		b.WriteString(joinWithLeadingSpace(expectedOnly))
		b.WriteString("\n")
	}
	if len(actualOnly) > 0 {
		b.WriteString("actual keys\t:")
		b.WriteString(joinWithLeadingSpace(actualOnly))
	}
	return b.String(), nil
}

// unifiedStringDiff renders a unified-diff-style comparison, matching the visible shape of
// jballerina's java-diff-utils-backed output. Not verified byte-for-byte against it.
func unifiedStringDiff(actual, expected string) string {
	actualLines := chunkLines(actual)
	expectedLines := chunkLines(expected)
	ops := diffLines(actualLines, expectedLines)

	hasDiff := false
	for _, op := range ops {
		if op.kind != ' ' {
			hasDiff = true
			break
		}
	}
	if !hasDiff {
		return "\n"
	}

	rawLines := make([]string, 0, len(ops)+3)
	rawLines = append(rawLines, "--- actual", "+++ expected",
		fmt.Sprintf("@@ -1,%d +1,%d @@", len(actualLines), len(expectedLines)))
	for _, op := range ops {
		rawLines = append(rawLines, string(op.kind)+op.line)
	}

	output := "\n"
	for _, line := range rawLines {
		switch {
		case strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-"):
			if strings.HasSuffix(output, "\n") {
				output += line + "\n"
			} else {
				output += "\n" + line + "\n"
			}
		case strings.HasPrefix(line, "@@ -"):
			output += "\n" + line + "\n\n"
		default:
			output += line + "\n"
		}
	}
	return strings.ReplaceAll(output, "\n\n", " \n \n ")
}

type diffOp struct {
	kind byte // ' ' (context), '-' (removed from actual), '+' (added in expected)
	line string
}

// chunkLines mirrors jballerina's AssertionDiffEvaluator#getValueList.
func chunkLines(s string) []string {
	const maxLen = 80
	var out []string
	if strings.Contains(s, "\n") {
		for _, line := range strings.Split(s, "\n") {
			out = append(out, chunkByLength(line, maxLen)...)
		}
		return out
	}
	return chunkByLength(s, maxLen)
}

func chunkByLength(s string, n int) []string {
	runes := []rune(s)
	if len(runes) <= n {
		return []string{s}
	}
	out := make([]string, 0, len(runes)/n+1)
	for i := 0; i < len(runes); i += n {
		end := min(i+n, len(runes))
		out = append(out, string(runes[i:end]))
	}
	return out
}

// diffLines computes a line-level diff via LCS dynamic programming.
func diffLines(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{' ', a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{'-', a[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', b[j]})
	}
	return ops
}

func stringListValues(v values.BalValue) []string {
	list, ok := v.(*values.List)
	if !ok {
		return nil
	}
	out := make([]string, list.Len())
	for i := 0; i < list.Len(); i++ {
		out[i], _ = list.Get(i).(string)
	}
	return out
}

// keysNotIn returns the keys of source that don't appear in other, preserving order.
func keysNotIn(source, other []string) []string {
	var diff []string
	for _, k := range source {
		found := false
		for _, o := range other {
			if k == o {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, k)
		}
	}
	return diff
}

func joinWithLeadingSpace(keys []string) string {
	var b strings.Builder
	for i, k := range keys {
		b.WriteString(" ")
		b.WriteString(k)
		if i != len(keys)-1 {
			b.WriteString(",")
		}
	}
	return b.String()
}

// sprintf mirrors jballerina's StringUtils#sprintf. Only %s is exercised by this module
// (assert.bal); other specifiers are best-effort, unverified against jballerina.
func sprintf(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	format, _ := args[0].(string)
	restArgs := stringifyRestArgs(args)

	var b strings.Builder
	argIdx := 0
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' || i+1 >= len(format) {
			b.WriteByte(c)
			continue
		}

		j := i + 1
		padding := ""
		for j < len(format) && (isDigit(format[j]) || format[j] == '.') {
			padding += string(format[j])
			j++
		}
		if j >= len(format) {
			b.WriteByte(c)
			continue
		}
		specifier := format[j]

		if specifier == '%' {
			b.WriteByte('%')
			i = j
			continue
		}

		if argIdx >= len(restArgs) {
			return nil, fmt.Errorf("not enough arguments for format string")
		}
		formatted, err := formatSpecifier(specifier, padding, restArgs[argIdx])
		if err != nil {
			return nil, err
		}
		b.WriteString(formatted)
		argIdx++
		i = j
	}
	return b.String(), nil
}

// stringifyRestArgs unpacks the `(any|error)... args` rest parameter: each
// call-site argument arrives as its own entry in args[1:], not a *values.List.
func stringifyRestArgs(args []values.BalValue) []values.BalValue {
	if len(args) < 2 {
		return nil
	}
	return args[1:]
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func formatSpecifier(specifier byte, padding string, arg values.BalValue) (string, error) {
	switch specifier {
	case 's':
		if arg == nil {
			return "", nil
		}
		visited := make(map[uintptr]bool)
		return applyPadding(padding, values.String(arg, visited)), nil
	case 'd':
		n, ok := arg.(int64)
		if !ok {
			return "", fmt.Errorf("%%d requires an int argument")
		}
		return applyPadding(padding, strconv.FormatInt(n, 10)), nil
	case 'f':
		f, ok := arg.(float64)
		if !ok {
			return "", fmt.Errorf("%%f requires a float argument")
		}
		return applyPadding(padding, strconv.FormatFloat(f, 'f', -1, 64)), nil
	case 'b', 'B':
		bv, ok := arg.(bool)
		if !ok {
			return "", fmt.Errorf("%%b requires a boolean argument")
		}
		s := strconv.FormatBool(bv)
		if specifier == 'B' {
			s = strings.ToUpper(s)
		}
		return applyPadding(padding, s), nil
	case 'x', 'X':
		n, ok := arg.(int64)
		if !ok {
			return "", fmt.Errorf("%%x requires an int argument")
		}
		s := strconv.FormatInt(n, 16)
		if specifier == 'X' {
			s = strings.ToUpper(s)
		}
		return applyPadding(padding, s), nil
	default:
		return "", fmt.Errorf("invalid format specifier: %%%c", specifier)
	}
}

func applyPadding(padding, s string) string {
	if padding == "" {
		return s
	}
	width, err := strconv.Atoi(strings.TrimSuffix(padding, "."))
	if err != nil || width <= len(s) {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

func initTestModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()

	runtime.RegisterExternFunction(rt, orgName, moduleName, "getBallerinaType", getBallerinaType)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "getStringDiff", getStringDiff)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "getKeysDiff", getKeysDiff)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "sprintf", sprintf)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "matchWildcard", matchWildcard)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "escapeSpecialCharacters", escapeSpecialCharacters)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "isSystemConsole", isSystemConsole)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "currentTimeInMillis", currentTimeInMillis)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "printValue", printValue)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "split", splitStringFactory(env))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "invokeFunction", invokeFunction)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "readModuleRerunEntry", readModuleRerunEntryFactory(env))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "writeModuleRerunEntry", writeModuleRerunEntry)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "writeModuleStatusReport", writeModuleStatusReport)
}

func init() {
	runtime.RegisterModuleInitializer(initTestModule)
}
