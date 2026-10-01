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
	"reflect"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type nodeSnapshot struct {
	node  ast.BLangNode
	state any
}

type symbolSnapshot struct {
	ref        model.SymbolRef
	ty         semtypes.SemType
	function   *model.TypedFunctionSignature
	dependent  bool
	paramTypes []semtypes.SemType
	returnType model.TypeOp
}

type ephemeralState struct {
	depth int
	// refusedDependent records that a candidate trial stopped at a
	// dependently-typed call. Monomorphizing one mutates shared symbol state that
	// snapshotArgumentState cannot capture, because the symbol is still deferred
	// when the snapshot is taken, so the trial cannot be rolled back.
	refusedDependent bool
}

// refuseDependentCall marks the in-flight candidate trial as having stopped at a
// dependently-typed call. It reports whether a trial is in flight at all.
func refuseDependentCall(t typeResolver) bool {
	state := resolverEphemeralState(t)
	if state == nil || state.depth == 0 {
		return false
	}
	state.refusedDependent = true
	return true
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

// childEphemeralState is the ephemeral state of a child resolver of t. It is
// owned by the goroutine resolving the child and starts inside every candidate
// trial t is in; mergeChildResolver hands the outcome back.
func childEphemeralState(t typeResolver) *ephemeralState {
	state := &ephemeralState{}
	if parent := resolverEphemeralState(t); parent != nil {
		state.depth = parent.depth
	}
	return state
}

// enterEphemeral starts a candidate trial. The trial gets its own worker
// message type store, so its sends publish without clashing with another
// trial or the final resolution, and it is discarded with the trial.
func enterEphemeral(t typeResolver) func() {
	state := resolverEphemeralState(t)
	if state == nil {
		return func() {}
	}
	state.depth++
	messageTypes := resolverMessageTypesSlot(t)
	previous := *messageTypes
	*messageTypes = t.compilerContext().NewWorkerMessageTypeStore()
	return func() {
		*messageTypes = previous
		state.depth--
	}
}

// resolverMessageTypes returns the worker message type store t publishes to
// and reads from.
func resolverMessageTypes(t typeResolver) *context.WorkerMessageTypeStore {
	return *resolverMessageTypesSlot(t)
}

func resolverMessageTypesSlot(t typeResolver) **context.WorkerMessageTypeStore {
	switch resolver := t.(type) {
	case *packageTypeResolver:
		return &resolver.messageTypes
	case *functionTypeResolver:
		return &resolver.messageTypes
	case *loopTypeResolver:
		return resolverMessageTypesSlot(resolver.parentResolver)
	default:
		return nil
	}
}

type argumentStateSnapshotter struct {
	t       typeResolver
	nodes   []nodeSnapshot
	symbols []symbolSnapshot
	seen    map[ast.BLangNode]struct{}
	seenRef map[model.SymbolRef]struct{}
}

func (s *argumentStateSnapshotter) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if _, ok := s.seen[node]; !ok {
		value := reflect.ValueOf(node)
		if value.Kind() != reflect.Pointer || value.IsNil() {
			s.t.internalError("argument state snapshot requires non-nil pointer AST nodes", diagnostics.Location{})
			return nil
		}
		state := reflect.New(value.Elem().Type())
		state.Elem().Set(value.Elem())
		s.nodes = append(s.nodes, nodeSnapshot{node: node, state: state.Interface()})
		s.seen[node] = struct{}{}
	}
	if ref, ok := declaredSymbol(node); ok {
		s.snapshotSymbol(ref)
	}
	return s
}

// declaredSymbol returns the symbol node declares, if any. Only those symbols
// belong to the argument subtree: a referenced symbol is owned by its
// declaration, which may be resolved concurrently by another worker.
func declaredSymbol(node ast.BLangNode) (model.SymbolRef, bool) {
	switch node := node.(type) {
	case *ast.BLangVariable, *ast.BLangFunction, *ast.BLangResourceMethod, *ast.BMethodDecl,
		*ast.BLangClassDefinition, *ast.BLangNamedWorkerDeclaration, *ast.BLangTypeDefinition,
		*ast.BLangXMLNS:
		return node.(ast.NodeWithSymbol).Symbol(), true
	case *ast.BLangFunctionTypeParam:
		return node.SymbolRef, true
	case *ast.BLangMappingKeyValueField:
		if key, ok := node.Key.Expr.(ast.BNodeWithSymbol); ok {
			return key.Symbol(), true
		}
		return model.SymbolRef{}, false
	default:
		return model.SymbolRef{}, false
	}
}

func (s *argumentStateSnapshotter) VisitTypeData(_ *ast.TypeData) ast.Visitor { return s }

func (s *argumentStateSnapshotter) snapshotSymbol(ref model.SymbolRef) {
	if ref.IsEmpty() {
		return
	}
	if _, ok := s.seenRef[ref]; ok {
		return
	}
	s.seenRef[ref] = struct{}{}
	snapshot := symbolSnapshot{ref: ref, ty: s.t.symbolType(ref)}
	cx := s.t.compilerContext()
	if signature, ok := cx.FunctionTypedSignature(ref); ok {
		snapshot.function = &signature
	}
	if paramTypes, returnType, ok := cx.DependentlyTypedFunctionType(ref); ok {
		snapshot.dependent = true
		snapshot.paramTypes = paramTypes
		snapshot.returnType = returnType
	}
	s.symbols = append(s.symbols, snapshot)
}

// snapshotArgumentState captures the argument subtrees and the symbols they
// declare, and returns a function restoring them after a candidate trial.
func snapshotArgumentState(t typeResolver, args []ast.BLangExpression) func() {
	snapshotter := &argumentStateSnapshotter{
		t:       t,
		seen:    make(map[ast.BLangNode]struct{}),
		seenRef: make(map[model.SymbolRef]struct{}),
	}
	for _, arg := range args {
		ast.Walk(snapshotter, arg)
	}
	return func() {
		for i := len(snapshotter.nodes) - 1; i >= 0; i-- {
			snapshot := snapshotter.nodes[i]
			reflect.ValueOf(snapshot.node).Elem().Set(reflect.ValueOf(snapshot.state).Elem())
		}
		cx := t.compilerContext()
		for _, snapshot := range snapshotter.symbols {
			cx.SetSymbolType(snapshot.ref, snapshot.ty)
			if snapshot.function != nil && !cx.SetFunctionTypedSignature(snapshot.ref, *snapshot.function) {
				t.internalError("function symbol changed while restoring argument state", diagnostics.Location{})
				continue
			}
			if snapshot.dependent && !cx.SetDependentlyTypedFunctionType(snapshot.ref, snapshot.paramTypes, snapshot.returnType) {
				t.internalError("dependently-typed function symbol changed while restoring argument state", diagnostics.Location{})
			}
		}
	}
}
