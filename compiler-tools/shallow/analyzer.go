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

// Package shallow reports package-level functions whose body is a single call
// without nested calls, function literals or composite literals. It also reports
// structs that only embed one struct and declare no methods, and interfaces that
// only embed one interface. A doc-comment //shallow:allow <reason> exempts a declaration.
package shallow

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports shallow package-level functions, structs and interfaces; see the package doc.
var Analyzer = &analysis.Analyzer{
	Name: "shallow",
	Doc:  "reports functions that only call another function, structs that only embed a struct and interfaces that only embed an interface",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	files := make([]*ast.File, 0, len(pass.Files))
	for _, file := range pass.Files {
		if !strings.HasSuffix(pass.Fset.File(file.Pos()).Name(), "_test.go") {
			files = append(files, file)
		}
	}
	methods, err := ownMethods(pass, files)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		checkFile(pass, file, methods)
	}
	return nil, nil
}

func ownMethods(pass *analysis.Pass, files []*ast.File) (map[string]bool, error) {
	methods := make(map[string]bool)
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			t := types.Unalias(pass.TypesInfo.Defs[fn.Name].(*types.Func).Signature().Recv().Type())
			if ptr, ok := t.(*types.Pointer); ok {
				t = ptr.Elem()
			}
			if named, ok := types.Unalias(t).(*types.Named); ok {
				methods[named.Origin().Obj().Name()] = true
			}
		}
	}
	ignored := make([]*ast.File, 0, len(pass.IgnoredFiles))
	variantTypes := make(map[string]bool)
	for _, path := range pass.IgnoredFiles {
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := pass.ReadFile(path)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(pass.Fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		if file.Name.Name != pass.Pkg.Name() {
			continue
		}
		ignored = append(ignored, file)
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
				for _, spec := range gen.Specs {
					variantTypes[spec.(*ast.TypeSpec).Name.Name] = true
				}
			}
		}
	}
	for _, file := range ignored {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
				name := receiverTypeName(fn.Recv.List[0].Type)
				if name != "" && !variantTypes[name] {
					methods[name] = true
				}
			}
		}
	}
	return methods, nil
}

func receiverTypeName(expr ast.Expr) string {
	expr = ast.Unparen(expr)
	if ptr, ok := expr.(*ast.StarExpr); ok {
		expr = ast.Unparen(ptr.X)
	}
	switch indexed := expr.(type) {
	case *ast.IndexExpr:
		expr = ast.Unparen(indexed.X)
	case *ast.IndexListExpr:
		expr = ast.Unparen(indexed.X)
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func checkFile(pass *analysis.Pass, file *ast.File, methods map[string]bool) {
	directives := make(map[*ast.Comment]string)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if reason, ok := parseDirective(comment.Text); ok {
				directives[comment] = reason
			}
		}
	}
	check := func(name *ast.Ident, doc *ast.CommentGroup, message string) {
		var first *ast.Comment
		if doc != nil {
			for _, comment := range doc.List {
				reason, ok := directives[comment]
				if !ok {
					continue
				}
				delete(directives, comment)
				if first != nil {
					pass.Reportf(comment.Pos(), "duplicate //shallow:allow directive")
					continue
				}
				first = comment
				if reason == "" {
					pass.Reportf(comment.Pos(), "//shallow:allow requires a reason")
				} else if message == "" {
					pass.Reportf(comment.Pos(), "%s is not shallow; remove the unnecessary //shallow:allow directive", name.Name)
				}
			}
		}
		if first == nil && message != "" {
			pass.Reportf(name.Pos(), "%s", message)
		}
	}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			message := ""
			if call := shallowCall(pass, decl); call != nil {
				message = "function " + decl.Name.Name + " only calls " + types.ExprString(call.Fun) + "; inline it into its callers or mark it //shallow:allow <reason>"
			}
			check(decl.Name, decl.Doc, message)
		case *ast.GenDecl:
			if decl.Tok != token.TYPE {
				continue
			}
			for _, s := range decl.Specs {
				spec := s.(*ast.TypeSpec)
				doc := spec.Doc
				if doc == nil && !decl.Lparen.IsValid() {
					doc = decl.Doc
				}
				check(spec.Name, doc, shallowType(pass, spec, methods))
			}
		}
	}
	for comment := range directives {
		pass.Reportf(comment.Pos(), "//shallow:allow must be in the doc comment of a function or type declaration")
	}
}

func parseDirective(text string) (string, bool) {
	const prefix = "//shallow:allow"
	rest, ok := strings.CutPrefix(text, prefix)
	if !ok || (rest != "" && rest[0] != ' ' && rest[0] != '\t') {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

func shallowCall(pass *analysis.Pass, fn *ast.FuncDecl) *ast.CallExpr {
	if fn.Recv != nil || fn.Body == nil || fn.Name.Name == "_" || fn.Name.Name == "init" ||
		(fn.Name.Name == "main" && pass.Pkg.Name() == "main") || len(fn.Body.List) != 1 {
		return nil
	}
	var expr ast.Expr
	switch stmt := fn.Body.List[0].(type) {
	case *ast.ExprStmt:
		expr = stmt.X
	case *ast.ReturnStmt:
		if len(stmt.Results) != 1 {
			return nil
		}
		expr = stmt.Results[0]
	default:
		return nil
	}
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok {
		return nil
	}
	tv := pass.TypesInfo.Types[call.Fun]
	if tv.IsType() || tv.IsBuiltin() {
		return nil
	}
	hasNestedEvaluation := false
	ast.Inspect(call, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.CallExpr:
			if node != call {
				hasNestedEvaluation = true
			}
		case *ast.FuncLit, *ast.CompositeLit:
			hasNestedEvaluation = true
		}
		return !hasNestedEvaluation
	})
	if hasNestedEvaluation {
		return nil
	}
	return call
}

func shallowType(pass *analysis.Pass, spec *ast.TypeSpec, methods map[string]bool) string {
	if spec.Assign.IsValid() {
		return ""
	}
	var kind string
	var embedded ast.Expr
	switch t := spec.Type.(type) {
	case *ast.StructType:
		if len(t.Fields.List) != 1 || len(t.Fields.List[0].Names) != 0 || methods[spec.Name.Name] {
			return ""
		}
		embedded = t.Fields.List[0].Type
		ty := pass.TypesInfo.TypeOf(embedded)
		if ptr, ok := ty.(*types.Pointer); ok {
			ty = ptr.Elem()
		}
		if _, ok := ty.Underlying().(*types.Struct); !ok {
			return ""
		}
		kind = "struct"
	case *ast.InterfaceType:
		if len(t.Methods.List) != 1 || len(t.Methods.List[0].Names) != 0 {
			return ""
		}
		embedded = t.Methods.List[0].Type
		switch embedded.(type) {
		case *ast.BinaryExpr, *ast.UnaryExpr:
			return ""
		}
		if _, ok := pass.TypesInfo.TypeOf(embedded).Underlying().(*types.Interface); !ok {
			return ""
		}
		kind = "interface"
	default:
		return ""
	}
	name := types.ExprString(embedded)
	return kind + " " + spec.Name.Name + " only embeds " + name + "; use " + name + " directly or mark it //shallow:allow <reason>"
}
