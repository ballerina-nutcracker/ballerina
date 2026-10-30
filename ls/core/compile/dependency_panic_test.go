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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
)

const dependencyPanicValue = "uncached dependency repository panic"

type panickingDependencyRepository struct {
	projects.Repository
	attempts int
}

func (r *panickingDependencyRepository) GetPackage(ctx context.Context, org, name, version string, options projects.ResolutionOptions) (*projects.Package, error) {
	if org == "acme" && name == "dep" {
		r.attempts++
		panic(dependencyPanicValue)
	}
	return r.Repository.GetPackage(ctx, org, name, version, options)
}

func (r *panickingDependencyRepository) GetPackageVersions(ctx context.Context, org, name string, options projects.ResolutionOptions) ([]projects.PackageVersion, error) {
	if org == "acme" && name == "dep" {
		return nil, nil
	}
	return r.Repository.GetPackageVersions(ctx, org, name, options)
}

func TestDependencyPanicPreservesLocalAndLaterModuleProgress(t *testing.T) {
	t.Parallel()
	for _, logging := range []bool{true, false} {
		name := "host writer"
		if !logging {
			name = "no-op default"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeExternalDependencyFixture(t, root)
			laterPath := filepath.Join(root, "modules", "later", "later.bal")
			if err := os.MkdirAll(filepath.Dir(laterPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(laterPath, []byte("public function later() returns int { return 42; }\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			bus := event.New()
			defer bus.Close()
			repository := &panickingDependencyRepository{
				Repository: projects.NewFileSystemRepository(os.DirFS(filepath.Join(root, "repository")), "."),
			}
			projectService := workspace.New(os.DirFS, bus, workspace.WithRepositories([]projects.Repository{repository}))
			var logs bytes.Buffer
			options := []Option{WithDebounce(0)}
			if logging {
				logger := observability.New(&logs)
				options = append(options, WithLogger(logger))
			}
			compiler := New(projectService, bus, options...)
			defer compiler.Shutdown()
			failed := collectGen(t, bus, event.CompilationFailed)
			succeeded := collectGen(t, bus, event.CompilationSucceeded)
			descriptor, err := projects.NewPackageDescriptorFromStrings("acme", "dep", "0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			var driver *packageDriver
			compiler.compileFn = func(pkg *projects.Package, cyc compileCycle) cycleResult {
				driver = newPackageDriver(pkg, cyc.openText, cyc.priorState, cyc.stale)
				driver.logger = cyc.logger
				driver.loadResolvedDependency(descriptor)
				driver.loadResolvedDependency(descriptor)
				modules := driver.topoModules()
				// Drive the independent module after the module using the unavailable dependency.
				for i, module := range modules {
					if module.Descriptor().Name().String() == "main" {
						modules[0], modules[i] = modules[i], modules[0]
						break
					}
				}
				driver.advanceAll(stageCFGAnalyzed)
				cyc.commit(driver.snapshotForCommit())
				resolution := pkg.Resolution().DiagnosticResult()
				byFile, resByFile := extractByFile(pkg, driver.allDiagnostics(), resolution.Diagnostics(), driver.env.DiagnosticEnv(), cyc.openText)
				return cycleResult{byFile: byFile, resByFile: resByFile, resolutionErrored: resolution.HasErrors(), descriptor: pkg.Descriptor().String()}
			}
			uri := fileURI(t, filepath.Join(root, "main.bal"))
			applyOpen(t, projectService, uri, externalDependencySource)
			compiler.Flush()

			if repository.attempts != 1 {
				t.Fatalf("GetPackage attempts = %d, want one uncached panic and no recursive retry", repository.attempts)
			}
			if len(failed.slice()) != 0 || len(succeeded.slice()) != 1 {
				t.Fatalf("failed generations = %v, succeeded generations = %v", failed.slice(), succeeded.slice())
			}
			project, err := projectService.Project(uri)
			if err != nil || project == nil {
				t.Fatalf("Project: %v", err)
			}
			generation, ok := projectService.Generation(root)
			if !ok || compiler.generationFailed(root, generation) {
				t.Fatal("dependency panic must not fail the root generation")
			}
			if _, ok := compiler.store.Stable(root); !ok {
				t.Fatal("dependency panic must retain the root snapshot")
			}
			packageID := semantics.PackageIdentifier{OrgName: "acme", ModuleName: "dep"}
			if !driver.externalMaterializing[packageID] {
				t.Error("dependency recursion guard was cleared after panic")
			}
			if _, ok := driver.publicSymbols[packageID]; ok {
				t.Error("panicked dependency must not publish symbols")
			}
			if _, ok := driver.externalProjections[packageID]; ok {
				t.Error("panicked dependency must not publish a projection")
			}
			mainDriver, ok := moduleDriverFor(t, driver, project.CurrentPackage(), "main")
			if !ok {
				t.Fatal("local module driver unavailable")
			}
			if mainDriver.currentStage() < stageSymbolResolved || mainDriver.pkgNode == nil {
				t.Errorf("local module did not retain a symbol-resolved package: stage = %v", mainDriver.currentStage())
			}
			missingSymbol := false
			for _, diag := range mainDriver.publishableDiagnostics() {
				message := diag.Message()
				if message == "Unknown symbol: Public" {
					missingSymbol = true
				}
				if strings.Contains(message, dependencyPanicValue) || strings.Contains(message, "INTERNAL_ERROR") {
					t.Errorf("panic leaked into compiler diagnostics: %s", message)
				}
			}
			if !missingSymbol {
				t.Error("missing dependency did not retain the missing-symbol diagnostic fallback")
			}
			laterDriver, ok := moduleDriverFor(t, driver, project.CurrentPackage(), "main.later")
			if !ok || laterDriver.currentStage() != stageCFGAnalyzed {
				t.Fatal("later independent module did not complete real compilation")
			}
			if logging {
				output := logs.String()
				for _, field := range []string{"level=ERROR", `recovered="` + dependencyPanicValue + `"`, "descriptor=" + descriptor.String()} {
					if !strings.Contains(output, field) {
						t.Errorf("host log missing %q: %s", field, output)
					}
				}
				if count := strings.Count(output, dependencyPanicValue); count != 1 {
					t.Errorf("dependency recovery records = %d, want one: %s", count, output)
				}
			} else if logs.Len() != 0 {
				t.Errorf("default logger wrote %q", logs.String())
			}
		})
	}
}
