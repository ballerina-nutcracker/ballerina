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
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	// chunkSize is how much of an entry one element of the stream from readEntry holds.
	chunkSize = 65536
	// copyBufferSize is how much of an entry extraction reads and judges against the limits at a time.
	copyBufferSize = 8192

	bytesExceeded int64 = -1
	ratioExceeded int64 = -2
)

func archiveOf(self *values.Object) *archive {
	v, _ := self.Get("$archive")
	a, _ := v.(*archive)
	return a
}

// openArchiveOf returns the archive of a reader, or the error a closed one gives.
func openArchiveOf(ctx *extern.Context, self *values.Object) (*archive, values.BalValue, error) {
	a := archiveOf(self)
	if a.isClosed() {
		err, ierr := zipError(ctx, invalidArchive, "cannot read the archive: it is closed")
		return nil, err, ierr
	}
	return a, nil, nil
}

func (e *zipExterns) open(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	path, _ := args[1].(string)
	fs := e.rt.Platform().FS
	info, err := fs.Stat(path)
	if err != nil {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot open the archive '%s': no file exists at that path", path))
	}
	if info.IsDir {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot open the archive '%s': it is a directory", path))
	}
	if !info.IsReadable {
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot open the archive '%s': the file cannot be read", path))
	}
	notValid := fmt.Sprintf("cannot open the archive '%s': it is not a valid ZIP archive", path)
	file, err := fs.OpenRandomAccess(path)
	if err != nil {
		return zipError(ctx, invalidArchive, notValid)
	}
	a, err := openArchive(file)
	if err != nil {
		_ = file.Close()
		return zipError(ctx, invalidArchive, notValid)
	}
	self.Put("$archive", a)
	return nil, nil
}

func (e *zipExterns) entries(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	a, closedErr, err := openArchiveOf(ctx, self)
	if a == nil {
		return closedErr, err
	}
	list, err := invokeHelper(ctx, "newEntries")
	if err != nil {
		return nil, err
	}
	entries, ok := list.(*values.List)
	if !ok {
		return nil, errors.New("zip: newEntries returned an unexpected value")
	}
	for _, entry := range a.entries {
		record, err := newEntryRecord(ctx, entry)
		if err != nil {
			return nil, err
		}
		entries.Append(ctx.TypeCtx(), record)
	}
	return entries, nil
}

func (e *zipExterns) indexOf(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	name, _ := args[1].(string)
	a, closedErr, err := openArchiveOf(ctx, self)
	if a == nil {
		return closedErr, err
	}
	return int64(a.indexOf(name)), nil
}

func (e *zipExterns) entryAt(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	index, _ := args[1].(int64)
	a, closedErr, err := openArchiveOf(ctx, self)
	if a == nil {
		return closedErr, err
	}
	return newEntryRecord(ctx, a.entries[index])
}

// newEntryRecord builds the Entry of an entry through the .bal newEntry helper. A time before the epoch
// reads as the epoch.
func newEntryRecord(ctx *extern.Context, entry *centralEntry) (values.BalValue, error) {
	seconds, nanos := entry.modified.Unix(), int64(entry.modified.Nanosecond())
	if seconds < 0 {
		seconds, nanos = 0, 0
	}
	var comment, mode values.BalValue
	if entry.comment != "" {
		comment = entry.comment
	}
	if m := entry.unixMode(); m != 0 {
		mode = int64(m & modeBits)
	}
	return invokeHelper(ctx, "newEntry", entry.name, entry.isDirectory(), entry.isSymlink(),
		int64(entry.uncompressedSize), int64(entry.compressedSize), methodName(entry.method),
		seconds, nanosToFraction(nanos), int64(entry.crc), comment, mode)
}

func methodName(method uint16) string {
	switch method {
	case methodStore:
		return "STORE"
	case methodDeflate:
		return "DEFLATE"
	}
	return "OTHER"
}

// nanosToFraction gives the fraction of a second with the nine digits Java's BigDecimal.valueOf(nanos, 9)
// keeps, except for zero, which has none.
func nanosToFraction(nanos int64) *decimal.Decimal {
	if nanos == 0 {
		return decimal.FromInt64(0)
	}
	d, _ := decimal.FromString(fmt.Sprintf("0.%09d", nanos))
	return d
}

// unreadable gives the error for an entry whose content this library cannot decode, or nil.
func unreadable(ctx *extern.Context, entry *centralEntry) (values.BalValue, error) {
	if entry.encrypted() {
		return entryError(ctx, unsupportedEntry,
			fmt.Sprintf("cannot read entry '%s': encrypted entries are not supported", entry.name), entry.name)
	}
	if entry.method != methodStore && entry.method != methodDeflate {
		return entryError(ctx, unsupportedEntry,
			fmt.Sprintf("entry '%s' uses a compression method that is not supported", entry.name), entry.name)
	}
	return nil, nil
}

func (e *zipExterns) readEntry(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	name, _ := args[1].(string)
	target, _ := args[2].(*values.TypeDesc)
	a, closedErr, err := openArchiveOf(ctx, self)
	if a == nil {
		return closedErr, err
	}
	index := a.indexOf(name)
	if index < 0 {
		return zipError(ctx, entryNotFound, fmt.Sprintf("the archive contains no entry named '%s'", name))
	}
	entry := a.entries[index]
	var content io.ReadCloser = io.NopCloser(bytes.NewReader(nil))
	if !entry.isDirectory() {
		if errValue, err := unreadable(ctx, entry); errValue != nil || err != nil {
			return errValue, err
		}
		content, _ = a.content(entry)
	}
	if target != nil && semtypes.IsSubtypeSimple(target.Constraint(), semtypes.Stream) {
		return e.newEntryStream(ctx, a, content, target.Constraint())
	}
	data, err := io.ReadAll(content)
	_ = content.Close()
	if err != nil {
		return zipError(ctx, invalidArchive, fmt.Sprintf("the content of entry '%s' could not be read", name))
	}
	return e.newByteArray(data), nil
}

// entryStream is the read position behind a stream handed out by readEntry. It is closed when the entry
// is read to the end, when the stream is closed, or when the archive is.
type entryStream struct {
	archive *archive
	content io.ReadCloser
	closed  bool
}

func (s *entryStream) release() {
	s.closed = true
	delete(s.archive.streams, s)
	_ = s.content.Close()
}

func (e *zipExterns) newEntryStream(ctx *extern.Context, a *archive, content io.ReadCloser,
	streamType semtypes.SemType) (values.BalValue, error) {
	// The stream methods run without an extern context, so the error they may return is shaped here.
	template, err := zipError(ctx, invalidArchive, "")
	if err != nil {
		return nil, err
	}
	tmpl, _ := template.(*values.Error)
	streamError := func(message string) values.BalValue {
		return values.NewError(tmpl.Type, message, nil, tmpl.TypeName, nil)
	}
	s := &entryStream{archive: a, content: content}
	a.mu.Lock()
	a.streams[s] = struct{}{}
	a.mu.Unlock()
	buf := make([]byte, chunkSize)
	next := func() values.BalValue {
		a.mu.Lock()
		defer a.mu.Unlock()
		if s.closed {
			return streamError("cannot read the entry: the stream is closed")
		}
		n, err := readChunk(s.content, buf)
		if err != nil {
			return streamError("cannot read the content of the entry")
		}
		if n == 0 {
			s.release()
			return nil
		}
		return values.NewMap(e.types.chunkTy, e.types.chunkAtom, false,
			[]values.MapEntry{{Key: "value", Value: e.newByteArray(buf[:n])}})
	}
	closeStream := func() values.BalValue {
		a.mu.Lock()
		defer a.mu.Unlock()
		if !s.closed {
			s.release()
		}
		return nil
	}
	return values.NewStream(streamType, next, closeStream), nil
}

func (e *zipExterns) closeArchive(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if err := archiveOf(self).close(); err != nil {
		return zipError(ctx, fileSystem, "cannot close the archive")
	}
	return nil, nil
}

func (e *zipExterns) extractEntry(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	index, _ := args[1].(int64)
	target, _ := args[2].(string)
	byteLimit, _ := args[3].(int64)
	maxRatio, _ := args[4].(int64)
	a, closedErr, err := openArchiveOf(ctx, self)
	if a == nil {
		return closedErr, err
	}
	entry := a.entries[index]
	if errValue, err := unreadable(ctx, entry); errValue != nil || err != nil {
		return errValue, err
	}
	if !representable(target) {
		return invalidPathError(ctx, target)
	}
	content, counter := a.content(entry)
	defer func() { _ = content.Close() }()
	result, failure := e.copyContent(content, counter, target, byteLimit, maxRatio)
	switch failure {
	case copyReadFailed:
		return zipError(ctx, invalidArchive, fmt.Sprintf("the content of entry '%s' could not be read", entry.name))
	case copyWriteFailed:
		return zipError(ctx, fileSystem, fmt.Sprintf("cannot extract entry '%s': the content could not be written to '%s'",
			entry.name, javaPathString(target)))
	}
	return result, nil
}

type copyFailure int

const (
	copyOK copyFailure = iota
	copyReadFailed
	copyWriteFailed
)

// copyContent writes an entry to target the way jBallerina does: the first chunk is read and judged
// before the file is opened, both limits are tested before each chunk is written, and a file a limit or
// an error stops part way is left as it is.
func (e *zipExterns) copyContent(content io.Reader, counter *countingReader, target string, byteLimit,
	maxRatio int64) (int64, copyFailure) {
	buf := make([]byte, copyBufferSize)
	count, err := readChunk(content, buf)
	if err != nil {
		return 0, copyReadFailed
	}
	if count > 0 {
		if byteLimit >= 0 && int64(count) > byteLimit {
			return bytesExceeded, copyOK
		}
		if ratioPassed(counter, int64(count), maxRatio) {
			return ratioExceeded, copyOK
		}
	}
	out, err := e.rt.Platform().FS.CreateSeekable(target, false)
	if err != nil {
		return 0, copyWriteFailed
	}
	written := int64(0)
	for count > 0 {
		if byteLimit >= 0 && written+int64(count) > byteLimit {
			return closeAfter(out, bytesExceeded)
		}
		if ratioPassed(counter, written+int64(count), maxRatio) {
			return closeAfter(out, ratioExceeded)
		}
		if _, err := out.Write(buf[:count]); err != nil {
			_ = out.Close()
			return 0, copyWriteFailed
		}
		written += int64(count)
		if count, err = readChunk(content, buf); err != nil {
			_ = out.Close()
			return 0, copyReadFailed
		}
	}
	return closeAfter(out, written)
}

func closeAfter(out io.Closer, result int64) (int64, copyFailure) {
	if err := out.Close(); err != nil {
		return 0, copyWriteFailed
	}
	return result, copyOK
}

// readChunk fills buf unless the content ends first, and reports the end as a count of zero. Unlike
// io.ReadFull it does not turn a truncated deflate stream, which flate reports as io.ErrUnexpectedEOF,
// into an end of content.
func readChunk(content io.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := content.Read(buf[n:])
		n += m
		if errors.Is(err, io.EOF) {
			return n, nil
		}
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// ratioPassed tells whether an entry has expanded past the ratio allowed, measured against the compressed
// bytes taken from the archive so far rather than the size the archive claims.
func ratioPassed(counter *countingReader, written, maxRatio int64) bool {
	consumed := counter.count
	if maxRatio <= 0 || consumed <= 0 || maxRatio > math.MaxInt64/consumed {
		return false
	}
	return written > maxRatio*consumed
}
