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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package locks provides re-entrant mutexes keyed by content-addressed
// string identifiers. Each restricted variable in the source program maps to
// a single program-wide mutex shared across all instances and modules.
//
// Release model: a held mutex is released only by a paired `LockEnd` BIR
// terminator emitted by BIR-gen, or by the interpreter's top-level recover
// (via extern.Context.ReleaseAllHeldLocks) when a panic is uncaught. `trap`
// does NOT participate in lock release: if a Go-level panic raised inside a
// `lock` body (div-by-zero, array OOB, nil deref, integer overflow, failed
// `check`, etc.) is recovered by an outer `trap`, the strand retains the
// lock. This is intentional at this stage — mutexes are strand-scoped and
// re-entrant, so the retained ownership is harmless within the same strand,
// and the interpreter is currently single-strand for the surfaces we ship.
// Cross-strand release-on-trap is deferred to when multi-strand `lock`-using
// code is exposed.
package locks

import (
	"math"
	"sync"
)

// ReentrantMutex is a strand-aware re-entrant mutex. The same strand may
// acquire the mutex any number of times; each acquisition must be paired
// with a release before another strand can take ownership.
// Here mu (and cond) is used to guarding the internal bookkeeping of owner
// and count not mutal excluion for the lock statement. That is provided via
// owner (and count)
type ReentrantMutex struct {
	mu    sync.Mutex
	cond  *sync.Cond
	owner uint64 // 0 == unowned
	count int
}

func newReentrantMutex() *ReentrantMutex {
	m := &ReentrantMutex{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// Lock acquires the mutex for the given strand, blocking until it is free
// or already owned by this strand.
func (r *ReentrantMutex) Lock(strandID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for r.owner != 0 && r.owner != strandID {
		r.cond.Wait()
	}
	if r.count == math.MaxInt {
		// I don't think this can happen in any realworld use case but if happens we can have
		// allsorts of difficult to detect deadlocks
		panic("ReentrantMutex re-entry counter overflow")
	}
	r.owner = strandID
	r.count++
}

// Unlock releases one acquisition by the given strand. Panics if the strand
// is not the current owner — that indicates a runtime invariant violation,
// not a Ballerina-level error.
func (r *ReentrantMutex) Unlock(strandID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.owner != strandID {
		// I can't think of any way this can happen in a valid BIR
		panic("ReentrantMutex.Unlock called by non-owning strand")
	}
	r.count--
	if r.count == 0 {
		r.owner = 0
		r.cond.Signal()
	}
}

// LockManager is a global content-addressed registry of re-entrant mutexes. Looking up
// the same key always yields the same *ReentrantMutex, lazily creating one
// on first access.
type LockManager struct {
	mu sync.Mutex
	m  map[string]*ReentrantMutex
}

func NewMutexes() *LockManager {
	return &LockManager{m: map[string]*ReentrantMutex{}}
}

// Get returns the mutex associated with key, allocating a fresh one on first
// access.
func (t *LockManager) Get(key string) *ReentrantMutex {
	t.mu.Lock()
	defer t.mu.Unlock()
	if m, ok := t.m[key]; ok {
		return m
	}
	m := newReentrantMutex()
	t.m[key] = m
	return m
}
