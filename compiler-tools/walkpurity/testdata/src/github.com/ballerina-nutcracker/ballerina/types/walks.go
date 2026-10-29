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

package types // want `ignored\.go is excluded by build constraints, so it can.t be checked`

import (
	"slices"
	"sort"
	"strings"
	"unsafe" // want `not allowed: unsafe`

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type collector struct {
	t    typeResolver
	deps map[int]bool
	refs []int
	node *ast.Node
}

func (c *collector) Visit(node ast.BLangNode) ast.Visitor {
	n, ok := node.(*ast.Node)
	if !ok {
		return c
	}
	if n.Symbol().IsEmpty() {
		return c
	}
	c.deps[c.t.unnarrowedSymbol(n.Ty)] = true
	c.refs = append(c.refs, c.t.symbolType(n.Ty))
	c.refs[0] = n.Ty
	c.node = n
	local := n.Ty
	local++
	sort.Ints(c.refs)
	return c
}

func (c *collector) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return c
}

func walkBody(c *collector, n *ast.Node) {
	ast.Walk(c, &n.Body)
	ast.WalkTypeData(c, &n.Data)
}

func readTy(n *ast.Node) int {
	return n.Ty
}

func conversions(n *ast.Node, ref model.SymbolRef) int {
	_ = ast.BLangNode(n)
	_ = len(n.Args)
	_ = ref.Index
	return int(n.Ty)
}

func recursiveA(n int) int {
	if n == 0 {
		return 0
	}
	return recursiveB(n - 1)
}

func recursiveB(n int) int {
	return recursiveA(n)
}

func writes(n *ast.Node, x int) {
	n.Ty = x                  // want `writes -> n\.Ty: write to ast node field`
	n.Args[0] = x             // want `writes -> n\.Args\[0\]: write to ast node field`
	*n = ast.Node{}           // want `writes -> \*n: write to ast node field`
	n.Count++                 // want `writes -> n\.Count: write to ast node field`
	for n.Ty = range n.Args { // want `writes -> n\.Ty: write to ast node field`
	}
}

func writesThroughBuiltins(n *ast.Node, src []int, local map[string]int) {
	copy(n.Args, src)         // want `writesThroughBuiltins -> copy: write to ast node field`
	clear(n.Args)             // want `writesThroughBuiltins -> clear: write to ast node field`
	delete(n.Attrs, "x")      // want `writesThroughBuiltins -> delete: write to ast node field`
	_ = append(n.Args[:0], 1) // want `writesThroughBuiltins -> append: write to ast node field`
	_ = append(n.Args, 1)
	copy(src, n.Args)
	clear(local)
}

func writesThroughWrapper(w *wrapper, x int) {
	w.Ty = x // want `writesThroughWrapper -> w\.Ty: write to ast node field`
}

type embeddingVisitor struct {
	embeddedBase
}

func (v embeddingVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return v
}

type splitVisitor struct {
	t typeResolver
}

func (v *splitVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return v
}

func setters(t typeResolver, ctx *context.CompilerContext, s sink, n *ast.Node) {
	ast.SetType(n, 1)                 // want `setters -> ast\.SetType: not allowed: ast\.SetType`
	n.SetSymbol(1)                    // want `setters -> n\.SetSymbol: not allowed: ast\.SetSymbol`
	n.AddArg(1)                       // want `setters -> n\.AddArg: not allowed: ast\.AddArg`
	setNodeType(t, n, 1)              // want `setters -> setNodeType: not allowed: setNodeType`
	setTypeData(t, n, ast.TypeData{}) // want `setters -> setTypeData: not allowed: setTypeData`
	setTypeDataType(t, &n.Data, 1)    // want `setters -> setTypeDataType: not allowed: setTypeDataType`
	ctx.SemanticError("x")            // want `setters -> ctx\.SemanticError: not allowed: context\.SemanticError`
	t.semanticError("x")              // want `setters -> t\.semanticError: not allowed: typeResolver\.semanticError`
	t.ensureResolved(1, 0)            // want `setters -> t\.ensureResolved: not allowed: typeResolver\.ensureResolved`
	s.record(1)                       // want `setters -> s\.record: not allowed: sink\.record`
}

func otherFunctions(xs []string, ref model.SymbolRef) {
	_ = pureHelper(1)       // want `otherFunctions -> pureHelper: not allowed: pureHelper`
	_ = strings.ToUpper("") // want `otherFunctions -> strings\.ToUpper: not allowed: strings\.ToUpper`
	sort.Strings(xs)        // want `otherFunctions -> sort\.Strings: not allowed: sort\.Strings`
	_ = ref.String()        // want `otherFunctions -> ref\.String: not allowed: model\.String`
}

func functionValues(c *collector, xs []int) {
	slices.SortFunc(xs, c.cmp) // want `functionValues -> slices\.SortFunc: not allowed: slices\.SortFunc` `functionValues -> c\.cmp: not allowed: cmp`
	f := pureHelper            // want `functionValues -> pureHelper: not allowed: pureHelper`
	_ = f
}

var hook func()

func unresolved(callback func(int)) {
	hook()      // want `unresolved -> hook: unresolved call`
	callback(1) // want `unresolved -> callback: unresolved call`
	local := func() {}
	local() // want `unresolved -> local: unresolved call`
}

var initialized = pureHelper(1) // want `initialized -> pureHelper: not allowed: pureHelper`

type localResolver interface {
	semanticError(message string)
}

func interfaceEscapes(t typeResolver, n *ast.Node, err error) {
	var local localResolver = t
	local.semanticError("x")                                                      // want `interfaceEscapes -> local\.semanticError: not allowed: localResolver\.semanticError`
	t.(interface{ ensureResolved(ref int, depth int) bool }).ensureResolved(1, 0) // want `not allowed: interface\{.*\}\.ensureResolved`
	any(n).(interface{ SetSymbol(int) }).SetSymbol(1)                             // want `not allowed: interface\{.*\}\.SetSymbol`
	_ = err.Error()                                                               // want `interfaceEscapes -> err\.Error: not allowed: Error`
}

func shadowedResolver(x any) {
	type typeResolver interface{ parent() int }
	_ = x.(typeResolver).parent() // want `shadowedResolver -> x\.\(typeResolver\)\.parent: not allowed: typeResolver\.parent`
	var r typeResolver = evil{}
	_ = r.parent() // want `shadowedResolver -> r\.parent: not allowed: typeResolver\.parent`
}

func rangeOverFuncs(h holder, seq func(func(int) bool)) {
	for range h.seq { // want `rangeOverFuncs -> h\.seq: unresolved call`
	}
	for range seq { // want `rangeOverFuncs -> seq: unresolved call`
	}
	for range sequence { // want `rangeOverFuncs -> sequence: not allowed: sequence`
	}
	for range 3 {
	}
}

func lineDirective(n *ast.Node) {
	disguised(n) // want `lineDirective -> disguised: not allowed: disguised`
}

func linked(n *ast.Node) // want `linked -> linked: function without a body`

func rangeOverTypeParam[F ~func(func(int) bool)](f F) {
	for range f { // want `rangeOverTypeParam -> f: unresolved call`
	}
}

func writesThroughUnsafe(n *ast.Node, p unsafe.Pointer) { // want `writesThroughUnsafe -> unsafe\.Pointer: not allowed: unsafe\.Pointer`
	_ = unsafe.Add(p, 8) // want `writesThroughUnsafe -> unsafe\.Add: not allowed: unsafe\.Add`
}

type promotedVisitor struct {
	foreignVisitor
}

func (promotedVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor { return nil }

func foreignVisitors(v ast.Visitor, n *ast.Node) {
	ast.Walk(promotedVisitor{}, n) // want `foreignVisitors -> promotedVisitor\{\}: visitor method Visit is not declared in walks.go`
	ast.Walk(v, n)                 // want `foreignVisitors -> v: visitor method Visit is not declared in walks.go` `foreignVisitors -> v: visitor method VisitTypeData is not declared in walks.go`
}
