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
	"bufio"
	"compress/flate"
	"encoding/binary"
	"errors"
	"hash"
	"hash/crc32"
	"io"
	"io/fs"
	"math"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

const (
	versionStore   = 10
	versionDeflate = 20
	versionZip64   = 45

	zip64Limit = 0xffffffff
	// dosBefore1980 is the DOS time 1980-01-01 00:00:00, which stands for any earlier time.
	dosBefore1980 = 1<<21 | 1<<16

	unixFileKind = 0o100000
	unixDirKind  = 0o040000
	dosDirAttr   = 0x10
	dosReadOnly  = 0x01
)

// writtenEntry is what the central directory records for an entry once it is written.
type writtenEntry struct {
	name             []byte
	platform         uint16
	method           uint16
	dosTime          uint32
	crc              uint32
	compressedSize   uint64
	uncompressedSize uint64
	internalAttrs    uint16
	externalAttrs    uint32
	headerOffset     uint64
	localExtra       []byte
	centralExtra     []byte
	comment          []byte
	zip64Placeholder bool
}

func (w *writtenEntry) needsZip64() bool {
	return w.compressedSize >= zip64Limit || w.uncompressedSize >= zip64Limit
}

func (w *writtenEntry) versionNeeded() uint16 {
	if w.needsZip64() {
		return versionZip64
	}
	if w.method == methodDeflate {
		return versionDeflate
	}
	return versionStore
}

// openEntry is the entry whose content is being written.
type openEntry struct {
	entry      *writtenEntry
	crc        hash.Hash32
	size       uint64
	compressor *flate.Writer
	dataStart  int64
}

// zipWriter writes an archive the way Apache Commons Compress does on a seekable file: the CRC and the
// sizes of an entry are written back into its local header once its content is written, so no entry
// carries a data descriptor.
type zipWriter struct {
	file    pal.SeekableFile
	out     *bufio.Writer
	offset  int64
	method  uint16
	level   int
	entries []*writtenEntry
	current *openEntry
	closed  bool
}

func newZipWriter(file pal.SeekableFile, level string) *zipWriter {
	w := &zipWriter{file: file, method: methodDeflate}
	w.out = bufio.NewWriter(countingWriter{w})
	switch level {
	case "NONE":
		w.method = methodStore
	case "FASTEST":
		w.level = flate.BestSpeed
	case "BEST":
		w.level = flate.BestCompression
	default:
		w.level = flate.DefaultCompression
	}
	return w
}

// countingWriter keeps the offset of the writer, counting the bytes as they reach the file.
type countingWriter struct {
	w *zipWriter
}

func (c countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.file.Write(p)
	c.w.offset += int64(n)
	return n, err
}

func (w *zipWriter) position() int64 {
	return w.offset + int64(w.out.Buffered())
}

func le16(b []byte, v uint16) []byte {
	return binary.LittleEndian.AppendUint16(b, v)
}

func le32(b []byte, v uint32) []byte {
	return binary.LittleEndian.AppendUint32(b, v)
}

func le64(b []byte, v uint64) []byte {
	return binary.LittleEndian.AppendUint64(b, v)
}

func size32(v uint64) uint32 {
	return uint32(min(v, zip64Limit))
}

// timeFields gives the DOS time of t in the local time zone and, for a time the DOS format cannot hold,
// the extended timestamp, when the seconds fit it, and the NTFS time.
func timeFields(t time.Time) (uint32, []byte) {
	local := t.In(time.Local)
	year := local.Year()
	var dos uint32
	if year < 1980 {
		dos = dosBefore1980
	} else {
		dos = uint32(year-1980)&0x7f<<25 | uint32(local.Month())<<21 | uint32(local.Day())<<16 |
			uint32(local.Hour())<<11 | uint32(local.Minute())<<5 | uint32(local.Second())>>1
	}
	if year >= 1980 && year <= 2107 {
		return dos, nil
	}
	var extra []byte
	if seconds := t.Unix(); seconds >= math.MinInt32 && seconds <= math.MaxInt32 {
		extra = le16(extra, extraTimestamp)
		extra = le16(extra, 5)
		extra = append(extra, 1)
		extra = le32(extra, uint32(int32(seconds)))
	}
	extra = le16(extra, extraNTFS)
	extra = le16(extra, 32)
	extra = le32(extra, 0)
	extra = le16(extra, 1)
	extra = le16(extra, 24)
	extra = le64(extra, uint64(t.Unix()*10_000_000+int64(t.Nanosecond()/100)+ntfsEpochOffset))
	extra = le64(extra, 0)
	extra = le64(extra, 0)
	return dos, extra
}

// zip64Placeholder is the Zip64 field Apache Commons Compress puts in the local header of an entry whose
// size is not known when the header is written, so that the sizes can be filled in afterwards.
func zip64Placeholder() []byte {
	field := le16(nil, extraZip64)
	field = le16(field, 16)
	return append(field, make([]byte, 16)...)
}

func (w *zipWriter) writeLocalHeader(e *writtenEntry) error {
	h := le32(make([]byte, 0, localHeaderLen+len(e.name)+len(e.localExtra)), localHeaderSig)
	h = le16(h, e.versionNeeded())
	h = le16(h, flagUTF8)
	h = le16(h, e.method)
	h = le32(h, e.dosTime)
	h = le32(h, e.crc)
	h = le32(h, size32(e.compressedSize))
	h = le32(h, size32(e.uncompressedSize))
	h = le16(h, uint16(len(e.name)))
	h = le16(h, uint16(len(e.localExtra)))
	h = append(h, e.name...)
	h = append(h, e.localExtra...)
	_, err := w.out.Write(h)
	return err
}

// startEntry writes the local header of an entry and opens it for content. An entry still open is
// finished first.
func (w *zipWriter) startEntry(e *writtenEntry, sizeKnown bool) error {
	if w.current != nil {
		if err := w.finishEntry(); err != nil {
			return err
		}
	}
	e.headerOffset = uint64(w.position())
	if !sizeKnown {
		e.zip64Placeholder = true
		e.localExtra = append(zip64Placeholder(), e.localExtra...)
	}
	if err := w.writeLocalHeader(e); err != nil {
		return err
	}
	open := &openEntry{entry: e, crc: crc32.NewIEEE(), dataStart: w.position()}
	if e.method == methodDeflate {
		// The level is one of the four the writer maps, all of which flate accepts.
		open.compressor, _ = flate.NewWriter(w.out, w.level)
	}
	w.current = open
	return nil
}

func (w *zipWriter) write(p []byte) error {
	w.current.crc.Write(p)
	w.current.size += uint64(len(p))
	if w.current.compressor != nil {
		_, err := w.current.compressor.Write(p)
		return err
	}
	_, err := w.out.Write(p)
	return err
}

// finishEntry completes the open entry and writes its CRC and sizes back into its local header.
func (w *zipWriter) finishEntry() error {
	open := w.current
	w.current = nil
	if open.compressor != nil {
		if err := open.compressor.Close(); err != nil {
			return err
		}
	}
	e := open.entry
	e.crc = open.crc.Sum32()
	e.uncompressedSize = open.size
	e.compressedSize = uint64(w.position() - open.dataStart)
	w.entries = append(w.entries, e)
	if e.needsZip64() && !e.zip64Placeholder {
		return errors.New("zip: the entry needs Zip64 but its size was declared")
	}
	return w.rewriteLocalHeader(e)
}

func (w *zipWriter) rewriteLocalHeader(e *writtenEntry) error {
	if err := w.out.Flush(); err != nil {
		return err
	}
	header := int64(e.headerOffset)
	patch := le16(nil, e.versionNeeded())
	if err := w.patchAt(header+4, patch); err != nil {
		return err
	}
	patch = le32(nil, e.crc)
	patch = le32(patch, size32(e.compressedSize))
	patch = le32(patch, size32(e.uncompressedSize))
	if err := w.patchAt(header+14, patch); err != nil {
		return err
	}
	if e.zip64Placeholder {
		patch = le64(nil, e.uncompressedSize)
		patch = le64(patch, e.compressedSize)
		if err := w.patchAt(header+localHeaderLen+int64(len(e.name))+4, patch); err != nil {
			return err
		}
	}
	_, err := w.file.Seek(w.offset, io.SeekStart)
	return err
}

func (w *zipWriter) patchAt(offset int64, data []byte) error {
	if _, err := w.file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	_, err := w.file.Write(data)
	return err
}

// copyRaw writes an entry whose stored content, CRC and sizes are already known.
func (w *zipWriter) copyRaw(e *writtenEntry, content io.Reader) error {
	if w.current != nil {
		if err := w.finishEntry(); err != nil {
			return err
		}
	}
	e.headerOffset = uint64(w.position())
	if err := w.writeLocalHeader(e); err != nil {
		return err
	}
	if _, err := io.CopyN(w.out, content, int64(e.compressedSize)); err != nil {
		return err
	}
	w.entries = append(w.entries, e)
	return nil
}

func (w *zipWriter) centralZip64(e *writtenEntry) []byte {
	var data []byte
	if e.needsZip64() {
		data = le64(data, e.uncompressedSize)
		data = le64(data, e.compressedSize)
	}
	if e.headerOffset >= zip64Limit {
		data = le64(data, e.headerOffset)
	}
	if data == nil {
		return nil
	}
	field := le16(nil, extraZip64)
	field = le16(field, uint16(len(data)))
	return append(field, data...)
}

func (w *zipWriter) writeCentralDirectory() error {
	start := w.position()
	for _, e := range w.entries {
		extra := append(w.centralZip64(e), e.centralExtra...)
		madeBy := uint16(versionDeflate)
		if e.needsZip64() || e.headerOffset >= zip64Limit {
			madeBy = versionZip64
		}
		h := le32(nil, centralHeaderSig)
		h = le16(h, e.platform<<8|madeBy)
		h = le16(h, e.versionNeeded())
		h = le16(h, flagUTF8)
		h = le16(h, e.method)
		h = le32(h, e.dosTime)
		h = le32(h, e.crc)
		h = le32(h, size32(e.compressedSize))
		h = le32(h, size32(e.uncompressedSize))
		h = le16(h, uint16(len(e.name)))
		h = le16(h, uint16(len(extra)))
		h = le16(h, uint16(len(e.comment)))
		h = le16(h, 0)
		h = le16(h, e.internalAttrs)
		h = le32(h, e.externalAttrs)
		h = le32(h, size32(e.headerOffset))
		h = append(h, e.name...)
		h = append(h, extra...)
		h = append(h, e.comment...)
		if _, err := w.out.Write(h); err != nil {
			return err
		}
	}
	return w.writeEnd(start, w.position()-start)
}

func (w *zipWriter) writeEnd(start, size int64) error {
	count := uint64(len(w.entries))
	var h []byte
	if count >= 0xffff || start >= zip64Limit || size >= zip64Limit {
		zip64End := w.position()
		h = le32(h, zip64EndSig)
		h = le64(h, 44)
		h = le16(h, versionZip64)
		h = le16(h, versionZip64)
		h = le32(h, 0)
		h = le32(h, 0)
		h = le64(h, count)
		h = le64(h, count)
		h = le64(h, uint64(size))
		h = le64(h, uint64(start))
		h = le32(h, zip64LocatorSig)
		h = le32(h, 0)
		h = le64(h, uint64(zip64End))
		h = le32(h, 1)
	}
	h = le32(h, endSig)
	h = le16(h, 0)
	h = le16(h, 0)
	h = le16(h, uint16(min(count, 0xffff)))
	h = le16(h, uint16(min(count, 0xffff)))
	h = le32(h, size32(uint64(size)))
	h = le32(h, size32(uint64(start)))
	h = le16(h, 0)
	_, err := w.out.Write(h)
	return err
}

// close writes the central directory and closes the file, whatever happens on the way.
func (w *zipWriter) close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	err := w.complete()
	if closeErr := w.file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (w *zipWriter) complete() error {
	if err := w.writeCentralDirectory(); err != nil {
		return err
	}
	return w.out.Flush()
}

// fileEntry describes an entry added from a file: its time and, where the platform keeps them, its
// permissions come from the source.
func fileEntry(name string, info *pal.FileInfo, method uint16) *writtenEntry {
	e := &writtenEntry{name: []byte(name), method: method, platform: platformFAT}
	e.dosTime, e.localExtra = timeFields(info.ModifiedAt)
	e.centralExtra = e.localExtra
	directory := isDirectoryName(name)
	if directory {
		e.method = methodStore
	}
	if info.HasPermissions {
		mode := info.Permissions & 0o777
		if directory {
			mode |= unixDirKind
		} else {
			mode |= unixFileKind
		}
		e.platform = platformUnix
		e.externalAttrs = mode << 16
		if mode&0o200 == 0 {
			e.externalAttrs |= dosReadOnly
		}
		if directory {
			e.externalAttrs |= dosDirAttr
		}
	}
	return e
}

// contentEntry describes an entry whose content the caller supplies, stamped with the current time.
func contentEntry(name string, now time.Time, method uint16) *writtenEntry {
	e := &writtenEntry{name: []byte(name), method: method, platform: platformFAT}
	e.dosTime, e.localExtra = timeFields(now)
	e.centralExtra = e.localExtra
	if isDirectoryName(name) {
		e.method = methodStore
	}
	return e
}

// isDirectoryName tells whether a name denotes a directory entry, which is stored empty.
func isDirectoryName(name string) bool {
	return strings.HasSuffix(name, "/")
}

// copiedEntry describes an entry copied from another archive under the name this library read it by.
// Everything stored about it is kept except the Zip64 field, which is written again where it is needed.
func copiedEntry(source *centralEntry) *writtenEntry {
	dos, _ := timeFields(source.modified)
	return &writtenEntry{
		name:             []byte(source.name),
		platform:         source.platform(),
		method:           source.method,
		dosTime:          dos,
		crc:              source.crc,
		compressedSize:   source.compressedSize,
		uncompressedSize: source.uncompressedSize,
		internalAttrs:    source.internalAttrs,
		externalAttrs:    source.externalAttrs,
		localExtra:       withoutField(source.localExtra, extraZip64),
		centralExtra:     withoutField(source.centralExtra, extraZip64),
		comment:          []byte(source.comment),
	}
}

func withoutField(extra []byte, id uint16) []byte {
	var kept []byte
	for len(extra) >= 4 {
		size := int(binary.LittleEndian.Uint16(extra[2:]))
		if 4+size > len(extra) {
			return append(kept, extra...)
		}
		if binary.LittleEndian.Uint16(extra) != id {
			kept = append(kept, extra[:4+size]...)
		}
		extra = extra[4+size:]
	}
	return kept
}

func isExist(err error) bool {
	return errors.Is(err, fs.ErrExist)
}
