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

package compile

import (
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
)

// moduleGenSnapshot retains a committed driver, Phase-1 status, exports,
// fingerprints, source hash, and external feature projections.
type moduleGenSnapshot struct {
	driver              *moduleDriver
	errored             bool
	exported            model.ExportedSymbolSpace
	fingerprint         fingerprint
	textHash            string
	externalProjections map[semantics.PackageIdentifier]ExternalModuleProjection
}

// packageGenState holds carried-forward module state for one root. Background
// cycles replace it wholesale; sealed reads capture it under genStatesMu.
// Inline diagnostic compiles do not read or commit this state.
type packageGenState struct {
	modules map[projects.ModuleID]moduleGenSnapshot
}

// genStateFor returns root's module state, creating an empty state on first use.
func (s *CompilationService) genStateFor(root string) *packageGenState {
	s.genStatesMu.Lock()
	defer s.genStatesMu.Unlock()
	st, ok := s.genStates[root]
	if !ok {
		st = &packageGenState{modules: make(map[projects.ModuleID]moduleGenSnapshot)}
		s.genStates[root] = st
	}
	return st
}

// setGenState replaces root's carried-forward module state.
func (s *CompilationService) setGenState(root string, modules map[projects.ModuleID]moduleGenSnapshot) {
	s.genStatesMu.Lock()
	defer s.genStatesMu.Unlock()
	s.genStates[root] = &packageGenState{modules: modules}
}

// evictGenState drops root's carried-forward module state.
func (s *CompilationService) evictGenState(root string) {
	s.genStatesMu.Lock()
	defer s.genStatesMu.Unlock()
	delete(s.genStates, root)
}
