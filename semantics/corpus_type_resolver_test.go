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

	"ballerina-lang-go/ast"
	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testphases"
)

// typeResolverSkipList is the type-resolver *additional* skip list, on top of
// the shared test_util.UnsupportedTests baseline.
var typeResolverSkipList = []string{
	// https://github.com/ballerina-platform/ballerina-lang-go/issues/417
	"subset8/08-xml/namespace12-v.bal",
}

func TestTypeResolver(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetValidAndPanicTests(t, test_util.AST)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testTypeResolution(t, testPair)
		})
	}
}

func testTypeResolution(t *testing.T, testCase test_util.TestCase) {
	if test_util.IsUnsupported(testCase.InputPath) || test_util.MatchesSkip(testCase.InputPath, typeResolverSkipList) {
		t.Skipf("Skipping type resolver test for %s", testCase.InputPath)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Type resolution panicked for %s: %v", testCase.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Errorf("loading lang libraries failed for %s: %v", testCase.InputPath, err)
		return
	}
	result, err := testphases.RunPipeline(env, cx, langlibs, testphases.PhaseTypeNarrowing, testCase.InputPath)
	if err != nil {
		t.Errorf("pipeline failed for %s: %v", testCase.InputPath, err)
		return
	}
	tyCtx := semtypes.ContextFrom(cx.GetTypeEnv())
	validator := &typeResolutionValidator{t: t, ctx: cx, tyCtx: tyCtx}
	ast.Walk(validator, result.Package)

	// If we reach here, type resolution completed without panicking
	t.Logf("Type resolution completed successfully for %s", testCase.InputPath)
}

type typeResolutionValidator struct {
	t     *testing.T
	ctx   *context.CompilerContext
	tyCtx semtypes.Context
}

func (v *typeResolutionValidator) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}

	// Validate that all BLangExpression nodes have their determined type set.
	// NEVER is a legitimate type for guaranteed-divergent expressions (e.g.
	// `check newError()` or `checkpanic newError()` whose inner type is
	// exactly `error`, so the non-error remainder is empty).
	if expr, ok := node.(ast.BLangExpression); ok {
		if semtypes.IsZero(expr.GetDeterminedType()) {
			v.t.Errorf("expression %T at %v does not have determined type set", expr, expr.GetPosition())
		}
	}

	if inv, ok := node.(*ast.BLangInvocation); ok && ast.IsStreamOperation(inv) {
		return v
	}
	if nodeWithSymbol, ok := node.(ast.BNodeWithSymbol); ok {
		symbol := nodeWithSymbol.Symbol()
		// Skip constant symbols (kind: 1) since they're resolved during semantic analysis
		if v.ctx.SymbolKind(symbol) == model.SymbolKindConstant {
			return v
		}
		if semtypes.IsZero(v.ctx.SymbolType(symbol)) {
			// FIXME: get rid of this
			if _, ok := node.(*ast.BLangConstant); ok {
				// constants will get their type set during semantic analysis
				return v
			}
			v.t.Errorf("symbol %s (kind: %v) does not have type set for node %T",
				v.ctx.SymbolName(symbol), v.ctx.SymbolKind(symbol), node)
		}
	}

	return v
}

func (v *typeResolutionValidator) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	if typeData.TypeDescriptor == nil {
		return nil
	}
	if semtypes.IsZero(typeData.Type) {
		v.t.Errorf("type not resolved for %+v", typeData)
	}
	return v
}
