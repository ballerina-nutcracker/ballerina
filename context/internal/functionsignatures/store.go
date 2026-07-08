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

// Package functionsignatures provides store used to store function signatures in env
package functionsignatures

import (
	"sync"

	"ballerina-lang-go/model"
)

// Handle is a an opaque pointer to signatures already stored in the store
type Handle struct {
	index int
}

// Store keeps the untyped signatures associated with function symbols.
type Store struct {
	mu         sync.RWMutex
	signatures []model.UntypedFunctionSignature
	associated map[model.SymbolRef]int
}

func NewStore() Store {
	return Store{
		associated: make(map[model.SymbolRef]int),
	}
}

func (s *Store) Allocate(params []model.Param, hasRest bool) Handle {
	s.mu.Lock()
	defer s.mu.Unlock()
	sig := model.NewUntypedFunctionSignature(params, hasRest)
	index := len(s.signatures)
	s.signatures = append(s.signatures, sig)
	return Handle{index}
}

func (s *Store) Associate(sym model.SymbolRef, handle Handle) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.associated[sym]; ok {
		return false
	}
	s.associated[sym] = handle.index
	return true
}

func (s *Store) Handle(sym model.SymbolRef) (Handle, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	idx, ok := s.associated[sym]
	return Handle{idx}, ok
}

func (s *Store) UpdateIncludedRecords(handle Handle, includedRecords []*model.IncludedRecordMetadata) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sig := s.signatures[handle.index]
	for i, metadata := range includedRecords {
		if metadata == nil {
			continue
		}
		sig.SetIncludedRecordMetadata(i, *metadata)
	}
	s.signatures[handle.index] = sig
	return true
}

func (s *Store) Get(ref model.SymbolRef) (model.UntypedFunctionSignature, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sig, _, ok := s.getUnsafe(ref)
	return sig, ok
}

func (s *Store) GetByHandle(handle Handle) model.UntypedFunctionSignature {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.signatures[handle.index]
}

func (s *Store) getUnsafe(ref model.SymbolRef) (model.UntypedFunctionSignature, int, bool) {
	handle, ok := s.associated[ref]
	if !ok {
		return model.UntypedFunctionSignature{}, 0, false
	}
	sig := s.signatures[handle]
	return sig, handle, true
}
