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

// Package objectsignatures provides store used to store object signatures in env
package objectsignatures

import (
	"sync"

	"github.com/ballerina-nutcracker/ballerina/model"
)

// Store keeps the untyped signatures associated with class and type symbols.
type Store struct {
	mu         sync.RWMutex
	signatures []model.UntypedObjectSignature
	associated map[model.SymbolRef]model.ObjectSignatureRef
}

func NewStore() Store {
	return Store{associated: make(map[model.SymbolRef]model.ObjectSignatureRef)}
}

func (s *Store) Allocate(finalFields []string) model.ObjectSignatureRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.signatures = append(s.signatures, model.NewUntypedObjectSignature(finalFields))
	return model.ObjectSignatureRef(len(s.signatures))
}

func (s *Store) Associate(sym model.SymbolRef, ref model.ObjectSignatureRef) bool {
	signatureIndex(ref)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.associated[sym]; ok {
		return ref == existing
	}
	s.associated[sym] = ref
	return true
}

func (s *Store) Ref(sym model.SymbolRef) (model.ObjectSignatureRef, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref, ok := s.associated[sym]
	return ref, ok
}

func (s *Store) UpdateFinalFields(ref model.ObjectSignatureRef, finalFields []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.signatures[signatureIndex(ref)] = model.NewUntypedObjectSignature(finalFields)
}

func (s *Store) Get(owner model.SymbolRef) (model.UntypedObjectSignature, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref, ok := s.associated[owner]
	if !ok {
		return model.UntypedObjectSignature{}, false
	}
	return s.signatures[signatureIndex(ref)], true
}

func (s *Store) GetByRef(ref model.ObjectSignatureRef) model.UntypedObjectSignature {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.signatures[signatureIndex(ref)]
}

func signatureIndex(ref model.ObjectSignatureRef) int {
	if ref == 0 {
		panic("object signature reference is unset")
	}
	return int(ref) - 1
}
