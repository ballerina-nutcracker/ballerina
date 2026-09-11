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
	"path/filepath"
	"testing"
)

// TestDocumentText_RegistrationKeyMatchesWorkspaceOpenTextKey guards the one
// silent-failure-mode point in design item 2's plumbing: documentText looks
// up a document's live buffer content by moduleFileRegistrationKey(module,
// doc.Name()), and workspace.ProjectService.OpenText is keyed by the
// didOpen/didChange URI's Path(). If these two ever diverge for a named
// module in a build project, documentText's openText lookup silently misses
// and falls back to stale content — no panic, no error, just wrong
// diagnostics at wrong offsets. This asserts the two keys are identical for a
// real named-module document, not just for the default module (which every
// other test in this package already exercises implicitly).
func TestDocumentText_RegistrationKeyMatchesWorkspaceOpenTextKey(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	greetModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.greet"))
	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, "file://"+greetPath)

	docID, ok := pkg.Project().DocumentID(greetURI.Path())
	if !ok {
		t.Fatalf("DocumentID(%s): not found", greetURI.Path())
	}
	doc := greetModule.Document(docID)
	if doc == nil {
		t.Fatalf("Document(%v): not found", docID)
	}

	key := moduleFileRegistrationKey(greetModule, doc.Name())
	if key != greetURI.Path() {
		t.Fatalf("moduleFileRegistrationKey = %q, want it to equal the workspace URI path %q (openText lookups key off the URI path — a mismatch here silently falls back to stale content)", key, greetURI.Path())
	}
}
