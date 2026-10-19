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
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

const (
	intFromStringSource   = "public function main() {\n    int|error n = int:fromString(\"123\");\n}\n"
	intFromStringImported = "import ballerina/lang.'int;\n\n" + intFromStringSource
	stringLengthSource    = "public function main() {\n    int n = string:length(\"abc\");\n}\n"
	arrayLengthSource     = "public function main() {\n    int[] a = [1, 2];\n    int n = array:length(a);\n}\n"
	runtimeSleepSource    = "public function main() {\n    runtime:sleep(1);\n}\n"
	runtimeSleepImported  = "import ballerina/lang.runtime;\n\n" + runtimeSleepSource
)

func TestImplicitLangLibResolvesWithoutImport(t *testing.T) {
	ps, svc := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, ps, u, intFromStringSource)

	requireNoUnknown(t, compileMessages(t, svc, u))
}

func TestSecondImplicitLangLibsResolveWithoutImport(t *testing.T) {
	for name, source := range map[string]string{"string": stringLengthSource, "array": arrayLengthSource} {
		t.Run(name, func(t *testing.T) {
			ps, svc := newTestServices(t)
			u := fileURI(t, "file:///workspace/main.bal")
			applyOpen(t, ps, u, source)

			requireNoUnknown(t, compileMessages(t, svc, u))
		})
	}
}

func TestExplicitLangLibImportStillResolvesAndCompilesOnce(t *testing.T) {
	ps, svc := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, ps, u, intFromStringImported)
	requireNoUnknown(t, compileMessages(t, svc, u))

	importingInt := spacesAfterAdvance(t, intAndStringSource("lang.'int"))
	importingString := spacesAfterAdvance(t, intAndStringSource("lang.'string"))
	if importingInt != importingString {
		t.Fatalf("symbol spaces importing lang.int = %d, importing lang.string = %d, want equal (each library compiled once)", importingInt, importingString)
	}
}

func TestLangRuntimeNeedsExplicitImport(t *testing.T) {
	ps, svc := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, ps, u, runtimeSleepSource)
	if messages := compileMessages(t, svc, u); len(messages) == 0 {
		t.Fatal("runtime:sleep without an import must report a diagnostic")
	}

	ps, svc = newTestServices(t)
	applyOpen(t, ps, u, runtimeSleepImported)
	if messages := compileMessages(t, svc, u); len(messages) != 0 {
		t.Fatalf("diagnostics = %q, want none with explicit lang.runtime import", messages)
	}
}

func TestImplicitLangLibDriversAdoptedAfterBodyEdit(t *testing.T) {
	ps := newDriverOnlyProjectService(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, ps, u, intFromStringSource)
	_, packageOne := currentPackage(t, ps, u)
	generationOne := newPackageDriver(packageOne, ps.OpenText, nil, nil)
	generationOne.advanceAll(stageTopLevelTypeResolved)
	committed := generationOne.snapshotForCommit()
	langLibsOne := externalSnapshots(t, packageOne, committed)
	if len(langLibsOne) < len(migratedLangLibs) {
		t.Fatalf("committed %d library modules, want at least %d", len(langLibsOne), len(migratedLangLibs))
	}

	updateDoc(t, ps, u.String(), strings.Replace(intFromStringSource, "123", "456", 1), 2)
	_, packageTwo := currentPackage(t, ps, u)
	generationTwo := newPackageDriver(packageTwo, ps.OpenText, &packageGenState{modules: committed}, nil)
	generationTwo.advanceAll(stageTopLevelTypeResolved)
	langLibsTwo := externalSnapshots(t, packageTwo, generationTwo.snapshotForCommit())

	for id, snapshotOne := range langLibsOne {
		snapshotTwo, ok := langLibsTwo[id]
		if !ok {
			t.Fatalf("library module %v missing from generation two", id)
		}
		if snapshotTwo.driver != snapshotOne.driver {
			t.Errorf("library module %v was rebuilt after a body edit", id)
		}
	}
}

func TestImplicitLangLibsResolveAfterImportChangeReload(t *testing.T) {
	ps, svc := newTestServices(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, ps, u, intFromStringSource)
	requireNoUnknown(t, compileMessages(t, svc, u))
	project, pkg := currentPackage(t, ps, u)

	updateDoc(t, ps, u.String(), "import ballerina/io;\n\n"+intFromStringSource, 2)

	requireReloaded(t, ps, u, project, pkg)
	requireNoUnknown(t, compileMessages(t, svc, u))
}

func TestSymbolSpacesDoNotGrowWithLangLibsAcrossGenerations(t *testing.T) {
	ps := newDriverOnlyProjectService(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, ps, u, intFromStringSource)
	_, pkg := currentPackage(t, ps, u)
	generation := newPackageDriver(pkg, ps.OpenText, nil, nil)
	generation.advanceAll(stageTopLevelTypeResolved)
	committed := generation.snapshotForCommit()
	libraryCost := symbolSpaceCount(pkg)

	var growth []int
	for version := int32(2); version <= 5; version++ {
		before := symbolSpaceCount(pkg)
		updateDoc(t, ps, u.String(), strings.Replace(intFromStringSource, "123", fmt.Sprint(version), 1), version)
		_, pkg = currentPackage(t, ps, u)
		generation = newPackageDriver(pkg, ps.OpenText, &packageGenState{modules: committed}, nil)
		generation.advanceAll(stageTopLevelTypeResolved)
		committed = generation.snapshotForCommit()
		growth = append(growth, symbolSpaceCount(pkg)-before)
	}
	for _, delta := range growth {
		if delta != growth[0] {
			t.Fatalf("symbol space growth per generation = %v, want constant", growth)
		}
	}
	if growth[0]*5 > libraryCost {
		t.Fatalf("growth per generation = %d against %d spaces after the first, want only the edited module", growth[0], libraryCost)
	}
}

func spacesAfterAdvance(t *testing.T, source string) int {
	t.Helper()
	ps := newDriverOnlyProjectService(t)
	u := fileURI(t, "file:///workspace/main.bal")
	applyOpen(t, ps, u, source)
	_, pkg := currentPackage(t, ps, u)
	newPackageDriver(pkg, ps.OpenText, nil, nil).advanceAll(stageTopLevelTypeResolved)
	return symbolSpaceCount(pkg)
}

func intAndStringSource(importedLib string) string {
	return "import ballerina/" + importedLib + ";\n\npublic function main() {\n    int|error n = int:fromString(\"1\");\n    int m = string:length(\"a\");\n}\n"
}

// newDriverOnlyProjectService creates a workspace without a background compiler.
func newDriverOnlyProjectService(t *testing.T) *workspace.ProjectService {
	t.Helper()
	bus := event.New()
	t.Cleanup(bus.Close)
	return workspace.New(os.DirFS, bus)
}

func symbolSpaceCount(pkg *projects.Package) int {
	env := pkg.Project().Environment().CompilerEnvironment()
	return env.NewSymbolSpace(*env.GetDefaultPackage()).SpaceIndex()
}
