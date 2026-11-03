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

package types

import "github.com/ballerina-nutcracker/ballerina/ast"

type typeResolver interface {
	symbolType(ref int) int
	unnarrowedSymbol(ref int) int
	semanticError(message string)
	ensureResolved(ref int, depth int) bool
}

type sink interface {
	record(x int)
}

func setNodeType(t typeResolver, n *ast.Node, ty int) {
	n.Ty = ty
}

func setTypeData(t typeResolver, n *ast.Node, data ast.TypeData) {
	n.Data = data
}

func setTypeDataType(t typeResolver, data *ast.TypeData, ty int) {
	data.Ty = ty
}

func pureHelper(n int) int {
	return n
}

func (c *collector) cmp(a, b int) int {
	c.t.semanticError("x")
	return a - b
}

func walkOutside(v ast.Visitor, n *ast.Node) {
	ast.Walk(v, n)           // want `AST walks must live in walks.go`
	ast.WalkTypeData(v, nil) // want `AST walks must live in walks.go`
}

func useWithoutCall(v ast.Visitor) {
	f := ast.Walk // want `AST walks must live in walks.go`
	f(v, nil)
}

type valueVisitor struct{} // want `ast.Visitor implementations must live in walks.go`

func (valueVisitor) Visit(node ast.BLangNode) ast.Visitor { return nil } // want `ast.Visitor methods must live in walks.go`

func (valueVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor { return nil } // want `ast.Visitor methods must live in walks.go`

type visitorLike interface {
	ast.Visitor
	extra()
}

type pointerVisitor struct{} // want `ast.Visitor implementations must live in walks.go`

func (*pointerVisitor) Visit(node ast.BLangNode) ast.Visitor { return nil } // want `ast.Visitor methods must live in walks.go`

func (*pointerVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor { return nil } // want `ast.Visitor methods must live in walks.go`

type wrapper struct {
	*ast.Node
}

type embeddedBase struct {
	t typeResolver
}

func (b embeddedBase) Visit(node ast.BLangNode) ast.Visitor { // want `ast.Visitor methods must live in walks.go`
	b.t.semanticError("x")
	return nil
}

func (v *splitVisitor) Visit(node ast.BLangNode) ast.Visitor { // want `ast.Visitor methods must live in walks.go`
	v.t.semanticError("x")
	return v
}

type evil struct {
	n *ast.Node
}

func (e evil) parent() int {
	e.n.Ty = 1
	return 0
}

type holder struct {
	seq func(func(int) bool)
}

func sequence(yield func(int) bool) {}

type foreignVisitor struct{}

func (foreignVisitor) Visit(node ast.BLangNode) ast.Visitor { return nil } // want `ast.Visitor methods must live in walks.go`
