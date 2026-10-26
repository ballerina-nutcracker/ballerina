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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
)

func TestOverlayFS(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "disk.bal"), []byte("disk"), 0o644); err != nil {
		t.Fatal(err)
	}
	view := overlayFS{base: os.DirFS(root), overlays: map[string][]byte{"disk.bal": []byte("buffer"), "new.bal": []byte("new")}, now: time.Now}
	for name, want := range map[string]string{"disk.bal": "buffer", "new.bal": "new"} {
		got, err := fs.ReadFile(view, name)
		if err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v", name, got, err)
		}
		info, err := fs.Stat(view, name)
		if err != nil || info.Size() != int64(len(want)) {
			t.Fatalf("%s: %v, %v", name, info, err)
		}
	}
	entries, err := fs.ReadDir(view, ".")
	if err != nil || len(entries) != 1 || entries[0].Name() != "disk.bal" {
		t.Fatalf("ReadDir: %v, %v", entries, err)
	}
	if _, err := view.Open("../disk.bal"); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("invalid path: %v", err)
	}
}

type failingSourceFS struct{ err error }

func (f failingSourceFS) Open(string) (fs.File, error) { return nil, f.err }

func TestDiscoveryFailureRetainsBufferWithoutPublication(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	bus := event.New()
	defer bus.Close()
	failure := false
	opener := func(dir string) fs.FS {
		if failure && dir == root {
			return failingSourceFS{fs.ErrPermission}
		}
		return os.DirFS(dir)
	}
	svc := New(opener, bus)
	u, err := FileURIFromPath(filepath.Join(root, "main.bal"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeOpen, URI: u, Text: "public function main() {}", Version: 1}); err != nil {
		t.Fatal(err)
	}
	oldKey, err := svc.ProjectKey(u)
	if err != nil {
		t.Fatal(err)
	}
	failure = true
	manifest, err := FileURIFromPath(filepath.Join(root, "Ballerina.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ApplyWatchedFile(context.Background(), WatchedFileChange{URI: manifest, Kind: WatchedFileCreate}); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("watch error=%v", err)
	}
	latest := "public function main() { int latest = 1; }"
	if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeUpdate, URI: u, Text: latest, Version: 2}); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Apply error=%v", err)
	}
	snap, ok := svc.Snapshot(u)
	if !ok || snap.Text != latest || snap.Version != 2 || snap.SourceRoot != "" || snap.Generation != 0 {
		t.Fatalf("snapshot=%+v, %v", snap, ok)
	}
	if project, _, ok := svc.CurrentProject(oldKey); ok || project != nil {
		t.Fatal("stale project remains current")
	}
	if project, err := svc.Project(u); project != nil || !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Project=%v, %v", project, err)
	}
	failure = false
	if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeUpdate, URI: u, Text: latest, Version: 3}); err != nil {
		t.Fatal(err)
	}
	if project, err := svc.Project(u); err != nil || project == nil {
		t.Fatalf("retry Project=%v, %v", project, err)
	}
}

func TestLoadedChangeDoesNotOpenDirectories(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	calls := 0
	opener := func(dir string) fs.FS { calls++; return os.DirFS(dir) }
	bus := event.New()
	defer bus.Close()
	svc := New(opener, bus)
	u, err := FileURIFromPath(filepath.Join(root, "main.bal"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeOpen, URI: u, Text: "public function main() {}", Version: 1}); err != nil {
		t.Fatal(err)
	}
	before := calls
	if _, err := svc.Apply(context.Background(), DocumentChange{Kind: ChangeUpdate, URI: u, Text: "public function main() { }", Version: 2}); err != nil {
		t.Fatal(err)
	}
	if calls != before {
		t.Fatalf("change made %d further opener calls", calls-before)
	}
}
