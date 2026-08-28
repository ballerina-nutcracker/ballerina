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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Parse parses the metamodel JSON, validates it against the schema, and
// builds an indexed view of the model.
func Parse(modelJSON, schemaJSON []byte) (*ParsedModel, error) {
	if err := validateWithSchema(modelJSON, schemaJSON); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}
	withLines := addLineNumbers(modelJSON)
	var model Model
	if err := json.Unmarshal(withLines, &model); err != nil {
		return nil, fmt.Errorf("parse metamodel: %w", err)
	}
	pm := &ParsedModel{
		Model:           model,
		Structures:      make(map[string]*Structure),
		Enumerations:    make(map[string]*Enumeration),
		TypeAliases:     make(map[string]*TypeAlias),
		References:      make(map[string]*Type),
		RequestMethods:  make(map[string]*Request),
		NotifyMethods:   make(map[string]*Notification),
	}
	for _, s := range model.Structures {
		if _, dup := pm.Structures[s.Name]; dup {
			return nil, fmt.Errorf("duplicate structure %q at line %d", s.Name, s.Line)
		}
		pm.Structures[s.Name] = s
		pm.References[s.Name] = &Type{Kind: KindReference, Name: s.Name, Line: s.Line}
	}
	for _, e := range model.Enumerations {
		if _, dup := pm.Enumerations[e.Name]; dup {
			return nil, fmt.Errorf("duplicate enumeration %q at line %d", e.Name, e.Line)
		}
		pm.Enumerations[e.Name] = e
		pm.References[e.Name] = &Type{Kind: KindReference, Name: e.Name, Line: e.Line}
	}
	for _, a := range model.TypeAliases {
		if _, dup := pm.TypeAliases[a.Name]; dup {
			return nil, fmt.Errorf("duplicate type alias %q at line %d", a.Name, a.Line)
		}
		pm.TypeAliases[a.Name] = a
		pm.References[a.Name] = &Type{Kind: KindReference, Name: a.Name, Line: a.Line}
	}
	for _, r := range model.Requests {
		if _, dup := pm.RequestMethods[r.Method]; dup {
			return nil, fmt.Errorf("duplicate request method %q at line %d", r.Method, r.Line)
		}
		pm.RequestMethods[r.Method] = r
	}
	for _, n := range model.Notifications {
		if _, dup := pm.NotifyMethods[n.Method]; dup {
			return nil, fmt.Errorf("duplicate notification method %q at line %d", n.Method, n.Line)
		}
		pm.NotifyMethods[n.Method] = n
	}
	if err := pm.validateReferences(); err != nil {
		return nil, err
	}
	return pm, nil
}

// ParsedModel is an indexed view of the metamodel.
type ParsedModel struct {
	Model           Model
	Structures      map[string]*Structure
	Enumerations    map[string]*Enumeration
	TypeAliases     map[string]*TypeAlias
	References      map[string]*Type
	RequestMethods  map[string]*Request
	NotifyMethods   map[string]*Notification
}

func validateWithSchema(modelJSON, schemaJSON []byte) error {
	var schemaDoc any
	if err := json.Unmarshal(schemaJSON, &schemaDoc); err != nil {
		return fmt.Errorf("unmarshal schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(embeddedLoader{})
	if err := compiler.AddResource("schema.json", schemaDoc); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	var modelDoc any
	if err := json.Unmarshal(modelJSON, &modelDoc); err != nil {
		return fmt.Errorf("unmarshal model: %w", err)
	}
	if err := schema.Validate(modelDoc); err != nil {
		return err
	}
	return nil
}

// embeddedLoader blocks remote schema fetches. The metamodel schema is
// self-contained, so any remote reference is a mistake.
type embeddedLoader struct{}

func (embeddedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("remote schema fetch disabled: %s", url)
}

func (embeddedLoader) LoadFromPath(path string) (any, error) {
	return nil, fmt.Errorf("path schema fetch disabled: %s", path)
}

// addLineNumbers injects a synthetic "line" field into every object that
// follows a '{'+newline delimiter, matching gopls's approach.
func addLineNumbers(buf []byte) []byte {
	var ans []byte
	for linecnt, i := 1, 0; i < len(buf); i++ {
		ans = append(ans, buf[i])
		if buf[i] == '{' && i+1 < len(buf) && buf[i+1] == '\n' {
			ans = append(ans, fmt.Sprintf(`"line": %d, `, linecnt)...)
		}
		if buf[i] == '\n' {
			linecnt++
		}
	}
	return ans
}

func (pm *ParsedModel) validateReferences() error {
	check := func(t *Type, ctx string) error {
		return pm.walkType(t, func(inner *Type) error {
			switch inner.Kind {
			case KindReference:
				if _, ok := pm.References[inner.Name]; !ok {
					return fmt.Errorf("%s: unresolved reference %q at line %d", ctx, inner.Name, inner.Line)
				}
			case KindBase:
				if !BaseTypes[inner.Name] {
					return fmt.Errorf("%s: unknown base type %q at line %d", ctx, inner.Name, inner.Line)
				}
			case KindMap:
				if inner.Key == nil {
					return fmt.Errorf("%s: map missing key at line %d", ctx, inner.Line)
				}
				if inner.Key.Kind != KindBase && inner.Key.Kind != KindReference {
					return fmt.Errorf("%s: map key must be base or reference at line %d", ctx, inner.Line)
				}
			case KindAnd:
				if len(inner.Items) == 0 {
					return fmt.Errorf("%s: empty intersection at line %d", ctx, inner.Line)
				}
			case KindOr:
				if len(inner.Items) == 0 {
					return fmt.Errorf("%s: empty union at line %d", ctx, inner.Line)
				}
			}
			return nil
		})
	}
	for _, s := range pm.Structures {
		for _, e := range s.Extends {
			if err := check(e, fmt.Sprintf("structure %q extends", s.Name)); err != nil {
				return err
			}
		}
		for _, m := range s.Mixins {
			if err := check(m, fmt.Sprintf("structure %q mixin", s.Name)); err != nil {
				return err
			}
		}
		for _, p := range s.Properties {
			if err := check(p.Type, fmt.Sprintf("structure %q property %q", s.Name, p.Name)); err != nil {
				return err
			}
		}
	}
	for _, e := range pm.Enumerations {
		if err := check(e.Type, fmt.Sprintf("enumeration %q", e.Name)); err != nil {
			return err
		}
	}
	for _, a := range pm.TypeAliases {
		if err := check(a.Type, fmt.Sprintf("alias %q", a.Name)); err != nil {
			return err
		}
	}
	for _, r := range pm.Model.Requests {
		if r.Params != nil {
			if err := check(r.Params, fmt.Sprintf("request %q params", r.Method)); err != nil {
				return err
			}
		}
		if r.Result != nil {
			if err := check(r.Result, fmt.Sprintf("request %q result", r.Method)); err != nil {
				return err
			}
		}
		if r.ErrorData != nil {
			if err := check(r.ErrorData, fmt.Sprintf("request %q errorData", r.Method)); err != nil {
				return err
			}
		}
		if r.RegistrationOptions != nil {
			if err := check(r.RegistrationOptions, fmt.Sprintf("request %q registrationOptions", r.Method)); err != nil {
				return err
			}
		}
	}
	for _, n := range pm.Model.Notifications {
		if n.Params != nil {
			if err := check(n.Params, fmt.Sprintf("notification %q params", n.Method)); err != nil {
				return err
			}
		}
		if n.RegistrationOptions != nil {
			if err := check(n.RegistrationOptions, fmt.Sprintf("notification %q registrationOptions", n.Method)); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkType recursively visits t and its component types.
func (pm *ParsedModel) walkType(t *Type, f func(*Type) error) error {
	if t == nil {
		return nil
	}
	if err := f(t); err != nil {
		return err
	}
	switch t.Kind {
	case KindArray:
		return pm.walkType(t.Element, f)
	case KindMap:
		if err := pm.walkType(t.Key, f); err != nil {
			return err
		}
		if val, ok := t.Value.(*Type); ok {
			if err := pm.walkType(val, f); err != nil {
				return err
			}
		}
	case KindAnd, KindOr, KindTuple:
		for _, item := range t.Items {
			if err := pm.walkType(item, f); err != nil {
				return err
			}
		}
	}
	return nil
}

// TypeName returns a human-readable type description for error messages.
func TypeName(t *Type) string {
	if t == nil {
		return "<nil>"
	}
	switch t.Kind {
	case KindBase, KindReference, KindStringLiteral:
		return t.Name
	case KindIntegerLiteral, KindBooleanLiteral:
		return fmt.Sprintf("%v", t.Value)
	case KindArray:
		return TypeName(t.Element) + "[]"
	case KindMap:
		var key string
		if t.Key != nil {
			key = TypeName(t.Key)
		}
		var val string
		if v, ok := t.Value.(*Type); ok {
			val = TypeName(v)
		}
		return "map[" + key + "]" + val
	case KindOr:
		parts := make([]string, len(t.Items))
		for i, item := range t.Items {
			parts[i] = TypeName(item)
		}
		return strings.Join(parts, " | ")
	case KindAnd:
		parts := make([]string, len(t.Items))
		for i, item := range t.Items {
			parts[i] = TypeName(item)
		}
		return strings.Join(parts, " & ")
	case KindTuple:
		parts := make([]string, len(t.Items))
		for i, item := range t.Items {
			parts[i] = TypeName(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case KindLiteral:
		return "literal"
	default:
		return t.Kind
	}
}

// LiteralProperties extracts the property list from a literal type's Value.
func LiteralProperties(t *Type) []Property {
	if t == nil || t.Kind != KindLiteral {
		return nil
	}
	lit, ok := t.Value.(map[string]any)
	if !ok {
		return nil
	}
	propsData, ok := lit["properties"].([]any)
	if !ok {
		return nil
	}
	var props []Property
	for _, p := range propsData {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		doc, _ := m["documentation"].(string)
		since, _ := m["since"].(string)
		depr, _ := m["deprecated"].(string)
		opt, _ := m["optional"].(bool)
		prop := Property{Name: name, Documentation: doc, Since: since, Deprecated: depr, Optional: opt, Line: lineFromMap(m)}
		if typeData, ok := m["type"].(map[string]any); ok {
			prop.Type = decodeType(typeData)
		}
		props = append(props, prop)
	}
	return props
}

func lineFromMap(m map[string]any) int {
	if v, ok := m["line"].(float64); ok {
		return int(v)
	}
	return 0
}

func decodeType(m map[string]any) *Type {
	t := &Type{Line: lineFromMap(m)}
	if kind, ok := m["kind"].(string); ok {
		t.Kind = kind
	}
	if name, ok := m["name"].(string); ok {
		t.Name = name
	}
	if items, ok := m["items"].([]any); ok {
		for _, it := range items {
			if im, ok := it.(map[string]any); ok {
				t.Items = append(t.Items, decodeType(im))
			}
		}
	}
	if element, ok := m["element"].(map[string]any); ok {
		t.Element = decodeType(element)
	}
	if key, ok := m["key"].(map[string]any); ok {
		t.Key = decodeType(key)
	}
	if v, ok := m["value"]; ok {
		switch val := v.(type) {
		case map[string]any:
			t.Value = decodeType(val)
		default:
			t.Value = val
		}
	}
	return t
}

// IsNullableUnion reports whether the union type contains an explicit null item.
func IsNullableUnion(t *Type) bool {
	if t.Kind != KindOr {
		return false
	}
	for _, item := range t.Items {
		if item.Kind == KindBase && item.Name == "null" {
			return true
		}
	}
	return false
}

// NullFreeItems returns union items with the null variant removed.
func NullFreeItems(t *Type) []*Type {
	var out []*Type
	for _, item := range t.Items {
		if item.Kind == KindBase && item.Name == "null" {
			continue
		}
		out = append(out, item)
	}
	return out
}

// CountUnions returns the total number of union types reachable from t.
func CountUnions(t *Type) int {
	count := 0
	if t == nil {
		return 0
	}
	if t.Kind == KindOr {
		count = 1
	}
	switch t.Kind {
	case KindArray:
		count += CountUnions(t.Element)
	case KindMap:
		count += CountUnions(t.Key)
		if v, ok := t.Value.(*Type); ok {
			count += CountUnions(v)
		}
	case KindAnd, KindOr, KindTuple:
		for _, item := range t.Items {
			count += CountUnions(item)
		}
	}
	return count
}

// DeepCopy returns a deep copy of t.
func DeepCopy(t *Type) *Type {
	if t == nil {
		return nil
	}
	cp := &Type{Kind: t.Kind, Name: t.Name, Value: t.Value, Line: t.Line}
	if t.Element != nil {
		cp.Element = DeepCopy(t.Element)
	}
	if t.Key != nil {
		cp.Key = DeepCopy(t.Key)
	}
	for _, item := range t.Items {
		cp.Items = append(cp.Items, DeepCopy(item))
	}
	if v, ok := t.Value.(*Type); ok {
		cp.Value = DeepCopy(v)
	}
	return cp
}

// NormalizeLiteral converts a literal type's Value from raw map into a Type.
func NormalizeLiteral(t *Type) *Type {
	if t == nil || t.Kind != KindLiteral {
		return t
	}
	props := LiteralProperties(t)
	out := DeepCopy(t)
	out.Value = props
	return out
}

// NeedPointer reports whether a Go type string is a pointer-like reference type.
func NeedPointer(goType string) bool {
	if goType == "" {
		return false
	}
	if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") || strings.HasPrefix(goType, "*") {
		return false
	}
	switch goType {
	case "bool", "string", "int", "int32", "int64", "uint32", "uint64", "float64", "any":
		return true
	}
	return false
}
