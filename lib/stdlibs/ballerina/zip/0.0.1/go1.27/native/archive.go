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
	"io"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

const (
	localHeaderSig   = 0x04034b50
	centralHeaderSig = 0x02014b50
	endSig           = 0x06054b50
	zip64LocatorSig  = 0x07064b50
	zip64EndSig      = 0x06064b50

	localHeaderLen   = 30
	centralHeaderLen = 46
	endLen           = 22
	zip64LocatorLen  = 20
	maxCommentLen    = 0xffff

	flagEncrypted = 0x1
	flagUTF8      = 0x800

	methodStore   = 0
	methodDeflate = 8

	platformFAT  = 0
	platformUnix = 3

	extraZip64     = 0x0001
	extraNTFS      = 0x000a
	extraTimestamp = 0x5455

	modeTypeMask = 0o170000
	modeSymlink  = 0o120000
	modeBits     = 0o7777

	// ntfsEpochOffset is the number of 100 ns intervals between 1601-01-01 and 1970-01-01.
	ntfsEpochOffset = 116444736000000000
)

var errInvalidArchive = errors.New("zip: not a valid archive")

// centralEntry is an entry as its central directory record and local header describe it.
type centralEntry struct {
	name             string
	rawName          []byte
	versionMadeBy    uint16
	flags            uint16
	method           uint16
	dosTime          uint32
	crc              uint32
	compressedSize   uint64
	uncompressedSize uint64
	diskStart        uint32
	internalAttrs    uint16
	externalAttrs    uint32
	headerOffset     uint64
	dataOffset       int64
	comment          string
	rawComment       []byte
	centralExtra     []byte
	localExtra       []byte
	modified         time.Time
}

func (e *centralEntry) platform() uint16 {
	return e.versionMadeBy >> 8 & 0x0f
}

// isDirectory follows Apache Commons Compress, which first turns `\` into `/` in the name of an entry
// made on FAT that holds no `/`, while the name this library reports keeps the `\`.
func (e *centralEntry) isDirectory() bool {
	n := len(e.name)
	if n > 0 && e.name[n-1] == '/' {
		return true
	}
	if e.platform() != platformFAT || n == 0 || e.name[n-1] != '\\' {
		return false
	}
	for i := 0; i < n; i++ {
		if e.name[i] == '/' {
			return false
		}
	}
	return true
}

// unixMode is the full mode the archive records, which only an entry made on a Unix platform has.
func (e *centralEntry) unixMode() uint32 {
	if e.platform() != platformUnix {
		return 0
	}
	return e.externalAttrs >> 16 & 0xffff
}

func (e *centralEntry) isSymlink() bool {
	return e.unixMode()&modeTypeMask == modeSymlink
}

func (e *centralEntry) encrypted() bool {
	return e.flags&flagEncrypted != 0
}

// archive is an open ZIP file and the index of its entries in the order they are stored.
type archive struct {
	mu         sync.Mutex
	file       pal.RandomAccessFile
	entries    []*centralEntry
	firstIndex map[string]int
	streams    map[*entryStream]struct{}
	closed     bool
}

func openArchive(file pal.RandomAccessFile) (*archive, error) {
	entries, centralStart, err := readCentralDirectory(file)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if err := resolveLocalHeader(file, e, centralStart); err != nil {
			return nil, err
		}
		e.modified = modifiedTime(e)
	}
	slices.SortStableFunc(entries, func(a, b *centralEntry) int {
		if a.diskStart != b.diskStart {
			return compareUint(uint64(a.diskStart), uint64(b.diskStart))
		}
		return compareUint(a.headerOffset, b.headerOffset)
	})
	a := &archive{
		file:       file,
		entries:    entries,
		firstIndex: make(map[string]int),
		streams:    make(map[*entryStream]struct{}),
	}
	for i, e := range entries {
		if _, ok := a.firstIndex[e.name]; !ok {
			a.firstIndex[e.name] = i
		}
	}
	return a, nil
}

func compareUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func (a *archive) indexOf(name string) int {
	if i, ok := a.firstIndex[name]; ok {
		return i
	}
	return -1
}

// close closes every stream still open on the archive, then the file.
func (a *archive) close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	for s := range a.streams {
		s.closed = true
	}
	clear(a.streams)
	return a.file.Close()
}

func (a *archive) isClosed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closed
}

func readUint16(r io.ReaderAt, off int64) (uint16, error) {
	var buf [2]byte
	if _, err := r.ReadAt(buf[:], off); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buf[:]), nil
}

func readUint32(r io.ReaderAt, off int64) (uint32, error) {
	var buf [4]byte
	if _, err := r.ReadAt(buf[:], off); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

func readUint64(r io.ReaderAt, off int64) (uint64, error) {
	var buf [8]byte
	if _, err := r.ReadAt(buf[:], off); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}

// findEnd returns the position of the end of central directory record, searched for backwards from the
// end of the file as far as the longest archive comment reaches.
func findEnd(file pal.RandomAccessFile) (int64, error) {
	size := file.Size()
	if size < endLen {
		return 0, errInvalidArchive
	}
	start := max(size-endLen-maxCommentLen, 0)
	buf := make([]byte, size-start)
	if _, err := file.ReadAt(buf, start); err != nil {
		return 0, err
	}
	for i := len(buf) - endLen; i >= 0; i-- {
		if binary.LittleEndian.Uint32(buf[i:]) == endSig {
			return start + int64(i), nil
		}
	}
	return 0, errInvalidArchive
}

func centralDirectoryOffset(file pal.RandomAccessFile, end int64) (int64, error) {
	if end >= zip64LocatorLen {
		if sig, err := readUint32(file, end-zip64LocatorLen); err == nil && sig == zip64LocatorSig {
			zip64End, err := readUint64(file, end-zip64LocatorLen+8)
			if err != nil {
				return 0, err
			}
			if sig, err := readUint32(file, int64(zip64End)); err != nil || sig != zip64EndSig {
				return 0, errInvalidArchive
			}
			offset, err := readUint64(file, int64(zip64End)+48)
			return int64(offset), err
		}
	}
	offset, err := readUint32(file, end+16)
	return int64(offset), err
}

// readCentralDirectory reads records for as long as their signatures hold, and returns them with the
// offset the central directory starts at. The entry count of the trailer is never trusted, so nothing is
// allocated from it.
func readCentralDirectory(file pal.RandomAccessFile) ([]*centralEntry, int64, error) {
	end, err := findEnd(file)
	if err != nil {
		return nil, 0, err
	}
	offset, err := centralDirectoryOffset(file, end)
	if err != nil {
		return nil, 0, err
	}
	if offset < 0 || offset > file.Size() {
		return nil, 0, errInvalidArchive
	}
	r := bufio.NewReader(io.NewSectionReader(file, offset, file.Size()-offset))
	var entries []*centralEntry
	var header [centralHeaderLen]byte
	for {
		if _, err := io.ReadFull(r, header[:4]); err != nil {
			return nil, 0, err
		}
		if binary.LittleEndian.Uint32(header[:4]) != centralHeaderSig {
			break
		}
		if _, err := io.ReadFull(r, header[4:]); err != nil {
			return nil, 0, err
		}
		e, err := readCentralEntry(r, header[:])
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		if sig, err := readUint32(file, 0); err == nil && sig == localHeaderSig {
			return nil, 0, errInvalidArchive
		}
	}
	return entries, offset, nil
}

func readCentralEntry(r io.Reader, h []byte) (*centralEntry, error) {
	le := binary.LittleEndian
	e := &centralEntry{
		versionMadeBy:    le.Uint16(h[4:]),
		flags:            le.Uint16(h[8:]),
		method:           le.Uint16(h[10:]),
		dosTime:          le.Uint32(h[12:]),
		crc:              le.Uint32(h[16:]),
		compressedSize:   uint64(le.Uint32(h[20:])),
		uncompressedSize: uint64(le.Uint32(h[24:])),
		diskStart:        uint32(le.Uint16(h[34:])),
		internalAttrs:    le.Uint16(h[36:]),
		externalAttrs:    le.Uint32(h[38:]),
		headerOffset:     uint64(le.Uint32(h[42:])),
	}
	nameLen, extraLen, commentLen := int(le.Uint16(h[28:])), int(le.Uint16(h[30:])), int(le.Uint16(h[32:]))
	rest := make([]byte, nameLen+extraLen+commentLen)
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, err
	}
	e.rawName = rest[:nameLen]
	e.centralExtra = rest[nameLen : nameLen+extraLen]
	e.rawComment = rest[nameLen+extraLen:]
	e.name = entryName(e.rawName, e.flags)
	e.comment = decodeUTF8(e.rawComment, commentReplacement)
	if err := applyZip64(e); err != nil {
		return nil, err
	}
	// Apache Commons Compress holds these as signed numbers and refuses an archive where one is negative.
	if e.uncompressedSize > math.MaxInt64 || e.compressedSize > math.MaxInt64 || e.headerOffset > math.MaxInt64 ||
		e.diskStart > math.MaxInt32 {
		return nil, errInvalidArchive
	}
	return e, nil
}

// applyZip64 takes the sizes, offset and disk the record leaves at their maximum from the Zip64 field,
// which holds only those, in that order.
func applyZip64(e *centralEntry) error {
	needSize := e.uncompressedSize == 0xffffffff
	needCompressed := e.compressedSize == 0xffffffff
	needOffset := e.headerOffset == 0xffffffff
	needDisk := e.diskStart == 0xffff
	if !needSize && !needCompressed && !needOffset && !needDisk {
		return nil
	}
	data, ok := extraField(e.centralExtra, extraZip64)
	if !ok {
		return nil
	}
	read := func(n int) (uint64, error) {
		if len(data) < n {
			return 0, errInvalidArchive
		}
		var v uint64
		if n == 8 {
			v = binary.LittleEndian.Uint64(data)
		} else {
			v = uint64(binary.LittleEndian.Uint32(data))
		}
		data = data[n:]
		return v, nil
	}
	var err error
	if needSize {
		if e.uncompressedSize, err = read(8); err != nil {
			return err
		}
	}
	if needCompressed {
		if e.compressedSize, err = read(8); err != nil {
			return err
		}
	}
	if needOffset {
		if e.headerOffset, err = read(8); err != nil {
			return err
		}
	}
	if needDisk {
		disk, err := read(4)
		if err != nil {
			return err
		}
		e.diskStart = uint32(disk)
	}
	return nil
}

// resolveLocalHeader finds where the content of an entry starts and reads the extra fields of its local
// header. Like Apache Commons Compress, the signature of the local header is not checked, and content that
// would run into the central directory makes the archive invalid.
func resolveLocalHeader(file pal.RandomAccessFile, e *centralEntry, centralStart int64) error {
	offset := int64(e.headerOffset)
	nameLen, err := readUint16(file, offset+26)
	if err != nil {
		return err
	}
	extraLen, err := readUint16(file, offset+28)
	if err != nil {
		return err
	}
	e.localExtra = make([]byte, extraLen)
	if _, err := file.ReadAt(e.localExtra, offset+localHeaderLen+int64(nameLen)); err != nil {
		return err
	}
	e.dataOffset = offset + localHeaderLen + int64(nameLen) + int64(extraLen)
	if e.dataOffset > centralStart || e.compressedSize > uint64(centralStart-e.dataOffset) {
		return errInvalidArchive
	}
	return nil
}

// extraField returns the data of the first field with the given id.
func extraField(extra []byte, id uint16) ([]byte, bool) {
	for len(extra) >= 4 {
		fieldID := binary.LittleEndian.Uint16(extra)
		size := int(binary.LittleEndian.Uint16(extra[2:]))
		if 4+size > len(extra) {
			return nil, false
		}
		if fieldID == id {
			return extra[4 : 4+size], true
		}
		extra = extra[4+size:]
	}
	return nil, false
}

// timeField returns the data of a time field, the local header's copy winning over the central one.
func timeField(e *centralEntry, id uint16) ([]byte, bool) {
	if data, ok := extraField(e.localExtra, id); ok {
		return data, true
	}
	return extraField(e.centralExtra, id)
}

// modifiedTime takes the time of an entry from the Info-ZIP extended timestamp, then from the NTFS
// field, which overrides it, and failing both from the DOS time read in the local time zone.
func modifiedTime(e *centralEntry) time.Time {
	var modified time.Time
	if data, ok := timeField(e, extraTimestamp); ok && len(data) >= 5 && data[0]&1 != 0 {
		modified = time.Unix(int64(int32(binary.LittleEndian.Uint32(data[1:]))), 0)
	}
	if ntfs, ok := ntfsModifiedTime(e); ok {
		modified = ntfs
	}
	if !modified.IsZero() {
		return modified
	}
	return dosToTime(e.dosTime)
}

func ntfsModifiedTime(e *centralEntry) (time.Time, bool) {
	data, ok := timeField(e, extraNTFS)
	if !ok || len(data) < 4 {
		return time.Time{}, false
	}
	data = data[4:]
	for len(data) >= 4 {
		tag := binary.LittleEndian.Uint16(data)
		size := int(binary.LittleEndian.Uint16(data[2:]))
		if tag == 1 {
			if size < 24 || len(data) < 4+24 {
				return time.Time{}, false
			}
			value := int64(binary.LittleEndian.Uint64(data[4:]))
			if value == 0 {
				return time.Time{}, false
			}
			intervals := value - ntfsEpochOffset
			seconds := floorDiv(intervals, 10_000_000)
			return time.Unix(seconds, (intervals-seconds*10_000_000)*100), true
		}
		if 4+size > len(data) {
			break
		}
		data = data[4+size:]
	}
	return time.Time{}, false
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// dosToTime reads a DOS date and time in the local time zone, normalising fields out of range the way
// Java's lenient Calendar does.
func dosToTime(dos uint32) time.Time {
	return time.Date(int(dos>>25&0x7f)+1980, time.Month(dos>>21&0x0f), int(dos>>16&0x1f),
		int(dos>>11&0x1f), int(dos>>5&0x3f), int(dos<<1&0x3e), 0, time.Local)
}

// rawContent is the stored, still compressed content of an entry.
func (a *archive) rawContent(e *centralEntry) *io.SectionReader {
	return io.NewSectionReader(a.file, e.dataOffset, int64(e.compressedSize))
}

// countingReader counts the compressed bytes the decompressor takes. Being an io.ByteReader keeps flate
// from reading ahead of what it needs.
type countingReader struct {
	r     *bufio.Reader
	count int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.count += int64(n)
	return n, err
}

func (c *countingReader) ReadByte() (byte, error) {
	b, err := c.r.ReadByte()
	if err == nil {
		c.count++
	}
	return b, err
}

// content returns the decompressed content of an entry and a count of the compressed bytes taken from
// the archive to produce it. Like Apache Commons Compress, the checksum is not verified.
func (a *archive) content(e *centralEntry) (io.ReadCloser, *countingReader) {
	counter := &countingReader{r: bufio.NewReader(a.rawContent(e))}
	if e.method == methodDeflate {
		return flate.NewReader(counter), counter
	}
	return io.NopCloser(counter), counter
}
