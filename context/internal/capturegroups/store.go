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

// Package capturegroups provides the store used to hold capture groups in env
package capturegroups

import (
	"fmt"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/model"
)

// Store keeps the capture groups allocated during symbol resolution. Symbol
// resolution adds to a group; type resolution only reads it.
type Store struct {
	mu     sync.RWMutex
	groups []map[model.SymbolRef]struct{}
}

func NewStore() Store {
	return Store{groups: []map[model.SymbolRef]struct{}{}}
}

// Allocate returns the handle of a new empty group.
func (s *Store) Allocate() model.CaptureGroupRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups = append(s.groups, make(map[model.SymbolRef]struct{}))
	return model.CaptureGroupRef(len(s.groups))
}

func (s *Store) Add(group model.CaptureGroupRef, ref model.SymbolRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groupAt(group)[ref] = struct{}{}
}

func (s *Store) Contains(group model.CaptureGroupRef, ref model.SymbolRef) bool {
	if group == 0 {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.groupAt(group)[ref]
	return ok
}

// groupAt must be called with s.mu held.
func (s *Store) groupAt(group model.CaptureGroupRef) map[model.SymbolRef]struct{} {
	// Sanity check: every handle must come from Allocate on this store.
	if group <= 0 || int(group) > len(s.groups) {
		panic(fmt.Sprintf("capture group reference %d is out of range [1, %d]", group, len(s.groups)))
	}
	return s.groups[group-1]
}
