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

// The parser composes the scanner's tokens into one node tree per document. It applies the rules
// jBallerina's composer applies while reading: directives, tag resolution, anchors and aliases, and
// duplicate keys.

package native

import (
	"strconv"
	"strings"
)

const coreTagPrefix = "tag:yaml.org,2002:"

type nodeKind int

const (
	nodeScalar nodeKind = iota
	nodeSequence
	nodeMapping
)

type node struct {
	kind  nodeKind
	value string
	// empty marks a node with no content, such as the value of `a:`.
	empty bool
	// typed holds the value of a scalar with an explicit int, float, bool or null tag; hasTyped tells a
	// tagged null from no tagged value.
	typed    any
	hasTyped bool
	items    []*node
	pairs    []pair
	// size caches expandedSize; zero means not yet computed.
	size int
}

const maxExpandedSize = 1 << 40

type pair struct {
	key   *node
	value *node
}

type parser struct {
	s       *scanner
	options *readOptions
	anchors map[string]*node

	tagHandles     map[string]string
	versionDefined bool
	streamStarted  bool
	// sawDocumentEnd records a `...` marker, which makes an input without documents one empty document.
	sawDocumentEnd bool
	// nodes counts the nodes composed and aliasNodes the nodes that aliases add when expanded.
	nodes      int
	aliasNodes int
}

func newParser(input string, options *readOptions) *parser {
	return &parser{
		s:          newScanner(input),
		options:    options,
		anchors:    make(map[string]*node),
		tagHandles: make(map[string]string),
	}
}

func (p *parser) next() (token, error) {
	return p.s.next()
}

func (p *parser) peek() (token, error) {
	return p.s.peek()
}

func (p *parser) unexpected() error {
	return p.s.errorHere("unexpected event")
}

// nextDocument composes the next document. It returns false at the end of the stream; explicit reports
// whether the document started with `---`.
func (p *parser) nextDocument() (doc *node, explicit bool, ok bool, err error) {
	if !p.streamStarted {
		p.streamStarted = true
		if err := p.s.checkPrintable(); err != nil {
			return nil, false, false, err
		}
		if _, err := p.next(); err != nil {
			return nil, false, false, err
		}
	}
	tok, err := p.peek()
	if err != nil {
		return nil, false, false, err
	}
	for tok.kind == tokenDocumentEnd {
		if _, err := p.next(); err != nil {
			return nil, false, false, err
		}
		p.sawDocumentEnd = true
		p.tagHandles = make(map[string]string)
		p.versionDefined = false
		next, err := p.peek()
		if err != nil {
			return nil, false, false, err
		}
		if next.start.line == tok.start.line && next.kind != tokenStreamEnd && next.kind != tokenDocumentEnd &&
			next.kind != tokenDocumentStart && !isDirective(next.kind) {
			return nil, false, false, p.s.errorAt("token cannot start in the same line as the document marker", next.end)
		}
		tok = next
	}
	if tok.kind == tokenStreamEnd {
		return nil, false, false, nil
	}
	hasDirectives, err := p.parseDirectives()
	if err != nil {
		return nil, false, false, err
	}
	if tok, err = p.peek(); err != nil {
		return nil, false, false, err
	}
	if hasDirectives && tok.kind != tokenDocumentStart {
		at := tok.start
		at.column++
		return nil, false, false, p.s.errorAt("invalid directive document", at)
	}
	if tok.kind == tokenDocumentStart {
		explicit = true
		startLine := tok.start.line
		if _, err := p.next(); err != nil {
			return nil, false, false, err
		}
		if tok, err = p.peek(); err != nil {
			return nil, false, false, err
		}
		if (tok.kind == tokenBlockMappingStart || tok.kind == tokenBlockSequenceStart) && tok.start.line == startLine {
			return nil, false, false, p.s.errorHere("block collection token cannot start in the same line as the directive marker")
		}
		if tok.kind == tokenDocumentStart || tok.kind == tokenDocumentEnd || tok.kind == tokenStreamEnd ||
			isDirective(tok.kind) {
			return &node{kind: nodeScalar, empty: true}, true, true, nil
		}
	}
	p.anchors = make(map[string]*node)
	doc, err = p.parseNode(true, false)
	if err != nil {
		return nil, false, false, err
	}
	if tok, err = p.peek(); err != nil {
		return nil, false, false, err
	}
	switch tok.kind {
	case tokenDocumentStart, tokenDocumentEnd, tokenStreamEnd:
	default:
		if isDirective(tok.kind) {
			return nil, false, false, p.s.errorAt("directives are not allowed in a bare document", tok.end)
		}
		return nil, false, false, p.s.errorHere("there can only be one root event to a document")
	}
	return doc, explicit, true, nil
}

func isDirective(kind tokenKind) bool {
	return kind == tokenVersionDirective || kind == tokenTagDirective || kind == tokenReservedDirective
}

func (p *parser) parseDirectives() (bool, error) {
	found := false
	for {
		tok, err := p.peek()
		if err != nil {
			return false, err
		}
		if !isDirective(tok.kind) {
			return found, nil
		}
		found = true
		if _, err := p.next(); err != nil {
			return false, err
		}
		switch tok.kind {
		case tokenVersionDirective:
			if p.versionDefined {
				return false, p.s.errorAt("YAML document version is already defined", tok.nameEnd)
			}
			p.versionDefined = true
			version, parseErr := strconv.ParseFloat(tok.suffix, 32)
			if parseErr != nil || version < 1.0 || version >= 2.0 {
				return false, p.s.errorAt("incompatible yaml version for the 1.2 parser", tok.end)
			}
		case tokenTagDirective:
			if _, exists := p.tagHandles[tok.handle]; exists {
				return false, p.s.errorAt("duplicate tag handle", tok.nameEnd)
			}
			if tok.suffix != coreTagPrefix {
				return false, p.s.errorAt("custom tags not supported", tok.end)
			}
			p.tagHandles[tok.handle] = tok.suffix
		}
	}
}

type nodeProperties struct {
	anchor    string
	hasAnchor bool
	tag       string
	hasTag    bool
}

func (p *parser) parseProperties() (nodeProperties, error) {
	var props nodeProperties
	for {
		tok, err := p.peek()
		if err != nil {
			return props, err
		}
		switch {
		case tok.kind == tokenAnchor && !props.hasAnchor:
			props.anchor, props.hasAnchor = tok.value, true
		case tok.kind == tokenTag && !props.hasTag:
			tag, err := p.resolveTag(tok)
			if err != nil {
				return props, err
			}
			props.tag, props.hasTag = tag, true
		default:
			return props, nil
		}
		if _, err := p.next(); err != nil {
			return props, err
		}
	}
}

func (p *parser) resolveTag(tok token) (string, error) {
	switch tok.handle {
	case "":
		return tok.suffix, nil
	case "!!":
		if prefix, ok := p.tagHandles["!!"]; ok {
			return prefix + tok.suffix, nil
		}
		return coreTagPrefix + tok.suffix, nil
	case "!":
		return "!" + tok.suffix, nil
	}
	prefix, ok := p.tagHandles[tok.handle]
	if !ok {
		return "", p.s.errorAt("tag handle is not defined", tok.end)
	}
	return prefix + tok.suffix, nil
}

// parseNode composes one node; indentlessSequence allows a block sequence at the same indentation as its
// parent mapping key.
func (p *parser) parseNode(block, indentlessSequence bool) (*node, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}
	if tok.kind == tokenAlias {
		return p.parseAlias(nodeProperties{})
	}
	props, err := p.parseProperties()
	if err != nil {
		return nil, err
	}
	if tok, err = p.peek(); err != nil {
		return nil, err
	}
	if props.hasTag && !tagFitsCollection(props.tag, tok.kind) {
		at := tok.end
		at.column++
		return nil, p.s.errorAt("tag schema not supported", at)
	}
	var n *node
	switch {
	case tok.kind == tokenAlias:
		return p.parseAlias(props)
	case indentlessSequence && tok.kind == tokenBlockEntry:
		n, err = p.parseIndentlessSequence()
	case tok.kind == tokenScalar:
		if _, err := p.next(); err != nil {
			return nil, err
		}
		n = &node{kind: nodeScalar, value: tok.value}
	case tok.kind == tokenFlowSequenceStart:
		n, err = p.parseFlowSequence()
	case tok.kind == tokenFlowMappingStart:
		n, err = p.parseFlowMapping()
	case block && tok.kind == tokenBlockSequenceStart:
		n, err = p.parseBlockSequence()
	case block && tok.kind == tokenBlockMappingStart:
		n, err = p.parseBlockMapping()
	case isDirective(tok.kind):
		return nil, p.s.errorAt("directives are not allowed in a bare document", tok.end)
	case props.hasAnchor || props.hasTag:
		n = &node{kind: nodeScalar, empty: true}
	default:
		return nil, p.unexpected()
	}
	if err != nil {
		return nil, err
	}
	p.nodes++
	if props.hasTag {
		if err := p.castData(n, props.tag); err != nil {
			return nil, err
		}
	}
	if props.hasAnchor {
		if _, exists := p.anchors[props.anchor]; exists && !p.options.allowAnchorRedefinition {
			return nil, p.s.errorHere("duplicate anchor definition")
		}
		p.anchors[props.anchor] = n
	}
	return n, nil
}

func (p *parser) parseAlias(props nodeProperties) (*node, error) {
	tok, err := p.next()
	if err != nil {
		return nil, err
	}
	switch {
	case props.hasAnchor && props.hasTag:
		return nil, p.s.errorAt("an alias node cannot have tag properties", tok.end)
	case props.hasTag:
		return nil, p.s.errorAt("an alias node cannot have a tag", tok.end)
	case props.hasAnchor:
		return nil, p.s.errorAt("an alias node cannot have an anchor", tok.end)
	}
	target, ok := p.anchors[tok.value]
	if !ok {
		return nil, p.s.errorAt("anchor does not exist", tok.end)
	}
	p.aliasNodes += expandedSize(target)
	if p.excessiveAliasing() {
		return nil, p.s.errorAt("document contains excessive aliasing", tok.end)
	}
	return target, nil
}

// excessiveAliasing applies go-yaml's limit on the share of nodes that come from expanding aliases, which
// conversion walks once per alias. Small documents may be almost all aliases; large ones may not.
func (p *parser) excessiveAliasing() bool {
	total := p.nodes + p.aliasNodes
	if p.aliasNodes <= 100 || total <= 1000 {
		return false
	}
	allowed := 0.99
	switch {
	case total >= 4000000:
		allowed = 0.10
	case total > 400000:
		allowed = 0.99 - 0.89*float64(total-400000)/3600000
	}
	return float64(p.aliasNodes)/float64(total) > allowed
}

// expandedSize is the number of nodes n stands for once its aliases are expanded, capped so that it
// cannot overflow.
func expandedSize(n *node) int {
	if n.size > 0 {
		return n.size
	}
	size := 1
	for _, item := range n.items {
		size += expandedSize(item)
	}
	for _, p := range n.pairs {
		size += expandedSize(p.key) + expandedSize(p.value)
	}
	n.size = min(size, maxExpandedSize)
	return n.size
}

// castData applies an explicit tag the way jBallerina does: !!str, !!seq and !!map keep the node,
// !!int, !!float, !!bool and !!null resolve the scalar with the schema's rules, and any other tag, or
// any of those under the failsafe schema, is not supported.
func (p *parser) castData(n *node, tag string) error {
	if !strings.HasPrefix(tag, coreTagPrefix) {
		return p.s.errorHere("tag schema not supported")
	}
	switch name := strings.TrimPrefix(tag, coreTagPrefix); {
	case name == "str" && n.kind == nodeScalar && !n.empty:
		return nil
	case name == "seq" && n.kind == nodeSequence, name == "map" && n.kind == nodeMapping:
		return nil
	case n.kind != nodeScalar || n.empty || p.options.schema == schemaFailsafe:
		return p.s.errorHere("tag schema not supported")
	case name == "int" || name == "float" || name == "bool" || name == "null":
		value, ok := resolveTagged(name, n.value, p.options.schema == schemaCore)
		if !ok {
			kind := map[string]string{"int": "int", "float": "float", "bool": "boolean", "null": "null"}[name]
			return p.s.errorHere("cannot cast " + n.value + " to " + kind)
		}
		n.typed, n.hasTyped = value, true
		return nil
	}
	return p.s.errorHere("tag schema not supported")
}

// tagFitsCollection reports whether a tag may precede the collection that starts with kind. Any tag may
// precede a scalar; it is checked once the scalar is read.
func tagFitsCollection(tag string, kind tokenKind) bool {
	switch kind {
	case tokenFlowSequenceStart, tokenBlockSequenceStart, tokenBlockEntry:
		return tag == coreTagPrefix+"seq"
	case tokenFlowMappingStart, tokenBlockMappingStart:
		return tag == coreTagPrefix+"map"
	}
	return true
}

func (p *parser) parseBlockSequence() (*node, error) {
	if _, err := p.next(); err != nil {
		return nil, err
	}
	seq := &node{kind: nodeSequence}
	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		switch tok.kind {
		case tokenBlockEnd:
			if _, err := p.next(); err != nil {
				return nil, err
			}
			return seq, nil
		case tokenBlockEntry:
			item, err := p.parseBlockEntryValue(false)
			if err != nil {
				return nil, err
			}
			seq.items = append(seq.items, item)
		default:
			return nil, p.unexpected()
		}
	}
}

func (p *parser) parseIndentlessSequence() (*node, error) {
	seq := &node{kind: nodeSequence}
	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		if tok.kind != tokenBlockEntry {
			return seq, nil
		}
		item, err := p.parseBlockEntryValue(true)
		if err != nil {
			return nil, err
		}
		seq.items = append(seq.items, item)
	}
}

func (p *parser) parseBlockEntryValue(indentless bool) (*node, error) {
	if _, err := p.next(); err != nil {
		return nil, err
	}
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}
	switch tok.kind {
	case tokenBlockEntry, tokenBlockEnd:
		return &node{kind: nodeScalar, empty: true}, nil
	case tokenKey, tokenValue:
		if indentless {
			return &node{kind: nodeScalar, empty: true}, nil
		}
	}
	return p.parseNode(true, false)
}

func (p *parser) parseBlockMapping() (*node, error) {
	if _, err := p.next(); err != nil {
		return nil, err
	}
	m := &node{kind: nodeMapping}
	keys := make(map[string]int)
	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		var key *node
		explicitKey := false
		switch tok.kind {
		case tokenBlockEnd:
			if _, err := p.next(); err != nil {
				return nil, err
			}
			return m, nil
		case tokenKey:
			if _, err := p.next(); err != nil {
				return nil, err
			}
			explicitKey = tok.end.index > tok.start.index && p.s.input[tok.start.index] == '?'
			if key, err = p.parseBlockMappingPart(); err != nil {
				return nil, err
			}
		case tokenValue:
			key = &node{kind: nodeScalar, empty: true}
		case tokenBlockMappingStart:
			return nil, &yamlError{message: "Cannot have nested mapping under a key-pair that is already assigned", noPosition: true}
		default:
			return nil, p.s.errorHere("expected a key for the block mapping")
		}
		value := &node{kind: nodeScalar, empty: true}
		if tok, err = p.peek(); err != nil {
			return nil, err
		}
		if tok.kind == tokenValue {
			if _, err := p.next(); err != nil {
				return nil, err
			}
			if value, err = p.parseBlockMappingPart(); err != nil {
				return nil, err
			}
		}
		if key.kind != nodeScalar && !explicitKey {
			return nil, p.s.errorHere("there can only be one root event to a document")
		}
		if err := p.addPair(m, keys, key, value); err != nil {
			return nil, err
		}
	}
}

func (p *parser) parseBlockMappingPart() (*node, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}
	switch tok.kind {
	case tokenKey, tokenValue, tokenBlockEnd:
		return &node{kind: nodeScalar, empty: true}, nil
	}
	return p.parseNode(true, true)
}

// addPair adds a key-value pair to a mapping; keys holds the index of each key's pair. Keys must be
// scalars; a repeated key is an error unless allowMapEntryRedefinition is set, in which case the last
// value replaces the earlier one in the key's first position, as in jBallerina.
func (p *parser) addPair(m *node, keys map[string]int, key, value *node) error {
	if key.kind != nodeScalar {
		return p.s.errorHere("expected a key for the block mapping")
	}
	name := keyString(key)
	if i, ok := keys[name]; ok {
		if !p.options.allowMapEntryRedefinition {
			return p.s.errorHere("cannot have duplicate map entries for '${key.toString()}")
		}
		m.pairs[i].value = value
		return nil
	}
	keys[name] = len(m.pairs)
	m.pairs = append(m.pairs, pair{key: key, value: value})
	return nil
}

// keyString is the field name a key node gives: its text, or "null" for an empty key.
func keyString(key *node) string {
	if key.empty {
		return "null"
	}
	return key.value
}

func (p *parser) parseFlowSequence() (*node, error) {
	if _, err := p.next(); err != nil {
		return nil, err
	}
	seq := &node{kind: nodeSequence}
	first := true
	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		if tok.kind == tokenFlowSequenceEnd {
			if _, err := p.next(); err != nil {
				return nil, err
			}
			return seq, nil
		}
		if !first {
			if tok.kind != tokenFlowEntry {
				return nil, p.unexpected()
			}
			if _, err := p.next(); err != nil {
				return nil, err
			}
			if tok, err = p.peek(); err != nil {
				return nil, err
			}
			if tok.kind == tokenFlowSequenceEnd {
				continue
			}
		}
		first = false
		var item *node
		if tok.kind == tokenKey {
			item, err = p.parseFlowPair()
		} else {
			item, err = p.parseFlowNode()
		}
		if err != nil {
			return nil, err
		}
		seq.items = append(seq.items, item)
	}
}

// parseFlowPair reads a single-pair mapping inside a flow sequence, such as `[a: b]`.
func (p *parser) parseFlowPair() (*node, error) {
	if _, err := p.next(); err != nil {
		return nil, err
	}
	key, value, err := p.parseFlowKeyValue()
	if err != nil {
		return nil, err
	}
	m := &node{kind: nodeMapping}
	if err := p.addPair(m, map[string]int{}, key, value); err != nil {
		return nil, err
	}
	return m, nil
}

func (p *parser) parseFlowKeyValue() (*node, *node, error) {
	key := &node{kind: nodeScalar, empty: true}
	tok, err := p.peek()
	if err != nil {
		return nil, nil, err
	}
	if !isFlowEnd(tok.kind) && tok.kind != tokenValue {
		if key, err = p.parseFlowNode(); err != nil {
			return nil, nil, err
		}
	}
	value := &node{kind: nodeScalar, empty: true}
	if tok, err = p.peek(); err != nil {
		return nil, nil, err
	}
	if tok.kind == tokenValue {
		if _, err := p.next(); err != nil {
			return nil, nil, err
		}
		if tok, err = p.peek(); err != nil {
			return nil, nil, err
		}
		if !isFlowEnd(tok.kind) {
			if value, err = p.parseFlowNode(); err != nil {
				return nil, nil, err
			}
		}
	}
	return key, value, nil
}

func isFlowEnd(kind tokenKind) bool {
	return kind == tokenFlowEntry || kind == tokenFlowSequenceEnd || kind == tokenFlowMappingEnd
}

func (p *parser) parseFlowNode() (*node, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}
	if tok.kind == tokenStreamEnd || tok.kind == tokenDocumentStart || tok.kind == tokenDocumentEnd {
		return nil, p.unexpected()
	}
	return p.parseNode(false, false)
}

func (p *parser) parseFlowMapping() (*node, error) {
	if _, err := p.next(); err != nil {
		return nil, err
	}
	m := &node{kind: nodeMapping}
	keys := make(map[string]int)
	first := true
	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		if tok.kind == tokenFlowMappingEnd {
			if _, err := p.next(); err != nil {
				return nil, err
			}
			return m, nil
		}
		if !first {
			if tok.kind != tokenFlowEntry {
				return nil, p.unexpected()
			}
			if _, err := p.next(); err != nil {
				return nil, err
			}
			if tok, err = p.peek(); err != nil {
				return nil, err
			}
			if tok.kind == tokenFlowMappingEnd {
				continue
			}
		}
		first = false
		if tok.kind == tokenKey {
			if _, err := p.next(); err != nil {
				return nil, err
			}
		}
		key, value, err := p.parseFlowKeyValue()
		if err != nil {
			return nil, err
		}
		if err := p.addPair(m, keys, key, value); err != nil {
			return nil, err
		}
	}
}

// resolveTagged resolves the text of a scalar with an explicit int, float, bool or null tag under the
// JSON schema, or the core schema when core is set. The patterns must match the whole text.
func resolveTagged(name, text string, core bool) (any, bool) {
	switch name {
	case "null":
		if text == "null" || core && (text == "Null" || text == "NULL" || text == "~") {
			return nil, true
		}
	case "bool":
		switch {
		case text == "true" || core && (text == "True" || text == "TRUE"):
			return true, true
		case text == "false" || core && (text == "False" || text == "FALSE"):
			return false, true
		}
	case "int":
		return resolveTaggedInt(text, core)
	case "float":
		return resolveTaggedFloat(text, core)
	}
	return nil, false
}

func resolveTaggedInt(text string, core bool) (any, bool) {
	if core && len(text) > 1 {
		switch {
		case strings.HasPrefix(text, "0o"):
			n, err := strconv.ParseInt(text[2:], 8, 64)
			return n, err == nil && isDigits(text[2:])
		case strings.HasPrefix(text, "0x"):
			n, err := strconv.ParseInt(text[2:], 16, 64)
			return n, err == nil && isHexDigits(text[2:])
		}
	}
	var digits string
	if core {
		digits = strings.TrimLeft(text, "+-")
		if len(text)-len(digits) > 1 {
			return nil, false
		}
	} else {
		digits = strings.TrimPrefix(text, "-")
	}
	if !isDigits(digits) {
		return nil, false
	}
	n, err := strconv.ParseInt(text, 10, 64)
	return n, err == nil
}

func resolveTaggedFloat(text string, core bool) (any, bool) {
	if core && len(text) > 1 {
		switch {
		case text[0] == '.' && isAnyOf(text[1:], "nan", "NaN", "NAN"):
			return nan(), true
		case text[0] == '.' && isAnyOf(text[1:], "inf", "Inf", "INF"):
			return posInf(), true
		case (text[0] == '+' || text[0] == '-') && len(text) > 2 && text[1] == '.' && isAnyOf(text[2:], "inf", "Inf", "INF"):
			if text[0] == '-' {
				return negInf(), true
			}
			return posInf(), true
		}
	}
	if !isYAMLFloat(text, core) {
		return nil, false
	}
	f, err := strconv.ParseFloat(text, 64)
	return f, err == nil || isRangeError(err)
}

func isAnyOf(s string, options ...string) bool {
	for _, o := range options {
		if s == o {
			return true
		}
	}
	return false
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isHexDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !isHex(c) {
			return false
		}
	}
	return true
}

// isYAMLFloat matches the JSON schema float pattern -?(0|\d+)(\.\d*)?([eE][-+]?\d+)?, or under the
// core schema [-+]?(\.\d+|\d+(\.\d*)?)([eE][-+]?\d+)?.
func isYAMLFloat(text string, core bool) bool {
	rest := text
	if rest != "" && (rest[0] == '-' || core && rest[0] == '+') {
		rest = rest[1:]
	}
	mantissa, exponent, hasExponent := strings.Cut(strings.Replace(rest, "E", "e", 1), "e")
	whole, fraction, hasFraction := strings.Cut(mantissa, ".")
	switch {
	case whole == "" && (!core || !hasFraction || fraction == ""):
		return false
	case whole != "" && !isDigits(whole):
		return false
	case hasFraction && fraction != "" && !isDigits(fraction):
		return false
	}
	if hasExponent {
		exponent = strings.TrimLeft(exponent, "+-")
		if !isDigits(exponent) {
			return false
		}
	}
	return true
}
