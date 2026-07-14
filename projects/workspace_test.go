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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ballerina-lang-go/projects"
	"ballerina-lang-go/test_util"
)

// Probes that exercise WORKSPACE-SUPPORT-SPEC.md acceptance criteria items
// not covered by TestWorkspaceProjectLoading and TestWorkspaceRepositoryPriority.

// TestWorkspace_DocumentIDAcrossMembers verifies WorkspaceProject.DocumentID
// finds documents across all member packages.
func TestWorkspace_DocumentIDAcrossMembers(t *testing.T) {
	require := test_util.NewRequire(t)

	absPath, err := filepath.Abs(filepath.Join("testdata", "workspace-simple"))
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	ws := result.Project().(*projects.WorkspaceProject)

	if _, ok := ws.DocumentID("pkg-a/main.bal"); !ok {
		t.Errorf("DocumentID(\"pkg-a/main.bal\") returned ok=false; expected to be found")
	}
	if _, ok := ws.DocumentID("pkg-b/lib.bal"); !ok {
		t.Errorf("DocumentID(\"pkg-b/lib.bal\") returned ok=false; expected to be found")
	}
	if _, ok := ws.DocumentID("does-not-exist.bal"); ok {
		t.Errorf("DocumentID(\"does-not-exist.bal\") returned ok=true; expected not found")
	}
}

// TestWorkspace_DocumentPathAcrossMembers verifies WorkspaceProject.DocumentPath
// resolves IDs from any member.
func TestWorkspace_DocumentPathAcrossMembers(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	absPath, err := filepath.Abs(filepath.Join("testdata", "workspace-simple"))
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	ws := result.Project().(*projects.WorkspaceProject)

	idA, ok := ws.DocumentID("pkg-a/main.bal")
	if !ok {
		t.Fatal("DocumentID should find pkg-a/main.bal")
	}
	assert.Equal("pkg-a/main.bal", ws.DocumentPath(idA))

	idB, ok := ws.DocumentID("pkg-b/lib.bal")
	if !ok {
		t.Fatal("DocumentID should find pkg-b/lib.bal")
	}
	assert.Equal("pkg-b/lib.bal", ws.DocumentPath(idB))
}

// TestWorkspace_Save verifies Save() iterates over members without panic.
func TestWorkspace_Save(t *testing.T) {
	require := test_util.NewRequire(t)

	absPath, err := filepath.Abs(filepath.Join("testdata", "workspace-simple"))
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	ws := result.Project().(*projects.WorkspaceProject)
	ws.Save()
}

// TestWorkspace_Duplicate verifies Duplicate() deep-copies the workspace.
func TestWorkspace_Duplicate(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	absPath, err := filepath.Abs(filepath.Join("testdata", "workspace-simple"))
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)

	original := result.Project().(*projects.WorkspaceProject)
	dup := original.Duplicate().(*projects.WorkspaceProject)

	assert.NotSame(original, dup)
	assert.Equal(original.Kind(), dup.Kind())
	assert.Equal(len(original.Projects()), len(dup.Projects()))
	assert.Equal(len(original.Manifest().Packages()), len(dup.Manifest().Packages()))

	for i := range original.Projects() {
		assert.NotSame(original.Projects()[i], dup.Projects()[i])
	}
}

// TestWorkspace_CircularDependency verifies cycle detection in WorkspaceResolution.
func TestWorkspace_CircularDependency(t *testing.T) {
	require := test_util.NewRequire(t)
	assert := test_util.New(t)

	dir := t.TempDir()
	writeFile(t, dir, "Ballerina.toml", `[workspace]
packages = ["pkg-x", "pkg-y"]
`)
	writeFile(t, dir, "pkg-x/Ballerina.toml", `[package]
org = "testorg"
name = "pkgx"
version = "1.0.0"
`)
	writeFile(t, dir, "pkg-x/main.bal", `import testorg/pkgy;
public function main() { pkgy:hello(); }
`)
	writeFile(t, dir, "pkg-y/Ballerina.toml", `[package]
org = "testorg"
name = "pkgy"
version = "1.0.0"
`)
	writeFile(t, dir, "pkg-y/lib.bal", `import testorg/pkgx;
public function hello() { pkgx:main(); }
`)

	result, err := loadProject(dir)
	require.NoError(err)

	ws := result.Project().(*projects.WorkspaceProject)
	resolution := ws.Resolution()
	require.NotNil(resolution)

	resDiags := resolution.DiagnosticResult()
	assert.True(resDiags.HasErrors(),
		"expected resolution to report an error for the workspace cycle")
	if !containsDiagnosticMentioning(resDiags, "circular") {
		t.Errorf("expected a diagnostic mentioning 'circular', got: %v",
			diagnosticMessages(resDiags))
	}
}

// --- helpers ---

func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func containsDiagnosticMentioning(diags projects.DiagnosticResult, needle string) bool {
	for _, d := range diags.Diagnostics() {
		if strings.Contains(d.Message(), needle) {
			return true
		}
	}
	return false
}

func diagnosticMessages(diags projects.DiagnosticResult) []string {
	var msgs []string
	for _, d := range diags.Diagnostics() {
		msgs = append(msgs, d.Message())
	}
	return msgs
}
