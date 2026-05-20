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

package packageresolution

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	_ "ballerina-lang-go/lib/rt" // register stdlib runtime functions (io.println, etc.)
	"ballerina-lang-go/lib/stdlibs"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/runtime"
)

// bundledEmbedFS mirrors how lib/stdlibs/embed.go bakes the production stdlib
// tree: a compile-time embedded fs.FS rooted at <org>/<name>/<version>/<plat>/.
// Holding the fixture under this package's testdata exercises the same
// FileSystemRepository path against an embedded FS without seeding the real
// lib/stdlibs/ballerina/ tree with placeholder content.
//
//go:embed testdata/bundled-embed/ballerina
var bundledEmbedFS embed.FS

// bundledEmbedRepo returns a FileSystemRepository over the test embed.FS rooted
// at the ballerina org dir, matching how production wires the real stdlib FS.
func bundledEmbedRepo(t *testing.T) *projects.FileSystemRepository {
	t.Helper()
	sub, err := fs.Sub(bundledEmbedFS, "testdata/bundled-embed")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}
	return projects.NewFileSystemRepository(sub, ".")
}

// TestBundledRepository_LoadsEmbeddedPackage drives the full path that a real
// bundled stdlib will take at runtime: a compile-time embed.FS handed to
// FileSystemRepository, which walks <org>/<name>/<version>/<platform>/ and
// loads a TOML-format bala (Bala.toml + Ballerina.toml + Dependencies.toml)
// into a usable *Package.
func TestBundledRepository_LoadsEmbeddedPackage(t *testing.T) {
	repo := bundledEmbedRepo(t)
	ctx := context.Background()
	opts := projects.ResolutionOptions{}

	versions, err := repo.GetPackageVersions(ctx, "ballerina", "dummypkg", opts)
	if err != nil {
		t.Fatalf("GetPackageVersions: %v", err)
	}
	if len(versions) != 1 || versions[0].String() != "0.1.0" {
		t.Fatalf("versions = %v, want [0.1.0]", versions)
	}

	pkg, err := repo.GetPackage(ctx, "ballerina", "dummypkg", "0.1.0", opts)
	if err != nil {
		t.Fatalf("GetPackage: %v", err)
	}
	if pkg == nil {
		t.Fatal("GetPackage returned nil")
	}
	if got := pkg.Descriptor().Org().Value(); got != "ballerina" {
		t.Errorf("org = %q, want ballerina", got)
	}
	if got := pkg.Descriptor().Name().Value(); got != "dummypkg" {
		t.Errorf("name = %q, want dummypkg", got)
	}
	if got := pkg.Descriptor().Version().String(); got != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", got)
	}

	// Misses must report cleanly so the real resolver can fall through to
	// the next repository in the chain.
	missing, err := repo.GetPackage(ctx, "ballerina", "unknown", "0.1.0", opts)
	if err != nil {
		t.Fatalf("GetPackage(unknown): %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for unknown package, got %v", missing.Descriptor())
	}
}

// TestBundledRepository_ResolverChainServesEmbedded wires the embed-backed
// repo through ProjectEnvironmentBuilder, exercising the same PackageResolver
// path that production uses. ResolveByName should return the embedded package
// with no central cache configured.
func TestBundledRepository_ResolverChainServesEmbedded(t *testing.T) {
	env := projects.NewProjectEnvironmentBuilder(bundledEmbedFS).
		WithRepositories([]projects.Repository{bundledEmbedRepo(t)}).
		Build()

	pkgs := env.PackageResolver().ResolveByName(
		context.Background(), "ballerina", "dummypkg", env.ResolutionOptions(),
	)
	if len(pkgs) != 1 {
		t.Fatalf("ResolveByName returned %d packages, want 1", len(pkgs))
	}
	if got := pkgs[0].Descriptor().Name().Value(); got != "dummypkg" {
		t.Errorf("resolved name = %q, want dummypkg", got)
	}
	if got := pkgs[0].Descriptor().Version().String(); got != "0.1.0" {
		t.Errorf("resolved version = %q, want 0.1.0", got)
	}
}

// TestBundledRepository_ConsumerProjectRuns is the end-to-end check: a user
// project imports `ballerina/dummypkg` and calls `dummypkg:dummy()`. The
// resolver chain is restricted to the test bundled repo, so a successful run
// proves the full pipeline — embed.FS resolution, bala loader, compilation,
// BIR generation, and interpretation — works for a bundled stdlib.
func TestBundledRepository_ConsumerProjectRuns(t *testing.T) {
	projectFs := os.DirFS("testdata/dummypkg-consumer")

	result, err := projects.Load(projectFs, ".", projects.ProjectLoadConfig{
		// ballerina/io is a real bundled stdlib now (resolved through the
		// repository chain, not a compiler intrinsic), so it must be present
		// alongside the test-only dummypkg repo for the import to resolve.
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

	pkg := result.Project().CurrentPackage()
	if pkg == nil {
		t.Fatal("loaded project has no current package")
	}

	compilation := pkg.Compilation()
	if diag := compilation.DiagnosticResult(); diag.HasErrors() {
		t.Fatalf("compilation diagnostics: %v", diag.Diagnostics())
	}

	birPkgs := projects.NewBallerinaBackend(compilation).BIRPackages()
	if len(birPkgs) == 0 {
		t.Fatal("backend produced no BIR packages")
	}

	var stdout bytes.Buffer
	testPal := pal.Platform{
		IO: pal.IO{
			Stdout: func(p []byte) (int, error) { return stdout.Write(p) },
			Stderr: func(p []byte) (int, error) { return len(p), nil },
		},
		Signals: pal.SignalSource{Signals: make(chan pal.Signal)},
	}

	tyEnv := result.Project().Environment().TypeEnv()
	rt := runtime.NewRuntime(testPal, tyEnv)
	for _, birPkg := range birPkgs {
		if err := rt.Init(*birPkg); err != nil {
			t.Fatalf("Init(%v): %v", birPkg.PackageID, err)
		}
	}
	rt.Listen()
	select {
	case exitCode := <-rt.ExitStatus:
		if exitCode != 0 {
			t.Fatalf("unexpected exit code: %d", exitCode)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runtime exit status")
	}

	if got := strings.TrimSpace(stdout.String()); got != "42" {
		t.Errorf("stdout = %q, want %q", got, "42")
	}
}
