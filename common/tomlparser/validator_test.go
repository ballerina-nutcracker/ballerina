// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package tomlparser

import (
	"os"
	"strings"
	"testing"

	"ballerina/tools/diagnostics"
)

var fsys = os.DirFS(".")

func TestSchemaFromPath(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema from path: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema should not be nil")
	}
}

func TestSchemaFromString(t *testing.T) {
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`

	schema, err := NewSchemaFromString(schemaJSON)
	if err != nil {
		t.Fatalf("Failed to create schema from string: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema should not be nil")
	}
}

func TestSchemaFromFile(t *testing.T) {
	file, err := os.Open("testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to open schema file: %v", err)
	}
	defer func() { _ = file.Close() }()

	schema, err := NewSchemaFromFile(file)
	if err != nil {
		t.Fatalf("Failed to create schema from file: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema should not be nil")
	}
}

func TestValidatorWithValidToml(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	tomlDoc, err := Read(fsys, "testdata/ballerina-package.toml")
	if err != nil {
		t.Fatalf("Failed to read TOML: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate(tomlDoc)
	if err != nil {
		t.Errorf("Validation should pass for valid TOML: %v", err)
	}

	if len(tomlDoc.Diagnostics()) > 0 {
		t.Errorf("Expected no diagnostics, but got %d", len(tomlDoc.Diagnostics()))
	}
}

func TestValidatorWithInvalidToml(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	tomlDoc, err := Read(fsys, "testdata/invalid-sample.toml")
	if err != nil {
		t.Fatalf("Failed to read TOML: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate(tomlDoc)
	if err == nil {
		t.Error("Validation should fail for invalid TOML")
	}

	if len(tomlDoc.Diagnostics()) == 0 {
		t.Error("Expected diagnostics for validation errors")
	}

	diagnostic := tomlDoc.Diagnostics()[0]
	if diagnostic.Severity != diagnostics.Error {
		t.Errorf("Expected ERROR severity, got %s", diagnostic.Severity)
	}

	if !strings.Contains(diagnostic.Message, "additionalProperties") &&
		!strings.Contains(diagnostic.Message, "unexpected") {
		t.Logf("Diagnostic message: %s", diagnostic.Message)
	}
}

func TestReadWithSchema(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	tomlDoc, err := ReadWithSchema(fsys, "testdata/ballerina-package.toml", schema)
	if err != nil {
		t.Errorf("ReadWithSchema should succeed for valid TOML: %v", err)
	}

	if tomlDoc == nil {
		t.Fatal("TOML document should not be nil")
	}

	org, ok := tomlDoc.GetString("package.org")
	if !ok || org != "foo" {
		t.Errorf("Expected org to be 'foo', got %s", org)
	}
}

func TestReadWithSchemaInvalid(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	tomlDoc, err := ReadWithSchema(fsys, "testdata/invalid-sample.toml", schema)
	if err == nil {
		t.Error("ReadWithSchema should fail for invalid TOML")
	}

	if tomlDoc == nil {
		t.Fatal("TOML document should not be nil even on validation error")
	}

	if len(tomlDoc.Diagnostics()) == 0 {
		t.Error("Expected diagnostics for validation errors")
	}
}

func TestReadStringWithSchema(t *testing.T) {
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"additionalProperties": false,
		"properties": {
			"name": {"type": "string"},
			"version": {"type": "string"}
		}
	}`

	schema, err := NewSchemaFromString(schemaJSON)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	tomlContent := `
name = "test"
version = "1.0.0"
`

	tomlDoc, err := ReadStringWithSchema(tomlContent, schema)
	if err != nil {
		t.Errorf("ReadStringWithSchema should succeed: %v", err)
	}

	if tomlDoc == nil {
		t.Fatal("TOML document should not be nil")
	}
}

func TestReadStringWithSchemaInvalid(t *testing.T) {
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"additionalProperties": false,
		"properties": {
			"name": {"type": "string"}
		}
	}`

	schema, err := NewSchemaFromString(schemaJSON)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	tomlContent := `
name = "test"
unexpected = "field"
`

	tomlDoc, err := ReadStringWithSchema(tomlContent, schema)
	if err == nil {
		t.Error("ReadStringWithSchema should fail for invalid TOML")
	}

	if len(tomlDoc.Diagnostics()) == 0 {
		t.Error("Expected diagnostics for validation errors")
	}
}

func TestReadStreamWithSchema(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	file, err := os.Open("testdata/ballerina-package.toml")
	if err != nil {
		t.Fatalf("Failed to open TOML file: %v", err)
	}
	defer func() { _ = file.Close() }()

	tomlDoc, err := ReadStreamWithSchema(file, schema)
	if err != nil {
		t.Errorf("ReadStreamWithSchema should succeed: %v", err)
	}

	if tomlDoc == nil {
		t.Fatal("TOML document should not be nil")
	}
}

func TestValidateMethod(t *testing.T) {
	tomlDoc, err := Read(fsys, "testdata/ballerina-package.toml")
	if err != nil {
		t.Fatalf("Failed to read TOML: %v", err)
	}

	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	err = tomlDoc.Validate(schema)
	if err != nil {
		t.Errorf("Validate should succeed: %v", err)
	}
}

func TestValidateMethodWithInvalidData(t *testing.T) {
	tomlDoc, err := Read(fsys, "testdata/invalid-sample.toml")
	if err != nil {
		t.Fatalf("Failed to read TOML: %v", err)
	}

	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	err = tomlDoc.Validate(schema)
	if err == nil {
		t.Error("Validate should fail for invalid TOML")
	}

	if len(tomlDoc.Diagnostics()) == 0 {
		t.Error("Expected diagnostics for validation errors")
	}
}

func TestSchemaWithRequiredFields(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/required-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	tomlDoc, err := Read(fsys, "testdata/missing-required.toml")
	if err != nil {
		t.Fatalf("Failed to read TOML: %v", err)
	}

	err = tomlDoc.Validate(schema)
	if err == nil {
		t.Error("Validation should fail when required fields are missing")
	}

	if !strings.Contains(err.Error(), "required") &&
		!strings.Contains(err.Error(), "missing") &&
		!strings.Contains(err.Error(), "version") {
		t.Logf("Error message: %s", err.Error())
	}
}

func TestSchemaValidationWithTypeError(t *testing.T) {
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"count": {"type": "number"}
		}
	}`

	schema, err := NewSchemaFromString(schemaJSON)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	tomlContent := `count = "not a number"`

	_, err = ReadStringWithSchema(tomlContent, schema)
	if err == nil {
		t.Error("Validation should fail for type mismatch")
	}
}

func TestSchemaValidationWithArrays(t *testing.T) {
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"tags": {
				"type": "array",
				"items": {"type": "string"}
			}
		}
	}`

	schema, err := NewSchemaFromString(schemaJSON)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	tomlContent := `tags = ["tag1", "tag2", "tag3"]`

	tomlDoc, err := ReadStringWithSchema(tomlContent, schema)
	if err != nil {
		t.Errorf("Validation should succeed for valid array: %v", err)
	}

	tags, ok := tomlDoc.GetArray("tags")
	if !ok || len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}
}

func TestSchemaValidationWithNestedObjects(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	tomlContent := `
[package]
org = "myorg"
name = "mypackage"
version = "1.0.0"
`

	tomlDoc, err := ReadStringWithSchema(tomlContent, schema)
	if err != nil {
		t.Errorf("Validation should succeed for nested object: %v", err)
	}

	name, ok := tomlDoc.GetString("package.name")
	if !ok || name != "mypackage" {
		t.Errorf("Expected package.name to be 'mypackage', got %s", name)
	}
}

func TestValidatorWithNilToml(t *testing.T) {
	schema, err := NewSchemaFromPath(fsys, "testdata/sample-schema.json")
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	validator := NewValidator(schema)
	err = validator.Validate(nil)
	if err == nil {
		t.Error("Validator should return error for nil TOML")
	}

	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Error should mention nil, got: %s", err.Error())
	}
}

func TestSchemaInvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid json`

	_, err := NewSchemaFromString(invalidJSON)
	if err == nil {
		t.Error("Should fail for invalid JSON schema")
	}
}
