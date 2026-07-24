/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package ast

import "ballerina/tools/diagnostics"

// TableArrayNode represents [[array-of-tables]] entries.
// Each [[array-of-tables]] occurrence appends a new TableNode child.
type TableArrayNode struct {
	key      string
	children []*TableNode
	loc      diagnostics.Location
}

func NewTableArrayNode(key string, loc diagnostics.Location) *TableArrayNode {
	return &TableArrayNode{
		key: key,
		loc: loc,
	}
}

func (n *TableArrayNode) Kind() TomlType            { return TypeTableArray }
func (n *TableArrayNode) Loc() diagnostics.Location { return n.loc }
func (n *TableArrayNode) KeyName() string           { return n.key }
func (n *TableArrayNode) Children() []*TableNode    { return n.children }

// AddChild appends a table to this array.
func (n *TableArrayNode) AddChild(t *TableNode) {
	n.children = append(n.children, t)
}

// ToList converts all children to []any (each element is map[string]any).
func (n *TableArrayNode) ToList() []any {
	result := make([]any, len(n.children))
	for i, child := range n.children {
		result[i] = child.ToMap()
	}
	return result
}
