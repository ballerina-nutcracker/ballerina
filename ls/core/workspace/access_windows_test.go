// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package workspace

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/projects"
)

func TestWindowsNativeURIRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "Case Preserved #.bal")
	lower := strings.ToLower(file[:1]) + file[1:]
	u, err := FileURIFromPath(lower)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.ToUpper(file[:1]) + file[1:]
	if u.Path() != want || !strings.Contains(u.String(), "Case%20Preserved%20%23.bal") {
		t.Fatalf("URI=%v path=%q", u, u.Path())
	}
	back, err := NewFileURI(u.String())
	if err != nil || back != u {
		t.Fatalf("round trip=%v, %v", back, err)
	}
	upper, err := FileURIFromPath(want)
	if err != nil || upper != u {
		t.Fatalf("drive identity=%v, %v", upper, err)
	}
	for _, invalid := range []string{`C:app.bal`, `\\server\share\main.bal`, `\current-drive\main.bal`} {
		if _, err := FileURIFromPath(invalid); err == nil {
			t.Errorf("accepted %q", invalid)
		}
	}
}

func TestWindowsDriveRootPathsAndDiscovery(t *testing.T) {
	t.Parallel()
	fixture := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixture, "Ballerina.toml"), []byte("[package]\norg=\"test\"\nname=\"root\"\nversion=\"0.1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	driveRoot := strings.ToUpper(filepath.VolumeName(fixture)) + `\`
	file := filepath.Join(driveRoot, "main.bal")
	openDir := func(dir string) fs.FS {
		if dir != driveRoot {
			t.Fatalf("unexpected directory %q", dir)
		}
		return os.DirFS(fixture)
	}
	root, found, err := projects.FindProjectRoot(openDir, file)
	if err != nil || !found || root != driveRoot {
		t.Fatalf("discovery=%q, %v, %v", root, found, err)
	}
	paths := SourcePaths{root: driveRoot}
	u, err := FileURIFromPath(file)
	if err != nil {
		t.Fatal(err)
	}
	name, ok := paths.NameOf(u)
	if !ok || name != "main.bal" {
		t.Fatalf("name=%q, %v", name, ok)
	}
	if back, ok := paths.URIOf(name); !ok || back != u {
		t.Fatalf("reverse=%v, %v", back, ok)
	}
}

func TestWindowsNestedModulesAndSiblingPrefixes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	bus := event.New()
	defer bus.Close()
	service := New(os.DirFS, bus)
	uris := make(map[string]DocumentURI)
	for _, folder := range []string{"shop", "shop2"} {
		dir := filepath.Join(root, folder)
		nested := filepath.Join(dir, "modules", "nested")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := "[package]\norg=\"test\"\nname=\"" + folder + "\"\nversion=\"0.1.0\"\n"
		if err := os.WriteFile(filepath.Join(dir, "Ballerina.toml"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(nested, "main.bal")
		if err := os.WriteFile(file, []byte("public function main() {}"), 0o644); err != nil {
			t.Fatal(err)
		}
		uri, err := FileURIFromPath(file)
		if err != nil {
			t.Fatal(err)
		}
		uris[folder] = uri
		if _, err := service.Apply(context.Background(), DocumentChange{Kind: ChangeOpen, URI: uri, Text: "public function main() { }", Version: 1}); err != nil {
			t.Fatal(err)
		}
		key, err := service.ProjectKey(uri)
		if err != nil || key != strings.ToUpper(dir[:1])+dir[1:] {
			t.Fatalf("key=%q, %v", key, err)
		}
		project, err := service.Project(uri)
		if err != nil || project == nil {
			t.Fatalf("Project=%v, %v", project, err)
		}
		name, ok := service.Paths(project).NameOf(uri)
		if !ok || name != "modules/nested/main.bal" {
			t.Fatalf("name=%q, %v", name, ok)
		}
		if _, ok := project.DocumentID(name); !ok {
			t.Fatalf("document missing for %q", name)
		}
	}
	project, err := service.Project(uris["shop"])
	if err != nil {
		t.Fatal(err)
	}
	paths := service.Paths(project)
	if _, ok := paths.NameOf(uris["shop2"]); ok {
		t.Fatal("sibling-prefix path admitted")
	}
	otherDrive := "Z"
	if strings.EqualFold(filepath.VolumeName(root), "Z:") {
		otherDrive = "Y"
	}
	other, err := FileURIFromPath(otherDrive + `:\main.bal`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := paths.NameOf(other); ok {
		t.Fatal("different-drive path admitted")
	}
}
