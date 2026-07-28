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

package semantics_test

import (
	"flag"
	"testing"

	"ballerina/context"
	"ballerina/semantics"
	"ballerina/semtypes"
	"ballerina/test_util"
	"ballerina/test_util/testphases"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var updateCFG = flag.Bool("update", false, "update expected CFG text files")

// cfgGenerationSkipList is the CFG-stage *additional* skip list, on top of
// the shared test_util.UnsupportedTests baseline.
var cfgGenerationSkipList = []string{
	// https://github.com/ballerina-nutcracker/ballerina/issues/417
	"subset8/08-xml/namespace12-v.bal",
}

func TestCFGGeneration(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetValidAndPanicTests(t, test_util.CFG)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testCFGGeneration(t, testPair)
		})
	}
}

// testCFGGeneration tests CFG generation for a single .bal file.
func testCFGGeneration(t *testing.T, testPair test_util.TestCase) {
	if test_util.IsUnsupported(testPair.InputPath) || test_util.MatchesSkip(testPair.InputPath, cfgGenerationSkipList) {
		t.Skipf("Skipping CFG generation test for %s", testPair.InputPath)
		return
	}

	// Catch panics during CFG generation
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic while generating CFG from %s: %v", testPair.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Errorf("loading lang libraries failed for %s: %v", testPair.InputPath, err)
		return
	}
	result, err := testphases.RunPipeline(env, cx, langlibs, testphases.PhaseCFG, testPair.InputPath)
	if err != nil {
		t.Errorf("pipeline failed for %s: %v", testPair.InputPath, err)
		return
	}

	cfg := result.CFG
	if cfg == nil {
		t.Errorf("CFG is nil for %s", testPair.InputPath)
		return
	}

	// Validate backedgeParents is a subset of parents for every block
	for _, err := range cfg.ValidateInvariants() {
		t.Errorf("CFG invariant violated in %s: function %v, block %d: backedgeParent %d is not in parents %v",
			testPair.InputPath, err.FuncRef, err.BlockID, err.BackedgeParent, err.Parents)
	}

	// Pretty print CFG output
	prettyPrinter := semantics.NewCFGPrettyPrinter(cx)
	actualCFG := prettyPrinter.Print(cfg)

	// If update flag is set, update expected file
	if *updateCFG {
		if test_util.UpdateIfNeeded(t, testPair.ExpectedPath, actualCFG) {
			t.Fatalf("Updated expected CFG file: %s", testPair.ExpectedPath)
		}
		return
	}

	// Read expected CFG text file
	expectedText := test_util.ReadExpectedFile(t, testPair.ExpectedPath)

	// Compare CFG text strings exactly
	if actualCFG != expectedText {
		diff := getCFGDiff(expectedText, actualCFG)
		t.Errorf("CFG text mismatch for %s\nExpected file: %s\n%s", testPair.InputPath, testPair.ExpectedPath, diff)
		return
	}
}

// getCFGDiff generates a detailed diff string showing differences between expected and actual CFG text.
func getCFGDiff(expectedText, actualText string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expectedText, actualText, false)
	return dmp.DiffPrettyText(diffs)
}
