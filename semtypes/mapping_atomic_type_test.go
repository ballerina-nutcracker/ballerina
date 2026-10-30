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

func TestMappingAtomicTypeRest(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	fields := []Field{{name: "a", typeOf: Int}}
	tests := []struct {
		name string
		rest SemType
	}{
		{"closed", Never},
		{"typed rest", String},
		{"open", CreateAnydata(cx)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewMappingDefinition()
			atomic := ToMappingAtomicType(cx, md.Define(env, fields, tt.rest))
			if atomic == nil {
				t.Fatal("expected a mapping atomic type")
			}
			if !IsSameType(cx, atomic.Rest(), tt.rest) {
				t.Errorf("got rest %s expected %s", ToString(cx, atomic.Rest()), ToString(cx, tt.rest))
			}
		})
	}
}

func TestMappingAtomicTypeFields(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	md := NewMappingDefinition()
	fields := []Field{
		{name: "name", typeOf: String},
		{name: "age", typeOf: Int, optional: true},
	}
	atomic := ToMappingAtomicType(cx, md.Define(env, fields, Boolean))
	if atomic == nil {
		t.Fatal("expected a mapping atomic type")
	}
	if got := atomic.FieldNames(); len(got) != 2 || got[0] != "age" || got[1] != "name" {
		t.Errorf("got field names %v expected [age name]", got)
	}
	innerVals := []struct {
		field    string
		expected SemType
	}{
		{"name", String},
		{"age", Int},
		{"other", Boolean},
	}
	for _, tt := range innerVals {
		if got := atomic.FieldInnerVal(tt.field); !IsSameType(cx, got, tt.expected) {
			t.Errorf("FieldInnerVal(%q): got %s expected %s", tt.field, ToString(cx, got), ToString(cx, tt.expected))
		}
	}
	if !IsSameType(cx, CellInnerVal(atomic.FieldCell("other")), atomic.Rest()) {
		t.Error("FieldCell of an undeclared field should be the rest cell")
	}
	optional := map[string]bool{"name": false, "age": true, "other": true}
	for field, expected := range optional {
		if got := atomic.IsOptional(cx, field); got != expected {
			t.Errorf("IsOptional(%q): got %v expected %v", field, got, expected)
		}
	}
}
