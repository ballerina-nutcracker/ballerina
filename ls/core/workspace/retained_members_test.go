// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package workspace

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"
)

func TestRetainedMemberDirectories(t *testing.T) {
	t.Parallel()
	name := "modules/tools/helper.bal"
	overlay := overlayFS{
		base:     fstest.MapFS{},
		overlays: map[string][]byte{name: []byte("public function helper() {}"), "new.bal": []byte("function newMember() {}")},
		members:  map[string]bool{name: true},
		now:      time.Now,
	}
	for dir, child := range map[string]string{".": "modules", "modules": "tools", "modules/tools": "helper.bal"} {
		entries, err := fs.ReadDir(overlay, dir)
		if err != nil || len(entries) != 1 || entries[0].Name() != child {
			t.Fatalf("retained directory %s: %v, %v", dir, entries, err)
		}
		info, err := fs.Stat(overlay, dir)
		if err != nil || !info.IsDir() {
			t.Fatalf("directory stat %s: %v, %v", dir, info, err)
		}
	}
	if data, err := fs.ReadFile(overlay, name); err != nil || string(data) != string(overlay.overlays[name]) {
		t.Fatalf("member text: %q, %v", data, err)
	}
	delete(overlay.members, name)
	if entries, err := fs.ReadDir(overlay, "."); err != nil || len(entries) != 0 {
		t.Fatalf("membership survived close: %v, %v", entries, err)
	}
	if _, err := fs.Stat(overlay, "modules/tools"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("synthetic parents survived close: %v", err)
	}
	overlay.base = failingSourceFS{err: fs.ErrPermission}
	overlay.members[name] = true
	if _, err := fs.ReadDir(overlay, "."); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("overlay masked permission failure: %v", err)
	}
}
