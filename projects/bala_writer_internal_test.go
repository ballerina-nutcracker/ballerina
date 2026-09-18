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

package projects

import (
	"archive/zip"
	"testing"
	"testing/fstest"
)

// TestWriteZipEntry_RecordsWrittenPath covers populateBalaArchive's shared
// dedup mechanism at the unit level: writeZipEntry must record zipPath in
// written so a later stage (addIncludeFile) can detect the collision.
func TestWriteZipEntry_RecordsWrittenPath(t *testing.T) {
	t.Parallel()
	var buf zipBufferForTest
	zw := zip.NewWriter(&buf)
	written := make(map[string]bool)

	if err := writeZipEntry(zw, "main.bal", []byte("content"), written); err != nil {
		t.Fatalf("writeZipEntry: %v", err)
	}
	if !written["main.bal"] {
		t.Error("expected written[\"main.bal\"] to be true after writeZipEntry")
	}
}

// TestAddIncludeFile_SkipsPathWrittenByEarlierStage covers the fix for the
// cross-category duplicate-entry divergence from Java's BalaWriter: an
// include pattern matching a path some earlier archive stage (module
// sources, docs, toml files) already wrote must be silently skipped, not
// written a second time. The fsys deliberately doesn't contain relPath, so
// if addIncludeFile failed to skip, the fs.ReadFile call would error and
// fail this test.
func TestAddIncludeFile_SkipsPathWrittenByEarlierStage(t *testing.T) {
	t.Parallel()
	var buf zipBufferForTest
	zw := zip.NewWriter(&buf)
	fsys := fstest.MapFS{} // empty: relPath below does not exist here
	written := map[string]bool{"main.bal": true}

	if err := addIncludeFile(zw, fsys, "main.bal", "main.bal", written); err != nil {
		t.Fatalf("addIncludeFile: %v", err)
	}
}

// zipBufferForTest is a minimal io.Writer + io.Seeker/WriterAt-free sink;
// zip.Writer only needs io.Writer for sequential writes in these tests.
type zipBufferForTest struct {
	data []byte
}

func (b *zipBufferForTest) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}
