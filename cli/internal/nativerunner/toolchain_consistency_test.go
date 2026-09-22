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

package nativerunner

// The Go toolchain version the tree targets is stored in five uncoupled
// places: go.work, the per-module go.mod directives, MinGoVersion here, the
// go-version pins in CI, and the go1.X platform segment of the bundled
// stdlib balas. Nothing in the build makes them agree, so a bump that misses
// one leaves a tree that still compiles and still passes every other test.
//
// These tests couple them. MinGoVersion is the anchor, because it is the one
// copy compiled into the binary: Available() gates native builds on it
// (local_executor.go), and it is stamped into the go directive of every
// generated native module and of the fallback native workspace. Let it drift
// below the tree's own go directive and Available() green-lights a toolchain
// that cannot build the interpreter -- silently recovered by a GOTOOLCHAIN=auto
// download, a hard "go.mod requires go >= X" failure under GOTOOLCHAIN=local
// or offline, far from its cause.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// The go and toolchain directives are always unindented, one per line,
	// which is what keeps these from matching module paths inside a
	// require block (those are tab-indented).
	goDirectiveRE        = regexp.MustCompile(`(?m)^go[ \t]+(\S+)[ \t]*$`)
	toolchainDirectiveRE = regexp.MustCompile(`(?m)^toolchain[ \t]+(\S+)[ \t]*$`)
	// actions/setup-go's version input, quoted or bare.
	setupGoVersionRE = regexp.MustCompile(`go-version:[ \t]*['"]?([0-9][0-9.]*)['"]?`)
	// The pinned golangci-lint, as it appears in a `go install` line.
	golangciPinRE = regexp.MustCompile(`golangci-lint/v2/cmd/golangci-lint@(v[0-9][0-9.]*)`)
)

// golangciPinSites are the three independent copies of the golangci-lint
// pin: what CI installs, what the commit-msg hook installs, and what the
// contributor guide tells a developer to install. Disagreement means a
// developer's local lint accepts what CI rejects, or the reverse.
var golangciPinSites = []string{
	filepath.Join(".github", "workflows", "golangci-lint.yml"),
	filepath.Join(".githooks", "commit-msg"),
	filepath.Join("doc", "guides", "DEVELOPING.md"),
}

// repoRoot walks up from the package directory to the repository root,
// identified by the same go.work that Available() looks for.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolving package dir: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("found no go.work above the package directory; these tests need the repository checkout")
		}
		dir = parent
	}
}

// directive returns the single capture of re in the file at path, failing the
// test when the directive is absent -- a missing directive must not read as
// an agreeing one.
func directive(t *testing.T, re *regexp.Regexp, path, what string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	m := re.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("%s declares no %s directive", path, what)
	}
	return m[1]
}

// workspaceGoVersion is the tree's language version, the value MinGoVersion
// and every module's go directive must agree with.
func workspaceGoVersion(t *testing.T) string {
	t.Helper()
	return directive(t, goDirectiveRE, filepath.Join(repoRoot(t), "go.work"), "go")
}

// workspaceToolchain is the tree's pinned toolchain (e.g. "go1.27.1"), the
// value CI must install.
func workspaceToolchain(t *testing.T) string {
	t.Helper()
	return directive(t, toolchainDirectiveRE, filepath.Join(repoRoot(t), "go.work"), "toolchain")
}

// goModFiles lists every module manifest in the tree, so a module added
// after a toolchain bump is covered without anyone remembering to list it.
func goModFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var found []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Module manifests under testdata belong to fixtures, which
			// pin versions deliberately.
			if name := d.Name(); name == ".git" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "go.mod" {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s for go.mod files: %v", root, err)
	}
	if len(found) == 0 {
		t.Fatalf("found no go.mod files under %s", root)
	}
	return found
}

// TestMinGoVersionMatchesWorkspace is the invariant the other toolchain
// tests cannot express: the minimum this package enforces at runtime is
// exactly the language version the tree is written against.
func TestMinGoVersionMatchesWorkspace(t *testing.T) {
	t.Parallel()
	want := workspaceGoVersion(t)
	if MinGoVersion != want {
		t.Errorf("MinGoVersion = %q, want %q (the go directive in go.work)\n"+
			"a lower MinGoVersion lets Available() accept a toolchain that cannot build this tree;\n"+
			"a higher one rejects toolchains that can", MinGoVersion, want)
	}
}

// TestModuleDirectivesMatchWorkspace covers a bump that updates most
// manifests but misses one, and a module added later that carries a stale
// directive copied from an older template.
func TestModuleDirectivesMatchWorkspace(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	wantGo := workspaceGoVersion(t)
	wantToolchain := workspaceToolchain(t)

	for _, path := range goModFiles(t) {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		if got := directive(t, goDirectiveRE, path, "go"); got != wantGo {
			t.Errorf("%s: go directive = %q, want %q (go.work)", rel, got, wantGo)
		}
		if got := directive(t, toolchainDirectiveRE, path, "toolchain"); got != wantToolchain {
			t.Errorf("%s: toolchain directive = %q, want %q (go.work)", rel, got, wantToolchain)
		}
	}
}

// TestCIGoVersionMatchesToolchain covers CI building the tree with a
// different toolchain than the one it pins -- which turns the pin into a
// suggestion and lets a version-sensitive vet or lint finding appear or
// vanish depending on which file you read.
func TestCIGoVersionMatchesToolchain(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	// go.work says "go1.27.1"; setup-go wants "1.27.1".
	want := strings.TrimPrefix(workspaceToolchain(t), "go")

	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatalf("globbing workflows: %v", err)
	}
	if len(workflows) == 0 {
		t.Fatal("found no workflows under .github/workflows")
	}

	pins := 0
	for _, path := range workflows {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, m := range setupGoVersionRE.FindAllStringSubmatch(string(data), -1) {
			pins++
			if m[1] != want {
				t.Errorf("%s: go-version %q, want %q (the toolchain directive in go.work)",
					filepath.Base(path), m[1], want)
			}
		}
	}
	if pins == 0 {
		t.Error("matched no go-version pins in any workflow; the pattern has gone stale")
	}
}

// TestBundledStdlibPlatformMatchesToolchain covers a bundled bala claiming a
// Go toolchain the tree no longer uses. Resolution cannot catch this:
// FileSystemRepository matches platform directories on the bare "go" prefix
// (platformGoPrefix), so a stale go1.X directory still resolves and every
// package test still passes.
//
// Scoped to lib/stdlibs, the balas actually shipped. Platform directories
// under testdata are fixtures and may pin a version deliberately.
func TestBundledStdlibPlatformMatchesToolchain(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	stdlibs := filepath.Join(root, "lib", "stdlibs")
	want := "go" + workspaceGoVersion(t)

	checked := 0
	err := filepath.WalkDir(stdlibs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || !strings.HasPrefix(d.Name(), "go1.") {
			return nil
		}
		checked++
		if d.Name() != want {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}
			t.Errorf("%s: platform directory %q, want %q (derived from go.work)", rel, d.Name(), want)
		}
		return filepath.SkipDir
	})
	if err != nil {
		t.Fatalf("walking %s: %v", stdlibs, err)
	}
	if checked == 0 {
		t.Errorf("found no go1.* platform directories under lib/stdlibs; the layout has changed")
	}
}

// TestGolangciLintPinsAgree covers a linter bump applied to CI but not to
// the hook or the guide. Nothing builds these three against each other, so
// the only symptom is a contributor whose pre-commit lint disagrees with the
// pull request check.
func TestGolangciLintPinsAgree(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	pins := make(map[string]string, len(golangciPinSites))
	for _, rel := range golangciPinSites {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		m := golangciPinRE.FindStringSubmatch(string(data))
		if m == nil {
			t.Errorf("%s pins no golangci-lint version; either the pin moved or the pattern has gone stale", rel)
			continue
		}
		pins[rel] = m[1]
	}
	if len(pins) != len(golangciPinSites) {
		return
	}

	want := pins[golangciPinSites[0]]
	for _, rel := range golangciPinSites[1:] {
		if pins[rel] != want {
			t.Errorf("%s pins golangci-lint %s, but %s pins %s",
				rel, pins[rel], golangciPinSites[0], want)
		}
	}
}
