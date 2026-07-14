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
	"path/filepath"
)

// BalaProject represents a Ballerina bala package loaded from the cache.
// This is used for loading dependency packages.
type BalaProject struct {
	BaseProject
	platform      string // e.g., "any"
	schemaVersion int    // bala schema version; < 4 means legacy (v3) module layout
}

// Compile-time check to verify BalaProject implements Project interface
var _ Project = (*BalaProject)(nil)

// newBalaProjectWithEnv creates a new BalaProject with a shared Environment.
// Use this when loading dependency packages that need to share the same
// PackageCache as the root project.
func newBalaProjectWithEnv(sourceRoot string, buildOptions BuildOptions, platform string, schemaVersion int, env *Environment) *BalaProject {
	project := &BalaProject{
		platform:      platform,
		schemaVersion: schemaVersion,
	}
	project.initBaseWithEnv(sourceRoot, buildOptions, env)
	return project
}

// Kind returns the project kind (BALA).
func (b *BalaProject) Kind() ProjectKind {
	return ProjectKindBala
}

// Platform returns the platform identifier (e.g., "any").
func (b *BalaProject) Platform() string {
	return b.platform
}

// TargetDir returns an empty string for bala projects (no build outputs).
func (b *BalaProject) TargetDir() string {
	return ""
}

// DocumentID returns the DocumentID for the given file path, if it exists in this project.
func (b *BalaProject) DocumentID(filePath string) (DocumentID, bool) {
	if b.CurrentPackage() == nil {
		return DocumentID{}, false
	}

	// Search through all modules
	for _, module := range b.CurrentPackage().Modules() {
		for _, docID := range module.DocumentIDs() {
			doc := module.Document(docID)
			if doc != nil && doc.Name() == filepath.Base(filePath) {
				// Validate full path to handle same-named files in different modules
				docPath := b.documentPathForModule(docID, module)
				if docPath == filePath {
					return docID, true
				}
			}
		}
	}

	return DocumentID{}, false
}

// documentPathForModule returns the file path for a document within a module.
//
// v4+ layout: default-module files sit at the bala root; non-default modules
// are under modules/<moduleNamePart>/.
//
// Legacy (schema < 4) layout: every module lives under modules/<moduleName>/,
// where moduleName is the full module name string (e.g. "mypkg" for the default
// module, "mypkg.sub" for a sub-module), matching what scanBalaModules scanned.
func (b *BalaProject) documentPathForModule(docID DocumentID, module *Module) string {
	doc := module.Document(docID)
	if doc == nil {
		return ""
	}
	docName := filepath.Base(doc.Name())
	if b.schemaVersion < 4 {
		return filepath.Join(b.sourceRoot, ModulesDir, module.ModuleName().String(), docName)
	}
	if module.IsDefaultModule() {
		return filepath.Join(b.sourceRoot, docName)
	}
	return filepath.Join(b.sourceRoot, ModulesDir, module.ModuleName().ModuleNamePart(), docName)
}

// DocumentPath returns the file path for the given DocumentID.
func (b *BalaProject) DocumentPath(documentID DocumentID) string {
	if b.CurrentPackage() == nil {
		return ""
	}

	moduleID := documentID.ModuleID()
	module := b.CurrentPackage().Module(moduleID)
	if module == nil {
		return ""
	}

	return b.documentPathForModule(documentID, module)
}

// Save is a no-op for bala projects (read-only).
func (b *BalaProject) Save() {
	// Bala projects are read-only
}

// Duplicate creates a deep copy of the bala project.
func (b *BalaProject) Duplicate() Project {
	duplicateBuildOptions := NewBuildOptions().AcceptTheirs(b.buildOptions)
	newProject := newBalaProjectWithEnv(b.sourceRoot, duplicateBuildOptions, b.platform, b.schemaVersion, b.Environment().Duplicate())
	ResetPackage(b, newProject)
	return newProject
}

func (b *BalaProject) Environment() *Environment {
	return b.environment
}
