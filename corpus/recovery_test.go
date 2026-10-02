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
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/nodebuilder"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/langlib"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/test_util/testphases"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

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
			typeContext := semtypes.ContextFrom(env.GetTypeEnv())
			printer := ast.PrettyPrinter{LambdaResolutionContext: &typeContext}
			actualAST := printer.Print(result.Package)
			diagnosticResult := projects.NewDiagnosticResult(cx.Diagnostics())
			var diagnosticText bytes.Buffer
			testharness.PrintDiagnostics(os.DirFS("."), &diagnosticText, diagnosticResult, cx.DiagnosticEnv())
			testharness.ValidateErrorMarkers(t, inputPath, string(content), diagnosticResult, cx.DiagnosticEnv())
			compareRecoveryGolden(t, strings.TrimSuffix(inputPath, ".bal")+".ast.txt", actualAST)
			compareRecoveryGolden(t, strings.TrimSuffix(inputPath, ".bal")+".diagnostics.txt", normalizeIntegrationStderr(diagnosticText.String()))
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

type recoveryPhases struct {
	completed testphases.Phase
}

func (p *recoveryPhases) run(phase testphases.Phase, execute func() error) error {
	if phase != p.completed+1 {
		return fmt.Errorf("recovery phase %d must follow phase %d", phase, p.completed)
	}
	if err := execute(); err != nil {
		return err
	}
	p.completed = phase
	return nil
}

func runRecoveryPipeline(env *context.CompilerEnvironment, cx *context.CompilerContext, langlibs *langlib.Symbols, inputPath string, content string) (*testphases.PipelineResult, error) {
	result := &testphases.PipelineResult{}
	phases := recoveryPhases{completed: testphases.PhaseParse - 1}
	cx.DiagnosticEnv().RegisterFile(inputPath, text.NewStringTextDocument(content))
	var syntaxTree *st.SyntaxTree
	if err := phases.run(testphases.PhaseParse, func() error {
		var err error
		syntaxTree, err = parser.GetSyntaxTree(cx, inputPath, content)
		return err
	}); err != nil {
		return nil, fmt.Errorf("parsing recovery source: %w", err)
	}
	if err := phases.run(testphases.PhaseAST, func() error {
		result.CompilationUnit = nodebuilder.GetRecoveredCompilationUnit(cx, syntaxTree)
		if result.CompilationUnit == nil {
			return fmt.Errorf("recovered compilation unit is nil")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if langlibs == nil {
		var err error
		langlibs, err = testphases.LoadLanglibs(env, cx)
		if err != nil {
			return nil, fmt.Errorf("loading recovery langlibs: %w", err)
		}
	}
	pkgID := result.CompilationUnit.GetPackageID()
	units := []*ast.BLangCompilationUnit{result.CompilationUnit}
	importedSymbols := make(map[string]model.ExportedSymbolSpace)
	if err := phases.run(testphases.PhaseSymbolResolution, func() error {
		var pkgScope model.Scope
		pkgScope, _, importedSymbols = semantics.ResolveSymbols(cx, *pkgID, units, langlibs.ImplicitImports, langlibs.PublicSymbols, nil, "", "")
		result.Package = nodebuilder.ToPackageFromCompilationUnits(cx, units)
		result.Package.PackageID = pkgID
		result.Package.Scope = pkgScope
		return nil
	}); err != nil {
		return nil, err
	}
	if err := phases.run(testphases.PhaseTypeResolution, func() error {
		semantics.ResolvePublicNodeTypes(cx, result.Package, importedSymbols)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := phases.run(testphases.PhaseTypeNarrowing, func() error {
		semantics.ResolvePrivateNodesTypes(cx, result.Package, importedSymbols)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := phases.run(testphases.PhaseSemanticAnalysis, func() error {
		semantics.AnalyzeSemantics(cx, result.Package, importedSymbols)
		return nil
	}); err != nil {
		return nil, err
	}
	for _, d := range cx.Diagnostics() {
		info := d.DiagnosticInfo()
		if info.Code() == "INTERNAL_ERROR" || info.Code() == "UNIMPLEMENTED_ERROR" || info.Severity() == diagnostics.Fatal {
			return nil, fmt.Errorf("non-recoverable diagnostic: %s: %s", info.Code(), d.Message())
		}
	}
	if phases.completed != testphases.PhaseSemanticAnalysis {
		return nil, fmt.Errorf("recovery reached phase %d, want semantic analysis", phases.completed)
	}
	return result, nil
}
