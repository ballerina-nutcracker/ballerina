// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 ( the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package workspace

import (
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
)

// makeEntry builds an indexEntry stand-in for index unit tests (the index only
// stores the entry; it never compiles, so a nil project is safe).
func makeEntry(root string, openDocs int, t time.Time) *indexEntry {
	return &indexEntry{project: nil, sourceRoot: root, lastTouched: t, openDocs: openDocs}
}

func TestProjectIndexEvictsBackgroundFirst(t *testing.T) {
	bus := event.New()
	defer bus.Close()
	var evicted []string
	bus.Subscribe([]event.Kind{event.ProjectEvicted}, func(e event.Event) {
		evicted = append(evicted, e.SourceRoot())
	})
	base := time.Unix(1000, 0)
	idx := newProjectIndex(2, func() time.Time { return base })

	// Two background entries, then one active entry.
	idx.entries["/bg1"] = makeEntry("/bg1", 0, base.Add(1*time.Second))
	idx.entries["/act1"] = makeEntry("/act1", 1, base.Add(2*time.Second))
	// Fill to capacity with a third (active) — triggers eviction on put.
	newEntry := makeEntry("/new1", 1, base.Add(3*time.Second))
	idx.putExisting(newEntry, bus)

	// The least-recently-touched background entry (/bg1) should be evicted.
	if _, ok := idx.entries["/bg1"]; ok {
		t.Fatal("expected background entry /bg1 to be evicted")
	}
	if _, ok := idx.entries["/act1"]; !ok {
		t.Fatal("expected active entry /act1 to survive")
	}
	if _, ok := idx.entries["/new1"]; !ok {
		t.Fatal("expected new entry /new1 to be stored")
	}
	if len(evicted) != 1 || evicted[0] != "/bg1" {
		t.Fatalf("evicted = %v, want [/bg1]", evicted)
	}
}

func TestProjectIndexEvictsLRUActiveWhenAllActive(t *testing.T) {
	bus := event.New()
	defer bus.Close()
	base := time.Unix(2000, 0)
	idx := newProjectIndex(2, func() time.Time { return base })

	idx.entries["/a1"] = makeEntry("/a1", 1, base.Add(1*time.Second))
	idx.entries["/a2"] = makeEntry("/a2", 1, base.Add(2*time.Second))
	// All entries active; adding a third evicts the LRU active (/a1).
	newEntry := makeEntry("/a3", 1, base.Add(3*time.Second))
	idx.putExisting(newEntry, bus)

	if _, ok := idx.entries["/a1"]; ok {
		t.Fatal("expected LRU active entry /a1 to be evicted")
	}
	if _, ok := idx.entries["/a2"]; !ok {
		t.Fatal("expected /a2 to survive")
	}
}

func TestProjectIndexMemoLookup(t *testing.T) {
	idx := newProjectIndex(8, time.Now)
	idx.memoSourceRoot("/f/a.bal", "/rootA")
	if root, ok := idx.lookupSourceRoot("/f/a.bal"); !ok || root != "/rootA" {
		t.Fatalf("lookup = %q ok=%v, want /rootA", root, ok)
	}
	idx.remove("/rootA")
	if _, ok := idx.lookupSourceRoot("/f/a.bal"); ok {
		t.Fatal("memo should be invalidated after remove")
	}
}

func TestProjectIndexInvalidateUnder(t *testing.T) {
	idx := newProjectIndex(8, time.Now)
	idx.memoSourceRoot("/d/x.bal", "/d")
	idx.memoSourceRoot("/d/sub/y.bal", "/d")
	idx.memoSourceRoot("/other/z.bal", "/other")
	idx.invalidateUnder("/d")
	if _, ok := idx.lookupSourceRoot("/d/x.bal"); ok {
		t.Fatal("/d/x.bal memo should be invalidated")
	}
	if _, ok := idx.lookupSourceRoot("/d/sub/y.bal"); ok {
		t.Fatal("/d/sub/y.bal memo should be invalidated")
	}
	if _, ok := idx.lookupSourceRoot("/other/z.bal"); !ok {
		t.Fatal("/other/z.bal memo should survive")
	}
}

func TestIsAncestor(t *testing.T) {
	cases := []struct {
		dir, p string
		want   bool
	}{
		{"/d", "/d", true},
		{"/d", "/d/x", true},
		{"/d", "/dx", false},
		{"/d/", "/d/x", true},
		{"/d", "/e/x", false},
		{"", "/x", false},
	}
	for _, c := range cases {
		if got := isAncestor(c.dir, c.p); got != c.want {
			t.Errorf("isAncestor(%q,%q) = %v, want %v", c.dir, c.p, got, c.want)
		}
	}
}
