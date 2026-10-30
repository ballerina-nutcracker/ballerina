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
//
// A note on the `*.bal` references throughout this package's comments
// (execute.bal, serialExecuter.bal, register.bal, report.bal, filter.bal,
// registration_api.bal): those files were the Ballerina orchestration
// engine this package replaced, and they no longer exist in this repo —
// they were deleted once this package took over (see the P8 implementation
// plan's Phase 3.4b). The references are kept deliberately, because they
// record *why* several non-obvious behaviors here are shaped the way they
// are; read them against git history or jballerina upstream, not against
// the current tree.
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
	Type    TestType // GeneralTest unless this Result came from a data row

	// RawMessage is true if Message is already a complete, standalone error
	// string that must NOT get WriteConsoleReport's synthetic trailing-"\n"
	// treatment (see its own doc comment on why that "+\n" exists at all).
	// Every ordinary test/row failure's Message comes from getErrorMessage's
	// convention (err.message() + "\n") somewhere upstream in the original
	// engine, which this port deliberately doesn't bake into Message itself
	// — RawMessage is the one documented exception: a data provider's own
	// invocation failure is reported via a bare `error("Failed to execute
	// the data provider")` (registerTestConfig), never passed through
	// getErrorMessage at all, so it must be formatted without that trick.
	RawMessage bool
}

// TestType mirrors types.bal's TestType enum — report.bal#getTestType reads
// this same distinction to decide a failed row's rerun-JSON sub-key
// quoting (statusreport.go), the one place it affects this port's output.
type TestType int

const (
	GeneralTest TestType = iota
	DataDrivenMapOfTuple
	DataDrivenTupleOfTuple
)

// panicMessage extracts a readable message from whatever runtime.InvokeFunction
// panics with. A failed assertion panics with *values.Error (confirmed by
// experiment, not assumed — see the implementation plan's Phase 3 notes);
// its Message field holds exactly the text assert.bal constructs. Any other
// panic shape is formatted generically, the same way a Ballerina `trap`
// catches any panic, not just assertion-shaped ones.
func panicMessage(r any) string {
	if balErr, ok := r.(*values.Error); ok {
		return balErr.Message
	}
	return fmt.Sprintf("%v", r)
}
