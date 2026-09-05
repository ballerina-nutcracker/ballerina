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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
)

// TestManifest_IconReadmeExportInclude parses testdata/manifest-fields-project's
// Ballerina.toml (icon, readme, [[package.modules]] export entries, include
// globs) and asserts every field lands on the in-memory PackageManifest.
func TestManifest_IconReadmeExportInclude(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "manifest-fields-project"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	manifest := result.Project().CurrentPackage().Manifest()

	if got, want := manifest.Icon(), "icon.png"; got != want {
		t.Errorf("Icon() = %q, want %q", got, want)
	}
	if got, want := manifest.Readme(), "README.md"; got != want {
		t.Errorf("Readme() = %q, want %q", got, want)
	}

	mods := manifest.Modules()
	if len(mods) != 2 {
		t.Fatalf("Modules() count = %d, want 2 (mods: %+v)", len(mods), mods)
	}
	byName := make(map[string]projects.ManifestModule, len(mods))
	for _, mod := range mods {
		byName[mod.Name()] = mod
	}

	extra, ok := byName["manifestfieldsproject.extra"]
	if !ok {
		t.Fatalf("Modules() missing explicitly-declared %q (mods: %+v)", "manifestfieldsproject.extra", mods)
	}
	if !extra.Export() {
		t.Error("extra module Export() = false, want true")
	}
	if got, want := extra.Description(), "Extra utilities"; got != want {
		t.Errorf("extra module Description() = %q, want %q", got, want)
	}
	if got, want := extra.Readme(), "modules/extra/README.md"; got != want {
		t.Errorf("extra module Readme() = %q, want %q", got, want)
	}

	undeclared, ok := byName["manifestfieldsproject.undeclared"]
	if !ok {
		t.Fatalf("Modules() missing auto-discovered %q (mods: %+v)", "manifestfieldsproject.undeclared", mods)
	}
	if undeclared.Export() {
		t.Error("auto-discovered undeclared module Export() = true, want false")
	}
	if got, want := undeclared.Readme(), "modules/undeclared/README.md"; got != want {
		t.Errorf("undeclared module Readme() = %q, want %q", got, want)
	}

	wantExported := map[string]bool{
		"manifestfieldsproject":       true,
		"manifestfieldsproject.extra": true,
	}
	exported := manifest.ExportedModules()
	if len(exported) != len(wantExported) {
		t.Errorf("ExportedModules() = %v, want %v", exported, wantExported)
	}
	for _, name := range exported {
		if !wantExported[name] {
			t.Errorf("ExportedModules() has unexpected %q (got: %v)", name, exported)
		}
	}

	wantInclude := []string{"docs/**", "assets/*.txt", "!assets/secret.txt"}
	include := manifest.Include()
	if len(include) != len(wantInclude) {
		t.Fatalf("Include() = %v, want %v", include, wantInclude)
	}
	for i, pattern := range wantInclude {
		if include[i] != pattern {
			t.Errorf("Include()[%d] = %q, want %q", i, include[i], pattern)
		}
	}
}

// TestPack_Include drives EmitBala end-to-end against
// testdata/manifest-fields-project and asserts that the include glob
// patterns (with negation) select exactly the expected files, bundled at
// their project-relative path in the archive.
func TestPack_Include(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "manifest-fields-project"))
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

	zr, err := zip.OpenReader(balaPath)
	if err != nil {
		t.Fatalf("open bala: %v", err)
	}
	defer func() { _ = zr.Close() }()
	got := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		got[f.Name] = true
	}

	wantIncluded := []string{"docs/guide.md", "assets/data.txt"}
	for _, name := range wantIncluded {
		if !got[name] {
			t.Errorf("bala missing included entry %q (entries: %v)", name, keys(got))
		}
	}

	if got["assets/secret.txt"] {
		t.Error("bala contains assets/secret.txt, which the !assets/secret.txt pattern should have excluded")
	}

	wantDocs := []string{
		"docs/README.md",
		"docs/icon.png",
		"docs/modules/manifestfieldsproject.extra/README.md",
		"docs/modules/manifestfieldsproject.undeclared/README.md",
	}
	for _, name := range wantDocs {
		if !got[name] {
			t.Errorf("bala missing docs entry %q (entries: %v)", name, keys(got))
		}
	}

	// Round-trip: the copied Ballerina.toml inside the bala re-parses through
	// the same manifest_builder.go code path. Icon/Readme are rewritten by
	// copyBallerinaToml to point at their docs/ location inside the bala
	// (not the original project-relative path, which no longer exists once
	// packed), so the reloaded manifest should reflect that new location.
	extractDir := t.TempDir()
	if err := unzipTo(balaPath, extractDir); err != nil {
		t.Fatalf("unzip: %v", err)
	}
	loaded, err := loadProject(extractDir)
	if err != nil {
		t.Fatalf("load extracted bala: %v", err)
	}
	loadedManifest := loaded.Project().CurrentPackage().Manifest()

	if got, want := loadedManifest.Icon(), "docs/icon.png"; got != want {
		t.Errorf("reloaded Icon() = %q, want %q", got, want)
	}
	if got, want := loadedManifest.Readme(), "docs/README.md"; got != want {
		t.Errorf("reloaded Readme() = %q, want %q", got, want)
	}

	// The "extra" module had no explicit readme line in the source
	// Ballerina.toml (auto-defaulted to modules/extra/README.md); packing
	// inserts one pointing at the docs/ location, which should round-trip.
	for _, mod := range loadedManifest.Modules() {
		if mod.Name() != "manifestfieldsproject.extra" {
			continue
		}
		if got, want := mod.Readme(), "docs/modules/manifestfieldsproject.extra/README.md"; got != want {
			t.Errorf("reloaded extra module Readme() = %q, want %q", got, want)
		}
	}

	wantExported := map[string]bool{
		"manifestfieldsproject":       true,
		"manifestfieldsproject.extra": true,
	}
	exported := loadedManifest.ExportedModules()
	if len(exported) != len(wantExported) {
		t.Errorf("reloaded ExportedModules() = %v, want %v", exported, wantExported)
	}
	for _, name := range exported {
		if !wantExported[name] {
			t.Errorf("reloaded ExportedModules() has unexpected %q (got: %v)", name, exported)
		}
	}
}

// TestPack_CustomReadmeAndIconTomlPathRewritten packs
// testdata/custom-readme-project, which declares non-default readme and
// icon names (readme="Package.md", icon="logo.png"), and asserts the copied
// Ballerina.toml inside the bala is rewritten to point at each file's actual
// bundled location (docs/Package.md, docs/logo.png) rather than their stale
// original paths — reproduces a real bug found by packing samples/fooLib,
// which uses the same readme="Package.md" pattern.
func TestPack_CustomReadmeAndIconTomlPathRewritten(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "custom-readme-project"))
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
	if got, want := pkg.Manifest().Readme(), "Package.md"; got != want {
		t.Fatalf("Readme() = %q, want %q (fixture precondition)", got, want)
	}
	if got, want := pkg.Manifest().Icon(), "logo.png"; got != want {
		t.Fatalf("Icon() = %q, want %q (fixture precondition)", got, want)
	}

	outDir := t.TempDir()
	balaPath, err := projects.NewBallerinaBackend(pkg.Compilation()).EmitBala(outDir)
	if err != nil {
		t.Fatalf("EmitBala: %v", err)
	}

	tomlContent, err := readZipEntry(balaPath, "Ballerina.toml")
	if err != nil {
		t.Fatalf("read Ballerina.toml from bala: %v", err)
	}
	for _, stale := range []string{`"Package.md"`, `"logo.png"`} {
		if strings.Contains(tomlContent, stale) {
			t.Errorf("packed Ballerina.toml still references the stale path %s:\n%s", stale, tomlContent)
		}
	}
	for _, rewritten := range []string{`"docs/Package.md"`, `"docs/logo.png"`} {
		if !strings.Contains(tomlContent, rewritten) {
			t.Errorf("packed Ballerina.toml missing rewritten path %s:\n%s", rewritten, tomlContent)
		}
	}

	// The files themselves must actually be at their rewritten locations.
	for _, doc := range []string{"Package.md", "logo.png"} {
		docContent, err := readZipEntry(balaPath, "docs/"+doc)
		if err != nil {
			t.Fatalf("read docs/%s from bala: %v", doc, err)
		}
		wantContent, err := os.ReadFile(filepath.Join(source, doc))
		if err != nil {
			t.Fatalf("read source %s: %v", doc, err)
		}
		if docContent != string(wantContent) {
			t.Errorf("docs/%s content = %q, want %q", doc, docContent, string(wantContent))
		}
	}

	// Round-trip: reload and confirm the manifest agrees with the archive.
	extractDir := t.TempDir()
	if err := unzipTo(balaPath, extractDir); err != nil {
		t.Fatalf("unzip: %v", err)
	}
	loaded, err := loadProject(extractDir)
	if err != nil {
		t.Fatalf("load extracted bala: %v", err)
	}
	loadedManifest := loaded.Project().CurrentPackage().Manifest()
	if got, want := loadedManifest.Readme(), "docs/Package.md"; got != want {
		t.Errorf("reloaded Readme() = %q, want %q", got, want)
	}
	if got, want := loadedManifest.Icon(), "docs/logo.png"; got != want {
		t.Errorf("reloaded Icon() = %q, want %q", got, want)
	}
}

// TestManifest_ReadmeAutoDiscovery parses testdata/readme-autodiscovery-project,
// whose Ballerina.toml never declares `readme = "..."`, and asserts Readme()
// still defaults to "README.md" because that file exists in the project.
func TestManifest_ReadmeAutoDiscovery(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "readme-autodiscovery-project"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	manifest := result.Project().CurrentPackage().Manifest()
	if got, want := manifest.Readme(), "README.md"; got != want {
		t.Errorf("Readme() = %q, want %q (auto-discovery default)", got, want)
	}
}

// TestPack_NoReadmeSkipsDocs packs testdata/no-readme-project, which sets
// `icon` but has no readme declared and no README.md file anywhere. It
// asserts the bala has no docs/ entries at all, locking in the ported Java
// quirk: addBalaDocs bundles nothing — not even a lone icon — when there's
// no readme.
func TestPack_NoReadmeSkipsDocs(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "no-readme-project"))
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
	if got := pkg.Manifest().Icon(); got != "icon.png" {
		t.Fatalf("Icon() = %q, want %q (fixture precondition)", got, "icon.png")
	}
	if got := pkg.Manifest().Readme(); got != "" {
		t.Fatalf("Readme() = %q, want empty (fixture precondition)", got)
	}

	outDir := t.TempDir()
	balaPath, err := projects.NewBallerinaBackend(pkg.Compilation()).EmitBala(outDir)
	if err != nil {
		t.Fatalf("EmitBala: %v", err)
	}

	zr, err := zip.OpenReader(balaPath)
	if err != nil {
		t.Fatalf("open bala: %v", err)
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "docs/") {
			t.Errorf("bala has docs/ entry %q, want none since the package has no readme", f.Name)
		}
	}
}
