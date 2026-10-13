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

package cfg

import (
	"slices"

	"github.com/ballerina-nutcracker/ballerina/common/bitset"
)

// initState tracks which names are in scope (tracked) and which of those are definitely initialized (init).
// Invariant: init ⊆ tracked.
type initState struct {
	tracked bitset.BitSet
	init    bitset.BitSet
}

func (s *initState) isEmpty() bool {
	return s.tracked.IsEmpty()
}

func (s *initState) isTracked(p bitset.Pos) bool {
	return s.tracked.Has(p)
}

func (s *initState) isInitialized(p bitset.Pos) bool {
	return s.init.Has(p)
}

func (s *initState) markTracked(p bitset.Pos) {
	s.tracked.Set(p)
}

func (s *initState) markInitialized(p bitset.Pos) {
	s.tracked.Set(p)
	s.init.Set(p)
}

func (s *initState) markAllInitialized(o *initState) {
	s.tracked.Or(&o.tracked)
	s.init.Or(&o.init)
}

// mergeWith joins two control flow paths: a name is tracked if either path tracks it and initialized only if both
// paths initialize it.
func (s *initState) mergeWith(o *initState) {
	s.tracked.Or(&o.tracked)
	s.init.And(&o.init)
}

func (s *initState) copyFrom(o *initState) {
	s.tracked.CopyFrom(&o.tracked)
	s.init.CopyFrom(&o.init)
}

// initStateStore holds the exit state of each block until all of its forward readers have consumed it. The last
// reader takes ownership of the state and released states are reused.
type initStateStore struct {
	exits    []*initState // by block id; nil = unprocessed or released
	readers  []int        // remaining forward (non-backedge) reads of exits[i]
	free     []*initState
	newState func() *initState
}

func newInitStateStore(fcfg *functionCFG, newState func() *initState) *initStateStore {
	st := &initStateStore{
		exits:    make([]*initState, len(fcfg.bbs)),
		readers:  make([]int, len(fcfg.bbs)),
		free:     nil,
		newState: newState,
	}
	for i := range fcfg.bbs {
		bb := &fcfg.bbs[i]
		for _, p := range bb.parents {
			if !slices.Contains(bb.backedgeParents, p) {
				st.readers[p]++
			}
		}
	}
	return st
}

// entryState merges the exit states of the non-backedge predecessors of bb. A predecessor whose exit is
// unprocessed or empty is ignored.
func (st *initStateStore) entryState(bb *basicBlock) *initState {
	var result *initState
	for _, p := range bb.parents {
		if slices.Contains(bb.backedgeParents, p) {
			continue
		}
		st.readers[p]--
		src := st.exits[p]
		if src == nil || src.isEmpty() {
			st.releaseIfDone(p)
			continue
		}
		if result == nil {
			result = st.take(p)
		} else {
			result.mergeWith(src)
			st.releaseIfDone(p)
		}
	}
	if result == nil {
		result = st.alloc()
	}
	return result
}

// take returns the exit state of p, stealing it if this is its last read and copying it otherwise.
func (st *initStateStore) take(p int) *initState {
	if st.readers[p] == 0 {
		s := st.exits[p]
		st.exits[p] = nil
		return s
	}
	s := st.alloc()
	s.copyFrom(st.exits[p])
	return s
}

func (st *initStateStore) setExit(bb *basicBlock, s *initState) {
	st.exits[bb.id] = s
	st.releaseIfDone(bb.id)
}

func (st *initStateStore) alloc() *initState {
	n := len(st.free)
	if n == 0 {
		return st.newState()
	}
	s := st.free[n-1]
	st.free = st.free[:n-1]
	var empty initState
	s.copyFrom(&empty)
	return s
}

func (st *initStateStore) releaseIfDone(p int) {
	if st.readers[p] == 0 && st.exits[p] != nil {
		st.free = append(st.free, st.exits[p])
		st.exits[p] = nil
	}
}
