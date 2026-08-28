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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package splice

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
)

// ELFSectionName is the ELF section the payload is stored under.
// Duplicated as linuxSectionName in
// cli/internal/executable/embed_linux.go.
const ELFSectionName = ".balexe.payload"

// elfSection64Size and elfHeader64Size are elf.Section64/Header64's
// on-disk sizes (encoding/binary packs their fields with no padding).
const (
	elfSection64Size = 64
	elfHeader64Size  = 64
)

// spliceELFSection returns a copy of stub with a new SHT_PROGBITS
// section named sectionName appended, holding payload verbatim. stub
// must be 64-bit ELF (ELFCLASS64).
//
// .shstrtab, not the section table, sits at true EOF in a Go binary, so
// the old table is left dead and a new one appended (old entries + one
// new, Shstrndx's Off/Size repatched), then e_shoff/e_shnum updated.
// SHF_ALLOC is omitted so execve's loader never maps it.
func spliceELFSection(stub []byte, sectionName string, payload []byte) ([]byte, error) {
	f, err := elf.NewFile(bytes.NewReader(stub))
	if err != nil {
		return nil, fmt.Errorf("parsing ELF stub: %w", err)
	}
	if f.Class != elf.ELFCLASS64 {
		return nil, fmt.Errorf("unsupported ELF class %s: only ELFCLASS64 stubs are supported", f.Class)
	}
	for _, s := range f.Sections {
		if s.Name == sectionName {
			return nil, fmt.Errorf("stub already has a %q section", sectionName)
		}
	}

	var hdr elf.Header64
	if err := binary.Read(bytes.NewReader(stub), f.ByteOrder, &hdr); err != nil {
		return nil, fmt.Errorf("reading ELF header: %w", err)
	}
	if hdr.Shentsize != elfSection64Size {
		return nil, fmt.Errorf("unsupported ELF section header entry size %d", hdr.Shentsize)
	}
	if hdr.Shnum == 0 {
		return nil, fmt.Errorf("unsupported ELF binary using extended section numbering")
	}
	if hdr.Shnum >= 0xff00-1 {
		return nil, fmt.Errorf("unsupported ELF section count %d: adding a section requires extended numbering", hdr.Shnum)
	}
	if hdr.Shstrndx == 0 {
		return nil, fmt.Errorf("ELF has no section name string table")
	}
	if int(hdr.Shstrndx) >= int(hdr.Shnum) || int(hdr.Shstrndx) >= len(f.Sections) {
		return nil, fmt.Errorf("invalid ELF string table index %d (Shnum=%d)", hdr.Shstrndx, hdr.Shnum)
	}

	oldShoff := int64(hdr.Shoff)
	oldShnum := int64(hdr.Shnum)
	tableSize := oldShnum * elfSection64Size
	// Subtraction, not oldShoff+tableSize, so a malformed huge Shoff can't
	// wrap the sum past int64 and slip through the bounds check.
	if oldShoff < 0 || tableSize < 0 || oldShoff > int64(len(stub)) || tableSize > int64(len(stub))-oldShoff {
		return nil, fmt.Errorf("ELF section header table out of range")
	}
	// Copy: the original bytes at oldShoff are left untouched in the
	// output (dead bytes); this copy is what gets patched and relocated.
	oldTable := append([]byte(nil), stub[oldShoff:oldShoff+tableSize]...)

	strtabSection := f.Sections[hdr.Shstrndx]
	strtabEnd := strtabSection.Offset + strtabSection.FileSize
	if strtabEnd < strtabSection.Offset || strtabEnd > uint64(len(stub)) {
		return nil, fmt.Errorf("ELF string table out of range")
	}
	oldStrtab := stub[strtabSection.Offset:strtabEnd]

	newStrtab := make([]byte, 0, len(oldStrtab)+len(sectionName)+1)
	newStrtab = append(newStrtab, oldStrtab...)
	nameOffset := uint32(len(newStrtab))
	newStrtab = append(newStrtab, sectionName...)
	newStrtab = append(newStrtab, 0)

	out := append([]byte(nil), stub...)

	payloadOff := uint64(len(out))
	out = append(out, payload...)

	strtabOff := uint64(len(out))
	out = append(out, newStrtab...)

	// Patch the copied Shstrndx entry's Off (byte 24) and Size (byte 32)
	// fields in place; every other field of every copied entry, including
	// the rest of this one, is left byte-for-byte as parsed.
	strEntryOff := int(hdr.Shstrndx) * elfSection64Size
	f.ByteOrder.PutUint64(oldTable[strEntryOff+24:strEntryOff+32], strtabOff)
	f.ByteOrder.PutUint64(oldTable[strEntryOff+32:strEntryOff+40], uint64(len(newStrtab)))

	newEntry := elf.Section64{
		Name:      nameOffset,
		Type:      uint32(elf.SHT_PROGBITS),
		Flags:     0, // no SHF_ALLOC: inert to the loader
		Addr:      0,
		Off:       payloadOff,
		Size:      uint64(len(payload)),
		Link:      0,
		Info:      0,
		Addralign: 1,
		Entsize:   0,
	}
	var newEntryBuf bytes.Buffer
	if err := binary.Write(&newEntryBuf, f.ByteOrder, newEntry); err != nil {
		return nil, fmt.Errorf("encoding new section header: %w", err)
	}

	newShoff := uint64(len(out))
	out = append(out, oldTable...)
	out = append(out, newEntryBuf.Bytes()...)

	hdr.Shoff = newShoff
	hdr.Shnum = uint16(oldShnum + 1)
	var hdrBuf bytes.Buffer
	if err := binary.Write(&hdrBuf, f.ByteOrder, hdr); err != nil {
		return nil, fmt.Errorf("encoding ELF header: %w", err)
	}
	copy(out[:elfHeader64Size], hdrBuf.Bytes())

	return out, nil
}
