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
	state.depth++
	return func() {
		state.depth--
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
	if symbolNode, ok := node.(ast.NodeWithSymbol); ok {
		s.snapshotSymbol(symbolNode.Symbol())
	}
	return s
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
