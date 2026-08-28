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

package uri

import (
	"testing"
)

func TestNewFileURIAcceptsFileScheme(t *testing.T) {
	u, err := NewFileURI("file:///workspace/main.bal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !u.IsFile() {
		t.Fatal("expected IsFile() to be true")
	}
	if u.Scheme() != "file" {
		t.Fatalf("scheme = %q, want %q", u.Scheme(), "file")
	}
	if u.Path() != "/workspace/main.bal" {
		t.Fatalf("path = %q, want %q", u.Path(), "/workspace/main.bal")
	}
	if u.Identity() != "file:///workspace/main.bal" {
		t.Fatalf("identity = %q, want %q", u.Identity(), "file:///workspace/main.bal")
	}
	if u.String() != "file:///workspace/main.bal" {
		t.Fatalf("string = %q, want %q", u.String(), "file:///workspace/main.bal")
	}
}

func TestNewFileURIRejectsNonFileSchemes(t *testing.T) {
	schemes := []string{
		"untitled:Untitled-1",
		"expr:///some/expr",
		"ai:///some/ai",
		"bala://org/pkg/1.0.0",
		"http://example.com/main.bal",
	}
	for _, raw := range schemes {
		if _, err := NewFileURI(raw); err == nil {
			t.Fatalf("NewFileURI(%q) expected error, got nil", raw)
		}
	}
}

func TestNewFileURIRejectsMalformed(t *testing.T) {
	if _, err := NewFileURI(""); err == nil {
		t.Fatal("empty URI expected error")
	}
}

func TestNewExprURI(t *testing.T) {
	u, err := NewExprURI("expr:///workspace/expr1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Scheme() != "expr" {
		t.Fatalf("scheme = %q, want %q", u.Scheme(), "expr")
	}
	if u.IsFile() {
		t.Fatal("expected IsFile() to be false for expr: URI")
	}
}

func TestNewAIURI(t *testing.T) {
	u, err := NewAIURI("ai:///workspace/ai1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Scheme() != "ai" {
		t.Fatalf("scheme = %q, want %q", u.Scheme(), "ai")
	}
	if u.IsFile() {
		t.Fatal("expected IsFile() to be false for ai: URI")
	}
}

func TestNewBalaURISchemeValidation(t *testing.T) {
	u, err := NewBalaURI("bala://ballerina/io/1.6.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Scheme() != "bala" {
		t.Fatalf("scheme = %q, want %q", u.Scheme(), "bala")
	}
	if u.IsFile() {
		t.Fatal("expected IsFile() to be false for bala: URI")
	}
	if u.Identity() != "bala://ballerina/io/1.6.0" {
		t.Fatalf("identity = %q, want %q", u.Identity(), "bala://ballerina/io/1.6.0")
	}
}

func TestNewBalaURIRejectsWrongScheme(t *testing.T) {
	if _, err := NewBalaURI("file:///workspace/main.bal"); err == nil {
		t.Fatal("NewBalaURI with file: scheme expected error")
	}
}

func TestPathPanicsForNonFileSchemes(t *testing.T) {
	schemes := []struct {
		name string
		raw  string
		fn   func(string) (DocumentURI, error)
	}{
		{"expr", "expr:///x", NewExprURI},
		{"ai", "ai:///x", NewAIURI},
		{"bala", "bala://org/pkg/1.0.0", NewBalaURI},
	}
	for _, tc := range schemes {
		t.Run(tc.name, func(t *testing.T) {
			u, err := tc.fn(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("Path() on %s: URI expected panic", tc.name)
				}
			}()
			_ = u.Path()
		})
	}
}

func TestDocumentURIComparableAsMapKey(t *testing.T) {
	u1, _ := NewFileURI("file:///workspace/main.bal")
	u2, _ := NewFileURI("file:///workspace/main.bal")
	u3, _ := NewFileURI("file:///workspace/other.bal")
	m := map[DocumentURI]int{}
	m[u1] = 1
	if m[u2] != 1 {
		t.Fatal("identical URIs should map to the same key")
	}
	if _, ok := m[u3]; ok {
		t.Fatal("different URIs should not map to the same key")
	}
}
