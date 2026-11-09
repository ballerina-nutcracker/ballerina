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
)

// TestTypeCacheReusesRecursiveFieldType walks the field of a recursive record level by level, as parsing nested
// input does, and checks the classifications stay bounded instead of growing with the depth.
func TestTypeCacheReusesRecursiveFieldType(t *testing.T) {
	env := semtypes.CreateTypeEnv()
	cx := semtypes.ContextFrom(env)
	md := semtypes.NewMappingDefinition()
	node := md.Define(env, []semtypes.Field{semtypes.FieldFrom("a", md.GetSemType(env), false, true)}, semtypes.CreateAnydata(cx))
	// A target inferred from typedesc<anydata> intersects the record with anydata, so each field type is minted afresh.
	node = semtypes.Intersect(node, semtypes.CreateAnydata(cx))
	typer := newTyper(newTypeCache(), cx, node, nil)
	ty := node
	for range 100 {
		ti := typer.classify(ty)
		if ti.kind != kindRecord {
			t.Fatalf("got kind %d, want a record", ti.kind)
		}
		ty = ti.mapping.FieldInnerVal("a")
	}
	if n := len(typer.cache.infos); n > 4 {
		t.Fatalf("cached %d classifications for one recursive record", n)
	}
}
