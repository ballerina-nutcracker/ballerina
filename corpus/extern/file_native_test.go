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
	goruntime "runtime"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
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

// skipIfWindows skips permission-dependent tests on Windows: os.Chmod there
// only toggles the read-only attribute (no execute bit gating directory
// traversal, no separate read/write/execute semantics), so the POSIX chmod
// modes used to deny access below don't produce the expected failures.
func skipIfWindows(t *testing.T) {
	t.Helper()
	if goruntime.GOOS == "windows" {
		t.Skip("skipping POSIX permission-dependent test on Windows")
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
	t.Parallel()
	skipIfWindows(t)
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

// TestFileNativeDirWritableByOwner checks that test(WRITABLE)/getMetaData
// report a directory's writability for the calling user rather than any write
// bit. Skipped on non-unix targets (e.g. WASM) where there is no access(2) or
// uid, so the check falls back to the mode bits.
func TestFileNativeDirWritableByOwner(t *testing.T) {
	t.Parallel()
	skipIfWindows(t)
	skipIfRoot(t)
	if goruntime.GOOS == "js" || goruntime.GOOS == "wasip1" {
		t.Skip("skipping: no per-user access check on WASM")
	}

	othersWritable := filepath.Join(t.TempDir(), "others-writable")
	if err := os.Mkdir(othersWritable, 0o755); err != nil {
		t.Fatal(err)
	}
	mustChmod(t, othersWritable, 0o577) // r-xrwxrwx: write bits set, but not for the owner

	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "file-writable-v", FuncName: "othersWritableDir",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return othersWritable, nil
			}},
	}
	runExtern(t, fileCase("file-native/file-writable-v"), testharness.NewTestPal(), externs)
}

// TestFileNativeSymlinkResolve exercises the successful branch of
// normalizePath(SYMLINK) (native `resolve`), which requires a real OS-level
// symlink — the file module has no symlink-creation function to build one
// from pure .bal.
func TestFileNativeSymlinkResolve(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "target-file.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink("target-file.txt", link); err != nil {
		t.Fatal(err)
	}

	symlinkDir := filepath.Join(root, "symlink-dir")
	if err := os.Mkdir(symlinkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(symlinkDir, "target-file.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target-file.txt", filepath.Join(symlinkDir, "link")); err != nil {
		t.Fatal(err)
	}

	dirTarget := filepath.Join(root, "dir-target")
	if err := os.Mkdir(dirTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirTarget, "a.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	dirLinkParent := filepath.Join(root, "dir-link-parent")
	if err := os.Mkdir(dirLinkParent, 0o755); err != nil {
		t.Fatal(err)
	}
	dirLink := filepath.Join(dirLinkParent, "dir-link")
	if err := os.Symlink(filepath.Join("..", "dir-target"), dirLink); err != nil {
		t.Fatal(err)
	}

	tree := filepath.Join(root, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "dir-target"), filepath.Join(tree, "sub-link")); err != nil {
		t.Fatal(err)
	}
	loop := filepath.Join(root, "loop")
	if err := os.Mkdir(loop, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".", filepath.Join(loop, "self")); err != nil {
		t.Fatal(err)
	}

	externs := []testharness.ExternRegistration{
		{Org: "$anon", Module: "file-symlink-v", FuncName: "treeWithDirLinkPath",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return tree, nil
			}},
		{Org: "$anon", Module: "file-symlink-v", FuncName: "loopDirPath",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return loop, nil
			}},
		{Org: "$anon", Module: "file-symlink-v", FuncName: "dirLinkPath",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return dirLink, nil
			}},
		{Org: "$anon", Module: "file-symlink-v", FuncName: "symlinkPath",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return link, nil
			}},
		{Org: "$anon", Module: "file-symlink-v", FuncName: "symlinkDirPath",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return symlinkDir, nil
			}},
	}
	runExtern(t, fileCase("file-native/file-symlink-v"), testharness.NewTestPal(), externs)
}

// TestFileNativeCopyReplace exercises copy onto the source itself, a hard
// link to it, a symlink to it, and a hard-linked destination replaced with
// REPLACE_EXISTING. The file module can create none of these links, so the Go
// test builds them.
func TestFileNativeCopyReplace(t *testing.T) {
	t.Parallel()
	skipIfWindows(t)
	root := t.TempDir()
	src := filepath.Join(root, "source.txt")
	if err := os.WriteFile(src, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hardLink := filepath.Join(root, "hard.txt")
	if err := os.Link(src, hardLink); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink("source.txt", link); err != nil {
		t.Fatal(err)
	}

	replaceDst := filepath.Join(root, "replace-dst.txt")
	if err := os.WriteFile(replaceDst, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	replaceDstHardLink := filepath.Join(root, "replace-dst-hard.txt")
	if err := os.Link(replaceDst, replaceDstHardLink); err != nil {
		t.Fatal(err)
	}

	pathExtern := func(name, path string) testharness.ExternRegistration {
		return testharness.ExternRegistration{Org: "$anon", Module: "file-copy-replace-v", FuncName: name,
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				return path, nil
			}}
	}
	externs := []testharness.ExternRegistration{
		pathExtern("sourcePath", src),
		pathExtern("hardLinkPath", hardLink),
		pathExtern("symlinkPath", link),
		pathExtern("replaceDstPath", replaceDst),
		pathExtern("replaceDstHardLinkPath", replaceDstHardLink),
	}
	runExtern(t, fileCase("file-native/file-copy-replace-v"), testharness.NewTestPal(), externs)
}
