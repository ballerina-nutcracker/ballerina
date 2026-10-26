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

package workspace

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewFileURIAcceptsFileScheme(t *testing.T) {
	t.Parallel()
	// These are intentional parser inputs, not filesystem-backed documents.
	raw, wantPath := "file:///workspace/main.bal", "/workspace/main.bal"
	if runtime.GOOS == "windows" {
		raw, wantPath = "file:///C:/workspace/main.bal", `C:\workspace\main.bal`
	}
	u, err := NewFileURI(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !u.IsFile() {
		t.Fatal("expected IsFile() to be true")
	}
	if u.Scheme() != "file" {
		t.Fatalf("scheme = %q, want %q", u.Scheme(), "file")
	}
	if u.Path() != wantPath {
		t.Fatalf("path = %q, want %q", u.Path(), wantPath)
	}
	if u.Identity() != raw {
		t.Fatalf("identity = %q, want %q", u.Identity(), raw)
	}
	if u.String() != raw {
		t.Fatalf("string = %q, want %q", u.String(), raw)
	}
}

func TestNewFileURIRejectsNonFileSchemes(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	if _, err := NewFileURI(""); err == nil {
		t.Fatal("empty URI expected error")
	}
}

func TestNewExprURI(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	// The fixed file URI intentionally tests scheme rejection without filesystem access.
	if _, err := NewBalaURI("file:///workspace/main.bal"); err == nil {
		t.Fatal("NewBalaURI with file: scheme expected error")
	}
}

func TestPathPanicsForNonFileSchemes(t *testing.T) {
	t.Parallel()
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

func TestFileURIDriveIdentityAndEscaping(t *testing.T) {
	t.Parallel()
	// Fixed drive URIs exercise parser normalization on every platform.
	lower, err := NewFileURI("file:///c:/Work/a%20b%23.bal")
	if err != nil {
		t.Fatal(err)
	}
	upper, err := NewFileURI("file:///C:/Work/a%20b%23.bal")
	if err != nil {
		t.Fatal(err)
	}
	if lower != upper || lower.String() != "file:///C:/Work/a%20b%23.bal" {
		t.Fatalf("identities: %v, %v", lower, upper)
	}
	other, err := NewFileURI("file:///C:/work/a%20b%23.bal")
	if err != nil || lower == other {
		t.Fatalf("component case was lost: %v, %v", other, err)
	}
}

func TestFileURIRejectsUnsupportedPaths(t *testing.T) {
	t.Parallel()
	// Keep these malformed parser inputs independent of native path conversion.
	for _, raw := range []string{"file://server/share/main.bal", "file:////server/share/main.bal", "file:C:app.bal", "file:///C:app.bal", "file:relative.bal", "file:///a\\\\b", "file:///a?query", "file:///a#fragment"} {
		if _, err := NewFileURI(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestDocumentURIComparableAsMapKey(t *testing.T) {
	t.Parallel()
	u1 := tempFileURI(t, "main.bal")
	u2, err := NewFileURI(u1.String())
	if err != nil {
		t.Fatal(err)
	}
	u3, err := FileURIFromPath(filepath.Join(filepath.Dir(u1.Path()), "other.bal"))
	if err != nil {
		t.Fatal(err)
	}
	m := map[DocumentURI]int{}
	m[u1] = 1
	if m[u2] != 1 {
		t.Fatal("identical URIs should map to the same key")
	}
	if _, ok := m[u3]; ok {
		t.Fatal("different URIs should not map to the same key")
	}
}
