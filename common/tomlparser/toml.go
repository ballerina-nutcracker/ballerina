// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tomlparser

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"strings"

	"ballerina/common/tomlparser/internal/parser"
	"ballerina/tools/diagnostics"
)

type Toml struct {
	rootNode    map[string]any
	diagnostics []Diagnostic
	content     string
}

type Diagnostic struct {
	Message  string
	Severity diagnostics.DiagnosticSeverity
	Location *Location
}

type Location struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
}

func readFile(fsys fs.FS, path string) (string, error) {
	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func readFromReader(reader io.Reader) (string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func Read(fsys fs.FS, path string) (*Toml, error) {
	content, err := readFile(fsys, path)
	if err != nil {
		return nil, err
	}
	return ReadString(content)
}

func ReadWithSchema(fsys fs.FS, path string, schema Schema) (*Toml, error) {
	content, err := readFile(fsys, path)
	if err != nil {
		return nil, err
	}
	return ReadStringWithSchema(content, schema)
}

func ReadStream(reader io.Reader) (*Toml, error) {
	content, err := readFromReader(reader)
	if err != nil {
		return nil, err
	}
	return ReadString(content)
}

func ReadStreamWithSchema(reader io.Reader, schema Schema) (*Toml, error) {
	content, err := readFromReader(reader)
	if err != nil {
		return nil, err
	}
	return ReadStringWithSchema(content, schema)
}

// ReadString parses TOML content using the native Go parser.
func ReadString(content string) (*Toml, error) {
	p := parser.NewParser(content)
	rootTable, parseErrors := p.Parse()

	t := &Toml{
		rootNode:    rootTable.ToMap(),
		diagnostics: convertParseErrors(parseErrors),
		content:     content,
	}

	if len(parseErrors) > 0 {
		return t, fmt.Errorf("TOML parse error: %s", parseErrors[0].Message)
	}
	return t, nil
}

func ReadStringWithSchema(content string, schema Schema) (*Toml, error) {
	t, err := ReadString(content)
	if err != nil {
		return t, err
	}

	validator := NewValidator(schema)
	validationErr := validator.Validate(t)
	if validationErr != nil {
		return t, validationErr
	}

	return t, nil
}

func (t *Toml) Validate(schema Schema) error {
	validator := NewValidator(schema)
	return validator.Validate(t)
}

func (t *Toml) Get(dottedKey string) (any, bool) {
	keys := splitDottedKey(dottedKey)
	return t.getValueByPath(keys)
}

func (t *Toml) GetString(dottedKey string) (string, bool) {
	val, ok := t.Get(dottedKey)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

func (t *Toml) GetInt(dottedKey string) (int64, bool) {
	val, ok := t.Get(dottedKey)
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}

func (t *Toml) GetFloat(dottedKey string) (float64, bool) {
	val, ok := t.Get(dottedKey)
	if !ok {
		return 0, false
	}
	f, ok := val.(float64)
	return f, ok
}

func (t *Toml) GetBool(dottedKey string) (bool, bool) {
	val, ok := t.Get(dottedKey)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

func (t *Toml) GetArray(dottedKey string) ([]any, bool) {
	val, ok := t.Get(dottedKey)
	if !ok {
		return nil, false
	}
	arr, ok := val.([]any)
	return arr, ok
}

func (t *Toml) GetTable(dottedKey string) (*Toml, bool) {
	val, ok := t.Get(dottedKey)
	if !ok {
		return nil, false
	}

	table, ok := val.(map[string]any)
	if !ok {
		return nil, false
	}

	return &Toml{
		rootNode:    table,
		diagnostics: nil,
		content:     "",
	}, true
}

func (t *Toml) GetTables(dottedKey string) ([]*Toml, bool) {
	val, ok := t.Get(dottedKey)
	if !ok {
		return nil, false
	}

	arr, ok := val.([]any)
	if !ok {
		if tableArr, ok := val.([]map[string]any); ok {
			result := make([]*Toml, len(tableArr))
			for i, table := range tableArr {
				result[i] = &Toml{
					rootNode:    table,
					diagnostics: nil,
					content:     "",
				}
			}
			return result, true
		}
		return nil, false
	}

	result := make([]*Toml, 0)
	for _, item := range arr {
		if table, ok := item.(map[string]any); ok {
			result = append(result, &Toml{
				rootNode:    table,
				diagnostics: nil,
				content:     "",
			})
		}
	}

	if len(result) == 0 {
		return nil, false
	}

	return result, true
}

func (t *Toml) Diagnostics() []Diagnostic {
	return t.diagnostics
}

func (t *Toml) ToMap() map[string]any {
	return t.rootNode
}

// To unmarshals the TOML document into target using a JSON bridge.
// The target struct may use `toml:"field_name"` tags; these are translated to
// JSON tags for the round-trip.  For simple cases (no custom tags) field names
// are matched case-insensitively by encoding/json.
func (t *Toml) To(target any) {
	// Build a JSON-tagged intermediate map that maps TOML keys to struct field
	// names or toml: tags, so json.Unmarshal can resolve them correctly.
	remapped := remapForTarget(t.rootNode, reflect.TypeOf(target))
	jsonBytes, err := json.Marshal(remapped)
	if err != nil {
		t.diagnostics = append(t.diagnostics, Diagnostic{
			Message:  fmt.Sprintf("internal serialisation error: %v", err),
			Severity: diagnostics.Error,
		})
		return
	}
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		t.diagnostics = append(t.diagnostics, Diagnostic{
			Message:  fmt.Sprintf("type mismatch: %v", err),
			Severity: diagnostics.Error,
		})
	}
}

// remapForTarget rewrites the keys of m so they match the field names (or
// json: tags) of the struct pointed to by targetType.  This is needed because
// TOML keys may use snake_case while Go struct fields use PascalCase.
// toml:"key" tags are honoured to locate the right field; the map is then
// re-keyed using the field name (or json: tag if present) so that
// json.Unmarshal can resolve it.
func remapForTarget(m map[string]any, targetType reflect.Type) map[string]any {
	// Dereference pointer types (target is typically a *Struct).
	for targetType != nil && targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}
	if targetType == nil || targetType.Kind() != reflect.Struct {
		return normaliseMapKeys(m)
	}

	// Build a lookup: toml-key → field descriptor.
	// excluded holds the lowercased field names of toml:"-" fields so their
	// TOML keys are dropped before reaching json.Unmarshal.
	type fieldDesc struct {
		jsonName  string
		fieldType reflect.Type
	}
	lookup := make(map[string]fieldDesc, targetType.NumField())
	excluded := make(map[string]bool)
	for i := 0; i < targetType.NumField(); i++ {
		f := targetType.Field(i)
		if tag, ok := f.Tag.Lookup("toml"); ok && tag == "-" {
			excluded[strings.ToLower(f.Name)] = true
			continue
		}
		tomlKey := strings.ToLower(f.Name)
		if tag, ok := f.Tag.Lookup("toml"); ok && tag != "" {
			tomlKey = strings.Split(tag, ",")[0]
		}
		jsonName := f.Name
		if tag, ok := f.Tag.Lookup("json"); ok && tag != "" && tag != "-" {
			jsonName = strings.Split(tag, ",")[0]
		}
		lookup[tomlKey] = fieldDesc{jsonName: jsonName, fieldType: f.Type}
		// Also store by lowercased field name for case-insensitive matching,
		// but only if that slot is not already claimed by a tag from an earlier
		// field — otherwise the name-based fallback would silently overwrite a
		// deliberate tag-based mapping.
		if lowerName := strings.ToLower(f.Name); lowerName != tomlKey {
			if _, exists := lookup[lowerName]; !exists {
				lookup[lowerName] = fieldDesc{jsonName: jsonName, fieldType: f.Type}
			}
		}
	}

	out := make(map[string]any, len(m))
	for k, v := range m {
		desc, ok := lookup[k]
		if !ok {
			// Try case-insensitive match.
			kl := strings.ToLower(k)
			desc, ok = lookup[kl]
		}
		if !ok {
			// Drop keys that map to a toml:"-" field; otherwise pass through
			// so json.Unmarshal can handle unknown keys.
			if excluded[strings.ToLower(k)] {
				continue
			}
			out[k] = normaliseValue(v)
			continue
		}
		// Recursively remap nested structs.
		var remappedVal any
		if subMap, ok2 := v.(map[string]any); ok2 {
			remappedVal = remapForTarget(subMap, desc.fieldType)
		} else if arr, ok2 := v.([]any); ok2 {
			result := make([]any, len(arr))
			for i, elem := range arr {
				if elemMap, ok3 := elem.(map[string]any); ok3 {
					result[i] = remapForTarget(elemMap, sliceElemType(desc.fieldType))
				} else {
					result[i] = normaliseValue(elem)
				}
			}
			remappedVal = result
		} else {
			remappedVal = normaliseValue(v)
		}
		out[desc.jsonName] = remappedVal
	}
	return out
}

// sliceElemType returns the element type of a slice/array type, or the type
// itself if it is not a slice/array.
func sliceElemType(t reflect.Type) reflect.Type {
	for t != nil && (t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Ptr) {
		t = t.Elem()
	}
	return t
}

// normaliseMapKeys converts map[string]any recursively, ensuring all nested
// maps are also map[string]any (needed for json.Marshal to work uniformly).
func normaliseMapKeys(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = normaliseValue(v)
	}
	return out
}

func normaliseValue(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		return normaliseMapKeys(tv)
	case []any:
		result := make([]any, len(tv))
		for i, elem := range tv {
			result[i] = normaliseValue(elem)
		}
		return result
	default:
		return v
	}
}

func splitDottedKey(dottedKey string) []string {
	return strings.Split(dottedKey, ".")
}

func (t *Toml) getValueByPath(keys []string) (any, bool) {
	current := any(t.rootNode)

	for _, key := range keys {
		key = strings.Trim(key, "\"")

		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		val, exists := currentMap[key]
		if !exists {
			return nil, false
		}

		current = val
	}

	return current, true
}

// convertParseErrors converts internal ParseError values to public Diagnostics.
func convertParseErrors(errs []parser.ParseError) []Diagnostic {
	if len(errs) == 0 {
		return make([]Diagnostic, 0)
	}
	result := make([]Diagnostic, 0, len(errs))
	for _, e := range errs {
		d := Diagnostic{
			Message:  e.Message,
			Severity: diagnostics.Error,
		}
		if e.Line > 0 {
			d.Location = &Location{
				StartLine:   e.Line,
				StartColumn: e.Column,
				EndLine:     e.EndLine,
				EndColumn:   e.EndCol,
			}
		}
		result = append(result, d)
	}
	return result
}
