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

package projects

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"testing"
	"testing/fstest"
)

// includeMatcherFixture mirrors the shape of Java's
// TestBalaWriter#testBuildProjectWithIncludes fixture closely enough to
// exercise the same glob features: bare filenames, root-only ("/x"),
// dir-only ("x/"), extension globs, "?", character classes/ranges (with
// negation), leading/trailing/mid "**", "!" negation, exact paths, and
// target/ exclusion.
func includeMatcherFixture() fstest.MapFS {
	return fstest.MapFS{
		"foo":                        {},
		"sub/foo":                    {},
		"bar":                        {},
		"sub/bar":                    {},
		"baz/inner.txt":              {},
		"qux":                        {},
		"quux/inner.txt":             {},
		"sub/quux/inner.txt":         {},
		"index.html":                 {},
		"sub/page.html":              {},
		"fooXbar.txt":                {},
		"plugz":                      {},
		"plugzz":                     {},
		"thuda":                      {},
		"thudc":                      {},
		"fredq":                      {},
		"fredz":                      {},
		"grault/garply":              {},
		"sub/grault/garply":          {},
		"waldo/xyzzy/inner":          {},
		"babble/mid/bar":             {},
		"babble/bar":                 {},
		"babble/notbar":              {},
		"readme.rs":                  {},
		"corge.rs":                   {},
		"include-resources/thud":     {},
		"include-resources/x.js":     {},
		"hoge/y":                     {},
		"config.bal":                 {},
		"config.toml":                {},
		"config.json":                {},
		"target/generated.txt":       {},
		"target/cache/generated.txt": {},
	}
}

func TestMatchIncludePattern(t *testing.T) {
	t.Parallel()
	fsys := includeMatcherFixture()

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{"bare filename matches at any depth", "foo", []string{"foo", "sub/foo"}},
		{"root-only leading slash", "/bar", []string{"bar"}},
		// A directory match expands to its individual files (not the bare
		// directory path) so a later "!" pattern can negate one of them.
		{"dir-only trailing slash", "baz/", []string{"baz/inner.txt"}},
		{"file does not match dir-only pattern", "qux/", nil},
		{"root-only + dir-only combined excludes nested matches", "/quux/", []string{"quux/inner.txt"}},
		{"extension glob", "*.html", []string{"index.html", "sub/page.html"}},
		{"mid-glob", "foo*bar.*", []string{"fooXbar.txt"}},
		{"single-char wildcard excludes longer names", "plug?", []string{"plugz"}},
		{"character class excludes non-members", "thud[ab]", []string{"thuda"}},
		{"character range", "fred[q-s]", []string{"fredq"}},
		{"negated character class", "fred[!q-s]", []string{"fredz"}},
		{"leading doublestar", "**/grault/garply", []string{"grault/garply", "sub/grault/garply"}},
		{"trailing doublestar", "waldo/xyzzy/**", []string{"waldo/xyzzy/inner"}},
		// "**" matches zero or more path segments, so this also matches
		// babble/bar directly (no intervening directory) per Java's own
		// glob spec ("** matches zero or more characters crossing
		// directory boundaries").
		// babble/notbar is a deliberate near-miss fixture entry: "**/" must
		// preserve the path-segment boundary, so it must not match even
		// though "notbar" ends in "bar".
		{"mid doublestar", "babble/**/bar", []string{"babble/mid/bar", "babble/bar"}},
		{"exact nested path", "include-resources/thud", []string{"include-resources/thud"}},
		{"brace alternation", "config.{bal,toml}", []string{"config.bal", "config.toml"}},
		{"target dir always excluded", "*", nil}, // see explicit check below instead of relying on this row
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := matchIncludePattern(fsys, tc.pattern, ".")
			if err != nil {
				t.Fatalf("matchIncludePattern(%q): %v", tc.pattern, err)
			}
			if tc.name == "target dir always excluded" {
				if slices.ContainsFunc(got, func(p string) bool { return p == "target" || p == "target/generated.txt" }) {
					t.Errorf("expected target/ to never match, got %v", got)
				}
				return
			}
			sort.Strings(got)
			want := slices.Clone(tc.want)
			sort.Strings(want)
			if !slices.Equal(got, want) {
				t.Errorf("matchIncludePattern(%q) = %v, want %v", tc.pattern, got, want)
			}
		})
	}
}

// TestResolveIncludePaths_NegationIsOrderSensitive covers "!pattern"'s
// documented semantics: it removes matches accumulated by *prior* patterns
// only, not a final-result filter — a positive pattern listed after a
// negation re-adds anything the negation removed.
func TestResolveIncludePaths_NegationIsOrderSensitive(t *testing.T) {
	t.Parallel()
	fsys := includeMatcherFixture()

	got, err := resolveIncludePaths(fsys, []string{"*.rs", "!corge.rs"}, ".")
	if err != nil {
		t.Fatalf("resolveIncludePaths: %v", err)
	}
	if slices.Contains(got, "corge.rs") {
		t.Errorf("expected corge.rs to be removed by the negation, got %v", got)
	}
	if !slices.Contains(got, "readme.rs") {
		t.Errorf("expected readme.rs to remain, got %v", got)
	}

	// Re-adding a negated pattern via a later positive pattern restores it.
	got, err = resolveIncludePaths(fsys, []string{"*.rs", "!corge.rs", "corge.rs"}, ".")
	if err != nil {
		t.Fatalf("resolveIncludePaths: %v", err)
	}
	if !slices.Contains(got, "corge.rs") {
		t.Errorf("expected corge.rs restored by the later positive pattern, got %v", got)
	}
}

// TestResolveIncludePaths_OverlappingPatternsProduceDuplicates documents
// that resolveIncludePaths itself does not dedup overlapping matches (e.g.
// a dir-only pattern expanding to a file, and a separate exact-path pattern
// matching that same file) — dedup happens later, when writing the bala
// archive.
func TestResolveIncludePaths_OverlappingPatternsProduceDuplicates(t *testing.T) {
	t.Parallel()
	fsys := includeMatcherFixture()

	got, err := resolveIncludePaths(fsys, []string{"hoge/", "hoge/y"}, ".")
	if err != nil {
		t.Fatalf("resolveIncludePaths: %v", err)
	}
	count := 0
	for _, p := range got {
		if p == "hoge/y" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected two hoge/y matches ('hoge/' expands to its files, 'hoge/y' matches directly), got %d in %v", count, got)
	}
}

// TestResolveIncludePaths_NegationAppliesInsideMatchedDirectory covers the
// fix for a real data-exposure bug: a dir-only pattern used to add a single
// bare directory entry, so a later "!" pattern negating one specific file
// inside that directory never matched anything in the accumulated list, and
// the file was archived anyway when the directory was later copied
// recursively. The dir-only match must expand to individual files so the
// negation can remove just one of them.
func TestResolveIncludePaths_NegationAppliesInsideMatchedDirectory(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"assets/data.txt":   {},
		"assets/secret.txt": {},
	}

	got, err := resolveIncludePaths(fsys, []string{"assets/", "!assets/secret.txt"}, ".")
	if err != nil {
		t.Fatalf("resolveIncludePaths: %v", err)
	}
	if slices.Contains(got, "assets") {
		t.Errorf("expected no bare directory entry, got %v", got)
	}
	if slices.Contains(got, "assets/secret.txt") {
		t.Errorf("expected assets/secret.txt to be excluded by the negation, got %v", got)
	}
	if !slices.Contains(got, "assets/data.txt") {
		t.Errorf("expected assets/data.txt to remain included, got %v", got)
	}
}

// TestResolveIncludePaths_NonExistentPatternIsNoOp mirrors Java's
// (unwired) projectWithNonExistingIncludes fixture: a pattern matching
// nothing is not an error.
func TestResolveIncludePaths_NonExistentPatternIsNoOp(t *testing.T) {
	t.Parallel()
	fsys := includeMatcherFixture()

	got, err := resolveIncludePaths(fsys, []string{"does-not-exist.txt", "does-not-exist-dir"}, ".")
	if err != nil {
		t.Fatalf("resolveIncludePaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no matches for non-existent patterns, got %v", got)
	}
}

// TestMatchIncludePattern_SkipsSymlinks uses a real filesystem (fstest.MapFS
// has no symlink concept) to verify a symlink is never collected — as both a
// direct file match and inside an expanded directory match — since
// addIncludeFile's later fs.ReadFile would transparently follow it,
// letting an include pattern archive a file from outside the project root.
func TestMatchIncludePattern_SkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires elevated privileges on windows")
	}
	t.Parallel()

	outsideDir := t.TempDir()
	secretPath := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("outside the project root"), 0o644); err != nil {
		t.Fatalf("writing secret file: %v", err)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("creating assets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "data.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatalf("writing assets/data.txt: %v", err)
	}
	// A symlink matched directly by a glob ("*.txt" at root).
	if err := os.Symlink(secretPath, filepath.Join(root, "escape.txt")); err != nil {
		t.Fatalf("creating root symlink: %v", err)
	}
	// A symlink matched only via directory expansion ("assets/").
	if err := os.Symlink(secretPath, filepath.Join(root, "assets", "escape.txt")); err != nil {
		t.Fatalf("creating nested symlink: %v", err)
	}

	fsys := os.DirFS(root)

	t.Run("direct glob match", func(t *testing.T) {
		t.Parallel()
		got, err := matchIncludePattern(fsys, "*.txt", ".")
		if err != nil {
			t.Fatalf("matchIncludePattern: %v", err)
		}
		if slices.Contains(got, "escape.txt") {
			t.Errorf("expected the symlink to be skipped, got %v", got)
		}
	})

	t.Run("directory expansion", func(t *testing.T) {
		t.Parallel()
		got, err := matchIncludePattern(fsys, "assets/", ".")
		if err != nil {
			t.Fatalf("matchIncludePattern: %v", err)
		}
		if slices.Contains(got, "assets/escape.txt") {
			t.Errorf("expected the nested symlink to be skipped, got %v", got)
		}
		if !slices.Contains(got, "assets/data.txt") {
			t.Errorf("expected assets/data.txt to remain included, got %v", got)
		}
	})
}

// TestRelativeToRootAndJoinRoot_NonTrivialRoot covers the root != "."
// branch of both functions — exercised in production when packing a
// workspace member package, whose SourceRoot() is its own subdirectory
// (e.g. "pkg-a") rather than fsys's own root.
func TestRelativeToRootAndJoinRoot_NonTrivialRoot(t *testing.T) {
	t.Parallel()

	if got := relativeToRoot(".", "foo/bar"); got != "foo/bar" {
		t.Errorf(`relativeToRoot(".", "foo/bar") = %q, want "foo/bar"`, got)
	}
	if got := joinRoot(".", "foo/bar"); got != "foo/bar" {
		t.Errorf(`joinRoot(".", "foo/bar") = %q, want "foo/bar"`, got)
	}

	if got := relativeToRoot("pkg-a", "pkg-a/assets/logo.png"); got != "assets/logo.png" {
		t.Errorf(`relativeToRoot("pkg-a", "pkg-a/assets/logo.png") = %q, want "assets/logo.png"`, got)
	}
	if got := joinRoot("pkg-a", "assets/logo.png"); got != "pkg-a/assets/logo.png" {
		t.Errorf(`joinRoot("pkg-a", "assets/logo.png") = %q, want "pkg-a/assets/logo.png"`, got)
	}
	if got := relativeToRoot("pkg-a", joinRoot("pkg-a", "assets/logo.png")); got != "assets/logo.png" {
		t.Errorf(`relativeToRoot("pkg-a", joinRoot("pkg-a", "assets/logo.png")) = %q, want "assets/logo.png"`, got)
	}
}

// TestMatchIncludePattern_NonTrivialRoot verifies include-pattern matching
// itself (not just relativeToRoot/joinRoot in isolation) works against a
// workspace member's own subdirectory as root, rather than fsys's own root.
func TestMatchIncludePattern_NonTrivialRoot(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"pkg-a/Ballerina.toml":  {},
		"pkg-a/main.bal":        {},
		"pkg-a/assets/logo.png": {},
		"pkg-b/other.png":       {},
	}

	got, err := matchIncludePattern(fsys, "assets/", "pkg-a")
	if err != nil {
		t.Fatalf("matchIncludePattern: %v", err)
	}
	want := []string{"assets/logo.png"}
	sort.Strings(got)
	if !slices.Equal(got, want) {
		t.Errorf(`matchIncludePattern(fsys, "assets/", "pkg-a") = %v, want %v`, got, want)
	}
}

func TestGlobToRegexp_InvalidPatterns(t *testing.T) {
	t.Parallel()
	for _, pattern := range []string{"thud[ab", "config.{bal,toml"} {
		if _, err := globToRegexp(pattern); err == nil {
			t.Errorf("globToRegexp(%q): expected an error for an unterminated group", pattern)
		}
	}
}
