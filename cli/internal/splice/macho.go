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
	"debug/macho"
	"encoding/binary"
	"fmt"
	"math"
)

// MachOSegmentName and MachOSectionName identify the segment/section the
// payload is stored under. Duplicated as darwinSegmentName/
// darwinSectionName in cli/internal/executable/embed_darwin.go.
const (
	MachOSegmentName = "__BALEXE"
	MachOSectionName = "__payload"
)

const (
	machoLcSegment64     = 0x19
	machoLcSymtab        = 0x2
	machoLcDysymtab      = 0xb
	machoLcDyldInfo      = 0x22
	machoLcDyldInfoOnly  = 0x80000022
	machoLcCodeSignature = 0x1d
	machoHeader64Size    = 32
	machoSegmentCmdSize  = 72 // segment_command_64 size
	machoSectionCmdSize  = 80 // section_64 size
	machoCodeSigCmdSize  = 16 // linkedit_data_command size (LC_CODE_SIGNATURE)

	// 16384: Apple Silicon's page size, and a multiple of x86_64's 4096.
	machoPageAlign = 16384

	machoSignID = "a.out" // matches cmd/link's own identifier; irrelevant for ad-hoc signing
)

// spliceMachOSection returns stub with a new MachOSegmentName/
// MachOSectionName section holding payload, re-signed with a fresh
// ad-hoc signature.
//
// The payload goes at __LINKEDIT's offset, shifting its content (and
// every load command into it) by a page-aligned delta — not appended at
// the old signature's spot, which would overlap __LINKEDIT and make the
// kernel refuse to run it despite `codesign --verify --strict` passing.
func spliceMachOSection(stub []byte, payload []byte) ([]byte, error) {
	f, err := macho.NewFile(bytes.NewReader(stub))
	if err != nil {
		return nil, fmt.Errorf("parsing Mach-O stub: %w", err)
	}
	defer func() { _ = f.Close() }()
	if f.Magic != macho.Magic64 {
		return nil, fmt.Errorf("unsupported Mach-O magic %#x: only 64-bit Mach-O stubs are supported", f.Magic)
	}
	for _, s := range f.Sections {
		if s.Name == MachOSectionName && s.Seg == MachOSegmentName {
			return nil, fmt.Errorf("stub already has a %s,%s section", MachOSegmentName, MachOSectionName)
		}
	}

	// Locate the load commands we'll need to patch: LC_CODE_SIGNATURE (if
	// any), __LINKEDIT/__TEXT segments, and LC_SYMTAB/LC_DYSYMTAB/
	// LC_DYLD_INFO[_ONLY] (each stores absolute offsets into __LINKEDIT).
	var linkeditSeg, textSeg *macho.Segment
	var oldSigOff, oldSigSize uint32
	haveOldSig := false
	sigCmdByteOff := -1
	linkeditCmdByteOff := -1
	symtabCmdByteOff := -1
	dysymtabCmdByteOff := -1
	dyldInfoCmdByteOff := -1
	var maxVMEnd uint64
	cursor := machoHeader64Size
	for _, l := range f.Loads {
		raw := l.Raw()
		cmd := f.ByteOrder.Uint32(raw[0:4])
		switch cmd {
		case machoLcCodeSignature:
			oldSigOff = f.ByteOrder.Uint32(raw[8:12])
			oldSigSize = f.ByteOrder.Uint32(raw[12:16])
			haveOldSig = true
			sigCmdByteOff = cursor
		case machoLcSymtab:
			symtabCmdByteOff = cursor
		case machoLcDysymtab:
			dysymtabCmdByteOff = cursor
		case machoLcDyldInfo, machoLcDyldInfoOnly:
			dyldInfoCmdByteOff = cursor
		}
		if seg, ok := l.(*macho.Segment); ok {
			if seg.Name == "__LINKEDIT" {
				linkeditSeg = seg
				linkeditCmdByteOff = cursor
			}
			if seg.Name == "__TEXT" {
				textSeg = seg
			}
			// __LINKEDIT's Memsz is about to grow substantially, so its
			// current value isn't a safe bound for a non-colliding VM address.
			if seg.Name != "__LINKEDIT" {
				if end := seg.Addr + seg.Memsz; end > maxVMEnd {
					maxVMEnd = end
				}
			}
		}
		cursor += len(raw)
	}
	if linkeditSeg == nil {
		return nil, fmt.Errorf("stub has no __LINKEDIT segment")
	}
	if textSeg == nil {
		return nil, fmt.Errorf("stub has no __TEXT segment")
	}

	oldLinkeditOff := linkeditSeg.Offset
	var codeEnd int64
	if haveOldSig {
		if int64(oldSigOff)+int64(oldSigSize) != int64(len(stub)) {
			return nil, fmt.Errorf("unexpected content after existing code signature")
		}
		codeEnd = int64(oldSigOff)
	} else {
		codeEnd = int64(oldLinkeditOff + linkeditSeg.Filesz)
		if codeEnd != int64(len(stub)) {
			return nil, fmt.Errorf("unexpected content after __LINKEDIT segment")
		}
	}
	if int64(oldLinkeditOff) > codeEnd {
		return nil, fmt.Errorf("__LINKEDIT offset is past the end of its own content")
	}
	linkeditContent := stub[oldLinkeditOff:codeEnd]

	// The new LC_SEGMENT_64 must fit in the slack between the load
	// commands and the first section's file data — verified, not assumed.
	firstSectionOff := int64(-1)
	for _, s := range f.Sections {
		if s.Size == 0 || s.Offset == 0 {
			// Offset 0 (other than the header itself) means a zero-fill
			// (S_ZEROFILL) section with no real file backing.
			continue
		}
		if firstSectionOff == -1 || int64(s.Offset) < firstSectionOff {
			firstSectionOff = int64(s.Offset)
		}
	}
	if firstSectionOff == -1 {
		return nil, fmt.Errorf("stub has no sections with file data")
	}
	neededCmdBytes := int64(machoSegmentCmdSize + machoSectionCmdSize)
	if !haveOldSig {
		neededCmdBytes += machoCodeSigCmdSize // amd64 Go builds have no existing LC_CODE_SIGNATURE to patch — insert one
	}
	availableSlack := firstSectionOff - int64(machoHeader64Size+int(f.Cmdsz))
	if neededCmdBytes > availableSlack {
		return nil, fmt.Errorf("insufficient Mach-O header slack to add a new segment command: need %d bytes, have %d",
			neededCmdBytes, availableSlack)
	}

	// delta is how far __LINKEDIT's content shifts: the new segment's
	// size, page-aligned so the relocated content lands on a page too.
	delta := alignUp64(uint64(len(payload)), machoPageAlign)
	newLinkeditOff := oldLinkeditOff + delta

	// Computed early to project __LINKEDIT's post-extension VM extent
	// before picking the new segment's address — using its pre-extension
	// size here caused a real VM collision, seen only as an execution
	// failure under Rosetta.
	sigOff := int64(oldLinkeditOff) + int64(delta) + int64(len(linkeditContent))
	sigSize := codesignSize(sigOff, machoSignID)
	if sigOff > math.MaxUint32 || sigSize > math.MaxUint32-sigOff {
		return nil, fmt.Errorf("output too large for LC_CODE_SIGNATURE's 32-bit dataoff/datasize: offset %d, size %d", sigOff, sigSize)
	}
	projectedLinkeditEnd := linkeditSeg.Addr + (uint64(sigOff) + uint64(sigSize) - newLinkeditOff)
	if projectedLinkeditEnd > maxVMEnd {
		maxVMEnd = projectedLinkeditEnd
	}

	payloadOff := int64(oldLinkeditOff)
	newVMAddr := alignUp64(maxVMEnd, machoPageAlign)
	sectionSize := uint64(len(payload))
	// Segment vmsize/filesize must be page-aligned (unaligned made Rosetta
	// refuse an otherwise valid amd64 binary); the section's own size
	// stays exact — readers trim to it, not the segment's padded size.
	segmentSize := delta

	// newCmd holds segment+section, plus a placeholder LC_CODE_SIGNATURE
	// for amd64 stubs with none to patch — its full length is the shift
	// amount used below.
	var newCmd bytes.Buffer
	writeMachOSegment64(&newCmd, f.ByteOrder, MachOSegmentName, newVMAddr, segmentSize, uint64(payloadOff), segmentSize, 0x1, 0x1, 1)
	writeMachOSection64(&newCmd, f.ByteOrder, MachOSectionName, MachOSegmentName, newVMAddr, sectionSize, uint32(payloadOff))
	newNcmd := f.Ncmd + 1
	insertingSigCmd := !haveOldSig
	codesigOffsetWithinNewCmd := newCmd.Len()
	if insertingSigCmd {
		writeMachOCodeSigCmd(&newCmd, f.ByteOrder, 0, 0) // placeholder dataoff/datasize, patched below
		newNcmd++
	}

	// Spliced into the header's slack, right before __LINKEDIT's command
	// — appending at the tail declares it out of file-offset order,
	// which Rosetta rejects even though strict validation accepts it.
	cmdRegionEnd := int(firstSectionOff)
	cmdRegion := append([]byte(nil), stub[machoHeader64Size:cmdRegionEnd]...)
	insertAt := linkeditCmdByteOff - machoHeader64Size
	origLinkeditCmdByteOff := linkeditCmdByteOff // captured before any reassignment below
	newCmdRegion := make([]byte, len(cmdRegion))
	copy(newCmdRegion, cmdRegion[:insertAt])
	copy(newCmdRegion[insertAt:], newCmd.Bytes())
	copy(newCmdRegion[insertAt+newCmd.Len():], cmdRegion[insertAt:])
	// Threshold must stay the ORIGINAL offset for every call below, not
	// whatever linkeditCmdByteOff was mutated to by an earlier call.
	shift := func(off int) int {
		if off < 0 {
			return -1
		}
		if off >= origLinkeditCmdByteOff {
			return off + newCmd.Len()
		}
		return off
	}
	linkeditCmdByteOff = shift(linkeditCmdByteOff)
	symtabCmdByteOff = shift(symtabCmdByteOff)
	dysymtabCmdByteOff = shift(dysymtabCmdByteOff)
	dyldInfoCmdByteOff = shift(dyldInfoCmdByteOff)
	if insertingSigCmd {
		sigCmdByteOff = machoHeader64Size + insertAt + codesigOffsetWithinNewCmd
	} else {
		sigCmdByteOff = shift(sigCmdByteOff)
	}

	out := append([]byte(nil), stub[:machoHeader64Size]...)
	f.ByteOrder.PutUint32(out[16:20], newNcmd) // FileHeader.Ncmd
	f.ByteOrder.PutUint32(out[20:24], f.Cmdsz+uint32(neededCmdBytes))
	out = append(out, newCmdRegion...)
	out = append(out, stub[cmdRegionEnd:oldLinkeditOff]...)
	out = append(out, payload...)
	if pad := int64(newLinkeditOff) - int64(len(out)); pad > 0 {
		out = append(out, make([]byte, pad)...)
	}
	out = append(out, linkeditContent...)

	// Shift every absolute offset into __LINKEDIT's relocated content by
	// delta; zero means "not present" and must stay untouched.
	shiftIfNonzero := func(fieldOff int) {
		v := f.ByteOrder.Uint32(out[fieldOff : fieldOff+4])
		if v != 0 {
			f.ByteOrder.PutUint32(out[fieldOff:fieldOff+4], v+uint32(delta))
		}
	}
	f.ByteOrder.PutUint64(out[linkeditCmdByteOff+40:linkeditCmdByteOff+48], newLinkeditOff) // Segment64.Offset
	if symtabCmdByteOff >= 0 {
		shiftIfNonzero(symtabCmdByteOff + 8)  // Symoff
		shiftIfNonzero(symtabCmdByteOff + 16) // Stroff
	}
	if dysymtabCmdByteOff >= 0 {
		for _, rel := range []int{32, 40, 48, 56, 64, 72} { // Tocoffset, Modtaboff, Extrefsymoff, Indirectsymoff, Extreloff, Locreloff
			shiftIfNonzero(dysymtabCmdByteOff + rel)
		}
	}
	if dyldInfoCmdByteOff >= 0 {
		for _, rel := range []int{8, 16, 24, 32, 40} { // rebase_off, bind_off, weak_bind_off, lazy_bind_off, export_off
			shiftIfNonzero(dyldInfoCmdByteOff + rel)
		}
	}

	// Patch LC_CODE_SIGNATURE before signing — those bytes are within the
	// hashed region, so patching after would corrupt the signature.
	// sigOff/sigSize were computed earlier; sanity-check against reality.
	if int64(len(out)) != sigOff {
		return nil, fmt.Errorf("internal error: assembled file length %d does not match precomputed signature offset %d", len(out), sigOff)
	}
	f.ByteOrder.PutUint32(out[sigCmdByteOff+8:sigCmdByteOff+12], uint32(sigOff))
	f.ByteOrder.PutUint32(out[sigCmdByteOff+12:sigCmdByteOff+16], uint32(sigSize))

	// __LINKEDIT's declared range must exactly cover from its (relocated)
	// start through the end of the new signature — strict validation
	// checks this, not just the page hashes.
	newLinkeditFilesz := uint64(sigOff) + uint64(sigSize) - newLinkeditOff
	f.ByteOrder.PutUint64(out[linkeditCmdByteOff+32:linkeditCmdByteOff+40], newLinkeditFilesz) // Memsz
	f.ByteOrder.PutUint64(out[linkeditCmdByteOff+48:linkeditCmdByteOff+56], newLinkeditFilesz) // Filesz

	// Now sign the whole assembled file (with every patch already
	// applied) and append the signature.
	sig := make([]byte, sigSize)
	codesignSign(sig, bytes.NewReader(out), machoSignID, sigOff, int64(textSeg.Offset), int64(textSeg.Filesz), true, haveOldSig)
	out = append(out, sig...)

	return out, nil
}

func alignUp64(v, align uint64) uint64 {
	if align == 0 {
		return v
	}
	return (v + align - 1) / align * align
}

func writeMachOSegment64(buf *bytes.Buffer, order binary.ByteOrder, name string, vmaddr, vmsize, fileoff, filesize uint64, maxprot, prot uint32, nsect uint32) {
	var nameBytes [16]byte
	copy(nameBytes[:], name)
	put32 := func(v uint32) { _ = binary.Write(buf, order, v) }
	put64 := func(v uint64) { _ = binary.Write(buf, order, v) }
	put32(machoLcSegment64)
	put32(uint32(machoSegmentCmdSize + int(nsect)*machoSectionCmdSize))
	buf.Write(nameBytes[:])
	put64(vmaddr)
	put64(vmsize)
	put64(fileoff)
	put64(filesize)
	put32(maxprot)
	put32(prot)
	put32(nsect)
	put32(0) // flags
}

func writeMachOCodeSigCmd(buf *bytes.Buffer, order binary.ByteOrder, dataoff, datasize uint32) {
	put32 := func(v uint32) { _ = binary.Write(buf, order, v) }
	put32(machoLcCodeSignature)
	put32(machoCodeSigCmdSize)
	put32(dataoff)
	put32(datasize)
}

func writeMachOSection64(buf *bytes.Buffer, order binary.ByteOrder, sectName, segName string, addr, size uint64, offset uint32) {
	var sectNameBytes, segNameBytes [16]byte
	copy(sectNameBytes[:], sectName)
	copy(segNameBytes[:], segName)
	put32 := func(v uint32) { _ = binary.Write(buf, order, v) }
	put64 := func(v uint64) { _ = binary.Write(buf, order, v) }
	buf.Write(sectNameBytes[:])
	buf.Write(segNameBytes[:])
	put64(addr)
	put64(size)
	put32(offset)
	put32(0) // align
	put32(0) // reloff
	put32(0) // nreloc
	put32(0) // flags (S_REGULAR, no special attributes — inert data)
	put32(0) // reserved1
	put32(0) // reserved2
	put32(0) // reserved3
}
