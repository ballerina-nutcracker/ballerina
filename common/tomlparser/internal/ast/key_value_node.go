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

// KeyValueNode represents a key = value pair.
// When the TOML source uses a dotted key (a.b.c = val), keys holds all
// segments; KeyName() returns only the last segment.
type KeyValueNode struct {
	keys  []string // all key segments (len >= 1)
	value ValueNode
	loc   diagnostics.Location
}

// NewKeyValueNode creates a single-key (non-dotted) KeyValueNode.
func NewKeyValueNode(key string, value ValueNode, loc diagnostics.Location) *KeyValueNode {
	return &KeyValueNode{keys: []string{key}, value: value, loc: loc}
}

// NewKeyValueNodeWithPath creates a dotted-key KeyValueNode.
// Used by the parser when it encounters a.b.c = val.
func NewKeyValueNodeWithPath(keys []string, value ValueNode, loc diagnostics.Location) *KeyValueNode {
	return &KeyValueNode{keys: keys, value: value, loc: loc}
}

func (n *KeyValueNode) Kind() TomlType            { return TypeKeyValue }
func (n *KeyValueNode) Loc() diagnostics.Location { return n.loc }
func (n *KeyValueNode) KeyName() string           { return n.keys[len(n.keys)-1] }
func (n *KeyValueNode) Keys() []string            { return n.keys }
func (n *KeyValueNode) Value() ValueNode          { return n.value }
func (n *KeyValueNode) NativeValue() any          { return n.value.NativeValue() }
