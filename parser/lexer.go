// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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
	"unicode"

	debugcommon "ballerina/common"
	"ballerina/parser/common"
	"ballerina/parser/tree"
	"ballerina/tools/text"
)

// TODO: we have lot of unbounded lookaheads which are implemented by incrementing a lookahead count and repeatedly
// calling

// FIXME: get rid of repeated l.reader references in ai code

const INITIAL_TRIVIA_CAPACITY = 1

type Lexer interface {
	NextToken() tree.STToken
	StartMode(mode ParserMode)
	SwitchMode(mode ParserMode)
	EndMode()
	GetCurrentMode() ParserMode
}

// TODO: introduce diagnostic context with flags and a channel
type lexer struct {
	reader  text.CharReader
	context LexerContext
}

type LexerContext struct {
	mode              ParserMode
	modeStack         []ParserMode
	leadingTriviaList []tree.STNode
	diagnostics       []tree.STNodeDiagnostic
}

func NewLexer(reader text.CharReader) *lexer {
	return &lexer{
		reader:  reader,
		context: LexerContext{},
	}
}

func (l *lexer) StartMode(mode ParserMode) {
	l.context.mode = mode
	l.context.modeStack = append(l.context.modeStack, mode)
}

func (l *lexer) SwitchMode(mode ParserMode) {
	l.context.modeStack = l.context.modeStack[:len(l.context.modeStack)-1]
	l.context.mode = mode
	l.context.modeStack = append(l.context.modeStack, mode)
}

func (l *lexer) EndMode() {
	if len(l.context.modeStack) == 0 {
		panic("cannot end mode: mode stack is empty")
	}
	l.context.modeStack = l.context.modeStack[:len(l.context.modeStack)-1]
	if len(l.context.modeStack) == 0 {
		l.context.mode = PARSER_MODE_DEFAULT_MODE
	} else {
		l.context.mode = l.context.modeStack[len(l.context.modeStack)-1]
	}
}

func (l *lexer) GetCurrentMode() ParserMode {
	return l.context.mode
}

func (l *lexer) NextToken() tree.STToken {
	var token tree.STToken
	switch l.context.mode {
	case PARSER_MODE_TEMPLATE:
		token = l.readTemplateToken()
	case PARSER_MODE_PROMPT:
		token = l.readPromptToken()
	case PARSER_MODE_REGEXP:
		token = l.readRegExpTemplateToken()
	case PARSER_MODE_INTERPOLATION:
		l.processLeadingTrivia()
		token = l.readTokenInInterpolation()
	case PARSER_MODE_INTERPOLATION_BRACED_CONTENT:
		l.processLeadingTrivia()
		token = l.readTokenInBracedContentInInterpolation()
	default:
		l.processLeadingTrivia()
		token = l.readToken()
	}
	if len(l.context.diagnostics) > 0 {
		token = tree.AddSyntaxDiagnostics(token, l.context.diagnostics)
		l.context.diagnostics = nil
	}
	debugcommon.DebugWriteLazy(debugcommon.DUMP_TOKENS, func() string { return tree.ToSexpr(token) })
	return token
}

func (l *lexer) readToken() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(common.EOF_TOKEN)
	}
	c := reader.Peek()
	if c == BACKSLASH {
		l.processUnquotedIdentifier()
		return l.getIdentifierToken()
	}
	reader.Advance()
	var token tree.STToken
	switch c {
	case COLON:
		token = l.getSyntaxToken(common.COLON_TOKEN)
	case SEMICOLON:
		token = l.getSyntaxToken(common.SEMICOLON_TOKEN)
	case DOT:
		token = l.processDot()
	case COMMA:
		token = l.getSyntaxToken(common.COMMA_TOKEN)
	case OPEN_PARANTHESIS:
		token = l.getSyntaxToken(common.OPEN_PAREN_TOKEN)
	case CLOSE_PARANTHESIS:
		token = l.getSyntaxToken(common.CLOSE_PAREN_TOKEN)
	case OPEN_BRACE:
		if reader.Peek() == PIPE {
			reader.Advance()
			token = l.getSyntaxToken(common.OPEN_BRACE_PIPE_TOKEN)
		} else {
			token = l.getSyntaxToken(common.OPEN_BRACE_TOKEN)
		}
	case CLOSE_BRACE:
		token = l.getSyntaxToken(common.CLOSE_BRACE_TOKEN)
	case OPEN_BRACKET:
		token = l.getSyntaxToken(common.OPEN_BRACKET_TOKEN)
	case CLOSE_BRACKET:
		token = l.getSyntaxToken(common.CLOSE_BRACKET_TOKEN)
	case PIPE:
		token = l.processPipeOperator()
	case QUESTION_MARK:
		if reader.Peek() == DOT && reader.PeekN(1) != DOT {
			reader.Advance()
			token = l.getSyntaxToken(common.OPTIONAL_CHAINING_TOKEN)
		} else if reader.Peek() == COLON {
			reader.Advance()
			token = l.getSyntaxToken(common.ELVIS_TOKEN)
		} else {
			token = l.getSyntaxToken(common.QUESTION_MARK_TOKEN)
		}
	case DOUBLE_QUOTE:
		token = l.processStringLiteral()
	case HASH:
		token = l.processDocumentationString()
	case AT:
		token = l.getSyntaxToken(common.AT_TOKEN)
	case EQUAL:
		token = l.processEqualOperator()
	case PLUS:
		token = l.getSyntaxToken(common.PLUS_TOKEN)
	case MINUS:
		if reader.Peek() == GT {
			reader.Advance()
			if reader.Peek() == GT {
				reader.Advance()
				token = l.getSyntaxToken(common.SYNC_SEND_TOKEN)
			} else {
				token = l.getSyntaxToken(common.RIGHT_ARROW_TOKEN)
			}
		} else {
			token = l.getSyntaxToken(common.MINUS_TOKEN)
		}
	case ASTERISK:
		token = l.getSyntaxToken(common.ASTERISK_TOKEN)
	case SLASH:
		token = l.processSlashToken()
	case PERCENT:
		token = l.getSyntaxToken(common.PERCENT_TOKEN)
	case LT:
		token = l.processTokenStartWithLt()
	case GT:
		token = l.processTokenStartWithGt()
	case EXCLAMATION_MARK:
		token = l.processExclamationMarkOperator()
	case BITWISE_AND:
		if reader.Peek() == BITWISE_AND {
			reader.Advance()
			token = l.getSyntaxToken(common.LOGICAL_AND_TOKEN)
		} else {
			token = l.getSyntaxToken(common.BITWISE_AND_TOKEN)
		}
	case BITWISE_XOR:
		token = l.getSyntaxToken(common.BITWISE_XOR_TOKEN)
	case NEGATION:
		token = l.getSyntaxToken(common.NEGATION_TOKEN)
	case BACKTICK:
		l.StartMode(PARSER_MODE_TEMPLATE)
		token = l.getBacktickToken()
	case SINGLE_QUOTE:
		token = l.processQuotedIdentifier()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		token = l.processNumericLiteral(c)
	default:
		if isIdentifierInitialChar(c) {
			token = l.processIdentifierOrKeyword()
		} else {
			invalidToken := l.processInvalidToken()
			token = l.NextToken()
			f := tree.CreateDiagnostic(&common.ERROR_INVALID_TOKEN, invalidToken)
			token = tree.AddSyntaxDiagnostic(token, f)
		}
	}
	return token
}

func (l *lexer) processInvalidToken() tree.STToken {
	reader := l.reader
	for !l.isEndOfInvalidToken() {
		reader.Advance()
	}
	tokenText := l.getLexeme()
	invalidToken := tree.CreateInvalidToken(tokenText)
	invalidNodeMinutiae := tree.CreateInvalidNodeMinutiae(invalidToken)
	l.context.leadingTriviaList = append(l.context.leadingTriviaList, invalidNodeMinutiae)
	return invalidToken
}

func (l *lexer) getLexeme() string {
	return l.reader.GetMarkedChars()
}

// Check if we are at a synchronization point where we can resume normal parsing.
func (l *lexer) isEndOfInvalidToken() bool {
	reader := l.reader
	if reader.IsEOF() {
		return true
	}
	currentChar := reader.Peek()
	switch currentChar {
	case NEWLINE, CARRIAGE_RETURN, SPACE, TAB:
		return true
	// Separators
	case SEMICOLON, COLON, DOT, COMMA, OPEN_PARANTHESIS, CLOSE_PARANTHESIS,
		OPEN_BRACE, CLOSE_BRACE, OPEN_BRACKET, CLOSE_BRACKET, PIPE,
		QUESTION_MARK, DOUBLE_QUOTE, SINGLE_QUOTE, HASH, AT, BACKTICK, DOLLAR:
		return true
	// Arithmetic operators
	case EQUAL, PLUS, MINUS, ASTERISK, SLASH, PERCENT, GT, LT,
		BACKSLASH, EXCLAMATION_MARK, BITWISE_AND, BITWISE_XOR:
		return true
	default:
		return isIdentifierFollowingChar(currentChar)
	}
}

func isIdentifierFollowingChar(c rune) bool {
	return isIdentifierInitialChar(c) || unicode.IsDigit(c)
}

func isIdentifierInitialChar(c rune) bool {
	return ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') || c == '_' || isUnicodeIdentifierChar(c)
}

// Ported from io.ballerina.compiler.tree.parser.AbstractLexer.java (line 242-255)
// Check whether a given char is a unicode identifier char.
//
// UnicodeIdentifierChar := ^ ( AsciiChar | UnicodeNonIdentifierChar )
// AsciiChar := 0x0 .. 0x7F
// UnicodeNonIdentifierChar := UnicodePrivateUseChar | UnicodePatternWhiteSpaceChar | UnicodePatternSyntaxChar
func isUnicodeIdentifierChar(c rune) bool {
	// check ASCII char range
	if 0x0000 <= c && c <= 0x007F {
		return false
	}

	// check UNICODE private use char
	if isUnicodePrivateUseChar(c) || isUnicodePatternWhiteSpaceChar(c) {
		return false
	}

	// Approximate Java's Character.isUnicodeIdentifierPart() using Go's unicode package:
	// - Letters (L category: Lu, Ll, Lt, Lm, Lo)
	// - Marks (M category: Mn, Mc, Me)
	// - Numbers (N category: Nd, Nl, No - includes numeric letters like Roman numerals)
	// - Connector punctuation (Pc category)
	return unicode.IsLetter(c) ||
		unicode.IsMark(c) ||
		unicode.IsNumber(c) ||
		unicode.Is(unicode.Pc, c)
}

// Ported from io.ballerina.compiler.tree.parser.AbstractLexer.java (line 265-267)
func isUnicodePrivateUseChar(c rune) bool {
	return (0xE000 <= c && c <= 0xF8FF) ||
		(0xF0000 <= c && c <= 0xFFFFD) ||
		(0x100000 <= c && c <= 0x10FFFD)
}

func (l *lexer) processNumericEscape() {
	// Process '\'
	reader := l.reader
	reader.Advance()
	l.processNumericEscapeWithoutBackslash()
}

func (l *lexer) processNumericEscapeWithoutBackslash() {
	// Process 'u {'
	reader := l.reader
	reader.AdvanceN(2)

	// Process code-point
	if !isHexDigit(byte(reader.Peek())) {
		l.reportLexerError(common.ERROR_INVALID_STRING_NUMERIC_ESCAPE_SEQUENCE)
		return
	}

	reader.Advance()
	for isHexDigit(byte(reader.Peek())) {
		reader.Advance()
	}

	// Process close brace
	if reader.Peek() != CLOSE_BRACE {
		l.reportLexerError(common.ERROR_INVALID_STRING_NUMERIC_ESCAPE_SEQUENCE)
		return
	}

	reader.Advance()
}

func (l *lexer) reportInvalidEscapeSequence(nextChar rune) {
	escapeSequence := string(nextChar)
	l.reportLexerError(common.ERROR_INVALID_ESCAPE_SEQUENCE, escapeSequence)
}

func isValidQuotedIdentifierEscapeChar(c rune) bool {
	// ASCII letters are not allowed
	if ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') {
		return false
	}

	// Unicode pattern white space characters are not allowed
	return !isUnicodePatternWhiteSpaceChar(c)
}

// TODO: validate we can't use unicode
func isUnicodePatternWhiteSpaceChar(c rune) bool {
	return c == 0x200E || c == 0x200F || c == 0x2028 || c == 0x2029
}

func (l *lexer) processIdentifierEnd() {
	reader := l.reader
	for !reader.IsEOF() {
		nextChar := reader.Peek()
		if isIdentifierFollowingChar(nextChar) {
			reader.Advance()
			continue
		}

		if nextChar != BACKSLASH {
			break
		}

		// IdentifierSingleEscape | NumericEscape
		nextChar = reader.PeekN(1)
		switch nextChar {
		case NEWLINE, CARRIAGE_RETURN, TAB:
			reader.Advance()
			l.reportLexerError(common.ERROR_INVALID_ESCAPE_SEQUENCE, "")
		case 'u':
			// NumericEscape
			if reader.PeekN(2) == OPEN_BRACE {
				l.processNumericEscape()
			} else {
				reader.AdvanceN(2)
			}
			continue
		default:
			if !isValidQuotedIdentifierEscapeChar(nextChar) {
				l.reportInvalidEscapeSequence(nextChar)
			}
			reader.AdvanceN(2)
			continue
		}
		break
	}
}

func (l *lexer) processIdentifierOrKeyword() tree.STToken {
	l.processUnquotedIdentifier()
	tokenText := l.getLexeme()
	switch tokenText {
	case INT:
		return l.getSyntaxToken(common.INT_KEYWORD)
	case FLOAT:
		return l.getSyntaxToken(common.FLOAT_KEYWORD)
	case STRING:
		return l.getSyntaxToken(common.STRING_KEYWORD)
	case BOOLEAN:
		return l.getSyntaxToken(common.BOOLEAN_KEYWORD)
	case DECIMAL:
		return l.getSyntaxToken(common.DECIMAL_KEYWORD)
	case XML:
		return l.getSyntaxToken(common.XML_KEYWORD)
	case JSON:
		return l.getSyntaxToken(common.JSON_KEYWORD)
	case HANDLE:
		return l.getSyntaxToken(common.HANDLE_KEYWORD)
	case ANY:
		return l.getSyntaxToken(common.ANY_KEYWORD)
	case ANYDATA:
		return l.getSyntaxToken(common.ANYDATA_KEYWORD)
	case NEVER:
		return l.getSyntaxToken(common.NEVER_KEYWORD)
	case BYTE:
		return l.getSyntaxToken(common.BYTE_KEYWORD)

	// Keywords
	case PUBLIC:
		return l.getSyntaxToken(common.PUBLIC_KEYWORD)
	case PRIVATE:
		return l.getSyntaxToken(common.PRIVATE_KEYWORD)
	case FUNCTION:
		return l.getSyntaxToken(common.FUNCTION_KEYWORD)
	case RETURN:
		return l.getSyntaxToken(common.RETURN_KEYWORD)
	case RETURNS:
		return l.getSyntaxToken(common.RETURNS_KEYWORD)
	case EXTERNAL:
		return l.getSyntaxToken(common.EXTERNAL_KEYWORD)
	case TYPE:
		return l.getSyntaxToken(common.TYPE_KEYWORD)
	case RECORD:
		return l.getSyntaxToken(common.RECORD_KEYWORD)
	case OBJECT:
		return l.getSyntaxToken(common.OBJECT_KEYWORD)
	case REMOTE:
		return l.getSyntaxToken(common.REMOTE_KEYWORD)
	case ABSTRACT:
		return l.getSyntaxToken(common.ABSTRACT_KEYWORD)
	case CLIENT:
		return l.getSyntaxToken(common.CLIENT_KEYWORD)
	case IF:
		return l.getSyntaxToken(common.IF_KEYWORD)
	case ELSE:
		return l.getSyntaxToken(common.ELSE_KEYWORD)
	case WHILE:
		return l.getSyntaxToken(common.WHILE_KEYWORD)
	case TRUE:
		return l.getSyntaxToken(common.TRUE_KEYWORD)
	case FALSE:
		return l.getSyntaxToken(common.FALSE_KEYWORD)
	case CHECK:
		return l.getSyntaxToken(common.CHECK_KEYWORD)
	case FAIL:
		return l.getSyntaxToken(common.FAIL_KEYWORD)
	case CHECKPANIC:
		return l.getSyntaxToken(common.CHECKPANIC_KEYWORD)
	case CONTINUE:
		return l.getSyntaxToken(common.CONTINUE_KEYWORD)
	case BREAK:
		return l.getSyntaxToken(common.BREAK_KEYWORD)
	case PANIC:
		return l.getSyntaxToken(common.PANIC_KEYWORD)
	case IMPORT:
		return l.getSyntaxToken(common.IMPORT_KEYWORD)
	case AS:
		return l.getSyntaxToken(common.AS_KEYWORD)
	case SERVICE:
		return l.getSyntaxToken(common.SERVICE_KEYWORD)
	case ON:
		return l.getSyntaxToken(common.ON_KEYWORD)
	case RESOURCE:
		return l.getSyntaxToken(common.RESOURCE_KEYWORD)
	case LISTENER:
		return l.getSyntaxToken(common.LISTENER_KEYWORD)
	case CONST:
		return l.getSyntaxToken(common.CONST_KEYWORD)
	case FINAL:
		return l.getSyntaxToken(common.FINAL_KEYWORD)
	case TYPEOF:
		return l.getSyntaxToken(common.TYPEOF_KEYWORD)
	case IS:
		return l.getSyntaxToken(common.IS_KEYWORD)
	case NULL:
		return l.getSyntaxToken(common.NULL_KEYWORD)
	case LOCK:
		return l.getSyntaxToken(common.LOCK_KEYWORD)
	case ANNOTATION:
		return l.getSyntaxToken(common.ANNOTATION_KEYWORD)
	case SOURCE:
		return l.getSyntaxToken(common.SOURCE_KEYWORD)
	case VAR:
		return l.getSyntaxToken(common.VAR_KEYWORD)
	case WORKER:
		return l.getSyntaxToken(common.WORKER_KEYWORD)
	case PARAMETER:
		return l.getSyntaxToken(common.PARAMETER_KEYWORD)
	case FIELD:
		return l.getSyntaxToken(common.FIELD_KEYWORD)
	case ISOLATED:
		return l.getSyntaxToken(common.ISOLATED_KEYWORD)
	case XMLNS:
		return l.getSyntaxToken(common.XMLNS_KEYWORD)
	case FORK:
		return l.getSyntaxToken(common.FORK_KEYWORD)
	case MAP:
		return l.getSyntaxToken(common.MAP_KEYWORD)
	case FUTURE:
		return l.getSyntaxToken(common.FUTURE_KEYWORD)
	case TYPEDESC:
		return l.getSyntaxToken(common.TYPEDESC_KEYWORD)
	case TRAP:
		return l.getSyntaxToken(common.TRAP_KEYWORD)
	case IN:
		return l.getSyntaxToken(common.IN_KEYWORD)
	case FOREACH:
		return l.getSyntaxToken(common.FOREACH_KEYWORD)
	case TABLE:
		return l.getSyntaxToken(common.TABLE_KEYWORD)
	case ERROR:
		return l.getSyntaxToken(common.ERROR_KEYWORD)
	case LET:
		return l.getSyntaxToken(common.LET_KEYWORD)
	case STREAM:
		return l.getSyntaxToken(common.STREAM_KEYWORD)
	case NEW:
		return l.getSyntaxToken(common.NEW_KEYWORD)
	case READONLY:
		return l.getSyntaxToken(common.READONLY_KEYWORD)
	case DISTINCT:
		return l.getSyntaxToken(common.DISTINCT_KEYWORD)
	case FROM:
		return l.getSyntaxToken(common.FROM_KEYWORD)
	case START:
		return l.getSyntaxToken(common.START_KEYWORD)
	case FLUSH:
		return l.getSyntaxToken(common.FLUSH_KEYWORD)
	case WAIT:
		return l.getSyntaxToken(common.WAIT_KEYWORD)
	case DO:
		return l.getSyntaxToken(common.DO_KEYWORD)
	case TRANSACTION:
		return l.getSyntaxToken(common.TRANSACTION_KEYWORD)
	case COMMIT:
		return l.getSyntaxToken(common.COMMIT_KEYWORD)
	case RETRY:
		return l.getSyntaxToken(common.RETRY_KEYWORD)
	case ROLLBACK:
		return l.getSyntaxToken(common.ROLLBACK_KEYWORD)
	case TRANSACTIONAL:
		return l.getSyntaxToken(common.TRANSACTIONAL_KEYWORD)
	case ENUM:
		return l.getSyntaxToken(common.ENUM_KEYWORD)
	case BASE16:
		return l.getSyntaxToken(common.BASE16_KEYWORD)
	case BASE64:
		return l.getSyntaxToken(common.BASE64_KEYWORD)
	case MATCH:
		return l.getSyntaxToken(common.MATCH_KEYWORD)
	case CONFLICT:
		return l.getSyntaxToken(common.CONFLICT_KEYWORD)
	case CLASS:
		return l.getSyntaxToken(common.CLASS_KEYWORD)
	case CONFIGURABLE:
		return l.getSyntaxToken(common.CONFIGURABLE_KEYWORD)
	case WHERE:
		return l.getSyntaxToken(common.WHERE_KEYWORD)
	case SELECT:
		return l.getSyntaxToken(common.SELECT_KEYWORD)
	case LIMIT:
		return l.getSyntaxToken(common.LIMIT_KEYWORD)
	case OUTER:
		return l.getSyntaxToken(common.OUTER_KEYWORD)
	case EQUALS:
		return l.getSyntaxToken(common.EQUALS_KEYWORD)
	case ORDER:
		return l.getSyntaxToken(common.ORDER_KEYWORD)
	case BY:
		return l.getSyntaxToken(common.BY_KEYWORD)
	case ASCENDING:
		return l.getSyntaxToken(common.ASCENDING_KEYWORD)
	case DESCENDING:
		return l.getSyntaxToken(common.DESCENDING_KEYWORD)
	case JOIN:
		return l.getSyntaxToken(common.JOIN_KEYWORD)
	case RE:
		if l.getNextNonWhiteSpaceOrNonCommentChar() == BACKTICK {
			return l.getSyntaxToken(common.RE_KEYWORD)
		}
		return l.getIdentifierToken()
	default:
		return l.getIdentifierToken()
	}
}

// TODO: These should be in the lexer where it just seek forward instead of this back and forth
func (l *lexer) getNextNonWhiteSpaceOrNonCommentChar() rune {
	lookaheadCount := 0
	reader := l.reader
	nextChar := reader.PeekN(lookaheadCount)
	for nextChar != unicode.MaxRune {
		switch nextChar {
		case SPACE, TAB, FORM_FEED, CARRIAGE_RETURN, NEWLINE:
			lookaheadCount++
		case SLASH:
			if reader.PeekN(lookaheadCount+1) == SLASH {
				lookaheadCount += 2
				lookaheadCount = l.skipComment(lookaheadCount)
				break
			}
			return nextChar
		default:
			return nextChar
		}
		nextChar = reader.PeekN(lookaheadCount)
	}
	return nextChar
}

func (l *lexer) skipComment(lookaheadCount int) int {
	reader := l.reader
	nextChar := reader.PeekN(lookaheadCount)
	for nextChar != unicode.MaxRune {
		switch nextChar {
		case NEWLINE:
		case CARRIAGE_RETURN:
		default:
			lookaheadCount += 1
			nextChar = reader.PeekN(lookaheadCount)
			continue
		}
		break
	}
	return lookaheadCount
}

func (l *lexer) processNumericLiteral(startChar rune) tree.STToken {
	reader := l.reader
	nextChar := reader.Peek()
	if l.isHexIndicator(startChar, nextChar) {
		return l.processHexLiteral()
	}

	len := 1
	for !reader.IsEOF() {
		switch nextChar {
		case DOT, 'e', 'E', 'f', 'F', 'd', 'D':
			nextNextChar := reader.PeekN(1)
			if nextChar == DOT &&
				(nextNextChar == DOT || l.isDecimalNumberFollowedIdentifier()) {
				// This is to handle two cases:
				// 1. More than one dot. e.g. 2...10
				// 2. Method call. e.g. 2.toString()
				break
			}

			// In sem-var mode, only decimal integer literals are supported
			if l.context.mode == PARSER_MODE_IMPORT_MODE {
				break
			}

			// Integer part of the float cannot have a leading zero
			if startChar == '0' && len > 1 {
				l.reportLexerError(common.ERROR_LEADING_ZEROS_IN_NUMERIC_LITERALS)
			}

			// Code would not reach here if the floating point starts with a dot
			return l.processDecimalFloatLiteral()
		default:
			if unicode.IsDigit(nextChar) {
				reader.Advance()
				len++
				nextChar = reader.Peek()
				continue
			}
		}
		break
	}

	// Integer cannot have a leading zero
	if startChar == '0' && len > 1 {
		l.reportLexerError(common.ERROR_LEADING_ZEROS_IN_NUMERIC_LITERALS)
	}

	return l.getLiteral(common.DECIMAL_INTEGER_LITERAL_TOKEN)
}

func (l *lexer) getLiteral(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	trailingTrivia := l.processTrailingTrivia()
	return tree.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (l *lexer) processDecimalFloatLiteral() tree.STToken {
	reader := l.reader
	nextChar := reader.Peek()

	// For float literals start with a DOT, this condition will always be false,
	// as the reader is already advanced for the DOT before coming here.
	if nextChar == DOT {
		reader.Advance()
		nextChar = reader.Peek()

		if !unicode.IsDigit(nextChar) {
			// Make sure there is at least one digit after the dot
			// e.g. 2., 2.e12
			l.reportLexerError(common.ERROR_MISSING_DIGIT_AFTER_DOT)
		}
	}

	for unicode.IsDigit(nextChar) {
		reader.Advance()
		nextChar = reader.Peek()
	}

	switch nextChar {
	case 'e', 'E':
		return l.processExponent(false)
	case 'f', 'F', 'd', 'D':
		return l.parseFloatingPointTypeSuffix()
	default:
		return l.getLiteral(common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
	}
}

func (l *lexer) parseFloatingPointTypeSuffix() tree.STToken {
	l.reader.Advance()
	return l.getLiteral(common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
}

func (l *lexer) processExponent(isHex bool) tree.STToken {
	// Advance reader as exponent indicator is already validated
	reader := l.reader
	reader.Advance()
	nextChar := reader.Peek()

	// Capture if there is a sign
	if nextChar == PLUS || nextChar == MINUS {
		reader.Advance()
		nextChar = reader.Peek()
	}

	// Make sure at least one digit is present after the indicator
	if !unicode.IsDigit(nextChar) {
		l.reportLexerError(common.ERROR_MISSING_DIGIT_AFTER_EXPONENT_INDICATOR)
	}

	for unicode.IsDigit(nextChar) {
		reader.Advance()
		nextChar = reader.Peek()
	}

	if isHex {
		return l.getLiteral(common.HEX_FLOATING_POINT_LITERAL_TOKEN)
	}

	switch nextChar {
	case 'f', 'F', 'd', 'D':
		return l.parseFloatingPointTypeSuffix()
	default:
		return l.getLiteral(common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
	}
}

func (l *lexer) reportLexerError(errorCode common.DiagnosticErrorCode, args ...any) {
	diagnostic := tree.CreateDiagnostic(&errorCode, args...)
	l.context.diagnostics = append(l.context.diagnostics, diagnostic)
}

func (l *lexer) isDecimalNumberFollowedIdentifier() bool {
	reader := l.reader
	lookahead := 1
	lookaheadChar := reader.PeekN(lookahead)

	if unicode.IsDigit(lookaheadChar) {
		return false
	}

	switch lookaheadChar {
	case 'e', 'E':
		lookahead++

		lookaheadChar = reader.PeekN(lookahead)
		if lookaheadChar == PLUS || lookaheadChar == MINUS {
			return false
		}

		for unicode.IsDigit(lookaheadChar) {
			lookahead++
			lookaheadChar = reader.PeekN(lookahead)
		}

		if lookaheadChar == 'd' || lookaheadChar == 'D' || lookaheadChar == 'f' || lookaheadChar == 'F' {
			lookahead++
		}
	case 'd', 'D', 'f', 'F':
		lookahead++
	default:
		break
	}

	lookaheadChar = reader.PeekN(lookahead)
	return isIdentifierFollowingChar(lookaheadChar)
}

func (l *lexer) isHexIntFollowedIdentifier() bool {
	reader := l.reader
	lookahead := 1
	lookaheadChar := reader.PeekN(lookahead)

	if unicode.IsDigit(lookaheadChar) {
		return false
	}

	for isHexDigit(byte(lookaheadChar)) {
		lookahead++
		lookaheadChar = reader.PeekN(lookahead)
	}

	switch lookaheadChar {
	case 'p', 'P':
		lookahead++

		lookaheadChar = reader.PeekN(lookahead)
		if lookaheadChar == PLUS || lookaheadChar == MINUS {
			return false
		}

		lookaheadChar = reader.PeekN(lookahead)
		for unicode.IsDigit(lookaheadChar) {
			lookahead++
			lookaheadChar = reader.PeekN(lookahead)
		}
	}

	return isIdentifierFollowingChar(lookaheadChar)
}

func (l *lexer) processHexLiteral() tree.STToken {
	reader := l.reader
	reader.Advance() // advance for "x" or "X"
	containsHexDigit := false

	for isHexDigit(byte(reader.Peek())) {
		reader.Advance()
		containsHexDigit = true
	}

	nextChar := reader.Peek()
	switch nextChar {
	case DOT:
		if l.isHexIntFollowedIdentifier() {
			// e.g. 0x.max(), 0xA2.max()
			return l.getHexIntegerLiteral()
		}

		reader.Advance()
		if !isHexDigit(byte(reader.Peek())) {
			// Make sure there is at least one hex-digit after the dot
			// e.g. 0x., 0xAB.
			l.reportLexerError(common.ERROR_MISSING_HEX_DIGIT_AFTER_DOT)
		}

		nextChar = reader.Peek()
		for isHexDigit(byte(nextChar)) {
			reader.Advance()
			nextChar = reader.Peek()
		}

		switch nextChar {
		case 'p', 'P':
			return l.processExponent(true)
		}
	case 'p', 'P':
		if !containsHexDigit {
			l.reportLexerError(common.ERROR_MISSING_HEX_NUMBER_AFTER_HEX_INDICATOR)
		}
		return l.processExponent(true)
	default:
		return l.getHexIntegerLiteral()
	}

	return l.getLiteral(common.HEX_FLOATING_POINT_LITERAL_TOKEN)
}

func (l *lexer) getHexIntegerLiteral() tree.STToken {
	lexeme := l.getLexeme()
	if lexeme == "0x" || lexeme == "0X" {
		l.reportLexerError(common.ERROR_MISSING_HEX_NUMBER_AFTER_HEX_INDICATOR)
	}

	return l.getLiteral(common.HEX_INTEGER_LITERAL_TOKEN)
}

func (l *lexer) isHexIndicator(startChar rune, nextChar rune) bool {
	return startChar == '0' && (nextChar == 'x' || nextChar == 'X')
}

func (l *lexer) processQuotedIdentifier() tree.STToken {
	l.processIdentifierEnd()
	if string(SINGLE_QUOTE) == l.getLexeme() {
		l.reportLexerError(common.ERROR_INCOMPLETE_QUOTED_IDENTIFIER)
	}
	return l.getIdentifierToken()
}

func (l *lexer) getBacktickToken() tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	// Trivia after the back-tick including whitespace belongs to the content of the back-tick.
	// Therefore, do not process trailing trivia for starting back-tick. We reach here only for
	// starting back-tick. Ending back-tick is processed by the template mode.
	trailingTrivia := tree.CreateEmptyNodeList()
	return tree.CreateTokenFrom(common.BACKTICK_TOKEN, leadingTrivia, trailingTrivia)
}

func (l *lexer) processExclamationMarkOperator() tree.STToken {
	reader := l.reader
	switch reader.Peek() {
	case EQUAL:
		reader.Advance()
		if reader.Peek() == EQUAL {
			// this is '!=='
			reader.Advance()
			return l.getSyntaxToken(common.NOT_DOUBLE_EQUAL_TOKEN)
		}
		// this is '!='
		return l.getSyntaxToken(common.NOT_EQUAL_TOKEN)
	default:
		// this is '!is'
		if l.isNotIsToken() {
			reader.AdvanceN(2)
			return l.getSyntaxToken(common.NOT_IS_KEYWORD)
		}
		// this is '!'
		return l.getSyntaxToken(common.EXCLAMATION_MARK_TOKEN)
	}
}

func (l *lexer) isNotIsToken() bool {
	reader := l.reader
	return (reader.Peek() == 'i' && reader.PeekN(1) == 's') &&
		(!isIdentifierFollowingChar(reader.PeekN(2)) && reader.PeekN(2) != BACKSLASH)
}

func (l *lexer) processTokenStartWithGt() tree.STToken {
	reader := l.reader
	if reader.Peek() == EQUAL {
		reader.Advance()
		return l.getSyntaxToken(common.GT_EQUAL_TOKEN)
	}

	if reader.Peek() != GT {
		return l.getSyntaxToken(common.GT_TOKEN)
	}

	nextChar := reader.PeekN(1)
	switch nextChar {
	case GT:
		if reader.PeekN(2) == EQUAL {
			// ">>>="
			reader.AdvanceN(2)
			return l.getSyntaxToken(common.TRIPPLE_GT_TOKEN)
		}
		return l.getSyntaxToken(common.GT_TOKEN)
	case EQUAL:
		// ">>="
		reader.AdvanceN(1)
		return l.getSyntaxToken(common.DOUBLE_GT_TOKEN)
	default:
		return l.getSyntaxToken(common.GT_TOKEN)
	}
}

func (l *lexer) processTokenStartWithLt() tree.STToken {
	reader := l.reader
	switch reader.Peek() {
	case EQUAL:
		reader.Advance()
		return l.getSyntaxToken(common.LT_EQUAL_TOKEN)
	case MINUS:
		nextNextChar := reader.PeekN(1)
		if unicode.IsDigit(nextNextChar) {
			return l.getSyntaxToken(common.LT_TOKEN)
		}
		reader.Advance()
		return l.getSyntaxToken(common.LEFT_ARROW_TOKEN)
	case LT:
		reader.Advance()
		return l.getSyntaxToken(common.DOUBLE_LT_TOKEN)
	default:
		return l.getSyntaxToken(common.LT_TOKEN)
	}
}

func (l *lexer) processSlashToken() tree.STToken {
	// check for the second char
	reader := l.reader
	if reader.Peek() != ASTERISK {
		return l.getSyntaxToken(common.SLASH_TOKEN)
	}

	reader.Advance()
	if reader.Peek() != ASTERISK {
		return l.getSyntaxToken(common.SLASH_ASTERISK_TOKEN)
	} else if reader.PeekN(1) == SLASH && reader.PeekN(2) == LT {
		reader.AdvanceN(3)
		return l.getSyntaxToken(common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN)
	} else {
		return l.getSyntaxToken(common.SLASH_ASTERISK_TOKEN)
	}
}

func (l *lexer) processEqualOperator() tree.STToken {
	reader := l.reader
	switch reader.Peek() {
	case EQUAL:
		reader.Advance()
		if reader.Peek() == EQUAL {
			// this is '==='
			reader.Advance()
			return l.getSyntaxToken(common.TRIPPLE_EQUAL_TOKEN)
		}
		// this is '=='
		return l.getSyntaxToken(common.DOUBLE_EQUAL_TOKEN)
	case GT:
		// this is '=>'
		reader.Advance()
		return l.getSyntaxToken(common.RIGHT_DOUBLE_ARROW_TOKEN)
	default:
		// this is '='
		return l.getSyntaxToken(common.EQUAL_TOKEN)
	}
}

func (l *lexer) processDocumentationString() tree.STToken {
	reader := l.reader
	nextChar := reader.Peek()
	for !reader.IsEOF() {
		switch nextChar {
		case CARRIAGE_RETURN, NEWLINE:
			// Advance reader for the new line
			if reader.Peek() == CARRIAGE_RETURN && reader.PeekN(1) == NEWLINE {
				reader.Advance()
			}
			reader.Advance()

			// Look ahead and see if next line also belongs to the documentation.
			// i.e. look for a `WS #` match
			// If there's a match, advance reader for the next line as well.
			// Otherwise terminate documentation content after the new line.
			lookAheadCount := 0
			lookAheadChar := reader.PeekN(lookAheadCount)
			for lookAheadChar == SPACE || lookAheadChar == TAB {
				lookAheadCount++
				lookAheadChar = reader.PeekN(lookAheadCount)
			}

			if lookAheadChar != HASH {
				// Next line does not belong to documentation, hence break
				break
			}

			reader.AdvanceN(lookAheadCount)
			nextChar = reader.Peek()
			continue
		default:
			reader.Advance()
			nextChar = reader.Peek()
			continue
		}
		break
	}

	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	trailingTrivia := tree.CreateEmptyNodeList() // No trailing trivia
	return tree.CreateLiteralValueToken(common.DOCUMENTATION_STRING, lexeme, leadingTrivia, trailingTrivia)
}

func (l *lexer) processStringLiteral() tree.STToken {
	reader := l.reader
	var nextChar rune
	for !reader.IsEOF() {
		nextChar = reader.Peek()
		switch nextChar {
		case NEWLINE, CARRIAGE_RETURN:
			l.reportLexerError(common.ERROR_MISSING_DOUBLE_QUOTE)
		case DOUBLE_QUOTE:
			reader.Advance()
		case BACKSLASH:
			switch reader.PeekN(1) {
			case 't', 'n', 'r', BACKSLASH, DOUBLE_QUOTE:
				reader.AdvanceN(2)
				continue
			case 'u':
				if reader.PeekN(2) == OPEN_BRACE {
					l.processNumericEscape()
				} else {
					l.reportLexerError(common.ERROR_INVALID_STRING_NUMERIC_ESCAPE_SEQUENCE)
					reader.AdvanceN(2)
				}
				continue
			default:
				l.reportInvalidEscapeSequence(reader.PeekN(1))
				reader.Advance()
				continue
			}
		default:
			reader.Advance()
			continue
		}
		break
	}

	return l.getLiteral(common.STRING_LITERAL_TOKEN)
}

func (l *lexer) processPipeOperator() tree.STToken {
	reader := l.reader
	switch reader.Peek() {
	case CLOSE_BRACE:
		reader.Advance()
		return l.getSyntaxToken(common.CLOSE_BRACE_PIPE_TOKEN)
	case PIPE:
		reader.Advance()
		return l.getSyntaxToken(common.LOGICAL_OR_TOKEN)
	default:
		return l.getSyntaxToken(common.PIPE_TOKEN)
	}
}

func (l *lexer) processDot() tree.STToken {
	reader := l.reader
	nextChar := reader.Peek()
	switch nextChar {
	case DOT:
		nextNextChar := reader.PeekN(1)
		switch nextNextChar {
		case DOT:
			reader.AdvanceN(2)
			return l.getSyntaxToken(common.ELLIPSIS_TOKEN)
		case LT:
			reader.AdvanceN(2)
			return l.getSyntaxToken(common.DOUBLE_DOT_LT_TOKEN)
		}
	case AT:
		reader.Advance()
		return l.getSyntaxToken(common.ANNOT_CHAINING_TOKEN)
	case LT:
		reader.Advance()
		return l.getSyntaxToken(common.DOT_LT_TOKEN)
	}

	if l.context.mode != PARSER_MODE_IMPORT_MODE && unicode.IsDigit(nextChar) {
		return l.processDecimalFloatLiteral()
	}
	return l.getSyntaxToken(common.DOT_TOKEN)
}

func (l *lexer) getIdentifierToken() tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	trailingTrivia := l.processTrailingTrivia()
	return tree.CreateIdentifierToken(lexeme, leadingTrivia, trailingTrivia)
}

func (l *lexer) processUnquotedIdentifier() {
	l.processIdentifierEnd()
}

func (l *lexer) getSyntaxToken(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := l.processTrailingTrivia()
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *lexer) getLeadingTrivia() tree.STNode {
	trivia := tree.CreateNodeList(l.context.leadingTriviaList...)
	l.context.leadingTriviaList = make([]tree.STNode, 0, INITIAL_TRIVIA_CAPACITY)
	return trivia
}

func (l *lexer) processTrailingTrivia() tree.STNode {
	triviaList := make([]tree.STNode, 0, INITIAL_TRIVIA_CAPACITY)
	l.processSyntaxTrivia(&triviaList, false)
	result := tree.CreateNodeList(triviaList...)
	return result
}

func (l *lexer) processSyntaxTrivia(triviaList *[]tree.STNode, isLeading bool) {
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
				break
			}
			return
		case SLASH:
			if reader.PeekN(1) == SLASH {
				*triviaList = append(*triviaList, l.processComment())
				break
			}
			return
		default:
			return
		}
	}
}

func (l *lexer) processComment() tree.STNode {
	reader := l.reader
	reader.AdvanceN(2)
	nextToken := reader.Peek()
	for !reader.IsEOF() {
		switch nextToken {
		case NEWLINE, CARRIAGE_RETURN:
		default:
			reader.Advance()
			nextToken = reader.Peek()
			continue
		}
		break
	}
	return tree.CreateMinutiae(common.COMMENT_MINUTIAE, l.getLexeme())
}

func (l *lexer) processEndOfLine() tree.STNode {
	reader := l.reader
	c := reader.Peek()
	switch c {
	case NEWLINE:
		reader.Advance()
		return tree.CreateMinutiae(common.END_OF_LINE_MINUTIAE, l.getLexeme())
	case CARRIAGE_RETURN:
		reader.Advance()
		if reader.Peek() == NEWLINE {
			reader.Advance()
		}
		return tree.CreateMinutiae(common.END_OF_LINE_MINUTIAE, l.getLexeme())
	default:
		panic("unreachable")
	}
}

func (l *lexer) processWhitespaces() tree.STNode {
	reader := l.reader
	for !reader.IsEOF() {
		c := reader.Peek()
		switch c {
		case SPACE, TAB, FORM_FEED:
			reader.Advance()
			continue
		default:
		}
		break
	}
	return tree.CreateMinutiae(common.WHITESPACE_MINUTIAE, l.getLexeme())
}

func (l *lexer) readTokenInBracedContentInInterpolation() tree.STToken {
	reader := l.reader
	reader.Mark()
	nextChar := reader.Peek()
	switch nextChar {
	case OPEN_BRACE:
		l.StartMode(PARSER_MODE_INTERPOLATION_BRACED_CONTENT)
	case CLOSE_BRACE:
		l.EndMode()
	case BACKTICK:
		// Recursively end backtick string related contexts
		for l.context.mode != PARSER_MODE_DEFAULT_MODE {
			l.EndMode()
		}
		reader.Advance()
		return l.getBacktickToken()
	default:
		// Otherwise read the token from default mode.
		break
	}

	return l.readToken()
}

func (l *lexer) readTokenInInterpolation() tree.STToken {
	reader := l.reader
	reader.Mark()
	nextChar := reader.Peek()
	switch nextChar {
	case OPEN_BRACE:
		// Start braced-content mode. This is to keep track of the
		// open-brace and the corresponding close-brace. This way,
		// those will not be mistaken as the close-brace of the
		// interpolation end.
		l.StartMode(PARSER_MODE_INTERPOLATION_BRACED_CONTENT)
		return l.readToken()
	case CLOSE_BRACE:
		// Close-brace in the interpolation mode definitely means its
		// then end of the interpolation.
		l.EndMode()
		reader.Advance()
		return l.getSyntaxTokenWithoutTrailingTrivia(common.CLOSE_BRACE_TOKEN)
	case BACKTICK:
		// If we are inside the interpolation, that means its no longer XML
		// mode, but in the default mode. Hence treat the back-tick in the
		// same way as in the default mode.
		fallthrough
	default:
		// Otherwise read the token from default mode.
		return l.readToken()
	}
}

func (l *lexer) getSyntaxTokenWithoutTrailingTrivia(kind common.SyntaxKind) tree.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := tree.CreateEmptyNodeList()
	return tree.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *lexer) processLeadingTrivia() {
	l.processSyntaxTrivia(&l.context.leadingTriviaList, true)
}

func (l *lexer) readRegExpTemplateToken() tree.STToken {
	reader := l.reader
	shouldProcessInterpolations := true
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case BACKTICK:
		reader.Advance()
		l.EndMode()
		return l.getSyntaxToken(common.BACKTICK_TOKEN)
	case DOLLAR:
		if reader.PeekN(1) == OPEN_BRACE {
			// Switch to interpolation mode. Then the next token will be read in that mode.
			l.StartMode(PARSER_MODE_INTERPOLATION)
			reader.AdvanceN(2)

			return l.getSyntaxToken(common.INTERPOLATION_START_TOKEN)
		}
		fallthrough
	default:
		if nextChar == OPEN_BRACKET {
			shouldProcessInterpolations = false
		}
		for !reader.IsEOF() {
			if shouldProcessInterpolations && reader.Peek() == BACKSLASH &&
				reader.PeekN(1) == OPEN_BRACKET {
				// Escaped open brackets are not considered as the start of a no interpolation context.
				reader.Advance()
			}
			reader.Advance()
			nextChar = reader.Peek()
			switch nextChar {
			case DOLLAR:
				if shouldProcessInterpolations && reader.PeekN(1) == OPEN_BRACE {
					break
				}
				continue
			case BACKTICK:
			case OPEN_BRACKET:
				shouldProcessInterpolations = false
				continue
			case CLOSE_BRACKET:
				shouldProcessInterpolations = true
				continue
			case BACKSLASH:
				if !shouldProcessInterpolations && reader.PeekN(1) == CLOSE_BRACKET {
					reader.Advance()
				}
				continue
			default:
				continue
			}
			break
		}
	}
	return l.getLiteral(common.TEMPLATE_STRING)
}

func (l *lexer) readPromptToken() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	if nextChar == CLOSE_BRACE {
		reader.Advance()
		l.EndMode()
		return l.getSyntaxToken(common.CLOSE_BRACE_TOKEN)
	}

	if nextChar == DOLLAR && reader.PeekN(1) == OPEN_BRACE {
		// Switch to interpolation mode. Then the next token will be read in that mode.
		l.StartMode(PARSER_MODE_INTERPOLATION)
		reader.AdvanceN(2)

		return l.getSyntaxToken(common.INTERPOLATION_START_TOKEN)
	}

	for !reader.IsEOF() {
		reader.Advance()
		nextChar = reader.Peek()
		nextNextChar := reader.PeekN(1)
		if nextChar == CLOSE_BRACE ||
			(nextChar == DOLLAR && nextNextChar == OPEN_BRACE) {
			break
		}

		if nextChar == BACKSLASH {
			if nextNextChar != CLOSE_BRACE && nextNextChar != BACKSLASH {
				l.reportInvalidEscapeSequence(reader.PeekN(1))
			}
			reader.Advance()
		}
	}

	return l.getLiteral(common.PROMPT_CONTENT)
}

func (l *lexer) readTemplateToken() tree.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(common.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	if nextChar == BACKTICK {
		reader.Advance()
		l.EndMode()
		return l.getSyntaxToken(common.BACKTICK_TOKEN)
	}

	if nextChar == DOLLAR && reader.PeekN(1) == OPEN_BRACE {
		// Switch to interpolation mode. Then the next token will be read in that mode.
		l.StartMode(PARSER_MODE_INTERPOLATION)
		reader.AdvanceN(2)

		return l.getSyntaxToken(common.INTERPOLATION_START_TOKEN)
	}

	for !reader.IsEOF() {
		reader.Advance()
		nextChar = reader.Peek()
		if nextChar == BACKTICK ||
			(nextChar == DOLLAR && reader.PeekN(1) == OPEN_BRACE) {
			break
		}
	}

	return l.getLiteral(common.TEMPLATE_STRING)
}

type ParserMode uint8

const (
	// Ballerina Parser
	PARSER_MODE_DEFAULT_MODE ParserMode = iota
	PARSER_MODE_IMPORT_MODE
	PARSER_MODE_TEMPLATE
	PARSER_MODE_INTERPOLATION
	PARSER_MODE_INTERPOLATION_BRACED_CONTENT
	PARSER_MODE_REGEXP
	PARSER_MODE_PROMPT

	// Documentation Parser
	PARSER_MODE_DOC_LINE_START_HASH
	PARSER_MODE_DOC_LINE_DIFFERENTIATOR
	PARSER_MODE_DOC_INTERNAL
	PARSER_MODE_DOC_PARAMETER
	PARSER_MODE_DOC_REFERENCE_TYPE
	PARSER_MODE_DOC_SINGLE_BACKTICK_CONTENT
	PARSER_MODE_DOC_DOUBLE_BACKTICK_CONTENT
	PARSER_MODE_DOC_TRIPLE_BACKTICK_CONTENT
	PARSER_MODE_DOC_CODE_REF_END
	PARSER_MODE_DOC_CODE_LINE_START_HASH

	// XML Parser
	PARSER_MODE_XML_CONTENT
	PARSER_MODE_XML_ELEMENT_START_TAG
	PARSER_MODE_XML_ELEMENT_END_TAG
	PARSER_MODE_XML_TEXT
	PARSER_MODE_XML_ATTRIBUTES
	PARSER_MODE_XML_COMMENT
	PARSER_MODE_XML_PI
	PARSER_MODE_XML_PI_DATA
	PARSER_MODE_XML_SINGLE_QUOTED_STRING
	PARSER_MODE_XML_DOUBLE_QUOTED_STRING
	PARSER_MODE_XML_CDATA_SECTION

	// RegExp Parser
	PARSER_MODE_RE_DISJUNCTION
	PARSER_MODE_RE_FLAG_EXPRESSION
	PARSER_MODE_RE_UNICODE_PROP_ESCAPE
	PARSER_MODE_RE_UNICODE_PROPERTY_VALUE
	PARSER_MODE_RE_ESCAPE
	PARSER_MODE_RE_CHAR_CLASS
	PARSER_MODE_RE_QUOTE_ESCAPE
)

const (
	PUBLIC        = "public"
	PRIVATE       = "private"
	FUNCTION      = "function"
	RETURN        = "return"
	RETURNS       = "returns"
	EXTERNAL      = "external"
	TYPE          = "type"
	RECORD        = "record"
	OBJECT        = "object"
	REMOTE        = "remote"
	ABSTRACT      = "abstract"
	CLIENT        = "client"
	IF            = "if"
	ELSE          = "else"
	WHILE         = "while"
	PANIC         = "panic"
	TRUE          = "true"
	FALSE         = "false"
	CHECK         = "check"
	FAIL          = "fail"
	CHECKPANIC    = "checkpanic"
	CONTINUE      = "continue"
	BREAK         = "break"
	IMPORT        = "import"
	AS            = "as"
	ON            = "on"
	RESOURCE      = "resource"
	LISTENER      = "listener"
	CONST         = "const"
	FINAL         = "final"
	TYPEOF        = "typeof"
	IS            = "is"
	NULL          = "null"
	LOCK          = "lock"
	ANNOTATION    = "annotation"
	SOURCE        = "source"
	WORKER        = "worker"
	PARAMETER     = "parameter"
	FIELD         = "field"
	ISOLATED      = "isolated"
	XMLNS         = "xmlns"
	FORK          = "fork"
	TRAP          = "trap"
	IN            = "in"
	FOREACH       = "foreach"
	TABLE         = "table"
	KEY           = "key"
	ERROR         = "error"
	LET           = "let"
	STREAM        = "stream"
	NEW           = "new"
	READONLY      = "readonly"
	DISTINCT      = "distinct"
	FROM          = "from"
	WHERE         = "where"
	SELECT        = "select"
	START         = "start"
	FLUSH         = "flush"
	DEFAULT       = "default"
	WAIT          = "wait"
	DO            = "do"
	TRANSACTION   = "transaction"
	TRANSACTIONAL = "transactional"
	COMMIT        = "commit"
	RETRY         = "retry"
	ROLLBACK      = "rollback"
	ENUM          = "enum"
	BASE16        = "base16"
	BASE64        = "base64"
	MATCH         = "match"
	CONFLICT      = "conflict"
	LIMIT         = "limit"
	JOIN          = "join"
	OUTER         = "outer"
	EQUALS        = "equals"
	ORDER         = "order"
	BY            = "by"
	ASCENDING     = "ascending"
	DESCENDING    = "descending"
	CLASS         = "class"
	CONFIGURABLE  = "configurable"
	NATURAL       = "natural"

	// For BFM only
	VARIABLE = "variable"
	MODULE   = "module"

	// Types
	INT      = "int"
	FLOAT    = "float"
	STRING   = "string"
	BOOLEAN  = "boolean"
	DECIMAL  = "decimal"
	XML      = "xml"
	JSON     = "json"
	HANDLE   = "handle"
	ANY      = "any"
	ANYDATA  = "anydata"
	SERVICE  = "service"
	VAR      = "var"
	NEVER    = "never"
	MAP      = "map"
	FUTURE   = "future"
	TYPEDESC = "typedesc"
	BYTE     = "byte"
	// Separators
	SEMICOLON         = ';'
	COLON             = ':'
	DOT               = '.'
	COMMA             = ','
	OPEN_PARANTHESIS  = '('
	CLOSE_PARANTHESIS = ')'
	OPEN_BRACE        = '{'
	CLOSE_BRACE       = '}'
	OPEN_BRACKET      = '['
	CLOSE_BRACKET     = ']'
	PIPE              = '|'
	QUESTION_MARK     = '?'
	DOUBLE_QUOTE      = '"'
	SINGLE_QUOTE      = '\''
	HASH              = '#'
	AT                = '@'
	BACKTICK          = '`'
	DOLLAR            = '$'

	// Arithmetic opera
	EQUAL            = '='
	PLUS             = '+'
	MINUS            = '-'
	ASTERISK         = '*'
	SLASH            = '/'
	PERCENT          = '%'
	GT               = '>'
	LT               = '<'
	BACKSLASH        = '\\'
	EXCLAMATION_MARK = '!'
	BITWISE_AND      = '&'
	BITWISE_XOR      = '^'
	NEGATION         = '~'

	// Other
	NEWLINE         = '\n' // equivalent to 0xA
	CARRIAGE_RETURN = '\r' // equivalent to 0xD
	TAB             = 0x9
	SPACE           = 0x20
	FORM_FEED       = 0xC

	RE = "re"
)
