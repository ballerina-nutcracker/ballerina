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

//go:build !native_interp

package interpsrc

import (
	"os"
	"path/filepath"
	"testing"
)

// The dev-path tests below share the fixed OS-temp cache location (by
// design — that's the behavior under test) and so cannot run in parallel
// with each other.

func TestContentHash_Deterministic(t *testing.T) {
	t.Parallel()
	h1, err := contentHash()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := contentHash()
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("contentHash not deterministic: %q vs %q", h1, h2)
	}
	if h1 == "" {
		t.Error("contentHash must not be empty")
	}
}

func TestExtractTo_DevReusesUnchangedContent(t *testing.T) {
	dir1, err := ExtractTo(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("first ExtractTo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir1, "go.mod")); err != nil {
		t.Fatalf("go.mod must exist after extraction: %v", err)
	}

	dir2, err := ExtractTo(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("second ExtractTo: %v", err)
	}
	if dir1 != dir2 {
		t.Errorf("dev extraction path must be stable: %q vs %q", dir1, dir2)
	}
}

func TestExtractTo_DevReplacesOnHashMismatch(t *testing.T) {
	dir, err := ExtractTo(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("ExtractTo: %v", err)
	}

	// Simulate a stale cache: a leftover file that must not survive a replace,
	// and a hash marker that no longer matches the current embedded content.
	stray := filepath.Join(dir, "stray-leftover-file.go")
	if err := os.WriteFile(stray, []byte("package stale"), 0o644); err != nil {
		t.Fatalf("writing stray file: %v", err)
	}
	hashFile := dir + ".hash"
	if err := os.WriteFile(hashFile, []byte("not-a-real-hash"), 0o644); err != nil {
		t.Fatalf("tampering hash marker: %v", err)
	}

	dir2, err := ExtractTo(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("ExtractTo after tampering: %v", err)
	}
	if dir2 != dir {
		t.Fatalf("dev extraction path must stay fixed: %q vs %q", dir, dir2)
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Error("stale extraction must be replaced, not merged with — stray file should be gone")
	}
	if _, err := os.Stat(filepath.Join(dir2, "go.mod")); err != nil {
		t.Fatalf("go.mod must exist after replace: %v", err)
	}

	want, err := contentHash()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(hashFile)
	if err != nil {
		t.Fatalf("reading hash marker after replace: %v", err)
	}
	if string(got) != want {
		t.Errorf("hash marker not updated after replace: got %q, want %q", got, want)
	}
}

func TestExtractTo_DevRecoversFromMissingGoMod(t *testing.T) {
	dir, err := ExtractTo(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("ExtractTo: %v", err)
	}

	// Corrupt the extraction without touching the hash marker, so the hash
	// still matches but the extracted tree itself is incomplete.
	if err := os.Remove(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("removing go.mod: %v", err)
	}

	dir2, err := ExtractTo(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("ExtractTo after corruption: %v", err)
	}
	if dir2 != dir {
		t.Fatalf("dev extraction path must stay fixed: %q vs %q", dir, dir2)
	}
	if _, err := os.Stat(filepath.Join(dir2, "go.mod")); err != nil {
		t.Errorf("go.mod must be re-extracted when missing despite matching hash: %v", err)
	}
}
