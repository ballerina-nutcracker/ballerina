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
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
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
	"github.com/ballerina-nutcracker/ballerina/tools/text"
	"golang.org/x/tools/txtar"
)

func TestRecovery(t *testing.T) {
	t.Parallel()
	count := 0
	err := filepath.WalkDir("recovery", func(inputPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(inputPath, ".txtar") {
			return nil
		}
		count++
		t.Run(strings.TrimSuffix(strings.TrimPrefix(inputPath, "recovery/"), ".txtar"), func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("recovery pipeline panicked: %v", r)
				}
			}()
			archive, err := txtar.ParseFile(inputPath)
			if err != nil {
				t.Fatal(err)
			}
			sourceName := strings.TrimSuffix(filepath.Base(inputPath), ".txtar") + ".bal"
			if len(archive.Files) < 2 || archive.Files[0].Name != sourceName || archive.Files[1].Name != "ast" {
				t.Fatalf("expected %s and ast sections in %s", sourceName, inputPath)
			}
			content := string(archive.Files[0].Data)
			sourcePath := filepath.Join(filepath.Dir(inputPath), sourceName)
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			astOnly := strings.HasPrefix(filepath.ToSlash(inputPath), "recovery/ast/")
			var result *testphases.PipelineResult
			if astOnly {
				result, err = runRecoveryAST(cx, sourcePath, content)
			} else {
				result, err = runRecoveryPipeline(env, cx, nil, sourcePath, content)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range cx.Diagnostics() {
				if diagnostic.DiagnosticInfo().Code() == "INTERNAL_ERROR" {
					t.Errorf("recovery pipeline reported an internal error: %s", diagnostic.String())
				}
			}
			testharness.ValidateErrorMarkers(t, sourcePath, content, projects.NewDiagnosticResult(cx.Diagnostics()), cx.DiagnosticEnv())
			printer := ast.PrettyPrinter{Fallback: printRecoveryFallback, ShowNodeLocations: astOnly, DiagnosticEnv: cx.DiagnosticEnv()}
			var actualAST string
			if astOnly {
				actualAST = printer.Print(result.CompilationUnit)
			} else {
				actualAST = printer.Print(result.Package) + printLambdaResolutionTypes(semtypes.ContextFrom(env.GetTypeEnv()), result.Package)
			}
			compareRecoveryGolden(t, inputPath, archive, actualAST)
			for _, section := range archive.Files[2:] {
				switch section.Name {
				case "symbols":
					assertRecoverySymbols(t, result, string(section.Data))
				case "diagnostic-counts":
					assertRecoveryDiagnosticCounts(t, cx, string(section.Data))
				default:
					t.Fatalf("unexpected section %s in %s", section.Name, inputPath)
				}
			}
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

func assertRecoverySymbols(t *testing.T, result *testphases.PipelineResult, expected string) {
	t.Helper()
	if result.Package == nil || result.Package.Scope == nil {
		t.Fatal("symbol expectations require semantic recovery")
	}
	for _, expectation := range strings.Fields(expected) {
		if len(expectation) < 2 || (expectation[0] != '+' && expectation[0] != '-') {
			t.Fatalf("invalid symbol expectation %q: use +name or -name", expectation)
		}
		name := expectation[1:]
		_, found := result.Package.Scope.GetSymbol(name)
		if want := expectation[0] == '+'; found != want {
			t.Errorf("symbol %s: present = %t, want %t", name, found, want)
		}
	}
}

func assertRecoveryDiagnosticCounts(t *testing.T, cx *context.CompilerContext, expected string) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(expected), "\n") {
		count, message, ok := strings.Cut(line, " ")
		want, err := strconv.Atoi(count)
		if !ok || err != nil || want < 0 {
			t.Fatalf("invalid diagnostic-count expectation %q: use count message", line)
		}
		got := 0
		for _, diagnostic := range cx.Diagnostics() {
			if diagnostic.Message() == message {
				got++
			}
		}
		if got != want {
			t.Errorf("diagnostic %q: count = %d, want %d", message, got, want)
		}
	}
}

func printRecoveryFallback(p *ast.PrettyPrinter, node ast.BLangNode) {
	switch node.(type) {
	case *ast.BLangExternFunctionBody:
		p.StartNode()
		p.PrintString("extern-function-body")
		p.EndNode()
	default:
		panic(fmt.Sprintf("unsupported recovery node type: %T", node))
	}
}

type lambdaCollector struct {
	lambdas []*ast.BLangLambdaFunction
}

func (c *lambdaCollector) Visit(node ast.BLangNode) ast.Visitor {
	if lambda, ok := node.(*ast.BLangLambdaFunction); ok {
		c.lambdas = append(c.lambdas, lambda)
	}
	return c
}

func (c *lambdaCollector) VisitTypeData(*ast.TypeData) ast.Visitor { return c }

// printLambdaResolutionTypes reports the types resolved for each lambda, which
// the AST printer does not show, so recovery goldens can assert them.
func printLambdaResolutionTypes(typeContext semtypes.Context, pkg *ast.BLangPackage) string {
	collector := &lambdaCollector{}
	ast.Walk(collector, pkg)
	if len(collector.lambdas) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n(lambda-types")
	for _, lambda := range collector.lambdas {
		sb.WriteString("\n  (")
		if lambda.Function == nil {
			sb.WriteString("<missing-function>")
			writeLambdaResolutionType(&sb, typeContext, "callable", lambda.GetDeterminedType())
			sb.WriteString(")")
			continue
		}
		sb.WriteString(lambda.Function.Name.GetValue())
		writeLambdaResolutionType(&sb, typeContext, "callable", lambda.GetDeterminedType())
		writeLambdaResolutionType(&sb, typeContext, "function", lambda.Function.GetDeterminedType())
		writeLambdaResolutionType(&sb, typeContext, "name", lambda.Function.Name.GetDeterminedType())
		if body, ok := lambda.Function.Body.(*ast.BLangExprFunctionBody); ok {
			writeLambdaResolutionType(&sb, typeContext, "expression-body", body.GetDeterminedType())
		}
		sb.WriteString(")")
	}
	sb.WriteString(")\n")
	return sb.String()
}

func writeLambdaResolutionType(sb *strings.Builder, typeContext semtypes.Context, label string, ty semtypes.SemType) {
	value := "unresolved"
	if !semtypes.IsZero(ty) {
		value = semtypes.ToString(typeContext, ty)
	}
	fmt.Fprintf(sb, " [%s=%s]", label, value)
}

func compareRecoveryGolden(t *testing.T, inputPath string, archive *txtar.Archive, actualAST string) {
	t.Helper()
	expectedAST := string(archive.Files[1].Data)
	if *update {
		archive.Files[1].Data = []byte(actualAST)
		if test_util.UpdateTxtarArchiveIfNeeded(t, inputPath, archive.Files) {
			t.Errorf("updated recovery AST: %s", inputPath)
		}
		return
	}
	if test_util.NormalizeNewlines(expectedAST) != test_util.NormalizeNewlines(actualAST) {
		t.Errorf("recovery AST mismatch: %s\n%s", inputPath, test_util.FormatExpectedGot(expectedAST, actualAST))
	}
}

// AST-only fixtures exercise source forms whose semantic recovery is not supported yet.
func runRecoveryAST(cx *context.CompilerContext, inputPath, content string) (*testphases.PipelineResult, error) {
	cx.DiagnosticEnv().RegisterFile(inputPath, text.NewStringTextDocument(content))
	tree, err := parser.GetSyntaxTree(cx, inputPath, content)
	if err != nil {
		return nil, err
	}
	unit := nodebuilder.GetRecoveredCompilationUnit(cx, tree)
	if unit == nil {
		return nil, fmt.Errorf("recovered compilation unit is nil")
	}
	return &testphases.PipelineResult{CompilationUnit: unit}, nil
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
	if phases.completed != testphases.PhaseSemanticAnalysis {
		return nil, fmt.Errorf("recovery reached phase %d, want semantic analysis", phases.completed)
	}
	return result, nil
}
