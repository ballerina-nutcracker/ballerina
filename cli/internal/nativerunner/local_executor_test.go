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

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"ballerina-lang-go/cli/internal/nativeexec"
)

func TestVersionAtLeast(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.26", "1.26", true},
		{"1.26.1", "1.26", true},
		{"1.25", "1.26", false},
		{"2.0", "1.26", true},
		{"1.26", "1.26.0", true},
		{"1.26.0", "1.26", true},
		{"1.0", "2.0", false},
		{"1.26rc1", "1.26", false},
		{"1.26beta2", "1.26", false},
	}
	for _, tc := range cases {
		got := versionAtLeast(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("versionAtLeast(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestModuleDirName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ballerinax/redis-native", "ballerinax_redis-native"},
		{"a/b/c", "a_b_c"},
		{"noSlash", "noSlash"},
		{"example.com/org/pkg", "example.com_org_pkg"},
	}
	for _, tc := range cases {
		got := moduleDirName(tc.in)
		if got != tc.want {
			t.Errorf("moduleDirName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWriteNativeFiles_CopiesGoFiles(t *testing.T) {
	srcFS := fstest.MapFS{
		"main.go":       {Data: []byte("package main\n")},
		"sub/helper.go": {Data: []byte("package sub\n")},
		"README.md":     {Data: []byte("# readme\n")},
		"config.yaml":   {Data: []byte("key: value\n")},
	}
	payload := &nativeexec.GoSourcePayload{GoFiles: srcFS, Module: "example.com/pkg"}

	dir := t.TempDir()
	if err := writeNativeFiles(dir, payload); err != nil {
		t.Fatalf("writeNativeFiles: %v", err)
	}

	checkFileContent(t, filepath.Join(dir, "main.go"), "package main\n")
	checkFileContent(t, filepath.Join(dir, "sub", "helper.go"), "package sub\n")

	for _, name := range []string{"README.md", "config.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("non-.go file %q must not be copied", name)
		}
	}
}

func TestWritePatchedGoMod_AppendsRequireReplace(t *testing.T) {
	interpRoot := t.TempDir()
	origMod := "module ballerina-lang-go\n\ngo 1.26\n"
	mustWriteFile(t, filepath.Join(interpRoot, "go.mod"), origMod)
	mustWriteFile(t, filepath.Join(interpRoot, "go.sum"), "")

	tmpDir := t.TempDir()
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{Module: "example.com/mypkg"},
	}

	patchedModFile, err := writePatchedGoMod(tmpDir, interpRoot, payloads, tmpDir)
	if err != nil {
		t.Fatalf("writePatchedGoMod: %v", err)
	}

	content := mustReadFile(t, patchedModFile)
	if !strings.Contains(content, "require example.com/mypkg v0.0.0") {
		t.Errorf("patched go.mod missing require directive:\n%s", content)
	}
	if !strings.Contains(content, "replace example.com/mypkg =>") {
		t.Errorf("patched go.mod missing replace directive:\n%s", content)
	}
	// Original module declaration must be preserved.
	if !strings.Contains(content, "module ballerina-lang-go") {
		t.Errorf("patched go.mod missing original module declaration:\n%s", content)
	}
}

func TestWritePatchedGoMod_MultiplePayloads(t *testing.T) {
	interpRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(interpRoot, "go.mod"), "module ballerina-lang-go\n\ngo 1.26\n")
	mustWriteFile(t, filepath.Join(interpRoot, "go.sum"), "")

	tmpDir := t.TempDir()
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{Module: "example.com/pkgA"},
		&nativeexec.GoSourcePayload{Module: "example.com/pkgB"},
	}

	patchedModFile, err := writePatchedGoMod(tmpDir, interpRoot, payloads, tmpDir)
	if err != nil {
		t.Fatalf("writePatchedGoMod: %v", err)
	}

	content := mustReadFile(t, patchedModFile)
	for _, mod := range []string{"example.com/pkgA", "example.com/pkgB"} {
		if !strings.Contains(content, "require "+mod) {
			t.Errorf("patched go.mod missing require for %s:\n%s", mod, content)
		}
		if !strings.Contains(content, "replace "+mod) {
			t.Errorf("patched go.mod missing replace for %s:\n%s", mod, content)
		}
	}
}

func TestWritePatchedGoMod_WritesPatchedGoSum(t *testing.T) {
	interpRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(interpRoot, "go.mod"), "module ballerina-lang-go\n\ngo 1.26\n")
	mustWriteFile(t, filepath.Join(interpRoot, "go.sum"), "github.com/foo/bar v1.0.0 h1:xxx\n")

	tmpDir := t.TempDir()
	payloads := []nativeexec.NativePayload{
		&nativeexec.GoSourcePayload{Module: "example.com/pkg"},
	}

	_, err := writePatchedGoMod(tmpDir, interpRoot, payloads, tmpDir)
	if err != nil {
		t.Fatalf("writePatchedGoMod: %v", err)
	}

	sumContent := mustReadFile(t, filepath.Join(tmpDir, "patched-go.sum"))
	if !strings.Contains(sumContent, "github.com/foo/bar") {
		t.Errorf("patched-go.sum must copy interpreter go.sum content:\n%s", sumContent)
	}
}

func TestWritePatchedGoMod_MissingGoMod(t *testing.T) {
	_, err := writePatchedGoMod(t.TempDir(), "/nonexistent/root", nil, t.TempDir())
	if err == nil {
		t.Error("expected error when interpreter go.mod is missing")
	}
}

// helpers

func checkFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("%s: got %q, want %q", path, string(data), want)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
