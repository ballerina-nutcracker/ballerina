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

package workspace

import (
	"sort"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// indexEntry is one slot in the source-root-keyed project index.
type indexEntry struct {
	project     projects.Project
	sourceRoot  string
	lastTouched time.Time // LRU stamp, updated on every resolve/publish hit
	openDocs    int       // count of open DocumentURIs resolved to this source root
	generation  uint64    // per-source-root monotonic counter (ticket 09), bumped on every accepted publish
}

// projectIndex is a count-bounded, source-root-keyed LRU cache of loaded
// projects, with a filePath→sourceRoot memo for the ADR-048 root-walk. It is
// unexported and behind ProjectService. Under ticket 09 the async compile
// engine reads it concurrently with Apply, so every method is mutex-guarded.
type projectIndex struct {
	mu               sync.Mutex
	entries          map[string]*indexEntry // sourceRoot → entry
	pathToSourceRoot map[string]string      // filePath → sourceRoot (memo)
	maxProjects      int
	now              func() time.Time
}

func newProjectIndex(maxProjects int, now func() time.Time) *projectIndex {
	return &projectIndex{
		entries:          make(map[string]*indexEntry),
		pathToSourceRoot: make(map[string]string),
		maxProjects:      maxProjects,
		now:              now,
	}
}

// get returns the entry for sourceRoot, updating its LRU stamp.
func (i *projectIndex) get(sourceRoot string) (*indexEntry, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[sourceRoot]
	if !ok {
		return nil, false
	}
	entry.lastTouched = i.now()
	return entry, true
}

// peek returns the entry for sourceRoot without updating the LRU stamp. Used
// by the async compile engine's read-only accessors (Generation/
// CurrentProject) so frequent reads do not thrash the LRU order.
func (i *projectIndex) peek(sourceRoot string) (*indexEntry, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[sourceRoot]
	return entry, ok
}

// putExisting stores an entry, replacing any existing entry for the same
// source root. If the index is full and this is a new source root, it evicts
// the least-recently-touched background entry first. Eviction publishes
// ProjectEvicted on the bus.
func (i *projectIndex) putExisting(entry *indexEntry, bus *event.Bus) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, ok := i.entries[entry.sourceRoot]; ok {
		entry.lastTouched = i.now()
		i.entries[entry.sourceRoot] = entry
		return
	}
	if len(i.entries) >= i.maxProjects {
		i.evictLRULocked(bus)
	}
	i.entries[entry.sourceRoot] = entry
}

// put stores a new entry. If the index is full, it evicts the
// least-recently-touched background entry (openDocs == 0) first; only if every
// entry is active does it evict the LRU active entry. Eviction publishes
// ProjectEvicted on the bus. Returns the stored entry and the evicted root
// (empty if none).
func (i *projectIndex) put(entry *indexEntry, bus *event.Bus) (*indexEntry, string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if existing, ok := i.entries[entry.sourceRoot]; ok {
		existing.project = entry.project
		existing.lastTouched = i.now()
		return existing, ""
	}
	evictedRoot := ""
	if len(i.entries) >= i.maxProjects {
		evictedRoot = i.evictLRULocked(bus)
	}
	i.entries[entry.sourceRoot] = entry
	return entry, evictedRoot
}

// evictLRU removes the least-recently-touched background entry; if all entries
// are active, removes the LRU active entry. Returns the evicted source root
// (empty if the index was empty).
func (i *projectIndex) evictLRU(bus *event.Bus) string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.evictLRULocked(bus)
}

func (i *projectIndex) evictLRULocked(bus *event.Bus) string {
	var victim *indexEntry
	for _, entry := range i.entries {
		if entry.openDocs == 0 {
			if victim == nil || entry.lastTouched.Before(victim.lastTouched) {
				victim = entry
			}
		}
	}
	if victim == nil {
		for _, entry := range i.entries {
			if victim == nil || entry.lastTouched.Before(victim.lastTouched) {
				victim = entry
			}
		}
	}
	if victim == nil {
		return ""
	}
	i.removeLocked(victim.sourceRoot)
	if bus != nil {
		bus.Publish(event.NewProjectEvictedEvent(victim.sourceRoot, event.EvictionLRU))
	}
	return victim.sourceRoot
}

// remove deletes an entry and invalidates the filePath→sourceRoot memo entries
// that pointed to it.
func (i *projectIndex) remove(sourceRoot string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.removeLocked(sourceRoot)
}

func (i *projectIndex) removeLocked(sourceRoot string) {
	delete(i.entries, sourceRoot)
	for filePath, root := range i.pathToSourceRoot {
		if root == sourceRoot {
			delete(i.pathToSourceRoot, filePath)
		}
	}
}

// lookupSourceRoot returns the memoized source root for filePath, if present.
func (i *projectIndex) lookupSourceRoot(filePath string) (string, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	root, ok := i.pathToSourceRoot[filePath]
	return root, ok
}

// memoSourceRoot records filePath → sourceRoot.
func (i *projectIndex) memoSourceRoot(filePath, sourceRoot string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.pathToSourceRoot[filePath] = sourceRoot
}

// invalidateUnder removes memo entries whose file path is under dir (or equals
// dir) — used when Ballerina.toml is created/deleted/changed under dir.
func (i *projectIndex) invalidateUnder(dir string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for filePath := range i.pathToSourceRoot {
		if filePath == dir || isAncestor(dir, filePath) {
			delete(i.pathToSourceRoot, filePath)
		}
	}
}

// supersede advances the source root's generation counter without publishing,
// so any in-flight compile cycle for root observes a generation that no longer
// matches its own and is gated out at the stale-publication gate (CE-E3). It
// is a no-op for an unknown root. It does not touch debounce timers
// (engine-owned) or close the bus.
func (i *projectIndex) supersede(root string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if entry, ok := i.entries[root]; ok {
		entry.generation++
	}
}

// supersedeAll advances every known source root's generation counter without
// publishing, so any in-flight compile cycle for those roots observes a
// generation that no longer matches its own and is gated out at the
// stale-publication gate (CE-E3). It is the workspace's contribution to
// shutdown; the compile engine's own closed flag provides the same gating
// defensively. It does not touch debounce timers (engine-owned) or close the
// bus (the wiring closes the bus last so the compile engine can Flush first).
func (i *projectIndex) supersedeAll() {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, entry := range i.entries {
		entry.generation++
	}
}

// sortDocumentURIs sorts document URIs by their string form for stable
// iteration order.
func sortDocumentURIs(us []uri.DocumentURI) {
	sort.Slice(us, func(i, j int) bool { return us[i].String() < us[j].String() })
}

// isAncestor reports whether dir is an ancestor of path (dir == path or path is
// under dir), using forward-slash semantics.
func isAncestor(dir, path string) bool {
	if dir == "" {
		return false
	}
	if path == dir {
		return true
	}
	prefix := dir
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	return len(path) > len(prefix) && path[:len(prefix)] == prefix
}
