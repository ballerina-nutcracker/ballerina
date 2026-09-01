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

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type XMLParseMode int

const (
	XMLTemplateMode XMLParseMode = iota
	XMLLenientMode
	XMLStrictContentMode
)

func FromBytes(data []byte) string { return string(data) }

func ParseAsXMLValue(tc semtypes.Context, content string, mode XMLParseMode) (XMLValue, error) {
	return parseXML(newXMLBuildCtx(tc, mode), content, mode)
}

type xmlBuildCtx struct {
	tc              semtypes.Context
	stringMapTy     semtypes.SemType
	stringMapAtomic *semtypes.MappingAtomicType
	readonly        bool
}

func newXMLBuildCtx(tc semtypes.Context, mode XMLParseMode) *xmlBuildCtx {
	md := semtypes.NewMappingDefinition()
	stringMapTy := md.Define(tc.Env(), nil, semtypes.String)
	return &xmlBuildCtx{tc: tc, stringMapTy: stringMapTy, stringMapAtomic: semtypes.ToMappingAtomicType(tc, stringMapTy), readonly: mode == XMLTemplateMode}
}

func (bc *xmlBuildCtx) stringMap(entries []MapEntry) *Map {
	return NewMap(bc.stringMapTy, bc.stringMapAtomic, bc.readonly, entries)
}

type lexicalNamespaceContext map[string]string

type lexicalNamespaceUndo struct {
	prefix   string
	previous string
	hadValue bool
}

type xmlParseFrame struct {
	e        *XMLElement
	children []XMLValue
	undos    []lexicalNamespaceUndo
	qname    xml.Name
}

func resolveLexicalName(name xml.Name, ctx lexicalNamespaceContext, element bool) (string, string, string, error) {
	prefix, local := name.Space, name.Local
	if !isNCName(local) || prefix != "" && !isNCName(prefix) {
		return "", "", "", fmt.Errorf("invalid XML qualified name %q", qualifiedLexicalName(name))
	}
	if prefix == "" {
		if element {
			return "", local, ctx[""], nil
		}
		return "", local, "", nil
	}
	uri, ok := ctx[prefix]
	if !ok || uri == "" {
		return "", "", "", fmt.Errorf("undeclared XML namespace prefix %q", prefix)
	}
	return prefix, local, uri, nil
}

func qualifiedLexicalName(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}

func parseXML(bc *xmlBuildCtx, content string, mode XMLParseMode) (XMLValue, error) {
	decoder := xml.NewDecoder(strings.NewReader(content))
	decoder.Strict = true
	ctx := lexicalNamespaceContext{"xml": XMLNamespaceURI}
	var stack []xmlParseFrame
	var top []XMLValue
	appendItem := func(item XMLValue) {
		if len(stack) == 0 {
			top = append(top, item)
		} else {
			stack[len(stack)-1].children = append(stack[len(stack)-1].children, item)
		}
	}
	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			var undos []lexicalNamespaceUndo
			var nsEntries []MapEntry
			localPrefixes := map[string]bool{}
			for _, attr := range t.Attr {
				isDeclaration := attr.Name.Space == "xmlns" || attr.Name.Space == "" && attr.Name.Local == "xmlns"
				if !isDeclaration {
					continue
				}
				prefix, key := "", "xmlns"
				if attr.Name.Space == "xmlns" {
					prefix, key = attr.Name.Local, "xmlns:"+attr.Name.Local
				}
				if localPrefixes[prefix] {
					return nil, fmt.Errorf("duplicate namespace declaration for prefix %q", prefix)
				}
				if err := validateNamespaceBinding(prefix, attr.Value); err != nil {
					return nil, err
				}
				localPrefixes[prefix] = true
				previous, hadValue := ctx[prefix]
				undos = append(undos, lexicalNamespaceUndo{prefix, previous, hadValue})
				ctx[prefix] = attr.Value
				nsEntries = append(nsEntries, MapEntry{Key: key, Value: attr.Value})
			}
			prefix, local, uri, err := resolveLexicalName(t.Name, ctx, true)
			if err != nil {
				return nil, err
			}
			if uri == XMLNSNamespaceURI {
				return nil, fmt.Errorf("element name cannot use XMLNS namespace")
			}
			usedInherited := map[string]bool{}
			addInherited := func(usedPrefix string) {
				if usedPrefix == "" || usedPrefix == "xml" || localPrefixes[usedPrefix] || usedInherited[usedPrefix] {
					return
				}
				usedInherited[usedPrefix] = true
				nsEntries = append(nsEntries, MapEntry{Key: "xmlns:" + usedPrefix, Value: ctx[usedPrefix]})
			}
			addInherited(prefix)
			var attrEntries []MapEntry
			seenAttrs := map[string]bool{}
			for _, attr := range t.Attr {
				if attr.Name.Space == "xmlns" || attr.Name.Space == "" && attr.Name.Local == "xmlns" {
					continue
				}
				aPrefix, aLocal, aURI, err := resolveLexicalName(attr.Name, ctx, false)
				if err != nil {
					return nil, err
				}
				if aURI == XMLNSNamespaceURI || aPrefix == "" && aLocal == "xmlns" {
					return nil, fmt.Errorf("invalid ordinary XML attribute name")
				}
				if err := ValidateXMLCharacters(attr.Value); err != nil {
					return nil, err
				}
				key := ExpandedXMLName(aURI, aLocal)
				if seenAttrs[key] {
					return nil, fmt.Errorf("duplicate XML attribute %q", key)
				}
				seenAttrs[key] = true
				attrEntries = append(attrEntries, MapEntry{Key: key, Value: attr.Value})
				addInherited(aPrefix)
			}
			e := NewXMLElement(bc.tc, prefix, local, uri, bc.stringMap(attrEntries), bc.stringMap(nsEntries), nil, bc.readonly)
			stack = append(stack, xmlParseFrame{e: e, undos: undos, qname: t.Name})
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("unexpected XML end element")
			}
			frame := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if qualifiedLexicalName(frame.qname) != qualifiedLexicalName(t.Name) {
				return nil, fmt.Errorf("mismatched XML end element")
			}
			frame.e.Children = NewNormalizedXMLSequence(frame.children)
			for i := len(frame.undos) - 1; i >= 0; i-- {
				undo := frame.undos[i]
				if undo.hadValue {
					ctx[undo.prefix] = undo.previous
				} else {
					delete(ctx, undo.prefix)
				}
			}
			appendItem(frame.e)
		case xml.CharData:
			body := string(t)
			if err := ValidateXMLCharacters(body); err != nil {
				return nil, err
			}
			if mode == XMLLenientMode && len(stack) == 0 && strings.TrimSpace(body) == "" {
				continue
			}
			appendItem(NewXMLText(body))
		case xml.Comment:
			if err := ValidateXMLComment(string(t)); err != nil {
				return nil, err
			}
			appendItem(NewXMLComment(string(t), bc.readonly))
		case xml.ProcInst:
			if strings.EqualFold(t.Target, "xml") && mode != XMLLenientMode {
				return nil, fmt.Errorf("reserved XML processing-instruction target %q", t.Target)
			}
			if mode != XMLLenientMode {
				if err := ValidateXMLProcessingInstruction(t.Target, string(t.Inst)); err != nil {
					return nil, err
				}
			}
			appendItem(NewXMLProcessingInstruction(t.Target, string(t.Inst), bc.readonly))
		case xml.Directive:
			if mode != XMLLenientMode {
				return nil, fmt.Errorf("XML directive is not allowed in content")
			}
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("unexpected end of XML content")
	}
	return NewNormalizedXMLSequence(top), nil
}
