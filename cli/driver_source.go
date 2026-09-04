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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package cli

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const devDriverDirName = "ballerina-driver-src-dev"

// ExtractDriverSource extracts the embedded CLI driver module into the build
// cache. The extracted directory is immutable once complete — for a given
// version (or, in dev builds, a given content hash) the embedded source is
// always byte-identical, so concurrent `bal` processes never need to delete
// or rewrite an existing, complete directory. Each writer instead extracts
// into its own private staging directory and atomically renames it into
// place, so a concurrent reader of the target directory only ever sees it
// fully absent or fully populated, never partially written or mid-delete.
func ExtractDriverSource(cacheRoot, version string) (string, error) {
	source := DriverSource()
	if source == nil {
		return "", errors.New("CLI driver source is not embedded in native interpreter builds")
	}
	if version == "dev" {
		hash, err := driverSourceHash(source)
		if err != nil {
			return "", fmt.Errorf("hashing embedded CLI driver source: %w", err)
		}
		dir := filepath.Join(os.TempDir(), devDriverDirName+"-"+hash)
		return installDriverSource(dir, os.TempDir(), source)
	}

	dir := filepath.Join(cacheRoot, "interpreter-src", version)
	return installDriverSource(dir, filepath.Dir(dir), source)
}

// installDriverSource returns dir unchanged if it's already a complete
// extraction; otherwise it extracts source into a fresh staging directory
// under stagingParent and atomically renames it to dir. If another process
// wins the race and installs dir first, the rename fails harmlessly — since
// dir is content-addressed (by version or content hash), that directory is
// guaranteed byte-identical to what this call would have installed, so it
// reuses it instead of erroring.
func installDriverSource(dir, stagingParent string, source fs.FS) (string, error) {
	if extractedDriverSourceComplete(dir) {
		return dir, nil
	}

	if err := os.MkdirAll(stagingParent, 0o755); err != nil {
		return "", fmt.Errorf("creating CLI driver source cache dir: %w", err)
	}
	staging, err := os.MkdirTemp(stagingParent, filepath.Base(dir)+"-staging-*")
	if err != nil {
		return "", fmt.Errorf("creating staging dir for CLI driver source: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if err := extractDriverSource(staging, source); err != nil {
		return "", fmt.Errorf("extracting CLI driver source: %w", err)
	}
	if err := os.Rename(staging, dir); err != nil {
		if extractedDriverSourceComplete(dir) {
			return dir, nil
		}
		// dir exists but is incomplete — not something a concurrent reader
		// could be relying on (this function never leaves dir in that state
		// itself; it only ever installs a complete extraction via the
		// rename above), so it's stale/corrupted and safe to replace.
		if err := os.RemoveAll(dir); err != nil {
			return "", fmt.Errorf("clearing incomplete CLI driver source cache: %w", err)
		}
		if err := os.Rename(staging, dir); err != nil {
			return "", fmt.Errorf("installing CLI driver source cache: %w", err)
		}
	}
	return dir, nil
}

func driverSourceHash(source fs.FS) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	h.Write([]byte(driverWorkspace))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func extractDriverSource(dir string, source fs.FS) error {
	if err := os.MkdirAll(filepath.Join(dir, "cli"), 0o755); err != nil {
		return err
	}
	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, "cli", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "go.work"), []byte(driverWorkspace), 0o644)
}

func extractedDriverSourceComplete(dir string) bool {
	for _, name := range []string{
		"go.work",
		filepath.Join("cli", "go.mod"),
		filepath.Join("cli", "cmd"),
		filepath.Join("cli", "internal", "balrt"),
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

const driverWorkspace = "go 1.26\n\nuse ./cli\n"
