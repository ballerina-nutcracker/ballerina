/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package parser

import (
	"fmt"

	"ballerina/common/tomlparser/internal/lexer"
	"ballerina/tools/diagnostics"
)

// ParseError is a semantic or syntactic error produced during parsing.
// Exposed to the public API via toml.go's convertParseErrors helper.
type ParseError struct {
	Message string
	Line    int
	Column  int
	EndLine int
	EndCol  int
}

// addError records a diagnostic on the parser.
func (p *Parser) addError(msg string, tok lexer.Token) {
	p.diagnostics = append(p.diagnostics, ParseError{
		Message: msg,
		Line:    tok.Line,
		Column:  tok.Column,
		EndLine: tok.Line,
		EndCol:  tok.Column + len([]rune(tok.Value)),
	})
}

// addErrorAt records a diagnostic at an explicit position.
func (p *Parser) addErrorAt(msg string, line, col int) {
	p.diagnostics = append(p.diagnostics, ParseError{
		Message: msg,
		Line:    line,
		Column:  col,
		EndLine: line,
		EndCol:  col,
	})
}

// addErrorAtLoc records a diagnostic at a position from a diagnostics.Location,
// resolving byte offsets to line/column using the parser's DiagnosticContext.
func (p *Parser) addErrorAtLoc(msg string, loc diagnostics.Location) {
	line := p.de.StartLine(loc) + 1
	col := p.de.StartColumn(loc) + 1
	p.addErrorAt(msg, line, col)
}

// expectToken asserts the next token has the expected kind.  If it does not,
// an error is recorded and the bad token is consumed so parsing can proceed.
// Returns the consumed token (may be of the wrong kind if recovery occurred).
func (p *Parser) expectToken(kind lexer.TokenKind) lexer.Token {
	tok := p.reader.peek()
	if tok.Kind == kind {
		return p.reader.read()
	}
	p.addError(fmt.Sprintf("expected %v but got %v (%q)", tokenKindName(kind), tokenKindName(tok.Kind), tok.Value), tok)
	// Consume the bad token for recovery.
	return p.reader.read()
}

// skipToRecovery skips tokens until a safe re-synchronisation point.
// Recovery points: EOF, newline, or open-bracket (start of next table header).
func (p *Parser) skipToRecovery() {
	for {
		tok := p.reader.peek()
		switch tok.Kind {
		case lexer.TokenEOF,
			lexer.TokenNewline,
			lexer.TokenOpenBracket:
			return
		}
		p.reader.read()
	}
}

// tokenKindName returns a human-readable name for a token kind.
func tokenKindName(k lexer.TokenKind) string {
	switch k {
	case lexer.TokenEOF:
		return "EOF"
	case lexer.TokenNewline:
		return "newline"
	case lexer.TokenOpenBracket:
		return "'['"
	case lexer.TokenCloseBracket:
		return "']'"
	case lexer.TokenOpenBrace:
		return "'{'"
	case lexer.TokenCloseBrace:
		return "'}'"
	case lexer.TokenEqual:
		return "'='"
	case lexer.TokenDot:
		return "'.'"
	case lexer.TokenComma:
		return "','"
	case lexer.TokenDoubleQuote:
		return "'\"'"
	case lexer.TokenTripleDoubleQuote:
		return "'\"\"\"'"
	case lexer.TokenSingleQuote:
		return "\"'\""
	case lexer.TokenTripleSingleQuote:
		return "\"'''\""
	case lexer.TokenIdentifier:
		return "identifier"
	case lexer.TokenDecimalInt:
		return "integer"
	case lexer.TokenDecimalFloat:
		return "float"
	case lexer.TokenTrue:
		return "'true'"
	case lexer.TokenFalse:
		return "'false'"
	default:
		return fmt.Sprintf("token(%d)", k)
	}
}
