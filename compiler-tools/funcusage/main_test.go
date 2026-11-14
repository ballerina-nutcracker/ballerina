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

package main

import (
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

var workspaceFiles = map[string]string{
	"go.work": "go 1.27\n\nuse (\n\t./lib\n\t./app\n\t./test_util\n)\n",

	"lib/go.mod": "module example.com/lib\n\ngo 1.27\n",
	"lib/lib.go": `package lib

func usedPrivate() {}

func unusedPrivate() {}

func recursivePrivate(n int) int {
	if n == 0 {
		return 0
	}
	return recursivePrivate(n - 1)
}

func testOnlyPrivate() {}

//funcusage:test-hook lets tests replace the clock
func hookPrivate() {}

//funcusage:test-hook
func hookWithoutReason() {}

//funcusage:test-hook stale
func hookUsed() {}

func valuePrivate() {}

var handler = valuePrivate

func debugOnlyPrivate() {}

func ExportedUsedByApp() { usedPrivate(); hookUsed() }

func ExportedUsedBySubpackage() {}

func ExportedUsedInPackage() {}

func ExportedUnused() {}

func ExportedTestOnly() {}

func Generic[T any](value T) T { return value }

type value struct{}

func (value) unusedMethod() {}

func (value) UnusedMethod() {}

func ExportedUsedByTestUtil() {}

//funcusage:test-hook exported for tests in other packages
func ExportedHookUsedInPackage() {}

func init() { ExportedUsedInPackage(); ExportedHookUsedInPackage(); buildHook() }
`,
	"lib/lib_test.go": "package lib\n\nfunc useForTest() { testOnlyPrivate() }\n",
	"lib/debug.go":    "//go:build debug\n\npackage lib\n\nfunc init() { debugOnlyPrivate() }\n\n//funcusage:test-hook\nfunc buildHook() {}\n",
	"lib/release.go":  "//go:build !debug\n\npackage lib\n\nfunc buildHook() {}\n",
	"lib/sub/sub.go":  "package sub\n\nimport \"example.com/lib\"\n\nfunc Use() { lib.ExportedUsedBySubpackage() }\n",

	"app/go.mod": "module example.com/app\n\ngo 1.27\n\nrequire example.com/lib v0.0.0\n",
	"app/main.go": `package main

import (
	"example.com/lib"
	"example.com/lib/sub"
)

func main() {
	lib.ExportedUsedByApp()
	sub.Use()
	_ = lib.Generic(1)
}
`,
	"app/main_test.go": "package main\n\nimport \"example.com/lib\"\n\nfunc useForTest() { lib.ExportedTestOnly() }\n",

	"test_util/go.mod":    "module example.com/testutil\n\ngo 1.27\n\nrequire example.com/lib v0.0.0\n",
	"test_util/helper.go": "package testutil\n\nimport \"example.com/lib\"\n\nfunc Helper() { lib.ExportedUsedByTestUtil() }\n",
}

func writeWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCheck(t *testing.T) {
	t.Parallel()
	root := writeWorkspace(t, workspaceFiles)
	findings, err := check(root, []string{"lib"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range findings {
		got = append(got, f.String())
	}
	want := []string{
		"lib/debug.go:7:1: funcusage: //funcusage:test-hook requires a reason",
		"lib/lib.go:5:6: funcusage: private function unusedPrivate is not used by production code; remove it, move it to a test file, or mark it //funcusage:test-hook <reason>",
		"lib/lib.go:7:6: funcusage: private function recursivePrivate is not used by production code; remove it, move it to a test file, or mark it //funcusage:test-hook <reason>",
		"lib/lib.go:14:6: funcusage: private function testOnlyPrivate is not used by production code; remove it, move it to a test file, or mark it //funcusage:test-hook <reason>",
		"lib/lib.go:19:1: funcusage: //funcusage:test-hook requires a reason",
		"lib/lib.go:22:1: funcusage: hookUsed is used by production code; remove the unnecessary //funcusage:test-hook directive",
		"lib/lib.go:35:6: funcusage: exported function ExportedUsedInPackage is not used outside package example.com/lib; unexport it",
		"lib/lib.go:37:6: funcusage: exported function ExportedUnused is not used by production code; remove it, move it to a test file, or mark it //funcusage:test-hook <reason>",
		"lib/lib.go:39:6: funcusage: exported function ExportedTestOnly is not used by production code; remove it, move it to a test file, or mark it //funcusage:test-hook <reason>",
		"lib/lib.go:49:6: funcusage: exported function ExportedUsedByTestUtil is not used by production code; remove it, move it to a test file, or mark it //funcusage:test-hook <reason>",
		"lib/lib.go:51:1: funcusage: ExportedHookUsedInPackage is used by production code; remove the unnecessary //funcusage:test-hook directive",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got:\n%v\nwant:\n%v", got, want)
	}
}

func TestCheckFailsOnBrokenSource(t *testing.T) {
	t.Parallel()
	files := map[string]string{}
	for name, content := range workspaceFiles {
		files[name] = content
	}
	files["lib/broken.go"] = "package lib\n\nfunc broken() { undefined() }\n"
	if _, err := check(writeWorkspace(t, files), []string{"lib"}); err == nil {
		t.Fatal("type errors must fail the check")
	}
}

func TestCheckFailsOnMissingModule(t *testing.T) {
	t.Parallel()
	if _, err := check(writeWorkspace(t, workspaceFiles), []string{"missing"}); err == nil {
		t.Fatal("missing module directory must fail the check")
	}
}

func TestDiagnostic(t *testing.T) {
	t.Parallel()
	f := finding{position: token.Position{Filename: "ast/nodes.go", Line: 7, Column: 6}, message: "m"}
	if got, want := f.String(), "ast/nodes.go:7:6: funcusage: m"; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestWorkspacePatternsWithSpaces(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "dir with space")
	for name, content := range workspaceFiles {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	patterns, err := workspacePatterns(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(patterns)
	if want := []string{"./app/...", "./lib/...", "./test_util/..."}; !slices.Equal(patterns, want) {
		t.Fatalf("got %v; want %v", patterns, want)
	}
}
