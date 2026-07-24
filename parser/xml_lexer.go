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

// Ported from XMLLexer.java.

package parser

import (
	"ballerina/parser/common"
	"ballerina/parser/tree"
	"ballerina/tools/text"
)

// xmlLexer satisfies the Lexer interface.
type xmlLexer struct {
	*lexer
}

func newXMLLexer(reader text.CharReader) *xmlLexer {
	inner := NewLexer(reader)
	inner.StartMode(PARSER_MODE_XML_CONTENT)
	return &xmlLexer{lexer: inner}
}

func (l *xmlLexer) NextToken() tree.STToken {
	var token tree.STToken
	switch l.context.mode {
	case PARSER_MODE_XML_CONTENT:
		token = l.readTokenInXMLContent()
	case PARSER_MODE_XML_ELEMENT_START_TAG:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLElement(true)
	case PARSER_MODE_XML_ELEMENT_END_TAG:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLElement(false)
	case PARSER_MODE_XML_TEXT:
		token = l.readTokenInXMLText()
	case PARSER_MODE_INTERPOLATION:
		token = l.readTokenInXMLInterpolation()
	case PARSER_MODE_XML_ATTRIBUTES:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLAttributes(true)
	case PARSER_MODE_XML_COMMENT:
		token = l.readTokenInXMLCommentOrCDATA(false)
	case PARSER_MODE_XML_PI:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLPI()
	case PARSER_MODE_XML_PI_DATA:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLPIData()
	case PARSER_MODE_XML_SINGLE_QUOTED_STRING:
		token = l.processXMLSingleQuotedString()
	case PARSER_MODE_XML_DOUBLE_QUOTED_STRING:
		token = l.processXMLDoubleQuotedString()
	case PARSER_MODE_XML_CDATA_SECTION:
		token = l.readTokenInXMLCommentOrCDATA(true)
	default:
		panic("xmlLexer.NextToken: unexpected parser mode")
	}

	if len(l.context.diagnostics) > 0 {
		token = tree.AddSyntaxDiagnostics(token, l.context.diagnostics)
		l.context.diagnostics = nil
	}
	return token
}

// XML trivia: whitespace and end-of-line only. No `//` comments.

func (l *xmlLexer) processLeadingXMLTrivia() {
	l.processXMLTrivia(&l.context.leadingTriviaList, true)
}

func (l *xmlLexer) processTrailingXMLTrivia() tree.STNode {
	triviaList := make([]tree.STNode, 0, INITIAL_TRIVIA_CAPACITY)
	l.processXMLTrivia(&triviaList, false)
	return tree.CreateNodeList(triviaList...)
}

func (l *xmlLexer) processXMLTrivia(triviaList *[]tree.STNode, isLeading bool) {
	reader := l.reader
	for !reader.IsEOF() {
		reader.Mark()
		c := reader.Peek()
		switch c {
		case SPACE, TAB, FORM_FEED:
			*triviaList = append(*triviaList, l.processWhitespaces())
		case CARRIAGE_RETURN, NEWLINE:
			*triviaList = append(*triviaList, l.processEndOfLine())
			if isLeading {
				continue
			}
			return
		default:
			return
		}
	}
}

func (l *xmlLexer) getXMLSyntaxToken(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := l.processTrailingXMLTrivia()
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLSyntaxTokenChecked(kind common.SyntaxKind, allowLeadingWS, allowTrailingWS bool) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	if !allowLeadingWS && leadingTrivia.BucketCount() != 0 {
		l.reportLexerError(common.ERROR_INVALID_WHITESPACE_BEFORE, kindStringValue(kind))
	}
	trailingTrivia := l.processTrailingXMLTrivia()
	if !allowTrailingWS && trailingTrivia.BucketCount() != 0 {
		l.reportLexerError(common.ERROR_INVALID_WHITESPACE_AFTER, kindStringValue(kind))
	}
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLSyntaxTokenWithoutTrailingWS(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := tree.CreateEmptyNodeList()
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLLiteralValueToken(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	trailingTrivia := l.processTrailingXMLTrivia()
	return tree.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLText(kind common.SyntaxKind) tree.STToken {
	return l.getXMLLiteralValueToken(kind)
}

func (l *xmlLexer) getXMLNameToken(allowLeadingWS bool) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	if !allowLeadingWS && leadingTrivia.BucketCount() != 0 {
		l.reportLexerError(common.ERROR_INVALID_WHITESPACE_BEFORE, lexeme)
	}
	trailingTrivia := l.processTrailingXMLTrivia()
	return tree.CreateIdentifierToken(lexeme, leadingTrivia, trailingTrivia)
}

// INTERPOLATION mode

func (l *xmlLexer) readTokenInXMLInterpolation() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}
	if reader.Peek() == CLOSE_BRACE {
		l.EndMode()
		reader.Advance()
		return l.getXMLSyntaxTokenWithoutTrailingWS(common.CLOSE_BRACE_TOKEN)
	}
	// Interpolation body should be empty (already substituted to `${}`). Fall back.
	l.EndMode()
	return l.NextToken()
}

// XML_CONTENT mode

func (l *xmlLexer) readTokenInXMLContent() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case BACKTICK:
		l.EndMode()
		return l.NextToken()
	case LT:
		reader.Advance()
		nextChar = reader.Peek()
		switch nextChar {
		case EXCLAMATION_MARK:
			if reader.PeekN(1) == MINUS && reader.PeekN(2) == MINUS {
				reader.AdvanceN(3)
				l.StartMode(PARSER_MODE_XML_COMMENT)
				return l.getXMLSyntaxTokenWithoutTrailingWS(common.XML_COMMENT_START_TOKEN)
			}
			if l.isCDATAStart() {
				reader.AdvanceN(8)
				l.StartMode(PARSER_MODE_XML_CDATA_SECTION)
				return l.getXMLSyntaxTokenWithoutTrailingWS(common.XML_CDATA_START_TOKEN)
			}
		case QUESTION_MARK:
			reader.Advance()
			l.StartMode(PARSER_MODE_XML_PI)
			return l.getXMLSyntaxTokenWithoutTrailingWS(common.XML_PI_START_TOKEN)
		case SLASH:
			l.StartMode(PARSER_MODE_XML_ELEMENT_END_TAG)
			return l.getXMLSyntaxTokenChecked(common.LT_TOKEN, false, false)
		}
		l.StartMode(PARSER_MODE_XML_ELEMENT_START_TAG)
		return l.getXMLSyntaxTokenChecked(common.LT_TOKEN, false, false)
	case DOLLAR:
		if reader.PeekN(1) == OPEN_BRACE {
			l.StartMode(PARSER_MODE_INTERPOLATION)
			reader.AdvanceN(2)
			return l.getXMLSyntaxToken(common.INTERPOLATION_START_TOKEN)
		}
	}

	l.StartMode(PARSER_MODE_XML_TEXT)
	return l.readTokenInXMLText()
}

func (l *xmlLexer) isCDATAStart() bool {
	r := l.reader
	return r.PeekN(1) == OPEN_BRACKET &&
		r.PeekN(2) == 'C' &&
		r.PeekN(3) == 'D' &&
		r.PeekN(4) == 'A' &&
		r.PeekN(5) == 'T' &&
		r.PeekN(6) == 'A' &&
		r.PeekN(7) == OPEN_BRACKET
}

// XML_ELEMENT modes

func (l *xmlLexer) readTokenInXMLElement(isStartTag bool) tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

	c := reader.Peek()
	switch c {
	case LT:
		if isStartTag {
			l.StartMode(PARSER_MODE_XML_CONTENT)
		} else {
			l.EndMode()
		}
		return l.NextToken()
	case GT:
		l.EndMode()
		if isStartTag {
			l.StartMode(PARSER_MODE_XML_CONTENT)
		}
		reader.Advance()
		return l.getXMLSyntaxTokenWithoutTrailingWS(common.GT_TOKEN)
	case SLASH:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(common.SLASH_TOKEN, isStartTag, false)
	case COLON:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(common.COLON_TOKEN, false, false)
	case DOLLAR:
		if reader.PeekN(1) == OPEN_BRACE {
			reader.AdvanceN(2)
			l.StartMode(PARSER_MODE_INTERPOLATION)
			return l.getXMLSyntaxToken(common.INTERPOLATION_START_TOKEN)
		}
	case BACKTICK:
		l.EndMode()
		return l.NextToken()
	}

	reader.Advance()
	tagName := l.processXMLName(c, false)
	l.StartMode(PARSER_MODE_XML_ATTRIBUTES)
	return tagName
}

func (l *xmlLexer) processXMLName(startChar rune, allowLeadingWS bool) tree.STToken {
	reader := l.reader
	isValid := isXMLNCNameStart(startChar)
	for !reader.IsEOF() && isXMLNCName(reader.Peek()) {
		reader.Advance()
	}
	if !isValid {
		l.reportLexerError(common.ERROR_INVALID_XML_NAME, l.getLexeme())
	}
	return l.getXMLNameToken(allowLeadingWS)
}

// XML_ATTRIBUTES mode

func (l *xmlLexer) readTokenInXMLAttributes(isStartTag bool) tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case LT, GT, SLASH, BACKTICK:
		l.EndMode()
		return l.readTokenInXMLElement(isStartTag)
	case COLON:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(common.COLON_TOKEN, false, false)
	case DOLLAR:
		if reader.PeekN(1) == OPEN_BRACE {
			reader.AdvanceN(2)
			l.StartMode(PARSER_MODE_INTERPOLATION)
			return l.getXMLSyntaxToken(common.INTERPOLATION_START_TOKEN)
		}
	case EQUAL:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(common.EQUAL_TOKEN, true, true)
	case DOUBLE_QUOTE:
		reader.Advance()
		l.StartMode(PARSER_MODE_XML_DOUBLE_QUOTED_STRING)
		return l.getXMLSyntaxTokenChecked(common.DOUBLE_QUOTE_TOKEN, false, false)
	case SINGLE_QUOTE:
		reader.Advance()
		l.StartMode(PARSER_MODE_XML_SINGLE_QUOTED_STRING)
		return l.getXMLSyntaxTokenChecked(common.SINGLE_QUOTE_TOKEN, false, false)
	}

	reader.Advance()
	return l.processXMLName(nextChar, true)
}

// XML quoted string modes

func (l *xmlLexer) processXMLDoubleQuotedString() tree.STToken {
	return l.processXMLQuotedString(DOUBLE_QUOTE, common.DOUBLE_QUOTE_TOKEN)
}

func (l *xmlLexer) processXMLSingleQuotedString() tree.STToken {
	return l.processXMLQuotedString(SINGLE_QUOTE, common.SINGLE_QUOTE_TOKEN)
}

func (l *xmlLexer) processXMLQuotedString(startingQuote rune, startQuoteKind common.SyntaxKind) tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case DOUBLE_QUOTE, SINGLE_QUOTE:
		if nextChar == startingQuote {
			reader.Advance()
			l.EndMode()
			return l.getXMLSyntaxTokenChecked(startQuoteKind, false, true)
		}
	case DOLLAR:
		if reader.PeekN(1) == OPEN_BRACE {
			reader.AdvanceN(2)
			l.StartMode(PARSER_MODE_INTERPOLATION)
			return l.getXMLSyntaxToken(common.INTERPOLATION_START_TOKEN)
		}
	}

scan:
	for !reader.IsEOF() {
		nextChar = reader.Peek()
		switch nextChar {
		case DOUBLE_QUOTE, SINGLE_QUOTE:
			if nextChar == startingQuote {
				break scan
			}
			reader.Advance()
			continue
		case BITWISE_AND:
			l.processXMLReferenceInQuotedString(startingQuote)
			continue
		case LT:
			reader.Advance()
			l.reportLexerError(common.ERROR_INVALID_CHARACTER_IN_XML_ATTRIBUTE_VALUE, string(LT))
			continue
		case DOLLAR:
			if reader.PeekN(1) == OPEN_BRACE {
				break scan
			}
			reader.Advance()
			continue
		default:
			reader.Advance()
		}
	}

	return l.getXMLText(common.XML_TEXT_CONTENT)
}

func (l *xmlLexer) processXMLReferenceInQuotedString(startingQuote rune) {
	nextChar := l.reader.Peek()
	switch nextChar {
	case DOUBLE_QUOTE, SINGLE_QUOTE:
		if nextChar == startingQuote {
			return
		}
	}
	l.processXMLReference()
}

func (l *xmlLexer) processXMLReference() {
	reader := l.reader
	reader.Advance()
	nextChar := reader.Peek()
	switch nextChar {
	case SEMICOLON:
		l.reportLexerError(common.ERROR_MISSING_ENTITY_REFERENCE_NAME)
		reader.Advance()
		return
	case HASH:
		l.processXMLCharRef()
	default:
		l.processXMLEntityRef()
	}
	if reader.Peek() == SEMICOLON {
		reader.Advance()
	} else {
		l.reportLexerError(common.ERROR_MISSING_SEMICOLON_IN_XML_REFERENCE)
	}
}

func (l *xmlLexer) processXMLCharRef() {
	reader := l.reader
	reader.Advance()
	if reader.Peek() == 'x' {
		reader.Advance()
		for isHexDigit(byte(reader.Peek())) {
			reader.Advance()
		}
	} else {
		for isDigit(byte(reader.Peek())) {
			reader.Advance()
		}
	}
}

func (l *xmlLexer) processXMLEntityRef() {
	reader := l.reader
	if !isXMLNCNameStart(reader.Peek()) {
		l.reportLexerError(common.ERROR_INVALID_ENTITY_REFERENCE_NAME_START)
	} else {
		reader.Advance()
	}
	for !reader.IsEOF() && isXMLNCName(reader.Peek()) {
		reader.Advance()
	}
}

// XML_TEXT mode

func (l *xmlLexer) readTokenInXMLText() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

scan:
	for !reader.IsEOF() {
		nextChar := reader.Peek()
		switch nextChar {
		case LT:
			break scan
		case DOLLAR:
			if reader.PeekN(1) == OPEN_BRACE {
				break scan
			}
			reader.Advance()
			continue
		case BITWISE_AND:
			l.processXMLReference()
			continue
		case BACKTICK:
			break scan
		default:
			reader.Advance()
		}
	}

	l.EndMode()
	return l.getXMLText(common.XML_TEXT_CONTENT)
}

// XML_COMMENT and XML_CDATA_SECTION modes

func (l *xmlLexer) readTokenInXMLCommentOrCDATA(isCdata bool) tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

	switch reader.Peek() {
	case MINUS:
		if !isCdata && reader.PeekN(1) == MINUS {
			if reader.PeekN(2) == GT {
				reader.AdvanceN(3)
				l.EndMode()
				return l.getXMLSyntaxTokenWithoutTrailingWS(common.XML_COMMENT_END_TOKEN)
			}
			reader.Advance()
			l.reportLexerError(common.ERROR_DOUBLE_HYPHEN_NOT_ALLOWED_WITHIN_XML_COMMENT)
		}
	case DOLLAR:
		if reader.PeekN(1) == OPEN_BRACE {
			reader.AdvanceN(2)
			l.StartMode(PARSER_MODE_INTERPOLATION)
			return l.getXMLSyntaxToken(common.INTERPOLATION_START_TOKEN)
		}
	case CLOSE_BRACKET:
		if isCdata && reader.PeekN(1) == CLOSE_BRACKET && reader.PeekN(2) == GT {
			reader.AdvanceN(3)
			l.EndMode()
			return l.getXMLSyntaxTokenWithoutTrailingWS(common.XML_CDATA_END_TOKEN)
		}
	}

scan:
	for !reader.IsEOF() {
		switch reader.Peek() {
		case MINUS:
			if !isCdata && reader.PeekN(1) == MINUS {
				if reader.PeekN(2) == GT {
					break scan
				}
				reader.AdvanceN(2)
				l.reportLexerError(common.ERROR_DOUBLE_HYPHEN_NOT_ALLOWED_WITHIN_XML_COMMENT)
			} else {
				reader.Advance()
			}
		case DOLLAR:
			if reader.PeekN(1) == OPEN_BRACE {
				break scan
			}
			reader.Advance()
		case BACKTICK:
			l.EndMode()
			break scan
		case CLOSE_BRACKET:
			if isCdata && reader.PeekN(1) == CLOSE_BRACKET && reader.PeekN(2) == GT {
				break scan
			}
			reader.Advance()
		default:
			reader.Advance()
		}
	}

	return l.getXMLText(common.XML_TEXT_CONTENT)
}

// XML_PI mode

func (l *xmlLexer) readTokenInXMLPI() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case QUESTION_MARK:
		if reader.PeekN(1) == GT {
			reader.AdvanceN(2)
			l.EndMode()
			return l.getXMLSyntaxToken(common.XML_PI_END_TOKEN)
		}
	case BACKTICK:
		l.EndMode()
		return l.NextToken()
	}

	reader.Advance()
	tagName := l.processXMLName(nextChar, false)
	l.StartMode(PARSER_MODE_XML_PI_DATA)
	return tagName
}

// XML_PI_DATA mode

func (l *xmlLexer) readTokenInXMLPIData() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(common.EOF_TOKEN)
	}

	switch reader.Peek() {
	case DOLLAR:
		if reader.PeekN(1) == OPEN_BRACE {
			reader.AdvanceN(2)
			l.StartMode(PARSER_MODE_INTERPOLATION)
			return l.getXMLSyntaxToken(common.INTERPOLATION_START_TOKEN)
		}
	case QUESTION_MARK:
		if reader.PeekN(1) == GT {
			reader.AdvanceN(2)
			l.EndMode()
			l.EndMode()
			return l.getXMLSyntaxToken(common.XML_PI_END_TOKEN)
		}
	}

scan:
	for !reader.IsEOF() {
		switch reader.Peek() {
		case QUESTION_MARK:
			if reader.PeekN(1) == GT {
				break scan
			}
			reader.Advance()
		case DOLLAR:
			if reader.PeekN(1) == OPEN_BRACE {
				break scan
			}
			reader.Advance()
		case BACKTICK:
			l.EndMode()
			break scan
		default:
			reader.Advance()
		}
	}

	return l.getXMLText(common.XML_TEXT_CONTENT)
}

// kindStringValue maps a SyntaxKind to its source-text representation for diagnostic messages.
// Mirrors Java SyntaxKind.stringValue() for the small set of kinds used in XML diagnostic args.
func kindStringValue(kind common.SyntaxKind) string {
	switch kind {
	case common.LT_TOKEN:
		return "<"
	case common.GT_TOKEN:
		return ">"
	case common.SLASH_TOKEN:
		return "/"
	case common.COLON_TOKEN:
		return ":"
	case common.EQUAL_TOKEN:
		return "="
	case common.DOUBLE_QUOTE_TOKEN:
		return "\""
	case common.SINGLE_QUOTE_TOKEN:
		return "'"
	default:
		return ""
	}
}
