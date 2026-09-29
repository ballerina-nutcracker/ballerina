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

import "testing"

func objectWithFieldRegion(env Env, readonly bool, region string) SemType {
	od := NewObjectDefinition()
	return od.Define(env, ObjectQualifiersFrom(false, readonly, NetworkQualifierNone), []Member{
		{Name: "x", ValueType: Int, Kind: MemberKindField, Visibility: region},
	})
}

// Corpus tests cover region mismatches only through class and module declarations; this pins the semtypes
// contract that regions are opaque strings compared for equality, independent of how the resolver builds them.
func TestObjectMemberRegionSubtype(t *testing.T) {
	t.Parallel()
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	moduleA := objectWithFieldRegion(env, false, "testorg/a")
	moduleA2 := objectWithFieldRegion(env, false, "testorg/a")
	moduleB := objectWithFieldRegion(env, false, "testorg/b")
	private := objectWithFieldRegion(env, false, "testorg/a:C")
	if !IsSubtype(cx, moduleA, moduleA2) || !IsSubtype(cx, moduleA2, moduleA) {
		t.Errorf("objects with identical regions should be subtypes of each other")
	}
	if IsSubtype(cx, moduleA, moduleB) || IsSubtype(cx, moduleB, moduleA) {
		t.Errorf("objects with different module regions should not be subtypes of each other")
	}
	if IsSubtype(cx, private, moduleA) || IsSubtype(cx, moduleA, private) {
		t.Errorf("objects with private and module regions should not be subtypes of each other")
	}
}

// The built-in object top type and readonly object type are semtypes internals. The readonly corpus case is blocked
// by #1043 (readonly class values are not readonly), and a readonly-qualified object is not a subtype of ValReadonly
// even with public members, so this checks that readonly & T is non-empty instead.
func TestObjectMemberRegionTopTypes(t *testing.T) {
	t.Parallel()
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := objectWithFieldRegion(env, false, "testorg/a:C")
	if !IsSubtype(cx, ty, Object) {
		t.Errorf("object with a non-public region should be a subtype of object")
	}
	if !IsEmpty(cx, Diff(ty, Object)) {
		t.Errorf("object with a non-public region should not remain after removing object")
	}
	roTy := Intersect(ValReadonly, objectWithFieldRegion(env, true, "testorg/a"))
	if IsEmpty(cx, roTy) {
		t.Errorf("readonly intersection of an object with a non-public region should not be empty")
	}
}

// Corpus goldens only print regions built from real module names; this pins the printed form of an arbitrary region.
func TestObjectMemberRegionToString(t *testing.T) {
	t.Parallel()
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersDefault, []Member{
		{Name: "y", ValueType: Int, Kind: MemberKindField, Visibility: VisibilityPublic},
		{Name: "x", ValueType: Int, Kind: MemberKindField, Visibility: "testorg/vis.foo"},
		{Name: "z", ValueType: Int, Kind: MemberKindField, Visibility: "testorg/vis.foo:C"},
	})
	actual := ToString(cx, ty)
	expected := "object { int x; public int y; private int z }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}
