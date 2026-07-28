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

//go:build !unix

// Compile-only fallback (e.g. Windows) so `go test ./...` builds everywhere.
// httpbench is only ever run on Linux CI / unix dev hosts, so process-group
// teardown is not needed here — a plain Kill suffices.

package main

import "os/exec"

func setProcAttr(_ *exec.Cmd) {}

// cmd.Process is always set here: the only caller registers this via defer
// after a successful cmd.Start(), which Go guarantees populates it.
func killGroup(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}
