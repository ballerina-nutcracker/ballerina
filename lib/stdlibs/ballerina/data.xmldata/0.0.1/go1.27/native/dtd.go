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

package native

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	// Woodstox's limits, which jBallerina keeps: entity expansions in a document, nested ones included, and
	// their nesting depth.
	maxEntityExpansions = 100000
	maxEntityDepth      = 500
	// maxEntityText bounds the characters entity expansions add to a document, which Woodstox leaves unbounded.
	maxEntityText = 1 << 26
)

// dtdEntity is an entity declared in the internal subset; value has its character references expanded.
type dtdEntity struct {
	value    string
	external bool
}

// entityCost is what one reference to a general entity adds to a document: its expansions, nested ones
// included, their nesting depth and its replacement text's length, saturating past their limits. message is
// set if the entity cannot expand.
type entityCost struct {
	expansions int
	depth      int
	length     int
	message    string
}

// attributeDefault is a default attribute; length is its value's length in characters.
type attributeDefault struct {
	name   xml.Name
	value  string
	length int
}

// dtd is a parsed internal subset. General entities expand only when the document references them, and every
// expansion, in the subset or the document, counts towards the limits.
type dtd struct {
	entities   map[string]dtdEntity
	parameters map[string]dtdEntity
	// declared holds each element's declared attributes, as the first declaration of an attribute is binding.
	declared  map[xml.Name]map[xml.Name]bool
	defaults  map[xml.Name][]attributeDefault
	costs     map[string]entityCost
	activePEs map[string]bool
	// expanded holds the replacement text of each entity the document references, for the decoder.
	expanded   map[string]string
	expansions int
	length     int
}

type dtdScanner struct {
	s string
	i int
}

func (d *dtdScanner) eof() bool { return d.i >= len(d.s) }

func (d *dtdScanner) peek() rune {
	r, _ := utf8.DecodeRuneInString(d.s[d.i:])
	return r
}

func (d *dtdScanner) skipSpace() bool {
	start := d.i
	for !d.eof() && strings.IndexByte(xmlWhitespace, d.s[d.i]) >= 0 {
		d.i++
	}
	return d.i > start
}

func (d *dtdScanner) unexpected(expected string) string {
	if d.eof() {
		return "Unexpected end of DOCTYPE declaration; " + expected
	}
	c := d.peek()
	return fmt.Sprintf("Unexpected character '%c' (code %d) (%s)", c, c, expected)
}

func (d *dtdScanner) name() (string, string) {
	start := d.i
	for !d.eof() {
		c, size := utf8.DecodeRuneInString(d.s[d.i:])
		if !isNameRune(c, d.i == start) {
			break
		}
		d.i += size
	}
	if d.i == start {
		return "", d.unexpected("expected a name start character")
	}
	return d.s[start:d.i], ""
}

func (d *dtdScanner) literal() (string, string) {
	if d.eof() || d.s[d.i] != '"' && d.s[d.i] != '\'' {
		return "", d.unexpected("expected a quoted literal")
	}
	end := strings.IndexByte(d.s[d.i+1:], d.s[d.i])
	if end < 0 {
		return "", "Unexpected end of DOCTYPE declaration; unclosed quoted literal"
	}
	value := d.s[d.i+1 : d.i+1+end]
	d.i += end + 2
	return value, ""
}

func (d *dtdScanner) keyword(word string) bool {
	if !strings.HasPrefix(d.s[d.i:], word) {
		return false
	}
	d.i += len(word)
	return true
}

// externalID reads an ExternalID after its keyword; the identified DTD or entity is never loaded.
func (d *dtdScanner) externalID(public bool) string {
	if !d.skipSpace() {
		return d.unexpected("expected whitespace")
	}
	if _, message := d.literal(); message != "" {
		return message
	}
	if public {
		if !d.skipSpace() {
			return d.unexpected("expected whitespace")
		}
		if _, message := d.literal(); message != "" {
			return message
		}
	}
	return ""
}

func isNameRune(c rune, first bool) bool {
	if first {
		return c == ':' || c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= 0xC0 && c != 0xD7 && c != 0xF7 && isXMLNameStart(c)
	}
	return isNameRune(c, true) || c == '-' || c == '.' || c >= '0' && c <= '9' || c == 0xB7 || c >= 0x300 && c <= 0x36F || c == 0x203F || c == 0x2040
}

func isXMLNameStart(c rune) bool {
	return c <= 0x2FF || c >= 0x370 && c <= 0x37D || c >= 0x37F && c <= 0x1FFF || c == 0x200C || c == 0x200D || c >= 0x2070 && c <= 0x218F || c >= 0x2C00 && c <= 0x2FEF || c >= 0x3001 && c <= 0xD7FF || c >= 0xF900 && c <= 0xFDCF || c >= 0xFDF0 && c <= 0xFFFD || c >= 0x10000 && c <= 0xEFFFF
}

// parseDoctype checks a DOCTYPE declaration's markup up to its internal subset, which it returns unchecked.
func parseDoctype(raw string) (string, string) {
	d := &dtdScanner{s: raw, i: len("<!DOCTYPE")}
	if !d.eof() && isNameRune(d.peek(), false) {
		word, _ := d.name()
		return "", fmt.Sprintf("Unrecognized XML directive '<!DOCTYPE%s' (misspelled DOCTYPE?).", word)
	}
	d.skipSpace()
	if _, message := d.name(); message != "" {
		return "", message
	}
	if d.skipSpace() {
		switch {
		case d.keyword("SYSTEM"):
			if message := d.externalID(false); message != "" {
				return "", message
			}
		case d.keyword("PUBLIC"):
			if message := d.externalID(true); message != "" {
				return "", message
			}
		}
		d.skipSpace()
	}
	subset := ""
	if !d.eof() && d.s[d.i] == '[' {
		end := strings.LastIndexByte(raw, ']')
		if end < d.i {
			return "", "Unexpected end of DOCTYPE declaration; expected ']' to end the internal subset"
		}
		subset = raw[d.i+1 : end]
		d.i = end + 1
		d.skipSpace()
	}
	if d.i != len(raw)-1 {
		return "", d.unexpected("expected '>' to end the DOCTYPE declaration")
	}
	return subset, ""
}

// parseInternalSubset checks the markup declarations of an internal subset, keeping the first declaration of
// each entity as XML 1.0 4.2 requires.
func parseInternalSubset(subset string) (*dtd, string) {
	t := &dtd{
		entities:   map[string]dtdEntity{},
		parameters: map[string]dtdEntity{},
		declared:   map[xml.Name]map[xml.Name]bool{},
		defaults:   map[xml.Name][]attributeDefault{},
		costs:      map[string]entityCost{},
		activePEs:  map[string]bool{},
		expanded:   map[string]string{},
	}
	if message := t.parseDeclarations(subset); message != "" {
		return nil, message
	}
	return t, ""
}

func (t *dtd) parseDeclarations(text string) string {
	d := &dtdScanner{s: text}
	for d.skipSpace(); !d.eof(); d.skipSpace() {
		var message string
		switch {
		case d.keyword("<!--"):
			end := strings.Index(d.s[d.i:], "-->")
			if end < 0 {
				return "Unexpected end of DOCTYPE declaration; unclosed comment"
			}
			if err := values.ValidateXMLComment(d.s[d.i : d.i+end]); err != nil {
				return err.Error()
			}
			d.i += end + len("-->")
		case d.keyword("<?"):
			message = d.processingInstruction()
		case d.keyword("<!ENTITY"):
			message = d.entityDecl(t)
		case d.keyword("<!ATTLIST"):
			message = d.attlistDecl(t)
		case d.keyword("<!ELEMENT"), d.keyword("<!NOTATION"):
			message = d.declaration()
		case d.keyword("%"):
			message = d.parameterReference(t)
		default:
			message = d.unexpected("in internal DTD subset; expected a '<' to start a directive")
		}
		if message != "" {
			return message
		}
	}
	return ""
}

// parameterReference expands a parameter entity reference between declarations to the declarations it holds.
func (d *dtdScanner) parameterReference(t *dtd) string {
	name, message := d.name()
	if message != "" {
		return message
	}
	if !d.keyword(";") {
		return d.unexpected("expected ';' to end the parameter entity reference")
	}
	entity, declared := t.parameters[name]
	switch {
	case !declared:
		// As in jBallerina, XML 1.0 4.1 lets a document that is not standalone reference an undeclared one.
		return ""
	case entity.external:
		return fmt.Sprintf("External parameter entity %q is not supported", name)
	case t.activePEs[name]:
		return fmt.Sprintf("Illegal entity expansion: entity %q expands itself recursively", name)
	case len(t.activePEs) >= maxEntityDepth:
		return fmt.Sprintf("Maximum entity expansion depth limit (%d) exceeded", maxEntityDepth)
	}
	if message := t.count(1, utf8.RuneCountInString(entity.value)); message != "" {
		return message
	}
	t.activePEs[name] = true
	defer delete(t.activePEs, name)
	return t.parseDeclarations(entity.value)
}

// count adds expansions, and the characters they add, to the document's totals.
func (t *dtd) count(expansions, length int) string {
	t.expansions += expansions
	if t.expansions > maxEntityExpansions {
		return fmt.Sprintf("Maximum entity expansion count limit (%d) exceeded", maxEntityExpansions)
	}
	t.length += length
	if t.length > maxEntityText {
		return fmt.Sprintf("Maximum entity expansion length limit (%d characters) exceeded", maxEntityText)
	}
	return ""
}

func (d *dtdScanner) processingInstruction() string {
	end := strings.Index(d.s[d.i:], "?>")
	if end < 0 {
		return "Unexpected end of DOCTYPE declaration; unclosed processing instruction"
	}
	pi := d.s[d.i : d.i+end]
	d.i += end + len("?>")
	target, content := pi, ""
	if split := strings.IndexAny(pi, xmlWhitespace); split >= 0 {
		target, content = pi[:split], strings.TrimLeft(pi[split:], xmlWhitespace)
	}
	if err := values.ValidateXMLProcessingInstruction(target, content); err != nil {
		return err.Error()
	}
	return ""
}

// attlistDecl reads an attribute-list declaration for the attribute defaults it declares.
func (d *dtdScanner) attlistDecl(t *dtd) string {
	if !d.skipSpace() {
		return d.unexpected("expected whitespace")
	}
	element, message := d.name()
	if message != "" {
		return message
	}
	for {
		spaced := d.skipSpace()
		if d.keyword(">") {
			return ""
		}
		if !spaced {
			return d.unexpected("expected whitespace")
		}
		name, message := d.name()
		if message != "" {
			return message
		}
		if !d.skipSpace() {
			return d.unexpected("expected whitespace")
		}
		cdata, message := d.attributeType()
		if message != "" {
			return message
		}
		if !d.skipSpace() {
			return d.unexpected("expected whitespace")
		}
		elementName, attribute := lexicalName(element), attributeDefault{name: lexicalName(name)}
		declared := t.declared[elementName]
		if declared == nil {
			declared = map[xml.Name]bool{}
			t.declared[elementName] = declared
		}
		first := !declared[attribute.name]
		declared[attribute.name] = true
		switch {
		case d.keyword("#REQUIRED"), d.keyword("#IMPLIED"):
			continue
		case d.keyword("#FIXED"):
			if !d.skipSpace() {
				return d.unexpected("expected whitespace")
			}
		}
		value, message := d.literal()
		if message != "" {
			return message
		}
		if attribute.value, message = t.attributeValue(value, cdata); message != "" {
			return message
		}
		attribute.length = utf8.RuneCountInString(attribute.value)
		if first {
			t.defaults[elementName] = append(t.defaults[elementName], attribute)
		}
	}
}

// attributeType reads an attribute type, reporting whether it is CDATA, whose values are not normalized further.
func (d *dtdScanner) attributeType() (bool, string) {
	if !d.eof() && d.s[d.i] == '(' {
		return false, d.enumeration()
	}
	word, message := d.name()
	if message != "" {
		return false, message
	}
	switch word {
	case "CDATA":
		return true, ""
	case "ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES", "NMTOKEN", "NMTOKENS":
		return false, ""
	case "NOTATION":
		if !d.skipSpace() {
			return false, d.unexpected("expected whitespace")
		}
		return false, d.enumeration()
	}
	return false, fmt.Sprintf("Unrecognized attribute type %q", word)
}

func (d *dtdScanner) enumeration() string {
	if !d.keyword("(") {
		return d.unexpected("expected '(' to start an enumeration")
	}
	end := strings.IndexByte(d.s[d.i:], ')')
	if end < 0 {
		return "Unexpected end of DOCTYPE declaration; unclosed enumeration"
	}
	d.i += end + 1
	return ""
}

// lexicalName splits a name as encoding/xml's RawToken does, keeping its prefix in Space.
func lexicalName(name string) xml.Name {
	if prefix, local, ok := strings.Cut(name, ":"); ok && prefix != "" && local != "" {
		return xml.Name{Space: prefix, Local: local}
	}
	return xml.Name{Local: name}
}

// declaration skips an element or notation declaration, whose content the reader does not use.
func (d *dtdScanner) declaration() string {
	if !d.skipSpace() {
		return d.unexpected("expected whitespace")
	}
	for !d.eof() {
		switch d.s[d.i] {
		case '>':
			d.i++
			return ""
		case '"', '\'':
			if _, message := d.literal(); message != "" {
				return message
			}
		default:
			d.i++
		}
	}
	return "Unexpected end of DOCTYPE declaration; unclosed markup declaration"
}

func (d *dtdScanner) entityDecl(t *dtd) string {
	if !d.skipSpace() {
		return d.unexpected("expected whitespace")
	}
	parameter := d.keyword("%")
	if parameter && !d.skipSpace() {
		return d.unexpected("expected whitespace")
	}
	name, message := d.name()
	if message != "" {
		return message
	}
	if !d.skipSpace() {
		return d.unexpected("expected whitespace")
	}
	var entity dtdEntity
	switch {
	case d.keyword("SYSTEM"):
		message = d.externalID(false)
		entity.external = true
	case d.keyword("PUBLIC"):
		message = d.externalID(true)
		entity.external = true
	default:
		var value string
		if value, message = d.literal(); message == "" {
			entity.value, message = entityValue(value)
		}
	}
	if message != "" {
		return message
	}
	if entity.external && !parameter && d.skipSpace() && d.keyword("NDATA") {
		if !d.skipSpace() {
			return d.unexpected("expected whitespace")
		}
		if _, message := d.name(); message != "" {
			return message
		}
	}
	d.skipSpace()
	if !d.keyword(">") {
		return d.unexpected("expected '>' to end the entity declaration")
	}
	entities := t.entities
	if parameter {
		entities = t.parameters
	}
	if _, declared := entities[name]; !declared {
		entities[name] = entity
	}
	return ""
}

// entityValue expands the character references in an entity value literal, leaving entity references for use.
func entityValue(literal string) (string, string) {
	var b strings.Builder
	for i := 0; i < len(literal); {
		switch literal[i] {
		case '%':
			return "", "Parameter entity references are not allowed in internal subset markup declarations"
		case '&':
			end := strings.IndexByte(literal[i:], ';')
			if end < 0 {
				return "", "Unexpected end of entity value; expected ';' to end the entity reference"
			}
			ref := literal[i+1 : i+end]
			if strings.HasPrefix(ref, "#") {
				c, message := charRef(ref[1:])
				if message != "" {
					return "", message
				}
				b.WriteRune(c)
			} else if first, _ := utf8.DecodeRuneInString(ref); ref == "" || !isNameRune(first, true) {
				return "", "Illegal entity reference &" + ref + ";"
			} else {
				b.WriteString(literal[i : i+end+1])
			}
			i += end + 1
		default:
			b.WriteByte(literal[i])
			i++
		}
	}
	if err := values.ValidateXMLCharacters(b.String()); err != nil {
		return "", err.Error()
	}
	return b.String(), ""
}

func charRef(ref string) (rune, string) {
	base, digits := 10, ref
	if strings.HasPrefix(ref, "x") {
		base, digits = 16, ref[1:]
	}
	n, err := strconv.ParseUint(digits, base, 32)
	if err != nil || values.ValidateXMLCharacters(string(rune(n))) != nil || n > utf8.MaxRune || n >= 0xD800 && n <= 0xDFFF {
		return 0, fmt.Sprintf("Illegal character entity &#%s;", ref)
	}
	return rune(n), ""
}

// walkReferences passes each run of literal text in value, then each reference in it, to visit.
func walkReferences(value, unterminated string, visit func(text, ref string) string) string {
	for value != "" {
		start := strings.IndexByte(value, '&')
		if start < 0 {
			return visit(value, "")
		}
		if start > 0 {
			if message := visit(value[:start], ""); message != "" {
				return message
			}
		}
		end := strings.IndexByte(value[start:], ';')
		if end < 0 {
			return unterminated
		}
		if message := visit("", value[start+1:start+end]); message != "" {
			return message
		}
		value = value[start+end+1:]
	}
	return ""
}

// resolve finds what a reference to a general entity costs without expanding it.
func (t *dtd) resolve(name string, active map[string]bool) entityCost {
	if cost, ok := t.costs[name]; ok {
		return cost
	}
	if active[name] {
		return entityCost{message: fmt.Sprintf("Illegal entity expansion: entity %q expands itself recursively", name)}
	}
	if len(active) >= maxEntityDepth {
		return entityCost{message: fmt.Sprintf("Maximum entity expansion depth limit (%d) exceeded", maxEntityDepth)}
	}
	cost := entityCost{expansions: 1, depth: 1}
	entity, declared := t.entities[name]
	switch {
	case !declared:
		cost.message = fmt.Sprintf("Undeclared general entity %q", name)
	case entity.external:
		cost.message = fmt.Sprintf("External entity %q is not supported", name)
	case strings.ContainsRune(entity.value, '<'):
		cost.message = fmt.Sprintf("Entity %q contains markup, which is not supported", name)
	default:
		active[name] = true
		cost.message = walkReferences(entity.value, fmt.Sprintf("Entity %q has an unterminated reference in its replacement text", name), func(text, ref string) string {
			switch {
			case ref == "":
				cost.length = min(cost.length+utf8.RuneCountInString(text), maxEntityText+1)
			case strings.HasPrefix(ref, "#"):
				_, message := charRef(ref[1:])
				cost.length = min(cost.length+1, maxEntityText+1)
				return message
			case predefinedEntities[ref] != "":
				cost.length = min(cost.length+1, maxEntityText+1)
			default:
				nested := t.resolve(ref, active)
				if nested.message != "" {
					return nested.message
				}
				cost.expansions = min(cost.expansions+nested.expansions, maxEntityExpansions+1)
				cost.depth = max(cost.depth, nested.depth+1)
				cost.length = min(cost.length+nested.length, maxEntityText+1)
			}
			return ""
		})
		delete(active, name)
		if cost.message == "" && cost.depth > maxEntityDepth {
			cost.message = fmt.Sprintf("Maximum entity expansion depth limit (%d) exceeded", maxEntityDepth)
		}
	}
	t.costs[name] = cost
	return cost
}

// write expands an entity resolve has accepted into b.
func (t *dtd) write(b *strings.Builder, name string) {
	walkReferences(t.entities[name].value, "", func(text, ref string) string {
		switch {
		case ref == "":
			b.WriteString(text)
		case strings.HasPrefix(ref, "#"):
			c, _ := charRef(ref[1:])
			b.WriteRune(c)
		case predefinedEntities[ref] != "":
			b.WriteString(predefinedEntities[ref])
		default:
			t.write(b, ref)
		}
		return ""
	})
}

// reference counts a reference in the document to a general entity and gives the decoder its replacement
// text. A reference the decoder cannot expand is left for it to report, with referenceError's message.
func (t *dtd) reference(name string) string {
	if _, declared := t.entities[name]; !declared {
		return ""
	}
	cost := t.resolve(name, map[string]bool{})
	if cost.message != "" {
		return ""
	}
	if message := t.count(cost.expansions, cost.length); message != "" {
		return message
	}
	if _, ok := t.expanded[name]; !ok {
		var b strings.Builder
		t.write(&b, name)
		t.expanded[name] = b.String()
	}
	return ""
}

func (t *dtd) referenceError(name string) string {
	return t.costs[name].message
}

var attributeSpace = strings.NewReplacer("\r\n", " ", "\t", " ", "\n", " ", "\r", " ")

// attributeValue expands and normalizes an attribute's default value as XML 1.0 3.3.3 describes.
func (t *dtd) attributeValue(literal string, cdata bool) (string, string) {
	if strings.ContainsRune(literal, '<') {
		return "", "Unexpected '<' in attribute value"
	}
	var b strings.Builder
	message := walkReferences(literal, "Unexpected end of attribute value; expected ';' to end the entity reference", func(text, ref string) string {
		switch {
		case ref == "":
			b.WriteString(attributeSpace.Replace(text))
		case strings.HasPrefix(ref, "#"):
			c, message := charRef(ref[1:])
			b.WriteRune(c)
			return message
		case predefinedEntities[ref] != "":
			b.WriteString(predefinedEntities[ref])
		default:
			cost := t.resolve(ref, map[string]bool{})
			if cost.message != "" {
				return cost.message
			}
			if message := t.count(cost.expansions, cost.length); message != "" {
				return message
			}
			var text strings.Builder
			t.write(&text, ref)
			b.WriteString(attributeSpace.Replace(text.String()))
		}
		return ""
	})
	if message != "" {
		return "", message
	}
	value := b.String()
	if err := values.ValidateXMLCharacters(value); err != nil {
		return "", err.Error()
	}
	if !cdata {
		value = strings.Join(strings.FieldsFunc(value, func(c rune) bool { return c == ' ' }), " ")
	}
	return value, ""
}

// applyDefaults counts each default attribute an element is given towards the entity text limit, as the
// element holds a copy of it.
func (t *dtd) applyDefaults(element xml.Name, attrs []xml.Attr) string {
	for _, attribute := range t.defaults[element] {
		if attribute.length == 0 || slices.ContainsFunc(attrs, func(given xml.Attr) bool { return given.Name == attribute.name }) {
			continue
		}
		if message := t.count(0, attribute.length); message != "" {
			return message
		}
	}
	return ""
}

// attributeDefaults returns the default attributes of each element, by its lexical name.
func (t *dtd) attributeDefaults() map[xml.Name][]xml.Attr {
	defaults := map[xml.Name][]xml.Attr{}
	for element, attributes := range t.defaults {
		for _, attribute := range attributes {
			defaults[element] = append(defaults[element], xml.Attr{Name: attribute.name, Value: attribute.value})
		}
	}
	return defaults
}

var predefinedEntities = map[string]string{"lt": "<", "gt": ">", "amp": "&", "apos": "'", "quot": `"`}
