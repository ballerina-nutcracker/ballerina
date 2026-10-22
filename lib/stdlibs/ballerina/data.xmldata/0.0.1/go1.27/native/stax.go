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
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/values"
)

// maxElementDepth is jBallerina's element nesting limit, which also bounds the recursion that builds the value.
const maxElementDepth = 1000

var errElementDepthExceeded = fmt.Errorf("Maximum Element Depth limit (%d) Exceeded", maxElementDepth) //nolint:staticcheck // jBallerina's message verbatim

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
	// declarationError is reported by the first next call.
	declarationError string
}

// xmlWhitespace is XML 1.0's S production, narrower than strings.TrimSpace's Unicode spaces.
const xmlWhitespace = " \t\r\n"

func newStaxReader(input string) *staxReader {
	// encoding/xml supports only version 1.0, so the reader checks the declaration itself and hands the decoder
	// the input with it blanked, keeping every offset and line.
	end := declarationEnd(input)
	blanked := []byte(input)
	for i := range end {
		if blanked[i] != '\n' {
			blanked[i] = ' '
		}
	}
	decoder := xml.NewDecoder(strings.NewReader(string(blanked)))
	decoder.Strict = true
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	r := &staxReader{
		input:   input,
		decoder: decoder,
		scopes:  []map[string]string{{"xml": values.XMLNamespaceURI}},
	}
	if end > 0 {
		r.declarationError = xmlDeclarationError(input[len("<?xml") : end-len("?>")])
	}
	return r
}

// declarationEnd returns the offset just past the XML declaration that opens input, or 0 if there is none.
func declarationEnd(input string) int {
	if !strings.HasPrefix(input, "<?xml") || len(input) > len("<?xml") && !strings.ContainsRune(xmlWhitespace+"?", rune(input[len("<?xml")])) {
		return 0
	}
	end := strings.Index(input, "?>")
	if end < 0 {
		return 0
	}
	return end + len("?>")
}

func (r *staxReader) hasNext() bool {
	return !r.finished
}

func (r *staxReader) eventType() staxEvent {
	return r.event
}

func (r *staxReader) next() (staxEvent, error) {
	if message := r.declarationError; message != "" {
		r.declarationError = ""
		return eventNone, r.syntaxError(message)
	}
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
		event, skip, err := r.accept(token, r.input[offset:r.decoder.InputOffset()])
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

// accept checks a token against the XML 1.0 rules encoding/xml leaves to the caller; raw is the token's markup.
func (r *staxReader) accept(token xml.Token, raw string) (staxEvent, bool, error) {
	switch t := token.(type) {
	case xml.StartElement:
		if r.depth == 0 && r.seenRoot {
			return eventNone, false, r.syntaxError("multiple root elements")
		}
		if r.depth >= maxElementDepth {
			return eventNone, false, errElementDepthExceeded
		}
		if message := charRefError(raw); message != "" {
			return eventNone, false, r.syntaxError(message)
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
		cdata := strings.HasPrefix(raw, "<![CDATA[")
		if r.depth == 0 {
			if cdata {
				return eventNone, false, r.syntaxError("CDATA not allowed in prolog/epilog")
			}
			if trimmed := strings.Trim(text, xmlWhitespace); trimmed != "" {
				return eventNone, false, r.syntaxError(fmt.Sprintf("unexpected character data %q outside the root element", trimmed))
			}
			return eventNone, true, nil
		}
		r.text = text
		if cdata {
			return eventCData, false, nil
		}
		if message := charRefError(raw); message != "" {
			return eventNone, false, r.syntaxError(message)
		}
		return eventCharacters, false, nil
	case xml.Comment:
		r.text = string(t)
		return eventComment, false, nil
	case xml.ProcInst:
		// The opening XML declaration never reaches the decoder, so any other use of the target is reserved.
		if strings.EqualFold(t.Target, "xml") {
			return eventNone, false, r.syntaxError(fmt.Sprintf("Illegal processing instruction target (%q); xml (case insensitive) is reserved by the specs.", t.Target))
		}
		r.text = string(t.Inst)
		return eventProcessingInstruction, false, nil
	case xml.Directive:
		switch {
		case r.depth > 0:
			return eventNone, false, r.syntaxError("Unrecognized XML directive; expected CDATA or comment ('<![CDATA[' or '<!--').")
		case !strings.HasPrefix(string(t), "DOCTYPE"):
			c, _ := utf8.DecodeRune(t)
			return eventNone, false, r.syntaxError(fmt.Sprintf("Unexpected character '%c' (code %d) after '<!' (malformed comment?)", c, c))
		case r.seenRoot:
			return eventNone, false, r.syntaxError("Can not have DOCTYPE declaration in epilog")
		}
		return eventDTD, false, nil
	}
	return eventNone, true, nil
}

// charRefError checks the character references in raw markup for surrogates: encoding/xml decodes them to
// U+FFFD, while it rejects every other character a reference may not expand to.
func charRefError(raw string) string {
	for i := strings.Index(raw, "&#"); i >= 0; i = strings.Index(raw, "&#") {
		raw = raw[i+len("&#"):]
		end := strings.IndexByte(raw, ';')
		if end < 0 {
			return ""
		}
		digits, base := raw[:end], 10
		if strings.HasPrefix(digits, "x") {
			digits, base = digits[1:], 16
		}
		if n, err := strconv.ParseUint(digits, base, 32); err == nil && n >= 0xD800 && n <= 0xDFFF {
			return fmt.Sprintf("Illegal character entity: expansion character (code 0x%x", n)
		}
	}
	return ""
}

// xmlDeclarationError checks the pseudo-attributes of an XML declaration: version first, then an
// optional encoding and standalone, in that order. It returns "" for a well-formed declaration.
func xmlDeclarationError(inst string) string {
	if strings.TrimSpace(inst) == "" {
		return `Illegal processing instruction target ("xml"); xml (case insensitive) is reserved by the specs.`
	}
	unexpected := func(rest, expected string) string {
		if rest == "" {
			rest = "?" // the declaration's closing "?>"
		}
		c, _ := utf8.DecodeRuneInString(rest)
		return fmt.Sprintf("Unexpected character '%c' (code %d); expected %s", c, c, expected)
	}
	names := []string{"version", "encoding", "standalone"}
	rest, next := inst, 0
	for {
		trimmed := strings.TrimLeft(rest, xmlWhitespace)
		if trimmed == "" {
			return ""
		}
		if next > 0 && len(trimmed) == len(rest) {
			return unexpected(trimmed, "either '?' or white space")
		}
		end := strings.IndexFunc(trimmed, func(c rune) bool { return c < 'a' || c > 'z' })
		if end < 0 {
			end = len(trimmed)
		}
		name := trimmed[:end]
		found := slices.Index(names[next:], name)
		switch {
		case next == 0 && name != "version":
			return unexpected(trimmed, "keyword 'version'")
		case found < 0:
			return unexpected(trimmed, `"?>" end marker`)
		}
		next += found + 1
		rest = strings.TrimLeft(trimmed[end:], xmlWhitespace)
		if !strings.HasPrefix(rest, "=") {
			return unexpected(rest, fmt.Sprintf("'=' after '%s'", name))
		}
		rest = strings.TrimLeft(rest[1:], xmlWhitespace)
		if rest == "" || rest[0] != '\'' && rest[0] != '"' {
			return unexpected(rest, fmt.Sprintf("a quote character enclosing value for '%s'", name))
		}
		closing := strings.IndexByte(rest[1:], rest[0])
		if closing < 0 {
			return fmt.Sprintf("Missing quote around XML pseudo-attribute '%s' value", name)
		}
		value := rest[1 : 1+closing]
		rest = rest[2+closing:]
		switch {
		case value == "":
			return fmt.Sprintf("Missing XML pseudo-attribute '%s' value", name)
		case name == "version" && value != "1.0" && value != "1.1":
			return fmt.Sprintf(`Invalid XML pseudo-attribute 'version' value '%s'; expected "1.0" or "1.1"`, value)
		case name == "standalone" && value != "yes" && value != "no":
			return fmt.Sprintf(`Invalid XML pseudo-attribute 'standalone' value '%s'; expected "yes" or "no"`, value)
		}
	}
}

func (r *staxReader) startElement(t xml.StartElement) error {
	if err := r.checkName(t.Name); err != nil {
		return err
	}
	for _, attr := range t.Attr {
		if err := r.checkName(attr.Name); err != nil {
			return err
		}
	}
	scope := map[string]string{}
	for k, v := range r.scopes[len(r.scopes)-1] {
		scope[k] = v
	}
	declared := map[string]bool{}
	for _, attr := range t.Attr {
		prefix := ""
		switch {
		case attr.Name.Space == "xmlns":
			prefix = attr.Name.Local
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
		default:
			continue
		}
		if declared[prefix] {
			if prefix == "" {
				return r.syntaxError("Duplicate default namespace declaration.")
			}
			return r.syntaxError(fmt.Sprintf("Duplicate declaration for namespace prefix '%s'.", prefix))
		}
		declared[prefix] = true
		if err := values.ValidateNamespaceBinding(prefix, attr.Value); err != nil {
			return r.syntaxError(err.Error())
		}
		scope[prefix] = attr.Value
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
		for _, seen := range r.attributes {
			if seen.name.ns == attrName.ns && seen.name.local == attrName.local {
				return r.syntaxError(fmt.Sprintf("Duplicate attribute '%s'.", values.ExpandedXMLName(attrName.ns, attrName.local)))
			}
		}
		r.attributes = append(r.attributes, staxAttribute{name: attrName, value: attr.Value})
	}
	return nil
}

// checkName rejects a name with an empty prefix or local part, which encoding/xml keeps whole in Local.
func (r *staxReader) checkName(name xml.Name) error {
	if strings.HasPrefix(name.Local, ":") {
		return r.syntaxError("Unexpected character ':' (code 58) (missing namespace prefix?)")
	}
	if strings.Contains(name.Local, ":") {
		return r.syntaxError("invalid XML name: " + name.Local)
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
