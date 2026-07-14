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
	"io/fs"
	"path"
	"path/filepath"
)

// BuildProject represents a Ballerina build project (project with Ballerina.toml).
type BuildProject struct {
	BaseProject
}

// Compile-time check to verify BuildProject implements Project interface
var _ Project = (*BuildProject)(nil)

// newBuildProject creates a new BuildProject with the given source root and build options.
func newBuildProject(fsys fs.FS, sourceRoot string, buildOptions BuildOptions) *BuildProject {
	project := &BuildProject{}
	project.initBase(fsys, sourceRoot, buildOptions)
	return project
}

// newBuildProjectWithEnv creates a new BuildProject with a pre-configured Environment.
// Use this when the Environment has been configured with repositories upfront.
func newBuildProjectWithEnv(sourceRoot string, buildOptions BuildOptions, env *Environment) *BuildProject {
	project := &BuildProject{}
	project.initBaseWithEnv(sourceRoot, buildOptions, env)
	return project
}

// Kind returns the project kind (BUILD).
func (b *BuildProject) Kind() ProjectKind {
	return ProjectKindBuild
}

// TargetDir returns the target directory for build outputs.
// If BuildOptions specifies a target directory, that is used; otherwise sourceRoot/target.
func (b *BuildProject) TargetDir() string {
	if targetDir := b.buildOptions.TargetDir(); targetDir != "" {
		return targetDir
	}
	return filepath.Join(b.sourceRoot, TargetDir)
}

// DocumentID returns the DocumentID for the given file path, if it exists in this project.
// It searches through all modules in the current package.
//
// The filePath argument may use either forward-slash or OS-native separators —
// it is normalized to the canonical forward-slash form before comparison so the
// lookup behaves consistently on Windows.
func (b *BuildProject) DocumentID(filePath string) (DocumentID, bool) {
	if b.CurrentPackage() == nil {
		return DocumentID{}, false
	}

	target := filepath.ToSlash(filePath)
	targetBase := path.Base(target)

	for _, module := range b.CurrentPackage().Modules() {
		for _, docID := range module.DocumentIDs() {
			doc := module.Document(docID)
			if doc != nil && path.Base(filepath.ToSlash(doc.Name())) == targetBase {
				if b.documentPathForModule(docID, module) == target {
					return docID, true
				}
			}
		}

		for _, docID := range module.TestDocumentIDs() {
			doc := module.Document(docID)
			if doc != nil && path.Base(filepath.ToSlash(doc.Name())) == targetBase {
				if b.documentPathForModule(docID, module) == target {
					return docID, true
				}
			}
		}
	}

	return DocumentID{}, false
}

// documentPathForModule computes the file path for a document in a module.
// Paths are returned in forward-slash form to match Document.Name() and the
// fs.FS convention, so callers see consistent paths across operating systems.
func (b *BuildProject) documentPathForModule(docID DocumentID, module *Module) string {
	doc := module.Document(docID)
	if doc == nil {
		return ""
	}

	// Document.Name() may be path-joined relative to the project root
	// (e.g., "modules/util/foo.bal" or "pkg-a/main.bal" for workspace
	// members). Extract the bare basename for path construction.
	docName := path.Base(filepath.ToSlash(doc.Name()))
	root := filepath.ToSlash(b.sourceRoot)

	if module.IsDefaultModule() {
		// Default module: files are in sourceRoot or sourceRoot/tests
		for _, testID := range module.TestDocumentIDs() {
			if testID.Equals(docID) {
				return path.Join(root, TestsDir, docName)
			}
		}
		return path.Join(root, docName)
	}

	// Named module: files are in sourceRoot/modules/<moduleName>
	moduleName := module.ModuleName().ModuleNamePart()
	modulePath := path.Join(root, ModulesDir, moduleName)

	for _, testID := range module.TestDocumentIDs() {
		if testID.Equals(docID) {
			return path.Join(modulePath, TestsDir, docName)
		}
	}
	return path.Join(modulePath, docName)
}

// DocumentPath returns the file path for the given DocumentID.
func (b *BuildProject) DocumentPath(documentID DocumentID) string {
	if b.CurrentPackage() == nil {
		return ""
	}

	// Find the module containing this document
	moduleID := documentID.ModuleID()
	module := b.CurrentPackage().Module(moduleID)
	if module == nil {
		return ""
	}

	return b.documentPathForModule(documentID, module)
}

// Save persists project changes to the filesystem.
// Currently a stub that returns nil.
func (b *BuildProject) Save() {
	// TODO: Implement actual save functionality
}

// Duplicate creates a deep copy of the build project.
// The duplicated project shares immutable state (IDs, descriptors, configs)
// but has independent compilation caches and lazy-loaded fields.
func (b *BuildProject) Duplicate() Project {
	// Create duplicate build options using AcceptTheirs pattern
	duplicateBuildOptions := NewBuildOptions().AcceptTheirs(b.buildOptions)
	// Create new environment with fresh caches but same repository config
	newProject := newBuildProjectWithEnv(b.sourceRoot, duplicateBuildOptions, b.Environment().Duplicate())
	ResetPackage(b, newProject)

	return newProject
}

func (b *BuildProject) Environment() *Environment {
	return b.environment
}
