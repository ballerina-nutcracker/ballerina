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
// Payload-round-trip, section-preservation, strict-codesign-validation,
// and execution coverage for Mach-O live in
// corpus/cli_integration_test.go (TestBalBuildDarwin*), testing the same
// code through the real bal build CLI, on both the linker-signed
// (arm64, host) and from-scratch (amd64, cross-compiled) signing paths.
// This file only covers rejection paths a real balrt stub can't
// exercise.

package splice

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalZeroSlackMachO64 hand-builds a minimal, valid 64-bit Mach-O
// binary (one __TEXT segment with one section, one empty __LINKEDIT
// segment, no existing signature) whose load-command area leaves
// exactly zero slack for a new segment command, to exercise
// spliceMachOSection's "insufficient header slack" rejection without a
// real toolchain build.
func minimalZeroSlackMachO64(t *testing.T) []byte {
	t.Helper()

	const (
		textCmdLen     = machoSegmentCmdSize + machoSectionCmdSize // 152
		linkeditCmdLen = machoSegmentCmdSize                       // 72, no sections
		cmdsz          = textCmdLen + linkeditCmdLen               // 224
		sectionDataOff = machoHeader64Size + cmdsz                 // 256: zero slack
		sectionDataLen = 16
		linkeditOff    = sectionDataOff + sectionDataLen // 272
		linkeditLen    = 8
	)

	var buf bytes.Buffer
	order := binary.LittleEndian
	putU32 := func(v uint32) { _ = binary.Write(&buf, order, v) }
	putU32(0xfeedfacf)    // MH_MAGIC_64
	putU32(0x0100000c)    // CPU_TYPE_ARM64
	putU32(0)             // cpusubtype
	putU32(2)             // MH_EXECUTE
	putU32(2)             // ncmds
	putU32(uint32(cmdsz)) // sizeofcmds
	putU32(0)             // flags
	putU32(0)             // reserved

	writeMachOSegment64(&buf, order, "__TEXT", 0x100000000, 0x4000, 0, sectionDataOff+sectionDataLen, 0x7, 0x5, 1)
	writeMachOSection64(&buf, order, "__text", "__TEXT", 0x100000000+sectionDataOff, sectionDataLen, sectionDataOff)
	writeMachOSegment64(&buf, order, "__LINKEDIT", 0x100004000, linkeditLen, linkeditOff, linkeditLen, 0x1, 0x1, 0)

	if buf.Len() != sectionDataOff {
		t.Fatalf("internal error building fixture: header+cmds = %d, want %d", buf.Len(), sectionDataOff)
	}
	buf.Write(make([]byte, sectionDataLen)) // fake __text bytes
	buf.Write(make([]byte, linkeditLen))    // fake __LINKEDIT content

	return buf.Bytes()
}

func TestEmbedMachO_FailsClearlyWhenHeaderSlackInsufficient(t *testing.T) {
	stubPath := filepath.Join(t.TempDir(), "zero-slack-macho")
	if err := os.WriteFile(stubPath, minimalZeroSlackMachO64(t), 0o755); err != nil {
		t.Fatalf("writing synthetic stub: %v", err)
	}
	f, err := macho.Open(stubPath)
	if err != nil {
		t.Fatalf("synthetic zero-slack fixture doesn't parse as Mach-O: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing synthetic fixture: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "packed")
	err = EmbedMachO(stubPath, []byte("payload"), outPath)
	if err == nil {
		t.Fatal("expected an error packing a stub with zero header slack")
	}
	if !strings.Contains(err.Error(), "insufficient Mach-O header slack") {
		t.Errorf("expected an 'insufficient Mach-O header slack' error, got: %v", err)
	}
}

func TestEmbedMachO_RejectsAlreadyPackedInput(t *testing.T) {
	packedOnce := filepath.Join(t.TempDir(), "packed-once")
	if err := EmbedMachO(darwinArm64StubPath, []byte("payload"), packedOnce); err != nil {
		t.Fatalf("first EmbedMachO: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "packed-twice")
	err := EmbedMachO(packedOnce, []byte("payload"), outPath)
	if err == nil {
		t.Fatal("expected an error packing an already-packed stub")
	}
	wantSubstr := fmt.Sprintf("stub already has a %s,%s section", MachOSegmentName, MachOSectionName)
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected error to contain %q, got: %v", wantSubstr, err)
	}
}
