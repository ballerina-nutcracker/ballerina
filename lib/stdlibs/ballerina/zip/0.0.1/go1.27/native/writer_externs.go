// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"fmt"
	"io"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func writerOf(self *values.Object) *zipWriter {
	v, _ := self.Get("$writer")
	w, _ := v.(*zipWriter)
	return w
}

// openWriterOf returns the writer of an ArchiveWriter, or the error a closed one gives.
func openWriterOf(ctx *extern.Context, self *values.Object) (*zipWriter, values.BalValue, error) {
	w := writerOf(self)
	if w.closed {
		err, ierr := zipError(ctx, invalidArchive, "cannot write to the archive: it is closed")
		return nil, err, ierr
	}
	return w, nil, nil
}

func (e *zipExterns) create(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	path, _ := args[1].(string)
	level, _ := args[2].(string)
	overwrite, _ := args[3].(bool)
	if !representable(path) {
		return invalidPathError(ctx, path)
	}
	fs := e.rt.Platform().FS
	if info, err := fs.Stat(path); err == nil && info.IsDir {
		return zipError(ctx, fileSystem,
			fmt.Sprintf("cannot create the archive '%s': a directory already exists at that path", path))
	}
	file, err := fs.CreateSeekable(path, !overwrite)
	if isExist(err) {
		return zipError(ctx, fileSystem,
			fmt.Sprintf("cannot create the archive '%s': a file already exists there; set 'overwrite' to replace it", path))
	}
	if err != nil {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot create the archive '%s': the file could not be created", path))
	}
	self.Put("$writer", newZipWriter(file, level))
	return nil, nil
}

func (e *zipExterns) addFile(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	name, _ := args[1].(string)
	source, _ := args[2].(string)
	w, closedErr, err := openWriterOf(ctx, self)
	if w == nil {
		return closedErr, err
	}
	if !representable(source) {
		return invalidPathError(ctx, source)
	}
	fs := e.rt.Platform().FS
	info, err := fs.Stat(source)
	if err != nil {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot add file '%s': no file exists at that path", source))
	}
	if err := writeFile(w, fs.OpenReadable, name, source, info); err != nil {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot add file '%s' to the archive", source))
	}
	return nil, nil
}

// writeFile writes a file, or a directory entry when the name ends in `/`. The entry is finished whatever
// the copy does, so that a file that could not be read costs one entry rather than the whole archive.
func writeFile(w *zipWriter, open func(string) (io.ReadCloser, error), name, source string,
	info *pal.FileInfo) error {
	entry := fileEntry(name, info, w.method)
	if isDirectoryName(name) {
		if err := w.startEntry(entry, true); err != nil {
			return err
		}
		return w.finishEntry()
	}
	if err := w.startEntry(entry, info.Size < zip64Limit); err != nil {
		return err
	}
	copyErr := copyFile(w, open, source)
	if err := w.finishEntry(); copyErr == nil {
		copyErr = err
	}
	return copyErr
}

func copyFile(w *zipWriter, open func(string) (io.ReadCloser, error), source string) error {
	in, err := open(source)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	buf := make([]byte, copyBufferSize)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			if werr := w.write(buf[:n]); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (e *zipExterns) startEntry(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	name, _ := args[1].(string)
	w, closedErr, err := openWriterOf(ctx, self)
	if w == nil {
		return closedErr, err
	}
	entry := contentEntry(name, e.rt.Platform().Time.Now(), w.method)
	if err := w.startEntry(entry, isDirectoryName(name)); err != nil {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot add entry '%s' to the archive", name))
	}
	return nil, nil
}

func (e *zipExterns) writeChunk(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	content, _ := args[1].(*values.List)
	w, closedErr, err := openWriterOf(ctx, self)
	if w == nil {
		return closedErr, err
	}
	if err := w.write(content.ToByteSlice()); err != nil {
		return zipError(ctx, fileSystem, "cannot write the content of the entry to the archive")
	}
	return nil, nil
}

func (e *zipExterns) finishEntry(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	w, closedErr, err := openWriterOf(ctx, self)
	if w == nil {
		return closedErr, err
	}
	if err := w.finishEntry(); err != nil {
		return zipError(ctx, fileSystem, "cannot complete the entry")
	}
	return nil, nil
}

func (e *zipExterns) copyEntry(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	reader, _ := args[1].(*values.Object)
	name, _ := args[2].(string)
	w, closedErr, err := openWriterOf(ctx, self)
	if w == nil {
		return closedErr, err
	}
	source := archiveOf(reader)
	if source.isClosed() {
		return zipError(ctx, invalidArchive, fmt.Sprintf("cannot copy entry '%s': the source archive is closed", name))
	}
	index := source.indexOf(name)
	if index < 0 {
		return zipError(ctx, entryNotFound, fmt.Sprintf("the source archive contains no entry named '%s'", name))
	}
	entry := source.entries[index]
	if entry.encrypted() {
		return entryError(ctx, unsupportedEntry,
			fmt.Sprintf("cannot copy entry '%s': encrypted entries cannot be written into another archive", name), name)
	}
	if err := w.copyRaw(copiedEntry(entry), source.rawContent(entry)); err != nil {
		return zipError(ctx, invalidArchive, fmt.Sprintf("cannot copy entry '%s' from the source archive", name))
	}
	return nil, nil
}

func (e *zipExterns) closeWriter(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if err := writerOf(self).close(); err != nil {
		return zipError(ctx, fileSystem, "cannot complete the archive")
	}
	return nil, nil
}
