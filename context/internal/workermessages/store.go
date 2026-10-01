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

// Store maps each worker message to the type of the value sent. A send
// publishes its type once; a receive blocks until it is published or poisoned.
type Store struct {
	mu      sync.Mutex
	entries map[model.WorkerMessageRef]*entry
}

type entryState uint8

const (
	entryUnset entryState = iota
	entryPublished
	entryPoisoned
)

type entry struct {
	ready chan struct{}
	ty    semtypes.SemType
	state entryState
}

func NewStore() *Store {
	return &Store{entries: make(map[model.WorkerMessageRef]*entry)}
}

// entryFor must be called with s.mu held.
func (s *Store) entryFor(ref model.WorkerMessageRef) *entry {
	e, ok := s.entries[ref]
	if !ok {
		e = &entry{ready: make(chan struct{})}
		s.entries[ref] = e
	}
	return e
}

// Publish records the type of the value sent as ref. Publishing a message
// twice is an error.
func (s *Store) Publish(ref model.WorkerMessageRef, ty semtypes.SemType) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entryFor(ref)
	if e.state != entryUnset {
		return fmt.Errorf("worker message %d published twice", ref)
	}
	e.ty = ty
	e.state = entryPublished
	close(e.ready)
	return nil
}

// PublishPoisonIfUnset releases the receive of ref without a type, unless the
// send already published one.
func (s *Store) PublishPoisonIfUnset(ref model.WorkerMessageRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entryFor(ref)
	if e.state != entryUnset {
		return
	}
	e.state = entryPoisoned
	close(e.ready)
}

// Type blocks until ref is published or poisoned. It reports false when ref
// was poisoned.
func (s *Store) Type(ref model.WorkerMessageRef) (semtypes.SemType, bool) {
	s.mu.Lock()
	e := s.entryFor(ref)
	s.mu.Unlock()
	<-e.ready
	return e.ty, e.state == entryPublished
}
