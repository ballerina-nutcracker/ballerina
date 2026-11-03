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

import (
	"sort"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type symbolDepCollector struct {
	t       typeResolver
	nodeSet map[model.SymbolRef]int
	deps    map[int]struct{}
}

func (c *symbolDepCollector) depends(ref model.SymbolRef) {
	unnarrowed := c.t.unnarrowedSymbol(ref)
	if idx, ok := c.nodeSet[unnarrowed]; ok {
		c.deps[idx] = struct{}{}
	}
}

func (c *symbolDepCollector) Visit(node ast.BLangNode) ast.Visitor {
	switch n := node.(type) {
	case *ast.BLangVarRef:
		c.depends(n.Symbol())
	case *ast.BLangConstRef:
		c.depends(n.Symbol())
	}
	return c
}

func (c *symbolDepCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

// symbolDeps returns the sorted indices in nodeSet of the symbols expr refers to.
func symbolDeps(t typeResolver, nodeSet map[model.SymbolRef]int, expr ast.BLangNode) []int {
	c := &symbolDepCollector{t: t, nodeSet: nodeSet, deps: make(map[int]struct{})}
	ast.Walk(c, expr)
	var deps []int
	for d := range c.deps {
		deps = append(deps, d)
	}
	sort.Ints(deps)
	return deps
}

type trialDependencyRefCollector struct {
	refs []model.SymbolRef
}

func (c *trialDependencyRefCollector) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	switch ref := node.(type) {
	case *ast.BLangVarRef, *ast.BLangConstRef, *ast.BLangUserDefinedType:
		if symbol := ref.(ast.BNodeWithSymbol).Symbol(); !symbol.IsEmpty() {
			c.refs = append(c.refs, symbol)
		}
	case *ast.BLangInvocation:
		// A method call still carries a deferred method symbol; its receiver's type is reached through other refs.
		if symbol, ok := ref.RawSymbol.(*model.SymbolRef); ok && symbol != nil && !symbol.IsEmpty() {
			c.refs = append(c.refs, *symbol)
		}
	case *ast.BLangObjectType:
		// Object type inclusions are held as symbols, not as walked type descriptor nodes.
		c.refs = append(c.refs, ref.Inclusions...)
	}
	return c
}

func (c *trialDependencyRefCollector) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

// collectTrialDependencyRefs returns the symbols roots refer to or call, in pre-order, keeping duplicates.
func collectTrialDependencyRefs(roots []ast.BLangExpression) []model.SymbolRef {
	c := &trialDependencyRefCollector{}
	for _, root := range roots {
		ast.Walk(c, root)
	}
	return c.refs
}
