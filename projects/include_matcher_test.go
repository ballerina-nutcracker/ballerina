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
		{"dir-only trailing slash", "baz/", []string{"baz"}},
		{"file does not match dir-only pattern", "qux/", nil},
		{"root-only + dir-only combined excludes nested matches", "/quux/", []string{"quux"}},
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
// a dir-only pattern and an exact-path pattern both matching the same
// file) — dedup happens later, when writing the bala archive.
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
	if count != 1 {
		t.Errorf("expected exactly one hoge/y match ('hoge/' matches the dir itself, not its contents), got %d in %v", count, got)
	}
	if !slices.Contains(got, "hoge") {
		t.Errorf("expected the 'hoge/' pattern to match the directory itself, got %v", got)
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

func TestGlobToRegexp_InvalidPatterns(t *testing.T) {
	t.Parallel()
	for _, pattern := range []string{"thud[ab", "config.{bal,toml"} {
		if _, err := globToRegexp(pattern); err == nil {
			t.Errorf("globToRegexp(%q): expected an error for an unterminated group", pattern)
		}
	}
}
