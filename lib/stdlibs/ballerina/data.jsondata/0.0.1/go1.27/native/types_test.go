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
	"fmt"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// TestTypeCacheReusesRecursiveFieldType walks the field of a recursive record level by level, as parsing nested
// input does, and checks the classifications stay bounded instead of growing with the depth.
func TestTypeCacheReusesRecursiveFieldType(t *testing.T) {
	t.Parallel()
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

// TestTypeCacheMissComparesOnlyEnclosingTypes classifies many unrelated recursive records, as parsing into each
// of them does, and checks a cache miss is compared with the types enclosing it rather than with every type
// classified before.
func TestTypeCacheMissComparesOnlyEnclosingTypes(t *testing.T) {
	t.Parallel()
	env := semtypes.CreateTypeEnv()
	cx := semtypes.ContextFrom(env)
	cache := newTypeCache()
	comparisons := 0
	cache.sameType = func(cx semtypes.Context, a, b semtypes.SemType) bool {
		comparisons++
		return semtypes.IsSameType(cx, a, b)
	}
	const records = 100
	for i := range records {
		md := semtypes.NewMappingDefinition()
		ld := semtypes.NewListDefinition()
		list := ld.Define(env, nil, semtypes.ListRest(md.GetSemType(env)))
		record := md.Define(env, []semtypes.Field{
			semtypes.FieldFrom(fmt.Sprintf("f%d", i), semtypes.Int, false, false),
			semtypes.FieldFrom("k", list, false, true),
		}, semtypes.CreateAnydata(cx))
		record = semtypes.Intersect(record, semtypes.CreateAnydata(cx))
		typer := newTyper(cache, cx, record, nil)
		ty := record
		for range 3 {
			listInfo := typer.classify(typer.classify(ty).mapping.FieldInnerVal("k"))
			ty = listInfo.list.Rest()
		}
	}
	if comparisons > 4*records {
		t.Fatalf("compared %d types to classify %d records", comparisons, records)
	}
}
