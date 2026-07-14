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

package projects_test

import (
	"path/filepath"
	"strings"
	"testing"

	"ballerina-lang-go/projects"
	"ballerina-lang-go/test_util"
)

func TestBalaProject_LoadSingleModule(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Load the mock bala package
	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "mockpkg", "1.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	project := result.Project()
	require.NotNil(project)

	// Verify project type
	assert.Equal(projects.ProjectKindBala, project.Kind())

	// Verify package info
	pkg := project.CurrentPackage()
	require.NotNil(pkg)

	assert.Equal("mockorg", pkg.PackageOrg().Value())
	assert.Equal("mockpkg", pkg.PackageName().Value())
	assert.Equal("1.0.0", pkg.PackageVersion().String())

	// Verify modules
	modules := pkg.Modules()
	assert.Len(modules, 1)

	defaultModule := pkg.DefaultModule()
	require.NotNil(defaultModule)
	assert.True(defaultModule.IsDefaultModule())

	// Verify documents
	docIDs := defaultModule.DocumentIDs()
	assert.Len(docIDs, 1)

	doc := defaultModule.Document(docIDs[0])
	require.NotNil(doc)
	assert.Equal("lib.bal", doc.Name())
	assert.True(strings.Contains(doc.TextDocument().String(), "public function greet"))
}

func TestBalaProject_LoadMultiModule(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Load the mock multi-module bala package
	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "multimod", "2.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	project := result.Project()
	require.NotNil(project)

	// Verify project type
	assert.Equal(projects.ProjectKindBala, project.Kind())

	// Verify package info
	pkg := project.CurrentPackage()
	require.NotNil(pkg)

	assert.Equal("mockorg", pkg.PackageOrg().Value())
	assert.Equal("multimod", pkg.PackageName().Value())
	assert.Equal("2.0.0", pkg.PackageVersion().String())

	// Verify modules (should have default + sub module)
	modules := pkg.Modules()
	assert.Len(modules, 2)

	// Find the sub module
	var subModule *projects.Module
	for _, m := range modules {
		if !m.IsDefaultModule() {
			subModule = m
			break
		}
	}
	require.NotNil(subModule)
	assert.Equal("sub", subModule.ModuleName().ModuleNamePart())
}

// TestBalaProject_DocNameIsBare verifies that Document.Name() returns the
// bare filename for files loaded from a bala package. The package-qualified
// key is an internal DiagnosticEnv concern and does not leak into the public
// document name.
func TestBalaProject_DocNameIsBare(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "multimod", "2.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	pkg := result.Project().CurrentPackage()
	require.NotNil(pkg)

	for _, mod := range pkg.Modules() {
		docIDs := mod.DocumentIDs()
		require.Len(docIDs, 1)
		doc := mod.Document(docIDs[0])
		require.NotNil(doc)

		var expected string
		if mod.IsDefaultModule() {
			expected = "main.bal"
		} else {
			expected = "sub.bal"
		}
		assert.Equal(expected, doc.Name())
	}
}

func TestBalaProject_Platform(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "mockpkg", "1.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	// Cast to BalaProject to access Platform()
	balaProject, ok := result.Project().(*projects.BalaProject)
	assert.True(ok)

	assert.Equal("any", balaProject.Platform())
}

func TestBalaProject_Duplicate(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "mockpkg", "1.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	original := result.Project()
	duplicate := original.Duplicate()

	// Verify they are different instances
	assert.NotSame(original, duplicate)

	// Verify same kind
	assert.Equal(original.Kind(), duplicate.Kind())

	// Verify same package info
	assert.Equal(
		original.CurrentPackage().PackageName().Value(),
		duplicate.CurrentPackage().PackageName().Value(),
	)
}

func TestBalaProject_TargetDir(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "mockpkg", "1.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	// Bala projects have no target directory
	assert.True(result.Project().TargetDir() == "")
}
