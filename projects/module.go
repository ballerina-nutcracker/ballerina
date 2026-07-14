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
	"sync"
)

// Module represents a Ballerina module.
// A module is a collection of Ballerina source files that belong together.
// Modules are immutable - use Modify() to create modified copies.
type Module struct {
	moduleCtx       *moduleContext
	packageInstance *Package

	// Lazy-loaded document caches (thread-safe)
	srcDocs     sync.Map // map[DocumentID]*Document
	testSrcDocs sync.Map // map[DocumentID]*Document
}

// newModule creates a Module from a moduleContext and Package.
// This is typically called internally during Package creation.
func newModule(ctx *moduleContext, packageInstance *Package) *Module {
	return &Module{
		moduleCtx:       ctx,
		packageInstance: packageInstance,
	}
}

// ModuleID returns the unique identifier for this module.
func (m *Module) ModuleID() ModuleID {
	return m.moduleCtx.getModuleID()
}

// ModuleName returns the module name.
func (m *Module) ModuleName() ModuleName {
	return m.moduleCtx.getModuleName()
}

// Descriptor returns the module descriptor containing metadata.
func (m *Module) Descriptor() ModuleDescriptor {
	return m.moduleCtx.getDescriptor()
}

// PackageInstance returns the containing package.
// This provides navigation up the object hierarchy: Document -> Module -> Package -> Project.
func (m *Module) PackageInstance() *Package {
	return m.packageInstance
}

// DocumentIDs returns a defensive copy of source document IDs.
func (m *Module) DocumentIDs() []DocumentID {
	return m.moduleCtx.getSrcDocumentIDs()
}

// TestDocumentIDs returns a defensive copy of test document IDs.
func (m *Module) TestDocumentIDs() []DocumentID {
	return m.moduleCtx.getTestSrcDocumentIDs()
}

// Document returns a document by ID.
// Documents are lazily loaded and cached using sync.Map for thread safety.
// Returns nil if the document is not found in this module.
func (m *Module) Document(documentID DocumentID) *Document {
	// Check source documents first
	if doc, ok := m.srcDocs.Load(documentID); ok {
		return doc.(*Document)
	}

	// Check test documents
	if doc, ok := m.testSrcDocs.Load(documentID); ok {
		return doc.(*Document)
	}

	// Try to load from context
	docCtx := m.moduleCtx.getDocumentContext(documentID)
	if docCtx == nil {
		return nil
	}

	// Create and cache the document
	newDoc := newDocument(docCtx, m)

	// Determine which cache to use based on document type
	if m.moduleCtx.isTestDocument(documentID) {
		actual, _ := m.testSrcDocs.LoadOrStore(documentID, newDoc)
		return actual.(*Document)
	}

	actual, _ := m.srcDocs.LoadOrStore(documentID, newDoc)
	return actual.(*Document)
}

// IsDefaultModule returns true if this is the default module of the package.
func (m *Module) IsDefaultModule() bool {
	return m.moduleCtx.isDefault()
}

// Project returns the project reference.
// This provides navigation up the object hierarchy to the project level.
func (m *Module) Project() Project {
	return m.moduleCtx.getProject()
}

// Modify returns a ModuleModifier for making immutable modifications to this module.
// Use the modifier to add/remove/update documents and call Apply() to create a new Module.
func (m *Module) Modify() *ModuleModifier {
	return newModuleModifier(m)
}

// ModuleModifier handles immutable module modifications.
// It follows the Builder pattern per project conventions.
type ModuleModifier struct {
	moduleID          ModuleID
	moduleDescriptor  ModuleDescriptor
	srcDocIDs         []DocumentID // Ordered list of source document IDs
	testSrcDocIDs     []DocumentID // Ordered list of test document IDs
	srcDocContextMap  map[DocumentID]*documentContext
	testDocContextMap map[DocumentID]*documentContext
	isDefaultModule   bool
	dependencies      []ModuleDescriptor
	packageInstance   *Package
	project           Project
}

// newModuleModifier creates a ModuleModifier from an existing module.
func newModuleModifier(oldModule *Module) *ModuleModifier {
	// Get copies of the document context maps and ordered ID slices
	srcDocContextMap := oldModule.moduleCtx.getSrcDocContextMap()
	testDocContextMap := oldModule.moduleCtx.getTestDocContextMap()
	srcDocIDs := oldModule.moduleCtx.getSrcDocumentIDs()
	testSrcDocIDs := oldModule.moduleCtx.getTestSrcDocumentIDs()

	return &ModuleModifier{
		moduleID:          oldModule.ModuleID(),
		moduleDescriptor:  oldModule.Descriptor(),
		srcDocIDs:         srcDocIDs,
		testSrcDocIDs:     testSrcDocIDs,
		srcDocContextMap:  srcDocContextMap,
		testDocContextMap: testDocContextMap,
		isDefaultModule:   oldModule.IsDefaultModule(),
		dependencies:      oldModule.moduleCtx.getModuleDescDependencies(),
		packageInstance:   oldModule.packageInstance,
		project:           oldModule.Project(),
	}
}

// AddDocument adds a source document to the module.
// Returns the modifier for method chaining.
func (mm *ModuleModifier) AddDocument(documentConfig DocumentConfig) *ModuleModifier {
	docID := documentConfig.DocumentID()
	disableSyntaxTree := mm.project.BuildOptions().CompilationOptions().DisableSyntaxTree()
	docContext := newDocumentContext(documentConfig, disableSyntaxTree, buildDiagKeyPrefix(mm.project, mm.moduleDescriptor))
	mm.srcDocContextMap[docID] = docContext
	mm.srcDocIDs = append(mm.srcDocIDs, docID)
	return mm
}

// AddTestDocument adds a test document to the module.
// Returns the modifier for method chaining.
func (mm *ModuleModifier) AddTestDocument(documentConfig DocumentConfig) *ModuleModifier {
	docID := documentConfig.DocumentID()
	disableSyntaxTree := mm.project.BuildOptions().CompilationOptions().DisableSyntaxTree()
	docContext := newDocumentContext(documentConfig, disableSyntaxTree, buildDiagKeyPrefix(mm.project, mm.moduleDescriptor))
	mm.testDocContextMap[docID] = docContext
	mm.testSrcDocIDs = append(mm.testSrcDocIDs, docID)
	return mm
}

// RemoveDocument removes a document by ID.
// The document is removed from both source and test collections.
// Returns the modifier for method chaining.
func (mm *ModuleModifier) RemoveDocument(documentID DocumentID) *ModuleModifier {
	delete(mm.srcDocContextMap, documentID)
	delete(mm.testDocContextMap, documentID)
	// Remove from ordered slices
	mm.srcDocIDs = removeDocumentID(mm.srcDocIDs, documentID)
	mm.testSrcDocIDs = removeDocumentID(mm.testSrcDocIDs, documentID)
	return mm
}

// removeDocumentID removes a DocumentID from a slice while preserving order.
func removeDocumentID(ids []DocumentID, toRemove DocumentID) []DocumentID {
	for i, id := range ids {
		if id.Equals(toRemove) {
			return append(ids[:i], ids[i+1:]...)
		}
	}
	return ids
}

// UpdateDocument updates a document by replacing its context with a new one.
// Returns the modifier for method chaining.
func (mm *ModuleModifier) UpdateDocument(documentConfig DocumentConfig) *ModuleModifier {
	disableSyntaxTree := mm.project.BuildOptions().CompilationOptions().DisableSyntaxTree()
	docContext := newDocumentContext(documentConfig, disableSyntaxTree, buildDiagKeyPrefix(mm.project, mm.moduleDescriptor))
	return mm.updateDocument(docContext)
}

// updateDocument is an internal method that updates a document context directly.
// This is used by DocumentModifier.Apply() to cascade changes.
func (mm *ModuleModifier) updateDocument(newDocContext *documentContext) *ModuleModifier {
	docID := newDocContext.documentID()

	// Check if it's a source or test document and update accordingly
	if _, ok := mm.srcDocContextMap[docID]; ok {
		mm.srcDocContextMap[docID] = newDocContext
	} else if _, ok := mm.testDocContextMap[docID]; ok {
		mm.testDocContextMap[docID] = newDocContext
	}
	return mm
}

// Apply creates a new Module with the modifications.
// This triggers a cascade of modifications up the object hierarchy:
// Module modification -> Package update.
func (mm *ModuleModifier) Apply() *Module {
	// Create new moduleContext with the updated document contexts
	newModuleCtx := newModuleContextFromMaps(
		mm.project,
		mm.moduleID,
		mm.moduleDescriptor,
		mm.isDefaultModule,
		mm.srcDocIDs,
		mm.testSrcDocIDs,
		mm.srcDocContextMap,
		mm.testDocContextMap,
		mm.dependencies,
	)

	// Update the package with the new module context
	// This triggers package modification which may cascade to project
	newPackage := mm.packageInstance.Modify().updateModule(newModuleCtx).Apply()

	// Return the module from the new package
	return newPackage.Module(mm.moduleID)
}
