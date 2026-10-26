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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

const (
	publishDiagnosticsMethod = "textDocument/publishDiagnostics"
	// Corpus synchronization point for out-of-band diagnostic notifications.
	flushMethod    = "$test/flush"
	cancelMethod   = "$/cancelRequest"
	setTraceMethod = "$/setTrace"
	// Corpus-only methods let tests control request completion and wait for replies.
	testBlockMethod        = "$test/blockRequest"
	testReleaseMethod      = "$test/releaseRequest"
	testFlushRequestMethod = "$test/flushRequests"
)

var (
	errServerExited          = errors.New("language server exited")
	errExitedWithoutShutdown = errors.New("language server exited without shutdown")
)

// Error codes from LSP (RequestCancelled) and JSON-RPC 2.0.
const (
	rpcRequestCancelled     = -32800
	rpcInvalidRequest       = -32600
	rpcServerNotInitialized = -32002
	rpcMethodNotFound       = -32601
	rpcInvalidParams        = -32602
	rpcParseError           = -32700
)

type requestEntry struct {
	cancel      context.CancelFunc
	replyOnce   sync.Once
	release     chan struct{}
	releaseOnce sync.Once
}

func (e *requestEntry) closeRelease() {
	e.releaseOnce.Do(func() { close(e.release) })
}

type requestRegistry struct {
	mu      sync.Mutex
	entries map[string]*requestEntry
}

func newRequestRegistry() *requestRegistry {
	return &requestRegistry{entries: make(map[string]*requestEntry)}
}

func (r *requestRegistry) register(id string, parent context.Context) (*requestEntry, context.Context, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[id]; ok {
		return nil, nil, false
	}
	ctx, cancel := context.WithCancel(parent)
	entry := &requestEntry{cancel: cancel}
	r.entries[id] = entry
	return entry, ctx, true
}

func (r *requestRegistry) lookup(id string) *requestEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.entries[id]
}

func (r *requestRegistry) unregister(id string) {
	r.mu.Lock()
	delete(r.entries, id)
	r.mu.Unlock()
}

func (r *requestRegistry) cancelAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entry := range r.entries {
		entry.cancel()
	}
}

// Server handles LSP messages and publishes workspace diagnostics.
type Server struct {
	transport      protocol.Transport
	projects       *workspace.ProjectService
	compiler       *compile.CompilationService
	bus            *event.Bus
	versionSupport bool
	initialized    bool
	shuttingDown   bool

	writeMu sync.Mutex // Prevents interleaved frames from concurrent response and diagnostic writes.

	ceDone        sync.WaitGroup // Diagnostic writes must finish before Flush returns.
	lastPubMu     sync.Mutex
	lastPublished map[string]uint64 // Root generations already claimed for diagnostic publication.

	registry  *requestRegistry
	requestWG sync.WaitGroup

	logger *observability.Logger
}

// Option configures a Server at construction time.
type Option func(*Server)

// WithLogger sets the logger used to derive request-scoped loggers.
// Logging is disabled by default; a nil logger leaves the current logger unchanged.
func WithLogger(logger *observability.Logger) Option {
	return func(s *Server) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// New creates a Server using the supplied transport and services, then applies
// opts in order. A non-nil bus enables event-driven diagnostic publication.
func New(transport protocol.Transport, projects *workspace.ProjectService, compiler *compile.CompilationService, bus *event.Bus, opts ...Option) *Server {
	s := &Server{
		transport:     transport,
		projects:      projects,
		compiler:      compiler,
		bus:           bus,
		lastPublished: make(map[string]uint64),
		registry:      newRequestRegistry(),
		logger:        observability.NewNoop(),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.subscribeDiagnostics(bus)
	s.subscribeEvictions(bus)
	return s
}

func (s *Server) subscribeEvictions(bus *event.Bus) {
	if bus == nil {
		return
	}
	bus.Subscribe([]event.Kind{event.ProjectEvicted}, func(e event.Event) {
		s.lastPubMu.Lock()
		delete(s.lastPublished, e.SourceRoot())
		s.lastPubMu.Unlock()
	})
}

// Serve reads and dispatches LSP messages until EOF, exit, or an error.
// An exit without a preceding shutdown returns an error.
func (s *Server) Serve() error {
	reader := bufio.NewReader(s.transport)
	for {
		payload, err := protocol.ReadFrame(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.handlePayload(payload); err != nil {
			if errors.Is(err, errServerExited) {
				return nil
			}
			return err
		}
	}
}

type requestLoggerCtxKey struct{}

func (s *Server) withRequestLogger(ctx context.Context, message protocol.Message) context.Context {
	logger := s.logger.With("method", message.Method)
	if idKey, ok := requestIDKey(message.ID); ok {
		logger = logger.With("id", idKey)
	}
	return context.WithValue(ctx, requestLoggerCtxKey{}, logger)
}

func loggerFromContext(ctx context.Context) *observability.Logger {
	if logger, ok := ctx.Value(requestLoggerCtxKey{}).(*observability.Logger); ok {
		return logger
	}
	return observability.NewNoop()
}

func (s *Server) handleMessage(message protocol.Message) error {
	ctx := s.withRequestLogger(context.Background(), message)
	loggerFromContext(ctx).Info("handling message")
	if len(message.ID) == 0 {
		return s.handleNotification(ctx, message)
	}
	return s.handleRequest(ctx, message)
}

func (s *Server) handleRequest(ctx context.Context, message protocol.Message) error {
	if s.shuttingDown {
		return s.writeError(message.ID, rpcInvalidRequest, "server is shutting down")
	}
	if !s.initialized && message.Method != "initialize" {
		return s.writeError(message.ID, rpcServerNotInitialized, "server not initialized")
	}
	if s.initialized && message.Method == "initialize" {
		return s.writeError(message.ID, rpcInvalidRequest, "already initialized")
	}
	switch message.Method {
	case "initialize", "shutdown":
		return s.handleLifecycleRequest(ctx, message)
	}
	return s.handleTrackedRequest(ctx, message)
}

// Lifecycle transitions run on the Serve goroutine, outside request cancellation.
func (s *Server) handleLifecycleRequest(ctx context.Context, message protocol.Message) error {
	var result any
	var ok bool
	switch message.Method {
	case "initialize":
		result, ok = s.handleInitialize(ctx, message.Params)
		if !ok {
			return s.writeError(message.ID, rpcInvalidParams, "invalid initialize parameters")
		}
	case "shutdown":
		s.handleShutdown()
	}
	return s.write(protocol.Response{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  result,
	})
}

func (s *Server) handleTrackedRequest(ctx context.Context, message protocol.Message) error {
	idKey, valid := requestIDKey(message.ID)
	if !valid {
		return nil
	}
	entry, reqCtx, ok := s.registry.register(idKey, ctx)
	if !ok {
		return s.writeError(message.ID, rpcInvalidRequest, "request id already in flight")
	}
	if message.Method == testBlockMethod {
		// Initialize before dispatch so a release cannot race the worker's startup.
		entry.release = make(chan struct{})
	}
	s.requestWG.Add(1)
	go func() {
		defer s.requestWG.Done()
		defer s.registry.unregister(idKey)
		res := s.dispatchTracked(reqCtx, message)
		if !res.handled {
			return
		}
		entry.replyOnce.Do(func() {
			if res.err == nil && reqCtx.Err() != nil {
				res.err = &protocol.RPCError{Code: rpcRequestCancelled, Message: "request cancelled"}
			}
			var err error
			if res.err != nil {
				err = s.writeError(message.ID, res.err.Code, res.err.Message)
			} else {
				err = s.write(protocol.Response{JSONRPC: "2.0", ID: message.ID, Result: res.result})
			}
			if err != nil {
				loggerFromContext(ctx).Error("response write failed", "error", err)
			}
		})
	}()
	return nil
}

// Handler errors take precedence over context cancellation when forming the reply.
type trackedResult struct {
	result  any
	err     *protocol.RPCError
	handled bool
}

func (s *Server) dispatchTracked(ctx context.Context, message protocol.Message) trackedResult {
	switch message.Method {
	case testBlockMethod:
		return s.handleTestBlockRequest(ctx, message)
	}
	return trackedResult{err: &protocol.RPCError{Code: rpcMethodNotFound, Message: "method not found"}, handled: true}
}

func (s *Server) handleShutdown() {
	if s.shuttingDown {
		return
	}
	s.shuttingDown = true
	s.cancelAndDrainTrackedRequests()
	if s.projects != nil {
		_ = s.projects.Shutdown(context.Background())
	}
	if s.compiler != nil {
		s.compiler.Shutdown()
	}
}

// Tracked handlers must finish before shutdown tears down workspace/compiler state
// or exit ends the Serve loop.
func (s *Server) cancelAndDrainTrackedRequests() {
	s.registry.cancelAll()
	s.requestWG.Wait()
}

func (s *Server) handleInitialize(_ context.Context, params json.RawMessage) (any, bool) {
	var initializeParams protocol.InitializeParams
	if json.Unmarshal(params, &initializeParams) != nil {
		return nil, false
	}
	s.initialized = true
	if caps, ok := initializeParams.Capabilities.TextDocument.Value(); ok {
		if diagCaps, ok := caps.PublishDiagnostics.Value(); ok {
			if versionSupport, ok := diagCaps.VersionSupport.Value(); ok {
				s.versionSupport = versionSupport
			}
		}
	}
	opts := protocol.TextDocumentSyncOptions{
		OpenClose: protocol.NewOptional(true),
		Change:    protocol.NewOptional(protocol.TextDocumentSyncKindIncremental),
		Save:      protocol.NewOptional(protocol.NewOrTextDocumentSyncOptionsSaveBoolean(true)),
	}
	return protocol.InitializeResult{Capabilities: protocol.ServerCapabilities{
		TextDocumentSync: protocol.NewOptional(protocol.NewOrServerCapabilitiesTextDocumentSyncTextDocumentSyncOptions(opts)),
	}}, true
}

func (s *Server) handleNotification(ctx context.Context, message protocol.Message) error {
	if message.Method == "exit" {
		s.cancelAndDrainTrackedRequests()
		if !s.shuttingDown {
			return errExitedWithoutShutdown
		}
		return errServerExited
	}
	if !s.initialized || s.shuttingDown {
		return nil
	}
	switch message.Method {
	case "initialized":
		return nil
	case setTraceMethod:
		// Runtime trace control is unsupported.
		return nil
	case flushMethod:
		return s.handleFlush()
	case testFlushRequestMethod:
		s.requestWG.Wait()
		return nil
	case testReleaseMethod:
		return s.handleTestReleaseRequest(message.Params)
	case cancelMethod:
		return s.handleCancelRequest(message.Params)
	case "textDocument/didOpen":
		return s.handleDidOpen(ctx, message.Params)
	case "textDocument/didChange":
		return s.handleDidChange(ctx, message.Params)
	case "textDocument/didClose":
		return s.handleDidClose(message.Params)
	case "textDocument/didSave":
		return nil
	case "workspace/didChangeWatchedFiles":
		return s.handleDidChangeWatchedFiles(ctx, message.Params)
	default:
		return nil
	}
}

func (s *Server) handleFlush() error {
	s.Flush()
	return nil
}

// The request goroutine owns the response, including cancellation errors.
func (s *Server) handleCancelRequest(params json.RawMessage) error {
	var cancelParams protocol.CancelParams
	if err := json.Unmarshal(params, &cancelParams); err != nil {
		return nil
	}
	idKey, ok := cancelIDKey(cancelParams.ID)
	if !ok {
		return nil
	}
	if entry := s.registry.lookup(idKey); entry != nil {
		entry.cancel()
	}
	return nil
}

// JSON encoding keeps string and integer IDs distinct: "1" is not 1.
func cancelIDKey(id protocol.OrCancelParamsId) (string, bool) {
	payload, err := json.Marshal(id)
	if err != nil {
		return "", false
	}
	return string(payload), true
}

func requestIDKey(raw json.RawMessage) (string, bool) {
	var id protocol.OrCancelParamsId
	if err := json.Unmarshal(raw, &id); err != nil {
		return "", false
	}
	return cancelIDKey(id)
}

func (s *Server) handleTestBlockRequest(ctx context.Context, message protocol.Message) trackedResult {
	idKey, _ := requestIDKey(message.ID)
	entry := s.registry.lookup(idKey)
	select {
	case <-entry.release:
		return trackedResult{result: map[string]any{"released": true}, handled: true}
	case <-ctx.Done():
		return trackedResult{err: &protocol.RPCError{Code: rpcRequestCancelled, Message: "request cancelled"}, handled: true}
	}
}

func (s *Server) handleTestReleaseRequest(params json.RawMessage) error {
	var p struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || len(p.ID) == 0 {
		return nil
	}
	idKey, ok := requestIDKey(p.ID)
	if !ok {
		return nil
	}
	if entry := s.registry.lookup(idKey); entry != nil && entry.release != nil {
		entry.closeRelease()
	}
	return nil
}

// Flush waits for queued and running compiles, diagnostic writes, and tiered
// event delivery. It is the corpus synchronization point for $test/flush.
// Callers must stop new work first; armed debounce timers are not drained.
func (s *Server) Flush() {
	if s.compiler != nil {
		s.compiler.Flush()
	}
	s.ceDone.Wait()
	if s.bus != nil {
		s.bus.Flush()
	}
}

func (s *Server) handleDidOpen(ctx context.Context, params json.RawMessage) error {
	var didOpen protocol.DidOpenTextDocumentParams
	if json.Unmarshal(params, &didOpen) != nil {
		return nil
	}
	docURI, err := workspace.NewFileURI(didOpen.TextDocument.URI)
	if err != nil {
		return nil
	}
	if _, err := s.projects.Apply(context.Background(), workspace.DocumentChange{
		Kind:       workspace.ChangeOpen,
		URI:        docURI,
		Text:       didOpen.TextDocument.Text,
		Version:    didOpen.TextDocument.Version,
		LanguageID: string(didOpen.TextDocument.LanguageID),
	}); err != nil {
		loggerFromContext(ctx).Error("workspace open failed", "error", err)
		return nil
	}
	return nil
}

func (s *Server) handleDidChange(ctx context.Context, params json.RawMessage) error {
	var didChange protocol.DidChangeTextDocumentParams
	if json.Unmarshal(params, &didChange) != nil {
		return nil
	}
	docURI, err := workspace.NewFileURI(didChange.TextDocument.URI)
	if err != nil {
		return nil
	}
	current, ok := s.projects.Snapshot(docURI)
	if !ok {
		return nil
	}
	fullText, ok := applyChanges(current.Text, didChange.ContentChanges)
	if !ok {
		return nil
	}
	if _, err := s.projects.Apply(context.Background(), workspace.DocumentChange{
		Kind:    workspace.ChangeUpdate,
		URI:     docURI,
		Text:    fullText,
		Version: didChange.TextDocument.Version,
	}); err != nil {
		loggerFromContext(ctx).Error("workspace update failed", "error", err)
		return nil
	}
	return nil
}

func (s *Server) handleDidClose(params json.RawMessage) error {
	var didClose protocol.DidCloseTextDocumentParams
	if json.Unmarshal(params, &didClose) != nil {
		return nil
	}
	docURI, err := workspace.NewFileURI(didClose.TextDocument.URI)
	if err != nil {
		return nil
	}
	if _, err := s.projects.Apply(context.Background(), workspace.DocumentChange{
		Kind: workspace.ChangeClose,
		URI:  docURI,
	}); err != nil {
		return nil
	}
	note := s.publishDiagnostics(didClose.TextDocument.URI, 0, false, nil)
	if note == nil {
		return nil
	}
	return s.write(*note)
}

func (s *Server) publishDiagnostics(uri string, version int32, includeVersion bool, diagnostics []protocol.Diagnostic) *protocol.Notification {
	if diagnostics == nil {
		diagnostics = []protocol.Diagnostic{}
	}
	params := protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}
	if includeVersion && s.versionSupport {
		params.Version = protocol.NewOptional(version)
	}
	return &protocol.Notification{
		JSONRPC: "2.0",
		Method:  publishDiagnosticsMethod,
		Params:  params,
	}
}

func (s *Server) writeError(id json.RawMessage, code int, message string) error {
	return s.write(protocol.ErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   protocol.RPCError{Code: code, Message: message},
	})
}

func (s *Server) write(message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	buf.Write(payload)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err = s.transport.Write(buf.Bytes())
	return err
}

func (s *Server) subscribeDiagnostics(bus *event.Bus) {
	if bus == nil {
		return
	}
	bus.SubscribeWithTier([]event.Kind{
		event.ResolutionDiagnosticsReady,
		event.CompilationDiagnosticsReady,
	}, event.TierBestEffort, func(e event.Event) {
		s.publishRootDiagnostics(e.SourceRoot(), e.Generation())
	})
}

// Both diagnostics-ready events refer to the same stable snapshot. Claim each
// generation once to avoid duplicate publication when both events arrive.
func (s *Server) publishRootDiagnostics(root string, gen uint64) {
	if g, ok := s.projects.Generation(root); !ok || g != gen {
		return
	}
	s.lastPubMu.Lock()
	if gen == s.lastPublished[root] {
		s.lastPubMu.Unlock()
		return
	}
	s.lastPublished[root] = gen
	s.lastPubMu.Unlock()

	diags, snapGen, ok := s.compiler.DiagnosticsFor(root)
	if !ok || snapGen != gen {
		return
	}
	openDocs := s.projects.OpenDocumentsUnder(root)
	s.ceDone.Add(len(openDocs))
	for _, u := range openDocs {
		u := u
		go func() {
			defer s.ceDone.Done()
			if g, ok := s.projects.Generation(root); !ok || g != gen {
				return
			}
			snap, hasSnap := s.projects.Snapshot(u)
			version := int32(0)
			include := false
			var text string
			if hasSnap {
				version = snap.Version
				include = true
				text = snap.Text
			}
			note := s.publishDiagnostics(u.String(), version, include, convertDiagnostics(diags[u], text))
			if note == nil {
				return
			}
			if err := s.write(*note); err != nil {
				s.logger.Error("diagnostics write failed", "error", err)
			}
		}()
	}
}

func (s *Server) handleDidChangeWatchedFiles(ctx context.Context, params json.RawMessage) error {
	var didChange protocol.DidChangeWatchedFilesParams
	if json.Unmarshal(params, &didChange) != nil {
		return nil
	}
	for _, fileEvent := range didChange.Changes {
		docURI, err := workspace.NewFileURI(fileEvent.URI)
		if err != nil {
			continue
		}
		change := workspace.WatchedFileChange{URI: docURI}
		switch fileEvent.Type {
		case protocol.FileChangeTypeCreated:
			change.Kind = workspace.WatchedFileCreate
		case protocol.FileChangeTypeDeleted:
			change.Kind = workspace.WatchedFileDelete
		case protocol.FileChangeTypeChanged:
			change.Kind = workspace.WatchedFileModified
		default:
			continue
		}
		if err := s.projects.ApplyWatchedFile(ctx, change); err != nil {
			loggerFromContext(ctx).Error("workspace watched-file update failed", "error", err)
			continue
		}
	}
	return nil
}
