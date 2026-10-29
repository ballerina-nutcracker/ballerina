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

// The conversion builds mutable mapping and list nodes, mirroring how jBallerina fills BMap and BArray values
// after attaching them to their parent, and turns the finished tree into runtime values at the end.

package native

import (
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// node is a *mapNode, an *arrayNode or a finished values.BalValue.
type node = any

type mapNode struct {
	ty     semtypes.SemType
	keys   []string
	values map[string]node
	// defaults are the values a jBallerina record value reports for fields that were never set: a
	// primitive int, float or boolean field reads as its zero value.
	defaults map[string]values.BalValue
	// required lists the non-optional fields, in field order. One never assigned reads as () like in a
	// jBallerina record value, whatever its declared type.
	required []recordField
}

func newMapNode(ty semtypes.SemType) *mapNode {
	return &mapNode{ty: ty, values: map[string]node{}, defaults: map[string]values.BalValue{}}
}

// newRecordNode is ValueCreator.createRecordValue for a record type.
func (t *typer) newRecordNode(ti typeInfo) *mapNode {
	n := newMapNode(ti.ty)
	for _, field := range t.recordFields(ti) {
		if field.optional {
			continue
		}
		n.required = append(n.required, field)
		switch t.classify(field.ty).kind {
		case kindInt, kindByte, kindIntSubtype:
			n.defaults[field.runtimeName] = int64(0)
		case kindFloat:
			n.defaults[field.runtimeName] = float64(0)
		case kindBoolean:
			n.defaults[field.runtimeName] = false
		}
	}
	return n
}

func (n *mapNode) get(key string) node {
	if v, ok := n.values[key]; ok {
		return v
	}
	return n.defaults[key]
}

func (n *mapNode) containsKey(key string) bool {
	_, ok := n.values[key]
	return ok
}

func (n *mapNode) put(key string, v node) {
	if _, ok := n.values[key]; !ok {
		n.keys = append(n.keys, key)
	}
	n.values[key] = v
}

func (n *mapNode) isEmpty() bool {
	return len(n.values) == 0
}

type arrayNode struct {
	ty    typeInfo
	items []node
}

// newArrayNode is ValueCreator.createArrayValue: a fixed-length array starts with its full length.
func (t *typer) newArrayNode(ty semtypes.SemType) *arrayNode {
	ti := t.classify(ty)
	n := &arrayNode{ty: ti}
	if ti.size > 0 {
		n.items = make([]node, ti.size)
	}
	return n
}

// add is BArray.add(index, value): setting past the end extends the array.
func (n *arrayNode) add(index int, v node) {
	for len(n.items) <= index {
		n.items = append(n.items, nil)
	}
	n.items[index] = v
}

func (n *arrayNode) append(v node) {
	n.items = append(n.items, v)
}

func (n *arrayNode) length() int {
	return len(n.items)
}

func (n *arrayNode) last() node {
	if len(n.items) == 0 {
		return nil
	}
	return n.items[len(n.items)-1]
}

// build turns a node tree into runtime values.
func (t *typer) build(v node) (values.BalValue, error) {
	switch v := v.(type) {
	case *mapNode:
		return t.buildMap(v)
	case *arrayNode:
		return t.buildArray(v)
	default:
		return v, nil
	}
}

func (t *typer) buildMap(n *mapNode) (values.BalValue, error) {
	entries := make([]values.MapEntry, 0, len(n.keys)+len(n.required))
	for _, key := range n.keys {
		value, err := t.build(n.values[key])
		if err != nil {
			return nil, err
		}
		entries = append(entries, values.MapEntry{Key: key, Value: value})
	}
	for _, field := range n.required {
		if n.get(field.runtimeName) == nil && !semtypes.ContainsBasicType(field.ty, semtypes.Nil) {
			return nil, fieldCannotConvertError(field.name, t.typeName(field.ty))
		}
		if n.containsKey(field.runtimeName) {
			continue
		}
		entries = append(entries, values.MapEntry{Key: field.runtimeName, Value: n.defaults[field.runtimeName]})
	}
	atomic := semtypes.ToMappingAtomicType(t.tc, n.ty)
	if atomic == nil {
		atomic = semtypes.ToMappingAtomicType(t.tc, t.mapType(t.anydataTy))
	}
	return values.NewMap(n.ty, atomic, t.isReadonly(n.ty), entries), nil
}

func (t *typer) buildArray(n *arrayNode) (values.BalValue, error) {
	items := make([]values.BalValue, len(n.items))
	for i, item := range n.items {
		if item == nil {
			filler, ok := values.FillerValue(t.tc, n.ty.element)
			if !ok {
				return nil, arraySizeMismatchError()
			}
			items[i] = filler
			continue
		}
		value, err := t.build(item)
		if err != nil {
			return nil, err
		}
		items[i] = value
	}
	atomic := n.ty.list
	restFiller, _ := values.FillerFactoryFor(t.tc, atomic.Rest())
	return values.NewList(n.ty.ty, atomic, t.isReadonly(n.ty.ty), restFiller, len(items), items), nil
}

// arrayValueFor is DataUtils.createArrayValue.
func (t *typer) arrayValueFor(ty semtypes.SemType) *arrayNode {
	switch t.classify(ty).kind {
	case kindJSON:
		return t.newArrayNode(t.arrayType(t.jsonTy))
	case kindAnydata:
		return t.newArrayNode(t.arrayType(t.anydataTy))
	default:
		return t.newArrayNode(ty)
	}
}
