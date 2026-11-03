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

// Package walkpurity checks that AST walks in the type resolver only gather data. Walks and visitors must live in
// walks.go, and walks.go may only write non-AST state and use functions declared in it or on an allowlist.
package walkpurity

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	astPkg    = "github.com/ballerina-nutcracker/ballerina/ast"
	walksFile = "walks.go"
)

// Interface methods declared in the analyzed package that walks may use.
var pureMethods = map[string]map[string]bool{
	"typeResolver": {
		"typeContext": true, "typeEnv": true, "expectedReturnType": true, "parent": true,
		"isEphemeral": true, "symbolType": true, "getSymbol": true, "unnarrowedSymbol": true,
		"symbolName": true, "functionSignature": true, "functionSignatureRef": true,
		"functionSignatureByRef": true, "mappingDefaults": true, "objectMethodTable": true,
		"lookupImportedSymbols": true, "hasImplicitImport": true,
	},
}

// Functions and methods declared in other packages that walks may use, keyed by package path. A method name allows
// that method on every type of the package, including interface methods.
var allowedFuncs = map[string]map[string]bool{
	astPkg: {"Walk": true, "WalkTypeData": true, "Symbol": true},
	"github.com/ballerina-nutcracker/ballerina/model": {"IsEmpty": true},
	"sort": {"Ints": true},
}

var Analyzer = &analysis.Analyzer{
	Name: "walkpurity",
	Doc:  "checks that AST walks live in walks.go and only use allowlisted functions",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	visitor := lookupVisitor(pass.Pkg)
	for _, file := range pass.Files {
		name := fileName(pass, file)
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name != walksFile {
			if visitor != nil {
				checkPlacement(pass, file, visitor)
			}
			continue
		}
		for _, decl := range file.Decls {
			checkPurity(pass, decl)
		}
	}
	checkIgnoredFiles(pass)
	return nil, nil
}

// checkIgnoredFiles reports files that build constraints exclude from this run, since neither rule can check them.
func checkIgnoredFiles(pass *analysis.Pass) {
	if len(pass.Files) == 0 {
		return
	}
	pos := pass.Files[0].Package
	for _, file := range pass.Files {
		if fileName(pass, file) == walksFile {
			pos = file.Package
		}
	}
	for _, path := range pass.IgnoredFiles {
		name := filepath.Base(path)
		if strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			continue
		}
		pass.Reportf(pos, "%s is excluded by build constraints, so it can't be checked", name)
	}
}

// fileName returns the base name of the file that declares node, ignoring //line directives.
func fileName(pass *analysis.Pass, node interface{ Pos() token.Pos }) string {
	return filepath.Base(pass.Fset.File(node.Pos()).Name())
}

func lookupVisitor(pkg *types.Package) *types.Interface {
	for _, imp := range pkg.Imports() {
		if imp.Path() != astPkg {
			continue
		}
		obj, ok := imp.Scope().Lookup("Visitor").(*types.TypeName)
		if !ok {
			return nil
		}
		iface, _ := obj.Type().Underlying().(*types.Interface)
		return iface
	}
	return nil
}

func checkPlacement(pass *analysis.Pass, file *ast.File, visitor *types.Interface) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if isAstWalk(pass.TypesInfo.Uses[n]) {
				pass.Reportf(n.Pos(), "AST walks must live in %s", walksFile)
			}
		case *ast.TypeSpec:
			obj, ok := pass.TypesInfo.Defs[n.Name].(*types.TypeName)
			if !ok {
				return true
			}
			ty := obj.Type()
			if types.IsInterface(ty) {
				return true
			}
			if types.Implements(ty, visitor) || types.Implements(types.NewPointer(ty), visitor) {
				pass.Reportf(n.Pos(), "ast.Visitor implementations must live in %s", walksFile)
			}
		case *ast.FuncDecl:
			if isVisitorMethod(pass.TypesInfo.Defs[n.Name], visitor) {
				pass.Reportf(n.Name.Pos(), "ast.Visitor methods must live in %s", walksFile)
			}
		}
		return true
	})
}

// isVisitorMethod reports whether obj is a method with the name and signature of an ast.Visitor method, so a visitor
// can't get it from outside walks.go through embedding or a method declared in another file.
func isVisitorMethod(obj types.Object, visitor *types.Interface) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Signature().Recv() == nil {
		return false
	}
	for method := range visitor.Methods() {
		if method.Name() == fn.Name() && types.Identical(method.Type(), fn.Type()) {
			return true
		}
	}
	return false
}

func isAstWalk(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != astPkg {
		return false
	}
	return fn.Name() == "Walk" || fn.Name() == "WalkTypeData"
}

// checkPurity reports AST field writes, unresolved calls, and uses of functions that are neither declared in walks.go
// nor allowlisted, anywhere in decl.
func checkPurity(pass *analysis.Pass, decl ast.Decl) {
	info := pass.TypesInfo
	owner := declName(decl)
	report := func(n ast.Expr, why string) {
		pass.Reportf(n.Pos(), "%s -> %s: %s", owner, types.ExprString(n), why)
	}
	addWrites := func(lhs ...ast.Expr) {
		for _, e := range lhs {
			if e != nil && writesAstField(info, e) {
				report(e, "write to ast node field")
			}
		}
	}
	checkUse := func(expr ast.Expr, ident *ast.Ident) {
		obj := info.Uses[ident]
		if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "unsafe" {
			report(expr, "not allowed: unsafe."+obj.Name())
			return
		}
		fn, ok := obj.(*types.Func)
		if !ok {
			return
		}
		if why, allowed := allowedUse(pass, fn.Origin()); !allowed {
			report(expr, why)
		}
	}
	var visit func(ast.Node) bool
	visit = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			if n.Path.Value == `"unsafe"` {
				report(n.Path, "not allowed: unsafe")
			}
			return false
		case *ast.FuncDecl:
			if n.Body == nil {
				report(n.Name, "function without a body")
			}
		case *ast.AssignStmt:
			addWrites(n.Lhs...)
		case *ast.IncDecStmt:
			addWrites(n.X)
		case *ast.RangeStmt:
			if n.Tok == token.ASSIGN {
				addWrites(n.Key, n.Value)
			}
			if rangesOverFuncValue(info, n.X) {
				report(n.X, "unresolved call")
			}
		case *ast.CallExpr:
			if tv, ok := info.Types[n.Fun]; ok && tv.IsType() {
				return true
			}
			switch callee := typeutil.Callee(info, n).(type) {
			case *types.Builtin:
				if writesAstThroughBuiltin(info, callee.Name(), n.Args) {
					report(n.Fun, "write to ast node field")
				}
			case *types.Func:
				if isAstWalk(callee) && len(n.Args) > 0 {
					checkWalkVisitor(pass, n.Args[0], report)
				}
			default:
				report(n.Fun, "unresolved call")
			}
		case *ast.SelectorExpr:
			checkUse(n, n.Sel)
			ast.Inspect(n.X, visit)
			return false
		case *ast.Ident:
			checkUse(n, n)
		}
		return true
	}
	ast.Inspect(decl, visit)
}

func declName(decl ast.Decl) string {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		return decl.Name.Name
	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.ValueSpec:
				return spec.Names[0].Name
			case *ast.TypeSpec:
				return spec.Name.Name
			}
		}
	}
	return ""
}

// allowedUse reports whether walks may use fn, and if not, why. An interface method is allowed only when listed, even
// if its interface is declared in walks.go, because the call dispatches to code outside walks.go.
func allowedUse(pass *analysis.Pass, fn *types.Func) (string, bool) {
	name := fn.Name()
	recv := fn.Signature().Recv()
	switch {
	case fn.Pkg() == nil:
		return "not allowed: " + name, false
	case fn.Pkg() != pass.Pkg:
		return "not allowed: " + fn.Pkg().Name() + "." + name, allowedFuncs[fn.Pkg().Path()][name]
	case recv != nil && types.IsInterface(recv.Type()):
		iface := interfaceName(recv.Type())
		return "not allowed: " + iface + "." + name, isPackageLevel(pass, recv.Type()) && pureMethods[iface][name]
	}
	return "not allowed: " + name, fileName(pass, fn) == walksFile
}

// writesAstThroughBuiltin reports whether a builtin call writes into its first argument, and that argument is reached
// through an ast value. append only writes into the existing backing array when it is given a slice expression.
func writesAstThroughBuiltin(info *types.Info, name string, args []ast.Expr) bool {
	if len(args) == 0 {
		return false
	}
	switch name {
	case "copy", "clear", "delete":
		return writesAstField(info, args[0])
	case "append":
		_, isSlice := ast.Unparen(args[0]).(*ast.SliceExpr)
		return isSlice && writesAstField(info, args[0])
	}
	return false
}

// isPackageLevel reports whether t is a named type declared at the top level of the analyzed package, so a local type
// that reuses a listed interface's name doesn't get its pure methods.
func isPackageLevel(pass *analysis.Pass, t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && pass.Pkg.Scope().Lookup(named.Obj().Name()) == named.Obj()
}

// rangesOverFuncValue reports whether a range statement over x calls a function value. Ranging over a declared function
// is a use of that function, which checkUse handles.
func rangesOverFuncValue(info *types.Info, x ast.Expr) bool {
	if _, ok := types.Unalias(info.TypeOf(x)).(*types.TypeParam); ok {
		return true
	}
	if _, ok := types.Unalias(info.TypeOf(x)).Underlying().(*types.Signature); !ok {
		return false
	}
	switch x := ast.Unparen(x).(type) {
	case *ast.Ident:
		_, ok := info.Uses[x].(*types.Func)
		return !ok
	case *ast.SelectorExpr:
		_, ok := info.Uses[x.Sel].(*types.Func)
		return !ok
	}
	return true
}

// checkWalkVisitor reports a visitor passed to ast.Walk whose Visit or VisitTypeData method is not declared in walks.go,
// including one reached through an interface or promoted from an embedded type.
func checkWalkVisitor(pass *analysis.Pass, arg ast.Expr, report func(ast.Expr, string)) {
	methods := types.NewMethodSet(pass.TypesInfo.TypeOf(arg))
	for _, name := range []string{"Visit", "VisitTypeData"} {
		sel := methods.Lookup(nil, name)
		if sel == nil {
			continue
		}
		method := sel.Obj()
		if method.Pkg() != pass.Pkg || fileName(pass, method) != walksFile {
			report(arg, "visitor method "+name+" is not declared in "+walksFile)
		}
	}
}

func interfaceName(t types.Type) string {
	if named, ok := types.Unalias(t).(*types.Named); ok {
		return named.Obj().Name()
	}
	return t.String()
}

// writesAstField reports whether lhs writes through a value of an ast type: x.f = …, x.f[i] = … or *x = …, where any
// base along the chain has an ast type or any selected field is declared in the ast package.
func writesAstField(info *types.Info, lhs ast.Expr) bool {
	for e := lhs; ; {
		var base ast.Expr
		switch v := ast.Unparen(e).(type) {
		case *ast.SelectorExpr:
			if isAstField(info.Selections[v]) {
				return true
			}
			base = v.X
		case *ast.IndexExpr:
			base = v.X
		case *ast.IndexListExpr:
			base = v.X
		case *ast.StarExpr:
			base = v.X
		case *ast.SliceExpr:
			base = v.X
		default:
			return false
		}
		if isAstNamed(info.TypeOf(base)) {
			return true
		}
		e = base
	}
}

// isAstField reports whether sel selects a field declared in the ast package, including one promoted through a
// non-ast wrapper type.
func isAstField(sel *types.Selection) bool {
	if sel == nil || sel.Kind() != types.FieldVal {
		return false
	}
	pkg := sel.Obj().Pkg()
	return pkg != nil && pkg.Path() == astPkg
}

// isAstNamed reports whether t is a named type, or a pointer to one, declared in the ast package.
func isAstNamed(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if ptr, ok := t.(*types.Pointer); ok {
		t = types.Unalias(ptr.Elem())
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == astPkg
}
