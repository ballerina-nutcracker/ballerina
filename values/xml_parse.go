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
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"ballerina/semtypes"
)

const xmlNamespaceURI = "http://www.w3.org/XML/1998/namespace"

// XMLParseMode selects how ParseAsXMLValue interprets its input.
type XMLParseMode int

const (
	// XMLTemplateMode re-parses XML template content under strict rules: XML
	// directives and the reserved "xml" processing-instruction target are
	// rejected, and the resulting value is readonly.
	XMLTemplateMode XMLParseMode = iota
	// XMLLenientMode parses external XML (e.g. io:fileReadXml): directives are
	// skipped and any processing instruction is accepted. The resulting value
	// is mutable.
	XMLLenientMode
)

// FromBytes interprets a byte slice as a UTF-8 encoded string, for callers that
// read raw XML bytes and hand them to ParseAsXMLValue.
func FromBytes(data []byte) string {
	return string(data)
}

// ParseAsXMLValue parses XML content into a Ballerina XML value. The mode
// selects the strict, readonly template semantics or the lenient, mutable
// semantics used for external XML. The map<string> type used for attribute and
// namespace maps is built from the runtime context (an ephemeral, non-cyclic
// type), so the only difference between the two modes is the parsing rules.
func ParseAsXMLValue(tc semtypes.Context, content string, mode XMLParseMode) (XMLValue, error) {
	bc := newXMLBuildCtx(tc, mode)
	if mode == XMLLenientMode {
		return parseXMLLenient(bc, content)
	}
	return parseXMLStrict(bc, content)
}

// xmlBuildCtx carries the type information and readonly flag used while building
// XML values during a parse.
type xmlBuildCtx struct {
	stringMapTy     semtypes.SemType
	stringMapAtomic *semtypes.MappingAtomicType
	readonly        bool
}

func newXMLBuildCtx(tc semtypes.Context, mode XMLParseMode) *xmlBuildCtx {
	md := semtypes.NewMappingDefinition()
	stringMapTy := md.DefineMappingTypeWrapped(tc.Env(), nil, semtypes.STRING)
	return &xmlBuildCtx{
		stringMapTy:     stringMapTy,
		stringMapAtomic: semtypes.ToMappingAtomicType(tc, stringMapTy),
		readonly:        mode == XMLTemplateMode,
	}
}

// stringMap builds a map<string> from entries, always returning a non-nil map.
func (bc *xmlBuildCtx) stringMap(entries []MapEntry) *Map {
	return NewMap(bc.stringMapTy, bc.stringMapAtomic, false, entries)
}

// stringMapOrNil is stringMap but returns nil for an empty entry set, matching
// the template parser's representation of attribute/namespace-free elements.
func (bc *xmlBuildCtx) stringMapOrNil(entries []MapEntry) *Map {
	if len(entries) == 0 {
		return nil
	}
	return bc.stringMap(entries)
}

type xmlParseElement struct {
	e  *XMLElement
	ns map[string]string
}

// parseXMLStrict implements XMLTemplateMode.
func parseXMLStrict(bc *xmlBuildCtx, content string) (XMLValue, error) {
	decoder := xml.NewDecoder(strings.NewReader(content))
	decoder.Strict = true
	var stack []xmlParseElement
	var top []XMLValue
	currentNS := map[string]string{}
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xml template re-parse failed: %v (source: %q)", err, content)
		}
		var item XMLValue
		switch t := tok.(type) {
		case xml.StartElement:
			ns := cloneStringMap(currentNS)
			attrs, namespaces := bc.xmlAttrsAndNamespaces(t.Attr, ns)
			name := xmlQualifiedName(t.Name, ns, true)
			stack = append(stack, xmlParseElement{e: NewXMLElement(name, attrs, namespaces, nil, bc.readonly), ns: currentNS})
			currentNS = ns
			continue
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("xml template re-parse failed: unexpected end element </%s> (source: %q)", t.Name.Local, content)
			}
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			currentNS = last.ns
			item = last.e
		case xml.CharData:
			if len(t) == 0 {
				continue
			}
			item = NewXMLText(string(t))
		case xml.Comment:
			item = NewXMLComment(string(t), bc.readonly)
		case xml.ProcInst:
			if strings.EqualFold(t.Target, "xml") {
				return nil, fmt.Errorf("xml processing instruction target %q is reserved", t.Target)
			}
			item = NewXMLProcessingInstruction(t.Target, string(t.Inst), bc.readonly)
		case xml.Directive:
			return nil, fmt.Errorf("xml directive not allowed in template content: <!%s>", string(t))
		default:
			continue
		}
		if len(stack) > 0 {
			parent := stack[len(stack)-1].e
			parent.Children = appendXMLChild(parent.Children, item)
		} else {
			top = append(top, item)
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("xml template re-parse failed: unexpected EOF (source: %q)", content)
	}
	return collapseXMLItems(top), nil
}

func appendXMLChild(children XMLValue, item XMLValue) XMLValue {
	if children == nil {
		return item
	}
	return NewNormalizedXMLSequence([]XMLValue{children, item})
}

func collapseXMLItems(items []XMLValue) XMLValue {
	if len(items) == 0 {
		return NewXMLText("")
	}
	if len(items) == 1 {
		return items[0]
	}
	return NewNormalizedXMLSequence(items)
}

func (bc *xmlBuildCtx) xmlAttrsAndNamespaces(attrs []xml.Attr, ns map[string]string) (*Map, *Map) {
	var nsEntries []MapEntry
	for _, attr := range attrs {
		if !isXMLNSParseAttr(attr) {
			continue
		}
		key := "xmlns"
		prefix := ""
		if attr.Name.Space == "xmlns" {
			key = "xmlns:" + attr.Name.Local
			prefix = attr.Name.Local
		}
		ns[prefix] = attr.Value
		nsEntries = append(nsEntries, MapEntry{Key: key, Value: attr.Value})
	}
	var attrEntries []MapEntry
	for _, attr := range attrs {
		if isXMLNSParseAttr(attr) {
			continue
		}
		attrEntries = append(attrEntries, MapEntry{Key: xmlQualifiedName(attr.Name, ns, false), Value: attr.Value})
	}
	return bc.stringMapOrNil(attrEntries), bc.stringMapOrNil(nsEntries)
}

func isXMLNSParseAttr(attr xml.Attr) bool {
	return attr.Name.Space == "xmlns" || attr.Name.Space == "" && attr.Name.Local == "xmlns"
}

func xmlQualifiedName(name xml.Name, ns map[string]string, element bool) string {
	if name.Space == "" {
		return name.Local
	}
	bestPrefix := ""
	for prefix, uri := range ns {
		if uri != name.Space {
			continue
		}
		if prefix == "" && element && bestPrefix == "" {
			continue
		}
		if prefix > bestPrefix {
			bestPrefix = prefix
		}
	}
	if bestPrefix != "" {
		return bestPrefix + ":" + name.Local
	}
	if name.Space == xmlNamespaceURI {
		return "xml:" + name.Local
	}
	return name.Local
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// parseXMLLenient implements XMLLenientMode.
func parseXMLLenient(bc *xmlBuildCtx, content string) (XMLValue, error) {
	dec := xml.NewDecoder(strings.NewReader(content))
	items, err := parseXMLItems(bc, dec, xmlNsCtx{}, true)
	if err != nil {
		return nil, err
	}
	switch len(items) {
	case 0:
		return NewNormalizedXMLSequence(nil), nil
	case 1:
		return items[0], nil
	default:
		return NewNormalizedXMLSequence(items), nil
	}
}

// xmlNsCtx maps namespace URI to prefix, accumulated as we descend into elements.
type xmlNsCtx map[string]string

func (c xmlNsCtx) child(attrs []xml.Attr) xmlNsCtx {
	ch := make(xmlNsCtx, len(c)+4)
	for k, v := range c {
		ch[k] = v
	}
	for _, attr := range attrs {
		switch {
		case attr.Name.Space == "xmlns":
			ch[attr.Value] = attr.Name.Local
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			ch[attr.Value] = ""
		}
	}
	return ch
}

func (c xmlNsCtx) qualifiedName(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	if prefix, ok := c[name.Space]; ok {
		if prefix == "" {
			return name.Local
		}
		return prefix + ":" + name.Local
	}
	return name.Local
}

func parseXMLItems(bc *xmlBuildCtx, dec *xml.Decoder, ctx xmlNsCtx, topLevel bool) ([]XMLValue, error) {
	var items []XMLValue
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				if !topLevel {
					return nil, fmt.Errorf("unexpected end of file inside element")
				}
				return items, nil
			}
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			elem, parseErr := parseXMLElement(bc, dec, t, ctx)
			if parseErr != nil {
				return nil, parseErr
			}
			items = append(items, elem)
		case xml.EndElement:
			if topLevel {
				return nil, fmt.Errorf("unexpected end element </%s>", t.Name.Local)
			}
			return items, nil
		case xml.CharData:
			body := string(t)
			if topLevel && strings.TrimSpace(body) == "" {
				continue
			}
			items = append(items, NewXMLText(body))
		case xml.Comment:
			items = append(items, NewXMLComment(string(t), bc.readonly))
		case xml.ProcInst:
			items = append(items, NewXMLProcessingInstruction(t.Target, string(t.Inst), bc.readonly))
		case xml.Directive:
			// skip DOCTYPE and similar directives
		}
	}
}

func parseXMLElement(bc *xmlBuildCtx, dec *xml.Decoder, start xml.StartElement, parentCtx xmlNsCtx) (*XMLElement, error) {
	ctx := parentCtx.child(start.Attr)
	name := ctx.qualifiedName(start.Name)

	var attrsEntries []MapEntry
	var nsEntries []MapEntry
	for _, attr := range start.Attr {
		switch {
		case attr.Name.Space == "xmlns":
			nsEntries = append(nsEntries, MapEntry{Key: "xmlns:" + attr.Name.Local, Value: attr.Value})
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			nsEntries = append(nsEntries, MapEntry{Key: "xmlns", Value: attr.Value})
		default:
			attrName := ctx.qualifiedName(attr.Name)
			attrsEntries = append(attrsEntries, MapEntry{Key: attrName, Value: attr.Value})
		}
	}

	attrs := bc.stringMap(attrsEntries)
	namespaces := bc.stringMap(nsEntries)

	children, err := parseXMLItems(bc, dec, ctx, false)
	if err != nil {
		return nil, err
	}
	var childVal XMLValue
	if len(children) > 0 {
		childVal = NewNormalizedXMLSequence(children)
	}

	return NewXMLElement(name, attrs, namespaces, childVal, bc.readonly), nil
}
