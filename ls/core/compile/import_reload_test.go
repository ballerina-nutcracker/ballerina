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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

const (
	noImportSource   = "public function main() {\n}\n"
	ioImportSource   = "import ballerina/io;\n\npublic function main() {\n    io:println(1);\n}\n"
	depUsageNoImport = "public function main() {\n    dep:Public value = {};\n}\n"
)

func compileMessages(t *testing.T, svc *CompilationService, u workspace.DocumentURI) []string {
	t.Helper()
	root, err := svc.projects.ProjectKey(u)
	if err != nil {
		t.Fatalf("ProjectKey: %v", err)
	}
	svc.Flush()
	byURI, _, ok := svc.DiagnosticsFor(root)
	if !ok {
		t.Fatal("scheduled compilation has no stable diagnostics")
	}
	diags, ok := byURI[u]
	if !ok {
		t.Fatalf("scheduled diagnostics omit %s", u)
	}
	messages := make([]string, 0, len(diags))
	for _, diag := range diags {
		messages = append(messages, diag.Message)
	}
	return messages
}

func hasUnknown(messages []string) bool {
	for _, message := range messages {
		if strings.HasPrefix(message, "Unknown import") || strings.HasPrefix(message, "Unknown symbol") {
			return true
		}
	}
	return false
}

func requireNoUnknown(t *testing.T, messages []string) {
	t.Helper()
	if hasUnknown(messages) {
		t.Fatalf("diagnostics = %q, want no unknown import or symbol", messages)
	}
}

func currentPackage(t *testing.T, ps *workspace.ProjectService, u workspace.DocumentURI) (projects.Project, *projects.Package) {
	t.Helper()
	project, err := ps.Project(u)
	if err != nil || project == nil {
		t.Fatalf("Project: %v", err)
	}
	return project, project.CurrentPackage()
}

func requireSamePackage(t *testing.T, ps *workspace.ProjectService, u workspace.DocumentURI, project projects.Project, pkg *projects.Package) {
	t.Helper()
	gotProject, gotPackage := currentPackage(t, ps, u)
	if gotProject != project || gotPackage != pkg {
		t.Fatal("package was reloaded, want the same project and package")
	}
}

func requireReloaded(t *testing.T, ps *workspace.ProjectService, u workspace.DocumentURI, project projects.Project, pkg *projects.Package) {
	t.Helper()
	gotProject, gotPackage := currentPackage(t, ps, u)
	if gotProject == project || gotPackage == pkg {
		t.Fatal("package was not reloaded, want a new project and package")
	}
}

func TestAddedImportResolvesAfterEdit(t *testing.T) {
	t.Parallel()
	ps, svc := newTestServices(t)
	u := tempFileURI(t, "main.bal")
	applyOpen(t, ps, u, noImportSource)
	requireNoUnknown(t, compileMessages(t, svc, u))

	updateDoc(t, ps, u.String(), ioImportSource, 2)

	requireNoUnknown(t, compileMessages(t, svc, u))
}

func TestAddedLocalRepositoryImportResolvesAfterEdit(t *testing.T) {
	t.Parallel()
	ps, svc, u := openExternalDependencyServices(t, noImportSource)
	requireNoUnknown(t, compileMessages(t, svc, u))

	updateDoc(t, ps, u.String(), externalDependencySource, 2)

	requireNoUnknown(t, compileMessages(t, svc, u))
}

func TestRemovedImportReloadsAndDropsDependencySymbols(t *testing.T) {
	t.Parallel()
	ps, svc, u := openExternalDependencyServices(t, externalDependencySource)
	requireNoUnknown(t, compileMessages(t, svc, u))
	project, pkg := currentPackage(t, ps, u)

	updateDoc(t, ps, u.String(), depUsageNoImport, 2)

	requireReloaded(t, ps, u, project, pkg)
	messages := compileMessages(t, svc, u)
	if !hasUnknown(messages) {
		t.Fatal("dep:Public must be an unknown symbol once the import is removed")
	}
}

func TestHalfTypedImportDoesNotReload(t *testing.T) {
	t.Parallel()
	ps, svc := newTestServices(t)
	u := tempFileURI(t, "main.bal")
	applyOpen(t, ps, u, noImportSource)
	compileMessages(t, svc, u)
	project, pkg := currentPackage(t, ps, u)

	updateDoc(t, ps, u.String(), "import ballerina/i\n"+noImportSource, 2)

	requireSamePackage(t, ps, u, project, pkg)
}

func TestReorderAndAliasChangesDoNotReload(t *testing.T) {
	t.Parallel()
	ps, svc := newTestServices(t)
	u := tempFileURI(t, "main.bal")
	applyOpen(t, ps, u, "import ballerina/io;\nimport ballerina/lang.'int;\n\n"+noImportSource)
	compileMessages(t, svc, u)
	project, pkg := currentPackage(t, ps, u)

	updateDoc(t, ps, u.String(), "import ballerina/lang.'int;\nimport ballerina/io;\n\n"+noImportSource, 2)
	requireSamePackage(t, ps, u, project, pkg)

	updateDoc(t, ps, u.String(), "import ballerina/lang.'int;\nimport ballerina/io as stdio;\n\n"+noImportSource, 3)
	requireSamePackage(t, ps, u, project, pkg)
}

func TestEditWithoutImportChangeKeepsPackageAndCarriesForward(t *testing.T) {
	t.Parallel()
	ps, svc := newTestServices(t)
	u := tempFileURI(t, "main.bal")
	applyOpen(t, ps, u, ioImportSource)
	requireNoUnknown(t, compileMessages(t, svc, u))
	project, pkg := currentPackage(t, ps, u)

	updateDoc(t, ps, u.String(), strings.Replace(ioImportSource, "io:println(1)", "io:println(2)", 1), 2)

	requireSamePackage(t, ps, u, project, pkg)
	requireNoUnknown(t, compileMessages(t, svc, u))
}

func openExternalDependencyServices(t *testing.T, source string) (*workspace.ProjectService, *CompilationService, workspace.DocumentURI) {
	t.Helper()
	root := t.TempDir()
	writeExternalDependencyFixture(t, root)

	bus := event.New()
	repository := projects.NewFileSystemRepository(os.DirFS(filepath.Join(root, "repository")), ".")
	ps := workspace.New(os.DirFS, bus, workspace.WithRepositories([]projects.Repository{repository}))
	svc := New(ps, bus, WithDebounce(0))
	t.Cleanup(func() { svc.Shutdown(); bus.Close() })
	u := fileURI(t, filepath.Join(root, "main.bal"))
	applyOpen(t, ps, u, source)
	return ps, svc, u
}
