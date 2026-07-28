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

import "ballerina/common/tomlparser/internal/lexer"

const tokenBufferCap = 20

// tokenReader wraps a Lexer with a lookahead ring buffer.
type tokenReader struct {
	lex   *lexer.Lexer
	buf   [tokenBufferCap]lexer.Token
	start int // ring-buffer read head
	size  int // number of tokens currently buffered
}

// newTokenReader creates a tokenReader backed by lex.
func newTokenReader(lex *lexer.Lexer) *tokenReader {
	return &tokenReader{lex: lex}
}

// read consumes and returns the next token.
func (r *tokenReader) read() lexer.Token {
	if r.size > 0 {
		tok := r.buf[r.start]
		r.start = (r.start + 1) % tokenBufferCap
		r.size--
		return tok
	}
	return r.lex.NextToken()
}

// peek returns the next token without consuming it (1-ahead).
func (r *tokenReader) peek() lexer.Token {
	return r.peekK(1)
}

// peekK returns the token k positions ahead (1-indexed) without consuming.
// tokenBufferCap (20) is intentionally larger than any lookahead depth needed
// for Ballerina.toml files (deepest real key: 2 dotted segments). A loop
// caller hitting k > 20 would require 11+ dotted segments, which is
// unreachable in any valid Ballerina manifest.
func (r *tokenReader) peekK(k int) lexer.Token {
	// Fill buffer until we have k tokens.
	for r.size < k {
		if r.size >= tokenBufferCap {
			// Buffer full — return last token in buffer as best-effort.
			break
		}
		tok := r.lex.NextToken()
		idx := (r.start + r.size) % tokenBufferCap
		r.buf[idx] = tok
		r.size++
	}
	if k > r.size {
		k = r.size
	}
	if k <= 0 {
		return lexer.Token{Kind: lexer.TokenEOF}
	}
	return r.buf[(r.start+k-1)%tokenBufferCap]
}
