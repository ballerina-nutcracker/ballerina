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

// Package splice embeds a bal build payload as a genuinely declared
// section in a stub binary — never raw trailing bytes, which invalidate
// Go's arm64 ad-hoc signature under `codesign --verify --strict`. One
// splicer per format: ELF (linux), PE (windows), Mach-O (darwin, also
// re-signed). Readers live in cli/internal/executable's embed_*.go.
//
// Named splice, not pack, to avoid confusion with bal's own `pack`
// command, which produces unrelated .bala archives.
package splice

import (
	"fmt"
	"os"
	"path/filepath"
)

// EmbedELF embeds payload into stub (a 64-bit ELF binary) as a new
// ELFSectionName section and atomically writes the result to outPath.
func EmbedELF(stubPath string, payload []byte, outPath string) error {
	return embed(stubPath, outPath, func(stub []byte) ([]byte, error) {
		return spliceELFSection(stub, ELFSectionName, payload)
	})
}

// EmbedPE embeds payload into stub (a 64-bit PE binary) as a new
// PESectionName section and atomically writes the result to outPath.
func EmbedPE(stubPath string, payload []byte, outPath string) error {
	return embed(stubPath, outPath, func(stub []byte) ([]byte, error) {
		return splicePESection(stub, PESectionName, payload)
	})
}

// EmbedMachO embeds payload into stub (a 64-bit Mach-O binary) as a new
// MachOSegmentName/MachOSectionName section, re-signs the result with a
// fresh ad-hoc signature, and atomically writes it to outPath.
func EmbedMachO(stubPath string, payload []byte, outPath string) error {
	return embed(stubPath, outPath, func(stub []byte) ([]byte, error) {
		return spliceMachOSection(stub, payload)
	})
}

// Embed dispatches to EmbedELF/EmbedPE/EmbedMachO by targetOS. The
// default case is unreachable today (executable.ValidatePlatform only
// ever allows linux/windows/darwin) — it exists so a future platform
// added without a matching case here fails loudly, not silently.
func Embed(stubPath string, payload []byte, outPath string, targetOS string) error {
	switch targetOS {
	case "darwin":
		return EmbedMachO(stubPath, payload, outPath)
	case "linux":
		return EmbedELF(stubPath, payload, outPath)
	case "windows":
		return EmbedPE(stubPath, payload, outPath)
	default:
		return fmt.Errorf("no packer implemented for platform %s", targetOS)
	}
}

// embed runs splice over stub's bytes and atomically writes the result
// to outPath, creating parent dirs and making it executable. Writes to
// a sibling temp file and renames on success, so a partial failure
// can't corrupt a pre-existing outPath.
func embed(stubPath, outPath string, splice func(stub []byte) ([]byte, error)) error {
	stub, err := os.ReadFile(stubPath)
	if err != nil {
		return fmt.Errorf("reading runner stub: %w", err)
	}

	spliced, err := splice(stub)
	if err != nil {
		return fmt.Errorf("embedding payload: %w", err)
	}

	outDir := filepath.Dir(outPath)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	tmp, err := os.CreateTemp(outDir, ".bal-pack-*")
	if err != nil {
		return fmt.Errorf("creating temp output file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op once renamed over outPath

	if _, err := tmp.Write(spliced); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing packed output: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp output file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("setting output file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return fmt.Errorf("renaming output file into place: %w", err)
	}
	return nil
}
