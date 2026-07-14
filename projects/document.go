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

package projects

import (
	"ballerina-lang-go/parser/tree"
	"ballerina-lang-go/tools/text"
)

// Document represents a Ballerina source file (.bal).
// It provides access to the document's syntax tree, text content, and metadata.
// Documents are immutable - use Modify() to create modified copies.
type Document struct {
	documentCtx *documentContext
	module      *Module
}

// newDocument creates a Document from a documentContext and Module.
// This is typically called internally during Module creation.
func newDocument(ctx *documentContext, module *Module) *Document {
	return &Document{
		documentCtx: ctx,
		module:      module,
	}
}

// DocumentID returns the unique identifier for this document.
func (d *Document) DocumentID() DocumentID {
	return d.documentCtx.documentID()
}

// Name returns the document filename (e.g., "main.bal").
func (d *Document) Name() string {
	return d.documentCtx.getName()
}

// Module returns the containing module.
// This provides navigation up the object hierarchy: Document -> Module -> Package -> Project.
func (d *Document) Module() *Module {
	return d.module
}

// SyntaxTree returns the parsed syntax tree for this document.
// The syntax tree is lazily parsed and cached.
func (d *Document) SyntaxTree() *tree.SyntaxTree {
	return d.documentCtx.getSyntaxTree()
}

// TextDocument returns the text document representation.
// The text document is lazily loaded and cached.
func (d *Document) TextDocument() text.TextDocument {
	return d.documentCtx.getTextDocument()
}

// Modify returns a DocumentModifier for making immutable modifications to this document.
// Use the modifier to change document content and call Apply() to create a new Document.
func (d *Document) Modify() *DocumentModifier {
	return newDocumentModifier(d)
}

// DocumentModifier handles immutable document modifications.
// It follows the Builder pattern per project conventions.
type DocumentModifier struct {
	content    string
	name       string
	documentID DocumentID
	oldModule  *Module
}

// newDocumentModifier creates a DocumentModifier from an existing document.
func newDocumentModifier(oldDocument *Document) *DocumentModifier {
	return &DocumentModifier{
		documentID: oldDocument.DocumentID(),
		name:       oldDocument.Name(),
		content:    oldDocument.TextDocument().String(),
		oldModule:  oldDocument.module,
	}
}

// WithContent sets the new content for the document.
// Returns the modifier for method chaining.
func (dm *DocumentModifier) WithContent(content string) *DocumentModifier {
	dm.content = content
	return dm
}

// Apply creates a new Document with the modifications.
// This triggers a cascade of modifications up the object hierarchy:
// Document modification -> Module update -> Package update.
func (dm *DocumentModifier) Apply() *Document {
	// Create new DocumentConfig with updated content
	newDocConfig := NewDocumentConfig(dm.documentID, dm.name, dm.content)

	// Get disableSyntaxTree from project's compilation options
	disableSyntaxTree := dm.oldModule.Project().BuildOptions().CompilationOptions().DisableSyntaxTree()

	// Create new documentContext with the updated config
	newDocContext := newDocumentContext(newDocConfig, disableSyntaxTree, buildDiagKeyPrefix(dm.oldModule.Project(), dm.oldModule.Descriptor()))

	// Update the module with the new document context
	// This triggers module modification which cascades to package
	newModule := dm.oldModule.Modify().updateDocument(newDocContext).Apply()

	// Return the document from the new module
	return newModule.Document(dm.documentID)
}
