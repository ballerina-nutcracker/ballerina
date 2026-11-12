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
	"errors"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

// Request identifies a completion position; Offset is a byte offset in Text.
type Request struct {
	URI    workspace.DocumentURI
	Text   string
	Offset int
}

// AvailablePackage identifies a package offered for import completion.
type AvailablePackage struct {
	Organization string
	Name         string
}

// ErrStaleText reports that the sealed module was parsed from text newer than the request text.
var ErrStaleText = errors.New("completion request text is stale")

// Option configures a completion service.
type Option func(*Service)

// WithAvailablePackages sets the packages offered for import completion.
func WithAvailablePackages(packages []AvailablePackage) Option {
	return func(s *Service) {
		s.packages = append([]AvailablePackage(nil), packages...)
	}
}

// Service provides completion candidates from sealed compiler modules.
type Service struct {
	compiler *compile.CompilationService
	packages []AvailablePackage
}

// New creates a completion service with the given compiler and options.
func New(compiler *compile.CompilationService, options ...Option) *Service {
	service := &Service{compiler: compiler}
	for _, option := range options {
		option(service)
	}
	return service
}

// Complete returns candidates at req.Offset, clamped to the bounds of req.Text.
// It returns ErrStaleText when the sealed module was parsed from text other than req.Text.
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
	text, ok := sm.SourceText(req.URI)
	if !ok {
		return emptyItems, nil
	}
	if text != req.Text {
		return nil, ErrStaleText
	}
	return completeAt(req, sm, s.packages), nil
}

func completeAt(req Request, sm compile.SealedModule, packages []AvailablePackage) (items []protocol.CompletionItem) {
	defer func() {
		if recover() != nil {
			items = emptyItems
		}
	}()
	if sm.PackageNode() == nil || sm.Context() == nil || sm.Stage() < compile.StageSymbolResolved {
		return emptyItems
	}
	return dispatch(newCursor(req, sm, packages))
}

var emptyItems = []protocol.CompletionItem{}
