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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package generate

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

// defaultMetamodelPath returns a plausible external location for the LSP 3.18
// metamodel when the caller has not set LSP_METAMODEL or LSP_METAMODEL_SCHEMA.
// It is intentionally outside the repository so the generator does not vendor
// the upstream model.
func defaultMetamodelPath() string {
	return "/Users/wso2/projects/resources/language-server-protocol/_specifications/lsp/3.18/metaModel/metaModel.json"
}

func defaultSchemaPath() string {
	return "/Users/wso2/projects/resources/language-server-protocol/_specifications/lsp/3.18/metaModel/metaModel.schema.json"
}

func metamodelPaths(t *testing.T) (modelPath, schemaPath string) {
	t.Helper()
	modelPath = os.Getenv("LSP_METAMODEL")
	if modelPath == "" {
		modelPath = defaultMetamodelPath()
	}
	schemaPath = os.Getenv("LSP_METAMODEL_SCHEMA")
	if schemaPath == "" {
		schemaPath = defaultSchemaPath()
	}
	if _, err := os.Stat(modelPath); err != nil {
		t.Skipf("metamodel not available at %s: %v (set LSP_METAMODEL)", modelPath, err)
	}
	if _, err := os.Stat(schemaPath); err != nil {
		t.Skipf("schema not available at %s: %v (set LSP_METAMODEL_SCHEMA)", schemaPath, err)
	}
	return modelPath, schemaPath
}

func TestParseRealMetamodel(t *testing.T) {
	modelPath, schemaPath := metamodelPaths(t)
	modelJSON, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read metamodel: %v", err)
	}
	schemaJSON, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	pm, err := Parse(modelJSON, schemaJSON)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if pm.Model.Version.Version != "3.18.0" {
		t.Fatalf("metamodel version %s, want 3.18.0", pm.Model.Version.Version)
	}
}

func TestGeneratedOutputIsDeterministicAndMatchesRepository(t *testing.T) {
	modelPath, schemaPath := metamodelPaths(t)
	modelJSON, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("read metamodel: %v", err)
	}
	schemaJSON, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	pm, err := Parse(modelJSON, schemaJSON)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	typesSrc1, jsonSrc1, err := Generate(pm, modelJSON)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	typesSrc2, jsonSrc2, err := Generate(pm, modelJSON)
	if err != nil {
		t.Fatalf("generate second: %v", err)
	}
	if !bytes.Equal(typesSrc1, typesSrc2) {
		t.Fatal("types output nondeterministic within same process")
	}
	if !bytes.Equal(jsonSrc1, jsonSrc2) {
		t.Fatal("json output nondeterministic within same process")
	}
	repoTypes, err := os.ReadFile(filepath.Join("..", "..", TypesFileName))
	if err != nil {
		t.Fatalf("read repository types file: %v", err)
	}
	repoJSON, err := os.ReadFile(filepath.Join("..", "..", JSONFileName))
	if err != nil {
		t.Fatalf("read repository json file: %v", err)
	}
	if !bytes.Equal(typesSrc1, repoTypes) {
		t.Errorf("types_generated.go differs from repository copy; regenerate with the current metamodel")
		for i := 0; i < len(typesSrc1) && i < len(repoTypes); i++ {
			if typesSrc1[i] != repoTypes[i] {
				t.Errorf("first types diff at %d: gen=%q repo=%q", i, typesSrc1[i:min(i+20, len(typesSrc1))], repoTypes[i:min(i+20, len(repoTypes))])
				break
			}
		}
	}
	if !bytes.Equal(jsonSrc1, repoJSON) {
		t.Errorf("json_generated.go differs from repository copy; regenerate with the current metamodel")
		for i := 0; i < len(jsonSrc1) && i < len(repoJSON); i++ {
			if jsonSrc1[i] != repoJSON[i] {
				t.Errorf("first json diff at %d: gen=%q repo=%q", i, jsonSrc1[i:min(i+20, len(jsonSrc1))], repoJSON[i:min(i+20, len(repoJSON))])
				break
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestPresenceWrappersRoundTrip(t *testing.T) {
	o := protocol.NewOptional(42)
	if !o.IsSet() {
		t.Fatal("Optional with value should be set")
	}
	v, ok := o.Value()
	if !ok || v != 42 {
		t.Fatalf("Optional value = %v, %v", v, ok)
	}

	var empty protocol.Optional[int]
	if empty.IsSet() {
		t.Fatal("zero Optional should not be set")
	}

	n := protocol.NewNullable("x")
	if !n.IsSet() || n.IsNull() {
		t.Fatal("Nullable value should be set and not null")
	}
	nn := protocol.NullNullable[string]()
	if !nn.IsNull() || nn.IsSet() {
		t.Fatal("Null Nullable should be null and not value-set")
	}

	on := protocol.NewOptionalNullable(7)
	if !on.IsSet() {
		t.Fatal("OptionalNullable value should be set")
	}
	onn := protocol.NullOptionalNullable[int]()
	if !onn.IsSet() || !onn.IsNull() {
		t.Fatal("Null OptionalNullable should be set and null")
	}
	var absent protocol.OptionalNullable[int]
	if absent.IsSet() {
		t.Fatal("absent OptionalNullable should not be set")
	}
}

func TestUnionWrapperVariantAccessors(t *testing.T) {
	partial := protocol.TextDocumentContentChangePartial{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 1},
		},
		Text: "x",
	}
	event := protocol.NewTextDocumentContentChangeEventTextDocumentContentChangePartial(partial)
	got, ok := event.TextDocumentContentChangePartial()
	if !ok {
		t.Fatal("expected partial variant")
	}
	if got.Text != partial.Text {
		t.Fatalf("variant text = %q, want %q", got.Text, partial.Text)
	}
	_, ok = event.TextDocumentContentChangeWholeDocument()
	if ok {
		t.Fatal("did not expect whole-document variant")
	}
}

func TestSmallModelGenerates(t *testing.T) {
	model := Model{
		Version: Metadata{
			Version: "3.18.0",
		},
		Structures: []*Structure{
			{
				Name: "Position",
				Properties: []Property{
					{Name: "line", Type: intType(), Documentation: "Line number"},
					{Name: "character", Type: intType()},
				},
			},
			{
				Name: "Range",
				Properties: []Property{
					{Name: "start", Type: refType("Position")},
					{Name: "end", Type: refType("Position")},
				},
			},
			{
				Name: "Diagnostic",
				Properties: []Property{
					{Name: "range", Type: refType("Range")},
					{Name: "severity", Type: refType("DiagnosticSeverity"), Optional: true},
					{Name: "message", Type: strType()},
					{Name: "code", Type: nullType(strType()), Optional: true},
				},
			},
		},
		Enumerations: []*Enumeration{
			{
				Name:          "DiagnosticSeverity",
				Type:          intType(),
				Documentation: "Severity levels",
				Values: []EnumerationEntry{
					{Name: "Error", Value: 1},
					{Name: "Warning", Value: 2},
				},
			},
		},
		TypeAliases: []*TypeAlias{
			{Name: "DocumentUri", Type: strType()},
		},
	}
	modelJSON, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal model: %v", err)
	}
	schemaJSON := []byte(`{"type":"object","additionalProperties":true}`)
	pm, err := Parse(modelJSON, schemaJSON)
	if err != nil {
		t.Fatalf("parse small model: %v", err)
	}
	typesSrc, jsonSrc, err := Generate(pm, modelJSON)
	if err != nil {
		t.Fatalf("generate small model: %v", err)
	}
	if !bytes.Contains(typesSrc, []byte("type Diagnostic struct")) {
		t.Error("expected Diagnostic struct in generated types")
	}
	if !bytes.Contains(typesSrc, []byte("Optional[DiagnosticSeverity")) {
		t.Error("expected Optional[DiagnosticSeverity] in generated types")
	}
	if !bytes.Contains(jsonSrc, []byte("func (o Optional")) {
		t.Error("expected Optional JSON methods in generated json")
	}
}

func intType() *Type   { return &Type{Kind: KindBase, Name: "integer"} }
func strType() *Type   { return &Type{Kind: KindBase, Name: "string"} }
func refType(name string) *Type {
	return &Type{Kind: KindReference, Name: name}
}
func nullType(t *Type) *Type {
	return &Type{Kind: KindOr, Items: []*Type{t, &Type{Kind: KindBase, Name: "null"}}}
}
