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

// Port of jBallerina's JsonParser state machine, which converts JSON text to the expected type while it
// reads it. It works on UTF-16 code units like the Java implementation, so line and column numbers in
// error messages match, and keeps the Java parser's stacks so that projection, union handling and field
// paths behave the same.

package native

import (
	"errors"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	charCR        = 0x000D
	charNewline   = 0x000A
	charTab       = 0x0009
	charSpace     = 0x0020
	charBackspace = 0x0008
	charFormfeed  = 0x000C
	charQuote     = '"'
	charBackslash = '\\'
	charSlash     = '/'
	charEOF       = 0xFFFF
)

type parserState int

const (
	stateDocStart parserState = iota
	stateDocEnd
	stateFirstFieldReady
	stateNonFirstFieldReady
	stateFieldName
	stateEndFieldName
	stateFieldValueReady
	stateStringFieldValue
	stateNonStringFieldValue
	stateNonStringValue
	stateStringValue
	stateFieldEnd
	stateStringArrayElementEscape
	stateStringArrayElementUnicode
	stateFieldNameUnicode
	stateFirstArrayElementReady
	stateNonFirstArrayElementReady
	stateStringArrayElement
	stateNonStringArrayElement
	stateArrayElementEnd
	stateStringFieldEscape
	stateStringValueEscape
	stateFieldNameEscape
	stateStringFieldUnicode
	stateStringValueUnicode
)

type parserContext int

const (
	contextMap parserContext = iota
	contextArray
)

// parseException is the Java JsonParserException: a syntax error reported with the line and column.
type parseException struct {
	message string
}

func (e *parseException) Error() string {
	return e.message
}

func throwExpected(chars ...string) error {
	message := "expected '" + chars[0]
	for _, ch := range chars[1:] {
		message += "' or '" + ch
	}
	return &parseException{message: message + "'"}
}

var errInvalidDocument = errors.New("invalid JSON document")

type stateMachine struct {
	*typer
	projection
	// currentNode is a *mappingBuilder or *listBuilder while a structure is being read, otherwise the
	// converted value.
	currentNode           any
	nodes                 []any
	hexBuilder            []uint16
	charBuff              []uint16
	index                 int
	line                  int
	column                int
	currentField          *recordField
	fieldHierarchy        []*fieldSet
	visitedFieldHierarchy []map[string]*recordField
	restTypes             []*semtypes.SemType
	expectedTypes         []*semtypes.SemType
	fieldNameHierarchy    [][]string
	jsonFieldDepth        int
	arrayIndexes          []int
	parserContexts        []parserContext
	unionDepth            int
}

func parse(t *typer, opts projection, input []uint16, target semtypes.SemType) (values.BalValue, error) {
	sm := &stateMachine{typer: t, projection: opts, line: 1}
	return sm.execute(input, target)
}

// utf16FromString and utf16FromBytes decode UTF-8 straight into UTF-16 code units. A UTF-16 encoding never
// has more code units than the UTF-8 encoding has bytes.
func utf16FromString(input string) []uint16 {
	buff := make([]uint16, 0, len(input))
	for _, r := range input {
		buff = utf16.AppendRune(buff, r)
	}
	return buff
}

// utf16FromBytes rejects a byte sequence that is not valid UTF-8; jBallerina replaces or drops it.
func utf16FromBytes(input []byte) ([]uint16, error) {
	buff := make([]uint16, 0, len(input))
	for offset := 0; offset < len(input); {
		r, size := utf8.DecodeRune(input[offset:])
		if r == utf8.RuneError && size == 1 {
			return nil, newJSONError("invalid UTF-8 byte at offset %d", offset)
		}
		buff = utf16.AppendRune(buff, r)
		offset += size
	}
	return buff, nil
}

func (sm *stateMachine) execute(buff []uint16, target semtypes.SemType) (values.BalValue, error) {
	ti := sm.classify(target)
	switch ti.kind {
	case kindRecord, kindMap:
		sm.expectedTypes = append(sm.expectedTypes, &target)
		fields, err := sm.mappingFields(ti)
		if err != nil {
			return nil, err
		}
		sm.updateExpectedType(fields, restTypeRef(ti))
	case kindList:
		sm.expectedTypes = append(sm.expectedTypes, &target)
		sm.arrayIndexes = append(sm.arrayIndexes, 0)
	case kindJSON, kindAnydata:
		sm.expectedTypes = append(sm.expectedTypes, &target)
		sm.updateExpectedType(emptyFieldSet(), &target)
	case kindUnsupported:
		return nil, unsupportedTypeError(sm.typeName(target))
	default:
		sm.expectedTypes = append(sm.expectedTypes, &target)
	}

	state := stateDocStart
	var err error
	for sm.index < len(buff) && err == nil {
		state, err = sm.transition(state, buff, sm.index, len(buff))
	}
	if err == nil {
		state, err = sm.transition(state, []uint16{charEOF}, 0, 1)
	}
	if err == nil && state != stateDocEnd {
		err = errInvalidDocument
	}
	if err != nil {
		var syntaxErr *parseException
		if errors.As(err, &syntaxErr) {
			return nil, parserExceptionError(syntaxErr.message, sm.line, sm.column)
		}
		return nil, err
	}
	return sm.materialize(sm.currentNode)
}

func (sm *stateMachine) mappingFields(ti typeInfo) (*fieldSet, error) {
	if ti.kind == kindRecord {
		return sm.recordFields(ti)
	}
	return emptyFieldSet(), nil
}

func restTypeRef(ti typeInfo) *semtypes.SemType {
	if rest, ok := restType(ti); ok {
		return &rest
	}
	return nil
}

func (sm *stateMachine) materialize(node any) (values.BalValue, error) {
	switch node := node.(type) {
	case *mappingBuilder:
		return node.build(sm.typer), nil
	case *listBuilder:
		return node.build(sm.typer)
	default:
		return node, nil
	}
}

func (sm *stateMachine) transition(state parserState, buff []uint16, i, count int) (parserState, error) {
	switch state {
	case stateDocStart:
		return sm.docStart(buff, i, count)
	case stateDocEnd:
		return sm.docEnd(buff, i, count)
	case stateFirstFieldReady:
		return sm.firstFieldReady(buff, i, count)
	case stateNonFirstFieldReady:
		return sm.nonFirstFieldReady(buff, i, count)
	case stateFieldName:
		return sm.fieldName(buff, i, count)
	case stateEndFieldName:
		return sm.endFieldName(buff, i, count)
	case stateFieldValueReady:
		return sm.fieldValueReady(buff, i, count)
	case stateStringFieldValue:
		return sm.stringContent(buff, i, count, stateStringFieldValue)
	case stateNonStringFieldValue:
		return sm.nonStringFieldValue(buff, i, count)
	case stateNonStringValue:
		return sm.nonStringValue(buff, i, count)
	case stateStringValue:
		return sm.stringContent(buff, i, count, stateStringValue)
	case stateFieldEnd:
		return sm.fieldEnd(buff, i, count)
	case stateFirstArrayElementReady:
		return sm.arrayElementReady(buff, i, count, true)
	case stateNonFirstArrayElementReady:
		return sm.arrayElementReady(buff, i, count, false)
	case stateStringArrayElement:
		return sm.stringContent(buff, i, count, stateStringArrayElement)
	case stateNonStringArrayElement:
		return sm.nonStringArrayElement(buff, i, count)
	case stateArrayElementEnd:
		return sm.arrayElementEnd(buff, i, count)
	case stateStringFieldEscape:
		return sm.escapedCharacter(buff, i, count, stateStringFieldValue)
	case stateStringArrayElementEscape:
		return sm.escapedCharacter(buff, i, count, stateStringArrayElement)
	case stateStringValueEscape:
		return sm.escapedCharacter(buff, i, count, stateStringValue)
	case stateFieldNameEscape:
		return sm.escapedCharacter(buff, i, count, stateFieldName)
	case stateStringFieldUnicode:
		return sm.unicodeHex(buff, i, count, stateStringFieldValue)
	case stateStringArrayElementUnicode:
		return sm.unicodeHex(buff, i, count, stateStringArrayElement)
	case stateStringValueUnicode:
		return sm.unicodeHex(buff, i, count, stateStringValue)
	default:
		return sm.unicodeHex(buff, i, count, stateFieldName)
	}
}

func isWhitespace(ch uint16) bool {
	return ch == charSpace || ch == charTab || ch == charNewline || ch == charCR
}

func (sm *stateMachine) processLocation(ch uint16) {
	if ch == '\n' {
		sm.line++
		sm.column = 0
	} else {
		sm.column++
	}
}

func (sm *stateMachine) appendChar(ch uint16) {
	sm.charBuff = append(sm.charBuff, ch)
}

func (sm *stateMachine) value() string {
	result := string(utf16.Decode(sm.charBuff))
	sm.charBuff = sm.charBuff[:0]
	return result
}

func (sm *stateMachine) docStart(buff []uint16, i, count int) (parserState, error) {
	state := stateDocStart
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch {
		case ch == '{':
			node, err := sm.initRootMapValue()
			if err != nil {
				return state, err
			}
			sm.currentNode = node
			state = stateFirstFieldReady
		case ch == '[':
			node, err := sm.initRootArrayValue()
			if err != nil {
				return state, err
			}
			sm.currentNode = node
			state = stateFirstArrayElementReady
		case isWhitespace(ch):
			continue
		case ch == charQuote:
			state = stateStringValue
		case ch == charEOF:
			return state, &parseException{message: "empty JSON document"}
		default:
			state = stateNonStringValue
		}
		break
	}
	sm.setIndex(state, stateNonStringValue, i)
	return state, nil
}

// setIndex resumes at i when the state needs to see the current character again, past it otherwise.
func (sm *stateMachine) setIndex(state, rescanState parserState, i int) {
	if state == rescanState {
		sm.index = i
	} else {
		sm.index = i + 1
	}
}

func (sm *stateMachine) docEnd(buff []uint16, i, count int) (parserState, error) {
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		if isWhitespace(ch) || ch == charEOF {
			continue
		}
		return stateDocEnd, &parseException{message: "JSON document has already ended"}
	}
	sm.index = i + 1
	if sm.unionDepth > 0 {
		converted, err := sm.verifyAndConvertToUnion(sm.currentNode)
		if err != nil {
			return stateDocEnd, err
		}
		sm.currentNode = converted
	}
	return stateDocEnd, nil
}

func (sm *stateMachine) firstFieldReady(buff []uint16, i, count int) (parserState, error) {
	state := stateFirstFieldReady
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch {
		case ch == charQuote:
			state = stateFieldName
		case isWhitespace(ch):
			continue
		case ch == '}':
			next, err := sm.checkUnionAndFinalizeNonArrayObject()
			if err != nil {
				return state, err
			}
			state = next
		default:
			return state, throwExpected("\"", "}")
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) nonFirstFieldReady(buff []uint16, i, count int) (parserState, error) {
	state := stateNonFirstFieldReady
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch {
		case ch == charQuote:
			state = stateFieldName
		case isWhitespace(ch):
			continue
		default:
			return state, throwExpected("\"")
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) arrayElementReady(buff []uint16, i, count int, first bool) (parserState, error) {
	state := stateNonFirstArrayElementReady
	if first {
		state = stateFirstArrayElementReady
	}
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		var err error
		switch {
		case isWhitespace(ch):
			continue
		case ch == charQuote:
			state = stateStringArrayElement
			err = sm.updateMemberExpectedType()
		case ch == '{':
			err = sm.updateNextMapValueBasedOnExpType()
			state = stateFirstFieldReady
		case ch == '[':
			err = sm.updateNextArrayValueBasedOnExpType()
			state = stateFirstArrayElementReady
		case first && ch == ']':
			state, err = sm.checkUnionAndFinalizeArrayObject()
		default:
			state = stateNonStringArrayElement
			err = sm.updateMemberExpectedType()
		}
		if err != nil {
			return state, err
		}
		break
	}
	sm.setIndex(state, stateNonStringArrayElement, i)
	return state, nil
}

func (sm *stateMachine) fieldName(buff []uint16, i, count int) (parserState, error) {
	state := stateFieldName
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch ch {
		case charQuote:
			if err := sm.handleFieldName(sm.value()); err != nil {
				return state, err
			}
			state = stateEndFieldName
		case charBackslash:
			state = stateFieldNameEscape
		case charEOF:
			return state, &parseException{message: "unexpected end of JSON document"}
		default:
			sm.appendChar(ch)
			continue
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) endFieldName(buff []uint16, i, count int) (parserState, error) {
	state := stateEndFieldName
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch {
		case isWhitespace(ch):
			continue
		case ch == ':':
			state = stateFieldValueReady
		default:
			return state, throwExpected(":")
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) fieldValueReady(buff []uint16, i, count int) (parserState, error) {
	state := stateFieldValueReady
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		var err error
		switch {
		case isWhitespace(ch):
			continue
		case ch == charQuote:
			state = stateStringFieldValue
		case ch == '{':
			err = sm.updateNextMapValue()
			state = stateFirstFieldReady
		case ch == '[':
			err = sm.updateNextArrayValue()
			state = stateFirstArrayElementReady
		default:
			state = stateNonStringFieldValue
		}
		if err != nil {
			return state, err
		}
		break
	}
	sm.setIndex(state, stateNonStringFieldValue, i)
	return state, nil
}

// stringContent reads the characters of a string field value, array element or root value.
func (sm *stateMachine) stringContent(buff []uint16, i, count int, source parserState) (parserState, error) {
	state := source
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch ch {
		case charQuote:
			next, err := sm.endString(source)
			if err != nil {
				return state, err
			}
			state = next
		case charBackslash:
			state = escapeStateOf(source)
		case charEOF:
			return state, &parseException{message: "unexpected end of JSON document"}
		default:
			sm.appendChar(ch)
			continue
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) endString(source parserState) (parserState, error) {
	switch source {
	case stateStringFieldValue:
		return stateFieldEnd, sm.processValue(true)
	case stateStringArrayElement:
		return stateArrayElementEnd, sm.processValue(true)
	default:
		value := sm.value()
		expType := &semtypes.String
		if sm.unionDepth == 0 {
			expType = sm.peekExpectedType()
		}
		converted, err := sm.convertAndUpdateCurrentNode(value, *expType, true)
		if err != nil {
			return stateDocEnd, err
		}
		sm.checkNullAndUpdateCurrentNode(converted)
		return stateDocEnd, nil
	}
}

func escapeStateOf(source parserState) parserState {
	switch source {
	case stateStringFieldValue:
		return stateStringFieldEscape
	case stateStringArrayElement:
		return stateStringArrayElementEscape
	case stateStringValue:
		return stateStringValueEscape
	default:
		return stateFieldNameEscape
	}
}

func unicodeStateOf(source parserState) parserState {
	switch source {
	case stateStringFieldValue:
		return stateStringFieldUnicode
	case stateStringArrayElement:
		return stateStringArrayElementUnicode
	case stateStringValue:
		return stateStringValueUnicode
	default:
		return stateFieldNameUnicode
	}
}

func (sm *stateMachine) nonStringFieldValue(buff []uint16, i, count int) (parserState, error) {
	state := stateNonStringFieldValue
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		var err error
		switch {
		case ch == '{':
			err = sm.updateNextMapValue()
			state = stateFirstFieldReady
		case ch == '[':
			state = stateFirstArrayElementReady
			err = sm.updateNextArrayValueBasedOnExpType()
		case ch == '}':
			if err = sm.processValue(false); err == nil {
				state, err = sm.checkUnionAndFinalizeNonArrayObject()
			}
		case ch == ']':
			if err = sm.processValue(false); err == nil {
				state, err = sm.checkUnionAndFinalizeArrayObject()
			}
		case ch == ',':
			err = sm.processValue(false)
			state = stateNonFirstFieldReady
		case isWhitespace(ch):
			err = sm.processValue(false)
			state = stateFieldEnd
		case ch == charEOF:
			return state, &parseException{message: "unexpected end of JSON document"}
		default:
			sm.appendChar(ch)
			continue
		}
		if err != nil {
			return state, err
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) nonStringArrayElement(buff []uint16, i, count int) (parserState, error) {
	state := stateNonStringArrayElement
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		var err error
		switch {
		case ch == '{':
			err = sm.updateNextMapValue()
			state = stateFirstFieldReady
		case ch == '[':
			err = sm.updateNextArrayValueBasedOnExpType()
			state = stateFirstArrayElementReady
		case ch == ']':
			if err = sm.processValue(false); err == nil {
				state, err = sm.checkUnionAndFinalizeArrayObject()
			}
		case ch == ',':
			err = sm.processValue(false)
			state = stateNonFirstArrayElementReady
			sm.updateIndexOfArrayElement()
		case isWhitespace(ch):
			err = sm.processValue(false)
			state = stateArrayElementEnd
		case ch == charEOF:
			return state, &parseException{message: "unexpected end of JSON document"}
		default:
			sm.appendChar(ch)
			continue
		}
		if err != nil {
			return state, err
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) nonStringValue(buff []uint16, i, count int) (parserState, error) {
	state := stateNonStringValue
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		if isWhitespace(ch) || ch == charEOF {
			sm.currentNode = nil
			if err := sm.processValue(false); err != nil {
				return state, err
			}
			state = stateDocEnd
			break
		}
		sm.appendChar(ch)
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) fieldEnd(buff []uint16, i, count int) (parserState, error) {
	state := stateFieldEnd
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch {
		case isWhitespace(ch):
			continue
		case ch == ',':
			state = stateNonFirstFieldReady
		case ch == '}':
			next, err := sm.checkUnionAndFinalizeNonArrayObject()
			if err != nil {
				return state, err
			}
			state = next
		default:
			return state, throwExpected(",", "}")
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) arrayElementEnd(buff []uint16, i, count int) (parserState, error) {
	state := stateArrayElementEnd
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		switch {
		case isWhitespace(ch):
			continue
		case ch == ',':
			sm.updateIndexOfArrayElement()
			state = stateNonFirstArrayElementReady
		case ch == ']':
			next, err := sm.checkUnionAndFinalizeArrayObject()
			if err != nil {
				return state, err
			}
			state = next
		default:
			return state, throwExpected(",", "]")
		}
		break
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) escapedCharacter(buff []uint16, i, count int, source parserState) (parserState, error) {
	var state parserState
	if i < count {
		ch := buff[i]
		sm.processLocation(ch)
		state = source
		switch ch {
		case '"':
			sm.appendChar(charQuote)
		case '\\':
			sm.appendChar(charBackslash)
		case '/':
			sm.appendChar(charSlash)
		case 'b':
			sm.appendChar(charBackspace)
		case 'f':
			sm.appendChar(charFormfeed)
		case 'n':
			sm.appendChar(charNewline)
		case 'r':
			sm.appendChar(charCR)
		case 't':
			sm.appendChar(charTab)
		case 'u':
			state = unicodeStateOf(source)
		default:
			return state, throwExpected("escaped characters")
		}
	}
	sm.index = i + 1
	return state, nil
}

func (sm *stateMachine) unicodeHex(buff []uint16, i, count int, source parserState) (parserState, error) {
	state := unicodeStateOf(source)
	for ; i < count; i++ {
		ch := buff[i]
		sm.processLocation(ch)
		if !isHexDigit(ch) {
			sm.hexBuilder = sm.hexBuilder[:0]
			return state, throwExpected("hexadecimal value of an unicode character")
		}
		sm.hexBuilder = append(sm.hexBuilder, ch)
		if len(sm.hexBuilder) >= 4 {
			code, _ := strconv.ParseUint(string(utf16.Decode(sm.hexBuilder)), 16, 16)
			sm.appendChar(uint16(code))
			sm.hexBuilder = sm.hexBuilder[:0]
			state = source
			break
		}
	}
	sm.index = i + 1
	return state, nil
}

func isHexDigit(ch uint16) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'F') || (ch >= 'a' && ch <= 'f')
}
