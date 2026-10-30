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

package compilerplugin_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/testplan"

	// Registering ballerina/test's compiler plugin happens through the
	// generated compilerpluginregistry, which projects.Load consults
	// automatically — no explicit wiring needed here, the same as any real
	// `bal` invocation.
	_ "github.com/ballerina-nutcracker/ballerina/compilerpluginregistry"
)

// loadTestdataProject loads and compiles the project — compilation is where
// the plugin actually runs, not loading, so the diagnostics returned here
// come from *compilation.DiagnosticResult(), not the load result.
func loadTestdataProject(t *testing.T, name string) *projects.PackageCompilation {
	t.Helper()

	absPath, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve user home: %v", err)
	}
	ballerinaEnvPath := os.Getenv(projects.BallerinaEnvVar)
	if ballerinaEnvPath == "" {
		ballerinaEnvPath = filepath.Join(userHome, projects.UserHomeDirName)
	}

	// SkipTests must be false — the plugin only ever sees @test:* functions
	// if test sources are actually part of compilation.
	buildOpts := projects.NewBuildOptionsBuilder().
		WithOffline(true).
		WithSkipTests(false).
		Build()

	result, err := projects.Load(os.DirFS(absPath), ".", projects.ProjectLoadConfig{
		BallerinaEnvFs: os.DirFS(ballerinaEnvPath),
		BuildOptions:   &buildOpts,
	})
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	if diag := result.Diagnostics(); diag.HasErrors() {
		t.Fatalf("project failed to load: %+v", diag.Diagnostics())
	}
	return result.Project().CurrentPackage().Compilation()
}

// TestProcessTests_FiresDuringCompilation confirms the plugin actually
// activates for a module that imports ballerina/test (no manual invocation —
// this is what a real `bal` compile does) and stashes a complete TestPlan.
// This fixture was originally copied from the now-retired
// cli/internal/testdiscovery/testdata/sample to check the two independent
// producers for field-by-field parity before cli/cmd/test.go's wiring
// switched over to this package exclusively (see the implementation plan's
// Phase 3.4 notes).
func TestProcessTests_FiresDuringCompilation(t *testing.T) {
	compilation := loadTestdataProject(t, "sample")
	if diag := compilation.DiagnosticResult(); diag.HasErrors() {
		var messages []string
		for _, d := range diag.Diagnostics() {
			messages = append(messages, d.Message())
		}
		t.Fatalf("unexpected compilation errors: %v", messages)
	}

	plan, ok := testplan.Load(testplan.ModuleKey{Org: "testorg", Module: "sample"})
	if !ok {
		t.Fatal("plugin did not stash a TestPlan — did it fire at all?")
	}

	if len(plan.Tests) != 3 {
		t.Fatalf("expected 3 @test:Config functions, got %d: %+v", len(plan.Tests), plan.Tests)
	}
	var testSetup, testMain, testDisabled *testplan.TestFunction
	for i := range plan.Tests {
		switch plan.Tests[i].Name {
		case "testSetup":
			testSetup = &plan.Tests[i]
		case "testMain":
			testMain = &plan.Tests[i]
		case "testDisabled":
			testDisabled = &plan.Tests[i]
		}
	}
	// testSetup must itself be a @test:Config function — confirmed via a
	// real `bal test` run that dependsOn referencing a plain, non-test
	// function fails at runtime ("either disabled or not included"), since
	// register.bal only resolves dependsOn against registered functions.
	if testSetup == nil {
		t.Fatal("testSetup not discovered")
	}
	if !testSetup.Enable {
		t.Error("testSetup: expected Enable=true (default)")
	}
	if testMain == nil {
		t.Fatal("testMain not discovered")
	}
	if !testMain.Enable {
		t.Error("testMain: expected Enable=true (default)")
	}
	if !reflect.DeepEqual(testMain.Groups, []string{"unit", "fast"}) {
		t.Errorf("testMain: expected Groups=[unit fast], got %v", testMain.Groups)
	}
	if !reflect.DeepEqual(testMain.DependsOn, []string{"testSetup"}) {
		t.Errorf("testMain: expected DependsOn=[testSetup], got %v", testMain.DependsOn)
	}
	if testDisabled == nil {
		t.Fatal("testDisabled not discovered")
	}
	if testDisabled.Enable {
		t.Error("testDisabled: expected Enable=false")
	}

	if len(plan.BeforeSuite) != 1 || plan.BeforeSuite[0].Name != "beforeAll" {
		t.Errorf("expected BeforeSuite=[beforeAll], got %+v", plan.BeforeSuite)
	}
	if len(plan.AfterSuite) != 1 || plan.AfterSuite[0].Name != "afterAll" || !plan.AfterSuite[0].AlwaysRun {
		t.Errorf("expected AfterSuite=[{afterAll true}], got %+v", plan.AfterSuite)
	}
	if len(plan.BeforeEach) != 1 || plan.BeforeEach[0].Name != "beforeEachTest" {
		t.Errorf("expected BeforeEach=[beforeEachTest], got %+v", plan.BeforeEach)
	}
	if len(plan.AfterEach) != 1 || plan.AfterEach[0].Name != "afterEachTest" {
		t.Errorf("expected AfterEach=[afterEachTest], got %+v", plan.AfterEach)
	}
	if len(plan.BeforeGroups) != 1 || plan.BeforeGroups[0].Name != "beforeUnitGroup" {
		t.Errorf("expected BeforeGroups=[beforeUnitGroup], got %+v", plan.BeforeGroups)
	}
	if len(plan.AfterGroups) != 1 || plan.AfterGroups[0].Name != "afterUnitGroup" || !plan.AfterGroups[0].AlwaysRun {
		t.Errorf("expected AfterGroups=[{afterUnitGroup ... true}], got %+v", plan.AfterGroups)
	}
}

// TestProcessTests_RejectsDependsOnNonTestFunction is a capability a plain
// pre-compilation AST walk has no way to provide at all (it can't report
// compiler diagnostics). Confirmed via a real `bal test` run that this
// exact fixture fails at runtime with "The dependent test function is
// either disabled or not included." — this turns that into a clear
// compile-time error instead.
func TestProcessTests_RejectsDependsOnNonTestFunction(t *testing.T) {
	compilation := loadTestdataProject(t, "bad-dependson")
	diag := compilation.DiagnosticResult()
	if !diag.HasErrors() {
		t.Fatal("expected a compile error for dependsOn referencing a non-test function")
	}
	var found bool
	for _, d := range diag.Diagnostics() {
		if strings.Contains(d.Message(), "dependsOn function 'helper' is not a @test:Config function") {
			found = true
		}
	}
	if !found {
		var messages []string
		for _, d := range diag.Diagnostics() {
			messages = append(messages, d.Message())
		}
		t.Fatalf("expected a dependsOn diagnostic, got: %v", messages)
	}
}
