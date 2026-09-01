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
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

const (
	publishDiagnosticsMethod = "textDocument/publishDiagnostics"
	// flushMethod is the corpus-only sentinel notification that triggers a
	// deterministic drain of the async compile engine + CE delivery so the
	// driver can read publishDiagnostics written out-of-band. It is never sent
	// by a real client.
	flushMethod = "$pal/flush"
	// cancelMethod is the standard LSP $/cancelRequest notification. It maps
	// a request id to context cancellation of that in-flight request only — it
	// does not call CompilationService.Cancel or supersede any source root
	// (see requestRegistry). Unknown, malformed, completed, and duplicate
	// cancellation notifications are no-ops.
	cancelMethod = "$/cancelRequest"
	// setTraceMethod is the standard LSP $/setTrace notification. The server
	// accepts it (and initialize's trace field) but implements both as
	// accept-and-discard no-ops — verbosity is fixed at process start, not
	// runtime-adjustable (observability strategy ADR).
	setTraceMethod = "$/setTrace"
	// testBlockMethod, testReleaseMethod, and testFlushRequestsMethod are
	// corpus-only sentinels (like flushMethod) that form the deterministic
	// test seam for request cancellation: a $pal/blockRequest request holds
	// until either $pal/releaseRequest (normal completion) or its context is
	// cancelled by $/cancelRequest; $pal/flushRequests waits for every tracked
	// request goroutine to finish so the driver can read replies without races.
	// They are never sent by a real client.
	testBlockMethod        = "$pal/blockRequest"
	testReleaseMethod      = "$pal/releaseRequest"
	testFlushRequestMethod = "$pal/flushRequests"
)

var (
	errServerExited          = errors.New("language server exited")
	errExitedWithoutShutdown = errors.New("language server exited without shutdown")
)

// rpcError codes for request cancellation dispatch. RequestCancelled is the
// LSP -32800 code; InvalidRequest is the JSON-RPC -32600 code used when a new
// request reuses an in-flight request id.
const (
	rpcRequestCancelled = -32800
	rpcInvalidRequest   = -32600
)

// requestEntry is the server-private registry record for one in-flight
// non-lifecycle request: the request's child context cancel function, a
// reply guard ensuring exactly one JSON-RPC response is written per request,
// and (for the corpus-only $pal/blockRequest seam) a one-shot release channel
// created at registration time so $pal/releaseRequest can find it without
// racing the blocked goroutine's startup.
type requestEntry struct {
	cancel      context.CancelFunc
	replyOnce   sync.Once
	release     chan struct{}
	releaseOnce sync.Once
}

func (e *requestEntry) closeRelease() {
	e.releaseOnce.Do(func() { close(e.release) })
}

// requestRegistry canonically keys valid string and integer JSON-RPC ids to
// their in-flight request entry. A new request reusing an in-flight id is
// rejected (register returns false) and cannot replace the original entry;
// the entry is removed when its goroutine completes.
type requestRegistry struct {
	mu      sync.Mutex
	entries map[string]*requestEntry
}

func newRequestRegistry() *requestRegistry {
	return &requestRegistry{entries: make(map[string]*requestEntry)}
}

// register derives a cancellable child of parent for the tracked request.
// parent carries the request-scoped logger attached at handleMessage (value
// only, no cancellation of its own), so deriving from it rather than
// context.Background() leaves cancellation semantics unchanged while making
// the logger reachable inside dispatchTracked's handlers via ctx.
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

// cancelAll cancels every in-flight request's context. Used by shutdown/exit
// to drain tracked work before completing the lifecycle transition.
func (r *requestRegistry) cancelAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entry := range r.entries {
		entry.cancel()
	}
}

type Server struct {
	transport                      protocol.Transport
	projects                       *workspace.ProjectService
	compiler                       *compile.CompilationService
	bus                            *event.Bus
	versionSupport                 bool
	initialized                    bool
	shuttingDown                   bool

	writeMu sync.Mutex // serializes framed writes (Serve + CE subscriber)

	ceDone        sync.WaitGroup // tracks in-flight CE→publish tasks (flush)
	lastPubMu     sync.Mutex
	lastPublished map[string]uint64 // per-root generation dedup for CE publishing

	registry  *requestRegistry // in-flight non-lifecycle requests
	requestWG sync.WaitGroup   // tracks in-flight request goroutines (drain)

	logger *observability.Logger
}

// Option configures a Server at construction time.
type Option func(*Server)

// WithLogger injects the observability facade the server attaches
// request-scoped (method+id) child loggers from at handleMessage. The
// default (unset) is observability.NewNoop(), so existing New() call sites
// need no change.
func WithLogger(logger *observability.Logger) Option {
	return func(s *Server) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// New creates a Server with the given transport and core services. The bus is
// used to subscribe to CE-E5a/E5b for out-of-band diagnostic publication. The
// services are injected explicitly — the corpus driver wires
// workspace.New(platform) and compile.New(platform, bus).
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

// subscribeEvictions clears the per-root generation dedup mark when a source
// root is evicted (WM-E2), so a subsequent reload — which restarts the
// generation counter at 1 — is not suppressed behind the evicted root's stale
// high-water mark. Without this the dedup in publishRootDiagnostics could drop
// the reload's first diagnostics.
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

func (s *Server) Serve() error {
	reader := bufio.NewReader(s.transport)
	for {
		message, err := protocol.ReadMessage(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.handleMessage(message); err != nil {
			if errors.Is(err, errServerExited) {
				return nil
			}
			return err
		}
	}
}

// requestLoggerCtxKey is the unexported context key the request-scoped
// logger is stored under. Kept private to this package — observability
// exposes no context-embedding API of its own (ticket 41 ADR scopes the
// facade to a concrete *Logger, not a context convention).
type requestLoggerCtxKey struct{}

// withRequestLogger attaches a child of s.logger carrying "method" (and,
// for requests, "id") to ctx. This runs at handleMessage — the single
// universal entry point for every inbound message — so the correlated
// logger is available for every dispatch path reached from here: lifecycle
// requests, notifications, and tracked requests alike, not just the methods
// dispatchTracked's switch happens to cover. Notifications carry no id, so
// only "method" is attached for them.
func (s *Server) withRequestLogger(ctx context.Context, message protocol.Message) context.Context {
	logger := s.logger.With("method", message.Method)
	if idKey, ok := requestIDKey(message.ID); ok {
		logger = logger.With("id", idKey)
	}
	return context.WithValue(ctx, requestLoggerCtxKey{}, logger)
}

// loggerFromContext returns the request-scoped logger attached by
// withRequestLogger, or a no-op logger if ctx carries none.
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
	switch message.Method {
	case "initialize", "shutdown":
		return s.handleLifecycleRequest(ctx, message)
	}
	return s.handleTrackedRequest(ctx, message)
}

// handleLifecycleRequest runs initialize/shutdown synchronously in the Serve
// loop — they are control-plane transitions, not tracked cancellable work.
func (s *Server) handleLifecycleRequest(ctx context.Context, message protocol.Message) error {
	if s.shuttingDown && message.Method != "shutdown" {
		return nil
	}
	var result any
	var ok bool
	switch message.Method {
	case "initialize":
		result, ok = s.handleInitialize(ctx, message.Params)
		if !ok {
			return nil
		}
	case "shutdown":
		result, ok = s.handleShutdown(ctx)
		if !ok {
			return nil
		}
	}
	return s.write(protocol.Response{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  result,
	})
}

// handleTrackedRequest runs a non-lifecycle request in a tracked goroutine
// with a cancellable context. A new request reusing an in-flight id is
// rejected with Invalid Request (-32600) and does not replace the original
// registry entry. New work is gated out once the server is shutting down.
func (s *Server) handleTrackedRequest(ctx context.Context, message protocol.Message) error {
	if s.shuttingDown {
		return nil
	}
	idKey, valid := requestIDKey(message.ID)
	if !valid {
		return nil
	}
	entry, reqCtx, ok := s.registry.register(idKey, ctx)
	if !ok {
		return s.write(protocol.ErrorResponse{
			JSONRPC: "2.0",
			ID:      message.ID,
			Error:   protocol.RPCError{Code: rpcInvalidRequest, Message: "request id already in flight"},
		})
	}
	if message.Method == testBlockMethod {
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
			if res.err != nil {
				s.write(protocol.ErrorResponse{
					JSONRPC: "2.0",
					ID:      message.ID,
					Error:   *res.err,
				})
				return
			}
			if reqCtx.Err() != nil {
				s.write(protocol.ErrorResponse{
					JSONRPC: "2.0",
					ID:      message.ID,
					Error:   protocol.RPCError{Code: rpcRequestCancelled, Message: "request cancelled"},
				})
				return
			}
			s.write(protocol.Response{
				JSONRPC: "2.0",
				ID:      message.ID,
				Result:  res.result,
			})
		})
	}()
	return nil
}

// trackedResult is the outcome of a non-lifecycle request handler. A concrete
// handler error (non-nil err) wins over cancellation only if the handler
// returned it; a handler that observes its context being cancelled returns
// errRequestCancelled so the reply is the -32800 error.
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
	return trackedResult{}
}

func (s *Server) handleShutdown(_ context.Context) (any, bool) {
	if s.shuttingDown {
		return nil, true
	}
	s.shuttingDown = true
	s.cancelAndDrainTrackedRequests()
	if s.projects != nil {
		_ = s.projects.Shutdown(context.Background())
	}
	if s.compiler != nil {
		s.compiler.Shutdown()
	}
	return nil, true
}

// cancelAndDrainTrackedRequests cancels every in-flight non-lifecycle request
// and waits for their goroutines to finish (each writes its detached
// RequestCancelled reply under writeMu) before returning. Used by shutdown
// (before the workspace/compiler drain) and exit (before ending the loop).
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

func (s *Server) handleNotification(_ context.Context, message protocol.Message) error {
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
		// Accept-and-discard no-op: verbosity is fixed at process start
		// (observability strategy ADR); dynamic trace control is unimplemented
		// by design, matching gopls/rust-analyzer precedent.
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
		return s.handleDidOpen(message.Params)
	case "textDocument/didChange":
		return s.handleDidChange(message.Params)
	case "textDocument/didClose":
		return s.handleDidClose(message.Params)
	case "textDocument/didSave":
		return nil
	case "workspace/didChangeWatchedFiles":
		return s.handleDidChangeWatchedFiles(message.Params)
	default:
		return nil
	}
}

// handleFlush is the corpus-only $pal/flush notification handler.
func (s *Server) handleFlush() error {
	s.Flush()
	return nil
}

// handleCancelRequest parses the $/cancelRequest notification's id and
// cancels only that in-flight request's context. It does not call
// CompilationService.Cancel, supersede a source root, or write a response —
// the cancelled request's goroutine writes the detached RequestCancelled reply
// itself. Unknown, malformed, completed, and duplicate cancellation
// notifications are no-ops: a malformed payload or non-string/non-integer id
// fails to decode and is ignored; an id with no registry entry is a lookup
// miss; an already-completed request's entry has been unregistered; a repeat
// cancel simply re-invokes an already-cancelled context's cancel (idempotent).
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

// cancelIDKey renders a CancelParams id as the canonical registry key, so a
// string id "1" (key `"1"`) and an integer id 1 (key `1`) remain distinct at
// cancellation and registration.
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

// handleTestBlockRequest is the corpus-only $pal/blockRequest handler: it
// holds the request until $pal/releaseRequest closes its release channel
// (normal completion → {"released": true}) or its context is cancelled by
// $/cancelRequest (→ RequestCancelled -32800). The release channel is created
// on the registry entry at registration time (in the Serve goroutine), so a
// subsequent $pal/releaseRequest finds it without racing this goroutine's
// startup. It never touches the compiler.
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

// handleTestReleaseRequest closes the release channel of the in-flight
// $pal/blockRequest with the matching id, letting it complete normally. An
// unknown, completed, or malformed id is a no-op.
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

// Flush is the deterministic drain point: it waits for the compile engine's
// in-flight cycles, the server's in-flight CE→publish tasks, and the tiered
// bus delivery, so all out-of-band publishDiagnostics have been written. It is
// the corpus driver's sync point (the $pal/flush sentinel); production code
// does not call it.
func (s *Server) Flush() {
	if s.compiler != nil {
		s.compiler.Flush()
	}
	s.ceDone.Wait()
	if s.bus != nil {
		s.bus.Flush()
	}
}

// handleDidOpen applies the open and returns nil — diagnostics arrive later via
// the CE subscriber (branch 5). No synchronous Compile or publishDiagnostics.
func (s *Server) handleDidOpen(params json.RawMessage) error {
	var didOpen protocol.DidOpenTextDocumentParams
	if json.Unmarshal(params, &didOpen) != nil {
		return nil
	}
	docURI, err := uri.NewFileURI(didOpen.TextDocument.URI)
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
		return nil
	}
	return nil
}

// handleDidChange applies the update and returns nil — diagnostics arrive later
// via the CE subscriber (branch 5).
func (s *Server) handleDidChange(params json.RawMessage) error {
	var didChange protocol.DidChangeTextDocumentParams
	if json.Unmarshal(params, &didChange) != nil {
		return nil
	}
	docURI, err := uri.NewFileURI(didChange.TextDocument.URI)
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
		return nil
	}
	return nil
}

// handleDidClose applies the close and synchronously clears diagnostics for the
// document (LSP clients expect diagnostics cleared on close).
func (s *Server) handleDidClose(params json.RawMessage) error {
	var didClose protocol.DidCloseTextDocumentParams
	if json.Unmarshal(params, &didClose) != nil {
		return nil
	}
	docURI, err := uri.NewFileURI(didClose.TextDocument.URI)
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

// write serializes a framed write: marshal, build the full framed message in a
// buffer (header+body), and Write once under writeMu so concurrent writes from
// the Serve goroutine and the CE-subscriber goroutine cannot interleave
// header/body bytes (gopls conn.writeMu precedent).
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

// subscribeDiagnostics wires the server's out-of-band diagnostic publication
// (branch 5). It subscribes to CE-E5a/CE-E5b (BEST_EFFORT) and, on each event,
// reads the stable snapshot via compile.DiagnosticsFor, applies the
// generation-staleness guard per document, converts to protocol.Diagnostic, and
// writes publishDiagnostics per open document via s.write. Each publish task
// increments s.ceDone so the flush sentinel can wait for it.
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

// publishRootDiagnostics publishes the current stable snapshot's diagnostics
// for every open document under root, gated by the event's generation. Under
// the no-mid-cycle-hooks model (branch 2) a clean cycle emits both CE-E5a and
// CE-E5b for the same generation, each carrying the complete post-cycle set;
// a resolution-error cycle emits CE-E5a only. The publish is therefore
// first-wins per generation: whichever of CE-E5a/CE-E5b delivers first
// publishes the complete set for the cycle and the other is a no-op. This is
// robust to a BEST_EFFORT drop of either event (the survivor still publishes)
// and to a source-root reload, which restarts the generation counter at 1 —
// an exact-equality guard (not <=) avoids suppressing the reload's diagnostics
// behind a stale high-water mark left by the previous incarnation.
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
			s.write(*note)
		}()
	}
}

// handleDidChangeWatchedFiles routes workspace/didChangeWatchedFiles to the
// project service. Only file: URIs are admitted — non-file: events are ignored,
// matching the file:-only routing of 09. Watched-file republishes flow through
// ApplyWatchedFile (which bumps the generation and publishes ProjectUpdated);
// diagnostics arrive later via the CE subscriber.
func (s *Server) handleDidChangeWatchedFiles(params json.RawMessage) error {
	var didChange protocol.DidChangeWatchedFilesParams
	if json.Unmarshal(params, &didChange) != nil {
		return nil
	}
	for _, fileEvent := range didChange.Changes {
		docURI, err := uri.NewFileURI(fileEvent.URI)
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
		if err := s.projects.ApplyWatchedFile(context.Background(), change); err != nil {
			continue
		}
	}
	return nil
}
