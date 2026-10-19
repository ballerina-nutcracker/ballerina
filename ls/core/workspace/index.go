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
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

// indexEntry is one slot in the source-root-keyed project index.
type indexEntry struct {
	project     projects.Project
	paths       SourcePaths
	sourceRoot  string
	lastTouched time.Time // LRU stamp
	openDocs    int       // zero marks a background entry for eviction
	generation  uint64    // monotonic lifetime stamp, never reused when a root returns
}

// projectIndex is a count-bounded LRU project cache with a file-path-to-root
// memo. Its mutex guards index operations; get returns a live entry.
type projectIndex struct {
	mu               sync.Mutex
	entries          map[string]*indexEntry // sourceRoot → entry
	pathToSourceRoot map[string]string      // filePath → sourceRoot (memo)
	maxProjects      int
	lastGeneration   uint64 // lifetime high-water mark; survives entry eviction
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

// peek returns an entry copy without updating its LRU stamp.
func (i *projectIndex) peek(sourceRoot string) (*indexEntry, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[sourceRoot]
	if !ok {
		return nil, false
	}
	snapshot := *entry
	return &snapshot, true
}

// putExisting assigns a fresh generation and stores the entry, replacing any
// existing entry for the same root. If the index is full for a new root, it evicts
// the least-recently-touched background entry first. Eviction publishes
// ProjectEvicted on the bus.
func (i *projectIndex) putExisting(entry *indexEntry, bus *event.Bus) uint64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	gen := i.advanceLocked(entry)
	if _, ok := i.entries[entry.sourceRoot]; ok {
		entry.lastTouched = i.now()
		i.entries[entry.sourceRoot] = entry
		return gen
	}
	if len(i.entries) >= i.maxProjects {
		i.evictLRULocked(bus)
	}
	i.entries[entry.sourceRoot] = entry
	return gen
}

func (i *projectIndex) advance(root string) (uint64, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[root]
	if !ok {
		return 0, false
	}
	return i.advanceLocked(entry), true
}

func (i *projectIndex) advanceLocked(entry *indexEntry) uint64 {
	i.lastGeneration++
	entry.generation = i.lastGeneration
	return entry.generation
}

// put updates an existing entry's project and LRU stamp or inserts a new entry,
// evicting via evictLRU when full. It returns the stored entry and evicted root.
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

// invalidateUnder removes memo entries whose file path is under or equal to dir.
func (i *projectIndex) invalidateUnder(dir string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for filePath := range i.pathToSourceRoot {
		if filePath == dir || isAncestor(dir, filePath) {
			delete(i.pathToSourceRoot, filePath)
		}
	}
}

// supersede advances root's generation without publishing; unknown roots are ignored.
func (i *projectIndex) supersede(root string) {
	i.advance(root)
}

// supersedeAll advances every indexed root's generation without publishing.
func (i *projectIndex) supersedeAll() {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, entry := range i.entries {
		i.advanceLocked(entry)
	}
}

// sortDocumentURIs sorts document URIs by their string form for stable
// iteration order.
func sortDocumentURIs(us []DocumentURI) {
	sort.Slice(us, func(i, j int) bool { return us[i].String() < us[j].String() })
}

// isAncestor compares native path components, not string prefixes.
func isAncestor(dir, file string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, file)
	return err == nil && rel != ".." && !strings.HasPrefix(filepath.ToSlash(rel), "../")
}

func (i *projectIndex) paths(project projects.Project) SourcePaths {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, entry := range i.entries {
		if entry.project == project {
			return entry.paths
		}
	}
	return SourcePaths{}
}
