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
	"github.com/ballerina-nutcracker/ballerina/parser/common"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

// xmlLexer satisfies the Lexer interface.
type xmlLexer struct {
	*lexer
}

func newXMLLexer(reader text.CharReader) *xmlLexer {
	inner := newLexer(reader)
	inner.StartMode(parserModeXmlContent)
	return &xmlLexer{lexer: inner}
}

func (l *xmlLexer) NextToken() st.STToken {
	var token st.STToken
	switch l.context.mode {
	case parserModeXmlContent:
		token = l.readTokenInXMLContent()
	case parserModeXmlElementStartTag:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLElement(true)
	case parserModeXmlElementEndTag:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLElement(false)
	case parserModeXmlText:
		token = l.readTokenInXMLText()
	case parserModeInterpolation:
		token = l.readTokenInXMLInterpolation()
	case parserModeXmlAttributes:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLAttributes(true)
	case parserModeXmlComment:
		token = l.readTokenInXMLCommentOrCDATA(false)
	case parserModeXmlPi:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLPI()
	case parserModeXmlPiData:
		l.processLeadingXMLTrivia()
		token = l.readTokenInXMLPIData()
	case parserModeXmlSingleQuotedString:
		token = l.processXMLSingleQuotedString()
	case parserModeXmlDoubleQuotedString:
		token = l.processXMLDoubleQuotedString()
	case parserModeXmlCdataSection:
		token = l.readTokenInXMLCommentOrCDATA(true)
	default:
		panic("xmlLexer.NextToken: unexpected parser mode")
	}

	if len(l.context.diagnostics) > 0 {
		token = st.AddSyntaxDiagnostics(token, l.context.diagnostics)
		l.context.diagnostics = nil
	}
	return token
}

// XML trivia: whitespace and end-of-line only. No `//` comments.

func (l *xmlLexer) processLeadingXMLTrivia() {
	l.processXMLTrivia(&l.context.leadingTriviaList, true)
}

func (l *xmlLexer) processTrailingXMLTrivia() st.STNode {
	triviaList := make([]st.STNode, 0, initialTriviaCapacity)
	l.processXMLTrivia(&triviaList, false)
	return st.CreateNodeList(triviaList...)
}

func (l *xmlLexer) processXMLTrivia(triviaList *[]st.STNode, isLeading bool) {
	reader := l.reader
	for !reader.IsEOF() {
		reader.Mark()
		c := reader.Peek()
		switch c {
		case space, tab, formFeed:
			*triviaList = append(*triviaList, l.processWhitespaces())
		case carriageReturn, newline:
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

func (l *xmlLexer) getXMLSyntaxToken(kind st.SyntaxKind) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := l.processTrailingXMLTrivia()
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLSyntaxTokenChecked(kind st.SyntaxKind, allowLeadingWS, allowTrailingWS bool) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	if !allowLeadingWS && leadingTrivia.BucketCount() != 0 {
		l.reportLexerError(common.ERROR_INVALID_WHITESPACE_BEFORE, kindStringValue(kind))
	}
	trailingTrivia := l.processTrailingXMLTrivia()
	if !allowTrailingWS && trailingTrivia.BucketCount() != 0 {
		l.reportLexerError(common.ERROR_INVALID_WHITESPACE_AFTER, kindStringValue(kind))
	}
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLSyntaxTokenWithoutTrailingWS(kind st.SyntaxKind) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := st.CreateEmptyNodeList()
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLLiteralValueToken(kind st.SyntaxKind) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	trailingTrivia := l.processTrailingXMLTrivia()
	return st.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (l *xmlLexer) getXMLText(kind st.SyntaxKind) st.STToken {
	return l.getXMLLiteralValueToken(kind)
}

func (l *xmlLexer) getXMLNameToken(allowLeadingWS bool) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	if !allowLeadingWS && leadingTrivia.BucketCount() != 0 {
		l.reportLexerError(common.ERROR_INVALID_WHITESPACE_BEFORE, lexeme)
	}
	trailingTrivia := l.processTrailingXMLTrivia()
	return st.CreateIdentifierToken(lexeme, leadingTrivia, trailingTrivia)
}

// INTERPOLATION mode

func (l *xmlLexer) readTokenInXMLInterpolation() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}
	if reader.Peek() == closeBrace {
		l.EndMode()
		reader.Advance()
		return l.getXMLSyntaxTokenWithoutTrailingWS(st.CLOSE_BRACE_TOKEN)
	}
	// Interpolation body should be empty (already substituted to `${}`). Fall back.
	l.EndMode()
	return l.NextToken()
}

// XML_CONTENT mode

func (l *xmlLexer) readTokenInXMLContent() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case backtick:
		l.EndMode()
		return l.NextToken()
	case lt:
		reader.Advance()
		nextChar = reader.Peek()
		switch nextChar {
		case exclamationMark:
			if reader.PeekN(1) == minus && reader.PeekN(2) == minus {
				reader.AdvanceN(3)
				l.StartMode(parserModeXmlComment)
				return l.getXMLSyntaxTokenWithoutTrailingWS(st.XML_COMMENT_START_TOKEN)
			}
			if l.isCDATAStart() {
				reader.AdvanceN(8)
				l.StartMode(parserModeXmlCdataSection)
				return l.getXMLSyntaxTokenWithoutTrailingWS(st.XML_CDATA_START_TOKEN)
			}
		case questionMark:
			reader.Advance()
			l.StartMode(parserModeXmlPi)
			return l.getXMLSyntaxTokenWithoutTrailingWS(st.XML_PI_START_TOKEN)
		case slash:
			l.StartMode(parserModeXmlElementEndTag)
			return l.getXMLSyntaxTokenChecked(st.LT_TOKEN, false, false)
		}
		l.StartMode(parserModeXmlElementStartTag)
		return l.getXMLSyntaxTokenChecked(st.LT_TOKEN, false, false)
	case dollar:
		if reader.PeekN(1) == openBrace {
			l.StartMode(parserModeInterpolation)
			reader.AdvanceN(2)
			return l.getXMLSyntaxToken(st.INTERPOLATION_START_TOKEN)
		}
	}

	l.StartMode(parserModeXmlText)
	return l.readTokenInXMLText()
}

func (l *xmlLexer) isCDATAStart() bool {
	r := l.reader
	return r.PeekN(1) == openBracket &&
		r.PeekN(2) == 'C' &&
		r.PeekN(3) == 'D' &&
		r.PeekN(4) == 'A' &&
		r.PeekN(5) == 'T' &&
		r.PeekN(6) == 'A' &&
		r.PeekN(7) == openBracket
}

// XML_ELEMENT modes

func (l *xmlLexer) readTokenInXMLElement(isStartTag bool) st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

	c := reader.Peek()
	switch c {
	case lt:
		if isStartTag {
			l.StartMode(parserModeXmlContent)
		} else {
			l.EndMode()
		}
		return l.NextToken()
	case gt:
		l.EndMode()
		if isStartTag {
			l.StartMode(parserModeXmlContent)
		}
		reader.Advance()
		return l.getXMLSyntaxTokenWithoutTrailingWS(st.GT_TOKEN)
	case slash:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(st.SLASH_TOKEN, isStartTag, false)
	case colon:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(st.COLON_TOKEN, false, false)
	case dollar:
		if reader.PeekN(1) == openBrace {
			reader.AdvanceN(2)
			l.StartMode(parserModeInterpolation)
			return l.getXMLSyntaxToken(st.INTERPOLATION_START_TOKEN)
		}
	case backtick:
		l.EndMode()
		return l.NextToken()
	}

	reader.Advance()
	tagName := l.processXMLName(c, false)
	l.StartMode(parserModeXmlAttributes)
	return tagName
}

func (l *xmlLexer) processXMLName(startChar rune, allowLeadingWS bool) st.STToken {
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

func (l *xmlLexer) readTokenInXMLAttributes(isStartTag bool) st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case lt, gt, slash, backtick:
		l.EndMode()
		return l.readTokenInXMLElement(isStartTag)
	case colon:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(st.COLON_TOKEN, false, false)
	case dollar:
		if reader.PeekN(1) == openBrace {
			reader.AdvanceN(2)
			l.StartMode(parserModeInterpolation)
			return l.getXMLSyntaxToken(st.INTERPOLATION_START_TOKEN)
		}
	case equal:
		reader.Advance()
		return l.getXMLSyntaxTokenChecked(st.EQUAL_TOKEN, true, true)
	case doubleQuote:
		reader.Advance()
		l.StartMode(parserModeXmlDoubleQuotedString)
		return l.getXMLSyntaxTokenChecked(st.DOUBLE_QUOTE_TOKEN, false, false)
	case singleQuote:
		reader.Advance()
		l.StartMode(parserModeXmlSingleQuotedString)
		return l.getXMLSyntaxTokenChecked(st.SINGLE_QUOTE_TOKEN, false, false)
	}

	reader.Advance()
	return l.processXMLName(nextChar, true)
}

// XML quoted string modes

func (l *xmlLexer) processXMLDoubleQuotedString() st.STToken {
	return l.processXMLQuotedString(doubleQuote, st.DOUBLE_QUOTE_TOKEN)
}

func (l *xmlLexer) processXMLSingleQuotedString() st.STToken {
	return l.processXMLQuotedString(singleQuote, st.SINGLE_QUOTE_TOKEN)
}

func (l *xmlLexer) processXMLQuotedString(startingQuote rune, startQuoteKind st.SyntaxKind) st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case doubleQuote, singleQuote:
		if nextChar == startingQuote {
			reader.Advance()
			l.EndMode()
			return l.getXMLSyntaxTokenChecked(startQuoteKind, false, true)
		}
	case dollar:
		if reader.PeekN(1) == openBrace {
			reader.AdvanceN(2)
			l.StartMode(parserModeInterpolation)
			return l.getXMLSyntaxToken(st.INTERPOLATION_START_TOKEN)
		}
	}

scan:
	for !reader.IsEOF() {
		nextChar = reader.Peek()
		switch nextChar {
		case doubleQuote, singleQuote:
			if nextChar == startingQuote {
				break scan
			}
			reader.Advance()
			continue
		case bitwiseAnd:
			l.processXMLReferenceInQuotedString(startingQuote)
			continue
		case lt:
			reader.Advance()
			l.reportLexerError(common.ERROR_INVALID_CHARACTER_IN_XML_ATTRIBUTE_VALUE, string(lt))
			continue
		case dollar:
			if reader.PeekN(1) == openBrace {
				break scan
			}
			reader.Advance()
			continue
		default:
			reader.Advance()
		}
	}

	return l.getXMLText(st.XML_TEXT_CONTENT)
}

func (l *xmlLexer) processXMLReferenceInQuotedString(startingQuote rune) {
	nextChar := l.reader.Peek()
	switch nextChar {
	case doubleQuote, singleQuote:
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
	case semicolon:
		l.reportLexerError(common.ERROR_MISSING_ENTITY_REFERENCE_NAME)
		reader.Advance()
		return
	case hash:
		l.processXMLCharRef()
	default:
		l.processXMLEntityRef()
	}
	if reader.Peek() == semicolon {
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

func (l *xmlLexer) readTokenInXMLText() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

scan:
	for !reader.IsEOF() {
		nextChar := reader.Peek()
		switch nextChar {
		case lt:
			break scan
		case dollar:
			if reader.PeekN(1) == openBrace {
				break scan
			}
			reader.Advance()
			continue
		case bitwiseAnd:
			l.processXMLReference()
			continue
		case backtick:
			break scan
		default:
			reader.Advance()
		}
	}

	l.EndMode()
	return l.getXMLText(st.XML_TEXT_CONTENT)
}

// XML_COMMENT and XML_CDATA_SECTION modes

func (l *xmlLexer) readTokenInXMLCommentOrCDATA(isCdata bool) st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

	switch reader.Peek() {
	case minus:
		if !isCdata && reader.PeekN(1) == minus {
			if reader.PeekN(2) == gt {
				reader.AdvanceN(3)
				l.EndMode()
				return l.getXMLSyntaxTokenWithoutTrailingWS(st.XML_COMMENT_END_TOKEN)
			}
			reader.Advance()
			l.reportLexerError(common.ERROR_DOUBLE_HYPHEN_NOT_ALLOWED_WITHIN_XML_COMMENT)
		}
	case dollar:
		if reader.PeekN(1) == openBrace {
			reader.AdvanceN(2)
			l.StartMode(parserModeInterpolation)
			return l.getXMLSyntaxToken(st.INTERPOLATION_START_TOKEN)
		}
	case closeBracket:
		if isCdata && reader.PeekN(1) == closeBracket && reader.PeekN(2) == gt {
			reader.AdvanceN(3)
			l.EndMode()
			return l.getXMLSyntaxTokenWithoutTrailingWS(st.XML_CDATA_END_TOKEN)
		}
	}

scan:
	for !reader.IsEOF() {
		switch reader.Peek() {
		case minus:
			if !isCdata && reader.PeekN(1) == minus {
				if reader.PeekN(2) == gt {
					break scan
				}
				reader.AdvanceN(2)
				l.reportLexerError(common.ERROR_DOUBLE_HYPHEN_NOT_ALLOWED_WITHIN_XML_COMMENT)
			} else {
				reader.Advance()
			}
		case dollar:
			if reader.PeekN(1) == openBrace {
				break scan
			}
			reader.Advance()
		case backtick:
			l.EndMode()
			break scan
		case closeBracket:
			if isCdata && reader.PeekN(1) == closeBracket && reader.PeekN(2) == gt {
				break scan
			}
			reader.Advance()
		default:
			reader.Advance()
		}
	}

	return l.getXMLText(st.XML_TEXT_CONTENT)
}

// XML_PI mode

func (l *xmlLexer) readTokenInXMLPI() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case questionMark:
		if reader.PeekN(1) == gt {
			reader.AdvanceN(2)
			l.EndMode()
			return l.getXMLSyntaxToken(st.XML_PI_END_TOKEN)
		}
	case backtick:
		l.EndMode()
		return l.NextToken()
	}

	reader.Advance()
	tagName := l.processXMLName(nextChar, false)
	l.StartMode(parserModeXmlPiData)
	return tagName
}

// XML_PI_DATA mode

func (l *xmlLexer) readTokenInXMLPIData() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getXMLSyntaxToken(st.EOF_TOKEN)
	}

	switch reader.Peek() {
	case dollar:
		if reader.PeekN(1) == openBrace {
			reader.AdvanceN(2)
			l.StartMode(parserModeInterpolation)
			return l.getXMLSyntaxToken(st.INTERPOLATION_START_TOKEN)
		}
	case questionMark:
		if reader.PeekN(1) == gt {
			reader.AdvanceN(2)
			l.EndMode()
			l.EndMode()
			return l.getXMLSyntaxToken(st.XML_PI_END_TOKEN)
		}
	}

scan:
	for !reader.IsEOF() {
		switch reader.Peek() {
		case questionMark:
			if reader.PeekN(1) == gt {
				break scan
			}
			reader.Advance()
		case dollar:
			if reader.PeekN(1) == openBrace {
				break scan
			}
			reader.Advance()
		case backtick:
			l.EndMode()
			break scan
		default:
			reader.Advance()
		}
	}

	return l.getXMLText(st.XML_TEXT_CONTENT)
}

// kindStringValue maps a SyntaxKind to its source-text representation for diagnostic messages.
// Mirrors Java SyntaxKind.stringValue() for the small set of kinds used in XML diagnostic args.
func kindStringValue(kind st.SyntaxKind) string {
	switch kind {
	case st.LT_TOKEN:
		return "<"
	case st.GT_TOKEN:
		return ">"
	case st.SLASH_TOKEN:
		return "/"
	case st.COLON_TOKEN:
		return ":"
	case st.EQUAL_TOKEN:
		return "="
	case st.DOUBLE_QUOTE_TOKEN:
		return "\""
	case st.SINGLE_QUOTE_TOKEN:
		return "'"
	default:
		return ""
	}
}
