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

package nativeexec

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
)

// FingerprintPayloads returns a hex-encoded SHA-256 digest that uniquely
// identifies a set of native payloads. Any extra seed bytes (e.g. the
// interpreter's go.mod + go.sum) are mixed in first so that changes outside
// the payloads themselves also trigger a rebuild.
func FingerprintPayloads(payloads []NativePayload, seeds ...[]byte) (string, error) {
	h := sha256.New()

	for _, seed := range seeds {
		sum := sha256.Sum256(seed)
		fmt.Fprintf(h, "seed:%x\n", sum)
	}

	sorted := slices.Clone(payloads)
	slices.SortFunc(sorted, func(a, b NativePayload) int {
		return strings.Compare(a.GoModuleName(), b.GoModuleName())
	})

	for _, payload := range sorted {
		fmt.Fprintf(h, "module:%s\n", payload.GoModuleName())
		err := fs.WalkDir(payload.FS(), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
				return err
			}
			data, err := fs.ReadFile(payload.FS(), p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			fmt.Fprintf(h, "%s:%x\n", p, sum)
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("fingerprinting %s: %w", payload.GoModuleName(), err)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// FingerprintPath returns the path where the fingerprint for binaryPath is stored.
func FingerprintPath(binaryPath string) string {
	return binaryPath + ".fingerprint"
}

// WriteFingerprint atomically writes fingerprint alongside binaryPath.
// It writes to a temp file first and renames to prevent a concurrent reader
// from observing a partial write.
func WriteFingerprint(binaryPath, fingerprint string) error {
	path := FingerprintPath(binaryPath)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(fingerprint), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
