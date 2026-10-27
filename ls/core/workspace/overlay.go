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
	"bytes"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

// overlayFS prefers open buffers. Listings retain only previously loaded open
// members, never arbitrary new buffer-only sources.
type overlayFS struct {
	base     fs.FS
	overlays map[string][]byte
	members  map[string]bool
	now      func() time.Time
}

func (f overlayFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if buf, ok := f.overlays[name]; ok {
		return &memFile{Reader: bytes.NewReader(buf), info: overlayFileInfo{name: path.Base(name), size: int64(len(buf)), modTime: f.now()}}, nil
	}
	file, err := f.base.Open(name)
	if errors.Is(err, fs.ErrNotExist) && f.retainedDirectory(name) {
		return &memFile{Reader: bytes.NewReader(nil), info: overlayFileInfo{name: path.Base(name), modTime: f.now(), directory: true}}, nil
	}
	return file, err
}

func (f overlayFS) retainedDirectory(name string) bool {
	for member := range f.members {
		if name == "." || strings.HasPrefix(member, name+"/") {
			return true
		}
	}
	return false
}

func (f overlayFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.base, name)
	if err != nil && (!errors.Is(err, fs.ErrNotExist) || !f.retainedDirectory(name)) {
		return nil, err
	}
	present := make(map[string]bool)
	for _, entry := range entries {
		present[entry.Name()] = true
	}
	prefix := ""
	if name != "." {
		prefix = name + "/"
	}
	for member := range f.members {
		if !strings.HasPrefix(member, prefix) {
			continue
		}
		rel := strings.TrimPrefix(member, prefix)
		child, _, directory := strings.Cut(rel, "/")
		if present[child] {
			continue
		}
		present[child] = true
		info := overlayFileInfo{name: child, modTime: f.now(), directory: directory}
		if !directory {
			info.size = int64(len(f.overlays[member]))
		}
		entries = append(entries, fs.FileInfoToDirEntry(info))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

type overlayFileInfo struct {
	name      string
	size      int64
	modTime   time.Time
	directory bool
}

func (i overlayFileInfo) Name() string { return i.name }
func (i overlayFileInfo) Size() int64  { return i.size }
func (i overlayFileInfo) Mode() fs.FileMode {
	if i.directory {
		return fs.ModeDir | 0o755
	}
	return 0o644
}
func (i overlayFileInfo) ModTime() time.Time { return i.modTime }
func (i overlayFileInfo) IsDir() bool        { return i.directory }
func (i overlayFileInfo) Sys() any           { return nil }

type memFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memFile) Close() error               { return nil }
