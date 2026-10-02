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

package corpus

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/nodebuilder"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/langlib"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/test_util/testphases"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

const recoveryTarget = testphases.PhaseTypeResolution

func TestRecovery(t *testing.T) {
	count := 0
	err := filepath.WalkDir("recovery", func(inputPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(inputPath, ".bal") {
			return nil
		}
		count++
		t.Run(strings.TrimSuffix(strings.TrimPrefix(inputPath, "recovery/"), ".bal"), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("recovery pipeline panicked: %v", r)
				}
			}()
			content, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatal(err)
			}
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			result, err := runRecoveryPipeline(env, cx, nil, inputPath, string(content))
			if err != nil {
				t.Fatal(err)
			}
			printer := ast.PrettyPrinter{}
			actualAST := printer.Print(result.Package)
			diagnosticResult := projects.NewDiagnosticResult(cx.Diagnostics())
			var diagnosticText bytes.Buffer
			testharness.PrintDiagnostics(os.DirFS("."), &diagnosticText, diagnosticResult, cx.DiagnosticEnv())
			testharness.ValidateErrorMarkers(t, inputPath, string(content), diagnosticResult, cx.DiagnosticEnv())
			compareRecoveryGolden(t, strings.TrimSuffix(inputPath, ".bal")+".ast.txt", actualAST)
			compareRecoveryGolden(t, strings.TrimSuffix(inputPath, ".bal")+".diagnostics.txt", diagnosticText.String())
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("no recovery fixtures discovered")
	}
}

func compareRecoveryGolden(t *testing.T, expectedPath, actual string) {
	t.Helper()
	if *update {
		if test_util.UpdateIfNeeded(t, expectedPath, actual) {
			t.Errorf("updated recovery golden: %s", expectedPath)
		}
		return
	}
	expected := test_util.ReadExpectedFile(t, expectedPath)
	if expected != actual {
		t.Errorf("recovery golden mismatch: %s\n%s", expectedPath, test_util.FormatExpectedGot(expected, actual))
	}
}

func runRecoveryPipeline(env *context.CompilerEnvironment, cx *context.CompilerContext, langlibs *langlib.Symbols, inputPath string, content string) (*testphases.PipelineResult, error) {
	result := &testphases.PipelineResult{}
	cx.DiagnosticEnv().RegisterFile(inputPath, text.NewStringTextDocument(content))
	tree, err := parser.GetSyntaxTree(cx, inputPath, content)
	if err != nil {
		return nil, fmt.Errorf("parsing recovery source: %w", err)
	}
	result.CompilationUnit = nodebuilder.GetRecoveredCompilationUnit(cx, tree)
	if result.CompilationUnit == nil {
		return nil, fmt.Errorf("recovered compilation unit is nil")
	}
	if langlibs == nil {
		langlibs, err = testphases.LoadLanglibs(env, cx)
		if err != nil {
			return nil, fmt.Errorf("loading recovery langlibs: %w", err)
		}
	}
	pkgID := result.CompilationUnit.GetPackageID()
	units := []*ast.BLangCompilationUnit{result.CompilationUnit}
	pkgScope, _, importedSymbols := semantics.ResolveSymbols(cx, *pkgID, units, langlibs.ImplicitImports, langlibs.PublicSymbols, nil, "", "")
	result.Package = nodebuilder.ToPackageFromCompilationUnits(cx, units)
	result.Package.PackageID = pkgID
	result.Package.Scope = pkgScope
	semantics.ResolvePublicNodeTypes(cx, result.Package, importedSymbols)
	completed := testphases.PhaseTypeResolution
	for _, d := range cx.Diagnostics() {
		info := d.DiagnosticInfo()
		if info.Code() == "INTERNAL_ERROR" || info.Code() == "UNIMPLEMENTED_ERROR" || info.Severity() == diagnostics.Fatal {
			return nil, fmt.Errorf("non-recoverable diagnostic: %s: %s", info.Code(), d.Message())
		}
	}
	if completed != recoveryTarget {
		return nil, fmt.Errorf("recovery reached phase %d, want enabled phase %d", completed, recoveryTarget)
	}
	return result, nil
}
