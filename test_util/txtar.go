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

package test_util

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

// NormalizeNewlines converts CRLF to LF and trims trailing newlines from the end of s.
func NormalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimRight(s, "\n")
}

// FormatExpectedGot formats a diff-style block for test failure messages.
func FormatExpectedGot(expected, got string) string {
	return "expected:\n" + formatIndentedLines(expected) + "\n\ngot:\n" + formatIndentedLines(got)
}

// LoadTxtarStdoutStderr parses a txtar archive that must contain exactly stdout and stderr members.
func LoadTxtarStdoutStderr(txtarPath string) (stdout, stderr string, err error) {
	archive, err := txtar.ParseFile(txtarPath)
	if err != nil {
		return "", "", err
	}

	var stdoutFound, stderrFound bool
	for _, f := range archive.Files {
		switch f.Name {
		case "stdout":
			stdout = string(f.Data)
			stdoutFound = true
		case "stderr":
			stderr = string(f.Data)
			stderrFound = true
		default:
			return "", "", fmt.Errorf("unexpected file %q (only stdout/stderr are allowed)", f.Name)
		}
	}

	if !stdoutFound || !stderrFound {
		return "", "", fmt.Errorf("missing required files (need stdout and stderr)")
	}

	return stdout, stderr, nil
}

// LoadTxtarStdoutStderrExitcode parses a txtar archive with stdout, stderr, and exitcode members.
// Each field is passed through NormalizeNewlines.
func LoadTxtarStdoutStderrExitcode(txtarPath string) (stdout, stderr, exitCode string, err error) {
	archive, err := txtar.ParseFile(txtarPath)
	if err != nil {
		return "", "", "", err
	}

	var stdoutFound, stderrFound, exitFound bool
	for _, f := range archive.Files {
		switch f.Name {
		case "stdout":
			stdout = NormalizeNewlines(string(f.Data))
			stdoutFound = true
		case "stderr":
			stderr = NormalizeNewlines(string(f.Data))
			stderrFound = true
		case "exitcode":
			exitCode = NormalizeNewlines(string(f.Data))
			exitFound = true
		default:
			return "", "", "", fmt.Errorf("unexpected file %q in %s", f.Name, txtarPath)
		}
	}

	if !stdoutFound || !stderrFound || !exitFound {
		return "", "", "", fmt.Errorf("missing stdout/stderr/exitcode entries in %s", txtarPath)
	}

	return stdout, stderr, exitCode, nil
}

// TxtarFilesStdoutStderr builds txtar file entries for stdout and stderr.
func TxtarFilesStdoutStderr(stdout, stderr string) []txtar.File {
	return []txtar.File{
		{Name: "stdout", Data: []byte(stdout)},
		{Name: "stderr", Data: []byte(stderr)},
	}
}

// UpdateTxtarArchiveIfNeeded writes the given txtar files to path when content differs from the existing file.
// It returns true when a new or changed file was written.
func UpdateTxtarArchiveIfNeeded(t *testing.T, expectedPath string, files []txtar.File) bool {
	t.Helper()
	archive := &txtar.Archive{Files: files}
	actual := txtar.Format(archive)

	existing, err := os.ReadFile(expectedPath)
	fileExists := err == nil
	if fileExists && bytes.Equal(existing, actual) {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(expectedPath), 0o755); err != nil {
		t.Fatalf("failed to create output directory for %s: %v", expectedPath, err)
	}
	if err := os.WriteFile(expectedPath, actual, 0o644); err != nil {
		t.Fatalf("failed to write output archive %s: %v", expectedPath, err)
	}
	return true
}

func formatIndentedLines(s string) string {
	const indent = "\t"
	if s == "" {
		return indent + "(empty)"
	}
	var b strings.Builder
	for line := range strings.SplitSeq(s, "\n") {
		b.WriteString(indent)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}
