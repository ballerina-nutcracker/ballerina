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
//
// Payload-round-trip, PE-validity, and execution coverage for PE live in
// corpus/cli_integration_test.go (TestBalBuildWindowsTarget*), testing
// the same code through the real bal build CLI. This file only covers
// rejection paths a real balrt stub can't exercise.

package splice

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalZeroSlackPE64 hand-builds a minimal, valid 64-bit PE binary
// with one section and SizeOfHeaders set to leave exactly zero room for
// a second section-header entry, to exercise splicePESection's
// "insufficient header slack" rejection without a real toolchain build
// (PE headers are small enough to hand-synthesize, unlike Mach-O).
func minimalZeroSlackPE64(t *testing.T) []byte {
	t.Helper()

	const (
		dosHeaderSize = 64
		fileAlign     = 0x200
		sectionAlign  = 0x1000
	)

	elfanew := uint32(dosHeaderSize)
	optHeaderSize := binary.Size(pe.OptionalHeader64{})
	sectionTableOff := dosHeaderSize + 4 + 20 + optHeaderSize
	sizeOfHeaders := uint32(sectionTableOff + peSectionHeaderSize) // exactly 1 entry, zero slack for a 2nd

	var buf bytes.Buffer
	dosHeader := make([]byte, dosHeaderSize)
	dosHeader[0], dosHeader[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(dosHeader[0x3C:], elfanew)
	buf.Write(dosHeader)
	buf.WriteString("PE\x00\x00")

	coff := pe.FileHeader{
		Machine:              pe.IMAGE_FILE_MACHINE_AMD64,
		NumberOfSections:     1,
		SizeOfOptionalHeader: uint16(optHeaderSize),
		Characteristics:      0x0002, // IMAGE_FILE_EXECUTABLE_IMAGE
	}
	if err := binary.Write(&buf, binary.LittleEndian, coff); err != nil {
		t.Fatalf("encoding COFF header: %v", err)
	}

	opt := pe.OptionalHeader64{
		Magic:                 0x20b, // PE32+
		AddressOfEntryPoint:   sectionAlign,
		BaseOfCode:            sectionAlign,
		ImageBase:             0x140000000,
		SectionAlignment:      sectionAlign,
		FileAlignment:         fileAlign,
		MajorSubsystemVersion: 6,
		SizeOfImage:           2 * sectionAlign,
		SizeOfHeaders:         sizeOfHeaders,
		Subsystem:             3, // IMAGE_SUBSYSTEM_WINDOWS_CUI
		SizeOfStackReserve:    0x100000,
		SizeOfStackCommit:     0x1000,
		SizeOfHeapReserve:     0x100000,
		SizeOfHeapCommit:      0x1000,
		NumberOfRvaAndSizes:   16,
	}
	if err := binary.Write(&buf, binary.LittleEndian, opt); err != nil {
		t.Fatalf("encoding optional header: %v", err)
	}

	var textName [8]byte
	copy(textName[:], ".text")
	sec := pe.SectionHeader32{
		Name:             textName,
		VirtualSize:      0x10,
		VirtualAddress:   sectionAlign,
		SizeOfRawData:    fileAlign,
		PointerToRawData: sizeOfHeaders,
		Characteristics:  uint32(pe.IMAGE_SCN_CNT_CODE | pe.IMAGE_SCN_MEM_EXECUTE | pe.IMAGE_SCN_MEM_READ),
	}
	if err := binary.Write(&buf, binary.LittleEndian, sec); err != nil {
		t.Fatalf("encoding section header: %v", err)
	}

	if pad := int(sizeOfHeaders) - buf.Len(); pad > 0 {
		buf.Write(make([]byte, pad))
	}
	buf.Write(make([]byte, fileAlign)) // .text's raw data, zero-filled

	return buf.Bytes()
}

func TestEmbedPE_FailsClearlyWhenHeaderSlackInsufficient(t *testing.T) {
	stubPath := filepath.Join(t.TempDir(), "zero-slack.exe")
	if err := os.WriteFile(stubPath, minimalZeroSlackPE64(t), 0o755); err != nil {
		t.Fatalf("writing synthetic stub: %v", err)
	}
	f, err := pe.Open(stubPath)
	if err != nil {
		t.Fatalf("synthetic zero-slack fixture doesn't parse as PE: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing synthetic fixture: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "packed.exe")
	err = EmbedPE(stubPath, []byte("payload"), outPath)
	if err == nil {
		t.Fatal("expected an error packing a stub with zero header slack")
	}
	if !strings.Contains(err.Error(), "insufficient PE header slack") {
		t.Errorf("expected an 'insufficient PE header slack' error, got: %v", err)
	}
}

func TestEmbedPE_RejectsAlreadyPackedInput(t *testing.T) {
	packedOnce := filepath.Join(t.TempDir(), "packed-once.exe")
	if err := EmbedPE(windowsAmd64StubPath, []byte("payload"), packedOnce); err != nil {
		t.Fatalf("first EmbedPE: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "packed-twice.exe")
	if err := EmbedPE(packedOnce, []byte("payload"), outPath); err == nil {
		t.Fatal("expected an error packing an already-packed stub")
	}
}
