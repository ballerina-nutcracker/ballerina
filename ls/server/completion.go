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
	"errors"
	"regexp"

	"github.com/ballerina-nutcracker/ballerina/ls/core/completion"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

// Runs on the Serve goroutine so the request reads the text it was sent against,
// not text from later didChange notifications.
func (s *Server) prepareCompletion(message protocol.Message) func(context.Context) trackedResult {
	var params protocol.CompletionParams
	if json.Unmarshal(message.Params, &params) != nil {
		return replyWith(trackedResult{err: &protocol.RPCError{Code: rpcInvalidParams, Message: "invalid completion parameters"}, handled: true})
	}
	empty := replyWith(trackedResult{result: []protocol.CompletionItem{}, handled: true})
	docURI, err := workspace.NewFileURI(params.TextDocument.URI)
	if err != nil {
		return empty
	}
	text, ok := s.projects.DocumentText(docURI)
	if !ok {
		return empty
	}
	offset, _ := utf16Offset(text, params.Position, true)
	return func(ctx context.Context) trackedResult {
		return s.handleCompletion(ctx, completion.Request{URI: docURI, Text: text, Offset: offset})
	}
}

func (s *Server) handleCompletion(ctx context.Context, req completion.Request) trackedResult {
	items, err := s.completion.Complete(ctx, req)
	if errors.Is(err, completion.ErrStaleText) {
		return trackedResult{result: protocol.CompletionList{IsIncomplete: true, Items: []protocol.CompletionItem{}}, handled: true}
	}
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
