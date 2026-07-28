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

// BoolValueNode holds a boolean value.
type BoolValueNode struct {
	value bool
	loc   diagnostics.Location
}

func NewBoolValueNode(value bool, loc diagnostics.Location) *BoolValueNode {
	return &BoolValueNode{value: value, loc: loc}
}

func (n *BoolValueNode) Kind() TomlType            { return TypeBoolean }
func (n *BoolValueNode) Loc() diagnostics.Location { return n.loc }
func (n *BoolValueNode) Value() bool               { return n.value }
func (n *BoolValueNode) NativeValue() any          { return n.value }
