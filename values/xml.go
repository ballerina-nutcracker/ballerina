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

import (
	"fmt"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type (
	XMLValue interface {
		Type() semtypes.SemType
		Readonly() bool
		XMLString() string
		IterItems() []XMLValue
	}

	XMLElement struct {
		Prefix       string
		LocalName    string
		NamespaceURI string
		Attributes   *Map
		// Namespaces holds XML namespace declarations to print on this element.
		// Keys are stored in already-printable form ("xmlns" or "xmlns:<prefix>");
		// values are URIs.
		Namespaces *Map
		Children   XMLValue
		semType    semtypes.SemType
		isReadonly bool
	}

	XMLSequence struct {
		Children   []XMLValue
		semType    semtypes.SemType
		isReadonly bool
	}

	XMLProcessingInstruction struct {
		Target     string
		Data       string
		semType    semtypes.SemType
		isReadonly bool
	}

	XMLText struct {
		Body    string
		semType semtypes.SemType
	}

	XMLComment struct {
		Body       string
		semType    semtypes.SemType
		isReadonly bool
	}
)

var (
	_ XMLValue = &XMLElement{}
	_ XMLValue = &XMLSequence{}
	_ XMLValue = &XMLProcessingInstruction{}
	_ XMLValue = &XMLText{}
	_ XMLValue = &XMLComment{}
)

func (e *XMLElement) Type() semtypes.SemType { return e.semType }

func (e *XMLElement) Readonly() bool { return e.isReadonly }

func (e *XMLElement) IterItems() []XMLValue { return []XMLValue{e} }

func (e *XMLElement) QualifiedName() string {
	if e.Prefix == "" {
		return e.LocalName
	}
	return e.Prefix + ":" + e.LocalName
}

func (e *XMLElement) ExpandedName() string {
	return ExpandedXMLName(e.NamespaceURI, e.LocalName)
}

func (e *XMLElement) SetExpandedName(name string) {
	if e.isReadonly {
		panic(NewErrorWithMessage("cannot mutate readonly XML element"))
	}
	uri, local, err := ParseExpandedXMLName(name)
	if err != nil || uri == XMLNSNamespaceURI {
		if err == nil {
			err = fmt.Errorf("element name cannot use the XMLNS namespace")
		}
		panic(NewErrorWithMessage(err.Error()))
	}
	if uri == "" {
		if value, ok := e.Namespaces.Get("xmlns"); ok && value.(string) != "" {
			panic(NewErrorWithMessage("unnamespaced XML element conflicts with its local default namespace"))
		}
	}
	e.Prefix, e.LocalName, e.NamespaceURI = "", local, uri
}

func (e *XMLElement) SetXMLChildren(children XMLValue) {
	if e.isReadonly {
		panic(NewErrorWithMessage("cannot mutate readonly XML element"))
	}
	normalized := NewNormalizedXMLSequence([]XMLValue{children})
	if XMLContainsElement(normalized, e) {
		panic(NewErrorWithMessage("XML child cycle"))
	}
	e.Children = normalized
}

func XMLContainsElement(value XMLValue, target *XMLElement) bool {
	visited := map[*XMLElement]bool{}
	var contains func(XMLValue) bool
	contains = func(value XMLValue) bool {
		for _, item := range value.IterItems() {
			element, ok := item.(*XMLElement)
			if !ok {
				continue
			}
			if element == target {
				return true
			}
			if !visited[element] {
				visited[element] = true
				if contains(element.Children) {
					return true
				}
			}
		}
		return false
	}
	return contains(value)
}

func (e *XMLElement) XMLString() string {
	var b strings.Builder
	e.writeXMLString(&b, newXMLNamespaceState(), 0)
	return b.String()
}

func writeXMLStringMap(b *strings.Builder, m *Map, kind string) {
	if m == nil {
		return
	}
	for _, k := range m.Keys() {
		v, _ := m.Get(k)
		sv, ok := v.(string)
		if !ok {
			panic(fmt.Sprintf("xml %s %q has non-string value of type %T", kind, k, v))
		}
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(EscapeXMLAttribute(sv))
		b.WriteByte('"')
	}
}

var xmlContentEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
)

var xmlAttributeEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	"\"", "&quot;",
)

// EscapeXMLContent escapes characters in XML text node bodies.
func EscapeXMLContent(s string) string {
	return xmlContentEscaper.Replace(s)
}

// EscapeXMLAttribute escapes characters in XML attribute values quoted with `"`.
func EscapeXMLAttribute(s string) string {
	return xmlAttributeEscaper.Replace(s)
}

func (s *XMLSequence) Type() semtypes.SemType { return s.semType }

func (s *XMLSequence) Readonly() bool { return s.isReadonly }

func (s *XMLSequence) IterItems() []XMLValue { return s.Children }

func (s *XMLSequence) XMLString() string {
	var b strings.Builder
	for _, child := range s.Children {
		b.WriteString(child.XMLString())
	}
	return b.String()
}

func (p *XMLProcessingInstruction) Type() semtypes.SemType { return p.semType }

func (p *XMLProcessingInstruction) Readonly() bool { return p.isReadonly }

func (p *XMLProcessingInstruction) IterItems() []XMLValue { return []XMLValue{p} }

func (p *XMLProcessingInstruction) XMLString() string {
	if strings.Contains(p.Data, "?>") {
		panic(NewErrorWithMessage(fmt.Sprintf("xml processing instruction %q data must not contain '?>'", p.Target)))
	}
	return "<?" + p.Target + " " + p.Data + "?>"
}

func (t *XMLText) Type() semtypes.SemType { return t.semType }

func (t *XMLText) Readonly() bool { return true }

func (t *XMLText) IterItems() []XMLValue {
	if t.Body == "" {
		return nil
	}
	return []XMLValue{t}
}

func (t *XMLText) XMLString() string {
	return EscapeXMLContent(t.Body)
}

func (c *XMLComment) Type() semtypes.SemType { return c.semType }

func (c *XMLComment) Readonly() bool { return c.isReadonly }

func (c *XMLComment) IterItems() []XMLValue { return []XMLValue{c} }

func (c *XMLComment) XMLString() string {
	if strings.Contains(c.Body, "--") || strings.HasSuffix(c.Body, "-") {
		panic(NewErrorWithMessage("xml comment body must not contain '--' or end with '-'"))
	}
	return "<!--" + c.Body + "-->"
}

func NewXMLElement(tc semtypes.Context, prefix, localName, namespaceURI string, attrs, namespaces *Map, children XMLValue, isReadonly bool) *XMLElement {
	if attrs == nil {
		attrs = NewXMLStringMap(tc, isReadonly, nil)
	}
	if namespaces == nil {
		namespaces = NewXMLStringMap(tc, isReadonly, nil)
	}
	if children == nil {
		children = NewXMLText("")
	}
	children = NewNormalizedXMLSequence([]XMLValue{children})
	ty := semtypes.XMLElement
	if isReadonly {
		ty = semtypes.ReadonlyXMLElement
	}
	return &XMLElement{Prefix: prefix, LocalName: localName, NamespaceURI: namespaceURI, Attributes: attrs, Namespaces: namespaces, Children: children, semType: ty, isReadonly: isReadonly}
}

func NewXMLStringMap(tc semtypes.Context, isReadonly bool, entries []MapEntry) *Map {
	md := semtypes.NewMappingDefinition()
	ty := md.Define(tc.Env(), nil, semtypes.String)
	return NewMap(ty, semtypes.ToMappingAtomicType(tc, ty), isReadonly, entries)
}

func NewXMLProcessingInstruction(target, data string, isReadonly bool) *XMLProcessingInstruction {
	ty := semtypes.XMLProcessingInstruction
	if isReadonly {
		ty = semtypes.ReadonlyXMLProcessingInstruction
	}
	return &XMLProcessingInstruction{Target: target, Data: data, semType: ty, isReadonly: isReadonly}
}

func NewValidatedXMLProcessingInstruction(target, data string, isReadonly bool) *XMLProcessingInstruction {
	if err := ValidateXMLProcessingInstruction(target, data); err != nil {
		panic(NewErrorWithMessage(err.Error()))
	}
	return NewXMLProcessingInstruction(target, data, isReadonly)
}

func NewXMLText(body string) *XMLText {
	return &XMLText{Body: body, semType: semtypes.XMLText}
}

func NewXMLComment(body string, isReadonly bool) *XMLComment {
	ty := semtypes.XMLComment
	if isReadonly {
		ty = semtypes.ReadonlyXMLComment
	}
	return &XMLComment{Body: body, semType: ty, isReadonly: isReadonly}
}

func NewValidatedXMLComment(body string, isReadonly bool) *XMLComment {
	if err := ValidateXMLComment(body); err != nil {
		panic(NewErrorWithMessage(err.Error()))
	}
	return NewXMLComment(body, isReadonly)
}

func xmlSequenceProperties(children []XMLValue) (semtypes.SemType, bool) {
	var childUnion = semtypes.Never
	isReadonly := true
	for _, child := range children {
		childUnion = semtypes.Union(childUnion, child.Type())
		isReadonly = isReadonly && child.Readonly()
	}
	return childUnion, isReadonly
}

func newXMLSequence(children []XMLValue, itemType semtypes.SemType, isReadonly bool) *XMLSequence {
	return &XMLSequence{
		Children:   children,
		semType:    semtypes.XMLSequence(itemType),
		isReadonly: isReadonly,
	}
}

// ConcatXML concatenates two canonical XML values.
func ConcatXML(left, right XMLValue) XMLValue {
	leftItems, rightItems := left.IterItems(), right.IterItems()
	if len(leftItems) == 0 {
		return right
	}
	if len(rightItems) == 0 {
		return left
	}
	leftSeq, ok := left.(*XMLSequence)
	if !ok {
		return NewNormalizedXMLSequence([]XMLValue{left, right})
	}
	if _, leftText := leftItems[len(leftItems)-1].(*XMLText); leftText {
		if _, rightText := rightItems[0].(*XMLText); rightText {
			return NewNormalizedXMLSequence([]XMLValue{left, right})
		}
	}
	itemType := semtypes.XMLItemType(leftSeq.semType)
	isReadonly := leftSeq.isReadonly
	for _, item := range rightItems {
		itemType = semtypes.Union(itemType, item.Type())
		isReadonly = isReadonly && item.Readonly()
	}
	children := make([]XMLValue, len(leftSeq.Children), len(leftSeq.Children)+len(rightItems))
	copy(children, leftSeq.Children)
	children = append(children, rightItems...)
	return newXMLSequence(children, itemType, isReadonly)
}

// NewNormalizedXMLSequence builds the canonical XML representation for items.
func NewNormalizedXMLSequence(items []XMLValue) XMLValue {
	normalized := make([]XMLValue, 0, len(items))
	var pendingText strings.Builder
	flushText := func() {
		if pendingText.Len() == 0 {
			return
		}
		normalized = append(normalized, NewXMLText(pendingText.String()))
		pendingText.Reset()
	}
	var appendItem func(XMLValue)
	appendItem = func(item XMLValue) {
		if item == nil {
			return
		}
		if seq, ok := item.(*XMLSequence); ok {
			for _, child := range seq.Children {
				appendItem(child)
			}
			return
		}
		if text, ok := item.(*XMLText); ok {
			if text.Body != "" {
				pendingText.WriteString(text.Body)
			}
			return
		}
		flushText()
		normalized = append(normalized, item)
	}
	for _, item := range items {
		appendItem(item)
	}
	flushText()
	switch len(normalized) {
	case 0:
		return NewXMLText("")
	case 1:
		return normalized[0]
	default:
		itemType, isReadonly := xmlSequenceProperties(normalized)
		return newXMLSequence(normalized, itemType, isReadonly)
	}
}
