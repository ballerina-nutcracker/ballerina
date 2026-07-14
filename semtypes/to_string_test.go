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
	"testing"

	"ballerina-lang-go/decimal"
)

func TestSimpleBasicType(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := Union(INT, STRING)
	actual := ToString(cx, ty)
	expected := "int|string"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestIntSingleton(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := IntConst(-10)
	actual := ToString(cx, ty)
	expected := "-10"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestIntUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	t1 := IntConst(-10)
	t2 := IntConst(10)
	actual := ToString(cx, Union(t1, t2))
	expected := "-10|10"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestIntUnion2(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	t1 := IntConst(1)
	t2 := IntConst(2)
	t3 := IntConst(3)
	actual := ToString(cx, Union(Union(t1, t2), t3))
	expected := "1|2|3"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestSpecialIntSubtypes(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	type testCase struct {
		ty       SemType
		expected string
	}
	var cases []testCase
	cases = append(cases, testCase{UINT8, "int:Unsigned8"})
	cases = append(cases, testCase{BYTE, "int:Unsigned8"})
	cases = append(cases, testCase{UINT16, "int:Unsigned16"})
	cases = append(cases, testCase{UINT32, "int:Unsigned32"})

	cases = append(cases, testCase{SINT8, "int:Signed8"})
	cases = append(cases, testCase{SINT16, "int:Signed16"})
	cases = append(cases, testCase{SINT32, "int:Signed32"})
	for _, each := range cases {
		actual := ToString(cx, each.ty)
		if actual != each.expected {
			t.Errorf("got %s expected %s", actual, each.expected)
		}
	}
}

func TestSpecialStringSubtypes(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, CHAR)
	expected := "string:Char"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestStringUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	t1 := StringConst("a")
	t2 := StringConst("bb")
	actual := ToString(cx, Union(t1, t2))
	expected := "\"a\"|\"bb\""
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestBasicTypeUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, Union(StringConst("a"), INT))
	expected := "int|\"a\""
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestBooleanSingleton(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, BooleanConst(true))
	expected := "true"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFloatSingleton(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, FloatConst(1.5))
	expected := "1.5"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestDecimalSingleton(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	val, err := decimal.FromString("1.5")
	if err != nil {
		t.Fatalf("failed to parse decimal: %v", err)
	}
	actual := ToString(cx, DecimalConst(*val))
	expected := "1.5"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestNilType(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, NIL)
	expected := "nil"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestListAtomicType(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ld := NewListDefinition()
	ty := ld.DefineListTypeWrapped(env, nil, 0, INT, CellMutability_CELL_MUT_LIMITED)
	actual := ToString(cx, ty)
	expected := "[int...]"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestListAtomicType1(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ld := NewListDefinition()
	ty := ld.DefineListTypeWrapped(env, []SemType{STRING}, 3, INT, CellMutability_CELL_MUT_LIMITED)
	actual := ToString(cx, ty)
	expected := "[string, string, string, int...]"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestListTypeUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ld1 := NewListDefinition()
	ty1 := ld1.DefineListTypeWrapped(env, []SemType{STRING}, 3, INT, CellMutability_CELL_MUT_LIMITED)

	ld2 := NewListDefinition()
	ty2 := ld2.DefineListTypeWrapped(env, nil, 0, INT, CellMutability_CELL_MUT_LIMITED)
	actual := ToString(cx, Union(ty1, ty2))
	expected := "[string, string, string, int...]|[int...]"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestListTypeDiff(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ld1 := NewListDefinition()
	ty1 := ld1.DefineListTypeWrapped(env, nil, 0, INT, CellMutability_CELL_MUT_LIMITED)

	ld2 := NewListDefinition()
	ty2 := ld2.DefineListTypeWrapped(env, nil, 0, SINT32, CellMutability_CELL_MUT_LIMITED)
	actual := ToString(cx, Diff(ty1, ty2))
	expected := "[int...]&¬[int:Signed32...]"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestListTypeIntersect(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ld1 := NewListDefinition()
	ty1 := ld1.DefineListTypeWrapped(env, nil, 0, INT, CellMutability_CELL_MUT_LIMITED)

	ld2 := NewListDefinition()
	ty2 := ld2.DefineListTypeWrapped(env, nil, 0, SINT32, CellMutability_CELL_MUT_LIMITED)
	actual := ToString(cx, Intersect(ty1, ty2))
	expected := "[int...]&[int:Signed32...]"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestMappingAtomicType(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	md := NewMappingDefinition()
	ty := md.DefineMappingTypeWrapped(env, nil, INT)
	actual := ToString(cx, ty)
	expected := "{| int... |}"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestMappingWithFields(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	md := NewMappingDefinition()
	fields := []Field{
		{Name: "name", Ty: STRING},
		{Name: "age", Ty: INT},
	}
	ty := md.DefineMappingTypeWrapped(env, fields, NEVER)
	actual := ToString(cx, ty)
	expected := "{| age: int, name: string, never... |}"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestMappingTypeUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	md1 := NewMappingDefinition()
	ty1 := md1.DefineMappingTypeWrapped(env, []Field{{Name: "x", Ty: INT}}, NEVER)

	md2 := NewMappingDefinition()
	ty2 := md2.DefineMappingTypeWrapped(env, []Field{{Name: "y", Ty: STRING}}, NEVER)
	actual := ToString(cx, Union(ty1, ty2))
	expected := "{| x: int, never... |}|{| y: string, never... |}"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestMappingTypeDiff(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	md1 := NewMappingDefinition()
	ty1 := md1.DefineMappingTypeWrapped(env, nil, INT)

	md2 := NewMappingDefinition()
	ty2 := md2.DefineMappingTypeWrapped(env, nil, SINT32)
	actual := ToString(cx, Diff(ty1, ty2))
	expected := "{| int... |}&¬{| int:Signed32... |}"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestMappingTypeIntersect(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	md1 := NewMappingDefinition()
	ty1 := md1.DefineMappingTypeWrapped(env, nil, INT)

	md2 := NewMappingDefinition()
	ty2 := md2.DefineMappingTypeWrapped(env, nil, SINT32)
	actual := ToString(cx, Intersect(ty1, ty2))
	expected := "{| int... |}&{| int:Signed32... |}"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestMappingTypeRO(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	md := NewMappingDefinition()
	ty := md.DefineMappingTypeWrapped(env, nil, INT)
	actual := ToString(cx, Intersect(ty, VAL_READONLY))
	expected := "readonly&{| int... |}"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestListTypeRO(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ld1 := NewListDefinition()
	ty1 := ld1.DefineListTypeWrapped(env, nil, 0, INT, CellMutability_CELL_MUT_LIMITED)

	ty2 := VAL_READONLY
	actual := ToString(cx, Intersect(ty1, ty2))
	expected := "readonly&[int...]"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFunctionType(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := funcHelper(env, createTupleType(env, INT), INT)
	actual := ToString(cx, ty)
	expected := "function(int) returns int"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFunctionTypeMultipleParams(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := funcHelper(env, createTupleType(env, INT, STRING), INT)
	actual := ToString(cx, ty)
	expected := "function(int, string) returns int"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFunctionTypeNoParams(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := funcHelper(env, createTupleType(env), NIL)
	actual := ToString(cx, ty)
	expected := "function() returns nil"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFunctionTypeUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty1 := funcHelper(env, createTupleType(env, INT), INT)
	ty2 := funcHelper(env, createTupleType(env, STRING), STRING)
	actual := ToString(cx, Union(ty1, ty2))
	expected := "function(int) returns int|function(string) returns string"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFunctionTypeIntersect(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty1 := funcHelper(env, createTupleType(env, INT), INT)
	ty2 := funcHelper(env, createTupleType(env, STRING), STRING)
	actual := ToString(cx, Intersect(ty1, ty2))
	expected := "function(int) returns int&function(string) returns string"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFunctionTypeWithUnionReturn(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := funcHelper(env, createTupleType(env, INT), Union(INT, NIL))
	actual := ToString(cx, ty)
	expected := "function(int) returns nil|int"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestFunctionTypeWithUnionParams(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := funcHelper(env, createTupleType(env, Union(INT, STRING)), NIL)
	actual := ToString(cx, ty)
	expected := "function(int|string) returns nil"
	if actual != expected {
		t.Errorf("got %s expected %s", actual, expected)
	}
}

func TestObjectSimpleFields(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
		{Name: "y", ValueTy: STRING, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, ty)
	expected := "object { public int x; public string y }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectWithMethod(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	methodTy := funcHelper(env, createTupleType(env, INT, INT), INT)
	ty := od.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "foo", ValueTy: methodTy, Kind: MemberKindMethod, Visibility: VisibilityPublic, Immutable: true},
	})
	actual := ToString(cx, ty)
	expected := "object { public function foo(int, int) returns int }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectWithFieldsAndMethods(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	methodTy := funcHelper(env, createTupleType(env, INT, INT), INT)
	ty := od.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "bar", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
		{Name: "foo", ValueTy: methodTy, Kind: MemberKindMethod, Visibility: VisibilityPublic, Immutable: true},
	})
	actual := ToString(cx, ty)
	expected := "object { public int bar; public function foo(int, int) returns int }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectIsolated(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersFrom(true, false, NetworkQualifierNone), []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, ty)
	expected := "isolated object { public int x }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectClient(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersFrom(false, false, NetworkQualifierClient), []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, ty)
	expected := "client object { public int x }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectService(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersFrom(false, false, NetworkQualifierService), []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, ty)
	expected := "service object { public int x }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectIsolatedService(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersFrom(true, false, NetworkQualifierService), []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, ty)
	expected := "isolated service object { public int x }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectRemoteMethod(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	methodTy := funcHelper(env, createTupleType(env, INT), STRING)
	ty := od.Define(env, ObjectQualifiersFrom(false, false, NetworkQualifierClient), []Member{
		{Name: "ping", ValueTy: methodTy, Kind: MemberKindRemoteMethod, Visibility: VisibilityPublic, Immutable: true},
	})
	actual := ToString(cx, ty)
	expected := "client object { public remote function ping(int) returns string }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectResourceMethod(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	methodTy := funcHelper(env, createTupleType(env), NIL)
	ty := od.Define(env, ObjectQualifiersFrom(false, false, NetworkQualifierService), []Member{
		{Name: "get", ValueTy: methodTy, Kind: MemberKindResourceMethod, Visibility: VisibilityPublic, Immutable: true},
	})
	actual := ToString(cx, ty)
	expected := "service object { public resource function get() returns nil }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectEmpty(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersDEFAULT, nil)
	actual := ToString(cx, ty)
	expected := "object"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectReadonlyIntersect(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od := NewObjectDefinition()
	ty := od.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, Intersect(ty, VAL_READONLY))
	expected := "readonly&object { public int x }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectTypeUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od1 := NewObjectDefinition()
	ty1 := od1.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	od2 := NewObjectDefinition()
	ty2 := od2.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "y", ValueTy: STRING, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, Union(ty1, ty2))
	expected := "object { public int x }|object { public string y }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectTypeIntersect(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od1 := NewObjectDefinition()
	ty1 := od1.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	od2 := NewObjectDefinition()
	ty2 := od2.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "y", ValueTy: STRING, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, Intersect(ty1, ty2))
	expected := "object { public int x }&object { public string y }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestObjectTypeDiff(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	od1 := NewObjectDefinition()
	ty1 := od1.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "x", ValueTy: INT, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	od2 := NewObjectDefinition()
	ty2 := od2.Define(env, ObjectQualifiersDEFAULT, []Member{
		{Name: "y", ValueTy: STRING, Kind: MemberKindField, Visibility: VisibilityPublic},
	})
	actual := ToString(cx, Diff(ty1, ty2))
	expected := "object { public int x }&¬object { public string y }"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestTypedescUnconstrained(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, TYPEDESC)
	expected := "typedesc"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLTop(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, XML)
	expected := "xml"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestTypedescConstrained(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := TypedescContaining(env, INT)
	actual := ToString(cx, ty)
	expected := "typedesc<int>"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLElement(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, XML_ELEMENT)
	expected := "xml:Element"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLComment(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, XML_COMMENT)
	expected := "xml:Comment"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLText(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, XML_TEXT)
	expected := "xml:Text"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLProcessingInstruction(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	actual := ToString(cx, XML_PI)
	expected := "xml:ProcessingInstruction"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLSequenceOfElement(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := XMLSequence(XML_ELEMENT)
	actual := ToString(cx, ty)
	expected := "xml<xml:Element>"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestTypedescConstrainedUnion(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := TypedescContaining(env, Union(INT, STRING))
	actual := ToString(cx, ty)
	expected := "typedesc<int|string>"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLSequenceOfComment(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := XMLSequence(XML_COMMENT)
	actual := ToString(cx, ty)
	expected := "xml<xml:Comment>"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLSequenceOfPI(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := XMLSequence(XML_PI)
	actual := ToString(cx, ty)
	expected := "xml<xml:ProcessingInstruction>"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}

func TestXMLNeverSequence(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	ty := XMLSingleton(XML_PRIMITIVE_NEVER)
	actual := ToString(cx, ty)
	expected := "xml<never>"
	if actual != expected {
		t.Errorf("got %q expected %q", actual, expected)
	}
}
