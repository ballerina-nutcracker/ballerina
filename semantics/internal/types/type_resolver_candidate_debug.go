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

//go:build debug

package types

import (
	"reflect"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type nodeCopy struct {
	node ast.BLangNode
	copy reflect.Value
}

// nodeCopier records a struct copy of every node reachable from the roots. Child nodes are compared through their
// own copies, so a slice of children that was replaced by an equal slice is not a difference. The copy shares slice
// backing arrays, maps and non-node pointees with the live node, so a write through those is not detected; only
// field assignments are.
type nodeCopier struct {
	t      typeResolver
	copies []nodeCopy
	seen   map[ast.BLangNode]struct{}
}

func (c *nodeCopier) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if _, ok := c.seen[node]; ok {
		return nil
	}
	c.seen[node] = struct{}{}
	value := reflect.ValueOf(node)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		c.t.internalError("ephemeral resolution check requires non-nil pointer AST nodes", diagnostics.Location{})
		return nil
	}
	nodeCopy := nodeCopy{node: node, copy: reflect.New(value.Elem().Type()).Elem()}
	nodeCopy.copy.Set(value.Elem())
	c.copies = append(c.copies, nodeCopy)
	return c
}

func (c *nodeCopier) VisitTypeData(_ *ast.TypeData) ast.Visitor { return c }

func assertUnchanged(t typeResolver, roots []ast.BLangExpression) func() {
	copier := &nodeCopier{t: t, seen: make(map[ast.BLangNode]struct{})}
	for _, root := range roots {
		ast.Walk(copier, root)
	}
	return func() {
		for _, nodeCopy := range copier.copies {
			if !reflect.DeepEqual(reflect.ValueOf(nodeCopy.node).Elem().Interface(), nodeCopy.copy.Interface()) {
				t.internalError("ephemeral resolution mutated the AST", nodeCopy.node.GetPosition())
				return
			}
		}
	}
}
