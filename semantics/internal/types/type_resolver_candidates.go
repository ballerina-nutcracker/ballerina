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

package types

import (
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type ephemeralState struct {
	depth int
	// unimplementedReported records that a trial reported an unimplemented error. Compilation fails on that
	// diagnostic, so a new expression whose trials all failed does not add its own.
	unimplementedReported bool
	// symbolTypes shadows the compiler context's symbol types while a trial runs. It is non-nil exactly while
	// depth > 0 and is shared by every trial in the nesting: a trial writes a symbol's type before reading it, so
	// stale entries from sibling candidates or inner trials are overwritten before they can be read.
	symbolTypes map[model.SymbolRef]semtypes.SemType
}

func (s *ephemeralState) lookupSymbolType(ref model.SymbolRef) (semtypes.SemType, bool) {
	ty, ok := s.symbolTypes[ref]
	return ty, ok
}

// setSymbolType records a trial-local symbol type. Callers guarantee depth > 0, so the map exists.
func (s *ephemeralState) setSymbolType(ref model.SymbolRef, ty semtypes.SemType) {
	s.symbolTypes[ref] = ty
}

func noteEphemeralUnimplemented(t typeResolver) {
	if state := resolverEphemeralState(t); state != nil && state.depth > 0 {
		state.unimplementedReported = true
	}
}

func resolverEphemeralState(t typeResolver) *ephemeralState {
	switch resolver := t.(type) {
	case *packageTypeResolver:
		return &resolver.ephemeralState
	case *functionTypeResolver:
		return resolver.ephemeralState
	case *loopTypeResolver:
		return resolverEphemeralState(resolver.parentResolver)
	default:
		return nil
	}
}

func enterEphemeral(t typeResolver) func() {
	state := resolverEphemeralState(t)
	if state == nil {
		return func() {}
	}
	if state.depth == 0 {
		state.symbolTypes = make(map[model.SymbolRef]semtypes.SemType)
	}
	state.depth++
	return func() {
		state.depth--
		if state.depth == 0 {
			state.symbolTypes = nil
		}
	}
}

type candidateOutcome uint8

const (
	candidateNone candidateOutcome = iota
	candidateOne
	candidateAmbiguous
	// candidateUnsupported means no candidate survived and a trial reported an unimplemented error.
	candidateUnsupported
)

// selectCandidate runs trial for each candidate while t is ephemeral and returns the only candidate whose trial
// succeeded. A trial resolves the expression's operands against its candidate without committing anything except
// the XML step lowering (see resolveXMLStepExpr), so the caller resolves them again, for real, against the selected
// candidate.
func selectCandidate[C any](t typeResolver, candidates []C, trial func(C) bool) (C, candidateOutcome) {
	var selected C
	state := resolverEphemeralState(t)
	if len(candidates) < 2 || state == nil {
		t.internalError("candidate selection requires at least two candidates and an ephemeral state", diagnostics.Location{})
		return selected, candidateNone
	}
	outerUnimplemented := state.unimplementedReported
	state.unimplementedReported = false
	survivors := runTrials(t, candidates, trial)
	unimplemented := state.unimplementedReported
	state.unimplementedReported = outerUnimplemented || unimplemented
	switch {
	case len(survivors) == 1:
		return survivors[0], candidateOne
	case len(survivors) > 1:
		return selected, candidateAmbiguous
	case unimplemented:
		return selected, candidateUnsupported
	default:
		return selected, candidateNone
	}
}

func runTrials[C any](t typeResolver, candidates []C, trial func(C) bool) []C {
	exitEphemeral := enterEphemeral(t)
	defer exitEphemeral()
	var survivors []C
	for _, candidate := range candidates {
		if trial(candidate) {
			survivors = append(survivors, candidate)
		}
	}
	return survivors
}
