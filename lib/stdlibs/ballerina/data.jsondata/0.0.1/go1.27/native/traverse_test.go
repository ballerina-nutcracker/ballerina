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

package native

import (
	"runtime"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestTraverseFormatsOnlyTheReturnedError converts invalid input nested in a union-typed field, where every
// level tries each union member and drops their errors, and checks the work grows linearly with the depth.
// Formatting the value into each dropped error makes the allocations grow with the square of the depth.
func TestTraverseFormatsOnlyTheReturnedError(t *testing.T) {
	// Not parallel: testing.AllocsPerRun counts allocations across the whole process.
	env := semtypes.CreateTypeEnv()
	cx := semtypes.ContextFrom(env)
	md := semtypes.NewMappingDefinition()
	nilableNode := semtypes.Union(md.GetSemType(env), semtypes.Nil)
	md.Define(env, []semtypes.Field{semtypes.FieldFrom("a", nilableNode, false, false)}, semtypes.Never)
	typer := newTyper(newTypeCache(), cx, nilableNode, nil)
	allocs := func(depth int) float64 {
		json := values.BalValue(typer.newJSONMappingBuilder().build(typer))
		for range depth {
			b := typer.newJSONMappingBuilder()
			b.put("a", json)
			json = b.build(typer)
		}
		return testing.AllocsPerRun(1, func() {
			if _, err := traverse(typer, projection{}, json, nilableNode); err == nil {
				t.Fatal("converted a value missing a required field")
			}
		})
	}
	small, large := allocs(500), allocs(1000)
	if large > 3*small {
		t.Fatalf("doubling the depth took %.0f allocations instead of %.0f", large, small)
	}
}

// TestTraverseReusesASucceededMember converts valid A|B input, where A fails only on the "b" after its nested
// A|B field succeeded, so every level tries B on the same subtree. Converting that subtree again for each
// ancestor's member makes the allocations grow with the square of the depth.
func TestTraverseReusesASucceededMember(t *testing.T) {
	// Not parallel: testing.AllocsPerRun counts allocations across the whole process.
	env := semtypes.CreateTypeEnv()
	cx := semtypes.ContextFrom(env)
	mdA, mdB := semtypes.NewMappingDefinition(), semtypes.NewMappingDefinition()
	ab := semtypes.Union(mdA.GetSemType(env), mdB.GetSemType(env))
	nilableAB := semtypes.Union(ab, semtypes.Nil)
	mdA.Define(env, []semtypes.Field{
		semtypes.FieldFrom("a", nilableAB, false, false),
		semtypes.FieldFrom("b", semtypes.String, false, true),
	}, semtypes.Never)
	mdB.Define(env, []semtypes.Field{
		semtypes.FieldFrom("a", nilableAB, false, false),
		semtypes.FieldFrom("b", semtypes.Int, false, false),
	}, semtypes.Never)
	typer := newTyper(newTypeCache(), cx, ab, nil)
	allocs := func(depth int) float64 {
		var json values.BalValue
		for range depth {
			b := typer.newJSONMappingBuilder()
			b.put("a", json)
			b.put("b", int64(1))
			json = b.build(typer)
		}
		return testing.AllocsPerRun(1, func() {
			if _, err := traverseInput(typer, projection{}, json, ab); err != nil {
				t.Fatalf("failed to convert valid input: %v", err)
			}
		})
	}
	small, large := allocs(500), allocs(1000)
	if large > 3*small {
		t.Fatalf("doubling the depth took %.0f allocations instead of %.0f", large, small)
	}
}

// TestTraverseReusesAMemberReachedThroughAMap converts invalid input into R|map<U>|U[], where R is
// record {| U? a; |}. R names the field it converts and map<U> does not, so a node is reached with every
// deque length up to its depth, and an attempt keyed by that length is converted again for each.
func TestTraverseReusesAMemberReachedThroughAMap(t *testing.T) {
	// Not parallel: testing.AllocsPerRun counts allocations across the whole process.
	env := semtypes.CreateTypeEnv()
	cx := semtypes.ContextFrom(env)
	mdR, mdMap, ldList := semtypes.NewMappingDefinition(), semtypes.NewMappingDefinition(), semtypes.NewListDefinition()
	u := semtypes.Union(semtypes.Union(mdR.GetSemType(env), mdMap.GetSemType(env)), ldList.GetSemType(env))
	mdR.Define(env, []semtypes.Field{semtypes.FieldFrom("a", semtypes.Union(u, semtypes.Nil), false, false)}, semtypes.Never)
	mdMap.Define(env, nil, u)
	ldList.Define(env, nil, semtypes.ListRest(u))
	typer := newTyper(newTypeCache(), cx, u, nil)
	allocs := func(depth int) float64 {
		leaf := typer.newJSONMappingBuilder()
		leaf.put("z", int64(1))
		json := values.BalValue(leaf.build(typer))
		for range depth {
			b := typer.newJSONMappingBuilder()
			b.put("a", json)
			json = b.build(typer)
		}
		return testing.AllocsPerRun(1, func() {
			if _, err := traverse(typer, projection{}, json, u); err == nil {
				t.Fatal("converted a value with an undefined field")
			}
		})
	}
	small, large := allocs(250), allocs(500)
	if large > 3*small {
		t.Fatalf("doubling the depth took %.0f allocations instead of %.0f", large, small)
	}
}

// TestPushFieldNameAllocatesOneSmallEntry bounds what a pushed name costs: names left on the deque stay
// reachable until the conversion returns, so a list of union-typed values holds one entry per name left.
func TestPushFieldNameAllocatesOneSmallEntry(t *testing.T) {
	// Not parallel: runtime.ReadMemStats counts allocations across the whole process.
	const names = 100000
	tr := &traverser{}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range names {
		tr.pushFieldName("a")
	}
	runtime.ReadMemStats(&after)
	if perName := (after.TotalAlloc - before.TotalAlloc) / names; perName > 32 {
		t.Fatalf("a pushed name allocated %d bytes, want at most 32", perName)
	}
}
