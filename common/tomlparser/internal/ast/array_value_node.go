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

// ArrayValueNode holds a TOML array of values.
type ArrayValueNode struct {
	elements []ValueNode
	loc      diagnostics.Location
}

func NewArrayValueNode(elements []ValueNode, loc diagnostics.Location) *ArrayValueNode {
	return &ArrayValueNode{elements: elements, loc: loc}
}

func (n *ArrayValueNode) Kind() TomlType            { return TypeArray }
func (n *ArrayValueNode) Loc() diagnostics.Location { return n.loc }
func (n *ArrayValueNode) Elements() []ValueNode     { return n.elements }

func (n *ArrayValueNode) NativeValue() any {
	result := make([]any, len(n.elements))
	for i, e := range n.elements {
		result[i] = e.NativeValue()
	}
	return result
}
