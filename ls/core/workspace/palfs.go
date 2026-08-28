// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 ( the "License"); you may not use this file except
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
	"io"
	"io/fs"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

// palFS is the overlay-augmented io/fs.FS the workspace builds at load time.
// It implements fs.FS, fs.StatFS, fs.ReadFileFS, and fs.ReadDirFS — the
// interfaces projects/config_creator.go consults. The open-buffer overlay wins
// on ReadFile and Stat (gopls-style); ReadDir delegates to the PAL filesystem
// (the overlay does not synthesize directory entries).
//
// palFS is only needed at load/reload: projects.Load reads source through it,
// then BaseProject discards the fsys. Ongoing edits flow through the modifier
// chain, not palFS.
type palFS struct {
	pal      pal.FS
	overlays map[string][]byte // absolute file path → unsaved buffer content
	now      func() time.Time
}

// Open implements fs.FS. Overlay files return an in-memory reader; every other
// path is served from PAL via Stat + ReadFile. Directory paths return a handle
// whose Stat reports IsDir so fs.Sub works.
func (f palFS) Open(name string) (fs.File, error) {
	if buf, ok := f.overlays[name]; ok {
		return &memFile{info: f.overlayInfo(name, buf), reader: newByteReader(buf)}, nil
	}
	info, err := f.pal.Stat(name)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	if info.IsDir() {
		return &memFile{info: info}, nil
	}
	data, err := f.pal.ReadFile(name)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	return &memFile{info: info, reader: newByteReader(data)}, nil
}

// Stat implements fs.StatFS. Overlay paths return a synthetic FileInfo; all
// others delegate to PAL.
func (f palFS) Stat(name string) (fs.FileInfo, error) {
	if buf, ok := f.overlays[name]; ok {
		return f.overlayInfo(name, buf), nil
	}
	return f.pal.Stat(name)
}

// ReadFile implements fs.ReadFileFS. Overlay paths return the buffer; all
// others delegate to PAL.
func (f palFS) ReadFile(name string) ([]byte, error) {
	if buf, ok := f.overlays[name]; ok {
		return buf, nil
	}
	return f.pal.ReadFile(name)
}

// ReadDir implements fs.ReadDirFS. Delegates to PAL — the overlay never
// synthesizes directory entries.
func (f palFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return f.pal.ReadDir(name)
}

func (f palFS) overlayInfo(name string, buf []byte) fs.FileInfo {
	return overlayFileInfo{name: name, size: int64(len(buf)), modTime: f.now()}
}

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
	info   fs.FileInfo
	reader *byteReader
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memFile) Read(p []byte) (int, error) {
	if f.reader == nil {
		return 0, io.EOF
	}
	return f.reader.read(p)
}
func (f *memFile) Close() error { return nil }

type byteReader struct {
	data []byte
	off  int
}

func newByteReader(data []byte) *byteReader { return &byteReader{data: data} }

func (r *byteReader) read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}
