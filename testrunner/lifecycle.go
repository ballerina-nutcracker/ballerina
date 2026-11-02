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
	"fmt"
	"io"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/testplan"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// HookFailure records a suite/group/each hook that panicked, tracked
// separately from a test Result's own pass/fail/skip outcome.
type HookFailure struct {
	Hook    string // e.g. "before test suite function"
	Message string
}

// RunReport is one full Run's outcome: every test Result and every
// HookFailure. The live console notices (a test or hook "has failed" as
// execution reaches it) are written directly to Run's w as they happen, not
// collected here — see w's doc comment on Run.
type RunReport struct {
	Results      []Result
	HookFailures []HookFailure
}

// ConsoleEvent is one live console notice in the order Run produced it —
// exactly one field set, since a single executeOne can interleave a test
// failure and a hook failure that can't be reconstructed later from
// Results/HookFailures alone.
type ConsoleEvent struct {
	Hook        *HookFailure
	TestFailure *TestFailureNotice
}

// TestFailureNotice is one "has failed" console line: HasSuffix is true for
// a general test or a data-driven row (suffix = key), false only when the
// data provider itself fails before any row exists.
type TestFailureNotice struct {
	Name      string
	Suffix    string
	HasSuffix bool
}

// Run executes a suite (BeforeSuite, every included test in graph.Order with
// its full hook/data-provider lifecycle, then AfterSuite). See this
// package's doc comment (run.go) for what's not yet covered.
//
// w receives each "has failed" console notice (test or hook) the instant it
// happens, interleaved with that test/hook's own output (which the invoked
// Ballerina function writes directly, e.g. via io:println) exactly as
// jballerina does — see execute.bal's inline println calls. A test or hook's
// own output isn't itself routed through w; only the notice is.
func Run(w io.Writer, rt *runtime.Runtime, org, module string, plan *testplan.TestPlan, graph *testplan.Graph, included map[string]bool, subKeyFilters SubKeyFilters) RunReport {
	r := &runner{
		w: w, rt: rt, org: org, module: module, plan: plan, graph: graph, included: included,
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
	w        io.Writer
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
}

func indexByName(tests []testplan.TestFunction) map[string]testplan.TestFunction {
	m := make(map[string]testplan.TestFunction, len(tests))
	for _, t := range tests {
		m[t.Name] = t
	}
	return m
}

// computeGroupTotals counts, per group, how many included and enabled tests
// belong to it.
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

	return RunReport{Results: results, HookFailures: r.hookFailures}
}

// executeOne dispatches to the data-driven or plain-test path based on
// whether t has a data provider.
func (r *runner) executeOne(t testplan.TestFunction) []Result {
	if t.DataProvider != "" {
		return r.executeDataDriven(t)
	}
	return []Result{r.executeSingle(t)}
}

// executeSingle runs one plain test through its group/each/before/after
// hooks. A failing before: hook Skips the test without invoking it; a
// failing after: hook doesn't change the test's own Result. Either cascades.
//
// An evaluation test (t.IsEvaluation()) is dispatched to executeEvaluation
// instead, which manages its own per-iteration each/before/after hooks
// internally (once per t.Runs, not once overall) — so BeforeEach/AfterEach
// here must not also wrap it, unlike the plain-test path below.
func (r *runner) executeSingle(t testplan.TestFunction) Result {
	r.runBeforeGroupHooks(t)

	var result Result
	var shouldSkipDependents bool
	if t.IsEvaluation() {
		result, shouldSkipDependents = r.executeEvaluation(t)
	} else {
		r.runBeforeEachHooks()
		skip := r.perTestSkip[t.Name] || r.shouldSkip
		shouldSkipDependents = skip
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
		r.runAfterEachHooks()
	}

	for _, g := range t.Groups {
		r.groupExecuted[g]++
	}
	r.runAfterGroupHooks(t)

	if shouldSkipDependents {
		for _, dep := range r.graph.Dependents[t.Name] {
			r.perTestSkip[dep] = true
		}
	}
	return result
}

// executeDataDriven runs a data-driven test: the provider is invoked once
// up front (a failure skips its whole lifecycle, no dependent cascade ever).
// Unlike BeforeEach/AfterEach, before:/after: run once per row. A row not
// matching a --tests sub-key filter gets no Result, but its hooks still run.
//
// An evaluation test (t.IsEvaluation()) is dispatched to
// executeDataDrivenEvaluation instead, which manages its own per-iteration
// each/before/after hooks internally (once per t.Runs, each iteration
// re-running every row) — so BeforeEach/AfterEach here must not also wrap
// it, unlike the plain-test path below. Sub-key filtering doesn't apply to
// an evaluation: jballerina's executeDataDrivenEvaluation has no such
// concept, and it exists specifically to narrow --rerun-failed re-runs of a
// single failed row, which doesn't make sense for a test whose failure is
// already an average across every row and every run.
func (r *runner) executeDataDriven(t testplan.TestFunction) []Result {
	rows, testType, err := r.dataProviderRows(t.DataProvider)
	if err != nil {
		writeConsoleEvent(r.w, ConsoleEvent{TestFailure: &TestFailureNotice{Name: t.Name, HasSuffix: false}})
		return []Result{{Name: t.Name, Status: Fail, Message: DataProviderFailedMessage, RawMessage: true}}
	}

	r.runBeforeGroupHooks(t)

	var results []Result
	if t.IsEvaluation() {
		results = r.executeDataDrivenEvaluation(t, rows)
	} else {
		r.runBeforeEachHooks()
		skip := r.perTestSkip[t.Name] || r.shouldSkip
		for _, row := range rows {
			name := t.Name + "#" + row.key
			switch {
			case skip:
				results = append(results, Result{Name: name, Status: Skip})
			case t.Before != "" && r.invokeHook(t.Before, "before test function for the test", false):
				// A before: failure reports this row under the test's bare
				// name, not "name#key" — matches jballerina, not a bug.
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
		r.runAfterEachHooks()
	}

	for _, g := range t.Groups {
		r.groupExecuted[g]++
	}
	r.runAfterGroupHooks(t)

	return results
}

// invoke calls fnName and reports the outcome under testName[#suffix] —
// distinct from fnName so a data-driven row reports as "name#key" while
// invoking its real function name. A failure also records a console event.
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
			writeConsoleEvent(r.w, ConsoleEvent{TestFailure: &TestFailureNotice{Name: testName, Suffix: suffix, HasSuffix: true}})
		}
	}()
	fn, ok := runtime.LookupFunction(r.rt, r.org, r.module, fnName)
	if !ok {
		result.Status = Fail
		result.Message = "function " + fnName + " not found"
		return result
	}
	retVal, err := runtime.InvokeFunction(r.rt, fn, args)
	if err != nil {
		result.Status = Fail
		result.Message = err.Error()
	} else if errVal, ok := retVal.(*values.Error); ok {
		result.Status = Fail
		result.Message = panicMessage(errVal)
	}
	return result
}

// executeEvaluation runs t.Runs iterations of a non-data-driven evaluation
// test (t.IsEvaluation(), t.DataProvider == ""), averaging pass/fail across
// runs against t.MinPassRate — ported from jballerina's
// executeNonDataDrivenEvaluation (concurrentExecuter.bal's isolated twin is
// the same logic, collapsed into one path since this port has no
// isolated/non-isolated execution distinction).
//
// Unlike a plain test, an evaluation's outcome is never printed as a live
// "has failed" console notice — only in WriteConsoleReport's batched
// pass/fail sections — exactly as jballerina's executeNonDataDrivenEvaluation
// only ever calls reportData.onPassed/onFailed, never handleTestFuncOutput's
// live println.
func (r *runner) executeEvaluation(t testplan.TestFunction) (result Result, shouldSkipDependents bool) {
	passed := 0
	anyAfterFailed := false
	var runResults []EvalRun
	for run := 1; run <= t.Runs; run++ {
		r.runBeforeEachHooks()
		if skip := r.perTestSkip[t.Name] || r.shouldSkip; skip {
			r.runAfterEachHooks()
			return Result{Name: t.Name, Status: Skip}, true
		}
		if t.Before != "" && r.invokeHook(t.Before, "before test function for the test", false) {
			// A before: failure aborts the whole evaluation immediately,
			// not just this one run — matches jballerina's
			// executeNonDataDrivenEvaluation, which returns rather than
			// continuing to the next iteration.
			r.runAfterEachHooks()
			return Result{Name: t.Name, Status: Skip}, true
		}

		ok, errMsg, hardErr := r.invokeEvaluationRun(t.Name, nil)
		if hardErr != nil {
			r.runAfterEachHooks()
			return Result{Name: t.Name, Status: Fail, Message: hardErr.Error(), RawMessage: true, Type: EvalTest}, true
		}
		if ok {
			passed++
		}
		runResults = append(runResults, EvalRun{ID: run, ErrorMessage: errMsg})

		if t.After != "" && r.invokeHook(t.After, "after test function for the test", false) {
			anyAfterFailed = true
		}
		r.runAfterEachHooks()
	}

	result = evaluationResult(t, float64(passed)/float64(t.Runs), runResults)
	return result, result.Status == Fail || anyAfterFailed
}

// executeDataDrivenEvaluation runs t.Runs iterations of a data-driven
// evaluation test, each iteration re-running every one of rows (fetched
// once, up front, same as a plain data-driven test) and computing that
// iteration's own passedEntries/totalEntries; the final score is the
// average of those per-iteration rates across all Runs — ported from
// jballerina's executeDataDrivenEvaluation.
//
// Unlike executeEvaluation, an after: hook failure here never cascades a
// skip to dependents — matching an asymmetry already present in
// jballerina's own source (executeAfterFunctionIsolated's result is
// discarded here, unlike executeNonDataDrivenEvaluation's
// afterFunctionResults tracking) — which costs nothing extra to preserve
// given this port's data-driven tests never cascade skip to dependents at
// all (see executeDataDriven's doc comment).
func (r *runner) executeDataDrivenEvaluation(t testplan.TestFunction, rows []dataRow) []Result {
	if len(rows) == 0 {
		return []Result{{Name: t.Name, Status: Fail, Message: "the data provider returned no data", RawMessage: true, Type: EvalTest}}
	}

	var cumulativePassRate float64
	var runResults []EvalRun
	for iteration := 1; iteration <= t.Runs; iteration++ {
		r.runBeforeEachHooks()
		if skip := r.perTestSkip[t.Name] || r.shouldSkip; skip {
			r.runAfterEachHooks()
			return []Result{{Name: t.Name, Status: Skip}}
		}
		if t.Before != "" && r.invokeHook(t.Before, "before test function for the test", false) {
			r.runAfterEachHooks()
			return []Result{{Name: t.Name, Status: Skip}}
		}

		passedEntries := 0
		outcomes := make([]EvalOutcome, 0, len(rows))
		for _, row := range rows {
			ok, errMsg, hardErr := r.invokeEvaluationRun(t.Name, row.args)
			if hardErr != nil {
				r.runAfterEachHooks()
				return []Result{{Name: t.Name, Status: Fail, Message: hardErr.Error(), RawMessage: true, Type: EvalTest}}
			}
			if ok {
				passedEntries++
			}
			outcomes = append(outcomes, EvalOutcome{ID: row.key, ErrorMessage: errMsg})
		}

		if t.After != "" {
			r.invokeHook(t.After, "after test function for the test", false)
		}
		r.runAfterEachHooks()

		passRate := float64(passedEntries) / float64(len(rows))
		cumulativePassRate += passRate
		runResults = append(runResults, EvalRun{ID: iteration, Outcomes: outcomes, PassRate: passRate})
	}

	return []Result{evaluationResult(t, cumulativePassRate/float64(t.Runs), runResults)}
}

// evaluationResult builds the final Pass/Fail Result for an evaluation
// test given its observed pass rate — shared by executeEvaluation and
// executeDataDrivenEvaluation, ported from jballerina's "evaluation
// passed/failed with an average pass rate of X" message (serialExecuter.bal
// #executeNonDataDrivenEvaluation/#executeDataDrivenEvaluation).
func evaluationResult(t testplan.TestFunction, observedPassRate float64, runs []EvalRun) Result {
	status, verb := Fail, "failed"
	if observedPassRate >= t.MinPassRate {
		status, verb = Pass, "passed"
	}
	return Result{
		Name:       t.Name,
		Status:     status,
		Message:    fmt.Sprintf("evaluation %s with an average pass rate of %s", verb, values.FormatFloat(observedPassRate)),
		RawMessage: true,
		Type:       EvalTest,
		Eval: &EvalSummary{
			TargetPassRate:   t.MinPassRate,
			ObservedPassRate: observedPassRate,
			RunResults:       runs,
		},
	}
}

// invokeEvaluationRun calls fnName once for one run of an evaluation test.
// passed is true if the call completed with no error. A *values.Error
// (assertion failure, e.g. test:assertEquals) reports passed=false with its
// message — one run simply "didn't pass," counted toward the pass rate but
// not a hard failure. Any other error (bad args, a panic that isn't a
// *values.Error) is hard: it aborts the whole evaluation immediately rather
// than being averaged in — jballerina's InvalidArgumentError distinction,
// the same panic-vs-*values.Error convention this package's own invoke
// already relies on.
func (r *runner) invokeEvaluationRun(fnName string, args []values.BalValue) (passed bool, errMsg string, hardErr error) {
	defer func() {
		if rec := recover(); rec != nil {
			if balErr, ok := rec.(*values.Error); ok {
				errMsg = panicMessage(balErr)
				return
			}
			hardErr = fmt.Errorf("%v", rec)
		}
	}()
	fn, ok := runtime.LookupFunction(r.rt, r.org, r.module, fnName)
	if !ok {
		return false, "", fmt.Errorf("function %s not found", fnName)
	}
	retVal, err := runtime.InvokeFunction(r.rt, fn, args)
	if err != nil {
		return false, "", err
	}
	if errVal, ok := retVal.(*values.Error); ok {
		return false, panicMessage(errVal), nil
	}
	return true, "", nil
}

// runSuiteHooks runs every hook unconditionally, stopping at the first
// failure and calling onFailure once.
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

// runBeforeGroupHooks fires each of t's groups' BeforeGroups hooks once, the
// first time any test in that group runs. A failure skips only t itself but
// marks the group so its AfterGroups hooks skip later too.
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

// runAfterGroupHooks fires each of t's groups' AfterGroups hooks once, when
// t is the last test processed in that group.
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

// invokeHook calls a hook by name, recording a HookFailure and ConsoleEvent
// on panic and returning true so callers apply their own skip semantics.
// includeName is false only for a per-test before:/after: hook, which has
// no name of its own to report.
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
	retVal, err := runtime.InvokeFunction(r.rt, fn, nil)
	if err != nil {
		r.recordHookFailure(HookFailure{Hook: label, Message: prefix + err.Error()})
		return true
	}
	if errVal, ok := retVal.(*values.Error); ok {
		r.recordHookFailure(HookFailure{Hook: label, Message: prefix + panicMessage(errVal)})
		return true
	}
	return false
}

func (r *runner) recordHookFailure(hf HookFailure) {
	r.hookFailures = append(r.hookFailures, hf)
	writeConsoleEvent(r.w, ConsoleEvent{Hook: &hf})
}
