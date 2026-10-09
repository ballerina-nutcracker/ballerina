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

package symbols_test

import (
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/testphases"
)

// TestSymbolStageInvariants checks properties that must hold for every corpus file
// reaching the symbol stage, which a per-file corpus golden cannot express:
//   - the symbol stage never panics
//   - a pipeline failure is always accompanied by a diagnostic
//   - the symbol stage never reports an INTERNAL_ERROR diagnostic
func TestSymbolStageInvariants(t *testing.T) {
	t.Parallel()
	testCases := test_util.GetTests(t, test_util.AST, func(path string) bool {
		if test_util.IsFutureTest(path) {
			return false
		}
		return strings.HasSuffix(path, "-v.bal") || strings.HasSuffix(path, "-p.bal") || strings.HasSuffix(path, "-e.bal")
	})
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			testSymbolStageInvariants(t, testCase)
		})
	}
}

func testSymbolStageInvariants(t *testing.T, testCase test_util.TestCase) {
	if test_util.IsUnsupported(testCase.InputPath) {
		t.Skipf("Skipping symbol stage invariant test for %s", testCase.InputPath)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("symbol stage panicked for %s: %v", testCase.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Fatalf("loading lang libraries failed for %s: %v", testCase.InputPath, err)
	}
	if _, err := testphases.RunPipeline(env, cx, langlibs, testphases.PhaseSymbolResolution, testCase.InputPath); err != nil {
		if !cx.HasDiagnostics() {
			t.Fatalf("pipeline failed for %s: %v", testCase.InputPath, err)
		}
		t.Skipf("pipeline stopped before the symbol stage for %s: %v", testCase.InputPath, err)
	}
	for _, d := range cx.Diagnostics() {
		if d.DiagnosticInfo().Code() == "INTERNAL_ERROR" {
			t.Errorf("INTERNAL_ERROR at the symbol stage for %s: %s", testCase.InputPath, d.Message())
		}
	}
}
