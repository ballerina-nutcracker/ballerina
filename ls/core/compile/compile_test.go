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

package compile

import (
	"context"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/uri"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func newTestServices(t *testing.T) (*workspace.ProjectService, *CompilationService) {
	t.Helper()
	platform, _ := palnative.NewPlatform()
	bus := event.New()
	projects := workspace.New(platform, bus)
	svc := New(projects, bus, WithDebounce(0))
	t.Cleanup(func() { svc.Shutdown(); bus.Close() })
	return projects, svc
}

func fileURI(t *testing.T, raw string) uri.DocumentURI {
	t.Helper()
	u, err := uri.NewFileURI(raw)
	if err != nil {
		t.Fatalf("NewFileURI(%q): %v", raw, err)
	}
	return u
}

// applyOpen publishes content through the workspace (fresh palFS + Load) so
// Compile reads the published CurrentPackage.
func applyOpen(t *testing.T, projects *workspace.ProjectService, u uri.DocumentURI, text string) {
	t.Helper()
	if _, err := projects.Apply(context.Background(), workspace.DocumentChange{
		Kind: workspace.ChangeOpen, URI: u, Text: text, Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("Apply open: %v", err)
	}
}

func TestCompileReturnsDiagnosticsForErrorSource(t *testing.T) {
	projects, svc := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projects, u, "public function main() {\n    int x = 1;\n}\n")
	result, err := svc.Compile(context.Background(), CompileRequest{URI: u})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(result.Diagnostics))
	}
	diag := result.Diagnostics[0]
	if diag.Severity != SeverityError {
		t.Errorf("severity = %v, want Error", diag.Severity)
	}
	if diag.Code != "SEMANTIC_ERROR" {
		t.Errorf("code = %q, want SEMANTIC_ERROR", diag.Code)
	}
	if diag.Message == "" {
		t.Error("message is empty")
	}
	if diag.StartLine != 1 || diag.StartChar != 4 {
		t.Errorf("start = line %d char %d, want line 1 char 4", diag.StartLine, diag.StartChar)
	}
	if diag.EndLine != 1 || diag.EndChar != 14 {
		t.Errorf("end = line %d char %d, want line 1 char 14", diag.EndLine, diag.EndChar)
	}
}

func TestCompileReturnsNoDiagnosticsForValidSource(t *testing.T) {
	projects, svc := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, projects, u, "public function main() {}\n")
	result, err := svc.Compile(context.Background(), CompileRequest{URI: u})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %d, want 0", len(result.Diagnostics))
	}
}

func TestCompileUnknownRootReturnsEmpty(t *testing.T) {
	_, svc := newTestServices(t)
	u := fileURI(t, "file:///workspace/never-opened.bal")
	result, err := svc.Compile(context.Background(), CompileRequest{URI: u})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %d, want 0", len(result.Diagnostics))
	}
}

func TestByteOffsetToLineCharUTF16(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		offset   int
		wantLine uint32
		wantChar uint32
		wantOK   bool
	}{
		{name: "ascii start", text: "abc\ndef\n", offset: 0, wantLine: 0, wantChar: 0, wantOK: true},
		{name: "ascii mid line0", text: "abc\ndef\n", offset: 2, wantLine: 0, wantChar: 2, wantOK: true},
		{name: "ascii at newline belongs to line0 end", text: "abc\ndef\n", offset: 3, wantLine: 0, wantChar: 3, wantOK: true},
		{name: "ascii line1 start", text: "abc\ndef\n", offset: 4, wantLine: 1, wantChar: 0, wantOK: true},
		{name: "ascii end of text trailing line", text: "abc\ndef\n", offset: 8, wantLine: 2, wantChar: 0, wantOK: true},
		{name: "crlf single break line1 start", text: "ab\r\ncd", offset: 4, wantLine: 1, wantChar: 0, wantOK: true},
		{name: "crlf at cr belongs to line0 end", text: "ab\r\ncd", offset: 2, wantLine: 0, wantChar: 2, wantOK: true},
		{name: "cr-only break", text: "ab\rcd", offset: 3, wantLine: 1, wantChar: 0, wantOK: true},
		{name: "surrogate pair byte char", text: "😀x", offset: 4, wantLine: 0, wantChar: 4, wantOK: true},
		{name: "offset after surrogate at ascii", text: "😀x", offset: 5, wantLine: 0, wantChar: 5, wantOK: true},
		{name: "surrogate on later line", text: "😀\n😀z", offset: 9, wantLine: 1, wantChar: 4, wantOK: true},
		{name: "negative offset invalid", text: "abc", offset: -1, wantOK: false},
		{name: "offset past end invalid", text: "abc", offset: 4, wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lineStarts := computeLineStarts(tc.text)
			line, char, ok := byteOffsetToLineChar(tc.text, lineStarts, tc.offset)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if line != tc.wantLine {
				t.Errorf("line = %d, want %d", line, tc.wantLine)
			}
			if char != tc.wantChar {
				t.Errorf("char = %d, want %d", char, tc.wantChar)
			}
		})
	}
}
