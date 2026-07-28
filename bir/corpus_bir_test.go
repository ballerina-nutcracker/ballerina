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

package bir_test

import (
	"flag"
	"os"
	"testing"

	"ballerina/bir"
	"ballerina/context"
	"ballerina/semtypes"
	"ballerina/test_util"
	"ballerina/test_util/testphases"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var supportedSubsets = []string{"subset1"}

var update = flag.Bool("update", false, "update expected BIR text files")

// readExpectedBIRText reads the expected BIR text file and returns its content.
// Returns the content and an error. If the file doesn't exist, the error will be os.ErrNotExist.
func readExpectedBIRText(filePath string) (string, error) {
	expectedTextBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(expectedTextBytes), nil
}

// getBIRDiff generates a detailed diff string showing differences between expected and actual BIR text.
func getBIRDiff(expectedText, actualText string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expectedText, actualText, false)
	return dmp.DiffPrettyText(diffs)
}

// birGenerationSkipList is the BIR-stage *additional* skip list, on top of
// the shared test_util.UnsupportedTests baseline.
var birGenerationSkipList = []string{
	// https://github.com/ballerina-nutcracker/ballerina/issues/417
	"subset8/08-xml/namespace12-v.bal",
}

func TestBIRGeneration(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetValidAndPanicTests(t, test_util.BIR)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testBIRGeneration(t, testPair)
		})
	}
}

// testBIRGeneration tests BIR generation for a single .bal file.
func testBIRGeneration(t *testing.T, testPair test_util.TestCase) {
	if test_util.IsUnsupported(testPair.InputPath) || test_util.MatchesSkip(testPair.InputPath, birGenerationSkipList) {
		t.Skipf("Skipping BIR generation test for %s", testPair.InputPath)
		return
	}

	// Catch panics during BIR generation
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic while generating BIR from %s: %v", testPair.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Errorf("loading lang libraries failed for %s: %v", testPair.InputPath, err)
		return
	}
	result, err := testphases.RunPipeline(env, cx, langlibs, testphases.PhaseBIR, testPair.InputPath)
	if err != nil {
		t.Errorf("pipeline failed for %s: %v", testPair.InputPath, err)
		return
	}

	if result.BIRPackage == nil {
		t.Errorf("BIR package is nil for %s", testPair.InputPath)
		return
	}

	verifyBIRPositions(t, result.BIRPackage, testPair.InputPath)

	// Pretty print BIR output
	prettyPrinter := bir.PrettyPrinter{}
	actualBIR := prettyPrinter.Print(semtypes.ContextFrom(env.GetTypeEnv()), *result.BIRPackage)

	// If update flag is set, update expected file
	if *update {
		if test_util.UpdateIfNeeded(t, testPair.ExpectedPath, actualBIR) {
			t.Fatalf("Updated expected BIR file: %s", testPair.ExpectedPath)
		}
		return
	}

	// Read expected BIR text file
	expectedText := test_util.ReadExpectedFile(t, testPair.ExpectedPath)

	// Compare BIR text strings exactly
	if actualBIR != expectedText {
		diff := getBIRDiff(expectedText, actualBIR)
		t.Errorf("BIR text mismatch for %s\nExpected file: %s\n%s", testPair.InputPath, testPair.ExpectedPath, diff)
		return
	}
}

func verifyBIRPositions(t *testing.T, pkg *bir.BIRPackage, inputPath string) {
	for _, fn := range pkg.Functions {
		verifyFunctionPositions(t, &fn, inputPath)
	}
	if pkg.MainFunction != nil {
		verifyFunctionPositions(t, pkg.MainFunction, inputPath)
	}
}

func verifyFunctionPositions(t *testing.T, fn *bir.BIRFunction, inputPath string) {
	for _, bb := range fn.BasicBlocks {
		for _, inst := range bb.Instructions {
			if bir.IsLocationEmpty(inst.GetPos()) {
				t.Errorf("instruction %T in %s (BB: %s) has no position info for %s", inst, fn.Name, bb.Id, inputPath)
			}
		}
		if bb.Terminator != nil && bir.IsLocationEmpty(bb.Terminator.GetPos()) {
			t.Errorf("terminator %T in %s (BB: %s) has no position info for %s", bb.Terminator, fn.Name, bb.Id, inputPath)
		}
	}
}
