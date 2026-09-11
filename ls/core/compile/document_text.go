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

// textProvider resolves a document's live, currently-open-buffer content by
// its absolute file path. It is nil for callers that have no live-buffer
// source (e.g. package-level Go tests constructing a packageDriver directly)
// — documentText falls back to the document's own TextDocument() in that
// case, matching pre-ticket-28 behavior exactly.
type textProvider func(filePath string) (string, bool)

// documentText returns doc's current text: openText's live buffer content if
// one exists for doc's file path, otherwise doc's own (possibly stale)
// TextDocument(). The lookup key is moduleFileRegistrationKey(module,
// doc.Name()), not project.DocumentPath(doc.DocumentID()) — the latter is
// unreliable for a SingleFileProject, whose DocumentPath returns the
// document's bare name rather than a source-root-joined path (see
// moduleFileRegistrationKey's own doc comment in stage.go), so it would never
// match the absolute path openText's caller (workspace.ProjectService
// .OpenText) is keyed by. moduleFileRegistrationKey's composition equals the
// document's absolute file path for a build or single-file project, which is
// exactly the URI path a didChange/didOpen resolves to.
func documentText(module *projects.Module, doc *projects.Document, openText textProvider) string {
	if openText != nil {
		path := moduleFileRegistrationKey(module, doc.Name())
		if text, ok := openText(path); ok {
			return text
		}
	}
	return doc.TextDocument().String()
}
