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

// Package projects_test contains integration tests for the Project API.
// These tests verify the complete project loading and compilation pipeline
// using real Ballerina source files.
package projects_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"ballerina-lang-go/projects"
	"ballerina-lang-go/test_util"
)

// TestLoadSingleFile tests loading a valid standalone Ballerina file.
func TestLoadSingleFile(t *testing.T) {
	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	projectPath := filepath.Join("testdata", "single-file", "main.bal")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	project := result.Project().(*projects.SingleFileProject)

	// Load the package
	currentPackage := project.CurrentPackage()
	require.NotNil(currentPackage)

	// Load the default module
	defaultModule := currentPackage.DefaultModule()
	require.NotNil(defaultModule)

	// Verify module has exactly one document
	docIDs := defaultModule.DocumentIDs()
	assert.Len(docIDs, 1)

	// Verify package has exactly one module
	moduleIDs := currentPackage.ModuleIDs()
	assert.Len(moduleIDs, 1)

	// Verify the module ID matches the default module
	assert.Equal(defaultModule.ModuleID(), moduleIDs[0])
}

// TestSingleFileTargetDirectory tests if the target directory for single files is resolved properly.
func TestSingleFileTargetDirectory(t *testing.T) {
	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	projectPath := filepath.Join("testdata", "single-file", "main.bal")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	// Without explicit BuildOptions.TargetDir(), target directory is empty.
	// Callers (CLI, playground) are responsible for providing a target directory.
	result, err := loadProject(absPath)
	require.NoError(err)

	project := result.Project().(*projects.SingleFileProject)
	assert.Equal("", project.TargetDir())
}

// TestSingleFileTargetDirectoryWithBuildOptions tests that BuildOptions.TargetDir() is respected.
func TestSingleFileTargetDirectoryWithBuildOptions(t *testing.T) {
	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	projectPath := filepath.Join("testdata", "single-file", "main.bal")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	// Create a temp directory and inject via BuildOptions
	tempDir := t.TempDir()
	buildOpts := projects.NewBuildOptionsBuilder().
		WithTargetDir(tempDir).
		Build()

	baseDir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)
	fsys := os.DirFS(baseDir)

	ballerinaEnvPath, err := getBallerinaEnvPath()
	require.NoError(err)
	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)

	result, err := projects.Load(fsys, fileName, projects.ProjectLoadConfig{
		BuildOptions:   &buildOpts,
		BallerinaEnvFs: ballerinaEnvFs,
	})
	require.NoError(err)

	project := result.Project().(*projects.SingleFileProject)
	assert.Equal(tempDir, project.TargetDir())
}

// TestDefaultBuildOptions tests the default build options for single file projects.
func TestDefaultBuildOptions(t *testing.T) {
	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	projectPath := filepath.Join("testdata", "single-file", "main.bal")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	project := result.Project().(*projects.SingleFileProject)
	buildOpts := project.BuildOptions()

	// Verify expected default buildOptions. OfflineBuild is true because the
	// test helper pins it; everything else reflects the manifest defaults.
	assert.True(buildOpts.SkipTests())
	assert.False(buildOpts.ObservabilityIncluded())
	assert.False(buildOpts.CodeCoverage())
	assert.True(buildOpts.OfflineBuild())
	assert.False(buildOpts.Experimental())
	assert.False(buildOpts.TestReport())
}

// TestOverrideBuildOptions tests overriding build options for single file projects.
func TestOverrideBuildOptions(t *testing.T) {
	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	projectPath := filepath.Join("testdata", "single-file", "main.bal")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	// Create build options with specific values
	buildOptions := projects.NewBuildOptionsBuilder().
		WithSkipTests(true).
		WithObservabilityIncluded(true).
		Build()

	result, err := loadProject(absPath, projects.ProjectLoadConfig{
		BuildOptions: &buildOptions,
	})
	require.NoError(err)

	project := result.Project().(*projects.SingleFileProject)
	buildOpts := project.BuildOptions()

	// Verify expected overridden buildOptions
	assert.True(buildOpts.SkipTests())
	assert.True(buildOpts.ObservabilityIncluded())
	assert.False(buildOpts.CodeCoverage())
	assert.False(buildOpts.Experimental())
	assert.False(buildOpts.TestReport())
}

// TestUpdateSingleFile tests updating document content in a single file project.
func TestUpdateSingleFile(t *testing.T) {
	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	filePath := filepath.Join("testdata", "single-file", "main.bal")
	absPath, err := filepath.Abs(filePath)
	require.NoError(err)

	newContent := "import ballerina/io;\n"

	// Load the project
	result, err := loadProject(absPath)
	require.NoError(err)

	singleFileProject := result.Project().(*projects.SingleFileProject)

	// Get the document ID
	oldModule := singleFileProject.CurrentPackage().DefaultModule()
	require.NotNil(oldModule)

	docIDs := oldModule.DocumentIDs()
	require.NotEmpty(docIDs)

	oldDocumentID := docIDs[0]
	oldDocument := oldModule.Document(oldDocumentID)
	require.NotNil(oldDocument)

	// Update the document
	updatedDoc := oldDocument.Modify().WithContent(newContent).Apply()

	// Verify document counts remain the same
	assert.Equal(len(oldDocument.Module().DocumentIDs()), len(updatedDoc.Module().DocumentIDs()))
	assert.Equal(len(oldDocument.Module().TestDocumentIDs()), len(updatedDoc.Module().TestDocumentIDs()))

	// Verify all document IDs are preserved
	updatedDocIDs := updatedDoc.Module().DocumentIDs()
	for _, docID := range oldDocument.Module().DocumentIDs() {
		found := slices.ContainsFunc(updatedDocIDs, docID.Equals)
		assert.True(found)
	}

	// Verify documents are different objects
	assert.NotSame(oldDocument, updatedDoc)

	// Verify old document content is unchanged
	oldContent := oldDocument.TextDocument().String()
	assert.NotEqual(newContent, oldContent)

	// Verify new document has updated content
	updatedContent := updatedDoc.TextDocument().String()
	assert.Equal(newContent, updatedContent)

	// Verify the package was updated
	updatedPackage := singleFileProject.CurrentPackage()
	assert.NotSame(oldModule.PackageInstance(), updatedPackage)

	// Verify the updated document is accessible from the updated package
	updatedModuleFromPkg := updatedPackage.Module(oldModule.ModuleID())
	require.NotNil(updatedModuleFromPkg)

	docFromUpdatedPkg := updatedModuleFromPkg.Document(oldDocumentID)
	require.NotNil(docFromUpdatedPkg)

	assert.Same(updatedDoc, docFromUpdatedPkg)
	assert.Same(updatedPackage, updatedDoc.Module().PackageInstance())
}

// TestProjectDuplicate tests duplicating a single file project.
//
// Skipped: Duplicate() creates a fresh documentContext whose lazy-loaded
// text.TextDocument is a new pointer with the same content. Compiling the
// duplicated project re-registers the same fileName under a different doc
// pointer, which trips DiagnosticEnv.RegisterFile's same-name-different-doc
// panic. Re-enable once DiagnosticEnv supports per-instance file identity
// (e.g., keyed on PackageID UUID).
func TestProjectDuplicate(t *testing.T) {
	t.Skip("DiagnosticEnv panics on Duplicate(): same fileName, different doc pointer")

	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	projectPath := filepath.Join("testdata", "single-file", "main.bal")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	originalProject := result.Project().(*projects.SingleFileProject)
	originalPackage := originalProject.CurrentPackage()

	// Duplicate the project
	duplicatedProject := originalProject.Duplicate()

	// Verify different project instances
	assert.NotSame(originalProject, duplicatedProject)

	// Verify same project kind
	assert.Equal(originalProject.Kind(), duplicatedProject.Kind())

	// Verify same source root
	assert.Equal(originalProject.SourceRoot(), duplicatedProject.SourceRoot())

	duplicatedPackage := duplicatedProject.CurrentPackage()

	// Verify different package instances
	assert.NotSame(originalPackage, duplicatedPackage)

	// Verify same packageId
	assert.Equal(originalPackage.PackageID(), duplicatedPackage.PackageID())

	// Verify same package name
	assert.Equal(originalPackage.PackageName().Value(), duplicatedPackage.PackageName().Value())

	originalModule := originalPackage.DefaultModule()
	duplicatedModule := duplicatedPackage.DefaultModule()

	// Verify different module instances
	assert.NotSame(originalModule, duplicatedModule)

	// Verify same moduleId
	assert.Equal(originalModule.ModuleID(), duplicatedModule.ModuleID())

	// Verify same number of documents
	originalDocIDs := originalModule.DocumentIDs()
	duplicatedDocIDs := duplicatedModule.DocumentIDs()
	assert.Equal(len(originalDocIDs), len(duplicatedDocIDs))

	// Verify document instances are different but have same content
	if len(originalDocIDs) > 0 && len(duplicatedDocIDs) > 0 {
		originalDoc := originalModule.Document(originalDocIDs[0])
		duplicatedDoc := duplicatedModule.Document(duplicatedDocIDs[0])

		assert.NotSame(originalDoc, duplicatedDoc)

		originalContent := originalDoc.TextDocument().String()
		duplicatedContent := duplicatedDoc.TextDocument().String()
		assert.Equal(originalContent, duplicatedContent)
	}

	// Verify both projects can compile independently
	originalCompilation := originalPackage.Compilation()
	assert.NotNil(originalCompilation)

	duplicatedCompilation := duplicatedPackage.Compilation()
	assert.NotNil(duplicatedCompilation)

	// Verify compilations are different instances
	assert.NotSame(originalCompilation, duplicatedCompilation)
}

// TestSingleFile_CompilationDiagnosticUnresolvedImport verifies the
// main-with-error.bal fixture (which imports an unknown module) reports
// compilation diagnostics.
func TestSingleFile_CompilationDiagnosticUnresolvedImport(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	absPath, err := filepath.Abs(filepath.Join("testdata", "single-file", "main-with-error.bal"))
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	pkg := result.Project().CurrentPackage()
	require.NotNil(pkg)

	diags := pkg.Compilation().DiagnosticResult()
	assert.True(diags.HasErrors(),
		"expected compilation errors for unresolved import, got: %v",
		diagnosticMessages(diags))
}
