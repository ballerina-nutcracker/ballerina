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

package corpus

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "ballerina/lib/rt" // register stdlib runtime functions (io.println, etc.)
	// foo has native code; bar is pure Ballerina. foo's native must be registered
	// so the interpreter can dispatch foo:add() at runtime.
	"ballerina/bir"
	_ "ballerina/corpus/package-resolution/testdata/bundled-embed/ballerina/foo/0.1.0/go1.26/native"
	"ballerina/lib/stdlibs"
	"ballerina/projects"
	"ballerina/runtime"
	"ballerina/semtypes"
	"ballerina/test_util/testharness"
)

const packageResolutionTestDataDir = "package-resolution/testdata"

// bundledEmbedFS mirrors how lib/stdlibs/embed.go bakes the production stdlib
// tree: a compile-time embedded fs.FS rooted at <org>/<name>/<version>/<plat>/.
// Holding the fixture under corpus/package-resolution/testdata exercises the same
// FileSystemRepository path against an embedded FS without seeding the real
// lib/stdlibs/ballerina/ tree with placeholder content.
//
//go:embed package-resolution/testdata/bundled-embed/ballerina
var bundledEmbedFS embed.FS

func bundledEmbedRepo(t *testing.T) *projects.FileSystemRepository {
	t.Helper()
	sub, err := fs.Sub(bundledEmbedFS, packageResolutionTestDataDir+"/bundled-embed")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}
	return projects.NewFileSystemRepository(sub, ".")
}

func interpretBIRPackagesStdout(t *testing.T, typeEnv semtypes.Env, birPkgs []*bir.BIRPackage) string {
	t.Helper()
	pal := testharness.NewTestPal()
	defer pal.Close()
	rt := runtime.NewRuntime(pal.Platform(), typeEnv)
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			t.Fatalf("Init(%v): %v", birPkg.PackageID, err)
		}
	}
	return strings.TrimSpace(pal.Stdout())
}

// TestBundledRepository_LoadsEmbeddedPackage verifies that both a pure-Ballerina
// package (bar) and a native package (foo) can be loaded from the embedded FS.
func TestBundledRepository_LoadsEmbeddedPackage(t *testing.T) {
	repo := bundledEmbedRepo(t)
	ctx := context.Background()
	opts := projects.ResolutionOptions{}

	for _, tc := range []struct {
		name    string
		version string
	}{
		{"foo", "0.1.0"}, // has native code
		{"bar", "0.1.0"}, // pure Ballerina
	} {
		t.Run(tc.name, func(t *testing.T) {
			versions, err := repo.GetPackageVersions(ctx, "ballerina", tc.name, opts)
			if err != nil {
				t.Fatalf("GetPackageVersions: %v", err)
			}
			if len(versions) != 1 || versions[0].String() != tc.version {
				t.Fatalf("versions = %v, want [%s]", versions, tc.version)
			}

			pkg, err := repo.GetPackage(ctx, "ballerina", tc.name, tc.version, opts)
			if err != nil {
				t.Fatalf("GetPackage: %v", err)
			}
			if pkg == nil {
				t.Fatal("GetPackage returned nil")
			}
			if got := pkg.Descriptor().Name().Value(); got != tc.name {
				t.Errorf("name = %q, want %q", got, tc.name)
			}
		})
	}

	missing, err := repo.GetPackage(ctx, "ballerina", "unknown", "0.1.0", opts)
	if err != nil {
		t.Fatalf("GetPackage(unknown): %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for unknown package, got %v", missing.Descriptor())
	}
}

// TestBundledRepository_ResolverChainServesEmbedded verifies both packages
// resolve through the ProjectEnvironmentBuilder resolver chain.
func TestBundledRepository_ResolverChainServesEmbedded(t *testing.T) {
	env := projects.NewProjectEnvironmentBuilder(bundledEmbedFS).
		WithRepositories([]projects.Repository{bundledEmbedRepo(t)}).
		Build()

	for _, name := range []string{"foo", "bar"} {
		pkgs := env.PackageResolver().ResolveByName(
			context.Background(), "ballerina", name, env.ResolutionOptions(),
		)
		if len(pkgs) != 1 {
			t.Fatalf("ResolveByName(%q) returned %d packages, want 1", name, len(pkgs))
		}
		if got := pkgs[0].Descriptor().Name().Value(); got != name {
			t.Errorf("resolved name = %q, want %q", got, name)
		}
	}
}

// TestBundledRepository_ConsumerProjectRuns is the end-to-end check: a user
// project imports ballerina/foo (native) and ballerina/bar (pure Ballerina).
// foo:add(3, 4) exercises native dispatch; bar:value() exercises pure Ballerina.
// Expected stdout: "7\n1\n"
func TestBundledRepository_ConsumerProjectRuns(t *testing.T) {
	projectFs := os.DirFS(filepath.Join(packageResolutionTestDataDir, "userProject"))

	result, err := projects.Load(projectFs, ".", projects.ProjectLoadConfig{
		Repositories: []projects.Repository{
			bundledEmbedRepo(t),
			projects.NewFileSystemRepository(stdlibs.FS, "."),
		},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if diag := result.Diagnostics(); diag.HasErrors() {
		t.Fatalf("load diagnostics: %v", diag.Diagnostics())
	}

	compilation := result.Project().CurrentPackage().Compilation()
	if diag := compilation.DiagnosticResult(); diag.HasErrors() {
		t.Fatalf("compilation diagnostics: %v", diag.Diagnostics())
	}

	birPkgs := projects.NewBallerinaBackend(compilation).BIRPackages()
	if len(birPkgs) == 0 {
		t.Fatal("backend produced no BIR packages")
	}

	got := interpretBIRPackagesStdout(t, result.Project().Environment().TypeEnv(), birPkgs)
	want := "7\n1"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}
