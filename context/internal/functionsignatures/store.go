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

// Store keeps the untyped signatures associated with function symbols.
type Store struct {
	mu         sync.RWMutex
	signatures []model.UntypedFunctionSignature
	associated map[model.SymbolRef]model.FunctionSignatureRef
}

func NewStore() Store {
	return Store{
		associated: make(map[model.SymbolRef]model.FunctionSignatureRef),
	}
}

func (s *Store) Allocate(params []model.Param, hasRest bool) model.FunctionSignatureRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	sig := model.NewUntypedFunctionSignature(params, hasRest)
	ref := model.FunctionSignatureRef(len(s.signatures) + 1)
	s.signatures = append(s.signatures, sig)
	return ref
}

func (s *Store) Associate(sym model.SymbolRef, ref model.FunctionSignatureRef) bool {
	signatureIndex(ref)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.associated[sym]; ok {
		return ref == existing
	}
	s.associated[sym] = ref
	return true
}

func (s *Store) Ref(sym model.SymbolRef) (model.FunctionSignatureRef, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref, ok := s.associated[sym]
	return ref, ok
}

func (s *Store) UpdateIncludedRecords(ref model.FunctionSignatureRef, includedRecords []*model.IncludedRecordMetadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := signatureIndex(ref)
	sig := s.signatures[index]
	for i, metadata := range includedRecords {
		if metadata == nil {
			continue
		}
		sig.SetIncludedRecordMetadata(i, *metadata)
	}
	s.signatures[index] = sig
}

func (s *Store) Get(owner model.SymbolRef) (model.UntypedFunctionSignature, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref, ok := s.associated[owner]
	if !ok {
		return model.UntypedFunctionSignature{}, false
	}
	return s.signatures[signatureIndex(ref)], true
}

func (s *Store) GetByRef(ref model.FunctionSignatureRef) model.UntypedFunctionSignature {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.signatures[signatureIndex(ref)]
}

func signatureIndex(ref model.FunctionSignatureRef) int {
	if ref == 0 {
		panic("function signature reference is unset")
	}
	return int(ref) - 1
}
