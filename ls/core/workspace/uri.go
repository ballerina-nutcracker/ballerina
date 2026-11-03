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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package workspace

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

type scheme string

const (
	schemeFile scheme = "file"
	schemeExpr scheme = "expr"
	schemeAI   scheme = "ai"
	schemeBala scheme = "bala"
)

// DocumentURI is a comparable URI identity suitable for map keys.
// Use the scheme-specific constructors to parse and validate input.
type DocumentURI struct {
	scheme scheme
	raw    string
	path   string
}

// NewFileURI parses and canonicalizes a local, absolute file: URI. Unsupported
// authorities, queries, fragments, and path forms return an error.
func NewFileURI(raw string) (DocumentURI, error) {
	return newURI(raw, schemeFile)
}

// NewExprURI parses raw as an expr: URI and validates its scheme.
func NewExprURI(raw string) (DocumentURI, error) {
	return newURI(raw, schemeExpr)
}

// NewAIURI parses raw as an ai: URI and validates its scheme.
func NewAIURI(raw string) (DocumentURI, error) {
	return newURI(raw, schemeAI)
}

// NewBalaURI parses raw as a bala: URI and validates its scheme.
func NewBalaURI(raw string) (DocumentURI, error) {
	return newURI(raw, schemeBala)
}

func newURI(raw string, expected scheme) (DocumentURI, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return DocumentURI{}, fmt.Errorf("uri: parse %q: %w", raw, err)
	}
	if parsed.Scheme != string(expected) {
		return DocumentURI{}, fmt.Errorf("uri: expected scheme %q, got %q", expected, parsed.Scheme)
	}
	u := DocumentURI{scheme: scheme(parsed.Scheme), raw: raw}
	if parsed.Scheme == string(schemeFile) {
		p := parsed.Path
		if parsed.Host != "" || parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.ContainsAny(p, "\\\\\x00") {
			return DocumentURI{}, fmt.Errorf("uri: unsupported file URI %q", raw)
		}
		if len(p) >= 3 && p[2] == ':' {
			if !isDriveLetter(p[1]) || len(p) < 4 || p[3] != '/' {
				return DocumentURI{}, fmt.Errorf("uri: drive-relative path %q", raw)
			}
			p = "/" + strings.ToUpper(p[1:2]) + p[2:]
		}
		u.raw = (&url.URL{Scheme: "file", Path: p}).String()
		if runtime.GOOS == "windows" {
			if len(p) < 4 || p[2] != ':' {
				return DocumentURI{}, fmt.Errorf("uri: expected absolute drive path %q", raw)
			}
			p = p[1:]
		}
		u.path = filepath.FromSlash(p)
	}
	return u, nil
}

// Scheme returns the URI scheme (e.g. "file", "expr", "ai", "bala").
func (u DocumentURI) Scheme() string {
	return string(u.scheme)
}

// IsFile reports whether the URI is a file: URI.
func (u DocumentURI) IsFile() bool {
	return u.scheme == schemeFile
}

// Path returns the native filesystem path for a file: URI and panics for
// non-file schemes.
func (u DocumentURI) Path() string {
	if u.scheme != schemeFile {
		panic(fmt.Sprintf("uri: Path() called on non-file scheme %q", u.scheme))
	}
	return u.path
}

// Identity returns the URI identity. File URIs have canonical escaping and
// uppercase drive letters; non-file URIs retain their input spelling.
func (u DocumentURI) Identity() string {
	return u.raw
}

// String returns the same URI string as Identity.
func (u DocumentURI) String() string {
	return u.raw
}

func isDriveLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// FileURIFromPath converts an absolute native path without consulting the host filesystem.
func FileURIFromPath(p string) (DocumentURI, error) {
	if strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//") || !filepath.IsAbs(p) {
		return DocumentURI{}, fmt.Errorf("uri: unsupported native path %q", p)
	}
	p = filepath.ToSlash(p)
	if runtime.GOOS == "windows" {
		p = "/" + p
	}
	return NewFileURI((&url.URL{Scheme: "file", Path: p}).String())
}
