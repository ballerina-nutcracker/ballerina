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
// needing to check out the ballerina repository separately.
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
	"path"
	"path/filepath"

	sourceast "github.com/ballerina-nutcracker/ballerina/ast"
	sourcebir "github.com/ballerina-nutcracker/ballerina/bir"
	sourcecli "github.com/ballerina-nutcracker/ballerina/cli"
	sourcecommon "github.com/ballerina-nutcracker/ballerina/common"
	sourcecontext "github.com/ballerina-nutcracker/ballerina/context"
	sourcedecimal "github.com/ballerina-nutcracker/ballerina/decimal"
	sourcedesugar "github.com/ballerina-nutcracker/ballerina/desugar"
	sourcelib "github.com/ballerina-nutcracker/ballerina/lib"
	sourcemodel "github.com/ballerina-nutcracker/ballerina/model"
	sourceparser "github.com/ballerina-nutcracker/ballerina/parser"
	sourceplatform "github.com/ballerina-nutcracker/ballerina/platform"
	sourceprojects "github.com/ballerina-nutcracker/ballerina/projects"
	sourceruntime "github.com/ballerina-nutcracker/ballerina/runtime"
	sourcesemantics "github.com/ballerina-nutcracker/ballerina/semantics"
	sourcesemtypes "github.com/ballerina-nutcracker/ballerina/semtypes"
	sourcetools "github.com/ballerina-nutcracker/ballerina/tools"
	sourcevalues "github.com/ballerina-nutcracker/ballerina/values"
)

// Each workspace module embeds its own source because go:embed cannot cross a
// Go module boundary. parser/testdata, corpus, test_util, and compiler-tools are
// intentionally excluded to keep the released binary small.

//go:embed go.mod go.sum interpsrc_stub.go
var rootSource embed.FS

type sourceTree struct {
	dir string
	fs  fs.FS
}

var interpreterSources = []sourceTree{
	{fs: rootSource},
	{dir: "ast", fs: sourceast.InterpreterSource()},
	{dir: "bir", fs: sourcebir.InterpreterSource()},
	{dir: "cli", fs: sourcecli.InterpreterSource()},
	{dir: "common", fs: sourcecommon.InterpreterSource()},
	{dir: "context", fs: sourcecontext.InterpreterSource()},
	{dir: "decimal", fs: sourcedecimal.InterpreterSource()},
	{dir: "desugar", fs: sourcedesugar.InterpreterSource()},
	{dir: "lib", fs: sourcelib.InterpreterSource()},
	{dir: "model", fs: sourcemodel.InterpreterSource()},
	{dir: "parser", fs: sourceparser.InterpreterSource()},
	{dir: "platform", fs: sourceplatform.InterpreterSource()},
	{dir: "projects", fs: sourceprojects.InterpreterSource()},
	{dir: "runtime", fs: sourceruntime.InterpreterSource()},
	{dir: "semantics", fs: sourcesemantics.InterpreterSource()},
	{dir: "semtypes", fs: sourcesemtypes.InterpreterSource()},
	{dir: "tools", fs: sourcetools.InterpreterSource()},
	{dir: "values", fs: sourcevalues.InterpreterSource()},
}

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
	if extractedSourceComplete(dir) {
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
	if existing, err := os.ReadFile(hashFile); err == nil && string(existing) == hash && extractedSourceComplete(dir) {
		return dir, nil
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
	for _, source := range interpreterSources {
		err := walkSource(source, func(p string, data []byte) error {
			h.Write([]byte(p))
			h.Write([]byte{0})
			h.Write(data)
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	h.Write([]byte(nativeWorkspace))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func extractAll(dst string) error {
	for _, source := range interpreterSources {
		err := walkSource(source, func(p string, data []byte) error {
			target := filepath.Join(dst, filepath.FromSlash(p))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		})
		if err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dst, "go.work"), []byte(nativeWorkspace), 0o644)
}

func walkSource(source sourceTree, visit func(string, []byte) error) error {
	return fs.WalkDir(source.fs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(source.fs, p)
		if err != nil {
			return err
		}
		return visit(path.Join(source.dir, p), data)
	})
}

func extractedSourceComplete(dir string) bool {
	for _, name := range []string{"go.mod", "go.work", filepath.Join("cli", "go.mod")} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

const nativeWorkspace = `go 1.26

use (
	.
	./ast
	./bir
	./cli
	./common
	./context
	./decimal
	./desugar
	./lib
	./model
	./parser
	./platform
	./projects
	./runtime
	./semantics
	./semtypes
	./tools
	./values
)

replace (
	ballerina v0.6.0 => .
	ballerina/ast v0.6.0 => ./ast
	ballerina/bir v0.6.0 => ./bir
	ballerina/cli v0.6.0 => ./cli
	ballerina/common v0.6.0 => ./common
	ballerina/context v0.6.0 => ./context
	ballerina/decimal v0.6.0 => ./decimal
	ballerina/desugar v0.6.0 => ./desugar
	ballerina/lib v0.6.0 => ./lib
	ballerina/model v0.6.0 => ./model
	ballerina/parser v0.6.0 => ./parser
	ballerina/platform v0.6.0 => ./platform
	ballerina/projects v0.6.0 => ./projects
	ballerina/runtime v0.6.0 => ./runtime
	ballerina/semantics v0.6.0 => ./semantics
	ballerina/semtypes v0.6.0 => ./semtypes
	ballerina/tools v0.6.0 => ./tools
	ballerina/values v0.6.0 => ./values
)
`
