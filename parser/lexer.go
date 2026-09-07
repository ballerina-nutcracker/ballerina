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

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser/common"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

// TODO: we have lot of unbounded lookaheads which are implemented by incrementing a lookahead count and repeatedly
// calling

// FIXME: get rid of repeated l.reader references in ai code

const initialTriviaCapacity = 1

type tokenLexer interface {
	NextToken() st.STToken
	StartMode(mode parserMode)
	SwitchMode(mode parserMode)
	EndMode()
	GetCurrentMode() parserMode
}

// TODO: introduce diagnostic context with flags and a channel
type lexer struct {
	reader   text.CharReader
	context  lexerContext
	ctx      *compilercontext.CompilerContext
	location diagnostics.Location
	debug    *debugOutput
}

type lexerContext struct {
	mode              parserMode
	modeStack         []parserMode
	leadingTriviaList []st.STNode
	diagnostics       []st.STNodeDiagnostic
}

func newLexer(ctx *compilercontext.CompilerContext, diagnosticIdentity string, reader text.CharReader, debug *debugOutput) *lexer {
	return &lexer{
		reader:   reader,
		context:  lexerContext{},
		ctx:      ctx,
		location: diagnostics.NewLocationForIdentity(ctx.DiagnosticEnv(), diagnosticIdentity, 0, 0),
		debug:    debug,
	}
}

func (l *lexer) internalError(message string) {
	l.ctx.InternalError(message, l.location)
}

func (l *lexer) StartMode(mode parserMode) {
	l.context.mode = mode
	l.context.modeStack = append(l.context.modeStack, mode)
}

func (l *lexer) SwitchMode(mode parserMode) {
	l.context.modeStack = l.context.modeStack[:len(l.context.modeStack)-1]
	l.context.mode = mode
	l.context.modeStack = append(l.context.modeStack, mode)
}

func (l *lexer) EndMode() {
	if len(l.context.modeStack) == 0 {
		l.internalError("cannot end mode: mode stack is empty")
		return
	}
	l.context.modeStack = l.context.modeStack[:len(l.context.modeStack)-1]
	if len(l.context.modeStack) == 0 {
		l.context.mode = parserModeDefaultMode
	} else {
		l.context.mode = l.context.modeStack[len(l.context.modeStack)-1]
	}
}

func (l *lexer) GetCurrentMode() parserMode {
	return l.context.mode
}

func (l *lexer) NextToken() st.STToken {
	var token st.STToken
	switch l.context.mode {
	case parserModeTemplate:
		token = l.readTemplateToken()
	case parserModePrompt:
		token = l.readPromptToken()
	case parserModeRegexp:
		token = l.readRegExpTemplateToken()
	case parserModeInterpolation:
		l.processLeadingTrivia()
		token = l.readTokenInInterpolation()
	case parserModeInterpolationBracedContent:
		l.processLeadingTrivia()
		token = l.readTokenInBracedContentInInterpolation()
	default:
		l.processLeadingTrivia()
		token = l.readToken()
	}
	if len(l.context.diagnostics) > 0 {
		token = st.AddSyntaxDiagnostics(token, l.context.diagnostics)
		l.context.diagnostics = nil
	}
	l.debug.write(l.debug.options.DumpTokens, func() string { return st.ToSexpr(token) })
	return token
}

func (l *lexer) readToken() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(st.EOF_TOKEN)
	}
	c := reader.Peek()
	if c == backslash {
		l.processUnquotedIdentifier()
		return l.getIdentifierToken()
	}
	reader.Advance()
	var token st.STToken
	switch c {
	case colon:
		token = l.getSyntaxToken(st.COLON_TOKEN)
	case semicolon:
		token = l.getSyntaxToken(st.SEMICOLON_TOKEN)
	case dot:
		token = l.processDot()
	case comma:
		token = l.getSyntaxToken(st.COMMA_TOKEN)
	case openParanthesis:
		token = l.getSyntaxToken(st.OPEN_PAREN_TOKEN)
	case closeParanthesis:
		token = l.getSyntaxToken(st.CLOSE_PAREN_TOKEN)
	case openBrace:
		if reader.Peek() == pipe {
			reader.Advance()
			token = l.getSyntaxToken(st.OPEN_BRACE_PIPE_TOKEN)
		} else {
			token = l.getSyntaxToken(st.OPEN_BRACE_TOKEN)
		}
	case closeBrace:
		token = l.getSyntaxToken(st.CLOSE_BRACE_TOKEN)
	case openBracket:
		token = l.getSyntaxToken(st.OPEN_BRACKET_TOKEN)
	case closeBracket:
		token = l.getSyntaxToken(st.CLOSE_BRACKET_TOKEN)
	case pipe:
		token = l.processPipeOperator()
	case questionMark:
		if reader.Peek() == dot && reader.PeekN(1) != dot {
			reader.Advance()
			token = l.getSyntaxToken(st.OPTIONAL_CHAINING_TOKEN)
		} else if reader.Peek() == colon {
			reader.Advance()
			token = l.getSyntaxToken(st.ELVIS_TOKEN)
		} else {
			token = l.getSyntaxToken(st.QUESTION_MARK_TOKEN)
		}
	case doubleQuote:
		token = l.processStringLiteral()
	case hash:
		token = l.processDocumentationString()
	case at:
		token = l.getSyntaxToken(st.AT_TOKEN)
	case equal:
		token = l.processEqualOperator()
	case plus:
		token = l.getSyntaxToken(st.PLUS_TOKEN)
	case minus:
		if reader.Peek() == gt {
			reader.Advance()
			if reader.Peek() == gt {
				reader.Advance()
				token = l.getSyntaxToken(st.SYNC_SEND_TOKEN)
			} else {
				token = l.getSyntaxToken(st.RIGHT_ARROW_TOKEN)
			}
		} else {
			token = l.getSyntaxToken(st.MINUS_TOKEN)
		}
	case asterisk:
		token = l.getSyntaxToken(st.ASTERISK_TOKEN)
	case slash:
		token = l.processSlashToken()
	case percent:
		token = l.getSyntaxToken(st.PERCENT_TOKEN)
	case lt:
		token = l.processTokenStartWithLt()
	case gt:
		token = l.processTokenStartWithGt()
	case exclamationMark:
		token = l.processExclamationMarkOperator()
	case bitwiseAnd:
		if reader.Peek() == bitwiseAnd {
			reader.Advance()
			token = l.getSyntaxToken(st.LOGICAL_AND_TOKEN)
		} else {
			token = l.getSyntaxToken(st.BITWISE_AND_TOKEN)
		}
	case bitwiseXor:
		token = l.getSyntaxToken(st.BITWISE_XOR_TOKEN)
	case negation:
		token = l.getSyntaxToken(st.NEGATION_TOKEN)
	case backtick:
		l.StartMode(parserModeTemplate)
		token = l.getBacktickToken()
	case singleQuote:
		token = l.processQuotedIdentifier()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		token = l.processNumericLiteral(c)
	default:
		if isIdentifierInitialChar(c) {
			token = l.processIdentifierOrKeyword()
		} else {
			invalidToken := l.processInvalidToken()
			token = l.NextToken()
			f := st.CreateDiagnostic(&common.ERROR_INVALID_TOKEN, invalidToken)
			token = st.AddSyntaxDiagnostic(token, f)
		}
	}
	return token
}

func (l *lexer) processInvalidToken() st.STToken {
	reader := l.reader
	for !l.isEndOfInvalidToken() {
		reader.Advance()
	}
	tokenText := l.getLexeme()
	invalidToken := st.CreateInvalidToken(tokenText)
	invalidNodeMinutiae := st.CreateInvalidNodeMinutiae(invalidToken)
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
	case newline, carriageReturn, space, tab:
		return true
	// Separators
	case semicolon, colon, dot, comma, openParanthesis, closeParanthesis,
		openBrace, closeBrace, openBracket, closeBracket, pipe,
		questionMark, doubleQuote, singleQuote, hash, at, backtick, dollar:
		return true
	// Arithmetic operators
	case equal, plus, minus, asterisk, slash, percent, gt, lt,
		backslash, exclamationMark, bitwiseAnd, bitwiseXor:
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

// Ported from io.ballerina.compiler.st.parser.AbstractLexer.java (line 242-255)
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

// Ported from io.ballerina.compiler.st.parser.AbstractLexer.java (line 265-267)
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
	if reader.Peek() != closeBrace {
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

		if nextChar != backslash {
			break
		}

		// IdentifierSingleEscape | NumericEscape
		nextChar = reader.PeekN(1)
		switch nextChar {
		case newline, carriageReturn, tab:
			reader.Advance()
			l.reportLexerError(common.ERROR_INVALID_ESCAPE_SEQUENCE, "")
		case 'u':
			// NumericEscape
			if reader.PeekN(2) == openBrace {
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

func (l *lexer) processIdentifierOrKeyword() st.STToken {
	l.processUnquotedIdentifier()
	tokenText := l.getLexeme()
	switch tokenText {
	case intKeyword:
		return l.getSyntaxToken(st.INT_KEYWORD)
	case float:
		return l.getSyntaxToken(st.FLOAT_KEYWORD)
	case stringKeyword:
		return l.getSyntaxToken(st.STRING_KEYWORD)
	case boolean:
		return l.getSyntaxToken(st.BOOLEAN_KEYWORD)
	case decimal:
		return l.getSyntaxToken(st.DECIMAL_KEYWORD)
	case xml:
		return l.getSyntaxToken(st.XML_KEYWORD)
	case jsonKeyword:
		return l.getSyntaxToken(st.JSON_KEYWORD)
	case handle:
		return l.getSyntaxToken(st.HANDLE_KEYWORD)
	case anyKeyword:
		return l.getSyntaxToken(st.ANY_KEYWORD)
	case anydata:
		return l.getSyntaxToken(st.ANYDATA_KEYWORD)
	case never:
		return l.getSyntaxToken(st.NEVER_KEYWORD)
	case byteKeyword:
		return l.getSyntaxToken(st.BYTE_KEYWORD)

	// Keywords
	case public:
		return l.getSyntaxToken(st.PUBLIC_KEYWORD)
	case private:
		return l.getSyntaxToken(st.PRIVATE_KEYWORD)
	case function:
		return l.getSyntaxToken(st.FUNCTION_KEYWORD)
	case returnKeyword:
		return l.getSyntaxToken(st.RETURN_KEYWORD)
	case returns:
		return l.getSyntaxToken(st.RETURNS_KEYWORD)
	case external:
		return l.getSyntaxToken(st.EXTERNAL_KEYWORD)
	case typeKeyword:
		return l.getSyntaxToken(st.TYPE_KEYWORD)
	case record:
		return l.getSyntaxToken(st.RECORD_KEYWORD)
	case object:
		return l.getSyntaxToken(st.OBJECT_KEYWORD)
	case remote:
		return l.getSyntaxToken(st.REMOTE_KEYWORD)
	case abstract:
		return l.getSyntaxToken(st.ABSTRACT_KEYWORD)
	case client:
		return l.getSyntaxToken(st.CLIENT_KEYWORD)
	case ifKeyword:
		return l.getSyntaxToken(st.IF_KEYWORD)
	case elseKeyword:
		return l.getSyntaxToken(st.ELSE_KEYWORD)
	case while:
		return l.getSyntaxToken(st.WHILE_KEYWORD)
	case trueKeyword:
		return l.getSyntaxToken(st.TRUE_KEYWORD)
	case falseKeyword:
		return l.getSyntaxToken(st.FALSE_KEYWORD)
	case check:
		return l.getSyntaxToken(st.CHECK_KEYWORD)
	case fail:
		return l.getSyntaxToken(st.FAIL_KEYWORD)
	case checkpanic:
		return l.getSyntaxToken(st.CHECKPANIC_KEYWORD)
	case continueKeyword:
		return l.getSyntaxToken(st.CONTINUE_KEYWORD)
	case breakKeyword:
		return l.getSyntaxToken(st.BREAK_KEYWORD)
	case panicKeyword:
		return l.getSyntaxToken(st.PANIC_KEYWORD)
	case importKeyword:
		return l.getSyntaxToken(st.IMPORT_KEYWORD)
	case as:
		return l.getSyntaxToken(st.AS_KEYWORD)
	case service:
		return l.getSyntaxToken(st.SERVICE_KEYWORD)
	case on:
		return l.getSyntaxToken(st.ON_KEYWORD)
	case resource:
		return l.getSyntaxToken(st.RESOURCE_KEYWORD)
	case listener:
		return l.getSyntaxToken(st.LISTENER_KEYWORD)
	case constKeyword:
		return l.getSyntaxToken(st.CONST_KEYWORD)
	case final:
		return l.getSyntaxToken(st.FINAL_KEYWORD)
	case typeof:
		return l.getSyntaxToken(st.TYPEOF_KEYWORD)
	case is:
		return l.getSyntaxToken(st.IS_KEYWORD)
	case null:
		return l.getSyntaxToken(st.NULL_KEYWORD)
	case lock:
		return l.getSyntaxToken(st.LOCK_KEYWORD)
	case annotation:
		return l.getSyntaxToken(st.ANNOTATION_KEYWORD)
	case source:
		return l.getSyntaxToken(st.SOURCE_KEYWORD)
	case varKeyword:
		return l.getSyntaxToken(st.VAR_KEYWORD)
	case worker:
		return l.getSyntaxToken(st.WORKER_KEYWORD)
	case parameter:
		return l.getSyntaxToken(st.PARAMETER_KEYWORD)
	case field:
		return l.getSyntaxToken(st.FIELD_KEYWORD)
	case isolated:
		return l.getSyntaxToken(st.ISOLATED_KEYWORD)
	case xmlns:
		return l.getSyntaxToken(st.XMLNS_KEYWORD)
	case fork:
		return l.getSyntaxToken(st.FORK_KEYWORD)
	case mapKeyword:
		return l.getSyntaxToken(st.MAP_KEYWORD)
	case future:
		return l.getSyntaxToken(st.FUTURE_KEYWORD)
	case typedesc:
		return l.getSyntaxToken(st.TYPEDESC_KEYWORD)
	case trap:
		return l.getSyntaxToken(st.TRAP_KEYWORD)
	case in:
		return l.getSyntaxToken(st.IN_KEYWORD)
	case foreach:
		return l.getSyntaxToken(st.FOREACH_KEYWORD)
	case table:
		return l.getSyntaxToken(st.TABLE_KEYWORD)
	case errorKeyword:
		return l.getSyntaxToken(st.ERROR_KEYWORD)
	case let:
		return l.getSyntaxToken(st.LET_KEYWORD)
	case stream:
		return l.getSyntaxToken(st.STREAM_KEYWORD)
	case newKeyword:
		return l.getSyntaxToken(st.NEW_KEYWORD)
	case readonly:
		return l.getSyntaxToken(st.READONLY_KEYWORD)
	case distinct:
		return l.getSyntaxToken(st.DISTINCT_KEYWORD)
	case from:
		return l.getSyntaxToken(st.FROM_KEYWORD)
	case start:
		return l.getSyntaxToken(st.START_KEYWORD)
	case flush:
		return l.getSyntaxToken(st.FLUSH_KEYWORD)
	case wait:
		return l.getSyntaxToken(st.WAIT_KEYWORD)
	case do:
		return l.getSyntaxToken(st.DO_KEYWORD)
	case transaction:
		return l.getSyntaxToken(st.TRANSACTION_KEYWORD)
	case commit:
		return l.getSyntaxToken(st.COMMIT_KEYWORD)
	case retry:
		return l.getSyntaxToken(st.RETRY_KEYWORD)
	case rollback:
		return l.getSyntaxToken(st.ROLLBACK_KEYWORD)
	case transactional:
		return l.getSyntaxToken(st.TRANSACTIONAL_KEYWORD)
	case enum:
		return l.getSyntaxToken(st.ENUM_KEYWORD)
	case base16Keyword:
		return l.getSyntaxToken(st.BASE16_KEYWORD)
	case base64Keyword:
		return l.getSyntaxToken(st.BASE64_KEYWORD)
	case match:
		return l.getSyntaxToken(st.MATCH_KEYWORD)
	case conflict:
		return l.getSyntaxToken(st.CONFLICT_KEYWORD)
	case class:
		return l.getSyntaxToken(st.CLASS_KEYWORD)
	case configurable:
		return l.getSyntaxToken(st.CONFIGURABLE_KEYWORD)
	case where:
		return l.getSyntaxToken(st.WHERE_KEYWORD)
	case selectKeyword:
		return l.getSyntaxToken(st.SELECT_KEYWORD)
	case limit:
		return l.getSyntaxToken(st.LIMIT_KEYWORD)
	case outer:
		return l.getSyntaxToken(st.OUTER_KEYWORD)
	case equals:
		return l.getSyntaxToken(st.EQUALS_KEYWORD)
	case order:
		return l.getSyntaxToken(st.ORDER_KEYWORD)
	case by:
		return l.getSyntaxToken(st.BY_KEYWORD)
	case ascending:
		return l.getSyntaxToken(st.ASCENDING_KEYWORD)
	case descending:
		return l.getSyntaxToken(st.DESCENDING_KEYWORD)
	case join:
		return l.getSyntaxToken(st.JOIN_KEYWORD)
	case re:
		if l.getNextNonWhiteSpaceOrNonCommentChar() == backtick {
			return l.getSyntaxToken(st.RE_KEYWORD)
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
		case space, tab, formFeed, carriageReturn, newline:
			lookaheadCount++
		case slash:
			if reader.PeekN(lookaheadCount+1) == slash {
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
		case newline:
		case carriageReturn:
		default:
			lookaheadCount += 1
			nextChar = reader.PeekN(lookaheadCount)
			continue
		}
		break
	}
	return lookaheadCount
}

func (l *lexer) processNumericLiteral(startChar rune) st.STToken {
	reader := l.reader
	nextChar := reader.Peek()
	if l.isHexIndicator(startChar, nextChar) {
		return l.processHexLiteral()
	}

	len := 1
	for !reader.IsEOF() {
		switch nextChar {
		case dot, 'e', 'E', 'f', 'F', 'd', 'D':
			nextNextChar := reader.PeekN(1)
			if nextChar == dot &&
				(nextNextChar == dot || l.isDecimalNumberFollowedIdentifier()) {
				// This is to handle two cases:
				// 1. More than one dot. e.g. 2...10
				// 2. Method call. e.g. 2.toString()
				break
			}

			// In sem-var mode, only decimal integer literals are supported
			if l.context.mode == parserModeImportMode {
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

	return l.getLiteral(st.DECIMAL_INTEGER_LITERAL_TOKEN)
}

func (l *lexer) getLiteral(kind st.SyntaxKind) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	trailingTrivia := l.processTrailingTrivia()
	return st.CreateLiteralValueToken(kind, lexeme, leadingTrivia, trailingTrivia)
}

func (l *lexer) processDecimalFloatLiteral() st.STToken {
	reader := l.reader
	nextChar := reader.Peek()

	// For float literals start with a DOT, this condition will always be false,
	// as the reader is already advanced for the DOT before coming here.
	if nextChar == dot {
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
		return l.getLiteral(st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
	}
}

func (l *lexer) parseFloatingPointTypeSuffix() st.STToken {
	l.reader.Advance()
	return l.getLiteral(st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
}

func (l *lexer) processExponent(isHex bool) st.STToken {
	// Advance reader as exponent indicator is already validated
	reader := l.reader
	reader.Advance()
	nextChar := reader.Peek()

	// Capture if there is a sign
	if nextChar == plus || nextChar == minus {
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
		return l.getLiteral(st.HEX_FLOATING_POINT_LITERAL_TOKEN)
	}

	switch nextChar {
	case 'f', 'F', 'd', 'D':
		return l.parseFloatingPointTypeSuffix()
	default:
		return l.getLiteral(st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
	}
}

func (l *lexer) reportLexerError(errorCode common.DiagnosticErrorCode, args ...any) {
	diagnostic := st.CreateDiagnostic(&errorCode, args...)
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
		if lookaheadChar == plus || lookaheadChar == minus {
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
		if lookaheadChar == plus || lookaheadChar == minus {
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

func (l *lexer) processHexLiteral() st.STToken {
	reader := l.reader
	reader.Advance() // advance for "x" or "X"
	containsHexDigit := false

	for isHexDigit(byte(reader.Peek())) {
		reader.Advance()
		containsHexDigit = true
	}

	nextChar := reader.Peek()
	switch nextChar {
	case dot:
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

	return l.getLiteral(st.HEX_FLOATING_POINT_LITERAL_TOKEN)
}

func (l *lexer) getHexIntegerLiteral() st.STToken {
	lexeme := l.getLexeme()
	if lexeme == "0x" || lexeme == "0X" {
		l.reportLexerError(common.ERROR_MISSING_HEX_NUMBER_AFTER_HEX_INDICATOR)
	}

	return l.getLiteral(st.HEX_INTEGER_LITERAL_TOKEN)
}

func (l *lexer) isHexIndicator(startChar rune, nextChar rune) bool {
	return startChar == '0' && (nextChar == 'x' || nextChar == 'X')
}

func (l *lexer) processQuotedIdentifier() st.STToken {
	l.processIdentifierEnd()
	if string(singleQuote) == l.getLexeme() {
		l.reportLexerError(common.ERROR_INCOMPLETE_QUOTED_IDENTIFIER)
	}
	return l.getIdentifierToken()
}

func (l *lexer) getBacktickToken() st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	// Trivia after the back-tick including whitespace belongs to the content of the back-tick.
	// Therefore, do not process trailing trivia for starting back-tick. We reach here only for
	// starting back-tick. Ending back-tick is processed by the template mode.
	trailingTrivia := st.CreateEmptyNodeList()
	return st.CreateTokenFrom(st.BACKTICK_TOKEN, leadingTrivia, trailingTrivia)
}

func (l *lexer) processExclamationMarkOperator() st.STToken {
	reader := l.reader
	switch reader.Peek() {
	case equal:
		reader.Advance()
		if reader.Peek() == equal {
			// this is '!=='
			reader.Advance()
			return l.getSyntaxToken(st.NOT_DOUBLE_EQUAL_TOKEN)
		}
		// this is '!='
		return l.getSyntaxToken(st.NOT_EQUAL_TOKEN)
	default:
		// this is '!is'
		if l.isNotIsToken() {
			reader.AdvanceN(2)
			return l.getSyntaxToken(st.NOT_IS_KEYWORD)
		}
		// this is '!'
		return l.getSyntaxToken(st.EXCLAMATION_MARK_TOKEN)
	}
}

func (l *lexer) isNotIsToken() bool {
	reader := l.reader
	return (reader.Peek() == 'i' && reader.PeekN(1) == 's') &&
		(!isIdentifierFollowingChar(reader.PeekN(2)) && reader.PeekN(2) != backslash)
}

func (l *lexer) processTokenStartWithGt() st.STToken {
	reader := l.reader
	if reader.Peek() == equal {
		reader.Advance()
		return l.getSyntaxToken(st.GT_EQUAL_TOKEN)
	}

	if reader.Peek() != gt {
		return l.getSyntaxToken(st.GT_TOKEN)
	}

	nextChar := reader.PeekN(1)
	switch nextChar {
	case gt:
		if reader.PeekN(2) == equal {
			// ">>>="
			reader.AdvanceN(2)
			return l.getSyntaxToken(st.TRIPPLE_GT_TOKEN)
		}
		return l.getSyntaxToken(st.GT_TOKEN)
	case equal:
		// ">>="
		reader.AdvanceN(1)
		return l.getSyntaxToken(st.DOUBLE_GT_TOKEN)
	default:
		return l.getSyntaxToken(st.GT_TOKEN)
	}
}

func (l *lexer) processTokenStartWithLt() st.STToken {
	reader := l.reader
	switch reader.Peek() {
	case equal:
		reader.Advance()
		return l.getSyntaxToken(st.LT_EQUAL_TOKEN)
	case minus:
		nextNextChar := reader.PeekN(1)
		if unicode.IsDigit(nextNextChar) {
			return l.getSyntaxToken(st.LT_TOKEN)
		}
		reader.Advance()
		return l.getSyntaxToken(st.LEFT_ARROW_TOKEN)
	case lt:
		reader.Advance()
		return l.getSyntaxToken(st.DOUBLE_LT_TOKEN)
	default:
		return l.getSyntaxToken(st.LT_TOKEN)
	}
}

func (l *lexer) processSlashToken() st.STToken {
	// check for the second char
	reader := l.reader
	if reader.Peek() != asterisk {
		return l.getSyntaxToken(st.SLASH_TOKEN)
	}

	reader.Advance()
	if reader.Peek() != asterisk {
		return l.getSyntaxToken(st.SLASH_ASTERISK_TOKEN)
	} else if reader.PeekN(1) == slash && reader.PeekN(2) == lt {
		reader.AdvanceN(3)
		return l.getSyntaxToken(st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN)
	} else {
		return l.getSyntaxToken(st.SLASH_ASTERISK_TOKEN)
	}
}

func (l *lexer) processEqualOperator() st.STToken {
	reader := l.reader
	switch reader.Peek() {
	case equal:
		reader.Advance()
		if reader.Peek() == equal {
			// this is '==='
			reader.Advance()
			return l.getSyntaxToken(st.TRIPPLE_EQUAL_TOKEN)
		}
		// this is '=='
		return l.getSyntaxToken(st.DOUBLE_EQUAL_TOKEN)
	case gt:
		// this is '=>'
		reader.Advance()
		return l.getSyntaxToken(st.RIGHT_DOUBLE_ARROW_TOKEN)
	default:
		// this is '='
		return l.getSyntaxToken(st.EQUAL_TOKEN)
	}
}

func (l *lexer) processDocumentationString() st.STToken {
	reader := l.reader
	nextChar := reader.Peek()
	for !reader.IsEOF() {
		switch nextChar {
		case carriageReturn, newline:
			// Advance reader for the new line
			if reader.Peek() == carriageReturn && reader.PeekN(1) == newline {
				reader.Advance()
			}
			reader.Advance()

			// Look ahead and see if next line also belongs to the documentation.
			// i.e. look for a `WS #` match
			// If there's a match, advance reader for the next line as well.
			// Otherwise terminate documentation content after the new line.
			lookAheadCount := 0
			lookAheadChar := reader.PeekN(lookAheadCount)
			for lookAheadChar == space || lookAheadChar == tab {
				lookAheadCount++
				lookAheadChar = reader.PeekN(lookAheadCount)
			}

			if lookAheadChar != hash {
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
	trailingTrivia := st.CreateEmptyNodeList() // No trailing trivia
	return st.CreateLiteralValueToken(st.DOCUMENTATION_STRING, lexeme, leadingTrivia, trailingTrivia)
}

func (l *lexer) processStringLiteral() st.STToken {
	reader := l.reader
	var nextChar rune
	for !reader.IsEOF() {
		nextChar = reader.Peek()
		switch nextChar {
		case newline, carriageReturn:
			l.reportLexerError(common.ERROR_MISSING_DOUBLE_QUOTE)
		case doubleQuote:
			reader.Advance()
		case backslash:
			switch reader.PeekN(1) {
			case 't', 'n', 'r', backslash, doubleQuote:
				reader.AdvanceN(2)
				continue
			case 'u':
				if reader.PeekN(2) == openBrace {
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

	return l.getLiteral(st.STRING_LITERAL_TOKEN)
}

func (l *lexer) processPipeOperator() st.STToken {
	reader := l.reader
	switch reader.Peek() {
	case closeBrace:
		reader.Advance()
		return l.getSyntaxToken(st.CLOSE_BRACE_PIPE_TOKEN)
	case pipe:
		reader.Advance()
		return l.getSyntaxToken(st.LOGICAL_OR_TOKEN)
	default:
		return l.getSyntaxToken(st.PIPE_TOKEN)
	}
}

func (l *lexer) processDot() st.STToken {
	reader := l.reader
	nextChar := reader.Peek()
	switch nextChar {
	case dot:
		nextNextChar := reader.PeekN(1)
		switch nextNextChar {
		case dot:
			reader.AdvanceN(2)
			return l.getSyntaxToken(st.ELLIPSIS_TOKEN)
		case lt:
			reader.AdvanceN(2)
			return l.getSyntaxToken(st.DOUBLE_DOT_LT_TOKEN)
		}
	case at:
		reader.Advance()
		return l.getSyntaxToken(st.ANNOT_CHAINING_TOKEN)
	case lt:
		reader.Advance()
		return l.getSyntaxToken(st.DOT_LT_TOKEN)
	}

	if l.context.mode != parserModeImportMode && unicode.IsDigit(nextChar) {
		return l.processDecimalFloatLiteral()
	}
	return l.getSyntaxToken(st.DOT_TOKEN)
}

func (l *lexer) getIdentifierToken() st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	lexeme := l.getLexeme()
	trailingTrivia := l.processTrailingTrivia()
	return st.CreateIdentifierToken(lexeme, leadingTrivia, trailingTrivia)
}

func (l *lexer) processUnquotedIdentifier() {
	l.processIdentifierEnd()
}

func (l *lexer) getSyntaxToken(kind st.SyntaxKind) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := l.processTrailingTrivia()
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *lexer) getLeadingTrivia() st.STNode {
	trivia := st.CreateNodeList(l.context.leadingTriviaList...)
	l.context.leadingTriviaList = make([]st.STNode, 0, initialTriviaCapacity)
	return trivia
}

func (l *lexer) processTrailingTrivia() st.STNode {
	triviaList := make([]st.STNode, 0, initialTriviaCapacity)
	l.processSyntaxTrivia(&triviaList, false)
	result := st.CreateNodeList(triviaList...)
	return result
}

func (l *lexer) processSyntaxTrivia(triviaList *[]st.STNode, isLeading bool) {
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
				break
			}
			return
		case slash:
			if reader.PeekN(1) == slash {
				*triviaList = append(*triviaList, l.processComment())
				break
			}
			return
		default:
			return
		}
	}
}

func (l *lexer) processComment() st.STNode {
	reader := l.reader
	reader.AdvanceN(2)
	nextToken := reader.Peek()
	for !reader.IsEOF() {
		switch nextToken {
		case newline, carriageReturn:
		default:
			reader.Advance()
			nextToken = reader.Peek()
			continue
		}
		break
	}
	return st.CreateMinutiae(st.COMMENT_MINUTIAE, l.getLexeme())
}

func (l *lexer) processEndOfLine() st.STNode {
	reader := l.reader
	c := reader.Peek()
	switch c {
	case newline:
		reader.Advance()
		return st.CreateMinutiae(st.END_OF_LINE_MINUTIAE, l.getLexeme())
	case carriageReturn:
		reader.Advance()
		if reader.Peek() == newline {
			reader.Advance()
		}
		return st.CreateMinutiae(st.END_OF_LINE_MINUTIAE, l.getLexeme())
	default:
		l.internalError("unreachable")
		return st.CreateEmptyNode()
	}
}

func (l *lexer) processWhitespaces() st.STNode {
	reader := l.reader
	for !reader.IsEOF() {
		c := reader.Peek()
		switch c {
		case space, tab, formFeed:
			reader.Advance()
			continue
		default:
		}
		break
	}
	return st.CreateMinutiae(st.WHITESPACE_MINUTIAE, l.getLexeme())
}

func (l *lexer) readTokenInBracedContentInInterpolation() st.STToken {
	reader := l.reader
	reader.Mark()
	nextChar := reader.Peek()
	switch nextChar {
	case openBrace:
		l.StartMode(parserModeInterpolationBracedContent)
	case closeBrace:
		l.EndMode()
	case backtick:
		// Recursively end backtick string related contexts
		for l.context.mode != parserModeDefaultMode {
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

func (l *lexer) readTokenInInterpolation() st.STToken {
	reader := l.reader
	reader.Mark()
	nextChar := reader.Peek()
	switch nextChar {
	case openBrace:
		// Start braced-content mode. This is to keep track of the
		// open-brace and the corresponding close-brace. This way,
		// those will not be mistaken as the close-brace of the
		// interpolation end.
		l.StartMode(parserModeInterpolationBracedContent)
		return l.readToken()
	case closeBrace:
		// Close-brace in the interpolation mode definitely means its
		// then end of the interpolation.
		l.EndMode()
		reader.Advance()
		return l.getSyntaxTokenWithoutTrailingTrivia(st.CLOSE_BRACE_TOKEN)
	case backtick:
		// If we are inside the interpolation, that means its no longer XML
		// mode, but in the default mode. Hence treat the back-tick in the
		// same way as in the default mode.
		fallthrough
	default:
		// Otherwise read the token from default mode.
		return l.readToken()
	}
}

func (l *lexer) getSyntaxTokenWithoutTrailingTrivia(kind st.SyntaxKind) st.STToken {
	leadingTrivia := l.getLeadingTrivia()
	trailingTrivia := st.CreateEmptyNodeList()
	return st.CreateTokenFrom(kind, leadingTrivia, trailingTrivia)
}

func (l *lexer) processLeadingTrivia() {
	l.processSyntaxTrivia(&l.context.leadingTriviaList, true)
}

func (l *lexer) readRegExpTemplateToken() st.STToken {
	reader := l.reader
	shouldProcessInterpolations := true
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	switch nextChar {
	case backtick:
		reader.Advance()
		l.EndMode()
		return l.getSyntaxToken(st.BACKTICK_TOKEN)
	case dollar:
		if reader.PeekN(1) == openBrace {
			// Switch to interpolation mode. Then the next token will be read in that mode.
			l.StartMode(parserModeInterpolation)
			reader.AdvanceN(2)

			return l.getSyntaxToken(st.INTERPOLATION_START_TOKEN)
		}
		fallthrough
	default:
		if nextChar == openBracket {
			shouldProcessInterpolations = false
		}
		for !reader.IsEOF() {
			if shouldProcessInterpolations && reader.Peek() == backslash &&
				reader.PeekN(1) == openBracket {
				// Escaped open brackets are not considered as the start of a no interpolation context.
				reader.Advance()
			}
			reader.Advance()
			nextChar = reader.Peek()
			switch nextChar {
			case dollar:
				if shouldProcessInterpolations && reader.PeekN(1) == openBrace {
					break
				}
				continue
			case backtick:
			case openBracket:
				shouldProcessInterpolations = false
				continue
			case closeBracket:
				shouldProcessInterpolations = true
				continue
			case backslash:
				if !shouldProcessInterpolations && reader.PeekN(1) == closeBracket {
					reader.Advance()
				}
				continue
			default:
				continue
			}
			break
		}
	}
	return l.getLiteral(st.TEMPLATE_STRING)
}

func (l *lexer) readPromptToken() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	if nextChar == closeBrace {
		reader.Advance()
		l.EndMode()
		return l.getSyntaxToken(st.CLOSE_BRACE_TOKEN)
	}

	if nextChar == dollar && reader.PeekN(1) == openBrace {
		// Switch to interpolation mode. Then the next token will be read in that mode.
		l.StartMode(parserModeInterpolation)
		reader.AdvanceN(2)

		return l.getSyntaxToken(st.INTERPOLATION_START_TOKEN)
	}

	for !reader.IsEOF() {
		reader.Advance()
		nextChar = reader.Peek()
		nextNextChar := reader.PeekN(1)
		if nextChar == closeBrace ||
			(nextChar == dollar && nextNextChar == openBrace) {
			break
		}

		if nextChar == backslash {
			if nextNextChar != closeBrace && nextNextChar != backslash {
				l.reportInvalidEscapeSequence(reader.PeekN(1))
			}
			reader.Advance()
		}
	}

	return l.getLiteral(st.PROMPT_CONTENT)
}

func (l *lexer) readTemplateToken() st.STToken {
	reader := l.reader
	reader.Mark()
	if reader.IsEOF() {
		return l.getSyntaxToken(st.EOF_TOKEN)
	}

	nextChar := reader.Peek()
	if nextChar == backtick {
		reader.Advance()
		l.EndMode()
		return l.getSyntaxToken(st.BACKTICK_TOKEN)
	}

	if nextChar == dollar && reader.PeekN(1) == openBrace {
		// Switch to interpolation mode. Then the next token will be read in that mode.
		l.StartMode(parserModeInterpolation)
		reader.AdvanceN(2)

		return l.getSyntaxToken(st.INTERPOLATION_START_TOKEN)
	}

	for !reader.IsEOF() {
		reader.Advance()
		nextChar = reader.Peek()
		if nextChar == backtick ||
			(nextChar == dollar && reader.PeekN(1) == openBrace) {
			break
		}
	}

	return l.getLiteral(st.TEMPLATE_STRING)
}

type parserMode uint8

const (
	// Ballerina Parser
	parserModeDefaultMode parserMode = iota
	parserModeImportMode
	parserModeTemplate
	parserModeInterpolation
	parserModeInterpolationBracedContent
	parserModeRegexp
	parserModePrompt

	// Documentation Parser
	parserModeDocLineStartHash
	parserModeDocLineDifferentiator
	parserModeDocInternal
	parserModeDocParameter
	parserModeDocReferenceType
	parserModeDocSingleBacktickContent
	parserModeDocDoubleBacktickContent
	parserModeDocTripleBacktickContent
	parserModeDocCodeRefEnd
	parserModeDocCodeLineStartHash

	// XML Parser
	parserModeXmlContent
	parserModeXmlElementStartTag
	parserModeXmlElementEndTag
	parserModeXmlText
	parserModeXmlAttributes
	parserModeXmlComment
	parserModeXmlPi
	parserModeXmlPiData
	parserModeXmlSingleQuotedString
	parserModeXmlDoubleQuotedString
	parserModeXmlCdataSection
)

const (
	public          = "public"
	private         = "private"
	function        = "function"
	returnKeyword   = "return"
	returns         = "returns"
	external        = "external"
	typeKeyword     = "type"
	record          = "record"
	object          = "object"
	remote          = "remote"
	abstract        = "abstract"
	client          = "client"
	ifKeyword       = "if"
	elseKeyword     = "else"
	while           = "while"
	panicKeyword    = "panic"
	trueKeyword     = "true"
	falseKeyword    = "false"
	check           = "check"
	fail            = "fail"
	checkpanic      = "checkpanic"
	continueKeyword = "continue"
	breakKeyword    = "break"
	importKeyword   = "import"
	as              = "as"
	on              = "on"
	resource        = "resource"
	listener        = "listener"
	constKeyword    = "const"
	final           = "final"
	typeof          = "typeof"
	is              = "is"
	null            = "null"
	lock            = "lock"
	annotation      = "annotation"
	source          = "source"
	worker          = "worker"
	parameter       = "parameter"
	field           = "field"
	isolated        = "isolated"
	xmlns           = "xmlns"
	fork            = "fork"
	trap            = "trap"
	in              = "in"
	foreach         = "foreach"
	table           = "table"
	key             = "key"
	errorKeyword    = "error"
	let             = "let"
	stream          = "stream"
	newKeyword      = "new"
	readonly        = "readonly"
	distinct        = "distinct"
	from            = "from"
	where           = "where"
	selectKeyword   = "select"
	start           = "start"
	flush           = "flush"
	wait            = "wait"
	do              = "do"
	transaction     = "transaction"
	transactional   = "transactional"
	commit          = "commit"
	retry           = "retry"
	rollback        = "rollback"
	enum            = "enum"
	base16Keyword   = "base16"
	base64Keyword   = "base64"
	match           = "match"
	conflict        = "conflict"
	limit           = "limit"
	join            = "join"
	outer           = "outer"
	equals          = "equals"
	order           = "order"
	by              = "by"
	ascending       = "ascending"
	descending      = "descending"
	class           = "class"
	configurable    = "configurable"
	natural         = "natural"

	// For BFM only
	variable = "variable"
	module   = "module"

	// Types
	intKeyword    = "int"
	float         = "float"
	stringKeyword = "string"
	boolean       = "boolean"
	decimal       = "decimal"
	xml           = "xml"
	jsonKeyword   = "json"
	handle        = "handle"
	anyKeyword    = "any"
	anydata       = "anydata"
	service       = "service"
	varKeyword    = "var"
	never         = "never"
	mapKeyword    = "map"
	future        = "future"
	typedesc      = "typedesc"
	byteKeyword   = "byte"
	// Separators
	semicolon        = ';'
	colon            = ':'
	dot              = '.'
	comma            = ','
	openParanthesis  = '('
	closeParanthesis = ')'
	openBrace        = '{'
	closeBrace       = '}'
	openBracket      = '['
	closeBracket     = ']'
	pipe             = '|'
	questionMark     = '?'
	doubleQuote      = '"'
	singleQuote      = '\''
	hash             = '#'
	at               = '@'
	backtick         = '`'
	dollar           = '$'

	// Arithmetic opera
	equal           = '='
	plus            = '+'
	minus           = '-'
	asterisk        = '*'
	slash           = '/'
	percent         = '%'
	gt              = '>'
	lt              = '<'
	backslash       = '\\'
	exclamationMark = '!'
	bitwiseAnd      = '&'
	bitwiseXor      = '^'
	negation        = '~'

	// Other
	newline        = '\n' // equivalent to 0xA
	carriageReturn = '\r' // equivalent to 0xD
	tab            = 0x9
	space          = 0x20
	formFeed       = 0xC

	re = "re"
)
