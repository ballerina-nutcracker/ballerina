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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

// UTF-16 boundary: The helpers below resolve protocol.TextEdit ranges
// (UTF-16 code-unit positions) to byte offsets in the full text. The server
// does this before calling workspace.Apply with resolved full text, keeping
// ls/core protocol-free. These helpers stay in ls/server per the core-service
// seam design.

func applyChanges(text string, changes []protocol.TextDocumentContentChangeEvent) (string, bool) {
	if len(changes) == 0 {
		return "", false
	}
	for _, change := range changes {
		partial, ok := change.TextDocumentContentChangePartial()
		if !ok {
			return "", false
		}
		start, ok := utf16Offset(text, partial.Range.Start)
		if !ok {
			return "", false
		}
		end, ok := utf16Offset(text, partial.Range.End)
		if !ok || end < start {
			return "", false
		}
		if rangeLength, ok := partial.RangeLength.Value(); ok && utf16Length(text[start:end]) != rangeLength {
			return "", false
		}
		text = text[:start] + partial.Text + text[end:]
	}
	return text, true
}

func utf16Offset(text string, position protocol.Position) (int, bool) {
	lineStart := 0
	line := uint32(0)
	for line < position.Line {
		nextLineStart, ok := nextLineStart(text, lineStart)
		if !ok {
			return 0, false
		}
		lineStart = nextLineStart
		line++
	}
	if position.Character == 0 {
		return lineStart, true
	}
	lineEnd := lineStart
	for lineEnd < len(text) && text[lineEnd] != '\n' && text[lineEnd] != '\r' {
		_, size := utf8.DecodeRuneInString(text[lineEnd:])
		lineEnd += size
	}
	character := uint32(0)
	for offset := lineStart; offset < lineEnd; {
		r, size := utf8.DecodeRuneInString(text[offset:])
		width := uint32(1)
		if r >= 0x10000 {
			width = 2
		}
		if character+width > position.Character {
			return 0, false
		}
		character += width
		offset += size
		if character == position.Character {
			return offset, true
		}
	}
	if character == position.Character {
		return lineEnd, true
	}
	return 0, false
}

func nextLineStart(text string, start int) (int, bool) {
	if start >= len(text) {
		return start, true
	}
	for offset := start; offset < len(text); {
		switch text[offset] {
		case '\n':
			return offset + 1, true
		case '\r':
			if offset+1 < len(text) && text[offset+1] == '\n' {
				return offset + 2, true
			}
			return offset + 1, true
		default:
			_, size := utf8.DecodeRuneInString(text[offset:])
			offset += size
		}
	}
	return 0, false
}

func utf16Length(text string) uint32 {
	length := uint32(0)
	for _, r := range text {
		length++
		if r >= 0x10000 {
			length++
		}
	}
	return length
}
