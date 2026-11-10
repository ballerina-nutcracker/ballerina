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

// Package workermessages provides the store type resolution uses to hand the
// type of each worker message from its send to its receive.
package workermessages

import (
	"fmt"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// Result is a resolved message type or a failure to resolve its send.
type Result struct {
	Type   semtypes.SemType
	Failed bool
}

// Store hands message type results from sends to receives.
type Store struct {
	mu      sync.Mutex
	entries map[model.WorkerMessageRef]*entry
}

type entry struct {
	result  *Result
	waiters []chan Result
}

func NewStore() *Store {
	return &Store{entries: make(map[model.WorkerMessageRef]*entry)}
}

// entryFor must be called with s.mu held.
func (s *Store) entryFor(ref model.WorkerMessageRef) *entry {
	e, ok := s.entries[ref]
	if !ok {
		e = &entry{}
		s.entries[ref] = e
	}
	return e
}

func (e *entry) publish(result Result) {
	e.result = &result
	for _, waiter := range e.waiters {
		waiter <- result
		close(waiter)
	}
	e.waiters = nil
}

// Publish records the type of the value sent as ref. Publishing a message
// twice is an error.
func (s *Store) Publish(ref model.WorkerMessageRef, ty semtypes.SemType) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entryFor(ref)
	if e.result != nil {
		return fmt.Errorf("worker message %d published twice", ref)
	}
	e.publish(Result{Type: ty})
	return nil
}

// FailIfUnset releases the receivers of ref if its send never published a type.
func (s *Store) FailIfUnset(ref model.WorkerMessageRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entryFor(ref)
	if e.result == nil {
		e.publish(Result{Failed: true})
	}
}

// Type returns a channel that yields one result once ref is resolved or fails.
// Each call returns an independent channel.
func (s *Store) Type(ref model.WorkerMessageRef) <-chan Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entryFor(ref)
	result := make(chan Result, 1)
	if e.result != nil {
		result <- *e.result
		close(result)
	} else {
		e.waiters = append(e.waiters, result)
	}
	return result
}
