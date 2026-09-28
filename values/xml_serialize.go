// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package values

import (
	"fmt"
	"strings"
)

type xmlNamespaceBinding struct {
	prefix string
	uri    string
	depth  int
	order  int
}

type xmlNamespaceUndo struct {
	prefix   string
	previous xmlNamespaceBinding
	hadValue bool
}

type xmlNamespaceState struct {
	active  map[string]xmlNamespaceBinding
	byURI   map[string]map[string]xmlNamespaceBinding
	history map[string][]xmlNamespaceBinding
}

func newXMLNamespaceState() *xmlNamespaceState {
	xmlBinding := xmlNamespaceBinding{prefix: "xml", uri: XMLNamespaceURI, depth: -1}
	return &xmlNamespaceState{
		active:  map[string]xmlNamespaceBinding{"xml": xmlBinding},
		byURI:   map[string]map[string]xmlNamespaceBinding{XMLNamespaceURI: {"xml": xmlBinding}},
		history: map[string][]xmlNamespaceBinding{XMLNamespaceURI: {xmlBinding}},
	}
}

func (s *xmlNamespaceState) set(binding xmlNamespaceBinding) xmlNamespaceUndo {
	previous, hadValue := s.active[binding.prefix]
	if hadValue {
		delete(s.byURI[previous.uri], previous.prefix)
	}
	s.active[binding.prefix] = binding
	if s.byURI[binding.uri] == nil {
		s.byURI[binding.uri] = map[string]xmlNamespaceBinding{}
	}
	s.byURI[binding.uri][binding.prefix] = binding
	s.history[binding.uri] = append(s.history[binding.uri], binding)
	return xmlNamespaceUndo{binding.prefix, previous, hadValue}
}

func (s *xmlNamespaceState) restore(undo xmlNamespaceUndo) {
	current := s.active[undo.prefix]
	delete(s.byURI[current.uri], undo.prefix)
	s.history[current.uri] = s.history[current.uri][:len(s.history[current.uri])-1]
	if !undo.hadValue {
		delete(s.active, undo.prefix)
		return
	}
	s.active[undo.prefix] = undo.previous
	if s.byURI[undo.previous.uri] == nil {
		s.byURI[undo.previous.uri] = map[string]xmlNamespaceBinding{}
	}
	s.byURI[undo.previous.uri][undo.prefix] = undo.previous
}

func (s *xmlNamespaceState) choose(uri string, depth int, local bool) (string, bool) {
	bindings := s.history[uri]
	bestDepth, bestOrder := -2, int(^uint(0)>>1)
	bestPrefix := ""
	found := false
	for i := len(bindings) - 1; i >= 0; i-- {
		binding := bindings[i]
		prefix := binding.prefix
		if active, ok := s.active[prefix]; !ok || active != binding {
			continue
		}
		if prefix == "" {
			continue
		}
		if local {
			if binding.depth < depth {
				break
			}
			if binding.depth != depth {
				continue
			}
			if !found || binding.order < bestOrder {
				bestPrefix, bestOrder, found = prefix, binding.order, true
			}
			continue
		}
		if binding.depth >= depth {
			continue
		}
		if found && binding.depth < bestDepth {
			break
		}
		if !found || binding.depth > bestDepth || binding.depth == bestDepth && binding.order < bestOrder {
			bestPrefix, bestDepth, bestOrder, found = prefix, binding.depth, binding.order, true
		}
	}
	return bestPrefix, found
}

type serializedXMLAttribute struct{ name, value string }
type serializedXMLNamespace struct{ prefix, uri string }

// splitXMLAttributes separates namespace declarations, stored as attributes in
// the XMLNS namespace, from the attributes to print by name.
func (e *XMLElement) splitXMLAttributes() ([]serializedXMLNamespace, []MapEntry) {
	var declared []serializedXMLNamespace
	var attrs []MapEntry
	for _, key := range e.Attributes.Keys() {
		value, _ := e.Attributes.Get(key)
		uri, local, err := ParseExpandedXMLName(key)
		if err != nil {
			panic(NewErrorWithMessage(err.Error()))
		}
		if uri == XMLNSNamespaceURI {
			declared = append(declared, serializedXMLNamespace{xmlNamespaceDeclarationPrefix(local), value.(string)})
			continue
		}
		attrs = append(attrs, MapEntry{Key: key, Value: value})
	}
	return declared, attrs
}

func (e *XMLElement) writeXMLString(b *strings.Builder, state *xmlNamespaceState, depth int) {
	declared, attrEntries := e.splitXMLAttributes()
	usedURIs := map[string]bool{}
	if e.NamespaceURI != "" {
		usedURIs[e.NamespaceURI] = true
	}
	for _, entry := range attrEntries {
		uri, _, _ := ParseExpandedXMLName(entry.Key)
		if uri != "" {
			usedURIs[uri] = true
		}
	}

	stored := make([]serializedXMLNamespace, 0, len(declared))
	undos := make([]xmlNamespaceUndo, 0, len(declared)+2)
	parentHasURI := make(map[string]bool, len(declared))
	localPrefixes := make(map[string]bool, len(declared))
	for _, ns := range declared {
		parentHasURI[ns.uri] = len(state.byURI[ns.uri]) > 0
		localPrefixes[ns.prefix] = true
	}
	localOrder := 0
	hasLocalDefault := false
	for _, ns := range declared {
		if ns.prefix == "" {
			hasLocalDefault = true
		}
		if ns.prefix != "" && usedURIs[ns.uri] && parentHasURI[ns.uri] {
			continue
		}
		stored = append(stored, ns)
		undos = append(undos, state.set(xmlNamespaceBinding{ns.prefix, ns.uri, depth, localOrder}))
		localOrder++
	}

	var synthesized []serializedXMLNamespace
	nextGeneratedPrefix := 0
	synthesize := func(prefix, uri string) {
		synthesized = append(synthesized, serializedXMLNamespace{prefix, uri})
		undos = append(undos, state.set(xmlNamespaceBinding{prefix, uri, depth, localOrder}))
		localOrder++
	}
	newPrefix := func(uri string) string {
		for {
			prefix := fmt.Sprintf("ns%d", nextGeneratedPrefix)
			nextGeneratedPrefix++
			if _, active := state.active[prefix]; active {
				continue
			}
			if !localPrefixes[prefix] {
				synthesize(prefix, uri)
				return prefix
			}
		}
	}
	nonDefaultCache := map[string]string{}
	selectNonDefault := func(uri string) string {
		if uri == XMLNamespaceURI {
			return "xml"
		}
		if prefix, ok := nonDefaultCache[uri]; ok {
			return prefix
		}
		if prefix, ok := state.choose(uri, depth, true); ok {
			nonDefaultCache[uri] = prefix
			return prefix
		}
		if prefix, ok := state.choose(uri, depth, false); ok {
			nonDefaultCache[uri] = prefix
			return prefix
		}
		prefix := newPrefix(uri)
		nonDefaultCache[uri] = prefix
		return prefix
	}

	attrs := make([]serializedXMLAttribute, 0, len(attrEntries))
	for _, entry := range attrEntries {
		value := entry.Value
		uri, local, _ := ParseExpandedXMLName(entry.Key)
		name := local
		if uri != "" {
			name = selectNonDefault(uri) + ":" + local
		}
		attrs = append(attrs, serializedXMLAttribute{name, value.(string)})
	}

	elementName := e.LocalName
	if e.NamespaceURI != "" {
		defaultBinding, hasDefault := state.active[""]
		if hasDefault && defaultBinding.uri == e.NamespaceURI && defaultBinding.depth == depth {
		} else if hasDefault && defaultBinding.uri == e.NamespaceURI && defaultBinding.depth < depth {
		} else if prefix, ok := state.choose(e.NamespaceURI, depth, true); ok {
			elementName = prefix + ":" + e.LocalName
		} else if prefix, ok := state.choose(e.NamespaceURI, depth, false); ok {
			elementName = prefix + ":" + e.LocalName
		} else if !hasLocalDefault {
			synthesize("", e.NamespaceURI)
		} else {
			elementName = newPrefix(e.NamespaceURI) + ":" + e.LocalName
		}
	} else if binding, ok := state.active[""]; ok && binding.uri != "" {
		if hasLocalDefault {
			panic(NewErrorWithMessage("unnamespaced XML element has a conflicting local default namespace"))
		}
		synthesize("", "")
	}

	b.WriteByte('<')
	b.WriteString(elementName)
	for _, attr := range attrs {
		b.WriteByte(' ')
		b.WriteString(attr.name)
		b.WriteString(`="`)
		b.WriteString(EscapeXMLAttribute(attr.value))
		b.WriteByte('"')
	}
	for _, ns := range append(stored, synthesized...) {
		b.WriteString(" xmlns")
		if ns.prefix != "" {
			b.WriteByte(':')
			b.WriteString(ns.prefix)
		}
		b.WriteString(`="`)
		b.WriteString(EscapeXMLAttribute(ns.uri))
		b.WriteByte('"')
	}
	if text, empty := e.Children.(*XMLText); empty && text.Body == "" {
		b.WriteString("/>")
	} else {
		b.WriteByte('>')
		writeXMLValue(b, e.Children, state, depth+1)
		b.WriteString("</")
		b.WriteString(elementName)
		b.WriteByte('>')
	}
	for i := len(undos) - 1; i >= 0; i-- {
		state.restore(undos[i])
	}
}

func writeXMLValue(b *strings.Builder, value XMLValue, state *xmlNamespaceState, depth int) {
	switch value := value.(type) {
	case *XMLElement:
		value.writeXMLString(b, state, depth)
	case *XMLSequence:
		for _, child := range value.Children {
			writeXMLValue(b, child, state, depth)
		}
	default:
		b.WriteString(value.XMLString())
	}
}
