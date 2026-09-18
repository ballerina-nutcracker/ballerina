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

//go:build !debug

package context

import (
	"testing"
	"unsafe"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// TestReleaseTracingIsANoOp asserts normal builds neither collect spans nor
// produce a recording, even when tracing is requested.
func TestReleaseTracingIsANoOp(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{Enabled: true})
	cx := NewCompilerContext(env)

	cx.StartNamedSpan("Parse", "main.bal").End()
	cx.StartPackageSpan("Desugaring", releaseTestPackageID()).End()

	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	if data != nil {
		t.Fatalf("TraceJSON = %s, want nil in a normal build", data)
	}
}

// TestReleaseTracingDoesNotAllocate guards the no-allocation contract for the
// instrumentation helpers themselves.
func TestReleaseTracingDoesNotAllocate(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{})
	cx := NewCompilerContext(env)

	pkgID := releaseTestPackageID()
	allocs := testing.AllocsPerRun(100, func() {
		cx.StartNamedSpan("Parse", "main.bal").End()
		cx.StartPackageSpan("Desugaring", pkgID).End()
	})
	if allocs != 0 {
		t.Fatalf("allocations per traced call = %v, want 0", allocs)
	}
}

// TestReleaseChildSpansAreNoOps asserts a child of any handle - zero or
// obtained from the context - records nothing and is safe to end.
func TestReleaseChildSpansAreNoOps(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{Enabled: true})
	cx := NewCompilerContext(env)

	child := TraceSpan{}.StartChild("Function", "main")
	if child != (TraceSpan{}) {
		t.Fatalf("child of the zero handle = %+v, want the zero handle", child)
	}
	child.End()

	cx.StartNamedSpan("Desugaring", "main.bal").StartChild("Function", "main").End()
	cx.StartPackageSpan("BIR Generation", releaseTestPackageID()).
		StartChild("Class", "Counter").
		StartChild("Method", "get").
		End()

	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	if data != nil {
		t.Fatalf("TraceJSON = %s, want nil in a normal build", data)
	}
}

// TestReleaseTraceSpanIsZeroSized keeps the handle free for the compiler to
// eliminate at every call site.
func TestReleaseTraceSpanIsZeroSized(t *testing.T) {
	if size := unsafe.Sizeof(TraceSpan{}); size != 0 {
		t.Fatalf("sizeof(TraceSpan) = %d, want 0", size)
	}
}

func releaseTestPackageID() *model.PackageID {
	return model.NewPackageID(
		model.DefaultPackageIDInterner, "myorg", []model.Name{"mymod"}, "1.0.0")
}

// TestReleaseNestedTracingIsANoOp asserts asking for a nested recording in a
// normal build still records nothing.
func TestReleaseNestedTracingIsANoOp(t *testing.T) {
	env := NewCompilerEnvironment(semtypes.CreateTypeEnv(), TraceOptions{Enabled: true, Nested: true})
	cx := NewCompilerContext(env)

	cx.StartNamedSpan("Desugaring", "main.bal").StartChild("Class", "Counter").End()

	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	if data != nil {
		t.Fatalf("TraceJSON = %s, want nil in a normal build", data)
	}
}
