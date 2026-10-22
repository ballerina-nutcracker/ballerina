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

// The converter binds composed YAML nodes to the expected type with the rules of jBallerina's composer:
// scalar text is converted by the target type rather than by YAML schema resolution.

package native

import (
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// conversionError is an error raised while binding a value to the expected type. It carries no position.
type conversionError struct {
	message string
}

func (e *conversionError) Error() string {
	return e.message
}

func newConversionError(format string, args ...any) *conversionError {
	return &conversionError{message: sprintf(format, args...)}
}

func arraySizeError() *conversionError {
	return newConversionError("array size is not compatible with the expected size")
}

// scope is where a value is being bound: the dotted path of the mapping keys above it, and the record
// field it binds to, if any.
type scope struct {
	path  []string
	field *recordField
}

func (sc scope) child(key string, field *recordField) scope {
	path := append(slices.Clone(sc.path), key)
	return scope{path: path, field: field}
}

func (sc scope) pathString() string {
	return strings.Join(sc.path, ".")
}

type converter struct {
	t       *typer
	options *readOptions
	// renames maps YAML keys to the renamed fields of the root record type.
	root    semtypes.SemType
	renames map[string]string
}

// fieldFor finds the declared field a key binds to. A renamed field binds only to its YAML key.
func (c *converter) fieldFor(shape *rowShape, key string) (*recordField, bool) {
	if len(c.renames) > 0 && semtypes.IsSameType(c.t.tc, shape.ty, c.root) {
		if name, renamed := c.renames[key]; renamed {
			f, ok := shape.fields[name]
			return f, ok
		}
		for _, name := range c.renames {
			if name == key {
				return nil, false
			}
		}
	}
	f, ok := shape.fields[key]
	return f, ok
}

func (c *converter) convert(n *node, ty semtypes.SemType, sc scope) (values.BalValue, error) {
	ti := c.t.classify(ty)
	switch n.kind {
	case nodeMapping:
		return c.convertMapping(n, ti, sc)
	case nodeSequence:
		return c.convertSequence(n, ti, sc)
	}
	return c.convertScalar(n, ti, sc)
}

func (c *converter) incompatible(text string, ty semtypes.SemType, sc scope) error {
	if sc.field != nil {
		return newConversionError("incompatible value '%s' for type '%s' in field '%s'", text, c.t.typeName(ty),
			sc.pathString())
	}
	return newConversionError("incompatible expected type '%s' for value '%s'", c.t.typeName(ty), text)
}

func (c *converter) convertScalar(n *node, ti typeInfo, sc scope) (values.BalValue, error) {
	switch {
	case n.empty:
		if c.t.isNilable(ti.ty) || ti.kind == kindJSON || ti.kind == kindAnydata {
			return nil, nil
		}
		return nil, c.incompatible("", ti.ty, sc)
	case n.hasTyped:
		if c.t.contains(ti.ty, n.typed) {
			return n.typed, nil
		}
		return nil, c.incompatible(n.value, ti.ty, sc)
	}
	if converted, ok := c.fromString(n.value, ti); ok {
		return converted, nil
	}
	return nil, c.incompatible(n.value, ti.ty, sc)
}

func (c *converter) convertMapping(n *node, ti typeInfo, sc scope) (values.BalValue, error) {
	switch ti.kind {
	case kindRecord, kindMap:
		return c.convertToMapping(n, c.t.rowShapeOf(ti), sc)
	case kindJSON:
		return c.convertToMapping(n, c.t.rowShapeOf(typeInfo{kind: kindMap, ty: c.t.jsonMapTy, mapping: c.t.jsonMap}), sc)
	case kindAnydata:
		return c.convertToMapping(n, c.t.rowShapeOf(typeInfo{kind: kindMap, ty: c.t.anydataMapTy, mapping: c.t.anydataMap}), sc)
	case kindUnion:
		return c.convertToUnion(n, ti, sc)
	case kindArray, kindTuple, kindUnsupported:
		return nil, newConversionError("invalid type '%s' expected 'map type'", c.t.typeName(ti.ty))
	}
	if sc.field != nil || len(sc.path) > 0 {
		return nil, newConversionError("invalid type for field '%s'", sc.pathString())
	}
	return nil, newConversionError("invalid type '%s' expected 'map type'", c.t.typeName(ti.ty))
}

func (c *converter) convertSequence(n *node, ti typeInfo, sc scope) (values.BalValue, error) {
	switch ti.kind {
	case kindArray, kindTuple:
		return c.convertToList(n.items, c.t.rowShapeOf(ti), sc)
	case kindJSON:
		return c.convertToList(n.items, c.t.rowShapeOf(typeInfo{kind: kindArray, ty: c.t.jsonListTy, list: c.t.jsonList}), sc)
	case kindAnydata:
		return c.convertToList(n.items, c.t.rowShapeOf(typeInfo{kind: kindArray, ty: c.t.anydataListTy, list: c.t.anydataList}), sc)
	case kindUnion:
		return c.convertToUnion(n, ti, sc)
	}
	return nil, newConversionError("invalid type '%s' expected 'list type'", c.t.typeName(ti.ty))
}

// convertToUnion binds a collection to the first union member it converts to.
func (c *converter) convertToUnion(n *node, ti typeInfo, sc scope) (values.BalValue, error) {
	for _, member := range ti.members {
		memberInfo := c.t.classify(member)
		if n.kind == nodeMapping && !memberInfo.isMapping() && memberInfo.kind != kindJSON && memberInfo.kind != kindAnydata {
			continue
		}
		if n.kind == nodeSequence && !memberInfo.isList() && memberInfo.kind != kindJSON && memberInfo.kind != kindAnydata {
			continue
		}
		if converted, err := c.convert(n, member, sc); err == nil {
			return converted, nil
		}
	}
	return nil, newConversionError("invalid type '%s' expected 'anydata'", c.t.typeName(ti.ty))
}

// convertToMapping binds a mapping node to a record or map. Unknown keys go to the rest type; a scalar
// that does not convert to the rest type is dropped, as in jBallerina.
func (c *converter) convertToMapping(n *node, shape rowShape, sc scope) (values.BalValue, error) {
	builder := newRowBuilder(&shape)
	remaining := shape.remainingFields()
	for _, p := range n.pairs {
		key := keyString(p.key)
		if f, declared := c.fieldFor(&shape, key); declared {
			delete(remaining, f.name)
			if c.options.nilAsOptionalField && f.optional && !f.nilable && p.value.kind == nodeScalar &&
				!p.value.empty && !p.value.hasTyped && p.value.value == "null" {
				continue
			}
			value, err := c.convert(p.value, f.ty, sc.child(f.name, f))
			if err != nil {
				return nil, err
			}
			builder.put(f.name, value)
			continue
		}
		if !shape.hasRest {
			if !c.options.allowDataProjection {
				return nil, newConversionError("undefined field '%s'", key)
			}
			continue
		}
		value, err := c.convert(p.value, shape.rest, sc.child(key, nil))
		if err != nil {
			if p.value.kind == nodeScalar {
				continue
			}
			return nil, err
		}
		builder.put(key, value)
	}
	for _, name := range shape.fieldOrder {
		f, missing := remaining[name]
		if !missing || f.optional {
			continue
		}
		if c.options.absentAsNilableType && f.nilable {
			builder.put(name, nil)
			continue
		}
		return nil, newConversionError("required field '%s' not present in YAML", name)
	}
	return builder.build(c.t)
}

func (c *converter) convertToList(items []*node, shape rowShape, sc scope) (values.BalValue, error) {
	builder := newRowBuilder(&shape)
	for i, item := range items {
		memberType, ok := shape.memberType(i)
		if !ok {
			if !c.options.allowDataProjection {
				return nil, arraySizeError()
			}
			break
		}
		value, err := c.convert(item, memberType, sc)
		if err != nil {
			return nil, err
		}
		builder.set(i, value)
	}
	return builder.build(c.t)
}

// fromString is jBallerina's Values.fromStringWithType: it converts scalar text to the target type.
func (c *converter) fromString(text string, ti typeInfo) (values.BalValue, bool) {
	switch ti.kind {
	case kindInt:
		return parseJavaLong(text)
	case kindIntSubtype:
		n, ok := parseJavaLong(text)
		if !ok || !c.t.contains(ti.ty, n) {
			return nil, false
		}
		return n, true
	case kindFloat:
		return parseJavaDouble(text)
	case kindDecimal:
		return parseJavaBigDecimal(text)
	case kindChar:
		return text, len([]rune(text)) == 1
	case kindString:
		return text, c.isSchemaString(text)
	case kindBoolean:
		switch text {
		case "true":
			return true, true
		case "false":
			return false, true
		}
		return nil, false
	case kindNil:
		return nil, text == "null"
	case kindFinite:
		return c.toSingleton(text, ti)
	case kindUnion:
		return c.toUnion(text, ti.members)
	case kindJSON, kindAnydata:
		return c.toUnion(text, jsonMemberTypes)
	}
	return nil, false
}

var jsonMemberTypes = []semtypes.SemType{
	semtypes.Nil, semtypes.Boolean, semtypes.Int, semtypes.Float, semtypes.Decimal, semtypes.String,
}

// isSchemaString reports whether the schema lets text be a string: the JSON schema reserves null, true
// and false, and the core schema also their capitalised forms and ~.
func (c *converter) isSchemaString(text string) bool {
	switch c.options.schema {
	case schemaJSON:
		return text != "null" && text != "true" && text != "false"
	case schemaCore:
		return !isAnyOf(text, "null", "Null", "NULL", "~", "true", "True", "TRUE", "false", "False", "FALSE")
	}
	return true
}

// unionPriority is jBallerina's TYPE_PRIORITY_ORDER; other members follow in their own order.
var unionPriority = map[typeKind]int{
	kindInt: 0, kindFloat: 1, kindDecimal: 2, kindNil: 3, kindBoolean: 4, kindJSON: 5, kindString: 6,
}

func (c *converter) toUnion(text string, members []semtypes.SemType) (values.BalValue, bool) {
	infos := make([]typeInfo, len(members))
	for i, member := range members {
		infos[i] = c.t.classify(member)
	}
	slices.SortStableFunc(infos, func(a, b typeInfo) int {
		return priorityOf(a) - priorityOf(b)
	})
	for _, info := range infos {
		if converted, ok := c.fromString(text, info); ok {
			return converted, true
		}
	}
	return nil, false
}

func priorityOf(ti typeInfo) int {
	if p, ok := unionPriority[ti.kind]; ok {
		return p
	}
	return len(unionPriority)
}

// toSingleton matches text against the singletons of a finite type: the text must equal the singleton's
// string form and convert to the singleton's own type.
func (c *converter) toSingleton(text string, ti typeInfo) (values.BalValue, bool) {
	for _, basic := range scalarBasicTypes {
		if semtypes.IsEmpty(c.t.tc, semtypes.Intersect(ti.ty, basic.ty)) {
			continue
		}
		converted, ok := c.fromString(text, typeInfo{kind: basic.kind, ty: basic.ty})
		if ok && c.t.contains(ti.ty, converted) && javaValueOf(converted) == text {
			return converted, true
		}
	}
	return nil, false
}
