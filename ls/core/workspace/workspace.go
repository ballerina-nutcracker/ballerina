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

// Package workspace owns open-document state and the project index. Changes
// carry full text, and project discovery uses a host-supplied directory opener.
// Open buffers overlay source files at load time and supply live text to compiles.
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

// WithRepositories sets the repositories used to load projects.
// If unset, projects.Load uses its default repositories.
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

// DocumentChange describes a document lifecycle mutation with resolved full text.
type DocumentChange struct {
	Kind       ChangeKind
	URI        DocumentURI
	Text       string // valid for ChangeOpen and ChangeUpdate
	Version    int32  // valid for ChangeOpen and ChangeUpdate
	LanguageID string // valid for ChangeOpen
}

// Snapshot is a value view of an open document's text and publication metadata.
// Generation is zero when no successful publication has been recorded.
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

// ProjectService owns open-document state and the source-root-keyed project index.
type ProjectService struct {
	openDir      func(string) fs.FS
	bus          *event.Bus
	mu           sync.RWMutex // guards documents and pendingRoots
	documents    map[DocumentURI]Snapshot
	pendingRoots map[DocumentURI]string // recovery membership only; guarded by mu
	index        *projectIndex
	now          func() time.Time
	repositories []projects.Repository
}

// New creates a ProjectService with a host directory opener and an event bus.
// Options configure the project count bound and package repositories.
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

// Apply updates document state and publishes a generation for open/update.
// Existing documents with unchanged import sets reuse the loaded project;
// missing documents or changed import sets trigger a reload. Cancellation is
// checked after storing text and discovering the root, before publication.
// Close removes the buffer without publishing.
func (s *ProjectService) Apply(ctx context.Context, change DocumentChange) (Snapshot, error) {
	if change.Kind == ChangeClose {

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
	gen := s.publish(change, sourceRoot)
	snap.Generation = gen

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

// publish advances an existing project's generation for content-only edits.
// Initial loads, missing documents, and changed import sets reload the project
// and publish lifecycle events.
func (s *ProjectService) publish(change DocumentChange, sourceRoot string) uint64 {
	s.mu.Lock()
	entry, ok := s.index.get(sourceRoot)
	if !ok || entry.project == nil {
		s.mu.Unlock()
		gen := s.reloadAt(sourceRoot, change.URI)
		s.publishLifecycle(sourceRoot, gen)
		return gen
	}
	if s.documentInCurrentPackage(entry.project, change) && !s.importSetChanged(entry.project, change) {
		gen := s.index.advance(sourceRoot)
		s.mu.Unlock()
		s.publishUpdated(sourceRoot, gen)
		return gen
	}
	s.mu.Unlock()

	gen := s.reloadAt(sourceRoot, change.URI)
	s.publishLifecycle(sourceRoot, gen)
	return gen
}

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

// importSetChanged compares complete import declarations against loaded text,
// ignoring aliases, ordering, and declarations with diagnostics.
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

func (s *ProjectService) publishLifecycle(sourceRoot string, gen uint64) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(event.NewProjectRegisteredEvent(sourceRoot))
	s.bus.Publish(event.NewProjectUpdatedEvent(sourceRoot, gen))
}

func (s *ProjectService) publishUpdated(sourceRoot string, gen uint64) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(event.NewProjectUpdatedEvent(sourceRoot, gen))
}

// reloadAt loads through a fresh overlay and replaces the index entry with a
// new generation, preserving any existing open-doc count. A load failure
// returns zero without replacing the entry.
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

// Generation returns the root's current generation, including supersessions,
// or 0/false if the root is unknown.
func (s *ProjectService) Generation(root string) (uint64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.index.peek(root)
	if !ok {
		return 0, false
	}
	return entry.generation, true
}

// Supersede advances root's generation without publishing; unknown roots are ignored.
func (s *ProjectService) Supersede(root string) {
	s.index.supersede(root)
}

// CurrentProject returns the indexed project and generation for root, or
// nil/0/false if no project is loaded.
func (s *ProjectService) CurrentProject(root string) (projects.Project, uint64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.index.peek(root)
	if !ok || entry.project == nil {
		return nil, 0, false
	}
	return entry.project, entry.generation, true
}

// OpenDocumentsUnder returns open file URIs resolved to root, sorted by URI.
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

// ProjectKey discovers a file URI's workspace key: a package directory or
// a single-file path, not the project's relative load name.
func (s *ProjectService) ProjectKey(u DocumentURI) (string, error) {
	return s.findSourceRootMemoized(u)
}

// Paths returns name conversion metadata for an indexed project, or a zero value
// if the project is not indexed.
func (s *ProjectService) Paths(project projects.Project) SourcePaths {
	return s.index.paths(project)
}

// Project discovers a file URI's root and returns its indexed project. It
// does not load projects; an unregistered or evicted root returns nil.
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

// Snapshot returns the current open-document state for the given URI.
func (s *ProjectService) Snapshot(u DocumentURI) (Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.documents[u]
	return snap, ok
}

// DocumentText returns open-buffer text, falling back to the loaded document's
// text. It returns false if neither source is available.
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

// ApplyWatchedFile rediscovers projects after Ballerina.toml changes and
// removes indexed projects for deleted, closed source files. It ignores
// non-file URIs and does not observe ctx cancellation.
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

// applyBallerinaTomlChange invalidates affected projects and rediscovers their
// open documents. Discovery errors retain buffers without restoring stale
// projects; successful discovery attempts one reload per distinct root.
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

// invalidateProjectsUnder removes affected owning projects and containing
// merge destinations, retaining their open buffers for rediscovery.
func (s *ProjectService) invalidateProjectsUnder(dir string) ([]DocumentURI, map[string]uint64) {
	var documents []DocumentURI
	previous := make(map[string]uint64)
	s.mu.Lock()
	for u := range s.documents {
		if !u.IsFile() {
			continue
		}
		root, known := s.recoveryRoot(u)
		if !isAncestor(dir, u.Path()) && (!known || !isAncestor(root, dir)) {
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

// recoveryRoot returns pending transition membership before the current memo.
// The caller must hold mu.
func (s *ProjectService) recoveryRoot(u DocumentURI) (string, bool) {
	if root, pending := s.pendingRoots[u]; pending {
		return root, true
	}
	return s.index.lookupSourceRoot(u.Path())
}

// applySourceFileChange removes the memoized owning project for a deleted,
// closed file. Other changes leave the index untouched.
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
		return nil
	}
	sourceRoot, ok := s.index.lookupSourceRoot(filePath)
	if !ok {
		return nil
	}
	s.index.remove(sourceRoot)
	return nil
}

// Shutdown supersedes all indexed generations without closing the event bus.
// It does not observe ctx cancellation.
func (s *ProjectService) Shutdown(ctx context.Context) error {
	_ = ctx
	s.index.supersedeAll()
	return nil
}
