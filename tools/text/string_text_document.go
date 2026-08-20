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

package text

const (
	CR = 13 // Carriage Return
	LF = 10 // Line Feed
)

// StringTextDocument represents a TextDocument created with a string.
type StringTextDocument = TextDocument

type stringTextDocumentImpl struct {
	text    string
	lineMap LineMap
}

func NewStringTextDocument(text string) StringTextDocument {
	return &stringTextDocumentImpl{
		text: text,
	}
}

func (std *stringTextDocumentImpl) String() string {
	return std.text
}

func (std *stringTextDocumentImpl) LinePositionFromTextPosition(textPosition int) (line, offset int, err error) {
	return std.lines().LinePositionFromPosition(textPosition)
}

func (std *stringTextDocumentImpl) lines() LineMap {
	if std.lineMap == nil {
		std.lineMap = NewLineMap(std.calculateTextLines())
	}
	return std.lineMap
}

func (std *stringTextDocumentImpl) calculateTextLines() []TextLine {
	var textLines []TextLine

	line := 0
	startOffset := 0

	index := 0
	textLength := len(std.text)

	var lengthOfNewLineChars int

	for index < textLength {
		if std.text[index] == CR || std.text[index] == LF {
			nextIndex := index + 1
			if std.text[index] == CR && nextIndex < textLength && std.text[nextIndex] == LF {
				lengthOfNewLineChars = 2
			} else {
				lengthOfNewLineChars = 1
			}

			endOffset := startOffset + (index - startOffset)
			textLines = append(textLines, NewTextLine(line, std.text[startOffset:index], startOffset, endOffset, lengthOfNewLineChars))

			line = line + 1
			startOffset = endOffset + lengthOfNewLineChars
			index = index + lengthOfNewLineChars
		} else {
			index = index + 1
		}
	}

	textLines = append(textLines, NewTextLine(line, std.text[startOffset:], startOffset, textLength, 0))

	return textLines
}
