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

package values

// Clone implements the clone abstract operation defined in the Ballerina spec
// (https://ballerina.io/spec/lang/master/#section_16.5).
//
// Unlike CloneWithType there is no target type, so every cloned container keeps
// the source's own inherent type, atomic type and filler, no numeric conversion
// happens and the operation always succeeds.
//
// An immutable value is returned as-is rather than copied, and the graph
// structure of the source is preserved: a cyclic value clones to a
// correspondingly cyclic clone, and a value reachable by more than one path is
// cloned once and shared by the clone's corresponding paths.
func Clone(value BalValue) BalValue {
	return clone(value, make(map[BalValue]BalValue))
}

// clone deep-copies value, consulting seen - an identity table from source
// value to its clone - before copying and populating it before recursing into
// children, which is what makes cycles terminate and sharing survive.
//
// The default case returns the value unchanged, so a value kind that is not
// statically Cloneable can never be silently mishandled.
func clone(value BalValue, seen map[BalValue]BalValue) BalValue {
	if cloned, found := seen[value]; found {
		return cloned
	}
	switch v := value.(type) {
	case *Map:
		if v.isReadonly {
			return v
		}
		return cloneMap(v, seen)
	case *List:
		if v.isReadonly {
			return v
		}
		return cloneList(v, seen)
	case XMLValue:
		if v.Readonly() {
			return v
		}
		return cloneXML(v, seen)
	default:
		return value
	}
}

func cloneMap(m *Map, seen map[BalValue]BalValue) *Map {
	cloned := NewMap(m.Type, m.atomic, false, nil)
	seen[m] = cloned
	for _, key := range m.Keys() {
		value, _ := m.Get(key)
		cloned.putUnchecked(key, clone(value, seen))
	}
	return cloned
}

func cloneList(l *List, seen map[BalValue]BalValue) *List {
	cloned := NewList(l.Type, l.atomic, false, l.filler, 0, nil)
	seen[l] = cloned
	cloned.elems = make([]BalValue, len(l.elems))
	for i, elem := range l.elems {
		cloned.elems[i] = clone(elem, seen)
	}
	return cloned
}

func cloneXML(x XMLValue, seen map[BalValue]BalValue) XMLValue {
	switch v := x.(type) {
	case *XMLElement:
		cloned := &XMLElement{
			Prefix:       v.Prefix,
			LocalName:    v.LocalName,
			NamespaceURI: v.NamespaceURI,
			semType:      v.semType,
		}
		seen[v] = cloned
		cloned.Attributes = cloneXMLStringMap(v.Attributes, seen)
		cloned.Namespaces = cloneXMLStringMap(v.Namespaces, seen)
		if v.Children != nil {
			cloned.Children = clone(v.Children, seen).(XMLValue)
		}
		return cloned
	case *XMLSequence:
		cloned := &XMLSequence{semType: v.semType}
		seen[v] = cloned
		cloned.Children = make([]XMLValue, len(v.Children))
		for i, child := range v.Children {
			cloned.Children[i] = clone(child, seen).(XMLValue)
		}
		return cloned
	case *XMLComment:
		cloned := &XMLComment{Body: v.Body, semType: v.semType}
		seen[v] = cloned
		return cloned
	case *XMLProcessingInstruction:
		cloned := &XMLProcessingInstruction{Target: v.Target, Data: v.Data, semType: v.semType}
		seen[v] = cloned
		return cloned
	default:
		return x
	}
}

func cloneXMLStringMap(m *Map, seen map[BalValue]BalValue) *Map {
	if m == nil {
		return nil
	}
	return clone(m, seen).(*Map)
}
