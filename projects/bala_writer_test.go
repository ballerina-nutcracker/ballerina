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

package projects_test

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ballerina/projects"
)

// TestPack_ReadBala loads and asserts that the .bala archive
// has the expected entries and that loading it produces a Bala
// project with the correct descriptor and module set.
func TestPack_ReadBala(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "myproject"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	pkg := result.Project().CurrentPackage()
	if pkg == nil {
		t.Fatal("source project has no CurrentPackage")
	}

	outDir := t.TempDir()
	balaPath, err := projects.NewBallerinaBackend(pkg.Compilation()).EmitBala(outDir)
	if err != nil {
		t.Fatalf("EmitBala: %v", err)
	}

	// Archive layout mirrors the source-project shape: default-module .bal
	// at the root, only sub-modules nest under modules/. Bala.toml is the
	// one bala-only addition.
	wantEntries := map[string]bool{
		"Bala.toml":         true,
		"Ballerina.toml":    true,
		"Dependencies.toml": true,
		"main.bal":          true,
		"util.bal":          true,
	}
	zr, err := zip.OpenReader(balaPath)
	if err != nil {
		t.Fatalf("open bala: %v", err)
	}
	got := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		got[f.Name] = true
	}
	_ = zr.Close()
	for name := range wantEntries {
		if !got[name] {
			t.Errorf("bala missing entry %q (entries: %v)", name, keys(got))
		}
	}

	// Extract into a temp dir and re-load through the bala loader. This
	// exercises createBalaProjectConfig end-to-end.
	extractDir := t.TempDir()
	if err := unzipTo(balaPath, extractDir); err != nil {
		t.Fatalf("unzip: %v", err)
	}

	loaded, err := loadProject(extractDir)
	if err != nil {
		t.Fatalf("load extracted bala: %v", err)
	}
	if got := loaded.Project().Kind(); got != projects.ProjectKindBala {
		t.Errorf("loaded project kind = %v, want bala", got)
	}
	loadedPkg := loaded.Project().CurrentPackage()
	if loadedPkg == nil {
		t.Fatal("extracted bala has no CurrentPackage")
	}
	if got := loadedPkg.Descriptor().Name().String(); got != "myproject" {
		t.Errorf("loaded package name = %q, want myproject", got)
	}
	if got := loadedPkg.Descriptor().Org().String(); got != "testorg" {
		t.Errorf("loaded package org = %q, want testorg", got)
	}
	if got := loadedPkg.Descriptor().Version().String(); got != "0.1.0" {
		t.Errorf("loaded package version = %q, want 0.1.0", got)
	}
}

// TestPack_MultiModuleLayout guards the bala v4 layout split: default-module
// .bal files at the archive root, sub-modules under modules/<sub>/ (never
// under modules/<pkgName>/ and never with dotted full names as directories).
func TestPack_MultiModuleLayout(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "multi-module-project"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	pkg := result.Project().CurrentPackage()
	if pkg == nil {
		t.Fatal("source project has no CurrentPackage")
	}

	outDir := t.TempDir()
	balaPath, err := projects.NewBallerinaBackend(pkg.Compilation()).EmitBala(outDir)
	if err != nil {
		t.Fatalf("EmitBala: %v", err)
	}

	wantEntries := map[string]bool{
		"Bala.toml":                true,
		"Ballerina.toml":           true,
		"Dependencies.toml":        true,
		"main.bal":                 true,
		"utils.bal":                true,
		"modules/services/svc.bal": true,
		"modules/storage/db.bal":   true,
	}
	zr, err := zip.OpenReader(balaPath)
	if err != nil {
		t.Fatalf("open bala: %v", err)
	}
	got := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		got[f.Name] = true
	}
	_ = zr.Close()
	for name := range wantEntries {
		if !got[name] {
			t.Errorf("bala missing entry %q (entries: %v)", name, keys(got))
		}
	}
	for name := range got {
		if !wantEntries[name] {
			t.Errorf("bala has unexpected entry %q (entries: %v)", name, keys(got))
		}
	}

	// Negative checks: lock in the structural split so a regression to the
	// old modules/<pkgName>/... shape (or dotted-name dirs / leaked sub-module
	// files at root) fails loudly.
	if got["modules/multimoduleproject/main.bal"] {
		t.Errorf("default-module file leaked under modules/<pkgName>/: modules/multimoduleproject/main.bal")
	}
	if got["modules/multimoduleproject/utils.bal"] {
		t.Errorf("default-module file leaked under modules/<pkgName>/: modules/multimoduleproject/utils.bal")
	}
	if got["modules/multimoduleproject.services/svc.bal"] {
		t.Errorf("dotted full module name used as directory: modules/multimoduleproject.services/svc.bal")
	}
	if got["modules/multimoduleproject.storage/db.bal"] {
		t.Errorf("dotted full module name used as directory: modules/multimoduleproject.storage/db.bal")
	}
	if got["svc.bal"] {
		t.Errorf("sub-module file leaked to archive root: svc.bal")
	}
	if got["db.bal"] {
		t.Errorf("sub-module file leaked to archive root: db.bal")
	}

	// Extract and re-load through the bala loader to verify the new layout
	// round-trips into the expected module set.
	extractDir := t.TempDir()
	if err := unzipTo(balaPath, extractDir); err != nil {
		t.Fatalf("unzip: %v", err)
	}

	loaded, err := loadProject(extractDir)
	if err != nil {
		t.Fatalf("load extracted bala: %v", err)
	}
	if got := loaded.Project().Kind(); got != projects.ProjectKindBala {
		t.Errorf("loaded project kind = %v, want bala", got)
	}
	loadedPkg := loaded.Project().CurrentPackage()
	if loadedPkg == nil {
		t.Fatal("extracted bala has no CurrentPackage")
	}
	mods := loadedPkg.Modules()
	if len(mods) != 3 {
		t.Fatalf("loaded module count = %d, want 3", len(mods))
	}
	wantNames := map[string]bool{
		"multimoduleproject":          true,
		"multimoduleproject.services": true,
		"multimoduleproject.storage":  true,
	}
	gotNames := make(map[string]bool, len(mods))
	defaults := 0
	for _, mod := range mods {
		gotNames[mod.ModuleName().String()] = true
		if mod.IsDefaultModule() {
			defaults++
		}
	}
	for name := range wantNames {
		if !gotNames[name] {
			t.Errorf("loaded modules missing %q (got: %v)", name, keys(gotNames))
		}
	}
	for name := range gotNames {
		if !wantNames[name] {
			t.Errorf("loaded modules has unexpected %q (got: %v)", name, keys(gotNames))
		}
	}
	if defaults != 1 {
		t.Errorf("default module count = %d, want 1", defaults)
	}
}

// TestPack_DependenciesToml drives the bala writer end-to-end against a real
// fixture with a transitive dep chain (transitive_app -> middlepkg -> leafpkg).
// It loads the project with testdata/repo wired as a FileSystemRepository,
// forces compilation (so resolution populates the graph), emits a .bala, then
// byte-compares the archive's Dependencies.toml entry against a pregenerated
// expected fixture.
func TestPack_DependenciesToml(t *testing.T) {
	t.Parallel()
	testRepoPath, err := filepath.Abs(filepath.Join("testdata", "repo", "bala"))
	if err != nil {
		t.Fatalf("abs repo path: %v", err)
	}
	source, err := filepath.Abs(filepath.Join("testdata", "project-with-transitive-dep"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}

	result, err := loadProject(source, projects.ProjectLoadConfig{
		Repositories: []projects.Repository{
			projects.NewFileSystemRepository(os.DirFS(testRepoPath), "."),
		},
	})
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	pkg := result.Project().CurrentPackage()
	if pkg == nil {
		t.Fatal("source project has no CurrentPackage")
	}

	// Force compilation so the resolver runs and the dep graph is populated.
	compilation := pkg.Compilation()
	if compilation == nil {
		t.Fatal("compilation is nil")
	}

	outDir := t.TempDir()
	balaPath, err := projects.NewBallerinaBackend(compilation).EmitBala(outDir)
	if err != nil {
		t.Fatalf("EmitBala: %v", err)
	}

	got, err := readZipEntry(balaPath, "Dependencies.toml")
	if err != nil {
		t.Fatalf("read Dependencies.toml from bala: %v", err)
	}

	wantBytes, err := os.ReadFile(filepath.Join(source, "expected-Dependencies.toml"))
	if err != nil {
		t.Fatalf("read expected-Dependencies.toml: %v", err)
	}
	want := string(wantBytes)

	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Errorf("Dependencies.toml mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestPack_EmitBala_NoBallerinaToml verifies that EmitBala returns an error
// when called on a package that has no Ballerina.toml (e.g. a bala loaded
// from the local cache). Such packages expose only a Bala.toml, so packing
// them is rejected rather than producing a corrupt archive.
func TestPack_EmitBala_NoBallerinaToml(t *testing.T) {
	t.Parallel()

	balaPath, err := filepath.Abs(filepath.Join("testdata", "repo", "bala", "mockorg", "mockpkg", "1.0.0", "any"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	result, err := loadProject(balaPath)
	if err != nil {
		t.Fatalf("load bala project: %v", err)
	}

	pkg := result.Project().CurrentPackage()
	if pkg == nil {
		t.Fatal("bala project has no CurrentPackage")
	}

	_, err = projects.NewBallerinaBackend(pkg.Compilation()).EmitBala(t.TempDir())
	if err == nil {
		t.Fatal("EmitBala on a bala project should return an error")
	}
	if !strings.Contains(err.Error(), "no Ballerina.toml") {
		t.Errorf("unexpected error: %v", err)
	}
}

// readZipEntry opens the archive at path and returns the contents of the entry
// with the given name. Returns an error if the entry is missing.
func readZipEntry(archive, name string) (string, error) {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer func() { _ = rc.Close() }()
		buf, err := io.ReadAll(rc)
		if err != nil {
			return "", err
		}
		return string(buf), nil
	}
	return "", fmt.Errorf("entry %q not found in %s", name, archive)
}

func keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// unzipTo expands archive into destDir.
func unzipTo(archive, destDir string) (err error) {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := zr.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	for _, f := range zr.File {
		dst := filepath.Join(destDir, f.Name)
		cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
		cleanDst := filepath.Clean(dst)
		if !strings.HasPrefix(cleanDst, cleanDest) {
			return fmt.Errorf("archive entry escapes destination: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			_ = out.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rcErr := rc.Close()
		outErr := out.Close()
		if copyErr != nil || rcErr != nil || outErr != nil {
			return errors.Join(copyErr, rcErr, outErr)
		}
	}
	return nil
}
