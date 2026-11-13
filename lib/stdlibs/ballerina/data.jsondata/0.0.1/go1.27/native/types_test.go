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

// TestTypeCacheReusesUnionWithValueSubtype classifies a record unioned with a proper subtype of a value type,
// once per value as parsing does, and checks each call reuses the first classification instead of minting new
// member types (which made parsing a list into (R|"none")[] quadratic).
func TestTypeCacheReusesUnionWithValueSubtype(t *testing.T) {
	t.Parallel()
	values := map[string]semtypes.SemType{
		"singleton": semtypes.StringConst("none"),
		"enum":      semtypes.Union(semtypes.StringConst("RED"), semtypes.StringConst("GREEN")),
		"byte":      semtypes.Byte,
		"int const": semtypes.IntConst(7),
	}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := semtypes.CreateTypeEnv()
			cx := semtypes.ContextFrom(env)
			record := closedRecord(env, semtypes.FieldFrom("a", semtypes.Int, false, false))
			cache := newTypeCache()
			union := semtypes.Union(record, value)
			field := closedRecord(env, semtypes.FieldFrom("f", union, false, false))
			interner := semtypes.NewSemtypeInterner()
			var first [2]semtypes.InternHandle
			for i := range 100 {
				typer := newTyper(cache, cx, union, nil)
				for pos, ty := range []semtypes.SemType{union, typer.classify(field).mapping.FieldInnerVal("f")} {
					ti := typer.classify(ty)
					if ti.kind != kindUnion || len(ti.members) != 2 {
						t.Fatalf("got kind %d with %d members, want a union of 2", ti.kind, len(ti.members))
					}
					handle := interner.Intern(ti.members[0])
					if i == 0 {
						first[pos] = handle
					} else if handle != first[pos] {
						t.Fatalf("classification %d minted a new record member", i)
					}
				}
			}
		})
	}
}

// TestTypeCacheKeepsValuePartsApart checks two unions that differ only in their value subtype are not given
// each other's classification.
func TestTypeCacheKeepsValuePartsApart(t *testing.T) {
	t.Parallel()
	env := semtypes.CreateTypeEnv()
	cx := semtypes.ContextFrom(env)
	record := closedRecord(env, semtypes.FieldFrom("a", semtypes.Int, false, false))
	typer := newTyper(newTypeCache(), cx, record, nil)
	for _, value := range []string{"a", "b", "a"} {
		ti := typer.classify(semtypes.Union(record, semtypes.StringConst(value)))
		member := ti.members[len(ti.members)-1]
		if !semtypes.IsSameType(cx, member, semtypes.StringConst(value)) {
			t.Fatalf("R|%q classified with value member %s", value, semtypes.ToString(cx, member))
		}
	}
}

func closedRecord(env semtypes.Env, fields ...semtypes.Field) semtypes.SemType {
	md := semtypes.NewMappingDefinition()
	return md.Define(env, fields, semtypes.Never)
}
