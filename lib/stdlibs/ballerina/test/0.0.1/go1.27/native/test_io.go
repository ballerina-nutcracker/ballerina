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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// bracketChars/specialChars mirror StringUtils#BRACKET_CHARACTERS/SPECIAL_CHARACTERS.
var bracketChars = []string{"{", "}", "[", "]", "(", ")"}
var specialChars = []string{",", `\n`, `\r`, `\t`, "\n", "\r", "\t", `"`, `\`, "!", "`"}

// urlEncodeChar matches Java's URLEncoder.encode behavior for the single
// characters this module ever encodes (ASCII punctuation/whitespace only).
func urlEncodeChar(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		fmt.Fprintf(&b, "%%%02X", s[i])
	}
	return b.String()
}

func encodeChars(s string, chars []string) string {
	for _, c := range chars {
		if strings.Contains(s, c) {
			s = strings.ReplaceAll(s, c, urlEncodeChar(c))
		}
	}
	return s
}

// isBalanced mirrors StringUtils#isBalanced.
func isBalanced(s string) bool {
	var stack []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			stack = append(stack, s[i])
		case ')', ']', '}':
			if len(stack) == 0 {
				return false
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			switch s[i] {
			case ')':
				if top == '{' || top == '[' {
					return false
				}
			case '}':
				if top == '(' || top == '[' {
					return false
				}
			case ']':
				if top == '(' || top == '{' {
					return false
				}
			}
		}
	}
	return len(stack) == 0
}

// escapeSpecialCharacters mirrors jballerina's StringUtils#escapeSpecialCharacters.
func escapeSpecialCharacters(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	key, _ := args[0].(string)
	if !isBalanced(key) {
		key = encodeChars(key, bracketChars)
	}
	key = encodeChars(key, specialChars)
	return key, nil
}

// matchWildcard mirrors jballerina's StringUtils#matchWildcard.
func matchWildcard(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	functionName, _ := args[0].(string)
	functionPattern, _ := args[1].(string)

	matched, err := MatchWildcard(functionName, functionPattern)
	if err != nil {
		return values.NewErrorWithMessage("Invalid wildcard pattern: " + err.Error()), nil
	}
	return matched, nil
}

// MatchWildcard mirrors jballerina's StringUtils#matchWildcard: name matches
// pattern where '*' matches any run of characters, everything else literal.
// Exported so cli/internal/testfilter can reuse the exact same algorithm for
// --tests wildcard matching, computed in Go ahead of registration, instead
// of duplicating it.
func MatchWildcard(name, pattern string) (bool, error) {
	segments := strings.Split(pattern, "*")
	quoted := make([]string, len(segments))
	for i, seg := range segments {
		quoted[i] = regexp.QuoteMeta(seg)
	}
	goPattern := strings.Join(quoted, ".*")

	re, err := regexp.Compile("^(?:" + goPattern + ")$")
	if err != nil {
		return false, err
	}
	return re.MatchString(name), nil
}

// isSystemConsole approximates jballerina's System.console() != null check via a TTY test.
func isSystemConsole(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false, nil
	}
	return (fi.Mode() & os.ModeCharDevice) != 0, nil
}

// currentTimeInMillis mirrors jballerina's CommonUtils#currentTimeInMillis.
func currentTimeInMillis(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return decimal.FromInt64(time.Now().UnixMilli()), nil
}

// printValue writes a value's informal string representation directly to
// stdout with no trailing separator — the leaf `print` jballerina's own
// println (a thin pure-Ballerina wrapper, see external.bal) calls in a loop.
//
// Routed through ctx.Env.Platform.IO.Stdout (the same platform-abstraction
// path ballerina/io's own println uses — see io.go's Write) rather than
// fmt.Print directly to os.Stdout: writing straight to os.Stdout bypassed
// that abstraction entirely, which corpus's test harness (via a swapped-in
// test pal.Platform) and any other stdout-virtualizing host rely on to
// capture output — confirmed as a real gap while adding ballerina/test's
// first corpus test coverage, not a hypothetical concern.
func printValue(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	if args[0] == nil {
		return nil, nil
	}
	visited := make(map[uintptr]bool)
	_, err := ctx.Env.Platform.IO.Stdout([]byte(values.String(args[0], visited)))
	return nil, err
}

// splitString mirrors jballerina's own (pure-Ballerina-wrapped) `split`,
// which itself delegates to Java's String#split — a REGEX split. This port
// only ever calls split with a literal "," delimiter (see filter.bal), so a
// literal (non-regex) split is behaviorally identical for every real call
// site; not a general regex-split substitute.
func splitStringFactory(env semtypes.Env) extern.NativeFunc {
	ld := semtypes.NewListDefinition()
	stringArrTy := ld.Define(env, nil, semtypes.ListRest(semtypes.String))
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		receiver, _ := args[0].(string)
		delimiter, _ := args[1].(string)
		parts := strings.Split(receiver, delimiter)
		items := make([]values.BalValue, len(parts))
		for i, p := range parts {
			items[i] = p
		}
		atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), stringArrTy)
		return values.NewList(stringArrTy, atomic, false, nil, 0, items), nil
	}
}

// invokeFunction dynamically invokes an arbitrary-signature function value
// with a runtime-determined argument list — substitutes for jballerina's
// `function:call(fn, ...params)` (see TODO.md: rest-argument spread at call
// sites, `...params`, isn't implemented, and no `lang.function` module exists
// in this port). Takes params as a plain array (not a rest parameter) so
// callers never need `...` spread to invoke it. A panic inside fn (e.g. an
// assertion failure) surfaces as a returned Go error here, which re-panics at
// this native's own call boundary — exactly what `trap` at the Ballerina call
// site is written to catch, matching jballerina's `trap function:call(...)`.
func invokeFunction(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	fn, ok := args[0].(*values.Function)
	if !ok {
		return nil, fmt.Errorf("invokeFunction: not a function value")
	}
	var params []values.BalValue
	if list, ok := args[1].(*values.List); ok {
		params = make([]values.BalValue, list.Len())
		for i := 0; i < list.Len(); i++ {
			params[i] = list.Get(i)
		}
	}
	return ctx.InvokeFunctionValue(fn, params)
}

// --- rerun_test.json / module_status.json narrow, record-shaped (de)serialization ---
//
// jballerina threads a generic `map<ModuleRerunJson>` (built via `json`/
// `.toString()`/`fromJsonStringWithType()`, all runtime built-ins in Java)
// through pure Ballerina code. None of those generic JSON primitives exist in
// this port yet (see TODO.md), so these natives narrowly (de)serialize only
// the two fixed record shapes this module actually needs, using Go's
// encoding/json — a reasonable adaptation of the same "native leaf function"
// pattern jballerina itself uses for JSON conversion, just scoped to this
// module's two concrete shapes instead of being fully generic.

type rerunEntryJSON struct {
	TestNames       []string            `json:"testNames"`
	TestModuleNames map[string]*string  `json:"testModuleNames"`
	SubTestNames    map[string][]string `json:"subTestNames"`
}

func readRerunFile(path string) (map[string]rerunEntryJSON, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]rerunEntryJSON{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out map[string]rerunEntryJSON
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// readModuleRerunEntry reads rerun_test.json and returns the single named
// module's entry as a ModuleRerunJson-shaped record value.
func readModuleRerunEntryFactory(env semtypes.Env) extern.NativeFunc {
	ld := semtypes.NewListDefinition()
	stringArrTy := ld.Define(env, nil, semtypes.ListRest(semtypes.String))
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		path, _ := args[0].(string)
		moduleName, _ := args[1].(string)

		all, err := readRerunFile(path)
		if err != nil {
			return values.NewErrorWithMessage("Invalid failed test data. Please run `bal test` command."), nil
		}
		entry, found := all[moduleName]
		if !found {
			return values.NewErrorWithMessage("Invalid failed test data. Please run `bal test` command."), nil
		}
		return rerunEntryToRecord(ctx, stringArrTy, entry), nil
	}
}

func rerunEntryToRecord(ctx *extern.Context, stringArrTy semtypes.SemType, entry rerunEntryJSON) *values.Map {
	listAtomic := semtypes.ToListAtomicType(ctx.TypeEnv(), stringArrTy)

	testNameItems := make([]values.BalValue, len(entry.TestNames))
	for i, n := range entry.TestNames {
		testNameItems[i] = n
	}
	testNamesList := values.NewList(stringArrTy, listAtomic, false, nil, 0, testNameItems)

	mapAtomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), semtypes.Mapping)
	moduleNamesEntries := make([]values.MapEntry, 0, len(entry.TestModuleNames))
	for k, v := range entry.TestModuleNames {
		var val values.BalValue
		if v != nil {
			val = *v
		}
		moduleNamesEntries = append(moduleNamesEntries, values.MapEntry{Key: k, Value: val})
	}
	testModuleNamesMap := values.NewMap(semtypes.Mapping, mapAtomic, false, moduleNamesEntries)

	subTestEntries := make([]values.MapEntry, 0, len(entry.SubTestNames))
	for k, v := range entry.SubTestNames {
		items := make([]values.BalValue, len(v))
		for i, s := range v {
			items[i] = s
		}
		subTestEntries = append(subTestEntries, values.MapEntry{
			Key: k, Value: values.NewList(stringArrTy, listAtomic, false, nil, 0, items),
		})
	}
	subTestNamesMap := values.NewMap(semtypes.Mapping, mapAtomic, false, subTestEntries)

	recordEntries := []values.MapEntry{
		{Key: "testNames", Value: testNamesList},
		{Key: "testModuleNames", Value: testModuleNamesMap},
		{Key: "subTestNames", Value: subTestNamesMap},
	}
	return values.NewMap(semtypes.Mapping, mapAtomic, false, recordEntries)
}

// writeModuleRerunEntry reads any existing rerun_test.json, replaces (or
// adds) the named module's entry, and writes the file back.
func writeModuleRerunEntry(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	path, _ := args[0].(string)
	moduleName, _ := args[1].(string)
	testNames := stringListValues(args[2])
	testModuleNames := stringOptMapValues(args[3])
	subTestNames := stringListMapValues(args[4])

	all, err := readRerunFile(path)
	if err != nil {
		all = map[string]rerunEntryJSON{}
	}
	all[moduleName] = rerunEntryJSON{
		TestNames:       testNames,
		TestModuleNames: testModuleNames,
		SubTestNames:    subTestNames,
	}

	data, err := json.Marshal(all)
	if err != nil {
		return values.NewErrorWithMessage(err.Error()), nil
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return values.NewErrorWithMessage(err.Error()), nil
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return values.NewErrorWithMessage(err.Error()), nil
	}
	return nil, nil
}

func stringOptMapValues(v values.BalValue) map[string]*string {
	m, ok := v.(*values.Map)
	if !ok {
		return nil
	}
	out := make(map[string]*string, m.Len())
	for _, k := range m.Keys() {
		val, _ := m.Get(k)
		if s, ok := val.(string); ok {
			sCopy := s
			out[k] = &sCopy
		} else {
			out[k] = nil
		}
	}
	return out
}

func stringListMapValues(v values.BalValue) map[string][]string {
	m, ok := v.(*values.Map)
	if !ok {
		return nil
	}
	out := make(map[string][]string, m.Len())
	for _, k := range m.Keys() {
		val, _ := m.Get(k)
		out[k] = stringListValues(val)
	}
	return out
}

// testStatusEntryJSON mirrors ballerina/test's TestStatusEntry record.
type testStatusEntryJSON struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	FailureMessage string `json:"failureMessage,omitempty"`
}

type moduleStatusReportJSON struct {
	TotalTests int                   `json:"totalTests"`
	Passed     int                   `json:"passed"`
	Failed     int                   `json:"failed"`
	Skipped    int                   `json:"skipped"`
	Tests      []testStatusEntryJSON `json:"tests"`
}

// writeModuleStatusReport mirrors jballerina's report.bal#moduleStatusReport,
// taking the already-computed fields directly (see TestStatusEntry in
// report.bal) instead of building a `map<json>` and calling `.toString()`.
func writeModuleStatusReport(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	totalTests, _ := args[1].(int64)
	passed, _ := args[2].(int64)
	failed, _ := args[3].(int64)
	skipped, _ := args[4].(int64)

	report := moduleStatusReportJSON{
		TotalTests: int(totalTests),
		Passed:     int(passed),
		Failed:     int(failed),
		Skipped:    int(skipped),
	}

	if list, ok := args[5].(*values.List); ok {
		report.Tests = make([]testStatusEntryJSON, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			entryMap, ok := list.Get(i).(*values.Map)
			if !ok {
				continue
			}
			entry := testStatusEntryJSON{}
			if v, ok := entryMap.Get("name"); ok {
				entry.Name, _ = v.(string)
			}
			if v, ok := entryMap.Get("status"); ok {
				entry.Status, _ = v.(string)
			}
			if v, ok := entryMap.Get("failureMessage"); ok {
				entry.FailureMessage, _ = v.(string)
			}
			report.Tests = append(report.Tests, entry)
		}
	}

	data, err := json.Marshal(report)
	if err != nil {
		return values.NewErrorWithMessage(err.Error()), nil
	}

	path, _ := args[0].(string)
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return values.NewErrorWithMessage(err.Error()), nil
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return values.NewErrorWithMessage(err.Error()), nil
	}
	return nil, nil
}
