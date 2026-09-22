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

//go:build !js && !wasm

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateModule_CleansUpOnWriteFailure covers createModule's cleanup
// path: runAdd's own pre-check always guarantees modulePath doesn't exist
// yet, and its own name validation always rejects a "/"-containing module
// name before createModule is ever reached, so this branch has no CLI
// trigger point — it's exercised here directly (white-box) with a
// moduleName containing a path separator, forcing the os.WriteFile to fail
// against a missing intermediate directory, and asserts the directory is
// removed rather than left behind half-written.
func TestCreateModule_CleansUpOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	modulePath := filepath.Join(dir, "util")

	err := createModule(modulePath, "nonexistent/util", "public function hello() returns string {\n}\n")
	if err == nil {
		t.Fatal("expected an error writing to a path with a missing intermediate directory")
	}
	if !strings.Contains(err.Error(), "failed to create") {
		t.Errorf("err = %q, want 'failed to create' message", err)
	}
	if _, statErr := os.Stat(modulePath); !os.IsNotExist(statErr) {
		t.Errorf("expected module directory to be cleaned up, stat err = %v", statErr)
	}
}
