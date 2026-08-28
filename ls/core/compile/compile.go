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

// Package compile provides CompilationService, the core compilation engine.
// Ticket 09 implements the resolution-first, bounded, single-flight-per-root
// dual-snapshot compiler engine: Apply publishes a package (modifier chain on
// a persistent per-source-root CompilerEnvironment) and ProjectUpdated (WM-E4);
// the engine's CompilationPipeline subscriber (CRITICAL) enqueues a compile
// cycle for (root, generation) on a bounded worker pool. A cycle runs the
// modifier-chain package's PackageCompilation (panic-recovered → CE-E2), then
// SnapshotStore.publishStable runs the stale-publication gate, stores the
// stable snapshot, and publishes CE-E1 / CE-E5a / CE-E5b. The server consumes
// CE-E5a/E5b out-of-band to publish publishDiagnostics.
//
// The synchronous Compile(ctx, CompileRequest) is retained as the fast path for
// semantic-query consumers (10/11): it reads SnapshotStore.Stable first (cache
// hit) and falls back to compiling inline.
package compile

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// CompileRequest carries the document URI to compile.
type CompileRequest struct {
	URI uri.DocumentURI
}

// CompileResult holds the core-defined diagnostics from a compilation.
type CompileResult struct {
	Diagnostics []CompilerDiagnostic
}

// Severity is a core-defined diagnostic severity.
type Severity uint8

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInformation
	SeverityHint
)

// CompilerDiagnostic is a core-defined diagnostic with byte-offset-derived
// positions. StartLine/StartChar/EndLine/EndChar use byte character offsets
// within each line (not UTF-16 code units). The server converts these to
// protocol.Position at the boundary.
type CompilerDiagnostic struct {
	StartLine uint32
	StartChar uint32
	EndLine   uint32
	EndChar   uint32
	Severity  Severity
	Code      string
	Message   string
}

// projectReader is the read-only view of the workspace the engine uses
// (CurrentProject/Generation/OpenDocumentsUnder). *workspace.ProjectService
// satisfies it; the concrete direct reference is retained for Compile's
// inline fallback.
type projectReader interface {
	CurrentProject(root string) (projects.Project, uint64, bool)
	Generation(root string) (uint64, bool)
	OpenDocumentsUnder(root string) []uri.DocumentURI
}

// cycleResult is the extracted outcome of one compile, grouped by file.
type cycleResult struct {
	byFile            map[string][]CompilerDiagnostic
	resByFile         map[string][]CompilerDiagnostic
	resolutionErrored bool
	descriptor        string
}

// compileFunc runs a package's compile and extracts its diagnostics. It may
// panic (the compiler re-panics on Phase-2 failures); the engine recovers and
// publishes CE-E2. The default is realCompilePackage; an internal seam allows
// tests to inject a panicking compile.
type compileFunc func(pkg *projects.Package) cycleResult

// Option configures the CompilationService at construction.
type Option func(*CompilationService)

// WithMaxWorkers overrides the worker pool size.
func WithMaxWorkers(n int) Option {
	return func(s *CompilationService) {
		if n > 0 {
			s.maxWorkers = n
		}
	}
}

// WithMaxStableSnapshots overrides the stable-snapshot count bound (ADR-058).
func WithMaxStableSnapshots(n int) Option {
	return func(s *CompilationService) {
		if n > 0 {
			s.store.maxStable = n
		}
	}
}

// WithDebounce overrides the per-source-root debounce. 0 disables debounce
// (each change compiles immediately, deterministically — the corpus setting).
func WithDebounce(d time.Duration) Option {
	return func(s *CompilationService) {
		s.debounce = d
	}
}

// WithQueueDepth overrides the pending-queue depth. The model is depth-1
// latest-wins; deeper values are stored but the engine keeps one pending slot.
func WithQueueDepth(n int) Option {
	return func(s *CompilationService) {
		_ = n // depth-1 latest-wins is the Phase-C bound; reserved.
	}
}

// CompilationService is the dual-snapshot compilation engine. It holds a direct
// *workspace.ProjectService read reference and the SnapshotStore, and
// subscribes to ProjectUpdated (CRITICAL) to enqueue compile cycles.
type CompilationService struct {
	projects  *workspace.ProjectService
	reader    projectReader
	bus       *event.Bus
	store     *SnapshotStore
	compileFn compileFunc

	cycleMu    sync.Mutex
	inFlight   map[string]bool
	pending    map[string]*cycleRequest
	idleCond   *sync.Cond
	workSem    chan struct{}
	maxWorkers int

	debounce       time.Duration
	debounceMu     sync.Mutex
	debounceTimers map[string]*time.Timer
	debounceGens   map[string]uint64

	knownRootsMu sync.Mutex
	knownRoots   map[string]struct{}

	shutdownOnce sync.Once
	closed       bool
}

type cycleRequest struct {
	root    string
	gen     uint64
	pkg     *projects.Package
	project projects.Project
}

// New creates a CompilationService wired to the project service (direct read
// reference) and the event bus. It subscribes to ProjectUpdated (CRITICAL) to
// enqueue compile cycles, and to ProjectRegistered/ProjectEvicted/
// ProjectKindTransitioned (inline) to maintain the known-roots set and evict
// snapshots.
func New(projects *workspace.ProjectService, bus *event.Bus, opts ...Option) *CompilationService {
	s := &CompilationService{
		projects:       projects,
		reader:         projects,
		bus:            bus,
		store:          newSnapshotStore(16, projects.Generation),
		compileFn:      realCompilePackage,
		inFlight:       make(map[string]bool),
		pending:        make(map[string]*cycleRequest),
		debounce:       150 * time.Millisecond,
		debounceTimers: make(map[string]*time.Timer),
		debounceGens:   make(map[string]uint64),
		knownRoots:     make(map[string]struct{}),
		maxWorkers:     defaultMaxWorkers(),
	}
	s.idleCond = sync.NewCond(&s.cycleMu)
	for _, opt := range opts {
		opt(s)
	}
	s.workSem = make(chan struct{}, s.maxWorkers)
	if bus != nil {
		bus.SubscribeWithTier([]event.Kind{event.ProjectUpdated}, event.TierCritical, s.enqueueCycle)
		bus.Subscribe([]event.Kind{
			event.ProjectRegistered,
			event.ProjectEvicted,
			event.ProjectKindTransitioned,
		}, s.handleLifecycle)
	}
	return s
}

func defaultMaxWorkers() int {
	n := runtime.NumCPU()
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	return n
}

// handleLifecycle maintains the known-roots set and evicts snapshots when a
// root is evicted (inline, 08 behavior retained).
func (s *CompilationService) handleLifecycle(e event.Event) {
	s.knownRootsMu.Lock()
	defer s.knownRootsMu.Unlock()
	switch e.Kind() {
	case event.ProjectRegistered:
		s.knownRoots[e.SourceRoot()] = struct{}{}
	case event.ProjectEvicted:
		delete(s.knownRoots, e.SourceRoot())
		s.store.evictRoot(e.SourceRoot())
	case event.ProjectKindTransitioned:
		if te, ok := e.(event.ProjectKindTransitionedEvent); ok {
			delete(s.knownRoots, te.OldRoot())
			s.store.evictRoot(te.OldRoot())
		}
		s.knownRoots[e.SourceRoot()] = struct{}{}
	}
}

func (s *CompilationService) isKnown(root string) bool {
	s.knownRootsMu.Lock()
	defer s.knownRootsMu.Unlock()
	_, ok := s.knownRoots[root]
	return ok
}

// enqueueCycle is the CRITICAL-tier subscriber for ProjectUpdated. It captures
// the published package and enqueues a compile cycle for (root, generation),
// respecting the depth-1 latest-wins single-flight queue. With a non-zero
// debounce it coalesces rapid changes via a per-root timer.
func (s *CompilationService) enqueueCycle(e event.Event) {
	root := e.SourceRoot()
	gen := e.Generation()
	if s.isClosed() {
		return
	}
	if s.debounce > 0 {
		s.scheduleDebounced(root, gen)
		return
	}
	s.enqueueImmediately(root, gen)
}

func (s *CompilationService) isClosed() bool {
	s.cycleMu.Lock()
	defer s.cycleMu.Unlock()
	return s.closed
}

// scheduleDebounced resets the per-root debounce timer to fire with the latest
// generation (coalescing rapid changes into one cycle).
func (s *CompilationService) scheduleDebounced(root string, gen uint64) {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()
	s.debounceGens[root] = gen
	if t := s.debounceTimers[root]; t != nil {
		t.Reset(s.debounce)
		return
	}
	s.debounceTimers[root] = time.AfterFunc(s.debounce, func() {
		s.debounceMu.Lock()
		fireGen := s.debounceGens[root]
		delete(s.debounceGens, root)
		delete(s.debounceTimers, root)
		s.debounceMu.Unlock()
		s.enqueueImmediately(root, fireGen)
	})
}

// enqueueImmediately captures the current package and submits the cycle
// (or parks it in the depth-1 pending slot if a cycle is already in flight).
func (s *CompilationService) enqueueImmediately(root string, gen uint64) {
	project, _, ok := s.reader.CurrentProject(root)
	if !ok || project == nil {
		return
	}
	pkg := project.CurrentPackage()
	if pkg == nil {
		return
	}
	req := &cycleRequest{root: root, gen: gen, pkg: pkg, project: project}
	s.cycleMu.Lock()
	if s.closed {
		s.cycleMu.Unlock()
		return
	}
	if s.inFlight[root] {
		s.pending[root] = req // depth-1 latest-wins
		s.cycleMu.Unlock()
		return
	}
	s.inFlight[root] = true
	s.cycleMu.Unlock()
	s.submit(req)
}

// submit runs a cycle on the worker pool. The workSem bounds concurrent
// compiles across roots to maxWorkers.
func (s *CompilationService) submit(req *cycleRequest) {
	s.workSem <- struct{}{}
	go func() {
		s.runCycle(req)
		<-s.workSem
		s.finishCycle(req.root)
	}()
}

// runCycle runs one compile cycle: pre-compile stale check, compile
// (panic-recovered → CE-E2), then SnapshotStore.publishStable (gate + store +
// CE-E1/E5a/E5b).
func (s *CompilationService) runCycle(req *cycleRequest) {
	if s.isClosed() {
		return
	}
	current, ok := s.reader.Generation(req.root)
	if !ok || req.gen != current {
		s.bus.Publish(event.NewCompilationCancelledEvent(req.root, req.gen))
		return
	}
	done := make(chan struct{})
	s.store.setInProgress(req.root, InProgressSnapshot{key: CompilationKey{req.root, req.gen}, done: done})
	defer func() {
		close(done)
		s.store.clearInProgress(req.root)
	}()

	var result cycleResult
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				s.bus.Publish(event.NewCompilationFailedEvent(req.root, req.gen))
			}
		}()
		result = s.compileFn(req.pkg)
	}()
	if panicked || s.isClosed() {
		return
	}
	snap := StableSnapshot{
		key:               CompilationKey{SourceRoot: req.root, Generation: req.gen},
		project:           req.project,
		pkg:               req.pkg,
		descriptor:        result.descriptor,
		byFile:            result.byFile,
		resByFile:         result.resByFile,
		resolutionErrored: result.resolutionErrored,
	}
	s.store.publishStable(snap, s.bus)
}

// finishCycle runs after a cycle's worker slot is freed. If a pending
// (superseding) cycle exists for the root it starts it (in-flight stays
// true); otherwise the root is idle and Flush/Shutdown waiters are woken.
func (s *CompilationService) finishCycle(root string) {
	s.cycleMu.Lock()
	p := s.pending[root]
	if p != nil {
		s.pending[root] = nil
	} else {
		s.inFlight[root] = false
		s.idleCond.Broadcast()
	}
	s.cycleMu.Unlock()
	if p != nil {
		s.submit(p)
	}
}

// Flush blocks until every in-flight compile has finished and every CE event
// has been delivered and drained by the tiered bus. It is the deterministic
// sync point the corpus driver uses instead of timing sleeps. Production code
// does not call it. With WithDebounce(0) (the corpus setting) it needs only
// pool-idle + bus-drain.
func (s *CompilationService) Flush() {
	// 1. Drain incoming lifecycle events (ProjectUpdated) so the pipeline
	//    drainer enqueues compile cycles on the pool.
	if s.bus != nil {
		s.bus.Flush()
	}
	// 2. Wait for every in-flight compile to finish.
	s.cycleMu.Lock()
	for !s.allIdleLocked() {
		s.idleCond.Wait()
	}
	s.cycleMu.Unlock()
	// 3. Drain the outgoing CE events the compiles published.
	if s.bus != nil {
		s.bus.Flush()
	}
}

func (s *CompilationService) allIdleLocked() bool {
	for _, f := range s.inFlight {
		if f {
			return false
		}
	}
	for _, p := range s.pending {
		if p != nil {
			return false
		}
	}
	return true
}

// Shutdown stops accepting new cycles, stops debounce timers, bounded-waits for
// in-flight compiles to finish (their results are gated out by the closed
// flag), flushes the bus, and clears the snapshot store. It is idempotent.
func (s *CompilationService) Shutdown() {
	s.shutdownOnce.Do(func() {
		s.cycleMu.Lock()
		s.closed = true
		s.cycleMu.Unlock()
		s.stopDebounceTimers()
		// Bounded-wait for in-flight compiles to finish. They observe s.closed
		// and skip publication; the worker pool drains.
		s.cycleMu.Lock()
		for !s.allIdleLocked() {
			s.idleCond.Wait()
		}
		s.cycleMu.Unlock()
		if s.bus != nil {
			s.bus.Flush()
		}
		s.store.evictAll()
	})
}

// Cancel is the $/cancelRequest mapping (design branch 3). Go cannot interrupt
// a running compile, so cancellation is boundary-only: it bumps the workspace
// generation of every root with an in-flight or depth-1 pending cycle, so the
// running compile finishes but its result is gated out at the stale-
// publication gate (CE-E3) and the pending cycle is dropped on dequeue. It
// does not stop debounce timers or close the bus. Per-root identification of
// the cancelled request is deferred to 10/11 — 09 has no cancellable
// per-document request to map a $/cancelRequest id to a source root — so
// Cancel applies to every active root.
func (s *CompilationService) Cancel() {
	if s.isClosed() {
		return
	}
	s.cycleMu.Lock()
	roots := make(map[string]struct{}, len(s.inFlight)+len(s.pending))
	for r, f := range s.inFlight {
		if f {
			roots[r] = struct{}{}
		}
	}
	for r, p := range s.pending {
		if p != nil {
			roots[r] = struct{}{}
		}
	}
	s.cycleMu.Unlock()
	for r := range roots {
		s.projects.Supersede(r)
	}
}

func (s *CompilationService) stopDebounceTimers() {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()
	for _, t := range s.debounceTimers {
		t.Stop()
	}
	s.debounceTimers = make(map[string]*time.Timer)
	s.debounceGens = make(map[string]uint64)
}

// DiagnosticsFor returns the per-open-document diagnostics from the latest
// stable snapshot for root, keyed by open-document URI, plus the snapshot's
// generation. ok is false if no stable snapshot exists. Diagnostics are
// core-defined; the server converts. The caller performs the generation-
// staleness guard before publishing.
func (s *CompilationService) DiagnosticsFor(root string) (diags map[uri.DocumentURI][]CompilerDiagnostic, generation uint64, ok bool) {
	snap, has := s.store.Stable(root)
	if !has {
		return nil, 0, false
	}
	out := make(map[uri.DocumentURI][]CompilerDiagnostic)
	for _, u := range s.reader.OpenDocumentsUnder(root) {
		out[u] = snap.diagsForFile(u.Path())
	}
	return out, snap.key.Generation, true
}

// Compile is the synchronous fast path for semantic-query consumers (10/11).
// It reads SnapshotStore.Stable first (cache hit — the perf win over 08's
// recompile-per-change) and falls back to compiling the published package
// inline. No diagnostic publication happens here under 09; that moves to the
// server CE subscriber.
func (s *CompilationService) Compile(ctx context.Context, req CompileRequest) (CompileResult, error) {
	_ = ctx
	project, err := s.projects.Project(req.URI)
	if err == nil && project != nil {
		if snap, ok := s.store.Stable(project.SourceRoot()); ok {
			return CompileResult{Diagnostics: snap.diagsForFile(req.URI.Path())}, nil
		}
	}
	if project == nil || !s.isKnown(project.SourceRoot()) {
		return CompileResult{}, nil
	}
	pkg := project.CurrentPackage()
	if pkg == nil {
		return CompileResult{}, nil
	}
	diags := extractForURI(project, pkg, req.URI.Path())
	return CompileResult{Diagnostics: diags}, nil
}

// extractForURI extracts the diagnostics for a single file path (the 08 inline
// path, retained as Compile's fallback).
func extractForURI(project projects.Project, pkg *projects.Package, fileName string) []CompilerDiagnostic {
	docID, ok := project.DocumentID(fileName)
	if !ok {
		return nil
	}
	module := pkg.Module(docID.ModuleID())
	if module == nil {
		return nil
	}
	doc := module.Document(docID)
	if doc == nil {
		return nil
	}
	text := doc.TextDocument().String()
	compilation := pkg.Compilation()
	env := compilation.DiagnosticEnv()
	lineStarts := computeLineStarts(text)
	var diags []CompilerDiagnostic
	for _, diag := range compilation.DiagnosticResult().Diagnostics() {
		location := diag.Location()
		if !diagnostics.LocationHasSource(location) {
			continue
		}
		if env.FileName(location) != fileName {
			continue
		}
		diags = append(diags, convertDiag(text, lineStarts, diag))
	}
	return diags
}

// realCompilePackage runs the package's compile and extracts all diagnostics
// grouped by file, plus the resolution subset and the resolution-error flag
// (branch 2: resolution vs compilation classification).
func realCompilePackage(pkg *projects.Package) cycleResult {
	comp := pkg.Compilation()
	env := comp.DiagnosticEnv()
	descriptor := pkg.Descriptor().Name().Value()
	byFile, resByFile, resErr := extractByFile(pkg, comp, env)
	return cycleResult{
		byFile:            byFile,
		resByFile:         resByFile,
		resolutionErrored: resErr,
		descriptor:        descriptor,
	}
}

// extractByFile extracts all diagnostics and the resolution subset, grouped by
// the env.FileName key (= the document's SyntaxTree.FilePath, i.e. the
// registrationKey / file path for current-package files). Documents are
// resolved within the captured immutable package so the extraction never
// reads the project's live currentPackage (which a concurrent Apply's modifier
// chain may swap). Diagnostics whose file is not a document in the package
// (dependency/manifest diags) are skipped.
func extractByFile(pkg *projects.Package, comp *projects.PackageCompilation, env *diagnostics.DiagnosticEnv) (byFile, resByFile map[string][]CompilerDiagnostic, resErr bool) {
	byFile = make(map[string][]CompilerDiagnostic)
	resByFile = make(map[string][]CompilerDiagnostic)
	type docInfo struct {
		text       string
		lineStarts []int
	}
	cache := make(map[string]docInfo)
	resolve := func(fname string) (docInfo, bool) {
		if d, ok := cache[fname]; ok {
			return d, true
		}
		for _, moduleID := range pkg.ModuleIDs() {
			module := pkg.Module(moduleID)
			if module == nil {
				continue
			}
			for _, docID := range module.DocumentIDs() {
				doc := module.Document(docID)
				if doc == nil {
					continue
				}
				if doc.SyntaxTree().FilePath() == fname {
					text := doc.TextDocument().String()
					info := docInfo{text: text, lineStarts: computeLineStarts(text)}
					cache[fname] = info
					return info, true
				}
			}
		}
		cache[fname] = docInfo{}
		return docInfo{}, false
	}
	extract := func(diags []diagnostics.Diagnostic, target map[string][]CompilerDiagnostic) {
		for _, diag := range diags {
			location := diag.Location()
			if !diagnostics.LocationHasSource(location) {
				continue
			}
			fname := env.FileName(location)
			info, ok := resolve(fname)
			if !ok {
				continue
			}
			d := convertDiag(info.text, info.lineStarts, diag)
			target[fname] = append(target[fname], d)
		}
	}
	extract(comp.DiagnosticResult().Diagnostics(), byFile)
	extract(comp.Resolution().DiagnosticResult().Diagnostics(), resByFile)
	resErr = comp.Resolution().DiagnosticResult().HasErrors()
	return byFile, resByFile, resErr
}

// convertDiag converts one compiler diagnostic to a CompilerDiagnostic using
// the document text and precomputed line starts.
func convertDiag(text string, lineStarts []int, diag diagnostics.Diagnostic) CompilerDiagnostic {
	location := diag.Location()
	startLine, startChar, ok := byteOffsetToLineChar(text, lineStarts, location.StartOffset())
	if !ok {
		return CompilerDiagnostic{}
	}
	endLine, endChar, ok := byteOffsetToLineChar(text, lineStarts, location.EndOffset())
	if !ok {
		return CompilerDiagnostic{}
	}
	info := diag.DiagnosticInfo()
	return CompilerDiagnostic{
		StartLine: startLine,
		StartChar: startChar,
		EndLine:   endLine,
		EndChar:   endChar,
		Severity:  toCoreSeverity(info.Severity()),
		Code:      info.Code(),
		Message:   diag.Message(),
	}
}

func byteOffsetToLineChar(text string, lineStarts []int, offset int) (uint32, uint32, bool) {
	if offset < 0 || offset > len(text) {
		return 0, 0, false
	}
	line := findLine(lineStarts, offset)
	lineStart := lineStarts[line]
	contentEnd := lineContentEnd(text, lineStart)
	column := offset - lineStart
	if column > contentEnd-lineStart {
		column = contentEnd - lineStart
	}
	if column < 0 {
		column = 0
	}
	return uint32(line), uint32(column), true
}

func toCoreSeverity(sev diagnostics.DiagnosticSeverity) Severity {
	switch sev {
	case diagnostics.Error, diagnostics.Fatal:
		return SeverityError
	case diagnostics.Warning:
		return SeverityWarning
	case diagnostics.Info:
		return SeverityInformation
	case diagnostics.Hint:
		return SeverityHint
	default:
		return SeverityError
	}
}

func findLine(lineStarts []int, offset int) int {
	for i := len(lineStarts) - 1; i >= 0; i-- {
		if offset >= lineStarts[i] {
			return i
		}
	}
	return 0
}

func lineContentEnd(text string, lineStart int) int {
	end := lineStart
	for end < len(text) && text[end] != '\r' && text[end] != '\n' {
		end++
	}
	return end
}

func computeLineStarts(text string) []int {
	starts := []int{0}
	for i := 0; i < len(text); {
		switch text[i] {
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				starts = append(starts, i+2)
				i += 2
			} else {
				starts = append(starts, i+1)
				i++
			}
		case '\n':
			starts = append(starts, i+1)
			i++
		default:
			i++
		}
	}
	return starts
}
