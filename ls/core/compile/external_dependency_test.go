// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package compile

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

const externalDependencySource = "import acme/dep;\n\npublic function main() {\n    dep:Public value = {};\n}\n"

func TestPackageDriver_UnchangedExternalDependencyCarriesForward(t *testing.T) {
	_, projectService, packageOne, documentURI := openExternalDependencyFixture(t)
	generationOne := newPackageDriver(packageOne, projectService.OpenText, nil, nil)
	generationOne.advanceAll(stageTopLevelTypeResolved)
	committed := generationOne.snapshotForCommit()
	externalOne := externalSnapshots(t, packageOne, committed)

	applyOpen(t, projectService, documentURI, externalDependencySource+"\n")
	project, err := projectService.Project(documentURI)
	if err != nil || project == nil {
		t.Fatalf("Project: %v", err)
	}
	packageTwo := project.CurrentPackage()
	generationTwo := newPackageDriver(packageTwo, projectService.OpenText, &packageGenState{modules: committed}, nil)
	generationTwo.advanceAll(stageTopLevelTypeResolved)
	externalTwo := externalSnapshots(t, packageTwo, generationTwo.snapshotForCommit())

	for id, snapshotOne := range externalOne {
		snapshotTwo, ok := externalTwo[id]
		if !ok {
			t.Fatalf("external module %v missing from generation two", id)
		}
		if snapshotTwo.driver != snapshotOne.driver {
			t.Errorf("external module %v was rebuilt despite unchanged content", id)
		}
	}
	if len(generationTwo.externalProjections) == 0 {
		t.Fatal("unchanged external dependency must still supply a projection")
	}
}

func TestPackageDriver_EditedExternalDependencyRecompiles(t *testing.T) {
	_, projectService, packageOne, documentURI := openExternalDependencyFixture(t)
	generationOne := newPackageDriver(packageOne, projectService.OpenText, nil, nil)
	generationOne.advanceAll(stageTopLevelTypeResolved)
	committed := generationOne.snapshotForCommit()
	externalOne := externalSnapshots(t, packageOne, committed)

	dependencyPackage := resolvedExternalDependency(t, packageOne)
	dependencyModule := dependencyPackage.ModuleByName(mustModuleName(t, dependencyPackage, "dep"))
	dependencyDocument := dependencyModule.Document(dependencyModule.DocumentIDs()[0])
	dependencyDocument.Modify().WithContent("public type Public record { string changed; };\n").Apply()
	applyOpen(t, projectService, documentURI, externalDependencySource+"\n")
	project, err := projectService.Project(documentURI)
	if err != nil || project == nil {
		t.Fatalf("Project: %v", err)
	}
	packageTwo := project.CurrentPackage()
	generationTwo := newPackageDriver(packageTwo, projectService.OpenText, &packageGenState{modules: committed}, nil)
	generationTwo.advanceAll(stageTopLevelTypeResolved)
	externalTwo := externalSnapshots(t, packageTwo, generationTwo.snapshotForCommit())

	for id, snapshotOne := range externalOne {
		snapshotTwo, ok := externalTwo[id]
		if !ok {
			t.Fatalf("external module %v missing from generation two", id)
		}
		if snapshotTwo.driver == snapshotOne.driver {
			t.Errorf("external module %v was carried forward after its content changed", id)
		}
	}
}

func openExternalDependencyFixture(t *testing.T) (pal.Platform, *workspace.ProjectService, *projects.Package, workspace.DocumentURI) {
	t.Helper()
	platform, cleanup := palnative.NewPlatform()
	t.Cleanup(cleanup)
	root := t.TempDir()
	writeExternalDependencyFixture(t, platform, root)

	bus := event.New()
	t.Cleanup(bus.Close)
	repository := workspace.NewFileSystemRepository(platform, filepath.Join(root, "repository"))
	projectService := workspace.New(platform, bus, workspace.WithRepositories([]projects.Repository{repository}))
	documentPath := filepath.Join(root, "main.bal")
	documentURI := fileURI(t, "file://"+documentPath)
	applyOpen(t, projectService, documentURI, externalDependencySource)
	project, err := projectService.Project(documentURI)
	if err != nil || project == nil {
		t.Fatalf("Project: %v", err)
	}
	packageNode := project.CurrentPackage()
	if packageNode == nil {
		t.Fatal("CurrentPackage() = nil")
	}
	return platform, projectService, packageNode, documentURI
}

func writeExternalDependencyFixture(t *testing.T, platform pal.Platform, root string) {
	t.Helper()
	files := map[string]string{
		"Ballerina.toml": "[package]\norg = \"test\"\nname = \"main\"\nversion = \"0.1.0\"\n",
		"main.bal":       externalDependencySource,
		"repository/acme/dep/0.1.0/go1.27/bala.json":             "{\"bala_version\":\"3.0.0\"}\n",
		"repository/acme/dep/0.1.0/go1.27/package.json":          "{\"organization\":\"acme\",\"name\":\"dep\",\"version\":\"0.1.0\",\"platform\":\"go1.27\",\"modules\":[]}\n",
		"repository/acme/dep/0.1.0/go1.27/dependency-graph.json": "{\"packages\":[{\"org\":\"acme\",\"name\":\"dep\",\"version\":\"0.1.0\",\"dependencies\":[],\"modules\":[]}],\"modules\":[{\"org\":\"acme\",\"package_name\":\"dep\",\"version\":\"0.1.0\",\"module_name\":\"dep\",\"dependencies\":[]}]}\n",
		"repository/acme/dep/0.1.0/go1.27/modules/dep/dep.bal":   "public type Public record {};\n",
	}
	for name, content := range files {
		fileName := filepath.Join(root, name)
		if err := platform.FS.MkdirAll(filepath.Dir(fileName), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(fileName), err)
		}
		if err := platform.FS.WriteFile(fileName, []byte(content)); err != nil {
			t.Fatalf("WriteFile(%s): %v", fileName, err)
		}
	}
}

func resolvedExternalDependency(t *testing.T, pkg *projects.Package) *projects.Package {
	t.Helper()
	for _, descriptor := range pkg.Resolution().ResolvedDependencies() {
		responses := pkg.Project().Environment().PackageResolver().ResolvePackages(
			context.Background(), []projects.ResolutionRequest{projects.NewResolutionRequest(*descriptor)}, pkg.Project().Environment().ResolutionOptions(),
		)
		if len(responses) == 1 && responses[0].IsResolved() && responses[0].Package() != nil {
			return responses[0].Package()
		}
	}
	t.Fatal("external dependency did not resolve")
	return nil
}

func externalSnapshots(t *testing.T, pkg *projects.Package, snapshots map[projects.ModuleID]moduleGenSnapshot) map[projects.ModuleID]moduleGenSnapshot {
	t.Helper()
	local := make(map[projects.ModuleID]bool)
	for _, id := range pkg.ModuleIDs() {
		local[id] = true
	}
	external := make(map[projects.ModuleID]moduleGenSnapshot)
	for id, snapshot := range snapshots {
		if !local[id] {
			external[id] = snapshot
		}
	}
	if len(external) == 0 {
		t.Fatal("external dependency modules were not committed")
	}
	return external
}
