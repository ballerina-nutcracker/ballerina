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

package compile

import (
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
)

func moduleIDFor(t *testing.T, pkg *projects.Package, name string) projects.ModuleID {
	t.Helper()
	module := pkg.ModuleByName(mustModuleName(t, pkg, name))
	if module == nil {
		t.Fatalf("no module named %q", name)
	}
	return module.ModuleID()
}

func TestTopoTailScope_IncludesEverythingFromEarliestDirtyOnward(t *testing.T) {
	t.Parallel()
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	pd := newPackageDriver(pkg, projSvc.OpenText, nil, nil)
	modules := pd.topoModules()

	greetID := moduleIDFor(t, pkg, "multimoduleproject.greet")
	consumerID := moduleIDFor(t, pkg, "multimoduleproject.consumer")
	mainID := moduleIDFor(t, pkg, "multimoduleproject")

	scope := topoTailScope(modules, map[projects.ModuleID]bool{greetID: true})

	for _, id := range []projects.ModuleID{greetID, consumerID, mainID} {
		if !scope[id] {
			t.Errorf("topoTailScope missing module id %v that must follow the dirty module in topo order", id)
		}
	}
}

func TestTopoTailScope_EmptyDirtyProducesEmptyScope(t *testing.T) {
	t.Parallel()
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)
	pd := newPackageDriver(pkg, projSvc.OpenText, nil, nil)
	modules := pd.topoModules()

	scope := topoTailScope(modules, map[projects.ModuleID]bool{})
	if len(scope) != 0 {
		t.Errorf("topoTailScope with empty dirty set = %v, want empty", scope)
	}
}

func TestDependentClosure_OnlyTrueDependents(t *testing.T) {
	t.Parallel()
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)

	depGraph := pkg.Resolution().ModuleDependencyGraph()
	reverse := reverseDependents(pkg, depGraph)

	greetID := moduleIDFor(t, pkg, "multimoduleproject.greet")
	consumerID := moduleIDFor(t, pkg, "multimoduleproject.consumer")
	mainID := moduleIDFor(t, pkg, "multimoduleproject")
	standaloneID := moduleIDFor(t, pkg, "multimoduleproject.standalone")

	scope := dependentClosure(map[projects.ModuleID]bool{greetID: true}, reverse)

	for _, id := range []projects.ModuleID{greetID, consumerID, mainID} {
		if !scope[id] {
			t.Errorf("dependentClosure missing true dependent id %v", id)
		}
	}
	if scope[standaloneID] {
		t.Error("dependentClosure incorrectly included standalone, which has no dependency edge to greet")
	}
}

func TestModuleTextHash_StableForUnchangedContent_DiffersOnEdit(t *testing.T) {
	t.Parallel()
	projSvc := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projSvc)
	greetModule := pkg.ModuleByName(mustModuleName(t, pkg, "multimoduleproject.greet"))

	h1 := moduleTextHash(greetModule, projSvc.OpenText)
	h2 := moduleTextHash(greetModule, projSvc.OpenText)
	if h1 != h2 {
		t.Fatalf("hash changed with no edit: %q != %q", h1, h2)
	}

	dir := multimoduleFixtureDir(t)
	greetURI := fileURI(t, filepath.Join(dir, "modules", "greet", "greet.bal"))
	applyOpen(t, projSvc, greetURI, "public function greeting() returns string {\n    return \"hi\";\n}\n")

	h3 := moduleTextHash(greetModule, projSvc.OpenText)
	if h3 == h1 {
		t.Fatal("hash unchanged after editing greet.bal's content")
	}
}
