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

package parser

import (
	"github.com/ballerina-nutcracker/ballerina/parser/common"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

var deprecatedChars = []rune{'D', 'e', 'p', 'r', 'e', 'c', 'a', 't', 'e', 'd'}

type documentationLexer struct {
	*lexer
	previousBacktickMode parserMode
}

func newDocumentationLexer(charReader text.CharReader, leadingTriviaList []st.STNode, diagnostics []st.STNodeDiagnostic, reporters ...*parserFailureReporter) *documentationLexer {
	lexer := newLexer(charReader, reporters...)
	lexer.context.leadingTriviaList = leadingTriviaList
	lexer.context.diagnostics = diagnostics
	lexer.StartMode(parserModeDocLineStartHash)
	return &documentationLexer{
		lexer:                lexer,
		previousBacktickMode: parserModeDefaultMode,
	}
}

func (dl *documentationLexer) NextToken() st.STToken {
	var token st.STToken
	switch dl.context.mode {
	case parserModeDocLineStartHash:
		dl.processLeadingTrivia()
		token = dl.readDocLineStartHashToken()
	case parserModeDocLineDifferentiator:
		dl.processLeadingTrivia()
		token = dl.readDocLineDifferentiatorToken()
	case parserModeDocInternal:
		token = dl.readDocInternalToken()
	case parserModeDocParameter:
		dl.processLeadingTrivia()
		token = dl.readDocParameterToken()
	case parserModeDocReferenceType:
		dl.processLeadingTrivia()
		token = dl.readDocReferenceTypeToken()
	case parserModeDocSingleBacktickContent:
		token = dl.readSingleBacktickContentToken()
	case parserModeDocDoubleBacktickContent:
		token = dl.readCodeContent(2)
	case parserModeDocTripleBacktickContent:
		token = dl.readCodeContent(3)
	case parserModeDocCodeRefEnd:
		token = dl.readCodeReferenceEndToken()
	case parserModeDocCodeLineStartHash:
		dl.processLeadingTrivia()
		token = dl.readCodeLineStartHashToken()
	default:
		dl.reader.Mark()
		return dl.getDocSyntaxToken(st.EOF_TOKEN)
	}

	if len(dl.context.diagnostics) > 0 {
		token = st.AddSyntaxDiagnostics(token, dl.context.diagnostics)
		dl.context.diagnostics = nil
	}
	return token
}

func (dl *documentationLexer) peek() rune {
	return dl.reader.Peek()
}

func (dl *documentationLexer) getLexeme() string {
	return dl.reader.GetMarkedChars()
}

func (dl *documentationLexer) isPossibleIdentifierStart(startChar rune) bool {
	return startChar == singleQuote || startChar == backslash || isIdentifierInitialChar(startChar)
}

func (dl *documentationLexer) processIdentifierEnd() {
	reader := dl.reader
	for !reader.IsEOF() {
		nextChar := reader.Peek()
		if isIdentifierFollowingChar(nextChar) {
			reader.Advance()
			continue
		}

		if nextChar != backslash {
			break
		}

		nextChar = reader.PeekN(1)
		switch nextChar {
		case newline, carriageReturn, tab:
			reader.Advance()
			dl.reportLexerError(common.WARNING_INVALID_ESCAPE_SEQUENCE, "")
		case 'u':
			// NumericEscape
			if reader.PeekN(2) == openBrace {
				dl.processNumericEscape()
			} else {
				reader.AdvanceN(2)
			}
			continue
		default:
			reader.AdvanceN(2)
			continue
		}
		break
	}
}

func (dl *documentationLexer) processNumericEscape() {
	// Process '\ u {'
	dl.reader.AdvanceN(3)

	// Process code-point
	if !isHexDigit(byte(dl.peek())) {
		return
	}

	dl.reader.Advance()
	for isHexDigit(byte(dl.peek())) {
		dl.reader.Advance()
	}

	// Process close brace
	if dl.peek() != closeBrace {
		return
	}

	dl.reader.Advance()
}

func (dl *documentationLexer) processLeadingTrivia() {
	dl.processSyntaxTrivia(&dl.context.leadingTriviaList, true)
}

func (dl *documentationLexer) processTrailingTrivia() st.STNode {
	triviaList := make([]st.STNode, 0, initialTriviaCapacity)
	dl.processSyntaxTrivia(&triviaList, false)
	return st.CreateNodeList(triviaList...)
}

func (dl *documentationLexer) processSyntaxTrivia(triviaList *[]st.STNode, isLeading bool) {
	reader := dl.reader
	for !reader.IsEOF() {
		reader.Mark()
		c := reader.Peek()
		switch c {
		case space, tab, formFeed:
			*triviaList = append(*triviaList, dl.processWhitespaces())
		case carriageReturn, newline:
			*triviaList = append(*triviaList, dl.processEndOfLine())
			if isLeading {
				continue
			}
			return
		default:
			return
		}
	}
}

func (dl *documentationLexer) processWhitespaces() st.STNode {
	reader := dl.reader
	for !reader.IsEOF() {
		c := reader.Peek()
		switch c {
		case space, tab, formFeed:
			reader.Advance()
			continue
		case carriageReturn, newline:
		default:
		}
		break
	}
	return st.CreateMinutiae(st.WHITESPACE_MINUTIAE, dl.getLexeme())
}

func (dl *documentationLexer) processEndOfLine() st.STNode {
	reader := dl.reader
	c := reader.Peek()
	switch c {
	case newline:
		reader.Advance()
		return st.CreateMinutiae(st.END_OF_LINE_MINUTIAE, dl.getLexeme())
	case carriageReturn:
		reader.Advance()
		if reader.Peek() == newline {
			reader.Advance()
		}
		return st.CreateMinutiae(st.END_OF_LINE_MINUTIAE, dl.getLexeme())
	default:
		dl.internalError("unreachable")
		return st.CreateEmptyNode()
	}
}

func (dl *documentationLexer) getLiteral(tokenKind st.SyntaxKind) st.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()
	trailingTrivia := dl.processTrailingTrivia()
	return st.CreateLiteralValueToken(tokenKind, lexeme, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocSyntaxToken(kind st.SyntaxKind) st.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	trailingTrivia := dl.processTrailingTrivia()
	dl.checkAndTerminateCurrentMode(trailingTrivia)
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocLiteralToken(kind st.SyntaxKind) st.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()
	trailingTrivia := dl.processTrailingTrivia()
	dl.checkAndTerminateCurrentMode(trailingTrivia)
	return st.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocIdentifierToken() st.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()
	return st.CreateIdentifierToken(lexeme, leadingTrivia, st.CreateEmptyNodeList())
}

func (dl *documentationLexer) getDocSyntaxTokenWithoutTrivia(kind st.SyntaxKind) st.STToken {
	leadingTrivia := dl.getLeadingTrivia()

	var trailingTrivia st.STNode
	triviaList := make([]st.STNode, 0, 1)

	nextChar := dl.peek()
	if nextChar == newline || nextChar == carriageReturn {
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
		dl.EndMode()
	}

	trailingTrivia = st.CreateNodeList(triviaList...)
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocLiteralWithoutTrivia(kind st.SyntaxKind) st.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()

	var trailingTrivia st.STNode
	triviaList := make([]st.STNode, 0, 1)

	nextChar := dl.peek()
	if nextChar == newline || nextChar == carriageReturn {
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
		dl.EndMode()
	}

	trailingTrivia = st.CreateNodeList(triviaList...)
	return st.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getCodeStartBacktickToken(kind st.SyntaxKind) st.STToken {
	leadingTrivia := dl.getLeadingTrivia()

	var trailingTrivia st.STNode
	triviaList := make([]st.STNode, 0, 1)

	nextChar := dl.peek()
	if nextChar == newline || nextChar == carriageReturn {
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
		dl.previousBacktickMode = dl.context.mode
		dl.SwitchMode(parserModeDocCodeLineStartHash)
	}

	trailingTrivia = st.CreateNodeList(triviaList...)
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getCodeLineStartHashToken() st.STToken {
	leadingTrivia := dl.getLeadingTrivia()

	// Trivia for # in a code line can only have following 3 cases.
	// single whitespace char, newline or single whitespace char followed by a newline
	var trailingTrivia st.STNode
	triviaList := make([]st.STNode, 0, 2)

	nextChar := dl.peek()
	switch nextChar {
	case space, tab, formFeed:
		dl.reader.Mark()
		dl.reader.Advance()
		singleWhitespace := st.CreateMinutiae(st.WHITESPACE_MINUTIAE, dl.getLexeme())
		triviaList = append(triviaList, singleWhitespace)

		nextChar = dl.peek()
		if nextChar == newline || nextChar == carriageReturn {
			dl.reader.Mark()
			triviaList = append(triviaList, dl.processEndOfLine())
		} else {
			dl.SwitchMode(dl.previousBacktickMode)
		}
	case carriageReturn, newline:
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
	default:
		dl.SwitchMode(dl.previousBacktickMode)
	}

	trailingTrivia = st.CreateNodeList(triviaList...)
	return st.CreateTokenFrom(st.HASH_TOKEN, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) checkAndTerminateCurrentMode(trailingTrivia st.STNode) {
	bucketCount := trailingTrivia.BucketCount()
	if bucketCount > 0 && trailingTrivia.ChildInBucket(bucketCount-1).Kind() == st.END_OF_LINE_MINUTIAE {
		dl.EndMode()
	}
}

func (dl *documentationLexer) getLeadingTrivia() st.STNode {
	trivia := st.CreateNodeList(dl.context.leadingTriviaList...)
	dl.context.leadingTriviaList = make([]st.STNode, 0, initialTriviaCapacity)
	return trivia
}

func (dl *documentationLexer) reportLexerError(code common.DiagnosticWarningCode, args ...interface{}) {
	var diagnosticCode diagnostics.DiagnosticCode = &code
	diagnostic := st.CreateDiagnostic(diagnosticCode, args...)
	dl.context.diagnostics = append(dl.context.diagnostics, diagnostic)
}

func (dl *documentationLexer) readDocLineStartHashToken() st.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := dl.peek()
	if nextChar == hash {
		dl.reader.Advance()
		dl.StartMode(parserModeDocLineDifferentiator)
		return dl.getDocSyntaxToken(st.HASH_TOKEN)
	}

	return dl.getDocSyntaxToken(st.EOF_TOKEN)
}

func (dl *documentationLexer) readDocLineDifferentiatorToken() st.STToken {
	c := dl.peek()
	switch c {
	case plus:
		return dl.processPlusToken()
	case hash:
		dl.SwitchMode(parserModeDocInternal)
		return dl.processDeprecationLiteralToken()
	case backtick:
		if dl.reader.PeekN(1) == backtick {
			return dl.processDoubleOrTripleBacktickToken()
		}
		fallthrough
	default:
		dl.SwitchMode(parserModeDocInternal)
		return dl.readDocInternalToken()
	}
}

func (dl *documentationLexer) processPlusToken() st.STToken {
	dl.reader.Advance() // Advance for +
	dl.SwitchMode(parserModeDocParameter)
	return dl.getDocSyntaxToken(st.PLUS_TOKEN)
}

func (dl *documentationLexer) processDoubleOrTripleBacktickToken() st.STToken {
	dl.reader.AdvanceN(2) // Advance for two backticks
	if dl.peek() == backtick {
		dl.reader.Advance()
		dl.SwitchMode(parserModeDocTripleBacktickContent)
		return dl.getCodeStartBacktickToken(st.TRIPLE_BACKTICK_TOKEN)
	} else {
		dl.SwitchMode(parserModeDocDoubleBacktickContent)
		return dl.getCodeStartBacktickToken(st.DOUBLE_BACKTICK_TOKEN)
	}
}

func (dl *documentationLexer) processDeprecationLiteralToken() st.STToken {
	lookAheadCount := 1
	lookAheadChar := dl.reader.PeekN(lookAheadCount)

	whitespaceCount := 0
	for lookAheadChar == space || lookAheadChar == tab {
		lookAheadCount++
		whitespaceCount++
		lookAheadChar = dl.reader.PeekN(lookAheadCount)
	}

	// Look ahead for a "Deprecated" word match.
	for i := range 10 {
		if lookAheadChar != deprecatedChars[i] {
			// No match. Hence return a documentation internal token.
			return dl.readDocInternalToken()
		}
		lookAheadCount++
		lookAheadChar = dl.reader.PeekN(lookAheadCount)
	}

	dl.processLeadingTrivia()
	dl.reader.Mark()
	dl.reader.Advance()
	dl.reader.AdvanceN(whitespaceCount)
	dl.reader.AdvanceN(10)
	return dl.getDocLiteralWithoutTrivia(st.DEPRECATION_LITERAL)
}

func (dl *documentationLexer) readDocInternalToken() st.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := dl.peek()
	if nextChar == backtick {
		dl.reader.Advance()
		nextChar = dl.peek()
		if nextChar != backtick {
			dl.SwitchMode(parserModeDocSingleBacktickContent)
			return dl.getDocSyntaxTokenWithoutTrivia(st.BACKTICK_TOKEN)
		}

		dl.reader.Advance()
		nextChar = dl.peek()
		if nextChar != backtick {
			dl.SwitchMode(parserModeDocDoubleBacktickContent)
			return dl.getCodeStartBacktickToken(st.DOUBLE_BACKTICK_TOKEN)
		}

		dl.reader.Advance()
		dl.SwitchMode(parserModeDocTripleBacktickContent)
		return dl.getCodeStartBacktickToken(st.TRIPLE_BACKTICK_TOKEN)
	}

	for !dl.reader.IsEOF() {
		switch nextChar {
		case newline, carriageReturn:
			dl.EndMode()
		case backtick:
		default:
			if isIdentifierInitialChar(nextChar) {
				hasDocumentationReference := dl.processDocumentationReference(nextChar)
				if hasDocumentationReference {
					dl.SwitchMode(parserModeDocReferenceType)
					break
				}
			} else {
				dl.reader.Advance()
			}
			nextChar = dl.peek()
			continue
		}
		break
	}

	if dl.getLexeme() == "" {
		return dl.readDocReferenceTypeToken()
	}
	return dl.getLiteral(st.DOCUMENTATION_DESCRIPTION)
}

func (dl *documentationLexer) processDocumentationReference(nextChar rune) bool {
	lookAheadChar := nextChar
	lookAheadCount := 0
	identifier := ""

	for isIdentifierInitialChar(lookAheadChar) {
		identifier += string(lookAheadChar)
		lookAheadCount++
		lookAheadChar = dl.reader.PeekN(lookAheadCount)
	}

	switch identifier {
	case typeKeyword, service, variable, varKeyword, annotation, module, function, parameter, constKeyword:
		for {
			switch lookAheadChar {
			case space, tab:
				lookAheadCount++
				lookAheadChar = dl.reader.PeekN(lookAheadCount)
				continue
			case backtick:
				if dl.reader.PeekN(lookAheadCount+1) != backtick {
					return true
				}
			default:
			}
			break
		}
		dl.reader.AdvanceN(lookAheadCount)
		return false
	default:
		dl.reader.AdvanceN(lookAheadCount)
		return false
	}
}

func (dl *documentationLexer) readDocParameterToken() st.STToken {
	dl.reader.Mark()
	nextChar := dl.peek()
	if dl.isPossibleIdentifierStart(nextChar) {
		if nextChar != backslash {
			dl.reader.Advance()
		}

		dl.processIdentifierEnd()
		var token st.STToken
		if dl.getLexeme() == returnKeyword {
			token = dl.getDocSyntaxToken(st.RETURN_KEYWORD)
		} else {
			token = dl.getDocLiteralToken(st.PARAMETER_NAME)
		}
		// If the parameter name is not followed by a minus token switch the mode.
		// However, if the parameter name ends with a newline DOC_PARAMETER mode is already ended.
		// Therefore, DOC_LINE_START_HASH is the active mode. In that case do not switch mode.
		if dl.peek() != minus && dl.context.mode != parserModeDocLineStartHash {
			dl.SwitchMode(parserModeDocInternal)
		}
		return token
	} else if nextChar == minus {
		dl.reader.Advance()
		dl.SwitchMode(parserModeDocInternal)
		return dl.getDocSyntaxToken(st.MINUS_TOKEN)
	} else {
		dl.SwitchMode(parserModeDocInternal)
		return dl.readDocInternalToken()
	}
}

func (dl *documentationLexer) readDocReferenceTypeToken() st.STToken {
	dl.reader.Mark()
	nextChar := dl.peek()
	if nextChar == backtick {
		dl.reader.Advance()
		dl.SwitchMode(parserModeDocSingleBacktickContent)
		return dl.getDocSyntaxTokenWithoutTrivia(st.BACKTICK_TOKEN)
	}

	for isIdentifierInitialChar(dl.peek()) {
		dl.reader.Advance()
	}
	return dl.processReferenceType()
}

func (dl *documentationLexer) processReferenceType() st.STToken {
	tokenText := dl.getLexeme()
	switch tokenText {
	case typeKeyword:
		return dl.getDocSyntaxToken(st.TYPE_DOC_REFERENCE_TOKEN)
	case service:
		return dl.getDocSyntaxToken(st.SERVICE_DOC_REFERENCE_TOKEN)
	case variable:
		return dl.getDocSyntaxToken(st.VARIABLE_DOC_REFERENCE_TOKEN)
	case varKeyword:
		return dl.getDocSyntaxToken(st.VAR_DOC_REFERENCE_TOKEN)
	case annotation:
		return dl.getDocSyntaxToken(st.ANNOTATION_DOC_REFERENCE_TOKEN)
	case module:
		return dl.getDocSyntaxToken(st.MODULE_DOC_REFERENCE_TOKEN)
	case function:
		return dl.getDocSyntaxToken(st.FUNCTION_DOC_REFERENCE_TOKEN)
	case parameter:
		return dl.getDocSyntaxToken(st.PARAMETER_DOC_REFERENCE_TOKEN)
	case constKeyword:
		return dl.getDocSyntaxToken(st.CONST_DOC_REFERENCE_TOKEN)
	default:
		return dl.getDocSyntaxToken(st.EOF_TOKEN)
	}
}

func (dl *documentationLexer) readSingleBacktickContentToken() st.STToken {
	dl.reader.Mark()
	nextChar := dl.peek()
	if nextChar == backslash {
		dl.processIdentifierEnd()
		return dl.getDocIdentifierToken()
	}

	dl.reader.Advance()
	switch nextChar {
	case backtick:
		dl.SwitchMode(parserModeDocInternal)
		return dl.getDocSyntaxTokenWithoutTrivia(st.BACKTICK_TOKEN)
	case dot:
		return dl.getDocSyntaxToken(st.DOT_TOKEN)
	case colon:
		return dl.getDocSyntaxToken(st.COLON_TOKEN)
	case openParanthesis:
		return dl.getDocSyntaxToken(st.OPEN_PAREN_TOKEN)
	case closeParanthesis:
		return dl.getDocSyntaxToken(st.CLOSE_PAREN_TOKEN)
	default:
		if dl.isPossibleIdentifierStart(nextChar) {
			dl.processIdentifierEnd()
			return dl.getDocIdentifierToken()
		}

		dl.processInvalidChars()
		return dl.getDocLiteralToken(st.CODE_CONTENT)
	}
}

func (dl *documentationLexer) processInvalidChars() {
	nextChar := dl.peek()
	for !dl.reader.IsEOF() {
		switch nextChar {
		case backtick, newline, carriageReturn:
		default:
			dl.reader.Advance()
			nextChar = dl.peek()
			continue
		}
		break
	}
}

func (dl *documentationLexer) readCodeContent(backtickCount int) st.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := dl.peek()
	for !dl.reader.IsEOF() {
		switch nextChar {
		case backtick:
			count := dl.getBackticksCount()
			if count == backtickCount {
				dl.SwitchMode(parserModeDocCodeRefEnd)
				break
			}
			dl.reader.AdvanceN(count)
			nextChar = dl.peek()
			continue
		case carriageReturn, newline:
			dl.previousBacktickMode = dl.context.mode
			dl.SwitchMode(parserModeDocCodeLineStartHash)
		default:
			dl.reader.Advance()
			nextChar = dl.peek()
			continue
		}
		break
	}

	if dl.getLexeme() == "" {
		// We only reach here for ``<empty_code>`` and ```<empty_code>```
		return dl.readCodeReferenceEndToken()
	}
	return dl.getLiteral(st.CODE_CONTENT)
}

func (dl *documentationLexer) getBackticksCount() int {
	count := 1
	for dl.reader.PeekN(count) == backtick {
		count += 1
	}
	return count
}

func (dl *documentationLexer) readCodeReferenceEndToken() st.STToken {
	dl.SwitchMode(parserModeDocInternal)
	if dl.peek() == backtick {
		dl.reader.Advance()
		if dl.peek() == backtick {
			dl.reader.Advance()
			if dl.peek() == backtick {
				dl.reader.Advance()
				return dl.getDocSyntaxTokenWithoutTrivia(st.TRIPLE_BACKTICK_TOKEN)
			} else {
				return dl.getDocSyntaxTokenWithoutTrivia(st.DOUBLE_BACKTICK_TOKEN)
			}
		}
	}
	return dl.getDocSyntaxToken(st.EOF_TOKEN)
}

func (dl *documentationLexer) readCodeLineStartHashToken() st.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(st.EOF_TOKEN)
	}
	nextChar := dl.peek()
	if nextChar == hash {
		dl.reader.Advance()
		return dl.getCodeLineStartHashToken()
	}
	return dl.getDocSyntaxToken(st.EOF_TOKEN)
}
