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

// ExtractDriverSource extracts the embedded CLI driver module into the build cache.
func ExtractDriverSource(cacheRoot, version string) (string, error) {
	source := DriverSource()
	if source == nil {
		return "", errors.New("CLI driver source is not embedded in native interpreter builds")
	}
	if version == "dev" {
		return extractDevDriverSource(source)
	}

	dir := filepath.Join(cacheRoot, "interpreter-src", version)
	if extractedDriverSourceComplete(dir) {
		return dir, nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("clearing incomplete CLI driver source cache: %w", err)
	}
	if err := extractDriverSource(dir, source); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("extracting CLI driver source: %w", err)
	}
	return dir, nil
}

func extractDevDriverSource(source fs.FS) (string, error) {
	hash, err := driverSourceHash(source)
	if err != nil {
		return "", fmt.Errorf("hashing embedded CLI driver source: %w", err)
	}

	dir := filepath.Join(os.TempDir(), devDriverDirName)
	hashFile := dir + ".hash"
	if existing, err := os.ReadFile(hashFile); err == nil && string(existing) == hash && extractedDriverSourceComplete(dir) {
		return dir, nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("clearing stale CLI driver source cache: %w", err)
	}
	if err := extractDriverSource(dir, source); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("extracting CLI driver source: %w", err)
	}
	if err := os.WriteFile(hashFile, []byte(hash), 0o644); err != nil {
		return "", fmt.Errorf("writing CLI driver source cache hash: %w", err)
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
