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

// The scanner turns YAML text into tokens. It follows the structure of libyaml's scanner (simple keys,
// an indentation stack and flow levels) and reports errors with jBallerina's messages.

package native

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

type tokenKind int

const (
	tokenStreamStart tokenKind = iota
	tokenStreamEnd
	tokenVersionDirective
	tokenTagDirective
	tokenReservedDirective
	tokenDocumentStart
	tokenDocumentEnd
	tokenBlockSequenceStart
	tokenBlockMappingStart
	tokenBlockEnd
	tokenFlowSequenceStart
	tokenFlowSequenceEnd
	tokenFlowMappingStart
	tokenFlowMappingEnd
	tokenBlockEntry
	tokenFlowEntry
	tokenKey
	tokenValue
	tokenAlias
	tokenAnchor
	tokenTag
	tokenScalar
)

type scalarStyle int

const (
	stylePlain scalarStyle = iota
	styleSingleQuoted
	styleDoubleQuoted
	styleLiteral
	styleFolded
)

// mark is a position in the input. column counts the characters consumed on the line, so the column of
// a character is its 1-based position, as in jBallerina.
type mark struct {
	index  int
	line   int
	column int
}

type token struct {
	kind  tokenKind
	start mark
	value string
	// handle and suffix hold a tag's handle and suffix; for a tag directive they hold the handle and
	// the prefix.
	handle string
	suffix string
	style  scalarStyle
	// end is the position just after the token.
	end mark
	// nameEnd is where jBallerina reports a duplicate directive: after the directive name, or after the
	// separator following a tag handle.
	nameEnd mark
}

type simpleKey struct {
	possible    bool
	required    bool
	tokenNumber int
	mark        mark
}

// yamlError is an error raised while reading the YAML text. It is reported with its position unless
// noPosition is set.
type yamlError struct {
	message    string
	line       int
	column     int
	noPosition bool
}

func (e *yamlError) Error() string {
	if e.noPosition {
		return e.message
	}
	return "'" + e.message + "' at line: '" + formatJavaInt(int64(e.line)) + "' column: '" +
		formatJavaInt(int64(e.column)) + "'"
}

type scanner struct {
	input []rune
	pos   mark

	tokens       []token
	tokensParsed int
	streamStart  bool
	streamEnd    bool
	// streamEndQueued stops the scanner from producing tokens past the end of the input.
	streamEndQueued bool

	indent           int
	indents          []int
	simpleKeyAllowed bool
	simpleKeys       []simpleKey
	flowLevel        int
	// blockValueLine is the line of the last block mapping value indicator, which no second key or
	// sequence entry may share.
	blockValueLine int
	// docStartLine is the line of the last `---` marker.
	docStartLine int
	lastScalar   token
}

func newScanner(input string) *scanner {
	return &scanner{
		input:          []rune(input),
		pos:            mark{line: 1},
		indent:         -1,
		simpleKeys:     []simpleKey{{}},
		blockValueLine: -1,
		docStartLine:   -1,
	}
}

func (s *scanner) errorAt(message string, m mark) *yamlError {
	column := m.column
	if m.index >= len(s.input) && column > 0 && len(s.input) > 0 && s.input[len(s.input)-1] != '\n' {
		// jBallerina never counts the last character of the input in a column.
		column--
	}
	return &yamlError{message: message, line: m.line, column: column}
}

func (s *scanner) errorHere(message string) *yamlError {
	return s.errorAt(message, s.pos)
}

func (s *scanner) peekAt(offset int) rune {
	i := s.pos.index + offset
	if i < 0 || i >= len(s.input) {
		return 0
	}
	return s.input[i]
}

func (s *scanner) eof() bool {
	return s.pos.index >= len(s.input)
}

func (s *scanner) eofAt(offset int) bool {
	return s.pos.index+offset >= len(s.input)
}

func (s *scanner) skip() {
	if s.eof() {
		return
	}
	if s.input[s.pos.index] == '\n' {
		s.pos.line++
		s.pos.column = 0
	} else {
		s.pos.column++
	}
	s.pos.index++
}

func (s *scanner) isBreakAt(offset int) bool {
	c := s.peekAt(offset)
	return !s.eofAt(offset) && (c == '\n' || (c == '\r' && s.peekAt(offset+1) == '\n'))
}

func (s *scanner) isBlankAt(offset int) bool {
	c := s.peekAt(offset)
	return !s.eofAt(offset) && (c == ' ' || c == '\t')
}

func (s *scanner) isBlankOrBreakOrEOFAt(offset int) bool {
	return s.eofAt(offset) || s.isBlankAt(offset) || s.isBreakAt(offset) || s.peekAt(offset) == '\r'
}

// skipBreak consumes a line break, \n or \r\n.
func (s *scanner) skipBreak() {
	if s.peekAt(0) == '\r' {
		s.skip()
	}
	s.skip()
}

func isFlowIndicator(c rune) bool {
	return c == ',' || c == '[' || c == ']' || c == '{' || c == '}'
}

// isPrintable is the YAML printable character set that jBallerina accepts anywhere in the input.
func isPrintable(c rune) bool {
	return c == 0x09 || c == 0x0A || c == 0x0D || c == 0x85 || (c >= 0x20 && c <= 0x7E) ||
		(c >= 0xA0 && c <= 0xD7FF) || (c >= 0xE000 && c <= 0xFFFD) || (c >= 0x10000 && c <= 0x10FFFF)
}

// checkPrintable reports the first character outside the printable set, as jBallerina's reader does
// before scanning.
func (s *scanner) checkPrintable() error {
	line, column := 1, 0
	for _, c := range s.input {
		if c == '\n' {
			line++
			column = 0
			continue
		}
		if !isPrintable(c) && c != utf8.RuneError {
			return &yamlError{message: "non printable character found", line: line, column: column}
		}
		column++
	}
	return nil
}

func (s *scanner) emit(tok token) {
	tok.end = s.pos
	s.tokens = append(s.tokens, tok)
	if tok.kind == tokenScalar {
		s.lastScalar = tok
	}
}

func (s *scanner) next() (token, error) {
	if err := s.fetchMoreTokens(); err != nil {
		return token{}, err
	}
	tok := s.tokens[0]
	s.tokens = s.tokens[1:]
	s.tokensParsed++
	if tok.kind == tokenStreamEnd {
		s.streamEnd = true
	}
	return tok, nil
}

func (s *scanner) peek() (token, error) {
	if err := s.fetchMoreTokens(); err != nil {
		return token{}, err
	}
	return s.tokens[0], nil
}

func (s *scanner) fetchMoreTokens() error {
	for {
		if s.streamEndQueued && len(s.tokens) > 0 {
			return nil
		}
		needMore := len(s.tokens) == 0
		if !needMore {
			if err := s.staleSimpleKeys(); err != nil {
				return err
			}
			for _, key := range s.simpleKeys {
				if key.possible && key.tokenNumber == s.tokensParsed {
					needMore = true
					break
				}
			}
		}
		if !needMore {
			return nil
		}
		if err := s.fetchNextToken(); err != nil {
			return err
		}
	}
}

func (s *scanner) fetchNextToken() error {
	if !s.streamStart {
		s.streamStart = true
		s.simpleKeyAllowed = true
		s.emit(token{kind: tokenStreamStart, start: s.pos})
		return nil
	}
	if err := s.scanToNextToken(); err != nil {
		return err
	}
	if err := s.staleSimpleKeys(); err != nil {
		return err
	}
	if err := s.unrollIndent(s.pos.column); err != nil {
		return err
	}
	if s.eof() {
		return s.fetchStreamEnd()
	}
	c := s.peekAt(0)
	if s.pos.column == 0 {
		if c == '%' {
			return s.fetchDirective()
		}
		if s.isDocumentIndicator('-') {
			return s.fetchDocumentIndicator(tokenDocumentStart)
		}
		if s.isDocumentIndicator('.') {
			return s.fetchDocumentIndicator(tokenDocumentEnd)
		}
	}
	switch {
	case c == '[':
		return s.fetchFlowCollectionStart(tokenFlowSequenceStart)
	case c == '{':
		return s.fetchFlowCollectionStart(tokenFlowMappingStart)
	case c == ']':
		return s.fetchFlowCollectionEnd(tokenFlowSequenceEnd)
	case c == '}':
		return s.fetchFlowCollectionEnd(tokenFlowMappingEnd)
	case c == ',':
		return s.fetchFlowEntry()
	case c == '-' && s.isBlankOrBreakOrEOFAt(1):
		return s.fetchBlockEntry()
	case c == '?' && (s.flowLevel > 0 || s.isBlankOrBreakOrEOFAt(1)):
		return s.fetchKey()
	case c == ':' && (s.isBlankOrBreakOrEOFAt(1) || (s.flowLevel > 0 && (isFlowIndicator(s.peekAt(1)) || s.adjacentValueAllowed()))):
		return s.fetchValue()
	case c == '*':
		return s.fetchAnchor(tokenAlias)
	case c == '&':
		return s.fetchAnchor(tokenAnchor)
	case c == '!':
		return s.fetchTag()
	case (c == '|' || c == '>'):
		if s.flowLevel > 0 {
			s.skip()
			return s.errorHere("cannot have a block node inside a flow node")
		}
		return s.fetchBlockScalar(c == '|')
	case c == '\'':
		return s.fetchFlowScalar(styleSingleQuoted)
	case c == '"':
		return s.fetchFlowScalar(styleDoubleQuoted)
	}
	if s.canStartPlain() {
		return s.fetchPlainScalar()
	}
	return s.errorHere("invalid yaml document")
}

// adjacentValueAllowed reports whether a ':' directly after a JSON-like flow key is a value indicator.
func (s *scanner) adjacentValueAllowed() bool {
	if len(s.tokens) == 0 {
		return false
	}
	last := s.tokens[len(s.tokens)-1]
	return last.kind == tokenScalar && (last.style == styleSingleQuoted || last.style == styleDoubleQuoted) ||
		last.kind == tokenFlowSequenceEnd || last.kind == tokenFlowMappingEnd
}

func (s *scanner) canStartPlain() bool {
	c := s.peekAt(0)
	if s.isBlankOrBreakOrEOFAt(0) || c == '\uFEFF' {
		return false
	}
	switch c {
	case '-', '?', ':':
		next := s.peekAt(1)
		return !s.isBlankOrBreakOrEOFAt(1) && (s.flowLevel <= 0 || !isFlowIndicator(next))
	case ',', '[', ']', '{', '}', '#', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
		return false
	}
	return true
}

func (s *scanner) isDocumentIndicator(c rune) bool {
	return s.peekAt(0) == c && s.peekAt(1) == c && s.peekAt(2) == c && s.isBlankOrBreakOrEOFAt(3)
}

func (s *scanner) scanToNextToken() error {
	for {
		atLineStart := s.pos.column == 0
		for s.peekAt(0) == ' ' || (s.peekAt(0) == '\t' && (s.flowLevel > 0 || !s.simpleKeyAllowed)) {
			s.skip()
		}
		if s.peekAt(0) == '\t' && atLineStart && s.flowLevel == 0 {
			// A tab in block indentation is an error unless the rest of the line is blank or a comment.
			offset := 0
			for s.isBlankAt(offset) {
				offset++
			}
			if !s.isBreakAt(offset) && !s.eofAt(offset) && s.peekAt(offset) != '#' {
				s.skip()
				s.skip()
				return s.errorHere("cannot have tab as an indentation")
			}
		}
		for s.isBlankAt(0) {
			s.skip()
		}
		if s.peekAt(0) == '#' {
			for !s.eof() && !s.isBreakAt(0) {
				s.skip()
			}
		}
		if !s.isBreakAt(0) {
			return nil
		}
		s.skipBreak()
		if s.flowLevel == 0 {
			s.simpleKeyAllowed = true
		}
	}
}

func (s *scanner) staleSimpleKeys() error {
	for i := range s.simpleKeys {
		key := &s.simpleKeys[i]
		if key.possible && (key.mark.line < s.pos.line || key.mark.index+1024 < s.pos.index) {
			if key.required {
				return s.errorAt("mapping keys cannot span multiple lines", key.mark)
			}
			key.possible = false
		}
	}
	return nil
}

func (s *scanner) saveSimpleKey() error {
	required := s.flowLevel == 0 && s.indent == s.pos.column
	if !s.simpleKeyAllowed {
		return nil
	}
	key := simpleKey{possible: true, required: required, tokenNumber: s.tokensParsed + len(s.tokens), mark: s.pos}
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	s.simpleKeys[len(s.simpleKeys)-1] = key
	return nil
}

func (s *scanner) removeSimpleKey() error {
	i := len(s.simpleKeys) - 1
	if s.simpleKeys[i].possible && s.simpleKeys[i].required {
		return s.errorAt("mapping keys cannot span multiple lines", s.simpleKeys[i].mark)
	}
	s.simpleKeys[i].possible = false
	return nil
}

// rollIndent opens a block collection when column is deeper than the current indentation; tokenNumber
// is the position in the queue where the start token goes (-1 appends it).
func (s *scanner) rollIndent(column, tokenNumber int, kind tokenKind, m mark) {
	if s.flowLevel > 0 || s.indent >= column {
		return
	}
	s.indents = append(s.indents, s.indent)
	s.indent = column
	tok := token{kind: kind, start: m}
	if tokenNumber == -1 {
		s.emit(tok)
		return
	}
	i := tokenNumber - s.tokensParsed
	s.tokens = append(s.tokens[:i], append([]token{tok}, s.tokens[i:]...)...)
}

// unrollIndent closes the block collections deeper than column. A line that dedents to a column between
// two indentation levels is an error.
func (s *scanner) unrollIndent(column int) error {
	if s.flowLevel > 0 {
		return nil
	}
	popped := false
	for s.indent > column {
		s.emit(token{kind: tokenBlockEnd, start: s.pos})
		s.indent = s.indents[len(s.indents)-1]
		s.indents = s.indents[:len(s.indents)-1]
		popped = true
	}
	if popped && column > s.indent && !s.eof() {
		s.skip()
		return s.errorHere("invalid indentation")
	}
	return nil
}

func (s *scanner) fetchStreamEnd() error {
	if err := s.unrollIndent(-1); err != nil {
		return err
	}
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	for i := range s.simpleKeys {
		s.simpleKeys[i].possible = false
	}
	s.simpleKeyAllowed = false
	s.streamEndQueued = true
	s.emit(token{kind: tokenStreamEnd, start: s.pos})
	return nil
}

func (s *scanner) fetchDocumentIndicator(kind tokenKind) error {
	if err := s.unrollIndent(-1); err != nil {
		return err
	}
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = false
	start := s.pos
	if kind == tokenDocumentStart {
		s.docStartLine = start.line
	}
	s.skip()
	s.skip()
	s.skip()
	s.emit(token{kind: kind, start: start})
	return nil
}

func (s *scanner) fetchFlowCollectionStart(kind tokenKind) error {
	if err := s.saveSimpleKey(); err != nil {
		return err
	}
	s.simpleKeys = append(s.simpleKeys, simpleKey{})
	s.flowLevel++
	s.simpleKeyAllowed = true
	start := s.pos
	s.skip()
	s.emit(token{kind: kind, start: start})
	return nil
}

func (s *scanner) fetchFlowCollectionEnd(kind tokenKind) error {
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	if s.flowLevel > 0 {
		s.flowLevel--
		s.simpleKeys = s.simpleKeys[:len(s.simpleKeys)-1]
	}
	s.simpleKeyAllowed = false
	start := s.pos
	s.skip()
	s.emit(token{kind: kind, start: start})
	return nil
}

func (s *scanner) fetchFlowEntry() error {
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = true
	start := s.pos
	s.skip()
	s.emit(token{kind: tokenFlowEntry, start: start})
	return nil
}

func (s *scanner) fetchBlockEntry() error {
	if s.flowLevel > 0 {
		s.skip()
		return s.errorHere("cannot have block sequence under flow collection")
	}
	if !s.simpleKeyAllowed {
		s.skip()
		if s.pos.line == s.blockValueLine {
			return s.errorHere("cannot have nested sequence for a defined value")
		}
		return s.errorHere("invalid yaml document")
	}
	s.rollIndent(s.pos.column, -1, tokenBlockSequenceStart, s.pos)
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = true
	start := s.pos
	s.skip()
	s.emit(token{kind: tokenBlockEntry, start: start})
	return nil
}

func (s *scanner) fetchKey() error {
	if s.flowLevel == 0 {
		if !s.simpleKeyAllowed {
			s.skip()
			return s.errorHere("invalid yaml document")
		}
		s.rollIndent(s.pos.column, -1, tokenBlockMappingStart, s.pos)
	}
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = s.flowLevel == 0
	start := s.pos
	s.skip()
	s.emit(token{kind: tokenKey, start: start})
	return nil
}

func (s *scanner) fetchValue() error {
	key := &s.simpleKeys[len(s.simpleKeys)-1]
	if key.possible {
		if s.flowLevel == 0 && key.mark.line == s.blockValueLine {
			s.skip()
			return s.errorHere("two block mapping keys cannot be defined in the same line")
		}
		i := key.tokenNumber - s.tokensParsed
		s.tokens = append(s.tokens[:i], append([]token{{kind: tokenKey, start: key.mark}}, s.tokens[i:]...)...)
		s.rollIndent(key.mark.column, key.tokenNumber, tokenBlockMappingStart, key.mark)
		key.possible = false
		s.simpleKeyAllowed = false
	} else {
		if s.flowLevel == 0 {
			if !s.simpleKeyAllowed {
				return s.misplacedValueError()
			}
			s.rollIndent(s.pos.column, -1, tokenBlockMappingStart, s.pos)
		}
		s.simpleKeyAllowed = s.flowLevel == 0
	}
	if s.flowLevel == 0 {
		s.blockValueLine = s.pos.line
	}
	start := s.pos
	s.skip()
	s.emit(token{kind: tokenValue, start: start})
	return nil
}

// misplacedValueError reports a mapping value indicator that cannot start a block mapping here, with the
// message jBallerina gives for the construct before it.
func (s *scanner) misplacedValueError() error {
	last := s.lastScalar
	if last.kind == tokenScalar && last.start.line == s.docStartLine {
		return s.errorHere("block collection token cannot start in the same line as the directive marker")
	}
	s.skip()
	switch {
	case s.pos.line == s.blockValueLine:
		return s.errorHere("two block mapping keys cannot be defined in the same line")
	case last.kind == tokenScalar && last.start.line < s.pos.line && last.start.line == s.blockValueLine:
		return &yamlError{message: "Cannot have nested mapping under a key-pair that is already assigned", noPosition: true}
	case last.kind == tokenScalar && last.start.line < s.pos.line:
		s.skip()
		return s.errorHere("there can only be one root event to a document")
	}
	return s.errorHere("mapping keys cannot span multiple lines")
}

func isAnchorChar(c rune) bool {
	return c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\uFEFF' && !isFlowIndicator(c) && isPrintable(c)
}

func (s *scanner) fetchAnchor(kind tokenKind) error {
	if err := s.saveSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = false
	start := s.pos
	s.skip()
	var name strings.Builder
	for !s.eof() && isAnchorChar(s.peekAt(0)) {
		name.WriteRune(s.peekAt(0))
		s.skip()
	}
	if name.Len() == 0 {
		return s.errorHere("invalid yaml document")
	}
	s.emit(token{kind: kind, start: start, value: name.String()})
	return nil
}

func isWordChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-'
}

func isURIChar(c rune) bool {
	return isWordChar(c) || strings.ContainsRune("#;/?:@&=+$,_.!~*'()[]%", c)
}

func (s *scanner) fetchTag() error {
	if err := s.saveSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = false
	start := s.pos
	var handle, suffix string
	if s.peekAt(1) == '<' {
		s.skip()
		s.skip()
		uri, err := s.scanURI(false)
		if err != nil {
			return err
		}
		if s.peekAt(0) != '>' {
			return s.errorHere("invalid yaml document")
		}
		s.skip()
		suffix = uri
		if uri == "!" {
			return s.errorHere("'verbatim tag' is not resolved. Hence, '!' is invalid")
		}
	} else {
		handle = s.scanTagHandle()
		if strings.HasSuffix(handle, "!") && len(handle) > 1 || handle == "!" && s.peekAt(0) == '!' {
			uri, err := s.scanURI(true)
			if err != nil {
				return err
			}
			suffix = uri
		} else {
			uri, err := s.scanURI(true)
			if err != nil {
				return err
			}
			suffix = strings.TrimPrefix(handle, "!") + uri
			handle = "!"
			if suffix == "" {
				handle, suffix = "", "!"
			}
		}
	}
	if !s.isBlankOrBreakOrEOFAt(0) && (s.flowLevel <= 0 || s.peekAt(0) != ',') {
		return s.errorHere("invalid yaml document")
	}
	s.emit(token{kind: tokenTag, start: start, handle: handle, suffix: suffix})
	return nil
}

// scanTagHandle reads `!`, `!!` or `!word!`; a primary tag `!suffix` returns `!suffix` without the
// closing `!`.
func (s *scanner) scanTagHandle() string {
	var b strings.Builder
	b.WriteRune('!')
	s.skip()
	for isWordChar(s.peekAt(0)) {
		b.WriteRune(s.peekAt(0))
		s.skip()
	}
	if s.peekAt(0) == '!' {
		b.WriteRune('!')
		s.skip()
	}
	return b.String()
}

func (s *scanner) scanURI(tagSuffix bool) (string, error) {
	var b []byte
	for !s.eof() {
		c := s.peekAt(0)
		if !isURIChar(c) || (tagSuffix && (c == '!' || isFlowIndicator(c))) {
			break
		}
		if c == '%' {
			if !isHex(s.peekAt(1)) || !isHex(s.peekAt(2)) {
				return "", s.errorHere("invalid yaml document")
			}
			n, _ := strconv.ParseUint(string([]rune{s.peekAt(1), s.peekAt(2)}), 16, 8)
			b = append(b, byte(n))
			s.skip()
			s.skip()
			s.skip()
			continue
		}
		b = utf8.AppendRune(b, c)
		s.skip()
	}
	return string(b), nil
}

func isHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func (s *scanner) fetchDirective() error {
	if err := s.unrollIndent(-1); err != nil {
		return err
	}
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = false
	start := s.pos
	s.skip()
	var name strings.Builder
	for !s.isBlankOrBreakOrEOFAt(0) {
		name.WriteRune(s.peekAt(0))
		s.skip()
	}
	nameEnd := s.pos
	for s.isBlankAt(0) {
		s.skip()
	}
	var params []string
	var paramEnds []mark
	for !s.eof() && !s.isBreakAt(0) && s.peekAt(0) != '#' && s.peekAt(0) != '\r' {
		var param strings.Builder
		for !s.isBlankOrBreakOrEOFAt(0) {
			param.WriteRune(s.peekAt(0))
			s.skip()
		}
		params = append(params, param.String())
		paramEnds = append(paramEnds, s.pos)
		for s.isBlankAt(0) {
			s.skip()
		}
	}
	tok := token{start: start, value: name.String(), nameEnd: nameEnd}
	switch name.String() {
	case "YAML":
		tok.kind = tokenVersionDirective
		if len(params) > 0 {
			tok.suffix = params[0]
		}
	case "TAG":
		tok.kind = tokenTagDirective
		if len(params) < 2 {
			return s.errorHere("invalid yaml document")
		}
		tok.handle, tok.suffix = params[0], params[1]
		tok.nameEnd = paramEnds[0]
		tok.nameEnd.column++
	default:
		tok.kind = tokenReservedDirective
	}
	s.emit(tok)
	return nil
}

func (s *scanner) fetchBlockScalar(literal bool) error {
	if err := s.removeSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = true
	tok, err := s.scanBlockScalar(literal)
	if err != nil {
		return err
	}
	s.emit(tok)
	return nil
}

func (s *scanner) fetchFlowScalar(style scalarStyle) error {
	if err := s.saveSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = false
	tok, err := s.scanFlowScalar(style)
	if err != nil {
		return err
	}
	s.emit(tok)
	return nil
}

func (s *scanner) fetchPlainScalar() error {
	if err := s.saveSimpleKey(); err != nil {
		return err
	}
	s.simpleKeyAllowed = false
	tok := s.scanPlainScalar()
	s.emit(tok)
	return nil
}

// scanBlockScalar reads a literal or folded block scalar with YAML 1.2 indentation, folding and chomping.
// At the top level, content may start in column 0, as jBallerina allows.
func (s *scanner) scanBlockScalar(literal bool) (token, error) {
	start := s.pos
	s.skip()
	chomping, increment := 0, 0
	for i := 0; i < 2; i++ {
		c := s.peekAt(0)
		switch {
		case (c == '+' || c == '-') && chomping == 0:
			chomping = 1
			if c == '-' {
				chomping = -1
			}
			s.skip()
		case c >= '1' && c <= '9' && increment == 0:
			increment = int(c - '0')
			s.skip()
		}
	}
	for s.isBlankAt(0) {
		s.skip()
	}
	if s.peekAt(0) == '#' && s.isBlankAt(-1) {
		for !s.eof() && !s.isBreakAt(0) {
			s.skip()
		}
	}
	if !s.eof() && !s.isBreakAt(0) {
		return token{}, s.errorHere("invalid block header")
	}
	var leadingBreak string
	if s.isBreakAt(0) {
		s.skipBreak()
	}
	indent := 0
	if increment > 0 {
		indent = increment
		if s.indent >= 0 {
			indent = s.indent + increment
		}
	}
	var text strings.Builder
	trailingBreaks, err := s.blockScalarBreaks(&indent)
	if err != nil {
		return token{}, err
	}
	leadingBlank := false
	for s.pos.column == indent && !s.eof() &&
		(s.pos.column != 0 || (!s.isDocumentIndicator('-') && !s.isDocumentIndicator('.'))) {
		trailingBlank := s.isBlankAt(0)
		if !literal && strings.HasPrefix(leadingBreak, "\n") && !leadingBlank && !trailingBlank {
			if trailingBreaks == "" {
				text.WriteByte(' ')
			}
		} else {
			text.WriteString(leadingBreak)
		}
		text.WriteString(trailingBreaks)
		leadingBlank = s.isBlankAt(0)
		for !s.eof() && !s.isBreakAt(0) {
			text.WriteRune(s.peekAt(0))
			s.skip()
		}
		if s.eof() {
			// jBallerina ends the last line with a line break even when the input does not.
			leadingBreak = "\n"
			trailingBreaks = ""
			break
		}
		s.skipBreak()
		leadingBreak = "\n"
		if trailingBreaks, err = s.blockScalarBreaks(&indent); err != nil {
			return token{}, err
		}
	}
	if chomping != -1 {
		text.WriteString(leadingBreak)
	}
	if chomping == 1 {
		text.WriteString(trailingBreaks)
	}
	style := styleFolded
	if literal {
		style = styleLiteral
	}
	return token{kind: tokenScalar, start: start, value: text.String(), style: style}, nil
}

// blockScalarBreaks consumes the indentation and empty lines before the next content line of a block
// scalar, detecting the content indentation when it is not known yet.
func (s *scanner) blockScalarBreaks(indent *int) (string, error) {
	var breaks strings.Builder
	maxIndent := 0
	for {
		for (*indent == 0 || s.pos.column < *indent) && s.peekAt(0) == ' ' {
			s.skip()
		}
		if s.pos.column > maxIndent {
			maxIndent = s.pos.column
		}
		if (*indent == 0 || s.pos.column < *indent) && s.peekAt(0) == '\t' {
			s.skip()
			return "", s.errorHere("cannot have tab as an indentation")
		}
		if !s.isBreakAt(0) {
			break
		}
		s.skipBreak()
		breaks.WriteByte('\n')
	}
	if *indent == 0 {
		*indent = maxIndent
		if *indent < s.indent+1 {
			*indent = s.indent + 1
		}
		if *indent < 1 && s.indent >= 0 {
			*indent = 1
		}
	}
	return breaks.String(), nil
}

func (s *scanner) scanFlowScalar(style scalarStyle) (token, error) {
	start := s.pos
	double := style == styleDoubleQuoted
	quote := '\''
	kind := "single-quoted"
	if double {
		quote = '"'
		kind = "double-quoted"
	}
	s.skip()
	var text strings.Builder
	for {
		if s.pos.column == 0 && (s.isDocumentIndicator('-') || s.isDocumentIndicator('.')) {
			name := "DIRECTIVE_MARKER"
			if s.peekAt(0) == '.' {
				name = "DOCUMENT_MARKER"
			}
			return token{}, s.errorHere("invalid token '" + name + "' inside the " + kind + " scalar")
		}
		if s.eof() {
			if !double {
				return token{}, s.errorHere("END of stream reached")
			}
			return token{}, s.errorHere("invalid token 'EOL' inside the " + kind + " scalar")
		}
		leadingBlanks := false
		for !s.isBlankOrBreakOrEOFAt(0) {
			c := s.peekAt(0)
			switch {
			case !double && c == '\'' && s.peekAt(1) == '\'':
				text.WriteRune('\'')
				s.skip()
				s.skip()
				continue
			case c == quote:
			case double && c == '\\' && s.isBreakAt(1):
				s.skip()
				s.skipBreak()
				leadingBlanks = true
			case double && c == '\\':
				if err := s.scanEscape(&text); err != nil {
					return token{}, err
				}
				continue
			default:
				text.WriteRune(c)
				s.skip()
				continue
			}
			break
		}
		if s.peekAt(0) == quote && !s.eof() {
			break
		}
		var whitespaces, leadingBreak, trailingBreaks strings.Builder
		for s.isBlankAt(0) || s.isBreakAt(0) || (s.peekAt(0) == '\r' && !s.eof()) {
			switch {
			case s.isBlankAt(0):
				if !leadingBlanks {
					whitespaces.WriteRune(s.peekAt(0))
				}
				s.skip()
			case !s.isBreakAt(0):
				s.skip()
			case !leadingBlanks:
				whitespaces.Reset()
				s.skipBreak()
				leadingBreak.WriteByte('\n')
				leadingBlanks = true
			default:
				s.skipBreak()
				trailingBreaks.WriteByte('\n')
			}
		}
		if leadingBreak.Len() > 0 && s.flowLevel == 0 && !s.eof() && s.pos.column <= s.indent &&
			(s.pos.column != 0 || (!s.isDocumentIndicator('-') && !s.isDocumentIndicator('.'))) {
			s.skip()
			return token{}, s.errorHere("insufficient indentation for a scalar")
		}
		switch {
		case !leadingBlanks:
			text.WriteString(whitespaces.String())
		case leadingBreak.Len() > 0 && trailingBreaks.Len() == 0:
			text.WriteByte(' ')
		default:
			text.WriteString(trailingBreaks.String())
		}
	}
	s.skip()
	return token{kind: tokenScalar, start: start, value: text.String(), style: style}, nil
}

var simpleEscapes = map[rune]string{
	'0': "\x00", 'a': "\a", 'b': "\b", 't': "\t", '\t': "\t", 'n': "\n", 'v': "\v", 'f': "\f",
	'r': "\r", 'e': "\x1b", ' ': " ", '"': "\"", '/': "/", '\\': "\\", 'N': "\u0085", '_': " ",
	'L': " ", 'P': " ",
}

func (s *scanner) scanEscape(text *strings.Builder) error {
	s.skip()
	if s.eof() {
		text.WriteRune('\\')
		return nil
	}
	c := s.peekAt(0)
	if escaped, ok := simpleEscapes[c]; ok {
		text.WriteString(escaped)
		s.skip()
		return nil
	}
	length := map[rune]int{'x': 2, 'u': 4, 'U': 8}[c]
	if length == 0 {
		return s.errorHere("invalid escape character")
	}
	s.skip()
	var hex strings.Builder
	for i := 0; i < length; i++ {
		if !isHex(s.peekAt(0)) || s.eof() {
			return s.errorHere("expected a unicode character after escaped char")
		}
		hex.WriteRune(s.peekAt(0))
		s.skip()
	}
	code, _ := strconv.ParseUint(hex.String(), 16, 32)
	if code > utf8.MaxRune || (code >= 0xD800 && code <= 0xDFFF) {
		return s.errorHere("expected a unicode character after escaped char")
	}
	text.WriteRune(rune(code))
	return nil
}

// scanPlainScalar reads a plain scalar, folding line breaks into spaces and keeping empty lines as line
// breaks.
func (s *scanner) scanPlainScalar() token {
	start := s.pos
	var text, leadingBreak, trailingBreaks, whitespaces strings.Builder
	leadingBlanks := false
	indent := s.indent + 1
	for s.pos.column != 0 || (!s.isDocumentIndicator('-') && !s.isDocumentIndicator('.')) {
		if s.peekAt(0) == '#' {
			break
		}
		for !s.isBlankOrBreakOrEOFAt(0) {
			c := s.peekAt(0)
			if c == ':' && (s.isBlankOrBreakOrEOFAt(1) || (s.flowLevel > 0 && isFlowIndicator(s.peekAt(1)))) {
				break
			}
			if s.flowLevel > 0 && isFlowIndicator(c) {
				break
			}
			if leadingBlanks || whitespaces.Len() > 0 {
				if leadingBlanks {
					if leadingBreak.Len() > 0 && trailingBreaks.Len() == 0 {
						text.WriteByte(' ')
					} else {
						text.WriteString(trailingBreaks.String())
					}
					leadingBreak.Reset()
					trailingBreaks.Reset()
					leadingBlanks = false
				} else {
					text.WriteString(whitespaces.String())
					whitespaces.Reset()
				}
			}
			text.WriteRune(c)
			s.skip()
		}
		if !s.isBlankAt(0) && !s.isBreakAt(0) {
			break
		}
		for s.isBlankAt(0) || s.isBreakAt(0) || (s.peekAt(0) == '\r' && !s.eof()) {
			if s.isBlankAt(0) {
				if leadingBlanks && s.pos.column < indent && s.peekAt(0) == '\t' {
					s.skip()
					continue
				}
				if !leadingBlanks {
					whitespaces.WriteRune(s.peekAt(0))
				}
				s.skip()
				continue
			}
			if s.peekAt(0) == '\r' && !s.isBreakAt(0) {
				break
			}
			if !leadingBlanks {
				whitespaces.Reset()
				s.skipBreak()
				leadingBreak.WriteByte('\n')
				leadingBlanks = true
			} else {
				s.skipBreak()
				trailingBreaks.WriteByte('\n')
			}
		}
		if s.flowLevel == 0 && s.pos.column < indent {
			break
		}
	}
	if leadingBlanks {
		s.simpleKeyAllowed = true
	}
	return token{kind: tokenScalar, start: start, value: text.String(), style: stylePlain}
}
