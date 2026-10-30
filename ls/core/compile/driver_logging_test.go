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
	"sync/atomic"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

const transitiveDependencyPanic = "transitive dependency materialization panic"

type panickingTransitiveRepository struct {
	*projects.FileSystemRepository
	attempts atomic.Int32
}

func (r *panickingTransitiveRepository) GetPackage(ctx context.Context, org, name, version string, options projects.ResolutionOptions) (*projects.Package, error) {
	if org == "acme" && name == "transitive" {
		if r.attempts.Add(1) == 1 {
			return nil, nil
		}
		panic(transitiveDependencyPanic)
	}
	return r.FileSystemRepository.GetPackage(ctx, org, name, version, options)
}

func TestRealCompilePackagePropagatesDependencyLogger(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeExternalDependencyFixture(t, root)
	dependencyGraph := `{"packages":[{"org":"acme","name":"dep","version":"0.1.0","dependencies":[{"org":"acme","name":"transitive","version":"0.1.0"}]}]}`
	graphPath := filepath.Join(root, "repository", "acme", "dep", "0.1.0", "go1.27", "dependency-graph.json")
	if err := os.WriteFile(graphPath, []byte(dependencyGraph), 0o644); err != nil {
		t.Fatal(err)
	}
	repository := &panickingTransitiveRepository{
		FileSystemRepository: projects.NewFileSystemRepository(os.DirFS(filepath.Join(root, "repository")), "."),
	}
	bus := event.New()
	defer bus.Close()
	projectService := workspace.New(os.DirFS, bus, workspace.WithRepositories([]projects.Repository{repository}))
	var logs bytes.Buffer
	logger := observability.New(&logs)
	compiler := New(projectService, bus, WithDebounce(0), WithLogger(logger))
	defer compiler.Shutdown()
	failed := collectGen(t, bus, event.CompilationFailed)
	succeeded := collectGen(t, bus, event.CompilationSucceeded)

	uri := fileURI(t, filepath.Join(root, "main.bal"))
	applyOpen(t, projectService, uri, "import acme/dep;\n\npublic function main() {\n    dep:Public value = {};\n    _ = value;\n}\n")
	compiler.Flush()

	if attempts := repository.attempts.Load(); attempts != 2 {
		t.Fatalf("GetPackage attempts = %d, want unresolved BFS lookup then uncached materialization panic", attempts)
	}
	if len(failed.slice()) != 0 || len(succeeded.slice()) != 1 {
		t.Fatalf("failed generations = %v, succeeded generations = %v", failed.slice(), succeeded.slice())
	}
	project, err := projectService.Project(uri)
	if err != nil || project == nil {
		t.Fatalf("Project: %v", err)
	}
	pkg := project.CurrentPackage()
	descriptor := pkg.Resolution().ResolvedDependencies()["acme/transitive"]
	if descriptor == nil {
		t.Fatal("manifest transitive descriptor was not retained after unresolved BFS lookup")
	}
	generation, ok := projectService.Generation(root)
	if !ok || compiler.generationFailed(root, generation) {
		t.Fatal("dependency-local recovery must not fail the root generation")
	}
	if _, ok := compiler.store.Stable(root); !ok {
		t.Fatal("real compilation did not retain a root snapshot")
	}
	state := compiler.genStateFor(root)
	for _, id := range pkg.ModuleIDs() {
		snapshot, ok := state.modules[id]
		if !ok || snapshot.driver == nil {
			t.Fatalf("root module %v has no committed driver", id)
		}
		if snapshot.driver.currentStage() != stageCFGAnalyzed {
			t.Fatalf("root module %v stopped at %v: %v", id, snapshot.driver.currentStage(), snapshot.driver.publishableDiagnostics())
		}
	}
	output := logs.String()
	for _, field := range []string{
		"level=ERROR",
		`msg="compile: recovered dependency panic"`,
		`recovered="` + transitiveDependencyPanic + `"`,
		"descriptor=" + descriptor.String(),
	} {
		if !strings.Contains(output, field) {
			t.Errorf("host log missing %q: %s", field, output)
		}
	}
	if count := strings.Count(output, transitiveDependencyPanic); count != 1 {
		t.Errorf("dependency recovery records = %d, want one: %s", count, output)
	}

	quietResult := realCompilePackage(pkg, compileCycle{openText: projectService.OpenText})
	if quietResult.descriptor != pkg.Descriptor().Name().Value() {
		t.Error("nil cycle logger prevented real compilation")
	}
	if attempts := repository.attempts.Load(); attempts != 3 {
		t.Errorf("nil cycle logger: GetPackage attempts = %d, want another recovered materialization panic", attempts)
	}
	if logs.String() != output {
		t.Error("nil cycle logger reused the host writer instead of the quiet default")
	}
}
