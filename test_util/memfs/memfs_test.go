// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package memfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/common/bfs"
)

func TestWriteFileOpenAndStat(t *testing.T) {
	fsys := New()

	name := "dir/hello.txt"
	content := []byte("hello world")
	perm := fs.FileMode(0o644)

	if err := bfs.WriteFile(fsys, name, content, perm); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	f, err := fsys.Open(name)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("content mismatch: got %q want %q", string(data), string(content))
	}

	info, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Size() != int64(len(content)) {
		t.Errorf("Size mismatch: got %d want %d", info.Size(), len(content))
	}
	if info.Mode() != perm {
		t.Errorf("Mode mismatch: got %v want %v", info.Mode(), perm)
	}
	if info.IsDir() {
		t.Errorf("file unexpectedly marked as directory")
	}
}

func TestCreateAndOpen_InvalidPath(t *testing.T) {
	fsys := New()
	invalid := "/abs/path.txt"

	if _, err := bfs.Create(fsys, invalid); err == nil || !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("Create: expected ErrInvalid, got %v", err)
	}
	if _, err := fsys.Open(invalid); err == nil || !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("Open: expected ErrInvalid, got %v", err)
	}
	if _, err := bfs.OpenFile(fsys, invalid, 0, 0o644); err == nil || !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("OpenFile: expected ErrInvalid, got %v", err)
	}
}

func TestOpen_NotExist(t *testing.T) {
	fsys := New()
	if _, err := fsys.Open("no/such/file.txt"); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestRemove_FileAndDirectoryPrefix(t *testing.T) {
	fsys := New()

	_ = bfs.WriteFile(fsys, "dir/a.txt", []byte("A"), 0o644)
	_ = bfs.WriteFile(fsys, "dir/sub/b.txt", []byte("B"), 0o644)
	_ = bfs.WriteFile(fsys, "c.txt", []byte("C"), 0o644)

	if err := bfs.Remove(fsys, "c.txt"); err != nil {
		t.Fatalf("Remove file error: %v", err)
	}
	if _, err := fsys.Open("c.txt"); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, got err: %v", err)
	}

	if err := bfs.Remove(fsys, "dir"); err != nil {
		t.Fatalf("Remove dir prefix error: %v", err)
	}
	if _, err := fsys.Open("dir/a.txt"); !os.IsNotExist(err) {
		t.Fatalf("dir/a.txt should be removed, got err: %v", err)
	}
	if _, err := fsys.Open("dir/sub/b.txt"); !os.IsNotExist(err) {
		t.Fatalf("dir/sub/b.txt should be removed, got err: %v", err)
	}

	if err := bfs.Remove(fsys, "does/not/exist"); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected ErrNotExist when removing non-existent, got %v", err)
	}
}

func TestMove_FileAndDirectory(t *testing.T) {
	fsys := New()

	_ = bfs.WriteFile(fsys, "a.txt", []byte("alpha"), 0o644)
	if err := bfs.Move(fsys, "a.txt", "b.txt"); err != nil {
		t.Fatalf("Move file error: %v", err)
	}
	if _, err := fsys.Open("a.txt"); !os.IsNotExist(err) {
		t.Fatalf("source should be gone after move, err: %v", err)
	}
	bf, err := fsys.Open("b.txt")
	if err != nil {
		t.Fatalf("destination open error: %v", err)
	}
	got, _ := io.ReadAll(bf)
	_ = bf.Close()
	if string(got) != "alpha" {
		t.Fatalf("moved file content mismatch: got %q", string(got))
	}

	_ = bfs.WriteFile(fsys, "src/x.txt", []byte("x"), 0o644)
	_ = bfs.WriteFile(fsys, "src/sub/y.txt", []byte("y"), 0o644)
	if err := bfs.Move(fsys, "src", "dst"); err != nil {
		t.Fatalf("Move dir error: %v", err)
	}
	if _, err := fsys.Open("src/x.txt"); !os.IsNotExist(err) {
		t.Fatalf("old prefixed file should not exist, err: %v", err)
	}
	if f, err := fsys.Open("dst/x.txt"); err != nil {
		t.Fatalf("dst/x.txt open error: %v", err)
	} else {
		_ = f.Close()
	}
	if f, err := fsys.Open("dst/sub/y.txt"); err != nil {
		t.Fatalf("dst/sub/y.txt open error: %v", err)
	} else {
		_ = f.Close()
	}

	if err := bfs.Move(fsys, "nope", "new"); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected ErrNotExist for move, got %v", err)
	}
}

func TestMkdirAll(t *testing.T) {
	fsys := New()

	// Test creating nested directories
	if err := bfs.MkdirAll(fsys, "a/b/c", 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	// Verify all directories were created
	for _, dir := range []string{"a", "a/b", "a/b/c"} {
		f, err := fsys.Open(dir)
		if err != nil {
			t.Fatalf("Open %s error: %v", dir, err)
		}
		info, err := f.Stat()
		if err != nil {
			t.Fatalf("Stat %s error: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", dir)
		}
		if info.Mode()&fs.ModeDir == 0 {
			t.Errorf("%s mode should have ModeDir set", dir)
		}
		_ = f.Close()
	}

	// Test creating directory that already exists (should not error)
	if err := bfs.MkdirAll(fsys, "a/b", 0o755); err != nil {
		t.Fatalf("MkdirAll on existing dir error: %v", err)
	}

	// Test invalid path
	if err := bfs.MkdirAll(fsys, "/absolute/path", 0o755); err == nil || !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("MkdirAll: expected ErrInvalid for absolute path, got %v", err)
	}

	// Test creating single directory
	if err := bfs.MkdirAll(fsys, "single", 0o755); err != nil {
		t.Fatalf("MkdirAll single dir error: %v", err)
	}
	f, err := fsys.Open("single")
	if err != nil {
		t.Fatalf("Open single error: %v", err)
	}
	info, _ := f.Stat()
	if !info.IsDir() {
		t.Errorf("single should be a directory")
	}
	_ = f.Close()
}

func TestMkdirAll_ReadDir(t *testing.T) {
	fsys := New()

	// Create directories and files
	if err := bfs.MkdirAll(fsys, "root/sub1/deep", 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := bfs.MkdirAll(fsys, "root/sub2", 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := bfs.WriteFile(fsys, "root/file.txt", []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Read root directory
	entries, err := fs.ReadDir(fsys, "root")
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	entryMap := make(map[string]fs.DirEntry)
	for _, e := range entries {
		entryMap[e.Name()] = e
	}

	for _, name := range []string{"file.txt", "sub1", "sub2"} {
		if _, ok := entryMap[name]; !ok {
			t.Errorf("expected entry %s not found", name)
		}
	}

	if entryMap["file.txt"].IsDir() {
		t.Errorf("file.txt should not be a directory")
	}
	if !entryMap["sub1"].IsDir() {
		t.Errorf("sub1 should be a directory")
	}
	if !entryMap["sub2"].IsDir() {
		t.Errorf("sub2 should be a directory")
	}
}
