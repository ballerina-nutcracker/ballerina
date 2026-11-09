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
	"time"
)

// TestRowBuilderWideRowIsLinear guards against a key lookup that scans the earlier entries: 100k keys
// take milliseconds when indexed and several seconds when scanned. Putting every key again must overwrite
// in place, as values.NewMap would otherwise hide duplicate entries from the corpus tests.
func TestRowBuilderWideRowIsLinear(t *testing.T) {
	const n = 100_000
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "k" + strconv.Itoa(i)
	}
	b := newRowBuilder(&rowShape{fields: map[string]*recordField{"k0": {name: "k0"}}})
	start := time.Now()
	for _, k := range keys {
		b.put(k, nil)
	}
	for i, k := range keys {
		b.put(k, int64(i))
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("putting %d keys took %v", n, elapsed)
	}
	if len(b.declared) != 1 || len(b.rest) != n-1 || !b.has("k0") {
		t.Fatalf("got %d declared and %d rest entries", len(b.declared), len(b.rest))
	}
	for i, e := range b.rest {
		if e.Key != keys[i+1] || e.Value != int64(i+1) {
			t.Fatalf("rest entry %d is %s=%v", i, e.Key, e.Value)
		}
	}
}
