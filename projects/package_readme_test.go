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
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
)

// TestPackage_ReadmeMd_CustomName loads testdata/custom-readme-project
// (readme="Package.md") and asserts Package.ReadmeMd() resolves the
// manifest-declared custom path, not just the default "README.md".
func TestPackage_ReadmeMd_CustomName(t *testing.T) {
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
	readme := pkg.ReadmeMd()
	if readme == nil {
		t.Fatal("ReadmeMd() = nil, want a document for readme=\"Package.md\"")
	}
	if got, want := readme.Name(), "Package.md"; got != want {
		t.Errorf("ReadmeMd().Name() = %q, want %q", got, want)
	}
	if readme.PackageInstance() != pkg {
		t.Error("ReadmeMd().PackageInstance() != the package it came from")
	}

	wantContent, err := os.ReadFile(filepath.Join(source, "Package.md"))
	if err != nil {
		t.Fatalf("read source Package.md: %v", err)
	}
	if got, want := readme.Content(), string(wantContent); got != want {
		t.Errorf("ReadmeMd().Content() = %q, want %q", got, want)
	}
}

// TestPackage_ReadmeMd_AutoDiscovered loads
// testdata/readme-autodiscovery-project, whose Ballerina.toml never
// declares `readme = "..."`, and asserts Package.ReadmeMd() still resolves
// the auto-discovered README.md.
func TestPackage_ReadmeMd_AutoDiscovered(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "readme-autodiscovery-project"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	pkg := result.Project().CurrentPackage()
	readme := pkg.ReadmeMd()
	if readme == nil {
		t.Fatal("ReadmeMd() = nil, want the auto-discovered README.md")
	}
	if got, want := readme.Name(), "README.md"; got != want {
		t.Errorf("ReadmeMd().Name() = %q, want %q", got, want)
	}
}

// TestPackage_ReadmeMd_Absent loads testdata/no-readme-project (no readme
// declared, no README.md present) and asserts ReadmeMd() is nil.
func TestPackage_ReadmeMd_Absent(t *testing.T) {
	t.Parallel()
	source, err := filepath.Abs(filepath.Join("testdata", "no-readme-project"))
	if err != nil {
		t.Fatalf("abs source: %v", err)
	}
	result, err := loadProject(source)
	if err != nil {
		t.Fatalf("load source project: %v", err)
	}

	if readme := result.Project().CurrentPackage().ReadmeMd(); readme != nil {
		t.Errorf("ReadmeMd() = %+v, want nil", readme)
	}
}

// TestPackage_ReadmeMd_BalaRoundTrip packs testdata/custom-readme-project,
// reloads the resulting bala, and asserts the reloaded package's
// ReadmeMd() still resolves — exercising the docs/ bundling and the
// Ballerina.toml path rewrite together.
func TestPackage_ReadmeMd_BalaRoundTrip(t *testing.T) {
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
	balaPath, err := projects.NewBallerinaBackend(pkg.Compilation()).EmitBala(t.TempDir())
	if err != nil {
		t.Fatalf("EmitBala: %v", err)
	}

	extractDir := t.TempDir()
	if err := unzipTo(balaPath, extractDir); err != nil {
		t.Fatalf("unzip: %v", err)
	}
	loaded, err := loadProject(extractDir)
	if err != nil {
		t.Fatalf("load extracted bala: %v", err)
	}

	readme := loaded.Project().CurrentPackage().ReadmeMd()
	if readme == nil {
		t.Fatal("reloaded ReadmeMd() = nil, want a document")
	}
	if got, want := readme.Name(), "Package.md"; got != want {
		t.Errorf("reloaded ReadmeMd().Name() = %q, want %q", got, want)
	}

	wantContent, err := os.ReadFile(filepath.Join(source, "Package.md"))
	if err != nil {
		t.Fatalf("read source Package.md: %v", err)
	}
	if got, want := readme.Content(), string(wantContent); got != want {
		t.Errorf("reloaded ReadmeMd().Content() = %q, want %q", got, want)
	}
}
