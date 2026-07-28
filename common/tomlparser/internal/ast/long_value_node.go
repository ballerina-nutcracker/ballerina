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

// IntValueNode holds an integer value (decimal, hex, octal, or binary).
type IntValueNode struct {
	value int64
	loc   diagnostics.Location
}

func NewIntValueNode(value int64, loc diagnostics.Location) *IntValueNode {
	return &IntValueNode{value: value, loc: loc}
}

func (n *IntValueNode) Kind() TomlType            { return TypeInteger }
func (n *IntValueNode) Loc() diagnostics.Location { return n.loc }
func (n *IntValueNode) Value() int64              { return n.value }
func (n *IntValueNode) NativeValue() any          { return n.value }
