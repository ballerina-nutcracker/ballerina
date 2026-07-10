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

// TableNode represents a [table] section or the implicit root document table.
type TableNode struct {
	key       string // last key segment; empty for root table
	entries   map[string]TopLevelNode
	order     []string // insertion order (Go maps are unordered)
	generated bool     // true when implicitly created for a dotted key
	loc       diagnostics.Location
}

// NewTableNode creates an explicit (non-generated) table node.
func NewTableNode(key string, loc diagnostics.Location) *TableNode {
	return &TableNode{
		key:     key,
		entries: make(map[string]TopLevelNode),
		loc:     loc,
	}
}

// NewGeneratedTableNode creates an implicit intermediate table node.
func NewGeneratedTableNode(key string, loc diagnostics.Location) *TableNode {
	return &TableNode{
		key:       key,
		entries:   make(map[string]TopLevelNode),
		generated: true,
		loc:       loc,
	}
}

func (t *TableNode) Kind() TomlType            { return TypeTable }
func (t *TableNode) Loc() diagnostics.Location { return t.loc }
func (t *TableNode) KeyName() string           { return t.key }
func (t *TableNode) Generated() bool           { return t.generated }

// Entries returns the map of child nodes.
func (t *TableNode) Entries() map[string]TopLevelNode { return t.entries }

// AddEntry inserts a child node.  Duplicate-key detection is done in the parser.
func (t *TableNode) AddEntry(key string, node TopLevelNode) {
	if _, exists := t.entries[key]; !exists {
		t.order = append(t.order, key)
	}
	t.entries[key] = node
}

// ReplaceGeneratedTable replaces a generated (implicit) table with an explicit one.
func (t *TableNode) ReplaceGeneratedTable(newTable *TableNode) {
	key := newTable.key
	existing, ok := t.entries[key]
	if !ok {
		return
	}
	existingTable, ok := existing.(*TableNode)
	if !ok || !existingTable.generated {
		return
	}
	// Merge generated entries into the replacement table.
	// Only carry over keys that the explicit table has not already defined;
	// an explicit entry must never be overwritten by a generated (implicit) one.
	for _, k := range existingTable.order {
		if _, exists := newTable.entries[k]; !exists {
			newTable.order = append(newTable.order, k)
			newTable.entries[k] = existingTable.entries[k]
		}
	}
	t.entries[key] = newTable
}

// ToMap converts the table to a map[string]any, recursively.
// This is what the transformer produces for the public Toml struct.
func (t *TableNode) ToMap() map[string]any {
	m := make(map[string]any, len(t.entries))
	for _, k := range t.order {
		v := t.entries[k]
		switch tv := v.(type) {
		case *KeyValueNode:
			m[k] = tv.NativeValue()
		case *TableNode:
			m[k] = tv.ToMap()
		case *TableArrayNode:
			m[k] = tv.ToList()
		default:
			m[k] = nil
		}
	}
	return m
}
