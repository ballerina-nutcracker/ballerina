// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"encoding/json"
	"regexp"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/ls/core/completion"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func (s *Server) handleCompletion(ctx context.Context, message protocol.Message) trackedResult {
	var params protocol.CompletionParams
	if json.Unmarshal(message.Params, &params) != nil {
		return trackedResult{}
	}
	docURI, err := workspace.NewFileURI(params.TextDocument.URI)
	if err != nil {
		return trackedResult{}
	}
	text, ok := s.projects.DocumentText(docURI)
	if !ok {
		return trackedResult{result: []protocol.CompletionItem{}, handled: true}
	}
	offset := byteOffsetFromPosition(text, params.Position)
	items, err := s.completion.Complete(ctx, completion.Request{
		URI:    docURI,
		Text:   text,
		Offset: offset,
	})
	if err != nil {
		if ctx.Err() != nil {
			return trackedResult{err: &protocol.RPCError{Code: rpcRequestCancelled, Message: "request cancelled"}, handled: true}
		}
		return trackedResult{err: &protocol.RPCError{Code: rpcInternalError, Message: "completion failed"}, handled: true}
	}
	if !s.snippetSupport {
		downgradeSnippetItems(items)
	}
	return trackedResult{result: items, handled: true}
}

var (
	snippetPlaceholderWithText = regexp.MustCompile(`\$\{\d+:([^{}]*)\}`)
	snippetTabStop             = regexp.MustCompile(`\$\{\d+\}`)
)

// Only ${n:text} placeholders without braces in text and ${n} tab stops are handled.
func downgradeSnippetItems(items []protocol.CompletionItem) {
	for i := range items {
		item := &items[i]
		if format, ok := item.InsertTextFormat.Value(); !ok || format != protocol.InsertTextFormatSnippet {
			continue
		}
		if text, ok := item.InsertText.Value(); ok {
			text = snippetPlaceholderWithText.ReplaceAllString(text, "$1")
			item.InsertText = protocol.NewOptional(snippetTabStop.ReplaceAllString(text, ""))
		}
		item.InsertTextFormat = protocol.Optional[protocol.InsertTextFormat]{}
	}
}

// Positions clamp to the line or document end; a position within a surrogate
// pair resolves to the rune's starting byte.
func byteOffsetFromPosition(text string, position protocol.Position) int {
	line := 0
	lineStart := 0
	for i := 0; i < len(text) && line < int(position.Line); {
		switch text[i] {
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				i += 2
			} else {
				i++
			}
			line++
			lineStart = i
		case '\n':
			i++
			line++
			lineStart = i
		default:
			_, size := utf8.DecodeRuneInString(text[i:])
			if size == 0 {
				return i
			}
			i += size
		}
	}
	if line < int(position.Line) {
		return len(text)
	}

	character := 0
	for i := lineStart; i < len(text); {
		if text[i] == '\r' || text[i] == '\n' || character >= int(position.Character) {
			return i
		}
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == utf8.RuneError && size == 0 {
			return i
		}
		width := len(utf16.Encode([]rune{r}))
		if character+width > int(position.Character) {
			return i
		}
		character += width
		i += size
	}
	return len(text)
}
