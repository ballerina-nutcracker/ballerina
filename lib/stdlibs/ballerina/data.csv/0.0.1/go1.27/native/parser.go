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

// Port of the state machine in jBallerina's CsvParser. Characters are fed one at a time, so the same
// machine serves whole-input parsing and parseToStream.

package native

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/ballerina-nutcracker/ballerina/values"
)

type parseState int

const (
	stateHeaderStart parseState = iota
	stateHeaderEnd
	stateHeaderQuote
	stateHeaderEscape
	stateHeaderUnicode
	stateRowStart
	stateRowEnd
	stateStringQuote
	stateStringEscape
	stateStringUnicode
)

// endOfInput stands for the end-of-input pseudo character jBallerina feeds after the last character.
// A real U+FFFF in the data is ordinary data here, where jBallerina mistakes it for the end of input.
const endOfInput rune = -1

type parser struct {
	cfg   *config
	t     *typer
	conv  *converter
	shape *rowShape
	// rootSize is the length of a closed result array, or -1.
	rootSize  int
	streaming bool

	state     parseState
	prevState parseState
	buf       []rune
	hex       []rune

	isValueStart                   bool
	isQuoteClosed                  bool
	insideComment                  bool
	skipTheRow                     bool
	isHeaderConfigExceedLineNumber bool

	// line and column locate the current character for error messages; column counts UTF-16 code
	// units, as jBallerina's does.
	line   int
	column int
	// lineNumber counts the rows and lines processed so far, including the header.
	lineNumber  int
	columnIndex int
	// rowIndex is the 1-based index of the next data row, which skipLines refers to.
	rowIndex int

	headers        []string
	fieldHierarchy map[string]*recordField
	fieldNames     map[string]*recordField

	row *rowBuilder
	// rowStarted is set once the current row passed the skipLines check and has its builder.
	rowStarted             bool
	isCurrentCsvNodeEmpty  bool
	currentCsvNodeLength   int
	isRowMaxSizeReached    bool
	isColumnMaxSizeReached bool
	addHeadersForOutput    bool
	arraySize              int
	rows                   []values.BalValue

	// skipLines and skipErr are cfg.skipDataRows, which every row consults.
	skipLines skipSet
	skipErr   error

	failSafe *failSafeLogger
	// recovering is set while the rest of a line that raised a tolerated error is discarded.
	recovering bool
	// lineRecovered is set when the last character fed ended the discarded line of a tolerated error.
	lineRecovered bool
	pendingErr    error
	pendingColumn int
	pendingLine   int
	currentLine   []rune
	previousLine  string
}

func newParser(cfg *config, t *typer, shape *rowShape, rootSize int, failSafe *failSafeLogger) *parser {
	skipLines, skipErr := cfg.skipDataRows()
	p := &parser{
		cfg:                   cfg,
		t:                     t,
		conv:                  cfg.converter(t),
		shape:                 shape,
		rootSize:              rootSize,
		line:                  1,
		rowIndex:              1,
		fieldHierarchy:        make(map[string]*recordField),
		fieldNames:            make(map[string]*recordField),
		isCurrentCsvNodeEmpty: true,
		failSafe:              failSafe,
		skipLines:             skipLines,
		skipErr:               skipErr,
	}
	if shape.kind == kindRecord {
		p.fieldHierarchy = shape.remainingFields()
	}
	if cfg.header != nil {
		p.state = stateHeaderStart
		return p
	}
	for _, header := range cfg.customHeadersIfHeadersAbsent {
		p.headers = append(p.headers, header)
		if f, ok := p.fieldHierarchy[header]; ok {
			p.fieldNames[header] = f
			delete(p.fieldHierarchy, header)
		}
	}
	for name, f := range p.fieldHierarchy {
		p.fieldNames[name] = f
	}
	p.state = stateRowStart
	return p
}

// feed runs one character through the machine. A tolerated failSafe error discards the rest of the
// offending line and resumes with the next row.
func (p *parser) feed(ch rune) error {
	p.trackLine(ch)
	p.lineRecovered = false
	if p.recovering {
		p.processLocation(ch)
		if ch == '\n' {
			return p.endRecovery()
		}
		return nil
	}
	err := p.transition(ch)
	if err == nil {
		return nil
	}
	if p.failSafe == nil || isFatal(err) {
		return p.locate(err)
	}
	p.pendingErr = err
	p.pendingColumn = p.columnIndex
	p.pendingLine = p.line
	p.resetAfterError()
	if ch == '\n' {
		p.pendingLine--
		p.lineRecovered = true
		return p.logPending(p.previousLine)
	}
	p.recovering = true
	return nil
}

// finish feeds the end of input. When a tolerated error discarded the input up to its end, the row
// has already ended, as in jBallerina.
func (p *parser) finish() error {
	if p.recovering {
		p.recovering = false
		p.lineRecovered = true
		if err := p.logPending(string(p.currentLine)); err != nil {
			return err
		}
	}
	if p.lineRecovered {
		p.state = stateRowEnd
	}
	err := p.transition(endOfInput)
	if err == nil && p.state != stateRowEnd && p.state != stateHeaderEnd && !p.isHeaderConfigExceedLineNumber {
		err = newTokenError("Invalid token found")
	}
	if err == nil {
		return nil
	}
	if p.failSafe == nil || isFatal(err) {
		return p.locate(err)
	}
	p.pendingErr = err
	p.pendingColumn = p.columnIndex
	p.pendingLine = p.line
	return p.logPending(string(p.currentLine))
}

func (p *parser) trackLine(ch rune) {
	if ch == '\n' {
		p.previousLine = string(p.currentLine)
		p.currentLine = p.currentLine[:0]
		return
	}
	p.currentLine = append(p.currentLine, ch)
}

func (p *parser) endRecovery() error {
	p.recovering = false
	p.lineRecovered = true
	return p.logPending(p.previousLine)
}

func (p *parser) logPending(offendingRow string) error {
	err := p.failSafe.log(p.pendingErr, p.pendingLine, p.pendingColumn+1, strings.TrimSuffix(offendingRow, "\r"))
	p.pendingErr = nil
	return err
}

// resetAfterError discards the partial row, as jBallerina does before resuming at the next line.
func (p *parser) resetAfterError() {
	p.updateLineAndColumnIndexes()
	p.state = stateRowStart
	p.isQuoteClosed = false
	p.isValueStart = false
	p.insideComment = false
	p.skipTheRow = false
	p.currentCsvNodeLength = 0
	p.hex = p.hex[:0]
}

// locate turns a token error into the INVALID_TOKEN error carrying the current location.
func (p *parser) locate(err error) error {
	if csvErr, ok := err.(*csvError); ok && csvErr.tokenError {
		return invalidTokenError(csvErr.message, p.line, p.column)
	}
	return err
}

func (p *parser) processLocation(ch rune) {
	switch {
	case ch == '\n':
		p.line++
		p.column = 0
	case ch > 0xFFFF:
		p.column += 2
	default:
		p.column++
	}
}

func (p *parser) transition(ch rune) error {
	if p.state == stateHeaderEnd {
		p.state = stateRowStart
		if ch == endOfInput {
			return nil
		}
	}
	switch p.state {
	case stateHeaderStart:
		return p.headerStart(ch)
	case stateHeaderQuote:
		return p.headerQuote(ch)
	case stateHeaderEscape, stateStringEscape:
		return p.escape(ch)
	case stateHeaderUnicode, stateStringUnicode:
		return p.unicode(ch)
	case stateRowStart:
		return p.rowStart(ch)
	case stateStringQuote:
		return p.stringQuote(ch)
	}
	return nil
}

func (p *parser) isNewLineOrEof(ch rune) bool {
	return ch == endOfInput || ch == '\n'
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func (p *parser) append(ch rune) {
	p.buf = append(p.buf, ch)
}

func (p *parser) value() string {
	v := string(p.buf)
	p.buf = p.buf[:0]
	return v
}

func (p *parser) peek() string {
	return string(p.buf)
}

func (p *parser) headerStart(ch rune) error {
	p.processLocation(ch)
	if ch == endOfInput {
		p.state = stateHeaderEnd
		return p.handleEndOfTheHeader(true)
	}
	if ch == '\r' {
		return nil
	}
	if int64(p.lineNumber) < *p.cfg.header {
		p.isHeaderConfigExceedLineNumber = true
		if p.isNewLineOrEof(ch) {
			p.lineNumber++
		}
		return nil
	}
	p.isHeaderConfigExceedLineNumber = false
	switch {
	case ch == p.cfg.comment:
		p.insideComment = true
	case !p.insideComment && ch == p.cfg.delimiter:
		if err := p.addHeader(true); err != nil {
			return err
		}
		p.columnIndex++
	case !p.insideComment && ch == p.cfg.textEnclosure:
		p.prevState = stateHeaderStart
		p.state = stateHeaderQuote
	case !p.insideComment && ch == p.cfg.escapeChar:
		p.prevState = stateHeaderStart
		p.state = stateHeaderEscape
	case p.insideComment && p.isNewLineOrEof(ch):
		p.insideComment = false
		p.state = stateHeaderEnd
		return p.handleEndOfTheHeader(true)
	case !p.insideComment && p.isNewLineOrEof(ch):
		p.state = stateHeaderEnd
		return p.handleEndOfTheHeader(true)
	case isWhitespace(ch):
		if p.isValueStart {
			p.append(ch)
		}
	default:
		if !p.insideComment {
			p.append(ch)
			p.isValueStart = true
		}
	}
	return nil
}

func (p *parser) headerQuote(ch rune) error {
	p.processLocation(ch)
	if ch == endOfInput {
		p.state = stateRowEnd
		return p.handleEndOfTheRow(true)
	}
	if ch == '\r' {
		return nil
	}
	switch {
	case ch == p.cfg.textEnclosure:
		p.isQuoteClosed = true
	case ch == p.cfg.delimiter && p.isQuoteClosed:
		if err := p.addHeader(false); err != nil {
			return err
		}
		p.columnIndex++
		p.isQuoteClosed = false
		p.state = stateHeaderStart
	case p.isNewLineOrEof(ch) && p.isQuoteClosed:
		p.isQuoteClosed = false
		p.state = stateHeaderEnd
		return p.handleEndOfTheHeader(false)
	case !p.isQuoteClosed && ch == p.cfg.escapeChar:
		p.prevState = stateHeaderQuote
		p.state = stateHeaderEscape
	default:
		p.appendQuoted(ch)
	}
	return nil
}

// appendQuoted appends a character of a quoted value. Text after a closing quote reopens the quote and
// keeps the closing quote character.
func (p *parser) appendQuoted(ch rune) {
	if p.isQuoteClosed {
		p.append(p.cfg.textEnclosure)
		p.isQuoteClosed = false
	}
	p.append(ch)
	p.isValueStart = true
}

func (p *parser) handleEndOfTheHeader(trim bool) error {
	p.isValueStart = false
	if err := p.addHeader(trim); err != nil {
		return err
	}
	if err := p.finalizeHeaders(); err != nil {
		return err
	}
	p.columnIndex = 0
	p.lineNumber++
	return nil
}

func (p *parser) addHeader(trim bool) error {
	p.isValueStart = false
	value := p.value()
	if trim {
		value = javaTrim(value)
	}
	if value == "" {
		return emptyHeaderError()
	}
	if p.shape.kind == kindRecord {
		if f, ok := p.fieldHierarchy[value]; ok {
			p.fieldNames[value] = f
			delete(p.fieldHierarchy, value)
		}
	}
	if slices.Contains(p.headers, value) {
		return duplicateHeaderError(value)
	}
	p.headers = append(p.headers, value)
	return nil
}

func (p *parser) finalizeHeaders() error {
	if len(p.headers) == 0 {
		return emptyHeaderError()
	}
	switch p.shape.kind {
	case kindRecord:
		if !p.shape.hasRest {
			for _, name := range p.shape.fieldOrder {
				f, remaining := p.fieldHierarchy[name]
				if !remaining || (p.cfg.absentAsNilableType && f.nilable) {
					continue
				}
				if !f.optional {
					return missingFieldError(f.name)
				}
			}
		}
	case kindArray:
		if p.shape.size != -1 && p.shape.size > len(p.headers) {
			return arraySizeError(len(p.headers))
		}
	case kindTuple:
		if p.shape.hasRest && p.shape.size > len(p.headers) {
			return tupleSizeError(len(p.headers))
		}
	}
	return nil
}

func (p *parser) rowStart(ch rune) error {
	p.processLocation(ch)
	if p.isRowMaxSizeReached {
		if ch == endOfInput {
			p.state = stateRowEnd
		}
		return nil
	}
	if ch == '\r' {
		return nil
	}
	if p.skipTheRow {
		if p.isEndOfTheRowAndValueIsNotEmpty(ch) {
			p.insideComment = false
			p.skipTheRow = false
			p.buf = p.buf[:0]
			if ch == endOfInput {
				p.state = stateRowEnd
			}
		} else {
			p.append(ch)
			p.isValueStart = true
		}
		return nil
	}
	if p.isCurrentCsvNodeEmpty && !p.rowStarted {
		skip, err := p.isSkipped(p.rowIndex)
		if err != nil {
			return err
		}
		if skip {
			p.updateLineAndColumnIndexes()
			p.skipTheRow = true
			return nil
		}
		p.row = newRowBuilder(p.shape)
		p.rowStarted = true
		if err := p.addHeadersAsTheFirstElementIfApplicable(); err != nil {
			return err
		}
	}
	switch {
	case !p.insideComment && ch == p.cfg.comment:
		err := p.handleEndOfTheRow(true)
		p.insideComment = true
		return err
	case !p.insideComment && ch == p.cfg.delimiter:
		return p.addRowValue(true)
	case !p.insideComment && ch == p.cfg.textEnclosure:
		p.prevState = stateRowStart
		p.state = stateStringQuote
	case !p.insideComment && ch == p.cfg.escapeChar:
		p.prevState = stateRowStart
		p.state = stateStringEscape
	case p.insideComment && p.isNewLineOrEof(ch):
		p.insideComment = false
		if ch == endOfInput {
			p.state = stateRowEnd
		}
	case p.isEndOfTheRowAndValueIsNotEmpty(ch):
		if ch == endOfInput {
			p.state = stateRowEnd
		}
		return p.handleEndOfTheRow(true)
	case isWhitespace(ch):
		if p.isValueStart {
			p.append(ch)
		}
	default:
		if !p.insideComment {
			p.append(ch)
			p.isValueStart = true
		}
	}
	return nil
}

func (p *parser) isEndOfTheRowAndValueIsNotEmpty(ch rune) bool {
	return p.isNewLineOrEof(ch) && (ch == endOfInput || !p.isCurrentCsvNodeEmpty || !isBlank(p.peek()))
}

func (p *parser) stringQuote(ch rune) error {
	p.processLocation(ch)
	if ch == endOfInput {
		p.state = stateRowEnd
		return p.handleEndOfTheRow(false)
	}
	if ch == '\r' {
		return nil
	}
	switch {
	case ch == p.cfg.textEnclosure:
		if p.isQuoteClosed {
			p.append(ch)
			p.isValueStart = true
			return nil
		}
		p.isQuoteClosed = true
	case ch == p.cfg.delimiter && p.isQuoteClosed:
		p.state = stateRowStart
		p.isQuoteClosed = false
		return p.addRowValue(false)
	case p.isNewLineOrEof(ch) && p.isQuoteClosed:
		p.state = stateRowStart
		p.isQuoteClosed = false
		return p.handleEndOfTheRow(false)
	case ch == p.cfg.escapeChar:
		p.prevState = stateStringQuote
		p.state = stateStringEscape
		p.isQuoteClosed = false
	default:
		p.appendQuoted(ch)
	}
	return nil
}

var escapedCharacters = map[rune]rune{
	'"': '"', '\\': '\\', '/': '/', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t',
}

func (p *parser) escape(ch rune) error {
	p.processLocation(ch)
	if ch == endOfInput {
		p.state = stateRowEnd
		return p.handleEndOfTheRow(true)
	}
	if escaped, ok := escapedCharacters[ch]; ok {
		p.appendEscaped(escaped)
		p.state = p.prevState
		return nil
	}
	if ch == 'u' {
		if p.state == stateStringEscape {
			p.state = stateStringUnicode
		} else {
			p.state = stateHeaderUnicode
		}
		return nil
	}
	return expectedTokenError("escaped characters")
}

func (p *parser) unicode(ch rune) error {
	p.processLocation(ch)
	if ch == endOfInput {
		p.state = stateRowEnd
		return p.handleEndOfTheRow(true)
	}
	if ch == '\r' {
		return nil
	}
	if !isHexDigit(ch) {
		p.hex = p.hex[:0]
		return expectedTokenError("hexadecimal value of an unicode character")
	}
	p.hex = append(p.hex, ch)
	p.isValueStart = true
	if len(p.hex) < 4 {
		return nil
	}
	code, _ := strconv.ParseUint(string(p.hex), 16, 16)
	p.hex = p.hex[:0]
	p.appendEscaped(rune(code))
	p.state = p.prevState
	return nil
}

// appendEscaped appends an escaped UTF-16 code unit, joining a low surrogate with a preceding high
// surrogate into one character.
func (p *parser) appendEscaped(unit rune) {
	if n := len(p.buf); n > 0 && utf16.IsSurrogate(p.buf[n-1]) && unit >= 0xDC00 && unit <= 0xDFFF {
		if joined := utf16.DecodeRune(p.buf[n-1], unit); joined != '�' {
			p.buf[n-1] = joined
			return
		}
	}
	p.append(unit)
}

func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'F') || (ch >= 'a' && ch <= 'f')
}

func (p *parser) handleEndOfTheRow(trim bool) error {
	p.isValueStart = false
	if err := p.handleCsvRow(trim); err != nil {
		return err
	}
	return p.checkRequiredFields()
}

func (p *parser) checkRequiredFields() error {
	for _, name := range p.shape.fieldOrder {
		f, remaining := p.fieldHierarchy[name]
		if !remaining || (p.cfg.absentAsNilableType && f.nilable) {
			continue
		}
		if !f.optional {
			return missingFieldError(f.name)
		}
	}
	return nil
}

func (p *parser) handleCsvRow(trim bool) error {
	value := p.peek()
	if trim {
		value = javaTrim(value)
	}
	if (!isBlank(value) || p.currentCsvNodeLength != 0) && !p.isColumnMaxSizeReached && !p.isRowMaxSizeReached {
		if err := p.addRowValue(trim); err != nil {
			return err
		}
	}
	if p.isCurrentCsvNodeEmpty {
		p.updateLineAndColumnIndexesWithoutRowIndexes()
		return nil
	}
	if err := p.finalizeTheRow(); err != nil {
		return err
	}
	p.updateLineAndColumnIndexes()
	return nil
}

func (p *parser) updateLineAndColumnIndexes() {
	p.rowIndex++
	p.updateLineAndColumnIndexesWithoutRowIndexes()
}

func (p *parser) updateLineAndColumnIndexesWithoutRowIndexes() {
	p.lineNumber++
	p.row = nil
	p.rowStarted = false
	p.isCurrentCsvNodeEmpty = true
	p.columnIndex = 0
	p.isColumnMaxSizeReached = false
	p.buf = p.buf[:0]
}

func (p *parser) isSkipped(rowIndex int) (bool, error) {
	if p.skipErr != nil {
		return false, p.skipErr
	}
	return p.skipLines.contains(int64(rowIndex)), nil
}

func (p *parser) addHeadersAsTheFirstElementIfApplicable() error {
	if p.addHeadersForOutput || !p.shape.isList() || !p.cfg.outputWithHeaders {
		return nil
	}
	if len(p.headers) > 0 {
		for _, header := range p.headers {
			if err := p.addHeaderAsRowValue(header); err != nil {
				return err
			}
		}
		if !p.isCurrentCsvNodeEmpty {
			if err := p.finalizeTheRow(); err != nil {
				return err
			}
			p.row = newRowBuilder(p.shape)
		}
	}
	p.addHeadersForOutput = true
	p.columnIndex = 0
	return nil
}

func (p *parser) addHeaderAsRowValue(header string) error {
	ti, ok, err := p.expectedRowType()
	if err != nil {
		return err
	}
	if ok {
		if err := p.convertAndUpdateCurrentRow(header, ti, nil); err != nil {
			return err
		}
	}
	p.columnIndex++
	return nil
}

// finalizeTheRow adds the row to the result. A closed result array keeps its first rows and ignores the
// rest of the input once it is full.
func (p *parser) finalizeTheRow() error {
	if p.row == nil {
		return nilRowError(p.t.typeName(p.shape.ty))
	}
	row, err := p.row.build(p.t)
	if err != nil {
		return err
	}
	if p.rootSize == -1 || p.arraySize < p.rootSize {
		p.rows = append(p.rows, row)
	}
	p.arraySize++
	p.currentCsvNodeLength = 0
	if p.arraySize == p.rootSize {
		p.isRowMaxSizeReached = true
	}
	return nil
}

func (p *parser) addRowValue(trim bool) error {
	if p.isColumnMaxSizeReached || p.isRowMaxSizeReached {
		return nil
	}
	p.isValueStart = false
	value := p.value()
	if trim {
		value = javaTrim(value)
	}
	ti, ok, err := p.expectedRowType()
	if err != nil {
		return err
	}
	if ok {
		var field *recordField
		if p.shape.kind == kindRecord {
			header, err := p.headerForColumn()
			if err != nil {
				return err
			}
			field = p.fieldNames[header]
		}
		if err := p.convertAndUpdateCurrentRow(value, ti, field); err != nil {
			return err
		}
	}
	p.columnIndex++
	return nil
}

// expectedRowType is the type of the value in the current column; false means the value is dropped.
func (p *parser) expectedRowType() (typeInfo, bool, error) {
	switch p.shape.kind {
	case kindRecord:
		header, err := p.headerForColumn()
		if err != nil {
			return typeInfo{}, false, err
		}
		if f, ok := p.fieldNames[header]; ok {
			return f.info, true, nil
		}
		if p.shape.hasRest {
			return p.shape.restInfo, true, nil
		}
		if p.cfg.allowDataProjection {
			return typeInfo{}, false, nil
		}
		return typeInfo{}, false, noFieldForHeaderError(header)
	case kindMap:
		return p.shape.restInfo, true, nil
	case kindArray, kindTuple:
		if ti, ok := p.shape.memberType(p.columnIndex); ok {
			return ti, true, nil
		}
		if p.cfg.allowDataProjection {
			p.isColumnMaxSizeReached = true
			return typeInfo{}, false, nil
		}
		if p.shape.kind == kindArray {
			return typeInfo{}, false, arraySizeError(p.shape.size)
		}
		return typeInfo{}, false, tupleSizeError(p.shape.size)
	}
	return typeInfo{}, false, nil
}

// headerForColumn is the header of the current column. Without a header row or custom headers the
// columns are named "1", "2", ...
func (p *parser) headerForColumn() (string, error) {
	if p.cfg.customHeadersIfHeadersAbsent == nil && p.cfg.header == nil {
		header := strconv.Itoa(p.columnIndex + 1)
		delete(p.fieldHierarchy, header)
		return header, nil
	}
	if p.columnIndex >= len(p.headers) {
		return "", invalidHeaderCountError()
	}
	return p.headers[p.columnIndex], nil
}

func (p *parser) convertAndUpdateCurrentRow(value string, ti typeInfo, field *recordField) error {
	if p.cfg.nilAsOptionalField && !p.t.isNilable(ti.ty) && p.conv.isNullValue(value) && field != nil && field.optional {
		return nil
	}
	converted, ok := p.conv.fromString(value, ti)
	p.isCurrentCsvNodeEmpty = false
	if !ok {
		if p.ignoresIncompatibleValue() {
			return nil
		}
		return invalidCastError(value, p.t.typeName(ti.ty))
	}
	if p.row == nil {
		p.currentCsvNodeLength++
		return nil
	}
	if p.shape.isMapping() {
		header, err := p.headerForColumn()
		if err != nil {
			return err
		}
		p.row.put(header, converted)
	} else {
		p.row.set(p.columnIndex, converted)
		if p.shape.kind == kindArray && p.shape.size != -1 && p.shape.size-1 < p.columnIndex {
			p.isColumnMaxSizeReached = true
		}
	}
	p.currentCsvNodeLength++
	return nil
}

// ignoresIncompatibleValue reports whether a value that does not convert is dropped silently: any map
// value, and a record value that would go to a rest field.
func (p *parser) ignoresIncompatibleValue() bool {
	switch p.shape.kind {
	case kindRecord:
		header, err := p.headerForColumn()
		if err != nil {
			return false
		}
		_, declared := p.fieldNames[header]
		return !declared
	case kindMap:
		return true
	}
	return false
}

// javaTrim is String.trim: it removes every character up to U+0020 from both ends.
func javaTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
}

// isBlank is String.isBlank: true when the string is empty or only white space.
func isBlank(s string) bool {
	return strings.TrimFunc(s, isJavaWhitespace) == ""
}

func isJavaWhitespace(r rune) bool {
	switch r {
	case ' ', ' ', ' ':
		return false
	}
	return r == ' ' || (r >= '\t' && r <= '\r') || (r >= '\u001C' && r <= '\u001F') ||
		r == ' ' || (r >= ' ' && r <= ' ') || (r >= ' ' && r <= ' ') ||
		r == ' ' || r == ' ' || r == ' ' || r == '　'
}
