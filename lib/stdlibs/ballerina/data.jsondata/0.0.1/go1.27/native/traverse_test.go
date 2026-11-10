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
