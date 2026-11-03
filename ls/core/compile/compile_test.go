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

package compile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
)

func newTestServices(t *testing.T) (*workspace.ProjectService, *CompilationService) {
	t.Helper()
	bus := event.New()
	projects := workspace.New(os.DirFS, bus)
	svc := New(projects, bus, WithDebounce(0))
	t.Cleanup(func() { svc.Shutdown(); bus.Close() })
	return projects, svc
}

func fileURI(t *testing.T, nativePath string) workspace.DocumentURI {
	t.Helper()
	u, err := workspace.FileURIFromPath(nativePath)
	if err != nil {
		t.Fatalf("FileURIFromPath(%q): %v", nativePath, err)
	}
	return u
}

func tempFileURI(t *testing.T, name string) workspace.DocumentURI {
	t.Helper()
	return fileURI(t, filepath.Join(t.TempDir(), name))
}

func TestFileURIFromNativePath(t *testing.T) {
	t.Parallel()
	nativePath := filepath.Join(t.TempDir(), "space dir", "main #.bal")
	u := fileURI(t, nativePath)
	if u.Path() != nativePath {
		t.Fatalf("path = %q, want %q", u.Path(), nativePath)
	}
	parsed, err := workspace.NewFileURI(u.String())
	if err != nil || parsed != u {
		t.Fatalf("round trip = %v, %v, want %v", parsed, err, u)
	}
	if _, err := os.Stat(nativePath); !os.IsNotExist(err) {
		t.Fatalf("URI conversion must leave the document absent: %v", err)
	}
}

func applyOpen(t *testing.T, projects *workspace.ProjectService, u workspace.DocumentURI, text string) {
	t.Helper()
	if _, err := projects.Apply(context.Background(), workspace.DocumentChange{
		Kind: workspace.ChangeOpen, URI: u, Text: text, Version: 1, LanguageID: "ballerina",
	}); err != nil {
		t.Fatalf("Apply open: %v", err)
	}
}

func TestRunCycleAfterShutdownDoesNotStartCompilation(t *testing.T) {
	t.Parallel()
	_, svc := newTestServices(t)
	svc.Shutdown()
	svc.runCycle(nil)
}

func TestByteOffsetToLineCharUTF16(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		text     string
		offset   int
		wantLine uint32
		wantChar uint32
		wantOK   bool
	}{
		{name: "ascii start", text: "abc\ndef\n", offset: 0, wantLine: 0, wantChar: 0, wantOK: true},
		{name: "ascii mid line0", text: "abc\ndef\n", offset: 2, wantLine: 0, wantChar: 2, wantOK: true},
		{name: "ascii at newline belongs to line0 end", text: "abc\ndef\n", offset: 3, wantLine: 0, wantChar: 3, wantOK: true},
		{name: "ascii line1 start", text: "abc\ndef\n", offset: 4, wantLine: 1, wantChar: 0, wantOK: true},
		{name: "ascii end of text trailing line", text: "abc\ndef\n", offset: 8, wantLine: 2, wantChar: 0, wantOK: true},
		{name: "crlf single break line1 start", text: "ab\r\ncd", offset: 4, wantLine: 1, wantChar: 0, wantOK: true},
		{name: "crlf at cr belongs to line0 end", text: "ab\r\ncd", offset: 2, wantLine: 0, wantChar: 2, wantOK: true},
		{name: "cr-only break", text: "ab\rcd", offset: 3, wantLine: 1, wantChar: 0, wantOK: true},
		{name: "surrogate pair byte char", text: "𐐀x", offset: 4, wantLine: 0, wantChar: 4, wantOK: true},
		{name: "offset after surrogate at ascii", text: "𐐀x", offset: 5, wantLine: 0, wantChar: 5, wantOK: true},
		{name: "surrogate on later line", text: "𐐀\n𐐀z", offset: 9, wantLine: 1, wantChar: 4, wantOK: true},
		{name: "negative offset invalid", text: "abc", offset: -1, wantOK: false},
		{name: "offset past end invalid", text: "abc", offset: 4, wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lineStarts := computeLineStarts(tc.text)
			line, char, ok := byteOffsetToLineChar(tc.text, lineStarts, tc.offset)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if line != tc.wantLine {
				t.Errorf("line = %d, want %d", line, tc.wantLine)
			}
			if char != tc.wantChar {
				t.Errorf("char = %d, want %d", char, tc.wantChar)
			}
		})
	}
}
