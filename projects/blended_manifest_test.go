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

// Package projects internal tests for blendedManifest.
// TestDepResFix_TransitiveLocalPin_HonoredViaBFS will land here after the
// local-repo branch rebase, where custom-repo dispatch in the resolver is
// fully wired.

package projects

import (
	"os"
	"strings"
	"testing"

	"ballerina-lang-go/tools/diagnostics"
)

// TestBlendedManifest_Dependency exercises blendedManifest.dependency in
// isolation.
func TestBlendedManifest_Dependency(t *testing.T) {
	t.Parallel()
	// Build a root manifest with a known set of [[dependency]] entries.
	v100, err := NewPackageVersionFromString("1.0.0")
	if err != nil {
		t.Fatalf("NewPackageVersionFromString: %v", err)
	}
	v200, err := NewPackageVersionFromString("2.0.0")
	if err != nil {
		t.Fatalf("NewPackageVersionFromString: %v", err)
	}

	rootDesc := NewPackageDescriptor(
		NewPackageOrg("consumerorg"),
		NewPackageName("consumerpkg"),
		v100,
	)

	// Deps: localpkg with user-specified repository "local", plainpkg with no repository.
	deps := []Dependency{
		NewDependencyWithRepository(
			NewPackageOrg("myorg"),
			NewPackageName("localpkg"),
			v100,
			"local",
		),
		NewDependency(
			NewPackageOrg("myorg"),
			NewPackageName("plainpkg"),
			v200,
		),
	}

	rootManifest := NewPackageManifestFromParams(PackageManifestParams{
		PackageDesc:  rootDesc,
		Dependencies: deps,
	})

	// Empty manifest for the "no deps" subtests.
	emptyDesc := NewPackageDescriptor(
		NewPackageOrg("emptyorg"),
		NewPackageName("emptypkg"),
		v100,
	)
	emptyManifest := NewPackageManifestFromParams(PackageManifestParams{
		PackageDesc: emptyDesc,
	})

	tests := []struct {
		name     string
		manifest PackageManifest
		org      string
		pkgName  string
		wantOk   bool
		wantRepo string
	}{
		{
			name:     "present with repository=local",
			manifest: rootManifest,
			org:      "myorg",
			pkgName:  "localpkg",
			wantOk:   true,
			wantRepo: "local",
		},
		{
			name:     "present with no repository field",
			manifest: rootManifest,
			org:      "myorg",
			pkgName:  "plainpkg",
			wantOk:   true,
			wantRepo: "",
		},
		{
			name:     "absent from root manifest",
			manifest: rootManifest,
			org:      "myorg",
			pkgName:  "unknownpkg",
			wantOk:   false,
			wantRepo: "",
		},
		{
			name:     "org match but name mismatch",
			manifest: rootManifest,
			org:      "myorg",
			pkgName:  "localpkgX",
			wantOk:   false,
			wantRepo: "",
		},
		{
			name:     "empty root manifest",
			manifest: emptyManifest,
			org:      "myorg",
			pkgName:  "anypkg",
			wantOk:   false,
			wantRepo: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bm := newBlendedManifest(withPackageManifest(tt.manifest))

			got, ok := bm.dependency(tt.org, tt.pkgName)
			if ok != tt.wantOk {
				t.Errorf("dependency(%q, %q) ok = %v, want %v", tt.org, tt.pkgName, ok, tt.wantOk)
			}
			if ok && got.Repository() != tt.wantRepo {
				t.Errorf("dependency(%q, %q) Repository() = %q, want %q",
					tt.org, tt.pkgName, got.Repository(), tt.wantRepo)
			}
		})
	}
}

// TestBlendedManifest_NilReceiver verifies the nil-guard on dependency():
// a nil *blendedManifest must return (zero, false) without panicking.
func TestBlendedManifest_NilReceiver(t *testing.T) {
	t.Parallel()
	var bm *blendedManifest
	got, ok := bm.dependency("anyorg", "anypkg")
	if ok {
		t.Errorf("nil receiver: dependency() ok = true, want false")
	}
	if got != (blendedDependency{}) {
		t.Errorf("nil receiver: dependency() returned non-zero value: %+v", got)
	}
}

// TestBuildBlendedManifest_RepoHit verifies that a dep with repository="local"
// is kept in the index when the PackageResolver can resolve it. Also exercises
// the blendedDependency Org/Name/Version accessors.
func TestBuildBlendedManifest_RepoHit(t *testing.T) {
	t.Parallel()
	v100, err := NewPackageVersionFromString("1.0.0")
	if err != nil {
		t.Fatalf("NewPackageVersionFromString: %v", err)
	}

	rootDesc := NewPackageDescriptor(NewPackageOrg("myorg"), NewPackageName("mypkg"), v100)
	manifest := NewPackageManifestFromParams(PackageManifestParams{
		PackageDesc: rootDesc,
		Dependencies: []Dependency{
			NewDependencyWithRepository(NewPackageOrg("mockorg"), NewPackageName("mockpkg"), v100, "local"),
		},
	})

	localRepo := NewFileSystemRepository(os.DirFS("testdata/repo/bala"), ".")
	resolver := newPackageResolverWithCustom(newPackageCache(), nil, map[string]Repository{"local": localRepo})

	var warned []diagnostics.Diagnostic
	bm := newBlendedManifest(
		withPackageManifest(manifest),
		withPackageResolver(resolver, ResolutionOptions{}),
		withDiagnosticReporter(func(d diagnostics.Diagnostic) { warned = append(warned, d) }),
	)

	got, ok := bm.dependency("mockorg", "mockpkg")
	if !ok {
		t.Fatal("dep should be in index after successful repo resolution")
	}
	if len(warned) != 0 {
		t.Errorf("expected no warnings, got %d", len(warned))
	}
	if got.Org().Value() != "mockorg" {
		t.Errorf("Org() = %q, want %q", got.Org().Value(), "mockorg")
	}
	if got.Name().Value() != "mockpkg" {
		t.Errorf("Name() = %q, want %q", got.Name().Value(), "mockpkg")
	}
	if got.Version().String() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", got.Version().String(), "1.0.0")
	}
	if got.Repository() != "local" {
		t.Errorf("Repository() = %q, want %q", got.Repository(), "local")
	}
}

// TestBuildBlendedManifest_RepoMiss verifies that a dep with repository="local"
// is dropped and a warning is emitted when the PackageResolver cannot resolve it.
// This exercises buildBlendedManifest's repo-miss branch and emitLocalRepoMissWarning.
func TestBuildBlendedManifest_RepoMiss(t *testing.T) {
	t.Parallel()
	v100, err := NewPackageVersionFromString("1.0.0")
	if err != nil {
		t.Fatalf("NewPackageVersionFromString: %v", err)
	}

	rootDesc := NewPackageDescriptor(NewPackageOrg("myorg"), NewPackageName("mypkg"), v100)
	manifest := NewPackageManifestFromParams(PackageManifestParams{
		PackageDesc: rootDesc,
		Dependencies: []Dependency{
			NewDependencyWithRepository(NewPackageOrg("mockorg"), NewPackageName("nosuchpkg"), v100, "local"),
		},
	})

	// "local" repo exists but does not contain nosuchpkg.
	localRepo := NewFileSystemRepository(os.DirFS("testdata/repo/bala"), ".")
	resolver := newPackageResolverWithCustom(newPackageCache(), nil, map[string]Repository{"local": localRepo})

	var warned []diagnostics.Diagnostic
	bm := newBlendedManifest(
		withPackageManifest(manifest),
		withPackageResolver(resolver, ResolutionOptions{}),
		withDiagnosticReporter(func(d diagnostics.Diagnostic) { warned = append(warned, d) }),
	)

	_, ok := bm.dependency("mockorg", "nosuchpkg")
	if ok {
		t.Error("dep should be dropped after resolution miss")
	}
	if len(warned) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warned))
	}
	msg := warned[0].Message()
	if !strings.Contains(msg, "nosuchpkg") {
		t.Errorf("warning should mention package name, got: %q", msg)
	}
	if !strings.Contains(msg, "'local'") {
		t.Errorf("warning should mention repo name, got: %q", msg)
	}
}
