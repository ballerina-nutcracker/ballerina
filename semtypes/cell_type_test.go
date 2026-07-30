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
// software distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package semtypes

import (
	"testing"
)

// TestTypeCellDisparity tests type and cell type disparity
// Ported from CellTypeTest.java:testTypeCellDisparity()
func TestTypeCellDisparity(t *testing.T) {
	env := CreateTypeEnv()
	ctx := ContextFrom(env)

	tests := []struct {
		name     string
		t1       SemType
		t2       SemType
		relation relation
	}{
		{
			name:     "Int vs cell(Int, NONE)",
			t1:       Int,
			t2:       testCell(env, Int, CellMutabilityNone),
			relation: relationNoRelation,
		},
		{
			name:     "Int vs cell(Int, LIMITED)",
			t1:       Int,
			t2:       testCell(env, Int, CellMutabilityLimited),
			relation: relationNoRelation,
		},
		{
			name:     "Int vs cell(Int, UNLIMITED)",
			t1:       Int,
			t2:       testCell(env, Int, cellMutabilityUnlimited),
			relation: relationNoRelation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSemTypeRelation(t, ctx, tt.t1, tt.t2, tt.relation)
		})
	}
}

// TestBasicCellSubtyping tests basic cell subtyping
// Ported from CellTypeTest.java:testBasicCellSubtyping()
func TestBasicCellSubtyping(t *testing.T) {
	env := CreateTypeEnv()
	ctx := ContextFrom(env)

	tests := []struct {
		name      string
		t1        SemType
		t2        SemType
		relations [3]relation // [NONE, LIMITED, UNLIMITED]
	}{
		{
			name:      "Int vs Int",
			t1:        Int,
			t2:        Int,
			relations: [3]relation{relationEqual, relationEqual, relationEqual},
		},
		{
			name:      "Boolean vs Boolean",
			t1:        Boolean,
			t2:        Boolean,
			relations: [3]relation{relationEqual, relationEqual, relationEqual},
		},
		{
			name:      "Byte vs Int",
			t1:        Byte,
			t2:        Int,
			relations: [3]relation{relationSubtype, relationSubtype, relationSubtype},
		},
		{
			name:      "Boolean vs Int",
			t1:        Boolean,
			t2:        Int,
			relations: [3]relation{relationNoRelation, relationNoRelation, relationNoRelation},
		},
		{
			name:      "Boolean vs Int|Boolean",
			t1:        Boolean,
			t2:        Union(Int, Boolean),
			relations: [3]relation{relationSubtype, relationSubtype, relationSubtype},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutabilities := []CellMutability{
				CellMutabilityNone,
				CellMutabilityLimited,
				cellMutabilityUnlimited,
			}

			for i, mut := range mutabilities {
				c1 := testCell(env, tt.t1, mut)
				c2 := testCell(env, tt.t2, mut)
				actual := getSemTypeRelation(ctx, c1, c2)
				if actual != tt.relations[i] {
					t.Errorf("mutability %v: got %v, want %v", mut, actual, tt.relations[i])
				}
			}
		})
	}
}

// TestCellSubtyping1 tests cell subtyping with unions
// Ported from CellTypeTest.java:testCellSubtyping1()
func TestCellSubtyping1(t *testing.T) {
	env := CreateTypeEnv()
	ctx := ContextFrom(env)

	tests := []struct {
		name     string
		t1       SemType
		t2       SemType
		relation relation
	}{
		// Set 1
		{
			name:     "cell(Int,NONE)|cell(Boolean,NONE) vs cell(Int|Boolean,NONE)",
			t1:       Union(testCell(env, Int, CellMutabilityNone), testCell(env, Boolean, CellMutabilityNone)),
			t2:       testCell(env, Union(Int, Boolean), CellMutabilityNone),
			relation: relationEqual,
		},
		{
			name:     "cell(Int,LIMITED)|cell(Boolean,LIMITED) vs cell(Int|Boolean,LIMITED)",
			t1:       Union(testCell(env, Int, CellMutabilityLimited), testCell(env, Boolean, CellMutabilityLimited)),
			t2:       testCell(env, Union(Int, Boolean), CellMutabilityLimited),
			relation: relationSubtype,
		},
		{
			name:     "cell(Int,UNLIMITED)|cell(Boolean,UNLIMITED) vs cell(Int|Boolean,UNLIMITED)",
			t1:       Union(testCell(env, Int, cellMutabilityUnlimited), testCell(env, Boolean, cellMutabilityUnlimited)),
			t2:       testCell(env, Union(Int, Boolean), cellMutabilityUnlimited),
			relation: relationEqual,
		},
		// Set 2
		{
			name:     "cell(Int,NONE)|cell(Boolean,NONE)|cell(String,NONE) vs cell(Int|Boolean|String,NONE)",
			t1:       Union(Union(testCell(env, Int, CellMutabilityNone), testCell(env, Boolean, CellMutabilityNone)), testCell(env, String, CellMutabilityNone)),
			t2:       testCell(env, Union(Union(Int, Boolean), String), CellMutabilityNone),
			relation: relationEqual,
		},
		{
			name:     "cell(Int,LIMITED)|cell(Boolean,LIMITED)|cell(String,LIMITED) vs cell(Int|Boolean|String,LIMITED)",
			t1:       Union(Union(testCell(env, Int, CellMutabilityLimited), testCell(env, Boolean, CellMutabilityLimited)), testCell(env, String, CellMutabilityLimited)),
			t2:       testCell(env, Union(Union(Int, Boolean), String), CellMutabilityLimited),
			relation: relationSubtype,
		},
		{
			name:     "cell(Int,UNLIMITED)|cell(Boolean,UNLIMITED)|cell(String,UNLIMITED) vs cell(Int|Boolean|String,UNLIMITED)",
			t1:       Union(Union(testCell(env, Int, cellMutabilityUnlimited), testCell(env, Boolean, cellMutabilityUnlimited)), testCell(env, String, cellMutabilityUnlimited)),
			t2:       testCell(env, Union(Union(Int, Boolean), String), cellMutabilityUnlimited),
			relation: relationEqual,
		},
		// Set 3
		{
			name:     "cell(roTuple(Int),NONE)|cell(roTuple(Boolean),NONE) vs cell(roTuple(Int|Boolean),NONE)",
			t1:       Union(testCell(env, testRoTuple(env, Int), CellMutabilityNone), testCell(env, testRoTuple(env, Boolean), CellMutabilityNone)),
			t2:       testCell(env, testRoTuple(env, Union(Int, Boolean)), CellMutabilityNone),
			relation: relationEqual,
		},
		{
			name:     "cell(tuple(Int),LIMITED)|cell(tuple(Boolean),LIMITED) vs cell(tuple(Int|Boolean),LIMITED)",
			t1:       Union(testCell(env, testTuple(env, Int), CellMutabilityLimited), testCell(env, testTuple(env, Boolean), CellMutabilityLimited)),
			t2:       testCell(env, testTuple(env, Union(Int, Boolean)), CellMutabilityLimited),
			relation: relationSubtype,
		},
		{
			name:     "cell(tuple(Int),UNLIMITED)|cell(tuple(Boolean),UNLIMITED) vs cell(tuple(Int|Boolean),UNLIMITED)",
			t1:       Union(testCell(env, testTuple(env, Int), cellMutabilityUnlimited), testCell(env, testTuple(env, Boolean), cellMutabilityUnlimited)),
			t2:       testCell(env, testTuple(env, Union(Int, Boolean)), cellMutabilityUnlimited),
			relation: relationSubtype,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSemTypeRelation(t, ctx, tt.t1, tt.t2, tt.relation)
		})
	}
}

// TestCellSubtyping2 tests cell subtyping with different mutability
// Ported from CellTypeTest.java:testCellSubtyping2()
func TestCellSubtyping2(t *testing.T) {
	env := CreateTypeEnv()
	ctx := ContextFrom(env)

	tests := []struct {
		name     string
		t1       SemType
		t2       SemType
		relation relation
	}{
		// test 1
		{
			name:     "cell(Int,NONE)|cell(Boolean,UNLIMITED)|cell(String,LIMITED) vs cell(Int|Boolean|String,UNLIMITED)",
			t1:       Union(Union(testCell(env, Int, CellMutabilityNone), testCell(env, Boolean, cellMutabilityUnlimited)), testCell(env, String, CellMutabilityLimited)),
			t2:       testCell(env, Union(Union(Int, Boolean), String), cellMutabilityUnlimited),
			relation: relationSubtype,
		},
		// test 2
		{
			name:     "cell(Int|Boolean|String,NONE) vs cell(Int,NONE)|cell(Boolean,UNLIMITED)|cell(String,LIMITED)",
			t1:       testCell(env, Union(Union(Int, Boolean), String), CellMutabilityNone),
			t2:       Union(Union(testCell(env, Int, CellMutabilityNone), testCell(env, Boolean, cellMutabilityUnlimited)), testCell(env, String, CellMutabilityLimited)),
			relation: relationSubtype,
		},
		// test 3
		{
			name:     "cell(Int,NONE)|cell(Boolean,UNLIMITED)|cell(String,LIMITED) vs cell(Int|Boolean|String,LIMITED)",
			t1:       Union(Union(testCell(env, Int, CellMutabilityNone), testCell(env, Boolean, cellMutabilityUnlimited)), testCell(env, String, CellMutabilityLimited)),
			t2:       testCell(env, Union(Union(Int, Boolean), String), CellMutabilityLimited),
			relation: relationNoRelation,
		},
		// test 4
		{
			name:     "cell(Int,NONE)|cell(Int,LIMITED)|cell(Int,UNLIMITED) vs cell(Int,UNLIMITED)",
			t1:       Union(Union(testCell(env, Int, CellMutabilityNone), testCell(env, Int, CellMutabilityLimited)), testCell(env, Int, cellMutabilityUnlimited)),
			t2:       testCell(env, Int, cellMutabilityUnlimited),
			relation: relationEqual,
		},
		// test 5
		{
			name:     "cell(Int,NONE)∩cell(Int,LIMITED)∩cell(Int,UNLIMITED) vs cell(Int,UNLIMITED)",
			t1:       Intersect(Intersect(testCell(env, Int, CellMutabilityNone), testCell(env, Int, CellMutabilityLimited)), testCell(env, Int, cellMutabilityUnlimited)),
			t2:       testCell(env, Int, cellMutabilityUnlimited),
			relation: relationSubtype,
		},
		// test 6
		{
			name:     "cell(Int,NONE)∩cell(Int,LIMITED)∩cell(Int,UNLIMITED) vs cell(Int,LIMITED)",
			t1:       Intersect(Intersect(testCell(env, Int, CellMutabilityNone), testCell(env, Int, CellMutabilityLimited)), testCell(env, Int, cellMutabilityUnlimited)),
			t2:       testCell(env, Int, CellMutabilityLimited),
			relation: relationSubtype,
		},
		// test 7
		{
			name:     "cell(Int,NONE)∩cell(Int,LIMITED)∩cell(Int,UNLIMITED) vs cell(Int,NONE)",
			t1:       Intersect(Intersect(testCell(env, Int, CellMutabilityNone), testCell(env, Int, CellMutabilityLimited)), testCell(env, Int, cellMutabilityUnlimited)),
			t2:       testCell(env, Int, CellMutabilityNone),
			relation: relationEqual,
		},
		// test 8
		{
			name:     "cell(Int,NONE)∩cell(Int,LIMITED)∩cell(Byte,LIMITED) vs cell(Byte,LIMITED)",
			t1:       Intersect(Intersect(testCell(env, Int, CellMutabilityNone), testCell(env, Int, CellMutabilityLimited)), testCell(env, Byte, CellMutabilityLimited)),
			t2:       testCell(env, Byte, CellMutabilityLimited),
			relation: relationSubtype,
		},
		// test 9
		{
			name:     "cell(Int,NONE)∩(cell(Byte,LIMITED)|cell(Boolean,LIMITED)) vs cell(Byte,NONE)",
			t1:       Intersect(testCell(env, Int, CellMutabilityNone), Union(testCell(env, Byte, CellMutabilityLimited), testCell(env, Boolean, CellMutabilityLimited))),
			t2:       testCell(env, Byte, CellMutabilityNone),
			relation: relationEqual,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSemTypeRelation(t, ctx, tt.t1, tt.t2, tt.relation)
		})
	}
}
