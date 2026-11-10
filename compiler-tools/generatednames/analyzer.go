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

// Package generatednames enforces centralized compiler-generated function names.
package generatednames

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

const root = "github.com/ballerina-nutcracker/ballerina/"
const constantsPackage = root + "common/constants"

var Analyzer = &analysis.Analyzer{
	Name: "generatednames",
	Doc:  "require common/constants for dollar-prefixed generated function names",
	Run:  run,
}

type checker struct {
	pass      *analysis.Pass
	edges     map[any][]any
	arguments map[*types.Var][]ast.Expr
	seeds     []any
	reported  map[token.Pos]bool
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Path() == constantsPackage {
		return nil, nil
	}
	c := checker{
		pass:      pass,
		edges:     make(map[any][]any),
		arguments: make(map[*types.Var][]ast.Expr),
		reported:  make(map[token.Pos]bool),
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ValueSpec:
				if len(n.Names) == len(n.Values) {
					for i, name := range n.Names {
						c.link(pass.TypesInfo.Defs[name], n.Values[i])
					}
				}
			case *ast.AssignStmt:
				if len(n.Lhs) == len(n.Rhs) {
					for i, lhs := range n.Lhs {
						if id, ok := lhs.(*ast.Ident); ok {
							c.link(pass.TypesInfo.ObjectOf(id), n.Rhs[i])
						}
						if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Name" && isNamed(pass.TypesInfo.TypeOf(sel.X), root+"ast", "BLangFunction") {
							c.seeds = append(c.seeds, n.Rhs[i])
						}
					}
				}
			case *ast.FuncDecl:
				fn, _ := pass.TypesInfo.Defs[n.Name].(*types.Func)
				if fn != nil && n.Body != nil {
					ast.Inspect(n.Body, func(child ast.Node) bool {
						if _, nested := child.(*ast.FuncLit); nested {
							return false
						}
						if ret, ok := child.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
							c.link(fn, ret.Results[0])
						}
						return true
					})
				}
			case *ast.CallExpr:
				c.call(n)
			case *ast.CompositeLit:
				c.symbolLiteral(n)
				if isNamed(pass.TypesInfo.TypeOf(n), root+"ast", "InvokableData") {
					c.seedField(n, "Name")
				}
			}
			return true
		})
	}
	seen := make(map[trace]bool)
	for _, seed := range c.seeds {
		c.follow(trace{node: seed}, seen)
	}
	return nil, nil
}

type callContext struct {
	call   *ast.CallExpr
	fn     *types.Func
	parent *callContext
}

type trace struct {
	node any
	ctx  *callContext
}

func (c *checker) follow(start trace, seen map[trace]bool) {
	pass := c.pass
	pending := []trace{start}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		node, ctx := current.node, current.ctx
		if node == nil || seen[current] {
			continue
		}
		seen[current] = true
		push := func(nodes ...any) {
			for _, node := range nodes {
				pending = append(pending, trace{node: node, ctx: ctx})
			}
		}
		for _, edge := range c.edges[node] {
			push(edge)
		}
		if param, ok := node.(*types.Var); ok {
			bound := false
			if ctx != nil {
				params := ctx.fn.Type().(*types.Signature).Params()
				for i := 0; i < params.Len() && i < len(ctx.call.Args); i++ {
					if params.At(i) == param {
						pending = append(pending, trace{node: ctx.call.Args[i], ctx: ctx.parent})
						bound = true
						break
					}
				}
			}
			if !bound {
				for _, arg := range c.arguments[param] {
					push(arg)
				}
			}
		}
		switch node := node.(type) {
		case *ast.BasicLit:
			if strings.HasPrefix(stringValue(pass, node), "$") {
				c.report(node)
			}
		case *ast.Ident:
			push(pass.TypesInfo.ObjectOf(node))
		case *ast.ParenExpr:
			push(node.X)
		case *ast.BinaryExpr:
			if node.Op == token.ADD {
				push(node.X)
				if emptyString(pass, node.X) {
					push(node.Y)
				}
			}
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				push(node.X)
			}
		case *ast.CompositeLit:
			if isNamed(pass.TypesInfo.TypeOf(node), root+"ast", "BLangIdentifier") {
				for _, elt := range node.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Value" {
							push(kv.Value)
						}
					}
				}
			}
		case *ast.CallExpr:
			fn, _ := typeutil.Callee(pass.TypesInfo, node).(*types.Func)
			if identifierBuilder(fn) && len(node.Args) > 0 {
				push(node.Args[0])
				continue
			}
			if fn != nil && fn.Pkg() == pass.Pkg {
				recursive := false
				for parent := ctx; parent != nil; parent = parent.parent {
					if parent.fn == fn {
						recursive = true
						break
					}
				}
				if !recursive {
					pending = append(pending, trace{node: fn, ctx: &callContext{call: node, fn: fn, parent: ctx}})
				}
			}
			if fn != nil && fn.Pkg() != nil && fn.Pkg().Path() == "fmt" && fn.Name() == "Sprintf" && len(node.Args) > 0 {
				push(node.Args[0])
				format := stringValue(pass, node.Args[0])
				for i := 1; strings.HasPrefix(format, "%s") && i < len(node.Args); i++ {
					push(node.Args[i])
					if !emptyString(pass, node.Args[i]) {
						break
					}
					format = strings.TrimPrefix(format, "%s")
				}
			}
		}
	}
}

func emptyString(pass *analysis.Pass, expr ast.Expr) bool {
	value := pass.TypesInfo.Types[expr].Value
	return value != nil && value.Kind() == constant.String && constant.StringVal(value) == ""
}

func isNamed(ty types.Type, pkg, name string) bool {
	if ty == nil {
		return false
	}
	ty = types.Unalias(ty)
	if ptr, ok := ty.(*types.Pointer); ok {
		ty = types.Unalias(ptr.Elem())
	}
	named, ok := ty.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == pkg && named.Obj().Name() == name
}

func identifierBuilder(fn *types.Func) bool {
	if fn == nil || fn.Pkg() == nil {
		return false
	}
	return fn.Pkg().Path() == root+"desugar" && fn.Name() == "newIdentifier" ||
		fn.Pkg().Path() == root+"semantics/internal/types" && fn.Name() == "xmlStepIdentifier"
}

func (c *checker) seedField(lit *ast.CompositeLit, field string) {
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == field {
				c.seeds = append(c.seeds, kv.Value)
			}
		}
	}
}

func (c *checker) link(from, to any) {
	if from != nil {
		c.edges[from] = append(c.edges[from], to)
	}
}

func (c *checker) call(call *ast.CallExpr) {
	fn, _ := typeutil.Callee(c.pass.TypesInfo, call).(*types.Func)
	if fn == nil || fn.Pkg() == nil {
		return
	}
	if functionNameSink(fn) && len(call.Args) > 0 {
		c.seeds = append(c.seeds, call.Args[0])
	}
	if fn.Pkg() != c.pass.Pkg {
		return
	}
	params := fn.Type().(*types.Signature).Params()
	for i := 0; i < params.Len() && i < len(call.Args); i++ {
		param := params.At(i)
		c.arguments[param] = append(c.arguments[param], call.Args[i])
	}
}

func functionNameSink(fn *types.Func) bool {
	switch fn.Pkg().Path() {
	case root + "model":
		switch fn.Name() {
		case "NewFunctionSymbol", "NewDependentlyTypedFunctionSymbol", "NewResourceMethodSymbol", "NewDependentlyTypedResourceMethodSymbol", "newOpaqueFunctionSymbol", "newPrivateOpaqueFunctionSymbol", "newOpaqueFunctionSymbolWithVisibility":
			return true
		}
	case root + "context":
		return fn.Name() == "CreateFunctionSymbol"
	}
	return false
}

func (c *checker) symbolLiteral(lit *ast.CompositeLit) {
	ty := c.pass.TypesInfo.TypeOf(lit)
	named, ok := ty.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != root+"desugar" || named.Obj().Name() != "desugaredSymbol" {
		return
	}
	var name ast.Expr
	function := false
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if key.Name == "name" {
			name = kv.Value
		}
		if key.Name == "kind" {
			sel, ok := kv.Value.(*ast.SelectorExpr)
			if ok {
				obj := c.pass.TypesInfo.ObjectOf(sel.Sel)
				function = obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == root+"model" && obj.Name() == "SymbolKindFunction"
			}
		}
	}
	if function && name != nil {
		c.seeds = append(c.seeds, name)
	}
}

func stringValue(pass *analysis.Pass, expr ast.Expr) string {
	value := pass.TypesInfo.Types[expr].Value
	if value == nil || value.Kind() != constant.String {
		return ""
	}
	return constant.StringVal(value)
}

func (c *checker) report(lit *ast.BasicLit) {
	if c.reported[lit.Pos()] {
		return
	}
	c.reported[lit.Pos()] = true
	c.pass.Reportf(lit.Pos(), "generated function name must use a naming constant from common/constants")
}
