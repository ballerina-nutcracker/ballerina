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

// Package compile provides staged compilation with a bounded worker pool,
// one running cycle and one latest-wins pending cycle per project root.
// Accepted generations supply stable diagnostics and sealed semantic views.
package compile

import (
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// CompileRequest identifies the document to compile.
type CompileRequest struct {
	URI workspace.DocumentURI
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

// CompilerDiagnostic uses zero-based lines and byte columns, not UTF-16 code
// units. The server converts positions at the protocol boundary.
type CompilerDiagnostic struct {
	StartLine uint32
	StartChar uint32
	EndLine   uint32
	EndChar   uint32
	Severity  Severity
	Code      string
	Message   string
}

type projectReader interface {
	CurrentProject(root string) (projects.Project, uint64, bool)
	Generation(root string) (uint64, bool)
	OpenDocumentsUnder(root string) []workspace.DocumentURI
}

type cycleResult struct {
	byFile            map[string][]CompilerDiagnostic
	resByFile         map[string][]CompilerDiagnostic
	resolutionErrored bool
	descriptor        string
}

// compileCycle supplies live text, prior module state, a supersession check,
// a module-state commit callback, and the logger for dependency recovery.
// Nil callbacks disable their operations; a nil logger uses the quiet default.
type compileCycle struct {
	openText   textProvider
	priorState *packageGenState
	stale      func() bool
	commit     func(map[projects.ModuleID]moduleGenSnapshot)
	logger     *observability.Logger
}

// compileFunc compiles a package and extracts diagnostics. runCycle recovers
// panics and publishes CompilationFailed.
type compileFunc func(pkg *projects.Package, cyc compileCycle) cycleResult

// Option configures the CompilationService at construction.
type Option func(*CompilationService)

// WithLogger sets the logger used for recovered compilation panics.
// Logging is disabled by default; a nil logger leaves the current logger unchanged.
func WithLogger(logger *observability.Logger) Option {
	return func(s *CompilationService) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithMaxWorkers overrides the worker pool size.
func WithMaxWorkers(n int) Option {
	return func(s *CompilationService) {
		if n > 0 {
			s.maxWorkers = n
		}
	}
}

// WithMaxStableSnapshots overrides the stable-snapshot count bound.
func WithMaxStableSnapshots(n int) Option {
	return func(s *CompilationService) {
		if n > 0 {
			s.store.maxStable = n
		}
	}
}

// WithDebounce overrides the per-source-root debounce. Zero disables the timer;
// changes are enqueued immediately.
func WithDebounce(d time.Duration) Option {
	return func(s *CompilationService) {
		s.debounce = d
	}
}

// WithQueueDepth is reserved and has no effect. Each root has one latest-wins
// pending slot.
func WithQueueDepth(n int) Option {
	return func(s *CompilationService) {
		_ = n
	}
}

// CompilationService schedules per-project compilation and retains stable results.
type CompilationService struct {
	projects  *workspace.ProjectService
	reader    projectReader
	bus       *event.Bus
	store     *SnapshotStore
	compileFn compileFunc
	logger    *observability.Logger

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

	openText textProvider

	genStatesMu sync.Mutex
	genStates   map[string]*packageGenState

	// failedGens holds failed-generation markers checked by sealed reads;
	// guarded by cycleMu.
	failedGens map[string]uint64

	shutdownOnce sync.Once
	closed       bool
}

type cycleRequest struct {
	root    string
	gen     uint64
	pkg     *projects.Package
	project projects.Project
}

// New creates a compilation service subscribed to workspace updates and lifecycle
// events. Call Shutdown before closing the bus.
func New(projectService *workspace.ProjectService, bus *event.Bus, opts ...Option) *CompilationService {
	s := &CompilationService{
		projects:       projectService,
		reader:         projectService,
		bus:            bus,
		store:          newSnapshotStore(16, projectService.Generation),
		compileFn:      realCompilePackage,
		logger:         observability.NewNoop(),
		inFlight:       make(map[string]bool),
		pending:        make(map[string]*cycleRequest),
		debounce:       150 * time.Millisecond,
		debounceTimers: make(map[string]*time.Timer),
		debounceGens:   make(map[string]uint64),
		knownRoots:     make(map[string]struct{}),
		openText:       projectService.OpenText,
		genStates:      make(map[string]*packageGenState),
		failedGens:     make(map[string]uint64),
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

// handleLifecycle tracks known roots and clears module state and failure
// markers on registration, eviction, or kind transition. Evictions and
// transitions also remove the old root's snapshots.
func (s *CompilationService) handleLifecycle(e event.Event) {
	s.knownRootsMu.Lock()
	defer s.knownRootsMu.Unlock()
	switch e.Kind() {
	case event.ProjectRegistered:
		s.knownRoots[e.SourceRoot()] = struct{}{}
		s.evictGenState(e.SourceRoot())
		s.clearFailedGen(e.SourceRoot())
	case event.ProjectEvicted:
		delete(s.knownRoots, e.SourceRoot())
		s.store.evictRoot(e.SourceRoot())
		s.evictGenState(e.SourceRoot())
		s.clearFailedGen(e.SourceRoot())
	case event.ProjectKindTransitioned:
		if te, ok := e.(event.ProjectKindTransitionedEvent); ok {
			delete(s.knownRoots, te.OldRoot())
			s.store.evictRoot(te.OldRoot())
			s.evictGenState(te.OldRoot())
			s.clearFailedGen(te.OldRoot())
		}
		s.knownRoots[e.SourceRoot()] = struct{}{}
		s.evictGenState(e.SourceRoot())
		s.clearFailedGen(e.SourceRoot())
	default:
		return
	}
}

func (s *CompilationService) isKnown(root string) bool {
	s.knownRootsMu.Lock()
	defer s.knownRootsMu.Unlock()
	_, ok := s.knownRoots[root]
	return ok
}

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

// Debounced cycles use the latest generation recorded for the root.
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
	if s.closed || (gen != 0 && s.failedGens[root] == gen) {
		s.cycleMu.Unlock()
		return
	}
	if s.inFlight[root] {
		s.pending[root] = req
		s.cycleMu.Unlock()
		return
	}
	s.inFlight[root] = true
	s.cycleMu.Unlock()
	s.submit(req)
}

// workSem bounds concurrent compiles across roots.
func (s *CompilationService) submit(req *cycleRequest) {
	s.workSem <- struct{}{}
	go func() {
		s.runCycle(req)
		<-s.workSem
		s.finishCycle(req.root)
	}()
}

func (s *CompilationService) newCompileCycle(req *cycleRequest) compileCycle {
	root := req.root
	gen := req.gen
	return compileCycle{
		openText:   s.openText,
		priorState: s.genStateFor(root),
		logger:     s.logger,
		stale: func() bool {
			current, ok := s.reader.Generation(root)

			return !ok || current != gen
		},
		commit: func(modules map[projects.ModuleID]moduleGenSnapshot) {
			s.setGenState(root, modules)
		},
	}
}

// runCycle rejects obsolete requests, recovers compiler panics, and submits
// completed diagnostics to the stable-publication gate.
func (s *CompilationService) runCycle(req *cycleRequest) {
	if s.isClosed() {
		return
	}
	current, ok := s.reader.Generation(req.root)
	if !ok || req.gen != current {
		s.bus.Publish(event.NewCompilationCancelledEvent(req.root, req.gen))
		return
	}
	if s.generationFailed(req.root, req.gen) {
		return
	}
	done := make(chan struct{})
	s.store.setInProgress(req.root, InProgressSnapshot{key: CompilationKey{req.root, req.gen}, done: done})
	defer func() {
		close(done)
		s.store.clearInProgress(req.root)
	}()

	cyc := s.newCompileCycle(req)
	var result cycleResult
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				s.markGenerationFailed(req.root, req.gen)
				s.logger.Error("compile: recovered panic", "recovered", r, "root", req.root, "generation", req.gen, "stack", string(debug.Stack()))
				s.bus.Publish(event.NewCompilationFailedEvent(req.root, req.gen))
			}
		}()
		result = s.compileFn(req.pkg, cyc)
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

// finishCycle runs after releasing the worker slot. The root stays in flight
// until its latest pending cycle finishes.
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

// Flush drains incoming events, waits for queued and running compiles, then
// drains outgoing events. It does not wait for armed debounce timers.
func (s *CompilationService) Flush() {

	if s.bus != nil {
		s.bus.Flush()
	}

	s.cycleMu.Lock()
	for !s.allIdleLocked() {
		s.idleCond.Wait()
	}
	s.cycleMu.Unlock()

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

// Shutdown stops accepting cycles and stops debounce timers, waits for the
// worker pool to drain, flushes the bus, and clears snapshots. It is idempotent.
func (s *CompilationService) Shutdown() {
	s.shutdownOnce.Do(func() {
		s.cycleMu.Lock()
		s.closed = true
		s.cycleMu.Unlock()
		s.stopDebounceTimers()

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

// Cancel supersedes every root with a running or pending cycle. Cycles check
// supersession at compilation boundaries and before stable publication.
// Debounce timers and the event bus are unchanged.
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

// DiagnosticsFor returns the latest stable diagnostics for open documents and
// the snapshot's generation, or ok=false if none exists. Callers must check
// generation freshness before publishing.
func (s *CompilationService) DiagnosticsFor(root string) (diags map[workspace.DocumentURI][]CompilerDiagnostic, generation uint64, ok bool) {
	snap, has := s.store.Stable(root)
	if !has {
		return nil, 0, false
	}
	out := make(map[workspace.DocumentURI][]CompilerDiagnostic)
	for _, u := range s.reader.OpenDocumentsUnder(root) {
		if name, ok := s.projects.Paths(snap.project).NameOf(u); ok {
			out[u] = snap.diagsForFile(name)
		}
	}
	return out, snap.key.Generation, true
}

// realCompilePackage drives all modules, commits module state unless aborted,
// and extracts file diagnostics plus the package-resolution diagnostics.
// An aborted run returns an empty result without committing.
func realCompilePackage(pkg *projects.Package, cyc compileCycle) cycleResult {
	descriptor := pkg.Descriptor().Name().Value()
	compEnv := pkg.Project().Environment().CompilerEnvironment()

	pd := newPackageDriver(pkg, cyc.openText, cyc.priorState, cyc.stale)
	if cyc.logger != nil {
		pd.logger = cyc.logger
	}
	pd.advanceAll(stageCFGAnalyzed)
	if pd.aborted() {
		return cycleResult{}
	}
	if cyc.commit != nil {
		cyc.commit(pd.snapshotForCommit())
	}

	resolution := pkg.Resolution().DiagnosticResult()
	byFile, resByFile := extractByFile(pkg, pd.allDiagnostics(), resolution.Diagnostics(), compEnv.DiagnosticEnv(), cyc.openText)
	return cycleResult{
		byFile:            byFile,
		resByFile:         resByFile,
		resolutionErrored: resolution.HasErrors(),
		descriptor:        descriptor,
	}
}

type docInfo struct {
	text       string
	lineStarts []int
}

func buildDocIndex(pkg *projects.Package, openText textProvider) map[string]docInfo {
	idx := make(map[string]docInfo)
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
			text := documentText(module, doc, openText)
			key := moduleFileRegistrationKey(module, doc.Name())
			idx[key] = docInfo{text: text, lineStarts: computeLineStarts(text)}
		}
	}
	return idx
}

// extractByFile groups diagnostics by compiler registration name using pkg's
// documents and live text. Diagnostics outside those documents are skipped.
func extractByFile(pkg *projects.Package, compileDiags, resDiags []diagnostics.Diagnostic, env *diagnostics.DiagnosticEnv, openText textProvider) (byFile, resByFile map[string][]CompilerDiagnostic) {
	byFile = make(map[string][]CompilerDiagnostic)
	resByFile = make(map[string][]CompilerDiagnostic)
	idx := buildDocIndex(pkg, openText)
	extract := func(diags []diagnostics.Diagnostic, target map[string][]CompilerDiagnostic) {
		for _, diag := range diags {
			location := diag.Location()
			if !diagnostics.LocationHasSource(location) {
				continue
			}
			fname := env.FileName(location)
			info, ok := idx[fname]
			if !ok {
				continue
			}
			target[fname] = append(target[fname], convertDiag(info.text, info.lineStarts, diag))
		}
	}
	extract(compileDiags, byFile)
	extract(resDiags, resByFile)
	return byFile, resByFile
}

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
