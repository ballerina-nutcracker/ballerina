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

// Package testrunner invokes @test:Config functions directly — no
// registration, no Ballerina-level orchestration engine — matching the
// original design requirement: "the test runtime does not have a main."
//
// Run (lifecycle.go) is a Go port of execute.bal/serialExecuter.bal's
// behavior: BeforeSuite/AfterSuite/BeforeEach/AfterEach/BeforeGroups/
// AfterGroups hooks, skip-cascade from a failed or skipped dependency, in
// the order testplan.BuildGraph establishes, plus data providers
// (dataprovider.go): map- and tuple-array-shaped, reported as "name#key"
// rows, with a data provider's own failure short-circuiting before any hook
// runs and never skip-cascading to dependents. Per-test before:/after: hooks
// (TestFunction.Before/After) are also covered: once per test for a plain
// test (a before: failure skips the test without invoking it; an after:
// failure still cascades skip to dependents even though the test itself may
// have passed), once per row for a data-driven test. Data-driven sub-key
// filtering (--tests func#key, subkey.go's SubKeyFilters) narrows which rows
// of an already-included data-driven test actually execute — a non-matching
// row is dropped with no Result at all, but its before:/after: hooks still
// run, and any transitively enabled dependent bypasses the restriction
// entirely.
package testrunner

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/values"
)

// Status is a test's outcome.
type Status int

const (
	Pass Status = iota
	Fail
	Skip
)

func (s Status) String() string {
	switch s {
	case Pass:
		return "pass"
	case Fail:
		return "fail"
	case Skip:
		return "skip"
	default:
		return "unknown"
	}
}

// Result is one test's outcome.
type Result struct {
	Name    string
	Status  Status
	Message string   // non-empty only when Status == Fail
	Type    TestType // GeneralTest unless this Result came from a data row or is an EvalTest

	// RawMessage is true when Message is printed as is, without the trailing
	// "\n" WriteConsoleReport adds to an ordinary failure message.
	RawMessage bool

	// Eval is non-nil only when Type == EvalTest — the per-run breakdown
	// behind Message's "evaluation passed/failed with an average pass rate
	// of X" summary.
	Eval *EvalSummary
}

// TestType distinguishes a general test from a data-driven row or an
// evaluation test. It decides a failed row's rerun-JSON sub-key quoting
// (statusreport.go), the one place it affects this port's output.
type TestType int

const (
	GeneralTest TestType = iota
	DataDrivenMapOfTuple
	DataDrivenTupleOfTuple
	EvalTest
)

// EvalSummary is an evaluation test's (TestFunction.IsEvaluation()) outcome
// across all of its Runs. ObservedPassRate is passedIterations/Runs for a
// non-data-driven evaluation, or the average of each run's own
// passedEntries/totalEntries for a data-driven one.
type EvalSummary struct {
	TargetPassRate   float64
	ObservedPassRate float64
	RunResults       []EvalRun
}

// EvalRun is one run (1..Runs) of an evaluation test. Outcomes is nil for a
// non-data-driven evaluation (ErrorMessage alone describes that run's single
// invocation), populated for a data-driven one (one entry per data-provider
// row that ran, PassRate that row set's own passedEntries/totalEntries).
type EvalRun struct {
	ID           int
	ErrorMessage string // non-data-driven only; "" if that run passed
	Outcomes     []EvalOutcome
	PassRate     float64 // data-driven only
}

// EvalOutcome is one data-provider row's result within one EvalRun of a
// data-driven evaluation test.
type EvalOutcome struct {
	ID           string
	ErrorMessage string // "" if that row passed
}

// panicMessage returns a *values.Error's message (what a failed assertion
// panics with) and formats any other value with %v.
func panicMessage(r any) string {
	if balErr, ok := r.(*values.Error); ok {
		return balErr.Message
	}
	return fmt.Sprintf("%v", r)
}
