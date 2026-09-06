// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver_test

import (
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/parser"
)

type readTrackingFS struct {
	fs.FS
	mu    sync.Mutex
	reads map[string]int
	opens map[string]int
}

func (f *readTrackingFS) Open(name string) (fs.File, error) {
	f.mu.Lock()
	f.opens[name]++
	f.mu.Unlock()
	return f.FS.Open(name)
}

func (f *readTrackingFS) ReadFile(name string) ([]byte, error) {
	f.mu.Lock()
	f.reads[name]++
	f.mu.Unlock()
	return fs.ReadFile(f.FS, name)
}

func (f *readTrackingFS) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads[name]
}

func (f *readTrackingFS) openCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opens[name]
}

func TestDirectoryDiscoveryUsesCallerBasenameWithoutWorkspaceScan(t *testing.T) {
	fsys := &readTrackingFS{FS: fstest.MapFS{
		"app/Ballerina.toml": &fstest.MapFile{Data: []byte("[package]\norg = \"testorg\"\nversion = \"1.0.0\"\n")},
		"app/main.bal":       &fstest.MapFile{Data: []byte("public function main() {}")},
		"Workspace.toml":     &fstest.MapFile{Data: []byte("this must not be read")},
	}, reads: make(map[string]int), opens: make(map[string]int)}
	cx := newDriverContext()
	sources, err := driver.FindSources(cx, fsys, "app", "caller-name")
	if err != nil {
		t.Fatal(err)
	}
	if sources.Descriptor.Name != "caller-name" {
		t.Fatalf("package name = %q, want caller basename", sources.Descriptor.Name)
	}
	if reads := fsys.count("Workspace.toml"); reads != 0 {
		t.Fatalf("workspace metadata reads = %d, want 0", reads)
	}
	if opens := fsys.openCount("Workspace.toml"); opens != 0 {
		t.Fatalf("workspace metadata opens = %d, want 0", opens)
	}
}

func TestParentedSingleFileIgnoresPackageDirNameAndReadsOnce(t *testing.T) {
	fsys := &readTrackingFS{FS: fstest.MapFS{
		"src/main.bal": &fstest.MapFile{Data: []byte("public function main() {}")},
	}, reads: make(map[string]int), opens: make(map[string]int)}
	cx := newDriverContext()
	sources, err := driver.FindSources(cx, fsys, "src/main.bal", "ignored-name")
	if err != nil {
		t.Fatal(err)
	}
	if sources.Descriptor.Name != "main" {
		t.Fatalf("single-file package name = %q, want file stem", sources.Descriptor.Name)
	}
	parsed, err := driver.Parse(cx, fsys, "src", sources, parser.DebugOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Modules[0].Documents[0].SourcePath; got != "main.bal" {
		t.Fatalf("SourcePath = %q, want main.bal", got)
	}
	if reads := fsys.count("src/main.bal"); reads != 1 {
		t.Fatalf("single-file reads = %d, want exactly 1", reads)
	}
}
