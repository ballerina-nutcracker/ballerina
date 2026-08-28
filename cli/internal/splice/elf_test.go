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
// Payload-round-trip, section-preservation, and execution coverage for
// ELF live in corpus/cli_integration_test.go (TestBalBuildLinuxTarget*),
// testing the same code through the real bal build CLI. This file only
// covers rejection paths a real balrt stub can't exercise.

package splice

import (
	"os"
	"path/filepath"
	"testing"
)

// minimalELF32Header builds just enough of a valid 32-bit little-endian
// ELF header (e_shnum=e_phnum=0, so elf.NewFile parses it with zero
// sections/programs) to exercise spliceELFSection's ELFCLASS64 rejection
// without a real 32-bit toolchain build.
func minimalELF32Header() []byte {
	h := make([]byte, 52) // ELF32 header size
	copy(h[0:4], []byte{0x7f, 'E', 'L', 'F'})
	h[4] = 1 // EI_CLASS = ELFCLASS32
	h[5] = 1 // EI_DATA = ELFDATA2LSB
	h[6] = 1 // EI_VERSION
	le := func(off, n int, v uint32) {
		for i := 0; i < n; i++ {
			h[off+i] = byte(v >> (8 * i))
		}
	}
	le(16, 2, 2)  // e_type = ET_EXEC
	le(18, 2, 3)  // e_machine = EM_386
	le(20, 4, 1)  // e_version
	le(36, 2, 52) // e_ehsize
	le(38, 2, 32) // e_phentsize
	le(42, 2, 40) // e_shentsize
	return h
}

func TestEmbedELF_RejectsNon64BitOrMalformedStub(t *testing.T) {
	t.Run("not an ELF file at all", func(t *testing.T) {
		stubPath := filepath.Join(t.TempDir(), "stub")
		if err := os.WriteFile(stubPath, []byte("not an elf file"), 0o755); err != nil {
			t.Fatalf("writing stub: %v", err)
		}
		outPath := filepath.Join(t.TempDir(), "packed")
		if err := EmbedELF(stubPath, []byte("payload"), outPath); err == nil {
			t.Fatal("expected an error packing a non-ELF stub")
		}
	})

	t.Run("32-bit ELF", func(t *testing.T) {
		stubPath := filepath.Join(t.TempDir(), "fake-32bit-elf")
		if err := os.WriteFile(stubPath, minimalELF32Header(), 0o755); err != nil {
			t.Fatalf("writing synthetic 32-bit ELF: %v", err)
		}
		outPath := filepath.Join(t.TempDir(), "packed")
		if err := EmbedELF(stubPath, []byte("payload"), outPath); err == nil {
			t.Fatal("expected an error packing a 32-bit ELF stub")
		}
	})
}

func TestEmbedELF_RejectsAlreadyPackedInput(t *testing.T) {
	packedOnce := filepath.Join(t.TempDir(), "packed-once")
	if err := EmbedELF(linuxAmd64StubPath, []byte("payload"), packedOnce); err != nil {
		t.Fatalf("first EmbedELF: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "packed-twice")
	if err := EmbedELF(packedOnce, []byte("payload"), outPath); err == nil {
		t.Fatal("expected an error packing an already-packed stub")
	}
}
