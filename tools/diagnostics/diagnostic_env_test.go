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

package diagnostics

import (
	"testing"

	"ballerina-lang-go/tools/text"
)

// TestRegisterFile_SamePointerTwice: registering the same TextDocument
// object under the same name twice must not panic.
func TestRegisterFile_SamePointerTwice(t *testing.T) {
	de := NewDiagnosticEnv()
	doc := text.TextDocumentFromText("content")
	de.RegisterFile("a.bal", doc)
	de.RegisterFile("a.bal", doc)
	if got := de.FileIndex("a.bal"); got != 1 {
		t.Errorf("FileIndex(%q) = %d, want 1", "a.bal", got)
	}
}

// TestRegisterFile_SameContentDifferentInstance covers re-registering a
// file with a distinct but content-identical TextDocument instance (e.g. a
// dependency re-parsed into a new object). Regression test: an off-by-one
// bug compared against the wrong slot in de.docs, panicking on a false
// mismatch.
func TestRegisterFile_SameContentDifferentInstance(t *testing.T) {
	de := NewDiagnosticEnv()
	// Fill earlier slots so the off-by-one bug would land on the wrong one.
	de.RegisterFile("b.bal", text.TextDocumentFromText("b content"))
	de.RegisterFile("c.bal", text.TextDocumentFromText("c content"))
	de.RegisterFile("a.bal", text.TextDocumentFromText("shared content"))

	// Re-parsed into a brand-new object, but with the same content.
	de.RegisterFile("a.bal", text.TextDocumentFromText("shared content"))

	loc := NewLocation(de, "a.bal", 0, 4)
	if got := de.FileName(loc); got != "a.bal" {
		t.Errorf("FileName after re-registration = %q, want %q", got, "a.bal")
	}
	if got := de.StartLine(loc); got != 0 {
		t.Errorf("StartLine after re-registration = %d, want 0", got)
	}
}

// TestRegisterFile_DifferentContentPanics covers a genuine name collision
// with different content — the content-based fallback must not mask it.
func TestRegisterFile_DifferentContentPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected RegisterFile to panic on a genuine name collision with different content")
		}
	}()
	de := NewDiagnosticEnv()
	de.RegisterFile("a.bal", text.TextDocumentFromText("original content"))
	de.RegisterFile("a.bal", text.TextDocumentFromText("different content"))
}
