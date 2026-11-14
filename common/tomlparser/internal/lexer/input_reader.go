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

package lexer

import "unicode/utf8"

// InputReader provides character-level lookahead over a UTF-8 source string.
// The source is converted to []rune once for O(1) indexed access.
type InputReader struct {
	runes    []rune // full source as rune slice for correct Unicode indexing
	pos      int    // current read position (next rune to consume)
	markPos  int    // position saved by Mark() for GetMarkedChars()
	line     int    // 1-based current line (updated on Read)
	col      int    // 1-based current column (updated on Read)
	bytePos  int    // current byte offset in the source string
	markByte int    // byte position saved by Mark()
}

// NewInputReader creates an InputReader from a source string.
func NewInputReader(source string) *InputReader {
	return &InputReader{
		runes: []rune(source),
		line:  1,
		col:   1,
	}
}

// IsEOF returns true when all runes have been consumed.
func (r *InputReader) IsEOF() bool {
	return r.pos >= len(r.runes)
}

// Peek returns the next rune without consuming it.
// Returns charEOF (-1) at end of input.
func (r *InputReader) Peek() rune {
	if r.pos >= len(r.runes) {
		return charEOF
	}
	return r.runes[r.pos]
}

// PeekAt returns the rune at position pos+k (0-indexed offset from current pos).
// Returns charEOF if out of bounds.
func (r *InputReader) PeekAt(k int) rune {
	idx := r.pos + k
	if idx >= len(r.runes) {
		return charEOF
	}
	return r.runes[idx]
}

// Advance consumes the next rune and updates line/column/byte offset tracking.
func (r *InputReader) Advance() {
	if r.pos >= len(r.runes) {
		return
	}
	c := r.runes[r.pos]
	r.bytePos += utf8.RuneLen(c)
	r.pos++
	if c == charNewline {
		r.line++
		r.col = 1
	} else {
		r.col++
	}
}

// AdvanceN consumes n runes.
func (r *InputReader) AdvanceN(n int) {
	for range n {
		r.Advance()
	}
}

// Mark saves the current position so that GetMarkedChars() can return the
// text consumed since the mark.
func (r *InputReader) Mark() {
	r.markPos = r.pos
	r.markByte = r.bytePos
}

// GetMarkedChars returns the text consumed since the last Mark() call.
func (r *InputReader) GetMarkedChars() string {
	if r.markPos >= r.pos {
		return ""
	}
	return string(r.runes[r.markPos:r.pos])
}

// Line returns the current 1-based line number (position of the NEXT rune to be read).
func (r *InputReader) Line() int {
	return r.line
}

// Col returns the current 1-based column (position of the NEXT rune to be read).
func (r *InputReader) Col() int {
	return r.col
}

// BytePos returns the current byte offset in the source string.
func (r *InputReader) BytePos() int {
	return r.bytePos
}
