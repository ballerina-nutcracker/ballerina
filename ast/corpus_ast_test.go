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

package ast_test

import (
	"flag"
	"fmt"
	"testing"

	"ballerina/ast"
	"ballerina/context"
	"ballerina/semtypes"
	"ballerina/test_util"
	"ballerina/test_util/testphases"
	"ballerina/tools/diagnostics"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// astGenerationSkipList is the AST-stage *additional* skip list, on top of the
// shared test_util.UnsupportedTests baseline.
var astGenerationSkipList = []string{}

func TestASTGeneration(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetValidAndPanicTests(t, test_util.AST)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testASTGeneration(t, testPair)
		})
	}
}

func testASTGeneration(t *testing.T, testCase test_util.TestCase) {
	if test_util.IsUnsupported(testCase.InputPath) || test_util.MatchesSkip(testCase.InputPath, astGenerationSkipList) {
		t.Skipf("Skipping AST generation test for %s", testCase.InputPath)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic while testing AST generation for %s: %v", testCase.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	result, err := testphases.RunPipeline(env, cx, nil, testphases.PhaseAST, testCase.InputPath)
	if err != nil {
		t.Errorf("pipeline failed for %s: %v", testCase.InputPath, err)
		return
	}
	prettyPrinter := ast.PrettyPrinter{}
	actualAST := prettyPrinter.Print(result.CompilationUnit)

	// If update flag is set, update expected file
	if *update {
		if test_util.UpdateIfNeeded(t, testCase.ExpectedPath, actualAST) {
			t.Errorf("updated expected AST file: %s", testCase.ExpectedPath)
		}
		return
	}

	// Read expected AST file
	expectedAST := test_util.ReadExpectedFile(t, testCase.ExpectedPath)

	// Compare AST strings exactly
	if actualAST != expectedAST {
		diff := getDiff(expectedAST, actualAST)
		t.Errorf("AST mismatch for %s\nExpected file: %s\n%s", testCase.InputPath, testCase.ExpectedPath, diff)
		return
	}
}

var update = flag.Bool("update", false, "update expected AST files")

// getDiff generates a detailed diff string showing differences between expected and actual AST strings.
func getDiff(expectedAST, actualAST string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expectedAST, actualAST, false)
	return dmp.DiffPrettyText(diffs)
}

// walkTestVisitor tracks node types visited during Walk traversal
type walkTestVisitor struct {
	t            *testing.T
	visitedTypes map[string]int
	nodeCount    int
}

func (v *walkTestVisitor) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	v.nodeCount++
	typeName := fmt.Sprintf("%T", node)
	v.visitedTypes[typeName]++

	if diagnostics.IsLocationEmpty(node.GetPosition()) {
		v.t.Errorf("node with missing position: %T", node)
	}

	return v
}

func (v *walkTestVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return v
}

// walkTraversalSkipList is the walk-traversal *additional* skip list, on top
// of the shared test_util.UnsupportedTests baseline. Currently empty -- every
// known failure is already covered by the shared baseline.
var walkTraversalSkipList = []string{}

func TestWalkTraversal(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetValidAndPanicTests(t, test_util.AST)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testWalkTraversal(t, testPair)
		})
	}
}

func testWalkTraversal(t *testing.T, testCase test_util.TestCase) {
	if test_util.IsUnsupported(testCase.InputPath) || test_util.MatchesSkip(testCase.InputPath, walkTraversalSkipList) {
		t.Skipf("Skipping walk traversal test for %s", testCase.InputPath)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Walk panicked for %s: %v", testCase.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	result, err := testphases.RunPipeline(env, cx, nil, testphases.PhaseAST, testCase.InputPath)
	if err != nil {
		t.Errorf("pipeline failed for %s: %v", testCase.InputPath, err)
		return
	}

	visitor := &walkTestVisitor{
		t:            t,
		visitedTypes: make(map[string]int),
	}
	ast.Walk(visitor, result.CompilationUnit)

	if visitor.nodeCount == 0 {
		t.Errorf("Walk visited 0 nodes for %s", testCase.InputPath)
	}

	if testing.Verbose() {
		t.Logf("File: %s, Total nodes: %d", testCase.InputPath, visitor.nodeCount)
		for typeName, count := range visitor.visitedTypes {
			t.Logf("  %s: %d nodes", typeName, count)
		}
	}
}
