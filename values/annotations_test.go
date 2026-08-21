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

package values

import (
	"slices"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestFieldAnnotationValuesSet(t *testing.T) {
	fields := NewFieldAnnotationValues()
	fields.Set("name", "org/mod:1.0.0:First", int64(1))
	fields.Set("name", "org/mod:1.0.0:Second", int64(2))
	fields.Set("age", "org/mod:1.0.0:First", true)

	if len(fields) != 2 {
		t.Fatalf("expected 2 annotated fields, got %d", len(fields))
	}
	if got := len(fields["name"]); got != 2 {
		t.Errorf("expected 2 annotations on 'name', got %d", got)
	}
	if got := fields["name"]["org/mod:1.0.0:Second"]; got != int64(2) {
		t.Errorf("unexpected value for repeated set: %v", got)
	}
	if got := fields["age"]["org/mod:1.0.0:First"]; got != true {
		t.Errorf("unexpected value on 'age': %v", got)
	}
}

func TestFieldAnnotationValuesSetOverwrites(t *testing.T) {
	fields := NewFieldAnnotationValues()
	fields.Set("name", "key", int64(1))
	fields.Set("name", "key", int64(2))
	if got := fields["name"]["key"]; got != int64(2) {
		t.Errorf("expected the later value to win, got %v", got)
	}
}

func TestFieldAnnotationValuesSortedFields(t *testing.T) {
	fields := NewFieldAnnotationValues()
	for _, name := range []string{"zeta", "alpha", "Mid", "beta"} {
		fields.Set(name, "key", true)
	}
	want := []string{"Mid", "alpha", "beta", "zeta"}
	if got := fields.SortedFields(); !slices.Equal(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestFieldAnnotationValuesSortedFieldsEmpty(t *testing.T) {
	if got := NewFieldAnnotationValues().SortedFields(); len(got) != 0 {
		t.Errorf("expected no fields, got %v", got)
	}
	var nilFields FieldAnnotationValues
	if got := nilFields.SortedFields(); len(got) != 0 {
		t.Errorf("expected no fields from a nil map, got %v", got)
	}
}

func TestNewTypeDescInitializesMaps(t *testing.T) {
	td := NewTypeDesc(semtypes.Int, nil)
	if td.Annotations == nil {
		t.Error("expected annotations to be initialized")
	}
	if td.FieldAnnotations == nil {
		t.Error("expected field annotations to be initialized")
	}
	if len(td.FieldAnnotations) != 0 {
		t.Errorf("expected no field annotations, got %v", td.FieldAnnotations)
	}
}

func TestNewTypeDescWithFieldAnnotationsKeepsValues(t *testing.T) {
	fields := NewFieldAnnotationValues()
	fields.Set("name", "key", int64(7))
	td := NewTypeDescWithFieldAnnotations(semtypes.Int, nil, fields)
	if got := td.FieldAnnotations["name"]["key"]; got != int64(7) {
		t.Errorf("expected the supplied field annotations to be kept, got %v", got)
	}
	if td.Annotations == nil {
		t.Error("expected annotations to be initialized")
	}
}
