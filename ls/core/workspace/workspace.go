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
// Project discovery and loading use a host-supplied directory opener. Open
// editor buffers overlay folder-relative source names at load time; subsequent
// content edits flow through OpenText without rereading source files.
package workspace

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
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

// WithRepositories configures package repositories used when the workspace
// loads projects. Normal construction leaves repositories unset, retaining
// projects.Load's default repository universe.
func WithRepositories(repositories []projects.Repository) Option {
	return func(s *ProjectService) {
		s.repositories = repositories
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
	URI        DocumentURI
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
	URI  DocumentURI
}

// ProjectService owns document lifecycle state and the source-root-keyed
// project index. It absorbs the former server.documentStore, replacing the
// string-keyed map with a DocumentURI-keyed map, and replaces the compile-owned
// synthetic filesystem with folder-relative project loading and in-place
// publication (ticket 28).
type ProjectService struct {
	openDir      func(string) fs.FS
	bus          *event.Bus
	mu           sync.RWMutex // guards the documents map (concurrent engine reads under 09)
	documents    map[DocumentURI]Snapshot
	pendingRoots map[DocumentURI]string // recovery membership only; guarded by mu
	index        *projectIndex
	now          func() time.Time
	repositories []projects.Repository
}

// New creates a ProjectService wired to a directory opener and synchronous
// event bus. The bus receives ProjectRegistered/ProjectEvicted/
// ProjectKindTransitioned events. Options configure the index bound.
func New(openDir func(string) fs.FS, bus *event.Bus, opts ...Option) *ProjectService {
	now := time.Now
	s := &ProjectService{
		openDir:      openDir,
		bus:          bus,
		documents:    make(map[DocumentURI]Snapshot),
		pendingRoots: make(map[DocumentURI]string),
		index:        newProjectIndex(defaultMaxProjects, now),
		now:          now,
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
		delete(s.pendingRoots, change.URI)
		return Snapshot{}, nil
	default:
		return Snapshot{}, fmt.Errorf("workspace: unknown change kind %d", change.Kind)
	}
}

// Apply applies a document lifecycle change: update the documents map,
// resolve the source root (memoized), and — for open/update — load a fresh
// project through the source overlay so open buffers are authoritative, then
// atomically replace the index entry. context.Context declares that
// cancellation flows through ctx; it is checked for Err() before publication.
// Phase B's calls are synchronous (no goroutines).
//
// The project is reloaded per publication rather than reusing a cached
// project's environment: the compiler's DiagnosticEnv is a per-Load singleton
// that panics on re-registration of a changed file (a documented latent
// limitation — see the skipped TestProjectDuplicate). The index still provides
// source-root memoization (the expensive ADR-048 walk) and count-bounding.
// Ticket 28 drops the former ADR-042 modifier chain (Document.Modify()
// .WithContent().Apply() → setCurrentPackage): edits publish in place (see
// publish/documentInCurrentPackage) and the compile package's incremental
// moduleDriver reset picks up the new content via OpenText instead.
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
		return snap, err
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
		if gen != 0 {
			delete(s.pendingRoots, change.URI)
		}
	}
	s.mu.Unlock()
	return snap, nil
}

// publish is ticket 28's incremental publication model: the first publish for
// a source root does one projects.Load to seed the persistent per-source-root
// CompilerEnvironment and initial package, and publishes ProjectRegistered (08
// behavior) + ProjectUpdated. Subsequent content publishes reuse the SAME
// project and the SAME immutable *projects.Package — the document's new text
// already landed in the documents map (applyDocumentMap, before publish is
// called), which OpenText exposes as the live-buffer override the compile
// package's driver reads instead of the frozen Document.TextDocument(). This
// drops the ADR-042 modifier chain (Document.Modify().WithContent().Apply(),
// which used to cascade a brand-new Package/Module/Document graph on every
// edit): the compiler's Package graph now only changes on a genuine
// structural reload (a new/removed file), triggering the incremental
// moduleDriver reset (ls/core/compile) for the affected module(s) instead of
// a full from-scratch rebuild. If the document is not yet in the package (a
// new file opened after the seed load), it falls back to a fresh Load.
func (s *ProjectService) publish(change DocumentChange, sourceRoot string) (uint64, error) {
	s.mu.Lock()
	entry, ok := s.index.get(sourceRoot)
	if !ok || entry.project == nil {
		s.mu.Unlock()
		gen := s.reloadAt(sourceRoot, change.URI)
		s.publishLifecycle(sourceRoot, gen)
		return gen, nil
	}
	if s.documentInCurrentPackage(entry.project, change) && !s.importSetChanged(entry.project, change) {
		gen, _ := s.index.advance(sourceRoot)
		s.mu.Unlock()
		s.publishUpdated(sourceRoot, gen)
		return gen, nil
	}
	s.mu.Unlock()
	// Document not in the current package (new file): fall back to a fresh
	// Load, advancing the generation.
	gen := s.reloadAt(sourceRoot, change.URI)
	s.publishLifecycle(sourceRoot, gen)
	return gen, nil
}

// documentInCurrentPackage reports whether change's document is already part
// of project's current package. It performs no mutation — content updates
// live only in the documents map (see OpenText) — this is purely the
// new-file-detection gate that decides whether publish can bump the
// generation in place or must fall back to a fresh Load.
func (s *ProjectService) documentInCurrentPackage(project projects.Project, change DocumentChange) bool {
	pkg := project.CurrentPackage()
	if pkg == nil {
		return false
	}
	name, contained := s.Paths(project).NameOf(change.URI)
	if !contained {
		return false
	}
	docID, ok := project.DocumentID(name)
	if !ok {
		return false
	}
	module := pkg.Module(docID.ModuleID())
	if module == nil {
		return false
	}
	return module.Document(docID) != nil
}

// importSetChanged reports whether change's live text declares a different
// import set than the same document in the project's current package. A
// package resolves its dependencies once from the text it was loaded with, so
// a changed import set needs a fresh Load. Only syntactically complete import
// declarations count, so a half-typed import does not reload.
func (s *ProjectService) importSetChanged(project projects.Project, change DocumentChange) bool {
	name, contained := s.Paths(project).NameOf(change.URI)
	if !contained {
		return false
	}
	docID, ok := project.DocumentID(name)
	if !ok {
		return false
	}
	doc := project.CurrentPackage().Module(docID.ModuleID()).Document(docID)
	live, ok := parseImportSet(change.Text)
	if !ok {
		return false
	}
	return !sameImportSet(live, importSet(doc.SyntaxTree()))
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

// reloadAt builds a fresh source overlay for the root, loads a brand-new project
// (fresh CompilerEnvironment and DiagnosticEnv), and atomically replaces the
// index entry with a fresh lifetime generation stamp. The open-doc count is
// preserved from any existing entry. A load failure is non-fatal: the
// documents map is still updated, and the next change retries.
func (s *ProjectService) reloadAt(sourceRoot string, u DocumentURI) uint64 {
	paths := SourcePaths{root: sourceRoot}
	loadName := "."
	if sourceRoot == u.Path() {
		paths.root = filepath.Dir(sourceRoot)
		loadName = filepath.Base(sourceRoot)
	}
	fsys := s.buildOverlay(paths)
	result, err := projects.Load(fsys, loadName, projects.ProjectLoadConfig{Repositories: s.repositories})
	project := result.Project()
	if err != nil || project == nil {
		return 0
	}
	if root := project.SourceRoot(); root != "" && root != "." {
		paths.prefix = root + "/"
	}
	entry := &indexEntry{
		project:     project,
		paths:       paths,
		sourceRoot:  sourceRoot,
		lastTouched: s.now(),
	}
	if old, ok := s.index.get(sourceRoot); ok {
		entry.openDocs = old.openDocs
	}
	return s.index.putExisting(entry, s.bus)
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
// race with a concurrent Apply's index-entry swap (a structural reload).
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
func (s *ProjectService) OpenDocumentsUnder(root string) []DocumentURI {
	s.mu.RLock()
	var open []DocumentURI
	for u := range s.documents {
		if u.IsFile() {
			open = append(open, u)
		}
	}
	s.mu.RUnlock()
	var out []DocumentURI
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
func (s *ProjectService) findSourceRootMemoized(u DocumentURI) (string, error) {
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

// findSourceRoot returns a package directory or the distinct single-file key.
func (s *ProjectService) findSourceRoot(filePath string) (string, error) {
	root, found, err := projects.FindProjectRoot(s.openDir, filePath)
	if err != nil {
		return "", err
	}
	if found {
		return root, nil
	}
	return filePath, nil
}

func (s *ProjectService) buildOverlay(paths SourcePaths) overlayFS {
	overlays := make(map[string][]byte)
	s.mu.RLock()
	for u, snap := range s.documents {
		if !u.IsFile() {
			continue
		}
		if name, ok := (SourcePaths{root: paths.root}).NameOf(u); ok {
			overlays[name] = []byte(snap.Text)
		}
	}
	s.mu.RUnlock()
	return overlayFS{base: s.openDir(paths.root), overlays: overlays, now: s.now}
}

// ProjectKey returns workspace identity, not the project's relative load name.
func (s *ProjectService) ProjectKey(u DocumentURI) (string, error) {
	return s.findSourceRootMemoized(u)
}

// Paths captures the name conversion metadata for a loaded project.
func (s *ProjectService) Paths(project projects.Project) SourcePaths {
	return s.index.paths(project)
}

// Project returns the resolved project for a file: URI from the index. It is
// the exported accessor CompilationService uses to read the published
// CurrentPackage via a direct reference. It does not auto-load: a source root
// that was never Apply'd (or was evicted) returns nil, so Compile returns an
// empty result.
func (s *ProjectService) Project(u DocumentURI) (projects.Project, error) {
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
func (s *ProjectService) Snapshot(u DocumentURI) (Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.documents[u]
	return snap, ok
}

// DocumentText returns u's current text: the live open-buffer content if the
// document is open, otherwise the loaded project's frozen document text (the
// same content the compile engine's documentText fallback reads for a
// document that was never opened through the workspace). A document whose
// root is unknown or not yet loaded reports ok=false.
func (s *ProjectService) DocumentText(u DocumentURI) (string, bool) {
	if snap, ok := s.Snapshot(u); ok {
		return snap.Text, true
	}
	project, err := s.Project(u)
	if err != nil || project == nil {
		return "", false
	}
	name, contained := s.Paths(project).NameOf(u)
	if !contained {
		return "", false
	}
	docID, ok := project.DocumentID(name)
	if !ok {
		return "", false
	}
	pkg := project.CurrentPackage()
	if pkg == nil {
		return "", false
	}
	module := pkg.Module(docID.ModuleID())
	if module == nil {
		return "", false
	}
	doc := module.Document(docID)
	if doc == nil {
		return "", false
	}
	return doc.TextDocument().String(), true
}

// OpenText resolves a compiler name within its owning project to a live
// editor buffer. Relative names from different projects never share buffers.
func (s *ProjectService) OpenText(project projects.Project, name string) (string, bool) {
	u, ok := s.Paths(project).URIOf(name)
	if !ok {
		return "", false
	}
	snap, ok := s.Snapshot(u)
	return snap.Text, ok
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
	base := filepath.Base(filePath)
	switch base {
	case projects.BallerinaTomlFile:
		return s.applyBallerinaTomlChange(change.Kind, filePath)
	default:
		return s.applySourceFileChange(change.Kind, filePath)
	}
}

// applyBallerinaTomlChange handles create/delete/change of a Ballerina.toml:
// invalidate affected current projects before discovery, then re-resolve all
// open documents. A discovery failure leaves buffers intact without exposing
// a stale project; successful discovery reloads each distinct project once.
func (s *ProjectService) applyBallerinaTomlChange(kind WatchedFileKind, tomlPath string) error {
	dir := filepath.Dir(tomlPath)
	documents, previous := s.invalidateProjectsUnder(dir)
	roots := make(map[DocumentURI]string)
	for _, u := range documents {
		root, err := s.findSourceRootMemoized(u)
		if err != nil {
			return err
		}
		roots[u] = root
	}
	generations := make(map[string]uint64)
	for _, u := range documents {
		root := roots[u]
		gen, loaded := generations[root]
		if !loaded {
			gen = s.reloadAt(root, u)
			generations[root] = gen
			if s.bus != nil && (kind == WatchedFileCreate || kind == WatchedFileDelete) && previous[root] != 0 {
				s.bus.Publish(event.NewProjectKindTransitionedEvent(root, root))
				s.publishUpdated(root, gen)
			} else {
				s.publishLifecycle(root, gen)
			}
		}
		s.mu.Lock()
		snap := s.documents[u]
		snap.SourceRoot = root
		snap.Generation = gen
		s.documents[u] = snap
		if gen != 0 {
			delete(s.pendingRoots, u)
		}
		s.mu.Unlock()
	}
	return nil
}

// Invalidate owning projects and containing merge destinations as units, so
// their open documents outside the manifest directory are also rediscovered.
func (s *ProjectService) invalidateProjectsUnder(dir string) ([]DocumentURI, map[string]uint64) {
	var documents []DocumentURI
	previous := make(map[string]uint64)
	s.mu.Lock()
	for u := range s.documents {
		if !u.IsFile() {
			continue
		}
		root, known := s.recoveryRoot(u)
		if !isAncestor(dir, u.Path()) && !(known && isAncestor(root, dir)) {
			continue
		}
		if known {
			previous[root] = 0
			if entry, ok := s.index.peek(root); ok {
				previous[root] = entry.generation
			}
		}
	}
	for u, snap := range s.documents {
		if !u.IsFile() {
			continue
		}
		root, known := s.recoveryRoot(u)
		_, affectedProject := previous[root]
		if !isAncestor(dir, u.Path()) && !affectedProject {
			continue
		}
		if known {
			s.pendingRoots[u] = root
		}
		documents = append(documents, u)
		snap.SourceRoot = ""
		snap.Generation = 0
		s.documents[u] = snap
	}
	s.mu.Unlock()
	s.index.invalidateUnder(dir)
	for root := range previous {
		s.index.remove(root)
		if s.bus != nil {
			s.bus.Publish(event.NewProjectEvictedEvent(root, event.EvictionKindTransition))
		}
	}
	sortDocumentURIs(documents)
	return documents, previous
}

// recoveryRoot is called with mu held. Keep the last owning project while a
// manifest transition is unresolved, even if partial discovery changed the memo.
func (s *ProjectService) recoveryRoot(u DocumentURI) (string, bool) {
	if root, pending := s.pendingRoots[u]; pending {
		return root, true
	}
	return s.index.lookupSourceRoot(u.Path())
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
