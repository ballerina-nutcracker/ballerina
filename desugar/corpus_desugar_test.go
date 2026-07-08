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

package desugar_test

import (
	"flag"
	"fmt"
	"sort"
	"strings"
	"testing"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/context"
	"ballerina-lang-go/desugar"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testphases"
	"ballerina-lang-go/tools/diagnostics"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var update = flag.Bool("update", false, "update expected desugared AST files")

// desugarSkipList is the desugar-stage *additional* skip list, on top of the
// shared test_util.UnsupportedTests baseline. Currently empty -- every known
// failure is already covered by the shared baseline.
var desugarSkipList = []string{}

func TestDesugar(t *testing.T) {
	flag.Parse()

	testPairs := test_util.GetValidAndPanicTests(t, test_util.Desugar)

	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel()
			testDesugar(t, testPair)
		})
	}
}

type walkTestVisitor struct {
	t *testing.T
}

func (v *walkTestVisitor) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}

	if diagnostics.IsLocationEmpty(node.GetPosition()) {
		v.t.Errorf("node with missing position: %T", node)
	}

	return v
}

func (v *walkTestVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return v
}

func testDesugar(t *testing.T, testCase test_util.TestCase) {
	if test_util.IsUnsupported(testCase.InputPath) || test_util.MatchesSkip(testCase.InputPath, desugarSkipList) {
		t.Skipf("Skipping desugar test for %s", testCase.InputPath)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Desugar panicked for %s: %v", testCase.InputPath, r)
		}
	}()

	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := context.NewCompilerContext(env)
	langlibs, err := testphases.LoadLanglibs(env, cx)
	if err != nil {
		t.Errorf("loading lang libraries failed for %s: %v", testCase.InputPath, err)
		return
	}
	result, err := testphases.RunPipeline(env, cx, langlibs, testphases.PhaseDesugar, testCase.InputPath)
	if err != nil {
		t.Errorf("pipeline failed for %s: %v", testCase.InputPath, err)
		return
	}

	// Serialize AST after desugaring
	prettyPrinter := ast.PrettyPrinter{Fallback: prettyPrintFallback}
	actualAST := prettyPrinter.Print(result.Package)

	// If update flag is set, update expected file
	if *update {
		if test_util.UpdateIfNeeded(t, testCase.ExpectedPath, actualAST, normalizeDesugaredAST) {
			t.Errorf("updated expected desugared AST file: %s", testCase.ExpectedPath)
		}
		return
	}

	// Read expected AST file
	expectedAST := test_util.ReadExpectedFile(t, testCase.ExpectedPath)

	// The synthetic init function emits assignments in a topo order whose
	// peer-statement order depends on Go's map iteration; commuting statements
	// that share no dependency edge produces an equivalent program. Import
	// declaration order can also vary. Compare these order-insensitive regions in
	// canonical order so the test is stable regardless of frontend collection
	// order.
	if normalizeDesugaredAST(actualAST) != normalizeDesugaredAST(expectedAST) {
		t.Errorf("Desugared AST mismatch for %s\nExpected file: %s\n%s",
			testCase.InputPath, testCase.ExpectedPath, getDiff(expectedAST, actualAST))
		return
	}

	visitor := &walkTestVisitor{t: t}
	ast.Walk(visitor, result.CompilationUnit)

	t.Logf("Desugar completed successfully for %s", testCase.InputPath)
}

func getDiff(expectedAST, actualAST string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expectedAST, actualAST, false)
	return dmp.DiffPrettyText(diffs)
}

// sexp is a minimal s-expression node used to canonicalise the desugared AST
// snapshot for comparison purposes.
type sexp struct {
	isAtom bool
	atom   string
	list   []*sexp
}

func tokenizeSExp(s string) []string {
	var toks []string
	i := 0
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '(' || c == ')' {
			toks = append(toks, string(c))
			i++
			continue
		}
		start := i
		for i < len(s) {
			c := s[i]
			if c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '(' || c == ')' {
				break
			}
			i++
		}
		toks = append(toks, s[start:i])
	}
	return toks
}

func parseSExp(toks []string, pos int) (*sexp, int, bool) {
	if pos >= len(toks) {
		return nil, pos, false
	}
	t := toks[pos]
	if t == "(" {
		pos++
		l := &sexp{}
		for pos < len(toks) && toks[pos] != ")" {
			child, np, ok := parseSExp(toks, pos)
			if !ok {
				return nil, np, false
			}
			l.list = append(l.list, child)
			pos = np
		}
		if pos >= len(toks) {
			return nil, pos, false
		}
		return l, pos + 1, true
	}
	if t == ")" {
		return nil, pos, false
	}
	return &sexp{isAtom: true, atom: t}, pos + 1, true
}

func printSExp(e *sexp) string {
	if e.isAtom {
		return e.atom
	}
	parts := make([]string, len(e.list))
	for i, c := range e.list {
		parts[i] = printSExp(c)
	}
	return "(" + strings.Join(parts, " ") + ")"
}

// canonicalisePackageImports sorts top-level import-package nodes by their
// printed form while leaving every non-import package child in place.
func canonicalisePackageImports(e *sexp) {
	if e.isAtom || !isPackageSExp(e) {
		return
	}

	imports := make([]*sexp, 0)
	importSlots := make([]int, 0)
	for i := 1; i < len(e.list); i++ {
		if isImportPackageSExp(e.list[i]) {
			imports = append(imports, e.list[i])
			importSlots = append(importSlots, i)
		}
	}
	sort.Slice(imports, func(i, j int) bool {
		return printSExp(imports[i]) < printSExp(imports[j])
	})
	for i, slot := range importSlots {
		e.list[slot] = imports[i]
	}
}

// canonicaliseInitFnBodies finds every (function init () () (block-function-body ...))
// node in the AST and sorts its block-function-body's child statements by
// their printed form. The synthetic init function's assignments commute as
// long as topo constraints are respected, so reordering peer assignments must
// not be observable by the snapshot.
func canonicaliseInitFnBodies(e *sexp) {
	if e.isAtom {
		return
	}
	if isInitFnSExp(e) {
		for _, child := range e.list {
			if child.isAtom || len(child.list) == 0 {
				continue
			}
			head := child.list[0]
			if !head.isAtom || head.atom != "block-function-body" {
				continue
			}
			stmts := child.list[1:]
			sort.Slice(stmts, func(i, j int) bool {
				return printSExp(stmts[i]) < printSExp(stmts[j])
			})
		}
	}
	for _, child := range e.list {
		canonicaliseInitFnBodies(child)
	}
}

func isInitFnSExp(e *sexp) bool {
	if e.isAtom || len(e.list) < 2 {
		return false
	}
	if !e.list[0].isAtom || e.list[0].atom != "function" {
		return false
	}
	if !e.list[1].isAtom || e.list[1].atom != "init" {
		return false
	}
	return true
}

func isPackageSExp(e *sexp) bool {
	return !e.isAtom && len(e.list) > 0 && e.list[0].isAtom && e.list[0].atom == "package"
}

func isImportPackageSExp(e *sexp) bool {
	return !e.isAtom && len(e.list) > 0 && e.list[0].isAtom && e.list[0].atom == "import-package"
}

func normalizeDesugaredAST(s string) string {
	toks := tokenizeSExp(s)
	if len(toks) == 0 {
		return s
	}
	root, _, ok := parseSExp(toks, 0)
	if !ok {
		return s
	}
	canonicalisePackageImports(root)
	canonicaliseInitFnBodies(root)
	return printSExp(root)
}

// prettyPrintFallback handles desugar-introduced AST nodes when serializing a
// desugared package via ast.PrettyPrinter. Wire it in by setting
// PrettyPrinter.Fallback to this function.
func prettyPrintFallback(p *ast.PrettyPrinter, node ast.BLangNode) {
	switch n := node.(type) {
	case *desugar.BLangServiceInit:
		p.StartNode()
		p.PrintString("service-init")
		p.EndNode()
	default:
		panic(fmt.Sprintf("desugar pretty printer: unsupported node %T", n))
	}
}
