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
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"
)

// resolveIncludePaths resolves the `include` glob patterns declared in
// Ballerina.toml against root (a directory within fsys, per the fs.FS
// convention "." denotes fsys's own root), returning root-relative paths
// (slash-separated) of every matching file or directory. A pattern prefixed
// with "!" removes previously matched paths instead of adding to them.
// Java source: io.ballerina.projects.util.ProjectUtils#getPathsMatchingIncludePatterns
func resolveIncludePaths(fsys fs.FS, patterns []string, root string) ([]string, error) {
	var matched []string
	for _, pattern := range patterns {
		if after, ok := strings.CutPrefix(pattern, "!"); ok {
			removed, err := matchIncludePattern(fsys, after, root)
			if err != nil {
				return nil, err
			}
			matched = slices.DeleteFunc(matched, func(p string) bool {
				return slices.Contains(removed, p)
			})
			continue
		}
		added, err := matchIncludePattern(fsys, pattern, root)
		if err != nil {
			return nil, err
		}
		matched = append(matched, added...)
	}
	return matched, nil
}

// matchIncludePattern walks root within fsys and returns every root-relative
// path matching pattern.
func matchIncludePattern(fsys fs.FS, pattern, root string) ([]string, error) {
	re, err := globToRegexp(pattern)
	if err != nil {
		return nil, fmt.Errorf("writeBala: invalid include pattern %q: %w", pattern, err)
	}

	var matches []string
	err = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel := relativeToRoot(root, p)

		if rel == TargetDir || strings.HasPrefix(rel, TargetDir+"/") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if re.MatchString(rel) && isCorrectIncludeMatch(rel, d.IsDir(), pattern) {
			matches = append(matches, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("writeBala: failed to read files matching include pattern %q: %w", pattern, err)
	}
	return matches, nil
}

// relativeToRoot strips root's "root/" prefix from p (fs.FS-style
// slash-separated paths), leaving p unchanged when root is "." — fs.FS's
// own-root marker — or empty.
func relativeToRoot(root, p string) string {
	if root == "" || root == "." {
		return p
	}
	return strings.TrimPrefix(p, root+"/")
}

// joinRoot is relativeToRoot's inverse: it re-attaches root to a
// root-relative path to produce an fsys path suitable for fs.Stat/fs.ReadFile.
func joinRoot(root, rel string) string {
	if root == "" || root == "." {
		return rel
	}
	return root + "/" + rel
}

// isCorrectIncludeMatch applies the extra restrictions a leading/trailing
// slash in the original pattern imposes: a leading "/" restricts matches to
// root-level entries only, a trailing "/" restricts matches to directories.
func isCorrectIncludeMatch(rel string, isDir bool, pattern string) bool {
	if strings.HasPrefix(pattern, "/") && strings.Contains(rel, "/") {
		return false
	}
	if strings.HasSuffix(pattern, "/") && !isDir {
		return false
	}
	return true
}

// globToRegexp translates an include pattern into an anchored regexp
// equivalent to java.nio.file.FileSystem's "glob:**/<pattern>" matcher,
// applied to a "/"-separated relative path: "**" matches any number of path
// segments, "*" matches within a single segment, "?" matches one character,
// "[abc]"/"[a-z]"/"[!abc]" match a character class (optionally negated), and
// "{alt1,alt2}" matches any one of the comma-separated alternatives.
func globToRegexp(pattern string) (*regexp.Regexp, error) {
	trimmed := strings.TrimPrefix(strings.TrimRight(pattern, "/"), "/")

	var sb strings.Builder
	sb.WriteString("^(?:.*/)?")
	if err := writeGlobBody(&sb, []rune(trimmed)); err != nil {
		return nil, fmt.Errorf("%w in pattern %q", err, pattern)
	}
	sb.WriteString("$")
	return regexp.Compile(sb.String())
}

// writeGlobBody translates one glob segment (either the whole pattern, or
// one "{...}" alternative) into regexp syntax, appending it to sb.
func writeGlobBody(sb *strings.Builder, runes []rune) error {
	for i := 0; i < len(runes); i++ {
		switch c := runes[i]; c {
		case '*':
			if i+1 < len(runes) && runes[i+1] == '*' {
				sb.WriteString(".*")
				i++
				if i+1 < len(runes) && runes[i+1] == '/' {
					i++
				}
			} else {
				sb.WriteString("[^/]*")
			}
		case '?':
			sb.WriteString("[^/]")
		case '[':
			end := indexRune(runes, i+1, ']')
			if end == -1 {
				return errors.New("unterminated character class")
			}
			body := string(runes[i+1 : end])
			sb.WriteString("[")
			if after, ok := strings.CutPrefix(body, "!"); ok {
				sb.WriteString("^")
				body = after
			}
			sb.WriteString(body)
			sb.WriteString("]")
			i = end
		case '{':
			end := indexRune(runes, i+1, '}')
			if end == -1 {
				return errors.New("unterminated brace alternation")
			}
			sb.WriteString("(?:")
			for j, alt := range strings.Split(string(runes[i+1:end]), ",") {
				if j > 0 {
					sb.WriteString("|")
				}
				if err := writeGlobBody(sb, []rune(alt)); err != nil {
					return err
				}
			}
			sb.WriteString(")")
			i = end
		default:
			sb.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return nil
}

// indexRune returns the index of the first occurrence of target in
// runes[from:], or -1 if not found.
func indexRune(runes []rune, from int, target rune) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == target {
			return i
		}
	}
	return -1
}
