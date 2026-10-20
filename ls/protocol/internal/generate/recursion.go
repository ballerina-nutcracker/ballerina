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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package generate

import (
	"fmt"
)

// RecursionDetector stores recursion flags used to select pointer types.
type RecursionDetector struct {
	recursive map[string]bool
}

// NewRecursionDetector builds type dependencies and computes recursion flags.
func NewRecursionDetector(pm *ParsedModel, reg *TypeRegistry) *RecursionDetector {
	d := &RecursionDetector{recursive: make(map[string]bool)}
	deps := make(map[string]map[string]bool)
	for name := range pm.Structures {
		deps[GoName(name)] = make(map[string]bool)
		for _, p := range reg.FlattenedProperties(name) {
			d.collectDeps(p.Type, GoName(name), deps[GoName(name)])
		}
	}
	for _, a := range pm.TypeAliases {
		deps[GoName(a.Name)] = make(map[string]bool)
		d.collectDeps(a.Type, GoName(a.Name), deps[GoName(a.Name)])
	}
	for name := range deps {
		seen := make(map[string]bool)
		d.reachable(name, name, deps, seen, 0)
	}
	return d
}

// IsRecursive reports whether name is marked recursive.
func (d *RecursionDetector) IsRecursive(name string) bool {
	return d.recursive[name]
}

func (d *RecursionDetector) collectDeps(t *Type, from string, out map[string]bool) {
	if t == nil {
		return
	}
	switch t.Kind {
	case KindReference:
		out[GoName(t.Name)] = true
	case KindArray:
		d.collectDeps(t.Element, from, out)
	case KindMap:
		if t.Key != nil {
			d.collectDeps(t.Key, from, out)
		}
		if v, ok := t.Value.(*Type); ok {
			d.collectDeps(v, from, out)
		}
	case KindAnd, KindOr, KindTuple:
		for _, item := range t.Items {
			d.collectDeps(item, from, out)
		}
	case KindLiteral:
		for _, p := range LiteralProperties(t) {
			d.collectDeps(p.Type, from, out)
		}
	}
}

func (d *RecursionDetector) reachable(start, current string, deps map[string]map[string]bool, seen map[string]bool, depth int) {
	if depth > 200 {
		return
	}
	if seen[current] {
		return
	}
	seen[current] = true
	if current == start && depth > 0 {
		d.recursive[start] = true
		return
	}
	for next := range deps[current] {
		d.reachable(start, next, deps, seen, depth+1)
	}
}

// PointerFor reports whether a direct reference to name in a non-array/map
// value position should be emitted as a pointer.
func (d *RecursionDetector) PointerFor(name string) bool {
	return d.IsRecursive(name)
}

// ElementPointerFor reports whether an array/map element of name should be a
// pointer to break a recursive value path.
func (d *RecursionDetector) ElementPointerFor(name string) bool {
	return d.IsRecursive(name)
}

// ValidateNoDirectRecursion rejects properties that directly reference their containing structure.
func ValidateNoDirectRecursion(pm *ParsedModel, reg *TypeRegistry) error {
	for name := range pm.Structures {
		for _, p := range reg.FlattenedProperties(name) {
			if isDirectSelfReference(p.Type, name) {
				return fmt.Errorf("structure %q has required non-array non-optional self-reference %q; cannot represent in Go", name, p.Name)
			}
		}
	}
	return nil
}

func isDirectSelfReference(t *Type, self string) bool {
	if t == nil {
		return false
	}
	if t.Kind == KindReference && t.Name == self {
		return true
	}
	return false
}
