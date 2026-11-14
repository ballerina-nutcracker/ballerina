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

package native

import (
	"strconv"
	"testing"
)

// TestRowBuilderWideRowIsLinear guards against a key lookup that scans the earlier entries. A row expected to be
// wider than indexThreshold must be indexed from its first key, and a row of unknown width once it outgrows the
// threshold. Putting every key again must overwrite in place, which also fails if building the index drops declared
// or rest entries, as values.NewMap would otherwise hide duplicate entries from the corpus tests.
func TestRowBuilderWideRowIsLinear(t *testing.T) {
	t.Parallel()
	const n = 10_000
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "k" + strconv.Itoa(i)
	}
	shape := &rowShape{typeInfo: typeInfo{kind: kindRecord}, fields: map[string]*recordField{"k0": {name: "k0"}}}
	for _, tc := range []struct {
		width     int
		indexedAt int
	}{{0, indexThreshold + 1}, {n, 1}} {
		b := newRowBuilder(shape, tc.width)
		for i, k := range keys {
			b.put(k, nil)
			if indexed := b.positions != nil; indexed != (i+1 >= tc.indexedAt) {
				t.Fatalf("width %d: indexed=%t after %d keys, want the index from key %d", tc.width, indexed, i+1,
					tc.indexedAt)
			}
		}
		if len(b.positions) != n {
			t.Fatalf("width %d: the index holds %d of %d keys", tc.width, len(b.positions), n)
		}
		for i, k := range keys {
			b.put(k, int64(i))
		}
		if len(b.declared) != 1 || len(b.rest) != n-1 || !b.has("k0") {
			t.Fatalf("width %d: got %d declared and %d rest entries", tc.width, len(b.declared), len(b.rest))
		}
		for i, e := range b.rest {
			if e.Key != keys[i+1] || e.Value != int64(i+1) {
				t.Fatalf("width %d: rest entry %d is %s=%v", tc.width, i, e.Key, e.Value)
			}
		}
	}
}
