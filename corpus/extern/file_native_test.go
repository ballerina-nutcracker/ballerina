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

package extern_test

import (
	"os"
	"path/filepath"
	"testing"

	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/test_util/testharness"
	"ballerina-lang-go/values"
)

// skipIfRoot skips permission-dependent tests when running as root, since
// root bypasses all POSIX permission checks (chmod-denied directories would
// still be writable/readable/traversable), making the expected failures
// impossible to observe.
func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("skipping permission-dependent test when running as root")
	}
}

// mustChmod chmods path and registers a cleanup that restores a writable
// mode before t.TempDir()'s own cleanup tries to remove the tree — cleanups
// run LIFO, so this must be registered after the chmod that restricts it.
func mustChmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(path, 0o755)
	})
}

// TestFileNativePermissionErrors exercises the permission-denied branches of
// createDir/create/rename/remove/normalizePath(SYMLINK)/readDir that require
// a real, restricted-permission directory on disk — not reachable through
// pure .bal setup since the file module exposes no chmod-equivalent.
func TestFileNativePermissionErrors(t *testing.T) {
	skipIfRoot(t)

	root := t.TempDir()

	noWriteParent := filepath.Join(root, "no-write-parent")
	if err := os.Mkdir(noWriteParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noWriteParent, "existing.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	mustChmod(t, noWriteParent, 0o555) // r-xr-xr-x: entries stat-able, not writable

	noExecParent := filepath.Join(root, "no-exec-parent")
	if err := os.Mkdir(noExecParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noExecParent, "child.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	mustChmod(t, noExecParent, 0o600) // rw-------: not even traversable

	noAccess := filepath.Join(root, "no-access")
	if err := os.Mkdir(noAccess, 0o755); err != nil {
		t.Fatal(err)
	}
	mustChmod(t, noAccess, 0o000) // stat-able via its (accessible) parent, not listable

	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "file-permission-v", FuncName: "noWriteParentDir",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return noWriteParent, nil
			}},
		{Org: "$anon", Module: "file-permission-v", FuncName: "noExecParentDir",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return noExecParent, nil
			}},
		{Org: "$anon", Module: "file-permission-v", FuncName: "noAccessDir",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return noAccess, nil
			}},
	}
	runExtern(t, fileCase("file-native/file-permission-v"), testharness.NewTestPal(), externs)
}

// TestFileNativeSymlinkResolve exercises the successful branch of
// normalizePath(SYMLINK) (native `resolve`), which requires a real OS-level
// symlink — the file module has no symlink-creation function to build one
// from pure .bal.
func TestFileNativeSymlinkResolve(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink("target-file.txt", link); err != nil {
		t.Fatal(err)
	}

	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "file-symlink-v", FuncName: "symlinkPath",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return link, nil
			}},
	}
	runExtern(t, fileCase("file-native/file-symlink-v"), testharness.NewTestPal(), externs)
}
