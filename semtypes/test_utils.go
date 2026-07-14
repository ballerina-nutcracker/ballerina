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

package semtypes

import (
	"fmt"
	"testing"
)

// testTuple creates a tuple type from the given members (mutable)
// Ported from TypeTestUtils.java:tuple()
func testTuple(env Env, members ...SemType) SemType {
	ld := NewListDefinition()
	return ld.TupleTypeWrapped(env, members...)
}

// testRoTuple creates a read-only tuple type from the given members
// Ported from TypeTestUtils.java:roTuple()
func testRoTuple(env Env, members ...SemType) SemType {
	ld := NewListDefinition()
	return ld.TupleTypeWrappedRo(env, members...)
}

// assertEqual asserts that two values are equal
func assertEqual(t *testing.T, actual, expected any, msgAndArgs ...any) {
	t.Helper()
	if actualSemType, ok := actual.(SemType); ok {
		if expectedSemType, ok := expected.(SemType); ok {
			if sameSemType(actualSemType, expectedSemType) {
				return
			}
		}
	}
	if actual != expected {
		msg := fmt.Sprintf("got %v, want %v", actual, expected)
		if len(msgAndArgs) > 0 {
			format := msgAndArgs[0].(string)
			args := msgAndArgs[1:]
			msg = fmt.Sprintf(format, args...) + ": " + msg
		}
		t.Error(msg)
	}
}

// assertTrue asserts that a condition is true
func assertTrue(t *testing.T, condition bool, msgAndArgs ...any) {
	t.Helper()
	if !condition {
		msg := "expected true"
		if len(msgAndArgs) > 0 {
			format := msgAndArgs[0].(string)
			args := msgAndArgs[1:]
			msg = fmt.Sprintf(format, args...)
		}
		t.Error(msg)
	}
}

// assertFalse asserts that a condition is false
func assertFalse(t *testing.T, condition bool, msgAndArgs ...any) {
	t.Helper()
	if condition {
		msg := "expected false"
		if len(msgAndArgs) > 0 {
			format := msgAndArgs[0].(string)
			args := msgAndArgs[1:]
			msg = fmt.Sprintf(format, args...)
		}
		t.Error(msg)
	}
}

// Relation represents the subtype relationship between two types
// Ported from CellTypeTest.java:Relation enum
type Relation string

const (
	RelationEqual      Relation = "="
	RelationSubtype    Relation = "<"
	RelationNoRelation Relation = "<>"
)

// getSemTypeRelation determines the relationship between two types
// Ported from CellTypeTest.java:getSemTypeRelation()
func getSemTypeRelation(ctx Context, t1, t2 SemType) Relation {
	s1 := IsSubtype(ctx, t1, t2)
	s2 := IsSubtype(ctx, t2, t1)

	if s1 && s2 {
		return RelationEqual
	} else if s1 {
		return RelationSubtype
	} else if s2 {
		// If t2 <: t1 but not t1 <: t2, this is a '>' relation
		// which should be converted to '<' by swapping arguments
		panic("'>' relation found which can be converted to a '<' relation")
	} else {
		return RelationNoRelation
	}
}

// assertSemTypeRelation asserts that two types have the expected relationship
// Ported from CellTypeTest.java:assertSemTypeRelation()
func assertSemTypeRelation(t *testing.T, ctx Context, t1, t2 SemType, expected Relation) {
	t.Helper()
	actual := getSemTypeRelation(ctx, t1, t2)
	if actual != expected {
		t.Errorf("type relation: got %v, want %v", actual, expected)
	}
}

// testCell creates a cell type containing the given type with specified mutability
// Ported from CellTypeTest.java:cell()
func testCell(env Env, ty SemType, mut CellMutability) SemType {
	return cellContainingWithEnvSemTypeCellMutability(env, ty, mut)
}
