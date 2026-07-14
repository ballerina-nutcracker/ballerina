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

// Package interpsrc embeds the interpreter Go source tree into the released
// bal binary so that end users can build native interpreter variants without
// needing to check out the ballerina-lang-go repository separately.
//
// When building the native interpreter itself (go build -tags native_interp),
// this file is excluded and interpsrc_stub.go is compiled instead, so the
// recursive embed is not included in the native interpreter binary.
package interpsrc

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// The embed includes all Go source packages needed to build the interpreter,
// plus go.mod and go.sum. parser/testdata (270 MB of test fixtures),
// corpus/, samples/, and compiler-tools/ are intentionally excluded to keep binary size small.

//go:embed go.mod go.sum interpsrc_stub.go
//go:embed ast bir cli common context decimal desugar lib model platform projects runtime semantics semtypes tools values
//go:embed parser/*.go parser/nodes.json parser/common parser/tree
var src embed.FS

// devDirName is the fixed cache directory (under the OS temp dir) used for
// local "dev" builds, where Version is never bumped between builds.
const devDirName = "ballerina-interpreter-src-dev"

// ExtractTo writes the embedded source tree into a cache directory and
// returns that path.
//
// For a real release version, the tree is extracted once to
// <cacheRoot>/interpreter-src/<version>/ and reused indefinitely: release
// content is immutable per version, so presence alone is a safe cache check.
//
// For local "dev" builds, version is always "dev", so instead the tree is
// extracted to a fixed path under the OS temp directory, keyed by a content
// hash of the embedded tree: unchanged content reuses the existing
// extraction (avoiding repeated extractions across a dev session), and
// changed content (a rebuilt bal binary with edited source) replaces it in
// place rather than accumulating stale copies.
func ExtractTo(cacheRoot, version string) (string, error) {
	if version == "dev" {
		return extractDev()
	}
	dir := filepath.Join(cacheRoot, "interpreter-src", version)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating interpreter source cache: %w", err)
	}
	if err := extractAll(dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("extracting interpreter source: %w", err)
	}
	return dir, nil
}

// extractDev extracts the embedded source tree to a fixed path under the OS
// temp directory. The extraction is keyed by a content hash stored in a
// sibling marker file: a matching hash skips re-extraction, while a mismatch
// (or a missing/removed extraction) replaces the directory in place, so
// repeated local rebuilds never leave multiple stale copies behind.
func extractDev() (string, error) {
	hash, err := contentHash()
	if err != nil {
		return "", fmt.Errorf("hashing embedded interpreter source: %w", err)
	}

	dir := filepath.Join(os.TempDir(), devDirName)
	hashFile := dir + ".hash"
	if existing, err := os.ReadFile(hashFile); err == nil && string(existing) == hash {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
	}

	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("clearing stale dev interpreter source cache: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating dev interpreter source cache: %w", err)
	}
	if err := extractAll(dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("extracting interpreter source: %w", err)
	}
	if err := os.WriteFile(hashFile, []byte(hash), 0o644); err != nil {
		return "", fmt.Errorf("writing dev interpreter source cache hash marker: %w", err)
	}
	return dir, nil
}

// contentHash returns a deterministic SHA-256 hex digest over every embedded
// file's path and content. fs.WalkDir visits entries in lexical order, so
// the result is stable across runs and changes whenever the embedded source
// tree changes (e.g. a local rebuild with edited source).
func contentHash() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00", p)
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func extractAll(dst string) error {
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
