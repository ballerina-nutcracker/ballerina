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
	"context"
	"os"
	"path/filepath"
	"testing"

	"ballerina/projects"
	"ballerina/test_util"
)

// TestModuleResolver_ExternalPackage tests that the module resolver can identify external imports.
func TestModuleResolver_ExternalPackage(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Load a project
	projectPath := filepath.Join("testdata", "myproject")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	pkg := result.Project().CurrentPackage()
	require.NotNil(pkg)

	// Verify the package can access the cache (through resolution)
	resolution := pkg.Resolution()
	require.NotNil(resolution)

	// The module dependency graph should exist
	assert.NotNil(resolution.ModuleDependencyGraph())
}

// TestPackageResolution_PackageNotFound verifies that resolving a package
// which does not exist in any configured repository returns an unresolved
// response without surfacing an error. With Offline=true and no remote
// source wired in, the resolver simply runs out of repositories to ask.
func TestPackageResolution_PackageNotFound(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	cachePath, err := filepath.Abs("testdata/repo/bala")
	require.NoError(err)

	cache := projects.NewFileSystemRepository(os.DirFS(cachePath), ".")
	remote := projects.NewRemoteRepository(cache)
	env := projects.NewProjectEnvironmentBuilder(os.DirFS(cachePath)).
		WithRepositories([]projects.Repository{remote}).
		WithBuildOptions(projects.NewBuildOptionsBuilder().WithOffline(true).Build()).
		Build()

	version, err := projects.NewPackageVersionFromString("1.0.0")
	require.NoError(err)

	resp := env.PackageResolver().ResolvePackages(
		context.Background(),
		[]projects.ResolutionRequest{
			projects.NewResolutionRequest(projects.NewPackageDescriptor(
				projects.NewPackageOrg("missingorg"),
				projects.NewPackageName("missingpkg"),
				version,
			)),
		},
		env.ResolutionOptions(),
	)
	require.Len(resp, 1)
	assert.False(resp[0].IsResolved(),
		"expected unresolved response for cache-miss + offline through RemoteRepository")
}

// TestBalaProject_ModuleStructure tests that bala project modules are correctly loaded.
func TestBalaProject_ModuleStructure(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Load single module bala
	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "mockpkg", "1.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	pkg := result.Project().CurrentPackage()
	require.NotNil(pkg)

	// Check module count
	modules := pkg.Modules()
	assert.Len(modules, 1)

	// Check default module
	defaultModule := pkg.DefaultModule()
	require.NotNil(defaultModule)

	// Check documents
	docs := defaultModule.DocumentIDs()
	assert.True(len(docs) >= 1, "expected at least 1 document")
}

// TestBalaProject_MultiModule tests multi-module bala package loading.
func TestBalaProject_MultiModule(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Load multi-module bala
	balaPath := filepath.Join("testdata", "repo", "bala", "mockorg", "multimod", "2.0.0", "any")
	absPath, err := filepath.Abs(balaPath)
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	pkg := result.Project().CurrentPackage()
	require.NotNil(pkg)

	// Should have 2 modules (default + sub)
	modules := pkg.Modules()
	assert.Len(modules, 2)

	// Verify module names
	hasDefaultModule := false
	for _, m := range modules {
		if m.ModuleName().String() == "multimod" {
			hasDefaultModule = true
			break
		}
	}
	assert.True(hasDefaultModule, "expected to find module 'multimod'")
}

// TestRepository_Integration tests the repository with real cache structure.
func TestRepository_Integration(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	cachePath, err := filepath.Abs("testdata/repo/bala")
	require.NoError(err)

	repo := projects.NewFileSystemRepository(os.DirFS(cachePath), ".")

	// Test GetPackageVersions
	versions, err := repo.GetPackageVersions(context.Background(), "mockorg", "mockpkg", projects.ResolutionOptions{})
	require.NoError(err)
	hasVersion := false
	for _, v := range versions {
		if v.String() == "1.0.0" {
			hasVersion = true
			break
		}
	}
	assert.True(hasVersion, "expected versions to contain 1.0.0")

	// Test Exists
	exists, err := repo.Exists(context.Background(), "mockorg", "mockpkg", "1.0.0")
	require.NoError(err)
	assert.True(exists)

	// Test non-existent
	exists, err = repo.Exists(context.Background(), "mockorg", "mockpkg", "9.9.9")
	require.NoError(err)
	assert.False(exists)

	// Test GetPackage
	loadedPkg, err := repo.GetPackage(context.Background(), "mockorg", "mockpkg", "1.0.0", projects.ResolutionOptions{})
	require.NoError(err)
	require.NotNil(loadedPkg)
	assert.Equal(projects.ProjectKindBala, loadedPkg.Project().Kind())
}

// TestPackageResolution_ExternalDependencyCompilation tests package resolution with external dependencies
// and verifies that compilation completes successfully.
func TestPackageResolution_ExternalDependencyCompilation(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Step 1: Prepare the test repository path
	testRepoPath, err := filepath.Abs("testdata/repo/bala")
	require.NoError(err)

	// Step 2: Load the test project with the test repository configured upfront
	projectPath := filepath.Join("testdata", "project-with-external-dep")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath, projects.ProjectLoadConfig{
		Repositories: []projects.Repository{
			projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
		},
	})
	require.NoError(err)
	require.NotNil(result)

	mainProject := result.Project()
	require.NotNil(mainProject)
	env := mainProject.Environment()

	// Step 3: Get the package
	pkg := mainProject.CurrentPackage()
	require.NotNil(pkg)

	// Step 4: Trigger compilation (resolves dependencies internally)
	compilation := pkg.Compilation()
	require.NotNil(compilation, "compilation should not be nil")

	// Step 5: Verify external package was resolved and cached
	cachedPkg := env.PackageCache().Get("mockorg", "mockpkg", "1.0.0")
	require.NotNil(cachedPkg, "mockpkg should be cached after compilation")

	// Step 6: Verify dependency graph
	resolution := pkg.Resolution()
	require.NotNil(resolution)

	packageDependencyGraph := resolution.DependencyGraph()
	assert.NotNil(packageDependencyGraph, "package dependency graph should exist")

	resolvedDeps := resolution.ResolvedDependencies()
	mockpkgDesc, found := resolvedDeps["mockorg/mockpkg"]
	assert.True(found, "resolved dependencies should contain mockorg/mockpkg")
	assert.Equal("mockorg", mockpkgDesc.Org().Value())
	assert.Equal("mockpkg", mockpkgDesc.Name().Value())
	assert.Equal("1.0.0", mockpkgDesc.Version().String())

	// Step 7: Verify compilation completed without errors
	diagnosticResult := compilation.DiagnosticResult()
	assert.NotNil(diagnosticResult, "diagnostic result should exist")

	for _, diag := range diagnosticResult.Diagnostics() {
		t.Logf("Diagnostic: %s", diag.Message())
	}

	assert.Equal(0, diagnosticResult.DiagnosticCount(), "expected no compilation errors")
}

// TestPackageResolution_PicksLatestVersion verifies that when a project
// imports an external package by org+name (no version pin) and the cache
// holds multiple versions on disk, the resolver picks the latest version.
//
// Fixture: project-with-multi-version-dep imports mockorg/versionedpkg.
// testdata/repo/bala/mockorg/versionedpkg has both 1.0.0 and 2.0.0 — only
// 2.0.0 should be loaded into the resolver's cache.
func TestPackageResolution_PicksLatestVersion(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	testRepoPath, err := filepath.Abs("testdata/repo/bala")
	require.NoError(err)

	projectPath := filepath.Join("testdata", "project-with-multi-version-dep")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath, projects.ProjectLoadConfig{
		Repositories: []projects.Repository{
			projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
		},
	})
	require.NoError(err)

	pkg := result.Project().CurrentPackage()
	require.NotNil(pkg)

	// Trigger resolution.
	_ = pkg.Compilation()

	resolved := pkg.Resolution().ResolvedDependencies()["mockorg/versionedpkg"]
	require.NotNil(resolved, "expected mockorg/versionedpkg to be resolved")
	assert.Equal("2.0.0", resolved.Version().String(),
		"resolver should pick the latest version when no version is pinned")

	cache := result.Project().Environment().PackageCache()
	require.NotNil(cache.Get("mockorg", "versionedpkg", "2.0.0"),
		"latest (2.0.0) should be in the cache")
	if cache.Get("mockorg", "versionedpkg", "1.0.0") != nil {
		t.Error("older versions should not be loaded when the latest satisfies the import")
	}
}

// TestPackageResolution_TransitiveDependency tests package resolution with transitive dependencies.
// The dependency chain is: project -> middlepkg -> leafpkg
//
// This test verifies that when compiling a project, the compilation process internally
// resolves and compiles dependencies in the correct order, including transitive dependencies.
func TestPackageResolution_TransitiveDependency(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Step 1: Prepare the test repository path
	testRepoPath, err := filepath.Abs("testdata/repo/bala")
	require.NoError(err)

	// Step 2: Load the test project with the test repository configured upfront
	projectPath := filepath.Join("testdata", "project-with-transitive-dep")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath, projects.ProjectLoadConfig{
		Repositories: []projects.Repository{
			projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
		},
	})
	require.NoError(err)
	require.NotNil(result)

	mainProject := result.Project()
	require.NotNil(mainProject)
	env := mainProject.Environment()

	// Step 3: Get the package and verify initial state
	pkg := mainProject.CurrentPackage()
	require.NotNil(pkg)

	// Only the main package should be cached initially
	assert.Equal(1, env.PackageCache().Size(), "only main package should be cached initially")

	// Step 4: Trigger compilation and verify compilation completes successfully
	compilation := pkg.Compilation()
	require.NotNil(compilation, "compilation should not be nil")

	diagnosticResult := compilation.DiagnosticResult()
	assert.Equal(0, diagnosticResult.DiagnosticCount())

	for _, diag := range diagnosticResult.Diagnostics() {
		t.Logf("Diagnostic: %s", diag.Message())
	}

	// Step 5: Verify external packages were resolved and cached during compilation.
	// middlepkg declares aaaleafpkg and leafpkg as direct deps; with the main
	// project that's 4 packages, plus the always-compiled implicit lang libs
	// (lang.int, lang.boolean, lang.decimal, lang.error, lang.string, lang.value,
	// lang.xml, lang.float, lang.array, lang.map, lang.object, lang.runtime), giving 16 packages total in the cache.
	assert.Equal(16, env.PackageCache().Size(), "expected 16 packages in cache after compilation")

	cachedMiddle := env.PackageCache().Get("mockorg", "middlepkg", "1.0.0")
	require.NotNil(cachedMiddle, "middlepkg should be cached after compilation")

	cachedLeaf := env.PackageCache().Get("mockorg", "leafpkg", "1.0.0")
	require.NotNil(cachedLeaf, "leafpkg should be cached after compilation")

	cachedAaaLeaf := env.PackageCache().Get("mockorg", "aaaleafpkg", "1.0.0")
	require.NotNil(cachedAaaLeaf, "aaaleafpkg should be cached after compilation")

	// Step 6: Verify dependency graph shows both direct and transitive dependencies
	resolution := pkg.Resolution()
	require.NotNil(resolution)

	resolvedDeps := resolution.ResolvedDependencies()

	// Verify direct dependency (middlepkg)
	middlepkgDesc, found := resolvedDeps["mockorg/middlepkg"]
	assert.True(found, "resolved dependencies should contain mockorg/middlepkg")
	assert.Equal("mockorg", middlepkgDesc.Org().Value())
	assert.Equal("middlepkg", middlepkgDesc.Name().Value())
	assert.Equal("1.0.0", middlepkgDesc.Version().String())

	// Step 7: Verify transitive dependency (leafpkg) is also in main project's resolved deps
	leafpkgDesc, found := resolvedDeps["mockorg/leafpkg"]
	assert.True(found, "resolved dependencies should contain transitive dep mockorg/leafpkg")
	assert.Equal("mockorg", leafpkgDesc.Org().Value())
	assert.Equal("leafpkg", leafpkgDesc.Name().Value())
	assert.Equal("1.0.0", leafpkgDesc.Version().String())

	// Step 8: Verify package dependency graph has correct edges
	packageDependencyGraph := resolution.DependencyGraph()
	require.NotNil(packageDependencyGraph)

	// The graph should show: project -> middlepkg -> {aaaleafpkg, leafpkg}
	nodes := packageDependencyGraph.ToTopologicallySortedList()
	assert.Len(nodes, 4, "expected 4 nodes in package dependency graph (project, middlepkg, aaaleafpkg, leafpkg)")
}

// TestPackageResolution_SharedDependencyEdge verifies that when the root project imports
// both B and C directly, and B also depends on C, the B→C edge is recorded in the
// dependency graph. Without this edge, topological sort has no reason to place C before
// B, which would cause B to be compiled before its dependency C is available.
//
// Project structure:
//
//	root → middlepkg (direct)
//	root → leafpkg   (direct)
//	middlepkg → leafpkg (transitive, but already visited)
func TestPackageResolution_SharedDependencyEdge(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	testRepoPath, err := filepath.Abs("testdata/repo/bala")
	require.NoError(err)

	absPath, err := filepath.Abs(filepath.Join("testdata", "project-with-shared-dep"))
	require.NoError(err)

	result, err := loadProject(absPath, projects.ProjectLoadConfig{
		Repositories: []projects.Repository{
			projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
		},
	})
	require.NoError(err)
	require.NotNil(result)

	pkg := result.Project().CurrentPackage()
	require.NotNil(pkg)

	compilation := pkg.Compilation()
	require.NotNil(compilation)
	assert.Equal(0, compilation.DiagnosticResult().DiagnosticCount())

	resolution := pkg.Resolution()
	require.NotNil(resolution)

	graph := resolution.DependencyGraph()
	require.NotNil(graph)

	middleDesc, err := projects.NewPackageDescriptorFromStrings("mockorg", "middlepkg", "1.0.0")
	require.NoError(err)
	leafDesc, err := projects.NewPackageDescriptorFromStrings("mockorg", "leafpkg", "1.0.0")
	require.NoError(err)

	// middlepkg must declare leafpkg as a direct dependency in the graph,
	// even though leafpkg is already a direct dependency of the root.
	middleDeps := graph.DirectDependencies(middleDesc)
	found := false
	for _, d := range middleDeps {
		if d.Equals(leafDesc) {
			found = true
			break
		}
	}
	assert.True(found, "dependency graph must contain the middlepkg→leafpkg edge")

	// leafpkg must appear before middlepkg in topological order.
	sorted := graph.ToTopologicallySortedList()
	leafIdx, middleIdx := -1, -1
	for i, d := range sorted {
		if d.Equals(leafDesc) {
			leafIdx = i
		}
		if d.Equals(middleDesc) {
			middleIdx = i
		}
	}
	assert.True(leafIdx != -1, "leafpkg must be present in the topologically sorted list")
	assert.True(middleIdx != -1, "middlepkg must be present in the topologically sorted list")
	if leafIdx != -1 && middleIdx != -1 {
		assert.True(leafIdx < middleIdx, "leafpkg must be ordered before middlepkg in topological sort")
	}
}

// TestPackageResolution_MultiModuleDependencies tests package resolution with multi-module
// dependencies at both direct and transitive levels.
// Structure:
//   - project imports mockorg/multiA (default module) and mockorg/multiA.util (submodule)
//   - multiA depends on mockorg/multiB (which has multiB and multiB.helper modules)
//
// This verifies that:
//  1. Multi-module packages are correctly resolved as single package dependencies
//  2. Importing different modules from the same package doesn't create duplicate entries
//  3. Transitive multi-module dependencies are correctly resolved
func TestPackageResolution_MultiModuleDependencies(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	// Step 1: Prepare the test repository path
	testRepoPath, err := filepath.Abs("testdata/repo/bala")
	require.NoError(err)

	// Step 2: Load the test project with the test repository
	projectPath := filepath.Join("testdata", "project-with-multimod-dep")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	result, err := loadProject(absPath, projects.ProjectLoadConfig{
		Repositories: []projects.Repository{
			projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
		},
	})
	require.NoError(err)
	require.NotNil(result)

	mainProject := result.Project()
	pkg := mainProject.CurrentPackage()

	// Step 3: Verify resolved dependencies
	resolution := pkg.Resolution()
	packageDependencyGraph := resolution.DependencyGraph()
	nodes := packageDependencyGraph.ToTopologicallySortedList()
	assert.Len(nodes, 3, "expected 3 nodes: project, multiA, multiB")

	// Step 4: Trigger compilation
	compilation := pkg.Compilation()
	require.NotNil(compilation)

	diagnosticResult := compilation.DiagnosticResult()
	for _, diag := range diagnosticResult.Diagnostics() {
		t.Logf("Diagnostic: %s", diag.Message())
	}
	assert.Equal(0, diagnosticResult.DiagnosticCount(), "expected no compilation errors")
}

// TestPackageResolution_ProjectLevelCache verifies the build-project default:
// when BallerinaEnvFs is not set, the loader falls back to
// fs.Sub(projectFs, ".ballerina"), which resolves to <project-path>/.ballerina/.
// External packages staged there must be picked up without any caller config.
func TestPackageResolution_ProjectLevelCache(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	projectPath := filepath.Join("testdata", "project-with-local-cache")
	absPath, err := filepath.Abs(projectPath)
	require.NoError(err)

	fsys := os.DirFS(absPath)

	// Intentionally do not pass BallerinaEnvFs — exercises the default fallback.
	result, err := projects.Load(fsys, ".")
	require.NoError(err)
	require.NotNil(result)

	mainProject := result.Project()
	require.NotNil(mainProject)

	// Get the package and trigger compilation
	pkg := mainProject.CurrentPackage()
	require.NotNil(pkg)

	compilation := pkg.Compilation()
	require.NotNil(compilation, "compilation should not be nil")

	// Verify external package was resolved from local cache
	env := mainProject.Environment()
	cachedPkg := env.PackageCache().Get("mockorg", "mockpkg", "1.0.0")
	require.NotNil(cachedPkg, "mockpkg should be resolved from .ballerina/ cache")

	// Verify no compilation errors
	diagnostics := compilation.DiagnosticResult()
	for _, diag := range diagnostics.Diagnostics() {
		t.Logf("Diagnostic: %s", diag.Message())
	}
	assert.Equal(0, diagnostics.DiagnosticCount(), "expected no compilation errors")
}

// TestPackageResolution_SingleFileProjectLevelCache verifies the single-file
// default: when BallerinaEnvFs is not set, the loader falls back to
// fs.Sub(projectFs, ".ballerina"), which resolves to <file-path-parent>/.ballerina/
// for a single .bal file. External packages staged there must be picked up
// without any caller config.
func TestPackageResolution_SingleFileProjectLevelCache(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	balFile := filepath.Join("testdata", "single-file-with-local-cache", "main.bal")
	absPath, err := filepath.Abs(balFile)
	require.NoError(err)

	baseDir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)
	fsys := os.DirFS(baseDir)

	// Intentionally do not pass BallerinaEnvFs — exercises the default fallback.
	result, err := projects.Load(fsys, fileName)
	require.NoError(err)
	require.NotNil(result)

	mainProject := result.Project()
	require.NotNil(mainProject)

	pkg := mainProject.CurrentPackage()
	require.NotNil(pkg)

	compilation := pkg.Compilation()
	require.NotNil(compilation, "compilation should not be nil")

	// Verify external package was resolved from the parent-directory local cache.
	env := mainProject.Environment()
	cachedPkg := env.PackageCache().Get("mockorg", "mockpkg", "1.0.0")
	require.NotNil(cachedPkg, "mockpkg should be resolved from <file-path-parent>/.ballerina/ cache")

	// Verify no compilation errors
	diagnostics := compilation.DiagnosticResult()
	for _, diag := range diagnostics.Diagnostics() {
		t.Logf("Diagnostic: %s", diag.Message())
	}
	assert.Equal(0, diagnostics.DiagnosticCount(), "expected no compilation errors")
}
