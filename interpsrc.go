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
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// The embed includes all Go source packages needed to build the interpreter,
// plus go.mod and go.sum. parser/testdata (270 MB of test fixtures),
// corpus/, and samples/ are intentionally excluded to keep binary size small.

//go:embed go.mod go.sum interpsrc_stub.go
//go:embed ast bir cli common compiler-tools context decimal desugar lib model platform projects runtime semantics semtypes tools values
//go:embed parser/*.go parser/nodes.json parser/common parser/tree
var src embed.FS

// ExtractTo writes the embedded source tree into <cacheRoot>/interpreter-src/<version>/
// and returns that path. If the directory already exists (same version), the
// extraction is skipped and the cached path is returned immediately.
func ExtractTo(cacheRoot, version string) (string, error) {
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
