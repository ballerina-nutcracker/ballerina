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

	"ballerina/ast"
	"ballerina/context"
	"ballerina/semtypes"
	"ballerina/test_util"
	"ballerina/test_util/testphases"
)

// semanticAnalysisSkipList is the semantic-analysis *additional* skip list,
// on top of the shared test_util.UnsupportedTests baseline.
var semanticAnalysisSkipList = []string{
	// https://github.com/ballerina-nutcracker/ballerina/issues/417
	"subset8/08-xml/namespace12-v.bal",
}

func TestSemanticAnalysis(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetValidAndPanicTests(t, test_util.AST)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testSemanticAnalysis(t, testPair)
		})
	}
}

func testSemanticAnalysis(t *testing.T, testCase test_util.TestCase) {
	if test_util.IsUnsupported(testCase.InputPath) || test_util.MatchesSkip(testCase.InputPath, semanticAnalysisSkipList) {
		t.Skipf("Skipping semantic analysis test for %s", testCase.InputPath)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Semantic analysis panicked for %s: %v", testCase.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Errorf("loading lang libraries failed for %s: %v", testCase.InputPath, err)
		return
	}
	result, err := testphases.RunPipeline(env, cx, langlibs, testphases.PhaseCFGAnalysis, testCase.InputPath)
	if err != nil {
		t.Errorf("pipeline failed for %s: %v", testCase.InputPath, err)
		return
	}
	if cx.HasErrors() {
		t.Fatalf("compiler context has errors for %s: %v", testCase.InputPath, cx.Diagnostics())
	}

	// Validate that all expressions have determinedTypes set
	validator := &semanticAnalysisValidator{t: t, ctx: cx}
	ast.Walk(validator, result.Package)

	// If we reach here, semantic analysis completed without panicking
	t.Logf("Semantic analysis completed successfully for %s", testCase.InputPath)
}

type semanticAnalysisValidator struct {
	t   *testing.T
	ctx *context.CompilerContext
}

func (v *semanticAnalysisValidator) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}

	// Validate determinedType is set on every expression node. NEVER is a
	// legitimate type for guaranteed-divergent expressions (e.g.
	// `check newError()` whose inner type is exactly `error`), so we only
	// flag the unset case.
	if _, ok := node.(ast.BLangExpression); ok {
		if semtypes.IsZero(node.GetDeterminedType()) {
			v.t.Errorf("determinedType not set for expression %T at %v",
				node, node.GetPosition())
		}
	}

	if inv, ok := node.(*ast.BLangInvocation); ok && ast.IsStreamOperation(inv) {
		return v
	}
	// Check if node has a symbol that should have type set
	if nodeWithSymbol, ok := node.(ast.BNodeWithSymbol); ok {
		symbol := nodeWithSymbol.Symbol()
		if semtypes.IsZero(v.ctx.SymbolType(symbol)) {
			v.t.Errorf("symbol %s (kind: %v) does not have type set for node %T at %v",
				v.ctx.SymbolName(symbol), v.ctx.SymbolKind(symbol), node, node.GetPosition())
		}
	}

	return v
}

func (v *semanticAnalysisValidator) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	if typeData == nil || typeData.TypeDescriptor == nil {
		return v
	}

	// Check if type descriptor has a symbol that should have type set
	if typeWithSymbol, ok := typeData.TypeDescriptor.(ast.BNodeWithSymbol); ok {
		symbol := typeWithSymbol.Symbol()
		if semtypes.IsZero(v.ctx.SymbolType(symbol)) {
			v.t.Errorf("symbol %s (kind: %v) does not have type set for type descriptor %T at %v",
				v.ctx.SymbolName(symbol), v.ctx.SymbolKind(symbol), typeData.TypeDescriptor, typeData.TypeDescriptor.GetPosition())
		}
	}

	return v
}

// semanticAnalysisErrorSkipList is the semantic-analysis-errors *additional*
// skip list, on top of the shared test_util.UnsupportedTests baseline.
// Currently empty -- every known failure is already covered by the shared
// baseline.
var semanticAnalysisErrorSkipList = []string{}

func TestSemanticAnalysisErrors(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetErrorTests(t, test_util.AST)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testSemanticAnalysisError(t, testPair)
		})
	}
}

func testSemanticAnalysisError(t *testing.T, testCase test_util.TestCase) {
	if test_util.IsUnsupported(testCase.InputPath) || test_util.MatchesSkip(testCase.InputPath, semanticAnalysisErrorSkipList) {
		t.Skipf("Skipping semantic analysis error test for %s", testCase.InputPath)
		return
	}

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Semantic analysis panicked for %s: %v", testCase.InputPath, r)
		}

		if !cx.HasDiagnostics() {
			t.Errorf("Expected compile-time diagnostics for %s, but no diagnostics were recorded", testCase.InputPath)
			return
		}

		t.Logf("Compile-time diagnostic correctly detected for %s", testCase.InputPath)
	}()

	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Errorf("loading lang libraries failed for %s: %v", testCase.InputPath, err)
		return
	}
	_, _ = testphases.RunPipeline(env, cx, langlibs, testphases.PhaseCFGAnalysis, testCase.InputPath)

	// If we reach here without panic, the defer will catch it
}
