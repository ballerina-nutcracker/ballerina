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

// Package goroutinedump prints live goroutines when a test finishes.
package goroutinedump

import (
	"runtime"
	"testing"
)

// AtCleanup prints every live goroutine when t finishes. It diagnoses wasm
// runs that print PASS but never exit: the built-in println writes
// synchronously on js/wasm, so the dump is emitted even if a goroutine left
// running afterwards starves the main goroutine.
func AtCleanup(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		buf := make([]byte, 1<<20)
		for {
			n := runtime.Stack(buf, true)
			if n < len(buf) {
				buf = buf[:n]
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		println("=== WASM-HANG-DIAG goroutines at cleanup of " + t.Name() + " ===")
		println(string(buf))
		println("=== WASM-HANG-DIAG end ===")
	})
}
