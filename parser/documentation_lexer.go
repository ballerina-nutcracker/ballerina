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
	"ballerina/parser/common"
	"ballerina/parser/tree"
	"ballerina/tools/diagnostics"
	"ballerina/tools/text"
)

var deprecatedChars = []rune{'D', 'e', 'p', 'r', 'e', 'c', 'a', 't', 'e', 'd'}

type documentationLexer struct {
	*lexer
	previousBacktickMode ParserMode
}

func newDocumentationLexer(charReader text.CharReader, leadingTriviaList []tree.STNode, diagnostics []tree.STNodeDiagnostic) *documentationLexer {
	lexer := NewLexer(charReader)
	lexer.context.leadingTriviaList = leadingTriviaList
	lexer.context.diagnostics = diagnostics
	lexer.StartMode(PARSER_MODE_DOC_LINE_START_HASH)
	return &documentationLexer{
		lexer:                lexer,
		previousBacktickMode: PARSER_MODE_DEFAULT_MODE,
	}
}

func (dl *documentationLexer) NextToken() tree.STToken {
	var token tree.STToken
	switch dl.context.mode {
	case PARSER_MODE_DOC_LINE_START_HASH:
		dl.processLeadingTrivia()
		token = dl.readDocLineStartHashToken()
	case PARSER_MODE_DOC_LINE_DIFFERENTIATOR:
		dl.processLeadingTrivia()
		token = dl.readDocLineDifferentiatorToken()
	case PARSER_MODE_DOC_INTERNAL:
		token = dl.readDocInternalToken()
	case PARSER_MODE_DOC_PARAMETER:
		dl.processLeadingTrivia()
		token = dl.readDocParameterToken()
	case PARSER_MODE_DOC_REFERENCE_TYPE:
		dl.processLeadingTrivia()
		token = dl.readDocReferenceTypeToken()
	case PARSER_MODE_DOC_SINGLE_BACKTICK_CONTENT:
		token = dl.readSingleBacktickContentToken()
	case PARSER_MODE_DOC_DOUBLE_BACKTICK_CONTENT:
		token = dl.readCodeContent(2)
	case PARSER_MODE_DOC_TRIPLE_BACKTICK_CONTENT:
		token = dl.readCodeContent(3)
	case PARSER_MODE_DOC_CODE_REF_END:
		token = dl.readCodeReferenceEndToken()
	case PARSER_MODE_DOC_CODE_LINE_START_HASH:
		dl.processLeadingTrivia()
		token = dl.readCodeLineStartHashToken()
	default:
		dl.reader.Mark()
		return dl.getDocSyntaxToken(common.EOF_TOKEN)
	}

	if len(dl.context.diagnostics) > 0 {
		token = tree.AddSyntaxDiagnostics(token, dl.context.diagnostics)
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
	return startChar == SINGLE_QUOTE || startChar == BACKSLASH || isIdentifierInitialChar(startChar)
}

func (dl *documentationLexer) processIdentifierEnd() {
	reader := dl.reader
	for !reader.IsEOF() {
		nextChar := reader.Peek()
		if isIdentifierFollowingChar(nextChar) {
			reader.Advance()
			continue
		}

		if nextChar != BACKSLASH {
			break
		}

		nextChar = reader.PeekN(1)
		switch nextChar {
		case NEWLINE, CARRIAGE_RETURN, TAB:
			reader.Advance()
			dl.reportLexerError(common.WARNING_INVALID_ESCAPE_SEQUENCE, "")
		case 'u':
			// NumericEscape
			if reader.PeekN(2) == OPEN_BRACE {
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
	if dl.peek() != CLOSE_BRACE {
		return
	}

	dl.reader.Advance()
}

func (dl *documentationLexer) processLeadingTrivia() {
	dl.processSyntaxTrivia(&dl.context.leadingTriviaList, true)
}

func (dl *documentationLexer) processTrailingTrivia() tree.STNode {
	triviaList := make([]tree.STNode, 0, INITIAL_TRIVIA_CAPACITY)
	dl.processSyntaxTrivia(&triviaList, false)
	return tree.CreateNodeList(triviaList...)
}

func (dl *documentationLexer) processSyntaxTrivia(triviaList *[]tree.STNode, isLeading bool) {
	reader := dl.reader
	for !reader.IsEOF() {
		reader.Mark()
		c := reader.Peek()
		switch c {
		case SPACE, TAB, FORM_FEED:
			*triviaList = append(*triviaList, dl.processWhitespaces())
		case CARRIAGE_RETURN, NEWLINE:
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

func (dl *documentationLexer) processWhitespaces() tree.STNode {
	reader := dl.reader
	for !reader.IsEOF() {
		c := reader.Peek()
		switch c {
		case SPACE, TAB, FORM_FEED:
			reader.Advance()
			continue
		case CARRIAGE_RETURN, NEWLINE:
		default:
		}
		break
	}
	return tree.CreateMinutiae(common.WHITESPACE_MINUTIAE, dl.getLexeme())
}

func (dl *documentationLexer) processEndOfLine() tree.STNode {
	reader := dl.reader
	c := reader.Peek()
	switch c {
	case NEWLINE:
		reader.Advance()
		return tree.CreateMinutiae(common.END_OF_LINE_MINUTIAE, dl.getLexeme())
	case CARRIAGE_RETURN:
		reader.Advance()
		if reader.Peek() == NEWLINE {
			reader.Advance()
		}
		return tree.CreateMinutiae(common.END_OF_LINE_MINUTIAE, dl.getLexeme())
	default:
		panic("unreachable")
	}
}

func (dl *documentationLexer) getLiteral(tokenKind common.SyntaxKind) tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()
	trailingTrivia := dl.processTrailingTrivia()
	return tree.CreateLiteralValueToken(tokenKind, lexeme, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocSyntaxToken(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	trailingTrivia := dl.processTrailingTrivia()
	dl.checkAndTerminateCurrentMode(trailingTrivia)
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocLiteralToken(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()
	trailingTrivia := dl.processTrailingTrivia()
	dl.checkAndTerminateCurrentMode(trailingTrivia)
	return tree.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocIdentifierToken() tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()
	return tree.CreateIdentifierToken(lexeme, leadingTrivia, tree.CreateEmptyNodeList())
}

func (dl *documentationLexer) getDocSyntaxTokenWithoutTrivia(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()

	var trailingTrivia tree.STNode
	triviaList := make([]tree.STNode, 0, 1)

	nextChar := dl.peek()
	if nextChar == NEWLINE || nextChar == CARRIAGE_RETURN {
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
		dl.EndMode()
	}

	trailingTrivia = tree.CreateNodeList(triviaList...)
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getDocLiteralWithoutTrivia(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()
	lexeme := dl.getLexeme()

	var trailingTrivia tree.STNode
	triviaList := make([]tree.STNode, 0, 1)

	nextChar := dl.peek()
	if nextChar == NEWLINE || nextChar == CARRIAGE_RETURN {
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
		dl.EndMode()
	}

	trailingTrivia = tree.CreateNodeList(triviaList...)
	return tree.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getCodeStartBacktickToken(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()

	var trailingTrivia tree.STNode
	triviaList := make([]tree.STNode, 0, 1)

	nextChar := dl.peek()
	if nextChar == NEWLINE || nextChar == CARRIAGE_RETURN {
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
		dl.previousBacktickMode = dl.context.mode
		dl.SwitchMode(PARSER_MODE_DOC_CODE_LINE_START_HASH)
	}

	trailingTrivia = tree.CreateNodeList(triviaList...)
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) getCodeLineStartHashToken() tree.STToken {
	leadingTrivia := dl.getLeadingTrivia()

	// Trivia for # in a code line can only have following 3 cases.
	// single whitespace char, newline or single whitespace char followed by a newline
	var trailingTrivia tree.STNode
	triviaList := make([]tree.STNode, 0, 2)

	nextChar := dl.peek()
	switch nextChar {
	case SPACE, TAB, FORM_FEED:
		dl.reader.Mark()
		dl.reader.Advance()
		singleWhitespace := tree.CreateMinutiae(common.WHITESPACE_MINUTIAE, dl.getLexeme())
		triviaList = append(triviaList, singleWhitespace)

		nextChar = dl.peek()
		if nextChar == NEWLINE || nextChar == CARRIAGE_RETURN {
			dl.reader.Mark()
			triviaList = append(triviaList, dl.processEndOfLine())
		} else {
			dl.SwitchMode(dl.previousBacktickMode)
		}
	case CARRIAGE_RETURN, NEWLINE:
		dl.reader.Mark()
		triviaList = append(triviaList, dl.processEndOfLine())
	default:
		dl.SwitchMode(dl.previousBacktickMode)
	}

	trailingTrivia = tree.CreateNodeList(triviaList...)
	return tree.CreateTokenFrom(common.HASH_TOKEN, leadingTrivia, trailingTrivia)
}

func (dl *documentationLexer) checkAndTerminateCurrentMode(trailingTrivia tree.STNode) {
	bucketCount := trailingTrivia.BucketCount()
	if bucketCount > 0 && trailingTrivia.ChildInBucket(bucketCount-1).Kind() == common.END_OF_LINE_MINUTIAE {
		dl.EndMode()
	}
}

func (dl *documentationLexer) getLeadingTrivia() tree.STNode {
	trivia := tree.CreateNodeList(dl.context.leadingTriviaList...)
	dl.context.leadingTriviaList = make([]tree.STNode, 0, INITIAL_TRIVIA_CAPACITY)
	return trivia
}

func (dl *documentationLexer) reportLexerError(code common.DiagnosticWarningCode, args ...interface{}) {
	var diagnosticCode diagnostics.DiagnosticCode = &code
	diagnostic := tree.CreateDiagnostic(diagnosticCode, args...)
	dl.context.diagnostics = append(dl.context.diagnostics, diagnostic)
}

func (dl *documentationLexer) readDocLineStartHashToken() tree.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := dl.peek()
	if nextChar == HASH {
		dl.reader.Advance()
		dl.StartMode(PARSER_MODE_DOC_LINE_DIFFERENTIATOR)
		return dl.getDocSyntaxToken(common.HASH_TOKEN)
	}

	return dl.getDocSyntaxToken(common.EOF_TOKEN)
}

func (dl *documentationLexer) readDocLineDifferentiatorToken() tree.STToken {
	c := dl.peek()
	switch c {
	case PLUS:
		return dl.processPlusToken()
	case HASH:
		dl.SwitchMode(PARSER_MODE_DOC_INTERNAL)
		return dl.processDeprecationLiteralToken()
	case BACKTICK:
		if dl.reader.PeekN(1) == BACKTICK {
			return dl.processDoubleOrTripleBacktickToken()
		}
		fallthrough
	default:
		dl.SwitchMode(PARSER_MODE_DOC_INTERNAL)
		return dl.readDocInternalToken()
	}
}

func (dl *documentationLexer) processPlusToken() tree.STToken {
	dl.reader.Advance() // Advance for +
	dl.SwitchMode(PARSER_MODE_DOC_PARAMETER)
	return dl.getDocSyntaxToken(common.PLUS_TOKEN)
}

func (dl *documentationLexer) processDoubleOrTripleBacktickToken() tree.STToken {
	dl.reader.AdvanceN(2) // Advance for two backticks
	if dl.peek() == BACKTICK {
		dl.reader.Advance()
		dl.SwitchMode(PARSER_MODE_DOC_TRIPLE_BACKTICK_CONTENT)
		return dl.getCodeStartBacktickToken(common.TRIPLE_BACKTICK_TOKEN)
	} else {
		dl.SwitchMode(PARSER_MODE_DOC_DOUBLE_BACKTICK_CONTENT)
		return dl.getCodeStartBacktickToken(common.DOUBLE_BACKTICK_TOKEN)
	}
}

func (dl *documentationLexer) processDeprecationLiteralToken() tree.STToken {
	lookAheadCount := 1
	lookAheadChar := dl.reader.PeekN(lookAheadCount)

	whitespaceCount := 0
	for lookAheadChar == SPACE || lookAheadChar == TAB {
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
	return dl.getDocLiteralWithoutTrivia(common.DEPRECATION_LITERAL)
}

func (dl *documentationLexer) readDocInternalToken() tree.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := dl.peek()
	if nextChar == BACKTICK {
		dl.reader.Advance()
		nextChar = dl.peek()
		if nextChar != BACKTICK {
			dl.SwitchMode(PARSER_MODE_DOC_SINGLE_BACKTICK_CONTENT)
			return dl.getDocSyntaxTokenWithoutTrivia(common.BACKTICK_TOKEN)
		}

		dl.reader.Advance()
		nextChar = dl.peek()
		if nextChar != BACKTICK {
			dl.SwitchMode(PARSER_MODE_DOC_DOUBLE_BACKTICK_CONTENT)
			return dl.getCodeStartBacktickToken(common.DOUBLE_BACKTICK_TOKEN)
		}

		dl.reader.Advance()
		dl.SwitchMode(PARSER_MODE_DOC_TRIPLE_BACKTICK_CONTENT)
		return dl.getCodeStartBacktickToken(common.TRIPLE_BACKTICK_TOKEN)
	}

	for !dl.reader.IsEOF() {
		switch nextChar {
		case NEWLINE, CARRIAGE_RETURN:
			dl.EndMode()
		case BACKTICK:
		default:
			if isIdentifierInitialChar(nextChar) {
				hasDocumentationReference := dl.processDocumentationReference(nextChar)
				if hasDocumentationReference {
					dl.SwitchMode(PARSER_MODE_DOC_REFERENCE_TYPE)
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
	return dl.getLiteral(common.DOCUMENTATION_DESCRIPTION)
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
	case TYPE, SERVICE, VARIABLE, VAR, ANNOTATION, MODULE, FUNCTION, PARAMETER, CONST:
		for {
			switch lookAheadChar {
			case SPACE, TAB:
				lookAheadCount++
				lookAheadChar = dl.reader.PeekN(lookAheadCount)
				continue
			case BACKTICK:
				if dl.reader.PeekN(lookAheadCount+1) != BACKTICK {
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

func (dl *documentationLexer) readDocParameterToken() tree.STToken {
	dl.reader.Mark()
	nextChar := dl.peek()
	if dl.isPossibleIdentifierStart(nextChar) {
		if nextChar != BACKSLASH {
			dl.reader.Advance()
		}

		dl.processIdentifierEnd()
		var token tree.STToken
		if dl.getLexeme() == RETURN {
			token = dl.getDocSyntaxToken(common.RETURN_KEYWORD)
		} else {
			token = dl.getDocLiteralToken(common.PARAMETER_NAME)
		}
		// If the parameter name is not followed by a minus token switch the mode.
		// However, if the parameter name ends with a newline DOC_PARAMETER mode is already ended.
		// Therefore, DOC_LINE_START_HASH is the active mode. In that case do not switch mode.
		if dl.peek() != MINUS && dl.context.mode != PARSER_MODE_DOC_LINE_START_HASH {
			dl.SwitchMode(PARSER_MODE_DOC_INTERNAL)
		}
		return token
	} else if nextChar == MINUS {
		dl.reader.Advance()
		dl.SwitchMode(PARSER_MODE_DOC_INTERNAL)
		return dl.getDocSyntaxToken(common.MINUS_TOKEN)
	} else {
		dl.SwitchMode(PARSER_MODE_DOC_INTERNAL)
		return dl.readDocInternalToken()
	}
}

func (dl *documentationLexer) readDocReferenceTypeToken() tree.STToken {
	dl.reader.Mark()
	nextChar := dl.peek()
	if nextChar == BACKTICK {
		dl.reader.Advance()
		dl.SwitchMode(PARSER_MODE_DOC_SINGLE_BACKTICK_CONTENT)
		return dl.getDocSyntaxTokenWithoutTrivia(common.BACKTICK_TOKEN)
	}

	for isIdentifierInitialChar(dl.peek()) {
		dl.reader.Advance()
	}
	return dl.processReferenceType()
}

func (dl *documentationLexer) processReferenceType() tree.STToken {
	tokenText := dl.getLexeme()
	switch tokenText {
	case TYPE:
		return dl.getDocSyntaxToken(common.TYPE_DOC_REFERENCE_TOKEN)
	case SERVICE:
		return dl.getDocSyntaxToken(common.SERVICE_DOC_REFERENCE_TOKEN)
	case VARIABLE:
		return dl.getDocSyntaxToken(common.VARIABLE_DOC_REFERENCE_TOKEN)
	case VAR:
		return dl.getDocSyntaxToken(common.VAR_DOC_REFERENCE_TOKEN)
	case ANNOTATION:
		return dl.getDocSyntaxToken(common.ANNOTATION_DOC_REFERENCE_TOKEN)
	case MODULE:
		return dl.getDocSyntaxToken(common.MODULE_DOC_REFERENCE_TOKEN)
	case FUNCTION:
		return dl.getDocSyntaxToken(common.FUNCTION_DOC_REFERENCE_TOKEN)
	case PARAMETER:
		return dl.getDocSyntaxToken(common.PARAMETER_DOC_REFERENCE_TOKEN)
	case CONST:
		return dl.getDocSyntaxToken(common.CONST_DOC_REFERENCE_TOKEN)
	default:
		return dl.getDocSyntaxToken(common.EOF_TOKEN)
	}
}

func (dl *documentationLexer) readSingleBacktickContentToken() tree.STToken {
	dl.reader.Mark()
	nextChar := dl.peek()
	if nextChar == BACKSLASH {
		dl.processIdentifierEnd()
		return dl.getDocIdentifierToken()
	}

	dl.reader.Advance()
	switch nextChar {
	case BACKTICK:
		dl.SwitchMode(PARSER_MODE_DOC_INTERNAL)
		return dl.getDocSyntaxTokenWithoutTrivia(common.BACKTICK_TOKEN)
	case DOT:
		return dl.getDocSyntaxToken(common.DOT_TOKEN)
	case COLON:
		return dl.getDocSyntaxToken(common.COLON_TOKEN)
	case OPEN_PARANTHESIS:
		return dl.getDocSyntaxToken(common.OPEN_PAREN_TOKEN)
	case CLOSE_PARANTHESIS:
		return dl.getDocSyntaxToken(common.CLOSE_PAREN_TOKEN)
	default:
		if dl.isPossibleIdentifierStart(nextChar) {
			dl.processIdentifierEnd()
			return dl.getDocIdentifierToken()
		}

		dl.processInvalidChars()
		return dl.getDocLiteralToken(common.CODE_CONTENT)
	}
}

func (dl *documentationLexer) processInvalidChars() {
	nextChar := dl.peek()
	for !dl.reader.IsEOF() {
		switch nextChar {
		case BACKTICK, NEWLINE, CARRIAGE_RETURN:
		default:
			dl.reader.Advance()
			nextChar = dl.peek()
			continue
		}
		break
	}
}

func (dl *documentationLexer) readCodeContent(backtickCount int) tree.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := dl.peek()
	for !dl.reader.IsEOF() {
		switch nextChar {
		case BACKTICK:
			count := dl.getBackticksCount()
			if count == backtickCount {
				dl.SwitchMode(PARSER_MODE_DOC_CODE_REF_END)
				break
			}
			dl.reader.AdvanceN(count)
			nextChar = dl.peek()
			continue
		case CARRIAGE_RETURN, NEWLINE:
			dl.previousBacktickMode = dl.context.mode
			dl.SwitchMode(PARSER_MODE_DOC_CODE_LINE_START_HASH)
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
	return dl.getLiteral(common.CODE_CONTENT)
}

func (dl *documentationLexer) getBackticksCount() int {
	count := 1
	for dl.reader.PeekN(count) == BACKTICK {
		count += 1
	}
	return count
}

func (dl *documentationLexer) readCodeReferenceEndToken() tree.STToken {
	dl.SwitchMode(PARSER_MODE_DOC_INTERNAL)
	if dl.peek() == BACKTICK {
		dl.reader.Advance()
		if dl.peek() == BACKTICK {
			dl.reader.Advance()
			if dl.peek() == BACKTICK {
				dl.reader.Advance()
				return dl.getDocSyntaxTokenWithoutTrivia(common.TRIPLE_BACKTICK_TOKEN)
			} else {
				return dl.getDocSyntaxTokenWithoutTrivia(common.DOUBLE_BACKTICK_TOKEN)
			}
		}
	}
	return dl.getDocSyntaxToken(common.EOF_TOKEN)
}

func (dl *documentationLexer) readCodeLineStartHashToken() tree.STToken {
	dl.reader.Mark()
	if dl.reader.IsEOF() {
		return dl.getDocSyntaxToken(common.EOF_TOKEN)
	}
	nextChar := dl.peek()
	if nextChar == HASH {
		dl.reader.Advance()
		return dl.getCodeLineStartHashToken()
	}
	return dl.getDocSyntaxToken(common.EOF_TOKEN)
}
