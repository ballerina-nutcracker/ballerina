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

package workspace

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

// fakePAL builds a pal.FS whose Stat/ReadFile/ReadDir report a counter so
// tests can assert call counts without touching disk.
type fakePAL struct {
	statCalls int
	dir       map[string][]byte // file path → content (disk)
}

func (f *fakePAL) fs() pal.FS {
	return pal.FS{
		Stat: func(p string) (fs.FileInfo, error) {
			f.statCalls++
			if _, ok := f.dir[p]; ok {
				return fakeInfo{name: p, isDir: false}, nil
			}
			return nil, fs.ErrNotExist
		},
		ReadFile: func(p string) ([]byte, error) {
			if b, ok := f.dir[p]; ok {
				return b, nil
			}
			return nil, fs.ErrNotExist
		},
		ReadDir: func(p string) ([]fs.DirEntry, error) {
			return nil, fs.ErrNotExist
		},
	}
}

type fakeInfo struct {
	name  string
	isDir bool
}

func (i fakeInfo) Name() string       { return i.name }
func (i fakeInfo) Size() int64        { return 0 }
func (i fakeInfo) Mode() fs.FileMode  { return 0o644 }
func (i fakeInfo) ModTime() time.Time { return time.Time{} }
func (i fakeInfo) IsDir() bool        { return i.isDir }
func (i fakeInfo) Sys() any           { return nil }

func TestPalFSOverlayWinsOnReadFile(t *testing.T) {
	f := &fakePAL{dir: map[string][]byte{"/d/foo.bal": []byte("disk")}}
	pfs := palFS{pal: f.fs(), overlays: map[string][]byte{"/d/foo.bal": []byte("buffer")}, now: time.Now}
	got, err := pfs.ReadFile("/d/foo.bal")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "buffer" {
		t.Fatalf("ReadFile = %q, want overlay 'buffer'", got)
	}
	// Non-overlay file delegates to PAL.
	got, err = pfs.ReadFile("/d/other.bal")
	if err == nil || string(got) != "" {
		t.Fatalf("ReadFile non-overlay = %q err=%v, want ErrNotExist", got, err)
	}
}

func TestPalFSOverlayStatReturnsSynthetic(t *testing.T) {
	f := &fakePAL{dir: map[string][]byte{"/d/foo.bal": []byte("disk")}}
	pfs := palFS{pal: f.fs(), overlays: map[string][]byte{"/d/absent.bal": []byte("buffer")}, now: time.Now}
	info, err := pfs.Stat("/d/absent.bal")
	if err != nil {
		t.Fatalf("Stat overlay: %v", err)
	}
	if info.IsDir() {
		t.Fatal("overlay Stat should report a file, not a dir")
	}
	if info.Size() != int64(len("buffer")) {
		t.Fatalf("Size = %d, want %d", info.Size(), len("buffer"))
	}
	// Non-overlay delegates to PAL.
	if _, err := pfs.Stat("/d/foo.bal"); err != nil {
		t.Fatalf("Stat delegate: %v", err)
	}
}

func TestPalFSReadDirDelegatesToPAL(t *testing.T) {
	f := &fakePAL{dir: map[string][]byte{}}
	pfs := palFS{pal: f.fs(), overlays: map[string][]byte{"/d/foo.bal": []byte("buffer")}, now: time.Now}
	// ReadDir does not synthesize overlay entries; it delegates (returns the
	// PAL error here).
	if _, err := pfs.ReadDir("/d"); err == nil {
		t.Fatal("ReadDir expected delegation error, got nil")
	}
}

func TestResolveMemoizesSourceRootWalk(t *testing.T) {
	// A ProjectService whose Stat counts calls so we can assert the
	// ADR-048 walk is memoized: the second Apply for the same path does not
	// re-walk the filesystem looking for Ballerina.toml.
	f := &fakePAL{dir: map[string][]byte{
		"/w/main.bal": []byte("public function main() {}"),
	}}
	platform := pal.Platform{FS: pal.FS{
		ReadFile: func(p string) ([]byte, error) {
			if b, ok := f.dir[p]; ok {
				return b, nil
			}
			return nil, fs.ErrNotExist
		},
		WriteFile: func(p string, d []byte) error { return nil },
		Stat: func(p string) (fs.FileInfo, error) {
			f.statCalls++
			if _, ok := f.dir[p]; ok {
				return fakeInfo{name: p, isDir: false}, nil
			}
			return nil, fs.ErrNotExist
		},
		ReadDir:  func(p string) ([]fs.DirEntry, error) { return nil, fs.ErrNotExist },
		MkdirAll: func(p string, m fs.FileMode) error { return nil },
	}}
	svc := New(platform, event.New())
	u := fileURI(t, "file:///w/main.bal")
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeOpen, URI: u, Text: "public function main() {}", Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	first := f.statCalls
	if _, err := svc.Apply(context.Background(), DocumentChange{
		Kind: ChangeUpdate, URI: u, Text: "public function main() {}", Version: 2,
	}); err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	second := f.statCalls
	// Under ticket 09's modifier-chain publication model a content update does
	// NOT reload: it reuses the persistent project via Document.Modify().Apply(),
	// so the ADR-048 walk is memoized AND no loadProject Ballerina.toml probe
	// runs. The stat delta is 0 (vs 08's 1-per-publish reload). If the walk had
	// repeated, the delta would be >= 3.
	if delta := second - first; delta != 0 {
		t.Fatalf("stat delta after second Apply = %d, want 0 (modifier chain must not reload on update)", delta)
	}
}
