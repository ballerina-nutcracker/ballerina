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
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

type Request struct {
	URI    workspace.DocumentURI
	Text   string
	Offset int
}

type AvailablePackage struct {
	Organization string
	Name         string
}

type Option func(*Service)

func WithAvailablePackages(packages []AvailablePackage) Option {
	return func(s *Service) {
		s.packages = append([]AvailablePackage(nil), packages...)
	}
}

// WithLogger injects the observability facade the service uses for the AST
// debug dump (see WithASTDebugLogging). The default (unset) is
// observability.NewNoop().
func WithLogger(logger *observability.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithASTDebugLogging turns on a per-request debug dump of the AST around
// the completion cursor: the chain's root node, pretty-printed, with the
// node dispatch() matched on bracketed by >>> <<<. This is a throwaway
// diagnostic aid for iterating on completion dispatch, not a shipped
// feature, so it's opt-in and off by default -- pair it with WithLogger to
// actually see output (a noop logger discards it).
func WithASTDebugLogging() Option {
	return func(s *Service) {
		s.debugAST = true
	}
}

type Service struct {
	compiler *compile.CompilationService
	packages []AvailablePackage
	logger   *observability.Logger
	debugAST bool
}

func New(compiler *compile.CompilationService, options ...Option) *Service {
	service := &Service{compiler: compiler, logger: observability.NewNoop()}
	for _, option := range options {
		option(service)
	}
	return service
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
	return completeAt(req, sm, s.packages, s.logger, s.debugAST), nil
}

func completeAt(req Request, sm compile.SealedModule, packages []AvailablePackage, logger *observability.Logger, debugAST bool) (items []protocol.CompletionItem) {
	defer func() {
		if recover() != nil {
			items = emptyItems
		}
	}()
	if sm.PackageNode() == nil || sm.Context() == nil || sm.Stage() < compile.StageLocalTypeResolved {
		return emptyItems
	}
	c := newCursor(req, sm, packages)
	if debugAST {
		dumpCursorAST(logger, c)
	}
	return dispatch(c)
}

var emptyItems = []protocol.CompletionItem{}
