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
	"context"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func TestSealedSourceRetainsParserInputAcrossEditAndEviction(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	bus := event.New()
	defer bus.Close()
	projects := workspace.New(platform, bus)
	compiler := New(projects, bus, WithDebounce(0))
	defer compiler.Shutdown()
	root := t.TempDir()
	if err := platform.FS.WriteFile(filepath.Join(root, "Ballerina.toml"), []byte("[package]\norg = \"testorg\"\nname = \"source66\"\nversion = \"0.1.0\"\n")); err != nil {
		t.Fatal(err)
	}
	if err := platform.FS.WriteFile(filepath.Join(root, "main.bal"), []byte("public function main() {}\n")); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(t, "file://"+filepath.Join(root, "main.bal"))
	before := "public function main() { /* exact overlay input */ }\n"
	applyOpen(t, projects, uri, before)
	old, ok := compiler.SealedModuleFor(context.Background(), uri)
	if !ok {
		t.Fatal("old view unavailable")
	}
	if text, ok := old.SourceText(uri); !ok || text != before {
		t.Fatalf("old source = %q %v", text, ok)
	}
	after := "public function main() { /* replacement parser input */ }\n"
	updateDoc(t, projects, uri.String(), after, 2)
	latest, ok := compiler.SealedModuleFor(context.Background(), uri)
	if !ok {
		t.Fatal("new view unavailable")
	}
	if text, ok := latest.SourceText(uri); !ok || text != after {
		t.Fatalf("new source = %q %v", text, ok)
	}
	if text, ok := old.SourceText(uri); !ok || text != before {
		t.Fatalf("old source changed = %q %v", text, ok)
	}
	other := fileURI(t, "file://"+filepath.Join(root, "other.bal"))
	otherText := "function sibling() {}\n"
	if err := platform.FS.WriteFile(other.Path(), []byte(otherText)); err != nil {
		t.Fatal(err)
	}
	applyOpen(t, projects, other, otherText)
	reloaded, ok := compiler.SealedModuleFor(context.Background(), uri)
	if !ok {
		t.Fatal("reloaded view unavailable")
	}
	if reloaded.sourceIDs[uri.Path()] == old.sourceIDs[uri.Path()] {
		t.Fatal("structural reload did not mint a new document identity")
	}
	if text, ok := reloaded.SourceText(uri); !ok || text != after {
		t.Fatalf("reloaded source = %q %v", text, ok)
	}
	if text, ok := reloaded.SourceText(other); !ok || text != otherText {
		t.Fatalf("added source = %q %v", text, ok)
	}
	if _, ok := old.SourceText(other); ok {
		t.Fatal("old view resolved an uncaptured identity through the live project")
	}
	if text, ok := latest.SourceText(uri); !ok || text != after {
		t.Fatalf("prior identity source = %q %v", text, ok)
	}
	compiler.Shutdown()
	if text, ok := old.SourceText(uri); !ok || text != before {
		t.Fatalf("evicted source changed = %q %v", text, ok)
	}
}

func TestSealedSourceCarriesForwardWithAST(t *testing.T) {
	projects := newProjectOnlyService(t)
	pkg, _ := openMultimoduleFixture(t, projects)
	first := newPackageDriver(pkg, projects.OpenText, nil, nil)
	first.advanceAll(stageCFGAnalyzed)
	state := first.snapshotForCommit()
	consumerID := moduleIDFor(t, pkg, "multimoduleproject.consumer")
	consumer := state[consumerID].driver
	consumerURI := fileURI(t, "file://"+filepath.Join(multimoduleFixtureDir(t), "modules", "consumer", "consumer.bal"))
	view := SealedModule{sources: consumer.sources, sourceIDs: consumer.sourceIDs}
	original, ok := view.SourceText(consumerURI)
	if !ok || original == "" {
		t.Fatal("source missing from initial AST")
	}
	greetURI := fileURI(t, "file://"+filepath.Join(multimoduleFixtureDir(t), "modules", "greet", "greet.bal"))
	applyOpen(t, projects, greetURI, "public function greeting() returns string { return \"changed\"; }\n")
	project, err := projects.Project(greetURI)
	if err != nil {
		t.Fatal(err)
	}
	next := newPackageDriver(project.CurrentPackage(), projects.OpenText, &packageGenState{modules: state}, nil)
	next.advanceAll(stageCFGAnalyzed)
	carried := next.drivers[consumerID]
	if carried != consumer {
		t.Fatal("unchanged consumer driver was not reused")
	}
	newView := SealedModule{sources: carried.sources, sourceIDs: carried.sourceIDs}
	if text, ok := newView.SourceText(consumerURI); !ok || text != original {
		t.Fatalf("carried source = %q %v", text, ok)
	}
	if text, ok := view.SourceText(consumerURI); !ok || text != original {
		t.Fatalf("old source = %q %v", text, ok)
	}
}
