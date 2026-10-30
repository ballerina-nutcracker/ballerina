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
)

func TestPruningMultipleDirtyRootsAndOrders(t *testing.T) {
	t.Parallel()
	orders := [][]string{{"a", "b", "d", "c", "leaf", ""}, {"d", "a", "b", "leaf", "c", ""}}
	for _, changed := range []string{"a", "d"} {
		for orderIndex, order := range orders {
			t.Run(changed+order[0], func(t *testing.T) {
				t.Parallel()
				ws := newProjectOnlyService(t)
				dir, err := filepath.Abs("testdata/pruning")
				if err != nil {
					t.Fatal(err)
				}
				main := fileURI(t, filepath.Join(dir, "main.bal"))
				applyOpen(t, ws, main, mustReadFixture(t, main.Path()))
				project, err := ws.Project(main)
				if err != nil {
					t.Fatal(err)
				}
				pkg := project.CurrentPackage()
				first := newPackageDriver(pkg, ws.OpenText, nil, nil)
				first.advanceAll(stageCFGAnalyzed)
				if len(first.allDiagnostics()) != 0 {
					t.Fatalf("fixture diagnostics: %v", first.allDiagnostics())
				}
				prior := first.snapshotForCommit()
				for _, root := range []string{"a", "d"} {
					text := "public function value() returns int { return 2; }\n"
					if root == changed {
						text = "public function value() returns string { return \"two\"; }\n"
					}
					applyOpen(t, ws, fileURI(t, filepath.Join(dir, "modules", root, root+".bal")), text)
				}
				project, err = ws.Project(main)
				if err != nil {
					t.Fatal(err)
				}
				pkg = project.CurrentPackage()
				next := newPackageDriver(pkg, ws.OpenText, &packageGenState{modules: prior}, nil)
				for _, suffix := range order {
					name := "pruning"
					if suffix != "" {
						name += "." + suffix
					}
					next.order = append(next.order, pkg.ModuleByName(mustModuleName(t, pkg, name)))
				}
				next.advanceAll(stageCFGAnalyzed)
				if n, ok := moduleDiagCount(t, next, pkg, "pruning.c"); !ok || n != 1 {
					t.Fatalf("order %d: missing consumer diagnostic: count=%d, driven=%v", orderIndex, n, ok)
				}
				for _, suffix := range []string{"a", "d"} {
					id := moduleIDFor(t, pkg, "pruning."+suffix)
					if !next.directlyDirty[id] || next.drivers[id] == prior[id].driver {
						t.Fatalf("dirty root %s was reused", suffix)
					}
				}
				leaf := moduleIDFor(t, pkg, "pruning.leaf")
				if next.drivers[leaf] != prior[leaf].driver {
					t.Fatal("unchanged intermediate must stop propagation along its own path")
				}
			})
		}
	}
}

func TestPruningUnknownAndFailedInputs(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"unknown prior", "unsupported", "failed"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			ws := newProjectOnlyService(t)
			pkg, _ := openMultimoduleFixture(t, ws)
			first := newPackageDriver(pkg, ws.OpenText, nil, nil)
			first.advanceAll(stageCFGAnalyzed)
			prior := first.snapshotForCommit()
			greet := moduleIDFor(t, pkg, "multimoduleproject.greet")
			consumer := moduleIDFor(t, pkg, "multimoduleproject.consumer")
			if scenario == "unknown prior" {
				snapshot := prior[greet]
				snapshot.fingerprint = fingerprint{}
				prior[greet] = snapshot
			}
			text := "public function greeting() returns string { return \"edited\"; }\n"
			if scenario == "unsupported" {
				text += "public type R record {| int n = 1; |};\n"
			}
			if scenario == "failed" {
				text = "public function greeting() returns UndefinedType { return 1; }\n"
			}
			uri := fileURI(t, filepath.Join(multimoduleFixtureDir(t), "modules/greet/greet.bal"))
			applyOpen(t, ws, uri, text)
			project, err := ws.Project(uri)
			if err != nil {
				t.Fatal(err)
			}
			next := newPackageDriver(project.CurrentPackage(), ws.OpenText, &packageGenState{modules: prior}, nil)
			next.advanceAll(stageCFGAnalyzed)
			if !next.resetScope[consumer] || next.drivers[consumer] == prior[consumer].driver {
				t.Fatal("unknown/unsupported/failed input must not reuse consumer")
			}
			if scenario == "failed" {
				if !next.phase1Errored[consumer] || next.drivers[consumer] != nil {
					t.Fatal("failed input must preserve dependent stage gating")
				}
			}
		})
	}
}

func TestPruningTwoInvalidFingerprintsCannotPermitReuse(t *testing.T) {
	t.Parallel()
	ws := newProjectOnlyService(t)
	openMultimoduleFixture(t, ws)
	uri := fileURI(t, filepath.Join(multimoduleFixtureDir(t), "modules/greet/greet.bal"))
	applyOpen(t, ws, uri, "public function greeting() returns string { return \"hello\"; }\npublic type R record {| int n = 1; |};\n")
	project, err := ws.Project(uri)
	if err != nil {
		t.Fatal(err)
	}
	pkg := project.CurrentPackage()
	first := newPackageDriver(pkg, ws.OpenText, nil, nil)
	first.advanceAll(stageCFGAnalyzed)
	prior := first.snapshotForCommit()
	applyOpen(t, ws, uri, "public function greeting() returns string { return \"edited\"; }\npublic type R record {| int n = 1; |};\n")
	project, err = ws.Project(uri)
	if err != nil {
		t.Fatal(err)
	}
	next := newPackageDriver(project.CurrentPackage(), ws.OpenText, &packageGenState{modules: prior}, nil)
	next.advanceAll(stageCFGAnalyzed)
	consumer := moduleIDFor(t, pkg, "multimoduleproject.consumer")
	if next.drivers[consumer] == prior[consumer].driver {
		t.Fatal("two invalid fingerprints must not permit dependent reuse")
	}
}
