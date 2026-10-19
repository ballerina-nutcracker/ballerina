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

// document_text.go supplies documentText, ticket 28's replacement for reading
// a document's content straight off projects.Document.TextDocument(). Ticket
// 28 drops the ADR-042 modifier chain (Document.Modify().WithContent()
// .Apply()), which used to be the only way an edit's new content ever reached
// the compiler's Package graph. Without it, a document's projects.Document
// stays frozen at whatever content it had when the package was first loaded;
// the workspace package's open-buffer map (ls/core/workspace.ProjectService
// .OpenText) becomes the single source of truth for a document's current
// text once it is open. documentText checks that live buffer first, falling
// back to the frozen TextDocument() for a document that was never opened
// through the workspace (e.g. an unopened dependency file read straight off
// disk at Load time).
package compile

import "github.com/ballerina-nutcracker/ballerina/projects"

// textProvider scopes compiler-relative names to their owning project. External
// packages have no editor buffers and fall back to their frozen source text.
type textProvider func(project projects.Project, name string) (string, bool)

// documentText reads the live editor buffer through the project's compiler
// registration name, falling back to the document captured at load time.
func documentText(module *projects.Module, doc *projects.Document, openText textProvider) string {
	if openText != nil {
		path := moduleFileRegistrationKey(module, doc.Name())
		if text, ok := openText(module.Project(), path); ok {
			return text
		}
	}
	return doc.TextDocument().String()
}
