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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// A minimal Avro schema shape resolver.
//
// goavro's Codec parses and validates a schema, but keeps its parsed form
// internal — it exposes no typed schema tree. Correct encoding still needs to
// know, at every leaf, whether a position is `int` or `long` (both map to a
// Ballerina int64, but need different Go types and different overflow
// behaviour), `float` or `double`, and — for a union — the exact declared
// name of each branch, since goavro's native representation of a union value
// is a Go map keyed by that name. This file builds that shape tree by walking
// the same schema JSON goavro parses, mirroring goavro's own name-resolution
// rules (see its name.go) closely enough that the two interpretations always
// agree on structure.

package native

import (
	"encoding/json"
	"fmt"
)

type shapeKind int

const (
	shapeNull shapeKind = iota
	shapeBoolean
	shapeInt
	shapeLong
	shapeFloat
	shapeDouble
	shapeBytes
	shapeString
	shapeRecord
	shapeEnum
	shapeArray
	shapeMap
	shapeUnion
	shapeFixed
)

// shape describes one position in an Avro schema tree, carrying exactly the
// structural information encodeValue/decodeValue need and nothing goavro
// already tracks for us (wire layout, validation).
type shape struct {
	kind shapeKind
	// fullName is the exact key goavro uses to identify this type as a union
	// branch: the bare Avro type name for primitives ("int", "string", ...),
	// "array"/"map" for those two, and the namespace-qualified name for a
	// named type (record/enum/fixed). Unused on a union shape itself — Avro
	// disallows a union directly containing another union.
	fullName string
	fields   []fieldShape // record
	item     *shape       // array
	value    *shape       // map
	branches []*shape     // union, in declared order
	size     int          // fixed
}

type fieldShape struct {
	name  string
	shape *shape
}

// parseSchemaShape parses schema the same way goavro.NewCodec will, building
// a parallel shape tree. Call both on the same string so a schema either
// parses under both or is rejected before either is trusted.
//
// It also returns schema with every "logicalType" key stripped, for handing
// to goavro.NewCodec instead of the original: unlike this shape walk, which
// only ever looks at "type"/"items"/"values"/"fields"/"size"/"name", goavro's
// codec builder treats a recognised logicalType (decimal, date, timestamp-*,
// ...) as its own type and applies conversions jBallerina's GenericDatumReader
// never does. Stripping the key makes goavro fall back to the underlying
// primitive, matching this shape tree and jBallerina's behaviour.
func parseSchemaShape(schema string) (*shape, string, error) {
	var raw any
	if err := json.Unmarshal([]byte(schema), &raw); err != nil {
		return nil, "", err
	}
	registry := make(map[string]*shape)
	s, err := buildShape(raw, "", registry)
	if err != nil {
		return nil, "", err
	}
	stripped, err := json.Marshal(stripLogicalType(raw))
	if err != nil {
		return nil, "", err
	}
	return s, string(stripped), nil
}

// stripLogicalType recursively removes "logicalType" keys from a parsed
// schema so goavro treats every logical type as its underlying primitive.
func stripLogicalType(raw any) any {
	switch v := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			if key == "logicalType" {
				continue
			}
			out[key] = stripLogicalType(value)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, value := range v {
			out[i] = stripLogicalType(value)
		}
		return out
	default:
		return raw
	}
}

func buildShape(raw any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	switch v := raw.(type) {
	case string:
		return buildShapeFromName(v, enclosingNamespace, registry)
	case []any:
		return buildUnionShape(v, enclosingNamespace, registry)
	case map[string]any:
		return buildShapeFromMap(v, enclosingNamespace, registry)
	default:
		return nil, fmt.Errorf("avro: unsupported schema value: %v", raw)
	}
}

func buildShapeFromName(typeName, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	if k, ok := primitiveKind(typeName); ok {
		return &shape{kind: k, fullName: typeName}, nil
	}
	if s, ok := registry[typeName]; ok {
		return s, nil
	}
	// Avro allows a bare reference to abbreviate a name defined in the
	// enclosing namespace — the same fallback goavro's codec.go applies.
	if enclosingNamespace != "" {
		if s, ok := registry[enclosingNamespace+"."+typeName]; ok {
			return s, nil
		}
	}
	return nil, fmt.Errorf("avro: unknown type name: %q", typeName)
}

func primitiveKind(typeName string) (shapeKind, bool) {
	switch typeName {
	case "null":
		return shapeNull, true
	case "boolean":
		return shapeBoolean, true
	case "int":
		return shapeInt, true
	case "long":
		return shapeLong, true
	case "float":
		return shapeFloat, true
	case "double":
		return shapeDouble, true
	case "bytes":
		return shapeBytes, true
	case "string":
		return shapeString, true
	}
	return 0, false
}

func buildUnionShape(members []any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	branches := make([]*shape, len(members))
	for i, member := range members {
		branch, err := buildShape(member, enclosingNamespace, registry)
		if err != nil {
			return nil, fmt.Errorf("avro: union member %d: %w", i, err)
		}
		branches[i] = branch
	}
	return &shape{kind: shapeUnion, branches: branches}, nil
}

func buildShapeFromMap(schemaMap map[string]any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	typeValue, ok := schemaMap["type"]
	if !ok {
		return nil, fmt.Errorf("avro: schema map missing \"type\"")
	}
	// A {"type": [...]} form nests a union; handled the same as a bare union.
	if members, ok := typeValue.([]any); ok {
		return buildUnionShape(members, enclosingNamespace, registry)
	}
	typeName, ok := typeValue.(string)
	if !ok {
		return nil, fmt.Errorf("avro: \"type\" ought to be a string or array")
	}
	if k, ok := primitiveKind(typeName); ok {
		return &shape{kind: k, fullName: typeName}, nil
	}
	switch typeName {
	case "array":
		return buildArrayShape(schemaMap, enclosingNamespace, registry)
	case "map":
		return buildMapShape(schemaMap, enclosingNamespace, registry)
	case "record", "error":
		return buildRecordShape(schemaMap, enclosingNamespace, registry)
	case "enum":
		return buildNamedLeaf(shapeEnum, schemaMap, enclosingNamespace, registry)
	case "fixed":
		return buildFixedShape(schemaMap, enclosingNamespace, registry)
	default:
		// A previously-defined type referenced via {"type": "Name"} — the
		// same abbreviated form buildShapeFromName resolves for a bare string.
		return buildShapeFromName(typeName, enclosingNamespace, registry)
	}
}

func buildArrayShape(schemaMap map[string]any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	items, ok := schemaMap["items"]
	if !ok {
		return nil, fmt.Errorf("avro: array schema missing \"items\"")
	}
	item, err := buildShape(items, enclosingNamespace, registry)
	if err != nil {
		return nil, fmt.Errorf("avro: array items: %w", err)
	}
	return &shape{kind: shapeArray, item: item, fullName: "array"}, nil
}

func buildMapShape(schemaMap map[string]any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	values, ok := schemaMap["values"]
	if !ok {
		return nil, fmt.Errorf("avro: map schema missing \"values\"")
	}
	value, err := buildShape(values, enclosingNamespace, registry)
	if err != nil {
		return nil, fmt.Errorf("avro: map values: %w", err)
	}
	return &shape{kind: shapeMap, value: value, fullName: "map"}, nil
}

// buildRecordShape registers the record's shape before resolving its fields,
// the same way goavro's makeRecordCodec does, so a field that recurses back
// to this record resolves to this same shape rather than reparsing it.
func buildRecordShape(schemaMap map[string]any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	s := &shape{kind: shapeRecord}
	namespace, err := registerNamedShape(s, schemaMap, enclosingNamespace, registry)
	if err != nil {
		return nil, err
	}
	rawFields, ok := schemaMap["fields"].([]any)
	if !ok {
		return nil, fmt.Errorf("avro: record %q missing \"fields\"", s.fullName)
	}
	fields := make([]fieldShape, len(rawFields))
	for i, rawField := range rawFields {
		fieldMap, ok := rawField.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("avro: record %q field %d is not an object", s.fullName, i)
		}
		name, ok := fieldMap["name"].(string)
		if !ok {
			return nil, fmt.Errorf("avro: record %q field %d missing \"name\"", s.fullName, i)
		}
		fieldType, ok := fieldMap["type"]
		if !ok {
			return nil, fmt.Errorf("avro: record %q field %q missing \"type\"", s.fullName, name)
		}
		fieldShapeValue, err := buildShape(fieldType, namespace, registry)
		if err != nil {
			return nil, fmt.Errorf("avro: record %q field %q: %w", s.fullName, name, err)
		}
		fields[i] = fieldShape{name: name, shape: fieldShapeValue}
	}
	s.fields = fields
	return s, nil
}

func buildFixedShape(schemaMap map[string]any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	s := &shape{kind: shapeFixed}
	if _, err := registerNamedShape(s, schemaMap, enclosingNamespace, registry); err != nil {
		return nil, err
	}
	size, ok := schemaMap["size"].(float64)
	if !ok {
		return nil, fmt.Errorf("avro: fixed %q missing \"size\"", s.fullName)
	}
	s.size = int(size)
	return s, nil
}

// buildNamedLeaf handles named types with no nested schema of their own
// (currently just enum, whose symbol list goavro itself validates).
func buildNamedLeaf(k shapeKind, schemaMap map[string]any, enclosingNamespace string, registry map[string]*shape) (*shape, error) {
	s := &shape{kind: k}
	if _, err := registerNamedShape(s, schemaMap, enclosingNamespace, registry); err != nil {
		return nil, err
	}
	return s, nil
}

// registerNamedShape computes s's full name following the same rule as
// goavro's newName (name.go): an explicit dot in the name wins outright,
// then an explicit "namespace" on this schema map, then the enclosing
// namespace, then no namespace at all. It registers s under that full name
// before returning, so a self- or forward-reference resolves to s.
func registerNamedShape(s *shape, schemaMap map[string]any, enclosingNamespace string, registry map[string]*shape) (namespace string, err error) {
	name, ok := schemaMap["name"].(string)
	if !ok {
		return "", fmt.Errorf("avro: named schema missing \"name\"")
	}
	explicitNamespace, _ := schemaMap["namespace"].(string)
	s.fullName, namespace = resolveFullName(name, explicitNamespace, enclosingNamespace)
	registry[s.fullName] = s
	return namespace, nil
}

func resolveFullName(name, explicitNamespace, enclosingNamespace string) (fullName, namespace string) {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name, name[:i]
		}
	}
	if explicitNamespace != "" {
		return explicitNamespace + "." + name, explicitNamespace
	}
	if enclosingNamespace != "" {
		return enclosingNamespace + "." + name, enclosingNamespace
	}
	return name, ""
}
