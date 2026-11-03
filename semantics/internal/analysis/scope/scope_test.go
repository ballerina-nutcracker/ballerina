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

package scope_test

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util/testphases"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

const forwardingSource = `
function f() {
}

function g() {
}
`

// TestSemanticErrIsForwardedUnchanged reports the same diagnostic twice
// through every context constructor and expects both to be recorded: no
// context may deduplicate or suppress a report.
//
// This is a unit test because a corpus test cannot report the same diagnostic
// twice at one location through a given context chain. The subtests run
// sequentially because they share one compiler context and count its
// diagnostics.
func TestSemanticErrIsForwardedUnchanged(t *testing.T) {
	t.Parallel()
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	ctx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, ctx)
	if err != nil {
		t.Fatal(err)
	}
	result, err := testphases.RunPipelineWithContent(env, ctx, langlibs, testphases.PhaseTypeNarrowing, "forwarding.bal", forwardingSource)
	if err != nil {
		t.Fatal(err)
	}
	functions := result.Package.Functions
	pkg := scope.NewPackage(ctx, semtypes.ContextFrom(ctx.GetTypeEnv()), result.Package, nil)
	fn := scope.NewFunction(pkg, functions[0])
	lambda := scope.NewLambda(fn, functions[1])
	lock := scope.NewLock(fn, &ast.BLangLock{})
	literal := ast.NewBLangLiteral(diagnostics.NewBuiltinLocation(), ast.LiteralKindNone, int64(1), "1", false)
	literal.Ty = semtypes.Int
	chains := map[string]scope.Context{
		"package":                  pkg,
		"function":                 fn,
		"lambda":                   lambda,
		"default parameter":        scope.NewExpressionClosure(fn, scope.DefaultParam),
		"record field default":     scope.NewExpressionClosure(pkg, scope.RecordFieldDefault),
		"lock":                     lock,
		"isolated expression":      scope.NewIsolatedExpr(lock, literal),
		"lambda in lock":           scope.NewLambda(lock, functions[1]),
		"isolated expression body": scope.NewExpressionClosure(scope.NewIsolatedExpr(lock, literal), scope.RecordFieldDefault),
	}
	for name, cx := range chains {
		t.Run(name, func(t *testing.T) {
			before := len(ctx.Diagnostics())
			loc := diagnostics.NewBuiltinLocation()
			cx.SemanticErr("duplicate", loc)
			cx.SemanticErr("duplicate", loc)
			if got := len(ctx.Diagnostics()) - before; got != 2 {
				t.Fatalf("expected 2 diagnostics, got %d", got)
			}
		})
	}
}
