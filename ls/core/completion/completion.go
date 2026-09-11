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
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Package completion implements lexical completion against sealed compiler
// generations.
package completion

import (
	stdcontext "context"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

type Request struct {
	URI    uri.DocumentURI
	Text   string
	Offset int
}

type Service struct {
	compiler *compile.CompilationService
}

func New(compiler *compile.CompilationService) *Service {
	return &Service{compiler: compiler}
}

func (s *Service) Complete(ctx stdcontext.Context, req Request) ([]protocol.CompletionItem, error) {
	if req.Offset < 0 {
		req.Offset = 0
	}
	if req.Offset > len(req.Text) {
		req.Offset = len(req.Text)
	}
	sm, ok := s.compiler.SealedModuleFor(ctx, req.URI)
	if !ok {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return emptyItems, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return completeAt(req, sm), nil
}

func completeAt(req Request, sm compile.SealedModule) (items []protocol.CompletionItem) {
	defer func() {
		if recover() != nil {
			items = emptyItems
		}
	}()
	if sm.PackageNode() == nil || sm.Context() == nil || sm.Stage() < compile.StageSymbolResolved {
		return emptyItems
	}
	cursor := newCursor(req, sm)
	switch cursor.kind {
	case kindModule:
		return lexicalItems(cursor, moduleKeywords)
	case kindBlock:
		return lexicalItems(cursor, blockKeywords)
	case kindLexical:
		return lexicalItems(cursor, nil)
	default:
		return emptyItems
	}
}

var emptyItems = []protocol.CompletionItem{}
