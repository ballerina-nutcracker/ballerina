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

// Command funcusage reports package-level functions of the given workspace
// modules that production code does not need: private functions that no
// production code references, and exported functions that no production code
// outside their package references. Test files and the root module's test_util
// helpers never count as uses. A function that exists only as a hook for tests
// is exempted by a doc-comment directive "//funcusage:test-hook <reason>" on
// each of its declarations.
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

const testHookDirective = "//funcusage:test-hook"

// testOnlyDir is the root-relative directory of test-only helper packages,
// whose references do not count as production uses.
const testOnlyDir = "test_util"

// buildTags lists build configurations whose union covers every
// build-constrained production file in the workspace.
var buildTags = [][]string{nil, {"-tags=debug,native_interp"}}

type finding struct {
	position token.Position
	message  string
}

func (f finding) String() string {
	return fmt.Sprintf("%s: funcusage: %s", f.position, f.message)
}

type function struct {
	pkg   string
	name  string
	decls []declaration
}

// declaration is one build-specific declaration of a function.
type declaration struct {
	position token.Position
	hook     hookDirective
}

type hookDirective struct {
	present  bool
	reason   string
	position token.Position
}

type usage struct {
	inPackage    bool
	otherPackage bool
}

type workspace struct {
	functions map[string]*function
	uses      map[string]*usage
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: funcusage <module directory>...")
		os.Exit(1)
	}
	findings, err := check(".", os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, f := range findings {
		fmt.Fprintln(os.Stderr, f)
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func check(root string, modules []string) ([]finding, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	checked, err := absoluteDirs(root, modules)
	if err != nil {
		return nil, err
	}
	patterns, err := workspacePatterns(root)
	if err != nil {
		return nil, err
	}
	ws := &workspace{functions: map[string]*function{}, uses: map[string]*usage{}}
	for _, flags := range buildTags {
		if err := ws.load(root, patterns, flags, checked); err != nil {
			return nil, err
		}
	}
	return ws.findings(root), nil
}

func absoluteDirs(root string, modules []string) ([]string, error) {
	dirs := make([]string, 0, len(modules))
	for _, module := range modules {
		dir := filepath.Join(root, module)
		info, err := os.Stat(dir)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", module)
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func workspacePatterns(root string) ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing workspace modules: %w", err)
	}
	var patterns []string
	for dir := range strings.Lines(string(output)) {
		dir = strings.TrimRight(dir, "\r\n")
		if dir == "" {
			continue
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, "./"+filepath.ToSlash(rel)+"/...")
	}
	return patterns, nil
}

func (ws *workspace) load(root string, patterns, flags []string, checked []string) error {
	cfg := &packages.Config{
		Mode:       packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:        root,
		BuildFlags: flags,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return err
	}
	var loadErrors []string
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			loadErrors = append(loadErrors, e.Error())
		}
	}
	if len(loadErrors) > 0 {
		return fmt.Errorf("loading packages %v:\n%s", flags, strings.Join(loadErrors, "\n"))
	}
	testOnly := []string{filepath.Join(root, testOnlyDir)}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			filename := pkg.Fset.Position(file.Pos()).Filename
			if within(filename, checked) {
				ws.collectFunctions(pkg, file)
			}
			if !within(filename, testOnly) {
				ws.collectUses(pkg, file)
			}
		}
	}
	return nil
}

func within(path string, dirs []string) bool {
	for _, dir := range dirs {
		if rel, err := filepath.Rel(dir, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (ws *workspace) collectFunctions(pkg *packages.Package, file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || isExempt(pkg, fn) {
			continue
		}
		key := functionKey(pkg.PkgPath, fn.Name.Name)
		f, ok := ws.functions[key]
		if !ok {
			f = &function{pkg: pkg.PkgPath, name: fn.Name.Name}
			ws.functions[key] = f
		}
		position := pkg.Fset.Position(fn.Name.Pos())
		// Files without build constraints are loaded once per build configuration.
		if slices.ContainsFunc(f.decls, func(d declaration) bool { return d.position == position }) {
			continue
		}
		f.decls = append(f.decls, declaration{position: position, hook: testHook(pkg.Fset, fn.Doc)})
	}
}

func isExempt(pkg *packages.Package, fn *ast.FuncDecl) bool {
	switch fn.Name.Name {
	case "init", "_":
		return true
	case "main":
		return pkg.Name == "main"
	}
	return false
}

func testHook(fset *token.FileSet, doc *ast.CommentGroup) hookDirective {
	if doc == nil {
		return hookDirective{}
	}
	for _, comment := range doc.List {
		rest, ok := strings.CutPrefix(comment.Text, testHookDirective)
		if ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
			return hookDirective{present: true, reason: strings.TrimSpace(rest), position: fset.Position(comment.Pos())}
		}
	}
	return hookDirective{}
}

func (ws *workspace) collectUses(pkg *packages.Package, file *ast.File) {
	for _, decl := range file.Decls {
		enclosing := ""
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
			enclosing = functionKey(pkg.PkgPath, fn.Name.Name)
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			declPkg, key, ok := packageFunction(pkg.TypesInfo.Uses[ident])
			if !ok || key == enclosing {
				return true
			}
			ws.recordUse(key, declPkg == pkg.PkgPath)
			return true
		})
	}
}

func packageFunction(obj types.Object) (pkg, key string, ok bool) {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return "", "", false
	}
	fn = fn.Origin()
	if fn.Signature().Recv() != nil {
		return "", "", false
	}
	return fn.Pkg().Path(), functionKey(fn.Pkg().Path(), fn.Name()), true
}

func (ws *workspace) recordUse(key string, samePackage bool) {
	u, ok := ws.uses[key]
	if !ok {
		u = &usage{}
		ws.uses[key] = u
	}
	if samePackage {
		u.inPackage = true
	} else {
		u.otherPackage = true
	}
}

func functionKey(pkg, name string) string {
	return pkg + "." + name
}

func (ws *workspace) findings(root string) []finding {
	var findings []finding
	for key, fn := range ws.functions {
		u := ws.uses[key]
		if u == nil {
			u = &usage{}
		}
		message := violation(fn, u)
		used := u.inPackage || u.otherPackage
		for _, decl := range fn.decls {
			switch {
			case decl.hook.present && decl.hook.reason == "":
				findings = append(findings, finding{relative(root, decl.hook.position), testHookDirective + " requires a reason"})
			case decl.hook.present && used:
				findings = append(findings, finding{relative(root, decl.hook.position), fmt.Sprintf("%s is used by production code; remove the unnecessary %s directive", fn.name, testHookDirective)})
			case !decl.hook.present && message != "":
				findings = append(findings, finding{relative(root, decl.position), message})
			}
		}
	}
	slices.SortFunc(findings, func(a, b finding) int {
		if c := strings.Compare(a.position.Filename, b.position.Filename); c != 0 {
			return c
		}
		return a.position.Line - b.position.Line
	})
	return findings
}

func violation(fn *function, u *usage) string {
	if !token.IsExported(fn.name) {
		if u.inPackage || u.otherPackage {
			return ""
		}
		return fmt.Sprintf("private function %s is not used by production code; remove it, move it to a test file, or mark it %s <reason>", fn.name, testHookDirective)
	}
	if u.otherPackage {
		return ""
	}
	if u.inPackage {
		return fmt.Sprintf("exported function %s is not used outside package %s; unexport it", fn.name, fn.pkg)
	}
	return fmt.Sprintf("exported function %s is not used by production code; remove it, move it to a test file, or mark it %s <reason>", fn.name, testHookDirective)
}

func relative(root string, position token.Position) token.Position {
	if rel, err := filepath.Rel(root, position.Filename); err == nil {
		position.Filename = filepath.ToSlash(rel)
	}
	return position
}
