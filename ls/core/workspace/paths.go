// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package workspace

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// SourcePaths pins a project's native filesystem root and compiler namespace.
// It is a value so sealed semantic views can retain it across project reloads.
type SourcePaths struct {
	root   string
	prefix string
}

// NameOf maps a contained editor document to its compiler registration name.
func (p SourcePaths) NameOf(u DocumentURI) (string, bool) {
	if !u.IsFile() || p.root == "" || !filepath.IsAbs(p.root) {
		return "", false
	}
	file := u.Path()
	if !filepath.IsAbs(file) {
		return "", false
	}
	name, err := filepath.Rel(p.root, file)
	name = filepath.ToSlash(name)
	if err != nil || name == ".." || strings.HasPrefix(name, "../") || !validSourceName(name) {
		return "", false
	}
	if filepath.Join(p.root, filepath.FromSlash(name)) != filepath.Clean(file) {
		return "", false
	}
	return p.prefix + name, true
}

// URIOf maps a compiler registration name to an editor URI.
// Names must be valid relative paths within the project's compiler namespace.
func (p SourcePaths) URIOf(name string) (DocumentURI, bool) {
	if p.root == "" || !filepath.IsAbs(p.root) || !validSourceName(name) {
		return DocumentURI{}, false
	}
	if p.prefix != "" {
		if !strings.HasPrefix(name, p.prefix) {
			return DocumentURI{}, false
		}
		name = strings.TrimPrefix(name, p.prefix)
		if !validSourceName(name) {
			return DocumentURI{}, false
		}
	}
	u, err := FileURIFromPath(filepath.Join(p.root, filepath.FromSlash(name)))
	return u, err == nil
}

func validSourceName(name string) bool {
	return fs.ValidPath(name) && !strings.ContainsAny(name, `\:`)
}
