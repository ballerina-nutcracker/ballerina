// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the Go distribution's LICENSE file
// (https://go.googlesource.com/go/+/refs/heads/master/LICENSE).
//
// Ported from the Go toolchain's cmd/internal/codesign (unexported, so
// not importable directly) — the same ad-hoc Mach-O signing algorithm
// cmd/link itself uses for darwin output. Adapted only to replace the
// cmd/internal/hash dependency with crypto/sha256 directly; the signing
// algorithm and byte layout are unchanged.

package splice

import (
	"crypto/sha256"
	"encoding/binary"
	"io"
)

const (
	pageSizeBits = 12
	pageSize     = 1 << pageSizeBits
)

// Constants and struct layouts are from
// https://opensource.apple.com/source/xnu/xnu-4903.270.47/osfmk/kern/cs_blobs.h
const (
	csmagicCodeDirectory     = 0xfade0c02 // CodeDirectory blob
	csmagicEmbeddedSignature = 0xfade0cc0 // embedded form of signature data
	csslotCodeDirectory      = 0          // slot index for CodeDirectory
	csHashTypeSHA256         = 2
	csExecSegMainBinary      = 0x1 // executable segment denotes main binary
	sha256Size               = sha256.Size
)

type blob struct {
	typ    uint32 // type of entry
	offset uint32 // offset of entry
	// data follows
}

func (b *blob) put(out []byte) []byte {
	out = put32be(out, b.typ)
	out = put32be(out, b.offset)
	return out
}

const blobSize = 2 * 4

type superBlob struct {
	magic  uint32 // magic number
	length uint32 // total length of superBlob
	count  uint32 // number of index entries following
	// blobs []blob
}

func (s *superBlob) put(out []byte) []byte {
	out = put32be(out, s.magic)
	out = put32be(out, s.length)
	out = put32be(out, s.count)
	return out
}

const superBlobSize = 3 * 4

type codeDirectory struct {
	magic         uint32 // magic number (csmagicCodeDirectory)
	length        uint32 // total length of codeDirectory blob
	version       uint32 // compatibility version
	flags         uint32 // setup and mode flags
	hashOffset    uint32 // offset of hash slot element at index zero
	identOffset   uint32 // offset of identifier string
	nSpecialSlots uint32 // number of special hash slots
	nCodeSlots    uint32 // number of ordinary (code) hash slots
	codeLimit     uint32 // limit to main image signature range
	hashSize      uint8  // size of each hash in bytes
	hashType      uint8  // type of hash (cdHashType* constants)
	_pad1         uint8  // unused (must be zero)
	pageSize      uint8  // log2(page size in bytes); 0 => infinite
	_pad2         uint32 // unused (must be zero)
	scatterOffset uint32
	teamOffset    uint32
	_pad3         uint32
	codeLimit64   uint64
	execSegBase   uint64
	execSegLimit  uint64
	execSegFlags  uint64
	// data follows
}

func (c *codeDirectory) put(out []byte) []byte {
	out = put32be(out, c.magic)
	out = put32be(out, c.length)
	out = put32be(out, c.version)
	out = put32be(out, c.flags)
	out = put32be(out, c.hashOffset)
	out = put32be(out, c.identOffset)
	out = put32be(out, c.nSpecialSlots)
	out = put32be(out, c.nCodeSlots)
	out = put32be(out, c.codeLimit)
	out = put8(out, c.hashSize)
	out = put8(out, c.hashType)
	out = put8(out, c._pad1)
	out = put8(out, c.pageSize)
	out = put32be(out, c._pad2)
	out = put32be(out, c.scatterOffset)
	out = put32be(out, c.teamOffset)
	out = put32be(out, c._pad3)
	out = put64be(out, c.codeLimit64)
	out = put64be(out, c.execSegBase)
	out = put64be(out, c.execSegLimit)
	out = put64be(out, c.execSegFlags)
	return out
}

const codeDirectorySize = 13*4 + 4 + 4*8

func put32be(b []byte, x uint32) []byte { binary.BigEndian.PutUint32(b, x); return b[4:] }
func put64be(b []byte, x uint64) []byte { binary.BigEndian.PutUint64(b, x); return b[8:] }
func put8(b []byte, x uint8) []byte     { b[0] = x; return b[1:] }
func puts(b, s []byte) []byte           { n := copy(b, s); return b[n:] }

// An empty Requirements blob plus nSpecialSlots=2 (slot -2 Requirements,
// slot -1 Info.plist-absent-as-zero) match what `codesign -s -` produces
// but Go's linker skips. Required: a from-scratch amd64 signature
// missing either piece passes `codesign --verify --strict` but Rosetta
// rejects it at load time.
const (
	csmagicRequirements = 0xfade0c01
	csslotRequirements  = 2
	nSpecialSlots       = 2
)

// emptyRequirementsBlob is a complete, valid, empty CSMAGIC_REQUIREMENTS
// blob: magic + length=12 + count=0, no requirement entries follow.
var emptyRequirementsBlob = []byte{
	0xfa, 0xde, 0x0c, 0x01,
	0x00, 0x00, 0x00, 0x0c,
	0x00, 0x00, 0x00, 0x00,
}

// codesignSize computes the size of the ad-hoc code signature. id is the
// identifier used for signing (a field in the CodeDirectory blob, with
// no significance in ad-hoc signing).
func codesignSize(codeSize int64, id string) int64 {
	nhashes := (codeSize + pageSize - 1) / pageSize
	idOff := int64(codeDirectorySize)
	hashOff := idOff + int64(len(id)+1) + nSpecialSlots*sha256Size
	cdirLen := hashOff + nhashes*sha256Size
	headerSize := int64(superBlobSize + 2*blobSize)
	return headerSize + cdirLen + int64(len(emptyRequirementsBlob))
}

// csLinkerSigned must only be set when re-signing an already
// linker-signed (arm64) binary — set on a from-scratch amd64 signature,
// it passes strict verification but Rosetta rejects it at load time.
const (
	csAdhoc        = 0x2
	csLinkerSigned = 0x20000
)

// codesignSign generates an ad-hoc code signature and writes it to out,
// which must have length at least codesignSize(codeSize, id). data is
// the unsigned file content (size codeSize). textOff/textSize locate
// __TEXT. isMain is always true here (balrt/bal are never dylibs).
// linkerSigned must only be true when re-signing an already
// linker-signed binary.
func codesignSign(out []byte, data io.Reader, id string, codeSize, textOff, textSize int64, isMain, linkerSigned bool) {
	nhashes := (codeSize + pageSize - 1) / pageSize
	idOff := int64(codeDirectorySize)
	hashOff := idOff + int64(len(id)+1) + nSpecialSlots*sha256Size
	cdirLen := hashOff + nhashes*sha256Size
	headerSize := int64(superBlobSize + 2*blobSize)
	sz := headerSize + cdirLen + int64(len(emptyRequirementsBlob))

	flags := uint32(csAdhoc)
	if linkerSigned {
		flags |= csLinkerSigned
	}

	sb := superBlob{
		magic:  csmagicEmbeddedSignature,
		length: uint32(sz),
		count:  2,
	}
	cdirBlob := blob{typ: csslotCodeDirectory, offset: uint32(headerSize)}
	reqBlob := blob{typ: csslotRequirements, offset: uint32(headerSize + cdirLen)}
	cdir := codeDirectory{
		magic:         csmagicCodeDirectory,
		length:        uint32(cdirLen),
		version:       0x20400,
		flags:         flags,
		hashOffset:    uint32(hashOff),
		identOffset:   uint32(idOff),
		nSpecialSlots: nSpecialSlots,
		nCodeSlots:    uint32(nhashes),
		codeLimit:     uint32(codeSize),
		hashSize:      sha256Size,
		hashType:      csHashTypeSHA256,
		pageSize:      uint8(pageSizeBits),
		execSegBase:   uint64(textOff),
		execSegLimit:  uint64(textSize),
	}
	if isMain {
		cdir.execSegFlags = csExecSegMainBinary
	}

	outp := out
	outp = sb.put(outp)
	outp = cdirBlob.put(outp)
	outp = reqBlob.put(outp)
	outp = cdir.put(outp)
	outp = puts(outp, []byte(id+"\000"))

	// Special slot hashes, in slot order -2 then -1 (i.e. Requirements
	// first, then Info.plist), landing immediately before the ordinary
	// code hashes at hashOffset — matching cs_blobs.h's special-slots-
	// grow-backward-from-hashOffset convention.
	reqHash := sha256.Sum256(emptyRequirementsBlob)
	outp = puts(outp, reqHash[:])               // slot -2: Requirements
	outp = puts(outp, make([]byte, sha256Size)) // slot -1: Info.plist (absent -> zero)

	var buf [pageSize]byte
	p := 0
	for p < int(codeSize) {
		n, err := io.ReadFull(data, buf[:])
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			panic(err)
		}
		if p+n > int(codeSize) {
			n = int(codeSize) - p
		}
		p += n
		h := sha256.Sum256(buf[:n])
		outp = puts(outp, h[:])
	}

	puts(outp, emptyRequirementsBlob)
}
