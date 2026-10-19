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
	"io/fs"
	"path"
	"time"
)

// overlayFS is a load-time, folder-relative view. Open buffers win, but
// directory listings deliberately contain only files in the supplied FS.
type overlayFS struct {
	base     fs.FS
	overlays map[string][]byte
	now      func() time.Time
}

func (f overlayFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if buf, ok := f.overlays[name]; ok {
		return &memFile{Reader: bytes.NewReader(buf), info: overlayFileInfo{name: path.Base(name), size: int64(len(buf)), modTime: f.now()}}, nil
	}
	return f.base.Open(name)
}

func (f overlayFS) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(f.base, name) }

type overlayFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (i overlayFileInfo) Name() string       { return i.name }
func (i overlayFileInfo) Size() int64        { return i.size }
func (i overlayFileInfo) Mode() fs.FileMode  { return 0o644 }
func (i overlayFileInfo) ModTime() time.Time { return i.modTime }
func (i overlayFileInfo) IsDir() bool        { return false }
func (i overlayFileInfo) Sys() any           { return nil }

type memFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memFile) Close() error               { return nil }
