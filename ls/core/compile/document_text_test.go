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

// Compiler registration and editor-name translation must agree for named
// modules, or live-buffer lookups silently fall back to frozen source text.
func TestDocumentText_RegistrationKeyMatchesWorkspaceOpenTextKey(t *testing.T) {
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	greetModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.greet"))
	dir := multimoduleFixtureDir(t)
	greetPath := filepath.Join(dir, "modules", "greet", "greet.bal")
	greetURI := fileURI(t, greetPath)

	name, _ := projSvc.Paths(pkg.Project()).NameOf(greetURI)
	docID, ok := pkg.Project().DocumentID(name)
	if !ok {
		t.Fatalf("DocumentID(%s): not found", greetURI.Path())
	}
	doc := greetModule.Document(docID)
	if doc == nil {
		t.Fatalf("Document(%v): not found", docID)
	}

	key := moduleFileRegistrationKey(greetModule, doc.Name())
	if key != name {
		t.Fatalf("moduleFileRegistrationKey = %q, want translated compiler name %q", key, name)
	}
}
