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
	goruntime "runtime"
	"testing"

	"ballerina-lang-go/test_util/testharness"
)

// skipIfNoFileWatch skips on platforms without a native directory-watch
// backend (js/wasm): fsnotify has no js/wasm backend, so file:Listener's
// 'start() always fails there with "fsnotify not supported on the current
// platform".
func skipIfNoFileWatch(t *testing.T) {
	t.Helper()
	if goruntime.GOOS == "js" {
		t.Skip("skipping file-watch-dependent test on js/wasm")
	}
}

// TestFileListenerEvents exercises the create/modify/delete dispatch of a
// single file:Listener service against a real OS-level directory watch.
func TestFileListenerEvents(t *testing.T) {
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-events-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerRecursive exercises dynamic recursive registration: a
// subdirectory created after start is itself watched for further events.
func TestFileListenerRecursive(t *testing.T) {
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-recursive-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerMultiService exercises dispatch to every service attached
// to the same listener.
func TestFileListenerMultiService(t *testing.T) {
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-multi-service-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerDetach exercises attach/detach: a detached service must
// stop receiving events.
func TestFileListenerDetach(t *testing.T) {
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-detach-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerAttachError exercises the attach-time validation requiring
// at least one of onCreate/onModify/onDelete.
func TestFileListenerAttachError(t *testing.T) {
	runExtern(t, fileCase("file-listener/file-listener-attach-error-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerInitError exercises Listener init validation: empty path,
// non-existent directory, and a path that is not a directory.
func TestFileListenerInitError(t *testing.T) {
	runExtern(t, fileCase("file-listener/file-listener-init-error-v"), testharness.NewTestPal(), nil)
}

// TestFileListenerStartError exercises start() failing when the watched
// directory is removed between init and start, so the OS-level watch add
// fails.
func TestFileListenerStartError(t *testing.T) {
	skipIfNoFileWatch(t)
	runExtern(t, fileCase("file-listener/file-listener-start-error-v"), testharness.NewTestPal(), nil)
}
