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
	"sort"
	"strings"
)

// TypeRegistry holds resolved type names and flattened declarations.
type TypeRegistry struct {
	pm *ParsedModel

	// Names assigned to generated anonymous types.
	typeNames map[*Type]string
	// Set of names already assigned.
	usedNames map[string]bool
	// Order in which generated types were discovered.
	genOrder []*Type
	// Resolved property sets for structures (flattened extends/mixins).
	structProps map[string][]Property
	// Union nullability wrappers already emitted to avoid duplicates.
	unionWrappers map[string]bool
}

// NewTypeRegistry resolves and indexes all named and anonymous types.
func NewTypeRegistry(pm *ParsedModel) (*TypeRegistry, error) {
	reg := &TypeRegistry{
		pm:            pm,
		typeNames:     make(map[*Type]string),
		usedNames:     make(map[string]bool),
		structProps:   make(map[string][]Property),
		unionWrappers: make(map[string]bool),
	}
	// Pre-populate named references.
	for name := range pm.References {
		reg.typeNames[&Type{Kind: KindReference, Name: name}] = GoName(name)
		reg.usedNames[GoName(name)] = true
	}
	// Alias names own anonymous underlying types.
	for _, a := range pm.Model.TypeAliases {
		if isAnonymousType(a.Type) {
			name := GoName(a.Name)
			reg.typeNames[a.Type] = name
			reg.usedNames[name] = true
			reg.genOrder = append(reg.genOrder, a.Type)
		}
	}
	// Flatten structure inheritance.
	for _, s := range pm.Structures {
		props, err := reg.flattenStructure(s)
		if err != nil {
			return nil, err
		}
		reg.structProps[s.Name] = props
	}
	// Name anonymous types reachable from named declarations and request/notification metadata.
	for _, s := range reg.sortedStructureSlice() {
		for _, p := range s.Properties {
			reg.nameType(p.Type, []string{s.Name, p.Name})
		}
	}
	for _, a := range pm.Model.TypeAliases {
		reg.nameType(a.Type, []string{a.Name})
	}
	for _, r := range reg.sortedRequestSlice() {
		reg.nameType(r.Params, []string{"Param", r.Method})
		reg.nameType(r.Result, []string{"Result", r.Method})
		reg.nameType(r.RegistrationOptions, []string{"RegOpt", r.Method})
		reg.nameType(r.ErrorData, []string{"ErrorData", r.Method})
	}
	for _, n := range reg.sortedNotificationSlice() {
		reg.nameType(n.Params, []string{"Param", n.Method})
		reg.nameType(n.RegistrationOptions, []string{"RegOpt", n.Method})
	}
	// Ensure all generated types have valid, non-colliding names.
	if err := reg.checkGeneratedNames(); err != nil {
		return nil, err
	}
	return reg, nil
}

func (reg *TypeRegistry) sortedStructureSlice() []*Structure {
	var list []*Structure
	for _, s := range reg.pm.Structures {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

func (reg *TypeRegistry) sortedRequestSlice() []*Request {
	var list []*Request
	list = append(list, reg.pm.Model.Requests...)
	sort.Slice(list, func(i, j int) bool { return list[i].Method < list[j].Method })
	return list
}

func (reg *TypeRegistry) sortedNotificationSlice() []*Notification {
	var list []*Notification
	list = append(list, reg.pm.Model.Notifications...)
	sort.Slice(list, func(i, j int) bool { return list[i].Method < list[j].Method })
	return list
}

// FlattenedProperties returns the flattened property list for a structure.
func (reg *TypeRegistry) FlattenedProperties(name string) []Property {
	return reg.structProps[name]
}

// GeneratedName returns the generated name for an anonymous type.
func (reg *TypeRegistry) GeneratedName(t *Type) (string, bool) {
	if t == nil {
		return "", false
	}
	// Normalize a reference to use the canonical reference object in our map.
	if t.Kind == KindReference {
		return GoName(t.Name), true
	}
	name, ok := reg.typeNames[t]
	return name, ok
}

// GeneratedTypes returns all anonymous types in discovery order.
func (reg *TypeRegistry) GeneratedTypes() []*Type {
	return reg.genOrder
}

// IsUnion reports whether t is a generated union (or nullable union).
func (reg *TypeRegistry) IsUnion(t *Type) bool {
	if t == nil {
		return false
	}
	if t.Kind == KindOr {
		return true
	}
	return false
}

// UnionItems returns the non-null variants of a union in metamodel order.
func (reg *TypeRegistry) UnionItems(t *Type) []*Type {
	return NullFreeItems(t)
}

func (reg *TypeRegistry) flattenStructure(s *Structure) ([]Property, error) {
	seen := make(map[string]Property)
	var order []string
	var collect func(name string) error
	collect = func(name string) error {
		st, ok := reg.pm.Structures[name]
		if !ok {
			return fmt.Errorf("structure %q extends/mixins unresolved reference %q", s.Name, name)
		}
		for _, e := range st.Extends {
			if e.Kind != KindReference {
				return fmt.Errorf("structure %q extends non-reference %q", name, TypeName(e))
			}
			if err := collect(e.Name); err != nil {
				return err
			}
		}
		for _, m := range st.Mixins {
			if m.Kind != KindReference {
				return fmt.Errorf("structure %q mixin non-reference %q", name, TypeName(m))
			}
			if err := collect(m.Name); err != nil {
				return err
			}
		}
		for _, p := range st.Properties {
			if existing, ok := seen[p.Name]; ok {
				if !compatibleProperties(existing, p) {
					return fmt.Errorf("structure %q inherits incompatible property %q from %q", s.Name, p.Name, name)
				}
				continue
			}
			seen[p.Name] = p
			order = append(order, p.Name)
		}
		return nil
	}
	if err := collect(s.Name); err != nil {
		return nil, err
	}
	var out []Property
	for _, name := range order {
		out = append(out, seen[name])
	}
	return out, nil
}

func compatibleProperties(a, b Property) bool {
	if a.Name != b.Name || a.Optional != b.Optional {
		return false
	}
	if TypeName(a.Type) == TypeName(b.Type) {
		return true
	}
	// A stringLiteral subtype discriminator is compatible with a base string.
	if a.Type.Kind == KindStringLiteral && b.Type.Kind == KindBase && b.Type.Name == "string" {
		return true
	}
	if b.Type.Kind == KindStringLiteral && a.Type.Kind == KindBase && a.Type.Name == "string" {
		return true
	}
	return false
}

// isAnonymousType reports whether t is an unnamed composite type.
func isAnonymousType(t *Type) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case KindAnd, KindOr, KindTuple, KindLiteral:
		return true
	}
	return false
}

func (reg *TypeRegistry) nameType(t *Type, path []string) string {
	if t == nil {
		return ""
	}
	if existing, ok := reg.typeNames[t]; ok {
		return existing
	}
	switch t.Kind {
	case KindBase:
		name := BaseTypeName(t.Name)
		reg.typeNames[t] = name
		return name
	case KindReference:
		name := GoName(t.Name)
		reg.typeNames[t] = name
		return name
	case KindStringLiteral, KindIntegerLiteral, KindBooleanLiteral:
		reg.typeNames[t] = "string"
		return "string"
	case KindArray:
		return "[]" + reg.nameType(t.Element, append(path, "Elem"))
	case KindMap:
		key := "string"
		if t.Key != nil {
			key = reg.nameType(t.Key, nil)
		}
		var val string
		if v, ok := t.Value.(*Type); ok {
			val = reg.nameType(v, append(path, "Value"))
		} else {
			val = "any"
		}
		name := "map[" + key + "]" + val
		reg.typeNames[t] = name
		return name
	case KindAnd:
		name := reg.allocateName(ContextName("And", path), t)
		for i, it := range t.Items {
			reg.nameType(it, append(path, fmt.Sprintf("Item%d", i)))
		}
		return name
	case KindOr:
		// If the union is just a nullable wrapper around a single non-null type,
		// we still generate a named union wrapper for the non-null shape so
		// that Optional/Nullable wrappers can refer to it unambiguously.
		name := reg.allocateName(ContextName("Or", path), t)
		for i, it := range t.Items {
			reg.nameType(it, append(path, fmt.Sprintf("Item%d", i)))
		}
		return name
	case KindTuple:
		name := reg.allocateName(ContextName("Tuple", path), t)
		for i, it := range t.Items {
			reg.nameType(it, append(path, fmt.Sprintf("Item%d", i)))
		}
		return name
	case KindLiteral:
		name := reg.allocateName(ContextName("Lit", path), t)
		// Literal properties are raw maps in the model; normalize for generation.
		_ = NormalizeLiteral(t)
		return name
	default:
		panic(fmt.Sprintf("unsupported type kind %q at line %d", t.Kind, t.Line))
	}
}

func (reg *TypeRegistry) allocateName(candidate string, t *Type) string {
	name := candidate
	for i := 2; ; i++ {
		if !reg.usedNames[name] {
			break
		}
		name = fmt.Sprintf("%s%d", candidate, i)
	}
	reg.typeNames[t] = name
	reg.usedNames[name] = true
	reg.genOrder = append(reg.genOrder, t)
	return name
}

// typeNamesByValue is no longer used; kept for compatibility.
func (reg *TypeRegistry) typeNamesByValue(name string) (*Type, bool) {
	for tp, nm := range reg.typeNames {
		if nm == name {
			return tp, true
		}
	}
	return nil, false
}

func (reg *TypeRegistry) checkGeneratedNames() error {
	// Every generated type must have a valid identifier name.
	for _, t := range reg.genOrder {
		name, ok := reg.typeNames[t]
		if !ok || name == "" {
			return fmt.Errorf("anonymous type at line %d has no generated name", t.Line)
		}
		if !isIdentifier(name) {
			return fmt.Errorf("generated name %q at line %d is not a valid Go identifier", name, t.Line)
		}
	}
	return nil
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !(r == '_' || unicodeIsLetter(r)) {
			return false
		}
		if !(r == '_' || unicodeIsLetter(r) || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func unicodeIsLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// SortedStructures returns structures sorted by name for deterministic output.
func (reg *TypeRegistry) SortedStructures() []*Structure {
	var list []*Structure
	for _, s := range reg.pm.Structures {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// SortedEnumerations returns enumerations sorted by name for deterministic output.
func (reg *TypeRegistry) SortedEnumerations() []*Enumeration {
	var list []*Enumeration
	for _, e := range reg.pm.Enumerations {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// SortedAliases returns aliases sorted by name for deterministic output.
func (reg *TypeRegistry) SortedAliases() []*TypeAlias {
	var list []*TypeAlias
	for _, a := range reg.pm.TypeAliases {
		list = append(list, a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// SortedRequests returns requests sorted by method for deterministic output.
func (reg *TypeRegistry) SortedRequests() []*Request {
	var list []*Request
	list = append(list, reg.pm.Model.Requests...)
	sort.Slice(list, func(i, j int) bool { return list[i].Method < list[j].Method })
	return list
}

// SortedNotifications returns notifications sorted by method for deterministic output.
func (reg *TypeRegistry) SortedNotifications() []*Notification {
	var list []*Notification
	list = append(list, reg.pm.Model.Notifications...)
	sort.Slice(list, func(i, j int) bool { return list[i].Method < list[j].Method })
	return list
}

// StringLiteralValue extracts the literal string value from a stringLiteral type.
func StringLiteralValue(t *Type) string {
	if t == nil || t.Kind != KindStringLiteral {
		return ""
	}
	v, _ := t.Value.(string)
	return v
}

// IntegerLiteralValue extracts the literal integer value from an integerLiteral type.
func IntegerLiteralValue(t *Type) int64 {
	if t == nil || t.Kind != KindIntegerLiteral {
		return 0
	}
	switch v := t.Value.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

// IsStringOnlyUnion reports whether a union consists solely of stringLiteral items.
func IsStringOnlyUnion(t *Type) bool {
	if t.Kind != KindOr {
		return false
	}
	for _, item := range t.Items {
		if item.Kind != KindStringLiteral {
			return false
		}
	}
	return len(t.Items) > 0
}

// IsLiteralType reports whether t is an anonymous literal structure.
func IsLiteralType(t *Type) bool {
	return t != nil && t.Kind == KindLiteral
}

// UnionKey returns a stable key for a union type's non-null items.
func UnionKey(t *Type) string {
	var parts []string
	for _, item := range NullFreeItems(t) {
		parts = append(parts, TypeName(item))
	}
	return strings.Join(parts, "|")
}
