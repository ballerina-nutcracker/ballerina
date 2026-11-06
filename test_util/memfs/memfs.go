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
	"io/fs"
	"path"
	"strings"
	"time"
)

// memFS is an in-memory filesystem that supports both files and directories.
// It implements fs.FS, fs.ReadDirFS, bfs.MutableFS, and bfs.WritableFS interfaces.
type memFS struct {
	entries map[string]*memEntry
}

// memEntry represents either a file or directory in the in-memory filesystem.
type memEntry struct {
	name    string
	data    []byte
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

// memFileHandle is an open handle to a memEntry, implementing fs.File.
type memFileHandle struct {
	entry  *memEntry
	reader *bytes.Reader
}

// memDirHandle is an open handle to a directory for reading entries.
type memDirHandle struct {
	entry   *memEntry
	entries []fs.DirEntry
	offset  int
}

func New() *memFS {
	return &memFS{
		entries: make(map[string]*memEntry),
	}
}

// mkdirAllInternal creates all directories in the path if they don't exist.
func (mfs *memFS) mkdirAllInternal(dirPath string, perm fs.FileMode) {
	if dirPath == "" {
		panic("dirPath cannot be empty")
	}

	if perm.Perm() != 0o755 {
		panic("unsupported FileMode: only 0o755 is supported")
	}

	if dirPath == "." {
		return
	}
	parts := strings.Split(dirPath, "/")
	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if current == "" {
			current = part
		} else {
			current = current + "/" + part
		}
		if _, exists := mfs.entries[current]; !exists {
			mfs.entries[current] = &memEntry{
				name:    path.Base(current),
				mode:    perm | fs.ModeDir,
				modTime: time.Now(),
				isDir:   true,
			}
		}
	}
}

func (mfs *memFS) Create(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrInvalid}
	}

	// Create parent directories
	if dir := path.Dir(name); dir != "." {
		mfs.mkdirAllInternal(dir, 0o755)
	}

	entry := &memEntry{
		name:    path.Base(name),
		mode:    0o644,
		modTime: time.Now(),
		isDir:   false,
	}
	mfs.entries[name] = entry

	return &memFileHandle{
		entry:  entry,
		reader: bytes.NewReader(entry.data),
	}, nil
}

func (mfs *memFS) MkdirAll(dirPath string, perm fs.FileMode) error {
	if !fs.ValidPath(dirPath) {
		return &fs.PathError{Op: "mkdir", Path: dirPath, Err: fs.ErrInvalid}
	}

	mfs.mkdirAllInternal(dirPath, perm)
	return nil
}

func (mfs *memFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	// Handle root directory
	if name == "." {
		return &memDirHandle{
			entry: &memEntry{
				name:  ".",
				mode:  fs.ModeDir | 0o755,
				isDir: true,
			},
			entries: mfs.readDirEntries("."),
		}, nil
	}

	entry, ok := mfs.entries[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	if entry.isDir {
		return &memDirHandle{
			entry:   entry,
			entries: mfs.readDirEntries(name),
		}, nil
	}

	return &memFileHandle{
		entry:  entry,
		reader: bytes.NewReader(entry.data),
	}, nil
}

func (mfs *memFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "openfile", Path: name, Err: fs.ErrInvalid}
	}

	entry, ok := mfs.entries[name]
	if !ok {
		// Create parent directories
		if dir := path.Dir(name); dir != "." {
			mfs.mkdirAllInternal(dir, 0o755)
		}

		entry = &memEntry{
			name:    path.Base(name),
			mode:    perm,
			modTime: time.Now(),
			isDir:   false,
		}
		mfs.entries[name] = entry
	}

	return &memFileHandle{
		entry:  entry,
		reader: bytes.NewReader(entry.data),
	}, nil
}

func (mfs *memFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	// For root directory
	if name == "." {
		return mfs.readDirEntries("."), nil
	}

	entry, ok := mfs.entries[name]
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	if !entry.isDir {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}

	return mfs.readDirEntries(name), nil
}

// readDirEntries returns direct children of a directory.
func (mfs *memFS) readDirEntries(dirPath string) []fs.DirEntry {
	var prefix string
	if dirPath == "." {
		prefix = ""
	} else {
		prefix = dirPath + "/"
	}

	seen := make(map[string]bool)
	var entries []fs.DirEntry

	for name := range mfs.entries {
		var childName string
		if prefix == "" {
			childName = name
		} else if after, ok := strings.CutPrefix(name, prefix); ok {
			childName = after
		} else {
			continue
		}

		// Get only direct children (no nested paths)
		if idx := strings.Index(childName, "/"); idx != -1 {
			childName = childName[:idx]
		}

		if childName == "" || seen[childName] {
			continue
		}
		seen[childName] = true

		// Check if this child is a directory or file
		fullPath := childName
		if prefix != "" {
			fullPath = prefix + childName
		}

		if e, exists := mfs.entries[fullPath]; exists {
			entries = append(entries, &dirEntry{entry: e})
		} else {
			// It's an implicit directory (exists only as part of a path)
			entries = append(entries, &dirEntry{
				entry: &memEntry{
					name:  childName,
					mode:  fs.ModeDir | 0o755,
					isDir: true,
				},
			})
		}
	}

	return entries
}

// Remove removes a file or directory and all its contents.
func (mfs *memFS) Remove(name string) error {
	removed := false

	// Remove the entry itself if it exists
	if _, exists := mfs.entries[name]; exists {
		delete(mfs.entries, name)
		removed = true
	}

	// Also remove all children (for directories)
	prefix := name + "/"
	for entryName := range mfs.entries {
		if strings.HasPrefix(entryName, prefix) {
			delete(mfs.entries, entryName)
			removed = true
		}
	}

	if !removed {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}

	return nil
}

// Move moves a file or directory from oldpath to newpath.
func (mfs *memFS) Move(oldpath, newpath string) error {
	type moveItem struct {
		oldName string
		newName string
		entry   *memEntry
	}

	var toMove []moveItem

	// Check for exact match (file or directory entry)
	if entry, exists := mfs.entries[oldpath]; exists {
		toMove = append(toMove, moveItem{
			oldName: oldpath,
			newName: newpath,
			entry:   entry,
		})
	}

	// Also move all children (for directories)
	oldPrefix := oldpath + "/"
	newPrefix := newpath + "/"

	for name, entry := range mfs.entries {
		if suffix, ok := strings.CutPrefix(name, oldPrefix); ok {
			toMove = append(toMove, moveItem{
				oldName: name,
				newName: newPrefix + suffix,
				entry:   entry,
			})
		}
	}

	if len(toMove) == 0 {
		return &fs.PathError{Op: "move", Path: oldpath, Err: fs.ErrNotExist}
	}

	for _, item := range toMove {
		delete(mfs.entries, item.oldName)
		item.entry.name = path.Base(item.newName)
		mfs.entries[item.newName] = item.entry
	}

	return nil
}

// WriteFile writes data to a file, creating it if necessary.
func (mfs *memFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: "writefile", Path: name, Err: fs.ErrInvalid}
	}

	// Create parent directories
	if dir := path.Dir(name); dir != "." {
		mfs.mkdirAllInternal(dir, 0o755)
	}

	mfs.entries[name] = &memEntry{
		name:    path.Base(name),
		data:    data,
		mode:    perm,
		modTime: time.Now(),
		isDir:   false,
	}

	return nil
}

// memFileHandle implements fs.File for regular files

func (h *memFileHandle) Close() error {
	return nil
}

func (h *memFileHandle) Read(p []byte) (int, error) {
	return h.reader.Read(p)
}

func (h *memFileHandle) Stat() (fs.FileInfo, error) {
	return h.entry, nil
}

// memDirHandle implements fs.File and fs.ReadDirFile for directories

func (h *memDirHandle) Close() error {
	return nil
}

func (h *memDirHandle) Read(p []byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: h.entry.name, Err: fs.ErrInvalid}
}

func (h *memDirHandle) Stat() (fs.FileInfo, error) {
	return h.entry, nil
}

func (h *memDirHandle) ReadDir(n int) ([]fs.DirEntry, error) {
	if n <= 0 {
		entries := h.entries[h.offset:]
		h.offset = len(h.entries)
		return entries, nil
	}

	if h.offset >= len(h.entries) {
		return nil, nil
	}

	end := min(h.offset+n, len(h.entries))

	entries := h.entries[h.offset:end]
	h.offset = end
	return entries, nil
}

// memEntry implements fs.FileInfo

func (e *memEntry) Name() string {
	return e.name
}

func (e *memEntry) Size() int64 {
	return int64(len(e.data))
}

func (e *memEntry) Mode() fs.FileMode {
	return e.mode
}

func (e *memEntry) ModTime() time.Time {
	return e.modTime
}

func (e *memEntry) IsDir() bool {
	return e.isDir
}

func (e *memEntry) Sys() any {
	return nil
}

// dirEntry implements fs.DirEntry

type dirEntry struct {
	entry *memEntry
}

func (d *dirEntry) Name() string {
	return d.entry.name
}

func (d *dirEntry) IsDir() bool {
	return d.entry.isDir
}

func (d *dirEntry) Type() fs.FileMode {
	return d.entry.mode.Type()
}

func (d *dirEntry) Info() (fs.FileInfo, error) {
	return d.entry, nil
}
