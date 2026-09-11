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
	"time"
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
		// cacheRoot (not the shared, world-writable os.TempDir()) keeps this
		// content-addressed by hash the same way the release path is
		// content-addressed by version — a predictable path under a shared
		// temp dir could be pre-created by another local user/process with
		// arbitrary content, which extractedDriverSourceComplete's
		// marker-file check wouldn't catch before it's trusted and fed to
		// `go build`.
		dir := filepath.Join(cacheRoot, "interpreter-src", devDriverDirName+"-"+hash)
		return installDriverSource(dir, filepath.Dir(dir), source)
	}

	dir := filepath.Join(cacheRoot, "interpreter-src", version)
	return installDriverSource(dir, filepath.Dir(dir), source)
}

// installDriverSource returns dir unchanged if it's already a complete
// extraction; otherwise it extracts source into a fresh staging directory
// under stagingParent, then — holding a lock scoped to dir — replaces
// whatever is at dir (absent or incomplete) with the staging copy. Since
// dir is content-addressed (by version or content hash), any complete dir
// is always byte-identical to what this call would install, so a
// concurrent installer that wins the race is reused rather than treated
// as an error.
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

	// The whole install decision for dir — recheck, remove any stale
	// remnant, rename staging into place — is serialized across every
	// caller (not just ones that fall back after a failed first attempt)
	// with an inter-process lock. Locking only the fallback path isn't
	// enough: a lock-free caller's own unguarded rename could otherwise
	// slip into the gap between a lock-holder's removal of an incomplete
	// dir and that holder's own replacement rename, corrupting the result.
	unlock, err := lockDriverSourceDir(dir)
	if err != nil {
		return "", fmt.Errorf("locking CLI driver source cache: %w", err)
	}
	defer unlock()

	if extractedDriverSourceComplete(dir) {
		return dir, nil
	}
	// dir is absent or incomplete — not something a concurrent reader
	// could be relying on (this function never leaves dir in that state
	// itself; it only ever installs a complete extraction via the rename
	// below), so it's stale/corrupted and, now serialized against every
	// other installer, safe to replace. RemoveAll on an absent dir is a
	// harmless no-op.
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("clearing incomplete CLI driver source cache: %w", err)
	}
	if err := os.Rename(staging, dir); err != nil {
		return "", fmt.Errorf("installing CLI driver source cache: %w", err)
	}
	return dir, nil
}

// lockDriverSourceDir acquires an exclusive, cross-process lock scoped to
// dir (a sibling "<dir>.lock" file) and returns a function that releases
// it. os.OpenFile with O_EXCL fails atomically if the lock file already
// exists, on every platform this CLI targets, giving simple mutual
// exclusion without a platform-specific flock dependency. A process killed
// while holding the lock would otherwise wedge every future installer
// forever, so acquisition gives up after lockTimeout and returns an error
// instead of hanging indefinitely — the stale lock file then needs manual
// removal, an accepted trade-off for a rare crash-during-install scenario.
const lockTimeout = 30 * time.Second

func lockDriverSourceDir(dir string) (unlock func(), err error) {
	lockPath := dir + ".lock"
	deadline := time.Now().Add(lockTimeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		// A concurrent holder of lockPath is reported as os.ErrExist on
		// POSIX, but on Windows CreateFile(CREATE_NEW) racing against
		// another open handle to the same path can instead surface as
		// ERROR_ACCESS_DENIED (os.ErrPermission) — both mean "contended,
		// keep retrying", not a real permission problem.
		if !os.IsExist(err) && !os.IsPermission(err) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out after %s waiting for lock %s", lockTimeout, lockPath)
		}
		time.Sleep(20 * time.Millisecond)
	}
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

const driverWorkspace = "go 1.27\n\nuse ./cli\n"
