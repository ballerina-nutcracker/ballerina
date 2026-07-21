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

// TomlType enumerates the semantic AST node kinds.
type TomlType int

const (
	TypeTable       TomlType = iota // [table] or root document
	TypeTableArray                  // [[table-array]]
	TypeKeyValue                    // key = value
	TypeString                      // "..." or '...'
	TypeInteger                     // integer literal
	TypeFloat                       // float literal
	TypeBoolean                     // true / false
	TypeArray                       // [ val, val ]
	TypeInlineTable                 // { k = v }
)

// Node is the base interface for all semantic AST nodes.
type Node interface {
	Kind() TomlType
	Loc() diagnostics.Location
}

// TopLevelNode can appear as an entry in a table's entries map.
type TopLevelNode interface {
	Node
	// KeyName returns the last key segment used to register this node in its parent table.
	KeyName() string
}
