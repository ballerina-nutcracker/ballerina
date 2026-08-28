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

package splice

import (
	"path/filepath"
	"testing"
)

// TestEmbed_MissingStub covers embed()'s own stub-read guard. In
// production ResolveStub already checks the stub exists before calling
// Embed, so this only guards a TOCTOU race — not reachable via the CLI,
// hence a unit test rather than a corpus one. EmbedELF is just the
// vehicle; embed() itself is format-agnostic.
func TestEmbed_MissingStub(t *testing.T) {
	missingStub := filepath.Join(t.TempDir(), "does-not-exist")
	outPath := filepath.Join(t.TempDir(), "program")

	if err := EmbedELF(missingStub, []byte("payload"), outPath); err == nil {
		t.Fatal("expected an error for a missing stub")
	}
}
