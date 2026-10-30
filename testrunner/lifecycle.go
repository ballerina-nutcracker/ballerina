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

package testrunner

import (
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/testplan"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// HookFailure records a BeforeSuite/AfterSuite/BeforeEach/AfterEach/
// BeforeGroups/AfterGroups hook that panicked — distinct from a test Result,
// the same way the old engine prints these separately from pass/fail/skip
// counts rather than folding them into a named test's own outcome.
type HookFailure struct {
	Hook    string // e.g. "before test suite function"
	Message string
}

// RunReport is the outcome of one full Run: every test Result, every hook
// failure (also tracked separately from per-test results, the same way the
// old engine prints these apart from pass/fail/skip counts), and the full
// chronological Events stream WriteConsoleReport replays to reproduce the
// old engine's print-as-it-happens console output exactly.
type RunReport struct {
	Results      []Result
	HookFailures []HookFailure
	Events       []ConsoleEvent
}

// ConsoleEvent is one live console notice recorded in the exact order Run
// produced it — a hook failure (execute.bal#printExecutionError) or a test/
// row failure's "name[:suffix] has failed." notice
// (handleTestFuncOutput/handleDataDrivenTestOutput/isTestReadyToExecute).
// Exactly one field is set. A single executeOne call can interleave both
// kinds — e.g. a test's own body fails, then its AfterGroups hook also
// fails — so these can't be reconstructed later from Results/HookFailures
// alone; each must be recorded at the moment it actually happens.
type ConsoleEvent struct {
	Hook        *HookFailure
	TestFailure *TestFailureNotice
}

// TestFailureNotice is one "has failed" console line. HasSuffix is true for
// both a general test (Suffix == "", printed as "name: has failed.") and a
// data-driven row (Suffix == the row key, printed as "name:key has
// failed."); it is false only for a data provider itself failing before any
// row exists (isTestReadyToExecute's diagnostics path), printed with no
// colon at all: "name has failed."
type TestFailureNotice struct {
	Name      string
	Suffix    string
	HasSuffix bool
}

// Run executes an entire suite — BeforeSuite, then every included test in
// graph.Order with its BeforeEach/BeforeGroups/AfterEach/AfterGroups/before:/
// after: lifecycle (including data providers and sub-key filtering), then
// AfterSuite — a Go port of execute.bal/serialExecuter.bal's behavior, not
// just a bare invocation loop. See this package's doc comment (run.go) for
// what's deliberately not yet covered.
func Run(rt *runtime.Runtime, org, module string, plan *testplan.TestPlan, graph *testplan.Graph, included map[string]bool, subKeyFilters SubKeyFilters) RunReport {
	r := &runner{
		rt: rt, org: org, module: module, plan: plan, graph: graph, included: included,
		byName:         indexByName(plan.Tests),
		perTestSkip:    make(map[string]bool),
		groupTotal:     computeGroupTotals(plan, included),
		groupExecuted:  make(map[string]int),
		groupSkipAfter: make(map[string]bool),
		subKeyFilters:  subKeyFilters,
	}
	return r.run()
}

type runner struct {
	rt       *runtime.Runtime
	org      string
	module   string
	plan     *testplan.TestPlan
	graph    *testplan.Graph
	included map[string]bool
	byName   map[string]testplan.TestFunction

	// shouldSkip mirrors execute.bal's module-level `shouldSkip`: once a
	// BeforeSuite/BeforeEach/AfterEach hook fails, it is set and never
	// cleared — every remaining test and Each-hook in the whole run is
	// skipped, matching the real engine's behavior exactly (confirmed by
	// reading it directly: there is no reset function anywhere).
	shouldSkip           bool
	shouldAfterSuiteSkip bool

	// perTestSkip is set on a test by name when one of its dependencies
	// failed or was itself skipped (skip-cascade) or its own BeforeGroups
	// hook failed — mirrors ExecutionManager.setSkip/isSkip.
	perTestSkip map[string]bool

	// groupTotal/groupExecuted are register.bal's GroupStatusRegistry,
	// scoped to the included+enabled subset (registration-time counts in
	// the old engine only ever covered what was actually registered, which
	// is exactly today's `included` set).
	groupTotal     map[string]int
	groupExecuted  map[string]int
	groupSkipAfter map[string]bool

	// subKeyFilters restricts which data rows of a data-driven test actually
	// run, per --tests "name#key" entries — see subkey.go. Nil/empty means
	// no sub-key restriction is in effect for any test.
	subKeyFilters SubKeyFilters

	hookFailures []HookFailure
	events       []ConsoleEvent
}

func indexByName(tests []testplan.TestFunction) map[string]testplan.TestFunction {
	m := make(map[string]testplan.TestFunction, len(tests))
	for _, t := range tests {
		m[t.Name] = t
	}
	return m
}

// computeGroupTotals counts, per group, how many included and enabled tests
// belong to it — register.bal's incrementTotalTest(group, enabled) did this
// at registration time, and `included` is the modern equivalent of "got
// registered at all."
func computeGroupTotals(plan *testplan.TestPlan, included map[string]bool) map[string]int {
	totals := make(map[string]int)
	for _, t := range plan.Tests {
		if included != nil && !included[t.Name] {
			continue
		}
		if !t.Enable {
			continue
		}
		for _, g := range t.Groups {
			totals[g]++
		}
	}
	return totals
}

func (r *runner) run() RunReport {
	r.runSuiteHooks(r.plan.BeforeSuite, "before test suite function", func(failed bool) {
		if failed {
			r.shouldSkip = true
			r.shouldAfterSuiteSkip = true
		}
	})

	var results []Result
	for _, name := range r.graph.Order {
		if r.included != nil && !r.included[name] {
			continue
		}
		t := r.byName[name]
		if !t.Enable {
			// A disabled test is excluded entirely, not counted as skipped
			// (matches jballerina).
			continue
		}
		results = append(results, r.executeOne(t)...)
	}

	r.runAfterSuiteHooks()

	return RunReport{Results: results, HookFailures: r.hookFailures, Events: r.events}
}

// executeOne dispatches to the data-driven or plain-test path based on
// whether t has a data provider — the two share before/after-each and
// before/after-group handling but differ in invocation count and, per
// executeDataDrivenTestSet, in skip-cascade behavior.
func (r *runner) executeOne(t testplan.TestFunction) []Result {
	if t.DataProvider != "" {
		return r.executeDataDriven(t)
	}
	return []Result{r.executeSingle(t)}
}

// executeSingle mirrors serialExecuter.bal's executeTest/executeNonDataDrivenTest
// exactly: before-group hooks, before-each, the per-test before: hook (if
// any — a failure here reports this test Skip without invoking it at all
// and still cascades skip to dependents, same as being pre-skipped), the
// test itself, the per-test after: hook (if any — a failure here does NOT
// change the test's already-recorded Result, but its failure also cascades
// skip to dependents via the same `||` as the test's own fail status),
// group-execution-count bookkeeping, after-each, after-group hooks, then
// skip-cascade to dependents.
func (r *runner) executeSingle(t testplan.TestFunction) Result {
	r.runBeforeGroupHooks(t)
	r.runBeforeEachHooks()

	skip := r.perTestSkip[t.Name] || r.shouldSkip
	var result Result
	shouldSkipDependents := skip
	switch {
	case skip:
		result = Result{Name: t.Name, Status: Skip}
	case t.Before != "" && r.invokeHook(t.Before, "before test function for the test", false):
		result = Result{Name: t.Name, Status: Skip}
		shouldSkipDependents = true
	default:
		result = r.invoke(t.Name, t.Name, "", nil, GeneralTest)
		afterFailed := false
		if t.After != "" {
			afterFailed = r.invokeHook(t.After, "after test function for the test", false)
		}
		shouldSkipDependents = result.Status == Fail || afterFailed
	}

	for _, g := range t.Groups {
		r.groupExecuted[g]++
	}
	r.runAfterEachHooks()
	r.runAfterGroupHooks(t)

	if shouldSkipDependents {
		for _, dep := range r.graph.Dependents[t.Name] {
			r.perTestSkip[dep] = true
		}
	}
	return result
}

// executeDataDriven mirrors serialExecuter.bal's executeDataDrivenTestSet:
// the data provider is invoked once, up front, before any hook runs — a
// failure there reports the whole test failed (no "#key" suffix) and skips
// its entire lifecycle (BeforeGroups/BeforeEach/AfterEach/AfterGroups,
// group-execution-count), matching isTestReadyToExecute's diagnostics
// short-circuit. A data-driven test's dependents are never skip-cascaded
// regardless of how many rows fail — executeDataDrivenTestSet never
// propagates a shouldSkipDependents value, unlike the single-test path.
//
// Unlike BeforeEach/AfterEach (once per test function, above), a per-test
// before:/after: hook is invoked once per row — ported from
// prepareDataDrivenTest, which is itself called once per (key, value) pair
// in the original executeDataDrivenTestSet loop.
//
// A row whose key doesn't match a --tests "name#key" sub-key filter (see
// subkey.go) is dropped silently — no Result at all, matching
// skipDataDrivenTest's plain `return` with no reportData call — but its
// before:/after: hooks still run regardless, since prepareDataDrivenTest
// calls executeAfterFunction unconditionally and the skip decision lives
// inside executeDataDrivenTest, downstream of the before: hook.
func (r *runner) executeDataDriven(t testplan.TestFunction) []Result {
	rows, testType, err := r.dataProviderRows(t.DataProvider)
	if err != nil {
		r.events = append(r.events, ConsoleEvent{TestFailure: &TestFailureNotice{Name: t.Name, HasSuffix: false}})
		return []Result{{Name: t.Name, Status: Fail, Message: DataProviderFailedMessage, RawMessage: true}}
	}

	r.runBeforeGroupHooks(t)
	r.runBeforeEachHooks()

	skip := r.perTestSkip[t.Name] || r.shouldSkip
	var results []Result
	for _, row := range rows {
		name := t.Name + "#" + row.key
		switch {
		case skip:
			results = append(results, Result{Name: name, Status: Skip})
		case t.Before != "" && r.invokeHook(t.Before, "before test function for the test", false):
			// prepareDataDrivenTest reports this under the test's bare name,
			// not "name#key" — the per-row suffix is dropped on this path in
			// the original engine too (ResultData.suffix defaults to ""),
			// ported exactly rather than "fixed".
			results = append(results, Result{Name: t.Name, Status: Skip})
		default:
			if r.subKeyMatches(t.Name, row.key) {
				results = append(results, r.invoke(t.Name, t.Name, row.key, row.args, testType))
			}
			if t.After != "" {
				r.invokeHook(t.After, "after test function for the test", false)
			}
		}
	}

	for _, g := range t.Groups {
		r.groupExecuted[g]++
	}
	r.runAfterEachHooks()
	r.runAfterGroupHooks(t)

	return results
}

// invoke calls the function named fnName (testName for a general test, the
// data provider's underlying test function for a row) and reports the
// outcome under testName[#suffix] — distinct from fnName so a data-driven
// row can be invoked via its test function's real name while reporting as
// "name#key". A failure also records a TestFailureNotice ConsoleEvent at the
// exact point it happens, matching handleTestFuncOutput/
// handleDataDrivenTestOutput's inline "has failed" println.
func (r *runner) invoke(fnName, testName, suffix string, args []values.BalValue, testType TestType) (result Result) {
	resultName := testName
	if suffix != "" {
		resultName = testName + "#" + suffix
	}
	result = Result{Name: resultName, Status: Pass, Type: testType}
	defer func() {
		if rec := recover(); rec != nil {
			result.Status = Fail
			result.Message = panicMessage(rec)
		}
		if result.Status == Fail {
			r.events = append(r.events, ConsoleEvent{TestFailure: &TestFailureNotice{Name: testName, Suffix: suffix, HasSuffix: true}})
		}
	}()
	fn, ok := runtime.LookupFunction(r.rt, r.org, r.module, fnName)
	if !ok {
		result.Status = Fail
		result.Message = "function " + fnName + " not found"
		return result
	}
	if _, err := runtime.InvokeFunction(r.rt, fn, args); err != nil {
		result.Status = Fail
		result.Message = err.Error()
	}
	return result
}

// runSuiteHooks runs every hook in names via invokeHook, unconditionally —
// BeforeSuite only ever runs once, with nothing to skip yet. onFailure is
// called once if any hook failed (the first failure stops the loop, the
// same as `check executeFunction(...)` propagating out of
// execute.bal#executeFunctions).
func (r *runner) runSuiteHooks(hooks []testplan.SuiteHook, label string, onFailure func(bool)) {
	for _, h := range hooks {
		if r.invokeHook(h.Name, label, true) {
			onFailure(true)
			return
		}
	}
}

func (r *runner) runAfterSuiteHooks() {
	for _, h := range r.plan.AfterSuite {
		if r.shouldAfterSuiteSkip && !h.AlwaysRun {
			continue
		}
		r.invokeHook(h.Name, "after test suite function", true)
	}
}

func (r *runner) runBeforeEachHooks() {
	if r.shouldSkip {
		return
	}
	for _, h := range r.plan.BeforeEach {
		if r.invokeHook(h.Name, "before each function for the test", true) {
			r.shouldSkip = true
			return
		}
	}
}

func (r *runner) runAfterEachHooks() {
	if r.shouldSkip {
		return
	}
	for _, h := range r.plan.AfterEach {
		if r.invokeHook(h.Name, "after each test function for the test", true) {
			r.shouldSkip = true
			return
		}
	}
}

// runBeforeGroupHooks fires each of t's groups' BeforeGroups hooks exactly
// once — the first time any test in that group is processed, matching
// GroupStatusRegistry.firstExecuted (executedTests[group] > 0). A failure
// skips only t itself (not the rest of the group) and marks the group so
// its AfterGroups hooks skip later too — ported exactly, including this
// asymmetry, from handleBeforeGroupOutput.
func (r *runner) runBeforeGroupHooks(t testplan.TestFunction) {
	for _, g := range t.Groups {
		if r.groupExecuted[g] > 0 {
			continue
		}
		hooks := groupHooksFor(r.plan.BeforeGroups, g)
		if len(hooks) == 0 {
			continue
		}
		if r.shouldSkip {
			continue
		}
		for _, h := range hooks {
			if r.invokeHook(h.Name, "before groups function for the test", true) {
				r.perTestSkip[t.Name] = true
				r.groupSkipAfter[g] = true
				break
			}
		}
	}
}

// runAfterGroupHooks fires each of t's groups' AfterGroups hooks exactly
// once — when t is the last test processed in that group, matching
// GroupStatusRegistry.lastExecuted (executedTests[group] == groupTotal).
func (r *runner) runAfterGroupHooks(t testplan.TestFunction) {
	for _, g := range t.Groups {
		if r.groupExecuted[g] != r.groupTotal[g] {
			continue
		}
		hooks := groupHooksFor(r.plan.AfterGroups, g)
		if len(hooks) == 0 {
			continue
		}
		skip := r.shouldSkip || r.groupSkipAfter[g]
		for _, h := range hooks {
			if skip && !h.AlwaysRun {
				continue
			}
			r.invokeHook(h.Name, "after test group function for the test", true)
		}
	}
}

func groupHooksFor(hooks []testplan.GroupHook, group string) []testplan.GroupHook {
	var matched []testplan.GroupHook
	for _, h := range hooks {
		for _, g := range h.Groups {
			if g == group {
				matched = append(matched, h)
				break
			}
		}
	}
	return matched
}

// invokeHook calls a hook function by name and records a HookFailure (plus a
// matching ConsoleEvent, at the exact point of failure — see
// execute.bal#printExecutionError) if it panics, returning true on failure
// so callers can apply their own skip/exit-code semantics without
// duplicating the invoke+recover dance.
//
// includeName is false only for a per-test before:/after: hook — ported from
// executeFunction's own two call shapes: a suite/each/group hook is invoked
// via executeFunction(testFunction), which sets invokedName =
// testFunction.name, but a before:/after: hook is invoked via
// executeFunction(<function>testFunction.after) — a bare `function` value
// with no TestFunction attached, so invokedName stays "" there. The
// resulting HookFailure.Message is exactly what printExecutionError expects
// to split on its first newline (""/name, then the message) either way.
func (r *runner) invokeHook(name, label string, includeName bool) (failed bool) {
	prefix := "\n"
	if includeName {
		prefix = name + "\n"
	}
	defer func() {
		if rec := recover(); rec != nil {
			failed = true
			r.recordHookFailure(HookFailure{Hook: label, Message: prefix + panicMessage(rec)})
		}
	}()
	fn, ok := runtime.LookupFunction(r.rt, r.org, r.module, name)
	if !ok {
		r.recordHookFailure(HookFailure{Hook: label, Message: prefix + "function " + name + " not found"})
		return true
	}
	if _, err := runtime.InvokeFunction(r.rt, fn, nil); err != nil {
		r.recordHookFailure(HookFailure{Hook: label, Message: prefix + err.Error()})
		return true
	}
	return false
}

func (r *runner) recordHookFailure(hf HookFailure) {
	r.hookFailures = append(r.hookFailures, hf)
	r.events = append(r.events, ConsoleEvent{Hook: &hf})
}
