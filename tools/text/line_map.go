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

import (
	"ballerina/common/errors"
)

// LineMap represents a collection of text lines in the TextDocument.
type LineMap interface {
	LinePositionFromPosition(position int) (line, offset int, err error)
}

type lineMapImpl struct {
	textLines []TextLine
	length    int
}

func NewLineMap(textLines []TextLine) LineMap {
	return &lineMapImpl{
		textLines: textLines,
		length:    len(textLines),
	}
}

func (lm lineMapImpl) LinePositionFromPosition(position int) (line, offset int, err error) {
	if err := lm.positionRangeCheck(position); err != nil {
		return 0, 0, err
	}
	textLine := lm.findLineFrom(position)
	offset = position - textLine.StartOffset()
	if l := textLine.Length(); offset > l {
		offset = l
	}
	return textLine.LineNo(), offset, nil
}

func (lm lineMapImpl) positionRangeCheck(position int) error {
	if position < 0 || position > lm.textLines[lm.length-1].EndOffset() {
		return errors.NewIndexOutOfBoundsError(position, lm.textLines[lm.length-1].EndOffset())
	}
	return nil
}

// findLineFrom returns the TextLine to which the given position belongs.
// Performs a binary search to find the matching text line.
func (lm lineMapImpl) findLineFrom(position int) TextLine {
	// Check boundary conditions
	if position == 0 {
		return lm.textLines[0]
	} else if position == lm.textLines[lm.length-1].EndOffset() {
		return lm.textLines[lm.length-1]
	}
	left := 0
	right := lm.length - 1
	for left <= right {
		lhs := left >> 1
		rhs := right >> 1
		middle := (lhs + rhs) + (left & right & 1)
		startOffset := lm.textLines[middle].StartOffset()
		endOffset := lm.textLines[middle].EndOffsetWithNewLines()
		if startOffset <= position && position < endOffset {
			return lm.textLines[middle]
		} else if endOffset <= position {
			left = middle + 1
		} else {
			right = middle - 1
		}
	}
	// This should never happen given the boundary checks above
	panic("binary search failed to find matching text line")
}
