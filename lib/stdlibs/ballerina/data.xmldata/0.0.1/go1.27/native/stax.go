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

// A pull reader that reports encoding/xml tokens as the StAX events jBallerina's XmlParser consumes: the same
// tokenizer the interpreter's own XML parser uses, with namespace resolution and CDATA sections reported as
// their own events.

package native

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/values"
)

type staxEvent int

const (
	eventNone staxEvent = iota
	eventStartElement
	eventEndElement
	eventCharacters
	eventCData
	eventComment
	eventProcessingInstruction
	eventDTD
	eventEndDocument
)

type staxName struct {
	ns     string
	local  string
	prefix string
}

type staxAttribute struct {
	name  staxName
	value string
}

type staxReader struct {
	input      string
	decoder    *xml.Decoder
	scopes     []map[string]string
	event      staxEvent
	name       staxName
	attributes []staxAttribute
	text       string
	depth      int
	seenRoot   bool
	finished   bool
	open       []xml.Name
}

func newStaxReader(input string) *staxReader {
	decoder := xml.NewDecoder(strings.NewReader(input))
	decoder.Strict = true
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	return &staxReader{
		input:   input,
		decoder: decoder,
		scopes:  []map[string]string{{"xml": values.XMLNamespaceURI}},
	}
}

func (r *staxReader) hasNext() bool {
	return !r.finished
}

func (r *staxReader) eventType() staxEvent {
	return r.event
}

func (r *staxReader) next() (staxEvent, error) {
	for {
		offset := r.decoder.InputOffset()
		token, err := r.decoder.RawToken()
		if errors.Is(err, io.EOF) {
			if r.depth > 0 || !r.seenRoot {
				return eventNone, r.syntaxError("unexpected EOF")
			}
			r.finished = true
			r.event = eventEndDocument
			return r.event, nil
		}
		if err != nil {
			return eventNone, err
		}
		event, skip, err := r.accept(token, offset)
		if err != nil {
			return eventNone, err
		}
		if !skip {
			r.event = event
			return event, nil
		}
	}
}

func (r *staxReader) syntaxError(message string) error {
	line, _ := r.decoder.InputPos()
	return &xml.SyntaxError{Msg: message, Line: line}
}

func (r *staxReader) accept(token xml.Token, offset int64) (staxEvent, bool, error) {
	switch t := token.(type) {
	case xml.StartElement:
		if r.depth == 0 && r.seenRoot {
			return eventNone, false, r.syntaxError("multiple root elements")
		}
		if err := r.startElement(t); err != nil {
			return eventNone, false, err
		}
		r.depth++
		r.seenRoot = true
		r.open = append(r.open, t.Name)
		return eventStartElement, false, nil
	case xml.EndElement:
		if len(r.open) == 0 {
			return eventNone, false, r.syntaxError(fmt.Sprintf("unexpected end element </%s>", qualified(t.Name)))
		}
		if open := r.open[len(r.open)-1]; open != t.Name {
			return eventNone, false, r.syntaxError(fmt.Sprintf("element <%s> closed by </%s>", qualified(open), qualified(t.Name)))
		}
		r.open = r.open[:len(r.open)-1]
		name, err := r.resolve(t.Name, true)
		if err != nil {
			return eventNone, false, err
		}
		r.name = name
		r.attributes = nil
		r.scopes = r.scopes[:len(r.scopes)-1]
		r.depth--
		return eventEndElement, false, nil
	case xml.CharData:
		text := string(t)
		if r.depth == 0 {
			if strings.TrimSpace(text) != "" {
				return eventNone, false, r.syntaxError(fmt.Sprintf("unexpected character data %q outside the root element", strings.TrimSpace(text)))
			}
			return eventNone, true, nil
		}
		r.text = text
		if strings.HasPrefix(r.input[offset:], "<![CDATA[") {
			return eventCData, false, nil
		}
		return eventCharacters, false, nil
	case xml.Comment:
		r.text = string(t)
		return eventComment, false, nil
	case xml.ProcInst:
		if t.Target == "xml" {
			return eventNone, true, nil
		}
		r.text = string(t.Inst)
		return eventProcessingInstruction, false, nil
	case xml.Directive:
		return eventDTD, false, nil
	}
	return eventNone, true, nil
}

func (r *staxReader) startElement(t xml.StartElement) error {
	scope := map[string]string{}
	for k, v := range r.scopes[len(r.scopes)-1] {
		scope[k] = v
	}
	for _, attr := range t.Attr {
		switch {
		case attr.Name.Space == "xmlns":
			scope[attr.Name.Local] = attr.Value
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
			scope[""] = attr.Value
		}
	}
	r.scopes = append(r.scopes, scope)
	name, err := r.resolve(t.Name, true)
	if err != nil {
		return err
	}
	r.name = name
	r.attributes = r.attributes[:0]
	for _, attr := range t.Attr {
		if attr.Name.Space == "xmlns" || attr.Name.Space == "" && attr.Name.Local == "xmlns" {
			continue
		}
		attrName, err := r.resolve(attr.Name, false)
		if err != nil {
			return err
		}
		r.attributes = append(r.attributes, staxAttribute{name: attrName, value: attr.Value})
	}
	return nil
}

func (r *staxReader) resolve(name xml.Name, element bool) (staxName, error) {
	scope := r.scopes[len(r.scopes)-1]
	if name.Space == "" {
		if element {
			return staxName{ns: scope[""], local: name.Local}, nil
		}
		return staxName{local: name.Local}, nil
	}
	uri, ok := scope[name.Space]
	if !ok {
		return staxName{}, r.syntaxError(fmt.Sprintf("undeclared namespace prefix %q", name.Space))
	}
	return staxName{ns: uri, local: name.Local, prefix: name.Space}, nil
}

func qualified(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}
