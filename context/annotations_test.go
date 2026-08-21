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

package context

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func newTestEnvironment() *CompilerEnvironment {
	return NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
}

func TestRecordFieldAnnotationValuesEmpty(t *testing.T) {
	env := newTestEnvironment()
	got := env.RecordFieldAnnotationValues(model.SymbolRef{Index: 1, SpaceIndex: 0})
	if got == nil {
		t.Fatal("expected an initialized map for a symbol with no annotations")
	}
	if len(got) != 0 {
		t.Errorf("expected no field annotations, got %v", got)
	}
}

func TestSetRecordFieldAnnotationValue(t *testing.T) {
	env := newTestEnvironment()
	person := model.SymbolRef{Index: 1, SpaceIndex: 0}
	env.SetRecordFieldAnnotationValue(person, "name", "org/mod:1.0.0:First", int64(1))
	env.SetRecordFieldAnnotationValue(person, "name", "org/mod:1.0.0:Second", int64(2))
	env.SetRecordFieldAnnotationValue(person, "age", "org/mod:1.0.0:First", true)

	got := env.RecordFieldAnnotationValues(person)
	if len(got) != 2 {
		t.Fatalf("expected 2 annotated fields, got %d", len(got))
	}
	if len(got["name"]) != 2 {
		t.Errorf("expected 2 annotations on 'name', got %d", len(got["name"]))
	}
	if got["age"]["org/mod:1.0.0:First"] != true {
		t.Errorf("unexpected value on 'age': %v", got["age"])
	}
}

// Field annotations are keyed by the enclosing type definition, so two record
// types must not see each other's fields.
func TestRecordFieldAnnotationValuesAreKeyedBySymbol(t *testing.T) {
	env := newTestEnvironment()
	person := model.SymbolRef{Index: 1, SpaceIndex: 0}
	other := model.SymbolRef{Index: 2, SpaceIndex: 0}
	env.SetRecordFieldAnnotationValue(person, "name", "key", int64(1))

	if got := env.RecordFieldAnnotationValues(other); len(got) != 0 {
		t.Errorf("expected no field annotations for an unrelated symbol, got %v", got)
	}
	if got := env.RecordFieldAnnotationValues(person); len(got) != 1 {
		t.Errorf("expected the original symbol to keep its annotations, got %v", got)
	}
}

// Record field annotations must not leak into the plain symbol annotation store
// that backs type-level annotations.
func TestRecordFieldAnnotationsAreSeparateFromSymbolAnnotations(t *testing.T) {
	env := newTestEnvironment()
	person := model.SymbolRef{Index: 1, SpaceIndex: 0}
	env.SetRecordFieldAnnotationValue(person, "name", "key", int64(1))

	if got := env.SymbolAnnotationValues(person); len(got) != 0 {
		t.Errorf("expected no type-level annotations, got %v", got)
	}

	env.SetSymbolAnnotationValue(person, "key", int64(9))
	if got := env.RecordFieldAnnotationValues(person)["name"]["key"]; got != int64(1) {
		t.Errorf("expected the field annotation to be unchanged, got %v", got)
	}
	if got := env.SymbolAnnotationValues(person)["key"]; got != int64(9) {
		t.Errorf("expected the type-level annotation to be stored, got %v", got)
	}
}
