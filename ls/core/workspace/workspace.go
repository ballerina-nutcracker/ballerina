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

// Package workspace provides ProjectService, the core service that owns
// document lifecycle state (open/change/close) and the source-root-keyed
// project index. Apply carries resolved full text — the server resolves
// protocol.TextEdit ranges to full text before calling Apply, keeping this
// package protocol-free.
//
// Ticket 08 replaces the Phase-A single-file synthetic overlayFS (owned by
// compile) with a PAL-backed, overlay-augmented io/fs.FS owned here: the
// project index resolves file: URIs to source roots (ADR-048 three-step),
// projects.Load reads through palFS, and Apply publishes atomically through
// the compiler's immutable modifier chain. A synchronous event bus carries
// project-lifecycle events.
package workspace

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// defaultMaxProjects is the conservative Phase-B count bound for the project
// index. ADR-057's weighted 2048MB budget is deferred.
const defaultMaxProjects = 32

// Option configures a ProjectService at construction.
type Option func(*ProjectService)

// WithMaxProjects overrides the project index count bound.
func WithMaxProjects(n int) Option {
	return func(s *ProjectService) {
		if n > 0 {
			s.index.maxProjects = n
		}
	}
}

// ChangeKind discriminates the kind of DocumentChange.
type ChangeKind uint8

const (
	// ChangeOpen opens a new document, storing its full text and version.
	ChangeOpen ChangeKind = iota
	// ChangeUpdate replaces the text of an existing document with a new
	// resolved full text, subject to version monotonicity.
	ChangeUpdate
	// ChangeClose removes a document from the store.
	ChangeClose
)

// DocumentChange describes a single document lifecycle mutation. The server
// resolves protocol.TextEdit ranges to full Text before constructing a
// DocumentChange, so this type carries no protocol types.
type DocumentChange struct {
	Kind       ChangeKind
	URI        uri.DocumentURI
	Text       string // valid for ChangeOpen and ChangeUpdate
	Version    int32  // valid for ChangeOpen and ChangeUpdate
	LanguageID string // valid for ChangeOpen
}

// Snapshot is a plain-struct, value-type view of a document's current state.
// No refcount or Release method ships in Phase B — there is no refcount state.
// Ticket 09 wraps Snapshot with a release func() when the dual-snapshot engine
// adds refcounting. SourceRoot is set on Apply for observability/testability.
// Generation (ticket 09) is the per-source-root monotonic counter of the
// publish that produced this snapshot.
type Snapshot struct {
	Text       string
	Version    int32
	LanguageID string
	SourceRoot string
	Generation uint64
}

// WatchedFileKind discriminates a watched-file change.
type WatchedFileKind uint8

const (
	// WatchedFileCreate is a file created on disk.
	WatchedFileCreate WatchedFileKind = iota
	// WatchedFileDelete is a file deleted from disk.
	WatchedFileDelete
	// WatchedFileModified is a file whose content changed on disk.
	WatchedFileModified
)

// WatchedFileChange describes a didChangeWatchedFiles event routed to the
// workspace.
type WatchedFileChange struct {
	Kind WatchedFileKind
	URI  uri.DocumentURI
}

// ProjectService owns document lifecycle state and the source-root-keyed
// project index. It absorbs the former server.documentStore, replacing the
// string-keyed map with a DocumentURI-keyed map, and replaces the compile-owned
// synthetic overlayFS with PAL-backed project loading and modifier-chain
// publication.
type ProjectService struct {
	platform  pal.Platform
	bus       *event.Bus
	mu        sync.RWMutex // guards the documents map (concurrent engine reads under 09)
	documents map[uri.DocumentURI]Snapshot
	index     *projectIndex
	now       func() time.Time
	stat      func(string) (fs.FileInfo, error)
}

// New creates a ProjectService wired to the given PAL platform and synchronous
// event bus. The bus receives ProjectRegistered/ProjectEvicted/
// ProjectKindTransitioned events. Options configure the index bound.
func New(platform pal.Platform, bus *event.Bus, opts ...Option) *ProjectService {
	now := time.Now
	s := &ProjectService{
		platform:  platform,
		bus:       bus,
		documents: make(map[uri.DocumentURI]Snapshot),
		index:     newProjectIndex(defaultMaxProjects, now),
		now:       now,
		stat:      platform.FS.Stat,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// applyDocumentMap carries the 07 logic: version monotonicity checking and text
// accumulation in the documents map.
func (s *ProjectService) applyDocumentMap(change DocumentChange) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch change.Kind {
	case ChangeOpen:
		snap := Snapshot{
			Text:       change.Text,
			Version:    change.Version,
			LanguageID: change.LanguageID,
		}
		s.documents[change.URI] = snap
		return snap, nil
	case ChangeUpdate:
		current, ok := s.documents[change.URI]
		if !ok {
			return Snapshot{}, fmt.Errorf("workspace: document not open: %s", change.URI)
		}
		if change.Version <= current.Version {
			return Snapshot{}, fmt.Errorf("workspace: stale version %d <= current %d", change.Version, current.Version)
		}
		snap := Snapshot{
			Text:       change.Text,
			Version:    change.Version,
			LanguageID: current.LanguageID,
		}
		s.documents[change.URI] = snap
		return snap, nil
	case ChangeClose:
		if _, ok := s.documents[change.URI]; !ok {
			return Snapshot{}, fmt.Errorf("workspace: document not open: %s", change.URI)
		}
		delete(s.documents, change.URI)
		return Snapshot{}, nil
	default:
		return Snapshot{}, fmt.Errorf("workspace: unknown change kind %d", change.Kind)
	}
}

// Apply applies a document lifecycle change: update the documents map,
// resolve the source root (memoized), and — for open/update — load a fresh
// project through palFS so the open-buffer overlay is authoritative, then
// atomically replace the index entry. context.Context declares that
// cancellation flows through ctx; it is checked for Err() before publication.
// Phase B's calls are synchronous (no goroutines).
//
// The project is reloaded per publication rather than reusing a cached
// project's environment: the compiler's DiagnosticEnv is a per-Load singleton
// that panics on re-registration of a changed file (a documented latent
// limitation — see the skipped TestProjectDuplicate). The index still provides
// source-root memoization (the expensive ADR-048 walk) and count-bounding.
// The ADR-042 modifier chain (Document.Modify().WithContent().Apply() →
// setCurrentPackage) is the deferred publication model, restored by a later
// ticket that lifts DiagnosticEnv to per-instance file identity.
func (s *ProjectService) Apply(ctx context.Context, change DocumentChange) (Snapshot, error) {
	if change.Kind == ChangeClose {
		// Resolve first (the document is still open) to find the index entry,
		// then remove from the map and decrement the open-doc count. No
		// publication on close — the document stays in the package until
		// eviction or a watched-file delete.
		sourceRoot, err := s.findSourceRootMemoized(change.URI)
		if err == nil {
			if entry, ok := s.index.get(sourceRoot); ok && entry.openDocs > 0 {
				entry.openDocs--
			}
		}
		snap, err := s.applyDocumentMap(change)
		return snap, err
	}
	snap, err := s.applyDocumentMap(change)
	if err != nil {
		return snap, err
	}
	sourceRoot, err := s.findSourceRootMemoized(change.URI)
	if err != nil {
		return snap, nil // source root unresolved — document map still updated
	}
	if change.Kind == ChangeOpen {
		if entry, ok := s.index.get(sourceRoot); ok {
			entry.openDocs++
		}
	}
	snap.SourceRoot = sourceRoot
	if err := ctx.Err(); err != nil {
		return snap, fmt.Errorf("workspace: cancelled before publication: %w", err)
	}
	gen, err := s.publish(change, sourceRoot)
	if err != nil {
		return snap, err
	}
	snap.Generation = gen
	// Persist SourceRoot + Generation into the documents-map view so
	// OpenDocumentsUnder and Snapshot reflect the publish.
	s.mu.Lock()
	if cur, ok := s.documents[change.URI]; ok {
		cur.SourceRoot = sourceRoot
		cur.Generation = gen
		s.documents[change.URI] = cur
	}
	s.mu.Unlock()
	return snap, nil
}

// publish is the ADR-042 modifier-chain publication model (ticket 09 branch 1).
// The first publish for a source root does one projects.Load to seed the
// persistent per-source-root CompilerEnvironment and initial package, and
// publishes ProjectRegistered (08 behavior) + ProjectUpdated. Subsequent
// content publishes reuse the SAME project via Document.Modify().WithContent()
// .Apply() so the shared CompilerEnvironment/DiagnosticEnv persists across
// generations (symbol-Location stability), bumping the generation and
// publishing ProjectUpdated. If the document is not yet in the package (a new
// file opened after the seed load), it falls back to a fresh Load.
func (s *ProjectService) publish(change DocumentChange, sourceRoot string) (uint64, error) {
	s.mu.Lock()
	entry, ok := s.index.get(sourceRoot)
	if !ok || entry.project == nil {
		s.mu.Unlock()
		s.reloadAt(sourceRoot, change.URI, 0)
		s.publishLifecycle(sourceRoot, 1)
		return 1, nil
	}
	if s.applyModifierChain(entry.project, change) {
		entry.generation++
		gen := entry.generation
		s.mu.Unlock()
		s.publishUpdated(sourceRoot, gen)
		return gen, nil
	}
	oldGen := entry.generation
	s.mu.Unlock()
	// Document not in the current package (new file): fall back to a fresh
	// Load, advancing the generation.
	s.reloadAt(sourceRoot, change.URI, oldGen)
	s.publishLifecycle(sourceRoot, oldGen+1)
	return oldGen + 1, nil
}

// applyModifierChain updates the document content through the immutable
// modifier chain on the persistent project, which sets a new current package
// on the same project (and thus the same CompilerEnvironment). It returns false
// if the document is not present in the current package (caller falls back to
// a fresh Load).
func (s *ProjectService) applyModifierChain(project projects.Project, change DocumentChange) bool {
	pkg := project.CurrentPackage()
	if pkg == nil {
		return false
	}
	docID, ok := project.DocumentID(change.URI.Path())
	if !ok {
		return false
	}
	module := pkg.Module(docID.ModuleID())
	if module == nil {
		return false
	}
	doc := module.Document(docID)
	if doc == nil {
		return false
	}
	doc.Modify().WithContent(change.Text).Apply()
	return true
}

// publishLifecycle publishes ProjectRegistered (first load / kind transition,
// 08 behavior) followed by ProjectUpdated (WM-E4) for the given generation.
func (s *ProjectService) publishLifecycle(sourceRoot string, gen uint64) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(event.NewProjectRegisteredEvent(sourceRoot))
	s.bus.Publish(event.NewProjectUpdatedEvent(sourceRoot, gen))
}

// publishUpdated publishes ProjectUpdated (WM-E4) for the given generation.
func (s *ProjectService) publishUpdated(sourceRoot string, gen uint64) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(event.NewProjectUpdatedEvent(sourceRoot, gen))
}

// reloadAt builds a fresh palFS for the source root, loads a brand-new project
// (fresh CompilerEnvironment and DiagnosticEnv), and atomically replaces the
// index entry. The open-doc count and the generation (advanced from prevGen)
// are preserved from any existing entry. A load failure is non-fatal: the
// documents map is still updated, and the next change retries.
func (s *ProjectService) reloadAt(sourceRoot string, u uri.DocumentURI, prevGen uint64) {
	fsys := s.buildPalFS(sourceRoot)
	project, err := s.loadProject(fsys, sourceRoot, u.Path())
	if err != nil || project == nil {
		return
	}
	entry := &indexEntry{
		project:     project,
		sourceRoot:  sourceRoot,
		lastTouched: s.now(),
		generation:  prevGen + 1,
	}
	if old, ok := s.index.get(sourceRoot); ok {
		entry.openDocs = old.openDocs
	}
	s.index.putExisting(entry, s.bus)
}

// Generation returns the per-source-root monotonic generation counter for the
// last accepted publish, or 0/false if the root is unknown.
func (s *ProjectService) Generation(root string) (uint64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.index.peek(root)
	if !ok {
		return 0, false
	}
	return entry.generation, true
}

// Supersede bumps the source root's generation counter without publishing, so
// any in-flight compile cycle for root is gated out at the stale-publication
// gate (CE-E3): the running compile finishes but its result is discarded. It
// is the $/cancelRequest mapping (design branch 3) — Go cannot interrupt a
// compile, so cancellation is boundary-only via the generation counter. It is
// a no-op for an unknown root and does not touch the debounce timers (owned
// by the compile engine) or close the bus.
func (s *ProjectService) Supersede(root string) {
	s.index.supersede(root)
}

// CurrentProject returns the published project for a source root plus its
// generation. Used by the compile engine to capture the package to compile.
// The project's current package is read under the state lock so it cannot
// race with a concurrent Apply's modifier-chain swap.
func (s *ProjectService) CurrentProject(root string) (projects.Project, uint64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.index.peek(root)
	if !ok || entry.project == nil {
		return nil, 0, false
	}
	return entry.project, entry.generation, true
}

// OpenDocumentsUnder returns the open file: DocumentURIs resolved to root, in
// stable (URI-sorted) iteration order. Used by the server CE subscriber
// (branch 10) to publish diagnostics for every open document in the accepted
// root.
func (s *ProjectService) OpenDocumentsUnder(root string) []uri.DocumentURI {
	s.mu.RLock()
	var open []uri.DocumentURI
	for u := range s.documents {
		if u.IsFile() {
			open = append(open, u)
		}
	}
	s.mu.RUnlock()
	var out []uri.DocumentURI
	for _, u := range open {
		sr, ok := s.index.lookupSourceRoot(u.Path())
		if !ok {
			sr2, err := s.findSourceRoot(u.Path())
			if err != nil {
				continue
			}
			sr = sr2
		}
		if sr == root {
			out = append(out, u)
		}
	}
	sortDocumentURIs(out)
	return out
}

// findSourceRootMemoized is the ADR-048 step 1 for a file: URI: find the
// project source root, memoized by file path. It is file:-only in 08.
func (s *ProjectService) findSourceRootMemoized(u uri.DocumentURI) (string, error) {
	filePath := u.Path()
	if root, ok := s.index.lookupSourceRoot(filePath); ok {
		return root, nil
	}
	root, err := s.findSourceRoot(filePath)
	if err != nil {
		return "", err
	}
	s.index.memoSourceRoot(filePath, root)
	return root, nil
}

// findSourceRoot walks up from filePath looking for Ballerina.toml (build/
// workspace project); on no hit for a .bal file it returns the file's
// directory (single-file project root). It is the expensive step and is
// memoized by resolve.
func (s *ProjectService) findSourceRoot(filePath string) (string, error) {
	dir := path.Dir(filePath)
	for {
		tomlPath := path.Join(dir, projects.BallerinaTomlFile)
		if info, err := s.stat(tomlPath); err == nil && !info.IsDir() {
			return dir, nil
		}
		parent := path.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// No Ballerina.toml ancestor: single-file project rooted at the file's
	// directory (matching loadSingleFileProject's sourceDir = path.Dir).
	return path.Dir(filePath), nil
}

// buildPalFS constructs an overlay-augmented filesystem carrying the current
// open-buffer contents for files under sourceRoot. Open buffers win on
// ReadFile/Stat so a single-file non-disk URI (and overlay-over-disk edits)
// load from the buffer, not disk.
func (s *ProjectService) buildPalFS(sourceRoot string) palFS {
	overlays := make(map[string][]byte)
	s.mu.RLock()
	for u, snap := range s.documents {
		if !u.IsFile() {
			continue
		}
		p := u.Path()
		if p == sourceRoot || isAncestor(sourceRoot, p) {
			overlays[p] = []byte(snap.Text)
		}
	}
	s.mu.RUnlock()
	return palFS{pal: s.platform.FS, overlays: overlays, now: s.now}
}

// loadProject loads the project at sourceRoot through palFS. For a
// single-file source root (no Ballerina.toml), it loads the single .bal file;
// the overlay lets a non-disk file load from its buffer.
func (s *ProjectService) loadProject(fsys palFS, sourceRoot, filePath string) (projects.Project, error) {
	// A single-file root: no Ballerina.toml at sourceRoot → load the file
	// directly (palFS overlays the buffer for a non-disk URI).
	tomlPath := path.Join(sourceRoot, projects.BallerinaTomlFile)
	if info, err := fsys.Stat(tomlPath); err != nil || info.IsDir() {
		result, err := projects.Load(fsys, filePath)
		if err != nil {
			return nil, err
		}
		return result.Project(), nil
	}
	result, err := projects.Load(fsys, sourceRoot)
	if err != nil {
		return nil, err
	}
	return result.Project(), nil
}

// Project returns the resolved project for a file: URI from the index. It is
// the exported accessor CompilationService uses to read the published
// CurrentPackage via a direct reference. It does not auto-load: a source root
// that was never Apply'd (or was evicted) returns nil, so Compile returns an
// empty result.
func (s *ProjectService) Project(u uri.DocumentURI) (projects.Project, error) {
	sourceRoot, err := s.findSourceRootMemoized(u)
	if err != nil {
		return nil, err
	}
	entry, ok := s.index.get(sourceRoot)
	if !ok {
		return nil, nil
	}
	return entry.project, nil
}

// Snapshot returns the current snapshot for the given URI, replacing the
// former documentStore.document lookup.
func (s *ProjectService) Snapshot(u uri.DocumentURI) (Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.documents[u]
	return snap, ok
}

// ApplyWatchedFile routes a didChangeWatchedFiles event. Ballerina.toml
// create/delete/change triggers an atomic ADR-024 kind transition (or reload);
// a source-file delete while not open drops the stale index entry so the next
// didOpen/didChange reloads fresh from disk (no modifier chain on the cached
// project, avoiding the DiagnosticEnv same-name/different-doc panic).
func (s *ProjectService) ApplyWatchedFile(ctx context.Context, change WatchedFileChange) error {
	_ = ctx
	if !change.URI.IsFile() {
		return nil
	}
	filePath := change.URI.Path()
	base := path.Base(filePath)
	switch base {
	case projects.BallerinaTomlFile:
		return s.applyBallerinaTomlChange(change.Kind, filePath)
	default:
		return s.applySourceFileChange(change.Kind, filePath)
	}
}

// applyBallerinaTomlChange handles create/delete/change of a Ballerina.toml:
// atomically invalidate the source-root memo, re-resolve, and reload the
// project fresh so the new kind (single-file ↔ build) or new manifest takes
// effect. Create/delete publish ProjectKindTransitioned when the source root
// identity is unchanged, or ProjectEvicted(oldRoot)+ProjectRegistered(newRoot)
// when the root identity changes (ADR-024). A content change publishes
// ProjectRegistered (a manifest reload is a replacement). With no open doc
// under dir, only the memo is invalidated; the next didOpen reloads the new kind.
func (s *ProjectService) applyBallerinaTomlChange(kind WatchedFileKind, tomlPath string) error {
	dir := path.Dir(tomlPath)
	anchor, oldRoot, found := s.findAnchorUnder(dir)
	s.index.invalidateUnder(dir)
	if !found {
		return nil
	}
	newRoot, err := s.findSourceRootMemoized(anchor)
	if err != nil {
		return nil
	}
	prevGen := uint64(0)
	if oldEntry, ok := s.index.get(oldRoot); ok {
		prevGen = oldEntry.generation
	}
	s.reloadAt(newRoot, anchor, prevGen)
	if s.bus == nil {
		return nil
	}
	newGen := prevGen + 1
	if kind == WatchedFileCreate || kind == WatchedFileDelete {
		if oldRoot == newRoot {
			s.bus.Publish(event.NewProjectKindTransitionedEvent(oldRoot, newRoot))
		} else {
			s.bus.Publish(event.NewProjectEvictedEvent(oldRoot, event.EvictionKindTransition))
			s.bus.Publish(event.NewProjectRegisteredEvent(newRoot))
		}
	} else {
		s.bus.Publish(event.NewProjectRegisteredEvent(newRoot))
	}
	s.bus.Publish(event.NewProjectUpdatedEvent(newRoot, newGen))
	return nil
}

// findAnchorUnder returns an open document under dir (or equal to dir) to
// anchor a kind-transition reload, plus the source root that doc resolved to
// before the transition. The anchor is the first matching open file: URI; the
// old root is taken from the filePath→sourceRoot memo (falling back to dir for
// a single-file doc that was never memoized).
func (s *ProjectService) findAnchorUnder(dir string) (uri.DocumentURI, string, bool) {
	s.mu.RLock()
	var found uri.DocumentURI
	foundOK := false
	var oldRoot string
	for u := range s.documents {
		if !u.IsFile() {
			continue
		}
		p := u.Path()
		if p == dir || isAncestor(dir, p) {
			r, ok := s.index.lookupSourceRoot(p)
			if !ok {
				r = dir
			}
			found = u
			oldRoot = r
			foundOK = true
			break
		}
	}
	s.mu.RUnlock()
	return found, oldRoot, foundOK
}

// applySourceFileChange handles create/change/delete of a .bal file. In 08:
// a deleted on-disk file that is not open drops the stale index entry so the
// next didOpen/didChange reloads fresh from disk (the fresh Load excludes the
// deleted file). Create/change on disk need no action — the next Load/Reload
// picks them up via ReadDir. A file that is still open keeps its overlay; the
// document stays in the package until close. This avoids the modifier chain
// (and the DiagnosticEnv same-name/different-doc panic) on the cached project.
func (s *ProjectService) applySourceFileChange(kind WatchedFileKind, filePath string) error {
	if kind != WatchedFileDelete {
		return nil
	}
	s.mu.RLock()
	open := false
	for u := range s.documents {
		if u.IsFile() && u.Path() == filePath {
			open = true
			break
		}
	}
	s.mu.RUnlock()
	if open {
		return nil // still open — overlay keeps content; no action in 09
	}
	sourceRoot, ok := s.index.lookupSourceRoot(filePath)
	if !ok {
		return nil
	}
	s.index.remove(sourceRoot)
	return nil
}

// Shutdown marks every known source root's generation superseded so in-flight
// compile cycles gate their results out at the stale-publication gate (CE-E3).
// It does NOT close the event bus — the wiring closes the bus last, after the
// compile engine's Shutdown has drained its worker pool and bus.Flush()ed the
// CRITICAL delivery channels — and it does NOT stop the debounce timers,
// which are owned by the compile engine (CompilationService). This keeps the
// branch-9 shutdown ordering (workspace supersede → compile drain+Flush → bus
// Close) and the engine-owned-timer boundary accurate.
func (s *ProjectService) Shutdown(ctx context.Context) error {
	_ = ctx
	s.index.supersedeAll()
	return nil
}
