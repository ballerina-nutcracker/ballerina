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

// Package uri provides DocumentURI identity for the language server core.
// A DocumentURI is a validated, scheme-typed URI value. Constructors parse
// and validate the scheme; only file: URIs are admitted end-to-end in Phase A.
// Broader URI routing (expr:, ai:, bala: overlay layers) is deferred to
// ticket 08.
package uri

import (
	"fmt"
	"net/url"
)

type scheme string

const (
	schemeFile scheme = "file"
	schemeExpr scheme = "expr"
	schemeAI   scheme = "ai"
	schemeBala scheme = "bala"
)

// DocumentURI is a validated document URI identity value. Construction is via
// NewFileURI, NewExprURI, NewAIURI, or NewBalaURI, each of which parses and
// validates the scheme. All fields are comparable, so DocumentURI can be used
// directly as a map key.
type DocumentURI struct {
	scheme scheme
	raw    string
	path   string
}

// NewFileURI parses raw as a file: URI and returns a DocumentURI. Returns an
// error if raw is not a valid file: URI. This replaces the former isFileURI
// bool check — callers handle the error instead of a false return.
func NewFileURI(raw string) (DocumentURI, error) {
	return newURI(raw, schemeFile)
}

// NewExprURI parses raw as an expr: URI and returns a DocumentURI.
// Phase-A scope: the constructor validates scheme; expr: is not admitted
// end-to-end until ticket 08.
func NewExprURI(raw string) (DocumentURI, error) {
	return newURI(raw, schemeExpr)
}

// NewAIURI parses raw as an ai: URI and returns a DocumentURI.
// Phase-A scope: the constructor validates scheme; ai: is not admitted
// end-to-end until ticket 08.
func NewAIURI(raw string) (DocumentURI, error) {
	return newURI(raw, schemeAI)
}

// NewBalaURI parses raw as a bala: URI and returns a DocumentURI.
// Phase-A scope: the constructor validates scheme; bala: is not admitted
// end-to-end until ticket 08.
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
		u.path = parsed.Path
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

// Path returns the file system path for a file: URI. It panics for non-file
// schemes per ADR-034 — non-file schemes do not have a file system path.
func (u DocumentURI) Path() string {
	if u.scheme != schemeFile {
		panic(fmt.Sprintf("uri: Path() called on non-file scheme %q", u.scheme))
	}
	return u.path
}

// Identity returns a stable string suitable for use as a map key or
// comparison basis. The identity is the raw URI string.
func (u DocumentURI) Identity() string {
	return u.raw
}

// String returns the raw URI string.
func (u DocumentURI) String() string {
	return u.raw
}
